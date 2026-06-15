# KEP-6063: Per-Pod PID Limit

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [API Changes](#api-changes)
  - [Valid Values](#valid-values)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
    - [Feature Gate](#feature-gate)
    - [API Validation](#api-validation)
    - [Kubelet Implementation](#kubelet-implementation)
    - [Pod Security Admission (PSA)](#pod-security-admission-psa)
    - [Eviction Manager Interaction](#eviction-manager-interaction)
    - [Node Declared Features Integration](#node-declared-features-integration)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (target 1.37)](#alpha-target-137)
    - [Beta (target 1.38)](#beta-target-138)
    - [GA (target 1.40)](#ga-target-140)
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
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Add support for setting PID limits at the Pod level to restrict below the node-level kubelet PID limit. This allows individual Pods to specify their own PID limits using `spec.resources.limits.pid` in the Pod specification. The effective limit is the lower of the Pod-specified value and the node's `podPidsLimit`.

## Motivation

Currently, PID limiting in Kubernetes is configured globally at the node level via the kubelet's `podPidsLimit` setting. This creates challenges when different workloads on the same node have different PID requirements:
1. **Infrastructure pods affected by low limits**: If a cluster administrator sets a low PID limit to protect against runaway processes, this limit applies equally to all pods on the node, including critical infrastructure pods that may legitimately need more PIDs.

2. **Agent and sandbox workloads**: Modern workloads like AI agent runtimes (Agent Sandbox, OpenShell) and service mesh sidecars (Istio) often have specific PID requirements that differ from typical application pods.

3. **No granular control**: Administrators cannot apply different PID limits to different types of workloads without node-level segregation, which is operationally expensive.

4. **Pod-level resource model**: With the introduction of Pod-level resources in Kubernetes, PID limits logically belong alongside other Pod-level resource constraints.


### Goals

- Allow Pods to specify PID limits via `spec.resources.limits.pid` in the Pod specification
- Restrict below the kubelet's default `podPidsLimit` when a Pod-level limit is set
- Use the lower value when both kubelet and Pod limits are specified
- Implement PID limiting at the Pod cgroup level for cgroupsv2
- Maintain backward compatibility with existing node-level PID limiting

### Non-Goals

- Support for non-Linux operating systems (PID cgroup limits are Linux-specific)
- Support for cgroupsv1 (this enhancement targets cgroupsv2 only)
- Container-level PID limits (this KEP focuses on Pod-level limits only)
- Automatic PID limit calculation based on workload characteristics
- Changes to the existing node-level PID limiting mechanism
- In-place resize of PID limits (in-place resize currently supports only CPU and memory)

## Proposal

Add a new `pid` resource type that can be specified under `spec.resources.limits` in the Pod specification. When set, the effective PID limit is `min(podPidsLimit, pod.spec.resources.limits.pid)` — the Pod can only restrict further, not exceed the node limit.

### API Changes
Extend the Pod specification to support PID limits:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: example-pod
spec:
  resources:
    limits:
      pid: "2048"         # Pod-level PID limit
  containers:
  - name: app
    image: my-app:latest
    resources:
      limits:
        cpu: "2"
        memory: "4Gi"
  - name: sidecar
    image: log-agent:latest
    resources:
      limits:
        cpu: "100m"
        memory: "64Mi"
```

- Only `spec.resources.limits.pid` is accepted; `spec.resources.requests.pid` is forbidden and will be rejected by the API server
- When `spec.resources.limits.pid` is specified, the kubelet applies this limit to the Pod's cgroup
- All containers in the Pod share the same PID pool enforced by the pod cgroup
- If both kubelet `podPidsLimit` and Pod `spec.resources.limits.pid` are set, the kubelet enforces `min(podPidsLimit, spec.resources.limits.pid)`. The Pod may request a value higher than the node's `podPidsLimit` (up to the API maximum of 16384), but the kubelet always caps at the node's `podPidsLimit`, ensuring the node administrator retains control.
- If `spec.resources.limits.pid` is not specified, the kubelet's `podPidsLimit` applies (existing behavior)


### Valid Values
Following the same constraints as node-level `podPidsLimit`:

 - **Minimum**: 1024 (conservative lower bound)
 - **Maximum**: 16384 (maximum supported in managed environments)
 - **Default**: If not specified, inherits kubelet's `podPidsLimit`

### User Stories

#### Story 1
Agent Sandbox Workloads

As a platform engineer running AI agent workloads (such as Agent Sandbox or
OpenShell), I need to restrict PID limits for agent pods because they run
untrusted code, while allowing trusted application pods to use the full
node default.

Current state: I must set a low `podPidsLimit` at the node level to
protect against runaway agent processes, but this unnecessarily restricts
all pods on the node, including trusted application pods that may
legitimately need more PIDs.

With this enhancement: I can keep the node's `podPidsLimit` at a
comfortable level (e.g., 4096) for trusted application pods. I can then
set `spec.resources.limits.pid: "2048"` on agent pods to contain untrusted
workloads. Because the kubelet enforces the lower of the two limits, agent
pods are safely restricted to 2048, while trusted application pods can use
the full node default of 4096.

#### Story 2
Infrastructure Pod Protection

As a cluster administrator, I want to set a conservative PID limit (2048) for
tenant workloads while allowing infrastructure pods (e.g., monitoring agents,
log collectors, service mesh proxies) to use higher limits (4096) on the same
nodes.

Current state: I must choose between protecting the node (a low node limit
that starves infrastructure pods) or allowing infrastructure pods to function
(a high node limit that risks PID exhaustion from tenant pods).

With this enhancement: I can set the kubelet's `podPidsLimit` to 4096 to
support infrastructure pods. I can then enforce
`spec.resources.limits.pid: "2048"` on tenant pods (via admission control or
default LimitRanges). Since the lower value wins, tenant pods are strictly
capped at 2048, while infrastructure pods can scale up to the node limit of
4096.

#### Story 3
Service Mesh Sidecar Requirements

As an application developer using Istio, my application pods with Envoy
sidecars need more PIDs than standard applications because the sidecar creates
multiple worker threads and connections.

Current state: If the default namespace limit is too low, I have no way to
increase my PID limit without the cluster administrator globally changing the
node-level `podPidsLimit`, which reduces overall cluster safety.

With this enhancement: The cluster administrator sets the node's
`podPidsLimit` to 8192. Standard application pods set
`spec.resources.limits.pid: "2048"` to stay conservative. For my Istio
pods, I set `spec.resources.limits.pid: "4096"` to give the Envoy sidecar
the headroom it needs. The kubelet enforces `min(8192, 4096) = 4096`,
granting my pod more PIDs without affecting other workloads or requiring
a node-level configuration change.

### Notes/Constraints/Caveats

cgroupsv2 only: This enhancement requires cgroupsv2. On cgroupsv1 nodes, the kubelet will reject pods that specify `spec.resources.limits.pid` during admission, rather than silently ignoring the field. This prevents a situation where the user expects PID enforcement but the node cannot deliver it.

Lower value precedence: When both kubelet and Pod specify limits, the lower value is enforced. This ensures that node-level protections cannot be bypassed by Pod specifications.

Hierarchical cgroup limits: PID limits are hierarchical in cgroups, so the most restrictive limit in the hierarchy applies.

Resource quota and limits: PID limits should be considered in cluster resource planning, though they do not consume schedulable resources like CPU or memory.

Pod Security Admission (PSA) compatibility: The `spec.resources.limits.pid` field has no PSA impact. Pods specifying PID limits are admitted under all PSA profiles (Privileged, Baseline, and Restricted) without triggering any policy violations. PSA does not inspect resource limits, so no special configuration or exemptions are needed.

Node Declared Features: This feature integrates with the Node Declared Features framework ([KEP-5328](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/5328-node-declared-features)) to handle version skew and mixed cgroupsv1/v2 clusters. Nodes that support per-pod PID limits declare `PerPodPIDLimit` in `node.status.declaredFeatures`, enabling the scheduler to avoid placing pods with `spec.resources.limits.pid` on incompatible nodes.

`hostPID` compatibility: Pods with `hostPID: true` and `spec.resources.limits.pid` are fully supported. The PID namespace (process visibility) and PID cgroup controller (process creation accounting) are independent kernel subsystems. `hostPID` allows the container to see host processes, but those processes belong to different cgroups and are not counted against the pod's PID limit. The cgroup `pids.max` limit is enforced regardless of PID namespace configuration.

### Risks and Mitigations

1. **Users expect to raise PID limits above node default**
   - Risk: The "lower value wins" design means Pods can only restrict, not raise. Users may expect `pid: 8192` to override a node `podPidsLimit` of 4096.
   - Mitigation: Clear documentation. The node admin retains control — raise `podPidsLimit` if workloads need more.

2. **Interaction with existing PID exhaustion protections**
   - Risk: A Pod could set a very low PID limit, making itself non-functional.
   - Mitigation: API validation enforces a minimum of 1024 and a maximum of 16384, so extremely low or high values are rejected. Even at the minimum (1024), the pod has sufficient PIDs to function.

3. **Version skew: apiserver accepts but kubelet ignores**
   - Risk: If only apiserver enables the gate, users think their limit is enforced but kubelet applies the node default.
   - Mitigation: Documented in Version Skew Strategy. Both components must enable the gate for enforcement.

## Design Details

#### Feature Gate
The feature is controlled by a new feature gate: `PerPodPIDLimit`.

This feature depends on the `PodLevelResources` feature gate (Beta, enabled by default
since v1.34) because `pid` is specified under `pod.spec.resources.limits`. If
`PodLevelResources` is disabled, `PerPodPIDLimit` cannot be enabled — the kubelet
and API server will reject the configuration at startup.

When the feature gate is disabled:

- The kube-apiserver rejects Pods that specify `pid` under `spec.resources.limits`
- Existing Pods that already contain `pid` are preserved and remain unchanged
- The kubelet ignores the `pid` limit and continues using the node-level `podPidsLimit` setting instead

#### API Validation
Add `pid` to the list of valid resource types

Validate that `pid` values are integers within the allowed range (1024-16384)

#### Kubelet Implementation
Parse Pod specification: The kubelet reads `spec.resources.limits.pid` from the Pod spec during Pod admission.

Limit enforcement: When creating the Pod cgroup, the kubelet sets effective PID limits based on:

- If Pod `spec.resources.limits.pid` is set: `min(podPidsLimit, pod.spec.resources.limits.pid)`
- If Pod `spec.resources.limits.pid` is not set: `podPidsLimit` (current behavior)

Validation: The kubelet validates that the PID limit is within acceptable bounds (1024 to 16384).

cgroupsv2 requirement: The kubelet checks the cgroup version and only applies Pod-level limits on cgroupsv2 systems. On cgroupsv1 systems, the kubelet rejects pods that specify `spec.resources.limits.pid` during admission.

Event on capping: When a pod's requested PID limit exceeds the node's `podPidsLimit`, the kubelet emits a `PIDLimitCapped` event on the pod, indicating the effective limit was capped at the node-defined maximum. This gives users visibility when their requested PID limit is not fully honored.

#### Pod Security Admission (PSA)

This feature has no PSA interaction. Pod Security Admission does not inspect
resource limits (`spec.resources.limits`), so pods specifying
`spec.resources.limits.pid` are admitted under all PSA profiles (Privileged,
Baseline, and Restricted) without triggering any policy violations. No special
configuration, exemptions, or PSA policy changes are required.

#### Eviction Manager Interaction

The kubelet eviction manager monitors node-level PID availability and can set
the `PIDPressure` node condition when available PIDs fall below configured
eviction thresholds. When PID pressure is detected, the eviction manager may
evict pods to recover node stability.

Per-pod PID limits do not modify the eviction manager's behavior, eviction
thresholds, or pod ranking logic. PID pressure detection continues to be based
on node-level PID availability (`pid.available`) and operates independently of
pod-level PID limits.

Per-pod PID limits act as a preventative control by limiting the maximum number
of processes that an individual pod may create. This can reduce the likelihood
of node-wide PID exhaustion and therefore reduce the frequency of
`PIDPressure` conditions.

#### Node Declared Features Integration

This feature integrates with the Node Declared Features framework
([KEP-5328](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/5328-node-declared-features),
GA since v1.36) to handle version skew and mixed cgroupsv1/v2 clusters
gracefully.

When the `PerPodPIDLimit` feature gate is enabled and the node supports
cgroupsv2, the kubelet declares `PerPodPIDLimit` in
`node.status.declaredFeatures` during bootstrap. This enables:

- **Scheduler filtering**: The scheduler infers that a pod with
  `spec.resources.limits.pid` requires `PerPodPIDLimit` and only places it on
  nodes that declare the feature. This prevents pods from being scheduled to
  cgroupsv1 nodes or nodes without the feature gate, avoiding kubelet admission
  rejections that would put the pod into a non-retriable `Failed` state.
- **Kubelet admission validation**: Before admitting a pod, the kubelet
  validates the pod's feature requirements against its declared features as a
  secondary check (handles node restart with a feature gate flip).

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

No prerequisite testing updates are required. The existing PID limiting tests
in `test/e2e_node/pids_test.go` (`PodPidsLimit` suite) cover the node-level
`podPidsLimit` behavior which this feature builds upon.

##### Unit tests

This feature touches API validation, field stripping, and kubelet cgroup enforcement. For Alpha, unit test coverage for the following packages is added:

* `pkg/apis/core/validation` will be updated with validation rules for the `pid` resource (range 1024-16384, container-level rejection, `requests.pid` forbidden).
* `pkg/api/pod` will be updated with field stripping logic when the `PerPodPIDLimit` gate is disabled (`requests.pid` is always stripped).
* `pkg/kubelet/cm` will be updated with `getPodPIDLimit` logic (pod limit lower/higher than node, no pod limit fallback), cgroupsv1 rejection error message, `PIDLimitCapped` event emission when the effective limit is capped, and static pod PID limit enforcement (kubelet applies `getPodPIDLimit` uniformly for both regular and static pods).
* `pkg/apis/core/helper` will be updated to exclude `ResourcePID` from `standardContainerResources`.
* `pkg/kubelet/eviction` will be updated to verify that PID pressure detection, eviction ranking, and pod selection are unaffected by `spec.resources.limits.pid` when the `PerPodPIDLimit` gate is enabled.
* `k8s.io/component-helpers/nodedeclaredfeatures` will be updated to verify that the kubelet declares `PerPodPIDLimit` in `node.status.declaredFeatures` when the feature gate is enabled and cgroupsv2 is available, and does not declare it when the gate is disabled or on cgroupsv1 nodes.

`k8s.io/kubernetes/pkg/apis/core/validation`: API validation of `pid` resource
`k8s.io/kubernetes/pkg/api/pod`: Field stripping when gate is disabled
`k8s.io/kubernetes/pkg/kubelet/cm`: getPodPIDLimit enforcement logic
`k8s.io/kubernetes/pkg/apis/core/helper`: ResourcePID helper functions
`k8s.io/kubernetes/pkg/kubelet/eviction`: Eviction manager unaffected by per-pod PID limits
`k8s.io/component-helpers/nodedeclaredfeatures`: Node Declared Features registration and inference

##### Integration tests

e2e tests provide good test coverage of the interaction between the API server (validation of `pid` resource) and the kubelet (cgroup enforcement). We may add integration tests before Beta to cover API admission edge cases that are not covered by the planned unit and e2e tests.

##### e2e tests

* Pod PID limit lower than node default is applied.
* Pod PID limit higher than node default is capped at node default and a `PIDLimitCapped` event is emitted.
* Multi-container pod shares a single pod-level PID limit.
* No pod PID limit falls back to node default.
* Pod with pid limit is admitted under Baseline PSA profile and limit is enforced.
* Pod with pid limit is admitted under Restricted PSA profile and limit is enforced.
* Pod specifying `spec.resources.limits.pid` on a cgroupsv1 node is rejected during admission.
* Static pod with `spec.resources.limits.pid` has PID limit enforced at the pod cgroup level (bypasses apiserver).
* Static pod without `spec.resources.limits.pid` falls back to node-level `podPidsLimit`.
* Node enters `PIDPressure` when available PIDs fall below configured eviction thresholds, regardless of pod-level PID limits.
* Eviction manager continues to evict pods under `PIDPressure`; pod-level PID limits do not alter eviction behavior or pod selection.
* Pod-level PID limits reduce aggregate PID consumption and can prevent a workload from triggering `PIDPressure` compared to the node-level default limit.
* PID pressure calculation is based on node-level PID availability and is unaffected by the configured value of `spec.resources.limits.pid`.
* Node with `PerPodPIDLimit` enabled and cgroupsv2 declares `PerPodPIDLimit` in `node.status.declaredFeatures`.
* Pod with `spec.resources.limits.pid` is not scheduled to a node that does not declare `PerPodPIDLimit`.
* Pod with `hostPID: true` and `spec.resources.limits.pid` has the pod cgroup `pids.max` set correctly; spawning processes up to the limit succeeds and fork fails with `EAGAIN` at the limit; host processes are visible via `ps` but are not counted against the pod's cgroup PID limit.

- [PerPodPIDLimit](https://github.com/kubernetes/kubernetes/blob/master/test/e2e_node/pids_test.go): [SIG Node](https://testgrid.k8s.io/sig-node-kubelet-serial), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=PerPodPIDLimit)
- [PerPodPIDLimit PSA Compatibility](https://github.com/kubernetes/kubernetes/blob/master/test/e2e_node/pids_test.go): validates admission and enforcement under Baseline and Restricted PSA profiles

### Graduation Criteria

#### Alpha (target 1.37)

- Feature implemented behind `PerPodPIDLimit` feature gate (disabled by default)
- Kubelet enforces `min(podPidsLimit, pod.spec.resources.limits.pid)` on Pod cgroup
- Unit tests and initial node e2e tests completed
- `kubelet_pod_pid_limit_applied_total` metric exposed

#### Beta (target 1.38)

- Feature enabled by default
- Node e2e tests stable in Testgrid for at least one release
- Downgrade and upgrade testing completed
- Address feedback and bugs reported during Alpha

#### GA (target 1.40)

- At least 2 releases in Beta without major bugs
- Remove feature gate
- Conformance tests added
- No negative user feedback from production usage

### Upgrade / Downgrade Strategy

Upgrade: No changes required for existing workloads. Enable the `PerPodPIDLimit`
feature gate to start using the feature.

Downgrade: Existing Pods with `spec.resources.limits.pid` continue running with
node-level `podPidsLimit`. The field is preserved on existing objects but
rejected on new Pods.

### Version Skew Strategy

This feature involves coordination between the kube-apiserver (API validation),
the kubelet (cgroup enforcement), and the scheduler (node filtering via Node
Declared Features). The following version skew scenarios are possible during
rolling upgrades:

**Apiserver ON, kubelet OFF:** The apiserver accepts `spec.resources.limits.pid`.
The kubelet does not declare `PerPodPIDLimit` in `node.status.declaredFeatures`,
so the scheduler avoids placing the pod on this node. The node-level
`podPidsLimit` applies as a safe fallback.

**Apiserver OFF, kubelet ON:** Two scenarios:

**Regular Pods:** The apiserver silently rejects `spec.resources.limits.pid`
from new pod specs, so the field never reaches the kubelet. The kubelet
applies the node-level `podPidsLimit`.

**Static Pods:** Static pods are defined directly on the node and bypass
the apiserver. Since the kubelet has the gate enabled, if
`spec.resources.limits.pid` is present, the kubelet will enforce the
configured validation and apply the pod-level PID limit logic accordingly.

**Both ON:** Full enforcement. The kubelet declares `PerPodPIDLimit` in
`node.status.declaredFeatures`, the scheduler places pods on compatible nodes,
the apiserver validates, and the kubelet enforces `min(podPidsLimit, spec.resources.limits.pid)`.

**Both OFF:** Feature disabled, existing behavior.

Enable on apiserver and kubelet simultaneously. If rolling, enable apiserver
first. The node-level `podPidsLimit` provides a safe fallback until kubelets
are upgraded.

In clusters with mixed node versions or mixed cgroupsv1/v2 nodes, the Node
Declared Features framework
([KEP-5328](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/5328-node-declared-features))
mitigates version skew automatically. Only nodes that declare `PerPodPIDLimit`
receive pods with `spec.resources.limits.pid`, preventing kubelet admission
rejections that would put the pod into a non-retriable `Failed` state.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `PerPodPIDLimit`
  - Components depending on the feature gate: `kube-apiserver`, `kubelet`

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes; disable gate and restart components. Existing Pods retain their cgroup PID settings until restarted.

###### What happens if we reenable the feature if it was previously rolled back?

Pods can again specify `spec.resources.limits.pid`; no state loss.

###### Are there any tests for feature enablement/disablement?

Unit tests cover gate toggle: validation accepts/rejects `pid` based on gate state, and field stripping preserves `pid` on existing objects when gate is off.

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

A rollout or rollback (enabling/disabling the feature gate and restarting
components) does not impact already running workloads.

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

No. This feature adds a new optional field (`spec.resources.limits.pid`)
and does not deprecate or remove any existing fields, flags, or APIs.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

An operator can check the `kubelet_pod_pid_limit_applied_total` metric,
which increments each time the kubelet applies a per-pod PID limit from
`spec.resources.limits.pid`. A non-zero and increasing value indicates
workloads are using the feature. As a fallback, operators can query the
API for pods with the field set:
`kubectl get pods --all-namespaces -o json | jq '.items[] | select(.spec.resources.limits.pid != null) | .metadata.name'`

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event Reason: `PIDLimitCapped` — emitted when a pod's requested PID limit
    exceeds the node's `podPidsLimit`, causing the effective limit to be capped
    at the node-defined maximum.
- [ ] API .status
  - Condition name: 
  - Other field: 
- [x] Other (treat as last resort)
  - Details: The `kubelet_pod_pid_limit_applied_total` Prometheus metric tracks
    how many pods have had per-pod PID limits applied.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- 100% of pods with a valid `spec.resources.limits.pid` should have
  their pod-level cgroup `pids.max` set to the correct effective value
  (`min(podPidsLimit, spec.resources.limits.pid)`) within the normal pod
  startup time.
- No increase in pod startup latency attributable to PID limit
  enforcement (the cgroup write is a single syscall during pod setup).

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `kubelet_pod_pid_limit_applied_total`
  - [Optional] Aggregation method: counter (monotonically increasing)
  - Components exposing the metric: kubelet

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No


### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- cgroupsv2
  - Usage description: PID limits are enforced via the cgroupsv2 PID controller (`pids.max`).
    - Impact of its outage on the feature: On cgroupsv1 nodes, pods specifying `spec.resources.limits.pid` are rejected during admission.
    - Impact of its degraded performance or high-error rates on the feature: N/A — cgroup is a kernel interface, not a service.

- `PodLevelResources` feature gate (Beta, enabled by default since v1.34)
  - Usage description: `pid` is specified under `pod.spec.resources.limits`, which requires `PodLevelResources`.
    - Impact of its outage on the feature: If `PodLevelResources` is disabled, `PerPodPIDLimit` cannot be enabled.
    - Impact of its degraded performance or high-error rates on the feature: N/A — feature gate is a binary on/off.

- `NodeDeclaredFeatures` feature gate (GA since v1.36)
  - Usage description: The kubelet declares `PerPodPIDLimit` in `node.status.declaredFeatures` when the feature gate is enabled and cgroupsv2 is available. The scheduler uses this to filter nodes for pods with `spec.resources.limits.pid`.
    - Impact of its outage on the feature: If `NodeDeclaredFeatures` is disabled, the scheduler cannot filter nodes based on per-pod PID limit support. The feature still works via graceful degradation (kubelet admission rejection on incompatible nodes).
    - Impact of its degraded performance or high-error rates on the feature: N/A — feature gate is a binary on/off.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. The kubelet reads `spec.resources.limits.pid` from the existing Pod spec it
already fetches. No new API calls, watches, or controllers are introduced.

###### Will enabling / using this feature result in introducing new API types?

No. This feature adds a new field to the existing Pod API type. No new API types
are introduced.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- API type(s): Pod
- Estimated increase in size: ~20 bytes per Pod when `spec.resources.limits.pid`
  is set. Pods that do not use this field are unchanged.
- Estimated amount of new objects: None. No new API objects are created.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. The kubelet performs one additional integer comparison (`min`) during pod
cgroup setup, which is negligible.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. This feature adds one integer field read and a `min()` comparison per pod
admission. No additional in-memory state, disk I/O, or network traffic.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. This feature restricts PID usage — it can only lower the effective PID limit
for a pod, never raise it above the node's `podPidsLimit`. It reduces the risk
of PID exhaustion rather than increasing it.

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

- 2026-05-05: Initial discussion in [sig-node weekly meeting](https://docs.google.com/document/d/1Ne57gvidMEWXR70OxxnRkYquAoMpt56o75oZtg-OeBg/edit?tab=t.0#heading=h.16dqthn53k2t)
- 2026-05-06: [Enhancement issue created](https://github.com/kubernetes/enhancements/issues/6063)
- 2026-05-06: [KEP PR created](https://github.com/kubernetes/enhancements/pull/6064)
- 2026-05-25: [Alpha implementation PR created](https://github.com/kubernetes/kubernetes/pull/139277) targeting v1.37

## Drawbacks

This feature only works on cgroupsv2 nodes. Pods specifying `spec.resources.limits.pid` will be rejected on cgroupsv1 nodes, which may cause confusion in mixed clusters where some nodes have been upgraded to cgroupsv2 and others have not. Workload authors need to be aware of the underlying cgroup version to avoid unexpected admission failures.

Additionally, the "lower value wins" design means pods can only restrict PID limits below the node default, not raise them. Users who expect `pid: 8192` to override a node `podPidsLimit` of 4096 may find this counterintuitive, though this is consistent with how Kubernetes treats node-level resource limits as the administrator's ceiling.

## Alternatives

1. **ulimit nproc via KEP-5758**: A potential alternative would be to introduce support for the nproc option in ulimit as part of KEP-5758. However, because ulimit is not enforced at the cgroup level, we chose not to include this approach in KEP-5758. For additional context and design considerations, see the KEP documentation: https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/5758-per-container-ulimits-configuration.

2. Both approaches below use NRI plugin implementations that intercept container creation to adjust the `pids.max` cgroup limit. Both were tested on a Kind cluster (see [testing details](https://docs.google.com/document/d/1LQJK1AAh7zI9v4xjrkkFkWJYvyeE49RRJWtIBeqG7J8/edit?tab=t.0)).

1. **NRI plugin with annotations**: The developer puts the desired PID limit directly on the pod as an annotation. The plugin reads the annotation, parses it as an integer, and injects a `ContainerAdjustment` with `LinuxPids.Limit` set accordingly. This gives per-pod granularity but means any developer can set an arbitrary value, and there's no admission validation unless you add a webhook separately.

2. **PriorityClass-based PID limits**: An admin defines a config file mapping PriorityClass names to PID limits. The plugin reads the pod's PriorityClass and applies the corresponding limit. This is more admin-controlled but semantically overloads PriorityClass — which is meant for scheduling preemption decisions — with resource-limit behavior.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
