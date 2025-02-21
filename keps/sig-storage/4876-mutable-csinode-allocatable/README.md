# KEP-4876: Mutable CSINode Allocatable Property

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
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Feature Gate](#feature-gate)
  - [API Changes](#api-changes)
    - [CSINode](#csinode)
    - [CSIDriver](#csidriver)
    - [Validation Changes](#validation-changes)
    - [Volume Plugin Manager](#volume-plugin-manager)
    - [NodeInfoManager Interface Extension](#nodeinfomanager-interface-extension)
    - [CSINode Update Behavior](#csinode-update-behavior)
    - [Pod Construction Changes](#pod-construction-changes)
    - [Scheduler Enhancements](#scheduler-enhancements)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes changes to make the `CSINode.Spec.Drivers[*].Allocatable.Count` field mutable and introduces a mechanism to update it dynamically based on user configuration at the CSI driver level. These updates can be triggered either by periodic intervals or by failure detection (such as volume attachment failures due to insufficient capacity). This improvement enhances the reliability of stateful pod scheduling by addressing mismatches between reported and actual attachment capacity on nodes.

## Motivation

Currently, a mismatch between the reported and actual attachment capacity on nodes can result in permanent scheduling errors and stuck workloads. This occurs when volume slots are taken after a CSI driver starts up, which results in `kube-scheduler` assigning stateful pods to nodes lacking the necessary capacity to support them. This mismatch can happen due to various scenarios, such as:

1. Operations out of band with respect to CSI drivers and Kubernetes:
   - Manual attachment of volumes by administrators or external controllers.

2. Multi-driver scenarios:
   - When multiple CSI drivers are used on a node and one driver's operations affect the available capacity for others.

3. Other devices consuming available slots:
   - Network interfaces taking up slots.
   - GPU or specialized hardware attachments that weren't present during CSI driver initialization.

These scenarios can lead to the CSI driver reporting an initial capacity that becomes inaccurate over time, causing the scheduler to make decisions based on outdated information. This results in pods being scheduled to nodes without sufficient capacity, ultimately getting stuck in a `ContainerCreating` state.

By making the `CSINode.Spec.Drivers[*].Allocatable.Count` field mutable and introducing a mechanism to update it dynamically, we can ensure that the scheduler always has information which more accurately represents the actual state of the world, significantly improving the reliability of stateful pod scheduling.

### Goals

- Make `CSINode.Spec.Drivers[*].Allocatable.Count` mutable.
- Enable CSI drivers to define the interval at which the `Allocatable.Count` value on each node is updated through the `CSIDriver` object.
- Automatically update `CSINode.Spec.Drivers[*].Allocatable.Count` upon detecting a failure in volume attachment due to insufficient capacity.

### Non-Goals

- Modifying the core scheduling logic of Kubernetes.
- Implementing cloud provider-specific solutions within Kubernetes core.
- Re-scheduling pods stuck in a `ContainerCreating` state.

## Proposal

### User Stories (Optional)

#### Story 1

As a cluster administrator, I want the reported attachment capacity on nodes to accurately reflect the actual capacity, so that stateful pods are reliably scheduled and do not become stuck in a `ContainerCreating` state due to insufficient capacity.

#### Story 2

As a cluster operator, I use volumes during node setup for components like kubelet, containerd, and additional drivers. These boot volumes, which are not managed by CSI, may be detached after setup, and I need a way to reclaim these slots for other uses. The current static capacity reporting doesn't allow for this flexibility.

#### Story 3

As a cluster operator, I need the Kubernetes scheduler to accurately count the number of available device slots for both storage volumes and network interfaces. On certain machine types, network interfaces and volumes share device slots, and network interfaces may be dynamically attached after the CSI driver is registered. This results in an inaccurate `Allocatable.Count` for volumes, causing stateful pods to be scheduled on nodes with insufficient capacity, ultimately getting stuck in a `ContainerCreating` state.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

The following risks are identified:

- Frequent updates/retrieval of the `CSINode` object could increase API server load.

- Frequent calls to a CSI driver's `NodeGetInfo` RPC endpoint may become expensive, particularly if the operation involves retrieving information from a remote server or performing resource-intensive tasks. Specifically, this is a concern at scale, where the cumulative cost of multiple nodes repeatedly querying for updates is more impactful.

- There's a race condition where the scheduler might assign a stateful pod to a node with insufficient capacity if the `CSINode.Spec.Drivers[*].Allocatable.Count` value hasn't been updated in time.

The risks are mitigated as follows:

- The use of the Kubernetes informer pattern in the scheduler. The scheduler uses a `CSINode` informer and lister to efficiently access and watch `CSINode` objects (this logic is already present).

- Allow users to opt in to this feature at a per-CSI driver granularity by configuring the `CSIDriver` object. Specifically, administrators will be able to fine-tune the interval update value via the `NodeAllocatableUpdatePeriodSeconds` attribute in the `CSIDriver` object as per their specific requirement.

- A reactive update mechanism is implemented to immediately update the `CSINode.Spec.Drivers[*].Allocatable.Count` value if a pod fails to enter a running state due volume attachment failures as a result of insufficient capacity. This ensures that even if a race occurs, Kubernetes quickly corrects itself and prevents further scheduling errors.

## Design Details

### Feature Gate

A new feature gate - `MutableCSINodeAllocatableCount` - will be introduced to control the functionality implemented by this KEP. When the feature gate is disabled, the `CSINode` object will remain immutable, maintaining the current behavior.

### API Changes

#### CSINode

The `CSINode.Spec.Drivers[*].Allocatable.Count` field will be made mutable. No changes to the object structs are needed, only the validation logic needs to be revised. For reference, these are the API fields this KEP proposes to make mutable.

```golang
// CSINodeDriver holds information about the specification of one CSI driver installed
type CSINodeDriver struct {
    ...
    // allocatable represents the volume resources of a node that are available for sc
    // +optional
    Allocatable *VolumeNodeResources
}
```

```golang
// VolumeNodeResources is a set of resource limits for scheduling of volumes.
type VolumeNodeResources struct {
    // Maximum number of unique volumes managed by the CSI driver that can be used on
    // A volume that is both attached and mounted on a node is considered to be used o
    // The same rule applies for a unique volume that is shared among multiple pods on
    // If this field is not specified, then the supported number of volumes on this no
    // +optional
    Count *int32
}
```

#### CSIDriver

A new field, `NodeAllocatableUpdatePeriodSeconds`, will be added to the `CSIDriverSpec` struct. This field allows a CSI driver to specify the interval at which the Kubelet should periodically query a driver's `NodeGetInfo` RPC endpoint to update the `CSINode` object. If this field is not set, updates will only occur in response to volume attachment failures as a result of no capacity.

```golang
// CSIDriverSpec is the specification of a CSIDriver.
type CSIDriverSpec struct {
    ...
    // NodeAllocatableUpdatePeriodSeconds specifies the interval between periodic updates of
    // the CSINode allocatable capacity for this driver. If not set, periodic updates
    // are disabled, and updates occur only upon detecting capacity-related failures.
    // The minimum allowed value for this field is 10 seconds.
    // +optional
    NodeAllocatableUpdatePeriodSeconds *int64
}
```

#### Validation Changes

The [ValidateCSINodeUpdate](https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/storage/validation/validation.go#L304) function in the API validation code path will be modified to allow updates to the `Allocatable.Count`
field when the feature gate is enabled:

```golang
func ValidateCSINodeUpdate(new, old *storage.CSINode) field.ErrorList {
    allErrs := ValidateCSINode(new)
    
    if utilfeature.DefaultFeatureGate.Enabled(features.MutableCSINodeAllocatableCount) {
        for _, oldDriver := range old.Spec.Drivers {
           for _, newDriver := range new.Spec.Drivers {
                // Allow Allocatable.Count to be modified
                // Ensure all other fields are unchanged
            }
        }
    } else {
        // Existing validation logic for when feature gate is disabled
    }
    return allErrs
}
```

This updated logic allows the `Allocatable.Count` field to be modified when the feature gate is enabled, while ensuring all other fields remain immutable. When the feature gate is disabled, it falls back to the existing validation logic for backward compatibility.

#### Volume Plugin Manager

A new goroutine will be started in VolumePluginMgr’s [Run()](https://github.com/kubernetes/kubernetes/blob/master/pkg/volume/plugins.go#L953) func if the `NodeAllocatableUpdatePeriodSeconds` is set to a nonzero value. This goroutine will periodically trigger updates to the `CSINode` object based on the specified interval:

```golang
func (pm *VolumePluginMgr) Run(stopCh <-chan struct{}) {
    if pm.csiNodeUpdateInterval > 0 {
        go wait.Until(pm.updateCSINodeInfo, pm.csiNodeUpdateInterval, stopCh)
    }
}
```

In case of a failure during the `updateCSINodeInfo` call, the `Allocatable.Count` will retain its current value and `updateCSINodeInfo` will be retried.


#### NodeInfoManager Interface Extension

The existing [NodeInfoManager](https://github.com/kubernetes/kubernetes/blob/master/pkg/volume/csi/nodeinfomanager/nodeinfomanager.go#L76) interface will be extended to include a new method for updating the `CSINode` object:

```golang
// Interface implements an interface for managing labels of a node
type Interface interface {
    CreateCSINode() (*storagev1.CSINode, error)
    ...
    // UpdateCSINode updates the CSINode object
    UpdateCSINode() error
}
```

#### CSINode Update Behavior

This table explains how updates to the `CSINode.Spec.Drivers[*].Allocatable.Count` field are handled, depending on the status of the `MutableCSINodeAllocatableCount` feature flag and the `NodeAllocatableUpdatePeriodSeconds` field in the `CSIDriver` object.

| **Feature Flag Status**                  | **`NodeAllocatableUpdatePeriodSeconds`** | **Behavior**                                                                                                                           |
|------------------------------------------|-------------------------------------|------------------------------------------------------------------------------------------------------------------------------------|
| Enabled                              | Set                                 | Periodic updates occur at the defined interval + when invalid state is detected (volume attachment failures due to `ResourceExhausted`)|
| Enabled                              | Not set                             | Updates occur only in response to volume attachment failures (`ResourceExhausted` errors)                                             |
| Disabled                             | Set                                 | `NodeAllocatableUpdatePeriodSeconds` is ignored; `Allocatable.Count` remains static and immutable                                              |
| Disabled                             | Not set                             | No updates occur; `Allocatable.Count` remains static and immutable                                                                    |


#### Pod Construction Changes

To address race conditions where the scheduler assigns stateful pods to nodes with insufficient capacity, Kubelet's pod construction process during [WaitForAttachAndMount](https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/volumemanager/volume_manager.go#L393) will now handle `ResourceExhausted` errors returned by CSI drivers during the `ControllerPublishVolume` RPC.

The `ResourceExhausted` error is directly reported on the `VolumeAttachment` object associated with the relevant attachment. To facilitate easier detection of `ResourceExhausted` errors from `VolumeAttachment` statuses, we propose adding a `StatusCode` field to the [VolumeError](https://github.com/kubernetes/api/blob/master/storage/v1/types.go#L219) struct.

```golang
if err := kl.volumeManager.WaitForAttachAndMount(pod); err != nil {
    if isResourceExhaustedError(err) {
        // Update CSINode using a backoff mechanism
        // Generate event for affected pod
    } else {
        // Existing error handling
    }
}
```

This change ensures that when a pod fails to be constructed due to insufficient volume attachment capacity, that both:

1. The `CSINode` object is promptly updated to reflect the actual available capacity, improving future scheduling decisions.
2. An event is added to the pod, providing visibility to cluster operators and enabling automated actions by components like
the Kubernetes [descheduler](https://github.com/kubernetes-sigs/descheduler) to fix the stateful pods stuck in `ContainerCreating`.

#### Scheduler Enhancements

The CSI volume limits scheduler plugin currently only registers for `CSINode` "Add" events. To ensure the scheduler promptly reacts to "Update" events as well, we need to modify the [EventsToRegister()](https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/framework/plugins/nodevolumelimits/csi.go#L85) function in the scheduler plugin to include:

```go
{Event: framework.ClusterEvent{Resource: framework.CSINode, ActionType: framework.Update}}
```

This enhancement makes it such that when the `Allocatable.Count` property is updated, the scheduler re-queues previously unschedulable pods to attempt scheduling with updated capacity information.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[X] I/we understand the owners of the involved components may require updates to
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

- `k8s.io/kubernetes/pkg/kubelet`: 2024-09-24 - 51%
- `k8s.io/kubernetes/pkg/apis/storage/validation`: 2024-09-24 - 96%
- `k8s.io/kubernetes/pkg/volume/plugins.go`: 2024-09-24 - 27.9%
- `k8s.io/kubernetes/pkg/volume/csi/nodeinfomanager`: 2024-09-24 - 76.6%

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- Test that updates to `CSINode.Spec.Drivers[*].Allocatable.Count` are properly reflected when the `MutableCSINodeAllocatableCount` feature gate is enabled.
- Test that periodic updates occur at the specified `NodeAllocatableUpdatePeriodSeconds`.
- Test that `ResourceExhausted` errors during volume attachment trigger an immediate update of the CSINode object.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- Test the impact on pod scheduling when `CSINode.Spec.Drivers[*].Allocatable.Count` is updated, ensuring that pods are not scheduled to nodes with insufficient capacity.
- Test the end-to-end workflow of updating `CSINode.Spec.Drivers[*].Allocatable.Count` using a CSI driver.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag.
- Initial unit tests/integration tests completed and enabled.

#### Beta

- Allowing time for feedback (at least 2 releases between beta and GA).
- All unit tests/integration/e2e tests completed and enabled.
  - Validate kubelet behavior when API server rejects `CSINode` updates (older API server version).
  - Validate CSI driver behavior with and without the `NodeAllocatableUpdatePeriodSeconds` field set.
  - Validate scheduler behavior remains consistent regardless of whether `CSINode.Spec.Drivers[*].Allocatable.Count` is being dynamically updated or not.
  - Validate cluster autoscalar behavior is not changed as a result of changes introduced in this KEP.
  - Validate pod construction failure handling in kubelet, ensuring it correctly updates the `CSINode` object when the feature is enabled and the API server supports it.

#### GA

- All beta criteria have been satisfied.
- Feature is stable.
- No bug reports / feedback / improvements to address.

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

- Upgrade Strategy
  - Upgrade the API server first to support mutable `CSINode.Spec.Drivers[*].Allocatable.Count` and the new `NodeAllocatableUpdatePeriodSeconds` field in `CSIDriver` object.
  - Upgrade nodes
  - Update CSI drivers to take advantage of the new feature, if desired.

- Downgrade Strategy
  - If downgrading the API server, ensure that nodes are downgraded first to avoid rejected `CSINode` update attempts.
  - CSI drivers using the `NodeAllocatableUpdatePeriodSeconds` feature should be reconfigured to not use this field before downgrading the API server.

### Version Skew Strategy

This enhancement primarily involves changes to the kubelet and the API server, with no impact on the scheduler. Here's how the system will behave in various version skew scenarios:

- API Server considerations
  - Older API server versions will reject updates to the `CSINode.Spec.Drivers[*].Allocatable.Count` field and won't recognize the `NodeAllocatableUpdatePeriodSeconds` field in the `CSIDriver` object.

- Kubelet version considerations
  - Newer kubelet (with this feature) + Older API server: The kubelet will attempt to update the `CSINode.Spec.Drivers[*].Allocatable.Count` field due to capacity failures, but these updates will be rejected by the API server.
  - Older kubelet + Newer API server: Volume attachment failures due to capacity issues will not trigger `CSINode` updates during pod construction.

- Scheduler considerations
  - The scheduler is not directly affected by this change and will continue to use the latest `CSINode.Spec.Drivers[*].Allocatable.Count` value for scheduling decisions, regardless of whether it's being updated or not.

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

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `MutableCSINodeAllocatableCount`
  - Components depending on the feature gate: `kube-apiserver`, `kube-controller-manager`, `kubelet`.

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

- The `CSINode.Spec.Drivers[*].Allocatable.Count` field becomes mutable and the kubelet will attempt to update this field when a pod fails to enter a ready state 
due to a volume attachment failure due to insufficient capacity.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

- Yes, the feature can be disabled by turning off the feature gate.

###### What happens if we reenable the feature if it was previously rolled back?

- The `CSINode.Spec.Drivers[*].Allocatable.Count` field will become mutable again.

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

- Yes, unit tests will be implemented to verify the behavior of the `ValidateCSINodeUpdate` function when the feature gate is enabled and disabled.

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

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

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

Yes, there will be new API calls to update the `CSINode` object:

```
API call type: PATCH
Estimated throughput: Depends on the `NodeAllocatableUpdatePeriodSeconds` setting and the frequency of volume attachment failures.
Originating component: Kubelet, KCM
```

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

- No, this feature does not introduce new API types.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

- No, this feature does not introduce new calls to the cloud provider directly. However, CSI drivers may make additional calls to retrieve updated capacity information.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

```
API Object: CSIDriver
Estimated increase in size: New `NodeAllocatableUpdatePeriodSeconds` field (approximately 32 bytes)
Estimated amount of new objects: No new objects, only modification of existing CSIDriver objects
```

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

- This feature should not impact existing SLIs/SLOs. The `CSINode` updates are asynchronous and should not directly affect pod startup times or API responsiveness.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

- The feature may result in a slight increase in CPU and network usage on nodes due to periodic `CSINode` updates and more frequent calls to the CSI driver's `NodeGetInfo` RPC.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

- This feature should not result in resource exhaustion of node resources. The additional goroutine and API calls are minimal and should not significantly impact the node's resources.

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

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

- Implementing a custom scheduler:  This approach was rejected for several reasons.
  - It would significantly degrade the customer experience, as users would need to deploy and manage an additional component.
  - This issue is not a niche use case; it affects a wide range of CSI drivers and cloud providers.
  - The default Kubernetes scheduler heavily relies on the `CSINode` allocatable object to make informed decisions about node capacity. Implementing a custom scheduler is arguably workaround solution
  that does not address the root cause and inherent limitation of the immutable `CSINode` object today.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
