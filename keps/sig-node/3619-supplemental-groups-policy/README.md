# KEP-3619: Fine-grained SupplementalGroups control

<!--
Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
  - [The issue](#the-issue)
  - [Steps to reproduce](#steps-to-reproduce)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Kubernetes API](#kubernetes-api)
    - [SupplementalGroupsPolicy in PodSecurityContext](#supplementalgroupspolicy-in-podsecuritycontext)
    - [User in ContainerStatus](#user-in-containerstatus)
    - [NodeFeatures in NodeStatus which contains SupplementalGroupsPolicy field](#nodefeatures-in-nodestatus-which-contains-supplementalgroupspolicy-field)
  - [CRI](#cri)
    - [SupplementalGroupsPolicy in SecurityContext](#supplementalgroupspolicy-in-securitycontext)
    - [user in ContainerStatus](#user-in-containerstatus-1)
    - [features in StatusResponse which contains supplemental_groups_policy field](#features-in-statusresponse-which-contains-supplemental_groups_policy-field)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: Deploy a Security Policy to enforce <code>SupplementalGroupsPolicy</code> field](#story-1-deploy-a-security-policy-to-enforce-supplementalgroupspolicy-field)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Kubernetes API](#kubernetes-api-1)
    - [SupplementalGroupsPolicy in PodSecurityContext](#supplementalgroupspolicy-in-podsecuritycontext-1)
    - [User in ContainerStatus](#user-in-containerstatus-2)
    - [NodeFeatures in NodeStatus which contains SupplementalGroupsPolicy field](#nodefeatures-in-nodestatus-which-contains-supplementalgroupspolicy-field-1)
  - [CRI](#cri-1)
    - [SupplementalGroupsPolicy in SecurityContext](#supplementalgroupspolicy-in-securitycontext-1)
    - [user in ContainerStatus](#user-in-containerstatus-3)
    - [features in StatusResponse which contains supplemental_groups_policy field](#features-in-statusresponse-which-contains-supplemental_groups_policy-field-1)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Introducing <code>RutimeClass</code>](#introducing-rutimeclass)
  - [Adjusting container image by users](#adjusting-container-image-by-users)
  - [Just fixing CRI implementations](#just-fixing-cri-implementations)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The KEP seeks to provide a way to choose correct behavior with how Container Runtimes (Containerd and CRI-O) are applying `SupplementalGroups` to the first container processes. The KEP describes the work needed to be done in Kubernetes or connected projects to make sure customers have a clear migration path - including detection and safe upgrade - if any of their workflows took a dependency on this arguably erroneous behavior.

### The issue

How supplemental groups attached to the container processes are defined in two levels in Kubernetes, one is OCI image level and the other is Kubernetes API level.

In [OCI image spec](https://github.com/opencontainers/image-spec), [`config.User` OCI image configuration](https://github.com/opencontainers/image-spec/blob/3a7f492d3f1bcada656a7d8c08f3f9bbd05e7406/config.md#:~:text=User%20string%2C%20OPTIONAL)(mirrored spec of [`USER` directive in `Dockerfile`](https://docs.docker.com/engine/reference/builder/#user)) is defined as follows:

> The username or UID which is a platform-specific structure that allows specific control over which user the process run as. This acts as a default value to use when the value is not specified when creating a container. For Linux based systems, all of the following are valid: `user`, `uid`, `user:group`, `uid:gid`, `uid:group`, `user:gid`. If `group`/`gid` is not specified, the default group and supplementary groups of the given `user`/`uid` in `/etc/passwd` from the container are applied.

In Kubernetes API level, `PodSecurityContext.{RunAsUser, RunAsGroup, SupplementalGroups}` relates to this. This API was designed to override `config.User` configuration of OCI images. However, in the current implementation, as described in [kubernetes/kubernetes#112879](https://github.com/kubernetes/kubernetes/issues/112879), even when a manifest defines both `RunAsGroup`, __group memberships defined in the container image for the UID__ are attached to the container process (see the [the next section](#steps-to-reproduce) for details). This behavior clearly diverges from the specification of OCI image configuration, especially the next sentence of [`config.User` OCI image configuration](https://github.com/opencontainers/image-spec/blob/3a7f492d3f1bcada656a7d8c08f3f9bbd05e7406/config.md#:~:text=User%20string%2C%20OPTIONAL)):

> If `group`/`gid` is not specified, the default group and supplementary groups of the given `user`/`uid` in `/etc/passwd` from the container are applied.

As described in [kubernetes/kubernetes#112879](https://github.com/kubernetes/kubernetes/issues/112879), the behavior is not documented well and is not widely known by most Kubernetes administrators and users. Moreover, this behavior causes security considerations in some cases.

### Steps to reproduce

Assume you have an image and a Pod manifest:

```Dockerfile
# Dockerfile
FROM ubuntu:22.04
# This generates /etc/group entry --> "group-in-image:x:50000:alice"
RUN groupadd -g 50000 group-in-image \
    && useradd -m -u 1000 alice \
    && gpasswd -a alice group-in-image
USER alice
```

```yaml
spec:
  # This overrides 
  # - USER directive in Dockerfile above by runAsUser and runAsGroup with "1000:1000", and 
  # - setting supplementalGroups
  # This spec expects NOT to attach gids defined in the image(/etc/group) to the container process
  # because this specifies gid by runAsGroup explicitly.
  securityContext: { runAsUser:1000, runAsGroup:1000, supplementalGroups:[60000]}
  containers:
    # Expected output: "uid=1000(alice) gid=1000(alice) groups=1000(alice),60000"
    # NOTE: "group-in-image" is not included here 
    #       because groups defined in /etc/group should not be attached
    #       when gids is specified in runAsGroup
  - image: the-image-above
    sh: ["id"] 
```

However, the current combination with Kubernetes and major container runtimes(at least containerd and cri-o) outputs(See [here](https://github.com/pfnet-research/strict-supplementalgroups-container-runtime/tree/reproduce-bypass-supplementalgroups) for more detailed reproduction code) includes "group-in-image" group of the first container process.

```console
uid=1000(alice) gid=1000(alice) groups=1000(alice),50000(group-in-image),60000
```

## Motivation

As described above, how supplemental groups attached to the first container process is complicated and not OCI image spec compliant.

Moreover, this causes security considerations as follows. When a cluster enforces some security policy for pods that protects the value of `RunAsGroup` and `SupplementalGroups`, the effect of its enforcement is limited, i.e., cluster users can easily bypass the policy enforcement just by using a custom image. If such a bypass happened, it would be unexpected behavior for most cluster administrators because the enforcement is almost useless. Moreover, the bypass will cause unexpected file access permission. In some use cases, the unexpected file access permission will be a security concern. For example, using `hostPath` volumes could be a severe problem because UID/GIDs matter in accessing files/directories in the volumes.

Kubernetes provides no API surface to prevent this bypass although it could sometimes lead to a security concern. Because the behavior is implemented in CRI implementations actually, To mitigate this, the cluster administrators will need to deploy a custom low-level container runtime(e.g., [pfnet-research/strict-supplementalgroups-container-runtime](https://github.com/pfnet-research/strict-supplementalgroups-container-runtime)) that modifies OCI container runtime spec(`config.json`) produced by CRI implementations (e.g., containerd, cri-o). A custom `RuntimeClass` would be introduced for it. Nevertheless, It would be an extra operational burden for cluster administrators.

Thus, this KEP proposes to offer a new API field named `SupplementalGroupsPolicy` that enables users to control supplemental groups attached to the first container process by following "principle of least surprise". The new API allows cluster administrators to deploy security policies that protect the `SupplementalGroupsPolicy` field in the cluster to avoid the unexpected bypass of `SupplementalGroups` described above. This KEP also proposes a way for users to detect which groups are _actually_ attached to container processes. This helps users/administrators identify which pods have _unexpected_ group permissions and choose the best `SupplementalGroupsPolicy` for them.

### Goals

- To Provide a new API field to control exactly which groups the container process belongs to
- Ensure there are clear steps documented for end users to detect if their workload is affected 
- (Optional) provide helper APIs and/or tooling to simplify the detection

### Non-Goals

- To provide a cluster-wide control method.
- To change the default behavior (a potentially breaking change)

## Proposal

This KEP proposes changes both on Kubernets API and CRI levels.

### Kubernetes API

_See also [Alternatives](#alternatives) section for rejected alternative plans._

#### SupplementalGroupsPolicy in PodSecurityContext

A new field named `SupplementalGroupsPolicy` will be introduced to `PodSecurityContext`. This field defines how supplemental groups of the first container process are calculated.

Allowed values are:

- `Merge`(_default if not specified_): This policy _always_ merges the provided `SupplementalGroups`(including `FsGroup`) with groups of the primary user from the image(`/etc/group` in the image).
  - Note: The primary user is specified with `RunAsUser`. If not specified, the user from the image config is used. Otherwise, the runtime default is used.
- `Strict`: This policy uses _only_ the provided `SupplementalGroups`(including `FsGroup`) as supplemental groups for the first container process. No groups from the image are extracted.

Note that both policies diverge from the semantics of [`config.User` OCI image configuration](https://github.com/opencontainers/image-spec/blob/3a7f492d3f1bcada656a7d8c08f3f9bbd05e7406/config.md#:~:text=User%20string%2C%20OPTIONAL). The purpose is to follow "principle of least surprise" as described in the previous section.

#### User in ContainerStatus

To provide users/administrators to know which identities are actually attached to the container process, it proposes to introduce new `User` field in `ContainerStatus`. `User` is an object which consists of `Uid`, `Gid`, `SupplementalGroups` fields for linux containers. This will help users to identify unexpected identities. This field is derived by CRI response (See [user in ContainerStatus](#user-in-containerstatus-1) section).

#### NodeFeatures in NodeStatus which contains SupplementalGroupsPolicy field

Because the actual control(calculation) of supplementary groups to be attached to the first container process will happen inside of CRI implementations (container runtimes), it proposes to add `NodeFeatures` field in `NodeStatus` which contains the `SupplementalGroupsPolicy` feature field inside of it like below so that kubernetes can correctly understand whether underlying CRI implementation implements the feature or not. The field is populated by CRI response.

```golang
type NodeStatus struct {
	// Features describes the set of features implemented by the CRI implementation.
	Features *NodeFeatures
}
type NodeFeatures struct {
	// SupplementalGroupsPolicy is set to true if the runtime supports SupplementalGroupsPolicy and ContainerUser.
	SupplementalGroupsPolicy *bool
}
```

Recently [KEP-3857: Recursive Read-only (RRO) mounts](https://kep.k8s.io/3857) introduced `RuntimeHandlers[].Features`. But it is not fit to use for this KEP because RRO mounts requires inspecting [the OCI runtime spec's Feature](https://github.com/opencontainers/runtime-spec/blob/main/features.md) to understand whether the low-level OCI runtime supports RRO or not. However, for this KEP(SupplementalGroupsPolicy), it does not need to inspect [the OCI runtime spec's Feature](https://github.com/opencontainers/runtime-spec/blob/main/features.md) because this KEP only affects  [`Process.User.additionalGid`](https://github.com/opencontainers/runtime-spec/blob/main/config.md#user) and does not depend on [the OCI runtime spec's Feature](https://github.com/opencontainers/runtime-spec/blob/main/features.md). So, introducing new `NodeFeatures` in `NodeStatus` does not conflict with `RuntimeHandlerFeatures` as we can clearly define how to use them as below:

- `NodeFeatures`(added in this KEP):
  - focusses on features that depend only on cri implementation, be independent of runtime handlers(low-level container runtimes), (i.e. it should not require to inspect to any information from oci runtime-spec's features).
- `RuntimeHandlerFeature` (introduced in KEP-3857):
  -  focuses features that depend on the runtime handlers, (i.e. dependent to the information exposed by oci runtime-spec's features).

See [this section](#runtimefeatures-in-nodestatus-which-contains-supplementalgroupspolicy-field-1) for details.

### CRI

#### SupplementalGroupsPolicy in SecurityContext

Symmetrical changes are needed. See [Design Details](#design-details) section.

#### user in ContainerStatus

To propagate identities of the container process to `ContainerStatus` in Kubernetes API, CRI changes would be needed. This proposes to define `ContainerUser` data type and add `user` field to `ContainerStatus` that is used in the response of `ContainerStatus` method. `ContainerUser` consists of `Uid`, `Gid` and `SupplementalGroups` fields.

```protobuf
// service RuntimeService {
//   rpc ContainerStatus(ContainerStatusRequest) returns (ContainerStatusResponse) {}
//  ...
// }
// message ContainerStatusResponse {
//  ContainerStatus status = 1;
//  ...
// }

message ContainerStatus {
  ...
  // user information of the container process
  ContainerUser user = ?;
}

message ContainerUser {
  // details in "Design Details" section
}
```

#### features in StatusResponse which contains supplemental_groups_policy field

To propagate whether the runtime supports fine-grained supplemental group control to `NodeFeatures.SupplementalGroupsPolicy`, it proposes to add a corresponding field `features` in `StatusResponse`. 

```proto
// service RuntimeService {
// ...
//     rpc Status(StatusRequest) returns (StatusResponse) {}
// }
message StatusResponse {
...
    // features describes the set of features implemented by the CRI implementation.
    // This field is supposed to propagate to NodeFeatures in Kubernetes API.
    RuntimeFeatures features = ?;
}
message RuntimeFeatures {
    // supplemental_groups_policy is set to true if the runtime supports SupplementalGroupsPolicy and ContainerUser.
    bool supplemental_groups_policy = 1;
}
```

As discussed in [Kubernetes API section](#runtimefeatures-in-nodestatus-which-contains-supplementalgroupspolicy-field), `RuntimeHandlerFeature` introduced in [KEP-3857](https://kep.k8s.io/3857) should focus on features only for ones which requires to inspect [OCI runtime spec's Feature](https://github.com/opencontainers/runtime-spec/blob/main/features.md). But `RuntimeFeatuers` proposed in this KEP should focus on ones which does NOT require to inepect it.


### User Stories (Optional)

#### Story 1: Deploy a Security Policy to enforce `SupplementalGroupsPolicy` field

Assume a multi-tenant kubernetes cluster with `hostPath` volumes below situations:

- Multi-tenant model is namespace-based (namespace per tenant(user/group) model)
  - access to each namespace is controlled by RBAC
- PSP(or other policy engines) is enforced in each namespace which protects
  - `runAsUser`, `runAsGroup`, `fsGroup`, `supplementalGroups` values
- A `hostPath` volume (say `/mnt/hostpath`) is maintained in all the nodes by administrators
  - with permission `drwxr-xr-x nobody nogroup /mnt/hostpath`
  - the directory mounts an NFS volume shared by all the tenants, and UIDs/GIDs are managed by the cluster admininistrators
  - Any tenant CAN create a directory under this directory
- There is a `/mnt/hostpath/private-to-gid-60000` which is fully private to `gid=60000`
  -  i.e. its permission is `drwxrwx--- nobody 60000 /mnt/hostpath/private-to-gid-60000` 
- There is `user-alice` namespace for `alice(uid=1000)`, and `alice` only belongs a `group-a(gid=50000)`
- cluster administrator enforces a policy for Pods with `/mnt/hostpath` `hostPath` volumes in `user-alice` namespace such that
  - `runAsUser, runAsGroup` must be `1000`
  - `supplementalGroups` must be `[60000]`
  - `fsGroup` must be one of `1000, 60000`
  - i.e. cluster administrator expects that all the container processes can only have `60000` as supplementary groups in `user-alice` namespace

As described in [Summary](#summary) section, `alice` can bypass the restriction by using a custom image. To mitigate the scenario, cluster administrators can deploy a security policy restricting `supplementalGroupsPolicy` in `user-alice` namespace such that:
  - `runAsUser, runAsGroup` must be `1000`
  - `supplementalGroups` must be `[60000]`
    - _this is not enough to avoid bypassing supplementary groups for container processes_
  - __`supplementalGroupsPolicy` must be `Strict`__
    - __this really needs to avoid the bypass completely__
  - `fsGroup` must be one of `1000, 60000`

Please note that a security policy without `supplementalGroupsPolicy` would lead to unexpected groups for the first process in the containers.

<!-- #### Story 2 -->

### Notes/Constraints/Caveats (Optional)

The proposal affects to the CRI implementations (e.g., containerd, cri-o, gVisor, etc.)

### Risks and Mitigations

- How to track the support status in CRI implementations of this proposal?
  - This feature is mainly implemented inside each CRI implementation.
- How to feature-gate this feature in CRI implementations?

## Design Details

### Kubernetes API

#### SupplementalGroupsPolicy in PodSecurityContext

A new field named `SupplementalGroupsPolicy` will be introduced to `PodSecurityContext`:

```go
type PodSecurityContext struct {
	...
	// A list of groups applied to the first process run in each container. 
	// supplementalGroupsPolicy can control how groups will be calculated.
	// Note that this field cannot be set when spec.os.name is windows.
	// +optional
	SupplementalGroups []int64
	// supplementalGroupsPolicy defines how supplemental groups of the first 
	// container processes are calculated.
	// Valid values are "Merge" and "Strict". 
	// If note specified, "Merge" is used.
	// Note that this field cannot be set when spec.os.name is windows.
	// +optional
	SupplementalGroupsPolicy *PodSecurityGroupsPolicy 
}

type PodSecurityGroupsPolicy string
const (
	// SecurityGroupsPolicyMerge policy always merges 
	// the provided SupplementalGroups (including FsGroup) 
	// with groups of the primary user from the container image(`/etc/group`).
	// Note: The primary user is specified with RunAsUser. 
	//       If not specified, the user from the image config is used. 
	//       Otherwise, the runtime default is used.
	SecurityGroupsPolicyMerge PodSecurityGroupsPolicy = "Merge"

	// SecurityGroupsPolicyStrict policy uses only 
	// the provided SupplementalGroups(including FsGroup) 
	// as supplemental groups for the first container process. 
	// No groups extracted from the container image.
	SecurityGroupsPolicyStrict PodSecurityGroupsPolicy = "Strict"
)
```

#### User in ContainerStatus

```golang
type ContainerStatus struct {
...
	// User indicates identities of the container process
	User ContainerUser
}
```

```golang
type ContainerUser struct {
	// Linux holds identity information of the process of the containers in Linux.
	// Note that this field cannot be set when spec.os.name is windows.
	Linux *LinuxContainerUser

	// Windows holds identity information of the process of the containers in Windows
	// This is just reserved for future use.
	// Windows *WindowsContainerUser
}

type LinuxContainerUser struct {
	// Uid is the primary uid of the container process
	Uid int64
	// Gid is the primary gid of the container process
	Gid int64
	// SupplementalGroups are the supplemental groups attached to the container process
	SupplementalGroups []int64
}

// This is just reserved for future use.
// type WindowsContainerUser struct {
// 	T.B.D.
// }
```

#### NodeFeatures in NodeStatus which contains SupplementalGroupsPolicy field

```golang
type NodeStatus struct {
	// Features describes the set of implemented features implemented by the CRI implementation.
	// +featureGate=SupplementalGroupsPolicy
	// +optional
	Features *NodeFeatures

	// The available runtime handlers.
	// +featureGate=RecursiveReadOnlyMounts
	// +optional
	RuntimeHandlers []RuntimeHandlers
}

// NodeFeatures describes the set of implemented features implemented by the CRI implementation.
// THE FEATURES CONTAINED IN THE NodeFeatures SHOULD DEPEND ON ONLY CRI IMPLEMENTATION, BE INDEPENDENT ON RUNTIME HANDLERS,
// (I.E. IT SHOULD NOT REQUIRE TO INSPECT TO ANY INFORMATION FROM OCI RUNTIME-SPEC'S FEATURES).
type NodeFeatures {
	// SupplementalGroupsPolicy is set to true if the runtime supports SupplementalGroupsPolicy and ContainerUser.
	// +optional
	SupplementalGroupsPolicy *bool
}

// NodeRuntimeHandler is a set of runtime handler information.
type NodeRuntimeHandler struct {
	// Runtime handler name.
	// Empty for the default runtime handler.
	// +optional
	Name string
	// Supported features in the runtime handlers.
	// +optional
	Features *NodeRuntimeHandlerFeatures
}

// NodeRuntimeHandlerFeatures is a set of features implementedy by the runtime handler.
// THE FEATURES CONTAINED IN THE NodeRuntimeHandlerFeatures SHOULD DEPEND ON THE RUNTIME HANDLERS,
// (I.E. DEPENDENT TO THE INFORMATION EXPOSED BY OCI RUNTIME-SPEC'S FEATURES).
type NodeRuntimeHandlerFeatures struct {
	// RecursiveReadOnlyMounts is set to true if the runtime handler supports RecursiveReadOnlyMounts.
	// +featureGate=RecursiveReadOnlyMounts
	// +optional
	RecursiveReadOnlyMounts *bool
	// Reserved: UserNamespaces *bool
}
```

### CRI

#### SupplementalGroupsPolicy in SecurityContext

cri-spec (`v1`) also needs to be updated similarly as follows. Comments are omitted because they are symmetric to Pods' one.

```proto
enum SupplementalGroupsPolicy {
    Merge = 0;
    Strict = 1;
}

message LinuxContainerSecurityContext {
...
    repeated int64 supplemental_groups;
    optional SupplementalGroupsPolicy supplemental_groups_policy;
}

message LinuxSandboxSecurityContext {
...
    repeated int64 supplemental_groups;
    optional SupplementalGroupsPolicy supplemental_groups_policy;
}
```

#### user in ContainerStatus

```protobuf

message ContainerStatus {
    ...
    // User holds user information of the container process
    ContainerUser user = ??;
}

message ContainerUser {
    // User information of Linux containers.
    LinuxContainerUser linux = 1;
    // User information of Windows containers.
    // This is just reserved for future use.
    // WindowsContainerUser windows = 2;
}


message LinuxContainerUser {
    // uid is the primary uid of the container process
    Int64Value uid = 1;
    // gid is the primary gid of the container process
    Int64Value gid = 2;
    // supplemental_groups are the supplemental groups attached to the container process
    repeated int64 supplemental_groups = 3;
}

// message WindowsContainerUser {
//     T.B.D.
// }
```

#### features in StatusResponse which contains supplemental_groups_policy field

```proto
// service RuntimeService {
// ...
//     rpc Status(StatusRequest) returns (StatusResponse) {}
// }
message StatusResponse {
...
    // Runtime handlers.
    repeated RuntimeHandler runtime_handlers = 3;

    // features describes the set of features implemented by the CRI implementation.
    // This field is supposed to propagate to NodeFeatures in Kubernetes API.
    RuntimeFeatures features = ?;
}

// RuntimeFeatures describes the set of features implemented by the CRI implementation.
// THE FEATURES CONTAINED IN THE RuntimeFeatures SHOULD DEPEND ON ONLY CRI IMPLEMENTATION, BE INDEPENDENT ON RUNTIME HANDLERS,
// (I.E. IT SHOULD NOT REQUIRE TO INSPECT TO ANY INFORMATION FROM OCI RUNTIME-SPEC'S FEATURES).
message RuntimeFeatures {
    // supplemental_groups_policy is set to true if the runtime supports SupplementalGroupsPolicy and ContainerUser.
    bool supplemental_groups_policy = 1;
}

// message RuntimeHandler {
//     // Name must be unique in StatusResponse.
//     // An empty string denotes the default handler.
//     string name = 1;
//     // Supported features.
//     RuntimeHandlerFeatures features = 2;
// }

// RuntimeHandlerFeatures is a set of features implementedy by the runtime handler.
// THE FEATURES CONTAINED IN THE RuntimeHandlerFeatures SHOULD DEPEND ON THE RUNTIME HANDLERS,
// (I.E. DEPENDENT TO THE INFORMATION EXPOSED BY OCI RUNTIME-SPEC'S FEATURES).
message RuntimeHandlerFeatures {
    bool recursive_read_only_mounts = 1;
    bool user_namespaces = 2;
}
```

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- `k8s.io/kubernetes/pkg/api/pod/util.go`: `2024-08-13` - `68.7%`
  - It tests `dropDisabledFields` for `PodSecurityContext.SupplementalGroups`, `ContainerStatus.User` fields
  - Note: The test these field values when enabling/disabling this feature.

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

See [e2e tests](#e2e-tests) below.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- Kubernetes: <https://github.com/kubernetes/kubernetes/blob/v1.31.0/test/e2e/node/security_context.go>
  - When creating a Pod with `SupplementalGroupsPolicy=Strict`
    - the containers in the pod will run with only groups specified by the API, and
    - once it starts, `ContainerStatus.User` contains the correct identities of the containers
  - When creating a Pod with `SupplementalGroupsPolicy=Merge`
    - the containers in the pod will run with groups specified by API and groups from the container image, and
    - once it starts, `ContainerStatus.User` contains the correct identities of the containers, and
  - When creating a Pod without `SupplementalGroupsPolicy` (equivalent behaviour with `Merge`)
    - the pod will run with with groups specified by API and groups from the image
    - once it starts, `ContainerStatus.User` contains the correct identities of the containers
  - _Note: above e2e tests will self-skip if the node does not support `SupplementalGroupsPolicyFeature` detected by `Node.Status.Featuers.SupplementalGroupsPolicy` field._
- critools(critest): <https://github.com/kubernetes-sigs/cri-tools/blob/v1.31.0/pkg/validate/security_context_linux.go>
  - Symmetric test cases with Kubernetes e2e tests except for the case of _without `SupplementalGroupsPolicy`_ because `SupplementalGroupsPolicy` always has value(default is `Merge`).
  - _Note: above tests will self-skip if the runtime does not support `SupplementalGroupsPolicyFeature` detected by `StatusResponse.features.supplemental_groups_policy` field._


### Graduation Criteria

Because this KEP's core implementation(i.e. `SupplementalGroupsPolicy` handling) lies inside of CRI implementations(e.g. containerd, cri-o), the graduation criteria contains the support statuses of the updated CRI by container runtimes.

#### Alpha

- At least one of the most popular Container Runtimes(e.g. containerd) implements the updated CRI and released
- Feature implemented behind a feature flag based on the Container Runtime
- Unit tests and initial e2e tests completed and enabled

#### Beta

- Several popular Container Runtimes(e.g. containerd and cri-o) support the updated CRI and released
- Fixed reported bugs from the community
- Additional integration tests and e2e tests are in Testgrid and linked in KEP

#### GA

- At least one of Container Runtimes which is not based on the classic container, gVisor for example, supports the updated CRI and released
- Assuming no negative user feedback based on production experience, promote after 2 releases in beta.
- [conformance tests] are added for `SupplementalGroupsPolicy` and `ContainerStatus.User` APIs

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

### Upgrade / Downgrade Strategy

### Version Skew Strategy

- CRI must support this feature, especially when using `SupplementalGroupsPolicy=Strict`.
- kubelet must be at least the version of control-plane components.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: SupplementalGroupsPolicy
  - Components depending on the feature gate: kube-apiserver, kubelet, (and CRI implementations(e.g. containerd, cri-o))
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No. Just introducing new API fields in Pod spec and CRI which does NOT change the default behavior.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes. It can be disabled after enabled. However, users should pay attention that gids of container processes in pods with `Strict` policy would change. It means the action might break the application in permission. We plan to provide a way for users to detect which pods are affected.

###### What happens if we reenable the feature if it was previously rolled back?

Just the policy `Stcict` is reenabled. Users should pay attention that gids of containers in pods with `Stcict` policy would change. It means that the action might break the application in permission. We plan to provide a way for users to detect which pods are affected.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

Yes, see [Unit tests](#unit-tests) section.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

A rollout may fail when at least one of the following components are too old because this KEP introduces the new Kubernetes API field:

| Component      | `supplementalGroupsPolicy` value that will cause an error |
|----------------|-----------------------------------------------------------|
| kube-apiserver | not null                                                  |
| kubelet        | not null                                                  |
| CRI runtime    | `Strict`                                                  |


For example, an error will be returned like this if kube-apiserver is too old:
```console
$ kubectl apply -f supplementalgroupspolicy.yaml
Error from server (BadRequest): error when creating "supplementalgroupspolicy.yaml": Pod in version "v1" cannot be handled as a Pod:
strict decoding error: unknown field "spec.securityContext.supplementalGroupsPolicy"
```

No impact on already running workloads.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

Look for an event saying indicating SupplementalGroupsPolicy is not supported by the runtime.
```console
$ kubectl get events -o json -w
...
{
    ...
    "kind": "Event",
    "message": "Error: SupplementalGroupsPolicyNotSupported",
    ...
}
...
```

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

During the beta phase, the following test will be manually performed:
- Enable the `SupplementalGroupsPolicy` feature gate for kube-apiserver and kubelet.
- Create a pod with `supplementalGroupsPolicy` specified.
- Disable the `SupplementalGroupsPolicy` feature gate for kube-apiserver, and confirm that the pod gets rejected.
- Enable the `SupplementalGroupsPolicy` feature gate again, and confirm that the pod gets scheduled again.
- Do the same for kubelet too.


###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

Inspect the `supplementalGroupsPolicy` fields in Pods. You can check if the following `jq` command prints non-zero number:

```bash
kubectl get pods -A -o json | jq '[.items[].spec.securityContext? | select(.supplementalGroupsPolicy)] | length'
```

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [x] API .status
  - Condition name: `containerStatuses.user`
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

- `supplementalGroupsPolicy=Strict`: 100% of pods were scheduled into a node with the feature supported.

- `supplementalGroupsPolicy=Merge`: 100% of pods were scheduled into a node with or without the feature supported.

- `supplementalGroupsPolicy` is unset: 100% of pods were scheduled into a node with or without the feature supported.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [x] Metrics
  - Metric name:
  - [Optional] Aggregation method: `kubectl get events -o json -w`
  - Components exposing the metric: kubelet -> kube-apiserver
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

Potentially, kube-scheduler could be implemented to avoid scheduling a pod with `supplementalGroupsPolicy: Strict`
to a node running CRI runtime which does not supported this feature.

In this way, the Event metric described above would not happen, and users would instead see `Pending` pods
as an error metric.

However, this is not planned to be implemented in kube-scheduler, as it seems overengineering.
Users may use `nodeSelector`, `nodeAffinity`, etc. to workaround this.

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

Container runtimes supporting [CRI api v0.31.0](https://github.com/kubernetes/cri-api/tree/v0.31.0) or above.

For example, 
- containerd: v2.0 or later
- CRI-O: v1.31 or later

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->


A pod with `supplementalGroupsPolicy: Strict` may be rejected by kubelet with the probablility of $$B/A$$,
where $$A$$ is the number of all the nodes that may potentially accept the pod,
and $$B$$ is the number of the nodes that may potentially accept the pod but does not support this feature.
This may affect scalability.

To evaluate this risk, users may run
`kubectl get nodes -o json | jq '[.items[].status.features]'`
to see how many nodes support `supplementalGroupsPolicy: true`.

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

No. Just introducing new API fields in Pod spec and CRI which does NOT change the default behavior.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

Precisely, yes because the kep introduces new API fields in Pods. But the increasing size can be negligible.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

No.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

A pod cannot be created, just as in other pods.

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

None.

###### What steps should be taken if SLOs are not being met to determine the problem?

- Make sure that the node is running with CRI runtime which supports this feature.
- Make sure that `crictl info` (with the latest crictl)
  reports that `supplemental_groups_policy` is supported.
  Otherwise upgrade the CRI runtime, and make sure that no relevant error is printed in
  the CRI runtime's log.
- Make sure that `kubectl get nodes -o json | jq '[.items[].status.features]'`
  (with the latest kubectl and control plane)
  reports that `supplementalGroupsPolicy` is supported.
  Otherwise upgrade the CRI runtime, and make sure that no relevant error is printed in
  kubelet's log.

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

- 2023-02-10: Initial KEP published.
- v1.31.0(2024-08-13): Alpha

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

N/A

## Alternatives

### Introducing `RutimeClass`

As described in the [Motivation](#motivation) section, cluster administrators would need to deploy a custom low-level container runtime(e.g., [pfnet-research/strict-supplementalgroups-container-runtime](https://github.com/pfnet-research/strict-supplementalgroups-container-runtime)) that modifies OCI container runtime spec(`config.json`) produced by CRI implementations (e.g., containerd, cri-o). A custom `RuntimeClass` would be introduced for it.

### Adjusting container image by users

Users could modify their container images to control the supplemental groups (i.e., modifying group memberships of the uid of the container). Although it is more work and users won't always have the option to do that.

### Just fixing CRI implementations

We could just fix CRI implementations directly without introducing new APIs.  The advantage is no API changes both on Kubernetes and CRI levels. However, the main downside of this approach is a breaking change that makes users confused.

## Infrastructure Needed (Optional)

N/A
