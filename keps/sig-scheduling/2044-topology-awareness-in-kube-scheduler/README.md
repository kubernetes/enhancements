# Topology awareness in Kube-scheduler

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
- [Goals](#goals)
- [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Design Consideration](#design-consideration)
  - [Changes to the API](#changes-to-the-api)
  - [Plugin implementation details](#plugin-implementation-details)
  - [Description of the Algorithm](#description-of-the-algorithm)
  - [Test Plan](#test-plan)
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
- [Implementation history](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Enable this capability out-of-tree](#enable-this-capability-out-of-tree)
  - [1:1 worker pod to node assignment](#11-worker-pod-to-node-assignment)
- [Infrastructure Needed](#infrastructure-needed)
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
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

Kubernetes clusters composed of nodes with complex hardware topology are becoming more prevalent.
[Topology Manager](https://kubernetes.io/docs/tasks/administer-cluster/topology-manager/) was
introduced in kubernetes as part of kubelet in order to extract the best performance out of
these high performance hybrid systems. It performs optimizations related to resource allocation
in order to make it more likely for a given pod to perform optimally. In scenarios where
Topology Manager is unable to align topology of requested resources based on the selected
Topology Manager policy, the pod is rejected with Topology Affinity Error.
[This](https://github.com/kubernetes/kubernetes/issues/84869) kubernetes issue provides
further context on how runaway pods are created because the scheduler is topology-unaware.

In order to address this issue, scheduler needs to choose a node considering resource availability
along with underlying resource topology and Topology Manager policy on the worker node.

This enhancement proposes changes to make kube-scheduler aware of node NUMA topology when making scheduling decisions

The changes/artifacts proposed as part of this KEP are:
1. A new scheduler plugin that makes topology-aware placement decisions
2. A new resource object, `NodeResourceTopology` to communicate NUMA status between kubelet and kube-scheduler
3. Kubelet changes to populate `NodeResourceTopology`

## Motivation

After Topology Manager was introduced, the problem of launching pod in the cluster where worker
nodes have different NUMA topology and different amount of resources in that topology became
actual. Pod could be scheduled on the node where total amount of resources are enough, but
resource distribution could not satisfy the appropriate Topology policy. In this case the pod
failed to start. Much better behaviour for scheduler would be to select appropriate node where
kubelet admit handlers may pass.

In order to enable topology aware scheduling in Kubernetes, resource topology information of the
nodes in the cluster needs to be exposed to the scheduler so that it can use it to make a more
informed scheduling decision. This KEP describes how it would be implemented.

## Goals

- Make scheduling process more precise when we have NUMA topology on the worker node.

## Non-Goals

- Change the PodSpec to allow requesting a specific node topology manager policy
- Changes to the Topology Manager and its policies.
- API changes with the Operating System or external components as all the required
  information is already available in Kubelet running on linux flavours that support NUMA nodes.
- Enable Windows systems to support CPU Manager or memory manager and hence Topology aware
  Scheduling in general.

## Proposal

Kube-scheduler plugin will be moved from kuberntes-sigs/scheduler-plugin (or out-of-tree)
plugin into the main tree as a built-in plugin. This plugin implements a simplified version
of Topology Manager and hence is different from original topology manager algorithm. Plugin
would be disabled by default and when enabled would check for the ability to run pod only in
case of single-numa-node policy Topology Manager policy on the node, since it is the most strict
policy, it implies that the launch on the node with other existing policies will be successful
if the condition for single-numa-node policy passed for the worker node.

To work, this plugin requires topology information of the available resources for each NUMA cell on worker nodes.

Kubelet will be responsible for collecting all necessary resource information of the pods,
based on allocatable resources on the node and allocated resources to pods. The NUMA nodes
would be represented as NUMA cells in Kubelet and the NodeResourceTopology would capture the 
resource information at a NUMA cell level granularity.

Once the information is captured in the NodeResourceTopology API, the scheduler can refer to
it like it refers to Node Capacity and Allocatable while making a Topology-aware Scheduling decision.

### User Stories

As a Kubernetes cluster operator managing a cluster with multiple bare metal worker nodes with NUMA
topology and Topology Manager enabled, I want the scheduler to be Topology-aware in order to ensure that the
pods are only placed on nodes where requested resources can be appropriately aligned by Topology Manager based on its policy.
The scheduler shouldn't send the pod to the node where kubelet will reject it with "Topology Affinity Error".
This issue leads to runaway pod creation if the pod is part of a Deployment or ReplicaSet as the associated controllers
notices the pod failure and keep creating another pod.


### Risks and Mitigations

Topology Manager on the worker node knows exact resources and their NUMA node allocated to pods but the and node resource
topology information is delivered to Topology aware scheduler plugin with latency meaning that the scheduler will not know
actual NUMA topology until the information of the available resources at a NUMA node level is evaluated in the kubelet which
could still lead to scheduling of pods to nodes where they won't be admitted by Topology Manager.

This can be mitigated if kube-scheduler provides a hint of which NUMA ID a pod should be assigned and Topology Manager on the
worker node takes that into account.

## Design Details

- add a new flag in Kubelet called `ExposeNodeResourceTopology` in the kubelet config or command line argument called `expose-noderesourcetopology` which allows 
  the user to specify when they would like Kubelet to compute and expose resource hardware topology information.
- The `ExposeNodeResourceTopology` flag is received from the kubelet config/command line args is propogated to the Container Manager.
- Based on the resources allocated to a pod, the topology associated with the resources is evaluated and populated as part of the NodeResourceTopology.
- Kubelet will collect information about resources allocated to running pods along with their topology, based on allocatable resources of the node and consumed
  resources by pods it will populate available resources with the associated topology information in NodeResourceTopology, where a NodeResourceTopology instance
  would represent a worker node.
  The name of the CRD instance is the name of the worker node
- A new in-tree scheduler plugin `NodeResourceTopologyMatch` is created that makes topology-aware placement decisions implements a simplified version of Topology
  Manager as part of the filter extension point to filter out nodes that are not suitable for the workload based on the resource request and the obtained 
  NodeResourceTopology information corresponding to that worker node. The scoring extension point to determine a score based on a configurable strategy
  would be done as a Beta feature. 
- The scheduler plugin `NodeResourceTopologyMatch` would be disabled by default and when enabled would also check for the ability to run pod only
  in case of Topology Manager policy of single-numa-node policy is configured on the node. Since it is the most strict policy, it implies that the launch
  on the node with other existing policies will be successful if the condition for single-numa-node policy passed for the worker node.
 

### Design Consideration

By default Kubelet would be resposible to fill the NodeResourceTopology instances, but a design consideration is to architect the
solution in such a way to allow future extension to facilitate the cloud administratore to disable the feature and introduce a custom external agents to fill the NodeResourceTopology data.

### Changes to the API

Code responsible for working with NodeResourceTopology API will be placed in the staging directory
at path staging/src/k8s.io/api/node/v1/types.go.

```go

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeResourceTopology is a specification for a hardware resources
type NodeResourceTopology struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	TopologyPolicies []string 		   `json:"topologyPolicies"`
	Cells            map[string]Cell   `json:"cells"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeResourceTopologyList is a list of NodeResourceTopology resources
type NodeResourceTopologyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NodeResourceTopology `json:"items"`
}

// Cell is the spec for a NodeResourceTopology resource
type Cell struct {
    Name       string           `json:"name"`
    Type       string           `json:"type"`
    Parent     string           `json:"parent,omitempty"`
    Costs      []CostInfo       `json:"costs,omitempty"`
    Attributes []AttributeInfo  `json:"attributes,omitempty"`
    Resources  []ResourceInfo   `json:"resources,omitempty"`
}

type ResourceInfo struct {
    Name        string `json:"name"`
    Allocatable string `json:"allocatable"`
    Capacity    string `json:"capacity"`
}

type CostInfo struct {
    Name  string `json:"name"`
    Value int	 `json:"value"`
}

type AttributeInfo struct {
    Name  string `json:"name"`
    Value string `json:"value"`
}
// Kubelet writes to NodeResourceTopology
// and scheduler plugin reads from it 
// Real world example of how these fields are populated is as follows:
// Cells:
//   Name:     node-1
//   Type:     Node
//   Costs:
//     Name:   node-0
//     Value:  20
//     Name:   node-1
//     Value:  10
//   Attributes:
//    Name: performance-profile
//    Value: high-performance-profile
//   Resources:
//     Name:         example.com/deviceB
//     Allocatable:  2
//     Capacity:     2
//     Name:         example.com/deviceA
//     Allocatable:  2
//     Capacity:     2
//     Name:         cpu
//     Allocatable:  4
//     Capacity:     4

```


### Plugin implementation details

### Description of the Algorithm

The algorithm which of the scheduler plugin is as follows:

1. At the filter extension point of the plugin, the QoS class of the pod is determined, in case it is a best effort pod or the 
   Topology Manager Policy configured on the node is not single-numa-node, the node is not considered for scheduling
1. The Topology Manager Scope is determined.
1. While interating through the containers of a pod
	* A bitmask is created where each bit corresponds to a NUMA cell and are all the bits are set. If the resources cannot be aligned on the NUMA cell,
	the bit should be unset.
	* For each resource requested in a container, a new resourceBitmask is created to determined which NUMA cell is a good fit for each resource
		1. If requested resource cannot be found on a node, it is unset as available NUMA cell
		1. If an unknown resource has 0 quantity, the NUMA cell should be left set.
	* The following checks are performed:
		1. Add NUMA cell to the resourceBitmask if resource is cpu and it's not guaranteed QoS, since cpu will flow
		1. Add NUMA cell to the resourceBitmask if resource is memory and it's not guaranteed QoS, since memory will flow
		1. Add NUMA cell to the resourceBitmask if zero quantity for non existing resource was requested
		1. otherwise check amount of resources
	* Once the resourceBitMark is determined it is ANDed with the cummulative bitmask
4. If resources cannot be aligned from the same NUMA cell for a container, alignment cannot be achieved for the entire pod and the resource cannot be 
   aligned in case of the pod under consideration. Such a pod is returned with a Status Unschedulable 

### Test Plan

It would be ensured that the components developed or modified for this feature can be easily tested.

* Unit Tests

Unit test for scheduler plugin (pkg/scheduler/framework/plugins/noderesources/node_resource_topology_match.go)
pkg/scheduler/framework/plugins/noderesources/node_resource_topology_match_test.go which test the plugin.

Separate tests for changes to Kubelet will also should be implemented.

* Integration Tests
   *  Default configuration (this plugin is disabled)
     * no side effect on basic scheduling flow (and performance)

   *  Enable this plugin
     * basic workflow of this feature works (decision by scheduler is admitted by kubelet)
     * basic negative path of this feature works (decision by scheduler is rejected by kubelet)

* End-to-end tests

Integration and End-to-end would Implementation of it does not constitute a difficulty, but requires appropriate multi-numa hardware for comprehensive testing of this feature. Comprehensive E2E testing of this would be done in order to graduate this feature from Alpha to Beta.




### Graduation Criteria

#### Alpha

- [ ] Introducing `NodeResourceTopology` API to faciliatate communication between kubelet and kube-scheduler 
- [ ] A new scheduler plugin `NodeResourceTopologyMatch` that makes topology-aware placement decisions
    - [ ] Implementation of Filter extension point
- [ ] Kubelet changes to populate `NodeResourceTopology` 
- [ ] Unit tests and integration tests from [Test plans](#test-plans).

#### Beta
 
- [ ] Implementation of Score extension point
- [ ] Add node E2E tests.
- [ ] Provide beta-level documentation.

#### GA

- Add Conformance Tests
- More rigorous testing—e.g., downgrade tests and scalability tests


### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

No changes are required on upgrade to maintain previous behaviour.

It is possible to downgrade kubelet on a node that was is using this capability by simply
disabling the `ExposeNodeResourceTopology` feature gate and disabling the `NodeResourceTopologyMatch`
scheduler plugin in case this plugin was enabled in the KubeScheduler configuration.   

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

Feature flag will apply to kubelet only, so version skew strategy is N/A.
This feature involves changes in the Kube-Scheduler and Kubelet. In case an older version of 
Kubelet is used with the updated scheduler plugin

In case an older version of Kubelet is used with an updated version of Scheduler,
the scheduler plugin even if enabled should behave as as it does without the introduction of 
the plugin. 

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

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.
-->

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ExposeNodeResourceTopology`
  - Components depending on the feature gate: kubelet
- [X] Enable Scheduler scheduler plugin `NodeResourceTopologyMatch` in the KubeScheduler config
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?  Yes, Feature gate must be set on kubelet start. To disable, kubelet must be
    restarted. Hence, there would be brief control component downtime on a
    given node.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
	See above; disabling would require brief node downtime.

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

Yes, Kubelet will collect information about resources allocated to running pods along with their topology, based on allocatable resources of the node and consumed resources by pods it will populate available resources with the associated topology information in NodeResourceTopology for the node.
In case a workload cannot be aligned on a NUMA cell where Toplogy Manager Policy is configured as single-numa-node NUMA, the chances of scheduling that workload
to such a node should be significantly reduced.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes, disabling the feature gate in Kubelet and disabling the scheduler plugin shuts down the feature completely.

###### What happens if we reenable the feature if it was previously rolled back?
No changes.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->

Specific e2e test will be added to demonstrate that the default behaviour is preserved when the feature gate and scheduler plugin is disabled, or when the feature is not used (2 separate tests)

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
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
-->

No.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->
Yes, the scheduler will be accessing the NodeResourceTopology instance corresponding to the node

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

Yes, NodeResourceTopology API is introduced and an instance per node would be created to be referenced by the scheduler plugin

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
Yes, there would NodeResourceTopology instances equal to the number of nodes in the cluster.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

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

##  Implementation history

- 2021-06-10: Initial KEP sent out for review, including Summary, Motivation, Proposal, Test plans and Graduation criteria.
- 2021-07-02: Updated version after first round of reviews


## Drawbacks

Topology Manager on the worker node knows exact resources and their NUMA node allocated to pods but the and node resource
topology information is delivered to Topology aware scheduler plugin with latency meaning that the scheduler will not know
actual NUMA topology until the information of the available resources at a NUMA node level is evaluated in the kubelet which
could still lead to scheduling of pods to nodes where they won't be admitted by Topology Manager.

## Alternatives


### Enable this capability out-of-tree
1. An external daemon to expose resource information along with NUMA topology of a node as a [CRD][1].

    The daemon runs on each node in the cluster as a daemonset and collects resources allocated to running pods along with associated topology (NUMA nodes) and provides information of the available resources (with numa node granularity) through a CRD instance created per node. [Enhancing](https://github.com/kubernetes-sigs/node-feature-discovery/issues/333) Node Feature Discovery [daemon](https://github.com/kubernetes-sigs/node-feature-discovery) or implementing standalone component like [Resource Topology Exporter](https://github.com/k8stopologyawareschedwg/resource-topology-exporter) would allow the information to be exposed as CRs corresponding to the node. The name of the CR is same as the name of the worker node. The daemon would use Podresources interface of the kubelet is described in [pkg/kubelet/apis/podresources/v1/api.proto](https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/apis/podresources/v1/api.proto) to collect used resources on a worker node along with their NUMA assignment to account the available resources on each NUMA node  


2. Out-of-tree scheduler plugin

    An out-of-tree [Node Resource Topology](https://github.com/kubernetes-sigs/scheduler-plugins/blob/master/pkg/noderesourcetopology/README.md) scheduler plugin would use the CRs exposed by the exporter daemon to make a Topology aware placement decision.

Cons:
1. [Code][2] responsible for working with NodeResourceTopology CRD API is stored separately in Topology-aware Scheduling github organization in [noderesourcetopology-api](https://github.com/k8stopologyawareschedwg/noderesourcetopology-api)
2. The implementation where the exporter uses PodResource API means that the API endpoints such as List and GetAllocatableResources require the daemon to consume the endpoints periodically. Because of this, in case of a huge influx of 
pods and polling interval being large, the accounting might not happen correctly leading to the scenario we are trying to avoid scheduling pods on nodes where kubelet will reject it with "Topology Affinity Error" and end up. A watch endpoint in podresource API might solve this issue.


### 1:1 worker pod to node assignment

So apart from kubelet and daemonsets, the pod will take the whole node and the application is responsible for forking processes and assign them to NUMA cells.

Cons:
Topology Manager can deal with the alignment of resources at [container and pod scope](https://kubernetes.io/docs/tasks/administer-cluster/topology-manager/#topology-manager-scopes). So if we have a pod with multiple containers, a process running inside the containers can be assigned to a NUMA cell. But
it is not possible in case there are multiple processes running inside the same container even if the application is smart and is capacble of forking processes
there is not way in Topology Manager in Kubelet to perform assignment of resource from the same NUMA cell for a process level.

## Infrastructure Needed

Hardware with Multi-NUMA systems for e2e tests


[1]: https://docs.google.com/document/d/12kj3fK8boNuPNqob6F_pPU9ZTaNEnPGaXEooW1Cilwg/edit
[2]: https://github.com/kubernetes/noderesourcetopology-api
