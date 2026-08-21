# KEP-5677: DRA Resource Availability Visibility

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Architecture](#architecture)
  - [User Stories](#user-stories)
    - [Story 1: Cluster Administrator Checking Pool Status](#story-1-cluster-administrator-checking-pool-status)
    - [Story 2: Developer Debugging Resource Allocation](#story-2-developer-debugging-resource-allocation)
    - [Story 3: Automation and Monitoring](#story-3-automation-and-monitoring)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Scaling Risks](#scaling-risks)
    - [Operational Risks](#operational-risks)
  - [Security Considerations](#security-considerations)
    - [RBAC](#rbac)
    - [Information Exposure](#information-exposure)
    - [Security Risks](#security-risks)
    - [Controller Security](#controller-security)
    - [Future Consideration: Namespace-scoped Requests](#future-consideration-namespace-scoped-requests)
- [Design Details](#design-details)
  - [API Definition](#api-definition)
    - [ResourcePoolStatusRequest Object](#resourcepoolstatusrequest-object)
    - [Spec Fields](#spec-fields)
    - [Status Fields](#status-fields)
    - [Companion API Change: <code>ResourceSlice.Spec.PartitionTypeAttribute</code>](#companion-api-change-resourceslicespecpartitiontypeattribute)
      - [Where validation happens](#where-validation-happens)
      - [Feature gate: <code>DRAPartitionableDevicesType</code>](#feature-gate-drapartitionabledevicestype)
  - [Controller Implementation](#controller-implementation)
    - [Controller in KCM](#controller-in-kcm)
    - [One-time Processing](#one-time-processing)
    - [Incomplete-Pool Handling and Requeue](#incomplete-pool-handling-and-requeue)
    - [Reusing Existing Informers](#reusing-existing-informers)
    - [Partitionable &amp; Consumable Device Accounting](#partitionable--consumable-device-accounting)
      - [Conditions and Metrics](#conditions-and-metrics)
      - [Devices That Are Both Partitionable and Consumable](#devices-that-are-both-partitionable-and-consumable)
    - [TTL-Based Cleanup](#ttl-based-cleanup)
    - [Controller RBAC](#controller-rbac)
  - [kubectl Integration](#kubectl-integration)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
    - [Unit tests](#unit-tests)
    - [Integration tests](#integration-tests)
    - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (1.36)](#alpha-136)
    - [Alpha (1.37)](#alpha-137)
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
  - [Alternative 1: Out-of-tree Aggregated API Server](#alternative-1-out-of-tree-aggregated-api-server)
  - [Alternative 2: Synchronous Review Pattern](#alternative-2-synchronous-review-pattern)
  - [Alternative 3: Status in ResourceSlice](#alternative-3-status-in-resourceslice)
  - [Alternative 4: Client-side only](#alternative-4-client-side-only)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in
  [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and
  SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests]
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints] must be hit by [Conformance Tests]
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for
  publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to
  mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website
[Conformance Tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md
[all GA Endpoints]: https://github.com/kubernetes/community/pull/1806

## Summary

This KEP addresses a visibility gap in Dynamic Resource Allocation (DRA) by
enabling users to view available device capacity across resource pools. While
ResourceSlices store capacity data and ResourceClaims track consumption, there
is currently no straightforward way for users to view the available capacity
remaining in a pool or on a node.

This enhancement introduces a **ResourcePoolStatusRequest** API following the
CertificateSigningRequest (CSR) pattern:

1. User creates a ResourcePoolStatusRequest object specifying a driver (required) and optional pool filter
2. A controller in kube-controller-manager watches for new requests
3. Controller computes pool availability and writes result to status
4. User reads the status to see pool availability
5. To recalculate, user deletes and recreates the request

This in-tree approach was chosen based on API review feedback to:
- Provide an always-available, in-sync solution with Kubernetes releases
- Follow established patterns (CSR, device taints with "None" effect)
- Control permissions via standard RBAC on the request object
- Avoid continuous controller overhead (one-time computation per request)

## Motivation

Dynamic Resource Allocation (DRA) provides a flexible framework for managing
specialized hardware resources like GPUs, FPGAs, and other accelerators.
However, the current implementation lacks visibility into resource availability:

**Current State:**
- ResourceSlices are cluster-scoped resources that publish total capacity of
  devices in a pool
- ResourceClaims are namespaced and track individual allocations
- Users with limited RBAC permissions cannot see ResourceClaims outside their
  namespace
- No API-level view of "available" vs "allocated" capacity
- Difficult to understand why scheduling is failing or plan capacity

**Problems this creates:**
1. **Debugging difficulty**: When pods fail to schedule due to insufficient
   resources, users cannot easily see what is available vs. what is consumed
2. **Capacity planning**: Cluster administrators cannot easily determine if
   more resources are needed
3. **Cross-namespace visibility**: Even cluster admins need to query multiple
   namespaces to understand total consumption

### Goals

- Provide pool-level availability summaries via a standard Kubernetes API
- Follow established request/status patterns (like CSR)
- Compute availability on-demand (only when requested)
- Always available in-tree, in-sync with Kubernetes releases
- Require driver specification, with optional pool name filter
- Provide cross-slice validation to surface pool consistency issues
- Control access via standard RBAC on the request object
- Keep ResourceClaim and ResourceSlice APIs unchanged, requiring no
  modifications to existing DRA drivers or scheduler
- Allow less-privileged users to access resource usage information without
  exposing data beyond their normal RBAC access (e.g., cross-namespace claims)

### Non-Goals

- Adding real-time metrics/monitoring (this is point-in-time status)
- Implementing quotas or limits based on availability (future work)
- Providing historical consumption data (use multiple requests for that)
- Watch support for continuous updates (create new requests instead)

## Proposal

This KEP proposes a **ResourcePoolStatusRequest** API following the
CertificateSigningRequest (CSR) pattern - an established Kubernetes pattern
for imperative operations through declarative APIs.

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              User Workflow                                  │
│                                                                             │
│   Step 1: CREATE               Step 2: WAIT              Step 3: READ       │
│   $ kubectl create             $ kubectl wait            $ kubectl get      │
│     -f request.yaml              --for=condition=Complete  ...-o yaml       │
│                                  <object-name>                              │
│   (kind: ResourcePoolStatusRequest, resource.k8s.io/v1alpha3)               │
└───────────┬─────────────────────────┬─────────────────────────┬─────────────┘
            │                         │                         │
            ▼                         ▼                         ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            kube-apiserver                                   │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │              ResourcePoolStatusRequest  (stored in etcd)              │  │
│  │                                                                       │  │
│  │  metadata:                                                            │  │
│  │    name: my-check                                                     │  │
│  │                                                                       │  │
│  │  spec:                              status:                           │  │
│  │    driver: example.com/gpu    ───►    poolCount: 1                    │  │
│  │    poolName: node-1                   pools:                          │  │
│  │                                       - driver: example.com/gpu       │  │
│  │                                         poolName: node-1              │  │
│  │                                         generation: 5                 │  │
│  │                                         resourceSliceCount: 1         │  │
│  │                                         totalDevices: 4               │  │
│  │                                         allocatedDevices: 3           │  │
│  │                                         availableDevices: 1           │  │
│  │                                         unavailableDevices: 0         │  │
│  │                                       conditions:                     │  │
│  │                                       - type: Complete                │  │
│  │                                         status: "True"                │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
                                        ▲
                                        │ Watch + UpdateStatus
                                        │
┌───────────────────────────────────────┴─────────────────────────────────────┐
│                          kube-controller-manager                            │
│                                                                             │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │                ResourcePoolStatusRequest Controller                    │ │
│  │                                                                        │ │
│  │  1. Watch for new ResourcePoolStatusRequest objects                    │ │
│  │  2. Skip if status is already set (one-time processing)                │ │
│  │  3. Read ResourceSlices matching spec filters (driver, poolName)       │ │
│  │  4. Read ResourceClaims to determine allocations                       │ │
│  │  5. Compute availability summary per pool (per-pool validationError    │ │
│  │     when observed slice count < expected; controller requeues to       │ │
│  │     give drivers time to publish remaining slices)                     │ │
│  │  6. Write result to status                                             │ │
│  │  7. Set condition Complete=True (or Failed=True on error)              │ │
│  │  8. TTL cleanup: completed requests deleted after 1h, pending          │ │
│  │     requests after 24h                                                 │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│  Reuses existing informers:                                                 │
│  ┌─────────────────┐  ┌─────────────────┐                                   │
│  │ ResourceSlices  │  │ ResourceClaims  │                                   │
│  └─────────────────┘  └─────────────────┘                                   │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Key design points:**

1. **CSR-like pattern**: User creates request, controller processes, user reads
   status - established pattern in Kubernetes
2. **One-time processing**: Controller skips requests that already have status,
   ensuring each request is processed exactly once
3. **Reuses existing informers**: Controller reuses ResourceSlice and
   ResourceClaim informers already in KCM, adding minimal overhead
4. **Always available**: In-tree implementation, no additional deployment needed
5. **Standard RBAC**: Access controlled via RBAC on ResourcePoolStatusRequest

### User Stories

#### Story 1: Cluster Administrator Checking Pool Status

As a cluster administrator, I want to see at a glance how many GPU resources
are available across my cluster so that I can understand current utilization
and plan for capacity expansion.

**Workflow:**
```bash
# Create a status request for all GPU pools
$ kubectl create -f - <<EOF
apiVersion: resource.k8s.io/v1alpha3
kind: ResourcePoolStatusRequest
metadata:
  name: check-gpus-$(date +%s)
spec:
  driver: example.com/gpu
EOF
resourcepoolstatusrequest.resource.k8s.io/check-gpus-1707300000 created

# Wait for processing
$ kubectl wait --for=condition=Complete resourcepoolstatusrequest/check-gpus-1707300000 --timeout=30s
resourcepoolstatusrequest.resource.k8s.io/check-gpus-1707300000 condition met

# View results
$ kubectl get resourcepoolstatusrequest/check-gpus-1707300000 -o yaml
apiVersion: resource.k8s.io/v1alpha3
kind: ResourcePoolStatusRequest
metadata:
  name: check-gpus-1707300000
spec:
  driver: example.com/gpu
status:
  poolCount: 2
  pools:
  - driver: example.com/gpu
    poolName: node-1
    nodeName: node-1
    generation: 1
    resourceSliceCount: 1
    totalDevices: 4
    allocatedDevices: 3
    availableDevices: 1
    unavailableDevices: 0
  - driver: example.com/gpu
    poolName: node-2
    nodeName: node-2
    generation: 1
    resourceSliceCount: 1
    totalDevices: 4
    allocatedDevices: 1
    availableDevices: 3
    unavailableDevices: 0
  conditions:
  - type: Complete
    status: "True"
    reason: CalculationComplete
    message: "Calculated status for 2 pools"
    lastTransitionTime: "2026-02-07T10:30:00Z"

# Cleanup (or wait for TTL - 1h after completion)
$ kubectl delete resourcepoolstatusrequest/check-gpus-1707300000
```

#### Story 2: Developer Debugging Resource Allocation

As a developer, when my pod fails to schedule because "insufficient DRA
resources", I want to understand what resources are available.

**Workflow:**
```bash
# Quick one-liner to check GPU availability
$ kubectl create -f - <<EOF && sleep 2 && \
  kubectl get resourcepoolstatusrequest/debug-check -o jsonpath='{.status.pools[*]}'
apiVersion: resource.k8s.io/v1alpha3
kind: ResourcePoolStatusRequest
metadata:
  name: debug-check
spec:
  driver: example.com/gpu
EOF

# Output shows which nodes have available GPUs:
# node-1: 0 available (fully allocated)
# node-2: 3 available
# node-3: 0 available (fully allocated)
```

#### Story 3: Automation and Monitoring

As an automation system, I want to periodically check resource availability
to trigger alerts or scaling actions.

**Workflow:**
```bash
#!/bin/bash
# Cron job that runs every 5 minutes

REQUEST_NAME="monitor-$(date +%s)"
DRIVER="example.com/gpu"

# Create request
kubectl create -f - <<EOF
apiVersion: resource.k8s.io/v1alpha3
kind: ResourcePoolStatusRequest
metadata:
  name: $REQUEST_NAME
spec:
  driver: $DRIVER
EOF

# Wait and get result
kubectl wait --for=condition=Complete resourcepoolstatusrequest/$REQUEST_NAME --timeout=60s
AVAILABLE=$(kubectl get resourcepoolstatusrequest/$REQUEST_NAME -o jsonpath='{.status.pools[*].availableDevices}' | tr ' ' '+' | bc)

# Alert if low
if [ "$AVAILABLE" -lt 5 ]; then
  echo "ALERT: Only $AVAILABLE devices available cluster-wide"
fi

# Cleanup (or let TTL delete it after 1h)
kubectl delete resourcepoolstatusrequest/$REQUEST_NAME
```

### Notes/Constraints/Caveats

1. **Asynchronous operation**: Unlike SubjectAccessReview (synchronous), this
   uses the CSR pattern where user must wait for controller processing.

2. **One-time calculation**: Each request is processed once. Once `status`
   is set it becomes immutable; metadata (labels, annotations) follows
   the standard object-meta update rules. To get updated data, delete and
   recreate the request.

3. **Automatic TTL cleanup**: Completed or failed requests are deleted by the
   controller 1 hour after their `Complete`/`Failed` condition is set.
   Pending requests (no status) are deleted 24 hours after creation to
   handle stuck requests. Users can still delete requests manually at any
   time.

4. **Controller processing delay**: Status is not immediate - controller must
   process the request. Typically completes within seconds.

5. **RBAC controls access**: Users need RBAC permission to create/read
   ResourcePoolStatusRequest objects to use this feature.

6. **Partitionable & consumable devices**: Alpha 1.36 counted each
   entry in `ResourceSlice.Spec.Devices` once per allocation result,
   which was misleading for two device shapes:

   - **Partitionable** (`DRAPartitionableDevices` feature gate): a
     single physical device may appear as multiple mutually-exclusive
     partitions that share a `CounterSet`. Counting devices ignores
     the shared bottleneck.
   - **Consumable** (`DRAConsumableCapacity` feature gate): a device
     with `allowMultipleAllocations=true` may serve many claims
     simultaneously. Counting each claim against `allocatedDevices`
     drove `availableDevices` to 0 on pools that still had free
     capacity (the `max(0, …)` floor in the controller hid the
     overcount as "0 available" rather than as a negative number).

   Alpha 1.37 added an optional `partitionSummary` list (a typed
   "devices-by-partition-type" view that nets out shared counter
   consumption) and a `shareableSummary` aggregate to each
   `PoolStatus`, capped the per-device contribution to
   `allocatedDevices` at 1, and skipped AdminAccess allocations in all
   accounting. `partitionSummary` is emitted when a grouping attribute
   can be resolved for the pool — either because the driver declared
   `ResourceSlice.Spec.PartitionTypeAttribute` on a slice, or because
   the request named `spec.defaultPartitionTypeAttribute`. A pool for
   which neither source names an attribute reports no
   `partitionSummary` at all. See
   [Partitionable & Consumable Device Accounting](#partitionable--consumable-device-accounting)
   under Controller Implementation.

7. **Incomplete pools**: When a pool's observed ResourceSlice count is less
   than `ResourceSliceCount` declared by the driver, the pool is considered
   incomplete and the controller requeues the request (up to 5 attempts) to
   give drivers time to publish remaining slices. The status is **not**
   written while any pool is incomplete. A request therefore either
   completes once every pool is whole — in which case no `PoolIncomplete:`
   marker survives into the result — or exhausts its retries and is left
   with `status` unset until the 24-hour pending TTL deletes it. Either way
   the incomplete state is never visible to a reader. Reaching a terminal
   state in this case is a Beta item; see [Beta](#beta) under Graduation
   Criteria.

8. **Generation handling**: ResourceSlices with older pool generations are
   ignored during computation (not counted as errors). Drivers are expected
   to delete old-generation slices eventually. The `generation` field in
   each PoolStatus reflects the highest generation observed.

9. **`unavailableDevices`**: in Alpha 1.36 always `0`. Alpha 1.37
   computes this from real device taints (`NoSchedule` and
   `NoExecute` effects) on each device. The count is taken over all
   devices in the pool regardless of whether they are also allocated,
   so a device that is both allocated and tainted is subtracted twice
   from `availableDevices`; the `max(0, …)` floor hides the resulting
   underflow. Reconciling this with the field's stated meaning
   ("not available due to taints … but are not allocated") is a Beta
   item; see [Beta](#beta) under Graduation Criteria.

### Risks and Mitigations

#### Scaling Risks

| Risk | Mitigation |
|------|------------|
| Request accumulation in etcd | Controller-side TTL cleanup (Alpha): 1h after completion, 24h for pending |
| Large status objects (many pools) | Required `driver` field bounds response; `limit` field capped at 1000 (default 100); status `pools` list capped at `maxItems=1000` |
| Controller processing spike | Work queue with default rate limiting; max 5 retries per request |
| Simultaneous request flood | Per-user rate limiting (planned for Beta) |

**Alpha approach:** The required `driver` field naturally bounds response
size to one driver's pools, with `limit` (default 100, max 1000) as an
additional cap. Built-in TTL cleanup runs every 10 minutes and deletes
completed requests after 1 hour and pending requests after 24 hours, so etcd
growth is bounded without user action. Cluster administrators can still
enforce additional object-count limits via admission webhooks (e.g.
Gatekeeper, Kyverno).

**Beta improvements:** Per-user rate limiting for request creation, and
consideration of configurable TTLs and a built-in cluster-wide object limit
if Alpha feedback indicates a need.

#### Operational Risks

| Risk | Mitigation |
|------|------------|
| Stale data if not recalculated | `Complete` condition's `lastTransitionTime` shows age; delete and recreate for fresh data |
| Controller not running | `status` stays nil (no `Complete` or `Failed` condition); user can detect; request will be auto-deleted after 24h pending TTL |
| Feature gate mismatch | Feature gate `DRAResourcePoolStatus` must be enabled on both kube-apiserver and kube-controller-manager |

### Security Considerations

#### RBAC

Access is controlled via standard RBAC on the ResourcePoolStatusRequest API.
**No new default ClusterRoles are created** - administrators must explicitly
grant access to users who need this feature.

- `cluster-admin` has full access automatically (existing wildcard permissions)
- Other users require explicit RBAC grants via custom ClusterRole/ClusterRoleBinding
- This feature is **not** added to `system:aggregate-to-admin` or similar roles

Example ClusterRole for granting access:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: pool-status-reader
rules:
- apiGroups: ["resource.k8s.io"]
  resources: ["resourcepoolstatusrequests"]
  verbs: ["create", "get", "list", "delete"]
```

Cluster administrators should carefully consider who receives this role,
as it exposes infrastructure information (see below).

#### Information Exposure

| User Role | Can See |
|-----------|---------|
| No RPSR access | Nothing |
| RPSR create/read | Pool summaries (counts only) |
| RPSR + Claim reader | Could correlate with claim data separately |

**What is exposed:**
- Pool names, driver names, node names
- Device counts (total, allocated, available)
- Validation errors (pool consistency issues)

**What is NOT exposed:**
- Which specific claims are using which devices
- Claim contents or pod information
- Raw ResourceSlice data
- Cross-namespace claim information

#### Security Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Infrastructure info disclosure | Low | RBAC controls access |
| DoS via request flooding | Medium | Work queue rate limiting |
| Cross-namespace claim leak | None | Design excludes claim details |
| Privilege escalation | None | Controller has read-only access |

#### Controller Security

The controller runs in KCM with existing permissions to read ResourceSlices
and ResourceClaims. No additional permissions are needed beyond what
device-taint-eviction controller already has.

#### Future Consideration: Namespace-scoped Requests

For environments requiring stricter isolation, a namespace-scoped variant
(similar to LocalSubjectAccessReview) could be added in future versions.
This would allow users to only see pools with devices allocated to claims
in their namespace.

## Design Details

### API Definition

#### ResourcePoolStatusRequest Object

The API is introduced in `resource.k8s.io/v1alpha3` (Kubernetes 1.36).

```yaml
apiVersion: resource.k8s.io/v1alpha3
kind: ResourcePoolStatusRequest
metadata:
  name: my-request
  # Cluster-scoped (no namespace)
spec:
  # Driver is REQUIRED - bounds response to one driver's pools.
  # Must be a DNS subdomain.
  driver: example.com/gpu

  # Filter by pool name (optional).
  # When set, must be a valid resource pool name (DNS subdomains separated by "/").
  poolName: node-1

  # Max pools to return (optional). Default: 100. Min: 1. Max: 1000.
  limit: 100

  # Grouping attribute to fall back to for partitionable pools whose
  # slices do not declare one (optional, gated by
  # DRAPartitionableDevicesType). Must be fully qualified.
  defaultPartitionTypeAttribute: gpu.example.com/profile

status:
  # Total number of pools that matched the filter (even if the response is
  # truncated by `limit`). If 0, no pools matched.
  poolCount: 4

  # First `spec.limit` matching pools, sorted by driver then pool name.
  # If len(pools) < poolCount, the response was truncated.
  pools:
  - driver: example.com/gpu
    poolName: node-1
    generation: 5                 # Pool generation observed (int64)
    nodeName: node-1              # Omitted for multi-node / mixed-node pools
    resourceSliceCount: 1         # Observed ResourceSlices at the latest generation
    totalDevices: 4
    allocatedDevices: 3
    availableDevices: 1
    unavailableDevices: 0         # 0 in Alpha 1.36; computed from device taints in Alpha 1.37
  - driver: example.com/gpu
    poolName: node-2
    generation: 5
    # validationError is set when a pool is incomplete (observed < expected
    # slice count). When set, count fields are unset. Max 256 bytes.
    validationError: "pool example.com/gpu/node-2 is incomplete: observed 1/2 slices at generation 5"
  # Partitionable pool (Alpha 1.37): one physical GPU offered as either a
  # full partition (80Gi cost) or two half partitions (40Gi cost each), all
  # backed by a single 80Gi CounterSet. A grouping attribute was resolved
  # for the pool, so the controller emits the typed view. Here one half
  # partition is in use, so 40Gi of the counter is consumed (debited
  # per-device, not per-claim), leaving 40Gi. The other half still fits;
  # the full partition no longer does.
  - driver: example.com/gpu
    poolName: node-3
    generation: 7
    nodeName: node-3
    resourceSliceCount: 1
    totalDevices: 3               # 1 full + 2 half device entries
    allocatedDevices: 1           # one half in use (cap-at-1)
    availableDevices: 2           # naive count — see partitionSummary for truth
    unavailableDevices: 0
    partitionSummary:             # emitted when a grouping attribute is
                                  # resolved for the pool; entries are keyed
                                  # by (attribute, type) and are measured
                                  # independently — they must not be summed
    - attribute: gpu.example.com/profile
      type: Full                  # value of the grouping attribute
      total: 1                    # device entries of this partition type
      allocatable: 0              # 40Gi left, Full needs 80Gi → blocked
    - attribute: gpu.example.com/profile
      type: Half
      total: 2
      allocatable: 1              # 1 fresh Half remains, 40Gi available, fits
  # Consumable pool (Alpha 1.37): devices with allowMultipleAllocations=true.
  # allocatedDevices counts each shared device once (cap-at-1), so it can read
  # "1 available" while capacity headroom remains; consult shareableSummary.
  - driver: example.com/gpu
    poolName: node-4
    generation: 3
    nodeName: node-4
    resourceSliceCount: 1
    totalDevices: 3
    allocatedDevices: 2
    availableDevices: 1
    unavailableDevices: 0
    shareableSummary:             # emitted only when the pool has shareable devices
      fullyAvailableDevices: 1    # devices with zero non-AdminAccess claims
      partiallyAvailableDevices: 2 # devices with at least one such claim,
                                   # regardless of remaining capacity
      capacity:                   # per-capacity-key aggregate across shareable devices
      - name: example.com/memory
        total: 240Gi
        consumed: 90Gi
        available: 150Gi

  # Conditions indicating processing status.
  # Known types: "Complete" (True when processed successfully) and
  # "Failed" (True when the request could not be processed). Max 10 entries.
  conditions:
  - type: Complete
    status: "True"
    reason: CalculationComplete
    message: "Calculated status for 4 pools (1 incomplete)"
    lastTransitionTime: "2026-02-07T10:30:00Z"
```

Once `status` is populated it becomes write-once (frozen via
`ValidateImmutableField`). `spec` is independently immutable from creation
(`+k8s:immutable`). `metadata` (labels, annotations) follows the standard
object-meta update rules through the main endpoint; the status subresource
strips metadata changes per the usual `ResetObjectMetaForStatus` convention.
To re-run a query, delete and recreate the request.

#### Spec Fields

The spec is **immutable after creation** (enforced via `+k8s:immutable`).
Updates to the spec are rejected by API validation regardless of whether
`status` has been written. Status write-once semantics are described in
the next section.

| Field | Type | Description |
|-------|------|-------------|
| `driver` | string (required) | DRA driver name — bounds response to one driver's pools. Must be a DNS subdomain. |
| `poolName` | `*string` (optional) | Filter by pool name. Must be a valid resource pool name (DNS subdomains separated by `/`). |
| `limit` | `*int32` (optional) | Max pools to return. Default **100**, min **1**, max **1000**. Defaulted by the apiserver, so the field is required after defaulting. |
| `defaultPartitionTypeAttribute` | `*string` (optional, gated by `DRAPartitionableDevicesType`) | Fully qualified name of a device attribute to use as the grouping attribute for partitionable pools whose slices declare none. A slice's own `PartitionTypeAttribute` always wins; this default applies only when **no** slice in the pool declares one, so a request can still get a `partitionSummary` from a driver that has not adopted the slice-side declaration. Cleared by the registry strategy when the gate is disabled, unless the stored object already sets it (standard ratcheting); the declarative `+k8s:ifDisabled(DRAPartitionableDevicesType)=+k8s:forbidden` rule backstops that drop. |

#### Status Fields

Status is a pointer (`*ResourcePoolStatusRequestStatus`). Presence of a
non-nil status indicates the request has been processed.

| Field | Type | Description |
|-------|------|-------------|
| `poolCount` | `*int32` (required) | Total pools matching filter (regardless of truncation). |
| `pools` | atomic list, max 1000 | First `spec.limit` matching pools, sorted by driver then pool name. Truncation is inferred from `len(pools) < poolCount`. |
| `pools.driver` | string (required) | DRA driver name. |
| `pools.poolName` | string (required) | Pool name from ResourceSlice. |
| `pools.generation` | int64 (required) | Latest pool generation observed. |
| `pools.nodeName` | `*string` (optional) | Node name for node-local pools. Omitted when the pool spans multiple nodes or has mixed/no node assignment. |
| `pools.resourceSliceCount` | `*int32` (optional, min 1) | Number of slices observed at the latest generation. Unset when `validationError` is set. |
| `pools.totalDevices` | `*int32` (optional) | Total devices across all slices. Unset when `validationError` is set. |
| `pools.allocatedDevices` | `*int32` (optional) | Devices allocated to claims. Unset when `validationError` is set. |
| `pools.availableDevices` | `*int32` (optional) | `totalDevices - allocatedDevices - unavailableDevices`. Unset when `validationError` is set. |
| `pools.unavailableDevices` | `*int32` (optional) | Count of devices with at least one `NoSchedule` or `NoExecute` taint, sourced from `ResourceSlice.Spec.Devices[].Taints` and matching `DeviceTaintRule`s. **0 in Alpha 1.36** (hard-coded); computed from real taints since Alpha 1.37. Counted over all devices in the pool regardless of allocation, so a device that is both allocated and tainted is subtracted twice from `availableDevices`. Unset when `validationError` is set. |
| `pools.validationError` | `*string` (optional, max 256 bytes) | Set when the pool's data could not be fully validated. When set, the count fields above may be unset (incomplete pool) or still populated (a view-level error, which clears `partitionSummary` but leaves the counts valid). The controller emits a stable, machine-readable prefix followed by `: ` and a free-form detail so operators can grep / alert on the specific case without parsing the message body. Prefixes as of Alpha 1.37 (provisional, may grow): `PoolIncomplete:` (observed slices < declared `ResourceSliceCount`) — note that this one is **computed but never persisted**, because the controller aborts the status write and requeues whenever any pool carries it, so in practice a reader never observes it; the remaining prefixes are permanent and are written to status: `PartitionTypeMissing:` (a grouped device lacks the resolved partition-type attribute, or the pool resolves a grouping attribute but publishes no `sharedCounters`), `PartitionCostMismatch:` (devices of the same partition type publish different `ConsumesCounters` costs), `PartitionSummaryOverCap:` (distinct partition types exceed the 32-item cap), `ShareableSummaryOverCap:` (distinct shareable capacity keys exceed the 32-item cap). Promoting this field to a structured `{reason, message}` pair (Condition-style) is tracked as a Beta consideration. |
| `pools.partitionSummary` | atomic list of `PartitionTypeStatus`, max 32, unique on (`attribute`, `type`) (Alpha 1.37, **provisional** — revisit at Beta) | Per-(attribute, partition-type) aggregate, emitted for a partitionable pool that publishes `SharedCounters` and for which a grouping attribute could be resolved — from `ResourceSlice.Spec.PartitionTypeAttribute` on a slice, or from `spec.defaultPartitionTypeAttribute` on the request. A pool that mixes partitions declared under different attributes reports each independently. When neither source names an attribute, the pool reports no `partitionSummary`. A grouped device missing the resolved attribute produces a per-pool `validationError`, as does a device whose `ConsumesCounters` cost differs from peers of the same type — both prevent silent bucketing. Cap of 32 is a provisional starting point that fits MIG-class pools (3–7 partition types typical); over-cap pools produce a per-pool `validationError` instead of silent truncation. Entries are sorted by (`attribute`, `type`). |
| `pools.partitionSummary.attribute` | string (required) | Fully qualified name of the device attribute whose value groups this entry — the `PartitionTypeAttribute` declared by the devices' own slice, or the request's `defaultPartitionTypeAttribute` when their slice declares none. |
| `pools.partitionSummary.type` | string (required) | Value of that attribute for devices in this group (e.g. `Full`, `Half`). |
| `pools.partitionSummary.total` | `*int32` (required) | Number of devices in the pool whose grouping attribute carries this value. |
| `pools.partitionSummary.allocatable` | `*int32` (required) | Number of additional devices of this partition type that can still be allocated under current shared-counter constraints, capped by the number of unallocated devices of this type in the pool. Computed by a greedy per-device fit check against `counterAvailable[s][c] = SharedCounters[s].Counters[c].Value − sum_{in-use d in s} d.ConsumesCounters[s][c]` (each in-use device debited once, per-device not per-claim — matches scheduler counter accounting). For the common single-counter-set case this reduces to `min(freshDevices[type], min over counters c of floor(counterAvailable[s_type][c] / consumesCounters[type][c]))`, where `freshDevices[type]` is the count of devices of this type currently unallocated. **Every entry is computed independently against the same baseline**, so entries describe mutually exclusive alternatives and must not be summed. See [Partitionable & Consumable Device Accounting](#partitionable--consumable-device-accounting) for the multi-counter-set algorithm. On shareable partitions (`allowMultipleAllocations=true`) this counts only fresh device slots, not capacity headroom on already-in-use devices; operators reading the same pool should consult `shareableSummary.capacity.available` for per-key headroom on shared devices. |
| `pools.shareableSummary` | `*ShareableSummaryStatus` (optional) | Pool-level aggregate for devices with `allowMultipleAllocations=true`. Omitted when the pool has no such devices. Per-device detail was intentionally not included: a per-device list would scale to hundreds of entries on large pools, so the aggregate gives the operator-relevant signal in three small numbers plus a per-capacity-key breakdown. |
| `pools.shareableSummary.fullyAvailableDevices` | `*int32` (required) | Count of shareable devices in the pool with **zero** non-AdminAccess claims. |
| `pools.shareableSummary.partiallyAvailableDevices` | `*int32` (required) | Count of shareable devices with **at least one** non-AdminAccess claim, regardless of how much of their capacity is still free — a fully consumed device is counted here, not excluded. `fullyAvailableDevices + partiallyAvailableDevices` equals the total number of shareable devices in the pool. Tightening this to the field's stated meaning ("some but not all capacity consumed") is a Beta item. |
| `pools.shareableSummary.capacity` | atomic list of `ShareableCapacityStatus`, max 32 (Alpha 1.37) | Per-capacity-key aggregate across all shareable devices in the pool, sorted by key. Cap of 32 matches the per-device combined `Attributes + Capacity` cap (no single device can carry more than 32 capacity keys); aggregation across devices may introduce additional keys but homogeneous-schema pools rarely exceed this. |
| `pools.shareableSummary.capacity.name` | string (required) | Capacity key as it appears in `ResourceSlice.Spec.Devices[].Capacity`. |
| `pools.shareableSummary.capacity.total` | `*resource.Quantity` (required) | Sum of `Device.Capacity[name].Value` across all shareable devices in the pool that carry this key. Devices that do not carry the key contribute nothing (rather than zero), which is the correct behaviour for heterogeneous-schema pools. |
| `pools.shareableSummary.capacity.consumed` | `*resource.Quantity` (required) | Sum of `DeviceRequestAllocationResult.ConsumedCapacity[name]` across **all** non-AdminAccess allocations in the pool, not only those on shareable devices. In practice `ConsumedCapacity` is only set for consumable devices, so the two coincide; the aggregate is keyed off `total`, so a key consumed but carried by no shareable device is not reported at all. |
| `pools.shareableSummary.capacity.available` | `*resource.Quantity` (required) | `total − consumed`, clamped at zero (never negative). |
| `conditions[]` | map list by `type`, max 10 | `Complete` (True when processed) or `Failed` (True on error). |

#### Companion API Change: `ResourceSlice.Spec.PartitionTypeAttribute`

Alpha 1.37 adds one optional field to `ResourceSliceSpec`. Because the
served ResourceSlice versions must stay in sync, it lands in **all three**
external versions (`resource.k8s.io/v1`, `v1beta1`, `v1beta2`) plus the
internal type, as protobuf field 9 immediately after `SharedCounters`:

```go
	// PartitionTypeAttribute names a string device attribute (by fully
	// qualified name, e.g. "gpu.example.com/profile") whose value labels
	// each device with its partition type, such as "Full" or "Half" for a
	// MIG-style GPU.
	//
	// When set, every partitionable device in the slice must carry the attribute
	// and devices sharing a value must share the same ConsumesCounters cost.
	//
	// +optional
	// +featureGate=DRAPartitionableDevicesType
	// +k8s:ifDisabled(DRAPartitionableDevicesType)=+k8s:forbidden
	// +k8s:ifEnabled(DRAPartitionableDevicesType)=+k8s:optional
	// +k8s:ifEnabled(DRAPartitionableDevicesType)=+k8s:format=k8s-resource-fully-qualified-name
	PartitionTypeAttribute *FullyQualifiedName `json:"partitionTypeAttribute,omitempty" protobuf:"bytes,9,opt,name=partitionTypeAttribute"`
```

Example. A driver that publishes one MIG-style GPU per node as three
mutually exclusive partition shapes — Full, Half, Quarter — declares
`PartitionTypeAttribute: gpu.example.com/profile` on each slice in the
pool, and each device sets that attribute to its shape:

```yaml
spec:
  driver: gpu.example.com
  pool:
    name: node-1
    generation: 1
    resourceSliceCount: 1
  partitionTypeAttribute: gpu.example.com/profile
  sharedCounters:
  - name: gpu-0
    counters:
      memory: { value: 80Gi }
  devices:
  - name: gpu-0-full
    attributes:
      gpu.example.com/profile: { string: "Full" }
    consumesCounters:
    - counterSet: gpu-0
      counters: { memory: { value: 80Gi } }
  - name: gpu-0-half-1
    attributes:
      gpu.example.com/profile: { string: "Half" }
    consumesCounters:
    - counterSet: gpu-0
      counters: { memory: { value: 40Gi } }
```

With this declared, the status controller emits a typed
`partitionSummary` entry per profile value (`Full`, `Half`, …) reporting
total and currently-allocatable device counts. The controller also
accepts the domain-stripped bare form of the attribute name on a device
(`profile` on a slice whose driver is `gpu.example.com`), matching how
drivers may abbreviate attributes in their own domain.

##### Where validation happens

Slice-side validation (`validatePartitionTypeAttribute` in
`pkg/apis/resource/validation/validation.go`) enforces the per-slice
rules at admission time:

- The field may only be set on a slice that declares at least one device
  with `ConsumesCounters`. A slice with no counter-consuming devices is
  rejected.
- Every counter-consuming device in the slice must carry the named
  attribute, and its value must be a **string**. Devices that consume no
  counters are exempt.

The remaining rules are inherently cross-slice and are therefore checked
by the resource pool status controller and surfaced as per-pool
`validationError`s rather than rejected at write time:

- Devices of the same partition type publishing different
  `ConsumesCounters` costs (`PartitionCostMismatch:`).
- A pool that resolves a grouping attribute but publishes no
  `sharedCounters` (`PartitionTypeMissing:`).

The slice-level rule is intentionally permissive about `SharedCounters`
themselves: a counter-consuming slice in a multi-slice pool can carry
only `Devices` and a reference to a counter set declared elsewhere in
the pool, so requiring `SharedCounters` on every slice that declares
this field would reject legitimate setups.

##### Feature gate: `DRAPartitionableDevicesType`

This field is **not** gated by `DRAResourcePoolStatus`. It has its own
gate, added in 1.37:

| Gate | Introduced | Stage | Default | Dependencies |
|------|-----------|-------|---------|--------------|
| `DRAPartitionableDevicesType` | 1.37 | Alpha | off | `DynamicResourceAllocation`, `DRAPartitionableDevices`, `DRAResourcePoolStatus` |

A separate gate is needed because the field lives on `ResourceSlice`,
which is served from the GA `resource.k8s.io/v1` group version — its
lifecycle has to be steerable independently of the alpha
ResourcePoolStatusRequest API. The dependency list encodes the two
reasons the field is only meaningful in combination: `SharedCounters`
are themselves gated by `DRAPartitionableDevices`, and
`ResourcePoolStatusRequest` is the only consumer of the grouping
attribute.

The same gate also controls the request-side
`ResourcePoolStatusRequestSpec.DefaultPartitionTypeAttribute`. Both
follow the standard gated-field convention: the registry strategy clears
the field on create and update while the gate is off
(`dropDisabledDRAPartitionableDevicesTypeFields` in the ResourceSlice and
ResourcePoolStatusRequest strategies), ratcheting so that an object which
already carries the field keeps it through subsequent updates. Because the
drop runs before validation, a write that sets the field on a
gate-disabled cluster is silently cleared rather than rejected; the
declarative `+k8s:ifDisabled(DRAPartitionableDevicesType)=+k8s:forbidden`
rule exists as a backstop for paths that bypass the strategy. The status
controller does not consult the gate — it groups on whatever attribute has
been persisted.

See also: [KEP-4815 (Partitionable
Devices)](/keps/sig-scheduling/4815-dra-partitionable-devices) for the
`SharedCounters` / `ConsumesCounters` machinery this field builds on.
### Controller Implementation

#### Controller in KCM

The controller is added to kube-controller-manager as a separate controller
named `resourcepoolstatusrequest-controller` with its own client (so
client-side throttling does not impact scheduling). It is registered in
`cmd/kube-controller-manager/app/resource.go` behind
`requiredFeatureGates: []featuregate.Feature{features.DRAResourcePoolStatus}`.

The controller:
1. Watches ResourcePoolStatusRequest (`resource.k8s.io/v1alpha3`) objects
   via informer. Only add and update events are handled; there is no
   delete handler, and slice / claim / taint-rule churn does not
   re-trigger anything.
2. Maintains a rate-limited work queue for processing, with up to 5 retries
   per request before dropping. Requests are keyed by bare object name,
   since the type is cluster-scoped.
3. Reuses the ResourceSlice, ResourceClaim and DeviceTaintRule informers
   from the stable `resource.k8s.io/v1` group already running in KCM.
4. Uses `UpdateStatus` to write results to the status subresource.
5. Runs a single worker (`controller.Run(ctx, 1)`). There is no
   `ConcurrentSyncs` configuration knob; a request is a one-shot
   read-and-summarise, so one worker has been sufficient. Revisiting this
   is a Beta item if scale testing shows otherwise.

#### One-time Processing

Following the request/status pattern, the controller processes each request
exactly once:

1. When a new ResourcePoolStatusRequest is created, it is added to the work queue.
2. Controller checks if `status` is already non-nil.
3. If non-nil, the request was already processed — controller skips it.
   This check happens both when enqueueing and again inside the sync, so a
   re-listed object is never reprocessed.
4. If nil, controller computes pool status and writes to `status`.
5. Once `status` is written, the request is complete: `status` is frozen
   write-once (validated via `ValidateImmutableField`), and `spec` is
   already immutable from creation (`+k8s:immutable`). Metadata (labels,
   annotations) remains mutable via the main endpoint per standard
   object-meta update rules.

To get fresh data, users delete and recreate the request. (See the TTL
cleanup section below for automatic deletion of old requests.)

#### Incomplete-Pool Handling and Requeue

When the number of ResourceSlices observed for a pool (at the latest
generation) is less than the pool's declared `ResourceSliceCount`, the pool
is considered incomplete:

- The pool's `validationError` is set in the computed status with a
  `PoolIncomplete:`-prefixed message (truncated to 256 bytes), and
  `resourceSliceCount`,
  `totalDevices`, `allocatedDevices`, `availableDevices`,
  `unavailableDevices`, `partitionSummary` and `shareableSummary` are all
  left unset.
- The sync then **returns an error before writing any status**, so the
  request is requeued (up to `maxRetries = 5`) to give drivers time to
  publish the remaining slices.
- If retries are exhausted, the key is forgotten and **no status is ever
  written**. The request stays with `status` unset — no `Complete`
  condition, no `Failed` condition, no metric sample, no event — until the
  24-hour pending TTL deletes it.

`ResourceSliceCount` is read from the first slice observed for the pool and
is not re-read from later slices.

The last point is a known rough edge: a user whose driver never finishes
publishing gets silence rather than a diagnosable object. Giving incomplete
pools a terminal state is tracked as a Beta item.

#### Reusing Existing Informers

The controller reuses the ResourceSlice, ResourceClaim and DeviceTaintRule
informers from the `resource.k8s.io/v1` informer factory already running in
KCM for other DRA controllers (e.g. device-taint-eviction), plus a
`resource.k8s.io/v1alpha3` informer for the requests themselves. This adds
minimal overhead since the shared informers are already cached in memory.
The controller constructor accepts these shared informers rather than
creating its own, following the established KCM pattern. The DeviceTaintRule
lister is only consulted when `DRADeviceTaintRules` is enabled.

#### Partitionable & Consumable Device Accounting

In Alpha 1.36 the controller computed `allocatedDevices` by walking
each `ResourceClaim.Status.Allocation.Devices.Results` and incrementing
a per-device counter. That arithmetic is correct for plain devices but
wrong for two API shapes the broader DRA stack supports:

- A single physical device can appear as multiple mutually-exclusive
  partitions that draw from a shared `CounterSet`
  (`DRAPartitionableDevices`).
- A device with `allowMultipleAllocations=true` can be reserved by
  many claims simultaneously, each consuming part of its capacity
  (`DRAConsumableCapacity`).

This work depends on `DRAPartitionableDevices`
([KEP-4815](/keps/sig-scheduling/4815-dra-partitionable-devices),
Beta in 1.36) and `DRAConsumableCapacity`
([KEP-5075](/keps/sig-scheduling/5075-dra-consumable-capacity),
Beta in 1.36). Both are Beta default-on by the time Alpha 1.37 ships,
so the fields the controller reads (`SharedCounters`,
`ConsumesCounters`, `AllowMultipleAllocations`, `ConsumedCapacity`) are
part of the served `resource.k8s.io/v1` surface. When either gate is
disabled on a cluster, the corresponding sub-object is omitted from the
response — the source fields are nil on incoming `ResourceSlice`
objects, the aggregation produces no entries (the slice stays nil
rather than being initialised to an empty `[]`), and `omitempty` keeps
the common-case payload shape unchanged.

Alpha 1.37 changed the aggregation to handle all three shapes
consistently:

1. **Per-device cap on `allocatedDevices`.** Allocation results are
   collected into a per-pool set keyed by device name, so a physical
   device is counted at most once regardless of how many
   non-AdminAccess claims reference it. This fixes the consumable
   overcount in Alpha 1.36 (where N claims on one device added N to
   the tally). The set is built from all claims without filtering on
   pool generation, so a stale claim referencing a device that no
   longer exists at the current generation still contributes.
2. **AdminAccess allocations are skipped** in every device, counter,
   and shareable-device tally. They are observers, not consumers,
   and counting them misleads administrators about real availability.
3. **`unavailableDevices`** is the count of devices with at
   least one `NoSchedule` or `NoExecute` taint (sourced from
   `ResourceSlice.Spec.Devices[].Taints` and any `DeviceTaintRule`
   matches), replacing the Alpha 1.36 hard-coded `0`. Taint-rule
   matching is deliberately narrow: only the `Driver`, `Pool` and
   `Device` selector fields are evaluated, the CEL `Selectors` and
   `DeviceClassName` fields are not, and a nil selector matches
   *nothing* (the inverse of the device-taint-eviction convention,
   chosen so an unscoped rule cannot silently zero out a pool's
   availability). External `DeviceTaintRule` matching is skipped
   entirely when `DRADeviceTaintRules` is disabled; embedded
   `Spec.Devices[].Taints` still contribute. Because the count is taken
   over all devices in the pool regardless of allocation, a device that
   is both allocated and tainted is subtracted twice from
   `availableDevices` — see the Beta criteria.
4. **`partitionSummary`** is emitted when the pool publishes
   `sharedCounters` **and** a grouping attribute can be resolved for
   at least one device. Resolution works pool-wide, not per device:

   - If **any** slice in the pool declares
     `ResourceSlice.Spec.PartitionTypeAttribute`, that declaration
     governs. Devices whose own slice declares nothing are left
     ungrouped, and the request's `defaultPartitionTypeAttribute` is
     ignored for the whole pool. This keeps a pool from being bucketed
     under two different attributes by accident.
   - If **no** slice in the pool declares one, the request's
     `spec.defaultPartitionTypeAttribute` (when set) is applied to every
     device in the pool.
   - If neither source names an attribute, the pool reports **no**
     `partitionSummary` at all.

   Each entry self-describes the attribute it was resolved from, so a
   pool whose slices declare different attributes reports each group
   independently and the reader can tell them apart. Attribute lookup on
   a device accepts either the fully qualified name or the
   domain-stripped bare name within the driver's own domain. Devices
   that consume no counters are not partitions and are excluded from the
   grouping.

   Per group G (identified by an `(attribute, type)` pair) the
   controller computes:
   - `total[G]` = count of devices in the pool in that group.
   - `fresh[G]` = devices in G currently unallocated (no non-AdminAccess
     claim references them).
   - `cost[G]` = the canonical per-device `ConsumesCounters` profile for
     the group, validated as homogeneous; mixed costs produce a per-pool
     `PartitionCostMismatch:` `validationError`.
   - `counterAvailable[s][c]` = `SharedCounters[s].Counters[c].Value`
     minus the sum of `d.ConsumesCounters[s][c]` over **unique in-use**
     non-AdminAccess devices `d` in the pool that consume from counter
     set `s`. Each in-use device is counted once regardless of how
     many claims reference it; this matches the scheduler's counter
     accounting for shareable partitions, where the counter check is
     skipped on subsequent allocations of an
     `allowMultipleAllocations=true` device.
   - `allocatable[G]` by a greedy per-device fit check: iterate fresh
     devices `d` in G, and for each one check that for every counter set
     `s` and counter `c` in `d.ConsumesCounters`,
     `counterAvailable[s][c] >= cost[G][s][c]`. If the check passes,
     increment `allocatable[G]` and deduct `cost[G]` from
     `counterAvailable[s]` so subsequent siblings drawing from the same
     counter set are accounted correctly. For the common case where every
     device in G consumes from a single counter set `s_G` with the same
     cost, this reduces to
     `allocatable[G] = min(fresh[G], min over c of floor(counterAvailable[s_G][c] / cost[G][c]))`.
     The fresh-device cap matters when counter headroom exceeds the
     supply of unallocated devices (otherwise the scalar would
     advertise impossible allocations).

   **Every group is evaluated independently against the same baseline**
   — the counter state is cloned per group rather than carried across
   groups. Entries therefore describe mutually exclusive alternatives
   ("you could allocate 1 more Full, *or* 2 more Half") and must not be
   summed.

   On shareable partitions (`allowMultipleAllocations=true`),
   `allocatable[G]` counts only fresh device slots; capacity headroom
   remaining on already-in-use devices is published separately under
   `shareableSummary.capacity`. Operators on hybrid pools should read
   both fields.

   A pool that resolves a grouping attribute but publishes no
   `sharedCounters` produces a `PartitionTypeMissing:` `validationError`,
   as does a grouped device that does not carry the resolved attribute.
   Any such error clears `partitionSummary` for that pool; the device
   counts computed in steps 1–3 remain valid and are still reported.
5. **`shareableSummary`** is emitted when the pool has any device
   with `allowMultipleAllocations=true`. The controller scans all
   shareable devices in the pool and produces three fields:
   `fullyAvailableDevices` (devices with zero non-AdminAccess claims),
   `partiallyAvailableDevices` (devices with at least one non-AdminAccess
   claim — membership in the in-use set, not a measure of remaining
   capacity, so a fully consumed device counts here too), and
   `capacity[]` — a per-capacity-key aggregate where each entry sums
   `Device.Capacity[name].Value` over the shareable devices carrying the
   key (`total`) and `DeviceRequestAllocationResult.ConsumedCapacity[name]`
   over all non-AdminAccess allocations in the pool (`consumed` — the
   consumption map is accumulated per pool, not per device class), with
   `available = total − consumed` clamped at zero. A per-device
   array would scale to hundreds of entries on large pools; the
   aggregate gives the operator-relevant signal far more compactly.
   Heterogeneous-schema pools are handled by the rule "devices that
   do not carry a given key contribute nothing to that key's total"
   — the aggregate stays correct rather than reporting zeros that
   would misrepresent capacity.

Both views are computed only for pools that pass the completeness check;
an incomplete pool reports neither.

`availableDevices` keeps its existing definition
(`totalDevices − allocatedDevices − unavailableDevices`). On plain
pools it is the operationally useful "how many more claims fit"
signal. **On pools with shared counters or shareable devices it is
not.** Two cases the operator must understand:

- **Partitionable pools.** When the bottleneck is a shared
  `CounterSet`, all device entries can be unallocated yet no further
  claim can fit — `availableDevices` will read high while no
  partition actually fits. Operators must consult
  `partitionSummary[G].allocatable`, which is the canonical signal:
  how many more devices of that partition type still fit under current
  counter state. On a pool for which no grouping attribute could be
  resolved, no such signal is published — which is the practical reason
  drivers are encouraged to declare `PartitionTypeAttribute`, and why
  `spec.defaultPartitionTypeAttribute` exists as a client-side escape
  hatch for drivers that have not yet done so.
- **Consumable / shareable pools.** With the cap-at-1 rule, every
  device with at least one claim is counted once in
  `allocatedDevices`. A pool of N shareable devices each holding
  one tiny claim will report `allocatedDevices=N` and
  `availableDevices=0`, even though most of each device's capacity
  is free. Operators must consult
  `shareableSummary.capacity.available` vs `.total` for the
  remaining-capacity signal, and the
  `fullyAvailableDevices`/`partiallyAvailableDevices` counts for the
  share-pattern signal.

This is documented as a deliberate trade-off: `availableDevices`
remains a stable scalar that older clients can use, and the new
sub-objects carry the precise truth. The KEP does not redefine
`availableDevices` per pool shape because doing so would silently
change its meaning for existing 1.36 consumers.

Both sub-objects are omitted when empty so plain pools stay compact.
`partitionSummary` carries `+k8s:maxItems=32` and
`shareableSummary.capacity` carries `+k8s:maxItems=32` (both
provisional — revisit at Beta). Pools larger than either cap produce a
per-pool `validationError` (`PartitionSummaryOverCap:` /
`ShareableSummaryOverCap:`) rather than silent truncation; the
controller measures size before populating the field and writes the
`validationError` directly, avoiding a rejected write against the
apiserver.

##### Conditions and Metrics

Exactly one condition is written per request, and the status is written
exactly once:

- `Complete` / `True` with reason `CalculationComplete` and message
  `Calculated status for N pools`, or
  `Calculated status for N pools (M incomplete)` when any pool carries a
  `validationError`. The wording is misleading: because a status is never
  written while any pool is incomplete, `M` can only ever count
  **view-level** errors (`PartitionTypeMissing:`, `PartitionCostMismatch:`,
  the two over-cap cases) — never an actually incomplete pool. It is also
  counted over the post-truncation list, so pools dropped by `limit` are
  not reflected. `observedGeneration` is not set.
- `Failed` / `True` with reason `CalculationFailed` and a message naming
  the failed list call. This is only reachable from a ResourceSlice,
  ResourceClaim or DeviceTaintRule lister failure.

The three metrics are recorded only on the two code paths that reach
`UpdateStatus`. In particular
`resourcepoolstatusrequest_controller_request_processing_errors_total`
counts `UpdateStatus` failures only — a lister failure produces a
`Failed` condition but no error sample — and the incomplete-pool requeue
path records nothing at all. Widening metric coverage is a Beta item.

##### Devices That Are Both Partitionable and Consumable

When a single physical device is both partitionable and consumable —
its partitions draw from a shared `CounterSet` *and* individual
partitions allow multiple allocations — consuming capacity on one
partition can make sibling partitions unallocatable through the shared
counter, even though those siblings still appear as unconsumed
devices.

The typed `partitionSummary` view addresses this directly: because
`allocatable[G] = min(fresh[G], min over c of floor(counterAvailable[c] / cost[G][c]))`
reads the *current* `counterAvailable` after each in-use device's
static `ConsumesCounters` has been subtracted, sibling partitions
blocked by a shared counter are already netted out. Concretely: a
device offered as one full partition (80Gi) or two half partitions
(40Gi each), all backed by one 80Gi `CounterSet`. Allocate the full
partition and the counter is fully consumed (80Gi cost, one in-use
device). The controller reports
`partitionSummary[full].allocatable = 0` (no counter headroom) and
`partitionSummary[half].allocatable = 0` (no counter headroom) —
exactly the bound operators need.

Two residual cases worth calling out:

- **Pools for which no grouping attribute could be resolved.**
  `availableDevices` is not netted out against shared counters, and
  `shareableSummary` reports raw device-capacity aggregates that do not
  subtract counter-blocked siblings. Such a pool publishes no precise
  counter signal at all. The fix is on the request or driver side:
  set `spec.defaultPartitionTypeAttribute`, or have the driver declare
  `ResourceSlice.Spec.PartitionTypeAttribute`.
- **Capacity headroom on shareable in-use partitions
  (`allowMultipleAllocations=true`).** The scheduler debits
  `ConsumesCounters` exactly once per device — subsequent claims
  against the same shareable device do not consume more counter
  capacity — and `partitionSummary[G].allocatable` follows the same
  rule (it counts only how many additional *fresh* devices of that
  group can be allocated). What it does **not** capture is how much
  per-claim capacity is still available on devices that are already
  in use. For that, operators on hybrid pools must read
  `shareableSummary.capacity`, which reports the remaining free
  capacity per key across all shareable devices in the pool.

The contract is: when `partitionSummary` is emitted, `allocatable` is
the precise bound on fresh-device allocations, and `shareableSummary` is
the precise bound on remaining capacity on already-shared devices. When
no grouping attribute resolves, the operator has no precise
counter-level signal — a gap that is closed by declaring the attribute
on either side.

#### TTL-Based Cleanup

The controller runs a cleanup loop every 10 minutes that deletes stale
ResourcePoolStatusRequest objects:

| State | TTL | Measured from |
|-------|-----|---------------|
| Completed / Failed (status set) | 1 hour | `LastTransitionTime` of `Complete`/`Failed` condition |
| Pending (status nil) | 24 hours | `CreationTimestamp` |

Deletion uses a UID precondition to avoid racing with user recreates. This
bounds etcd growth without requiring user cleanup and is implemented in
Alpha (earlier than originally planned for Beta).

#### Controller RBAC

The controller's ClusterRole `system:controller:resourcepoolstatusrequest-controller`
grants:

- `get`, `list`, `watch`, **`delete`** on `resourcepoolstatusrequests`
  (delete needed for TTL cleanup)
- `update`, `patch` on `resourcepoolstatusrequests/status`
- `get`, `list`, `watch` on `resourceslices`, `resourceclaims` and
  `devicetaintrules`
- standard events permissions

The ClusterRole is only installed when `DRAResourcePoolStatus` is enabled.
The events grant is currently unused — the controller has no event recorder
and emits no events — and either wiring up events or dropping the grant is a
Beta clean-up item.

### kubectl Integration

Standard kubectl commands work against the singular resource name
`resourcepoolstatusrequest` (plural `resourcepoolstatusrequests`). The
implementation also registers custom table columns so `kubectl get` returns
a useful summary view:

| Column | Source |
|--------|--------|
| Name | `metadata.name` |
| Driver | `spec.driver` |
| Total | `sum(status.pools[].totalDevices)` |
| Available | `sum(status.pools[].availableDevices)` |
| Allocated | `sum(status.pools[].allocatedDevices)` |
| Unavailable | `sum(status.pools[].unavailableDevices)` |
| Errors | count of pools with `validationError` |
| Pools | `status.poolCount` |
| Status | `Pending` / `Complete` / `Complete (m/n pools)` if truncated / `Failed` |
| Completed | Age derived from `Complete`/`Failed` condition `lastTransitionTime` |

```bash
# Create request
$ kubectl create -f request.yaml

# Wait for completion
$ kubectl wait --for=condition=Complete resourcepoolstatusrequest/my-request

# Get status
$ kubectl get resourcepoolstatusrequest/my-request -o yaml

# List all requests
$ kubectl get resourcepoolstatusrequests

# Delete request (or let the TTL sweeper delete it 1h after completion)
$ kubectl delete resourcepoolstatusrequest/my-request
```

No short name (e.g. `rpsr`) and no resource category are registered in
Alpha; adding them is a follow-up for Beta.

There is also no `kubectl describe` support. Because the table columns are
cluster-wide sums, the per-pool detail that is the whole point of the API —
`status.pools[]`, `partitionSummary`, `shareableSummary` and
`validationError` — is only reachable via `-o yaml`/`-o jsonpath`. Adding a
describer is the largest remaining UX gap and is a Beta requirement.

### Test Plan

#### Prerequisite testing updates

None required.

#### Unit tests

Coverage locations:
- Pool status computation (`pkg/controller/resourcepoolstatusrequest/controller_test.go`)
- Partition / shareable view computation and attribute resolution
  (`pkg/controller/resourcepoolstatusrequest/views_test.go`)
- Validation (`pkg/apis/resource/validation/validation_resourcepoolstatusrequest_test.go`)
- Declarative validation equivalence
  (`test/declarative_validation/resource/resourcepoolstatusrequest/declarative_validation_test.go`)
- Registry strategy (`pkg/registry/resource/resourcepoolstatusrequest/strategy.go`
  → `strategy_test.go`)
- Metrics (`pkg/controller/resourcepoolstatusrequest/metrics/metrics_test.go`)
- Printer columns (`pkg/printers/internalversion/printers_test.go`)

Test cases (Alpha 1.36):
- Driver only (all pools for that driver)
- Driver and pool name filter
- No matching pools for driver
- Missing driver field (validation error)
- Various allocation states
- Incomplete pools (observed slice count < expected) cause a requeue and
  no status write, both before and after retries are exhausted
  (`TestSyncRequestRequeuesIncompletePool` asserts no status update is
  issued in either case)
- Permanent view-level errors *are* written to status
  (`TestSyncRequestWritesStructuralViewError`)
- Older-generation slices ignored (generation handling)
- Mixed / multi-node pools leave `nodeName` unset
- One-time processing (skip if `status != nil`)
- Spec / metadata immutability after status is set
- TTL cleanup: completed (1h) and pending (24h) requests deleted
- `limit` respected; `poolCount` reflects total matches

Added in Alpha 1.37:
- **Cap-at-1 for shareable devices**: a single device with
  `allowMultipleAllocations=true` and several concurrent claims
  contributes exactly `1` to `allocatedDevices`.
- **AdminAccess skipped**: an AdminAccess allocation against a
  device does not increment `allocatedDevices`. Because AdminAccess is
  filtered out while building the shared in-use and consumed-capacity
  maps, the same exclusion carries into
  `shareableSummary.partiallyAvailableDevices`,
  `shareableSummary.capacity.consumed` and
  `partitionSummary.allocatable`; asserting that explicitly at the view
  level is a Beta test-coverage item.
- **`unavailableDevices` from taints**: a pool with `M` devices,
  `K` of which carry a `NoSchedule` or `NoExecute` taint (via
  `Spec.Devices[].Taints` or a matching `DeviceTaintRule`), reports
  `unavailableDevices=K`. A `None`-effect taint — and any unrecognised
  effect — leaves the device available. The `DeviceTaintRule` branch is
  covered in both states, with `DRADeviceTaintRules` explicitly enabled
  and explicitly disabled.
- **Attribute resolution** (`TestResolveDevicePartitions`,
  `TestResolvePartitionType`): a slice-declared
  `PartitionTypeAttribute` wins pool-wide and suppresses the request
  default; the request's `defaultPartitionTypeAttribute` applies only
  when no slice declares one; the domain-stripped bare attribute name
  resolves; a non-string or absent attribute value does not group.
- **View gating** (`TestComputePoolViews_PartitionAttributeGatesView`):
  a pool with `sharedCounters` and no resolvable attribute emits no
  `partitionSummary`; a pool that resolves an attribute but publishes no
  `sharedCounters` emits a `PartitionTypeMissing:` `validationError`.
- **`partitionSummary` aggregation**: a pool whose slice declares
  `sharedCounters: [{name: gpu-0, counters: {memory: {value: 80Gi}}}]`,
  a partition-type attribute, and three device entries (1 `full`
  consuming 80Gi, 2 `half` consuming 40Gi each; all drawing from
  `gpu-0`). Walk four states using
  `allocatable[G] = min(fresh[G], min over c of floor(counterAvailable[c] / cost[G][c]))`:
  - Nothing in use → `freshFull=1, freshHalf=2, counterAvailable=80Gi`
    → `allocatable[full]=1, allocatable[half]=2`. Note the entries are
    alternatives, not a sum.
  - One half in use → `freshFull=1, freshHalf=1, counterAvailable=40Gi`
    → `allocatable[full]=0, allocatable[half]=1`.
  - The full partition in use → `freshFull=0, freshHalf=2, counterAvailable=0Gi`
    → `allocatable[full]=0, allocatable[half]=0`.
  - **Fresh-device cap binds first** (counter has more headroom than
    devices): same pool with `counters: {memory: {value: 800Gi}}`,
    nothing in use → `allocatable[full]=1, allocatable[half]=2`
    (not 10/20). Confirms the `fresh[G]` clamp.
- **Multi-counter-set pools** (`TestComputePartitionSummary_MultiCounterSet`):
  a group whose devices draw from more than one counter set is bounded
  by the tightest counter across all of them.
- **Per-device counter consumption (cap-at-1 for counters)**: a pool
  with one shareable partition (`allowMultipleAllocations=true`,
  consuming 40Gi from `gpu-0`) reserved by 3 concurrent non-AdminAccess
  claims computes `partitionSummary` against
  `counterAvailable = capacity − 40Gi`, not `capacity − 120Gi`,
  matching the scheduler's per-device counter accounting.
- **`partitionSummary` validation**: a group resolving to two devices of
  partition type `full` with different `ConsumesCounters` costs (one
  80Gi, the other 60Gi) produces a `PartitionCostMismatch:`
  `validationError` instead of an inconsistent `allocatable`; a grouped
  device that does not carry the resolved attribute produces
  `PartitionTypeMissing:`. Either error clears `partitionSummary` while
  leaving the device counts populated.
- **`shareableSummary` aggregation**: a pool with three devices
  (`nic-0`, `nic-1`, `nic-2`, all `allowMultipleAllocations=true`,
  each with `bandwidth=10Gi`), where `nic-0` has two claims totalling
  7Gi, `nic-1` has one claim of 2Gi and `nic-2` has no claims,
  reports
  `shareableSummary = {fullyAvailableDevices: 1, partiallyAvailableDevices: 2,
  capacity: [{name: bandwidth, total: 30Gi, consumed: 9Gi, available: 21Gi}]}`.
- **`shareableSummary` heterogeneous-schema handling**: a pool with
  two devices that carry different capacity keys (`nic-a` has
  `bandwidth=10Gi`, `nic-b` has `packets-per-sec=1M`) produces a
  `capacity[]` with two entries; each entry's `total` only sums
  the device(s) that carry that key.
- **Hybrid pools** (`TestComputePoolViews_Hybrid`): a pool that is both
  partitionable and consumable populates both views consistently.
- **Both sub-objects omitted on plain pools**: a pool with no
  `sharedCounters` and no `allowMultipleAllocations=true` device
  produces a `PoolStatus` with `partitionSummary` and
  `shareableSummary` both absent.
- **`+k8s:maxItems` over-cap**: a pool with >32 distinct partition types
  or >32 distinct capacity keys yields the corresponding
  `validationError` rather than silent truncation. The controller
  measures size before populating the field, avoiding a rejected write
  against the apiserver.

Gaps carried into Beta: there is no `storage_test.go` for
`pkg/registry/resource/resourcepoolstatusrequest/storage`, and the
strategy unit tests currently cover only partition-attribute handling.
Both are listed under [Beta](#beta).

#### Integration tests

Located at `test/integration/dra/resourcepoolstatusrequest_test.go`. These
verify controller behavior end-to-end against a real apiserver (started
with `--runtime-config=resource.k8s.io/v1alpha3=true`) and a real
controller, with in-memory driver data.

Implemented cases:
1. `ProcessRequest` — controller starts, watches requests, processes new
   ones, and populates status with correct pool data.
2. `OneTimeProcessing` — processed requests are skipped.
3. `LimitTruncation` — `limit` respected and truncation reflected via
   `poolCount` vs `len(pools)`.
4. `FilterByPoolName` — pool-name filter applied.
5. `ValidationErrors` — apiserver admission validation, not per-pool
   status: a missing `driver` and out-of-range `limit` values are
   rejected at create time. Per-pool `validationError` reporting is
   covered by unit tests, not here.
6. `PartitionSummary` — a pool with `sharedCounters`, a declared
   partition-type attribute, and devices that consume counters reports
   the expected per-group `total` / `allocatable`.
7. `ShareableSummary` — a pool with `allowMultipleAllocations=true`
   devices and claims that set `consumedCapacity` reports the expected
   `fullyAvailableDevices`, `partiallyAvailableDevices` and per-key
   `capacity[]`, with `allocatedDevices` capped at 1 per device.

The suite runs a single `feature-enabled` matrix entry with
`DynamicResourceAllocation`, `DRAResourcePoolStatus`,
`DRAPartitionableDevices`, `DRAPartitionableDevicesType` and
`DRAConsumableCapacity` all on. Storage-path coverage lives separately in
`test/integration/etcd/data.go`.

Not yet covered, and therefore listed under [Beta](#beta): a
feature-gate-disabled matrix entry, RBAC enforcement (the test server
runs with `AlwaysAllow`), and a scale case at ≥100 pools with ≥1000
expired requests.

#### e2e tests

E2E tests live in the existing DRA e2e suite at `test/e2e/dra/dra.go`,
using the existing test-driver (`test/e2e/dra/test-driver/`) behind
`f.WithFeatureGate(features.DRAResourcePoolStatus)`.

Implemented (Alpha 1.36):
1. Conformance-style resource lifecycle (create / get / list / watch /
   update / patch / delete) for `resource.k8s.io/v1alpha3
   ResourcePoolStatusRequest`, asserting spec immutability via
   label-only updates.
2. "should report pool status with correct device counts": create a
   request, wait for the `Complete` condition, and assert that the single
   `network` pool reports `totalDevices=10`, `allocatedDevices=0`,
   `availableDevices=10`, `unavailableDevices=0`, `resourceSliceCount=1`,
   `generation=1`, `nodeName=nil`.
3. "should reflect allocated devices after pod is scheduled": schedule a
   pod that consumes devices, then create a new request and assert the
   updated `allocatedDevices` / `availableDevices`.
4. Filter and shape cases: "should populate status for all matching
   pools", "should filter by pool name", "should truncate results when
   limit is reached", "should return empty pools for unknown driver",
   "should not reprocess after status is set" (asserted via unchanged
   resourceVersion), "should set NodeName when all slices share the same
   node", "should clear NodeName when pool has slices on different
   nodes".

Added in Alpha 1.37 (the "control plane views" context):
5. "should report partitionSummary for a partitionable pool" (with
   `DRAPartitionableDevices` and `DRAPartitionableDevicesType`): seed the
   test driver to publish a pool with `sharedCounters`, a declared
   partition-type attribute, and devices that consume counters; assert
   `partitionSummary` shows the expected `total` / `allocatable` per
   group.
6. "should report no partition summary when no partition type is
   declared" (with `DRAPartitionableDevices` only): the same pool without
   a declared attribute publishes no `partitionSummary`.
7. "should report shareableSummary for a pool with shareable devices"
   (with `DRAConsumableCapacity`): assert `fullyAvailableDevices`,
   `partiallyAvailableDevices`, the per-key `capacity[]` aggregate, and
   `allocatedDevices` (cap-at-1 verified end-to-end).

Not yet covered, and listed under [Beta](#beta): the `UpdateStatus`
endpoint in the CRUD block (an explicit TODO in `dra.go` notes it must be
added before graduation), and an AdminAccess-invisibility case at the e2e
or integration level (currently unit-tested only, and only against the
device counts).

Note: Testing with production DRA drivers (e.g., GPU drivers) is outside
the scope of CI and is validated separately by driver vendors.

### Graduation Criteria

#### Alpha (1.36)

- API defined and implemented in `resource.k8s.io/v1alpha3`
- Controller added to kube-controller-manager behind feature gate
  `DRAResourcePoolStatus` (default off), gated on
  `DynamicResourceAllocation`
- Basic kubectl workflow works, including custom table columns
- Unit, integration, and e2e tests (including conformance-style resource
  lifecycle) passing in CI
- Automatic TTL cleanup of completed (1h) and pending (24h) requests —
  moved to Alpha to bound etcd growth without requiring user cleanup
- Controller-side requeue for pools whose slices are not fully published,
  with a per-pool `validationError` computed for them
- Full object immutability once `status` is set
- Documentation

#### Alpha (1.37)

A second Alpha cycle was taken instead of an immediate Beta
graduation. The reasoning, strongest first:

1. **The 1.36 API did not correctly describe partitionable or
   consumable devices.** The 1.36 controller incremented
   `allocatedDevices` per allocation result, which (a) overcounted
   on devices with `allowMultipleAllocations=true` (consumable) and
   (b) did not reflect shared-counter consumption on partitionable
   devices. The visible symptom was `availableDevices=0` reported on
   pools that actually had free capacity. Fixing this required new
   API fields (`partitionSummary`, `shareableSummary`) plus a new
   optional field on `ResourceSlice` (`PartitionTypeAttribute`) to
   drive the typed view — not just a controller patch — and adding
   new API surface in Beta is exactly what Alpha cycles exist to
   avoid.
2. **No production DRA driver had been validated against.**
   The Beta criteria require out-of-tree validation against at least
   one production DRA driver; no driver-side code change is needed
   (the controller reads existing ResourceSlice / allocation fields),
   but operational validation in a real-driver environment cannot be
   back-filled inside the same release that graduates to Beta.
3. **Several Alpha reviewer follow-ups were open** (batched TTL
   deletes, deterministic metrics tests, e2e assertion tightening).
4. **Limited soak.** Alpha shipped in 1.36
   (kubernetes/kubernetes#137028); only one release had elapsed.

What shipped in 1.37 (kubernetes/kubernetes#140170):

- API stayed at `resource.k8s.io/v1alpha3`; feature gate
  `DRAResourcePoolStatus` stayed Alpha, default off.
- **`ResourceSlice.Spec.PartitionTypeAttribute`** — a new optional
  `*FullyQualifiedName` on `ResourceSliceSpec` (protobuf field 9), so
  drivers can declare a grouping attribute for partition types. It
  landed in `resource.k8s.io/v1`, `v1beta1` and `v1beta2` together,
  since the served ResourceSlice versions must stay in sync.
- **A second feature gate, `DRAPartitionableDevicesType`** (1.37,
  Alpha, default off; depends on `DynamicResourceAllocation`,
  `DRAPartitionableDevices` and `DRAResourcePoolStatus`) to gate that
  slice field and its request-side counterpart independently of the
  alpha status API.
- **`spec.defaultPartitionTypeAttribute`** on
  `ResourcePoolStatusRequestSpec` — a request-side grouping attribute
  used when no slice in the pool declares one, so a client can get an
  accurate `partitionSummary` from a driver that has not yet adopted
  the slice-side declaration.
- **`partitionSummary` on `PoolStatus`** — a list of
  `PartitionTypeStatus` entries keyed by `(attribute, type)`, each
  reporting `total` and `allocatable`, netting out shared-counter
  consumption.
- **`shareableSummary` on `PoolStatus`** — `fullyAvailableDevices`,
  `partiallyAvailableDevices`, and per-capacity-key
  `total`/`consumed`/`available` aggregates for pools containing at
  least one `allowMultipleAllocations=true` device.
- **Per-device cap on `allocatedDevices`**, fixing the consumable
  overcount.
- **AdminAccess allocations skipped** in all device, counter and
  shareable-device tallies.
- **`unavailableDevices` computed from real device taints**
  (`NoSchedule` / `NoExecute`, embedded and via `DeviceTaintRule`),
  replacing the Alpha 1.36 hard-coded `0`.
- Controller migrated to the `resource.k8s.io/v1` `DeviceTaintRule`
  API.
- Unit, integration and e2e coverage for all of the above.

Two items in the original 1.37 plan changed during API review and are
recorded here because they are visible in the shipped shape:

- **The `counterSets` fallback view was dropped.** The plan was for
  pools without a declared grouping attribute to receive a raw
  per-`CounterSet` capacity dump. Review preferred not to add a
  second, verbose status shape for the same question; the request-side
  `defaultPartitionTypeAttribute` covers the "driver has not adopted
  the convention yet" case, and pools with no resolvable attribute
  simply report no partition view.
- **`partitionSummary` entries carry the attribute they were grouped
  by.** The plan assumed one grouping attribute per pool. The shipped
  list is keyed by `(attribute, type)` and unique on that pair, so a
  pool whose slices declare different attributes reports each group
  independently rather than being rejected.

Two items from the Alpha reviewer follow-ups were **not** completed in
1.37 and carry into Beta: batched / paced TTL-delete sweeps, and scale
validation at ≥100 pools with ≥1000 expired requests. The metrics-test
follow-up was addressed — the tests now compare gathered output with the
timing-dependent histogram fields stripped — though they still exercise
the metric objects rather than the controller's emission paths.

#### Beta

Beta criteria will be revisited after the Alpha 1.37 work lands
(`partitionSummary` / `counterSets` / `shareableSummary`,
`ResourceSlice.Spec.PartitionTypeAttribute`, `unavailableDevices`,
cap-at-1, AdminAccess skip) and the feature has soaked across the
1.36 + 1.37 cycles. The target milestone and API-version graduation
plan are intentionally left open at this point.

#### GA

- At least 2 releases as beta
- Validated at scale (1000+ pools)
- kubectl plugin for better UX (optional)
- Documentation complete

### Upgrade / Downgrade Strategy

**Upgrade (Alpha 1.36 → Alpha 1.37):**
- Feature gate stays Alpha, default off — no behavioural change for
  clusters that do not opt in.
- API stays at `resource.k8s.io/v1alpha3` for the status object.
  Stored objects from 1.36 remain readable; the new optional fields
  (`partitionSummary`, `shareableSummary`, and the request's
  `defaultPartitionTypeAttribute`) are populated by the 1.37 controller
  when the source data warrants it. Older clients ignore the unknown
  fields.
- `ResourceSlice.Spec.PartitionTypeAttribute` (new in
  `resource.k8s.io/v1`, `v1beta1` and `v1beta2`, gated behind the new
  `DRAPartitionableDevicesType` gate) is an additive optional field.
  Slices written by 1.36 leave it unset, so pools published by an
  un-updated driver report no `partitionSummary` unless the request
  supplies `spec.defaultPartitionTypeAttribute`. Drivers that adopt the
  convention opt in slice by slice; because a single declaration
  anywhere in the pool governs pool-wide, partial adoption leaves the
  undeclared devices ungrouped rather than mixing views.
- The change to `allocatedDevices` semantics (cap at 1 per physical
  device) is a behavioural change, not an API change. It will be
  called out in 1.37 release notes because Alpha 1.36 clients that
  scripted around the inflated counts will see different numbers.

**Downgrade (disable feature gate):**
- Disable `DRAResourcePoolStatus` on both kube-apiserver and
  kube-controller-manager.
- Existing `ResourcePoolStatusRequest` objects become inaccessible,
  but no workload impact.
- No persistent state outside these objects, so downgrade does not
  require a data migration.

### Version Skew Strategy

- **kube-apiserver and kube-controller-manager** must both have
  `DRAResourcePoolStatus` enabled. The gate is Alpha (default off) in
  both 1.36 and 1.37, so both components must opt in explicitly. The
  API also lives in `resource.k8s.io/v1alpha3`, which is disabled by
  default, so the apiserver additionally needs
  `--runtime-config=resource.k8s.io/v1alpha3=true`.
- **1.36 ↔ 1.37 skew:** Status API is `resource.k8s.io/v1alpha3` in
  both releases. In the supported direction, a 1.36 KCM against a 1.37
  apiserver simply does not populate the fields added in 1.37; readers
  see a `PoolStatus` without `partitionSummary` or `shareableSummary`,
  which is indistinguishable from a pool that has neither.
- **`ResourceSlice.Spec.PartitionTypeAttribute` skew:** the field
  lives in served `resource.k8s.io/v1` (and `v1beta1` / `v1beta2`).
  A 1.37 apiserver with `DRAPartitionableDevicesType` disabled
  (default) drops the field on write, so drivers that set it on a
  gate-disabled cluster see it silently cleared — the same shape as
  other gated optional fields; objects that already carry it ratchet
  through updates unchanged. A 1.36 apiserver does not know the field
  and drops it as an unknown field. In both cases the controller sees
  the attribute unset and simply publishes no `partitionSummary` for
  that pool unless the request names a default.
- **Older kubectl** can create/read objects via the standard
  `v1alpha3` endpoint without changes.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: DRAResourcePoolStatus
  - Components: kube-apiserver, kube-controller-manager
- [x] Feature gate
  - Feature gate name: DRAPartitionableDevicesType
  - Components: kube-apiserver

`DRAResourcePoolStatus` gates the API type, its storage, the controller and
the controller's bootstrap ClusterRole. It depends on
`DynamicResourceAllocation`.

`DRAPartitionableDevicesType` gates `ResourceSlice.Spec.PartitionTypeAttribute`
and `ResourcePoolStatusRequestSpec.DefaultPartitionTypeAttribute` — the two
ways a grouping attribute reaches the controller. It depends on
`DynamicResourceAllocation`, `DRAPartitionableDevices` and
`DRAResourcePoolStatus`. Those dependencies are enforced: enabling
`DRAPartitionableDevicesType` while any of them is disabled makes
feature-gate validation fail at component start, rather than being silently
ignored. Disabling it while `DRAResourcePoolStatus` is on simply means
partitionable pools report no `partitionSummary`. It is enforced only in the
apiserver — the controller does not consult it and groups on whatever
attribute has been persisted.

Because the API lives in `resource.k8s.io/v1alpha3`, which is disabled by
default, the apiserver also needs
`--runtime-config=resource.k8s.io/v1alpha3=true`.

###### Does enabling the feature change any default behavior?

No. Users must explicitly create ResourcePoolStatusRequest objects.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disable the feature gate. Existing requests become inaccessible but
no workload impact.

###### What happens if we reenable the feature if it was previously rolled back?

Existing requests (if any) become visible again. Unprocessed requests will
be processed by the controller.

###### Are there any tests for feature enablement/disablement?

Partially. The controller is registered behind
`requiredFeatureGates: []featuregate.Feature{features.DRAResourcePoolStatus}`,
its bootstrap ClusterRole is installed only when the gate is on, and the
gated fields are covered by strategy and declarative-validation unit tests
for both gate states. The integration suite, however, runs a single
`feature-enabled` matrix entry — there is no gate-disabled integration case
today. Adding one is a Beta requirement; see [Beta](#beta).

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

**Rollout failures:**
- Feature gate not enabled on both apiserver and KCM
- RBAC not configured for users

**Impact on workloads:**
- None. This is a read-only visibility feature.

###### What specific metrics should inform a rollback?

- High error rate on request processing
- Controller crash loops
- Excessive API server load from requests

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be tested manually before Beta promotion and documented here. For Alpha,
the feature is behind a feature gate and has no persistent state that could
cause issues during upgrade/downgrade cycles.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- Check if ResourcePoolStatusRequest objects exist: `kubectl get resourcepoolstatusrequests`
- Check controller metrics: `resourcepoolstatusrequest_controller_requests_processed_total > 0`

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: N/A (no events emitted)
- [x] API .status
  - The presence of a non-nil `status` indicates the controller has
    processed the request.
  - Condition type `Complete` with status `"True"` signals a successful
    calculation; `Failed` with `"True"` signals a processing error (the
    condition `message` carries details).
  - The `Complete`/`Failed` condition's `lastTransitionTime` indicates
    when the calculation occurred (this replaces the originally proposed
    `status.observationTime` field, which was dropped during API review).
- [ ] Other (Alarm, К8s resources status)

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- Request processing: 99% of requests complete within 30 seconds
- No impact on existing scheduling or pod startup SLOs

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

All metrics use the subsystem `resourcepoolstatusrequest_controller` and are
labeled by `driver_name`. Stability level: ALPHA.

- [x] Metrics
  - Metric name: `resourcepoolstatusrequest_controller_request_processing_duration_seconds`
    - Aggregation method: histogram (exponential buckets starting at 1ms, 15 buckets × base 2)
    - Labels: `driver_name`
    - Components exposing the metric: kube-controller-manager
  - Metric name: `resourcepoolstatusrequest_controller_request_processing_errors_total`
    - Aggregation method: counter
    - Labels: `driver_name`
    - Components exposing the metric: kube-controller-manager
  - Metric name: `resourcepoolstatusrequest_controller_requests_processed_total`
    - Aggregation method: counter
    - Labels: `driver_name`
    - Components exposing the metric: kube-controller-manager
- [ ] Other (describe)

Coverage caveat: all three metrics are recorded only on the code paths that
reach `UpdateStatus`. `..._request_processing_errors_total` therefore counts
`UpdateStatus` failures only — a ResourceSlice / ResourceClaim /
DeviceTaintRule lister failure produces a `Failed` condition but no error
sample — and the incomplete-pool requeue path records nothing at all.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Yes. Two gaps follow from the coverage caveat above and are tracked as Beta
items: requeues and give-ups on incomplete pools are invisible, and
calculation failures that produce a `Failed` condition are not counted as
errors.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

| Dependency | Usage | Impact of Unavailable | Impact of Degraded | Can Operate Without |
|------------|-------|----------------------|-------------------|---------------------|
| kube-controller-manager | Runs the ResourcePoolStatusRequest controller | Requests will not be processed (status stays empty) | Slower processing | No (required for status computation) |
| DRA drivers | Create ResourceSlices that are aggregated | No pools to report (empty results) | Incomplete pool data | Yes (returns empty/partial results) |

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes:

| API Call Type | Estimated Throughput | Originating Component |
|---------------|---------------------|----------------------|
| CREATE ResourcePoolStatusRequest | User-driven, typically < 1/min per user | kubectl / client applications |
| GET ResourcePoolStatusRequest | User-driven, typically < 10/min per user | kubectl / client applications |
| DELETE ResourcePoolStatusRequest | User-driven, typically < 1/min per user | kubectl / client applications |
| UPDATE ResourcePoolStatusRequest/status | 1 per request created | kube-controller-manager |
| LIST/WATCH ResourceSlices | Reuses existing informer (no new calls) | kube-controller-manager |
| LIST/WATCH ResourceClaims | Reuses existing informer (no new calls) | kube-controller-manager |

###### Will enabling / using this feature result in introducing new API types?

Yes:

| API Type | Supported Operations | Estimated Max Objects |
|----------|---------------------|----------------------|
| ResourcePoolStatusRequest | CREATE, GET, LIST, DELETE, WATCH | Hundreds per cluster (user-managed, ephemeral) |

Note: Objects are intended to be short-lived. Built-in TTL cleanup (Alpha)
deletes completed requests 1 hour after completion and pending requests
24 hours after creation.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of existing API objects?

One existing type changes: `ResourceSlice.Spec` gains an optional
`PartitionTypeAttribute *FullyQualifiedName` (gated behind
`DRAPartitionableDevicesType`), present in `resource.k8s.io/v1`,
`v1beta1` and `v1beta2`. It is a single, bounded string per slice
and is omitted unless the driver opts in, so per-slice size is
effectively unchanged on existing clusters.

Alpha 1.37 also adds optional `partitionSummary` (`+k8s:maxItems=32`,
provisional) and `shareableSummary` (a fixed-shape sub-object with an
inner `capacity[]` capped at `+k8s:maxItems=32`) to each `PoolStatus`.
Both are omitted on plain pools, so the typical response size is
unchanged; on partitionable or consumable pools the response grows by a
bounded, small amount — both are aggregates and are much smaller than
the per-device lists they summarise.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No impact on scheduling or pod startup.

###### Will enabling / using this feature result in non-negligible increase of resource usage?

Minimal:
- etcd: Small objects, bounded by built-in TTL cleanup (Alpha: 1h completed / 24h pending)
- KCM: Reuses existing `resource.k8s.io/v1` informers for ResourceSlice and ResourceClaim, adds a small controller with its own work queue
- API server: Standard API operations
- Response size: Bounded by the required `driver` field (one driver's pools), the `limit` field (default 100, max 1000), the `+k8s:maxItems=1000` constraint on `status.pools`, and `+k8s:maxItems=32` on each of `partitionSummary` and `shareableSummary.capacity` per pool

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. This feature runs entirely in kube-controller-manager and kube-apiserver:
- No node-level resources are consumed
- No new processes or sockets created on nodes
- No file system operations on nodes
- Controller uses existing informers (no additional watch connections)

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Requests cannot be created or read. No workload impact.

###### What are other known failure modes?

| Failure Mode | Description | Detection | Mitigations | Diagnostics | Testing |
|--------------|-------------|-----------|-------------|-------------|---------|
| Controller not running | ResourcePoolStatusRequest controller in KCM is not running or crashed | Requests stay with `status` unset (no `Complete`/`Failed` condition); `resourcepoolstatusrequest_controller_requests_processed_total` stays at 0 | Restart KCM, check KCM logs | Check KCM logs for controller startup errors, verify feature gate enabled | Covered by integration tests |
| Informers not synced | ResourceSlice or ResourceClaim informers have not completed initial sync | Controller logs warning, requests delayed | Wait for informer sync, check API server connectivity | Check KCM logs for informer sync status | Covered by integration tests |
| Incomplete pool data | Fewer slices published than `ResourceSliceCount` declared by the driver | Controller requeues up to 5 times, giving the driver time to finish publishing; once the pool is whole the status is written normally, with no incompleteness marker | Ensure driver fully publishes slices; retry by recreating request | While requeueing, the request stays `Pending`; check driver logs and `kubectl get resourceslices` for the pool | Covered by unit tests |
| Pool never completes | A driver stops publishing partway through a generation, so a pool stays below its declared `ResourceSliceCount` | The request keeps `status` unset indefinitely — no `Complete` or `Failed` condition, no metric sample — because the sync returns an error before writing status and gives up after 5 retries | Fix the driver so it publishes the full slice set; the request is removed by the 24h pending TTL | `kubectl get` shows `Pending` with no `COMPLETED` timestamp; check driver logs and `kubectl get resourceslices` for the pool | Requeue path covered by unit tests; giving this case a terminal state is a Beta item |
| Request accumulation | Users create many requests | etcd storage grows, `kubectl get resourcepoolstatusrequests` shows many objects | Built-in TTL cleanup deletes completed requests after 1h, pending after 24h | List requests, check etcd metrics; check KCM cleanup logs | Covered by integration tests |

###### What steps should be taken if SLOs are not being met?

1. Check KCM logs for controller errors
2. Check controller metrics
3. Verify informers are synced
4. Check for excessive request volume

## Implementation History

- 2025-12-20: KEP created in provisional state
- 2026-01-15: Design revision - ResourceSlice status to ResourcePool
- 2026-02-07: Design revision - in-tree CSR-like pattern per API review
- 2026-02-10: KEP merged as implementable (#5749)
- 2026-02/03: Alpha implementation in kubernetes/kubernetes — API shipped
  in `resource.k8s.io/v1alpha3` (not `v1alpha1`) with several API-review
  driven changes: `status` is now a pointer and the whole object is
  immutable once populated; `observationTime` removed (use the
  `Complete`/`Failed` condition's `lastTransitionTime`); top-level
  `validationErrors` and `truncated` removed (per-pool `validationError`
  and `len(pools) < poolCount` used instead); `sliceCount` renamed to
  `resourceSliceCount`; count fields made pointers so they can be left
  unset for incomplete pools; added `Failed` condition type; explicit
  `limit` bounds (default 100, max 1000); and TTL-based cleanup moved
  into Alpha.
- 1.36 (Alpha): feature gate `DRAResourcePoolStatus` (default off);
  API shipped at `resource.k8s.io/v1alpha3` (kubernetes/kubernetes#137028)
- 1.37 (Alpha): second Alpha cycle on `v1alpha3` to correctly handle
  partitionable and consumable devices
  (kubernetes/kubernetes#140170). Added `partitionSummary` and
  `shareableSummary` to `PoolStatus`,
  `spec.defaultPartitionTypeAttribute` to the request,
  `ResourceSlice.Spec.PartitionTypeAttribute` (in `v1`, `v1beta1` and
  `v1beta2`), and a second feature gate `DRAPartitionableDevicesType`.
  Capped `allocatedDevices` at one per physical device, skipped
  AdminAccess allocations, and computed `unavailableDevices` from real
  device taints. Two API-review changes relative to the plan: the
  `counterSets` fallback view was dropped in favour of the request-side
  `defaultPartitionTypeAttribute`, and `partitionSummary` entries are
  keyed by `(attribute, type)` rather than assuming one grouping
  attribute per pool. See "Alpha (1.37)" in Graduation Criteria.

## Drawbacks

1. **Asynchronous operation**: User must wait for controller, unlike sync APIs
   - Mitigation: Processing is fast (seconds); `kubectl wait --for=condition=Complete` helps

2. **Objects persist briefly in etcd**: Each request is a cluster-scoped object
   - Mitigation: Controller-side TTL cleanup (Alpha) — 1h after completion, 24h for pending

3. **Not real-time**: Shows point-in-time snapshot, not live data
   - Mitigation: `Complete` condition `lastTransitionTime` shows age; delete and recreate for fresh data

## Alternatives

### Alternative 1: Out-of-tree Aggregated API Server

Deploy a separate aggregated API server (like metrics-server) that computes
pool status on-demand.

**Pros:**
- On-demand computation (no persistence)
- Independent release cycle
- No etcd storage

**Cons:**
- Additional deployment to manage
- Not always available by default
- Duplicate informers add API server load

**Rejected because:** API review preferred in-tree solution that is always
available and in-sync with Kubernetes releases.

### Alternative 2: Synchronous Review Pattern

Use SubjectAccessReview-like pattern where status is computed synchronously
in the API server during the Create call.

**Pros:**
- Immediate response
- No persistence needed
- Simpler user flow

**Cons:**
- Cannot reuse KCM informers (would need informers in API server)
- Computation in API server request path
- No established pattern for this in resource.k8s.io

**Rejected because:** Would require new informers in API server; CSR pattern
is more established for operations that need controller processing.

### Alternative 3: Status in ResourceSlice

Add a Status field to ResourceSlice to track per-device allocations.

**Pros:**
- No new API type

**Cons:**
- Increases ResourceSlice size significantly
- RBAC issues: claim info exposed to slice readers
- Cross-pool aggregation awkward

**Rejected because:** Size, churn, and RBAC concerns from API review.

### Alternative 4: Client-side only

Only provide kubectl plugin that computes everything locally.

**Pros:**
- No server-side changes
- Zero cluster overhead

**Cons:**
- Each invocation fetches all slices and claims
- Poor performance for large clusters
- No API for automation tools

**Rejected because:** Poor performance at scale; no API for automation.

## Infrastructure Needed

None - this is an in-tree feature.
