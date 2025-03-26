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
    - [The algorithm](#the-algorithm)
  - [Bug in etcd progress notification](#bug-in-etcd-progress-notification)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Performance](#performance)
  - [What if the watch cache is stale?](#what-if-the-watch-cache-is-stale)
- [Design Details](#design-details)
  - [Pagination](#pagination)
    - [Option: Serve 1st page of paginated requests from the watch cache](#option-serve-1st-page-of-paginated-requests-from-the-watch-cache)
    - [Future work: Enable pagination in the watch cache](#future-work-enable-pagination-in-the-watch-cache)
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
- [Alternatives](#alternatives)
  - [Per-request override](#per-request-override)
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

To serve a consistent view from the watch cache, we first need to make a
consistent (quorum) read. We can use the getCurrentResourceVersionFromStorage
function, which was added as part of the [Watch-List KEP]. This function makes a
range quorum request that is expected to return no data, but will return the
latest resource version. The acquired resource version can be passed to the
`waitUntilFreshAndBlock` function to wait until the watch cache is ready to
serve.

Just waiting for a watch event with a fresh enough resource version would be
enough if the watch was observing all changes in etcd. However, the apiserver
establishes a separate watch for each resource. For resources with infrequent
changes, there is no guarantee that a watch event will be delivered at all.

To handle this case, we propose that the watch cache "request progress" of the
watch. This is an etcd feature added in v3.4, that is not yet used by
Kubernetes. It allows clients to get an immediate "progress notification" event
on watch. A progress notification informs the client about the progress of the
watch, even if there have been no updates to the watched keys. For support of
Kubernetes clusters using etcd v3.3, see "What if the watch cache is stale?"
section.

A single request is not enough to ensure that the wait on resource version
terminates. If etcd receives a burst of events, the watch might be in the middle
of processing a large batch of events when the watch cache requests progress.
The resulting progress notification will return a resource version earlier than
requested, which could cause an infinite wait if there are no follow-up events.

To rely on progress notification for freshness, the watch cache will
periodically (every 100ms) request progress until it reaches the requested
revision or the request times out.

#### The algorithm

When a consistent LIST request is received and the watch cache is enabled:

- Get the current revision from etcd for the resource type being served.
- Check if the watch cache already has the current revision, if not:
  - Add a "waiting read" and notify the goroutine running in background.
  - Wait for watch cache to catch up. If wait times out, reject the request.
    (see "What if the watch cache is stale?" section for details)
  - Remove a "waiting read"
- Serve the request from watch cache

In background, run a dedicated goroutine that will:
- Wait for changes to "waiting read" count.
- Repeat as long there is at least one "waiting read":
  - Send `WatchProgressRequest` request to etcd.
  - Wait 100ms

Consistent GET requests will continue to be served directly from etcd. We will
only serve consistent LIST requests from cache.

[Watch-List KEP]: /keps/sig-api-machinery/3157-watch-list

### Bug in etcd progress notification

Only recently community discovered a bug [etcd-io/etcd#15220] for requesting
progress notification. This bug causes a race between sending an event and
progress notification with the same revision. Normally we would expect to event
to always come first, however due to the bug, progress notification might be
sent first. Serving a consistent lists in that situation could result in
returning too early and missing the event. The bug was only fixed in v3.4.25 and
v3.5.8 (both 9 months old), which is too fresh for Kubernetes to not handle.

Our tests have shown that with the bug manual progress notification cannot be
trusted at all. It causes a silent corruption that cannot be automatically
detected prior to acting upon the corrupted data. For Beta, we propose to
conditionally enable the feature in a way that mitigates the risk of unaware
users stumbling on the bug.

We propose to implement a safeguard that will prevent enabling
`ConsistentListFromCache` on etcd version with broken progress notification.
During kube-apiserver startup we will verify etcd version by calling `Status`
method and checking etcd patch version. The condition will only pass if all etcd
endpoints have etcd version that includes the fix for [etcd-io/etcd#15220].

The proposed behavior of the safeguard for Beta:
* etcd version could not be acquired or etcd new enough - warn, but continue with whatever the current value of the feature gate is
* etcd too old, feature gate not explicitly specified - warn and disable the feature
* etcd too old, feature gate explicitly enabled - abort
* etcd new enough - use whatever the current value of the feature gate is

The proposed behavior of the safeguard for GA (feature gate locked to true):
* etcd version could not be acquired - warn, but continue with the feature enabled
* etcd too old - abort
* etcd new enough - enable feature

Checking etcd version during start should be a good enough solution as we
generally don't expect etcd clusters to downgrade. As of etcd v3.5 it is not
an officially supported feature.

[etcd-io/etcd#15220]: https://github.com/etcd-io/etcd/issues/15220

### Risks and Mitigations

### Performance

Progress notify is requested on client, so all watchers opened with this client
will get the notification. This is not a problem for Kubernetes as we maintain
one client per resource type, but could cause an issue if we attempt to reuse
the client across resource types (kubernetes/kubernetes#114458). This issue can
be mitigated by having multiple grpc watch streams within a single etcd client.

When etcd opens a watch it assigns it a stream based on its [context metadata].
So by default all watches opened on single client share single grpc stream.
To implement reusing etcd for multiple resources we should consider adding
unique metadata for each resource, forcing etcd client to create a separate
grpc stream for each of them. This way requesting progress notification on a
specific resource will result in only that single watch being notified.

[context metadata]: https://github.com/etcd-io/etcd/blob/a6ab774458411a6c0ea08f5df97e4dcc9a836345/client/v3/watch.go#L1070-L1075
[etcd-io/etcd#15220]: https://github.com/etcd-io/etcd/issues/15220

### What if the watch cache is stale?

This design requires wait for a watch cache to catch up to the needed revision.
There might be situation where watch is not providing updates causing watch cache to be permanently stale.
For example if watch stream is clogged, or when using etcd version v3.3 and older that don't provide progress notifications.

If the watch cache doesn't catch up in within some time limit we either fail the request or have a fallback.

If the fallback is to forward consistent reads to etcd, a cascading failure
is likely to occur if caches become stale and a large number of read requests
are forwarded to etcd.

Since falling back to etcd won't work, we should fail the requests and rely on
rate limiting to prevent cascading failure.  I.e. `Retry-After` HTTP header (for
well-behaved clients) and [Priority and Fairness](https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/20190228-priority-and-fairness.md).

In order to mitigate such problems, let's present how the system currently works
in different cases. In addition to that, we add column indicating whether a given
case will change how watchcache implementation will be handling the request.

| ResourceVersion | ResourceVersionMatch | Continuation      | Limit         | etcd implementation                     | watchcache implementation                          | changed                                                                               |
|-----------------|----------------------|-------------------|---------------|-----------------------------------------|----------------------------------------------------|---------------------------------------------------------------------------------------|
| _unset_         | _unset_              | _unset_           | _unset_       | Quorum read request                     | Delegated to etcd                                  | Yes, read etcd RV. Wait for cache synced to _RV_+ and list from cache                 |
| _unset_         | _unset_              | _unset_           | _N_           | Quorum read request                     | Delegated to etcd                                  | Yes, read etcd RV. Wait for cache synced to _RV_+ and list up to "N" items from cache |
| _unset_         | _unset_              | _token_           | _unset_ / _N_ | Read request from RV encoded in _token_ | Delegated to etcd                                  |                                                                                       |
| _unset_         | _Exact_              | _unset_ / _token_ | _unset_ / _N_ | Fails [validation]                      | Fails [validation]                                 |                                                                                       |
| _unset_         | _NotOlderThan_       | _unset_           | _unset_ / _N_ | Fails [validation]                      | Fails [validation]                                 |                                                                                       |
| _unset_         | _NotOlderThan_       | _token_           | _unset_ / _N_ | Fails [validation]                      | Fails [validation]                                 |                                                                                       |
| _0_             | _unset_              | _unset_           | _unset_ / _N_ | Quorum read request                     | List from cache ignoring _limit_                   |                                                                                       |
| _0_             | _unset_              | _token_           | _unset_ / _N_ | Quorum read request                     | Delegated to etcd                                  |                                                                                       |
| _0_             | _Exact_              | _unset_ / _token_ | _unset_ / _N_ | Fails [validation]                      | Fails [validation]                                 |                                                                                       |
| _0_             | _NotOlderThan_       | _unset_           | _unset_ / _N_ | Quorum read request                     | List from cache ignoring _limit_                   |                                                                                       |
| _0_             | _NotOlderThan_       | _token_           | _unset_ / _N_ | Read request from RV encoded in _token_ | Delegated to etcd                                  |                                                                                       |
| _RV_            | _unset_              | _unset_           | _unset_       | Quorum read request                     | Wait for cache synced to _RV_+ and list from cache |                                                                                       |
| _RV_            | _unset_              | _unset_           | _N_           | Read request from RV=_RV_               | Delegated to etcd                                  |                                                                                       |
| _RV_            | _unset_              | _token_           | _unset_ / _N_ | Read request from RV encoded in _token_ | Delegated to etcd                                  | Deferred                                                                              |
| _RV_            | _Exact_              | _unset_           | _unset_ / _N_ | Read request from RV=_RV_               | Delegated to etcd                                  |                                                                                       |
| _RV_            | _Exact_              | _token_           | _unset_ / _N_ | Fails [validation]                      | Fails [validation]                                 |                                                                                       |
| _RV_            | _NotOlderThan_       | _unset_           | _unset_       | Quorum read request + check for _RV_    | Wait for cache synced to _RV_+ and list from cache |                                                                                       |
| _RV_            | _NotOlderThan_       | _unset_           | _N_           | Quorum read request + check for _RV_    | Delegated to etcd                                  | Yes, wait for cache synced to _RV_+ and list up to "N" items from cache               |
| _RV_            | _NotOlderThan_       | _token_           | _unset_/ _N_  | Fails [validation]                      | Fails [validation]                                 |                                                                                       |

For watch requests both `Continuation` and `Limit` parameters are ignored (we should
have added validation rules for them in the past), but we have `SendInitialEvents` one.
The table for watch requests look like the following

| ResourceVersion | ResourceVersionMatch | SendInitialEvents      | etcd implementation                            | watchcache implementation               | changed  |
|-----------------|----------------------|------------------------|------------------------------------------------|-----------------------------------------|----------|
| _unset_         | _unset_              | _unset_                | Quorum list + watch stream                     | Delegate to etcd                        | Deferred |
| _unset_         | _unset_              | false / true           | Fails [validation]                             | Fails [validation]                      |          |
| _unset_         | _NotOlderThan_       | _unset_                | Fails [validation]                             | Fails [validation]                      |          |
| _unset_         | _NotOlderThan_       | false                  | Watch stream from etcd RV                      | Read etcd RV. Watch stream from it      |          |
| _unset_         | _NotOlderThan_       | true                   | Quorum list + watch stream                     | Wait RV > etcd RV. List + watch stream  |          |
| _unset_         | _Exact_              | _unset_ / false / true | Fails [validation]                             | Fails [validation]                      |          |
| _0_             | _unset_              | _unset_                | Quorum list + watch stream                     | List + watch stream                     |          |
| _0_             | _unset_              | false / true           | Fails [validation]                             | Fails [validation]                      |          |
| _0_             | _NotOlderThan_       | _unset_                | Fails [validation]                             | Fails [validation]                      |          |
| _0_             | _NotOlderThan_       | false                  | Watch stream from etcd RV                      | Watch stream from current watchcache RV |          |
| _0_             | _NotOlderThan_       | true                   | Quorum list + watch stream                     | List + watch stream                     |          |
| _0_             | _Exact_              | _unset_ / false / true | Fails [validation]                             | Fails [validation]                      |          |
| _RV_            | _unset_              | _unset_                | Watch stream from RV                           | Watch stream from RV                    |          |
| _RV_            | _unset_              | false / true           | Fails [validation]                             | Fails [validation]                      |          |
| _RV_            | _NotOlderThan_       | _unset_                | Fails [validation]                             | Fails [validation]                      |          |
| _RV_            | _NotOlderThan_       | false                  | Check RV > etcd RV. Watch stream from RV       | Watch stream from RV                    |          |
| _RV_            | _NotOlderThan_       | true                   | Check RV > etcd RV. Quorum list + watch stream | Wait for RV. List + watch stream        |          |
| _RV_            | _Exact_              | _unset_ / false / true | Fails [validation]                             | Fails [validation]                      |          |

[validation]: https://github.com/kubernetes/kubernetes/blob/release-1.30/staging/src/k8s.io/apimachinery/pkg/apis/meta/internalversion/validation/validation.go#L28
[etcd resolution]: https://github.com/kubernetes/kubernetes/blob/release-1.30/staging/src/k8s.io/apiserver/pkg/storage/etcd3/store.go#L589-L627

As presented in the above tables, the semantics for a given request server from
etcd and watchcache is a little bit different. It's a consequence of the fact that:
* etcd design supports only `Exact` semantics - it allows for consistent list
  from a given resource version (either specific value or "now").
  The semantics of `NotOlderThan` is implemented as getting consistent list from
  "now" and checking if it satisfies the condition.
* watchcache design supports only `NotOlderThan` semantics - it always waits
  until its resource version is at least as fresh as requested resource version
  and then returns the result from its current state

For the above reason, sending the same request to etcd and watchcache, especially
when cluster state is changing, may legitimately return different results.

In order to allow debugging results returned from watchcache in a runnning cluster,
the only reasonable procedure is:
* send a request that is served from watchcache
* send a request setting `ResourceVersionMatch=Exact` and `ResourceVersions` to value
  returned from the request returned in a previous point
* compare the two results

The existing API already allows us to achieve it.

To further allow debugging and improve confidence we will provide users with the
following tools:
* a dedicated `apiserver_watch_cache_read_wait` metric to detect a problem with
  watch cache.
* a `inconsistency detector` that for requests served from watchcache will be able
  to send a request to etcd (as described above) and compare the results

Metric `apiserver_watch_cache_read_wait` will measure wait time experienced by 
reads for watch cache to become fresh. If user notices a latency request in
they can use this metric to confirm that the issue is caused by watch cache.

The `inconsistency detector` will get enabled in our CI to detect issues with
the introduced mechanism.

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

Ongoing work to support pagination in watch cache: https://github.com/kubernetes/kubernetes/issues/108003

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

scalability tests verifying that introducing etcd progress notify events
don't degrade performance/scalability and verifying that there are substantial
benefits to enabling consistent reads from cache.

##### Unit tests

Unit test with a mock storage backend (instead of an actual etcd) that
various orderings of progress notify events and "current revision" response
result in the watch cache serving consistent read requests correctly

- `k8s.io/kubernetes/vendor/k8s.io/apiserver/pkg/storage/cacher`: `13.06.2023` - `84`
- `k8s.io/kubernetes/vendor/k8s.io/apiserver/pkg/storage/etcd3`: `12.06.2023` - `75`

##### Integration tests

##### e2e tests

Introduce e2e test that run both with etcd progress notify events enabled
and disable to ensure both configurations work correctly (both with this
feature enabled and disabled)

Benchmark consistent reads from cache against consistent reads.
Added performance tests to https://testgrid.k8s.io/sig-scalability-experiments, following 2 scenarios:
* Small objects - 300`000 configmaps each 1KB of size. 
* Large objects - 300 configmaps each 1MB of size.

In both cases we put load of 1 LIST per second with selector selecting no objects.

Comparing resource usage and latency with and without consistent list from watch cache enabled.
* 2-10 times reduction in CPU usage
* 20-50 times reduction of latency

|                          | Handled List requests [qps] | kube-apiserver CPU [cores] |        |        | etcd CPU [cores] |        |        | LIST latency [ms] |          |          |
|--------------------------| --------------------------- | -------------------------- | ------ | ------ | ---------------- | ------ | ------ | ----------------- | -------- | -------- |
|                          |                             | 50%ile                     | 90%ile | 99%ile | 50%ile           | 90%ile | 99%ile | 50%ile            | 90%ile   | 99%ile   |
| [Baseline]               | 0                           | 0.10                       | 0.11   | 0.12   | 0.18             | 0.19   | 0.19   | 25.00             | 45.00    | 49.50    |
| [Enabled Large Objects]  | 1                           | 0.09                       | 0.11   | 0.11   | 0.18             | 0.19   | 0.19   | 25.00             | 45.00    | 49.49    |
| [Disabled Large Objects] | 1                           | 3.13                       | 3.14   | 3.16   | 1.73             | 16.16  | 16.37  | 1438.49           | 1856.13  | 1985.61  |
| [Enabled Small Objects]  | 1                           | 0.63                       | 0.64   | 0.68   | 0.23             | 2.11   | 2.16   | 499.32            | 582.04   | 648.00   |
| [Disabled Small Objects] | 0.86                        | 6.92                       | 70.85  | 71.41  | 3.57             | 3.72   | 3.75   | 10493.83          | 17910.71 | 21800.00 |

[Baseline]: https://prow.k8s.io/view/gs/kubernetes-jenkins/logs/ci-kubernetes-e2e-gci-gce-scalability-consistent-list-from-cache-on-small-objects/1682379213627199488
[Enabled Large Objects]: https://prow.k8s.io/view/gs/kubernetes-jenkins/logs/ci-kubernetes-e2e-gci-gce-scalability-consistent-list-from-cache-on-large-objects/1682379213509758976
[Disabled Large Objects]: https://prow.k8s.io/view/gs/kubernetes-jenkins/logs/ci-kubernetes-e2e-gci-gce-scalability-consistent-list-from-cache-off-large-objects/1682741604768550912
[Enabled Small Objects]: https://prow.k8s.io/view/gs/kubernetes-jenkins/logs/ci-kubernetes-e2e-gci-gce-scalability-consistent-list-from-cache-on-small-objects/1682379213627199488
[Disabled Small Objects]: https://prow.k8s.io/view/gs/kubernetes-jenkins/logs/ci-kubernetes-e2e-gci-gce-scalability-consistent-list-from-cache-off-small-objects/1682741604877602816


### Graduation Criteria

#### Alpha

- Feature is implemented behind a feature gate
- Unpaginated LIST requests is served from watch cache
- First page of paginated requests is served from watch cache
- Feature performance is validated via scalability tests

#### Beta

- Feature is enabled by default.
- Metric `apiserver_watch_cache_read_wait` is implemented.
- Inconsistency detector is implemented and enabled in CI
- Deprecate support of etcd v3.3.X, v3.4.24 and v3.5.7

#### GA

- Drop support of etcd v3.3.X, v3.4.24 and v3.5.7
- Feedback is collected and addressed.

### Upgrade / Downgrade Strategy

N/A, kube-apiserver watch case is stateless.

### Version Skew Strategy

N/A, kube-apiserver watch case is stateless.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- Feature gate
  - Feature gate name: `ConsistentListFromCache`
  - Components depending on the feature gate: kube-apiserver

###### Does enabling the feature change any default behavior?

No, we only change implementation details of apiserver watch cache usage.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, by disabling the feature gate (given it's in-memory feature nothing else is needed).

###### What happens if we reenable the feature if it was previously rolled back?

No impact, new requests will be served from watch cache.

###### Are there any tests for feature enablement/disablement?

No changes in API types, based on the PRR instructions those tests are not
needed.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Not a workload feature, so it cannot impact running workload.
API servers with feature enabled will experience different latency and throughput of LIST requests.
If the apiserver watch cache is lagging it might cause a LIST requests to fail.

###### What specific metrics should inform a rollback?

Users should check for increase if their apiserver latency and
the `apiserver_watch_cache_read_wait` metric for direct impact of this feature to it.

We expect 99th percentile of `apiserver_watch_cache_read_wait` to be below 200ms (2 progress notify pull periods).

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No need for tests as this feature only change apiserver behavior without any perisstent side effects.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

We will want to deprecate support of etcd versions with broken progress
notifications (v3.3, v3.4.24, v3.5.7) in Kubernetes 1.30. During deprecation
user will receive warning about using deprecated etcd version.

Deprecation will be mostly informative as we don't expect to drop the support
until v3.4 and v3.4 minors are officially supported by etcd.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

No workload visible change. 

###### How can someone using this feature know that it is working for their instance?

Check if 99th percentile of `apiserver_watch_cache_read_wait` metric is below 200ms.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

https://github.com/kubernetes/community/blob/master/sig-scalability/slos/api_call_latency.md

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

https://github.com/kubernetes/community/blob/master/sig-scalability/slos/api_call_latency.md

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Watch latency metric, however it is not known how to measure it correctly.
Proposed metric `apiserver_watch_cache_read_wait` should be a good enough approximation of healthiness of this feature.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

Yes, etcd version needs to be at least v3.4.25+ or v3.5.8+.

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

Yes, it might increase latency of processing LIST requests.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

We expect that this feature will reduce resource usage of kube-apiserver and etcd.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

- Etcd watch stream starvation
  - Detection: Check out apiserver latency metric and `apiserver_watch_cache_read_wait` metric.
  - Mitigation: Disable the `ConsistentListFromCache` feature flag.
  - Diagnostics: 503 HTTP logs
  - Testing: Tests for watch cache timeout

###### What steps should be taken if SLOs are not being met to determine the problem?

Use `apiserver_watch_cache_read_wait` metric to check impact on latency.
Use per-request override to compare latency when reading from watch cache vs etcd.

## Implementation History

* 1.28 - Alpha
* 1.31 - Beta

## Alternatives

Do nothing:

- Leaves the "stale read" problem unsolved, although we have a PR fixing reflector relist which helps mitigate the larger issue.
- Does not impact scale or performance.

Allow clients to manage the initial resource version they provide to reflectors, but don’t implement this optimization:

- Many clients will most likely continue to use resourceVersion=”0” even if it violates their consistency needs
- Clients that transition to use resourceVersion=”” will pay a high scale/performance cost
- We don't expect clients to attempt to keep track of the last resourceVersion they observed. If they do attempt this, we are concerned that they might get it wrong and introduce subtle and difficult to debug issues as a result.

Do a dynamic fallback based on watch cache wait time.

- We expect watch being starved to happen very rarely, meaning its logic needs to be very simple to ensure it works properly.
- Simple fallback will rather not do a better job then just a manual fallback.

### Per-request override

To enable debugging, we considered introducing per-request override to disable
watchcache to force the request to be served from etcd. This would allow us
to compare request results without impacting other requests or requiring to
redeploy the whole cluster. However, as described in the KEP itself, the results
of the same requests served from watchcache and etcd may legitimately return
different results. As a result, the proposed debugging mechanism was decided
to better serve its purpose.

We also considered automatic fallback. However, we expect watch being
starved to happen very rarely, meaning its logic needs to be very simple to
ensure it works properly. A simple fallback will not bring much benefit over
what user can do manually. It will just make the harder to understand and
predict behavior. APF estimates cost just based on request parameters,
before it is passed to storage. If fallback was based on state of watch cache,
cost of request would change after the APF decision increasing the risk of overload.
