---
title: Implement maxUnavailable for StatefulSets
authors:
  - "@krmayankk"
owning-sig: sig-apps
participating-sigs:
  - sig-apps
reviewers:
  - "@janetkuo"
approvers:
  - TBD
editor: TBD
creation-date: 2018-12-29
last-updated: 2018-12-29
status: provisional
see-also:
  - n/a
replaces:
  - n/a
superseded-by:
  - n/a
---

# Implement maxUnavailable in StatefulSet

## Table of Contents

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Proposal](#proposal)
    * [User Stories [optional]](#user-stories-optional)
      * [Story 1](#story-1)
    * [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    * [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)
* [Drawbacks [optional]](#drawbacks-optional)
* [Alternatives [optional]](#alternatives-optional)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Summary

The purpose of this enhancement is to implement maxUnavailable for StatefulSet during RollingUpdate. When a StatefulSet’s 
`.spec.updateStrategy.type` is set to `RollingUpdate`, the StatefulSet controller will delete and recreate each Pod 
in the StatefulSet. The updating of each Pod currently happens one at a time when `spec.podManagementPolicy` is `OrderedReady`. 
With support for `maxUnavailable`, the updating will proceed `maxUnavailable` number of pods at a time in `OrderedReady` case
only.


## Motivation

Consider the following scenarios:-

1: My containers publish metrics to a time series system. If I am using a Deployment, each rolling update creates a new pod name and hence the metrics 
published by these new pod starts a new time series which makes tracking metrics for the application difficult. While this could be mitigated, 
it requires some tricks on the time series collection side. It would be so much better, If we could use a StatefulSet object so my object names doesnt 
change and hence all metrics goes to a single time series. This will be easier if StatefulSet is at feature parity with Deployments.
2: My Container does some initial startup tasks like loading up cache or something that takes a lot of time. If we used StatefulSet, we can only go one 
pod at a time which would result in a slow rolling update. If we did maxUnavailable for StatefulSet with a greater than 1 number, it would allow for a 
faster rollout.
3: My Stateful clustered application, has followers and leaders, with followers being many more than 1. My application can tolerate many followers going 
down at the same time. I want to be able to do faster rollouts by bringing down 2 or more followers at the same time. This is only possible if StatefulSet
supports maxUnavailable in Rolling Updates.
4: Sometimes i just want easier tracking of revisions of a rolling update. Deployment does it through ReplicaSets and has its own nuances. Understanding 
that requires diving into the complicacy of hashing and how replicasets are named. Over and above that, there are some issues with hash collisions which 
further complicate the situation(I know they were solved). StatefulSet introduced ControllerRevisions in 1.7 which I believe are easier to think and reason 
about. They are used by DaemonSet and StatefulSet for tracking revisions. It would be so much nicer if all the use cases of Deployments can be met and we 
could track the revisions by ControllerRevisions.

With this feature in place, when using StatefulSet with maxUnavailable >1, the user understands that this would not cause issues with their Stateful 
Applications which have per pod state and identity while still providing all of the above written advantages.

### Goals
StatefulSet RollingUpdate strategy will contain an additional parameter called `maxUnavailable` to control how many Pods will be brought down at a time,
during Rolling Update.

### Non-Goals
maxUnavailable is only implemeted to affect the Rolling Update of StatefulSet. Considering maxUnavailable for Pod Management Policy of Parallel is beyond 
the purview of this KEP.

## Proposal

### User Stories

#### Story 1
As a User of Kubernetes, I should be able to update my StatefulSet, more than one Pod at a time, in a RollingUpdate way, if my Stateful app can tolerate 
more than one pod being down, thus allowing my update to finish much faster. 

### Implementation Details

#### API Changes

Following changes will be made to the Rolling Update Strategy for StatefulSet.

```go
// RollingUpdateStatefulSetStrategy is used to communicate parameter for RollingUpdateStatefulSetStrategyType.
type RollingUpdateStatefulSetStrategy struct {
	// THIS IS AN EXISTING FIELD
        // Partition indicates the ordinal at which the StatefulSet should be
        // partitioned.
        // Default value is 0.
        // +optional
        Partition *int32 `json:"partition,omitempty" protobuf:"varint,1,opt,name=partition"`

	// NOTE THIS IS THE NEW FIELD BEING PROPOSED
	// The maximum number of pods that can be unavailable during the update.
        // Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%).
        // Absolute number is calculated from percentage by rounding down.
        // Defaults to 1.
        // +optional
        MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty" protobuf:"bytes,2,opt,name=maxUnavailable"`
	
	...
}
```

- By Default, if maxUnavailable is not specified, its value will be assumed to be 1 and StatefulSets will follow their old behavior. This
  will also help while upgrading from a release which doesnt support maxUnavailable to a release which supports this field.
- If maxUnavailable is specified, it cannot be greater than total number of replicas.
- If maxUnavailable is specified and partition is also specified, MaxUnavailable cannot be greater than `replicas-partition`
- podManagementPolicy = OrderedReady
  If a partition is specified, maxUnavailable will only apply to all the pods which are staged by the partition. Which means all Pods with 
an ordinal that is greater than or equal to the partition will be updated when the StatefulSet’s .spec.template is updated. Lets say total 
replicas is 5 and partition is set to 2 and maxUnavailable is set to 2. If the image is changed in this scenario, pods with ordinal 4 and 3 will go
down at the same time(because of maxUnavailable), once they are running and ready, pods with ordinal 2 will go down. Pods with ordinal 0
and 1 will remain untouched due the partition.
- podManagementPolicy = [Parallel](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#parallel-pod-management)
  here maxUnavailable does not make sense since in this policy, pods are launched or terminated in parallel and do not wait for pods to become
  running and ready. So we will never know when two terminate the next batch of maxUnavailable pods.


#### Implementation

https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/statefulset/stateful_set_control.go#L504
```go
...
	podsDeleted := 0
	// we terminate the Pod with the largest ordinal that does not match the update revision.
	for target := len(replicas) - 1; target >= updateMin; target-- {

		// delete the Pod if it is not already terminating and does not match the update revision.
		if getPodRevision(replicas[target]) != updateRevision.Name && !isTerminating(replicas[target]) {
			klog.V(2).Infof("StatefulSet %s/%s terminating Pod %s for update",
				set.Namespace,
				set.Name,
				replicas[target].Name)
			err := ssc.podControl.DeleteStatefulPod(set, replicas[target])
			status.CurrentReplicas--

			// NEW CODE HERE
			if podsDeleted < set.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable {
				podsDeleted ++;
				continue;
			}
			return &status, err
		}

		// wait for unhealthy Pods on update
		if !isHealthy(replicas[target]) {
			klog.V(4).Infof(
				"StatefulSet %s/%s is waiting for Pod %s to update",
				set.Namespace,
				set.Name,
				replicas[target].Name)
			return &status, nil
		}

	}
...
```

### Risks and Mitigations
We are proposing a new field called `maxUnavailable` whose default value will be 1. In this mode, StatefulSet will behave exactly like its current behavior.
Its possible we introduce a bug in the implementation. The mitigation currently is that is disabled by default in Alpha phase for people to try out and give
feedback. 
In Beta phase when its enabled by default, people will only see issues or bugs when `maxUnavailable` is set to something greater than 1. Since people have
tried this feature in Alpha, we would have time to fix issues.


### Upgrades/Downgrades

- Upgrades
 When upgrading from a release without this feature, to a release with maxUnavailable, we will set maxUnavailable to 1. This would give users the same default
 behavior they have to come to expect of in previous releases
- Downgrades
 When downgrading from a release with this feature, to a release without maxUnavailable, there are two cases
 -- if maxUnavailable is greater than 1 -- in this case user  can see unexpected behavior(Find out what is the recommendation here(Warning, disable upgrade, drop field, etc? )
 -- if maxUnavailable is less than equal to 1 -- in this case user wont see any difference in behavior

### Tests

- maxUnavailable =1, Same behavior as today in OrderedReady
- maxUnavailable greater than 1 without partition in OrderedReady
- maxUnavailable greater than replicas without partition in OrderedReady
- maxUnavailable greater than 1 with partition and staged pods less then maxUnavailable in OrderedReady
- maxUnavailable greater than 1 with partition and staged pods same as maxUnavailable in OrderedReady
- maxUnavailable greater than 1 with partition and staged pods greater than maxUnavailable in OrderedReady
- maxUnavailable greater than 1 with partition and maxUnavailable greater than replicas in OrderedReady

## Graduation Criteria

- Alpha: Initial support for maxUnavailable in StatefulSets added. Disabled by default. 
- Beta:  Enabled by default with default value of 1. 


## Implementation History

- KEP Started on 1/1/2019
- Implementation PR and UT by 3/15

## Drawbacks [optional]

Why should this KEP _not_ be implemented.

## Alternatives

- Users who need StatefulSets stable identity and are ok with getting a slow rolling update will continue to use StatefulSets. Users who
are not ok with a slow rolling update, will continue to use Deployments with workarounds for the scenarios mentioned in the Motivations
section.
- Another alternative would be to use OnDelete and deploy your own Custom Controller on top of StatefulSet Pods. There you can implement 
your own logic for deleting more than one pods in a specific order. This requires more work on the user but give them ultimate flexibility.

