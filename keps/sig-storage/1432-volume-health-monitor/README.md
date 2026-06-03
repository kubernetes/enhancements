# KEP-1432: Volume Health Monitor

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Volume deleted out-of-band](#story-1-volume-deleted-out-of-band)
    - [Story 2: Backend unreachable from a subset of nodes](#story-2-backend-unreachable-from-a-subset-of-nodes)
    - [Story 3: Local-storage volume with degraded LVM backend](#story-3-local-storage-volume-with-degraded-lvm-backend)
    - [Story 4: Cross-driver dashboards](#story-4-cross-driver-dashboards)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Where reports live, and why](#where-reports-live-and-why)
  - [The CSI surface](#the-csi-surface)
  - [The Kubernetes surface](#the-kubernetes-surface)
  - [Reconciliation contract](#reconciliation-contract)
  - [Authorization](#authorization)
  - [Feature gate](#feature-gate)
  - [Test Plan](#test-plan)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Use only metrics and events for health reporting](#use-only-metrics-and-events-for-health-reporting)
  - [Embed VolumeCondition in existing RPCs (the original alpha)](#embed-volumecondition-in-existing-rpcs-the-original-alpha)
  - [A standalone per-PVC VolumeHealth CRD](#a-standalone-per-pvc-volumehealth-crd)
  - [A per-PVC map[node]Health directly on pvc.status](#a-per-pvc-mapnodehealth-directly-on-pvcstatus)
  - [PV taints with NoEffect](#pv-taints-with-noeffect)
  - [DRA-style device taints for storage (KEP-5055)](#dra-style-device-taints-for-storage-kep-5055)
  - [Push-based health ingest](#push-based-health-ingest)
  - [A richer error enum](#a-richer-error-enum)
- [Future enhancement](#future-enhancement)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

A `PersistentVolumeClaim` can be bound, mounted, and serving traffic
while the underlying storage is silently failing. The backend volume
might have been deleted out-of-band by an admin who didn't realize it
was still in use. A multipath link can drop and shave throughput in
half. The filesystem can corrupt, or a node can lose its data-plane
network to the storage backend without losing its control-plane
network. In all of these cases, the storage system knows or could
detect that something is wrong; Kubernetes today has no first-class
way to see it.

This KEP defines a uniform mechanism for CSI drivers to report volume
and backend health, and surfaces those reports on three durable
Kubernetes status fields where automation can act on them. Drivers
report through four new RPCs: `ControllerListVolumeHealth` and
`ControllerGetVolumeHealth` on the controller plugin, and
`NodeGetVolumeHealth` and `NodeGetStorageHealth` on the node plugin.
The `csi-external-health-monitor-controller` sidecar consumes the
controller-side RPCs and writes to
`PersistentVolumeClaim.Status.HealthStatus`. The kubelet consumes the
node-side RPCs and writes to `Pod.Status.VolumeHealth` (per pod, per
volume) and `CSINode.Status.StorageHealth` (per driver, per node). A
single feature gate, `CSIVolumeHealth`, gates the kubelet and the
apiserver.

This is not the first attempt. An alpha for KEP-1432 shipped in
Kubernetes v1.21 that overloaded `ListVolumes`, `ControllerGetVolume`,
and `NodeGetVolumeStats` with an embedded `VolumeCondition` field and
surfaced findings only as Kubernetes events. The original API never
graduated, and the CSI spec is now removing it
([container-storage-interface/spec#604][spec-pr]). This KEP
documents the replacement and resets graduation to alpha.

[spec-pr]: https://github.com/container-storage-interface/spec/pull/604

## Motivation

Storage backends fail in ways that Kubernetes cannot see today.
A disk can develop bad sectors or a thin pool can run out of space. An LVM volume group can lose a
physical volume. A network partition can sever a node's data
path to the storage backend while the control plane keeps
working. In every case the CSI driver on the node or the
storage controller knows something is wrong, but Kubernetes has
no API through which drivers can report that knowledge and no
status field where operators or automation can read it.

Without a health signal, the only indication of a storage
problem is a pod that hangs on I/O, a mount that fails
repeatedly, or an alert from outside the cluster. Operators
must cross-reference storage-vendor dashboards with Kubernetes
objects by hand, and remediation controllers have nothing
machine-readable to act on. The gap widens with scale: a
cluster running thousands of PVCs across hundreds of nodes
cannot rely on manual correlation.

An alpha for KEP-1432 shipped in Kubernetes v1.21 that
attempted to fill this gap by embedding a `VolumeCondition`
in existing `ListVolumes`, `ControllerGetVolume`, and
`NodeGetVolumeStats` RPCs and surfacing findings as Kubernetes
events. That approach had three structural limitations that
prevented graduation: health was coupled to stat and list RPCs
that serve different purposes on different timelines;
`NodeGetVolumeStats` can only be called for staged or published
volumes, which excludes the most interesting unhealthy cases
like corrupt filesystems or failed mounts; and events are
ephemeral, unstructured, and cannot drive a remediation
controller.

This KEP replaces the original alpha with dedicated health
RPCs separated from stats, a node-side RPC that can report on
volumes that were never successfully mounted, and durable
status fields on PVC, Pod, and CSINode that automation can
consume.

### Goals

- Provide a CSI-spec-level mechanism for drivers to report
  per-volume health from the controller, per-volume health from the
  node (including for volumes that have not been successfully
  published), and overall storage-backend health from a node.
- Surface those reports on stable Kubernetes API status fields:
  controller-side volume health on
  `PersistentVolumeClaim.Status.HealthStatus`, node-side per-volume
  health on `Pod.Status.VolumeHealth`, and node-side backend health
  on `CSINode.Status.StorageHealth`.
- Define a small, machine-parseable error vocabulary
  (`Inaccessible`, `DataLoss`, `Degraded`, `StorageUnreachable`,
  `StorageDegraded`) that admits driver-specific elaboration via
  `reason` and `message`. The vocabulary is intentionally
  extensible: future CSI spec revisions MAY add values, drivers
  that adopt a newer spec MAY report them alongside existing
  ones, and COs MUST tolerate values they do not recognize. A
  CO that does not recognize a value SHOULD surface it for
  observability and MUST NOT misclassify it as healthy.
- Make health reporting opt-in per driver (via CSI capabilities) and
  per cluster (via the `CSIVolumeHealth` feature gate). A driver
  that implements no health capability is not probed and incurs no
  cost.
- Keep ownership consistent with the existing Node Authorizer model:
  nodes never write PVC status.

### Non-Goals

This KEP defines reporting, not reaction. Kubernetes will not start
gating scheduling, admission, or volume reconciliation on health
values. Future KEPs may layer behavior on top (a CSIDriver opt-in to
prevent pod scheduling on `Inaccessible` volumes is a natural next
step), but they are out of scope here.

The error vocabulary is also deliberately small. It does not cover
storage-backend specifics like multipath loss, capacity exhaustion,
or disk-level performance degradation. Those signals belong in the
`reason` field, where drivers can name them precisely without forcing
every CO to learn a new enum value.

In-tree volume plugins are out of scope; this design applies only to
CSI drivers.

## Proposal

The CSI spec gains four RPCs, the Kubernetes API gains three optional
status fields, and the kubelet and apiserver gain one feature gate.
That is the entire user-visible surface.

The controller plugin learns to answer either "give me all volumes
that currently have adverse conditions" (paginated, via
`ControllerListVolumeHealth`) or "what's the condition of *this*
volume" (via `ControllerGetVolumeHealth`). A driver can implement
either or both. The `csi-external-health-monitor-controller` sidecar,
which already runs alongside many CSI controllers, polls whichever is
available and writes the result to
`pvc.status.healthStatus`. Because the sidecar lives in the control
plane, the cluster's existing trust boundary handles the question of
who can write what to a PVC.

The node plugin learns to answer two questions: "what's the condition
of this volume on this node" (via `NodeGetVolumeHealth`) and "what's
the condition of the storage backend from this node" (via
`NodeGetStorageHealth`). The kubelet calls them and writes the
results to `pod.status.volumeHealth` for the pods running on that
node and to `csinode.status.storageHealth` for the driver registered
on that node. Both targets are objects the kubelet can already write
to under its existing Node Authorizer permissions, so no new
authorization surface is needed for the node-side path.

The two reports are independent. A volume can be reported as
`Inaccessible` from a node, because that node lost its data-plane
route to the backend, while the controller plugin reports the same
volume as healthy. A volume can also be reported as `Degraded` from
the controller plane while a node sees it just fine, because the
backend's monitoring caught a slow trend that the data path hasn't
yet expressed. Forcing these two perspectives into one truth would
hide useful information; this design surfaces them separately and
lets consumers decide what to do.

### User Stories

#### Story 1: Volume deleted out-of-band

A storage admin accidentally deletes a backend volume that is bound
to a PVC. The controller plugin learns about it on its next
reconciliation against the backend and starts reporting
`{status: Inaccessible, reason: VolumeNotFound}` from
`ControllerListVolumeHealth`. The external monitor sidecar writes
that onto `pvc.status.healthStatus`. The application owner sees the
condition on `kubectl describe pvc` and recreates the PVC from a
snapshot. None of the nodes hosting the workload need to be involved.
In fact, if the workload still has a mount cached in the kernel page
cache, the nodes might not notice for some time.

#### Story 2: Backend unreachable from a subset of nodes

A network change makes the storage backend's data path unreachable
from one rack of nodes while the control plane keeps working fine.
CSI controller calls all succeed and report the volume healthy. On
the affected nodes the kubelet calls `NodeGetStorageHealth`, gets
back `{status: StorageUnreachable, reason: NetworkPartition}`, and
writes that onto `csinode.status.storageHealth` for the driver. A
future scheduler opt-in (out of scope) can use that to avoid placing
new pods needing the driver onto those nodes. Today, an operator can
see the condition with a single `kubectl get csinode` and route
around it manually.

#### Story 3: Local-storage volume with degraded LVM backend

A node-local CSI driver like TopoLVM manages LVM logical volumes
on the node's physical disks. A physical disk backing a volume
group starts returning I/O errors. The kubelet calls
`NodeGetVolumeHealth`; the driver reads LVM attributes and
returns `{status: DEGRADED, reason: PartialActivation}`. The
kubelet writes this onto `pod.status.volumeHealth` for pods
using the affected volume, and a database operator initiates a
replica rebuild before data loss occurs.

If a thin pool is exhausted, the volume cannot be staged at all.
Under the old alpha, `NodeGetVolumeStats` could not be called
for unstaged volumes, so the failure was invisible.
`NodeGetVolumeHealth` lets the driver report
`{status: INACCESSIBLE, reason: ThinPoolOutOfDataSpace}` even
for volumes that never reached a mounted state.

At the backend level, `NodeGetStorageHealth` lets the driver
report volume-group-wide conditions like
`{status: STORAGE_DEGRADED, reason: PhysicalVolumeMissing}` on
`csinode.status.storageHealth`.

#### Story 4: Cross-driver dashboards

Clusters that run more than one CSI driver (block, file, object)
get a uniform health schema across all of them. A platform team's
Prometheus dashboard can query
`csi_volume_health_status{status="Inaccessible"}` and see unhealthy
volumes regardless of which driver they live on, because every
driver writes through the same status fields with the same enum
values.

### Risks and Mitigations

The most-debated risk is write churn. Every node updates pods it
runs whenever a volume's reported health changes, every driver
reports on every volume it manages, and at scale this could amplify
into a substantial PATCH rate against the apiserver. The design
addresses this in two ways: both writers compute the desired
`Conditions` list, compare it element-by-element to the on-disk
value, and PATCH only on difference; in steady state, with no
unhealthy volumes, the sustained PATCH rate is zero. Probe intervals
are tunable on both writers, and operators of large clusters can
relax them without changing the contract this KEP defines.

A second risk is compromised or malicious nodes lying about a
volume's health to influence operator behavior. This was the
original objection to per-PVC node-keyed health, and it's why the
original alpha used events. The present design routes node-side
reports through `Pod.Status` and `CSINode.Status`, both of which the
kubelet can already write to under existing Node Authorizer rules,
scoped to its own pods and its own node, and never grants any node
access to PVC status. A compromised node can lie about its own pods
and its own node, which it could already do; it cannot taint a PVC
seen by the rest of the cluster.

A third risk is driver misclassification. A driver that reports
`DataLoss` for a transient `Degraded` condition could trigger
aggressive remediation in downstream automation. The mitigation is
in the spec text: each enum value has explicit MUST and MUST NOT
guidance, and conformance tests exercise the expected patterns
against the mock driver.

## Design Details

### Where reports live, and why

Three Kubernetes objects gain one optional status field each. The
choice of object is the most consequential decision in this KEP, so
it's worth saying plainly why each is where it is.

Controller-reported volume health goes on
`PersistentVolumeClaim.Status.HealthStatus`. The PVC is the object
users already look at when they want to know about a volume, both
through `kubectl describe pvc` and through dashboards that watch
PVC status. The writer is the external monitor sidecar, which runs
in the control plane. No node ever writes here, which closes off
the malicious-node concern that originally pushed the team away
from per-PVC health.

Node-reported per-volume health goes on `Pod.Status.VolumeHealth`,
keyed by the volume name from `pod.spec.volumes`. Three forces
push this decision: first, the kubelet already writes
`Pod.Status` for pods bound to its own node, and the Node Authorizer
already scopes that permission tightly, so no new authorization
surface is needed. Second, the data is naturally pod-scoped: a
filesystem corruption observed from one node is an observation of
that node, and the most useful place to surface it is on the pods
that are actually using the volume from that node. Third, this
choice mirrors KEP-4680 (Resource Health on PodStatus for DRA),
which made the same call for the same reasons; readers building
cross-stack workload health dashboards can treat DRA resource
health and CSI volume health uniformly.

A reasonable alternative would have been a per-PVC
`map[node]HealthReport` field, with each kubelet writing its own
key. We rejected it on two grounds. The first is staleness: a node
that reported `Inaccessible` and then died would leave its entry on
the PVC indefinitely, requiring a sweeper. The second is trust:
allowing kubelets to write PVC status, even with map-keyed-by-node
SSA, reopens the malicious-node concern that the controller-only
write path closes. Storing per-node observations on `Pod.Status`
sidesteps both: pod GC handles staleness, and the kubelet's
existing authorization scope handles trust.

Node-reported backend health goes on
`CSINode.Status.StorageHealth`, keyed by driver name. The CSINode
object is already where the kubelet records facts about the CSI
drivers installed on its node, so this is a natural extension of
that pattern. The kubelet on each node is the sole writer for that
node's CSINode, just as it is for `CSINode.Spec.Drivers` today.

### The CSI surface

The corresponding CSI spec change is
[container-storage-interface/spec#604][spec-pr].

The legacy alpha is removed in that PR. The deletions are:

- the `volume_condition` field on `ListVolumesResponse.VolumeStatus`,
  `ControllerGetVolumeResponse.VolumeStatus`, and
  `NodeGetVolumeStatsResponse`;
- the standalone `message VolumeCondition`; and
- the `VOLUME_CONDITION` controller and node capability values.

None of these were ever declared stable.
Drivers that implemented them continue to compile against the old
spec but will need to migrate to the new RPCs to surface health
under this KEP.

The shared types are `VolumeHealth` (per-volume) and
`StorageBackendHealth` (per-backend). Each carries a status drawn
from a small enum, a required CamelCase `reason` that distinguishes
distinct conditions sharing the same status, and an optional
human-readable `message`. `VolumeHealth` carries a list of entries
rather than a single status, because a volume can exhibit multiple
concurrent conditions and forcing the driver to choose one would
lose information.

```protobuf
message VolumeHealth {
  option (alpha_message) = true;

  message VolumeHealthEntry {
    // The health status category. REQUIRED.
    VolumeHealthErrorType status = 1;

    // A brief CamelCase machine-parseable reason. REQUIRED.
    // Together with status, reason forms the unique identity of
    // an entry: the Plugin MUST NOT return multiple
    // VolumeHealthEntry messages for the same volume with the
    // same (status, reason) combination.
    string reason = 2;

    // A user-friendly description. OPTIONAL.
    string message = 3;
  }

  // The ID of the volume. REQUIRED.
  string volume_id = 1;

  // Health statuses associated with the volume. An empty list
  // means no adverse health condition is known by the Plugin.
  //
  // The SP MAY report multiple concurrent conditions. A future
  // CSI version MAY add VolumeHealthErrorType values; a Plugin
  // that adopts a newer spec MAY report newer values alongside
  // existing ones, but MUST NOT remove an older error entry
  // until that condition is no longer present. The CO MAY
  // ignore unknown VolumeHealthErrorType values.
  //
  // OPTIONAL.
  repeated VolumeHealthEntry health_statuses = 2;
}

enum VolumeHealthErrorType {
  UNKNOWN_VOLUME_HEALTH_TYPE = 0;

  // The volume is not accessible. From
  // ControllerListVolumeHealth / ControllerGetVolumeHealth, the
  // CO MAY interpret this as the volume not being accessible
  // from any node. From NodeGetVolumeHealth, the CO MAY
  // interpret this as the volume not being accessible from that
  // node.
  INACCESSIBLE = 1;

  // Data loss is known or strongly suspected on the underlying
  // volume.
  DATA_LOSS = 2;

  // The volume is usable but is not operating optimally.
  DEGRADED = 3;
}
```

```protobuf
message StorageBackendHealth {
  option (alpha_message) = true;

  // Health status. REQUIRED.
  StorageHealthErrorType status = 1;

  // A brief CamelCase machine-parseable reason. REQUIRED.
  // The Plugin MUST NOT return multiple StorageBackendHealth
  // messages with the same (status, reason, volume_capability)
  // combination.
  string reason = 2;

  // A user-friendly description. OPTIONAL.
  string message = 3;

  // Volume capability affected. OPTIONAL. When set, COs MAY
  // interpret the condition as affecting only volumes of the
  // given capability. For example, RWX multi-attach may be
  // degraded while RWO is unaffected.
  VolumeCapability volume_capability = 4;
}

enum StorageHealthErrorType {
  UNKNOWN_STORAGE_HEALTH_ERROR_TYPE = 0;

  // The storage backend is unreachable from this node. Volumes
  // using this backend are expected to be unavailable.
  STORAGE_UNREACHABLE = 1;

  // The storage backend is operating in a degraded state
  // (e.g. reduced path count, high latency). Volumes using this
  // backend may experience reduced performance.
  STORAGE_DEGRADED = 2;
}
```

`ControllerListVolumeHealth` is the preferred RPC for the external
monitor sidecar. It is paginated, and one polling cycle covers the
volumes the driver knows about. The spec asks drivers to *return
health information about all the volumes that they know about* and
*SHOULD omit volumes with no known adverse health condition*; in
practice every well-behaved driver will return only adverse-condition
entries, but the CO is required to tolerate either shape.

The CO must also tolerate inconsistent paging. The spec says
explicitly that volumes created, deleted, or transitioning health
during a paged list call MAY produce duplicates, omissions, or both,
and the CO SHALL NOT expect a consistent view across pages. This
shapes the reconciliation contract below: a single list pass cannot
reliably tell the CO that a volume has *recovered* (an absence might
be a recovery, or it might be a paging gap), so the sidecar's
recovery rule is more careful than its detection rule.

```protobuf
message ControllerListVolumeHealthRequest {
  option (alpha_message) = true;
  int32 max_entries = 1;
  string starting_token = 2;
  map<string, string> secrets = 3 [(csi_secret) = true];
}

message ControllerListVolumeHealthResponse {
  option (alpha_message) = true;
  // List of volume health entries. Drivers SHOULD omit volumes
  // with no known adverse health condition.
  repeated VolumeHealth entries = 1;
  string next_token = 2;
}
```

For drivers whose backends cannot enumerate efficiently, the sidecar
falls back to per-volume `ControllerGetVolumeHealth` calls. A driver
that advertises neither `LIST_VOLUME_HEALTH` nor `GET_VOLUME_HEALTH`
is not probed by the sidecar at all, and no controller-side health
is recorded for its volumes; this is the default for drivers that
have not opted in.

```protobuf
message ControllerGetVolumeHealthRequest {
  option (alpha_message) = true;
  string volume_id = 1;
  map<string, string> secrets = 2 [(csi_secret) = true];
}

message ControllerGetVolumeHealthResponse {
  option (alpha_message) = true;
  VolumeHealth volume_health = 1;
}
```

On the node side, `NodeGetVolumeHealth` differs from
`NodeGetVolumeStats` in one important way. `NodeGetVolumeStats` is
contractually valid only for staged or published volumes, which
excludes the most interesting unhealthy cases. `NodeGetVolumeHealth`
MAY be called for volumes the CO has merely *attempted* to stage or
publish, including ones whose mount failed because the filesystem
is corrupt. The original alpha could never reach those.

```protobuf
message NodeGetVolumeHealthRequest {
  option (alpha_message) = true;
  string volume_id = 1;
  // The path where the volume is or was expected to be published
  // on the node. OPTIONAL.
  string volume_publish_path = 2;
  // The path where the volume is or was staged. OPTIONAL.
  string staging_target_path = 3;
}

message NodeGetVolumeHealthResponse {
  option (alpha_message) = true;
  VolumeHealth volume_health = 1;
}
```

`NodeGetStorageHealth` covers the conditions that are node-local and
not visible from the controller plane: a top-of-rack switch failure,
a host-side multipath collapse, a NIC misconfiguration that severs
this node's data path while leaving the control plane intact.

```protobuf
message NodeGetStorageHealthRequest {
  option (alpha_message) = true;
  map<string, string> secrets = 1 [(csi_secret) = true];
}

message NodeGetStorageHealthResponse {
  option (alpha_message) = true;
  // Health information for storage backends or classes available
  // from this node. An empty list means the node plugin observes
  // no adverse condition.
  repeated StorageBackendHealth backend_health = 1;
}
```

Drivers advertise support through four new capabilities. The
controller adds `LIST_VOLUME_HEALTH` and `GET_VOLUME_HEALTH` to its
`ControllerServiceCapability` enum; the node adds `GET_VOLUME_HEALTH`
and `STORAGE_HEALTH` to its `NodeServiceCapability` enum. The
controller and node spell their per-volume capability the same way
(`GET_VOLUME_HEALTH`); they are disambiguated by which service-capability
enum they belong to, the way other CSI capabilities already are.

A driver may implement any combination of these or none at all. A
driver that advertises only `GET_VOLUME_HEALTH` on the controller is
probed per-volume by the sidecar; a driver that advertises
`LIST_VOLUME_HEALTH` is probed via list, even if it also advertises
`GET_VOLUME_HEALTH`.

### The Kubernetes surface

Three optional status fields are added to existing types. All are
gated by the `CSIVolumeHealth` feature gate at the apiserver, with
the standard "drop on save when disabled, preserve when already
set" behavior.

The PVC field is the canonical place to look at controller-side
health for a bound volume. It is owned by the external monitor
sidecar and never written by nodes.

```go
type PersistentVolumeClaimStatus struct {
    // ... existing fields ...

    // healthStatus contains the latest controller-reported
    // health information for the volume bound to this claim.
    // Populated by the csi-external-health-monitor-controller
    // sidecar. Nodes do not write this field.
    // +optional
    // +featureGate=CSIVolumeHealth
    HealthStatus *VolumeHealthStatus `json:"healthStatus,omitempty"`
}

type VolumeHealthStatus struct {
    // conditions is the set of adverse conditions reported by
    // the CSI controller plugin. An empty list (or absence)
    // means the controller plugin reports no adverse condition.
    // Conditions are uniquely identified by the (status, reason)
    // tuple, matching the CSI spec's uniqueness rule for
    // VolumeHealthEntry.
    // +optional
    // +listType=map
    // +listMapKey=status
    // +listMapKey=reason
    Conditions []VolumeHealthCondition `json:"conditions,omitempty"`

    // lastProbeTime is the most recent time the CO obtained a
    // response from the controller plugin.
    LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
}

type VolumeHealthCondition struct {
    // status is the machine-parseable category. One of
    // "Inaccessible", "DataLoss", "Degraded".
    Status VolumeHealthStatusType `json:"status"`
    // reason is a brief CamelCase machine-parseable reason
    // (e.g. "VolumeNotFound"). Required; together with status
    // it forms the unique identity of a condition entry.
    Reason string `json:"reason"`
    // message is a human-readable description.
    // +optional
    Message string `json:"message,omitempty"`
    // lastTransitionTime is when this condition entry first
    // appeared at its current (status, reason) tuple.
    LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

type VolumeHealthStatusType string
const (
    VolumeHealthInaccessible VolumeHealthStatusType = "Inaccessible"
    VolumeHealthDataLoss     VolumeHealthStatusType = "DataLoss"
    VolumeHealthDegraded     VolumeHealthStatusType = "Degraded"
)
```

`Conditions` is a `+listType=map` keyed by `(status, reason)`,
matching the CSI spec's uniqueness rule for `VolumeHealthEntry`. A
driver reporting two distinct categories produces two entries, and a
driver reporting the same category with two different reasons (for
example `Degraded`/`HighLatency` and `Degraded`/`MultipathLoss`) also
produces two entries; both forms are first-class conditions, not
collisions. A driver re-reporting an entry whose `(status, reason)`
tuple is unchanged updates `message` and `lastProbeTime` only and
preserves `lastTransitionTime`.

The Pod field is owned by the kubelet on the pod's node. It mirrors
the two-level shape of KEP-4680: an outer entry per `pod.spec.volumes`
name, an inner list of conditions per entry.

```go
type PodStatus struct {
    // ... existing fields ...

    // volumeHealth surfaces node-reported health for each volume
    // the pod is using. Populated by the kubelet on the pod's
    // node. Entries are keyed by the volume name from
    // pod.spec.volumes.
    // +optional
    // +listType=map
    // +listMapKey=name
    // +featureGate=CSIVolumeHealth
    VolumeHealth []PodVolumeHealth `json:"volumeHealth,omitempty"`
}

type PodVolumeHealth struct {
    // name matches an entry in pod.spec.volumes.
    Name string `json:"name"`

    // conditions is the set of adverse conditions reported by
    // the CSI node plugin for this volume on this node. Keyed by
    // (status, reason) to match the CSI spec's uniqueness rule for
    // VolumeHealthEntry.
    // +optional
    // +listType=map
    // +listMapKey=status
    // +listMapKey=reason
    Conditions []VolumeHealthCondition `json:"conditions,omitempty"`

    LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
}
```

The CSINode field is owned by the kubelet on the node, keyed by
driver name. CSINode does not have a `/status` subresource today;
this KEP adds one, and extends the Node Authorizer and the
NodeRestriction admission plugin to allow the kubelet to PATCH it
for its own node only. This matches the scoping pattern the kubelet
already uses to PATCH `csinodes` (no subresource) for
`spec.drivers` registration.

```go
type CSINodeStatus struct {
    // storageHealth is the set of backend health reports for
    // each CSI driver registered on the node, as observed by
    // the kubelet via NodeGetStorageHealth. A single driver may
    // report multiple conditions; entries are uniquely identified
    // by the (name, status, reason) tuple, matching the CSI
    // spec's uniqueness rule for StorageBackendHealth scoped to
    // the registered driver.
    // +optional
    // +listType=map
    // +listMapKey=name
    // +listMapKey=status
    // +listMapKey=reason
    // +featureGate=CSIVolumeHealth
    StorageHealth []StorageHealthCondition `json:"storageHealth,omitempty"`
}

type StorageHealthCondition struct {
    // name is the CSI driver name, matching CSINodeDriver.name.
    Name string `json:"name"`

    // status is one of "StorageUnreachable", "StorageDegraded".
    Status StorageHealthStatusType `json:"status"`

    // reason is a brief CamelCase machine-parseable reason.
    // Required; together with name and status it forms the unique
    // identity of a condition entry.
    Reason string `json:"reason"`

    // message is a human-readable description.
    // +optional
    Message string `json:"message,omitempty"`

    // accessModes are the access modes affected. An empty list
    // means all access modes are affected.
    // +optional
    AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`

    // volumeMode is the volume mode (Filesystem or Block)
    // affected. Nil means both are affected.
    // +optional
    VolumeMode *corev1.PersistentVolumeMode `json:"volumeMode,omitempty"`

    LastProbeTime      metav1.Time `json:"lastProbeTime,omitempty"`
    LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

type StorageHealthStatusType string
const (
    StorageHealthUnreachable StorageHealthStatusType = "StorageUnreachable"
    StorageHealthDegraded    StorageHealthStatusType = "StorageDegraded"
)
```

`AccessModes` and `VolumeMode` exist because backends can degrade
asymmetrically: a multipath fault may affect RWX multi-attach while
RWO works fine, and a block-mode regression on a backend that serves
both block and filesystem volumes may not affect filesystem volumes
at all. These fields let drivers express that asymmetry; consumers
that don't care can ignore them.

### Reconciliation contract

Both writers (the sidecar and the kubelet) do the same thing on
each polling cycle: call the driver, build the desired `Conditions`
list from the response, and PATCH the API object only if the new
list differs from what's stored.

The driver's report is authoritative. The writer overwrites the
stored list with the driver's; it does not merge. A condition the
driver no longer reports is dropped on the next PATCH. The
`+listType=map` keying on `(status, reason)` preserves
`LastTransitionTime` for entries whose tuple is unchanged.

A failed RPC is not a recovery. The writer leaves the stored
conditions in place and increments
`csi_volume_health_probe_total{result="error"}`. Treating an RPC
error as "healthy" would let any driver crash or network blip clear
real conditions.

`ControllerListVolumeHealth` needs one extra rule. The spec lets
paged list results be inconsistent: a volume can appear on cycle N
and be absent from cycle N+1 just because paging shifted, not
because it recovered. The sidecar therefore clears a previously-
unhealthy volume only after two consecutive complete list cycles
in which the volume is absent. A driver that also advertises
`GET_VOLUME_HEALTH` lets the sidecar skip the second cycle and
confirm with a single `ControllerGetVolumeHealth` call instead.
For `ControllerGetVolumeHealth` and `NodeGetVolumeHealth`, an empty
`health_statuses` is the explicit recovery signal and clears the
stored conditions immediately.

The kubelet does not call `NodeGetVolumeHealth` for a volume it has
never attempted to mount. Drivers may gate health probing on
internal state set up at mount time, and asking before that point
isn't useful.

### Authorization

The three writers each need permission scoped as tightly as the
existing model allows. Two of the three already exist; one is new.

- The external monitor sidecar's service account is granted PATCH
  on `persistentvolumeclaims/status`. The sidecar already has the
  GET and LIST permissions it needs on PVCs and PVs in its
  upstream RBAC manifest, so this is the only permission added.
- Kubelet PATCH on `pods/status` is already authorized today by
  the Node Authorizer, with the NodeRestriction admission plugin
  scoping it to pods bound to the kubelet's own node. No new
  authorization surface is needed for `Pod.Status.VolumeHealth`;
  it is just another field reachable through the existing
  `pods/status` subresource. NodeRestriction's existing
  pod-binding check enforces own-pod scoping.
- Kubelet PATCH on `csinodes/status` is new. CSINode has no
  `/status` subresource today, and the Node Authorizer explicitly
  rejects all `csinodes` subresources. This KEP adds the `/status`
  subresource on the CSINode resource and extends both the Node
  Authorizer and NodeRestriction admission so a kubelet can PATCH
  the CSINode whose name matches its own node and no other. The
  scoping is identical to the kubelet's existing PATCH of the main
  `csinodes` resource for `spec.drivers`.

### Feature gate

A single feature gate, `CSIVolumeHealth`, gates the apiserver and
kubelet. When disabled, the apiserver drops the new fields on save
(preserving them if already set on the old object, per the standard
field-level feature-gate handling), the kubelet does not call the
new RPCs, and the Node Authorizer extensions are inactive.

The external monitor sidecar does not depend on a feature gate. Its
deployment is the controller-side opt-in. If the apiserver does not
have `CSIVolumeHealth` enabled, the sidecar's writes are silently
dropped server-side, which is the right behavior: the sidecar
doesn't need to know whether the cluster has the feature on, only
that its own writes succeed or fail.

We will include an optimization to avoid empty updates from external monitor
sidecar if health updates are being rejected by the API server (which indicates
`CSIVOlumeHealth` featuregate is disabled in API server), we will modify external sidecar
behavior to post health updates at a much slower rate.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

Two pieces of test infrastructure are extended in lockstep with the
implementation:

- [`kubernetes-csi/csi-test`](https://github.com/kubernetes-csi/csi-test): The CSI sanity
  tests that verify if plugins implement these calls correctly.
- [`kubernetes-csi/csi-driver-host-path`](https://github.com/kubernetes-csi/csi-driver-host-path):
  the hostpath driver gains a flag to inject specific health states,
  for upstream e2e jobs.

##### Unit tests

- `k8s.io/kubernetes/pkg/apis/core/validation` and
  `k8s.io/kubernetes/pkg/apis/storage/validation`: validation of
  the new fields, including the composite-key uniqueness rules
  (`(status, reason)` for `VolumeHealthCondition` and
  `(name, status, reason)` for `StorageHealthCondition`) and
  rejection of empty `reason` values.
- `pkg/registry/core/{persistentvolumeclaim,pod}` and
  `pkg/registry/storage/csinode`: feature-gate drop-on-save,
  including the case where the field is already set on the old
  object and the gate is now disabled.
- `pkg/kubelet/volumemanager`: probing, `Pod.Status` patching, and
  no-op suppression.
- `plugin/pkg/auth/authorizer/node`: own-pod and own-node scoping
  for the new PATCH permissions.
- `kubernetes-csi/external-health-monitor`: capability detection,
  list-vs-get fallback, no-op suppression, and the
  two-cycle (or Get-confirmed) recovery rule.

##### Integration tests

None

##### e2e tests

The e2e tests at this stage will mostly be written using 
hostpath CSI driver and mock injection hooks in e2e framework.

- Volume-side enums (`Inaccessible`, `DataLoss`, `Degraded`)
  injected via the hostpath driver: verify each surfaces on
  `pvc.status.healthStatus` and `pod.status.volumeHealth`, then
  recovery clears the condition.
- Backend-side enums (`StorageUnreachable`, `StorageDegraded`)
  injected via the hostpath driver: verify each surfaces on
  `csinode.status.storageHealth`, then recovery clears.

### Graduation Criteria

#### Alpha

- The CSI spec changes land in
  [container-storage-interface/spec][spec-pr].
- The mock CSI driver implements the new RPCs.
- The `CSIVolumeHealth` feature gate is plumbed through the apiserver
  and kubelet, including field validation, drop-on-save, the new
  `csinodes/status` subresource, and the Node Authorizer and
  NodeRestriction admission extensions.
- The external monitor sidecar uses the new RPCs and writes to
  `pvc.status.healthStatus`.
- Initial unit and integration tests are in place.

#### Beta

- All e2e tests are implemented, in TestGrid, and stable for at
  least two minor releases.
- At least two CSI drivers other than the mock driver implement
  the new capabilities in production.
- A scalability test on a cluster of at least 5,000 PVCs across at
  least 100 nodes confirms that steady-state apiserver write rate
  remains at or near zero with no-op suppression in place.
- Documentation is merged into kubernetes.io.

#### GA

- No outstanding bugs against the feature for two releases.
- At least one downstream remediation user (a reactor controller
  built on top of the alpha or beta status fields) is in production.
- Conformance tests for the new RPCs run against drivers that
  advertise the relevant capabilities.

### Upgrade / Downgrade Strategy

The upgrade order is:

1. Roll out the apiserver with `CSIVolumeHealth` enabled. The new
   status fields begin to be accepted.
2. Roll out kubelets with the gate enabled.
3. Update CSI drivers that implement the new capabilities. Drivers
   that don't are unaffected.
4. Deploy the external monitor sidecar alongside CSI controllers
   that opt in to controller-side reporting.
   
Step #2 and Step#3 can be performed in any order. If driver is upgraded
before kubelet has `CSIVOlumeHealth` featuregate enabled, volume
health simply will not be queried.

Downgrade reverses the order. The sidecar should be removed before
the apiserver gate is disabled, to avoid spurious failed-PATCH log
noise. Disabling the gate on the apiserver stops new writes to the
new fields and drops them from new objects on the next write;
existing values on disk are preserved until the next write to the
object. Older kubelets without the feature simply do not call the
new RPCs.

### Version Skew Strategy

The skew matrix is small. Health writes are best-effort and never
on the data path, so every skew combination degrades cleanly:

- New kubelet against old apiserver: the kubelet attempts to PATCH
  the new status subresources, gets rejected, logs the failure, and
  continues. Running workloads are unaffected; only health
  visibility is suppressed.
- New apiserver with old kubelets: no node-side writes happen.
  Controller-side reports on PVC still work, because the sidecar
  writes to the apiserver directly.
- Mixed kubelet rollout: `Pod.Status.VolumeHealth` is populated
  inconsistently across the cluster, depending on which node a pod
  runs on. Consumers handle this the way they handle any other
  best-effort `Pod.Status` field. Absence of a value is not the
  same as a healthy report, and dashboards joining health to pod
  identity should be node-aware.
- Old driver: a driver that does not advertise the new capabilities
  is not probed by either writer, and the feature is dormant for
  that driver. There is no pressure on driver authors to upgrade.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `CSIVolumeHealth`
  - Components depending on the feature gate: `kube-apiserver`,
    `kubelet`.

For controller-side reporting, deployment of the
`csi-external-health-monitor-controller` sidecar is also required.
The sidecar's deployment is itself the controller-side opt-in.

###### Does enabling the feature change any default behavior?

The apiserver accepts and persists three new optional status
fields, which are not populated unless something writes to them.
A kubelet whose nodes have CSI drivers advertising the node-side
`GET_VOLUME_HEALTH` or `STORAGE_HEALTH` capability issues periodic
RPCs to those drivers. No scheduling, admission, or
volume reconciliation behavior changes.

###### Can the feature be disabled once it has been enabled?

Yes. Disabling the feature gate stops new writes to the new
fields and drops them from new objects on the next write per
standard field-level feature-gate handling. Existing values on
disk are preserved until the next write to the object. Disabling
does not break any existing workload: nothing in this KEP changes
scheduling, admission, or pod-admission outcomes.

###### What happens if we reenable the feature if it was previously rolled back?

Probing resumes. Status fields begin to be populated again on the
next polling cycle. Stale values from before the rollback are
overwritten on the next write.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests in the relevant `pkg/registry/...` packages
exercise the field-level feature gate, including the case where
the field was set on the old object and the gate is now disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The feature does not touch the data path. A rollout failure
(a sidecar crash, a kubelet panic on the new code path) is detectable
through standard component metrics. Running workloads continue to
run; only the visibility of volume health is affected.

###### What specific metrics should inform a rollback?

A rising `csi_volume_health_probe_total{result="error"}` rate
post-rollout indicates driver-side or transport failures. A rising
apiserver PATCH rate for `persistentvolumeclaims/status`,
`pods/status`, or `csinodes/status` that correlates with the
rollout indicates either a bug in a writer or a misbehaving driver
flapping its reports. Both writers are designed to suppress no-op
writes, so a sustained rate is a signal worth investigating.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Manual validation across at least one minor version skew (n / n-1)
is planned during alpha. Automated upgrade/downgrade e2e is in
scope for beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

The CSI spec removes the legacy alpha `VOLUME_CONDITION` capability
and `volume_condition` fields. No deprecation period applies because
the prior shape was alpha and never declared stable. Drivers that
implemented the alpha shape continue to compile against the old
spec but must migrate to the new RPCs to surface health under this
KEP.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

`kubectl get pvc -A -o json | jq '.items[] | select(.status.healthStatus != null)'`
enumerates PVCs with controller-reported health. The analogous
queries against `pods` and `csinodes` enumerate node-side health.
The `csi_volume_health_probe_total` counter, exposed by both the
sidecar and the kubelet, shows the ongoing probing rate.

###### How can someone using this feature know that it is working for their instance?

- [X] API .status
  - Fields: `pvc.status.healthStatus`,
    `pod.status.volumeHealth`,
    `csinode.status.storageHealth`.
- [X] Metrics
  - `csi_volume_health_probe_total` increments when probes run.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The success rate of CSI health RPCs is comparable to the success
rate of other CSI RPCs the driver implements; this KEP does not
introduce a new latency tier. The lag between a driver reporting a
new condition and that condition appearing on the corresponding
Kubernetes status field is bounded by one probe interval.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - `csi_volume_health_probe_duration_seconds`: histogram of
    health-probe RPC latency.
  - `csi_volume_health_probe_total`: counter of probes by
    outcome.
  - `csi_controller_volume_health_status`: gauge posted per condition from controller for every unhealthy volume.
  - `csi_node_storage_health_status`: gauge posted per condition from node for every unhealthy volume.
  - `csi_node_storage_backend_health_status`: gauge posted from node for overall health of the storage backend as visible from CSI driver.
  
  Exact label sets are defined alongside the implementation, not
  in this KEP, so they can evolve without a KEP amendment.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Not at alpha. Reviewer feedback during alpha → beta will inform
additions.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

The feature has two dependencies, both of which are opt-in:

- The `csi-external-health-monitor-controller` sidecar, for
  controller-side health reporting. Absent it, the controller side
  is dormant and the feature degrades gracefully to node-side
  reporting only.
- CSI driver implementations of the new RPCs and capabilities, for
  the corresponding probes to do anything. Absent them, the feature
  is dormant for that driver.

### Scalability

###### Will enabling / using this feature result in any new API calls?

- The sidecar PATCHes `persistentvolumeclaims/status` at most once
  per PVC per polling cycle, with no-op suppression on unchanged
  values.
- The kubelet PATCHes `pods/status` at most once per pod per
  probing cycle, with no-op suppression.
- The kubelet PATCHes `csinodes/status` at most once per driver
  per probing cycle, with no-op suppression.
- The sidecar LIST/WATCHes `persistentvolumeclaims` and
  `persistentvolumes`, which is already done by similar sidecars.

In steady state with no unhealthy volumes, no-op suppression brings
sustained PATCH rate to zero. Under sustained all-unhealthy load,
the rate is bounded by the writer's polling interval, which is
operator-configurable.

###### Will enabling / using this feature result in introducing new API types?

No new API types. Three new optional status fields on existing
types: PVC, Pod, CSINode.

###### Will enabling / using this feature result in any new calls to the cloud provider?

Indirectly, via the CSI driver: `ControllerListVolumeHealth`
implementations typically call into the storage backend's API.
The cost depends on the driver and backend; this KEP does not
prescribe an implementation.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Three new optional status fields. Each entry holds a small list
of conditions (status / reason / message / timestamps). Fields
are populated only when a driver reports adverse conditions, so
typical steady-state cost is zero.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. The feature runs out-of-band of the volume attach / mount
path.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

The external monitor sidecar's resource profile is that of the
existing alpha sidecar, applied to a slightly different set of
RPCs. The kubelet adds one periodic gRPC call per registered
driver and a per-PVC call bounded by the set of `InUse` volumes.
Apiserver work is bounded by the PATCH rates above.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The kubelet and the sidecar log errors and back off. No status
writes happen. When the apiserver is reachable again, the next
polling cycle resumes writing.

###### What are other known failure modes?

- A driver returning errors on `ControllerListVolumeHealth` is
  detectable through `csi_volume_health_probe_total` with a
  non-success result label. The sidecar falls back to per-volume
  `ControllerGetVolumeHealth` if the driver advertises that
  capability; otherwise it logs and skips the cycle.
- A driver reporting stale `Inaccessible` after a transient outage
  has resolved shows up as `LastTransitionTime` not advancing
  despite `LastProbeTime` advancing. The mitigation is to file a
  driver bug; operators can clear
  `pvc.status.healthStatus.conditions` by hand, and the sidecar
  will overwrite on the next cycle.
- A sidecar uninstalled while unhealthy entries are present leaves
  those entries on PVCs with no writer to clear them. Re-installing
  the sidecar clears stale entries on the next cycle if the volume
  has recovered; an admin can also clear the field directly.

###### What steps should be taken if SLOs are not being met to determine the problem?

Inspect the relevant component's metrics and logs (sidecar for
controller-side, kubelet for node-side), correlate with the
driver's own logs, and identify whether the issue is in the
driver, the CO, or the apiserver path.

## Implementation History

- 2019-05-30: Original KEP submitted as monitoring-only.
- 2020-05-12: KEP creation date in current form.
- 2021-01-17: Update for original alpha (v1.21).
- 2021-09-02: Add volume health to kubelet metrics.
- 2026-02-25: Working group concludes that node-reported per-volume
  health belongs on `Pod.Status` (DRA-aligned) and storage-backend
  health on `CSINode.Status`.
- 2026-05-21: Provisional CSI spec PR opened
  ([container-storage-interface/spec#604][spec-pr]).
- 2026-05-27: KEP rewritten end-to-end against the redesigned CSI
  APIs and Kubernetes status fields. Targeting v1.37 alpha.

## Drawbacks

N/A.

## Alternatives

### Use only metrics and events for health reporting

An earlier version of this KEP only used metrics and events for health reporting.
Events are an operator-affordance, not a state
machine. They have a TTL, they aren't joined to the PVC by API
consumers in any structured way, and they can't be driven by a
remediation controller without screen-scraping. Anyone serious about
acting on volume health needs durable status.

### Embed VolumeCondition in existing RPCs (the original alpha)

The original alpha embedded `VolumeCondition` in `ListVolumes`,
`ControllerGetVolume`, and `NodeGetVolumeStats` and surfaced findings
as Kubernetes events and metrics. The Motivation section explains why this shape
is being replaced rather than promoted.

### A standalone per-PVC VolumeHealth CRD

Considered, with controller-reported health and a
`map[node]HealthReport` of node-reported health. Routing node writes
through a CRD doesn't address the trust concern that pushed the
original design away from per-PVC health, because nodes still have to
be granted PATCH on the CRD; it just moves the authorization problem.
It also adds CRD installation, RBAC, and lifecycle burden to
operators. Storing per-node observations on `Pod.Status` instead
avoids both, and pod GC handles staleness automatically.

### A per-PVC map[node]Health directly on pvc.status

Considered for the same reasons and rejected for the same reasons.
Granting kubelets PATCH on `pvc/status`, even with map-keyed-by-node
SSA, reopens the malicious-node concern that the controller-only
write path closes.

### PV taints with NoEffect

Considered as a way to encode health in an existing mechanism. PV
taints were never approved upstream for general use, and `Taint` was
designed to express scheduling effects, not status; the `value` field
is a single string, so encoding multiple concurrent conditions
degenerates into an ad-hoc protocol.

### DRA-style device taints for storage (KEP-5055)

Considered. We could not identify a use case where a workload author
would write a `Toleration` for, say, `Degraded` storage. Storage
remediation is overwhelmingly an operator concern (rebuild from
snapshot, fail over to a replica), not a tolerations concern.
`DataLoss` does not translate cleanly into a taint effect, because
the right reaction is application-specific. Backend health on
`CSINode` would need multiple taints per driver per access mode per
volume mode, which would force `Taint` to grow new fields.

### Push-based health ingest

Considered. CSI is pull-based by design, and introducing push
semantics here would diverge from the rest of the contract. Future
versions of CSI may add streaming RPCs; at that point this KEP can
revisit.

### A richer error enum

A larger enum like `MULTIPATH_LOSS`, `OUT_OF_CAPACITY`, `DISK_FULL`,
`PERFORMANCE_DEGRADED` was considered. Most of these candidates are
not CO-actionable: they are storage-admin or application-author
concerns, and the `reason` field already gives drivers a place to
surface storage-specific signals. Adding them to the enum would force
every CO consumer to handle them as permanently-supported values,
which is a lot of weight to pay for signals that don't drive CO
behavior.

## Future enhancement 

This section describes future enhancements we are planning to volume health reporting
that will make current feature set more rich and error reporting more granular.

1. Let CSI driver configure polling interval for both node and controller health via 
configurable values in `CSIDriver` object.

2. The topic of specific backend of a driver being affected with bad health condition was
brought up during reviews. Many drivers support multiple backends - such as ceph or topolvm,
where driver can provision, publish a volume from specific backend. Usually backend information
is opaque to k8s. But We could in future add `backend_id` as a first class field in PV and further 
specify it in health responses. This will allow drivers to report volume health at more granular 
level from different backends.

3. Report entire storage backend health from control-plane as well.

## Infrastructure Needed

- Mock CSI driver
  ([`kubernetes-csi/csi-test`](https://github.com/kubernetes-csi/csi-test))
  extended to implement the new RPCs and capabilities.
- Hostpath CSI driver
  ([`kubernetes-csi/csi-driver-host-path`](https://github.com/kubernetes-csi/csi-driver-host-path))
  extended with a flag to inject specific health states for upstream
  e2e jobs.
- `csi-external-health-monitor-controller` sidecar
  ([`kubernetes-csi/external-health-monitor`](https://github.com/kubernetes-csi/external-health-monitor))
  updated to use the new RPCs and write to
  `pvc.status.healthStatus`.

[spec-pr]: https://github.com/container-storage-interface/spec/pull/604
