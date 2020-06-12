<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up.  KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please ensure to complete all
  fields in that template.  One of the fields asks for a link to the KEP.  You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "title", "authors", "owning-sig",
  "status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary", and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG that are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly.  The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved.  Any KEP
marked as a `provisional` is a working document and subject to change.  You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused.  If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement", for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example.  If there are
new details that belong in the KEP, edit the KEP.  Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable` or significant changes once
it is marked `implementable` must be approved by each of the KEP approvers.
If any of those approvers is no longer appropriate than changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross cutting KEPs).
-->
# KEP-1698: generic ephemeral inline volumes

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Persistent Memory as DRAM replacement for memcached](#persistent-memory-as-dram-replacement-for-memcached)
    - [Local LVM storage as scratch space](#local-lvm-storage-as-scratch-space)
    - [Read-only access to volumes with data](#read-only-access-to-volumes-with-data)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Example](#example)
- [Design Details](#design-details)
  - [PVC meta data](#pvc-meta-data)
  - [Preventing accidental collision with existing PVCs](#preventing-accidental-collision-with-existing-pvcs)
  - [Pod events](#pod-events)
  - [Feature gate](#feature-gate)
  - [Modifying volumes](#modifying-volumes)
  - [Late binding](#late-binding)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes a more generic mechanism for specifying and using ephemeral
volumes. In contrast to the ephemeral volume types that are built into Kubernetes
(empty dir, secrets, config map) and CSI ephemeral volumes, the volume can be
provided by any storage driver that supports dynamic provisioning. All of the
normal volume operations (snapshotting, resizing, snapshotting, the future
storage capacity tracking, etc.) are supported.

This is achieved by embedding all of the parameters for a PersistentVolumeClaim
inside a pod spec and automatically creating a PersistentVolumeClaim with those parameters
for the pod. Then provisioning and pod scheduling work as for a pod with
a manually created PersistentVolumeClaim.

## Motivation

Kubernetes supports several kinds of ephemeral volumes, but the functionality
of those is limited to what is implemented inside Kubernetes. For example, EmptyDir can
only provide a directory on the root disk or RAM. This functionality cannot
be extended by third-party storage vendors.

For light-weight, local volumes, CSI ephemeral volumes were introduced.
Those can be provided by CSI drivers which were specifically written
to support this Kubernetes feature. Normal, standard-compliant CSI
drivers are not supported.

The design for CSI ephemeral volumes is not a good fit for more traditional
storage systems because:
- The normal API for selecting volume parameters (like size and
  storage class) is not supported.
- Integration into storage capacity aware pod scheduling is
  challenging and would depend on extending the CSI ephemeral inline
  volume API.
- CSI drivers need to be adapted and have to take over some of the
  work normally done by Kubernetes, like provisioning volumes,
  attaching them to nodes and tracking of orphaned volumes.

### User Stories

#### Persistent Memory as DRAM replacement for memcached

Recent releases of memcached added [support for using Persistent
Memory](https://memcached.org/blog/persistent-memory/) (PMEM) instead
of normal DRAM. When deploying memcached through one of the app
controllers, `EphemeralVolumeSource` makes it possible to request a volume
of a certain size from a CSI driver like
[PMEM-CSI](https://github.com/intel/pmem-csi).

#### Local LVM storage as scratch space

Applications working with data sets that exceed the RAM size can
request local storage with performance characteristics or size that is
not met by the normal Kubernetes `EmptyDir` volumes. For example,
[TopoLVM](https://github.com/cybozu-go/topolvm) was written for that
purpose.

#### Read-only access to volumes with data

Provisioning a volume might result in a non-empty volume:
- [restore a snapshot](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#volume-snapshot-and-restore-volume-from-snapshot-support)
- [cloning a volume](https://kubernetes.io/docs/concepts/storage/volume-pvc-datasource)
- [generic data populators](https://github.com/kubernetes/enhancements/blob/master/keps/sig-storage/20200120-generic-data-populators.md)

For those, it might make sense to mount the volume read-only inside
the pod to prevent accidental modification of the data. For example,
the goal might be to just retrieve the data and/or copy it elsewhere.

### Goals

- Volumes can be specified inside the pod spec ("inline").
- Volumes are created for specific pods and deleted after the pod
  terminates ("ephemeral").
- A normal, unmodified storage driver can be selected via a storage class.
- The volume will be created using the normal storage provisioning
  mechanism, without having to modify the driver or its deployment.
- Storage capacity tracking can be enabled also for such volumes.
- Eventual API alignment with CSI ephemeral inline volumes, as well as
  current ephemeral in-tree volumes (EmptyDir, etc.), to ensure
  consistency across ephemeral volume APIs and minimize user
  confusion.

### Non-Goals

- This will not replace CSI ephemeral inline volumes because the
  goals for those (light-weight, local volumes) are not a good fit
  for the approach proposed here.
- These inline volumes will always be ephemeral. If making them persistent
  was allowed, some additional controller would be needed to manage them
  after pod termination, in which case it will probably be simpler to
  also create them separately.

## Proposal

A new volume source will be introduced:

```
type EphemeralVolumeSource struct {
    // Will be created as a stand-alone PVC to provision the volume.
    // Required, must not be nil.
    VolumeClaimTemplate *PersistentVolumeClaimTemplate
    ReadOnly             bool
}
```

This mimics a
[`PersistentVolumeClaimVolumeSource`](https://pkg.go.dev/k8s.io/api/core/v1?tab=doc#PersistentVolumeClaimVolumeSource),
except that it contains a `PersistentVolumeClaimTemplate` instead of
referencing a `PersistentVolumeClaim` by name. The content of this
`PersistentVolumeClaimTemplate` is identical to
`PersistentVolumeClaim` and just uses a different name to clarify the
intend, which is to serve as template for creating a volume claim
object. Embedding a full object is similar to the StatefulSet API and
allows storing labels and annotations. The name follows the example
set by [`PodTemplate`](https://pkg.go.dev/k8s.io/api/core/v1?tab=doc#PodTemplate).

`VolumeClaimTemplate` is defined as a pointer because in the future there
might be alternative ways to specify how the ephemeral
volume gets provisioned, in which case `nil` will become a valid value.

It gets embedded in the existing [`VolumeSource`
struct](https://pkg.go.dev/k8s.io/api/core/v1?tab=doc#VolumeSource)
struct as another alternative to the other ways of providing the
volume, like
[`CSIVolumeSource`](https://pkg.go.dev/k8s.io/api/core/v1?tab=doc#CSIVolumeSource)
and
[`PersistentVolumeClaimVolumeSource`](https://pkg.go.dev/k8s.io/api/core/v1?tab=doc#PersistentVolumeClaimVolumeSource).

```
type VolumeSource struct {
   ...
   CSI                   *CSIVolumeSource
   PersistentVolumeClaim *PersistentVolumeClaimVolumeSource
   ...
   Ephemeral             *EphemeralVolumeSource
   ...
}
```

A new controller in `kube-controller-manager` is responsible for
creating new PVCs for each such ephemeral inline volume. It does that:
- with a deterministic name that is a concatenation of pod name and
  the `Volume.Name` of the volume,
- in the namespace of the pod,
- with the pod as owner.

Kubernetes already prevents adding, removing or updating volumes in a
pod and the ownership relationship ensures that volumes get deleted
when the pod gets deleted, so the new controller only needs to take
care of creating missing PVCs.

When the [volume scheduling
library](https://github.com/kubernetes/kubernetes/tree/v1.18.0/pkg/controller/volume/scheduling)
inside the kube-scheduler or kubelet encounter such a volume inside a
pod, they determine the name of the PVC and proceed as they currently
do for a `PersistentVolumeClaimVolumeSource`.

### Risks and Mitigations

Enabling this feature allows users to create PVCs indirectly if they can
create pods, even if they do not have permission to create them
directly. Cluster administrators must be made aware of this. If this
does not fit their security model, they can disable the feature
through the feature gate that will be added for the feature.

In addition, with a new
[`FSType`](https://github.com/kubernetes/kubernetes/blob/1fb0dd4ec5134014e466509163152112626d52c3/pkg/apis/policy/types.go#L278-L309)
it will be possible to limit the usage of this volume source via the
[PodSecurityPolicy
(PSP)](https://kubernetes.io/docs/concepts/policy/pod-security-policy/#volumes-and-file-systems).

The normal namespace quota for PVCs in a namespace still applies, so
even if users are allowed to use this new mechanism, they cannot use
it to circumvent other policies.

## Example

Here is a full example for a higher-level object that uses a generic ephemeral
inline volume:

```
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fluentd-elasticsearch
  namespace: kube-system
