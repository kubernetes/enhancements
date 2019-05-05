---
title: Pod Level Resource Quota
authors:
  - "@resouer"
  - "@qinguoan"
  - "@zhangxiaoyu-zidif"
owning-sig: sig-node
participating-sigs:
  - sig-node
reviewers:
  - "@DawnChen"
  - "@yujuhong"
approvers:
  - "@DawnChen"
  - "@yujuhong"
editor: TBD
creation-date: 2019-05-05
last-updated: 2019-05-05
status: provisional
---

# Table of Contents

- [Table of Contents](#table-of-contents)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
  - [Implementation Details](#implementation-details)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Implementation History](#implementation-history)
- 

## Summary
The purpose of this enhancement is to redefine QoS class and Pod resource capacity's measurement methods.

In the current QoS system, all containers resources of a Pod will be considered when calculate the Pod QoS class and Pod resource capacity. In the proposal, sidecar containers' reources will be ignored in above calculations.

## Motivation

Now the current QoS class considers all containers' resources, but as for sidecar containers, its measurement is not quite proper. Sidecar container usuallly cost not much resources and meanwhile its resources is hard to allocated. So user do not define its resource when describing a pod commonly which would result in that the Pod is NOT guaranteed and has lower priority. We need to modify QoS class measurement to make a pod still be guaranteed without setting sidecar containers' resources.


### Goals

* Ignore sidecar containers' resources when calculate Pod's QoS class
* Ignore sidecar containers' resources when calculate Pod's resource capacity.

### Non-Goals

* It does NOT add any QoS class.
* It does NOT change Pod or container spec.

## Proposal

### User Stories

#### Story 1

Considered that there is a pod with one main container and one sidecar container and its resource quota is `4cores and 8Gi`. 

As for main container:
```yaml
# main container
resources:
  limits:
    memory: 8Gi
    cpu: 4
  requests:
    memory: 8Gi
    cpu: 4
```

As for sidecar container:
```yaml
# sidecar container
resources:
  limits:
    memory: 1Gi
    cpu: 1
  requests:
    memory: 0
    cpu: 0
```

Pod resources request = `4cores and 8Gi`.
i.e.: Scheduler will only consider the main container's request.

As for the whole Pod cgroup limit:
i.e.: Pod resource limit =`4cores and 8Gi`.

As for the main container cgroup limit:
i.e.: main container resource limit =`4cores and 8Gi`.

As for the sidecat container cgroup limit:
i.e.: sidecat container resource limit =`1cores and 1Gi`.


### Implementation Details
Step1：
Add featuregates：PodLevelResourceQuota
```go
PodLevelResourceQuota utilfeature.Feature = "PodLevelResourceQuota"
```

**NOTICE** kube-apiserver & kubelet both should be set to `true`.

Step2：
Modify QoS measurement:

```go
// qos.go
func GetPodQOS(pod *v1.Pod) v1.PodQOSClass {
	...
	isGuaranteed := true
	for _, container := range pod.Spec.Containers {
		// process requests
		if utilfeature.DefaultFeatureGate.Enabled(features.PodLevelResourceQuota) 		    {
			if ... # request values and limit values of main containers are equal
		    { 
			 	return v1.PodQOSGuaranteed
			}
		}
```

step3：
Modify Pod's limit measurement:

```go
if utilfeature.DefaultFeatureGate.Enabled(features.PodLevelResourceQuota) && isSidecarContainer(container) {
	continue
} else {
	addResourceList(limits, container.Resources.Limits)
}

```

### Risks and Mitigations


## Implementation History
