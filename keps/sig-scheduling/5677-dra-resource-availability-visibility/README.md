# KEP-5677: DRA Resource Availability Visibility

<!-- toc -->
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [User Stories](#user-stories)
      - [Story 1: Cluster Administrator Monitoring Resources](#story-1-cluster-administrator-monitoring-resources)
      - [Story 2: Developer Debugging Resource Allocation](#story-2-developer-debugging-resource-allocation)
      - [Story 3: Capacity Planning](#story-3-capacity-planning)
    - [Notes/Constraints/Caveats](#notesconstraintscaveats)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Design Details](#design-details)
    - [API Changes](#api-changes)
      - [ResourceSlice Status](#resourceslice-status)
      - [Status Fields](#status-fields)
    - [Controller Implementation](#controller-implementation)
      - [ResourceSlice Status Controller](#resourceslice-status-controller)
      - [Cross-Slice Validation](#cross-slice-validation)
    - [kubectl Integration](#kubectl-integration)
      - [kubectl describe resourceslice](#kubectl-describe-resourceslice)
      - [kubectl describe node](#kubectl-describe-node)
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
- [Check if controller-manager is running](#check-if-controller-manager-is-running)
- [Check if status is being populated](#check-if-status-is-being-populated)
  - [Implementation History](#implementation-history)
  - [Drawbacks](#drawbacks)
  - [Alternatives](#alternatives)
    - [Alternative 1: Status in ResourceClaim instead of ResourceSlice](#alternative-1-status-in-resourceclaim-instead-of-resourceslice)
    - [Alternative 2: New cluster-scoped ResourcePool object](#alternative-2-new-cluster-scoped-resourcepool-object)
    - [Alternative 3: Metrics-based visibility only](#alternative-3-metrics-based-visibility-only)
    - [Alternative 4: Aggregated API server for DRA](#alternative-4-aggregated-api-server-for-dra)
    - [Alternative 5: External controller with custom CRD](#alternative-5-external-controller-with-custom-crd)
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

This KEP addresses a critical visibility gap in Dynamic Resource Allocation (DRA) by enabling users to view available device capacity on nodes and resource pools. While ResourceSlices store capacity data and ResourceClaims track consumption, there is currently no straightforward way for users to view the available capacity remaining in a pool or on a node.

This enhancement introduces:
1. A Status field in ResourceSlice to track resource consumption and availability
2. A ResourceSlice status controller that aggregates allocation information across ResourceClaims
3. Cross-slice validation for resource pools to ensure consistency
4. Enhanced `kubectl describe` commands for ResourceSlice and Node to display availability information

## Motivation

Dynamic Resource Allocation (DRA) provides a flexible framework for managing specialized hardware resources like GPUs, FPGAs, and other accelerators. However, the current implementation lacks visibility into resource availability:

**Current State:**
- ResourceSlices are cluster-scoped resources that publish total capacity of devices in a pool
- ResourceClaims are namespaced and track individual allocations
- Most users cannot see all ResourceClaims consuming resources from a pool (due to namespace boundaries and RBAC)
- No API-level view of "available" vs "allocated" capacity
- Difficult to understand why scheduling is failing or plan capacity

**Problems this creates:**
1. **Debugging difficulty**: When pods fail to schedule due to insufficient resources, users cannot easily see what is available vs. what is consumed
2. **Capacity planning**: Cluster administrators cannot easily determine if more resources are needed
3. **Quota management**: Without visibility into consumption, it's hard to understand quota usage
4. **Cross-namespace visibility**: Since ResourceClaims are namespaced, even cluster admins need to query multiple namespaces to understand total consumption

### Goals

- Add a Status subresource to ResourceSlice that tracks consumption and availability per device/resource
- Implement a ResourceSlice status controller that watches ResourceClaims and updates ResourceSlice status
- Provide cross-slice validation to ensure consistency across ResourceSlices in the same pool
- Enhance `kubectl describe resourceslice` to show capacity, allocated, and available resources
- Enhance `kubectl describe node` to show DRA resource information for node-local resources
- Ensure the solution works for both node-local and network-attached devices
- Support single-device, consumable capacity (multi-allocatable), and partitionable devices

### Non-Goals

- Modifying the core scheduling algorithm (this is purely about visibility)
- Changing how ResourceClaims are allocated
- Adding real-time metrics/monitoring (this is API-level status, not metrics)
- Implementing quotas or limits based on availability (future work)
- Changing the ResourceSlice or ResourceClaim allocation model
- Providing historical consumption data (this is point-in-time status)

## Proposal

### User Stories

#### Story 1: Cluster Administrator Monitoring Resources

As a cluster administrator, I want to see at a glance how many GPU resources are available across my cluster so that I can:
- Understand current resource utilization
- Plan for capacity expansion
- Debug why pods are not scheduling

**Current experience:**
```bash
$ kubectl get resourceslices
NAME                    DRIVER              POOL        DEVICES
node-1-gpu-slice-1     example.com/gpu     node-1      4
node-2-gpu-slice-1     example.com/gpu     node-2      4

# No way to see how many are allocated/available!
# Must query all ResourceClaims across all namespaces and manually correlate
```

**Proposed experience:**
```bash
$ kubectl describe resourceslice node-1-gpu-slice-1
Name:         node-1-gpu-slice-1
Driver:       example.com/gpu
Pool:         node-1
Devices:      4
Status:
  Devices:
    gpu-0:
      Status:  Allocated
      Allocated To:  default/my-gpu-claim
      Capacity:
        memory: 16Gi
      Available:
        memory: 0
    gpu-1:
      Status:  Available
      Capacity:
        memory: 16Gi
      Available:
        memory: 16Gi
  Summary:
    Total Devices: 4
    Allocated: 1
    Available: 3
```

#### Story 2: Developer Debugging Resource Allocation

As a developer, when my pod fails to schedule because "insufficient DRA resources", I want to understand:
- What resources are required
- What resources are available
- Why the match failed

**Current experience:**
- Pod event: "0/3 nodes are available: 3 Insufficient example.com/gpu"
- No visibility into which nodes have what available
- Cannot see resource consumption details

**Proposed experience:**
```bash
$ kubectl describe node node-1
...
DRA Resources:
  Driver: example.com/gpu
  Pool:   node-1
  Devices:
    Total:      4
    Allocated:  3
    Available:  1
  Resource Details:
    - Device: gpu-0 (Allocated to default/ml-training)
    - Device: gpu-1 (Allocated to default/ml-inference)
    - Device: gpu-2 (Allocated to team-a/batch-job)
    - Device: gpu-3 (Available)
```

#### Story 3: Capacity Planning

As a capacity planner, I want to query resource availability across the cluster to:
- Identify nodes or pools that are under-utilized
- Understand trends in resource consumption
- Plan for workload placement

**Proposed experience:**
```bash
$ kubectl get resourceslices -o custom-columns=\
NAME:.metadata.name,\
POOL:.spec.pool.name,\
TOTAL:.status.summary.totalDevices,\
ALLOCATED:.status.summary.allocatedDevices,\
AVAILABLE:.status.summary.availableDevices

NAME                    POOL      TOTAL   ALLOCATED   AVAILABLE
node-1-gpu-slice-1     node-1    4       3           1
node-2-gpu-slice-1     node-2    4       1           3
node-3-gpu-slice-1     node-3    4       4           0
```

### Notes/Constraints/Caveats

1. **Eventually consistent**: The status information is eventually consistent. There may be a brief period where the status doesn't reflect very recent allocations.

2. **Namespace boundaries**: ResourceClaims are namespaced, but ResourceSlice status is cluster-scoped. The status will reference claims by namespace/name, but this doesn't bypass RBAC for accessing the claims themselves.

3. **Multi-allocatable devices**: For devices that support consumable capacity (KEP-5075), the status will track both device-level and capacity-level consumption.

4. **Cross-slice coordination**: When a pool spans multiple ResourceSlices, the status controller must correctly aggregate information across all slices.

5. **Performance**: The controller must be efficient when watching large numbers of ResourceClaims and updating ResourceSlice status.

### Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Performance impact of watching all ResourceClaims | Controller could be slow or use excessive resources | Use informers with proper indexing; implement rate limiting and batching |
| Status inconsistency during updates | Users may see stale availability data | Document eventually-consistent nature; use generation numbers for consistency checks |
| Large number of ResourceSlices | Status updates could overwhelm API server | Implement batch updates; use Status subresource; add rate limiting |
| Security: exposing allocation details | Status reveals which claims are allocated | Document that this is intended; users should use RBAC to restrict ResourceSlice access if needed |
| Cross-slice validation overhead | Validating consistency across slices could be expensive | Implement efficient caching; only validate on changes |

## Design Details

### API Changes

#### ResourceSlice Status

Add a new `Status` field to ResourceSlice. This requires updating the ResourceSlice API to include a status subresource.

```go
// ResourceSlice represents one or more resources in a pool of similar resources
type ResourceSlice struct {
    metav1.TypeMeta
    metav1.ObjectMeta

    Spec ResourceSliceSpec

    // Status describes the current state of resources in this slice,
    // including allocation and availability information.
    // +optional
    Status ResourceSliceStatus `json:"status,omitempty"`
}

// ResourceSliceStatus describes the current state of resources in a ResourceSlice
type ResourceSliceStatus struct {
    // Devices contains per-device status information.
    // Only populated for ResourceSlices that contain devices.
    // +optional
    // +listType=map
    // +listMapKey=name
    Devices []DeviceStatus `json:"devices,omitempty"`

    // Summary provides aggregate information about resources in this slice.
    // +optional
    Summary *ResourceSummary `json:"summary,omitempty"`

    // PoolStatus provides information about the overall pool that this
    // ResourceSlice belongs to. Only one ResourceSlice per pool should
    // have this field populated (typically the one with the lowest name
    // lexicographically).
    // +optional
    PoolStatus *ResourcePoolStatus `json:"poolStatus,omitempty"`

    // ObservedGeneration is the generation of the ResourceSlice spec that
    // this status corresponds to.
    // +optional
    ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// DeviceStatus describes the current status of a single device
type DeviceStatus struct {
    // Name is the name of the device, matching a device in Spec.Devices
    // +required
    Name string `json:"name"`

    // State indicates the current availability state of the device.
    // This is derived from allocations and capacity, not reported by drivers.
    // +required
    State DeviceState `json:"state"`

    // StateReason provides additional detail when State is Unavailable.
    // For example: "InsufficientSharedCapacity" for partitionable devices.
    // +optional
    StateReason string `json:"stateReason,omitempty"`

    // Allocations lists all current allocations of this device.
    // For single-allocation devices, this will have at most one entry.
    // For multi-allocatable devices (consumable capacity), this may have multiple entries.
    // +optional
    // +listType=atomic
    Allocations []DeviceAllocation `json:"allocations,omitempty"`

    // Capacity represents the total capacity of consumable resources for this device.
    // Only set for multi-allocatable devices.
    // +optional
    Capacity ResourceList `json:"capacity,omitempty"`

    // AvailableCapacity represents the remaining available capacity after subtracting
    // all allocations. Only set for multi-allocatable devices.
    // +optional
    AvailableCapacity ResourceList `json:"availableCapacity,omitempty"`
}

// DeviceState represents the availability state of a device.
// Note: Health/error conditions are tracked separately in KEP-5283.
type DeviceState string

const (
    // DeviceStateAvailable indicates the device is fully available for allocation
    DeviceStateAvailable DeviceState = "Available"

    // DeviceStatePartiallyAllocated indicates the device has some allocations
    // but still has available capacity (only for consumable capacity devices)
    DeviceStatePartiallyAllocated DeviceState = "PartiallyAllocated"

    // DeviceStateAllocated indicates the device is fully allocated
    // (no more capacity available)
    DeviceStateAllocated DeviceState = "Allocated"

    // DeviceStateUnavailable indicates the device cannot accept allocations
    // even though it may not be directly allocated. This happens with
    // partitionable devices when shared/parent resources are exhausted.
    // See StateReason for details.
    DeviceStateUnavailable DeviceState = "Unavailable"
)

// DeviceAllocation describes a single allocation of a device or portion of a device
type DeviceAllocation struct {
    // ClaimNamespace is the namespace of the ResourceClaim
    // +required
    ClaimNamespace string `json:"claimNamespace"`

    // ClaimName is the name of the ResourceClaim
    // +required
    ClaimName string `json:"claimName"`

    // ClaimUID is the UID of the ResourceClaim for strong reference
    // +required
    ClaimUID types.UID `json:"claimUID"`

    // Request identifies which request within the claim this allocation satisfies.
    // For claims with multiple requests, this disambiguates which request.
    // +optional
    Request string `json:"request,omitempty"`

    // Capacity represents the portion of device capacity consumed by this allocation.
    // Only set for multi-allocatable devices.
    // +optional
    Capacity ResourceList `json:"capacity,omitempty"`
}

// ResourceSummary provides aggregate information about resources
type ResourceSummary struct {
    // TotalDevices is the total number of devices in this ResourceSlice
    // +optional
    TotalDevices int32 `json:"totalDevices,omitempty"`

    // AllocatedDevices is the number of devices that have at least one allocation
    // +optional
    AllocatedDevices int32 `json:"allocatedDevices,omitempty"`

    // AvailableDevices is the number of devices that are completely unallocated
    // +optional
    AvailableDevices int32 `json:"availableDevices,omitempty"`

    // PartiallyAllocatedDevices is the number of multi-allocatable devices
    // that have some but not all capacity allocated
    // +optional
    PartiallyAllocatedDevices int32 `json:"partiallyAllocatedDevices,omitempty"`

    // TotalCapacity represents total consumable capacity across all devices.
    // Only populated for multi-allocatable devices.
    // +optional
    TotalCapacity ResourceList `json:"totalCapacity,omitempty"`

    // AllocatedCapacity represents total allocated consumable capacity.
    // Only populated for multi-allocatable devices.
    // +optional
    AllocatedCapacity ResourceList `json:"allocatedCapacity,omitempty"`

    // AvailableCapacity represents total remaining consumable capacity.
    // Only populated for multi-allocatable devices.
    // +optional
    AvailableCapacity ResourceList `json:"availableCapacity,omitempty"`
}

// ResourcePoolStatus provides status information about an entire pool
type ResourcePoolStatus struct {
    // Conditions represent observations about the pool's state
    // +optional
    // +listType=map
    // +listMapKey=type
    Conditions []PoolCondition `json:"conditions,omitempty"`

    // ValidationErrors contains errors found during cross-slice validation
    // +optional
    // +listType=atomic
    ValidationErrors []string `json:"validationErrors,omitempty"`

    // ObservedSliceCount is the number of ResourceSlices observed for this pool
    // at the current generation
    // +optional
    ObservedSliceCount int32 `json:"observedSliceCount,omitempty"`

    // ExpectedSliceCount is the expected number of ResourceSlices (from pool.ResourceSliceCount)
    // +optional
    ExpectedSliceCount int32 `json:"expectedSliceCount,omitempty"`
}

// PoolCondition describes a condition of a resource pool
type PoolCondition struct {
    // Type of pool condition
    // +required
    Type PoolConditionType `json:"type"`

    // Status of the condition
    // +required
    Status ConditionStatus `json:"status"`

    // LastTransitionTime is the last time the condition transitioned
    // +optional
    LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

    // Reason is a brief machine-readable explanation
    // +optional
    Reason string `json:"reason,omitempty"`

    // Message is a human-readable explanation
    // +optional
    Message string `json:"message,omitempty"`
}

// PoolConditionType is the type of pool condition
type PoolConditionType string

const (
    // PoolComplete indicates all expected ResourceSlices for the pool are present
    PoolComplete PoolConditionType = "Complete"

    // PoolValid indicates cross-slice validation passed
    PoolValid PoolConditionType = "Valid"
)

// ResourceList is an alias for v1.ResourceList
type ResourceList = v1.ResourceList
```

#### Status Fields

The status provides:
- **Per-device information**: Which devices are allocated, to which claims, and available capacity
- **Aggregate information**: Summary counts and capacity across the slice
- **Pool-level information**: Validation status for the entire pool

### Controller Implementation

#### ResourceSlice Status Controller

A new controller in kube-controller-manager will maintain ResourceSlice status:

**Responsibilities:**
1. Watch ResourceClaims and ResourceSlices
2. For each ResourceSlice, determine which devices are allocated by examining all ResourceClaims
3. Update ResourceSlice status with allocation and availability information
4. Handle both single-device and consumable-capacity allocations

**Implementation approach:**
```go
type ResourceSliceStatusController struct {
    resourceSliceClient resourcev1beta2.ResourceSliceInterface
    resourceClaimLister resourcev1beta2.ResourceClaimLister
    resourceSliceLister resourcev1beta2.ResourceSliceLister

    workqueue workqueue.RateLimitingInterface
}

// Reconcile updates the status of a ResourceSlice based on current allocations
func (c *ResourceSliceStatusController) Reconcile(ctx context.Context, sliceName string) error {
    slice := c.getResourceSlice(sliceName)
    if slice == nil {
        return nil
    }

    // Find all ResourceClaims that reference devices in this slice
    claims := c.findRelevantClaims(slice)

    // Build device status from claims
    deviceStatuses := c.buildDeviceStatuses(slice, claims)

    // Calculate summary
    summary := c.calculateSummary(deviceStatuses, slice.Spec.Devices)

    // Perform cross-slice validation if needed
    poolStatus := c.validatePool(slice)

    // Update status
    status := ResourceSliceStatus{
        Devices:            deviceStatuses,
        Summary:            summary,
        PoolStatus:         poolStatus,
        ObservedGeneration: slice.Generation,
    }

    return c.updateStatus(ctx, slice, status)
}
```

**Indexing:**
The controller will use informer indexes to efficiently query:
- ResourceClaims by driver and pool name
- ResourceSlices by pool name and generation

**Event handling:**
- When a ResourceClaim is created/updated/deleted, enqueue all ResourceSlices for the affected driver/pool
- When a ResourceSlice is created/updated, enqueue it for status update
- Use work queues with rate limiting to prevent API server overload

#### Cross-Slice Validation

For pools that span multiple ResourceSlices, the controller validates consistency:

1. **Device name uniqueness**: Ensure no device name appears in multiple slices of the same pool
2. **Generation consistency**: All slices in a pool should have the same generation number
3. **Pool completeness**: Verify that the observed number of slices matches the expected count

**Validation logic:**
```go
func (c *ResourceSliceStatusController) validatePool(slice *ResourceSlice) *ResourcePoolStatus {
    // Get all slices for this pool at the current generation
    slices := c.getPoolSlices(slice.Spec.Driver, slice.Spec.Pool.Name, slice.Spec.Pool.Generation)

    var errors []string

    // Check device name uniqueness
    deviceNames := make(map[string]string) // device name -> slice name
    for _, s := range slices {
        for _, device := range s.Spec.Devices {
            if existingSlice, exists := deviceNames[device.Name]; exists {
                errors = append(errors, fmt.Sprintf(
                    "device %s appears in both %s and %s",
                    device.Name, existingSlice, s.Name))
            }
            deviceNames[device.Name] = s.Name
        }
    }

    // Check pool completeness
    expectedCount := slice.Spec.Pool.ResourceSliceCount
    observedCount := int32(len(slices))

    conditions := []PoolCondition{
        {
            Type:   PoolComplete,
            Status: metav1.ConditionStatus(observedCount == expectedCount),
            Reason: "SliceCountMatch",
        },
        {
            Type:   PoolValid,
            Status: metav1.ConditionStatus(len(errors) == 0),
            Reason: "ValidationPassed",
        },
    }

    // Only update pool status on one slice (lexicographically first)
    if !c.isPoolStatusOwner(slice, slices) {
        return nil
    }

    return &ResourcePoolStatus{
        Conditions:         conditions,
        ValidationErrors:   errors,
        ObservedSliceCount: observedCount,
        ExpectedSliceCount: expectedCount,
    }
}
```

#### Consistency Handling

The status controller is an observer, not a fixer. It reports inconsistencies but does not attempt to resolve them. Resolution is the responsibility of the DRA driver, scheduler/allocator, or cluster administrators.

**Key principle:** `ResourceSlice.Status.Devices[]` only contains entries for devices that exist in `ResourceSlice.Spec.Devices[]`. The controller never creates phantom status entries for removed devices.

**Scenarios and handling:**

1. **Device removed from ResourceSlice spec while ResourceClaim still references it:**
   - The device no longer exists in spec, so it has no DeviceStatus entry
   - Status controller scans ResourceClaims that reference this pool
   - Detects allocation pointing to non-existent device
   - Reports in `PoolStatus.ValidationErrors`: "ResourceClaim ns/name references non-existent device 'X' in pool 'Y'"
   - **Resolution:** This is typically a driver bug or race condition. The DRA driver should either restore the device to the ResourceSlice, or the scheduler/allocator should deallocate the claim. The status controller does NOT fix this - only reports it.

2. **ResourceClaim deleted but status not yet updated:**
   - Normal eventual consistency - status will catch up on next reconcile
   - DeviceStatus will change from showing the allocation to showing device as available
   - No special handling needed beyond normal reconciliation latency

3. **New device added to ResourceSlice spec:**
   - Status controller creates DeviceStatus with the device shown as available (assuming no claims reference it yet)
   - Normal operation

4. **ResourceSlice spec updated during status write (race condition):**
   - Use optimistic locking via `resourceVersion`
   - On conflict, controller re-fetches fresh spec and retries
   - Standard Kubernetes controller pattern

5. **Pool generation changes (driver updates pool):**
   - Status controller only processes slices at the current generation
   - Old-generation slices are ignored until driver updates them
   - Prevents mixing old and new pool configurations in validation

**Controller responsibilities vs. other components:**

| Scenario | Status Controller | DRA Driver | Scheduler/Allocator |
|----------|------------------|------------|---------------------|
| Device removed with active allocation | Reports error | Should restore device or signal removal | Should deallocate orphaned claims |
| Stale allocation data | Updates on next reconcile | N/A | N/A |
| Cross-slice validation errors | Reports in PoolStatus | Should fix ResourceSlice definitions | N/A |
| Spec/status conflict | Retries with fresh data | N/A | N/A |

### kubectl Integration

#### kubectl describe resourceslice

Enhanced output will include status information:

```
Name:         node-1-gpu-slice-1
Namespace:
Labels:       <none>
Annotations:  <none>
API Version:  resource.k8s.io/v1beta2
Kind:         ResourceSlice

Spec:
  Driver:  example.com/gpu
  Pool:
    Name:                  node-1
    Generation:            5
    Resource Slice Count:  1
  Node Name:              node-1
  Devices:
    Name:  gpu-0
    Attributes:
      Model:   A100
      Memory:  40Gi
    Name:  gpu-1
    Attributes:
      Model:   A100
      Memory:  40Gi

Status:
  Observed Generation:  5
  Summary:
    Total Devices:       2
    Allocated Devices:   1
    Available Devices:   1
  Devices:
    Name:  gpu-0
    State: Allocated
    Allocations:
      Claim Namespace:  default
      Claim Name:       ml-training-gpu
      Claim UID:        abc-123
    Name:  gpu-1
    State: Available

Events:  <none>
```

#### kubectl describe node

For node-local devices, add a DRA Resources section:

```
Name:               node-1
...

DRA Resources:
  Driver:  example.com/gpu
  Pool:    node-1
  Summary:
    Total Devices:       2
    Allocated Devices:   1
    Available Devices:   1
  Devices:
    gpu-0:
      Status:       Allocated
      Allocated To: default/ml-training-gpu
      Attributes:
        Model:      A100
        Memory:     40Gi
    gpu-1:
      Status:       Available
      Attributes:
        Model:      A100
        Memory:     40Gi
```

**Implementation:**
- kubectl will query ResourceSlices with `spec.nodeName=<node-name>`
- Display status information from ResourceSlice.Status
- Handle multiple drivers and pools on the same node

### Test Plan

#### Prerequisite testing updates

None required.

#### Unit tests

Coverage targets for Alpha:
- `k8s.io/kubernetes/pkg/controller/resourceslicestatus`: 80%+
  - Building device status from claims
  - Calculating summaries
  - Cross-slice validation logic
  - Indexing and querying

- `k8s.io/kubectl/pkg/describe`: 75%+
  - ResourceSlice describe formatting
  - Node describe DRA section formatting

Test cases:
- Single device allocation/deallocation
- Multi-allocatable device with partial allocation
- Pool spanning multiple ResourceSlices
- Cross-slice validation (duplicate devices, missing slices)
- Generation changes
- Device attribute changes
- Empty ResourceSlices

#### Integration tests

Integration tests will verify:
1. Controller correctly updates ResourceSlice status when ResourceClaims are created/updated/deleted
2. Status reflects allocations from multiple ResourceClaims
3. Cross-slice validation detects errors
4. Status subresource works correctly (status updates don't increment spec generation)
5. Eventually consistent updates converge to correct state
6. Rate limiting prevents API server overload
7. Multi-allocatable device capacity tracking

Test scenarios:
- Create ResourceSlice → verify empty status
- Create ResourceClaim → verify ResourceSlice status updates
- Delete ResourceClaim → verify status cleared
- Multiple claims for multi-allocatable device → verify capacity tracking
- Pool with multiple slices → verify cross-slice validation
- Generation change → verify status updates for new generation

#### e2e tests

E2E tests will verify end-to-end functionality:
1. Deploy DRA driver that publishes ResourceSlices
2. Create ResourceClaim
3. Verify ResourceSlice status shows allocation via kubectl
4. Verify node description shows DRA resources
5. Delete ResourceClaim
6. Verify status updated to show resource available

Test with:
- Node-local devices
- Network-attached devices
- Single-allocation devices
- Multi-allocatable devices (consumable capacity)

### Graduation Criteria

#### Alpha

- Feature implemented behind `DRAResourceAvailabilityVisibility` feature gate
- ResourceSlice Status API added to v1beta2 (or v1beta3 if API version bumps)
- ResourceSlice status controller implemented in kube-controller-manager
- kubectl describe enhancements implemented
- Unit and integration tests completed
- Documentation for API and kubectl usage

#### Beta

- Feature enabled by default
- E2E tests in Testgrid and passing
- At least one DRA driver using and validating the status information
- Performance testing with large numbers of ResourceSlices and ResourceClaims
- Metrics for monitoring controller performance
- User feedback incorporated
- Cross-slice validation proven stable

**Potential Beta enhancements (based on user feedback):**
- Vendor-defined summarization attributes: Allow DeviceClass or ResourceSlice to specify which device attributes should be highlighted in summaries (e.g., NVIDIA MIG partition types)
- Synthetic `kubectl describe resourcepool` command for aggregated pool-level views

#### GA

- At least 2 releases as beta
- Multiple DRA drivers using the feature in production
- Performance validated at scale (1000+ devices, 10000+ claims)
- No major bugs or design issues reported
- Documentation complete and accurate
- Conformance tests if applicable

### Upgrade / Downgrade Strategy

**Upgrade:**
- When feature gate is enabled, controller starts populating status
- Existing ResourceSlices get status populated asynchronously
- No impact on existing allocations or scheduling
- kubectl gracefully handles missing status (pre-upgrade ResourceSlices)

**Downgrade:**
- When feature gate is disabled, controller stops updating status
- Status field may contain stale data but is ignored
- No impact on scheduling (scheduler doesn't depend on status)
- kubectl gracefully handles missing status

**API compatibility:**
- Status is a new optional field, safe to add
- Old clients ignore status
- New clients handle missing status gracefully

### Version Skew Strategy

**Control plane components:**
- kube-controller-manager: Runs the status controller when feature gate enabled
- kube-apiserver: Serves ResourceSlice with status subresource
- kube-scheduler: Does not depend on status (only uses spec)

**Skew scenarios:**
1. **New apiserver, old controller-manager**: Status field exists but not populated (graceful degradation)
2. **Old apiserver, new controller-manager**: Controller cannot update status (feature disabled)
3. **Multiple controller-managers at different versions**: Leader election ensures only one updates status

**Node components:**
- kubelet and DRA drivers are not affected (they don't use ResourceSlice status)

**kubectl:**
- New kubectl with old cluster: Status section empty/omitted
- Old kubectl with new cluster: Status field ignored

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: DRAResourceAvailabilityVisibility
  - Components depending on the feature gate:
    - kube-controller-manager (for ResourceSlice status controller)
    - kubectl (for enhanced describe output)

###### Does enabling the feature change any default behavior?

No. This feature only adds status information to ResourceSlice objects. It does not change:
- How ResourceSlices are created or managed by drivers
- How ResourceClaims are allocated
- How pods are scheduled
- Any existing API fields or behaviors

Users will see new status information when describing ResourceSlices or Nodes, but this is purely additive.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate will:
- Stop the ResourceSlice status controller from running
- Stop kubectl from displaying status information

The status field in ResourceSlice objects will remain but will not be updated. This is safe because:
- Scheduling does not depend on status (only spec)
- DRA drivers do not depend on status
- Old status data is harmless (just stale)

No manual cleanup is required.

###### What happens if we reenable the feature if it was previously rolled back?

The controller will resume and repopulate status from current ResourceClaims. Since the controller watches both ResourceSlices and ResourceClaims, it will:
1. Detect ResourceSlices with stale or missing status
2. Query current ResourceClaims
3. Update status to reflect current state

This typically happens within a few seconds for each ResourceSlice.

###### Are there any tests for feature enablement/disablement?

Yes, integration tests will verify:
- Feature gate disabled: status controller does not run, status not updated
- Feature gate enabled: status controller runs and populates status
- Toggle: disable → enable → status repopulated correctly
- Status updates don't affect spec or scheduling

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

**Rollout failure scenarios:**
1. **Controller bugs**: If the status controller has bugs, it might:
   - Fail to update status correctly (user-facing only, no pod impact)
   - Generate excessive API calls (could load API server)
   - Panic/crash-loop (controller-manager has other controllers, limited blast radius)

2. **API server load**: Many status updates could increase API server load
   - Mitigation: Rate limiting, batching, efficient indexing

**Impact on workloads:**
- **None for existing pods**: Status is read-only information, doesn't affect running pods
- **None for scheduling**: Scheduler uses ResourceSlice spec, not status
- **None for allocation**: Allocation is based on ResourceClaims and ResourceSlice spec

**Rollback scenarios:**
- Disable feature gate → controller stops → status becomes stale
- No impact on workloads (status is informational only)

###### What specific metrics should inform a rollback?

Metrics to monitor:
- `dra_resourceslice_status_update_errors_total` - High error rate indicates controller issues
- `dra_resourceslice_status_update_duration_seconds` - High latency indicates performance issues
- `apiserver_request_duration_seconds{resource="resourceslices",subresource="status"}` - API server latency
- `workqueue_depth{name="resourceslice_status"}` - Queue backup indicates controller falling behind

Rollback triggers:
- Error rate > 5% for sustained period (> 10 minutes)
- P99 status update latency > 5 seconds
- Work queue depth growing unbounded (> 1000 items)

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be tested in integration tests:
- Start with feature disabled
- Enable feature → verify status populates
- Disable feature → verify controller stops updating
- Re-enable feature → verify status repopulates

Will be tested manually during development with test clusters.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No. This is a purely additive feature.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The feature provides visibility, not functionality used by workloads. Operators can determine if the feature is enabled by:
1. Check feature gate: `kubectl get --raw /api/v1 | jq '.features'`
2. Check if ResourceSlices have status populated: `kubectl get resourceslice <name> -o jsonpath='{.status}'`
3. Check controller logs for status controller initialization

###### How can someone using this feature know that it is working for their instance?

- [x] API .status
  - Other field: ResourceSlice.Status should be populated with device statuses and summary
  - Users can run `kubectl describe resourceslice <name>` and see status information
  - Status should reflect current ResourceClaim allocations

Validation steps:
1. Create a ResourceClaim that allocates a device
2. Run `kubectl describe resourceslice` for the slice containing that device
3. Verify status shows the device as allocated to the claim
4. Delete the ResourceClaim
5. Verify status updates to show device as available

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- **Status update latency**: 95% of ResourceSlice status updates complete within 5 seconds of a ResourceClaim change
- **Status accuracy**: 99% of status queries return data consistent with actual allocations (within eventual consistency window)
- **Controller availability**: Status controller uptime > 99.9% (same as other kube-controller-manager controllers)

These are reasonable because:
- Status is eventually consistent, brief delays are acceptable
- Status is informational, not critical for cluster operation
- Controller runs in kube-controller-manager with standard HA setup

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `dra_resourceslice_status_update_duration_seconds`
  - Aggregation method: Histogram (P50, P95, P99)
  - Components exposing the metric: kube-controller-manager

  - Metric name: `dra_resourceslice_status_update_errors_total`
  - Aggregation method: Counter (rate)
  - Components exposing the metric: kube-controller-manager

  - Metric name: `dra_resourceslice_status_controller_sync_total`
  - Aggregation method: Counter (rate)
  - Components exposing the metric: kube-controller-manager

  - Metric name: `workqueue_depth{name="resourceslice_status"}`
  - Aggregation method: Gauge
  - Components exposing the metric: kube-controller-manager

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

For Beta, consider adding:
- `dra_resourceslice_status_staleness_seconds`: Time since last successful status update per ResourceSlice
- `dra_resourceslice_claims_tracked_total`: Number of ResourceClaims being tracked for status updates
- `dra_pool_validation_errors_total`: Cross-slice validation errors by pool

These would help operators understand:
- Whether status is staying current
- Scale of objects being tracked
- Pool consistency issues

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No external services required. Dependencies are only on standard Kubernetes components:
- kube-apiserver: To store ResourceSlice status
- kube-controller-manager: To run the status controller
- DRA drivers: Must create ResourceSlices (but this is already required for DRA)

The feature enhances existing DRA functionality and does not introduce new external dependencies.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes, the ResourceSlice status controller will make API calls:

- **List ResourceSlices**: Once on startup, then watch
  - Estimated throughput: 1 initial list + ongoing watch events
  - Originating component: ResourceSlice status controller in kube-controller-manager

- **List ResourceClaims**: Once on startup, then watch
  - Estimated throughput: 1 initial list + ongoing watch events
  - Originating component: ResourceSlice status controller

- **Update ResourceSlice status**: When allocations change
  - Estimated throughput: Proportional to ResourceClaim churn rate
  - Originating component: ResourceSlice status controller
  - Typical: Few updates per minute in steady state
  - Worst case: O(ResourceSlices) updates when many claims created/deleted simultaneously

Mitigation:
- Use informers (watch, not polling)
- Batch status updates (coalesce multiple changes)
- Rate limiting on work queue

###### Will enabling / using this feature result in introducing new API types?

No. This feature adds a Status field to the existing ResourceSlice type.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No. This feature is purely Kubernetes API operations.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, ResourceSlice objects will have a Status field added:

- **API type**: ResourceSlice
- **Estimated increase in size**:
  - Minimum: ~200 bytes (empty status with metadata)
  - Per device: ~150-300 bytes (device status, conditions, allocations)
  - Per allocation: ~100 bytes (claim reference, capacity)
  - Example: ResourceSlice with 4 devices, 2 allocated = ~1-1.5KB status

- **Estimated amount of new objects**: None (adding field to existing objects)

For a cluster with 100 nodes, 4 devices per node (400 total ResourceSlices):
- Status overhead: 400-600 KB total
- This is negligible compared to typical etcd size

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No significant impact expected:

- **List ResourceSlices**: Minimal increase due to status field (few KB per slice)
- **Scheduling**: No impact (scheduler doesn't use status)
- **Pod startup**: No impact (status is informational)

Status updates use the status subresource, so they don't trigger spec watches or unnecessary reconciliation.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

**kube-controller-manager:**
- CPU: Low increase for status controller reconciliation
  - Estimate: +1-5% CPU in steady state
  - Spike: +10-20% CPU during mass ResourceClaim changes
- RAM: Moderate increase for informer caches
  - ResourceSlice informer: ~1-2 MB per 100 ResourceSlices
  - ResourceClaim informer: ~5-10 MB per 1000 ResourceClaims
  - Total estimate: +10-50 MB for typical clusters
- Network: Additional status update API calls (minimal, few KB/s)

**etcd:**
- Storage: Status field adds to ResourceSlice size
  - Estimate: +400-600 KB for cluster with 400 devices (negligible)

**kube-apiserver:**
- CPU: Minimal increase for serving status subresource
- RAM: Minimal increase (status is part of ResourceSlice)

These increases are acceptable given the value provided.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. This feature runs in kube-controller-manager (control plane) and does not:
- Create new processes
- Open additional sockets (uses existing API client)
- Create files
- Affect nodes or node-level components

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

**API server unavailable:**
- Controller cannot update status (expected behavior)
- Controller will retry with exponential backoff
- When API server recovers, controller resumes and updates status
- No impact on existing workloads (status is informational)

**etcd unavailable:**
- API server is also unavailable (same as above)
- No special handling needed

The controller uses standard Kubernetes client patterns with retries and backoff.

###### What are other known failure modes?

- **[Status becomes stale]**
  - Detection: `dra_resourceslice_status_staleness_seconds` metric (if implemented), or check `status.observedGeneration` vs `metadata.generation`
  - Mitigations: Check controller logs, restart controller-manager, verify feature gate enabled
  - Diagnostics: Controller logs with level 2+ will show reconciliation activity
  - Testing: Integration tests verify status updates on ResourceClaim changes

- **[Cross-slice validation errors]**
  - Detection: `status.poolStatus.validationErrors` field populated
  - Mitigations: DRA driver should fix ResourceSlice definitions (duplicate device names, incorrect slice counts)
  - Diagnostics: `kubectl describe resourceslice` shows validation errors
  - Testing: Integration tests verify validation logic

- **[Controller falling behind]**
  - Detection: `workqueue_depth` metric increasing, status update latency increasing
  - Mitigations: Scale up controller-manager, investigate ResourceClaim churn rate
  - Diagnostics: Controller logs show queue depth and processing rate
  - Testing: Scale tests with many ResourceClaims

###### What steps should be taken if SLOs are not being met to determine the problem?

1. **Check controller health:**
   ```bash
   # Check if controller-manager is running
   kubectl get pods -n kube-system -l component=kube-controller-manager

   # Check controller logs for errors
   kubectl logs -n kube-system <controller-manager-pod> | grep resourceslice
   ```

2. **Check metrics:**
   ```promql
   # Status update error rate
   rate(dra_resourceslice_status_update_errors_total[5m])

   # Status update latency
   histogram_quantile(0.99, dra_resourceslice_status_update_duration_seconds)

   # Work queue depth
   workqueue_depth{name="resourceslice_status"}
   ```

3. **Verify feature gate:**
   ```bash
   kubectl get --raw /api/v1 | jq '.features' | grep DRAResourceAvailabilityVisibility
   ```

4. **Check API server health:**
   - High API server latency can slow status updates
   - Check `apiserver_request_duration_seconds`

5. **Manual verification:**
   ```bash
   # Check if status is being populated
   kubectl get resourceslices -o custom-columns=NAME:.metadata.name,STATUS:.status.summary

   # Compare with actual ResourceClaims
   kubectl get resourceclaims -A
   ```

## Implementation History

- 2025-12-20: KEP created in provisional state

## Security Considerations

### Cross-Namespace Information Exposure

ResourceSlice is a cluster-scoped resource, but ResourceClaims are namespaced. The status field contains references to ResourceClaims (namespace/name) which could expose information about claims in namespaces the user cannot access.

**Mitigation: kubectl-side RBAC filtering**

- ResourceSlice.Status stores full allocation details including claim namespace/name
- kubectl checks the user's RBAC permissions before displaying claim references
- For claims in namespaces the user cannot access, kubectl elides the reference:
  - Full access: `Allocated to team-a/gpu-claim-1`
  - Restricted: `Allocated (1 claim)`
- Users always see device states and availability counts (the primary use case)
- Only claim references are filtered based on RBAC

**Why this approach:**
- Keeps API simple (single status field, no subresources)
- Primary use case (checking availability) works for all users
- Admins with full RBAC access get complete debugging information
- No additional API calls required
- Follows existing kubectl patterns for RBAC-aware display

**Alternative considered:** A separate `resourceslices/allocations` subresource with independent RBAC. This is architecturally cleaner but adds API complexity for a display-only concern.

## Drawbacks

1. **API server storage overhead**: Every ResourceSlice gains a status field, increasing etcd storage
   - Mitigation: Status is relatively small (1-2 KB per slice)

2. **Eventual consistency complexity**: Users need to understand that status may lag behind actual allocations
   - Mitigation: Clear documentation, use generation numbers to detect staleness

3. **Controller complexity**: Cross-slice validation and efficient status updates add complexity
   - Mitigation: Thorough testing, clear code structure, good observability

4. **Security consideration**: Status reveals which ResourceClaims are allocated, potentially exposing namespace information
   - Mitigation: kubectl-side RBAC filtering elides claim references for unauthorized users (see [Security Considerations](#security-considerations))

## Alternatives

### Alternative 1: Status in ResourceClaim instead of ResourceSlice

One approach considered was adding status to ResourceClaim objects to show total pool capacity and availability. While this would fit naturally with the existing ResourceClaim model and wouldn't require a new controller, it fails to address the fundamental visibility problem. Since ResourceClaims are namespaced, capacity information would either be duplicated across all namespaces or require cross-namespace references, both of which are problematic. More importantly, users cannot see available resources before creating a claim, which is the core use case this KEP addresses. This alternative was rejected because it doesn't solve the cross-namespace visibility problem that motivated this enhancement.

### Alternative 2: New cluster-scoped ResourcePool object

Another option was to create a new ResourcePool CRD that aggregates information across ResourceSlices. This would provide a clean separation of concerns and could include additional pool-level metadata. However, introducing a new API type adds unnecessary complexity to the user experience, requiring both ResourceSlice and ResourcePool objects to be understood and managed. DRA drivers would need to manage both types, and this doesn't integrate naturally with the existing ResourceSlice API that drivers already publish. The ResourceSlice status approach achieves the same goals with less API surface and better integration with existing patterns. This alternative was rejected as unnecessarily complex when ResourceSlice status is sufficient.

### Alternative 3: Metrics-based visibility only

We considered exposing device availability only through metrics rather than API status. This would avoid any API changes and leverage existing monitoring infrastructure. However, metrics are not queryable via kubectl, which is the primary tool administrators use for cluster inspection. Additionally, metrics may not be available in all clusters, and they don't provide the API-level visibility needed to correlate availability with specific ResourceClaims. This approach would create a poor user experience for the primary use cases of debugging and capacity planning. This alternative was rejected because it doesn't meet the user experience goals that require API-level queryability through kubectl.

### Alternative 4: Aggregated API server for DRA

Building a separate aggregated API server specifically for DRA information was considered. While this would be very flexible and could provide advanced querying capabilities, it requires massive engineering effort to build and maintain a separate component. Clusters would need to deploy and manage this additional API server, adding operational complexity. For the relatively straightforward use case of displaying resource availability, this solution is vastly over-engineered. This alternative was rejected as disproportionate to the actual need, which can be solved with a simple status controller.

### Alternative 5: External controller with custom CRD

The final alternative considered was allowing DRA drivers to create their own status CRDs for reporting availability. This would be flexible for driver-specific needs and require no changes to core Kubernetes APIs. However, it would result in inconsistent experiences across different drivers, with each potentially using different status formats and query patterns. There would be no standard kubectl integration, forcing users to learn different APIs for each driver they use. This also doesn't solve cross-driver visibility, as each driver's status would be isolated. This alternative was rejected because it creates an inconsistent user experience and fails to provide a standardized solution for the DRA ecosystem.

## Infrastructure Needed

No special infrastructure required. Implementation uses:
- Existing Kubernetes API infrastructure
- Existing kube-controller-manager framework
- Existing kubectl describe framework
- Standard e2e test infrastructure

