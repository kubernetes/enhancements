---
title: KEP Template
authors:
  - "@jpbetz"
  - "@wojtek-t"
owning-sig: sig-api-machinery
participating-sigs:
  - sig-scalability
reviewers:
  - TBD
  - "@wojtek-t"
  - "@jingyih"
approvers:
  - TBD
editor: TBD
creation-date: 2019-12-10
last-updated: 2019-12-10
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
- [Proposal](#proposal)
  - [Leveraging the Progress Notify Mechanism](#leveraging-the-progress-notify-mechanism)
    - [Alternative 1: Use WithProgressNotify to enable automatic watch updates](#alternative-1-use-withprogressnotify-to-enable-automatic-watch-updates)
    - [Alternative 2: Use WatchProgressRequest to request watch updates when needed](#alternative-2-use-watchprogressrequest-to-request-watch-updates-when-needed)
    - [Comparing the alternatives](#comparing-the-alternatives)
  - [Make the kube-apiserver aware of etcd's minor version](#make-the-kube-apiserver-aware-of-etcds-minor-version)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Pagination](#pagination)
  - [Test Plan](#test-plan)
  - [Rollout Plan](#rollout-plan)
    - [Serving consistent reads from cache](#serving-consistent-reads-from-cache)
    - [Reflectors](#reflectors)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
- [Potential Future Improvements](#potential-future-improvements)
<!-- /toc -->

## Summary

Consistent reads may be served from cache so long as:
- A consistent (quorum) read is first made to etcd to get the latest "revision"
- The data in the watch cache no older than the latest "revision" just from etcd

etcd 3.4 supports progress events. [Progress event
interval](https://github.com/etcd-io/etcd/blob/e6980b1f9fc008837abb9177e87e893e48d565c1/etcdserver/api/v3rpc/watch.go#L67)
are sent by etcd automatically, and [progress
notify](https://github.com/etcd-io/etcd/issues/9855) requests a progress event
and was added specifically to make it easy to check if a cache that is updated
via an etcd watch is up-to-date relative to some etcd revision).

This KEP summarizes how we can take advantage of progress events efficiently
determine how up-to-date kubernetes watch caches are then serve reads from the
watch cache when they are sufficiently up-to-date.

## Motivation

Serving reads from the watch cache is more performant and scalable than reading
them from etcd, deserializing them, converting them to the desired type and then
garbage collecting all the objects that were allocated during the read.

We will need to measure the impact to performance and scalability, but we have
enough data and experience from prior improvements made by the watch cache to be
confident there is significant scale/perf opportunity, and we would like to
measure it.

We expect the biggest gain to be from node-originating requests (e.g. kubelet
listing pods scheduled on its node). For those requests, the size of the
response is small (it fits a single page, assuming you won't make it extremely
small), whereas the number of objects to process is proportional to cluster-size
(so fairly big). For example, when kubelets requests pods schedule against it in
a 5000 node cluster with 30pods/node, the kube-apiserver must list the 150k pods
from etcd and then filter that list down to the list of 30 pods that the kubelet
actually need. This must occur for each list request from each of the 5000
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
- Improve the scailability and performance of Kubernetes for Get and List requests, when the watch cache is enabled

## Proposal

### Leveraging the Progress Notify Mechanism

Guard this by a `WatchCacheConsistentReads` feature gate.

#### Alternative 1: Use WithProgressNotify to enable automatic watch updates

Create etcd watches with `WithProgressNotify` enabled (available in all etcd 3.x versions).

When `WithProgressNotify` is enabled on an etcd watch, etcd sends progress
events to the watch automatically. By default etcd sends progress events every
10 minutes, which is not frequent enough to be useful for our needs, so we will
modify etcd to send them must more frequently, e.g. every 20ms.

When an consistent read request is received and the watch cache is enabled:
- Get the current revision from etcd using a range read with limit=0 for the resource type being served. Etcd can serve this type of request efficiently, and the resulting revision is strongly consistent (guaranteed to be the latest revision via a quorum read).
- Use the existing `waitUntilFreshAndBlock` function in the watch cache to wait briefly (20ms?) for the watch to catch up to the current revision.
- If the block times out, skip the cache and serve the request directly from storage (etcd).

#### Alternative 2: Use WatchProgressRequest to request watch updates when needed

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

#### Comparing the alternatives

We will experiment with modifying etcd to automatically send progress events in
short intervals of 20ms or so to see if this keeps the latest observed revision
of watch caches sufficiently up-to-date that they rarely need to fall back to
serving resourceVersion="" requests directly from etcd.  We will check for cache
hit ratios and check scalability. If these are promising we will then explore
what the optimal interval to have etcd send progress events is. If this works
well enough we will go with alternative 1 and will not need to implement
alternative 2.

We will also explore how long the kube-apiserver should wait for a watch cache
to catch up to the needed revision. I.e. for a progress event interval of 20ms,
should the kube-apiserver also wait 20ms to a desired revision to become
available or should it fallback to making the request to etcd sooner?

Optional: For some or all of the etcd progress watch events, also create a
kubernetes "bookmark" watch event and send it to kube-apiserver clients so that
reflectors and shared informers are kept up-to-date. The benefit of this is that
it minimizes the chance that these clients will end up with an out-of-date
resource version and need to relist (which can impact scalability). See [Watch
Bookmarks](https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/20190206-watch-bookmark.md)
for details.

### Make the kube-apiserver aware of etcd's minor version

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

### Risks and Mitigations

Alternative 2 can increase the number of round trips to etcd: One is required to
get the latest revision, another might be needed to request the progress
notify. Both of these requests are cheap and scalable, but we need to understand
the latency impact of waiting for both of them before serving a response back to
the client. If this turns out to have performance implications, it could be
partially mitigated by configuring etcd to send regular progress events, which
is something we will explore.

In the worst case, the “wait until fresh” duration is exceeded, and then the
request must be served from etcd anyway, which will be higher latency than
immediately serving from etcd like we do today. We will need to set the
“wait until fresh” duration to relatively short value (e.g. 20ms), which
partially mitigates this.

## Design Details

### Pagination

Because the watch cache does not paginate responses, clients that were
previously getting pagination for RV="" requests will start getting unpaginated
responses when this feature is enabled.

The impacts of this can be explored when the feature is in alpha.

### Test Plan

We will need to scale and performance test this carefully. We’ll need to measure
latency, throughput, volume of progress notify requests, and change to volume of
list requests served from storage instead of from cache.

### Rollout Plan

#### Serving consistent reads from cache

Guard it with the `WatchCacheConsistentReads` feature gate.
Communicate that clusters administrators should set `--storage-backend` with major.minor version, .e.g. `--storage-backend=etcd3.4` to benefit from features requiring newer etcd versions.

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

## Potential Future Improvements

Modify etcd to allow echo back a user provided ID in progress events.

- Client generates a UUID and provides to the ProgressNotify request
- Once client sees a progress event with the same UUID, it knows the watch is up-to-date
- This reduces the worst case number of round trips required to do a consistent read from two to one since client doesn't need to get the lastest revision from etcd first
