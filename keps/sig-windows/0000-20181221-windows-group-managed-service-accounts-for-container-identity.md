---
kep-number: 0000
title: Windows Group Managed Service Accounts for Container Identity
authors:
  - "@patricklang,@jeremywx,@ddebroy"
owning-sig: "sig-windows"
participating-sigs:
  - "sig-auth"
  - "sig-node"
  - "sig-windows"
reviewers:
  - @liggitt
  - @@mikedanese
  - @yujuhong
  - @patricklang
approvers:
  - @liggitt
  - @yujuhong
  - @patricklang
editor: TBD
creation-date: 2018-06-20
last-updated: 2019-01-16
status: provisional
---

# Windows Group Managed Service Accounts for Container Identity


## Table of Contents

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Proposal](#proposal)
    * [User Stories [optional]](#user-stories-optional)
      * [Web Applications with MS SQL Server](#Web-Applications-with-MS-SQL-Server)
    * [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    * [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)
* [Drawbacks [optional]](#drawbacks-optional)
* [Alternatives [optional]](#alternatives-optional)

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


## Proposal

### Background

#### What is Active Directory?
Windows applications and services typically use Active Directory identities to facilitate authentication and authorization between resources. In a traditional virtual machine scenario, the computer is joined to an Active Directory domain which enables it to use Kerberos, NTLM, and LDAP to identify and query for information about other resources in the domain. When a computer is joined to Active Directory, it is given an unique identity represented as a computer object in LDAP.

#### What is a Windows service account?
A Group Managed Service Account (GMSA) is a shared Active Directory identity that enables common scenarios such as authenticating and authorizing incoming requests and accessing downstream resources such as a database server, file share, or other workload. It can be used by multiple authorized users or computers at the same time.

#### How is it applied to containers?
To achieve the scale and speed required for containers, Windows uses a group managed service account in lieu of individual computer accounts to enable Windows containers to communicate with Active Directory. As of right now, the Host Computer Service (which exposes the interface to manage containers) in Windows cannot use individual Active Directory computer & user accounts - it only supports GMSA.

Different containers on the same machine can use different GMSAs to represent the specific workload they are hosting, allowing operators to granularly control which resources a container has access to. However, to run a container with a GMSA identity, an additional parameter must be supplied to the Windows Host Compute Service to indicate the intended identity. This proposal seeks to add support in Kubernetes for this parameter to enable Windows containers to communicate with other enterprise resources.

It's also worth noting that Docker implements this in a different way that's not managed centrally. It requires dropping a file on the node and referencing it by name, eg: docker run --credential-spec='file://foo.json' . For more details see the Microsoft doc.


### User Stories


#### Web Applications with MS SQL Server
A developer has a Windows web app that they would like to deploy in a container with Kubernetes. For example, it may have a web tier that they want to scale out running ASP.Net hosted in the IIS web server. Backend data  is stored in a Microsoft SQL database, which all of the web servers need to access behind a load balancer. An Active Directory Group Managed Service Account is used to avoid hardcoded passwords, and the web servers run with that credential today. The SQL Database has a user with read/write access to that GMSA so the web servers can access it. When they move the web tier into a container, they want to preserve the existing authentication and authorization models.

When this app moves to production on containers, the team will need to coordinate to make sure the right GMSA is carried over to the container deployment.

1. An Active Directory domain administrator will:

  - Join all Windows Kubernetes nodes to the Active Directory domain.
  - Provision the GMSA and gives details to application admin.
  - Assign access to a AD security group to control what machines can use this GMSA. The AD security group should include all authorized Kubernetes nodes.

2. A Kubernetes cluster admin (or cluster setup tools or an operator) will install a cluster scoped CRD (of kind GMSACredSpec) whose instances will be used to store GMSA credential spec configuration:

```
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: gmsacredspecs.auth.k8s.io
spec:
  group: auth.k8s.io
  version: v1alpha1
  names:
    kind: GMSACredSpec
    plural: gmsacredspecs
  scope: Cluster
```

A GMSACredSpec may be used by sidecar containers across different namespaces. Therefore the CRD needs to be cluster scoped.

3. A Kubernetes cluster admin will create a GMSACredSpec object populated with the credspec configuration associated with a desired GMSA:

  - The cluster admin may run a Windows PowerShell script to generate the YAML definition of a GMSACredSpec object populated with the GMSA credspec details. This doesn't contain any passwords or crypto secrets.

Example credspec.yaml for GMSA webapplication1:

```
apiVersion: auth.k8s.io/v1alpha1
kind: GMSACredSpec
metadata:
  name: "webapp1-credspec"
credspec:
  ActiveDirectoryConfig:
    GroupManagedServiceAccounts:
    - Name: WebApplication1
      Scope: DEMO
    - Name: WebApplication1
      Scope: contoso.com
  CmsPlugins:
  - ActiveDirectory
  DomainJoinConfig:
    DnsName: contoso.com
    DnsTreeName: contoso.com
    Guid: 244818ae-87ca-4fcd-92ec-e79e5252348a
    MachineAccountName: WebApplication1
    NetBiosName: DEMO
    Sid: S-1-5-21-2126729477-2524075714-3094792973
```
  
  - With the YAML from above (or generated manually), the cluster admin will create a GMSACredSpec object in the cluster.

4. A Kubernetes cluster admin will configure RBAC for the GMSACredSpec so that only desired service accounts can use the GMSACredSpec:

  - Create a cluster wide role to get and "use" the GMSACredSpec:

```
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: webapp1-gmsa-user
rules:
- apiGroups: ["auth.k8s.io"]
  resources: ["gmsacredspecs"]
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
  kind: Role
  name: webapp1-gmsa-user
  apiGroup: rbac.authorization.k8s.io
```
  
5. Application admins will deploy app pods that require a GMSA identity along with a Service Account authorized to use the GMSAs. There will be two options regarding how to specify the GMSACredSpec for pods and containers:

  - Specify the name of the desired GMSACredSpec (e.g. `webapp1-credspec`): In the alpha stage of this feature, the name of the desired GMSACredSpec can be set through an annotation on the pod: `pod.alpha.kubernetes.io/windows-gmsa-credspec-name` as well as for each container through annotations of the form: `container.windows-gmsa-credspec-name.alpha.kubernetes.io/container-name`. In the beta stage, the annotations will be promoted to a field in the securityContext of the pod: `podspec.securityContext.windows.gmsaCredSpecName` and in the securityContext of each container:  `podspec.container[i].securityContext.windows.gmsaCredSpecName`. The GMSACredSpec name for a container will override the GMSACredSpec entries for the whole pod.

  - Specify the contents of the `credpec` field of GMSACredSpec that gets passed down to the runtime: In the alpha stage of this feature, a JSON representation of the contents of the desired GMSACredSpec can be set through an annotation on the pod: `pod.alpha.kubernetes.io/windows-gmsa-credspec` as well as for each container through annotations of the form: `container.windows-gmsa-credspec.alpha.kubernetes.io/container-name`. In the beta stage, the annotations will be promoted to a field the securityContext of the pod `podspec.securityContext.windows.gmsaCredSpec` and in the securityContext of each  container: `podspec.container[i].securityContext.windows.gmsaCredSpec`.

It is expected that users will typically use the first option as it is more user friendly. The ability to specify credspecs for each container within a pod aligns with how security attributes like `runAsGroup`, `runAsUser`, etc. can be specified at the pod level and overridden at the container level if necessary.

6. A mutating webhook admission controller, GMSAAuthorizer, will act on pod creations and execute the following:

  - Check the pod spec for references to names of GMSACredSpec objects in the pod (and per-container) annotations [for Alpha] or securityContext fields [Beta onwards]. Make sure the GMSACredSpec objects exist and use the contents of the `credspec` field of the GMSACredSpec to populate annotations `pod.alpha.kubernetes.io/windows-gmsa-credspec` and `container.windows-gmsa-credspec.alpha.kubernetes.io/container-name` [in Alpha] or `podspec.securityContext.windows.gmsaCredSpec` and `podspec.container[i].securityContext.windows.gmsaCredSpec` [Beta onwards]. If the specified names of GMSACredSpec cannot be resolved, pod creation will be failed with 404: Not Found.
  
  - Check the specified ServiceAccount is authorized for the `use` verb on all specified GMSACredSpecs at the pod level and for each container. If the authorization check fails due to absence of requisite RBAC roles, the pod creation will be failed with a 403: Forbidden. 

7. The Windows CRI implementation accesses the `gmsaCredSpec` field for each container in a pod and copies the contents to the [OCI windows.CredentialSpec](https://github.com/opencontainers/runtime-spec/blob/master/config-windows.md#credential-spec) field.

8. The Windows OCI implementation validates the credspec and fails to start the container if its invalid or access to the GMSA from the node is denied.

9. The container starts with the Windows Group Managed Service Account as expected, and can authenticate to the database.

10. During any subsequent pod update, any changes to a pod's `securityContext` will be blocked (as is the case today) by `ValidatePodUpdate`. Updates of the annotations for GMSACredSpec will be rejected by GMSAAuthorizer.


### Implementation Details/Notes/Constraints [optional]

#### GMSA specification for pods and containers
In the Alpha phase of this feature we will use annotations on a pod:
  - References to names of GMSACredSpec objects will be specified through the following annotations:
    - At the pod level: `pod.alpha.kubernetes.io/windows-gmsa-credspec-name`
    - At the container level: `container.windows-gmsa-credspec-name.alpha.kubernetes.io/container-name`
  - The contents of the credspec will be specified through or populated in the following annotations:
    - At the pod level: `pod.alpha.kubernetes.io/windows-gmsa-credspec`
    - At the container level: `container.windows-gmsa-credspec.alpha.kubernetes.io/container-name`

In the Beta phase of this feature, the annotations will graduate to fields in the pod spec:
  - References to names of GMSACredSpec objects will be specified through the following fields:
    - At the pod level `podspec.securityContext.windows.gmsaCredSpecName`
    - At the container level: `podspec.container[i].securityContext.windows.gmsaCredSpecName`
  - The contents of the credspec will be specified through or populated in the following fields:
    - At the pod level: `podspec.securityContext.windows.gmsaCredSpec`
    - At the container level: `podspec.container[i].securityContext.windows.gmsaCredSpec`

#### GMSAAuthorizer Admission Controller Webhook
A new admission controller webhook, GMSAAuthorizer will be implemented to act on pod creation and updates.

During pod creation, GMSAAuthorizer will first check for references to names of GMSACredSpec objects either through annotations or podspec fields. Each reference will be looked up using the name. If the name is not found, pod creation is failed with a 404 Not Found error. If found, the contents of the `credspec` field of the GMSACredSpec object will be used to populate the annotations `pod.alpha.kubernetes.io/windows-gmsa-credspec` and `podspec.container[i].securityContext.windows.gmsaCredSpecName` [in Alpha] or fields `podspec.securityContext.windows.gmsaCredSpec` and `podspec.container[i].securityContext.windows.gmsaCredSpec` [Beta onwards]. 

Next, GMSAAuthorizer will ensure the service account specified for the pod is authorized for a special `use` verb on the GMSACredSpec objects whose `credspec` field has been used to populate the gmsaCredSpec annotations/fields in the podspec. GMSAAuthorizer will generate custom `AttributesRecord`s with `verb` set to `use`, `name` set to the GMSACredSpec object and `user` set to the service account of the pod. Finally, the `AttributesRecord`s will be passed to authorizers to check against RBAC configurations. A failure from the authorizes results in a 403: Forbidden.

During pod updates, changes to the credspec annotations will be blocked by GMSAAuthorizer.

#### Changes in CRI's WindowsContainerSecurityContext struct and DetermineEffectiveSecurityContext
  - A new field `CredentialSpec String` will be added to `WindowsContainerSecurityContext`
  - `DetermineEffectiveSecurityContext` will be enhanced to populate `CredentialSpec` for Windows containers based on credspec specification for containers and pod. It will need to obtain the credspec contents from the `pod.alpha.kubernetes.io/windows-gmsa-credspec` and `container.windows-gmsa-credspec.alpha.kubernetes.io/container-name` annotations during the Alpha phase. The enhancements will be enabled under a feature flag during Alpha phase of the feature.

#### Changes in Dockershim
The `applyWindowsContainerSecurityContext` function will create a temporary file with a unique name on the host file system with the contents of `WindowsContainerSecurityContext.CredentialSpec`. The file path will be used to populate `HostConfig.SecurityOpt` with a credspec file specification. The credspec file will be deleted after `StartContainer` in invoked as well as in various error handling paths between container creation and start.

#### Changes in Windows OCI runtime
The Windows OCI runtime already has support for `windows.CredentialSpec` and is implemented in Moby (Docker).

### Risks and Mitigations

#### Threat vectors and countermeasures
1. Prevent an unauthorized user from referring to an existing GMSA configmap in the pod spec: The GMSAAuthorizer Admission Controller along with RBAC policies with the `use` verb on a GMSA configmap ensures only users allowed by the kubernetes admin can refer to the GMSA configmap in the pod spec.
2. Prevent an unauthorized user from using an existing Service Account that is authorized to use an existing GMSA configmap: The GMSAAuthorizer Admission Controller checks the `user` as well as the service account associated with the pod have `use` rights on the GMSA configmap.
3. Prevent an unauthorized user from reading the GMSA credspec and using it directly through docker on Windows hosts connected to AD that user has access to: RBAC policy on the GMSA configmaps should only allow `get` verb for authorized users.

## Graduation Criteria

- alpha - initial implementation with webhook and annotations on pods with CRI enhancements under feature flag disabled by default; manual e2e tests passing.
- beta - annotations promoted to fields in podspec and CRI feature flag enabled by default; basic e2e test infrastructure in place in Azure with Windows Active Directory.
- ga - e2e test enhancements and passing consistently 


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

For certain authentication use cases, a preferred approach may be to surface a volume to the pod with the necessary data that a pod needs to assume an identity injected in the volume. In case of GMSA support, nothing inside the containers of a pod perform any special steps around assuming an identity as that is taken care of by the container runtime at container startup. A container runtime driven solution like GMSA however does require CRI enhancements as mentioned earlier.

### Specifying only the name of GMSACredSpec objects in pod spec fields/annotations

To keep the pod spec changes minimal, we considered having a single field/annotation that specifies the name of the GMSACredSpec object (rather than an additional field that is populated with the contents of the credspec). This approach had the following drawbacks compared to retrieving and storing the credspec data inside annotations/fields:

- Complicates the Windows CRI code with logic to look up GMSACredSpec objects which may be failure prone.
- Requires the CRI code to be able to access GMSACredSpec objects which may require extra RBAC configuration in a locked down environment.
- Contents of `credspec` in a GMSACredSpec object being referred to may change after pod creation. This leads to confusing behavior.



<!-- end matter --> 
<!-- references -->
[oci-runtime](https://github.com/opencontainers/runtime-spec/blob/master/config-windows.md#credential-spec)
[manage-serviceaccounts](https://docs.microsoft.com/en-us/virtualization/windowscontainers/manage-containers/manage-serviceaccounts)
