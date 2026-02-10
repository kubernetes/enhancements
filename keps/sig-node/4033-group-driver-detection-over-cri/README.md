# [KEP-4033](https://github.com/kubernetes/enhancements/issues/4033): Discover cgroup driver from CRI

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [CRI API](#cri-api)
  - [Kubelet](#kubelet)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [x] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [x] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This enhancement adds the ability for the container runtime to instruct kubelet
which cgroup driver to use. This removes the need for specifying cgroup driver
in the kubelet configuration and eliminates the possibility of misaligned
cgroup driver configuration between the kubelet and the runtime.

## Motivation

The responsibility of managing the Linux cgroups is currently split between the
kubelet and the container runtime. Kubelet takes care of the pod (sandbox)
level cgroups whereas the runtime is responsible for per-container cgroups.
There currently are two different low-level management interfaces for cgroups:
manipulating the cgroupfs directly or using the systemd system management
daemon to manage them. Currently, both the kubelet and the container runtime
has a configuration setting for selecting the cgroup driver (cgroupfs or
systemd). These settings must be in sync, both kubelet and the runtime
configured to use the same driver as the two drivers are incompatible because
of a different kind of cgroups hierarchy used in them. Having kubelet and the
container runtime to use non-matching cgroup drivers can cause hard to
understand failures in container creation or inconsistent resource allocation
on the node. This – two independent configuration settings for the same thing –
is a common cause for user errors. Instead of having a split brain situation,
there should be a single source of truth for the cgroup driver.

### Goals

- make kubelet automatically use the same cgroup driver as the container
  runtime, making it unnecessary to specify cgroup driver in kubelet
  configuration
- maintain backward compatibility to run kubelet on top of an older container
  runtime that doesn’t support this new feature

### Non-Goals

## Proposal

### User Stories (Optional)

#### Story 1

As a cluster administrator, I would like to simplify my node configuration by
configuring fields just once, and not needing to synchronize between multiple
processes.

#### Story 2

As a novice Kubernetes user, I would like to easily be able to start pods with
all runtimes, even if they have differing opinions on defaults.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

Field adoption could be considered a risk, though the CRI implementations work
closely with SIG Node and the feature will move along with CRI implementation
adoption.

## Design Details

### CRI API

Extend the CRI runtime API to inform the kubelet which cgroup driver should be
used. A new RuntimeConfig rpc is added to query the information.

```diff
 // Runtime service defines the public APIs for remote container runtimes
 service RuntimeService {
 ...
+    // RuntimeConfig returns configuration information of the runtime.
+    // A couple of notes:
+    // - The RuntimeConfigRequest object is not to be confused with the contents of UpdateRuntimeConfigRequest.
+    //   The former is for having runtime tell Kubelet what to do, the latter vice versa.
+    // - It is the expectation of the Kubelet that these fields are static for the lifecycle of the Kubelet.
+    //   The Kubelet will not re-request the RuntimeConfiguration after startup, and CRI implementations should
+    //   avoid updating them without a full node reboot.
+    rpc RuntimeConfig(RuntimeConfigRequest) returns (RuntimeConfigResponse) {}
 }
 
+message RuntimeConfigRequest {}
 
+message RuntimeConfigResponse {
+    // Configuration information for Linux-based runtimes. This field contains
+    // global runtime configuration options that are not specific to runtime
+    // handlers.
+    LinuxRuntimeConfiguration linux = 1;
+}
 
+message LinuxRuntimeConfiguration {
+    // Cgroup driver to use
+    // Note: this field should not change for the lifecycle of the Kubelet,
+    // or while there are running containers.
+    // The Kubelet will not re-request this after startup, and will construct the cgroup
+    // hierarchy assuming it is static.
+    // If the runtime wishes to change this value, it must be accompanied by removal of
+    // all pods, and a restart of the Kubelet. The easiest way to do this is with a full node reboot.
+    CgroupDriver cgroup_driver = 1;
+}
 
+enum CgroupDriver {
+    SYSTEMD = 0;
+    CGROUPFS = 1;
+}
```

### Kubelet

Kubelet will be modified to support the new field.

If available the cgroup driver information received from the container runtime
will take precedence over cgroupDriver setting from the kubelet config (or
`--cgroup-driver` command line flag). If the runtime does not provide
information about the cgroup driver, then kubelet will fall back to using its
own configuration (`cgroupDriver` from kubeletConfig or the `--cgroup-driver`
flag). In beta, resorting to the fallback behavior will produce a log message like:

```
cgroupDriver option has been deprecated and will be dropped in a future release. Please upgrade to a CRI implementation that supports cgroup-driver detection.
```

The `--cgroup-driver` flag and the cgroupDriver configuration option will be
deprecated when support for the feature is graduated to GA.
The configurations flags (and the related fallback behavior) will be removed in
Kubernetes 1.37. This aligns well with containerd v1.7 going out of support, which is the last
remaining supported CRI that doesn't have support for this field.
At the point the kubelet refuses to start if the CRI runtime does not support
the feature.

Between version 1.34 and 1.36, the kubelet will emit a counter metric (`cri_losing_support`) when a CRI implementation is
used that doesn't have support for the RuntimeConfig CRI call. This metric will have a label describing the version support will be dropped by.
If one node in a cluster has containerd running with 1.7, the metric will look like `cri_losing_support{,version="1.37"} 1`.

Kubelet startup is modified so that connection to the CRI server (container
runtime) is established and RuntimeConfig is queried before initializing the
kubelet internal container-manager which is responsible for kubelet-side cgroup
management. RuntimeConfig query is expected to
succeed, an error (error response or timeout) is regarded as a failed
initialization of the runtime service and kubelet will exit with an error
message and an error code.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

No prerequisites have been identified.

##### Unit tests

- `k8s.io/kubernetes/pkg/kubelet/kuberuntime`: `2023-06-15` - `66.1%`

Kubelet unit tests that use the
[fake_runtime](https://github.com/kubernetes/cri-api/blob/master/pkg/apis/testing/fake_runtime_service.go)
will be updated to verify the Kubelet is correctly inheriting the cgroup
driver.

##### Integration tests

No new integration tests for kubelet are planned.

##### e2e tests

No new e2e tests for kubelet are planned.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag, fallback to old behavior if flag is enabled but runtime support not present.
- Initial unit tests completed and enabled


#### Beta

- Feature implemented, with the feature gate enabled by default.
- Released versions of CRI-O and containerd runtime implementations support the feature

#### GA

- No bugs reported in the previous cycle.
- Deprecate kubelet cgroupDriver configuration option and `--cgroup-driver` flag.
- Remove feature gate
- All issues and gaps identified as feedback during beta are resolved

### Upgrade / Downgrade Strategy

The fallback behavior will prevent the majority of regressions, as Kubelet will
choose a cgroup driver, same as it used to before this KEP, even when the
feature gate is on.

The feature gate is another layer of protection, requiring admins to specifically opt-into this behavior.

### Version Skew Strategy

If either kubelet or the container runtime running on the node does not support
the new field in the CRI API, they just resort to the existing behavior of
respecting their individual cgroup-driver setting. That is, if the node has a
container runtime that does not support this field the kubelet will use its
cgroupDriver setting from kubeletConfig (or `--cgroup-driver` commandline
flag). This is also the case if the kubelet does not support the new field:
the information about cgroup driver advertised by the runtime will be just
ignored by kubelet and it will resort to its own configuration settings. Note:
this does present a configuration skew risk, but that risk is the same as
currently exists today.

The fallback behavior will be removed along with the `--cgroup-driver` flag and
cgroupDriver option in a few releases after GA, as per the
[Kubernetes deprecation policy][deprecation-policy].
At this point the kubelet relies on the
container runtime to implement the feature. In practice, this means the cluster
must use at least containerd v2.0 or cri-o v1.28 as a prerequisite for
upgrading.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: KubeletCgroupDriverFromCRI
  - Components depending on the feature gate: kubelet

###### Does enabling the feature change any default behavior?

Yes.

When the runtime is updated to a version that supports this, kubelet
will ignore the cgroupDriver config option/flag. However, this change in
behavior should not cause any breakages (on the contrary, it should fix
scenarios where the kubelet `--cgroup-driver` setting is incorrectly
configured). With old versions of the container runtimes (that don't support
the new field in the CRI API) the default behavior is not changed.

When the `--cgroup-driver` setting is removed, the fallback behavior is dropped
and the kubelet requires the CRI runtime to implement the feature (see
[Version Skew Strategy](#version-skew-strategy)).

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

In alpha and beta, yes, through the feature gate.

In GA, no.

###### What happens if we reenable the feature if it was previously rolled back?

Kubelet starts to use the cgroup driver instructed by the runtime. Potentially
fixing a broken/misbehaving node if the kubelet cgroupDriver option (or
`--cgroup-driver` flag) was incorrectly set.

###### Are there any tests for feature enablement/disablement?

Unit tests for the feature gate will be written.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout/rollback could fail only in the way it currently does: cgroup driver
skew between CRI server (containe runtime) and Kubelet resulting in nodes going
NotReady. This is only possible when the CRI server and Kubelet are not both
upgraded to support the feature and are both not configured to agree on the
CgroupDriver as they must be today.

###### What specific metrics should inform a rollback?

`cri_losing_support` metric will be populated on nodes where the CRI implementation will one day lose support. After 1.37, kubelet will fatally error,
so admins should upgrade their out of support CRI implementations (if `version==1.37`).

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not planned as there is no persistent state associated with the feature. Manual
testing of the feature gate (in addition to the unit tests) is performed.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

Yes, the CgroupDriver field of the Kubelet configuration (and the
corresponding `--cgroup-driver` flag) will be marked as deprecated.

After GA, the CgroupDriver configuration option and the `--cgroup-driver` flag
will be removed in a future release as per the
[Kubernetes deprecation policy][deprecation-policy]

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Kubelet and container runtime version. The
[`crictl`](https://kubernetes.io/docs/tasks/debug/debug-cluster/crictl/) tool
can be used to determine if the container runtime supports the feature (`crictl
info`).

###### How can someone using this feature know that it is working for their instance?

The metric `cri_losing_support` when `version == 1.37` will indicate those nodes will be out of support in 1.37.
If that metric is unpopulated, the feature is on (as it's GA) and the flag fallback is not being used.

After GA, the CgroupDriver configuration option and the `--cgroup-driver` flag
will be removed in a future release, in accordance with the
[Kubernetes deprecation policy][deprecation-policy]. At that point, the kubelet
will refuse to start if the required feature is not functioning correctly. This
failure can be observed in system logs, with the node either entering a
NotReady state or failing to register during cluster bootstrap. The behavior
will be similar to other critical CRI server errors.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

The metric `cri_losing_support` when `version == 1.37` will indicate those nodes will be out of support in 1.37.
If that metric is unpopulated, the feature is on (as it's GA) and the flag fallback is not being used.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

A CRI (server) implementation of the correct version. However, the feature will
fallback if the CRI implementation doesn’t support the feature.

After GA, the fallback behavior will be removed in a future release, as per the
[Kubernetes deprecation policy][deprecation-policy]. At this point, a
sufficiently recent version of the CRI runtime is a hard requirement.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

For the Kubernetes API, no.

For the CRI API, yes. Although the CRI fields and messages are not exposed to
the user.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

For the Kubernetes API, no.

For the CRI API, yes, minimally.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Not noticeably.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Not applicable. This feature is node-local and between kubelet and the
container runtime, only.

###### What are other known failure modes?

Same that exists today: Kubelet and the CRI server (container runtime) not
agreeing on the CgroupDriver while one of them doesn’t support the feature.

After GA, the fallback behavior will be removed in a future release, as per the
[Kubernetes deprecation policy][deprecation-policy]. At this point,
the kubelet requires the CRI runtime to implement the feature and will
refuse to start if it is not supported. As a result, the minimum required
versions for containerd is v2.0 and for cri-o is v1.28.

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A.

## Implementation History

- v1.28: alpha
- v1.31: beta

## Drawbacks

## Alternatives

Make kubelet the configuration point for cgroup driver so that kubelet would
inform the runtime which cgroup driver to use. This could be achieved e.g.
without any changes to the CRI API by the CRI implementation guessing the
cgroup driver based on the path of the CgroupParent of the pod, passed down in
the RunPodSandboxRequest. However, SIG Node has decided that the CRI
implementation should begin to be the source of truth for low-level choices
like this, and thus this approach was chosen.

## Infrastructure Needed (Optional)
