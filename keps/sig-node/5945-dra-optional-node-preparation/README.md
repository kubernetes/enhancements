# KEP-5945: DRA Optional Node Operations

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Deploying controller-managed resources without node-local drivers](#deploying-controller-managed-resources-without-node-local-drivers)
    - [Skipping only cleanup for devices that self-release](#skipping-only-cleanup-for-devices-that-self-release)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
    - [API Server Handling](#api-server-handling)
  - [Allocator Changes](#allocator-changes)
  - [Kubelet Changes](#kubelet-changes)
  - [Node Declared Features Integration](#node-declared-features-integration)
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
  - [Alternative 1: DeviceClass-level configuration](#alternative-1-deviceclass-level-configuration)
  - [Alternative 2: Claim-level declaration](#alternative-2-claim-level-declaration)
  - [Alternative 3: Kubelet Auto-Discovery / gRPC probe with timeout](#alternative-3-kubelet-auto-discovery--grpc-probe-with-timeout)
  - [Alternative 4: Centralized catch-all no-op plugin](#alternative-4-centralized-catch-all-no-op-plugin)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website
[Conformance Tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md
[all GA Endpoints]: https://github.com/kubernetes/community/pull/1806

## Summary

This KEP introduces **Optional Node Operations** to Dynamic Resource Allocation
(DRA), allowing resource drivers to declare that node preparation and/or node
unpreparation is not required for their devices. Currently, the kubelet assumes
it must always coordinate with a node-local DRA driver via gRPC to prepare
allocated devices before container start (`NodePrepareResources`), and to
unprepare them during pod termination (`NodeUnprepareResources`).

In some cases, node preparation or cleanup is a pure no-op. Requiring it forces
administrators and vendors to deploy and maintain empty node-local drivers on
every node, which introduces unnecessary operational complexity and risk.

By introducing a `SkipNodeOperations` field to the
`ResourceSliceSpec` and propagating it to the device allocation results at
scheduling time, the kubelet can safely skip driver lookup and gRPC calls for
devices that do not require these node-local actions.

## Motivation

In Dynamic Resource Allocation (DRA), the kubelet coordinates with a node-local
driver via gRPC to prepare allocated devices before container start
(`NodePrepareResources`) and to unprepare them upon pod termination
(`NodeUnprepareResources`). For node-local accelerators (such as PCIe GPUs or
local FPGAs), this node-level setup is critical to check device health,
partition memory, and configure mount paths.

However, there is an emerging class of resources whose lifecycles are managed
entirely in the control plane and published centrally by a controller as
`ResourceSlice` objects. These resources require absolutely zero node-local
setup. Under the current architecture, the kubelet still assumes a node-local
driver exists, forcing administrators to deploy and maintain wasteful "no-op"
node DaemonSets just to answer gRPC calls with empty success responses. If one
of these dummy helper plugins crashes or is missing, the kubelet's unprepare
hook fails and retries indefinitely, leaving terminating pods permanently "stuck
in Terminating" and blocking cluster upgrades and node drains.

To resolve this architectural mismatch and accommodate modern deployments, we
need a way for resource drivers to declare that node preparation and cleanup
can be skipped. Bypassing these gRPC hooks directly at the `ResourceSlice` level
allows vendors to deploy central-only controllers with zero worker node
footprints. It also provides the flexibility to support mixed hardware
topologies—where a single driver manages some devices requiring node-level
preparation and others that do not—without splitting the driver or forcing
unnecessary footprints onto worker nodes.

### Goals

- Allow resource drivers to declare that node-local operations (preparation and
  clean-up) are optional for devices.
- Propagate this configuration from the `ResourceSlice` to the final allocated
  `ResourceClaim.Status.Allocation` result.
- Update the kubelet to skip driver lookup and gRPC preparation/unpreparation
  steps when node operations are explicitly configured as skipped.
- Maintain backward compatibility: by default, all existing DRA drivers must
  continue to require node-local preparation and unpreparation.

### Non-Goals

- Eliminate node-local preparation entirely.
- Enable users to override this infrastructure requirement at the individual
  `ResourceClaimSpec` level.

## Proposal

We propose adding a `SkipNodeOperations` field to
`ResourceSliceSpec` and `DeviceRequestAllocationResult`.

1. **API Definition**: The driver/controller publisher sets `skipNodeOperations`
   in `ResourceSlice` resources if the published devices do not require
   node-local setup or cleanup.
2. **Control Plane Resolution**: The allocator/scheduler resolves the referenced
   `ResourceSlice` during allocation, and copies this configuration into
   `ResourceClaim.Status.Allocation.Devices.Results[i].SkipNodeOperations`.
3. **Node Execution**: The kubelet reads this field from the `ResourceClaim`'s
   allocation results. If all allocated devices for a given driver within a
   claim skip an operation, the kubelet bypasses the corresponding
   gRPC call (`NodePrepareResources` or `NodeUnprepareResources`) to the
   node-local resource driver.

### User Stories

#### Deploying controller-managed resources without node-local drivers
As a cluster administrator or vendor using a central driver controller, I want
to offer resources (e.g., cluster-wide shared resource pools, logically
partitioned network services, or pure control-plane gating drivers like
[dra-driver-image-configurator](https://github.com/gke-labs/dra-drivers/tree/main/dra-driver-image-configurator))
where availability is discovered and published as
`ResourceSlice` resources centrally by the controller. Because the devices
require no node-local plumbing or mount operations on worker nodes, there is no
node driver deployed. The controller publishes these resources with
`skipNodeOperations: ["*"]`. When users request these
devices, the kubelet launches the pods immediately and cleanly, without
complaining about missing node-local drivers, and without requiring any node
driver DaemonSet to be present in the cluster.

#### Skipping only cleanup for devices that self-release

As a driver vendor whose devices need node-local setup at pod start but whose
cleanup happens automatically (for example, because the resource is released by
the control plane when the claim is deallocated), I want to publish
`skipNodeOperations: ["NodeUnprepareResources"]`. The kubelet still calls
`NodePrepareResources`, so the device is set up correctly, but pod termination
never blocks on an unprepare call.

### Risks and Mitigations

- **Dynamic ResourceSlice Changes**: An administrator or controller could update
  `SkipNodeOperations` in a `ResourceSlice` while claims are
  already allocated.
  - *Mitigation*: While freezing the allocation configuration into the
    `ResourceClaim` status ensures consistent execution for already running pods,
    it also means that if a driver's requirements are updated in-place, existing
    claims will still use the older configuration. Specifically:
    - If a driver changes from skipping to requiring node preparation, existing
      claims will still have node preparation skipped by the kubelet, causing pods
      to run without the required hardware setup.
    - If a driver changes from requiring to skipping node preparation, and the
      node-local driver is decommissioned, existing claims will still require
      node preparation, causing the kubelet to fail or hang waiting for the
      missing driver plugin.

    Because this skew is inherent to the decoupled nature of scheduling and runtime,
    this risk must be managed operationally: cluster administrators must perform driver
    upgrades and migrations carefully, ensuring no active claims/pods exist for the
    driver before changing its configuration or decommissioning node-local driver components.
- **Backward Compatibility & Out-of-Tree / Custom Allocators**: Old scheduler
  clients or out-of-tree custom driver controllers/allocators might write
  allocation results without setting the new `SkipNodeOperations` field.
  - *Mitigation*: The behavior depends on whether the driver uses optional node operations:
    1. **For drivers that do not use optional node operations** (i.e., require node-local setup):
       The fields default to omitted (not skipped).
       The kubelet will execute node preparation and clean-up as normal. This guarantees 100% backward
       compatibility with all existing schedulers, custom controllers, and running workloads.
    2. **For drivers that use optional node operations** (and do not deploy a node-local driver):
       If an old or out-of-tree allocator fails to copy the field from the `ResourceSlice` to the
       `ResourceClaim` status, the kubelet will default to executing preparation and fail because no
       node-local driver is running. To mitigate this:
       - Custom allocators/schedulers must be upgraded to support and copy the new field before they can
         be used with optional-operations drivers.
       - Alternatively, during transitions, operators can deploy a minimal, "no-op" node-local daemon for
         the driver to satisfy the kubelet's gRPC calls until the allocator is upgraded.
- **Unknown operation names**: Future versions may add new values to the
  `SkipNodeOperation` enum.
  - *Mitigation*: The kubelet ignores values it does not recognize, so an older
    kubelet reading a newer slice never skips an operation it does not
    understand.

## Design Details
### API Changes

1. **`ResourceSliceSpec`**:
   ```go
   type ResourceSliceSpec struct {
       ...
       // SkipNodeOperations lists node-local resource operations (gRPC calls)
       // that will be skipped for the devices in this slice when determining whether
       // operations are necessary on the node. If all allocated devices for a driver in
       // a claim skip an operation, that gRPC call will be skipped. Valid values are:
       //
       // - "NodePrepareResources": NodePrepareResources gRPC calls are skipped. This
       //   value cannot be specified unless "NodeUnprepareResources" is also listed
       //   (or "*" is specified).
       // - "NodeUnprepareResources": NodeUnprepareResources gRPC calls are skipped.
       // - "*": All node-local resource operations are skipped.
       //
       // Other values may be added in the future. The kubelet must ignore unknown
       // values.
       //
       // +optional
       // +listType=set
       // +featureGate=DRAOptionalNodeOperations
       SkipNodeOperations []SkipNodeOperation `json:"skipNodeOperations,omitempty" protobuf:"bytes,10,rep,name=skipNodeOperations,casttype=SkipNodeOperation"`
   }

   // +enum
   type SkipNodeOperation string

   const (
       SkipNodeOperationNodePrepareResources   SkipNodeOperation = "NodePrepareResources"
       SkipNodeOperationNodeUnprepareResources SkipNodeOperation = "NodeUnprepareResources"
       SkipNodeOperationAll                    SkipNodeOperation = "*"
   )
   ```

2. **`DeviceRequestAllocationResult`**:
   ```go
   type DeviceRequestAllocationResult struct {
       ...
       // SkipNodeOperations lists node-local resource operations (gRPC calls)
       // that will be skipped for this allocated device when determining whether
       // operations are necessary on the node. If all allocated devices for a driver in
       // a claim skip an operation, that gRPC call will be skipped. It is a copy of
       // the ResourceSlice.spec.skipNodeOperations value at the time when the device was allocated.
       //
       // +optional
       // +listType=set
       // +featureGate=DRAOptionalNodeOperations
       SkipNodeOperations []SkipNodeOperation `json:"skipNodeOperations,omitempty" protobuf:"bytes,11,rep,name=skipNodeOperations,casttype=SkipNodeOperation"`
   }
   ```

#### API Server Handling

Feature gate enforcement uses the standard **drop-disabled-fields** pattern:

* **When the `DRAOptionalNodeOperations` feature gate is disabled**:
  * **New Resources**: `spec.skipNodeOperations` on a `ResourceSlice`, and
    `status.allocation.devices.results[*].skipNodeOperations` on a
    `ResourceClaim`, are silently **dropped** (set to `nil`) on write.
  * **Existing Resources**: if the old object already has the field populated,
    it is **preserved** across updates. This means objects written while the
    gate was enabled survive a downgrade and continue to work when the gate is
    re-enabled, and unrelated updates to those objects do not fail.
* **When the feature gate is enabled**:
  * The field is validated and persisted normally.

Validation that applies regardless of the gate:

* Values must be members of the `SkipNodeOperation` enum, and the list is a set
  (`+listType=set`), so duplicates are rejected. Both are handled by declarative
  validation.
* `"NodePrepareResources"` may only be listed if `"NodeUnprepareResources"` or
  `"*"` is also listed. Otherwise a pod could be admitted and started on a node
  with no node-local driver, since preparation is skipped, and then hang in
  `Terminating` because unpreparation is still required and no driver can serve
  it.

### Allocator Changes

During scheduling, the structured parameters allocator resolves `ResourceSlices`
that contain the allocated devices. When a device's slice has a non-empty
`SkipNodeOperations`, the allocator treats the device as **unallocatable** in
either of these cases:

- The `DRAOptionalNodeOperations` feature gate is disabled in the
  scheduler/allocator.
- The candidate node does not advertise `DRAOptionalNodeOperations` in
  `node.status.declaredFeatures`.

Filtering the device out (rather than failing the whole claim) lets the
allocator fall back to another device or another node, and leaves the pod
`Pending` with a normal unschedulable status if no candidate remains.

Otherwise the allocator copies the `SkipNodeOperations` set verbatim from the
`ResourceSliceSpec` into each `DeviceRequestAllocationResult` under
`ResourceClaim.Status.Allocation.Devices.Results`.

### Kubelet Changes

When Kubelet prepares resources for an allocated claim, it evaluates the
allocated devices' status:
1. **Aggregation**: Because Kubelet invokes preparation and clean-up per-claim
   and per-driver, it computes, for each driver in the claim, the set of
   operations that *every* device allocated from that driver skips. A device
   listing `"*"` counts as skipping every operation.
2. **Checkpointing**: Kubelet caches this aggregated set inside its
   checkpointed, claim-specific state (`ClaimInfo`) so it is safely preserved
   across Kubelet restarts. To ensure robust upgrade/downgrade compatibility, the
   checkpoint serialization must be forward and backward compatible.
3. **Bypassing**: During `PrepareResources`, the DRA manager skips both the
   driver registry lookup and the `NodePrepareResources` gRPC call for any
   driver whose cached set contains `NodePrepareResources` or `"*"`. During
   `UnprepareResources` it does the same for `NodeUnprepareResources` or `"*"`.
   Skipping the registry lookup is what allows a driver with no node-local
   component to work at all. Each skipped call increments
   `dra_node_prepare_skips_total` or `dra_node_unprepare_skips_total` for that
   driver.
4. **Disabled Feature Gate Behavior**: If the `DRAOptionalNodeOperations`
   feature gate is disabled on the kubelet, computing the driver state for a
   claim whose allocation result asks to skip anything returns an error
   (`DRAOptionalNodeOperations feature gate is disabled on kubelet`). This
   surfaces as a `PrepareResources` failure, so the pod stays in
   `ContainerCreating` with a `FailedPrepareDynamicResources` event rather than
   running with an unprepared device. Because the check lives in the shared
   `ClaimInfo` construction path, it applies uniformly rather than relying on a
   separate admission check.

   Note that pods which were already prepared keep their checkpointed state and
   are unprepared from that state, so disabling the gate does not strand a
   running pod in `Terminating`.

### Node Declared Features Integration

To manage version skew safely during rolling upgrades, this KEP integrates with
the **Node Declared Features** framework
This allows the control plane to dynamically discover if a node's Kubelet
supports optional node operations before scheduling workloads, preventing pods
from being scheduled to incompatible nodes.

We register a new declared feature:
*   **Feature Name**: `DRAOptionalNodeOperations`
*   **Associated Feature Gate**: `DRAOptionalNodeOperations`
*   **Discovery Logic (Kubelet)**: A node declares support for
    `DRAOptionalNodeOperations` in its `node.status.declaredFeatures` if and
    only if the `DRAOptionalNodeOperations` feature gate is enabled on the
    Kubelet.
*   **Enforcement (Scheduler)**: The generic NDF `InferForScheduling` hook
    returns `false` for this feature, because a claim's `status.allocation` is
    not set until PreBind and therefore cannot be inspected at PreFilter time.
    Enforcement instead lives in the `dynamicresources` scheduler plugin, in two
    places:
    1.  **Allocating a new claim**: the allocator will not select a device whose
        slice sets `SkipNodeOperations` unless the candidate node declares
        `DRAOptionalNodeOperations`.
    2.  **Filtering with an already-allocated claim**: if any allocated device
        in the claim sets `SkipNodeOperations` and the node does not declare
        the feature, the node is rejected.
*   **Max Version**: `nil`. The feature remains a scheduling constraint; it can
    be given a max version once the feature is GA and the supported kubelet
    skew guarantees support.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

- **API Server Validation**: In
  `pkg/apis/resource/validation/validation_resourceslice_test.go` and
  `pkg/registry/resource/`:
  - Verify valid combinations of `skipNodeOperations` are accepted and invalid
    values (or `"NodePrepareResources"` alone) are rejected.
  - Verify field is dropped when feature gate is disabled, and preserved if
    already in use.
- **Allocator Unit Tests**: In
  `staging/src/k8s.io/dynamic-resource-allocation/structured/`:
  - Verify that `SkipNodeOperations` in `ResourceSliceSpec`
    is correctly propagated to `AllocationResult`.
  - Verify that devices with `SkipNodeOperations` are not selected when the
    feature is disabled or the node lacks support.
- **Kubelet DRA Manager Unit Tests**: In `pkg/kubelet/cm/dra/manager_test.go`:
  - Assert that `prepareResources` and `unprepareResources` bypass gRPC calls
    when the driver's aggregated set covers the operation.
  - Assert that `prepareResources` fails when the feature gate is disabled on
    the node and skipping is requested.
- **Shared Library Unit Tests**: In
  `staging/src/k8s.io/component-helpers/nodedeclaredfeatures/features/draoptionalnodeoperations/feature_test.go`:
  - Verify the node declared feature for `DRAOptionalNodeOperations` behaves
    correctly.
- **Kubelet Checkpoint State Unit Tests**: In
  `pkg/kubelet/cm/dra/claiminfo_test.go`:
  - Verify backward and forward compatibility of the serialized `ClaimInfo`
    checkpoint state:
    - **Forward Compatibility (Downgrade/Rollback)**: Verify that a checkpoint
      file written by a Kubelet running version N (containing the new
      `SkipNodeOperations` field in `ClaimInfo`) can be
      successfully parsed and deserialized by a Kubelet running version N-1 (or
      with the feature gate disabled) without parsing errors or crashes, with
      unrecognized fields being safely ignored.
    - **Backward Compatibility (Upgrade)**: Verify that an older checkpoint file
      written by a Kubelet running version N-1 (which completely lacks the new
      field) is successfully parsed and deserialized by Kubelet version N, with
      the field defaulting to not skipping (ensuring we do not skip
      preparation/unpreparation for legacy claims).

##### Integration tests

- **Scheduler Filtering Integration Tests**: In
  `test/integration/dra/optional_node_operations.go`:
  - Verify that a pod or pod group requiring `DRAOptionalNodeOperations` (having a claim
    allocated with `SkipNodeOperations`) is successfully scheduled to a
    node that advertises the feature.
  - Verify that the scheduler filters out (rejects) nodes that do not advertise
    the feature (representing older Kubelets or nodes with the feature gate
    disabled).
  - Verify that if no compatible nodes are available, the pod remains in the
    `Pending` state with a `FailedScheduling` event indicating the missing
    `DRAOptionalNodeOperations` feature on nodes.

##### e2e tests

Basic End-to-End test cases (Scenario 1 & 2) were implemented in Alpha inside `test/e2e/dra/dra.go` to validate
`skipNodeOperations` configurations using different
driver configurations. Scenario 3 (Upgrade / Downgrade and Feature Gate Rollback)
will be added for Beta.

###### Scenario 1: Driver without node-local components (Pure Control-Plane)
This scenario validates that we can run workloads using drivers that do not
deploy any node-local components.

- **Setup**: Deploy a DRA test driver without node gRPC components running on
  worker nodes (`WithKubelet = false`).
- Test Case 1.1: Fully skipped node operations (`skipNodeOperations: ["*"]`)
  - **API Configuration**: Publish `ResourceSlices` with `skipNodeOperations: ["*"]`.
  - **Workload**: Deploy a Pod referencing this resource.
  - **Assertions**:
    - The Pod reaches the `Running` phase successfully.
    - The allocated device can be accessed.
    - No `FailedPrepareDynamicResources` warnings are posted to the Pod events.
    - Pod deletion completes cleanly and immediately (does not hang in
      `Terminating` waiting for unprepare).
- Test Case 1.2: Missing node component failure (`skipNodeOperations` omitted)
  - **API Configuration**: Publish `ResourceSlices` with `skipNodeOperations` omitted.
  - **Workload**: Deploy a Pod referencing this resource.
  - **Assertions**:
    - The Pod gets stuck in `ContainerCreating`
      with `FailedPrepareDynamicResources` errors because the kubelet tries to
      contact the non-existent node driver.

###### Scenario 2: Driver with node-local components (Standard Driver)
This scenario validates that the kubelet invokes the node-local
driver when `skipNodeOperations` is omitted.

- **Setup**: Deploy a standard DRA test driver that includes node-local gRPC
  components.
- Test Case 2.1: Standard Node Execution (`skipNodeOperations` omitted)
  - **API Configuration**: Publish `ResourceSlices` with `skipNodeOperations` omitted.
  - **Workload**: Deploy a Pod.
  - **Assertions**:
    - The Pod reaches the `Running` phase.
    - Assert that the driver's `NodePrepareResources` **was** called.
    - Delete the Pod.
    - Assert that the driver's `NodeUnprepareResources` **was** called.

###### Scenario 3: Upgrade / Downgrade and Feature Gate Rollback
This scenario validates that the system behaves correctly during a rolling
upgrade or downgrade/rollback of the feature gate.

- **Test Case 3.1: Rolling Upgrade (N-1 to N)**:
  - **Setup**: Start with a cluster running version N-1 (feature gate disabled).
    Deploy a DRA driver.
  - **Action 1 (Control Plane Upgrade)**: Upgrade the control plane to version N
    (feature gate enabled).
    - **Assertions**:
      - If we deploy a new workload using a control-plane-only driver (no
        node-local components):
        - The Pod remains in the `Pending` state (unschedulable). The scheduler's
          `dynamicresources` plugin must filter out all N-1 worker nodes because
          they do not advertise `DRAOptionalNodeOperations` in
          `node.status.declaredFeatures`.
        - The pod must not be scheduled to any N-1 node.
  - **Action 2 (Kubelet Upgrade)**: Upgrade the kubelets to version N.
    - **Assertions**:
      - Once a Kubelet is upgraded to N and advertises `DRAOptionalNodeOperations`,
        verify that the pending workload is **automatically scheduled** to that node,
        successfully bypasses node preparation, transitions to `Running`, and runs successfully.
      - Verify that deleting the control-plane-only workload completes
        immediately without trying to contact a node-local driver.
- **Test Case 3.2: Feature Gate Rollback / Downgrade (N to N-1)**:
  - **Setup**: Start with a cluster running version N (feature gate enabled).
    Deploy a standard DRA driver and a workload using `skipNodeOperations: ["*"]`.
  - **Action 1 (Control Plane Downgrade)**: Downgrade the control plane to N-1
    (feature gate disabled).
    - **Assertions**:
      - The API server allows the existing `ResourceSlice` objects to remain
        valid and not be rejected on unrelated updates.
      - The running workload on the N kubelet continues to run without
        interruption.
  - **Action 2 (Kubelet Downgrade / Feature Gate Rollback)**: Downgrade the
    Kubelet binary to N-1, or disable the `DRAOptionalNodeOperations` feature
    gate on Kubelet version N, and restart the Kubelet.
    - **Assertions**:
      - **Checkpoint Recovery**: The Kubelet starts up successfully and parses
        the checkpoint file without errors or crashes.
        - *For Kubelet version N (gate disabled)*: The Kubelet successfully
          recovers the full `ClaimInfo` state including the saved
          `skipNodeOperations` set, and the already-prepared pod is unprepared
          from that state.
        - *For Kubelet version N-1*: The Kubelet successfully parses the
          checkpoint by ignoring the unknown field, and recovers the rest of
          the state with nothing skipped.
      - **Workload Deletion**:
        - Verify that the kubelet behaves according to the
          [Disabled Feature Gate Behavior](#kubelet-changes) section
          upon workload deletion.
- **Test Case 3.3: Upgrade -> Downgrade -> Upgrade (N-1 -> N -> N-1 -> N)**:
  - **Setup**: Start with a cluster running version N-1 (feature gate disabled).
    Deploy a standard DRA driver.
  - **Action 1 (Upgrade)**: Upgrade the cluster to version N (feature gate
    enabled).
    - Deploy a workload using a driver configured with `skipNodeOperations: ["*"]`.
    - Assert that the workload runs successfully.
  - **Action 2 (Downgrade)**: Downgrade the cluster to version N-1 (feature gate
    disabled).
    - **Assertions**:
      - Assert that the API server allows the existing
        `ResourceSlice` (which has `skipNodeOperations: ["*"]`) to remain valid and unmodified.
      - Delete the workload and verify that the kubelet behaves according to the
        [Disabled Feature Gate Behavior](#kubelet-changes) section.
  - **Action 3 (Upgrade Again)**: Upgrade the cluster back to version N (feature
    gate enabled).
    - Deploy a new workload using the same driver.
    - Assert that the new field is respected, and the workload runs
      successfully.
    - Assert that any pre-existing resource slices that survived the downgrade
      cycle continue to function correctly with the re-enabled feature gate.

### Graduation Criteria

#### Alpha

- Feature implemented behind the `DRAOptionalNodeOperations` feature flag (off
  by default).
- Full unit and basic E2E test suites (Scenario 1 & 2) implemented and green.

#### Beta

- Enable the feature gate by default.
- E2E upgrade/downgrade and rollback test suites (Scenario 3) implemented and green.
- Gather real-world feedback from developers and vendors deploying
  controller-managed DRA drivers.
- Ensure no regressions or performance issues are observed in large clusters.

#### GA
- Feature gate locked to true.

### Upgrade / Downgrade Strategy

- **Upgrade**:
  - When the cluster control plane and nodes are upgraded, all preexisting
    claims (where the new field is absent) automatically evaluate
    to not skipped. This guarantees no change in behavior for running
    workloads.
  - Newer claims can utilize drivers that publish resource slices configured
    with `skipNodeOperations` to bypass
    node-local execution.
  - During rolling upgrades, the scheduler will
    automatically restrict the scheduling of pods using these newer "no-prep" claims
    to upgraded nodes that advertise support for `DRAOptionalNodeOperations`.
- **Downgrade**:
  - If a cluster is downgraded to a version where `DRAOptionalNodeOperations`
    is disabled/unavailable, the kubelet will ignore the skip field and default
    to the legacy behavior of expecting node preparation.
  - If any pods are running using a driver without node-local drivers, those
    pods will fail to restart or delete cleanly if the kubelet tries to invoke
    node-local gRPC calls that don't exist. Operators must ensure all pods using
    no-prep claims are terminated before downgrading, or ensure temporary no-op
    drivers are running during downgrade transitions.

### Version Skew Strategy

- **Older kubelet (N-1 and older) / Upgraded Control Plane (N)**:
  - **Automated Version Skew Protection**: If the control plane is upgraded and
    generates allocations with `skipNodeOperations` set, the `dynamicresources`
    scheduler plugin requires the target node to declare the
    `DRAOptionalNodeOperations` feature.
  - Because older worker nodes running older Kubelets ($N-1$ and older) do not
    support the feature gate, they will not advertise
    `DRAOptionalNodeOperations` in their `node.status.declaredFeatures`.
  - The scheduler will **automatically filter out these older nodes** during the
    scheduling cycle, guaranteeing that the pod will only land on compatible,
    upgraded nodes.
- **Upgraded kubelet (N) / Older Control Plane (N-1 and older)**:
  - If the control plane has not been upgraded yet, any new allocations will not
    have `skipNodeOperations` set in the status.
  - An upgraded kubelet (N) will read the absent fields and default to not skipping
    (requiring node preparation/unpreparation).
  - The behavior depends on whether the driver uses optional node operations:
    - **For drivers that do not use optional node operations** (i.e., require node-local setup):
      The fallback ensures backward-compatible, safe execution because the
      node-local driver is running and kubelet will coordinate with it as normal.
    - **For drivers that use optional node operations** (and do not deploy a node-local driver):
      The fallback means the upgraded kubelet will attempt to coordinate
      with the local driver and fail because no node-local driver is running.
      - *Mitigation*: The control plane must be upgraded before these optional-operations
        drivers can be deployed, or a temporary, minimal "no-op" node-local daemon must
        be deployed to satisfy the kubelet's gRPC calls during the transition window.

    *Note*: This same fallback behavior occurs if the control plane is upgraded (N)
    but the active custom allocator or scheduler has not been upgraded to support
    KEP-5945 yet and fails to copy the field.

- **Kubelet Feature Gate Disabled / SkipNodeOperations configured**:
  - If the control plane has the gate enabled and writes `skipNodeOperations`, but the upgraded kubelet has the gate
    disabled:
    - Pods requesting `skipNodeOperations` will
      fail `PrepareResources` with a `DRAOptionalNodeOperations feature gate is disabled on kubelet`
      error.
    - For already running pods (in case the feature gate was disabled after
      successful `PrepareResources`), the kubelet will honor `skipNodeOperations` and skip the
      unprepare call during `UnprepareResources`, allowing the pod to terminate cleanly.

- **Scheduler Feature Gate Disabled / SkipNodeOperations configured**:
  - If the `DRAOptionalNodeOperations` feature gate is disabled in the
    scheduler/allocator, but a driver publishes `ResourceSlices` with
    `skipNodeOperations` (e.g., due to
    inconsistent feature gates in a rolling upgrade, or lingering slices after
    downgrade), the allocator treats those devices as unallocatable.
  - This ensures that we fail allocation early in the scheduling lifecycle
    (which allows rescheduling/retry after correcting the configuration), rather
    than scheduling the pod incorrectly (where fields are not copied to the
    claim status and the kubelet subsequently gets stuck expecting a node-local
    driver).

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `DRAOptionalNodeOperations`
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-scheduler
    - kubelet

The gate is off by default in alpha (v1.37).
It also depends on the `DynamicResourceAllocation` and `NodeDeclaredFeatures`
gates.

###### Does enabling the feature change any default behavior?

No. By default, absent fields evaluate to empty (which defaults to not
skipping in code), meaning all resource claims continue to require node
preparation and cleanup unless explicitly set to skip in the published
`ResourceSlice` by the driver.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Setting the feature gate to `false` and restarting components will disable
it. If disabled, any new allocations for standard drivers will proceed normally.
However, any allocation requests targeting drivers
that set `skipNodeOperations` in their `ResourceSlices`
will fail during allocation, preventing workloads from scheduling into a state
where node preparation is incorrectly expected by the kubelet but cannot be satisfied.

###### What happens if we reenable the feature if it was previously rolled back?

Re-enabling the feature gate is safe. Any claims allocated while the feature was
disabled will have an empty `skipNodeOperations` in their status, so they will
continue to be processed with node-local preparation. Newly allocated claims
after re-enablement can once again utilize no-prep resource pools. No state
corruption or data loss occurs.

###### Are there any tests for feature enablement/disablement?

Yes:

- Registry strategy unit tests verify the drop-on-disabled and
  preserve-if-already-in-use behavior for both `ResourceSlice` and
  `ResourceClaim` status.
- Allocator unit tests verify that devices with `skipNodeOperations` are not
  selected when the feature is disabled.
- Kubelet unit tests verify that `PrepareResources` fails when the gate is
  disabled but the allocation result asks to skip an operation, and that the
  `ClaimInfo` checkpoint written with the field set can still be read back
  after the gate is disabled (and by an older kubelet).

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

- Nodes that do not yet advertise `DRAOptionalNodeOperations`
  are filtered out by the scheduler, so pods wait until a supporting node is
  available.
- A rollback can fail if pods were deployed relying on a driver with no
  node-local driver, blocking those pods' restarts.
- *Mitigation*: Operators should ensure no pods using `skipNodeOperations` are active in the
  cluster before disabling the feature gate.

###### What specific metrics should inform a rollback?

An increase in `dra_operations_duration_seconds` or
`FailedPrepareDynamicResources` warnings on the kubelet, indicating the kubelet
is attempting node preparation and blocking/failing due to missing node drivers.
- `dra_node_prepare_skips_total` / `dra_node_unprepare_skips_total` failing to
  increase for drivers configured with `skipNodeOperations`.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This will be tested as part of the Beta graduation criteria using the
upgrade/downgrade E2E test plan. See [Scenario 3: Upgrade / Downgrade and
Feature Gate Rollback](#scenario-3-upgrade--downgrade-and-feature-gate-rollback)
in the Test Plan for the detailed test cases.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

By exposing and monitoring the kubelet-side counter metrics
`dra_node_prepare_skips_total` and
`dra_node_unprepare_skips_total`, or by auditing active `ResourceClaim`
allocations to check if `.status.allocation.devices.results[*].skipNodeOperations` is non-empty.

###### How can someone using this feature know that it is working for their instance?

- [x] API .status
  - Other field: `.status.allocation.devices.results[*].skipNodeOperations` will list the skipped operations in
    the `ResourceClaim`.
  - Workloads run successfully without node-local drivers deployed.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- Bypassing the node driver lookup should reduce pod startup latency
  (`prepareResources`) for resources not requiring node preparation to
  near-zero.
- 0% error rate in kubelet resource preparation for no-prep claims.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `dra_operations_duration_seconds`
    - Components exposing the metric: kubelet
  - Metric name: `dra_node_prepare_skips_total`, `dra_node_unprepare_skips_total`
    - Components exposing the metric: kubelet
- [x] Other
  - Details: rate of `FailedPrepareDynamicResources` pod events.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No. The metrics `dra_node_prepare_skips_total` and
`dra_node_unprepare_skips_total` were introduced in alpha (v1.37).

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. It does not depend on any external services or separate cluster-level
services. It is an in-tree feature within kube-apiserver, kube-scheduler, and
kubelet (it interacts with the in-tree Node Declared Features framework and
Dynamic Resource Allocation).

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. It reuses existing API objects and calls.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, slightly:
- API type(s): `ResourceSlice` (`spec.skipNodeOperations`) and `ResourceClaim` (`status.allocation.devices.results[*].skipNodeOperations`).
- Estimated increase in size: An optional string slice (`[]SkipNodeOperation`), typically 0 to ~30 bytes when populated.
- Estimated amount of new objects: None.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. It actually reduces time taken by kubelet pod startup since it skips gRPC
lookups and network calls.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. Bypassing node-local drivers reduces total cluster-wide memory and CPU
consumption by eliminating unnecessary helper daemonsets.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. In fact, it prevents resource exhaustion by eliminating the need to run
dummy daemonsets on every node for drivers without node-local drivers, which
saves PIDs, memory, and sockets.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The kubelet relies on its locally saved `ClaimInfo` cache. If etcd is down, new
claims cannot be created/scheduled, but existing pods can be terminated cleanly
without requiring API server calls for no-prep claims.

###### What are other known failure modes?

- **Misconfigured driver skip settings**: If a driver controller misconfigures `skipNodeOperations` for a physical
device that *does* require node preparation, the kubelet will skip preparation,
causing containers to start without necessary mounts or initialization, leading
to container application crashes.
  - *Mitigation*: Driver developers and administrators must ensure that
  `skipNodeOperations` is only applied to
  `ResourceSlice`s representing resources that require absolutely no node-local
  preparation or device plumbing on the worker nodes.

- **Driver requirements change in-place**: If a driver's node preparation requirements
  are updated in-place (e.g., changing `skipNodeOperations` in new resource slices),
  existing claims will still use the older configuration. Specifically:
  - If changing from skipping to requiring preparation, existing claims will still
    have node preparation skipped by the kubelet (potentially causing pod failures).
  - If changing from requiring to skipping preparation and decommissioning the node-local
    driver, existing claims will still require node preparation, causing the kubelet to
    fail or hang waiting for the missing driver plugin.
  - *Mitigation*: Administrators must perform such migrations/upgrades carefully
  (e.g., ensuring no active claims or pods exist for the driver before updating
  its configuration or decommissioning node-local driver components).

- **Older or custom allocator fails to copy field**: If an older or custom scheduler/allocator does
  not support copying the skip field from the `ResourceSlice` to the `ResourceClaim` status, the
  kubelet will default to executing node preparation, which will fail if the driver has no node-local
  component deployed on the worker nodes.
  - *Mitigation*: Ensure the custom allocator/scheduler is upgraded to support and copy the new fields
    before deploying optional-operations drivers, or temporarily run a minimal "no-op" node-local
    daemon for the driver.

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Verify if the affected Pod has `FailedPrepareDynamicResources` events.
2. Inspect the associated `ResourceClaim` status: `kubectl get resourceclaim
   <claim-name> -o yaml`.
3. Check if `.status.allocation.devices.results[*].skipNodeOperations` is set.
   If it is empty but the driver is configured with `skipNodeOperations`
   in its `ResourceSlice`, verify if the scheduler or custom allocator has been upgraded to
   support KEP-5945 and correctly copies this field.
4. If allocation itself is failing for the pod's claims with errors indicating that
   the optional node operations feature is disabled in the scheduler, verify that the
   `DRAOptionalNodeOperations` feature gate is enabled in the scheduler/allocator components.
5. If `PrepareResources` fails with a `DRAOptionalNodeOperations feature gate is disabled on kubelet` error,
   verify that the `DRAOptionalNodeOperations` feature gate is enabled on the
   target kubelet.
6. If a terminating pod was deleted and skipped cleanup, verify if it had
   `skipNodeOperations` set in its allocation result, which allows bypassing
   cleanup even when the feature gate is disabled.
7. If resource preparation succeeded (skipped) but the container fails to start
   or run because of missing hardware access, verify that the `ResourceSlice`
   was not misconfigured. If the device actually requires node-local prep,
   `NodePrepareResources` (or `"*"`) must not be listed in `skipNodeOperations`.

## Implementation History

- **2026-05-21**: KEP drafted and proposed as Provisional for Alpha stage.
- **v1.37**: KEP marked `implementable`; alpha implementation merged
  (`DRAOptionalNodeOperations` feature gate, off by default).

## Drawbacks

- Adds a new configuration field to the API, which increases API surface
  area. However, this is necessary to support controller-managed or logical
  resources natively without node-local drivers in a clean way.

## Alternatives

### Alternative 1: DeviceClass-level configuration
Configure this on the cluster-scoped `DeviceClassSpec`.
- *Reason for Rejection*: The cluster administrator shouldn't have to specify
  whether a device needs node preparation. Shifting it to `ResourceSlice`
  (driver-owned) makes it fully automatic and matches the driver's self-declared
  capability.

### Alternative 2: Claim-level declaration
Allow users to declare `skipNodeOperations` in their `ResourceClaimSpec`.
- *Reason for Rejection*: Users should not be concerned with, or even know
  about, the underlying node-level physical or logical prep requirements of the
  hardware. This is an operational and infrastructure concern that belongs
  entirely to the vendor and scheduler/kubelet.

### Alternative 3: Kubelet Auto-Discovery / gRPC probe with timeout
Instead of using an API field, the kubelet could automatically probe for a local
driver. If no driver is registered after a short timeout, it assumes preparation
is not needed and starts the pod.
- *Reason for Rejection*: This is extremely risky. The kubelet cannot
  distinguish between "no driver is supposed to be here" and "the driver is
  crashed, slow to start, or overloaded". Using a timeout would result in flaky
  pod startups, silent failures, and potential security/consistency issues where
  containers launch before their local devices are fully prepared. Explicit
  declaration via the API is highly deterministic and secure.

### Alternative 4: Centralized catch-all no-op plugin
Deploy a generic, "no-op" DRA driver (such as
[dra-driver-noop](https://github.com/gke-labs/dra-drivers/tree/main/dra-driver-noop))
configured centrally to register under specific DRA driver names and handle the
node preparation calls by immediately returning success without doing any actual
work.
- *Reason for Rejection*: While this allows running without modifying the DRA
  API, it has several drawbacks:
  - **Operational Overhead**: It requires deploying and managing an additional
    daemon/driver on nodes just to satisfy the Kubelet's handshake, increasing
    operational complexity.
  - **Mixed-mode coordination**: It is difficult to coordinate in environments
    with "mixed-mode" resources, where some devices of a particular driver name
    require actual node-local preparation (and thus need a real driver) while
    others do not. A static "catch-all" driver cannot easily co-exist or
    coordinate with a real driver registering under the same driver name on the
    same node to selectively handle or bypass preparation.
  - **Lack of Explicit Intent**: It hides the logical nature of the resource
    behind a dummy driver, making debugging and cluster observation more
    difficult compared to an explicit `SkipNodeOperations` field in the
    `ResourceSlice`.
