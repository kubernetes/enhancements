# Skip Volume Ownership Change

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Graduation Criteria](#graduation-criteria)
  - [Test Plan](#test-plan)
  - [Monitoring](#monitoring)
- [Implementation History](#implementation-history)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
- [Infrastructure Needed [optional]](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

Currently before a volume is bind-mounted inside a container the permissions on
that volume are changed recursively to the provided fsGroup value.  This change
in ownership can take an excessively long time to complete, especially for very
large volumes (>=1TB) as well as a few other reasons detailed in [Motivation](#motivation).
To solve this issue we will add a new field called `pod.Spec.SecurityContext.FSGroupChangePolicy` and
allow the user to specify how they want the permission and ownership change for volumes used by pod to happen.

## Motivation

When a volume is mounted on the node, we recursively change permissions of volume
before bind mounting the volume inside container. The reason of doing this is to ensure
that volumes are readable/writable by provided fsGroup.

But this presents following problems:
 - An application(many popular databases) which is sensitive to permission bits changing
   underneath may refuse to start whenever volume being used inside pod gets mounted on
   different node.
 - If volume has a large number of files, performing recursive `chown` and `chmod`
   could be slow and could cause timeout while starting the pod.

### Goals

 - Allow volume ownership and permission change to be skipped during mount

### Non-Goals

 - In some cases if user brings in a large enough volume from outside, the first time ownership and permission change still could take lot of time.
 - On SELinux enabled distributions we will still do recursive chcon whenever applicable and handling that is outside the scope.
 - This proposal does not attempt to fix two pods using same volume with conflicting fsgroup. `FSGroupChangePolicy` also will be only applicable to volume types which support setting fsgroup.

## Proposal

We propose that an user can optionally opt-in to skip recursive ownership(and permission) change on the volume if volume already has right permissions.

### Implementation Details/Notes/Constraints [optional]

Currently Volume permission and ownership change is done using a breadth-first algorithm starting from root directory of the volume. As part of this proposal we will change the algorithm to modify permissions/ownership of the root directory after all directories/files underneath have their permissions changed.

When creating a pod, we propose that `pod.Spec.SecurityContext` field expanded to include a new field called `FSGroupChangePolicy` which can have following possible values:

 - `Always` --> Always change the permissions and ownership to match fsGroup. This is the current behavior and it will be the default one when this proposal is implemented. Algorithm that performs permission change however will be changed to perform permission change of top level directory last.
 - `OnRootMismatch` --> Only perform permission and ownership change if permissions of top level directory does not match with expected permissions and ownership.

```go
type PodFSGroupChangePolicy string

const(
    OnRootMismatch PodFSGroupChangePolicy = "OnRootMismatch"
    AlwaysChangeVolumePermission PodFSGroupChangePolicy = "Always"
)

type PodSecurityContext struct {
    // FSGroupChangePolicy ← new field
    // Defines behavior of changing ownership and permission of the volume
    // before being exposed inside Pod. This field will only apply to
    // volume types which support fsGroup based ownership(and permissions).
    // It will have no effect on ephemeral volume types such as: secret, configmaps
    // and emptydir.
    // + optional
    FSGroupChangePolicy *PodFSGroupChangePolicy
}
```

### Risks and Mitigations

- One of the risks is if user volume's permission was previously changed using old algorithm(which changes permission of top level directory first) and user opts in for `OnRootMismatch` `FSGroupChangePolicy` then we can't distinguish if the volume was previously only partially recursively chown'd.

## Production Readiness Review Questionnaire

### Feature enablement and rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate
    - Feature gate name: ConfigurableFSGroupPolicy
    - Components depending on the feature gate: kubelet, kube-apiserver

* **Does enabling the feature change any default behavior?**
  No enabling the feature gate does not change any default behaviour.

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  Yes the feature-gate can be disabled once enabled. This will cause all volume mounts that
  require permission change to use recursive permission and ownership change. This is the default
  behaviour without the feature gate.

* **What happens if we reenable the feature if it was previously rolled back?**
  For pods that have expected value in `pod.Spec.SecurityContext.FSGroupChangePolicy` as defined in this KEP,
  it will start using specified policy.

* **Are there any tests for feature enablement/disablement?**
  There aren't any e2e but there are unit tests that cover this behaviour.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**
  Since this feature requires users to opt-in by setting new field in pod spec, it should
  not impact already running workloads.

* **What specific metrics should inform a rollback?**
  If after enabling this feature and setting `pod.Spec.SecurityContext.FSGroupChangePolicy`
  users notice an increase in volume mount time via `storage_operation_duration_seconds{operation_name=volume_mount}`
  or an increase in mount error count via `storage_operation_errors_total{operation_name=volume_mount}`
  then a cluster-admin may want to rollback the feature.

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**
  We have not fully tested upgrade and rollback. We have unit tests that cover the scenario
  of feature gate being enabled and then disabled. But we will need to do more upgrade->downgrade->upgrade testing.

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**
  This feature deprecates no existing functionality.


### Monitoring requirements

* **How can an operator determine if the feature is in use by workloads?**
  Operator can query `pod.Spec.SecurityContext.fsGroupChangepolicy` field and identify if this is being set to non-default values.

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**
  - [x] Metrics
    - mount operation duration:
        - Metric name: storage_operation_duration_seconds{operation_name=volume_mount}
        - [Optional] Aggregation method: percentile
        - Components exposing the metric: kubelet
    - mount operation errors:
        - Metric name: storage_operation_errors_total{operation_name=volume_mount}
        - [Optional] Aggregation method: cumulative counter
        - Components exposing the metric: kubelet
    - volume ownerhip change timing mtrics: We are also going to add metrics that track time it takes for volume ownerhip change to happen. We will update this section with the name of metrics.



* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  It is hard to give numbers that an admin could use to determine health of mount operation. In general we expect that after this feature is rolled out
  and user starts using it by selecting `OnRootMismatch` `fsGroupChangepolicy` then - `storage_operation_duration_seconds{operation_name=volume_mount}`
  should go down and there should not be an spike in mount error metric (reported via `storage_operation_errors_total{operation_name=volume_mount}`).

* **Are there any missing metrics that would be useful to have to improve
  observability if this feature?**
  As documented above we already have metrics for tracking mount timing. We are planning to add a metric
  for time it takes to change permission of a volume before mount but this is not necessary for observability of this feature.

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
  Not applicable

### Scalability

* **Will enabling / using this feature result in any new API calls?**
  This feature should cause no new API calls.

* **Will enabling / using this feature result in introducing new API types?**
  This feature introduces no new API types.

* **Will enabling / using this feature result in any new calls to cloud
  provider?**
  No. This feature has no cloud-provider integration.

* **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**
  Since this feature adds a new field to pod's securitycontext, it will increase API size of
  Pod object:
  Describe them providing:
  - API type(s): Pod
  - Estimated increase in size: (e.g. new annotation of size 32B): 35B


* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**
  No. If anything this feature will reduce time it takes for a Pod to start.

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  No. This feature should not cause any increase in memory or CPU usage of the affected component.

### Troubleshooting

Troubleshooting section serves the `Playbook` role as of now. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now we leave it here though.

* **How does this feature react if the API server and/or etcd is unavailable?**
  Not applicable

* **What are other known failure modes?**
  - A brown field volume has correct top level permission but incorrect leaf level permission.
    - Detection: If user has used `fsGroupChangePolicy` as `OnRootMismatch` in pod's security context and somehow top level directory has right permission but subdirectories don't then when pod attempts to read/write those subdirectories it could fail. So there are no metrics for this error but could result in pod not running correctly.
    - Mitigations: This failure does not affect existing workloads and user can switch to using `Always` in `fsGroupChangePolicy` of pod's security context.
    - Diagnostics: Pod does not run correctly.
    - Testing: We do not attempt to check for leaf level perissions if `fsGroupChangePolicy` is set to `OnRootMismatch`(which would defeat the purpose of this feature). But since user needs to opt-in for using this feature, they should be aware of side effects.

* **What steps should be taken if SLOs are not being met to determine the problem?**
  If admin notices an increase in mount errors or increase in mount timings as documented in SLIs then an admin could:
      - check number of pods that are setting non-defaults in `pod.Spec.SecurityContext.fsGroupChangePolicy`
      - Check volume mount and latency metrics (as described in SLI)
      - Check kubelet logs for mount errors or problems.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Graduation Criteria

* Alpha in 1.18 provided all tests are passing and gated by the feature Gate
   `ConfigurableFSGroupPermissions` and set to a default of `False`

* Beta in 1.20 with design validated by at least two customer deployments
  (non-production), with discussions in SIG-Storage regarding success of
  deployments.  A metric will be added to report time taken to perform a
  volume ownership change. Also e2e tests that verify volume permissions with various `FSGroupChangePolicy`.
* GA in 1.21, with Node E2E tests in place tagged with feature Storage


[umbrella issues]: https://github.com/kubernetes/kubernetes/issues/69699

### Test Plan

A test plan will consist of the following tests

* Basic tests including a permutation of the following values
  - pod.Spec.SecurityContext.FSGroupChangePolicy (OnRootMismatch/Always)
  - PersistentVolumeClaimStatus.FSGroup (matching, non-matching)
  - Volume Filesystem existing permissions (none, matching, non-matching, partial-matching?)
* E2E tests


### Monitoring

We will add a metric that measures the volume ownership change times.

## Implementation History

- 2020-01-20 Initial KEP pull request submitted

## Drawbacks [optional]


## Alternatives [optional]

We considered various alternatives before proposing changes mentioned in this proposal.
- We considered using a shiftfs(https://github.com/linuxkit/linuxkit/tree/master/projects/shiftfs) like solution for mounting volumes inside containers without changing permissions on the host. But any such solution is technically not feasible until support in Linux kernel improves.
- We also considered redesigning volume permission API to better support different volume types and different operating systems because `fsGroup` is somewhat Linux specific. But any such redesign has to be backward compatible and given lack of clarity about how the new API should look like, we aren't quite ready to do that yet.

## Infrastructure Needed [optional]
