# KEP-5972: Dynamic Containers

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Hierarchical Scheduling](#hierarchical-scheduling)
  - [Decoupling Runtime Initialization](#decoupling-runtime-initialization)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Autonomous Framework Governance (Ray, Slurm)](#autonomous-framework-governance-ray-slurm)
    - [The Agentic Sandbox (AI-Native)](#the-agentic-sandbox-ai-native)
    - [Stateful Fast-Path (Hot-Swap Migration)](#stateful-fast-path-hot-swap-migration)
    - [In-Place Upgrades (Sidecars, Daemons)](#in-place-upgrades-sidecars-daemons)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Third-Party Controllers that assume containers are static](#third-party-controllers-that-assume-containers-are-static)
    - [Container Status update API load](#container-status-update-api-load)
    - [Changes to Resize behavior](#changes-to-resize-behavior)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
    - [Dynamic Subresource](#dynamic-subresource)
    - [Limitations](#limitations)
    - [No SecurityContext escalations](#no-securitycontext-escalations)
    - [Container Status](#container-status)
    - [Allocated Subresource](#allocated-subresource)
  - [Allocation](#allocation)
    - [Image update allocation](#image-update-allocation)
  - [Container Termination](#container-termination)
  - [Security Considerations](#security-considerations)
  - [Implementation Details](#implementation-details)
  - [Test Plan](#test-plan)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Allocation Checkpoint](#allocation-checkpoint)
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
  - [Optimized Pods](#optimized-pods)
  - [Ephemeral Containers](#ephemeral-containers)
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
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Dynamic Containers is a feature that lets *main* containers be added to and removed from a pod while it's running. By decoupling scheduling, initialization, and workload execution, this feature transforms the Kubernetes Pod from a static list of containers into a highly dynamic execution envelope, enabling high-churn, low-latency container operations.

## Motivation

Dynamic Container mutation is the next step in an ongoing effort to transform Kubernetes into a more dynamic execution environment that is capable of meeting the evolving demands of modern AI workloads, including high-performance batch workloads leveraging special-purpose frameworks, and low-latency execution environments for agentic workloads.

### Hierarchical Scheduling

Hierarchical scheduling delegates fine-grained resource management and container lifecycles to specialized Control Planes (e.g., Ray, Slurm) within a Kubernetes-allocated Pod (representing a "Resource Budget"). This approach allows high-performance frameworks to maintain their specialized scheduling advantages while leveraging Kubernetes's managed infrastructure.

Under this model, Dynamic Containers can be used as part of a **two-tier scheduling architecture** that allows Kubernetes to act as the macro-scheduler while frameworks handle micro-scheduling:
  - **L1 (Kubernetes):** Orchestrates global placement (e.g., PodGroups) and enforces a secure "Resource Envelope" (the Pod).
  - **L2 (Framework CP):** Specialized CPs perform fine-grained L2 scheduling *inside* that envelope—partitioning CPU/Memory and managing sub-cgroups in real-time.

### Decoupling Runtime Initialization

For latency-critical workloads, pods can be pre-warmed so they are ready to receive worker containers. This can include:
  - Scheduling & resource allocation
  - Pod Sandbox startup (`RunPodSandbox`), including CNI initialization
  - Volume attachment & initialization
  - Device attachment & initialization
  - Generic application-level initialization (via InitContainers)

Decoupling these elements from workload startup will be critical in achieving the sub-100ms latency required for interactive agentic workloads.

### Goals

  - Decouple scheduling, initialization, and workload execution.
  - Enable high-churn addition and removal of containers.
  - Enable low-latency workload startup, targeting < 100ms of overhead.

### Non-Goals

The scope of this KEP is deliberately minimized for more effective execution. I expect these non-goals to be addressed in subsequent KEPs at a later date.
  - Introducing new pod states or phases.
  - Enabling dynamic volume management (aside from mounting preallocated volumes).
  - Enabling "dynamic DRA" (i.e. mutable resource claims on running pods).
  - Introducing an alternative request path to mutate pods that bypasses the cluster API server.
  - Changes impacting pod creation: this KEP is purely scoped to behavior on pod update.

## Proposal

### User Stories

#### Autonomous Framework Governance (Ray, Slurm)

This enables specialized Control Planes (Raylet/Slurmlet) to act as "Local Pod Controllers".
  - **Value:** Low-latency task spawning and guaranteed framework stability without re-invoking the Kube-Scheduler.

#### The Agentic Sandbox (AI-Native)

An "Agent" container spawns ephemeral tool-execution sandboxes on-the-fly within preallocated pods.
  - **Value:** By bypassing the Kubernetes control plane, agents can interact with tools with minimal overhead.

#### Stateful Fast-Path (Hot-Swap Migration)

A "Shadow Shell" pod is pre-scheduled on a target node. During migration, the workload state is streamed node-to-node and injected directly into the waiting shell.
  - **Value:** The workload resumes execution in **sub-second levels**, while the KCP reconciles the new Pod status in the background.

#### In-Place Upgrades (Sidecars, Daemons)

Non-disruptive in-place upgrades of sidecar containers and system workloads.
  - **Value:** The sidecar container is swapped out for an upgrade version with minimal downtime. In
    some cases, a blue-green deployment strategy can be used for zero-downtime upgrades.

### Risks and Mitigations

#### Third-Party Controllers that assume containers are static

**Risk:** The Kubernetes ecosystem (including operators, service meshes, logging forwarders,
CI/CD tools) may assume a Pod's `.spec.containers` array is strictly immutable after
creation. If a container is added dynamically, external controllers that rely solely on cached
`CREATE` events may miss the workload entirely, panic due to index out-of-bounds errors, or fail to
inject necessary sidecars/environment variables.

**Mitigation:** This represents a paradigm shift in Kubernetes workload APIs. Dynamic Containers
will be opt-in via a feature gate. We will provide extensive documentation and release notes
outlining that `.spec.containers` is no longer static, guiding ecosystem maintainers to update their
controllers to reconcile on Pod `UPDATE` events instead of relying on cached initialization state.

#### Container Status update API load

**Risk:** The ability to add and remove short-lived containers at high frequencies will lead to a
proportional surge in Pod status updates. A single short-lived container generates multiple status
transitions (`Waiting/Unallocated` -> `Running` -> `Terminated`). At scale, this could overwhelm the
Kubelet's status manager, degrade `kube-apiserver` performance, and exhaust etcd write capacity.

This risk is not directly addressed by this KEP, but we may explore options to batch and rate limit
pod status updates in the future. Status update coalescing is already being explored for init
containers in v1.37.

#### Changes to Resize behavior

**Risk:** This proposal gates container image updates on the atomic allocation step. Currently,
users expect an image update to trigger a container restart immediately, even if an accompanying
resource resize is deferred due to node capacity. Changing this behavior means image updates could
unexpectedly be blocked by resource allocations.

**Mitigation:** This change corrects a race condition/atomicity violation, ensuring Pod updates are
treated as proper transactions (we shouldn't run a new image if we can't secure the resources for
its accompanying spec). We will explicitly document this behavioral shift. Image updates blocked by
a deferred resize will be recorded in a Kubelet log line.

## Design Details

### API Changes

This proposal does not introduce any new fields. The API changes are limited to two new subresources

1. `/dynamic`: Enables the new update functionality, in addition to updates allowed on the main
resource, and `/resize`.
2. `/allocated`: Returns the allocated version of the pod, served directly from the Kubelet.

#### Dynamic Subresource

Dynamic container mutation is allowed only through the new `/dynamic` subresource. Update
permission on this subresource is NOT granted to the default `edit` Role, and must be explicitly
granted by a `cluster-admin`.

Containers are added and removed via an update on the pod resource.
  - Multiple containers can be added and removed in a single request.
  - Standard pod validation applies to the updated pod.
  - Additional validation applied to cover the limitations listed below.
      - The container name must be unique among all containers AND all container statuses.
  - `.spec.containers` must be non-empty (i.e. cannot remove the last container).
  - Container changes cannot be made once a `DeletionTimestamp` is set on the pod.

In addition to adding & removing containers, container & pod resizes will also now permitted via
this `/dynamic` subresource, along with updates allowed on the main pod resource (including
updates to images, gracePeriods, etc.). The `resize` subresource will still be supported so that
permission to resize *without* permission to modify running code can be granted. The same validation
rules apply.

Future extensions that enable additional pod dynamism and mutability (such as dynamic volume
provisioning) will be added to this same subresource.

#### Limitations

To scope down the proposal for the initial release, the following limitations are put in place.
These are not inherent requirements for the feature, and can be reevaluated as new additive features
in the future.

  - Only main containers can be added or removed (not init containers).
  - Container `SecurityContext` cannot escalate permissions. See below.
  - The pod must be in an initialized state (i.e. all init containers have completed) before any dynamic container changes can be made.
  - This proposal does not allow for general container mutation. A corollary is that a container
    with the same name can only be added after the previous container with that name is *completely*
    removed (see discussion of termination below).
  - Container resources cannot change the QOS class of the pod (in other words, containers added to a best-effort pod cannot specify resource requirements).
  - Containers can only mount volumes or name ResourceClaims already mounted/claimed by the pod.
  - Resize limitations apply: Windows is not supported, swap enabled pods are not supported.
  - Because container resources are treated as a resize, added or removed Containers can only request resources that are supported by in-place resize.
  - A pod must have at least one main container.
  - Privileged containers cannot be added.
  - HostPorts cannot be used by newly added containers.

#### No SecurityContext escalations

Dynamically added containers are forbidden from escalating the SecurityContext permissions of the
pod. In other words, they can only add or allow permissions already granted to other containers in
the pod. More specifically:

- `Capabilities`:
  - Cannot add a capability that isn't already added by an existing container.
  - Must drop any capabilities that are dropped by ALL existing containers.
- `Privileged`: Never allowed.
- `SELinuxOptions`, `RunAsUSer`, `RunAsGroup`, `SeccompProfile`, `AppArmorProfile`: Can only use values already used by an existing container (or the PodSecurityContext)
- `WindowsOptions`: N/A (windows not supported)
- `RunAsNonRoot`, `ReadOnlyRootFilestystem`: Must be set if ALL containers set these.
- `AllowPrivilegeEscalation`: Must be set to `false` if ALL containers explicitly disable (the implicit default is `true`).
- `ProcMount`: Can only be set to `Unmasked` if another container already has an unmasked proc mount.

#### Container Status

Containers that have been added but not allocated will generate a `ContainerStatus` with the
`Waiting` state, and the reason `Unallocated`.

Containers that have been removed from the allocation will remain in the `Running` state until
terminated. After termination, the container status will remain until garbage collected (see below).

#### Allocated Subresource

A new read-only `/allocated` subresource on pods will surface the allocated pod spec. The allocated
pod will be fetched directly from the Kubelet on-demand, thus avoiding additional storage overhead
in the API server.

The Kubelet will serve a new `/allocatedPods` endpoint that functions similarly to `/pods` and
`/runningPods`, and returns a list of all the allocated pods on the node. An allocated pod is
represented as a `v1.Pod` object, but the status field is left blank. The subpath
`/allocatedPods/<POD UID>` can be used to fetch an individual allocated pod, which is what the API
server will use to serve the allocated pod subresource.

**Justification:** The desired pod spec (stored in the pod resource) can diverge from the allocated
pod spec (stored locally by the Kubelet) for an arbitrary amount of time. For in-place resize, this
was handled by surfacing allocated resources in the pod status. This KEP (and the expected follow
ups) significantly increases the API surface of the mutable / allocatable portion of the Pod spec,
and simply mirroring these fields into the pod status will drastically inflate the size of the pod
object. Similarly, duplicating every pod into an `AllocatedPod` object would heavily inflate the API
server object count and data size.

### Allocation

Newly added containers go through an allocation step that works exactly the same as in-place pod
resize. In other words, adding a new container with additional resources is considered a pod resize,
and can put the pod into a deferred (or even infeasible) state.

Allocation is an atomic operation, so if multiple containers are added and removed together, the changes will only be allocated if ALL of the changes can be allocated.

Allocation will save (and checkpoint) the full container spec for any allocated containers (see [Image update allocation](#image-update-allocation) for an additional implication of this change).

Once allocated, the container changes will be made during the next pod sync. `UpdatePodFromAllocation` will modify the pod spec to reflect the allocated containers.

Adding containers is straightforward: the Kubelet sees that the new container is not running and adds it. Removal is more complicated.

Allocation will be halted once a pod enters a terminating state.

#### Image update allocation

Today, image update and resource allocation are totally separate. This means that if I resize a
container, and the resize is deferred, and I also change the container image, the Kubelet will
restart the container with the new image even while the resize is still deferred. This violates the
atomic allocation principle.

Starting with the enablement of the `DynamicContainers` feature, container image update will also be
gated by the atomic allocation step. This means that in the above scenario, the Kubelet would *not*
restart the container with the new image until the resize (and any other changes) can be allocated.

### Container Termination

When a running container is removed from a pod (and the changes are allocated), the Kubelet must terminate the container. `computePodActions` will be updated to identify containers that are running but not part of the PodSpec and queue them up for a `killContainer` action.

`kuberuntime_manager.killContainer` already supports reading the required container information from the container runtime annotations, so we do not need to store the container spec anywhere else. The operation also respects the grace period.

Once a removed container is terminated, its `ContainerStatus` will be kept in the pod status. The Kubelet will preserve removed container statuses in the API when it updates the pod status. Garbage collection will be applied to removed container statuses.

1. Keep up to N (maybe 10) removed container statuses: After more than N removed container statuses have accrued, delete the oldest status.
2. Enhance `containerGC` to clear the container status when the actual container object is garbage collected.

Logs from removed containers will be managed by the [existing container garbage collection](https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/kuberuntime/kuberuntime_gc.go).

### Security Considerations

**Admission Configuration:** If custom admission webhooks are configured to intercept only `CREATE`
operations for Pods, they can be bypassed by adding a violating container via update. Container image is already mutable,
so the only new risk introduced is granting new permissions. This is mitigated by explicitly forbidding new containers from escalating permissions.
See [No SecurityContext escalations](#no-securitycontext-escalations).

### Implementation Details

**Probe Manager**
  - Currently: sets up container probes when the pod is added
  - Proposed: Add/start container probes when starting a container, and remove/stop probes when a container is terminated.
  - Note: adding a new container with a readiness probe will cause the pod to become unready, until that container is ready.

**Topology Managers**
- Depends on `InPlacePodVerticalScalingExclusiveCPUs` / `InPlacePodVerticalScalingExclusiveMemory`.
  Passing the allocation step involves running the pod through admission again, which is where the
  CPU & memory allocation is managed. No updates should be needed for container additions. Whether
  changes are needed for remove depends on how resource downsizing is implemented for the above
  features.

**Allocation Manager Admission handling**
  - Treat new containers as an allocation step, similar to resize.

**Admission Handlers** need to handle updates to the pod:
  - `PodResizeValidator`
  - `LimitRanger`

### Test Plan

[ ] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

##### Unit tests

- `pkg/kubelet/kuberuntime`: Validate `computePodActions` correctly identifies and gracefully terminates unallocated containers.
- `pkg/kubelet/status`: Test logic surrounding status batching and the GC retention limit for `ContainerStatuses`.
- `pkg/api/pod`: Validate admission updates (ensuring unique names, preventing init container modifications, and blocking privileged containers).
- `pkg/apis/core/validation`: Coverage of validation changes.
- `pkg/kubelet/prober`: Coverage of updated probe manager functionality.
- `pkg/kubelet/allocation`: Coverage of updated allocation functionality.

##### Integration tests

- Test routing for the newly proxy-served `/allocated` subresource.

##### e2e tests

- End-to-end lifecycle verification: Add a container -> Verify `Running` -> Remove container -> Verify `Terminated` -> Verify GC limits cap the status list at 10.
- Expand resource quota & resize test cases to include Dynamic Containers coverage.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag (`DynamicContainers`).
- Initial e2e tests completed and enabled.
- API validation and Kubelet `/allocatedPods` endpoint functional.

#### Beta

- Gather feedback from developers and ecosystem maintainers (e.g., Ray, Slurm).
- Ecosystem research & outreach for static container assumptions.
- Decide on (and implement) strategy for legacy admission controllers.
- Decide on strategy for resolving the Kubelet / Scheduler resize race condition.

#### GA

- Conformance tests expanded to cover dynamic container APIs.

### Upgrade / Downgrade Strategy

- **Upgrade:** Enabling the feature gate introduces no behavioral changes to existing pods until a
  client explicitly issues an `UPDATE` that modifies `.spec.containers`.

- **Downgrade:** If disabled, the API server will reject any further structural changes to
  `.spec.containers`. Kubelet will continue to operate on the final pod spec.

#### Allocation Checkpoint

The format of the allocation manager state checkpoint will change to checkpoint the entire pod spec,
necessitating a new version of the checkpoint. The new schema must use field names that don't
collide with the previous version to avoid unmarshalling errors.

### Version Skew Strategy

The Kubelet will declare support for DynamicContainers in it's declared features, and an `update` gating feature will be added in the NodeDeclaredFeatures framework.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `DynamicContainers`
  - Components depending on the feature gate: `kube-apiserver`, `kubelet`

Additionally, a cluster admin can prevent dynamic container mutation with a ValidatingAdmissionPolicy, such as:

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: disable-dynamic-containers
spec:
  failurePolicy: Fail
  matchConstraints:
    resourceRules:
      - apiGroups:   [""]
        apiVersions: ["v1"]
        operations:  ["UPDATE"]
        resources:   ["pods/dynamic"]
  validations:
    - expression: >-
        object.spec.containers.map(c, c.name) == oldObject.spec.containers.map(c, c.name)
      message: "Dynamic Containers are disabled: adding or removing main containers is not allowed."
```

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling it will cause the API server to reject subsequent structural changes to `.spec.containers`.

###### What happens if we reenable the feature if it was previously rolled back?

The API server will resume accepting additions and removals to `.spec.containers`, and the Kubelet will resume reconciling the changes.

There are no storage changes in the API.

###### Are there any tests for feature enablement/disablement?

None planned.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollout failures will not impact existing workloads that do not attempt to mutate containers.
Rollbacks are safe; they block future dynamic additions but leave currently running containers
intact until they naturally terminate.

###### What specific metrics should inform a rollback?

- Significant increase in API server 5xx errors or latency for `UPDATE /pods` calls.
- High error rates in Kubelet logs regarding `computePodActions` or allocation failures.
- Unacceptable increases in etcd disk write latency due to status churn.
- Third party controller error rates.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Operators can monitor for Pods that have `ContainerStatus` entries with the `Waiting` state and `Unallocated` reason, or monitor API Server audit logs for `UPDATE` events modifying `.spec.containers`.

###### How can someone using this feature know that it is working for their instance?

Monitor container status for newly added or removed containers.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- Dynamic container addition to `Running` state transition overhead < 500ms (excluding container image pulling & assuming resource availability).

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Generic Kubelet & CRI health metrics.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

TODO

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No, but other services may need to be updated to account for mutable containers.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes. `UPDATE` calls to the `/pods` resource will increase as workloads add and remove containers.
Kubelet `PATCH` calls to `/status` will also increase proportionally to container churn. Clients
querying the `allocated` subresource will trigger the API server to proxy calls to the Kubelet.

###### Will enabling / using this feature result in introducing new API types?

No new API objects, but a new read-only `allocated` subresource is introduced on the existing `Pod` resource.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes. The Pod object's `status.containerStatuses` will grow as containers are added and removed. This is strictly bounded by the GC limit of 10 removed container statuses.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

TODO

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No (not beyond what is already exposed by containers).

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The Kubelet will continue running existing containers. It will not be able to receive new container additions or process removals until the control plane is restored.

###### What are other known failure modes?

- **Container fails to allocate:**
  - Detection: Container stays in `Waiting` state with reason `Unallocated`, `PodResizePending` condition is present.
  - Mitigations: Ensure the node has sufficient resources or scale down other workloads.
  - Diagnostics: `PodResizePending` condition message.

###### What steps should be taken if SLOs are not being met to determine the problem?

Evaluate CRI operation latency metrics.

## Implementation History

- **2026-04-30:** Initial proposal shared with SIG-Node: https://docs.google.com/document/d/1LUttwty4xIU9gcLGSvnu1qVqAztxT3kl53ZdpM8q7o0/edit
- **2026-06-08:** Initial KEP drafted.

## Drawbacks

Mutating `.spec.containers` violates a long-standing architectural assumption in the Kubernetes
ecosystem that pods are fundamentally immutable execution envelopes. This will require updates to
third-party controllers, service meshes, and logging tools.

## Alternatives

### Optimized Pods

The primary problem this proposal aims to address is running a (potentially very short lived)
workload with minimal latency and overhead. Can we just optimize pod startup and overhead instead?

Even if we can optimize pod startup to within our target, this is still not accounting for potential
workload initialization costs. In other words, with Dynamic Containers I can create a pod that runs
several init containers to prepare for the workload, such as populating a volume, prepulling a
model, setting up a database sidecar, or performing a credential exchange. No matter how fast we get
minimal pod startup, we cannot optimize away these factors. Solving for this additional
initialization work would require complicated cross-pod coordination.

### Ephemeral Containers

Ephemeral containers are already mutable, so maybe we should expand the scope of ephemeral
containers rather than making regular containers mutable?

The largest gaps are that ephemeral containers are **not restartable or removable**. Additionally,
they lack supoort for:
  - Probes
  - Lifecycle hooks
  - Volume subpath mounts
  - Container ports
  - Resources

In aggregate, these gaps are significant enough that it would be a larger change to close these gaps
in ephemeral containers than make regular containers mutable.
