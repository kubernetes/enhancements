---
kep-number: 0000
title: Windows Group Managed Service Accounts for Container Identity
authors:
  - "@patricklang,@jeremywx,@ddebroy"
owning-sig: "sig-windows"
participating-sigs:
  - "sig-auth"
  - "sig-windows"
reviewers:
  - "sig-node"
  - "sig-auth"
  - "sig-arch"
approvers:
  - "sig-node"
  - "sig-auth"
  - "sig-arch"
editor: TBD
creation-date: 2018-06-20
last-updated: 2018-12-27
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
- Prevent pods from being inadvertently scheduled with access to a GMSA that the operator shouldn't have rights to

### Non-Goals

- Running Linux applications using GMSA or a general Kerberos based authentication system
- Replacing any existing Kubernetes authorization or authentication controls
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

1. Active Directory domain administrator will:

  - Join all Windows Kubernetes nodes to the Active Directory domain.
  - Provision the GMSA and gives details to application admin.
  - Assign access to a AD security group to control what machines can use this GMSA. The AD security group should include all authorized Kubernetes nodes.

2. A Kubernetes admin will create a configmap (in the desired namespace) and populate it with configuration information associated with the GMSA:
  - Run a Windows PowerShell script to verify the account exists, and capture enough data to uniquely identify the GMSA into a JSON file (credspec). This doesn't contain any passwords or crypto secrets.

Example credspec.json
```
{
  "CmsPlugins": [
    "ActiveDirectory"
  ],
  "DomainJoinConfig": {
    "DnsName": "contoso.com",
    "Guid": "244818ae-87ca-4fcd-92ec-e79e5252348a",
    "DnsTreeName": "contoso.com",
    "NetBiosName": "DEMO",
    "Sid": "S-1-5-21-2126729477-2524075714-3094792973",
    "MachineAccountName": "WebApplication1"
  },
  "ActiveDirectoryConfig": {
    "GroupManagedServiceAccounts": [
      {
        "Name": "WebApplication1",
        "Scope": "DEMO"
      },
      {
        "Name": "WebApplication1",
        "Scope": "contoso.com"
      }
    ]
  }
}
```

  - Create a configmap in the namespace with the contents of the JSON - `kubectl create configmap webserver-credspec --from-file=gmsa-cred-spec=/path/to/credspec.json`

3. Secure the configmap with the GMSA credspec so that only desired users/service accounts within a namespace can view, modify or use the GMSA configmap:
  - Create a role to view, modify or use the GMSA configmap:

```
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  namespace: default
  name: webserver-gmsa-user
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  resourceNames: ["webserver-credspec"]
  verbs: ["get", "patch", "delete", "update", "use"]
```

  - Create a rolebinding and assign desired users (for example, jane below) or service accounts to the above role:

```
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: use-webserver-gmsa
  namespace: default
subjects:
- kind: User
  name: jane
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: webserver-gmsa-user
  apiGroup: rbac.authorization.k8s.io
```
  
