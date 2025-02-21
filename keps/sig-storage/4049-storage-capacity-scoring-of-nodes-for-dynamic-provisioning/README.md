<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

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
# KEP-4049: Storage Capacity Scoring of Nodes for Dynamic Provisioning

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
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Modify stateData to be able to store StorageCapacity](#modify-statedata-to-be-able-to-store-storagecapacity)
  - [Get the capacity of nodes for dynamic provisioning](#get-the-capacity-of-nodes-for-dynamic-provisioning)
  - [Scoring of nodes for dynamic provisioning](#scoring-of-nodes-for-dynamic-provisioning)
  - [Conditions for scoring static or dynamic provisioning](#conditions-for-scoring-static-or-dynamic-provisioning)
  - [Feature Gate Consolidation](#feature-gate-consolidation)
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
  - [Weighting Static Provisioning Scores and Dynamic Provisioning Scores](#weighting-static-provisioning-scores-and-dynamic-provisioning-scores)
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

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [X] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [X] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [X] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
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

This KEP proposes adding a way to score nodes for dynamic provisioning of PVs. This scoring method is based on storage capacity in the VolumeBinding plugin. 
By considering the amount of free space that nodes have, it is possible to dynamically schedule pods on the node that has the most or least free space.

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

## Motivation

Storage capacity needs to be considered when:

- we want to resize after a node-local PV is scheduled. In this case we need to select a node with as much free space as possible. 
- we want to select a node with less free node space to reduce the number of nodes as much as possible.

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### Goals

- To modify the scoring logic to count on dynamic provisioning in addition to the current, considering only static provisioning.

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

### Non-Goals

- To change how to score nodes for static provisioning.

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

## Proposal

- Node scores based on available space can be taken into account when performing dynamic provisioning.

Cluster admin can configure the scoring logic using a new field in [`VolumeBindingArgs`](https://github.com/kubernetes/kubernetes/blob/1bb62cd27506f86d4b3f71a61a78e892aa2dbca1/pkg/scheduler/apis/config/types_pluginargs.go#L146-L169) of `kubescheduler.config.k8s.io`. The scoring logic is global for the whole cluster and we propose two values:

- Prefer a node with the least allocatable.
- Prefer a node with the maximum allocatable.

Considering the common scenario of local storage, we want to leave room for volume expansion after node allocation. The default setting is to prefer a node with the maximum allocatable.

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

We want to leave room for volume expansion after node allocation. In this case, we want to allocate the node that has the maximum amount of free space. 

#### Story 2

We want to reduce the number of nodes as much as possible to reduce costs when using a cloud environment. In this case, we want to allocate the node that has the smallest amount of sufficiently free space left.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

| Risk                                                                             | Impact | Mitigation                                         |
| -------------------------------------------------------------------------------- | ------ | -------------------------------------------------- |
| Misconfiguration of storage capacity scoring parameters                          | Medium | Provide documentation                              |
| Potential performance overhead due to additional scoring calculations            | Low    | Optimize scoring algorithms                        |
| Loss of optimized scheduling after downgrading to a version without this feature | Medium | Explain the impact of downgrading in documentation |

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

We modify the existing VolumeBinding plugin to achieve scoring of nodes for dynamic provisioning.

### Modify stateData to be able to store StorageCapacity

We modify the struct called `PodVolumes` contained in `stateData` to score nodes for dynamic provisioning.

The struct of `stateData` is as follows:

```go
type stateData struct {
	...
	// podVolumesByNode holds the pod's volume information found in the Filter
	// phase for each node
	// it's initialized in the PreFilter phase
	podVolumesByNode map[string]*PodVolumes
	...
}
```

By making the following changes to `PodVolumes`, `CSIStorageCapacity` can be stored.

```diff
+ type DynamicProvision struct {
+ 	PVC      *v1.PersistentVolumeClaim
+ 	Capacity *storagev1.CSIStorageCapacity
+ }

type PodVolumes struct {
	StaticBindings []*BindingInfo
-   DynamicProvisions []*v1.PersistentVolumeClaim
+ 	DynamicProvisions []*DynamicProvision
}
```

### Get the capacity of nodes for dynamic provisioning

Add `CSIStorageCapacity` to the return value of the `volumeBinder.hasEnoughCapacity` method. This returns the `DynamicProvision.Capacity` field in the case of dynamic provisioning.

```diff
- func (b *volumeBinder) hasEnoughCapacity(provisioner string, claim *v1.PersistentVolumeClaim, storageClass *storagev1.StorageClass, node *v1.Node) (bool, error) {
+ func (b *volumeBinder) hasEnoughCapacity(provisioner string, claim *v1.PersistentVolumeClaim, storageClass *storagev1.StorageClass, node *v1.Node) (bool, *storagev1.CSIStorageCapacity, error) {
	quantity, ok := claim.Spec.Resources.Requests[v1.ResourceStorage]
	if !ok {
		// No capacity to check for.
- 		return true, nil
+ 		return true, nil, nil
	}

	// Only enabled for CSI drivers which opt into it.
	driver, err := b.csiDriverLister.Get(provisioner)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Either the provisioner is not a CSI driver or the driver does not
			// opt into storage capacity scheduling. Either way, skip
			// capacity checking.
- 			return true, nil
+ 			return true, nil, nil
		}
- 		return false, err
+ 		return false, nil, err
	}
	if driver.Spec.StorageCapacity == nil || !*driver.Spec.StorageCapacity {
- 		return true, nil
+ 		return true, nil, nil
	}

	// Look for a matching CSIStorageCapacity object(s).
	// TODO (for beta): benchmark this and potentially introduce some kind of lookup structure (https://github.com/kubernetes/enhancements/issues/1698#issuecomment-654356718).
	capacities, err := b.csiStorageCapacityLister.List(labels.Everything())
	if err != nil {
- 		return false, err
+ 		return false, nil, err
	}

  sizeInBytes := quantity.Value()
	for _, capacity := range capacities {
		if capacity.StorageClassName == storageClass.Name &&
			capacitySufficient(capacity, sizeInBytes) &&
			b.nodeHasAccess(node, capacity) {
			// Enough capacity found.
- 			return true, nil
+ 			return true, capacity, nil
		}
	}

	// TODO (?): this doesn't give any information about which pools where considered and why
	// they had to be rejected. Log that above? But that might be a lot of log output...
	klog.V(4).InfoS("Node has no accessible CSIStorageCapacity with enough capacity for PVC",
		"node", klog.KObj(node), "PVC", klog.KObj(claim), "size", sizeInBytes, "storageClass", klog.KObj(storageClass))
- 	return false, nil
+ 	return false, nil, nil
}
```

### Scoring of nodes for dynamic provisioning

The `Score` method in the current VolumeBinding plug-in scores nodes considering only static provisioning. The scoring applies to every entry in `podVolumes.StaticBindings`.

In this KEP, add the scoring of nodes for dynamic provisioning in the `Score` method of the VolumeBinding plugin. The scoring applies to every entry in `podVolumes.DynamicProvisions` where `Capacity` is not equal to `nil`.

Scoring for dynamic provisioning is executed if there are no `StaticBindings`. In other words, if there is only static provisioning or both static and dynamic provisioning, the scoring will be done as usual for static provisioning. Then, if there is only dynamic provisioning, the following will be set to `classResources` and passed to the `scorer` function:

- `Requested: provision.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)]`
- `Capacity: CSIStorageCapacity`

By doing this, we can calculate scores to nodes for dynamic provisioning in a way that is based on the `Shape` setting of `VolumeBindingArgs`, and which takes into account the amount of free space the nodes have.

```diff
// Score invoked at the score extension point.
func (pl *VolumeBinding) Score(ctx context.Context, cs *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	if pl.scorer == nil {
		return 0, nil
	}
	state, err := getStateData(cs)
	if err != nil {
		return 0, framework.AsStatus(err)
	}
	podVolumes, ok := state.podVolumesByNode[nodeName]
        if !ok {
		return 0, nil
	}
-       // group by storage class
+
        classResources := make(classResourceMap)
-       for _, staticBinding := range podVolumes.StaticBindings {
-               class := staticBinding.StorageClassName()
-               storageResource := staticBinding.StorageResource()
-               if _, ok := classResources[class]; !ok {
-                       classResources[class] = &StorageResource{
-                               Requested: 0,
-                               Capacity:  0,
+       if len(podVolumes.StaticBindings) != 0 {
+               // group static biding volumes by storage class
+               for _, staticBinding := range podVolumes.StaticBindings {
+                       class := staticBinding.StorageClassName()
+                       storageResource := staticBinding.StorageResource()
+                       if _, ok := classResources[class]; !ok {
+                               classResources[class] = &StorageResource{
+                                       Requested: 0,
+                                       Capacity:  0,
+                               }
+                       }
+                       classResources[class].Requested += storageResource.Requested
+                       classResources[class].Capacity += storageResource.Capacity
+               }
+       } else {
+               // group dynamic biding volumes by storage class
+               for _, provision := range podVolumes.DynamicProvisions {
+                       if provision.Capacity == nil {
+                               continue
+                       }
+                       class := *provision.PVC.Spec.StorageClassName
+                       if _, ok := classResources[class]; !ok {
+                               classResources[class] = &StorageResource{
+                                       Requested: 0,
+                                       Capacity:  0,
+                               }
                        }
+                       requestedQty := provision.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)]
+                       classResources[class].Requested += requestedQty.Value()
+                       classResources[class].Capacity += provision.Capacity.Capacity.Value()
                }
-               classResources[class].Requested += storageResource.Requested
-               classResources[class].Capacity += storageResource.Capacity
        }
+
        return pl.scorer(classResources), nil
}
```

Users can select the scoring logic from the following options in `VolumeBindingArgs`. The scoring logic is the same among all Pod + PVC(s).

- (a) Prefer a node with the least allocatable.
- (b) Prefer a node with the maximum allocatable.

Considering the common scenario of local storage, we want to leave room for volume expansion after node allocation. The default setting is to prefer a node with the maximum allocatable.

### Conditions for scoring static or dynamic provisioning

About the `Score` function, the score will be calculated with the existing way (only static provisioning is taken into account) if at least one PVC was statically provisioned. Otherwise, the score will be calculated from dynamic provisioning.

Implementation idea:

```diff
func (pl *VolumeBinding) Score(ctx context.Context, cs *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	...

+ 	if len(static) != 0 {
+ 		return static_score, nil;	// Same value as the current method
+ 	} else {
+ 		return dynamic_score, nil;	// Propose in this KEP
+ 	}
- 	return pl.scorer(classResources), nil
}
```

### Feature Gate Consolidation

The `StorageCapacityScoring` feature gate will now control the functionality previously managed by the `VolumeCapacityPriority` feature gate, which will be deprecated. This consolidation focuses on enabling node scoring based on storage capacity, limited to the behaviors necessary for `StorageCapacityScoring`. Specifically, [the utilization shape points](https://github.com/kubernetes/enhancements/tree/49cff2e7c62800d1c87dbf5fac02c506209d1409/keps/sig-storage/1845-prioritization-on-volume-capacity#configuring-the-utilization-shape-points) have been supported because they are required for `StorageCapacityScoring`. However, [the weight of storage class](https://github.com/kubernetes/enhancements/tree/49cff2e7c62800d1c87dbf5fac02c506209d1409/keps/sig-storage/1845-prioritization-on-volume-capacity#configuring-the-weight-of-storage-class) has not been implemented ([ref1](https://github.com/kubernetes/enhancements/blob/10ff969c18772dc82cda02f7e9140c4840413e5d/keps/sig-storage/1845-prioritization-on-volume-capacity/README.md?plain=1#L201-L203), [ref2](https://github.com/kubernetes/kubernetes/pull/96347/files#diff-c4f8e3891e057e1f662eda397ce4f97d29de7d779d8a886ba2ec5a6df6f9d989R44)), and there are no plans to require it for `StorageCapacityScoring`, so it will not be implemented. For more details on the original proposal, see [KEP-1845](https://github.com/kubernetes/enhancements/blob/master/keps/sig-storage/1845-prioritization-on-volume-capacity/README.md).

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

Nothing in particular.

##### Unit tests

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

The following unit tests are planned:

- Are the scores assigned to nodes for dynamic provisioning appropriate for the amount of free space?
- Are the amount of free space score of nodes for dynamic provisioning and the Static Bindings score both functional?

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

The scoring function will be tested in test/integration/volumescheduling/storage_capacity_scoring_test.go.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

The following e2e tests are planned:

- When only static provisioning is available, or a mixture of static provisioning and dynamic provisioning is available:
  - Does it pass traditional tests?
- When only dynamic provisioning is available:
  - Is the Pod placed on the node with the largest available space by default?
  - When `VolumeBindingArgs` is set to "Prefer a node with the maximum allocatable", is the Pod placed on the node with the largest available space?
  - When `VolumeBindingArgs` is set to "Prefer a node with the least allocatable", is the Pod placed on the node that meets the requested size but has the smallest available space?
  - Does the Pod placement fail if no node meets the requested size?
  - Even when the Pod is recreated, is the placement in the node performed as expected above?

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

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

#### Alpha

- Add `StorageCapacityScoring` feature gate
- E2e tests completed

#### Beta

- One release with positive feedback from users

#### GA

- No users complaining about the new behavior

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

1. Upgrading the cluster to support storage capacity scoring for dynamic provisioning:
   - After the upgrade, the scheduler will be able to score nodes based on their storage capacity for dynamic provisioning. This will involve additional checks and calculations to ensure that nodes with sufficient capacity are prioritized.
   - Existing configurations and API usage will remain compatible, but administrators may need to review and adjust their storage class configurations to fully leverage the new scoring mechanism.

2. Downgrading the cluster to a version without storage capacity scoring for dynamic provisioning:
   - If the cluster is downgraded, the scheduler will revert to the previous behavior where storage capacity scoring for dynamic provisioning is not considered.
   - Any Pods created after the upgrade will still exist, but their scheduling will no longer take storage capacity into account, potentially leading to less optimal placement.
   - No additional changes to invocations or configurations are required, but administrators should be aware that the enhanced scheduling capabilities will be lost.

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

Nothing in particular.

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

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: StorageCapacityScoring
  - Components depending on the feature gate: kube-scheduler

###### Does enabling the feature change any default behavior?

The scheduling behavior is changed if this function is enabled.

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, this feature can be disabled after it has been enabled by setting the feature gate to false again. In doing so, the scoring for VolumeBinding will revert to the current method. This change won't affect the behavior of existing Pods.

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

Re-enabling the feature from a rolled-back state will result in scheduling that considers dynamic provisioning. There will be no impact on existing running Pods.

###### Are there any tests for feature enablement/disablement?

Yes. We will add unit tests with and without the feature gate enabled.

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

Turning the feature gate flag on/off only changes scheduling scoring. So there is no possibility of impacting workloads that are already running.

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

A spike on metric `schedule_attempts_total{result="error|unschedulable"}` when this feature gate is enabled.

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not applicable, yet.

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No, it isn't.

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

If enabled, this feature applies to all workloads which uses delay binding PVCs. Also non-zero value of metric `plugin_execution_duration_seconds{plugin="VolumeBinding",extension_point="Score"}` is a sign indicating this feature is in use.
Unfortunately, there is no way to distinguish whether only static provisioning is being considered (the current behavior) or both static and dynamic provisioning are being considered (the new behavior).

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

Pods that use only dynamically provisioned PVCs will be scheduled to nodes with more available capacity.

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

It may affect the time taken by scheduling. Clarify it during the beta phase.

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

Clarify this during the beta phase.

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name: `plugin_execution_duration_seconds{plugin="VolumeBinding",extension_point="Score"}`
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Nothing in particular.

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

No.

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

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

No.

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

No.

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Yes, it may affect the time taken by scheduling.

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

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

No, this feature will not exhaust node resources such as PIDs, sockets, or inodes.

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

The behavior in such cases does not change. This proposal only modifies one of the plugins in the kube-scheduler.

###### What are other known failure modes?

Not applicable, yet.

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

Check the kube-scheduler logs.

## Implementation History

- 2023-05-30 Initial KEP sent out for review

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

- The implementation of storage capacity scoring for dynamic provisioning may introduce complexity in the scheduling process. This could potentially lead to increased scheduling latency as the scheduler performs additional checks and calculations.

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

### Weighting Static Provisioning Scores and Dynamic Provisioning Scores

The scoring function will return the sum of the static score and the dynamic score, each multiplied by their respective weights. The weights are determined by the ratio of static and dynamic capacities.

Implementation idea for the `Score` function:

```go
func (pl *VolumeBinding) Score(...) (int64, *framework.Status) {
  ...
  return (static_weight) * static_score + (1-static_weight) * dynamic_score;
}
```

Ultimately, the current design was chosen. The reasons are as follows:

- Conflict issue: In this approach, there is a possibility that the static provisioning and dynamic provisioning scores could cancel each other out, leading to inaccurate scoring.
- Feasibility of implementation: The current design was deemed more feasible and clearer in terms of implementation.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
