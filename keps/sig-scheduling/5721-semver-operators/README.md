# KEP-5721: Semantic Version Comparison Operators for Tolerations and NodeAffinity

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1 — Cluster operator ensures Kubelet version before scheduling pod](#story-1--cluster-operator-ensures-kubelet-version-before-scheduling-pod)
    - [Story 2 — Tolerate running workloads on nodes with older versions of CNI](#story-2--tolerate-running-workloads-on-nodes-with-older-versions-of-cni)
    - [Story 3 — Container Runtime version based scheduling for sensitive pods](#story-3--container-runtime-version-based-scheduling-for-sensitive-pods)
    - [Story 4 — Persistent Volume node affinity based on kernel version](#story-4--persistent-volume-node-affinity-based-on-kernel-version)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Scheduler Performance Regression](#scheduler-performance-regression)
    - [Invalid SemVer Node Label or Taint](#invalid-semver-node-label-or-taint)
    - [Controller Hot-Loop When Feature Gate is Disabled](#controller-hot-loop-when-feature-gate-is-disabled)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Semantics](#semantics)
  - [Implementation](#implementation)
    - [Feature Gate Definition](#feature-gate-definition)
    - [API Validation](#api-validation)
    - [Scheduler Logic](#scheduler-logic)
      - [Toleration Logic](#toleration-logic)
      - [Node Affinity Logic For MatchExpressions](#node-affinity-logic-for-matchexpressions)
      - [Persistent Volumes NodeAffinity](#persistent-volumes-nodeaffinity)
      - [Additional Changes](#additional-changes)
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

This enhancement introduces Semantic Versioning (SemVer) comparison capabilities to Kubernetes scheduling and storage. Similar to how KEP-5471 introduces integer operators (Lt, Gt) for Tolerations, this KEP adds `SemverLt`, `SemverGt`, and `SemverEq` operators to **core/v1 Toleration**, **core/v1 NodeAffinity**, and **core/v1 PersistentVolume NodeAffinity**. This enables granular control over workload placement and volume attachment based on versioned node attributes (e.g., Kubelet version, kernel version, driver versions) without requiring manual enumeration of all target versions.

## Motivation

Many scheduling decisions and storage attachment decisions depend on component versions (e.g., kernel versions, or driver versions). However, the current scheduling framework restricts NodeSelector and Tolerations to set-based operators (In, NotIn, Exists, DoesNotExist). Similarly, PersistentVolume node affinity is also restricted to the same set-based operators.

This restriction forces users to treat ordered versions as unrelated strings. To target a node with a version "greater than X" users will want concrete semantic versioning comparisons. This applies to both pod scheduling decisions and persistent volume attachment decisions where volumes may require specific node capabilities that are version-dependent (e.g. kernel features, storage driver versions).

### Goals

- Add semantic version based operators to tolerations so pods can match taints like `node.kubernetes.io/kubelet-version=v1.28.0` using `SemverGt`, `SemverLt`, or `SemverEq` operators.
- Add semantic version based operators `SemverGt`, `SemverLt`, and `SemverEq` to node affinity so that scheduling and volume attachment decisions can be based on version comparisons. This applies to Pod NodeAffinity and PersistentVolume NodeAffinity.
- Backward-compatible and opt-in via a feature gate.
- Zero operational performance impact on existing pod scheduling using `Equal` and `Exists` operators.

### Non-Goals

- Changing the behavior of existing NodeAffinity operators (In, NotIn, Exists, DoesNotExist, Gt, Lt).
- Changing the behavior of existing Toleration operators (Equal, Exists).
- Supporting non-SemVer versioning schemes (e.g., CalVer, custom version formats).

## Proposal

### User Stories (Optional)

#### Story 1 — Cluster operator ensures Kubelet version before scheduling pod

As a cluster operator, I want to schedule a Pod that relies on a specific Kubelet feature introduced in `v1.32.0`, So that it only lands on nodes that have been upgraded, regardless of the patch version.

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
        - matchExpressions:
          - key: "node.kubernetes.io/kubelet-version"
            operator: SemverGt
            values: ["v1.31.99"]
```
#### Story 2 — Tolerate running workloads on nodes with older versions of CNI

As a cluster operator, I am upgrading Calico CNI to version v3.28.0. During a phased upgrade of all nodes, I tainted nodes that are running an older CNI version. I run some workloads that can tolerate running on an older CNI. I will use semver operators to make sure that the old CNI node is tolerated by this workload.

**Example Configuration:**

```yaml
apiVersion: v1
kind: Node
metadata:
  name: old-cni-node-1
spec:
  taints:
  - key: "cni.projectcalico.org/version"
    value: "v3.27.2"
    effect: NoSchedule
---
apiVersion: v1
kind: Pod
metadata:
  name: tolerant-pod
spec:
  tolerations:
  - key: "cni.projectcalico.org/version"
    operator: "SemverLt"
    value: "v3.28.0"
    effect: "NoSchedule"
```
#### Story 3 — Container Runtime version based scheduling for sensitive pods

As a cluster operator, I want to deploy a critical workload that utilizes Linux Usernamespaces. However, I still have nodes in the cluster that run containerd version v1.6 that does not support Linux usernamespaces. I want to make sure that the workload will only be deployed on containerd version greater than v2.0.0

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: modern-app
spec:
  # using linux usernamespaces
  hostUsers: false
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: "node.kubernetes.io/container-runtime-version"
            operator: "SemverGt"
            values: ["2.0.0"]
```
#### Story 4 — Persistent Volume node affinity based on kernel version

As a storage administrator, I am managing persistent volumes that require specific kernel features available only in newer kernel versions. For example, a volume using advanced filesystem features needs kernel version 5.10.0 or higher. I want to ensure the persistent volume can only be attached to nodes running compatible kernel versions.

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
      - matchExpressions:
        - key: "node.kubernetes.io/kernel-version"
          operator: "SemverGt"
          values: ["5.10.0"]
  persistentVolumeReclaimPolicy: Retain
  storageClassName: advanced-storage
```

### Notes/Constraints/Caveats (Optional)

- **SemVer-Only Support**: The implementation will only support versioning comparisons based on SemVer specifications; other versioning schemes will be rejected by the API server during validation.

- **Node Affinity Single Element Requirement**: The implementation will validate that if the semantic version operators are used for node affinity, then NodeSelector values array must contain only one element, similar to the behavior of `Gt` and `Lt`.

- **Toleration Parsing Requirements**: The toleration value must be parseable as Semantic Versions for SemVer operators (`SemverLt`, `SemverGt`, `SemverEq`). If parsing fails, the toleration does not match.

- **Node Affinity Parsing Requirements**: The node affinity value must be parseable as Semantic Versions for SemVer operators (`SemverLt`, `SemverGt`, `SemverEq`). If parsing fails, node affinity matching rule does not match

- **NodeFieldSelector Limitation**: The semver comparison operators (`SemverLt`, `SemverGt`, `SemverEq`) are only supported in `matchExpressions` within NodeSelectorTerms. They are **not** supported in `matchFields` as field selectors have different validation requirements

- **Semver Tolerant Parsing**: The implementation will use `semver.ParseTolerant` to parse semver values which currently trims spaces, removes a "v" prefix, adds a 0 patch number to versions with only major and minor components specified, and removes leading 0s.

- **Non-Version Taint Values**: When a pod toleration uses `SemverLt`, `SemverGt`, or `SemverEq` operators, it only matches taints with SemVer values. If a node has a taint with a non-SemVer value, the toleration will not match, and the pod cannot schedule on that node.
  
  **Example**: 
  - Node taint: `node.kubernetes.io/containerRuntimeVersion=containerd://2.1.4:NoSchedule`
  - Pod toleration: `{key: "node.kubernetes.io/containerRuntimeVersion", operator: "SemverLt", value: "2.2.0"}`
  - **Result**: Toleration does not match and pod cannot schedule on this node because the container runtime version has a `containerd://` prefix
  - The pod remains `Pending` and can schedule on other nodes with valid version taints
  - The pod is not failed or rejected entirely

- **Non-Version Affinity Values**: When a pod node affinity uses `SemverLt`, `SemverGt`, or `SemverEq` operators, Label value in case of `MatchExpression` must be a parsable semver version otherwise the pod will not match scheduling requirements on these nodes.

- **Alpha Restrictions**: When `TaintTolerationNodeAffinitySemverComparisonOperators=false`, the API server rejects pods using the new operators in tolerations matching, and in node affinity matching.

- **Parsing Overhead**: Each taint/toleration or affinity rule match with semver operators requires semver parsing.

### Risks and Mitigations

#### Scheduler Performance Regression

**Risk**: version parsing during taint/toleration matching or node affinity matching rules could degrade scheduler performance, especially in clusters with thousands of taints and labels.

**Mitigation**:

- Parse versions only when new operators are used.
- Existing `Equal`/`Exists` operators execute identical code paths with no additional overhead.
- Consider caching parsed values in scheduler data structures if performance issues arise
- Feature gate allows disabling if performance problems occur

#### Invalid SemVer Node Label or Taint

**Risk**: Node labels and taints are currently free-form strings and are not validated for SemVer compliance at registration time. This creates two problematic scenarios:

1. **Invalid node-side values**: A node may carry a taint or a label like `node.kubernetes.io/containerRuntimeVersion=containerd://2.1.4` instead of `node.kubernetes.io/containerRuntimeVersion=2.1.4`

2. **Delayed detection**: Since node labels/taints are not validated at node registration time, misconfigurations are only detected during scheduling when a pod with `SemverLt`/`SemverGt`/`SemverEq` operators attempts to match against them.

This can lead to:
- Pods stuck in `Pending` state indefinitely
- Unclear error messages for cluster operators
- Silent scheduling failures for `preferredDuringSchedulingIgnoredDuringExecution` affinity (pod schedules but ignores the preference)

**Mitigation**:

**1. Pod-Side Validation (Admission Time)**

- **Tolerations**: API server validation strictly requires that toleration values using `SemverLt`, `SemverGt`, or `SemverEq` operators must be valid SemVer strings. Invalid values are rejected during pod admission:
  ```
  spec.tolerations[0].value: Invalid value: "containerd://2.1.4":
  Invalid character(s) found in major number "containerd:"
  ```

- **Node Affinity**: API server validation strictly requires that node affinity requirement values using `SemverLt`, `SemverGt`, or `SemverEq` operators must be valid SemVer strings. Invalid values are rejected during pod admission:
  ```
  spec.affinity.nodeAffinity...matchExpressions[0].values[0]: Invalid value: "v1.2.x":
  Invalid character(s) found in patch number "x"
  ```

This ensures that users cannot create pods with invalid SemVer values on the pod side.

**2. Node-Side Handling (Scheduling Time)**

When a pod with valid SemVer operators encounters a node with invalid taint/label values:

- **Tolerations**: If a node taint value cannot be parsed as SemVer, the `compareSemVerValues` function returns `false`, meaning the toleration does not match:
  - For `NoSchedule`/`NoExecute` taints: Pod cannot schedule on that node
  - For `PreferNoSchedule` taints: Node receives lower score
  - Scheduler event: `0/N nodes are available: X node(s) had untolerated taint {key: invalid-value}`
  - Scheduler logs (Error level): `"failed to parse taint value as semantic version" taint="invalid-value"`

- **Node Affinity**: If a node label value cannot be parsed as SemVer, the affinity matching returns `false`:
  - For `requiredDuringSchedulingIgnoredDuringExecution`: Pod cannot schedule on that node
  - For `preferredDuringSchedulingIgnoredDuringExecution`: Node receives 0 score contribution for that term (pod can still schedule)
  - Scheduler logs (V(10) level): `"Parse semver failed for value X in label Y"`

The behavior is **fail-safe**: Invalid values cause matching to fail, preventing pods from scheduling on potentially incompatible nodes.

#### Controller Hot-Loop When Feature Gate is Disabled

**Risk**: If a workload controller (Deployment, StatefulSet, Job, etc.) has a pod template that uses `SemverLt` or `SemverGt` or `SemverEq` operators, and the feature gate is disabled or was disabled after being enabled, the controller will enter a hot-loop:

1. Controller attempts to create a pod from the template
2. API server validation rejects the pod with error: `Unsupported value: "SemverLt": supported values: "Equal", "Exists"`
3. Controller immediately retries pod creation and this cycle repeats indefinitely

This is particularly problematic during rollback/downgrade scenarios or for multi-cluster deployments where the feature gate state differs across clusters.

**Mitigation**:

- Before disabling the feature gate, cluster operators should identify all workloads using `SemverLt`/`SemverGt`/`SemverEq` operators via API discovery or scanning tools
- The Upgrade/downgrade documentation should explicitly warn about this scenario and provide steps to identify affected workloads
- The `apiserver_request_total` metric can be used to detect hot-loop conditions

## Design Details

### API Changes

**File**: `staging/src/k8s.io/api/core/v1/types.go`

Extend `core/v1.Toleration.Operator` to accept, in addition to `Equal` and `Exists`:

- `SemverLt`: match if version of toleration.value < version of taint.value
- `SemverGt`: match if version of toleration.value > version of taint.value
- `SemverEq`: match if version of toleration.value = version of taint.value
- `Equal`/`Exists`: Remain unchanged

```go
// A toleration operator is the set of operators that can be used in a toleration.
// +enum
type TolerationOperator string

const (
	TolerationOpExists TolerationOperator = "Exists"
	TolerationOpEqual  TolerationOperator = "Equal"

  // New semver comparison operators (feature-gated)
  TolerationOpSemverLt TolerationOperator = "SemverLt"    // Version Less than
  TolerationOpSemverGt TolerationOperator = "SemverGt"    // Version Greater than
  TolerationOpSemverEq TolerationOperator = "SemverEq"    // Version Equals to
)
```

**File**: `staging/src/k8s.io/api/core/v1/types.go`

Extend `core/v1.NodeSelectorRequirement.Operator` to accept, in addition to `In`, `NotIn`, `Exists`, `DoesNotExists`, `Gt` and `Lt`:

- `SemverLt`: match if version of value of NodeSelectorRequirement.Key < version of NodeSelectorRequirement.Values[0]
- `SemverGt`: match if version of value of NodeSelectorRequirement.Key > version of NodeSelectorRequirement.Values[0]
- `SemverEq`: match if version of value of NodeSelectorRequirement.Key = version of NodeSelectorRequirement.Values[0]
- `In`/`NotIn`/`Exists`/`DoesNotExists`/`Gt`/`Lt`: Remain unchanged

```go
// A node selector requirement is a selector that contains values, a key, and an operator
// that relates the key and values.
type NodeSelectorRequirement struct {
	// The label key that the selector applies to.
	Key string `json:"key" protobuf:"bytes,1,opt,name=key"`
	// Represents a key's relationship to a set of values.
	// Valid operators are In, NotIn, Exists, DoesNotExist. Gt, Lt, SemverGt, SemverLt, and SemverEq.
	Operator NodeSelectorOperator `json:"operator" protobuf:"bytes,2,opt,name=operator,casttype=NodeSelectorOperator"`
	// An array of string values. If the operator is In or NotIn,
	// the values array must be non-empty. If the operator is Exists or DoesNotExist,
	// the values array must be empty. If the operator is Gt or Lt, the values
	// array must have a single element, which will be interpreted as an integer.
  // If the operator is SemverGt, SemverLt, or SemverEq, the values array must have a single
  // element, which will be interpreted as a string.
	// This array is replaced during a strategic merge patch.
	// +optional
	// +listType=atomic
	Values []string `json:"values,omitempty" protobuf:"bytes,3,rep,name=values"`
}

// A node selector operator is the set of operators that can be used in
// a node selector requirement.
// +enum
type NodeSelectorOperator string

const (
	NodeSelectorOpIn           NodeSelectorOperator = "In"
	NodeSelectorOpNotIn        NodeSelectorOperator = "NotIn"
	NodeSelectorOpExists       NodeSelectorOperator = "Exists"
	NodeSelectorOpDoesNotExist NodeSelectorOperator = "DoesNotExist"
	NodeSelectorOpGt           NodeSelectorOperator = "Gt"
	NodeSelectorOpLt           NodeSelectorOperator = "Lt"

  // New semver comparison operators (feature-gated)
  NodeSelectorOpSemverGt  NodeSelectorOperator = "SemverGt"
  NodeSelectorOpSemverLt  NodeSelectorOperator = "SemverLt"
  NodeSelectorOpSemverEq  NodeSelectorOperator = "SemverEq"
)

```

### Semantics

1. Pod Node Affinity

- When `SemverGt`, `SemverLt`, or `SemverEq` are used as the `NodeSelectorOperator`, The values array must contain exactly one element. If the values array is empty or contains multiple strings, the Pod is rejected during validation.

- The single value is parsed as a Semantic Version. If the parsing fails, the requirement evaluates to false (does not match).

- For `preferredDuringSchedulingIgnoredDuringExecution` the `weight` is added to the score If the node label or node field satisfies the SemVer comparison. Otherwise 0 will be added to the score if a mismatch.

2. PersistentVolume Node Affinity

- When `SemverGt`, `SemverLt`, or `SemverEq` are used as the `NodeSelectorOperator` in a PersistentVolume's node affinity, the values array must contain exactly one element. If the values array is empty or contains multiple strings, the PersistentVolume is rejected during validation.

- The single value is parsed as a Semantic Version. If the parsing fails during volume attachment, the node does not satisfy the volume's node affinity requirements.

- PersistentVolume node affinity uses the same matching logic as Pod node affinity through shared `NodeSelectorTerm` implementation. This ensures consistent behavior between pod scheduling and volume attachment decisions.

3. Tolerations

- For Taints with the `PreferNoSchedule` effect, the SemVer operators determine whether the taint is tolerated during the scoring phase:

- **Tolerated taints**: Do not count against the node's score.
- **Intolerated taints**: Count against the node's score.
- **Scoring**: Unchanged - nodes with fewer intolerable `PreferNoSchedule` taints receive higher scores.

**Example:**

Node `A` has taint `version=v1.0.0:PreferNoSchedule`.
Node `B` has taint `version=v2.0.0:PreferNoSchedule`.
Pod has toleration operator: `SemverGt`, value: `v1.5.0`.

Result: The Pod tolerates Node B (2.0 > 1.5) but does not tolerate Node A. Therefore, Node B receives a higher score (no penalty) compared to Node A.

### Implementation

#### Feature Gate Definition

**File**: `pkg/features/kube_features.go`

```go
const (
    // TaintTolerationNodeAffinitySemverComparisonOperators enables semver comparison operators (SemverLt, SemverGt, SemverEq) for tolerations and node affinity
    TaintTolerationNodeAffinitySemverComparisonOperators featuregate.Feature = "TaintTolerationNodeAffinitySemverComparisonOperators"
)

var defaultKubernetesFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
    TaintTolerationNodeAffinitySemverComparisonOperators: {Default: false, PreRelease: featuregate.Alpha},
}
```

#### API Validation

**File**: `pkg/apis/core/validation/validation.go`

##### PodValidationOption modification

```go
// PodValidationOptions contains the different settings for pod validation
type PodValidationOptions struct {
	....
	// Allow semver node affinity and toleration comparison operators (SemverGt, SemverLt, SemverEq)
	AllowTaintTolerationNodeAffinitySemverComparisonOperators bool
```

##### Validate tolerant spec

```go
func ValidateTolerations(tolerations []core.Toleration, fldPath *field.Path, opts PodValidationOptions) field.ErrorList {
	allErrors := field.ErrorList{}
	for i, toleration := range tolerations {
		....
		case core.TolerationOpSemverEq, core.TolerationOpSemverGt, core.TolerationOpSemverLt:
			if !opts.AllowTaintTolerationNodeAffinitySemverComparisonOperators {
				validValues := []core.TolerationOperator{core.TolerationOpEqual, core.TolerationOpExists}
				allErrors = append(allErrors, field.NotSupported(idxPath.Child("operator"), toleration.Operator, validValues))
				break
			}
			// non-strictly validate semver version by using semver.ParseTolerant
			if _, err := semver.ParseTolerant(toleration.Value); err != nil {
				allErrors = append(allErrors, field.Invalid(idxPath.Child("value"), toleration.Value, err.Error()))
			}
		default:
			validValues := []core.TolerationOperator{core.TolerationOpEqual, core.TolerationOpExists}
			allErrors = append(allErrors, field.NotSupported(idxPath.Child("operator"), toleration.Operator, validValues))
		}
```

##### Validate NodeSelectorRequirement for nodeaffinity

```go
// ValidateNodeSelectorRequirement tests that the specified NodeSelectorRequirement fields has valid data
func ValidateNodeSelectorRequirement(rq core.NodeSelectorRequirement, allowInvalidLabelValueInRequiredNodeAffinity bool, fldPath *field.Path, opts PodValidationOptions) field.ErrorList {
	allErrs := field.ErrorList{}
	switch rq.Operator {
	....
	case core.NodeSelectorOpSemverEq, core.NodeSelectorOpSemverGt, core.NodeSelectorOpSemverLt:
		if !opts.AllowTaintTolerationNodeAffinitySemverComparisonOperators {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("operator"), rq.Operator, "not a valid selector operator"))
			break
		}
		if len(rq.Values) != 1 {
			allErrs = append(allErrs, field.Required(fldPath.Child("values"), "must be specified single value when `operator` is 'SemverLt' or 'SemverGt' or 'SemverEq'"))
		}
		// non-strictly validate semver version by using semver.ParseTolerant
		for _, val := range rq.Values {
			if _, err := semver.ParseTolerant(val); err != nil {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("values"), rq.Values, err.Error()))
			}
		}
	default:
		allErrs = append(allErrs, field.Invalid(fldPath.Child("operator"), rq.Operator, "not a valid selector operator"))
```

##### Validate the Volume Node affinity by passing the feature gate to volume options

```go
// PersistentVolumeSpecValidationOptions contains the different settings for PeristentVolume validation
type PersistentVolumeSpecValidationOptions struct {
	// Allow users to modify the class of volume attributes
	EnableVolumeAttributesClass bool
	// Allow invalid label-value in RequiredNodeSelector
	AllowInvalidLabelValueInRequiredNodeAffinity bool
	// Allow semver comparison operators
	AllowTaintTolerationNodeAffinitySemverComparisonOperators bool
}
```

```go
func validateVolumeNodeAffinity(nodeAffinity *core.VolumeNodeAffinity, opts PersistentVolumeSpecValidationOptions, fldPath *field.Path) (bool, field.ErrorList) {
	...
	if nodeAffinity.Required != nil {
		allErrs = append(allErrs, ValidateNodeSelector(nodeAffinity.Required, opts.AllowInvalidLabelValueInRequiredNodeAffinity, fldPath.Child("required"), PodValidationOptions{AllowTaintTolerationNodeAffinitySemverComparisonOperators: opts.AllowTaintTolerationNodeAffinitySemverComparisonOperators})...)
	} else {
		allErrs = append(allErrs, field.Required(fldPath.Child("required"), "must specify required node constraints"))
	}

	return true, allErrs
}
```

##### Backward Compatibility Validation

Backward compatibility logic makes sure to allow semver operators if they are already in use by tolerations or node affinity after the feature gate is being disabled.

**File**: `pkg/api/pod/util.go`

```go
func GetValidationOptionsFromPodSpecAndMeta(podSpec, oldPodSpec *api.PodSpec, podMeta, oldPodMeta *metav1.ObjectMeta) apivalidation.PodValidationOptions {
  ...
opts.AllowTaintTolerationNodeAffinitySemverComparisonOperators = allowTaintTolerationNodeAffinitySemverComparisonOperators(oldPodSpec)
```

```go
func tolerationNodeAffinitySemverComparisonOperatorsInUse(podSpec *api.PodSpec) bool {
	if podSpec == nil {
		return false
	}
	for _, toleration := range podSpec.Tolerations {
		if toleration.Operator == api.TolerationOpSemverEq || toleration.Operator == api.TolerationOpSemverGt || toleration.Operator == api.TolerationOpSemverLt {
			return true
		}
	}
	// check if the semver operators are in use by node affinity
	if podSpec.Affinity != nil && podSpec.Affinity.NodeAffinity != nil {
		for _, preferredAffinityTerm := range podSpec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
			for _, nodeSelectorReq := range preferredAffinityTerm.Preference.MatchExpressions {
				if nodeSelectorReq.Operator == api.NodeSelectorOpSemverEq || nodeSelectorReq.Operator == api.NodeSelectorOpSemverGt || nodeSelectorReq.Operator == api.NodeSelectorOpSemverLt {
					return true
				}
			}
		}
		if podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
			for _, reqAffinityTerm := range podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
				for _, nodeSelectorReq := range reqAffinityTerm.MatchExpressions {
					if nodeSelectorReq.Operator == api.NodeSelectorOpSemverEq || nodeSelectorReq.Operator == api.NodeSelectorOpSemverGt || nodeSelectorReq.Operator == api.NodeSelectorOpSemverLt {
						return true
					}
				}
			}
		}
	}
	return false
}

func allowTaintTolerationNodeAffinitySemverComparisonOperators(oldPodSpec *api.PodSpec) bool {
	// allow the operators if the feature gate is enabled or the old pod spec uses
	// comparison operators
	if utilfeature.DefaultFeatureGate.Enabled(features.TaintTolerationNodeAffinitySemverComparisonOperators) ||
		tolerationNodeAffinitySemverComparisonOperatorsInUse(oldPodSpec) {
		return true
	}
	return false
}
```

**File**: `pkg/apis/core/helper/helpers.go`

```go
// HasSemverComparisonOperator checks if there's a semver comparison operator
// being used in the NodeSelectorTerm MatchExpression opertor
func HasSemverComparisonOperator(terms []core.NodeSelectorTerm) bool {
	for _, term := range terms {
		for _, expression := range term.MatchExpressions {
			switch expression.Operator {
			case core.NodeSelectorOpSemverEq, core.NodeSelectorOpSemverGt, core.NodeSelectorOpSemverLt:
				return true
			}
		}
	}
	return false
}
```

```go
// for PersistentVolumeSpecValidationOptions
opts.AllowTaintTolerationNodeAffinitySemverComparisonOperators = helper.HasSemverComparisonOperator(terms)
```

#### Scheduler Logic

##### Toleration Logic

**File**: `staging/src/k8s.io/component-helpers/scheduling/corev1/helpers.go`

```go
func (t *Toleration) ToleratesTaint(logger klog.Logger, taint *Taint, enableComparisonOperators, enableSemverComparisonOperators bool) bool {
	....
	case TolerationOpLt, TolerationOpGt:
		// If comparison operators are disabled, this toleration doesn't match
		if !enableComparisonOperators {
			return false
		}
		return compareNumericValues(logger, t.Value, taint.Value, t.Operator)
	case TolerationOpSemverLt, TolerationOpSemverGt, TolerationOpSemverEq:
		// If Semver comparison operators are disabled, this toleration doesn't match
		if !enableSemverComparisonOperators {
			return false
		}
		return compareSemVerValues(logger, t.Value, taint.Value, t.Operator)
	default:
		return false
	}
}

// compareSemVerValues performs Semver comparison between toleration and taint values
func compareSemVerValues(logger klog.Logger, tolerationVal, taintVal string, op TolerationOperator) bool {

	tolerationVersion, err := semver.ParseTolerant(tolerationVal)
	if err != nil {
		logger.Error(err, "failed to parse tolartion value as semantic version", "toleration", tolerationVal)
		return false
	}

	taintVersion, err := semver.ParseTolerant(taintVal)
	if err != nil {
		logger.Error(err, "failed to parse taint value as semantic version", "taint", taintVal)
		return false
	}

	switch op {
	case TolerationOpSemverEq:
		return taintVersion.EQ(tolerationVersion)
	case TolerationOpSemverGt:
		return taintVersion.GT(tolerationVersion)
	case TolerationOpSemverLt:
		return taintVersion.LT(tolerationVersion)
	default:
		return false
	}
}
```

##### Node Affinity Logic for MatchExpressions

The match logic is the one the does the parsing of the semver if the selection.Operator is one of the three operators used for semver comparison, which are defined earlier.

```go
func (r *Requirement) Matches(ls Labels) bool {
	switch r.operator {
	...
	case selection.VersionEquals, selection.VersionGreaterThan, selection.VersionLessThan:
		if !utilfeature.DefaultFeatureGate.Enabled(features.TaintTolerationNodeAffinitySemverComparisonOperators) {
			return false
		}

		val, exists := ls.Lookup(r.key)
		if !exists {
			return false
		}

		lsVersion, err := semver.ParseTolerant(val)
		if err != nil {
			klog.V(10).Infof("Parse semver failed for value %+v in label %+v, %+v", val, ls, err)
			return false
		}

		// There should be only one strValue in r.strValues, and can be converted to a semver.
		if len(r.strValues) != 1 {
			klog.V(10).Infof("Invalid values count %+v of requirement %#v, for 'SemverGt', 'SemverLt', `SemverEq` operators, exactly one value is required", len(r.strValues), r)
			return false
		}

		var rVersion semver.Version
		for i := range r.strValues {
			rVersion, err = semver.ParseTolerant(r.strValues[i])
			if err != nil {
				klog.V(10).Infof("Parse semver failed for value %+v in requirement %#v, for 'SemverGt', 'SemverLt', `SemverEq` operators, the value must be a semver", r.strValues[i], r)
				return false
			}
		}

		return (r.operator == selection.VersionGreaterThan && lsVersion.GT(rVersion)) || (r.operator == selection.VersionLessThan && lsVersion.LT(rVersion)) || (r.operator == selection.VersionEquals && lsVersion.EQ(rVersion))
	default:
		return false
	}
}
```

The `nodeSelectorRequirementsAsSelector` defines the selection operators and receive the value of the feature gate which we pass all the way from the scheduler framework down to this function.

```go
// nodeSelectorRequirementsAsSelector converts the []NodeSelectorRequirement api type into a struct that implements
// labels.Selector.
func nodeSelectorRequirementsAsSelector(nsm []v1.NodeSelectorRequirement, path *field.Path, enableSemverComparisonOperators bool) (labels.Selector, []error) {
	if len(nsm) == 0 {
		return labels.Nothing(), nil
	}
	var errs []error
	selector := labels.NewSelector()
	for i, expr := range nsm {
...
		case v1.NodeSelectorOpSemverEq:
			if !enableSemverComparisonOperators {
				errs = append(errs, field.NotSupported(p.Child("operator"), expr.Operator, validSelectorOperators))
				continue
			}
			op = selection.VersionEquals
		case v1.NodeSelectorOpSemverLt:
			if !enableSemverComparisonOperators {
				errs = append(errs, field.NotSupported(p.Child("operator"), expr.Operator, validSelectorOperators))
				continue
			}
			op = selection.VersionLessThan
		case v1.NodeSelectorOpSemverGt:
			if !enableSemverComparisonOperators {
				errs = append(errs, field.NotSupported(p.Child("operator"), expr.Operator, validSelectorOperators))
				continue
			}
			op = selection.VersionGreaterThan
```

##### Persistent Volumes NodeAffinity

The logic of the PV node affinity is shared by the same NodeSelector functions, and is being called by the `VolumeBinding` plugin:

```
pkg/scheduler/framework/plugins/volumebinding/
    - volume_binding.go:424  - Filter() entry point
    - binder.go:287         - FindPodVolumes()
    - binder.go:842         - checkBoundClaims()
    - binder.go:878         - findMatchingVolumes()

  staging/src/k8s.io/component-helpers/storage/volume/
    - helpers.go:68         - CheckNodeAffinity()
    - pv_helpers.go:186     - FindMatchingVolume()

  staging/src/k8s.io/component-helpers/scheduling/corev1/
    - helpers.go:40         - MatchNodeSelectorTerms()
```

##### Additional changes

Since the `dynamicresources.go` is also calling `NewLazyErrorNodeSelector()`:

```
From DynamicResources Plugin (dynamicresources.go:446):
    └─> NewNodeSelector()
        └─> NewLazyErrorNodeSelector()
            └─> newNodeSelectorTerm()
                └─> nodeSelectorRequirementsAsSelector()
```

Then the implementation will have to add the featuregate option back to the `DynamicResources` plugin, however it wont be utilized since the feature only targets NodeAffinity.

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

The following unit test files include coverage for semver comparison operators:

1. **pkg/api/pod/util_test.go**
   - `TestAllowTaintTolerationNodeAffinitySemverComparisonOperators`: Tests feature gate enabled/disabled scenarios and backward compatibility for pods using semver operators in tolerations and node affinity

2. **pkg/apis/core/validation/validation_test.go**
   - Tests for validating semver operators in pod specifications
   - Tests for validating semver operators in PersistentVolume specifications
   - Tests for PersistentVolume validation backward compatibility

3. **staging/src/k8s.io/api/core/v1/toleration_test.go**
   - Tests for `compareSemVerValues` function with various version formats
   - Tests for toleration matching with semver operators

4. **staging/src/k8s.io/component-helpers/scheduling/corev1/nodeaffinity/nodeaffinity_test.go**
   - Tests for node affinity matching with semver operators
   - Tests for version comparison edge cases

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

- Feature implemented behind `TaintTolerationNodeAffinitySemverComparisonOperators` feature gate (disabled by default)
- API validation for version operators in place
- Taint/toleration matching logic supports `SemverLt`, `SemverGt`, `SemverEq` operators  
- Node affinity matching logic supports `SemverLt`, `SemverGt`, `SemverEq` operators  

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
  Enable the feature gate in kube-apiserver first then kube-scheduler. This ensures the API server can accept and validate pods with the new operators before the kube-scheduler tries to process them.

#### Downgrade
  Disable the feature gate in in kube-scheduler then kube-apiserver. Since we want to stop the kube-scheduler from processing the new operators first, then stop the API server from accepting new pods with those operators. This prevents the scheduler from trying to handle features the API server would reject.
  
**What happens when the scheduler doesn't recognize SemverGt/SemverLt/SemverEq operators for tolerations:**

When the feature gate is disabled and the scheduler encounters a pod with `SemverGt`/`SemverLt`/`SemverEq` operator:

- The toleration filter returns `false` (doesn't match)
- Pod is considered to have untolerated taints
- Filter returns `UnschedulableAndUnresolvable` status
- Pod remains in Pending state.
   - Feature gate on/off test cases

**What happens when the scheduler doesn't recognize SemverGt/SemverLt/SemverEq operators for nodeAffinity:**

When the feature gate is disabled and the scheduler encounters a pod with `SemverGt`/`SemverLt`/`SemverEq` operator:
- The affinity match function returns `false` (doesn't match)
- In case of `requiredDuringSchedulingIgnoredDuringExecution` the pod will remain in pending state.
- In case of `preferredDuringSchedulingIgnoredDuringExecution` The pod will successfully schedule . However, the specific term containing the SemVer operator will evaluate to false and contribute 0 points to the node's score.

### Version Skew Strategy

The skew between kubelet and control-plane components are not impacted. The kube-scheduler is expected to match the kube-apiserver minor version, but may be up to one minor version older (to allow live upgrades).

In the release it's been added, the feature is disabled by default and not recognized by other components.

Whoever enabled the feature manually would take the risk of component like kube-scheduler being old and not recognize the fields.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `TaintTolerationNodeAffinitySemverComparisonOperators`
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-scheduler

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, but with caveats for existing workloads using the new operators.

Impact on existing pods with `SemverGt`/`SemverLt`/`SemverEq` operators when feature is disabled:

1. **Already-running pods**: Continue running normally. The kubelet doesn't need to re-evaluate tolerations for running pods.

2. **Unscheduled/pending pods**: 
   - Remain in the cluster but cannot be scheduled
   - In case of Tolerations:
   - The scheduler's TaintToleration or NodeAffinity plugin won't recognize `SemverGt`/`SemverLt`/`SemverEq` operators and will treat them as non-matching
   - These pods will remain in Pending state with events indicating untolerated taints

3. **New pod creation**: 
   - API server validation will **reject** new pods with Gt/Lt operators
   - Error in case of tolerations: `spec.tolerations[].operator: Unsupported value: "SemverGt": supported values: "Equal", "Exists"`
   - Error in case of affinity: `spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].operator: Invalid value: "SemverEq": not a valid selector operator`

4. **Pod updates**:
   - Cannot update existing pods (even those already in etcd) if they contain `SemverGt`/`SemverLt`/`SemverEq` operators
   - Validation runs on update and will reject the unsupported operators

###### What happens if we reenable the feature if it was previously rolled back?

Extended toleration operators will be respected again:
- Existing pods with `SemverGt`/`SemverLt`/`SemverEq` operators in etcd become valid and schedulable
- New pods can be created with `SemverGt`/`SemverLt`/`SemverEq` operators
- The scheduler will properly evaluate version comparisons

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

**Rollout**: The feature enablement itself is safe and shouldn't impact existing workloads. It's an opt-in feature that only affects pods explicitly using `SemverGt`/`SemverLt`/`SemverEq` operators.

**Rollback**: Can impact workloads if not done carefully:

1. Running pods with `SemverGt`/`SemverLt`/`SemverEq` operators: continue running (safe)
2. Pending pods with `SemverGt`/`SemverLt`/`SemverEq` operators: become stuck in Pending state, as:
   - They remain in etcd but validation rejects them
   - The scheduler won't recognize the operators
   - Force deletion may be required: `kubectl delete pod <name> --force --grace-period=0`
3. Workload controllers (Deployments, StatefulSets, etc.):
   - If the pod template uses `SemverGt`/`SemverLt`/`SemverEq` operators, the controller cannot create new pods
   - Rolling updates will fail
 
  **Recommended rollback procedure to prevent hot loop**:
  1. Update identified workloads to remove semantic version operators.
  2. Delete pending pods that use `SemverGt`/`SemverLt`/`SemverEq` operators
  3. Disable feature gate in kube-scheduler first, then kube-apiserver


###### What specific metrics should inform a rollback?

- `scheduler_scheduling_duration_seconds`: Increased scheduling latency may indicate performance issues with version parsing
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
   # Check for pods with numeric toleration operators
   kubectl get pods -A -o jsonpath='{range .items[*]}{.metadata.name}{": "}{.spec.tolerations[?(@.operator=="SemverGt")]}{"\n"}{end}' | grep -v "^[^:]*: *$"
   kubectl get pods -A -o jsonpath='{range .items[*]}{.metadata.name}{": "}{.spec.tolerations[?(@.operator=="SemverLt")]}{"\n"}{end}' | grep -v "^[^:]*: *$"
   kubectl get pods -A -o jsonpath='{range .items[*]}{.metadata.name}{": "}{.spec.tolerations[?(@.operator=="SemverEq")]}{"\n"}{end}' | grep -v "^[^:]*: *$"
   kubectl get pods -A -o jsonpath='{range .items[*]}{.metadata.name}{": "}{.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[*].matchExpressions[?(@.operator=="SemverGt")]}{"\n"}{end}' | grep -v "^[^:]*: *$"
   kubectl get pods -A -o jsonpath='{range .items[*]}{.metadata.name}{": "}{.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[*].matchExpressions[?(@.operator=="SemverEq")]}{"\n"}{end}' | grep -v "^[^:]*: *$"
   kubectl get pods -A -o jsonpath='{range .items[*]}{.metadata.name}{": "}{.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[*].matchExpressions[?(@.operator=="SemverLt")]}{"\n"}{end}' | grep -v "^[^:]*: *$"
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
  - Event Message: "node(s) had untolerated taint {<taint-key>: <taint-value>}" (e.g., with version taint)
- [x] API .spec.tolerations
  - Observe tolerations with `operator: SemverGt` or `operator: SemverLt` or `operator: SemverEq` on pods

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

Yes, an extension to an existing metric:

**Extend `scheduler_plugin_evaluation_total` with a `status` label**

Currently, `scheduler_plugin_evaluation_total` tracks plugin evaluation counts with labels: `plugin`, `extension_point`, `profile`. We propose adding a `status` label (similar to `scheduler_plugin_execution_duration_seconds`) to enable monitoring of plugin outcomes, including errors.

The status label will use framework status codes: `Success`, `Unschedulable`, `UnschedulableAndUnresolvable`, `Error`, etc.

 >Note: Currently, integer parsing failures for Gt/Lt operators result in the toleration not matching (returning `Unschedulable` status), similar to how label selectors behave. This means parsing errors are not distinguished from legitimate mismatches in metrics. Future enhancements could modify the implementation to return `Error` status for parsing failures to improve debuggability.

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

Potentially yes, but the impact should be **minimal**. The semver toleration operators as well as node affinity operators could slightly increase time for operations covered by existing SLIs/SLOs due to semver parsing overhead and validation overhead.

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
