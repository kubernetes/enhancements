# KEP-5313: PlacementDecision API

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: GPU-aware AI training](#story-1-gpu-aware-ai-training)
    - [Story 2: Progressive rollout](#story-2-progressive-rollout)
    - [Story 3: Disaster recovery](#story-3-disaster-recovery)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

Today every multicluster scheduler publishes its own API to convey where a workload should run,
forcing downstream tools such as GitOps engine, workload orchestrator, progressive rollout controller,
or AI/ML pipeline having to understand a scheduler specific API.

This KEP introduces a vendor neutral `PlacementDecision` API that standardizes
the output of multicluster placement calculations.  
A `PlacementDecision` object is data only: a namespaced list of chosen clusters
whose `clusterName` values must map one to one to `ClusterProfile` names defined by the
[ClusterProfile API](https://github.com/kubernetes/enhancements/pull/4322).
Any scheduler can emit the object and any consumer can watch it.
Neither side needs to know about the other, enabling true plug and play in the multicluster stack.

```mermaid
flowchart LR
    Scheduler/PlacementController -- writes --> PlacementDecisions
    Tools["AI/ML Pipeline<br>GitOps Engine<br>Progressive Rollout Controller<br>Workload Orchestrator"] -- reads from --> PlacementDecisions
    Tools -- performs actions on --> Spokes["Spoke/Managed Clusters<br>- cluster1<br>- cluster2<br>- cluster2<br> -etc."]
```

## Motivation

A typical multicluster setup:

1. Scheduler: examines the fleet (`ClusterProfile` objects),
   and other signals/metrics and decides *where* a workload should land.
2. Consumer: GitOps engine, workload orchestrator, progressive rollout controller, AI/ML pipeline
   read that decision and act (usually by creating [Work](https://github.com/kubernetes-sigs/work-api) objects).

Currently every scheduler have its own API for step #1
so each consumer might need to learn different APIs which slow down integration,
locking users to specific vendors, and complicate RBAC/validation work.

A standardized `PlacementDecision` API:

* Decouples schedulers from consumers swap either side without rewriting the other.
* Aligns with the SIG-Multicluster `ClusterProfile` inventory.
* Simplifies RBAC: one resource schema to secure.
* Allows consumers (GitOps engines, workload orchestrators, progressive rollout controllers,
  AI/ML pipelines) to act on placement results through one standardized PlacementDecision API.

### Goals

* Define a namespaced scope, minimalistic, data only `PlacementDecision` API that lists selected clusters.
* Support continuous rescheduling: decision list may be updated.
* Provide a general approach for consumers to read the placement decision.
* Guarantee that every (`clusterNamespace`, `clusterName`) pair matches a
  `ClusterProfile` (`metadata.namespace`, `metadata.name`) in the fleet.
* Provide label conventions so consumers can retrieve all slices of one placement.
* Leave room for schedulers implementations.

### Non-Goals

* Describing how a scheduler made its choice (Placement API spec).
* Describing how consumers access selected clusters.
* Embedding orchestration logic or consumer feedback in `PlacementDecision`.
* Replace Work API which is responsible for actually applying the workload.

## Proposal

### Terminology

- **Placement**: A scheduler request that asks "where should this workload run?".

- **Scheduler**: Placement controller that writes `PlacementDecisions` based on `ClusterProfiles` and
  scheduling/placement requirements/specs.

- **Consumer**: Any controller (GitOps engine, workload orchestrator, progressive rollout controller, AI/ML pipeline)
  that watches `PlacementDecisions` and acts.

### API Definition

```
// PlacementDecision publishes the selected clusters for one Placement.
type PlacementDecision struct {
  metav1.TypeMeta   `json:",inline"`
  metav1.ObjectMeta `json:"metadata,omitempty"`

  // Up to 100 ClusterDecisions per object (slice) to stay well below the etcd limit.
  // +kubebuilder:validation:MinItems=0
  // +kubebuilder:validation:MaxItems=100
  Decisions []ClusterDecision `json:"decisions"`

  // PlacementRef is an immutable field that ties this decision back to its originating placement.
  // The referenced placement might not be a Kubernetes resource.
  // The name could be a correlation key understood by the scheduler and consumers.
  // +kubebuilder:validation:XValidation:rule="self == oldSelf",message="placementRef is immutable"
  PlacementRef PlacementRef `json:"placementRef"`

  // Optional: Name of the scheduler that created this decision.
  // +optional
  SchedulerName string `json:"schedulerName,omitempty"`
}

// Label that links all slices to their originating Placement request.
// Immutable and must equal the value of the placementName field.
const PlacementLabel = "multicluster.x-k8s.io/placement"

// Optional: Label that indicate the index position of this slice when order matters.
const DecisionIndexLabel = "multicluster.x-k8s.io/decision-index"

// ClusterDecision references a target ClusterProfile for placement.
type ClusterDecision struct {
  // Reference to the target ClusterProfile.
	ClusterProfileRef ClusterProfileRef `json:"clusterProfileRef"

  // Optional: Reason to why this cluster was chosen.
  // +optional
  Reason string `json:"reason,omitempty"`
}

// ClusterProfileRef references a ClusterProfile by namespace and name.
type ClusterProfileRef struct {
	// Namespace of the referenced ClusterProfile.
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`

	// Name of the referenced ClusterProfile.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// PlacementRef is an immutable field that ties this decision back to its originating placement.
// The referenced placement might not be a Kubernetes resource.
// The name could be a correlation key understood by the scheduler and consumers.
type PlacementRef struct {
  // Name of the Placement that this decision answers.
  // Must equal the value of the label multicluster.x-k8s.io/placement.
  // +kubebuilder:validation:MinLength=1
  // +kubebuilder:validation:MaxLength=63
  Name string `json:"name"`

  // Optional: API group of the Placement resource if one exists.
  // Omit when the placement is not represented as a Kubernetes object.
  // +optional
  APIGroup string `json:"apiGroup,omitempty"`

  // Optional: kind of the Placement resource if one exists.
  // Omit when the placement is not represented as a Kubernetes object.
  // +optional
  Kind string `json:"kind,omitempty"`

  // Optional: namespace of the Placement resource when it is namespaced.
  // Omit when the placement is not represented as a Kubernetes object or
  // cluster scoped placement resource.
  // +optional
  Namespace string `json:"namespace,omitempty"`
}
```

### API Example

```
apiVersion: multicluster.x-k8s.io/v1alpha1
kind: PlacementDecision
metadata:
  name: app-placement-decision-0
  namespace: argocd
  labels:
    multicluster.x-k8s.io/decision-index: 0
    multicluster.x-k8s.io/placement: app-placement
schedulerName: multicluster-placement-controller
placementRef:
  apiGroup: multicluster.x-k8s.io
  kind: Placement
  namespace: argocd
  name: app-placement
decisions:
- clusterProfileRef:
    namespace: fleet1
    name: cluster1
  reason: "GPUs available"
- clusterProfileRef:
    namespace: fleet1
    name: cluster2
  reason: "GPUs available"
```

### Slicing

* Following [EndpointSlice](https://kubernetes.io/docs/concepts/services-networking/endpoint-slices/) design,
  a single Placement can fan out to N `PlacementDecision` slices,
  each limited to 100 clusters (`EndpointSlice`'s default).
* All slices for one `Placement` MUST carry the same
  `multicluster.x-k8s.io/placement=<placement-name>` label so consumers can List with a label selector.
* If a scheduler needs to preserve the order of selected clusters and the result spans multiple slices,
  it should label each PlacementDecision with `multicluster.x-k8s.io/decision-index=<index>`
  where <index> starts at 0 and increments by 1.
  Consumers that require ordering can sort by this label.

### Lifecycle
- **Create**: The scheduler creates the slice with the list of clusters in the decisions,
  and set the label `multicluster.x-k8s.io/placement=<placement-name>`.
  When order matters, set `multicluster.x-k8s.io/decision-index` on every slice.
  The scheduler may choose to populate the reason for each decision for consumers/end-users
  (ie, for debugging purposes).

- **Update / Reschedule**: The scheduler may add or remove clusters in  decisions at any time.
  If the number of target clusters crosses the 100 limit,
  it must create or delete slices to maintain the slicing rule.
  The value of required label `multicluster.x-k8s.io/placement` should not change and treated as immutable.
  If order changes, update decision-index values accordingly so consumers can detect the new order.

  If heavy churn is a concern, a scheduler may treat `decisions` as an unordered set and
  maintain it in a deterministic order (ie, alphabetical sorting).
  When the cluster set itself has not changed, this stable ordering produces an identical set of clusters,
  so the API server skips the write and no extra change events reach consumers.

- **Delete**: When a placement is no longer required,
  the scheduler deletes every related `PlacementDecision` slice.
  Consumers should react to the delete event and remove any workload previously applied to the listed clusters.

### Ownership

- The scheduler that creates the `PlacementDecision` owns the object.
  It is solely responsible for all writes (`create`, `update`, `patch`, `delete`).
  The consumers of the `PlacementDecision` MUST treat the object as read only (`get`, `list`, `watch`).
- RBAC will enforce this contract by granting the scheduler write verbs on `PlacementDecisions`,
  while limiting consumers to read only access.

### Relationship to other SIG-Multicluster (SIG-MC) APIs
* **ClusterProfile** The inventory. Each decision must reference a matching name `ClusterProfile`
* **Work API** The workload. A consumer may read `PlacementDecision` then for each cluster creates `Work`.

### Consumer Feedback
Consumer feedback is intentionally out of scope for PlacementDecision.
The PlacementDecision object's sole purpose is to publish the scheduler’s chosen cluster list.
Once it has been created then it should be treated as read only by consumers.

Allowing consumers to update the same PlacementDecision would complicate lifecycle ownership
(whether the scheduler or the consumer is responsible for adding, updating, or removing cluster entries).
It would also complicate the security/permission because maybe a malicious consumer could
update the decision and move some workloads to unintended clusters.

When consumers need to do feedback to the scheduler,
they should do so through a separate channel like events, metrics,
or maybe even a purpose built PlacementFeedback API
so they have clear write authority and the scheduler can decide what to do with that feedback in the end.

### User Stories (Optional)

#### Story 1: GPU-aware AI training

* ML scheduler scores every `ClusterProfile` by available GPUs, cost, etc.  
* Scheduler writes `PlacementDecision` listing the best clusters.
* A GitOps/Work-API syncer watches the decision and deploys the training job only to the listed clusters.
* If cost rises or GPUs becoming unavailable, the scheduler updates `PlacementDecision` with new list,
  the syncer act accordingly.

#### Story 2: Progressive rollout

* A progressive rollout controller begins with some canary clusters.  
* It creates `PlacementDecision` containing just those clusters.  
* Gradually updates `PlacementDecision` based on health and then eventually all clusters.  
* Consumers watch and deploy only where the decision says.  

#### Story 3: Disaster recovery

* A DR controller monitors `ClusterProfiles` status.
* Controller updates the corresponding `PlacementDecision`,
  replacing primary cluster with standby cluster.
* Syncer deletes workloads from the failed cluster and recreates them in the standby cluster.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

## Design Details

* Scope: Namespace scoped for RBAC parity with Work and ClusterProfile.

* The resource is pure data following `EndpointSlice` convention.

* Max size: 100 ClusterDecision entries per slice keeps object well below etcd limit.

* Validation: A webhook may verify that every (clusterNamespace, clusterName)
  pair has a matching ClusterProfile in the fleet.
  The field `placementName` is populated and immutable.
  The label `multicluster.x-k8s.io/placement=<placement-name>` is populated, immutable,
  and value equal to `placementName` field.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

* Unit tests for CRD defaults/validation.

* Ensuring slice size <= 100 and required labels exists.

* Integration tests: fake scheduler writes decisions and fake consumer verifies watches.
  Scale test with large number of clusters and placements.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

- <test>: <link to test coverage>

##### e2e tests

- <test>: <link to test coverage>

### Graduation Criteria

#### Alpha

- A CRD definition and generated client.
- A dummy controller and unit test to validate the CRD and client.

#### Beta

- Gather feedback from users during the Alpha stage to identify any
  issues, limitations, or areas for improvement. Address this feedback
  by making the necessary changes to the API and iterating on its design
  and functionality.
- At least two providers and one consumer using `PlacementDecision` API.
- Conformance test suite for schedulers.
- Metrics for slice count and QPS.
- Backwards compatible field/label stability.

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing ie. downgrade tests and scalability
  tests
- Allowing time for feedback
- Stability: The API should demonstrate stability in terms of its
  reliability.
- Functionality: The API should provide the necessary functionality for
  multicluster scheduling, including the ability to distribute workloads
  across clusters. This should be validated through a series of
  functional tests and real-world use cases.
- Integration: Ensure that the API can be easily integrated with popular
  workload distribution tools, such as GitOps and Work API. This may
  involve developing plugins or extensions for these tools or providing
  clear guidelines on how to integrate them with the unified API.
- Performance and Scalability: Conduct performance and scalability tests
  to ensure that the API can handle a large number of clusters and
  workloads without degrading its performance. This may involve stress
  testing the API with a high volume of requests or simulating
  large-scale deployments.

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug
reports, in back-to-back releases.

### Upgrade / Downgrade Strategy

### Version Skew Strategy

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [x] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

- No default Kubernetes behavior is currently planned to be based on
  this feature; it is designed to be used by the separately installed,
  out-of-tree, multicluster management providers and consumers.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

- Yes, as this feature only describes a CRD, it can most directly be
  disabled by uninstalling the CRD.

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

- As a dependency only for an out-of-tree component, there will not be
  e2e tests for feature enablement/disablement of this CRD in core
  Kubernetes. The e2e test can be provided by multicluster management
  providers who support this API.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

###### What specific metrics should inform a rollback?

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

### Dependencies


###### Does this feature depend on any specific services running in the cluster?

### Scalability

###### Will enabling / using this feature result in any new API calls?

###### Will enabling / using this feature result in introducing new API types?

###### Will enabling / using this feature result in any new calls to the cloud provider?

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

## Drawbacks

## Alternatives

- Status quo: every multicluster provider/scheduler ships its own API leads to consumer bloat and vendor lock-in.

- Extending `Work API`: overloads a workload syncner API with scheduling details which couples the where with the what.

## Infrastructure Needed (Optional)
