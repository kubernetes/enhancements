# Non graceful node shutdown

This includes the Summary and Motivation sections.

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Handle the Return of Shutdown Node](#handle-the-return-of-shutdown-node)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [rely on the taint as a lock mechanism (taint and untaint by the same actor)](#rely-on-the-taint-as-a-lock-mechanism-taint-and-untaint-by-the-same-actor)
<!-- /toc -->

## Release Signoff Checklist

- [] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those
approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

In case of a node being shutdown (i.e. not running) the control plane doesn't make the right
assumptions to enable stateful workload to fail-over. This KEP introduces a flow to do so.

This KEP is part of collection of KEPs that intends to enhance stateful workload management on top of Kubernetes.

## Motivation

Today kubernetes doesn't enable storage providers that are able to safely detach volumes (i.e. only detach when volumes aren't being written to) to do so. This proposal aims to introduce a new CSI capability that if advertised would trigger volume detach if a node is partitioned from the control plane.

### Goals

* Increase the availability of stateful workloads.
* Automate self-healing for stateful workloads.
* This proposal only target CSI volumes.

### Non-Goals

* Implement in-cluster logic to handle node/control plane partitioning. 
* Enable detach for all the storage providers.
* Existing in-tree volumes are not targeted by this proposal.

## Proposal

User stories:

* As a cluster administrator I want my stateful workload to failover in case a node is partitioned from the control plane without any intervention when it's safe to do so.

* As a developer I would like to rely on a self-healing platform for my stateful workloads.

