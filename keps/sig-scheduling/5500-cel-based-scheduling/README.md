# KEP-5500: CEL Based Comparisons to Tolerations and NodeAffinity

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1 — Tolerate multiple taint conditions with OR logic](#story-1--tolerate-multiple-taint-conditions-with-or-logic)
    - [Story 2 — Tolerate taints based on semantic version comparison](#story-2--tolerate-taints-based-on-semantic-version-comparison)
    - [Story 3 — Node affinity matching labels containing specific string](#story-3--node-affinity-matching-labels-containing-specific-string)
    - [Story 4 — Node affinity with semantic version comparison](#story-4--node-affinity-with-semantic-version-comparison)
    - [Story 5 — PersistentVolume node affinity with CEL expression](#story-5--persistentvolume-node-affinity-with-cel-expression)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Scheduler Performance Regression](#scheduler-performance-regression)
    - [CEL Expression Evaluation Errors](#cel-expression-evaluation-errors)
    - [Controller Hot-Loop When Feature Gate is Disabled](#controller-hot-loop-when-feature-gate-is-disabled)
    - [Security: CEL Expression Complexity](#security-cel-expression-complexity)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
    - [NodeSelectorTerm Changes](#nodeselectorterm-changes)
    - [Toleration Changes](#toleration-changes)
  - [Semantics](#semantics)
    - [1. Pod Node Affinity](#1-pod-node-affinity)
    - [2. PersistentVolume Node Affinity](#2-persistentvolume-node-affinity)
    - [3. Tolerations](#3-tolerations)
  - [Implementation](#implementation)
    - [Feature Gate Definition](#feature-gate-definition)
    - [CEL Compiler](#cel-compiler)
      - [Toleration Compiler](#toleration-compiler)
      - [Node Affinity Compiler](#node-affinity-compiler)
      - [CEL Expression Compilation](#cel-expression-compilation)
      - [CEL Expression Evaluation](#cel-expression-evaluation)
    - [CEL Cache](#cel-cache)
    - [API Validation](#api-validation)
      - [PodValidationOptions modification](#podvalidationoptions-modification)
      - [Validate Toleration CEL Expression](#validate-toleration-cel-expression)
      - [Validate Node Affinity CEL Expression](#validate-node-affinity-cel-expression)
      - [Validate NodeSelectorTerm with CEL Expressions](#validate-nodeselectorterm-with-cel-expressions)
      - [Validate Volume Node Affinity with CEL Expressions](#validate-volume-node-affinity-with-cel-expressions)
    - [Scheduler Logic](#scheduler-logic)
      - [Scheduler Plugin Features](#scheduler-plugin-features)
      - [TaintToleration Plugin](#tainttoleration-plugin)
      - [CEL Toleration Matching Logic](#cel-toleration-matching-logic)
      - [NodeAffinity Plugin](#nodeaffinity-plugin)
      - [CEL Node Affinity Matching Logic](#cel-node-affinity-matching-logic)
      - [Persistent Volumes Node Affinity](#persistent-volumes-node-affinity)
      - [DRA Support Note](#dra-support-note)
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
    - [Upgrade](#upgrade)
    - [Downgrade](#downgrade)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This enhancement introduces Common Expression Language (CEL) support for Kubernetes scheduling constraints. The KEP adds a new `matchCELExpressions` field to **core/v1 NodeSelectorTerm**, enabling CEL expressions for node selection in **Pod NodeAffinity** and **PersistentVolume NodeAffinity**. Additionally, a new `expression` field is added to **core/v1 Toleration** for CEL-based taint matching.

CEL expressions provide a powerful and extensible mechanism for expressing complex scheduling constraints, including semantic version comparisons, string manipulation, and compound logical conditions—all within a single, validated expression.

## Motivation

CEL provides a standardized, extensible expression language already used throughout Kubernetes (ValidatingAdmissionPolicy, CRD validation, authorization). By introducing CEL for scheduling constraints, users gain:

- **Semantic version comparisons**: Schedule pods based on kubelet version, kernel version, or driver versions using expressions like `semver.compare(node.labels['node.example/kubelet-version'], '>=1.28.0')`
- **Compound conditions**: Express multiple conditions in a single expression, such as `semver.compare(node.labels['operating-system.example/kernel-version'], '>=5.10') && semver.compare(node.labels['another.example/container-runtime-version'], '>=2.0')`
- **String operations**: Parse and manipulate label values directly, for example extracting version numbers from prefixed strings using `node.labels['runtime'].split('://')[1]`
- **Flexible taint matching**: Match taints using CEL expressions with access to `taint.key`, `taint.value`, `taint.effect`, and `taint.timeAdded`
- **Extensibility**: New scheduling capabilities can be added through CEL library extensions without requiring core API changes
- **Familiar syntax**: Users already working with CEL in ValidatingAdmissionPolicy or CRD validation can apply the same knowledge to scheduling

### Goals

- Add a `matchCELExpressions` field to `NodeSelectorTerm` that accepts CEL expressions for node selection, affecting Pod NodeAffinity and PersistentVolume NodeAffinity.
- Add an `expression` field to `Toleration` that accepts CEL expressions for taint matching.
- Provide built-in CEL functions for common scheduling use cases, including semantic version comparison (e.g., `semver.compare(node.labels['version'], '>=1.28.0')`).
- Ensure CEL expressions are validated at admission time for syntax correctness, type safety, and cost limits.
- Maintain backward compatibility—existing scheduling configurations continue to work unchanged.
- Gate the feature behind `TaintTolerationNodeAffinityCEL` feature flag, disabled by default in Alpha.

### Non-Goals

- Replacing existing NodeAffinity operators (`In`, `NotIn`, `Exists`, `DoesNotExist`, `Gt`, `Lt`). The existing `matchExpressions` and `matchFields` remain fully supported.
- Replacing existing Toleration operators (`Equal`, `Exists`). The existing operator-based tolerations remain fully supported.
- Providing CEL support for inter-pod affinity/anti-affinity (this may be considered in a future KEP).
- Providing CEL support for the `nodeSelector` field in PodSpec (only NodeAffinity is in scope).
- Providing CEL support for Dynamic Resource Allocation (DRA) NodeSelectors (this may be considered in a future KEP).

## Proposal

### User Stories

#### Story 1 — Tolerate multiple taint conditions with OR logic

As a cluster operator, I have nodes with different maintenance taints. Some nodes have `maintenance=security-patch:NoSchedule` and others have `maintenance=hardware-upgrade:NoSchedule`. I want my critical monitoring workload to tolerate either of these maintenance taints so it can continue running during maintenance windows.

**Example Configuration:**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: monitoring-agent
spec:
  tolerations:
  - expression: "taint.key == 'maintenance' && (taint.value == 'security-patch' || taint.value == 'hardware-upgrade')"
  containers:
  - name: monitor
    image: monitoring:latest
```

#### Story 2 — Tolerate taints based on semantic version comparison

As a cluster operator, I am upgrading Calico CNI across the cluster. Nodes running older CNI versions are tainted with their version. I want workloads that are compatible with CNI v3.25.0 or newer to tolerate these taints and schedule on any node meeting that version requirement.

**Example Configuration:**

```yaml
apiVersion: v1
kind: Node
metadata:
  name: node-1
spec:
  taints:
  - key: "cni.projectcalico.org/version"
    value: "v3.27.2"
    effect: NoSchedule
---
apiVersion: v1
kind: Pod
metadata:
  name: compatible-workload
spec:
  tolerations:
  - expression: "taint.key == 'cni.projectcalico.org/version' && semver.compare(taint.value, '>=3.25.0') && taint.effect == 'NoSchedule'"
  containers:
  - name: app
    image: myapp:latest
```

#### Story 3 — Node affinity matching labels containing specific string

As a cluster operator, I label nodes with their rack location (e.g., `topology.kubernetes.io/rack=us-west-2a-rack-42`). I want to schedule pods only on nodes whose rack label contains a specific datacenter prefix.

**Example Configuration:**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: regional-app
spec:
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchCELExpressions:
          - "node.labels['topology.kubernetes.io/rack'].contains('us-west-2')"
  containers:
  - name: app
    image: myapp:latest
```

#### Story 4 — Node affinity with semantic version comparison

As a cluster operator, I want to schedule a Pod that requires Kubelet version 1.30.0 or higher because it uses a feature introduced in that version. I want to use CEL's semver functions to express this constraint.

**Example Configuration:**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: modern-app
spec:
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchCELExpressions:
          - "semver.compare(node.labels['node.example/kubelet-version'], '>=1.30.0')"
  containers:
  - name: app
    image: myapp:latest
```

#### Story 5 — PersistentVolume node affinity with CEL expression

As a storage administrator, I manage persistent volumes that require specific kernel features. I want to ensure volumes using advanced filesystem features are only attached to nodes with kernel version 5.15.0 or higher and that have the `storage-optimized` label.

**Example Configuration:**

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: advanced-storage-pv
spec:
  capacity:
    storage: 100Gi
  accessModes:
    - ReadWriteOnce
  nodeAffinity:
    required:
      nodeSelectorTerms:
      - matchCELExpressions:
        - "semver.compare(node.labels['operating-system.example/kernel-version'], '>=5.15.0')"
        - "'storage-optimized' in node.labels"
  persistentVolumeReclaimPolicy: Retain
  storageClassName: advanced-storage
```

### Notes/Constraints/Caveats (Optional)

- **CEL Expression Cost Limits**: CEL expressions are subject to cost limits to prevent resource exhaustion. Expressions that exceed the cost budget will be rejected at admission time. The cost limit aligns with existing Kubernetes CEL implementations (e.g., ValidatingAdmissionPolicy). The maximum expression length is 10 Ki, and the estimated cost limit for evaluation is based on logical steps.

- **CEL Environment for Node Affinity**: The CEL environment for `matchCELExpressions` in NodeSelectorTerm exposes the following variable:
  - `node.labels` - A map of the node's labels (type: `map(string, string)`)

- **CEL Environment for Tolerations**: The CEL environment for the `expression` field in Tolerations exposes the following variables:
  - `taint.key` - The taint key (type: `string`)
  - `taint.value` - The taint value (type: `string`)
  - `taint.effect` - The taint effect (type: `string`, one of: `NoSchedule`, `PreferNoSchedule`, `NoExecute`)
  - `taint.timeAdded` - The time when the taint was added (type: `timestamp`, only set for `NoExecute` taints)

- **CEL Libraries Available**: The standard Kubernetes CEL environment is available, which includes CEL standard functions and macros, as well as the Kubernetes extension library.

- **Expression Evaluation Failure Behavior**: If a CEL expression fails to evaluate at scheduling time (e.g., due to referencing a non-existent label without checking existence first), the behavior depends on context:
  - For `requiredDuringSchedulingIgnoredDuringExecution`: The node is considered non-matching, and the pod cannot schedule on that node.
  - For `preferredDuringSchedulingIgnoredDuringExecution`: The term contributes 0 to the node's score.
  - For tolerations with `expression`: The toleration does not match the taint.

- **AND Semantics Within NodeSelectorTerm**: All expressions within a single `NodeSelectorTerm` must evaluate to `true` for the term to match (AND semantics). Multiple `NodeSelectorTerms` are ORed together.

- **Mutual Exclusivity**: The `expression` field in Tolerations is mutually exclusive with the existing `operator`, `key`, and `value` fields. If `expression` is set, the other fields must be empty.

- **Alpha Restrictions**: When `TaintTolerationNodeAffinityCEL=false`, the API server rejects pods and PersistentVolumes using `matchCELExpressions` in node affinity, and pods using `expression` in tolerations.

- **Immutability**: CEL expressions in `matchCELExpressions` and toleration `expression` fields follow the same immutability rules as other scheduling constraints—they cannot be modified after pod creation.

- **CEL Expression Caching**: The scheduler maintains LRU caches for compiled CEL expressions—one for node affinity expressions and one for toleration expressions. Expressions are compiled once and cached for reuse across scheduling cycles. This significantly reduces the overhead of CEL evaluation during the Filter and Score phases.

### Risks and Mitigations

#### Scheduler Performance Regression

**Risk**: CEL expression evaluation during taint/toleration matching or node affinity matching could degrade scheduler performance, especially in clusters with thousands of nodes and complex expressions.

**Mitigation**:

- CEL expressions are compiled once and cached in an LRU cache for reuse across scheduling cycles
- The scheduler maintains separate caches for node affinity and toleration expressions
- Cost limits prevent overly complex expressions from being accepted
- Feature gate allows disabling if performance problems occur
- Performance benchmarking will be conducted before Beta graduation

#### CEL Expression Evaluation Errors

**Risk**: CEL expressions may fail at scheduling time due to:
1. Referencing labels that don't exist on certain nodes
2. Type mismatches in expressions
3. Runtime errors in CEL functions (e.g., invalid semver strings)

This can lead to:
- Pods stuck in `Pending` state if all nodes fail expression evaluation
- Silent scheduling failures for `preferredDuringSchedulingIgnoredDuringExecution` affinity

**Mitigation**:

**1. Admission-Time Validation**

- CEL expressions are compiled and type-checked at admission time
- Syntax errors and type mismatches are rejected immediately:
  ```
  spec.affinity.nodeAffinity...matchCELExpressions[0]: Invalid value:
  "node.labels['foo'] > 5": type mismatch: expected bool, got int
  ```

**2. Runtime Behavior**

- Expression evaluation failures at scheduling time are treated as non-matching (fail-safe behavior)
- For `requiredDuringSchedulingIgnoredDuringExecution`: Pod cannot schedule on that node
- For `preferredDuringSchedulingIgnoredDuringExecution`: Node receives 0 score contribution
- For tolerations: The toleration does not match the taint
- Scheduler logs will capture evaluation errors for debugging

#### Controller Hot-Loop When Feature Gate is Disabled

**Risk**: If a workload controller (Deployment, StatefulSet, Job, etc.) has a pod template that uses `matchCELExpressions` or toleration `expression` fields, and the feature gate is disabled or was disabled after being enabled, the controller will enter a hot-loop:

1. Controller attempts to create a pod from the template
2. API server validation rejects the pod with error about unsupported fields
3. Controller immediately retries pod creation and this cycle repeats indefinitely

This is particularly problematic during rollback/downgrade scenarios or for multi-cluster deployments where the feature gate state differs across clusters.

**Mitigation**:

- Before disabling the feature gate, cluster operators should identify all workloads using CEL expressions via API discovery or scanning tools
- The Upgrade/downgrade documentation should explicitly warn about this scenario and provide steps to identify affected workloads
- The `apiserver_request_total` metric can be used to detect hot-loop conditions

#### Security: CEL Expression Complexity

**Risk**: Malicious or poorly written CEL expressions could consume excessive CPU during evaluation, potentially impacting scheduler performance.

**Mitigation**:

- CEL cost limits are enforced at admission time to reject overly complex expressions
- The maximum expression length is limited to 10 KiB
- Runtime cost tracking prevents runaway evaluations
- Limited CEL environment exposed for both affinity and tolerations

## Design Details

### API Changes

**File**: `staging/src/k8s.io/api/core/v1/types.go`

#### NodeSelectorTerm Changes

Add a new `matchCELExpressions` field to the `NodeSelectorTerm` struct:

```go
// +structType=atomic
type NodeSelectorTerm struct {
	// A list of node selector requirements by node's labels.
	// +optional
	// +listType=atomic
	MatchExpressions []NodeSelectorRequirement `json:"matchExpressions,omitempty" protobuf:"bytes,1,rep,name=matchExpressions"`
	// A list of node selector requirements by node's fields.
	// +optional
	// +listType=atomic
	MatchFields []NodeSelectorRequirement `json:"matchFields,omitempty" protobuf:"bytes,2,rep,name=matchFields"`
	// A list of CEL expressions that must all evaluate to true for a node to match.
	// Each expression has access to 'node.labels' (map of node labels).
	// All expressions must evaluate to true for the term to match (AND semantics).
	// +featureGate=TaintTolerationNodeAffinityCEL
	// +optional
	// +listType=atomic
	MatchCELExpressions []string `json:"matchCELExpressions,omitempty" protobuf:"bytes,3,rep,name=matchCELExpressions"`
}
```

**Constants for CEL expression limits:**

```go
// CELNodeAffinityExpressionMaxCost specifies the cost limit for a single node affinity
// CEL expression evaluation during pod scheduling.
const CELNodeAffinityExpressionMaxCost = 1000000

// CELNodeAffinityExpressionMaxLength specifies the maximum length for CEL expression
// used in Node Affinity
const CELNodeAffinityExpressionMaxLength = 10 * 1024
```

#### Toleration Changes

Add a new `expression` field to the `Toleration` struct:

```go
// The pod this Toleration is attached to tolerates any taint that matches
// the triple <key,value,effect> using the matching operator <operator>.
type Toleration struct {
	// Key is the taint key that the toleration applies to. Empty means match all taint keys.
	// If the key is empty, operator must be Exists; this combination means to match all values and all keys.
	// +optional
	Key string `json:"key,omitempty" protobuf:"bytes,1,opt,name=key"`
	// Operator represents a key's relationship to the value.
	// Valid operators are Exists and Equal. Defaults to Equal.
	// Exists is equivalent to wildcard for value, so that a pod can
	// tolerate all taints of a particular category.
	// +optional
	Operator TolerationOperator `json:"operator,omitempty" protobuf:"bytes,2,opt,name=operator,casttype=TolerationOperator"`
	// Value is the taint value the toleration matches to.
	// If the operator is Exists, the value should be empty, otherwise just a regular string.
	// +optional
	Value string `json:"value,omitempty" protobuf:"bytes,3,opt,name=value"`
	// Effect indicates the taint effect to match. Empty means match all taint effects.
	// When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.
	// +optional
	Effect TaintEffect `json:"effect,omitempty" protobuf:"bytes,4,opt,name=effect,casttype=TaintEffect"`
	// TolerationSeconds represents the period of time the toleration (which must be
	// of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default,
	// it is not set, which means tolerate the taint forever (do not evict). Zero and
	// negative values will be treated as 0 (evict immediately) by the system.
	// +optional
	TolerationSeconds *int64 `json:"tolerationSeconds,omitempty" protobuf:"varint,5,opt,name=tolerationSeconds"`
	// Expression is a CEL expression that evaluates whether this toleration matches a taint.
	//
	// The expression must evaluate to a boolean value. The expression has access to the
	// following CEL variables:
	//
	// - 'taint': the taint being evaluated, with the following properties:
	//   - taint.key (string): the taint key
	//   - taint.value (string): the taint value
	//   - taint.effect (string): the taint effect (NoSchedule, PreferNoSchedule, or NoExecute)
	//   - taint.timeAdded (timestamp): when the taint was added (only for NoExecute taints)
	// When Expression is set, this toleration is evaluated using CEL instead of the
	// traditional key/operator/value/effect matching.
	//
	// +featureGate=TaintTolerationNodeAffinityCEL
	// +optional
	Expression string `json:"expression,omitempty" protobuf:"bytes,6,opt,name=expression"`
}
```

**Constants for CEL expression limits:**

```go
// CELTolerationExpressionMaxCost specifies the cost limit for a single toleration
// CEL expression evaluation during pod scheduling.
const CELTolerationExpressionMaxCost = 1000000

// CELTolerationExpressionMaxLength specifies the maximum length for CEL expression
// used in Toleration
const CELTolerationExpressionMaxLength = 10 * 1024
```

### Semantics

#### 1. Pod Node Affinity

- When `matchCELExpressions` is specified in a `NodeSelectorTerm`, each CEL expression must evaluate to `true` for the term to match (AND semantics).

- CEL expressions have access to `node.labels` as a `map(string, string)`.

- If any CEL expression fails to evaluate (e.g., runtime error), the term is considered non-matching.

- For `requiredDuringSchedulingIgnoredDuringExecution`: All `NodeSelectorTerms` are ORed together. At least one term must match for the pod to be schedulable on the node.

- For `preferredDuringSchedulingIgnoredDuringExecution`: The `weight` is added to the node's score if all CEL expressions in the term evaluate to `true`. If any expression evaluates to `false` or fails, the weight is not added (0 contribution).

- `matchCELExpressions` can be combined with `matchExpressions` and `matchFields` in the same `NodeSelectorTerm`. All must match for the term to match.

#### 2. PersistentVolume Node Affinity

- PersistentVolume node affinity uses the same `NodeSelectorTerm` structure and therefore supports `matchCELExpressions` in the same way as Pod node affinity.

- CEL expressions in PersistentVolume node affinity are evaluated during volume attachment to determine if a node satisfies the volume's requirements.

- The matching logic is shared through the common `NodeSelectorTerm` implementation, ensuring consistent behavior between pod scheduling and volume attachment decisions.

#### 3. Tolerations

- When a toleration has `expression` set, the CEL expression is evaluated against each taint on the node.

- The expression must evaluate to a boolean `true` for the toleration to match a specific taint.

- For taints with `NoSchedule` effect: If no toleration matches, the pod cannot be scheduled on the node.

- For taints with `PreferNoSchedule` effect: If no toleration matches, the taint counts against the node's score. Nodes with fewer unmatched `PreferNoSchedule` taints receive higher scores.

- For taints with `NoExecute` effect: If no toleration matches, running pods are evicted from the node.

- When `expression` is set, the `key`, `operator`, `value`, and `effect` fields are ignored for matching purposes (but `tolerationSeconds` is still respected for `NoExecute` taints).

**Example:**

Node `A` has taint `cni-version=v3.24.0:PreferNoSchedule`.
Node `B` has taint `cni-version=v3.27.0:PreferNoSchedule`.
Pod has toleration with expression: `taint.key == 'cni-version' && semver.compare(taint.value, '>=3.25.0')`.

Result: The Pod tolerates Node B (3.27.0 >= 3.25.0) but does not tolerate Node A (3.24.0 < 3.25.0). Therefore, Node B receives a higher score (no penalty) compared to Node A.

### Implementation

#### Feature Gate Definition

**File**: `pkg/features/kube_features.go`

```go
const (
    // TaintTolerationNodeAffinityCEL enables CEL expressions for tolerations and node affinity
    TaintTolerationNodeAffinityCEL featuregate.Feature = "TaintTolerationNodeAffinityCEL"
)

var defaultKubernetesFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
    TaintTolerationNodeAffinityCEL: {Default: false, PreRelease: featuregate.Alpha},
}
```

#### CEL Compiler

**File**: `staging/src/k8s.io/kube-scheduler/cel/compile.go`

The CEL compiler provides two compilation environments—one for node affinity expressions and one for toleration expressions. The compilers are lazily initialized and cached for reuse.

```go
const (
	taintVar       = "taint"
	taintKeyVar    = "key"
	taintValueVar  = "value"
	taintEffectVar = "effect"
	taintTimeAdded = "timeAdded"

	nodeVar       = "node"
	nodeLabelsVar = "labels"
)

var (
	lazyTolerationCompilerMutex sync.Mutex
	lazyTolerationCompiler      *compiler

	lazyAffinityCompilerMutex sync.Mutex
	lazyAffinityCompiler      *compiler
)

type compiler struct {
	taintType *apiservercel.DeclType
	nodeType  *apiservercel.DeclType
	envset    *environment.EnvSet
}

func GetTolerationCompiler() *compiler {
	lazyTolerationCompilerMutex.Lock()
	defer lazyTolerationCompilerMutex.Unlock()

	if lazyTolerationCompiler == nil {
		lazyTolerationCompiler = newTolerationCompiler()
	}
	return lazyTolerationCompiler
}

func GetAffinityCompiler() *compiler {
	lazyAffinityCompilerMutex.Lock()
	defer lazyAffinityCompilerMutex.Unlock()

	if lazyAffinityCompiler == nil {
		lazyAffinityCompiler = newAffinityCompiler()
	}
	return lazyAffinityCompiler
}
```

##### Toleration Compiler

The toleration compiler creates a CEL environment with a `taint` variable containing `key`, `value`, `effect`, and `timeAdded` fields:

```go
func newTolerationCompiler() *compiler {
	envset := environment.MustBaseEnvSet(environment.DefaultCompatibilityVersion())

	fieldsV135 := []*apiservercel.DeclField{
		field(taintKeyVar, taintKeyType, true),
		field(taintValueVar, taintValueType, true),
		field(taintEffectVar, taintEffectType, true),
		field(taintTimeAdded, taintTimeAddedType, true),
	}
	taintTypeV135 := apiservercel.NewObjectType("kubernetes.Taint", fields(fieldsV135...))

	versioned := []environment.VersionedOptions{
		{
			IntroducedVersion: version.MajorMinor(1, 35),
			EnvOptions: []cel.EnvOption{
				cel.Variable(taintVar, taintTypeV135.CelType()),
			},
			DeclTypes: []*apiservercel.DeclType{
				taintTypeV135,
			},
		},
	}
	envset, err := envset.Extend(versioned...)
	if err != nil {
		panic(fmt.Errorf("internal error building CEL environment: %w", err))
	}

	return &compiler{envset: envset, taintType: taintTypeV135}
}
```

##### Node Affinity Compiler

The node affinity compiler creates a CEL environment with a `node` variable containing a `labels` map:

```go
func newAffinityCompiler() *compiler {
	envset := environment.MustBaseEnvSet(environment.DefaultCompatibilityVersion())

	fieldsV135 := []*apiservercel.DeclField{
		field(nodeLabelsVar, nodeLabelsMapType, true),
	}
	nodeTypeV135 := apiservercel.NewObjectType("kubernetes.Node", fields(fieldsV135...))

	versioned := []environment.VersionedOptions{
		{
			IntroducedVersion: version.MajorMinor(1, 35),
			EnvOptions: []cel.EnvOption{
				cel.Variable(nodeVar, nodeTypeV135.CelType()),
			},
			DeclTypes: []*apiservercel.DeclType{
				nodeTypeV135,
			},
		},
	}
	envset, err := envset.Extend(versioned...)
	if err != nil {
		panic(fmt.Errorf("internal error building CEL environment: %w", err))
	}

	return &compiler{envset: envset, nodeType: nodeTypeV135}
}
```

##### CEL Expression Compilation

The `CompileCELExpression` method compiles a CEL expression and returns a `CompilationResult`:

```go
type CompilationResult struct {
	Program     cel.Program
	Error       *apiservercel.Error
	Expression  string
	OutputType  *cel.Type
	Environment *cel.Env
	MaxCost     uint64
}

func (c compiler) CompileCELExpression(expression string, options Options) CompilationResult {
	env, err := c.envset.Env(ptr.Deref(options.EnvType, environment.StoredExpressions))
	if err != nil {
		return resultError(fmt.Sprintf("unexpected error loading CEL environment: %v", err), apiservercel.ErrorTypeInternal)
	}

	ast, issues := env.Compile(expression)
	if issues != nil {
		return resultError("compilation failed: "+issues.String(), apiservercel.ErrorTypeInvalid)
	}

	expectedReturnType := cel.BoolType
	if ast.OutputType() != expectedReturnType && ast.OutputType() != cel.AnyType {
		return resultError(fmt.Sprintf("must evaluate to %v or the unknown type, not %v", expectedReturnType.String(), ast.OutputType().String()), apiservercel.ErrorTypeInvalid)
	}

	prog, err := env.Program(ast,
		cel.CostLimit(ptr.Deref(options.CostLimit, v1.CELTolerationExpressionMaxCost)),
		cel.InterruptCheckFrequency(celconfig.CheckFrequency),
	)
	if err != nil {
		return resultError("program instantiation failed: "+err.Error(), apiservercel.ErrorTypeInternal)
	}

	// ... cost estimation logic
	return CompilationResult{
		Program:     prog,
		Expression:  expression,
		OutputType:  ast.OutputType(),
		Environment: env,
		MaxCost:     costEst.Max,
	}
}
```

##### CEL Expression Evaluation

The `CompilationResult` provides methods to evaluate expressions against taints or node labels which must return a boolean:

```go
func (c CompilationResult) TolerationsMatches(input Taint) (bool, *cel.EvalDetails, error) {
	variables := map[string]any{
		taintVar: map[string]any{
			taintKeyVar:    input.Key,
			taintValueVar:  input.Value,
			taintEffectVar: input.Effect,
			taintTimeAdded: input.TimeAdded,
		},
	}

	result, details, err := c.Program.Eval(variables)
	if err != nil {
		return false, details, err
	}
	resultBool, _ := result.ConvertToNative(boolType)
	return resultBool.(bool), details, nil
}

func (c CompilationResult) NodeAffinityMatches(nodeLabels map[string]string) (bool, *cel.EvalDetails, error) {
	variables := map[string]any{
		nodeVar: map[string]any{
			nodeLabelsVar: nodeLabels,
		},
	}

	result, details, err := c.Program.Eval(variables)
	if err != nil {
		return false, details, err
	}
	resultBool, _ := result.ConvertToNative(boolType)
	return resultBool.(bool), details, nil
}
```

#### CEL Cache

**File**: `staging/src/k8s.io/kube-scheduler/cel/cache.go`

The cache provides thread-safe LRU caching of compiled CEL expressions. This ensures that expensive CEL compilation is done once and reused across multiple scheduling cycles.

```go
// Cache is a thread-safe LRU cache for a compiled CEL expression.
type Cache struct {
	compileMutex keymutex.KeyMutex
	cacheMutex   sync.RWMutex
	cache        *lru.Cache
	compiler     *compiler
}

// NewTolerationCache creates a cache for compiled Toleration CEL expressions.
func NewTolerationCache(maxCacheEntries int) *Cache {
	return &Cache{
		compileMutex: keymutex.NewHashed(0),
		cache:        lru.New(maxCacheEntries),
		compiler:     GetTolerationCompiler(),
	}
}

// NewAffinityCache creates a cache for compiled Node Affinity CEL expressions.
func NewAffinityCache(maxCacheEntries int) *Cache {
	return &Cache{
		compileMutex: keymutex.NewHashed(0),
		cache:        lru.New(maxCacheEntries),
		compiler:     GetAffinityCompiler(),
	}
}

// GetOrCompile checks whether the cache already has a compilation result
// and returns that if available. Otherwise it compiles, stores successful
// results and returns the new result.
func (c *Cache) GetOrCompile(expression string) CompilationResult {
	// Compiling a CEL expression is expensive enough that it is cheaper
	// to lock a mutex than doing it several times in parallel.
	c.compileMutex.LockKey(expression)
	defer c.compileMutex.UnlockKey(expression)

	cached := c.get(expression)
	if cached != nil {
		return *cached
	}

	expr := c.compiler.CompileCELExpression(expression, Options{DisableCostEstimation: true})
	if expr.Error == nil {
		c.add(expression, &expr)
	}
	return expr
}

func (c *Cache) add(expression string, expr *CompilationResult) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.cache.Add(expression, expr)
}

func (c *Cache) get(expression string) *CompilationResult {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()
	expr, found := c.cache.Get(expression)
	if !found {
		return nil
	}
	return expr.(*CompilationResult)
}
```

#### API Validation

**File**: `pkg/apis/core/validation/validation.go`

##### PodValidationOptions modification

```go
// PodValidationOptions contains the different settings for pod validation
type PodValidationOptions struct {
	....
	// Allow Toleration and Node Affinity CEL Expressions
	AllowTaintTolerationNodeAffinityCEL bool
}
```

##### Validate Toleration CEL Expression

The `validateTolerationCELExpression` function validates CEL expressions in tolerations. It checks:
1. The feature gate is enabled
2. The expression is not used alongside key, value, operator, or effect fields
3. The expression length doesn't exceed the maximum
4. The expression compiles successfully
5. The expression cost doesn't exceed the limit

```go
func validateTolerationCELExpression(toleration core.Toleration, fldPath *field.Path, opts PodValidationOptions) field.ErrorList {
	allErrs := field.ErrorList{}
	if !opts.AllowTaintTolerationNodeAffinityCEL {
		allErrs = append(allErrs, field.Forbidden(fldPath.Child("expression"), "toleration CEL expressions require feature gate TaintTolerationNodeAffinityCEL"))
		return allErrs
	}

	if toleration.Key != "" || toleration.Value != "" || toleration.Operator != "" || toleration.Effect != "" {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("expression"), toleration.Expression, "expression cannot be used with key, value, operator, or effect fields"))
		return allErrs
	}

	if len(toleration.Expression) > v1.CELTolerationExpressionMaxLength {
		allErrs = append(allErrs, field.TooLong(fldPath.Child("expression"), "" /*unused*/, v1.CELTolerationExpressionMaxLength))
		return allErrs
	}

	// Compile the expression and make sure it compiles and doesnt violate the maximum cost
	result := schdlrcel.GetTolerationCompiler().CompileCELExpression(toleration.Expression, schdlrcel.Options{})
	if result.Error != nil {
		allErrs = append(allErrs, convertCELErrorToValidationError(fldPath.Child("expression"), toleration.Expression, result.Error))
	} else if result.MaxCost > v1.CELTolerationExpressionMaxCost {
		allErrs = append(allErrs, field.Forbidden(fldPath.Child("expression"), "too complex, exceeds cost limit"))
	}

	return allErrs
}
```

##### Validate Node Affinity CEL Expression

The `ValidateNodeAffinityCELExpression` function validates CEL expressions in node affinity. It performs similar checks as toleration validation.

```go
// ValidateNodeAffinityCELExpression tests that the specified CEL expression has valid data
func ValidateNodeAffinityCELExpression(expr string, allowTaintTolerationNodeAffinityCEL bool, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if !allowTaintTolerationNodeAffinityCEL {
		allErrs = append(allErrs, field.Forbidden(fldPath.Child("expression"), "Node Affinity CEL expressions require feature gate TaintTolerationNodeAffinityCEL"))
		return allErrs
	}

	if len(expr) > v1.CELNodeAffinityExpressionMaxLength {
		allErrs = append(allErrs, field.TooLong(fldPath.Child("expression"), "" /*unused*/, v1.CELNodeAffinityExpressionMaxLength))
		return allErrs
	}

	// Compile the expression and make sure it compiles and doesnt violate the maximum cost
	result := schdlrcel.GetAffinityCompiler().CompileCELExpression(expr, schdlrcel.Options{})
	if result.Error != nil {
		allErrs = append(allErrs, convertCELErrorToValidationError(fldPath.Root(), expr, result.Error))
	} else if result.MaxCost > v1.CELTolerationExpressionMaxCost {
		allErrs = append(allErrs, field.Forbidden(fldPath.Root(), "too complex, exceeds cost limit"))
	}

	return allErrs
}
```

##### Validate NodeSelectorTerm with CEL Expressions

```go
// ValidateNodeSelectorTerm tests that the specified node selector term has valid data
func ValidateNodeSelectorTerm(term core.NodeSelectorTerm, allowInvalidLabelValueInRequiredNodeAffinity, allowTaintTolerationNodeAffinityCEL bool, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for j, req := range term.MatchExpressions {
		allErrs = append(allErrs, ValidateNodeSelectorRequirement(req, allowInvalidLabelValueInRequiredNodeAffinity, fldPath.Child("matchExpressions").Index(j))...)
	}

	for j, req := range term.MatchFields {
		allErrs = append(allErrs, ValidateNodeFieldSelectorRequirement(req, fldPath.Child("matchFields").Index(j))...)
	}

	for j, req := range term.MatchCELExpressions {
		allErrs = append(allErrs, ValidateNodeAffinityCELExpression(req, allowTaintTolerationNodeAffinityCEL, fldPath.Child("matchCELExpressions").Index(j))...)
	}

	return allErrs
}
```

##### Validate Volume Node Affinity with CEL Expressions

```go
// PersistentVolumeSpecValidationOptions contains the different settings for PeristentVolume validation
type PersistentVolumeSpecValidationOptions struct {
	// Allow users to modify the class of volume attributes
	EnableVolumeAttributesClass bool
	// Allow invalid label-value in RequiredNodeSelector
	AllowInvalidLabelValueInRequiredNodeAffinity bool
	// Allow Node Affinity CEL Expressions
	AllowTaintTolerationNodeAffinityCEL bool
}

func ValidationOptionsForPersistentVolume(pv, oldPv *core.PersistentVolume) PersistentVolumeSpecValidationOptions {
	opts := PersistentVolumeSpecValidationOptions{
		EnableVolumeAttributesClass:                  utilfeature.DefaultMutableFeatureGate.Enabled(features.VolumeAttributesClass),
		AllowInvalidLabelValueInRequiredNodeAffinity: false,
		AllowTaintTolerationNodeAffinityCEL:          utilfeature.DefaultFeatureGate.Enabled(features.TaintTolerationNodeAffinityCEL),
	}
	// ... backward compatibility logic
	return opts
}

func validateVolumeNodeAffinity(nodeAffinity *core.VolumeNodeAffinity, opts PersistentVolumeSpecValidationOptions, fldPath *field.Path) (bool, field.ErrorList) {
	allErrs := field.ErrorList{}

	if nodeAffinity == nil {
		return false, allErrs
	}

	if nodeAffinity.Required != nil {
		allErrs = append(allErrs, ValidateNodeSelector(nodeAffinity.Required, opts.AllowInvalidLabelValueInRequiredNodeAffinity, opts.AllowTaintTolerationNodeAffinityCEL, fldPath.Child("required"))...)
	} else {
		allErrs = append(allErrs, field.Required(fldPath.Child("required"), "must specify required node constraints"))
	}

	return true, allErrs
}
```

#### Scheduler Logic

##### Scheduler Plugin Features

**File**: `pkg/scheduler/framework/plugins/feature/feature.go`

The scheduler plugins receive the feature gate state through the `Features` struct:

```go
type Features struct {
	...
	EnableTaintTolerationNodeAffinityCEL bool
}

func NewSchedulerFeaturesFromGates(featureGate featuregate.FeatureGate) Features {
	return Features{
		...
		EnableTaintTolerationNodeAffinityCEL: featureGate.Enabled(features.TaintTolerationNodeAffinityCEL),
	}
}
```

##### TaintToleration Plugin

**File**: `pkg/scheduler/framework/plugins/tainttoleration/taint_toleration.go`

The TaintToleration plugin is updated to support CEL expressions in tolerations:

```go
type TaintToleration struct {
	...
	enableTolerationCEL                      bool
	celCache                                 *schedulercel.Cache
}

func New(_ context.Context, _ runtime.Object, h fwk.Handle, fts feature.Features) (fwk.Plugin, error) {
	return &TaintToleration{
		handle:                                   h,
		enableSchedulingQueueHint:                fts.EnableSchedulingQueueHint,
		enableTaintTolerationComparisonOperators: fts.EnableTaintTolerationComparisonOperators,
		enableTolerationCEL:                      fts.EnableTaintTolerationNodeAffinityCEL,
		celCache:                                 schedulercel.NewTolerationCache(10),
	}, nil
}

// Filter invoked at the filter extension point.
func (pl *TaintToleration) Filter(ctx context.Context, state fwk.CycleState, pod *v1.Pod, nodeInfo fwk.NodeInfo) *fwk.Status {
	...
	taint, isUntolerated := v1helper.FindMatchingUntoleratedTaint(logger, node.Spec.Taints, pod.Spec.Tolerations,
		helper.DoNotScheduleTaintsFilterFunc(),
		pl.enableTaintTolerationComparisonOperators, pl.enableTolerationCEL, pl.celCache)
	....
}
```

##### CEL Toleration Matching Logic

**File**: `staging/src/k8s.io/component-helpers/scheduling/corev1/helpers.go`

The toleration matching logic is extended to support CEL expressions, the `toleratesTaintViaCEL` receive a CEL cache that was initialized for the plugin:

```go
// toleratesTaintViaCEL evaluates a toleration's CEL expression against a taint.
func toleratesTaintViaCEL(logger klog.Logger, toleration *v1.Toleration, taint *v1.Taint, celCache *schedulercel.Cache) bool {
	if celCache == nil || len(toleration.Expression) == 0 {
		return false
	}

	// Compile or retrieve cached CEL expression
	compilationResult := celCache.GetOrCompile(toleration.Expression)
	if compilationResult.Error != nil {
		logger.Error(nil, "failed to compile CEL expression", "expression", toleration.Expression, "error", compilationResult.Error.Detail)
		return false
	}

	timeAddedStr := ""
	if taint.TimeAdded != nil {
		timeAddedStr = taint.TimeAdded.String()
	}
	matches, _, err := compilationResult.TolerationsMatches(schedulercel.Taint{
		Key:       taint.Key,
		Value:     taint.Value,
		Effect:    string(taint.Effect),
		TimeAdded: timeAddedStr,
	})

	if err != nil {
		logger.Error(err, "CEL Runtime error", "expression", toleration.Expression)
		return false
	}

	return matches
}

// TolerationsTolerateTaint checks if taint is tolerated by any of the tolerations.
func TolerationsTolerateTaint(logger klog.Logger, tolerations []v1.Toleration, taint *v1.Taint, enableComparisonOperators, enableTolerationCEL bool, celCache *schedulercel.Cache) bool {
	for i := range tolerations {
		// First check if CEL expression exists and evaluate it
		if enableTolerationCEL && len(tolerations[i].Expression) > 0 {
			if toleratesTaintViaCEL(logger, &tolerations[i], taint, celCache) {
				return true
			}
			// If CEL expression exists but doesn't match, continue to next toleration
			continue
		}

		// Fall back to standard toleration matching
		if tolerations[i].ToleratesTaint(logger, taint, enableComparisonOperators) {
			return true
		}
	}
	return false
}

// FindMatchingUntoleratedTaint checks if the given tolerations tolerates
// all the filtered taints, and returns the first taint without a toleration
func FindMatchingUntoleratedTaint(logger klog.Logger, taints []v1.Taint, tolerations []v1.Toleration, inclusionFilter taintsFilterFunc, enableComparisonOperators, enableTolerationCEL bool, celCache *schedulercel.Cache) (v1.Taint, bool) {
	filteredTaints := getFilteredTaints(taints, inclusionFilter)
	for _, taint := range filteredTaints {
		if !TolerationsTolerateTaint(logger, tolerations, &taint, enableComparisonOperators, enableTolerationCEL, celCache) {
			return taint, true
		}
	}
	return v1.Taint{}, false
}
```

##### NodeAffinity Plugin

**File**: `pkg/scheduler/framework/plugins/nodeaffinity/node_affinity.go`

The NodeAffinity plugin is updated to support CEL expressions in node affinity:

```go
type NodeAffinity struct {
	...
	enableNodeAffinityCEL     bool
	celCache                  *schedulercel.Cache
}

func New(_ context.Context, plArgs runtime.Object, h fwk.Handle, fts feature.Features) (fwk.Plugin, error) {
	args, err := getArgs(plArgs)
	if err != nil {
		return nil, err
	}
	pl := &NodeAffinity{
		handle:                    h,
		enableSchedulingQueueHint: fts.EnableSchedulingQueueHint,
		enableNodeAffinityCEL:     fts.EnableTaintTolerationNodeAffinityCEL,
		celCache:                  schedulercel.NewAffinityCache(10),
	}
	// ... configure addedNodeSelector and addedPrefSchedTerms
	return pl, nil
}

// Filter checks if the Node matches the Pod .spec.affinity.nodeAffinity
func (pl *NodeAffinity) Filter(ctx context.Context, state fwk.CycleState, pod *v1.Pod, nodeInfo fwk.NodeInfo) *fwk.Status {
	node := nodeInfo.Node()

	if pl.addedNodeSelector != nil && !pl.addedNodeSelector.Match(node, pl.enableNodeAffinityCEL, pl.celCache) {
		return fwk.NewStatus(fwk.UnschedulableAndUnresolvable, errReasonEnforced)
	}

	s, err := getPreFilterState(state)
	if err != nil {
		s = &preFilterState{requiredNodeSelectorAndAffinity: nodeaffinity.GetRequiredNodeAffinity(pod)}
	}

	match, _ := s.requiredNodeSelectorAndAffinity.Match(node, pl.enableNodeAffinityCEL, pl.celCache)
	if !match {
		return fwk.NewStatus(fwk.UnschedulableAndUnresolvable, ErrReasonPod)
	}

	return nil
}
```

##### CEL Node Affinity Matching Logic

**File**: `staging/src/k8s.io/component-helpers/scheduling/corev1/nodeaffinity/nodeaffinity.go`

The node affinity matching logic is extended to support `matchCELExpressions`:

```go
type nodeSelectorTerm struct {
	matchLabels         labels.Selector
	matchFields         fields.Selector
	matchCELExpressions []string // CEL expressions to evaluate against node labels
	parseErrs           []error
}

func newNodeSelectorTerm(term *v1.NodeSelectorTerm, path *field.Path) nodeSelectorTerm {
	var parsedTerm nodeSelectorTerm
	// ... parse matchExpressions and matchFields

	if len(term.MatchCELExpressions) != 0 {
		// Store CEL expressions as-is; validation happens at API level,
		// compilation and evaluation happen at match time with the cache.
		parsedTerm.matchCELExpressions = term.MatchCELExpressions
	}
	return parsedTerm
}

func (t *nodeSelectorTerm) match(nodeLabels labels.Set, nodeFields fields.Set, enableNodeAffinityCEL bool, celCache *schedulercel.Cache) (bool, []error) {
	....
	if enableNodeAffinityCEL && len(t.matchCELExpressions) > 0 {
		if celCache == nil {
			return false, []error{errors.NewAggregate([]error{nil})}
		}

		for _, expr := range t.matchCELExpressions {
			compilationResult := celCache.GetOrCompile(expr)
			if compilationResult.Error != nil {
				klog.ErrorS(nil, "Failed to compile CEL expression", "expression", expr, "error", compilationResult.Error.Detail)
				return false, nil
			}
			matches, _, err := compilationResult.NodeAffinityMatches(map[string]string(nodeLabels))
			if err != nil {
				klog.ErrorS(err, "CEL runtime error", "expression", expr)
				return false, nil
			}
			if !matches {
				return false, nil
			}
		}
	}
	return true, nil
}
```

##### Persistent Volumes Node Affinity

The VolumeBinding plugin uses the same `NodeSelector` matching functions from `component-helpers/scheduling/corev1/nodeaffinity`. When `matchCELExpressions` is specified in a PersistentVolume's node affinity, the matching logic automatically evaluates the CEL expressions against the node's labels.

```
pkg/scheduler/framework/plugins/volumebinding/
    - volume_binding.go     - Filter() entry point
    - binder.go             - FindPodVolumes(), checkBoundClaims(), findMatchingVolumes()

staging/src/k8s.io/component-helpers/storage/volume/
    - helpers.go            - CheckNodeAffinity()
    - pv_helpers.go         - FindMatchingVolume()

staging/src/k8s.io/component-helpers/scheduling/corev1/nodeaffinity/
    - nodeaffinity.go       - Match() with CEL support
```

##### DRA Support Note

CEL expressions in `matchCELExpressions` for DRA (Dynamic Resource Allocation) are explicitly **out of scope** for this KEP. The `ResourceSlice.spec.nodeSelector` and `ResourceSlice.spec.devices[].basic.nodeSelector` will not support `matchCELExpressions` in the initial implementation.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

The following unit test files include coverage for cel expressions for NodeAffinity and Tolerations:

1. **pkg/apis/core/validation/validation_test.go**
   - Tests for validating Toleration CEL expressions
   - Tests for validating Node Affinity CEL expressions
   - Tests for validating NodeSelectorTerms with MatchCELExpressions
   

```sh
--- PASS: TestTolerationCELExpressionValidation (0.00s)
    --- PASS: TestTolerationCELExpressionValidation/valid_CEL_expression_with_feature_gate_enabled (0.00s)
    --- PASS: TestTolerationCELExpressionValidation/valid_CEL_expression_with_complex_logic (0.00s)
    --- PASS: TestTolerationCELExpressionValidation/CEL_expression_with_feature_gate_disabled (0.00s)
    --- PASS: TestTolerationCELExpressionValidation/CEL_expression_with_key_field_set (0.00s)
    --- PASS: TestTolerationCELExpressionValidation/CEL_expression_with_value_field_set (0.00s)
    --- PASS: TestTolerationCELExpressionValidation/CEL_expression_with_operator_field_set (0.00s)
    --- PASS: TestTolerationCELExpressionValidation/CEL_expression_with_effect_field_set (0.00s)
    --- PASS: TestTolerationCELExpressionValidation/invalid_CEL_expression_syntax (0.00s)
    --- PASS: TestTolerationCELExpressionValidation/CEL_expression_with_wrong_return_type (0.00s)
PASS

--- PASS: TestNodeAffinityCELExpressionValidation (0.00s)
    --- PASS: TestNodeAffinityCELExpressionValidation/valid_CEL_expression_with_feature_gate_enabled (0.00s)
    --- PASS: TestNodeAffinityCELExpressionValidation/valid_CEL_expression_with_label_value_check (0.00s)
    --- PASS: TestNodeAffinityCELExpressionValidation/CEL_expression_with_feature_gate_disabled (0.00s)
    --- PASS: TestNodeAffinityCELExpressionValidation/invalid_CEL_expression_syntax (0.00s)
    --- PASS: TestNodeAffinityCELExpressionValidation/CEL_expression_with_wrong_return_type (0.00s)
PASS

--- PASS: TestNodeSelectorTermWithCELExpressions (0.00s)
    --- PASS: TestNodeSelectorTermWithCELExpressions/valid_term_with_matchCELExpressions (0.00s)
    --- PASS: TestNodeSelectorTermWithCELExpressions/valid_term_with_multiple_matchCELExpressions (0.00s)
    --- PASS: TestNodeSelectorTermWithCELExpressions/term_with_matchCELExpressions_and_matchExpressions (0.00s)
    --- PASS: TestNodeSelectorTermWithCELExpressions/term_with_matchCELExpressions_feature_gate_disabled (0.00s)
    --- PASS: TestNodeSelectorTermWithCELExpressions/term_with_invalid_CEL_expression (0.00s)
PASS

```

2. **staging/src/k8s.io/kube-scheduler/cel/compile_test.go**
   - Tests for Tolerations compiler
   - Tests for Node Affinity compiler

```sh
--- PASS: TestTolerationCompiler (0.00s)
    --- PASS: TestTolerationCompiler/type-error (0.00s)
    --- PASS: TestTolerationCompiler/taint-value-equals (0.00s)
    --- PASS: TestTolerationCompiler/taint-effect-equals (0.00s)
    --- PASS: TestTolerationCompiler/taint-key-startswith (0.00s)
    --- PASS: TestTolerationCompiler/taint-value-empty (0.00s)
    --- PASS: TestTolerationCompiler/false (0.00s)
    --- PASS: TestTolerationCompiler/taint-key-equals (0.00s)
    --- PASS: TestTolerationCompiler/taint-key-not-equals (0.00s)
    --- PASS: TestTolerationCompiler/taint-effect-not-equals (0.00s)
    --- PASS: TestTolerationCompiler/taint-key-and-value (0.00s)
    --- PASS: TestTolerationCompiler/taint-key-or-value (0.00s)
    --- PASS: TestTolerationCompiler/taint-key-contains (0.00s)
    --- PASS: TestTolerationCompiler/true (0.00s)
    --- PASS: TestTolerationCompiler/syntax-error (0.00s)
PASS

--- PASS: TestNodeAffinityCompiler (0.01s)
    --- PASS: TestNodeAffinityCompiler/label-exists-and-value-equals (0.00s)
    --- PASS: TestNodeAffinityCompiler/label-exists-or-other (0.00s)
    --- PASS: TestNodeAffinityCompiler/syntax-error (0.00s)
    --- PASS: TestNodeAffinityCompiler/label-not-exists (0.00s)
    --- PASS: TestNodeAffinityCompiler/label-value-startswith (0.00s)
    --- PASS: TestNodeAffinityCompiler/label-value-contains (0.00s)
    --- PASS: TestNodeAffinityCompiler/multiple-labels (0.00s)
    --- PASS: TestNodeAffinityCompiler/empty-labels (0.00s)
    --- PASS: TestNodeAffinityCompiler/true (0.00s)
    --- PASS: TestNodeAffinityCompiler/false (0.00s)
    --- PASS: TestNodeAffinityCompiler/type-error (0.00s)
    --- PASS: TestNodeAffinityCompiler/label-exists (0.00s)
    --- PASS: TestNodeAffinityCompiler/label-value-equals (0.00s)
    --- PASS: TestNodeAffinityCompiler/label-value-not-equals (0.00s)
PASS
```

3. **staging/src/k8s.io/kube-scheduler/cel/cache_test.go**
   - Tests for Toleration get from cache or compile
   - Tests for Node Affinity get from cache or compile
   - Tests for do not cache errors
   - Tests for LRU evicitons
   - 

```sh
--- PASS: TestTolerationCacheGetOrCompile (0.00s)
--- PASS: TestAffinityCacheGetOrCompile (0.00s)
--- PASS: TestCacheDoesNotCacheErrors (0.00s)
--- PASS: TestCacheLRUEviction (0.00s)
```

4. **staging/src/k8s.io/component-helpers/scheduling/corev1/nodeaffinity/nodeaffinity_test.go**
	- Tests for NodeAffinity matching logic

```sh
--- PASS: TestNodeAffinityCELMatching (0.00s)
    --- PASS: TestNodeAffinityCELMatching/CEL_expression_matches_label_exists (0.00s)
    --- PASS: TestNodeAffinityCELMatching/CEL_expression_label_does_not_exist (0.00s)
    --- PASS: TestNodeAffinityCELMatching/CEL_expression_matches_label_value (0.00s)
    --- PASS: TestNodeAffinityCELMatching/CEL_expression_label_value_mismatch (0.00s)
    --- PASS: TestNodeAffinityCELMatching/CEL_expression_with_startsWith (0.00s)
    --- PASS: TestNodeAffinityCELMatching/CEL_expression_with_contains (0.00s)
    --- PASS: TestNodeAffinityCELMatching/multiple_CEL_expressions_(AND_semantics) (0.00s)
    --- PASS: TestNodeAffinityCELMatching/multiple_CEL_expressions_one_fails_(AND_semantics) (0.00s)
    --- PASS: TestNodeAffinityCELMatching/multiple_terms_(OR_semantics)_first_matches (0.00s)
    --- PASS: TestNodeAffinityCELMatching/multiple_terms_(OR_semantics)_second_matches (0.00s)
    --- PASS: TestNodeAffinityCELMatching/mix_of_CEL_and_matchExpressions (0.00s)
    --- PASS: TestNodeAffinityCELMatching/mix_of_CEL_and_matchExpressions_-_CEL_fails (0.00s)
    --- PASS: TestNodeAffinityCELMatching/CEL_expression_always_true (0.00s)
PASS
```
5. **staging/src/k8s.io/component-helpers/scheduling/corev1/helpers_test.go**
	- Tests for Toleration CEL matching

```sh
--- PASS: TestTolerationCELMatching (0.00s)
    --- PASS: TestTolerationCELMatching/CEL_expression_matches_taint_key (0.00s)
    --- PASS: TestTolerationCELMatching/CEL_expression_does_not_match_taint_key (0.00s)
    --- PASS: TestTolerationCELMatching/CEL_expression_matches_taint_key_and_value (0.00s)
    --- PASS: TestTolerationCELMatching/CEL_expression_matches_taint_key_but_not_value (0.00s)
    --- PASS: TestTolerationCELMatching/CEL_expression_matches_taint_effect (0.00s)
    --- PASS: TestTolerationCELMatching/CEL_expression_with_startsWith (0.00s)
    --- PASS: TestTolerationCELMatching/CEL_expression_with_contains (0.00s)
    --- PASS: TestTolerationCELMatching/CEL_expression_tolerates_all_taints_with_true (0.00s)
    --- PASS: TestTolerationCELMatching/multiple_taints_with_one_CEL_toleration_matching_all (0.00s)
    --- PASS: TestTolerationCELMatching/multiple_taints_with_CEL_toleration_matching_only_some (0.00s)
    --- PASS: TestTolerationCELMatching/mix_of_CEL_and_standard_tolerations (0.00s)
PASS
```

##### Integration tests

Update the following integration tests to include new operators:

1. **TestTaintTolerationFilter:** (`filters/filters_test.go`)
2. **TestTaintTolerationScoring:** (`scoring/priorities_test.go`)
3. **TestTaintNodeByCondition:** (`taint/taint_test.go`)
4. **TestNodeSelectorRequirementsAsSelector**: (`nodeaffinity/nodeaffinity_test.go`)
5. **TestPreferredSchedulingTermsScore**: (`nodeaffinity/nodeaffinity_test.go`)
6. **TestPodMatchesNodeSelectorAndAffinityTerms**: (`nodeaffinity/nodeaffinity_test.go`)
7. **TestNodeSelectorMatch**: (`nodeaffinity/nodeaffinity_test.go`)
8. **General Scheduler Tests:** (`scheduler_test.go`):
9. **PersistentVolume Tests:** (`storage/persistentvolume_test.go`):

##### e2e tests

The existing e2e tests will be extended to cover the new taints cases and new affinity cases introduced in this KEP:

- **Node Taints e2e Tests:** (test/e2e/node/taints.go)
- **Scheduler Taints e2e Tests:** (test/e2e/scheduling)
- **PersistentVolume e2e Tests:** (test/e2e/storage/persistent_volumes.go)

### Graduation Criteria

#### Alpha

- Feature implemented behind `TaintTolerationNodeAffinityCEL` feature gate (disabled by default)
- API validation for version operators in place
- Taint/toleration matching logic supports cel expressions
- Node affinity matching logic supports cel expressions

#### Beta

- Feature enabled by default
- Feedback collected from early adopters in SIG-Scheduling
- Performance testing shows that there is no significant scheduler latency increase nor memory usage increase.
- Stress testing.

#### GA

- Evidence of real-world adoption.
- Complete scalability validation.

### Upgrade / Downgrade Strategy

#### Upgrade
  Enable the feature gate in kube-apiserver first then kube-scheduler. This ensures the API server can accept and validate pods with the CEL expressions before the kube-scheduler tries to process them.

#### Downgrade
  Disable the feature gate in in kube-scheduler then kube-apiserver. Since we want to stop the kube-scheduler from processing the CEL expressions first, then stop the API server from accepting new pods with CEL expressions. This prevents the scheduler from trying to handle features the API server would reject.
  
**What happens when the scheduler doesn't recognize CEL expression field for tolerations:**

When the feature gate is disabled and the scheduler encounters a pod `expression` field for tolerations:
- The expression field is dropped during deserialization (unknown field).
- The Toleration object is interpreted with default values (Operator: Equal, Key: empty).
- This "empty" toleration fails to match any standard node taint.
- Filter returns `UnschedulableAndUnresolvable` status
- Pod remains in Pending state.

**What happens when the scheduler doesn't recognize matchCELExpressions field nodeAffinity:**

When the feature gate is disabled and the scheduler encounters a pod with `matchCELExpressions` field for node affinity:
- The `matchCELExpressions` field is dropped during deserialization (unknown field).
- The NodeSelectorTerm is interpreted as having empty matchExpressions and empty matchFields.

### Version Skew Strategy

The skew between kubelet and control-plane components are not impacted. The kube-scheduler is expected to match the kube-apiserver minor version, but may be up to one minor version older (to allow live upgrades).

In the release it's been added, the feature is disabled by default and not recognized by other components.

Whoever enabled the feature manually would take the risk of component like kube-scheduler being old and not recognize the fields.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `TaintTolerationNodeAffinityCEL`
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-scheduler

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Impact on existing pods with CEL fields when feature is disabled:

1. Already-running pods: Continue running normally.

2. Unscheduled/pending pods:
	- **Tolerations**: The scheduler will ignore the `expression` field, fail to match the taint, and the Pod will remain Pending.
	- **Node Affinity**: The scheduler will ignore the `matchCELExpressions` field. If the term has no other constraints, it will treat the term as matching all nodes. This may result in pods being scheduled on incorrect nodes.

3. New pod creation:
	- API server validation will reject new pods using `matchCELExpressions` or `expression` in tolerations.

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

### Rollout, Upgrade and Rollback Planning


###### How can a rollout or rollback fail? Can it impact already running workloads?

**Rollout**: The feature enablement itself is safe and shouldn't impact existing workloads. It's an opt-in feature that only affects pods explicitly using `matchCELExpressions` for node affinity or `expressions` for tolerations.

**Rollback**: Can impact workloads if not done carefully:

1. Running pods with `matchCELExpressions` and `expressions` fields: continue running (safe)
2. Pending pods with `matchCELExpressions` and `expressions` fields: become stuck in Pending state, as:
   - They remain in etcd but validation rejects them
   - The scheduler won't recognize the fields
   - Force deletion may be required: `kubectl delete pod <name> --force --grace-period=0`
3. Workload controllers (Deployments, StatefulSets, etc.):
   - If the pod template uses `matchCELExpressions` and `expressions` fields, the controller cannot create new pods
   - Rolling updates will fail
 
  **Recommended rollback procedure to prevent hot loop**:
  1. Update identified workloads to remove CEL expressions fields
  2. Delete pending pods that use `matchCELExpressions` and `expressions` fields
  3. Disable feature gate in kube-scheduler first, then kube-apiserver


###### What specific metrics should inform a rollback?

- `scheduler_scheduling_duration_seconds`: Increased scheduling latency may indicate performance issues with CEL compiling
- `apiserver_request_total`: Spike in validation errors may indicate controller hot-loops

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?


###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

1. **Metrics**:

```promql
   # Number of pods evaluated by TaintToleration plugin
   scheduler_plugin_evaluation_total{plugin="TaintToleration"} > 0
   
   # Monitor rate of pods rejected by TaintToleration plugin
   rate(scheduler_plugin_evaluation_total{plugin="TaintToleration", status=~"Unschedulable.*"}[5m])
   
   # Rate of successful evaluations
   rate(scheduler_plugin_evaluation_total{plugin="TaintToleration", status="Success"}[5m])
   
   # Plugin execution duration
   rate(scheduler_framework_extension_point_duration_seconds{plugin="TaintToleration"}[5m])
```

2. **API Queries**:

```bash
	kubectl get pods -A -o jsonpath='{range .items[*]}{.metadata.namespace}/{.metadata.name}: {.spec.tolerations[?(@.expression)]}{"\n"}{end}' | grep -v ": \[\]$"
	kubectl get pods -A -o jsonpath='{range .items[*]}{.metadata.namespace}/{.metadata.name}: {.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[*].matchCELExpressions}{"\n"}{end}' | grep -v ": \[\]$"
	kubectl get pv -o jsonpath='{range .items[*]}{.metadata.name}: {.spec.nodeAffinity.required.nodeSelectorTerms[*].matchCELExpressions}{"\n"}{end}' | grep -v ": \[\]$"
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
	- Event Message:
		- For Tolerations: "0/X nodes are available: X node(s) had untolerated taint {<key>: <value>}".
		- For Node Affinity: "0/X nodes are available: X node(s) didn't match Pod's node affinity/selector"
- [x] API Verification
	- Observe tolerations with `expressions` field on pods
	- Observe node affinity with `matchCELExpressions` on pods

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
  - Metric name:
    - `scheduler_framework_extension_point_duration_seconds`
    - `scheduler_plugin_evaluation_total`
    - Components exposing the metric: `kube-scheduler`

###### Are there any missing metrics that would be useful to have to improve observability of this feature?


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

Potentially yes.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

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

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
