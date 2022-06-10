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
    - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha -&gt; Beta](#alpha---beta)
    - [Beta -&gt; GA](#beta---ga)
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
	// number is calculated from percentage by rounding up.
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
	// Default value is 0.
	// Example: when this is set to 30%, at most 30% of the total number of nodes
	// that should be running the daemon pod (i.e. status.desiredNumberScheduled)
	// can have their a new pod created before the old pod is marked as deleted.
	// The update starts by launching new pods on 30% of nodes. Once an updated
	// pod is available (Ready for at least minReadySeconds) the old DaemonSet pod
	// on that node is marked deleted. If the old pod becomes unavailable for any
	// reason (Ready transitions to false, is evicted, or is drained) an updated
	// pod is immediately created on that node without considering surge limits.
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

A user who uses HostNetwork but does not declare HostPorts and attempts to use MaxSurge with processes that listen on the host network should see errors from the network stack when their process attempts to bind a port (such as `cannot bind to address: port in use`) and the new pod will crash and go into a crashloop. Users should expect to see these failures as they would any other "my application does not start on Kubernetes" error via pod status, daemonset status conditions, and pod logs.

Building a daemonset that hands off between two host level processes with any degree of coordination is an advanced topic and is up to the workload author. The simplest daemonsets may use pod network without any host level sharing and will benefit significantly from maxSurge during updates by reducing downtime at the cost of extra resources. As more complex sharing (host network, disk resources, unix domain sockets, configuration) is needed, the author is expected to leverage custom readiness probes, process start conditions, and process coordination mechanisms (like disks, networking, or shared memory) across pods. Debugging those interactions will be in the domain of the workload author.


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


#### Prerequisite testing updates
[x] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

##### Unit tests

```
`k8s.io/kubernetes/pkg/apis/apps/validation`                     `06/06/2022`:     `90.6% of statements` `The tests added for the current feature in this package touches the daemonSet Spec field. No new tests are needed for promotion to GA`
`k8s.io/kubernetes/pkg/apis/apps/validation/validation.go:387`:	 `06/06/2022`:		`100.0% of statements`
`k8s.io/kubernetes/pkg/controller/daemon`: `06/06/2022`: `70.7% of statements`    `The tests added for the current feature in this package touches the daemonSet update strategies. No new tests are needed for promotion to GA`
`k8s.io/kubernetes/pkg/registry/apps/daemonset`: `06/06/2022`: `31.1% of statements`  `The tests added for the current feature in this package makes sure that the kubernetes version upgrades/downgrades won't have any impact on the new field to the daemonSet api when persisting to etcd. No new tests are needed for promotion to GA`
`k8s.io/kubernetes/pkg/registry/apps/daemonset/strategy.go:129`:	`06/06/2022`:	`100.0% of statements`
```
##### Integration tests

A new integration which exercises maxSurge when `RollingUpdate` is used as update strategy will be added to [DS integration test suite](https://github.com/kubernetes/kubernetes/blob/master/test/integration/daemonset/daemonset_test.go)

##### e2e tests

An e2e test which exercises maxSurge when `RollingUpdate` is used as update strategy is added for daemonsets.

- should surge pods onto nodes when spec was updated and update strategy is RollingUpdate: [test grid](https://storage.googleapis.com/k8s-triage/index.html?test=should%20surge%20pods)


### Graduation Criteria

This will be added as a alpha field enhancement to DaemonSets with a backward compatible default. After sufficient exposure this field would be promoted to beta, and then to GA in successive releases. The feature gate for this field will be `DaemonSetUpdateSurge`.

#### Alpha
  - Complete feature behind a featuregate
  - Have proper unit and e2e tests

#### Alpha -> Beta
  - Gather feedback from the community

#### Beta -> GA
   Atleast one of example of user benefitting from this feature:
   - OpenShift has few critical [DS](https://github.com/openshift/cluster-dns-operator/blob/d87dd223e67c476220451d254d878209c50324a7/pkg/operator/controller/controller_dns_node_resolver_daemonset.go#L80) where maxSurge is beneficial


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
    - Feature gate name: `DaemonSetUpdateSurge`
    - Components depending on the feature gate: `kube-apiserver`, `kube-controller-manager`
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

	Yes, when the feature gate is disabled the field is ignored and can be cleared by
  an end user. A workload using this alpha feature would no longer be able to surge
  and would fall back to the default MaxUnavailable value (which is minimum 1).

* **What happens if we reenable the feature if it was previously rolled back?**

	The field would become active and whatever new values were present would cause
	the surge feature to become active. If the field name were changed old values
  would be lost and the controller would default to using maxUnavailable 1.

  To clear the field from etcd, disable the gate and perform a no-op PUT on every
  daemonset.

* **Are there any tests for feature enablement/disablement?**

  A unit test will verify disablement ignores surge and behaves as MaxUnavailable=1

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  It shouldn't impact already running workloads. This is an opt-in feature since users need to explicitly set the MaxSurge parameter in the DaemonSetSet spec's RollingUpdate i.e `.spec.strategy.rollingUpdate.maxSurge` field.
  if the feature is disabled the field is preserved if it was already set in the presisted DaemonSetSet object, otherwise it is silently dropped.

* **What specific metrics should inform a rollback?**
  MaxSurge in DaemonSet doesn't get respected and additional surge pods won't be
  created. We consider the feature to be failing if enabling the featuregate and giving
  appropriate value to MaxSurge doesn't cause additional surge pods to be created.


* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Manually tested. No issues were found when we enabled the feature gate -> disabled it ->
  re-enabled the feature gate. Upgrade -> downgrade -> upgrade scenario was tested manually.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**
  None

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  By checking the DaemonSetSet's `.spec.strategy.rollingUpdate.maxSurge` field. The additional workload pods created should be respecting the value specified in the
  maxSurge field.


* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [x] Other (treat as last resort)
    - Details: The number of pods that are created above the desired amount of pods during an update when this feature is enabled can be compared to maxSurge value available in
    the DaemonSetSet definition. This can be used to determine the health of this feature.
    The existing metrics like `kube_daemonset_status_number_available` and `kube_daemonset_status_number_unavailable` can be used to track additional pods created

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  All the surge pods created should be within the value(% or number) of maxSurge field provided 99.99% of the time. The additinal pods created should ensure that the workload
  service is available 99.99% of time during updates.

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  None. It is part of kube-controller-manager.


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
This feature will not work if the API server or etcd is unavailable as the controller-manager won't be even able get events or updates for DaemonSetSets. If the API server and/or etcd is unavailable during the mid-rollout, the featuregate would not be enabled and controller-manager wouldn't start since it cannot communicate with the API server

* **What are other known failure modes?**
  - MaxSurge not respected and too many pods are created
    - Detection: Looking at `kube_daemonset_status_number_available` and `kube_daemonset_status_number_unavailable` metrics.
    - Mitigations: Disable the `DaemonSetUpdateSurge` feature flag
    - Diagnostics: Controller-manager when starting at log-level 4 and above
    - Testing: Yes, e2e tests are already in place
  - MaxSurge not respected and very few pods are created. This causes the workloads to be not be available at 99.99%
    - Detection: Looking at `kube_daemonset_status_number_available` and `kube_daemonset_status_number_unavailable` metrics.
    - Mitigations: Disable the `DaemonSetUpdateSurge` feature flag
    - Diagnostics: Controller-manager when starting at log-level 4 and above
    - Testing: Yes, e2e tests are already in place
  - maxUnavailable should be set to 0 even when maxSurge is configured
    - Detection: Looking at the `.spec.strategy.rollingUpdate.maxSurge` and `.spec.strategy.rollingUpdate.maxUnavailable`
    - Mitigations: Setting maxUnavailable to appropriate value
    - Diagnostics: Controller-manager when starting at log-level 4 and above
    - Testing: Yes, e2e tests are already in place

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- 2021-02-09: Initial KEP merged
- 2021-03-05: Initial implementation merged
- 2021-04-30: Graduate the feature to Beta proposed
- 2022-05-10: Graduate the feature to stable proposed