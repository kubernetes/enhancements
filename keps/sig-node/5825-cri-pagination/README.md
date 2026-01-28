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
# KEP-5825: CRI Pagination

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
    - [ListContainers](#listcontainers)
    - [ListPodSandbox](#listpodsandbox)
    - [Other List operations](#other-list-operations)
  - [Token Implementation](#token-implementation)
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
  - [Server-side streaming](#server-side-streaming)
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

This KEP proposes adding token-based pagination to CRI's `List*` RPCs. Currently, these APIs return all results in a single response, causing failures on nodes with 10k+ containers when responses exceed the 16 MB gRPC message limit. The proposed solution follows AIP-158 pagination patterns without default page size.

## Motivation

CRI's `List*` RPCs currently return all results without pagination. On nodes with many containers, this causes Responses exceeding the gRPC limit (currently 16 MB) cause complete RPC failures

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

- Add pagination support to `List*` RPCs
- Maintain backward compatibility with existing kubelet/runtime combinations
- Follow established pagination patterns (AIP-158) as much as possible

### Non-Goals

- Changing the default gRPC message size limit
- Modifying container/pod garbage collection behavior
- Implementing server-side filtering beyond existing `ContainerFilter`/`PodSandboxFilter`

## Proposal

Add optional pagination fields to `List*` request/response messages.

### User Stories

**Story 1: High CronJob Churn**

As a cluster operator running hundreds of CronJobs, I want kubelet to successfully list containers even when thousands of completed job containers exist, so that new pods can be scheduled without hitting gRPC limits.

**Story 2: CI/CD Node**

As a platform engineer managing CI nodes that run many short-lived containers, I want to avoid kubelet failures when container counts spike, so that CI pipelines remain reliable.

### Notes/Constraints/Caveats

1. **Pagination is not needed for network efficiency**: Kubelet and CRI runtimes communicate over Unix sockets. Pagination primarily benefits when the message size exceeds the gRPC limit.

### Risks and Mitigations

1. **Consistency during modifications**: Containers are created/deleted frequently. During pagination:
  - New containers may be missed
  - Deleted containers may still appear
  - Duplicates possible if ordering shifts

    Kubelet reconciliation handles eventual consistency.

2. **Contrary to kubelet design**: Kubelet is designed to see an atomic view of all the pods.

   We can set it as high as possible (100k containers) and create a wrapper function to call the CRI API until all pages are loaded.
   It won't require multiple calls until it reaches close to the limit, and even if it does, it looks like a single call from the caller.

## Design Details

### API Changes

#### ListContainers

**ListContainersRequest:**
```protobuf
message ListContainersRequest {
    ContainerFilter filter = 1;
    int32 page_size = 2;      // 0 = no limit (backward compat)
    string page_token = 3;    // empty = first page
}
```

**ListContainersResponse:**
```protobuf
message ListContainersResponse {
    repeated Container containers = 1;
    string next_page_token = 2;  // empty = no more pages
}
```

#### ListPodSandbox

**ListPodSandboxRequest:**
```protobuf
message ListPodSandboxRequest {
    PodSandboxFilter filter = 1;
    int32 page_size = 2;      // 0 = no limit (backward compat)
    string page_token = 3;    // empty = first page
}
```

**ListPodSandboxResponse:**
```protobuf
message ListPodSandboxResponse {
    repeated PodSandbox items = 1;
    string next_page_token = 2;  // empty = no more pages
}
```

**Key design decision**: `page_size = 0` means "return all results" (deviates from AIP-158's "use default"). This is critical for backward compatibility—old runtimes ignore the field (defaulting to 0), and new runtimes treat 0 as "no pagination."

Also in gRPC with Protocol Buffers (proto3), unknown fields are silently ignored by the receiver. Even if the runtime doesn't support the feature, it can work as if the feature is off.

#### Other List operations

Same changes for:
- ListContainerStats
- ListPodSandboxStats
- ListMetricDescriptors
- ListPodSandboxMetrics
- ListImages (ImageService)

**Behavior Matrix:**

| Kubelet | Runtime | page_size sent | Result |
|---------|---------|----------------|--------|
| Old | Old | (none) | Full list |
| Old | New | (none) | Full list |
| New | Old | >0 | Full list |
| New | New | >0 | Paginated |

### Token Implementation

Tokens must be opaque, URL-safe, and tamper-resistant. It can vary among runtimes. This is an example using HMAC-signed tokens:

```go
type PageToken struct {
    CreatedAt int64  `json:"c"`  // last seen creation timestamp
    ID        string `json:"i"`  // last seen container/sandbox ID
    Version   int    `json:"v"`  // token version for future changes
}

func (t *PageToken) Encode(secret []byte) string {
    payload, _ := json.Marshal(t)
    mac := hmac.New(sha256.New, secret)
    mac.Write(payload)
    sig := mac.Sum(nil)[:16]
    return base64.RawURLEncoding.EncodeToString(append(sig, payload...))
}

func DecodePageToken(token string, secret []byte) (*PageToken, error) {
    data, err := base64.RawURLEncoding.DecodeString(token)
    if err != nil || len(data) < 16 {
        return nil, errors.New("invalid token")
    }
    sig, payload := data[:16], data[16:]
    mac := hmac.New(sha256.New, secret)
    mac.Write(payload)
    if !hmac.Equal(sig, mac.Sum(nil)[:16]) {
        return nil, errors.New("invalid token signature")
    }
    var t PageToken
    if err := json.Unmarshal(payload, &t); err != nil {
        return nil, err
    }
    return &t, nil
}
```

### Test Plan

- [x] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

#### Prerequisite testing updates

#### Unit tests

- Pagination logic with various page sizes

#### Integration tests

- Pagination with containerd and CRI-O
- Mixed version kubelet/runtime scenarios
- Large container count scenarios (10k+ containers)

#### e2e tests

- End-to-end pagination on nodes with high container counts
- Verify sync loop behavior with paginated responses

### Graduation Criteria

#### Alpha

- Feature gate: `CRIListPagination`
- Pagination fields added to CRI proto (optional)
- Kubelet implements pagination
- Reference implementation in containerd or CRI-O
- Basic unit and integration tests

#### Beta

- Feature gate enabled by default
- Both containerd and CRI-O implement pagination
- E2E tests passing
- Metrics for pagination usage added
- Documentation updated

#### GA

- Feature gate locked to enabled

### Upgrade / Downgrade Strategy

**Upgrade path:**
1. Upgrade runtime and kubelet (order-agnostic)

**Downgrade path:**
1. Downgrade runtime and kubelet (order-agnostic)

We can do them order-agnostic, because if either of them is not supported, it safely falls back to the full-list behavior.

### Version Skew Strategy

The feature discovery mechanism handles all version skew scenarios:

- **New kubelet + Old runtime**: Kubelet detects no pagination support, uses `page_size=0`
- **Old kubelet + New runtime**: Runtime receives no pagination params, returns full list
- **New kubelet + New runtime**: Full pagination enabled

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `CRIListPagination`
  - Components depending on the feature gate: kubelet

###### Does enabling the feature change any default behavior?

No. When enabled, kubelet will use pagination only if the runtime supports. The end result (complete container/pod list) is identical.

###### Can the feature be disabled once it has been enabled?

Yes. Disabling the feature gate causes kubelet to use `page_size=0`, returning to unpaginated behavior.

###### What happens if we reenable the feature if it was previously rolled back?

Pagination resumes if the runtime supports. No state is persisted.

###### Are there any tests for feature enablement/disablement?

Yes, integration tests cover enable/disable scenarios.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollout/rollback cannot impact running workloads. Pagination only affects how container/pod lists are retrieved, not container lifecycle.

###### What specific metrics should inform a rollback?

To be considered for beta.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

To be tested during Alpha phase.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

To be considered for beta.

###### How can someone using this feature know that it is working for their instance?

To be considered for beta.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

To be considered for beta.

###### What are the SLIs (Service Level Indicators) an operator can use?

To be considered for beta.

###### Are there any missing metrics that would be useful?

To be considered for beta.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. Only requires a CRI-compatible runtime (containerd, CRI-O) with pagination support.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new API types. Additional CRI calls only when pagination is used (one per page vs. one total).

###### Will enabling / using this feature result in introducing new API types?

No new types; only new optional fields on existing messages.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Potentially slight increase in total list time due to multiple round trips when there's multiple pages, but each round trip is cheap (Unix socket IPC).

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Not applicable. This feature is kubelet-to-runtime communication only.

###### What are other known failure modes?

| Failure Mode | Detection | Mitigation |
|--------------|-----------|------------|
| Token corruption/tampering | HMAC validation fails | Return error, kubelet retries from beginning |
| Runtime crashes mid-pagination | RPC error | Kubelet retries complete list operation |
| Inconsistent results | Duplicates/gaps in reconciliation | Kubelet reconciliation handles eventual consistency |

###### What steps should be taken if SLOs are not being met?

Collect kubelet logs and metrics

## Implementation History

- 2026-01-21: KEP created

## Drawbacks

1. **Implementation burden**: All CRI implementations (containerd, CRI-O, etc.) must implement pagination consistently.

2. **Additional requests**: It may require a few additional requests to retrieve the complete list when pagination is enabled.

3. **Ordering requirements**: Pagination requires stable ordering. Container IDs are random hashes; ordering by creation time may add overhead.

## Alternatives

### Increase gRPC message limit

**Pros:**
- Simple, no API changes required
- Already done twice (4MB→8MB→16MB)

**Cons:**
- Only delays the problem

### Improve garbage collection

**Pros:**
- Addresses root cause (too many dead containers)
- No API changes needed

**Cons:**
- May conflict with forensics/debugging needs
- Doesn't help legitimate high-container-count workloads

### Server-side streaming

**Pros:**
- True incremental processing

**Cons:**
- Larger API change
- More complex error handling
- Less alignment with existing Kubernetes patterns

### Offset-based pagination

**Pros:**
- Simpler token (just an integer offset)
- Easier to implement

**Cons:**
- Vulnerable to inconsistency when items are added/deleted
- Less aligned with AIP-158 recommendations

## References

### Standards and Documentation

- [Google AIP-158 - Pagination](https://google.aip.dev/158)
- [Google AIP-4233 - Client Library Pagination](https://google.aip.dev/client-libraries/4233)
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
