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
  - [x] e2e Tests for all Beta API Operations (endpoints)
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

This KEP proposes adding a new `Unused` condition on `PersistentVolumeClaimStatus`
to determine whether a PVC is currently in use by a Pod. This enables cluster
administrators to identify unused PVCs and implement cleanup policies for
storage that is no longer in use. The `lastTransitionTime` of the condition
indicates when the PVC last changed between used and unused states.

## Motivation

PersistentVolumeClaims can accumulate over time in a cluster. When applications
are deleted or migrated, their associated PVCs may be left behind, consuming storage
resources and incurring costs. It'd be helpful for cluster admins to identify how
long a PVC has been sitting unused, to then clean up unutilized storage. Currently,
Kubernetes provides no mechanism to determine when a PVC was last actively used by a workload.

Only Kubernetes has accurate knowledge of when a PVC is mounted by a Pod, making
this the ideal place to track usage.

### Goals

- Add an `Unused` condition type to `PersistentVolumeClaim` status conditions
- Set the condition to reflect whether the PVC is currently in use by any non-terminal Pod

### Non-Goals

- Automatically deleting unused PVCs (this is a decision for cluster administrators)
- Tracking which specific Pod last used the PVC (only tracking when, not who)
- Providing PVC usage recommendations or alerts

## Proposal

Add a new condition type `Unused` to the `PersistentVolumeClaim` status conditions.
This condition is managed by the PVC protection controller (in kube-controller-manager)
and reflects whether the PVC is currently referenced by any non-terminal Pod.

When the last Pod referencing a PVC is deleted or reaches a terminal state, the
condition is set to `Status=True` with `Reason="NoPodsUsingPVC"` and
`Message="No pods are currently referencing this PVC"`. When a new Pod starts
referencing the PVC, the condition is set to `Status=False` with
`Reason="PodUsingPVC"` and `Message="A pod is currently referencing this PVC"`.

The `lastTransitionTime` of the condition indicates when the PVC last changed
between used and unused states, providing the equivalent of an "unused since"
timestamp.

### User Stories (Optional)

#### Story 1: Cluster Administrator

As a cluster administrator, I want to identify PVCs that have not been used
in the last X days so that I can review them for potential deletion and
reduce storage costs.

### Notes/Constraints/Caveats (Optional)

Notes:

- Definition of 'unused': A PVC is considered unused when no non-terminal Pod
   references it. The `Unused` condition reflects this state:
   - `Unused` condition with `Status=True` means the PVC is not referenced by
     any non-terminal Pod.
   - `Unused` condition with `Status=False` means at least one non-terminal Pod
     references the PVC.
   - No `Unused` condition present means the feature was recently enabled and no
     transition has been observed yet, or the PVC has not yet gone through a
     usage cycle.
- Granularity: The `lastTransitionTime` on the condition indicates when the
   PVC last changed between used and unused states. A PVC referenced by a
   long-running Pod will have the condition set to `Status=False`. When that
   Pod terminates, the condition transitions to `Status=True` and
   `lastTransitionTime` is updated accordingly.

### Risks and Mitigations

One risk is API server churn on KCM startup. When the feature is enabled (which
leads to KCM restarts), the PVC protection controller will try to re-process all
PVCs (one PVC at a time, from the queue) to ensure the conditions are accurate,
including any changes that were missed while offline.

In some clusters with many PVCs, this may cause the controller being throttled,
which may lead to slight delays in updating the condition on PVCs (which
is another risk too - see last point). This is an expected behavior for now and should
be documented to convey the risks to the users.

Another risk/point of confusion is when the feature is disabled, existing `Unused`
condition values remain in etcd, but are no longer updated, potentially becoming
misleading. A similar mitigation approach could be adopted - to document about the
fact that disabling the feature freezes existing condition values, and that
administrators should not rely on the condition while the feature is disabled.

One more risk is that the condition transition time may not be entirely accurate.
The `lastTransitionTime` on the `Unused` condition represents when the controller
observed no Pods referencing the PVC. It does not represent when the volume was
actually unmounted at the infrastructure level and became actually unused (which
could be delay of seconds or minutes). The only component that knows the longest
time known to Kubernetes since volume was not used by a Pod is the kubelet, when
it does the last unmount - but we'd not like our kubelet to update PVCs. The
reported unused duration may be shorter than actual, but should never be longer.
Mitigation approach here would be to document this information clearly.

## Design Details

Changes required for this KEP:

