<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# [KEP-5055](https://github.com/kubernetes/enhancements/issues/5055): DRA: device taints and tolerations

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Degraded Devices](#degraded-devices)
    - [External Health Monitoring](#external-health-monitoring)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
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
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

With Dynamic Resource Allocation (DRA), DRA drivers publish information about
the devices that they manage in ResourceSlices. This information is used by the
scheduler when selecting devices for user requests in ResourceClaims.

With this KEP, DRA drivers can mark devices as tainted such that they won't be
used for scheduling new pods. In addition, pods already running with access to
a tainted device can be stopped automatically. Cluster administrators can do
the same by creating a
[ResourceSlicePatch](../5027-dra-admin-controlled-device-attributes) with a
taint.

Users can decide to ignore specific taints by adding tolerations to their
ResourceClaim.

## Motivation

### Goals

- Enable taking devices offline for maintenance while still allowing test pods
  to request and use those devices. Being able to do this one device at a time
  minimizes service level disruption.

- Enable users to decide whether they want to keep running a workload in a degraded
  mode while a device is unhealthy or prefer to get pods rescheduled.

### Non-Goals

- Not part of the plan for alpha: developing a kubectl command for managing device taints.
  This may be reconsidered.

## Proposal

### User Stories

#### Degraded Devices

A driver itself can detect problems which may or may not be tolerable for
workloads, like degraded performance due to overheating. Removing such devices
from the ResourceSlice would unconditionally prevent using them for new
pods. Instead, publishing with a taint informs users about this degradation and
leaves them the choice whether the device is still usable enough to run pods.
It also automates stopping pods which don't tolerate such a degradation.

#### External Health Monitoring

As cluster admin, I am deploying a vendor-provided DRA driver together with a
separate monitoring component for hardware aspects that are not available or
not supported by that DRA driver. When that component detects problems, it can
check its policy configuration and decide to take devices offline by creating
a ResourceSlicePatch with a taint for affected devices.

### Risks and Mitigations

A device can be identified by its names (`<driver name>/<pool name>/<device
name>`) and/or by its attributes (for example, a unique ID). It was a conscious
decision for core DRA to not require that the name is tied to one particular
hardware instance to support hot-swapping. Admins might favor using the names
whereas health monitoring might prefer to be specific and use a vendor-defined
unique ID. Both are supported, which creates additional complexity.

Without a kubectl extension similar to `kubectl taint nodes`, the user
experience for admins will be a bit challenging. They need to decide how to
identify the device (by name or with a CEL expression), manually create a
ResourceSlicePatch with a unique name, then remember to remove that
ResourceSlicePatch again. For beta, support in `kubectl` for common
operations may be needed.

Users might be tempted to tolerate taints to get their pods running. They do
that at their own risk. Depending on the taint, the application then may not
get the performance it needs (degraded hardware) or may fail at runtime
(hardware gets turned off). Admission controllers or validating admission
policies could be deployed to limit which tolerations may be used, but as
taints are not defined by Kubernetes itself, none of that is part of Kubernetes
itself.

## Design Details

The feature is following the approach and APIs taken for node taints and
applies them to devices. A new controller watches tainted devices and deletes
pods using them unless they tolerate the device taint, similar to the
[taint-eviction-controller](https://github.com/kubernetes/kubernetes/blob/32130691a4cb8a1034b999341c40e48d197f5465/pkg/controller/tainteviction/taint_eviction.go#L81-L83). A pod which is running or has finalizers will not get removed
immediately. Instead, the `DeletionTimestamp` gets set. That's okay for
the purpose of this KEP:
- The kubelet will stop any running containers and mark the pod as completed.
- The ResourceClaim controller will remove such a completed pod from the claim's
  `ReservedFor` and deallocate the claim once it has no consumers.

Taints are cumulative as long as the key and effect pairs are different:
- Taints defined by an admin in a ResourceSlicePatch get added to the
  set of taints defined by the DRA driver in a ResourceSlice.
- Taints with the same key and effect get overwritten, using the same
  precedence as for attributes.

This merging will be implemented by the same code that also
overrides device attributes.

To ensure consistency among all pods sharing a ResourceClaim, the toleration
for taints gets added to the request in a ResourceClaim, not the pod. This also
avoids conflicts like one pod tolerating a taint for scheduling and some other
pod not tolerating that.

Device and node taints are applied independently. A node taint applies to all
pods on a node, whereas a device taint affects claim allocation and only those
pods using the claim.

### API

The ResourceSlice content gets extended:

```Go
// BasicDevice defines one device instance.
type BasicDevice struct {
    ...

    // If specified, the device's taints.
    //
    // The maximum number of taints is 8.
    //
    // This is an alpha field and requires enabling the DRADeviceTaints
    // feature gate.
    //
    // +optional
    // +listType=atomic
    // +featureGate=DRADeviceTaints
    Taints []DeviceTaint
}

// DeviceTaintsMaxLength is the maximum number of taints per device.
const DeviceTaintsMaxLength = 8

// The device this DeviceTaint is attached to has the "effect" on
// any claim and, through the claim, to pods that do not tolerate
// the Taint.
type DeviceTaint struct {
    // The taint key to be applied to a device.
    // Must be a label name.
    //
    // +required
    Key string

    // The taint value corresponding to the taint key.
    // Must be a label value.
    //
    // +optional
    Value string

    // The effect of the taint on claims that do not tolerate the taint
    // and through such claims on the pods using them.
    // Valid effects are NoSchedule and NoExecute. PreferNoSchedule as used for
    // nodes is not valid here.
    //
    // +required
    Effect DeviceTaintEffect

    // ^^^^
    //
    // Implementing PreferNoSchedule would depend on a scoring solution for DRA.
    // It might get added as part of that.

    // TimeAdded represents the time at which the taint was added.
    // For NoExecute taints, the current time is set automatically
    // when adding such a taint. There is no default for other taints.
    //
    // +optional
    TimeAdded *metav1.Time
}
```

Taint has the exact same fields as a v1.Taint, but the description is a bit
different. In particular, PreferNoSchedule is not valid.

Tolerations get added to a DeviceRequest:

```Go
type DeviceRequest struct {
    ...

    // If specified, the request's tolerations.
    //
    // Tolerations for NoSchedule are required to allocate a
    // device which has a taint with that effect. The same applies
    // to NoExecute.
    //
    // In addition, should any of the allocated devices get tainted
    // with NoExecute after allocation and that effect is not tolerated,
    // then all pods consuming the ResourceClaim get deleted to evict
    // them. The scheduler will not let new pods reserve the claim while
    // it has these tainted devices. Once all pods are evicted, the
    // claim will get deallocated.
    //
    // The maximum number of tolerations is 16.
    //
    // This is an alpha field and requires enabling the DRADeviceTaints
    // feature gate.
    //
    // +optional
    // +listType=atomic
    // +featureGate=DRADeviceTaints
    Tolerations []DeviceToleration
}

// DeviceTolerationsMaxLength is the maximum number of tolerations in a DeviceRequest.
const DeviceTolerationsMaxLength = 16

// The ResourceClaim this Toleration is attached to tolerate any taint that matches
// the triple <key,value,effect> using the matching operator <operator>.
type DeviceToleration struct {
    // Key is the taint key that the toleration applies to. Empty means match all taint keys.
    // If the key is empty, operator must be Exists; this combination means to match all values and all keys.
    // Must be a label name.
    //
    // +optional
    Key string

    // Operator represents a key's relationship to the value.
    // Valid operators are Exists and Equal. Defaults to Equal.
    // Exists is equivalent to wildcard for value, so that a ResourceClaim can
    // tolerate all taints of a particular category.
    //
    // +optional
    Operator TolerationOperator

    // Value is the taint value the toleration matches to.
    // If the operator is Exists, the value should be empty, otherwise just a regular string.
    // Must be a label value.
    //
    // +optional
    Value string

    // Effect indicates the taint effect to match. Empty means match all taint effects.
    // When specified, allowed values are NoSchedule and NoExecute.
    //
    // +optional
    Effect DeviceTaintEffect

    // TolerationSeconds represents the period of time the toleration (which must be
    // of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default,
    // it is not set, which means tolerate the taint forever (do not evict). Zero and
    // negative values will be treated as 0 (evict immediately) by the system.
    //
    // +optional
    TolerationSeconds *int64
}

// A toleration operator is the set of operators that can be used in a toleration.
//
// +enum
type DeviceTolerationOperator string

const (
    DeviceTolerationOpExists DeviceTolerationOperator = "Exists"
    DeviceTolerationOpEqual  DeviceTolerationOperator = "Equal"
)
```

As with Taint, these structs get duplicated to document DRA specific
behavior and to ensure that future extensions do not get inherited
accidentally.

Generated conversion code might make it possible to reuse existing helper
code. Alternatively, that code can be copied.

The DevicePatch also gets extended. It is possible to use
admin-controlled taints without enabling attribute overrides by enabling the
`v1alpha3` API and only the `DRADeviceTaints` feature, while leaving
`DRAAdminControlledDeviceAttributes` disabled, because then the
ResourceSlicePatch type is available with only the fields needed for
taints.

```Go
type DevicePatch struct {
    ...

    // If specified, the device's taints. Taints with unique key and effect
    // get added to the set of taints of the device. When key and effect
    // are used in multiple places, the same precedence rules as for attributes apply
    // (see the priority field).
    //
    // The maximum number of tolerations is 16.
    //
    // This is an alpha field and requires enabling the DRADeviceTaints
    // feature gate.
    //
    // +optional
    // +listType=atomic
    // +featureGate=DRADeviceTaints
    Taints []DeviceTaint
```

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

<!--
Generated with:

go test -cover ./pkg/apis/resource/validation  ./staging/src/k8s.io/dynamic-resource-allocation/structured | sed -e 's/.*\(k8s.io[a-z/-]*\).*coverage: \(.*\) of statements/- `\1`: \2/' | sort

-->

v1.32.0:

- `k8s.io/dynamic-resource-allocation/structured`: 91.3%
- `k8s.io/kubernetes/pkg/apis/resource/validation`: 98.6%
- `k8s.io/kubernetes/pkg/controller/tainteviction`: 81.8%

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

Integration tests for the new eviction manager will be useful to ensure that
permissions are correct.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

Useful E2E tests are checking that the scheduler really honors taints during
scheduling. Adding a taint in a ResourceSlice must evict a running pod. Same
for adding a taint through a ResourceSlicePatch.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Additional tests are in Testgrid and linked in KEP

#### GA

- 3 examples of real-world usage
- Allowing time for feedback
- [Conformance tests]

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

### Upgrade / Downgrade Strategy

Tainting gets disabled when downgrading to a release without support for it or
when disabling the feature. The effect is as if the taints weren't set.

### Version Skew Strategy

During version skew where the apiserver supports the feature and the scheduler
doesn't, taints can be set without encountering errors or
warnings, but they won't have any effect.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

It is possible to disable the feature through the feature gate while leaving
the API group enabled. This enables cleanup through the API.

Re-enabling is supported because ResourceSlicePatches remain in etcd even if
they are inaccessible and existing taints and tolerations are preserved during
updates.

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: DRADeviceTaints
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-scheduler
    - kube-controller-manager
- [X] Other
  - Describe the mechanism: resource.k8s.io/v1alpha3 API group
  - Will enabling / disabling the feature require downtime of the control
    plane? Yes, in the apiserver.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? No.

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. The behavior of scheduling changes when it was in use.
Running applications are not affected.

###### What happens if we reenable the feature if it was previously rolled back?

It takes effect again for scheduling and may evict pods.

###### Are there any tests for feature enablement/disablement?

This will be covered through unit tests for the apiserver and scheduler.

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

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

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
- [ ] API .status
  - Condition name: 
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

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

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

### Scalability

See [../5027-dra-admin-controlled-device-attributes/README.md#scalability] for a
discussion of the scalability of patching devices. The same applies to applying
taints through ResourceSlicePatch objects.

Handling eviction scales with the number of claims and pods using those claims.

###### Will enabling / using this feature result in any new API calls?

A fixed, small number of clients (primarily the scheduler and controller
manager) need to start watching ResourceSlicePatches.

Pods are already watched in the controller manager. Evicting them adds one call
per pod.

###### Will enabling / using this feature result in introducing new API types?

ResourceSlicePatches must be created explicitly by admins or controller
operated by admins. Kubernetes itself does not create them.

The number of ResourceSlicePatches is expected to be orders of
magnitude smaller than the number of ResourceSlices.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Enabling it doesn't. Using tolerations increases the size of ResourceClaims and
ResourceClaimTemplates.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Pod scheduling may become a bit slower because of the additional checks, but
only when pods use claims. There are no SLI/SLOs for pods using claims.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

For scheduling, tracking taints should be comparable to the overhead for
patching attributes.

For eviction, additional data structures will be needed to track taints and
tolerations. This should not be too large.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No, because the feature is not used on nodes.

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

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 1.33: first KEP revision and implementation

## Drawbacks

Distributing information across different objects of different types makes it
harder for users to get a complete view.

## Alternatives

The existing taint-eviction-controller could be extended to cover device
taints. However, cloning it lowers the risk of breaking existing stable functionality.

Tolerations for device taints could also be added to individual pods. This
seems less useful because if pods share the same claim, they are typically part
of one larger application with identical tolerations. Experimenting with a new
API in the beta ResourceClaim type is a bit easier than it would be in the GA
Pod type.
