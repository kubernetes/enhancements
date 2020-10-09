# Allow DaemonSets to surge during update like Deployments

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [Workload Implications](#workload-implications)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Implications to drain](#implications-to-drain)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

Daemonsets allow two update strategies - OnDelete which only replaces pods when they are deleted and RollingUpdate which supports MinAvailable like Deployments but not Surge. Daemonsets should support Surge in order to minimize DaemonSet downtime on nodes. This will allow daemonset workloads to implement zero-downtime upgrades.

## Motivation

DaemonSets are a key enabler of Kubernetes system-level integrations like CNI, CSI, or per-node functionality. These integrations may have availability impacts on workloads during daemonset updates for a number of reasons, including image pull time or setup. While increasing availability of these daemonsets often requires development investment to manage the handoff between the old instance and the new instance, without the ability to have two pods on the same node these handoffs are complex to implement and typically require higher level orchestration (such as running two daemonsets and round robining updates, or using the OnDelete strategy and orchestrating pod deletes when nodes will be rebooted).

It should be possible for a node level integration to offer zero-downtime upgrades via a DaemonSet without resorting to a higher level orchestration.

### Goals

- Add support for Surge to the DaemonSet rolling update strategy

## Proposal

### Implementation Details/Notes/Constraints

The design of Deployment rolling updates introduced the surge concept, and the initial design for DaemonSet updates considered the implications of adding the Surge strategy later (https://github.com/kubernetes/community/blob/master/contributors/design-proposals/apps/daemonset-update.md#future-plans). [StatefulSets may also surge in a workload specific fashion](https://github.com/kubernetes/enhancements/pull/1863), so this design should be as consistent as possible with existing concepts but clearly denote where the workload concept differs from other controllers.

We would add `MaxSurge *intstr.IntOrString` to the RollingUpdate daemonset upgrade strategy. It would have a default value of 0, preserving current behavior. We would allow MaxUnavailable to be 0 when MaxSurge is set.

```
// Spec to control the desired behavior of daemon set rolling update.
type RollingUpdateDaemonSet struct {
	// The maximum number of DaemonSet pods that can be unavailable during the
	// update. Value can be an absolute number (ex: 5) or a percentage of total
	// number of DaemonSet pods at the start of the update (ex: 10%). Absolute
	// number is calculated from percentage by rounding down to a minimum of one.
	// This cannot be 0 if MaxSurge is 0
	// Default value is 1.
	// Example: when this is set to 30%, at most 30% of the total number of nodes
	// that should be running the daemon pod (i.e. status.desiredNumberScheduled)
	// can have their pods stopped for an update at any given time. The update
	// starts by stopping at most 30% of those DaemonSet pods and then brings
	// up new DaemonSet pods in their place. Once the new pods are available,
	// it then proceeds onto other DaemonSet pods, thus ensuring that at least
	// 70% of original number of DaemonSet pods are available at all times during
	// the update.
	// +optional
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty" protobuf:"bytes,1,opt,name=maxUnavailable"`

	// The maximum number of nodes with an existing available DaemonSet pod that
	// can have an updated DaemonSet pod during during an update.
	// Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%).
	// This can not be 0 if MaxUnavailable is 0.
	// Absolute number is calculated from percentage by rounding up to a minimum of 1.
	// Defaults to 25%.
	// Example: when this is set to 30%, at most 30% of the total number of nodes
	// that should be running the daemon pod (i.e. status.desiredNumberScheduled)
	// can have their a new pod created before the old pod is marked as deleted.
  // The update starts by launching new pods on 30% of nodes. Once an updated
  // pod is available (Ready for at least minReadySeconds) the old DaemonSet pod
  // on that node is marked deleted. If the old pod becomes unavailable for any
  // reason (Ready transitions to false, is evicted, or is drained) an updated
  // pod is immediatedly created on that node without considering surge limits.
	// Allowing surge implies the possibility that the resources consumed by the
	// daemonset on any given node can double if the readiness check fails, and
	// so resource intensive daemonsets should take into account that they may
	// cause evictions during disruption.
	// +optional
	MaxSurge *intstr.IntOrString `json:"maxSurge,omitempty" protobuf:"bytes,2,opt,name=maxSurge"`
```

Unlike Deployments, MaxSurge only considers nodes that have an available old pod and will instantly launch updated pods if no old available pod is detected on a node. An available pod is defined the same way as Deployments - the pod is not terminating, pod is Ready, and pod has been Ready for MinReadySeconds.

In the event a rollout cannot proceed due to hitting the MaxSurge limit (due to any condition, whether scheduling, new pods not going ready) the controller should pause creating new pods until conditions change.

DaemonSet pods are slightly more constrained than Deployments when it comes to scheduling issues since each pod is tied to a single node, so it is worth describing exactly how surge pods that violate same node constraints would be handled consistent with Deployments. The most common conflict is use of HostPort within the pod spec across two versions, which would prevent the second pod from landing and the rollout from proceeding. An identical failure would occur with a Deployment of scale 4 on a 3 node cluster - the rollout would be prohibited because the fourth pod could not be scheduled, and so should be handled identically by this controller. It is user error to specify impossible scheduling constraints, and the correct way to convey that is via status conditions on the DaemonSet (which is a separate proposal).

In order to reduce confusion for new users, we will start by rejecting HostPort use in daemonset when MaxSurge is non-zero. A user will not be able to update a daemonset to MaxSurge != 0 if HostPort is set, or update a HostPort if MaxSurge is set, without receiving a validation error. If the MaxSurge feature gate is off, the validation rule is bypassed, and a user who turns off the gate, sets both fields, and then enables the gate will have failing pods but will be able to update their daemonset to either remove surge or remove the host port safely.

### Workload Implications

There are three main workload types that seek to minimize disruption:

1. Infrastructure that should be quickly replaced during update (CNI plugins, CSI plugins).
2. Infrastructure that wishes to hand off a node resource during an upgrade (socket, namespace, process)
3. Infrastructure that must remain 100% available to support workloads (networking components, proxies).

In general, all of these benefit from minimizing the time between old pod shutting down and new pod starting up. MaxSurge allows components to arbitrarily approach zero disruption by careful tuning of their launch scripts and access to shared resources, such as sockets or shared disk.

Infrastructure invoked by Kubernetes components (CRI, CNI, CSI) can usually fall within the first category and may require some coordination from the invoking process to minimize downtime. For instance, the Kubelet may retry certain types of CSI errors transparently to mitigate brief disruption to a CSI plugin. Or the container runtime may retry certain CNI errors if the plugin is not available.

The second category of workload requires some coordination between the old and new container - for instance, reusing a host volume and checking for file locking on shared resources, or using the SO_REUSEPORT option to start listening on an interface and share old and new traffic. In general the workload author is assumed to understand how to minimize disruption and Kubernetes is only giving them an overlapping window of execution before beginning the termination of the old process. The readiness probe should be used by the workload author to manage this transition as in other workload flows.

The last category is the most difficult to achieve and generally combines categories 1 and 2 along with careful tuning. Networking plugins that provide pod network capability may have one or more daemon processes that are desirable to deliver containerized, but any disruption to those critical pods may impact other workloads. In most cases, the capability to overlap execution provided by the MaxSurge is sufficient to allow those components to adapt to zero-downtime updates.

In the future, [service topology](https://github.com/kubernetes/enhancements/issues/2004) will have implications for services implemented as daemonsets across all nodes. The update strategy for surge or drain will need to take into account topology, although the full details of that are outside the scope of this design. In general, service owners using daemonset surge will wish to maximize availability and minimize the risk of disruption during update.

### Risks and Mitigations

The primary risk is a bug in the implementation of the controller that causes excessive pod creations or deletions, as we have experienced during previous enhancements to workload controllers. The best mitigation for that scenario is unit testing to ensure the update strategy is stable and general purpose stress e2e testing of the controller.

Because we are widening validation for MaxUnavailable, we must ensure that during an upgrade old apiservers can still handle that field. The alpha release of this field would have special logic that, if MaxSurge is set and dropped, a value of MaxUnavailable 0 would be set to 1 (the minimum allowed unavailable). The alpha controller would also special case this check when the gate was off. When a cluster was upgraded to beta with the gate on by default, the old controller and apiservers would treat `MaxSurge != 0, MaxUnavailable == 0` as `MaxSurge == 0, MaxUnavailable == 1` until they themselves were upgraded.

## Design Details

### Implications to drain

DaemonSets currently ignore unschedulable, but triggering a drain of a node and choosing to delete daemonsets would ensure that if the old pod can be deleted
the daemonset controller immediately schedules a new pod onto that node when
MaxSurge is in play (because the invariant that there must be at least one
pod). If the old pod delays deletion, then the new pod has a chance to accept handoff from the old pod exactly like a normal rolling surge update.

### Test Plan

* Unit tests covering the daemonset controller behavior in all major edge cases
* E2E test for surge strategy that verifies expected recovery behavior and that the controller settles
  * Testing should set up conflicting rules like HostPort and verify that surge fails and the correct daemonset condition is set and events are generated.
	* A test should cover a pod going unready during rollout and verifying it is immediately replaced.

### Graduation Criteria

This will be added as a alpha field enhancement to DaemonSets with a backward compatible default. After sufficient exposure this field would be promoted to beta, and then to GA in successive releases. The feature gate for this field will be `DaemonSetUpdateSurge`.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/20190731-production-readiness-review-process.md.

The production readiness review questionnaire must be completed for features in
v1.19 or later, but is non-blocking at this time. That is, approval is not
required in order to be in the release.

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

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name:
    - Components depending on the feature gate:
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**

	No

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**

	Yes, when the feature gate is disabled the field is ignored and can be cleared.
	A workload using this alpha feature would no longer be able to surge and would
	fall back to the default MaxUnavailable value (which is minimum 1).

* **What happens if we reenable the feature if it was previously rolled back?**

	The field would become active and whatever new values were present would cause
	the surge feature to become active. If the field were changed the user would have
	to use the new alpha field.

* **Are there any tests for feature enablement/disablement?**

  A unit test will verify disablement ignores surge and behaves as MaxUnavailable=1

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

	No, the controller will perform roughly the same order of magnitude calls as
	for the normal strategy.

* **Will enabling / using this feature result in introducing new API types?**

	No.

* **Will enabling / using this feature result in any new calls to the cloud
provider?**

	No.

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**

	No, except for the explicit user chosen field on the daemonset.

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**

	No, only broken Daemonsets in surge configurations would fail to roll out.
	In both strategies, the readiness check gates the SLO of rollout.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**

	No, the calculations for this controller change are of the same magnitude as
	the existing flow.

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

- Initial PR:
