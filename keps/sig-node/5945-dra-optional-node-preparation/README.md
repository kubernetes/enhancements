# KEP-5945: DRA Optional Node Preparation

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Deploying controller-managed resources without node-local drivers](#deploying-controller-managed-resources-without-node-local-drivers)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
    - [API Server Handling and Ratcheting Validation](#api-server-handling-and-ratcheting-validation)
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

This KEP introduces **Optional Node Preparation** to Dynamic Resource Allocation
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

We propose adding a boolean field `SkipNodeOperations` to
`ResourceSliceSpec` and `DeviceRequestAllocationResult`.

1. **API Definition**: The driver/controller publisher sets `SkipNodeOperations:
   true` in `ResourceSlice` resources if the published devices do not require
   node-local setup or cleanup.
2. **Control Plane Resolution**: The allocator/scheduler resolves the referenced
   `ResourceSlice` during allocation, and copies this configuration into
   `ResourceClaim.Status.Allocation.Devices.Results[i].SkipNodeOperations`.
3. **Node Execution**: The kubelet reads this field from the `ResourceClaim`'s
   allocation results. If all allocated devices for a given driver within a
   claim set `SkipNodeOperations: true`, the kubelet bypasses the corresponding
   gRPC calls (`NodePrepareResources` and `NodeUnprepareResources`) to the
   node-local resource driver.

### User Stories

#### Deploying controller-managed resources without node-local drivers
As a cluster administrator or vendor using a central driver controller, I want
to offer resources (e.g., cluster-wide shared resource pools or logically
partitioned network services) where availability is discovered and published as
`ResourceSlice` resources centrally by the controller. Because the devices
require no node-local plumbing or mount operations on worker nodes, there is no
node driver deployed. The controller publishes these resources with
`SkipNodeOperations: true`. When users request these
devices, the kubelet launches the pods immediately and cleanly, without
complaining about missing node-local drivers, and without requiring any node
driver DaemonSet to be present in the cluster.

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
  - *Mitigation*: The behavior depends on whether the driver uses optional node preparation:
    1. **For drivers that do not use optional node preparation** (i.e., require node-local setup):
       The pointer fields default to `nil` when omitted, which is treated as `false` (not skipped).
       The kubelet will execute node preparation and clean-up as normal. This guarantees 100% backward
       compatibility with all existing schedulers, custom controllers, and running workloads.
    2. **For drivers that use optional node preparation** (and do not deploy a node-local driver):
       If an old or out-of-tree allocator fails to copy the skip fields from the `ResourceSlice` to the
       `ResourceClaim` status, the kubelet will default to executing preparation and fail because no
       node-local driver is running. To mitigate this:
       - Custom allocators/schedulers must be upgraded to support and copy the new fields before they can
         be used with optional-preparation drivers.
       - Alternatively, during transitions, operators can deploy a minimal, "no-op" node-local daemon for
         the driver to satisfy the kubelet's gRPC calls until the allocator is upgraded.

## Design Details
### API Changes

1. **`ResourceSliceSpec`**:
   ```go
   type ResourceSliceSpec struct {
       ...
       // SkipNodeOperations indicates that node-local resource operations (NodePrepareResources and NodeUnprepareResources gRPC calls)
       // are not required for the devices in this slice. Defaults to nil (false).
       // +optional
       SkipNodeOperations *bool `json:"skipNodeOperations,omitempty" protobuf:"varint,9,opt,name=skipNodeOperations"`
   }
   ```

2. **`DeviceRequestAllocationResult`**:
   ```go
   type DeviceRequestAllocationResult struct {
       ...
       // SkipNodeOperations indicates that node-local operations are not required for this allocated device.
       // Typically copied from the corresponding ResourceSliceSpec by the allocator/scheduler. Defaults to nil (false).
       // +optional
       SkipNodeOperations *bool `json:"skipNodeOperations,omitempty" protobuf:"varint,11,opt,name=skipNodeOperations"`
   }
   ```

#### API Server Handling and Ratcheting Validation

The `kube-apiserver` validates the new `SkipNodeOperations` field against the
state of the `DRAOptionalNodePreparation` feature gate to prevent workloads from
entering a broken state. To support safe cluster rollbacks and downgrades, this
feature gate enforcement is implemented using **Ratcheting Validation** (in
accordance with the [Kubernetes API Changes
Guide](https://github.com/kubernetes/community/blob/main/contributors/devel/sig-architecture/api_changes.md#ratcheting-validation)):

* **When the feature gate is disabled**:
  * **New Resources (POST)**: Any attempt to create a `ResourceSlice` or
    allocate a `ResourceClaim` (via its status) with `SkipNodeOperations`
    set to `true` is **rejected** with a validation error.
  * **Existing Resources (PUT)**: The API server allows updates to existing
    resources that already have this field set to `true` (e.g., persisted
    while the feature gate was enabled before a downgrade), provided the update
    does not attempt to newly enable or modify this field. Any transition of
    this field from `nil`/`false` to `true` is **rejected**.
* **When the feature gate is enabled**:
  * The fields are validated and persisted normally.

### Allocator Changes

During scheduling, the structured parameters allocator resolves `ResourceSlices`
that contain the allocated devices.

If the `DRAOptionalNodePreparation` feature gate is enabled:
- The allocator extracts the `SkipNodeOperations` boolean
  value from the corresponding `ResourceSliceSpec` and copies it directly into
  each `DeviceRequestAllocationResult` under
  `ResourceClaim.Status.Allocation.Devices.Results`.

If the `DRAOptionalNodePreparation` feature gate is disabled:
- If any resolved `ResourceSlice` has `SkipNodeOperations`
  set to `true`, the allocator will fail the allocation of this claim. This
  prevents scheduling pods when node operations cannot be safely bypassed or
  properly requested.

### Kubelet Changes

When Kubelet prepares resources for an allocated claim, it evaluates the
allocated devices' status:
1. **Aggregation**: Because Kubelet invokes preparation and clean-up per-claim,
   Kubelet can only bypass node operations if **all** devices for a given driver
   allocated in a claim have `SkipNodeOperations` set to `true`.
2. **Checkpointing**: Kubelet caches this aggregated property inside its
   checkpointed, claim-specific state (`ClaimInfo`) so it is safely preserved
   across Kubelet restarts. To ensure robust upgrade/downgrade compatibility, the
   checkpoint serialization must be forward and backward compatible.
3. **Bypassing**: During `PrepareResources` and `UnprepareResources`, the DRA manager
   checks the claim's cached properties. If skipping is enabled for the driver
   under that claim (meaning all allocated devices have `SkipNodeOperations`
   explicitly set to `true` in the allocation result), it bypasses driver registry
   lookup and the respective gRPC calls (`NodePrepareResources` or `NodeUnprepareResources`),
   allowing container startup/pod termination to proceed immediately. If any device
   has a `nil` or `false` value, it defaults to `false` (do not skip).
4. **Disabled Feature Gate Behavior**: If the `DRAOptionalNodePreparation`
   feature gate is disabled on the kubelet:
   - **Early Admission Failure**: If the `NodeDeclaredFeatures` framework is
     active, the Kubelet's pod admission handler will use the shared library to
     infer the pod's requirements. If a pod requires
     `DraOptionalNodePreparation` (because its allocated claims specify skipping
     node operations) but the Kubelet has the feature gate disabled (meaning it does
     not declare the feature in its status), the Kubelet will **reject the pod
     during admission**. This prevents the pod from attempting to run and
     failing later.
   - **Defense-in-Depth for `PrepareResources`**: If a pod somehow bypasses the
     Kubelet's admission handler, the DRA manager's existing check acts as a
     secondary defense: if a claim's allocation result specifies
     `SkipNodeOperations: true`, the DRA manager fails
     `PrepareResources` immediately with a `DRAOptionalNodePreparationDisabled`
     error, preventing the pod from running with uninitialized hardware.
   - **Safe Rollback for `UnprepareResources`**: Since we already validate and
     fail during admission or `PrepareResources`, we do not need any additional
     checks or errors during `UnprepareResources` if the feature gate is
     disabled. Specifically, if a pod with `SkipNodeOperations: true` was already
     processed (e.g., when the feature gate was enabled) but the feature gate is
     subsequently disabled, the Kubelet will still skip the unprepare call and
     allow the pod to terminate cleanly. This honors the original intent and
     prevents the pod from getting permanently stuck in the `Terminating` state.
     Since the pod is already running, it is not subject to new admission checks
     during termination.

### Node Declared Features Integration

To manage version skew safely during rolling upgrades, this KEP integrates with
the **Node Declared Features** framework
This allows the control plane to dynamically discover if a node's Kubelet
supports optional node preparation before scheduling workloads, preventing pods
from being scheduled to incompatible nodes.

We register a new declared feature:
*   **Feature Name**: `DraOptionalNodePreparation`
*   **Associated Feature Gate**: `DRAOptionalNodePreparation`
*   **Discovery Logic (Kubelet)**: A node declares support for
    `DraOptionalNodePreparation` in its `node.status.declaredFeatures` if and
    only if the `DRAOptionalNodePreparation` feature gate is enabled on the
    Kubelet.
*   **Inference Logic (Scheduler & Admission)**: The control plane infers that a
    Pod requires the `DraOptionalNodePreparation` feature if:
    1.  The Pod references one or more `ResourceClaims`.
    2.  At least one of those claims is allocated (has an `AllocationResult`).
    3.  Within the allocation result, any device has `SkipNodeOperations`
    set to `true`. If these conditions are met, the Pod is
    marked as requiring `DraOptionalNodePreparation` on the target node.
*   **Max Version**: This feature ceases to be a scheduling constraint once the
    `DRAOptionalNodePreparation` feature graduates to GA and the minimum
    supported Kubelet version in the cluster skew policy guarantees support.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

- **API Server Validation Unit Tests**: In `pkg/apis/resource/validation/validation_test.go`:
  - Verify the four ratcheting validation cases when the feature gate is disabled:
    - New valid (field `nil`/`false`) -> Succeeds.
    - New invalid (field `true`) -> Fails.
    - Old valid (field `nil`/`false`) -> Succeeds.
    - Old invalid (field `true` in old object, unchanged in new) -> Succeeds.
- **Allocator Unit Tests**: In
  `staging/src/k8s.io/dynamic-resource-allocation/structured/allocator_test.go`:
  - Verify that `SkipNodeOperations` in `ResourceSliceSpec`
    is correctly propagated to `AllocationResult` (covering combinations of
    true, false, and omitted cases).
- **Kubelet DRA Manager Unit Tests**: In `pkg/kubelet/cm/dra/manager_test.go`:
  - Mock claims with valid values of `SkipNodeOperations` (true, false, nil).
  - Assert that `prepareResources` and `unprepareResources` behave accordingly:
    - If `SkipNodeOperations` is `true`, `prepareResources` and `unprepareResources` bypass the plugin manager and succeed immediately.
    - If `SkipNodeOperations` is `false` or `nil`, `prepareResources` and `unprepareResources` attempt to call the local driver.
- **Shared Library Unit Tests**: In
  `staging/src/k8s.io/component-helpers/nodedeclaredfeatures/features/draoptionalnodepreparation_test.go`:
  - Verify the node declared feature for `DraOptionalNodePreparation` behaves
    correctly.
- **Kubelet Admission Unit Tests**: In Kubelet pod admission tests:
  - Verify that the Kubelet's pod admission handler rejects a pod requiring
    `DraOptionalNodePreparation` if the feature gate is disabled on the Kubelet.
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
      the field defaulting to `false`/`nil` (ensuring we do not skip
      preparation/unpreparation for legacy claims).

##### Integration tests

- **Scheduler Filtering Integration Tests**: In
  `test/integration/scheduler/filters/`:
  - Verify that a pod requiring `DraOptionalNodePreparation` (having a claim
    allocated with skip fields set to `true`) is successfully scheduled to a
    node that advertises the feature.
  - Verify that the scheduler filters out (rejects) nodes that do not advertise
    the feature (representing older Kubelets or nodes with the feature gate
    disabled).
  - Verify that if no compatible nodes are available, the pod remains in the
    `Pending` state with a `FailedScheduling` event indicating the missing
    `DraOptionalNodePreparation` feature on nodes.

##### e2e tests

We will add new End-to-End test cases inside `test/e2e/dra/dra.go` to validate
`SkipNodeOperations` configurations using different
driver configurations.

###### Scenario 1: Driver without node-local components (Pure Control-Plane)
This scenario validates that we can run workloads using drivers that do not
deploy any node-local components.

- **Setup**: Deploy a DRA test driver without node gRPC components running on
  worker nodes (`WithKubelet = false`).
- Test Case 1.1: Fully skipped node operations (`SkipNodeOperations: true`)
  - **API Configuration**: Publish `ResourceSlices` with `SkipNodeOperations` set to
    `true`.
  - **Workload**: Deploy a Pod referencing this resource.
  - **Assertions**:
    - The Pod reaches the `Running` phase successfully.
    - No `FailedPrepareDynamicResources` warnings are posted to the Pod events.
    - Pod deletion completes cleanly and immediately (does not hang in
      `Terminating` waiting for unprepare).
- Test Case 1.2: Missing node component failure (`SkipNodeOperations: false`)
  - **API Configuration**: Publish `ResourceSlices` with `SkipNodeOperations` set
    to `false` (or omitted).
  - **Workload**: Deploy a Pod referencing this resource.
  - **Assertions**:
    - The Pod gets stuck in `ContainerCreating`
      with `FailedPrepareDynamicResources` errors because the kubelet tries to
      contact the non-existent node driver.

###### Scenario 2: Driver with node-local components (Standard Driver)
This scenario validates that the kubelet invokes the node-local
driver when `SkipNodeOperations` is `false`.

- **Setup**: Deploy a standard DRA test driver that includes node-local gRPC
  components.
- Test Case 2.1: Standard Node Execution (`SkipNodeOperations: false`)
  - **API Configuration**: Publish `ResourceSlices` with `SkipNodeOperations:
    false`.
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
          `NodeDeclaredFeatures` plugin must filter out all N-1 worker nodes because
          they do not advertise the `DraOptionalNodePreparation` feature in their status.
        - The pod must not be scheduled to any N-1 node.
  - **Action 2 (Kubelet Upgrade)**: Upgrade the kubelets to version N.
    - **Assertions**:
      - Once a Kubelet is upgraded to N and advertises `DraOptionalNodePreparation`,
        verify that the pending workload is **automatically scheduled** to that node,
        successfully bypasses node preparation, transitions to `Running`, and runs successfully.
      - Verify that deleting the control-plane-only workload completes
        immediately without trying to contact a node-local driver.
- **Test Case 3.2: Feature Gate Rollback / Downgrade (N to N-1)**:
  - **Setup**: Start with a cluster running version N (feature gate enabled).
    Deploy a standard DRA driver and a workload using `SkipNodeOperations: true`.
  - **Action 1 (Control Plane Downgrade)**: Downgrade the control plane to N-1
    (feature gate disabled).
    - **Assertions**:
      - The API server's ratcheting validation allows the existing
        `ResourceSlice` objects to remain valid and not be rejected on unrelated
        updates.
      - The running workload on the N kubelet continues to run without
        interruption.
  - **Action 2 (Kubelet Downgrade / Feature Gate Rollback)**: Downgrade the
    Kubelet binary to N-1, or disable the `DRAOptionalNodePreparation` feature
    gate on Kubelet version N, and restart the Kubelet.
    - **Assertions**:
      - **Checkpoint Recovery**: The Kubelet starts up successfully and parses
        the checkpoint file without errors or crashes.
        - *For Kubelet version N (gate disabled)*: The Kubelet successfully
          recovers the full `ClaimInfo` state including the saved
          `SkipNodeOperations: true` setting.
        - *For Kubelet version N-1*: The Kubelet successfully parses the
          checkpoint by ignoring the unknown fields, and recovers the rest of
          the state while defaulting the missing skip field to `false`.
      - **Workload Deletion**:
        - Verify that the kubelet behaves according to the
          [Disabled Feature Gate Behavior](#disabled-feature-gate-behavior) section
          upon workload deletion.
- **Test Case 3.3: Upgrade -> Downgrade -> Upgrade (N-1 -> N -> N-1 -> N)**:
  - **Setup**: Start with a cluster running version N-1 (feature gate disabled).
    Deploy a standard DRA driver.
  - **Action 1 (Upgrade)**: Upgrade the cluster to version N (feature gate
    enabled).
    - Deploy a workload using a driver configured with `SkipNodeOperations: true`.
    - Assert that the workload runs successfully.
  - **Action 2 (Downgrade)**: Downgrade the cluster to version N-1 (feature gate
    disabled).
    - **Assertions**:
      - Assert that the API server's ratcheting validation allows the existing
        `ResourceSlice` (which has `SkipNodeOperations: true`) to remain valid and unmodified.
      - Delete the workload and verify that the kubelet behaves according to the
        [Disabled Feature Gate Behavior](#disabled-feature-gate-behavior) section.
  - **Action 3 (Upgrade Again)**: Upgrade the cluster back to version N (feature
    gate enabled).
    - Deploy a new workload using the same driver.
    - Assert that the new field is respected, and the workload runs
      successfully.
    - Assert that any pre-existing resource slices that survived the downgrade
      cycle continue to function correctly with the re-enabled feature gate.

### Graduation Criteria

#### Alpha

- Feature implemented behind the `DRAOptionalNodePreparation` feature flag (off
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
    claims (where the new pointer field is absent/`nil`) automatically evaluate
    to `false` (not skipped). This guarantees no change in behavior for running
    workloads.
  - Newer claims can utilize drivers that publish resource slices configured
    with `SkipNodeOperations: true` to bypass
    node-local execution.
  - During rolling upgrades, the scheduler's `NodeDeclaredFeatures` plugin will
    automatically restrict the scheduling of pods using these newer "no-prep" claims
    to upgraded nodes that advertise support for `DraOptionalNodePreparation`.
- **Downgrade**:
  - If a cluster is downgraded to a version where `DRAOptionalNodePreparation`
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
    generates allocations with `SkipNodeOperations: true`,
    the scheduler's `NodeDeclaredFeatures` plugin will automatically
    infer that the pod requires the `DraOptionalNodePreparation` feature.
  - Because older worker nodes running older Kubelets ($N-1$ and older) do not
    support the feature gate, they will not advertise
    `DraOptionalNodePreparation` in their `node.status.declaredFeatures`.
  - The scheduler will **automatically filter out these older nodes** during the
    scheduling cycle, guaranteeing that the pod will only land on compatible,
    upgraded nodes.
- **Upgraded kubelet (N) / Older Control Plane (N-1 and older)**:
  - If the control plane has not been upgraded yet, any new allocations will not
    have `SkipNodeOperations` set in the status.
  - An upgraded kubelet (N) will read the absent fields and default to `false`
    (requiring node preparation/unpreparation).
  - The behavior depends on whether the driver uses optional node preparation:
    - **For drivers that do not use optional node preparation** (i.e., require node-local setup):
      The fallback to `false` ensures backward-compatible, safe execution because the
      node-local driver is running and kubelet will coordinate with it as normal.
    - **For drivers that use optional node preparation** (and do not deploy a node-local driver):
      The fallback to `false` means the upgraded kubelet will attempt to coordinate
      with the local driver and fail because no node-local driver is running.
      - *Mitigation*: The control plane must be upgraded before these optional-preparation
        drivers can be deployed, or a temporary, minimal "no-op" node-local daemon must
        be deployed to satisfy the kubelet's gRPC calls during the transition window.

    *Note*: This same fallback behavior occurs if the control plane is upgraded (N)
    but the active custom allocator or scheduler has not been upgraded to support
    KEP-5945 yet and fails to copy the field.

- **Kubelet Feature Gate Disabled / SkipNodeOperations set to true**:
  - If the control plane has the gate enabled and writes `SkipNodeOperations: true`, but the upgraded kubelet has the gate
    disabled:
    - Pods requesting `SkipNodeOperations: true` will
      fail `PrepareResources` with a clear `DRAOptionalNodePreparationDisabled`
      error.
    - For already running pods (in case the feature gate was disabled after
      successful `PrepareResources`), the kubelet will honor `SkipNodeOperations: true` and skip the
      unprepare call during `UnprepareResources`, allowing the pod to terminate cleanly.

- **Scheduler Feature Gate Disabled / SkipNodeOperations set to true**:
  - If the `DRAOptionalNodePreparation` feature gate is disabled in the
    scheduler/allocator, but a driver publishes `ResourceSlices` with
    `SkipNodeOperations: true` (e.g., due to
    inconsistent feature gates in a rolling upgrade, or lingering slices after
    downgrade), the scheduler/allocator will fail the allocation of those
    claims.
  - This ensures that we fail allocation early in the scheduling lifecycle
    (which allows rescheduling/retry after correcting the configuration), rather
    than scheduling the pod incorrectly (where fields are not copied to the
    claim status and the kubelet subsequently gets stuck expecting a node-local
    driver).

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `DRAOptionalNodePreparation`
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-scheduler
    - kubelet

###### Does enabling the feature change any default behavior?

No. By default, absent pointer fields evaluate to `nil` (which defaults to
`false` in code), meaning all resource claims continue to require node
preparation and cleanup unless explicitly set to `true` in the published
`ResourceSlice` by the driver.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Setting the feature gate to `false` and restarting components will disable
it. If disabled, any new allocations for standard drivers will proceed normally
(propagating `false` or `nil`). However, any allocation requests targeting drivers
that set `SkipNodeOperations: true` in their `ResourceSlices`
will fail during allocation, preventing workloads from scheduling into a state
where node preparation is incorrectly expected by the kubelet but cannot be satisfied.

###### What happens if we reenable the feature if it was previously rolled back?

Re-enabling the feature gate is safe. Any claims allocated while the feature was
disabled will have the skip field as `false` in their status, so they will
continue to be processed with node-local preparation. Newly allocated claims
after re-enablement can once again utilize no-prep resource pools. No state
corruption or data loss occurs.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests in the allocator will verify that when the feature gate is
disabled, if any `ResourceSlice` has `SkipNodeOperations: true`,
the allocator returns an error and fails allocation.
Kubelet unit tests will verify that when the feature gate is disabled on the node:
- It fails `PrepareResources` if any active claim has `SkipNodeOperations: true`.
- During `UnprepareResources` of an already running pod, it still skips cleanup if the claim
  has `SkipNodeOperations: true`, allowing the pod to terminate cleanly.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

- A rollback can fail if pods were deployed relying on a driver with no
  node-local driver. If rolled back, the kubelet will start expecting a node
  driver, blocking those pods' termination or restarts.
- *Mitigation*: Operators should ensure no no-prep pods are active in the
  cluster before disabling the feature gate.

###### What specific metrics should inform a rollback?

An increase in `dra_operations_duration_seconds` or
`FailedPrepareDynamicResources` warnings on the kubelet, indicating the kubelet
is attempting node preparation and blocking/failing due to missing node drivers.

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
`kubelet_dra_node_prepare_skips_total` and
`kubelet_dra_node_unprepare_skips_total`, or by auditing active `ResourceClaim`
allocations to check if `.status.allocation.devices.results[*].skipNodeOperations` is set to `true`.

###### How can someone using this feature know that it is working for their instance?

- [x] API .status
  - Other field: `.Status.Allocation.Devices.Results[*].SkipNodeOperations` will be `true` in
    the `ResourceClaim`.
  - Workloads run successfully without node-local drivers deployed.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- Bypassing the node driver lookup should reduce pod startup latency
  (`prepareResources`) for resources not requiring node preparation to
  near-zero.
- 0% error rate in kubelet resource preparation for no-prep claims.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- kubelet metrics: `kubelet_dra_operations_duration_seconds` for `prepare` and
  `unprepare` actions.
- Core Event rate for `FailedPrepareDynamicResources`.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Yes, we propose introducing two new kubelet-side counter metrics:
`kubelet_dra_node_prepare_skips_total` and
`kubelet_dra_node_unprepare_skips_total` (partitioned by `driver_name`). This
will track the total number of preparation and cleanup operations skipped
because the claim's resources do not require node-local setup, allowing
operators to easily monitor optional preparation usage without querying the API
server.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

Yes. This feature depends on the **Node Declared Features** framework.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. It reuses existing API objects and calls.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, slightly. An optional boolean pointer field is added to
`ResourceSliceSpec` and `DeviceRequestAllocationResult`.

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

- **Misconfigured driver skip settings**: If a driver controller misconfigures `SkipNodeOperations: true` for a physical
device that *does* require node preparation, the kubelet will skip preparation,
causing containers to start without necessary mounts or initialization, leading
to container application crashes.
  - *Mitigation*: Driver developers and administrators must ensure that
  `SkipNodeOperations: true` is only applied to
  `ResourceSlice`s representing resources that require absolutely no node-local
  preparation or device plumbing on the worker nodes.

- **Driver requirements change in-place**: If a driver's node preparation requirements
  are updated in-place (e.g., changing `SkipNodeOperations` in new resource slices),
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
    before deploying optional-preparation drivers, or temporarily run a minimal "no-op" node-local
    daemon for the driver.

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Verify if the affected Pod has `FailedPrepareDynamicResources` events.
2. Inspect the associated `ResourceClaim` status: `kubectl get resourceclaim
   <claim-name> -o yaml`.
3. Check if `.status.allocation.devices.results[*].skipNodeOperations` is set to
   `true`. If it is `nil` or `false` but the driver is configured with `SkipNodeOperations: true`
   in its `ResourceSlice`, verify if the scheduler or custom allocator has been upgraded to
   support KEP-5945 and correctly copies this field.
4. If allocation itself is failing for the pod's claims with errors indicating that
   the optional node preparation feature is disabled in the scheduler, verify that the
   `DRAOptionalNodePreparation` feature gate is enabled in the scheduler/allocator components.
5. If `PrepareResources` fails with a `DRAOptionalNodePreparationDisabled` error,
   verify that the `DRAOptionalNodePreparation` feature gate is enabled on the
   target kubelet.
6. If a terminating pod was deleted and skipped cleanup, verify if it had
   `SkipNodeOperations: true` in its allocation result, which allows bypassing
   cleanup even when the feature gate is disabled.
7. If resource preparation succeeded (skipped) but the container fails to start
   or run because of missing hardware access, verify that the `ResourceSlice`
   was not misconfigured. If the device actually requires node-local prep,
   `SkipNodeOperations` must be set to `false` (or omitted).

## Implementation History

- **2026-05-21**: KEP drafted and proposed as Provisional for Alpha stage.

## Drawbacks

- Adds a new boolean configuration field to the API, which increases API surface
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
Allow users to declare `SkipNodeOperations: true` in their `ResourceClaimSpec`.
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
