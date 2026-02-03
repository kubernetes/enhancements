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
  - [1. API Changes](#1-api-changes)
  - [2. Feature Gate](#2-feature-gate)
  - [3. API Validation](#3-api-validation)
  - [4. Scheduler Logic](#4-scheduler-logic)
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
- [x] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This enhancement introduces Semantic Versioning (SemVer) comparison to Taints and Tolerations and NodeAffinity. Similar to [KEP-5471](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5471-enable-sla-based-scheduling), this KEP adds `SemverLt`, `SemverGt`, and `SemverEq` operators to **core/v1 Toleration**, **core/v1 NodeAffinity**, **core/v1 PersistentVolume NodeAffinity**. This enables granular control over workload placement, volume attachment based on versioned node attributes (e.g., Kubelet version, kernel version, GPU driver versions) without requiring manual enumeration of all target versions.

## Motivation

Many scheduling decisions and storage attachment decisions depend on component versions (e.g., kernel versions, or driver versions). However, the current scheduling framework restricts NodeSelector and Tolerations to set-based operators (In, NotIn, Exists, DoesNotExist). Similarly, PersistentVolume node affinity is also restricted to the same set-based operators.

This restriction forces users to treat ordered versions as unrelated strings. To target a node with a version "greater than X" users will want concrete semantic versioning comparisons. This applies to both pod scheduling decisions and persistent volume attachment decisions where volumes may require specific node capabilities that are version-dependent (e.g. kernel features, storage driver versions).

### Goals

- Add semantic version based comparison capabilities to tolerations so pods can match taints that contain Semver based values.
- Add semantic version based comparison capabilities to node affinity so that scheduling and volume attachment decisions can be based on version comparisons. This applies to Pod NodeAffinity and PersistentVolume NodeAffinity.

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
          - key: "node.example/kubelet-version"
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
          - key: "node.example/container-runtime-version"
            operator: "SemverGt"
            values: ["2.0.0"]
```

#### Story 4 — Persistent Volume node affinity based on kernel version

As a storage administrator, I am managing persistent volumes that require specific kernel features available only in newer kernel versions. For example, a volume using advanced filesystem features needs a kernel version greater than 5.10.0. I want to ensure the persistent volume can only be attached to nodes running compatible kernel versions.

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
        - key: "node.example/kernel-version"
          operator: "SemverGt"
          values: ["5.10.0"]
  persistentVolumeReclaimPolicy: Retain
  storageClassName: advanced-storage
```

### Notes/Constraints/Caveats (Optional)

- **SemVer-Only Support**: The implementation will only support versioning comparisons based on SemVer specifications; other versioning schemes will be rejected by the API server during validation.

- **SemVer Library Version**: The current implementation will use `github.com/blang/semver/v4` library to parse and compare Tolerations and Affinity versions, the same library is used in the current code base.

- **Node Affinity Single Element Requirement**: The implementation will validate that if the semantic version operators are used for node affinity, then NodeSelector values array must contain only one element, similar to the behavior of `Gt` and `Lt`.

- **Toleration Parsing Requirements**: The toleration value must be parseable as Semantic Versions for SemVer operators (`SemverLt`, `SemverGt`, `SemverEq`). If parsing fails, the toleration does not match.

- **Node Affinity Parsing Requirements**: The node affinity value must be parseable as Semantic Versions for SemVer operators (`SemverLt`, `SemverGt`, `SemverEq`). If parsing fails, the node affinity matching rule does not match.

- **NodeFieldSelector Limitation**: The semver comparison operators (`SemverLt`, `SemverGt`, `SemverEq`) are only supported in `matchExpressions` within NodeSelectorTerms. They are **not** supported in `matchFields` as field selectors have different validation requirements.

- **Semver Tolerant Parsing**: The implementation will use `semver.ParseTolerant` to parse semver values which currently trims spaces, removes a "v" prefix, adds a 0 patch number to versions with only major and minor components specified, and removes leading 0s.

- **Non-Version Taint Values**: When a pod toleration uses `SemverLt`, `SemverGt`, or `SemverEq` operators, it only matches taints with SemVer values. If a node has a taint with a non-SemVer value, the toleration will not match, and the pod cannot schedule on that node.
  
- **Non-Version Affinity Values**: When a pod node affinity uses `SemverLt`, `SemverGt`, or `SemverEq` operators, the label value in the case of `MatchExpression` must be a parsable semver version; otherwise, the pod will not match scheduling requirements on these nodes.

- **Alpha Restrictions**: When `TolerationAffinitySemverOperators=false`, the API server rejects pods using the new operators in tolerations matching, and in node affinity matching.

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

### 1. API Changes

New semver-based operators will be added for both affinity and toleration:

**Toleration Operators:** Extend `core/v1.Toleration.Operator` with the following operators:

- `SemverLt`: match if version of taint.value < version of toleration.value
- `SemverGt`: match if version of taint.value > version of toleration.value
- `SemverEq`: match if version of taint.value = version of toleration.value

```go
  TolerationOpSemverLt TolerationOperator = "SemverLt"
  TolerationOpSemverGt TolerationOperator = "SemverGt"
  TolerationOpSemverEq TolerationOperator = "SemverEq"
```

**NodeSelector Operators**: Extend `core/v1.NodeSelectorRequirement.Operator` with the following operators: 

- `SemverLt`: match if version of value of NodeSelectorRequirement.Key < version of NodeSelectorRequirement.Values[0]
- `SemverGt`: match if version of value of NodeSelectorRequirement.Key > version of NodeSelectorRequirement.Values[0]
- `SemverEq`: match if version of value of NodeSelectorRequirement.Key = version of NodeSelectorRequirement.Values[0]

```go
  NodeSelectorOpSemverGt  NodeSelectorOperator = "SemverGt"
  NodeSelectorOpSemverLt  NodeSelectorOperator = "SemverLt"
  NodeSelectorOpSemverEq  NodeSelectorOperator = "SemverEq"
```

### 2. Feature Gate

A new feature gate `TolerationAffinitySemverOperators` will be added to the list of feature gates to allow API validation on Semver operators and to execute semver comparison in the scheduler logic.

### 3. API Validation

1. **Toleration Validation**:
  - A new validation case is added to toleration validation when `SemverGt`, `SemverLt`, or `SemverEq` is used while the feature gate `TolerationAffinitySemverOperators` is enabled.
  - The validation uses `semver.ParseTolerant` to non-strictly parse the version value; if parsing fails, a validation error is appended.
  - If a semver operator is used when the feature gate is disabled, the operator is reported as unsupported.

2. **Pod NodeAffinity Validation**:
  - A new validation case is added to NodeAffinity when `SemverGt`, `SemverLt`, or `SemverEq` is used while the feature gate `TolerationAffinitySemverOperators` is enabled.
  - If the values are empty or more than one value is specified in `NodeSelectorRequirement`, validation fails.
  - If exactly one value is specified, the validation uses `semver.ParseTolerant` to non-strictly parse the version value; if parsing fails, a validation error is appended.

3. **PersistentVolume NodeAffinity Validation**:
  - The feature gate `TolerationAffinitySemverOperators` value is passed to the PersistentVolume NodeAffinity validation function.
  - The validation reuses the same Pod NodeAffinity validation logic.

4. **Backward Compatibility Validation**:
  - Backward compatibility logic ensures that semver operators are allowed if they are already in use by an existing object, even after the feature gate is disabled.
  - **Pod Backward Compatibility**
    - During pod validation, the old pod spec is checked to determine whether semver operators should be allowed.
    - Semver operators are allowed if the feature gate is enabled OR if the old pod spec already uses semver operators in tolerations or node affinity.
    - The check inspects tolerations, preferred node affinity terms, and required node affinity terms for `SemverEq`, `SemverGt`, or `SemverLt` operators in `MatchExpressions`.
  - **PersistentVolume Backward Compatibility**:
    - During PersistentVolume validation, the old PV's node affinity is checked for semver operators.
    - The validation iterates through `NodeSelectorTerms` and checks if any `MatchExpressions` use semver operators.

### 4. Scheduler Logic

1. **Toleration Matching**:
  - When a pod toleration uses `SemverGt`, `SemverLt`, or `SemverEq` operators, the toleration matching logic performs semver comparison between the toleration value and the taint value.
  - If the feature gate is disabled, the toleration does not match.
  - Both the toleration value and taint value are parsed using `semver.ParseTolerant`. If either parsing fails, the toleration does not match and an error is logged.
  - The comparison is performed as follows:
    - `SemverEq`: matches if the taint version equals the toleration version.
    - `SemverGt`: matches if the taint version is greater than the toleration version.
    - `SemverLt`: matches if the taint version is less than the toleration version.
  - For `PreferNoSchedule` taints, tolerated taints do not count against the node's score, while untolerated taints reduce the node's score.

2. **Pod Node Affinity Matching**:
  - When `SemverGt`, `SemverLt`, or `SemverEq` are used in `MatchExpressions`, the matching logic performs semver comparison between the node label value and the requirement value.
  - The values array must contain exactly one element; otherwise, matching returns false.
  - Both the node label value and the requirement value are parsed using `semver.ParseTolerant`. If either parsing fails, the match returns false.
  - The comparison is performed as follows:
    - `SemverEq`: matches if the node label version equals the requirement version.
    - `SemverGt`: matches if the node label version is greater than the requirement version.
    - `SemverLt`: matches if the node label version is less than the requirement version.
  - For `requiredDuringSchedulingIgnoredDuringExecution`, the pod cannot schedule on nodes that do not satisfy the requirement.
  - For `preferredDuringSchedulingIgnoredDuringExecution`, the weight is added to the node's score if the requirement is satisfied; otherwise, 0 is added.

3. **PersistentVolume Node Affinity Matching**:
  - PersistentVolume node affinity uses the same matching logic as Pod node affinity through shared `NodeSelectorTerm` implementation.
  - The `VolumeBinding` scheduler plugin calls the node affinity matching logic when filtering nodes for volume attachment.
  - If a node label value cannot be parsed as a valid semver, the node does not satisfy the volume's node affinity requirements.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

###### Coverage of existing packages

- `k8s.io/kubernetes/pkg/api/pod`: `2026-02-03` - `80.7`
- `k8s.io/kubernetes/pkg/apis/core/validation`: `2026-02-03` - `85.3`
- `staging/src/k8s.io/api/core/v1/toleration_test.go`: `2026-02-03` - `75.0`
- `staging/src/k8s.io/component-helpers/scheduling/corev1/nodeaffinity`: `2026-02-03` - `95.4`

##### Integration tests

The following integration tests will be added or extended to cover semver comparison operators:

- `test/integration/scheduler/filters/filters_test.go`:
  - Extend `TestTaintTolerationFilter` to include test cases for `SemverGt`, `SemverLt`, and `SemverEq` operators with feature gate enabled/disabled scenarios
  - Extend `TestNodeAffinityFilter` to include test cases for semver operators in `MatchExpressions`

- `test/integration/scheduler/scoring/priorities_test.go`:
  - Extend `TestTaintTolerationScoring` to verify scoring behavior with `PreferNoSchedule` taints and semver tolerations
  - Extend `TestNodeAffinityScoring` to verify scoring behavior with `preferredDuringSchedulingIgnoredDuringExecution` using semver operators

- `test/integration/scheduler/scheduler_test.go`:
  - Add `TestTaintTolerationSemverIntegration` to test end-to-end scheduling with semver tolerations, including dynamic taint changes and pod rescheduling scenarios

- `test/integration/scheduler/volumebinding/volume_binding_test.go`:
  - Extend volume binding tests to verify PersistentVolume node affinity matching with semver operators

##### e2e tests

The following e2e tests will be added or extended to cover semver comparison operators:

- `test/e2e/scheduling/taints.go`:
  - Add test cases for pod scheduling with semver toleration operators against node taints

- `test/e2e/scheduling/predicates.go`:
  - Add test cases for pod scheduling with semver node affinity operators

- `test/e2e/storage/persistent_volumes.go`:
  - Add test cases for PersistentVolume attachment with semver node affinity constraints

### Graduation Criteria

#### Alpha

- Feature implemented behind `TolerationAffinitySemverOperators` feature gate (disabled by default)
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
  Disable the feature gate in kube-scheduler then kube-apiserver. Since we want to stop the kube-scheduler from processing the new operators first, then stop the API server from accepting new pods with those operators. This prevents the scheduler from trying to handle features the API server would reject.
  
**What happens when the scheduler doesn't recognize SemverGt/SemverLt/SemverEq operators for tolerations:**

When the feature gate is disabled and the scheduler encounters a pod with `SemverGt`/`SemverLt`/`SemverEq` operator:

- The toleration filter returns `false` (doesn't match)
- Pod is considered to have untolerated taints
- Filter returns `UnschedulableAndUnresolvable` status
- Pod remains in Pending state.

**What happens when the scheduler doesn't recognize SemverGt/SemverLt/SemverEq operators for nodeAffinity:**

When the feature gate is disabled and the scheduler encounters a pod with `SemverGt`/`SemverLt`/`SemverEq` operator:
- The affinity match function returns `false` (doesn't match)
- In case of `requiredDuringSchedulingIgnoredDuringExecution` the pod will remain in pending state.
- In case of `preferredDuringSchedulingIgnoredDuringExecution` the pod will successfully schedule. However, the specific term containing the SemVer operator will evaluate to false and contribute 0 points to the node's score.

### Version Skew Strategy

The skew between kubelet and control-plane components is not impacted. The kube-scheduler is expected to match the kube-apiserver minor version, but may be up to one minor version older (to allow live upgrades).

In the release where it is added, the feature is disabled by default and not recognized by other components.

Whoever enables the feature manually takes the risk of components like kube-scheduler being old and not recognizing the fields.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `TolerationAffinitySemverOperators`
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
   - API server validation will **reject** new pods with `SemverGt`, `SemverLt`, or `SemverEq` operators for both Tolerations and Affinity rules.
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

Yes, the following unit tests will be added to cover feature gate enablement/disablement scenarios:

- `pkg/api/pod/util_test.go`:
  - A test will be added to verify backward compatibility when the feature gate is disabled but existing pods already use semver operators. This will cover scenarios for tolerations and node affinity with feature gate enabled/disabled.

- `pkg/apis/core/validation/validation_test.go`:
  - A test will be added to verify validation of tolerations with `SemverEq`, `SemverGt`, and `SemverLt` operators when the feature gate is enabled vs disabled.
  - A test will be added to verify validation of node affinity `NodeSelectorRequirement` with semver operators when the feature gate is enabled vs disabled.

- `staging/src/k8s.io/component-helpers/scheduling/corev1/nodeaffinity/nodeaffinity_test.go`:
  - A test will be added to verify node selector matching with semver operators when the feature gate is enabled vs disabled.
  - A test will be added to verify preferred scheduling terms with semver operators when the feature gate is enabled vs disabled.

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

Not yet. This will be tested manually before the feature graduates to Beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The operator can observe running workloads by querying the API and making sure that semver operators are used, for example:

   ```bash
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

We should continue to maintain the existing scheduler SLOs.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name:
    - `scheduler_framework_extension_point_duration_seconds`
    - `scheduler_plugin_evaluation_total`
    - Components exposing the metric: `kube-scheduler`

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No new metrics will be added to this feature.

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

The feature does not change the behavior when the API or etcd is unavailable, the scheduler will simply not be able to communicate with the API and schedule new pods, which should be the existing behavior regardless the feature.

###### What are other known failure modes?

- [Invalid semver value on node taint/label]
  - Detection: Pods remain in Pending state with `FailedScheduling` events indicating untolerated taints or unmatched node affinity.
  - Mitigations: Fix the node taint/label to contain a valid semver value, or update the pod toleration/affinity to use a different operator.
  - Diagnostics: Scheduler logs at Error level will show `"failed to parse taint value as semantic version"` or at V(10) level `"Parse semver failed for value X in label Y"`.
  - Testing: Unit tests cover invalid semver parsing scenarios.

- [Controller hot-loop when feature gate is disabled]
  - Detection: Spike in `apiserver_request_total` metric with validation errors for pod creation.
  - Mitigations: Update workload controllers to remove semver operators from pod templates before disabling the feature gate.
  - Diagnostics: API server logs will show repeated validation errors for unsupported operator values.
  - Testing: Unit tests cover feature gate disabled scenarios.

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