This KEP plans to introduce the following CSI capabilities as described in this [PR](https://github.com/container-storage-interface/spec/pull/477).

* In the [CSI spec](https://github.com/container-storage-interface/spec/blob/master/spec.md), introduce `UNPUBLISH_FENCE` controller service capability and `FORCE_UNPUBLISH` node service capability.
  * The `UNPUBLISH_FENCE` controller service capability indicates that the SP supports ControllerUnpublishVolume.fence field.
  * The `FORCE_UNPUBLISH` node service capability indicates that the SP supports the NodeUnpublishVolume.force field. It also indicates that the SP supports the NodeUnstageVolume.force field if it also has the STAGE_UNSTAGE_VOLUME node service capability.

* In the [CSIDriverSpec](https://github.com/kubernetes/kubernetes/blob/v1.20.2/pkg/apis/storage/types.go#L266), add a `SafeDetach` boolean to indicate if it is safe to detach.

```go
// CSIDriverSpec is the specification of a CSIDriver.
type CSIDriverSpec struct {
  ...

  // SafeDetach indicates this CSI volume driver is able to perform detach
  // operations. When set to true, the CSI volume driver needs to ensure that 
  // ControllerUnpublishVolume() implementation is checking whether a volume
  // can be detached safely
  // +optional
  SafeDetach *bool
}

```

Existing logic:
* When a node is partitioned from the control plane, the health check in Node lifecycle controller, part of kube-controller-manager, sets Node v1.NodeReady Condition to False or Unknown (unreachable) if lease is not renewed for a specific grace period. Node Status becomes NotReady.

* After 300 seconds (default), the Taint Manager tries to delete Pods on the Node after detecting that the Node is NotReady. The Pods will be stuck in terminating status.

Proposed logic change:
* The Pod GC Controller, part of the kube-controller-manager, would need to go through all the Pods in terminating state, verify that the Node is NotReady, and then check if the `SafeDetach` flag in CSIDriver is set to true. If so, forcefully delete pods that are stuck.

* Specifically the Pod GC Controller (the only controller able to forcefully delete pods) would also check now for stateful pods, and select those that satisfy ALL of the following conditions:

  * pods that are backed by a CSI driver that supports `SafeDetach`.
  * pods that are backed by volumes that aren't marked as `ReadWriteMany`. This would avoid breaking having multiple replicas of the same pod.

* Once pods are selected and forcefully deleted, the attachdetach reconciler should check if the `SafeDetach` flag in CSIDriver is set to true. If so, it should skip the `attachedVolume.MountedByNode` check for the volumes backing these pods and allow `volumeAttachement` to be deleted.

* This would trigger the deletion of the `volumeAttachement` objects. This would allow `ControllerUnpublishVolume` to happen before `NodeUnpublishVolume` and/or `NodeUnstageVolume` are called.

* When the `external-attacher` detects the `volumeAttachement` object is being deleted, it needs to check if the `UNPUBLISH_FENCE` CSI controller capability is true. If so, it calls CSI driver's `ControllerUnpublishVolume` and sets `ControllerUnpublishVolume.fence` field to true. The CSI driver would need to ensure that no volume is being used through a mount point. If unable to determine the usage, the CSI driver can return an error without detaching the volume. The `external-attacher` would then retry with backoffs.

* The Pod GC Controller would also apply a `quarantine` taint on the nodes whose pods are forcefully deleted due to the shutdown. This should happen before the pods are being evicted. The "quarantine" taint indicates that new pods should not be scheduled on the node.

### Handle the Return of Shutdown Node

If the nodes that were previously shutdown with pods forcefully deleted come back, we need to make sure they are cleaned up before pods are scheduled on them again.

The health check in Node lifecycle controller, part of kube-controller-manager, sets Node v1.NodeReady Condition to True if lease is renewed again. Node Status becomes Ready.

When the node comes up, Kubelet volume manager reconciler will try to clean up mounts by calling `NodeUnpublishVolume`. Kubelet volume manager reconciler will also try to reconstruct global mount for CSI volumes and call `NodeUnstageVolume`. If there isn't enough information to reconstruct global mount, Kubelet volume manager should cleanup the [global mount directory](/var/lib/kubelet/plugins/kubernetes.io/csi/pv/{pvname}/globalmount) to avoid problems when new pods are scheduled to this node again.

The Pod GC Controller should also remove the `quarantine` taint from the nodes that are already cleaned up and are safe to run pods again. We need to make sure the taint is only removed after detach is finished.

Note: We'll handle the temporarily partitioned node the same way as the shutdown node. Since the `quarantine` taint is applied on the node, we'll clean up everything on the node when its network connectivity is resumed before removing the taint and allowing pods to be scheduled on the node again.

### Risks and Mitigations

This KEP introduces changes, and if not tested correctly could result in data corruption for users.
To mitigate this we plan to have a high test coverage and to introduce this enhancement as an alpha

## Design Details

### Test Plan

This feature will be tested with the following approaches:

- unit tests
- integration
- e2e

we also plan to test this with different version Skews.

### Graduation Criteria

This KEP will be treated as a new feature, and will be introduced with a new feature gate, `NonGracefulFailover`.

This enhancement will go through the following maturity levels: alpha, beta and stable.

Graduation criteria between these levels to be determined.

### Upgrade / Downgrade Strategy

once enabled by default, the upgrades should be smooth as the default behaviour is off by default. when downgrading/rollbacking with this behaviour enabled, there should be no issues as the control plane with the logic is still there.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.

-->

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: NonGracefulFailover
    - Components depending on the feature gate: kube-controller-manager, kubelet
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**
  <!--
  Any change of default behavior may be surprising to users or break existing
  automations, so be extremely careful here.
  -->
  Yes. If this feature is enabled, the pod gc controller in kube-controller-manager
  would need to ensure that `SafeDetach` option in `CSIDriver` is true before
  forcefully deleting pods.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Also set `disable-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g., if this is a runtime
  feature, can it break the existing applications?).
  Yes. If this feature is disable once it has been enabled, it falls back to
  default behavior before the feature is enabled, meaning the pod gc controller
  will not check the `SafeDetach` option in `CSIDriver` and make decision
  depending on that.

* **What happens if we reenable the feature if it was previously rolled back?**
  If we reenable the feature if it was previously rolled back, the pod gc controller
  will again check the `SafeDetach` option in `CSIDriver` and make sure it is
  true before forcefully deleting pods.

* **Are there any tests for feature enablement/disablement?**
  <!--
  The e2e framework does not currently support enabling or disabling feature
  gates. However, unit tests in each component dealing with managing data, created
  with and without the feature, are necessary. At the very least, think about
  conversion tests if API types are being modified.
  -->
  We will add unit test for feature enablement/disablement.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At a high level, this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide comprehensive guidance, but at the very
  high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  Think about both cluster-level services (e.g. metrics-server) as well
  as node-level agents (e.g. specific version of CRI). Focus on external or
  optional services that are needed. For example, if this feature depends on
  a cloud provider API, or upon an external software-defined storage or network
  control plane.

  For each of these, fill in the following—thinking about running existing user workloads
  and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:

### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
  focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)

* **Will enabling / using this feature result in introducing new API types?**
  Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)

* **Will enabling / using this feature result in any new calls to the cloud
provider?**

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**
  Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data sent and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits].

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**
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

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- 2019-06-26: Initial KEP published.
- 2020-11-10: KEP updated to handle part of the node partitioning

## Alternatives

In order to achieve failover properties, the control plane needs to coordinate with Kubelet.

### rely on the taint as a lock mechanism (taint and untaint by the same actor)

1. gc_controller taints the node objects when it observe a node is shutdown
2. gc_controller starts removing forcefully the pods
3. gc_controller untaint the node when the eviction is done

the kubelet will:

1. start and fetch the node object from the apiserver
  1.a. fails: doesn't start containers unless it tolerates the taint
  1.b. success
2. check for the taint
  2.a. exists: same as 1.a
  2.b. doesn't exist: start normally

This solution has many scenarios to consider and can be error-prone, this is why we added the lease as Locking
mechanism to ensure we're not missing something.
