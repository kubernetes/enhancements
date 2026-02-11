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
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes adding a new `UnusedSince` status field on a `PersistentVolumeClaim`
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

- Add an `UnusedSince` timestamp field to `PersistentVolumeClaimStatus`
- Update this field with a timestamp when a PVC is last used by a Pod

### Non-Goals

- Automatically deleting unused PVCs (this is a decision for cluster administrators)
- Tracking which specific Pod last used the PVC (only tracking when, not who)
- Providing PVC usage recommendations or alerts

## Proposal

Add a new field `UnusedSince` of type `metav1.Time` to the `PersistentVolumeClaimStatus`
struct. This field will be updated by the PVC protection controller (in kube-controller-manager)
when the PVC transitions from being in use to not being in use.

The definition of a PVC not being in use is when the last Pod referencing it has
been deleted or reached a terminal state.

### User Stories (Optional)

#### Story 1: Cluster Administrator

As a cluster administrator, I want to identify PVCs that have not been used
in the last X days so that I can review them for potential deletion and
reduce storage costs.

### Notes/Constraints/Caveats (Optional)

Notes:

- Definition of 'last used': For the purposes of this KEP, a PVC is considered
   'last used' when the last Pod referencing it has been deleted/terminated. Since
   multiple Pods can share the same PVC, it's important to update the last used
   status when the last Pod referencing it goes down.
- Granularity: The timestamp represents the last time the PVC was referenced
   by a Pod, not continuous usage tracking. A PVC re-mounted by a long-running Pod
   will clear the timestamp to `nil`. If `UnusedSince` is not-nil, that'd mean the
   PVC is not in use. If it is `nil`, it'd either mean it's currently in use, or
   that it has never completed a usage cycle.

### Risks and Mitigations

One risk is API server churn on KCM startup. When the feature is enabled (which
leads to KCM restarts), the PVC protection controller will try to re-process all
PVCs (one PVC at a time, from the queue) to ensure the timestamps are accurate,
including any changes that were missed while offline.

In some clusters with many PVCs, this may cause the controller being throttled,
which may lead to slight delays in updating the timestamp values on the PVCs (which
is another risk too - see last point). This is an expected behavior for now and should
be documented to convey the risks to the users.

Another risk/point of confusion is when the feature is disabled, existing `UnusedSince`
values remain in etcd, but are no longer updated, potentially becoming misleading. A
similar mitigation approach could be adopted - to document about the fact that
disabling the feature freezes existing values, and that administrators should not rely
on the field while the feature is disabled.

One more risk is that the timestamp value may not be entirely accurate. The `UnusedSince`
timestamp represents when the controller observed no Pods referencing the PVC. It does
not represent when the volume was actually unmounted at the infrastructure level and
became actually unused (which could be delay of seconds or minutes). The only component
that knows the longest time known to Kubernetes since volume was not used by a Pod is
the kubelet, when it does the last unmount - but we'd not like our kubelet to update PVCs.
The reported unused duration may be shorter than actual, but should never be longer.
Mitigation approach here would be to document this information clearly.

## Design Details

Changes required for this KEP:

1. Add a new field to `PersistentVolumeClaimStatus` in `core/v1` to track the timestamp:

   ```go
   type PersistentVolumeClaimStatus struct {
     // existing fields...

     // UnusedSince is the timestamp that represents when the PVC last transitioned
     // to not being in use. When the PVC is currently in use, this field is nil.
     // It is updated when the last Pod referencing this PVC is deleted or reaches a
     // terminal state, and cleared when a new Pod starts referencing the PVC.
     // +optional
     UnusedSince *metav1.Time `json:"unusedSince,omitempty" protobuf:"bytes,10,opt,name=UnusedSince"`
   }
   ```

1. Update the timestamp whenever the PVC transitions to not being in use anymore.

   - The PVC Protection controller already watches Pod events and checks when a
      PVC transitions from "in use" to "not in use".
   - The implementation can extend this existing logic:
      - When a Pod is deleted, the controller enqueues all the affected PVCs
      - During sync, the controller checks if the PVC is still in use by a Pod
      - Existing behavior: If it's not in use, and the `deletionTimestamp` is set,
         it proceeds with finalizer removal. If `deletionTimestamp` is not set, it returns early.
      - New behavior: When it determines a PVC is not in use, and the `deletionTimestamp`
         is not set (not queued for deletion), it should update the `status.UnusedSince`
         timestamp to the current timestamp. (Note: If `deletionTimestamp` is set, we skip
         updating `UnusedSince` since the PVC is being deleted and the timestamp
         would serve no purpose)
         Conversely, when the PVC transitions from not in use to in use (i.e., a Pod starts
         referencing it) the controller should clear `UnusedSince` to `nil`. This ensures
         the field doesn't reflect stale values when it's currently in use.

