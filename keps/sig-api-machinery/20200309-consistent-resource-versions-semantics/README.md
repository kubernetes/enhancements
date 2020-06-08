# Consistent Resource Version Semantics for List

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [Add a ResourceVersionMatch query parameter](#add-a-resourceversionmatch-query-parameter)
  - [Backward Compatibility](#backward-compatibility)
  - [Impact on get calls](#impact-on-get-calls)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Alternatives Considered](#alternatives-considered)
  - [Alternative: Introduce ExactResourceVersion and MinResourceVersion parameters](#alternative-introduce-exactresourceversion-and-minresourceversion-parameters)
  - [Alternative: Use syntax in the query string](#alternative-use-syntax-in-the-query-string)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

Make resource version semantics consistent for list requests regardless of pagination.

## Motivation

Resource version semantics are inconsistent when using pagination. When a list
request is made with a resourceVersion but no limit, “Not older than” semantics
apply, but once a limit is set, “Exact” semantics apply. See [API Concepts: Resource Version](https://kubernetes.io/docs/reference/using-api/api-concepts/#resource-versions)
for details about the semantics.

The inconsistency is confusing and problematic. A client expecting “Not older
than” semantics has no reason to expect the “410 Gone” HTTP responses from the
server that can be sent when “Exact” semantics are applied and when the
requested resource version is older than what is available in watch cache or the
etcd compaction window. Even if the author of code to make a list request was
aware of this inconsistency, the code to make a list request might be separated
from the code setting a limit on the request. For example, ListerWatchers can be
layered in to add pagination limits.

This was discussed in [Feb 12th, 2020 SIG API machinery bi-weekly meeting](https://docs.google.com/document/d/1x9RNaaysyO0gXHIr1y50QFbiL1x8OWnk2v3XnrdkT5Y/edit#bookmark=id.3kvpricxohe8).

### Goals

- Improve API to make resource version semantics consistent.
- Backward compaibility. In particular, support the case of a client making a request using the
  updated API against an server serving the old API.

## Proposal

### Add a ResourceVersionMatch query parameter

Add an optional `ResourceVersionMatch` paramater to `ListOptions` with the enumeration values:

```
// ResourceVersionMatch specifies how the ResourceVersion parameter is applied. ResourceVersionMatch
// only applies to list calls and may only be set if ResourceVersion is also set.
type ResourceVersionMatch string

const (
	// ResourceVersionMatchNotOlderThan matches data at least as new as the provided
	// ResourceVersion. The newest available data is preferred, but any data not
	// older than this ResourceVersion may be served.
	// For list calls, this guarantees that ResourceVersion in the ListMeta is not older than the requested
	// ResourceVersion, but does not make any guarantee about the ResourceVersion in the ObjectMeta
	// of the list items since ObjectMeta.ResourceVersion tracks when an object was last updated,
	// not how up-to-date the object is when served.
	ResourceVersionMatchNotOlderThan ResourceVersionMatch = "NotOlderThan"
	// ResourceVersionMatchExact matches data at the exact ResourceVersion
	// provided. If the provided ResourceVersion is unavailable, the server responds with
	// HTTP 410 “Gone”.
	// For list calls, this guarantees that ResourceVersion in the ListMeta is the same as the requested
	// ResourceVersion, but does not make any guarantee about the ResourceVersion in the ObjectMeta
	// of the list items since ObjectMeta.ResourceVersion tracks when an object was last updated,
	// not how up-to-date the object is when served.
	ResourceVersionMatchExact ResourceVersionMatch = "Exact"
)
```

```
type ListOptions struct {
    ...
	// ResourceVersion sets a constraint on what resource versions a request may be served from.
	// If unset, the request is served from latest resource version to ensure strong consistency
	// (i.e. served from etcd via a quorum read). Setting a ResourceVersion is preferable in cases
	// where a ResourceVersion is known and can be used as a constraint since better performance
	// and scalability can be achieved for a cluster by avoiding quorum reads.
	// When ResourceVersion is set for list, it is highly recommended ResourceVersionMatch is also set.
	//
	// ResourceVersion for watch:
	// - if unset or 0, start a watch at any resource version, the most recent resource version available
	//   is preferred, but not required; any starting resource version is allowed. It is
	//   possible for the watch to start at a much older resource version that the client
	//   has previously observed, particularly in high availability configurations, due to
	//   partitions or stale caches. Clients that cannot tolerate this should not start a
	//   watch with this semantic. To establish initial state, the watch begins with synthetic
	//   “Added” events for all resources instances that exist at the starting resource version.
	//   All following watch events are for all changes that occurred after the resource version
	//   the watch started at;
	// - if non-zero, start a watch at an exact resource version. The watch events are for all
	//   changes after the provided resource version. Watch is not started with synthetic “Added”
	//   events for the provided resource version. The client is assumed to already have the
	//   initial state at the starting resource version since the client provided the resource version.
	//
	// ResourceVersion for list:
	// - if unset, then the result is returned at the most recent resource version. The returned
	//   data must be consistent (i.e. served from etcd via a quorum read);
	// - if set and ResourceVersionMatch is set, ResourceVersion is applied according to the ResourceVersionMatch rule;
	// - if set and ResourceVersionMatch is unset or the server ignores ResourceVersionMatch, the legacy behavior applies:
	//   - if 0, the result may be at any resource version. The newest available resource version
	//     is preferred, but strong consistency is not required; data at any resource version may
	//     be served. It is possible for the request to return data at a much older resource version
	//     that the client has previously observed, particularly in high availability configurations,
	//     due to partitions or stale caches. Clients that cannot tolerate this should not use this
	//     semantic;
	//   - if non-zero and Limit is unset, the ResourceVersionMatchNotOlderThan rule applies implicitly;
	//   - if non-zero and Limit is set, the ResourceVersionMatchExact rule applies implicitly.
	//
	// Defaults to unset
	// +optional
	ResourceVersion string `json:"resourceVersion,omitempty" protobuf:"bytes,4,opt,name=resourceVersion"`

	// ResourceVersionMatch determines how ResourceVersion is applied to list calls.
	// It is highly recommmend that ResourceVersionMatch be set for list calls where ResourceVersion is
	// set. If ResourceVersion is unset, ResourceVersionMatch is ignored.
	// For backward compatibility, clients must tolerate the server ignoring ResourceVersionMatch:
	// - When using ResourceVersionMatchNotOlderThan and Limit is set, clients must handle HTTP 410 “Gone” responses.
	//   For example, the client might retry with a newer ResourceVersion or fall back to ResourceVersion="".
	// - When using ResourceVersionMatchExact and Limit is unset, clients must verify that the ResourceVersion in the
	//   ListMeta of the response matches the requested ResourceVersion, and handle the case where it does not. For
	//   example, the client might fall back to the a request with Limit set.
	// Unless you have strong consistency requirements, using ResourceVersionMatchNotOlderThan and a known
	// ResourceVersion is preferable since it can achieve better performance and scalability of your cluster
	// than leaving ResourceVersion and ResourceVersionMatch unset, which requires quorum read to be served.
	// Defaults to unset
	// +optional
	ResourceVersionMatch ResourceVersionMatch `json:"resourceVersionMatch,omitempty" protobuf:"bytes,10,opt,name=resourceVersionMatch"`
    ...
}
```

### Backward Compatibility

Versions of the kube-apiserver that pre-date the introduction of `ResourceVersionMatch`
parameter will ignore it. Client will need to tolerate this as documented on ResourceVersionWatch:

- When using ResourceVersionMatchNotOlderThan and Limit is set, clients must handle HTTP 410 “Gone” responses.
  For example, the client might retry with a newer ResourceVersion or fall back to a ResourceVersion="" request.
- When using ResourceVersionMatchExact and Limit is unset, clients must verify that the ResourceVersion in the
  ListMeta of the response matches the requested ResourceVersion, and handle the case where it does not. For
  example, the client might fall back to the a request with Limit set.

When 'ResourceVersionMatch' is not provided, the behavior is the same as before it was introduced.

### Impact on get calls

We will not add `ResourceVersionMatch` to get calls.

get provides consistent `NotOlderThan` semantics when `ResourceVersion` is set, which are easy
to understand and doesn't need to be changed.

get does not need the flexibility of `Exact` resource version lookup since it is already possible to
use a list with `fieldSelector=metadata.name=foo` to get a single object with list.

### Risks and Mitigations

The main risk to this proposal that it complicates the API surface area, resulting in an API that is
more difficult to understand and use. But the existing behavior already complicates the API. With
this proposal, clients at least have the ability to use a consistent set of semantics.

Another risk is that clients will either not realize, or not be sufficiently
motivated, to update their code to move away from the legacy behavior. This
can be mitigated a couple ways:
- Update client libraries (client-go, ...) to discourage using the
  legacy behavior, and eventually to disallow it.
- At some point in the future, start providing warnings to the client that legacy behavior. E.g. via
  https://github.com/kubernetes/kubernetes/pull/73032.

## Alternatives Considered

### Alternative: Introduce ExactResourceVersion and MinResourceVersion parameters

Deprecate `ResourceVersion` and introduce `ExactResourceVersion` and `MinResourceVersion`.

**Backward Compatibility:**

Versions of the kube-apiserver that pre-date the introduction of the `ExactResourceVersion`
and `MinResourceVersion` parameters will ignore them, resulting in a quorum read. For clients
to tolate responses from servers that ignore the new parameters in a backward compatible way:

- When using `MinResourceVersion`, clients can either:
  - Also include `ResourceVersion` and get legacy semantics if `MinResourceVersion` is ignored.
    They will need to handle HTTP 410 “Gone” responses if Limit is set.
  - Tolerate a quorum read, which is guaranteed to provided the newest ResourceVersion
    but has scalability/performance implications.
- When using `ExactResourceVersion` also set `ResourceVersion` and check the ResourceVersion in the ListMeta of the response.
  If it does not match, fall back to a request with a Limit set.

**Advantages**

- Deprecating `ResourceVersion` highlights to API users that they need change calls that use ResourceVersion.

**Disadvantages**

- Since the field will deprecated but never removed, in practice we have 3 options to understand instead of 2.
- Clients have to include `ResourceVersion` even when using the new parameters for backward compatibility,
  resulting in `?ResourceVersion=3847&MinResourceVersion=3847` which is less readable than
  `?ResourceVersion=3847&ResourceVersionMatch=NotOlderThan`.

### Alternative: Use syntax in the query string

Introduce syntax (`=N` and `>=N`) instead of additional parameters.

**Advantages**

- consise query parameters: `resourceVersion=234`, `resourceVersion>=234`.

**Disadvantages**

- Still need to deal with backward compatibility, so we end up having to ask clients to do
  `resourceVersion=234&resourceVersion>=234`.
- Not clear how we would support 'Exact' matches. `==` doesn't work as well in query params
  so we'd need to select something else, and we already use `=` for the legacy case.


## Implementation History

- Proof of concept: https://github.com/kubernetes/kubernetes/compare/master...jpbetz:rv-semantics