1. Add a new condition type constant to PVC condition types in `core/v1`:

   ```go
   // In staging/src/k8s.io/api/core/v1/types.go
   PersistentVolumeClaimUnused PersistentVolumeClaimConditionType = "Unused"
   ```

   The condition uses the following states:

   | Status | Reason | Message |
   | :--- | :--- | :--- |
   | `True` | `NoPodsUsingPVC` | `No pods are currently referencing this PVC` |
   | `False` | `PodUsingPVC` | `A pod is currently referencing this PVC` |

1. In the PVC protection controller, add logic to set/update the `Unused` condition:

   - The PVC Protection controller already watches Pod events and checks when a
      PVC transitions from "in use" to "not in use".
   - The implementation extends this existing logic:
      - When a Pod is deleted, the controller enqueues all the affected PVCs
      - During sync, the controller checks if the PVC is still in use by a Pod
      - Existing behavior: If it's not in use, and the `deletionTimestamp` is set,
         it proceeds with finalizer removal. If `deletionTimestamp` is not set, it returns early.
      - New behavior: When it determines a PVC is not in use, and the `deletionTimestamp`
         is not set (not queued for deletion), it sets the `Unused` condition to
         `Status=True`. (Note: If `deletionTimestamp` is set, we skip updating the
         condition since the PVC is being deleted and the condition would serve no purpose.)
         Conversely, when the PVC transitions from not in use to in use (i.e., a Pod starts
         referencing it) the controller sets the `Unused` condition to `Status=False`.
   - The controller uses two separate check functions: `podUsesPVCForDeletion`
      (existing, for finalizer logic) and `podUsesPVCForUnusedSince` (new, considers
      unscheduled pods too).

1. Add a Feature Gate named `PersistentVolumeClaimUnusedSinceTime` that is disabled
   by default in alpha and enabled by default in beta. The condition is not managed
   unless the gate is enabled.

Note (definition of in use): A PVC is considered to be in use if a Pod references
it in `pod.spec.volumes` and that Pod is not in a terminal state (succeeded/failed).

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

The following packages will be modified and require test coverage:

- `pkg/controller/volume/pvcprotection`: tests for the `Unused` condition being
set to `Status=True` when the last Pod referencing a PVC terminates, and set to
`Status=False` when a Pod references the PVC.

- [`pkg/controller/volume/pvcprotection`](https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/volume/pvcprotection/pvc_protection_controller_test.go): `2026-01-18` - `74.1%`

##### Integration tests

Integration and e2e tests would be pretty identical for this feature.
We can possibly skip this using the reasoning that e2e tests would provide
more value.

##### e2e tests

e2e tests will be added to verify the feature works in a real cluster environment:

- `Unused` condition set to `Status=True` when last Pod referencing PVC terminates
- `Unused` condition set to `Status=False` when a Pod references the PVC
- Feature gate enable/disable behavior

### Graduation Criteria

#### Alpha

- Feature implemented successfully behind a feature gate
- Unit tests added to test out feature enablement/disablement, and passing

#### Beta

- Feature implemented and stable in alpha for one release
- Initial e2e tests completed and enabled
- Comprehensive unit test coverage
- E2E tests added and passing
- PRR completed and approved

#### GA

- Feature enabled by default for at least one release (beta)
- No major bugs reported
- Conformance tests if applicable

### Upgrade / Downgrade Strategy

Upgrading and downgrading is safe.

Upgrade:
For pre-existing PVCs: After upgrading, the `Unused` condition will not be
present on PVCs until a transition is observed by the controller. PVCs that
are never used after upgrade won't have this condition until the controller
processes them.

