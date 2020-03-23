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
last-updated: 2020-03-09
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
  - [Option 1: Add a new query parameter to set desired semantics](#option-1-add-a-new-query-parameter-to-set-desired-semantics)
  - [Option 2: Introduce new parameters](#option-2-introduce-new-parameters)
  - [Option 3: Use syntax in the query string](#option-3-use-syntax-in-the-query-string)
  - [Risks and Mitigations](#risks-and-mitigations)
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

- Fix resource version semantics to be consistent for list and get.
- Retain backward compatibility.

## Proposal

### Option 1: Add a new query parameter to set desired semantics

Add an optional `ResourceVersionMatch` paramater to `ListOptions` and
`GetOptions` with the enumeration values:

* `Legacy` (Default, Deprecated): ResourceVersionMatch acts as if set to “Minimum”
  unless Limit is set, in which case it acts as if set to “Exact” .
* `Exact`: Return data at the exact ResourceVersion provided. If the provided
  ResourceVersion is unavailable respond with HTTP 410 “Gone”.
* `Minimum`: Return data at least as new as the provided ResourceVersion. The
  newest available data is preferred, but any data not older than this
  ResourceVersion may be served. Note that this ensures only that the objects
  returned are no older than they were at the time of the provided
  ResourceVersion; the resource version in the ObjectMeta of an individual
  object may be older than the provided ResourceVersion so long it is for the
  latest modification to the object at the time of the provided resource
  version.

The `ResourceVersion` documentation would also be updated to:
...
 When specified for list:
 - if unset, then the result is returned from remote storage based on
   quorum-read flag;
 - if it's 0, the result may contain arbitrarily old data, no guarantee; Only
   valid with ResourceVersionMatch of “Minimum” (or “Legacy” acting as
   “Minimum”).
 - if set to non zero, ResourceVersionMatch applies.
 +optional
 
This option has the advantage of not deprecating any top level fields and
making it clear what resource version matching options are available.

It has the disadvanatage of placing the deprecated behavior on the default
value of an new optional parameter, which it at risk of not being noticed.

**Backward Compatibility:**

Versions of the kube-apiserver that pre-date the introduction of `ResourceVersionMatch`
parameter will ignore it.

Versions that support it will include `ResourceVersionMatch` in list responses.
Clients that set `ResourceVersionMatch` in the request are expected to check for
`ResourceVersionMatch` in the response. If it is absent from the response, the client
must assume the server does not support `ResourceVersionMatch` and must interpret the
response as `Legacy`.

### Option 2: Introduce new parameters

Deprecate `ResourceVersion` and introduce `ExactResourceVersion` and `MinResourceVersion`.

The three cases are equivalent to the `ResourceVersionMatch` cases from option 1
and would use the equivalent documentation.

This makes it obvious that the `ResourceVersion` parameter is deprecated. It
does this as the API aesthetic cost of having a top level parameter be forever
deprecated.

**Backward Compatibility:**

Versions of the kube-apiserver that pre-date the introduction of the `ExactResourceVersion`
and `MinResourceVersion` parameters will ignore them, resulting in a quorum read.

Versions that support them will need to include information in the list response to make
it clear to the client what resource version match rule has been used. This could be
done by adding optional `ExactResourceVersion` and `MinResourceVersion` fields to list responses
which are set to whatever parameter was provided in the request.

### Option 3: Use syntax in the query string

Introduce syntax (`=N` and `>=N`) instead of additional parameters.

The disadvantage of this is that many frameworks expect query parameters to be
`=` separated key value pairs. It would also need to somehow retain backward
compatibility (`==N` for exact, `=N` for legacy)?.

**Backward Compatibility:**

TODO

### Risks and Mitigations

The main risk to this approach is that it complicates the API surface area,
resulting in an API that is more difficult to understand and use. But the
existing behavior already complicates the API so it should be addressed. We
still do need to be mindful of the impact to API asthetics when addressing it.

Another risk is that clients will either not realize, or not be sufficiently
motivated, to update their code to move away from the legacy behavior. This
can be mitigated a couple ways:
- Update client bindings (client-go, ...) to discourage using the
  legacy behavior, and eventually to disallow it.
- At some point in the future, start logging warning on the server when the
  legacy behavior is used to make it more obvious what needs to be changed?
