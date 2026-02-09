# KEP-3973: Consider Terminating Pods in Deployments

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1 (Optional)](#story-1-optional)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
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
to spin them as soon as possible. This KEP proposes to add new fields `status.terminatingReplicas`
to both Deployments and ReplicaSets in order to improve managed pod observability to eventually
improve these scenarios in future efforts ([KEP-5882](https://github.com/kubernetes/enhancements/issues/5882)).

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

- Deployments and ReplicaSets should indicate a number of managed terminating pods in their status
  field.

### Non-Goals

- Changes to scaling or rollout behavior that take terminating pods into consideration.

## Proposal

This KEP proposes to add new fields `status.terminatingReplicas` to both Deployments and ReplicaSets
to track the number of terminating pods.

### User Stories (Optional)

#### Story 1 (Optional)

As an application user, I would like to track the number of instances that perform useful work
during the entire lifecycle of a pod.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

## Design Details

We should keep the current counting behavior for `.status.replicas` regardless of any policy or
feature gate, for backwards compatibility reasons. Current consumers of the Deployment API
are only expecting non-terminating pods to be present in this field.

To satisfy the requirement for tracking terminating pods, and for implementation purposes of
follow-up feature(s), we propose a new field `.status.terminatingReplicas` to the ReplicaSet's and
Deployment's status. The follow-up feature, Deployment Pod Replacement Policy, is being implemented
by [KEP-5882](https://github.com/kubernetes/enhancements/issues/5882).

### API

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

    // The number of terminating pods for this replica set. Terminating pods have a non-null .metadata.deletionTimestamp
    // and have not yet reached the Failed or Succeeded .status.phase.
    //
    // This is a beta field and requires enabling DeploymentReplicaSetTerminatingReplicas feature (enabled by default).
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

    // Total number of terminating pods targeted by this deployment. Terminating pods have a non-null
    // .metadata.deletionTimestamp and have not yet reached the Failed or Succeeded .status.phase.
    //
    // This is a beta field and requires enabling DeploymentReplicaSetTerminatingReplicas feature (enabled by default).
    // +optional
    TerminatingReplicas *int32 `json:"terminatingReplicas,omitempty" protobuf:"varint,9,opt,name=terminatingReplicas"`
    ...``
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
- The current behavior remains unchanged when the DeploymentReplicaSetTerminatingReplicas feature gate is disabled. 
  The `.status.terminatingReplicas` field should be 0 in that case.
- Add a new test that correctly counts .status.terminatingReplicas when the DeploymentReplicaSetTerminatingReplicas feature gate is enabled.

Deployment
- The current behavior remains unchanged when the DeploymentReplicaSetTerminatingReplicas feature gate is disabled.
  The `.status.terminatingReplicas` field should be 0 in that case.
- Add a new test that correctly counts .status.terminatingReplicas when the DeploymentReplicaSetTerminatingReplicas feature gate is enabled.

The core packages (with their unit test coverage) which are going to be modified during the implementation:
- `k8s.io/kubernetes/pkg/apis/apps/v1`: `9 December 2023` - `71.4%`
- `k8s.io/kubernetes/pkg/apis/apps/validation`: `9 December 2023` - `92.3%`
- `k8s.io/kubernetes/pkg/controller/deployment`: `9 December 2023` - `61.7%`
- `k8s.io/kubernetes/pkg/controller/replicaset`: `9 December 2023` - `78.9%`

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
- The current behavior remains unchanged when the DeploymentReplicaSetTerminatingReplicas feature gate is disabled.
- Add a new test that correctly counts `.status.terminatingReplicas` when the DeploymentReplicaSetTerminatingReplicas feature gate is enabled.

<!--
- <test>: <link to test coverage>
-->

- [TestTerminatingReplicas](https://github.com/kubernetes/kubernetes/blob/bfafa32d90958a8fe7a2ce09ed553fdfef4edd98/test/integration/replicaset/replicaset_test.go#L1073-L1140): https://storage.googleapis.com/k8s-triage/index.html?test=TestTerminatingReplicas


<!-- test/integration/deployment/deployment_test.go -->
Deployment
- The current behavior remains unchanged when the DeploymentReplicaSetTerminatingReplicas feature gate is disabled.
- Add a new test that correctly counts `.status.terminatingReplicas` when the DeploymentReplicaSetTerminatingReplicas feature gate is enabled.

- [TestTerminatingReplicasDeploymentStatus](https://github.com/kubernetes/kubernetes/blob/bfafa32d90958a8fe7a2ce09ed553fdfef4edd98/test/integration/deployment/deployment_test.go#L1316-L1409): https://storage.googleapis.com/k8s-triage/index.html?test=TestTerminatingReplicasDeploymentStatus
- [TestRecreateDeploymentForPodReplacement](https://github.com/kubernetes/kubernetes/blob/bfafa32d90958a8fe7a2ce09ed553fdfef4edd98/test/integration/deployment/deployment_test.go#L1411-L1626): https://storage.googleapis.com/k8s-triage/index.html?test=TestRecreateDeploymentForPodReplacement
- [TestRollingUpdateAndProportionalScalingForDeploymentPodReplacement](https://github.com/kubernetes/kubernetes/blob/bfafa32d90958a8fe7a2ce09ed553fdfef4edd98/test/integration/deployment/deployment_test.go#L1628-L1831): https://storage.googleapis.com/k8s-triage/index.html?test=TestRollingUpdateAndProportionalScalingForDeploymentPodReplacement

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

N/A - the testing should be fully covered by integration tests. This feature
(`.status.terminatingReplicas`) is planned to be used by [KEP-5882](https://github.com/kubernetes/enhancements/issues/5882)
so it will eventually be part of the e2e test suite.

### Graduation Criteria

#### Alpha

- Feature gates disabled by default.
- Unit, enablement/disablement, e2e, and integration tests implemented and passing.


#### Beta
- Feature gates enabled by default.
- Any test that checks Deployment and Replicaset status is updated to count updates to `.status.terminatingReplicas`.
- Integration tests are in Testgrid and linked in the KEP.
- Add new metrics to `kube-state-metrics`.


#### GA
- Every bug report is fixed.
- Confirm the stability of integration tests.
- DeploymentReplicaSetTerminatingReplicas feature gate is ignored.

### Upgrade / Downgrade Strategy

The kube-apiserver should be upgraded first and downgraded last in order to ensure that the
kube-controller-manager can update the status fields.

### Version Skew Strategy

We need to consider the version skew between kube-controller-manager and the apiserver.

If the feature is not enabled on the apiserver or on the kube-controller-manager, then the
`.status.terminatingReplicas` will not be reconciled and cannot be used to estimate the number of
terminating replicas on both ReplicaSets and Deployments.

The kube-apiserver should be upgraded first and downgraded last in order to ensure that the
kube-controller-manager can update the status fields.

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
  - Feature gate name: DeploymentReplicaSetTerminatingReplicas
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-controller-manager

###### Does enabling the feature change any default behavior?

Yes, we start reporting `.status.TerminatingReplicas` for ReplicaSet and Deployments.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. 

By disabling the feature:
- Actors reading `.status.TerminatingReplicas` for ReplicaSet and Deployments will see the field to
  be omitted (observe 0 pods), once the status is reconciled by the controllers.

###### What happens if we reenable the feature if it was previously rolled back?

The ReplicaSet and Deployment controllers will start reconciling the `.status.terminatingReplicas`
again.

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

Appropriate enablement/disablement tests have been added to the replicaset and deployment `strategy_test.go`
and unit tests in alpha.
- [TestReplicaSetStatusStrategyWithDeploymentReplicaSetTerminatingReplicas](https://github.com/kubernetes/kubernetes/blob/cb58c79c767ee1348cd59e075fb6248a10f8eef2/pkg/registry/apps/replicaset/strategy_test.go#L156-L218)
- [TestStatusUpdatesWithDeploymentReplicaSetTerminatingReplicas](https://github.com/kubernetes/kubernetes/blob/cb58c79c767ee1348cd59e075fb6248a10f8eef2/pkg/registry/apps/deployment/strategy_test.go#L70-L132)

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

The rollout should not fail as the feature is hidden behind a feature gate and new optional field.

During a rollout a new `.status.terminatingReplicas` field will be introduced on Deployments and
ReplicaSets. This can cause problems for existing clients and users who do not expect and
incorrectly handle new status fields.

###### What specific metrics should inform a rollback?

kube-controller-manager's `deployment` workqueue metrics such as `workqueue_retries_total`,
`workqueue_depth`, `workqueue_work_duration_seconds_bucket` can be observed. A sudden increase in
these metrics can indicate a problem with the `DeploymentReplicaSetTerminatingReplicas` feature.


###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

1. Create a cluster in 1.34.
2. Create a Deployment and observe that the fields `.status.terminatingReplicas` are missing in both
   the ReplicaSet and the Deployment that was created.
3. Upgrade to 1.35.
4. Trigger a new Deployment rollout and observe that the fields `.status.terminatingReplicas`
   are being properly reconciled in both the ReplicaSet and the Deployment when the pods are deleted.
5. Downgrade to 1.34.
6. Observe that the fields `.status.terminatingReplicas` are missing in both
   the ReplicaSet and the Deployment.
7. Upgrade to 1.35.
8. Trigger a new Deployment rollout and observe that the fields `.status.terminatingReplicas`
   are being properly reconciled in both the ReplicaSet and the Deployment when the pods are deleted.


###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The operator can observe `.status.terminatingReplicas` on both ReplicaSets and Deployments.
The same field is being added as a metric and can be observed there as well:
`kube_replicaset_status_terminating_replicas` and `kube_deployment_status_replicas_terminating`.

###### How can someone using this feature know that it is working for their instance?

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
    - Components exposing the metric: `kube-state-metrics`
  - Metric name: `kube_deployment_status_replicas_terminating`
    - Components exposing the metric: `kube-state-metrics`

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

`kube_replicaset_status_terminating_replicas` and `kube_deployment_status_replicas_terminating` have been added to kube-state-metrics during beta graduation (https://github.com/kubernetes/kube-state-metrics/pull/2708).

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

No, it will use the existing calls for creating and reconciling Deployments, ReplicaSets, and Pods.

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

###### What steps should be taken if SLOs are not being met to determine the problem?

Inspecting the `kube-controller-manager` logs at an increased log level for any failures in
deployment and replicaset controllers.

## Implementation History

- 2023-05-01: First version of the KEP opened (https://github.com/kubernetes/enhancements/pull/3974).
- 2023-12-12: Second version of the KEP opened (https://github.com/kubernetes/enhancements/pull/4357).
- 2024-05-29: Added a Deployment Scaling Changes and a New Annotation for ReplicaSets section (https://github.com/kubernetes/enhancements/pull/4670).
- 2024-11-22: Added a Deployment Completion and Progress Changes section (https://github.com/kubernetes/enhancements/pull/4976).
- 2025-04-01: Introduced DeploymentReplicaSetTerminatingReplicas FG to split .status.terminatingReplicas feature from DeploymentPodReplacementPolicy (https://github.com/kubernetes/kubernetes/pull/131088)
- 2025-06-11: Fixed ReplicationController reconciliation when the DeploymentReplicaSetTerminatingReplicas feature gate is enabled (https://github.com/kubernetes/kubernetes/issues/131821)
- 2026-02-03: [KEP-3973](https://github.com/kubernetes/enhancements/issues/3973) was split into a [KEP-5882](https://github.com/kubernetes/enhancements/issues/5882), which focuses on the DeploymentPodReplacementPolicy feature.

## Drawbacks

N/A

## Alternatives

N/A

## Infrastructure Needed (Optional)