1. Add a Feature Gate named `PersistentVolumeClaimUnusedSinceTime` that is disabled by default in alpha.
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

- `pkg/controller/volume/pvcprotection`: tests for `UnusedSince` being set when
the last Pod referencing a PVC terminates, and not set when other Pods still
reference the PVC.

- `pkg/controller/volume/pvcprotection`: `2026-01-18` - `74.1%`

##### Integration tests

Integration tests can be added to verify the core controller logic:

- `UnusedSince` set when last Pod referencing PVC terminates
- `UnusedSince` set to `nil` when other Pods still reference the PVC
- Feature gate enable/disable behavior

Note: Integration and e2e tests would be pretty identical for this feature.
We can possibly skip this using the reasoning that e2e tests would provide
more value and might help catch more bugs. \[TBD\]

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

##### e2e tests

e2e tests will be added to verify the feature works in a real cluster environment.

- `UnusedSince` set when last Pod referencing PVC terminates
- `UnusedSince` set to `nil` when other Pods still reference the PVC
- Feature gate enable/disable behavior

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

Upgrade:
For pre-existing PVCs: After upgrading, the `UnusedSince` status field will not be
present on the PVCs until it transitions from being in use to not in use. PVCs that
are never used after upgrade won't have this field.

For new PVCs: PVCs that are created after upgrading, will be created with `UnusedSince`
set to `nil`. The value of the field will be populated on first transition from in use
to not in use. PVCs that are never used after creation will retain the `nil` value.

Downgrade:
When downgrading to a version without this feature, the field value (if set) will
be preserved in etcd. Older controller-managers would simply ignore this field.
The field value might go stale if transition happens during the downgraded versions
but the updating process resumes when the version is upgraded and the first transition
occurs.

### Version Skew Strategy

| API Server | KCM | Behavior |
| :--- | :--- | :--- |
| off | off | Existing Kubernetes behavior. |
| on | off | Existing Kubernetes behavior. The `UnusedSince` field exists but is never populated by the controller. Only users can set it manually via the API. |
| off | on | PVC protection controller may attempt to set `UnusedSince`, which will be dropped by the API server since the field is not recognized. |
| on | on | New behavior. `UnusedSince` is updated when a PVC transitions from in use to not in use, and cleared when it transitions to in use. |

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `PersistentVolumeClaimUnusedSinceTime`
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-controller-manager

###### Does enabling the feature change any default behavior?

No. The field is not read by any other Kubernetes component for any purposes
and so, existing workflows that do not explicitly read this field would remain
unaffected. The field also is not available on existing PVCs after enabling the
feature, until it transitions from being in use to not in use, which is when
the value is populated with the timestamp.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, disabling the feature will stop the controller from updating the status field.
The value however, once set, would remain stored in etcd, but become stale - disabling
doesn't remove it. If the field was set to nil (either never set, or currently in use),
it remains nil and cannot be newly populated while the feature gate is disabled.

###### What happens if we reenable the feature if it was previously rolled back?

The controller will resume updating the status field. If the PVC didn't transition
while the feature was disabled, the data already stored represents the correct last used
timestamp. If the PVC did transition while the feature was disabled, the data might either
be stale or missing. There could be 2 scenarios:

- If a PVC transitioned from not in use to in use, the old timestamp is retained
  instead of being cleared to nil, thus becoming stale.
- If a PVC transitioned from in use to not in use, the value remains nil instead
  of being set to a timestamp

In either case, the value will be corrected on the next transition after the
feature is re-enabled.

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

Operators can check if the PVCs have `status.UnusedSince` field populated.

###### How can someone using this feature know that it is working for their instance?

- [X] API .status
  - Other field: `pvc.Status.UnusedSince`

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

No, depends only upon the core Kubernetes components available to a functioning
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

Yes. All PVC objects will have an entirely new status field `UnusedSince` to hold
the timestamp value. Estimated increase in size would be < 50B.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No, this feature operates in the control-plane and doesn't affect node resources.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The controller will be unable to update the `UnusedSince` status field.

###### What are other known failure modes?

One failure mode would be the delay in updating the timestamp values for clusters
having a large number of PVCs (this has been discussed in the Risks and Mitigations
section). This might sometimes lead to timestamp values not being entirely accurate.
In such cases however, the time reported can be shorter than actual time the PVC was
unused, but it should never be longer.

This feature extends the PVC Protection controller logic with an additional
status field update. Other failure modes would be similar to existing failure modes
of the controller. If KCM is unavailable, timestamps won't be updated. If the API
server and/or etcd is unavailable, the timestamps won't be updated (covered in the
section above).

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
