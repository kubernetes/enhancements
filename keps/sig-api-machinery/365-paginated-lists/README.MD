# KEP-365: Paginated API Lists

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
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

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [X] e2e Tests for all Beta API Operations (endpoints)
  - [X] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [X] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [X] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [X] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

In this KEP we propose exposing a simple chunking mechanism to allow large API responses
to be broken into consistent partial responses. Clients that can tolerate such behavior
would be able to opt-in for it by specifying a desired maximum number of results to return
in a `LIST` call. This enhancement is critical for large clusters to significantly improve
variations in peak memory use on the server and reduce long tail request latency.

## Motivation

On large clusters, performing API queries that return all of the objects of a given resource type (GET /api/v1/pods, GET
 /api/v1/secrets) can lead to significant variations in peak memory use on the server and contribute substantially to 
 long tail request latency.

When loading very large sets of objects -- some clusters are now reaching 100k pods or equivalent numbers of supporting 
resources -- the system must:

* Construct the full range description in etcd in memory and serialize it as protobuf in the client
  * Some clusters have reported over 500MB being stored in a single object type
  * This data is read from the underlying datastore and converted to a protobuf response
  * Large reads to etcd can block writes to the same range (https://github.com/coreos/etcd/issues/7719)
* The data from etcd has to be transferred to the apiserver in one large chunk
* The `kube-apiserver` also has to deserialize that response into a single object, and then re-serialize it back to the 
client
  * Much of the decoded etcd memory is copied into the struct used to serialize to the client
* An API client like `kubectl get` will then decode the response from JSON or protobuf
  * An API client with a slow connection may not be able to receive the entire response body within the default 60s 
    timeout
    * This may cause other failures downstream of that API client with their own timeouts
  * The recently introduced client compression feature can assist
  * The large response will also be loaded entirely into memory

The standard solution for reducing the impact of large reads is to allow them to be broken into smaller reads via a 
technique commonly referred to as paging or chunking. 
By efficiently splitting large list ranges from etcd to clients into many smaller list ranges, we can reduce the peak 
memory allocation on etcd and the apiserver, without losing the consistent read invariant our clients depend on.

This proposal does not cover general purpose ranging or paging for arbitrary clients, such as allowing web user 
interfaces to offer paged output, but does define some parameters for future extension. 
To that end, this proposal uses the phrase "chunking" to describe retrieving a consistent snapshot range read from the 
API server in distinct pieces.

Our primary consistent store etcd3 offers support for efficient chunking with minimal overhead, and mechanisms exist 
for other potential future stores such as SQL databases or Consul to also implement a simple form of consistent 
chunking.

Relevant issues:

* https://github.com/kubernetes/kubernetes/issues/2349

### Goals

- Expose List API chunking mechanism as opt-in for clients

### Non-Goals

- Force all clients to use pagination for now


## Proposal

### Terminology

**Consistent list** - A snapshot of all resources at a particular moment in time that has a single `resourceVersion` 
that clients can begin watching from to receive updates. All Kubernetes controllers depend on this semantic. 
Allows a controller to refresh its internal state, and then receive a stream of changes from the initial state.

**API paging** - API parameters designed to allow a human to view results in a series of "pages".

**API chunking** - API parameters designed to allow a client to break one large request into multiple smaller requests 
without changing the semantics of the original request.


### Risks and Mitigations

- Security implications of returning a key in the continue token are discussed below

## Design Details

Expose a simple chunking mechanism to allow large API responses to be broken into consistent partial responses. 
Clients would indicate a tolerance for chunking (opt-in) by specifying a desired maximum number of results to return in
a `LIST` call. 
The server would return up to that amount of objects, and if more exist it would return a `continue` parameter that the 
client could pass to receive the next set of results.  
The server would be allowed to ignore the limit if it does not implement limiting (backward compatible), but it is not 
allowed to support limiting without supporting a way to continue the query past the limit (may not implement `limit` 
without `continue`).

```
GET /api/v1/pods?limit=500
{
  "metadata": {"continue": "ABC...", "resourceVersion": "147"},
  "items": [
     // no more than 500 items
   ]
}
GET /api/v1/pods?limit=500&continue=ABC...
{
  "metadata": {"continue": "DEF...", "resourceVersion": "147"},
  "items": [
     // no more than 500 items
   ]
}
GET /api/v1/pods?limit=500&continue=DEF...
{
  "metadata": {"resourceVersion": "147"},
  "items": [
     // no more than 500 items
   ]
}
```

The token returned by the server for `continue` would be an opaque serialized string that would contain a simple 
serialization of a version identifier (to allow future extension), and any additional data needed by the server storage 
to identify where to start the next range. 

The continue token is not required to encode other filtering parameters present on the initial request, and clients may 
alter their filter parameters on subsequent chunk reads. However, the server implementation **may** reject such changes 
with a `400 Bad Request` error, and clients should consider this behavior undefined and left to future clarification. 
Chunking is intended to return consistent lists, and clients **should not** alter their filter parameters on subsequent 
chunk reads.

If the resource version parameter specified on the request is inconsistent with the `continue` token, the server 
**must** reject the request with a `400 Bad Request` error.

The schema of the continue token is chosen by the storage layer and is not guaranteed to remain consistent for clients 
- clients **must** consider the continue token as opaque. 
Server implementations **should** ensure that continue tokens can persist across server restarts and across upgrades.

Servers **may** return fewer results than `limit` if server side filtering returns no results such as when a `label` or 
`field` selector is used. 
If the entire result set is filtered, the server **may** return zero results with a valid `continue` token. 
A client **must** use the presence of a `continue` token in the response to determine whether more results are 
available, regardless of the number of results returned. 
A server that supports limits **must not** return more results than `limit` if a `continue` token is also returned. 
If the server does not return a `continue` token, the server **must** return all remaining results. 
The server **may** return zero results with no `continue` token on the last call.

The server **may** limit the amount of time a continue token is valid for. Clients **should** assume continue tokens 
last only a few minutes.

The server **must** support `continue` tokens that are valid across multiple API servers. The server **must** support a 
mechanism for rolling restart such that continue tokens are valid after one or all API servers have been restarted.

### Proposed Implementations

etcd3 is the primary Kubernetes store and has been designed to support consistent range reads in chunks for this use case. 
The etcd3 store is an ordered map of keys to values, and Kubernetes places all keys within a resource type under a 
common prefix, with namespaces being a further prefix of those keys. 
A read of all keys within a resource type is an in-order scan of the etcd3 map, and therefore we can retrieve in chunks 
by defining a start key for the next chunk that skips the last key read.

etcd2 will not be supported as it has no option to perform a consistent read and is on track to be deprecated in Kubernetes.  
Other databases that might back Kubernetes could either choose to not implement limiting, or leverage their own 
transactional characteristics to return a consistent list. 
In the near term our primary store remains etcd3 which can provide this capability at low complexity.

Implementations that cannot offer consistent ranging (returning a set of results that are logically equivalent to 
receiving all results in one response) must not allow continuation, because consistent listing is a requirement of the 
Kubernetes API list and watch pattern.

#### etcd3

For etcd3 the continue token would contain a resource version (the snapshot that we are reading that is consistent 
across the entire LIST) and the start key for the next set of results. 
Upon receiving a valid continue token the apiserver would instruct etcd3 to retrieve the set of results at a given 
resource version, beginning at the provided start key, limited by the maximum number of requests provided by the 
continue token (or optionally, by a different limit specified by the client). 
If more results remain after reading up to the limit, the storage should calculate a continue token that would begin at 
the next possible key, and the continue token set on the returned list.

The storage layer in the apiserver must apply consistency checking to the provided continue token to ensure that 
malicious users cannot trick the server into serving results outside of its range. 
The storage layer must perform defensive checking on the provided value, check for path traversal attacks, and have 
stable versioning for the continue token.

#### Possible SQL database implementation

A SQL database backing a Kubernetes server would need to implement a consistent snapshot read of an entire resource 
type, plus support changefeed style updates in order to implement the WATCH primitive. 
A likely implementation in SQL would be a table that stores multiple versions of each object, ordered by key and 
version, and filters out all historical versions of an object. 
A consistent paged list over such a table might be similar to:

    SELECT * FROM resource_type WHERE resourceVersion < ? AND deleted = false AND namespace > ? AND name > ? LIMIT ? ORDER BY namespace, name ASC

where `namespace` and `name` are part of the continuation token and an index exists over 
`(namespace, name, resourceVersion, deleted)` that makes the range query performant. 
The highest returned resource version row for each `(namespace, name)` tuple would be returned.

### Security implications of returning last or next key in the continue token

If the continue token encodes the next key in the range, that key may expose info that is considered security sensitive, 
whether simply the name or namespace of resources not under the current tenant's control, or more seriously the name of 
a resource which is also a shared secret (for example, an access token stored as a kubernetes resource). 
There are a number of approaches to mitigating this impact:

1. Disable chunking on specific resources
2. Disable chunking when the user does not have permission to view all resources within a range
3. Encrypt the next key or the continue token using a shared secret across all API servers
4. When chunking, continue reading until the next visible start key is located after filtering, so that start keys are 
always keys the user has access to.

In the short term we have no supported subset filtering (i.e. a user who can LIST can also LIST ?fields= and vice 
versa), so 1 is sufficient to address the sensitive key name issue. Because clients are required to proceed as if 
limiting is not possible, the server is always free to ignore a chunked request for other reasons. 
In the future, 4 may be the best option because we assume that most users starting a consistent read intend to finish 
it, unlike more general user interface paging where only a small fraction of requests continue to the next page.

### Handling expired resource versions

If the required data to perform a consistent list is no longer available in the storage backend (by default, old 
versions of objects in etcd3 are removed after 5 minutes), the server **must** return a `410 Gone ResourceExpired` 
status response (the same as for watch), which means clients must start from the beginning.

```
# resourceVersion is expired
GET /api/v1/pods?limit=500&continue=DEF...
{
  "kind": "Status",
  "code": 410,
  "reason": "ResourceExpired"
}
```

Some clients may wish to follow a failed paged list with a full list attempt.

The 5 minute default compaction interval for etcd3 bounds how long a list can run.  
Since clients may wish to perform processing over very large sets, increasing that timeout may make sense for large clusters. 
It should be possible to alter the interval at which compaction runs to accommodate larger clusters.

#### Types of clients and impact

Some clients such as controllers, receiving a 410 error, may instead wish to perform a full LIST without chunking.

* Controllers with full caches
  * Any controller with a full in-memory cache of one or more resources almost certainly depends on having a consistent 
    view of resources, and so will either need to perform a full list or a paged list, without dropping results
* `kubectl get`
  * Most administrators would probably prefer to see a very large set with some inconsistency rather than no results 
    (due to a timeout under load).  They would likely be ok with handling `410 ResourceExpired` as "continue from the 
    last key I processed"
* Migration style commands
  * Assuming a migration command has to run on the full data set (to upgrade a resource from json to protobuf, or to 
    check a large set of resources for errors) and is performing some expensive calculation on each, very large sets 
    may not complete over the server expiration window.

For clients that do not care about consistency, the server **may** return a `continue` value on the `ResourceExpired` 
error that allows the client to restart from the same prefix key, but using the latest resource version.  
This would allow clients that do not require a fully consistent LIST to opt in to partially consistent LISTs but still 
be able to scan the entire working set. 
It is likely this could be a sub field (opaque data) of the `Status` response under `statusDetails`.

### Rate limiting

Since the goal is to reduce spikiness of load, the standard API rate limiter might prefer to rate limit page requests 
differently from global lists, allowing full LISTs only slowly while smaller pages can proceed more quickly.

### Chunk by default?

On a very large data set, chunking trades total memory allocated in etcd, the apiserver, and the client for higher 
overhead per request (request/response processing, authentication, authorization).  
Picking a sufficiently high chunk value like 500 or 1000 would not impact smaller clusters, but would reduce the peak 
memory load of a very large cluster (10k resources and up).  
In testing, no significant overhead was shown in etcd3 for a paged historical query which is expected since the etcd3 
store is an MVCC store and must always filter some values to serve a list.

For clients that must perform sequential processing of lists (kubectl get, migration commands) this change dramatically 
improves initial latency - clients got their first chunk of data in milliseconds, rather than seconds for the full set. 
It also improves user experience for web consoles that may be accessed by administrators with access to large parts of the system.

It is recommended that most clients attempt to page by default at a large page size (500 or 1000) and gracefully degrade to not chunking.

### Plan

The initial chunking implementation would focus on consistent listing on server and client as well as measuring the 
impact of chunking on total system load, since chunking will slightly increase the cost to view large data sets because 
of the additional per page processing. 
The initial implementation should make the fewest assumptions possible in constraining future backend storage.

For the initial alpha release, chunking would be behind a feature flag and attempts to provide the `continue` or `limit` 
flags should be ignored. While disabled, a `continue` token should never be returned by the server as part of a list.

Future work might offer more options for clients to page in an inconsistent fashion, or allow clients to directly 
specify the parts of the namespace / name keyspace they wish to range over (paging).

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

n/a

##### Unit tests

- staging/src/k8s.io/apiserver/pkg/storage/etcd3: 2023-07-21 - 74%

##### Integration tests

- `TestListOptions`: https://storage.googleapis.com/k8s-triage/index.html?pr=1&job=integration&test=TestListOptions
- `TestListResourceVersion0`: https://storage.googleapis.com/k8s-triage/index.html?pr=1&job=integration&test=TestListResourceVersion0
- `TestAPIListChunking`: https://storage.googleapis.com/k8s-triage/index.html?pr=1&job=integration&test=TestAPIListChunking
- `TestAPIListChunkingWithLabelSelector`: https://storage.googleapis.com/k8s-triage/index.html?pr=1&job=integration&test=TestAPIListChunkingWithLabelSelector

##### e2e tests

- `Servers with support for API chunking`: https://storage.googleapis.com/k8s-triage/index.html?pr=1&test=Servers%20with%20support%20for%20API%20chunking

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- All tests are stable
- Scalability impact of the feature is asessed

#### GA

- e2e tests are graduated to conformance


### Upgrade / Downgrade Strategy

API chunking will be an opt-in feature. Additionally, it's purely in-memory feature
so upgrade/downgrade is not a problem.

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: APIListChunking
  - Components depending on the feature gate: kube-apiserver
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

No - clients have to explicitly opt-in for pagination.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes - by disabling the feature gate.

###### What happens if we re-enable the feature if it was previously rolled back?

Clients will be able to request paginated LIST requests.

###### Are there any tests for feature enablement/disablement?

No - pagination is implemented purely in-memory.

### Rollout, Upgrade and Rollback Planning


###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollout can't impact already running workloads.

Clients that opt-in for pagination may be unable to list large collections when
using pagination due to limited time when continuation is available (especially
when server is throttling requests) - one of the requests is failing with `410 (Gone)`
status code. Fallback to non-paginated request is recommended
in that situation.

###### What specific metrics should inform a rollback?

Increased number of errors on LIST requests: `apiserver_requests_total` metric.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

The feature is enabled by default since release 1.9 so it got battle tested across many
providers and environments over last 5 years.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

This isn't a workload feature.
In general, the following two kube-apiserve metrics should reflect the healtiness
of the API:
  - apiserver_request_duration_seconds
  - apiserver_requests_total

###### How can someone using this feature know that it is working for their instance?

n/a - this isn't a workload feature

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

[API call latency SLO]

[API call latency SLO]: https://github.com/kubernetes/community/blob/master/sig-scalability/slos/api_call_latency.md


###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

[API call latency SLI]

[API call latency SLI]: https://github.com/kubernetes/community/blob/master/sig-scalability/slos/api_call_latency.md

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

no

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No new dependencies

### Scalability

###### Will enabling / using this feature result in any new API calls?

A single LIST API call may now result in few paginated calls to get the same result.

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No - thanks to pagination we should actually reduce API call latencies.
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

In should reduce RAM usage on etcd and kube-apiserver.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

API calls can't be served, so pagintion also doesn't work.

###### What are other known failure modes?

- Fail to list the whole collection:
  - Detection: Clients are not able to LIST the whole collection with one of the
      consecutive requests failing with `410 (Gone)` status code
  - Mitagation: Fallback to non-paginated API call (or eventually migrate to
      watch-list functionality once it reaches Stable).
  - Diagnostics: `apiserver_requests_total` metric with `410` status code
  - Testing: Covered by integration tests

###### What steps should be taken if SLOs are not being met to determine the problem?

Adjusting max-requests-inflights/max-mutating-requests-inflight to reduce load on kube-apiserver/etcd.

## Implementation History

- v1.8: Alpha release
- v1.9: Beta release

## Drawbacks

N/A

## Alternatives

Compression from the apiserver and between the apiserver and etcd can reduce total network bandwidth, but cannot reduce
the peak CPU and memory used inside the client, apiserver, or etcd processes.

Various optimizations exist that can and should be applied to minimizing the amount of data that is transferred from
etcd to the client or number of allocations made in each location, but do not change how response size scales with number of entries.

## Potential future extensions

Not all LIST requests are served from etcd. Our API allows clients to opt-in for the LIST result to be
served from kube-apiserver cache (called watchcache), being propagated via watch from etcd.
However, as of 1.28 release, watchcache still doesn't support pagination. This makes the logic that
we use extremely non-intuitive, as it effectively works as following:

- if the `continuation` is set - always pass the request to etcd (and honor the LIMIT)
- if resourveVersion='0' - ignore the LIMIT parameter and return result from watchcache
- if resourceVersion=<some concrete RV> and LIMIT is set - pass the request to etcd (and honor the LIMIT)
- if resourceVersion=<some concrete RV> and LIMIT is not set - serve the request from watchcache
   - which effectively results in different semantics about data freshness depending on LIMIT being set or not

We propose to add support for pagination in watchcache as following:

1. As of now, watchcache (in addition to transaction log which is not relevant for this proposal) contains
  the current (consistent at a given RV) state of all objects of a given type. This is currently stored
  in client-go `Indexer`, which internally stores that in hash-map.
  We propose to replace the usage of `Indexer` with an implementation that will be build on top of
  BTree that supports copy-on-write semantic. We are planning to use the implementation from
  [https://github.com/google/btree] (although it may require adding more extensive test coverage).
1. When a request with LIMIT set reaches watchcache, we clone (create a copy-onwrite copy) of the current
  state and serve the request from that copy. (The additional advantage is that after cloned, which is
  very fast operation, we no longer block any other operations as lock is no longer needed).
1. When the request is processed, we additionally cache a root to the cloned copy in the watchcache.
  It is necessary to later serve the continuation of that request from the exact same resourceVersion.
  We will keep the cloned copy in the cache as long as the resourceVersion at which it was grabbed
  is cleared up from the watchcache transaction log (so we don't need any additional hooks).

There are three problems that still need to be solved in the above solution.

1. The `Indexer` in which the current state is stored also supports a set of predefined indexes.
  As of 04/2022, there is exactly one index defined for exactly one resource type - for Pods we
  support index by `spec.nodeName`.
  There are two main things we can do here:
  - instead of storing an index in map {value}->{list of keys of objects whose index matches value}
    switch to also use a BTree for the index. This would be the most complete implementation although
    the drawback is that with the copy-on-write semantics, the update of the list means effectively
    a need to copy the list and inserting the modified one. This can become expensive if those lists
    become longer.
  - drop support indexing for continuations. The index is just an optimization so we don't need it
    for correctness (on the cost of less efficient processing). However, given that we only support
    indexing by `spec.nodeName` and the default LIMIT is set to 500, in huge majority of cases
    (in particular Kubelet listing its pods before watching) the whole response will get returned
    in the single call. As a result, we may consider not cloning the index itself and in case
    continuation request will be needed, just go over all objects without the index.

  Given that the index is just an optimization and the need for it is not clear for now, we will
  just start with the second option above (dropping support for indexes for continuations) and
  will reconsider adding that once we observe the need for it.

2. The HA kube-apiserver. The fact that the original request was processed by kube-apiserver A
  doesn't mean that the continuation request will not be send to kube-apiserver B. In the following
  sketch of the algorithm, kube-apiaserver B will not have the cloned state from the given RV
  though.
  Again we have two main approaches here:
  - for every transaction processed by watchcache, we will be making a copy-on-write copy of the
    current state. That means that transaction will not only contain the data about what was
    changed, but also a point to the state from that point (copy-on-write BTree).
  - ignore that usecase for now, and just delegate the continuation request to etcd if the
    copy-on-write state for a resource version specified by continuation doesn't exist in memory

  Similarly as above, processing continuation in watchcache is just an optimization and we don't
  strictly need it. However, the first option is simple to implement to we will implement it
  and use it unless a significant cpu/memory overhead will be shown by the benchmarks.

3. The length of `out-of-history` window. Currently in watchcache we're storing transactions from
  the last ~75 seconds (we're clearing transactions older than that when a new transaction is
  being processed). This means that continuation will be served from watchcache only for ~75
  seconds. However, in etcd have 2.5 to 5 minutes of history. As a result, we may consider
  extending the length of the transaction log to match the minimum 2.5 minutes from above.

The API chunking feature is heavily used so we decided to finally graduate it to GA without
implementing the above proposal. However, we should consider implementing it at some point.

[https://github.com/google/btree]: https://github.com/google/btree
