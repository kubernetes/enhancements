# KEP-3541: Add Recreate Update Strategy to StatefulSet

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Current Behavior and Problems](#current-behavior-and-problems)
  - [Why Existing Solutions Are Insufficient](#why-existing-solutions-are-insufficient)
  - [Proposed Solution Benefits](#proposed-solution-benefits)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: CI/CD Platform Team](#story-1-cicd-platform-team)
    - [Story 2: Stateless Web Application](#story-2-stateless-web-application)
    - [Story 3: Development/Experiment Environment](#story-3-developmentexperiment-environment)
    - [Story 4: External Data Storage](#story-4-external-data-storage)
    - [Story 5: LeaderWorkerSet (LWS) Use Case](#story-5-leaderworkerset-lws-use-case)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Risk: Unintended Data Loss](#risk-unintended-data-loss)
- [Design Details](#design-details)
  - [Detailed Algorithm Specification](#detailed-algorithm-specification)
    - [Current RollingUpdate Algorithm](#current-rollingupdate-algorithm)
    - [Proposed Recreate Strategy Algorithm](#proposed-recreate-strategy-algorithm)
  - [API Changes](#api-changes)
    - [Spec Changes](#spec-changes)
  - [Implementation Changes](#implementation-changes)
  - [Comparison with Existing Solutions](#comparison-with-existing-solutions)
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
  - [Downtime Requirement](#downtime-requirement)
  - [Limited Rollback Options](#limited-rollback-options)
- [Alternatives](#alternatives)
  - [Alternative 1: PodProgressTimeoutSeconds Field in RollingUpdate Strategy](#alternative-1-podprogresstimeoutseconds-field-in-rollingupdate-strategy)
  - [Alternative 2: EnforcedRollingUpdate Strategy](#alternative-2-enforcedrollingupdate-strategy)
  - [Alternative 3: (Now Primary Solution): Recreate Strategy](#alternative-3-now-primary-solution-recreate-strategy)
  - [Alternative 4: Add Force Flag to RollingUpdate](#alternative-4-add-force-flag-to-rollingupdate)
  - [Alternative 5: Enhance Parallel Policy](#alternative-5-enhance-parallel-policy)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

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
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

StatefulSets currently offer two update strategies: `OnDelete` (manual) and `RollingUpdate` (automatic, default). When using `RollingUpdate` with the default `podManagementPolicy: OrderedReady`, StatefulSets follow sequential ordering where each individual pod must be Running and Ready before the controller proceeds to update the next pod. Even with the `maxUnavailable` option (which allows multiple pods to be updated simultaneously), the controller still requires each pod to reach Ready state before moving forward but stuck pods halt the entire update process. While `podManagementPolicy: Parallel` allows pods to be updated simultaneously without waiting for Ready state, stuck pods remain and are not automatically replaced. This design ensures data safety for stateful workloads but creates a critical operational problem.

When a StatefulSet update results in pods that fail to reach Ready state (due to configuration errors, resource constraints, etc..), the rolling update process becomes permanently stuck. Even after applying a corrected configuration, the controller will not automatically replace the broken pods, requiring manual intervention to delete stuck pods.

This behavior has generated significant user frustration across multiple GitHub issues ([#67250](https://github.com/kubernetes/kubernetes/issues/67250), [#60164](https://github.com/kubernetes/kubernetes/issues/60164), [#109597](https://github.com/kubernetes/kubernetes/issues/109597)) with users reporting:

- Broken CI/CD pipelines requiring manual intervention
- Inability to automatically recover from configuration mistakes
- Operational burden in managing stateful applications

This KEP proposes adding a new `Recreate` update strategy to StatefulSets, mirroring the behavior of Deployments' 
Recreate strategy. This strategy deletes all pods, waits for full termination, then creates new pods according 
to `podManagementPolicy`. This provides a simple, predictable way to handle stuck pods and enables automated recovery for workloads that can tolerate downtime (CI/CD environments, stateless applications using StatefulSet 
for pod identity, applications with external data storage, and use cases like LeaderWorkerSet). The `Recreate` 
strategy offers a clean parallel with existing Kubernetes patterns, simplifies controller logic, and provides users with explicit control over update behavior.

## Motivation

### Current Behavior and Problems

StatefulSets with `RollingUpdate` strategy follow this algorithm:

1. if `podManagementPolicy: OrderedReady` (default)

   1. Update pods in reverse ordinal order (N-1, N-2, ..., 0)
   2. For each pod, wait until it becomes Running and Ready before proceeding to the next
   3. If any pod fails to become Ready, the entire update process halts
   4. Even when a corrected configuration is applied, stuck pods are never automatically replaced

2. if `podManagementPolicy: Parallel`

   1. Update all pods simultaneously (or up to `maxUnavailable` at a time if specified)
   2. Pods are created/deleted without waiting for Ready state
   3. Stuck pods do not block other pods from being updated
   4. Even when a corrected configuration is applied, stuck pods are never automatically replaced

The current approach was designed for stateful workloads where data persistence is critical, pod identity and storage are tightly coupled, or automatic pod deletion could cause data loss.

This behavior has significant impact across multiple scenarios:

**CI/CD Pipeline Failures**: Teams report broken deployments that require manual intervention, breaking automation:

```yaml
# Example: A typo in image name breaks the entire update
apiVersion: apps/v1
kind: StatefulSet
spec:
  template:
    spec:
      containers:
      - name: app
        image: myapp:v2.0.0-typo  # ImagePullBackOff
        # Update gets stuck, requires manual pod deletion
```

**Operational Overhead**: Platform teams must build custom controllers or fix it manually to handle stuck updates.

### Why Existing Solutions Are Insufficient

1. [MaxUnavailable](https://github.com/kubernetes/enhancements/issues/961) doesn't address the core issue.
    The `maxUnavailable` option in `RollingUpdate` strategy allows multiple pods to be updated simultaneously, but its behavior depends on `podManagementPolicy`.

    ```yaml
      spec:
        podManagementPolicy: Parallel
        updateStrategy:
          type: RollingUpdate
          rollingUpdate:
            maxUnavailable: 2  # Can update 2 pods at once
    ```

    With `podManagementPolicy: Parallel` + `maxUnavailable: 2`, multiple pods can be updated simultaneously, but if any pod fails to reach Ready state, it remains stuck and requires manual cleanup. Stuck pods don't block other pods from updating, but they are never automatically replaced (see section 2 below).

    With `podManagementPolicy: OrderedReady`, updates happen one pod at a time in reverse ordinal order. If any pod fails to reach Ready state, the entire rolling update process halts completely, even with `maxUnavailable` configured. The controller waits indefinitely for stuck pods to become Ready.

    Example Scenario with `podManagementPolicy: OrderedReady`:
        - StatefulSet with 5 replicas
        - Update pod `app-4` first
        - `app-4` gets stuck in `ImagePullBackOff`
        - Even after fixing the image name, `app-4` remains stuck
        - Update process cannot proceed to `app-3`, `app-2`, `app-1`, or `app-0`
        - Manual intervention still required: `kubectl delete pod app-4`

2. Custom Controllers
  Some teams have built custom controllers to delete stuck pods, but this:

     - Duplicates StatefulSet controller logic
     - Creates maintenance burden
     - May conflict with StatefulSet controller behavior
     - Lacks integration with StatefulSet status and events

### Proposed Solution Benefits

Adding `Recreate` update strategy to StatefulSets addresses these issues by:

1. Stuck pods are cleared and replaced during updates without manual intervention
2. Clean algorithm with no complexity around timeout tracking or transient failure detection
3. Consistency with Kubernetes Patterns (Deployment) Recreate strategy.
4. Handles All Stuck Scenarios, regardless of whether pods are stuck in ImagePullBackOff, Pending, CrashLoopBackOff, or any other state

### Goals

1. Add a new `Recreate` update strategy type to StatefulSet, providing a third option alongside `OnDelete` and `RollingUpdate`
2. Align StatefulSet update strategies with Deployment patterns for API consistency
3. Enable automated recovery from stuck pod states without manual intervention
4. Provide a simple, predictable update behavior for workloads that can tolerate downtime
5. Support use cases like CI/CD environments, stateless applications, external storage applications, and LeaderWorkerSet patterns

### Non-Goals

1. Change default behavior of StatefulSet updates (opt-in via explicit `type: Recreate` configuration)
2. Add timeout-based progressive failure detection (use Recreate for simplicity)
3. Change Recreate deletion semantics (all pods are always deleted simultaneously but recreate ordering follows `podManagementPolicy`)
4. Replace Deployment-style revision management (StatefulSets continue to directly manage Pods)

## Proposal

### User Stories

#### Story 1: CI/CD Platform Team

**Context**: A platform team manages hundreds of StatefulSet deployments across development and staging environments. Their CI/CD system requires end-to-end automation, but StatefulSet rolling updates break automation when pods get stuck. The team either has to implement custom "garbage collection" logic or accept that automated deployments will fail and require manual intervention. Since these are non-production environments, downtime during updates is acceptable.

**Solution**: With `updateStrategy: type: Recreate` configured, when an update with incorrect configuration is applied, all pods are deleted and new pods are created. If they fail, the deployment fails quickly and clearly. When a corrected configuration is applied, the Recreate strategy deletes all broken pods and creates fresh ones, allowing the CI/CD pipeline to complete without manual intervention. The downtime is acceptable in CI/CD environments where fast, automated recovery is more important than uptime.

#### Story 2: Stateless Web Application

**Context**: A web application uses StatefulSet for predictable pod naming but doesn't store critical data locally. When resource limit typos cause pods to get stuck in Pending state, the entire update halts even though pod replacement is safe. The application can tolerate brief downtime during updates.

**Solution**: With `updateStrategy: type: Recreate` configured, when an update encounters issues, all pods are deleted and recreated cleanly. This eliminates the need for manual pod deletion since stuck pods are automatically cleared. The brief downtime is acceptable for this stateless application that primarily uses StatefulSet for pod identity rather than stateful semantics.

#### Story 3: Development/Experiment Environment

**Context**: Developers using StatefulSet for experiments face constant frustration - every time a rolling update breaks due to configuration errors, they must manually delete stuck pods after applying fixes. This manual intervention disrupts the development workflow. Uptime is not a concern in development environments.

**Solution**: With `updateStrategy: type: Recreate` configured, developers get fast, clean resets - when an update fails, applying a fix automatically deletes all broken pods and creates fresh ones. This enables a smoother development experience without requiring cluster operator intervention or manual pod cleanup. The Recreate strategy's simplicity makes it ideal for rapid iteration in development.

#### Story 4: External Data Storage

**Context**: A database application stores all persistent data on network-attached storage (not local pod storage). Pod replacement is completely safe since no local data would be lost, but the StatefulSet controller treats it as a traditional stateful workload and requires manual intervention. The application can tolerate brief downtime for clean updates.

**Solution**: With `updateStrategy: type: Recreate` configured, the controller automatically deletes and recreates all pods during updates, which is safe for this architecture since all data persists externally. The Recreate strategy provides clean, predictable updates without concerns about stuck pods, and the brief downtime is acceptable given the data safety guarantees from external storage.

#### Story 5: LeaderWorkerSet (LWS) Use Case

**Context**: Developers use StatefulSet as the high-level controller workload for [LWS](https://github.com/kubernetes-sigs/lws). However, it behaves more like a Deployment - there's no ordering dependency between different replicas. They only need the ordinal index for pod identification. When a replica fails during updates, the entire StatefulSet update gets stuck, even though there's no actual ordering requirement between replicas. The LeaderWorkerSet pattern can tolerate brief downtime for updates.

**Solution**: With `updateStrategy: type: Recreate` configured, all replicas are cleanly deleted and recreated during updates, eliminating stuck pod scenarios entirely. This aligns perfectly with the deployment-like nature of LWS workloads, providing simple and predictable updates for applications that use StatefulSet primarily for pod identity rather than traditional stateful semantics. The Recreate strategy's "all or nothing" approach matches the LWS pattern where all workers restart together.

### Notes/Constraints/Caveats

- Strategy Type Change Does Not Trigger Rollout: changing only `.spec.updateStrategy.type` from `RollingUpdate` to `Recreate` (or vice versa) does not trigger a new rollout. This is consistent with Deployment behavior. The StatefulSet controller uses the `controller-revision-hash` label to identify pod revisions, which is computed from `.spec.template` content only.

The Recreate behavior will only be triggered when users either:
   1. Make a change to `.spec.template`
   2. Force a rollout using `kubectl rollout restart`

### Risks and Mitigations

#### Risk: Unintended Data Loss

**Risk Description**: If `Recreate` strategy is used on StatefulSets with local persistent data and PersistentVolumeClaims, the downtime could affect applications expecting sequential updates. However, data on PVCs is preserved since Recreate only deletes pods, not volumes.

**Mitigation Strategies**:

1. Documentation: Clear guidance on when to use `Recreate` strategy - suitable for workloads that can tolerate downtime
2. No Default Change: Opt-in behavior - existing workloads continue using safe `RollingUpdate` (current behavior unchanged)
3. Explicit Strategy Selection: Users must explicitly set `type: Recreate`, preventing accidental usage
4. Clear Events: Events emitted during the recreate process to show deletion and recreation phases
5. Status Conditions: StatefulSet status clearly reflects the recreate process state
6. PVC Preservation: PersistentVolumeClaims are not deleted, so data on volumes persists across recreate operations

## Design Details

### Detailed Algorithm Specification

#### Current RollingUpdate Algorithm

```
FOR i = replicas-1 To i >= 0 DO i-- 
    If pod[i] needs update Then
        wait_for_predecessors_ready(i+1 to replicas-1)
        If !pod[i].Running Or !pod[i].Ready Then
            return // STUCK - wait for manual intervention
        ENDIF
        update_pod(i)
        wait_until_ready(pod[i])
    ENDIF
ENDFOR
```

The algorithm halts when `pod[i]` is not Running or Ready, even if a fix is applied.

#### Proposed Recreate Strategy Algorithm

```
// Recreate Strategy Algorithm
// Uses controller-revision-hash label to identify pod revision (same as RollingUpdate)
// updateRevision = hash of current spec.template (computed by controller)

current_phase = determine_phase()

IF current_phase == "NeedsDeletion" THEN
    // Phase 1: Delete all pods with old revision
    emit_event("RecreateStarted", "Deleting all pods for Recreate update")
    set_condition("Progressing", status="True", reason="RecreateInProgress")
    
    // Delete ALL pods owned by this StatefulSet that have old revision
    // This handles orphaned pods with ordinals >= replicas
    FOR each pod in pods:
        IF pod.Labels["controller-revision-hash"] != updateRevision THEN
            IF pod.DeletionTimestamp == nil THEN
                delete_pod(pod)
            ENDIF
        ENDIF
    ENDFOR
    return // Reconcile again after deletions are issued
ENDIF

IF current_phase == "WaitingTermination" THEN
    // Phase 2: Wait for all old-revision pods to be fully removed from etcd
    // Controller watches pods and will reconcile when deletions complete
    // Note: Only emit event on first entry to this phase (tracked via condition)
    return
ENDIF

IF current_phase == "ReadyForCreation" THEN
    // Phase 3: Create pods with new revision according to podManagementPolicy
    IF podManagementPolicy == OrderedReady THEN
        // Create in ascending ordinal order; only create the next ordinal when predecessor is Running and Ready
        i = lowest ordinal in [0, replicas-1] such that pod i does not exist
        IF i is defined THEN
            IF i == 0 OR (pod i-1 exists AND is Running and Ready) THEN
                create_pod(i, updateRevision)
            ENDIF
        ENDIF
    ELSE
        // Parallel: create all missing pods at once
        FOR i = 0 TO replicas-1:
            IF pod with ordinal i does not exist THEN
                create_pod(i, updateRevision)
            ENDIF
        ENDFOR
    ENDIF
    return // Reconcile again to check creation progress
ENDIF

IF current_phase == "Complete" THEN
    // All replicas exist with current revision
    set_condition("Progressing", status="True", reason="RecreateComplete")
    return
ENDIF

// Helper: Determine current phase based on pod states
FUNCTION determine_phase():
    pods = get_all_pods_for_statefulset()  // All pods owned by this StatefulSet
    
    old_revision_pods_active = 0      // Old revision, not yet deleted
    old_revision_pods_terminating = 0 // Old revision, has DeletionTimestamp
    new_revision_pods = 0             // Current revision (not terminating)
    
    FOR each pod in pods:
        IF pod.Labels["controller-revision-hash"] != updateRevision THEN
            // Pod has old revision
            IF pod.DeletionTimestamp == nil THEN
                old_revision_pods_active++
            ELSE
                old_revision_pods_terminating++
            ENDIF
        ELSE
            // Pod has current revision
            IF pod.DeletionTimestamp == nil THEN
                new_revision_pods++
            ENDIF
            // Note: new revision pods with DeletionTimestamp are ignored
            // (could happen if user manually deleted, will be recreated)
        ENDIF
    ENDFOR
    
    // Phase 1: Any old-revision pods that haven't been deleted yet
    IF old_revision_pods_active > 0 THEN
        return "NeedsDeletion"
    ENDIF
    
    // Phase 2: Old pods are terminating, wait for full removal
    IF old_revision_pods_terminating > 0 THEN
        return "WaitingTermination"
    ENDIF
    
    // Phase 3: No old pods remain, but we don't have enough new pods yet
    IF new_revision_pods < replicas THEN
        return "ReadyForCreation"
    ENDIF
    
    // Phase 4: All replicas exist with current revision
    return "Complete"
END FUNCTION
```

Key Characteristics:

1. Uses `controller-revision-hash` label (same as RollingUpdate) to identify old vs new pods
2. All old-revision pods are fully terminated before any new pods are created
3. Guarantees old and new pods never run simultaneously
4. Deletes all old-revision pods including orphans with ordinals >= replicas
5. Since all pods are forcibly deleted, updates cannot become permanently blocked
6. Explicit downtime: Users opt-in knowing there will be unavailability between deletion and creation phases
7. Safe to retry deletions and creations on controller restart
8. Recreation phase respects `podManagementPolicy`

### API Changes

#### Spec Changes

```go
// StatefulSetUpdateStrategyType is a string enumeration type that represents the update strategy type for StatefulSets
type StatefulSetUpdateStrategyType string

const (
    // RollingUpdateStatefulSetStrategyType indicates that pods in a StatefulSet will be updated in reverse ordinal order
    RollingUpdateStatefulSetStrategyType StatefulSetUpdateStrategyType = "RollingUpdate"
    // OnDeleteStatefulSetStrategyType indicates that pods in a StatefulSet will only be updated when manually deleted
    OnDeleteStatefulSetStrategyType StatefulSetUpdateStrategyType = "OnDelete"
    // RecreateStatefulSetStrategyType indicates that all pods will be fully terminated before new ones are created
    RecreateStatefulSetStrategyType StatefulSetUpdateStrategyType = "Recreate"
)
```


Example Usage:

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: web
spec:
  replicas: 10
  updateStrategy:
    type: Recreate
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
```

**Behavior**:

- When update is triggered (e.g., template change):
  1. All pods (web-0 through web-9) are deleted simultaneously
  2. Controller waits for all pods to fully terminate
  3. All new pods (web-0 through web-9) are created according to their `.spec.podManagementPolicy`
- Downtime occurs between deletion and recreation phases
- No stuck pod scenarios - all pods are forcibly deleted

### Implementation Changes

The implementation requires changes to the StatefulSet controller in `pkg/controller/statefulset/stateful_set_control.go`:

1. Strategy Type Handling:
   - Add new case for `RecreateStatefulSetStrategyType` in update strategy switch statement
   - Implement separate update path for Recreate strategy alongside existing RollingUpdate and OnDelete paths

2. Recreate Update Logic:
   - Phase 1 - Deletion: Iterate through all pods and delete them (similar to scale-down operation)
   - Phase 2 - Wait for Termination: Check all pods for `deletionTimestamp`; reconcile periodically until all pods are fully terminated
   - Phase 3 - Recreation: Create all new pods according to `spec.podManagementPolicy`

3. Status Condition Management:
   - Add `Progressing` condition to StatefulSet status

4. Validation:
   - API validation in `pkg/apis/apps/validation/validation.go`
   - Validate `type: Recreate` can be set on StatefulSet
   - No additional fields required for Recreate strategy (unlike RollingUpdate which has partition, maxUnavailable)

5. Respect Ordering Semantics:
   - Recreate strategy according to `podManagementPolicy` settings
   - All pods deleted at once and then re-created according to `podManagementPolicy` settings

### Comparison with Existing Solutions

| Solution                           | Sequential Ordering | Automatic Recovery | Downtime | Behavior When Pod Stuck         | Use Case                                     |
| ---------------------------------- | ------------------- | ------------------ | -------- | ------------------------------- | -------------------------------------------- |
| `RollingUpdate` (default)          | Yes                 | No                 | No       | Halts completely, waits forever | Traditional stateful apps                    |
| `RollingUpdate` + `maxUnavailable` | Yes (batched)       | No                 | No       | **Still halts completely**      | Faster updates, but same stuck problem       |
| `OnDelete`                         | Yes (manual)        | No                 | No       | Fully manual control            | Maximum safety/control                       |
| **`Recreate` (proposed)**          | No                  | Yes                | Yes      | All pods deleted and recreated  | CI/CD, stateless apps, external storage, LWS |

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `pkg/apis/apps/validation/validation.go`: `2025-10-13` - `92.8%`
- `pkg/controller/statefulset/stateful_set_control.go`: `2025-10-13` - `91.5%`
- `pkg/controller/statefulset/stateful_pod_control.go`: `2025-10-13` - `89.6%`
- `pkg/registry/apps/statefulset/strategy.go`: `2025-10-13` - `83.9%`

##### Integration tests

We should cover below scenarios:

- Without `type: Recreate`: Existing StatefulSets with `RollingUpdate` and `OnDelete` continue to work unchanged (backward compatibility)
- With `type: Recreate` configured:
  - All pods are deleted when update is triggered (template spec change)
  - Controller waits for all pods to fully terminate (no pods with deletionTimestamp remain)
  - All new pods are created after termination complete
  - Status condition `Progressing=True`
  - Status condition `Progressing=True` with `reason=RecreateComplete` after pods created
  - Recreate strategy respects `podManagementPolicy`
- PVC preservation: PersistentVolumeClaims are not deleted during Recreate (only pods are deleted)
- Stuck pod handling: Pods stuck in any state are forcibly deleted (ImagePullBackOff, Pending, CrashLoopBackOff, etc.)
- Validation: API validation accepts `type: Recreate` on StatefulSet

##### e2e tests

The following e2e tests will be added to `test/e2e/apps/statefulset.go`:

- StatefulSet with `type: Recreate` successfully deletes and recreates all pods during update
- Recreate works with stuck pods (ImagePullBackOff scenario - pods are deleted and new ones created)
- Recreate waits for full termination before creating new pods (no mixed old/new state)
- Recreate preserves PersistentVolumeClaims (data persists across recreation)
- Recreate respects `podManagementPolicy` during recreation
- StatefulSets without `type: Recreate` maintain current RollingUpdate/OnDelete behavior (backward compatibility)
- Controller restart during Recreate resumes correctly from last phase

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag.
- Unit and integration tests passed as designed in [TestPlan](#test-plan).

#### Beta

- Feature is enabled by default.
- Address reviews and bug reports from Alpha users.
- e2e tests:
  - Add links to testgrid results
  - Verify zero flakes over 2+ weeks

#### GA

- No negative feedback from developers.
- Consider conformance test if feature becomes widely adopted and part of core contract
- Ensure existing conformance tests for basic RollingUpdate continue to pass

### Upgrade / Downgrade Strategy

#### Upgrade

This feature is protected by the feature-gate `StatefulSetRecreateStrategy`, which must be enabled on both `kube-apiserver` and `kube-controller-manager`.

**Component Dependencies**:

- kube-apiserver: Validates and persists the `type: Recreate` strategy in the StatefulSet spec
- kube-controller-manager: Implements the Recreate strategy logic (delete all, wait for termination, create all)

**Upgrade Sequence**

1. Enable feature gate on kube-apiserver first
2. Enable feature gate on kube-controller-manager
3. Create/update StatefulSets with `updateStrategy.type: Recreate`

**Partial Upgrade Behavior**

- If apiserver has feature enabled but kube-controller-manager does not:
  - API server accepts `type: Recreate` strategy
  - Strategy type is persisted in etcd
  - Kube-controller-manager ignores Recreate type and falls back to default RollingUpdate behavior
  - No errors, but Recreate behavior is not active
  
- If apiserver does NOT have feature enabled but kube-controller-manager does:
  - API server rejects create/update requests that set `type: Recreate` with a validation error
  - Users cannot create or switch to Recreate until the apiserver has the feature enabled.
  - Kube-controller-manager cannot process Recreate in this skew because no `StatefulSet` with `type: Recreate` can be stored.

> Enable the feature gate on `kube-apiserver` first, then `kube-controller-manager` to ensure smooth transition.

#### Downgrade

- The older apiserver does not recognize `type: Recreate` and will reject create/update requests that set it. 
- StatefulSets that already have `type: Recreate` stored in etcd remain stored, but any update that touches the spec may be rejected unless the strategy is changed back to RollingUpdate/OnDelete first
- The controller in the older version ignores Recreate and behaves as RollingUpdate for those existing objects

### Version Skew Strategy

This feature has dependencies between control plane components.

1. kube-apiserver v1.xx+1 (feature enabled) and kube-controller-manager v1.xx (no feature)

   - API accepts `type: Recreate`, controller ignores it
   - StatefulSets fall back to default RollingUpdate behavior
   - StatefulSets are functional, just without Recreate strategy feature
   - No errors or warnings

2. kube-apiserver v1.xx (no feature) and kube-controller-manager v1.xx+1 (feature enabled)

  - API server rejects create/update requests that set `type: Recreate` with a validation error
  - Users cannot create or update StatefulSets to use Recreate until apiserver is upgraded and the feature is enabled
  - Enable the feature on kube-apiserver first, then on kube-controller-manager

3. Mixed control plane during rolling upgrade

   - During control plane upgrade, apiservers and controller-managers may have different versions, and the feature may be enabled or disabled. The behavior depends on the leader's version:
     - If leader has feature enabled: Recreate strategy is processed correctly
     - If leader has feature disabled: Recreate strategy is ignored, falls back to RollingUpdate behavior
     - Leader may change during upgrade, causing behavior to switch between Recreate and RollingUpdate

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: StatefulSetRecreateStrategy
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-controller-manager

###### Does enabling the feature change any default behavior?

No. Enabling the `StatefulSetRecreateStrategy` feature gate does not change any default behavior.

The `type: Recreate` strategy is **opt-in**. When not explicitly set:

- StatefulSets behave exactly as they do today (default `RollingUpdate` behavior)
- All existing StatefulSet update strategies continue to work unchanged

The feature only activates when users explicitly configure `spec.updateStrategy.type: Recreate` in their StatefulSet spec.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the feature can be disabled.

###### What happens if we reenable the feature if it was previously rolled back?

The feature works normally again. StatefulSets with `type: Recreate` in their spec will immediately start using Recreate behavior for the next update.

###### Are there any tests for feature enablement/disablement?

No, unit and integration tests will be added to cover feature gate enablement/disablement scenarios.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

**Rollout Failures:**

- If apiserver and controller-manager have different feature gate states, `type: Recreate` may be accepted but ignored (falls back to RollingUpdate)
- API validation accepts `type: Recreate` as valid strategy type (no complex validation needed)

**Rollback Failures:**

- If the strategy type was not changed back, StatefulSets with `type: Recreate` will fall back to RollingUpdate behavior and Recreate behavior will be ignored.

**Impact on Running Workloads:**

- No impact on StatefulSets without `type: Recreate`
- StatefulSets with `type: Recreate` will experience downtime during updates (i.e. all pods are deleted before new ones are created)

###### What specific metrics should inform a rollback?

- `statefulset_unavailable_replicas` shows how many Statefulset replicas are unavailable
- `workqueue_depth{name="statefulset"}` shows the current depth of the StatefulSet controller queue
- `workqueue_queue_duration_seconds{name="statefulset"}` shows how long items wait in queue before processing
- `workqueue_retries_total{name="statefulset"}` shows retry counts which may indicate processing failures

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No, tests will be added to cover upgrade and rollback scenarios.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No. This feature adds a new strategy type `Recreate` to `spec.updateStrategy.type`. No deprecations of existing fields or APIs nor removals of existing functionality.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- By querying StatefulSets using kubectl:

```sh
kubectl get statefulsets -A -o json | \
  jq '.items[] | select(.spec.updateStrategy.type == "Recreate") | 
  {namespace: .metadata.namespace, name: .metadata.name, strategy: .spec.updateStrategy.type}'
```

- By checking StatefulSet status conditions:

```sh
kubectl get statefulsets -A -o json | \
  jq '.items[] | select(.status.conditions[]? | select(.type=="Progressing"))'
```

###### How can someone using this feature know that it is working for their instance?

- [] Events
- [x] API .status
  - Condition name: `Progressing`
- [x] Metrics (existing metrics [kube-state-metrics](https://github.com/kubernetes/kube-state-metrics/blob/release-2.18/docs/metrics/workload/statefulset-metrics.md?plain=1))
  - `kube_statefulset_replicas`
  - `kube_statefulset_status_replicas_ready`
  - `kube_statefulset_status_replicas_current`

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- 100% of StatefulSets without `type: Recreate` behave identically to pre-feature behavior
- 99% of Recreate updates complete within (pod termination time + pod startup time + 30s)
- 0% of pods are left in mixed old/new spec states after Recreate update

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics (existing metrics [kube-state-metrics](https://github.com/kubernetes/kube-state-metrics/blob/release-2.18/docs/metrics/workload/statefulset-metrics.md?plain=1))
  - Metric(s) name: 
    - `kube_statefulset_status_replicas_available`
    - `kube_statefulset_status_replicas_ready`
    - `kube_statefulset_status_replicas_current`
    - Components exposing the metric: kube-state-metrics
  - Metric name: 
    - `statefulset_unavailable_replicas`
    - Components exposing the metric: kube-controller-manager
  - These metrics reflect the StatefulSet `.status` (availableReplicas, readyReplicas, currentReplicas). They have labels `statefulset` and `namespace`, so operators can filter by StatefulSet to monitor a specific StatefulSet during Recreate
  - During Recreate updates, the values show the transition from all pods deleted (0 available) to all new pods created and ready

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No. The existing StatefulSet metrics provide sufficient observability for the Recreate strategy.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new types of API calls. If the feature gate is enabled but no StatefulSet uses `type: Recreate`, then no additional API calls occur.

When Recreate strategy is used during an update, the following existing API call types are made:

- Pod Deletion (DELETE /api/v1/namespaces/{ns}/pods/{name})
- Pod Creation (POST /api/v1/namespaces/{ns}/pods)
- StatefulSet Status Update (PUT /apis/apps/v1/namespaces/{ns}/statefulsets/{name}/status)
- Event Creation (POST /api/v1/namespaces/{ns}/events)

###### Will enabling / using this feature result in introducing new API types?

No. A new strategy type `Recreate` is added to the existing `StatefulSetUpdateStrategyType` enum, but no new API types are introduced.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, minor increases in size when `type: Recreate` is used.

Per StatefulSet using Recreate strategy:

- **Spec**: ~8 bytes (strategy type enum value: "Recreate")
- **Status**: ~150-200 bytes when Progressing condition is active
- **Total**: ~160-210 bytes per StatefulSet

For a cluster with 1000 StatefulSets using Recreate strategy:

- Total increase: ~160-210 KB
- Impact: Negligible compared to typical etcd usage (multi-GB scale)

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

- API Server Operations:
  - GET/LIST StatefulSets: No impact (strategy type is standard enum field, standard deserialization)
  - CREATE/UPDATE StatefulSets: Minimal impact (~10-20μs for validating strategy type enum).

- StatefulSet Controller Reconciliation:
  - With feature enabled but strategy not set to Recreate: No additional overhead.
  - With Recreate strategy: Same overhead as manual pod deletion + creation operations.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

- Etcd Operations:
  - Minimal increase in object size when using Recreate strategy (~8 bytes for strategy type enum value + ~150-200 bytes for status conditions when active).
- Memory/CPU:
  - Memory (per StatefulSet): ~8 bytes for strategy type enum value.
  - CPU: Strategy type comparison on each reconciliation: ~1-2μs (simple string comparison).
- Network I/O:
  - An additional ~8 bytes per StatefulSet spec when Recreate strategy is set, and ~150-200 bytes per status update when Progressing condition is active.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No, the feature does not introduce new node resource exhaustion risks beyond existing mechanism.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The feature behaves similar to existing controllers which depend on API server and etcd availability.

- API Server Unavailable: StatefulSet controller cannot read/write StatefulSet or Pod objects, so all updates halt.
- etcd Unavailable: Similar to API server unavailability, no state changes can be persisted.

No special handling is required as this feature only changes the update progression logic, not the fundamental dependency on API server/etcd availability.

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Examine Metrics (`kube_statefulset_status_replicas_available`, `kube_statefulset_status_replicas_ready`)
   - If `kube_statefulset_status_replicas_available` is stuck at 0 for extended period → pods may be stuck in termination
   - If `kube_statefulset_status_replicas_current` is increasing but `kube_statefulset_status_replicas_ready` is not → pods may be failing to start
2. Check if pods are stuck in termination (long grace periods, finalizers blocking deletion)
3. Verify pod startup time is reasonable (image pull, initialization containers, readiness probes)

## Implementation History

- 2022-09-26: Initial KEP Created
- 2025-07-29: Updated the KEP after changing the ownership  
- 2025-10-13: Pivoted KEP from `EnforcedRollingUpdate` strategy to `podProgressTimeoutSeconds` field based on sig-apps feedback. This approach better handles transient vs permanent failures and aligns with Deployment semantics.
- 2025-12-01: Pivoted KEP from `podProgressTimeoutSeconds` to `Recreate` strategy based on sig-apps meeting ([meeting recording](https://www.youtube.com/watch?v=W7VuKDvAtjg&list=PL69nYSiGNLP2LMq7vznITnpd2Fk1YIZF3)). Key feedback:
  - Progress deadline seconds in Deployments do not terminate pods, but podProgressTimeoutSeconds proposal would terminate pods
  - Deleting/terminating pods based on readiness signals is problematic and disruptive  
  - Group consensus favored Recreate for simplicity and consistency with existing Kubernetes APIs

## Drawbacks

### Downtime Requirement

The Recreate strategy causes downtime during updates since all pods are deleted before new ones are created:

- Service Interruption: Application is completely unavailable during the deletion/recreation window
- Not Suitable for All Workloads: Traditional stateful applications requiring high availability cannot use this strategy
- User Expectation Management: Users must understand and accept downtime implications

Mitigation:

- Clear documentation emphasizing downtime implications
- Explicit opt-in via `type: Recreate` (no accidental usage)
- Recommendation to use for appropriate workloads (CI/CD, stateless apps, development environments)

### Limited Rollback Options

During a Recreate update, there's no gradual rollback:

- If new version has issues, all pods are affected (no gradual detection)
- Cannot compare old vs new pods side-by-side during rollout
- Must wait for full recreation cycle to attempt fixes

Mitigation:

- Clear events and status conditions during Recreate process
- Users can choose RollingUpdate for gradual rollouts where needed
- Quick feedback loop due to fast recreation (all pods start together)

## Alternatives

### Alternative 1: PodProgressTimeoutSeconds Field in RollingUpdate Strategy

Extend the existing `RollingUpdate` strategy with a `podProgressTimeoutSeconds` field (similar to Deployment's `progressDeadlineSeconds`) that allows timeout-based detection of stuck pods.

API Example:

```yaml
spec:
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      podProgressTimeoutSeconds: 600  # Wait 10 minutes per pod
      maxUnavailable: 1
```

Algorithm: For each pod in reverse ordinal order, delete and create new pod, wait for Ready state with timeout. If pod doesn't become Ready within `podProgressTimeoutSeconds`, delete and recreate it.

Pros:

- Maintains sequential ordering guarantees
- Distinguishes transient failures (slow image pulls) from permanent failures (misconfig)
- Works with existing `maxUnavailable` and `partition` fields
- Allows fine-grained control over timeout per workload

Cons:

- Complexity: Requires tracking per-pod creation timestamps and deadline state across reconciliation loops
- Timeout Configuration Burden: Users must choose appropriate timeout values (too short = unnecessary churn, too long = slow recovery)
- Doesn't Solve All Scenarios: Still blocks on transient issues until timeout expires
- Controller Complexity: Adds significant complexity to StatefulSet controller logic

Why Not Chosen as Primary Solution: Based on sig-apps meeting feedback ([meeting link](https://www.youtube.com/watch?v=W7VuKDvAtjg&list=PL69nYSiGNLP2LMq7vznITnpd2Fk1YIZF3)), the group favored the simpler Recreate strategy approach. Key concerns raised:

- Progress deadline in Deployments does **not** terminate pods when deadline is reached, but this proposal would
- Using readiness signals to terminate pods is problematic and disruptive
- The timeout-based approach adds complexity that may not be necessary for the primary use cases (CI/CD, stateless apps, external storage)
- Recreate strategy is "pretty bare" and has direct parallel with Deployment patterns, making it easier to implement and understand

### Alternative 2: EnforcedRollingUpdate Strategy

Add a new update strategy type `EnforcedRollingUpdate` that immediately deletes and replaces stuck pods without timeout during rolling updates.

API Example:

```yaml
spec:
  updateStrategy:
    type: EnforcedRollingUpdate
    enforcedRollingUpdate:
      maxUnavailable: 1
```

Algorithm: When `pod[i]` needs update, delete it immediately regardless of current state, create new pod, wait for Ready.

Pros:

- Simpler than timeout-based approach (no deadline tracking)
- Maintains some ordering through sequential updates
- Immediate action on stuck pods

Cons:

- Cannot distinguish transient from permanent failures (network delays, CI/CD pipeline delays, slow image pulls)
- Still maintains sequential ordering, which adds complexity
- Doesn't solve initial deployment failure, only works when spec changes

Why Not Chosen: Similar concerns as Alternative 1, but Recreate is even simpler by removing ordering requirements entirely.

### Alternative 3: (Now Primary Solution): Recreate Strategy

NOTE: This alternative was chosen as the primary solution for this KEP based on sig-apps meeting feedback.

Add a `Recreate` update strategy (matching Deployment's Recreate strategy) that deletes all pods before creating new ones.

API Example:

```yaml
spec:
  updateStrategy:
    type: Recreate
```

Algorithm: Delete all pods, wait for termination, create all new pods according to `spec.podManagementPolicy`.

Pros:

- No complexity around stuck pods or timeout tracking
- All pods deleted before new ones created, guaranteeing clean state
- Simple, predictable behavior aligned with Deployment patterns
- Can quickly replace all pods regardless of their current state
- No need to configure timeouts or tune parameters

Cons:

- No ordering during deletion (all at once). Ordering during creation only when podManagementPolicy
- Not suitable for traditional stateful workloads requiring zero-downtime updates

Why Chosen as Primary Solution: Based on sig-apps meeting discussion, this approach is:

- Simpler to implement and understand (matches existing Deployment Recreate pattern)
- Addresses the primary use cases (CI/CD, stateless apps, external storage, LeaderWorkerSet)  
- Avoids concerns about terminating pods based on readiness/timeout signals
- Provides explicit opt-in behavior where users accept downtime for automated recovery

### Alternative 4: Add Force Flag to RollingUpdate

Add a boolean field like `spec.updateStrategy.rollingUpdate.forceUpdate: true`.

Pros:

- Minimal API change

Cons:

- Same issue as Alternative 1; cannot distinguish transient from permanent failures
- Less discoverable than dedicated field
- Boolean flag doesn't allow tuning timeout per workload

Why Not Chosen: Recreate strategy is clearer about behavior and simpler to implement.

### Alternative 5: Enhance Parallel Policy

Extend `podManagementPolicy: Parallel` to automatically replace stuck pods during updates.

Pros:

- Reuses existing field
- Already has parallel semantics

Cons:

- Loses sequential ordering guarantees
- Confuses semantics of `podManagementPolicy` (affects both scaling and updates) vs `updateStrategy` (updates only)
- Less explicit than dedicated strategy type
- Doesn't automatically delete all pods for clean state

Why Not Chosen: Recreate strategy as a dedicated update strategy type is clearer and more explicit. It also aligns better with Deployment patterns.

## Infrastructure Needed (Optional)

N/A
