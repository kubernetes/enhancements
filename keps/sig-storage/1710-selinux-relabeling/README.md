# Speed up SELinux volume relabeling using mounts

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [SELinux intro](#selinux-intro)
  - [SELinux label assignment](#selinux-label-assignment)
  - [Volume mounting](#volume-mounting)
  - [SELinux support in Kubernetes volumes](#selinux-support-in-kubernetes-volumes)
  - [Privileged containers](#privileged-containers)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    - [API changes](#api-changes)
      - [<code>CSIDriver</code>](#csidriver)
      - [<code>PodSecurityContext</code>](#podsecuritycontext)
    - [Conflicts with other Pods](#conflicts-with-other-pods)
    - [Single-node conflicts](#single-node-conflicts)
    - [Multiple-node conflicts](#multiple-node-conflicts)
  - [<code>CSIDriver</code> examples](#csidriver-examples)
  - [User Stories [optional]](#user-stories-optional)
    - [Story 1: default Pod](#story-1-default-pod)
    - [Story 2: Mount with <code>-o context</code> option](#story-2-mount-with--o-context-option)
    - [Story 3: cluster upgrade](#story-3-cluster-upgrade)
  - [CSI driver considerations](#csi-driver-considerations)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Required kubelet changes](#required-kubelet-changes)
    - [Volume Reconstruction](#volume-reconstruction)
  - [SELinuxController](#selinuxcontroller)
  - [Implementation phases](#implementation-phases)
    - [Phase 1](#phase-1)
    - [Phase 2](#phase-2)
    - [Phase 3](#phase-3)
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
  - [Mount <strong>all</strong> volumes with <code>-o context</code>, without any <code>SELinuxChangePolicy</code>](#mount-all-volumes-with--o-context-without-any-selinuxchangepolicy)
  - [Make SELinuxMount opt-in and not opt-out](#make-selinuxmount-opt-in-and-not-opt-out)
  - [Allow opt-in (or opt-out) globally via kubelet flags](#allow-opt-in-or-opt-out-globally-via-kubelet-flags)
  - [kube-state-metrics](#kube-state-metrics)
    - [PromQL](#promql)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [X] e2e Tests for all Beta API Operations (endpoints)
  - [x] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [x] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [x] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This KEP tries to speed up the way that volumes (incl. persistent volumes) are made available to Pods on systems with SELinux in enforcing mode.
Currently, the container runtime recursive relabels all files on a volume before a container can be started. This is slow for volumes with large amount of directories and files.

We propose to use mount option `-o context=XYZ` to set SELinux context of all files on a volume, without recursive walk through the volume.
The enhancement describes situations when such option can/cannot be used, why it's Kubernetes who must care about such a mount option, and possible breaking changes of the new Kubernetes behavior.

This KEP is split into three phases:
1. ReadWriteOncePod volumes are mounted with `-o context` by default.
   All other volumes are recursively relabeled by the container runtime.
   With feature gate `SELinuxMountReadWriteOncePod`, beta + on by default in v1.28.
2. Same as 1., but we provide metrics that show what Pods will break when *all* volumes are mounted with `-o context` and provide a proactive opt-out by setting `SELinuxChangePolicy: Recursive` in PodSpec.
   With feature gate `SELinuxChangePolicy`.
   Alpha in 1.32.
3. All volumes are mounted with `-o context` by default, users can opt-out by setting `SELinuxChangePolicy: Recursive` in PodSpec.
   With feature gate `SELinuxMount`, alpha without opt-out since 1.30, adding the opt-out field as alpha in 1.32.

Initially, we thought we could do 2. without opt-out, but we found that it may break valid use cases.

## SELinux intro
SELinux is a complex topic. Here is a brief overview of how it works in the container world.

On Linux machines with SELinux in enforcing mode, SELinux tries to prevent users that escaped from a container to access the host OS and also to access other containers running on the host.
It does so by running each container with an unique *SELinux label* (sometimes called *SELinux context*), such as `system_u:system_r:container_t:s0:c309,c383`, and labeling all content on all volumes with the corresponding label (`system_u:object_r:container_file_t:s0:c309,c383`).
Only process with the label `...:container_t:s0:c309,c383` can access files with label `container_file_t:s0:c309,c383`.
Therefore, a rogue user who escaped boundaries of its container, with label say `container_file_t:s0:c68,c222`, cannot access data of other containers, because the label of the attacker's process does not match any other container on the system.
Even processes running as root (UID 0) are denied access to these files, unless they run with the right SELinux label or as privileged containers.

Further in this KEP we assume that the SELinux is enabled on the system. This KEP has absolutely no effect on systems that run without SELinux. Kubelet already knows if SELinux is enabled and does not do anything with it if it's disabled or not available (e.g. on Windows).

See [SELinux documentation](https://selinuxproject.org/page/NB_MLS) for more details.

In this document we use `container_t` and `container_file_t` labels for container processes / files, which are the default labels on Fedora based distributions (AlmaLinux, CentOS, Red Hat Enterprise Linux, Rocky Linux, ...).
For example, Debian uses `svirt_lxc_net_t` and `svirt_lxc_file_t` as the default labels for containers, but the principles are the same.
The implementation of this KEP does not depend on the actual labels used in the system.

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
    * This applies to `/bin/mount` provided by [util-linux](github.com/util-linux/util-linux), which is the default in RHEL and Debian based distros.

Note that volumes mounted with `-o context` don't have `seclabel` in their mount options.
In addition, calling `chcon` there will fail with `Operation not supported`.

### SELinux support in Kubernetes volumes
Currently, Kubernetes *knows*(*) which volume plugins support SELinux (i.e. supports extended attributes on a filesystem the plugin provides).
If SELinux is supported for a volume, kubelet passes the volume to the container runtime with ":Z" option ([`selinux_relabel`](https://github.com/kubernetes/cri-api/blob/648d24775c39780dfc367536f00197be64534684/pkg/apis/runtime/v1/api.proto#L205) in CRI).
The container runtime then **recursively relabels** all files on the volume to either the label set in PodSpec/Container or the random value allocated by the container runtime itself.

*) These in-tree volume plugins don't support SELinux: HostPath, NFS and Portworx.
All other volume plugins support it.
This knowledge is hardcoded in the in-tree volume plugins (e.g. [NFS](https://github.com/kubernetes/kubernetes/blob/0c5c3d8bb97d18a2a25977e92b3f7a49074c2ecb/pkg/volume/nfs/nfs.go#L235)).

For CSI, kubelet currently uses following heuristics:

1. Let the CSI driver mount the volume as usual (via `NodeStage` + `NodePublish` CSI calls).
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

Side note about SELinuxOptions in a privileged Pod: at least CRI-O ignores `SELinuxOptions` in a privileged Pod.
The following Pod runs as `spc_t:s0`, not as `spc_t:s0:c967,c968`:

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
    - name: foo
      securityContext:
        privileged: true
        seLinuxOptions:
          level: "s0:c967,c968"
```

## Motivation

* File relabeling in CRI can be slow for big volumes.
* Avoiding out-of-space issues when relabeling almost full volumes. When a volume is almost full, CRI can fail to relabel volumes on it, since SELinux labels may need some little space on the volume.
* Allowing access to read-only volumes. A read-only volume can still be mounted with `-o context=XYZ` and provide files with the right labels to a Pod.
* Mounting volumes with the right SELinux label is a bit safer. Consider [CVE-2021-25741](https://access.redhat.com/security/cve/cve-2021-25741) - here a user can cause Kubernetes to provide host's filesystem to an innocent pod. Without this KEP, CRI will actually relabel the host's files so the Pod can access them. With this KEP, the attacker could still fool Kubernetes to mount host filesystem to the Pod, but the pod would not be able to access it, because (unprivileged) pods are denied accessing files on the host due to SELinux policy.

### Goals

* Mount volumes with the correct SELinux label using `-o context=XYZ` mount option and avoid recursive change of all files on the volume.
  * Do it _by default_ for all volumes.
  * Provide _explicit opt-out_ for workloads that mix privileged + unprivileged pods or pods with different SELinux labels.
    * Provide metrics + events to identify pods that can't start due to conflicts with other pods, so user can apply the opt-out.
    * Provide metrics + events to identify pods that are running now, however, they won't start if a feature gate gets enabled (to warn before upgrade / before enabling the feature gate).
    * Provide metrics + events to identify pods that are running now, however, they won't start if the pods end up on the same node (to warn after upgrade / after the feature gate is enabled).

### Non-Goals

* Change container runtimes / CRI.

## Proposal

* Allow CSI drivers to announce if they support mounting with `-o context` mount option via `CSIDriver spec.SELinuxMount bool` field.
  * Under `SELinuxMountReadWriteOncePod` feature gate, beta + enabled by default in 1.28.
  * Default value is `false`, i.e. no support for SELinux mount. It needs explicit opt-in.


* Update kubelet to mount all volumes with `-o context` mount option, unless a Pod explicitly opts out.
  * **Phase 1**: limited to ReadWriteOncePod (RWOP) volumes, under `SELinuxMountReadWriteOncePod` feature gate. RWOP volumes can't be used by multiple Pods and thus this cannot break any workload that mixes usage of the same volume by multiple Pods in parallel.
  * **Phase 2**: same as Phase 1, the feature is still limited to RWOP volumes.
    This phase enables opt-out that becomes active in Phase 3.
  * **Phase 3**: all volumes are mounted, under `SELinuxMount` feature gate. It can break existing applications that need to mix privileged and unprivileged Pods using the same volume in parallel.
    * We propose to send Pod events to immediately show why such Pods are stuck `ContainerCreating`.
    * We propose metrics and events to identify such Pods before the `SELinuxMount` feature gate is enabled, to identify potential issues before a cluster upgrade / enabling the feature gate.
    * We propose metrics and events to identify such Pods _after_ the `SELinuxMount` feature gate is enabled, to identify pods that are running only because they run on different nodes. If they landed on the same node, one of them would be stuck `ContainerCreating`, because they mix privileged and unprivileged pods.

* The opt-out is realized by a new Pod field `PodSpec.SecurityContext.SELinuxChangePolicy` with values `MountOption` and `Recursive` (opt-out). `null` means `MountOption`.
  * We need the field to be available in a cluster **before** `SELinuxMount` feature gate gets enabled, so cluster admins can fix their Pods and add opt-out before they upgrade to a version with `SELinuxMount` feature gate enabled.
  * Proposing a new feature gate `SELinuxChangePolicy`, alpha in 1.32.
    It must be enabled by default **before** the `SELinuxMount` feature gate is enabled by default.

Reason for a new field in `PodSpec.SecurityContext`:
* `FSGroupChangePolicy` with a similar purpose is there already.
* While it feels like a property of volume / PVC (all Pods that use a volume should use the same setting), we don't allow platform specific fields in PVCs.
* Another place would be a PV, but users can't edit it.

Reason for `MountOption` as the new default:
* Current telemetry numbers from OpenShift show that only a small number of clusters would be broken by this change. See [risks and mitigations](#risks-and-mitigations) for numbers.

* Report metrics and events about what Pods are not working / may not work after upgrade by a new kube-controller-manager SELinuxController.
    * These metrics need to cross-check *all* Pods on *all* Nodes, therefore it can't be in kubelet.
    * The kubernetes scheduler would be able to match only *schedulable* Pods, ignoring custom schedulers and DaemonSets.
    * This new controller will be opt-in, so it won't run on clusters that don't care about SELinux.
    * This new controller will provide only information (metrics, events), it won't affect Pod lifecycle.

To sum it up from a different perspective, we propose that kubelet mounts a Pod's volume with `-o context=XYZ` when *all* these conditions are met:
* Pod has `Pod.Spec.SecurityContext.SELinuxChangePolicy: MountOption` or `null` and `SELinuxMount` feature gate is enabled (Phase 3).
  * Or the volume is RWOP and only feature gate `SELinuxMountReadWriteOncePod` is enabled (Phase 1 and Phase 2).
* Pod has SELinux label set, at least in `Spec.SecurityContext.SELinuxOptions.Level` or all containers have `Spec.Containers[*].SecurityContext.SELinuxOptions.Level`.
  * When a `PodSecurityContext` or `SecurityContext` specifies incomplete SELinux label (i.e. omits `SELinuxOptions.User`, `.Role` or `.Type`), kubelet fills the blanks from the system defaults provided by [ContainerLabels() from go-selinux bindings](https://github.com/opencontainers/selinux/blob/621ca21a5218df44259bf5d7a6ee70d720a661d5/go-selinux/selinux_linux.go#L770).
    [See Story 2 below](#story-2).
* The CSI driver responsible for the volume announces support for mounting with `-o context` mount option by setting `CSIDriver.Spec.SELinuxMount: true` in the CSIDriver object.
  For in-tree volume plugins, kubelet has hardcoded knowledge about which volume plugins support `-o context` (iSCSI, FibreChannel) and which don't (all others, esp. NFS and all ephemeral volumes like EmptyDir, Secrets and ConfigMap).

When any of these conditions is not met, kubelet + the container runtime performs recursive relabeling of the volume as before.

When the volume is already mounted with a different SELinux label (or without it), the Pod is stuck `ContainerCreating` until the other Pod(s) that use the volume are deleted and the volume is unmounted.

### Implementation Details/Notes/Constraints [optional]

#### API changes

##### `CSIDriver`
Feature gate `SELinuxMountReadWriteOncePod`, **beta + on by default since Kubernetes 1.28.**

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

The default value is `false` to ensure backward compatibility.

##### `PodSecurityContext`
Feature gate `SELinuxChangePolicy`, **proposing alpha in 1.32.**

```go
// PodSecurityContext holds pod-level security attributes and common container settings.
// Some fields are also present in container.securityContext.  Field values of
// container.securityContext take precedence over field values of PodSecurityContext.
type PodSecurityContext struct {
// ...
	// seLinuxChangePolicy defines how the container's SELinux label is applied to all volumes used by the Pod.
	// It has no effect on nodes that do not support SELinux or when the volume does not support SELinux.
    // Valid values are "MountOption" and "Recursive". If not specified, "MountOption" is used.
	// It affects only in-tree iSCSI and FibreChannel volumes, and CSI volumes that announce "SELinuxMount: true" in their CSIDriver instance.
	// It affects only Pods that have SELinux label set, either in PodSecurityContext or in SecurityContext of all containers.
    // Note that this field cannot be set when spec.os.name is windows.
    // +optional
    SELinuxChangePolicy *PodSELinuxChangePolicy
}

// PodSELinuxChangePolicy defines how the container's SELinux label is applied to all volumes used by the Pod.
type PodSELinuxChangePolicy string

const (
	// Recursive relabeling of all Pod volumes by the container runtime.
	// This may be slow for large volumes, but allows mixing privileged and unprivileged Pods sharing the same volume on the same node.
    SELinuxChangePolicyRecursive PodSELinuxChangePolicy = "Recursive"
	// MountOption mounts all Pod volumes with `-o context` mount option.
	// This requires all Pods that share the same volume to use the same SELinux label.
	// It is not possible to share the same volume among privileged and unprivileged Pods.
    SELinuxChangePolicyMountOption PodSELinuxChangePolicy = "MountOption"
)
```

#### Conflicts with other Pods

It is Pod (Deployment, StatefulSet, DaemonSet) author responsibility to set the `SELinuxChangePolicy` and SELinux labels on Pods correctly.
For example, these cases can happen if they are careless.

#### Single-node conflicts
All these cases assume that the volume used in the examples does support mount with `-o context` and all Pods run on the same node, and they access the same volume.

**Mixing Pods with different SELinux labels**: In both cases we can see that only one of the pods can run and access the volume, but they differ in which Pod it is.
* Pod A with `SELinuxChangePolicy: Recursive` and label `c1,c2` runs, Pod B with `SELinuxChangePolicy: Recursive` with label `c8,c9` is about to start.
  * This is the behavior before this KEP. All files on the volume were relabeled to `c1,c2` when A started. When Pod B starts, the container runtime relabels all files on the volume to `c8,c9` and Pod A loses access to data on the volume (i.e. it will get `EPERM` to all OS calls).
* Pod A with `SELinuxChangePolicy: MountOption` and label `c1,c2` runs, Pod B with `SELinuxChangePolicy: MountOption` with label `c8,c9` is about to start.
  * Kubelet mounted the volume with `-o context=c1,c2` for Pod A. Pod B wants the same volume mounted with `-o context=c8,c9`. Pod B will stay in `ContainerCreating` state until Pod A finishes and kubelet unmounts the volume, so it can be re-mounted for Pod B. Kubelet will provide an event that describes why Pod B is not running.


**Mixing different `SELinuxChangePolicy` values**: Only one of Pods can run, all others are stuck `ContainerCreating`:
* Pod A with `SELinuxChangePolicy: MountOption` runs, Pod B with `SELinuxChangePolicy: Recursive` is about to start.
  * Kubelet mounted the volume with `-o context=<a label>` for Pod A. Pod B wants the same volume mounted with no `-o context`. Pod B will stay in `ContainerCreating` state until Pod A finishes and kubelet unmounts the volume, so it can be re-mounted for Pod B.
* Pod A with `SELinuxChangePolicy: Recursive` runs, Pod B with `SELinuxChangePolicy: MountOption` is about to start.
  * Kubelet mounted the volume without `-o context` for Pod A. Pod B wants the same volume mounted with `-o context=<b label>`. Pod B will stay in `ContainerCreating` state until Pod A finishes and kubelet unmounts the volume, so it can be re-mounted for Pod B.

**Mixing privileged and unprivileged Pods**: This is the largest behavior change in this KEP.
* A privileged pod A runs with *any* `SELinuxChangePolicy`, Pod B with `SELinuxChangePolicy: Recursive` with label `c8,c9` is about to start.
  * Kubelet mounted the volume without `-o context` for Pod A (`spc_t` does not have any `cX,cY` categories) and the volume was not relabeled. When Pod B starts, the container runtime relabels all files on the volume to `c8,c9`. **Both pods can run in parallel** and access the volume (A is privileged / `spc_t`, B has the right label).
* A privileged pod A runs with *any* `SELinuxChangePolicy`, Pod B with `SELinuxChangePolicy: MountOption` with label `c8,c9` is about to start.
  * Kubelet mounted the volume without `-o context` for Pod A (`spc_t` does not have any `cX,cY` categories) and the volume was not relabeled. Pod B wants the same volume mounted with `-o context=c8,c9`. Pod B will stay in `ContainerCreating` state until Pod A finishes and kubelet unmounts the volume, so it can be re-mounted for Pod B. **A and B cannot run in parallel**.

**Mixing Pods with different SELinux labels and subpaths**: This is a behavior change in this KEP. We expect that it does not affect anyone or only a very small number of users.
* Pod A with `SELinuxChangePolicy: Recursive` and label `c1,c2` runs and uses subpath `/dir1` in its Pod definition, Pod B with `SELinuxChangePolicy: Recursive` with label `c8,c9`, using subpath `/dir2` of the volume is about to start.
    * This is the behavior before this KEP. The container runtime relabelled only the subpath `/dir1` with `c1,c2` when A started. When Pod B starts, the container runtime relabels only the subpath `dir2` with `c8,c9`. Both Pod A and Pod B can run in parallel and access their subpaths.
  * This situation will be reported in the metrics proposed below, so the user can fix it before `SELinuxMount` is enabled by default. Actually, the metrics won't care about subpaths, it will look like two pods sharing the whole volume.
* Pod A with `SELinuxChangePolicy: MountOption` and label `c1,c2` runs and uses subpath `/dir1` in its Pod definition, Pod B with `SELinuxChangePolicy: MountOption` with label `c8,c9`, using subpath `/dir2` of the volume is about to start.
    * Kubelet mounted **the whole volume** with `-o context=c1,c2` for Pod A. Pod B will stay in `ContainerCreating` state until Pod A finishes and kubelet unmounts the volume, so it can be re-mounted for Pod B.
    * This situation will be reported in the metrics proposed below.

#### Multiple-node conflicts

**Mixing Pods with different SELinux labels on different nodes**: There is a slight difference in behavior.
* Pod A with `SELinuxChangePolicy: Recursive` and label `c1,c2` runs on Node Alpha, Pod B with `SELinuxChangePolicy: Recursive` with label `c8,c9` is about to start on node Beta.
    * This is the behavior before this KEP. All files on the volume were relabeled to `c1,c2` when A started. When Pod B starts, the container runtime relabels all files on the volume to `c8,c9` and Pod A loses access to data on the volume (i.e. it will get `EPERM` to all OS calls).
* Pod A with `SELinuxChangePolicy: MountOption` and label `c1,c2` runs on Node Alpha, Pod B with `SELinuxChangePolicy: MountOption` with label `c8,c9` is about to start on node Beta.
    * Both pods can run. Mounting the volume with `-o context` is node-local operation, so the volume can be mounted with different `-o context` options on different nodes.
    * This situation will be reported in the metrics proposed below, so the user can fix it before the pods end up on the same node.
* Pod A with `SELinuxChangePolicy: Recursive` and label `c1,c2` runs on Node Alpha, Pod B with `SELinuxChangePolicy: MountOption` with label `c8,c9` is about to start on node Beta.
    * Both pods can run. On node Alpha, the volume will be recursively relabeled and Pod A can access it. On node Beta, the volume will be mounted with `-o context=c8,c9`, ignoring how the files are labeled in the storage backend. Pod B can access it.
  * This situation will be reported in the metrics proposed below, so the user can fix it before the pods end up on the same node.

Those are only examples, more cases can happen. In general, a Pod that wants the volume mounted with a different `-o context` option than it's currently used must wait for all pods that use the volume to get deleted and the volume unmounted.

### `CSIDriver` examples

Following table illustrates usage of CSIDriver `Spec.SELinuxMount` field.
Using AWS EBS CSI driver and NFS CSI drivers are used as an example of a volume based on a block device and a shared filesystem.

| Volume        | CSIDriver.SELinuxMount | mount opts                     | docker run -v |    |
|---------------|------------------------|--------------------------------|---------------|----|
| iSCSI in-tree | N/A                    | `-o context=XYZ`               |               | 1) |
| AWS EBS CSI   | true                   | `-o context=XYZ`               |               | 2) |
| AWS EBS CSI   | unset or false         |                                | `:Z`          | 3) |
| NFS1 CSI      | true                   | `-o context=XYZ,noshareacache` |               | 4) |
| NFS2 CSI      | unset or false         |                                |               | 5) |

1.Kubelet *knows* that the in-tree iSCSI plugin supports mounting with `-o context`. The mount option is then used (if pod label is known) and the container runtime does not relabel the volume.

2. AWS EBS CSI driver ships CSIDriver instance with `SELinuxMount: true`. The behavior is then the same as for in-tree volume plugin.

3. Here we show behavior of "old" CSI drivers, that ship their `CSIDriver` with `SELinuxMount` unset (or `false`).
   Kubelet mounts the volume without any `-o context` option and detects that the volume supports SELinux (by inspecting mount options - it can find `seclabel` there).
   Therefore, it passes `:Z` to the container runtime to recursively relabel files on the volume.

4. This must be a NFS CSI driver that detects `-o context` mount option and automatically adds `nosharecache` to allow mounting the same volume with different SELinux label on the same node.
   The CSI driver announces this capability by setting `SELinuxMount: true` in its CSIDriver instance.
   Kubelet will mount the volumes with proper label.

5. This is a NFS CSI driver that does not support `-o context` mount option.
   Kubelet then mounts the volume without any extra options.
   There is no `seclabel` in the mount options, so kubelet does not pass any `:Z` to the container runtime, the volume does not support SELinux labels.

### User Stories [optional]

#### Story 1: default Pod

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

1. Kubelet does not see any SELinux label set for the pod thus mounts `myclaim` PVC as usual and if the underlying volume supports SELinux, it passes it to the container runtime with ":Z".
   Kubelet passes also implicit Secret token volume with ":Z".
2. Container runtime allocates a new unique SELinux label for the pod and recursively relabels all volumes with ":Z" to this label.

This KEP does not change anything in this story.

#### Story 2: Mount with `-o context` option

User (or something else, e.g. an admission webhook) configures SELinux label for a pod using a volume

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

**Feature gates `SELinuxMountReadWriteOncePod == false` && `SELinuxMount == false`**:
* Same as the previous story. Kubelet mounts the volume without any SELinux option + the container runtime relabels the volumes recursively.

**Feature gates `SELinuxMountReadWriteOncePod == true` && `SELinuxMount == false`**:
* If `myclaim` is a RWOP volume (`Spec.AccessModes == ["ReadWriteOncePod']`)  *and* the corresponding CSI drivers support SELinux mount, kubelet fills the blanks in the `seLinuxOptions` from the system defaults (`user: system_u`, `role: object_r`, `type: container_t` on Fedora based distros), translates them to a file label (`container_t` -> `container_file_t`) and mounts the volume with `-o context=system_u:object_r:container_file_t:s0:c10,c0`.
* If `myclaim` is any other volume, kubelet mounts the volume without any SELinux option + the container runtime relabels the volume recursively.
* The secret token volume is relabeled by the container runtime, because Secret and Projected volumes do not support SELinux mount.

**Feature gates `SELinuxMountReadWriteOncePod == true` && `SELinuxMount == true`**:
* Since there is no `SELinuxChangePolicy` set, kubelet implies `MountOption`.
  If the corresponding CSI driver (or in-tree volume plugin) support SELinux mount, kubelet fills the blanks in the `seLinuxOptions` from the system defaults as described above and the volume is mounted with `-o context=system_u:object_r:container_file_t:s0:c10,c0`.
* Otherwise, kubelet mounts the volume without any SELinux option + the container runtime relabels the volume recursively.
* The secret token volume is relabeled by the container runtime, because Secret and Projected volumes do not support SELinux mount.

**Feature gates `SELinuxMountReadWriteOncePod == false` && `SELinuxMount == true`**:
* Invalid configuration, kubelet behaves as if `SELinuxMount == false`, i.e. recursive relabeling for everything.
  (Should kubelet `exit()` here?).

For example, OpenShift as a Kubernetes distribution deploys a webhook that can inject SELinux label from namespace annotation into all Pods in the namespace.
Therefore, if configured properly, all Pods in the same namespace run with the same label and they can access data of each other.

#### Story 3: cluster upgrade

1. Cluster admin runs a cluster with all feature gates disabled (i.e. `SELinuxMountReadWriteOncePod == false` && `SELinuxMount == false` && `SELinuxChangePolicy == false`).
  Kubelet / the container runtime recursively relabels all volumes.
2. Cluster admin updates to 1.28, with `SELinuxMountReadWriteOncePod` enabled by default.
   Kubelet / the container runtime recursively relabels all volumes except for RWOP.
   Since RWOP volume can be used only in a single Pod, no application should break.
   (Nobody complained so far).
    * At this point, kubelet starts reporting `volume_manager_selinux_volume_context_mismatch_warnings_total` metric.
3. Cluster admin updates to 1.N, where `SELinuxChangePolicy` is enabled by default and enables the new SELinuxController in kube-controller-manager.
  Kubelet mount behavior stays the same as it is in the previous step.
  Cluster admin can check the cluster metrics and they can proactively opt out from mounting all volumes with SELinux by setting `SELinuxChangePolicy: Recursive` in all Pods that need to mix privileged and unprivileged Pods.
  The field accepts only value `Recursive` at this stage!
  The field value has no effect on volume mounting or relabeling, however, it is reflected in `volume_manager_selinux_volume_context_mismatch_warnings_total` and `selinux_warning_controller_selinux_volume_conflict` metrics.
  The cluster admin can see that nr. of problematic Pods decreases.
    * A kubernetes distribution may choose to block upgrade to 1.M (that has `SELinuxMount` enabled), until the cluster admin fixes all problematic Pods.
4. Cluster admin updates to 1.M, where `SELinuxMount` is enabled by default.
  `SELinuxChangePolicy` field now accepts `MountOption` value and kubelet uses the field when mounting all volumes.

There can be multiple releases between 1.N and 1.M, gradually tightening the upgrade criteria and severity of potential alerts.

### CSI driver considerations

CSI driver vendors need to explicitly opt-in their CSI drivers for this feature.

1. They must ship their CSIDriver instance with `CSIDriver.Spec.SELinuxMount: true`.
2. They must run their CSI driver Pods with `/sys/fs/selinux` and `/etc/selinux/config` shared from the host via HostPath volumes!
   Because `/bin/mount` in the driver container evaluates these files and throws away any SELinux mount options if the files are not present.

We will document this requirement in our documentation that faces CSI driver vendors in gihub.com/kubernetes-csi/docs.

### Risks and Mitigations

This KEP changes the default behavior of Kubernetes.
We provide set of metrics and events to detect potential issues before the feature is enabled by default, so users can fix their workloads before the change.

Phase 1 + 2 (no breaking change yet):
* `volume_manager_selinux_volume_context_mismatch_warnings_total`: counter of Pods that 100% would not start if `SELinuxMount` feature gate is enabled.
  * It is emitted only when the `SELinuxMountReadWriteOncePod` feature gate is enabled (on by default in 1.28).
  * It is emitted by kubelet, in the code path that will really block starting a pod when `SELinuxMount` feature gate is enabled.
  * It misses Pods that run on different nodes that would not run if they landed on the same node.
* `selinux_warning_controller_selinux_volume_conflict`: names of Pods that may not start if `SELinuxMount` feature gate is enabled *and* the Pods land on the same node.
* SELinuxController, if enabled, will send events to Pods that may not start if `SELinuxMount` feature gate is enabled.

As of 2024-09-04, telemetry numbers from OpenShift show that less than 2% of the clusters have `volume_manager_selinux_volume_context_mismatch_warnings_total > 0`.
More than 50% of these potentially-broken clusters have less than 10 of such Pods.

As of 2026-01-13, telemetry numbers from OpenShift show:
* ~0.9% of the clusters with Kubernetes 1.32 and newer have `volume_manager_selinux_volume_context_mismatch_warnings_total > 0`.
* ~0.3% of the clusters with Kubernetes 1.33 and newer have any SELinux warning controller warnings. Admins of those clusters need to evaluate the warnings and fix their Pods before Kubernetes upgrade to a version where `SELinuxMount` is GA.
We consider these numbers good enough to consider Phase 2 beta successful and proceed with GA, aiming at Phase 3 GA in the subsequent release.

Phase 3:
* `volume_manager_selinux_volume_context_mismatch_errors_total`: counter of Pods that did not start.
  * It is emitted only when the `SELinuxMount` feature gate is enabled.
  * It is emitted by kubelet when a Pod really cannot start.
  * It misses Pods that run on different nodes that would not run if they landed on the same node.
  * Since kubelet re-tries to start Pods periodically, the counter will increase until the Pod starts or is deleted.

* Kubelet will send pod events when a pod can't start, because its volume(s) are mounted with different `-o context` option (or without it).
  * Either with names of the conflicting Pods, when they are in the same namespace.
  * Or just generic "volume XYZ is already mounted with a different SELinux label" when A and B are in different namespaces, so users cannot peek into other namespaces.
* SELinuxController, if enabled, will send events to Pods that may not start if they land on the same node.

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

### SELinuxController

A new kube-controller-manager (KCM) controller that watches Pods, PVCs, PVs and CSIDrivers and reports metrics and events when it detects a conflict.
The controller is enabled only when SELinuxChangePolicy feature gate is enabled.
Even when the feature gate is GA, the controller is opt-in, as most Kubernetes deployments do not use SELinux.

The controller can safely ignore Pod updates, because Pod volumes and any SELinux options cannot be updated after creation.
However, the controller needs to watch for PVC, PV and CSIDriver updates, because they can be created or updated (e.g. bound) after a Pod that uses them is created.

When it sees a pair of Pods that conflict, it emits a metric and an event to both Pods.
* The event either mentions both Pods by name (if both are in the same namespace) or just something generic (when they are in different namespaces, so the event does not leak sensitive information).
  The controller keeps in-memory list of Pods+volumes that were reported, so it does not emit the same event again on re-sync.
  The same event may be sent after KCM restart.
* The metric is called `selinux_warning_controller_selinux_volume_conflict`.
  Value of the metric for a given pod1+pod2 combination is either 1 or the metric is not reported at all.
  In a cluster without any conflicts, the metrics is empty.
  * Example of SELinuxChangePolicy conflict on two pods:
    ```
    selinux_warning_controller_selinux_volume_conflict{
      pod1_name="testpod-c1",
      pod1_namespace="default",
      pod1_value="MountOption",
      pod2_name="testpod-c2",
      pod2_namespace="default",
      pod2_value="Recursive",
      property="SELinuxChangePolicy"}
    ```
  * Example of SELinux label conflict on two pods:
    ```
    selinux_warning_controller_selinux_volume_conflict{
      pod1_name="testpod-c1",
      pod1_namespace="default",
      pod1_value="system_u:object_r:container_file_t:s0:c0,c1",
      pod2_name="testpod-c2",
      pod2_namespace="default",
      pod2_value="system_u:object_r:container_file_t:s0:c0,c2",
      property="SELinuxLabel"}
    ```
    * This metric must not be reported for Privileged and unprivileged Pods with both having `SELinuxChangePolicy: Recursive`.
    * This metric must not be reported for two Pods with `SELinuxChangePolicy: Recursive` and two different labels using the same volume.
      User has explicitly opted out from the SELinuxMount behavior, they must have other means how to run two pods with different SELinux labels, for example anti-affinity or using different subpath of the volumes.
* A cluster admin can list conflicting pods by querying the metric easily.
    
TBD: limit the metric to X thousands of conflicts?

Drawbacks:

* The controller does not evaluate any Pod (anti) affinity or other scheduling rules or custom schedulers.
  It may report a conflict even when the Pods may never run on the same node.
* The controller may report a conflict when two Pods are scheduled to the same node, but they will run serially there.
  For example, one pod is already being deleted and the other has just been scheduled there.
  Kubelet's `volume_manager_selinux_volume_context_mismatch_warnings_total` metric is more accurate in this case.
* The controller cannot read the SELinux default container labels from the operating system.
  KCM often runs in a container and does not have access to `/etc/selinux` on the worker nodes.
  As consequence, two labels that are equivalent from the SELinux point of view, may be reported as different, such as these two `seLinuxOptions` snippets: `{"type": "container_t", "level": "s0:c10,c0"}` and `{"level": "s0:c10,c1"}`.
  `container_t` is the default type label for containers on Fedora, so kubelet is able to fill it in the `seLinuxOptions` when it is not set and see they're equivalent.
  KCM does not know the default on nodes and treats empty fields in `seLinuxOptions` as *uncomparable* - it does not emit any event in the above example.

### Implementation phases

Due to change of Kubernetes behavior, we will implement the feature only for cases where it can't break anything first.

#### Phase 1
- Implement the feature only for volumes that are backed by PersistentVolumeClaims with `ReadWriteOncePod` access mode.
  Such volumes can be used only in a single pod and two pods can't ever conflict when using it.
- Collect metrics of how many other pods would fail because they use a RWO/RWX volume that's used by a pod with different SELinux label on the same node.

This phase went Beta (enabled by default) in Kubernetes 1.28, without `SELinuxChangePolicy` field in PodSpec!

#### Phase 2
Based on Phase 1 results:
- Introduce `SELinuxChangePolicy` field, under `SELinuxChangePolicy` feature gate.
  - In this phase, the only allowed value is `Recursive`.
  - The field is not interpreted, it's used only by cluster admins to proactively opt-out from `SELinuxMount`
  - We need to graduate it to Beta / enabled by default before enabling `SELinuxMount` feature gate.
- Implement the new SELinuxController.

#### Phase 3
- Actually interpret `SELinuxChangePolicy` field in PodSpec.

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
* Check that kubelet emits proper metrics when it can't start two pods with different SELinux labels using the same volume on the same node._
* Check that the SELinux warning controller emits events when pods conflict + emit the described metrics.
* Prepare e2e job that runs with SELinux in Enforcing mode.
  * Done:
    * https://testgrid.k8s.io/kops-k8s-ci#kops-aws-selinux: for features enabled by default.
    * https://testgrid.k8s.io/kops-k8s-ci#kops-aws-selinux-alpha: for all alpha features enabled.
    * https://testgrid.k8s.io/kops-distro-rhel8#kops-aws-selinux-changepolicy: for `SELinuxChangePolicy` enabled + `SELinuxMount` disabled.
    * https://testgrid.k8s.io/presubmits-kubernetes-nonblocking#pull-kubernetes-e2e-gce-storage-selinux: for PRs (needs explicit `/test ` in a PR).

All these e2e tests use only CSI volumes. All in-tree volume types that support SELinux and dynamic provisioning were migrated to CSI already.

### Graduation Criteria

* Alpha of Phase 1:
  * Provided all tests defined above are passing and gated by the feature gate `SELinuxMountReadWriteOncePod` and set to a default of `false`.
  * Documentation exists.
* Beta of Phase 1:
  * E2e tests implemented + green.
  * The feature gate is `true` by default.
* Evaluation:
  * During the next release after Phase 1 is beta (= the feature is enabled by default), collect reports from users about possible breakage.
  * KEP author has access to usage data from OpenShift, a Kubernetes distro that runs with SELinux in enforcing mode.
* Alpha of Phase 2 + 3:
  * Implemented `SELinuxChangePolicy` **with a separate alpha feature gate `SELinuxChangePolicy`** as preparation for `SELinuxMount` feature gate graduation.
  * Implemented SELinuxController.
* Beta of Phase 2 + 3 (`SELinuxChangePolicy` is beta and enabled by default; `SELinuxMount` is beta, but disabled by default).
  * E2e tests implemented + green.
  * Telemetry numbers from OpenShift show that <5% of clusters would need to change any of their Pods.
  * This phase signalizes that the feature is ready for real testing.
    Only non-breaking parts (`SELinuxChangePolicy`) are enabled by default.
    Users willing to test `SELinuxMount` must enable it explicitly.
* GA of Phase 2 (`SELinuxChangePolicy` + `SELinuxMountReadWriteOncePod` are GA and locked to default, `SELinuxMount` is beta and disabled by default):
  * All known issues fixed. Otherwise, we will GA Phase 1 only.
  * Users can update their clusters safely, there is no breaking change yet.
    Users willing to test `SELinuxMount` must enable it explicitly.
  * This phase allows production clusters to check what Pods (Deployments, StatefulSets) need update and fix them before the breaking part (`SELinuxMount`) is enabled by default in the next phase.
* GA of Phase 3 (`SELinuxMount` is GA and locked to default):
  * At least 1 release after `SELinuxChangePolicy` is GA to give cluster admins enough time to apply `SELinuxChangePolicy` to their Pods.
  * Telemetry numbers from OpenShift show that <2% of clusters would need to change any of their Pods (i.e. most clusters already applied opt-out).
  * This is the phase that may break existing applications during cluster upgrade.
    Users that use SELinux should carefully evaluate the metrics emitted by kubelet and SELinuxWarningController and fix their workloads before upgrade to this version.

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

When `SELinuxMount` feature gate gets enabled in a cluster, it can break existing applications, see [Conflicts with other Pods](#conflicts-with-other-pods) for an example.

We will provide metrics to detect potential issues before the feature is enabled by default, so users can fix their workloads before the change. See [Risks and Mitigations](#risks-and-mitigations).

Enabling / disabling the feature gate in a kubelet requires node to be drained, otherwise kubelet may remount volumes from running Pods.
Kubelet upgrade requires the same.

### Version Skew Strategy

The newly introduced API fields are used only by kubelet.
We expect the usual upgrade / downgrade strategy (API server version must be higher or equal to kubelet version), and the same for the feature gates (API server must have the feature gate enabled before it's enabled in a kubelet).

Therefore, kubelet with `SELinuxMountReadWriteOncePod` or `SELinuxMount` enabled can expect that the API server provides corresponding API fields.

## Production Readiness Review Questionnaire

### Feature enablement and rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: `SELinuxMountReadWriteOncePod` (beta in 1.28)
    - Feature gate name: `SELinuxChangePolicy` (alpha in 1.30, proposing beta in 1.33)
      - To enable `SELinuxChangePolicy` feature gate, `SELinuxMountReadWriteOncePod` **must** be enabled too.
    - Feature gate name: `SELinuxMount` (alpha in 1.30, proposing beta in 1.33)
      - To enable `SELinuxMount` feature gate, `SELinuxMountReadWriteOncePod` and `SELinuxChangePolicy` **must** be enabled too.
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

  **Yes.** See [Conflict with other Pods](#conflicts-with-other-pods) for details.
  We offer metrics + events + proactive opt-out per Pod before the breaking part (`SELinuxMount`) is enabled by default.

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  Also set `rollback-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g. if this is runtime
  feature, can it break the existing applications?).

  Yes, it can be disabled / rolled back.

  * When `SELinuxMountReadWriteOncePod` feature gate is disabled, corresponding
    `CSIDriver.SELinuxMount` API field get cleared and Kubernetes uses previous SELinux label handling.

    * If the feature gate is enabled/disabled in kubelet without draining the node,
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

  Existing workloads can be affected, this KEP changes the default behavior of kubelet.
  Pods can be `ContanerCreating` for undefined time.
  In this kep we propose events and metrics and an opt-out mechanism, all of them available *before* the feature is enabled by default.

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
  It will be tested before `SELinuxMount` gets enabled by default.

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
    - All `*_errors_total` metrics below cover real errors when a Pod can't start.
      - It applies to `ReadWriteOncePod` volumes when only `SELinuxMountReadWriteOncePod` feature gate is enabled.
      - It applies to all volumes when both `ReadWriteOncePod` and `SELinuxMount` feature gate is enabled.
    - All `*_warnings_total` metrics below are reported when only `SELinuxMountReadWriteOncePod` feature gate is enabled and shows **future** errors that would appear after both `SELinuxMountReadWriteOncePod` and `SELinuxMount` feature gates are enabled.
      This will be evaluated in Phase 2.
        1. `volume_manager_selinux_container_errors_total` + `volume_manager_selinux_container_warnings_total`: Number of errors when kubelet cannot compute SELinux label for a container.
          This indicates an error converting Pod's SELinuxOptions into SELinux label string by github.com/opencontainers/selinux/go-selinux library.
          Reading its source code, this should never happen, but one never knows.
          Labels: none, this error is not related to volumes. 
        1. `volume_manager_selinux_pod_context_mismatch_errors_total` + `volume_manager_selinux_pod_context_mismatch_warnings_total`: Number
          of errors when a Pod defines different SELinux labels for its containers that use the same volume.
          Before this feature, only one container in such a Pod could access the volume.
          With this feature, the Pod won't even start.
          This metric captures nr. of failed Pod starts, including periodic retries.
          Labels: `access_mode`, to determine issues with RWOP volumes.
        1. `volume_manager_selinux_volume_context_mismatch_errors_total` + `volume_manager_selinux_volume_context_mismatch_warnings_total`:
          Number of errors when a Pod uses a volume that is already mounted with a different SELinux label than the Pod needs.
          Before this feature, both pods would start, but only one such pod could access the volume.
          With this feature, one of the Pods won't even start.
          Labels: `access_mode`, to determine issues with RWOP volumes, and `volume_plugin` to match a volume plugin / CSI driver and its `SELinuxMount` flag.   
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
This did not work even before this KEP (except for the case where a privileged and unprivileged Pods shared the same volume), now it's just more obvious.
IMO we can't base SLO on this.

* **Are there any missing metrics that would be useful to have to improve
  observability if this feature?**
  Describe the metrics themselves and the reason they weren't added (e.g. cost,
  implementation difficulties, etc.).

No.

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**

For extended metrics and events, SELinuxController must be enabled in KCM.

### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirms the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**

  * No new API calls are required in kubelet, its CSI volume plugin already has CSIDriver informer.
  * KCM will emit new events when SELinuxWarningController is enabled. It already has Pod, PV, PVC, CSIDriver informers and does not do other API calls.

* **Will enabling / using this feature result in introducing new API types?**

  No new API types.

* **Will enabling / using this feature result in any new calls to cloud
  provider?**

  No new calls to cloud providers.

* **Will enabling / using this feature result in increasing size or count of the existing API objects?**

  * CSIDriver gets one new field. We expect only few CSIDriver objects in a cluster.
  * PodSpec gets one new field, and we expect it to be `null` for the vast majority of Pods.
  * Event(s) will be created for every conflicting Pod pair when SELinuxWarningController is enabled.

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

  No. KCM and Kubelet already has a cache of desired / existing mounts, we need to add
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
      In addition, each such Pod has an event about SELinux label mismatch.
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
  * On-by-default reverted, because of a volume reconstruction issue.
* 1.28: Beta, on by default.
* 1.30: `SELinuxMountReadWriteOncePod` still beta, SELinuxMount (early) alpha.
  * Implement bare minimum of `SELinuxMount` for experiments, including:
    * Extend SELinux mount to all volume access modes.
    * Add label with volume access mode to `volume_manager_selinux_volume_context_mismatch_errors_total` and similar metrics.
* 1.32: `SELinuxMountReadWriteOncePod` still beta, `SELinuxMount` alpha, `SELinuxChangePolicy` alpha.
  * We discovered that sharing volumes between privileged and unprivileged containers as described [here](#privileged-containers) is a valid use case.
    we cannot mount *all* volumes with `-o context` and it must be an explicit opt-out using `SELinuxChangePolicy: Recursive`.
  * Implement `SELinuxChangePolicy` as an alpha field.
* 1.33: Graduate `SELinuxMount` to beta / disabled by default, `SELinuxChangePolicy` to beta / enabled by default.
  * Add e2e tests for the SELinuxWarningController.
  * Test on non-Fedora based Linux distribution (e.g. Debian) with SELinux enabled.
* 1.36: Graduate `SELinuxMountReadWriteOncePoc` and `SELinuxChangePolicy` to GA.
  * Keep `SELinuxMount` beta / disabled by default.
  * Publish a blog or similar documentation that upgrade from 1.36 to 1.37 may introduce breaking changes on clusters with SELinux enabled and cluster admins are advised to check metrics and events.
  * This is based on favorable telemetry data reported by OpenShift on 2026-01-13, see [Risks and Mitigations](#risks-and-mitigations).
* Optimistic plan: 1.37 GA of everything.

## Drawbacks [optional]

* The API is slightly different that `FSGroupChangePolicy`, which may create confusion.
* The feature changes the default behavior.
* The proposed metrics may be complex to gather (see Appendix) *and* they may miss a short-living Pods.

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

### Make SELinuxMount opt-in and not opt-out

I.e. the old behavior is still the default.

* Safer, there is no behavior change. Users that want to benefit from the new feature must opt-in.
* Mounting with SELinux is safe for RWOP volumes, and therefore we would like to have it enabled by default.
  The resulting `SELinuxChangePolicy` values would be:
  * `MountReadWriteOncePod` - mount RWOP volumes with SELinux mount option, recursive relabeling for the rest. This is the default.
  * `Recursive` - explicit opt-out.
  * `MountOption` - mount all volumes with SELinux mount option, needs explicit opt-in.
  * We consider such API harder to understand.

### Allow opt-in (or opt-out) globally via kubelet flags

Instead of `SELinuxChangePolicy` in every PodSpec, we considered having a global flag to enable or disable the feature for all Pods in a cluster.
For example by a new kubelet flag `--enable-selinux-mount`.

This is a good "plan B", if the API changes proposed here are not acceptable.

### kube-state-metrics
Instead of adding a new SELinuxController to KCM, we considered addin new metrics to kube-state-metrics.
This is a viable option to report *metrics* about conflicting Pods, but kube-state-metrics can't send *events*.

Today, kube-state-metrics exports metrics that allow users to join Pods with their PVCs and PVs and e.g. see how many Pods use the same PV.

We need to add:

* `kube_pod_security_context`: fields from pod's `spec.securityContext`. Especially `seLinuxOptions` and future `selinuxChangePolicy`.
* `kube_pod_container_security_context`: pod's `spec.containers[*].securityContext` fields. We will need at least `privileged` and `seLinuxOptions`.
* `kube_pod_container_volume_mount`: report volumes used by containers. The existing `kube_pod_spec_volumes_persistentvolumeclaims_info` tracks volumes used in pods, we need *containers*.
* `kube_csidriver_info`: report CSIDriver object fields. At least `seLinuxMount` flag.
* *Maybe* `kube_persistentvolume_unique_volume_id`, because `kube_persistentvolume_info` misses some volume plugins (in-tree vSphere, Cinder, AzureFile) and it's clumsy to work with in general.
* *Maybe*  `kube_pod_container_security_context_info`, with defaulted SELinux fields from PodSecurityContex at and SecurityContext at container level, to save a lot of `label_replaces` in PromQL.

There is a proof of concept code in https://github.com/kubernetes/kube-state-metrics/pull/2513.

All these changes do not need any feature gate. They may be useful also when SELinuxMount feature is removed from k/k.

With these metrics, it should be possible to query Prometheus to list Pods that share the same volume, but need it with a different SELinux mount options.
This list would include Pods that run on a different nodes.
Those would not start if they run on the same node.
A Kubernetes distribution that uses SELinux could alert on them or block a cluster upgrade until the list is empty.
A cluster admin can use this list to check what would break during upgrade and update their Pods / StatefulSet / Deployments / DaemonSets with `SELinuxChangePolicy: Recursive` to opt out from the new behavior.

**Due to nature of kube-state-metrics and Prometheus, it may miss a Pod that was living only shortly between two scrapes.
We don't plan to fix this.**

#### PromQL

These promQL queries should be then possible:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: selinuxmount-test
spec:
  groups:
    - name: selinuxmount.rules
      rules:
      # Add unique_volume_id label to kube_persistentvolume_info that identifies a volume in the storage backend.
      # PVs A and B represent the same volume in the storage backend if and only if they have the same unique_volume_id.
      # The string is not very human friendly.
      - record: kube_persistentvolume_info:unique_volume_id
        expr: |
          label_join(kube_persistentvolume_info, "unique_volume_id", "#", "gce_persistent_disk_name", "ebs_volume_id", "azure_disk_name", "fc_wwids", "fc_lun", "fc_target_wwns", "iscsi_target_portal", "iscsi_iqn", "iscsi_lun", "nfs_server", "nfs_path", "csi_driver", "csi_volume_handle", "local_path", "host_path")

      # Add pod's SecurityContext to kube_pod_container_security_context
      - record: kube_pod_container_security_context:pod_selinux
        expr: |
          kube_pod_security_context * on (pod, namespace, uid) group_right(pod_selinux_user, pod_selinux_role, pod_selinux_type, pod_selinux_level) kube_pod_container_security_context

      # Compute correct SELinux label of a container in final_selinux_* labels and join it into final_selinux_label label that looks like "system_r:system_u:container_t:s0:c1,c92".
      - record: kube_pod_container_security_context:final_selinux_label
        expr: |
          # Rules:
          # - for privileged containers, the label is hardcoded to system_r:system_u:spc_t:s0
          # - if set in a container, use that
          # - if set in pod, use that
          # - if not set at all and the level is set, set the missing value as system_r:system_u:container_t:<level>
          # We use label_replace below and thus the conditions are in the opposite order - start with container_t, overwrite it with the value from pod, the value from container container, and spc_t, when their conditions are met
          label_replace(
          label_join(
          label_replace(
          label_replace(
          label_replace(
          label_replace(
          label_replace(
          label_replace(
          label_replace(
          label_replace(
          label_replace(
          label_replace(
          label_replace(
          label_replace(
          label_replace(
          label_replace(
          label_replace(
            # Compute level first, other replaces will depend on it.
            # Start with pod_selinux_level as final_selinux_level
            kube_pod_container_security_context:pod_selinux, "final_selinux_level", "$1", "pod_selinux_level", "(.*)"),
            # replace it by container_selinux_user if it is set
            "final_selinux_level", "$1", "container_selinux_level", "(.+)"),
            # replace it by s0 for privileged pods
            "final_selinux_level", "s0", "privileged", "true"),

            # user: start with system_u if level is set
            "final_selinux_user", "system_u", "final_selinux_level", "(.+)"),
            # replace it by pod_selinux_user, if set
            "final_selinux_user", "$1", "pod_selinux_user","(.+)"),
            # replace it by container_selinux_user if it is set
            "final_selinux_user", "$1", "container_selinux_user", "(.+)"),
            # replace it by system_u for privileged pods
            "final_selinux_user", "system_u", "privileged", "true"),

            # same for _role
            "final_selinux_role", "system_r", "final_selinux_level", "(.+)"),
            "final_selinux_role", "$1", "pod_selinux_role","(.+)"),
            "final_selinux_role", "$1", "container_selinux_role", "(.+)"),
            "final_selinux_role", "system_r", "privileged", "true"),

            # same for _type
            "final_selinux_type", "container_t", "final_selinux_level", "(.+)"),
            "final_selinux_type", "$1", "pod_selinux_type","(.+)"),
            "final_selinux_type", "$1", "container_selinux_type", "(.+)"),
            "final_selinux_type", "spc_t", "privileged", "true"),

            # This is label_join(): join all final_selinux_ labels into one string as final_selinux_label
            "final_selinux_label", ":", "final_selinux_user", "final_selinux_role", "final_selinux_type", "final_selinux_level"),
            # label_replace(): replace ":::" from the label_join above (= no label set in a pod) with "", just because it looks better
            "final_selinux_label", "", "final_selinux_label", "^:::$")

      # Add final_selinux_label to kube_pod_container_volume_mount.
      # Augmented by csidriver.SELinuxMount, if set.
      # This does a massive join from container -> pod -> pvc -> pv -> csidriver.
      - record: kube_pod_container_volume_mount:selinux_label
        expr: |
          label_replace(
            kube_pod_container_volume_mount
              # Join final_selinux_label
              * on(uid, pod, container) group_left (final_selinux_label) kube_pod_container_security_context:final_selinux_label
              # Join CSIDriver.Spec.SELinuxMount by joining pvc -> pv -> csidriver
              * on (uid, volume) group_left(persistentvolumeclaim) kube_pod_spec_volumes_persistentvolumeclaims_info
              * on (namespace, persistentvolumeclaim) group_left(volumename) kube_persistentvolumeclaim_info
              * on (volumename) group_left (csi_driver, csi_volume_handle, unique_volume_id) label_replace(kube_persistentvolume_info:unique_volume_id, "volumename", "$1", "persistentvolume", "(.+)")
              * on(csi_driver) group_left (selinux_mount) kube_csidriver_info,

            # label_replace: set final_selinux_label to "" when the CSI driver does not support SELinuxMount
            "final_selinux_label", "", "selinux_mount", "false")
```

* List volumes that are used with two or more SELinux labels:
  `sum by(unique_volume_id) (max by (unique_volume_id, final_selinux_label) (kube_pod_container_volume_mount:selinux_label)) > 1`
* For each volume, list PVCs + pods + containers that use it:
  `kube_pod_container_volume_mount:selinux_label{unique_volume_id="<volume ID from the previous query>"}`

TODO:
* Augment it by `SELinuxChangePolicy` field.
