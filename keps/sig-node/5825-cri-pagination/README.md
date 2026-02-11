<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-5825: CRI List Streaming

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Message Size Analysis](#message-size-analysis)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
    - [StreamContainers](#streamcontainers)
    - [StreamPodSandboxes](#streampodsandboxes)
    - [Other Stream Operations](#other-stream-operations)
    - [Behavior Matrix](#behavior-matrix)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
    - [Unit tests](#unit-tests)
    - [Integration tests](#integration-tests)
    - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
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
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Increase gRPC message limit](#increase-grpc-message-limit)
  - [Improve garbage collection](#improve-garbage-collection)
  - [Token-based pagination](#token-based-pagination)
  - [Offset-based pagination](#offset-based-pagination)
- [References](#references)
  - [Standards and Documentation](#standards-and-documentation)
  - [Related Issues and PRs](#related-issues-and-prs)
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
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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

This KEP proposes adding server-side streaming RPCs to CRI's `List*` operations. Currently, these APIs return all results in a single unary response, causing failures on nodes with 10k+ containers when responses exceed the 16 MB gRPC message limit. The proposed solution introduces new streaming RPCs (e.g., `StreamContainers`, `StreamPodSandboxes`) that stream results one item at a time, bypassing the per-message size limit.

## Motivation

CRI's `List*` RPCs currently return all results in a single unary response without any way to break up large result sets. On nodes with many containers, responses exceeding the gRPC limit (currently 16 MB) cause complete RPC failures.

### Message Size Analysis

| Message    | Typical Size | Conservative Estimate |
|------------|--------------|-----------------------|
| Container  | 1.0-1.8 KB   | ~1.5 KB               |
| PodSandbox | 0.9-1.6 KB   | ~1.2 KB               |

**When the 16 MB limit is exceeded:**

| Resource   | Count  | Estimated Size | Exceeds Limit? |
|------------|--------|----------------|----------------|
| Containers | 10,000 | ~15 MB         | No (close)     |
| Containers | 11,000 | ~16.5 MB       | **Yes**        |
| Pods       | 13,000 | ~15.6 MB       | No (close)     |
| Pods       | 14,000 | ~16.8 MB       | **Yes**        |

### Goals

- Add streaming support to `List*` operations via new streaming RPCs
- Maintain backward compatibility with existing kubelet/runtime combinations
- Eliminate the gRPC message size limit as a bottleneck for list operations

### Non-Goals

- Changing the default gRPC message size limit
- Modifying container/pod garbage collection behavior
- Implementing server-side filtering beyond existing `ContainerFilter`/`PodSandboxFilter`
- Modifying the existing unary `List*` RPCs

## Proposal

Add new server-side streaming RPCs alongside the existing unary `List*` RPCs. Each new RPC (e.g., `StreamContainers`, `StreamPodSandboxes`) accepts the same filter as its unary counterpart and streams results back one item per message.

### User Stories

**Story 1: High CronJob Churn**

As a cluster operator running hundreds of CronJobs, I want kubelet to successfully list containers even when thousands of completed job containers exist, so that new pods can be scheduled without hitting gRPC limits.

**Story 2: CI/CD Node**

As a platform engineer managing CI nodes that run many short-lived containers, I want to avoid kubelet failures when container counts spike, so that CI pipelines remain reliable.

### Notes/Constraints/Caveats

1. **Streaming does not reduce kubelet memory usage**: Because kubelet uses a wrapper function that collects all streamed items into a single list before processing, the full result set is still held in memory. Streaming solely addresses the gRPC per-message size limit, not memory pressure.

### Risks and Mitigations

1. **Consistency during modifications**: Containers are created/deleted frequently during streaming. It is the runtime's responsibility to maintain consistency and ensure no duplicate containers appear across the streamed results.

2. **Kubelet atomic view**: Kubelet is designed to see an atomic view of all pods.

   Mitigation: Kubelet creates a wrapper function that receives all streamed items and presents the aggregated result as a single list to the caller.

3. **Stream interruption**: If the runtime crashes or the stream is interrupted mid-transfer, partial results are discarded.

   Mitigation: Kubelet discards any partial results received so far and retries the entire streaming call. This is no worse than the unary RPC failure case.

## Design Details

### API Changes

New server-side streaming RPCs are added to the `RuntimeService` and `ImageService`. The existing unary RPCs remain unchanged for backward compatibility.

#### StreamContainers

**Service definition:**
```protobuf
service RuntimeService {
    // Existing unary RPC (unchanged)
    rpc ListContainers(ListContainersRequest) returns (ListContainersResponse) {}

    // New server-side streaming RPC
    rpc StreamContainers(StreamContainersRequest) returns (stream StreamContainersResponse) {}
}
```

**StreamContainersRequest:**
```protobuf
message StreamContainersRequest {
    ContainerFilter filter = 1;
}
```

**StreamContainersResponse:**
```protobuf
message StreamContainersResponse {
    Container container = 1;
}
```

The runtime sends one `StreamContainersResponse` message per container over the stream.
How the runtime iterates over containers is an implementation detail; the simplest approach is to build the full list (as the unary RPC already does) and iterate over it.
When all containers have been sent, the runtime returns `nil` from the gRPC handler, which closes the stream and signals `io.EOF` to the kubelet.

#### StreamPodSandboxes

**Service definition:**
```protobuf
service RuntimeService {
    // Existing unary RPC (unchanged)
    rpc ListPodSandbox(ListPodSandboxRequest) returns (ListPodSandboxResponse) {}

    // New server-side streaming RPC
    rpc StreamPodSandboxes(StreamPodSandboxesRequest) returns (stream StreamPodSandboxesResponse) {}
}
```

**StreamPodSandboxesRequest:**
```protobuf
message StreamPodSandboxesRequest {
    PodSandboxFilter filter = 1;
}
```

**StreamPodSandboxesResponse:**
```protobuf
message StreamPodSandboxesResponse {
    PodSandbox pod_sandbox = 1;
}
```

#### Other Stream Operations

The same streaming pattern will be applied to the following existing unary RPCs:

| Existing Unary RPC    | New Streaming RPC       | Service        |
|-----------------------|-------------------------|----------------|
| ListContainers        | StreamContainers        | RuntimeService |
| ListPodSandbox        | StreamPodSandboxes      | RuntimeService |
| ListContainerStats    | StreamContainerStats    | RuntimeService |
| ListPodSandboxStats   | StreamPodSandboxStats   | RuntimeService |
| ListPodSandboxMetrics | StreamPodSandboxMetrics | RuntimeService |
| ListImages            | StreamImages            | ImageService   |

Each streaming RPC follows the same pattern: a request message containing the same fields as the existing unary request, and a streamed response where each message contains a single item of the corresponding resource type.

`ListMetricDescriptors` is excluded because it returns a fixed set of metric descriptors whose total size will not approach the gRPC message limit.

#### Behavior Matrix

| Kubelet | Runtime | Behavior                                                                   |
|---------|---------|----------------------------------------------------------------------------|
| Old     | Old     | Unary RPC, full list                                                       |
| Old     | New     | Unary RPC, full list (streaming RPCs exist but are not called)             |
| New     | Old     | Kubelet calls streaming RPC, gets `UNIMPLEMENTED`, falls back to unary RPC |
| New     | New     | Streaming RPC used                                                         |

### Test Plan

- [x] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

#### Prerequisite testing updates

#### Unit tests

- Test kubelet streaming wrapper: verify it correctly aggregates all items from a mock stream into a single list.
- Test fallback behavior: verify kubelet falls back to the unary RPC when the streaming RPC returns `UNIMPLEMENTED`.
- Test stream error handling: verify kubelet discards partial results and returns an error when the stream fails mid-transfer.

#### Integration tests

kubelet does not have integration tests.

#### e2e tests

- End-to-end streaming on nodes with high container counts.
- Using CRI-proxy, verify that:
  - kubelet calls the streaming RPC when the feature gate is enabled.
  - kubelet falls back to the unary RPC when the runtime does not implement the streaming RPC.
  - kubelet correctly aggregates all streamed items into a complete list.
- Cover the [behavior matrix](#behavior-matrix) scenarios (old/new kubelet x old/new runtime).

### Graduation Criteria

#### Alpha

- Feature gate: `CRIListStreaming`
- Streaming RPCs added to CRI proto
- Kubelet implements streaming with fallback to unary RPCs
- Reference implementation in containerd or CRI-O
- Basic unit tests

#### Beta

- Feature gate enabled by default
- Both containerd and CRI-O implement streaming RPCs
- E2E tests passing
- Metrics for streaming usage added
- Documentation updated

#### GA

- All supported runtimes implement streaming RPCs
- Feature gate locked to enabled
- Existing unary RPCs remain available (no removal, it can be a different KEP)

### Upgrade / Downgrade Strategy

**Upgrade path:**
1. Upgrade runtime and kubelet (order-agnostic)

**Downgrade path:**
1. Downgrade runtime and kubelet (order-agnostic)

Upgrade and downgrade operations are order-agnostic because kubelet automatically falls back to unary RPCs when the runtime does not support streaming. The streaming RPCs are additive and do not modify existing RPCs.

### Version Skew Strategy

The fallback mechanism handles all version skew scenarios:

- **New kubelet + Old runtime**: Kubelet calls the streaming RPC. The runtime returns `UNIMPLEMENTED`. Kubelet falls back to the unary RPC and retrieves the full list.
- **Old kubelet + New runtime**: Kubelet uses the existing unary RPC. The streaming RPCs exist on the runtime but are never called.
- **New kubelet + New runtime**: Kubelet uses the streaming RPCs.

Because the streaming RPCs are entirely new RPCs (not modifications to existing ones), there is no risk of field-level incompatibility. The version skew is handled purely through gRPC status codes.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `CRIListStreaming`
  - Components depending on the feature gate: kubelet

###### Does enabling the feature change any default behavior?

No. When enabled, kubelet will use streaming RPCs if the runtime supports them, falling back to unary RPCs otherwise. The end result (complete container/pod list) is identical.

###### Can the feature be disabled once it has been enabled?

Yes. Disabling the feature gate causes kubelet to use only the existing unary RPCs.

###### What happens if we reenable the feature if it was previously rolled back?

Streaming resumes if the runtime supports it. No state is persisted.

###### Are there any tests for feature enablement/disablement?

No dedicated enablement/disablement tests are planned.
The feature is stateless and toggling requires a kubelet restart, so the e2e tests that cover the [behavior matrix](#behavior-matrix)
(including the fallback to unary RPCs) provide sufficient coverage.

To be discussed further before beta graduation.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollout/rollback cannot impact running workloads. Streaming only affects how container/pod lists are retrieved, not container lifecycle.

###### What specific metrics should inform a rollback?

`kubelet_cri_list_streaming_failure_total`, Type: `Counter`, Label: `operation`

This counter increments when a streaming RPC encounters an error during `Recv()`, causing kubelet to discard any items received so far. A sustained increase may indicate runtime instability.

`kubelet_cri_list_streaming_fallback_total`, Type: `Counter`, Label: `operation`

This counter increments when kubelet falls back to the unary RPC because the runtime returned `UNIMPLEMENTED` for the streaming RPC.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

To be tested during Alpha phase.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Not applicable. It's not a workload-level feature.

###### How can someone using this feature know that it is working for their instance?

- `kubelet_cri_list_streaming_fallback_total` is **not** incrementing: the runtime supports streaming and kubelet is using the streaming RPCs.
- `kubelet_cri_list_streaming_failure_total` is **not** incrementing: streams are completing successfully without errors.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

None of the existing SLOs are applicable to this enhancement.
Runtime-side metrics can be used as SLIs for latency.
- `crio_operations_latency_seconds_total` (CRI-O)

To be discussed further before beta graduation.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `kubelet_cri_list_streaming_failure_total`
  - Tracks how often streaming RPCs fail mid-stream.
- [x] Metrics
  - Metric name: `kubelet_cri_list_streaming_fallback_total`
  - Tracks how often kubelet falls back to unary RPCs.
- [x] Runtime Metrics
  - Metric name: `crio_operations_latency_seconds_total` (CRI-O)
  - Tracks CRI RPC latency.

To be discussed further before beta graduation.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No. The metrics described above will be added as part of this enhancement.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. This feature only requires a CRI-compatible runtime (containerd, CRI-O) with streaming RPC support.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new Kubernetes API calls. The streaming RPCs replace the unary CRI calls with a single streaming call per list operation.

###### Will enabling / using this feature result in introducing new API types?

New streaming RPC methods and their request/response messages are added to the CRI proto. No new Kubernetes API types.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Streaming has per-message framing overhead since each item is sent as a separate gRPC message, but this is negligible over Unix sockets.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Not applicable. This feature is kubelet-to-runtime communication only.

###### What are other known failure modes?

| Failure Mode                       | Detection                                             | Mitigation                                                                 |
|------------------------------------|-------------------------------------------------------|----------------------------------------------------------------------------|
| Runtime does not support streaming | gRPC `UNIMPLEMENTED` status code                      | Kubelet caches the result and falls back to unary RPC for subsequent calls |
| Runtime crashes mid-stream         | `kubelet_cri_list_streaming_failure_total` increments | Kubelet discards partial results and retries the entire streaming call     |

###### What steps should be taken if SLOs are not being met?

If `kubelet_cri_list_streaming_fallback_total` is incrementing, the runtime does not implement the streaming RPCs. Upgrade the runtime to a version that supports streaming.

If `kubelet_cri_list_streaming_failure_total` is incrementing, the runtime is failing mid-stream. Operators should investigate the runtime logs for errors and consider upgrading the runtime.

Disabling the feature gate will cause kubelet to fall back to unary RPCs. Note that if the node has enough resources to exceed the 16 MB gRPC message limit, the unary RPCs will also fail.

## Implementation History

- 2026-01-21: KEP created

## Drawbacks

1. **Implementation burden**: All CRI implementations (containerd, CRI-O, etc.) must implement the new streaming RPCs.

2. **Proto surface area**: New RPCs and message types are added to the CRI proto, increasing the API surface.

3. **Stream lifecycle management**: Runtimes must manage stream lifecycle correctly, including proper stream termination and error handling.

## Alternatives

### Increase gRPC message limit

**Pros:**
- Simple, no API changes required
- Already done twice (4MB->8MB->16MB)

**Cons:**
- Only delays the problem

### Improve garbage collection

**Pros:**
- Addresses root cause (too many dead containers)
- No API changes needed

**Cons:**
- May conflict with forensics/debugging needs
- Doesn't help legitimate high-container-count workloads

### Token-based pagination

**Pros:**
- Follows established patterns (AIP-158)
- No new RPCs needed, only new fields on existing messages

**Cons:**
- Requires token management (generation, validation)
- Requires stable ordering semantics for consistency
- Multiple independent round trips instead of a single stream

### Offset-based pagination

**Pros:**
- Simpler token (just an integer offset)
- Easier to implement

**Cons:**
- Vulnerable to inconsistency when items are added/deleted
- Less aligned with AIP-158 recommendations

## References

### Standards and Documentation

- [gRPC Server-side Streaming](https://grpc.io/docs/what-is-grpc/core-concepts/#server-streaming-rpc)
- [kubernetes/cri-api](https://github.com/kubernetes/cri-api)
- [CRI in Kubernetes Blog](https://kubernetes.io/blog/2016/12/container-runtime-interface-cri-in-kubernetes/)
- [CRI API Version Skew Policy](https://www.kubernetes.dev/docs/code/cri-api-version-skew-policy/)

### Related Issues and PRs

- [#63858](https://github.com/kubernetes/kubernetes/issues/63858) - Original gRPC message limit bug (2018)
- [#63977](https://github.com/kubernetes/kubernetes/pull/63977) - Increased CRI limit from 4MB to 8MB
- [#64672](https://github.com/kubernetes/kubernetes/pull/64672) - Increased CRI limit from 8MB to 16MB
- [#90340](https://github.com/kubernetes/kubernetes/issues/90340) - Feature request for pagination
- [#107190](https://github.com/kubernetes/kubernetes/issues/107190) - Discussion on CRI feature implementation
- [#131407](https://github.com/kubernetes/kubernetes/issues/131407) - High CronJob count hitting gRPC limit (2025)
- [#134750](https://github.com/kubernetes/kubernetes/issues/134750) - 16MB limit exceeded with ~18.5k containers (2025)
