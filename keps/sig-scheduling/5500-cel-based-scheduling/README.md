# KEP-5500: CEL Based Comparisons to Tolerations and NodeAffinity

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1 — Tolerate taints based on semantic version comparison](#story-1--tolerate-taints-based-on-semantic-version-comparison)
    - [Story 2 — Node affinity matching labels starts with specific string](#story-2--node-affinity-matching-labels-starts-with-specific-string)
    - [Story 3 — Node affinity with semantic version comparison](#story-3--node-affinity-with-semantic-version-comparison)
    - [Story 4 — PersistentVolume node affinity with CEL expression](#story-4--persistentvolume-node-affinity-with-cel-expression)
    - [Story 5 — Matching multiple node taints](#story-5--matching-multiple-node-taints)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Feature Gate](#feature-gate)
  - [API Validation](#api-validation)
    - [Examples](#examples)
  - [Scheduler Logic](#scheduler-logic)
    - [CEL Compiler and Cache](#cel-compiler-and-cache)
    - [TaintToleration Plugin](#tainttoleration-plugin)
    - [NodeAffinity Plugin](#nodeaffinity-plugin)
    - [Other Affected Plugins](#other-affected-plugins)
    - [Semantics](#semantics)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
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

This enhancement introduces support for Common Expression Language (CEL) in Taint/Toleration and NodeAffinity. The KEP adds a new `matchCELExpressions` field to **core/v1 NodeSelectorTerm**, enabling CEL expressions for node selection in **Pod NodeAffinity** and **PersistentVolume NodeAffinity**. Additionally, a new `expression` field is added to **core/v1 Toleration** for CEL-based taint matching.

CEL expressions provide a powerful and extensible mechanism for expressing complex scheduling constraints, including semantic version comparisons, string manipulation, and compound logical conditions all within a single, validated expression.

## Motivation

CEL provides a standardized, extensible expression language already used throughout Kubernetes (ValidatingAdmissionPolicy, CRD validation, authorization). By introducing CEL for scheduling constraints, users gain:

- **Semantic version comparisons**: Schedule pods based on kubelet version, kernel version, or driver versions using expressions like `semver.compare(node.labels['node.example/kubelet-version'], '>=1.28.0')`
- **Compound conditions**: Express multiple conditions in a single expression, such as `semver.compare(node.labels['operating-system.example/kernel-version'], '>=5.10') && semver.compare(node.labels['another.example/container-runtime-version'], '>=2.0')`
- **String operations**: Parse and manipulate label values directly, for example extracting version numbers from prefixed strings using `node.labels['runtime'].split('://')[1]`
- **Flexible taint matching**: Match taints using CEL expressions with access to `taint.key`, `taint.value`, `taint.effect`, and `taint.timeAdded`

### Goals

- Allow CEL expression evaluation for NodeAffinity by adding `matchCELExpressions` field to `core/v1.NodeSelectorTerm` that accepts CEL expressions for node selection.
- Allow CEL expression evaluation for Tolerations by adding `expression` field to `core/v1.Toleration` that accepts CEL expressions for taint matching.
- Provide builtin CEL functions for common scheduling use cases, including functions supported by Kubernetes CEL.
- Ensure CEL expressions are validated at admission time for syntax correctness, type safety, and cost limits.
- Gate the feature behind `TaintTolerationNodeAffinityCEL` feature flag, disabled by default in Alpha.
- Maintain backward compatibility.

### Non-Goals

- Replacing existing NodeAffinity `matchExpressions` or `matchFields`.
- Replacing existing Toleration fields.
- Providing CEL support for inter-pod affinity/anti-affinity.

## Proposal

### User Stories

#### Story 1 — Tolerate taints based on semantic version comparison

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

#### Story 2 — Node affinity matching labels starts with specific string

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
          - "node.labels['topology.kubernetes.io/rack'].startsWith('us-west-2')"
  containers:
  - name: app
    image: myapp:latest
```

#### Story 3 — Node affinity with semantic version comparison

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

#### Story 4 — PersistentVolume node affinity with CEL expression

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

#### Story 5 — Matching multiple node taints

As a cluster operator I manage in my cluster a node that has multiple taints that start with the same prefix `node.example/` I want to be able to tolerate specific pods to all of these taints without iterating on each of them.

**Example Configuration:**

```yaml
apiVersion: v1
kind: Node
metadata:
  name: node-1
spec:
  taints:
  - key: "node.example/taint1"
    value: "foo"
    effect: NoSchedule
  - key: "node.example/taint2"
    value: "foo"
    effect: NoSchedule
  - key: "node.example/taint3"
    value: "foo"
    effect: NoSchedule
  - key: "node.example/taint4"
    value: "foo"
    effect: NoSchedule
---
apiVersion: v1
kind: Pod
metadata:
  name: compatible-workload
spec:
  tolerations:
  - expression: "taint.key.startsWith('node.example/')"
  containers:
  - name: app
    image: myapp:latest
```

### Notes/Constraints/Caveats (Optional)

- **CEL Expression Cost Limits and Expression Length**: CEL expressions are subject to cost limits to prevent resource exhaustion. Expressions that exceed the cost budget will be rejected at admission time. The maximum expression length is 10 KiB, and the estimated cost limit for evaluation is based on logical steps.
- **CEL Environment for Node Affinity**: The CEL environment for `matchCELExpressions` in NodeSelectorTerm exposes the following variable:
  - `node.labels` - A map of the node's labels (type: `map(string, string)`)
- **CEL Environment for Tolerations**: The CEL environment for the `expression` field in Tolerations exposes the following variables:
  - `taint.key` - The taint key (type: `string`)
  - `taint.value` - The taint value (type: `string`)
  - `taint.effect` - The taint effect (type: `string`)
  - `taint.timeAdded` - The time when the taint was added (type: `timestamp`)
- **CEL Libraries Available**: The standard Kubernetes CEL environment is available, which includes CEL standard functions and macros, as well as the Kubernetes extension library.
- **Expression Evaluation Failure Behavior**: If a CEL expression fails to evaluate at scheduling time, the behavior depends on context:
  - For `requiredDuringSchedulingIgnoredDuringExecution`: The node is considered non-matching, and the pod cannot schedule on that node.
  - For `preferredDuringSchedulingIgnoredDuringExecution`: The term contributes 0 to the node's score.
  - For tolerations with `expression`: The toleration does not match the taint.
- **Mutual Exclusivity**: The `expression` field in Tolerations is mutually exclusive with the existing `operator`, `key`, and `value` fields. If `expression` is set, the other fields must be empty.
- **Alpha Restrictions**: When `TaintTolerationNodeAffinityCEL=false`, the API server rejects pods and PersistentVolumes using `matchCELExpressions` in node affinity, and pods using `expression` in tolerations.
- **Immutability**: CEL expressions in `matchCELExpressions` and toleration `expression` fields follow the same immutability rules as other scheduling constraints they cannot be modified after pod creation.
- **CEL Expression Caching**: Adding LRU caches for compiled CEL expressions for both NodeAffinity and Tolerations, the expressions are compiled once and cached for reuse across scheduling cycles. 

### Risks and Mitigations

1. Scheduler Performance Regression

**Risk**:

CEL expression evaluation during taint/toleration matching or node affinity matching could degrade scheduler performance, especially in clusters with thousands of nodes and complex expressions.

**Mitigation**:

The scheduler will maintain separate LRU caches for both node affinity and toleration expression which will reduce the risk of recompiling expressions during scheduling cycles, as well as adding cost limit and expression length limits at validation time.

2. CEL Expression Evaluation Errors

**Risk**:

CEL expressions may fail at scheduling time, which can lead to:
- Pods stuck in `Pending` state if all nodes fail expression evaluation
- Silent scheduling failures for `preferredDuringSchedulingIgnoredDuringExecution` affinity

**Mitigation**:

This can be mitigated at admission time validation by validating syntax errors and type mismatches, for example when admitting a pod that uses a toleration expression `"taint.key == foo && taint.value == 'bar'"`:

```
spec.tolerations[0].expression: Invalid value: "taint.key == foo && taint.value == 'bar'": compilation failed: ERROR: <input>:1:14: undeclared reference to 'foo' (in container '')
 | taint.key == foo && taint.value == 'bar'
 | .............^
```
Expression evaluation failures at scheduling time are treated as non-matching for both NodeAffinity and Tolerations:

For **NodeAffinity**:
- For `requiredDuringSchedulingIgnoredDuringExecution`: Pod cannot schedule on that node
- For `preferredDuringSchedulingIgnoredDuringExecution`: Node receives 0 score contribution

For **Tolerations**:
- For tolerations: The toleration does not match the taint

## Design Details

### API Changes

1. **NodeSelectorTerm Changes**

A new `matchCELExpressions` field is added to `core/v1.NodeSelectorTerm`:

- **Field**: `matchCELExpressions []string` is a list of CEL expressions that must all evaluate to `true` for the term to match.
- **Feature Gate**: Gated by `TaintTolerationNodeAffinityCEL`.

2. **Toleration Changes**

A new `expression` field is added to `core/v1.Toleration`:

- **Field**: `expression string` is a single CEL expression that evaluates whether this toleration matches a taint.
- **Feature Gate**: Gated by `TaintTolerationNodeAffinityCEL`.


### Feature Gate

The `TaintTolerationNodeAffinityCEL` feature gate controls:
- API validation of `matchCELExpressions` and toleration `expression` fields.
- Scheduler evaluation of CEL expressions during scheduling.

When disabled, the API server rejects pods and PersistentVolumes using these fields.

### API Validation

Validation of CEL expressions occurs at admission time:

1. **Toleration Validation**:
  - A new validation case is added to toleration validation that will reject the `expression` field if the feature gate is disabled.
  - The pod is rejected if `expression` is used along with other fields `key`, `value`, `operator`, or `effect`.
  - The expression is rejected if compilation fails.
  - The expression is rejected if it exceeds expression length or cost limit.

2. **Pod NodeAffinity Validation**:
  - A new validation case is added to NodeAffinity that will reject the `matchCELExpressions` field if the feature gate is disabled.
  - The expression is rejected if compilation fails.
  - The expression is rejected if it exceeds expression length or cost limit. 

3. **PersistentVolume NodeAffinity Validation**:
  - The validation reuses the same Pod NodeAffinity validation logic.

4. **Backward Compatibility Validation**:
  - When updating an existing pod, validation allows CEL expressions if the old pod already used them, even if the feature gate is now disabled. This prevents breaking existing workloads during rollback.
  - For tolerations: If the old pod has any toleration with the `expression` field set, CEL expressions are allowed in the update.
  - For node affinity: If the old pod has any `NodeSelectorTerm` with `matchCELExpressions` set, CEL expressions are allowed in the update.
  - For PersistentVolumes: Similar logic applies - if the existing PV uses `matchCELExpressions` in its node affinity, updates are allowed.

#### Examples

- Toleration fails at validation because it exceeds the maximum expression length of 10KiB

```yaml
spec:
  tolerations:
  - expression: "taint.key == `value1` && taint.value == `value2` && taint.effect == `NoSchedule` && taint.key.startsWith(`prefix`) && taint.value.endsWith(`suffix`) && taint.key.contains(`mid`) && taint.value.contains(`inner`) && taint.key.size() > 0 && taint.value.size() > 0 //...very long expression"
```
API will respond with:

```
The Pod "example" is invalid: spec.tolerations[0].expression: Too long: may not be more than 10240 bytes
```

- NodeAffinity fails at validation because it exceeds maximum cost

```yaml
    nodeSelectorTerms:
    - matchCELExpressions:
      - 'node.labels.all(k, node.labels.all(v, k.matches(".*") && v.matches(".*")))'
```
API will respond with:
```
The Pod "example" is invalid: spec: Forbidden: too complex, exceeds cost limit
```

### Scheduler Logic

#### CEL Compiler and Cache

The feature introduces two separate CEL compilers and caches for tolerations and node affinity, each with their own CEL environment and variable definitions, each compiler will include all standard Kubernetes CEL libraries, and will perform cost estimation based on the declared field sizes.

- The Toleration compiler environment will expose the following variable:

`taint`:
   - `taint.key`
   - `taint.value`
   - `taint.effect`
   - `taint.timeAdded`

- The NodeAffinity compiler environment will expose the following variable:

`node`:
   - `node.labels`

Each cache is a thread-safe LRU cache that stores compiled CEL programs:

```go
type Cache struct {
    compileMutex keymutex.KeyMutex
    cacheMutex   sync.RWMutex
    cache        *lru.Cache
    compiler     *compiler
}
```
The compiler will perform cost estimation basead on the following criteria:

- **Maximum Length**: 10 KiB per expression.
- **Maximum Cost**: 1,000,000 cost units per expression evaluation.

#### TaintToleration Plugin

The TaintToleration plugin maintains a CEL cache and evaluates toleration expressions during the Filter phase. When a toleration has an `expression` field, it is evaluated against each taint using the cached compiled expression.

#### NodeAffinity Plugin

The NodeAffinity plugin maintains a separate CEL cache and evaluates `matchCELExpressions` during the Filter and Score phases. Each expression is compiled once and cached for reuse across scheduling cycles. The cache is passed to the final selector match.

#### Other Affected Plugins

The following plugins will also need to maintain a CEL cache since they make use of either toleration matching checks or affinity checks:

- **NodeUnschedulable Plugin**: will add a toleration CEL cache
- **VolumeBinding Plugin**: will add affinity CEL cache
- **PodTopologySpread Plugin**: will add both affinity and toleration CEL caches

#### Semantics

**Pod Node Affinity:**
- Each CEL expression in `matchCELExpressions` must evaluate to `true` for the term to match (AND semantics within a term).
- For `preferredDuringSchedulingIgnoredDuringExecution`: The weight is added only if all expressions evaluate to `true`.
- `matchCELExpressions` can be combined with `matchExpressions` and `matchFields` in the same term. All must match.
- If a CEL expression fails to evaluate, the term is considered non-matching.

**PersistentVolume Node Affinity:**

PersistentVolume node affinity uses the same `NodeSelectorTerm` structure and supports `matchCELExpressions` identically to Pod node affinity. The matching logic is shared through the common implementation.

**Tolerations:**
- The CEL expression is evaluated against each taint on the node.
- The expression must evaluate to `true` for the toleration to match that specific taint.
- For `NoSchedule` taints: Pod cannot schedule if no toleration matches.
- For `PreferNoSchedule` taints: Unmatched taints count against the node's score.
- For `NoExecute` taints: Running pods are evicted if no toleration matches.
- Expression evaluation failures are treated as non-matching.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

###### Existing Coverage
- `k8s.io/component-helpers/scheduling/corev1/`: `2026-02-03` - `100%`
- `k8s.io/component-helpers/scheduling/corev1/nodeaffinity`: `2026-02-03` - `95.4%`
- `k8s.io/component-helpers/storage/volume`: `2026-02-03` - `77.6%`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodeunschedulable`: `2026-02-03` - `84.4%`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/podtopologyspread`: `2026-02-03` - `88.1%`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/tainttoleration`: `2026-02-03` - `85.9%`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodeaffinity`: `2026-02-03` - `83.2%`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/volumebinding`: `2026-02-03` - `83.9%`
- `k8s.io/kubernetes/pkg/apis/core/validation/`: `2026-02-03` - `85.3%`
- `k8s.io/kubernetes/pkg/api/pod`: `2026-02-03` - `80.3%`

##### Integration tests

The following integration tests will be added or extended to cover CEL expressions for both affinity and toleration:

- `test/integration/scheduler/filters/filters_test.go`:
  - Extend `TestTaintTolerationFilter` to include test cases to test using `expression` with feature gate enabled/disabled scenarios
  - Extend `TestNodeAffinityFilter` to include test cases for CEL expressions in `matchCELExpressions`

- `test/integration/scheduler/scoring/priorities_test.go`:
  - Extend `TestTaintTolerationScoring` to verify scoring behavior with `PreferNoSchedule` taints when using CEL expressions
  - Extend `TestNodeAffinityScoring` to verify scoring behavior with `preferredDuringSchedulingIgnoredDuringExecution` using `matchCELExpressions`

- `test/integration/scheduler/scheduler_test.go`:
  - Add `TestTaintTolerationCELExpression` to test end-to-end scheduling with CEL expressions

- `test/integration/scheduler/volumebinding/volume_binding_test.go`:
  - Extend volume binding tests to verify PersistentVolume node affinity matching with CEL Expressions

##### e2e tests

The existing e2e tests will be extended to cover the new taints cases and new affinity cases introduced in this KEP:

- **Node Taints e2e Tests:** (test/e2e/node/taints.go)
- **Scheduler Taints e2e Tests:** (test/e2e/scheduling)
- **PersistentVolume e2e Tests:** (test/e2e/storage/persistent_volumes.go)

### Graduation Criteria

#### Alpha

- Feature implemented behind `TaintTolerationNodeAffinityCEL` feature gate (disabled by default)
- API validation for version operators in place
- Taint/toleration matching logic supports CEL expressions
- Node affinity matching logic supports CEL expressions

#### Beta

- Feature enabled by default
- Feedback collected from early adopters in SIG-Scheduling
- Performance testing shows that there is no significant scheduler latency increase or memory usage increase.
- Stress testing.

#### GA

- Evidence of real-world adoption.
- Complete scalability validation.

### Upgrade / Downgrade Strategy

#### Upgrade
  Enable the feature gate in kube-apiserver first then kube-scheduler. This ensures the API server can accept and validate pods with the CEL expressions before the kube-scheduler tries to process them.

#### Downgrade
  Disable the feature gate in kube-scheduler first, then kube-apiserver. Since we want to stop the kube-scheduler from processing the CEL expressions first, then stop the API server from accepting new pods with CEL expressions. This prevents the scheduler from trying to handle features the API server would reject.
  
**What happens when the scheduler doesn't recognize CEL expression field for tolerations:**

When the feature gate is disabled and the scheduler encounters a pod `expression` field for tolerations:
- The expression field is dropped during deserialization.
- The Toleration object is interpreted with default values (Operator: Equal, Key: empty).
- This "empty" toleration fails to match any standard node taint.
- Filter returns `UnschedulableAndUnresolvable` status
- Pod remains in Pending state.

**What happens when the scheduler doesn't recognize matchCELExpressions field for nodeAffinity:**

When the feature gate is disabled and the scheduler encounters a pod with `matchCELExpressions` field for node affinity:
- The `matchCELExpressions` field is dropped during deserialization.
- The NodeSelectorTerm is interpreted as having empty matchExpressions and empty matchFields.

### Version Skew Strategy

The skew between kubelet and control-plane components is not impacted. The kube-scheduler is expected to match the kube-apiserver minor version, but may be up to one minor version older (to allow live upgrades).

In the release where it is added, the feature is disabled by default and not recognized by other components.

Whoever enables the feature manually takes the risk of components like kube-scheduler being old and not recognizing the fields.

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

1. Already-running pods: Continue running normally for pod affinity rules, however pods that were already tolerating a `NoExecute` node taints will be evicted.

2. Unscheduled/pending pods:
	- **Tolerations**: The scheduler will ignore the `expression` field, fail to match the taint, and the Pod will remain Pending.
	- **Node Affinity**: The scheduler will ignore the `matchCELExpressions` field. If the term has no other constraints, it will treat the term as matching all nodes. This may result in pods being scheduled on incorrect nodes.

3. New pod creation:
	- API server validation will reject new pods using `matchCELExpressions` or `expression` in tolerations.

4. Pod updates: For both Affinity and Toleration the feature will make sure to detect that CEL has been in use in the Pod Validation Options and updates will be allowed.

###### What happens if we reenable the feature if it was previously rolled back?

Pods that were created with CEL expressions while the feature was enabled will resume normal scheduling behavior. The scheduler will recognize and evaluate the `matchCELExpressions` and `expression` fields again. No special migration or manual intervention is required.

###### Are there any tests for feature enablement/disablement?

Yes, the following unit tests will be added to cover feature gate enablement/disablement scenarios:

- `pkg/api/pod/util_test.go`:
  - A test will be added to verify backward compatibility when the feature gate is disabled but existing pods already use CEL expressions. This will cover scenarios for tolerations and node affinity with feature gate enabled/disabled.

- `pkg/apis/core/validation/validation_test.go`:
  - A test will be added to verify validation of tolerations with `expression` field when the feature gate is enabled vs disabled.
  - A test will be added to verify validation of node affinity `NodeSelectorRequirement` with `matchCELExpressions` field when the feature gate is enabled vs disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

**Rollout**: The feature enablement itself is safe and shouldn't impact existing workloads. It's an opt-in feature that only affects pods explicitly using `matchCELExpressions` for node affinity or `expression` for tolerations.

**Rollback**: Can impact workloads if not done carefully:

1. Running pods with `matchCELExpressions` and `expression` fields: continue running (safe) with an exception for pods were already tolerating a `NoExecute` taint which they will be evicted.
2. Pending pods with `matchCELExpressions` and `expression` fields: become stuck in Pending state, as:
   - They remain in etcd but validation rejects them
   - The scheduler won't recognize the fields
   - Force deletion may be required: `kubectl delete pod <name> --force --grace-period=0`
3. Workload controllers (Deployments, StatefulSets, etc.):
   - If the pod template uses `matchCELExpressions` and `expression` fields, the controller cannot create new pods
   - Rolling updates will fail

**Recommended rollback procedure to prevent hot loop**:
1. Update identified workloads to remove CEL expression fields
2. Delete pending pods that use `matchCELExpressions` and `expression` fields
3. Disable feature gate in kube-scheduler first, then kube-apiserver

###### What specific metrics should inform a rollback?

- `scheduler_scheduling_duration_seconds`: Increased scheduling latency may indicate performance issues with CEL compiling
- `apiserver_request_total`: Spike in validation errors may indicate controller hot-loops

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Manual testing will be performed during Alpha:
- Upgrade: Enable the feature gate, create pods with CEL expressions, verify scheduling works correctly.
- Downgrade: Disable the feature gate, verify existing pods continue running but new pods with CEL expressions are rejected.
- Re-upgrade: Re-enable the feature gate, verify pending pods with CEL expressions can now be scheduled.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Operator can use API queries to determine if the fields are used in either NodeAffinity or Toleration:

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
	- Observe tolerations with `expression` field on pods
	- Observe node affinity with `matchCELExpressions` on pods

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The scheduler should maintain the same SLOs as before.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name:
    - `scheduler_framework_extension_point_duration_seconds`
    - `scheduler_plugin_evaluation_total`
    - Components exposing the metric: `kube-scheduler`

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No additional metrics are planned for Alpha.

### Dependencies


###### Does this feature depend on any specific services running in the cluster?

N/A

### Scalability

###### Will enabling / using this feature result in any new API calls?

No, the feature is designed to be an enhancement to existing logic without introducing any new API communication patterns.

###### Will enabling / using this feature result in introducing new API types?

The KEP adds a new `matchCELExpressions` field to **core/v1 NodeSelectorTerm**, and `expression` field to existing API type **core/v1 Toleration**

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

This feature does not change the behavior when the API server and/or etcd is unavailable. The scheduler already depends on the API server for pod and node information. If the API server is unavailable, scheduling operations are paused regardless of this feature.

###### What are other known failure modes?

- CEL expression evaluation fails at scheduling time due to missing or malformed node labels/taint values (e.g., invalid semver string for `semver.compare()`).
  - Detection: Pods stuck in `Pending` state with `FailedScheduling` events. Monitor `scheduler_plugin_evaluation_total` for unexpected spikes.
  - Mitigations: Expression evaluation failures are treated as non-matching, so pods remain pending but already running workloads are unaffected. Users can update pod specs to use expressions that handle edge cases.
  - Diagnostics: Scheduler logs will show CEL evaluation errors with the specific expression and error message. Events on the pod will indicate which nodes failed matching.
  - Testing: Unit tests cover various CEL evaluation failure scenarios including type mismatches, missing variables, and invalid CEL expressions.

- Controller hot-loop when feature gate is disabled but workloads with CEL expressions exist.
  - Detection: Monitor `apiserver_request_total` for spikes in validation errors from controllers attempting to recreate pods with CEL expressions.
  - Mitigations: Before disabling the feature gate, update all Deployments/StatefulSets/DaemonSets/Jobs to remove CEL expressions from pod templates. Delete pending pods that use CEL expressions.
  - Diagnostics: API server logs will show repeated validation rejection messages for pods with `matchCELExpressions` or toleration `expression` fields.
  - Testing: Integration tests verify behavior when feature gate is toggled off while pods with CEL expressions exist.

- Scheduler performance degradation with complex or numerous CEL expressions.
  - Detection: Monitor `scheduler_framework_extension_point_duration_seconds` for increased latency. Watch `scheduler_pending_pods` for queue buildup.
  - Mitigations: The LRU cache reduces repeated compilation overhead. Cost limits at validation time prevent overly expensive expressions.
  - Diagnostics: Enable scheduler profiling to identify slow plugin execution. Check cache hit rates in scheduler metrics.
  - Testing: Benchmark tests measure scheduling latency with various CEL expression complexities and cache scenarios.

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check `scheduler_framework_extension_point_duration_seconds` to identify if TaintToleration or NodeAffinity plugins are causing latency.
2. Review scheduler logs for CEL evaluation errors.
3. If performance is unacceptable, consider simplifying CEL expressions or reducing the number of pods using CEL-based scheduling.
4. As a last resort, disable the feature gate to revert to standard scheduling behavior.

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
