# KEP-5030: Integrate Volume Attach limit into cluster autoscaler

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
- [Cluster Autoscaler changes](#cluster-autoscaler-changes)
  - [Scaling a node-group that already has one or more nodes.](#scaling-a-node-group-that-already-has-one-or-more-nodes)
  - [Scaling from zero](#scaling-from-zero)
- [Kubernetes Scheduler change](#kubernetes-scheduler-change)
  - [Handling Node Readliness](#handling-node-readliness)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

As part of this KEP we are proposing that, tools like cluster-autoscaler are aware of number of volumes that can be attached to a node.


## Motivation

Currently cluster-autoscaler doesn’t take into account, volume-attach limit that a node may have when scaling nodes to support unschedulable pods.

This leads to bunch of problems:
- If there are unschedulable pods that require more volume than one supported by newly created nodes, there will still be unschedulable pods left.

- Since a node does not come up with a CSI driver typically, usually too many pods get scheduled on a node, which may not be supportable by the node in the first place. This leads to bunch of pods, just stuck.

Once cluster-autoscaler is aware of CSI volume attach limits, we can fix kubernete's builtin scheduler to not schedule pods to nodes that don't have CSI driver installed and if pods require given CSI volumes. Also since 
cluster-autoscaler isn't aware of CSI volume attach limits, when it scales nodes for pending pods it can't accurately determine how many nodes are required for pending pods that use CSI volumes. For example: if there are 20 pending pods(that use CSI volumes) and assuming cpu, memory and other critireas are met, cluster-autoscaler will not accurately take into account how many nodes are required to satisfy volume attach limits of a node.

After the fixes we are proposing in cluster-autoscaler are made, cluster-autoscaler should accurately calculate number of nodes it needs to spin up to satisfy volume constraints of pending pods. 

### Goals

- Modify cluster-autoscaler so as it is aware of CSI volume limits.
- Fix scheduler, so as it doesn't schedule pods that require given CSI volume to a node that doesn't have CSI driver installed.

### Non-Goals

- Deschedule pods that can't fit a node because of race conditions.
- Fixing other autoscalers like Karpenter is out of scope for current proposal.

## Proposal

As part of this proposal we are proposing changes into both cluster-autoscaler and kubernetes's built-in scheduler.

1. Fix cluster-autoscaler so as it takes into account attach limits when scaling nodes from 0 in a nodegroup.
2. Fix cluster-autoscaler so as it takes into account attach limits when scaling nodegroups with existing nodes.
3. Fix kubernetes built-in scheduler so as we do not schedule pods to nodes that doesn't have CSI driver installed.

While, changes into both CAS and scheduler can happen behind same featuregate that is being proposed in this enhancement, we propose delaying defaule enablement of scheduler change
that prevents scheduling of pods to a node that doesn't have CSI driver installed until a release when Cluster-AutoScaler(CAS) changes have been GAed and meet N-3 version skew critirea. See - version skew section for more information.


### User Stories (Optional)

#### Story 1
- User has more than one pod that is pending because no existing node has any attach limit left.
- Cluster autoscaler evaluates existing nodegroups.
- It picks a nodegroup based on existing critireas and it accurately determines number of nodes it needs to spin up based on volumes that pending pods require.

#### Story 2
- A Kubernetes admin has one or more node where CSI driver is not installed. 
- Without explicitly tainting the node or using node affinity in worklods, nodes which don't have CSI driver installed aren't used for scheduling pods that require volume.

### Notes/Constraints/Caveats (Optional)

Scheduler changes must be vendored into CAS repository prior to release, so as both scheduler and CAS can work with same feature gate.

But in independently running scheduler(i.e out of CAS process) in k8s cluster, the feature-gate will not be enabled by default until `VolumeLimitScaling` featuregate
has gone GA in CAS and satisifies version skew critirea of CAS and kube-scheduler.

### Risks and Mitigations

While, changes into both CAS and scheduler can happen behind same featuregate that is being proposed in this enhancement, we propose delaying default enablement of scheduler change
that prevents scheduling of pods to a node that doesn't have CSI driver installed until a release when Cluster-AutoScaler(CAS) changes have been GAed and meet N-3 version skew critirea. See - version skew section for more information.


<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

## Cluster Autoscaler changes

We can split the implementation in cluster-autoscaler in two parts:
- Scaling a node-group that already has one or more nodes.
- Scaling a node-group that doesn’t have one or more nodes (Scaling from zero).

### Scaling a node-group that already has one or more nodes.

1. To ensure that nodes which were recently started but do not have CSI driver installed yet are considered as upcoming nodes and hence are properly handled via scaleup operation, we propose a mechanism similar to recently introduced mechanism for DRA resources. See section - "Handling Node Readliness" for more details.

2. We propose that, we add volume limits and installed CSI driver information to framework.NodeInfo objects. So -

```
type NodeInfo struct {
....
....
// CSINodes contains all CSINodes exposed by this Node.
CSINode *storagev1.CSINode
..
}

```

3. We propose that, when saving `ClusterState` , we capture and add `CSINode` information in cluster snapshot. The updated signature of `SetClusterState` function would look like:

```go
SetClusterState(nodes []*apiv1.Node, scheduledPods []*apiv1.Pod, draSnapshot *drasnapshot.Snapshot, csiSnapshot *csisnapshot.Snapshot) error
```

Both delta and basic snapshot implementation would store `csiSnapshot` along with dra and other information.

4. Since scaling of a nodegroup requires creation of santized templateNodeInfo from existing `nodeInfo` objects, we need to ensure that we are creating santized `CSINode` objects from real `CSINode` objects associated with existing `nodeInfo` object in nodegroup. We need to make associated changes into `node_info_utils.go` to take that into account:

```go
templateNodeInfo := framework.NewNodeInfo(sanitizedExample.Node(), sanitizedExample.LocalResourceSlices, expectedPods...)
if example.CSINode != nil {
  templateNodeInfo.AddCSINode(createSanitizedCSINode(example.CSINode, templateNodeInfo))
}
```


5. We propose that, when getting nodeInfosForGroups , the return nodeInfo map also contains csiNode information, which can be used later on for scheduling decisions.

```
nodeInfosForGroups, autoscalerError := a.processors.TemplateNodeInfoProvider.Process(autoscalingContext, readyNodes, daemonsets, a.taintConfig, currentTime)
```

This should generally work out of box when  nodeInfo is extracted from previously stored cluster snapshot via:

```go
// will return wrapped framework.NodeInfo with both DRA and CSINode information
ctx.ClusterSnapshot.GetNodeInfo(node.Name)
```

Please note that, we will have to handle the case of scaling from 0, separately from
scaling from 1, because in former case - no CSI volume limit information will be available
If no node exists in a NodeGroup.

6. We further propose creation or extension of existing `StorageInfos` interface, so as both scheduler and CAS can work with the previously created fake `CSINode` objects. Without this change, both the hinting_simulator and estimator, which triggers scheduler plugin runs will not be able to find the templated `CSINode` object we created in previous step.

Making aforementioned changes should allow us to handle scaling of nodes from 1.

### Scaling from zero

Scaling from zero should work similar to scaling from 1, but the main problem is - we do not have NodeInfo which can tell us what would be the CSI attach limit on the node which is being spun up in a NodeGroup.

We propose that we introduce similar annotation as CPU, Memory resources in cluster-api to process attach limits available on a node.

We have to introduce similar mechanism in various cloudproviders which return Template objects to incorporate volume limits. This will allow us to handle the case of scaling from zero.


## Kubernetes Scheduler change

We also propose that, if given node is not reporting any installed CSI drivers, we do not schedule pods that need CSI volumes to that node.

The proposed change is small and a draft PR is available here - https://github.com/kubernetes/kubernetes/pull/130702

This will stop too many pods crowding a node, when a new node is spun up and node is not yet reporting volume limits.

But this alone is not enough to fix the underlying problem. Cluster-autoscaler must be fixed so as it is aware of attach limits of a node via CSINode object.

We also need to ensure that `StorageInfos` interface that is shared between CAS and scheduler is extended for `CSINode` objects, so as CAS can run scheduler plugins with templated `CSINode` objects.

### Handling Node Readliness 

We propose to handle node readiness in similar way it was handled for DRA in - https://github.com/kubernetes/autoscaler/pull/8109 . The basic idea is, we compare using `TemplateNodeInfo`, what would be the expected CSI drivers available on the node and if node doesn't yet have those drivers installed, we consider node as not-ready.

Alternatives:

1.We propose a similar label as GPULabel added to the node that is supposed to come up with a CSI driver. This would ensure that, nodes which are supposed to have a certain CSI driver installed aren’t considered ready - https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/core/static_autoscaler.go#L979 until CSI driver is installed there.

However, we also propose that a node will be considered ready as soon as corresponding CSI driver is being reported as installed via corresponding CSINode object.

A node which is ready  but does not have CSI driver installed within certain time limit will be considered as NotReady and removed from the cluster.

2. A more exhaustive solution to node readiness is being proposed in - https://github.com/kubernetes/enhancements/pull/5416 , we are open to the idea of using it when it becomes usable from CAS. 


### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

After this proposal is implemented, simulated scheduling in CAS should work with fake `CSINode` objects 
which report real volume limits and hence scheduling should accurately count number of required nodes
for pending pods.

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- k8s.io/autoscaler/cluster-autoscaler/core: 06/10/2025 - 77.3%

##### Integration tests

None

##### e2e tests

We are planning to add e2e tests that verify behaviour of cluster autoscaler when it scales nodes for pods that require volumes.

We will add tests that validate both scaling from 0 and scaling from 1 use cases.

### Graduation Criteria

#### Alpha

- All of the planned code changes for alpha will be done in cluster-autoscaler and kubernetes (scheduler in particular) repository. 
- We plan to implement changes in cluster-autoscaler so as it can consider volume limits when scaling cluster.
- Make changes in `kube-scheduler` so as it can stop scheduling of pods that require CSI volume if underlying CSI volume is not installed on the node.
- Initial e2e tests completed and enabled.
- All of the changes in CAS and kube-scheduler will be behind `VolumeLimitScaling` featuregate.

<!---
#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

### Upgrade / Downgrade Strategy

In general Upgrade and Downgrade of `cluster-autoscaler` should be fine, it just means how CA scales nodes will 
change.

We do not want to downgrade to a version of CAS that has `VolumeLimitScaling` disabled while kube-scheduler has it enabled. 
See Version Skew strategy for more details.

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

### Version Skew Strategy

This feature has no interaction with kubelet and other components running on the node. 

However, if this feature is enabled in kube-scheduler (not part of CAS but externally running kube-scheduler) and CAS is older and has this feature disabled, 
then we may run into an issue where CAS creates a node to satisfy pod requirements, but kube-scheduler will not schedule pods to the node until CSI driver
is installed. 

To satisfy this version skew, we propose:

1. We will only enable this feature in kube-scheduler *after* corresponding feature-gate has been GAed in CAS and meets version skew criteria of CAS and kube's control-plane components.
2. What this means is, when we enable `VolumeLimitScaling` feature in kube-scheduler, last 3 versions of CAS should already have this feature enabled and running by default and hence there should not be any version skew issues in case of downgrades.

Just to make it clearer, although it should never happen - if feature-gate is disabled in scheduler but enabled in CAS, that will *never* be a problem, because in that case, CAS will probably take into account CSI volume limits when creating nodes and since kube-scheduler *yet* doesn't have limit for upcoming nodes, it will place those pods on those nodes without any restriction (like how it does today).

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

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

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `VolumeLimitScaling`
  - Components depending on the feature gate: `cluster-autoscaler`, `kube-scheduler`
- [x] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
    Yes, it should require restart of CAS and kube-scheduler.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

Yes, it will cause cluster-autoscaler to consider volume limits when scaling nodes. It will also cause scheduler to not schedule pods to nodes that don't have CSI driver.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

For CAS:
- Yes. This will simply cause old behaviour to be restored.

For kube-scheduler:
- This feature will *not* be enabled until the CAS this feature GAed and meets the version skew requirement, in which case disabling the feature in scheduler will simply cause
scheduler to place pods on a node, which does not have CSI driver installed (current behaviour basically). Just to make it clearer, although it should never happen - if feature-gate is disabled in scheduler but enabled in CAS, that is not a problem.

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

For CAS:

- The feature will start working same as before. 

For kube-scheduler:

- It should be work fine

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout of this feature in CAS would be considered failing if somehow CAS is not creating 
appropriate number of nodes to accommodate CSI volumes required by pods.

A rollout of this feature in kube-scheduler would be considered failing if kube-scheduler is still
placing pods to nodes that doesn't have CSI driver installed.

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

In CAS if `unschedulable_pods_count` metric consistently reports a number of pods pending of scheduling, in general that would be
a good indication that something is broken in CAS. In general, this in itself doesn't mean those pending pods use CSI volumes
but we are considering enhancing existing metrics with that information.

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

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
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

Depends on cluster-autoscaler running in the cluster.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

After the changes in this PR are merged, CAS now may have to read `CSINode` objects 
before scaling decisions, but CAS was *already* reading `CSINode` objects via 
scheduler plugins it vendors, because those plugins need `CSINode` listers.

Overall - this should not result in any new API calls.

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

###### Will enabling / using this feature result in introducing new API types?

In the v1.35 alpha release, we are not considering introducing new API types yet. 

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

In general I think, it should result in not any new calls to the cloud provider. If anything, once 
this feature is enabled in both CAS and kube-scheduler, it should prevent scheduling of pods to the nodes
which can't reasonably accommodate them. And hence it should result in reduction of API calls we make
to the cloudprovider.

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

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

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

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

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives


Certain Kubernetes vendors taint the node when a new node is created and CSI driver has logic to remove the taint when CSI driver starts on the node.
- https://github.com/kubernetes-sigs/azuredisk-csi-driver/pull/2309

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
