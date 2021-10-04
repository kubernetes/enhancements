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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This KEP tries to speed up the way that volumes (incl. persistent volumes) are made available to Pods on systems with SELinux in enforcing mode.
Current way includes recursive relabeling of all files on a volume before a container can be started. This is slow for large volumes.

We propose to use mount option `-o context=XYZ` to set SELinux context of all files on a volume, without recursive walk through the volume.
The enhancement describes situations when such option can/cannot be used, why it's Kubernetes who must care about such a mount option, and possible breaking changes of the new Kubernetes behavior.

## SELinux intro
On Linux machines with SELinux in enforcing mode, SELinux tries to prevent users that escaped from a container to access the host OS and also to access other containers running on the host.
It does so by running each container with unique *SELinux context* (such as `system_u:system_r:container_t:s0:c309,c383`) and labeling all content on all volumes with the corresponding label (`system_u:object_r:container_file_t:s0:c309,c383`).
Only process with the context `...:container_t:s0:c309,c383` can access files with label `container_file_t:s0:c309,c383`.
Therefore, a rogue user who escaped boundaries of its container cannot access data of other containers, because volumes of each container have different label.
Even processes running as root (UID 0) are denied access to these files, unless they run with the right SELinux context. 

Further in this KEP we assume that the SELinux is enabled on the system. This KEP has absolutely no effect on systems that run without SELinux. Kubelet already knows if SELinux is enabled and does not do anything with it if it's disabled or not available (e.g. on Windows). 

