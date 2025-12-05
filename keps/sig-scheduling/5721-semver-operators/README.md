# KEP-5721: Semantic Version Comparison Operators to Tolerations and NodeAffinity

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1 — Cluster operator ensures Kubelet version before scheduling pod](#story-1--cluster-operator-ensures-kubelet-version-before-scheduling-pod)
    - [Story 2 — Tolerate Running workloads on nodes with older versions of CNI](#story-2--tolerate-running-workloads-on-nodes-with-older-versions-of-cni)
    - [Story 3 — Container Runtime version based scheduling for sensitive pods](#story-3--Container-Runtime-version-based-scheduling-for-sensitive-pods)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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

This enhancement introduces Semantic Versioning (SemVer) comparison capabilities to Kubernetes scheduling. Similar to how KEP-5471 introduces integer operators (`Lt`, `Gt`) for Tolerations, this KEP adds `SemverLt`, `SemverGt`, and `SemverEq` operators to both **core/v1 Toleration** and **core/v1 NodeAffinity**. 

## Motivation

Many scheduling decsisions depends on components versions (eg. kernel versions, or driver versions). However, the current scheduling framework restricts NodeSelector and Tolerations to set-based operators (In, NotIn, Exists, DoesNotExist).

This restriction forces users to treat ordered versions as unrelated strings. To target a node with a version "greater than X," users will want a concrete semantic versioning comparisons.

### Goals

- Add semantic version based operators to tolerations so pods can match taints like `node.kubernetes.io/kubelet-version=v1.28.0` using `SemverGt`, `SemverLt`, or `SemverEq` operators.
- Add semantic version based operators `SemverGt`, `SemverLt`, and `SemverEq` to node affinity selectors so that scheduling decisions can be based on version comparisons.
- Backward compatible and opt‑in via a feature gate.
- Zero operational performance impact on existing pod scheduling using `Equal` and `Exists` operators.

### Non-Goals

- Changing NodeAffinity behavior.
- Changing Toleration behavior.

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
        - matchFields:
          - key: "status.nodeInfo.kubeletVersion"
            operator: SemverGt
            values: ["v1.31.99"]
```
#### Story 2 — Tolerate Running workloads on nodes with older versions of CNI

As a cluster operator, I am upgrading Calico CNI to version v3.28.0, during a phased upgrade of all nodes, I tainted nodes that are running an older CNI version, I run some workloads that can tolerate running on an older CNI, I will use semver operators to make sure that the old CNI node is tolerated by this workload.

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

As a cluster operator, I want to deploy critical workload that utilizes Linux Usernamespaces, however I still have nodes in the cluster that runs containerd version v1.6 that does not support Linux usernamespaces, I want to make sure that the workload will only be deployed on containerd version greater than v2.0.0

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
            values: "2.0.0"
```

### Notes/Constraints/Caveats (Optional)

- **SemVer-Only Support**: The implementation will only support versioining comparisons based on SemVer specifications, other versioining schemes will be rejected by the API server during validation.

- **Node Affinity Single Element Requirement**: The implementation will validate that if the semantic version operators are used for node affinity, then NodeSelector values array must contain only one element, similar to the behavior of `Gt` and `Lt`.

- **Toleration Parsing Requirements**: The toleration value must be parseable as Semantic Versions for SemVer operators (`SemverLt`, `SemverGt`, `SemverEq`). If parsing fails, the toleration does not match.

- **Node Affinity Parsing Requirements**: The node affinity value must be parseable as Semantic Versions for SemVer operators (`SemverLt`, `SemverGt`, `SemverEq`). If parsing fails, node affinity matching rule does not match

- **Non-Version Taint Values**: When a pod toleration uses `SemverLt`, `SemverGt`, or `SemverEq` operators, it only matches taints with SemVer values. If a node has a taint with a non-SemVer value, the toleration will not match, and the pod cannot schedule on that node.
  
  **Example**: 
  - Node taint: `node.kubernetes.io/containerRuntimeVersion=containerd://2.1.4:NoSchedule`
  - Pod toleration: `{key: "node.kubernetes.io/containerRuntimeVersion", operator: "SemverLt", value: "2.2.0"}`
  - **Result**: Toleration does not match and pod cannot schedule on this node because the container runtime version has a `containerd://` prefix
  - The pod remains `Pending` and can schedule on other nodes with valid version taints
  - The pod is not failed or rejected entirely

- **Non-Version Affinity Values**: When a pod node affinity uses `SemverLt`, `SemverGt`, or `SemverEq` operators, either Label value in case of `MatchExpression` or Node field value in case of `MatchFields` must be semver parsable, otherwise the pod will not match scheduling requirements on these nodes.

- **Alpha Restrictions**: When `AffinityTaintTolerationSemverComparisonOperatorss=false`, the API server rejects pods using the new operators in tolerations matching, and in node affinity matching.

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

**Risk**: Node labels or taints are currently free-form strings and are not validated for SemVer compliance at registration time (e.g. a node may carry a taint like this `node.kubernetes.io/containerRuntimeVersion=containerd://2.1.4` instead of `node.kubernetes.io/containerRuntimeVersion=2.1.4`). Since taint values are not validated at node registration time, these misconfigurations are only detected during scheduling when a pod with `SemverLt`/`SemverGt`/`SemverEq` tolerations attempts to match. This can lead to pods remaining in `Pending` state without clear indication of the root cause.

**Mitigation**:

- Pod validation: Current validation strictly enforces that only `Equal` and `Exists` operators are allowed. Users with version taint values today must explicitly change the operator to `SemverLt` or `SemverGt`, at which point pod-side validation will catch non-version toleration values and reject the pod spec before scheduling.

#### Controller Hot-Loop When Feature Gate is Disabled

**Risk**: If a workload controller (Deployment, StatefulSet, Job, etc.) has a pod template that uses `SemverLt` or `SemverGt` or `SemverEq` operators, and the feature gate is disabled or was disabled after being enabled, the controller will enter a hot-loop:

1. Controller attempts to create a pod from the template
2. API server validation rejects the pod with error: `Unsupported value: "SemverLt": supported values: "Equal", "Exists"`
3. Controller immediately retries pod creation and this cycle repeats indefinitely

This is particularly problematic during rollback/downgrade scenarios or for multi-cluster deployments where the feature gate state differs across clusters.

**Mitigation**:

- Before disabling the feature gate, cluster operators should identify all workloads using `SemverLt`/`SemverGt`/`SemverEq` operators via API discovery or scanning tools
- The Upgrade/downgrade documentation should explicitly warns about this scenario and provides steps to identify affected workloads
- The `apiserver_request_total` metric can be used to detect hot-loop conditions

## Design Details

### API Changes

**File**: `staging/src/k8s.io/api/core/v1/types.go`

Extend `core/v1.Toleration.Operator` to accept, in addition to `Equal` and `Exists`:

- `SemverLt`: match if version of toleration.value < version of taint.value
- `SemverGt`: match if version of toleration.value > version taint.value
- `SemverEq`: match if version of toleration.value = version taint.value
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

1. Node Affinity

- When `SemverGt`, `SemverLt`, or `SemverEq` are used as the `NodeSelectorOperator`, The values array must contain exactly one element. If the values array is empty or contains multiple strings, the Pod is rejected during validation.

- The single value is parsed as a Semantic Version. If the parsing fails, the requirement evaluates to false (does not match).

- For `preferredDuringSchedulingIgnoredDuringExecution` the `weight` is added to the score If the node label or node field satisfies the SemVer comparison. Otherwise 0 will be added to the score if a mismatch.

2. Tolerations

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
    // AffinityTaintTolerationSemverComparisonOperatorss enables semver comparison operators (SemverLt, SemverGt, SemverEq) for tolerations and node affinity
    AffinityTaintTolerationSemverComparisonOperatorss featuregate.Feature = "AffinityTaintTolerationSemverComparisonOperatorss"
)

var defaultKubernetesFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
    AffinityTaintTolerationSemverComparisonOperatorss: {Default: false, PreRelease: featuregate.Alpha},
}
```

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

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

### Graduation Criteria

#### Alpha

- Feature implemented behind `AffinityTaintTolerationSemverComparisonOperatorss` feature gate (disabled by default)
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
  
**What happens when the scheduler doesn't recognize SemverGt/SemverLt/SemverEq operators:**

When the feature gate is disabled and the scheduler encounters a pod with `SemverGt`/`SemverLt`/`SemverEq` operator:

- The toleration filter returns `false` (doesn't match)
- Pod is considered to have untolerated taints
- Filter returns `UnschedulableAndUnresolvable` status
- Pod remains in Pending state.
   - Feature gate on/off test cases

### Version Skew Strategy

The skew between kubelet and control-plane components are not impacted. The kube-scheduler is expected to match the kube-apiserver minor version, but may be up to one minor version older (to allow live upgrades).

In the release it's been added, the feature is disabled by default and not recognized by other components.

Whoever enabled the feature manually would take the risk of component like kube-scheduler being old and not recognize the fields.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `AffinityTaintTolerationSemverComparisonOperatorss`
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
   - The scheduler's TaintToleration plugin won't recognize `SemverGt`/`SemverLt`/`SemverEq` operators and will treat them as non-matching
   - These pods will remain in Pending state with events indicating untolerated taints

3. **New pod creation**: 
   - API server validation will **reject** new pods with Gt/Lt operators
   - Error: `spec.tolerations[].operator: Unsupported value: "SemverGt": supported values: "Equal", "Exists"`

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

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

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

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

N/A

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

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

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

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
