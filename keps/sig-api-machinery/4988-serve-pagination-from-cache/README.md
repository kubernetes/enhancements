# KEP-4988 Snapshottable API server cache

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Snapshotting](#snapshotting)
  - [Cache Inconsistency detection](#cache-inconsistency-detection)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Memory overhead](#memory-overhead)
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

The kube-apiserver's caching mechanism (watchcache) efficiently serves requests
for the latest observed state. However, `LIST` requests for previous states,
either via pagination or by specifying a `resourceVersion`, bypass the cache and
are served directly from etcd. This significantly increases the performance cost,
and in aggregate, can cause stability issues. This is especially pronounced when
dealing with large resources, as transferring large data blobs through multiple
systems can create significant memory pressure. This document proposes an
enhancement to the kube-apiserver's caching layer to enable efficient serving all
`LIST` requests from the cache.

## Motivation

When the API server serves a `LIST` requests directly from etcd, it introduces
significant stability and reliability concerns:

*   **Unpredictable Memory Pressure:** Retrieving data from etcd and constructing
    responses involves significant memory allocations on the API server.
    The volume of data retrieved from etcd can vary drastically depending on
    object sizes. This results in unpredictable memory pressure, making it difficult
    to provision resources effectively and increasing the risk of Out-of-Memory (OOM) errors.
*   **Ineffective API Priority and Fairness (APF) Throttling:** The API server's
    overload protection mechanism, API Priority and Fairness (APF), primarily
    throttles based on the *predicted cost* of a request, which is derived from
    factors like latency and object count. While these factors provide some
    indication of computational cost, they do not accurately reflect the memory
    footprint. Crucially, we lack visibility into the per-request memory allocations.
    Therefore, APF cannot effectively throttle requests based on actual memory usage,
    leaving the API server vulnerable to memory exhaustion.

These issues with serving data directly from etcd lead to unpredictable and volatile API server memory usage.

Remarkably, the API server already maintains all the necessary data in the watchcache.
By enabling all `LIST` requests to be served from the watchcache, we can
significantly reduce memory pressure and improve the effectiveness of APF throttling,
leading to a more stable and reliable API server.

### Goals

- Reduce memory allocations by supporting all types of LIST requests from cache
- Ensure responses returned by cache are consistent with etcd

### Non-Goals

- Change semantics of the `LIST` request
- Support indexing when serving for all types of requests.
- Enforce that no client requests are served from etcd

## Proposal

This proposal leverages the recent rewrite of the watchcache storage layer to
use a B-tree (https://github.com/kubernetes/kubernetes/pull/126754) to enable
efficient serving of remaining types of LIST requests from the watchcache.
This approach aims to improve the performance and stability of the API server by
reducing the need to access etcd directly for historical data.

While the overall goal is to have watch cache support all types of request, we
will split the rollout into multiple phases that will be controlled by separate
feature flags.
* Phase 1: Support of paginated request.
* Phase 2: Support of exact revision match.

See also https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/365-paginated-lists#potential-future-extensions

This change increases the K8s dependency on cache, the impact of a bug in caching logic is severe.
Any incorrect behavior will be persisted in a local apiserver, and is extreamly hard to debug.
As the proposed changes will in the end cover all apiserver list calls behind a cache, we need a better way to detect. 
We propose to add a automatic mechanism for validating cache consistency with etcd,
allowing users to trust it's results withount need to manually debug.

### Snapshotting

1. **Snapshot Creation:** When a watch event is received, the cacher creates
   a snapshot of the B-tree based cache using the efficient [Clone()[] method.
   This method creates a lazy copy of the tree structure, minimizing overhead.
   Since the watch cache already stores the history of watch events,
   the B-tree maintains just pointers to the in-use memory, storing only minimal necessary data.
2. **Snapshot Storage:** Snapshots are stored in a separate tree data structure,
   keyed by resourceVersion. This tree structure facilitates efficient lookup of
   the "nextSmaller" element, as resourceVersions are not necessarily sequential.
3. **Serving:** When a request requiring response based on previous snapshot arrives,
   the API server performs the following steps:
  - Extract the resourceVersion from request.
  - Looks up the "nextSmaller" snapshot based on the resourceVersion.
  - Constructs the response using data from the retrieved snapshot.
  **Edge cases:**
    - Requested resourceVersion is smaller than any available snapshot:
      This indicates that the requested data has been cleaned up. In this scenario,
      the API server falls back to serving the request from etcd.
    - Requested resourceVersion is larger than the latest snapshot:
      This could indicate a future resourceVersion or a situation where the watch
      cache is lagging behind. The API server performs a consistent read from
      etcd to confirm the existence of the future resourceVersion or waits for
      the watch cache to catch up.
4. **Snapshot Cleanup:** Snapshots are subject to a Time-To-Live (TTL) mechanism
   similar to watch events. The proposed approach leverages the existing process
   that limits the number of events to 10,000 within a 75-second window
   (configurable via request timeout). Additionally, snapshots are purged during
   cache re-initialization.

[Clone()]: https://pkg.go.dev/github.com/google/btree#BTree.Clone

### Cache Inconsistency detection

For the first iteration of the mechanism we will calculate the hash based on LIST response
at some specific RV of each resource and expose it as a metric.

For Alpha users should be able to setup an alert to detect if hash values for etcd and cache don't match.
For Beta we plan to introduce a automatic action to purge the cache if inconsistent with etcd.

Once every 5 minutes (default compaction interval) for each resource we will
send a LIST request to both etcd and watch cache and calculate hash of the response objects.
Hash calculations will be shifted by random jitter (1-5 minutes) to minimize stacking multiple concurrent calculations.
First we will make a non-consistant list with `RV=0` from cache, read the revision and send a exact revision list request `RV=X` to etcd.
This way will validate the latest available in cache RV without needing to account cache staleness.

New metric `apiserver_storage_hash` will expose last results of calculating hash using [64bit FNV](https://pkg.go.dev/hash/fnv) algorithm.
To calculate the hash from LIST response we will calculate hash of response structure using approach similar to https://github.com/gohugoio/hashstructure.
While decoding and calculating hash on the whole structure might be costly, we think it's acceptable if done with such high period when compared to cost of normal LISTs from etcd.
Calculating hash on the structure itself allows us to mitigate issues with object versioning.

Example metric:
```
apiserver_storage_hash{resource="pods", storage="etcd", hash="f364dcd6b58ebf020cec3fe415e726ab16425b4d0344ac6b551d2769dd01b251"} 1
apiserver_storage_hash{resource="pods", storage="cache", hash="f364dcd6b58ebf020cec3fe415e726ab16425b4d0344ac6b551d2769dd01b251"} 1
```
Metric values for each resource should be updated atomically to prevent false positives.

### Risks and Mitigations

#### Memory overhead

B-tree snapshots are designed to minimize memory overhead by storing pointers to
the actual objects, rather than the objects themselves. Since the objects are
already cached to serve watch events, the primary memory impact comes from the
B-tree structure itself. To quantify the memory overhead, we run 5k scalability tests.
They should represent the worst case scenario, as they utilize large number of small objects.
The results are promising:

* **Object Allocations:** Allocation profile collected during the test test has
  shown an increase of 7GB in object allocations, which translates to a
  negligible 0.2% of total allocations.
* **Memory Usage:** Memory in use profile collected during the test has shown
  Btree memory usage of 300MB, representing a 1.3% of total memory used.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

- Ensure the pagination is well tested

##### Unit tests

- `k8s/apiserver/pkg/storage/cache`: `2024-12-12` - `<test coverage>`

##### Integration tests

We should add a test to validate purging of watch cache.

##### e2e tests

We should add a tests that validates metrics exposed for inconsistency detection.
Test should cover couple of resources including resources with conversion.

### Graduation Criteria

#### Alpha

- Snapshotting implemented behind a feature gate disabled by default.
- Inconsistency detection is implemented behind a feature gate enabled by default.

#### Beta

- Inconsistency detection mechanism is qualified and no mismatch detected.

#### GA

TODO

### Upgrade / Downgrade Strategy

The feature is purely in-memory so update/downgrade doesn't require any
specific considerations.

### Version Skew Strategy

Feature touches only kube-apiserver and coordination between individual
instances is not needed.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: PaginationFromCache
  - Components depending on the feature gate: kube-apiserver
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

Yes, kube-apiserver paginating LIST requests will no longer require request to etcd.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, via disabling the feature-gate in kube-apiserver.

###### What happens if we reenable the feature if it was previously rolled back?

The feature is purely in-memory so it will just work as enabled for the first time.

###### Are there any tests for feature enablement/disablement?

The feature is purely in-memory so feature enablement/disablement will not provide
additional value on top of feature tests themselves.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?


###### What specific metrics should inform a rollback?

Mismatch in hash label for different storage exposed by `apiserver_storage_hash` metric by the same apiserver.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

NO

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

This is control-plane feature, not a workload feature.

###### How can someone using this feature know that it is working for their instance?

This is control-plane feature, not a workload feature.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

[API call latency SLO](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/api_call_latency.md)

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

[API call latency SLI](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/api_call_latency.md)

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No, we expect the [API call latency SLI](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/api_call_latency.md) to improve.


###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Overall we expect that cost of serving pagination will go down, however caching
might increase RAM usage, if the client reads the first page, but never
paginates. We expect that most controllers will read all pages.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The feature is kube-apiserver feature - it just doesn't work if kube-apiserver is unavailable.

###### What are other known failure modes?

Inconsistency of watch cache, should be addressed by the consistency checking mechanism.
For the first iteration we will enable users to define an alert on a metric and detect if cache became inconsistent with etcd.

###### What steps should be taken if SLOs are not being met to determine the problem?

Disabling the feature-gate.

## Implementation History

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
