# KEP-3973: Consider Terminating Pods in Deployments

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
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [Consideration for Other Controllers](#consideration-for-other-controllers)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Feature Impact](#feature-impact)
    - [kubectl Skew](#kubectl-skew)
- [Design Details](#design-details)
  - [Deployment Behavior Changes](#deployment-behavior-changes)
  - [ReplicaSet Status and Deployment Status Changes](#replicaset-status-and-deployment-status-changes)
  - [Deployment Completion and Progress Changes](#deployment-completion-and-progress-changes)
  - [Deployment Scaling Changes and a New Annotation for ReplicaSets](#deployment-scaling-changes-and-a-new-annotation-for-replicasets)
  - [kubectl Changes](#kubectl-changes)
  - [API](#api)
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
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
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

## Summary

Deployments have inconsistent behavior in how they handle terminating pods, depending on the rollout
strategy and when scaling the Deployments. In some scenarios it may be advantageous to wait for 
terminating pods to terminate before spinning new ones. In other scenarios it might be beneficial
to spin them as soon as possible. This KEP proposes to add a new field `.spec.podReplacementPolicy`
to Deployments to allow users to specify the desired behavior.

## Motivation
In certain cases, deployment can momentarily have more pods than described by the deployment
definition. 

For example during a rollout with a `RollingUpdate` deployment strategy the following inequation
should hold true:
(`.spec.replicas - .spec.strategy.rollingUpdate.maxUnavailable =< .status.replicas =< .spec.replicas + .spec.strategy.rollingUpdate.maxSurge`)
But the actual number of replicas (pods) can be higher due to the terminating (marked with a
`deletionTimestamp`) pods being present which are not accounted for in `.status.replicas`.

This happens not only in a rollout, but also in other cases where pods are deleted by an actor
other than the deployment controller (e.g. eviction). 

Terminating pods can stay up for a considerable amount of time (driven by pod's
`.spec.terminationGracePeriodSeconds`). Although terminating pods are not considered part of a
deployment and are not counted in its status, this can cause problems with resource usage and
scheduling:


1. Unnecessary autoscaling of nodes in tight environments and driving up cloud costs. This can hurt
   especially if multiple deployments are rolled out at the same time, or if a large
   `.spec.terminationGracePeriodSeconds` value is requested. See the following issues for more
   details: [kubernetes/kubernetes#95498](https://github.com/kubernetes/kubernetes/issues/95498),
   [kubernetes/kubernetes#99513](https://github.com/kubernetes/kubernetes/issues/99513),
   [kubernetes/kubernetes#41596](https://github.com/kubernetes/kubernetes/issues/41596),
   [kubernetes/kubernetes#97227](https://github.com/kubernetes/kubernetes/issues/97227).
2. A problem also arises in contentious environments where pods are fighting over resources. This
   can bring up exponential backoff for not yet started pods into big numbers and unnecessarily
   delay start of such pods until they pop from the queue when there are computing resources to run
   them. This can slow down the deployment considerably. This is described in issue
   [kubernetes/kubernetes#98656](https://github.com/kubernetes/kubernetes/issues/98656). In that
   issue, the resources were limited by a quota, but this can be due to other reasons as well. This
   can occur also in high availability scenarios where pods are expected to run only on certain
   nodes, and pod anti-affinity forbids to run two pods on the same node.
3. Terminating pods can still do useful work or hold old connections. Users would like to track
   this work through the deployment's status. See
   [kubernetes/kubernetes#110171](https://github.com/kubernetes/kubernetes/issues/110171) for more
   details.

[kubernetes/kubernetes#107920](https://github.com/kubernetes/kubernetes/issues/107920) issue is covering this as well.

### Goals

- Deployments should allow an option to either wait for its pods to terminate before creating new
  pods, or to create the pods immediately. This should take into consideration the Deployment
  strategy.
- Deployments and ReplicaSets should indicate a number of managed terminating pods in their status
  field.

### Non-Goals

## Proposal

This KEP proposes to introduce a new  `.spec.podReplacementPolicy` field (similar to Job's
`.spec.podReplacementPolicy` in [kubernetes/enhancements#3939](https://github.com/kubernetes/enhancements/issues/3939))
that would control how many pods should be present at any given time.

The termination of a Deployment/ReplicaSet pod is always triggered by a pod deletion due to
an enforced pod field `restartPolicy: Always`.

We are distinguishing between terminating and terminated pods. 
- Terminating pods are running pods with a `deletionTimestamp`. 
- Terminated pods are pods with a `deletionTimestamp` that have reached the `Succeeded` or `Failed` phase
  and are subsequently removed from etcd.

Unfortunately, the current behavior is inconsistent with how we treat terminating and terminated pods
in the deployment controller.

- The Recreate Deployment strategy waits for terminating pods to terminate before creating
  (scheduling) new pods.
- The RollingUpdate deployment strategy does not wait for terminating pods and creates (schedules)
  new pods immediately.
- Scaling up a Deployment also does not wait for terminating pods and creates (schedules) new pods
  right away.

Unfortunately, in Deployments with a Recreate strategy we can get mixed behavior. The
deployment will wait for old pods to terminate during a rollout, but will ignore the terminating
pods when scaling the pods. So it is still possible to end up with a larger number of pods than
`.spec.replicas`.

### User Stories (Optional)

#### Story 1

As an application user, I would prefer predictable number of pods in my cluster to prevent any
scheduling issues and unnecessary autoscaling of nodes. I would also like to achieve consistent
allocation of other scarce resources to pods.

#### Story 2

As an application user, I would like to keep the old behavior of fast scaling of pods and do not
mind the higher utilization of resources.

#### Story 3

As an application user, I would like to track the number of instances that perform useful work
during the entire lifecycle of a pod.

### Notes/Constraints/Caveats (Optional)

#### Consideration for Other Controllers

This feature is not considered for standalone ReplicaSets except for tracking the terminating pods.
The reason for this is that ReplicaSet behavior is meant to be simple and used as a building block
by other high-level controllers. If we included the PodReplacementPolicy in both ReplicaSets and
Deployments, it would be hard to reconcile these fields because a ReplicaSet only has the local view
of its own pods. The Deployment has the complete picture of all the pods (through ReplicaSet's status)
in its ReplicaSets and can make the correct balancing decision. Adding such a feature to ReplicaSets
could also pose a threat to third-party controllers that embed ReplicaSets in their resource
definitions, as this could alter their behavior.

This feature is also not desirable for StatefulSets and DaemonSets, because by design we wait until
old pods terminate before creating new pods.

This feature is already implemented for Jobs ([KEP-3939](https://github.com/kubernetes/enhancements/tree/master/keps/sig-apps/3939-allow-replacement-when-fully-terminated)).

### Risks and Mitigations

#### Feature Impact

Deployment rollouts might be slower when using the `TerminationComplete` PodReplacementPolicy.

Deployment rollouts might consume excessive resources when using the `TerminationStarted` PodReplacementPolicy.

This is mitigated by making this feature opt-in.

#### kubectl Skew
The `deployment.kubernetes.io/replicaset-replicas-before-scale` annotation should be removed during
deployment rollback when annotations are copied from the ReplicaSet to the Deployment. Support for
this removal will be added to kubectl in the same release as this feature. Therefore, rollback using
an older kubectl will not be supported until one minor release after the feature first reaches
alpha. The documentation for Deployments will include a notice about this.

If an older kubectl version is used, the impact should be minimal. The deployment may end up with an
unnecessary `deployment.kubernetes.io/replicaset-replicas-before-scale` annotation. The deployment
controller then synchronizes Deployment annotations back to the ReplicaSet. This is done by the
Deployment controller, which will ignore this new annotations if the feature gate is on.

The bug should be mainly visual (extra annotation in the Deployment), unless the feature is turned
on and off in a succession. In this case, incorrect annotations could end up on a ReplicaSet, which
would affect the scaling proportions during a rollout.

## Design Details

### Deployment Behavior Changes

Recreate rollout logic:
- Terminating (TerminationStarted):
    1. Scale down old ReplicaSet(s) to 0.
    2. Wait until all the pods are at least terminating.
    3. Create new replica set.
- Terminated (TerminationComplete): Current behaviour.

RollingUpdate rollout logic:
- Terminating (TerminationStarted): Current behaviour.
- Terminated (TerminationComplete): When checking if a new replica set can be scaled up during a rollout, we should
  consider terminating pods of all ReplicaSets as well and not spawn an amount of replicas that
  would be higher than Deployment's `.spec.replicas + .spec.strategy.rollingUpdate.maxSurge`.
  This will be implemented by checking ReplicaSet's `.spec.replicas`, `.status.replicas` and
  `.status.terminatingReplicas` to determine the number of pods.

Scaling logic:
- Terminating (TerminationStarted): Current behaviour.
- Terminated (TerminationComplete):
    - When scaling up across one or more ReplicaSets, we should consider terminating pods of all
      ReplicaSets as well and not spawn replicas that would be higher than Deployment's
      `.spec.replicas + .spec.strategy.rollingUpdate.maxSurge`. This will be implemented by
      checking ReplicaSet's `.spec.replicas`, `.status.replicas` and `.status.terminatingReplicas`
      to determine the number of pods. See [Deployment Scaling Changes and a New Annotation for ReplicaSets](#deployment-scaling-changes-and-a-new-annotation-for-replicasets)
      for more details.
    - Changing scaling down logic is not necessary, and we can scale down as many pods as we want
      because the policy does not affect this since we are not replacing the pods.

### ReplicaSet Status and Deployment Status Changes

We should keep the current counting behavior for `.status.replicas` regardless of the
PodReplacementPolicy, for backwards compatibility reasons. Current consumers of the Deployment API
are only expecting non-terminating pods to be present in this field.

To satisfy the requirement for tracking terminating pods, and for implementation purposes,
we propose a new field `.status.terminatingReplicas` to the ReplicaSet's and Deployment's
status.

### Deployment Completion and Progress Changes

Currently, when the latest ReplicaSet is fully saturated and all of its pods become available, the
Deployment is declared complete. However, there may still be old terminating pods. These pods can
still be ready and hold/accept connections, meaning that the transition to the latest revision is
not fully complete.

To avoid unexpected behavior, we should not declare the deployment complete until all of its
terminating replicas have been fully terminated. We will therefore delay setting a `NewRSAvailable`
reason to the `DeploymentProgressing` condition, when `TerminationComplete` policy is used.

We will also update the `LastUpdateTime` of the `DeploymentProgressing` condition when the number of
terminating pods decreases to reset the progress deadline.

### Deployment Scaling Changes and a New Annotation for ReplicaSets

Currently, scaling is done proportionally over all ReplicaSets to mitigate the risk of losing
availability during a rolling update.

To calculate the new ReplicaSet size, we need to know
- `replicasBeforeScale`: The `.spec.replicas` of the ReplicaSet before the scaling began.
- `deploymentMaxReplicas`: Equals to `.spec.replicas + .spec.strategy.rollingUpdate.maxSurge` of
  the current Deployment.
- `deploymentMaxReplicasBeforeScale`: Equals to
  `.spec.replicas + .spec.strategy.rollingUpdate.maxSurge` of the old Deployment. This information
  is stored in the `deployment.kubernetes.io/max-replicas` annotation in each ReplicaSet.

Then we can calculate a new size for each ReplicaSet proportionally as follows:

$$
newReplicaSetReplicas = replicasBeforeScale * \frac{deploymentMaxReplicas}{deploymentMaxReplicasBeforeScale}
$$

This is currently done in the [getReplicaSetFraction](https://github.com/kubernetes/kubernetes/blob/1cfaa95cab0f69ecc62ad9923eec2ba15f01fc2a/pkg/controller/deployment/util/deployment_util.go#L492-L512)
function. The leftover pods are added to the largest ReplicaSet (or newest if more than one ReplicaSet has the largest number of pods).

This results in the following scaling behavior. 

The first scale operation occurs at T2 and the second scale at T3.

| Time | Terminating Pods | RS1 Replicas | RS2 Replicas | RS3 Replicas | All RS Total | Deployment .spec.replicas | Deployment .spec.replicas + MaxSurge | Scale ratio |
|------|------------------|--------------|--------------|--------------|--------------|---------------------------|--------------------------------------|-------------| 
| T1   | any amount       | 60           | 30           | 20           | 110          | 100                       | 110                                  | -           |
| T2   | any amount       | 71           | 35           | 24           | 130          | 120                       | 130                                  | 1.182       |
| T3   | any amount       | 76           | 38           | 26           | 140          | 130                       | 140                                  | 1.077       | 

With the `TerminationComplete` PodReplacementPolicy, scaling cannot proceed immediately if there
are terminating pods present, in order to adhere to the Deployment constraints. We need to scale
some ReplicaSets fully and some partially. And we have to postpone scaling to the future when
terminating pods disappear.

A single scale operation occurs at T2.

| Time | Terminating Pods | RS1 Replicas | RS2 Replicas | RS3 Replicas | All RS Total | Deployment .spec.replicas | Deployment .spec.replicas + MaxSurge | Scale ratio |
|------|------------------|--------------|--------------|--------------|--------------|---------------------------|--------------------------------------|-------------| 
| T1   | 15               | 50           | 30           | 20           | 100          | 100                       | 110                                  | -           |
| T2   | 15               | 59           | 35           | 21           | 115          | 120                       | 130                                  | 1.182       |
| T3   | 5                | 66           | 35           | 24           | 125          | 120                       | 130                                  | -           | 
| T4   | 0                | 71           | 35           | 24           | 130          | 120                       | 130                                  | -           | 

To proceed with the scaling in the future (T3), we need to remember both `replicasBeforeScale` and
`deploymentMaxReplicasBeforeScale` to calculate the original scale ratio. The terminating pods can
take a long time to terminate and there can be many steps and ReplicaSet updates between T2 and T3.
If we were to use the current number of ReplicaSet or Deployment replicas in any of these steps
(including T3), we would calculate an incorrect scale ratio.

- `deploymentMaxReplicasBeforeScale` is already stored in the `deployment.kubernetes.io/max-replicas`
  ReplicaSet annotation. The main change is that we need to keep the old Deployment max replicas
  value in the annotation until the partial scale for a ReplicaSet is complete.
- To remember `replicasBeforeScale`, we will introduce a new annotation called
  `deployment.kubernetes.io/replicaset-replicas-before-scale`, which will be added to the
  Deployment's ReplicaSets that are being partially scaled. This annotation will be removed once
  the partial scaling is complete. This annotation will be added and managed by the deployment
  controller.

These two ReplicaSet annotation will be used to calculate the original scale ratio for the partial
scaling.

The following example shows a first scale at T2 and a second scale at T3.

| Time | Terminating Pods | RS1 Replicas | RS2 Replicas | RS3 Replicas | All RS Total | Deployment .spec.replicas | Deployment .spec.replicas + MaxSurge | Scale ratio           |
|------|------------------|--------------|--------------|--------------|--------------|---------------------------|--------------------------------------|-----------------------| 
| T1   | 15               | 50           | 30           | 20           | 100          | 100                       | 110                                  | -                     |
| T2   | 15               | 59           | 35           | 21           | 115          | 120                       | 130                                  | 1.182                 |
| T3   | 15               | 66           | 38           | 21           | 125          | 130                       | 140                                  | 1.077 (1.273 from T1) | 
| T4   | 5                | 72           | 38           | 25           | 135          | 130                       | 140                                  | -                     | 
| T5   | 0                | 77           | 38           | 25           | 140          | 130                       | 140                                  | -                     | 

- At T2, a ful scale was done for RS1 with a ratio of 1.182. RS1 can then use the new scale ratio
  at T3 with a value of 1.077.
- RS2 has been partially scaled (1.182 ratio) and RS3 has not been scaled at all at T2 due to the
  terminating pods. When a new scale occurs at T3, RS2 and RS3 have not yet completed the first
  scale. So their annotations still point to the T1 state. A new ratio of 1.273 is calculated and
  used for the second scale.

As we can see, we will get a slightly different result when compared to the first table. This is
due to the consecutive scales and the fact that the last scale is not yet fully completed.

The consecutive partial scaling behavior is a best effort. We still adhere to all deployment
constraints and have a bias toward scaling the largest ReplicaSet. To implement this properly we
would have to introduce a full scaling history, which is probably not worth the added complexity.

### kubectl Changes

Similar to `deployment.kubernetes.io/max-replicas`, we have to remove
`deployment.kubernetes.io/replicaset-replicas-before-scale` annotations from [annotationsToSkip](https://github.com/kubernetes/kubernetes/blob/9e2075b3c87061d25759b0ad112266f03601afd8/staging/src/k8s.io/kubectl/pkg/polymorphichelpers/rollback.go#L184)
to support rollbacks.
See [kubectl Skew](#kubectl-skew) for more details.

### API

```golang
// DeploymentPodReplacementPolicy specifies the policy for creating Deployment Pod replacements.
// Default is a mixed behavior depending on the DeploymentStrategy
// +enum 
type DeploymentPodReplacementPolicy string
const (
// TerminationStarted policy creates replacement Pods when the old Pods start
// terminating (have a non-null .metadata.deletionTimestamp). The total number
// of Deployment Pods can be greater than specified by the Deployment's
// .spec.replicas and the DeploymentStrategy.
TerminationStarted DeploymentPodReplacementPolicy = "TerminationStarted"
// TerminationComplete policy creates replacement Pods only when the old Pods
// are fully terminated (reach Succeeded or Failed phase). The old Pods are
// subsequently removed. The total number of the Deployment Pods is
// limited by the Deployment's .spec.replicas and the DeploymentStrategy.
//
// This policy will also delay declaring the deployment as complete until all
// of its terminating replicas have been fully terminated.
TerminationComplete DeploymentPodReplacementPolicy = "TerminationComplete"
)
```

```golang
type DeploymentSpec struct {
    ...
    // podReplacementPolicy specifies when to create replacement Pods. 
	// Possible values are:
    // - TerminationStarted policy creates replacement Pods when the old Pods start
	//   terminating (have a non-null .metadata.deletionTimestamp). The total number
	//   of Deployment Pods can be greater than specified by the Deployment's
	//   .spec.replicas and the DeploymentStrategy.
    // - TerminationComplete policy creates replacement Pods only when the old Pods
	//   are fully terminated (reach Succeeded or Failed phase). The old Pods are
	//   subsequently removed. The total number of the Deployment Pods is
	//   limited by the Deployment's .spec.replicas and the DeploymentStrategy.
	//   This policy will also delay declaring the deployment as complete until all
	//   of its terminating replicas have been fully terminated.
    //
    // The default behavior when the policy is not specified depends on the DeploymentStrategy:
	// - Recreate strategy uses TerminationComplete behavior when recreating the deployment,
	//   but uses TerminationStarted when scaling the deployment.
	// - RollingUpdate strategy uses TerminationStarted behavior for both rolling out and
	//   scaling the deployments.
	//
	// This is an alpha field. Enable DeploymentPodReplacementPolicy to be able to
	// use this field.
    // +optional
    PodReplacementPolicy *DeploymentPodReplacementPolicy `json:"podReplacementPolicy,omitempty" protobuf:"bytes,10,opt,name=podReplacementPolicy,casttype=podReplacementPolicy"`
    ...
}
```

```golang
type ReplicaSetStatus struct {
    ...
    // Replicas is the most recently observed number of non-terminating replicas.
    // More info: https://kubernetes.io/docs/concepts/workloads/controllers/replicationcontroller/#what-is-a-replicationcontroller
    Replicas int32 `json:"replicas" protobuf:"varint,1,opt,name=replicas"`

    // The number of non-terminating pods that have labels matching the labels of the pod template of the replicaset.
    // +optional
    FullyLabeledReplicas int32 `json:"fullyLabeledReplicas,omitempty" protobuf:"varint,2,opt,name=fullyLabeledReplicas"`

    // readyReplicas is the number of non-terminating pods targeted by this ReplicaSet with a Ready Condition.
    // +optional
    ReadyReplicas int32 `json:"readyReplicas,omitempty" protobuf:"varint,4,opt,name=readyReplicas"`

    // The number of available non-terminating replicas (ready for at least minReadySeconds) for this replica set.
    // +optional
    AvailableReplicas int32 `json:"availableReplicas,omitempty" protobuf:"varint,5,opt,name=availableReplicas"`

    // The number of terminating pods (have a non-null .metadata.deletionTimestamp) for this replica set. 
    //
    // This is an alpha field. Enable DeploymentPodReplacementPolicy to be able to use this field.
    // +optional
    TerminatingReplicas *int32 `json:"terminatingReplicas,omitempty" protobuf:"varint,7,opt,name=terminatingReplicas"`
    ...
}
```

```golang
type DeploymentStatus struct {
    ...
    // Total number of non-terminating pods targeted by this deployment (their labels match the selector).
    // +optional
    Replicas int32 `json:"replicas,omitempty" protobuf:"varint,2,opt,name=replicas"`

    // Total number of non-terminating pods targeted by this deployment that have the desired template spec.
    // +optional
    UpdatedReplicas int32 `json:"updatedReplicas,omitempty" protobuf:"varint,3,opt,name=updatedReplicas"`

    // readyReplicas is the number of non-terminating pods targeted by this Deployment with a Ready Condition.
    // +optional
    ReadyReplicas int32 `json:"readyReplicas,omitempty" protobuf:"varint,7,opt,name=readyReplicas"`

    // Total number of available non-terminating pods (ready for at least minReadySeconds) targeted by this deployment.
    // +optional
    AvailableReplicas int32 `json:"availableReplicas,omitempty" protobuf:"varint,4,opt,name=availableReplicas"`

    // Total number of unavailable pods targeted by this deployment. This is the total number of
    // pods that are still required for the deployment to have 100% available capacity. They may
    // either be pods that are running but not yet available or pods that still have not been created.
    // +optional
    UnavailableReplicas int32 `json:"unavailableReplicas,omitempty" protobuf:"varint,5,opt,name=unavailableReplicas"`

    // Total number of terminating pods (have a non-null .metadata.deletionTimestamp) targeted by this deployment.
    //
    // This is an alpha field. Enable DeploymentPodReplacementPolicy to be able to use this field.
    // +optional
    TerminatingReplicas *int32 `json:"terminatingReplicas,omitempty" protobuf:"varint,9,opt,name=terminatingReplicas"`
    ...
}
```

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

We assess that the deployment and replicaset controllers have adequate test coverage for places
which might be impacted by this enhancement. Thus, no additional tests prior implementing this
enhancement are needed.

##### Unit tests

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

Unit tests covering:

ReplicaSet
- The current behavior remains unchanged when the DeploymentPodReplacementPolicy feature gate is disabled. 
  The `.status.terminatingReplicas` field should be 0 in that case.
- Add a new test that correctly counts .status.TerminatingReplicas when the DeploymentPodReplacementPolicy feature gate is enabled.


Deployment
- The current behavior remains unchanged when the DeploymentPodReplacementPolicy feature gate is disabled or PodReplacementPolicy is nil.
  The `.status.terminatingReplicas` field should be 0 in that case.
- Add a test wrapper for any relevant tests, to ensure that they are run with all possible PodReplacementPolicy values correctly.
  The relevant tests are those that expect some behavior on Pod deletion, and are affected by this change.
- New unit tests should be added for any new helper functions.
- Test that the status is computed correctly.
- Test feature gate enablement and disablement.


The core packages (with their unit test coverage) which are going to be modified during the implementation:
- `k8s.io/kubernetes/pkg/apis/apps/v1`: `9 December 2023` - `71.4%` <!--(extension of Deployment and ReplicaSet API)-->
- `k8s.io/kubernetes/pkg/apis/apps/validation`: `9 December 2023` - `92.3%` <!--(validation of the PodReplacementPolicy)-->
- `k8s.io/kubernetes/pkg/controller/deployment`: `9 December 2023` - `61.7%` <!--(implementation of the PodReplacementPolicy)-->
- `k8s.io/kubernetes/pkg/controller/deployment/util`: `9 December 2023` - `50.1%` <!--(implementation of the PodReplacementPolicy)-->
- `k8s.io/kubernetes/pkg/controller/replicaset`: `9 December 2023` - `78.9%` <!--(implementation of the counting of the TerminatingReplicas)-->
- `k8s.io/kubernetes/pkg/controller`: `9 December 2023` - `71.2%` <!--(creating or updating the utility functions in controller_utils.go)-->

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

Integration tests covering:

<!-- test/integration/replicaset/replicaset_test.go -->
ReplicaSet
- The current behavior remains unchanged when the DeploymentPodReplacementPolicy feature gate is disabled.
- Add a new test that correctly counts `.status.terminatingReplicas` when the DeploymentPodReplacementPolicy feature gate is enabled.

<!--
- <test>: <link to test coverage>
-->

<!-- test/integration/deployment/deployment_test.go -->
Deployment
- The current behavior remains unchanged when the DeploymentPodReplacementPolicy feature gate is disabled or PodReplacementPolicy is nil.
- Add a test wrapper for any relevant tests, to ensure that they are run with all possible PodReplacementPolicy values correctly.
  The relevant tests are those that expect some behavior on Pod deletion, and are affected by this change.
- Add new tests that observe rollout and scaling transitions for all possible PodReplacementPolicy values and
  ensure that `.status.terminatingReplicas` is correctly counted when the DeploymentPodReplacementPolicy feature gate is enabled.

<!--
- <test>: <link to test coverage>
-->

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

<!--
test/e2e/apps/replica_set.go
test/e2e/upgrades/apps/replica_sets.go

test/e2e/apps/deployment.go
test/e2e/upgrades/apps/deployments.go
test/e2e/storage/utils/deployment.go
-->

- Update all existing e2e tests to account for `.status.TerminatingReplicas` field in both ReplicaSets and Deployments.
- Test that a Deployment with `RollingUpdate` strategy and a `TerminationComplete` PodReplacementPolicy does not
  exceed the amount of pods specified by `spec.replicas + .spec.strategy.rollingUpdate.maxSurge` when
  rolling out new revisions and/or scaling the deployment at any point in time.
- Test scaling of Deployments that are in the middle of a rollout (even with more than 2 revisions).
  Verify that scaling is done proportionally across all ReplicaSets when terminating pods are
  present. Scale these deployments in a succession, even when the previous scale has not yet
  completed.

<!--
- <test>: <link to test coverage>
-->

### Graduation Criteria

#### Alpha

- Feature gate disabled by default.
- Unit, enablement/disablement, e2e, and integration tests implemented and passing.
- Document [kubectl Skew](#kubectl-skew) for alpha.


#### Beta
- Feature gate enabled by default.
- Any test that checks Deployment and Replicaset status is updated to count updates to `.status.TerminatingReplicas`.
- `.spec.podReplacementPolicy` is nil by default and preserves the original behavior.
- E2e and integration tests are in Testgrid and linked in the KEP.
- add new metrics to `kube-state-metrics`
- Remove documentation for [kubectl Skew](#kubectl-skew) that was introduced in alpha.


#### GA
- Every bug report is fixed.
- Confirm the stability of e2e and integration tests.
- DeploymentPodReplacementPolicy feature gate is ignored.

### Upgrade / Downgrade Strategy

No changes required for existing cluster to use the enhancement.

### Version Skew Strategy

We need to consider the version skew between kube-controller-manager and the apiserver.

If the feature is enabled on the apiserver, but not in the kube-controller-manager, then the `.spec.podReplacementPolicy`
field can be set, but the feature will not function.


If the feature is not enabled on the apiserver, and it is enabled in the kube-controller-manager, then
- The feature cannot be used for new workloads.
- Workloads that have the `.spec.podReplacementPolicy` field set will use the new behavior.

Also, as mentioned in [kubectl Skew](#kubectl-skew), kubectl skew is not supported in the alpha version.

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

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: DeploymentPodReplacementPolicy
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-controller-manager

###### Does enabling the feature change any default behavior?

No, the behavior is only changed when users specify the `podReplacementPolicy` in
the Deployment spec.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. 

By disabling the feature:
- Extra pods can appear during a deployment rollout or scaling. This can increase the number of pods
  that need to be scheduled, and it can have an impact on the resource consumption.
- Actors reading `.status.TerminatingReplicas` for ReplicaSet and Deployments will see the field to
  be omitted (observe 0 pods), once the status is reconciled by the controllers.

As mentioned in [kubectl Skew](#kubectl-skew), kubectl skew is not supported in alpha. If an older
unsupported version of kubectl was used, it is important to remove the
`deployment.kubernetes.io/replicaset-replicas-before-scale` annotation from all Deployments and
ReplicaSets after disabling this feature. This should prevent any unexpected behavior on the next
enablement.

###### What happens if we reenable the feature if it was previously rolled back?

The ReplicaSet and Deployment controllers will start reconciling the `.status.TerminatingReplicas`
and behave according to the `.spec.podReplacementPolicy`.

Similar to the section above, it is important to make sure that the
`deployment.kubernetes.io/replicaset-replicas-before-scale` annotation is removed from all
Deployments and ReplicaSets before the re-enablement.

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

Appropriate enablement/disablement tests will be added to the replicaset and deployment `strategy_test.go`
and unit tests in alpha.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

The rollout should not fail as the feature is hidden behind a feature gate and new optional field.

During a rollout a new `.status.TerminatingReplicas` field will be introduced on Deployments and
ReplicaSets. This can cause problems for existing clients and users who do not expect and
incorrectly handle new status fields.

During a rollback, the `.spec.podReplacementPolicy` field will be ignored. This will cause
workloads that use this field to fall back to the original deployment rollout and scaling behaviour.
This can be problematic for workloads that are not expecting:
 - excessive number of pods
 - excessive resource consumption
 - slower or faster deployment rollout or scaling speed

This can also affect other workloads, for example by exhausting resources on a node.


###### What specific metrics should inform a rollback?

kube-controller-manager's `deployment` workqueue metrics such as `workqueue_retries_total`,
`workqueue_depth`, `workqueue_work_duration_seconds_bucket` can be observed. A sudden increase in
these metrics can indicate a problem with the `DeploymentPodReplacementPolicy` feature.

Deployment pods can be watched for incorrect number of pods during a deployment rollout or scaling.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

TBD: Manual upgrade->downgrade->upgrade path will be tested.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The operator can observe `.status.terminatingReplicas` on both ReplicaSets and Deployments.
The same field is being added as a metric and can be observed there as well:
`kube_replicaset_status_terminating_replicas` and `kube_deployment_status_replicas_terminating`.

###### How can someone using this feature know that it is working for their instance?


When using the `TerminationComplete` PodReplacementPolicy, the user should not see an excess of running and
terminating pods created that is greater than the deployment's `.spec.replicas` and its deployment
strategy.

When using the `TerminationStarted` PodReplacementPolicy, the user should see an excess of running and
terminating pods created that is greater than the deployment's `.spec.replicas` and its deployment
strategy. This will in turn make the deployment rollout faster.

The terminating pods can be observed in `.status.terminatingReplicas` on both ReplicaSets and Deployments.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

We do not propose any SLO/SLI for this feature.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `workqueue_retries_total` can be used to see if there is a sudden increase in sync retries after enabling the feature
    - Aggregation method: name = "deployment"
    - Components exposing the metric: `kube-controller-manager`
  - Metric name: `workqueue_depth` can be used to see if there is a sudden increase in unprocessed deployment objects after enabling the feature
    - Aggregation method: name = "deployment"
    - Components exposing the metric: `kube-controller-manager`
  - Metric name: `workqueue_work_duration_seconds_bucket` can be used to see if there is a sudden increase in duration of syncing deployment objects after enabling the feature
    - Aggregation method: name = "deployment"
    - Components exposing the metric: `kube-controller-manager`
  - Metric name: `kube_replicaset_status_terminating_replicas`
    - Components exposing the metric: `kube-state-metrics` (TBD)
  - Metric name: `kube_deployment_status_replicas_terminating`
    - Components exposing the metric: `kube-state-metrics` (TBD)

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

`kube_replicaset_status_terminating_replicas` and `kube_deployment_status_replicas_terminating` will be added to kube-state-metrics in beta.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

No, it will use the existing calls for creating and reconciling Deployments, ReplicaSets, and Pods.
The number of calls may be higher in some scenarios if a large `spec.strategy.rollingUpdate.maxSurge`
value is specified. But the maximum number of calls per deployment should be similar to when
`spec.strategy.rollingUpdate.maxSurge` value is set to 1 when the feature is disabled.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes.

- API: ReplicaSet
  Estimated increase in size:
    - New field in ReplicaSet status about 4 bytes.
- API: Deployment
  Estimated increase in size:
    - New field in Deployment spec about 11 bytes.
    - New field in Deployment status about 4 bytes.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

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

No change in behavior. Deployment and ReplicaSet controllers might fail in reconciling their
objects and in turn stop deployment rollout or scaling.

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

N/A

TBD

###### What steps should be taken if SLOs are not being met to determine the problem?

Inspecting the `kube-controller-manager` logs at an increased log level for any failures in
deployment and replicaset controllers.

## Implementation History

- 2023-05-01: First version of the KEP opened (https://github.com/kubernetes/enhancements/pull/3974).
- 2023-12-12: Second version of the KEP opened (https://github.com/kubernetes/enhancements/pull/4357).
- 2024-29-05: Added a Deployment Scaling Changes and a New Annotation for ReplicaSets section (https://github.com/kubernetes/enhancements/pull/4670).
- 2024-22-11: Added a Deployment Completion and Progress Changes section (https://github.com/kubernetes/enhancements/pull/4976).

## Drawbacks

Deployment might be slower when using the `TerminationComplete` PodReplacementPolicy.

Deployment might consume excessive resources when using the `TerminationStarted` PodReplacementPolicy.

## Alternatives

This feature could be implemented by a different controller that manages ReplicaSets,
but, there is a need for this feature to be implemented by a Deployment controller, because
many existing workloads can benefit from this feature.

## Infrastructure Needed (Optional)
