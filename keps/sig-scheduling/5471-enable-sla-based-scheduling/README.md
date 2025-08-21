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
    - [Story 6 — Kubernetes version compatibility for critical workloads](#story-6--kubernetes-version-compatibility-for-critical-workloads)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Scheduler Performance Regression](#scheduler-performance-regression)
    - [API Compatibility and Version Skew](#api-compatibility-and-version-skew)
    - [Edge Cases in Value Parsing](#edge-cases-in-value-parsing)
    - [Cross-SIG Impact](#cross-sig-impact)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
    - [Toleration Operator Extensions](#toleration-operator-extensions)
    - [CEL Semver Library Extensions](#cel-semver-library-extensions)
  - [Semantics](#semantics)
  - [Implementation](#implementation)
    - [Feature Gate Definition](#feature-gate-definition)
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

Extend **Node/Device Toleration** to support **numeric comparison operators** when matching **Node/Device Taints**:

**Toleration Enhancements:**

- New operators: `Lt`, `Gt` (in addition to existing `Equal`/`Exists`)
- Automatic type detection: integer comparison for numeric values, semantic version comparison for semver values
- Primary motivation: allow pods to opt‑in to nodes by `SLA/failure‑probability` values or version requirements

**CEL Enhancements:**

- Extended semver library with additional comparison and utility functions
- Enhanced integration with Kubernetes version parsing standards
- Consistent semantic version handling across validation policies

This creates a unified approach to threshold-based and version-based placement/validation across Kubernetes, preserving the well‑understood safety model of taints/tolerations while enabling sophisticated policy enforcement through CEL expressions.

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
- Enable automatic type detection for integer and semantic version comparisons in tolerations.
- Enhance CEL semver library with additional comparison functions for consistent version handling.
- Keep behavior consistent with existing effects (`NoSchedule`, `PreferNoSchedule`, `NoExecute`).
- Provide unified semantic version handling across scheduling and admission control.
- Backward compatible and opt‑in via feature gates.

### Non-Goals

- Standardizing an SLA key or unit (clusters may choose any integer scale, e.g., 950 for 95.0%).
- Implementing workload‑level "70/30" mix semantics.
- Changing NodeAffinity behavior.
- Replacing existing CEL semver functions (maintaining backward compatibility).
- Adding new taint/toleration keys (focusing on operator extensions only).

### Benefits for implementing this feature for DRA and AI Workloads

In addition to general scheduling improvements, SLA‑aware opt‑in via tolerations has specific advantages for `Dynamic Resource Allocation (DRA)` and `AI/ML`:

- DRA steers GPUs/accelerators resource claims by node reliability: critical workloads get high‑SLA capacity while batch workloads use cheaper pools. Taints block risky pools and evict when capacity degrades.

- AI/ML pipelines can place latency‑sensitive inference on high‑SLA nodes while directing batch to run on spot nodes. When spot nodes are reclaimed, taints trigger graceful drain and controlled failover.

| Benefit                        | Impact on DRA                                             | Impact on AI/ML workloads                               |
| ------------------------------ | --------------------------------------------------------- | ------------------------------------------------------- |
| **Cost–reliability trade-off** | Critical workloads stay on premium nodes; batch uses spot | Inference on reliable nodes; training on cheaper pools  |
| **Workload-aware placement**   | Different claim types target appropriate node tiers       | Pipeline stages match their reliability requirements    |
| **Graceful preemption**        | `NoExecute` provides controlled eviction timing           | Predictable failover for training and serving workloads |
| **Resource fairness**          | Prevents monopolization of premium capacity               | Teams share reliable accelerators fairly                |
| **Elastic scaling**            | Bursts overflow to lower-SLA pools safely                 | HPA scales to spot with clear boundaries                |
| **Policy transparency**        | Node reliability classes are explicit and auditable       | Platform teams enforce clear reliability tiers          |

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
# Batch training workload tolerates degraded devices
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

#### Story 6 — Kubernetes version compatibility for critical workloads

As a cluster operator managing a mixed-version Kubernetes cluster during rolling upgrades, I want to ensure critical workloads only run on nodes with Kubernetes version >= 1.20.0 due to specific API features they require, while allowing development workloads to tolerate older versions.

Semantic version tolerations provide precise version-based placement without hardcoding specific versions in NodeAffinity rules.

**Example Configuration:**

```yaml
# Older node running Kubernetes 1.19.5
apiVersion: v1
kind: Node
metadata:
  name: old-node-1
spec:
  taints:
  - key: node.kubernetes.io/version
    value: "1.19.5"
    effect: NoSchedule
---
# Production workload requires Kubernetes >= 1.20.0
apiVersion: apps/v1
kind: Deployment
metadata:
  name: production-api
spec:
  template:
    spec:
      tolerations:
      - key: node.kubernetes.io/version
        operator: SemverGt
        value: "1.20.0"
        effect: NoSchedule
      containers:
      - name: api
        image: myapp:latest
---
# Development workload tolerates older versions
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dev-workload
spec:
  template:
    spec:
      tolerations:
      - key: node.kubernetes.io/version
        operator: SemverGt
        value: "1.33.0"  # Can run on 1.33+ nodes
        effect: NoSchedule
```

### Notes/Constraints/Caveats (Optional)

**Value Format and Type Detection:**

- **Automatic Type Detection**: The system automatically determines whether to use integer or semantic version comparison based on value format. Both toleration and taint values must be parseable as the same type for comparison to occur.

- **Integer Comparison Requirements**:
  - Values must be parseable as signed 64-bit integers
  - Decimal numbers (e.g., `"95.5"`) are not supported to honor Kubernetes' avoidance of floating-point precision issues
  - Leading zeros are preserved in parsing (`"0950"` vs `"950"` are numerically equal)

- **Semantic Version Requirements**:
  - Values must conform to [Semantic Versioning 2.0.0](https://semver.org/) specification requiring exactly 3 components (major.minor.patch)
  - Supports both prefixed (`"v1.20.1"`) and non-prefixed (`"1.20.1"`) formats following Kubernetes conventions  
  - Invalid semver strings (e.g., `"1.20"`, `"1.20.1.1"`, `"1"`) cause validation errors
  - Pre-release versions (e.g., `1.20.0-alpha.1`) have lower precedence than release versions (`1.20.0`)
  - Build metadata (`+build.123`) is ignored in comparisons per semver specification

- **Type Mismatch Handling**: If toleration and taint values cannot be parsed as the same type (integer vs semver), the toleration does not match. This prevents unexpected behavior and ensures type safety.

**General Constraints:**

- **Alpha Restrictions**: When `TaintTolerationComparisonOperators=false`, the API server rejects pods using the new operators.

- **Strict Validation**: Unlike existing `Equal`/`Exists` operators which accept any string values, comparison operators require properly formatted values. This may catch existing invalid configurations.

- **Parsing Overhead**: Each taint/toleration match with comparison operators requires parsing attempts (integer first, then semver), adding computational cost.

  > Note: A taint like `foo=95.5:NoSchedule` or `foo=v1.20.1:NoSchedule` is valid since taint values follow label values syntax. The type detection and parsing occurs during toleration matching, not validation.

### Risks and Mitigations

#### Scheduler Performance Regression

**Risk**: Value parsing (integer and semantic version) during taint/toleration matching could degrade scheduler performance, especially in clusters with thousands of taints.

**Mitigation**:

- Parsing only occurs when comparison operators are used (no impact on existing workloads)
- Integer parsing is attempted first (fast path) before trying semver parsing
- Consider caching parsed values in scheduler data structures if performance issues arise
- Feature gate allows disabling if performance problems occur

#### API Compatibility and Version Skew

**Risk**: Pods using new operators cannot be scheduled if some schedulers don't support the feature, creating deployment failures during upgrades.

**Mitigation**:

- Feature gate prevents usage until all components are upgraded
- Clear upgrade documentation specifying component upgrade order
- Backward compatibility testing ensures existing workloads continue functioning
- Gradual rollout recommendations for production clusters

#### Edge Cases in Value Parsing

**Risk**: Unexpected behavior with edge cases like integer overflow, leading zeros, malformed semver, or type mismatches could cause scheduling failures.

**Mitigation**:

- Use Go's standard `strconv.ParseInt()` and Kubernetes' `utilversion.ParseSemantic()` with well-defined error handling
- Comprehensive unit tests covering edge cases (overflow, underflow, malformed strings, type mismatches)
- API validation accepts values that are parseable as either integer or semver
- Clear error messages help users identify and fix configuration issues
- Type mismatch handling prevents unexpected cross-type comparisons

#### Cross-SIG Impact

- SIG-Node
- SIG-Apps
- SIG-Cluster-Lifecycle
- WG-Node-Lifecycle
- WG-Device-Management

## Design Details

### API Changes

#### Toleration Operator Extensions

**File**: `staging/src/k8s.io/api/core/v1/types.go`

Extend `core/v1.Toleration.Operator` to accept, in addition to `Equal` and `Exists`:

- `Lt`: match if toleration.value < taint.value (supports both integer and semantic version comparison)
- `Gt`: match if toleration.value > taint.value (supports both integer and semantic version comparison)
- `Equal`/`Exists`: Remain unchanged

**Comparison Type Detection:**
The system automatically determines comparison type based on value format:

1. **Integer comparison**: If both toleration and taint values can be parsed as integers
2. **Semantic version comparison**: If both values can be parsed as semantic versions but not as integers
3. **No match**: If values cannot be parsed consistently or there's a type mismatch

```go
// TolerationOperator is the set of operators that can be used in a toleration.
type TolerationOperator string

const (
    TolerationOpEqual  TolerationOperator = "Equal"
    TolerationOpExists TolerationOperator = "Exists"
    
    // Numeric operators (feature-gated)
    // Support both integer and semantic version comparison based on value format
    TolerationOpLt TolerationOperator = "Lt"    // Less than
    TolerationOpGt TolerationOperator = "Gt"    // Greater than
)
```

#### CEL Semver Library Extensions

**File**: `staging/src/k8s.io/apiserver/pkg/cel/library/semverlib.go`

Extend the existing CEL semver library with additional comparison functions that align with toleration operators:

```go
// New CEL semver functions to add:

// isGreaterThanOrEqual: Returns true if the receiver is greater than or equal to the operand
// isLessThanOrEqual: Returns true if the receiver is less than or equal to the operand

// Usage in CEL expressions:
//   semver("1.20.1").isGreaterThanOrEqual(semver("1.20.0"))    // true
//   semver("1.19.5").isLessThanOrEqual(semver("1.20.0"))       // true

// Enhanced parsing with Kubernetes conventions:
//   isSemverK8s(string): Validates using Kubernetes semantic version requirements
//   semverK8s(string): Parses using Kubernetes version parsing logic
```

**File**: `staging/src/k8s.io/apiserver/pkg/cel/library/library.go`

```go
// Register enhanced semver library by default
func ExtensionLibs(env *cel.Env) cel.EnvOption {
    return cel.Lib(&ExtensionLibs{
        Version: math.MaxUint32,
        SemverExtensions: true, // Enable enhanced semver functions
    })
}
```

### Semantics

**Comparison Operators (`Lt`, `Gt`) with Automatic Type Detection:**

The operators automatically determine comparison type based on value format:

**Integer Comparison Mode:**

- Triggered when both toleration and taint values can be parsed as signed 64-bit integers
- To honor Kubernetes APIs that avoid floating-point numbers, uses integers (i.e.; 950 = 95.0%, 999 = 99.9%, 800 = 80.0%)
- Uses Go's `strconv.ParseInt()` for parsing
- Follows standard integer ordering rules

**Semantic Version Comparison Mode:**

- Triggered when both values can be parsed as semantic versions but not as integers
- Values must follow [Semantic Versioning 2.0.0](https://semver.org/) specification (e.g., `1.20.1`, `1.19.0-alpha.1`, `2.0.0-beta.1+build.123`)
- Comparison follows semver precedence rules:
  - Major version takes highest precedence
  - Minor version takes second precedence  
  - Patch version takes third precedence
  - Pre-release versions have lower precedence than release versions
  - Build metadata is ignored in precedence comparisons
- Uses Kubernetes' own semantic version parsing (`k8s.io/apimachinery/pkg/util/version.ParseSemantic`) which supports the "v" prefix

**Type Mismatch Handling:**

- If toleration value is integer but taint value is semver (or vice versa): no match
- If either value cannot be parsed as integer or semver: no match
- This ensures type safety and prevents unexpected behavior

**PreferNoSchedule Behavior:**
For `PreferNoSchedule` taints, comparison operators determine whether the taint is considered tolerated for scoring:

- **Tolerated taints**: Do not count against the node's score.
- **Intolerated taints**: Count against the node's score.
- **Scoring**: Unchanged - nodes with fewer intolerable `PreferNoSchedule` taints receive higher scores.

This maintains consistent soft-preference behavior while enabling both threshold-based SLA matching and version-based placement. For example:

- A pod with `operator: Gt, value: "950"` will prefer nodes with SLA ≥ 950 over nodes with SLA < 950 (integer comparison)
- A pod with `operator: Gt, value: "1.20.0"` will prefer nodes running 1.20.x or higher over nodes running 1.19.x (semver comparison)

### Implementation

#### Feature Gate Definition

**File**: `pkg/features/kube_features.go`

```go
const (
    // TaintTolerationComparisonOperators enables numeric and semantic version comparison operators for tolerations
    TaintTolerationComparisonOperators featuregate.Feature = "TaintTolerationComparisonOperators"
    
    // CELSemverExtensions enables enhanced semantic version functions in CEL expressions
    CELSemverExtensions featuregate.Feature = "CELSemverExtensions"
)

var defaultKubernetesFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
    TaintTolerationComparisonOperators: {Default: false, PreRelease: featuregate.Alpha},
    CELSemverExtensions: {Default: false, PreRelease: featuregate.Alpha},
}
```

**1. API Validation** - `pkg/apis/core/validation/validation.go`

```go
func validateTolerations(tolerations []core.Toleration, fldPath *field.Path) field.ErrorList {
    allErrors := field.ErrorList{}
    
    for i, toleration := range tolerations {
        idxPath := fldPath.Index(i)
        
        // Existing validation...
        
        // Validate comparison operators (feature-gated)
        switch toleration.Operator {
        case core.TolerationOpLt, core.TolerationOpGt:
            if !utilfeature.DefaultFeatureGate.Enabled(features.TaintTolerationComparisonOperators) {
                allErrors = append(allErrors, field.Invalid(idxPath.Child("operator"), 
                    toleration.Operator, "comparison operators require TaintTolerationComparisonOperators feature gate"))
                continue
            }
            
            // Validate value is parseable as either int64 or semantic version
            isValidInt := false
            isValidSemver := false
            
            if _, err := strconv.ParseInt(toleration.Value, 10, 64); err == nil {
                isValidInt = true
            }
            
            if _, err := utilversion.ParseSemantic(toleration.Value); err == nil {
                isValidSemver = true
            }
            
            if !isValidInt && !isValidSemver {
                allErrors = append(allErrors, field.Invalid(idxPath.Child("value"),
                    toleration.Value, "value must be a valid integer or semantic version for comparison operators"))
            }
        }
    }
    return allErrors
}
```

**2. Scheduler Logic** - `staging/src/k8s.io/component-helpers/scheduling/corev1/helpers.go`

```go
// ToleratesTaint checks if the toleration tolerates the taint.
func (t *Toleration) ToleratesTaint(taint *Taint) bool {
    // Existing key and effect matching logic...
    
    switch t.Operator {
    // ...
    case TolerationOpLt, TolerationOpGt:
        // Feature gate check is not needed here as validation already handles it
        return compareValues(t.Value, taint.Value, t.Operator)
    default:
        return false
    }
}

func compareValues(tolerationVal, taintVal string, op TolerationOperator) bool {
    // Try integer comparison first
    if tVal, tErr := strconv.ParseInt(tolerationVal, 10, 64); tErr == nil {
        if nVal, nErr := strconv.ParseInt(taintVal, 10, 64); nErr == nil {
            return compareIntegers(tVal, nVal, op)
        }
        // Toleration is int but taint is not - type mismatch
        return false
    }
    
    // Try semantic version comparison
    if tVer, tErr := utilversion.ParseSemantic(tolerationVal); tErr == nil {
        if nVer, nErr := utilversion.ParseSemantic(taintVal); nErr == nil {
            return compareSemvers(tVer, nVer, op)
        }
        // Toleration is semver but taint is not - type mismatch
        return false
    }
    
    // Neither integer nor semver
    return false
}

func compareIntegers(tolerationVal, taintVal int64, op TolerationOperator) bool {
    switch op {
    case TolerationOpLt:
        return tolerationVal < taintVal
    case TolerationOpGt:
        return tolerationVal > taintVal
    default:
        return false
    }
}

func compareSemvers(tolerationVer, taintVer *utilversion.Version, op TolerationOperator) bool {
    switch op {
    case TolerationOpLt:
        return tolerationVer.LessThan(taintVer)
    case TolerationOpGt:
        return tolerationVer.GreaterThan(taintVer)
    default:
        return false
    }
}
```

**3. CEL Semver Library Extensions** - `staging/src/k8s.io/apiserver/pkg/cel/library/semverlib.go`

```go
// Enhanced semver comparison functions
func registerSemverExtensions(registry *cel.Registry) {
    // Add isGreaterThanOrEqual function
    registry.Register(
        "isGreaterThanOrEqual",
        cel.MemberOverload("semver_is_greater_than_or_equal_semver", 
            []*cel.Type{semverType, semverType}, cel.BoolType,
            cel.BinaryBinding(func(lhs, rhs ref.Val) ref.Val {
                left := lhs.(*SemverObject)
                right := rhs.(*SemverObject) 
                return types.Bool(left.version.GTE(right.version))
            })),
    )
    
    // Add isLessThanOrEqual function  
    registry.Register(
        "isLessThanOrEqual",
        cel.MemberOverload("semver_is_less_than_or_equal_semver",
            []*cel.Type{semverType, semverType}, cel.BoolType,
            cel.BinaryBinding(func(lhs, rhs ref.Val) ref.Val {
                left := lhs.(*SemverObject)
                right := rhs.(*SemverObject)
                return types.Bool(left.version.LTE(right.version))
            })),
    )
    

}

// Kubernetes-compatible semver parsing
func semverK8s(str string) (*SemverObject, error) {
    // Use Kubernetes' own version parser for consistency
    version, err := utilversion.ParseSemantic(str)
    if err != nil {
        return nil, err
    }
    
    // Convert to blang/semver for CEL compatibility
    semverStr := version.String()
    blangVersion, err := semver.Parse(semverStr)
    if err != nil {
        return nil, err
    }
    
    return &SemverObject{version: blangVersion}, nil
}
```

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->
N/A

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

All core changes must be covered by unit tests, covering both integer and semantic version comparison in tolerations and CEL:

**Toleration Tests:**

- **API Validation Tests:** (staging/src/k8s.io/api/core/v1/toleration_test.go)
- **Scheduler Helper Tests:** (staging/src/k8s.io/component-helpers/scheduling/corev1/helpers_test.go)
- **Validation Tests:** (pkg/apis/core/validation/validation_test.go)
- **CEL Semver Library Tests:** (staging/src/k8s.io/apiserver/pkg/cel/library/semverlib_test.go)
- **CEL Integration Tests:** (staging/src/k8s.io/apiserver/pkg/admission/plugin/cel/cel_test.go)
- `<package>`: `<date>` - `<test coverage>`

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

The following scenarios need to be covered in integration tests:

**Toleration Integration Tests:**

- **Scheduler Integration Tests:** will be extended to cover the new comparison operators:(pkg/scheduler/framework/plugins/tainttoleration/taint_toleration_test.go)
- **ValidatingAdmissionPolicy Integration Tests:** (test/integration/admissionregistration/validating_admission_policy_test.go)
- **CRD Validation Integration Tests:** (test/integration/apiserver/openapi_test.go)
- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

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
The existing e2e tests will be extended to cover the new comparison operators introduced in this KEP:

- **Taints e2e Tests:** (test/e2e/node/taints.go)
- **ValidatingAdmissionPolicy e2e Tests:** (test/e2e/apimachinery/validating_admission_policy.go)
- **CRD e2e Tests:** (test/e2e/apimachinery/crd_validation.go)
- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved 

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- N examples of real-world usage
- N installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

<!--
- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

#### Alpha

- Feature implemented behind `TaintTolerationComparisonOperators` feature gate (disabled by default)
- API validation for comparison operators with automatic type detection
- Taint/toleration matching logic supports `Lt`, `Gt` operators for both integer and semver values
- Feature implemented behind `CELSemverExtensions` feature gate (disabled by default)
- Enhanced semver library with additional comparison functions
- Integration with Kubernetes version parsing standards  

#### Beta

- `TaintTolerationComparisonOperators` feature gate enabled by default
- Feedback collected from early adopters in SIG-Scheduling and SIG-Node
- Performance testing shows no significant scheduler latency increase
- DRA integration tested and validated
- `CELSemverExtensions` feature gate enabled by default
- Feedback collected from SIG-API-Machinery and admission control users
- Performance impact on CEL expression evaluation measured and optimized
- Integration testing with ValidatingAdmissionPolicies and CRD validation

#### GA

- No performance regressions in scheduler throughput under sustained load
- DRA integration proven stable in production environments
- No performance impact on admission control latency
- Demonstrated value in complex version constraint scenarios

### Upgrade / Downgrade Strategy

**Toleration Operators:**

- **Upgrade**: Enable `TaintTolerationComparisonOperators` feature gate in both API Server and Scheduler components
- **Downgrade**: Disable the feature gate; existing workloads with comparison operators will be rejected during validation
- **Migration**: No automatic migration required; new operators are opt-in

**CEL Semver Extensions:**

- **Upgrade**: Enable `CELSemverExtensions` feature gate in API Server components  
- **Downgrade**: Disable the feature gate; ValidatingAdmissionPolicies and CRDs using enhanced functions will fail validation
- **Migration**: Enhanced semver functions are additive; existing CEL expressions continue to work

**Compatibility Considerations:**

- Both feature gates can be enabled independently
- Existing toleration operators (`Equal`, `Exists`) remain unchanged
- Existing CEL semver functions maintain full backward compatibility
- Cross-feature integration (toleration + CEL validation) provides additional value but is not required

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

The skew between kubelet and control-plane components are not impacted. The kube-scheduler is expected to match the kube-apiserver minor version, but may be up to one minor version older (to allow live upgrades).

In the release it's been added, the feature is disabled by default and not recognized by other components.
Whoever enabled the feature manually would take the risk of component like kube-scheduler being old and not recognize the fields.

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

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `TaintTolerationComparisonOperators`
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-scheduler

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->
Yes.

###### What happens if we reenable the feature if it was previously rolled back?

SLA toleration will be respected again.

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
Tests have been added in the integration tests. See [Integration tests](#integration-tests) for more details.

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
It shouldn't impact already running workloads. It's an opt-in feature.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

- `scheduler_scheduling_duration_seconds`
- `scheduler_scheduling_attempts_total`
- `scheduler_scheduling_attempts_total`
- `apiserver_request_total`

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->
Will be considered for beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->
No.

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

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [x] Events
  - Event Reason: FailedScheduling
  - Event Message: "node(s) had untolerated taint `node.kubernetes.io/sla`: `950`"
- [x] API .spec.taints
  - Other field: `key: node.kubernetes.io/sla`
- [x] API .spec.tolerations
  - Other field: `node.kubernetes.io/sla`

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

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->
Yes, a new metrics:

- `scheduler_numeric_tolerations_total`: To measure the number of pods scheduled using numeric toleration operators.
- `scheduler_numeric_taint_mismatches_total`: To measure the scheduling failures due to numeric taint/toleration mismatches.

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

N/A

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
No, the feature is designed to be an enhancement to existing logic without introducing any new API communication patterns.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->
No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->
No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->
Potentially yes, but the impact should be **minimal**. The numeric toleration operators feature could slightly increase time for operations covered by existing SLIs/SLOs due to integer parsing overhead and validation overhead.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->
No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->
No.

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

Same as existing taint/toleration system which is graceful degradation.

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
A failure mode due to numeric toleration operators have integer parsing errors from malformed taint/toleration values causing pods to be rejected with clear error messages.

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

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

- 2025-08-11: Initial KEP

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

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
<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/website]: https://git.k8s.io/website
