# KEP-5823: Pod-Level Checkpoint/Restore

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
  - [Targeted use cases](#targeted-use-cases)
    - [Accelerating startup of applications with long initialization times](#accelerating-startup-of-applications-with-long-initialization-times)
    - [Enabling fault-tolerance for long-running workloads](#enabling-fault-tolerance-for-long-running-workloads)
    - [Pod migration across nodes for load balancing and maintenance](#pod-migration-across-nodes-for-load-balancing-and-maintenance)
- [Proposal](#proposal)
  - [Implementation](#implementation)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [CRI API Extensions](#cri-api-extensions)
    - [CheckpointPod](#checkpointpod)
    - [RestorePod](#restorepod)
  - [Kubelet API Endpoints](#kubelet-api-endpoints)
    - [Checkpoint Endpoint](#checkpoint-endpoint)
    - [Restore Endpoint](#restore-endpoint)
  - [PodCheckpoint Objects](#podcheckpoint-objects)
    - [PodCheckpoint](#podcheckpoint)
    - [Pod-Snapshot-Controller](#pod-snapshot-controller)
  - [Restore Mechanism](#restore-mechanism)
  - [Checkpoint Content](#checkpoint-content)
    - [Pod Specification and Metadata](#pod-specification-and-metadata)
    - [Container Runtime State](#container-runtime-state)
    - [Shared Pod Resources](#shared-pod-resources)
  - [Pod Lifecycle](#pod-lifecycle)
  - [TCP Connection Handling](#tcp-connection-handling)
  - [Security Implications](#security-implications)
    - [Privilege model](#privilege-model)
    - [Sensitive memory contents](#sensitive-memory-contents)
    - [Denial of service via excessive checkpointing](#denial-of-service-via-excessive-checkpointing)
    - [automountServiceAccountToken on restore](#automountserviceaccounttoken-on-restore)
    - [Path traversal protection](#path-traversal-protection)
    - [Status and spec separation](#status-and-spec-separation)
  - [Test Plan](#test-plan)
      - [Prerequisite testing](#prerequisite-testing)
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
  - [Rejected Approaches](#rejected-approaches)
- [Prior Art](#prior-art)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/website]: https://git.k8s.io/website
[kubelet checkpoint]: https://kubernetes.io/docs/reference/node/kubelet-checkpoint-api
[OCI image annotations]: https://kubernetes.io/blog/2022/12/05/forensic-container-checkpointing-alpha/#restore-checkpointed-container-k8s
[criu-coordinator]: https://github.com/checkpoint-restore/criu-coordinator
[criu-image-streamer]: https://github.com/checkpoint-restore/criu-image-streamer
[TCP connection repair]: https://lwn.net/Articles/495304/
[KEP-2008]: https://git.k8s.io/enhancements/keps/sig-node/2008-forensic-container-checkpointing
[volume snapshots]: https://kubernetes.io/docs/concepts/storage/volume-snapshots/
[pod-snapshot-controller]: https://github.com/checkpoint-restore/pod-snapshot-controller

## Summary

This proposal defines CRI APIs, kubelet endpoints, and controllers together with  Kubernetes objects
for managing the lifecycle and artifacts of these operations to enable native support for Pod-level
checkpoint and restore. The scope of the proposal is limited to warm start and fault-tolerance use cases,
with outline of the API design to accommodate other use cases such as suspend/resume (with IP preservation)
and live migration (streaming checkpoint data between nodes). These use cases will be addressed in future KEPs.

The core idea is to outline the minimal set of Container Runtime Interface (CRI) and kubelet
extensions required for Pod-level checkpoint and restore, and to provide a clear path for
iteratively building on top of these APIs to address the broader set of use cases and requirements.

In this KEP, checkpoints represent the runtime state of a Pod, where the checkpoint format and low-level
implementation details are left to the container runtime (e.g., containerd, CRI-O), the OCI runtime (runc, crun),
and the underlying checkpoint/restore mechanism (e.g., CRIU, gVisor).

While Pod-level checkpointing is inspired by the existing [kubelet checkpoint] API and extends that
container checkpointing mechanism to Pods, the restore functionality is a larger addition as Kubernetes
currently supports container restore only via [OCI image annotations].

This proposal defines Pod-level checkpoint and restore as a single, cohesive feature as checkpointing
without restore would be incomplete and impractical for the use cases motivating this work.

## Motivation

The existing [kubelet checkpoint] API was originally inspired by the checkpoint/restore functionality
of container engines such as Podman. However, unlike these container engines, Kubernetes is responsible
for managing, scaling, and coordinating workloads across an entire cluster of machines. As a result,
container-level checkpointing alone does not adequately support many Kubernetes-native workflows and
higher-level operations that require preserving and restoring the full Pod state. This KEP aims to
remove this barrier by enabling a Pod-level checkpoint and restore mechanisms that is aligned with
the core Kubernetes abstractions.

### Goals

- Introduce Pod-level checkpoint and restore support to the CRI API (`CheckpointPod`, `RestorePod`).
- Extend the kubelet API with checkpoint and restore endpoints for Pod runtime state.
- Define `PodCheckpoint` object and the Pod-level restore operation.

### Non-Goals

The following items are out of scope for this KEP. Each is expected to be addressed in a
follow-on enhancement.

- Pod live migration with low latency or SLO guarantees. This requires streaming checkpoint data
  directly between nodes (without intermediate storage) and IP-address preservation for
  established TCP connections across nodes. This is partially addressed today by [criu-image-streamer]
  and [TCP connection repair], but once Pods are scheduled they are bound to a specific node
  (`pod.Spec.NodeName`), and Kubernetes does not currently guarantee network identity preservation
  across restores.

- In-place restore (same Pod UID, same Pod object). The initial implementation creates a new Pod
  from a checkpoint. Restoring into the same Pod object requires modifying Pod lifecycle
  semantics, which has deep ecosystem implications for controllers, schedulers, and monitoring
  tools.

- Cross-node restore. The initial implementation focuses on same-node restore. Cross-node restore
  requires a checkpoint transport mechanism.

- Checkpoint and restore of shared Pod resources such as shared memory, volumes, and Dynamic
  Resource Allocation (DRA) devices.

- Scheduling integration (workload-aware preemption with checkpoint awareness, eviction request
  interceptors).

- Distributed or multi-Pod coordinated checkpointing (e.g., synchronized checkpoint of a
  distributed training job). Requires external coordination tools such as [criu-coordinator].

- Handling of exec sessions and port-forward. Support for preserving and restoring exec sessions
  and port-forward can be explored in a future enhancement proposal.

- Checkpoint portability across heterogeneous environments such as different CPU and GPU
  architectures, kernel versions, container runtimes, or device drivers.

- Checkpoint lifecycle management including resource quotas, limits, retention policies, and
  garbage collection of checkpoint data.

- Application-triggered checkpointing. When creating multiple clones from the same checkpoint,
  the workload may need to refresh state such as session keys, random number generator states,
  and certificates. A future KEP will explore a common mechanism for applications to be notified
  of being cloned. For example, gVisor provides a special file (`/proc/gvisor/checkpoint`) that
  blocks until a restore is complete, allowing applications to refresh state on resume.

### Targeted use cases

#### Accelerating startup of applications with long initialization times

This is the primary driver of the alpha scope. The cold start time of many applications, such
as LLM inference services and Java applications, can reach several minutes due to complex
initialization steps that must complete before the service can accept requests or process data.
Pod checkpointing allows the initialized state of a running application to be saved to
persistent storage and later restored on demand, enabling services to resume execution without
repeating expensive initialization steps. This is the canonical warm start use case and is fully
covered by the alpha scope (new Pod created from a checkpoint on the same node).

#### Enabling fault-tolerance for long-running workloads

Training jobs for large AI models run on hundreds or thousands of GPUs and often execute for
weeks or months. Hardware and system failures are inevitable and can force jobs to restart from
scratch, resulting in significant loss of time and computational resources. Pod-level
checkpointing allows the runtime state of these workloads to be captured and restored on
failure. For example, when a training job is preempted by a batch scheduler, Pod-level
checkpoint/restore can capture and later resume the runtime state to avoid restarting the
training job. Partially served by the alpha scope: single-Pod checkpoint/restore is covered;
distributed coordination across many Pods requires a follow-on enhancement.

#### Pod migration across nodes for load balancing and maintenance

Cluster operators often need to rebalance workloads across nodes to respond to changing resource
requirements or planned maintenance events such as kernel upgrades, security patching, or node
replacement. These operations typically rely on Pod eviction and rescheduling, which forces
applications to restart and rebuild in-memory state. Pod checkpoint/restore preserves execution
state across the move, significantly reducing recovery time compared to full Pod restarts.
Partially served by the alpha scope: checkpoint and create a new Pod on the same node is
covered; cross-node migration requires a follow-on cross-node transport enhancement, and live
migration semantics require a follow-on live migration enhancement.


## Proposal

### Implementation

In this proposal, we aim to provide CRI functionality to checkpoint and restore a running Pod, which
includes all containers running in the Pod, along with Pod-level metadata and configurations. This
functionality is inspired by [kubelet checkpoint], but extends it to the Pod level, allowing to capture
and restore the execution state of a Pod, rather than individual containers. The exact implementation
details of this checkpoint/restore mechanism are left to the container runtime, but we expect the Pod
checkpoint to capture the complete execution context of all processes running in containers, including
in-memory state, process hierarchies, open file descriptors, and Pod-level configuration and metadata.

The implementation consists of three layers:

1. **CRI APIs** (`CheckpointPod`, `RestorePod`): Container runtime interface for the actual
   checkpoint/restore operations, implemented by container runtimes such as containerd and CRI-O.

2. **Kubelet endpoints** (`POST /checkpoint/{ns}/{pod}`, `POST /restore/{ns}/{checkpointName}`):
   Kubelet HTTP endpoints that translate API requests into CRI calls. The kubelet suspends health
   checks during checkpointing, resolves the CRI sandbox ID, and manages checkpoint storage at
   `/var/lib/kubelet/pod-checkpoints/`. For restore, the kubelet reads pod sandbox configuration
   from the checkpoint, assigns a new Pod UID, updates cgroup parent paths, and delegates to the
   container runtime.

3. **API objects** in the `checkpoint.k8s.io` API group that provide declarative management of
   checkpoint operations. `PodCheckpoint` is a namespace-scoped standalone object. Restore is
   triggered by a new optional `restoreFrom` field on Pod spec rather than a separate object;
   see [Restore Mechanism](#restore-mechanism). A [pod-snapshot-controller] watches
   `PodCheckpoint` objects and calls the kubelet checkpoint endpoint through the API server
   node proxy.

### Risks and Mitigations

The main risk is the complexity of implementing Pod-level checkpoint and restore within the scope
defined by the Non-Goals above, particularly in a way that is portable across different container
runtimes and Kubernetes environments while also ensuring security and reliability.

This is mitigated by defining a minimal set of kubelet and CRI extensions that enable
an iterative approach.

Specific risks and mitigations:

- Privilege model shift. The existing container-level checkpoint API is reachable only by users
  with privileged access to the kubelet (node administrator or SSH). Exposing Pod-level
  checkpoint and restore through namespaced API objects is a different security model: it lets
  regular users trigger an operation that captures full process memory, including secrets.
  Mitigations: (a) scope checkpoint resources as namespace-scoped; (b) keep the node-proxy
  privilege on the controller service account and never grant it to users; (c) treat checkpoint
  artifacts as sensitive data with the same handling as Secrets; (d) provide pre-defined
  viewer/editor/admin ClusterRoles for per-namespace binding. See
  [Security Implications](#security-implications).

- Application awareness is required. Checkpoint and restore are not transparent to applications:
  in-memory secrets, tokens, environment variables, and cached hostnames persist through restore,
  and selective memory scrubbing is not feasible. Applications must cooperate for correctness.
  See [Application Awareness and Restore Contract](#application-awareness-and-restore-contract).

- GPU memory overhead. Checkpointing workloads that run on NVIDIA GPUs typically requires the GPU
  state to be temporarily copied into host memory. This increases the memory requirement for the
  container, and adjusting the memory limit while checkpoint/restore is in progress may be
  necessary.

- Probe interference. To prevent checkpoint failures caused by transient processes (e.g., from
  exec probes, `kubectl exec`, attach sessions, or logging agents), the kubelet must suspend all
  probe executions for a Pod during its checkpointing window. Preserving exec or attach sessions
  and port-forwarding is out of scope for the initial implementation; because some probes use
  exec sessions, those are out of scope as well. The handling of active exec or attach sessions
  at checkpoint time is implementation-specific and may vary across OCI runtimes. Whether the
  kubelet rejects a checkpoint request in such cases will be clarified during implementation.

- Multi-Pod coordination. Checkpointing applications that are distributed across multiple Pods
  requires coordination to ensure consistency across checkpoints. Cross-Pod coordination is out
  of scope for this KEP and must be handled by external tools such as [criu-coordinator] or by
  application-level synchronization.

- Temporary unavailability during checkpoint. During the checkpointing window, the containers in
  the checkpointed Pod are frozen to create a consistent checkpoint. The duration of this window
  varies with the workload (for example, amount of memory at the time of checkpointing) and the
  underlying checkpoint mechanism, leading to temporary unavailability. The checkpointing state
  must be exposed by the container runtime via the Pod or Container Status API so clients can
  detect it. During this window, Ephemeral Containers may not be started for the checkpointed
  Pod, and any existing Ephemeral Containers are frozen along with the regular containers.

- Disk consumption. Large checkpoint artifacts can consume significant disk space. Size depends
  on the container root filesystem writable layer, the memory usage of running processes at the
  time of checkpointing, and any applied data compression, making precise estimation in advance
  difficult. Checkpoint retention and deletion mechanisms and appropriate storage limits must be
  configured in advance to prevent node disk pressure. A dedicated checkpoint lifecycle
  management enhancement is planned; see [Roadmap](#roadmap).

- Denial of service via excessive checkpointing. Unrestricted checkpoint operations can exhaust
  node disk space. This risk also applies to the existing container-level checkpoint API.
  Initial mitigation is the existing checkpoint-restore operator retention policy; a longer-term
  mitigation will come with the checkpoint lifecycle management enhancement.

## Design Details

### CRI API Extensions

This KEP proposes the following CRI APIs for Pod-level checkpoint/restore, inspired by the
ContainerCheckpoint API.

#### CheckpointPod

Proposed CRI API extension for CheckpointPod:

```proto
service RuntimeService {
    ...
    // CheckpointPod creates a Pod-level checkpoint. If the pod sandbox does not
    // exist or the checkpoint operation fails, the call returns an error.
    rpc CheckpointPod(CheckpointPodRequest) returns (CheckpointPodResponse) {}
    ...
}

message CheckpointPodRequest {
    // ID of the pod sandbox to be checkpointed.
    string pod_sandbox_id = 1;
    // Location path where the checkpoint will be saved
    string path = 2;
    // Timeout in seconds for the checkpoint to complete.
    // Timeout of zero means to use the CRI default.
    // Timeout > 0 means to use the user specified timeout.
    int64 timeout = 3;
    // Checkpoint options passed to the container runtime.
    map<string, string> options = 4;
}

message CheckpointPodResponse {}
```

In the event of a timeout, the container runtime should return an error indicating that the checkpoint
operation did not complete within the specified time limit and ensuring that any partially created
checkpoint artifacts are cleaned up. The kubelet should handle this error appropriately, for example
by returning a timeout error to the caller of the `CheckpointPod` API.

#### RestorePod

```proto
service RuntimeService {
    ...
    // RestorePod restores a pod sandbox from a checkpoint
    rpc RestorePod(RestorePodRequest) returns (RestorePodResponse) {}
    ...
}

message RestorePodRequest {
    // Location path where the checkpoint will be restored from.
    string path = 1;
    // Optional pod sandbox configuration to override checkpoint metadata.
    // If not specified, the pod will be restored with its original configuration.
    PodSandboxConfig config = 2;
    // Timeout in seconds for the restore to complete.
    // Timeout of zero means to use the CRI default.
    // Timeout > 0 means to use the user specified timeout.
    int64 timeout = 3;
    // Restore options passed to the container runtime.
    map<string, string> options = 4;
    // Container configurations for all containers in the pod.
    // This includes mount configurations that tell the runtime where to mount
    // host paths (e.g., /etc/hosts, termination logs, volumes) into the containers.
    // The runtime should match containers from the checkpoint with these configs
    // by container name and apply the mount configurations.
    repeated ContainerConfig container_configs = 5;
}

message RestorePodResponse {
    // ID of the restored pod sandbox
    string pod_sandbox_id = 1;
}
```

For security reasons, certain fields in the Pod spec (e.g., namespace) are blocked from changing during
the restore. In particular, the Pod UID must remain automatically generated by Kubernetes and must not
be user-specifiable. While this KEP does not include the complete set of restricted fields, we expect
the exact fields and validation rules to be fully specified as part of the implementation enforcing
strict validation to Pod spec. A future KEP might also explore utilizing RBAC permissions to verify
that the requesting user has access to both the source and destination namespaces.

In the event of a timeout, the container runtime should return an error indicating that the restore
operation did not complete within the specified time limit and ensuring that any partially restored
artifacts are cleaned up. The kubelet should handle this error appropriately, for example by
returning a timeout error to the caller of the `RestorePod` API.

### Kubelet API Endpoints

The kubelet exposes HTTP endpoints for Pod-level checkpoint and restore, gated behind the
`KubeletLocalPodCheckpointRestore` feature gate.

#### Checkpoint Endpoint

```
POST /checkpoint/{podNamespace}/{podName}[?timeout={seconds}]
```

The checkpoint endpoint:
1. Validates the target Pod exists and is in the `Running` phase.
2. Resolves the CRI sandbox ID from the Pod's runtime status.
3. Generates a checkpoint path: `/var/lib/kubelet/pod-checkpoints/checkpoint-{podName}_{namespace}-{timestamp}`.
4. Calls the `CheckpointPod` CRI API with the sandbox ID and checkpoint path.
5. Returns the checkpoint path in the response:

```json
{"items": ["/var/lib/kubelet/pod-checkpoints/checkpoint-myapp_default-2026-03-10T20:38:11Z"]}
```

#### Restore Endpoint

```
POST /restore/{podNamespace}/{checkpointName}
```

The restore endpoint:
1. Validates the checkpoint name (rejects path traversal characters `/` and `..`).
2. Resolves the full checkpoint path within `/var/lib/kubelet/pod-checkpoints/`.
3. Reads Pod sandbox configuration from the checkpoint.
4. Looks up the existing Pod object from the API server (required for CNI network setup).
5. Assigns a new Pod UID and updates the cgroup parent path and log directory.
6. Builds `ContainerConfig` entries from the Pod spec for mount configurations.
7. Calls the `RestorePod` CRI API with the checkpoint path and configuration.
8. Returns the restored sandbox ID:

```json
{"podSandboxId": "containerd-generated-sandbox-id"}
```

### PodCheckpoint Objects

To provide declarative management of checkpoint operations, this KEP introduces a new
built-in Kubernetes API type, `PodCheckpoint`, in the `checkpoint.k8s.io/v1alpha1` API group.
`PodCheckpoint` is a first-class Kubernetes resource (not a CRD); it is served by the API
server alongside core types such as `Pod` and `Node`. The design follows the Kubernetes
[volume snapshots] pattern: a checkpoint is a standalone object with its own lifecycle that
can outlive the source Pod and be used to create multiple new Pods. The restore side makes
use of a new `restoreFrom` field on Pod spec described below.

#### PodCheckpoint

A `PodCheckpoint` object triggers a checkpoint of a named Pod. The controller watches for these
objects, calls the kubelet checkpoint endpoint through the API server node proxy, and updates the
status to reflect the result.

```yaml
apiVersion: checkpoint.k8s.io/v1alpha1
kind: PodCheckpoint
metadata:
  name: my-checkpoint
spec:
  # Name of the running Pod to checkpoint.
  sourcePodName: my-app
  # Optional timeout in seconds (0 = use container runtime default).
  timeoutSeconds: 30
status:
  # Phase: Pending, InProgress, Ready, or Failed.
  phase: Ready
  # Node where the source Pod was running when checkpointed.
  nodeName: node-1
  # Path to checkpoint data on the node.
  checkpointLocation: /var/lib/kubelet/pod-checkpoints/checkpoint-my-app_default-2026-03-10T20:38:11Z
  # List of checkpointed containers.
  containers:
  - name: main
    image: my-app:latest
  conditions:
  - type: Ready
    status: "True"
    reason: CheckpointCompleted
```

#### Pod-Snapshot-Controller

The [pod-snapshot-controller] manages the lifecycle of `PodCheckpoint` objects. It
communicates with the kubelet through the API server node proxy
(`/api/v1/nodes/{node}/proxy/checkpoint/...`).

The checkpoint reconciliation flow:

1. Watch for new `PodCheckpoint` objects.
2. Validate that the target Pod exists, is `Running`, and has all init containers completed.
3. Set phase to `InProgress` and record the node name.
4. Call `POST /api/v1/nodes/{node}/proxy/checkpoint/{namespace}/{pod}` with optional timeout.
5. On success, set phase to `Ready` and store the checkpoint location and container info.
6. On failure, set phase to `Failed` with error details in conditions.

Restore does not require controller involvement: the kubelet drives restore directly from the
Pod spec (see [Restore Mechanism](#restore-mechanism)).

### Restore Mechanism

Restore is triggered by a new optional field on Pod spec rather than by a separate API object.
A user creates a Pod with `spec.restoreFrom` set to a checkpoint reference; the kubelet
observes this during `SyncPod` and calls `restorePodSandbox()` instead of `createPodSandbox()`.
Pod creation is the restore, in a single step.

This shape collapses the restore flow into the normal Pod admission and scheduling path: the
scheduler places the Pod (subject to the node-affinity constraints needed to land on a node
that has the checkpoint data), the CNI plugin sets up networking against the Pod object
exactly as it would for a fresh Pod, and the kubelet swaps the sandbox creation step for
sandbox restore. No placeholder Pod, no separate object lifecycle, and no `nodes/proxy`
permission for restore.

### Checkpoint Content

Different container and OCI runtimes may implement the Pod checkpointing mechanism differently, and the
exact content of a checkpoint may vary. However, we expect that a Pod checkpoint will capture the
complete execution context of all processes and threads running in containers, including memory state,
process hierarchies, open file descriptors, and Pod-level configuration and metadata.

In the context of this proposal, support for volumes and network configuration is considered out of scope
for the initial implementation. However, the checkpoint should capture the necessary information to allow
the runtime to configure the network stack and reattach to the same volumes during restore.

A high-level overview of the Pod checkpoint contains the following:

#### Pod Specification and Metadata

A Pod checkpoint captures all information required for the Pod to be restored. This includes
not only the serialized Pod specification, but also any node-local state maintained by the
kubelet that is necessary to correctly restore execution. In this context, *Pod specification*
refers to the CRI `PodSandboxConfig` passed from the kubelet to the container runtime. This is
distinct from the full `v1.PodSpec` defined in the API server. This KEP aims to describe an
initial high-level approach to Pod-level checkpointing and leave the implementation details
to each container runtime.

The checkpoint includes:
- Serialized Pod specification
- Pod UID, namespace, labels, annotations, and owner references
- Resource requests and limits
- Scheduling constraints and security contexts
- Status of all containers, including containers that have completed execution

This proposal considers checkpointing while init containers are running out for
the initial implementation, and the handling of init containers may be further
explored in future iterations.

Note that the `RestorePod` API allows users to optionally override fields such as resource requests and
limits. During restore, the process tree inside containers is recreated from the application state
captured during checkpointing. This operation requires, for example, open file descriptors and memory
allocations to be recreated with the same offsets and contents as at the time of checkpointing, in order
to ensure correct application behavior.

For Pods using Dynamic Resource Allocation, resource claims are expected to be resolved during restore,
and the checkpoint should capture the necessary information to allow the runtime to reattach to the same
resource claims.

#### Container Runtime State

The container runtime state includes OCI container configurations, security contexts, the writable layer
of the container filesystem, and checkpoint images representing the state of the Pod that capture the
complete execution context of all processes and threads running in containers necessary to recreate them,
and resume their execution as it was at the time the checkpoint was created.

#### Shared Pod Resources

This KEP focuses on providing the fundamental building blocks for capturing and restoring the execution
state of containers within a Pod, along with Pod-level metadata and configurations. Support for shared
Pod resources such as shared memory and volumes is out of scope for the initial implementation.

### Pod Lifecycle

Pod-level checkpointing is permitted only on a Pod that is bound to a node, has all init
containers completed, and has all regular containers started and running. Checkpoint requests on
Pods that do not meet these preconditions must be rejected before reaching the container
runtime. Init container semantics and partial-ready states are out of scope for this KEP.

During checkpointing, all containers in the Pod are frozen (using the Pod-level cgroup freezer)
as a prerequisite for creating a consistent checkpoint. Each container is then checkpointed
individually, and the cgroup is unfrozen at the end of this operation.

The kubelet must suspend liveness and readiness probes while a Pod is being checkpointed. Frozen
cgroups may cause probes to time out, and without suspension the kubelet would kill the Pod
mid-checkpoint. A Pod status condition (`Checkpointing=True`) is set so that higher-level
controllers can observe this state.

### TCP Connection Handling

The initial implementation uses a TCP-close approach: all established TCP connections are closed
when a Pod is checkpointed. TCP-established connection preservation (restoring connections to
their pre-checkpoint state) requires CNI changes across all implementations and is deferred to a
future live migration KEP. IP address preservation across checkpoint/restore also requires CNI
changes and has been confirmed as feasible by SIG Network but represents significant work.

### Security Implications

Like the container-level checkpoint API described in [KEP-2008], the kubelet Pod-level
checkpoint and restore endpoints are restricted to callers with privileged access to the
kubelet API. The namespaced API objects defined in this KEP are the user-facing surface; users
never need direct kubelet access.

#### Privilege model

The existing container-level checkpoint API requires node administrator or SSH access to reach
the kubelet endpoint. Exposing Pod-level checkpoint and restore through namespaced API objects
is a different security model. Mitigations:

- `PodCheckpoint` is namespace-scoped and may only target Pods in the same namespace; the
  controller enforces same-namespace lookups. Restore is triggered by `spec.restoreFrom` on a
  Pod and so is naturally scoped to the namespace in which the Pod is created. Cross-namespace
  restore is not possible by construction.
- The `nodes/proxy` permission permits arbitrary kubelet API calls and is granted only to the
  pod-snapshot-controller service account. It is never granted to end-user roles. Restore
  does not require `nodes/proxy` because it flows through the normal Pod admission path.
- The controller exposes no user-facing API beyond the `PodCheckpoint` resource; all kubelet
  interaction for checkpointing is mediated by the controller.
- Pre-defined namespaced ClusterRoles (viewer, editor, admin) are provided so administrators
  can bind checkpoint and restore access per namespace with `RoleBinding`.
- `sourcePodName` on `PodCheckpoint` and `spec.restoreFrom` on Pod are immutable after
  creation, preventing post-creation namespace-escape attempts.

#### Sensitive memory contents

Checkpoint data may contain sensitive information from process memory, including secrets,
tokens, and encryption keys. Checkpoint artifacts must be treated as sensitive data, stored
with the handling expected for Secrets, and subject to the same access controls. Encryption of
checkpoint data at rest is CRIU-level work and is out of scope for this KEP.

#### Denial of service via excessive checkpointing

Unrestricted checkpoint operations can exhaust node disk space. This risk also applies to the
existing container-level checkpoint API. Initial mitigation is the existing checkpoint-restore
operator retention policy; checkpoint lifecycle management is a follow-on enhancement.

#### automountServiceAccountToken on restore

Service account tokens mounted into the original Pod may be invalid or expired when a
checkpoint is restored. Checkpointable workloads should disable token automounting and refresh
tokens explicitly after restore; a formal opt-out or automatic token refresh mechanism will be
specified before Beta.

#### Path traversal protection

The kubelet restore endpoint validates checkpoint names, rejecting paths containing `/` or
`..`, and verifies that the resolved checkpoint path remains within the checkpoint storage
directory (`/var/lib/kubelet/pod-checkpoints/`).

#### Status and spec separation

Users write `spec`, controllers write `status`. `PodCheckpoint` is a built-in API type and
the REST storage layer enforces this separation: the main-object strategy strips `status` on
user updates, and the status-object strategy strips `spec` on controller updates. Neither
side can cross the boundary.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing

- Kubelet probe suspension during the paused Pod phase (existing behaviour for forensic
  container checkpointing) must be generalized to Pod-level checkpoints.
- The CRI conformance suite must be extended to cover the new `CheckpointPod` and `RestorePod`
  RPCs once at least one runtime implements them.

##### Unit tests

Coverage baselines will be captured when the implementation PR is opened.

Unit tests must cover at least:

- Kubelet endpoint argument validation (valid, missing, and malformed Pod reference).
- Path traversal rejection on the restore endpoint (`/`, `..`, absolute paths, symlink
  escapes).
- Pod phase precondition (checkpoint rejected unless the Pod is `Running` with all init
  containers completed).
- CRI timeout propagation (kubelet returns a timeout error on `CheckpointPodRequest.timeout`
  expiry).
- Feature gate disabled: endpoints return 404 or 405.
- Cgroup freeze and unfreeze sequence ordering and error recovery.
- Pod condition `Checkpointing=True` is set and cleared around the operation.

##### Integration tests

CRI API changes must be implemented by at least one container engine. Because the kubelet has
no integration test harness, validation uses `test/e2e_node`, which effectively serves as the
kubelet integration suite. The following scenarios must pass before Alpha:

- `CheckpointPod` happy path: checkpoint a single-container Pod; verify the returned path
  exists and the archive is non-empty.
- `RestorePod` happy path: restore that Pod; verify a new sandbox ID is returned.
- Probe suspension: a Pod with a 1 second liveness probe is not killed during a multi-second
  checkpoint window.
- Runtime does not implement the new RPC: the kubelet returns a typed error rather than a
  panic or generic 500.
- Feature gate disabled: the checkpoint endpoint returns 405 and `spec.restoreFrom` is
  rejected at Pod admission.
- `spec.restoreFrom` happy path: the kubelet sees the field during `SyncPod`, calls
  `restorePodSandbox()`, and the Pod transitions to `Running`.

##### e2e tests

Alpha ships with e2e tests that validate the Pod-level checkpoint and restore flow against at
least one CRI implementation (containerd with a CRIU-based runtime). The initial e2e tests
tolerate the absence of CRI support and skip with a clear message on runtimes that have not
yet adopted the new RPCs; they become required as runtime support lands.

The alpha e2e suite covers:

- End-to-end warm start: create a counter Pod, wait for it to increment, create a
  `PodCheckpoint`, wait for `Ready`, create a new Pod from the checkpoint, and verify the
  counter resumes from the saved value.
- Multi-container Pod: verify the per-container freeze sequence and that all containers are
  present and in the correct state after restore.
- Same-node restore: restore on the same node as the checkpoint (the only supported mode in
  alpha).
- Failure paths: missing or `Pending` checkpoint referenced by `spec.restoreFrom`; checkpoint
  data missing on the target node; restore Pod scheduled to a node that does not have the
  checkpoint.
- RBAC boundary: a user with `editor` access in one namespace cannot create a `PodCheckpoint`
  referencing a Pod in another namespace, and cannot create a Pod with `spec.restoreFrom`
  pointing to a `PodCheckpoint` in another namespace.

Beta adds:

- A second CRI implementation.
- Runbook-driven failure mode coverage (see [Troubleshooting](#troubleshooting)).
- Observability metrics presence and shape.

### Graduation Criteria

#### Alpha

- CRI API extensions for `CheckpointPod` and `RestorePod` implemented and documented.
- Kubelet checkpoint and restore HTTP endpoints implemented behind the
  `KubeletLocalPodCheckpointRestore` feature gate.
- `PodCheckpoint` defined and implemented. Restore trigger implemented as a new optional
  `restoreFrom` field on Pod spec.
- Pod-snapshot-controller implemented.
- End-to-end warm start workflow: checkpoint a running Pod, create a new Pod from that
  checkpoint on the same node. Demonstrated against at least one CRI implementation.
- e2e tests described in the [Test Plan](#test-plan) pass in CI on supported runtimes and skip
  cleanly on unsupported runtimes.
- Pre-defined viewer, editor, and admin ClusterRoles published for the namespaced resources.
- Alpha-level PRR answered.

#### Beta

- At least two CRI implementations support the new RPCs. Low-level runtime support is
  available in released versions.
- Metrics listed under [Monitoring Requirements](#monitoring-requirements) are emitted and
  covered by tests.
- Documented runbook for every failure mode listed under [Troubleshooting](#troubleshooting).
- `automountServiceAccountToken` handling on restore has a specified contract (opt-out or
  automatic refresh).
- A formal opt-in signal for checkpointable workloads is specified.
- Additional e2e testing for stabilization; known issues and gaps documented.
- No open CVE-class issues for the feature.

#### GA

- Feature has been stable in Beta for at least two Kubernetes releases.
- Feedback gathered from production deployments.
- Conformance tests cover all GA endpoints.
- At least three major container runtimes support the feature.
- User-facing documentation published on kubernetes.io.

### Upgrade / Downgrade Strategy

On upgrade, if the container runtime implements the required CRI API changes, the kubelet may enable
and use Pod-level checkpointing when the feature gate is enabled. If the container runtime does
not implement the new CRI APIs, the kubelet will return an error on checkpoint API calls, and the
feature will be unavailable.

On downgrade, if the container runtime does not implement the Pod-level checkpoint CRI API,
the kubelet will return an error on checkpoint API calls, and the feature will be unavailable.
Similarly, if the kubelet is downgraded to a version that does not implement these CRI APIs,
the feature will be unavailable.

### Version Skew Strategy

The CRI API extensions and kubelet endpoints are local to the node. The `PodCheckpoint`
built-in API type is served by the API server; the pod-snapshot-controller runs in-tree (as
part of `kube-controller-manager` or as an out-of-tree controller depending on the final
implementation decision) and communicates with the kubelet through the API server node
proxy. Restore is driven entirely by the kubelet from `spec.restoreFrom` on the Pod object;
no controller is involved on the restore path. Version skew considerations:

- If the kubelet supports the new CRI API but the container runtime does not,
  the kubelet will return an error when the checkpoint Pod API is called,
  and the feature will not be available.

- If the container runtime supports the new CRI APIs but the kubelet does not,
  the feature will not be available since there is no kubelet API to trigger
  the Pod-level checkpoint operations.

- If the pod-snapshot-controller is deployed but the kubelet does not support the
  checkpoint/restore endpoints, the controller will receive errors when attempting
  to call the kubelet through the API server node proxy, and operations will be
  marked as `Failed`.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `KubeletLocalPodCheckpointRestore`
  - Components depending on the feature gate: Kubelet

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. By disabling the `KubeletLocalPodCheckpointRestore` feature gate.

###### What happens if we reenable the feature if it was previously rolled back?

The Pod-level checkpoint/restore functionality will become available again.

###### Are there any tests for feature enablement/disablement?

Tests will be extended to verify the functionality is turned off when feature gate
is disabled and turned on when enabled.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

The feature is node-local: it is gated by a kubelet feature flag and has no API server
component beyond the namespaced resources. Rollout consequences:

- Partial rollout. In a cluster where the feature gate is enabled on some kubelets and not
  others, checkpoint and restore operations succeed only on nodes where the kubelet has the
  feature enabled. The controller must tolerate this: operations targeting unsupported nodes
  are marked `Failed` with a clear reason rather than stuck `InProgress` indefinitely.
- Mid-rollout kubelet restart. A checkpoint in flight when the kubelet restarts either
  completes and cleans up on restart or leaves a partial checkpoint file on disk. The kubelet
  garbage-collects partial files on start; the controller marks the `PodCheckpoint` as
  `Failed`.
- Version skew. If the kubelet has the feature gate enabled but the container runtime does not
  implement the new CRI RPCs, the CRI call fails with `Unimplemented` and the kubelet returns
  a typed error. The controller marks the operation `Failed` with a clear reason.
- Already-running workloads. Not affected. No existing Pod behaviour changes when the feature
  is enabled. Only Pods targeted by an explicit checkpoint or restore request are impacted,
  and the checkpoint window pauses them only for the duration of the operation.
- Rollback. Disabling the feature gate on a kubelet has no effect on existing Pods and no
  persistent state is left behind. Checkpoint artifacts remain on disk until the retention
  policy cleans them up. Operations initiated after rollback fail with a typed error.

###### What specific metrics should inform a rollback?

It is possible to query the number of failed checkpoint operations using the
*kubelet* metrics API endpoint `kubelet_runtime_operations_errors_total`.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No data is stored, so re-enabling starts from a clean slate.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

This feature is only visible at the kubelet API. An operator can query kubelet-exposed metrics to determine if it is
being used.

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event reasons on `PodCheckpoint`: `CheckpointStarted`, `CheckpointSucceeded`,
    `CheckpointFailed`.
  - Event reasons on the restored Pod: `RestoreStarted`, `RestoreSucceeded`, `RestoreFailed`.
  - Event reason on the source Pod: `CheckpointingPod`, set when the checkpoint window starts
    and cleared when it ends.
- [x] API `.status`
  - `PodCheckpoint.status.phase` transitions through `Pending`, `InProgress`, and terminates
    at `Ready` or `Failed`.
  - `PodCheckpoint.status.conditions[type=Ready]` with reasons `CheckpointInProgress`,
    `CheckpointCompleted`, `CheckpointFailed`.
  - On the source Pod: a condition `Checkpointing=True` while the checkpoint window is active
    (see [Pod Lifecycle](#pod-lifecycle)).
  - On the restored Pod: `spec.restoreFrom` records the `PodCheckpoint` that produced it; a
    condition `Restoring=True` is set while the sandbox restore is in flight and cleared once
    the Pod is `Running`.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

A failed checkpoint does not affect the source workload; the source Pod keeps running after
the attempt. The expected behaviour is binary: a checkpoint either succeeds or fails, with no
partial state reflected as success.

For Alpha there is no SLO beyond "operations return a typed success or failure response."
Formal SLOs will be defined before Beta once production data is available.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Kubelet metrics (emitted when the feature gate is enabled):

- `kubelet_pod_checkpoint_operations_total{result="success|failure"}`, counter.
- `kubelet_pod_checkpoint_duration_seconds`, histogram with buckets sized for sub-second
  through multi-minute checkpoints.
- `kubelet_pod_restore_operations_total{result="success|failure"}`, counter.
- `kubelet_pod_restore_duration_seconds`, histogram.
- `kubelet_pod_checkpoint_size_bytes`, histogram of produced checkpoint archive sizes.
- `kubelet_runtime_operations_errors_total{operation_type="checkpoint_pod|restore_pod"}`,
  existing kubelet metric extended to cover the new CRI calls.

Controller metrics:

- `podcheckpoint_phase_total{phase="Pending|InProgress|Ready|Failed"}`, counter of phase
  transitions.
- `podcheckpoint_reconcile_duration_seconds`, histogram.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

- Per-container CRIU dump phase timings (requires CRI-level instrumentation).
- Disk pressure signal before checkpoint write (currently observable only after failure).
- Attribution of checkpoint storage consumption to the owning workload (covered by the future
  checkpoint lifecycle management enhancement).

### Dependencies

The container runtime must support the `CheckpointPod` and `RestorePod` CRI API calls.
This functionality relies on checkpoint/restore mechanisms provided by low-level OCI
container runtimes such as `runc`, `crun`, `youki`, or secure sandbox container runtimes
such as `gVisor`. These OCI container runtimes require [CRIU](https://criu.org/Main_Page)
(Checkpoint/Restore In Userspace) to be installed, while `gVisor` provides its own internal
checkpoint/restore implementation. In addition, there are some workload-specific dependencies,
such as the [cuda-checkpoint](https://github.com/NVIDIA/cuda-checkpoint) utility required to
support workloads running on NVIDIA GPUs.

###### Does this feature depend on any specific services running in the cluster?

This feature does not require any specific services to be running in the cluster.
However, the container runtime must support the Pod Checkpoint/Restore CRI API calls.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

Yes, only when explicitly invoked by a user. Creating a `PodCheckpoint` triggers a single
`POST /api/v1/nodes/{node}/proxy/checkpoint/{ns}/{pod}` from the controller to the kubelet.
Restore does not introduce any new API calls beyond the normal Pod create that already
occurs in the existing Pod lifecycle. There are no periodic or background API calls.

###### Will enabling / using this feature result in introducing new API types?

Yes:

- `PodCheckpoint` in the `checkpoint.k8s.io/v1alpha1` API group, namespace-scoped. One object
  per checkpoint operation.
- A new optional field `restoreFrom` on Pod spec referencing a `PodCheckpoint` in the same
  namespace.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Pod spec gains one optional field, `restoreFrom`, referencing a `PodCheckpoint` in the same
namespace. The additional bytes are negligible (a single reference string or
`TypedLocalObjectReference`).

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. Normal Pod lifecycle operations are unchanged. The checkpoint window pauses the source
Pod (visible via the `Checkpointing=True` condition) but does not alter any measured SLIs for
unrelated Pods.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

During checkpointing the memory pages of all processes running in the checkpointed containers will be saved to disk.
In addition, the read-write layer of the rootfs of checkpointed containers is included as part of the checkpoint.
As a result, disk usage is expected to increase by the compressed size of these checkpoints.

To avoid running out of disk space, cluster administrators can utilize the checkpoint retention policies
provided by the Checkpoint/Restore operator: <https://github.com/checkpoint-restore/checkpoint-restore-operator>

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

As described in the Risks and Mitigations section, creating checkpoints is expected to increase disk usage.
To mitigate this, cluster administrators can leverage the checkpoint retention policies provided by the
Checkpoint/Restore Operator: <https://github.com/checkpoint-restore/checkpoint-restore-operator>

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

- In-flight operations complete. The kubelet checkpoint and restore endpoints do not require
  API server connectivity to execute the CRI call; the kubelet completes the operation
  against the container runtime and records the result locally.
- Status updates are deferred. The pod-snapshot-controller cannot reconcile `PodCheckpoint`
  objects without the API server; operations remain `InProgress` until the controller can
  connect again, at which point it re-examines kubelet state and updates status. No progress
  is lost.
- New operations are blocked. Users cannot create new `PodCheckpoint` objects or Pods with
  `spec.restoreFrom` without the API server. This is expected and identical to every other
  Pod-create-driven flow.

###### What are other known failure modes?

- Container runtime does not implement the new CRI RPCs.
  - Detection: `kubelet_runtime_operations_errors_total{operation_type="checkpoint_pod"}`
    increases; API `.status` shows `Failed` with reason `RuntimeDoesNotSupportCheckpoint`.
  - Mitigation: upgrade the runtime to a version that supports the new RPCs, or disable the
    feature gate.
  - Diagnostics: kubelet logs `failed to call CheckpointPod: Unimplemented` at V(2).
  - Testing: an e2e test that injects a runtime without CRI support.
- Checkpoint timeout.
  - Detection: `kubelet_pod_checkpoint_operations_total{result="failure"}` increases; status
    reason `CheckpointTimeout`.
  - Mitigation: raise `PodCheckpoint.spec.timeoutSeconds`; reduce the workload in-memory
    footprint; checkpoint fewer Pods concurrently.
  - Diagnostics: kubelet logs `checkpoint timed out after %d seconds` at V(2); CRIU logs in
    the runtime.
  - Testing: a unit test on timeout propagation and an e2e test with an artificially short
    timeout.
- Disk exhaustion on the checkpoint directory.
  - Detection: the node enters `DiskPressure`; checkpoint operations fail with
    `no space left on device`.
  - Mitigation: configure retention via the checkpoint-restore operator; resize
    `/var/lib/kubelet`; drain the node.
  - Diagnostics: kubelet logs and `kubectl describe node` `DiskPressure` events.
  - Testing: manual; automated coverage comes with the checkpoint lifecycle management
    enhancement.
- Node proxy call fails (checkpoint path).
  - Detection: `podcheckpoint_phase_total{phase="Failed"}` increases; condition reason
    `NodeProxyFailed`.
  - Mitigation: confirm the controller service account has `create` on `nodes/proxy`;
    confirm the target node is `Ready`; retry.
  - Diagnostics: controller logs the HTTP status and body.
  - Testing: an e2e test that removes `nodes/proxy` from the controller.
- Restore Pod scheduled to a node without the checkpoint data.
  - Detection: the restore Pod stays in `ContainerCreating`; kubelet event
    `CheckpointDataMissing`.
  - Mitigation: ensure `spec.restoreFrom` is paired with appropriate node affinity matching
    the node that holds the checkpoint (recorded in `PodCheckpoint.status.nodeName`); cross-
    node checkpoint transport is a follow-on enhancement.
  - Diagnostics: kubelet logs the resolved checkpoint path and the missing-file error.
  - Testing: an e2e test that creates a Pod with `restoreFrom` pinned to the wrong node.
- Probe suspension not honoured.
  - Detection: the source Pod enters `Failed` or `OOMKilled` during the checkpoint window;
    metric `kubelet_pod_checkpoint_operations_total{result="failure"}` increases.
  - Mitigation: implementation bug in the kubelet; no operator-side mitigation.
  - Diagnostics: kubelet logs probe execution against a Pod with `Checkpointing=True`.
  - Testing: unit test that the probe manager skips probes while `Checkpointing=True`; an
    e2e test with an aggressive liveness probe.
- CNI plugin fails network setup for the restore Pod.
  - Detection: the restore Pod stays in `ContainerCreating`; events show
    `FailedCreatePodSandBox`.
  - Mitigation: CNI specific; some plugins require Pod annotations to be added to the CNI
    plugin allow-list. The restore path uses the standard Pod create flow, so any CNI plugin
    that supports normal Pods supports restore as well.
  - Diagnostics: `kubectl describe pod` on the restore Pod and CNI plugin logs.
  - Testing: an e2e test against at least one CNI implementation.
- Clock skew on checkpoint filename timestamp.
  - Detection: filename collisions or overwritten checkpoints.
  - Mitigation: include a monotonically increasing suffix alongside the timestamp.
  - Diagnostics: kubelet logs the full generated path.
  - Testing: a unit test on path generation.

###### What steps should be taken if SLOs are not being met to determine the problem?

SLOs are not yet formalized; this section will be completed for Beta. For Alpha, the
operator should:

1. Check `kubelet_pod_checkpoint_operations_total{result="failure"}` and the corresponding
   error metric for a pattern (single node, single runtime, one Pod, or systemic).
2. Check the affected checkpoint objects for `status.conditions` with reason strings matching
   the failure modes above.
3. If the kubelet is the source of failure, capture kubelet logs at V(4) and the runtime
   CRIU logs for the affected container.
4. If the controller is the source, capture controller logs and the HTTP response from the
   node proxy.

## Implementation History

- 2026-01-29: KEP opened.
- 2026-03-10: Design details expanded with kubelet restore endpoint, `PodCheckpoint` API
  object, pod-snapshot-controller, security considerations, and prior art.
- 2026-04-23: Scope narrowed to the warm start use case. Status set to `provisional` pending
  API review.
- 2026-05-14: Restore mechanism finalized: restore is triggered by a new optional
  `restoreFrom` field on Pod spec; the previously proposed `PodRestore` object is removed.
  `PodCheckpoint` is to be a built-in Kubernetes API type rather than a CRD.

## Drawbacks

There are no drawbacks that we are aware of.

## Alternatives

- **Container-level checkpointing.** Rejected because it cannot preserve runtime state in shared
  namespaces or multi-container consistency. Pod is the fundamental unit in Kubernetes; all
  higher-level controllers (Deployments, StatefulSets, Jobs) operate on Pods. VM-based runtimes
  (Kata, gVisor) checkpoint at Pod level, not container level, so a Pod-level API naturally
  accommodates them.

### Rejected Approaches

- **Restart policy extension ("fromCheckpoint").** Adding a "fromCheckpoint" value to the Pod restart
  policy was rejected because restart policy has "failure recovery" semantics. Checkpoint/restore serves
  many use cases beyond failure (scaling, migration, preemption, warm start), making this semantically
  misleading and too narrow.

- **Labels/annotations for checkpoint opt-in.** Using labels or annotations to mark Pods as
  checkpointable was rejected because labels have no RBAC protection; anyone can remove them. This
  is not suitable for security-sensitive functionality in core Kubernetes.

- **Container image name override for restore.** Replacing the container image name with a checkpoint
  archive path to trigger restore (as used in the existing forensic checkpointing feature) was rejected
  because it does not work for Pod-level restore (what image name to use for a multi-container Pod?)
  and creates confusing Pod generation semantics.

- **Parent cgroup freezer for atomic Pod freeze.** Using the parent cgroup freezer to freeze an entire
  Pod atomically was rejected because CRIU is not aware of the parent cgroup freezer. CRIU needs to
  unfreeze individual containers for parasite code injection, and processes are frozen one-by-one
  internally. Per-container cgroup freezing is simpler and works correctly with CRIU.

- **Kubelet-only scope (no API server changes).** Keeping the KEP scope to kubelet-level changes only
  was rejected because a restored Pod with no API server representation is not useful. Even if alpha
  does not fully implement API server awareness, the KEP must describe the path to a useful end-to-end
  feature.

- **Separate `PodRestore` API object.** A standalone `PodRestore` resource that references a
  `PodCheckpoint` and is reconciled by a controller (which then creates a placeholder Pod and
  calls the kubelet restore endpoint through `nodes/proxy`) was considered and rejected. The
  separate-object shape duplicates the Pod lifecycle, requires a placeholder Pod with
  surrogate spec fields to satisfy CNI plugins, requires `nodes/proxy` on the restore path,
  and introduces a second status state machine that must be kept in sync with the restored
  Pod's own status. The `spec.restoreFrom` field on Pod spec collapses restore into the
  normal Pod create flow: the scheduler, CNI plugins, controllers, and observability tooling
  all see a single Pod object with its standard lifecycle, and the only change is that the
  kubelet swaps `createPodSandbox()` for `restorePodSandbox()` when `spec.restoreFrom` is
  set. The trade-off is a small Pod spec addition, which is justified by the simplification
  on every other axis.

- **`PodCheckpoint` as a CRD.** Shipping `PodCheckpoint` as a CRD (in an out-of-tree
  controller bundle) was considered and rejected for the in-tree KEP scope. As a CRD, the
  type would not be installed by default on every cluster, would not benefit from the
  API server's built-in validation, conversion, and defaulting machinery for core types, and
  would not have the same upgrade and conformance guarantees as built-in Kubernetes
  resources. Because checkpoint and restore is intended to be a first-class Kubernetes
  capability and is tightly coupled to the kubelet and CRI APIs (which are themselves
  first-class), `PodCheckpoint` is defined as a built-in API type.

## Prior Art

- **Google GKE Pod Snapshotting.** CRD-based snapshot policies with gVisor integration,
  application-triggered checkpointing, and GCS storage.

- **Alibaba Cloud Agent Sandbox.** CRD-based sandbox/checkpoint APIs with pause/awake lifecycle,
  shared storage with lazy loading, and a separate checkpoint object with its own lifecycle.

- **CastAI.** Production checkpoint/restore with node-to-node streaming and low-latency live
  migration, approximately 2 years of production experience.

- **Nvidia Dynamo.** Inference engine with checkpoint storage types (OCI image, PVC volume, S3 bucket).

- **Kubernetes Volume Snapshots.** API design precedent (`VolumeSnapshot`, `VolumeSnapshotContent`,
  `VolumeSnapshotClass`) used as the model for the checkpoint side of this KEP.

- **Checkpoint/Restore Operator.** Existing out-of-tree operator for container-level checkpoint/restore
  in Kubernetes.

- **Podman.** Container-level checkpoint/restore with both same-ID and new-ID restore options.

- **JVM CRaC (Coordinated Restore at Checkpoint).** Checkpoint/restore integration with the Java
  runtime that garbage-collects unused memory and closes external resources before checkpoint
  and regenerates required state on restore. Reference pattern for the
  [Application Awareness and Restore Contract](#application-awareness-and-restore-contract).
