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
  - [Handling Node Readiness](#handling-node-readiness)
  - [When it is safe to Prevent pod placement?](#when-it-is-safe-to-prevent-pod-placement)
    - [What happens if cluster-admin opts-in to prevent pod scheduling but autoscaler does not have CSI attach limit awareness?](#what-happens-if-cluster-admin-opts-in-to-prevent-pod-scheduling-but-autoscaler-does-not-have-csi-attach-limit-awareness)
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
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
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

Fix cluster-autoscaler (CAS) to be aware of node's volume attach limits when scaling new nodes and prevent scheduler from placing pods on nodes that do not have a particular CSI driver installed.


## Motivation

When scaling new nodes to satisfy pending pods in a cluster, currently cluster-autoscaler (CAS) does not take into account volume attach limits (available via CSI) an upcoming node may have, this could result in insufficient number of nodes created to satisfy pending pods. With this KEP, we will make changes into CAS so that when running simulations to estimate number of nodes necessary to satisfy pending pods or when running scheduler simulations on upcoming nodes, it takes into account CSI volume attach limits via templated `CSINode` objects.

There is also a gap in implementation of `NodeVolumeLimits` scheduler plugin which was left intentionally to take into account the fact that, CAS will run this plugin without any templated `CSINode` objects during creation of new nodes and hence plugin permits placement of unlimited number of pods to nodes even if no CSI driver is installed on them. With this KEP - we aim to close the gap in `NodeVolumeLimits` scheduler plugin, so that scheduler will not place pods on nodes which aren't reporting any CSI driver information, if a CSI driver decides to do so.

To summarize:

- Scheduler CSI plugin assumes that "no information about a CSI driver published in a CSINode" means "no limits for volumes from that driver".
- For existing Nodes with CSI driver information already published, CA correctly takes the volume limits into account when running scheduler filters in simulations (e.g. when packing pending Pods on existing Nodes in the cluster at the beginning of the loop).
- For fake "upcoming" Nodes created in-memory by CA during scale-up simulations the corresponding "upcoming" CSINode is not created/taken into account. So the volume limits are not taken into account when running scheduler filters, which makes CA pack more Pods per Node than actually fit, which makes it undershoot scale-ups.
- For existing Nodes with CSI driver information already published, scheduler correctly takes the volume limits into account when scheduling.
- For new Nodes with not all CSI driver information published yet, scheduler can let Pods in that can't actually run on the Node.

After:

- By default, the scheduler CSI plugin still assumes that "no information about a CSI driver published in a CSINode" means "the node can handle unlimited amount of volumes".
- Only when explicitly opted in in CSIDriver instance, the scheduler CSI plugin assumes that "no information about a CSI driver published in a CSINode" means "the node cannot handle any volumes".
- No change for existing Nodes with CSI driver information already published - CA and scheduler still behave correctly.
- Scheduler waits until all relevant CSI driver info is published before scheduling a Pod, removing the race condition for new Nodes.
- Cluster Autoscaler correctly simulates "upcoming" CSINodes for "upcoming" Nodes and makes correct scale-up decisions.

### Goals

- Modify cluster-autoscaler so that it is aware of CSI volume limits.
- Fix scheduler, so that it doesn't schedule pods that require given CSI volume to a node that doesn't have CSI driver installed.

### Non-Goals

- Deschedule pods that can't fit on a node because of race conditions.
- Fixing other autoscalers like Karpenter is out of scope for current proposal.

## Proposal

As part of this proposal we are proposing changes into both cluster-autoscaler and kubernetes's built-in scheduler.

1. Fix cluster-autoscaler so that it takes into account attach limits when scaling nodes from 0 in a nodegroup.
2. Fix cluster-autoscaler so that it takes into account attach limits when scaling nodegroups with existing nodes.
3. Fix kubernetes built-in scheduler so that we do not schedule pods to nodes that doesn't have CSI driver installed with admin opt-in via `CSIDriver` object.

Just to reiterate we are not going to change default scheduling policy of pods that use CSI volumes. Using the new change in scheduler, which actually
prevents pod placement to nodes without CSI driver will require explicit opt-in by Cluster admins.

The reason, we decided to make the change an explicit opt-in is because:

1. It completely decouples CAS and scheduler changes. When CAS imports the scheduler, we preserve the default behaviour and only if cluster-admin or Kubernetes distributor is sure that it is safe to do, then can enable this behaviour. See Implementation section for when it is safe to enable new behaviour in a cluster.
2. For autoscalers such as Karpenter etc, which may still not have CSI node awareness builtin, this allows cluster-admin or Kubernetes distro to make the decision of whether to block pod scheduling to nodes without driver or not.
3. This allows us to release scheduler changes sooner and completely decoupled from various autoscalers, because the new feature requires explicit opt-in by the cluster-admin.

### User Stories (Optional)

#### Story 1
- User has more than one pod that is pending because no existing node has any attach limit left.
- Cluster autoscaler evaluates existing nodegroups.
- It picks a nodegroup based on existing critireas and it accurately determines number of nodes it needs to spin up based on volumes that pending pods require.

#### Story 2
- A Kubernetes admin has one or more node where CSI driver is not installed.
- Without explicitly tainting the node or using node affinity in workloads, nodes which don't have CSI driver installed aren't used for scheduling pods that require volume.

### Notes/Constraints/Caveats (Optional)

1. To fully utilize CSI node limit awareness in cluster-autoscaler, the cloudprovider interface MUST implement `TemplateNodeInfo` interface that also returns `CSINode` object with templated nodeinfo - https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/cloudprovider/clusterapi/clusterapi_nodegroup.go#L383
2. To prevent pod placement on nodes without CSI driver, the `CSIDriver` object must have an explicit opt-in.

### Risks and Mitigations

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

1. To ensure that nodes which were recently started but do not have CSI driver installed yet are considered as upcoming nodes and hence are properly handled via scaleup operation, we propose a mechanism similar to recently introduced mechanism for DRA resources. See section - "Handling Node Readiness" for more details.

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

4. Since scaling of a nodegroup requires creation of sanitized templateNodeInfo from existing `nodeInfo` objects, we need to ensure that we are creating sanitized `CSINode` objects from real `CSINode` objects associated with existing `nodeInfo` object in nodegroup. We need to make associated changes into `node_info_utils.go` to take that into account:

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

6. We further propose creation or extension of existing `StorageInfos` interface, so that both scheduler and CAS can work with the previously created fake `CSINode` objects. Without this change, both the hinting_simulator and estimator, which triggers scheduler plugin runs will not be able to find the templated `CSINode` object we created in previous step.

Making aforementioned changes should allow us to handle scaling of nodes from 1.

### Scaling from zero

Scaling from zero should work similar to scaling from 1, but the main problem is - we do not have NodeInfo which can tell us what would be the CSI attach limit on the node which is being spun up in a NodeGroup.

We propose to enhance `TemplateNodeInfo` function to report CSI volume limits via mechanism that was implemented for DRA. As such we aren't proposing a brand new mechanism for reporting CSI volume limits but rather we are using existing mechanism available from cloudprovide's implementation of NodeInfosForGroups.

A future enhancement could incorporate https://github.com/kubernetes/autoscaler/issues/7799 when it becomes available.

## Kubernetes Scheduler change

We also propose that the new scheduler behavior is opt-in via a new field in `CSIDriver`. If given node is not reporting any installed CSI drivers and `CSIDriver` has explicitly opted in, we do not schedule pods that need CSI volumes to that node.

```golang
type CSIDriverSpec struct {
    ....
    ....
    // if set to true, it will cause scheduler to prevent pod placement
    // to nodes where no CSI driver is installed.
    //   Defaults: false
    PreventPodPlacementWithoutDriver *bool
}
```

The proposed change is small and a draft PR is available here - https://github.com/kubernetes/kubernetes/pull/130702
This will stop too many pods crowding a node, when a new node is spun up and node is not yet reporting volume limits.

Along with this, we will also enhance error reporting from scheduler when scheduling of a pod fails in `NodeVolumeLimits` plugin, due to `CSINode` related errors:


1. When driver is missing on the node, we will return `CSIDriverMissingOnNode` error.
2. When `CSINode` object itself is missing on the node, we will return `CSINodeMissing`.

We also need to ensure that `StorageInfos` interface that is shared between CAS and scheduler is extended for `CSINode` objects, so that CAS can run scheduler plugins with templated `CSINode` objects.

### Handling Node Readiness

We propose to handle node readiness in a similar way to how it was handled for DRA in - https://github.com/kubernetes/autoscaler/pull/8109 . The basic idea is, we compare using `TemplateNodeInfo`, what would be the expected CSI drivers available on the node and if node doesn't yet have those drivers installed, we consider node as not-ready.

Currently handling of `TemplateNodeInfo` has an issue that reduces its usefulness when cloudprovider has not implemented changes necessary for DRA or CSI, even when nodegroup already has one or more nodes available in it, because current implementation always defers to templated `NodeInfo` returned by the cloudprovider. While not blocking for this KEP, we will try and address this issue when implementing the necessary changes for CSI.

Alternatives:

1.We propose a similar label as GPULabel added to the node that is supposed to come up with a CSI driver. This would ensure that, nodes which are supposed to have a certain CSI driver installed aren’t considered ready - https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/core/static_autoscaler.go#L979 until CSI driver is installed there.

However, we also propose that a node will be considered ready as soon as corresponding CSI driver is being reported as installed via corresponding CSINode object.

A node which is ready  but does not have CSI driver installed within certain time limit will be considered as NotReady and removed from the cluster.

2. A more exhaustive solution to node readiness is being proposed in - https://github.com/kubernetes/enhancements/pull/5416 , we are open to the idea of using it when it becomes usable from CAS.


### When it is safe to Prevent pod placement?

Generally speaking it is safe to prevent pod placement to nodes without CSI driver in scheduler, if cluster-admin or Kubernetes distro is aware that, it is running a version of autoscaler (CAS or Karpenter), which is aware of CSI attach limits coming from the node.  For CAS this generally means, `enable-csi-node-aware-scheduling` flag should be enabled and set to true in the version of CAS that is running in the cluster.


#### What happens if cluster-admin opts-in to prevent pod scheduling but autoscaler does not have CSI attach limit awareness?

If autoscaler has updated `NodeVolumeLimits` plugin from the scheduler but has otherwise has `enable-csi-node-aware-scheduling` flag disabled in CAS (or has no `CSINode` awareness), then CAS will *not* be able to schedule any pods that use CSI volume during its simulations on new nodes. The kube-scheduler will keep rejecting simulated node because, it will not have any `CSINode` information. This will be bad and autoscaling will be more or less broken for pods that require CSI volumes.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
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

We will also update the unit tests in scheduler to handle new error conditions.

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
- k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodevolumelimits/csi.go: 14/10/2025 - 78%

##### Integration tests

None

##### e2e tests

###### Cluster AutoScaler

We are planning to add e2e tests that verify behaviour of cluster autoscaler when it scales nodes for pods that require volumes.

We will add tests that validate both scaling from 0 and scaling from 1 use cases.

###### Kube Scheduler

We will add e2e tests in k/k repo for scheduler, so as scheduler behaviour is tested for following conditions:

1. When `CSINode` is reported but driver is not installed.
2. When no `CSINode` is reported from the node at all.

Please note other conditions are already tested via - https://github.com/kubernetes/kubernetes/blob/9b9cd768a05782b6cfeef62bec7696b441d7ad93/test/e2e/storage/csimock/csi_volume_limit.go#L15


### Graduation Criteria

#### Alpha

- All of the planned code changes for alpha will be done in cluster-autoscaler and kubernetes (scheduler in particular) repository.
- We plan to implement changes in cluster-autoscaler so that it can consider volume limits when scaling cluster.
- Make changes in `kube-scheduler` so that it can stop scheduling of pods that require CSI volume if underlying CSI volume is not installed on the node, with `CSIDriver` opt-in.
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

If customers have opted-in to prevent pod placement via aforementioned `CSIDriver` change, it is not recommended to disable `enable-csi-node-aware-scheduling` flag.

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

The interaction between CAS (or other autoscalers such as Karpenter) and kube-scheduler is resolved by requiring explicit opt-in
via `CSIDriver` to prevent pod placement.

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
  - Feature gate name: `VolumeLimitScaling` (in `kube-scheduler` and `kube-apiserver`)
  - `enable-csi-node-aware-scheduling"` flag in `CAS`.
- [x] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
    Yes, it should require restart of CAS and kube-scheduler.

###### Does enabling the feature change any default behavior?

No, the scheduler and autoscaler defaul behavior is the same. A CSI driver must opt-in via its CSIDriver instance to get the new behavior.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

For CAS:
- Yes. This will simply cause old behaviour to be restored, but `PreventPodPlacementWithoutDriver` should be disabled (if enabled manually) in `CSIDriver` object before disabling this feature in CAS.

For kube-scheduler:
- The feature gate in kube-scheduler can be disabled without problems.

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

In v1.36 we are making chages into `CSIDriver` object by adding the field `PreventPodPlacementWithoutDriver`.

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
