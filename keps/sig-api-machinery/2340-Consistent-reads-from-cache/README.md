# Consistent Reads from Cache

Kubernetes Get and List requests are guaranteed to be "consistent reads" if the
`resourceVersion` parameter is not provided. Consistent reads are served from
etcd using a "quorum read".

But often the watch cache contains sufficiently up-to-date data to serve the
read request, and could serve it far more efficiently.

This KEP proposes a mechanism to serve most reads from the watch cache
while still providing the same consistency guarantees as serving the
read from etcd.

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Consistent reads from cache](#consistent-reads-from-cache)
    - [Use RequestProgress to enable automatic watch updates](#use-requestprogress-to-enable-automatic-watch-updates)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Dependency on manual watch progress notifications](#dependency-on-manual-watch-progress-notifications)
  - [What if the watch cache is stale?](#what-if-the-watch-cache-is-stale)
- [Design Details](#design-details)
  - [Pagination](#pagination)
    - [Option: Serve 1st page of paginated requests from the watch cache](#option-serve-1st-page-of-paginated-requests-from-the-watch-cache)
    - [Future work: Enable pagination in the watch cache](#future-work-enable-pagination-in-the-watch-cache)
  - [Test Plan](#test-plan)
      - [Correctness Tests](#correctness-tests)
      - [Performance Tests](#performance-tests)
      - [Scalability Tests](#scalability-tests)
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
- [Alternatives](#alternatives)
- [Potential Future Improvements](#potential-future-improvements)
<!-- /toc -->

## Summary

Consistent reads may be served from cache so long as:
- A consistent (quorum) read is first made to etcd to get the latest "revision"
- The data in the watch cache no older than the latest "revision" just from etcd

etcd watches support "progress events", which provide an updated revision and a
guarantee that all future watch events will be newer than the that revision. 
Etcd client can request a progress notification from server. The progress 
notification allow the etcd watcher to know how up-to-date the watch stream 
is. This is thanks to [bookmarkable] property of etcd watch that guarantees that
all events with revision below progress notification have been delivered.

This KEP summarizes how we can take advantage of progress events efficiently
determine how up-to-date kubernetes watch caches are then serve reads from the
watch cache when they are sufficiently up-to-date.

[bookmarkable]: https://etcd.io/docs/v3.6/learning/api_guarantees/#watch-apis

## Motivation

Serving reads from the watch cache is more performant and scalable than reading
them from etcd, deserializing them, applying selectors, converting them to the
desired version and then garbage collecting all the objects that were allocated
during the whole process.

We will need to measure the impact to performance and scalability, but we have
enough data and experience from prior work with the watch cache to be confident
there is significant scale/perf opportunity here, and we would like to introduce
an alpha implementation.

We expect the biggest gain to be from node-originating requests (e.g. kubelet
listing pods scheduled on its node). For those requests, the size of the
response is small (it fits a single page, assuming you won't make it extremely
small), whereas the number of objects to process is proportional to cluster-size
(so fairly big). For example, when kubelets requests pods schedule against it in
a 5k node cluster with 30pods/node, the kube-apiserver must list the 150k pods
from etcd and then filter that list down to the list of 30 pods that the kubelet
actually need. This must occur for each list request from each of the 5k
kubelets. If served from watch cache, this same request can be served from
built-in index filtering out the 30 pods each kubelet needs from the data in the
cache.

In addition to the improvements to scale and performance, we aim to resolve a
specific problem. The long standing "stale read" issue
(https://github.com/kubernetes/kubernetes/issues/59848) remains open because
reflectors default to resourceVersion=”0” for their initial list requests. If
the reflectors instead use a consistent read for their initial list request,
they could not "going back in time" when components are restarted and this issue
would be solved. "Going back in time" can currently happen if the initial list
request is served from a stale watch cache with data much older than the
reflector has previously observed or if the api-server or etcd are partitioned.

We have held off on switching reflectors to using consistent read for the
initial list, even though we know it is more correct, due to concerns with the
impact on large scale use cases. But if we serve consistent reads from cache,
there would be very little difference in scalability to how the kube-apiserver
serves the resourceVersion="0" list requests from reflectors today.

### Goals

- Resolve the "stale read" problem (https://github.com/kubernetes/kubernetes/issues/59848)
- Improve the scalability and performance of Kubernetes for Get and List requests, when the watch cache is enabled

### Non-Goals

- Remove all true quorum reads.
- Serving pagination continuation from watch cache.

## Proposal

### Consistent reads from cache

Guard this by a `WatchCacheConsistentReads` feature gate.

This requires using `WatchProgressRequest` which is only available in etcd 3.4+, and so would
require we make the kube-apiserver aware of etcd's minor version, which is described in more detail later.


#### Use RequestProgress to enable automatic watch updates

When a consistent LIST request is received and the watch cache is enabled:

- Get the current revision from etcd for the resource type being served.
  Use the [getCurrentResourceVersionFromStorage] added as part of [Watch-List KEP].
- If the cache already has the current revision, serve the request from cache. If not,
  - Send a `WatchProgressRequest` to etcd on the watch channel that the watch cache is consuming.
- Use the existing `waitUntilFreshAndBlock` function in the watch cache to wait briefly for the watch to catch up to the current revision.
- If the block times out, the request will result in rejection. (see "What if the watch cache is stale?" section for details)

Consistent GET requests will continue to be served directly from etcd. We will
only serve consistent LIST requests from cache.

As a consequence of this number of etcd progress notification will increase
number. [Watch Bookmarks KEP] utilized etcd periodic progress notifications to
notify kube-apiserver clients of their progress. It used periodic watch
notifications that are naturally limited on etcd side. To maintain the same
behavior we will limit watch bookmarks to send them only one per 5 seconds.

[getCurrentResourceVersionFromStorage]: https://github.com/kubernetes/kubernetes/blob/3f247e59edfd4083242ad7271d076a38291760ff/staging/src/k8s.io/apiserver/pkg/storage/cacher/cacher.go#L1246-L1278
[Watch-List KEP]: /keps/sig-api-machinery/3157-watch-list
[Watch Bookmarks KEP]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/956-watch-bookmark

### Risks and Mitigations

### Dependency on manual watch progress notifications

<<[UNRESOLVED @serathius]>>
Validate performance implications of using `WatchProgressRequest`.
<<[/UNRESOLVED]>>

<<[UNRESOLVED @serathius]>>
Propose how kube-apiserver should fallback if etcd watch notification doesn't work:
* Flag providing etcd version to kube-apiserver `--storage-backend=etcd3.4`
* Flag to enable/disable the feature `--allow-using-progress-notify`
* Have kube-apiserver check cluster version in etcd `/version` endpoint.
  Retry the check logic if `WatchProgressRequest` fails.
* Fallback to reading from etcd if no progress notification within `X` seconds.
  <<[/UNRESOLVED]>>

### What if the watch cache is stale?

This design requires wait for a watch cache to catch up to the needed revision
for consistent reads. If the cache doesn't catch up within some time limit we
either fail the request for have a fallback.

If the fallback it to forward consistent reads to etcd, a cascading failure
is likely to occur if caches become stale and a large number of read requests
are forwarded to etcd.

Since falling back to etcd won't work, we should fail the requests and rely on
rate limiting to prevent cascading failure.  I.e. `Retry-After` HTTP header (for
well behaved clients) and [Priority and Fairness](https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/20190228-priority-and-fairness.md).

## Design Details

### Pagination

Given that the watch cache does not paginate responses, how can clients requesting
pagination for resourceVersion="" reads be supported?

#### Option: Serve 1st page of paginated requests from the watch cache

Only serve the 1st page of paginated requests from the watch cache. The watch
cache would need to construct the appropriate continuation token such that the
subsequent pages can be served from etcd.

An even more conservative approach would be to only serve paginated requests
that fit within a single page from the watch cache, in which cache the watch
cache doesn't need to construct continuation tokens at all.

In practice, this options might be sufficient to get the bulk of the scalability
benefits of serving consistent reads from cache. For example, the kubelet LIST
pods use case would be handled, as would similar cases. Not all cases would
be handled.

#### Future work: Enable pagination in the watch cache

The problem is that the watch cache ("isn't able to perform
continuations")[https://github.com/kubernetes/kubernetes/blob/789dc873f6816cc2b9b39e77a9b94f478d3a3134/staging/src/k8s.io/apiserver/pkg/storage/cacher/cacher.go#L595].
The watch cache is designed to only serve LIST requests at [the latest resource version is has
available](https://github.com/kubernetes/kubernetes/blob/789dc873f6816cc2b9b39e77a9b94f478d3a3134/staging/src/k8s.io/apiserver/pkg/storage/cacher/watch_cache.go#L115).

To support watch cache pagination:
- The watch cache would to keep a comparable resource version history to the
  default etcd compaction history of 5 minutes.
- The watch cache would need to be resturctured so that it can serve LIST for
  the resource versions it has returned continuation tokens to clients for.

Both of these are major changes, and would require scalability validation.

Potential approach:

- Watch cache is getting LIST request with pagination
- List everything from the internal cache and have pointers to those objects
- Return first LIMIT of those, and in the internal map store the remaining ones indexed by "continuation token" that we just generated
- GC the items from this map after N seconds from insertion
- Continuation is set in the request, we lookup that map and return next LIMIT items, if the item doesn't exist in the map we (either fallback to etcd or return an error - probably the former)

Memory would need to be somehow bound with this approach.

### Test Plan

##### Correctness Tests

- Verify that we don't violate the linerizability guranentees of consistent reads:
  - Unit test with a mock storage backend (instead of an actual etcd) that
    various orderings of progress notify events and "current revision" response
    result in the watch cache serving consistent read requests correctly
  - Soak test to ensure that consistent reads always return data at resource
    versions no older that previous writes occurred at. In either e2e tests,
    scalability tests or a dedicated tester that we run for an extended
    duration, we can add a checker that periodically performs writes and
    consistent reads and ensure the read resource versions are not older than
    the resource versions of the writes.
  - Introduce e2e test that run both with etcd progress notify events enabled
    and disable to ensure both configurations work correctly (both with this
    feature enabled and disabled)

##### Performance Tests

- Benchmark consistent reads from cache against consistent reads from etcd for:
  - list result sizes of 1, 10, ..., 100000
  - object sizes of 5kb, 25kb, 100kb
  - measure latency and throughput
  - document results in this KEP

##### Scalability Tests

- 5k scalability tests verifying that introducing etcd progress notify events
  don't degrade performance/scailability (early results available here:
  https://github.com/kubernetes/kubernetes/pull/86769)
- 5k scalability tests verifying that there are substantial scalability benefits
  to enabling consistent reads from cache for the pod list from kubelet use case
  - Latency output contains what we need to ensure the impact to latency of
    delaying consistent reads for the progress notify interval is what we expect
    (~250ms more latency for these requests on average)
  - Scalability output contains what we need to ensure we are within SLOs and
    our scalability goals
  - Since pod list requests were previously served from the watch cache (but
    without a consistency guarantee), we expect scalability to be roughly the
    same as baseline (but with the benefit of improved correctness)

### Graduation Criteria

#### Alpha

- Feature is implemented behind a feature gate
- Unpaginated LIST requests is served from watch cache
- First page of paginated requests is served from watch cache

#### Beta


#### GA

- Feature is enabled by default

### Upgrade / Downgrade Strategy

N/A, kube-apiserver watch case is stateless.

### Version Skew Strategy

N/A, kube-apiserver watch case is stateless.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- Feature gate
  - Feature gate name: `WatchCacheConsistentReads`
  - Components depending on the feature gate: kube-apiserver

###### Does enabling the feature change any default behavior?

No, we only change implementation details of apiserver watch cache usage.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes

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

- [ ] Events
  - Event Reason:
- [ ] API .status
  - Condition name:
  - Other field:
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Use existing kube-apiserver SLOs. 

TODO: Provide link

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name: TODO: provide exact name of apiserver latency metric
  - [Optional] Aggregation method:
  - Components exposing the metric:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Watch latency metric.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

N/A

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

Yes, it might increase latency of processing non-streaming read-only API.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

We expect that this feature will reduce resource usage of kube-apiserver and etcd.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

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

* 1.28 - Move to implementable.

## Alternatives

Do nothing:

- Leaves the "stale read" problem unsolved, although we have a PR fixing reflector relist which helps mitigate the larger issue.
- Does not impact scale or performance.

Allow clients to manage the initial resource version they provide to reflectors, but don’t implement this optimization:

- Many clients will most likely continue to use resourceVersion=”0” even if it violates their consistency needs
- Clients that transition to use resourceVersion=”” will pay a high scale/performance cost
- We don't expect clients to attempt to keep track of the last resourceVersion they observed. If they do attempt this, we are concerned that they might get it wrong and introduce subtle and difficult to debug issues as a result.


## Potential Future Improvements

Modify etcd to allow echo back a user provided ID in progress events.

- Client generates a UUID and provides to the ProgressNotify request
- Once client sees a progress event with the same UUID, it knows the watch is up-to-date
- This reduces the worst case number of round trips required to do a consistent read from two to one since client doesn't need to get the lastest revision from etcd first

Potential optimiation: We could delay requests, accumulate multiple in-flight
read requests over some short time period, and at the end of the period, get the
current revision from etcd, wait for the watch cache to catch up, and then serve
all the the in-flight reads from cache. This would reduce the number of "get
current revision" requests that need to be made to etcd in exchange for higher
request latency (but only for consistent reads). A simple implementation would
be to do this on a fix interval, where, obviously, if there were not reqests
during the period, we don't bother to fetch a current revision from etcd. It
is unclear if this will result in actual gain, and it would complicate the
code, so should be explored with care.
