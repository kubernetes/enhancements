
---
title: Pod PVC QoS limitation proposal
authors:
  - "@pacoxu"

owning-sig: sig-storage

participating-sigs:
  - sig-scheduling

reviewers:
  - "@mattcary"
  - "@"
  - "@"
  - "@"

approvers:
  - "@"
  - "@"
  - "@"

editor: TBD
creation-date: 2020-07-08
last-updated: 2020-07-08
status: intialized
see-also:
  - ""

replaces:

superseded-by:

---

# Volume QoS Limits

## Table of Contents

<!-- toc -->
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [API Change](#api-change)
    - [Implementation](#implementation)
    - [User Stories](#user-stories)
    - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Design Details](#design-details)
    - [Test Plan](#test-plan)
    - [Graduation Criteria](#graduation-criteria)
        - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
        - [Beta -&gt; GA Graduation](#beta---ga-graduation)
    - [Upgrade / Downgrade / Version Skew Strategy](#upgrade--downgrade--version-skew-strategy)
      - [Interaction with old <code>AttachVolumeLimit</code> implementation](#interaction-with-old--implementation)
  - [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

## Summary

Better resource management for iops or bps of block device in pvc.
In some senarios, we need to limit PV's iops limit or Pod iops on a device or volume at runtime.

For container runtime, dockerd provides blkio related params in docker run to limit a device's iops and bps. [docker reference #block-io-bandwidth-blkio-constraint](https://docs.docker.com/engine/reference/run/#block-io-bandwidth-blkio-constraint)


## Motivation
The proposal would benifit in below two senarios.

* Local Storage(local disk devices), better speed:
For instances, I want to use local storage(local device) to gain the storage local speed.
These PVCs in my cluster are blocking devices.

* rootfs (preventing being overused by some pods)
Besides, the rootfs is shared by all pods in your node, the limitation will help you make node more stable. When one pod in your node heavily uses rootfs, it may effect other pods on the same node. The limitation would help as well.


### Goals

- The limit is runtime limitation for block device when it is mounted to the pod. 
- Only blocking device will be limited with specified volume device id. 
- This should be implemented with cgroup, so it will only work beyond cgroup capability.
- An extended feature that can be provided later is that we can limit iops of rootfs for container runtime.  This will reduce the influcences between pods， running on the same host.


### Non-Goals

- The limit is runtime limitation for block device when it is mounted to the pod. The limit is not a volume limitation in IaaS's aspect. If the device is used by multi pods, each pod should limit the iops by itself. 
- For volume capability of iops, it is a physical limitation on device(PV), and this is not the same as the QoS of PVC in this proposal.
- As cgroup implement has its limitations, we will not mention kernel buffered writings issues with cgroup limitations. This should be fixed or optimized in kernel side. Detailes will be mentioned in the `risks and limitations` below.


## Proposal

After open a feature gate "VolumeQos=True" or "DeviceIOPS=true", pod volume qos can be added by annotation like below. The alpha MVP would use annotations.


```
annotations:
    qos.volume.storage.daocloud.io: >-
        {"pvc": "snap-03", "iops": {"read": 2000, "write": 1000}, "bps":
        {"read": 1000000, "write": 1000000}}
```


Then kubelet can get the mount point of the pvc and the device id. Then we use the cgroup to limit iops and bps of the pod.
We can just edit the cgroup limit files under the pod /sys/fs/cgroup/blkio/kubepods/pod/<Container_ID>/...
For instance, to limit read iops of a pod


```
 echo "<block_device_maj:min> <value>" > /sys/fs/cgroup/blkio/kubepods/pod<UID>/blkio.throttle.read_iops_device
```

To manage the QoS of containers, it can be supported in the future if there are more than 1 contaienrs in the pod, and we may add different iops limits for each container.

### Advantage of Proposal:

* The implementation is based on pod dir, so this can be adapted to any Container Runtimes. If the implementation is based on dockerd iops limitation feature, other container runtime may would support this only if they implement similar features. 
* This is a k8s way, not a docker way. From cluster aspect, application owner set the pv for pods and kubelet could detect which is mount point for the pod, but which device at runtime is mounted to the pod is not visable for application owner .

This is the first/alpha MVP. If this is a suggested way to implement QoS of pvc, we can add iops/bps to container limits like cpu\memory.

![image](https://user-images.githubusercontent.com/2010320/88526364-72201680-d02e-11ea-8dde-891c4f96ffaf.png)
The difference is in step 5 that kubelet will read pod annotation and add iops limits by cgroup to pod

### Beta implementation of this proposal:

Add the limitation directly to volumes properties like below:

```
apiVersion: v1
kind: Pod
metadata:
  name: example
  labels:
    dce.daocloud.io/component: example
spec:
  containers:
  - image: daocloud.io/daocloud/dao-2048:latest
    name: local-2048-1
    volumeMounts:
    - mountPath: /data
      name: volume
    - mountPath: /var/run/secrets/kubernetes.io/serviceaccount
      name: default-token-cv7mt
      readOnly: true
  serviceAccount: default
  serviceAccountName: default
  volumes:
  - name: volume
    persistentVolumeClaim:
      claimName: local-storage-pvc-w-1
    iops:
      read: 2000
      write: 1000
    bps:
      read: 1000000
      write: 1000000
  - name: default-token-cv7mt
    secret:
      defaultMode: 420
      secretName: default-token-cv7mt
```

This would be optional properties for volume in Pod Spec. This is the beta design. 


### API Change
None

### Implementation


#### Implementation detail

The in Kubelet creates pods. 
This will between the volume is mounted to the host and pod is started.
If CSI and kubelet can check which device does the pvc use, the iops limit can apply dynamicly according to PVC name.


##### Code Design


### User Stories

1. A device is a shared volume in kubernetes clusters. We want to the write iops limitation of pod in cluster 1 is 1000 and the limitation for pod in cluster 2 to be 2000. 

2. There are 100 pods that are running on the same host, among which there is a pod that uses disk heavily. Other node may be effected and we want to se an write/read iops limitation for the container on the container rootfs.



other stories can be found in 
#70364
#27000
#70573
#70364
#70980
#92068


### Implementation Details/Notes/Constraints


### Risks and Mitigations
I think iops limiting would be a great idea, but I'm not sure if the current cgroups implementation will effectively implement it. For example, with non-direct device access, writes are buffered in the kernel and something like 80% of them will not be accounted to a cgroup (instead they're all aggregated together). I did a little bit of experimentation here: https://gitlab.com/mattcary/blkio_cgroups/-/blob/master/data/blkio_cgroup.md (sorry that the writeup is not very polished).




## Design Details

TBD

### Test Plan

TBD

### Graduation Criteria

##### Alpha -> Beta Graduation


##### Beta -> GA Graduation


### Upgrade / Downgrade / Version Skew Strategy


#### Interaction with old `AttachVolumeLimit` implementation



## Implementation History

# Alternatives


