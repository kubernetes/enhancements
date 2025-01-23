# API gzip compression support

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [1.16 Beta](#116-beta)
  - [Revisit Beta](#revisit-beta)
  - [1.33 GA](#133-ga)
  - [Implementation Details](#implementation-details)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
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

### 1.16 Beta

* Update the existing incomplete alpha API compression to:
  * Only occur on API requests
  * Only occur on very large responses (>128KB)
* Promote to beta and enable by default since this is a standard feature of HTTP servers
  * Test at large scale to mitigate risk of regression, tune as necessary

### Revisit Beta

There is a [revist issue](https://github.com/kubernetes/kubernetes/issues/112296) by @shyamjvs (https://docs.google.com/document/d/1rMlYKOVyujboAEG2epxSYdx7eyevC7dypkD_kUlBxn4/edit?tab=t.0)

- v1.26 [Reduce default gzip compression level from 4 to 1 in apiserver](https://github.com/kubernetes/kubernetes/pull/112299)
- v1.25 [Add flag to disable compression for local traffic](https://github.com/kubernetes/kubernetes/pull/111507)
- v1.26 [Add a "DisableCompression" option to kubeconfig](https://github.com/kubernetes/kubernetes/pull/112309)
- v1.26 [Add --disable-compression flag to kubectl](https://github.com/kubernetes/kubernetes/pull/112580)

### 1.33 GA

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
disabled until 1.33.

Some clients may be requesting gzip and not be correctly handling gzipped responses. An effort should
be made to educate client authors that this change is coming, but in general we do not consider
incorrect client implementations to block implementation of standard HTTP features. The easy mitigation
for many clients is to disable sending `Accept-Encoding` (Go is unusual in providing automatic
transparent compression in the client ecosystem - many client libraries still require opt-in behavior).

## Graduation Criteria

Transparent compression must be implemented in the more focused fashion described in this KEP. The
scalability sig must sign off that the chosen limit (128KB) does not cause a regression in 5000 node
clusters, which may cause us to revise the limit up.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: APIResponseCompression
  - Components depending on the feature gate: kube-apiserver
- [x] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane? No.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? No.

### Monitoring Requirements

<!--
For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

A new `--disable-compression` flag has been added to kubectl (default = false). When true, it opts out of response compression for all requests to the apiserver. This can help improve list call latencies significantly when client-server network bandwidth is ample (>30MB/s) or if the server is CPU-constrained.

A new "DisableCompression" field (default = false) has been added to kubeconfig under cluster info. When set to true, clients using the kubeconfig opt out of response compression for all requests to the apiserver. This can help improve list call latencies significantly when client-server network bandwidth is ample (>30MB/s) or if the server is CPU-constrained.

New flag `--disable-compression-for-client-ips` can be used to control client address ranges for which traffic shouldn't be compressed.

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: N/A
- [ ] API .status
  - Condition name: N/A
  - Other field: N/A
- [ ] Other (treat as last resort)
  - Details:N/A

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?


###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name: N/A
  - [Optional] Aggregation method: N/A
  - Components exposing the metric: N/A
- [ ] Other (treat as last resort)
  - Details: N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

### Scalability

<--
For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

In 1.26, kube-apiserver: gzip compression switched from level 4 to level 1 to improve large list call latencies in exchange for higher network bandwidth usage (10-50% higher). This increases the headroom before very large unpaged list calls exceed request timeout limits.

For the change, there is a detailed doc showing why this change is safe and useful - https://docs.google.com/document/d/1rMlYKOVyujboAEG2epxSYdx7eyevC7dypkD_kUlBxn4

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

This is mentioned in Risk and Mitigations.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The unavailability of the API server or etcd will result in errors or timeouts for clients,
and the gzip compression feature will not mitigate these issues. 

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

If SLOs are not being met, we should first check the CPU usage of the kube-apiserver.
If the CPU usage is high, we should consider disabling compression for the clients.

## Implementation History

* 1.7 Kubernetes added alpha implementation behind disabled flag
* Updated proposal with more scoped implementation for Beta in 1.16 that addresses prior issues
