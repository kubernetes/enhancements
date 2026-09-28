# KEP-6318: In-place Updates to Container Probes

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1 (Optional)](#story-1-optional)
    - [Story 2 (Optional)](#story-2-optional)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Behavior](#api-behavior)
  - [Kubelet Behavior](#kubelet-behavior)
  - [Adding and removing probes](#adding-and-removing-probes)
  - [Restart and termination behavior](#restart-and-termination-behavior)
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
  - [Recreate the Pod](#recreate-the-pod)
  - [Add an annotation to disable probes](#add-an-annotation-to-disable-probes)
  - [Add a probe subresource](#add-a-probe-subresource)
  - [Only update workload templates](#only-update-workload-templates)
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

Container probes are part of a Pod's desired configuration, but today their configuration cannot be changed after the Pod is created. Correcting a probe, tuning it after observing a workload, or temporarily removing a liveness probe therefore requires replacing the Pod.

This KEP makes liveness, readiness, and startup probes mutable for regular containers and restartable init containers. A user can update the existing probe fields through the normal Pod `UPDATE` or `PATCH` APIs. The kubelet reconciles the new configuration without recreating the Pod or restarting the container merely because its probe changed.

## Motivation

Probe settings are often chosen before an application is run under production load. In practice, an unsuitable timeout or threshold may only become apparent later. A liveness probe that repeatedly fails can also make an incident harder to diagnose because the container is restarted before an operator can inspect it. The current remedy is to update a workload template and replace the Pod, which loses the state of that particular container and may add disruption at exactly the wrong time.

There has been long-standing interest in changing or disabling probes on a running Pod, including the use cases described in [kubernetes/kubernetes#57187](https://github.com/kubernetes/kubernetes/issues/57187). The existing probe fields already express the desired configuration. Allowing carefully scoped updates to those fields is more direct than introducing a second mechanism for overriding them.

### Goals

- Allow liveness, readiness, and startup probes to be added, changed, or removed on a running Pod.
- Apply probe changes without restarting a container solely because its probe configuration changed.
- Support regular containers and restartable init containers consistently.

### Non-Goals

- Making other container fields mutable.
- Supporting probe updates for ordinary init containers, ephemeral containers, mirror Pods, static Pods, terminating Pods, or terminal Pods.
- Updating a controller's Pod template when an individual Pod is changed.
- Adding a probe subresource, a new API type, or a CRI operation.
- Providing conditional or policy-driven probes based on the state of other workloads.
- Providing a probe-specific acknowledgement that every kubelet has applied a particular update.

## Proposal

When the `MutableContainerProbes` feature gate is enabled, the Pod API permits changes to `livenessProbe`, `readinessProbe`, and `startupProbe` on regular containers. The same fields may be changed on init containers whose `restartPolicy` is `Always`. All other Pod mutability rules continue to apply.

Probe changes use the normal Pod `UPDATE` and `PATCH` endpoints. A request that also changes an immutable field is rejected as a whole. Successful persistence of the update does not mean that the kubelet has already run the new probe; propagation follows the usual PodSpec convergence model.

The kubelet reconciles the probes in the latest PodSpec with its probe workers. It creates workers as needed, updates existing workers, and disables workers whose probes are removed without publishing an artificial probe failure. Results produced by an obsolete probe configuration or a previous container instance are not allowed to affect the current container.

### User Stories (Optional)

#### Story 1 (Optional)

As an operator investigating liveness-probe-triggered restarts, I want to temporarily relax or remove the affected Pod's liveness probe so that I have time to inspect the running container and its in-memory state without replacing the Pod to change the probe. After diagnosis, I want to restore the probe or apply a corrected configuration.

#### Story 2 (Optional)

As the owner of a stateful workload, such as a database, I want to adjust an overly short liveness probe timeout when temporary storage contention delays health-check responses even though the application is still making progress. This lets me retain liveness checking without restarting the container or replacing the Pod solely to correct the probe. Readiness continues to determine whether the Pod can serve traffic.

If the database is still recovering and has not yet passed its startup probe, I also want to increase the startup probe's failure threshold to allow recovery to complete without a probe-triggered restart. Changes intended for replacement Pods must also be made to the owning workload's Pod template.

### Notes/Constraints/Caveats (Optional)

- An update to an individual Pod is not copied back to the owning workload. A replacement Pod will use the controller's Pod template, so users that want the change to persist across replacement must update that template as well.
- Adding or changing a liveness or startup probe can subsequently restart a container when the new probe fails. The update itself does not restart it.
- Existing `update` or `patch` permission on Pods is sufficient. This is worth noting for `exec` probes because an update may change a command that the kubelet runs in a container.

### Risks and Mitigations

An incorrect liveness probe can restart a healthy container, while an incorrect readiness probe can remove it from Service endpoints. That risk already exists when a Pod is created, but mutable probes allow the effect to be introduced to a running workload. Existing probe validation remains in place, and updates remain subject to Pod RBAC, admission, managed fields, and audit. Documentation will recommend changing controller templates as the normal path and using per-Pod changes deliberately.

A probe may finish at the same time its configuration is changed or removed. Without additional ordering, a stale failure could affect container status or trigger a restart. The kubelet will associate results with the Pod, container instance, probe type, and current execution of the configuration. Results that no longer match are discarded. In-flight attempts are also cancelled when a handler or timeout changes, or when the probe is removed.

During a skewed rollout, an API server may accept a change before the kubelet on the target node can apply it dynamically. The recommended rollout order is to upgrade and enable kubelets before enabling API servers. The PodSpec remains valid on older components, and no new API field needs conversion.

Allowing an `exec` probe to change also lets a Pod updater change a command that the kubelet executes inside an existing container. This does not add an RBAC verb, and Pod updates remain subject to admission and audit, but policy authors should account for the broader effect of the existing Pod update permission.

## Design Details

### API Behavior

The feature does not add API fields. It changes update validation for the following existing fields:

- `spec.containers[*].livenessProbe`
- `spec.containers[*].readinessProbe`
- `spec.containers[*].startupProbe`
- the same three fields on `spec.initContainers[*]` when the init container has `restartPolicy: Always`

The new probe value is fully validated before PodSpec immutability is checked. Container count, names, ordering, and all fields outside the allowed probes remain subject to the current update rules. Updates that change probes on Pods with a `deletionTimestamp`, or on Pods in the `Succeeded` or `Failed` phase, are rejected.

Static and mirror Pods remain managed from their source manifests. Users must change that source rather than patching the mirror Pod through the API.

### Kubelet Behavior

The probe manager will reconcile the desired workers every time the kubelet processes an updated Pod. A worker is still identified by Pod UID, container name, and probe type. Reconciliation adds missing workers, updates enabled workers, and disables workers for probes that were removed. A disabled worker remains available for reuse until the probe is re-enabled or the Pod is cleaned up.

Workers use an immutable snapshot of probe configuration for each attempt. The manager invalidates attempts when their handler or timeout is replaced, when the probe is disabled, when the worker is removed, or when the container ID changes. A late result from an invalid attempt is ignored even if cancellation did not stop the underlying probe in time.

When the kubelet applies a probe addition, update, or removal for a running Pod, it emits a `Normal` Pod event with the reason `ContainerProbeChanged`. The event message identifies the container, probe type, and action. This event is a best-effort troubleshooting signal that the node observed and applied the change; it is not a durable acknowledgement or a mechanism for inventorying use of the feature.

The following flow summarizes how kubelet coordinates health checks when it
observes a Pod update. It describes the intended behavior rather than specific
implementation calls.

![Health-check reconciliation flow](ReconcilePod.png)

The behavior of individual changes is:

| Change | Consecutive result count | Published status | Next execution |
| --- | --- | --- | --- |
| `periodSeconds` | Preserved | Preserved | Recalculated from the previous completion time |
| `initialDelaySeconds` | Preserved | Preserved | Recalculated before the first probe; otherwise no effect for the current container |
| `terminationGracePeriodSeconds` | Preserved | Preserved | Used by a later probe-triggered termination |
| Success or failure threshold | Reset | Preserved | Continues on the current schedule |
| Handler or `timeoutSeconds` | Reset | Preserved | The old attempt is invalidated and the new configuration is used immediately |
| Probe added or re-added | Initialized | Determined by probe type | A new worker is started or a disabled worker is re-enabled |
| Probe removed | Removed | Recomputed as if no probe were configured | Scheduling stops; the cached result is removed and the disabled worker is retained for reuse |

If multiple fields change together, the strongest applicable behavior in the table is used. In particular, a handler change also resets the consecutive result count even when the period changes in the same update.

For a threshold-only update, a probe that was already in flight may count as the first result under the new threshold. It cannot retain the count from the old threshold.

Changing only `periodSeconds` neither clears an accumulated threshold count nor makes a container temporarily unready. The next run is based on the completion time of the previous probe. If that time plus the new period is already in the past, the worker runs promptly rather than waiting for another full period.

### Adding and removing probes

Adding a readiness probe to a running, started container uses the current readiness semantics: the initial result is failure and the container is not Ready until the probe reaches its success threshold. Adding a liveness probe starts with a successful initial result, so the addition itself does not restart the container.

Once `Started` is true for a container ID, adding or changing a startup probe does not change it back to false. The configuration is retained and is used if the container is restarted with a new container ID. If the current container has not started, the startup worker is created and run normally.

Removing a readiness probe makes a running container Ready when it is already Started. Removing a startup probe makes a running container Started and allows readiness and liveness probing to proceed; an existing readiness probe is scheduled promptly. Removing a liveness probe does not change Ready or Started. In all cases, cached results and stale in-flight results for the removed probe are discarded before Pod status is recomputed. Probe execution stops, while its disabled worker state is retained until the probe is re-enabled or the Pod is cleaned up. Re-enabling the probe reuses that worker so that it cannot overlap an earlier attempt that did not stop immediately after cancellation.

Restartable init containers follow the same rules. Ordinary init containers are excluded because their probes do not have the continuing lifecycle of an app container or sidecar.

### Restart and termination behavior

Probe configuration is persisted in the PodSpec, so a kubelet restart rebuilds workers from the most recently observed spec. Timers, consecutive counts, cached results, and in-flight attempts are in-memory state and are initialized again, as they are today. No new checkpoint is introduced.

The initial status after a kubelet restart continues to follow the behavior selected by `ChangeContainerStatusOnKubeletRestart`; mutable probes do not add a second restart policy.

When a container restarts within the same Pod, results for the old container ID are removed and all probe state is initialized for the new instance. A newly created Pod has a new UID and does not inherit a per-Pod probe update unless its controller template was also changed.

Pod termination takes precedence over probe reconciliation. No workers are created or replaced after termination begins, and outstanding liveness and startup attempts are invalidated. Final Pod removal cleans up all probe workers and cached results, including those belonging to restartable init containers.

### Test Plan

- [x] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

##### Prerequisite testing updates

No prerequisite test refactoring is required. Existing probe manager tests provide fake clocks, result managers, and container status that can be extended for update cases.

##### Unit tests

- `k8s.io/kubernetes/pkg/kubelet/prober`: `2026-09-05` - `84.5%`
- `k8s.io/kubernetes/pkg/kubelet`: `2026-09-05` - `71.5%`
- `k8s.io/kubernetes/pkg/apis/core/validation` - `86.0%`

##### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)
-->

No dedicated integration tests are needed.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.
-->

Node e2e tests will update the readiness, liveness, and startup probes of a running Pod and verify the corresponding `Ready`, `Started`, and restart behavior while confirming that the update itself does not restart the container. The tests will cover adding, modifying, and removing probes for regular containers and restartable init containers, and verify that a new container instance uses the latest startup probe configuration.

### Graduation Criteria

#### Alpha

- The feature is implemented behind the `MutableContainerProbes` feature gate.
- Node e2e tests cover the primary update paths for regular containers and restartable init containers.
- User-facing documentation describes update semantics and version skew.

#### Beta

- All three probe types and all supported field changes have stable automated coverage.
- Upgrade, downgrade, and feature disablement have been tested.
- Operational experience shows no unresolved correctness issues with stale results, container restarts, or Pod status transitions.
- Monitoring and troubleshooting guidance is complete.
- Feedback from Alpha users has been addressed.

#### GA

- The feature has been Beta for at least two releases.
- No significant unresolved issues attributable to mutable probes remain.
- SIG Node agrees that the update semantics have proven stable in production.

### Upgrade / Downgrade Strategy

Upgrading does not change existing Pods or probe behavior. Cluster operators should upgrade kubelets first, enable the gate on them, and then enable it on API servers. Users opt in by updating a probe on an existing Pod.

Disabling the feature gate on API servers causes subsequent probe changes to be rejected. It does not rewrite probe configuration already stored in a PodSpec. After a gate-disabled kubelet restarts, it builds workers from that stored configuration, but it does not dynamically reconcile later probe updates. No API data conversion is required during downgrade.

### Version Skew Strategy

- A new API server with the gate disabled rejects probe changes as it does today.
- A new API server with the gate enabled may persist a change for an old or gate-disabled kubelet, but that kubelet is not guaranteed to apply the change to an already running container. It can use the stored configuration when it next creates probe workers.
- A new kubelet cannot receive a mutable probe update through an old or gate-disabled API server because the update is rejected.
- During an API server rollout, clients may observe inconsistent acceptance until the gate is configured consistently on all API server instances.
- The scheduler, controller manager, container runtime, and CRI do not require changes for this feature.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `MutableContainerProbes`
  - Components depending on the feature gate:
    - `kube-apiserver`
    - `kubelet`

###### Does enabling the feature change any default behavior?

Existing Pods and probes are unchanged. The API server begins accepting a class of Pod updates that was previously rejected, and a gate-enabled kubelet applies those explicit updates to running containers.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disable the gate on API servers to prevent further probe updates and on kubelets to stop dynamic reconciliation. Probe values already stored in Pods are not rolled back and remain valid configuration. Running containers are not restarted as part of disabling the gate.

###### What happens if we reenable the feature if it was previously rolled back?

The API server accepts probe updates again, and kubelets resume reconciling the probe configuration in the latest PodSpec. No probe-specific state needs to be restored.

###### Are there any tests for feature enablement/disablement?

Planned unit tests cover updates with the gate enabled and disabled, including Pods whose probe configuration was changed before the gate was disabled. The feature will not graduate from Alpha without this coverage.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

If API servers are enabled before kubelets, an update can be accepted but not applied dynamically on an older node. During an API server rollout, otherwise identical requests can also be accepted or rejected depending on which API server handles them. Enabling kubelets first and completing each component's rollout before moving to the next avoids these cases.

The feature can affect a running workload only after a user changes its probe. A bad liveness or startup probe can lead to restarts, and a bad readiness probe can affect traffic. Rolling back the gate does not restore an earlier probe value; users should first patch the Pod back to a known-good configuration if the configuration itself is the problem.

###### What specific metrics should inform a rollback?

Unexpected changes in `prober_probe_total`, especially failures for liveness or startup probes after updates, and regressions in kubelet Pod worker duration should inform a rollback. Pod events and container restart counts provide workload-level confirmation.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet. This will be tested before Beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Pod update audit records show changes to probe fields. The stored PodSpec shows the current configuration, although it does not by itself show whether that configuration was supplied at creation or by an update. Audit records can only provide this detail when the cluster's audit policy captures request bodies. No new per-workload metric is introduced.

The kubelet event emitted when a probe is added, updated, or removed is intended for troubleshooting an individual Pod. Because events are best-effort and short-lived, they are not used to determine feature adoption across workloads.

###### How can someone using this feature know that it is working for their instance?

- [x] API `.status`
  - Condition name: `Ready` and `ContainersReady` for readiness changes
  - Other field:
- [x] Events
  - Event Reason: `ContainerProbeChanged`, plus existing probe failure and container restart events

Users can also observe the behavior of the updated handler. For example, a readiness probe changed from a failing endpoint to a healthy one will result in the container becoming Ready after the configured threshold is reached. The `ContainerProbeChanged` event shows that the kubelet observed and applied a probe addition, update, or removal, but remains a best-effort signal rather than a durable acknowledgement.

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

- [x] Metrics
  - Metric name: `prober_probe_total`, `prober_probe_duration_seconds`, and `kubelet_pod_worker_duration_seconds`
  - Aggregation method: compare rates and latency before and after rollout, grouped by node and probe type where labels permit
  - Components exposing the metric: kubelet
- [x] Other
  - Details: Pod update request errors and latency from API server metrics

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A counter for probe worker reconciliation failures may be useful if Alpha experience shows that existing kubelet errors and probe metrics do not make those failures clear. This is not initially proposed because reconciliation is local and retried as part of Pod sync. Alpha experience will determine whether a dedicated metric provides useful information beyond existing Pod worker and probe metrics.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

The kubelet creates a Pod event when it applies a probe addition, update, or removal. These writes occur only in response to Pod changes; there is no periodic API traffic associated with the feature. Users and automation may also issue ordinary Pod `UPDATE` or `PATCH` requests when they change a probe, and their volume depends on how often workloads are changed.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Pod update validation compares the allowed probe fields, and kubelet Pod sync reconciles probe workers. Both operations are linear in the number of containers and probes in the Pod and are expected to be negligible relative to the existing work in those paths. API request and kubelet Pod worker metrics will be monitored during Alpha.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

Frequent updates can cause probe attempts to be cancelled and restarted. The number of workers remains bounded, but an attempt whose cancellation is delayed may temporarily retain a socket or exec request until its timeout. API request throttling, probe timeouts, cancellation, and stale-result rejection bound the impact. Tests will include repeated updates with blocked probes.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Probe updates cannot be submitted or persisted while the API is unavailable. Kubelets continue running the last probe configuration they observed, just as they continue using the last observed PodSpec today.

###### What are other known failure modes?

- **A probe update is rejected.** Check that `MutableContainerProbes` is enabled on every API server, the Pod is neither terminating nor terminal, and the request changes only supported probe fields.
- **An accepted update is not applied on a node.** Check the node version and kubelet feature gate, then compare the Pod generation with the generation observed by the kubelet. Version-skew tests cover this case.
- **A workload becomes unready or starts restarting.** Inspect the current probe in the PodSpec, Pod events, container status, and `prober_probe_total`. Restore the previous probe or remove it in another Pod update before disabling the feature gate. Node e2e tests cover these status transitions.
- **A result from an old configuration is observed.** Kubelet logs and probe events can identify the timing of the update and result. Unit tests exercise blocked in-flight probes, removal, and container ID changes.

###### What steps should be taken if SLOs are not being met to determine the problem?

First determine whether failures are in Pod update admission or kubelet reconciliation. For the former, inspect API response codes, audit records, and API server request metrics. For the latter, compare Pod generation, node version, kubelet configuration, `kubelet_pod_worker_duration_seconds`, probe metrics, Pod events, and container status. Revert the probe to a known-good value to distinguish configuration errors from reconciliation errors.

## Implementation History

- 2026-09-03: Initial KEP proposed for Alpha.

## Drawbacks

This change expands the set of mutable Pod fields and makes a potentially disruptive operation available without a controller rollout. It also adds concurrency and state-management complexity to the kubelet probe manager. Finally, a per-Pod edit can drift from its controller template and disappear when the Pod is replaced.

## Alternatives

### Recreate the Pod

The current approach is to update the workload template and replace the Pod. This keeps the controller as the only source of configuration, but it restarts the container and discards the exact instance an operator may need to inspect. It also adds avoidable disruption for simple probe tuning.

### Add an annotation to disable probes

An annotation could temporarily suppress probe execution without changing the probe fields. It would cover only the disable case, introduce a second source of probe configuration, and require separate precedence rules. Updating the existing probe fields supports disabling, tuning, and changing handlers with the normal Pod validation and field ownership model.

### Add a probe subresource

A dedicated subresource could provide separate authorization and an explicit application status. It would also add a new API surface, storage and admission semantics, client support, and another write path for configuration already stored in the PodSpec. Normal Pod updates are sufficient for the intended scope.

### Only update workload templates

Changing a Deployment or StatefulSet template remains the preferred way to change future Pods, but it cannot help with an existing container whose state must be preserved. This KEP complements template updates rather than replacing them.
