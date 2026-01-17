# KEP-5677: DRA Resource Availability Visibility

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Part 1: ResourcePool Object (API-visible)](#part-1-resourcepool-object-api-visible)
  - [Part 2: Client-Side Utility (kubectl)](#part-2-client-side-utility-kubectl)
  - [User Stories](#user-stories)
    - [Story 1: Cluster Administrator Monitoring Resources](#story-1-cluster-administrator-monitoring-resources)
    - [Story 2: Developer Debugging Resource Allocation](#story-2-developer-debugging-resource-allocation)
    - [Story 3: Capacity Planning](#story-3-capacity-planning)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Security Considerations](#security-considerations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
    - [ResourcePool Object](#resourcepool-object)
    - [ResourcePool Status Fields](#resourcepool-status-fields)
  - [Controller Implementation](#controller-implementation)
    - [ResourcePool Controller](#resourcepool-controller)
    - [Cross-Slice Validation](#cross-slice-validation)
  - [Client-Side Utility Library](#client-side-utility-library)
  - [kubectl Integration](#kubectl-integration)
    - [kubectl describe resourcepool](#kubectl-describe-resourcepool)
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
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Alternative 1: Status in ResourceSlice](#alternative-1-status-in-resourceslice)
  - [Alternative 2: Status in ResourceClaim](#alternative-2-status-in-resourceclaim)
  - [Alternative 3: Metrics-based visibility only](#alternative-3-metrics-based-visibility-only)
  - [Alternative 4: Client-side only (no API changes)](#alternative-4-client-side-only-no-api-changes)
  - [Alternative 5: Custom apiserver endpoint with report generator](#alternative-5-custom-apiserver-endpoint-with-report-generator)
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

This KEP addresses a critical visibility gap in Dynamic Resource Allocation (DRA) by enabling users to view available device capacity across resource pools. While ResourceSlices store capacity data and ResourceClaims track consumption, there is currently no straightforward way for users to view the available capacity remaining in a pool or on a node.

This enhancement introduces:
1. A new **ResourcePool** cluster-scoped object that aggregates availability information across ResourceSlices
2. A **ResourcePool controller** that watches ResourceSlices and ResourceClaims to maintain pool-level summaries
3. A **client-side utility library** for detailed allocation debugging (used by kubectl)
4. Enhanced **kubectl describe** commands for ResourcePool and Node to display availability information

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

- Introduce a new **ResourcePool** object that provides pool-level availability summaries
- Implement a **ResourcePool controller** that aggregates information from ResourceSlices and ResourceClaims
- Provide **cross-slice validation** to surface pool consistency issues (e.g., duplicate device names, missing slices)
- Create a **client-side utility library** for detailed allocation debugging
- Enhance **kubectl describe resourcepool** to show capacity, allocated, and available resources
- Enhance **kubectl describe node** to show DRA resource information for node-local resources
- Ensure the solution works for both node-local and network-attached devices
- Support single-device, consumable capacity (multi-allocatable), and partitionable devices

### Non-Goals

- Modifying the core scheduling algorithm (this is purely about visibility)
- Changing how ResourceClaims are allocated
- Adding real-time metrics/monitoring (this is API-level status, not metrics)
- Implementing quotas or limits based on availability (future work)
- **Modifying ResourceSlice or ResourceClaim APIs** (no status added to these objects)
- Providing historical consumption data (this is point-in-time status)
- Exposing individual claim references in the API (kept client-side for RBAC reasons)

## Proposal

This KEP proposes a two-part solution:

### Part 1: ResourcePool Object (API-visible)

A new **ResourcePool** cluster-scoped object that provides:
- **Summary counts**: total, allocated, available, unavailable devices
- **Pool validation status**: conditions indicating pool health
- **Cross-slice validation errors**: issues like duplicate device names

ResourcePool contains **only summaries** - no individual claim references. This keeps the object:
- Constant size (O(1), not O(claims))
- Free of RBAC/permission issues (no namespace-scoped data exposed)
- Lightweight for API server and etcd

### Part 2: Client-Side Utility (kubectl)

A **utility library** that takes ResourceSlices + ResourceClaims and computes:
- Detailed per-device allocation status
- Which claims are using which devices
- Validation issues

This is used by `kubectl describe` to show detailed information **when the user has permission** to fetch the relevant ResourceClaims. Users without permission still see the ResourcePool summary.

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
$ kubectl get resourcepools
NAME                      DRIVER            TOTAL   ALLOCATED   AVAILABLE
example.com-gpu.node-1    example.com/gpu   4       3           1
example.com-gpu.node-2    example.com/gpu   4       1           3

$ kubectl describe resourcepool example.com-gpu.node-1
Name:         example.com-gpu.node-1
Driver:       example.com/gpu
Pool:         node-1
Node:         node-1

Status:
  Summary:
    Total Devices:       4
    Allocated Devices:   3
    Available Devices:   1
    Unavailable Devices: 0
  Conditions:
    Type: Complete   Status: True   Reason: AllSlicesPresent
    Type: Valid      Status: True   Reason: ValidationPassed
  Observed Slice Count:   1
  Expected Slice Count:   1

# With admin permissions, kubectl fetches claims and shows details:
Device Details (from ResourceSlices + ResourceClaims):
  gpu-0:  Allocated  -> default/ml-training-claim
  gpu-1:  Allocated  -> default/ml-inference-claim
  gpu-2:  Allocated  -> team-a/batch-job-claim
  gpu-3:  Available
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
$ kubectl describe resourcepool example.com-gpu.node-1
Name:         example.com-gpu.node-1
...
Status:
  Summary:
    Total Devices:       4
    Allocated Devices:   4
    Available Devices:   0   # <-- No GPUs available on this node!

# Developer can see node-1 is fully allocated, try another node
$ kubectl describe resourcepool example.com-gpu.node-2
Status:
  Summary:
    Total Devices:       4
    Allocated Devices:   1
    Available Devices:   3   # <-- GPUs available here
```

#### Story 3: Capacity Planning

As a capacity planner, I want to query resource availability across the cluster to:
- Identify nodes or pools that are under-utilized
- Understand trends in resource consumption
- Plan for workload placement

**Proposed experience:**
```bash
$ kubectl get resourcepools -o custom-columns=\
NAME:.metadata.name,\
DRIVER:.spec.driver,\
TOTAL:.status.summary.totalDevices,\
ALLOCATED:.status.summary.allocatedDevices,\
AVAILABLE:.status.summary.availableDevices

NAME                      DRIVER            TOTAL   ALLOCATED   AVAILABLE
example.com-gpu.node-1    example.com/gpu   4       3           1
example.com-gpu.node-2    example.com/gpu   4       1           3
example.com-gpu.node-3    example.com/gpu   4       4           0
```

### Notes/Constraints/Caveats

1. **Eventually consistent**: The ResourcePool status is eventually consistent. There may be a brief period where the status doesn't reflect very recent allocations.

2. **Summary only in API**: Individual claim references are NOT stored in ResourcePool. This avoids RBAC issues and keeps the object small. Detailed allocation info is computed client-side by kubectl.

3. **User permissions for details**: `kubectl describe resourcepool` shows detailed allocation info only if the user has permission to list ResourceClaims in the relevant namespaces. Otherwise, only the summary is shown.

4. **Multi-allocatable devices**: For devices that support consumable capacity (KEP-5075), the summary tracks device counts, not capacity details. Capacity details are available via kubectl's client-side computation.

5. **Partitionable devices**: Devices may be "Unavailable" (not allocated but cannot accept allocations due to shared resource constraints). The summary includes an `unavailableDevices` count.

### Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| ResourcePool controller overhead | Additional controller watching slices/claims | Efficient informers with indexing; rate limiting |
| Stale ResourcePool status | Users may see outdated availability | Document eventual consistency; use observedGeneration |
| Version skew for kubectl utility | Old kubectl may not compute correctly | Document requirement to use kubectl >= cluster version |
| Many ResourcePools in large clusters | API server load for listing | ResourcePools are O(pools), not O(devices) - manageable |

### Security Considerations

**RBAC and Information Exposure:**

1. **ResourcePool is cluster-scoped**: Any user with permission to read ResourcePool objects can see summary counts (total, allocated, available) for all pools. This reveals aggregate resource utilization but not which namespaces or workloads are consuming resources.

2. **No claim references in API**: Individual ResourceClaim namespace/name pairs are NOT stored in ResourcePool. This prevents information leakage where a user could discover claim names in namespaces they don't have access to.

3. **Client-side claim resolution**: When kubectl fetches detailed allocation info, it only shows claims the user has permission to read. Users without cross-namespace ResourceClaim access see only the summary from ResourcePool.

4. **Graceful degradation**: If a user lacks permission to list ResourceClaims in certain namespaces, kubectl shows:
   - Devices allocated to accessible claims: full details
   - Devices allocated to inaccessible claims: shown as "Allocated" without claim reference
   - This preserves the count accuracy while respecting RBAC boundaries

**Threat Model:**

| Threat | Mitigation |
|--------|------------|
| User discovers claim names in other namespaces | Claim references are client-side only, respecting user's RBAC |
| User infers workload patterns from allocation counts | Summary counts are intentionally coarse; acceptable for cluster-scoped visibility |
| Malicious driver creates misleading ResourcePool | ResourcePool is controller-managed, not driver-managed |

## Design Details

### API Changes

#### ResourcePool Object

A new cluster-scoped resource in the `resource.k8s.io` API group:

```go
// ResourcePool provides aggregated availability information for a pool of resources.
// ResourcePool objects are created and maintained by the ResourcePool controller,
// not by users or DRA drivers.
type ResourcePool struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    // Spec identifies the pool this object represents.
    // This is immutable after creation.
    Spec ResourcePoolSpec `json:"spec"`

    // Status contains the current availability and validation status.
    // +optional
    Status ResourcePoolStatus `json:"status,omitempty"`
}

// ResourcePoolSpec identifies a resource pool.
type ResourcePoolSpec struct {
    // Driver is the name of the DRA driver that manages this pool.
    // +required
    Driver string `json:"driver"`

    // PoolName is the name of the pool as specified in ResourceSlice.Spec.Pool.Name.
    // +required
    PoolName string `json:"poolName"`

    // NodeName is set if this pool is associated with a specific node.
    // +optional
    NodeName string `json:"nodeName,omitempty"`
}

// ResourcePoolStatus describes the current state of a resource pool.
type ResourcePoolStatus struct {
    // Summary provides aggregate device counts for this pool.
    // +optional
    Summary ResourcePoolSummary `json:"summary,omitempty"`

    // Conditions represent observations about the pool's state.
    // +optional
    // +listType=map
    // +listMapKey=type
    Conditions []metav1.Condition `json:"conditions,omitempty"`

    // ValidationErrors contains errors found during cross-slice validation.
    // Limited to the first 10 errors.
    // +optional
    // +listType=atomic
    ValidationErrors []string `json:"validationErrors,omitempty"`

    // TruncatedErrorCount is the total count when ValidationErrors is truncated.
    // Zero if not truncated.
    // +optional
    TruncatedErrorCount int32 `json:"truncatedErrorCount,omitempty"`

    // ObservedSliceCount is the number of ResourceSlices observed for this pool.
    // +optional
    ObservedSliceCount int32 `json:"observedSliceCount,omitempty"`

    // ExpectedSliceCount is the expected number of ResourceSlices
    // (from ResourceSlice.Spec.Pool.ResourceSliceCount).
    // +optional
    ExpectedSliceCount int32 `json:"expectedSliceCount,omitempty"`

    // ObservedGeneration is the pool generation this status corresponds to.
    // +optional
    ObservedGeneration int64 `json:"observedGeneration,omitempty"`

    // LastUpdateTime is when the status was last updated.
    // +optional
    LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}

// ResourcePoolSummary provides aggregate information about pool resources.
type ResourcePoolSummary struct {
    // TotalDevices is the total number of devices across all ResourceSlices in this pool.
    TotalDevices int32 `json:"totalDevices"`

    // AllocatedDevices is the number of devices that have at least one allocation.
    AllocatedDevices int32 `json:"allocatedDevices"`

    // AvailableDevices is the number of devices that are fully available for allocation.
    AvailableDevices int32 `json:"availableDevices"`

    // UnavailableDevices is the number of devices that cannot accept allocations
    // due to constraints (e.g., partitionable device shared resource exhaustion).
    // +optional
    UnavailableDevices int32 `json:"unavailableDevices,omitempty"`

    // PartiallyAllocatedDevices is the number of multi-allocatable devices
    // that have some but not all capacity allocated (consumable capacity).
    // +optional
    PartiallyAllocatedDevices int32 `json:"partiallyAllocatedDevices,omitempty"`
}
```

**Key design decisions:**
- **No claim references**: Individual claim namespace/name are NOT stored. This avoids RBAC issues and keeps size O(1).
- **Summary only**: Counts provide visibility without exposing sensitive allocation details.
- **Controller-managed**: Users don't create ResourcePools; the controller does.
- **Immutable spec**: The spec identifies the pool and doesn't change.

#### ResourcePool Status Fields

| Field | Description |
|-------|-------------|
| `summary.totalDevices` | Total devices in all ResourceSlices for this pool |
| `summary.allocatedDevices` | Devices with at least one allocation |
| `summary.availableDevices` | Devices fully available |
| `summary.unavailableDevices` | Devices unavailable due to constraints (partitionable) |
| `summary.partiallyAllocatedDevices` | Devices with partial capacity used (consumable) |
| `conditions[Complete]` | All expected ResourceSlices are present |
| `conditions[Valid]` | Cross-slice validation passed |
| `validationErrors` | Specific validation issues (max 10) |
| `observedSliceCount` | Number of slices found |
| `expectedSliceCount` | Number of slices expected |

### Controller Implementation

#### ResourcePool Controller

A new controller in kube-controller-manager maintains ResourcePool objects:

**Responsibilities:**
1. Watch ResourceSlices - detect new pools, pool changes
2. Watch ResourceClaims - detect allocation changes
3. Create ResourcePool objects when new pools appear
4. Update ResourcePool status with current summary and validation
5. Delete ResourcePool objects when all slices for a pool are gone

**Implementation:**
```go
type ResourcePoolController struct {
    resourcePoolClient  resourcev1.ResourcePoolInterface
    resourceSliceLister listers.ResourceSliceLister
    resourceClaimLister listers.ResourceClaimLister

    // Index: pool key -> slice names
    slicesByPool cache.Indexer
    // Index: driver+pool -> claim names
    claimsByPool cache.Indexer

    workqueue workqueue.RateLimitingInterface
}

func (c *ResourcePoolController) Reconcile(ctx context.Context, poolKey string) error {
    driver, poolName := parsePoolKey(poolKey)

    // Get all ResourceSlices for this pool
    slices := c.slicesByPool.ByIndex("pool", poolKey)
    if len(slices) == 0 {
        // Pool gone - delete ResourcePool
        return c.deleteResourcePool(ctx, poolKey)
    }

    // Get all ResourceClaims that allocate from this pool
    claims := c.claimsByPool.ByIndex("pool", poolKey)

    // Build device allocation map
    allocations := c.buildAllocationMap(slices, claims)

    // Calculate summary
    summary := c.calculateSummary(slices, allocations)

    // Validate cross-slice consistency
    validationErrors := c.validatePool(slices)

    // Build conditions
    conditions := c.buildConditions(slices, validationErrors)

    // Create or update ResourcePool
    return c.updateResourcePool(ctx, driver, poolName, slices, summary, conditions, validationErrors)
}

func (c *ResourcePoolController) calculateSummary(
    slices []*ResourceSlice,
    allocations map[string][]AllocationInfo,
) ResourcePoolSummary {
    var total, allocated, available, unavailable, partial int32

    for _, slice := range slices {
        for _, device := range slice.Spec.Devices {
            total++
            allocs := allocations[device.Name]

            switch {
            case len(allocs) == 0 && !isUnavailable(device, slice):
                available++
            case len(allocs) == 0 && isUnavailable(device, slice):
                unavailable++
            case isPartiallyAllocated(device, allocs):
                partial++
                allocated++ // Also counted as allocated
            default:
                allocated++
            }
        }
    }

    return ResourcePoolSummary{
        TotalDevices:              total,
        AllocatedDevices:          allocated,
        AvailableDevices:          available,
        UnavailableDevices:        unavailable,
        PartiallyAllocatedDevices: partial,
    }
}
```

**ResourcePool naming convention:**
- Name: `<sanitized-driver>.<pool-name>`
- Example: `example.com-gpu.node-1` for driver `example.com/gpu`, pool `node-1`
- Driver name sanitized: `/` → `-`, must be valid DNS subdomain

#### Cross-Slice Validation

The controller validates consistency across ResourceSlices in a pool:

1. **Device name uniqueness**: No device name appears in multiple slices
2. **Generation consistency**: All slices should have the same pool generation
3. **Pool completeness**: Observed slice count matches expected count

```go
func (c *ResourcePoolController) validatePool(slices []*ResourceSlice) []string {
    var errors []string

    // Check device name uniqueness
    deviceNames := make(map[string]string) // device -> slice name
    for _, slice := range slices {
        for _, device := range slice.Spec.Devices {
            if existing, ok := deviceNames[device.Name]; ok {
                errors = append(errors, fmt.Sprintf(
                    "device %q appears in both %s and %s",
                    device.Name, existing, slice.Name))
            }
            deviceNames[device.Name] = slice.Name
        }
    }

    // Check generation consistency
    generations := make(map[int64]int)
    for _, slice := range slices {
        generations[slice.Spec.Pool.Generation]++
    }
    if len(generations) > 1 {
        errors = append(errors, "ResourceSlices have inconsistent pool generations")
    }

    // Truncate errors if needed
    if len(errors) > 10 {
        errors = errors[:10]
    }

    return errors
}
```

### Client-Side Utility Library

A Go library for detailed allocation computation, used by kubectl:

```go
// Package drausage provides utilities for computing DRA resource usage.
package drausage

// PoolUsage contains detailed usage information for a resource pool.
type PoolUsage struct {
    Driver   string
    PoolName string
    NodeName string

    Devices []DeviceUsage
    Summary PoolSummary

    ValidationErrors []string
}

// DeviceUsage contains allocation details for a single device.
type DeviceUsage struct {
    Name        string
    State       DeviceState // Available, Allocated, PartiallyAllocated, Unavailable
    StateReason string      // For Unavailable state

    // Allocations lists claims using this device.
    // Only populated if caller has permission to fetch those claims.
    Allocations []AllocationInfo

    // For consumable capacity devices
    TotalCapacity     v1.ResourceList
    AllocatedCapacity v1.ResourceList
    AvailableCapacity v1.ResourceList
}

// AllocationInfo describes a single allocation.
type AllocationInfo struct {
    ClaimNamespace string
    ClaimName      string
    ClaimUID       types.UID
    Request        string
    Capacity       v1.ResourceList // For consumable capacity
}

// ComputePoolUsage calculates detailed usage for a pool.
// The caller must provide ResourceSlices and ResourceClaims they have access to.
func ComputePoolUsage(
    slices []*resourcev1.ResourceSlice,
    claims []*resourcev1.ResourceClaim,
) (*PoolUsage, error) {
    // ... implementation
}
```

**Benefits of client-side approach:**
- No RBAC issues: Only shows claims the user can access
- No API bloat: Detailed data not stored in ResourcePool
- Fresh data: Computed on demand from current state
- Flexible: Can be used by any Go client, not just kubectl

### kubectl Integration

#### kubectl describe resourcepool

Shows ResourcePool status plus optional detailed allocation info:

```
$ kubectl describe resourcepool example.com-gpu.node-1
Name:         example.com-gpu.node-1
Labels:       <none>
Annotations:  <none>
API Version:  resource.k8s.io/v1beta2
Kind:         ResourcePool

Spec:
  Driver:     example.com/gpu
  Pool Name:  node-1
  Node Name:  node-1

Status:
  Summary:
    Total Devices:              4
    Allocated Devices:          3
    Available Devices:          1
    Unavailable Devices:        0
    Partially Allocated Devices: 0
  Conditions:
    Type: Complete   Status: True   LastTransitionTime: 2025-06-15T10:00:00Z
    Type: Valid      Status: True   LastTransitionTime: 2025-06-15T10:00:00Z
  Observed Slice Count:    1
  Expected Slice Count:    1
  Observed Generation:     5
  Last Update Time:        2025-06-15T12:30:00Z

# If user has permission to list ResourceClaims, kubectl fetches them
# and computes detailed allocation info using the client-side library:

Device Details:
  NAME    STATE      ALLOCATED TO
  gpu-0   Allocated  default/ml-training-claim
  gpu-1   Allocated  default/ml-inference-claim
  gpu-2   Allocated  team-a/batch-job-claim
  gpu-3   Available  -

Events:  <none>
```

**For users without ResourceClaim access:**
```
$ kubectl describe resourcepool example.com-gpu.node-1
...
Status:
  Summary:
    Total Devices:       4
    Allocated Devices:   3
    Available Devices:   1

Device Details:
  (Requires permission to list ResourceClaims for detailed allocation info)

Events:  <none>
```

#### kubectl describe node

For node-local pools, shows DRA resource summary:

```
$ kubectl describe node node-1
Name:               node-1
...

DRA Resources:
  Pool: example.com-gpu.node-1
    Driver:     example.com/gpu
    Total:      4
    Allocated:  3
    Available:  1
```

**Implementation:**
- kubectl queries ResourcePools where `spec.nodeName` matches
- Displays summary from ResourcePool status
- Optionally shows device details if user has permission

### Test Plan

#### Prerequisite testing updates

None required.

#### Unit tests

Coverage targets for Alpha:
- `k8s.io/kubernetes/pkg/controller/resourcepool`: 80%+
  - Pool discovery from ResourceSlices
  - Summary calculation
  - Cross-slice validation
  - ResourcePool lifecycle (create/update/delete)

- `k8s.io/kubernetes/staging/src/k8s.io/kubectl/pkg/drausage`: 80%+
  - Client-side usage computation
  - Allocation mapping
  - Permission handling

- `k8s.io/kubectl/pkg/describe`: 75%+
  - ResourcePool describe formatting
  - Node describe DRA section

Test cases:
- Empty pool (no slices)
- Single slice pool
- Multi-slice pool
- Allocation changes
- Cross-slice validation errors
- Generation changes
- Consumable capacity devices
- Partitionable devices

#### Integration tests

Integration tests will verify:
1. Controller creates ResourcePool when ResourceSlices appear
2. ResourcePool status updates when ResourceClaims change
3. Cross-slice validation detects errors
4. ResourcePool deleted when all slices removed
5. Summary counts are accurate
6. Rate limiting prevents API server overload

Test scenarios:
- Create ResourceSlice → ResourcePool created
- Create ResourceClaim → summary updated
- Delete ResourceClaim → summary updated
- Multiple slices for one pool → aggregated correctly
- Validation error → surfaced in status
- Delete all slices → ResourcePool deleted

#### e2e tests

E2E tests will verify end-to-end functionality:
1. Deploy DRA driver that publishes ResourceSlices
2. Verify ResourcePool created with correct summary
3. Create ResourceClaim
4. Verify ResourcePool summary shows allocation
5. Run `kubectl describe resourcepool` and verify output
6. Verify `kubectl describe node` shows DRA resources
7. Delete ResourceClaim
8. Verify summary updated

### Graduation Criteria

#### Alpha

- Feature implemented behind `DRAResourceAvailabilityVisibility` feature gate
- ResourcePool API added to resource.k8s.io/v1beta2
- ResourcePool controller implemented in kube-controller-manager
- Client-side utility library implemented
- kubectl describe enhancements implemented
- Unit and integration tests completed
- Documentation for API and kubectl usage

#### Beta

- Feature enabled by default
- E2E tests in Testgrid and passing
- At least one DRA driver tested with the feature
- Performance testing with large numbers of pools and claims
- Metrics for monitoring controller performance
- User feedback incorporated

**Potential Beta enhancements:**
- Vendor-defined summarization attributes
- Additional filters for kubectl get resourcepools

#### GA

- At least 2 releases as beta
- Multiple DRA drivers using the feature in production
- Performance validated at scale (1000+ pools, 10000+ claims)
- No major bugs or design issues reported
- Documentation complete and accurate

### Upgrade / Downgrade Strategy

**Upgrade:**
- When feature gate is enabled, controller starts creating ResourcePool objects
- ResourcePools are created asynchronously for existing pools
- No impact on existing allocations or scheduling
- kubectl gracefully handles missing ResourcePools

**Downgrade:**
- When feature gate is disabled, controller stops running
- Existing ResourcePool objects can be cleaned up manually or left (harmless)
- No impact on scheduling or DRA functionality

**API compatibility:**
- ResourcePool is a new type, safe to add
- Old clients don't know about ResourcePool (graceful degradation)
- New clients handle missing ResourcePools gracefully

### Version Skew Strategy

**Control plane components:**
- kube-controller-manager: Runs ResourcePool controller when feature gate enabled
- kube-apiserver: Serves ResourcePool API
- kube-scheduler: Does not use ResourcePool (no dependency)

**Skew scenarios:**
1. **New apiserver, old controller-manager**: ResourcePool API exists but objects not created
2. **Old apiserver, new controller-manager**: Controller cannot create ResourcePools (feature disabled)
3. **Multiple controller-managers**: Leader election ensures only one runs

**kubectl:**
- New kubectl with old cluster: ResourcePool commands return "not found"
- Old kubectl with new cluster: ResourcePool commands not available

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: DRAResourceAvailabilityVisibility
  - Components depending on the feature gate:
    - kube-controller-manager (ResourcePool controller)
    - kube-apiserver (ResourcePool API)
    - kubectl (describe enhancements)

###### Does enabling the feature change any default behavior?

No. This feature only adds new ResourcePool objects. It does not change:
- How ResourceSlices are created or managed by drivers
- How ResourceClaims are allocated
- How pods are scheduled
- Any existing API fields or behaviors

###### Can the feature be disabled once it has been enabled?

Yes. Disabling the feature gate will:
- Stop the ResourcePool controller from running
- Stop kubectl from using ResourcePool-related commands

ResourcePool objects will remain but not be updated. They can be manually deleted or left (harmless).

###### What happens if we reenable the feature if it was previously rolled back?

The controller will resume, update existing ResourcePools, and create any missing ones.

###### Are there any tests for feature enablement/disablement?

Yes, integration tests will verify enable/disable behavior.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

**Rollout failures:**
- Controller bugs could fail to create/update ResourcePools (user-facing only)
- No impact on running workloads (ResourcePool is informational)

**Impact on workloads:**
- None. ResourcePool is read-only information for visibility.

###### What specific metrics should inform a rollback?

- `resourcepool_controller_sync_errors_total` - High error rate
- `resourcepool_controller_sync_duration_seconds` - High latency
- `workqueue_depth{name="resourcepool"}` - Growing unbounded

###### Were upgrade and rollback tested?

Will be tested in integration tests.

###### Is the rollout accompanied by any deprecations and/or removals?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use?

- Check for ResourcePool objects: `kubectl get resourcepools`
- Check controller metrics in kube-controller-manager

###### How can someone using this feature know that it is working?

- ResourcePool objects exist for each pool
- ResourcePool status reflects current allocations
- `kubectl describe resourcepool` shows expected information

###### What are the reasonable SLOs for the enhancement?

- ResourcePool status update latency: 95% within 10 seconds of allocation change
- ResourcePool accuracy: 99% consistent with actual allocations

###### What are the SLIs?

- `resourcepool_controller_sync_duration_seconds` (histogram)
- `resourcepool_controller_sync_errors_total` (counter)
- `resourcepool_controller_pools_total` (gauge)

###### Are there any missing metrics?

For Beta: staleness metric, claim tracking count.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No external dependencies. Requires:
- kube-apiserver
- kube-controller-manager
- DRA drivers creating ResourceSlices (existing requirement)

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes:
- Controller watches ResourceSlices and ResourceClaims
- Controller creates/updates ResourcePool objects
- kubectl fetches ResourcePools and optionally ResourceClaims

###### Will enabling / using this feature result in introducing new API types?

Yes: ResourcePool (cluster-scoped)

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No. ResourceSlice and ResourceClaim are unchanged.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No impact on scheduling or pod startup.

###### Will enabling / using this feature result in non-negligible increase of resource usage?

**kube-controller-manager:**
- CPU: Low (watching and reconciling)
- RAM: Moderate (informer caches) - ~10-50MB for typical clusters
- Network: Minimal (watch streams + ResourcePool updates)

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Controller retries with backoff. No impact on workloads.

###### What are other known failure modes?

- **ResourcePool becomes stale**: Check controller logs, verify feature gate
- **Validation errors not surfacing**: Check controller is running

###### What steps should be taken if SLOs are not being met?

1. Check controller-manager health
2. Check controller metrics
3. Verify feature gate enabled
4. Check API server health

## Implementation History

- 2025-12-20: KEP created in provisional state
- 2026-01-15: Major design revision based on API review feedback - changed from ResourceSlice status to ResourcePool object

## Drawbacks

1. **New API type**: Adds ResourcePool to the API surface
   - Mitigation: Clean separation of concerns, doesn't modify existing types

2. **Controller overhead**: Additional controller watching slices and claims
   - Mitigation: Efficient informers, rate limiting, only updates on changes

3. **Eventual consistency**: ResourcePool may lag behind actual allocations
   - Mitigation: Document behavior, use observedGeneration

4. **Limited detail in API**: Individual claim references only available via kubectl (client-side)
   - Mitigation: This is intentional for RBAC reasons; admins can still debug via kubectl

## Alternatives

### Alternative 1: Status in ResourceSlice

The original design added a Status field to ResourceSlice to track per-device allocations.

**Pros:**
- No new API type
- Allocation details in API

**Cons:**
- Increases ResourceSlice size (O(allocations) per slice)
- Increases ResourceSlice write churn
- RBAC issues: claim namespace/name exposed to anyone who can read ResourceSlices
- Cross-slice pool status awkward (which slice holds pool status?)

**Rejected because:** API review feedback indicated concerns about size, churn, and RBAC. ResourcePool provides a cleaner separation.

### Alternative 2: Status in ResourceClaim

Add pool availability to ResourceClaim status.

**Cons:**
- ResourceClaims are namespaced - can't show cross-namespace availability
- Users can't see availability before creating a claim
- Duplication if multiple claims need pool info

**Rejected because:** Doesn't solve the cross-namespace visibility problem.

### Alternative 3: Metrics-based visibility only

Expose availability only through Prometheus metrics.

**Cons:**
- Not queryable via kubectl
- May not be available in all clusters
- Can't correlate with specific claims

**Rejected because:** Poor user experience for the primary kubectl use case.

### Alternative 4: Client-side only (no API changes)

Only provide kubectl utilities, no ResourcePool object.

**Pros:**
- No API changes
- No runtime controller cost

**Cons:**
- No API-visible pool status
- Can't see pool validation errors without running kubectl
- Other tools can't access availability info

**Rejected because:** API-visible pool status is valuable for monitoring and automation. However, this approach is still used for detailed allocation info.

### Alternative 5: Custom apiserver endpoint with report generator

Add a custom aggregated API endpoint (e.g., `/apis/resource.k8s.io/v1/poolreports`) that dynamically generates availability reports on-demand rather than storing ResourcePool objects.

**Pros:**
- Always fresh data (computed on request)
- No storage cost in etcd
- No controller reconciliation overhead
- Could support flexible query parameters (filter by driver, node, etc.)

**Cons:**
- Higher API server CPU cost per request (must compute on every call)
- No caching benefits - repeated queries recompute
- More complex implementation (custom aggregated API server)
- Harder to watch for changes (no resourceVersion semantics)
- Inconsistent with Kubernetes resource model (not a persisted object)
- Cannot be used with standard tooling expecting resources (informers, controllers, kubectl get --watch)
- Latency scales with cluster size on every request

**Rejected because:** The ResourcePool approach fits better with Kubernetes conventions - it's a standard resource that can be watched, cached by informers, and used with existing tooling. The controller overhead is minimal (only updates on changes), while the custom endpoint would compute on every request. For large clusters, amortizing the computation cost via a controller is more efficient than recomputing per-request.

## Infrastructure Needed

No special infrastructure required. Implementation uses:
- Existing Kubernetes API infrastructure
- Existing kube-controller-manager framework
- Existing kubectl framework
- Standard e2e test infrastructure
