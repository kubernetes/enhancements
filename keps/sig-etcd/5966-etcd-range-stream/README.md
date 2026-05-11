# KEP-5966: etcd RangeStream

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Stream Message Layout](#stream-message-layout)
  - [Supported Options](#supported-options)
  - [Chunk Sizing](#chunk-sizing)
  - [Unsupported Pass Through](#unsupported-pass-through)
  - [Implementation Changes](#implementation-changes)
  - [Test Plan](#test-plan)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
  - [Kubernetes API Server Integration](#kubernetes-api-server-integration)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
- [Implementation History](#implementation-history)
<!-- /toc -->

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
key-count limit. The first chunk uses a small initial limit; after
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

### Test Plan

##### Unit tests

- Server-side `RangeStream` chunking, revision pinning, and
  `CountOnly` short-circuit behavior.

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

### Kubernetes API Server Integration

RangeStream is gated behind a `RangeStream` feature gate in kube-apiserver
(Alpha in 1.37, default disabled).

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

### Upgrade / Downgrade Strategy

RangeStream is a new server-side RPC. Older clients that do not call
`RangeStream` are completely unaffected. On downgrade to an etcd version
without `RangeStream`, clients calling the RPC will receive an
`Unimplemented` gRPC error and should fall back to unary `Range`.

## Implementation History

- 2026-03-18: KEP created

