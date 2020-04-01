---
title: consistent-resource-version-semantics
authors:
  - "@jpbetz"
owning-sig: sig-api-machinery
reviewers:
approvers:
  - "@lavalamp"
  - "@deads2k"
creation-date: 2020-03-09
last-updated: 2020-04-01
status: provisional
---

# Title

consistent-resource-version-semantics

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [Add a ResourceVersionMatch query parameter](#add-a-resourceversionmatch-query-parameter)
  - [Backward Compatibility](#backward-compatibility)
  - [Get support?](#get-support)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Alternatives Considered](#alternatives-considered)
  - [Alternative: Introduce ExactResourceVersion and MinResourceVersion parameters](#alternative-introduce-exactresourceversion-and-minresourceversion-parameters)
  - [Alternative: Use syntax in the query string](#alternative-use-syntax-in-the-query-string)
<!-- /toc -->

## Summary

Make resource version semantics consistent for list and get requests regardless of
pagination.

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

Add an optional `ResourceVersionMatch` paramater to `ListOptions` and
`GetOptions` with the enumeration values:

```
// ResourceVersionMatch specifies how the ResourceVersion parameter is applied. ResourceVersionMatch
// may only be set if ResourceVersion is also set.
type ResourceVersionMatch string

const (
	// ResourceVersionMatchNotOlderThan matches data at least as new as the provided
	// ResourceVersion. The newest available data is preferred, but any data not
	// older than this ResourceVersion may be served.
	// This guarantees that ResourceVersion in the ListMeta is not older than the requested
	// ResourceVersion, but does not make any guarantee about the ResourceVersion in the ObjectMeta
	// of the list items since ObjectMeta.ResourceVersion tracks when an object was last updated,
	// not how up-to-date the object is when served.
	ResourceVersionMatchNotOlderThan ResourceVersionMatch = "NotOlderThan"
	// ResourceVersionMatchExact matches data at the exact ResourceVersion
	// provided. If the provided ResourceVersion is unavailable, the server responds with
	// HTTP 410 “Gone”.
	// This guarantees that ResourceVersion in the ListMeta is the same as the requested
	// ResourceVersion, but does not make any guarantee about the ResourceVersion in the ObjectMeta
	// of the list items since ObjectMeta.ResourceVersion tracks when an object was last updated,
	// not how up-to-date the object is when served.
	ResourceVersionMatchExact ResourceVersionMatch = "Exact"
)
```

```
type ListOptions struct {
    ...
    // When specified with a watch call, shows changes that occur after that particular version of a resource.
	// Defaults to changes from the beginning of history.
	// If set for a list call, ResourceVersionMatch should also be set.
	// When specified for list:
	// - if unset, then the result is returned from remote storage based on quorum-read flag;
	// - if set and ResourceVersionMatch is set, requests that the server apply the ResourceVersionMatch rule;
	// - if set and ResourceVersionMatch is unset or the server ignores ResourceVersionMatch, the legacy behavior applies:
	//   - if 0, the result may contain arbitrarily old data, no guarantee;
	//   - if non-zero and Limit is unset, ResourceVersionMatchNotOlderThan rule applies implicitly;
	//   - if non-zero and Limit is set, ResourceVersionMatchExact rules applies implicitly.
	// +optional
	ResourceVersion string `json:"resourceVersion,omitempty" protobuf:"bytes,4,opt,name=resourceVersion"`

	// ResourceVersionMatch determines how ResourceVersion is applied. Not supported for watch calls.
	// ResourceVersionMatch SHOULD be set for list calls where ResourceVersion is set. If ResourceVersion is unset,
	// ResourceVersionMatch is ignored.
	// For backward compatibility, clients must tolerate the server ignoring ResourceVersionMatch:
	// - When using ResourceVersionMatchNotOlderThan and Limit is set, clients must handle HTTP 410 “Gone” responses.
	//   For example, the client might retry with a newer ResourceVersion or fall back to a ResourceVersion="" request.
	// - When using ResourceVersionMatchExact and Limit is unset, clients must verify that the ResourceVersion in the
	//   ListMeta of the response matches the requested ResourceVersion, and handle the case where it does not. For
	//   example, the client might fall back to the a request with Limit set.
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

### Get support?

`ResourceVersionMatch` can also be added to the get operation for consistency. `Get` currently
`NotOlderThan` semantics by default and this would add support for `Exact`.

In order to be backward compatible, get responses must include the `ResourceVersion` that the request was served at.
For list responses this is provided in `ListMeta`, but get responses do not have a wrapper object like `ListMeta`.
The `ObjectMeta.ResourceVersion` cannot be used because it represents the resource version that the object was
created or last modified at, not the resource version is was served from.

Options:
- Don't support `ResourceVersionMatch` for get.
- Add a header that provides the `ResourceVersion` the get was served at back to clients, e.g. `ServedAtResourceVersion: 43049`

I am currently considering not including `ResourceVersionMatch` since:
- The header approach sets precidence for an approach I'm not sure we want to encourage in api-machinery
- get currently does not have the semantic consistency problems of list, and so does not urgently need this parameter
- it is possible to use list to get a single item at a specific resource version already

### Risks and Mitigations

The main risk to this proposal that it complicates the API surface area, resulting in an API that is
more difficult to understand and use. But the existing behavior already complicates the API. With
this proposal, clients at least have the ability to use a consistent set of semantics.

Another risk is that clients will either not realize, or not be sufficiently
motivated, to update their code to move away from the legacy behavior. This
can be mitigated a couple ways:
- Update client bindings (client-go, ...) to discourage using the
  legacy behavior, and eventually to disallow it.
- At some point in the future, start logging warning on the server when the
  legacy behavior is used to make it more obvious what needs to be changed?

## Alternatives Considered

### Alternative: Introduce ExactResourceVersion and MinResourceVersion parameters

Deprecate `ResourceVersion` and introduce `ExactResourceVersion` and `MinResourceVersion`.

The three cases are equivalent to the `ResourceVersionMatch` cases from option 1
and would use the equivalent documentation.

This makes it obvious that the `ResourceVersion` parameter is deprecated. It
does this as the API aesthetic cost of having a top level parameter be forever
deprecated.

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

- Clients have to include `ResourceVersion` even when using the new parameters for backward compatibility.
- `ResourceVersion` becomes deperacated but can never be removed.

### Alternative: Use syntax in the query string

Introduce syntax (`=N` and `>=N`) instead of additional parameters.

The disadvantage of this is that many frameworks expect query parameters to be
`=` separated key value pairs. It would also need to somehow retain backward
compatibility (`==N` for exact, `=N` for legacy)?.