See [SELinux documentation](https://selinuxproject.org/page/NB_MLS) for more details.


### SELinux context assignment
In Kubernetes, the SELinux context of a pod is assigned in two ways:
1. Either it is set by user in PodSpec or Container: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/.
1. If not set in Pod/Container, the container runtime will allocate a new unique SELinux context and assign it to a pod (container) by itself.

### Volume mounting
Linux kernel, with SELinux compiled in, allows `mount -o context=system_u:system_r:container_t:s0:c309,c383 <what> <where>` to mount a volume and pretend that all files on the volume have given SELinux label.

Following conditions must be met:
* It must be the first mount of the volume! It does not work for subsequent mounts of the volume, incl. bind-mounts.
    * Second mount of the same volume fails with `mount point not mounted or bad option` when the `-o context=` does not match the first mount.
    * A bind mount inherits `-o context=XYZ` option from the original mount (if it was set there). 
* `/bin/mount` process must detect that the kernel supports SELinux. [It does so](https://github.com/util-linux/util-linux/blob/441f9b9303d015f1777aec7168807d58feacca31/libmount/src/context_mount.c#L291) by checking that both `/etc/selinux/config` file exists and `/sys/fs/selinux` is a mount point with "selinuxfs" type.
    * `/bin/mount` **silently** throws away the SELinux mount options otherwise.

Note that volumes mounted with `-o context` don't have `seclabel` in their mount options.
In addition, calling `chcon` there will fail with `Operation not supported`.

### SELinux support in volumes
Currently, Kubernetes *knows*(*) which volume plugins support SELinux (i.e. supports extended attributes on a filesystem the plugin provides).
If SELinux is supported for a volume, it passes the volume to the container runtime with ":Z" option ([`selinux_relabel`](https://github.com/kubernetes/cri-api/blob/648d24775c39780dfc367536f00197be64534684/pkg/apis/runtime/v1/api.proto#L205) in CRI).
The container runtime then **recursively relabels** all files on the volume to either the label set in PodSpec/Container or the random value allocated by the container runtime itself.

*) These in-tree volume plugins don't support SELinux: Azure File, CephFS, GlusterFS, HostPath, NFS and Portworx.
All other volume plugins support it.
This knowledge is hardcoded in in-tree volume plugins (e.g. [NFS](https://github.com/kubernetes/kubernetes/blob/0c5c3d8bb97d18a2a25977e92b3f7a49074c2ecb/pkg/volume/nfs/nfs.go#L235)).

For CSI, kubelet currently uses following heuristics:

1. Mount the volume as usual (via `NodeStage` + `NodePublish` CSI calls).
2. Check mount options of the volume mount dir. If and only if it contains `seclabel` mount option, the volume supports SELinux and kubelet will pass ":Z" to the container runtime, which then relabels all files there.

Note that we'll use docker ":Z" option as shortcut for `selinux_relabel` CRI option further in the text.

## Motivation

* File relabeling in CRI can be slow for big volumes.
* Avoiding out-of-space issues when relabeling almost full volumes. When a volume is almost full, CRI can fail to relabel volumes on it, since SELinux labels may need some little space on the volume. 
* Allowing access to read-only volumes. A read-only volume can still be mounted with `-o context=XYZ` and provide files with the right labels to a Pod.
* Mounting volumes with the right SELinux context is a bit safer. Consider [CVE-2021-25741](https://access.redhat.com/security/cve/cve-2021-25741) - here a user can cause Kubernetes to provide host's filesystem to an innocent pod. Without this KEP, CRI will actually relabel the host's files so the Pod can access them. With this KEP, the attacker could still fool Kubernetes to mount host filesystem to the Pod, but the pod would not be able to access it, because (unprivileged) pods are denied accessing files on the host due to SELinux policy.


### Goals

* If possible, mount volumes with the correct SELinux context using `-o context=XYZ` mount option and avoid recursive change of all files on the volume.

### Non-Goals

* Change container runtimes / CRI.

## Proposal

* When kubelet *knows* SELinux context of a pod (i.e. at least `Pod.Spec.SecurityContext.SELinuxOptions.Level` is set) AND kubelet *knows* that a volume supports mounting with `-o context`:
  * kubelet passes `-o context=XYZ` to `MountDevice()` and `SetUp()` calls of the volume.
    * A volume plugin / CSI driver will get these as regular mount options and use them to mount the volume.
  * kubelet does not pass any special SELinux option to the container runtime (explicitly, no `:Z`).
    Files on the volume already have the right SELinux context, no recursive relabeling happens there.
  * When a `PodSecurityContext` specifies incomplete SELinux label (i.e. omits `SELinuxOptions.User`, `.Role` or `.Type`), kubelet fills the blanks from the system defaults provided by [ContainerLabels() from go-selinux bindings](https://github.com/opencontainers/selinux/blob/621ca21a5218df44259bf5d7a6ee70d720a661d5/go-selinux/selinux_linux.go#L770).
    [See Story 2 below](#story-2).

* When kubelet does not know SELinux context of a pod, it falls back to the current behavior.
  Based on [the heuristics described above](#SELinux-support-in-volumes), it may pass `:Z` option to pod volumes, the container runtime allocates a new SELinux context for the pod and relabel all the volumes recursively.

* When kubelet knows that the volume does not support mounting with `-o context` (or when it is not sure), it falls back to the current behavior.
  Based on [the heuristics described above](#SELinux-support-in-volumes), it may pass `:Z` option to pod volumes, and the container will relabel the volume recursively.

**How kubelet knows *if* a volume supports mounting with SELinux?**

[As described above](#volume-mounting), any volume can be mounted with `-o context=XYZ`, as long as it's the first mount of the volume.

* For mounting of block devices, it's safe to assume that `-o context=XYZ` always works - kubelet / CSI drivers mounts such a volume as whole.
* For shared volumes, such as NFS, GlusterFS or CephFS, it's hard for kubelet to distinguish what is a volume and what is a directory on it.
  For example, when mounting `example.com:/exports/archive/jsafrane/projects/foo`, `example.com:/exports/archive` is name of the volume (from kernel point of view) and `jsafrane/projects/foo` is path inside the volume.
  It can still be mounted with `-o context=XYZ`, however, all subsequent mounts of `example.com:/exports/archive` must use the same `-o context=XYZ` option!
  In Kubernetes, we do not have any way to tell if a NFS, GlusterFS or CephFS PVs are using the same volumes (NFS/Gluster/CephFS shares), and therefore kubelet cannot use `-o context=XYZ` for these volume plugins.

* For CSI drivers, kubelet has no clue if the PVs provided by the driver are independent and each of them can be mounted with a different `-o context=` value, or some (or all) PVs are in fact the same volume from kernel point of view, only a different subdirectory of it.
    Therefore, we need CSI driver vendors to provide such information in CSIDriver object of their driver:

    ```go
    // In storage.k8s.io/v1:
    
    
    // CSIDriverSpec is the specification of a CSIDriver.
    type CSIDriverSpec struct {
        // SELinuxMountSupported specifies if the CSI driver supports "-o context"
        // mount option.
        //
        // When "true", Kubernetes may call NodeStage / NodePublish with "-o context=xyz" mount
        // option. The CSI driver must ensure that all volumes can be mounted with different
        // `-o context` options. This is typical for storage backends that provide volumes
        // as filesystems on block devices or as independent shared volumes.
        //
        // When "false", Kubernetes won't pass any special SELinux mount options to the driver.
        // This is typical for volumes that represent subdirectories of a bigger shared filesystem.
        //
        // Default is "false".
        SELinuxMountSupported *bool;
        ...
    }
    
    // For context:
    type CSIDriver struct {
        Spec CSIDriverSpec
    }
    ```
    The default value is `false` to ensure backward compatibility.

### Implementation Details/Notes/Constraints [optional]

#### Behavioral changes
This KEP changes behavior of Kubernetes when two pods with different SELinux contexts use the same volume.
Let Pod A with SELinux context X runs and Pod B with SELinux context Y is about to start on the same node and both use the same volume.

* *Before this KEP*: Pod A suddenly starts getting "permission denied" errors when accessing files on the volume, because the container runtime re-labeled all files on it with label Y when starting pod B. Pod B will start just fine and can access the volume.
* *As proposed in this KEP*: Pod B won't even start, because the volume is already mounted with `-o context=X`.
  Since kubelet tracks SELinux contexts of all mounts it manages, it will see that a new pod wants to use an already mounted volume with a different context, and it will fail with a message like `volume X is already used by pod Y with another SELinux context`.
  Note that this will not work for mounts of the volume done by something else than kubelet.
  In that case, kubelet will pass `-o context=X` to the CSI driver, the driver will pass it to kernel and kernel will fail with a generic `mount: wrong fs type, bad option, bad superblock on /dev/sdb, missing codepage or helper program, or other error`.
  `/bind/mount` / kernel is not able to tell which mount option is wrong.

A special case of the previous example is when two pods with different SELinux contexts use the same volume, but different subpaths of it.
The container runtime then re-labels only these subpaths and as long as the subpaths are different, both pods can run today.
**This will not be possible with this KEP** - while the container runtime operates on subpaths, kubelet always mounts the whole volume. The first mount with `-o context=X` will succeed, the second with `-o context=Y` will fail. The second Pod will be `ContainerCreating` as described above.

From this reason we propose to take [phased approach with this KEP](#implementation-phases).

### Examples

Following table captures interaction between actual filesystems on a volume and newly introduced behavior. AWS EBS CSI driver and NFS CSI drivers are used as an example of a volume based on a block device and a shared filesystem.

| Volume          | CSIDriver.SELinuxMountSupported | mount opts       | docker run -v |    |
|-----------------|---------------------------------|------------------|---------------|----|
| AWS EBS in-tree | N/A                             | `-o context=XYZ` |               | 1) |
| AWS EBS CSI     | true                            | `-o context=XYZ` |               | 2) |
| AWS EBS CSI     | unset or false                  |                  | `:Z`          | 3) |
| NFS1 CSI        | true                            | `-o context=XYZ` |               | 4) |
| NFS2 CSI        | unset or false                  |                  |               | 5) |

1) Kubelet knows that the in-tree AWS EBS plugin supports mounting with `-o context`. The mount option is then used (if pod context is known) and the container runtime does not relabel the volume.
2) AWS EBS CSI driver ships CSIDriver instance with `SELinuxMountSupported: true`. The behavior is the same as for in-tree volume plugin.
3) Here we show behavior of "old" CSI drivers, that ship their `CSIDriver` with `SELinuxMountSupported` unset (or `false`). Kubelet mounts the volume without any `-o context` option and detects that the volume supports SELinux (by inspecting mount options - it can find `seclabel` there). Therefore, it passes `:Z` to the container runtime to recursively relabel files on the volume. 

4) This must be a NFS CSI driver where **all** its volumes are independent NFS shares, because the CSI driver vendor (or cluster admin) set `SELinuxMountSupported: true`.
   Kubelet will mount the volumes with proper context.
5) This is a NFS CSI driver where the PVs may subdirectories of a bigger NFS share.
   `-o context` cannot be used by these volumes, because kernel knows they come from the same share and allows only the first mount from such share with `-o context`.
   Kubelet then mounts the volume without any extra options.
   It detects that the volume does not support SELinux (no `seclabel` in mount options after mount) and does not pass any `:Z` option to the container runtime.

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

1. Kubelet does not see any SELinux context set for the pod thus mounts `myclaim` PVC as usual and if the underlying volume supports SELinux, it passes it to the container runtime with ":Z".
   Kubelet passes also implicit Secret token volume with token with ":Z".
2. Container runtime allocates a new unique SELinux label for the pod and recursively relabels all volumes with ":Z" to this label. 


#### Story 2

User (or something else, e.g. an admission webhook) configures SELinux label for a pod.

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

1. Kubelet sees SELinux context in the pod. It files rest of the label from system defaults and gets `system_u:object_r:container_file_t:s0:c10,c0`.
2. Kubelet calls MountDevice() / SetUp() calls of the volume plugin with this explicit SELinux context.
3. The volume plugin (or CSI driver underneath), if it supports SELinux, adds `-o context=system_u:object_r:container_file_t:s0:c10,c0` to all its mount calls.
    * Here the CSI volume plugin checks `CSIDriver.SELinuxMountSupported` of the corresponding CSI driver.
4. Kubelet passes no SELinux option to CRI, resulting in no recursive `chcon` in the container runtime.

For example, OpenShift as a Kubernetes distribution, deploys a webhook that can inject SELinux context from namespace annotation into all Pods in the namespace.
Therefore, if configured properly, all Pods in the same namespace run with the same context and they can access data of each other.

### CSI driver considerations

CSI driver vendors need to explicitly opt-in their CSI drivers for this feature.

1. They must ship their CSIDriver instance with `CSIDriver.Spec.SELinuxMountSupported: true`.
2. They must run their CSI driver Pods with `/sys/fs/selinux` and `/etc/selinux/config` shared from the host via HostPath volumes!
   Because `/bin/mount` in the driver container evaluates these files and throws away any SELinux mount options if the files are not present.

We will document this requirement in our documentation that faces CSI driver vendors in gihub.com/kubernetes-csi/docs.

### Risks and Mitigations

This KEP changes behavior of Kubernetes when two pods with different SELinux contexts use the same volume. See [Behavioral changes](#behavioral-changes) above.
There is a risk that existing applications will get broken when the feature is enabled.
To mitigate this risk, we propose:

* Implement this feature in [several phases](#Implementation-phases).
* Expose metrics + alerts before the feature is enabled by default and could break anyone.

## Design Details

### Required kubelet changes

Apart from the obvious API change and behavior described above, kubelet + volume plugins need not so obvious changes: 

* Kubelet's VolumeManager needs to track which SELinux label should get a volume in global mount (to call `MountDevice()` with the right mount options).
  * It must call `UnmountDevice()` even when another pod wants to re-use a mounted volume, but it has a different SELinux context.
  * After kubelet restart, kubelet must reconstruct the original SELinux label it used to SetUp and MountDevice of each volume.
    * Volume reconstruction must be updated to get the SELinux label from mount (in-tree volume plugins) or stored json file (CSI).
      This label must be updated in VolumeManager's ActualStateOfWorld after reconstruction.
  * Reconciler must check also SELinux context used to mount a volume (both mounted devices and volumes) before considering what operation to take on a volume (`MountVolume` or `UnmountVolume`/`UnmountDevice` or nothing).
    It must throw proper error message telling that a Pod can't start because its volume is used by another Pod with a different SELinux context.
    * This is a good point to capture any metrics proposed below.
* Volume plugins will get SELinux context as a new parameter of `MountDevice` and `SetUp`/`SetupAt` calls (resp. as a new field in `DeviceMounterArgs` / `MounterArgs`).
  * Each volume plugin can choose to use the mount option `-o context=` (e.g. when `CSIDriver.SELinuxRelabelPolicy` is `true`) or ignore it (e.g. in-tree volume plugins for shared filesystems or when `CSIDriver.SELinuxRelabelPolicy` is `false` or `nil`).
  * Each volume plugin then returns `SupportsSELinux` from `GetAttributes()` call, depending on if it wants the container runtime to relabel the volume (`true`) or not (`false`; the volume was already mounted with the right label or it does not support SELinux at all).
    It will report error when the context in `/proc/mounts` does not match the expected value.
* When a CSI driver announces `SELinuxMountSupported: true`, kubelet will check that `-o context=X` was correctly applied after `NodePublish()`.
  It is a failure on CSI driver side, that it announces something that it is not able to fulfill.
  All pods that use such a volume will be ContainerCreating until the CSI driver fixes the mount (i.e., probably forever), with a message that it's CSI driver fault.
  This error is already part of generic `storage_operation_duration_seconds` metric (with a label for failures).
  * Note that kubelet can't check mount options after `NodeStage`, because a CSI driver does not need to mount during NodeStage or it may choose to mount to another directory than the staging one.

### Implementation phases

Due to change of Kubernetes behavior, we will implement the feature only for cases where it can't break anything first.

#### Phase 1
- Implement the feature only for volumes that are backed by PersistentVolumeClaims with `ReadWriteOncePod` access mode.
  Such volumes can be used only in a single pod and two pods can't ever conflict when using it.
- Collect metrics of how many other pods would fail because they use a RWO/RWX volume that's used by a pod with different SELinux context on the same node.
  TBD: create Info level alert ("please consider re-architecting your app not to share volumes this way and/or report it to sig-storage")?
  
This phase can go Beta (be enabled by default) or even GA without breaking anything.

#### Phase 2
Based on Phase 1 results:
- Extend the implementation to all volumes, i.e. to in-line volumes and PVCs with any access mode.
- Bump severity of the alert to Warning.
- Announce the behavior change and deprecate the old behavior.

If Phase 1 shows that too many applications would be broken, then go GA only with Phase 1, i.e. `ReadWriteOncePod` PVCs.
Even that will help users to avoid recursive relabeling of volumes if their application can use `ReadWriteOncePod` PVCs.

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
* Check that proper metric + alert is emitted when kubelet can't start two pods with different SELinux contexts using the same volume on the same node._
  * These tests might use only CSI volumes, GCE PD in-tree volume plugin that we use for e2e tests might be already migrated to CSI by that time.
* Prepare e2e job that runs with SELinux in Enforcing mode!

### Graduation Criteria

* Alpha of Phase 1:
  * Provided all tests defined above are passing and gated by the feature gate `SELinuxMountReadWriteOncePod` and set to a default of `false`.
  * Documentation exists.
* Beta of Phase 1:
  * The feature gate is `true` by default.
* Evaluation:
  * During the next release after Phase 1 is beta (= the feature is enabled by default), collect reports from users about possible breakage.
  * KEP author has access to usage data from OpenShift, a Kubernetes distro that runs with SELinux in enforcing mode.
* Alpha of Phase 2:
  * Only if nr. of broken apps is low!
    * To be discussed in sig-storage and sig-arch?.
  * Publish deprecation note about changed behavior.
  * Implement Phase 2 **with a separate alpha feature gate `SELinuxMount`**.
* GA: all known issues fixed + deprecation period is over. Otherwise, we will GA Phase 1 only.

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
    - Feature gate name: `SELinuxMountReadWriteOncePod`
    - Components depending on the feature gate: apiserver (API validation only), kubelet
  - [ ] Other
    - Describe the mechanism: 
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**
  Any change of default behavior may be surprising to users or break existing
  automations, so be extremely careful here.
  
  **YES!** See [Behavioral changes](#behavioral-changes) above.

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  Also set `rollback-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g. if this is runtime
  feature, can it break the existing applications?).
  
  Yes, it can be disabled / rolled back.
  Corresponding API fields get cleared and Kubernetes uses previous SELinux label handling.
  If the feature gate is enabled/disabled in kubelet without draining the node, volumes mounted by the previous kubelet are still mounted with the same mount option and thus may / may not have `-o context=` mount option.
  I.e. the disabled / enabled feature affects only newly started Pods.
  Kubelet can umount volumes mounted by the previous kubelet as usual.

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

  TBD: We propose a metric above, file its name here.
  
* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and do that now.

  TBD: this must be tested probably manually.

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

  **YES!** See [Behavioral changes](#behavioral-changes) and [Implementation phases](implementation-phases) above.

### Monitoring requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metrics. Operations against Kubernetes API (e.g.
  checking if there are objects with field X set) may be last resort. Avoid
  logs or events for this purpose.

  TBD

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

  TBD

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At the high-level this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide a comprehensive guidance, but at the very
  high level (they needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

  TBD

* **Are there any missing metrics that would be useful to have to improve
  observability if this feature?**
  Describe the metrics themselves and the reason they weren't added (e.g. cost,
  implementation difficulties, etc.).

  TBD

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  Think about both cluster-level services (e.g. metrics-server) as well
  as node-level agents (e.g. specific version of CRI). Focus on external or
  optional services that are needed. For example, if this feature depends on
  a cloud provider API, or upon an external software-defined storage or network
  control plane.

  For each of the fill in the following, thinking both about running user workloads
  and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high error rates on the feature:

  No deps.

### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirms the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
  focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)

  No new API calls are required. Kubelet / CSI volume plugin already has CSIDriver informer.
   
* **Will enabling / using this feature result in introducing new API types?**
  Describe them providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
  
  No new API types.

* **Will enabling / using this feature result in any new calls to cloud
  provider?**

  No new calls to cloud providers.
  
* **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**
  Describe them providing:
  - API type(s):
  - Estimated increase in size: (e.g. new annotation of size 32B)
  - Estimated amount of new objects: (e.g. new Object X for every existing Pod)
  
  CSIDriver gets one new field. We expect only few CSIDriver objects in a cluster.

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.
  
  No.

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data send and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits][].

  No.
  
### Troubleshooting

Troubleshooting section serves the `Playbook` role as of now. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now we leave it here though.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

  Kubelet can't start Pods "as usual" - it already has a `CSIDriver` informer.

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

  TBD

* **What steps should be taken if SLOs are not being met to determine the problem?**

  TBD

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

* 1.25: Alpha

## Drawbacks [optional]

* This KEP changes behavior of volumes shared by multiple pods, where each of them has a different SELinux label. See [Shared Volumes](#shared-volumes) for detail.
* The API is slightly different that `FSGroupChangePolicy`, which may create confusion.

## Alternatives [optional]

### `FSGroupChangePolicy` approach
The same approach & API as in `FSGroupChangePolicy` can be used.
**This is a viable option!**

If kubelet knows SELinux context that should be applied to a volume && hypothetical `SELinuxChangePolicy` is `OnRootMismatch`, it would check context only of the top-level directory of a volume and recursively `chcon` all files only when the top level dir does not match.
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
It could check the top-level of this subdir only and recursively change SELinux context there, however, this could leave different subdirectories of the volume with different SELinux labels and checking top-level directory only does not work.
With solution implemented in kubelet, we can always check top level directory of the whole volume and change context on the whole volume too.

### Move SELinux label management to kubelet
Right now, it's the container runtime who assigns labels to containers that don't have any specific `SELinuxOptions`.
We could move SELinux label assignment to kubelet.
This change would require significant changes both in kubelet (to manage the contexts) and CRI (to list used context after kubelet restart).
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
