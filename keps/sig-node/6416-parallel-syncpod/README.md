# KEP-999999: Parallel Container Operations in Kubelet SyncPod

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Current SyncPod Ordering](#current-syncpod-ordering)
  - [Proposal: Maintain Strict ordering semantics](#proposal-maintain-strict-ordering-semantics)
    - [Parallel Container Kills](#parallel-container-kills)
    - [Parallel Resize](#parallel-resize)
    - [Parallel ephemeral container startup](#parallel-ephemeral-container-startup)
    - [*Serial InitContainer startup](#serial-initcontainer-startup)
    - [Parallel main container startup](#parallel-main-container-startup)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1 (Optional)](#story-1-optional)
    - [Story 2 (Optional)](#story-2-optional)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Implicit Container Ordering Dependencies:](#implicit-container-ordering-dependencies)
    - [Other Risks](#other-risks)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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
  - [Async PostStartHook](#async-poststarthook)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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

Kubelet currently performs all operations in SyncPod in serial. This KEP proposes executing the
independent operations in parallel, while still maintaining the strict ordering of dependent
operations.

## Motivation

The serial execution of container operations in SyncPod are currently a blocker to running pods with
a higher container count. Consider pod startup: Kubelet starts the containers one at a time, waiting
for each container to have finished all the startup steps (including image pulling & post-start
hook). Not only does this delay the startup of the later containers, it also makes the Kubelet
unresponsive to events in the earlier containers while the the later containers are still starting.

### Goals

Reduce SyncPod latency to make high-container count pods more responsive, via:

- Parallelize regular container startup (`ContainersToStart`: image pull, `CreateContainer`, `StartContainer`, `postStart` hook) and termination (`ContainersToKill`: `preStop` hook, `StopContainer`).

### Non-Goals

- Independent container syncing (see [Alternatives](#alternatives))
- Changing existing pod-level ordering semantics

## Proposal

### Current SyncPod Ordering

https://github.com/kubernetes/kubernetes/blob/ec0fed81861cee5f94f4abb5133a9eb073bef04e/pkg/kubelet/kuberuntime/kuberuntime_manager.go#L1565-L1576

1. Compute sandbox and container changes.
2. Kill pod sandbox if necessary.
    1. Kill running containers **in parallel** (respecting sidecar termination ordering, modulo grace period)
        1. Run pre-stop hook (with grace period)
        2. `StopContainer`
    2. `StopPodSandbox`
3. (else) Kill any containers that should not be running (in serial)
    1. Run pre-stop hook (with grace period)
    2. `StopContainer`
4. Create sandbox if necessary.
    1. Invoke OnPodSandboxReady to notify Kubelet to update pod status.
5. Resize pod & running containers.
    1. Volume downsize
    2. Memory resize
        1. Downsize Pod
        2. Downsize Containers
        3. Upsize Containers
        4. Upsize Pod
    3. Volume upsize
    4. CPU resize (same pattern as memory)
6. Start ephemeral containers.
7. Start init containers.
8. Start normal containers.

Container start ordering:

1. Pull image volume images (serially)
2. Pull container image
3. `CreateContainer`
4. `StartContainer`
5. Execute post start lifecycle hook

### Proposal: Maintain Strict ordering semantics

Ordering is strictly maintained, but individual steps are executed in parallel.

EXCEPTION: volume & container image pulling executed as a single parallel step.

#### Parallel Container Kills

We already kill containers concurrently as part of Kill pod sandbox. We should reuse the same code
for killing a subset of containers without killing the pod sandbox.

All containers must terminate before moving on to the next step.

#### Parallel Resize

The strict resize ordering is maintained, but individual resize steps can be performed in parallel.
For each step, the individual items are executed in goroutines, and the step completion is blocked
on all goroutines successfully completing.

1. Downsize volumes in parallel
2. Downsize containers in parallel
3. Upsize containers in parallel
4. Upsize volumes in parallel

#### Parallel ephemeral container startup

The use case for parallel ephemeral container startup is not clear, but we should do so for consistency.

#### *Serial InitContainer startup

Init container startup does **NOT** happen in parallel. InitContainer startup will continue to maintain strict ordering.

#### Parallel main container startup

Each container is started in parallel, creating a go routine for the full container startup cycle:

1. Pull images (container + volumes) in parallel
2. `CreateContainer`
3. `StartContainer`
4. Post-start hook

### User Stories (Optional)

#### Story 1 (Optional)

- Multi-container AI/ML or mesh workload with 20+ application containers starts in `O(1)` wall-clock time instead of summing image pull and `postStart` latencies serially.

#### Story 2 (Optional)

- Pod with multiple crashed containers after recovers concurrently without waiting behind individual `preStop` or `postStart` hooks.

### Notes/Constraints/Caveats (Optional)

The changes proposed here allow the Kubelet to execute multiple container operations within a
**single SyncPod iteration**, but it does nothing for operations that are spread across separate
SyncPod invocations. Consider this case:

1. Container A: liveness probe fails, trigger's SyncPod
2. SyncPod begins execution, completes `computePodActions`
3. Container B: liveness probe fails
4. SyncPod continues with Container A restart
5. SyncPod completes
6. Next SyncPod executes, picks up Container B liveness failure

In this scenario, we see that handling Container B's liveness failure is still blocked on the serial execution of Container A. This behavior will still limit the responsiveness of the Kubelet for large pods with high-churn.

### Risks and Mitigations

#### Implicit Container Ordering Dependencies:

*Risk:* Previously, main container startup was ordered, so an earlier container in the list would
start before a later one. Similarly, a failure to start an earlier container would prevent later
containers from restarting. However, this behavior was always best-effort, and various factors (such
as high load) could stall container startup and let a later container start first.

*Mitigation:* Workloads with strict container startup ordering should switch to using sidecars,
which *do* guarantee start ordering. This is a behavioral change. The proposed mitigation is
communication & community outreach, such as *ACTION REQUIRED* release notes. The change will be
guarded by the `KubeletParallelContainerOps` feature gate with a phased rollout.

The alternative is to make the new behavior opt-in with a new pod field.

#### Other Risks

- **CRI Runtime & Node Resource Spikes:**
  - *Risk:* Concurrent `PullImage`, `CreateContainer`, `StartContainer`, and `StopContainer` calls cause CPU, disk I/O (rootfs overlay unpack/mount), and gRPC spikes.
  - *Mitigation:* TBD: Monitor and benchmark.
- **Containerd Shim Bottleneck:**
  - *Risk:* Containerd runs a single shim process per-pod. Parallel container operations may be bottle-necked by the shim process.
  - *Mitigation:* TBD: Monitor and benchmark.
- **Partial Failure Recovery & Backoff Contention:**
  - *Risk:* Concurrent container operations experience mixed successes and failures (`ErrImagePull`, `ErrCreateContainer`, `ErrPostStartHook`).
  - *Mitigation:* A failure in parallel step is handled before moving to the next step. This is largely consistent with current behavior, but means if one container fails to start it will not block other containers in the pod from starting. Multiple failures are aggregated into the `SyncResult`.

## Design Details

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

- Audit and refactor `pkg/kubelet/kuberuntime/kuberuntime_manager_test.go` fake runtime assertions to support non-deterministic call ordering for parallel container operations.

##### Unit tests

- `k8s.io/kubernetes/pkg/kubelet/kuberuntime`: `2026-09-17` - `78%`
- Verify strict serial execution of `InitContainersToStart` when `!podHasInitialized`.
- Verify concurrent execution of `ContainersToStart`, `ContainersToKill`, and post-init `InitContainersToStart` when `KubeletParallelContainerOps` is enabled.
    - Leverage SyncTest and injected gates to test variations on execution ordering.
- Verify partial failure isolation (one container fails `EnsureImageExists` or `PostStartHook` while sibling containers succeed).

##### Integration tests

- N/A — node-local kubelet runtime execution covered by unit tests with fake CRI and node e2e tests against real CRI runtimes.

##### e2e tests

- Add node e2e test in `test/e2e_node/` verifying parallel startup and termination of
  multi-container pods with slow `postStart`/`preStop` hooks and sidecar restarts. Verify guaranteed
  start/stop ordering of InitContainers.

### Graduation Criteria

#### Alpha

- Feature implemented behind `KubeletParallelContainerOps` feature gate (default `false`).
- Unit tests and node e2e tests passing.

#### Beta

- Conduct stress testing & benchmarking, no blockers found
- Community outreach regarding changes to ordering assumptions
- Manual enable / disable testing across Kubelet restarts
- Feature gate enabled by default

#### GA

- Two minor releases in Beta with zero concurrency regressions or CRI runtime deadlocks reported.
- Feature gate locked to `true` and deprecated.

#### Deprecation

- N/A — additive internal execution concurrency with no deprecated APIs or flags.

### Upgrade / Downgrade Strategy

- Purely node-local kubelet runtime change controlled by feature gate; upgrade/downgrade requires only toggling `KubeletParallelContainerOps` and restarting kubelet without workload disruption.

### Version Skew Strategy

- N/A — node-local kubelet `SyncPod` change with no API server, scheduler, or controller-manager interaction.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `KubeletParallelContainerOps`
  - Components depending on the feature gate: `kubelet`

###### Does enabling the feature change any default behavior?

- Yes; regular container starts/stops in `SyncPod` execute concurrently rather than serially,
  changing ordering for sibling containers. See [Implicit Container Ordering
  Dependencies](#implicit-container-ordering-dependencies) for discussion.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

- Yes; disabling `KubeletParallelContainerOps` and restarting kubelet immediately reverts `SyncPod` to serial container execution without affecting running containers.

###### What happens if we reenable the feature if it was previously rolled back?

- Subsequent `SyncPod` reconciliation loops resume parallel container operations cleanly.

###### Are there any tests for feature enablement/disablement?

- Unit tests in `pkg/kubelet/kuberuntime/kuberuntime_manager_test.go` parameterize `KubeletParallelContainerOps` toggled on and off.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

- Already running containers are unaffected; if CRI runtime deadlock or high CPU/I/O contention occurs during rollout, new container creations may time out until rolled back.

###### What specific metrics should inform a rollback?

- Spikes in `kubelet_runtime_operations_errors_total`, `kubelet_started_containers_errors_total`, or `kubelet_pod_start_duration_seconds`.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

- Will be validated via node e2e tests toggling feature gate across kubelet restarts prior to Beta graduation.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

- N/A - additive internal kubelet feature gate with no deprecations or removals.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- N/A - usage is implicit for all pods if the feature gate is enabled.

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event Reason: `Created`, `Started`, and `Killing` events emitted with overlapping timestamps for sibling containers.
- [x] Other (treat as last resort)
  - Details: Container `StartedAt` timestamps in `PodStatus.ContainerStatuses` reflect concurrent startup.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- Pod startup latency (excluding image pull) for N regular containers scales sub-linearly (`O(N / concurrency_limit)`) with no increase in container start error rates.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `kubelet_pod_start_duration_seconds`, `kubelet_started_containers_errors_total`, `kubelet_runtime_operations_duration_seconds`, `kubelet_runtime_operations_errors_total`
  - [Optional] Aggregation method: Histogram quantiles (p50, p99) and error counters
  - Components exposing the metric: `kubelet`

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

- No.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- Container Runtime Interface (CRI) implementation (containerd / CRI-O) handling concurrent gRPC `CreateContainer`, `StartContainer`, and `StopContainer` requests thread-safely.

### Scalability

###### Will enabling / using this feature result in any new API calls?

- N/A — node-local execution with pod status updates coalesced per `SyncPod` cycle.

###### Will enabling / using this feature result in introducing new API types?

- N/A — no new API types introduced.

###### Will enabling / using this feature result in any new calls to the cloud provider?

- N/A — node-local container runtime execution with no cloud provider interaction.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- N/A — existing Pod and Node API object schemas and counts unchanged.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

- No; reduces `kubelet_pod_start_duration_seconds` for multi-container pods.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

- Transient CPU and disk I/O bursts on nodes during concurrent container creation/start.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

- Unbounded concurrency could spike CRI gRPC sockets and disk I/O. Implicitly bounded by `GOMAXPROCS`.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

- N/A — node-local `SyncPod` execution on cached pod state independent of control plane availability.

###### What are other known failure modes?

- **CRI Runtime RPC Contention / Timeouts:**
  - *Detection:* Increase in `kubelet_runtime_operations_errors_total{operation_type="create_container|start_container"}` latency or timeout errors.
  - *Mitigations:* Disable `KubeletParallelContainerOps` feature gate.
  - *Diagnostics:* Kubelet logs showing concurrent `CreateContainer`/`StartContainer` context deadline exceeded errors.
  - *Testing:* Node stress test launching high-container-count pods concurrently, required for Beta.

###### What steps should be taken if SLOs are not being met to determine the problem?

- Check kubelet CRI latency metrics (`kubelet_runtime_operations_duration_seconds`), container runtime daemon logs, and node disk I/O utilization; disable the feature gate if runtime contention is observed.

## Implementation History

- 2026-09-22: Initial KEP drafted.

## Drawbacks

- Increases concurrency complexity and synchronization requirements inside `kuberuntime_manager.go`.
- Alters deterministic sequential container startup order for regular application containers that implicitly relied on undocumented spec-order startup.

## Alternatives

### Async PostStartHook

PostStartHooks can run for a long time. Even with the changes proposed in this KEP, a long-running
post-start hook will still block the completion of SyncPod, preventing the Kubelet from responding
to other lifecycle events and triggering a new SyncPod iteration.

If we treat the post start hook like a special one-off container Start probe, then it's execution
could continue asynchronously and unblock SyncPod. This could leverage the changes proposed in
https://github.com/kubernetes/kubernetes/pull/140704, and would gate other probe execution, and
allow the Kubelet to respond to a hook failure appropriately.

This should be considered as a follow-up extension, but isn't proposed as part of the initial alpha
changes.

## Infrastructure Needed (Optional)

- N/A — existing Kubernetes node CI test infrastructure sufficient.
