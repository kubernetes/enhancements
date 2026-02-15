# KEP-5823: Pod-Level Checkpoint/Restore

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation](#implementation)
  - [User Stories](#user-stories)
    - [Optimizing resource utilization for interactive workloads](#optimizing-resource-utilization-for-interactive-workloads)
    - [Accelerating startup of applications with long initialization times](#accelerating-startup-of-applications-with-long-initialization-times)
    - [Enabling fault-tolerance for long-running workloads](#enabling-fault-tolerance-for-long-running-workloads)
    - [Interruption-aware scheduling with checkpoint/restore](#interruption-aware-scheduling-with-checkpointrestore)
    - [Pod migration across nodes for load balancing and maintenance](#pod-migration-across-nodes-for-load-balancing-and-maintenance)
    - [Investigating security incidents with forensic checkpointing](#investigating-security-incidents-with-forensic-checkpointing)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [CheckpointPod](#checkpointpod)
  - [RestorePod](#restorepod)
  - [Checkpoint Content](#checkpoint-content)
    - [Pod Specification and Metadata](#pod-specification-and-metadata)
    - [Container Runtime State](#container-runtime-state)
    - [Shared Pod Resources](#shared-pod-resources)
  - [Pod Lifecycle](#pod-lifecycle)
  - [Restore Semantics](#restore-semantics)
  - [Security Implications](#security-implications)
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
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
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

## Summary

This proposal aims to define minimal CRI APIs for Pod-level checkpoint and restore, as well as a kubelet
API for Pod checkpointing.

The core idea behind this proposal is to outline the minimal set of Container Runtime Interface (CRI) API
and kubelet extensions required for this functionality, and to provide a clear path for iteratively building
on top of these APIs to address the broader set of use cases and requirements. In this KEP, checkpoints represent
the runtime state of a Pod, where the exact checkpoint format and low-level implementation details are left to
the container runtime (e.g., containerd, CRI-O), the OCI runtime (runc, crun), and the underlying checkpoint/restore
mechanism (e.g., CRIU, gVisor).

While Pod-level checkpointing is inspired by the existing [kubelet checkpoint] API and extends this concept
to Pod level, the restore functionality represents a larger addition as Kubernetes currently supports container
restore only with [OCI image annotations]. This proposal intentionally approaches Pod-level CRI APIs for both
checkpoint and restore as a single, cohesive feature, recognizing that checkpointing without restore would be
incomplete and impractical for many real-world use cases that require both capabilities. The kubelet API for
Pod checkpointing is designed to invoke this CRI API, while the kubelet restore functionality will be explored
in a future iteration of the enhancement proposal.

## Motivation

The current [kubelet checkpoint] API was originally inspired by the checkpoint/restore functionality
of container engines such as Podman. However, unlike these container engines, Kubernetes is responsible
for managing, scaling, and coordinating workloads across an entire cluster of machines. As a result,
container-level checkpointing alone does not adequately support many Kubernetes-native workflows and
higher-level operations that require preserving and restoring the full Pod state. This KEP aims to
remove this barrier by enabling a Pod-level checkpoint and restore mechanisms that align with the
Kubernetes core abstractions.

### Goals

The goal of this KEP is to introduce support for Pod-level checkpoint and restore to the CRI API,
and extending the *kubelet* API to support checkpointing the runtime state of a Pod, including
running containers and Pod-level metadata.

### Non-Goals

The list below outlines extensions and out-of-scope items that can be addressed in future enhancement proposals.

- Pod live migration semantics are considered out of scope for this KEP. This functionality requires
  an efficient handling and transfer of the checkpoint state as well as addressing a number of
  non-trivial challenges, such as migrating the IP address of a Pod from one node to another and
  preserving established TCP connections. Some of these challenges have been partially addressed,
  for example, with [criu-image-streamer] and [TCP connection repair], however, once Pods are
  scheduled, they are bound to a specific node (`pod.Spec.NodeName`), and Kubernetes does not
  provide guarantees around preserving network identity or other node-specific runtime characteristics
  across restores. Thus, this KEP focuses on enabling Pod-level checkpoint and restore as a foundational
  building block, leaving full live migration semantics to be explored in future enhancements.

- Checkpoint and restore of shared Pod resources is out of scope for the initial implementation. This KEP
  focuses primarily on capturing and restoring the execution state of containers and Pod-level metadata.
  Support for shared resources such as shared memory, volumes, and Dynamic Resource Allocation (DRA) devices
  will be considered in future iterations and enhancement proposals.

- Scheduler and API Server integrations are out of scope for this KEP. This proposal focuses on defining
  the core kubelet and CRI extensions required to enable Pod-level checkpoint and restore. Integrating
  this functionality with the Kubernetes scheduler and API Server to enable use cases such as
  interruption-aware scheduling and Pod migration will be addressed in subsequent enhancements.

- Handling of exec sessions and port-forward is out of scope for the initial implementation.
  Support for preserving and restoring exec sessions and port-forward can be explored in a
  future enhancement proposal.

- Checkpoint portability across heterogeneous environments such as different CPU and GPU architectures,
  kernel versions, container runtimes, or device drivers, are out of scope for this proposal.

- Defining resource quotas, limits, and policies around Pod checkpointing is out of scope for this KEP.
  Considerations such as the maximum number of checkpoints per namespace, total disk space used by
  checkpoints, and retention policies for checkpoints will be explored in future iterations of the
  enhancement proposal.

- Checkpoint lifecycle management is out of scope for the initial implementation and will be
  explored in future iterations of the enhancement proposal.

- When creating multiple clones from the same checkpoint, the workload may need to be aware of this
  to refresh state like session keys, random number generator states, and certificates. A future KEP
  will explore a common mechanism that applications can use to be notified being cloned. For example,
  gVisor provides a special file (`/proc/gVisor/checkpoint`) that blocks until a restore is complete.
  This allows applications to read from this file, refresh their state when the read completes,
  and then read it again to wait for the next event.

## Proposal

### Implementation

In this proposal, we aim to provide CRI functionality to checkpoint and restore a running Pod, which
includes all containers running in the Pod, along with Pod-level metadata and configurations. This
functionality is inspired by [kubelet checkpoint], but extends it to the Pod level, allowing to capture
and restore the execution state of a Pod, rather than individual containers. The exact implementation
details of this checkpoint/restore mechanism are left to the container runtime, but we expect the Pod
checkpoint to capture the complete execution context of all processes running in containers, including
in-memory state, process hierarchies, open file descriptors, and Pod-level configuration and metadata.

A high level view on the implementation is that triggering the kubelet Pod checkpoint API endpoint
will trigger the `CheckpointPod` CRI API endpoint, which will create a checkpoint of the running Pod
sandbox and store it in a default location. The checkpoint can then be used to restore the Pod sandbox
by triggering the `RestorePod` CRI API endpoint with the location of the checkpoint and an optional
new Pod configuration.

### User Stories

This KEP is intended as an incremental milestone rather than a complete solution. The user stories
outlined here will not be fully satisfied by this proposal alone. These gaps are intentional and are
expected to be addressed in subsequent KEPs or follow-up work as the design and implementation evolve.

#### Optimizing resource utilization for interactive workloads

Interactive GPU workloads, such as such as computational notebooks (e.g., JupyterHub) and AI platforms
(e.g., Open WebUI, Ollama), are becoming increasingly popular in scientific research and data analysis.
However, efficiently allocating GPU resources in multi-tenant Kubernetes environments is challenging due
to the unpredictable usage patterns of these workloads. Pod checkpoint/restore enables dynamic
reallocation of GPU resources based on real-time workload demands, allowing cluster operators to optimize
resource utilization without the need for modifying existing applications.

#### Accelerating startup of applications with long initialization times

The cold start time of many applications, such as LLM inference services and Java applications, can often
reach several minutes due to a sequence of complex initialization steps that must complete before the
service can accept requests or process data. Pod checkpointing allows to transparently save the
initialized state of running applications to persistent storage, and and later restore it on demand,
enabling services to resume execution without repeating expensive initialization steps.

#### Enabling fault-tolerance for long-running workloads

Training jobs for large AI models such as LLaMA 4 run on hundreds or thousands of GPUs and often execute
for weeks or months. In these long-running workloads, hardware and system failures are inevitable and can
otherwise force jobs to restart from scratch, resulting in significant loss of time and computational
resources. Pod-level checkpointing allows to transparently capture the state of these workloads and to
resume execution from the most recent snapshot in the case of failures. For example, when a KubeFlow
TrainJob is preempted by Kueue, Pod-level checkpoint/restore can be used to capture, and later resume
the runtime state to avoid restarting the training job (https://github.com/kubeflow/trainer/issues/2777).

#### Interruption-aware scheduling with checkpoint/restore

Kubernetes clusters managing large-scale compute infrastructure often orchestrate millions of
simultaneously running jobs. Although many applications are designed to be resilient to failures,
evictions due to resource contention and scheduled maintenance events reduce the overall efficiency
due to the time required to rebuild complex application state. Interruption-aware scheduling aims to
leverage the Pod checkpoint/restore capabilities to preserve the application state of preempted
workloads and enable migration between machines.

#### Pod migration across nodes for load balancing and maintenance

Cluster operators often need to rebalance workloads across nodes to respond to changing resource
requirements or planned maintenance events such as kernel upgrades, security patching, or node
replacement. These operations typically rely on Pod eviction and rescheduling, which forces applications
to restart and rebuild in-memory state.  This restart can lead to potentially significant downtime for
long-running and stateful workloads. The checkpoint and restore functionality enables running Pods to be
migrated from one node to another. Unlike live migration scenario, this approach does not require
iterative memory synchronization or strict latency guarantees, making it more practical to deploy at
scale within the existing Kubernetes model. While some downtime is expected during the checkpoint and
restore phases, the preserved execution state significantly reduces recovery time compared to full Pod
restarts.

#### Investigating security incidents with forensic checkpointing

Building on the use case described in KEP-2008, Pod-level checkpointing extends forensic capabilities
from individual containers to the full Pod, enabling forensic readiness in Kubernetes clusters. Forensic
readiness refers to the ability of a system to proactively prepare for security investigations by
ensuring that forensic evidence can be captured quickly, consistently, and with minimal operational
impact when an incident occurs.  Pod-level checkpointing enables cluster operators and security teams to
integrate forensic data collection into incident response workflows, allowing clusters to remain
operational while preserving evidentiary data. The Pod checkpoints can be treated as immutable
evidence artifacts and securely exported for offline analysis. Investigators can then examine the
complete snapshot of the Pod’s execution context, including in-memory state, process hierarchies,
open file descriptors, network connections, mounted volumes, and Pod-level configuration and metadata.
This approach preserves the relationships between co-located containers, allowing investigators to
reconstruct attack timelines, identify privilege escalation paths, and understand cross-container
interactions.

### Risks and Mitigations

The main risk we face is the complexity of implementing Pod-level checkpoint and restore within the
scope defined by the Non-Goals above, particularly in a way that is portable across different container
runtimes and Kubernetes environments, while also ensuring security and reliability. We aim to mitigate
this risk by defining a minimal set of kubelet and CRI extensions that enable an iterative approach
to implementing this functionality, allowing for incremental improvements and adjustments based on
real-world feedback and use cases.

Some specific challenges and mitigations include:

- Checkpointing workloads that run on NVIDIA GPUs typically requires the GPU state to be temporarily
  saved into host memory. This results in increasing the memory requirement for the container, and it
  might be needed to adjust the memory limit while checkpoint/restore is in progress.

- To prevent checkpoint failures caused by transient processes (e.g., from exec probes, kubectl exec,
  attach sessions, or logging agents), the kubelet should temporarily suspend all probe executions for
  a Pod during its checkpointing window. As described in the Non-Goals, preserving exec/attach sessions
  and port-forwarding semantics is out of scope for the initial implementation. And because some probes
  use exec sessions, those will also be out of scope. The handling of active exec or attach sessions at
  checkpoint time is implementation-specific and currently may vary across OCI runtimes (runc, gVisor).
  Whether the kubelet rejects a checkpoint request in such cases will be clarified in a follow-up iteration.

- Checkpointing applications that are distributed across multiple Pods requires coordination mechanisms
  to ensure consistency across checkpoints. As noted in the Non-Goals, cross-Pod coordination are out of
  scope for this KEP and must be handled by external tools ([criu-coordinator]) or  application-level
  synchronization.

- During the checkpointing window, the containers running in the checkpointed Pod are temporarily placed
  in a frozen (paused) state to facilitate the creation of a consistent checkpoint. The duration of this
  window can vary based on the workload (e.g., the amount of memory being used at the time of checkpointing)
  and the underlying checkpointing mechanism. This can lead to temporary unavailability of the applications.
  To mitigate this, the paused/checkpointing state must be exposed by the container runtime via the Pod and/or
  Container Status API to allow clients to detect it. To enasure checkpoint consistency and prevent unexpected
  failures, During this checkpointing window, Ephemeral Containers may not be started for the checkpointed Pod,
  and any existing Ephemeral Containers should be frozen (paused) along with the regular containers.

- Large checkpoint artifacts can consume significant disk space. Their size depends on several factors,
  including the container root filesystem writable layer, the memory usage of running processes at the
  time of checkpointing, and any applied data compression. This makes precise estimation in advance difficult.
  Disk space requirements and storage limits can be approximated by considering the application CPU and GPU
  memory footprint at the time of checkpointing together with the writable layer size. To prevent node disk
  pressure, checkpoint retention and deletion mechanisms, as well as appropriate storage limits, must be
  configured in advance.

## Design Details

This KEP proposes the following CRI APIs for Pod-level checkpoint/restore, inspired by the
ContainerCheckpoint API. Exposing this APIs to the API server is left as an extension to be
explored in a future enhancement proposal, that will take into consideration of the API design,
security implications, and integration with Kubernetes scheduling and lifecycle management.

### CheckpointPod

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

### RestorePod

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

The Pod-level checkpointing operation can be performed only on running Pods, where the Pod has been bound
to a node, all of the containers have been started, and at least one container is running. During this
operation, all running containers are temporarily frozen (paused) to create a consistent checkpoint.

### Restore Semantics

In this proposal, we focus on providing only the CRI API for restoring a Pod from a checkpoint,
and the kubelet restore functionality will be explored in a future iteration of the enhancement
proposal. Restoring a Pod from a checkpoint may result in either:

- Recreating a Pod with a new identity, or
- Restoring the original Pod identity

The initial implementation focuses on restoring Pods as new objects to enable use cases such as creating
multiple Pods from a single checkpoint and accelerating application startup. Restoring Pods with their
original identity for use cases such as live migration may be considered in future proposals.

### Security Implications

Similar to the ContainerCheckpoint API described in [KEP-2008], the Pod-level checkpoint and restore
APIs described in this proposal are restricted to users with privileged access to the kubelet API.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `pkg/kubelet`: `2026-02-13` - `71.5%`
- `pkg/kubelet/container`: `2026-02-13` - `81.6%`
- `pkg/kubelet/server`: `2026-02-13` - `73.1%`
- `pkg/kubelet/kuberuntime`: `2026-02-13` - `68.4%`

##### Integration tests

CRI API changes need to be implemented by at least one container engine. Since kubelet does not
have integration tests, validation will be performed using `test/e2e_node`, which effectively
serves as kubelet's integration test suite. These tests will verify that the kubelet can
successfully call the new CRI APIs and correctly handle responses and error scenarios.

##### e2e tests

Alpha release will include e2e tests with the expectation that no CRI implementation provides the newly
added RPC calls. Once CRI implementation provide the relevant RPC calls the e2e tests will not fail
but need to be extended.

### Graduation Criteria

#### Alpha

- Implement CRI API extensions for CheckpointPod and RestorePod.
- Introduce the new feature gate and minimal kubelet implementation to support Pod-level checkpoint.
- Implement e2e tests to validate the core functionality of checkpointing Pods.

#### Beta

- Mainstream container runtimes and low-level container runtimes (e.g., containerd/CRI-O, runc/crun) have
  released generally available versions that support Pod-level checkpoint and restore.
- Additional e2e testing to ensure feature availability and stabilization.
- Document the limitations of the feature and any known issues or gaps that need to be resolved before GA.

#### GA

- Gather feedback from real-world users.
- The feature has been stable in Beta for at least 2 Kubernetes releases.
- Multiple major container runtimes support the feature.

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

This feature is entirely local to the Kubelet. There is no dependency on the
control plane or other components, however, it is worth considering the
version skew between the kubelet and the container runtime:

- If the kubelet supports the new CRI API but the container runtime does not,
  the kubelet will return an error when the checkpoint Pod API is called,
  and the feature will not be available.

- If the container runtime supports the new CRI APIs but the kubelet does not,
  the feature will not be available since there is no kubelet API to trigger
  the Pod-level checkpoint operations.

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

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

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

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason:
- [ ] API .status
  - Condition name:
  - Other field:
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The expectation is that it should always succeed. A failed checkpoint does not
break the actual workload. A failed checkpoint only means that the checkpoint
request failed without effects on the workload. The expectation is also that
checkpointing either is always successful or never. From today's point of view this
means that the expectation is 100% availability or 0% availability. Experience
in Podman/Docker and other container engines so far indicates that.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Currently the *kubelet* collects metrics in the bucket `checkpointpod`. This can be
used to determine the health of the service.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

CRI stats will be added for this as well as kubelet metrics tracking whether an
operation failed or succeeded.

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

This feature introduces new CRI APIs for Pod-level checkpoint and restore.
The kubelet will invoke the checkpoint API when a checkpoint operation is
explicitly triggered. No periodic or background API calls will be made.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. It will only affect checkpoint and restore CRI API calls.

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

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 2026-01-29: Initial version of this KEP

## Drawbacks

There are no drawbacks that we are aware of.

## Alternatives

- Container-level checkpointing
  - Rejected because it cannot preserve runtime state in shared namespaces or multi-container consistency.
