# KEP-5471: Extended Toleration Operators for Threshold-Based Placement

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Why not NodeAffinity alone?](#why-not-nodeaffinity-alone)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
  - [Benefits for implementing this feature for DRA and AI Workloads](#benefits-for-implementing-this-feature-for-dra-and-ai-workloads)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1 — Cluster operator using mixed on-demand and spot nodes](#story-1--cluster-operator-using-mixed-on-demand-and-spot-nodes)
    - [Story 2 — AI inference service with strict SLOs](#story-2--ai-inference-service-with-strict-slos)
    - [Story 3 — AI training workload balancing cost and reliability](#story-3--ai-training-workload-balancing-cost-and-reliability)
    - [Story 4 — DRA GPU claim management](#story-4--dra-gpu-claim-management)
    - [Story 5 — DRA device-level error budget management](#story-5--dra-device-level-error-budget-management)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Scheduler Performance Regression](#scheduler-performance-regression)
    - [API Compatibility and Version Skew](#api-compatibility-and-version-skew)
    - [Edge Cases in Numeric Parsing](#edge-cases-in-numeric-parsing)
    - [Taint Misconfiguration Detection](#taint-misconfiguration-detection)
    - [Cross-SIG Impact](#cross-sig-impact)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Semantics](#semantics)
  - [Implementation](#implementation)
    - [Feature Gate Definition](#feature-gate-definition)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Performance tests](#performance-tests)
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

## Summary

Extend **core/v1 Toleration** to support **numeric comparison operators** when matching **Node Taints**:

- New operators: `Lt`, `Gt` (in addition to existing `Equal`/`Exists`).
- Primary motivation: allow pods to opt‑in to nodes by `SLA/failure‑probability` values published as taints (e.g., `node.kubernetes.io/sla=950`).
- Scheduler impact is limited to the existing TaintToleration Filter; no new stages or algorithms.

This preserves the well‑understood safety model of taints/tolerations (eviction via`NoExecute`) while enabling threshold‑based placement similar to numeric NodeAffinity, but with better operational semantics.

## Motivation

Many clusters blend (**on‑demand/higher‑SLA**) and (**spot-preemptible/lower‑SLA**) nodes. Platform teams want a safe default keeping most workloads away from risky capacity, while allowing specific workloads to opt‑in with explicit thresholds like `SLA ≥ 95%`.

### Why not NodeAffinity alone?

For the “node SLA / failure‑probability” use‑case, NodeAffinity can express minimum or exact SLA thresholds via label comparisons, but it’s not sufficient for the operational goals here:

- **Policy orientation:** NodeAffinity is per‑pod; to keep most pods away from low‑SLA nodes you'd have to edit every workload.
- **Taints invert control**: nodes declare risk; only pods with a matching toleration may land.
- **Eviction semantics:** Affinity has no eviction. Taints support `NoExecute` with `tolerationSeconds`, letting operators drain/evict pods when a node's SLA class drops or a spot reclaim hits.
- **Operational ergonomics:** Centralized, node‑side policy is consistent with other safety taints (e.g., disk-pressure, memory-pressure). Teams opt‑in, reducing config drift.

From a scheduling perspective, adding numeric operators to tolerations only adjusts match logic. It does not change queueing, scoring, or preemption algorithms.

### Goals

- Add comparison operators to tolerations so pods can match taints like `node.kubernetes.io/sla=<int>` using thresholds.
- Keep behavior consistent with existing effects (`NoSchedule`, `PreferNoSchedule`, `NoExecute`).
- Backward compatible and opt‑in via a feature gate.
- Zero operational performance impact on existing pod scheduling using `Equal` and `Exists` operators.

### Non-Goals

- Standardizing an SLA key or unit (clusters may choose any integer scale, e.g., 950 for 95.0%).
- Implementing workload‑level "70/30" mix semantics.
- Changing NodeAffinity behavior.

### Benefits for implementing this feature for DRA and AI Workloads

In addition to general scheduling improvements, SLA‑aware opt‑in via tolerations has specific advantages for `Dynamic Resource Allocation (DRA)` and `AI/ML`:

- DRA steers GPUs/accelerators resource claims by node reliability: critical workloads get high‑SLA capacity while interruptible batch workloads use cheaper pools. Taints block risky pools and evict when capacity degrades.

- AI/ML pipelines can place latency‑sensitive inference on high‑SLA nodes while directing checkpoint-able batch workloads to run on spot nodes. When spot nodes are reclaimed, taints trigger graceful drain and controlled failover.

| Benefit                        | Impact on DRA                                                           | Impact on AI/ML workloads                                              |
| ------------------------------ | ----------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| **Cost–reliability trade-off** | Critical workloads stay on premium nodes; interruptible batch uses spot | Inference on reliable nodes; checkpoint-able training on cheaper pools |
| **Workload-aware placement**   | Different claim types target appropriate node tiers                     | Pipeline stages match their reliability requirements                   |
| **Graceful preemption**        | `NoExecute` provides controlled eviction timing                         | Predictable failover for training and serving workloads                |
| **Resource fairness**          | Prevents monopolization of premium capacity                             | Teams share reliable accelerators fairly                               |
| **Elastic scaling**            | Bursts overflow to lower-SLA pools safely                               | HPA scales to spot with clear boundaries                               |
| **Policy transparency**        | Node reliability classes are explicit and auditable                     | Platform teams enforce clear reliability tiers                         |

## Proposal

### User Stories (Optional)

#### Story 1 — Cluster operator using mixed on-demand and spot nodes

As a cluster operator, I want a default repel from spot (low-SLA) nodes so that only workloads that explicitly tolerate them can land there.

I also want to set numeric SLA thresholds in tolerations (e.g., `Gt 950`) so pods can opt-in to reliable nodes or specific SLA bands without having to hardcode every SLA class in NodeAffinity rules.

**Example Configuration:**

```yaml
# Spot nodes with 80% SLA get a repelling taint
apiVersion: v1
kind: Node
metadata:
  name: spot-node-1
spec:
  taints:
  - key: node.kubernetes.io/sla
    value: "800"
    effect: NoSchedule
---
# Cost-optimized workload explicitly tolerates SLA >= 750
apiVersion: v1
kind: Pod
spec:
  tolerations:
  - key: node.kubernetes.io/sla
    operator: Gt
    value: "750"
    effect: NoSchedule
---
apiVersion: v1
kind: Pod
metadata:
  name: flexible-sla-workload
spec:
  tolerations:
  # Accept nodes with SLA >= 900 (SLA = 900 OR SLA > 900)
  - key: node.kubernetes.io/sla
    operator: Equal
    value: "900"
    effect: NoSchedule
  - key: node.kubernetes.io/sla
    operator: Gt
    value: "900"
    effect: NoSchedule
---
# Critical workload will not be scheduled until a suitable high reliability node has capacity
apiVersion: v1
kind: Pod
metadata:
  name: critical-workload
spec:
  tolerations:
  - key: node.kubernetes.io/sla
    operator: Gt
    value: "950"
    effect: NoSchedule
```

#### Story 2 — AI inference service with strict SLOs

As an AI platform engineer, I want to ensure my latency-critical inference pods only run on nodes with SLA ≥ 95%, and I want them to be evicted if the node's SLA rating drops below that threshold.

Taints and tolerations with numeric comparisons give me this eviction capability, which NodeAffinity cannot provide.

**Example Configuration:**

```yaml
# High-SLA on-demand node
apiVersion: v1
kind: Node
metadata:
  name: ondemand-node-1
spec:
  taints:
  - key: node.kubernetes.io/sla
    value: "950"
    effect: NoExecute
---
# Inference service requires SLA >= 950 with 30s grace period
apiVersion: apps/v1
kind: Deployment
metadata:
  name: inference-service
spec:
  template:
    spec:
      tolerations:
      - key: node.kubernetes.io/sla
        operator: Gt
        value: "950"
        effect: NoExecute
        tolerationSeconds: 30
```

#### Story 3 — AI training workload balancing cost and reliability

As an ML engineer running large distributed training, I want to run most worker pods on cheaper spot GPU nodes, but keep certain roles (e.g., parameter servers, checkpoint writers) on SLA ≥ 99.9% on-demand GPUs.

With numeric tolerations, I can opt-in only the pods that are safe to run on spot, while letting the cluster's default taints repel all others.

**Example Configuration:**

```yaml
# Parameter server requires ultra-high reliability
apiVersion: v1
kind: Pod
metadata:
  name: parameter-server
spec:
  tolerations:
  - key: node.kubernetes.io/sla
    operator: Gt
    value: "999"  # 99.9% SLA
    effect: NoSchedule
  containers:
  - name: ps
    resources:
      requests:
        nvidia.com/gpu: 1
---
# Training workers can tolerate spot nodes
apiVersion: v1
kind: Pod
metadata:
  name: training-worker
spec:
  tolerations:
  - key: node.kubernetes.io/sla
    operator: Gt
    value: "800"  # 80% SLA acceptable
    effect: NoSchedule
  containers:
  - name: worker
    resources:
      requests:
        nvidia.com/gpu: 4
```

#### Story 4 — DRA GPU claim management

As a DRA driver implementer, I want to combine device resource claims with node SLA constraints so that GPU claims can only bind to nodes meeting a minimum reliability, unless the workload explicitly tolerates lower values.

This ensures DRA allocations are both resource-correct and reliability-compliant.

**Example Configuration:**

```yaml
# High-SLA GPU device published by DRA driver
apiVersion: resource.k8s.io/v1alpha4
kind: ResourceSlice
metadata:
  name: gpu-node-01-slice
spec:
  driver: nvidia.com/gpu
  pool:
    name: gpu-node-01
    generation: 1
  devices:
  - name: gpu-node-01-device-0
    basic:
      attributes:
        memory: "32Gi"
        compute-capability: "8.6"
      capacity:
        count: 1
    # Driver applies SLA taint based on node reliability metrics
    taints:
    - key: node.kubernetes.io/sla
      value: "980"  # 98% SLA
      effect: NoSchedule
---
# DRA claim with SLA constraints  
apiVersion: resource.k8s.io/v1alpha4
kind: ResourceClaim
metadata:
  name: gpu-claim-high-sla
spec:
  devices:
    requests:
    - name: gpu
      deviceClassName: nvidia-a100
      tolerations:
      # Only accept GPUs with SLA >= 950 (95%)
      - key: node.kubernetes.io/sla
        operator: Gt
        value: "950"
        effect: NoSchedule
---
# Pod using DRA claim with SLA requirements
apiVersion: v1
kind: Pod
metadata:
  name: dra-workload
spec:
  resourceClaims:
  - name: gpu-claim
    resourceClaimName: gpu-claim-high-sla
  tolerations:
  - key: node.kubernetes.io/sla
    operator: Gt
    value: "950"  # Ensure GPU nodes meet SLA requirements
    effect: NoSchedule
  containers:
  - name: ml-workload
    resources:
      claims:
      - name: gpu-claim
```

#### Story 5 — DRA device-level error budget management

As a platform engineer managing GPU clusters with varying reliability states, I want to allocate devices based on their remaining error budget using numeric tolerations. So that critical workloads only get devices with sufficient reliability headroom while allowing degraded devices to serve less sensitive workloads.

This will get the critical inference fresh devices (>24h error budget), batch training can use aging devices (1-24h), and severely degraded devices (<1h) are excluded from allocation entirely, enabling graceful device lifecycle management.

**Example Configuration:**

```yaml
# Driver taints devices with low error budget
kind: ResourceSlice
spec:
  driver: device.example.com
  devices:
  - name: gpu-node-01-device-0
    attributes:
      memory: "32Gi"
      compute-capability: "8.6"
    # Driver applies taint when error budget drops below 10 hours
    taints:
    - key: device.example.com/error-budget-in-hours
      value: "8"  # 8 hours remaining
      effect: NoSchedule
---
# Critical inference workload requires high-reliability devices
kind: ResourceClaim
metadata:
  name: inference-gpu-claim
spec:
  requests:
  - name: high-reliability-gpu
    deviceClassName: device.example.com
    tolerations:
    # Only accept devices with >24 hours error budget
    - key: device.example.com/error-budget-in-hours
      operator: Gt
      value: "24"
      effect: NoSchedule
---
# Batch Short-lived batch training workload tolerates degraded devices
kind: ResourceClaim
metadata:
  name: training-gpu-claim
spec:
  requests:
  - name: batch-gpu
    deviceClassName: device.example.com
    tolerations:
    # Accept devices with >1 hour error budget
    - key: device.example.com/error-budget-in-hours
      operator: Gt
      value: "1"
      effect: NoSchedule
```

### Notes/Constraints/Caveats (Optional)

- **Integer-Only Support**: The implementation supports signed 64-bit integers only. Pod specs containing toleration values with decimal numbers (e.g., `"95.5"`) will be rejected by the API server during validation when using numeric comparison operators.

- **Parsing Requirements**: The toleration value must be parseable as integers for numeric operators (`Lt`, `Gt`). If fails parsing, the toleration does not match.

  > Note: A taint like `foo=95.5:NoSchedule` is valid since taint values follow label values syntax, which allows. The numeric parsing/validation is enforced on toleration *only*.

- **Alpha Restrictions**: When `TaintTolerationComparisonOperators=false`, the API server rejects pods using the new operators.

- **Strict Validation**: Unlike existing `Equal`/`Exists` operators which accept any string values, numeric operators require valid integer strings. This may catch existing invalid configurations.

- **Leading Zeros Validation**: The API validation will reject taint and toleration values that contain leading zeros (e.g., `"0950"`, `"007"`) when used with numeric operators (`Lt`, `Gt`). This ensures consistent behavior and prevents the ambiguity between string and numeric interpretations. Only values without leading zeros are accepted (e.g., `"950"`, `"7"`).

- **Parsing Overhead**: Each taint/toleration match with numeric operators requires integer parsing.

- Invalid taints meant to be used with the new comparison operators (e.g., `node.kubernetes.io/sla=95.5` and `node.kubernetes.io/version=1`) are not detected at admission time.

- **Taint Misconfiguration Risk**: When nodes have taints with non-numeric values (e.g., `node.kubernetes.io/sla=high` instead of `node.kubernetes.io/sla=950`) that are intended for use with numeric operators, the misconfiguration is only detected during pod scheduling attempts, not at taint creation time. This can lead to scheduling failures that are difficult to diagnose.

### Risks and Mitigations

#### Scheduler Performance Regression

**Risk**: Integer parsing during taint/toleration matching could degrade scheduler performance, especially in clusters with thousands of taints.

**Mitigation**:

- Parse integers only when new operators are used.
- Existing `Equal`/`Exists` operators execute identical code paths with no additional overhead.
- Consider caching parsed values in scheduler data structures if performance issues arise
- Feature gate allows disabling if performance problems occur

#### API Compatibility and Version Skew

**Risk**: Pods using new operators cannot be scheduled if some schedulers don't support the feature, creating deployment failures during upgrades.

**Mitigation**:

- Feature gate prevents usage until all components are upgraded
- Clear upgrade documentation specifying component upgrade order
- Backward compatibility testing ensures existing workloads continue functioning
- Gradual rollout recommendations for production clusters

#### Edge Cases in Numeric Parsing

**Risk**: Unexpected behavior with edge cases like integer overflow, leading zeros, or malformed input could cause scheduling failures. Leading zeros in values (e.g., `"0950"`) could create user confusion about whether values are treated as strings or numbers.

**Mitigation**:

- Use Go's standard `strconv.ParseInt()` with well-defined error handling
- Comprehensive unit tests covering edge cases (overflow, underflow, malformed strings, leading zeros)
- API validation rejects pods with unparseable values rather than silently failing
- **API validation explicitly rejects values with leading zeros** when using numeric operators to eliminate confusion
- Clear error messages help users identify and fix configuration issues
- Documentation clearly states that leading zeros are not permitted for numeric operators
- **Performance validation via scheduler-perf tests** to ensure no measurable scheduling latency degradation from integer parsing overhead

#### Taint Misconfiguration Detection

**Risk**: Node taints intended for numeric comparison may contain non-numeric values (e.g., `node.kubernetes.io/sla=high` instead of `node.kubernetes.io/sla=950`), causing scheduling failures that are only detected during pod placement attempts rather than at taint creation time.

**Mitigation**:

- Clear documentation and examples showing proper numeric taint configuration
- Enhanced error messages in scheduling events that clearly indicate parsing failures
- Scheduler logging for taint parsing failures to help cluster admins identify misconfigured nodes even when pods successfully schedule on other nodes with valid numeric taints
- Monitoring and alerting on scheduling failures due to taint parsing errors

#### Cross-SIG Impact

- SIG-Node
- SIG-Apps
- SIG-Cluster-Lifecycle
- WG-Node-Lifecycle
- WG-Device-Management

## Design Details

### API Changes

**File**: `staging/src/k8s.io/api/core/v1/types.go`

Extend `core/v1.Toleration.Operator` to accept, in addition to `Equal` and `Exists`:

- `Lt`: match if toleration.value < taint.value
- `Gt`: match if toleration.value > taint.value
- `Equal`/`Exists`: Remain unchanged

```go
// TolerationOperator is the set of operators that can be used in a toleration.
type TolerationOperator string

const (
    TolerationOpEqual  TolerationOperator = "Equal"
    TolerationOpExists TolerationOperator = "Exists"
    
    // New numeric comparison operators (feature-gated)
    TolerationOpLt TolerationOperator = "Lt"    // Less than
    TolerationOpGt TolerationOperator = "Gt"    // Greater than
)
```

### Semantics

- To honor Kubernetes APIs that avoids floating-point numbers where possible due to precision and parsing issues, The new toleration operators will be introduced as integers (i.e.; 950 = 95.0%, 999 = 99.9%, 800 = 80.0%).
- For `PreferNoSchedule` taints, numeric operators only determine whether the taint is considered as tolerated for scoring:

- **Tolerated taints**: Do not count against the node's score.
- **Intolerated taints**: Count against the node's score.
- **Scoring**: Unchanged - nodes with fewer intolerable `PreferNoSchedule` taints receive higher scores.

This maintains consistent soft-preference behavior while enabling threshold-based SLA matching. For example, A pod requiring SLA > 95% will prefer nodes with SLA ≥ 950 over nodes with SLA < 950, but won't be blocked from scheduling on lower-SLA nodes if higher-SLA capacity is unavailable.

### Implementation

#### Feature Gate Definition

**File**: `pkg/features/kube_features.go`

```go
const (
    // TaintTolerationComparisonOperators enables numeric comparison operators (Lt, Gt) for tolerations
    TaintTolerationComparisonOperators featuregate.Feature = "TaintTolerationComparisonOperators"
)

var defaultKubernetesFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
    TaintTolerationComparisonOperators: {Default: false, PreRelease: featuregate.Alpha},
}
```

**1. API Validation** - `pkg/apis/core/validation/validation.go`

```go
func validateTolerations(tolerations []core.Toleration, fldPath *field.Path) field.ErrorList {
    allErrors := field.ErrorList{}
    
    for i, toleration := range tolerations {
        idxPath := fldPath.Index(i)
        
        // Existing validation...
        
        // New: Validate numeric operators (feature-gated)
        switch toleration.Operator {
        case core.TolerationOpLt, core.TolerationOpGt:
            if !utilfeature.DefaultFeatureGate.Enabled(features.TaintTolerationComparisonOperators) {
                allErrors = append(allErrors, field.Invalid(idxPath.Child("operator"), 
                    toleration.Operator, "numeric operators require TaintTolerationComparisonOperators feature gate"))
                continue
            }
            
            // Validate value is parseable as int64
            if _, err := strconv.ParseInt(toleration.Value, 10, 64); err != nil {
                allErrors = append(allErrors, field.Invalid(idxPath.Child("value"),
                    toleration.Value, "value must be a valid integer for numeric operators"))
                continue
            }
            
            // Reject values with leading zeros to prevent confusion
            if len(toleration.Value) > 1 && toleration.Value[0] == '0' && toleration.Value != "0" {
                allErrors = append(allErrors, field.Invalid(idxPath.Child("value"),
                    toleration.Value, "leading zeros are not allowed in numeric values (use '950' instead of '0950')"))
            }
        }
    }
    return allErrors
}
```

**2. Scheduler Logic** - `staging/src/k8s.io/component-helpers/scheduling/corev1/helpers.go`

```go
// ToleratesTaint checks if the toleration tolerates the taint.
func (t *Toleration) ToleratesTaint(taint *Taint) (bool, error) {
     switch t.Operator {
    // Existing key and effect matching logic...
    
    // Handle existing operators first. This ensures
    // zero performance impact for existing Equal/Exists scenarios.
    case TolerationOpLt, TolerationOpGt:
        // Feature gate check is not needed here as validation already handles it
        // Only parse values when comparison operators are actually used
        return compareValues(t.Value, taint.Value, t.Operator)
    default:
        return false, errors.New("cannot handle the operator")
    }
}

// return error to inform the user what went wrong, not only that the toleration is not matching for any node.
func compareValues(tolerationVal, taintVal string, op TolerationOperator) (bool, error) {
    tVal, tErr := strconv.ParseInt(tolerationVal, 10, 64)
    if tErr != nil {
        return false, tErr // Invalid toleration value
    }
    
    nVal, nErr := strconv.ParseInt(taintVal, 10, 64)  
    if nErr != nil {
        // Log taint parsing failures to help cluster admins identify misconfigured nodes
        // even when pods can still schedule on other nodes with valid numeric taints
        klog.Warningf("Failed to parse taint value %q as integer for numeric comparison: %v", taintVal, nErr)
        return false, nErr // Invalid taint value
    }
    
    switch op {
    case TolerationOpLt:
        return tVal < nVal, nil
    case TolerationOpGt:
        return tVal > nVal, nil
    }
}
```

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

N/A

##### Unit tests

All core changes must be covered by unit tests, in both Taint API, validation, and scheduler sides. Tests must specifically cover leading zeros behavior (e.g., `"0950"` vs `"950"`):

- `staging/src/k8s.io/api/core/v1/toleration_test.go`: Sep-16-2025 - 66.7%
- `staging/src/k8s.io/component-helpers/scheduling/corev1/helpers_test.go`: Sep-16-2025 - 100%
- `pkg/apis/core/validation/validation_test.go`: Sep-16-2025 - 85.1%
- `pkg/scheduler/framework/plugins/tainttoleration/taint_toleration_test.go`: Sep-16-2025 - 86.9%

##### Performance tests

- Establish current scheduling latency for workloads using only `Equal`/`Exists` operators
- Verify that enabling the feature gate with no comparison operators used shows no measurable performance difference.
- **Scheduler Performance Tests:** will be extended to cover the new taints cases introduced in this KEP:(test/integration/scheduler_perf)

##### Integration tests

The following scenarios need to be covered in integration tests:

- Feature gate's enabling/disabling
- **Scheduler Integration Tests:** will be extended to cover the new taints cases introduced in this KEP:(test/integration/scheduler)

##### e2e tests

The existing e2e tests will be extended to cover the new taints cases introduced in this KEP:

- **Node Taints e2e Tests:** (test/e2e/node/taints.go)
- **Scheduler Taints e2e Tests:** (test/e2e/scheduling)

### Graduation Criteria

#### Alpha

- Feature implemented behind `TaintTolerationComparisonOperators` feature gate (disabled by default)
- API validation for numeric operators in place
- Taint/toleration matching logic supports `Lt`, `Gt` operators  

#### Beta

- Feature enabled by default
- Feedback collected from early adopters in SIG-Scheduling
- Performance testing shows that there is no significant scheduler latency increase nor memory usage increase.
- Implement feature for DRA APIs
- Stress testing.

#### GA

- Evidence of real-world adoption.
- Complete scalability validation.

### Upgrade / Downgrade Strategy

- Upgrade
  - Enable the feature gate in both API Server and Scheduler.
- Downgrade
  - Disable the feature gate in both API Server and Scheduler

### Version Skew Strategy

The skew between kubelet and control-plane components are not impacted. The kube-scheduler is expected to match the kube-apiserver minor version, but may be up to one minor version older (to allow live upgrades).

In the release it's been added, the feature is disabled by default and not recognized by other components.
Whoever enabled the feature manually would take the risk of component like kube-scheduler being old and not recognize the fields.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `TaintTolerationComparisonOperators`
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-scheduler

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

###### What happens if we reenable the feature if it was previously rolled back?

SLA toleration will be respected again.

###### Are there any tests for feature enablement/disablement?

Tests have been added in the integration tests. See [Integration tests](#integration-tests) for more details.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

It shouldn't impact already running workloads. It's an opt-in feature.

###### What specific metrics should inform a rollback?

- `scheduler_scheduling_duration_seconds`
- `scheduler_scheduling_attempts_total`
- `apiserver_request_total`

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be considered for beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

1. **Metrics**:

   ```promql
   # Number of pods using numeric tolerations
   scheduler_numeric_tolerations_total > 0
   
   # Rate of numeric comparison operations
   rate(scheduler_framework_extension_point_duration_seconds{plugin="TaintToleration"}[5m])
   ```

2. **API Queries**:

   ```bash
   # Check for pods with numeric toleration operators
   kubectl get pods -A -o jsonpath='{range .items[*]}{.metadata.name}{": "}{.spec.tolerations[?(@.operator=="Gt")]}{"\n"}{end}' | grep -v "^[^:]*: *$"
   
   # Count nodes with numeric taints (SLA example)
   kubectl get nodes -o jsonpath='{range .items[*]}{.spec.taints[?(@.key=="node.kubernetes.io/sla")]}{"\n"}{end}' | wc -l
   ```

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event Reason: FailedScheduling
  - Event Message: "node(s) had untolerated taint `node.kubernetes.io/sla`: `950`"
- [x] API .spec.taints
  - Other field: `key: node.kubernetes.io/sla`
- [x] API .spec.tolerations
  - Other field: `node.kubernetes.io/sla`

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name:
    - `scheduler_scheduling_attempts_total`
    - `scheduler_framework_extension_point_duration_seconds`
    - Components exposing the metric: `kube-scheduler`
  - Metric name:
    - `kube_pod_status_phase`
    - `kube_pod_status_scheduled_time`
    - Components exposing the metric: `kube-apiserver`

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Yes, a new metrics:

- `scheduler_numeric_taint_evaluations_total`: tracks each numeric evaluation with its result.
- `scheduler_numeric_tolerations_total`: tracks successful scheduling with numeric tolerations.
These metrics provide visibility into:

1. How frequently the numeric toleration feature is being used
2. The effectiveness of numeric taint/toleration matching
3. Per-profile usage patterns for multi-scheduler setups

In addition, the scheduler has an existing `scheduler_unschedulable_pods` metric that handles the multiple failure reasons by incrementing for each plugin that rejects a pod.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

N/A

### Scalability

###### Will enabling / using this feature result in any new API calls?

No, the feature is designed to be an enhancement to existing logic without introducing any new API communication patterns.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Potentially yes, but the impact should be **minimal**. The numeric toleration operators feature could slightly increase time for operations covered by existing SLIs/SLOs due to integer parsing overhead and validation overhead.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Same as existing taint/toleration system which is graceful degradation.

###### What are other known failure modes?

A failure mode due to numeric toleration operators have integer parsing errors from malformed taint/toleration values causing pods to be rejected with clear error messages.

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

- 2025-08-11: Initial KEP

## Drawbacks

## Alternatives

There are many different alternatives were considered:

1. **New Dedicated SLA API Resource:**  Create `SLAPolicy` CRD
   - **Pros:** Clean separation, rich policy definitions.
   - **Cons:** New API surface, additional complexity, breaks unified taint/toleration model.
2. **Custom Scheduler Plugin:** Use scheduling plugin with SLA-aware logic, [placement-policy-scheduler-plugins](https://github.com/Azure/placement-policy-scheduler-plugins)
   - **Pros:** Full scheduling control, rich logic possible
   - **Cons:**
     - Out-of-tree scheduler plugin to maintain and manage
     - Doesn't leverage existing taint/toleration infrastructure.
3. **Node Labels + Enhanced NodeAffinity:** Use labels instead of taints, extend NodeAffinity matching.
   - **Pros:** Leverages existing label system.
   - **Cons:**
     - No default push-back behavior
     - No eviction semantics
     - Labels aren't meant for operational constraints.

## Infrastructure Needed (Optional)
