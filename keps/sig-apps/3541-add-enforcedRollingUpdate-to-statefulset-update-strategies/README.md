# KEP-3541: Add podProgressTimeoutSeconds to StatefulSet RollingUpdate Strategy

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
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Risk: Unintended Data Loss](#risk-unintended-data-loss)
- [Design Details](#design-details)
  - [Detailed Algorithm Specification](#detailed-algorithm-specification)
    - [Current RollingUpdate Algorithm](#current-rollingupdate-algorithm)
    - [Proposed RollingUpdate with podProgressTimeoutSeconds Algorithm](#proposed-rollingupdate-with-podprogresstimeoutseconds-algorithm)
  - [API Changes](#api-changes)
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
  - [Timeout Configuration Complexity](#timeout-configuration-complexity)
  - [Potential for Misuse](#potential-for-misuse)
  - [Implementation Complexity](#implementation-complexity)
- [Alternatives](#alternatives)
  - [Alternative 1: EnforcedRollingUpdate Strategy](#alternative-1-enforcedrollingupdate-strategy)
  - [Alternative 2: Recreate Strategy](#alternative-2-recreate-strategy)
  - [Alternative 3: Add Force Flag to RollingUpdate](#alternative-3-add-force-flag-to-rollingupdate)
  - [Alternative 4: Enhance Parallel Policy](#alternative-4-enhance-parallel-policy)
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

This KEP extends the existing `RollingUpdate` strategy with a `podProgressTimeoutSeconds` field (similar to [`progressDeadlineSeconds`](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#failed-deployment) in Deployments), that works alongside `maxUnavailable`, `partition`, and `podManagementPolicy`, which allows StatefulSets to automatically detect and replace pods that fail to become Ready within a configurable timeout. This distinguishes transient failures (network delays, slow image pulls) from permanent failures (configuration errors), enabling automated recovery while maintaining sequential ordering and backward compatibility.

## Motivation

### Current Behavior and Problems

StatefulSets with `RollingUpdate` strategy follow this algorithm:

**1. if `podManagementPolicy: OrderedReady` (default)**:
1. Update pods in reverse ordinal order (N-1, N-2, ..., 0)
2. For each pod, wait until it becomes Running and Ready before proceeding to the next
3. If any pod fails to become Ready, the entire update process halts
4. Even when a corrected configuration is applied, stuck pods are never automatically replaced

**2. if `podManagementPolicy: Parallel`**:
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

**1. [MaxUnavailable](https://github.com/kubernetes/enhancements/issues/961) Doesn't Address the Core Issue**
The `maxUnavailable` option in `RollingUpdate` strategy allows multiple pods to be updated simultaneously, but its behavior depends on `podManagementPolicy`:

```yaml
spec:
  podManagementPolicy: OrderedReady
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 2  # Can update 2 pods at once
```

For the `podManagementPolicy: OrderedReady`, even with `maxUnavailable: 2`, if any pod fails to reach Ready state, the rolling update process still halts completely. The controller waits indefinitely for stuck pods to become Ready, even after applying a fix.

**Example Scenario with `podManagementPolicy: OrderedReady`**:

- StatefulSet with 5 replicas, `maxUnavailable: 2`
- Update pods `app-4` and `app-3` simultaneously
- `app-4` gets stuck in `ImagePullBackOff`
- Even after fixing the image name, `app-4` remains stuck
- Update process cannot proceed to `app-2`, `app-1`, or `app-0`
- Manual intervention still required: `kubectl delete pod app-4`

**2. Parallel Policy Workaround**
Setting `podManagementPolicy: Parallel` doesn't solve the core issue:

- While stuck pods don't block the update process from proceeding, they are never automatically replaced
- Stuck pods remain in broken state indefinitely, even after applying a corrected configuration
- Sequential ordering guarantees are lost (undesirable for many use cases)
- Parallel policy affects scaling behavior, not just updates
- Manual intervention is still required to delete and replace stuck pods

**3. Custom Controllers**
Some teams have built custom controllers to delete stuck pods, but this:

- Duplicates StatefulSet controller logic
- Creates maintenance burden
- May conflict with StatefulSet controller behavior
- Lacks integration with StatefulSet status and events

### Proposed Solution Benefits

Adding `podProgressTimeoutSeconds` to StatefulSet `RollingUpdate` addresses these issues by:

1. **Automated Recovery**: Pods that fail to become Ready within the deadline are automatically deleted and recreated without manual intervention
2. **Transient vs Permanent Failure Detection**: Timeout distinguishes temporary issues from permanent failures
3. **Safety Preservation**: Sequential ordering is maintained; only Ready pods allow progression to next ordinal
4. **Works with Existing Features**: Compatible with `maxUnavailable`, `podManagementPolicy`, and `minReadySeconds`

### Goals

1. Extend `RollingUpdate` strategy with `podProgressTimeoutSeconds` field to enable timeout-based detection of stuck pods
2. Automatically replace pods that fail to become Ready within the configured deadline
3. Maintain sequential ordering guarantees for StatefulSet updates
4. Provide clear status conditions and events when progress deadline is exceeded (similar to Deployments)
5. Support both `OrderedReady` and `Parallel` pod management policies

### Non-Goals

1. Change default behavior of StatefulSet updates (opt-in via `podProgressTimeoutSeconds` configuration)
3. Add health-based condemnation without timeout (pods are only replaced after deadline)
4. Replace Deployment-style revision management (StatefulSets continue to directly manage Pods)

## Proposal

### User Stories

#### Story 1: CI/CD Platform Team

**Context**: A platform team manages hundreds of StatefulSet deployments across development and staging environments. Their CI/CD system requires end-to-end automation, but StatefulSet rolling updates break automation when pods get stuck. The team either has to implement custom "garbage collection" logic or accept that automated deployments will fail and require manual intervention.

**Solution**: With `podProgressTimeoutSeconds: 600` configured, the StatefulSet controller would wait up to 10 minutes for the pod to become Ready. If the pod remains stuck, it's automatically deleted and recreated. When the fixed configuration is applied, the new pod starts successfully, allowing the CI/CD pipeline to complete without manual intervention.

#### Story 2: Stateless Web Application

**Context**: A web application uses StatefulSet for predictable pod naming and ordered startup, but doesn't store critical data locally. When resource limit typos cause pods to get stuck in Pending state, the entire update halts even though pod replacement is safe.

**Solution**: With `podProgressTimeoutSeconds` configured, the controller waits for the configured timeout before automatically replacing stuck pods. This eliminates the need for manual pod deletion in environments where data loss is not a concern, while still allowing time for legitimate resource provisioning delays.

#### Story 3: Development/Experiment Environment

**Context**: Developers using StatefulSet for experiments face constant frustration - every time a rolling update breaks due to configuration errors, they must manually delete stuck pods after applying fixes. This manual intervention disrupts the development workflow.

**Solution**: With `podProgressTimeoutSeconds: 300` (5 minutes), developers get fast feedback - if a pod doesn't become Ready within the timeout, it's automatically replaced. This enables a smoother development experience without requiring cluster operator intervention.

#### Story 4: External Data Storage

**Context**: A database application stores all persistent data on network-attached storage (not local pod storage). Pod replacement is completely safe since no local data would be lost, but the StatefulSet controller treats it as a traditional stateful workload and requires manual intervention.

**Solution**: With `podProgressTimeoutSeconds` configured, the controller automatically replaces pods that fail to start within the timeout, which is safe for this architecture since all data persists externally.

#### Story 5: LeaderWorkerSet (LWS) Use Case

**Context**: Developers use StatefulSet as the high-level controller workload for [LWS](https://github.com/kubernetes-sigs/lws). However, it behaves more like a Deployment - there's no ordering dependency between different replicas. They only need the ordinal index for pod identification. When a replica fails during updates, the entire StatefulSet update gets stuck, even though there's no actual ordering requirement between replicas.

**Solution**: With `podProgressTimeoutSeconds` and `podManagementPolicy: Parallel`, failed replicas are automatically replaced after the timeout without blocking other replicas. This enables automated recovery for deployment-like workloads that use StatefulSet for pod identity rather than traditional stateful ordering.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

#### Risk: Unintended Data Loss

**Risk Description**: If `podProgressTimeoutSeconds` is misconfigured on StatefulSets with local persistent data, automatic pod replacement could cause data loss. We should document this.

**Mitigation Strategies**:

1. **Documentation**: Clear guidance on when to use `podProgressTimeoutSeconds` and recommended timeout values for different workload types
2. **No Default Value**: Opt-in behavior - existing workloads continue using safe `RollingUpdate` with indefinite wait (current behavior)
3. **Clear Events**: Events emitted with `ProgressDeadlineExceeded` reason when pods are replaced, including elapsed time
4. **Status Conditions**: `Progressing=False` condition clearly indicates when deadline is exceeded
5. **Validation**: API validation prevents clearly dangerous configurations (e.g., `podProgressTimeoutSeconds < minReadySeconds`)

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

**Problem**: The algorithm halts when `pod[i]` is not Running or Ready, even if a fix is applied.

#### Proposed RollingUpdate with podProgressTimeoutSeconds Algorithm

```
FOR i = replicas-1 To i >= 0 DO i-- 
    If pod[i] needs update THEN
        wait_for_predecessors_ready(i+1 to replicas-1)
        
        // Check if pod is in a state that prevents update
        If pod[i] exists AND (!pod[i].Running OR !pod[i].Ready) THEN
            // NEW: Check if podProgressTimeoutSeconds is configured and exceeded
            If podProgressTimeoutSeconds is configured THEN
                elapsed = current_time - pod[i].creation_time
                If elapsed > podProgressTimeoutSeconds THEN
                    // Deadline exceeded - delete and recreate
                    delete_pod(i)
                    emit_event("ProgressDeadlineExceeded", pod[i])
                    set_condition("Progressing", status="False", reason="ProgressDeadlineExceeded")
                ELSE
                    // Still within deadline - wait and retry
                    return  // Check again on next reconciliation
                ENDIF
            ELSE
                // No deadline configured - use current behavior (wait forever)
                return
            ENDIF
        ENDIF
        
        // Proceed with normal update
        If pod[i] exists THEN
            delete_pod(i)
        ENDIF
        create_pod(i)
        record_pod_creation_time(pod[i])  // Track for deadline
        wait_until_ready(pod[i])  // Wait for new pod to be ready (with deadline check)
        
    ENDIF
ENDFOR
```

**Key Differences from Current Behavior**:

1. **Timeout-Based Detection**: Waits for `podProgressTimeoutSeconds` before considering a pod permanently stuck
2. **Transient Failure Tolerance**: Network delays, slow image pulls, CI/CD pipeline delays are tolerated within the deadline
3. **Automatic Replacement**: After deadline, pod is deleted and recreated (similar to manual `kubectl delete pod`)
4. **Status Conditions**: Sets `Progressing=False` with `ProgressDeadlineExceeded` reason (matches Deployment behavior)
5. **Events**: Emits clear events when deadline is exceeded for observability

### API Changes

```go
// RollingUpdateStatefulSetStrategy is used to communicate parameter for RollingUpdateStatefulSetStrategyType.
type RollingUpdateStatefulSetStrategy struct {
    // Partition indicates ordinal at which the StatefulSet should be partitioned.
    // Default value is 0.
    // +optional
    Partition *int32 `json:"partition,omitempty"`
    
    // The maximum number of pods that can be unavailable during the update.
    // Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%).
    // Absolute number is calculated from percentage by rounding up. This can not be 0.
    // Defaults to 1. This field is alpha-level and is only honored by servers that enable the
    // MaxUnavailableStatefulSet feature. The field applies to all pods in the range 0 to
    // Replicas-1. That means if there is any unavailable pod in the range 0 to Replicas-1, it
    // will be counted towards MaxUnavailable.
    // +optional
    MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
    
    // NEW: PodProgressTimeoutSeconds specifies the maximum time in seconds for a pod to
    // become Ready during a rolling update before it is considered stuck and automatically
    // replaced. This field is optional. When not specified, pods will wait indefinitely
    // (current behavior). Similar to Deployment's podProgressTimeoutSeconds, but applies
    // per-pod rather than to the entire rollout.
    //
    // The deadline is measured from pod creation time. If a pod does not become Ready
    // within this duration, it will be deleted and recreated, and a Progressing condition
    // with status=False and reason=ProgressDeadlineExceeded will be added to the StatefulSet.
    //
    // This field works with both OrderedReady and Parallel pod management policies:
    // - OrderedReady: Each pod must become Ready (or exceed deadline) before next pod updates
    // - Parallel: All pods update simultaneously, each with independent deadline tracking
    //
    // Minimum value is 1. Must be greater than minReadySeconds if both are specified.
    // +optional
    PodProgressTimeoutSeconds *int32 `json:"podProgressTimeoutSeconds,omitempty"`
}

// StatefulSetUpdateStrategy indicates the strategy that the StatefulSet
```
**Example Usage**:

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: web
spec:
  replicas: 10
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      podProgressTimeoutSeconds: 600  # Wait 10 minutes for each pod to become Ready before replacing stuck ones
      maxUnavailable: 3 # Update 3 pods at a time
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
```

**Behavior**:
- Pods 9, 8, 7 start updating simultaneously (`maxUnavailable: 3`)
- Each pod can `podProgressTimeoutSeconds: 600`, wait up to 10 minutes for pod to become Ready
- If pod 9 exceeds timeout after 600s, it's deleted and recreated
- Meanwhile, pods 8 and 7 can still be progressing normally
- Pod must be Ready for `minReadySeconds`, deadline applies to reaching Ready state
- Only after 3 pods are Ready (or replaced) do pods 6, 5, 4 start updating

### Implementation Changes

The implementation requires changes to the StatefulSet controller in `pkg/controller/statefulset/stateful_set_control.go`:

1. **Deadline Tracking**:
   - Track pod creation timestamps in StatefulSet controller state
   - On each reconciliation loop, check elapsed time since pod creation
   - Compare elapsed time against `spec.updateStrategy.rollingUpdate.podProgressTimeoutSeconds`

2. **Pod Update Logic Enhancement**:
   - **Before**: When a pod is not Running/Ready, controller returns immediately (stuck forever)
   - **After**: When `podProgressTimeoutSeconds` is configured:
     - If `elapsed_time < podProgressTimeoutSeconds`: Continue waiting (return for next reconciliation)
     - If `elapsed_time >= podProgressTimeoutSeconds`: Delete pod and emit event
   - **Without configuration**: Maintain current behavior (wait indefinitely)

3. **Status Condition Management**:
   - Add `Progressing` condition to StatefulSet status (similar to Deployment)
   - Set `status: "False"` and `reason: ProgressDeadlineExceeded` when deadline exceeded
   - Reset condition to `status: "True"` when pod becomes Ready or new update starts

4. **Event Emission**:
   - Emit `ProgressDeadlineExceeded` event when deleting stuck pod
   - Include pod name, elapsed time, and configured deadline in event message
   - Emit `ProgressingResumed` event when previously stuck pod is replaced successfully

5. **Validation**:
   - Validate `podProgressTimeoutSeconds >= 1` (minimum 1 second)
   - Validate `podProgressTimeoutSeconds > minReadySeconds` if both specified
   - API validation in `pkg/apis/apps/validation/validation.go`

6. **Ordering Guarantees**:
   - Maintain existing ordering behavior: highest ordinal pods updated first
   - With `OrderedReady`: Wait for predecessor pods (or their deadlines) before proceeding
   - With `Parallel`: Track deadlines independently for each pod
   - Safety check: If highest ordinal pod exceeds deadline repeatedly, halt progression (prevent cascading failures)

### Comparison with Existing Solutions

| Solution                           |podManagementPolicy|  Sequential Ordering | Automatic Recovery | Transient Failure Handling | Behavior When Pod Stuck           | Use Case                               |
| ---------------------------------- | ------------------- | ----|------------------ | -------------------------- | --------------------------------- | -------------------------------------- |
| `RollingUpdate` (default) | OrderedReady          | Yes )  | No                 | No (waits forever)         | Halts completely, waits forever   | Traditional stateful apps              |
| `RollingUpdate` + `maxUnavailable` | OrderedReady |Yes   | No                 | No (waits forever)         | **Still halts completely**        | Faster updates, but same stuck problem |
| `RollingUpdate` + `Parallel` | Parallel | No  | No                 | No (pods remain broken)         | Pods update but stay broken        | Faster updates, but same stuck problem |
| `OnDelete`  | Parallel or OrderedReady                       | Yes (manual)        | No                 | No (manual control)        | Fully manual control              | Maximum safety/control                 |
| `Parallel` + `maxUnavailable`      | Parallel|No                  | No                 | No (pods remain broken)    | Pods update but stay broken       | Fast updates, manual cleanup needed    |
| `RollingUpdate` + `podProgressTimeoutSeconds` | OrderedReady | Yes | Yes | Yes (timeout-based) | Waits for deadline, then recreates | Automated recovery with ordering |
| `RollingUpdate` + `podProgressTimeoutSeconds` | Parallel | No | Yes | Yes (timeout-based) | Waits for deadline, then recreates | Fast automated recovery |
| `RollingUpdate` + `podProgressTimeoutSeconds` + `maxUnavailable` | OrderedReady | Yes (batched) | Yes | Yes (timeout-based) | Batch updates with per-pod timeouts | Fast automated recovery with ordering |

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

- **Without `podProgressTimeoutSeconds`**: Broken StatefulSet will not recover even after applying a fixed configuration (current behavior preserved)
- **With `podProgressTimeoutSeconds` configured**: 
  - Pod that fails to become Ready within deadline is automatically deleted and recreated
  - Status condition `Progressing=False` with `reason=ProgressDeadlineExceeded` is set
  - Event `ProgressDeadlineExceeded` is emitted with pod name and elapsed time
  - After replacement, if pod becomes Ready, condition is reset to `Progressing=True`
- **Transient failure handling**: Pod that becomes Ready within deadline is NOT replaced (e.g., slow image pull)
- **With `podManagementPolicy: OrderedReady`**: Pods are updated sequentially, each respecting its own deadline
- **With `podManagementPolicy: Parallel`**: Multiple pods can exceed deadlines independently
- **Safety check**: If highest ordinal pod repeatedly fails to become Ready even after replacement, lower ordinals are not updated
- **Validation**: API validation rejects `podProgressTimeoutSeconds < 1` and `podProgressTimeoutSeconds <= minReadySeconds`

##### e2e tests

The following e2e tests will be added to `test/e2e/apps/statefulset.go`:

- StatefulSet with `podProgressTimeoutSeconds` successfully recreates stuck pods after timeout
- Timeout works with `podManagementPolicy: OrderedReady`
- Timeout works with `podManagementPolicy: Parallel`
- Timeout works with `maxUnavailable`
- Pods that become Ready within timeout are not recreated
- StatefulSets without `podProgressTimeoutSeconds` maintain current behavior (backward compatibility)

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

This feature is protected by the feature-gate `StatefulSetProgressDeadline`, which must be enabled on both `kube-apiserver` and `kube-controller-manager`.

**Component Dependencies**
- **kube-apiserver**: Validates and persists the `podProgressTimeoutSeconds` field in the StatefulSet spec
- **kube-controller-manager**: Implements the timeout tracking and pod recreation logic

**Upgrade Sequence**

1. Enable feature gate on kube-apiserver first
2. Enable feature gate on kube-controller-manager
3. Create/update StatefulSets with `podProgressTimeoutSeconds` field

**Partial Upgrade Behavior**
- If apiserver has feature enabled but kube-controller-manager does not:
  - API server accepts `podProgressTimeoutSeconds` field
  - Field is persisted in etcd
  - Kube-controller-manager ignores the field, which reverts to current behavior (waits indefinitely)
  - No errors, but timeout behavior is not active
  
- If apiserver does NOT have feature enabled but kube-controller-manager does:
  - API server rejects StatefulSets with `podProgressTimeoutSeconds` field
  - Kube-controller-manager cannot process timeout

> Enable the feature gate on `kube-apiserver` first, then `kube-controller-manager` to ensure smooth transition.

#### Downgrade

If you configured `podProgressTimeoutSeconds`, when downgrading to a version without this feature,
you should remove `podProgressTimeoutSeconds` field from your StatefulSet spec or the API server will reject the update.

If the field is not removed before downgrade:
- Older API server will reject updates to StatefulSets containing `podProgressTimeoutSeconds`
- Existing StatefulSets with the field in etcd may cause reconciliation warnings/errors
- Manual cleanup required to edit StatefulSet resources to remove the field


### Version Skew Strategy

This feature has dependencies between control plane components.

**1. kube-apiserver v1.35 (feature enabled) and kube-controller-manager v1.34 (no feature)**:
   - API accepts field, controller ignores it
   - StatefulSets wait indefinitely (current behavior)
   - StatefulSets are functional, just without timeout feature

**2. kube-apiserver v1.34 (no feature) and kube-controller-manager v1.35 (feature enabled)**:
   - API rejects `podProgressTimeoutSeconds` field
   - This violates version skew policy (controller ahead of apiserver)

**3. Mixed control plane during rolling upgrade**:
  - During control plane upgrade, some apiservers may have feature enabled, others not
   - StatefulSet updates may succeed or fail depending on which apiserver handles the request

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: StatefulSetProgressDeadline
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-controller-manager

###### Does enabling the feature change any default behavior?

No. Enabling the `StatefulSetProgressDeadline` feature gate does not change any default behavior.

The `podProgressTimeoutSeconds` field is **optional** and has no default value. When the field is not specified:
- StatefulSets behave exactly as they do today (wait indefinitely for pods to become Ready)
- All existing StatefulSet update strategies continue to work unchanged

The feature only activates when users explicitly configure `spec.updateStrategy.rollingUpdate.podProgressTimeoutSeconds` in their StatefulSet spec.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the feature can be disabled, but requires cleanup of affected resources first for a clean rollback. This should be done as follow:
- Remove the field from all StatefulSets
- Disable feature gate on kube-controller-manager
- Disable feature gate on kube-apiserver

###### What happens if we reenable the feature if it was previously rolled back?

The feature works normally again. Behavior depends on whether cleanup was performed during rollback.

If the cleanup was performed, feature gate can be re-enabled without issues and StatefulSets behave normally (wait indefinitely, no timeout). While if the cleanup was not performed (field still in etcd), StatefulSets with `podProgressTimeoutSeconds` in etcd immediately start using timeout behavior, kube-apiserver accepts the field again in updates, and kube-controller-manager reads existing values and begins tracking timeouts. If pods have been waiting for a longer time than the configured timeout, they may be recreated.

###### Are there any tests for feature enablement/disablement?

Yes, unit and integration tests will cover feature gate enablement/disablement scenarios.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

**Rollout Failures:**
- If apiserver and controller-manager have different feature gate states, field may be accepted but ignored
- API validation rejects invalid `podProgressTimeoutSeconds` values (i.e. < 1, or < minReadySeconds) (no workload impact)

**Rollback Failures:**
- if the field was not removed, StatefulSets with `podProgressTimeoutSeconds` cannot be updated after feature is disabled and will require manual cleanup

**Impact on Running Workloads:**
- No impact on StatefulSets without `podProgressTimeoutSeconds` or pods that are Running and Ready
- Potential impact if the timeout set too low causes pods repeatedly deleted/recreated
- Only pods failing to become Ready during updates are affected; feature does not delete healthy pods

###### What specific metrics should inform a rollback?
- `statefulset_unavailable_replicas` shows how many Statefulset replicas are unavailable

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Yes, these scenarios will be covered in unit and integration tests.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?
No. This feature adds new optional field `spec.updateStrategy.rollingUpdate.podProgressTimeoutSeconds`. No deprecations of existing fields or APIs nor removals of existing functionality.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- By querying StatefulSets using kubectl:
```sh
kubectl get statefulsets -A -o json | \
  jq '.items[] | select(.spec.updateStrategy.rollingUpdate.podProgressTimeoutSeconds != null) | 
  {namespace: .metadata.namespace, name: .metadata.name, timeout: .spec.updateStrategy.rollingUpdate.podProgressTimeoutSeconds}'
```
- By checking StatefulSet status conditions:
```sh
kubectl get statefulsets -A -o json | \
  jq '.items[] | select(.status.conditions[]? | select(.type=="Progressing" and .reason=="PodProgressTimeoutExceeded"))'
```
- By monitoring events:
```sh
kubectl get events -A --field-selector reason=PodProgressTimeoutExceeded
```

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event Reason: `PodProgressTimeoutExceeded` - emitted when a pod exceed the `podProgressTimeoutSeconds`
- [x] API .status
  - Condition name: `PodProgressTimeoutExceeded` with status `False`
- [x] Metrics
  - `statefulset_unavailable_replicas` - tracks how many Statefulset replicas are unavailable

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- 99% of pods that fail to become Ready are recreated within `podProgressTimeoutSeconds + 30s`  
- 0% of pods that become Ready within `podProgressTimeoutSeconds` are deleted
- 100% of StatefulSets without `podProgressTimeoutSeconds` behave identically to pre-feature behavior


###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: **existing** `statefulset_unavailable_replicas`
  - Components exposing the metric: kube-controller-manager

###### Are there any missing metrics that would be useful to have to improve observability of this feature?
No.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

If the feature gate has been enabled but no StatefulSet use/set the new field, then No new API calls. Additional API calls occur when timeouts are triggered, but these are the same types of calls already made by StatefulSet controller:

1. Pod Deletion (DELETE /api/v1/namespaces/{ns}/pods/{name})
2. StatefulSet Status Update (PUT /apis/apps/v1/namespaces/{ns}/statefulsets/{name}/status)
3. Event Creation (POST /api/v1/namespaces/{ns}/events)

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, minor increases in size when feature is used. When the StatefulSet spec is configured, the object will be increased by ~40 bytes per StatefulSet, in addition of ~150 bytes per StatefulSet when condition is set (when timeout triggered).

Per StatefulSet using feature:
- Spec: +40 bytes (one-time, when configured)
- Status: +150 bytes (when condition set)
- Total: ~190 bytes per StatefulSet

For a cluster with 1000 StatefulSets using this feature:
- Total increase: ~190 KB
- Impact: Negligible compared to typical etcd usage


###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

- API Server Operations:
  - GET/LIST StatefulSets: No impact (field is optional, standard deserialization)
  - CREATE/UPDATE StatefulSets: Minimal impact (~10-20μs for validating one additional int32 field). **Impact: Negligible**

- StatefulSet Controller Reconciliation:
  - With feature enabled but not triggered: additional ~1-5μs per reconciliation loop. **Impact Negligible**
  - With feature enabled and timeout triggered: same overhead as manual pod deletion. **Impact: None**

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

- Etcd Operations:
  - Minimal increase in object size (~200 bytes per StatefulSet). **Impact: Negligible** 
- Memory/CPU:
  - Memory (per StatefulSet): ~8 bytes for int32 + ~16 bytes for timestamp tracking. **Impact: Negligible**
  - CPU: timestamp comparison on each reconciliation: ~1-5μs. **Impact: Negligible** 
- Network I/O:
  - An additional of ~40 bytes per StatefulSet spec when field is set, and ~150 bytes per status update when condition set. **Impact: Negligible**

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No, the feature does not introduce new node resource exhaustion risks beyond existing mechanism.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The feature behaves similar to existing controllers which depend on API server and etcd availability.

- **API Server Unavailable**: StatefulSet controller cannot read/write StatefulSet or Pod objects, so all updates halt.
- **etcd Unavailable**: Similar to API server unavailability, no state changes can be persisted.

No special handling is required as this feature only changes the update progression logic, not the fundamental dependency on API server/etcd availability.

###### What are other known failure modes?
N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check StatefulSet Status and Events
2. Examine Metrics
3. Check if `podProgressTimeoutSeconds` value is appropriate for the workload
  

## Implementation History

- 2022-09-26: Initial KEP Created
- 2025-07-29: Updated the KEP after changing the ownership  
- 2025-10-13: Pivoted KEP from `EnforcedRollingUpdate` strategy to `podProgressTimeoutSeconds` field based on sig-apps feedback. This approach better handles transient vs permanent failures and aligns with Deployment semantics.

## Drawbacks

### Timeout Configuration Complexity

Users need to choose appropriate `podProgressTimeoutSeconds` values:

- **Too short**: May delete pods during legitimate slow starts (image pulls, initialization)
- **Too long**: Delays recovery from permanent failures

### Potential for Misuse

Users might configure `podProgressTimeoutSeconds` without understanding the data safety implications:

- **Risk**: Automatic pod replacement could cause data loss in stateful applications with local data
- **Risk**: Loss of debugging opportunities when pods are quickly replaced
- **Risk**: Masking underlying infrastructure or configuration issues

Mitigation: 
- Clear documentation on when to use this feature
- No default value (opt-in behavior)
- Warning events when deadline is exceeded
- Recommendation to use with external storage or stateless workloads

### Implementation Complexity

Requires tracking per-pod creation timestamps and deadline state across reconciliation loops, adding complexity to the StatefulSet controller.

## Alternatives

### Alternative 1: EnforcedRollingUpdate Strategy

Add a new update strategy type `EnforcedRollingUpdate` that immediately deletes and replaces stuck pods without timeout.

**API Example**:
```yaml
spec:
  updateStrategy:
    type: EnforcedRollingUpdate  # New strategy type
    enforcedRollingUpdate:
      maxUnavailable: 1
```

**Algorithm**: When `pod[i]` needs update, delete it immediately regardless of current state, create new pod, wait for Ready.

**Pros**:
- Simpler algorithm (no deadline tracking)
- Clearer semantic separation (distinct strategy type)
- Immediate action on stuck pods

**Cons**:
- Cannot distinguish transient from permanent failures (network delays, CI/CD pipeline delays, slow image pulls)
- Doesn't solve initial deployment failure, only works when spec changes 

**Why Not Chosen**: The inability to distinguish transient failures from permanent ones is a critical flaw. As noted in sig-apps discussion, this would cause unnecessary pod churn during normal operations (slow registry, CI/CD delays).

### Alternative 2: Recreate Strategy

Add a `Recreate` update strategy (matching Deployment's Recreate strategy) that deletes all pods before creating new ones.

**API Example**:
```yaml
spec:
  updateStrategy:
    type: Recreate  # New strategy type
```

**Algorithm**: Delete all pods, wait for termination, create all new pods simultaneously.

**Pros**:
- No complexity around stuck pods
- All pods deleted before new ones created
- Can quickly replace all pods with previous version

**Cons**:
- Total downtime, since all pods deleted simultaneously
- Loses ordering guarantees for no sequential update
- Doesn't address the core problem, which is `automated recovery with ordering`

**Why Not Chosen as Primary Solution**: While useful for some use cases (e.g., LeaderWorkerSet), it doesn't address the primary pain point of wanting automated recovery while maintaining sequential ordering.

### Alternative 3: Add Force Flag to RollingUpdate

Add a boolean field like `spec.updateStrategy.rollingUpdate.forceUpdate: true`.

**Pros**:
- Minimal API change

**Cons**:
- Same issue as Alternative 1; cannot distinguish transient from permanent failures
- Less discoverable than dedicated field
- Boolean flag doesn't allow tuning timeout per workload

**Why Not Chosen**: Timeout-based approach is more flexible and handles transient failures.

### Alternative 4: Enhance Parallel Policy

Extend `podManagementPolicy: Parallel` to automatically replace stuck pods.

**Cons**:
- Loses sequential ordering guarantees
- Confuses semantics of `podManagementPolicy` (scaling) vs `updateStrategy` (updates)

**Why Not Chosen**: Sequential ordering is a key requirement for many StatefulSet use cases.

## Infrastructure Needed (Optional)
N/A