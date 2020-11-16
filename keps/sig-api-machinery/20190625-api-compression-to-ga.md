---
title: Graduate API gzip compression support to GA
authors:
  - "@smarterclayton"
owning-sig: sig-api-machinery
participating-groups:
  - sig-cli
reviewers:
  - "@lavalamp"
  - "@liggitt"
approvers:
  - "@liggitt"
  - "@lavalamp"
editor: TBD
creation-date: 2019-03-22
last-updated: 2019-03-22
status: implementable
see-also:
  - "https://github.com/kubernetes/kubernetes/issues/44164"
replaces:
superseded-by:
---

# Graduate API gzip compression to GA

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [1.16](#116)
  - [1.17](#117)
  - [Implementation Details](#implementation-details)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

Kubernetes sometimes returns extremely large responses to clients outside of its local network, resulting in long delays for components that integrate with the cluster in the list/watch controller pattern. Kubernetes should properly support transparent gzip response encoding, while ensuring that the performance of the cluster does not regress for small requests.

## Motivation

In large Kubernetes clusters the size of protobuf or JSON responses may exceed hundreds of megabytes, and clients that are not on fast local networks or colocated with the master may experience bandwidth and/or latency issues attempting to synchronize their state with the server (in the case of custom controllers). Many HTTP servers and clients support transparent compression by use of the `Accept-Encoding` header, and support for gzip can reduce total bandwidth requirements for integrating with Kubernetes clusters for JSON by up to 10x and for protobuf up to 8x.

### Goals

Allow standard HTTP transparent `Accept-Encoding: gzip` behavior to work for large Kubernetes API requests, without impacting existing Go language clients (which are already sending that header) or causing a performance regression on the Kubernetes apiservers due to the additional CPU necessary to compress small requests.

### Non-Goals

* Support other compression formats like Snappy due to limited client support
* Compress non-API responses
* Compress watch responses

## Proposal

### 1.16

* Update the existing incomplete alpha API compression to:
  * Only occur on API requests
  * Only occur on very large responses (>128KB)
* Promote to beta and enable by default since this is a standard feature of HTTP servers
  * Test at large scale to mitigate risk of regression, tune as necessary

### 1.17

* Promote to GA


### Implementation Details

Kubernetes has had an alpha implementation of transparent gzip encoding since 1.7. However, this
implementation was never graduated because it caused client misbehavior and the issues were not resolved.

After reviewing the code, the problems in the prior implementation were that it attempted to globally
provide transparent compression as an HTTP middleware component at a much higher level than was necessary.
The bugs that prevented enablement involved double compression of nested responses and failures to
correctly handle flushing of lower level primitives. We do not need to GZIP compress all HTTP endpoints
served by the Kubernetes API server (such as watch requests, exec requests, OpenAPI endpoints which provide
their own compression). Our implementation may satisfy its goals of reducing latency for large requests if
we narrowly scope compression to only those endpoints that need compression.

A further complexity is that the standard Go client library (which Kubernetes has leveraged since 1.0)
always requests compression. Performance testing showed that enabling compression for all suitable
API responses (objects returned via GET, LIST, UPDATE, PATCH) caused a significant performance regression
in both CPU usage (2x) and tail latency (2-5x) on the Kubernetes apiservers. This is due to the additional
CPU costs for performing compression, which impacts tail latency of small requests due to increased
apiserver load. Since forcing all clients in the ecosystem to disable transparent compression by default
is impractical and cannot be done in a gradual manner, we need to apply a more suitable heuristic than
"did the client request transparent compression". According to the HTTP spec, a server may ignore an
`Accept-Encoding` header for any reason, which means we decide *when* we want to compress, not just
whether we compress.

The preferred approach is to only compress responses returned by the API server when encoding objects
that are large enough for compression to benefit the client but not unduly burden the server. In general,
the target of this optimization is extremely large LIST responses which are usually multiple megabytes
in size. These requests are infrequent (<1% of all reads) and when network bandwidth is lower than typical
datacenter speeds (1 GBps) the benefit in reduced latency for clients outweighs the slightly higher CPU
cost for compression.

We experimentally determined a size cut-off for compression that caused no regression on the Kubernetes
density and load tests in either 99th percentile latency or kube-apiserver CPU usage of 128KB, which is
roughly the size of 50 average pods (2.2kb from a large Kubernetes cluster with a diverse workload). This
implementation applies this specific heuristic to the place in the Kubernetes code path where we encode
the body of a response from a single input `[]byte` buffer due to how Kubernetes encodes and manages
responses, which removes the side-effects and unanticipated complexity in the prior implementation.

Given that this is standard HTTP server behavior and can easily be tested with unit, integration, and
our complete end-to-end test suite (due to all of our clients already requesting gzip compression),
there is minimal risk in rolling this out directly to GA. We suggest preserving the feature gate so that
an operator can disable this behavior if they experience a regression in highly-tuned large-scale deployments.

### Risks and Mitigations

The primary risk is that an operator running Kubernetes very close to the latency and tolerance limits
on a very large and overloaded Kubernetes apiserver who runs an unusually high percentage of large
LIST queries on high bandwidth networks would experience higher CPU use that causes them to hit a CPU
limit. In practice, the cost of gzip proportional to the memory and CPU costs of Go memory allocation
on very large serialization and deserialization grows sublinear, so we judge this unlikely. However,
to give administrators an opportunity to react, we would preserve the feature gate and allow it to be
disabled until 1.17.

Some clients may be requesting gzip and not be correctly handling gzipped responses. An effort should
be made to educate client authors that this change is coming, but in general we do not consider
incorrect client implementations to block implementation of standard HTTP features. The easy mitigation
for many clients is to disable sending `Accept-Encoding` (Go is unusual in providing automatic
transparent compression in the client ecosystem - many client libraries still require opt-in behavior).

## Graduation Criteria

Transparent compression must be implemented in the more focused fashion described in this KEP. The
scalability sig must sign off that the chosen limit (128KB) does not cause a regression in 5000 node
clusters, which may cause us to revise the limit up.

## Implementation History

* 1.7 Kubernetes added alpha implementation behind disabled flag
* Updated proposal with more scoped implementation for Beta in 1.16 that addresses prior issues
