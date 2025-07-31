# KEP-3541: Add enforcedRollingUpdate to StatefulSet update strategies

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Current Behavior and Problems](#current-behavior-and-problems)
  - [Real-World Impact](#real-world-impact)
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
    - [Proposed EnforcedRollingUpdate Algorithm](#proposed-enforcedrollingupdate-algorithm)
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
  - [Increased Complexity](#increased-complexity)
  - [Potential for Misuse](#potential-for-misuse)
  - [Maintenance Burden](#maintenance-burden)
  - [Breaking Traditional StatefulSet Guarantees](#breaking-traditional-statefulset-guarantees)
- [Alternatives](#alternatives)
  - [Alternative 1: Enhance Parallel Policy](#alternative-1-enhance-parallel-policy)
  - [Alternative 2: Add Force Flag to RollingUpdate](#alternative-2-add-force-flag-to-rollingupdate)
  - [Alternative 3: Custom Resource/Operator Approach](#alternative-3-custom-resourceoperator-approach)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
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

StatefulSets currently offer two update strategies: `OnDelete` (manual) and `RollingUpdate` (automatic, default). When using `RollingUpdate`, StatefulSets follow sequential ordering where each individual pod must be Running and Ready before the controller proceeds to update the next pod. Even with the `maxUnavailable` option (which allows multiple pods to be updated simultaneously), the controller still requires each pod to reach Ready state before moving forward - stuck pods halt the entire update process. This design ensures data safety for stateful workloads but creates a critical operational problem.

**The Problem**: When a StatefulSet update results in pods that fail to reach Ready state (due to configuration errors, resource constraints, or other issues), the rolling update process becomes permanently stuck. Even after applying a corrected configuration, the controller will not automatically replace the broken pods, requiring manual intervention to delete stuck pods.

**Community Impact**: This behavior has generated significant user frustration across multiple GitHub issues ([#67250](https://github.com/kubernetes/kubernetes/issues/67250), [#60164](https://github.com/kubernetes/kubernetes/issues/60164), [#109597](https://github.com/kubernetes/kubernetes/issues/109597)) with users reporting:

- Broken CI/CD pipelines requiring manual intervention
- Inability to automatically recover from configuration mistakes
- Operational burden in managing stateful applications

**Proposed Solution**: This KEP introduces `EnforcedRollingUpdate`, a new update strategy that automatically replaces broken pods during rolling updates while maintaining sequential ordering for safety. This provides an escape hatch for operators who need automated recovery while preserving the option for manual control in sensitive environments.

## Motivation

### Current Behavior and Problems

StatefulSets with `RollingUpdate` strategy follow this algorithm:

1. Update pods in reverse ordinal order (N-1, N-2, ..., 0)
2. For each pod, wait until it becomes Running and Ready before proceeding
3. If any pod fails to become Ready, the entire update process halts
4. Even when a corrected configuration is applied, stuck pods are never automatically replaced

The current approach was designed for stateful workloads where:

- Data persistence is critical
- Pod identity and storage are tightly coupled
- Automatic pod deletion could cause data loss
- Manual intervention ensures careful data recovery

**The pros for the current approach**: Many StatefulSet use cases don't require this level of manual intervention:

- **Stateless workloads using StatefulSet for pod identity** (web servers, workers)
- **Applications with external data storage** (databases with network-attached storage)
- **CI/CD environments** where automated recovery is essential

### Real-World Impact

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

**Operational Burden**: Platform teams must build custom controllers or fix it manually to handle stuck updates.

### Why Existing Solutions Are Insufficient

**1. maxUnavailable Doesn't Address the Core Issue**
The `maxUnavailable` option in `RollingUpdate` strategy allows multiple pods to be updated simultaneously, but it does **not** solve the stuck pod problem:

```yaml
spec:
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 2  # Can update 2 pods at once
```

However, even with `maxUnavailable: 2`, if any pod fails to reach Ready state, the rolling update process still halts completely. The controller waits indefinitely for stuck pods to become Ready, even after applying a fix.

**Example Scenario**:

- StatefulSet with 5 replicas, `maxUnavailable: 2`
- Update pods `app-4` and `app-3` simultaneously
- `app-4` gets stuck in `ImagePullBackOff`
- Even after fixing the image name, `app-4` remains stuck
- Update process cannot proceed to `app-2`, `app-1`, or `app-0`
- Manual intervention still required: `kubectl delete pod app-4`

**2. Parallel Policy Workaround**
Setting `podManagementPolicy: Parallel` with `maxUnavailable` doesn't solve the core issue:

- Pods still must be Running and Ready before rolling update proceeds to the next batch
- A single stuck pod still halts the entire update process  
- Sequential ordering guarantees are lost (undesirable for many use cases)
- Parallel policy affects scaling behavior, not just updates

**2. Custom Controllers**
Some teams have built custom controllers to delete stuck pods, but this:

- Duplicates StatefulSet controller logic
- Creates maintenance burden
- May conflict with StatefulSet controller behavior
- Lacks integration with StatefulSet status and events

### Proposed Solution Benefits

`EnforcedRollingUpdate` addresses these issues by:

1. **Automated Recovery**: Stuck pods are automatically replaced without manual intervention
2. **Safety Preservation**: Sequential ordering is maintained; only Ready pods allow progression
3. **Opt-in Design**: Existing workloads are unaffected; teams choose the appropriate strategy

### Goals

Provide a new rolling-update strategy `enforcedRollingUpdate` to statefulSet.

### Non-Goals

Change to current implementation to forced updating.

## Proposal

### User Stories

#### Story 1: CI/CD Platform Team

**Context**: A platform team manages hundreds of StatefulSet deployments across development and staging environments. Their CI/CD system requires end-to-end automation, but StatefulSet rolling updates break automation when pods get stuck. The team either has to implement custom "garbage collection" logic or accept that automated deployments will fail and require manual intervention.

**Solution**: With `EnforcedRollingUpdate`, the StatefulSet controller would automatically replace the stuck pod when the fixed configuration is applied, allowing the CI/CD pipeline to complete without manual intervention.

#### Story 2: Stateless Web Application

**Context**: A web application uses StatefulSet for predictable pod naming and ordered startup, but doesn't store critical data locally. When resource limit typos cause pods to get stuck in Pending state, the entire update halts even though pod replacement is safe.

**Solution**: With `EnforcedRollingUpdate`, the controller would automatically replace stuck pods after configuration fixes are applied, eliminating the need for manual pod deletion in environments where data loss is not a concern.

#### Story 3: Development/Experiment Environment

**Context**: Developers using StatefulSet for experiments face constant frustration - every time a rolling update breaks due to configuration errors, they must manually delete stuck pods after applying fixes. This manual intervention disrupts the development workflow.

**Solution**: With `EnforcedRollingUpdate`, developers can apply configuration fixes and the StatefulSet controller will automatically replace broken pods, enabling a smoother development experience without requiring cluster operator intervention.

#### Story 4: External Data Storage

**Context**: A database application stores all persistent data on network-attached storage (not local pod storage). Pod replacement is completely safe since no local data would be lost, but the StatefulSet controller treats it as a traditional stateful workload and requires manual intervention.

**Solution**: With `EnforcedRollingUpdate`, the controller would automatically replace pods during updates, which is safe for this architecture since all data persists externally.

#### Story 5: LeaderWorkerSet (LWS) Use Case

**Context**: Developers use StatefulSet as the high-level controller workload for [LWS](https://github.com/kubernetes-sigs/lws). However, it behaves more like a Deployment - there's no ordering dependency between different replicas. They only need the ordinal index for pod identification. When a replica fails during updates, the entire StatefulSet update gets stuck, even though there's no actual ordering requirement between replicas.

**Solution**: With `EnforcedRollingUpdate`, the LWS can continue updating other replicas even when individual replicas fail, since replica ordering is not critical for this use case. This enables automated recovery for deployment-like workloads that use StatefulSet for pod identity rather than traditional stateful ordering.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

#### Risk: Unintended Data Loss

**Risk Description**: If `EnforcedRollingUpdate` is misconfigured on StatefulSets with local persistent data, automatic pod replacement could cause data loss. We should document this.

**Mitigation Strategies**:

1. **Documentation**: Clear guidance on when to use each strategy
2. **Naming Convention**: The "Enforced" prefix clearly indicates forcing behavior
3. **Opt-in Design**: Existing workloads continue using safe `RollingUpdate` by default
4. **Events**: Clear events when pods are forcibly replaced

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

#### Proposed EnforcedRollingUpdate Algorithm

```
FOR i = replicas-1 To i >= 0 DO i-- 
    If pod[i] needs update THEN
        wait_for_predecessors_ready(i+1 to replicas-1)
        
        // Key difference: proceed with update regardless of current pod state
        If pod[i] exists THEN
            delete_pod(i)  // Force replacement of stuck pods
            emit_event("ForcedReplacement", pod[i])
        ENDIF
        create_pod(i)
        wait_until_ready(pod[i])  // Still wait for new pod to be ready
        
        // Safety check: if highest ordinal pod fails, halt progression
        if i == replicas-1 AND !pod[i].Ready THEN
            return // Don't proceed to lower ordinals if latest fails
        ENDIF
    ENDIF
ENDFOR
```

### API Changes

```go
type StatefulSetUpdateStrategyType string

const (
    RollingUpdateStatefulSetStrategyType StatefulSetUpdateStrategyType = "RollingUpdate"
    OnDeleteStatefulSetStrategyType StatefulSetUpdateStrategyType = "OnDelete"
    
    // EnforcedRollingUpdateStatefulSetStrategyType indicates that update will be
    // applied to all Pods in the StatefulSet with respect to the StatefulSet
    // ordering constraints. When a scale operation is performed with this
    // strategy, new Pods will be created from the specification version indicated
    // by the StatefulSet's updateRevision. And whatever the pod status is healthy
    // or broken, the rolling update process will not stuck.
    EnforcedRollingUpdateStatefulSetStrategyType StatefulSetUpdateStrategyType = "EnforcedRollingUpdate"
)

// StatefulSetUpdateStrategy indicates the strategy that the StatefulSet
// controller will use to perform updates.
type StatefulSetUpdateStrategy struct {
    // Type indicates the type of the StatefulSetUpdateStrategy.
    // Default is RollingUpdate.
    // +optional
    Type StatefulSetUpdateStrategyType `json:"type,omitempty"`
    // RollingUpdate is used to communicate parameters when Type is RollingUpdateStatefulSetStrategyType.
    // +optional
    RollingUpdate *RollingUpdateStatefulSetStrategy `json:"rollingUpdate,omitempty"`
    // EnforcedRollingUpdate is used to communicate parameters when Type is EnforcedRollingUpdateStatefulSetStrategyType.
    // +optional
    EnforcedRollingUpdate *RollingUpdateStatefulSetStrategy `json:"enforcedRollingUpdate,omitempty"`
}
```

### Implementation Changes

1. Before, when updating statefulSet, we should make sure that current replicas should be Running, Ready and Available.
Now when StatefulSetUpdateStrategyType is EnforcedRollingUpdate, we'll continue the process.

2. Before, when deleting so called condemned pods who are pods beyond the updated statefulSet replicas, we'll wait for all predecessors to be Running and Ready prior to attempting a deletion.
Now when StatefulSetUpdateStrategyType is EnforcedRollingUpdate, we'll continue the process.

When updating pods that doesn't match the update revision, we'll keep the logic and if the largest ordinal pod failed to turn to health, we'll not update the smaller ordinal pods.

### Comparison with Existing Solutions

| Solution                           | Sequential Ordering | Automatic Recovery | Behavior When Pod Stuck           | Use Case                               |
| ---------------------------------- | ------------------- | ------------------ | --------------------------------- | -------------------------------------- |
| `RollingUpdate`                    | Yes                 | No                 | Halts completely, waits forever   | Traditional stateful apps              |
| `RollingUpdate` + `maxUnavailable` | Yes                 | No                 | **Still halts completely**        | Faster updates, but same stuck problem |
| `OnDelete`                         | Yes                 | No                 | Fully manual control              | Maximum safety/control                 |
| `Parallel` + `maxUnavailable`      | No                  | No                 | Halts, loses ordering             | Not suitable for ordered apps          |
| `EnforcedRollingUpdate`            | Yes                 | Yes                | Automatically replaces stuck pods | Automated recovery needed              |

`maxUnavailable` allows updating multiple pods simultaneously but does not change the fundamental behavior if any pod gets stuck, the entire update halts. The controller still waits for all pods to become Ready before proceeding.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `pkg/apis/apps/validation`: `2022-09-28` - `90.8%`
- `pkg/controller/statefulset`: `2022-09-28` - `85.5%`
- `pkg/registry/apps/statefulset`: `2022-09-28` - `66.7%`

##### Integration tests

We should cover below scenarios:

- With update strategy set to `rollingUpdate`, broken statefulSet will not recover even applying with a fixed configuration.
- With update strategy set to `enforcedRollingUpdate`, broken statefulSet will recover after applying with a fixed configuration.
- With update strategy set to `enforcedRollingUpdate`, broken statefulSet will not rolling update after applying with another unfixed configuration.
- With update strategy set to `rollingUpdate` or `enforcedRollingUpdate`, and `podManagementPolicy` set to `parallel`,
broken statefulSet will recover after applying with a fixed configuration.

##### e2e tests

This feature only impacts the statefulset Controller, so integration tests should be enough.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag.
- Unit and integration tests passed as designed in [TestPlan](#test-plan).

#### Beta

- Feature is enabled by default.
- Address reviews and bug reports from Alpha users.

#### GA

- No negative feedback from developers.

### Upgrade / Downgrade Strategy

#### Upgrade

This feature is protected by the feature-gate `EnforcedRollingUpdateInStatefulSet`, and it's an opt-in strategy
for end-users to choose, it won't change previous behaviors if you don't configure anything.

If you want to use the feature, you should enable this feature gate.

#### Downgrade

If you configured the rolling update strategy to `enforcedRollingUpdate`, when downgrading, you should reconfigure
this strategy to what we supported.

### Version Skew Strategy

No. It general remains the same version with api-server.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: EnforcedRollingUpdateInStatefulSet
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-controller-manager

###### Does enabling the feature change any default behavior?

No, it's backwards compatible.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, we can disable the feature gate.

###### What happens if we reenable the feature if it was previously rolled back?

This feature works again.

###### Are there any tests for feature enablement/disablement?

Yes, we'll cover this in unit tests.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

###### What specific metrics should inform a rollback?

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event Reason: `ForcedReplacement` - emitted when a pod is forcibly replaced
  - Event Reason: `EnforcedRollingUpdateStarted` - emitted when enforced rolling update begins
- [x] API .status
  - Condition name: `StatefulSetUpdateStrategy` - indicates active strategy
  - Other field: `.status.updateRevision` - shows progression through update
- [x] Metrics
  - `statefulset_forced_pod_replacements_total` - tracks forced replacements per StatefulSet

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

**Update Progression SLO**: 99% of StatefulSet updates using EnforcedRollingUpdate should complete without permanent stuck states (measured over 24h periods)

**Forced Replacement Latency**: 95% of forced pod replacements should complete within 5 minutes of configuration fix being applied

**Safety SLO**: 0% of forced replacements should occur on StatefulSets where the latest ordinal pod fails to reach Ready state (safety mechanism functioning)

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `statefulset_enforced_rolling_updates_total`
  - Aggregation method: Rate of successful vs failed enforced rolling updates
  - Components exposing the metric: kube-controller-manager
- [x] Metrics
  - Metric name: `statefulset_forced_pod_replacements_total`
  - Aggregation method: Counter with success/failure labels
  - Components exposing the metric: kube-controller-manager
- [x] Metrics
  - Metric name: `statefulset_update_duration_seconds`
  - Aggregation method: Histogram of update completion times by strategy type
  - Components exposing the metric: kube-controller-manager

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

**Proposed Additional Metrics**:

1. `statefulset_stuck_pod_duration_seconds`: Histogram tracking how long pods remain in non-Ready state before forced replacement
2. `statefulset_strategy_distribution`: Gauge showing distribution of update strategies in use across cluster
3. `statefulset_safety_halt_total`: Counter of times EnforcedRollingUpdate halted due to highest ordinal pod failure (safety mechanism)

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The feature behaves identically to existing StatefulSet update strategies:

- **API Server Unavailable**: StatefulSet controller cannot read/write StatefulSet or Pod objects, so all updates halt regardless of strategy
- **etcd Unavailable**: Similar to API server unavailability - no state changes can be persisted
- **Recovery**: When connectivity is restored, the controller resumes from the last consistent state and continues with the configured strategy

No special handling is required as this feature only changes the update progression logic, not the fundamental dependency on API server/etcd availability.

###### What are other known failure modes?

**1. Highest Ordinal Pod Consistently Failing**

- **Detection**: Metrics show `statefulset_safety_halt_total` increasing; StatefulSet status shows update stalled
- **Mitigations**:
  - Investigate pod logs and events for highest ordinal pod
  - Consider reverting to previous working configuration
  - Switch temporarily to `OnDelete` strategy for manual control
- **Diagnostics**:
  - `kubectl describe statefulset <name>` shows events about safety halt
  - `kubectl describe pod <name>-<highest-ordinal>` shows pod-specific issues
- **Testing**: Integration tests cover this scenario

**2. Resource Quota Exhaustion During Forced Replacement**

- **Detection**: Pods stuck in Pending state with resource quota errors
- **Mitigations**:
  - Increase resource quotas
  - Reduce resource requests in StatefulSet spec
  - Temporarily scale down other workloads
- **Diagnostics**: Events on StatefulSet and Pods show quota-related errors
- **Testing**: Unit tests simulate quota exhaustion scenarios

**3. PVC Deletion Race Conditions**

- **Detection**: New pods fail to start due to PVC conflicts
- **Mitigations**:
  - StatefulSet controller waits for PVC cleanup before creating new pods
  - Implement proper PVC lifecycle management
- **Diagnostics**: Pod events show PVC mounting errors
- **Testing**: Integration tests cover PVC lifecycle scenarios

###### What steps should be taken if SLOs are not being met to determine the problem?

**If Update Progression SLO is failing (updates getting stuck)**:

1. Check `statefulset_safety_halt_total` metric - high values indicate safety mechanism activating
2. Examine highest ordinal pod: `kubectl describe pod <statefulset>-<N-1>`
3. Review StatefulSet events: `kubectl describe statefulset <name>`
4. Check resource availability and quotas
5. Consider temporary strategy change to `OnDelete` for manual control

**If Forced Replacement Latency SLO is failing**:

1. Check cluster resource availability (CPU, memory, storage)
2. Examine pod scheduling issues: `kubectl get events --field-selector involvedObject.kind=Pod`
3. Review image pull times and registry connectivity
4. Check for admission controller delays
5. Monitor `statefulset_update_duration_seconds` histogram for patterns

**If Safety SLO is violated (forced replacements when highest pod fails)**:

1. Immediately investigate controller logs for bugs
2. Check for controller version skew or configuration issues
3. Review recent controller updates or configuration changes
4. Consider disabling feature gate until issue is resolved

## Implementation History

- 2022-09-26: Initial KEP Created

## Drawbacks

### Increased Complexity

Adding a third update strategy increases the API surface and requires users to understand the behavioral differences between:

- `OnDelete`: Fully manual
- `RollingUpdate`: Conservative, halt on failures
- `EnforcedRollingUpdate`: Aggressive, continue despite failures

### Potential for Misuse

Users might choose `EnforcedRollingUpdate` without understanding the data safety implications, potentially leading to:

- Unintended data loss in stateful applications
- Loss of debugging opportunities when pods are automatically replaced
- Masking underlying infrastructure or configuration issues

### Maintenance Burden

The StatefulSet controller becomes more complex with additional code paths, testing requirements, and edge cases to handle.

### Breaking Traditional StatefulSet Guarantees

StatefulSets traditionally prioritize safety over availability. This feature introduces an availability-first option that may conflict with user expectations about StatefulSet behavior.

## Alternatives

### Alternative 1: Enhance Parallel Policy

Extend `podManagementPolicy: Parallel` to support automatic pod replacement when pods are stuck.

**Pros**:

- Reuses existing API surface
- Doesn't require new strategy type

**Cons**:

- Loses sequential ordering guarantees that many users need
- Confuses the semantics of `podManagementPolicy` vs `updateStrategy`
- `Parallel` policy has different scaling behavior that users might not want

Reason for not being considered: Sequential ordering is a key requirement for many StatefulSet use cases. Users want automated recovery without losing ordering guarantees.

### Alternative 2: Add Force Flag to RollingUpdate

Add a boolean field like `spec.updateStrategy.rollingUpdate.force: true` to existing RollingUpdate strategy.

**Pros**:

- Few API change
- Clear relationship to existing strategy

**Cons**:

- Less discoverable than a distinct strategy type
- Overloads the meaning of `RollingUpdate`

Reason for not being considered: Strategy types provide clearer semantics and better discoverability.

### Alternative 3: Custom Resource/Operator Approach

Encourage users to build custom controllers or operators to handle stuck pod cleanup.

**Pros**:

- No changes to core Kubernetes
- Maximum flexibility for specific use cases

**Cons**:

- Duplicates StatefulSet controller logic
- Creates operational burden for every team needing this functionality
- Potential conflicts with StatefulSet controller
- No standardization across community

Reason for not being considered: This is a common enough problem that it warrants a standardized solution in the core controller.

## Infrastructure Needed (Optional)
