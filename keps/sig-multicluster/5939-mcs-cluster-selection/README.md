<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [x] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [x] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [x] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [x] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [x] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-5939: Cluster Selection for Multi-Cluster Services

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Locality-based failover](#story-1-locality-based-failover)
    - [Story 2: Multi-dimension selection with arbitrary properties](#story-2-multi-dimension-selection-with-arbitrary-properties)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Overview](#api-overview)
    - [Property Selector](#property-selector)
    - [Validation](#validation)
  - [Cluster Selection Algorithm](#cluster-selection-algorithm)
    - [Resolving Cluster Selectors](#resolving-cluster-selectors)
    - [Consuming Cluster Selector Results](#consuming-cluster-selector-results)
  - [Interaction with Service Traffic Distribution](#interaction-with-service-traffic-distribution)
  - [Reading ClusterProperties from Constituent Clusters](#reading-clusterproperties-from-constituent-clusters)
  - [Reference Implementation in kubernetes-sigs/mcs-api](#reference-implementation-in-kubernetes-sigsmcs-api)
  - [Conflict Resolution](#conflict-resolution)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA graduation](#beta---ga-graduation)
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
  - [EndpointSlice Active Label](#endpointslice-active-label)
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

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website
[MCS-API]: /keps/sig-multicluster/1645-multi-cluster-services-api
[ClusterProperty]: /keps/sig-multicluster/2149-clusterid
[PlacementDecision]: /keps/sig-multicluster/5313-placement-decision-api
[ClusterProfile]: /keps/sig-multicluster/4322-cluster-inventory
[mcs-api-repo]: https://github.com/kubernetes-sigs/mcs-api

## Summary

The [Multi-Cluster Services API (MCS-API)][MCS-API] lets users export a
Kubernetes `Service` with `ServiceExport` so that it becomes consumable across
a clusterset. When the same `Service` is exported from multiple clusters, each
importing cluster gets a single automatically generated `ServiceImport`.

Today there is no standard way to express which clusters should be preferred. A
consumer cannot say "prefer my own cluster, then my region, then anywhere else".

This KEP introduces a **cluster selection** concept for directing traffic. It
adds an ordered `clusterSelectors` field to `ServiceExport` that lets users
select subsets of constituent clusters based on
[`ClusterProperty`][ClusterProperty] values.

Each entry can use a `propertySelector` based on the standard
`metav1.LabelSelector` type, except that `matchProperties` replaces `matchLabels`
and it may contain a special `@SameAsImporter@` value. The MCS implementation
publishes the clusters matched by each selector in
`ServiceImport.status.clusterSelectorResults` so that consumers can act on them.

## Motivation

MCS-API supports Service fields such as `trafficDistribution` (for example,
`PreferSameZone`) that can steer traffic across clusters. However, these fields
operate only at the node or zone level.

Real-world multi-cluster deployments span regions, countries, and continents.
Users need to keep traffic local to reduce latency and cost while retaining
failover to other clusters. Without a standard API, each MCS implementation
provides its own mechanism, typically through annotations. For example, Cilium's
[Service Affinity](https://docs.cilium.io/en/stable/network/clustermesh/affinity/)
annotation can select only the local cluster or all remote clusters. A standard
CRD field would support more expressive selection and make configurations
portable across MCS implementations.

### Goals

- Allow users to steer traffic to preferred clusters.
- Act as an additional layer on top of `trafficDistribution` and similar
  Service API fields, not a replacement.
- Use [`ClusterProperty`][ClusterProperty] as the source of cluster metadata
  for selection.
- Support arbitrary (user-defined) `ClusterProperty` keys, not only a
  pre-defined set.
- Support multiple selection criteria based on `ClusterProperty`, including
  "same value as the importing cluster" and arbitrary values.
- Define an unambiguous algorithm for MCS implementations and for third-party
  integrations such as Gateway API.

### Non-Goals

- Select individual endpoints. This KEP is only about selecting clusters.
- Introduce complex routing rules, those should be handled by service-mesh
  or Gateway API implementations.
- Enforce hard traffic boundaries. Cluster selection express preferences that
  `ServiceImport` consumers consider when making routing decisions.

## Proposal

This KEP builds on two existing APIs:

1. **[MCS-API]**: provides `ServiceExport`/`ServiceImport` and the conflict
    resolution model.

2. **[About API / `ClusterProperty`][ClusterProperty]**: provides a standard
    way to attach key/value metadata to clusters. This KEP uses those
    properties as the data source for cluster selection.

This KEP adds a new `clusterSelectors` field to `ServiceExport.spec`. The MCS
implementation reconciles the field across the clusterset and writes the result
to `ServiceImport.spec`. Exact-match conflict resolution ensures that all
constituent clusters receive the same configuration. On a conflict, the oldest
`ServiceExport` takes precedence.

`clusterSelectors` is an ordered list whose entries select sets of preferred
clusters. Currently, each entry supports only a `PropertySelector`. The selector
is evaluated against `ClusterProperty` key/value pairs. The MCS implementation
must publish the ordered results in
`ServiceImport.status.clusterSelectorResults`. Consumers should evaluate the
results in order and use the first one with available endpoints by default. If
none has available endpoints, they should use all constituent clusters.

### User Stories

#### Story 1: Locality-based failover

The `api` service is exported from multiple clusters across regions. Each
cluster prefers its own endpoints first, then endpoints in the same region,
then endpoints in the same continent, and finally endpoints in any cluster.

```yaml
apiVersion: multicluster.x-k8s.io/v1beta1
kind: ServiceExport
metadata:
  name: api
spec:
  clusterSelectors:
    - propertySelector:
        matchExpressions:
          - key: cluster.clusterset.k8s.io
            operator: In
            values:
              - "@SameAsImporter@"
    - propertySelector:
        matchProperties:
          region.topology.k8s.io: "@SameAsImporter@"
    - propertySelector:
        matchProperties:
          continent.topology.k8s.io: "@SameAsImporter@"
```

For instance, one of the resulting `ServiceImport`s could look like this:

```yaml
apiVersion: multicluster.x-k8s.io/v1beta1
kind: ServiceImport
metadata:
  name: api
spec:
  # Other ServiceImport spec fields are omitted for brevity.
  clusterSelectors:
    - propertySelector:
        matchExpressions:
          - key: cluster.clusterset.k8s.io
            operator: In
            values:
              - "@SameAsImporter@"
    - propertySelector:
        matchProperties:
          region.topology.k8s.io: "@SameAsImporter@"
    - propertySelector:
        matchProperties:
          continent.topology.k8s.io: "@SameAsImporter@"
status:
  clusters:
    - cluster: cluster-a
    - cluster: cluster-b
    - cluster: cluster-c
    - cluster: cluster-d
  clusterSelectorResults:
    - clusters:
        - cluster-a
    - clusters:
        - cluster-a
        - cluster-b
    - clusters:
        - cluster-a
        - cluster-b
        - cluster-c
```

Cluster selection can be combined with the Service-level
`trafficDistribution: PreferSameZone` to further prefer same-zone endpoints
within the selected clusters.

#### Story 2: Multi-dimension selection with arbitrary properties

Some workloads need rules that combine multiple dimensions. For example, an ML
inference service spanning clusters with different GPU capabilities may select
clusters by both locality and GPU tier.

The following selectors prefer clusters in the same continent with high-end
GPUs, then clusters in the same continent with any GPU, and finally any cluster
with a GPU.

```yaml
apiVersion: multicluster.x-k8s.io/v1beta1
kind: ServiceExport
metadata:
  name: inference-server
spec:
  clusterSelectors:
    # Same continent with high-end GPU
    - propertySelector:
        matchProperties:
          continent.topology.k8s.io: "@SameAsImporter@"
          gpu-tier.mycompany.com: "high"
    # Same continent with any GPU
    - propertySelector:
        matchProperties:
          continent.topology.k8s.io: "@SameAsImporter@"
        matchExpressions:
          - key: gpu-tier.mycompany.com
            operator: Exists
    # Any cluster with a GPU
    - propertySelector:
        matchExpressions:
          - key: gpu-tier.mycompany.com
            operator: Exists
```

Note that `continent.topology.k8s.io` is not a standard key today.

### Notes/Constraints/Caveats

Cluster selection applies only to `ClusterIP` (non-headless) multi-cluster
services. Headless `ServiceImport`s return individual pod IPs via DNS and are
not compatible with cluster selection. If `clusterSelectors` is set on a
`ServiceExport` for a headless service, the `ServiceExport` must receive a
`Valid` condition with status `False` and reason `InvalidServiceType`.

A future revision may add selection criteria beyond `propertySelector`, such as
CEL expressions or externally driven selection. For now, `ServiceExport`
supports only inline selection through `propertySelector` and `ClusterProperty`
data.

A dedicated `PropertySelector` type avoids confusion with selectors over
`ClusterProfile` labels and leaves room to add such selectors in the future.

Any future criteria would be mutually exclusive within each entry. Different
entries could use different criteria. Evaluation would remain ordered.

CEL support needs more discussion. The evaluation model, cost estimation, and
library configuration are unresolved.

Externally driven selection could build on
[`PlacementDecision`][PlacementDecision]. Adding it would likely require a
dedicated KEP.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

### API Overview

The `ServiceExport.spec` gains a `clusterSelectors` field:

```go
type ServiceExportSpec struct {
    // ... existing fields ...

    // ClusterSelectors is an ordered list of entries. Each entry selects a set
    // of preferred clusters.
    // +optional
    // +listType=atomic
    // +kubebuilder:validation:MaxItems=16
    ClusterSelectors []ClusterSelector `json:"clusterSelectors,omitempty"`
}
```

The `ServiceImport.spec` gains the same field, populated by the MCS
implementation after conflict resolution:

```go
type ServiceImportSpec struct {
    // ... existing fields ...

    // ClusterSelectors is the reconciled ordered list derived from the
    // constituent ServiceExports. Each entry selects a set of preferred
    // clusters.
    // +optional
    // +listType=atomic
    // +kubebuilder:validation:MaxItems=16
    ClusterSelectors []ClusterSelector `json:"clusterSelectors,omitempty"`
}
```

The `ClusterSelector` type is defined as follows:

```go
// ClusterSelector selects a set of clusters.
type ClusterSelector struct {
    // PropertySelector selects clusters by matching ClusterProperty
    // key/value pairs. The special value "@SameAsImporter@" may appear in
    // matchProperties values or matchExpressions values when the operator is
    // In or NotIn. It means "use the same value as the importing cluster for
    // this key."
    // +required
    PropertySelector *PropertySelector `json:"propertySelector"`
}

// PropertySelector is a property query over a set of clusters. The results of
// matchProperties and matchExpressions are ANDed. An empty PropertySelector is
// invalid.
// +structType=atomic
type PropertySelector struct {
    // MatchProperties is a map of ClusterProperty key/value pairs. A single
    // key/value pair is equivalent to an element of matchExpressions whose key
    // field is the map key, operator is In, and values contains only the map
    // value. The requirements are ANDed.
    // +optional
    MatchProperties map[string]string `json:"matchProperties,omitempty"`

    // MatchExpressions is a list of ClusterProperty selector requirements. The
    // requirements are ANDed.
    // +optional
    // +listType=atomic
    MatchExpressions []PropertySelectorRequirement `json:"matchExpressions,omitempty"`
}

// PropertySelectorRequirement and PropertySelectorOperator mirror the fields
// and operators of their metav1 equivalents. Their definitions are omitted
// here for brevity.
```

The `ServiceImport.status` gains a `clusterSelectorResults` field. The MCS
implementation populates this field for consumers such as its own datapath or
Gateway API implementations:

```go
type ServiceImportStatus struct {
    // ... existing fields ...

    // ObservedGeneration is the ServiceImport generation observed when this
    // status was last updated. It helps determine whether the status has caught
    // up with spec changes.
    // +optional
    ObservedGeneration int64 `json:"observedGeneration,omitempty"`

    // ClusterSelectorResults is the ordered list of effective cluster selector
    // results for this Service. Consumers must use this field directly. They
    // must not correlate results with spec.clusterSelectors entries because
    // the two can be out of sync.
    // +optional
    // +listType=atomic
    ClusterSelectorResults []ClusterSelectorResult `json:"clusterSelectorResults,omitempty"`
}

type ClusterSelectorResult struct {
    // Clusters is the list of source clusters whose ClusterProperties matched
    // this selector. Each value is a cluster ID as exposed in
    // ServiceImport.status.clusters and in the
    // multicluster.kubernetes.io/source-cluster EndpointSlice label.
    // +optional
    // +listType=atomic
    Clusters []string `json:"clusters,omitempty"`
}
```

Note that `observedGeneration` is intended for debugging status freshness.
Consumers should treat the current `clusterSelectorResults` value as effective
even when `observedGeneration` lags `metadata.generation`.

#### Property Selector

`PropertySelector` is based on `metav1.LabelSelector`, except that
`matchProperties` replaces `matchLabels`. It uses the same operators and
matching semantics, but is evaluated against `ClusterProperty` key/value pairs
rather than object labels.

The special value `@SameAsImporter@` may appear in `matchProperties` or
`matchExpressions` values when the operator is `In` or `NotIn`. During
evaluation, it is replaced with the importing cluster's value for that key.
If the importing cluster does not have that property, the selector matches
no clusters. As with `trafficDistribution`, users can say "prefer clusters in
the same region as me" without hard-coding a specific region. Every
`ServiceExport` for the service can use the same configuration.

The MCS implementation may resolve `@SameAsImporter@` when populating
`ServiceImport.spec.clusterSelectors`. Consumers can then use the selectors
without resolving the placeholder themselves.

#### Validation

An empty `propertySelector` is invalid. In that case, the `ServiceExport` must
receive a `Valid` condition with status `False` and reason
`InvalidClusterSelector`.

Otherwise, `PropertySelector` follows the validation of `metav1.LabelSelector`
with two differences: keys must be valid `ClusterProperty` resource names, and
values are not restricted to Kubernetes label-value syntax. Invalid selectors
must either be rejected by API validation or reported on the `ServiceExport`
with a `Valid` condition with status `False` and reason
`InvalidClusterSelector`.

If future selection criteria are added, each entry must still specify exactly
one criterion. Specifying multiple fields in one entry, such as both
`propertySelector` and a CEL expression, is invalid. The `ServiceExport` must
receive a `Valid` condition with status `False` and reason
`InvalidClusterSelector`.

### Cluster Selection Algorithm

#### Resolving Cluster Selectors

The MCS implementation must evaluate `clusterSelectors` in each importing
cluster and publish the ordered results in `status.clusterSelectorResults`.

Results follow selector order. An empty `clusters` list means the selector
matched no clusters. Endpoint conditions must not affect these results.

If selector evaluation fails, the MCS implementation must preserve the complete
list of previously computed results rather than publish a partial update.

#### Consuming Cluster Selector Results

Consumers that support cluster selection, such as the datapath of an MCS
implementation or a Gateway API implementation, must follow this algorithm by
default:

1. For each result in `status.clusterSelectorResults` (in order):
   1. Collect endpoints from the listed source clusters that are eligible for
      the traffic being handled.
   2. If there is at least one eligible endpoint that is ready or serving, use
      those source clusters and stop. Otherwise, continue to the next result.
2. If no result has at least one eligible endpoint that is ready or serving,
   use all constituent clusters.

Endpoint eligibility depends on the relevant port, protocol, and
`internalTrafficPolicy`. Whether endpoints are also filtered by address family
is implementation specific. Different traffic contexts may therefore select
different preferred clusters.

Consumers may use additional logic to advance to the next result, for example
when there are too few endpoints or based on runtime metrics such as latency.
However, such behavior must be explicitly enabled by the user for the affected
Service because it may override the user's intent. (For example: does the user
prefer certain clusters because they are faster or because they are cheaper?)

When the selected clusters change, consumers that manage live traffic should
make a best effort to avoid abrupt connection termination.

### Interaction with Service Traffic Distribution

Cluster selection runs before Service-level `trafficDistribution` such as
`PreferSameZone`. `trafficDistribution` then applies only to endpoints in the
selected clusters.

For now, `trafficDistribution` does not influence which result is chosen.
By default, consumers must evaluate the results in order and use the first result
that has at least one eligible endpoint that is ready or serving, even when none
of those endpoints is in the consumer's zone. In other words,
`trafficDistribution` never causes the next result to be considered. Separate
implementation-specific logic may do so when explicitly enabled as described
above.

### Reading ClusterProperties from Constituent Clusters

The MCS implementation may query remote clusters and cache the data in memory.
It may also use a hub cluster, a central registry, or
[`ClusterProfile`][ClusterProfile] objects reflected locally. This KEP does not
prescribe a mechanism. A future revision may define one.

### Reference Implementation in kubernetes-sigs/mcs-api

The [`kubernetes-sigs/mcs-api`][mcs-api-repo] repository will provide the CRD
types for the new `clusterSelectors` field on `ServiceExport` and
`ServiceImport`. It will also provide a generic `PreparedSelector` API for
evaluating selectors:

```go
// Cluster represents a cluster in the clusterset. Implementations of this
// interface provide their own backing type.
type Cluster interface {
    // GetID returns the cluster ID (typically the value of the
    // cluster.clusterset.k8s.io ClusterProperty).
    GetID() string
    // GetProperties returns the ClusterProperty key/value pairs.
    GetProperties() map[string]string
}

// PreparedSelector is a compiled ClusterSelector ready for evaluation.
type PreparedSelector[T Cluster] interface {
    // SelectClusters returns the subset of candidates that match.
    SelectClusters(candidates []T) ([]T, error)
}

// PrepareSelector compiles a ClusterSelector for the given importer.
// It resolves any @SameAsImporter@ placeholders using the importer's
// properties. The returned PreparedSelector is bound to the importer's current
// properties and can be reused across evaluations. The caller must call
// PrepareSelector again when those properties change.
func PrepareSelector[T Cluster](selector ClusterSelector, importer T) (PreparedSelector[T], error)
```

This API provides reusable selector preparation and evaluation, not a complete
cluster-selection implementation.

### Conflict Resolution

`clusterSelectors` is a global property of the multi-cluster service. It
follows the standard MCS-API conflict resolution policy:

- All `ServiceExport`s for the same service must have the same ordered
  `clusterSelectors` list. If they differ, the `ServiceExport` with the oldest
  `creationTimestamp` takes precedence.
- On a conflict, all `ServiceExport`s must receive a `Conflict` condition with
  status `True` and reason `ClusterSelectorsConflict`.
- The MCS implementation must write the resolved value to the `ServiceImport`.

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


### Graduation Criteria


#### Alpha -> Beta graduation

- At least two MCS implementations support the feature.
- Conformance tests are available and pass for at least two MCS implementations.

#### Beta -> GA graduation

- [MCS-API] (KEP-1645) and [`ClusterProperty`][ClusterProperty] (KEP-2149) have
  both graduated to stable.
- Conformance tests are mature and extensive.
- Most active MCS implementations support the feature and pass the conformance
  tests.
- Support for CEL expressions is either specified as an additional selection
  criterion or explicitly ruled out.

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

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

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->


###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

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

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

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

<!--
This section must be completed when targeting beta to a release.
-->

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

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

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

`@SameAsImporter@` overloads `PropertySelector` values. This is an unusual
pattern in the Kubernetes ecosystem. The About API (KEP-2149) places no
character restrictions on general `ClusterProperty` values. As a result, a
literal property value of `@SameAsImporter@` would be indistinguishable from the
placeholder. Such a collision is unlikely in practice. The convenience for
common use cases justifies the risk.

## Alternatives

### EndpointSlice Active Label

One alternative was to mark MCS `EndpointSlice`s with a label such as
`multicluster.kubernetes.io/active`. Consumers would then select only active
`EndpointSlice`s. This would simplify consumers but make `EndpointSlice`
synchronization more complex and expensive. The MCS implementation would need
to react to every endpoint condition change and every `ClusterProperty` change
across all source clusters.

This approach would also be prone to race conditions. When transitioning from
one selector to another, the MCS implementation would need to mark the new
selector's `EndpointSlice`s as active before marking the old selector's
`EndpointSlice`s as inactive. Otherwise, consumers could temporarily observe no
active `EndpointSlice`s. This would cause incorrect cluster selection.

This KEP instead exposes the cluster selector results without accounting for
endpoint conditions. This approach is easier to implement correctly and
efficiently. Consumers observe the results and the endpoint conditions together
and choose the preferred clusters themselves.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
