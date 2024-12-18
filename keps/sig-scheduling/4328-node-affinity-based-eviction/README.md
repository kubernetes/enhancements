# KEP-4328: Affinity Based Eviction

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [manage multi tenant cluster with dedicated nodes](#manage-multi-tenant-cluster-with-dedicated-nodes)
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
- [Alternatives](#alternatives)
  - [taints](#taints)
  - [descheduler](#descheduler)
  - [some existing controller](#some-existing-controller)
  - [Implement it in kubelet](#implement-it-in-kubelet)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

This KEP aims to introduce node-affinity-based eviction.
In this KEP, we will introduce a new nodeAffinity type: `requiredDuringSchedulingRequiredDuringExecution`.
A new controller called `node-affinity-eviction` will be added to the kube-controller-manager.
This controller will monitor changes in node labels, and try to evict pods if their `requiredDuringSchedulingRequiredDuringExecution` node affinity are no longer met.
If eviction fails, the pods that failed to be evicted will be rechecked for violations of the `requiredDuringSchedulingRequiredDuringExecution` affinity during the next node label update. 
If, in the next eviction round, they still violate the `requiredDuringSchedulingRequiredDuringExecution` affinity, they will be attempted for eviction again.
This controller will respect PDB during the eviction.

We'll start using _the Eviction API, respecting PDB_ to evict pods during alpha stage for simplicity, 
and may transfer to the _Evacuation API_ in future stage.

## Motivation

- Provide users with greater control over pod placement.
- Ensure predictable and consistent pod behavior in various scenarios.
- Address use cases with specific placement and execution requirements.
  
### Goals

- Introduce the `RequiredDuringSchedulingRequiredDuringExecution` **nodeAffinity** type.
- Add `node-affinity-eviction` controller to ensure pods being evicted if the selectors are no longer met at runtime, with the respect of PDB the same time.
- DaemonSet controller will take the `RequiredDuringSchedulingRequiredDuringExecution` into consideration, 
when recalculating the eligible nodes where the DaemonSet pods can run.

### Non-Goals

- This KEP does not aim to introduce the `RequiredDuringSchedulingRequiredDuringExecution` **podAffinity** type and **podAntiAffinity** type.

## Proposal
- Implement the `requiredDuringSchedulingRequiredDuringExecution` node affinity type.
- Update the kube-scheduler, kube-controller-manager and kube-apiserver.
- Define the behavior, syntax, and use cases in documentation.
- Facilitate testing, adoption, and maintenance of this feature.

### User Stories 

#### manage multi tenant cluster with dedicated nodes

I have some nodes labeled with "userA=allow" and "userB=allow", which means that these nodes are available to user A and user B.
As the scale grows I want to add new nodes only for user B while leaving the current nodes only for user A.
I want to migrate all pods available to user B to new nodes, and I want to ensure HA of my services during the migration.

Without `node-affinity-eviction`, I have to remove "userB=allow" label of node, and delete the pods manually. Also I can't use taints because they don't respect PDBs.
With `node-affinity-eviction`, I can simply delete the "userB=allow" label from the existing nodes 
to re-schedule all pods of user B to these new nodes.

### Notes/Constraints/Caveats (Optional)

N/A

### Risks and Mitigations

The major concern may be that during the eviction, some pods may fail to be evicted. Users can use `pod_failed_eviction_node_affinity_total` metric to find them.  

Typically, the reason for a failed eviction is due to a violation of the PDB. 
However, it is also possible that there could be issues with Kubernetes components or a misconfiguration. 
If the eviction of the target pod fails, users should first check if the eviction violates the pod's PDB configuration. 
If it does violate the PDB, users should manually adjust the replica setting of the owner of the pod to ensure that the eviction no longer violates the PDB configuration. Alternatively, users can manually terminate the pod.
Another thing users can check is whether there is incorrect configuration, such as multiple PodDisruptionBudgets referencing the same Pod.
kube-apiserver and controller-manager logs can be also checked for failures.

## Design Details

Introducing a new type of NodeAffinity:
```go
type NodeSelector struct {
	NodeSelectorTerms []NodeSelectorTerm
}

type NodeAffinity struct {
  // If the affinity requirements specified by this field are not met at
  // scheduling time, the pod will not be scheduled onto the node.
  // If the affinity requirements specified by this field cease to be met
  // at some point during pod execution (e.g. due to an update), the system
  // will try to eventually evict the pod from its node.
  // For v1alpha1, evictions are performed by calling the Eviction API.
  // This may change in later versions.
  // +optional
  RequiredDuringSchedulingRequiredDuringExecution *NodeSelector
}
```

Add a controller called `node-affinity-eviction` 

The controller do the following things:

- Listening to the changes of node labels
- Iterating over all pods assigned to the node(excluding mirror pods), checks the NodeAffinity field, if `RequiredDuringSchedulingRequiredDuringExecution` exists, checks if `NodeSelector` still match the new node. 
- If `RequiredDuringSchedulingRequiredDuringExecution` is no loger met, trying to evict the pod.

- For alpha stage:
  - With each occurrence of a node label update event, a new eviction round is triggered. In each round of eviction, the controller will iterate through each pod on that node to determine whether the pod should be evicted.
  - It will use Eviction API to evict the pod(The API takes the PDBs into consideration, so we do not have to use the PDBs explicitly here). 
  - If eviction for one pod fails, We will skip the eviction of the target pod in the current round. In the next eviction round, we will recheck the pod and may trigger eviction based on the results of the check.

- After the eviction in the current round is completed, we will print logs to record the pods that failed to be evicted. And increase the metric `pod_failed_eviction_node_affinity_total` labeled with pods' name. So that users can find out this and take care of the pod manually.
- To avoid any disruption to KCM caused by excessive logging, the logs will be kept minimal and will use the debug level, containing only the names of the pods, ensuring simplicity.


After a node's label is updated, the DaemonSet controller will recalculate the eligible nodes where the DaemonSet pods can run. During this process, the DaemonSet controller will take the `RequiredDuringSchedulingRequiredDuringExecution` into consideration, similar to `RequiredDuringSchedulingIgnoredDuringExecution`. 

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

Currently coverages:
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/nodeaffinity`: `2023-12-13` - `82.5`
- `k8s.io/kubernetes/pkg/apis/core/validation`:`2023-12-13` - `83.9`
- `k8s.io/kubernetes/pkg/controller/nodeaffinityeviction`:`2023-12-13` - `0` 
- `k8s.io/kubernetes/pkg/controller/daemon`:`2024-06011` - `68.9`

These tests will be added:
- New tests will be added to `/pkg/scheduler/framework/plugins/nodeaffinity` to ensure scheduler handles `RequiredDuringSchedulingRequiredDuringExecution` correctly when the feature is enabled/disabled.
- New tests will be added to `/pkg/apis/core/validation` to ensure the validation logic for `RequiredDuringSchedulingRequiredDuringExecution` is corrct when the feature is enabled/disabled.
- A new directory `/pkg/controller/nodeaffinityeviction` will be added
  - New tests will be added to ensure that evict is correctly triggered when the feature is enabled.
  - New tests will be added to ensure that evict is **not** triggered when the feature is disabled. 
  
##### Integration tests

These tests will be added:
- Test that pods with `RequiredDuringSchedulingRequiredDuringExecution` node affinity are correctly scheduled when the feature is enabled.
- Test that pods with `RequiredDuringSchedulingRequiredDuringExecution` node affinity are evicted if the corresponding node's label changed and the selectors are no longer met and the eviction doesn't violate the pod's PDB configuration. .
- Test that pods with `RequiredDuringSchedulingRequiredDuringExecution` node affinity are **not** evicted if the corresponding node's label changed and the selectors are no longer met when the feature is disabled.
- Test that `RequiredDuringSchedulingRequiredDuringExecution` node affinity is ignored during scheduling when the feature is disabled.
- Test that DaemonSet with `RequiredDuringSchedulingRequiredDuringExecution` does not enter a scheduling and descheduling hot loop after being evicted. 
- Test that pods with `RequiredDuringSchedulingRequiredDuringExecution` node affinity are  not evicted if the eviction violates the pod's PDB configuration. 

##### e2e tests

These tests will be added:
- `RequiredDuringSchedulingRequiredDuringExecution` will be tested during both scheduling and execution.
  1. create a node that has a label
  2. create a pod with `RequiredDuringSchedulingRequiredDuringExecution` node affinity matches the node.
  3. verify that the pod is correctly scheduled to the node
  4. remove the node label
  5. verify that the pod is evicted
  6. create a new pod with same template
  7. verify that the pod failed scheduling
  8. create a new node which label matches the pod's node-affinity
  9. verify that the pod created in 6 is scheduled to the new node
   
### Graduation Criteria

#### Alpha

- Feature implemented behind a feature gate
- Initial unit/e2e tests completed and enabled
- Documentation is added to demonstrate why this is useful how exactly affinity needs to be configured
- Documentation is added to demonstrate potential pitfalls of the eviction and handling of these.
- Initial metrics are added

#### Beta

- Gather feedback from developers and surveys
- Additional e2e tests are completed(if needed)
- Additional metrics are added(if needed) 
- See if this kep can align with `Evacuation API`

#### GA

- No negative feedback
- No bug issues reported

### Upgrade / Downgrade Strategy

#### Upgrade
- Enable the feature gate in kube-apiserver, kube-controller-manager and kube-scheduler, and set `.spec.affinity.nodeAffinity.requiredDuringSchedulingRequiredDuringExecution` in pod.

#### Downgrade

- Disable the feature gate in kube-apiserver, kube-controller-manager and kube-scheduler, so that previously configured
  `.spec.affinity.nodeAffinity.requiredDuringSchedulingRequiredDuringExecution` value will be ignored.
- However, the `.spec.affinity.nodeAffinity.requiredDuringSchedulingRequiredDuringExecution` value of a Pod is preserved if it's previously configured; otherwise get silently dropped.

### Version Skew Strategy

kube-controller-manager, kube-scheduler and kube-apiserver will need to enable the feature gate for the full featureset
to be present. 

If kube-apiserver doesn't enable the feature gate, this new affinity will be silently dropped during pod creating, and will be silently dropped during pod updating if it's not been used.

If kube-apiserver enable the feature gate, and:

if only the kube-controller-manager enables the feature gate, the node affinity will be ignored during the scheduling phase. However, DaemonSet pods will still adhere to the `requiredDuringSchedulingRequiredDuringExecution` because the nodes running DaemonSets are determined within the DaemonSet controller.

if only the kube-scheduler enables the feature gate, the node affinity will be ignored during the execution phase. However, DaemonSet pods will not adhere to the `requiredDuringSchedulingRequiredDuringExecution` because the nodes running DaemonSets are determined within the DaemonSet controller.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: NodeAffinityEviction
  - Components depending on the feature gate: kube-controller-manager, kube-scheduler, kube-apiserver

###### Does enabling the feature change any default behavior?

No. It's a new API field, so no default behavior will be impacted.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

The feature can be disabled in Alpha and Beta versions by restarting kube-controller-manager, kube-scheduler and kube-apiserver with the feature-gate off. 
After the feature-gate is disabled and kube-controller-manager is restarted, 
`node-affinity-eviction` controller will also be disabled.
In terms of Stable versions, users can choose to opt-out by not setting the corresponding field.

###### What happens if we reenable the feature if it was previously rolled back?

The `RequiredDuringSchedulingRequiredDuringExecution` node affinity will take effect as expected.

###### Are there any tests for feature enablement/disablement?

Yes. 
Unit tests for this will be added to `pkg/registry/core/pod/strategy_test.go`.
Unit/Integration tests for the new controller will be added to makie sure that pods are not evicted when the feature is disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

It shouldn't impact already running workloads. It's an opt-in feature, and users need to set
`.spec.affinity.nodeAffinity.requiredDuringSchedulingRequiredDuringExecution` field to use this feature.

When this feature is disabled by the feature flag, the already created Pod's `.spec.affinity.nodeAffinity.requiredDuringSchedulingRequiredDuringExecution`
field is preserved, however, the newly created Pod's `.spec.affinity.nodeAffinity.requiredDuringSchedulingRequiredDuringExecution` field is silently dropped.

###### What specific metrics should inform a rollback?

We will introduce two new metric for this feature. Please refer to the "SLIs" section below for its specific meaning.
If the value of metric `pod_failed_eviction_node_affinity_total` or `evictions_node_affinity_total` is larger than expected, users should check their system and may need to rollback.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be tested manually.

Test1:
- Create a node that has a label
- Start a local Kubernetes cluster (feature gate defaulted disabled)
- Create a Pod `test-pod` with `RequiredDuringSchedulingRequiredDuringExecution` node affinity matches the node
- Check Pod's `requiredDuringSchedulingRequiredDuringExecution` node affinty gets dropped due to disabled feature gate
- Delete the Pod

- Re-start kube-controller-manager, kube-scheduler and kube-apiserver with feature gate enabled
- Create `test-pod` again.
- Verify that the pod is correctly scheduled to the node
- Remove the node label
- Verify that the pod is evicted
- Add the node's label back
- Create `test-pod` again
  
- Re-start kube-controller-manager, kube-scheduler and kube-apiserver with feature gate disabled
- Verify the old `.spec.affinity.nodeAffinity.requiredDuringSchedulingRequiredDuringExecution` reserved
- Remove the node label 
- Verify that the pod is not evicted
- Add the node's label back 

- Re-start kube-controller-manager, kube-scheduler and kube-apiserver with feature gate enabled
- Remove the node label
- Verify that the pod is evicted 

Test2:
- Start kube-controller-manager, kube-scheduler and kube-apiserver with feature gate enabled.
- Create a daemonSet with requiredDuringSchedulingRequiredDuringExecution. 
- Verify that the pod runs only on nodes that meet the nodeAffinity.
- Remove the node label.
- Verify that the daemonSet pod on that node is evicted and not recreated.
- Add the node's label back.
- Verify that the daemonSet pod runs on the same node again.
  
- Re-start kube-controller-manager, kube-scheduler and kube-apiserver with feature gate disabled
- Verify the old `.spec.affinity.nodeAffinity.requiredDuringSchedulingRequiredDuringExecution` reserved
- Remove the node label 
- Verify that the pod is not evicted
- Add the node's label back 

- Re-start kube-controller-manager, kube-scheduler and kube-apiserver with feature gate enabled
- Remove the node label
- Verify that the daemonSet pod on that node is evicted and not recreated.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- Check the corresponding metrics(details in SLIs part below).
- Inspect the `.spec.affinity.nodeAffinity.requiredDuringSchedulingRequiredDuringExecution` configuration.

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event Reason: NodeAffinityEviction
- [x] API .status
  - Condition reason: RequiredNodeAffinityViolation
- [x] Other (treat as last resort)
  - Details: Pod with `RequiredDuringSchedulingRequiredDuringExecution` must meet affinity rule during both the scheduling phase and the execution phase.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name:
    - evictions_node_affinity_total(counts of the eviction/evacuation subresources created by the controller)
    - pod_failed_eviction_node_affinity_total(counts of the pods that failed to be evicted, labeled by pod name)
###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Two new metric `evictions_node_affinity_total` and `pod_failed_eviction_node_affinity_total` will be added.

### Dependencies

None identified.

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

- There will be calls to evict pods with eviction subresource. 

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- No to existing API objects that doesn't use this feature.
- For API objects that use this feature:
  - API type: Pod
  - Estimated increase in size: 
    - New field `.spec.affinity.nodeAffinity.requiredDuringSchedulingRequiredDuringExecution`. Which is a `NodeSelector` struct.
    - NodeSelector:
      - NodeSelectorTerms (slice header): 24 bytes
      - Total: 24 bytes
    - NodeSelectorTerm:
      - MatchExpressions (slice header): 24 bytes
      - MatchFields (slice header): 24 bytes
      - Total: 48 bytes
    - NodeSelectorRequirement:
      - Key (string): 16 bytes (assuming an average size)
      - Operator (string): 16 bytes (assuming an average size)
      - Values (slice header): 24 bytes (assuming one element)
      - Values[0] (string): 16 bytes (assuming an average size)
      - Total: 88 bytes
    - The estimated size of the `requiredDuringSchedulingRequiredDuringExecution` would be approximately 176 bytes(assuming we only defined one rule in it).
    
###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

If `.spec.affinity.nodeAffinity.requiredDuringSchedulingRequiredDuringExecution` field is defined, this feature will slightly increase the pod scheduling time and increase the possibility of pods being evicted.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

The scheduler have to process `.spec.affinity.nodeAffinity.requiredDuringSchedulingRequiredDuringExecution` parameter which may result in some small increase in CPU usage.
A new controller will be added to kube-controller-manager and will also result in some small increase in CPU usage.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

During the downtime of API server and/or etcd:

- Running workloads that don't need to be evicted function well.
- Running workloads that need to be evicted by unmatched `.spec.affinity.nodeAffinity.requiredDuringSchedulingRequiredDuringExecution` will stay in current state
as API requests will be rejected.

###### What are other known failure modes?

None identified.

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 2023-12-13: Initial draft KEP

## Drawbacks

None identified

## Alternatives

### taints
  
  Taint can achieve this feature in some scenarios. But nodeaffinity has more operators for users to choose from to cover more scenarios.
  Also taints are meant to protect nodes, so they don't respect PDB.

### descheduler

  Bringing a new component to the system introduces both a lot of uncertainty and additional learning costs. And as a basic feature, this should be part of kubernetes.
  If users only wants to remove pods violating `requiredNodeAffinity`, they can remove descheduler from their system after this feature is implemented.
  However, descheduler also contains some features that are not implemented in this KEP. For example, reschedule of pods with `preferNodeAffinity`.
 

### some existing controller
  
  Creating a manager under some existing controller can implement this feature too.
  However, using a new controller can not only improves code organization but also makes it easier to improve affinity-related scheduling or build custom implementations of the affinity based eviction.
  For example, if we are going to implement RequiredDuringExecution podAffinity/podAntiAffinity in the furture, we can reuse this new affinity-controller.
  Also, the function of this controller is very independent. Implementing it under other controllers will not reduce a lot of redundant code.

### Implement it in kubelet
  
  This feature only cares about node labels and pod's node-affinity, it doesn't care other resources related to pod or node. As kubelet is already a rather complicated component, creating a separate controller can help improve code organization while achieving the goal, makes it easier to improve NodeAffinityEviction feature.