spec:
  selector:
    matchLabels:
      name: fluentd-elasticsearch
  template:
    metadata:
      labels:
        name: fluentd-elasticsearch
    spec:
      containers:
      - name: fluentd-elasticsearch
        image: quay.io/fluentd_elasticsearch/fluentd:v2.5.2
        volumeMounts:
        - name: varlog
          mountPath: /var/log
        - name: scratch
          mountPath: /scratch
      volumes:
      - name: varlog
        hostPath:
          path: /var/log
      - name: scratch
        ephemeral:
          metadata:
            labels:
              type: fluentd-elasticsearch-volume
          spec:
            accessModes: [ "ReadWriteOnce" ]
            storageClassName: "scratch-storage-class"
            resources:
              requests:
                storage: 1Gi
```

The DaemonSet controller will create pods with names like
`fluentd-elasticsearch-b96sd` and the new controller will then add a
PVC called `fluentd-elasticsearch-b96sd-scratch` for that pod.

## Design Details

### PVC meta data

The namespace is the same as for the pod and the name is the
concatenation of pod name and the `Volume.Name`. This is guaranteed to
be unique for the pod and each volume in that pod. Care must be taken
by the user to not exceed the length limit for object names, otherwise
volumes cannot be created.

Other meta data is copied from the `VolumeClaimTemplate.ObjectMeta`.

### Preventing accidental collision with existing PVCs

The new controller will only create missing PVCs. It neither needs to
delete (handled by garbage collection) nor update (`PodSpec.Volumes`
is immutable) existing PVCs. Therefore there is no risk that the
new controller will accidentally modify unrelated PVCs.

The volume scheduling library and kubelet must verify that the PVC has
an owner reference to the pod before using it. Otherwise they must ignored it. This
ensures that a volume is not used accidentally for a pod in case of a
name conflict.

### Pod events

Errors that occur while creating the PVC will be reported as events
for the pod and thus be visible to the user. This includes quota
violations and name collisions.

### Feature gate

The `GenericEphemeralVolumes` feature gate controls whether:
- the new controller is active in the `kube-controller-manager`,
- new pods can be created with a `EphemeralVolumeSource`,
- anything that specifically acts upon an `EphemeralVolumeSource` (scheduler,
  kubelet, etc.) instead of merely copying it (statefulset controller)
  accepts the new volume source.

Existing pods with such a volume will not be started when the feature
gate is off.

### Modifying volumes

Once the PVC for an ephemeral volume has been created, it can be updated
directly like other PVCs. The new controller will not interfere with that
because it never updates PVCs. This can be used to control features
like volume resizing.

### Late binding

Ideally, provisioning should use late binding (aka
`WaitForFirstConsumer`). The initial implementation assumes that the
user is taking care of that by selecting that provisioning mode in the
storage class for the PVC.

Later, `kube-scheduler` and `external-provisioner` can be changed to
automatically enable late binding for PVCs which are owned by a pod.

### Test Plan

- Unit tests will be added for the API change.
- Unit tests will cover the functionality of the controller, similar
  to
  https://github.com/kubernetes/kubernetes/blob/v1.18.2/pkg/controller/volume/persistentvolume/pv_controller_test.go.
- Unit tests need to cover the positive case (feature enabled) as
  well as negative case (feature disabled or feature used incorrectly).
- A new [storage test
  suite](https://github.com/kubernetes/kubernetes/blob/2b2cf8df303affd916bbeda8c2184b023f6ee53c/test/e2e/storage/testsuites/base.go#L84-L94)
  will be added which tests ephemeral volume creation, usage and deletion
  in combination with all drivers that support dynamic volume
  provisioning.

### Graduation Criteria

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Errors emitted as pod events
- Decide whether `CSIVolumeSource` (in beta at the moment) should be
  merged with `EphemeralVolumeSource`
- Decide whether in-tree ephemeral volume sources, like EmptyDir (GA
  already), should also be added EphemeralVolumeSource for sake of API
  consistency
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- 3 examples of real world usage
- Downgrade tests and scalability tests
- Allowing time for feedback

### Upgrade / Downgrade Strategy

When downgrading to a cluster which does not support generic ephemeral inline
volumes (either by disabling the feature flag or an actual version
downgrade), pods using such volumes will no longer be started.

### Version Skew Strategy

As with downgrades, having some of the relevant components at an older
version will prevent pods from starting.

## Production Readiness Review Questionnaire

### Feature enablement and rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate
    - Feature gate name: GenericEphemeralVolumes
    - Components depending on the feature gate:
      - kube-apiserver
      - kube-controller-manager
      - kubelet

* **Does enabling the feature change any default behavior?**
  If users are allowed to create pods but not PVCs, then generic ephemeral inline volumes
  grants them permission to create PVCs indirectly. Cluster admins must take
  that into account in their permission model.

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  Yes, by disabling the feature gates. Existing pods with generic ephemeral inline
  volumes that haven't started yet will not be able to start up anymore, because
  kubelet does not know what to do with the volume. It also will not know how
  to unmount such volumes which would cause pods to get stuck, so nodes
  should be drained before removing the feature gate in kubelet.

* **What happens if we reenable the feature if it was previously rolled back?**
  Pods that got stuck will work again.

* **Are there any tests for feature enablement/disablement?**
  Yes, unit tests for the apiserver and kubelet.

### Rollout, Upgrade and Rollback Planning

Will be added before the transition to beta.

* **How can a rollout fail? Can it impact already running workloads?**

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**

### Monitoring requirements

Will be added before the transition to beta.

* **How can an operator determine if the feature is in use by workloads?**

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

* **Are there any missing metrics that would be useful to have to improve
  observability if this feature?**

### Dependencies

Will be added before the transition to beta.

* **Does this feature depend on any specific services running in the cluster?**

### Scalability

Will be added before the transition to beta.

* **Will enabling / using this feature result in any new API calls?**

* **Will enabling / using this feature result in introducing new API types?**

* **Will enabling / using this feature result in any new calls to cloud
  provider?**

* **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**

### Troubleshooting

Will be added before the transition to beta.

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- Kubernetes 1.19: alpha (tentative)

## Alternatives

### Embedded PVC with status

The alternative to creating the PVC is to modify components that
currently interact with a PVC such that they can work with stand-alone
PVC objects (like they do now) and with the embedded PVCs inside
pods. The downside is that this then no longer works with unmodified
CSI deployments because extensions in the CSI external-provisioner
will be needed.

Some of the current usages of PVC will become a bit unusual (status
update inside pod spec) or tricky (references from PV to PVC).

The advantage is that no automatically created PVCs are
needed. However, other controllers also create user-visible objects
(statefulset -> pod and PVC, deployment -> replicaset -> pod), so this
concept is familiar to users.

### Extending CSI ephemeral volumes

In the current CSI ephemeral volume design, the CSI driver only gets
involved after a pod has already been scheduled onto a node. For feature
parity with normal volumes, the CSI driver would have to reimplement
the work done by external-provisioner and external-attacher. This makes
CSI drivers specific to Kubernetes, which is against the philosophy of
CSI.

The Kubernetes API would have to be extended to make the scheduler aware of volume
size.

Kubelet would need to be extended to evict a pod when volume creation fails
because storage is exhausted. Currently it just retries until someone
or something else deletes the pod.

All of that would lead to additional complexity both in Kubernetes and in
CSI drivers. Handling volume provisioning and attaching through the
existing mechanisms is easier and compatible with future enhancements
of those.
