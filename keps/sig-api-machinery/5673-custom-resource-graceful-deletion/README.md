# KEP-5673: Graceful Deletion for Custom Resources

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: VM Graceful Shutdown](#story-1-vm-graceful-shutdown)
    - [Story 2: Database Connection Draining](#story-2-database-connection-draining)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Deletion State Machine](#deletion-state-machine)
  - [Scenario Matrix](#scenario-matrix)
  - [Implementation Strategy](#implementation-strategy)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
    - [Unit tests](#unit-tests)
    - [Integration tests](#integration-tests)
    - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Requirements for Stable](#requirements-for-stable)
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
  - [Alternative 1: CRD-level Opt-in](#alternative-1-crd-level-opt-in)
  - [Alternative 2: Annotation-based Grace Period](#alternative-2-annotation-based-grace-period)
  - [Alternative 3: Controller-managed Grace Period](#alternative-3-controller-managed-grace-period)
  - [Alternative 4: Support Grace Period Without Finalizers](#alternative-4-support-grace-period-without-finalizers)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
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
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes adding support for graceful deletion to Custom Resources (CRs)
by honoring the `--grace-period` flag (and `DeleteOptions.GracePeriodSeconds`)
during deletion operations. Currently, when users delete a Custom Resource with
a specified grace period, the value is ignored, leading to unexpected behavior
for controllers and operators that rely on finalizers and graceful shutdown logic.

This enhancement will ensure that when a Custom Resource is deleted with a grace
period, the `metadata.deletionGracePeriodSeconds` field is properly set, allowing
finalizers to observe this value and implement appropriate graceful shutdown
behavior.

This KEP proposes implementing this as stable behavior without a feature gate,
as the change is purely additive, backward compatible, and poses minimal risk.

## Motivation

Custom Resources are first-class citizens in Kubernetes and are widely used to
extend the platform with custom workloads and infrastructure. Many of these
resources represent stateful systems (VMs, databases, network infrastructure)
that require graceful shutdown procedures to avoid data loss or service
disruption.

Currently, the graceful deletion mechanism available for built-in resources like
Pods is not available for Custom Resources. When users issue deletion commands
with grace periods like `kubectl delete vm/my-vm --grace-period=30`, the grace
period is ignored, and the resource is either deleted immediately (if no
finalizers exist) or waits indefinitely for finalizers without any grace period
information.

This creates a significant gap in the API machinery's handling of Custom
Resources compared to core resources, forcing custom resource authors to
implement their own grace period mechanisms or forego graceful deletion
altogether.

### Goals

- Enable Custom Resources to support graceful deletion via
  `DeleteOptions.GracePeriodSeconds`
- Populate `metadata.deletionGracePeriodSeconds` on Custom Resources when a
  grace period is specified
- Ensure finalizers can observe and react to the grace period
- Maintain backward compatibility with existing Custom Resource definitions
- Define clear semantics for all combinations of deletion state, finalizer
  state, and grace period options
- Provide comprehensive test coverage for all deletion scenarios
- Implement as stable behavior without requiring a feature gate, given the
  low-risk and backward-compatible nature of the change

### Non-Goals

- Changing the deletion behavior of built-in Kubernetes resources
- Automatically implementing graceful shutdown logic for Custom Resources
  (this remains the responsibility of controllers/operators via finalizers)
- Enforcing grace period limits on Custom Resources (controllers are responsible
  for honoring the grace period)
- Supporting grace periods for Custom Resources without finalizers (consistent
  with Pod behavior where grace periods only apply when there's cleanup work)
- Changing the CRD API or requiring CRD authors to explicitly opt-in to this
  feature
- Requiring a feature gate for this change (the enhancement is backward compatible
  and low-risk)

## Proposal

### User Stories

#### Story 1: VM Graceful Shutdown

As a user running virtual machines in Kubernetes using a custom VM resource,
I want to be able to specify a grace period when deleting a VM so that:

```bash
kubectl delete vm/my-vm --grace-period=300
```

This allows the VM controller's finalizer to:
1. Observe the 300-second grace period from `metadata.deletionGracePeriodSeconds`
2. Initiate a graceful shutdown of the VM (ACPI shutdown signal)
3. Wait up to 300 seconds for the VM to shut down cleanly
4. Force terminate if the grace period expires
5. Remove the finalizer and allow the resource to be deleted

#### Story 2: Database Connection Draining

As a user operating a custom database resource, I want to specify different
grace periods for different deletion scenarios:

```bash
# Normal maintenance - long grace period for connection draining
kubectl delete database/prod-db --grace-period=600

# Emergency shutdown - short grace period
kubectl delete database/test-db --grace-period=10

# Force delete - immediate cleanup
kubectl delete database/temp-db --grace-period=0 --force
```

The database controller can observe these different grace periods and adjust its
cleanup behavior accordingly.

### Risks and Mitigations

**Risk**: Existing controllers might not be prepared to handle
`deletionGracePeriodSeconds` being set on Custom Resources.

**Mitigation**: The field being set is purely informational unless controllers
explicitly check and use it. Existing controllers that don't check this field
will continue to work exactly as before - they'll ignore the field and use their
existing finalizer logic. This is a purely additive change with zero breaking
potential. Controllers can adopt the new behavior at their own pace.

**Risk**: Controllers might not respect the grace period, leaving resources in
a deletion state indefinitely.

**Mitigation**: This is already a concern with finalizers in general. This KEP
doesn't change the existing finalizer timeout behavior. Cluster administrators
already have mechanisms to forcefully remove finalizers if needed. Documentation
will emphasize controller responsibilities. The grace period is advisory, not
enforced by the API server.

**Risk**: Confusion about the interaction between grace period, finalizers, and
forced deletion.

**Mitigation**: Comprehensive documentation and test coverage will clarify all
scenarios. The design details section explicitly covers all 12 scenarios
identified by reviewers.

**Risk**: Implementing without a feature gate means we cannot easily roll back.

**Mitigation**: The change is purely additive and backward compatible. The worst
case is that the field is set but ignored by controllers, which is safe. If
critical issues are discovered, we can document known issues and address them
in subsequent releases. The low-risk nature of this change (adding an optional
metadata field) makes a feature gate unnecessary overhead.

## Design Details

### API Changes

No new API fields are needed. We will utilize the existing
`metadata.deletionGracePeriodSeconds` field that is already present in
`ObjectMeta` but currently only used for Pods.

When a Custom Resource is deleted with a grace period:
- `metadata.deletionTimestamp` will be set (existing behavior)
- `metadata.deletionGracePeriodSeconds` will be set to the requested grace period
  (new behavior)

### Deletion State Machine

The deletion of a Custom Resource can be in one of the following states:

1. **Not Deleting**: `deletionTimestamp` is `nil`
2. **Graceful Deletion**: `deletionTimestamp` is set, finalizers present,
   `deletionGracePeriodSeconds` > 0
3. **Force Deletion**: `deletionTimestamp` is set, finalizers present,
   `deletionGracePeriodSeconds` = 0
4. **Final Deletion**: `deletionTimestamp` is set, no finalizers

### Scenario Matrix

The following table defines the behavior for all 12 scenarios identified in the
PR review (3 existing object deletion states × 2 finalizer states × 2 grace
period states):

| #  | Existing deletionTimestamp | Finalizers     | GracePeriodSeconds | CheckGracefulDelete Return | GracePeriodSeconds Set | Notes                                                |
|----|----------------------------|----------------|--------------------|----------------------------|------------------------|------------------------------------------------------|
| 1  | nil                        | Has finalizers | nil                | false                      | N/A                    | Immediate deletion blocked by finalizers             |
| 2  | nil                        | Has finalizers | 0 (force)          | true                       | 0                      | Force deletion, finalizers must complete immediately |
| 3  | nil                        | Has finalizers | >0                 | true                       | User-provided value    | Graceful deletion with grace period                  |
| 4  | nil                        | No finalizers  | nil                | false                      | N/A                    | Immediate deletion (no cleanup needed)               |
| 5  | nil                        | No finalizers  | 0 (force)          | false                      | N/A                    | Immediate deletion (no cleanup needed)               |
| 6  | nil                        | No finalizers  | >0                 | false                      | N/A                    | No finalizers = no cleanup = no grace period needed  |
| 7  | Set                        | Has finalizers | nil                | false                      | N/A                    | Already deleting, no change                          |
| 8  | Set                        | Has finalizers | 0 (force)          | true                       | 0                      | Update to force deletion                             |
| 9  | Set                        | Has finalizers | >0                 | true                       | User-provided value    | Update grace period (if shorter)                     |
| 10 | Set                        | No finalizers  | nil                | false                      | N/A                    | Already past grace period, final deletion            |
| 11 | Set                        | No finalizers  | 0 (force)          | false                      | N/A                    | Already past grace period, final deletion            |
| 12 | Set                        | No finalizers  | >0                 | false                      | N/A                    | Already past grace period, final deletion            |

**Key Decision Points**:

1. Graceful deletion is **only supported for resources with finalizers** (rows 1-3, 7-9).
   This is consistent with Pod behavior where grace periods only matter when
   there's cleanup work to do.

2. When `deletionTimestamp` is already set and the last finalizer is removed
   (rows 10-12), the resource proceeds to final deletion immediately. The grace
   period has already been applied during initial deletion.

3. Subsequent delete calls on an already-deleting resource can:
   - Force immediate deletion by setting `gracePeriodSeconds=0` (row 8)
   - Request a shorter grace period (row 9, only if shorter than remaining time)
   - Otherwise, the existing deletion state is maintained (row 7)

### Implementation Strategy

The implementation will modify the `customResourceStrategy` in
`staging/src/k8s.io/apiextensions-apiserver/pkg/registry/customresource/strategy.go`:

```go
// CheckGracefulDelete updates the delete option with the desired grace value
func (a customResourceStrategy) CheckGracefulDelete(ctx context.Context, obj runtime.Object, options *metav1.DeleteOptions) bool {
	// No grace period requested
	if options == nil || options.GracePeriodSeconds == nil {
		return false
	}

	metaObj, err := meta.Accessor(obj)
	if err != nil {
		return false
	}

	// Only support graceful deletion if the object has finalizers
	if len(metaObj.GetFinalizers()) == 0 {
		return false
	}

	// If already deleting, allow grace period updates only if:
	// 1. Force delete (gracePeriodSeconds == 0)
	// 2. Shorter grace period than remaining time
	if metaObj.GetDeletionTimestamp() != nil {
		if *options.GracePeriodSeconds == 0 {
			// Force delete
			return true
		}
		
		// Calculate remaining grace period
		if metaObj.GetDeletionGracePeriodSeconds() != nil {
			remainingTime := time.Until(metaObj.GetDeletionTimestamp().Add(
				time.Duration(*metaObj.GetDeletionGracePeriodSeconds()) * time.Second))
			requestedTime := time.Duration(*options.GracePeriodSeconds) * time.Second
			
			// Only allow shortening the grace period
			if requestedTime < remainingTime {
				return true
			}
		}
		
		return false
	}

	return true
}
```

Additionally, the generic deletion code in `staging/src/k8s.io/apiserver/pkg/registry/generic/registry/store.go`
will need to be updated to:

1. Pass the grace period information to the strategy's `CheckGracefulDelete` method
2. Set `metadata.deletionGracePeriodSeconds` when the strategy returns `true`
3. Handle the update when the last finalizer is removed (scenarios 10-12)

### Test Plan

#### Prerequisite testing updates

None required. This feature adds new behavior that is backward compatible with
existing deletion mechanisms.

#### Unit tests

Unit tests will cover:

- `staging/src/k8s.io/apiextensions-apiserver/pkg/registry/customresource/strategy.go`:
  Test all 12 scenarios in the scenario matrix
  - All combinations of deletion state, finalizer state, and grace period
  - Edge cases (negative grace period, very large grace period)

- Generic deletion code in `staging/src/k8s.io/apiserver/pkg/registry/generic/registry/store.go`:
  - Verify `deletionGracePeriodSeconds` is set when `CheckGracefulDelete` returns true
  - Verify the field is not set when no finalizers are present

Target coverage: >90% for modified code

#### Integration tests

Integration tests will be added in `test/integration/apiserver/customresource/`:

1. **Basic Graceful Deletion**:
   - Create a CR with a finalizer
   - Delete with `--grace-period=30`
   - Verify `deletionTimestamp` and `deletionGracePeriodSeconds` are set
   - Remove finalizer before grace period expires
   - Verify resource is deleted

2. **Force Deletion**:
   - Create a CR with a finalizer
   - Delete with `--grace-period=0 --force`
   - Verify `deletionGracePeriodSeconds=0`
   - Verify resource is deleted immediately after finalizer removal

3. **Grace Period Update**:
   - Create a CR with a finalizer and delete with `--grace-period=60`
   - Issue second delete with `--grace-period=10`
   - Verify grace period is updated to the shorter value
   - Issue third delete with `--grace-period=30`
   - Verify grace period remains at 10 (not extended)

4. **No Finalizers**:
   - Create a CR without finalizers
   - Delete with `--grace-period=30`
   - Verify resource is deleted immediately and `deletionGracePeriodSeconds` is not set

5. **Finalizer Removal During Grace Period**:
   - Create a CR with a finalizer and delete with `--grace-period=60`
   - Wait 5 seconds
   - Remove finalizer
   - Verify resource is deleted immediately (doesn't wait for remaining grace period)

6. **Multiple Finalizers**:
   - Create a CR with two finalizers
   - Delete with `--grace-period=30`
   - Remove one finalizer
   - Verify resource still exists with grace period
   - Remove second finalizer
   - Verify resource is deleted

#### e2e tests

E2e tests will be added in `test/e2e/apimachinery/`:

1. **End-to-End Custom Controller**:
   - Deploy a test CRD and controller
   - Controller uses finalizer and respects `deletionGracePeriodSeconds`
   - Delete CR with various grace periods
   - Verify controller behavior (cleanup timing, forced deletion)

2. **Kubectl Integration**:
   - Test `kubectl delete --grace-period=N` with CRs
   - Test `kubectl delete --force --grace-period=0` with CRs
   - Verify `kubectl describe` shows grace period information

### Graduation Criteria

This KEP proposes direct implementation as a stable feature without the typical
alpha/beta graduation process. This is justified because:

1. **Backward Compatible**: The change only adds information (a metadata field)
   that existing controllers can ignore
2. **Low Risk**: The field is advisory; it doesn't change deletion mechanics
3. **No Breaking Changes**: Controllers that don't check the field continue to
   work exactly as before
4. **Simple Logic**: The implementation is straightforward with no complex edge cases
5. **Follows Existing Patterns**: Mirrors how graceful deletion works for Pods

#### Requirements for Stable

- All 12 scenarios in the scenario matrix have comprehensive test coverage
- Integration tests demonstrate all deletion flows work correctly  
- E2e tests with real controllers verify the feature works end-to-end
- Documentation clearly explains the behavior and controller implementation
- Code review approval from SIG API Machinery maintainers
- At least 2 example controllers demonstrating usage patterns
- Performance testing shows no significant impact on deletion operations

### Upgrade / Downgrade Strategy

**Upgrade**: When upgrading to a version with this feature:
- CRs deleted with grace periods will immediately have `deletionGracePeriodSeconds` set
- Existing CRs continue to work without changes
- Controllers that don't use grace periods are unaffected
- Controllers can start using the feature by checking for
  `deletionGracePeriodSeconds` in their finalizer logic
- No migration or configuration changes required

**Downgrade**: When downgrading from a version with this feature:
- CRs in a grace period state will continue to be deleted when finalizers are removed
- The `deletionGracePeriodSeconds` field will be preserved but ignored by older
  API servers
- Controllers should handle the absence of this field gracefully (fallback behavior)
- No data loss or corruption expected
- Existing finalizer cleanup logic continues to work

### Version Skew Strategy

This feature only affects the API server behavior. There are no node components
or client-side changes required.

**API Server Skew**: In an HA setup with mixed API server versions during upgrade:
- Requests to API servers with the feature will set `deletionGracePeriodSeconds`
- Requests to API servers without the feature will not set the field
- Controllers should handle both cases (field present/absent)
- This is acceptable during upgrade windows (typically < 1 hour for rolling upgrades)
- The behavior is eventually consistent once all API servers are upgraded

**Client Skew**: kubectl and other clients don't need updates. The
`--grace-period` flag already exists and sends `DeleteOptions.GracePeriodSeconds`.
All client versions work with both old and new API servers.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [ ] Feature gate
  - Feature gate name: N/A
  - Components depending on the feature gate: N/A
  
- [x] Other
  - Describe the mechanism: This feature is always enabled once the Kubernetes
    version containing it is deployed. It cannot be disabled.
  - Will enabling / disabling the feature require downtime of the control plane?
    N/A - feature is always enabled
  - Will enabling / disabling the feature require downtime or reprovisioning of a node? No

**Rationale for no feature gate**: This change is purely additive and backward
compatible. It only sets an additional metadata field that existing controllers
can safely ignore. The worst-case scenario is that the field is set but not used,
which poses no risk to cluster stability or functionality.

###### Does enabling the feature change any default behavior?

Yes, but in a backward-compatible way. When present, Custom Resources with
finalizers will have `metadata.deletionGracePeriodSeconds` set when deleted with
a grace period. However, this only provides additional information to
controllers; it doesn't change the deletion mechanics. Controllers that don't
check this field will continue to work exactly as before.

From a user perspective: The `kubectl delete --grace-period=N` flag will now
work for Custom Resources just as it does for Pods, making the platform more
consistent and intuitive.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Not directly - there is no feature gate to disable. However, rollback is achieved
by downgrading to a previous Kubernetes version. When downgraded:
- The `deletionGracePeriodSeconds` field will stop being set on new deletions
- Existing CRs with the field set will retain it, but it will be ignored
- Controllers that check for the field should handle it being absent gracefully
- No data corruption or loss expected

The inability to disable via feature gate is acceptable because:
1. The change is purely additive (no behavior is removed or changed)
2. Existing functionality is preserved (controllers ignoring the field still work)
3. The field being set poses no risk even if not used

###### What happens if we reenable the feature if it was previously rolled back?

Since there's no feature gate, "re-enabling" means upgrading back to a version
with the feature. The feature will resume working normally - CRs deleted with
grace periods will have the field set as expected.

###### Are there any tests for feature enablement/disablement?

Since there is no feature gate, tests focus on verifying the feature works
correctly and doesn't break existing behavior:

- Unit tests verify grace periods are honored when set
- Integration tests verify backward compatibility (controllers that don't use
  grace periods continue to work)
- Tests verify behavior during API server version skew (mixed old/new API servers)

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The rollout cannot fail in a way that impacts running workloads. The feature only
affects deletion operations on Custom Resources. Existing workloads continue to run.

Potential considerations during rollout:
- In HA setups with mixed API server versions during upgrade, some deletions may
  get grace periods (new API servers) while others don't (old API servers)
- Controllers that expect the grace period field should handle it being absent
  gracefully (which is good practice regardless)
- This is acceptable and temporary (only during the upgrade window)

###### What specific metrics should inform a rollback?

- Increased deletion latency for Custom Resources (metric:
  `apiserver_request_duration_seconds` filtered by DELETE operations on CRDs)
- Increased errors during deletion operations (metric:
  `apiserver_request_total` with `code` 5xx)
- CRs stuck in deletion state longer than expected (requires monitoring by
  operators)

If these metrics show significant degradation after upgrading, a rollback
(downgrade) should be considered. However, given the minimal nature of the change,
such issues are unlikely.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This will be tested as part of the implementation:
- Upgrade: Verify CRs deleted before upgrade complete deletion after upgrade
- Upgrade: Verify new deletions after upgrade properly set grace periods
- Downgrade: Verify CRs with grace periods in progress complete deletion after downgrade
- Upgrade again: Verify new deletions use grace periods correctly

Testing during API server version skew (mixed old/new) will also be included.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

1. Check if CRs have `deletionGracePeriodSeconds` set:
   ```bash
   kubectl get <crd-resource> -o jsonpath='{.metadata.deletionGracePeriodSeconds}'
   ```

2. Use the metric `apiserver_crd_graceful_deletion_total` (counter) to track:
   - Number of CRs deleted with grace periods
   - Breakdown by CRD type

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: N/A
  
- [x] API .status
  - Other field: `metadata.deletionGracePeriodSeconds` - When a CR is deleted with
    a grace period, this field will be populated. Users can verify with:
    ```bash
    kubectl get <resource> -o yaml | grep deletionGracePeriodSeconds
    ```

- [ ] Other (treat as last resort)

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- 99.9% of DELETE operations with grace periods should correctly set
  `deletionGracePeriodSeconds` within normal API request latency
- No increase in API server resource usage beyond normal DELETE operation overhead
- Deletion latency should not increase significantly (< 1ms added latency for
  grace period logic)

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `apiserver_request_duration_seconds` (existing metric, filtered
    by DELETE operations on custom resources)
  - [Optional] Aggregation method: histogram, 99th percentile
  - Components exposing the metric: kube-apiserver

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. This feature is entirely within the kube-apiserver and doesn't depend on any
external services.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. The feature modifies existing DELETE operations but doesn't introduce new API calls.

###### Will enabling / using this feature result in introducing new API types?

No. This feature uses existing API types and fields.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- API type(s): Custom Resources (all CRDs)
- Estimated increase in size: +8 bytes per CR in deletion state (for the
  `deletionGracePeriodSeconds` field, which is an int64)
- Estimated amount of new objects: None

The increase is minimal and only affects CRs currently being deleted.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No significant increase expected. The added logic in `CheckGracefulDelete` is
O(1) and involves simple field checks and arithmetic. Estimated overhead: < 1ms
per DELETE operation.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The feature adds minimal logic to the DELETE path:
- CPU: Negligible (simple field checks and assignments)
- RAM: +8 bytes per CR in deletion state
- Disk: No impact on etcd size (field is already part of ObjectMeta)
- I/O: No additional I/O

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. This feature only affects the API server and doesn't impact nodes.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

This feature is part of the API server's DELETE operation. If the API server or
etcd is unavailable, DELETE operations will fail as they do today. This feature
doesn't change that behavior.

###### What are other known failure modes?

1. **Controller doesn't respect grace period**:
   - Detection: CRs remain in deletion state longer than expected, or are deleted
     too quickly despite long grace periods
   - Mitigations: This is a controller implementation issue, not a feature issue.
     Provide documentation and examples for controller authors.
   - Diagnostics: Check CR's `deletionTimestamp` and `deletionGracePeriodSeconds`;
     monitor controller logs
   - Testing: E2e tests with example controllers

2. **Grace period set but finalizers missing**:
   - Detection: `deletionGracePeriodSeconds` is not set despite using `--grace-period`
   - Mitigations: This is expected behavior (grace periods only apply to resources
     with finalizers). Document clearly.
   - Diagnostics: Check if resource has finalizers:
     `kubectl get <resource> -o jsonpath='{.metadata.finalizers}'`
   - Testing: Integration test covering this scenario (test case 4 in the test plan)

3. **Feature not working but controller expects grace period**:
   - Detection: Controller logs warnings about missing grace period field
   - Mitigations: Controllers should handle missing field gracefully (use default
     or fallback behavior). This could happen during version skew.
   - Diagnostics: Check if all API servers are at the same version:
     `kubectl get nodes -o wide`
   - Testing: Integration test with API server version skew

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check API server metrics for increased DELETE latency:
   ```
   apiserver_request_duration_seconds{verb="DELETE",resource="<crd>"}
   ```

2. Check for errors in API server logs related to graceful deletion

3. Check if specific CRDs are affected or all CRDs

4. Review controller logs for finalizer handling issues

5. If issues persist and appear related to this feature, file a bug report with
   sig-api-machinery. As a temporary workaround, controllers can ignore the grace
   period field and use their existing logic.

## Implementation History

- 2025-10-29: KEP created and proposed with stable implementation (no feature gate)
- TBD: Implementation merged
- TBD: Released in Kubernetes v1.XX

## Drawbacks

- Adds complexity to the custom resource deletion logic
- Controllers must be updated to take advantage of the feature (though they
  continue to work without updates)
- Potential for confusion about when grace periods apply (only with finalizers)

## Alternatives

### Alternative 1: CRD-level Opt-in

Instead of a feature gate, require CRD authors to explicitly enable graceful
deletion via a field in the CRD spec:

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: virtualmachines.example.com
spec:
  gracefulDeletionEnabled: true
```

**Pros**:
- More explicit control for CRD authors
- No surprises for existing controllers

**Cons**:
- Requires CRD API changes
- Inconsistent with how other features work
- Adds burden on CRD authors to understand and enable
- Doesn't align with the philosophy that CRs should work like built-in resources

**Decision**: Rejected. Graceful deletion should be a platform feature, not
something CRD authors opt into.

### Alternative 2: Annotation-based Grace Period

Instead of using `DeleteOptions.GracePeriodSeconds`, use an annotation:

```bash
kubectl annotate vm/my-vm deletion.kubernetes.io/grace-period=30
kubectl delete vm/my-vm
```

**Pros**:
- More explicit and discoverable
- Could be set by automation

**Cons**:
- Inconsistent with how deletion works for built-in resources
- Requires two operations instead of one
- Doesn't work with existing kubectl flags
- More complex for users

**Decision**: Rejected. Should be consistent with existing Kubernetes deletion semantics.

### Alternative 3: Controller-managed Grace Period

Leave grace period management entirely to controllers via annotations or spec fields:

```yaml
apiVersion: example.com/v1
kind: VirtualMachine
spec:
  terminationGracePeriodSeconds: 30
```

**Pros**:
- No API server changes needed
- Controllers have full control

**Cons**:
- Inconsistent across different CRDs (every CRD would implement it differently)
- Can't be set at deletion time (must be pre-configured)
- Doesn't work with kubectl flags
- Doesn't feel like native Kubernetes behavior

**Decision**: Rejected. This KEP aims to provide platform-level support, not
force every controller to reinvent this mechanism.

### Alternative 4: Support Grace Period Without Finalizers

Allow grace periods even for CRs without finalizers (unlike Pods):

**Pros**:
- More flexible
- Simpler mental model (always allow grace periods)

**Cons**:
- Inconsistent with Pod behavior
- Grace periods without cleanup work are meaningless
- Would require API server to wait before deleting (performance impact)
- More complex implementation

**Decision**: Rejected. Grace periods should only apply when there's work to do
(i.e., finalizers present), consistent with Pod behavior.

