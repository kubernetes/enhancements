---
title: Consistent Reads from Cache
authors:
  - "@jpbetz"
  - "@wojtek-t"
owning-sig: sig-api-machinery
participating-sigs:
  - sig-scalability
reviewers:
  - "@wojtek-t"
  - "@jingyih"
approvers:
  - "@lavalamp"
  - "@deads2k"
  - "@wojtek-t"
editor: TBD
creation-date: 2019-12-10
last-updated: 2020-02-19
status: provisional
see-also:
replaces:
superseded-by:
---

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
  - [Consistent reads from cache](#consistent-reads-from-cache-1)
    - [Use WithProgressNotify to enable automatic watch updates](#use-withprogressnotify-to-enable-automatic-watch-updates)
    - [Determining if etcd is sending progress notify events](#determining-if-etcd-is-sending-progress-notify-events)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Progress notify interval selection](#progress-notify-interval-selection)
  - [Pagination](#pagination)
    - [Option: Continue to serve all paginated requests from etcd](#option-continue-to-serve-all-paginated-requests-from-etcd)
    - [Option: Serve 1st page of paginated requests from the watch cache](#option-serve-1st-page-of-paginated-requests-from-the-watch-cache)
    - [Option: Enable pagination in the watch cache](#option-enable-pagination-in-the-watch-cache)
    - [Rejected Option: Return unpaginated responses to paginated list requests](#rejected-option-return-unpaginated-responses-to-paginated-list-requests)
  - [What if the watch cache is stale?](#what-if-the-watch-cache-is-stale)
  - [Ability to Opt-out](#ability-to-opt-out)
  - [Test Plan](#test-plan)
  - [Rollout Plan](#rollout-plan)
    - [Serving consistent reads from cache](#serving-consistent-reads-from-cache)
    - [Reflectors](#reflectors)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
- [Rejected alternatives](#rejected-alternatives)
    - [Use WatchProgressRequest to request watch updates when needed](#use-watchprogressrequest-to-request-watch-updates-when-needed)
- [Potential Future Improvements](#potential-future-improvements)
<!-- /toc -->

## Summary

Consistent reads may be served from cache so long as:
- A consistent (quorum) read is first made to etcd to get the latest "revision"
- The data in the watch cache no older than the latest "revision" just from etcd

etcd watches support "progress events", which provide an updated revision and a
guarantee that all future watch events will be newer than the that revision.  If
an etcd watch is configured with `WithProgressNotify` enabled, etcd
automatically sends progress events at a regular interval. The "progress events"
allow a etcd watcher to know how up-to-date the watch stream is relative a
particular revision.

This KEP summarizes how we can take advantage of progress events efficiently
determine how up-to-date kubernetes watch caches are then serve reads from the
watch cache when they are sufficiently up-to-date.

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
kubelets. If served from watch cache, this same request can be served by simply
filtering out the 30 pods each kubelet needs from the data in the cache.

In addition to the improvements to scale and performance, we aim to resolve a
specific problem. The long standing "stale read" issue
(https://github.com/kubernetes/kubernetes/issues/59848) remains open because
reflectors default to resourceVersion=”0” for their initial list requests. If
the reflectors instead use a consistent read for their initial list request,
they could not "going back in time" when components are restarted and this issue
would be solved. "Going back in time" can curently happen if the initial list
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

<<[UNRESOLVED @deads]>>
- Avoid allowing true quorum reads. We should think carefully about this, see: https://github.com/kubernetes/enhancements/pull/1404#discussion_r381528406
<<[/UNRESOLVED]>>

## Proposal

### Consistent reads from cache

Guard this by a `WatchCacheConsistentReads` feature gate.

#### Use WithProgressNotify to enable automatic watch updates

Create etcd watches with `WithProgressNotify` enabled (available in all etcd 3.x versions).

When `WithProgressNotify` is enabled on an etcd watch, etcd sends progress
events to the watch automatically. By default etcd sends progress events every
10 minutes, which is not frequent enough to be useful for our needs, so we will
modify etcd to send them more frequently.

When an consistent LIST request is received and the watch cache is enabled:

- Get the current revision from etcd for the resource type being served. The returned revision is strongly consistent (guaranteed to be the latest revision via a quorum read).
- Use the existing `waitUntilFreshAndBlock` function in the watch cache to wait briefly for the watch to catch up to the current revision.
- If the block times out, the request will result in rejection. (see "What if the watch cache is stale?" section for details)

To get the revsion we have some options: 

- Use an etcd range request with `WithCount` enabled so etcd return only a count and revision 
- Use an etcd range request against a known empty range with limit=1 as an additional guard (since etcd does not allow for limit=0)

Consistent GET requests will continue to be served directly from etcd. We will
only serve consistent LIST requests from cache.

Important: We are planning to set the progress notify interval to 250ms, which will introduce up to 250ms latency to consistent LIST requests.

Optional: For some (but not all) of the etcd progress watch events, also create a
kubernetes "bookmark" watch event and send it to kube-apiserver clients so that
reflectors and shared informers are kept up-to-date. The benefit of this is that
it minimizes the chance that these clients will end up with an out-of-date
resource version and need to relist (which can impact scalability). See [Watch
Bookmarks](https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/20190206-watch-bookmark.md)
for details.

#### Determining if etcd is sending progress notify events

It is possible to automatically determine if etcd is sending progress notify events.
Each watch cache could keep track of when it received the last progress notify event.
If it has been sufficiently long since the last one was received, or if none have
ever been received, the watch cache should assume etcd is not sending progress notify,
and not attempt to serve consistent reads from cache, falling back to serving
the reads directly from etcd.

### Risks and Mitigations

Configuring etcd to send progress notify events may have performance implications.
@mm4tt and @wojtek-t have run [some
experiments](https://github.com/kubernetes/kubernetes/pull/86769) that suggest
that a progress notify interval of 50ms results in a noticeable increase in both
etcd and kube-apiserver CPU utilization and some increased kube-apiserver serving
latency. We intend to address this by first trying a 250ms progress notify interval,
if ths performs well we will use that as our interval. If not, we will dig in
to figure out why the events are impacting performance and see if we can optimize
it away.

## Design Details

### Progress notify interval selection

Not all LIST requests require consistency, and we're taking the view that if you
really want to have a consistent LIST (we should explicitly exclude GETs from
it), then you may need to pay additional tax latency for it. For this reason, we
intend to start with a 250ms progress notify interval, which will on average 125ms
latency to each consistent LIST request.

The requests this is expected to impact are:

- Reflector list/relist requests, which occur at startup and after a reflector
  falls to far behind processing events (e.g. it was partitioned or resource starved)
- Controllers that directly perform consistent LIST requests

In all cases, increasing latency in exchange for higher overall system
throughput seems a good trade off. Use cases that need low latency have multiple
options: Watching resources with shared informers or reflectors, LIST requests
with a minimum resource version specified.

During our testing of this feature we will gather more data about the impact of
selecting 250ms for the progress notify interval. We will also use alpha to
gather feedback from the community on the latency impact.

### Pagination

Given that the watch cache does not paginate responses, how can clients requesting
pagination for resourceVersion="" reads be supported?

From the below options, we are currently favoring the "Continue to serve all
paginated requests from etcd" option for alpha since this would not disrupt clients,
and would still allow us to experiment with enabling consistent reads from cache
selectively where we believe it will have the most impact.

Later, we could transition to "Serve 1st page of paginated requests from the watch cache"
which would expand cache usage to a much larger proportion of all consistent read requests.

<<[UNRESOLVED]>>
That kubectl makes paginated, so if we enable this feature for paginated requests,
which may add latency to `kubectl get`. We need to be clear on the behavior.
<<[/UNRESOLVED]>>

#### Option: Continue to serve all paginated requests from etcd

Only start serving unpaginated LIST requests with resourceVersion="" from cache. Clients
using pagination would be unaffected.

The complication with this option is that, by default, reflectors paginate when
they list/relist so their list requests would miss the cache. We could address
this by configuring reflectors that would benefit most from this feature, like
the pod list requests from kubelet pod, to not paginate.

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

#### Option: Enable pagination in the watch cache

The problem is that the watch cache ("isn't able to perform
continuations")[https://github.com/kubernetes/kubernetes/blob/789dc873f6816cc2b9b39e77a9b94f478d3a3134/staging/src/k8s.io/apiserver/pkg/storage/cacher/cacher.go#L595].
The watch cache is designed to only serve LIST requests at [the latest resource version is has
available](https://github.com/kubernetes/kubernetes/blob/789dc873f6816cc2b9b39e77a9b94f478d3a3134/staging/src/k8s.io/apiserver/pkg/storage/cacher/watch_cache.go#L115).

To supporting watch cache pagination:
- The watch cache would to keep a comparable resource version history to the
  default etcd compaction history of 5 minutes.
- The watch cache would need to be resturctured so that is can serve LIST for
  the resource versions it has returned continuation tokens to clients for.
  
Both of these are major changes, and would require scalability validation.

Potential approach:

- Watch cache is getting LIST request with pagination
- List everything from the internal cache and have pointers to those objects
- Return first LIMIT of those, and in the internal map store the remaining ones indexed by "continuation token" that we just generated
- GC the items from this map after N seconds from insertion
- Continuation is set in the request, we lookup that map and return next LIMIT items, if the item doesn't exist in the map we (either fallback to etcd or return an error - probably the former)

Memory would need to be somehow bound with this approach.

#### Rejected Option: Return unpaginated responses to paginated list requests

Both for backward compatibility with older versions of Kubernetes that do not support
pagination, and for compatibility with api-servers that have the watch cache enabled,
clients must be able to tolerate unpaginated responses for paginated LIST requests.

The kube-apiserver already switches between serving paginated responses and
serving unpaginated responses depending on if the watch cache is enabled for the
types requested. 

User cases that have disabled the watch cache will still receive paginated responses.

Use cases where paginated was being used to reduce load on etcd, but are able to
tolerate the data volume of the list being returned in a single response, should
not impacted.

The most likely problem with this approach is that, because LIST with
resourceVersion="" is the only way to paginate data from the api-server when the
watch-cache is enabled (which is the default), clients that need pagination
(e.g. to avoid receiving too much data in a single request) will be relying on
LIST resourceVersion="".

We are not planning to pursue this option.

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

### Ability to Opt-out

<<[UNRESOLVED @deads2k]>>
How to opt out of this behavior and still get a "normal" quorum read? We'll need this ability for our own debugging if nothing else.
See https://github.com/kubernetes/enhancements/pull/1404#issuecomment-588433911
<<[/UNRESOLVED]>>

### Test Plan

Correctness:

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

Performance:

- Benchmark consistent reads from cache against consistent reads from etcd for:
  - list result sizes of 1, 10, ..., 100000
  - object sizes of 5kb, 25kb, 100kb
  - measure latency and throughput
  - document results in this KEP

Scalability:

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

### Rollout Plan

#### Serving consistent reads from cache

Guard feature with the `WatchCacheConsistentReads` feature gate.

#### Reflectors

- Provide a way for reflectors to be configured to use resourceVersion=”” for initial list, but for backward compatibility, resourceVersion=”0” must remain the default for reflectors.
- Upgrade the reflectors of in-tree components to use resourceVersion=”" based on a flag or configuration option. Administrators and administrative tools would need to enable this only when using etcd 3.4 or higher.
- If at some point in the (far) future, the lowest etcd supported version for kubernetes is 3.4 or higher, reflectors could be changed to default to resourceVersion="".

### Graduation Criteria

Beta:

- Imapct to pagination (calls that previously were paginated by etcd will be unpaginated when served from the watch cache) is understood and addressed.

## Implementation History

TODO

## Alternatives

Do nothing:

- Leaves the "stale read" problem unsolved, although we have a PR fixing reflector relist which helps mitigate the larger issue.
- Does not impact scale or performance.

Allow clients to manage the initial resource version they provide to reflectors, but don’t implement this optimization:

- Many clients will most likely continue to use resourceVersion=”0” even if it violates their consistency needs
- Clients that transition to use resourceVersion=”” will pay a high scale/performance cost
- We don't expect clients to attempt to keep track of the last resourceVersion they observed. If they do attempt this, we are concerned that they might get it wrong and introduce subtle and difficult to debug issues as a result.

## Rejected alternatives

#### Use WatchProgressRequest to request watch updates when needed

etcd 3.4+ provides a `WatchProgressRequest` request that can be made on a watch channel. When requested,

etcd will send a progress event on that watch as soon as possible.

When an consistent read request is received and the watch cache is enabled:
- Get the current revision from etcd using a range read with limit=0, just like in alternative 1.
- If the cache already has the current revision, serve the request from cache
- If the current revision is not in the cache:
  - Send a `WatchProgressRequest` to etcd on the watch channel that the watch cache is consuming.
- Use the existing waitUntilFreshAndBlock function in the watch cache to wait for the watch to catch up to the current revision
- If the block times out, skip the cache and serve the request directly from storage.

This alternative requires using `WatchProgressRequest` which is only available in etcd 3.4+, and so would
require we make the kube-apiserver aware of etcd's minor version, which is described in more detail later.

Make the kube-apiserver aware of etcd's minor version:

This is only needed if we go with Alternative 2 (Use WatchProgressRequest to
request watch updates when needed) since Alterative 1 is compatible with all
etcd 3.x versions.

Allow the etcd minor version to be specified in the kube-apiserver
`--storage-backend` flag, e.g. `--storage-backend=etcd3.4`.  When client
connections to etcd servers are established (and maybe periodically after that
as well), check the current etcd version (via the etcd 'version' API) and warn
the user if the version is older than the version of etcd provided in the
flag.

if `--storage-backend=etcd3` is provided, the minor etcd version will default to
the detected etcd version minor version. This is for ease of use, since
defaulting to `etcd3.0` would require additional configuration management by
administrators. To make this safe, if the api-server gets an error when making a
request that is enabled for newer etcd versions, it should warn in the logs that
there might be an etcd version mismatch and fallback to the etcd functionality
supported by all etcd 3.x versions.

Also, Due to etcd upgrades and downgrades, there is no way to automatically
detect the etcd version is a way that is guaranteed to be always correct, so the
administrator must be able to set the version to a desired version.

In addition to etcd minor version detection, all features requiring features
introduced at a specific etcd minor version will have feature gates and will go
through the usual kubernetes stability levels promotions (alpha, beta, GA).

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
