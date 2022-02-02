# Non graceful node shutdown

This includes the Summary and Motivation sections.

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Use Cases](#use-cases)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Handle the Return of Shutdown Node](#handle-the-return-of-shutdown-node)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Unit tests](#unit-tests)
  - [E2E tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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
  - [Node fencing](#node-fencing)
  - [SafeDetach Option](#safedetach-option)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.
- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation - e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those
approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

In this KEP, we are proposing a way that allows stateful workloads to failover to a different node successfully after the original node is shutdown or in a non-recoverable state such as the hardware failure or broken OS.

## Motivation

The Graceful Node Shutdown [KEP](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2000-graceful-node-shutdown) introduced a way to detect a node shutdown and handle it gracefully. However, a node shutdown action may not be detected by Kubelet's Node Shutdown Mananger, either because the command does not trigger the inhibitor locks mechanism used by Kubelet or because of a user error, i.e., the ShutdownGracePeriod and ShutdownGracePeriodCriticalPods are not configured properly.

When a node is shutdown but not detected by Kubelet's Node Shutdown Manager, the pods that are part of a StatefulSet will be stuck in terminating status on the shutdown node and cannot move to a new running node. This is because Kubelet on the shutdown node is not available to delete the pods so the StatefulSet cannot create a new pod with the same name. If there are volumes used by the pods, the VolumeAttachments will not be deleted from the original shutdown node so the volumes used by these pods cannot be attached to a new running node. As a result, the application running on the StatefulSet cannot function properly. If the original shutdown node comes up, the pods will be deleted by Kubelet and new pods will be created on a different running node. If the original shutdown node does not come up, these pods will be stuck in terminating status on the shutdown node forever.

In this KEP, we are proposing a way to handle node shutdown cases that are not detected by Kubelet's Node Shutdown Manager. The pods will be forcefully deleted in this case, trigger the deletion of the VolumeAttachments, and new pods will be created on a new running node so that application can continue to function.

Similarly, this approach can also be applied to the case when a node is in a non-recoverable state such as the hardware failure or broken OS.

### Use Cases

* If user wants to intentionally shutdown a node, he/she can validate whether graceful node shutdown feature works. If Kubelet is able to detect node is shutting down, it will gracefully delete pods, and new pods will be created on another running node.
* If graceful shutdown is not working or node is in non-recoverable state due to hardware failure or broken OS, etc., user now can enable this feature, and add `out-of-service=nodeshutdown:NoExecute` taint which will be explained in detail below to trigger non-graceful shutdown behavior.

### Goals

* Increase the availability of stateful workloads when a node is shutdown, or in other non-recoverable cases such as the hardware failure or broken OS.
* Requires user intervention to detect node shutdown/failure cases, because currently there is no good way of detecting whether node is in the middle of restarting.
  * Once the feature in this KEP is developed and tested, we plan to work on automatic way of detecting and fencing node for shutdown/failure use cases and re-evaluating some approaches listed in the Alternatives section.

### Non-Goals

* Node/control plane partitioning other than a node shutdown is not covered by the proposal, but will be addressed in the future and built on top of this design.
  * We do not have a way to determine whether it is safe to detach in the case of node partitioning. In the Alternatives section, we discussed the node fencing approach and explained why that is not selected for this KEP but will be considered after this KEP is implemented.
* Implement in-cluster logic to handle node/control plane partitioning.

## Proposal

In this section, we are describing the proposed changes to enable workloads to failover to a running node when the original node is shutdown but not detected by Kubelet's Node Shutdown Manager, or when the node is in other non-recoverable cases such as the hardware failure or broken OS.

Note: The proposal in the current KEP involves user manually add/remove a taint. In the future, we will look into a more automatic approach that does not require these manual steps.

When users verify that a node is already in shutdown or power off state (not in the middle of restarting), either user intentionally shut it down or node is down due to hardware failures, OS issues, they can taint the node with `out-of-service` with `NoExecute` effect.

Existing logic:
1. When a node is not reachable from the control plane, the health check in Node lifecycle controller, part of kube-controller-manager, sets Node v1.NodeReady Condition to False or Unknown (unreachable) if lease is not renewed for a specific grace period. Node Status becomes NotReady.

1. After 300 seconds (default), the Taint Manager tries to delete Pods on the Node after detecting that the Node is NotReady. The Pods will be stuck in terminating status.

Proposed logic change:
1. [Proposed change] This proposal requires a user to apply a `out-of-service` taint on a node when the user has confirmed that this node is shutdown or in a non-recoverable state due to the hardware failure or broken OS. Note that user should only add this taint if the node is not coming back at least for some time. If the node is in the middle of restarting, this taint should not be used.

1. [Proposed change] In the Pod GC Controller, part of the kube-controller-manager, add a new function called gcTerminating. This function would need to go through all the Pods in terminating state, verify that the node the pod scheduled on is NotReady. If so, do the following:
  1.  Upon seeing the `out-of-service` taint, the Pod GC Controller will forcefully delete the pods on the node if there are no matching tolation on the pods. This new `out-of-service` taint has `NoExecute` effect, meaning the pod will be evicted and a new pod will not schedule on the shutdown node unless it has a matching toleration. For example, `out-of-service=nodeshutdown:NoExecute` or `out-of-service=hardwarefailure:NoExecute`. We suggest to use `NoExecute` effect in taint to make sure pods will be evicted (deleted) and fail over to other nodes.
  1. We'll follow taint and toleration policy. If a pod is set to tolerate all taints and effects, that means user does NOT want to evict pods when node is not ready. So GC controller will filter out those pods and only forcefully delete pods that do not have a matching toleration. If your pod tolerates the `out-of-service` taint, then it will not be terminated by the taint logic, therefore none of this applies.

1. [Proposed change] Once pods are selected and forcefully deleted, the attachdetach reconciler should check the `out-of-service` taint on the node. If the taint is present, the attachdetach reconciler will not wait for 6 minutes to do force detach. Instead it will force detach right away and allow `volumeAttachment` to be deleted.

1. This would trigger the deletion of the `volumeAttachment` objects. For CSI drivers, this would allow `ControllerUnpublishVolume` to happen without `NodeUnpublishVolume` and/or `NodeUnstageVolume` being called first. Note that there is no additional code changes required for this step. This happens automatically after the `Proposed change` in the previous step to force detach right away.

1. When the `external-attacher` detects the `volumeAttachment` object is being deleted, it calls CSI driver's `ControllerUnpublishVolume`.

Note: [Existing] If the `out-of-service:NoExecute` taint is applied to a node that is healthy and running, the Taint Manager will try to delete the pods without the matching toleration. Pods will be deleted by Kubelet because the node is healthy and running. New pods will be created on a different running node and application will continue to run.

### Handle the Return of Shutdown Node

If the nodes that were previously shutdown with pods forcefully deleted come back, we need to make sure they are cleaned up before pods are scheduled on them again.

[Proposed Change] We require the user to manually remove the `out-of-service` taint after the pods are moved to a new node and the user has checked that the shutdown node has been recovered since the user was the one who originally added the taint. This makes sure no new pods are scheduled on the node before the `out-of-service` taint is removed. In the future, we can enhance the Pod GC Controller to automatically remove the `out-of-service` taint, after verifying that no pods are attached to the node, and no volume attachments on the node.

Note that there is no additional code change required for the return of shutdown node.

[Existing Logic] The health check in Node lifecycle controller, part of kube-controller-manager, sets Node v1.NodeReady Condition to True if lease is renewed again. Node Status becomes Ready.

[Existing Logic] Note that if the `out-of-service` taint was not added to the shutdown node, the pods will be stuck in terminating status forever if the shutdown node remains down. If the shutdown node comes up, however, the original `volumeAttachment` will be deleted and the pods will be moved to a new node successfully.

### Risks and Mitigations

This KEP introduces changes, and if not tested correctly could result in data corruption for users.
To mitigate this we plan to have a high test coverage and to introduce this enhancement as an alpha

## Design Details

### Test Plan

### Unit tests
* Add unit tests to affected components in kube-controller-manager:
  * Add tests in Pod GC Controller for the new logic to clean up pods and the `out-of-service` taint.
  * Add tests in Attachdetach Controller for the changed logic that allow volumes to be forcefully detached without wait.

### E2E tests
*   New E2E tests to validate workloads move successfully to another running node when a node is shutdown.
    * Feature gate for `NonGracefulFailover` is disabled, feature is not active.
    * Feature gate for `NonGracefulFailover` is enabled. Add `out-of-service` taint after node is shutdown:
      * Verify workloads are moved to another node successfully.
      * Verify the `out-of-service` taint is removed after the shutdown node is cleaned up.
* Add stress and scale tests before moving from beta to GA.

We also plan to test this with different version Skews.

### Graduation Criteria

This KEP will be treated as a new feature, and will be introduced with a new feature gate, `NonGracefulFailover`.

This enhancement will go through the following maturity levels: alpha, beta and stable.

#### Alpha

* Initial feature implementation in kube-controller-manager.
* Add basic unit tests.

#### Alpha -> Beta Graduation

* Gather feedback from developers and surveys.
* Unit tests and e2e tests outlined in design proposal implemented.
* Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

* Feature is deployed in production and have gone through at least one Kubernetes upgrade.
* More rigorous forms of testing—e.g., downgrade tests and scalability tests.
* Allowing time for feedback.

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
    - Components depending on the feature gate: kube-controller-manager
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
      Yes.
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
      No.

* **Does enabling the feature change any default behavior?**
  <!--
  Any change of default behavior may be surprising to users or break existing
  automations, so be extremely careful here.
  -->
  Yes. If this feature is enabled, the pod gc controller in kube-controller-manager
  would need to ensure that `out-of-service` taint is applied on a node before
  forcefully deleting pods.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Also set `disable-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g., if this is a runtime
  feature, can it break the existing applications?).
  Yes. If this feature is disable once it has been enabled, it falls back to
  default behavior before the feature is enabled, meaning the pod gc controller
  will not check the `out-of-service` taint and make decision depending on that.

* **What happens if we reenable the feature if it was previously rolled back?**
  If we reenable the feature if it was previously rolled back, the pod gc controller
  will again check the `out-of-service` taint before forcefully deleting pods.

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
- 2021-08-26: The scope of the KEP is narrowed down to handle a real node shutdown. Test plan is updated. Node partitioning will be handled in the future and it can be built on top of this design.
- 2021-12-03: Removed `SafeDetach` flag. Requires a user to add the `out-of-service` taint when he/she knows the node is shutdown.

## Alternatives

### Node fencing

[Node fencing](https://github.com/rootfs/node-fencing) was brought up as an alternative during KEP reviews. This approach depends on the user to enter what commands are required to shut down a node as the commands differ in different environment.

```
- kind: ConfigMap
  apiVersion: v1
  metadata:
   name: fence-method-fence-rhevm-node1
   namespace: default
  data:
   method.properties: |
          agent_name=fence-rhevm
          namespace=default
          ip=ovirt.com  # address to the rhevm management
          username=admin@internal
          password-script=/usr/sbin/fetch_passwd
          ssl-insecure=true
          plug=vm-node1  # the vm name
          action=reboot
          ssl=true
          disable-http-filter=true
```

This approach seems too intrusive as a general in-tree solution. Since the scope of this KEP is narrowed down to handle a real node shutdown scenario, we do not need the node fencing approach. I think what has been proposed in this KEP also does not prevent us from supporting the node partitioning case in the future. We can reconsider this approach after the proposed solution is implemented.

### SafeDetach Option

This proposal introduces a new option in CSIDriver. If set to true, it would trigger volume detach if a node is shutdown.

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

Since we have modified the KEP and require the node to be shutdown, we no longer need this `SafeDetach` flag. Currently no CSI driver can truly detect whether it is safe to detach itself. We can re-visit this option later when handling network partitioning case.
