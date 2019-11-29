---
title: Image Pull Progress
authors:
  - "@saschagrunert"
owning-sig: sig-node
participating-sigs:
  - sig-cli
  - sig-api-machinery
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2010-10-25
last-updated: 2010-11-29
status: provisional
---

# Image Pull Progress

## Table of Contents

<!-- toc -->

- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal / User Stories](#proposal--user-stories)
  - [CRI API enhancement](#cri-api-enhancement)
  - [Kubernetes API additions](#kubernetes-api-additions)
  - [CLI modifications](#cli-modifications)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Scalability](#scalability)
  - [Rollout, Upgrade, and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Dependencies](#dependencies)
  - [Monitoring requirements](#monitoring-requirements)
  - [Troubleshooting](#troubleshooting)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
  <!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before [Enhancement
Freeze](https://github.com/kubernetes/sig-release/tree/master/releases) of the
targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP
      (this should be a link to the KEP location in kubernetes/enhancements, not the
      initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG
      Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for
      publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to
      mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once
it is marked `implementable` should be approved by each of the KEP approvers. If
any of those approvers is no longer appropriate than changes to that list should
be approved by the remaining approvers and/or the owning SIG (or SIG-arch for
cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every
time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

Target of this enhancement is to expose the progress of a container image pull
to the end user via the API and the CLI. When a Kubernetes workload gets
created, then the user has now the possibility to obtain additional information,
like the overall amount of bytes already downloaded, the needed bytes to be
downloaded and the estimated pull time based on the currently available
bandwidth. These information are directly available via tools like `kubectl`,
because each workload exposes this state directly via an API.

## Motivation

During Kubernetes workload creation, the time needed to pull the container image
from the remote location varies depending on multiple factors. In low-bandwidth
scenarios the image pull step might consume a significant amount time
especially if the container image is large. Kubernetes provides currently no
possibility to expose this pull progress to the user, whereas the only indicator
is the `ContainerCreating` or `PullingImage` state.

### Goals

- Enhance the Container Runtime Interface (CRI) to be able to stream all
  necessary information from the underlying container runtime
- Expose the progress to the Kubernetes API
- Providing a way how to display the progress to the end user via the CLI

### Non-Goals

Everything which is not related to the container image pull procedure

## Proposal / User Stories

The overall implementation can be split up into three user stories.

### CRI API enhancement

The Kubernetes runtime API `ImageService` has to be modified to provide an
additional streaming remote procedure call (RPC). This server-side stream
provides continuously image pull related metadata during the whole image pull
process. The new endpoint would be added in addition to the already available
`PullImage` RPC and re-uses the request data type:

```protobuf
service ImageService {
    …

    // PullImageStream pulls an image and provides continuous metadata during
    // that time
    rpc PullImageProgress(PullImageRequest) returns (stream PullImageProgressResponse) {}
}

enum PullImageState {
    STARTED = 0;
    PULLING = 1;
    FAILED  = 2;
    DONE    = 3;
}

message PullImageProgressResponse {
    // The current state of the image pull proress
    PullImageState state = 1;

    // The overall size of the image in bytes
    uint64 size = 2;

    // The amount of data already retrieved in bytes
    uint64 current_offset = 3;

    // String indicating the reason for the current state, like a failure message
    string reason = 4;
}
```

The `PullImageState` provides richer information to the kubelet in which state
the image pull currently resides as well as a fine granular error messages
directly from the runtimes via the `reason` field.

### Kubernetes API additions

The required API will be deployed via a Custom Resource Definition (CRD), which
can be enabled via a feature gate. The definition applies cluster wide, whereas
a single custom resource will be created on a per-node basis. This means that
one custom resource gets managed per node, which contains the list of images and
their current pull progress/status.

The kubelet updates the Custom Resources during an image pull, which indicate
the pull progress per container image. The amount of how often the kubelet
updates the resources are configurable in a range between 1s and 1min.

To reduce the impact of the added feature, a kubelet configuration option will
be added which is (for now) disabled per default. This option completely avoids
calling the new gRPC API and falls back to the current `PullImage` RPC.

### CLI modifications

The command line interface of Kubernetes (`kubectl`) does right now provide all
necessary features to display the additional pull progress information.

### Implementation Details/Notes/Constraints

The major caveat for this implementation approach is the increased amount of
resource updates from the kubelet during an image pull. This should be limited in a
fashion that the API surface is not strongly impacted.

## Production Readiness Review Questionnaire

### Feature enablement and rollback

- **How can this feature be enabled / disabled in a live cluster?**

  Via the kubelet dynamic configuration feature if enabled, where we will add a
  dedicated option for it. A kubelet restart will be required if the dynamic
  configuration is disabled.

- **Can the feature be disabled once it has been enabled (i.e., can we roll
  back the enablement)?**

  Yes. If the image is currently being pulled and the feature gets disabled, the
  gRPC context gets closed immediately. This will make the image pull fail,
  whereas the kubelet has to re-pull the image afterwards via the unary gRPC
  call.

- **Will enabling / disabling the feature require downtime for the control
  plane?**

  No

- **Will enabling / disabling the feature require downtime or reprovisioning
  of a node?**

  No, if dynamic kubelet configuration is enabled.

- **What happens if a cluster with this feature enabled is rolled back? What
  happens if it is subsequently upgraded again?**

  Nothing, since the configuration option (enabled or disabled) is only a
  toggle and relies on no other state.

- **Are there tests for this?**

  Yes, end to end tests for the kubelet.

### Scalability

- **Will enabling / using the feature result in any new API calls?
  Describe them with their impact keeping in mind the supported limits
  (e.g. 5000 nodes per cluster, 100 pods/s churn) focusing mostly on:**

  - **components listing and/or watching resources they didn't before**
  - **API calls that may be triggered by changes of some Kubernetes
    resources (e.g. update object X based on changes of object Y)**
  - **periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)**

  Yes, we create and update custom resources during the image pull process. The
  frequency can be configured on the kubelet as well. This frequency will limit
  the amount of updates on slow image downloads in the worst case.

  **Example:** We assume an average container image size of 100 MiB and an
  available bandwidth of 100 MBit/s. This means that the overall image
  transfer from the remote registry would take around 8 seconds.

  Now we assume a workload throughput of 100 new Pods/s with uniquely used
  images, so we have a window of round about 800 pods downloading a container
  image at the same time.

  If we configure the feature to an resource update period of 1 second, then it
  would result in between 0 and 800 QPS to the API Server.

- **Will enabling / using the feature result in supporting new API types? How
  many objects of that type will be supported (and how that translates to
  limitations for users)?**

  Yes, the API will be deployed via a CRD.

  (TODO: I'm not sure if there is a need to limit the amount of custom
  resources)

- **Will enabling / using the feature result in increasing size or count of
  the existing API objects?**

  Yes, the amount of resources and their updates will increase up to the
  configured update period multiplied by the amount of pods pulling an image at
  the same time.

- **Will enabling / using the feature result in increasing time taken by any
  operations covered by existing SLIs/SLOs (e.g. by adding additional
  work, introducing new steps in between, etc.)? Please describe the details
  if so.**

  No, it should not influence anything mentioned. The calculation of the
  pull progress is left out for sake of triviality.

- **Will enabling / using the feature result in non-negligible increase of
  resource usage (CPU, RAM, disk IO, ...) in any components? Things to keep in
  mind include: additional in-memory state, additional non-trivial
  computations, excessive access to disks (including increased log volume),
  significant amount of data sent and/or received over network, etc.**

  No

### Rollout, Upgrade, and Rollback Planning

### Dependencies

- **Does this feature depend on any specific services running in the cluster
  (e.g., a metrics service)?**

  Yes the container runtime needs to support the new streaming CRI API method
  `PullImageProgress()`.

- **How does this feature respond to complete failures of the services on
  which it depends?**

  If the feature does not work as expected or does not work at all, the
  overall image pull functionality will not be affected, but the additional
  metadata (pull progress) is not visible to the user.

- **How does this feature respond to degraded performance or high error rates
  from services on which it depends?**

  It will report these errors to the API via the custom resource. This was not
  done before and is also a purpose of this feature.

### Monitoring requirements

- **How can an operator determine if the feature is in use by workloads?**

  N/A (The feature is not used directly by workloads)

- **How can an operator determine if the feature is functioning properly?**

  Image pull progress should be exposed via the custom resource in the
  configured intervals.

- **What are the service level indicators an operator can use to determine the
  health of the service?**

  The kubelet is reporting a prometheus metric for the number of failed calls
  to the new CRI API method or failed resource updates. The SLI will correlate
  to the aggregation of this metric for all nodes.

- **What are reasonable service level objectives for the feature?**

  N/A (The feature is not related to any service level objective)

### Troubleshooting

- **What are the known failure modes?**

  The will not start if the feature is enabled (configured) but not supported
  by the runtime.

- **How can those be detected via metrics or logs?**

  Yes, the kubelet will log the failure and provide a new metric for that.

- **What are the mitigations for each of those failure modes?**

  Updating the container runtime to support the new CRI API.

- **What are the most useful log messages and what logging levels do they
  require?**

  Since the kubelet should reject startup if the runtime does not support the
  feature, it will log on error level.

- **What steps should be taken if SLOs are not being met to determine the
  problem?**

  N/A (There is no SLO anticipated)

## Design Details

### Test Plan

<!-- TBD -->

### Graduation Criteria

<!-- TBD -->

### Upgrade / Downgrade Strategy

<!-- TBD -->

### Version Skew Strategy

<!-- TBD -->

## Implementation History

<!-- TBD -->
