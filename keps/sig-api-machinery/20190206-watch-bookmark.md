---
title: Watch Bookmark
authors:
  - "@wojtek-t"
owning-sig: sig-api-machinery
participating-sigs:
  - sig-scalability
reviewers:
  - "@jpbetz"
approvers:
  - "@deads2k"
  - "@lavalamp"
creation-date: 2019-02-06
last-updated: 2019-04-30
status: implemented
see-also:
  - "https://github.com/kubernetes/kubernetes/issues/73585"
replaces:
  - n/a
superseded-by:
  - n/a
---

# Watch bookmark

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Test Plan](#test-plan)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Rejected alternatives](#rejected-alternatives)
  - [Cache in kube-apiserver](#cache-in-kube-apiserver)
  - [API for send bookmark](#api-for-send-bookmark)
<!-- /toc -->

## Summary

Watch API is one of the fundaments of Kubernetes API. The recommended pattern
for using watch API is to retrieve a collection of resources using consistent
list and then initiate a watch starting from a resourceVersion returned by the
list operation. If the client watch is disconnected, a new one can be restarted
from the last returned resourceVersion.

This proposal make restarting watches cheaper from kube-apiserver performance
perspective.

## Motivation

While running different scalability tests we observed that restarting watches
may cause significant load on kube-apiserver when watcher is observing a small
percentage of changes (due to field or label selector). In extreme cases,
reestablishing such watcher may even lead to falling out of history window
and "resource version too old" errors (that requires full relist for that
watcher).

The reason for that is the fact that even if the last item received by watcher
has resourceVersion rv1, we may already know that there aren't any changes
a given watcher is interested in up to rv2 (rv1 < rv2), but we don't have any
way of communicating it to the watcher. As a result, when restarting a watch,
client again sends rv1 as a starting point, and we process all events with
resourceVersion between rv1 and rv2 again unnecessarily.

The proposal presents a proper solution for that problem.

### Goals

- Reduce load on apiserver by minimizing amount of unnecessary watch events
that need to be processed after restarting a watch.
- Reduce amount of undesired "resource version too old" errors on reestablishing
a watch.

### Non-Goals

The following are nice-to-haves, but not primary goals:

- Improve overall watch throughput and/or latency.

## Proposal

We propose introducing a new type of watch event called `Bookmark`. With that
change, the possible watch event types will be:
```
  Added    EventType = "ADDED"
  Modified EventType = "MODIFIED"
  Deleted  EventType = "DELETED"
  Error    EventType = "ERROR"
  Bookmark EventType = "BOOKMARK"
```

Watch event with type Bookmark will represent information that all the objects
up to a given resourceVersion has been processed for a given watcher. So even
if the last event of other types contained object with resourceVersion rv1,
receiving a bookmark with resourceVersion rv2 means that there aren't
any interesting objects for that watcher in between.

Given that we don't want to wait for v2 version of watch API with that change,
an obvious requirement for introducing it is backward compatibility. Currently
watch Event type looks as following:
```
  type Event struct {
    Type EventType
    Object runtime.Object
  }
```

As a result, we will represent bookmark event by setting Bookmark type and
Object of appropriate type with just ObjectMeta.ResourceVersion field set.

Unfortunately, such a change would break existing clients, as an example
[official decoder][]. As a result, we will extend `ListOptions` (which is
how we pass options to watch) with a boolean field where user can opt-in
for watch bookmarks:
```
  type ListOptions struct {
    ...
    AllowWatchBookmarks bool
  }
```
We consciously make it just a boolean flag - this gives kube-apiserver an
ability to choose when they should be send without setting any expectations
on user side how frequently and when it would be happening. In particular
client isn't guaranteed to get any bookmarks.
Such addition to `ListOptions` is also safe from backward-compatibility
point of view, old kube-apiserver will simply drop this field if set.

Once the API is extended, we add a support for sending bookmarks to watchcache.
The exact policy of sending them is to be determined, the ideas include:
- send a bookmark if there weren't any event send to user in the last X
seconds
- if we know we will be closing (e.g. due to timeout) a watch, try to send
a bookmark immediately before that

Finalizing the decision shouldn't block the initial version of this KEP.

[official decoder]: https://github.com/kubernetes/kubernetes/blob/5d4795e14e02ac29273009d86ba3c5012684d5f4/staging/src/k8s.io/client-go/rest/watch/decoder.go#L57


### Risks and Mitigations

Sending "watch bookmarks" may break clients not understanding them.
As a result, we make them explicitly opt-in.

### Test Plan

For `Alpha` a set of unit tests will be added to verify that bookmarks are
send as assumed by the implementation.
Given the nature of the feature (no guarantees that bookmarks will be send
to the watcher), no e2e tests will be added.

For `Beta` a metrics exposing number of processed `init events` is gathered
by our scalability test framework so that the impact can be clearly proved
on our dashboards.

## Graduation Criteria

Beta:
- Proved scalability/performance gain. With a simple POC I was able to
reduce amount of processed "init events" by ~40x. So setting the minimum
goal on 10x without adding any additional visible overhead.
- Generated informers make use of this new API.

GA:
- Enabled by default in Kubelet for watching pods for a release.
- No complaints about the API for a release

## Implementation History

- 2019-02-12: KEP Summary, Moativation and Proposal merged
- 2019-03-27: API changes approved in API review
- 2019-04-16: Implementation merged
- v1.15: Launched in `Alpha`

## Rejected alternatives

The most important considered alternatives are mentioned below.

### Cache in kube-apiserver

Instead of introducing an API for bookmarks, we can try memorizing in watchcache
what we already processed for a watcher and when it is restarted use that
information. However, that would require being able to identify and match a
watcher across restarts which is non-trivial. Moreover, it doesn't work in
HA setups with multiple kube-apiserver.

### API for send bookmark

This is similar to what was done an etcd, where an API was added to notify
all watchers about current resourceVersion. However, such API would be hard
to manage in kube-apiserver, and we don't really want to notify everyone at
the same time.
