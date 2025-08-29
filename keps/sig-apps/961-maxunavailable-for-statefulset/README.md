# KEP-961: Implement maxUnavailable in StatefulSet

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
- [Table of Contents](#table-of-contents)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Implementation Details](#implementation-details)
    - [API Changes](#api-changes)
    - [Implementation](#implementation)
    - [Metrics](#metrics)
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
    - [Upgrade](#upgrade)
    - [Downgrade](#downgrade)
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

Items marked with (R) are required _prior to targeting to a milestone / release_.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [x] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
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

## Table of Contents

<!-- toc -->

- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
  - [Implementation Details](#implementation-details)
    - [API Changes](#api-changes)
    - [Implementation](#implementation)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Upgrades/Downgrades](#upgradesdowngrades)
  - [Tests](#tests)
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
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

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

The purpose of this enhancement is to implement maxUnavailable for StatefulSet during RollingUpdate.
When a StatefulSet’s `.spec.updateStrategy.type` is set to `RollingUpdate`, the StatefulSet controller
will delete and recreate each Pod in the StatefulSet. The updating of each Pod currently happens one at a time.
With support for `maxUnavailable`, the updating will proceed `maxUnavailable` number of pods at a time.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Consider the following scenarios:

1. My containers publish metrics to a time series system. If I am using a Deployment, each rolling
   update creates a new pod name and hence the metrics published by this new pod starts a new time series
   which makes tracking metrics for the application difficult. While this could be mitigated, it requires
   some tricks on the time series collection side. It would be so much better, If we could use a
   StatefulSet object so my object names doesnt change and hence all metrics goes to a single time series. This will be easier if StatefulSet is at feature parity with Deployments.
2. My Container does some initial startup tasks like loading up cache or something that takes a lot of
   time. If we used StatefulSet, we can only go one pod at a time which would result in a slow rolling
   update. If StatefulSet supported maxUnavailable with value greater than 1, it would allow for a faster
   rollout since a total of maxUnavailable number of pods could be loading up the cache at the same time.
3. My Stateful clustered application, has followers and leaders, with followers being many more than 1. My application can tolerate many followers going down at the same time. I want to be able to do faster
   rollouts by bringing down 2 or more followers at the same time. This is only possible if StatefulSet
   supports maxUnavailable in Rolling Updates.
4. Sometimes I just want easier tracking of revisions of a rolling update. Deployment does it through
   ReplicaSets and has its own nuances. Understanding that requires diving into the complicacy of hashing
   and how ReplicaSets are named. Over and above that, there are some issues with hash collisions which
   further complicate the situation(I know they were solved). StatefulSet introduced ControllerRevisions
   in 1.7 which are much easier to think and reason about. They are used by DaemonSet and StatefulSet for
   tracking revisions. It would be so much nicer if all the use cases of Deployments can be met in
   StatefulSet's and additionally we could track the revisions by ControllerRevisions. Another way of
   saying this is, all my Deployment use cases are easily met by StatefulSet, and additionally I can enjoy
   easier revision tracking only if StatefulSet supported `maxUnavailable`.

With this feature in place, when using StatefulSet with maxUnavailable >1, the user is making a
conscious choice that more than one pod going down at the same time during rolling update, would not
cause issues with their Stateful Application which have per pod state and identity. Other Stateful
Applications which cannot tolerate more than one pod going down, will resort to the current behavior of one pod at a time Rolling Updates.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

StatefulSet RollingUpdate strategy will contain an additional parameter called `maxUnavailable` to
control how many Pods will be brought down at a time, during Rolling Update.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

None.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

This document proposes to add new field to StatefulSet's RollingUpdate configuration 
settings (`.spec.updateStrategy.rollingUpdate`) called `maxUnavailable`, which will 
allow setting the maximum number of pods that can be unavailable during the update.
This new field can be an absolute, positive number (eg. `5`) or a percentage of desired 
pods (ex: `10%`). When the field is not specified, it will default to 1 to maintain 
previous behavior of rolling one pod at a time.

### User Stories

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

As a User of Kubernetes, I should be able to update my StatefulSet, more than one Pod at a time, in a
RollingUpdate manner, if my Stateful app can tolerate more than one pod being down, thus allowing my
update to finish much faster.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

With `maxUnavailable` field set, we'll bring down more than one pod at a time, if your application can't tolerate
this behavior, you should absolutely leave the field unset or use 1, which will match default behavior.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

We are proposing a new field called `maxUnavailable` whose default value will be 1. In this mode, StatefulSet will behave exactly like its current behavior.
It's possible we introduce a bug in the implementation. The mitigation currently is that this feature is disabled by default in Alpha phase for people to try out and give
feedback.
In Beta phase when its enabled by default, users will only see issues or bugs when `maxUnavailable` is set to something greater than 1. Since people have
tried this feature in Alpha, we would have time to fix issues.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Implementation Details

#### API Changes

Following changes will be made to the Rolling Update Strategy for StatefulSet.

```go
// RollingUpdateStatefulSetStrategy is used to communicate parameter for RollingUpdateStatefulSetStrategyType.
type RollingUpdateStatefulSetStrategy struct {
	// THIS IS AN EXISTING FIELD
        // Partition indicates the ordinal at which the StatefulSet should be
        // partitioned.
        // Default value is 0.
        // +optional
        Partition *int32

	// NOTE THIS IS THE NEW FIELD BEING PROPOSED
	// The maximum number of pods that can be unavailable during the update.
        // Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%).
        // Absolute number is calculated from percentage by rounding down.
        // Defaults to 1.
        // +optional
        MaxUnavailable *intstr.IntOrString

	...
}
```

- By Default, if maxUnavailable is not specified, its value will be assumed to be 1 and StatefulSets
  will follow their old behavior. This will also help while upgrading from a release which doesn't support maxUnavailable to a release which supports this field.
- If maxUnavailable is specified, it cannot be greater than total number of replicas.
- If maxUnavailable is specified and partition is also specified, MaxUnavailable cannot be greater than `replicas-partition`
- If a partition is specified, maxUnavailable will only apply to all the pods which are staged by the
  partition. Which means all Pods with an ordinal that is greater than or equal to the partition will be
  updated when the StatefulSet’s .spec.template is updated. Let's say total replicas is 5 and partition is set to 2 and maxUnavailable is set to 2. If the image is changed in this scenario, following
  are the possible behavior choices we have:

  1.  Pods with ordinal 4 and 3 will start Terminating at the same time (because of maxUnavailable). Once they are both running and available, pods with ordinal 2 will start Terminating. Pods with ordinal 0 and 1
      will remain untouched due the partition. In this choice, the number of pods terminating is not always
      maxUnavailable, but sometimes less than that. For e.g. if pod with ordinal 3 is running and available but 4 is not, we still wait for 4 to be running and available before moving on to 2. This implementation avoids
      out of order Terminations of pods.
  2.  Pods with ordinal 4 and 3 will start Terminating at the same time (because of maxUnavailable). When any of 4 or 3 are running and available, pods with ordinal 2 will start Terminating. This could violate
      ordering guarantees, since if 3 is running and available, then both 4 and 2 are terminating at the same
      time out of order. If 4 is running and available, then both 3 and 2 are Terminating at the same time and no ordering guarantees are violated. This implementation guarantees, that always there are maxUnavailable number of Pods Terminating except the last batch.
  3.  Pod with ordinal 4 and 3 will start Terminating at the same time (because of maxUnavailable). When 4 is running and available, 2 will start Terminating. At this time both 2 and 3 are terminating. If 3 is
      running and available before 4, 2 won't start Terminating to preserve ordering semantics. So at this time,
      only 1 is unavailable although we requested 2.
  4.  (Implemented approach) Introduce a field in Rolling Update, which decides whether we want maxUnavailable with ordering or without ordering guarantees. Depending on what the user wants, this Choice can either choose behavior 1 or 3 if ordering guarantees are needed or choose behavior 2 if they don't care. To simplify this further
      PodManagementPolicy today supports `OrderedReady` or `Parallel`. The `Parallel` mode only supports scale up and tear down of StatefulSets and currently doesn't apply to Rolling Updates. So instead of coming up
      with a new field, we could use the PodManagementPolicy to choose the behavior the User wants.

              1. PMP=Parallel will now apply to RollingUpdate. This will choose behavior described in 2 above.
                 This means always maxUnavailable number of Pods are terminating at the same time except in
                 the last case and no ordering guarantees are provided.
              2. PMP=OrderedReady with maxUnavailable can choose one of behavior 1 or 3.

NOTE: The goal is faster updates of an application. In some cases , people would need both ordering
and faster updates. In other cases they just need faster updates and they dont care about ordering as
long as they get identity.

Choice 1 is simpler to reason about. It does not always have maxUnavailable number of Pods in
Terminating state. It does not guarantee ordering within the batch of maxUnavailable Pods. The maximum
difference in the ordinals which are Terminating out of Order, cannot be more than maxUnavailable.

Choice 2 always offers maxUnavailable number of Pods in Terminating state. This can sometime lead to
pods terminating out of order. This will always lead to the fastest rollouts. The maximum difference in the ordinals which are Terminating out of Order, can be more than maxUnavailable.

Choice 3 always guarantees than no two pods are ever Terminating out of order. It sometimes does that,
at the cost of not being able to Terminate maxUnavailable pods. The implementation for this might be
complicated.

Choice 4 provides a choice to the users and hence takes the guessing out of the picture on what they
will expect. Implementing Choice 4 using PMP would be the easiest.

#### Implementation

The alpha release we are going with Choice 4 with support for both PMP=Parallel and PMP=OrderedReady.
For PMP=Parallel, we will use Choice 2.
For PMP=OrderedReady, the plan for alpha was to go with Choice 3 to ensure we can support ordering guarantees while also
making sure the rolling updates are fast, but for simplicity Choice 1 was implemented instead.

We are keeping implementation the same for beta release, going the simpler route, but instead of checking for healthy pods in the part of the code that decides what is a `unavailablePod`, we check for `isUnavailable`, so that `minReadySeconds` is respected.

Alpha implementation:

https://github.com/kubernetes/kubernetes/blob/v1.33.0-beta.0/pkg/controller/statefulset/stateful_set_control.go#L718

```go
...
	 // we compute the minimum ordinal of the target sequence for a destructive update based on the strategy.
        updateMin := 0
	maxUnavailable := 1
        if set.Spec.UpdateStrategy.RollingUpdate != nil {
                updateMin = int(*set.Spec.UpdateStrategy.RollingUpdate.Partition)

		// NEW CODE HERE
		maxUnavailable, err = intstrutil.GetValueFromIntOrPercent(intstrutil.ValueOrDefault(set.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable, intstrutil.FromInt(1)), int(replicaCount), false)
		if err != nil {
			return &status, err
		}
	}

	var unavailablePods []string
	// we terminate the Pod with the largest ordinal that does not match the update revision.
	for target := len(replicas) - 1; target >= updateMin; target-- {

		// delete the Pod if it is not already terminating and does not match the update revision.
		if getPodRevision(replicas[target]) != updateRevision.Name && !isTerminating(replicas[target]) {
			klog.V(2).Infof("StatefulSet %s/%s terminating Pod %s for update",
				set.Namespace,
				set.Name,
				replicas[target].Name)
			if err := ssc.podControl.DeleteStatefulPod(set, replicas[target]); err != nil {
				return &status, err
			}

			// After deleting a Pod, dont Return From Here Yet.
			// We might have maxUnavailable greater than 1
			status.CurrentReplicas--
		}

		// wait for unhealthy Pods on update
		if !isHealthy(replicas[target]) {
			// If this Pod is unhealthy regardless of revision, count it in
			// unavailable pods
			unavailablePods = append(unavailablePods, replicas[target].Name)
			klog.V(4).Infof(
				"StatefulSet %s/%s is waiting for Pod %s to update",
				set.Namespace,
				set.Name,
				replicas[target].Name)
		}

		// NEW CODE HERE
		// If at anytime, total number of unavailable Pods exceeds maxUnavailable,
		// we stop deleting more Pods for Update
		if len(unavailablePods) >= maxUnavailable {
			klog.V(4).Infof(
				"StatefulSet %s/%s is waiting for unavailable Pods %v to update, max Allowed to Update Simultaneously %v",
				set.Namespace,
				set.Name,
				unavailablePods,
				maxUnavilable)
			return &status, nil
		}

	}
...
```

New proposed implementation: https://github.com/kubernetes/kubernetes/pull/130909

#### Metrics

We'll add two new metrics:
- **statefulset_max_unavailable**: tracks the current `.spec.updateStrategy.rollingUpdate.maxUnavailable` value. This gauge reflects the configured maximum number of pods that can be unavailable during rolling updates, providing visibility into the availability constraints.
- **statefulset_unavailable_replicas**: tracks the current number of unavailable pods in a StatefulSet. This gauge reflects the real-time count of pods that are either missing or unavailable (i.e., not ready for `.spec.minReadySeconds`).

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

- [x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

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

Testcases:

- maxUnavailable =1, Same behavior as today with PodManagementPolicy as `OrderedReady` or `Parallel`
- Each of these Tests can be run in PodManagementPolicy = `OrderedReady` or `Parallel` and the Update
  should happen at most maxUnavailable Pods at a time in ordered or parallel fashion respectively.
- maxUnavailable greater than 1 without partition
- maxUnavailable greater than replicas without partition
- maxUnavailable greater than 1 with partition and staged pods less then maxUnavailable
- maxUnavailable greater than 1 with partition and staged pods same as maxUnavailable
- maxUnavailable greater than 1 with partition and staged pods greater than maxUnavailable
- maxUnavailable greater than 1 with partition and maxUnavailable greater than replicas

New testcases being added:

- Feature enablement/disablement test

Coverage:

- `pkg/apis/apps/v1`: `2025-06-12` - `31.1%`
- `pkg/apis/apps/v1beta2`: `2025-06-12` - `31.1%`
- `pkg/apis/apps/validation`: `2025-06-12` - `92.5%`
- `pkg/controller/statefulset`: `2025-06-12` - `86.5%`
- `pkg/registry/apps/statefulset`: `2025-06-12` - `81.4%`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- `test/integration/statefulset`: we'll add a test to verify that the number of pods brought down each
  time is less or equal to configured maxUnavailable and podManagementPolicy settings are honored.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- `test/e2e/apps/statefulset.go`: 
  - test that rolling updates are working correctly for both PodManagementPolicy types when the MaxUnavailable is used. 
  - include a test that fails currently but passes when https://github.com/kubernetes/kubernetes/issues/112307 is fixed, with a 
    StatefulSet setting `minReadySeconds` and `updateStrategy.rollingUpdate.maxUnavailable` and checking for a correct rollout specially when scaling down during a rollout.
  - https://github.com/kubernetes/kubernetes/pull/133717

## Graduation Criteria

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

In general, we try to use the same stages (alpha, beta, GA), regardless of how the
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

- Initial support for maxUnavailable in StatefulSets added. Disabled by default with default value of 1.

#### Beta

- Enabled by default with default value of 1 with upgrade/downgrade tested at least manually.
- Added `statefulset_max_unavailable` and `statefulset_unavailable_replicas` metrics to in-tree.
- It is necessary to update the `firstUnhealthyPod` calculation to correctly call processCondemned. New tests should cover this and take into consideration that the controller should first wait for the predecessor condemned pods to become available before deleting them and delete the pod with the highest ordinal number
- minReadySeconds and maxUnavailable bugs https://github.com/kubernetes/kubernetes/issues/123911, https://github.com/kubernetes/kubernetes/issues/112307, https://github.com/kubernetes/kubernetes/issues/119234 and https://github.com/kubernetes/kubernetes/issues/123918 should be fixed before promotion of maxUnavailable.
- Additional unit/e2e/integration tests listed in the test plan should be added covering the newly found bugs.
- Users should be warned that maxUnavailable works differently for each podManagementPolicy (e.g. for `OrderedReady` it is not applied until the StatefulSet had a chance to fully scale up). This can result in slower rollouts. For parallel this can skip ordering. This should be both mentioned in the API doc and website as a requirements for beta graduation.

#### GA

- No critical bugs reported and no bad feedbacks collected.

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

#### Upgrade

- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
  - This is a new field, it will only be enabled after an upgrade. To maintain the
    previous behavior, you can disable the feature gate manually in Beta or leave
    the field unconfigured.
  - This field was set in alpha before upgrading:
    - If the flag was enabled, it continue to be enabled in Beta
    - If the flag was disabled, it will continue to be disbled in Beta
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
  - This feature is enabled by default in Beta so you can configure the maxUnavailable
    to meet your requirements

#### Downgrade

- If we try to set this field, it will be silently dropped.
- If the object with this field already exists, we maintain is as it is, but the controller
  will just ignore this field.
- downgrade of kube-apiserver to a version with the feature flag was disabled, when the field was previously set - the field gets ignored by apiserver, and controller behaves as if the field is not set
- downgrade of kube-controller-manager to version with the feature flag disabled, when the field was previously set - the field gets ignored by controller, and controller behaves as if the field is not set
- downgrade of either kube-apiserver or kube-controller-manager to a version with the feature flag was disabled, when the field was not set the behavior doesn't change

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

- If both api-server and kube-controller-manager have a version where the feature is enabled, then `maxUnavailable` will take effect.
- If only api-server has a version where the feature is enabled, because statefulset controller is unaware of this feature, then we'll
  fall into the default rolling-update behavior, one pod at a time.
- If only kube-controller-manager has a version where the feature is enabled, then this field is invisible to users, we'll keep the default
  rolling-update behavior, one pod at a time.
- If both api-server and kube-controller-manager have a version where the feature is enabled, then only one Pod will be updated
  at a time as the default behavior.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: MaxUnavailableStatefulSet
  - Components depending on the feature gate: kube-apiserver and kube-controller-manager

###### Does enabling the feature change any default behavior?

No, the default behavior remains the same.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, you can disable the feature-gate manually once it's in Beta, this will affect StatefulSets that are in the middle of a rollout when the feature gets disabled.
If the feature is disabled in the middle of the rollout, the rollout will continue as if maxUnavailable is 1. If there is more than one pod 
undergoing an update, it will wait for them to complete their update, and continue as if maxUnavailable is 1.

###### What happens if we reenable the feature if it was previously rolled back?

If we disable the feature-gate and reenable it again, then the `maxUnavailable` will take effect again. [Even if this is a new field, it only gets cleared if it was not set before (if it was nil). So if it is set it continues to exist and is enforced.](https://github.com/kubernetes/kubernetes/blob/bb67eb5bd237334d6d0343aa382b351002653404/pkg/registry/apps/statefulset/strategy.go#L120-L137) This will affect StatefulSets that are in the middle of a rollout when the feature gets re-enabled.

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

There are unit tests which make sure the field is correctly dropped
on feature enable and disabled, see [strategy tests](https://github.com/kubernetes/kubernetes/blob/23698d3e9f4f3b9738ba5a6fcefd17894a00624f/pkg/registry/apps/statefulset/strategy_test.go#L391-L417).

Feature enablement/disablement test will also be added when graduating to Beta as [TestStatefulSetStartOrdinalEnablement](https://github.com/kubernetes/kubernetes/blob/23698d3e9f4f3b9738ba5a6fcefd17894a00624f/pkg/registry/apps/statefulset/strategy_test.go#L473)

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will roll out across nodes.
-->

The rollout or rollback of the `maxUnavailable` feature for StatefulSets primarily affects how updates are managed, aiming to minimize disruptions. However, several scenarios could lead to potential issues:

Incorrect `maxUnavailable` Configurations: Setting `maxUnavailable` too high or too low could lead to service disruptions or slow update processes, respectively.

Feature Flag Consistency Across Control Plane: Inconsistencies in applying the feature flag MaxUnavailableStatefulSet across the control plane components, especially in HA clusters, can lead to unpredictable behaviors. The impact varies based on the combination of the feature flag state (on/off) for kube-apiserver and kube-controller-manager:

1. Feature Enabled on kube-apiserver and Disabled on kube-controller-manager:

  - The maxUnavailable setting is accepted by the API server but ignored by the controller. This could lead to slower rollouts than expected because the controller doesn't act on the provided `maxUnavailable` values.

2. Feature Disabled on kube-apiserver and Enabled on kube-controller-manager:

  - Attempts to use `maxUnavailable` in StatefulSet specifications will be rejected by the API server. Users won't be able to leverage the feature even though the controller is capable of interpreting it.

3. Feature Enabled on Both kube-apiserver and kube-controller-manager:

  - This is the ideal scenario where the feature works as intended, allowing users to specify `maxUnavailable` for controlled updates.

4. Feature Disabled on Both kube-apiserver and kube-controller-manager:

  - StatefulSet updates revert to the default Kubernetes behavior, ignoring any `maxUnavailable` settings even if previously specified.

In scenarios 1 and 2, running workloads could be impacted due to the mismatch in feature flag settings. For instance, expected rolling update strategies might not be applied, potentially affecting application availability or update speed.

To mitigate these risks, it's crucial to ensure consistent feature flag settings across all control plane components during rollout and to be prepared to adjust configurations based on observed behavior. Monitoring the progress and outcomes of StatefulSet updates closely during the initial adoption phase is recommended to quickly identify and resolve any issues.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

Administrators should monitor the following metrics to assess the need for a rollback:

- `statefulset_unavailability_violation`: Ths metric reflects the number of times the maxUnavailable condition is violated (i.e. spec.replicas - availableReplicas > maxUnavailable).
Multiple violations of maxUnavailable might indicate issues with feature behavior.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

A manual test was performed, as follows:

1. Create a cluster in 1.33.
2. Upgrade to 1.35.
3. Create StatefulSet A with spec.updateStrategy.rollingUpdate.maxUnavailable set to 3, with 6 replicas
4. Verify a rollout and check if only 3 pods are unavailable at a time ([currently with a bug if podManagementPolicy is set to Parallel](https://github.com/kubernetes/kubernetes/issues/112307))
5. Downgrade to 1.33.
6. Verify that the rollout only has 1 pod unavailable at a time, similar to setting maxUnavailable to 1
7. Create another StatefulSet B not setting maxUnavailable (leaving it nil)
8. Upgrade to 1.35.
9. Verify that the rollout has default behavior of only having one pod unavailable at a time
   Verify that the `maxUnavailable` can be set again to StatefulSet A and test the rollout behavior

TODO:
A manual test will be performed, as follows:

1. Create a cluster in 1.33.
2. Upgrade to 1.35.
3. Create StatefulSet A with spec.updateStrategy.rollingUpdate.maxUnavailable set to 3, with 6 replicas
4. Verify a rollout and check if only 3 pods are unavailable at a time
5. Check if rollout is also fine with podManagementPolicy set to Parallel
6. Downgrade to 1.33.
7. Verify that the rollout only has 1 pod unavailable at a time, similar to setting maxUnavailable to 1 (MaxUnavailableStatefulSet feature gate disabled by default).
8. Create another StatefulSet B not setting maxUnavailable (leaving it nil)
9. Upgrade to 1.35.
10. Verify that the rollout has default behavior of only having one pod unavailable at a time
    Verify that the `maxUnavailable` can be set again to StatefulSet A and test the rollout behavior

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

If their StatefulSet rollingUpdate section has the field `maxUnavailabl`e specified with
a value different from 1. While in alpha and beta, the feature-gate needs to be enabled.

The command bellow should show the maxUnavailable value:

```
kubectl get statefulsets -o yaml | grep maxUnavailable
```

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: 
- [x] API .spec
  - Condition name: 
  - Other field: .spec.updateStrategy.rollingUpdate.maxUnavailable
- [X] Other (treat as last resort)
  - Details: Users can view the `statefulset_unavailable_replicas` or `statefulset_max_unavailable` metrics to see if there have been instances
where the feature is not working as intended.

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

Startup latency of schedulable stateful pods should follow the [existing latency SLOs](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/slos.md#steady-state-slisslos).

`statefulset_unavailable_replicas` > `statefulset_max_unavailable` must not exceed the limit.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [x] Metrics

  - Metric name: `workqueue_depth`
    - Scope: Measures the current depth of the work queue for StatefulSet.
    - Components exposing the metric: `kube-controller-manager`
  - Metric name: `workqueue_adds_total`
    - Scope: Counts the total number of StatefulSet operations added to the work queue.
    - Components exposing the metric: `kube-controller-manager`
  - Metric name: `workqueue_queue_duration_seconds`
    - Scope: Total number of seconds that items spent waiting in a specific work queue.
    - Components exposing the metric: `kube-controller-manager`
  - Metric name: `workqueue_work_duration_seconds`
    - Scope: Observes the time taken to process StatefulSet operations from the work queue.
    - Components exposing the metric: `kube-controller-manager`
- Metric name: `workqueue_retries_total`
    - Scope: Counts the total number of retries for StatefulSet update operations within the work queue. This metric provides insight into the stability and reliability of the StatefulSet update process, indicating potential issues when high.
    - Components Exposing the Metric: `kube-controller-manager`

  - Metric name: `statefulset_unavailability_violation`
    - Scope: Counts the number of times maxUnavailable has been violated (i.e. `.spec.replicas` - availableReplicas > maxUnavailable).
    - Components Exposing the Metric: `kube-controller-manager`

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

None.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

It doesn't make any extra API calls.

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

A struct gets added to every StatefulSet object which has three fields, one 32 bit integer and two fields of type string.
The struct in question is IntOrString.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

The controller-manager will see very negligible and almost un-noticeable increase in cpu usage.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The RollingUpdate will fail or will not be able to proceed if etcd or API server is unavailable and
hence this feature will also not be able to be used.

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

- Incorrect Handling of minReadySeconds During StatefulSet Updates with Parallel Pod Management
  - Detection:
    - Monitor the `statefulset_unavailable_replicas` and `statefulset_max_unavailable` metrics of the StatefulSet during rolling updates. A large value of this metric could indicate the issue.
    - Review StatefulSet events or controller logs for rapid succession of pod updates without adherence to minReadySeconds, which could confirm that the delay is not being respected.
  - Mitigations:
    - Temporarily adjust the podManagementPolicy to OrderedReady as a workaround to ensure minReadySeconds is respected during updates, though this may slow down the rollout process.
    - Setting maxUnavailable back to 1 or disabling the feature is a possibility.
    - Upgrade your cluster to 1.34, where this feature is promoted to beta, and the issue is fixed.
  - Diagnostics:
    - Events should be the first place one should look when trying to diagnose this failure mode.
    - If events are not sufficient, detailed logging could be enabled for the StatefulSet controller to capture the sequence and timing of pod updates during a rollout.
    - Look for patterns or log entries indicating that a new pod update is initiated before the minReadySeconds period has elapsed for the previously updated pod.
  - Testing: e2e tests for this issue will be added as part of this promotion.

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 2019-01-01: KEP created.
- 2019-08-30: PR Implemented with tests covered.
- bugs found in alpha and blockers to promotion @knelasevero @atiratree @bersalazar @leomichalski
  - 2025-07-07: https://github.com/kubernetes/kubernetes/pull/130909
  - 2025-09-01: https://github.com/kubernetes/kubernetes/pull/130951
- 2025-09-30: Bump to Beta.

## Drawbacks

It can lead to increased application unavailability/outage during rollouts when this feature is enabled and used (maxUnavailable set to values greater than 1).

## Alternatives

- Users who need StatefulSets stable identity and are ok with getting a slow rolling update will continue to use StatefulSets. Users who
  are not ok with a slow rolling update, will continue to use Deployments with workarounds for the scenarios mentioned in the Motivations
  section.
- Another alternative would be to use OnDelete and deploy your own Custom Controller on top of StatefulSet Pods. There you can implement
  your own logic for deleting more than one pods in a specific order. This requires more work on the user but give them ultimate flexibility.

## Infrastructure Needed (Optional)

No.
