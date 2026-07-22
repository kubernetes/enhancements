# KEP-4988 Snapshottable API server cache

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Serving list from snapshots](#serving-list-from-snapshots)
  - [Watch cache compaction](#watch-cache-compaction)
  - [Cache Inconsistency Detection Mechanism](#cache-inconsistency-detection-mechanism)
- [Risks and Mitigations](#risks-and-mitigations)
    - [Snapshot memory overhead](#snapshot-memory-overhead)
    - [Consistency checking overhead](#consistency-checking-overhead)
- [Design Details](#design-details)
  - [Snapshotting algorithm](#snapshotting-algorithm)
    - [Hashing algorithm](#hashing-algorithm)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [x] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [x] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The kube-apiserver's caching mechanism (watchcache) efficiently serves requests
for the latest observed state. However, `LIST` requests for previous states
(e.g., via pagination or by specifying a `resourceVersion`) often bypass this
cache and are served directly from etcd. This direct etcd access significantly
increases performance costs and can lead to stability issues, particularly
with large resources, due to memory pressure from transferring large data blobs.

This KEP proposes an enhancement to the kube-apiserver's watch cache to
generate B-tree snapshots, allowing it to serve `LIST` requests for previous
states directly from the cache. This change aims to improve API server
performance and stability. To support this snapshotting mechanism,
this proposal also details changes to the watch cache's compaction behavior to maintain Kubernetes Conformance
and introduces an automatic cache inconsistency detection mechanism.

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

- Reduce memory allocations by serving historical LIST requests from cache
- Maintain Kubernetes conformance with regards to compaction
- Prevent inconsistent responses returned by cache due to bugs in caching logic

### Non-Goals

- Change semantics of the `LIST` request
- Support indexing when serving for all types of requests.
- Enforce that no client requests are served from etcd
- Support etcd server side compaction for watch cache
- Detection of watch cache memory corruption

## Proposal

We propose that the watch cache generate B-tree snapshots, allowing it to serve `LIST` requests for previous states.
These snapshots will be stored for the same duration as watch history and compacted using the same mechanisms.
This improves API server performance and stability by minimizing direct etcd access for historical data retrieval.
It also aligns with the future extensions outlined in [KEP-365: Paginated Lists].

Compaction is an important behavior, covered by Kubernetes Conformance tests.
Supporting compaction is required to ensure consistent behavior regardless of whether the watch cache is enabled or disabled.
Storing historical data in the watch cache, as this KEP proposes, breaks conformance.
Currently, watch cache is only compacted when it becomes full.
For resources with infrequent changes, this means data could be retained indefinitely,
far beyond etcd's compaction point, as highlighted in [#131011].
Therefore, to maintain conformance and ensure predictable behavior,
we propose that the existing etcd compaction mechanism also be responsible for compacting the snapshots in cache.

This proposal increases reliance on the watchcache, significantly elevating the impact of bugs in watch or caching logic.
Triggering a bug would no longer impact a single client but affect the cache read by all clients connecting to a particular API server.
As the proposed changes will result in all requests being served from the cache,
it would be exceptionally difficult to debug errors, as comparing responses to etcd would no longer be an option.
Consequently, we propose an automatic cache inconsistency detection mechanism that can run in production and replace manual debugging.
It will automate checking consistency against etcd, protecting against bugs in the watch cache or etcd watch implementation.
It is important to note that we do not plan to implement protection from memory corruption like bitflips.

[KEP-365: Paginated Lists]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/365-paginated-lists#potential-future-extensions
[#131011]: https://github.com/kubernetes/kubernetes/issues/131011#issuecomment-2747497808

### Serving list from snapshots

The snapshotting mechanism utilizes ability of B-tree to create
lazy copies of itself. This allows us to create snapshot on each watch event.
Those snapshots capture the state of cache at historical resourceVersion,
and can be used to serve `LIST` requests, by finding aprioripate snapshot and just reading from it.

### Watch cache compaction

We will expand the existing mechanism for compacting etcd to also compact the watch cache.
Kubernetes supports periodic configuring compaction by default executed every 5 minutes.
In the current algorithm each API Server executes a optimistic write on `compact_rev_key` key to store revision to be compacted.
The one that is first to write successfully, executes the compaction request against etcd.
We will expand it by opening a watch on `compact_rev_key` key, and informing watch cache about succesfull compactions done by any API server.
When watch cache is informed about compaction, it will truncate snapshot history up to that revision.
To avoid changes of existing behavior, we will not compact watch history; this should be considered in the future.

### Cache Inconsistency Detection Mechanism

The mechanism periodically calculates and compares a hash of the data for each resource in both the etcd and the watch cache.

It will be developed across multiple phases:
*  **Alpha:** In this phase, the detection will enabled only in the test environment.
  Enabled via `KUBE_WATCHCACHE_CONSISTANCY_CHECKER` environment variable,
  we will run in Kubernetes e2e tests to ensure that the mechanism works as expected.
  On mismatch the apiserver will panic making it easy to detect in tests.
*  **Beta:** The detection will be enabled by default. If an inconsistency is detected,
  snapshots stored in cache will be purged and the system will automatically fall
  back to serving LIST requests from etcd for the affected resource.
  This mechanism will only impact LIST requests that would be served from watch cache snapshots,
  effectively reverting to the behavior prior to this proposal,
  while other requests will continue to be served from the cache.
  Fallback will not be permanent, but will last until the next successful consistency check.

To monitor consistency failures we will expose `storage_consistency_checks_total` metric.

## Risks and Mitigations

#### Snapshot memory overhead

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

#### Consistency checking overhead

Periodic execution of consistency checking will introduce additional overhead.
This load is not negligible, as it requires downloading and decoding data from etcd.
For safety we still think it's important that feature is enabled by default,
however we want to leave an option to disable it.
For that we will introduce `DetectCacheInconsistency` feature gate in Beta.

For future we plan to improve etcd API to support cheap consistency checks.
At that point disabling inconsistency checks will no longer be needed.

## Design Details

### Snapshotting algorithm

1. **Snapshot Creation:** When a watch event is received, the cacher creates
   a snapshot of the B-tree based cache using the efficient [Clone()] method.
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

[Clone()]: https://pkg.go.dev/github.com/google/btree#BTree.Clone

#### Hashing algorithm

Every 5 minutes, for each resource, we calculate hash for each resource.
A non-consistent `LIST` request (`RV=0`) is sent to the watch cache to retrieve its latest available RV.
This revision is then used to make a consistent `LIST` request (`RV=X`, where X is the revision from the cache) to etcd.
This ensures comparison of the cache's latest state with the corresponding state in etcd,
without explicit handling of potential cache staleness.

The 64-bit FNV algorithm (as implemented in [`hash/fnv`]([https://pkg.go.dev/hash/fnv](https://pkg.go.dev/hash/fnv)))
is used to calculate the hash of object's namespace, name, and resourceVersion joined by a '/' byte.
This should allow us to detect inconsistencies caused by bugs in applying watch events or bugs in etcd watch stream.

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
- Inconsistency detection is behind environment variable
- Inconsistency detection run in e2e tests

#### Beta

- Inconsistency detection mechanism is qualified and no mismatch detected.
- Inconsistency detection moved behind a feature gate `DetectCacheInconsistency` enabled by default.
- Automatic fallback to etcd is implemented
- Pass Kubernetes conformance tests for compaction

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

Snapshotting should automatically fallback to serving from etcd if inconsistency is detected.
Rollback should be consider if there is a high number of inconsistencies detected by `storage_consistency_checks_total` metric.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No need for tests, this feature doesn't cause any persistent side effects.

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

Yes, we are adding `storage_consistency_checks_total` to count the number of consistency checks performed and their outcomes.

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
For the first iteration we will enable users to define an alert on a metric and detect if cache becomes inconsistent with etcd.

###### What steps should be taken if SLOs are not being met to determine the problem?

Disabling the feature-gate.

## Implementation History

- 1.33: Alpha
- 1.34: Beta

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