For new PVCs: PVCs that are created after upgrading will not have the `Unused`
condition initially. The condition will be added on the first observed transition
(e.g., when a Pod referencing the PVC terminates or when the controller first
evaluates the PVC's usage state).

Downgrade:
When downgrading to a version without this feature, the condition (if set) will
be preserved in etcd. Older controller-managers would simply ignore this condition.
The condition might go stale if transitions happen during the downgraded versions
but the updating process resumes when the version is upgraded and the first transition
occurs.

### Version Skew Strategy

| API Server | KCM | Behavior |
| :--- | :--- | :--- |
| off | off | Existing Kubernetes behavior. |
| on | off | Existing Kubernetes behavior. The `Unused` condition type is recognized but never set by the controller. Only users can set it manually via the API. |
| off | on | PVC protection controller may attempt to set the `Unused` condition, which will be dropped by the API server since the condition type is not recognized. |
| on | on | New behavior. `Unused` condition is set to `Status=True` when a PVC transitions to not being in use, and `Status=False` when it transitions to being in use. |

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `PersistentVolumeClaimUnusedSinceTime`
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-controller-manager

###### Does enabling the feature change any default behavior?

No. The condition is not read by any other Kubernetes component for any purposes
and so, existing workflows that do not explicitly read this condition would remain
unaffected. The condition also is not present on existing PVCs after enabling the
feature, until the controller observes a transition, which is when the condition
is added.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, disabling the feature will stop the controller from updating the `Unused`
condition. The condition however, once set, would remain stored in etcd, but
become stale - disabling doesn't remove it. If the condition was not yet set,
it remains absent and cannot be newly added while the feature gate is disabled.

###### What happens if we reenable the feature if it was previously rolled back?

The controller will resume managing the `Unused` condition. If the PVC didn't
transition while the feature was disabled, the condition already stored represents
the correct state. If the PVC did transition while the feature was disabled, the
condition might be stale or missing. There could be 2 scenarios:

- If a PVC transitioned from not in use to in use, the old condition (`Status=True`)
  is retained instead of being updated to `Status=False`, thus becoming stale.
- If a PVC transitioned from in use to not in use, the condition remains at
  `Status=False` (or absent) instead of being set to `Status=True`.

In either case, the condition will be corrected on the next transition after the
feature is re-enabled.

###### Are there any tests for feature enablement/disablement?

Unit tests for enabling and disabling feature gate are required for alpha - see "Graduation criteria" section.

The tests should verify the correct handling of the `Unused` condition in relation to the
feature gate state. Correct handling means the condition is correctly set/updated when
the PVC transitions between used and unused states while the feature gate is enabled,
and the condition values are preserved when the feature gate is disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The controller may temporarily queue all PVCs for re-evaluation on startup. In
large clusters this may cause temporary throttling. Running workloads are not
affected since the `Unused` condition is informational only and does not influence
PVC deletion protection logic or any other controller behavior.

###### What specific metrics should inform a rollback?

Watch for excessive PVC status update errors in kube-controller-manager logs.
Monitor API server request rates for PVC status updates. An unexpected increase
in PVC status update call volume or error rates could indicate a problem with the
feature.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not tested yet.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Operators can check if PVCs have an `Unused` condition in their `.status.conditions`.

###### How can someone using this feature know that it is working for their instance?

- [X] API .status
  - Condition type: `Unused`
  - Other field: Check `.status.conditions` for a condition with `type: Unused`

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

99% of PVC unused condition transitions should be reflected within 60 seconds
of the triggering event (e.g., Pod deletion or Pod creation referencing the PVC).

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - No metric
- [x] Other (treat as last resort)
  - Rate of PVC status update errors in kube-controller-manager
  - Latency between Pod deletion and `Unused` condition update on the PVC. Observable by comparing Pod deletion timestamp with the `lastTransitionTime` on the `Unused` condition.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes. A new UpdateStatus call for PersistentVolumeClaim. Estimated throughput would be
one UpdateStatus call per PVC when transitioning between in use and not in use states.
The originating component is PVC Protection Controller (in kube-controller-manager).

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes. PVC objects will have a new `Unused` condition in their `.status.conditions`
list. Estimated increase in size would be < 200B per PVC (condition includes type,
status, reason, message, and lastTransitionTime).

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The controller will be unable to update the `Unused` condition on PVCs.

###### What are other known failure modes?

One failure mode would be the delay in updating the condition for clusters
having a large number of PVCs (this has been discussed in the Risks and Mitigations
section). This might sometimes lead to the `lastTransitionTime` not being entirely
accurate. In such cases however, the time reported can be shorter than actual time
the PVC was unused, but it should never be longer.

This feature extends the PVC Protection controller logic with an additional
status condition update. Other failure modes would be similar to existing failure modes
of the controller. If KCM is unavailable, conditions won't be updated. If the API
server and/or etcd is unavailable, the conditions won't be updated (covered in the
section above).

###### What steps should be taken if SLOs are not being met to determine the problem?

Users should check kube-controller-manager logs for errors related to
PVC status updates.

## Implementation History

- 1.36: alpha
- 1.37: beta

## Drawbacks

None.

## Alternatives

One alternative was the use the deletion of `VolumeAttachment` objects as triggers
for updating the status. This was ruled out because of the fact that not all
volume types create a `VolumeAttachment` object, restricting the scope of the KEP.

Another alternative considered was using a dedicated status field (`UnusedSince *metav1.Time`)
on `PersistentVolumeClaimStatus` instead of a condition. The condition-based approach
was chosen because conditions are Kubernetes-idiomatic, integrate with existing PVC
condition infrastructure, and provide standardized fields (type, status, reason,
message, lastTransitionTime) that are well understood by tooling and operators.

## Infrastructure Needed (Optional)

None.
