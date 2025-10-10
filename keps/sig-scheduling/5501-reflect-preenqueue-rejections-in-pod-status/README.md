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
    - [1. Reusing <code>framework.Status</code> and mandatory reporting](#1-reusing-frameworkstatus-and-mandatory-reporting)
    - [2. Handling the <code>SchedulingGated</code> conflict](#2-handling-the-schedulinggated-conflict)
    - [3. Delay configuration](#3-delay-configuration)
  - [User Stories](#user-stories)
    - [Diagnosing the Pods using DRA](#diagnosing-the-pods-using-dra)
    - [Using custom Gang Scheduling implementation](#using-custom-gang-scheduling-implementation)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Stale or missing Pod status on patch failure](#stale-or-missing-pod-status-on-patch-failure)
    - [Impact on scheduling throughput and latency](#impact-on-scheduling-throughput-and-latency)
- [Design Details](#design-details)
    - [1. Interface and plugins](#1-interface-and-plugins)
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
  - [Detailed comparison for five design alternatives](#detailed-comparison-for-five-design-alternatives)
    - [Primary evaluation criteria](#primary-evaluation-criteria)
    - [Comparison table](#comparison-table)
    - [Analysis and justification for the chosen design](#analysis-and-justification-for-the-chosen-design)
  - [Rejected alternative: detailed analysis of the &quot;Explicit + Immediate&quot; model](#rejected-alternative-detailed-analysis-of-the-explicit--immediate-model)
    - [1. How should plugins provide the status message?](#1-how-should-plugins-provide-the-status-message)
    - [2. Should all PreEnqueue plugins report the status?](#2-should-all-preenqueue-plugins-report-the-status)
    - [3. What reason should be used for the Pod condition?](#3-what-reason-should-be-used-for-the-pod-condition)
    - [4. How should outdated messages be handled?](#4-how-should-outdated-messages-be-handled)
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
that operate transparently to the user (where rejection is not directly derivable from a Pod spec),
as the reason for the delay is completely hidden.
While the Scheduling Gates feature does apply a status condition, it does so via the kube-apiserver
with its own reason (`SchedulingGated`).

This proposal aims to create a unified and consistent mechanism within the scheduler
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

When any `PreEnqueue` plugin rejects a Pod, the scheduler will capture the rejection message from the returned `framework.Status` object.
To handle transient rejections and reduce API server load, the scheduler will not act on this immediately.
Instead, it will use a **delayed dispatch mechanism** by modifying the existing API Dispatcher.

A pending status update will be cached for the Pod for a configurable duration (e.g., 5 seconds).
If the Pod is retried and the rejection reason is resolved or changes during this period,
the pending update will be cancelled or overwritten. If the delay period expires,
the scheduler will asynchronously patch the Pod's status with the last-captured rejection message,
using the new reason `NotReadyForScheduling`.

**Note:** This entire feature will be controlled by a new feature gate, `SchedulerPreEnqueuePodStatus`.
The enablement of this feature will be dependent on the `SchedulerAsyncAPICalls` feature gate being enabled,
as the asynchronous patching mechanism is critical to protect scheduler throughput.
Since the `SchedulerAsyncAPICalls` feature was disabled by default in v1.34,
successfully enabling the `SchedulerPreEnqueuePodStatus` feature in v1.35 will depend on re-enabling the `SchedulerAsyncAPICalls` feature.

#### 1. Reusing `framework.Status` and mandatory reporting

To avoid a breaking change and to be consistent with other extension points' rejections,
this design reuses the existing `framework.Status` object. It is expected that plugin developers will write high-quality,
user-friendly messages in this field, similar to the existing convention for Filter plugins.
**All rejections** from `PreEnqueue` plugins will be considered for reporting.
This ensures that the user always sees the most recent rejection reason, solving the "stale message" problem.

#### 2. Handling the `SchedulingGated` conflict

Other challenge is the conflict with the `SchedulingGated` feature, where the kube-apiserver also sets a status condition
with its dedicated `PodReasonSchedulingGated`. This will be resolved by **unifying the rejection message**.

1. The `SchedulingGates` plugin within the scheduler will be modified to return a `framework.Status`
   with a message identical to the one generated by the kube-apiserver (`"Scheduling is blocked due to non-empty scheduling gates"`).
2. The scheduler's de-duplication logic (`LastPreEnqueueRejectionMessage`)
   will be populated with the Pod's existing status conditions when it is first processed.
3. When a gated Pod is rejected, the scheduler will generate the unified message,
   compare it to the message already on the Pod object, see that they are identical, and **skip sending a redundant patch**.
 when processing pods with scheduling gates.

This creates a tight coupling between the two components but effectively de-duplicates the updates, doesn't require any change to the kube-apiserver,
and **preserves** the current scheduler performance when processing pods with scheduling gates.

#### 3. Delay configuration

The delay value will be **hardcoded to a default of 5 seconds**. This value is chosen to provide a reasonable balance
between handling common transient errors (e.g., cache propagation delays) and providing timely feedback for persistent issues.
Future consideration may be given to making this value configurable in the kube-scheduler's config
if a strong need for administrative tuning is identified.

### User Stories

#### Diagnosing the Pods using DRA

As a DRA user, I deployed a Pod that requires a device managed by Dynamic Resource Allocation.
The corresponding resource claim is still pending (not being detected).
Instead of the Pod sitting in a generic `Pending` state, I want to see a condition on the Pod with a message like
"Waiting for resource claim 'my-claim' to be present". This allows me to quickly diagnose
that the issue lies outside the scheduler's placement logic.
I also don't want to see transient errors that resolve in seconds. 
The delayed dispatch mechanism is crucial for filtering out this noise.

#### Using custom Gang Scheduling implementation

As a data scientist running a distributed training job, I submit a batch of Pods that must be scheduled together (a "gang").
The custom gang scheduling logic, using a `PreEnqueue` plugin, blocks all these Pods from entering
the queue until there are enough resources for all of them to pass the scheduling.
Currently, all the Pods would just be displayed as `Pending`. With this feature, each Pod's status would be
updated with some message indicating it is waiting on other members of the gang, providing clear insight into the job's state.

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

3. The delaying mechanism is expected to reduce the load on kube-apiserver by canceling
   or overwriting the pending API calls. In addition, plugin developers will be encouraged to create error messages
   that are as consistent as possible to reduce the number of updates required for a single Pod.

4. The existing, highly optimized path for the SchedulingGates plugin (status set by kube-apiserver)
   is left intact to avoid any performance regression in that scenario.

5. Future work: As the scheduler evolves, introducing batched status updates
   could further mitigate the impact of many simultaneous rejections.

## Design Details

The implementation will be primarily located within the kube-scheduler's internal scheduling queue.

#### 1. Interface and plugins

This design requires **no breaking changes** to the `PreEnqueuePlugin` interface.
The existing signature, which returns a `*framework.Status`, will be used.

**In-Tree Plugin Changes:**

- **SchedulingGates:** Message will be made consistent with the kube-apiserver.
- **DefaultPreemption** and **DynamicResources** messages will be improved to make sure they are high-quality and user-friendly

### 2. API changes

A new pod reason constant, `PodReasonNotReadyForScheduling`, will be added to `staging/src/k8s.io/api/core/v1/types.go`:

```go
// PodReasonNotReadyForScheduling reason in PodScheduled PodCondition means that the scheduler
// has not yet attempted to schedule the pod because it has failed one or more preliminary checks.
PodReasonNotReadyForScheduling = "NotReadyForScheduling"
```

**Note:** This doesn't require other changes in the `core/v1` API or kube-apiserver.

### 3. Preventing redundant API calls

To prevent API server flooding when a pod is rejected for the same reason repeatedly, message caching will be introduced.

- A new field will be added to the `framework.QueuedPodInfo` struct within the queue, for example, `LastPreEnqueueRejectionMessage`.
- Before dispatching a status patch, the scheduler will compare the new rejection message
  with the cached `LastPreEnqueueRejectionMessage`.
- The asynchronous call to the API server will only be enqueued if the status is new or different from the cached one.
  This deduplication is important for performance.

### 4. Asynchronous dispatch and state management

The status patch will be sent using the existing asynchronous API dispatcher.

- The scheduler's main goroutine will not be blocked.
- **The API dispatcher will be modified to handle delayed API calls.** This will be a new capability of the dispatcher,
  and existing non-delayed calls will be unaffected.
- If the Pod is re-evaluated and passes the `PreEnqueue` stage shortly after a rejection,
  a call to remove the `NotReadyForScheduling` condition will be enqueued.
  This "clear" operation will also be delayed to prevent  status flickering if the Pod is immediately rejected for another reason.
  The API dispatcher can optimize this by cancelling the pending messages for the same Pod.

### 5. Logic flow

The core logic will be implemented in the `runPreEnqueuePlugins` function. The following code illustrates the flow:

```go
func (p *PriorityQueue) runPreEnqueuePlugins(ctx context.Context, pInfo *framework.QueuedPodInfo) {
	// ...
	for _, pl := range p.preEnqueuePluginMap[pod.Spec.SchedulerName] {
		// ...
		s := p.runPreEnqueuePlugin(ctx, logger, pl, pInfo, shouldRecordMetric)
		if !s.IsSuccess() {
			// --- Rejection Path ---
			if p.apiDispatcher == nil || !p.preEnqueuePodStatusEnabled {
				return // Gates are disabled, stop processing.
			}
			rejectionMessage := s.Message()
			
			// Set the last known message from the Pod object on first sight.
			if pInfo.lastPreEnqueueRejectionMessage == "" {
				pInfo.lastPreEnqueueRejectionMessage = getMessageFromPod(pInfo.Pod)
			}
			if pInfo.lastPreEnqueueRejectionMessage == rejectionMessage {
				return // Message is unchanged, de-duplicate.
			}
			
			// Enqueue a *delayed* PodStatusPatch with PodReasonNotReadyForScheduling and the new message.
			// Emit an Event with the same reason and message.
			pInfo.lastPreEnqueueRejectionMessage = rejectionMessage
			return
		}
	}
	// --- Success Path ---
	if p.apiDispatcher == nil || !p.preEnqueuePodStatusEnabled {
		return // Gates are disabled, stop processing.
	}
	// If the pod had a status condition before, enqueue a PodStatusPatch to clear it.
	if pInfo.lastPreEnqueueRejectionMessage == "" {
    return
	}
	// Enqueue a *delayed* PodStatusPatch call to clear the NotReadyForScheduling condition.
	pInfo.lastPreEnqueueRejectionMessage = ""
}
```

#### Rejection path

When a Pod is being considered, the scheduler will iterate through all registered `PreEnqueue` plugins.
The following logic will be executed if any plugin rejects the Pod:

1. The scheduler will first verify that both the asynchronous API dispatcher is present
   and the `SchedulerPreEnqueuePodStatus` feature gate is enabled.

2. The scheduler will compare the incoming `fwk.Status` message with a cached message from the previous attempt
   (`pInfo.LastPreEnqueueRejectionMessage`). If the messages are identical, the function returns to prevent redundant work.

3. If the checks pass, the scheduler proceeds:
   - It immediately updates its internal state by setting `pInfo.LastPreEnqueueRejectionMessage` to the new message.
   - It constructs the condition to patch and enqueues it with a delay into the asynchronous API dispatcher.
   - An `Event` is emitted for the Pod with the reason `NotReadyForScheduling` and the rejection message.

#### Success path

If the Pod successfully passes all `PreEnqueue` plugins:

1. The scheduler will check if a `LastPreEnqueueRejectionMessage` is present from a previous failed attempt.

2. If a cached message exists (meaning the Pod was previously rejected but is now ready),
   the scheduler immediately clears its internal state by setting `pInfo.LastPreEnqueueRejectionMessage` to `""`.
   It then constructs a request to clear the condition and enqueues that request with a delay into the asynchronous API dispatcher.

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
    The deduplication, queueing and delaying mechanisms are designed to mitigate this.

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

### Detailed comparison for five design alternatives

To choose the best design, five distinct models were evaluated, mapping out the full design space.
The two primary distinction points were the Interface (how a plugin provides information)
and the Dispatch mechanism (how the scheduler acts on that information),
with a sub-variant for the Implicit Interface model (with or without an opt-out).

The five models were:
1. **Explicit + Immediate:** A new `PreEnqueueResult` is introduced.
  Plugins explicitly return a user message. The update is sent immediately.
1. **Explicit + Delayed (Hybrid model):** A new `PreEnqueueResult` is introduced.
  The scheduler waits for a delay before sending the last captured *user-facing* message.
1. **Implicit + Delayed (with Opt-out):** The existing `framework.Status` is used.
  A plugin can opt out by returning an empty `Status` message.
  Otherwise, the scheduler waits for a delay before sending the last captured internal message.
1. **Implicit + Delayed (no Opt-out, KEP Proposal):** The existing `framework.Status` is used.
  *Every* rejection is reported after a delay, unless it's overwritten.
1. **Implicit + Immediate (Baseline):** The existing `framework.Status` is used.
  The scheduler immediately sends the internal message as a status update.

The following sections define the evaluation criteria, present the comparison,
and provide the justification for selecting Model 4 (Implicit + Delayed, no Opt-out) as the chosen approach.

#### Primary evaluation criteria

To provide a structured comparison, the following key design goals and constraints were considered for each alternative:

**1. User and developer experience**

This is the primary motivation for the KEP. It evaluates the impact on both the end-users debugging their Pods and the developers building custom scheduler plugins.

- **User message quality:** The clarity and actionability of the information presented in the Pod's status.
- **Stale message risk:** The danger of the system showing outdated, actually incorrect information.
- **Handles transient issues:** The ability to intelligently reduce status update noise from short-lived, self-resolving rejections.
- **Plugin developer control:** The ability given to the plugin developers to control the reporting behavior.
- **Interface consistency:** How well the API fits with existing, established patterns in the scheduler framework.
- **Breaking change:** Whether the change requires all (PreEnqueue) plugin developers to update their code.

**2. System health and observability**

This evaluates the impact on the cluster as a whole, its performance, and the ability of administrators to monitor its behavior.

- **`kube-apiserver` load:** The performance impact of new API calls generated by the feature.
- **Logging quality:** Ensuring the quality and completeness of the scheduler's internal logs are not weakened by the new feature.
- **`SchedulingGated` conflict:** Checking whether the model would require non-trivial changes to keep the current behavior of reporting SchedulingGates rejection.
  Or migrating those to the new framework.

**3. Implementation and operational cost**

This evaluates the engineering overhead of building and maintaining the feature.

- **Implementation complexity:** The engineering cost to build and maintain the feature.
- **Configuration complexity:** The complexity of correctly configuring the feature.

#### Comparison table

| **Feature**                    | **1. Explicit + Immediate**          | **2. Explicit + Delayed (Hybrid)**         | **3. Implicit + Delayed (Opt-out)**          | **4. Implicit + Delayed (No opt-out, KEP)**  | **5. Implicit + Immediate (Baseline)** |
| :----------------------------- | :----------------------------------- | :----------------------------------------- | :------------------------------------------- | :------------------------------------------- | :------------------------------------- |
| **User message quality**       | 🟢 **High** (Dedicated field)        | 🟢 **High** (Dedicated field)               | 🟡 **Variable** (Reuses `fwk.Status`)        | 🟡 **Variable** (Reuses `fwk.Status`)        | 🟡 **Variable** (Reuses `fwk.Status`)   |
| **Logging quality**            | 🟢 **High** (Decoupled from status)  | 🟢 **High** (Decoupled from status)         | 🔴 **Low** (Opt-out forces empty log reason) | 🟡 **Variable** (Tied to user message)       | 🟡 **Variable** (Tied to user message)  |
| **Interface consistency**      | 🟡 **Medium** (New return pattern)   | 🟡 **Medium** (New return pattern)          | 🟢 **High** (Existing pattern)               | 🟢 **High** (Existing pattern)               | 🟢 **High** (Existing pattern)          |
| **Handles transient issues**   | 🟡 **Good** (Opt-out only)           | 🟢 **Excellent** (Delay + explicit opt-out) | 🟢 **Excellent** (Delay + implicit opt-out)  | 🟡 **Good** (Delay only)                     | 🔴 **Poor** (Noisy)                     |
| **Stale message risk**         | 🔴 **High** (Can be stale)           | 🟡 **Medium** (Can be stale or delayed)     | 🟡 **Medium** (Can be stale or delayed)      | 🟢 **Low** (Always latest, but delayed)      | 🟢 **Lowest** (Always latest)           |
| **Plugin developer control**   | 🟢 **High** (If, what, when)         | 🟡 **Partial** (If, what)                   | 🟡 **Partial** (If, what)                    | 🔴 **Minimal** (What)                        | 🟡 **Partial** (What and when)          |
| **`SchedulingGates` conflict** | 🟢 **Solved** (Via explicit opt-out) | 🟢 **Solved** (Via explicit opt-out)        | 🟢 **Solved** (Via implicit opt-out)         | 🔴 **Requires addressing**                   | 🔴 **Requires addressing**              |
| **Configuration complexity**   | 🟢 **None**                          | 🔴 **High** (Requires delay value)          | 🔴 **High** (Requires delay value)           | 🔴 **High** (Requires delay value)           | 🟢 **None**                             |
| **Implementation complexity**  | 🟡 **Medium** (Framework changes)    | 🔴 **Highest** (Framework + API Dispatcher) | 🔴 **High** (API Dispatcher)                 | 🔴 **High** (API Dispatcher)                 | 🟢 **Low** (Queue change only)          |
| **`kube-apiserver` load**      | 🟡 **Medium** (Skip transient)       | 🟢 **Lowest** (Skip transient, merging)     | 🟢 **Lowest** (Skip transient, merging)      | 🟡 **Medium** (Merging)                      | 🔴 **High** (Always sends new statuses) |
| **Breaking change**            | 🔴 **Yes**                           | 🔴 **Yes**                                  | 🟢 **No**                                    | 🟢 **No**                                    | 🟢 **No**                               |

#### Analysis and justification for the chosen design

After a rigorous evaluation of the five models, **Model 4 (Implicit + Delayed, No opt-out)** was selected as the final design.
This decision was based on a pragmatic assessment of trade-offs, prioritizing the stability and a reliable user experience.

The reasoning is as follows:

1. **Rejection of explicit models (1 & 2):**
   The `Explicit` models, while offering better user message and logging quality, were rejected primarily due to the introduction of a new message field apart from the actual `fwk.Status`.

2.  **Rejection of the simplest implicit models (5 & 3):**
    * **Model 5 (Implicit + Immediate)** was rejected as too naive for the use cases.
      Its inability to handle transient issues would lead to excessive `kube-apiserver` load and a noisy user experience.
    * **Model 3 (Implicit + Delayed with opt-out)** was a strong proposal. However, the proposed solution for the `SchedulingGated` conflict
      is a more direct approach that*makes an explicit opt-out mechanism unnecessary. Adding an opt-out feature that also degrades logging quality
      (by forcing an empty `fwk.Status` message) was assessed as an unnecessary complication.

3. **Selection of Model 4 (Implicit + Delayed, No opt-out):**
   * **Its strengths:** It successfully avoids a breaking change and resuses the well-known `fwk.Status`.
     Crucially, it provides a reliable solution to the stale message problem, which would harm the user experience.
     It also handles transient issues reasonably well via the delay mechanism.
   * **Its accepted trade-offs:** This design knowingly accepts certain compromises. The user message quality is tied to developer discipline for internal log messages.
     The delay mechanism introduces configuration complexity.

### Rejected alternative: detailed analysis of the "Explicit + Immediate" model

For completeness, this section provides a detailed breakdown of the original "Explicit + Immediate" proposal (Model 1).

#### 1. How should plugins provide the status message?

**Proposed:** Adjust the `PreEnqueue` plugin interface to return an additional,
optional, user-facing message alongside the `Status` object.
This will introduce a **breaking change** to all plugins that implement the `PreEnqueue` extension point.

This flexibility is important for plugins that wish to avoid reporting transient rejections.
For example, the `DynamicResources` plugin might observe a rejection because the scheduler processes a Pod faster
than a `ResourceClaim` becomes visible through the watcher (see a [comment](https://issues.k8s.io/129698#issuecomment-2614991955)).
In such cases where the condition is expected to resolve in seconds, populating the status would be inappropriate noise.

- **Pros:**
  - Gives plugin authors direct control over the user-facing message and the decision to report it.
  - Decouples internal rejection reasons from user-facing messages.
  - Allows plugins to opt out of reporting simply by returning an empty message, a clear and unambiguous signal.

- **Cons:**
  - Requires a breaking change to the `PreEnqueue` plugin interface.
  - Plugin developers are used to to the framework's standard pattern of returning a single `Status` object to signal an outcome.
    The new `Message` could initially cause confusion about the relationship between the two return values,
    which must be addressed with clear documentation and in-tree examples.

- **Alternatives Considered:**
  - Use the raw status message: This is simpler as it requires no interface changes.
    However, these messages are often not written well for end-users.
    Additionally, to allow plugins to opt out, an implicit mechanism would be needed,
    such as returning a `Status` with an empty `reasons` field. This approach has two significant flaws:
    - It would force plugins to omit the debugging information from the scheduler's logs
      just to control a user-facing feature, degrading observability. Preserving detailed logging is essential.
    - Relying on implicit "magic" behavior makes the framework harder to use and less straightforward for plugin developers.
      The proposed explicit `PreEnqueueResult` provides a much clearer and more self-documenting API.

#### 2. Should all PreEnqueue plugins report the status?

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
  - Mandatory reporting with a delay (cooldown period): This approach attempts to reduce API churn by waiting for a brief,
    configurable delay before reporting a rejection. While better than immediate mandatory reporting, this has some flaws:
    - It removes control from the plugins, which are in the best position to know whether a rejection is truly transient
      or a persistent issue that needs a status update.
    - A single, global delay value is difficult to set correctly for all plugins and all possible cluster states.
    - It doesn't guarantee relevance. A Pod's PreEnqueue retry can be significantly delayed due to a long scheduling queue or other factors,
      making the delayed message just as irrelevant as an immediate one. While a delay could be a beneficial additional mechanism
      to reduce update bursts in the future, it's not a suitable main control mechanism.

#### 3. What reason should be used for the Pod condition?

**Proposed:** Introduce a new PodCondition reason: `NotReadyForScheduling`.
The status reporting for Scheduling Gates will be left intact (i.e., handled by the kube-apiserver with the `SchedulingGated` reason)
to avoid any disruption to existing behavior.

- **Pros:**
  - It clearly communicates the stage of rejection.
  - It doesn't require any changes in the Cluster Autoscaler or other components to preserve their current behavior.
    Such components can later opt-in to recognize the new reason and develop logic around it, allowing for safer, optional enhancement.
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

#### 4. How should outdated messages be handled?

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
    The message from Plugin A would persist. However, this behavior is consistent with the existing `Unschedulable` condition,
    which can also become stale between scheduling retries. This is a known and accepted trade-off.
  
- **Alternatives Considered:**
  - Actively clearing the message: The scheduler could clear the condition if,
    on a subsequent check, the original rejecting plugin no longer rejects the pod.
    This would provide a more accurate real-time status but could cause confusion
    if the condition appears and disappears while the Pod remains `Pending` for another, unreported reason.
  - Update with a generic message: In the case where a Pod is blocked by a non-reporting plugin,
    the scheduler could generate a generic message (e.g., "Pod is blocked on PreEnqueue by plugin: Plugin B").
    This has a two flaws:
    - While it identifies the blocking plugin, it doesn't explain why the Pod was blocked, which is the essential information for debugging.
      A generic message isn't a significant improvement over a stale (but potentially actionable) message.
    - The status patch is a best-effort, asynchronous API call that can fail. Since there is no guaranteed retry mechanism,
      the system cannot promise that the status is perfectly up-to-date. Adding complexity to generate generic messages
      might not be a sufficient reason.
  - Not clearing the message: The scheduler could just leave the `NotReadyForScheduling` condition
    even if all `PreEnqueue` plugins passed. This might make the Pod harder to debug and track.
