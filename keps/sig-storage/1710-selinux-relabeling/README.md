# Skip SELinux relabeling of volumes

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [SELinux intro](#selinux-intro)
  - [SELinux context assignment](#selinux-context-assignment)
  - [Volumes](#volumes)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    - [<code>mount -o context</code>](#)
  - [New Kubernetes behavior](#new-kubernetes-behavior)
  - [Shared volumes](#shared-volumes)
  - [<code>CSIDriver.Spec.SELinuxMountSupported</code>](#-1)
  - [Examples](#examples)
  - [User Stories [optional]](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional-1)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
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
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (optional)](#infrastructure-needed-optional)
- [Implementation History](#implementation-history-1)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
  - [<code>FSGroupChangePolicy</code> approach](#-approach)
  - [Change container runtime](#change-container-runtime)
  - [Move SELinux label management to kubelet](#move-selinux-label-management-to-kubelet)
  - [Merge <code>FSGroupChangePolicy</code> and <code>SELinuxRelabelPolicy</code>](#merge--and-)
<!-- /toc -->

## Release Signoff Checklist

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This KEP tries to speed up the way that volumes (incl. persistent volumes) are made available to Pods on systems with SELinux in enforcing mode.
Current way includes recursive relabeling of all files on a volume before a container can be started. This is slow for large volumes.

## Motivation

### SELinux intro
On Linux machines with SELinux in enforcing mode, SELinux tries to prevent users that escaped from a container to access the host OS and also to access other containers running on the host.
It does so by running each container with unique *SELinux context* (such as `system_u:system_r:container_t:s0:c309,c383`) and labeling all content on all volumes with the corresponding label (`system_u:object_r:container_file_t:s0:c309,c383`).
Only process with the context `...:container_t:s0:c309,c383` can access files with label `container_file_t:s0:c309,c383`, even if the process runs as root.
Therefore rogue user cannot access potentially secret data of other containers, because volumes of each container have different label.

In further text, we're going to shorten both `system_u:system_r:container_t:s0:c309,c383` (context of a process) and `system_u:object_r:container_file_t:s0:c309,c383` (label of a file) to `s0:309:383`.

See [SELinux documentation](https://selinuxproject.org/page/NB_MLS) for more details.

### SELinux context assignment
In Kubernetes, the SELinux context of a pod is assigned in two ways:
1. Either it is set by user in PodSpec or Container: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/.
1. If not set in Pod/Container, the container runtime will allocate a new unique SELinux context and assign it to a pod (container) by itself.

### Volumes
Currently, Kubernetes *knows*(*) which volume plugins supports SELinux (i.e. supports extended attributes on a filesystem the plugin provides).
If SELinux is supported for a volume, it passes the volume to the container runtime with ":Z" option ("private unshared").
The container runtime then **recursively relabels** all files on the volume to either the label set in PodSpec/Container or the random value allocated by the container runtime itself.

**This relabeling needs to traverse through the whole volume, and it can be slow for volumes with a large amount of files.**

*) These in-tree volume plugins don't support SELinux: Azure File, CephFS, GlusterFS, HostPath, NFS, Portworx and Quobyte.
All other volume plugins support it.
This knowledge is hardcoded in in-tree volume plugins (e.g. [NFS](https://github.com/kubernetes/kubernetes/blob/0c5c3d8bb97d18a2a25977e92b3f7a49074c2ecb/pkg/volume/nfs/nfs.go#L235)).

For CSI, kubelet uses following heuristics:

1. Mount the volume (via `NodeStage` + `NodePublish` CSI calls).
2. Check mount options of the volume mount dir. If and only if it contains `seclabel` mount option, the volume supports SELinux.

### Goals

Optionally (chosen by user), do not recursively relabel content of the volumes.

### Non-Goals

Change container runtimes / CRI.

## Proposal

Offer option in `Pod.Spec.PodSecurityContext to` *mount* volumes with the right labels instead of recursive relabeling:

```go
type SELinuxRelabelPolicy string

const (
    OnVolumeMount SELinuxRelabelPolicy = "OnVolumeMount"
    AlwaysRelabel SELinuxRelabelPolicy = "Always"
)

type PodSecurityContext struct {
    // seLinuxRelabelPolicy ← new field
    // Defines behavior of changing SELinux labels of the volume before being exposed inside Pod.
    // Valid values are "OnVolumeMount" and "Always". If not specified, "Always" is used.
    // "Always" policy recursively changes SELinux labels on all files on all volumes used by the Pod.
    // "OnVolumeMount" tries to mount volumes used by the Pod with the right context and skip recursive ownership
    // change. Kubernetes may fall back to policy "Always" if a storage backed does not support this policy.
    // This field is ignored for Pod's volumes that do not support SELinux.
    // + optional
    SELinuxRelabelPolicy *SELinuxRelabelPolicy

    // For context:
    // fsGroupChangePolicy defines behavior of changing ownership and permission of the volume
    // before being exposed inside Pod. This field will only apply to
    // volume types which support fsGroup based ownership(and permissions).
    // It will have no effect on ephemeral volume types such as: secret, configmaps
    // and emptydir.
    // Valid values are "OnRootMismatch" and "Always". If not specified defaults to "Always".
    // +optional
    FSGroupChangePolicy *PodFSGroupChangePolicy `json:"fsGroupChangePolicy,omitempty" protobuf:"bytes,9,opt,name=fsGroupChangePolicy"`

    ...
}
```

See https://github.com/kubernetes/enhancements/blob/master/keps/sig-storage/20200120-skip-permission-change.md for similar API for ownership change for fsGroup.
This KEP should follow API provided for fsGroup closely, however, the implementation is different (`mount` here vs. recursive `chown` in the other KEP).

In order to allow `SELinuxRelabelPolicy: OnVolumeMount` for volumes provided by CSI drivers, kubelet must know if a CSI driver supports SELinux or not.

```go
// In storage.k8s.io/v1:


// CSIDriverSpec is the specification of a CSIDriver.
type CSIDriverSpec struct {
    // SELinuxMountSupported specifies if the CSI driver supports "-o context"
    // mount option.
    //
    // When "true", Kubernetes may call NodeStage / NodePublish with "-o context=xyz" mount
    // option for volumes of a pod with
    // podSecurityContext.seLinuxRelabelPolicy ="OnVolumeMount".
    //
    // When "false", Kubernetes won't pass any special SELinux mount options to the driver.
    // podSecurityContext.seLinuxRelabelPolicy "OnVolumeMount" is silently ignored.
    //
    // Default is "false".
    SELinuxMountSupporteded *bool;
    ...
}

// For context:
type CSIDriver struct {
    Spec CSIDriverSpec
}
```

### Implementation Details/Notes/Constraints [optional]

#### `mount -o context`
Linux kernel, with SELinux compiled in, allows `mount -o context=s0:c309,c383 <what> <where>` to mount a volume and pretend that all files on the volume have given SELinux label.
It works only for the first mount of the volume!
It does not work for bind-mounts or any subsequent mount of the same volume.

Note that volumes mounted with `-o context` don't have `seclabel` in their mount options.
In addition, calling `chcon` there will fail with `Operation not supported`.

### New Kubernetes behavior

* If kubelet *knows* SELinux context of a pod / container to run (i.e. Pod/Container contains at least `SELinuxOptions.Level`):
    * And pod's `SELinuxRelabelPolicy` is `OnVolumeMount`:
        * And if the in-tree volume plugin supports SELinux / `CSIDriver.Spec.SELinuxMountSupported` is explicitly `true`:
            * Kubelet tries to mount the volume for the Pod with given SELinux label using `mount -o context=XYZ`.
                * Kubelet makes sure the option is passed to the first mount in all in-tree volume plugins (incl. ephemeral volumes like Secrets).
                * Kubelet passes it as a mount option to all CSI calls for given volume.
            * After the volume is mounted, kubelet checks that the root of the volume has the expected SELinux label, i.e. that the volume was mounted correctly.
                * If the volume root has expected label, kubelet passes the volume to the container runtime without any ":z" or ":Z" options - no relabeling is necessary.
                * If the volume root has unexpected label, for example when CSI driver did not apply `-o context` correctly, or the volume was already mounted with a different context,
                  volume plugin reports an error and kubelet fails to start the pod.
                  It is CSI driver fault that it advertises SELinux support and then fails to apply it.

* Nothing changes when `CSIDriver.Spec.SELinuxMountSupported` is `false` or not set:
    * CSI volume plugin calls CSI without any special SELinux mount options and it autodetects, if the volume supports SELinux or not by presence of `seclabel` mount option.
      This is current kubelet behavior.

* Nothing changes if kubelet does not know the SELinux context of a pod (`SELinuxOptions.Level` is empty) or pod's `SELinuxRelabelPolicy` is `Always`.
    * Volume is mounted without any SELinux options and passed to the container runtime with or without ":Z", depending on if the volume plugin supports SELinux or not (by checking `seclabel` mount option).
      The container runtime allocates a new SELinux context and recursively relabels all files on the volume.
      This is current kubelet behavior.

Validation:

* Kubernetes checks that `SELinuxRelabelPolicy` field can be used in a pod only when at least `SELinuxOptions.Level` is set.

When a Pod specifies incomplete SELinux label (i.e. omits `SELinuxOptions.User`, `.Role` or `.Type`), kubelet fills the blanks from the system defaults provided by [ContainerLabels() from go-selinux bindings](https://github.com/opencontainers/selinux/blob/621ca21a5218df44259bf5d7a6ee70d720a661d5/go-selinux/selinux_linux.go#L770).

### Shared volumes

If a single PV that supports SELinux labels is being shared by multiple pods, each of them must have the same SELinux context.
Currently, a running pod with context `A` will lose access to all files on a volume if a pod with context `B` starts and uses the same volume, because the container runtime relabels the volume for pod `B`.
This behavior changes with this KEP: kubelet mounts the volume with `-o context=A` for  the first pod.
It tries to do the same for the second pod with `-o context=B`, however, the volume has already been mounted and `mount -o context=B` fails.
Pod `B` can't start on the same node until pod `A` dies and kubelet unmounts its volumes.

We don't think that this is a bug in the design.
Only one pod will have access to the volume, this KEP only changes the selection.

The only different behavior is when two pods with different SELinux context use the same volume, but different SubPath - they are working with `Always` policy, as the container runtime relabeled only the subpaths, with `OnVolumeMount` the whole volume must have the same context.

### `CSIDriver.Spec.SELinuxMountSupported`

The new field `CSIDriver.Spec.SELinuxMountSupported` is important so kubelet knows if mounts of volumes provided by the driver are independent on each other.
There are CSI drivers that actually use a single [NFS](https://github.com/kubernetes-incubator/external-storage/tree/master/nfs-client)
or [GlusterFS](https://github.com/kubernetes-incubator/external-storage/tree/master/gluster/glusterfs)
export and provide subdirectories of this export as individual PVs.
If kubelet mounts such PV (i.e. a subdirectory) with `-o context=A`, all subsequent mounts of the same NFS/Gluster export must have the same SELinux context, despite being different PVs from Kubernetes perspective.

Since kubelet does not know about such limitation of a CSI driver, `CSIDriver.Spec.SELinuxMountSupported=false` (or `nil`) is needed to turn off mounting with `-o context`.

### Examples

Following table captures interaction between actual filesystems on a volume  and newly introduced flags. Hypothetic iscsi and NFS CSI drivers are used as an example of a volume based on a block device and shared filesystem.

| Volume       | CSIDriver.SELinuxMountSupported | Pod.SELinuxRelabelPolicy | mount opts | docker run -v |    |
|--------------|---------------------------------|--------------------------|------------|---------------|----|
| iscsi + ext4 | *                               | Always                   | -          | :Z            | 1) |
|              |                                 |                          |            |               |    |
| iscsi + ext4 | false / nil                     | OnVolumeMount            | -          | :Z            | 2) |
| iscsi + ext4 | true                            | OnVolumeMount            | -o context | -             | 3) |
|              |                                 |                          |            |               |    |
| iscsi + ntfs | true                            | OnVolumeMount            | -o context | -             | 3) |
| iscsi + ntfs | false / nil                     | OnVolumeMount            | -          | -             | 4) |
| iscsi + ntfs | *                               | Always                   | -          | -             | 5) |
|              |                                 |                          |            |               |    |
| nfs          | true                            | OnVolumeMount            | -o context | -             | 6) |
| nfs          | false / nil                     | OnVolumeMount            | -          | -             | 7) |

1) Using `:Z`, because `seclabel` was autodetected in mount options (ext4 supports SELinux).
2) `OnVolumeMount` is ignored when `SELinuxMountSupported` is `false`.
   While iscsi + ext4 supports `mount -o context`, either cluster admin did not update the CSIDriver yet (upgrading from older cluster) or has another reason for this.
   Using `:Z`, because `seclabel` was autodetected in mount options.
3) CSI driver supports `-o context` and pod asks for it.
4) `OnVolumeMount` is ignored when `SELinuxMountSupported` is `false`.
   Using no `:Z`, because `seclabel` was not detected in mount options (ntfs does not support SELinux).
5) ntfs mount does not have `seclabel` option, so kubelet won’t pass `:Z` to CRI.

NFS behaves largely as iscsi+ntfs, however these two cases are interesting:

6) Here CSI driver vendor says that all volumes are independent and `mount -o context` is safe. For example, when all volumes are separate NFS shares.
7) CSI driver vendor explicitly declares that mount of a volume with context `A` may affect mounts of other volumes provided by this driver with different context. For example, when all the volumes are subdirectories of a single NFS share.

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

1. Kubelet does not see any `SELinuxRelabelPolicy` configured in the pod and thus mounts `myclaim` PVC as usual and if the underlying volume supports SELinux, it passes it to the container runtime with ":Z".
   Kubelet passes also implicit Secret volume with token with ":Z".
2. Container runtime allocates a new unique SELinux label to the pod and recursively relabels all volumes with ":Z" to this label. 



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

No change from current Kubernetes behavior:

1. Kubelet does not see any `SELinuxRelabelPolicy` configured in the pod and thus mounts `myclaim` PVC as usual and if the underlying volume supports SELinux, it passes it to the container runtime with ":Z".
   Kubelet passes also implicit Secret volume with token with ":Z".
2. Container runtime uses SELinux label "s0:c10,c0", as instructed by Kubernetes. It will recursively relabels all volumes with ":Z" to this label.

#### Story 3

User (or something else, e.g. an admission webhook) configures SELinux label for a pod.
User chooses `SELinuxRelabelPolicy: "OnVolumeMount"`, because they expect a potentially large volume to be used by the pod.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: testpod
spec:
  securityContext:
    seLinuxOptions:
      level: s0:c10,c0
    seLinuxRelabelPolicy: OnVolumeMount
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

In this case, kubelet tries to mount all pod's volumes with `-o context=s0:c10,c0` mount option`.
If it succeeds, it passes the volume to the container runtime without ":Z" and the container runtime does not relabel the volume.
See [New Kubernetes behavior](#new-kubernetes-behavior) for error cases.



### Implementation Details/Notes/Constraints [optional]

### Risks and Mitigations

## Design Details

### Test Plan

* Unit tests:
   * API validation (all permutations missing / present PodSecurityPolicy.SELinuxOptions & SELinuxRelabelPolicy & container.SecurityPolicy.SELinuxOptions)
   * Passing mount options from kubelet to volume plugins.
* E2e tests:
   * Check no recursive `chcon` is done on a volume when not needed /
   * Check recursive `chcon` is done on a volume when needed (with a matrix of SELinuxOptions / SELinuxRelabelPolicy).
* Prepare e2e job that runs with SELinux in Enforcing mode!

### Graduation Criteria

* Alpha:
 * Provided all tests defined above are passing and gated by the feature gate `SELinuxRelabelPolicy` and set to a default of `false`.
 * Documentation exists.
* Beta: with discussions in SIG-Storage regarding success of deployments. A metric will be added to report time taken to perform a volume ownership change. Feature gate `ConfigurableFSGroupPolicy` is `true`.
* GA: all known issues fixed.

### Upgrade / Downgrade Strategy

`SELinuxRelabelPolicy` becomes "invisible" or dropped in an downgraded cluster. Container runtime will get ":Z" on volumes and they will do slow recursive chown as they do today.

### Version Skew Strategy

## Production Readiness Review Questionnaire

### Feature enablement and rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: SELinuxRelabelPolicy
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
  
  No, default behavior is the same as before.

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  Also set `rollback-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g. if this is runtime
  feature, can it break the existing applications?).
  
  Yes, it can be disabled / rolled back. Corresponding API fields get cleared and Kubernetes uses previous SELinux label handling. 

* **What happens if we reenable the feature if it was previously rolled back?**

  Nothing special happens.
  
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
  
  Running workloads are not affected during rollout, because they don't use the new API fields. 

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and do that now.

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

### Monitoring requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metrics. Operations against Kubernetes API (e.g.
  checking if there are objects with field X set) may be last resort. Avoid
  logs or events for this purpose.

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
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

* **Are there any missing metrics that would be useful to have to improve
  observability if this feature?**
  Describe the metrics themselves and the reason they weren't added (e.g. cost,
  implementation difficulties, etc.).

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
  
  Pod gets one new field.

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.
  
  Each CSI volume setup (mount) may introduce a mount check (for `seclabel`),
  i.e. parsing whole /proc/mounts. It should be OK, since we already do mount
  check in the most volume plugins.

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

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

<!--
Major milestones in the life cycle of a KEP should be tracked in this section.
Major milestones might include
- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider and why did you rule them out?  These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (optional)

<!--
Use this section if you need things from the project/SIG.  Examples include a
new subproject, repos requested, github details.  Listing these here allows a
SIG to get the process for these resources started right away.
-->



## Implementation History

* 1.19: Alpha

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
