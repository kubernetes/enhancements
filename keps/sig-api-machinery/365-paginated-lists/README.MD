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
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
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

- unit tests for the pagination logic in etcd3 client
- integration tests checking valid and invalid combinations of limit and continue both
  with watchcache enabled and disabled
- e2e tests verifying that clients opting in for chunking can retrieve consistent
  paginated results

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

TODO: Fill this in

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

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

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

- [ ] Events
  - Event Reason:
- [ ] API .status
  - Condition name:
  - Other field:
- [ ] Other (treat as last resort)
  - Details:

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

- v1.8: Alpha release
- v1.9: Beta release

## Drawbacks

N/A

## Alternatives

Compression from the apiserver and between the apiserver and etcd can reduce total network bandwidth, but cannot reduce
the peak CPU and memory used inside the client, apiserver, or etcd processes.

Various optimizations exist that can and should be applied to minimizing the amount of data that is transferred from
etcd to the client or number of allocations made in each location, but do not change how response size scales with number of entries.
