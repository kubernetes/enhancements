---
kep-number: 33
title: Application Pod Graceful Update
authors:
  - "@zhan849"
owning-sig: sig-apps
participating-sigs:
reviewers:
  - "@kow3ns"
  - "@janetkuo"
approvers:
  - "@kow3ns"
  - "@janetkuo"
editor: TBD
creation-date: 2019-03-16
last-updated: 2019-03-16
see-also:
  - https://github.com/kubernetes/kubernetes/issues/75136
---


# Application Pod Graceful Update

## Table of Contents

- [Table of Contents](#table-of-contents)
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
- [Proposal](#proposal)
    - [User Stories](#user-stories-optional)
    - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints-optional)
    - [Risks and Mitigations](#risks-and-mitigations)
- [Testing Plan](#testing-plan)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)


## Release Signoff Checklist
- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


## Summary
During application update, we always recreate pods, which can sometimes create unnecessary churns and overhead to the cluster, it would be more 
desirable to "patch" the Pod when possible.


## Motivation

`Pod` provides people with good abstraction of deploying modualizrized applications with sidecar containers for supporting functionalities such as security, 
logging, metrics, proxying, etc. People from different teams can develop their functionalities separately and deploy images with resource isolation and
failure isolation. While different teams have different development / release cycles, we currently always recreate `Pod` during update, which introduced the
following issues:

- Extensive API server access
    - Multiple controllers / kubelet reconcile states, event generation, etc) increases etcd pressure and cluster churns with large scale deployment
- Affects availability
    - `Deployment` is less of a concern as we are able to create new replica first before tearing down the old one
    - For `DaemonSet`, and especially `StatefulSet`, we are not able to create new replica first and there will be down time.
    - When cluster scale is large, control critical path (i.e. reconciliation, scheduling, volume movement, etc) latency can. be huge and further affect availability
- Increase unnecessary application overhead
    - Especially for `StatefulSet`, restarting the main container(s) can be costly as their startup routine are likely to include potential
    data movement, data loading to "warm-up" before serving data, and there are upstream / downstream churns as well

Given the above painpoints, it is a good-to-have improvement that we can update `Pod` by patching `Pod` and restart containers locally when possible.


### Goals
- For `StatefulSet` and `DaemonSet`, user should be able to update Pods by restarting containers locally when **only** non-init container (`Spec.Template.Spec.Containers`) contains image change

### NonGoals
- `Deployment` graceful Pod update. I put this as non-goals for the following reasons:
    - Based on current implementation (by using 2 replicaset), it's hard to implement Pod patching cleanly while keeping current behavior without major refactoring / redesign
    - Stateless services has less startup cost, and since we create new replica first, availability is less likely to get affected
- Graceful update Pod in scenarios other than image change, such as, local volume mount change, environment variable change, etc


## Proposal

### User Stories
- As an application developer, I don't want my application to be restarted unnecessarily just because other sidecar teams wants to roll out their changes,
as crash handling routine such as leadership switch might affect availability
- As a sidecar developer, I don't want to wait for application team's next release cycle to roll out my changes - some application teams has really long 
release cycles and that increases the number of versions we need to maintain
- As a engineering manager, I don't want to migrate to k8s as my current VM environment can support separate container deployment but k8s does not, which
is a regression to me
- As a infrastructure developer, I don't want to get paged about cluster churns such as etcd / k8s api latency / failure spikes as currently there are
a lot of api calls taking place during Pod restart

### Implementation Details/Notes/Constraints
Application updates are controlled by update strategy in their specs:
- For `StatefulSet`, it is `StatefulSetUpdateStrategy`
- For `DaemonSet`, it is `DaemonSetUpdateStrategy`
These update strategies can be extended with an additional field describing how their pods should be updated.


#### API Changes
First we define a generic pod update strategy enumeration type, which can be applied to any application spec
```go
// PodUpdateStrategyType is a string enumeration type that enumerates
// all possible ways we can update a Pod when updating application
type PodUpdateStrategyType string

const (
    // InPlaceIfPossiblePodUpdateStrategyType indicates that we try to patch Pod (upgrade
    // Pod in place) instead of recreate Pod when possible. Currently we patch Pod only
    // when any of the non-init containers (those in Spec.Containers) has image changes.
    // Any other change in Spec will result in a fall back in "Recreate"
    InPlaceIfPossiblePodUpdateStrategyType = "InPlaceIfPossible"

    // RecreatePodUpdateStrategyType indicates that we always delete Pod and create new Pod
    // during Pod update, which is the current behavior
    RecreatePodUpdateStrategyType = "Recreate"
)
```

Then we add this strategy to application's rolling update policies, as with "OnDelete", pods will always be recreated.
```go
type RollingUpdateStatefulSetStrategy struct {
    // +optional
    Partition *int32 `json:"partition,omitempty" protobuf:"varint,1,opt,name=partition"`

    // PodUpdatePolicy indicates how pods should be updated
    // Defautl value is "Recreate"
    // +optional
    PodUpdatePolicy PodUpdateStrategyType `json:"podUpdatePolicy,omitempty" protobuf:"bytes,2,opt,name=podUpdatePolicy"`
}

type RollingUpdateDaemonSet struct {
    // +optional
    MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty" protobuf:"bytes,1,opt,name=maxUnavailable"`

    // PodUpdatePolicy indicates how pods should be updated
    // Defautl value is "Recreate"
    // +optional
    PodUpdatePolicy PodUpdateStrategyType `json:"podUpdatePolicy,omitempty" protobuf:"bytes,2,opt,name=podUpdatePolicy"`
}
```


#### Determining If Pod Should Be Updated In-place
We will use the following logic to determine whether we should upgrade Pods in place or not, a.k.a, if there is only `Spec.Template.Spec.Containers` image changes:
```go
func (ssc *defaultStatefulSetControl) eligibleForInPlaceUpgrade(currentSet, updateSet *apps.StatefulSet) bool {
    currentContainers := currentSet.Spec.Template.Spec.Containers
    updateContainers := updateSet.Spec.Template.Spec.Containers
    
    // short path - if there is container count change, pod should be killed and rescheduled
    if len(currentContainers) != len(updateContainers) {
        return false
    }
    
    // safe to update currentSet directly but we need to make sure it is deep copied to avoid updating cache
    for i := range updateContainers {
        currentContainers[i].Image = updateContainers[i].Image
    }
    
    return apiequality.Semantic.DeepEqual(updateSet, currentSet)
}
```
Daemonset will use same logic to determine if there are only container image changes.

#### Revision Handling
For statefulset, current statefulset reconciliation logic in `updateStatefulSet()` function prepares pods to be deleted (condemned), and pods that should
remain. In the collection of pods that should remain, Pods are created with target revision before entering the routine of actually updating the Pod. 
We should honor current revision / history handling as they should not be affected by update strategy

For daemonset, similarly, revision history is handled in the `rollingUpdate()` function, which computes pod to delete and nodes need new pod. We should honor current revision / history handling as they should not be affected by update strategy change. 


#### Changes In Related Components
Regarding the following involved components:
- **Stateful Set Controller:** Currently in the `updateStatefulSet()` code path, Pods get deleted directly, and rely on next reconciliation to to re create Pod. We need to compute the correct action to upgrade pod and branch off to patch pod or delete pod
- **DaemonSet Conotroller:** Currently in `rollingUpdate()` function, it computes a slice of pods to delete and rely on next reconciliation to create pods with new version. If the daemonset is eligible for in place pod update, it should not delete the pod but update pod instead.
- **Kubelet:** No change needed on kubelet, as its Pod reconciliation will tear down containers that don't meet Pod spec and create new containers.


### Risks and Mitigations

1. It is possible that the new image size increases enormously than the current image, which might add to disk pressure. Since kubelet performs container
image GC, this can be mitigated


```go
// TODO: will take more feedback from the community about whether we need a feature gate for this or not
```

## Testing Plan

We will need regular unit tests and integration tests to cover scenarios where statefulset or daemonset has only container image change. We need to verify
that under such scenarios, Pods will not be recreated but only patched, and other upgrade constraints, revisions will still be honored.


## Graduation Criteria
```go
// TODO: if there is no feature gate needed, probably don't need graduation criteria either.
```

## Implementation History
- 2019-03-20: First draft of KEP
- 2019-06-28: Addressed community's comment, added more implementation details















