# Speed up SELinux volume relabeling using mounts

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [SELinux intro](#selinux-intro)
  - [SELinux context assignment](#selinux-context-assignment)
  - [Volume mounting](#volume-mounting)
  - [SELinux support in volumes](#selinux-support-in-volumes)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    - [Behavioral changes](#behavioral-changes)
  - [Examples](#examples)
  - [User Stories [optional]](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [CSI driver considerations](#csi-driver-considerations)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Required kubelet changes](#required-kubelet-changes)
    - [Volume Reconstruction](#volume-reconstruction)
  - [Implementation phases](#implementation-phases)
    - [Phase 1](#phase-1)
    - [Phase 2](#phase-2)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
  - [<code>FSGroupChangePolicy</code> approach](#fsgroupchangepolicy-approach)
  - [Change container runtime](#change-container-runtime)
  - [Move SELinux label management to kubelet](#move-selinux-label-management-to-kubelet)
  - [Merge <code>FSGroupChangePolicy</code> and <code>SELinuxRelabelPolicy</code>](#merge-fsgroupchangepolicy-and-selinuxrelabelpolicy)
  - [Implement kubelet admission](#implement-kubelet-admission)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [X] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [x] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This KEP tries to speed up the way that volumes (incl. persistent volumes) are made available to Pods on systems with SELinux in enforcing mode.
Current way includes recursive relabeling of all files on a volume before a container can be started. This is slow for large volumes.

We propose to use mount option `-o context=XYZ` to set SELinux context of all files on a volume, without recursive walk through the volume.
The enhancement describes situations when such option can/cannot be used, why it's Kubernetes who must care about such a mount option, and possible breaking changes of the new Kubernetes behavior.

This KEP is split into two phases:
1. ReadWriteOncePod volumes are mounted with `-o context` by default.
   All other volumes are recursively relabeled by the container runtime.
   With feature gate `SELinuxMountReadWriteOncePod`, beta + on by default in v1.29.
2. Users can opt-in for all volumes to be mounted with `-o context` by setting `SELinuxChangePolicy: UseMountOption` in PodSpec.  
   With feature gate `SELinuxMount`, alpha since 1.30.

Initially, we thought we could do 2. without opt-in, but we found that it may break valid use cases.

## SELinux intro
SELinux is a complex topic. Here is a brief overview of how it works in the container world.

On Linux machines with SELinux in enforcing mode, SELinux tries to prevent users that escaped from a container to access the host OS and also to access other containers running on the host.
It does so by running each container with unique *SELinux label* (sometimes called *SELinux context*), such as `system_u:system_r:container_t:s0:c309,c383`, and labeling all content on all volumes with the corresponding label (`system_u:object_r:container_file_t:s0:c309,c383`).
Only process with the label `...:container_t:s0:c309,c383` can access files with label `container_file_t:s0:c309,c383`.
Therefore, a rogue user who escaped boundaries of its container, with label say `container_file_t:s0:c68,c222`, cannot access data of other containers, because the label of the attacker's process does not match any other container on the system.
Even processes running as root (UID 0) are denied access to these files, unless they run with the right SELinux label or as privileged containers.

Further in this KEP we assume that the SELinux is enabled on the system. This KEP has absolutely no effect on systems that run without SELinux. Kubelet already knows if SELinux is enabled and does not do anything with it if it's disabled or not available (e.g. on Windows).

See [SELinux documentation](https://selinuxproject.org/page/NB_MLS) for more details.

### SELinux label assignment
In Kubernetes, the SELinux label of a pod is assigned in two ways:
1. Either it is set by user in PodSpec or Container: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/.
1. If not set in Pod/Container, the container runtime will allocate a new unique SELinux label and assign it to a pod (container) by itself.

### Volume mounting
Linux kernel, with SELinux compiled in, allows `mount -o context=system_u:system_r:container_t:s0:c309,c383 <what> <where>` to mount a volume and pretend that all files on the volume have given SELinux label.

Following conditions must be met:
* It must be the first mount of the volume! It does not work for subsequent mounts of the volume, incl. bind-mounts.
    * Second mount of the same volume fails with `mount point not mounted or bad option` when the `-o context=` does not match the first mount.
    * A bind mount inherits `-o context=XYZ` option from the original mount (if it was set there).
    * Shared filesystems may have a special mount option to treat separate mounts of the same volume as "the first" mounts.
      I.e. it's possible to mount the same shared volume with different `-o context` options on the same node, however, these mount options may impact performance.
      * NFS: `mount -o nosharecache`.
      * CIFS: `mount -o nosharesock`.
      * CephFS: no extra mount option is needed.
      * GlusterFS: no extra mount option is needed.
* `/bin/mount` process must detect that the kernel supports SELinux. [It does so](https://github.com/util-linux/util-linux/blob/441f9b9303d015f1777aec7168807d58feacca31/libmount/src/context_mount.c#L291) by checking that both `/etc/selinux/config` file exists and `/sys/fs/selinux` is a mount point with "selinuxfs" type.
    * `/bin/mount` **silently** throws away the SELinux mount options otherwise.

Note that volumes mounted with `-o context` don't have `seclabel` in their mount options.
In addition, calling `chcon` there will fail with `Operation not supported`.

### SELinux support in Kubernetes volumes
Currently, Kubernetes *knows*(*) which volume plugins support SELinux (i.e. supports extended attributes on a filesystem the plugin provides).
If SELinux is supported for a volume, it passes the volume to the container runtime with ":Z" option ([`selinux_relabel`](https://github.com/kubernetes/cri-api/blob/648d24775c39780dfc367536f00197be64534684/pkg/apis/runtime/v1/api.proto#L205) in CRI).
The container runtime then **recursively relabels** all files on the volume to either the label set in PodSpec/Container or the random value allocated by the container runtime itself.

*) These in-tree volume plugins don't support SELinux: HostPath, NFS and Portworx.
All other volume plugins support it.
This knowledge is hardcoded in in-tree volume plugins (e.g. [NFS](https://github.com/kubernetes/kubernetes/blob/0c5c3d8bb97d18a2a25977e92b3f7a49074c2ecb/pkg/volume/nfs/nfs.go#L235)).

For CSI, kubelet currently uses following heuristics:

1. Mount the volume as usual (via `NodeStage` + `NodePublish` CSI calls).
2. Check mount options of the volume mount dir. If and only if it contains `seclabel` mount option, the volume supports SELinux and kubelet will pass ":Z" to the container runtime, which then relabels all files there.

Note that we'll use docker ":Z" option as shortcut for `selinux_relabel` CRI option further in the text.

### Privileged containers

Privileged containers get SELinux label `system_u:system_r:spc_t:s0`, where `spc` means "super privileged container".
Such a container can access any file on the host, regardless of its SELinux label, and any file of any other container.
From SELinux perspective, a rogue process, escaping privileged container boundary, can do anything to any file on the host.
Regular UID / GID checks still apply, so the process either needs to run as root or exploit a CVE to get such permissions.
Privileged Pods also have exception in the container runtime, the runtime does not perform any no recursive relabeling for them.

As consequence of the above, it's possible to run an unprivileged Pod with a SELinux label say `c309,c383` and a privileged Pod with `spc_t` and both can access the same volume in parallel.
When the unprivileged Pod starts, its volumes are relabeled with `c309,c383` and the pod can access its volumes.
When the privileged Pod starts, it does not initiate any relabeling, and at the same time it can access files with any labels, incl. `c309,c383`.

**This was discovered in 1.30 - 1.31, when SELinuxMountReadWriteOncePod was beta + enabled by default.**

## Motivation

* File relabeling in CRI can be slow for big volumes.
* Avoiding out-of-space issues when relabeling almost full volumes. When a volume is almost full, CRI can fail to relabel volumes on it, since SELinux labels may need some little space on the volume.
* Allowing access to read-only volumes. A read-only volume can still be mounted with `-o context=XYZ` and provide files with the right labels to a Pod.
* Mounting volumes with the right SELinux label is a bit safer. Consider [CVE-2021-25741](https://access.redhat.com/security/cve/cve-2021-25741) - here a user can cause Kubernetes to provide host's filesystem to an innocent pod. Without this KEP, CRI will actually relabel the host's files so the Pod can access them. With this KEP, the attacker could still fool Kubernetes to mount host filesystem to the Pod, but the pod would not be able to access it, because (unprivileged) pods are denied accessing files on the host due to SELinux policy.

### Goals

* Mount volumes with the correct SELinux label using `-o context=XYZ` mount option and avoid recursive change of all files on the volume.
  * Do it _by default_ for volumes that can't be shared among Pods, i.e. ReadWriteOncePod PVs.
  * Do it as opt-in for all other volumes, both in-line in Pods or PersistentVolumes.
    As explained above, mixing privileged and unprivileged Pods sharing the same volume is a use case that works today. It can't work with `-o context` mount, see [example below](#conflicts-with-other-pods) with Privileged pods.  
    
### Non-Goals

* Change container runtimes / CRI.

## Proposal

Introduce a new PodSpec field `Spec.SecurityContext.SELinuxChangePolicy` with the following values:
* `UseMountOptionForReadWriteOncePod`: mount RWOP volumes with `-o context` (see other conditions below), all other volumes are relabeled recursively by the container runtime.
  This is the new default. It's safe, because a single RWOP volume can be used only by a single Pod and can't conflict with other Pods running with potentially different SELinux labels.
  * _Naming is hard. Other suggestions were: `MountReadWriteOncePod`, `UseMountOptionForRWOP`, `Conservative`._ 
* `Recursive`: all Pod's volumes are relabeled recursively by the container runtime when the Pod starts.
  * This was the default behavior before this KEP.
* `UseMountOption` tries to mount all Pod volumes with `-o context` mount option (see other conditions below).
  This may not be safe for all Pods, esp. when mixing privileged and unprivileged Pods that use the same volume in parallel. See [below](#conflicts-with-other-pods).
  It is responsibility of the Pod author to set the same SELinux label + to all Pods that use the same volumes in parallel!
  * _Other suggestions were: `Mount`, `MountAll`, `OnMount`._
* `null`: Implies `UseMountOptionForReadWriteOncePod`.
  
Kubelet will mount a Pod's volume with `-o context=XYZ` when *all* these conditions are met:
* Pod has `Pod.Spec.SecurityContext.SELinuxChangePolicy: UseMountOptionForReadWriteOncePod` (when the volume is a RWOP PV) or `UseMountOption` (for all other volumes).
* Pod has SELinux label set, at least in `Spec.SecurityContext.SELinuxOptions.Level` or all containers have `Spec.Containers[*].SecurityContext.SELinuxOptions.Level`.
  * When a `PodSecurityContext` or `SecurityContext` specifies incomplete SELinux label (i.e. omits `SELinuxOptions.User`, `.Role` or `.Type`), kubelet fills the blanks from the system defaults provided by [ContainerLabels() from go-selinux bindings](https://github.com/opencontainers/selinux/blob/621ca21a5218df44259bf5d7a6ee70d720a661d5/go-selinux/selinux_linux.go#L770).
    [See Story 2 below](#story-2).
* The CSI driver responsible for the volume announces support for mounting with `-o context` mount option by setting `CSIDriver.Spec.SELinuxMount: true` in the CSIDriver object.
  For in-tree volume plugins, kubelet has hardcoded knowledge about which volume plugins support `-o context` (iSCSI, FibreChannel) and which don't (all others, esp. NFS and all ephemeral volumes like Secrets and ConfigMap).
  
When any of these conditions are not met, kubelet + the container runtime performs recursive relabeling of the volume as before.

New admission:
* When a Pod has `SELinuxChangePolicy: UseMountOption`, it must have also `Spec.SecurityContext.SELinuxOptions.Level` or all `Spec.Containers[*].SecurityContext.SELinuxOptions.Level` set.
  It's not useful to ask for `-o context` mount and not provide SELinux label to use.
  Since `UseMountOption` is an explicit opt-in, we don't need to worry about old / existing Pods getting invalid during cluster update.
  
It is responsibility of the Pod (Deployment, StatefulSet, DaemonSet) author to set the same SELinux label + the same `SELinuxChangePolicy` to all Pods that use the same volumes in parallel!

### Implementation Details/Notes/Constraints [optional]

#### API changes
`CSIDriver` is extended with a new field `SELinuxMount` to announce if it supports `-o context` mount option:

```go
// In storage.k8s.io/v1:

// CSIDriverSpec is the specification of a CSIDriver.
type CSIDriverSpec struct {
// SELinuxMount specifies if the CSI driver supports "-o context"
// mount option.
//
// When "true", Kubernetes may call NodeStage / NodePublish with "-o context=xyz" mount
// option. The CSI driver must ensure that all volumes can be mounted with different
// `-o context` options. This is typical for storage backends that provide volumes
// as filesystems on block devices or as independent shared volumes.
// It is task of the CSI driver to add any necessary mount options to allow
// mounting with `-o context`, for example `nosharesock` for CIFS or
// `nosharecache` for NFS.
//
// When "false", Kubernetes won't pass any special SELinux mount options to the driver.
// This is typical for volumes that cannot mount a volume with `-o context` mount option.
//
// Default is "false".
SELinuxMount *bool;
...
}

// For context:
type CSIDriver struct {
Spec CSIDriverSpec
}
```

**CSIDriver.SELinuxMount is controlled by feature gate `SELinuxMountReadWriteOncePod` and is beta since Kubernetes 1.30.**

The default value is `false` to ensure backward compatibility.

`PodSecurityContext` is extended with a new field `SELinuxChangePolicy`:

```go
// SecurityContext holds security configuration that will be applied to a container.
// Some fields are present in both SecurityContext and PodSecurityContext.  When both
// are set, the values in SecurityContext take precedence.
type SecurityContext struct {
// ...
	// seLinuxChangePolicy defines how the container's SELinux label is applied to all volumes used by the Pod.
	// It has no effect on nodes that do not support SELinux or when the volume does not support SELinux.
    // Valid values are "UseMountOptionForReadWriteOncePod", "Recursive", and "UseMountOption". If not specified, "UseMountOptionForReadWriteOncePod" is used.
	// It affects only in-tree iSCSI and FibreChannell volumes, and CSI volumes that announce SELinuxMount: true in their CSIDriver instance.
	// It affects only Pods that have SELinux label set, either in PodSecurityContext or in SecurityContext of all containers.
    // Note that this field cannot be set when spec.os.name is windows.
    // +optional
    SELinuxChangePolicy *PodSELinuxChangePolicy
}

// PodSELinuxChangePolicy defines how the container's SELinux label is applied to all volumes used by the Pod.
type PodSELinuxChangePolicy string

const (
	// UseMountOptionForReadWriteOncePod mounts PersistentVolumes with access mode ReadWriteOncePod with `-o context` mount option.
	// All other volumes are relabeled recursively by the container runtime.
	// Mounting with `-o context` is instant and does not require the container runtime to recursively walk through the volume.
	// _Other ideas: MountReadWriteOncePod, UseMountOptionForRWOP, Conservative_
    SELinuxChangePolicyUseMountOptionForReadWriteOncePod PodSELinuxChangePolicy = "UseMountOptionForReadWriteOncePod"
    // Recursive relabeling of all Pod volumes by the container runtime.
    SELinuxChangePolicyRecursive PodSELinuxChangePolicy = "Recursive"
	// UseMountOption mounts all Pod volumes with `-o context` mount option.
	// This requires all Pods that share the same volume to use the same SELinux label. It is not possible to share the same volume among privileged and unprivileged Pods.
	// _Other ideas: Mount, MountAll, OnMount_
    SELinuxChangePolicyUseMountOption PodSELinuxChangePolicy = "UseMountOption"
)
```

**SELinuxChangePolicy is newly proposed API, under `SELinuxMount` feature gate and aiming at alpha in 1.32.**

#### Conflicts with other Pods

It is Pod (Deployment, StatefulSet, DaemonSet) author responsibility to set the `SELinuxChangePolicy` and SELinux labels on Pods correctly. For example, these cases can happen if they are careless. All these cases assume that the volume does support mount with `-o context` and all pods run on the same node.

* Pod A with `SELinuxChangePolicy: Recursive` and label `c1,c2` runs, Pod B with `SELinuxChangePolicy: Recursive` with label `c8,c9` is about to start and both use the same volume.
  * This was possible also before the KEP. When Pod B starts, the container runtime relabels all files on the volume to `c8,c9` and Pod A loses access to data on the volume (will get `EPERM` to all OS calls).
* Pod A with `SELinuxChangePolicy: UseMountOption` and label `c1,c2` runs, Pod B with `SELinuxChangePolicy: UseMountOption` with label `c8,c9` is about to start and both use the same volume.
  * Pod A mounted the volume with `-o context=c1,c2`. Pod B wants the same volume mounted with `-o context=c8,c9`. Pod B will stay in `ContainerCreating` state until Pod A finishes and unmounts the volume, so it can be re-mounted for Pod B.
  
* Pod A with `SELinuxChangePolicy: UseMountOption` runs, Pod B with `SELinuxChangePolicy: Recursive` is about to start and both use the same volume.
  * Pod A mounted the volume with `-o context=<a label>`. Pod B wants the same volume mounted with no `-o context`. Pod B will stay in `ContainerCreating` state until Pod A finishes and unmounts the volume, so it can be re-mounted for Pod B.
* Pod A with `SELinuxChangePolicy: Recursive` runs, Pod B with `SELinuxChangePolicy: UseMountOption` is about to start and both use the same volume.
  * Pod A mounted the volume without `-o context`. Pod B wants the same volume mounted with `-o context=<b label>`. Pod B will stay in `ContainerCreating` state until Pod A finishes and unmounts the volume, so it can be re-mounted for Pod B.
  
* A privileged pod A runs with *any* `SELinuxChangePolicy`, Pod B with `SELinuxChangePolicy: Recursive` with label `c8,c9` is about to start and both use the same volume.
  * Pod A mounted the volume without `-o context` (`spc_t` does not have any `cX,cY` categories) and the volume was not relabeled. When Pod B starts, the container runtime relabels all files on the volume to `c8,c9`. Both pods can access the volume (A is privileged / `spc_t`, B has the right label).
* A privileged pod A runs with *any* `SELinuxChangePolicy`, Pod B with `SELinuxChangePolicy: UseMountOption` with label `c8,c9` is about to start and both use the same volume.
  * Pod A mounted the volume without `-o context` (`spc_t` does not have any `cX,cY` categories) and the volume was not relabeled. Pod B wants the same volume mounted with `-o context=c8,c9`. Pod B will stay in `ContainerCreating` state until Pod A finishes and unmounts the volume, so it can be re-mounted for Pod B. **A and B cannot run in parallel. This is a significant change from the current `Recursive` behavior, so `UseMountOption` cannot be the new default.**

`SELinuxChangePolicy: UseMountOptionForReadWriteOncePod` falls into `UseMountOption` for RWOP volumes and `Recursive` for all other volumes. Since there can be no Pod B for RWOP volumes, it feels more like `Recursive` when conflicts mentioned above happen. All these conflicts were exactly the same before this KEP and thus `UseMountOptionForReadWriteOncePod` can be a safe default.

Those are only examples, more cases can happen. In general, a Pod that wants the volume mounted with a different `-o context` option than it's currently used must wait for all pods that use the volume to get deleted and the volume unmounted.
Kubelet will send an event explaining what the Pod B is waiting for.

### Examples

Following table captures interaction between actual filesystems on a volume and newly introduced behavior with `SELinuxChangePolicy: UseMountOption`. AWS EBS CSI driver and NFS CSI drivers are used as an example of a volume based on a block device and a shared filesystem.

| Volume        | CSIDriver.SELinuxMount | mount opts                     | docker run -v |    |
|---------------|---------------------------------|--------------------------------|---------------|----|
| iSCSI in-tree | N/A                             | `-o context=XYZ`               |               | 1) |
| AWS EBS CSI   | true                            | `-o context=XYZ`               |               | 2) |
| AWS EBS CSI   | unset or false                  |                                | `:Z`          | 3) |
| NFS1 CSI      | true                            | `-o context=XYZ,noshareacache` |               | 4) |
| NFS2 CSI      | unset or false                  |                                |               | 5) |

1) Kubelet knows that the in-tree iSCSI plugin supports mounting with `-o context`. The mount option is then used (if pod context is known) and the container runtime does not relabel the volume.
2) AWS EBS CSI driver ships CSIDriver instance with `SELinuxMount: true`. The behavior is the same as for in-tree volume plugin.
3) Here we show behavior of "old" CSI drivers, that ship their `CSIDriver` with `SELinuxMount` unset (or `false`). Kubelet mounts the volume without any `-o context` option and detects that the volume supports SELinux (by inspecting mount options - it can find `seclabel` there). Therefore, it passes `:Z` to the container runtime to recursively relabel files on the volume.

4) This must be a NFS CSI driver that detects `-o context` mount option and automatically adds `nosharecache` to allow mounting the same volume with different SELinux label on the same node.
   The CSI driver announces this capability by setting `SELinuxMount: true` in its CSIDriver instance.
   Kubelet will mount the volumes with proper label.
5) This is a NFS CSI driver that does not support `-o context` mount option.
   Kubelet then mounts the volume without any extra options.

### User Stories [optional]

#### Story 1

User does not configure anything special in their pods:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: testpod
spec:
  containers:
    - image: nginx
      name: nginx
      volumeMounts:
        - name: vol
          mountPath: /mnt/test
  volumes:
      - name: vol
        persistentVolumeClaim:
          claimName: myclaim
```

No change from current Kubernetes behavior:

1. Kubelet does not see any SELinux label set for the pod thus mounts `myclaim` PVC as usual and if the underlying volume supports SELinux, it passes it to the container runtime with ":Z".
   Kubelet passes also implicit Secret token volume with token with ":Z".
2. Container runtime allocates a new unique SELinux label for the pod and recursively relabels all volumes with ":Z" to this label.


#### Story 2

User (or something else, e.g. an admission webhook) configures SELinux label for a pod using RWOP volume

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: testpod
spec:
  securityContext:
    seLinuxOptions:
      level: s0:c10,c0
  containers:
    - image: nginx
      name: nginx
      volumeMounts:
        - name: vol
          mountPath: /mnt/test
  volumes:
      - name: vol
        persistentVolumeClaim:
          claimName: myclaim
```

1. Since no `SELinuxChangePolicy` is set, kubelet implies `UseMountOptionForReadWriteOncePod`.
1. Kubelet observes that `myclaim` is RWOP PVC.
1. Kubelet sees SELinux context in the pod. It dereferences the `myclaim` PVC and sees that the PV volume supports SELinux. 
1. Kubelet computes rest of the label from system defaults and gets `system_u:object_r:container_file_t:s0:c10,c0`.
1. Kubelet calls MountDevice() / SetUp() calls of the volume plugin with this explicit SELinux context.
1. The volume plugin (or CSI driver underneath), if it supports SELinux, adds `-o context=system_u:object_r:container_file_t:s0:c10,c0` to all its mount calls.
    * Here the CSI volume plugin checks `CSIDriver.SELinuxMount` of the corresponding CSI driver.
1. Kubelet passes no SELinux option to CRI, resulting in no recursive `chcon` in the container runtime.

For example, OpenShift as a Kubernetes distribution, deploys a webhook that can inject SELinux label from namespace annotation into all Pods in the namespace.
Therefore, if configured properly, all Pods in the same namespace run with the same label and they can access data of each other.

Users can use ReadWriteOncePod volumes as a very safe volume that can benefit from fast relabeling using mount options, without thinking too much about it.

#### Story 3

User (or something else, e.g. an admission webhook) configures SELinux label for a pod using RWO volume and `UseMountOption` policy

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: testpod
spec:
  securityContext:
    seLinuxChangePolicy: UseMountOption
    seLinuxOptions:
      level: s0:c10,c0
  containers:
    - image: nginx
      name: nginx
      volumeMounts:
        - name: vol
          mountPath: /mnt/test
  volumes:
      - name: vol
        persistentVolumeClaim:
          claimName: myclaim
```

Kubelet observes `SELinuxChangePolicy` is set and follows a similar code path as the previous story, ending up with the driver mounting the volume with `-o context=system_u:object_r:container_file_t:s0:c10,c0`.

The user is responsible for setting `seLinuxChangePolicy: UseMountOption` and `seLinuxOptions.level: s0:c10,c0` to all Pods that will access the same volume in parallel.
If the volume is used by a Pod with different SELinux label, or a Pod with a different policy, the Pod will stay in `ContainerCreating` state until the volume is unmounted by the other Pod.

### CSI driver considerations

CSI driver vendors need to explicitly opt-in their CSI drivers for this feature.

1. They must ship their CSIDriver instance with `CSIDriver.Spec.SELinuxMount: true`.
2. They must run their CSI driver Pods with `/sys/fs/selinux` and `/etc/selinux/config` shared from the host via HostPath volumes!
   Because `/bin/mount` in the driver container evaluates these files and throws away any SELinux mount options if the files are not present.

We will document this requirement in our documentation that faces CSI driver vendors in gihub.com/kubernetes-csi/docs.

### Risks and Mitigations

This KEP does not change the default behavior of Kubernetes.
When users opt-in for the new behavior using `SELinuxChangePolicy: AllMount`, they must ensure that all Pods that share the same volume have the same SELinux label and the same `SELinuxChangePolicy`.

We provide metrics:
* To detect that a Pod is waiting for another Pod to finish and unmount a volume (already implemented, `volume_manager_selinux_volume_context_mismatch_errors_total` / `volume_manager_selinux_volume_context_mismatch_warnings_total`)

## Design Details

### Required kubelet changes

Apart from the obvious API change and behavior described above, kubelet + volume plugins need not so obvious changes:

* Kubelet's VolumeManager needs to track which SELinux label should get a volume in global mount (to call `MountDevice()` with the right mount options).
  * It must call `UnmountDevice()` even when another pod wants to re-use a mounted volume, but it has a different SELinux label.
  * After kubelet restart, kubelet must reconstruct the original SELinux label it used to SetUp and MountDevice of each volume.
    See Volume Reconstruction below.
  * Reconciler must check also SELinux label used to mount a volume (both mounted devices and volumes) before considering what operation to take on a volume (`MountVolume` or `UnmountVolume`/`UnmountDevice` or nothing).
    It must throw proper error message telling that a Pod can't start because its volume is used by another Pod with a different SELinux label.
    * This is a good point to capture any metrics proposed below.
* Volume plugins will get SELinux label as a new parameter of `MountDevice` and `SetUp`/`SetupAt` calls (resp. as a new field in `DeviceMounterArgs` / `MounterArgs`).
  * Each volume plugin can choose to use the mount option `-o context=` (e.g. when `CSIDriver.SELinuxMount` is `true`) or ignore it (e.g. in-tree volume plugins for shared filesystems or when `CSIDriver.SELinuxMount` is `false` or `nil`).
  * Each volume plugin then returns `SupportsSELinux` from `GetAttributes()` call, depending on if it wants the container runtime to relabel the volume (`true`) or not (`false`; the volume was already mounted with the right label or it does not support SELinux at all).
    It will report error when the label in `/proc/mounts` does not match the expected value.
* When a CSI driver announces `SELinuxMount: true`, kubelet will check that `-o context=X` was correctly applied after `NodePublish()`.
  It is a failure on CSI driver side, that it announces something that it is not able to fulfill.
  All pods that use such a volume will be ContainerCreating until the CSI driver fixes the mount (i.e., probably forever), with a message that it's CSI driver fault.
  This error is already part of generic `storage_operation_duration_seconds` metric (with a label for failures).
  * Note that kubelet can't check mount options after `NodeStage`, because a CSI driver does not need to mount during NodeStage or it may choose to mount to another directory than the staging one.

#### Volume Reconstruction

This work was separated into its own KEP + Feature [#3756](https://github.com/kubernetes/enhancements/issues/3756) and went GA in Kubernetes 1.30.

Here in this KEP we will need only to add SELinux label to reconstructed volumes.

### Implementation phases

Due to change of Kubernetes behavior, we will implement the feature only for cases where it can't break anything first.

#### Phase 1
- Implement the feature only for volumes that are backed by PersistentVolumeClaims with `ReadWriteOncePod` access mode.
  Such volumes can be used only in a single pod and two pods can't ever conflict when using it.
- Collect metrics of how many other pods would fail because they use a RWO/RWX volume that's used by a pod with different SELinux label on the same node.

This phase went Beta (enabled by default) in Kubernetes 1.28, without `SELinuxChangePolicy` field in PodSpec!

#### Phase 2
Based on Phase 1 results:
- Introduce `SELinuxChangePolicy` field in PodSpec
  - We discovered that sharing volumes between privileged and unprivileged containers as described [here](#privileged-containers) is a valid use case.
    We cannot mount *all* volumes with `-o context` and it must be an explicit opt-in using `SELinuxChangePolicy: UseMountOption`.
  - Implement it as an alpha field + graduate it with SELinuxMount feature gate to beta + GA.
  
### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

The main part will be implemented in:

* k8s.io/kubernetes/pkg/kubelet/volumemanager: 2022-06-07 - 53%

##### Integration tests

No existing / new tests for volume mounting there.

##### e2e tests

* Check no recursive `chcon` is done on a volume when not needed.
* Check recursive `chcon` is done on a volume when needed.
* Check that proper metric + alert is emitted when kubelet can't start two pods with different SELinux labels using the same volume on the same node._
  * These tests might use only CSI volumes, GCE PD in-tree volume plugin that we use for e2e tests might be already migrated to CSI by that time.
* Prepare e2e job that runs with SELinux in Enforcing mode.
  * Done:
    * https://testgrid.k8s.io/kops-k8s-ci#kops-aws-selinux: for features enabled by default. 
    * https://testgrid.k8s.io/kops-k8s-ci#kops-aws-selinux-alpha: for alpha features.
    * https://testgrid.k8s.io/presubmits-kubernetes-nonblocking#pull-kubernetes-e2e-gce-storage-selinux: for PRs (needs explicit `/test ` in a PR).
  
### Graduation Criteria

* Alpha of Phase 1:
  * Provided all tests defined above are passing and gated by the feature gate `SELinuxMountReadWriteOncePod` and set to a default of `false`.
  * Documentation exists.
* Beta of Phase 1:
  * The feature gate is `true` by default.
* Evaluation:
  * During the next release after Phase 1 is beta (= the feature is enabled by default), collect reports from users about possible breakage.
  * KEP author has access to usage data from OpenShift, a Kubernetes distro that runs with SELinux in enforcing mode.
* Alpha of Phase 2:.
  * Implement `SELinuxChangePolicy` **with a separate alpha feature gate `SELinuxMount`**.
* GA: all known issues fixed. Otherwise, we will GA Phase 1 only.

### Upgrade / Downgrade Strategy

N/A. This feature affects only mounts. It does not depend on version of Kubernetes on other nodes or in the control plane.
New / old kubelet will still be able to unmount volumes mounted by old / new kubelet as usual.

### Version Skew Strategy

N/A. This feature affects only mounts. It does not depend on version of Kubernetes on other nodes or in the control plane.

## Production Readiness Review Questionnaire

### Feature enablement and rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: `SELinuxMountReadWriteOncePod` (beta in 1.27)
    - Feature gate name: `SELinuxMount` (alpha in 1.30)
      - To enable `SELinuxMount` feature gate, `SELinuxMountReadWriteOncePod` **must** be enabled too.
    - Components depending on the feature gate: apiserver (API validation only), kubelet
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node?

* **Does enabling the feature change any default behavior?**
  Any change of default behavior may be surprising to users or break existing
  automations, so be extremely careful here.

  No.

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  Also set `rollback-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g. if this is runtime
  feature, can it break the existing applications?).

  Yes, it can be disabled / rolled back.

  * When `SELinuxMountReadWriteOncePod` feature gate is disabled, corresponding
    API fields get cleared and Kubernetes uses previous SELinux label handling.
    If the feature gate is enabled/disabled in kubelet without draining the node,
    volumes mounted by the previous kubelet are still mounted with the same
    mount option and thus may / may not have `-o context=` mount option.
    Disabled `SELinuxMountReadWriteOncePod` automatically disables
    `SELinuxMount` feature gate.
  * When `SELinuxMount` feature gate is disabled and
    `SELinuxMountReadWriteOncePod` enabled, kubelet will handle SELinux mounts
    only for RWOP volumes. Similarly to previous case, RWO / RWX volumes mounted by the
    previous kubelet may be still mounted with `-o context` mount option.

  In both cases, the disabled / enabled feature affects only newly started Pods.
  A new kubelet can still umount volumes mounted by the previous kubelet as
  usual.

  To prevent any issues during enabling / disabling any of the feature gates
  or kubelet upgrade, we recommended draining the node before the change.
  
* **What happens if we reenable the feature if it was previously rolled back?**
  
  Nothing special happens, see the previous bullet.

* **Are there any tests for feature enablement/disablement?**
  The e2e framework does not currently support enabling and disabling feature
  gates. However, unit tests in each component dealing with managing data created
  with and without the feature are necessary. At the very least, think about
  conversion tests if API types are being modified.

  We plan unit tests for enabled / disable feature.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g. what if some components will restart
  in the middle of rollout?

  This KEP affects only kubelet behavior and only mounts on the node where kubelet runs.
  Different nodes in a cluster can have the feature enabled/disabled without any issues.
  
* **What specific metrics should inform a rollback?**

  `volume_manager_selinux_volume_context_mismatch_errors_total` show that Pods
  that were potentially running before upgrade can't work now.

  When `SELinuxMount` feature gate goes alpha, the metric will have
  a label with volume access mode, so a cluster admin can tell if
  disabling `SELinuxMount` is enough or if `SELinuxMountReadWriteOncePod`
  must be disabled too.

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and do that now.

  This was tested manually before releasing `SELinuxMountReadWriteOncePod`
  enabled by default.

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

  No.

### Monitoring requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metrics. Operations against Kubernetes API (e.g.
  checking if there are objects with field X set) may be last resort. Avoid
  logs or events for this purpose.

  - Metrics described below show up.
  - (Some) volumes are mounted with `-o context` mount option
    (i.e. ssh to kubelet and read the mount table or read CSI driver logs).

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**

  - [x] Metrics
    - All `errors_total` metrics below cover real errors when a Pod can't start.
      - It applies to `ReadWriteOncePod` volumes when only `SELinuxMountReadWriteOncePod` feature gate is enabled.
      - It applies to all volumes when both `ReadWriteOncePod` and `SELinuxMount` feature gate is enabled.
    - All `warnings_total` metrics below are reported when only `SELinuxMountReadWriteOncePod` feature gate is enabled and shows **future** errors that would appear after both `SELinuxMountReadWriteOncePod` and `SELinuxMount` feature gates are enabled.
      This will be evaluated in Phase 2.
    - 1. `volume_manager_selinux_container_errors_total` + `volume_manager_selinux_container_warnings_total`: Number of errors when kubelet cannot compute SELinux label for a container.
        This indicates an error converting SELinux label into SELinux label by github.com/opencontainers/selinux/go-selinux library.
        Reading its source code, this should never happen, but one never knows.
      1. `volume_manager_selinux_pod_context_mismatch_errors_total` + `volume_manager_selinux_pod_context_mismatch_warnings_total`: Number of errors when a Pod defines different SELinux labels for its containers that use the same volume.
         Before this feature, only one container in such a Pod could access the volume.
         With this feature, the Pod won't even start.
         This metric captures nr. of failed Pod starts, including periodic retries.
      1. `volume_manager_selinux_volume_context_mismatch_errors_total` + `volume_manager_selinux_volume_context_mismatch_warnings_total`: Number of errors when a Pod uses a volume that is already mounted with a different SELinux label than the Pod needs.
         Before this feature, both pods would start, but only one such pod could access the volume.
         With this feature, one of the Pods won't even start.
    - `pod_start_sli_duration_seconds`: Duration in seconds to start a pod, excluding time to pull images and run init containers, measured from pod creation timestamp to when all its containers are reported as started and observed via watch.
      This is already existing metric, it should not be worse than before this KEP, because CRI does not need to relabel (some) volume mounts.
    - Components exposing the metric: kubelet

  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At the high-level this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide a comprehensive guidance, but at the very
  high level (they needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code


- `pod_start_sli_duration_seconds` is the same or better than before this feature, because CRI does not need to relabel (some) volume mounts.

All the other metrics above mostly indicate that *user* has made a mistake and use the same volume in two Pods with the same SELinux label.
This did not work even before this KEP (except for the case where two pods use different subpaths), now it's just more obvious.
IMO we can't base SLO on this.

* **Are there any missing metrics that would be useful to have to improve
  observability if this feature?**
  Describe the metrics themselves and the reason they weren't added (e.g. cost,
  implementation difficulties, etc.).

No.

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**

No deps.

### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirms the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**

  No new API calls are required. Kubelet / CSI volume plugin already has CSIDriver informer.

* **Will enabling / using this feature result in introducing new API types?**

  No new API types.

* **Will enabling / using this feature result in any new calls to cloud
  provider?**

  No new calls to cloud providers.

* **Will enabling / using this feature result in increasing size or count

  CSIDriver gets one new field. We expect only few CSIDriver objects in a cluster.

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.

  `pod_start_sli_duration_seconds` should actually get better, not worse.

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data send and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits][].

  No. Kubelet already has a cache of desired / existing mounts, we need to add
  a string with SELinux label to each one, which should be negligible.

* **Can enabling / using this feature result in resource exhaustion of some node
  resources (PIDs, sockets, inodes, etc.)?**

  Not in Kubernetes.

  A CSI driver may need to use a specific mount option to allow mounting the
  same volume with different SELinux label on the same node, which may have
  negative impact on memory, CPU or sockets. For example, NFS may require
  `nosharecache` mount option that disables sharing of caches between mounts of
  the same volume.

### Troubleshooting

Troubleshooting section serves the `Playbook` role as of now. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now we leave it here though.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

  Kubelet can't start Pods if it can't reach the API server and populate its
  `CSIDriver` informer. This was the case also before this KEP.

* **What are other known failure modes?**
  For each of them fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without loogging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debugging the issue?
      Not required until feature graduated to Beta.
    - Testing: Are there any tests for failure mode? If not describe why.

  - *Kubelet des not start new Pods*
    - Detection: `volume_manager_selinux_container_errors_total`, `volume_manager_selinux_pod_context_mismatch_errors_total` or `volume_manager_selinux_volume_context_mismatch_errors_total` grows.
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
      Workloads that run keep running, only new Pods can't start.
    - Diagnostics: Kubelet emits events to Pod why the metrics are growing. Admin must find the affected Pods and check why they can't run, typically there is another Pod on the node that uses the same volume, but with a different SELinux label.
    - Testing: Yes, see e2e tests above.

* **What steps should be taken if SLOs are not being met to determine the problem?**

  Downgrade and/or disable the feature gate?

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

* 1.25: Partial implementation of alpha.
  * Volume reconstruction after kubelet start does not reconstruct SELinux labels.
* 1.26: Alpha with everything implemented.
* 1.27: Targeting beta.
  * Volume reconstruction separated into its own KEP + Feature [#3756](https://github.com/kubernetes/enhancements/issues/3756).
* 1.30: `SELinuxMountReadWriteOncePod` still beta, SELinuxMount (early) alpha.
  * Implement bare minimum of `SELinuxMount` for experiments, including:
    * Extend SELinux mount to all volume access modes.
    * Add label with volume access mode to `volume_manager_selinux_volume_context_mismatch_errors_total` and similar metrics. 
* 1.32: `SELinuxMountReadWriteOncePod` still beta, SELinuxMount alpha.
  * We discovered that sharing volumes between privileged and unprivileged containers as desceribed [here](#privileged-containers) is a valid use case.
    we cannot mount *all* volumes with `-o context` and it must be an explicit opt-in using `SELinuxChangePolicy: UseMountOption`.
  * Implement `SELinuxChangePolicy` as an alpha field.

## Drawbacks [optional]

* The API is slightly different that `FSGroupChangePolicy`, which may create confusion.

## Alternatives [optional]

### `FSGroupChangePolicy` approach
The same approach & API as in `FSGroupChangePolicy` can be used.
**This is a viable option!**

If kubelet knows SELinux label that should be applied to a volume && hypothetical `SELinuxChangePolicy` is `OnRootMismatch`, it would check label only of the top-level directory of a volume and recursively `chcon` all files only when the top level dir does not match.
This could be done together with recursive change for `fsGroup`.
Kubelet would not use ":Z" when passing the volume to container runtime.

With `SELinuxChangePolicy: Always`, usual ":Z" is passed to container runtime and it relabels all volumes recursively.

Advantages:
* Simplicity, both to users and implementation-wise. Follow `FSGroupChangePolicy` approach and do `chcon` instead of `chown`.

Disadvantages:
* Speed, Kubernetes must recursively `chcon` all files on a volume when the volume is used for the first time.
  With `mount -o context`, no relabeling is needed.

### Change container runtime

We considered implementing something like `SELinuxChangePolicy: OnRootMismatch` in the container runtime.
It would do the same as `PodFSGroupChangePolicy: OnRootMismatch` in [fsGroup KEP], however, in the container runtime.

This approach cannot work because of `SubPath`.
If a Pod uses a volume with SubPath, container runtime gets only a subdirectory of the volume.
It could check the top-level of this subdir only and recursively change SELinux label there, however, this could leave different subdirectories of the volume with different SELinux labels and checking top-level directory only does not work.
With solution implemented in kubelet, we can always check top level directory of the whole volume and change label on the whole volume too.

### Move SELinux label management to kubelet
Right now, it's the container runtime who assigns labels to containers that don't have any specific `SELinuxOptions`.
We could move SELinux label assignment to kubelet.
This change would require significant changes both in kubelet (to manage the labels) and CRI (to list used label after kubelet restart).
As benefit, kubelet would mount volumes for *all* pods quickly, not only those that have explicit `SELinuxOptions`.
We are not sure if it's possible to change the default behavior to `OnVolumeMount`, without any field in `PodSecurityPolicy`.

### Merge `FSGroupChangePolicy` and `SELinuxRelabelPolicy`
With this API, user could ask for any shortcuts that are available regarding SELinux relabeling and ownership change for FSGroup:

```go
const (
  // The heuristic policy acts like setting both the OnVolumeMount policy and the OnRootMismatch policy.
  HeuristicVolumeChangePolicy VolumeChangePolicy = "Heuristic"
  RecursiveVolumeChangePolicy VolumeChangePolicy = "Recursive"
)

type PodSecurityContext struct {
  ...
  VolumeChangePolicy *VolumeChangePolicy
  ...
}
```

In the vast majority of cases it's what users want.

However, this field is not flexible enough to accommodate special cases.
If supported by the storage backend and the volume is consumed as whole, `SELinuxRelabelPolicy: OnMount` always works.
At the same time, `FSGroupChangePolicy: OnRootMismatch` may not be desirable for volumes that are modified outside of Kubernetes,
where various files on the volume may get random owners.

With a single `VolumeChangePolicy`, user has to fall back to `Recursive` policy and SELinux labels would be unnecessarily changed.

### Implement kubelet admission

It's not really an alternative, but it's worth mentioning that we originally proposed to implement kubelet admission for Pods with SELinux labels.

Kubelet _could_ reject a Pod that uses a volume with a different SELinux label than the volume is currently mounted with on the node, right during Pod admission.
Such a Pod would end in `Failed` state with appropriate message, and it would never reach kubelet internal states (e.g. DSW, ASW, MountDevice, SetUp, etc.).

We thought it would send a clear message to the user that such a pod can't start, and why.
During implementation and testing it has shown that it's very noisy. Consider this experiment:

1. User runs a single-replica Deployment with no SELinux label set. The Deployment uses a single volume. The Deployment runs Pod A on Node 1.
1. User edits the Deployment and sets the SELinux label of the Pod, because they want to benefit from `SELinuxMount` feature.
1. Deployment / ReplicaSet controllers start a new Pod B with the new SELinux label.
1. Scheduler (accidentally?) puts Pod B to Node 1, where Pod A still runs.
1. Kubelet rejects the new Pod, because the volume is already mounted without any SELinux label, while the new Pod wants a specific one.
1. Deployment / ReplicaSet controllers start another Pod with the new SELinux label. Scheduler puts it to Node 1 again.
1. Kubelet rejects the Pod again. This loop can continue forever, as longs as the scheduler picks Node 1 and Pod A still runs there.

I got hundreds Pods just created and rejected in a minute or so. While it sends a clear message to the user that something is wrong, it's just too noisy for the API server.

**As result, we decided to remove pod admission in kubelet from this KEP.**

Expected behavior without Pod admission is that kubelet admits the Pod B and eventually its volumes reach DSW.
The Pod B will be `ContainerCreating` until Pod A terminates and the volume can be mounted with the right SELinux label.
The same behavior was already implemented for `ReadWriteOncePod` AccessMode.

### Mount **all** volumes with `-o context`, without any `SELinuxChangePolicy`

Initially, we wanted kubelet to do the right thing automatically, without any user intervention.
Later found out that sharing the same volume among privileged and unprivileged containers is a valid use case.
That use case would not be possible if all mount used `-o context`:

1. An privileged Pod A runs. Its volumes are mounted without any special `-o context`, privileged pods run with `spc_t` label.
2. An unprivileged Pod B with label `c1,c2` is about to start, and it wants to use volume already mounted to Pod A.
   Kubelet can't mount the volume with `-o context=c1,c2`, because the volume is already mounted.
   It needs to wait for Pod A to get deleted + its volume unmounted.

That's not what user wants, they want A and B to run at the same time and access the same data, which is possible with recursive relabeling.
