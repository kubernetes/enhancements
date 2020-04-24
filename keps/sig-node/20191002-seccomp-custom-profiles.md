---
title: seccomp-custom-profiles
authors:
  - "@pjbgf"
owning-sig: sig-node
participating-sigs:
  - sig-auth
reviewers:
  - "@tallclair"
approvers:
  - "@tallclair"
editor: TBD
creation-date: 2019-10-02
last-updated: 2019-10-29
status: provisional
see-also:
  - "https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/seccomp.md"
  - "https://github.com/kubernetes/enhancements/pull/1148"
  - "https://github.com/kubernetes/enhancements/pull/1257"
---

# seccomp-custom-profiles

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Usage Scenarios](#usage-scenarios)
    - [Profile tampering protection](#profile-tampering-protection)
    - [Rollout of profile changes](#rollout-of-profile-changes)
    - [Starting containers with non-existent profile](#starting-containers-with-non-existent-profile)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
- [Design Details](#design-details)
  - [No persistence to disk](#no-persistence-to-disk)
  - [Feature Gate](#feature-gate)
  - [Test Plan](#test-plan)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
  - [Graduation Criteria](#graduation-criteria)
      - [Alpha](#alpha)
      - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
      - [Beta -&gt; GA Graduation](#beta---ga-graduation)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [Use CRD instead of ConfigMap](#use-crd-instead-of-configmap)
    - [Reasons for <code>ConfigMap</code>](#reasons-for-)
    - [Reasons against it](#reasons-against-it)
    - [Start the deprecation process for <code>localhost/&lt;path&gt;</code>](#start-the-deprecation-process-for-)
    - [Downstream seccomp support awareness](#downstream-seccomp-support-awareness)
- [References](#references)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

A proposal to enhance user-defined profiles support for seccomp by allowing them to be stored as `ConfigMap` objects.



## Motivation

The main motivation for this change is to bring wider seccomp adoption by making it easier for users to create and maintain their own seccomp profiles.
Use-defined profiles is a fundamental piece for users to get real value out of using seccomp, allowing them to decrease their cluster's attack surface. 

The current support relies on profiles being saved as files in the nodes. They need to exist across all cluster nodes where the pods using them may be scheduled. It is the user's responsibility to ensure the files are correctly saved on the nodes and their changes are synchronised as the profiles evolve. The scope of this proposal is to make that process more seamless.

Kubernetes supports Seccomp in some capacity since version 1.3, but from then on this feature has been kept largely untouched. This change comes quite timely as there are now two other KEPs bringing enhancements on this space by [making this feature GA](https://github.com/kubernetes/enhancements/pull/1148) and [enabling it by default on audit mode](https://github.com/kubernetes/enhancements/pull/1257). 


### Goals

- Add support to load seccomp profiles from ConfigMaps.
- Provide a new mechanism for profiles to be sent to CRI without the use of files in the node's filesystem.
- Avoid breaking changes for Kubernetes api and user workloads.


### Non-Goals

- Changes to make Seccomp GA. This is being covered by another [KEP](20190717-seccomp-ga.md).
- Changes to `PodSecurityPolicy`.
- Windows Support.



## Proposal

Add support to user-defined profiles being loaded from `ConfigMap` objects. The unstructured nature of this object type removes the potential tight coupling with OCI and downstream components, and the impact in case the seccomp profile format was to change in the future.

Users will be able to create profiles in `ConfigMap` objects and refer to them as `configmap/<profile-name>/<file-name>`. Note that the `ConfigMap` objects will have to be placed in the same namespace as where the Pods will reside. Reusable cross namespaces user-defined profiles will not be supported at this point.

The profile definition would be passed to the CRI as a serialised json object inside a `dockerOpt` object, in the same way that it is done currently for file based profiles, removing the need of files being saved and synchronised across nodes.

```
jsonSeccomp, _ := json.Marshal(profile.Spec)
return []dockerOpt{{"seccomp", string(jsonSeccomp), seccompProfileName}}, nil
```

The new config profiles would be mapped to an additional Kubernetes profile type, which is contingent on the GA API changes proposed in [#1148](https://github.com/kubernetes/enhancements/pull/1148):

```golang
const (
    SeccompProfileUnconfined SeccompProfileType = "Unconfined"
    SeccompProfileRuntime    SeccompProfileType = "Runtime"
    SeccompProfileLocalhost  SeccompProfileType = "Localhost"
    SeccompProfileConfigmap  SeccompProfileType = "ConfigMap"
)
```

User-defined Seccomp profiles would be created this way:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  namespace: secure-projects-ns
  name: webapi-seccomp
data:
  profile-block.json: |-
    { "defaultAction": "SCMP_ACT_ERRNO", ... }
  profile-complain.json: |-
    { "defaultAction": "SCMP_ACT_LOG", ... }
``` 

The two profiles inside the `ConfigMap` above would be referenced respectively by:

- `configmap/webapi-seccomp/profile-block.json`
- `configmap/webapi-seccomp/profile-complain.json`

The only restriction for using those profiles is that the `webapi-seccomp` object must reside in the same namespace as the pods/containers that refers to it, in this case `secure-projects-ns`. 


### Usage Scenarios

#### Profile tampering protection
Just before the container creation time the sha256 hash of the profile contents will be taken and saved within the `pod.status.containerStatuses.SecurityContext` object. 
This enables users to verify whether the applied profile is the exact same as the profile saved on the given `ConfigMap` object. 

This will require changes to `ContainerStatus`:

 ```golang
 // ContainerStatus represents the status of a container.
type ContainerStatus struct {
	...
	// Status of the container.
	SecurityContext SecurityContextStatus
	...
}

 // SecurityContextStatus represents the security context status of a container.
type SecurityContextStatus struct {
	// SeccompProfileHash of the SeccompProfile applied.
	SeccompProfileHash string
}
 ```

A similar approach would also be beneficial to AppArmor profiles. 


#### Rollout of profile changes

Seccomp profiles are applied at container creation time, therefore updating an existing user-defined profile will not cause any changes to the running containers that are using it. 
They will need to be restarted in order for the changes to take effect, which users will have to do manually. 

However, the recommended approach for making profile changes is to create a new `ConfigMap` with the desired profile and then trigger a rolling-update, by changing the configmap the pods are pointing to. The old `ConfigMap` can be deleted once there are no running pods referring to it.


#### Starting containers with non-existent profile

When the Kubelet is creating a container using this seccomp profile type, it must validate that the profile is valid. For a profile reference of `configmap/webapi-seccomp/profile-block.json`, the Kubelet will check for the existence of a `ConfigMap` named `web-seccomp` in the same namespace as the pod/container referencing it. It will also check whether that `ConfigMap` contains a file named `profile-block.json` and that it is a valid json. If any of those evaluations are false, the container will not be created and an error should be logged instead. 

Once the container is running, the `ConfigMap` holding the profile will not be re-evaluated. If that `ConfigMap` is deleted at this point, nothing will happen until the container is restarted, in which case the same rules as above will apply, leading to the container failing to start.

The Kubelet will only validate whether the contents of the profile file is a valid `json` file. The container runtime will be responsible for validating it further. The Kubelet will not validate or use any other files apart from the one being referenced.


### User Stories

#### Story 1
As a user, I want to be able to create and maintain custom seccomp profiles without having to physically copy files across all my existing (and new) nodes.


#### Story 2
As a user, I want to be able to create and maintain custom seccomp profiles by using kubectl, in the same way I do other Kubernetes objects. 


#### Story 3
As a user, I want to be able to determine whether custom seccomp profiles have been changed since applied to running containers. 



## Design Details

### No persistence to disk

The profiles will not be persisted to disk. The Kubelet will fetch the contents from the `ConfigMap` which will later be passed onto the CRI, as per code below:

```golang
scmpWithouPrefix := strings.TrimPrefix(seccompProfile, "configmap/")
configAndFile := strings.Split(scmpWithouPrefix, "/")
...
seccompCfg, err := apiclient.GetConfigMapWithRetry(client, podNamespace, configAndFile[0])
...
profileData, ok := seccompCfg.Data[configAndFile[1]]
if !ok {
  ...
}
if !isValidJson(profileData){
  ...
}

return []dockerOpt{{"seccomp", profileData, ""}}, nil
```

### Feature Gate

A new feature gate will be created to enable the use of Seccomp ConfigMaps, as per below:

`--feature-gate=SeccompConfigMaps=true`

The proposed default value for this feature gate will change through the [Graduation Process](#graduation-criteria).

### Test Plan

Seccomp already has E2E tests, but the tests are guarded by the [Feature:Seccomp] tag and not run in the standard test suites.

Prior to [being marked GA](20190717-seccomp-ga.md), the feature tag will be removed from the seccomp tests.

Unit tests will be created for both kubelet and API server to account for configmap profiles, taking into account scenarios mentioned at the [usage scenarios](#usage-scenarios) section.

E2E tests will be required to ensure profile changes won't impact running containers.

### Upgrade / Downgrade Strategy

No upgrade changes required - both localhost and configmap profiles will be supported in parallel.

### Version Skew Strategy

Error handling for users trying to use the new `ConfigMap` based profiles on older versions of either Kubelet or API server is already implemented. Messages will differ depending on what component does not support the new feature:

- **Not Supported at API Server** (i.e. before implementation of this proposal):
Pod creation will fail with the error message `must be a valid seccomp profile`. 

- **Supported at API Server but not at Kubelet** (i.e. Kubelet running on older version):
Pod creation will fail with the error message `unknown seccomp profile option: configmap/config-name`. 

The error message above may change once seccomp [becomes GA](20190717-seccomp-ga.md).

### Graduation Criteria


##### Alpha

- API changes to map Built-in profiles to an additional `Kubernetes` type.
- Feature gate is disabled by default (i.e. `--feature-gate=SeccompConfigMaps=false`).
- Provide alpha level documentation.
- Unit tests.

##### Alpha -> Beta Graduation

- Feature gate is enabled by default (i.e. `--feature-gate=SeccompConfigMaps=true`).
- Approach community for feedback.
- Provide beta level documentation.
- E2E tests.

##### Beta -> GA Graduation

- Provide comprehensive documentation.
- Feature gate is removed.
- Potential changes based on community feedback.


## Implementation History
- 2019-10-02: Initial KEP
- 2019-10-15: Minor changes
- 2019-10-17: Add alternative to use CRD instead of ConfigMap
- 2019-10-27: Add improvements to Design, Version Skew Strategy and Graduation
- 2019-10-29: Add improvements to alternatives and multiple files support
- 2020-04-24: Fix grammar

## Alternatives

### Use CRD instead of ConfigMap

The decision to use `ConfigMap` was to avoid unnecessary complexity. However, this can be later reviewed through the Graduation process. The reasons considered whilst assessing the two different approaches were:

#### Reasons for `ConfigMap`

- Smaller implementation footprint.
- Official type for storing files.
- Allows for multiple files to be stored on a single object.
- Due to its free form nature, provides loose coupling with downstream dependencies (i.e. OCI).

#### Reasons against it

- The configmap won't be mounted into a container, so it won't be reusing any Kubelet code.
- A CRD could be cluster-scoped (although I'm on the fence about whether that's actually desirable).
- A dedicated type could be immutable, to avoid some uncertainty around updates.
- A fully-structured CRD could provide static type checking for early failure (although this creates the undesirable tight coupling to the OCI format).



#### Start the deprecation process for `localhost/<path>`

The proposed changes will better support custom profiles. Starting the deprecation process of the existing method would signal users what the end goal is. However, this can be started once the new approach GA's.


#### Downstream seccomp support awareness

Validation could be added to assert whether the Seccomp Profile could be applied by the downstream dependencies on a _per- node_ basis, and lead to a list of available profiles for each node. This would benefit clusters with heterogeneus nodes. It would also benefit the usage of the current `localhost/<path>` profile, which an administrator has no way to tell which nodes have them and which ones don't. 

This can be treated as an incremental enhancement in the future, based on the community feedback.


## References
- [Original Seccomp Proposal](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/seccomp.md)
- [Seccomp GA KEP](https://github.com/kubernetes/enhancements/pull/1148)
- [Seccomp enabled by default](https://github.com/kubernetes/enhancements/pull/1257)
