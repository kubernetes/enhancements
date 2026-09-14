# KEP-3386: Kubelet Evented PLEG for Better Performance

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1 (Optional)](#story-1-optional)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
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

Items marked with (R) are required prior to targeting a milestone or release.

- [x] (R) Enhancement issue in the release milestone links to this KEP.
- [x] (R) KEP approvers have approved the KEP as `implementable`.
- [x] (R) Design details are documented.
- [ ] (R) The test plan covers stream failure, recovery, and concurrent relist requests.
- [ ] (R) Graduation criteria are satisfied.
- [ ] (R) Production Readiness Review is completed and approved.
- [x] Implementation history is current.
- [ ] User-facing documentation is published, if required.

## Summary

> **Important:** This KEP has been substantially refactored. Before the refactor, Evented PLEG acted as a second source of kubelet state: it used CRI event payloads to update the pod cache and emit lifecycle events, and changed Generic PLEG polling and fallback behavior. After the refactor, Evented PLEG only uses stopped container events to request a targeted relist. Generic PLEG keeps its normal global relist period and remains the only component that reads runtime state, updates the pod cache, and emits lifecycle events.

When `EventedPLEG` is enabled, kubelet watches the CRI `GetContainerEvents` stream. Each valid `CONTAINER_STOPPED_EVENT` asks `GenericPLEG` to relist the affected pod immediately.

`GenericPLEG` remains the only source of pod lifecycle events. It continues to relist all pods at the normal interval, reads the current runtime state, updates the kubelet pod cache, and emits `PodLifecycleEvent` objects. Evented PLEG does not update kubelet state directly.

The CRI event stream is an optimization, not a source of truth. If the stream is unavailable, delayed, or delivers duplicate or stale events, kubelet may lose the latency improvement, but Generic PLEG continues to reconcile runtime state.

## Motivation

Kubelet needs to detect container state changes that it did not initiate, such as a normal exit, a failure, or an OOM kill. `GenericPLEG` detects these changes by polling the runtime. This level-driven reconciliation is reliable, but pod processing does not begin until the next relist.

This KEP adds the server-streaming `GetContainerEvents` RPC to CRI. A stopped container event identifies the affected pod and triggers an immediate relist of that pod. The relist reads the current runtime state and follows the same reconciliation path as a global relist.

Earlier versions of this KEP treated Evented PLEG as a second source of kubelet state. That design wrote directly to the cache, emitted lifecycle events from the stream, increased the Generic PLEG relist period, and switched Generic PLEG configuration when the stream failed. This revision removes those behaviors so that kubelet state does not depend on an edge-triggered stream.

### Goals

- Reduce the latency between a CRI container termination and kubelet pod reconciliation.
- Preserve one authoritative path for runtime observation, cache mutation, and pod lifecycle event generation.
- Preserve Generic PLEG's normal global relist behavior whether Evented PLEG is enabled, disconnected, or unsupported by the runtime.
- Recover automatically from arbitrary-duration stream interruptions without requiring a kubelet restart.
- Coalesce relist requests received while a pod's relists are suspended, and bound the remaining on-demand queue.
- Make loss of the fast path observable without making it a kubelet health failure.

### Non-Goals

- Reducing the Generic PLEG global relist frequency or promising lower steady-state polling CPU usage. Generic PLEG continues at its normal period.
- Replacing polling or making CRI events a durable, ordered, exactly-once log.
- Updating the pod cache, running-pod/container metrics, or `PodLifecycleEvent` objects directly from a CRI event payload.
- Accelerating `CONTAINER_CREATED_EVENT`, `CONTAINER_STARTED_EVENT`, or `CONTAINER_DELETED_EVENT`. Those events are observed for stream metrics but do not request a relist in this KEP.
- Reconstructing every intermediate container transition that occurred while both the stream and runtime's queryable state were unavailable. Recovery converges to current runtime state and retains the same transient-state limitations as Generic PLEG with the feature disabled.
- Addressing container image relisting.

## Proposal

Evented PLEG watches the CRI event stream. For each valid stopped container event, it calls `GenericPLEG.RequestRelist(podUID)`.

Generic PLEG continues to run with its normal relist period and health threshold. It remains the only component that reads runtime state, updates PLEG records and the pod cache, and emits `PodLifecycleEvent` objects. Evented PLEG does not use status fields from the event as kubelet state. It only uses the event type and pod UID to request a fresh read from the runtime.

CRI events are best effort and may be delayed, duplicated, reordered, or lost. Generic PLEG continues its periodic relist regardless of stream state, so event delivery does not affect correctness.

### User Stories (Optional)

#### Story 1 (Optional)

As an operator running distributed AI/ML training or inference workloads, I want kubelet to detect a failed worker container quickly so pod reconciliation and workload recovery can begin without waiting for the next global relist.

### Notes/Constraints/Caveats (Optional)

The CRI event stream is a best-effort latency hint. It has no replay or resume mechanism, and kubelet does not assume that events are ordered or delivered exactly once. A runtime needs to populate `pod_sandbox_status.metadata.uid` on stopped container events for kubelet to request a targeted relist. An older or incompatible runtime may return `Unimplemented`, close the stream, or omit the required metadata; in each case, only the fast path is unavailable and Generic PLEG continues normally.

### Risks and Mitigations

- **Termination storms can create excessive per-pod relists.** Requests received while a pod's relists are suspended are coalesced into one pending relist. Outside that window, the on-demand queue remains bounded. One Generic PLEG dispatcher serializes global and per-pod relists and gives the periodic global relist priority.
- **A stream interruption can lose events.** Generic PLEG continues its normal global relist throughout the interruption. A successful global relist after reconnection is the recovery boundary.
- **A runtime can deliver stale or duplicate events after reconnection.** Each event only requests a fresh read. It cannot overwrite the cache with event payload data, although duplicate events may cause redundant relist requests.
- **The runtime may not implement the stream.** Reconnection is rate-limited; Generic PLEG remains active and healthy. The feature's fast path remains unavailable until a compatible runtime is installed or the gate is disabled.
- **A full on-demand queue can drop latency hints.** Queue capacity is bounded, and dropped requests are observable. The next global relist still reconciles the current state.

## Design Details

This KEP adds the following API to CRI:

```protobuf
// GetContainerEvents gets container events from the CRI runtime
rpc GetContainerEvents(GetEventsRequest) returns (stream ContainerEventResponse) {}

message GetEventsRequest {}

message ContainerEventResponse {
    // ID of the container
    string container_id = 1;

    // Type of the container event
    ContainerEventType container_event_type = 2;

    // Creation timestamp of this event
    int64 created_at = 3;

    // Sandbox status
    PodSandboxStatus pod_sandbox_status = 4;

    // Container statuses
    repeated ContainerStatus containers_statuses = 5;
}

enum ContainerEventType {
    // Container created
    CONTAINER_CREATED_EVENT = 0;

    // Container started
    CONTAINER_STARTED_EVENT = 1;

    // Container stopped
    CONTAINER_STOPPED_EVENT = 2;

    // Container deleted
    CONTAINER_DELETED_EVENT = 3;
}
```

The earlier design proposed `PodSandboxStatusRequest.includeContainers` so that Generic PLEG could request container statuses and a timestamp through `PodSandboxStatus`. That field was never added to the CRI API and is not part of this design. Generic PLEG continues to obtain current state through its normal CRI queries, so no replacement request field is needed.

The event path and the existing reconciliation path interact as follows:

```text
CRI GetContainerEvents stream
        |
        | CONTAINER_STOPPED_EVENT + pod UID
        v
EventedPLEG watcher ---- RequestRelist(pod UID) ----+
                                                     |
successful SyncPod ---- RequestRelist(pod UID) ------+--> bounded on-demand queue
                                                     |    in GenericPLEG
normal global timer ---------------------------------+
                                                          |
                                                          v
                                              query current CRI state
                                                          |
                                                          v
                                            reconcile GenericPLEG records
                                                          |
                                      +-------------------+------------------+
                                      v                                      v
                              update pod cache                   emit PodLifecycleEvent
```

Neither request source performs the relist. Both enqueue work for Generic PLEG, so global and on-demand relists use the same state comparison, cache update, event filtering, error handling, and pod reinspection logic.

Kubelet always constructs `GenericPLEG` with the normal relist period and health threshold. The existing `PLEG` health check continues to report Generic PLEG health. When the `EventedPLEG` feature gate is enabled, kubelet also starts the stream watcher. Evented PLEG has no separate lifecycle event channel, cache reference, health check, relist period, or fallback mode. Disabling the feature gate and restarting kubelet stops the watcher without changing Generic PLEG.

For each `ContainerEventResponse`, Evented PLEG validates that `pod_sandbox_status` and its metadata are present and that `pod_sandbox_status.metadata.uid` is non-empty. Without a pod UID, a targeted relist cannot be requested, so the event is logged and ignored. The watcher records event creation-to-receipt latency, ignores event types other than `CONTAINER_STOPPED_EVENT`, and calls `RequestRelist` for every stopped event regardless of exit code, reason, or whether a `ContainerStatus` is attached.

Evented PLEG does not distinguish between a clean exit, a failure, and an OOM kill. It also does not treat `containers_statuses` or `pod_sandbox_status` as a complete snapshot. Generic PLEG determines the current state through its normal `GetPod` and `GetPodStatus` calls. If the pod is deleted before the targeted query, the request is a no-op and the next global relist removes any remaining record. If Generic PLEG has already observed the stopped container, the targeted relist finds no state change and does not emit another lifecycle event.

Generic PLEG uses a bounded queue for per-pod relist requests. One dispatcher handles shutdown, due global relists, and targeted pod relists, in that order. Global and targeted relists use the same synchronization boundary to prevent races in `podRecords`, cache updates, and lifecycle event generation. Each request records its enqueue time. Generic PLEG may skip a request if a newer global relist has already covered it. When the queue is full, Generic PLEG may drop a request; this only delays detection until the next global relist.

Evented PLEG and the post-`SyncPod` path both call `RequestRelist`. Requests received while relists for a pod are suspended are coalesced into one pending relist. Outside suspension, requests enter the bounded queue normally and are not deduplicated by pod UID. Global relists retain priority.

Some stopped events result from runtime calls made by `SyncPod`, for example when kubelet restarts a container after a liveness probe fails. Relisting on each event would add unnecessary work and could observe an intermediate state. Kubelet therefore suspends relists for a pod while `SyncPod` is running for that pod. Requests received during the sync are retained and coalesced. When `SyncPod` returns, including after an error or cancellation, kubelet releases the hold and queues one relist if necessary. A global relist may continue processing other pods, but does not publish an intermediate state for the suspended pod. Lock ordering between the pod worker and Generic PLEG needs to avoid deadlocks.

`GetContainerEvents` does not support replay or resumption, so events may be lost while the stream is disconnected. Generic PLEG continues its normal global relist and recovers the current runtime state. Event payloads are never replayed into the kubelet cache. The watcher retries for the lifetime of the kubelet: the first reconnect attempt is immediate, later attempts use exponential backoff with jitter capped at 60 seconds, and retries continue at the cap instead of permanently disabling the fast path. EOF and a stream that closes without an error are handled like other connection failures. A connection needs to remain healthy for 60 seconds before the backoff is reset, which avoids a tight loop when the runtime is flapping. Kubelet shutdown cancels the active RPC and any pending retry timer.

If the runtime returns `Unimplemented`, the watcher continues retrying at the maximum interval. This allows a runtime upgrade to make the stream available without a kubelet restart. Repeated errors are logged at low verbosity or are rate-limited. A delayed event received after reconnection only triggers a fresh runtime read. If Generic PLEG has already observed the stop, the relist emits no lifecycle event. The `created_at` field is used only to measure latency; it does not affect event ordering or cache updates.

Existing metrics retained by the implementation include:

- `kubelet_evented_pleg_connection_error_count`
- `kubelet_evented_pleg_connection_success_count`
- `kubelet_evented_pleg_connection_latency_seconds`
- `kubelet_pleg_pod_relist_duration_seconds`
- the existing Generic PLEG relist interval, duration, last-seen, and discarded event metrics

The implementation also needs to expose stream connection state, reconnect attempts, and the number of relist requests queued, coalesced, or dropped. Metric names and stability levels will be reviewed with SIG Instrumentation. Pod UID, container ID, error text, and runtime endpoint need to be excluded from metric labels.

### Test Plan

- [x] We understand that the owners of the affected components may require updates to existing tests before this enhancement is implemented.

##### Unit tests

- `k8s.io/kubernetes/pkg/kubelet/pleg`: `2026-09-01` - `84.8%`

##### Integration tests

No test under `test/integration` is currently planned because the behavior is local to kubelet, Generic PLEG, and the CRI runtime. Cross-component behavior is covered by kubelet unit tests and node e2e tests with a real CRI runtime.

##### e2e tests

Existing Evented PLEG e2e jobs:

- [`pull-kubernetes-e2e-kind-evented-pleg`](https://testgrid.k8s.io/presubmits-kubernetes-nonblocking#pull-kubernetes-e2e-kind-evented-pleg)
- [`pull-node-crio-evented-pleg`](https://testgrid.k8s.io/sig-node-cri-o#pull-node-crio-evented-pleg)
- [`pull-kubernetes-node-containerd-evented-pleg-e2e`](https://testgrid.k8s.io/sig-node-presubmits#pr-containerd-evented-pleg-gce-e2e)

Existing general-purpose e2e tests will also be run with an Evented PLEG configuration.

### Graduation Criteria

#### Alpha

- The feature is disabled by default and guarded by `EventedPLEG`.
- Generic PLEG runs at its normal global period and is the sole cache and lifecycle-event writer.
- Only stopped events request targeted relists.
- Existing node e2e pod-lifecycle tests pass.

#### Beta

- The watcher reconnects indefinitely with capped, jittered backoff and recovers after a runtime restart without restarting kubelet.
- Per-pod relist suspension and request merging prevent Evented PLEG from relisting a pod in the middle of `SyncPod`.

#### GA

- Beta criteria have remained satisfied for at least two releases.
- Upgrade, downgrade, runtime restart, and feature disablement are continuously tested.
- Operational documentation is published and SIG Node agrees that field experience justifies graduation.

### Upgrade / Downgrade Strategy

The runtime may be upgraded before or after kubelet. Generic PLEG remains active in either order. If kubelet is upgraded before the runtime supports `GetContainerEvents`, the watcher retries with backoff until the RPC becomes available.

Disabling `EventedPLEG` requires a kubelet restart. The restarted kubelet runs only Generic PLEG at the same normal period it used while the feature was enabled. There is no persisted Evented PLEG state to migrate or roll back.

### Version Skew Strategy

Version skew only applies between kubelet and its local CRI runtime. A new kubelet with an old runtime continues using Generic PLEG and rate-limits stream retries. An old kubelet ignores `GetContainerEvents`. This feature does not introduce Kubernetes API or control-plane version skew.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `EventedPLEG`
  - Components depending on the feature gate: kubelet

A kubelet restart is required after changing the gate.

###### Does enabling the feature change any default behavior?

It adds a low-latency targeted relist after container-stop notifications. It does not change Generic PLEG's global relist period, health check, cache ownership, or lifecycle-event ownership.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Restart kubelet with `EventedPLEG=false`. Generic PLEG continues with the same normal configuration.

###### What happens if we reenable the feature if it was previously rolled back?

The watcher opens a new stream. Generic PLEG's next successful normal global relist establishes the recovery boundary without coordination from the watcher. No prior stream state is required.

###### Are there any tests for feature enablement/disablement?

Unit and node e2e tests need to cover enablement, disablement, unsupported runtimes, stream disconnection, restart, and re-enablement.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The stream may be unsupported, unavailable, malformed, or slow. These conditions only affect termination-detection latency. Generic PLEG continues polling and remains the source of pod status and lifecycle events. Kubelet and runtime outages have the same node-level impact as they do when this feature is disabled.

The main feature-specific load risk is a burst of stopped container events. Coalescing while a pod's relists are suspended, a bounded queue, global relist priority, and queue metrics limit and expose the additional work.

###### What specific metrics should inform a rollback?

Operators should monitor connection failures, disconnected time, dropped relist requests, Generic PLEG relist latency, and kubelet and runtime CPU usage. A stream failure is not a correctness failure while Generic PLEG is healthy. Disabling the feature removes stream retries and targeted relists during an investigation.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Automated tests need to cover a new kubelet with old and new runtimes, runtime restart while the stream is active, downgrade to a kubelet without the watcher, and re-upgrade. Generic PLEG needs to converge in every case.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The feature gate or configuration shows whether the feature is enabled. The required connection-state metric will show whether the fast path is currently usable. Generic PLEG last-seen and health remain separate correctness-path signals.

###### How can someone using this feature know that it is working for their instance?

The final metric set needs to make it possible to correlate connection success, relist requests from Evented PLEG, and per-pod relist latency to determine whether stopped container events are reaching Generic PLEG. Evented PLEG does not emit lifecycle events directly.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

With a healthy stream, kubelet should normally detect a container termination before the next global relist. Without a healthy stream, detection latency returns to the Generic PLEG baseline. Reconnect traffic needs to remain bounded, and a stream outage does not fail the PLEG health check.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Relevant signals include stream connection state, reconnect attempts, per-pod relist duration, dropped relist requests, and the existing Generic PLEG relist and last-seen metrics. End-to-end tests need to measure the time from container exit to the pod status update.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Yes. Kubelet needs metrics for connection state, retries, and relist request outcomes.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- **CRI runtime with `GetContainerEvents` support**
  - Usage: supplies optional stop-event latency hints.
  - Impact of an outage or incompatibility: the fast path is unavailable and reconnects are rate-limited; Generic PLEG continues normally.
  - Impact of degraded performance: delayed events may cause redundant relists, but fresh runtime reads and periodic global relists preserve correctness.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No Kubernetes API calls are added. The feature adds one long-lived local CRI stream and targeted local CRI queries after container stops.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No control-plane operation covered by an existing Kubernetes SLI or SLO gains additional work. Kubelet performs an additional local runtime query after a valid stopped-container event; that work is bounded and does not block the periodic global relist.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

High container termination rates may increase CPU and I/O in kubelet and the local CRI runtime. While a pod's relists are suspended, requests for that pod are coalesced into one pending relist. Outside suspension, queue capacity bounds pending work, and the dispatcher gives global relists priority. Stress tests and queue metrics need to validate these bounds.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

A faulty reconnect loop could churn sockets or goroutines. A single watcher, context cancellation, and capped jittered backoff prevent unbounded retries. Tests need to verify that goroutines and streams do not accumulate.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Stream watching and PLEG reconciliation are local to kubelet and the CRI runtime. API server or etcd unavailability affects kubelet status reporting as usual but does not change stream recovery.

###### What are other known failure modes?

- **Disconnected or flapping stream**
  - Detection: connection state, reconnect attempts, and connection-error metrics.
  - Mitigation: Generic PLEG continues normally; disable `EventedPLEG` if stream handling contributes to node pressure.
  - Diagnostics: rate-limited kubelet logs and CRI runtime logs.
  - Testing: unit and node e2e tests need to interrupt and restore the stream.
- **Event without a pod UID**
  - Detection: kubelet logs show that the event was ignored; the fast-path relist count does not increase.
  - Mitigation: upgrade or correct the runtime; Generic PLEG continues normally.
  - Diagnostics: inspect the runtime version and event metadata.
  - Testing: unit tests need to cover missing sandbox status, metadata, and UID.
- **Delayed or duplicate events**
  - Detection: relist requests may increase without a corresponding lifecycle transition.
  - Mitigation: no immediate action is required unless redundant work creates sustained load; fresh runtime reads prevent stale cache writes.
  - Diagnostics: compare event latency and relist metrics with runtime logs.
  - Testing: unit and node e2e tests need to inject delayed and duplicate events.
- **Full relist queue**
  - Detection: dropped-request metrics increase.
  - Mitigation: investigate kubelet or runtime saturation; the normal global relist still provides convergence.
  - Diagnostics: inspect queue, relist-duration, kubelet CPU, and runtime-operation metrics.
  - Testing: stress tests need to verify bounded work and continued global relists.
- **Unhealthy Generic PLEG**
  - Detection: existing PLEG health, last-seen, and runtime-operation signals.
  - Mitigation: investigate the runtime and kubelet independently of the event stream.
  - Diagnostics: use existing PLEG and runtime logs.
  - Testing: existing Generic PLEG coverage remains applicable.

###### What steps should be taken if SLOs are not being met to determine the problem?

First check Generic PLEG health, then inspect stream state, reconnect attempts, the relist queue, and runtime logs. If stream handling or targeted relists are contributing to the problem, disable `EventedPLEG` and restart kubelet. This does not change Generic PLEG behavior.

## Implementation History

- v1.26: initial Alpha implementation, disabled by default
  - <https://github.com/kubernetes/kubernetes/pull/111642>
  - <https://github.com/kubernetes/kubernetes/pull/111384>
- v1.27: Beta, default disabled
  - <https://github.com/kubernetes/kubernetes/pull/115967>
  - <https://github.com/kubernetes/test-infra/pull/28366>
  - <https://github.com/kubernetes/test-infra/pull/28592>
- v1.29: bug fix
  - <https://github.com/kubernetes/kubernetes/pull/120942>
- v1.30: reverted to Alpha and backported because of static-pod failures
  - <https://github.com/kubernetes/kubernetes/issues/121349>
  - <https://github.com/kubernetes/kubernetes/issues/121003>
  - <https://github.com/kubernetes/kubernetes/pull/122697>
  - <https://github.com/kubernetes/kubernetes/pull/122475>
- v1.36: introduce Generic PLEG on-demand relisting
  - <https://github.com/kubernetes/kubernetes/pull/137362>
- v1.37: narrow Evented PLEG to the container-termination hint path
  - <https://github.com/kubernetes/kubernetes/pull/139262>

## Drawbacks

- Generic PLEG keeps its normal polling cost, so this narrower design does not deliver the original KEP's steady-state CPU reduction goal.
- A termination adds a targeted CRI query that may be followed soon by a global relist. Coalescing during `SyncPod` and skipping requests already covered by a newer global relist reduce but cannot eliminate this duplicate work.
- The event stream remains operationally complex even though it is no longer a correctness dependency.
- Without a CRI replay cursor, recovery can converge current state but cannot guarantee reconstruction of every unobservable intermediate transition.

## Alternatives

- **Replace Generic PLEG with Evented PLEG.** Rejected because the CRI event stream is edge-triggered and has no durable replay. Without Generic PLEG, a disconnect, runtime restart, or missed event could leave the kubelet cache stale indefinitely. Generic PLEG provides the level-driven reconciliation needed to recover current runtime state.
- **Retain Evented PLEG as a second state producer.** Rejected because writing event payloads directly to the cache would introduce multiple writers, timestamp races, and the risk that a stale event overwrites newer state. It would also make correctness depend on stream delivery.
- **Increase Generic PLEG's relist period while connected.** Rejected because a longer period would increase reconciliation latency when events are lost or the runtime silently stops sending them. It would also require kubelet to switch modes based on stream health. Keeping the normal period provides a consistent reconciliation baseline.
- **Trigger targeted relists for every CRI event type.** Rejected because container stops are the relevant state changes that kubelet commonly does not initiate. Relisting on created, started, and deleted events would add runtime load without a demonstrated correctness or latency benefit.
- **Stop reconnecting after a fixed number of failures.** Rejected because a runtime can be unavailable longer than a fixed retry window or be upgraded in place. Capped indefinite backoff restores the optimization without kubelet restart and keeps retry load bounded.
- **Add durable CRI event replay.** Not required. A cursor and replay protocol could preserve events across a disconnect, but would substantially expand the CRI contract. Generic PLEG already reconciles kubelet with the current runtime state.
