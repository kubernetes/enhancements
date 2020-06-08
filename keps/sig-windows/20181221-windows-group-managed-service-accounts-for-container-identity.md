---
title: Windows Group Managed Service Accounts for Container Identity
authors:
  - "@ddebroy"
  - "@jeremywx"
  - "@patricklang"
owning-sig: sig-windows
participating-sigs:
  - sig-auth
  - sig-node
  - sig-architecture
  - sig-docs
reviewers:
  - "@liggitt"
  - "@mikedanese"
  - "@yujuhong"
  - "@patricklang"
approvers:
  - "@liggitt"
  - "@yujuhong"
  - "@patricklang"
editor: TBD
creation-date: 2018-11-29
last-updated: 2020-03-20
status: implemented
---

# Windows Group Managed Service Accounts for Container Identity


## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Background](#background)
    - [What is Active Directory?](#what-is-active-directory)
    - [What is a Windows service account?](#what-is-a-windows-service-account)
    - [How is it applied to containers?](#how-is-it-applied-to-containers)
  - [User Stories](#user-stories)
    - [Web Applications with MS SQL Server](#web-applications-with-ms-sql-server)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    - [GMSA specification for pods and containers](#gmsa-specification-for-pods-and-containers)
    - [GMSAExpander webhook](#gmsaexpander-webhook)
    - [GMSAExpander and GMSAAuthorizer Webhooks](#gmsaexpander-and-gmsaauthorizer-webhooks)
    - [Changes in Kubelet/kuberuntime for Windows:](#changes-in-kubeletkuberuntime-for-windows)
    - [Changes in CRI API:](#changes-in-cri-api)
    - [Changes in Dockershim](#changes-in-dockershim)
    - [Changes in CRIContainerD](#changes-in-cricontainerd)
    - [Changes in Windows OCI runtime](#changes-in-windows-oci-runtime)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Threat vectors and countermeasures](#threat-vectors-and-countermeasures)
    - [Transitioning from Alpha annotations to Beta/Stable fields](#transitioning-from-alpha-annotations-to-betastable-fields)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives](#alternatives)
  - [Other authentication methods](#other-authentication-methods)
  - [Injecting credentials from a volume](#injecting-credentials-from-a-volume)
  - [Specifying only the name of GMSACredentialSpec objects in pod spec fields/annotations](#specifying-only-the-name-of-gmsacredentialspec-objects-in-pod-spec-fieldsannotations)
  - [Enforce presence of GMSAAuthorizer and RBAC mode to enable GMSA functionality in Kubelet](#enforce-presence-of-gmsaauthorizer-and-rbac-mode-to-enable-gmsa-functionality-in-kubelet)
<!-- /toc -->

## Summary

Active Directory is a service that is built-in and commonly used on Windows Server deployments for user and computer identity. Apps are run using Active Directory identities to enable common user to service, and service to service authentication and authorization. This proposal aims to support a specific type of identity, Group Managed Service Accounts (GMSA), for use with Windows Server containers. This will allow an operator to choose a GMSA at deployment time, and run containers using it to connect to existing applications such as a database or API server without changing how the authentication and authorization are performed.

## Motivation

There has been a lot of interest in supporting GMSA for Windows containers since it's the only way for a Windows application to use an Active Directory identity. This is shown in asks and questions from both public & private conversations:

- https://github.com/kubernetes/kubernetes/issues/51691 "For windows based containers, there is a need to access shared objects using domain user contexts."
- Multiple large customers are asking the Microsoft Windows team to enable this feature through container orchestrators
- Multiple developers have blogged how to use this feature, but all are on standalone machines instead of orchestrated through Kubernetes
  - https://artisticcheese.wordpress.com/2017/09/09/enabling-integrated-windows-authentication-in-windows-docker-container/
  - https://community.cireson.com/discussion/3853/example-cireson-scsm-portal-on-docker-windows-containers
  - https://cloudiqtech.com/windows-2016-docker-containers-using-gmsa-connect-to-sql-server/
  - https://success.docker.com/api/asset/.%2Fmodernizing-traditional-dot-net-applications%2F%23integratedwindowsauthentication


### Goals

- Windows users can run containers with an existing GMSA identity on Kubernetes
- No extra files or Windows registry keys are needed on each Windows node. All needed data should flow through Kubernetes+Kubelet APIs
- Prevent pods from being inadvertently scheduled with service accounts that do not have access to a GMSA

### Non-Goals

- Running Linux applications using GMSA or a general Kerberos based authentication system
- Replacing any existing Kubernetes authorization or authentication controls. Specifically, a subset of users cannot be restricted from creating pods with Service Accounts authorized to use certain GMSAs within a namespace if the users are already authorized to create pods within that namespace. Namespaces serve as the ACL boundary today in Kubernetes and we do not try to modify or enhance this in the context of GMSAs to prevent escalation of privilege through a service account authorized to use certain GMSAs.
- Providing unique container identities. By design, Windows GMSA are used where multiple nodes are running apps as the same Active Directory principal.
- Isolation between container users and processes running as the GMSA. Windows already allows users and system services with sufficient privilege to create processes using a GMSA.
- Preventing the node from having access to the GMSA. Since the node already has authorization to access the GMSA, it can start processes or services using as the GMSA. Containers do not change this behavior.
- Restricting specification of GMSA credential specs in pods or containers if RBAC is not enabled or the admission webhook described below is not installed/enabled.


## Proposal

### Background

#### What is Active Directory?
Windows applications and services typically use Active Directory identities to facilitate authentication and authorization between resources. In a traditional virtual machine scenario, the computer is joined to an Active Directory domain which enables it to use Kerberos, NTLM, and LDAP to identify and query for information about other resources in the domain. When a computer is joined to Active Directory, it is given an unique identity represented as a computer object in LDAP.

#### What is a Windows service account?
A Group Managed Service Account (GMSA) is a shared Active Directory identity that enables common scenarios such as authenticating and authorizing incoming requests and accessing downstream resources such as a database server, file share, or other workload. It can be used by multiple authorized users or computers at the same time.

#### How is it applied to containers?
To achieve the scale and speed required for containers, Windows uses a group managed service account in lieu of individual computer accounts to enable Windows containers to communicate with Active Directory. As of right now, the Host Computer Service (which exposes the interface to manage containers) in Windows cannot use individual Active Directory computer & user accounts - it only supports GMSA.

Different containers on the same machine can use different GMSAs to represent the specific workload they are hosting, allowing operators to granularly control which resources a container has access to. However, to run a container with a GMSA identity, an additional parameter must be supplied to the Windows Host Compute Service to indicate the intended identity. This proposal seeks to add support in Kubernetes for this parameter to enable Windows containers to communicate with other enterprise resources.

It's also worth noting that Docker implements this in a different way that's not managed centrally. It requires dropping a file on the node and referencing it by name, eg: docker run --credential-spec='file://gmsa-cred-spec1.json' . For more details see the Microsoft doc.


### User Stories


#### Web Applications with MS SQL Server
A developer has a Windows web app that they would like to deploy in a container with Kubernetes. For example, it may have a web tier that they want to scale out running ASP.Net hosted in the IIS web server. Backend data is stored in a Microsoft SQL database, which all of the web servers need to access behind a load balancer. An Active Directory Group Managed Service Account is used to avoid hardcoded passwords, and the web servers run with that credential today. The SQL Database has a user with read/write access to that GMSA so the web servers can access it. When they move the web tier into a container, they want to preserve the existing authentication and authorization models.

When this app moves to production on containers, the team will need to coordinate to make sure the right GMSA is carried over to the container deployment.

1. An Active Directory domain administrator will:

  - Join all Windows Kubernetes nodes to the Active Directory domain.
  - Provision the GMSA and gives details to Kubernetes cluster admin.
  - Assign access to a AD security group to control what machines can use this GMSA. The AD security group should include all authorized Kubernetes nodes.

2. A Kubernetes cluster admin (or cluster setup tools or an operator) will install a cluster scoped CRD (of kind GMSACredentialSpec) whose instances will be used to store GMSA credential spec configuration:

```
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: gmsacredentialspecs.windows.k8s.io
spec:
  group: windows.k8s.io
  version: v1alpha1
  names:
    kind: GMSACredentialSpec
    plural: gmsacredentialspecs
  scope: Cluster
  validation:
    openAPIV3Schema:
      properties:
        credspec:
          description: GMSA Credential Spec
          type: object
```

A GMSACredentialSpec may be used by sidecar containers across different namespaces. Therefore the CRD needs to be cluster scoped.

3. A Kubernetes cluster admin will create a GMSACredentialSpec object populated with the credential spec associated with a desired GMSA:

  - The cluster admin may run a utility Windows PowerShell script to generate the YAML definition of a GMSACredentialSpec object populated with the GMSA credential spec details. Note that the credential spec YAML follows the structure of the credential spec (in JSON) as referred to in the [OCI spec](https://github.com/opencontainers/runtime-spec/blob/master/config-windows.md#credential-spec). The utility Powershell script for generating the YAML will be largely identical to the already published [Powershell script](https://github.com/MicrosoftDocs/Virtualization-Documentation/blob/live/windows-server-container-tools/ServiceAccounts/CredentialSpec.psm1) with the following differences: [a] it will output the credential spec in YAML format and [b] it will encapsulate the credential spec data within a Kubernetes object YAML (of kind GMSACredentialSpec). The GMSACredentialSpec YAML will not contain any passwords or crypto secrets. Example credential spec YAML for a GMSA webapplication1:

```
apiVersion: windows.k8s.io/v1alpha1
kind: GMSACredentialSpec
metadata:
  name: "webapp1-credspec"
credspec:
  ActiveDirectoryConfig:
    GroupManagedServiceAccounts:
    - Name: WebApplication1
      Scope: CONTOSO
    - Name: WebApplication1
      Scope: contoso.com
  CmsPlugins:
  - ActiveDirectory
  DomainJoinConfig:
    DnsName: contoso.com
    DnsTreeName: contoso.com
    Guid: 244818ae-87ca-4fcd-92ec-e79e5252348a
    MachineAccountName: WebApplication1
    NetBiosName: CONTOSO
    Sid: S-1-5-21-2126729477-2524075714-3094792973
```

  - With the YAML from above (or generated manually), the cluster admin will create a GMSACredentialSpec object in the cluster.

4. A Kubernetes cluster admin will configure RBAC for the GMSACredentialSpec so that only desired service accounts can use the GMSACredentialSpec:

  - Create a cluster wide role to get and "use" the GMSACredentialSpec:

```
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: webapp1-gmsa-user
rules:
- apiGroups: ["windows.k8s.io"]
  resources: ["gmsacredentialspecs"]
  resourceNames: ["webapp1-credspec"]
  verbs: ["get", "use"]
```

  - Create a rolebinding and assign desired service accounts to the above role:

```
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: use-webapp1-gmsa
  namespace: default
subjects:
- kind: ServiceAccount
  name: webapp-sa
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: webapp1-gmsa-user
  apiGroup: rbac.authorization.k8s.io
```

5. Application admins will deploy app pods that require a GMSA identity along with a Service Account authorized to use the GMSAs. There will be two ways to specify the GMSA credential spec details for pods and containers. It is expected that users will typically use the first option as it is more user friendly. The second option is available mainly due to an artifact of the design choices made and described here for completeness.

  - Specify the name of the desired GMSACredentialSpec object (e.g. `webapp1-credspec`): If an application administrator wants containers to be initialized with a GMSA identity, specifying the names of the desired GMSACredentialSpec objects is mandatory. In the alpha stage of this feature, the name of the desired GMSACredentialSpec can be set through an annotation on the pod (applicable to all containers): `pod.alpha.windows.kubernetes.io/gmsa-credential-spec-name` as well as for each container through annotations of the form: `<containerName>.container.alpha.windows.kubernetes.io/gmsa-credential-spec-name`. In the beta stage, the annotations will be superseded by fields in the securityContext of the pod: `podspec.securityContext.windows.gmsaCredentialSpecName` and in the securityContext of each container:  `podspec.container[i].securityContext.windows.gmsaCredentialSpecName`. The GMSACredentialSpec name for a container will override the GMSACredentialSpec name specified for the whole pod. Sample pod spec showing specification of GMSACredentialSpec name at the pod level and overriding it for one of the containers:

```
apiVersion: v1
kind: Pod
metadata:
  name: iis
  labels:
    name: iis
  annotations: {
    pod.alpha.windows.kubernetes.io/gmsa-credential-spec-name : webapp1-credspec
    iis.container.alpha.windows.kubernetes.io/gmsa-credential-spec-name : webapp2-credspec
  }
spec:
  containers:
    - name: iis
      image: mcr.microsoft.com/windows/servercore/iis:windowsservercore-ltsc2019
      ports:
        - containerPort: 80
    - name: logger
      image: eventlogger:2019
      ports:
        - containerPort: 80
  nodeSelector:
    beta.kubernetes.io/os : windows
```

  - Specify the contents of the `credspec` field of GMSACredentialSpec that gets passed down to the container runtime: Specifying the credential spec contents in JSON form is optional and unnecessary. GMSAExpander will automatically populate this field (as described in the next section) based on the name of the GMSACredentialSpec object. In the alpha stage of this feature, a JSON representation of the contents of the desired GMSACredentialSpec may be set through an annotation on the pod (applicable to all containers): `pod.alpha.windows.kubernetes.io/gmsa-credential-spec` as well as for each container through annotations of the form: `<containerName>.container.alpha.windows.kubernetes.io/gmsa-credential-spec`. In the beta stage and beyond, the annotations will be superseded by a field the securityContext of the pod `podspec.securityContext.windows.gmsaCredentialSpec` and in the securityContext of each  container: `podspec.container[i].securityContext.windows.gmsaCredentialSpec`. The credential spec JSON for a container will override the credential spec JSON specified for the whole pod.

  The ability to specify credential specs for each container within a pod aligns with how security attributes like `runAsGroup`, `runAsUser`, etc. can be specified at the pod level and overridden at the container level if desired.

  Note that as this feature graduates to Beta, support for the annotations will be removed in favor of securityContext fields in podspec. The implication of the removal of support for the Alpha annotations is covered in the Risks and Mitigations section later.

6. A mutating webhook admission controller, GMSAExpander, will act on pod creations. GMSAExpander will look up the GMSACredentialSpec object referred to by name and use the contents in the `credspec` field to populate the GMSA credential spec JSON if absent or empty in the necessary annotations [in Alpha] or securityContext fields [Beta onwards]. Specifics of the checks performed, fields affected and error scenarios for GMSAExpander is covered in details in the Implementation section below.

7. A validating webhook admission controller, GMSAAuthorizer, will act on pod creations (as well as updates as discussed later in Step 11) and execute a series of checks and authorization around the GMSA annotations [in Alpha] or securityContext fields [in Beta]. Specifics of the annotations or securityContext fields examined and authorizations checks performed along with error scenarios for GMSAAuthorizer is covered in details in the Implementation section below.

8. Kubelet.exe in Windows nodes will examine the credential spec related annotations [in Alpha] or securityContext fields [Beta onwards] for a given pod as well as for each container in the pod. For each container, Kubelet will compute an effective credential spec - either the credential spec specified for the whole pod or a credential spec specified specifically for the container. During Alpha, Kubelet will set the effective credential spec for each container as annotation: `<containerName>.container.alpha.windows.kubernetes.io/gmsa-credential-spec`. Beta onwards, Kubelet will set the effective credential spec for each container in a new security context field `WindowsContainerSecurityContext.CredentialSpec` which will require an update to the CRI API. Please see the Implementation section below for details on the enhancements necessary in Kubelet and CRI API.

9. The Windows CRI implementation will access the credential spec JSON through annotations [`<containerName>.container.alpha.windows.kubernetes.io/gmsa-credential-spec` for Alpha] or securityContext field [`WindowsContainerSecurityContext.CredentialSpec` Beta onwards] in `CreateContainerRequest` for each container in a pod. The CRI implementation will transmit the credential spec JSON through a runtime implementation dependent mechanism to a specific container runtime. For example:
 - Docker (to be supported in Alpha): will receive the path to a file created on the node's file system under `C:\ProgramData\docker\CredentialSpecs\` and populated by dockershim with the credential spec JSON. Docker will read the contents of the credential spec file and pass it to Windows Host Compute Service (HCS) when creating and starting a container in Windows.
 - ContainerD (to be supported in Beta): will receive a OCI Spec with [windows.CredentialSpec]( https://github.com/opencontainers/runtime-spec/blob/master/config-windows.md#credential-spec) populated by CRIContainerD with the credential spec JSON. The OCI spec will be passed to a OCI runtime like RunHCS.exe when starting a container.
Please see the Implementation section below for details on the enhancements necessary for select CRI implementations and their corresponding runtimes.

10. The Windows container runtime (Docker or ContainerD + RunHCS) will depend on Windows Host Compute Service (HCS) APIs/vmcompute.exe to start the containers and assign them user identities  corresponding to the GMSA configured for each container. Using the GMSA identity, the processes within the container can authenticate to a service protected by GMSA like database. The containers that fail to start due to invalid GMSA configuration (as determined by HCS or container runtime like Docker) will end up in a failed state with the following when described:
```
State:              Waiting
      Reason:           CrashLoopBackOff
Last State:         Terminated
      Reason:           ContainerCannotRun
      Message:          [runtime specific error e.g. "encountered an error during CreateContainer:..."]
```

There are a couple of failure scenarios to consider in the context of incorrect GMSA configurations for containers and hosts where containers do get started in spite of incorrect GMSA configuration:

 - On a Windows node not connected to AD, a GMSA credential spec JSON is passed that is structurally valid JSON: Windows HCS APIs are not able to detect the fact that the Windows node is not connected to a domain and starts the container normally.
 - On a Windows node connected to AD, a GMSA credential spec JSON is passed that the node is not configured to use: Windows HCS APIs are not able to detect the fact that the Windows node is not authorized to use the GMSA and starts the container.

In the above scenarios, after a container has started with the GMSA credential spec JSON configured, within the container, validation commands like `whoami /upn` fails with `Unable to get User Principal Name (UPN) as the current logged-on user is not a domain user` and `nltest /query` fails with `ERROR_NO_LOGON_SERVERS`/`ERROR_NO_TRUST_LSA_SECRET`. An init container is recommended to be added to pod specs (where GMSA configurations are desired) to validate that the desired domain can be reached (through `nltest /query`) as well as the desired identity can be obtained (through `whoami /upn`) with the configured GMSA credential specs. If the validation commands fail, the init container will exit with failures and thus prevent pod bringup. The failure can be discovered by describing the pod and looking for errors along the lines of the following for the GMSA validating init container:
```
State:           Waiting
      Reason:        CrashLoopBackOff
Last State:      Terminated
      Reason:        Error
```

11. During any pod update, any changes to a pod's `securityContext` will be blocked (as is the case today) by `ValidatePodUpdate` Beta onwards. Updates of the annotations associated with GMSA will be rejected by GMSAAuthorizer in Alpha stage. Note that modifications or removal of the named GMSACredentialSpec, or removal of authorization does not disrupt update/deletion of previously created pods.


### Implementation Details/Notes/Constraints [optional]

#### GMSA specification for pods and containers
In the Alpha phase of this feature we will use the following annotations on a pod for GMSA credential spec:
  - References to names of GMSACredentialSpec objects will be specified through the following annotations:
    - At the pod level: `pod.alpha.windows.kubernetes.io/gmsa-credential-spec-name`
    - At the container level: `<containerName>.container.alpha.windows.kubernetes.io/gmsa-credential-spec-name`
  - The contents of the credential spec will be specified through or populated in the following annotations:
    - At the pod level: `pod.alpha.windows.kubernetes.io/gmsa-credential-spec`
    - At the container level: `<containerName>.container.alpha.windows.kubernetes.io/gmsa-credential-spec`

In the Beta phase of this feature, support for above the annotations above will be dropped. The annotations will be superseded by fields in the pod spec:
  - References to names of GMSACredentialSpec objects will be specified through the following fields:
    - At the pod level `podspec.securityContext.windows.gmsaCredentialSpecName`
    - At the container level: `podspec.container[i].securityContext.windows.gmsaCredentialSpecName`
  - The contents of the credential spec will be specified through or populated in the following fields:
    - At the pod level: `podspec.securityContext.windows.gmsaCredentialSpec`
    - At the container level: `podspec.container[i].securityContext.windows.gmsaCredentialSpec`

Note that the `windows.gmsaCredentialSpecName` and `windows.gmsaCredentialSpec` fields of the `securityContext` struct is speculative at the moment and may change in the future. The names/parents of the GMSA fields will depend on how the exact structure and representation of OS specific `securityContext` fields evolve.

#### GMSAExpander webhook
A new webhook, GMSAExpander, will be implemented and configured to act on pod creation. It will perform the following steps:

  - In Alpha, check if GMSA credential spec JSON annotations [`pod.alpha.windows.kubernetes.io/gmsa-credential-spec` or `<containerName>.container.alpha.windows.kubernetes.io/gmsa-credential-spec`] corresponding to each reference to names of GMSACredentialSpec objects in annotations [`pod.alpha.windows.kubernetes.io/gmsa-credential-spec-name` or `<containerName>.container.alpha.windows.kubernetes.io/gmsa-credential-spec-name/container-name`] is present and populated.

  - In Beta, check if securityContext field [`podspec.securityContext.windows.gmsaCredentialSpec` or `podspec.container[i].securityContext.windows.gmsaCredentialSpec`] corresponding to each reference to names of GMSACredentialSpec objects in securityContext fields [`podspec.securityContext.windows.gmsaCredentialSpecName` or `podspec.container[i].securityContext.windows.gmsaCredentialSpecName` is populated.

  - If the GMSA credential spec JSON annotation is absent or empty or the securityContext field is empty, look up the GMSACredentialSpec object by name. If GMSACredentialSpec object does not exist, return error 422 Unprocessable Entity with message indicating GMSACredentialSpec object with specified name could not be found. If the GMSACredentialSpec object exists, obtain the data in the `credspec` member and convert it to JSON. Next, create a new annotation [`pod.alpha.windows.kubernetes.io/gmsa-credential-spec` or `<containerName>.container.alpha.windows.kubernetes.io/gmsa-credential-spec`] if absent and populate it with the JSON (for Alpha) or populate the securityContext field [`podspec.securityContext.windows.gmsaCredentialSpec` or `podspec.container[i].securityContext.windows.gmsaCredentialSpec`] with the JSON (Beta onwards).

Note that the annotations will not be processed/populated once the feature graduates to Beta and the securityContext fields will be used instead.

#### GMSAExpander and GMSAAuthorizer Webhooks
A new webhook, GMSAAuthorizer will be implemented to act on pod creation and updates and perform several checks and validations.

During pod creation, the following checks and validations will be executed:

  - Authorize the service account specified for the pod to use specified GMSACredentialSpec objects: First look up all references to GMSACredentialSpec objects by name specified through annotations [`pod.alpha.windows.kubernetes.io/gmsa-credential-spec-name` and `<containerName>.container.alpha.windows.kubernetes.io/gmsa-credential-spec-name` in Alpha] or securityContext fields [`podspec.securityContext.windows.gmsaCredentialSpecName` and `podspec.container[i].securityContext.windows.gmsaCredentialSpecName` Beta onwards]. Next, generate custom `AttributesRecord`s with `verb` set to `use`, `name` set to the specified name of a GMSACredentialSpec object and `user` set to the service account of the pod. Finally, the `AttributesRecord`s will be passed to authorizers to check against RBAC configurations. A failure from the authorizer results in a response 403: Forbidden with message indicating the GMSACredentialSpec object to which the access was denied.

  - Check each GMSA credential spec JSON has an associated reference to a GMSACredentialSpec object by name: The GMSA credential spec JSON may be populated in annotations or securityContext fields directly by app admins or through GMSAExpander. For each GMSA credential spec JSON specification in annotations [`pod.alpha.windows.kubernetes.io/gmsa-credential-spec` and `<containerName>.container.alpha.windows.kubernetes.io/gmsa-credential-spec` in Alpha] or securityContext fields [`podspec.securityContext.windows.gmsaCredentialSpec` or `podspec.container[i].securityContext.windows.gmsaCredentialSpec` Beta onwards], locate a corresponding reference to a GMSACredentialSpec object by name in annotations [`pod.alpha.windows.kubernetes.io/gmsa-credential-spec-name` or `<containerName>.container.alpha.windows.kubernetes.io/gmsa-credential-spec-name` in Alpha] or securityContext fields [`podspec.securityContext.windows.gmsaCredentialSpecName` or `podspec.container[i].securityContext.windows.gmsaCredentialSpecName` Beta onwards]. If the reference to a GMSACredentialSpec object by name is not found, pod creation will be failed with 422: Unprocessable Entity along with a message indicating the GMSA credential spec JSON whose name was absent. If the reference is found, establish pair <GMSA CredentialSpec object name, GMSA credential spec JSON>.

  - Validate contents of GMSA credential spec JSON: For each <GMSA CredentialSpec object name, GMSA credential spec JSON> pair established above, compare, in a deep equal fashion, the contents of the `credspec` member of the GMSACredentialSpec object (referred to by name) with the GMSA credential spec JSON obtained from annotations/securityContext fields. If the deep equal comparison fails, pod creation will be failed with 422: Unprocessable Entity along with a message indicating the mismatch and the contents of the credential spec JSONs that did not match.

During pod updates, changes to the credential spec annotations will be blocked by GMSAAuthorizer and failed with 400: BadRequest. Note that modifications or removal of the named GMSACredentialSpec, or removal of authorization does not disrupt update/deletion of previously created pods.

Note that the annotations will not be processed/populated once the feature graduates to Beta and the securityContext fields will be used instead.

If the GMSAAuthorizer webhook is not installed and configured, no authorization checks will be performed on the contents of the credential spec JSON. This will allow arbitrary credential spec JSON to be specified for pods/containers and sent down to the container runtime. Therefore when configuring Windows worker nodes for GMSA support, in the Alpha stage, Kubernetes cluster administrators need to ensure that the GMSAAuthorizer webhook is installed and configured.

#### Changes in Kubelet/kuberuntime for Windows:

In the Alpha phase, `applyPlatformSpecificContainerConfig` will be enhanced (under a feature flag: `WindowsGMSA`) to analyze the credential spec related annotations on the pod [`pod.alpha.windows.kubernetes.io/gmsa-credential-spec` and `<containerName>.container.alpha.windows.kubernetes.io/gmsa-credential-spec`] and determine an effective credential spec for each container:
 - If `<containerName>.container.alpha.windows.kubernetes.io/gmsa-credential-spec` is populated, effective credential spec of the container is set to that value.
 - If `<containerName>.container.alpha.windows.kubernetes.io/gmsa-credential-spec` is absent but `pod.alpha.windows.kubernetes.io/gmsa-credential-spec` is populated, effective credential spec of the container is set to the contents of `pod.alpha.windows.kubernetes.io/gmsa-credential-spec`.
 - If `<containerName>.container.alpha.windows.kubernetes.io/gmsa-credential-spec` is absent and `pod.alpha.windows.kubernetes.io/gmsa-credential-spec` is absent effective credential spec is nil.
Next, annotation: `container.alpha.windows.kubernetes.io/gmsa-credential-spec` in `ContainerConfig` will be populated with the effective credential spec of the container.

In the Beta phase, the logic in `applyPlatformSpecificContainerConfig` to populate annotation `container.alpha.windows.kubernetes.io/gmsa-credential-spec` in `ContainerConfig` will be removed. Instead, `DetermineEffectiveSecurityContext` will be enhanced (also under a feature flag: `WindowsGMSA`) to analyze the `securityContext.windows.gmsaCredentialSpec` fields for the pod overall and each container in the podspec and determine an effective credential spec for each container in the same fashion described above (and as it does today for several fields like RunAsUser, etc). Next, `ContainerConfig.WindowsContainerSecurityContext.CredentialSpec` will be populated with the effective credential spec for the container.

#### Changes in CRI API:

In the Alpha phase, no changes will be required in the CRI API. Annotation `container.alpha.windows.kubernetes.io/gmsa-credential-spec` in `ContainerConfig` will contain the credential spec JSON associated with each container.

In the Beta phase, a new field `CredentialSpec String` will be added to `WindowsContainerSecurityContext` in `ContainerConfig`. This field will be populated with the credential spec JSON of a Windows container by Kubelet.

#### Changes in Dockershim

The GMSA credential spec will be passed to Docker through temporary entries in the Windows registry under SOFTWARE\Microsoft\Windows NT\CurrentVersion\Virtualization\Containers\CredentialSpecs. The registry entries will be created with unique key names that have a common prefix. The contents of the registry entries will be used to populate `HostConfig.SecurityOpt` with a credential spec file specification. The registry entries will be deleted as soon as `CreateContainer` has been invoked on the Docker client. An alternative implementation considered was to utilize files instead of registry entries but the path and drive where the files can be stored is hard to determine as there could be multiple installations of different versions of Docker engine under different directory paths.

During Alpha, `dockerService.CreateContainer` function will be enhanced (under a feature flag: `WindowsGMSA`) to create the temporary registry entries and populate them with the contents of `container.alpha.windows.kubernetes.io/gmsa-credential-spec` annotation in `CreateContainerRequest.ContainerConfig`. Beta onwards, `dockerService.CreateContainer` (under a feature flag: `WindowsGMSA`) will use the contents of `WindowsContainerSecurityContext.CredentialSpec` to populate the registry values.

#### Changes in CRIContainerD

During Alpha, updating the CRI API and thus enabling interactions with ContainerD as a runtime is not planned. Once the CRI API has been updated to pass the `WindowsContainerSecurityContext.CredentialSpec` during Beta, CRIContainerD should be able to access the credential spec JSON. At that point, CRIContainerD will need to be enhanced to populate the [windows.CredentialSpec]( https://github.com/opencontainers/runtime-spec/blob/master/config-windows.md#credential-spec) field of the OCI runtime spec for Windows containers with the credential spec JSON passed through CRI.

#### Changes in Windows OCI runtime

The Windows OCI runtime already has support for `windows.CredentialSpec` and is implemented in Moby/Docker as well hcsshim/runhcs.

### Risks and Mitigations

#### Threat vectors and countermeasures

1. Prevent an unauthorized user from referring to an existing GMSA configmap in the pod spec: The GMSAAuthorizer Admission Controller along with RBAC policies with the `use` verb on a GMSA configmap ensures only users allowed by the kubernetes admin can refer to the GMSA configmap in the pod spec.
2. Prevent an unauthorized user from using an existing Service Account that is authorized to use an existing GMSA configmap: The GMSAAuthorizer Admission Controller checks the `user` as well as the service account associated with the pod have `use` rights on the GMSA configmap.
3. Prevent an unauthorized user from reading the GMSA credential spec and using it directly through docker on Windows hosts connected to AD that user has access to: RBAC policy on the GMSA configmaps should only allow `get` verb for authorized users.

#### Transitioning from Alpha annotations to Beta/Stable fields

Logic to process annotations used to specify GMSA details in Alpha phase will be removed once the feature graduates to beta. Since the annotations are only used during the Alpha phase, this deprecation and removal is compliant with [Kubernetes guidelines] (https://kubernetes.io/docs/reference/using-api/deprecation-policy/#deprecating-a-flag-or-cli). When upgrading from a version of Kubernetes with GMSA support in Alpha to a version where GMSA support has graduated to Beta, pod yamls with annotations for GMSA will need to be rewritten to specify the GMSA details in securityContext fields. The opposite conversion will need to be authored in pod YAMLs when downgrading from a version with Beta support of GMSA to Alpha support.


## Graduation Criteria

- alpha - Initial implementation with webhook and annotations on pods with no API changes in PodSpec or CRI. Kubelet and Dockershim enhancements will be guarded by a feature flag `WindowsGMSA` and disabled by default. Manual e2e tests with domain joined Window nodes with Docker as the container runtime in a cluster needs to pass.
- beta - Support for the Alpha annotations will be removed and replaced with new fields in PodSpec and CRI API. GMSA Annotations on a set of pods from a cluster upgraded from a Kubernetes version supporting GMSA configuration in alpha will not be supported and cluster operator will need to rewrite the YAMLs to specify the fields in podspec that supersede the annotations. Removal of support for Alpha annotations is allowed by [Kubernetes guidelines] (https://kubernetes.io/docs/reference/using-api/deprecation-policy/#deprecating-a-flag-or-cli). Feature flag `WindowsGMSA` will be enabled by default for Kubelet and dockershim. Basic e2e test infrastructure in place in Azure leveraging the test suites for Windows e2e along with dedicated DC host VMs. Automated testing will target Docker container run time but some manual testing of ContainerD integration also needs to succeed.
- ga - e2e tests passing consistently and tests targeting ContainerD/RunHCS  passing as well assuming ContainerD/RunHCS for Windows is stable.


## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded

## Drawbacks [optional]

Why should this KEP _not_ be implemented.

## Alternatives

### Other authentication methods

There are other ways to handle user-service and service-service authentication, but they generally require code changes. This proposal is focused on enabling customers to use existing on-premises Active Directory identity in containers.

For cloud-native applications, there are other alternatives:

- Kubernetes secrets - if both services are run in Kubernetes, this can be used for username/password or preshared secrets available to each app
- PKI - If you have a PKI infrastructure, you could choose to deploy application-specific certificates and change applications to trust specific public keys or intermediate certificates
- Cloud-provider service accounts - there may be other token-based providers available in your cloud. Apps can be modified to use these tokens and APIs for authentication and authorization requests.

### Injecting credentials from a volume

For certain authentication use cases, a preferred approach may be to surface a volume to the pod with the necessary data that a pod needs to assume an identity injected in the volume. In these cases, the container needs to implement logic to consume and act on the injected data.

In case of GMSA support, nothing inside the containers of a pod perform any special steps around assuming an identity as that is taken care of by the container runtime at container startup. A container runtime driven solution like GMSA however does require CRI enhancements as mentioned earlier.

### Specifying only the name of GMSACredentialSpec objects in pod spec fields/annotations

To keep the pod spec changes minimal, we considered having a single field/annotation that specifies the name of the GMSACredentialSpec object (rather than an additional field that is populated with the contents of the credential spec). This approach had the following drawbacks compared to retrieving and storing the credential spec data inside annotations/fields:

- Complicates the Windows CRI code with logic to look up GMSACredentialSpec objects which may be failure prone.
- Requires the kubelet to be able to access GMSACredentialSpec objects which may require extra RBAC configuration in a locked down environment.
- Contents of `credspec` in a GMSACredentialSpec object being referred to may change after pod creation. This leads to confusing behavior.

### Enforce presence of GMSAAuthorizer and RBAC mode to enable GMSA functionality in Kubelet

In order to enforce authorization of service accounts in a cluster before they can be used in conjunction with an approved GMSA, we considered adding checks in the Kubelet layer processing GMSA credential specs. Such enforcement however does not align well with parallel mechanisms and also leads to core Kubernetes code being opinionated about something that may not be necessary.

Today, if PSP or RBAC mode is not configured in a cluster, nothing stops pods with special capabilities from being scheduled. To align with this, we should allow GMSA configurations on pods to be enabled without requiring GMSAAuthorizer to be running and RBAC mode to be enabled.

Further, decoupling basic GMSA functionality in the Kubelet and CRI layers from authorization keeps the core Kuberenetes code non-opinionated around enforcement of authorization of service accounts for GMSA usage. Kubernetes cluster setup tools as well as Kubernetes distribution vendors can ensure that RBAC mode is enabled and GMSAAuthorizer is configured and installed when Windows nodes joined to a domain are deployed in a cluster.


<!-- end matter -->
<!-- references -->
[oci-runtime](https://github.com/opencontainers/runtime-spec/blob/master/config-windows.md#credential-spec)
[manage-serviceaccounts](https://docs.microsoft.com/en-us/virtualization/windowscontainers/manage-containers/manage-serviceaccounts)
