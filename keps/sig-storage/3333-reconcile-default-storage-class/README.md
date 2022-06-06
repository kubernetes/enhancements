# KEP-3333: Retroactive default StorageClass assignment

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
    - [Behavior change](#behavior-change)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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
  - [New annotation](#new-annotation)
  - [Bind to <code>&quot;&quot;</code> PVs at least once](#bind-to--pvs-at-least-once)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

We intend to change behaviour of default storage class assignment to be
retroactive for existing persistent volume claims without any storage class
assigned. This changes the existing Kubernetes behaviour slightly, which is
further described in sections below.

A PVC with `pvc.spec.storageClassName=nil` will **always** get a default storage
class, regardless if the storage class is created before or after the PVC
is created.

## Motivation

When user needs to provision a storage they create a PVC to request a volume. 
A control loop looks for any new PVCs and based on current state of the 
cluster the volume will be provided using one of the following methods:

* Static provisioning - PVC did not specify any storage class and there is
  already an existing PV that can be bound to it. Alternatively users can set
  `pvc.spec.storageClassName=""` to disable dynamic provisioning explicitly.
* Dynamic provisioning - there is no existing PV that could be bound but PVC 
  did specify a storage class or there is exactly one storage class in the 
  cluster marked as default.

Considering the "normal" operation described above there are additional
cases that can be problematic:

1. It’s hard to mark a different SC as the default one. Cluster admin can
   choose between two bad solutions:

    1. Cluster has two default SCs for a short time, i.e. admin marks the new
       default SC as default and then marks the old default SC as non-default. When
       there are two default SCs in a cluster, the PVC admission plugin refuses to
       accept new PVCs with `pvc.spec.storageClassName = nil`. Hence, cluster users
       may get errors when creating PVCs at the wrong time. They must know it’s a
       transient error and manually retry later.

    2. Cluster has no default SC for a short time, i.e. admin marks the old
       default SC as non-default and then marks the new default SC as default.
       Since there is no default SC for some time, PVCs with
       `pvc.spec.storageClassName = nil` created during this time will not get any
       SC and are Pending forever. Users must be smart enough to delete the PVC and
       re-create it.

2. When users want to change the default SC parameters, they must delete the SC
   and re-create it, Kubernetes API does not allow change in the SC. So there is no
   default SC for some time and second case above applies here too.

3. Defined ordering during cluster installation. Kubernetes cluster installation
   tools must be currently smart enough to create a default SC before starting
   anything that may create PVCs that need it. If such a tool supports multiple
   cloud providers, storage backends and addons that require storage (such an image
   registry), it may be quite complicated to do the ordering right.

4. If a default storage class does not exist, PVCs with
   `pvc.spec.storageClassName=nil` are bound to PVs with
   `pv.spec.storageClassName=""`. This can be confusing to users, the PVC is bound
   to different PVs or dynamically provisioned, depending on if the default SC
   exists or not at the time when the PVC is created.

### Goals

* Loosen ordering requirements between PVCs and default SC creation.
* Improve user experience with `pvc.spec.storageClassName=nil`.

### Non-Goals

* Introduce new API for default SCs.

## Proposal

Right now, the default SC is applied to PVCs in an admission plugin, i.e. only
during PVC creation. Later on, PV controller will bind PVCs with
`pvc.spec.storageClassName=nil` to PVs with `pv.spec.storageClassName=""`,
if such a PV exists.

We want to change the existing `pvc.spec.storageClassName=nil` behavior
to always require a default SC. It would never bind to a PV with
`pv.spec.storageClassName=""` and it would be `Pending` until a default SC
is created. When a default SC is created, the PVC `pvc.spec.storageClassName`
will be updated with the new default SC name in the PV controller.

Any PVs with `pv.spec.storageClassName=""` will be able to bind only to a PVC
with `pvc.spec.storageClassName=""`.

This behavior should be simpler from the user perspective.

We plan to re-use the existing `storageclass.kubernetes.io/is-default-class`
SC annotation.

### User Stories (Optional)

#### Story 1

Admin needs to change the default SC from SC1 to SC2
1. The admin marks the current default SC1 as non-default.
2. Another user creates PVC requesting a default SC, by leaving
   `pvc.spec.storageClassName=nil`. The default SC does not exist at
   this point, therefore the admission plugin leaves the PVC untouched with
   `pvc.spec.storageClassName=nil`.
3. The admin marks SC2 as default.
4. PV controller, when reconciling the PVC, updates
   `pvc.spec.storageClassName=nil` to the new SC2.
5. PV controller uses the new SC2 when binding / provisioning the PVC.

#### Story 2

An installation tool wants to install Kubernetes with a CSI driver providing
a default SC and an application that wants to use it (such as image registry).

1. The installer creates PVC for the image registry first, requesting the
   default storage class by leaving `pvc.spec.storageClassName=nil`.
2. The installer creates a default SC.
3. PV controller, when reconciling the PVC, updates
   `pvc.spec.storageClassName=nil` to the new default SC.
4. PV controller uses the new default SC when binding / provisioning the PVC.


#### Story 3 (current behavior)

User wants to provision a volume and there is one default storage class set by
admin.

1. Admin creates a default storage class.
2. Another user creates PVC requesting a default SC, by leaving
   `pvc.spec.storageClassName=nil`. Since the default SC already exists
   the admission plugin changes the `nil` to a name of the default storage 
   class.

### Notes/Constraints/Caveats (Optional)

#### Behavior change

Currently, when a default SC is not available at PVC creation time, a PVC
requesting a default SC (`pvc.spec.storageClassName=nil`) will keep
`pvc.spec.storageClassName=nil` forever. Such PVC can be bound only to PV with
`pvc.spec.storageClassName=""`.

With the new behavior, the PV controller will wait until a default SC exists.
This may break an existing cluster that depends on the behavior described above.
We expect that the new behavior is better than the existing one. Users that
want to bind their PVC to PVs with `pv.spec.storageClassName=""` can create
PVCs explicitly requesting `pvc.spec.storageClassName=""` to provision those
PVs statically.

### Risks and Mitigations

This KEP changes current Kubernetes behavior as discussed above.

Risk: Users depend on existing Kubernetes behavior.

Mitigation:

* Document the behavior change with a release note.

* Suggest users that expect to bind to PVs with `storageClassName=""` to create
  PVCs with `storageClassName=""` and not `storageClassName=nil` in Kubernetes
  docs.

## Design Details

This KEP requires changes in:
* kube-controller-manager / its PV controller:
    * Binding of PVCs with `pvc.spec.storageClassName=nil` must be changed to
      ignore PVs with `pv.spec.storageClassName=""`.
    * `pvc.spec.storageClassName=nil` in PVC must be reconciled to a current
      default SC, if it exists, or after it's created.

* kube-apiserver / PVC update validations:
    * PVC update from  `pvc.spec.storageClassName=nil` to
      `pvc.spec.storageClassName=<storage class name>` is now forbidden, and it
      must be allowed for this KEP to work.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

Kubernetes already has a good coverage of PV/PVC binding tests in
[test/e2e/storage/persistent_volumes.go](https://github.com/kubernetes/kubernetes/blob/ccfac6d3200f63656575819e7b5976c12c3019a6/test/e2e/storage/persistent_volumes.go)

There are no e2e tests that cover behavior of the default SC presence / absence,
they will be added only with the new behavior.

##### Unit tests

All changes should be only in the packages below, which have enough unit test
coverage, we will add only unit tests for the new or changed code.

- `pkg/controller/volume/persistentvolume`: 2022-06-03 - 79%
- `pkg/apis/core/validation/`: 2022-06-03 - 82%

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- test/integration/volume/persistent_volumes_test.go: Has few cases for PV/PVC
  binding.

We plan to extend these test to include default SC and how it will be applied to
existing PVCs. We use integration tests almost like e2e tests of the new
behavior to be sure that change of the default SC won't affect other tests
running in parallel.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

There are no tests that would check how the default SC, because it would need to
create/delete default SCs, which could affect other tests running in parallel
that may expect that a default SC exists (typically StatefulSet e2e tests).

We plan to add a few `[Disruptive]` `[Serial]` tests with the new behavior as
"smoke" tests of the new behavior, still, most of the tests will be integration
ones.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag.
- Initial integration tests completed and enabled.

#### Beta

- Implement and enable e2e tests, visible in Testgrid and linked in KEP.

#### GA

- No users complaining about the new behavior.
- Scalability tests with existing framework (Kubemark?, TBD).
- Allowing time for feedback (at least 2 releases between beta and GA).
- No conformance tests, since we don't test StorageClasses there.
- Manually test version skew between the API server and KCM. See the expected behavior below in Version Skew Strategy.

#### Deprecation

- Announce deprecation and support policy of the existing flag.
- Two versions passed since introducing the functionality that deprecates the
  flag (to address version skew).
- Address feedback on usage/changed behavior, provided on GitHub issues.
- Deprecate the flag.

### Upgrade / Downgrade Strategy

No change in cluster upgrade / downgrade process.

### Version Skew Strategy

This feature is implemented only in the API server and KCM and controlled by
`RetroactiveDefaultStorageClass` feature gate. Following cases may happen:

| API server | KCM | behavior                                                                                                                         |
|------------|-----|----------------------------------------------------------------------------------------------------------------------------------|
| off | off | Existing Kubernetes behavior.                                                                                                    |
| on | off| Existing Kubernetes behavior, only users can change `pvc.spec.storageClassName=nil` to a SC name.                                |
| off | on | PV controller may try to change `pvc.spec.storageClassName=nil` to a new default SC name, which will fail on the API server. (*) |
| on | on | New behavior.                                                                                                                    |

*) For this case, we strongly suggest that the feature is enabled in the API
server first and then in KCM. Similarly, the feature should be disabled in KCM
first and then in the API server. This follows generic Kubernetes version skew
support.

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

- [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name:
    - Components depending on the feature gate:
- [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

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

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

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

### New annotation

Keep the existing behavior of `storageclass.kubernetes.io/is-default-class`
annotation as it is, but add a new SC annotation, say
`storageclass.kubernetes.io/default-is-retroactive`, that  will retroactively
mark all existing PVCs with `storageClassName: nil` to the new default SC.

This way, we don't break existing Kubernetes behavior, trading it for a bigger
complexity on user side - they need to use two annotations instead of one.

We assume that more users will want the new behavior introduced in this KEP
than users that want to keep the existing behavior.

### Bind to `""` PVs at least once

Today, if there is no default SC, a PVC with `pvc.spec.storageClassName=nil`
will be bound to PV with `pv.spec.storageClassName=""`.
To keep at least part of this behavior, PV controller could try to bind such
PVCs to PV at least once and only after that change
`pvc.spec.storageClassName` to a newly created SC.

We find this behavior to be too complicated for users - a PVC with
`pvc.spec.storageClassName=nil`:
* Would be dynamically provisioned by a default
  SC, if the SC exists at the point when PVC is created.
* Or, when the default SC does not exist at the time of PVC creation, then
  it would be bound to existing PV with `pv.spec.storageClassName=""`.
* Or, when such PV does not exist, then provisioned by a newly created default
  SC.

We think that using `pvc.spec.storageClassName=nil` *only* for a default SC,
regardless when the SC or PVC is created, is more robust user experience.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->