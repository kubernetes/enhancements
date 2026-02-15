# KEP-5500: CEL Based Comparisons to Tolerations

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1 — Tolerate a family of taints by key prefix](#story-1--tolerate-a-family-of-taints-by-key-prefix)
    - [Story 2 — Tolerate taints based on kernel version comparison](#story-2--tolerate-taints-based-on-kernel-version-comparison)
    - [Story 3 — Tolerate taints matching a regex pattern](#story-3--tolerate-taints-matching-a-regex-pattern)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [CEL Compiler and Cache](#cel-compiler-and-cache)
  - [API Validation](#api-validation)
    - [Examples](#examples)
  - [Scheduler Logic](#scheduler-logic)
    - [TaintToleration Plugin](#tainttoleration-plugin)
      - [Toleration Seconds](#toleration-seconds)
    - [Other Affected Plugins and Components](#other-affected-plugins-and-components)
    - [Semantics For CEL Toleration Matching](#semantics-for-cel-toleration-matching)
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
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This enhancement introduces support for Common Expression Language (CEL) in Taint/Toleration. The KEP adds a new `expression` field to **core/v1 Toleration** for CEL-based taint matching.

CEL expressions provide an extensible mechanism for expressing scheduling constraints that are not possible with the existing toleration operators, including semantic version comparisons, string manipulation, and compound logical conditions all within a single, validated expression.

## Motivation

Kubernetes tolerations currently support only a small set of operators: `Equal` and `Exists` and the newly introduced numeric operators in [KEP-5471](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5471-enable-sla-based-scheduling). While sufficient for many use cases, these operators lack flexibility for cases where the user requires taint matching based on complex logic, value comparisons, or compound conditions.

Common Expression Language (CEL) provides an extensible expression language already used throughout Kubernetes (ValidatingAdmissionPolicy, CRD validation, authorization, and DRA device selection). Kubernetes already extends the CEL standard library with a rich string library, semver library, regex library, and other libraries that allow the user to write expressive matching logic in a single, compact expression. Reusing CEL for tolerations keeps the API consistent with patterns that cluster operators already use in other parts of Kubernetes.

By introducing a CEL `expression` field on `core/v1.Toleration`, this KEP gives operators a way to write toleration rules that were previously not possible with the existing operators, while preserving full backward compatibility with the existing toleration fields.

### Goals

- Introduce a new `expression` field to `core/v1.Toleration` that accepts a CEL expression for taint matching.
- Implement CEL compiler and cache that are consistent with current implementations of CEL in Kubernetes.
- Maintain backward compatibility with the existing toleration fields.
- Ensure CEL expressions are validated at admission time for syntax correctness, type safety, and cost limits.
- Gate the feature behind a feature flag, disabled by default in Alpha.

### Non-Goals

- Replacing existing Toleration fields (`key`, `operator`, `value`, `effect`).
- Extending current Toleration operators (`Exists`, `Equal`, `Gt`, `Lt`).
- Providing CEL support for NodeAffinity.

## Proposal

Adding a new field to `core/v1.Toleration` called `expression`, which will allow the user to use CEL expression to match node taints.

### User Stories

#### Story 1 — Tolerate a family of taints by key prefix

As a cluster operator, I run a shared cluster where nodes are tainted with environment-specific keys like `env.example.com/dev`, `env.example.com/staging`, and `env.example.com/testing`. A log-collection DaemonSet needs to run on all nodes regardless of the environment taint. Today I have to add a separate toleration for each environment key, and update the DaemonSet manifest whenever a new environment is added. With CEL I can write a single toleration that matches any taint whose key starts with `env.example.com/`.

**Example Configuration:**

```yaml
apiVersion: v1
kind: Node
metadata:
  name: node-1
spec:
  taints:
  - key: "env.example.com/dev"
    effect: NoSchedule
  - key: "env.example.com/staging"
    effect: NoSchedule
---
apiVersion: v1
kind: Pod
metadata:
  name: log-collector
spec:
  tolerations:
  - expression: "taint.key.startsWith('env.example.com/')"
  containers:
  - name: collector
    image: log-collector:latest
```

#### Story 2 — Tolerate taints based on kernel version comparison

As a cluster operator, I taint nodes with their kernel version (e.g. `key: "node.example.com/kernel-version"`, `value: "5.4.0"`). Some workloads require kernel features only available in version 5.15.0 or higher. I want those workloads to only tolerate nodes running kernel 5.15.0+, so they are not scheduled on older nodes.

**Example Configuration:**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
spec:
  tolerations:
  - expression: "taint.key == 'node.example.com/kernel-version' && semver(taint.value).isGreaterThan(semver('5.15.0'))"
  containers:
  - name: app
    image: my-app:latest
```

#### Story 3 — Tolerate taints matching a regex pattern

As a cluster operator, I taint nodes based on their physical location using keys like `zone-a1-rack-03`, `zone-a2-rack-15`, `zone-b1-rack-07`, etc. I want a latency-sensitive workload to only run on nodes in zones a1 and a2. With the existing operators I cannot express a pattern match on taint keys. With CEL I can use a regex to match only the zones I need.

**Example Configuration:**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
spec:
  tolerations:
  - expression: "taint.key.matches('^zone-a[12]-rack-[0-9]+$')"
  containers:
  - name: app
    image: my-app:latest
```

### Notes/Constraints/Caveats

- **CEL Expression Cost Limits and Expression Length**: CEL expressions are subject to cost limits to prevent resource exhaustion. Expressions that exceed the cost budget will be rejected at admission time. The maximum expression length is 10 KiB, and the estimated cost limit for evaluation is based on logical steps.

- **CEL Environment for Tolerations**: The CEL environment for the `expression` field in Tolerations exposes the following variables:
  - `taint.key` - The taint key (type: `string`)
  - `taint.value` - The taint value (type: `string`)
  - `taint.effect` - The taint effect (type: `string`)

- **CEL Libraries Available**: The standard Kubernetes CEL environment is available, which includes CEL standard functions and macros, as well as the Kubernetes extension library.

- **Mutual Exclusivity**: The `expression` field in Tolerations is mutually exclusive with the existing `operator`, `key`, and `value` fields. If `expression` is set, the other fields must be empty.

- **Alpha Restrictions**: When `TaintTolerationCEL=false`, the API server rejects pods using `expression` in tolerations.

- **Immutability**: CEL expressions in the toleration `expression` field follow the same immutability rules as other scheduling constraints; they cannot be modified after pod creation.

- **CEL Expression Caching**: Adding LRU caches for compiled CEL expressions for Tolerations, the expressions are compiled once and cached for reuse across scheduling cycles. 

### Risks and Mitigations

1. Scheduler Performance Regression

CEL expression evaluation during taint/toleration matching could degrade scheduler performance, especially in clusters with thousands of nodes and complex expressions. The feature will follow the same CEL implementation for DRA that introduces an LRU cache for the CEL expressions which will avoid recompiling expressions for each taint matching and cached expressions will be reused throughout the scheduling cycles.

The feature will also add cost limits and expression length constraints which will reduce the risk of degrading the scheduler performance through complex expressions or complicated logic.

2. CEL Expression Evaluation Errors

CEL expressions may fail at scheduling time, which can lead to the pod stuck in `Pending` state, this can be mitigated by validating the CEL expression at admission time of the pod.

## Design Details

### API Changes

A new `expression` field is added to `core/v1.Toleration`:

- **Field**: `expression string` is a single CEL expression that evaluates whether this toleration matches a taint.

```go
type Toleration struct {
    ...
    // Expression is a CEL expression that evaluates whether this toleration matches a taint,
	// if set, the CEL expression will be evaluated to determine toleration matching.
    // The expression must evaluate to a boolean value.
	// +featureGate=TaintTolerationCEL
	// +optional
	Expression string
```
Two new constants will be added to the API to represent the expression length limit and the cost limit:

```go
// CELTolerationExpressionMaxCost specifies the cost limit for a single toleration
// CEL expression evaluation during pod scheduling.
// This is the same value as PerCallLimit in k8s.io/apiserver/pkg/apis/cel/config.go
// which gives roughly 0.1 second for each expression evaluation.
const CELTolerationExpressionMaxCost = 1000000

// CELTolerationExpressionMaxLength specifies the maximum length for CEL expression
// used in Toleration.
const CELTolerationExpressionMaxLength = 10 * 1024
```

Both values follow the same constraints used by DRA's CEL device selection in KEP-4381. The cost limit of 1,000,000 comes from the Kubernetes apiserver CEL `PerCallLimit` constant, which is the same limit used by ValidatingAdmissionPolicy and CRD validation as well.

### CEL Compiler and Cache

The feature introduces a CEL compiler and cache implementation for tolerations, following the same pattern used by DRA's implementation for device selection in [KEP-4381](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/4381-dra-structured-parameters). The compiler and cache will use the constraints constants described in the previous section. The compiler will include all standard Kubernetes CEL libraries. The toleration compiler environment will expose the following variables:

```go
type Taint struct {
	Key       string
	Value     string
	Effect    string
}
```
The cache is a thread-safe LRU cache that stores compiled CEL programs:

```go
type Cache struct {
    compileMutex keymutex.KeyMutex
    cacheMutex   sync.RWMutex
    cache        *lru.Cache
    compiler     *compiler
}
```

**Cost Estimation**

The cost limit is checked at two points:

1. **At admission time**: The compiler estimates the worst case cost of the expression based on the declared maximum sizes of the taint variables (`taint.key`, `taint.value`, `taint.effect`). If the estimated cost exceeds `CELTolerationExpressionMaxCost`, the pod admission will fail stating the expression is too complex.

2. **At evaluation time**: The compiled program has the same cost limit configured. When the expression is evaluated during scheduling, the CEL runtime tracks the actual cost of each operation and aborts if it exceeds the limit. This is needed because the compile time estimate is a worst case approximation and may underestimate the actual cost in some cases.

### API Validation

Validation of CEL expressions occurs at admission time:

1. **Toleration Validation**:
  - A new validation case is added to toleration validation that will reject the `expression` field if the feature gate is disabled.
  - The pod is rejected if `expression` is used along with other fields `key`, `value`, `operator`, or `effect`.
  - The expression is rejected if compilation fails.
  - The expression is rejected if it exceeds expression length or cost limit.

2. **Backward Compatibility Validation**:
  - When updating an existing pod, validation allows CEL expressions if the old pod already used them, even if the feature gate is now disabled. This prevents breaking existing workloads during pod updates.
  - If the old pod has any toleration with the `expression` field set, CEL expressions are allowed in the update.

#### Examples

- Toleration fails at API validation because it exceeds the maximum expression length of 10KiB or cost limit budget.

```
The Pod "example" is invalid: spec.tolerations[0].expression: Too long: may not be more than 10240 bytes
```

- Toleration fails at API validation because `expression` field is used along with other toleration fields

```
The Pod "compatible-workload" is invalid: spec.tolerations[0].expression: Invalid value: "taint.key.startsWith('node.example/')": expression cannot be used with key, value, operator, or effect fields
```

### Scheduler Logic

#### TaintToleration Plugin

The TaintToleration plugin will initialize the CEL cache in PreFilter and evaluate toleration expressions during the Filter and Score phase. When a toleration has an `expression` field, it is evaluated against each taint using the cached compiled expression.

The plugin calls a helper function that does the actual toleration matching against node taints `helper.TolerationsTolerateTaint`, this where the cache will be passed and used to get or compile the CEL expression and evaluated against node taints.

##### Toleration Seconds

As the [API Validation](#api-validation) describes, each toleration will be validated such that the `expression` field can not be used along with `key`, `value`, `opertor`, or `effect` fields, however the `tolerationSeconds` can still be used, this will cause any expression that tolerated a `NoExecute` taint to follow the same logic and be evicted after the tolerationSeconds passes, an extra logic will be added to the taint eviction controller to support handling of the expression field if set for toleration.

#### Other Affected Plugins and Components

The following plugins and integration points also perform toleration matching and will use a CEL toleration cache. The scheduler creates a single shared cache at startup and passes it to all scheduler plugins via the framework. The DaemonSet controller, TaintEviction controller, and kubelet each initialize their own independent cache.

- **NodeUnschedulable Plugin**

The plugin calls the helper function `TolerationsTolerateTaint` in two occurrences, during the filter extension point to make sure that the pod tolerates the unschedulable taint, the second call happens when the pod tolerations are updated in the QueueingHint handler, both occurrences use the shared scheduler cache passed to the helper function to evaluate CEL tolerations against the unschedulable taint.

- **PodTopologySpread Plugin**

The plugin calls the helper function `FindMatchingUntoleratedTaint` when `NodeInclusionPolicyInPodTopologySpread` feature is enabled and the inclusion policy is set to honor node taints, the shared scheduler cache is passed to the helper function to evaluate the CEL expression if found for tolerations.

- **Scheduler EventHandler**

The scheduler event handler calls the toleration matching helper function `FindMatchingUntoleratedTaint` on node updates, the shared scheduler cache is passed to the helper function to evaluate tolerations with CEL expressions.

- **DaemonSet Controller**

The DaemonSet controller calls `FindMatchingUntoleratedTaint` in function `NodeShouldRunDaemonPod` to make sure that the pod can run on a node, the controller initializes its own CEL cache and passes it to the helper function to evaluate tolerations with CEL expressions.

- **TaintEviction Controller**

The TaintEviction controller calls helper function `GetMatchingTolerations` to make sure that the pod tolerates all node taints, the controller initializes its own CEL cache and passes it to the helper function to evaluate tolerations with CEL expressions.

- **Kubelet Lifecycle Predicate**

Kubelet calls `FindMatchingUntoleratedTaint` during lifecycle predicate checks, it initializes its own CEL cache and passes it to the helper function to evaluate tolerations with CEL expressions.

#### Semantics For CEL Toleration Matching

- The CEL expression is evaluated against each taint on the node.
- The expression must evaluate to `true` for the toleration to match that specific taint.
- Expression evaluation failures are treated as non-matching.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

###### Existing Coverage

- `k8s.io/component-helpers/scheduling/corev1/`: `2026-02-03` - `100%`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodeunschedulable`: `2026-02-03` - `84.4%`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/podtopologyspread`: `2026-02-03` - `88.1%`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/tainttoleration`: `2026-02-03` - `85.9%`
- `k8s.io/kubernetes/pkg/apis/core/validation/`: `2026-02-03` - `85.3%`
- `k8s.io/kubernetes/pkg/api/pod`: `2026-02-03` - `80.3%`
- `k8s.io/kubernetes/pkg/controller/daemon`: `2026-02-06` - `69.7%`
- `k8s.io/kubernetes/pkg/controller/tainteviction`: `2026-02-06` - `78.4%`
- `k8s.io/kubernetes/pkg/kubelet/lifecycle`: `2026-02-06` - `63.4%`

New tests will be added to cover the following:

- The toleration matching logic in `k8s.io/component-helpers/scheduling/corev1` pkg
- The API valication logic when the feature gate is enabled in `k8s.io/kubernetes/pkg/apis/core/validation` pkg
- The backward compatibility logic when the feature gate is disabled but existing pods already use CEL expressions in `k8s.io/kubernetes/pkg/api/pod` pkg
- The use of celcache in different plugins for `tainttoleration`, `podtopologyspread`, `nodeunschedulable` plugins
- The `TaintEviction` controller that handles the `tolerationSeconds` settings in each toleration
- The cel compiler and cache tests in `k8s.io/kube-scheduler`

##### Integration tests

The following integration tests will be added or extended to cover CEL expressions for toleration:

- `test/integration/scheduler/filters/filters_test.go`:
  - Extend `TestTaintTolerationFilter` to include test cases to test using `expression` with feature gate enabled/disabled scenarios

- `test/integration/scheduler/scoring/priorities_test.go`:
  - Extend `TestTaintTolerationScoring` to verify scoring behavior with `PreferNoSchedule` taints when using CEL expressions

- `test/integration/scheduler/scheduler_test.go`:
  - Add `TestTaintTolerationCELExpression` to test end-to-end scheduling with CEL expressions

##### e2e tests

The existing e2e tests will be extended to cover the new toleration cases introduced in this KEP:

- **Node Taints e2e Tests:** (test/e2e/node/taints.go)
- **Scheduler Taints e2e Tests:** (test/e2e/scheduling)

### Graduation Criteria

#### Alpha

- Feature implemented behind `TaintTolerationCEL` feature gate (disabled by default)
- API validation for CEL expressions in place
- Taint/toleration matching logic supports CEL expressions

#### Beta

- Feature enabled by default
- Feedback collected from early adopters in SIG-Scheduling
- Performance testing shows that there is no significant scheduler latency increase or memory usage increase.
- Stress testing.

#### GA

- TBD in Beta release.

### Upgrade / Downgrade Strategy

#### Upgrade
Enable the feature gate on kubelet first, then kube-controller-manager, then kube-scheduler, and finally kube-apiserver. This ensures all consuming components can handle CEL tolerations before the API server starts accepting them.

#### Downgrade
Disable the feature gate on kube-apiserver first to stop new CEL tolerations from entering the system, then kube-scheduler, kube-controller-manager, and finally kubelet.
  
**What happens when the scheduler doesn't recognize CEL expression field for tolerations:**

When the feature gate is disabled and the scheduler encounters a pod `expression` field for tolerations:
- The expression field is dropped during deserialization.
- The Toleration object is interpreted with default values (Operator: Equal, Key: empty).
- This "empty" toleration fails to match any standard node taint.
- Filter returns `UnschedulableAndUnresolvable` status
- Pod remains in Pending state.

### Version Skew Strategy

The feature impacts four components in total: `kube-apiserver`, `kube-scheduler`, `kube-controller-manager`, and `kubelet` as described in [Other Affected Plugins and Components](#other-affected-plugins-and-components) section. The feature gate must be enabled on all four components for the feature to be fully functional, there are different scenarios that need to be covered when versions differ or when the feature gate is enabled on some components but not others.

1. New version of `kube-apiserver`

For older versions of `kube-scheduler` or if the feature is disabled the API server will accept and persist pods with `expression` field, however the scheduler will not recognize the `expression` field and will treat the toleration as an empty toleration with no key, effect, value, or operator. Pods relying solely on CEL tolerations to tolerate `NoSchedule` or `NoExecute` taints will remain Pending. 

For older versions `kube-controller-manager` or if the feature is disabled the API server will accept and persist pods with `expression` field, however the controller-manager will not recognize the field in two situations:

- **Taint Eviction Controller**: The CEL toleration will be treated as non-matching. Running pods that rely on a CEL expression to tolerate a `NoExecute` taint will be **incorrectly evicted** because the controller does not understand the expression match.

- **DaemonSet Controller**: Without CEL support, CEL tolerations are treated as non-matching. The controller may incorrectly decide not to create DaemonSet pods on nodes where the CEL toleration should match, resulting in DaemonSet pods missing from nodes they should run on.

For `kubelet` without the feature: the API server will accept the pods with CEL tolerations and the scheduler (if enabled) will correctly evaluate and schedule the pod to a node, however the kubelet will fail to match the taint in the lifecycle predicate admission check. The pod would be rejected by the kubelet at admission even though the scheduler placed it correctly. Users must ensure the feature is enabled on kubelet before scheduling pods with CEL tolerations to those nodes.

2. `kube-apiserver` does not have the feature, other components do

The API server will reject any new pods with `expression` field at validation time, so no CEL tolerations can enter the system and other components are not impacted. If already existing pods with CEL tolerations are found in the system (e.g. after an API server rollback), the scheduler, controller-manager, and kubelet will evaluate the CEL expressions normally since they have the feature enabled.

The correct enablement order of the feature is as follows:

1. **kubelet** 
2. **kube-controller-manager** 
3. **kube-scheduler** 
4. **kube-apiserver** 

The disablement order should be the reverse of the previous list: disable on `kube-apiserver` first to stop new CEL tolerations from entering the system, then `kube-scheduler`, `kube-controller-manager`, and finally `kubelet`.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `TaintTolerationCEL`
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-scheduler
    - kube-controller-manager
    - kubelet

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Impact on existing pods with CEL fields when feature is disabled:

1. Already-running pods: Continue running normally for Pod tolerations rules, however pods that were already tolerating `NoExecute` node taints will be evicted.
2. Unscheduled/pending pods: The scheduler will ignore the `expression` field, fail to match the taint, and the Pod will remain Pending.
3. New pod creation: API server validation will reject new pods using `expression` in tolerations.
4. Pod updates:
   - Adding tolerations: If the feature gate is disabled, the API server will reject adding new tolerations with the `expression` field.
   - Removing tolerations: Removing tolerations with the `expression` field will always be allowed, regardless of the feature gate state.
5. Taint eviction controller: With the feature gate disabled on controller manager, the controller will not evaluate CEL expressions. Running pods that relied on a CEL expression to tolerate a `NoExecute` taint will be evicted.
6. DaemonSet controller: With the feature gate disabled, the controller will treat CEL tolerations as non-matching. DaemonSet pods that relied on CEL expressions to tolerate node taints may not be created on those nodes.
7. Kubelet: With the feature gate disabled, the kubelet lifecycle predicate will not evaluate CEL expressions. Pods that relied on a CEL expression to tolerate a taint will fail the lifecycle admission check.

###### What happens if we reenable the feature if it was previously rolled back?

Pods that were created with CEL expressions while the feature was enabled will resume normal scheduling behavior. The scheduler will recognize and evaluate the `expression` field again. No special migration or manual intervention is required.

###### Are there any tests for feature enablement/disablement?

Yes, the following unit tests will be added to cover feature gate enablement/disablement scenarios:

- `pkg/api/pod/util_test.go`:
  - A test will be added to verify backward compatibility when the feature gate is disabled but existing pods already use CEL expressions. This will cover scenarios for tolerations with feature gate enabled/disabled.

- `pkg/apis/core/validation/validation_test.go`:
  - A test will be added to verify validation of tolerations with `expression` field when the feature gate is enabled vs disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

**Rollout**: The feature enablement itself is safe and shouldn't impact existing workloads. It's an opt-in feature that only affects pods explicitly using `expression` for tolerations.

**Rollback**: Can impact workloads if not done carefully:

1. Running pods with `expression` field: continue running (safe) with an exception for pods that were already tolerating a `NoExecute` taint, which will be evicted.
2. Pending pods with `expression` fields: become stuck in Pending state, as:
   - They remain in etcd but validation rejects them
   - The scheduler won't recognize the fields
   - Force deletion may be required: `kubectl delete pod <name> --force --grace-period=0`
3. Workload controllers (Deployments, StatefulSets, etc.):
   - If the pod template uses `expression` field, the controller cannot create new pods and rolling updates will fail

**Recommended rollback procedure to prevent hot loop**:
1. Update identified workloads to remove CEL expression fields
2. Delete pending pods that use `expression` field
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

Operator can use API queries to determine if the field is used in tolerations:

```bash
	kubectl get pods -A -o jsonpath='{range .items[*]}{.metadata.namespace}/{.metadata.name}: {.spec.tolerations[?(@.expression)]}{"\n"}{end}' | grep -v ": \[\]$"
```

###### How can someone using this feature know that it is working for their instance?

- [x] Events
	- Event Reason: FailedScheduling
	- Event Message:
		- "0/X nodes are available: X node(s) had untolerated taint {<key>: <value>}".
- [x] API Verification
	- Observe tolerations with `expression` field on pods

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

The KEP adds a new field `expression` to existing API type **core/v1 Toleration**

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes. The new `expression` string field on `core/v1.Toleration` increases the Pod object size. Each expression is limited to 10 KiB.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Potentially yes.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

CEL expression compilation and evaluation will consume CPU in all components that perform toleration matching. Each component initializes its own bounded LRU cache for compiled CEL expressions:

- `kube-scheduler`: A single shared cache is created at scheduler startup and shared across all scheduler plugins (TaintToleration, NodeUnschedulable, PodTopologySpread, EventHandler).
- `kube-controller-manager`: The DaemonSet controller and the TaintEviction controller each initialize their own cache.
- `kubelet`: The lifecycle predicate handler initializes its own cache.

The caches should reduce CPU time by reusing compiled expressions instead of recompiling on every evaluation. Each cache is bounded by size to limit memory usage.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

This feature does not change the behavior when the API server and/or etcd is unavailable. The scheduler already depends on the API server for pod and node information. If the API server is unavailable, scheduling operations are paused regardless of this feature.

###### What are other known failure modes?

- CEL expression evaluation fails at scheduling time due to missing or malformed node taint values (e.g., invalid semver string passed to `semver()`).
  - Detection: Pods stuck in `Pending` state with `FailedScheduling` events. Monitor `scheduler_plugin_evaluation_total` for unexpected spikes.
  - Mitigations: Expression evaluation failures are treated as non-matching, so pods remain pending but already running workloads are unaffected.
  - Diagnostics: Scheduler logs will show CEL evaluation errors with the specific expression and error message. Events on the pod will indicate which nodes failed matching.
  - Testing: Unit tests cover various CEL evaluation failure scenarios including type mismatches, missing variables, and invalid CEL expressions.

- Controller hot-loop when feature gate is disabled but workloads with CEL expressions exist.
  - Detection: Monitor `apiserver_request_total` for spikes in validation errors from controllers attempting to recreate pods with CEL expressions.
  - Mitigations: Before disabling the feature gate, update all Deployments/StatefulSets/DaemonSets/Jobs to remove CEL expressions from pod templates. Delete pending pods that use CEL expressions.
  - Diagnostics: API server logs will show repeated validation rejection messages for pods with toleration `expression` fields.
  - Testing: Integration tests verify behavior when feature gate is toggled off while pods with CEL expressions exist.

- Scheduler performance degradation with complex or numerous CEL expressions.
  - Detection: Monitor `scheduler_framework_extension_point_duration_seconds` for increased latency. Watch `scheduler_pending_pods` for queue buildup.
  - Mitigations: The LRU cache reduces repeated compilation overhead. Cost limits at validation time prevent overly expensive expressions.
  - Diagnostics: Enable scheduler profiling to identify slow plugin execution. Check cache hit rates in scheduler metrics.
  - Testing: Benchmark tests measure scheduling latency with various CEL expression complexities and cache scenarios.

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check `scheduler_framework_extension_point_duration_seconds` to identify if TaintToleration or other plugins are causing latency.
2. Review scheduler logs for CEL evaluation errors.
3. If performance is unacceptable, consider simplifying CEL expressions or reducing the number of pods using CEL-based scheduling.
4. As a last resort, disable the feature gate to revert to standard scheduling behavior.

## Implementation History

- `2026-01-22`: Initial KEP Implementation

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

1. Wildcard matching in toleration keys This was proposed in [KEP-5869](https://github.com/kubernetes/enhancements/pull/5880).this approach covers prefix matching for taint keys, however it can't handle version comparisons, compound conditions, or value-based matching.

2. Adding more built-in operators to tolerations similar to [KEP-5471](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5471-enable-sla-based-scheduling) which introduces `Gt`, and `Lt` operators toleration can support other operators that can provide more functionality for taint matching.
