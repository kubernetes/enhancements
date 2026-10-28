# KEP-6274: GPU support for Pod-level Checkpoint/Restore

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Inference warm start](#inference-warm-start)
    - [Single-process training recovery](#single-process-training-recovery)
  - [Scope and terminology](#scope-and-terminology)
  - [Deployment profiles](#deployment-profiles)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API and feature-gate behavior](#api-and-feature-gate-behavior)
  - [Current implementation evidence](#current-implementation-evidence)
  - [Checkpoint content](#checkpoint-content)
  - [Checkpoint flow](#checkpoint-flow)
  - [Restore flow](#restore-flow)
  - [DRA allocation path](#dra-allocation-path)
    - [ResourceClaim lifecycle](#resourceclaim-lifecycle)
    - [Compatibility selection](#compatibility-selection)
  - [GPU identity remapping](#gpu-identity-remapping)
  - [Runtime and CRI contract](#runtime-and-cri-contract)
  - [Failure and retry semantics](#failure-and-retry-semantics)
  - [Security and privacy](#security-and-privacy)
  - [Component responsibilities](#component-responsibilities)
  - [Implementation workstreams](#implementation-workstreams)
  - [Open Questions](#open-questions)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
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
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed](#infrastructure-needed)
- [References](#references)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required prior to targeting a milestone or release.

- [ ] (R) Enhancement issue in a release milestone, linking to this KEP
- [ ] (R) KEP approvers have approved the KEP as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place
- [ ] (R) Graduation criteria are in place
- [ ] (R) Production readiness review is completed and approved
- [ ] User-facing documentation is published
- [ ] Supporting runtime, GPU Operator, DRA-driver, and CRIU documentation is linked

## Summary

[KEP-5823] introduces Pod-level checkpoint and restore, but deliberately excludes
devices attached through the Kubernetes device-plugin framework or Dynamic
Resource Allocation (DRA), as well as device memory and driver state. This KEP
extends that mechanism so a Pod using a GPU can be checkpointed and restored as
one operation.

The Alpha design supports same-node restore into a new Pod while the source Pod
continues running. The new Pod receives a new GPU allocation through DRA. At
checkpoint time, the runtime stores the complete GPU restore data in its
protected archive and reports a small set of placement requirements. Kubelet
records those requirements in `PodCheckpoint.status`. When the restore Pod is
created, the ResourceClaim controller adds them as constraints on the generated
claim before normal DRA scheduling and device preparation run.

The public requirements contain the DRA driver name, an opaque
`driverStackID`, and named minimum capacities, with GPU memory as the Alpha
capacity. Raw GPU identifiers, GPU memory contents, driver state, exact software
versions, and individual file identities remain in the runtime archive. The
runtime uses that complete private record to validate the prepared allocation
and map the checkpointed GPU identity to the new GPU before any restored process
is allowed to run.

NVIDIA GPUs are the first implementation and end-to-end validation target. The
Kubernetes contract is device- and vendor-neutral so that AMD GPUs and other
checkpointable devices can implement it later without changing the API.

[KEP-5823]: /keps/sig-node/5823-pod-level-checkpoint-restore/README.md

## Motivation

GPU workloads are among the workloads that benefit most from checkpoint and
restore. Initializing an inference server or recovering a training process can
require loading large model and optimizer state, rebuilding host state, and
warming device kernels and caches. CPU-only process checkpointing is not enough:
the GPU driver owns memory allocations, execution contexts, streams, and device
handles that must be transitioned together with the process tree.

Linux GPU checkpoint implementations exist below Kubernetes. For example, the
CUDA Driver API can lock a process, move its GPU memory into host memory, release
the GPU allocation, restore it onto a compatible GPU, and unlock the process.
CRIU GPU plugins coordinate those operations with process checkpoint and restore.
The missing contract is at the Kubernetes layer:

- a restored Pod has a new UID and therefore a new device allocation;
- the source Pod remains running in the KEP-5823 Alpha lifecycle and retains its
  device allocation;
- device allocation and DRA preparation must complete before process restore;
- the newly allocated physical device may differ from the checkpointed device;
- DRA has structured device attributes, but claim ownership and Pod-template
  equality must remain correct across restore;
- checkpoint-specific requirements must survive long enough to constrain the
  new claim before scheduling;
- the restore environment must provide the same restore-relevant driver stack
  and GPU libraries used at checkpoint time; and
- Kubernetes must never report a CPU-only checkpoint as successful when GPU
  state was omitted.

This KEP defines those lifecycle boundaries without teaching Kubernetes how to
interpret a particular vendor's driver state.

### Goals

- Checkpoint and restore a Pod using one full GPU on Linux.
- Support GPUs allocated through DRA using a generated ResourceClaim from a
  ResourceClaimTemplate.
- Preserve the KEP-5823 Alpha lifecycle: same-node restore, a new Pod UID, and a
  source Pod that remains running.
- Allocate and prepare a new device before calling CRI `RestorePod`.
- Record bounded device restore requirements with the checkpoint and use them
  to exclude incompatible DRA devices before allocation.
- Let the runtime map the original device to the new allocation and perform the
  final compatibility check without starting a fresh process on failure.
- Treat CPU and GPU capture as one checkpoint result: report success only when
  both artifacts are complete and mutually consistent, and otherwise discard
  partial output.
- Pin the source `ResourceClaimTemplate` by UID and make the controller's
  restore-time changes strictly additive to the generated claim.
- Define cleanup, retry, observability, security, feature-gate, and version-skew
  behavior for device-bearing checkpoints.
- Validate the design with NVIDIA GPUs allocated through DRA.
- Keep the Kubernetes contract vendor-neutral and suitable for a future second
  GPU implementation.

### Non-Goals

- Cross-node or cross-cluster restore. This remains dependent on the future
  checkpoint transport work in KEP-5823.
- Stopping or deleting the source Pod after checkpoint, or releasing its
  resources for the restored Pod.
- In-place restore into the source Pod object or UID.
- Live migration or a restore-latency SLO.
- More than one GPU visible to the GPU process in Alpha.
- Pods with another device allocation in addition to the single GPU in Alpha.
- Claims shared through a PodGroup or another workload-level claim mechanism.
- Multiple GPU-using processes, CUDA IPC peers, or more than one GPU-using
  container in a Pod.
- MIG, time slicing, MPS, vGPU, VFIO, or other shared, partitioned, or
  reconfigurable device modes.
- NCCL state, distributed or multi-Pod checkpoint coordination, ComputeDomains,
  network-identity preservation, or Pod IP rewriting.
- Checkpointing CUDA Unified Memory or allocations exported through
  `cuMemExportToShareableHandle` while those remain unsupported by the GPU
  checkpoint implementation.
- Storing raw NVIDIA driver versions, CUDA versions, GPU models, GPU UUIDs,
  per-file digests, or CRIU plugin versions in Kubernetes APIs. Alpha stores an
  opaque driver-scoped stack identifier instead.
- A custom scheduler plugin or a dependency on GPU Feature Discovery labels.
- GPUs allocated directly through the device-plugin framework. Alpha focuses on
  DRA's structured device selection; device-plugin support may be reconsidered
  after the DRA path is proven.
- Requiring the NVIDIA GPU Operator. Preinstalled drivers and toolkit components
  remain valid when they satisfy the same node/runtime contract.
- AMD GPU support in Alpha. AMDGPU/ROCm maintainers should review the contract,
  and a second implementation is a Beta goal.

## Proposal

Introduce a `PodLevelCheckpointRestoreDevices` feature gate layered on
`PodLevelCheckpointRestore`. The gate is registered in kube-apiserver,
kube-controller-manager, and kubelet. The API server stores and validates the
device restore requirements, the ResourceClaim controller applies them when it
generates a restore claim, and kubelet handles the CRI and status portions of
the contract. The scheduler needs no new logic because it evaluates the
resulting claim through normal DRA processing.

On checkpoint, the runtime discovers the GPU used by the sandbox and creates
one canonical requirement record. It commits the complete record to the opaque
checkpoint directory and includes the bounded placement subset in
`CheckpointPodResponse`. Kubelet associates that response with the source Pod's
logical DRA claim and request, adds the DRA driver and
`ResourceClaimTemplate` UID from trusted Kubernetes state, and stores the
result in `PodCheckpoint.status`. The ResourceClaim controller records that UID
when it creates the source claim, using the protected
`resource.kubernetes.io/resource-claim-template-uid` annotation. An in-tree
admission check rejects that reserved key in
`ResourceClaimTemplate.spec.metadata.annotations`. On a Pod-owned ResourceClaim
create that contains the key, admission performs a synthetic authorization
check for the `set` verb on the virtual
`resourceclaims/template-provenance` subresource. Only the built-in
ResourceClaim-controller role receives that permission. Updates cannot change
or remove the value. Kubelet reads it from the allocated claim rather than
gaining permission to read ResourceClaimTemplates.

On restore, the ResourceClaim controller copies the matching
`ResourceClaimTemplate` and adds system-owned constraints for the recorded DRA
driver, exact `driverStackID`, and minimum capacities. The DRA driver publishes
the stack attribute only for checkpoint-capable, exclusive full devices, so
matching that attribute also excludes its shared and partitioned devices. These
changes only narrow the copied claim. The referenced `DeviceClass` remains
mutable and is resolved at restore time, as it is for ordinary Pod creation;
this KEP does not preserve the class contents that existed when the source Pod
was allocated. The scheduler allocates a matching device, the DRA driver
prepares it, and kubelet passes the resulting CDI configuration to `RestorePod`.
The runtime then checks the actual device and the complete private requirements,
builds the old-to-new device map, restores CPU and GPU state, and reports the
sandbox and container IDs while the restored processes are still stopped, as
required by KEP-5823.

### User Stories

#### Inference warm start

An operator runs an inference Pod on a node with two compatible GPUs. After the
server has loaded model state and reached a useful execution point, the operator
creates a `PodCheckpoint`. The source continues serving. A second Pod with an
equivalent spec and `spec.restoreFrom` is scheduled to the same node, obtains the
second GPU, and resumes from the captured process state without repeating the
full initialization path.

The first Alpha does not claim a production warm-start latency SLO. It proves
that Kubernetes allocation, device remapping, and process restoration form a
correct lifecycle.

#### Single-process training recovery

An application periodically produces Pod checkpoints during single-GPU
fine-tuning. If the application needs to be cloned for validation or manually
recovered on the same node, an authorized user creates a new Pod from a Ready
checkpoint. The new Pod obtains another compatible full GPU and resumes from the
captured iteration.

Automatic failover after node loss, releasing the source GPU, and multi-GPU or
distributed training recovery are future work.

### Scope and terminology

The following terms are distinct in this KEP:

- **Device allocation:** the scheduler assigns a DRA device to a Pod through a
  ResourceClaim.
- **Device preparation:** a node plugin performs host-side operations necessary
  before a DRA device can be consumed. For DRA this is `NodePrepareResources`.
- **Device injection:** kubelet and the container runtime apply devices, mounts,
  environment, annotations, or CDI edits to a container.
- **Device checkpoint state:** opaque memory and driver state captured by the
  GPU checkpoint implementation.
- **Device restore requirements:** the bounded, scheduler-facing subset of the
  checkpoint requirements. Alpha records the logical DRA claim and request,
  DRA driver, source ResourceClaimTemplate UID, `driverStackID`, and named
  minimum capacities.
- **Driver stack ID:** an opaque, driver-scoped identifier, no more than 64
  characters, for the restore-relevant GPU hardware class and host driver
  stack. The GPU runtime and DRA driver define the same ID algorithm. For the
  NVIDIA implementation it distinguishes incompatible chip types or device
  modes, kernel and user-space driver stacks, CDI-injected libraries such as
  `libcuda.so`, and the CUDA checkpoint ABI. It includes an implementation
  scheme version so a change to the ID algorithm is explicit. Kubernetes
  compares the value for equality but does not parse it.
- **Restore file manifest:** the runtime-private list of files that CRIU or the
  GPU restore implementation must reopen or map during restore. Each entry has
  a path and strong content identity. This includes host- or CDI-injected GPU
  libraries and is never copied into the Kubernetes API.
- **Original device identity:** the runtime-private identity of each GPU visible
  to the checkpointed process.
- **Restore device identity:** the runtime-private identity resolved from the
  new Pod allocation.
- **Compatibility:** a two-step decision. DRA first filters devices using the
  public driver stack ID and minimum-capacity requirements. The runtime then
  checks the prepared device against the complete archive, including device
  properties omitted from the public projection, checkpoint-format constraints,
  and exact restore-file identities. Runtime validation remains authoritative
  because node state can change after scheduling.
- **Device mapping:** the complete mapping from every original visible GPU to a
  restore GPU.

Alpha permits CPU-only sidecars supported by KEP-5823, but exactly one container
may use a GPU, exactly one GPU process is supported, and exactly one full GPU may
be visible to that process.

Because the source Pod remains running and retains an exclusive GPU, successful
Alpha restore normally requires a second compatible GPU on the same node. A node
with only one exclusive GPU cannot run the source and restored Pods concurrently.

### Deployment profiles

The initial NVIDIA implementation validates GPU Operator plus the NVIDIA DRA
driver. The GPU Operator provisions the driver and Container Toolkit/CDI
integration. The NVIDIA device plugin is disabled, and the NVIDIA DRA driver
allocates full GPUs through generated ResourceClaims. The tested DRA driver
build must also publish `driverStackID` and `memory` for each checkpoint-capable
full GPU and must not publish the stack attribute for a partitioned or
shareable device.

The GPU Operator is a deployment and lifecycle tool, not part of the Kubernetes
checkpoint correctness contract. A cluster with a preinstalled driver, toolkit,
and equivalent plugins is supported when the same runtime prerequisites and
tests pass.

CDI is the device handoff: the DRA driver reports CDI device names and supported
container runtimes apply the corresponding device specification. The contract
does not require GPU Operator's NRI mode, and NRI is not inserted into the
checkpoint or restore ordering.

### Risks and Mitigations

- **A GPU checkpoint failure can damage the source process.** The CUDA
  checkpoint API can report an unrecoverable failed state. The runtime must
  inspect and report the final state, resume the process only when that
  transition is valid, fail the entire Pod checkpoint, and remove partial
  artifacts. The KEP does not promise that every failed GPU checkpoint leaves
  the source workload healthy.
- **The advertised device can differ from the restore environment.** The
  generated claim filters on the checkpoint's driver stack ID and minimum
  capacities before allocation. The runtime still checks the prepared device and
  private file manifest before the restored process runs, which catches driver
  or mount changes after the ResourceSlice was published.
- **The source holds the only usable GPU.** Documentation and admission examples
  require a node with at least two compatible full GPUs. Stopping the source and
  transferring its allocation is not hidden inside this enhancement.
- **GPU memory can create large host-memory and disk pressure.** Implementations
  must honor the parent checkpoint timeout and concurrency controls, expose
  duration and checkpoint-size metrics, fail cleanly on insufficient resources,
  and avoid unbounded parallel GPU checkpoints.
- **Checkpoint support can change with the node software stack.** The DRA
  driver publishes a driver stack ID instead of asking Kubernetes to interpret
  version strings. The runtime also checks the exact archive format and files,
  and supported combinations remain part of the implementation's test matrix.
- **Raw device identity can reveal node inventory.** GPU UUIDs and the detailed
  restore file manifest remain in the protected checkpoint archive and do not
  appear in Kubernetes status, events, or metric labels.
  The bounded driver, stack ID, template UID, and capacity requirements are
  stored in status because the scheduler needs them.
- **Retries may repeatedly invoke expensive restore.** Transient failures use
  the existing kubelet sandbox backoff. A persistent incompatibility sets a
  `Restoring=False` condition, and kubelet suppresses ordinary retries as soon
  as it receives `FailedPrecondition`. A kubelet crash before the condition is
  stored can cause another attempt; this narrow at-least-once window is explicit
  rather than hidden. Recovery requires correcting the compatibility problem
  and recreating the Pod.
- **The initial implementation is NVIDIA-only.** Kubernetes-facing behavior is
  expressed in terms of allocated devices, CDI, opaque runtime state, and CRI
  results. AMDGPU reviewers will be asked to validate that no NVIDIA assumption
  leaks into the Kubernetes contract.

## Design Details

### API and feature-gate behavior

Alpha adds `status.deviceRestoreRequirements` to `PodCheckpoint` and extends
the CRI `CheckpointPodResponse`. It adds no field to `Pod`, `ResourceClaim`,
`ResourceClaimTemplate`, or `ResourceSlice`, and no new top-level API kind. A
supporting DRA driver publishes two existing structured-parameter values under
its driver domain: a string attribute named `driverStackID` and a byte-valued
capacity named `memory`.

The status field is a map list keyed by `claimName` and `requestName`. Alpha
writes exactly one entry; the API allows at most 32 so a later multi-device
stage does not require a second storage shape. The following excerpt defines
the Go and JSON shape. The implementation will assign new, non-conflicting
protobuf field numbers and generate the usual serialization code.

```go
type PodCheckpointStatus struct {
    // Existing fields omitted.

    // +optional
    // +listType=map
    // +listMapKey=claimName
    // +listMapKey=requestName
    // +k8s:maxItems=32
    DeviceRestoreRequirements []PodCheckpointDeviceRestoreRequirement `json:"deviceRestoreRequirements,omitempty"`
}

type PodCheckpointDeviceRestoreRequirement struct {
    // Name of the PodResourceClaim in the checkpointed Pod template.
    // +required
    ClaimName string `json:"claimName"`
    // Name of the exact DRA DeviceRequest used by the GPU container.
    // +required
    RequestName string `json:"requestName"`
    // DRA driver that owned the source allocation.
    // +required
    Driver string `json:"driver"`
    // UID of the ResourceClaimTemplate named by ClaimName.
    // +required
    ResourceClaimTemplateUID types.UID `json:"resourceClaimTemplateUID"`
    // Opaque exact-match value also published by the DRA driver.
    // +required
    DriverStackID string `json:"driverStackID"`
    // Minimum values for named DRA device capacities.
    // +required
    // +listType=map
    // +listMapKey=name
    // +k8s:maxItems=32
    MinimumCapacities []PodCheckpointDeviceCapacityRequirement `json:"minimumCapacities"`
}

type PodCheckpointDeviceCapacityRequirement struct {
    // +required
    Name  resourcev1.QualifiedName `json:"name"`
    // +required
    Value resource.Quantity        `json:"value"`
}
```

`claimName` and `requestName` use the existing DNS-label validation for their
source fields. `driver` uses the DRA driver-name validation and its 63-character
limit. `resourceClaimTemplateUID` is required. `driverStackID` is a non-empty
ASCII token matching `[A-Za-z0-9][A-Za-z0-9._:+-]{0,63}`, within the DRA
string-attribute limit. `minimumCapacities` is a map list keyed by a fully
qualified DRA capacity name, has at most 32 entries, and requires a positive
quantity for every value.

Alpha requires exactly one `memory` entry under the source driver's domain. Its
value is derived from the source device's trusted ResourceSlice after kubelet
requires exact agreement with the runtime response. Alpha also requires one
`Exactly` request whose allocation mode is unset or `ExactCount`, whose
effective count is one, and whose `adminAccess` is not true. Across the entire
Pod there must be exactly one DRA allocation result and one checkpointable
device. When `DRAConsumableCapacity` is enabled, that device's
`allowMultipleAllocations` must be false or unset; when it is disabled, DRA
cannot advertise shareable devices through that field. In both cases the DRA
driver must omit `driverStackID` from shared or partitioned devices. Kubelet
rejects `FirstAvailable`, `All`, direct ResourceClaims, PodGroup or other shared
claims, additional device allocations, and ambiguous mappings before
checkpoint.

The field is empty for CPU-only checkpoints and required for a checkpoint that
contains device state. Kubelet writes it in the same status update that records
completion and makes the checkpoint Ready. Users cannot write the status
subresource, and the requirements cannot change after the checkpoint becomes
Ready.

The `PodLevelCheckpointRestoreDevices` feature gate defaults to disabled and has
no effect unless `PodLevelCheckpointRestore` is also enabled. It is registered
in kube-apiserver, kube-controller-manager, and kubelet.

With the device gate disabled:

- checkpointing a Pod for which kubelet has an allocated device-plugin device
  or a prepared DRA device fails the `PodCheckpoint` with `Ready=False` and
  reason `DeviceCheckpointUnsupported` before `CheckpointPod` is called;
- restoring a device-bearing Pod fails before `RestorePod` is called and emits
  a `DeviceRestoreUnsupported` warning event;
- CPU-only checkpoint and restore behavior is identical to KEP-5823;
- the `PodRestoreAuthorization` admission plugin rejects new device-bearing
  restore Pods while preserving requirements already stored in a
  `PodCheckpoint`;
- the ResourceClaim controller refuses to create or reuse a generated claim for
  an already-admitted restore Pod whose checkpoint has device requirements;
  that Pod remains Pending; and
- existing checkpoint data is retained and may be used after re-enablement.

With the gate enabled, kubelet permits the CRI calls for the restricted DRA
shape described above. A Pod with an allocated device-plugin device fails before
the CRI call with `DeviceCheckpointUnsupported` or
`DeviceRestoreUnsupported`. For a device-bearing checkpoint, an empty
`CheckpointPodResponse` fails with reason `DeviceCheckpointUnsupported`. A
malformed, ambiguous, or source-ResourceSlice-inconsistent response fails with
reason `DeviceCheckpointFailed`. Kubelet never records a checkpoint with either
failure as Ready.
A runtime that implements the Pod RPCs but cannot handle the prepared DRA device
must return `FailedPrecondition`. It must not silently produce or restore a
CPU-only checkpoint. `Unimplemented` remains reserved for a missing Pod-level
CRI RPC.

The condition reasons above are stable user-facing classifications, not a new
condition type. A device-bearing `PodCheckpoint` is Ready only when its runtime
archive and complete `deviceRestoreRequirements` have both been recorded. The
restoring Pod continues to use the KEP-5823 `Restoring` condition. Detailed
runtime diagnostics remain in the condition or event message with bounded
length and no raw device identifiers.

### Current implementation evidence

The standalone NVIDIA Snapshot project demonstrates CUDA/CRIU ordering,
immutable artifacts, launch-job state, GPU identity mapping, and restore with
device-plugin and DRA allocations. It is useful implementation evidence, but
Alpha standardizes only the DRA path and the project does not yet implement
this KEP's contract:

- capture targets exactly one container instead of every running container in
  the Pod;
- a privileged node agent invokes CRIU and CUDA helpers outside the KEP-5823
  `CheckpointPod` and `RestorePod` CRI calls;
- restore replaces the process in a running placeholder container instead of
  creating CRI containers in `CREATED` state for kubelet pre-start hooks and
  `StartContainer`;
- GPU identity discovery uses the DRA API, kubelet PodResources, and
  `nvidia-smi`. A CRI runtime cannot assume access to those out-of-band
  interfaces;
- failure after the CUDA transition can terminate the source process, so it
  does not prove the source-running success contract; and
- higher-level workflows may permit a cold start when optional restore is
  unavailable. An explicit `spec.restoreFrom` operation must instead fail
  closed.

As an Alpha prerequisite, a CRI runtime prototype must use the actual Pod RPCs,
consume kubelet-generated restore configs, restore all KEP-5823 containers,
resume the source after success, and demonstrate fail-closed behavior.

### Checkpoint content

`status.checkpointedPodTemplate` records the allocated Pod shape used for the
restore equality check. This KEP adds the bounded
`status.deviceRestoreRequirements` projection described above.

The runtime-owned checkpoint directory contains the complete device record,
including:

- an inventory of every GPU visible to the checkpointed process;
- original runtime-private device identities;
- GPU memory and driver state;
- the checkpoint implementation and format version;
- target-device requirements such as architecture or chip type, device mode,
  minimum memory, driver and checkpoint API constraints, and required device
  features or topology;
- a manifest of restore-visible files with their expected paths and strong
  content digests, optionally supplemented by build IDs; and
- whether the workload uses an unsupported feature.

CRIU records open file descriptors and file-backed memory mappings. Restore must
therefore provide the expected backing content at the recorded path, unless the
runtime supports an explicit, validated remap. The private manifest validates
every GPU-related restore-visible file, including image-provided and host- or
CDI-mounted libraries such as `libcuda.so`. Any generic base-image identity gap
remains governed by the parent KEP's container-filesystem contract.

The archive also contains the same driver stack ID and minimum-capacity values
that the runtime reports to kubelet. This makes the runtime artifact the
canonical source for validating the public projection. Kubernetes does not
parse or version the archive. It stores only the bounded projection and never
copies raw device identities, exact live memory use, file paths, or digests.
For Alpha, the public `memory` requirement is the source GPU's total memory
capacity; it does not expose the workload's live GPU-memory use.

### Checkpoint flow

The flow extends the KEP-5823 kubelet operation:

1. Kubelet resolves the source Pod and applies all parent preconditions.
2. Kubelet checks its allocated device state. If any allocated device is found
   and `PodLevelCheckpointRestoreDevices` is disabled, or if the Pod uses a
   device-plugin allocation, the operation fails before CRI. With the gate
   enabled, kubelet proves from Pod and DRA state that there is exactly one
   Pod-owned generated claim, one supported exact request, one allocation
   result across the Pod, and one non-shareable device. Direct, PodGroup,
   workload-shared, administrative, `All`, and additional allocations fail
   before CRI.
3. Before invoking the expensive operation, kubelet validates the protected
   ResourceClaimTemplate UID annotation and resolves the source allocation
   against the latest complete ResourceSlice generation. The source device must
   publish `driverStackID` and `memory`. When `DRAConsumableCapacity` is enabled,
   `allowMultipleAllocations` must be false or unset. Missing, malformed, or
   ambiguous provenance fails the checkpoint without calling the runtime.
4. Kubelet suspends probes, resolves the sandbox ID, and calls CRI
   `CheckpointPod` with that ID and a bounded deadline. The runtime enumerates
   every container selected by the KEP-5823 Pod-level contract, pauses the
   complete set before capturing any container, keeps the set paused until
   capture finishes, and resumes it before completing the call.
5. The runtime detects a supported GPU workload from its sandbox and container
   state. One integration owner coordinates device transitions; CRIU and an
   external helper must not both issue the CUDA checkpoint transitions.
6. For the NVIDIA implementation, the GPU integration performs the equivalent
   of:
   - `RUNNING -> LOCKED` with a bounded lock;
   - `LOCKED -> CHECKPOINTED`, moving GPU memory to host memory and releasing
     device resources;
   - CPU process-tree checkpoint through CRIU;
   - restoration of the source process's GPU state onto its original GPU; and
   - `LOCKED -> RUNNING` before the kubelet resumes the Pod.
7. Before committing the archive, the runtime records the complete restore-file
   manifest and the canonical scheduling projection. It includes exactly one
   bounded device requirement in `CheckpointPodResponse` for the Alpha shape.
8. The runtime reports success only after CPU and GPU artifacts are complete,
   represent the same consistent point in the workload's execution, the GPU
   checkpoint API reports the source in `RUNNING`, and the original allocation
   is reattached. This confirms the transition, not application health; normal
   probes and end-to-end tests verify that the workload can still use the GPU.
   The result is all-or-nothing; CPU and GPU state need not be captured by one
   hardware operation.
9. On any failure, the runtime removes partial output as required by the CRI
   contract, attempts to resume the source process only when the
   device checkpoint API reports that resuming is valid, and reports the first
   causal error. Kubelet records `DeviceCheckpointFailed` when it can classify
   the error as device-related; otherwise it uses the parent `CheckpointFailed`
   reason.
10. After a successful CRI call, kubelet validates the response, associates its
    single entry with the prevalidated claim, request, and allocation result,
    and adds the driver and protected ResourceClaimTemplate UID from Kubernetes
    state. It re-resolves the latest complete ResourceSlice generation and
    requires exact equality for `driverStackID` and, in Alpha, for the runtime's
    `memory` value and the source device's published total memory capacity. It
    also rechecks that the device is not shareable. A changed or inconsistent
    value fails the checkpoint and removes its artifact.
    Kubelet writes the requirements, completion data, and `Ready=True` in one
    status update. If that update fails, the archive may remain committed, but
    the checkpoint is not Ready; normal interrupted-operation reconciliation
    keeps the result fail-closed.

CPU threads are not necessarily suspended by the GPU driver's lock operation;
the runtime's KEP-5823 all-container pause is still required to make the
Pod-wide image consistent.

The exact ordering between the all-container pause and GPU lock is an
implementation detail that the prerequisite experiment must settle. The
required invariant is that no CPU thread can mutate GPU-visible process state
between freezing GPU state and capturing the CPU checkpoint image.

### Restore flow

Restore reuses the normal scheduling and node admission path:

1. A user creates a new Pod whose `spec.restoreFrom` references a Ready
   `PodCheckpoint`. The Pod spec matches `status.checkpointedPodTemplate` under
   the parent KEP's equality rules.
2. KEP-5823 admission validates the checkpoint and injects affinity to its node.
3. Before creating the generated ResourceClaim, the ResourceClaim controller
   verifies the recorded ResourceClaimTemplate UID. It copies the template and
   adds the checkpoint-derived driver, driver stack ID, and capacity constraints
   to the recorded exact request.
4. The scheduler allocates that claim through normal DRA processing. If no
   advertised device satisfies all original and checkpoint-derived constraints,
   the Pod remains Pending and `RestorePod` is not called.
5. The new Pod receives its own device allocation. Kubelet completes
   `NodePrepareResources`, resolves the checkpoint, rechecks Pod-spec equality,
   and verifies the device feature gate.
6. Kubelet generates a complete restore-time `ContainerConfig` for each
   checkpointed container. Prepared DRA state contributes CDI device names
   exactly as for a normal container start.
7. Kubelet calls `RestorePod` with the opaque checkpoint path, sandbox config,
   runtime handler, restore options, and complete container configs.
8. The runtime creates only the non-executing sandbox and mount environment
   needed to apply the restore-time mounts and CDI edits. It then matches
   containers by name, reads original device identities from the checkpoint,
   resolves restore identities from the applied configuration, and verifies the
   actual GPU, driver stack ID, and every required restore-visible file against
   the private manifest. It constructs a complete device map only after those
   checks pass and before CRIU creates restored tasks.
9. The runtime restores the sandbox and process tree without allowing the
   restored process to execute, restores GPU state onto the new allocation, and
   reports the restored sandbox and container IDs.
10. Kubelet executes the parent KEP's internal pre-start lifecycle hooks and
   `StartContainer` sequence. PostStart is not rerun.

The runtime must never start the image's original command as a fallback. A
transient GPU restore error fails sandbox creation and leaves the Pod in the
normal kubelet backoff path. `FailedPrecondition` indicates a persistent
incompatibility and follows the non-retryable behavior described below.

### DRA allocation path

#### ResourceClaim lifecycle

Alpha supports a Pod-owned claim generated from a `ResourceClaimTemplate`.
PodGroup and other workload-shared claims are rejected. Pod creation therefore
generates a new ResourceClaim for the new Pod UID. The source Pod retains its
claim and device. The scheduler allocates the new claim on the checkpoint node,
and kubelet completes `NodePrepareResources` before it enters the
`restorePodSandbox` path.

The new claim is not copied from the source claim's allocation result. The
ResourceClaim controller copies the current ResourceClaimTemplate only when its
UID matches the UID in the checkpoint. It then appends the system-owned restore
constraints to the recorded request. The source allocation result, generated
claim name, device name, and CDI name are not reused. Kubelet includes the CDI
names reported for the new allocation in the restore container config.

`ResourceClaimTemplate.spec` is immutable while the object exists, but the
object can be deleted and another object can be created with the same name. The
checkpoint therefore records its UID. If the template is absent or its UID has
changed, the controller does not create the claim, emits
`DeviceClaimTemplateChanged`, and leaves the Pod Pending. Once the controller
has created the claim, later deletion of the template does not alter the
immutable claim spec.

The ResourceClaim controller places the template UID annotation on every
template-generated source claim when the device feature is enabled. A
pre-existing source claim without the protected annotation cannot produce a
Ready device checkpoint. The new in-tree admission check rejects the reserved
annotation in template metadata, requires the
`resourceclaims/template-provenance` synthetic authorization on generated-claim
create, and rejects every later change or removal. The controller overwrites any
in-memory copied value before create and validates the annotation when
considering an orphaned source claim. Thus kubelet does not treat ordinary
user-writable metadata as trusted provenance.

The generated claim starts as a copy of the matching template and is
changed only by adding checkpoint constraints to the named request. This
system-owned narrowing does not change the restoring Pod spec or the parent
Pod-template equality check. It cannot remove a selector, lower a capacity
requirement, change a DeviceClass, enable administrative access, or modify
configuration.

`DeviceClass.spec` is mutable and is deliberately resolved under current
cluster policy when the restore claim is allocated. The controller retains the
template's class reference but does not claim that the referenced class still
has its source-time contents. A class change may make the Pod unschedulable or
cause final runtime validation to fail. In this KEP, "only narrow" means that
the controller's synthesized claim never removes or relaxes a field copied from
the current matching template; it does not make `DeviceClass` immutable.

This is a deliberate exception to the current `ResourceClaimTemplateSpec`
contract that generated claim specs are copied unchanged. Its API documentation
will state that a claim generated for a restore Pod is the template spec plus
the controller-owned narrowing constraints defined by this KEP. Claims for
ordinary Pods remain exact copies.

Direct `resourceClaimName` references and PodGroup or other workload-shared
claims are unsupported in Alpha. Such a claim may be reserved for or configured
around more than one Pod and is not guaranteed to be safe for the restored Pod.
Kubelet rejects checkpoint creation for either shape, so users do not receive a
Ready checkpoint that Alpha cannot restore. The rejection is
`DeviceClaimUnsupported`.

Claim cleanup follows normal DRA Pod deletion semantics. A transient
`RestorePod` failure does not delete or unprepare the claim because kubelet may
retry sandbox creation. A persistent incompatibility also retains the claim,
but kubelet suppresses further restore calls after recording the terminal
condition. When the Pod is deleted, normal `NodeUnprepareResources` and claim
cleanup run. All node plugin operations must remain idempotent across kubelet
and plugin restarts.

#### Compatibility selection

The ResourceClaim controller appends one fixed, implementation-owned CEL
selector to the recorded `Exactly` request. The expression requires:

- `device.driver` to equal the recorded DRA driver;
- the `driverStackID` attribute in that driver's domain to exist and equal the
  recorded value; and
- every named capacity to exist and be at least its recorded minimum. Alpha has
one such check for `memory`.

The selector deliberately does not reference `device.allowMultipleAllocations`,
which is absent from the CEL environment when `DRAConsumableCapacity` is
disabled. Instead, a conforming DRA driver publishes `driverStackID` only for
checkpoint-capable, exclusive full devices. When the consumable-capacity gate
is enabled, kubelet also rejects a source device whose
`allowMultipleAllocations` is true. The runtime always rejects a shared or
partitioned prepared target as a final guard against an incorrect
advertisement.

The controller generates the expression from typed, validated fields and safely
quotes each literal. The runtime cannot supply CEL. Existing template selectors
remain in place, so every original selector and the new selector must pass. A
missing driver stack attribute or memory capacity makes a device ineligible
instead of aborting allocation evaluation. Reconciliation is idempotent and
does not append the same selector twice.

If the template already has the maximum number of selectors, a capacity name
cannot be represented safely, or the generated expression would exceed the DRA
CEL limit, kubelet rejects checkpoint creation when it can detect the problem
from the source claim. The ResourceClaim controller repeats those checks and
fails claim generation rather than dropping or weakening a constraint.

During orphan recovery, the controller reuses a claim only when its Pod owner
reference, existing PodResourceClaim annotation, and complete immutable spec
match the claim it would synthesize from the Pod, Ready checkpoint, and current
template. Otherwise it leaves the Pod Pending and reports the mismatch. No
additional user-visible checkpoint digest annotation is needed.

The DRA driver and GPU runtime jointly define the driver stack ID and its
versioning rules. Equivalent supported stacks must produce the same value;
changes that can make a checkpoint unrestorable must produce a different value.
The ID includes every hardware or driver property that requires exact matching;
minimum capacities remain separate so larger compatible devices can match.
The value is an eligibility filter, not proof of restorability. ResourceSlices
describe live node state and may change after allocation, while the protected
archive is the durable record. The runtime therefore repeats the check against
the actual prepared environment.

DRA Device Binding Conditions may keep a Pod in scheduler PreBind until a
device is attached or otherwise ready. They do not replace kubelet
`NodePrepareResources`, runtime checkpoint compatibility validation, or GPU
state restoration.

KEP-5004's DRA extended-resource bridge may provide a migration path for
workloads that currently request device-plugin extended resources. It is not
part of the Alpha contract; Alpha requires the generated ResourceClaim lifecycle
described above.

### GPU identity remapping

The checkpoint must retain every device visible to the GPU process, not merely
the device on which the application submitted work. The restore implementation
constructs a complete old-to-new mapping and includes identity mappings for any
device that does not change.

Alpha restricts the visible set to one full GPU, making the mapping one-to-one.
For the CUDA implementation, the target must have the required driver stack ID
and enough memory, and must pass the private device and file checks. The runtime
or CRIU GPU plugin passes the mapping to the CUDA Driver API restore operation.
Kubernetes never receives or interprets the UUID pair.

The runtime must resolve the new identity from the actual applied allocation,
not from mutable node labels. In CDI mode, this may require resolving the CDI
device name against the node's CDI specification. The implementation must prove
that the identity comes from the same allocation represented by the CRI
container config.

Multi-GPU support is a future extension. It will require a complete mapping for
every visible GPU, topology and peer-access validation, and clear behavior when
only a subset of targets can be allocated.

### Runtime and CRI contract

This KEP adds a bounded response field to the KEP-5823 `CheckpointPod` RPC. It
does not add a new RPC or change `RestorePodRequest`:

```protobuf
message CheckpointPodResponse {
    repeated DeviceRestoreRequirement device_restore_requirements = 1;
}

message DeviceRestoreRequirement {
    string driver_stack_id = 1;
    repeated DeviceCapacityRequirement minimum_capacities = 2;
}

message DeviceCapacityRequirement {
    string name = 1;
    string quantity = 2;
}
```

- `CheckpointPodRequest` identifies the sandbox. The runtime enumerates the
  complete container set required by KEP-5823 and inspects its live device
  configuration.
- checkpoint output is an opaque runtime-owned directory.
- `CheckpointPodResponse.device_restore_requirements` contains one entry per
  supported checkpointed device allocation. An Alpha entry contains a
  `driver_stack_id` string and a bounded list of qualified capacity names and
  quantity strings.
- `RestorePodRequest.container_configs` contains the complete new container
  configuration, including devices and CDI names.
- the runtime must remove partial checkpoint output on failure and must not
  modify the input checkpoint during restore.
- `RestorePodResponse` identifies containers prepared but not executing their
  restored process.

The response does not contain a GPU UUID, CDI name, DRA pool or device name,
claim name, request name, driver name, or CEL expression. Kubelet obtains those
identities from its trusted Pod and DRA state. Kubelet canonicalizes any
unqualified capacity name under the recorded driver domain before storing it.
Before CRI, kubelet must prove there is exactly one DRA allocation result across
the Pod, exactly one exposed checkpointable device, and no device-plugin or
other DRA allocation. The response must then contain exactly one entry, so the
association is unambiguous. Kubelet rejects an empty, duplicate, oversized, or
unmappable response. If those generic invariants cannot be established from
kubelet's Pod and DRA state, Alpha reports `DeviceClaimUnsupported`; it does not
guess from a vendor resource name. Multi-device support must define an explicit
correlation key before relaxing this restriction.

The response uses the same limits as the stored API: a 64-character stack ID,
at most 32 unique capacity names, valid DRA qualified names, and positive
Kubernetes quantities. The Alpha runtime provides one `memory` capacity whose
value must exactly match the source device's published total capacity.
Implementations must update the CRI client, kubelet runtime interfaces,
instrumentation, fakes, and proxies to carry the response instead of discarding
it.

The runtime commits the same values inside the protected archive before it
reports success. Repeated or recovered checkpoint handling must reproduce the
same response. At restore time, normal container configuration and CDI data
identify the new allocation; the private archive supplies the original identity
and exact compatibility requirements.

Runtime error behavior is:

- `Unimplemented` when the Pod-level checkpoint or restore RPC is not
  implemented;
- `FailedPrecondition` when the current checkpoint, workload state, or target
  device allocation cannot satisfy the operation and retrying the same request
  cannot succeed;
- `Unavailable` when device handling or another node dependency is temporarily
  unavailable;
- `InvalidArgument` for malformed runtime options;
- `DeadlineExceeded` or cancellation when the kubelet deadline expires;
- another non-success status for internal or host failures, preserving the
  original causal message.

For a device-bearing restore, kubelet treats `FailedPrecondition` as a
persistent failure and reports `DeviceRestoreIncompatible`. A future structured
CRI detail may provide a more specific reason. Kubelet does not classify errors
by matching runtime error strings.

### Failure and retry semantics

A checkpoint succeeds only when the runtime has completed the CPU and GPU
artifacts for the same consistent point in the workload's execution, committed
the complete restore requirements, and successfully resumed the source GPU
process in the `RUNNING` state. If any step fails, the runtime discards partial
output and reports an error. Kubelet does not inspect the runtime-owned
artifacts or GPU state. It records the checkpoint as Ready only after the CRI
call succeeds and it has validated and persisted the complete public
requirements. CPU and GPU state do not need to be captured simultaneously.

[KEP-5823] and [KEP-2008] assume that a failed CPU-only checkpoint does not
affect the source workload. GPU checkpointing cannot provide the same guarantee
after the runtime begins the GPU checkpoint transition: the driver may enter
an unrecoverable state from which the source process cannot resume.

The runtime owns rollback between GPU state transitions because only it knows
the checkpoint API's transition rules. On a checkpoint error it must inspect
the process state and:

- unlock only from a state for which unlock is valid;
- never attempt a normal transition when the checkpoint API reports a failed
  state;
- remove partial checkpoint output;
- report whether the source resumed successfully in its diagnostic
  message.

If rollback succeeds, the source process is back in `RUNNING` even though the
checkpoint fails. If the GPU driver instead reports an unrecoverable state,
Kubernetes cannot guarantee workload health while the host process remains
alive. There is no Pod phase named `Error`: the source Pod may remain `Running`
at the Kubernetes phase level until the process exits or its application health
checks fail. Kubelet records the checkpoint as `Ready=False` with reason
`DeviceCheckpointFailed`, emits a warning event on the source Pod, and states in
the condition message whether the source resumed successfully or its health must
be verified.

Restore is all-or-nothing at the CRI boundary. On error, no restored process may
be running, the runtime removes any partially created sandbox and containers,
and the checkpoint remains unchanged and reusable.

If no advertised device satisfies the generated claim, the Pod remains Pending
under normal scheduler and DRA retry behavior. No sandbox is created and no
expensive restore is attempted. This condition can clear when a matching device
becomes available or the driver republishes corrected ResourceSlices.

Transient errors follow the normal kubelet sandbox backoff. For a device-bearing
restore, `FailedPrecondition` means that retrying with the same checkpoint, Pod
configuration, and prepared allocation cannot succeed. The runtime must not use
this status for a transient internal or host failure. After runtime cleanup
completes, kubelet sets `Restoring=False` with reason
`DeviceRestoreIncompatible`, emits a warning event when it first records that
condition, and immediately suppresses further calls in memory. Once the
condition is persisted, kubelet does not call `RestorePod` again for that Pod
UID, including after restart. If kubelet fails between the CRI response and the
status update, recovery may attempt restore again. Kubernetes provides
at-least-once execution in that window; the runtime's all-or-nothing cleanup and
idempotency requirements keep it safe, but the KEP does not claim exactly-once
execution. The Pod remains `Pending`; this is a terminal restore outcome for
that Pod instance, not a terminal Pod phase.

Alpha does not replace an allocation in place. The generated claim remains
allocated and prepared until the Pod is deleted, at which point normal
`NodeUnprepareResources` and claim cleanup run. After correcting the eligible
device pool or node stack, the user recreates the Pod, which creates a new Pod
UID and generated ResourceClaim. This avoids steady repeated restore and a loop
that silently cycles scarce devices.

Restart recovery must be tested after each durable side effect:

- after DRA preparation but before `RestorePod`;
- during GPU lock or checkpoint;
- after partial CPU checkpoint output;
- after the runtime creates a restored sandbox but before it returns;
- after GPU restore but before unlock;
- after `FailedPrecondition` but before kubelet records
  `DeviceRestoreIncompatible`, including the documented at-least-once retry;
- after CRI returns but before kubelet records restored IDs.

### Security and privacy

KEP-5823 authorization and path-confinement rules remain authoritative. This
extension adds no user-supplied host path, device identifier, helper binary, or
driver option. Kubelet supplies the checkpoint directory and device
configuration from its own device-manager and DRA state.

GPU memory and driver state are sensitive workload memory. They inherit the
parent KEP's restrictive node-local storage, retention, and restore authorization
requirements. GPU UUIDs, DRA pool and device names, CDI names, exact live memory
use, file paths, build IDs and content digests, and checkpoint-format details
remain in the protected runtime archive. They must not appear in status, events,
ordinary logs, or metric labels.

The public status contains only logical claim and request names, the DRA driver,
the ResourceClaimTemplate UID, the opaque driver stack ID, and the source
device's memory capacity. The driver stack ID is a non-secret equivalence token
that is already published for scheduling. Kubernetes compares it exactly and
must not expose it as an event detail or metric label. Its encoding must not
contain a raw device identifier or a reversible list of file paths or digests.
Some DRA drivers already publish physical identifiers in ResourceSlices under
their existing API contract. This KEP neither relies on those identifiers for
scheduling nor copies them into checkpoint status, events, or metrics.

The container runtime, low-level runtime, CRIU GPU plugin, and device-specific
helper form the node's trusted computing base. Implementations must load helpers
and plugins only from administrator-controlled node configuration. A Pod image
or checkpoint must not select an arbitrary host executable or plugin path. The
runtime validates checkpoint metadata, CDI data, every device mapping, and the
restore file manifest before allowing a restored process to execute. Manifest
parsing is bounded, paths are resolved within the expected container or
administrator-controlled host mounts, and symlink traversal cannot redirect
validation to an unrelated file.

Kubelet validates all runtime-derived lengths and quantities before storing
them. It obtains claim, request, driver, and template identity from its own Pod
and DRA state. The ResourceClaim controller constructs a fixed selector and
quotes values; it never evaluates runtime-provided CEL. A compromised node can
deny its own restore by publishing false requirements, but the generated
constraint cannot remove or relax a field copied from the current matching
template. The current `DeviceClass` remains part of normal administrator policy
at restore time.

This KEP grants no new Kubernetes API access to the runtime or GPU Operator.
DRA API access remains with existing scheduler, kubelet, and DRA components; the
core path must not copy the standalone agent's cluster-wide API lookup into a
container runtime.

GPU checkpointing amplifies the parent KEP's denial-of-service risk because
device memory may be staged in host memory and checkpoint storage. The parent
deadline, concurrency, storage-accounting, and cleanup controls apply to the
combined CPU/GPU operation. GPU-specific work must remain bounded by the CRI
context and stop after cancellation.

### Component responsibilities

| Component | Responsibilities | Must not do |
|---|---|---|
| API server and KEP-5823 admission | Store and validate bounded requirements, enforce the synthetic authorization and immutability of the template UID annotation, authorize restore, validate Pod-template equality, and pin same-node placement | Parse GPU state, file manifests, or driver versions |
| ResourceClaim controller | Stamp and verify template provenance, reject device claim creation while gated off, and add fixed, narrowing constraints to the generated restore claim | Remove or weaken fields copied from the matching template |
| Scheduler | Allocate DRA claims and honor their ordinary selectors | Parse checkpoint archives or run a GPU-specific restore plugin |
| Kubelet | Correlate the CRI response with trusted DRA state, persist status, order allocation/preparation before restore, and surface outcomes | Serialize GPU driver state or expose raw device IDs |
| Container runtime | Produce bounded requirements, coordinate all-or-nothing Pod checkpoint/restore, resolve devices, validate the private manifest, and clean partial state | Allocate Kubernetes resources or mutate claims |
| CRIU GPU plugin | Coordinate the device checkpoint API with process dump/restore and pass the complete device map | Compete with a second GPU transition owner |
| GPU Operator | Provision and validate driver, toolkit/CDI, DRA driver, and node prerequisites | Become a required Kubernetes control-plane dependency |
| DRA driver | Publish `driverStackID` only for checkpoint-capable exclusive full devices, publish their total memory, allocate claims, prepare devices, and report CDI names | Read or modify checkpoint archives |

### Implementation workstreams

The work is split by ownership:

1. **Evidence prototypes**
   - complete a DRA ResourceClaimTemplate restore to a second compatible GPU;
   - trace every allocation, CDI, runtime, CRIU, and CUDA identity handoff;
   - inject failures and record source-process state.
2. **Kubernetes API and ResourceClaim controller**
   - add validation, serialization, and feature-gate handling for
     `status.deviceRestoreRequirements`;
   - add the admission and authorization checks for the reserved template UID
     annotation and reject that key in ResourceClaimTemplate metadata;
   - stamp and validate the annotation on source and orphaned generated claims;
   - watch referenced Ready checkpoints when generating restore claims;
   - refuse device restore claim creation when the gate is disabled;
   - validate the template UID and append fixed, AND-only driver, stack, and
     capacity constraints;
   - update the ResourceClaimTemplate copy contract and validate generated-claim
     checkpoint identity during orphan recovery;
   - preserve normal claim ownership, immutability, and idempotent reconciliation.
3. **Kubernetes kubelet**
   - register and enforce `PodLevelCheckpointRestoreDevices`;
   - reject device-plugin allocations and recognize prepared DRA resources;
   - reject direct, shared, administrative, multi-result, and shareable claim
     shapes and correlate the CRI response with DRA state;
   - validate the source stack ID and capacities against its ResourceSlice;
   - persist complete requirements before recording the checkpoint as Ready;
   - classify supported CRI failures and add bounded events and tests;
   - test allocation and preparation ordering before `RestorePod`.
4. **CRI and conformance**
   - add the bounded `CheckpointPodResponse` requirement fields;
   - propagate the response through CRI clients, kubelet runtime interfaces,
     instrumentation, fakes, and proxy layers;
   - add conformance assertions only for vendor-neutral observable behavior;
   - reject incomplete device responses and any fresh-process fallback.
5. **Runtime and CRIU**
   - implement all-or-nothing CPU/GPU checkpoint and restore completion;
   - store a self-contained device inventory and restore file manifest;
   - produce the bounded scheduling projection from the same canonical record;
   - resolve the restore allocation and pass a complete device map;
   - make cleanup and at-least-once retry safe across restarts.
6. **GPU Operator**
   - publish a supported checkpoint-capable configuration and validator checks;
   - document GPU Operator-managed and preinstalled-driver variants;
   - avoid requiring GFD labels for correctness.
7. **NVIDIA DRA driver**
   - publish a stable `driverStackID` and byte-valued `memory` capacity only for
     checkpoint-capable, exclusive full GPUs;
   - validate full-GPU, non-shareable claim and CDI behavior during restore;
   - test driver/plugin upgrade and restart around prepared claims;
   - test that incompatible driver, library, device, or checkpoint-ABI changes
     produce a different stack ID.
8. **CI and documentation**
   - create a periodic node e2e environment with two compatible GPUs;
   - publish the tested component matrix and troubleshooting runbook.

### Open Questions

The following must be resolved before targeting Alpha:

1. Is one GPU-using process enforceable by the runtime, and how is a violation
   distinguished from unsupported CUDA IPC?
2. What exact ordering between the KEP-5823 all-container pause and GPU lock
   provides a consistent image for current CRIU GPU plugins?
3. Can a runtime report in a structured form that the source process did not
   resume successfully, or is a CRI response extension needed?
4. What canonical input set and version prefix does each implementation use for
   `driverStackID`, and which upgrades can intentionally preserve it?
5. Which exact GPU Operator, driver, toolkit, DRA-driver, CRIU, containerd, and
   CRI-O versions comprise the Alpha support matrix?

### Test Plan

[x] I/we understand the owners of the involved components may require updates
to existing tests before implementation is accepted.

#### Prerequisite testing updates

Before targeting Alpha:

- KEP-5823 Pod checkpoint and restore node e2e tests must run against at least
  one real CRI implementation.
- A dedicated real-GPU periodic environment must provide at least two compatible
  full GPUs on one Linux node.
- The environment must archive kubelet, runtime, CRIU GPU plugin, DRA driver,
  GPU Operator, and GPU driver logs on failure.
- The DRA prototype tests described in Implementation Workstreams must pass and
  publish an exact version matrix.

Unit coverage baselines will be recorded when implementation PRs are opened.

#### Unit tests

Kubelet and kuberuntime tests cover:

- both feature gates enabled, parent enabled/device disabled, and both disabled;
- CPU-only Pods remain unaffected by the device gate;
- allocated device-plugin devices are rejected before CRI;
- direct, PodGroup, shared, administrative, `All`, multi-result, additional,
  and shareable allocation shapes are rejected before CRI;
- missing or invalid protected template provenance is rejected before CRI;
- the single CRI requirement is associated with the logical claim and exact
  request using kubelet DRA state;
- empty, duplicate, oversized, or unmappable CRI requirements fail the
  checkpoint;
- the runtime stack ID and memory exactly match the source device's current
  complete ResourceSlice generation;
- `Ready=True` is written only in the same status update as the validated
  device restore requirement;
- no CRI call occurs for an unsupported device-bearing operation;
- DRA `PrepareDynamicResources` precedes `RestorePod`;
- DRA CDI names are preserved in the restore config;
- `Unimplemented`, `FailedPrecondition`, deadline, and unknown errors are
  classified without broad text matching;
- retry backoff does not duplicate state or produce unbounded events, in-memory
  suppression begins immediately after `FailedPrecondition`, and the persisted
  terminal condition suppresses calls after kubelet restart;
- Pod deletion after restore failure unprepares DRA claims normally;
- raw device, allocation, and CDI identities and the file manifest are absent
  from API status, events, and metric labels.

Runtime unit and integration tests cover the GPU checkpoint transitions,
complete device inventory, one-to-one mapping, malformed or incomplete map
rejection, bounded requirement generation, private/public requirement
consistency, restore-file manifest validation, partial artifact cleanup, and
safe repeated invocation in the documented at-least-once failure window.

DRA driver tests verify that equivalent devices and stacks publish the same
driver stack ID across plugin restart, incompatible hardware, driver-library,
or checkpoint-ABI changes publish a different ID, unsupported devices omit the
attribute, memory is published as a byte-valued total capacity, and shareable or
partitioned devices omit the stack attribute.

API and ResourceClaim controller unit tests cover:

- protobuf and JSON round trips, generated deepcopy code, validation bounds,
  feature-gate handling, and map-list uniqueness for
  `deviceRestoreRequirements`;
- status-subresource authorization;
- requirement immutability after `Ready=True`;
- matching and changed ResourceClaimTemplate UIDs;
- rejection of the reserved annotation in template metadata, synthetic
  authorization on generated-claim create, immutability, and orphan validation;
- preserving every field copied from the current template while adding the
  driver, driver stack ID, and minimum-capacity checks;
- missing attributes and capacities, safe CEL literal construction, and
  idempotent reconciliation; and
- a restore Pod remaining Pending without a generated claim when its template
  identity is invalid or the controller feature gate is disabled.

#### Integration tests

API-server tests verify that KEP-5823 authorization, affinity injection, and
Pod-template equality continue to work for Pods with ResourceClaimTemplates.
They also verify validation, feature-gate behavior, storage round trips, and
preservation of existing device requirements while the gate is disabled.

ResourceClaim controller integration tests verify that a restore claim is an
unchanged copy of the matching template except for the single system-owned,
narrowing selector, and that the scheduler observes the resulting normal DRA
requirements. A missing or changed template UID, selector-limit overflow, or
orphaned claim whose owner, PodResourceClaim annotation, or complete spec does
not match must prevent claim reuse or creation. A `DeviceClass` change uses the
current class as normal DRA allocation does; checkpoint constraints remain
additive and final runtime validation remains fail-closed.

Kubelet orchestration tests use a fake device manager for the unsupported-path
check and fake DRA managers and CRI runtimes to verify ordering and restart
convergence. Fakes do not count as GPU restore validation.

CRI conformance tests should assert:

- device-bearing checkpoint cannot succeed after omitting required runtime
  device state;
- every supported checkpointed allocation has a complete bounded response
  entry, and an ambiguous association is rejected;
- restore receives complete container configs and either succeeds or returns a
  defined gRPC failure;
- partial checkpoint output and partially restored runtime objects are cleaned;
- restore never falls back to a fresh process.

The precise conformance mechanism remains unresolved because the generic CRI
test environment may not have a checkpointable device.

#### e2e tests

Real-hardware node e2e covers:

- GPU Operator and NVIDIA DRA driver: checkpoint a running CUDA counter and
  restore it to a second compatible full GPU using a ResourceClaimTemplate
  while the source remains running;
- validate computation state, not merely Pod phase or CRIU exit status;
- compatible devices publish the same driver stack ID and are selected through
  the generated claim;
- an incompatible driver stack or insufficient memory leaves the Pod Pending
  and causes zero `RestorePod` calls;
- deleting and recreating the ResourceClaimTemplate under the same name blocks
  claim generation because its UID changed;
- a shareable advertised device is excluded by the generated claim;
- a driver stack change after allocation is rejected by the runtime before any
  restored process executes and remains terminal across kubelet restart;
- a kubelet crash after runtime `FailedPrecondition` but before the Pod condition
  is stored may repeat the call but leaves no running restored process; after
  the condition is stored, restart causes no further call;
- changing `libcuda.so` or another required restore-visible file is rejected by
  private-manifest validation;
- no second GPU available leaves the restored Pod unschedulable rather than
  stealing the source allocation;
- unsupported UVM and shared-allocation workloads fail the checkpoint;
- kubelet, runtime, and DRA-driver restart cases follow the
  documented recovery behavior;
- timeout and cancellation leave no Ready checkpoint or partial runtime object;
- feature-gate disablement rejects GPU operations and leaves CPU-only operations
  working;
- device-plugin GPU Pods are rejected before a checkpoint or restore CRI call;
- containerd and CRI-O are exercised as their KEP-5823 implementations become
  available.

### Graduation Criteria

#### Alpha

- KEP-5823 is available in the target branch and implemented by at least one
  released CRI runtime.
- The device feature gate is implemented and disabled by default.
- `PodCheckpoint.status.deviceRestoreRequirements` and the bounded
  `CheckpointPodResponse` fields are implemented with validation and skew
  tests.
- Device-plugin allocations are rejected before the checkpoint or restore CRI
  call.
- Direct, PodGroup, shared, administrative, multi-result, and shareable DRA
  allocations are rejected before the checkpoint CRI call.
- NVIDIA full-GPU checkpoint and same-node restore pass through the DRA
  ResourceClaimTemplate path.
- Admission protects the template UID provenance, and the ResourceClaim
  controller validates it and adds only the fixed driver, driver stack ID,
  and minimum-capacity constraints.
- The NVIDIA DRA driver publishes `driverStackID` and `memory` on every
  supported non-shareable full GPU.
- A restored Pod uses a different compatible GPU while the source Pod continues
  running.
- All-or-nothing CPU/GPU checkpoint completion, restore-file validation,
  runtime cleanup, the at-least-once crash window, and persistent
  late-incompatibility behavior are documented and tested.
- When no compatible device is advertised, the restore Pod remains Pending
  without a CRI restore attempt.
- Continuous real-GPU CI runs the core happy paths and failure cases.
- Security review and Alpha PRR are complete.

#### Beta

- At least two CRI runtime implementations support the device-bearing Pod RPCs.
- Multi-GPU, single-process restore is supported with complete mapping and
  topology validation, or explicitly removed from this KEP's graduation path.
- A second GPU vendor implements the Kubernetes contract, or an alternative
  vendor-neutral conformance implementation demonstrates portability.
- Driver stack ID stability and portability rules are validated across the
  supported upgrade matrix.
- Metrics, runbooks, upgrade/downgrade tests, and failure-injection coverage are
  complete.
- All known security, monitoring, functional, and testing gaps are resolved.

#### GA

- The feature has remained Beta for at least two releases.
- Multiple production users report successful operation across the supported
  runtime and DRA-driver matrix.
- No unresolved data-loss, source-corruption, device-leak, or privilege issues
  remain.
- Any API or CRI additions have completed their own required graduation and
  compatibility review.

### Upgrade / Downgrade Strategy

Upgrading does not change existing Pods or CPU-only checkpoints. Cluster
operators first upgrade the runtime, GPU checkpoint implementation, GPU node
stack, and DRA driver. They wait until the ResourceSlices contain a valid
`driverStackID` and `memory` only on eligible exclusive full devices, then
upgrade every API server and controller manager. No older controller manager
may remain able to lead before the control-plane gates are enabled. Operators
enable the gate on selected kubelets last.

For rollback, operators first disable new device operations at admission and on
kubelets. Existing source and restored Pods continue running. The API server
preserves stored `deviceRestoreRequirements`, and the ResourceClaim controller
does not create an unconstrained restore claim while its gate is disabled.
Operators keep the new API and controller code until affected restore Pods have
been removed, then roll back the node stack if retained checkpoints no longer
need it.

A runtime or driver downgrade may make an existing GPU checkpoint unreadable.
The archive is runtime-defined, so Kubernetes cannot convert it. Operators must
retain the runtime/driver compatibility matrix associated with retained
checkpoints and validate downgrade before changing a node. The runtime rejects
an incompatible archive; it must not attempt a fresh start.

### Version Skew Strategy

- **API server older than kubelet:** this skew is unsupported for the device
  gate. A server that drops the new field may briefly persist `Ready=True`
  without requirements; checking the status response is detection, not a
  transactional guarantee. If misconfigured this way, kubelet attempts a
  corrective `Ready=False` update and removes the artifact. The supported
  control-plane-first rollout keeps the kubelet gate disabled until every API
  server stores and returns the field.
- **API server rollback:** an API server version that does not know the status
  field may drop it during an update. Rolling back that far is unsupported while
  device checkpoints are retained for restore. Admission rejects a GPU restore
  whenever the referenced checkpoint lacks complete requirements.
- **Controller manager older than the API server:** an older ResourceClaim
  controller may copy the template without the checkpoint constraints. The
  runtime's final check prevents an unsafe restore, but the rollout must not
  enable device restore until every controller manager that can become leader
  supports claim narrowing. Disabling the gate on a new controller makes it
  refuse claim creation; it never falls back to an unconstrained copy.
- **Kubelet newer than runtime:** `Unimplemented` is reported when the Pod RPC
  is missing. A runtime that provides no device requirements cannot produce a
  Ready GPU checkpoint. `FailedPrecondition` reports an unsupported workload or
  prepared target.
- **Runtime newer than kubelet:** additive response fields may be ignored by an
  older kubelet. Such a kubelet cannot have the device gate enabled, and a
  restore of any GPU checkpoint without stored requirements is rejected.
- **Mixed kubelet fleet:** KEP-5823 pins restore to the checkpoint node. That
  node must still have the device gate and compatible runtime enabled. A node
  capability advertisement may improve UX in a future cross-node stage but is
  not required for same-node Alpha.
- **DRA driver skew:** a driver that does not publish `driverStackID` or
  `memory` has no Alpha-compatible device. A conforming driver omits the stack
  attribute from a shareable or partitioned device, so it cannot match. A
  changed ID prevents new allocation. `NodePrepareResources` failure still
  prevents `RestorePod`.
- **GPU driver or CRIU plugin skew:** runtime validation is authoritative. Exact
  files and archive formats remain an operator-managed support matrix. The
  runtime rejects any mismatch before a restored process executes.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

The `PodLevelCheckpointRestoreDevices` feature gate enables the device extension
in kube-apiserver, kube-controller-manager, and kubelet.
`PodLevelCheckpointRestore` must also be enabled. The documented rollout uses
rolling component restarts and requires no node reprovisioning.

###### Does enabling the feature change any default behavior?

No. The gate defaults to disabled. When enabled, only explicit Pod checkpoint
or restore operations involving supported DRA devices can invoke the GPU-aware
CRI path. Device-plugin allocations continue to fail before CRI.

###### Can the feature be disabled once it has been enabled?

Yes. Running workloads are not changed. With the gate disabled, admission and
the ResourceClaim controller fail closed for new device restores, and kubelet
rejects new device CRI operations. CPU-only KEP-5823 operations continue. The
API server preserves device requirements that were already stored.

Disabling the gate does not delete archives or `PodCheckpoint` objects. Cluster
operators remain responsible for normal checkpoint retention and sensitive-data
handling.

###### What happens if we reenable the feature after rollback?

New operations resume. Existing checkpoints can be restored only if the current
runtime, plugin, and driver can read them and a compatible device is available.

###### Are there tests for feature enablement/disablement?

Yes. API-server, ResourceClaim controller, kubelet, and node e2e tests cover the
gate enabled, disabled, and disable-then-reenable cases, including coexistence
with the parent gate and preservation of stored requirements.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A node may have the kubelet gate enabled before its runtime or GPU checkpoint
plugin is ready. A DRA driver may also publish no stack ID, or an older
ResourceClaim controller may omit the generated selector. The rollout order and
gate checks prevent these combinations from serving device restore. Explicit
operations fail closed, while ordinary running GPU Pods are unchanged. A
checkpoint attempt itself may affect its source process if the GPU driver enters
an unrecoverable state; this is an operation risk rather than a passive rollout
side effect.

DRA plugin or GPU driver rollout can disrupt device allocation independently of
this KEP. Operators must follow their vendor's drain and upgrade process. The
KEP does not bypass DRA preparation or GPU Operator upgrade safety.

###### What specific metrics should inform a rollback?

- a sustained increase in device-bearing checkpoint or restore failures;
- device preparation failures;
- runtime operation errors for `checkpoint_pod` or `restore_pod`;
- GPU process health failures following checkpoint attempts;
- leaked prepared claims, sandboxes, or GPU allocations;
- node memory or disk pressure correlated with GPU checkpoint operations.

###### Were upgrade and rollback tested?

Not yet. Before Beta, tests will cover gate enable/disable/re-enable, kubelet
restart, runtime restart, DRA driver restart, and supported GPU driver/runtime
upgrade and downgrade combinations.

###### Is the rollout accompanied by deprecations or removals?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The existing kubelet operation counters record Pod checkpoint and restore
attempts. Implementations add a bounded `device_allocation` label with values
`none`, `dra`, or `unsupported` if metrics review approves the added series. No
device ID, Pod ID, claim ID, GPU model, driver version, stack ID, or capacity is
a metric label.

An incompatibility filtered during scheduling does not increment a kubelet
restore counter because `RestorePod` is never called. Operators identify that
case through the Pod's scheduling condition and normal ResourceClaim and
scheduler events.

###### How can someone using this feature know that it is working for their instance?

- `PodCheckpoint.status.conditions[type=Ready]` becomes `True` only after CPU
  and GPU state are complete, the source has resumed, and the complete bounded
  requirements are stored.
- A failed checkpoint uses `DeviceCheckpointUnsupported`,
  `DeviceClaimUnsupported`, or `DeviceCheckpointFailed` where the error is
  classifiable.
- The restoring Pod uses the parent `Restoring` condition and becomes Running
  only after the restored GPU process is started.
- Warning events `DeviceRestoreUnsupported` and
  `DeviceRestoreIncompatible` distinguish persistent device failures.
- A restore Pod with no matching advertised GPU remains Pending with normal DRA
  scheduling diagnostics and no restore attempt.

Application-level validation remains necessary to prove resumed GPU
computation. Pod phase alone cannot prove that CUDA state is usable.

###### What are the reasonable SLOs for the enhancement?

Alpha defines no latency SLO. Every operation must produce an unambiguous
success or failure, never a CPU-only success for a GPU workload. A successful
checkpoint must leave the source GPU process running, and a failed restore must
leave no running restored process.

Beta will define latency and failure-rate objectives from real workload and
checkpoint-size data.

###### What SLIs can an operator use to determine feature health?

- `kubelet_pod_checkpoint_operations_total{result,device_allocation}`
- `kubelet_pod_checkpoint_duration_seconds{device_allocation}`
- `kubelet_pod_checkpoint_size_bytes`
- `kubelet_pod_restore_operations_total{result,device_allocation}`
- `kubelet_pod_restore_duration_seconds{device_allocation}`
- `kubelet_runtime_operations_errors_total{operation_type="checkpoint_pod|restore_pod"}`
- existing DRA operation duration and error metrics

The additional label remains provisional pending metrics review.

###### Are there missing metrics that would be useful?

- GPU lock, device-memory transfer, CRIU dump, GPU restore, and unlock phase
  durations are runtime-specific and not visible to kubelet.
- The amount of GPU memory staged into host memory would help capacity planning
  but may require vendor runtime instrumentation.
- A count of sources left in an unrecoverable GPU state requires a structured
  runtime outcome that is not part of the device-requirement response.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- **KEP-5823 implementation**
  - Provides `PodCheckpoint`, `spec.restoreFrom`, kubelet lifecycle, and CRI Pod
    checkpoint/restore RPCs.
  - If unavailable, this enhancement cannot be invoked.
- **Compatible CRI runtime and low-level runtime**
  - Coordinate Pod sandbox and container checkpoint/restore.
  - Return the bounded scheduling requirements from `CheckpointPod` and keep the
    same values in the protected archive.
  - Outage or incompatibility causes the explicit operation to fail; existing
    running Pods are not passively affected.
- **CRIU and a GPU-aware plugin, or an equivalent runtime mechanism**
  - Captures CPU process and GPU driver state under one runtime operation.
  - Records and validates the restore-visible file manifest, including injected
    GPU libraries.
  - Missing support must report failure rather than omit GPU state.
- **GPU driver checkpoint implementation**
  - For NVIDIA, the Linux CUDA Driver API provides process lock, checkpoint,
    restore, unlock, and device mapping. The exact supported driver is probed
    and documented by the runtime.
- **DRA driver**
  - Publishes the same stable `driverStackID` used by the runtime and the named
    capacities required for selection, allocates the restore GPU, and prepares
    it on the node. A DRA outage prevents claim allocation or preparation.
- **API server and ResourceClaim controller**
  - Store and validate the bounded requirements, protect the source-template
    UID annotation, and add narrowing constraints to restore claims.
  - If unavailable or gate-disabled, no unconstrained device restore claim may
    be created.
- **CDI-capable container runtime**
  - Resolves the DRA-allocated GPU into the container.
- **GPU Operator, optional**
  - Automates the NVIDIA node stack and validator. Kubernetes correctness does
    not depend on its controller being continuously available after the node
    components are installed.

### Scalability

###### Will enabling or using this feature result in new API calls?

The ResourceClaim controller adds `PodCheckpoint` objects to its informer cache
and looks up the referenced Ready checkpoint when it generates a restore claim.
For each device checkpoint, kubelet reads the complete ResourceSlice generation
that contains the allocated source device so it can compare the runtime summary
with the driver's published values. DRA Pods still create one generated
ResourceClaim per template reference. No component periodically polls
checkpoint compatibility.

###### Will this feature introduce new API types?

No.

###### Will it result in new cloud-provider calls?

No.

###### Will it increase the size or count of existing API objects?

`PodCheckpoint.status` gains at most 32 bounded requirement entries; Alpha uses
one. Each entry contains short logical names, one UID, one stack ID, and at most
32 named quantities. A generated restore `ResourceClaim.spec` gains one bounded
CEL selector, and each supported device in a `ResourceSlice` gains one
`driverStackID` attribute if it did not already publish it. Object counts do not
increase beyond the normal generated claim.

###### Will it increase time taken by operations covered by existing SLOs?

Pod startup is delayed by checkpoint reading, CPU process restore, GPU memory
transfer, compatibility validation, and driver initialization. This path is used
only when `spec.restoreFrom` is set and is not included in Alpha latency SLOs.
Normal Pod startup is unchanged.

###### Will it cause a non-negligible increase in resource usage?

Yes, during explicit operations:

- GPU memory is staged into host memory and checkpoint storage;
- CRIU and the runtime consume CPU and disk I/O;
- source and restored Pods concurrently consume GPUs after successful restore;
- DRA maintains one claim per Pod as usual.

Kubelet and API-server memory growth is bounded because they retain only the
small public projection, never GPU state or the file manifest. Runtime memory
and disk requirements can be proportional to allocated GPU memory and the
number of restore-visible files.

###### Can it exhaust node resources?

Yes. Host memory, checkpoint disk space, I/O bandwidth, GPU capacity, and
operation worker capacity can be exhausted. The implementation reuses the
parent checkpoint timeout and concurrency bounds, performs free-space and
allocation checks where available, cleans partial output, and relies on normal
scheduling to reserve the restore GPU. Before Beta, the runtime must document
and test its upper bounds for concurrent GPU checkpoints.

### Troubleshooting

###### How does the feature react if the API server or etcd is unavailable?

KEP-5823 behavior applies. A new `PodCheckpoint` or restore Pod cannot be
created. If the API server becomes unavailable during checkpoint, kubelet may
complete the node-local runtime operation but cannot finalize status until API
connectivity returns. Runtime cleanup and source resumption must not depend on a
successful status write.

###### What are other known failure modes?

- **Device feature disabled or runtime support absent**
  - Detection: `DeviceCheckpointUnsupported` or
    `DeviceRestoreUnsupported`; runtime-operation error metrics.
  - Mitigation: enable the gate only after validating the node stack; do not
    retry by changing opaque runtime options.
  - Testing: unit and node e2e gate/runtime-capability cases.
- **No second compatible GPU**
  - Detection: the Pod remains Pending with scheduler or DRA diagnostics; no
    `RestorePod` call occurs.
  - Mitigation: use a node with at least two compatible GPUs or wait for future
    source-stop/cross-node support. A newly available matching ResourceSlice can
    make the Pod schedulable without recreating it.
  - Testing: real-hardware capacity and incompatibility cases.
- **ResourceClaimTemplate was deleted and recreated**
  - Detection: `DeviceClaimTemplateChanged`; no generated claim or restore call.
  - Mitigation: retain the original template for the checkpoint lifetime or
    create a new checkpoint from a Pod using the replacement template.
  - Testing: delete-and-recreate integration and e2e cases.
- **DRA preparation fails**
  - Detection: `FailedPrepareDynamicResources` event and DRA metrics; no
    `RestorePod` call is issued.
  - Mitigation: repair or restart the DRA node plugin; delete the Pod to trigger
    normal unprepare and claim cleanup if necessary.
  - Testing: fake-manager ordering test and real driver restart test.
- **GPU checkpoint enters an unrecoverable source state**
  - Detection: `DeviceCheckpointFailed`, application health failures, and
    runtime diagnostics reporting the GPU checkpoint API's failed state.
  - Mitigation: stop checkpoint attempts, preserve diagnostics, and restart the
    workload through its owning controller if it cannot recover.
  - Testing: failure injection where the driver/runtime permits it.
- **Partial checkpoint or restored sandbox remains after restart**
  - Detection: disk growth, leaked sandbox/device allocation, runtime errors,
    or a checkpoint stuck in progress.
  - Mitigation: runtime startup recovery removes incomplete artifacts; kubelet
    KEP-5823 startup reconciliation marks interrupted operations failed.
  - Testing: restart after each durable side effect.
- **Checkpoint format, driver stack, file, or runtime mismatch**
  - Detection: a driver stack mismatch normally remains unschedulable. Drift
    after allocation or a private file or archive mismatch produces one
    `FailedPrecondition` restore failure and a bounded diagnostic naming the
    component category without device IDs.
  - Mitigation: restore the tested node stack version or recreate a checkpoint
    under the new stack.
  - Testing: supported upgrade/downgrade matrix before Beta.
- **Unsupported UVM, sharing, or IPC state**
  - Detection: checkpoint fails before Ready; source health is rechecked.
  - Mitigation: use a supported workload configuration. CUDA IPC is deferred
    until job-scoped launch and ordered peer transitions are integrated.
  - Testing: explicit unsupported-workload node e2e.

###### What steps should be taken if SLOs are not being met?

1. Separate scheduling or DRA preparation delay from runtime restore duration.
2. Inspect kubelet operation and DRA metrics without enabling high-cardinality
   labels.
3. Check the Pod and `PodCheckpoint` conditions and first warning event.
4. Correlate kubelet, runtime, CRIU plugin, and GPU driver logs using the Pod UID
   and operation time; do not publish GPU UUIDs in shared telemetry.
5. Verify the node's tested version matrix, host memory, disk space, and target
   device compatibility.
6. Stop new device checkpoint operations or disable the device gate if source
   health failures or resource exhaustion are increasing.

## Drawbacks

- The Alpha same-node, source-running lifecycle requires spare compatible GPU
  capacity and therefore does not yet provide resource-reclaim migration.
- Existing device-plugin workloads must migrate to DRA before using this Alpha.
- Runtime-defined archives make long-term compatibility an implementation
  concern and can tie retained checkpoints to a node-stack matrix.
- GPU checkpointing can consume host memory and disk comparable to device memory
  and can temporarily pause latency-sensitive workloads.
- An unrecoverable GPU checkpoint error can leave the source process unhealthy
  without an immediate Kubernetes phase transition.
- Real-GPU continuous testing is materially more expensive than ordinary node
  e2e coverage.

## Alternatives

**Extend KEP-5823 instead of creating a follow-on KEP.** Rejected because
KEP-5823 deliberately isolates the base Pod API and runtime lifecycle from
device claim, allocation, compatibility, and driver-state questions. A separate
KEP allows independent feature gating and review by WG Device Management.

**Add GPU model, driver version, UUID, or file-manifest fields to
`PodCheckpoint.status`.** Rejected. Model and version strings do not fully
describe restore compatibility, UUIDs expose node inventory, and paths or file
digests expose details of the node software stack. The selected API stores only
an opaque, bounded driver stack ID and named minimum capacities. The complete
record remains in the protected runtime archive.

**Match only the GPU driver version.** Rejected. CRIU may need the exact content
of file-backed mappings and open files, including `libcuda.so`, and the same
version string does not prove identical libraries, checkpoint ABI, device mode,
or chip compatibility. `driverStackID` supplies the scheduling equivalence token
and the runtime verifies the private file manifest.

**Let the runtime provide a DRA CEL expression.** Rejected. Arbitrary CEL would
make validation, authorization, and forward compatibility depend on runtime
text. The selected design accepts typed, bounded values and has the
ResourceClaim controller construct one fixed, safely quoted selector that can
only narrow the template.

**Validate compatibility only after allocation.** Rejected. A persistent
mismatch would retain a scarce allocation and could invoke an expensive restore
before kubelet records a terminal failure. Checkpoint-derived DRA constraints
normally keep such a Pod Pending without a runtime call; final runtime
validation remains necessary for drift and private requirements.

**Use a custom scheduler plugin that reads `PodCheckpoint`.** Rejected. DRA
already provides structured device matching, the base Alpha is same-node, and a
GPU-specific scheduler data path would duplicate allocation logic and couple
the scheduler to runtime-private checkpoint formats.

**Use GPU Feature Discovery labels as the correctness mechanism.** Rejected.
Labels describe a node rather than the selected device, are mutable, may be
stale, and are not a driver guarantee. They remain valid operator hints.

**Reuse the source GPU or ResourceClaim.** Rejected for Alpha because the source
Pod remains running and owns its exclusive allocation. Transfer requires source
stop/delete semantics and controller coordination that KEP-5823 defers.

**Support device-plugin allocations in Alpha.** Deferred. Device-plugin extended
resources expose integer capacity rather than structured device attributes, so
an incompatible GPU may be discovered only after Pod binding and allocation.
Kubelet cannot swap that allocation, and supporting the path would add another
lifecycle and real-hardware test matrix before the DRA contract is proven.
KEP-5004's DRA extended-resource bridge may provide a migration path for
existing workloads.

**Support only the device plugin.** Rejected because DRA is the Kubernetes API
for structured device selection and provides the compatibility-aware placement
used by this KEP.

**Pass old and new GPU UUIDs through opaque checkpoint/restore options.**
Rejected. Options are user-supplied runtime knobs, not a trusted transport for
automatically discovered allocation identity. The runtime already owns the old
inventory and receives the new allocation in container configs.

**Add old and new device identities to CRI.** Rejected for Alpha. The new
`CheckpointPodResponse` carries requirements, not raw identities. The runtime
owns the source inventory, kubelet derives logical claim information from DRA
state, and the normal restore-time container configs identify the prepared
target. Multi-device support must revisit correlation without exposing UUIDs.

**Implement restore through a GPU Operator controller or admission webhook.**
Rejected as the core mechanism. GPU Operator should provision and validate the
node stack; Kubernetes owns Pod allocation and kubelet/CRI lifecycle. An
out-of-tree controller may provide higher-level workflows without becoming a
correctness dependency.

## Infrastructure Needed

- A periodic Linux node e2e job with at least two compatible full GPUs on one
  node.
- An NVIDIA DRA-driver allocation test variant using a ResourceClaimTemplate.
- Artifact collection for Kubernetes, runtime, CRIU, GPU Operator, DRA-driver,
  and GPU driver logs.

## References

- [KEP-5823: Pod-level Checkpoint/Restore]
- [KEP-2008: Forensic Container Checkpointing]
- [KEP-4381: DRA Structured Parameters]
- [KEP-4817: ResourceClaim Device Status]
- [KEP-5004: DRA Extended Resource]
- [KEP-5007: DRA Device Binding Conditions]
- [KEP-4680: Resource Health Status]
- [NVIDIA GPU Operator]
- [GPU Operator CDI and NRI support]
- [GPU Operator with the NVIDIA DRA driver]
- [DRA Driver for NVIDIA GPUs]
- [CUDA Driver API checkpointing]

[KEP-5823: Pod-level Checkpoint/Restore]: /keps/sig-node/5823-pod-level-checkpoint-restore/README.md
[KEP-2008]: /keps/sig-node/2008-forensic-container-checkpointing/README.md
[KEP-2008: Forensic Container Checkpointing]: /keps/sig-node/2008-forensic-container-checkpointing/README.md
[KEP-4381: DRA Structured Parameters]: /keps/sig-node/4381-dra-structured-parameters/README.md
[KEP-4817: ResourceClaim Device Status]: /keps/sig-node/4817-resource-claim-device-status/README.md
[KEP-5004: DRA Extended Resource]: /keps/sig-scheduling/5004-dra-extended-resource/README.md
[KEP-5007: DRA Device Binding Conditions]: /keps/sig-scheduling/5007-device-attach-before-pod-scheduled/README.md
[KEP-4680: Resource Health Status]: /keps/sig-node/4680-add-resource-health-to-pod-status/README.md
[NVIDIA GPU Operator]: https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/latest/
[GPU Operator CDI and NRI support]: https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/latest/cdi.html
[GPU Operator with the NVIDIA DRA driver]: https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/latest/dra-intro-install.html
[DRA Driver for NVIDIA GPUs]: https://dra-driver-nvidia-gpu.sigs.k8s.io/docs/
[CUDA Driver API checkpointing]: https://docs.nvidia.com/cuda/cuda-driver-api/group__CUDA__CHECKPOINT.html
