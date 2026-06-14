# KEP-5966: etcd RangeStream

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Stream Message Layout](#stream-message-layout)
  - [Supported Options](#supported-options)
  - [Chunk Sizing](#chunk-sizing)
  - [Unsupported Pass Through](#unsupported-pass-through)
  - [Implementation Changes](#implementation-changes)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Kubernetes API Server Integration](#kubernetes-api-server-integration)
  - [Graduation Criteria](#graduation-criteria)
    - [Beta](#beta)
    - [GA](#ga)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The unary Range RPC in etcd builds the entire response in memory before
sending. For large result sets this causes server-side memory spikes (the KV
slice, serialized protobuf, and gRPC send buffer all coexist) and redundant
work when clients paginate (repeated Range calls with increasing keys
recompute the total count on every page by walking the full B-tree index).

This KEP proposes a new server-streaming `RangeStream` RPC that reuses
`RangeRequest` and returns results in chunks. The server performs pagination
internally with adaptive chunk sizing and pins to a single MVCC revision for
consistency. The total count is derived from the running tally of streamed
keys, eliminating the separate index traversal required by paginated Range.

The feature requires etcd 3.7+. With older etcd, kube-apiserver detects
`Unimplemented` and continues using unary `Range` with no behavior change.

## Motivation

The current unary Range RPC has two key problems at scale:

1. **Server-side memory spikes** — the entire response (KV slice, serialized
   protobuf, gRPC send buffer) must coexist in memory before sending.
2. **Redundant work with client-side pagination** — each paginated Range call
   recomputes the total count by walking the full B-tree index, turning
   per-page cost into O(total_keys) instead of O(limit).

### Goals

- Reduce server-side memory usage for large Range responses by streaming
  results in chunks instead of buffering the entire response.
- Eliminate redundant count computation across paginated requests by deriving
  the total from the keys visited during streaming.

### Non-Goals

- Supporting custom sort orders in streaming mode. Clients that need
  non-default sort order should use the existing unary `Range` RPC.

## Proposal

Add a server-streaming `RangeStream` RPC to the etcd KV service that accepts
the existing `RangeRequest` and returns a stream of `RangeStreamResponse`
messages. The server handles pagination internally, pins to a single MVCC
revision for snapshot consistency, and uses adaptive chunk sizing to
auto-tune for different value sizes. The merged stream produces identical
results to a single unary `Range()` call.

If the pinned revision is compacted during streaming, the server closes the
stream with `ErrCompacted`. Clients receive this error from `Recv()` and
should retry the request.

### Notes/Constraints/Caveats

- Clients should not depend on the internal structure of the stream message
  layout (which chunks contain which fields). The contract is that
  `proto.Merge()` across all chunks produces a result identical to a single
  `Range()` call. Clients must merge chunks sequentially in stream order.
- The server opens a new short-lived bbolt read transaction for each chunk
  rather than holding a single long-running transaction for the entire stream.
  Consistency is maintained by pinning the MVCC revision after the first chunk
  and reusing it for all subsequent chunks. If the pinned revision has been
  compacted by the time a later chunk is read, the server returns
  `ErrCompacted` and the client retries. This per-chunk transaction model
  avoids the bbolt caveat where long-running read transactions can block write
  transactions when the database needs to remap/allocate new pages.

### Risks and Mitigations

- **Stream interrupted by compaction.** If the pinned revision is compacted
  mid-stream, the server returns `ErrCompacted`. kube-apiserver surfaces
  this as a watch cache initialization failure, and the cacher retries
  initialization. The behavior is identical to a paginated list that races
  compaction. A unary `Range` without pagination technically avoids this
  problem, because a single non-streaming call does not pin a revision
  across multiple reads. But at the scale where a stream takes longer than
  kube-apiserver's compaction interval (5 minutes by default) to complete,
  the unary response approaches protobuf's 2 GB message limit and is
  itself unlikely to succeed.

## Design Details

### API Changes

A new `RangeStream` RPC is added to the KV service, along with a new
`RangeStreamResponse` wrapper message:

```protobuf
service KV {
  rpc RangeStream(RangeRequest) returns (stream RangeStreamResponse) {}
}

message RangeStreamResponse {
  option (versionpb.etcd_version_msg) = "3.7";
  RangeResponse range_response = 1;
}
```

`RangeStreamResponse` wraps `RangeResponse` so that `proto.Merge()` across
all chunks produces the same result as a single `Range()` call. The wrapper
also leaves room for future streaming-specific fields (e.g., progress,
mid-stream errors).

### Stream Message Layout

| Message              | Contents                                              |
|----------------------|-------------------------------------------------------|
| Intermediate chunks  | Kvs only                                              |
| Final chunk          | Header (ClusterId, MemberId, RaftTerm, Revision), Kvs, Count, More |

Count is included in the final chunk because the server already visits
every key during streaming, so the total is a free byproduct of the
stream itself—no additional tree traversal is needed. Placing count on
the first chunk would require an upfront O(total_keys) index walk before
any data flows. Clients reassemble by merging all messages.

Clients seeking parity with unary `Range` do not need to inspect this
layout. Merging the chunks via `proto.Merge` yields a `RangeResponse`
identical to the unary result. The layout matters only for clients
that want to process the stream chunk by chunk as it arrives.

### Supported Options

`RangeStream` accepts the full `RangeRequest` message and supports all
fields (e.g., `limit`, `keys_only`, `count_only`, `min_mod_revision`,
`max_mod_revision`, `min_create_revision`, `max_create_revision`)
except non-default sort order. Requests with non-default sort order
require server-side post-processing that defeats streaming. The server
returns `Unimplemented` for these requests and clients should use the
unary `Range` RPC instead.

### Chunk Sizing

The streaming handler reuses the existing range path with an adaptive
key-count limit. The first chunk uses a small initial limit. After
each chunk the limit is doubled or halved based on the previous
response size relative to a target derived from `MaxRequestBytes`,
letting the server converge on chunk sizes appropriate for the
observed value sizes without the client having to guess.

### Unsupported Pass Through

- **leasing/kv**, **namespace/kv**, **ordering/kv**, **grpcproxy**:
  return `Unimplemented`. Callers should not use `RangeStream` through
  these wrappers and should use unary `Range` instead.

### Implementation Changes

The following components are modified:

- **Proto** (`api/etcdserverpb/rpc.proto`): New `RangeStream` RPC on the KV
  service. New `RangeStreamResponse` message wrapping `RangeResponse`.
- **v3rpc** (`server/etcdserver/api/v3rpc/key.go`): `kvServer.RangeStream` —
  validates the request and delegates to `EtcdServer.RangeStream`.
- **EtcdServer** (`server/etcdserver/v3_server.go`): `RangeStream` — same auth
  and linearizability path as unary Range. `rangeStream` — the chunking loop:
  adaptive sizing, revision pinning, cursor advancement via
  `append(lastKey, '\x00')`.
- **Client** (`client/v3/kv.go`): `RangeStreamToRangeResponse` — reassembles
  a stream into a single `RangeResponse` so callers can transparently switch
  between unary and streaming.
- **etcdctl**: A `--stream` flag on `etcdctl get` issues a `RangeStream`
  request and reassembles the chunks into a `RangeResponse` for display.
  This provides a direct CLI entry point, useful for inspections and
  comparisons against unary `get` on the same key range.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

No prerequisite test refactors are required. The kube-apiserver integration
reuses the existing watch cache initialization test surface. The etcd
integration extends existing Range test cases with a `RangeStream` parallel
assertion.

##### Unit tests

- Server-side `RangeStream` chunking, revision pinning, and
  `CountOnly` short-circuit behavior.
- kube-apiserver: tests cover the watch cache initialization path
  routing through `RangeStream` and event construction from streamed
  chunks.
- Backwards compatibility: unit tests cover the version skew case where
  the gate is enabled against an etcd that returns `Unimplemented`,
  confirming the fallback to unary `Range` does not regress.

##### Integration tests

- Integration tests (`tests/integration/v3_grpc_test.go`) — every existing
  Range test case also calls `RangeStream` and diffs the reassembled response
  against the unary result.
- Transparent transform from `Get` to use `RangeStream` to add subtests
  for all existing Get tests (`tests/integration/cache_test.go`,
  `tests/common/kv_test.go`).
- Robustness tests (Note: `tests/robustness/coverage` in the etcd repository
  will need updating once Kubernetes actually starts making calls, as monitored
  in [ci-etcd-k8s-coverage-amd64](https://testgrid.k8s.io/sig-etcd-periodics#ci-etcd-k8s-coverage-amd64)).
- kube-apiserver: integration tests will be added for correctness and
  benchmarks against the unary `Range` path.

##### e2e tests

e2e tests will be added to exercise the streaming behavior.

### Kubernetes API Server Integration

RangeStream is gated behind a `EtcdRangeStream` feature gate in kube-apiserver
(Beta in 1.37).

The primary integration point is the watch cache initialization path.
When the feature gate is enabled, the watch cache `sync()` uses
`KV.GetStream` to receive chunks incrementally. Each chunk's key-value
pairs are converted to synthetic "created" events and queued inline
without assembling the full list response in memory.

For direct `GetList` calls (e.g., from controllers or when WatchList is
disabled), the store consumes `KV.GetStream` and decodes each chunk's
key-value pairs inline as they arrive, overlapping network I/O with
decode. When the server returns `Unimplemented` or the feature gate is
disabled, the store falls back to paginated `List` with a conservative
limit. The `storage.Interface` is unchanged.

### Graduation Criteria

#### Beta

- Feature implemented behind the `EtcdRangeStream` feature gate, default on
  in Beta.
- Watch cache initialization routes through `RangeStream` with
  automatic `Unimplemented` fallback for older etcd.
- Feature is covered with unit, integration, and e2e tests.
- Scalability test on a 5000-node cluster measuring latency on large
  list responses.
- `etcdctl get --stream` available for direct CLI access to `RangeStream`.

#### GA

- Conformance tests.
- At least two cloud providers have run the feature in production and
  confirmed it is stable.

### Upgrade / Downgrade Strategy

RangeStream is a new server-side RPC. Older clients that do not call
`RangeStream` are completely unaffected. On downgrade to an etcd version
without `RangeStream`, clients calling the RPC will receive an
`Unimplemented` gRPC error and should fall back to unary `Range`.

### Version Skew Strategy

Skew is between kube-apiserver and etcd. The supported combinations:

- gate on + etcd 3.7 or newer: uses `RangeStream`.
- gate on + etcd 3.6 or older: falls back to unary `Range`.
- gate off + any etcd version: uses unary `Range`.

We will reuse kube-apiserver's existing
`storage.FeatureSupportChecker` to cache the feature result and avoid
calling the streaming endpoint repeatedly when running against an
older etcd version. The checker probes etcd periodically, so an
etcd upgrade from 3.6 to 3.7 is picked up by a running kube-apiserver
without a restart. If the cached response is incorrect (eg: cluster
downgrade), the `Unimplemented` fallback will still fall back to
`Range`.

For HA kube-apiserver during a rolling upgrade, each apiserver
negotiates with etcd on its own. An apiserver on the old version uses
unary `Range` and one on the new version uses `RangeStream` when
available. Watch cache state is per-apiserver, so they don't interact.
For an etcd cluster rolling upgrade, requests may land on different
members during the transition. Once a `RangeStream` call fails,
kube-apiserver falls back to unary `Range`. For an etcd downgrade
after kube-apiserver has started using RangeStream, the fallback is
always live. The next call returns `Unimplemented` and kube-apiserver
reverts to unary `Range`. Nothing on the apiserver side needs to be
migrated.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `EtcdRangeStream`
  - Components depending on the feature gate: `kube-apiserver`

On the etcd side, the feature is always on at v3.7+.

###### Does enabling the feature change any default behavior?

Watch cache initialization internally switches from a paginated `Range` loop to a single
`RangeStream` call. The data returned is the same and there is no visible user facing change.
It is estimated that etcd memory and watch cache initialization time will improve.

Clusters on etcd 3.6 or older are unaffected: the gate is on but
kube-apiserver falls back to unary `Range` on the first `Unimplemented`
response.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Setting `EtcdRangeStream=false` and restarting kube-apiserver fully
disables the feature. The feature is stateless.

###### What happens if we reenable the feature if it was previously rolled back?

Same as enabling the feature for the first time.

###### Are there any tests for feature enablement/disablement?

Yes, testing will be added for parity of watch cache behavior with
RangeStream on and off. Skew against an etcd without `RangeStream` is
covered by unit tests that inject an `Unimplemented` response and
assert kube-apiserver falls back to paginated `Range`, plus manual
verification against older etcd binaries.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

No known failure modes.

###### What specific metrics should inform a rollback?

- `apiserver_watch_cache_initialization_duration_seconds{group, resource}`
  increases significantly compared to average.
- `etcd_request_duration_seconds{operation="listStream"}` p99 reports extremely high numbers.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Yes. We will specifically test the upgrade, downgrade, and upgrade
path. Tests cover both the startup `storage.FeatureSupportChecker`
decision and the runtime `Unimplemented` fallback.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Check whether `etcd_request_duration_seconds_count{operation="listStream"}`
is present and greater than zero. If the gate is on but the metric is
absent or stays at zero, the etcd server is returning `Unimplemented`
(look for the `etcd server does not support RangeStream` warning in
apiserver logs).

###### How can someone using this feature know that it is working for their instance?

  - Details: This is a backend-only optimization. It has no per-object
    user-visible signal. Operators verify behavior via two metrics:
    1. `etcd_request_duration_seconds_count{operation="listStream"}`
       increments after each watch cache sync.
    2. `apiserver_watch_cache_initialization_duration_seconds` for the
       affected resources is lower than the pre-enablement baseline.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

No regression from the previous behavior of using unary `Range`.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - Metric name: `apiserver_watch_cache_initialization_duration_seconds`
  - Components exposing the metric: kube-apiserver
- [X] Metrics
  - Metric name: `etcd_request_duration_seconds`
  - Components exposing the metric: kube-apiserver

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

n/a.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

The feature depends on the etcd server supporting the `RangeStream`
RPC. This is an version dependency (etcd 3.7+), not an additional
cluster service.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes, but they replace existing calls rather than adding new traffic.
On a kube-apiserver with the gate on:

- API call type: `RangeStream` (gRPC server-streaming) to etcd.
- Estimated throughput: one `RangeStream` request instead of `Range` request for list requests to etcd. In cases of pagination, N range requests are condensed into one RangeStream request.`
- Originating component: kube-apiserver.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

[slo]: https://github.com/kubernetes/community/blob/master/sig-scalability/slos/slos.md

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

- **etcd unavailable:** identical to today. The watch cache
  initialization fails, the cacher retries with backoff.
- **kube-apiserver unavailable:** The feature will not function
  without kube-apiserver availability.

###### What are other known failure modes?

- **Stream interrupted by compaction**
  - Detection: apiserver logs include `mvcc: required revision has been
    compacted`.
  - Mitigations: kube-apiserver retries the watch cache initialization
    with a newer revision.
  - Diagnostics: structured error from etcd is propagated.
  - Testing: covered by etcd integration tests in
    `tests/integration/v3_grpc_test.go`.

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Confirm the feature is in use:
   `etcd_request_duration_seconds_count{operation="listStream"}` > 0.
2. Compare `apiserver_watch_cache_initialization_duration_seconds` to the
   pre-enablement baseline for the affected `group, resource`. If the
   metric regressed, disable the gate.

## Implementation History

- 2026-03-18: KEP created
- 2026-05-08: Expanded design with kube-apiserver integration details and
  filled out the production readiness review for beta targeting v1.37.

