# KEP-1698: generic ephemeral inline volumes

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [User Stories](#user-stories)
    - [Persistent Memory as DRAM replacement for memcached](#persistent-memory-as-dram-replacement-for-memcached)
    - [Local LVM storage as scratch space](#local-lvm-storage-as-scratch-space)
    - [Read-only access to volumes with data](#read-only-access-to-volumes-with-data)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
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
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Embedded PVC with status](#embedded-pvc-with-status)
  - [Extending CSI ephemeral volumes](#extending-csi-ephemeral-volumes)
  - [Extending app controllers](#extending-app-controllers)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [X] (R) Production readiness review completed
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

This KEP proposes a more generic mechanism for specifying and using
ephemeral volumes. In contrast to the ephemeral volume types that are
built into Kubernetes (`EmptyDir`, `Secrets`, `ConfigMap`) and [CSI
ephemeral
volumes](https://kubernetes.io/docs/concepts/storage/volumes/#csi-ephemeral-volumes),
the volume can be provided by any storage driver that supports dynamic
provisioning. All of the normal volume operations (snapshotting,
resizing, snapshotting, the future storage capacity tracking, etc.)
are supported.

This is achieved by embedding all of the parameters for a PersistentVolumeClaim
inside a pod spec and automatically creating a PersistentVolumeClaim with those parameters
for the pod. Then provisioning and pod scheduling work as for a pod with
a manually created PersistentVolumeClaim.

## Motivation

Kubernetes supports several kinds of ephemeral volumes, but the functionality
of those is limited to what is implemented inside Kubernetes.

CSI ephemeral volumes made it possible to extend Kubernetes with CSI
drivers that provide light-weight, local volumes which [*inject
arbitrary states, such as configuration, secrets, identity, variables
or similar
information*](https://github.com/kubernetes/enhancements/blob/master/keps/sig-storage/20190122-csi-inline-volumes.md#motivation).
CSI drivers must be modified to support this Kubernetes feature,
i.e. normal, standard-compliant CSI drivers will not work.

This KEP does the same for volumes that are more like `EmptyDir`, for
example because they consume considerable resources, either locally on
a node or remotely. But the mechanism for creating such volumes is not
limited to just empty directories: all standard-compliant CSI drivers
and existing mechanisms for populating volumes with data (like
restoring from snapshots) will be supported, so this enables a variety
of use cases.

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

Generic ephemeral volumes provide a simple API for starting pods with
such volumes. For them, mounting the volume read-only inside the pod
may make sense to prevent accidental modification of the data. For
example, the goal might be to just retrieve the data and/or copy it
elsewhere.

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

In addition, with a new `ephemeral` value for
[`FSType`](https://github.com/kubernetes/kubernetes/blob/1fb0dd4ec5134014e466509163152112626d52c3/pkg/apis/policy/types.go#L278-L309)
it will be possible to limit the usage of this volume source via the
[PodSecurityPolicy
(PSP)](https://kubernetes.io/docs/concepts/policy/pod-security-policy/#volumes-and-file-systems).
If a PSP exists, `FSType` either has to include `all` or `ephemeral`
for this feature to be allowed. If no PSP exists, the feature is
allowed.

Adding that new value is an API change for PSP because it changes
validation. When the feature is disabled, validation must tolerate
this new value in updates of existing PSP objects that already contain
the value, but must not allow it when creating a new PSP or updating a
PSP that does not already contain the value. When the feature is
enabled, validation must allow this value on any create or update.

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
  merged with `EphemeralVolumeSource`: no, instead the goal is
  to [rename `CSIVolumeSource`](https://github.com/kubernetes/enhancements/issues/596#issuecomment-726185967)
- Decide whether in-tree ephemeral volume sources, like EmptyDir (GA
  already), should also be added EphemeralVolumeSource for sake of API
  consistency: [no](https://docs.google.com/document/d/1yAe3SPPosgC_QgmnY7oJTmZYWrqLrii1oA4de67DEcw/edit),
  this just causes API churn without tangible benefits
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

  Yes, unit tests for the apiserver, kube-controller-manager and kubelet cover scenarios
  where the feature is disabled or enabled. Tests for transitions
  between these states will be added before beta.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**

A rollout could fail because the implementation turns out to be
faulty. Such bugs may cause unexpected shutdowns of kube-scheduler,
kube-apiserver, kube-controller-manager and kubelet. For the API
server, broken support for the new volume type may also show up as 5xx
error codes for any object that embeds a `VolumeSource` (Pod,
StatefulSet, DaemonSet, etc.).

Already running workloads should not be affected unless they depend on
these components at runtime and bugs cause unexpected shutdowns.

* **What specific metrics should inform a rollback?**

One indicator are unexpected restarts of the cluster control plane
components. Another are an increase in the number of pods that fail to
start. In both cases further analysis of logs and pod events is needed
to determine whether errors are related to this feature.

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**

Not yet, but will be done manually before transition to beta.

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**

No.

### Monitoring requirements

* **How can an operator determine if the feature is in use by workloads?**

There will be pods which have a non-nil
`VolumeSource.Ephemeral.VolumeClaimTemplate`.


* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**

The service here is the Kubernetes control plane. Overall health and
performance can be observed by measuring the the pod creation rate for
pods using generic ephemeral inline volumes. Such [a
SLI](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/pod_startup_latency.md)
is defined for pods without volumes and work in progress for pods with
volumes.

For kube-controller-manager, a metric that exposes the usual work
queue metrics data (like queue length) will be made available with
"ephemeral_volume" as name. Here is one example after processing a
single pod with a generic ephemeral volume:

```
workqueue_adds_total{name="ephemeral_volume"} 1
workqueue_depth{name="ephemeral_volume"} 0
workqueue_longest_running_processor_seconds{name="ephemeral_volume"} 0
workqueue_queue_duration_seconds_bucket{name="ephemeral_volume",le="1e-08"} 0
...
workqueue_queue_duration_seconds_bucket{name="ephemeral_volume",le="9.999999999999999e-05"} 1
workqueue_queue_duration_seconds_bucket{name="ephemeral_volume",le="0.001"} 1
...
workqueue_queue_duration_seconds_bucket{name="ephemeral_volume",le="+Inf"} 1
workqueue_queue_duration_seconds_sum{name="ephemeral_volume"} 4.8201e-05
workqueue_queue_duration_seconds_count{name="ephemeral_volume"} 1
workqueue_retries_total{name="ephemeral_volume"} 0
workqueue_unfinished_work_seconds{name="ephemeral_volume"} 0
workqueue_work_duration_seconds_bucket{name="ephemeral_volume",le="1e-08"} 0
...
workqueue_work_duration_seconds_bucket{name="ephemeral_volume",le="0.1"} 1
...
workqueue_work_duration_seconds_bucket{name="ephemeral_volume",le="+Inf"} 1
workqueue_work_duration_seconds_sum{name="ephemeral_volume"} 0.035308659
workqueue_work_duration_seconds_count{name="ephemeral_volume"} 1
```

Furthermore, counters of PVC creation attempts and failed attempts
will be added. There should be no failures. If there are any, analyzing
the logs of kube-controller manager will provide further insights into
the reason why they occurred.

```
ephemeral_volume_controller_create_total 1
ephemeral_volume_controller_create_failures_total 0
```

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

The goal is to achieve the same pod creation rate for pods using
generic ephemeral inline volumes as for pods that use PVCs which get
created separately. To make this comparable, the storage class should
use late binding.

This will need further discussion before going to GA.

* **Are there any missing metrics that would be useful to have to improve
  observability of this feature?**

No.

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**

A dynamic provisioner from some kind of storage system is needed:

 * Volume provisioner
   * Usage description:
     * Impact of its outage on the feature: pods that use generic inline volumes
       provided by the storage system will not be able to start
     * Impact of its degraded performance or high-error rates on the
       feature: slower pod startup

### Scalability

* **Will enabling / using this feature result in any new API calls?**

Enabling will not change anything.

Using the feature in a pod will lead to one PVC creation per inline
volume, followed by garbage collection of those PVCs when the pod
terminates.

* **Will enabling / using this feature result in introducing new API types?**

No.

* **Will enabling / using this feature result in any new calls to cloud
  provider?**

Enabling the feature doesn't. Using it will cause new calls to cloud
providers, but the amount is exactly the same as without this feature:
for each per-pod volume, a PVC has to be created (either manually or
using this feature) and a volume needs to be provisioned in a storage
backend. When a pod terminates, that volume needs to be deleted again.

* **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**

Enabling it will not change existing objects. Using it in a pod spec
will increase the size by one `PersistentVolumeClaimTemplate` per
inline volume and cause one PVC to be created for each inline volume.

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**

There is a SLI for [scheduling of pods without
volumes](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/pod_startup_latency.md)
with a corresponding SLO. Those are not expected to be affected.

A SLI for scheduling of pods with volumes is work in progress. The SLO
for it will depend on the specific storage driver.

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**

Potentially in kube-scheduler and kube-controller-manager, but mostly only if
the feature is actually used. Merely enabling it will cause the new controller
in kube-controller-manager to check new pods for the new volume type, which
should be fast. In kube-scheduler the feature adds an additional case to
switch statements that check for persistent volume sources.

### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?**

Pods will not start and volumes for them will not get provisioned.

* **What are other known failure modes?**

As [explained
above](#preventing-accidental-collision-with-existing-pvcs), the PVC
that needs to be created for a pod may conflict with an already
existing PVC that was created independently of the pod. In such a
case, the pod will not be able to start until that independent PVC is
deleted. This scenario will be exposed as events for the pod by
kube-controller-manager.

If the storage system fails to provision volumes, then this will be
exposed as events for the PVC and (depending on the storage system)
may also show up in metrics data.

* **What steps should be taken if SLOs are not being met to determine the problem?**

SLOs only exist for pods which don't use the new feature. If those are
somehow affected, then error messages in the kube-scheduler and kube-controller-manager
output may provide additional information.

## Implementation History

- Kubernetes 1.19: alpha

## Drawbacks

Allowing users to create PVCs indirectly through pod creation is a new
capability which must be considered in a cluster's security policy. As
explained in [risks and mitigations above](#risks-and-mitigations),
the feature has to be configurable on a per-user basis.

Making a pod the owner of PVC objects introduces a new dependency
which may cause problems in code that expects that pods can always be
deleted immediately. One example was the PVC protection controller
which waited for pod deletion before allowing the PVC to be deleted:
it had to be extended to allow PVC deletion also when the pod had
terminated, but not deleted yet.

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

For storage capacity aware pod scheduling, the Kubernetes API would
have to be extended to make the scheduler aware of the ephemeral
volume's size. Kubelet would need to be extended to evict a pod when
volume creation fails because storage is exhausted. Currently it just
retries until someone or something else deletes the pod.

All of that would lead to additional complexity both in Kubernetes and in
CSI drivers. Handling volume provisioning and attaching through the
existing mechanisms is easier and compatible with future enhancements
of those.

### Extending app controllers

Instead of extending the pod spec, the spec of app controllers could
be extended. Then the app controllers could create per-pod PVCs and
also delete them when no longer needed. This will require making more
changes to the API and the Kubernetes code than the proposed changed
for the pod spec. Furthermore, those changes also would have to be
repeated in third-party app controllers which create pods directly.

It also does not avoid the issue with existing cluster security
policies.
