# KEP-2839: In-use protection

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
- [Glossary](#glossary)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Other Design Considerations](#other-design-considerations)
    - [Behavior with <code>ownerReference</code>](#behavior-with-)
    - [Namespace Deletion](#namespace-deletion)
    - [Block Adding Additional <code>Liens</code> while Deleting](#block-adding-additional--while-deleting)
    - [Race of removing/adding liens](#race-of-removingadding-liens)
    - [Unresolved issues](#unresolved-issues)
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

This KEP proposes a generic feature to protect objects from deletion while they are marked as in-use.

## Motivation

Currently, "a generic mechanism that prevents a resource from being deleted when it is in-use" doesn't exist.
Instead, for such a use case, each controller, like [pv-protection](https://github.com/kubernetes/enhancements/issues/499) and [pvc-protection](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/postpone-pvc-deletion-if-used-in-a-pod.md), implements its own logic to protect used objects from being deleted while in-use.
These controllers use [Finalizers](https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/) to block deletion.
Finalizers block the _completion_ of a delete operation, but they do not prevent the deletion from starting.
Once started, a delete _will_ complete and can not be aborted.

Finalizers may not be appropriate for some resource protection use cases, because they don't prevent other controllers from executing their pre-deletion actions.
The order of execution of pre-deletion actions across finalizers is not defined.
As a result, other controllers will execute their pre-deletion actions while a finalizer for protection still exists.

This KEP aims to provide a generic mechanism for protection that solves the above issues with Finalizers, because such use cases are also found for protecting secrets and configmaps in [KEP-2639](https://github.com/kubernetes/enhancements/pull/2640).

Potential use cases are:
- To guarantee proper order of deletion:
  - Protect `PersistentVolume` while being used by `PersistentVolumeClaim` (Replace the background logic for pv-protection)
  - Protect `PersistentVolumeClaim` while being used by `Pod` (Replace the background logic for pvc-protection)
  - Protect `Secret` while being used by `Pod`, `PersistentVolume`, `VolumeSnapshotContent` (Secret protection above)
  - Protect `ConfigMap` while being used by `Pod`
  - Protect `StorageClass` while being used by `PersistentVolume`
  - Protect `VolumeSnapshot` while being used by `VolumeSnapshotContent`
  - Protect `VolumeSnapshotClass` while being used by `VolumeSnapshotContent`
  - Protect `User`, `Group`, `ServiceAccount` and `Role` from `RoleBinding`
  - Protect `User`, `Group`, `ServiceAccount` and `ClusterRole` from `ClusterRoleBinding`
  - Protect dependent resources while owner resources aren't request to be deleted
- To avoid accidental deletion of important resources:
  - Protect resources that controllers or users marked as important

### Goals

- Provide a generic mechanism that prevents a resource, including CRD, from being deleted when it shouldn't be deleted
- Implement the feature as an advisory feature

### Non-Goals

- Provide a specific mechanism to decide which particular objects should be prevented from being deleted when other particular objects exist
- Provide a generic mechanism that prevents a resource from being updated when it shouldn't be updated
- Implement any of the above potential use cases

## Proposal

A new field `Liens` to mark the object not to be deleted is introduced in object Metadata.
Deletion requests for resources with non-empty `Liens` will be blocked by a newly introduced validation in api-server.

## Glossary

- Lien: An indication that some entity is relying on an API object. Deletion of objects with liens will be prevented until the interested party releases the lien.

### User Stories (Optional)

#### Story 1

Another controller to protect Secret from deletion watches all Secrets and their potential user objects, like `Pod`, `PersistentVolumes`, and `VolumeSnapshotContent`. Once it find the Secret is used by one of the objects, it updates the Secret's `Liens` and ask this mechanism to block deletion request for the secret.

#### Story 2

A user knows this is an important object, and that terraform likes to delete and recreate objects, so the user sets a lien so that an accidental deletion will fail.

<!--
### Notes/Constraints/Caveats (Optional)
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

## Design Details

Most of the design ideas come from [here](https://github.com/kubernetes/kubernetes/issues/10179).
Users or controllers can add a string to `Liens` to ask to protect a resource.
A newly introduced validation in api-server will reject deletion requests for resources with non-empty `Liens` and return ["409 Conflict" error code](https://datatracker.ietf.org/doc/html/rfc2616#section-10.4.10).

`Liens` is defined as a slice of strings, like `Finalizers`.
The strings need to be namespaced keys.
Multiple users or controllers can add their `Liens` for their own purpose, and they can remove their own `Liens` when it is no longer needed.
`Liens` should be added per controller or per user basis.
Deletion requests of the resource are blocked until the last `Liens` for the resource is removed
(Just to be clear, the difference between `Liens` and `Finalizers` is that `Liens` blocks the deletion request itself, while `Finalizers` blocks the deletion to be completed).

A PoC implementation for in-use protection can be found, [here](https://github.com/mkimuram/kubernetes/commits/lien).
Also, how it can be consumed by Secret protection controller can be found, [here](https://github.com/mkimuram/secret-protection/tree/lien).

### API Changes

`Liens` is added to `ObjectMeta`.

```go
Liens []string
```

Validation criteria of the field are as follows:
- Keys must be namespaced (example: `kubernetes.io/secret-protection`;  `foo.example/bar`)
- Maximum length of each key is 253 characters
- Maximum number of keys is 32

### Other Design Considerations

#### Behavior with `ownerReference`

Lien itself only blocks a deletion request to an object that is added to.
However, it may not be clear how lien behaves when used with [`ownerReference`](https://kubernetes.io/docs/concepts/architecture/garbage-collection/#owners-dependents).
Therefore, this section describes the behavior.

In summary, lien will block cascading deletion, but would not block deletion of any dependent resources individually.

For example, let's assume that we have Deployment A which manages Replicaset B which manages Pod C.
In this situation, ownerRefernces are set from Pod C to Replicaset B and from Replicaset B to Deployment A.

If a lien is set to Deployment A, only a deletion request to Deployment A is blocked.
Users can still request to delete Replicaset B and Pod C directly, but they can't request to delete them through Deployment A.

If a lien is set to Replicaset B, only a deletion request to Replicaset B is blocked.
Users can still request to delete Deployment A and Pod C directly.
When deletion of the Deployment A is requested, whether it completes immediately or not depends on the cascading policy and Replicaset B's `blockOwnerDeletion`.
The behaviors are as follows:

- Foreground cascading deletion:
  - `blockOwnerDeletion=true`: Deletion of Deployment A isn't completed until Replicaset B is deleted
  - otherwise: Deployment A is deleted immediately
- Background cascading deletion: Deployment A is deleted immediately

#### Namespace Deletion

It may also not be clear how [namespace deletion](https://kubernetes.io/docs/tasks/administer-cluster/namespaces/#deleting-a-namespace) behaves with lien.
Therefore, this section describes the behavior.

If a lien is set to a namespace, only a deletion request to the namespace is blocked.
Users can still request to delete each object in the namespace directly, but they can't request to delete it through the namespace deletion.

If a lien is set to some of objects in a namespace, a deletion request to the namespace isn't blocked.
This means that the resources with lien in the namespace will eventually be deleted by namespace lifecycle controller.
Users can request to delete its namespace or other objects in the namespace.

In addition, users may expect that the order of resource deletions inside a namespace is guaranteed even when requested through the namespace deletion.
For alpha, resources with liens are deleted in a nondeterministic order on the namespace deletion.
Handling order of deletion is a beta blocker and won't be included in the initial implementation.

#### Block Adding Additional `Liens` while Deleting

`Liens` shouldn't be added after `DeletionTimestamp` is non-nil, which means pre-deletion processes for finalizers are being handled.
Adding `Liens` while deleting itself won't do any harm for the cluster, because the resource will be deleted without any additional deletion requests that will be blocked by `Liens`.
However, users may think that the successful addition of the `Liens` means that the resource won't be deleted until the `Liens` are deleted, which isn't true.
To avoid such a misunderstanding from users, the API server must block any request for adding additional `Liens` to a resource with non-nil `DeletionTimestamp`.

#### Race of removing/adding liens

A cluster-admin would be racing to update the object to remove the lien and delete the object before a namespace editor is able to place the lien back. As a result, a namespace admin or editor can create an object that a cluster-admin or namespace lifecycle controller cannot delete, which shouldn't be allowed.

To mitigate the risk, a new `IgnoreLiens` API option in `DeleteOptions` to force delete a resource with liens will be added.

Specifying the option should only be allowed to a limited set of users and groups, therefore deletion request will be blocked in api-server, if invalid users request deletion with the option.
Restricting the `IgnoreLiens` API option to specific users is beta blocker and won't be included in the initial implementation.
Therefore, in the initial implementation, any users with delete permission will be able to delete resources with liens by specifying the `IgnoreLiens` API option.

#### Unresolved issues

- Decision: When you delete a namespace and an object in that namespace has a lien:
  - Does the namespace and object get deleted anyway?
    - [x] Yes.
  - Does the namespace deletion get blocked (i.e., the delete request fails) if any object in the namespace has a lien?
    - [x] No.
  - Does the other objects in the namespace get deleted, but that not object, preventing the namespace deletion from completing?
    - [x] No. If you delete the namespace, the content will be removed.
  - Does the namespace deletion not proceed until all objects in the namespace are free of liens?
    - [ ] No. If you delete the namespace, the content will be removed. Namespace lifecycle cleanup will not honor liens.
    - [x] No. If you delete the namespace, the content will be removed. Namespace lifecycle cleanup will honor liens only to guarantee the deletion order inside a namespace (guarantee of the deletion order is a beta blocker).
    - [ ] Yes. Namespace lifecycle cleanup will honor liens. If you delete the namespace, it won't be deleted until all the content will be deleted.
- Name:
    - [x] "liens" is short and precise, but an unusual word
    - [ ] Other candidates: InUse, deleteInhibitors, hold, lease, claim, deletionBlockers, protections, guards

Please also see the original comments [here](https://github.com/kubernetes/enhancements/pull/2840#issuecomment-1023774538) and [here](https://github.com/kubernetes/enhancements/pull/2840#issuecomment-1024437024).

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
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

- `staging/src/k8s.io/apiserver/pkg/registry/rest`: `2022/6/13` - `83.3%`
- `staging/src/k8s.io/kubectl/pkg/cmd/delete`: `2022/6/13` - `76.1%`
- `pkg/controller/namespace/deletion`: `2022/6/13` - `68.6%`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- ["test-lien"](https://github.com/kubernetes/kubernetes/blob/master/test/integration/apiserver/apply/apply_crd_test.go): <link to test coverage>
- ["test-namespace-conditions"](https://github.com/kubernetes/kubernetes/blob/master/test/integration/namespace/ns_conditions_test.go#L47): <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- Verify immediate deletion of a secret with empty liens: <link to test coverage>
- Verify that setting liens field is blocked if key is not namespaced: <link to test coverage>
- Verify that setting liens field is blocked if the length of any keys is longer than 253: <link to test coverage>
- Verify that setting liens field is blocked if the number of the keys is more than 32: <link to test coverage>
- Verify that secret with non-empty liens is not removed immediately: <link to test coverage>
- Verify that foreground owner deletion isn't complete while dependent with blockOwnerDeletion=true and lien exists: <link to test coverage>
- Verify that namespace deletion completes while a resource with lien exists in the namespace: <link to test coverage>
- Verify that adding liens to non-nil DeletionTimestamp fails: <link to test coverage>

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Additional tests are in Testgrid and linked in KEP

#### GA

- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

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
- Upgrade:
  - Method: Enable the InUseProtection feature gate
  - Behavior:
    - Setting lien field is allowed unless `DeletionTimestamp` is non-nil
    - Deletion request for resources with non-empty lien is blocked
- Downgrade: 
  - Method: Disable the InUseProtection feature gate
  - Behavior:
    - Setting lien field is blocked
    - Deletion request for resources with non-empty lien isn't blocked

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

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: InUseProtection
  - Components depending on the feature gate: kube-apiserver
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

Deletion requests for an object are blocked while its `Liens` field is non-empty.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, by disabling the feature gates.

###### What happens if we reenable the feature if it was previously rolled back?

Deletion requests for an object are blocked while its `Liens` field is non-empty, again.

###### Are there any tests for feature enablement/disablement?

Tests covering feature enablement/disablement will be added prior Alpha release.

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

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

Not directly, but users or controllers may call more deletion requests to retry.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Describe them, providing:
  - API type(s): `ObjectMeta`
  - Estimated increase in size: a slice of `Liens` of size 8,096B (253 * 32) at most
  - Estimated amount of new objects: a new slice of `Liens` for every existing `ObjectMeta`

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

Deletion won't also happens if the API server and/or etcd are unavailable for all controllers.
Therefore, it won't affect the protection.

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
- Garbage collector continues to try to delete a dependent resource that lien is added
  - Detection:
    - Deletion of an owner of a resource isn't completed.
    - A dependent resource isn't deleted.
  - Mitigations: Delete the lien on the dependent resource.
  - Diagnostics:
    - Log messages: [message](https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/garbagecollector/garbagecollector.go#L475) like below repeatedly logged.
	  `I0908 22:00:47.800913 4063637 garbagecollector.go:475] "Processing object" object="default/nginx-deployment-66b6c48dd5" objectUID=d38e5bc2-1b10-4f08-8e54-6c6c9afbfe3c kind="ReplicaSet" virtual=false`
	- Log Level: 2
  - Testing: E2E test `Verify that foreground owner deletion isn't complete while dependent with blockOwnerDeletion=true and lien exists` covers this failure mode.

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

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->
- Each controller for protecting objects implements its own logic. However, it requires much implementations for the same logic and potentially provides inconsistent user interfaces, like many different finalizers and different ways to opt-out, to users.
- Implement similar mechanism by using finalizers or admission webhook to block deletion

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
