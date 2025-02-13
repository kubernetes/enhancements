# KEP-4988 Snapshottable API server cache

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Snapshotting](#snapshotting)
  - [Cache Inconsistency Detection Mechanism](#cache-inconsistency-detection-mechanism)
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
use a B-tree ([kubernetes/kubernetes#126754](https://github.com/kubernetes/kubernetes/pull/126754)) to enable
efficient serving of remaining types of LIST requests from the watchcache.
This aims to improve API server performance and stability by minimizing direct etcd access for historical data retrieval.
This aligns with the future extensions outlined in KEP-365 (Paginated Lists): [link to KEP](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/365-paginated-lists#potential-future-extensions).

However, this increased reliance on the watchcache significantly elevates the impact of any bugs in the caching logic.
Incorrect behavior would be locally within API server memory, making debugging exceptionally difficult.
Given that the proposed changes will ultimately route *all* API server LIST calls through the cache,
a robust mechanism for detecting inconsistencies is crucial.
Therefore, we propose an automatic mechanism to validate cache consistency with etcd,
providing users with confidence in the cache's accuracy without requiring manual debugging efforts.

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

### Cache Inconsistency Detection Mechanism

We will periodically calculate a hash of the data in both the etcd datastore
and the watch cache and compare them. For the Alpha phase, detection will be passive.
A metric will be exposed, allowing users to configure alerts that trigger on hash mismatch,
thus indicatory of potential inconsistency and enabling us to validate the mechanism.
For the Beta phase, detection will become active, with automatic fallback to etcd if
inconsistency is detected. This way we automaticaly restore the previous behavior.

The implementation works as follows. Every 5 minutes, for each resource,
a hash calculation is performed. To avoid concurrent calculations,
the start time for each resource's calculation is randomly offset by 1 to 5 minutes.
A non-consistent `LIST` request (`RV=0`) is sent to the watch cache to retrieve its latest available RV.
This revision is then used to make a consistent `LIST` request (`RV=X`, where X is the revision from the cache) to etcd.
This ensures comparison of the cache's latest state with the corresponding state in etcd,
without explicit handling of potential cache staleness.

The 64-bit FNV algorithm (as implemented in [`hash/fnv`]([https://pkg.go.dev/hash/fnv](https://pkg.go.dev/hash/fnv)))
is used to calculate the hash over the entire structure of the `LIST` response,
using a technique similar to [gohugoio/hashstructure](https://github.com/gohugoio/hashstructure).
While calculating the hash of the entire structure is computationally more expensive,
the infrequency of this operation (every 5 minutes) makes the cost acceptable compared
to frequent `LIST` operations directly against etcd.
Hashing the entire structure helps prevent issues arising from object versioning differences.

The metric includes labels for `resource` (e.g., "pods"), `storage` (either "etcd" or "cache"), and `hash` (the calculated hash value). Example:
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

- Add tests for LIST with pagination and providing exact RV.

##### Unit tests

- `k8s/apiserver/pkg/storage/cache`: `2024-12-12` - `<test coverage>`

##### Integration tests

We should add a test to validate fallback to serving from etcd.

##### e2e tests

We should add a tests that validates metrics exposed for inconsistency detection.
Test should cover couple of resources including resources with conversion.

### Graduation Criteria

#### Alpha

- Snapshotting implemented behind a feature gate disabled by default.
- Inconsistency detection is implemented behind a feature gate enabled by default.

#### Beta

- Inconsistency detection mechanism is qualified and no mismatch detected.
- Fallback to etcd mechanism is implemented

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

```
feature-gates:
  - name: DetectCacheInconsistency
    components:
      - kube-apiserver
  - name: ListFromCacheSnapshot
    components:
      - kube-apiserver
```

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

Yes, we are adding `apiserver_storage_hash` to check cache consistency.

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
