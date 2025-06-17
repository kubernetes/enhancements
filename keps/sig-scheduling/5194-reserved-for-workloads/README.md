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
# KEP-5194: DRA ReservedFor Workloads

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
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Higher memory usage by the device_taint_eviction controller](#higher-memory-usage-by-the-device_taint_eviction-controller)
    - [The number of pods that can share a ResourceClaim will not be unlimited](#the-number-of-pods-that-can-share-a-resourceclaim-will-not-be-unlimited)
- [Design Details](#design-details)
  - [Background](#background)
    - [Deallocation](#deallocation)
    - [Finding pods using a ResourceClaim](#finding-pods-using-a-resourceclaim)
  - [Proposal](#proposal-1)
    - [API](#api)
    - [Implementation](#implementation)
      - [Deallocation](#deallocation-1)
      - [Finding pods using a ResourceClaim](#finding-pods-using-a-resourceclaim-1)
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
  - [Increase the size limit on the ReservedFor field](#increase-the-size-limit-on-the-reservedfor-field)
  - [Relax validation without API changes](#relax-validation-without-api-changes)
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

One of the features of Dynamic Resource Allocation is that multiple pods can
share a single ResourceClaim, which means they also share the allocated devices. This
enables several important use-cases. However, currently the number
of pods that can share a single ResourceClaim is limited to 256. We have concrete
use-cases that require that thousands of pods share a single ResourceClaim. With this
KEP, the hard limit on the number of pods will be removed.

## Motivation

Training workloads that uses TPUs can be very large, requiring over 9,000
TPUs for a single training job. The number of TPUs for each node is usually 4, meaning
that the job will run across more than 2,000 nodes. Due to topology constraints, TPU slices
are usually modeled in DRA as multi-host devices, meaning that a single DRA device
can represent thousands of TPUs. As a result, all pods running the workload will
therefore share a single ResourceClaim. The current limit of 256 pods sharing a
ResourceClaim is therefore too low.

### Goals

- Enable ResourceClaims to be shared by any number of pods.

### Non-Goals


## Proposal

Rather than expecting the `ReservedFor` field to contain an exhaustive list of
all pods using the ResourceClaim, we propose letting the controller managing
a ResourceClaim specify the reference to the resource consuming the claim
in the spec. This will then be used as the consumer of the ResourceClaim, removing
the need for every pod in the workload to be listed. 

Increasing the allowed number of pods in the list was considered, but rejected
for two primary reasons:
* The size of AI workloads are getting larger so it is hard to come up with a new
  threshold
* Having a list with thousands of entries is neither a good nor scalable solution.

The `ReservedFor` list already accepts generic resource references, so this
field doesn't need to be changed. However, we are proposing adding two new
fields to the `ResourceClaim` type:
* `spec.ReservedFor` which allows the creator of a `ResourceClaim` to specify in
  the spec which resource is the consumer of the `ResourceClaim`. When the first pod
  referencing the `ResourceClaim` is scheduled, the reference will be copied into
  the `status.ReservedFor` list.
* `status.allocation.ReservedForAnyPod` which will be set to `true` by the DRA
  scheduler plugin at allocation time when the `spec.ReservedFor` field is copied
  into the `status.ReservedFor` list. If `status.allocation.ReservedForAnyPod` is
  set to `true`, the kubelet will skip the check that requires pods to be listed as
  a consumer of the claim when starting the pod.

### Risks and Mitigations

#### Higher memory usage by the device_taint_eviction controller
The device_taint_eviction controller will need to keep an index of which pods are
referenced from each ResourceClaim, so it can evict the correct pods when devices
are tainted. This will require some additional memory.

#### The number of pods that can share a ResourceClaim will not be unlimited
Removing this limit does not mean that the number of pods that can share a ResourceClaim
will be unlimited. As part of the
[scale testing effort for DRA](https://github.com/kubernetes/kubernetes/issues/131198),
we will test the scalability of the number of pods sharing a ResourceClaim so we can
provide guidance as to what is a safe number.

## Design Details

### Background

The `ReservedFor` field on the `ResourceClaimStatus` is currently used for two purposes:

#### Deallocation
Devices are allocated to a `ResourceClaim` when the first pod referencing the claim is
scheduled. Other pods can also share the `ResourceClaim` in which case they share the
devices. Once no pods are consuming the claim, the devices should be deallocated to they
can be allocted to other claims. The `ReservedFor` list is used to keep track of pods
consuming a `ResourceClaim`. Pods are added to the list by the DRA scheduler plugin
during scheduling and removed from the list by the resourceclaim controller when pods are
deleted or finish running. An empty list means there are no current consumers of the claim
and it can be deallocated.

#### Finding pods using a ResourceClaim
It is used by the DRA scheduler plugin, the kubelet, and the device_taint_eviction
controller to find pods that are using the ResourceClaim:

1. The kubelet uses this to make sure it only runs pods that where the claims have been allocated
   to the pod. It can verify this by checking that the Pod is listed in the `ReservedFor` list.

1. The DRA scheduler plugin uses the list to find claims that have zero or only
   a single pod using it, and is therefore a candidate for deallocation in the `PostFilter` function.

1. The device_taint_eviction controller uses the `ReservedFor` list to find the pods that need to be evicted
   when one or more of the devices allocated to a ResourceClaim is tainted (and the ResourceClaim
   does not have a toleration).

So the solution needs to:
* Give the resourceclaim controller a way to know when there are no more consumers of a ResourceClaim so
  it can be deallocated.
* Give controllers a way to list the pods consuming or referencing a ResourceClaim.

### Proposal

#### API

The exact set of proposed API changes can be seen below (`...` is used in places where new fields
are added to existing types):

```go
// ResourceClaimSpec defines what is being requested in a ResourceClaim and how to configure it.
type ResourceClaimSpec struct {
  ...

  // ReservedFor specifies the resource that will be consuming the claim. If set, the
  // reference will be copied into the status.ReservedFor list when the claim is allocated.
  //
  // When this field is set it is the responsibility of the entity that created the
  // ResourceClaim to remove the reference from the status.ReservedFor list when there
  // are no longer any pods consuming the claim.
  //
  // Most user-created ResourceClaims should not set this field. It is more typically
  // used by ResourceClaims created and managed by controllers.
  //
  // +featureGate=DRAReservedForWorkloads
  // +optional
  ReservedFor *ResourceClaimConsumerReference
}

// AllocationResult contains attributes of an allocated resource.
type AllocationResult struct {
  ...

  // ReservedForAnyPod specifies whether the ResourceClaim can be used by
  // any pod referencing it. If set to true, the kubelet will not check whether
  // the pod is listed in the staus.ReservedFor list before running the pod.
  //
  // +featureGate=DRAReservedForWorkloads
  // +optional
  ReservedForAnyPod *bool
}
```

The `ResourceClaimConsumerReference` type already exists:

```go
// ResourceClaimConsumerReference contains enough information to let you
// locate the consumer of a ResourceClaim. The user must be a resource in the same
// namespace as the ResourceClaim.
type ResourceClaimConsumerReference struct {
  // APIGroup is the group for the resource being referenced. It is
  // empty for the core API. This matches the group in the APIVersion
  // that is used when creating the resources.
  // +optional
  APIGroup string
  // Resource is the type of resource being referenced, for example "pods".
  // +required
  Resource string
  // Name is the name of resource being referenced.
  // +required
  Name string
  // UID identifies exactly one incarnation of the resource.
  // +required
  UID types.UID
}
```

#### Implementation
Whenever the scheduler (i.e. the DRA scheduler plugin) tries to schedule a pod that
references a `ResourceClaim` with an empty `status.ReservedFor` list, it knows that this
is the first pod that will be consuming the claim.

If the `spec.ReservedFor` field in the ResourceClaim is not set, the scheduler will handle
the `ResourceClaim` in the same way as now, and will add the `Pod` to the `ReservedFor` list
if devices could be allocated for the claim. Any additional pods that reference the `ResourceClaim`
will also be added to the list.

If the `spec.ReservedFor` field is set, the scheduler will copy this reference to the
`ReservedFor` list, rather than adding a reference to the `Pod`. It will also update the
`status.allocation.ReservedForAnyPod` field to `true`. When any other pods referencing
the `ResourceClaim` is scheduled and the scheduler sees a non-Pod reference in the `ReservedFor`
list, it will not add a reference to the pod.

##### Deallocation
The resourceclaim controller will remove Pod references from the `ReservedFor` list just
like it does now using the same logic. But for non-Pod references, it
will be the responsibility of the controller/user that created the `ResourceClaim` to
remove the reference to the non-Pod resource from the `ReservedFor` list when no pods
are consuming the `ResourceClaim` and no new pods will be created that references
the `ResourceClaim`.

The resourceclaim controller will then discover that the `ReservedFor` list is empty
and therefore know that it is safe to deallocate the `ResourceClaim`.

This requires that the controller/user has permissions to update the status
subresource of the `ResourceClaim`. The resourceclaim controller will also try to detect if
the resource referenced in the `ReservedFor` list has been deleted from the cluster, but
that requires that the controller has permissions to get or list resources of the type. If the
resourceclaim controller is not able to check, it will just wait until the reference in
the `ReservedFor` list is removed. The resourceclaim controller will not have a watch
on the workload resource, so there is no guarantee that the controller will realize that
the resource has been deleted. This is an extra check since it is the responsibility of
the workload controller to update the claim.

##### Finding pods using a ResourceClaim
If the reference in the `ReservedFor` list is to a non-Pod resource, controllers can no longer
use the list to find all pods consuming the `ResourceClaim`. Instead they will look up all
pods referencing the `ResourceClaim`, which can be done by using a watch on Pods and maintaining
an index of `ResourceClaim` to pods referencing it. This can be done using the informer cache.

The list of pods referencing a `ResourceClaim` is not exactly the same as the list of pods
consuming a `ResourceClaim` as specified in the `ReservedFor` list. References to pods in the
`ReservedFor` list only contains pods that have been processed by the DRA scheduler plugin and
is scheduled to use the `ResourceClaim`. It is possible to have pods that reference `ResourceClaim`,
but haven't yet been scheduled. This distinction is important for some of the usages of the
`ReservedFor` list described above:

1. If the kubelet sees that the `status.allocation.ReservedForAnyPod` is set, it will skip
   the check that the Pod is listed in the `ReservedFor` list and just run the pod.

1. If the DRA scheduler plugin is trying to find candidates for deallocation in
   the `PostFilter` function and sees a `ResourceClaim` with a non-Pod reference, it will not
   attempt to deallocate. The plugin has no way to know how many pods are actually consuming
   the `ResourceClaim` without the explicit list in the `ReservedFor` list and therefore it will
   not be safe to deallocate.

1. The device_taint_eviction controller will use the list of pods referencing the `ResourceClaim`
   to determine the list of pods that needs to be evicted. In this situation, it is ok if the
   list includes pods that haven't yet been scheduled.


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

None

##### Unit tests

<!--
Generated with:

go test -cover ./pkg/scheduler/framework/plugins/dynamicresources ./pkg/controller/resourceclaim ./pkg/controller/devicetainteviction ./pkg/kubelet/cm/dra | sed -e 's/.*\(k8s.io[a-z/-]*\).*coverage: \(.*\) of statements/- `\1`: \2/' | sort

-->


- `k8s.io/kubernetes/pkg/controller/devicetainteviction`: `06/05/2025` - 89.9%
- `k8s.io/kubernetes/pkg/controller/resourceclaim`: `06/05/2025` - 74.2%
- `k8s.io/kubernetes/pkg/kubelet/cm/dra`: `06/05/2025` - 79.4%
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/dynamicresources`: `06/05/2025` - 79.3%

##### Integration tests

Scheduler perf tests will be added to assess the performance impact of this change.

##### e2e tests

Additional e2e tests will be added to verify the behavior added in this KEP.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Additional tests are in Testgrid and linked in KEP
- Performance impact of the feature has been measured and found to be acceptable
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved
- Revisit whether the responsibility of removing the workload resource reference from
  the `ReservedFor` list should be with the workload controller (as proposed in this design)
  or be handled by the resourceclaim controller.

#### GA

- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved
- [conformance tests]

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md


### Upgrade / Downgrade Strategy

The feature will no longer work if downgrading to a release without support for it.
The API server will no longer accept the new fields and the other components will
not know what to do with them. So the result is that the `ReservedFor` list will only
have references to pod resources like today.

Any ResourceClaims that have already been allocated when the feature was active will
have non-pod references in the `ReservedFor` list after a downgrade, but the controllers
will not know how to handle it. There are two problems that will arise as a result of
this:
- The workload controller will also have been downgraded if it is in-tree, meaning that
  it will not remove the reference to workload resource from the `ReservedFor` list, thus
  leading to a situation where the claim will never be deallocated.
- For new pods that gets scheduled, the scheduler will add pod references in the
  `ReservedFor` list, despite there being a non-pod reference here. So it ends up with
  both pod and non-pod references in the list. We can manage both pod and non-pod
  references in the list by letting the workload controllers add the non-pod reference
  even if it sees pod references and making sure that the resourceclaim controller removes
  pod references even if there are non-pod references in the list. For deallocation, it is
  only safe when no pods are consuming the claim, so both workload and pod reference should
  be removed once that is true.

We will also provide explicit recommendations for how users can manage downgrades or
disabling this feature. This means manually updating the references in the `ReservedFor` list
to be pods rather than the reference to workload resources. We don't plan on providing
automation for this.

### Version Skew Strategy

If the kubelet is on a version that doesn't support the feature but the rest of the
components are, workloads will be scheduled, but the kubelet will refuse to run it
since it will still check whether the `Pod` is references in the `ReservedFor` list.

If the API server is on a version that supports the feature, but the scheduler
is not, the scheduler will not know about the new fields added, so it will
put the reference to the `Pod` in the `ReservedFor` list rather than the reference
in the `spec.ReservedFor` list. It will do this even if there is already a non-pod
reference in the `spec.ReservedFor` list. This leads to the challenge described
in the previous section.

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
  - Feature gate name: DRAReservedForWorkloads
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-scheduler
    - kube-controller-manager
    - kubelet

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Applications that were already running will continue to run. But if a pod have to be
re-admitted by a kubelet where the feature has been disabled, it will not be able to, since
the kubelet will not find a reference to the pod in the `ReservedFor` list.

The feature will also be disabled for in-tree workload controllers, meaning that they will
not remove the reference to the workload resource from the `ReservedFor` list. This means the list will never
be empty and the resourceclaim controller will never deallocate the claim.

###### What happens if we reenable the feature if it was previously rolled back?

It will take affect again and will impact how the `ReservedFor` field is used during allocation
and deallocation. Since this scenario allows a ResourceClaim with the `spec.ReservedFor` field
to be set and then have the scheduler populate the `ReservedFor` list with pods when the feature
is disabled, we will end up in a situation where the `ReservedFor` list can contain both non-pod
and pod references. We need to make sure all components can handle that.

###### Are there any tests for feature enablement/disablement?

This will be covered through unit tests for the apiserver, scheduler, resourceclaim controller and
kubelet.

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

It does require that:
- The device_taint_eviction controller watches Pods 

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes and no. We are adding two new fields to the ResourceClaim type, but neither are of a collection type
so they should have limited impact on the total size of the objects. However, this feature means that
we no longer need to keep a complete list of all pods using a ResourceClaim, which can significantly
reduce the size of ResourceClaim objects shared by many pods.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

It might require some additional memory usage in the resourceclaim controller since it will need to keep an index
of ResourceClaim to Pods.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

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

- 1.34: first KEP revision

## Drawbacks

This complicates the allocation and deallocation logic somewhat as there will be two
separate ways to manage the allocation and deallocation process for ResourceClaims.

It also leads to additional work for the device_taint_eviction controller since it needs
to maintain an index to find all pods using a ResourceClaim rather than just looking at
the list of pods in the `ReservedFor` list.

## Alternatives

### Increase the size limit on the ReservedFor field
The simplest solution here would be to just increase the size limit on the
`ReservedFor` field to a larger number. But having a large list of pod references
is not a good way to handle it and could at least in theory run into the size limit
of Kubernetes resources. Also, we would need to have some limit on the size, and whatever
number we choose it might still be too small for the largest workloads.

### Relax validation without API changes
The current proposal adds explicit support for non-pod references in the `ReservedFor` list
by adding the new `spec. An alternative is to let the workload controller be responsible for
not only removing the reference in the `ReservedFor` list when there are no longer any pods
consuming the `ResourceClaim`, but also adding the reference after creating the `ResourceClaim`.
This will require that the validation is relaxed to allow entries in the `ReservedFor` list
without any allocation. This would also require that the Kubelet checks for non-Pod references
in the `ReservedFor` list and skips the check before running pods if it finds any.

This isn't all that different than the proposed solution, but the solution described above
was considered superior as it makes the new feature more explicit.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
