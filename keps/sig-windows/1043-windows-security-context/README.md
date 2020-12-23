# Windows specific options in Pod Security Context and Container Security Context

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Specify GMSA credential spec and RunAsUserName for a pod](#specify-gmsa-credential-spec-and-runasusername-for-a-pod)
    - [Specify distinct GMSA credential spec for a container in a pod (while retaining RunAsUserName from pod spec)](#specify-distinct-gmsa-credential-spec-for-a-container-in-a-pod-while-retaining-runasusername-from-pod-spec)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    - [Validation of fields](#validation-of-fields)
    - [Specification of both GMSA credspec and RunAsUserName](#specification-of-both-gmsa-credspec-and-runasusername)
    - [Changes in kubelet](#changes-in-kubelet)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade and Version Skew Strategy](#upgrade--downgrade-and-version-skew-strategy)
- [Implementation History](#implementation-history)
- [Implementation Roadmap](#implementation-roadmap)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
<!-- /toc -->

## Release Signoff Checklist

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

In this KEP, we propose API enhancements in the Kubernetes pod spec to capture Windows OS specific security options from the perspective of Windows workload identity in containers. Initially the enhancements will cover fields pertinent to GMSA credential specs and the username with which to execute the container entry-point. More fields may be added in the future. Please note that this is a KEP with a very limited scope focussing mainly on the API enhancements needed to support GMSA credential spec details and the RunAsUsername field. Details around overall GMSA functionality can be found in the [GMSA](https://github.com/kubernetes/enhancements/blob/master/keps/sig-windows/20181221-windows-group-managed-service-accounts-for-container-identity.md) KEP and are not repeated in this KEP.

## Motivation

 There are two important motivations for the API enhancements described in this KEP:
 
 1. With the introduction of Alpha support for GMSA in Kubernetes v1.14, references to GMSA credential spec custom resources need to be specified through annotations at the pod and container level. However, as detailed in the related [GMSA](https://github.com/kubernetes/enhancements/blob/master/keps/sig-windows/20181221-windows-group-managed-service-accounts-for-container-identity.md) KEP, we want the ability to specify references to GMSA credential specs directly in the pod/container specs as fields (without having to use annotations) beyond the Alpha stage.
 
 2. The Windows implementation of the dockershim, CRI and the low-level OCI spec can already handle a username (instead of UID), which is interpreted inside the container to create a process as the intended user. This however is not surfaced as a field in the pod/container specs that an operator can specify. We want the ability to specify the desired username in the pod/container specs as fields and be able to pass them to the configured Windows runtime.

### Goals

Propose API enhancements in existing `PodSecurityContext` and `SecurityContext` structs for pods and individual containers respectively to allow operators to specify:
- Name of a GMSA credential spec custom resource 
- Full GMSA credential spec JSON
- A Windows username whose identity will be used to kick off the entrypoint process in containers.

### Non-Goals

- Details around GMSA end-2-end functionality and interaction with webhooks that is covered in the [GMSA](https://github.com/kubernetes/enhancements/blob/master/keps/sig-windows/20181221-windows-group-managed-service-accounts-for-container-identity.md) KEP.
- Implementation details and security considerations around how a GMSACredentialSpecName is expanded to GMSACredentialSpec as that is covered in details in the [GMSA](https://github.com/kubernetes/enhancements/blob/master/keps/sig-windows/20181221-windows-group-managed-service-accounts-for-container-identity.md) KEP.
- Details around how GMSA credential specs or Windows username is passed through CRI and interpreted by Windows container run-times like Docker or ContainerD. Enhancements related to GMSA in CRI is already covered in the [GMSA](https://github.com/kubernetes/enhancements/blob/master/keps/sig-windows/20181221-windows-group-managed-service-accounts-for-container-identity.md) KEP. Enhancements related to Username in CRI was introduced a while back in a [PR](https://github.com/kubernetes/kubernetes/pull/64009) 

## Proposal

In this KEP we propose a new field named `WindowsOptions` in the `PodSecurityContext` struct (associated with a pod spec) and the `SecurityContext` struct (associated with each container in a pod). `WindowsOptions` will be a pointer to a new struct of type `WindowsSecurityContextOptions`. The `WindowsOptions` struct will contain Windows OS specific security attributes scoped either at the pod level (applicable to all containers in the pod) or at the individual container level (that can override the pod level specification). This is inspired by the existing `SELinuxOptions` field that groups Linux specific SELinux options. Initially, the `WindowsSecurityContextOptions` struct will have the following fields:

- GMSACredentialSpecName: A string specifying the name of a GMSA credential spec custom resource
- GMSACredentialSpec: A string specifying the full credential spec JSON string associated with a GMSA credential spec
- RunAsUserName: A string specifying the user name in Windows to run the entrypoint of the container

More fields may be added to the `WindowsSecurityContextOptions` struct as desired in the future.

Note that the GMSACredentialSpec will almost never be populated by operators. It will be auto populated by a webhook that will look up the credential spec JSON associated with GMSACredentialSpecName and expand it. Details of the interactions with the webhook is covered in [GMSA](https://github.com/kubernetes/enhancements/blob/master/keps/sig-windows/20181221-windows-group-managed-service-accounts-for-container-identity.md) KEP

### User Stories

There could be various ways in which an operator may specify individual members of `WindowsOptions` in a pod spec scoped either at the pod level or at an individual container level.

#### Specify GMSA credential spec and RunAsUserName for a pod

```
apiVersion: v1
kind: Pod
metadata:
  name: iis
  labels:
    name: iis
spec:
  securityContext:
    windowsOptions:
      gmsaCredentialSpecName: webapp1-credspec
      runAsUserName: "NT AUTHORITY\\NETWORK SERVICE"
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

#### Specify distinct GMSA credential spec for a container in a pod (while retaining RunAsUserName from pod spec)

```
apiVersion: v1
kind: Pod
metadata:
  name: iis
  labels:
    name: iis
spec:
  securityContext:
    windowsOptions:
      gmsaCredentialSpecName: webapp1-credspec
      runAsUserName: "NT AUTHORITY\\NETWORK SERVICE"
  containers:
    - name: iis
      image: mcr.microsoft.com/windows/servercore/iis:windowsservercore-ltsc2019
      ports:
        - containerPort: 80
    - name: logger
      securityContext:
        windowsOptions:
          gmsaCredentialSpecName: eventlogger-credspec
      image: eventlogger:2019
      ports:
        - containerPort: 80
  nodeSelector:
    beta.kubernetes.io/os : windows
```

### Implementation Details/Notes/Constraints [optional]

The `WindowsSecurityContextOptions` struct will be defined as follows in pkg/apis/core/types.go

```
// WindowsSecurityContextOptions contain Windows-specific options and credentials.
type WindowsSecurityContextOptions struct {
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

Field `WindowsOptions *WindowsSecurityContextOptions` will be added to `SecurityContext` and `PodSecurityContext` structs.

#### Validation of fields

The fields within the `WindowsSecurityContextOptions` struct will be validated as follows:
- GMSACredentialSpecName: This field will be the name of a custom resource. It should follow the rules associated with [naming of resources](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names) in Kubernetes. Validation will make sure the maximum number of characters is 253  and consist of lower case alphanumeric characters, -, and .
- GMSACredentialSpec: The size of the CredentialSpec JSON blob should be limited to avoid abuse. We will limit it to 64K which allows for a lot of room to specify extremely complicated credential specs. Typically this JSON blob is not expected to be more than 1K based on experience so far.
- RunAsUserName: This field needs to allow for a valid set of usernames allowed for Windows containers. Currently the OCI spec or Windows documentation does not specify any clear restrictions around the length of this parameter or restrictions around usage of special characters when passed to the container runtime and eventually to Windows HCS. So Kubernetes validation won't enforce any character set validation. A maximum length of 256 characters will be allowed to prevent abuse.

At this time, no additional API validation logic will be implemented on the pod spec's existing security context structs to check `SELinuxOptions` or other Linux specific fields like `RunAsUser`/`RunAsGroup` cannot be specified along with `WindowsOptions` within the same security context. If such validation and fast fail behavior is desired and external admission controller can implement the additional checks in future.

#### Specification of both GMSA credspec and RunAsUserName

Note that both GMSA credspec and RunAsUserName may be specified. Specification of one field is not mutually exclusive with the other. RunAsUserName governs the local user identity used to log into the container. This is decoupled from the GMSA domain identity used to interact with network resources. To use GMSA identity, processes in the container should run as "Local System" or "Network Service" users. However Kubernetes won't enforce any rules around specification of these fields. For further details, please refer [here](https://docs.microsoft.com/en-us/virtualization/windowscontainers/manage-containers/manage-serviceaccounts#configuring-your-application-to-use-the-gmsa)

#### Changes in kubelet

An effective value for a field in a container's `SecurityContext.WindowsOptions` will be determined by calling `DetermineEffectiveSecurityContext`. If a value is specified for a field in the pod's `PodSecurityContext.WindowsOptions`, `DetermineEffectiveSecurityContext` will apply it to the corresponding field in a container's `SecurityContext.WindowsOptions` unless that field is already set with a different value in the container's `SecurityContext.WindowsOptions`.

### Risks and Mitigations

None

## Design Details

### Test Plan

Unit tests will be added around `DetermineEffectiveSecurityContext` to ensure for each field in `WindowsOptions` the values in a container's `SecurityContext.WindowsOptions` can override values of the fields in the pod's `PodSecurityContext.WindowsOptions`.

An e2e test case with `[sig-windows]` label will be added to the Windows test runs to exercise and verify RunAsUserName is correctly taking effect. The test will try to bring up a pod with a container based on NanoServer, default user `ContainerUser` and the entrypoint trying to execute a privileged operation like changing the routing table inside the container. The pod spec will specify `ContainerAdministrator` as the RunAsUserName to verify the privileged operation within the container can successfully execute. Without RunAsUserName working correctly, the default user in the container will fail to execute the privileged operation and the container will exit prematurely.

Overall test plans for GMSA are covered in the [GMSA](https://github.com/kubernetes/enhancements/blob/master/keps/sig-windows/20181221-windows-group-managed-service-accounts-for-container-identity.md) KEP.

### Graduation Criteria

The API change introduced in this KEP will not be feature gated in any form but will be considered "alpha" fields initially. The code for populating the proposed fields in this KEP will be feature gated in an `alpha` state for at least one release as detailed in the Version Skew Strategy section below.

### Upgrade / Downgrade and Version Skew Strategy

Only feature gated code will populate the new `WindowsOptions` field in the Kubernetes release that the field is introduced in (e.g. 1.15). If the feature gates are disabled (default configuration), the new fields will be set to empty/zero values during creation and updates. After stabilizing for one release cycle, code paths that populate the `WindowsOptions` field may be enabled by default (e.g. in 1.16). This will adhere with the [published guidelines](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md#alpha-field-in-existing-api-version) for introducing "alpha" fields in current stable APIs. This rollout plan ensures a smooth upgrade in the context of persistence of data for the new fields in clusters with masters/APIServer in a HA configuration.

## Implementation History

https://github.com/kubernetes/kubernetes/pull/73609 introducing RunAsUsername was held back from 1.14 since it was too late for API review. This PR will be rebased and adjusted to meet this KEP's proposed API.

## Implementation Roadmap

Here is the planned sequence of PRs to introduce the API change:

First, we will introduce an empty `WindowsSecurityContextOptions` struct and corresponding `WindowsOptions` fields in the `PodSecurityContext` and `SecurityContext` structs.

Next, independent feature oriented PRs [for GMSA and RunAsUsername for now] will introduce the necessary fields in the `WindowsSecurityContextOptions` struct. These PRs need to follow existing guidelines around API spec changes: API changes in an isolated commit, generated changes in an isolated commit, and implementation in one or more commits.

If development of the features progress in parallel, generated protobuf field IDs may conflict and require appropriate updates and rebasing.

## Drawbacks [optional]

## Alternatives [optional]

The main alternatives to the API changes is to continue to use annotations as they are done already for GMSA. However, we do not want to continue to rely on annotations for a feature as it matures.

We considered the option to embed the Windows specific fields directly into the `PodSecurityContext` and `SecurityContext` structs (without grouping them into the `WindowsOptions` struct). However, grouping the fields into OS-specific structs will align more with potential future re-design efforts of the pod spec where there will be structs grouping OS independent fields and OS specific fields. Some of the current Linux fields that do not apply to Windows will not be grouped right away for backward compatibility.
