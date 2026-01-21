# KEP-5823: Pod-Level Checkpoint/Restore

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation](#implementation)
    - [Accelerating startup of applications with long initialization times](#accelerating-startup-of-applications-with-long-initialization-times)
    - [Enabling fault-tolerance for long-running workloads](#enabling-fault-tolerance-for-long-running-workloads)
    - [Pod migration across nodes for load balancing and maintenance](#pod-migration-across-nodes-for-load-balancing-and-maintenance)
  - [User Stories](#user-stories)
    - [Story 1: Warm-starting a slow-initializing service](#story-1-warm-starting-a-slow-initializing-service)
    - [Story 2: Surviving failures in long-running workloads](#story-2-surviving-failures-in-long-running-workloads)
    - [Story 3: Preserving state across node maintenance](#story-3-preserving-state-across-node-maintenance)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [CRI API Extensions](#cri-api-extensions)
    - [CheckpointPod](#checkpointpod)
    - [RestorePod](#restorepod)
  - [Kubelet Checkpoint and Restore Handling](#kubelet-checkpoint-and-restore-handling)
    - [Checkpoint Handling](#checkpoint-handling)
  - [PodCheckpoint Objects](#podcheckpoint-objects)
    - [PodCheckpoint](#podcheckpoint)
    - [Pod-Snapshot-Controller](#pod-snapshot-controller)
      - [Asynchronous checkpoint flow](#asynchronous-checkpoint-flow)
      - [Scalability follow-up (post-alpha): node-scoped watch](#scalability-follow-up-post-alpha-node-scoped-watch)
  - [Restore Mechanism](#restore-mechanism)
    - [End-to-end restore walkthrough](#end-to-end-restore-walkthrough)
  - [Post-Checkpoint State Semantics](#post-checkpoint-state-semantics)
  - [Checkpoint Content](#checkpoint-content)
    - [Pod Specification and Metadata](#pod-specification-and-metadata)
    - [Container Runtime State](#container-runtime-state)
    - [Shared Pod Resources](#shared-pod-resources)
    - [Checkpoint Storage Location](#checkpoint-storage-location)
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
- [Implementation History](#implementation-history)
- [Open Questions](#open-questions)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Rejected Approaches](#rejected-approaches)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes, i.e., [kubernetes/kubernetes], we require the following Release
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
- [ ] Supporting documentation, e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website
[kubelet checkpoint]: https://kubernetes.io/docs/reference/node/kubelet-checkpoint-api
[OCI image annotations]: https://kubernetes.io/blog/2022/12/05/forensic-container-checkpointing-alpha/#restore-checkpointed-container-k8s
[criu-coordinator]: https://github.com/checkpoint-restore/criu-coordinator
[criu-image-streamer]: https://github.com/checkpoint-restore/criu-image-streamer
[TCP connection repair]: https://lwn.net/Articles/495304/
[KEP-2008]: https://git.k8s.io/enhancements/keps/sig-node/2008-forensic-container-checkpointing
[volume snapshots]: https://kubernetes.io/docs/concepts/storage/volume-snapshots/
[pod-snapshot-controller]: https://github.com/checkpoint-restore/pod-snapshot-controller
[kubernetes/kubernetes#116965]: https://github.com/kubernetes/kubernetes/issues/116965

## Summary

This proposal defines CRI APIs, kubelet support, and controllers together with Kubernetes objects
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
remove this barrier by enabling a Pod-level checkpoint and restore mechanism that is aligned with
the core Kubernetes abstractions.

### Goals

- Introduce Pod-level checkpoint and restore support to the CRI API (`CheckpointPod`, `RestorePod`).
- Add kubelet support to execute Pod-level checkpoints (by watching `PodCheckpoint` objects for the
  Pods it runs) and restores (driven declaratively through `pod.Spec.restoreFrom`); neither uses an
  imperative kubelet HTTP endpoint.
- Define the `PodCheckpoint` object and the Pod-level restore operation.

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
  requires a checkpoint transport mechanism. This planned future functionality is also why a
  restore that cannot yet proceed leaves the Pod `Pending` and is retried rather than failed
  (see [Restore Mechanism](#restore-mechanism)).

- Stopping or releasing the source Pod after a checkpoint (the `Stopped` post-checkpoint
  state). Alpha only supports leaving the source Pod `Running`. Terminating or deleting the
  source Pod re-introduces the terminated-but-not-deleted issues handled by Graceful Node
  Shutdown and its follow-ons (StatefulSet recreation, volume detach, controller replacement
  races), and its only use case, migration, is itself a Non-Goal above. The terminate-vs-delete
  semantics will be designed with SIG Apps and SIG Storage in the migration follow-up (see
  [Post-Checkpoint State Semantics](#post-checkpoint-state-semantics)).

- Checkpoint and restore of shared Pod resources such as shared memory and volumes.

- Checkpoint and restore of devices attached to Pods, including Dynamic Resource Allocation (DRA) claims, device-plugin devices (e.g. NVIDIA GPUs via the device plugin API), and any
associated device memory or driver state. Device support will be addressed in a follow-on KEP.

- Scheduling integration (workload-aware preemption with checkpoint awareness, eviction request
  interceptors).

- Distributed or multi-Pod coordinated checkpointing (e.g., synchronized checkpoint of a
  distributed training job). Requires external coordination tools such as [criu-coordinator].

- Handling of exec sessions, port-forward, and Ephemeral Containers. Support for preserving
  and restoring exec sessions, port-forward, and active Ephemeral Container sessions can be
  explored in a future enhancement proposal.

- Checkpoint portability across heterogeneous environments such as different CPU and GPU
  architectures, kernel versions, container runtimes, or device drivers.

- Checkpoint lifecycle management including resource quotas, limits, retention policies, and
  garbage collection of both checkpoint **data** (on-node archives) and checkpoint **objects**
  (`PodCheckpoint` resources in etcd). This is deferred, not dismissed: it is a substantial
  concern in its own right (per-namespace quotas, retention/TTL, GC triggered by disk pressure,
  bounding the number of `PodCheckpoint` objects, and attributing storage to the owning workload)
  that warrants a dedicated design rather than expanding the initial scope. For the initial
  implementation the interim mitigation is the existing
  [checkpoint-restore operator](https://github.com/checkpoint-restore/checkpoint-restore-operator)
  retention policy; the kubelet also garbage-collects its own partial/aborted checkpoint archives
  (see [Asynchronous checkpoint flow](#asynchronous-checkpoint-flow)). A design discussion for a
  dedicated checkpoint lifecycle management enhancement — covering both archive and object GC — is
  committed as a Beta graduation criterion (see [Beta](#beta)).

- Node-scoped field-selector routing of `PodCheckpoint` objects. For alpha each kubelet watches
  cluster-wide and filters locally by Pod ownership; narrowing the watch with a control-plane-set
  `spec.nodeName` plus a mutating admission plugin is a non-breaking follow-up (see
  [Follow-up: node-scoped routing](#scalability-follow-up-post-alpha-node-scoped-watch)).

- Application-triggered checkpointing. When creating multiple clones from the same checkpoint,
  the workload may need to refresh state such as session keys, random number generator states,
  and certificates. A future KEP will explore a common mechanism for applications to be notified
  of being cloned. For example, gVisor provides a special file (`/proc/gvisor/checkpoint`) that
  blocks until a restore is complete, allowing applications to refresh state on resume.

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

2. **Kubelet checkpoint execution.** The kubelet is the component that performs a checkpoint.
   It **watches `PodCheckpoint` objects** and acts on those whose source Pod it manages (see
   [Pod-Snapshot-Controller](#pod-snapshot-controller)); when it observes a non-terminal object
   for one of its Pods it runs the operation in the background:
   it validates the request, suspends health checks, resolves the CRI sandbox ID, manages
   checkpoint storage at `/var/lib/kubelet/pod-checkpoints/`, captures the source Pod spec, and
   **writes the result back to the `PodCheckpoint` status** itself (see
   [Asynchronous checkpoint flow](#asynchronous-checkpoint-flow)). For restore, the kubelet reads
   Pod sandbox configuration from the checkpoint, assigns a new Pod UID, updates cgroup parent
   paths, and delegates to the container runtime.

3. **API objects** in the `checkpoint.k8s.io` API group that provide declarative management of
   checkpoint operations. `PodCheckpoint` is a namespace-scoped standalone object. The owning
   kubelet finds the object by watching `PodCheckpoint`s and matching `spec.sourcePodName` against
   the Pods it runs, so no control-plane-to-kubelet call is needed. (Node-scoped field-selector
   routing is a follow-up; see
   [Follow-up: node-scoped routing](#scalability-follow-up-post-alpha-node-scoped-watch).)
   Restore is triggered by a new optional `restoreFrom` field on the Pod spec rather than a
   separate object; see [Restore Mechanism](#restore-mechanism). A [pod-snapshot-controller]
   manages `PodCheckpoint` **lifecycle** (finalizers, garbage collection, and — in a follow-on —
   cross-node archive transport); it is **not** on the checkpoint execution path and never
   contacts the kubelet directly.

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

### User Stories

#### Story 1: Warm-starting a slow-initializing service

As an application operator running a service with a long initialization phase (e.g. an LLM
inference server that loads model weights, or a JVM application with a lengthy warm-up), I want
to checkpoint a Pod once it has finished initializing and create new Pods from that checkpoint,
so that subsequent instances become ready in seconds instead of repeating expensive startup
work. See [Accelerating startup of applications with long initialization times](#accelerating-startup-of-applications-with-long-initialization-times).

#### Story 2: Surviving failures in long-running workloads

As a platform engineer running long-lived workloads such as multi-week AI training jobs, I want
to capture a Pod's runtime state and restore it after a failure or preemption, so that the
workload resumes from its last checkpoint rather than restarting from scratch and losing days of
computation. See [Enabling fault-tolerance for long-running workloads](#enabling-fault-tolerance-for-long-running-workloads).

#### Story 3: Preserving state across node maintenance

As a cluster operator performing planned maintenance (kernel upgrades, security patching, node
replacement) or rebalancing workloads, I want to checkpoint a Pod and restore it from that
checkpoint, so that in-memory execution state is preserved across the move and recovery time is
reduced compared to a full Pod restart. See [Pod migration across nodes for load balancing and maintenance](#pod-migration-across-nodes-for-load-balancing-and-maintenance).

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
  Mitigations: (a) scope checkpoint resources as namespace-scoped; (b) drive checkpoint
  execution by having the kubelet watch objects and act on those for its own Pods, so no
  node-proxy privilege is granted to any principal; (c) treat checkpoint
  artifacts as sensitive data with the same handling as Secrets; (d) provide pre-defined
  viewer/editor/admin ClusterRoles for per-namespace binding. See
  [Security Implications](#security-implications).

- Application awareness is required. Checkpoint and restore are not transparent to applications:
  in-memory secrets, tokens, environment variables, and cached hostnames persist through restore,
  and selective memory scrubbing is not feasible. Applications must cooperate for correctness.

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
  detect it. During this window, the kubelet rejects requests to start new Ephemeral Containers
  for the checkpointed Pod. Behavior of any pre-existing Ephemeral Container sessions at
  checkpoint time is out of scope.

- Disruption during the freeze window. Suspending probes stops the kubelet from killing the Pod,
  but a frozen Pod is still unavailable, and neither reporting it not-Ready (StatefulSet/PDB/
  descheduler components may evict or delete it) nor masking it as Ready (Service endpoints
  blackhole traffic) is fully correct. The `Checkpointing=True` condition surfaces the state but
  cannot enforce "do not disrupt / do not route" without ecosystem adoption. This is the same
  class of problem as [kubernetes/kubernetes#116965] and overlaps with EvictionRequest-style
  termination signals; the KEP commits to converging on a common mechanism with SIG-Node and
  SIG-Apps (a [Beta](#beta) graduation criterion). For alpha the mitigation is operational: keep
  the window short and avoid checkpointing Pods under active disruption pressure. See
  [Pod Lifecycle](#pod-lifecycle).

- Disk consumption. Large checkpoint artifacts can consume significant disk space. Size depends
  on the container root filesystem writable layer, the memory usage of running processes at the
  time of checkpointing, and any applied data compression, making precise estimation in advance
  difficult. Checkpoint retention and deletion mechanisms and appropriate storage limits must be
  configured in advance to prevent node disk pressure. A dedicated checkpoint lifecycle
  management enhancement is planned.

- Denial of service via excessive checkpointing. Unrestricted checkpoint operations can exhaust
  node disk space. This risk also applies to the existing container-level checkpoint API. For
  alpha the kubelet cleans up its own partial/aborted archives and clusters can layer on the
  checkpoint-restore operator's retention policy; before Beta the kubelet itself gains in-tree
  garbage collection of checkpoints so node disk safety does not depend on an out-of-tree
  operator (see [Denial of service via excessive checkpointing](#denial-of-service-via-excessive-checkpointing)).

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

// PostCheckpointState selects the state the Pod's processes should be left
// in once the checkpoint image has been written.
enum PostCheckpointState {
    // RUNNING leaves the Pod's processes running after the snapshot has
    // been written ("live snapshot" semantics). This is the default.
    POST_CHECKPOINT_STATE_RUNNING = 0;
    // STOPPED leaves the Pod stopped once the checkpoint is complete. The CRI
    // enum reserves this value so runtimes may implement it ahead of Kubernetes,
    // but in alpha the kubelet only ever sends RUNNING (see Post-Checkpoint State
    // Semantics).
    POST_CHECKPOINT_STATE_STOPPED = 1;
}

message CheckpointPodRequest {
    // ID of the pod sandbox to be checkpointed.
    string pod_sandbox_id = 1;
    // Directory the runtime writes the checkpoint into. A Pod checkpoint is a
    // collection of runtime-defined files (not a single archive object); their
    // layout and format are opaque to Kubernetes. The runtime writes them under
    // this directory and nowhere else (the kubelet owns it for storage accounting
    // and path confinement).
    string path = 2;
    // (No timeout field: the kubelet bounds the operation with the gRPC call
    // deadline, set from PodCheckpoint.spec.timeoutSeconds. The runtime honours
    // the context deadline and cleans up partial artifacts when it fires.)
    //
    // Checkpoint options passed to the container runtime.
    // Reserved for runtime-specific pass-through configuration; behaviour
    // that the CRI itself must branch on belongs in dedicated fields.
    map<string, string> options = 4;
    // State the runtime MUST leave the Pod's processes in after the
    // checkpoint archive has been written. Defaults to RUNNING (the Pod is
    // left running). Runtimes that cannot honour the requested state SHOULD
    // return an error. See Post-Checkpoint State Semantics for the
    // end-to-end contract.
    PostCheckpointState post_checkpoint_state = 5;
}

// Empty: the checkpoint is written under the request's `path` directory, which
// the caller (kubelet) provided and already knows, so there is no separate
// location or object name to return.
message CheckpointPodResponse {}
```

The kubelet bounds the checkpoint by setting the gRPC call deadline from
`PodCheckpoint.spec.timeoutSeconds` (rather than passing a timeout field in the request). When the
deadline fires, the runtime's context is cancelled; the runtime should abort, clean up any
partially created checkpoint artifacts, and return an error. The kubelet handles that error by
cleaning up and recording the failure on the `PodCheckpoint` status as `CheckpointFailed` (see
[Asynchronous checkpoint flow](#asynchronous-checkpoint-flow)).

#### RestorePod

```proto
service RuntimeService {
    ...
    // RestorePod restores a pod sandbox from a checkpoint
    rpc RestorePod(RestorePodRequest) returns (RestorePodResponse) {}
    ...
}

message RestorePodRequest {
    // Directory containing the checkpoint to restore from: the directory the
    // runtime wrote during CheckpointPod (a collection of runtime-defined files).
    string path = 1;
    // Pod sandbox configuration supplied by the kubelet, with node-local restore-time
    // updates (new Pod UID, cgroup parent path, log directory). Pod-spec equality between
    // the live Pod and status.checkpointedPodTemplate of the referenced PodCheckpoint is
    // enforced at API-server admission (and re-checked by the kubelet before this call);
    // arbitrary user overrides are not permitted.
    PodSandboxConfig config = 2;
    // (No timeout field: as with CheckpointPod, the kubelet bounds the operation
    // with the gRPC call deadline rather than a request field.)
    //
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

As with checkpoint, the kubelet bounds the restore with the gRPC call deadline rather than a
request field. When it fires the runtime should abort, clean up any partially restored artifacts,
and return an error. The kubelet cleans up, records the failure as an event on the restore Pod,
and leaves the Pod `Pending` so the restore is retried on the next sync; restore is driven
declaratively by `spec.restoreFrom`, so there is no synchronous caller to return the error to. The
admission, authorization, and pod-template-equality semantics around restore are described in
[Restore Mechanism](#restore-mechanism), not here.

### Kubelet Checkpoint and Restore Handling

Pod-level checkpoint and restore are driven declaratively through the API, not through imperative
kubelet HTTP endpoints. Both operations are gated behind the `PodLevelCheckpointRestore` feature
gate.

#### Checkpoint Handling

There is no imperative checkpoint HTTP endpoint. The kubelet **watches `PodCheckpoint` objects**
and executes the checkpoint when it observes a non-terminal object whose source Pod
(`spec.sourcePodName`) it manages. This mirrors how restore is handled and keeps any privileged
trigger off the user-facing path. (For alpha the kubelet watches cluster-wide and filters
locally by Pod ownership; node-scoped field-selector routing is a follow-up, see
[Follow-up: node-scoped routing](#scalability-follow-up-post-alpha-node-scoped-watch).)

The kubelet's checkpoint handling (the canonical execution flow referenced elsewhere in this KEP):
1. Selects `PodCheckpoint` objects whose `spec.sourcePodName` resolves to a Pod present on this
   node and whose object is not in a terminal state (`Ready=True`, or `Ready=False` with reason
   `CheckpointFailed`/`SourcePodReplaced`).
2. Acquires a per-Pod in-flight guard so a re-observed or duplicate object does not start a second
   checkpoint and so a checkpoint never overlaps a restore on the same Pod.
3. Validates Pod readiness (bound to the node, all non-restartable init containers completed,
   regular containers and restartable sidecars running). This execution-time gate is the kubelet's
   authority and is separate from the API-server RBAC gate on the object (see
   [Security Implications](#security-implications)).
4. Pins the source instance by comparing the live Pod UID with `spec.sourcePodUID`. If they differ
   the original instance was replaced; the kubelet fails the checkpoint with `Ready=False`, reason
   `SourcePodReplaced`, rather than checkpointing the new instance, and records the resolved UID in
   `status.sourcePodUID`.
5. Requests the `RUNNING` post-checkpoint state from the CRI. Alpha always leaves the source Pod
   running, so this is fixed; the `Stopped` behavior and its user-facing field are deferred to the
   migration follow-up (see [Post-Checkpoint State Semantics](#post-checkpoint-state-semantics)).
6. Captures the source Pod's metadata and spec, strips node-local and cluster-specific fields (see
   [Pod Specification and Metadata](#pod-specification-and-metadata)), and writes the result to
   `status.checkpointedPodTemplate` for the spec-equality check used on restore (see
   [Restore Mechanism](#restore-mechanism)). The kubelet reads the live Pod object directly.
7. Suspends the Pod's probes, resolves the CRI sandbox ID, and calls the `CheckpointPod` CRI API
   in the background, writing the archive under the kubelet's checkpoint root (for example
   `/var/lib/kubelet/pod-checkpoints/checkpoint-{podName}_{namespace}-{timestamp}`). It records the
   location in `status.checkpointLocation` as the node-local source (`type: NodeLocal` with
   `nodeLocal.path` relative to that root), not as an absolute host path.
8. On completion, writes the result to the `PodCheckpoint` status (see
   [Asynchronous checkpoint flow](#asynchronous-checkpoint-flow)): `Ready=True`/`CheckpointCompleted`
   with `checkpointLocation` on success, or `Ready=False`/`CheckpointFailed` with a reason on
   failure.

Restore is likewise declarative. There is no restore HTTP endpoint: restore is driven through
`pod.Spec.restoreFrom` and the kubelet's normal `SyncPod` path (see
[Restore Mechanism](#restore-mechanism)); the kubelet swaps sandbox creation for sandbox restore
when it observes the field, so no imperative restore call to the kubelet is needed.

### PodCheckpoint Objects

To provide declarative management of checkpoint operations, this KEP introduces a new
built-in Kubernetes API type, `PodCheckpoint`, in the `checkpoint.k8s.io/v1alpha1` API group.
`PodCheckpoint` is a first-class Kubernetes resource (not a CRD); it is served by the API
server alongside core types such as `Pod` and `Node`. The design follows the Kubernetes
[volume snapshots] pattern: a checkpoint is a standalone object with its own lifecycle that
can outlive the source Pod and be used to create multiple new Pods. The restore side makes
use of a new `restoreFrom` field on Pod spec described below.

#### PodCheckpoint

A `PodCheckpoint` object triggers a checkpoint of a named Pod. The kubelet that runs the named
source Pod watches for these objects, acts on the ones whose source Pod it manages, performs the
checkpoint, and records the result on the status (see
[Asynchronous checkpoint flow](#asynchronous-checkpoint-flow)).

`PodCheckpoint` supports two field selectors so checkpoints can be listed by the objects they
relate to: `spec.sourcePodName` (all checkpoints of a given source Pod) and `status.nodeName` (all
checkpoints whose data resides on a given node). These are registered as selectable fields on the
REST storage, the same way `Pod` exposes `spec.nodeName`/`status.phase`. Adding a selectable field
is backward-compatible, so further selectors can be added later without an incompatible change.

The Go types (served from `staging/src/k8s.io/api/checkpoint/v1alpha1/types.go`):

```go
// PodCheckpoint represents a request to checkpoint a running Pod, together
// with the resulting checkpoint metadata.
type PodCheckpoint struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the checkpoint to be taken. It is immutable after creation.
	Spec PodCheckpointSpec `json:"spec"`

	// status reflects the observed state of the checkpoint operation. It is
	// written by the kubelet that owns the Pod and is read-only for users.
	// +optional
	Status PodCheckpointStatus `json:"status,omitempty"`
}

// PodCheckpointList is a list of PodCheckpoint objects.
type PodCheckpointList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PodCheckpoint `json:"items"`
}

// PodCheckpointSpec describes which Pod to checkpoint and how.
type PodCheckpointSpec struct {
	// sourcePodName is the name of the running Pod to checkpoint. The Pod must
	// exist in the same namespace as this PodCheckpoint. Immutable. Required in
	// alpha (validation rejects an empty value); it is marked optional in the
	// schema so a future selector-based or controller-populated mode (for example
	// checkpointing a ReplicaSet replica without naming one) can relax it without
	// an incompatible API change.
	// +optional
	SourcePodName string `json:"sourcePodName,omitempty"`

	// sourcePodUID, if set, pins the checkpoint to a specific Pod instance: the
	// kubelet checkpoints the Pod only if the live Pod named sourcePodName has
	// this exact UID, and fails the checkpoint otherwise (reason
	// SourcePodReplaced). Because a Pod name can be reused (the original Pod may
	// be deleted and a new Pod created with the same name), a name alone does not
	// identify an instance. Callers that need instance pinning set this field when
	// creating the PodCheckpoint, so the instance is fixed across the window
	// between creation and the kubelet acting on it; without it a same-name
	// replacement could be checkpointed by mistake. A future enhancement may add
	// admission-time defaulting to populate it automatically from the named Pod.
	// Immutable.
	// +optional
	SourcePodUID *types.UID `json:"sourcePodUID,omitempty"`

	// timeoutSeconds is the maximum time the checkpoint operation may take.
	// A nil value or 0 means the container runtime default is used.
	// +optional
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty"`
}

// Note: alpha leaves the source Pod running after a checkpoint, so the kubelet
// always requests the RUNNING post-checkpoint state from the CRI. The user-facing
// choice (a postCheckpointState field on PodCheckpointSpec) is intentionally not
// added to the API yet; it will be introduced together with the "Stopped"
// behavior in the migration follow-up, when it is actually used. The CRI enum
// (CheckpointPodRequest.post_checkpoint_state) reserves the value ahead of that so
// runtimes can implement it. See Post-Checkpoint State Semantics.

// PodCheckpointStatus reports the observed state of the checkpoint operation.
// (There is no top-level observedGeneration: the spec is immutable, so the
// object's generation never advances, and each condition already carries its own
// observedGeneration.)
type PodCheckpointStatus struct {

	// nodeName is the node where the source Pod was running when checkpointed
	// and where the checkpoint data resides.
	// +optional
	NodeName string `json:"nodeName,omitempty"`

	// sourcePodUID is the UID of the Pod instance the kubelet actually
	// checkpointed (or is checkpointing). It is recorded when the kubelet picks
	// up the object for visibility and so that a later UID change for the same
	// name is detected and fails the checkpoint. This guards only changes observed
	// once the kubelet acts; to also cover the window before it picks up the
	// object, set spec.sourcePodUID at creation time.
	// +optional
	SourcePodUID *types.UID `json:"sourcePodUID,omitempty"`

	// checkpointLocation describes where the checkpoint data is stored. It is a
	// discriminated union keyed by type so checkpoint storage can grow to other
	// backends (object storage, a PersistentVolumeClaim) by adding members,
	// without an incompatible change. The kubelet sets it when the checkpoint is
	// Ready; in alpha the only backend is the node that took the checkpoint
	// (NodeLocal). See Checkpoint Storage Location.
	// +optional
	CheckpointLocation *CheckpointSource `json:"checkpointLocation,omitempty"`

	// completionTime is the time the checkpoint completed (the archive was
	// written and the checkpoint became Ready), set by the kubelet. It is the
	// time the captured state corresponds to, and is used for freshness and for
	// retention/GC (for example deleting checkpoints older than a threshold). It
	// is distinct from metadata.creationTimestamp, which is when the
	// PodCheckpoint object was created.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// checkpointedPodTemplate is the sanitized Pod template (metadata + spec)
	// captured from the source Pod at checkpoint time. It is the authoritative
	// record a restore Pod's spec is validated against. Node-local and
	// cluster-specific fields (e.g. nodeName, status, uid, resourceVersion,
	// managedFields) are excluded so the template stays portable.
	// +optional
	CheckpointedPodTemplate *core.PodTemplateSpec `json:"checkpointedPodTemplate,omitempty"`

	// checkpointedContainers lists the checkpointed regular (non-init) containers
	// as a visibility convenience; the authoritative set is in
	// checkpointedPodTemplate. Named to parallel checkpointedPodTemplate, since
	// these describe the checkpointed Pod, not the PodCheckpoint object.
	// +optional
	// +listType=map
	// +listMapKey=name
	CheckpointedContainers []PodCheckpointContainerStatus `json:"checkpointedContainers,omitempty"`

	// checkpointedInitContainers lists the checkpointed init containers, kept
	// separate from checkpointedContainers to mirror PodStatus. It records the
	// completed non-restartable init containers and any running restartable init
	// containers (sidecars). On restore, completed init containers are reflected as
	// completed and are not re-run; running sidecars are restored running and
	// remain restartable init containers.
	// +optional
	// +listType=map
	// +listMapKey=name
	CheckpointedInitContainers []PodCheckpointContainerStatus `json:"checkpointedInitContainers,omitempty"`

	// conditions represents the latest observations of the checkpoint's state.
	// The "Ready" condition is the single source of truth for checkpoint
	// progress (see reason constants below).
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// PodCheckpointContainerStatus identifies a container captured in the checkpoint.
type PodCheckpointContainerStatus struct {
	// name of the checkpointed container.
	Name string `json:"name"`
	// image the container was running at checkpoint time.
	Image string `json:"image"`
}

// CheckpointSource describes where a checkpoint's data is stored. It is a
// discriminated union: the member matching type is set. New backends are added
// as new members (and new type values) without an incompatible change.
// +union
type CheckpointSource struct {
	// type indicates which backend holds the checkpoint data. In alpha the only
	// value is "NodeLocal".
	// +unionDiscriminator
	Type CheckpointSourceType `json:"type"`

	// nodeLocal is set when type is "NodeLocal": the checkpoint is stored on the
	// node that took it (status.nodeName).
	// +optional
	NodeLocal *NodeLocalCheckpointSource `json:"nodeLocal,omitempty"`
}

// CheckpointSourceType enumerates the checkpoint storage backends.
// +enum
type CheckpointSourceType string

const (
	// CheckpointSourceTypeNodeLocal stores the checkpoint on the node that took
	// it. It is the only backend implemented in alpha.
	CheckpointSourceTypeNodeLocal CheckpointSourceType = "NodeLocal"
)

// NodeLocalCheckpointSource locates a checkpoint stored on the node that took it.
type NodeLocalCheckpointSource struct {
	// path is the location of the checkpoint data relative to the kubelet's
	// configured checkpoint root directory on the node; it is not an absolute
	// host path. The kubelet resolves it against its root on restore and rejects
	// any path that escapes the root.
	Path string `json:"path"`
}

// PodCheckpointReady is the type of the summary condition on a PodCheckpoint.
const PodCheckpointReady = "Ready"

// Reasons for the "Ready" condition (the single source of truth for checkpoint progress):
//   Pending                -> status: "False"
//   CheckpointInProgress   -> status: "False"
//   CheckpointCompleted    -> status: "True"
//   CheckpointFailed       -> status: "False" (message carries detail)
//   SourcePodReplaced      -> status: "False" (live Pod's UID != spec.sourcePodUID)
const (
	PodCheckpointReasonPending           = "Pending"
	PodCheckpointReasonInProgress        = "CheckpointInProgress"
	PodCheckpointReasonCompleted         = "CheckpointCompleted"
	PodCheckpointReasonFailed            = "CheckpointFailed"
	PodCheckpointReasonSourcePodReplaced = "SourcePodReplaced"
)
```

Example object:

```yaml
apiVersion: checkpoint.k8s.io/v1alpha1
kind: PodCheckpoint
metadata:
  name: my-checkpoint
spec:
  # Name of the running Pod to checkpoint.
  sourcePodName: my-app
  # Optional: pin to a specific Pod instance. If set, the checkpoint fails
  # (reason SourcePodReplaced) unless the live Pod named above has this UID,
  # so a recreated same-name Pod is never checkpointed by mistake.
  sourcePodUID: 7b2c1e4a-0e3a-4f1b-9c2d-2a5f6e8d1234
  # Optional timeout in seconds (0 = use container runtime default).
  timeoutSeconds: 30
  # Note: alpha always leaves the source Pod running. A user-facing
  # postCheckpointState field is not part of the API yet; it arrives with the
  # "Stopped" behavior in the migration follow-up.
status:
  # Node where the source Pod was running when checkpointed.
  nodeName: node-1
  # UID of the Pod instance that was actually checkpointed (recorded by the
  # kubelet; a later UID change for the same name fails the checkpoint).
  sourcePodUID: 7b2c1e4a-0e3a-4f1b-9c2d-2a5f6e8d1234
  # Where the checkpoint data is stored. A discriminated union so storage can
  # grow to other backends later; alpha only sets the node-local backend, whose
  # path is relative to the kubelet's checkpoint root on the node.
  checkpointLocation:
    type: NodeLocal
    nodeLocal:
      path: checkpoint-my-app_default-2026-03-10T20:38:11Z
  # Time the checkpoint completed (archive written / became Ready), set by the
  # kubelet. Used for freshness and retention/GC; distinct from
  # metadata.creationTimestamp (when the PodCheckpoint object was created).
  completionTime: "2026-03-10T20:38:12Z"
  # Sanitized Pod template (metadata + spec) captured from the source Pod at
  # checkpoint time. This is the authoritative record that a restore Pod's
  # spec is validated against, by the API server at admission time and by the
  # kubelet before the CRI restore call. The kubelet populates it; it is
  # part of status and is therefore immutable to users. Node-local and
  # cluster-specific fields (nodeName, nodeSelector entries referencing
  # internal nodes, status, uid, resourceVersion, managedFields) are excluded
  # so the template stays portable.
  checkpointedPodTemplate:
    metadata:
      labels:
        app: my-app
      annotations: {}
    spec:
      containers:
      - name: main
        image: my-app:latest
      # ...remaining scheduling constraints, resource requirements, and
      # security contexts captured from the source Pod.
  # Regular (non-init) containers captured in the checkpoint (visibility
  # convenience; the full set is in checkpointedPodTemplate).
  checkpointedContainers:
  - name: main
    image: my-app:latest
  # Init containers captured in the checkpoint, kept separate to mirror PodStatus:
  # completed non-restartable init containers and any running sidecars.
  checkpointedInitContainers:
  - name: setup
    image: my-app-init:latest
  # The "Ready" condition is the single source of truth for checkpoint state.
  # Its status/reason/message carry the checkpoint progress detail:
  #   pending      -> status: "False", reason: Pending
  #   in progress  -> status: "False", reason: CheckpointInProgress
  #   ready         -> status: "True",  reason: CheckpointCompleted
  #   failed       -> status: "False", reason: CheckpointFailed (message has detail)
  conditions:
  - type: Ready
    status: "True"
    reason: CheckpointCompleted
    message: "checkpoint archive written successfully"
    observedGeneration: 1
    lastTransitionTime: "2026-03-10T20:38:12Z"
```

#### Pod-Snapshot-Controller

The pod-snapshot-controller ships in-tree as part of `kube-controller-manager` and manages the
**lifecycle** of `PodCheckpoint` objects (a standalone prototype is maintained out-of-tree at
[pod-snapshot-controller]). It is deliberately **out of the checkpoint execution path**: it does
not contact the kubelet and never blocks on a checkpoint operation. The kubelet observes
`PodCheckpoint` objects directly and finalizes them on `status` (see
[Asynchronous checkpoint flow](#asynchronous-checkpoint-flow)). This watch-based execution is
consistent with how the rest of Kubernetes works — no controller issues a direct request to a
kubelet — and removes the node-proxy round trip and its throughput bottleneck.

```mermaid
sequenceDiagram
    actor User as User / workload controller
    participant API as kube-apiserver
    participant KCM as pod-snapshot-controller
    participant Kubelet as kubelet (owns the Pod)
    participant CRI as container runtime

    rect rgb(245,245,245)
    Note over User,CRI: Checkpoint
    User->>API: create PodCheckpoint (sourcePodName, optional sourcePodUID)
    API-->>Kubelet: watch event (kubelet matches sourcePodName to a local Pod)
    Kubelet->>Kubelet: validate readiness, pin sourcePodUID,<br/>suspend probes, capture checkpointedPodTemplate
    Kubelet->>CRI: CheckpointPod(sandboxID, RUNNING)
    CRI-->>Kubelet: archive written
    Kubelet->>API: status Ready=True/CheckpointCompleted, nodeName=self<br/>(NodeRestriction: source Pod must be on this node)
    KCM-->>API: watch for lifecycle only (finalizers, GC) — off this path
    end

    rect rgb(245,245,245)
    Note over User,CRI: Restore
    User->>API: create Pod (spec.restoreFrom = checkpoint name)
    API->>API: authorize "restore" verb; inject nodeAffinity=status.nodeName;<br/>validate pod-template equality (authoritative)
    API-->>API: scheduler binds Pod to status.nodeName (node affinity)
    API-->>Kubelet: Pod assigned; SyncPod observes spec.restoreFrom
    Kubelet->>API: read PodCheckpoint (location, template);<br/>re-validate equality (defense in depth)
    Kubelet->>CRI: RestorePod(...)
    CRI-->>Kubelet: sandbox + containers restored
    end
```

Object routing is by Pod ownership. Each kubelet watches `PodCheckpoint` objects and acts only on
those whose `spec.sourcePodName` resolves to a Pod it currently runs; objects for Pods on other
nodes are ignored. The creator may set `spec.sourcePodUID` to pin a specific instance (see below).
For alpha the kubelet watches cluster-wide and filters locally — `PodCheckpoint` objects are
low-volume, short-lived request objects, so this is acceptable; narrowing the watch with a
node-scoped field selector is a non-breaking follow-up (see
[Follow-up: node-scoped routing](#scalability-follow-up-post-alpha-node-scoped-watch)).

The kubelet's checkpoint execution flow — selecting objects for its own Pods, pinning the source
instance, capturing the template, suspending probes, calling the CRI, and finalizing `status` — is
the canonical list in [Checkpoint Handling](#checkpoint-handling). The point relevant here is that
it runs entirely on the kubelet (a per-Pod in-flight guard de-duplicates overlapping work and keeps
a checkpoint and a restore from running on the same Pod at once); the controller is not involved at
any step.

The controller's responsibilities are lifecycle only: managing the restore-lock finalizer,
garbage-collecting `PodCheckpoint` objects (see
[Denial of service via excessive checkpointing](#denial-of-service-via-excessive-checkpointing)),
and — in a follow-on enhancement — copying archives between nodes for cross-node restore.

Restore does not require controller involvement either: the kubelet drives restore directly from
the Pod spec (see [Restore Mechanism](#restore-mechanism)).

##### Asynchronous checkpoint flow

A checkpoint can take minutes (it scales with the workload's in-memory footprint), so execution
is decoupled from any client: there is no synchronous trigger to hold open.

- **Dispatch (object → kubelet).** Each kubelet watches `PodCheckpoint` objects and acts on a
  non-terminal object whose source Pod it runs. Observing such an object *is* the trigger; no
  control-plane component calls the kubelet. The kubelet starts the checkpoint in the background
  and a single watch event never ties up a worker for the length of the operation.
- **Result (kubelet → API server).** The kubelet performs the checkpoint in the background and
  writes the terminal outcome (including `status.nodeName=<its node>`) to the named
  `PodCheckpoint` status itself. The kubelet's `system:node` role grants `update`/`patch` on
  `podcheckpoints/status` (the Node authorizer permits the write via this rule), and the
  `NodeRestriction` admission plugin scopes it: it allows the write only when the checkpoint's
  source Pod (`spec.sourcePodName`) is bound to the requesting node, reusing the same node↔Pod
  relationship that already limits a kubelet to writing its own Pods' status. A kubelet therefore
  cannot finalize a checkpoint for a Pod it does not run. See [Privilege model](#privilege-model).

Restart and idempotency semantics:

- **Controller restart mid-operation.** Irrelevant to in-flight checkpoints — the controller is
  not on the execution path. The result is written to the object by the kubelet regardless of
  controller state; lifecycle reconciliation (finalizers, GC) simply resumes from the object
  state on restart.
- **Kubelet restart mid-operation.** The CRI checkpoint is not resumable. On startup the kubelet
  garbage-collects any partial archive and finalizes the `PodCheckpoint` as `Ready=False` with
  reason `CheckpointFailed` (the in-flight guard does not survive the restart), so the object
  does not hang in `CheckpointInProgress` and can then be retried.

##### Scalability follow-up (post-alpha): node-scoped watch

This is post-alpha optimization work, not part of this KEP's alpha; it is included here only so the
alpha's cluster-wide watch has a documented, non-breaking path to scale (it is a
[Non-Goal](#non-goals) for now).

For alpha each kubelet watches `PodCheckpoint` objects cluster-wide and filters locally by Pod
ownership (above). Because these objects are low-volume and short-lived this is acceptable, but in
a large cluster every kubelet receives every `PodCheckpoint`. A future enhancement narrows each
kubelet's watch to its own node. It is purely additive and introduces no breaking change:

- Add an optional, control-plane-set `spec.nodeName` to `PodCheckpoint` (immutable; this mirrors
  how kubelets already watch Pods by `spec.nodeName`). Objects created before the field exists
  simply lack it.
- A mutating admission plugin resolves `spec.sourcePodName` at create, sets `spec.nodeName` from
  the Pod's node (and `spec.sourcePodUID` from its UID), and rejects the create if the Pod is
  missing or unscheduled.
- Register `spec.nodeName` as a selectable field so each kubelet can watch with
  `spec.nodeName=<its node>`. The kubelet keeps the local Pod-ownership check as the source of
  truth, so an object that predates the field (empty `spec.nodeName`) still works via the
  cluster-wide fallback during the transition.
- `NodeRestriction` can then scope the kubelet's status write by `spec.nodeName` directly instead
  of resolving the source Pod.

The admission plugin and the field-selector watch ship together under the same feature gate, so
there is no window in which a narrowed watch would miss an un-annotated object.

### Restore Mechanism

Restore is triggered by a new optional field on Pod spec rather than by a separate API object.
A user creates a Pod with `spec.restoreFrom` set to the name of a `PodCheckpoint` object
in the same namespace. The kubelet observes this during `SyncPod` and calls `restorePodSandbox()`
instead of `createPodSandbox()`. Pod creation is the restore, in a single step.

This shape reuses the normal Pod admission and scheduling path: admission authorizes the restore
and injects a node-affinity constraint targeting the checkpoint's node, the **scheduler** places
the Pod on that node, the CNI plugin sets up networking against the Pod object exactly as it would
for a fresh Pod, and the kubelet swaps the sandbox creation step for sandbox restore. The Pod is
scheduled like any other Pod — admission does **not** bind it directly with `spec.nodeName` — so
the scheduler is the component that reasons about node capability and placement feasibility (and,
via Node Declared Features, can avoid nodes that do not support restore; see
[Version Skew Strategy](#version-skew-strategy)). This keeps the layering right: the kubelet is not
the first component to discover that the chosen node cannot satisfy the restore. No placeholder
Pod, no separate object lifecycle, and no `nodes/proxy` permission for restore.

`spec.restoreFrom` is a name reference. After API-server admission (which authorizes the
requester for the `restore` verb on the referenced `PodCheckpoint`, injects a node-affinity
constraint pinning the Pod to the checkpoint's node, and validates pod-template equality against
`status.checkpointedPodTemplate`) and after the scheduler binds the Pod to that node, the kubelet
reads the `PodCheckpoint` object
to resolve the checkpoint location. `status.nodeName` identifies the node holding the data and
`status.checkpointLocation.nodeLocal.path` is the path relative to the kubelet's checkpoint root;
the kubelet resolves it against its own root to read the archive. The kubelet rejects the restore
if its own node does not match `status.nodeName` (cross-node restore is currently out of scope).

The two checks have distinct roles. The API server enforces access control and pod-spec equality
at admission. The kubelet runs the equality check again before the CRI restore as a safeguard.

1. **API-server admission.** When `spec.restoreFrom` is set, the `PodRestoreAuthorization`
   admission plugin does three things. It authorizes the `restore` verb on the referenced
   `PodCheckpoint`. It injects a required node affinity targeting the checkpoint's node
   (`status.nodeName`), so the scheduler places the Pod there rather than the API server binding it
   directly. And it checks that the Pod's spec matches the spec in `status.checkpointedPodTemplate`,
   rejecting a mismatch and reporting the field that differs.

   The equality check ignores the fields the restore flow introduces: `spec.restoreFrom` (the
   trigger the source Pod never had) and the node placement it adds (the injected node affinity, and
   the `spec.nodeName` the scheduler sets when it binds the Pod). The plugin already reads the
   `PodCheckpoint` to build the affinity, so the template is on hand, and the spec is already
   defaulted at admission, so the comparison is straightforward. Checking it here rejects a
   mismatched Pod at creation, with the offending field reported to the user, instead of admitting a
   Pod the node would only reject later.

   Admission can compare only once the checkpoint is `Ready` and its template is populated, which
   is the normal case. If a Pod is admitted against a checkpoint that is not `Ready` yet, the
   kubelet runs the equality check when it acts (see below).
2. **Kubelet, before the CRI restore.** The kubelet validates the live Pod's spec against
   `status.checkpointedPodTemplate` again, with the same two exemptions, and rejects a mismatch
   before calling `RestorePod`. This guards the window between admission and execution, so the
   kubelet never restores against a spec it has not checked itself. The kubelet compares against
   the object field rather than parsing the opaque checkpoint archive, which is owned entirely by
   the container runtime.

#### End-to-end restore walkthrough

This example traces a single restore from a `Ready` `PodCheckpoint` through to a restored
Pod in `Running` state. It assumes the `PodLevelCheckpointRestore` feature gate is
enabled (on `kube-apiserver`, `kube-controller-manager`, and the target node's `kubelet`)
and the container runtime implements the `RestorePod` CRI RPC.

**Pre-conditions.** A `PodCheckpoint` named `myapp-snapshot-01` exists in namespace `team-a`
with its `Ready` condition set to `True`, recording the node on which the source Pod was
checkpointed and the on-node archive path:

```yaml
apiVersion: checkpoint.k8s.io/v1alpha1
kind: PodCheckpoint
metadata:
  name: myapp-snapshot-01
  namespace: team-a
status:
  nodeName: node-1
  checkpointLocation:
    type: NodeLocal
    nodeLocal:
      path: checkpoint-myapp_team-a-2026-05-28T10:14:22Z
  checkpointedPodTemplate:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
      - name: app
        image: registry.example.com/myapp:v1.4.0
      # ...scheduling constraints, resources, and security contexts as captured.
  conditions:
  - type: Ready
    status: "True"
    reason: CheckpointCompleted
    message: "checkpoint archive written successfully"
    observedGeneration: 1
```

**Step 1 - User submits restore request.** The user applies a Pod manifest with
`spec.restoreFrom` set to the checkpoint name. The user does **not** set `spec.nodeName` —
admission injects the node-affinity constraint in Step 2 and the scheduler places the Pod. The
Pod's spec must match `status.checkpointedPodTemplate` of `myapp-snapshot-01` (admission enforces
this in Step 2, and the kubelet re-checks in Step 5):

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp-restored
  namespace: team-a
spec:
  restoreFrom: myapp-snapshot-01
  # No nodeName: admission injects a required node affinity for the checkpoint's
  # node and the scheduler binds the Pod there.
  containers:
  - name: app
    image: registry.example.com/myapp:v1.4.0
    # ...rest of spec must match the spec inside myapp-snapshot-01
```

**Step 2 - API server admission.** The API server validates the Pod spec as usual. Because
`spec.restoreFrom` is set, the API server additionally:

- Issues a `SubjectAccessReview` for the `restore` verb on
  `podcheckpoints/myapp-snapshot-01` in namespace `team-a` against the requester's identity.
  Failure rejects the create with `Forbidden`. This is the verb split described in
  [Privilege model](#privilege-model): `create` on `PodCheckpoint` gates checkpoint creation;
  the dedicated `restore` verb on the referenced object gates restore.
- Injects a required node affinity targeting `myapp-snapshot-01.status.nodeName` (`node-1`) — a
  `nodeSelectorTerm` with `matchFields: [{key: metadata.name, operator: In, values: [node-1]}]`.
  If the user already set a conflicting `spec.nodeName` or node affinity, the create is rejected
  with `Forbidden`. This constrains placement to the checkpoint's node as a scheduling requirement,
  rather than binding the Pod directly.
- Checks that the incoming Pod's spec matches the spec in
  `myapp-snapshot-01.status.checkpointedPodTemplate`, ignoring the restore-introduced fields
  (`spec.restoreFrom`, the injected node affinity, and the `spec.nodeName` the scheduler later
  sets). A mismatch rejects the create with `Invalid` and names the field that differs. The kubelet
  checks this again in Step 5.

All three checks are done by the `PodRestoreAuthorization` admission plugin.

The Pod is persisted with a new Pod UID; the original checkpointed Pod's UID is not reused.

**Step 3 - Scheduling.** The Pod enters the scheduling queue like any other Pod. The injected
node affinity constrains it to `node-1` (the node holding the checkpoint), so the scheduler binds
it there — setting `spec.nodeName` — after checking the node is feasible (resources, taints, and,
where available, the node's declared restore capability). If `node-1` cannot accommodate the Pod,
it stays `Pending` with a normal scheduler `Unschedulable` reason, rather than being bound to a
node that later cannot satisfy the restore.

**Step 4 - Kubelet observes the Pod.** The kubelet on `node-1` receives the Pod via the
standard watch path. Admission, volume setup, and CNI network setup against the Pod object
are unchanged from a non-restored Pod; the restore-specific logic is confined to sandbox
creation.

**Step 5 - Kubelet validation gates.** In `SyncPod`, the kuberuntime manager observes
`pod.Spec.RestoreFrom != nil` and routes to `restorePodSandbox()`. Before issuing the
`RestorePod` CRI call, the kubelet enforces, in order:

1. **Checkpoint resolution.** The kubelet reads the `PodCheckpoint` `myapp-snapshot-01` and pulls
   `status.nodeName` and `status.checkpointLocation`. A missing or non-`Ready`
   `PodCheckpoint` fails with event reason `CheckpointNotReady` and the Pod stays in
   `Pending`.
2. **Node match.** The kubelet rejects the restore unless it is running on
   `status.nodeName`. If the originally-checkpointed Pod has since moved, restore still
   targets the node where the checkpoint data lives, not the Pod's current location.
   Cross-node restore is out of scope for alpha and is rejected with reason
   `CheckpointWrongNode`.
3. **Pod-spec equality.** The kubelet compares the live Pod's spec with
   `myapp-snapshot-01.status.checkpointedPodTemplate` again, ignoring the same restore-introduced
   fields (`spec.restoreFrom`, the injected node affinity, and the scheduler-assigned
   `spec.nodeName`). A mismatch fails with `PodSpecMismatch` and names the field in the
   event message. Admission already checked this in Step 2; the kubelet repeats it so it never
   restores against a spec it has not confirmed, and to cover the case where the checkpoint became
   `Ready` only after the Pod was admitted.

Alongside these checks, the kubelet takes a per-Pod restore lock keyed by the Pod's
`(namespace, name)` rather than its UID (see [Privilege model](#privilege-model)). This is an
in-memory, node-local lock, separate from the `PodCheckpoint`'s API-level restore-lock finalizer.
A UID key would never collide, because each restore attempt has a fresh UID and the per-UID pod
worker already serializes its own syncs. The identity that needs serializing is the
`(namespace, name)` the sandbox is created under, which is unique at any moment.

The lock matters in one narrow case: a restoring Pod is deleted and a new Pod with the same name
(a new UID, possibly a different checkpoint) starts restoring before the first restore finishes.
The second restore finds the lock held, stays `Pending` with `Restoring=False` and reason
`RestoreInProgress` (and an event of the same reason), and retries on the next sync. It proceeds
once the first restore releases the lock. Reusing a name over time never contends, because the
lock is released when each restore finishes and two Pods with the same `(namespace, name)` cannot
exist at once.

**Step 6 - `RestorePod` CRI call.** The kubelet generates the sandbox config from the Pod
object exactly as for a fresh Pod (log directory, cgroup parent, CNI annotations), with
node-local fields overridden at restore time. It then calls `RestorePod` on the container
runtime with the checkpoint path resolved from `status.checkpointLocation.nodeLocal.path` against
its checkpoint root, the sandbox config, and
per-container `ContainerConfig` entries carrying mount information (`/etc/hosts`,
termination log paths, and any volumes already supported by the runtime). The runtime
restores the sandbox and all containers from the archive, attaches the network namespace
via CNI, and returns the new sandbox ID. The normal `SyncPod` container start steps
(`startContainer` for init, regular, and ephemeral containers) are skipped: the restored
containers are already running inside the restored sandbox.

**Step 7 - Status converges.** The kubelet updates Pod status:

- `status.phase` transitions `Pending` to `Running`.
- The `Restoring=True` condition is cleared once the sandbox is up and container statuses
  are `Running`.
- An event `RestoreSucceeded` is recorded on the Pod.
- Container `restartCount` continues from the value captured in the checkpoint.

The Pod is now indistinguishable from any other `Running` Pod for controllers, schedulers,
and monitoring tooling. `spec.restoreFrom` remains on the Pod as a record of provenance and
is ignored by subsequent `SyncPod` invocations once containers are `Running`.

**Failure rollback.** A failed restore is a failed sandbox creation: the kubelet swapped
`createPodSandbox` for `restorePodSandbox`, so the same handling applies. If any step from 5 to 6
fails after the restore serialization lock is acquired, the kubelet releases the lock, the
container runtime cleans up any partial sandbox, and the kubelet records a Pod event with one of
the reasons above and a `Restoring=False` condition. The Pod stays `Pending` and the kubelet
retries with backoff, the same as it does for `FailedCreatePodSandBox` or `ImagePullBackOff`; it
is not moved to `Failed`.

A restore that cannot proceed yet leaves the Pod `Pending` and is retried; it is not failed
outright. The referenced checkpoint may not be `Ready` yet, or (once cross-node transfer lands)
its data may not be on the node yet but could be copied there later. In both cases the same Pod
proceeds once the checkpoint becomes available, with no need to resubmit it.

This follows the usual Kubernetes pattern of declaring intent and letting the dependency be
satisfied out of order. A Pod that names a ConfigMap, Secret, or PersistentVolumeClaim that does
not exist yet is admitted and waits rather than being rejected, and a PersistentVolumeClaim may
name a `VolumeSnapshot` as its data source before that snapshot is ready to use. Restoring from a
`PodCheckpoint` works the same way: the Pod is admitted, and the kubelet waits for the checkpoint
and validates it when it acts. A restore that never succeeds is retried with backoff like any
other start failure, rather than being moved to `Failed`, and stays visible through the Pod's
events and conditions.

### Post-Checkpoint State Semantics

The post-checkpoint state selects what happens to the source Pod once the archive has been
written. In the CRI it is a typed enum (rather than a boolean or an `options` key) so the runtime
and kubelet can branch on it without parsing opaque pass-through configuration, and so additional
states can be added later. Alpha always uses `Running`, and the Kubernetes API does not expose the
choice yet.

- **Running (default).** After the archive is written, the runtime resumes execution
  of all processes in the Pod and the containers continue running. This is the right mode
  for warm start, snapshotting, and most fault-tolerance flows, where the source workload
  should keep serving while the archive is in storage. `Running` is **the only mode
  implemented in alpha**, and it matches the existing container-level checkpoint API
  behaviour: after a successful checkpoint the source container keeps running.
- **Stopped (reserved; not implemented in alpha).** The intent of `Stopped` is that, after
  the archive is written, the source Pod is not resumed but instead released so a restore
  can take over elsewhere. This is a migration concern, and **cross-node restore and live
  migration are Non-Goals for alpha** (see [Non-Goals](#non-goals)). The value is defined in
  the CRI enum for forward compatibility, but in alpha the kubelet always requests `Running` and
  never sends `Stopped`, and there is no Kubernetes API field for a user to request it (see
  [Checkpoint Handling](#checkpoint-handling)). It becomes user-selectable when the migration
  follow-up implements it.

**Why `Stopped` is deferred.** "Terminate the source Pod but leave the object" is exactly
the terminated-but-not-deleted state that Graceful Node Shutdown and its follow-ons had to
work through, and the same issues apply here:

- A terminated-not-deleted Pod owned by a StatefulSet blocks recreation of a same-name Pod
  until the object is deleted (see [KEP-2268](https://git.k8s.io/enhancements/keps/sig-storage/2268-non-graceful-shutdown)).
- `VolumeAttachments` are not released until the Pod is deleted, so merely terminating the
  source Pod would **not** free the volumes a migration needs, which undercuts the stated
  purpose of `Stopped`.
- For controller-owned Pods, terminating the source triggers the controller's normal
  replacement semantics (an unwanted replica, plus ReadWriteOnce-volume races), which is the
  problem space addressed by the Job/Deployment pod-replacement-policy work
  ([KEP-3939](https://git.k8s.io/enhancements/keps/sig-apps/3939-allow-replacement-when-fully-terminated),
  [KEP-5882](https://git.k8s.io/enhancements/keps/sig-apps/5882-deployment-pod-replacement-policy)).

Because the only use case `Stopped` serves (migration) is out of alpha scope, and because a
terminate-only `Stopped` would not even deliver resource release, the full semantics,
including whether the source Pod is terminated or **deleted**, a terminal status reason on
the source Pod analogous to GNS, and integration with controller replacement and volume
detach, are deferred to the migration follow-up and will be designed with SIG Apps and SIG
Storage.

**CRI field.** A dedicated `post_checkpoint_state` field of enum type `PostCheckpointState`
on `CheckpointPodRequest` (see [CheckpointPod](#checkpointpod)). The CRI enum retains the
`STOPPED` value so runtimes may implement it ahead of Kubernetes, but in alpha the kubelet
only ever sends `RUNNING`.

**Kubernetes API.** There is no `postCheckpointState` field on `PodCheckpoint` in alpha. Because alpha
always leaves the source Pod running, the field would have a single legal value and do nothing, so
it is not added to the API yet. It will be introduced together with the `Stopped` behavior in the
migration follow-up, when it is actually used. Until then the kubelet always requests `RUNNING`
from the CRI.

**Interaction with restore.** The post-checkpoint state affects only the checkpoint side and
has no effect on the restore path: the archive contents are identical regardless of what
happens to the source Pod afterward. It only controls what happens to the *source* Pod once
the archive is written.

### Checkpoint Content

A Pod checkpoint is captured at two layers:

- Kubernetes-level (`PodCheckpoint` object, including the recorded Pod template in
  `status.checkpointedPodTemplate`, and node-local kubelet state). This layer is owned by
  Kubernetes and is described by this KEP.
- Container runtime-level (memory state, process hierarchies, open file descriptors,
  filesystem writable layers, and other low-level state). This layer is opaque to Kubernetes;
  its format and contents are owned by the container runtime and the underlying checkpoint mechanism (CRIU, gVisor, etc.). Different runtimes may implement this layer differently,
  and the format is allowed to change between runtime versions.

In the context of this proposal, support for volumes and network configuration is considered
out of scope for the initial implementation. However, the checkpoint must capture the
information necessary for the runtime to configure the network stack and reattach to the
same volumes during restore.

#### Pod Specification and Metadata

A Pod checkpoint captures all information required for the Pod to be restored at the
Kubernetes level. This information lives in two distinct places:

- **`PodCheckpoint.status.checkpointedPodTemplate`** is the API-level record: the object
  metadata and `v1.PodSpec` captured from the source Pod by the kubelet at checkpoint time. The
  restore Pod's spec is validated against it (see [Restore Mechanism](#restore-mechanism)). It
  lives in `status`, so only the kubelet writes it and users cannot change it, which is what makes
  it safe to compare against. The template is stored in full rather than as a hash for two
  reasons: when a restore is rejected the user needs to know which fields differ, and a hash only
  says whether something differs; and the template is read by the restore path and by clients that
  create a Pod from a checkpoint, so the fields themselves are needed. The lifecycle controller
  does not read it; only the kubelet and the restore path do. Each object is a few kilobytes, so
  the scaling concern is the number of objects, which checkpoint garbage collection bounds (see
  [Denial of service via excessive checkpointing](#denial-of-service-via-excessive-checkpointing)).
- **Node-local kubelet state**, including the CRI `PodSandboxConfig` passed from the kubelet
  to the container runtime, which is distinct from the `v1.PodSpec` defined at the API
  server and is needed to correctly recreate the sandbox at restore time.

`checkpointedPodTemplate` records:
- The serialized Pod specification (`v1.PodSpec`)
- Labels, annotations, and owner references
- Resource requests and limits
- Scheduling constraints and security contexts

To keep the record portable (needed for the future cross-node and cross-cluster restore cases),
the kubelet drops fields that are node-local or specific to the source cluster before writing it:
`spec.nodeName`, `nodeSelector` and affinity entries that name specific nodes, and the Pod
`status`, `uid`, `resourceVersion`, and `managedFields`. The equality check on restore skips these
same fields, plus `spec.restoreFrom`, which the restore Pod sets but the source Pod never had.
Container statuses, including containers that have already finished, are recorded separately in
the runtime archive and the `status.checkpointedContainers` and `status.checkpointedInitContainers`
lists.

Checkpointing requires all non-restartable init containers to have completed; restartable init
containers (sidecars) may still be running. The completed init containers and the running sidecars
are recorded in `status.checkpointedInitContainers` (kept separate from regular containers, mirroring
`PodStatus`). On restore, the running sidecars are restored running and remain restartable init
containers, while the completed init containers are reflected as completed from the captured state
and are not re-run. Checkpointing a Pod whose non-restartable init containers are still running is
out of scope for the initial implementation.

Pod spec changes between checkpoint and restore are not permitted in the initial
implementation. The API server checks spec equality against `status.checkpointedPodTemplate` at
admission, and the kubelet checks it again before the CRI restore call (see
[Restore Mechanism](#restore-mechanism)); either one rejects a mismatch. The comparison is on the
Pod spec, runs after API defaulting, and skips `spec.restoreFrom` and the node-placement fields the
restore flow adds (the injected node affinity and the scheduler-assigned `spec.nodeName`) along
with the node-local fields listed above. Users needing to change resource requests or limits
should do so after restore using the existing in-place Pod resize mechanism. During restore,
the process tree inside containers is recreated from the application state captured during
checkpointing: open file descriptors and memory allocations are recreated with the same
offsets and contents as at the time of checkpointing, so allowing arbitrary spec mutation
between checkpoint and restore would risk correctness violations.

`checkpointedPodTemplate` records the **allocated** state of the Pod — what is actually running —
not the desired spec. This matters when the two have diverged: with in-place Pod resize
(KEP-1287) the desired resources in `pod.spec` may differ from the resources actually allocated to
the running containers (a pending resize), and similar allocated-vs-desired divergence can arise
for the container set. Because the checkpoint captures the processes as they actually run, the
template must describe the allocated resources and container set, so the kubelet populates it from
the allocated state rather than copying `pod.spec` verbatim. The kubelet serializes the checkpoint
against in-place resize and other Pod updates for the duration of the checkpoint window (the
per-Pod in-flight guard), so the captured state is internally consistent — a caller gets the
pre-update (allocated) state or the post-update state, never a torn mix. Whether a *pending*
desired change should also be recorded and reapplied on restore (versus restoring the allocated
state and letting the user re-resize) is left open (see [Open Questions](#open-questions)).

#### Container Runtime State

The container runtime archive captures the complete execution context of all processes and
threads running in containers, including OCI container configurations, security contexts,
filesystem writable layers, and the checkpoint images needed to recreate the processes and
resume their execution. The exact contents and format of this archive are determined by
the container runtime and are opaque to Kubernetes.

#### Shared Pod Resources

This KEP focuses on providing the fundamental building blocks for capturing and restoring the execution
state of containers within a Pod, along with Pod-level metadata and configurations. Support for shared
Pod resources such as shared memory and volumes is out of scope for the initial implementation.

#### Checkpoint Storage Location

`status.checkpointLocation` is a discriminated union (`CheckpointSource`) that names the backend
holding the checkpoint data, keyed by a `type` discriminator. In alpha the only backend is
`NodeLocal`: the archive is stored on the node that took it, under the kubelet's checkpoint root
directory, and `nodeLocal.path` is the path relative to that root. The status deliberately does not
expose an absolute host path: the WG agreed on 2026-03-05 that the checkpoint location in status
should be implementation-agnostic and not expose filesystem paths, and that checkpoint storage
should be able to grow into other backends.

The union shape is what makes that growth additive. Other backends (for example a
`PersistentVolumeClaim`, or object storage such as an OCI registry or S3 bucket) are added later as
new members and new `type` values, so a checkpoint can live somewhere other than the node's local
disk. This is needed for cross-node restore (where the archive must be reachable from another node)
and for the distribution use cases discussed by the WG. Because the field is already a union,
adding those members is not an incompatible change; alpha clients keep using `NodeLocal`. Only the
`NodeLocal` backend is in scope for alpha — no other members are defined yet (a discriminated union
keyed by `type` follows the current API convention for unions rather than the older
`VolumeSource`-style implicit one-of).

### Pod Lifecycle

Pod-level checkpointing is permitted only on a Pod that is bound to a node, has all
non-restartable init containers completed, and has all regular containers and any restartable init
containers (sidecars) running. Checkpoint requests on Pods that do not meet these preconditions
must be rejected before reaching the container runtime. Checkpointing a Pod whose non-restartable
init containers are still running, and partial-ready states, are out of scope for this KEP.

During checkpointing, all containers in the Pod are frozen (using the Pod-level cgroup freezer)
as a prerequisite for creating a consistent checkpoint. Each container is then checkpointed
individually, and the cgroup is unfrozen at the end of this operation.

The kubelet must suspend liveness and readiness probes while a Pod is being checkpointed. Frozen
cgroups may cause probes to time out, and without suspension the kubelet would kill the Pod
mid-checkpoint. A Pod status condition (`Checkpointing=True`) is set so that higher-level
controllers can observe this state.

The `Checkpointing=True` condition is **observability only and does not, by itself, protect the
Pod from disruption during the freeze window.** Suspending probes prevents the *kubelet* from
killing the Pod, but the frozen Pod is still genuinely unavailable, and there is a real tension
that this KEP does not attempt to solve unilaterally:

- If the Pod is reported **not Ready** during the freeze, controllers and policies that treat
  not-ready as unhealthy (StatefulSet, PodDisruptionBudgets, descheduler-style components) may
  permit or actively trigger its eviction/deletion.
- If the Pod is instead **masked as Ready**, Service endpoints keep routing traffic to a frozen
  Pod, where it is blackholed.

A bespoke `Checkpointing` condition could express "temporarily unavailable, do not disrupt and do
not route," but only if the ecosystem adopts it, which is slow and partial. This is the same
class of problem as [kubernetes/kubernetes#116965] (pods that are temporarily unavailable but
should not be disrupted), and it overlaps with the signal an EvictionRequest-style API may need
during termination. Rather than invent a one-off mechanism, this KEP commits to converging on a
**common, right-granularity signal** with SIG-Node and SIG-Apps; until that exists, the freeze
window is a documented limitation (see [Risks and Mitigations](#risks-and-mitigations)) and the
condition is informational. The mitigation for alpha is operational: checkpoint Pods that can
tolerate a brief unavailability, keep the checkpoint window short (it is bounded by the
configurable timeout), and avoid checkpointing Pods under active disruption pressure.

### TCP Connection Handling

The initial implementation uses a TCP-close approach: all established TCP connections are closed
when a Pod is checkpointed. TCP-established connection preservation (restoring connections to
their pre-checkpoint state) requires CNI changes across all implementations and is deferred to a
future live migration KEP. IP address preservation across checkpoint/restore also requires CNI
changes and has been confirmed as feasible by SIG Network but represents significant work.

### Security Implications

Unlike the container-level checkpoint API described in [KEP-2008], which is reached through a
privileged kubelet endpoint, Pod-level checkpoint and restore expose no user-facing kubelet
endpoint: the namespaced API objects defined in this KEP are the user-facing surface, and users
never need direct kubelet access.

#### Privilege model

The existing container-level checkpoint API requires node administrator or SSH access to reach
the kubelet endpoint. Exposing Pod-level checkpoint and restore through namespaced API objects
is a different security model. Mitigations:

- `PodCheckpoint` is namespace-scoped and may only target Pods in the same namespace; the
  API server enforces same-namespace lookups. `spec.restoreFrom` is a name reference and is
  resolved in the namespace of the Pod that carries it; a user cannot point a Pod in
  namespace `A` at a `PodCheckpoint` in namespace `B`.
- No principal is granted the `nodes/proxy` permission for this feature. Checkpoint is driven by
  the kubelet watching `PodCheckpoint` objects and acting on those for Pods it runs, and restore
  flows through the normal Pod admission path; because no control-plane component calls the
  kubelet, there is no node-proxy privilege to grant or contain.
- Because the checkpoint flow is asynchronous, the kubelet writes the checkpoint result back to
  the `PodCheckpoint` status (see [Asynchronous checkpoint flow](#asynchronous-checkpoint-flow)).
  This write is tightly scoped: the `system:node` role grants `update`/`patch` on
  `podcheckpoints/status` only (this is what the Node authorizer evaluates), and the
  `NodeRestriction` admission plugin narrows it by allowing the write only when the checkpoint's
  source Pod (`spec.sourcePodName`) is bound to the requesting node, reusing the same node↔Pod
  relationship that already limits a kubelet to writing its own Pods' status. The kubelet cannot create,
  delete, or modify the `spec` of a `PodCheckpoint`, and cannot finalize a checkpoint for a Pod it
  does not run. (This was reviewed with SIG Auth alongside the `restore` verb.)
- The lifecycle controller exposes no user-facing API beyond the `PodCheckpoint` resource; users
  never interact with the kubelet directly. The kubelet acts on objects it observes and finalizes
  their status, but neither path is reachable by end users.
- Pre-defined namespaced ClusterRoles (viewer, editor, admin) are provided so administrators
  can bind checkpoint and restore access per namespace with `RoleBinding`.
- `sourcePodName` and `sourcePodUID` on `PodCheckpoint` are immutable after creation,
  preventing post-creation namespace-escape attempts and ensuring the pinned instance cannot
  be swapped after the object is admitted. `spec.restoreFrom` on Pod is *not* immutable:
  sequential re-restores from a different `PodCheckpoint` are a legitimate use case (rollback,
  repeated warm-start from a different snapshot).
- Pod-spec equality is validated against `status.checkpointedPodTemplate`, which is
  written by the kubelet at checkpoint time and immutable to users (see [Status and spec separation](#status-and-spec-separation)),
  so a user cannot forge the record being compared against. The API server enforces this equality
  at admission (the authoritative check, in the `PodRestoreAuthorization` plugin), and the kubelet
  re-checks it before the CRI restore call as defense in depth; either rejects the restore on
  mismatch. The referenced `PodCheckpoint` is always resolved in the restoring Pod's own
  namespace, so a Pod cannot reference a checkpoint in another namespace. Together this prevents an
  attacker who can edit a Pod from swapping in a foreign checkpoint to read its memory contents.
- Concurrent restores targeting the same `(namespace, name)` are serialized by a node-local,
  in-memory lock in the kubelet; only one restore may be in flight per `(namespace, name)` at
  a time. The key is `(namespace, name)` rather than Pod UID because every restore attempt
  carries a fresh UID, so a UID key would never collide. The lock is process-local and
  ephemeral: it is not a cluster-wide or API-level lock and is not persisted across kubelet
  restarts; after a restart an interrupted restore is simply retried and the container
  runtime cleans up any partial sandbox.
- The API server distinguishes two operations on a `PodCheckpoint`, which are authorized
  separately (this model was reviewed with SIG Auth):
  - **Reading** the object (`get`/`list`/`watch`) returns its JSON representation: conditions,
    `nodeName`, `checkpointLocation`, and the captured pod template. This is ordinary object
    access.
  - **Restoring** from the checkpoint reconstructs the captured process and memory state into
    a new Pod, which is more sensitive than reading the object. It is gated by a dedicated
    `restore` verb on the named `PodCheckpoint`: a Pod with `spec.restoreFrom` set is admitted
    only if the requester is authorized for `restore` on that `PodCheckpoint` in the Pod's
    namespace.

  `create` on `PodCheckpoint` separately gates checkpoint creation. Because `PodCheckpoint`
  is a real, served API object (it can be `kubectl get`-ed, not an authorization-only
  placeholder), expressing the restore permission as a verb on the resource is consistent
  with standard Kubernetes authorization. The split lets administrators grant restore access
  independently of checkpoint-create or read access, for example to consumers of warm-start
  checkpoints. The restore authorization is enforced in-tree by the API server before the
  request reaches the kubelet.
- The `restore` authorization is evaluated against the identity that issues the Pod create
  request: the user for a directly-created Pod, or the controller's ServiceAccount for a Pod
  created from a Pod template (Deployment, Job, etc.). `spec.restoreFrom` is intended for
  directly-created, one-shot restores (restoring the same captured memory image into multiple
  replicas is not a supported use case), so workload controllers are not granted the `restore`
  verb by default, and a controller can create restoring Pods only if explicitly granted
  `restore` on the referenced checkpoints.

Permission checks are enforced by the API server before the request reaches the kubelet.
Pod-readiness checks (non-restartable init containers completed, Pod is `Running`) are separately
enforced by the kubelet at execution time and may reject an otherwise-authorized request.

#### Sensitive memory contents

Checkpoint data may contain sensitive information from process memory, including secrets,
tokens, and encryption keys. Checkpoint artifacts must be treated as sensitive data, stored
with the handling expected for Secrets, and subject to the same access controls. Encryption of
checkpoint data at rest is CRIU-level work and is out of scope for this KEP.

#### Denial of service via excessive checkpointing

Unrestricted checkpointing can exhaust two distinct resources: a node's **disk** (the checkpoint
archives) and the cluster's **etcd** (the `PodCheckpoint` objects).

**On-node disk.** Repeated checkpoints can fill a node's disk, the same way the existing
container-level checkpoint API can. While the feature is in alpha and off by default, the kubelet
cleans up its own partial and aborted archives, and clusters that want stronger retention can use
the [checkpoint-restore operator](https://github.com/checkpoint-restore/checkpoint-restore-operator).
That is fine for alpha, but it is not a good enough story once the feature is on by default: we
should not ask every cluster to install an out-of-tree operator just to keep checkpoints from
exhausting the disk. So before Beta the kubelet needs to handle this itself: checkpoint storage
should count toward node disk pressure and be subject to the usual eviction and garbage
collection, with a limit on how much is kept locally. We treat that as a Beta blocker (see
[Beta](#beta)).

**etcd objects.** A `PodCheckpoint` is a first-class object, and use cases such as periodic
fault-tolerance checkpointing or checkpointing every Pod of a large training job can create a
large, unbounded population of them. Each object also carries Pod-derived metadata (the captured
template used for the spec-equality check), so an unbounded population is not free in etcd. For
alpha this is bounded operationally — the feature is off by default, the typical warm-start
pattern keeps only the latest one or two checkpoints per workload, and `PodCheckpoint` objects are
namespace-scoped and subject to the usual RBAC and (if configured) `ResourceQuota`. Before the
feature is on by default we will bound the object population directly: object garbage collection
(and an optional per-object TTL/expiry, keyed off `status.completionTime`) is part of the same
lifecycle-management enhancement as on-node retention, and is a Beta blocker.

The larger, cluster-level lifecycle management (quotas, retention policies, and attributing
storage back to a workload) is a separate piece of work we will take up in a follow-on
enhancement; it is useful, but the node-level and object-level protections above stand on their
own and do not wait for it.

#### automountServiceAccountToken on restore

Service account tokens mounted into the original Pod may be invalid or expired when a
checkpoint is restored. Checkpointable workloads should disable token automounting and refresh
tokens explicitly after restore; a formal opt-out or automatic token refresh mechanism will be
specified before Beta.

#### Path traversal protection

`status.checkpointLocation.nodeLocal.path` is a path relative to the kubelet's checkpoint root,
which makes this check straightforward. Before invoking the CRI restore, the kubelet resolves it
against its own root and verifies the result stays within that root, rejecting the restore
otherwise: absolute paths, `..` traversal, and symlink escapes are all rejected. Because the recorded value is
relative, a malformed or tampered `PodCheckpoint` status cannot point the runtime at an arbitrary
host path.

#### Status and spec separation

Users write `spec`; `status` is written only through the status subresource, and only by the
kubelet that runs the source Pod — the `InProgress` condition, `nodeName`, captured template,
pinned UID, and the terminal `Completed`/`Failed` condition with `checkpointLocation`.
`PodCheckpoint` is a built-in API type and the REST storage layer enforces the separation: the
main-object strategy strips `status` on user/controller updates (so the controller's finalizer
write cannot touch `status`), and the status-object strategy strips `spec` on any status update.
The kubelet's status write is additionally scoped by `NodeRestriction` to checkpoints whose
source Pod is bound to its own node (see [Privilege model](#privilege-model)).

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

- Kubelet probe suspension during the checkpoint freeze window (net-new for Pod-level
  checkpoints; container checkpointing does not suspend probes today) must be added.
- The CRI conformance suite must be extended to cover the new `CheckpointPod` and `RestorePod`
  RPCs once at least one runtime implements them.

##### Unit tests

Coverage baselines will be captured when the implementation PR is opened.

Unit tests must cover at least:

- Kubelet `PodCheckpoint` sync: it acts only on objects whose source Pod it runs, pins the
  source Pod by UID (a UID mismatch fails as `SourcePodReplaced`), and de-duplicates a checkpoint
  already in flight for the same Pod.
- Path traversal rejection on the restore path (`status.checkpointLocation.nodeLocal.path` must
  resolve to a clean path within the checkpoint storage directory; `..`, absolute paths, and
  symlink escapes are rejected).
- Pod phase precondition (checkpoint rejected unless the Pod is `Running` with all init
  containers completed).
- Timeout enforcement (the kubelet sets the CRI call's gRPC deadline from
  `spec.timeoutSeconds`, and an expiry is recorded on the `PodCheckpoint` as `CheckpointFailed`).
- Feature gate disabled: the kubelet does not watch or act on `PodCheckpoint` objects (no
  checkpoint is started).
- Cgroup freeze and unfreeze sequence ordering and error recovery.
- Pod condition `Checkpointing=True` is set and cleared around the operation.

##### Integration tests

CRI API changes must be implemented by at least one container engine. Because the kubelet has
no integration test harness, validation uses `test/e2e_node`, which effectively serves as the
kubelet integration suite. The following scenarios must pass before Alpha:

- `CheckpointPod` happy path: create a `PodCheckpoint` for a single-container Pod on the node;
  the kubelet acts on it, finalizes the `PodCheckpoint` as `Ready=True` with
  `checkpointLocation.type: NodeLocal`, and the archive at the resolved
  `checkpointLocation.nodeLocal.path` exists and is non-empty.
- Async contract: a re-observed object while a checkpoint is in flight does not start a second
  checkpoint (the per-Pod in-flight guard de-duplicates).
- `RestorePod` happy path: restore that Pod; verify a new sandbox ID is returned.
- Probe suspension: a Pod with a 1 second liveness probe is not killed during a multi-second
  checkpoint window.
- Runtime does not implement the new RPC: the kubelet finalizes the `PodCheckpoint` as
  `Ready=False`/`CheckpointFailed` rather than panicking.
- Feature gate disabled: `PodCheckpoint` objects are not served (so none can be created or acted
  on) and `spec.restoreFrom` is rejected at Pod admission.
- `spec.restoreFrom` happy path: the kubelet sees the field during `SyncPod`, calls
  `restorePodSandbox()`, and the Pod transitions to `Running`.
- Admission equality and affinity injection: the `PodRestoreAuthorization` plugin rejects a restore
  Pod whose spec does not match a `Ready` checkpoint's `status.checkpointedPodTemplate` (exempting
  `spec.restoreFrom` and the injected node affinity), admits one that matches, and injects the
  required node affinity targeting `status.nodeName` (rejecting a user-supplied conflicting
  `spec.nodeName`/affinity).

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
- Kubelet object-watch checkpoint execution implemented behind the
  `PodLevelCheckpointRestore` feature gate: the kubelet watches `PodCheckpoint` objects and
  finalizes the status of those for Pods it runs (no imperative HTTP endpoint; restore is likewise
  driven by `spec.restoreFrom` during `SyncPod`).
- `PodCheckpoint` defined and implemented. Restore trigger implemented as a new optional
  `restoreFrom` field on Pod spec.
- `PodRestoreAuthorization` admission plugin implemented: authorizes the `restore` verb on the
  referenced `PodCheckpoint`, injects a node-affinity constraint pinning the Pod to the
  checkpoint's node (so it is scheduled there rather than binding `spec.nodeName` directly), and
  authoritatively validates Pod-spec equality against `status.checkpointedPodTemplate`.
- Field selectors `spec.sourcePodName` and `status.nodeName` registered on the `PodCheckpoint`
  REST storage, so checkpoints can be listed by source Pod or by node.
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
- A common, right-granularity mechanism for protecting a temporarily-unavailable Pod from
  disruption during the checkpoint freeze window (without blackholing traffic) has been agreed
  with SIG-Node and SIG-Apps, coordinated with [kubernetes/kubernetes#116965] and any
  EvictionRequest-style termination signal, rather than relying on the informational
  `Checkpointing=True` condition alone (see [Pod Lifecycle](#pod-lifecycle)).
- The kubelet keeps checkpoints from filling a node's disk on its own, so this no longer depends
  on the out-of-tree operator. Checkpoint storage counts toward node disk pressure and can be
  evicted or garbage-collected like other kubelet-managed data, and there is a limit on how much
  is kept locally. This blocks Beta (see
  [Denial of service via excessive checkpointing](#denial-of-service-via-excessive-checkpointing)).
- The cluster-level lifecycle management (quotas, retention policies, and storage attribution)
  has been discussed with SIG-Node and a follow-up is scoped. It can land as its own KEP and does
  not block Beta once the kubelet-side protection above is in place.
- A terminal/give-up signal for repeatedly-failing restores is decided (today a failing restore
  retries with backoff indefinitely, like `ImagePullBackOff`); needed once restore integrates with
  workload controllers (e.g. Jobs with a `backoffLimit`) so they get a terminal signal rather than
  a Pod stuck `Pending` forever.
- Additional e2e testing for stabilization; known issues and gaps documented.
- No open CVE-class issues for the feature.

#### GA

- Feature has been stable in Beta for at least two Kubernetes releases.
- Feedback gathered from production deployments.
- Conformance tests cover all GA endpoints.
- At least three major container runtimes support the feature.
- User-facing documentation published on kubernetes.io.

### Upgrade / Downgrade Strategy

On upgrade, the kubelet watches and executes Pod-level checkpoints once the `PodLevelCheckpointRestore`
feature gate is enabled and the container runtime implements the required CRI APIs. If the runtime
does not implement the new CRI APIs, the kubelet's background `CheckpointPod` CRI call fails and the
kubelet finalizes any `PodCheckpoint` it picks up as `Ready=False`/`CheckpointFailed`, so the feature
is effectively unavailable on that node.

On downgrade, the feature becomes unavailable in either of two ways, neither of which errors out
existing workloads. A kubelet rolled back to a version that does not implement this feature stops
watching `PodCheckpoint` objects, so they are simply never picked up (see the Version Skew bullets
below). A CRI call to a runtime that no longer implements the Pod-level checkpoint API fails, leaving
the `PodCheckpoint` `Ready=False`/`CheckpointFailed`.

### Version Skew Strategy

The CRI API extensions and checkpoint execution are local to the node. The `PodCheckpoint`
built-in API type is served by the API server. The kubelet watches `PodCheckpoint` objects and
performs the checkpoint for those whose source Pod it runs; no control-plane component calls the
kubelet. The
pod-snapshot-controller ships in-tree as part of `kube-controller-manager` and reconciles
`PodCheckpoint` lifecycle only (finalizers, garbage collection). All three components are gated by
the `PodLevelCheckpointRestore` feature gate. Restore is driven entirely by the kubelet from
`spec.restoreFrom` on the Pod object; no controller is involved on the restore path.
Version skew considerations:

- If the kubelet supports the new CRI API but the container runtime does not, the kubelet's
  background `CheckpointPod`/`RestorePod` CRI call fails (`Unimplemented`) and the kubelet
  finalizes the `PodCheckpoint` as `Ready=False`/`CheckpointFailed`.

- If the container runtime supports the new CRI APIs but the kubelet does not, the feature is
  unavailable: an older kubelet does not watch `PodCheckpoint` objects or issue the new CRI calls,
  so checkpoints are never executed.

- Whether a given node supports Pod checkpoint/restore is surfaced through **Node Declared
  Features**: a node advertises the capability only when its kubelet runs the feature and (for
  restore) the runtime implements the CRI RPCs. The control plane uses that signal so version skew
  fails fast rather than stranding work:
  - On `PodCheckpoint` creation, admission resolves the source Pod's node and rejects the request
    if that node does not declare the checkpoint capability.
  - On restore, the scheduler avoids nodes that do not declare restore support when placing the
    Pod (the injected node affinity already constrains it to the checkpoint's node; the capability
    signal matters for the future cross-node case and as a general guard).
  If a node nonetheless lacks runtime CRI support at execution time, the kubelet's background CRI
  call fails and it finalizes the `PodCheckpoint` as `Ready=False`/`CheckpointFailed` — the backstop
  for the runtime axis, which the node feature cannot fully cover. (Node Declared Features is itself
  a dependency; enforcement lands when it is available — see [Open Questions](#open-questions).)

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
  - Feature gate name: `PodLevelCheckpointRestore`
  - Components depending on the feature gate:
    - `kube-apiserver` - serves the `PodCheckpoint` built-in type, gates and validates the
      `restoreFrom` Pod-spec field, and runs the `PodRestoreAuthorization` admission plugin
      (the `restore`-verb authorization, the injected node-affinity constraint pinning the Pod to
      the checkpoint's node, and the
      authoritative pod-template equality check).
    - `kube-controller-manager` - runs the in-tree pod-snapshot-controller that reconciles
      `PodCheckpoint` lifecycle (finalizers and garbage collection); it is not on the checkpoint
      execution path.
    - `kubelet` - watches `PodCheckpoint` objects and acts on those for Pods it runs,
      issues the CRI `CheckpointPod`/`RestorePod` calls, populates
      `status.checkpointedPodTemplate`/`status.sourcePodUID`, finalizes the `PodCheckpoint` status
      (scoped by the Node authorizer / `NodeRestriction`), and acts on `spec.restoreFrom` during
      `SyncPod`.

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. By disabling the `PodLevelCheckpointRestore` feature gate.

While the gate is off the API server stops serving the `checkpoint.k8s.io`
group, so any `PodCheckpoint` objects created while it was on are **stranded**:
they remain in etcd but are not served (`get`/`list`/`delete` return 404), the
controller and kubelet ignore them, and on-disk checkpoint archives stay until
cleaned up. They consume no node or runtime resources while inert. Two caveats:

- A `PodCheckpoint` that still carries the restore-lock finalizer (a delete
  requested while a restore was in flight) cannot be removed until the gate is
  re-enabled and the controller clears the finalizer; while the gate is off the
  object is stuck and unservable.
- Re-enabling makes these objects served and reconciled again (see the next
  question), so the clean way to drain stranded objects is to re-enable, let the
  controller settle, then delete them; an admin can also remove them directly
  from etcd.

###### What happens if we reenable the feature if it was previously rolled back?

The feature keeps no in-memory state across the gate flip, so re-enabling starts
from a clean slate. Per component:

- `kube-apiserver`: serves the `checkpoint.k8s.io` group and the Pod
  `spec.restoreFrom` field again. Any `PodCheckpoint` objects that survived the
  rollback (they remain stored, just inert) become readable and a `restoreFrom`
  reference to one is honored again.
- `kube-controller-manager`: the pod-snapshot-controller starts again and
  resumes its lifecycle reconcile (finalizers, garbage collection); it is not on
  the checkpoint execution path.
- `kubelet`: resumes watching `PodCheckpoint` objects and executing checkpoints
  for the Pods it runs, and resumes writing `PodCheckpoint` status.

Operations in flight at the moment of rollback are not resumed on re-enable: a
checkpoint interrupted by disabling the gate was already finalized as failed
(`Ready=False`/`CheckpointFailed`), so a re-enabled cluster simply processes new
requests.

###### Are there any tests for feature enablement/disablement?

Yes. The e2e framework cannot enable or disable feature gates, so this is covered by unit and
integration tests. Because the `PodLevelCheckpointRestore` gate guards all three components, the
coverage is per-component:

- `kube-apiserver`. The `spec.restoreFrom` field uses the standard `dropDisabledFields`
  handling: it is cleared when the gate is off unless it was already set on the old object
  (ratcheting). This is the disable-after-write "switch" test the PRR template calls for. The
  gating logic and its ratcheting are implemented in `pkg/api/pod/util.go` and tested by
  `TestGetValidationOptionsRestoreFrom` in `pkg/api/pod/util_test.go`; validation rejects
  `restoreFrom` on create when the gate is off, and the `PodRestoreAuthorization` admission plugin
  returns early and leaves the Pod untouched when the gate is off, covered by the "feature gate
  disabled is a no-op" case in `TestPodRestoreAuthorization`
  (`plugin/pkg/admission/podrestoreauthorization/admission_test.go`). The plugin is registered and
  enabled by default in `pkg/kubeapiserver/options/plugins.go` (imported, added to
  `AllOrderedPlugins` and the default-on set, and registered via `Register`); the
  `PodLevelCheckpointRestore` gate guards its behavior rather than its registration, so it is always
  in the admission chain but inert when the gate is off.
- `kubelet`. With the gate off the kubelet does not watch or act on `PodCheckpoint` objects, so
  no checkpoint is started, covered by a kubelet unit test asserting the `PodCheckpoint` watch is
  inactive when the gate is off. `spec.restoreFrom` is a no-op so the
  kubelet creates a fresh sandbox rather than restoring, covered by `TestSyncPodRestoreFromGatedByFeature`
  (`pkg/kubelet/kuberuntime`). The `Restoring` condition is kubelet-owned only with the gate on,
  covered by `TestPodConditionByKubeletRestoring` (`pkg/kubelet/types`); the
  `podcheckpoints/status` write and its `NodeRestriction` scoping are likewise exercised only with
  the gate on.
- `kube-controller-manager`. The pod-snapshot-controller is registered behind the gate via its
  `ControllerDescriptor` `requiredFeatureGates`, so it does not run when the gate is off (no
  restore-lock finalizers are reconciled). An integration test in `test/integration/podcheckpoint`
  asserts that with the gate disabled the `checkpoint.k8s.io` group is not served, so a
  `PodCheckpoint` cannot be created, and that with the gate enabled a `PodCheckpoint` can be
  created and the controller adds and later removes the restore-lock finalizer as a Pod restores
  from it.

Disablement does not affect running workloads. Existing `PodCheckpoint` objects remain stored in
etcd but unserved while the gate is off (the `checkpoint.k8s.io` group is not served); a Pod's
`restoreFrom` field is likewise inert. See [Can the feature be disabled](#can-the-feature-be-disabled-once-it-has-been-enabled-ie-can-we-roll-back-the-enablement) for how these stranded objects are drained on re-enable.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

The same `PodLevelCheckpointRestore` feature gate guards all three components: the
`kube-apiserver` (the `PodCheckpoint` type, the `restoreFrom` field, and the admission plugins),
the `kube-controller-manager` (the pod-snapshot-controller, lifecycle only), and the `kubelet`
(watching `PodCheckpoint` objects, the CRI calls, and the status write). The actual
checkpoint/restore work is still node-local, so most rollout consequences are scoped to
individual nodes. Rollout consequences:

- Partial rollout. In a cluster where the feature gate is enabled on some kubelets and not
  others, checkpoint and restore operations succeed only on nodes where the kubelet has the
  feature enabled. A `PodCheckpoint` whose source Pod is on a node whose kubelet does
  not watch/execute checkpoints is simply not picked up: it stays in its initial state with no
  `Ready` condition set, and an operator sees that no node has acted on it. Enabling the gate on
  that node lets its kubelet act on the
  pending object.
- Mid-rollout kubelet restart. Because the flow is asynchronous, the checkpoint result is
  written to the `PodCheckpoint` status by the kubelet. A checkpoint in flight when the kubelet
  restarts is not resumable: on startup the kubelet garbage-collects any partial archive and
  finalizes the `PodCheckpoint` as `Ready=False` with reason `CheckpointFailed`, so the object
  never hangs in `CheckpointInProgress`.
- Version skew. If the kubelet has the feature gate enabled but the container runtime does not
  implement the new CRI RPCs, the background CRI call fails with `Unimplemented`, and the kubelet
  finalizes the `PodCheckpoint` as `Ready=False` with reason `CheckpointFailed`.
- Already-running workloads. Not affected. No existing Pod behaviour changes when the feature
  is enabled. Only Pods targeted by an explicit checkpoint or restore request are impacted,
  and the checkpoint window pauses them only for the duration of the operation.
- Rollback. Disabling the feature gate on a kubelet has no effect on existing Pods and no
  persistent state is left behind. Checkpoint artifacts remain on disk until the retention
  policy cleans them up. Operations initiated after rollback fail with a typed error.

###### What specific metrics should inform a rollback?

A sustained or systemic rise in checkpoint/restore failures, observed on the
kubelet metrics endpoint:

- `kubelet_pod_checkpoint_operations_total{result="failure"}` and
  `kubelet_pod_restore_operations_total{result="failure"}` — the primary signal.
- `kubelet_runtime_operations_errors_total{operation_type="checkpoint_pod|restore_pod"}`
  — the underlying CRI errors.

Failures concentrated on a single node or runtime point at that node (roll back
the kubelet there) rather than the feature; a cluster-wide rise after enabling
the gate is the signal to roll back the feature. A failed checkpoint does not
disrupt the source workload, so at alpha the threshold is an operator judgment
rather than a hard SLO.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet. This will be answered after the alpha implementation: a manual
upgrade -> downgrade -> upgrade test will be performed and the results recorded
here as part of graduating toward Beta.

The expected behavior: the feature gate guards new behavior and the new
`PodCheckpoint` API type. The only new persisted state is `PodCheckpoint`
objects; a downgraded (gate-disabled) control plane stops serving/reconciling
them and the kubelet stops watching them, so existing objects become inert
rather than causing errors, and re-enabling resumes cleanly. As an alpha,
off-by-default feature, full skew/rollback test coverage is not required at this
stage.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

An operator can see the feature in use through `PodCheckpoint` objects and their status in the
API, Pods carrying `spec.restoreFrom`, and the kubelet-exposed checkpoint/restore metrics.

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event reasons on `PodCheckpoint`: `CheckpointStarted`, `CheckpointSucceeded`,
    `CheckpointFailed`.
  - Event reasons on the restored Pod: `RestoreSucceeded` on success; on the failure or retry
    paths the reason names the cause — `CheckpointNotReady`, `CheckpointWrongNode`,
    `PodSpecMismatch`, `CheckpointDataMissing`, or `RestoreInProgress` (transient, while the Pod
    waits on the kubelet's restore serialization lock). The Pod stays `Pending` and is retried for
    the non-terminal cases (see [End-to-end restore walkthrough](#end-to-end-restore-walkthrough)).
    A spec mismatch is normally surfaced earlier still: admission rejects the Pod create
    synchronously with an `Invalid` error naming the offending field, so the user sees it at
    `kubectl apply` time and no Pod is created. The kubelet's `PodSpecMismatch` event is the
    defense-in-depth path that only fires in the narrow window where a restore was admitted against
    a not-yet-`Ready` checkpoint.
  - Event reason on the source Pod: `CheckpointingPod`, emitted when the checkpoint window
    starts (the matching `Checkpointing=True` condition is what is set and later cleared).
- [x] API `.status`
  - `PodCheckpoint.status.conditions[type=Ready]` is the single source of truth for checkpoint
    state: `status: "False"` with reason `Pending` or `CheckpointInProgress` while the operation
    is underway, `status: "True"` with reason `CheckpointCompleted` on success, and
    `status: "False"` with reason `CheckpointFailed` (detail in the message) on failure.
    Each condition carries its own `observedGeneration`.
  - On the source Pod: a condition `Checkpointing=True` while the checkpoint window is active
    (see [Pod Lifecycle](#pod-lifecycle)).
  - On the restored Pod: `spec.restoreFrom` records the `PodCheckpoint` that produced it. A
    condition `Restoring=True` is set while that Pod's own sandbox restore is in flight and
    cleared once the Pod is `Running`. If the restore is blocked because another restore for
    the same `(namespace, name)` holds the kubelet's restore serialization lock, the Pod instead carries
    `Restoring=False` with reason `RestoreInProgress` while it waits and retries. These
    conditions report node-local kubelet state; they are not an API-level lock.

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

`PodCheckpoint` object metrics:

- `podcheckpoint_ready_condition_total{status="True|False",reason="Pending|CheckpointInProgress|CheckpointCompleted|CheckpointFailed|SourcePodReplaced"}`,
  counter of `Ready` condition transitions, emitted by the kubelet that writes the status.
- `podcheckpoint_reconcile_duration_seconds`, histogram, emitted by the pod-snapshot-controller
  for its lifecycle reconcile (finalizers, garbage collection).

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

Yes, only when explicitly invoked by a user. Creating a `PodCheckpoint` generates one watch event
to the owning kubelet (no control-plane-to-kubelet call), plus a small bounded number of
`PodCheckpoint` status writes: the `CheckpointInProgress` update and the kubelet's terminal
`Completed`/`Failed`
update. The flow is event-driven (no polling of the kubelet) and there are no periodic or
background API calls. Restore does not introduce any new API calls beyond the normal Pod create
that already occurs in the existing Pod lifecycle.

###### Will enabling / using this feature result in introducing new API types?

Yes:

- `PodCheckpoint` in the `checkpoint.k8s.io/v1alpha1` API group, namespace-scoped. One object
  per checkpoint operation. Its `status.checkpointedPodTemplate` embeds a sanitized
  `PodTemplateSpec` captured from the source Pod; this is bounded by the size of a single Pod
  template (kilobyte-scale, well within the etcd per-object limit) and is written once when
  the checkpoint reaches `Ready`. The full template is kept rather than a hash (so the restore
  equality check can report which fields differ); the scaling concern is object *count*, bounded
  by garbage collection — see
  [increasing size or count](#will-enabling--using-this-feature-result-in-increasing-size-or-count-of-the-existing-api-objects).
- A new optional field `restoreFrom` on Pod spec referencing a `PodCheckpoint` in the same
  namespace.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Pod spec gains one optional field, `restoreFrom`, a name reference to a `PodCheckpoint` in the
same namespace. The additional bytes are negligible (a single name string).

The feature also adds `PodCheckpoint` objects — one per checkpoint operation, each embedding a
sanitized, kilobyte-scale `PodTemplateSpec` (kept in full rather than as a hash so the equality
check can report which fields differ and the restore path can consume the fields; see
[Pod Specification and Metadata](#pod-specification-and-metadata)). Per-object size is small, so
the scaling factor is the object *count*: workloads that checkpoint repeatedly (periodic
fault-tolerance, or per-Pod across a large Job) could accumulate many objects. The count is
bounded by checkpoint garbage collection (a Beta blocker; see
[Denial of service via excessive checkpointing](#denial-of-service-via-excessive-checkpointing))
and, for the warm-start pattern, by keeping only the latest one or two checkpoints.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. Normal Pod lifecycle operations are unchanged. The checkpoint window pauses the source
Pod (visible via the `Checkpointing=True` condition) but does not alter any measured SLIs for
unrelated Pods.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

During checkpointing the memory pages of all processes running in the checkpointed containers will be saved to disk.
In addition, the read-write layer of the rootfs of checkpointed containers is included as part of the checkpoint.
As a result, disk usage is expected to increase by the compressed size of these checkpoints. CPU, RAM, and IO see a
transient spike on the checkpointed node for the duration of the freeze-and-dump window; there is no steady-state
increase for unrelated Pods or components.

For alpha the kubelet cleans up its own partial and aborted archives, and clusters that want stronger retention can
use the out-of-tree [checkpoint-restore operator](https://github.com/checkpoint-restore/checkpoint-restore-operator).
Before the feature is on by default, in-tree kubelet garbage collection makes checkpoint storage count toward node
disk pressure and be evicted or collected like other kubelet-managed data — a Beta blocker (see
[Denial of service via excessive checkpointing](#denial-of-service-via-excessive-checkpointing)).

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

The primary node resource at risk is disk: checkpoint archives accumulate on the node (and each archive consumes
inodes), so an unbounded population can exhaust the checkpoint storage directory. A restored Pod consumes PIDs,
sockets, and file descriptors like any equivalent Pod of the same shape — restore does not multiply these beyond the
normal Pod footprint. For alpha the mitigation is the kubelet's partial-archive cleanup plus the out-of-tree
[checkpoint-restore operator](https://github.com/checkpoint-restore/checkpoint-restore-operator) retention policies;
before Beta the kubelet gains in-tree garbage collection so node disk safety does not depend on an out-of-tree
operator (a Beta blocker; see
[Denial of service via excessive checkpointing](#denial-of-service-via-excessive-checkpointing)).

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

- In-flight checkpoints still run to completion against the container runtime; the CRI call
  itself does not need the API server, and the archive is written to disk. The kubelet cannot
  record the outcome on the `PodCheckpoint` while the API server is down, so it retries that
  status write until it succeeds; the object stays `CheckpointInProgress` in the meantime and no
  result is lost. A restore in progress reads its referenced `PodCheckpoint` from the API server,
  so a restore that has not yet resolved the checkpoint is retried by the kubelet once
  connectivity returns.
- Status converges once the API server is back. Because the kubelet writes the terminal status
  (not the controller), there is nothing to poll: the kubelet's pending status write lands and
  the controller observes it through its normal watch. New triggers are not lost either: an
  object left `CheckpointInProgress` is re-driven on the next reconcile.
- New operations are blocked. Users cannot create new `PodCheckpoint` objects or Pods with
  `spec.restoreFrom` without the API server. This is expected and identical to every other
  Pod-create-driven flow.

###### What are other known failure modes?

- Container runtime does not implement the new CRI RPCs.
  - Detection: `kubelet_runtime_operations_errors_total{operation_type="checkpoint_pod"}`
    increases; the `Ready` condition is `False` with reason `CheckpointFailed` (message: the
    runtime does not implement the CRI RPCs).
  - Mitigation: upgrade the runtime to a version that supports the new RPCs, or disable the
    feature gate.
  - Diagnostics: kubelet logs `failed to call CheckpointPod: Unimplemented` at V(2).
  - Testing: an e2e test that injects a runtime without CRI support.
- Checkpoint timeout.
  - Detection: `kubelet_pod_checkpoint_operations_total{result="failure"}` increases; the
    `Ready` condition is `False` with reason `CheckpointFailed` (message notes the timeout).
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
- Checkpoint object is never picked up by a kubelet.
  - Detection: the `PodCheckpoint` stays with no `Ready` condition (or stuck at
    `CheckpointInProgress`) indefinitely, with no terminal status written.
  - Mitigation: confirm the source Pod (`spec.sourcePodName`) is bound to a node, that node's
    kubelet has the `PodLevelCheckpointRestore` gate enabled, and the kubelet is watching
    `PodCheckpoint` objects (it requires `list`/`watch` on `podcheckpoints`).
  - Diagnostics: the kubelet logs `Starting PodCheckpoint watch` at startup and a per-object
    sync error if one occurs; `kubectl describe podcheckpoint` shows the last recorded condition
    (or none).
  - Testing: an e2e test that creates a `PodCheckpoint` for a Pod on a node with the gate
    disabled and asserts it is not acted on.
- Kubelet cannot write the result back to the `PodCheckpoint` status.
  - Detection: the object stays `CheckpointInProgress` after the kubelet logs a completed
    checkpoint; the kubelet logs a `Forbidden`/conflict error updating `podcheckpoints/status`.
  - Mitigation: confirm `NodeRestriction` is enabled and the kubelet's node has the expected
    Node-authorizer grant on `podcheckpoints/status`; confirm the source Pod is bound to that
    node.
  - Diagnostics: kubelet logs the status-write error; `kubectl describe podcheckpoint` shows the
    last recorded condition.
  - Testing: an e2e test asserting a kubelet cannot finalize a `PodCheckpoint` for a Pod on a
    different node.
- Checkpoint archive missing on the pinned node.
  - Detection: the restore Pod stays in `ContainerCreating`; kubelet event
    `CheckpointDataMissing`. (Admission injects a node affinity for `status.nodeName`, so the Pod
    is scheduled to the node that recorded the checkpoint; this case is the archive being absent on
    that node — for example garbage-collected — not the Pod landing on the wrong node.)
  - Mitigation: confirm the archive still exists under the kubelet's checkpoint root on
    `status.nodeName`; cross-node checkpoint transport (which would let the restore run elsewhere)
    is a follow-on enhancement.
  - Diagnostics: kubelet logs the resolved checkpoint path and the missing-file error.
  - Testing: an e2e test where the checkpoint archive is removed from the pinned node before
    restore; plus a unit test that admission rejects a user-supplied `spec.nodeName`/affinity that
    conflicts with the injected constraint.
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
4. If an object is not being picked up at all (no `Ready` condition and no
   `CheckpointInProgress`), confirm the source Pod (`spec.sourcePodName`) is bound to a node, that
   node's kubelet has the `PodLevelCheckpointRestore` gate enabled, and that kubelet is watching
   `PodCheckpoint` objects. `status.nodeName` is written by the kubelet only once it picks the
   object up, so it is expected to be empty here and is not a useful signal for this case.

## Implementation History

- 2026-01-29: KEP opened.

## Open Questions

These are design questions to resolve during implementation; they do not change the alpha API
shape described above.

- **Should `CheckpointPod` and `RestorePod` be their own CRI service rather than methods on
  `RuntimeService`?** A separate service — alongside the existing `RuntimeService` and
  `ImageService` — could let checkpoint and restore be implemented by a component other than the
  container runtime, and would make testing and development easier by allowing an independent
  implementation or test double. The trade-off to tease out is that checkpoint/restore still needs
  deep runtime cooperation (freezing containers, driving CRIU through the OCI runtime, access to
  sandbox and container state), and a separate service means the kubelet has to discover and dial a
  second endpoint with its own version negotiation. To be decided during implementation.

- **Should the source-Pod identifiers be grouped into a `spec.sourcePod` reference object (a
  `SourcePodReference`) instead of the flat `spec.sourcePodName` and `spec.sourcePodUID` fields?**
  The WG settled that the source-Pod name is sufficient for the initial API (with the optional
  `sourcePodUID` for instance pinning); grouping the two into a reference object was raised, and it
  could also host a future selector-based source (checkpointing a replica without naming a specific
  Pod). Deferred to API review. Moving flat fields into a struct is an incompatible change, so it
  would be settled before the API stabilizes.

- **When the allocated Pod has a pending desired change (e.g. an in-place resize in progress),
  should the checkpoint also record that intent and reapply it on restore?** The checkpoint
  captures the allocated (actual) state, so the immediate behavior is settled: restore recreates
  what was running. What is open is whether to additionally preserve a not-yet-applied desired
  change so the restored Pod resumes converging toward it, versus restoring the allocated state and
  leaving the user to re-issue the resize/update. To be decided during implementation, and it
  becomes more pressing as features like Dynamic Containers widen the allocated-vs-desired gap.

- **Timing of the Node Declared Features dependency.** Restore relies on the scheduler (and
  checkpoint-create admission) to avoid nodes that cannot satisfy a restore, which is best driven by
  a node-advertised capability via Node Declared Features (see
  [Version Skew Strategy](#version-skew-strategy)). Whether that gating is required for alpha or is a
  fast-follow depends on Node Declared Features' own availability and maturity; until it lands, the
  injected node affinity still pins the restore to the checkpoint's node and the kubelet's
  `CheckpointFailed` path remains the runtime-axis backstop.

- **Deeper scheduler integration for cross-node restore.** For alpha the injected affinity hard-pins
  the restore to the single node that holds the checkpoint. Once cross-node checkpoint transport
  exists, the constraint relaxes from "this exact node" to "a node that can actually run this
  checkpoint," which is a richer scheduling problem (and may want a scheduler plugin rather than a
  static affinity term). Beyond "has (or can be given) the archive and supports restore," that
  includes node **compatibility**: a checkpoint can only restore on a node whose CPU architecture —
  and likely kernel version and CRIU/gVisor versions — match where it was taken, or the runtime
  cannot process the snapshot. Some of these signals (CRIU/gVisor versions in particular) are not
  exposed to the control plane today, so surfacing them (for example through Node Declared Features
  or node status) is a prerequisite worth designing early, even though portability across
  heterogeneous environments is a [Non-Goal](#non-goals) for alpha. A likely shape is a scheduler
  plugin backed by the checkpoint controller that hints at compatible nodes. Node migration is large
  enough to be its own KEP, designed with SIG Scheduling.

## Drawbacks

The feature is useful but not free of trade-offs, several of which are inherent to
checkpoint/restore:

- Checkpoint and restore are not transparent to applications: in-memory secrets, tokens,
  environment values, and cached hostnames persist through a restore, so workloads must
  cooperate for correctness (see [Risks and Mitigations](#risks-and-mitigations)).
- The checkpoint freeze window makes the source Pod temporarily unavailable, and there is no
  clean ecosystem mechanism yet to keep controllers from disrupting it without blackholing
  traffic (see [Pod Lifecycle](#pod-lifecycle)).
- Checkpoint archives can be large and consume node disk; robust lifecycle management (quotas,
  retention) is deferred to a follow-on, with an in-tree GC floor required for Beta.
- The asynchronous control flow adds moving parts (the kubelet writes status, scoped by
  `NodeRestriction`, and interrupted checkpoints are reconciled on kubelet restart) compared to
  a single synchronous call.

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
