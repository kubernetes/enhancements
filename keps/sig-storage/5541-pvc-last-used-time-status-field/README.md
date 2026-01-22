# KEP-5541: Report Last Used Time on a PVC

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: Cluster Administrator](#story-1-cluster-administrator)
    - [Story 2: Developer](#story-2-developer)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
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

Items marked with (R) are required _prior to targeting to a milestone / release_.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes adding a new `lastUsed` status field on a `PersistentVolumeClaim`
to determine when a PVC was last used by a Pod. This enables cluster
administrators to identify unused PVCs and implement cleanup policies for
storage that is no longer in use.

## Motivation

PersistentVolumeClaims can accumulate over time in a cluster. When applications
are deleted or migrated, their associated PVCs may be left behind, consuming storage
resources and incurring costs. It'd be helpful for cluster admins to identify how
long a PVC has been sitting unused, to then clean up unutilized storage. Currently,
Kubernetes provides no mechanism to determine when a PVC was last actively used by a workload.

Only Kubernetes has accurate knowledge of when a PVC is mounted by a Pod, making
this the ideal place to track usage.

### Goals

- Add a `lastUsedTime` timestamp field to `PersistentVolumeClaimStatus` (the naming is
   yet to be decided)
- Update this field with a timestamp when a PVC is last used by a Pod

### Non-Goals

- Automatically deleting unused PVCs (this is a decision for cluster administrators)
- Tracking which specific Pod last used the PVC (only tracking when, not who)
- Providing PVC usage recommendations or alerts

## Proposal

Add a new field `lastUsedTime` of type `metav1.Time` to the `PersistentVolumeClaimStatus`
struct. This field will be updated by the PVC protection controller (in kube-controller-manager)
when the PVC transitions from being in use to not being in use.

The definition of a PVC not being in use is when the last Pod referencing it has
been deleted or reached a terminal state.

### User Stories (Optional)

#### Story 1: Cluster Administrator

As a cluster administrator, I want to identify PVCs that have not been used
in the last X days so that I can review them for potential deletion and
reduce storage costs.

#### Story 2: Developer

As a developer, I want to see when my PVCs were last used so that I can
clean up test resources I've forgotten about.

### Notes/Constraints/Caveats (Optional)

Notes:

- Definition of 'last used': For the purposes of this KEP, a PVC is considered
   'last used' when the last Pod referencing it has been deleted/terminated. Since
   multiple Pods can share the same PVC, its important to update the last used
   status when the last Pod referencing it goes down.
- Granularity: The timestamp represents the last time the PVC was referenced
   by a Pod, not continuous usage tracking. A PVC re-mounted by a long-running Pod
   will show the time it was last used, and the current timestamp.

Caveats:

One caveat of this proposal is that admins won't see any immediate changes after
this feature has been enabled/disabled. This is because of how and when the new
status field needs to be added/removed.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

TBD

## Design Details

Changes required for this KEP:

1. Add a new field to `PersistentVolumeClaimStatus` in `core/v1` to track the timestamp:

   ```go
   type PersistentVolumeClaimStatus struct {
     // existing fields...

     // LastUsedTime is the timestamp that represents when the PVC last transitioned
     // to not being in use.
     // It is updated when the last Pod referencing this PVC is deleted or reaches a
     // terminal state.
     // +optional
     LastUsedTime *metav1.Time `json:"lastUsedTime,omitempty" protobuf:"bytes,10,opt,name=lastUsedTime"`
   }
   ```

1. Update the timestamp whenever the PVC transitions to not being in use anymore.

   - The PVC Protection controller already watched Pod events and checks when a
      PVC transitions from "in use" to "not in use".
   - The implementation can extend this existing logic:
      - When a Pod is deleted, the controller enqueues all the affected PVCs
      - During sync, the controller checks if the PVC is still in use by an Pod
      - Existing behavior: If it's not in use, and the `deletionTimestamp` is set,
         it proceeds with finalizer removal. If `deletionTimestamp` is not set, it returns early.
      - New behavior: When it determines a PVC is not in use, and the `deletionTimestamp`
         is not set (not queued for deletion), it should update the `status.LastUsedTime`
         timestamp to the current timestamp. (Note: If `deletionTimestamp` is set, we skip
         updating `lastUsedTime` since the PVC is being deleted and the timestamp
         would serve no purpose)

1. Add a Feature Gate named `PVCLastUsedTime` that is disabled by default in alpha.
   The timestamp field exists but is not populated unless gate is enabled.

Note (definition of in use): A PVC is considered to be in use if a Pod references
it in `pod.spec.volumes` and that Pod is not in a terminal state (succeeded/failed).

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

The following packages will be modified and require test coverage:

- `pkg/controller/volume/pvcprotection`: tests for `lastUsedTime` being set when
the last Pod referencing a PVC terminates, and not set when other Pods still
reference the PVC.

- `pkg/controller/volume/pvcprotection`: `2026-01-18` - `74.1%`

##### Integration tests

Integration tests can be added to verify the core controller logic:

- `lastUsedTime` set when last Pod referencing PVC terminates
- `lastUsedTime` not set when other Pods still reference the PVC
- Feature gate enable/disable behavior

Note: Integration and e2e tests would be pretty identical for this feature.
We can possibly skip this using the reasoning that e2e tests would provide
more value and might help catch more bugs. \[TBD\]

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

##### e2e tests

e2e tests will be added to verify the feature works in a real cluster environment.

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

### Graduation Criteria

#### Alpha

- Feature implemented successfully behind a feature gate
- Unit tests added to test out feature enablement/disablement, and passing
- Initial e2e tests completed and enabled

#### Beta

TBD

#### GA

TBD

### Upgrade / Downgrade Strategy

Upgrading and downgrading is safe.

When upgrading, the new status field will be added to all PVC objects, and will
be updated appropriately next time when the PVC transitions from being in use to
not in use. PVCs that are never used after upgrade will retain the initial `nil`
value for this field.

When downgrading to a version without this feature, the field value (if set) will
be preserved in etcd. Older controller-managers would simply ignore this field.
The field value might go stale if transition happens during the downgraded versions
but the updating process resumes when the version is upgraded and the first transition
occurs.

### Version Skew Strategy

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

TBD

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `PVCLastUsedTime`
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-controller-manager

###### Does enabling the feature change any default behavior?

Yes, all PVCs will start to contain the new `LastUsedTime` status field. This field
however is only updated when a PVC transitions from being in use to not in use. The field
is not read by any other Kubernetes component for any purposes and so, existing
workflows that do not explicitly read this field would remain unaffected.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, disabling the feature will stop the controller from updating the status field.
The value however, once set, would remain stored in etcd, but become stale - disabling
doesn't remove it.

###### What happens if we reenable the feature if it was previously rolled back?

The controller will resume updating the status field. If the PVC didn't transition
while the feature was disabled, the data already stored represents the correct last used
timestamp. If the PVC did transition to not being in used, the stale timestamp is still
retained and would be updated when the PVCs are used again and then released.

###### Are there any tests for feature enablement/disablement?

Unit tests for enabling and disabling feature gate are required for alpha - see "Graduation criteria" section.

The tests should verify the correct handling of the new PVC status field in relation to the
feature gate state. Correct handling means the value of the newly added status field is
correctly added/updated when the PVC transitions to not being in use, while the feature
gate is enabled, and the values are persisted when the feature gate is disabled.

### Rollout, Upgrade and Rollback Planning

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
Yet to be completed for beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Operators can check if the PVCs have `status.LastUsedTime` field populated.

###### How can someone using this feature know that it is working for their instance?

- [X] API .status
  - Other field: `pvc.status.lastUsedTime`

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

###### Does this feature depend on any specific services running in the cluster?

No, depends only upon the core Kubrenetes components available to a functioning
cluster.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes. A new PATCH call for PersistentVolumeClaimStatus. Estimated throughput would be
one PATCH call per PVC when transitioning from in use to not in use. The originating
component is PVC Protection Controller (in kube-controller-manager).

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes. All PVC objects will have an entirely new status field `lastUsedTime` to hold
the timestamp value. Estimated increase in size would be < 50B.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No, this feature operates in the control-plane and doesn't affect node resources.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The controller will be unable to update the `lastUsedTime` status field.

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

Users should check kube-controller-manager logs for errors related to
PVC status updates.

## Implementation History

- 1.36: alpha

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

One alternative was the use the deletion of `VolumeAttachment` objects as triggers
for updating the status field. This was ruled out because of the fact that not all
volume types create a `VolumeAttachment` object, restricting the scope of the KEP.

## Infrastructure Needed (Optional)

None.
