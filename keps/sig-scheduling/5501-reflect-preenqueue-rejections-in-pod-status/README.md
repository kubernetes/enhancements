# KEP-5501: Reflect PreEnqueue rejections in Pod status

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [1. How should plugins provide the status message?](#1-how-should-plugins-provide-the-status-message)
  - [2. Should all PreEnqueue plugins report the status?](#2-should-all-preenqueue-plugins-report-the-status)
  - [3. What reason should be used for the Pod condition?](#3-what-reason-should-be-used-for-the-pod-condition)
  - [4. How should outdated messages be handled?](#4-how-should-outdated-messages-be-handled)
  - [User Stories](#user-stories)
    - [Diagnosing the Pods using DRA](#diagnosing-the-pods-using-dra)
    - [Using Deferred ResourceQuota Enforcement](#using-deferred-resourcequota-enforcement)
    - [Using custom Gang Scheduling implementation](#using-custom-gang-scheduling-implementation)
    - [Cluster Autoscaler and resource provisioning](#cluster-autoscaler-and-resource-provisioning)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Stale or missing Pod status on patch failure](#stale-or-missing-pod-status-on-patch-failure)
    - [Impact on scheduling throughput and latency](#impact-on-scheduling-throughput-and-latency)
- [Design Details](#design-details)
  - [1. PreEnqueue plugin interface change](#1-preenqueue-plugin-interface-change)
  - [2. API changes](#2-api-changes)
  - [3. Preventing redundant API calls](#3-preventing-redundant-api-calls)
  - [4. Asynchronous dispatch and state management](#4-asynchronous-dispatch-and-state-management)
  - [5. Logic flow](#5-logic-flow)
    - [Rejection path](#rejection-path)
    - [Success path](#success-path)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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

This KEP proposes enhancing the Kubernetes scheduler to update a Pod's status
with a descriptive message when a `PreEnqueue` plugin rejects it.
When a Pod is rejected at this early stage, it currently remains in a generic `Pending` state
with no direct feedback in its status, making observability and debugging difficult.
By reflecting the rejection reason in the Pod's status (and emitting a corresponding event),
cluster administrators and users will gain immediate insight into why a Pod is not being scheduled.

## Motivation

The `PreEnqueue` extension point in the scheduler framework is an important gate
that performs preliminary checks before a Pod is allowed into the scheduling cycle.
This prevents the scheduler from wasting cycles on Pods that cannot be scheduled due to fundamental issues.
This extension point is already utilized by some features like Scheduling Gates and Dynamic Resource Allocation (DRA).
Moreover, it is a planned integration point for future capabilities such as [Deferred ResourceQuota Enforcement](https://kep.k8s.io/5465)
and advanced [Gang Scheduling](https://kep.k8s.io/4671) implementations.

Currently, when a `PreEnqueue` plugin rejects a Pod, this decision is not communicated back to the user
via the Pod's status. The Pod simply remains `Pending`, leaving the user to manually inspect scheduler logs
or other components to diagnose the issue. This lack of feedback is particularly problematic for plugins
that operate transparently to the user, as the reason for the delay is completely hidden.
While the Scheduling Gates feature does apply a status condition, it does so via the kube-apiserver
with its own reason (`SchedulingGated`).

This proposal aims to create a unified and flexible mechanism within the scheduler
to report `PreEnqueue` rejections. By applying a condition to the Pod and emitting an event,
we can significantly improve the debuggability and observability of the scheduling process.
The implementation of this feature is made straightforward and performant by the
[Asynchronous API calls during scheduling KEP](https://kep.k8s.io/5229),
which allows the scheduler to dispatch status updates without blocking its main control loop.

This enhancement would also benefit other features that could be migrated to `PreEnqueue`,
such as volume binding, which currently needs to rely on `PreFilter` to check if all necessary PVCs are created
([issue](https://issues.k8s.io/129698)).
The current lack of feedback prevents us from extending `PreEnqueue` use cases.

### Goals

- Update a Pod's `.status.conditions` with a descriptive reason and message
  when a `PreEnqueue` plugin rejects the Pod.
- Utilize the existing asynchronous API calls feature
  to perform these status updates without impacting scheduler performance.
- Provide a mechanism for existing `PreEnqueue` use cases (like `DynamicResources` plugin)
  to adopt this new status reporting.
- Investigate migrating the responsibility for setting the `SchedulingGated` condition
  from the kube-apiserver to the kube-scheduler for consistency.

### Non-Goals

- This KEP will not change the core rejection logic of any existing `PreEnqueue` plugin,
  only how the outcome of that logic is communicated.
- This KEP will not alter the scheduling cycle or the behavior of other scheduler extension points.

## Proposal

When a `PreEnqueue` plugin rejects a Pod, the scheduler will asynchronously patch the Pod's status
to reflect this outcome. The patch will add or update a `PodCondition` to indicate that the Pod is not schedulable
due to a `PreEnqueue` check. To prevent redundant API calls, the scheduler will internally cache the last-reported status
and skip the patch if the rejection reason has not changed.

**Note:** This entire feature will be controlled by a new feature gate, `SchedulerPreEnqueuePodStatus`.
The enablement of this feature will be dependent on the `SchedulerAsyncAPICalls` feature gate being enabled,
as the asynchronous patching mechanism is critical to protect scheduler throughput.

### 1. How should plugins provide the status message?

**Proposed:** Adjust the `PreEnqueue` plugin interface to return an additional,
optional, user-facing message alongside the `Status` object.
This will introduce a **breaking change** to all plugins that implement the `PreEnqueue` extension point.

This flexibility is important for plugins that wish to avoid reporting transient rejections.
For example, the `DynamicResources` plugin might observe a rejection because the scheduler processes a Pod faster
than a `ResourceClaim` becomes visible through the watcher (see a [comment](https://issues.k8s.io/129698#issuecomment-2614991955)).
In such cases where the condition is expected to resolve in seconds, populating the status would be inappropriate noise.

- **Pros:**
  - Gives plugin authors explicit control over what users see.
  - Decouples internal rejection reasons from user-facing messages.
  - Allows plugins to opt out of reporting by returning an empty message.

- **Cons:**
  - Requires a breaking change to the `PreEnqueue` plugin interface.

- **Alternatives Considered:**
  - Use the raw status message: This is simpler as it requires no interface changes.
    However, these messages are often not written well for end-users.
    It also makes it difficult for a plugin to conditionally opt out of reporting a status.

### 2. Should all PreEnqueue plugins report the status?

**Proposed:** Make status reporting an opt-in or easily controllable behavior at the plugin level
(e.g., by returning an empty message described above).

- **Pros:**
  - Reduces API server load and potential "noise" for plugins that reject pods for very short,
    transient reasons (e.g., DRA, asynchronous preemption).
  - Gives plugin developers the flexibility to decide if a status update is valuable for their specific logic.

- **Cons:**
  - Could lead to perceived inconsistency, where some `PreEnqueue` rejections appear on the Pod status
    while others do not.

- **Alternatives Considered:**
  - Mandatory reporting for all rejections: This would make the scheduler's behavior uniform.
    However, it might create significant API churn for plugins that reject pods frequently and for short durations,
    negatively impacting performance and UX.

### 3. What reason should be used for the Pod condition?

**Proposed:** Introduce a new PodCondition reason: `NotReadyForScheduling`.
The status reporting for Scheduling Gates will be left intact (i.e., handled by the kube-apiserver with the `SchedulingGated` reason)
to avoid any disruption to existing behavior.

- **Pros:**
  - It clearly communicates the stage of rejection.
  - It provides a specific identifier for this state,
    allowing components like Cluster Autoscaler to develop logic around it.
  - It avoids breaking the established semantics of existing reasons and the clients that rely on them.

- **Cons:**
  - Introduces a new value to the API, which must be documented and maintained.

- **Alternatives Considered:**
  - PodReasonUnschedulable (`Unschedulable`): Reusing this reason would be somewhat incorrect,
    as the Pod has not failed a scheduling attempt (i.e., it was never evaluated against nodes).
    This could confuse users and break tooling that expects an `Unschedulable` status to be accompanied
    by scheduling failure events.
  - PodReasonSchedulingGated (`SchedulingGated`): This reason is explicitly tied to the Scheduling Gates feature.
    Reusing it would overload its meaning and break clients that rely on its specific semantic link to the spec.

### 4. How should outdated messages be handled?

**Proposed:** The lifecycle of the `NotReadyForScheduling` condition must be strictly managed
to reflect the Pod's current state.

1. On rejection: When a `PreEnqueue` plugin rejects a Pod, the `NotReadyForScheduling` condition is added
   or updated with the relevant message.
2. On success: When a Pod is re-evaluated and successfully passes all `PreEnqueue` plugins,
   the `NotReadyForScheduling` condition should be removed.

This ensures the condition is not stale. Once cleared, the Pod enters the scheduling queue in a clean `Pending` state,
waiting for the main scheduling cycle. This cycle will then result in a new state:
either the `PodScheduled` condition becoming True or the `Unschedulable` condition being added.

- **Pros:**
  - Guarantees the Pod's status is an accurate reflection of its current state.
  - Provides a clear, logical progression for users to follow.

- **Cons:**
  - A stale message could be displayed if a Pod is first rejected by a reporting plugin (Plugin A)
    and then subsequently blocked by a non-reporting plugin (Plugin B).
    The message from Plugin A would persist.
  
- **Alternatives Considered:**
  - Actively clearing the message: The scheduler could clear the condition if,
    on a subsequent check, the original rejecting plugin no longer rejects the pod.
    This would provide a more accurate real-time status but could cause confusion
    if the condition appears and disappears while the Pod remains `Pending` for another, unreported reason.
  - Not clearing the message: The scheduler could just leave the `NotReadyForScheduling` condition
    even if all `PreEnqueue` plugins passed. This might make the Pod harder to debug and track.

### User Stories

#### Diagnosing the Pods using DRA

As a DRA user, I deployed a Pod that requires a device managed by Dynamic Resource Allocation.
The corresponding resource claim is still pending (not being detected).
Instead of the Pod sitting in a generic `Pending` state, I want to immediately see a condition on the Pod with a message like
"Waiting for resource claim 'my-claim' to be present". This allows me to quickly diagnose
that the issue lies outside the scheduler's placement logic.
However, I *don't* want to see transient errors that resolve in seconds,
so the plugin's ability to opt out of reporting is crucial.

#### Using Deferred ResourceQuota Enforcement

As an application developer, I deploy a Pod that exceeds my namespace's resource quota, but has a scheduling gate.
The quota system is configured to defer enforcement, placing the Pod in a queue.
After I remove my Pod's scheduling gate, it still remains `Pending`.
With this KEP, the future Deferred ResourceQuota plugin would implement the new `PreEnqueue` interface
and return a `PreEnqueueResult`. This would update the Pod's status with a clear message,
explaining that the Pod is waiting for resource quota to become available in namespace.
This tells me exactly why my Pod isn't running and that I need to either wait for resources to be freed up or request a larger quota.

#### Using custom Gang Scheduling implementation

As a data scientist running a distributed training job, I submit a batch of Pods that must be scheduled together (a "gang").
The custom gang scheduling logic, using a `PreEnqueue` plugin, blocks all these Pods from entering
the queue until there are enough resources for all of them to pass the scheduling.
Currently, all the Pods would just be displayed as `Pending`. With this feature, each Pod's status would be
updated with some message indicating it is waiting on other members of the gang, providing clear insight into the job's state.

#### Cluster Autoscaler and resource provisioning

As a cluster operator, I rely on the Cluster Autoscaler (CA) to automatically provision new nodes when there are unschedulable pods,
discovered by the `PodReasonUnschedulable` condition. Currently, when a Pod is rejected by a `PreEnqueue` plugin like `DynamicResources`,
it remains `Pending` without any status. The CA typically ignores these pods because it doesn't recognize them as being blocked
by a lack of node resources. This means if a Pod requires a special resource (like a GPU from a DRA driver) and no nodes with that resource exist,
the CA will not scale up. With this KEP, the Pod will get a `NotReadyForScheduling` condition.
The Cluster Autoscaler can be updated to recognize this condition as a valid trigger for scale-up.
When it sees a Pod with this condition, it could correctly request a new node from a node pool that provides required resource.

### Risks and Mitigations

#### Stale or missing Pod status on patch failure

**Risk:** The asynchronous API call to patch the Pod's status could fail due to network issues,
API server unavailability, or other errors.

**Mitigation:** The scheduler's internal Pod state will not be affected by a failed API call.
The lack of a status condition is no worse than the current behavior.
Furthermore, any subsequent event that causes the Pod to be re-evaluated by the `PreEnqueue` plugins
will trigger a retry of the status patch. Exploring a more robust retry mechanism
within the asynchronous API calls feature itself would be a beneficial future enhancement.

#### Impact on scheduling throughput and latency

**Risk:** Adding logic to process and send status updates, even asynchronously,
could introduce overhead that impacts scheduler throughput,
especially in larger clusters where many Pods are rejected by `PreEnqueue` plugins almost simultaneously.
The API dispatcher in the scheduler could become a bottleneck.

**Mitigation:**
1. The reliance on the Asynchronous API calls feature is the primary mitigation,
   ensuring that the scheduler's main loop is not blocked waiting for the kube-apiserver response.

2. The impact on scheduling latency and throughput under heavy load will be measured using performance tests.

3. Allowing plugins to opt out of reporting a status (as detailed in the proposal above)
   prevents unnecessary API calls for short-lived or transient rejections.

4. Initially, the existing, highly optimized path for the SchedulingGates plugin (status set by kube-apiserver)
   can be left intact to avoid any performance regression in that scenario.

5. Future work: As the scheduler evolves, introducing batched status updates
   could further mitigate the impact of many simultaneous rejections.

## Design Details

The implementation will be primarily located within the kube-scheduler's internal scheduling queue.

### 1. PreEnqueue plugin interface change

A new struct, `PreEnqueueResult`, will be introduced. The `PreEnqueue` method signature in the `PreEnqueuePlugin` interface
will be changed to return this new struct alongside the existing `framework.Status`.

The new type and the modified interface will be defined in `pkg/scheduler/framework/interface.go`:
```go
// PreEnqueueResult holds information from the PreEnqueue plugin.
type PreEnqueueResult struct {
	// Message is the user-friendly message to be put in the Pod's status condition.
	// If empty, the plugin opts out from reporting the message to the Pod's condition.
	Message string
}
// ...
type PreEnqueuePlugin interface {
  // ...
	PreEnqueue(ctx context.Context, p *v1.Pod) (*PreEnqueueResult, *Status)
}
```

**In-Tree Plugin Changes:**

- **SchedulingGates:** Will always return an empty message, as its status will continue to be set by the kube-apiserver.
- **DefaultPreemption:** Will always return an empty message, as its state is transient
  and the `Unschedulable` status from the `PostFilter` stage is more descriptive.
- **DynamicResources:** Will be updated to return a descriptive message when a `ResourceClaim` is missing.
  It can also return a `nil` result to avoid reporting transient delays,
  for example when waiting for a `ResourceClaim` to become available in the scheduler's cache.

### 2. API changes

A new pod reason constant, `PodReasonNotReadyForScheduling`, will be added to `staging/src/k8s.io/api/core/v1/types.go`:

```go
// PodReasonNotReadyForScheduling reason in PodScheduled PodCondition means that the scheduler
// has not yet attempted to schedule the pod because it has failed one or more preliminary checks.
PodReasonNotReadyForScheduling = "NotReadyForScheduling"
```

This doesn't require other changes in the `core/v1` API or kube-apiserver.

### 3. Preventing redundant API calls

To prevent API server flooding when a pod is rejected for the same reason repeatedly, message caching will be introduced.

- A new field will be added to the `PodInfo` struct within the queue, for example, `lastPreEnqueueRejectionMessage`.
- Before dispatching a status patch, the scheduler will compare the new rejection message
  with the cached `lastPreEnqueueRejectionMessage`.
- The asynchronous call to the API server will only be dispatched if the status is new or different from the cached one.
  This deduplication is important for performance. Moreover, when the new message is empty or the whole `PreEnqueueResult` is `nil`,
  the call won't be send, as explained in the [how should outdated messages be handled](#4-how-should-outdated-messages-be-handled) section.

### 4. Asynchronous dispatch and state management

The status patch will be sent using the existing asynchronous API dispatcher.

- The scheduler's main goroutine will not be blocked. It will add the patch request to a queue processed by the dispatcher.
- As an alternative, redundancy checks outlined above could be moved to the API call's implementation.
- If the Pod is re-evaluated and passes the `PreEnqueue` stage shortly after a rejection,
  the a call to remove the `NotReadyForScheduling` condition will be enqueued.
  Such behavior can be optimized by the API dispatcher internals, preventing from sending any API call.

### 5. Logic flow

The core logic will be implemented in the `runPreEnqueuePlugins` function. The following code illustrates the flow:

```go
func (p *PriorityQueue) runPreEnqueuePlugins(ctx context.Context, pInfo *framework.QueuedPodInfo) {
	// ...
	for _, pl := range p.preEnqueuePluginMap[pod.Spec.SchedulerName] {
		// ...
		result, s := p.runPreEnqueuePlugin(ctx, logger, pl, pInfo, shouldRecordMetric)
		if !s.IsSuccess() {
			// --- Rejection Path ---
			if p.apiDispatcher == nil || !p.preEnqueuePodStatusEnabled {
				return // Gates are disabled, stop processing.
			}
			if result == nil || result.Message == "" {
				return // Plugin opted out of reporting.
			}
			if pInfo.lastPreEnqueueRejectionMessage == result.Message {
				return // Message is unchanged, de-duplicate.
			}
			// Enqueue the PodStatusPatch call with PodReasonNotReadyForScheduling and result.Message.
			// Emit an Event with the same reason and message.
			pInfo.lastPreEnqueueRejectionMessage = result.Message
			return
		}
	}
	// --- Success Path ---
	if p.apiDispatcher == nil || !p.preEnqueuePodStatusEnabled {
		return // Gates are disabled, stop processing.
	}
	if pInfo.lastPreEnqueueRejectionMessage == "" {
		return // No previous rejection message to clear.
	}
	// Enqueue a PodStatusPatch call to clear the NotReadyForScheduling condition.
	pInfo.lastPreEnqueueRejectionMessage = ""
}
```

#### Rejection path

When a Pod is being considered, the scheduler will iterate through all registered `PreEnqueue` plugins.
The following logic will be executed if any plugin rejects the Pod:

1. The scheduler will first perform several checks to ensure a status update should proceed:
   - It verifies that both the asynchronous API calls are enabled (API dispatcher is present)
     and the `SchedulerPreEnqueuePodStatus` feature gate is enabled.
   - It inspects the `PreEnqueueResult` returned by the plugin. If the `Message` field is empty,
     the plugin has opted out of reporting a status for this rejection.

2. The scheduler will compare the incoming `result.Message` with a cached message from the previous attempt,
   stored in the `PodInfo` object (e.g., `pInfo.lastPreEnqueueRejectionMessage`). If the new message is identical to the cached one,
   it means the Pod's status is already up-to-date, and the function can return to prevent redundant API calls.

3. If all checks pass, the scheduler proceeds with the update:
   - It constructs the condition to patch and enqueues it into the asynchronous API dispatcher.
   - An `Event` is emitted for the Pod with the reason `NotReadyForScheduling` and the rejection message.
   - The scheduler updates the `pInfo.lastPreEnqueueRejectionMessage` with the new message.

#### Success path

If the Pod successfully passes all `PreEnqueue` plugins:

1. The scheduler will check the `PodInfo` object to see if a `lastPreEnqueueRejectionMessage` is present from a previous failed attempt.

2. If a cached message exists (meaning the Pod was previously rejected but is now ready),
   the scheduler will construct the request to clear the condition and enqueue it into the asynchronous API dispatcher.
   This will remove the `PodScheduled` condition where the `reason` is `NotReadyForScheduling`.

3. The `pInfo.lastPreEnqueueRejectionMessage` field is cleared.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `pkg/scheduler/backend/queue`: `2025-08-27` - `91.3%`

##### Integration tests

- [`test/integration/scheduler`](https://github.com/kubernetes/kubernetes/tree/master/test/integration/scheduler)
  - Modify and add test cases covering the feature (with feature flag enabled and disabled).
- [`test/integration/scheduler_perf`](https://github.com/kubernetes/kubernetes/tree/master/test/integration/scheduler_perf) 
  - Verify performance with benchmarks that use the scheduling gates and other `PreEnqueue` plugins.

##### e2e tests

The integration tests listed above should cover all the scenarios,
so implementing e2e tests is a redundant effort.

### Graduation Criteria

This feature will be introduced in Beta, as it is an enhancement to existing scheduler-internal logic.

#### Beta

- Implement the feature behind a feature gate (`SchedulerPreEnqueuePodStatus`), enabled by default.
- All in-tree plugins that implement `PreEnqueue` are updated to the new interface.
- Implement all tests from the [Test Plan](#test-plan).
- Scheduling performance is verified using benchmarks to show no regression.

#### GA

- No negative feedback or critical bugs reported for at least one release.

### Upgrade / Downgrade Strategy

**Upgrade**

During the beta period, the feature gate `SchedulerPreEnqueuePodStatus` is enabled by default,
so users don't need to opt in. This is a purely in-memory feature for the kube-scheduler,
so no special actions are required outside the scheduler.

**Downgrade**

Users need to disable the feature gate. Any existing `NotReadyForScheduling` conditions on Pods
will remain until the Pod is scheduled or its status is otherwise updated.
`PreEnqueueResult` returned by the `PreEnqueue` plugins will be ignored.

### Version Skew Strategy

This feature is entirely self-contained within the kube-scheduler.
It does not interact with any other components in a way that would be affected by version skew.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `SchedulerPreEnqueuePodStatus`
  - Components depending on the feature gate: kube-scheduler

###### Does enabling the feature change any default behavior?

Yes. When enabled, Pods that are rejected by a `PreEnqueue` plugin can have their `.status.conditions` updated.
It does not change the core scheduling logic, only the reporting path.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate and restarting the kube-scheduler is sufficient to rollback.
The scheduler will stop adding or updating these conditions.
Existing conditions on Pods will not be cleaned up automatically
but will be overwritten by subsequent scheduling decisions.

###### What happens if we reenable the feature if it was previously rolled back?

The scheduler will resume adding and updating the `NotReadyForScheduling` condition on Pods
as they are rejected by `PreEnqueue` plugins.

###### Are there any tests for feature enablement/disablement?

Given it's a purely in-memory feature and enablement/disablement requires restarting the component 
(to change the value of the feature flag), having feature tests is enough.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout could fail if there is a bug in the new logic that causes the kube-scheduler to panic or perform poorly.
This would impact the scheduling of new pods but would not impact already running workloads (Pods).
A rollback is very safe as it simply disables the new functionality and has no impact on running workloads (Pods).

###### What specific metrics should inform a rollback?

- A significant increase in `scheduler_event_handling_duration_seconds`, meaning that event handling performance is affected.
- A significant increase in `scheduler_scheduling_attempt_duration_seconds` or `scheduler_pod_scheduling_sli_duration_seconds`,
  meaning that Pod scheduling performance is affected.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No. This feature is an in-memory feature of the scheduler
and thus calculations start from the beginning every time the scheduler is restarted.
So, just upgrading it and upgrade->downgrade->upgrade are both the same.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

An operator can query for Pods with the `NotReadyForScheduling` condition.

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event Reason: `NotReadyForScheduling`
- [x] API .status
  - Condition name: `PodScheduled` condition with a reason `NotReadyForScheduling`

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

In the default scheduler, we should see the throughput around 100-150 pods/s 
([ref](https://perf-dash.k8s.io/#/?jobname=gce-5000Nodes&metriccategoryname=Scheduler&metricname=LoadSchedulingThroughput&TestName=load)),
and this feature shouldn't bring any regression there.

Based on that `scheduler_schedule_attempts_total` shouldn't be less than 100 in a second,
if there are enough unscheduled pods within the cluster.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name:
    - `scheduler_schedule_attempts_total`
    - `scheduler_pod_scheduling_sli_duration_seconds`
    - `scheduler_event_handling_duration_seconds`
  - Components exposing the metric: kube-scheduler

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes.

- API call type: `PATCH` on `pods/status`.
- Originating component: kube-scheduler.
  - Estimated throughput: The rate of these calls is directly proportional
    to the rate at which new Pods are processed and rejected by PreEnqueue plugins.
    In a steady state with few new pods, the throughput will be low.
    In a high-throughput environment, it could be multiple calls per second.
    The deduplication and queueing mechanisms are designed to mitigate this.

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, it will add one entry to the `.status.conditions` array of a Pod object
when it is in a `NotReadyForScheduling` state. This increase is small and transient.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

There will be a marginal, likely negligible, increase in the time taken for the `PreEnqueue` extension point
due to the added logic for checking and queueing the status update.
The use of asynchronous calls ensures this does not impact the core scheduling latency of the Pod.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Memory: The kube-scheduler will have a small increase in memory usage
to cache the last-reported status for each Pod in the scheduling queue.

CPU: A minor increase in CPU usage to handle the new logic and dispatch the API call.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The API dispatcher will attempt to send the patch request.
If the kube-apiserver is unavailable, the request will fail.
This failure is logged, but it does not crash the scheduler or affect the scheduling.
The PreEnqueue rejection will just not be reported in the Pod status, leading to the former behavior.

###### What are other known failure modes?

Unknown

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

- 1st Sep 2025: The initial KEP is submitted.

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
