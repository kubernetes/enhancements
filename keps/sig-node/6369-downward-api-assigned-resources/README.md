# KEP-6369: Pod Assigned Resource Exposure via Downward API

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Reacting to an upcoming cpuset change during scale-down](#story-1-reacting-to-an-upcoming-cpuset-change-during-scale-down)
    - [Story 2: Discovering exclusive CPUs and memory NUMA nodes without reading cgroup files](#story-2-discovering-exclusive-cpus-and-memory-numa-nodes-without-reading-cgroup-files)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Relationship with PodLevelResourceManagers](#relationship-with-podlevelresourcemanagers)
- [Design Details](#design-details)
  - [Implementation](#implementation)
    - [<code>NodeDeclaredFeatures</code> Integration](#nodedeclaredfeatures-integration)
    - [Resource Field Extensions](#resource-field-extensions)
    - [Downward API Volume Exposure](#downward-api-volume-exposure)
    - [Downward API Environment Variable Exposure](#downward-api-environment-variable-exposure)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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
  - [1. Read the cgroup files from inside the container](#1-read-the-cgroup-files-from-inside-the-container)
  - [2. Query the kubelet pod resources endpoint](#2-query-the-kubelet-pod-resources-endpoint)
  - [3. Expose the assignments in the pod status](#3-expose-the-assignments-in-the-pod-status)
  - [4. Volume files only, without environment variables](#4-volume-files-only-without-environment-variables)
  - [5. Include the amount of memory in <code>assigned.memset</code>](#5-include-the-amount-of-memory-in-assignedmemset)
  - [6. Expose the assignments through DRA](#6-expose-the-assignments-through-dra)
  - [7. Inject the assignments as files, the way DRA device attributes are](#7-inject-the-assignments-as-files-the-way-dra-device-attributes-are)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This proposal extends the Downward API to expose the exclusive CPUs and the memory NUMA nodes assigned to a container, both as volume files and as environment variables. The key value of this KEP is that a workload can learn the cpuset it is about to be given before that set is applied, which is not possible from cgroup files or the pod status — those reflect only the currently applied set. This feature combined with [KEP-6122](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/6122-configurable-scaling-delay) which introduces guaranteed delay for scaling down exclusive CPUs, gives the workload a chance to learn about new cpuset and evacuate workload from CPUs to be released. This extension is controlled by the new `DownwardAPIAssignedResources` feature gate.

This KEP was split from [KEP-6122](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/6122-configurable-scaling-delay) because the two features (configurable scale-down delay and downward API exposure of assigned resources) are functionally independent. This separation improves document readability and reduces complexity, making each feature easier to understand and review.

## Motivation

Latency-sensitive applications often require exclusive CPUs to achieve predictable performance and resource isolation. These applications commonly use CPU affinity to minimize performance degradation caused by CPU migration.

The key motivation for this KEP is that a workload running on exclusive CPUs has no way to learn an upcoming change to its cpuset before that change is applied. Cgroup files and the pod status reflect only the set that is currently in effect, never the one that is about to be applied. When scaling down guaranteed QoS pods, containers need to know in advance which CPUs will be removed from their cpuset so they can take preparatory actions — such as migrating workloads away from affected CPUs — and avoid performance degradation caused by CPU migration, core sharing, and sudden CPU loss during the removal of active CPUs. The workload has no other way to learn which CPUs will remain in its pool after scaling down, because the kubelet makes that decision authoritatively. Together with [KEP-6122](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/6122-configurable-scaling-delay) (which introduces configurable scale-down delay), this feature allows latency-sensitive applications to obtain the assigned cpuset in advance via the downward API: the volume file is updated with the new set before that set is applied to the container.

Memory assignments are exposed from the start here. They were part of the original KEP-6122 draft, were dropped from its Alpha scope for timing reasons, and remained an Alpha2 criterion there; approval of KEP-6122 was explicitly conditioned on treating CPU and memory consistently. Covering both in this KEP's Alpha settles that, and the compact list representation described below was requested during review.

### Goals

* Expose CPU and Memory assignments to containers via the downward API with `DownwardAPIAssignedResources` feature gate enabled:
   + `assigned.cpuset`: The desired exclusive cpuset (Linux cpuset format, e.g. `0-3,7,12-15`). Empty string when no exclusive CPUs are assigned.
   + `assigned.memset`: The assigned memory NUMA nodes, in the same list format (e.g. `0-1`). Empty string when no memory is assigned.
* Integrate with `NodeDeclaredFeatures` so that a node declares whether it can serve these values, and the scheduler keeps pods requesting them off nodes that cannot.

### Non-Goals

* Allow containers to specify which CPUs to remove during scale-down.
* Let a pod influence which CPUs or memory NUMA nodes it is assigned. These values report a decision, they do not take part in making it.
* Expose anything beyond what is assigned to the container itself, such as the assignments of other pods or the node's full topology.
* Guarantee a window in which a workload can react to a change. That is the subject of [KEP-6122](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/6122-configurable-scaling-delay).
* Expose the pod-level pools — the CPU and memory "bubbles" — from the `PodLevelResourceManagers` feature ([KEP-5526](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/5526-pod-level-resource-managers)). These pools are an implementation detail of `PodLevelResourceManagers` and have no direct effect on containers running the workload. See [Relationship with PodLevelResourceManagers](#relationship-with-podlevelresourcemanagers) for details. (Note: container-level assignments allocated by `PodLevelResourceManagers` remain a goal of this KEP.)

## Proposal

This proposal extends the Downward API with two new values of the existing `ResourceFieldRef.Resource` field: `assigned.cpuset`, carrying the CPU Manager's cpuset for the container, and `assigned.memset`, carrying the Memory Manager's set of memory NUMA nodes. Exposure of both is gated by the `DownwardAPIAssignedResources` feature gate.

### User Stories

#### Story 1: Reacting to an upcoming cpuset change during scale-down

As a developer of a latency-sensitive workload running on exclusive CPUs, I want to scale my container vertically — both up and down — without disrupting traffic, so that I scale down to pay only for the resources I use, save energy, and free CPUs for other workloads; or scale up when I need to handle more traffic in rush hours.

Scaling down must not interrupt the traffic being processed, so once the decision to scale is taken the workload needs time to move its load off the cores that are about to be freed. Because the decision of which specific CPUs are assigned is made authoritatively by the kubelet, the workload must be able to learn that decision and evacuate the cores that will be released before it happens.

This is the only channel through which a workload can learn an assignment that has been computed but not yet applied. Cgroup files and the pod status reflect only the set that is currently in effect, never the one that is about to be applied. Together with [KEP-6122](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/6122-configurable-scaling-delay), the workload learns the cpuset it is about to be given: during a scale-down the volume file is updated with the new set before that set is applied to the container, so the workload can move work off the CPUs being removed. Without KEP-6122 the value still changes, but there is no guaranteed window in which to react.

#### Story 2: Discovering exclusive CPUs and memory NUMA nodes without reading cgroup files

As a developer of a latency-sensitive workload, I want to discover the exclusive CPUs and memory NUMA nodes assigned to my container without reading cgroup files, so that I can pin my worker threads to exactly those cores and allocate my buffers on the correct NUMA nodes.

A DPDK-style workload is given four exclusive CPUs and pins its worker threads to exactly those cores. It also allocates its buffers on the memory NUMA nodes it was actually assigned, rather than inferring them from the CPUs it happens to run on. Today it has to discover both from inside the container by reading its cgroup files, which is an implementation detail rather than a stable contract. With this feature it reads `assigned.cpuset` and `assigned.memset` instead — from an environment variable if it only needs the values at startup, or from a volume file if it wants to follow later changes.

The Downward API gives a single, uniform interface for both CPU and memory that is independent of the host operating system and also covers the advance-notice use case described in [Story 1](#story-1-reacting-to-an-upcoming-cpuset-change-during-scale-down).

### Notes/Constraints/Caveats

The two exposure channels have different update semantics, and a workload must choose accordingly:

- **Volume file** (`assigned.cpuset`, `assigned.memset`): updated whenever the assignment changes, including during a resize. A workload that needs to follow changes must read the volume file.
- **Environment variable**: evaluated when the container is created and not updated afterwards. It does not follow a resize; it is re-evaluated only when the container is recreated.

Both channels expose an empty string (`""`) when the container has no exclusive CPUs or no memory assigned, so a workload must handle the empty case rather than treat it as an error.

### Risks and Mitigations

**Architectural Coupling Concern:** The Downward API was designed to expose fields of the pod spec and status — declarative state the user wrote — whereas `assigned.cpuset` and `assigned.memset` are node-local runtime state computed by the CPU Manager and the Memory Manager. Exposing them this way couples the Downward API to those implementations, which could constrain how CPU management evolves, in particular alongside a DRA CPU driver. This concern was raised during the review of KEP-6122 and accepted there as a non-blocker for Alpha.

**Mitigation:** The coupling is bounded by the fact that the kubelet's static CPU policy and the DRA CPU driver cannot run on the same node, so the two mechanisms do not compete for the same workloads. Rather than being specific to the CPU Manager, the exposed file path is intended as a shared contract that a DRA driver can publish to as well; this direction was proposed by the dra-driver-cpu maintainers, and it is tracked in [dra-driver-cpu#181](https://github.com/kubernetes-sigs/dra-driver-cpu/issues/181). If the coupling has to be undone later, the pod status alternative in [Alternatives](#alternatives) covers the same use cases without naming a resource manager.

**Security Considerations:** No new information is disclosed. A container can already read both values from its own cgroup files — `cpuset.cpus` for the CPUs and `cpuset.mems` for the memory NUMA nodes — so this KEP changes how the values are delivered, not who can see them. The set of memory NUMA nodes does reveal part of the node's topology, but only the part already assigned to that container and already readable by it. No new data recipients are created: the volume file and the environment variable are visible to exactly the processes that can read the container's cgroup files today.

### Relationship with PodLevelResourceManagers

[PodLevelResourceManagers](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/5526-pod-level-resource-managers) (KEP-5526) introduces pod-level resource pools — a CPU set ("CPU bubble") and a set of memory NUMA blocks ("memory bubble") — from which individual containers receive their assignments. Both the CPU Manager and the Memory Manager use the same partitioning model: each pod-level pool is divided into exclusive per-container slices and a shared pool for the remaining containers. This KEP and KEP-5526 address different levels of the same stack:

- **Container-level assignments** (the exclusive CPUs and memory NUMA nodes each container actually runs on) are exposed by this KEP via `assigned.cpuset` and `assigned.memset`. When `PodLevelResourceManagers` is active, those container-level assignments are still made by the CPU Manager and the Memory Manager and are still meaningful to the workload, so they remain in scope.
- **Pod-level pools** (the CPU and memory bubbles themselves) are an implementation detail of `PodLevelResourceManagers` and have no direct effect on the containers running the workload. Exposing them is a [non-goal](#non-goals) of this KEP.

In short, this KEP exposes what each container is assigned regardless of whether `PodLevelResourceManagers` is involved; it does not expose the pod-level pools that `PodLevelResourceManagers` manages internally.

## Design Details

### Implementation

The feature builds on the existing Downward API framework. The `DownwardAPIAssignedResources` feature gate has a different job in each component: in kube-apiserver it decides whether a pod referencing `assigned.cpuset` or `assigned.memset` is accepted, and in the kubelet it decides whether those values are produced and whether the node declares the feature.

#### `NodeDeclaredFeatures` Integration

This feature integrates with the Node Declared Features framework ([KEP-5328](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/5328-node-declared-features), GA since v1.37), so that a pod asking for these values is only placed on a node able to serve it.

When the `DownwardAPIAssignedResources` feature gate is enabled, the kubelet declares `DownwardAPIAssignedResources` in `node.status.declaredFeatures` during bootstrap. The declaration depends on the feature gate alone, not on which resource manager policies are configured: it states that the node understands these values, and an empty value is the correct answer for a container with no exclusive CPUs or no assigned memory.

The scheduler infers that a pod referencing `assigned.cpuset` or `assigned.memset` requires the feature and only places it on nodes that declare it. This keeps such a pod off a kubelet that predates the feature, where the downward API setup for the container would fail. A pod that reaches a node without going through the scheduler — a static pod, or one placed by a custom scheduler — is not covered by this filtering; see [Version Skew Strategy](#version-skew-strategy) for what the kubelet does then.

Once the feature graduates to GA and the feature gate is removed, every kubelet serves these values and the declared feature is no longer needed. Declared features are temporary by design in KEP-5328 and are removed as part of the post-GA cleanup.

#### Resource Field Extensions

Two new values, `assigned.cpuset` and `assigned.memset`, are added to the existing `ResourceFieldRef.Resource` field:

* resource: limits.cpu
   + A container's CPU limit
* resource: requests.cpu
   + A container's CPU request
* resource: limits.memory
   + A container's memory limit
* resource: requests.memory
   + A container's memory request
* resource: limits.hugepages-*
   + A container's hugepages limit
* resource: requests.hugepages-*
   + A container's hugepages request
* resource: limits.ephemeral-storage
   + A container's ephemeral-storage limit
* resource: requests.ephemeral-storage
   + A container's ephemeral-storage request
* **resource: assigned.cpuset** *(NEW)*
   + **A container's desired set of exclusive CPUs.**
* **resource: assigned.memset** *(NEW)*
   + **A container's desired set of assigned memory NUMA nodes.**

Both values use the Linux list format — `0-3,7,12-15` for CPUs, `0-1` for NUMA nodes. They mirror the `cpuset.cpus` and `cpuset.mems` pair of the cgroup cpuset controller, where `cpuset.mems` is the set of memory NUMA nodes and is the established counterpart of `cpuset.cpus`. The amount of memory assigned is deliberately not part of `assigned.memset`, since the Downward API already exposes it through `limits.memory`.

#### Downward API Volume Exposure

The Volume Manager gets the CPU state from the CPU Manager, and writes it to the Downward API volume file `assigned.cpuset`, which exposes the cpuset to the container:
- **If the container has exclusive CPUs assigned**:
  - during a scale-down: the file contains the newly allocated cpuset (before it is applied to the container).
  - at other times: the file contains the cpuset the container currently holds.
- **Otherwise**: the file contains an empty string (`""`).

The Volume Manager gets the Memory state from the Memory Manager, and writes it to the Downward API volume file `assigned.memset`, which exposes the set of memory NUMA nodes to the container:
- **If the container has memory assigned**: the file contains those NUMA nodes (e.g. `0-1`).
- **Otherwise**: the file contains an empty string (`""`).

#### Downward API Environment Variable Exposure

The environment variable for CPU exposure gets the CPU state from the CPU Manager when the container is created:
- **If the container has exclusive CPUs assigned**: the variable exposes the exclusive cpuset.
- **Otherwise**: the variable is empty (`""`).

The environment variable for Memory exposure gets the Memory state from the Memory Manager when the container is created:
- **If the container has memory assigned**: the variable exposes those NUMA nodes (e.g. `0-1`).
- **Otherwise**: the variable is empty (`""`).

**Note:** An environment variable is evaluated when the container is created and is not updated afterwards, so it does not follow a resize; it is re-evaluated only when the container is recreated. Volume file values, in contrast, are updated whenever the assignment changes. A workload that needs that value to follow a resize must read the volume file.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None

##### Unit tests

We plan on adding or extending tests in the following files.

API validation:
- `pkg/apis/core/validation/validation_test.go`: `2026-09-19` - `88.4%` — the two new values accepted when the gate is enabled and rejected when it is disabled, for both a downward API volume item and a container environment variable.
- `pkg/api/pod/util_test.go`: `2026-09-19` - `72.1%` — wiring the gate into the pod validation options, and the ratcheting rule that keeps a value already in use permitted once the gate is off.

Producing the values:
- `pkg/volume/downwardapi/downwardapi_test.go`: `2026-09-19` - `53.2%` — writing `assigned.cpuset` and `assigned.memset` into the volume, an empty value when the container has no assignment, and an empty value when the gate is disabled.
- `pkg/kubelet/kubelet_pods_test.go`: `2026-09-19` - `84.6%` — the environment variable path, including that the value is taken at container creation and not refreshed afterwards.

Node Declared Features:
- `staging/src/k8s.io/component-helpers/nodedeclaredfeatures/features`: a new package for this feature and its registration, following the per-feature packages already present there.

##### Integration tests

- **Validation ratcheting**: with the gate disabled in kube-apiserver, creating a pod that references these values is rejected, while updating a pod that already references them — including through a resize — is accepted.
- **Scheduler filtering**, in `test/integration/scheduler/filters/`:
  - a pod referencing these values is scheduled to a node that declares `DownwardAPIAssignedResources`;
  - nodes that do not declare it are filtered out, representing older kubelets or nodes with the gate disabled;
  - with no declaring node available, the pod stays `Pending` with a `FailedScheduling` event naming the missing feature.

##### e2e tests

- **Volume exposure**: a container referencing `assigned.cpuset` and `assigned.memset` in a downward API volume sees the CPUs and the memory NUMA nodes it was assigned.
- **Volume follows a resize**: after a scale-down the volume file shows the newly allocated cpuset.
- **Environment variable exposure**: a container referencing the same values as environment variables sees them at startup, and they do not change after a resize.
- **No assignment**: a container with no exclusive CPUs and no assigned memory sees empty values.
- **Gate disabled on the node**: the container keeps running and sees empty values rather than failing.
- **Gate rollback and rollout**: with the gate disabled, creating a pod referencing these values is rejected while a pod already using them keeps running and can still be updated; after re-enabling, the volume files are filled in again.

### Graduation Criteria

#### Alpha

* Two new values, `assigned.cpuset` and `assigned.memset`, are accepted by kube-apiserver behind the `DownwardAPIAssignedResources` feature gate and rejected when it is disabled.
* The kubelet produces both values for downward API volume files and for container environment variables.
* With the gate disabled the kubelet produces an empty value rather than failing the container.
* The kubelet declares `DownwardAPIAssignedResources` in `node.status.declaredFeatures`, and the scheduler filters on it.
* Unit, integration, and e2e tests as described in the test plan.

#### Beta

* No unresolved critical bugs, and bugs reported by users have been addressed.
* Alternatives to the coupling between the Downward API and the resource managers have been evaluated, together with the DRA work tracked in [dra-driver-cpu#181](https://github.com/kubernetes-sigs/dra-driver-cpu/issues/181).

#### GA

* Allow time for feedback (6+ months).
* Make sure all risks have been addressed.

#### Deprecation

N/A

### Upgrade / Downgrade Strategy

**Upgrade.** No change is required of an existing cluster. Nothing exposes these values unless a pod asks for them. To use the feature, enable `DownwardAPIAssignedResources` on kube-apiserver and on the kubelets, and reference `assigned.cpuset` or `assigned.memset` from a downward API volume item or a container environment variable.

**Changing what a pod exposes.** Downward API volume items and container environment variables are part of the pod spec and cannot be changed on a running pod, so adding or removing these values means recreating the pod.

**No state is stored.** The values are computed from the CPU Manager's and the Memory Manager's current state every time a volume is written, so there is nothing to migrate, and nothing a newer kubelet could leave behind that an older one would have to read.

The two downgrade paths differ in whether the target version knows these values at all.

**The target version knows the values, with the feature gate disabled.** The values are preserved on existing pods and rejected on new ones, so a pod already using them keeps running and can still be updated. Its volume files become empty, while its environment variables keep whatever they were given when the container was created.

**The target version does not know the values.** kube-apiserver rejects pods that reference them: these are new values of an existing field, so validation checks them against a closed set of allowed names and fails with an unsupported container resource error. Operators should remove these references from pod specs before such a downgrade.

### Version Skew Strategy

This feature involves coordination between kube-apiserver (field validation), the kubelet (producing the values), and the scheduler (node filtering via Node Declared Features).

**New apiserver, older kubelet.** The apiserver accepts `assigned.cpuset` and `assigned.memset`. An older kubelet does not know these resource names at all, so it cannot produce a value for them and the downward API setup for such a container fails. Node Declared Features prevents this from being reached: such a kubelet does not declare `DownwardAPIAssignedResources`, so the scheduler does not place these pods on it.

**Old apiserver, newer kubelet.** Not a supported configuration, since the [version skew policy](https://kubernetes.io/releases/version-skew-policy/#kubelet) requires that the kubelet not be newer than kube-apiserver. Were it to occur anyway, the apiserver would reject the pod: these are new values of an existing field, and an apiserver that does not know them fails validation with an unsupported container resource error.

**Apiserver ON, kubelet OFF.** The pod is admitted, but the node does not declare the feature and the scheduler avoids it. If such a pod runs there anyway — a gate flip under a running pod, or a pod placed without the scheduler — the kubelet exposes an empty value instead of failing the pod. Unlike the older kubelet above, this one has the code and can degrade gracefully.

**Apiserver OFF, kubelet ON.** New pods using these values are rejected by the apiserver, so they never reach the kubelet. A pod that already uses them keeps them and is still served by the kubelet, since validation permits a value already in use. Static pods bypass the apiserver, so a static pod using these values is served regardless of the gate there.

**Both ON.** Full behavior: the apiserver validates the values, the kubelet declares the feature and produces the files and environment variables, and the scheduler places these pods only on nodes that declare it.

**Both OFF.** Feature disabled, existing behavior.

In clusters with mixed node versions, the Node Declared Features framework ([KEP-5328](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/5328-node-declared-features)) handles the skew on its own: only nodes declaring the feature receive pods that use these values. The operator therefore does not have to upgrade every kubelet before enabling the gate, and a pod that no node can serve stays unschedulable instead of failing on a node that cannot produce the values.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

This feature requires enabling the following feature gate

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `DownwardAPIAssignedResources`
  - Components depending on the feature gate: kube-apiserver, kubelet

The gate has a different job in each component. In kube-apiserver it decides whether a pod referencing `assigned.cpuset` or `assigned.memset` is accepted. In the kubelet it decides whether those values are produced and whether the node declares the feature. It has to be enabled on both for the feature to work; [Version Skew Strategy](#version-skew-strategy) describes what happens when only one of them has it.

###### Does enabling the feature change any default behavior?

No. Enabling the gate changes nothing on its own: the values are produced only for pods that reference `assigned.cpuset` or `assigned.memset`, and a pod that references neither behaves exactly as before.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, and no workload is disrupted by it.

**Disabling on kube-apiserver:** new pods referencing `assigned.cpuset` or `assigned.memset` are rejected, while pods that already reference them keep the reference and keep running, since validation permits a value already in use.

**Disabling on kubelet:** the kubelet writes an empty value into the volume files and leaves the environment variables as they were. Containers keep running; they only stop being told what they were assigned. The node also stops declaring the feature, so the scheduler will not place further pods needing it there.

###### What happens if we reenable the feature if it was previously rolled back?

**On kube-apiserver:** new pods referencing these values are accepted again.

**On the kubelet:** the volume files of pods already referencing them are filled in again at the next write. Environment variables are not, because they are evaluated when the container is created; a container has to be recreated to pick up a correct value there.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests exercise the feature gate switch itself: that the validation option follows the gate, and that a value already in use in the old spec stays permitted once the gate is off, so that disabling it does not break updates of running pods. On the kubelet side, unit tests cover that a container is given an empty value when the gate is off, rather than the downward API setup failing.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

**Highly available control plane.** During a rollout, the gate may be enabled on some apiservers and not others. Creating a pod that references these values then succeeds or fails depending on which apiserver serves the request. The failure is an explicit validation error rather than silent acceptance, and it disappears once the rollout completes. Updates of pods that already reference them are unaffected, because the validation option is derived from the old spec and does not depend on the gate state of the apiserver handling the request.

**Rolling the gate out across nodes.** A kubelet starts declaring `DownwardAPIAssignedResources` once the gate is enabled on it. Until enough nodes declare it, a pod referencing these values stays `Pending` with a scheduling event, rather than running somewhere that cannot serve it. That is a visible and recoverable state.

**Already running workloads are not affected.** A rollback does not kill pods. With the gate disabled, the kubelet writes an empty value into the volume file and leaves the environment variables as they were, so a container keeps running and only loses the information. Nothing about the CPU or memory assignment itself changes — this feature only reports it.

###### What specific metrics should inform a rollback?

No dedicated metrics are added in Alpha. The signals to watch are pods that stay `Pending` because no node declares the feature, and pod creation failures caused by an apiserver that still has the gate disabled. Both are visible without access to the nodes.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Local testing plan.

**Feature gate enable → disable → enable**

1. Cluster with the gate enabled on kube-apiserver and the kubelet. Create a pod with a downward API volume and an environment variable for `assigned.cpuset` and `assigned.memset`.
   - Verify the volume files and the environment variables carry the assigned CPUs and memory NUMA nodes.
   - Resize the pod down and verify the volume files follow the new assignment while the environment variables keep their original values.
2. Disable the gate on kube-apiserver and the kubelet and restart both.
   - Verify the pod keeps running and can still be updated, since the values are already in use.
   - Verify the volume files are now empty.
   - Verify that creating a new pod using these values is rejected.
3. Re-enable the gate on both and restart.
   - Verify the volume files carry the assigned values again.
   - Verify a newly created pod using these values is admitted.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

N/A

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

No metric is added in Alpha. The references live in the pod spec, under `spec.volumes[].downwardAPI.items[].resourceFieldRef.resource` and `spec.containers[].env[].valueFrom.resourceFieldRef.resource`, so an operator determines use by listing pods whose spec names `assigned.cpuset` or `assigned.memset`. Because the references are nested in arrays, this needs a JSON query rather than a field selector. The nodes able to serve them are those declaring `DownwardAPIAssignedResources` in `node.status.declaredFeatures`.

###### How can someone using this feature know that it is working for their instance?

- [X] Other (treat as last resort)
  - Details: For a single pod, this is visible from inside the container, without access to node logs or metrics. Read the value and compare it with the container's own cgroup files: a working setup gives the same set in `assigned.cpuset` as in `cpuset.cpus`, and in `assigned.memset` as in `cpuset.mems`. An empty value for a container that does hold exclusive CPUs or assigned memory means the node is not serving the feature — see [Troubleshooting](#troubleshooting).

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

These are the guarantees this KEP can make on its own. The window in which a workload can act on an upcoming change belongs to [KEP-6122](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/6122-configurable-scaling-delay), not here.

- A non-empty value matches the assignment the resource manager currently holds for that container.
- When an assignment changes, the volume file is updated within one kubelet sync period.
- While a change has been computed but not yet applied to the container, the volume file shows the new assignment rather than the old one.
- An environment variable reflects the assignment as of container creation and is not updated afterwards. This is a property of the mechanism, not a failure.
- This feature never delays a resize; it only reports.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Other (treat as last resort)
  - Details:
    - Time from a resource manager computing a new assignment to the corresponding volume file being written.
    - Number of volume files whose contents disagree with the assignment the resource manager holds, which should be zero.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Yes. A histogram of the delay between an assignment changing and the volume file being written, and a counter of failed writes, would let an operator check the SLOs above without inspecting individual pods. Neither is implemented in Alpha.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No new in-cluster or external services. The feature relies on the following, all in-tree:

- CPU Manager `static` policy (`--cpu-manager-policy=static`)
  - Usage description: `assigned.cpuset` has a value only for containers holding exclusive CPUs, which exist only under this policy.
    - Impact of its outage on the feature: under any other policy no container has exclusive CPUs, so `assigned.cpuset` is always empty.
    - Impact of its degraded performance or high-error rates on the feature: N/A, a kubelet configuration.
- Memory Manager `Static` policy (`--memory-manager-policy=Static`)
  - Usage description: `assigned.memset` has a value only for containers the Memory Manager has assigned memory to, which happens only under this policy. The value is read from the Memory Manager's state through `GetMemoryBlocks`; the Memory Manager itself is unchanged.
    - Impact of its outage on the feature: with `None`, no container has assigned memory, so `assigned.memset` is always empty.
    - Impact of its degraded performance or high-error rates on the feature: N/A, a kubelet configuration.
- Node Declared Features ([KEP-5328](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/5328-node-declared-features), GA since v1.37)
  - Usage description: the kubelet declares `DownwardAPIAssignedResources` whenever the feature gate is enabled, and the scheduler uses it to keep pods requesting these values off nodes that would not understand them.
    - Impact of its outage on the feature: a pod may be placed on a node that cannot serve the request — it then receives empty values, or, on a kubelet predating the feature, the downward API setup for the container fails. The same applies to pods placed without the scheduler, such as static pods.
    - Impact of its degraded performance or high-error rates on the feature: a stale node status could misroute pods for as long as the declared features are out of date, with the same bounded consequence.

Neither resource manager policy gates the declaration: a node declares the feature whenever the feature gate is enabled. Nothing is lost by that, because a node not running the static policies has no exclusive assignments to report in the first place, so an empty value is the accurate answer rather than a degraded one. The declaration states that the node understands these fields, not that it currently has assignments to report.

No new container runtime capability is required: the kubelet writes the values into the Downward API volume and the container environment itself, without involving CRI.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. The kubelet reports its declared features as part of the node status it already sends, and the values themselves are written locally into the pod's volume, without involving the API server.

###### Will enabling / using this feature result in introducing new API types?

No new API types, and no new field. Two new values, `assigned.cpuset` and `assigned.memset`, become valid for the existing `ResourceFieldRef.Resource` field; they are listed in [Resource Field Extensions](#resource-field-extensions).

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Only for pods that use the feature, and no differently from any other downward API reference: such a pod carries one volume item or one environment variable entry naming the new value, which costs the same as naming `limits.cpu` does today. No new objects are created.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. Producing a value is an in-memory lookup in the CPU Manager or the Memory Manager, and it is written into a volume the pod already mounts, so nothing is added to the pod startup path beyond what any downward API item costs.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. One small file per reference, in a volume the pod already mounts.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. The number of files is bounded by the number of containers referencing these values, exactly as for any other downward API item.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The values come from the kubelet's own CPU Manager and Memory Manager state, not from the control plane, so the downward API volume files of a running pod keep being updated while the apiserver is unreachable. Creating a pod that uses these values requires the apiserver, as any pod creation does.

###### What are other known failure modes?

- A container is given an empty value although the node is expected to support the feature
  - Detection: the volume file or the environment variable is empty while the container does have exclusive CPUs or assigned memory.
  - Mitigations: enable the gate on that node, or move the pod to a node that declares the feature.
  - Diagnostics: kubelet logs, and `node.status.declaredFeatures` on the node in question.
  - Testing: the unit and e2e tests for the gate being disabled.
- A pod using these values stays `Pending`
  - Detection: the pod has no node assigned and the scheduler reports that no node satisfies its required features.
  - Mitigations: enable the gate on at least one node, or remove the fields from the pod.
  - Diagnostics: scheduling events from `kubectl describe pod`, and the declared features of the candidate nodes.
  - Testing: the scheduler filtering integration test.
- An environment variable is stale after a resize
  - Detection: the environment variable disagrees with the corresponding volume file.
  - Mitigations: none. This is by design; workloads that need the value to follow a resize must read the volume file.
  - Diagnostics: compare the environment variable with the volume file for the same resource.
  - Testing: the e2e case verifying that volume files follow a resize while environment variables keep their original values.

###### What steps should be taken if SLOs are not being met to determine the problem?

The SLOs concern the accuracy and the freshness of the exposed values.

If a value disagrees with the container's actual assignment, first check whether it is an environment variable. Those are set when the container is created and are expected to be stale after a resize, which is not a violation. For a volume file, compare it with the container's `cpuset.cpus` and `cpuset.mems`, and look in the kubelet log for failures writing the volume.

If a value is empty where an assignment does exist, the node is not serving the feature: check that the gate is enabled there, that the node declares `DownwardAPIAssignedResources`, and that the relevant resource manager policy is not `None`.

If a volume file lags an assignment change by much more than a kubelet sync period, check the kubelet's sync configuration first, then whether the pod is being synced at all.

## Implementation History

- 2026-09-15: KEP created by splitting the Downward API exposure out of [KEP-6122](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/6122-configurable-scaling-delay) ([#6370](https://github.com/kubernetes/enhancements/pull/6370))
- 2026-09-17: Memory exposure brought into the Alpha scope, with `assigned.memset` defined as the set of assigned memory NUMA nodes

## Drawbacks

N/A

## Alternatives

### 1. Read the cgroup files from inside the container

* **Description**: The container can discover its assigned CPUs and memory NUMA nodes through cgroup-version-agnostic interfaces such as `sched_getaffinity(2)` or `/proc/self/status` (`Cpus_allowed_list`, `Mems_allowed_list`), or by reading `cpuset.cpus` and `cpuset.mems` through cgroupfs directly.
* **Why Rejected**: All these mechanisms show only the set that is currently applied, never the one that is about to be applied, so they cannot serve the advance-notice use case that motivates this KEP together with KEP-6122. While `sched_getaffinity` and `/proc/self/status` are cgroup-version-agnostic, they use separate interfaces for CPU and memory, and are an implementation detail rather than a contract. Direct cgroupfs access has the additional drawback of differing paths between cgroup v1 and v2, dependence on cgroup namespace usage, and no guarantee across runtimes. Every workload would carry its own fragile parser for something Kubernetes can state plainly.

### 2. Query the kubelet pod resources endpoint

* **Description**: The kubelet already reports concrete CPU and memory assignments through the gRPC service at `/var/lib/kubelet/pod-resources/kubelet.sock` ([KEP-2043](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2043-pod-resource-concrete-assigments)).
* **Why Rejected**: That endpoint is designed for node monitoring agents, not for workloads. Using it from an ordinary container means mounting a host socket into the pod, which grants visibility into every pod on the node — a privilege escalation that is hard to justify for a workload that only needs to know about itself. The Downward API exists precisely to give a pod information about itself without such access.

### 3. Expose the assignments in the pod status

* **Description**: Report the assigned CPUs and memory NUMA nodes in the pod status, for example as `pod.status.resourceAssignments`, and let the workload watch its own pod. Suggested during the review of KEP-6122 as a way to decouple the exposure from the CPU Manager implementation.
* **Why Rejected for Alpha**: This requires API credentials and RBAC inside the workload, plus a watch per pod on the apiserver, to deliver information the node already has locally. It also makes the value's freshness depend on the control plane being reachable, whereas a Downward API volume is written from local state. It remains the strongest candidate should the coupling discussed in [Risks and Mitigations](#risks-and-mitigations) need to be undone, since it would cover CPU, memory, and future resource types through one field.

### 4. Volume files only, without environment variables

* **Description**: Expose the values only as Downward API volume files, since an environment variable cannot be updated after the container has started.
* **Why Rejected**: A workload that needs the value only at startup, to pin its threads once, is better served by an environment variable than by mounting a volume. Offering the values in one form but not the other would also make `assigned.cpuset` and `assigned.memset` behave unlike every other `ResourceFieldRef` value, which is available in both. The staleness is real and is recorded as a known failure mode in [Troubleshooting](#troubleshooting).

### 5. Include the amount of memory in `assigned.memset`

* **Description**: Report both the memory NUMA nodes and the assigned memory size, for example `memory:2097152000,NUMA:[0-1]`.
* **Why Rejected**: The size is already available through `limits.memory` in the same Downward API, so it would be redundant. It would also force a composite encoding into a field whose every other value is a plain list or quantity, and make `assigned.memset` structurally unlike `assigned.cpuset` for no gain.

### 6. Expose the assignments through DRA

* **Description**: Have the Dynamic Resource Allocation framework provide the abstraction instead, for example by publishing the assignment through CDI. Suggested during the review of KEP-6122.
* **Why Deferred**: The DRA CPU driver and the kubelet's static CPU policy cannot run on the same node today, so DRA cannot serve the workloads this KEP targets. Exposing DRA allocations the same way is tracked in [dra-driver-cpu#181](https://github.com/kubernetes-sigs/dra-driver-cpu/issues/181), and the intent is for the driver to use the same file path, which makes this a future extension of the contract rather than a competing design.

### 7. Inject the assignments as files, the way DRA device attributes are

* **Description**: Rather than letting a pod request these values through `ResourceFieldRef`, have the component that knows them inject them. [KEP-5304](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/5304-dra-attributes-downward-api) does this for DRA device attributes: the driver's metadata is bind-mounted into the container through a CDI spec and appears at a well-known path. Applied here, the CPU Manager and the Memory Manager would inject their assignments the same way, with nothing in the pod spec asking for them. Suggested in [kubernetes/kubernetes#136015](https://github.com/kubernetes/kubernetes/pull/136015#issuecomment-5683234122), on the grounds that these values are not part of the Pod API and therefore sit oddly in an API meant to project pod fields downward.

* **Why Rejected for Alpha**: Its main attraction is that it would remove this KEP's API change altogether, but three things do not carry over. KEP-5304's path and CDI spec are keyed by a claim and a request, and exclusive CPUs managed by the kubelet have neither, so the convention would have to be reinvented along with a way to emit container edits without a DRA driver. Nothing in the pod spec would reference the assignment, so there would be no anchor for the scheduler to filter on and no way for a workload to state that it needs the value — KEP-5304 has the ResourceClaim for that, and this KEP has no equivalent. And a file is then the only possible form, so the environment variable in [Use Cases](#use-cases) would be lost. This remains the most direct answer to the coupling concern in [Risks and Mitigations](#risks-and-mitigations) and a candidate for revisiting.

## Infrastructure Needed

N/A
