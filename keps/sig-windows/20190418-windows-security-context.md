---
title: Windows security context API changes
authors:
  - "@ddebroy"
owning-sig: sig-windows
participating-sigs:
reviewers:
  - "@patricklang"
  - "@liggitt"
approvers:
  - "@liggitt"
editor: TBD
creation-date: 2019-04-18
last-updated: 2019-04-18
status: provisional
see-also:
  - "keps/sig-windows/20181221-windows-group-managed-service-accounts-for-container-identity.md"
replaces:
superseded-by:
---

# Windows specific options in Pod Security Context and Container Security Context

## Table of Contents

- [Title](#title)
  - [Table of Contents](#table-of-contents)
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Design Details](#design-details)
    - [Test Plan](#test-plan)
    - [Graduation Criteria](#graduation-criteria)
    - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Version Skew Strategy](#version-skew-strategy)
  - [Implementation History](#implementation-history)
  - [Drawbacks [optional]](#drawbacks-optional)
  - [Alternatives [optional]](#alternatives-optional)
  - [Infrastructure Needed [optional]](#infrastructure-needed-optional)

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

In this KEP, we propose API enhancements in the Kubernetes pod spec to capture Windows OS specific security options from the perspective of Windows workload identity in containers. Initially the enhancements will cover fields pertinent to GMSA credential specs and the username with which to execute the container entry-point. More fields may be added in the future.

## Motivation

 There are two important motivations for the API enhancements described in this KEP:
 
 1. With the introduction of alpha support for GMSA in Kubernetes v1.14, references to GMSA credential spec custom resources need to be specified through annotations at the pod and container level. However, as detailed in the related [GMSA](https://github.com/kubernetes/enhancements/blob/master/keps/sig-windows/20181221-windows-group-managed-service-accounts-for-container-identity.md) KEP, we want the ability to specify references to GMSA credential specs directly in the pod/container specs as fields (without having to use annotations) beyond the alpha stage.
 
 2. The Windows dockershim, CRI and OCI spec can already handle a username (instead of UID), which is interpreted inside the container to create a process as the intended user. This however is not surfaced as a field in the pod/container specs that an operator can specify. We want the ability to specify the desired username in the pod/container specs as fields and be able to pass them to the configured Windows runtime.

### Goals

Propose API enhancements in existing `PodSecurityContext` and `SecurityContext` structs for pods and individual containers respectively to allow operators to specify:
- Name of a GMSA credential spec custom resource 
- Full GMSA credential spec JSON 
- A Windows username whose identity will be used to kick off the entrypoint process in containers.

### Non-Goals

- Details around GMSA end2end functionality and user stories as that is covered in the [GMSA](https://github.com/kubernetes/enhancements/blob/master/keps/sig-windows/20181221-windows-group-managed-service-accounts-for-container-identity.md) KEP.
- Implementation details and security considerations around how a GMSACredentialSpecName is expanded to GMSACredentialSpec as that is covered in details in the [GMSA](https://github.com/kubernetes/enhancements/blob/master/keps/sig-windows/20181221-windows-group-managed-service-accounts-for-container-identity.md) KEP.
- Details around how GMSA credential specs or Windows Username is passed through CRI and interpreted by Windows container run-times like Docker or ContainerD. Enhancements related to GMSA in CRI is already covered in the [GMSA](https://github.com/kubernetes/enhancements/blob/master/keps/sig-windows/20181221-windows-group-managed-service-accounts-for-container-identity.md) KEP. Enhancements related to Username in CRI was introduced a while back in a [PR](https://github.com/kubernetes/kubernetes/pull/64009) 

## Proposal

In this KEP we propose a new field named `WindowsOptions` in the `PodSecurityContext` struct (associated with a pod spec) and the `SecurityContext` struct (associated with each container in a pod). `WindowsOptions` will be a pointer to a new struct of type `WindowsSecurityOptions`. The `WindowsSecurityOptions` struct will contain Windows OS specific security attributes scoped either at the pod level (applicable to all containers in the pod) or at the individual container level (that can override the pod level specification). Initially, the `WindowsSecurityOptions` struct will have the following fields:

- GMSACredentialSpecName: A string specifying the name of a GMSA credential spec custom resource
- GMSACredentialSpec: A string specifying the full credential spec JSON string associated with a GMSA credential spec
- RunAsUserName: A string specifying the user name in Windows to run the entrypoint of the container

More fields may be added to the `WindowsSecurityOptions` struct as desired in the future.

### Implementation Details/Notes/Constraints [optional]

The `WindowsSecurityOptions` struct will be defined as follows in pkg/apis/core/types.go

```
// WindowsSecurityOptions contain Windows-specific options and credentials.
type WindowsSecurityOptions struct {
	// GMSACredentialSpecName is the name of the GMSA credential spec to use.
	// +optional
	GMSACredentialSpecName string

	// GMSACredentialSpec is where the GMSA admission webhook
	// (https://github.com/kubernetes-sigs/windows-gmsa) inlines the contents of the
	// GMSA credential spec named by the `GmsaCredentialSpecName` field.
	// +optional
	GMSACredentialSpec string

	// RunAsUserName is the local user context used to log in to the container
	// +optional
	RunAsUserName string
}
```

Field `WindowsOptions *WindowsSecurityOptions` will be added to `SecurityContext` and `PodSecurityContext` structs.

#### Specification of both GMSA credspec and RunAsUserName

Note that both GMSA credspec and RunAsUserName may be specified. Specification of one field is not mutually exclusive with the other. RunAsUserName governs the local user identity used to log into the container. This is decoupled from the GMSA domain identity used to interact with network resources. To use GMSA identity, processes in the container should run as "Local System" or "Network Service" users. However Kubernetes won't enforce any rules around specification of these fields. For further details, please refer [here](https://docs.microsoft.com/en-us/virtualization/windowscontainers/manage-containers/manage-serviceaccounts#configuring-your-application-to-use-the-gmsa)

#### Changes in kubelet

An effective value for a field in a container's `SecurityContext.WindowsOptions` will be determined by calling `DetermineEffectiveSecurityContext`. If a value is specified for a field in the pod's `PodSecurityContext.WindowsOptions`, `DetermineEffectiveSecurityContext` will apply it to the corresponding field in a container's `SecurityContext.WindowsOptions` unless that field is already set with a different value in the container's `SecurityContext.WindowsOptions`.

### Risks and Mitigations

None

## Design Details

### Test Plan

Unit tests will be added around `DetermineEffectiveSecurityContext` to ensure for each field in `WindowsSecurityOptions` the values in a container's `SecurityContext.WindowsOptions` can override values of the fields in the pod's `PodSecurityContext.WindowsOptions`.

Overall test plans for GMSA are covered in the GMSA KEP.

### Graduation Criteria

The API change introduced in this KEP will not be feature gated in any form. So this section is not applicable in the context of this KEP.

### Upgrade / Downgrade Strategy

### Version Skew Strategy

## Implementation History

## Drawbacks [optional]

## Alternatives [optional]

The main alternatives to the API changes is to continue to use annotations as they are done already for GMSA. 
