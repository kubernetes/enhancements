# KEP-6051: CRI Streaming Authorization Hook

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Defense in Depth](#story-1-defense-in-depth)
    - [Story 2: Audit Logging for Compliance](#story-2-audit-logging-for-compliance)
    - [Story 3: Custom Runtime Authorization](#story-3-custom-runtime-authorization)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
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

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/guide/README.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Adds an optional `AuthorizeStream` hook to the CRI streaming server configuration.
The hook is invoked after a short-lived stream URL token is validated and before
exec, attach, or port-forward streams are served. When `nil`, existing behavior
is preserved — making this fully backward-compatible.

## Motivation

The CRI streaming package (`k8s.io/cri-streaming`) currently relies entirely on
a single-use, short-lived random token (6 bytes of crypto/rand, base64url-encoded,
1-minute TTL, max 1000 in-flight) for stream request security. There is no
authentication or authorization at the streaming server level, as noted by the
long-standing TODO in `server.go`:

```
// TODO(tallclair): Add auth(n/z) interface & handling.
func NewServer(config Config, runtime Runtime) (Server, error) {
```

The current security model assumes:
1. The kubelet has already authorized the original CRI request before calling
   the runtime to generate a streaming URL.
2. The streaming server is only accessible on localhost.
3. The single-use token is sufficient to prevent replay attacks.

However, these assumptions do not cover all deployment scenarios. A CRI runtime
that listens on a non-loopback interface, or a token that leaks via logs or
network monitoring, leaves the streaming endpoint without an independent
authorization check.

### Goals

- Provide a stream-time authorization point in the CRI streaming server
- Allow CRI runtime implementers to enforce custom authorization policies
- Maintain full backward compatibility (nil hook = existing behavior)
- Enable audit logging of streaming access at the CRI shim level

### Non-Goals

- Change the existing token-based request cache mechanism
- Define a specific authorization policy (that is the implementer's responsibility)
- Add authentication (only authorization at stream time)
- Modify the CRI gRPC API

## Proposal

Add an `AuthorizeStream func(http.ResponseWriter, *http.Request) error` field
to the `streaming.Config` struct. When non-nil, the server calls this function
after token validation succeeds and before serving the stream. If the function
returns a non-nil error, the request is rejected with `403 Forbidden`.

### Call Order

1. HTTP request arrives at `/exec/<token>` (or `/attach/`, `/portforward/`)
2. Token validated via `cache.Consume(token)` — 404 if invalid or expired
3. **NEW**: `AuthorizeStream(w, r)` called — 403 if rejected
4. Stream served via `ServeExec()` / `ServeAttach()` / `ServePortForward()`

### User Stories

#### Story 1: Defense in Depth

As a CRI runtime implementer, I want to verify authorization at the streaming
boundary so that even if the kubelet's authorization is bypassed (e.g., token
leak via logs), the streaming server independently rejects unauthorized
connections.

#### Story 2: Audit Logging for Compliance

As a cluster operator in a regulated environment, I need to log all streaming
access (exec/attach/port-forward) at the CRI shim level for audit purposes.
The `AuthorizeStream` hook provides an injection point to record who accessed
which container stream and when, without modifying the kubelet.

#### Story 3: Custom Runtime Authorization

As a CRI runtime vendor, I want to enforce runtime-specific authorization
policies (e.g., container-level RBAC, namespace isolation) at stream time,
without modifying the kubelet or the CRI gRPC API.

### Notes/Constraints/Caveats

- The hook runs on every streaming request, so it should be fast to avoid
  adding latency to exec/attach/port-forward setup.
- The hook receives the standard `http.ResponseWriter` and `http.Request`,
  giving implementers access to headers, TLS peer certificates, and request
  metadata for authorization decisions.
- The hook is called **after** token validation, so it does not need to
  validate the token itself.

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Hook returns error for legitimate requests | Hook is opt-in via config; nil = no-op. Implementers control the policy. |
| Hook adds latency to stream setup | Document that hooks should be fast. Add metrics for authorization duration. |
| Hook panics, crashing the server | Document that implementers should handle panics. Consider recover() wrapper in future iterations. |
| Breaking change if Config struct is used with positional initialization | Use named field initialization (already the standard Go practice for structs with many fields). |

## Design Details

### Config Change

```go
type Config struct {
    // ... existing fields ...

    // AuthorizeStream, if non-nil, is called after the stream token is
    // validated and before the stream is served. If it returns a non-nil
    // error, the request is rejected with HTTP 403 Forbidden.
    AuthorizeStream func(http.ResponseWriter, *http.Request) error
}
```

### Server Integration

In each of the three serve handlers (`serveExec`, `serveAttach`, `servePortForward`),
the hook is called immediately after `cache.Consume(token)` succeeds:

```go
func (s *Server) serveExec(req *restful.Request, resp *restful.Response) {
    token := req.PathParameter("token")
    cachedRequest, err := s.cache.Consume(token)
    if err != nil {
        // ... handle 404 ...
        return
    }
    // NEW: stream-time authorization
    if s.config.AuthorizeStream != nil {
        if err := s.config.AuthorizeStream(resp, req.Request); err != nil {
            http.Error(resp, err.Error(), http.StatusForbidden)
            return
        }
    }
    // ... serve the stream ...
}
```

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

No prerequisite testing updates are required. The existing streaming server tests
provide a solid foundation for testing the new hook.

##### Unit tests

- `k8s.io/cri-streaming/pkg/streaming`: test AuthorizeStream for:
  - nil hook (default, backward-compatible path)
  - hook returns nil (allow)
  - hook returns error (reject with 403)
  - hook is called after token validation (not called for invalid tokens)
- Coverage target: >90% for new code paths

- `k8s.io/cri-streaming/pkg/streaming`: `<date>` - `<test coverage>`

##### Integration tests

Integration tests are not strictly necessary for this change since the hook is
a simple callback that does not interact with other Kubernetes components.
The unit tests and node e2e tests provide sufficient coverage.

##### e2e tests

- Node e2e tests for exec/attach/port-forward with:
  - Feature gate disabled (default behavior preserved)
  - Feature gate enabled, hook allows the request
  - Feature gate enabled, hook denies the request (expect 403)

### Graduation Criteria

#### Alpha

- Feature implemented behind `CRIStreamingAuthorization` feature gate
- Unit tests with >90% coverage on new code paths
- Node e2e tests for hook allow/deny paths
- At least one CRI runtime (containerd or cri-o) expresses interest

#### Beta

- Feedback from at least 2 CRI runtime implementations (containerd, cri-o)
- Integration tests stable on testgrid
- Metrics for authorization requests and latency available
- All known issues from alpha resolved

#### GA

- 2 releases since beta
- All known issues resolved
- Demonstrated usage in at least 2 production-grade CRI runtimes

### Upgrade / Downgrade Strategy

- **Upgrade**: The `AuthorizeStream` field defaults to `nil`. Existing
  configurations without the field work without changes. No migration needed.
- **Downgrade**: Removing the field from Config is backward-compatible since
  Go ignores unknown struct fields during unmarshaling. The feature gate can
  be disabled to revert to previous behavior.

### Version Skew Strategy

This enhancement only affects the CRI streaming server within the same node.
There is no cross-component coordination required:
- The kubelet does not need to know whether the CRI shim uses the hook.
- The CRI shim can enable the hook independently of the kubelet version.
- No version skew concerns between control plane and nodes.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `CRIStreamingAuthorization`
  - Components depending on the feature gate: kubelet

###### Does enabling the feature change any default behavior?

No. When `AuthorizeStream` is nil (the default), behavior is identical to
the current implementation. The feature gate only enables the code path
that checks for the hook's presence.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Setting the feature gate to `false` and restarting the kubelet disables
the hook check. Existing streaming connections are not affected (they are
already established).

###### What happens if we reenable the feature if it was previously rolled back?

No side effects. The hook is stateless and evaluated per-request.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests will cover both enabled and disabled states.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A failed rollout would only affect new streaming requests (exec/attach/port-forward).
Already running containers are not affected. If the hook is misconfigured and
rejects all requests, new exec/attach/port-forward operations would fail with 403,
but existing pod operations continue normally.

###### What specific metrics should inform a rollback?

- High rate of `cri_streaming_authorization_requests_total` with status=rejected
- Increased `cri_streaming_authorization_duration_seconds` indicating slow hooks
- Increased exec/attach/port-forward failure rate

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

To be tested during alpha.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- Check if the `CRIStreamingAuthorization` feature gate is enabled on the kubelet.
- Check the `cri_streaming_authorization_requests_total` metric for non-zero values.

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as last resort)
  - CRI runtime logs will indicate when the authorization hook is invoked.
  - The `cri_streaming_authorization_requests_total` metric tracks allowed vs. rejected requests.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- The authorization hook should add <10ms latency to stream setup (p99).
- The hook should not cause stream request failures unless explicitly rejecting
  unauthorized requests.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `cri_streaming_authorization_requests_total`
  - [Optional] Aggregation method: count by status (allowed/rejected)
  - Components exposing the metric: CRI runtime shim
- [x] Metrics
  - Metric name: `cri_streaming_authorization_duration_seconds`
  - [Optional] Aggregation method: histogram
  - Components exposing the metric: CRI runtime shim

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No. The two proposed metrics (request count by status and duration) are sufficient
for monitoring the authorization hook.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. The authorization hook is a callback within the CRI streaming server process.
It may call external services (e.g., an OPA server), but that is the
implementer's choice and not a hard dependency.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new Kubernetes API calls. The hook is a local callback within the CRI shim.
Implementers may choose to make external calls (e.g., to an authorization
service), but that is outside the scope of this KEP.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Stream setup time may increase by the duration of the authorization hook call.
For well-implemented hooks, this should be <10ms (p99).

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The hook is a per-request callback with minimal overhead.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. The hook does not allocate persistent resources.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The hook operates within the CRI shim and does not depend on the API server or
etcd. If an implementer's hook calls the API server, that is their concern.

###### What are other known failure modes?

- **Hook returns error for all requests**: All new streaming requests fail with 403.
  - Detection: High `cri_streaming_authorization_requests_total` with status=rejected.
  - Mitigation: Disable the feature gate or fix the hook implementation.
  - Testing: Unit test for hook error path.
- **Hook takes too long**: Stream setup latency increases.
  - Detection: High `cri_streaming_authorization_duration_seconds`.
  - Mitigation: Implement a timeout in the hook or disable the feature gate.

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check `cri_streaming_authorization_duration_seconds` for elevated latency.
2. Check `cri_streaming_authorization_requests_total` for high rejection rates.
3. Review CRI runtime logs for hook errors.
4. Disable the `CRIStreamingAuthorization` feature gate to restore default behavior.

## Implementation History

- 2026-05-01: KEP opened as provisional
- 2026-04-27: Initial proof-of-concept PR opened (kubernetes/kubernetes#138616)

## Drawbacks

- Adds a new callback to the streaming Config, increasing the API surface of
  `k8s.io/cri-streaming`. However, the change is minimal and backward-compatible.
- CRI runtime implementers must opt in to use the hook, which means the security
  benefit is not automatic.

## Alternatives

1. **Status quo**: Rely solely on kubelet-side authorization and the token-based
   request cache. Rejected because the long-standing TODO acknowledges this gap,
   and the single-token model does not provide defense in depth.

2. **Mutual TLS between kubelet and CRI shim**: Would add operational complexity
   (certificate management) and does not address all threat scenarios (e.g.,
   authorized but logged token replay).

3. **Extend CRI gRPC API with auth metadata**: Would require changes to the CRI
   API proto definitions and all runtime implementations. The hook approach is
   lighter-weight and does not require gRPC changes.

4. **Middleware-based approach**: Use HTTP middleware on the streaming server
   instead of a config field. This would work but is less discoverable and
   harder to document than a named config field.

## Infrastructure Needed (Optional)

None.