4. Application admins deploy app pods with a reference to the name of an authorized GMSA configmap (e.g. `webserver-credspec`). In the alpha and beta stage of this feature, the reference to the GMSA configmap will be set through an annotation on the pod: `pod.[alpha|beta].kubernetes.io/windows-gmsa-config-map`. In the beta and GA stage, the reference to GMSA configmap may be set through `spec.securityContext.windows.credentialSpecConfig` in the pod spec.
5. During pod creation, a new Admission Controller, GMSAAuthorizer, will ensure the `user` (and `spec.serviceAccountName` if specified) is authorized for the `use` verb on the GMSA configmap. If the authorization check fails due to absence of RBAC roles, the pod creation fails.
6. The Windows CRI implementation accesses the configmap with the GMSA credspec and copies the contents of entry with key `gmsa-cred-spec` to the [OCI windows.CredentialSpec](https://github.com/opencontainers/runtime-spec/blob/master/config-windows.md#credential-spec) field.
7. The Windows OCI implementation validates the credspec and fails to start the container if its invalid or access to the GMSA from the node is denied.
8. The container starts with the Windows Group Managed Service Account as expected, and can authenticate to the database.
9. During any subsequent pod update, any changes to a pod's `securityContext` is blocked (as is the case today) by `ValidatePodUpdate`.


### Implementation Details/Notes/Constraints [optional]

#### Reference to the GMSA configmap from pods
In the Alpha and Beta phase of this feature, the reference to the GMSA configmap will be set through an annotation: `pod.[alpha|beta].kubernetes.io/windows-gmsa-config-map` on the pod. In the GA phase, a new Windows OS-specific field `CredentialSpecConfig String` will be added to `SecurityContext` to be populated with the name of the GMSA credspec configmap.

#### GMSAAuthorizer Admission Controller
A new admission controller, GMSAAuthorizer will be implemented to act on pod creation and updates. 

During pod creation, the admission controller will first check if the reference to a GMSA configmap is set (either through annotation `pod.[alpha|beta].kubernetes.io/windows-gmsa-config-map` or pod's `securityContext.windows.credentialSpecConfig` [in GA]) on the pod. If set, the admission controller will ensure the user of the request as well as any service account specified for the pod are authorized for a special `use` verb on the GMSA configmap referenced from the pod. The admission controller will generate custom `AttributesRecord`s with `verb` set to `use`, `name` set to the GMSA configmap and `user` set to the user of the request as well as service account if specified. Next, the `AttributesRecord`s will be passed to authorizers to check against RBAC configurations.

During pod updates, changes to the `pod.[alpha|beta].kubernetes.io/windows-gmsa-config-map` annotation will be blocked. `ValidatePodUpdate` already ensures a pod's `securityContext` cannot be updated. So no further checks/blocks are necessary to prevent update operations around the reference to the credspec configmap on pods in GA.

#### Changes in CRI's WindowsContainerSecurityContext struct
A new field `CredentialSpec String` will be added to `WindowsContainerSecurityContext`. It will be populated by `generateWindowsContainerConfig` for Windows containers with the contents of key `gmsa-cred-spec` in the GMSA configmap reference from the pod (either through the annotation `pod.[alpha|beta].kubernetes.io/windows-gmsa-config-map` or `securityContext.windows.credentialSpecConfig` in the pod spec).

#### Changes in Dockershim
The `applyWindowsContainerSecurityContext` function will create a temporary file with a unique name on the host file system with the contents of the credspec. The file path will be used to populate `HostConfig.SecurityOpt` with a credspec file specification. The credspec file will be deleted after `StartContainer` in invoked as well as in various error handling paths between container creation and start.

#### Changes in Windows OCI runtime
The Windows OCI runtime already has support for `windows.CredentialSpec` and is implemented in Moby (Docker).

### Risks and Mitigations

#### Threat vectors and countermeasures
1. Prevent an unauthorized user from referring to an existing GMSA configmap in the pod spec: The GMSAAuthorizer Admission Controller along with RBAC policies with the `use` verb on a GMSA configmap ensures only users allowed by the kubernetes admin can refer to the GMSA configmap in the pod spec.
2. Prevent an unauthorized user from using an existing Service Account that is authorized to use an existing GMSA configmap: The GMSAAuthorizer Admission Controller checks the `user` as well as the service account associated with the pod have `use` rights on the GMSA configmap.
3. Prevent an unauthorized user from reading the GMSA credspec and using it directly through docker on Windows hosts connected to AD that user has access to: RBAC policy on the GMSA configmaps should only allow `get` verb for authorized users.

## Graduation Criteria

- alpha - initial implementation
- beta - design validated by two or more customer proof of concept deployments, recorded success in SIG-Windows meetings or mailing lists
- ga - Node E2E tests in place, tagged with features GMSA & Windows


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

### Associate GMSAs with Kubernetes Service Accounts

We considered associating/mapping the GMSA credspec to Kubernetes Service Accounts. The problem with this approach is that there is no way to restrict a user from specifying service accounts to use for the application pods within a given namespace. We do not want a user who is not specifically authorized to use a GMSA credspec by kubernetes admins within a given namespace to be able to create pods with that GMSA credspec. If there is a mechanism to restrict users from being able to specify arbitrary service accounts (as part of podspec) in a namespace, this path maybe considered.

<!-- end matter --> 
<!-- references -->
[oci-runtime](https://github.com/opencontainers/runtime-spec/blob/master/config-windows.md#credential-spec)
[manage-serviceaccounts](https://docs.microsoft.com/en-us/virtualization/windowscontainers/manage-containers/manage-serviceaccounts)
