# KEP-1693: Warning mechanism for use of deprecated APIs

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Server-side changes](#server-side-changes)
  - [Client-side changes](#client-side-changes)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Graduation Criteria](#graduation-criteria)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

- [x] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [ ] KEP approvers have approved the KEP status as `implementable`
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This enhancement makes it easier for users and cluster administrators
to recognize and remedy use of deprecated APIs.
Users are presented with informative warnings at time of use.
Administrators are given metrics that show deprecated API use,
and audit annotations that can be used to identify particular API clients.

## Motivation

Kubernetes has many deprecations in flight at any given moment, in various stages, with various time horizons.
Keeping track of all of them is difficult, and has historically required careful reading of release notes,
and manually sweeping for use of deprecated features.

### Goals

* When a user makes a request to a deprecated API, present them with a warning
  that includes the target removal release and any replacement API
* Allow a cluster administrator to programatically determine if deprecated APIs are being used:
  * Filtered to particular APIs
  * Filtered to particular operations (e.g. get, list, create, update)
  * Filtered to APIs targeting removal in particular releases
* Allow a cluster administrator to programatically identify particular clients using deprecated APIs

### Non-Goals

While the proposed warning mechanism is generic enough to carry arbitrary warnings,
the following items are out of scope for the first iteration of this feature:

* Allowing extensions mechanisms like admission webhooks to contribute warnings
* Surfacing warnings about other non-fatal problems
  (for example, [problematic API requests](http://issue.k8s.io/64841#issuecomment-395141013)
  that cannot be rejected for compatibility reasons)

## Proposal

### Server-side changes

When a deprecated API is used:

1. Add a `Warning` header to the response
2. Increment a counter metric with labels for the API group, version, resource, API verb, and target removal major/minor version
3. Record an audit annotation indicating the request used a deprecated API

### Client-side changes

In client-go:

1. Parse `Warning` headers from server responses
2. Provide default warning handler implementations, e.g.:
   1. ignore warnings
   2. dedupe warnings
   3. log warnings with klog.Warn
3. Add a process-wide warning handler (defaulting to the klog.Warn handler)
4. Add a per-client warning handler (defaulting to the process-wide warning handler)

In kubectl, configure the per-process handler to:

1. dedupe warnings (only print a given warning once per invocation)
2. log to stderr with a `Warning:` prefix
3. color the `Warning:` prefix if stderr is a terminal and `$TERM != "dumb"` and `$NO_COLOR` is unset

In kube-apiserver and kube-controller-manager, configure the process-wide handler to ignore warnings

## Design Details

Server-side:

* Add a handler chain filter that attaches a WarningRecorder implementation to the request context
* Add a WarningRecord implementation that deduplicates the warning message per request and writes a `Warning` header
  * the header structure is defined in [RFC2616#14.46](https://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.46)
  * warnings would be written with code `299` and warn-agent of `-`
* In the endpoint installer, decorate handlers for deprecated APIs:
  * add the warning header
  * increment the counter for the deprecated use metric
  * add an audit annotation indicating the request was to a deprecated API

Client-side:

* Parse `Warning` headers in server responses
* Ignore malformed warning headers and warnings with codes other than `299`,
  ensuring that this enhancement will not cause any previously successful API request to fail
* Add the parsed warning headers to the `rest.Result` struct
* Pass parsed warnings through the per-client or per-process warning handler

### Test Plan

- Unit tests
  - `Warning` header generation
  - `Warning` header parsing (including tolerating/ignoreing malformed headers)
  - per-process / per-client warning handler precedence is honored
- Integration tests
  - warning headers are returned when making requests to deprecated APIs
  - deprecated metrics are incremented when making requests to deprecated APIs
  - audit annotations are added when making requests to deprecated APIs

### Risks and Mitigations

**Metric cardinality**

In the past, we have had problems with unbounded metric labels increasing cardinality of 
metrics and causing significant memory/storage use. Limiting these metrics to bounded values
(API group, version, resource, API verb, target removal release) and omitting unbounded values
(resource instance name, client username, etc), metric cardinality is controlled.

Annotating audit events for the deprecated API requests allows an administrator to locate
the particular client making deprecated requests when metrics indicate an investigation is needed.

**Additional stderr / warning output**

Additional warning messages may be unexpected by kubectl or client-go consumers.
However, kubectl and client-go already output warning messages to stderr or via `klog.Warn`.
client-go consumers can programmatically modify or suppress the warning output at a per-process or per-client level.

### Graduation Criteria

The structure of the `Warning` header is RFC-defined and unversioned.
The RFC defines the behavior of the `299` warning code as follows:

> The warning text can include arbitrary information to be presented to a human user or logged.
> A system receiving this warning MUST NOT take any automated action.

Because the server -> client warning format is fixed, and the warnings do not
drive automated action in clients, graduation criteria is primarily oriented
toward the stability level of the administrator metrics, and the ability to 
disable the server sending warnings during the beta period.

#### Beta

* API server output of `Warning` headers for deprecated API use is feature-gated and enabled by default
* Server metric for deprecated API use is marked as beta-level
* client-go logs warnings by default
* kubectl outputs warnings to stderr

#### GA

* API server output of warning headers for deprecated API use is unconditionally enabled
* Server metric for deprecated API use is marked as stable

### Upgrade / Downgrade Strategy

client-go consumers wanting to suppress default warning messages would need to override the per-process warning handler.
Note that client-go already [logs warning messages](https://grep.app/search?q=klog.Warn&filter[repo][0]=kubernetes/client-go).

### Version Skew Strategy

Old clients making requests to a new API server ignore `Warning` headers.

New clients making requests to old API servers handle requests without `Warning` headers normally.

## Implementation History

- 2020-04-16: KEP introduced
