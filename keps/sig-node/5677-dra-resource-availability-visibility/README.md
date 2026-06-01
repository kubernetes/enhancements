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
  - [Controller Implementation](#controller-implementation)
    - [Controller in KCM](#controller-in-kcm)
    - [One-time Processing](#one-time-processing)
    - [Incomplete-Pool Handling and Requeue](#incomplete-pool-handling-and-requeue)
    - [Reusing Existing Informers](#reusing-existing-informers)
    - [Partitionable &amp; Consumable Device Accounting](#partitionable--consumable-device-accounting)
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
   is set the entire object (metadata included) becomes immutable. To get
   updated data, delete and recreate the request.

3. **Automatic TTL cleanup**: Completed or failed requests are deleted by the
   controller 1 hour after their `Complete`/`Failed` condition is set.
   Pending requests (no status) are deleted 24 hours after creation to
   handle stuck requests. Users can still delete requests manually at any
   time.

4. **Controller processing delay**: Status is not immediate - controller must
   process the request. Typically completes within seconds.

5. **RBAC controls access**: Users need RBAC permission to create/read
   ResourcePoolStatusRequest objects to use this feature.

6. **Partitionable & consumable devices** (Alpha 1.36 limitation —
   addressed in Alpha 1.37): in 1.36 the controller counts each
   entry in `ResourceSlice.Spec.Devices` once per allocation result,
   which is misleading for two device shapes:

   - **Partitionable** (`DRAPartitionableDevices` feature gate): a
     single physical device may appear as multiple mutually-exclusive
     partitions that share a `CounterSet`. Counting devices ignores
     the shared bottleneck.
   - **Consumable** (`DRAConsumableCapacity` feature gate): a device
     with `allowMultipleAllocations=true` may serve many claims
     simultaneously. Counting each claim against `allocatedDevices`
     drives `availableDevices` to 0 on pools that still have free
     capacity (the `max(0, …)` floor in the controller hides the
     overcount as "0 available" rather than as a negative number).

   Alpha 1.37 adds an optional `partitionSummary[]` sub-object
   (a typed "devices-by-partition-type" view that nets out shared
   counter consumption) and a `shareableSummary` aggregate to each
   `PoolStatus`, caps the per-device contribution to
   `allocatedDevices` at 1, and skips AdminAccess allocations in all
   accounting. `partitionSummary[]` is emitted when the driver
   declares a per-pool grouping attribute on the slice
   (`ResourceSlice.Spec.PartitionTypeAttribute`, new in 1.37); pools
   without that declaration fall back to a raw `counterSets[]` dump.
   See
   [Partitionable & Consumable Device Accounting](#partitionable--consumable-device-accounting)
   under Controller Implementation.

7. **Incomplete pools**: When a pool's observed ResourceSlice count is less
   than `ResourceSliceCount` declared by the driver, the pool is reported
   with `validationError` set and device-count fields left unset. The
   controller requeues the request (up to 5 attempts) to give drivers time
   to publish remaining slices.

8. **Generation handling**: ResourceSlices with older pool generations are
   ignored during computation (not counted as errors). Drivers are expected
   to delete old-generation slices eventually. The `generation` field in
   each PoolStatus reflects the highest generation observed.

9. **`unavailableDevices`**: in Alpha 1.36 always `0`. Alpha 1.37
   computes this from real device taints (`NoSchedule` and
   `NoExecute` effects) on each device.

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
  # backed by a single 80Gi CounterSet. The driver declared a grouping
  # attribute on the slice, so the controller emits the typed view. Here one
  # half partition is in use, so 40Gi of the counter is consumed (debited
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
    partitionSummary:             # emitted when the slice declares
                                  # spec.partitionTypeAttribute
    - type: full                  # value of the declared grouping attribute
      total: 1                    # device entries of this partition type
      allocatable: 0              # 40Gi left, full needs 80Gi → blocked
    - type: half
      total: 2
      allocatable: 1              # 1 fresh half remains, 40Gi available, fits
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
      partiallyAvailableDevices: 2
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

Once `status` is populated the entire object (including `metadata` and `spec`)
is immutable; update requests are rejected by the API server. Users must
delete and recreate to re-run a query.

#### Spec Fields

The spec is **immutable after creation** (enforced via `+k8s:immutable`), and
the entire object becomes immutable once `status` is set. Updates to the spec
are rejected by API validation.

| Field | Type | Description |
|-------|------|-------------|
| `driver` | string (required) | DRA driver name — bounds response to one driver's pools. Must be a DNS subdomain. |
| `poolName` | `*string` (optional) | Filter by pool name. Must be a valid resource pool name (DNS subdomains separated by `/`). |
| `limit` | `*int32` (optional) | Max pools to return. Default **100**, min **1**, max **1000**. |

#### Status Fields

Status is a pointer (`*ResourcePoolStatusRequestStatus`). Presence of a
non-nil status indicates the request has been processed.

| Field | Type | Description |
|-------|------|-------------|
| `poolCount` | `*int32` (required) | Total pools matching filter (regardless of truncation). |
| `pools[]` | atomic list, max 1000 | First `spec.limit` matching pools, sorted by driver then pool name. Truncation is inferred from `len(pools) < poolCount`. |
| `pools[].driver` | string (required) | DRA driver name. |
| `pools[].poolName` | string (required) | Pool name from ResourceSlice. |
| `pools[].generation` | int64 (required) | Latest pool generation observed. |
| `pools[].nodeName` | `*string` (optional) | Node name for node-local pools. Omitted when the pool spans multiple nodes or has mixed/no node assignment. |
| `pools[].resourceSliceCount` | `*int32` (optional) | Number of slices observed at the latest generation. Unset when `validationError` is set. |
| `pools[].totalDevices` | `*int32` (optional) | Total devices across all slices. Unset when `validationError` is set. |
| `pools[].allocatedDevices` | `*int32` (optional) | Devices allocated to claims. Unset when `validationError` is set. |
| `pools[].availableDevices` | `*int32` (optional) | `totalDevices - allocatedDevices - unavailableDevices`. Unset when `validationError` is set. |
| `pools[].unavailableDevices` | `*int32` (optional) | Count of physical devices with at least one `NoSchedule` or `NoExecute` taint. **0 in Alpha 1.36** (hard-coded); **computed from `ResourceSlice.Spec.Devices[].Taints` and matching `DeviceTaintRule`s in Alpha 1.37**. Unset when `validationError` is set. |
| `pools[].validationError` | `*string` (optional, max 256 bytes) | Set when the pool's data could not be fully validated (e.g., incomplete slice publication). When set, count fields above may be unset. |
| `pools[].partitionSummary[]` | atomic list of `PartitionTypeStatus`, max 32 (Alpha 1.37, **provisional** — revisit at Beta) | Per-partition-type aggregate, emitted when the pool's slices declare `ResourceSlice.Spec.PartitionTypeAttribute` (new in 1.37). Each entry reports `type` (the value of the declared attribute on devices in this group), `total` (count of devices in the pool with this value), and `allocatable` (how many more devices of this type can still be allocated given current shared-counter state). Devices in the pool whose `ConsumesCounters` cost differs from peers of the same type produce a per-pool `validationError` rather than silent skew. Cap of 32 is a provisional starting point that fits MIG-class pools (3–7 partition types typical); over-cap pools produce a per-pool `validationError` instead of silent truncation. **Mutually exclusive with `counterSets[]`** — a pool emits one view or the other depending on whether the driver declared a grouping attribute. |
| `pools[].partitionSummary[].type` | string (required) | Value of the device attribute named by `ResourceSlice.Spec.PartitionTypeAttribute` for devices in this group. |
| `pools[].partitionSummary[].total` | int32 (required) | Number of devices in the pool whose declared attribute carries this value. |
| `pools[].partitionSummary[].allocatable` | int32 (required) | Number of additional devices of this partition type that can still be allocated under current shared-counter constraints, capped by the number of unallocated devices of this type in the pool. Computed by a greedy per-device fit check against `counterAvailable[s][c] = SharedCounters[s].Counters[c].Value − sum_{in-use d in s} d.ConsumesCounters[s][c]` (each in-use device debited once, per-device not per-claim — matches scheduler counter accounting). For the common single-counter-set case this reduces to `min(freshDevices[type], min over counters c of floor(counterAvailable[s_type][c] / consumesCounters[type][c]))`, where `freshDevices[type]` is the count of devices of this type currently unallocated. See [Partitionable & Consumable Device Accounting](#partitionable--consumable-device-accounting) for the multi-counter-set algorithm. On shareable partitions (`allowMultipleAllocations=true`) this counts only fresh device slots, not capacity headroom on already-in-use devices; operators reading the same pool should consult `shareableSummary.capacity[].available` for per-key headroom on shared devices. |
| `pools[].counterSets[]` | atomic list of `CounterSetStatus`, max 32 (Alpha 1.37, **provisional** — revisit at Beta) | **Fallback view**, emitted only when the pool has `sharedCounters` but the slices do not declare `ResourceSlice.Spec.PartitionTypeAttribute`. Per-`CounterSet` capacity / consumed / available, derived from `ResourceSlice.Spec.SharedCounters` and the `consumesCounters` of each **unique in-use** non-AdminAccess device (counter cost is debited per device, not per claim, matching scheduler behaviour for shareable partitions). Omitted on pools with no shared counters and on pools where the driver opted into the typed `partitionSummary[]` view (drivers are encouraged to declare a grouping attribute so this verbose fallback is not used). The spec-side per-slice cap is `ResourceSliceMaxCounterSets = 8`; pools can contain many slices with no per-pool cap upstream, so the status cap of 32 is a deliberate starting point rather than a mirror of any spec constant. Over-cap pools produce a per-pool `validationError` instead of silent truncation. **Note:** `CounterSetStatus` is a new type, not a reuse of the spec-side `CounterSet`. The spec's `Counter` only carries `Value` (inventory); a status-side type is needed to add the `consumed` and `available` fields without overloading the spec type. |
| `pools[].counterSets[].name` | string (required) | Counter-set name as declared in `ResourceSlice.Spec.SharedCounters[].Name`. |
| `pools[].counterSets[].counters` | `map[string]CounterStatus` (required) | Per-counter status. `CounterStatus` is a new type with three required `resource.Quantity` fields: `capacity` (mirrors the spec-side `Counter.Value`), `consumed` (sum of consumption from **unique in-use** non-AdminAccess devices), and `available` (`capacity − consumed`, never negative). |
| `pools[].shareableSummary` | `*ShareableSummaryStatus` (optional) | Pool-level aggregate for devices with `allowMultipleAllocations=true`. Omitted when the pool has no such devices. Per-device detail was intentionally not included: a per-device list would scale to hundreds of entries on large pools, so the aggregate gives the operator-relevant signal in three small numbers plus a per-capacity-key breakdown. |
| `pools[].shareableSummary.fullyAvailableDevices` | int32 (required) | Count of shareable devices in the pool with **zero** non-AdminAccess claims. |
| `pools[].shareableSummary.partiallyAvailableDevices` | int32 (required) | Count of shareable devices with **at least one** non-AdminAccess claim. `fullyAvailableDevices + partiallyAvailableDevices` equals the total number of shareable devices in the pool. |
| `pools[].shareableSummary.capacity[]` | atomic list of `ShareableCapacityStatus`, max 32 (Alpha 1.37) | Per-capacity-key aggregate across all shareable devices in the pool. Cap of 32 matches the per-device combined `Attributes + Capacity` cap (no single device can carry more than 32 capacity keys); aggregation across devices may introduce additional keys but homogeneous-schema pools rarely exceed this. |
| `pools[].shareableSummary.capacity[].name` | string (`QualifiedName`, required) | Capacity key as it appears in `ResourceSlice.Spec.Devices[].Capacity`. |
| `pools[].shareableSummary.capacity[].total` | `resource.Quantity` (required) | Sum of `Device.Capacity[name].Value` across all shareable devices in the pool that carry this key. Devices that do not carry the key contribute nothing (rather than zero), which is the correct behaviour for heterogeneous-schema pools. |
| `pools[].shareableSummary.capacity[].consumed` | `resource.Quantity` (required) | Sum of `DeviceRequestAllocationResult.ConsumedCapacity[name]` across non-AdminAccess allocations on shareable devices that carry this key. |
| `pools[].shareableSummary.capacity[].available` | `resource.Quantity` (required) | `total − consumed`, clamped at zero (never negative). |
| `conditions[]` | map list by `type`, max 10 | `Complete` (True when processed) or `Failed` (True on error). |

#### Companion API Change: `ResourceSlice.Spec.PartitionTypeAttribute`

Alpha 1.37 adds one optional field to `ResourceSliceSpec`
(`staging/src/k8s.io/api/resource/v1/types.go`):

```go
// PartitionTypeAttribute names a device attribute whose value
// groups devices in this pool by partition type. When declared on
// every slice in a pool (all with the same value), the resource
// pool status controller emits a typed partitionSummary[] view
// instead of the verbose counterSets[] dump. Leaving the field
// unset on every slice in a pool keeps the counterSets[] fallback;
// a mix of set and unset across slices in the same pool, or
// different values, produces a per-pool validationError.
//
// Only meaningful when SharedCounters is set somewhere in the
// pool. Declarative validation rejects this field only when the
// slice carries neither Devices nor any reference to a counter
// set; all pool-level consistency checks are controller-side.
//
// The value must be a domain-qualified attribute key in the same
// shape used by DeviceSelector.MatchAttribute /
// DistinctAttribute.
//
// +optional
// +featureGate=DRAResourcePoolStatus
PartitionTypeAttribute *FullyQualifiedName `json:"partitionTypeAttribute,omitempty" protobuf:"bytes,9,opt,name=partitionTypeAttribute"`
```

Feature-gate interaction: the field is gated behind
`DRAResourcePoolStatus`, but it is only effective when
`DRAPartitionableDevices` is also enabled (since the field is only
meaningful for pools with `SharedCounters`, which is itself gated by
`DRAPartitionableDevices`). When only one gate is enabled, the field
is silently dropped at write time (gate off) or accepted but ignored
by the status controller (`DRAResourcePoolStatus` off).

Validation rules (declarative on the slice, plus controller-side
per-pool checks; the slice-level rule is intentionally permissive
because a counter-consuming slice need not declare its own
`SharedCounters`):

- **Declarative slice-level check (loose):** the field may be set on
  any `ResourceSlice` and is rejected only when the slice carries
  neither `Devices` nor any reference to a counter set — i.e. the
  slice has no way to participate in counter-typed accounting. All
  cross-slice / pool-level enforcement is controller-side.
- **Pool-level: SharedCounters must exist.** The status controller
  emits a per-pool `validationError` when `PartitionTypeAttribute`
  is declared on any slice in a pool that has no `SharedCounters`
  anywhere.
- **Pool-level: cross-slice consistency.** All slices in a pool that
  has shared counters must declare the same `PartitionTypeAttribute`
  value (or all leave it unset). A mismatch — different values, or
  a mix of set and unset — is a per-pool `validationError`.
- **Pool-level: every device carries the attribute.** For the
  controller to compute `partitionSummary[].type`, every device in a
  pool that opts into the typed view must carry the attribute named
  by `PartitionTypeAttribute`. Devices missing the attribute produce
  a per-pool `validationError` rather than being silently bucketed.
- **Pool-level: homogeneous `ConsumesCounters` per partition type.**
  Devices in the same partition-type group must share the same
  `ConsumesCounters` cost; mixed costs in one group produce a
  per-pool `validationError` since the `allocatable[T]` math
  assumes a single canonical cost per type.

Cross-KEP relationship: the field lives on the `ResourceSlice` shape
owned by [KEP-4815](/keps/sig-scheduling/4815-dra-partitionable-devices),
but it is consumed only by this KEP's controller and is gated behind
`DRAResourcePoolStatus`. The slice schema change must be reviewed
together with KEP-4815 so the field shape stays coherent with the
rest of `ResourceSliceSpec`.

### Controller Implementation

#### Controller in KCM

The controller is added to kube-controller-manager as a separate controller
named `resourcepoolstatusrequest-controller` with its own client (so
client-side throttling does not impact scheduling). It is registered in
`cmd/kube-controller-manager/app/resource.go`.

The controller:
1. Watches ResourcePoolStatusRequest (`resource.k8s.io/v1alpha3`) objects
   via informer.
2. Maintains a rate-limited work queue for processing, with up to 5 retries
   per request before dropping.
3. Reuses existing ResourceSlice and ResourceClaim informers (from the
   stable `resource.k8s.io/v1` group) already running in KCM.
4. Uses `UpdateStatus` to write results to the status subresource.

#### One-time Processing

Following the CSR pattern, the controller processes each request exactly once:

1. When a new ResourcePoolStatusRequest is created, it is added to the work queue.
2. Controller checks if `status` is already non-nil.
3. If non-nil, the request was already processed — controller skips it.
4. If nil, controller computes pool status and writes to `status`.
5. Once `status` is written, the entire object is immutable (spec, metadata,
   and status all rejected for update by the registry strategy / validation).

To get fresh data, users delete and recreate the request. (See the TTL
cleanup section below for automatic deletion of old requests.)

#### Incomplete-Pool Handling and Requeue

When the number of ResourceSlices observed for a pool (at the latest
generation) is less than the pool's declared `ResourceSliceCount`, the pool
is considered incomplete:

- The controller sets `pools[i].validationError` with a message (truncated
  to 256 bytes) and leaves `resourceSliceCount`, `totalDevices`,
  `allocatedDevices`, `availableDevices`, and `unavailableDevices` unset.
- The request is requeued (up to `maxRetries = 5`) so drivers have time to
  publish remaining slices before the status is finalized.
- If retries are exhausted, the latest calculated status (with the
  `validationError` markers) is still written so users see the issue.

#### Reusing Existing Informers

The controller reuses ResourceSlice and ResourceClaim informers from the
`resource.k8s.io/v1` informer factory already running in KCM for other DRA
controllers (e.g. device-taint-eviction). This adds minimal overhead since
the informers are already cached in memory. The controller constructor
accepts these shared informers rather than creating its own, following the
established KCM pattern.

#### Partitionable & Consumable Device Accounting

In Alpha 1.36 the controller computes `allocatedDevices` by walking
each `ResourceClaim.Status.Allocation.Devices.Results` and incrementing
a per-device counter — see `pkg/controller/resourcepoolstatusrequest/controller.go`
(the `calculatePoolStatus` step that builds `allocationData`). That
arithmetic is correct for plain devices but wrong for two API shapes
the broader DRA stack supports:

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
Beta in 1.36, GA target 1.37). Both are at Beta default-on by the
time Alpha 1.37 of this KEP ships, so the fields we read
(`SharedCounters`, `ConsumesCounters`, `AllowMultipleAllocations`,
`ConsumedCapacity`) are part of the served `resource.k8s.io/v1`
surface. When either gate is disabled on a cluster, the corresponding
sub-object is omitted from the response — the source fields are nil
on incoming `ResourceSlice` objects, the aggregation produces no
entries (the slice stays nil rather than being initialised to an
empty `[]`), and `omitempty` keeps the common-case payload shape
unchanged.

Alpha 1.37 changes the aggregation to handle all three shapes
consistently:

1. **Per-device cap on `allocatedDevices`.** A physical device is
   counted at most once regardless of how many non-AdminAccess
   claims reference it. This fixes the consumable overcount in
   Alpha 1.36 (where N claims on one device added N to the
   tally).
2. **AdminAccess allocations are skipped** in every device, counter,
   and shareable-device tally. They are observers, not consumers,
   and counting them misleads administrators about real availability.
3. **`unavailableDevices`** is the count of physical devices with at
   least one `NoSchedule` or `NoExecute` taint (sourced from
   `ResourceSlice.Spec.Devices[].Taints` and any `DeviceTaintRule`
   matches), replacing the Alpha 1.36 hard-coded `0`. When
   `DRADeviceTaintRules` is disabled (default-off as of 1.36), only
   embedded `Spec.Devices[].Taints` contribute and external
   `DeviceTaintRule` matching is skipped silently. Embedded taints
   alone are sufficient on most clusters because `DRADeviceTaints` is
   Beta default-on.
4. **`partitionSummary[]`** is emitted when the pool's slices declare
   `ResourceSlice.Spec.PartitionTypeAttribute` (and the pool has any
   `sharedCounters`). The controller groups devices by the value of
   that attribute and, per group T, computes:
   - `total[T]` = count of devices in the pool whose attribute equals T.
   - `freshDevices[T]` = count of devices in group T that are
     currently unallocated (no non-AdminAccess claim references them).
   - `cost[T]` = the canonical per-device `ConsumesCounters` profile
     for partition type T, read from any device in the group
     (validated as homogeneous across the group; mixed costs produce
     a per-pool `validationError`).
   - `counterAvailable[s][c]` = `SharedCounters[s].Counters[c].Value`
     minus the sum of `d.ConsumesCounters[s][c]` over **unique in-use**
     non-AdminAccess devices `d` in the pool that consume from counter
     set `s`. Each in-use device is counted once regardless of how
     many claims reference it; this matches the scheduler's counter
     accounting for shareable partitions
     (`staging/src/k8s.io/dynamic-resource-allocation/structured/internal/incubating/allocator_incubating.go`,
     where the counter check is skipped on subsequent allocations of
     an `allowMultipleAllocations=true` device).
   - `allocatable[T]` is computed by a greedy per-device fit check:
     iterate fresh devices `d` in group T, and for each one check
     that for every counter set `s` and counter `c` in `d.ConsumesCounters`,
     `counterAvailable[s][c] >= cost[T][s][c]`. If the check passes,
     increment `allocatable[T]` and deduct `cost[T]` from
     `counterAvailable[s]` so subsequent siblings drawing from the
     same counter set are accounted correctly. For the common case
     where every device in group T consumes from a single counter
     set `s_T` with the same cost, this reduces to
     `allocatable[T] = min(freshDevices[T], min over c of floor(counterAvailable[s_T][c] / cost[T][c]))`.
     The fresh-device cap matters when counter headroom exceeds the
     supply of unallocated devices (otherwise the scalar would
     advertise impossible allocations).

   On shareable partitions (`allowMultipleAllocations=true`),
   `allocatable[T]` counts only fresh device slots; capacity headroom
   remaining on already-in-use devices is published separately under
   `shareableSummary.capacity[]`. Operators on hybrid pools should
   read both fields.
5. **`counterSets[]` (fallback)** is emitted when the pool has any
   `sharedCounters` and the slices do **not** declare
   `PartitionTypeAttribute`. The controller initialises each entry
   from the pool's `ResourceSlice.Spec.SharedCounters`, then for each
   **unique in-use** non-AdminAccess device in the pool walks the
   device's `consumesCounters` and adds to `consumed` exactly once
   (per-device, not per-claim — same scheduler-side dedup rule as
   the typed view). `available` is `capacity − consumed`. Drivers
   are encouraged to declare the grouping attribute so the typed
   view replaces this verbose dump; the fallback is kept so drivers
   that have not yet adopted the convention still get an
   operator-readable signal.
6. **`shareableSummary`** is emitted when the pool has any device
   with `allowMultipleAllocations=true`. The controller scans all
   shareable devices in the pool and produces three fields:
   `fullyAvailableDevices` (devices with zero non-AdminAccess claims),
   `partiallyAvailableDevices` (devices with at least one non-AdminAccess
   claim), and `capacity[]` — a per-capacity-key aggregate where
   each entry sums `Device.Capacity[name].Value` over devices
   carrying the key (`total`) and
   `DeviceRequestAllocationResult.ConsumedCapacity[name]` over
   non-AdminAccess allocations on those devices (`consumed`), with
   `available = total − consumed` clamped at zero. A per-device
   array would scale to hundreds of entries on large pools; the
   aggregate gives the operator-relevant signal far more compactly.
   Heterogeneous-schema pools are handled by the rule "devices that
   do not carry a given key contribute nothing to that key's total"
   — the aggregate stays correct rather than reporting zeros that
   would misrepresent capacity.

`availableDevices` keeps its existing definition
(`totalDevices − allocatedDevices − unavailableDevices`). On plain
pools it is the operationally useful "how many more claims fit"
signal. **On pools with shared counters or shareable devices it is
not.** Two cases the operator must understand:

- **Partitionable pools.** When the bottleneck is a shared
  `CounterSet`, all device entries can be unallocated yet no further
  claim can fit — `availableDevices` will read high while no
  partition actually fits. Operators must consult
  `partitionSummary[T].allocatable` (when the driver declared
  `PartitionTypeAttribute`, this is the canonical signal: how many
  more devices of partition type `T` still fit under current counter
  state) or, on pools that fall back to the raw view,
  `counterSets[].counters[].available`.
- **Consumable / shareable pools.** With the cap-at-1 rule, every
  device with at least one claim is counted once in
  `allocatedDevices`. A pool of N shareable devices each holding
  one tiny claim will report `allocatedDevices=N` and
  `availableDevices=0`, even though most of each device's capacity
  is free. Operators must consult
  `shareableSummary.capacity[].available` vs `.total` for the
  remaining-capacity signal, and the
  `fullyAvailableDevices`/`partiallyAvailableDevices` counts for the
  share-pattern signal.

This is documented as a deliberate trade-off: `availableDevices`
remains a stable scalar that older clients can use, and the new
sub-objects carry the precise truth. The KEP does not redefine
`availableDevices` per pool shape because doing so would silently
change its meaning for existing 1.36 consumers.

All three sub-objects are omitted when empty so plain pools stay
compact. `partitionSummary[]` and the fallback `counterSets[]` both
carry `+k8s:maxItems=32` (provisional — revisit at Beta; pools larger
than this produce a per-pool `validationError` rather than silent
truncation). `shareableSummary.capacity[]` carries `+k8s:maxItems=32`
to match the per-device combined `Attributes + Capacity` cap.
`partitionSummary[]` and `counterSets[]` are mutually exclusive per
pool — the controller emits one or the other, never both.

##### Devices That Are Both Partitionable and Consumable

When a single physical device is both partitionable and consumable —
its partitions draw from a shared `CounterSet` *and* individual
partitions allow multiple allocations — consuming capacity on one
partition can make sibling partitions unallocatable through the shared
counter, even though those siblings still appear as unconsumed
devices.

The typed `partitionSummary[]` view addresses this directly: because
`allocatable[T] = min(freshDevices[T], min over c of floor(counterAvailable[c] / cost[T][c]))`
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

- **Pools using the `counterSets[]` fallback (no
  `PartitionTypeAttribute` declared).** `availableDevices` is not
  netted out against shared counters and `shareableSummary` reports
  raw device-capacity aggregates that do not subtract counter-blocked
  siblings; operators must read
  `counterSets[].counters[].available` for the authoritative bound.
  This is the same caveat as the original sunya-ch example.
- **Capacity headroom on shareable in-use partitions
  (`allowMultipleAllocations=true`).** The scheduler debits
  `ConsumesCounters` exactly once per device — subsequent claims
  against the same shareable device do not consume more counter
  capacity — and `partitionSummary[T].allocatable` follows the same
  rule (it counts only how many additional *fresh* devices of type T
  can be allocated). What it does **not** capture is how much
  per-claim capacity is still available on devices that are already
  in use. For that, operators on hybrid pools must read
  `shareableSummary.capacity[]`, which reports the remaining free
  capacity per key across all shareable devices in the pool.

The 1.37 contract is: when a typed view is emitted, `allocatable` is
the precise bound on fresh-device allocations; `shareableSummary` is
the precise bound on remaining capacity on already-shared devices.
When only the fallback view is emitted,
`counterSets[].counters[].available` is the authoritative counter
signal. Either way the operator has a precise signal — they do not
have to reconstruct it from `availableDevices`.

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
- `get`, `list`, `watch` on `resourceslices` and `resourceclaims`
- standard events permissions

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

No short name (e.g. `rpsr`) is registered in Alpha; adding one is a possible
follow-up for Beta.

### Test Plan

#### Prerequisite testing updates

None required.

#### Unit tests

Coverage targets:
- Pool status computation (`pkg/controller/resourcepoolstatusrequest/controller_test.go`)
- Validation (`pkg/apis/resource/validation/validation_resourcepoolstatusrequest_test.go`)
- Registry strategy / declarative validation (`pkg/registry/resource/resourcepoolstatusrequest/declarative_validation_test.go`)
- Metrics (`pkg/controller/resourcepoolstatusrequest/metrics/metrics_test.go`)
- Printer columns (`pkg/printers/internalversion/printers_test.go`)

Test cases (Alpha 1.36):
- Driver only (all pools for that driver)
- Driver and pool name filter
- No matching pools for driver
- Missing driver field (validation error)
- Various allocation states
- Incomplete pools (observed slice count < expected) produce per-pool
  `validationError`, count fields unset, and requeue
- Older-generation slices ignored (generation handling)
- Mixed / multi-node pools leave `nodeName` unset
- One-time processing (skip if `status != nil`)
- Spec / metadata immutability after status is set
- TTL cleanup: completed (1h) and pending (24h) requests deleted
- `limit` respected; `poolCount` reflects total matches

Additional cases (Alpha 1.37):
- **Cap-at-1 for shareable devices**: a single device with
  `allowMultipleAllocations=true` and three concurrent claims
  contributes exactly `1` to `allocatedDevices`, not `3`.
- **AdminAccess skipped**: an AdminAccess allocation against a
  device does not increment `allocatedDevices`, does not move
  `shareableSummary.partiallyAvailableDevices`, does not contribute to
  `shareableSummary.capacity[].consumed`, does not subtract from any
  `counterSets[].counters[].available` (fallback view), and does not
  reduce any `partitionSummary[].allocatable` (typed view).
- **`unavailableDevices` from taints**: a pool with `M` devices,
  `K` of which carry a `NoSchedule` or `NoExecute` taint (via
  `Spec.Devices[].Taints` or matching `DeviceTaintRule`), reports
  `unavailableDevices=K`. The `DeviceTaintRule` branch of this test
  must explicitly enable the `DRADeviceTaintRules` gate (Beta
  default-off as of 1.36); the embedded-taint branch only needs
  `DRADeviceTaints` (Beta default-on).
- **`partitionSummary[]` aggregation (typed view)**: a pool whose
  slice declares
  `sharedCounters: [{name: gpu-0, counters: {memory: {value: 80Gi}}}]`,
  `spec.partitionTypeAttribute: example.com/partitionType`, and three
  device entries (1 `full` consuming 80Gi, 2 `half` consuming 40Gi each;
  all drawing from `gpu-0`). Walk four states using
  `allocatable[T] = min(freshDevices[T], min over c of floor(counterAvailable[c] / cost[T][c]))`:
  - Nothing in use → `freshFull=1, freshHalf=2, counterAvailable=80Gi`
    → `allocatable[full]=1, allocatable[half]=2`.
  - One half in use → `freshFull=1, freshHalf=1, counterAvailable=40Gi`
    → `allocatable[full]=0, allocatable[half]=1`.
  - The full partition in use → `freshFull=0, freshHalf=2, counterAvailable=0Gi`
    → `allocatable[full]=0, allocatable[half]=0`.
  - **Fresh-device cap binds first** (counter has more headroom than
    devices): same pool with `counters: {memory: {value: 800Gi}}`,
    nothing in use → `allocatable[full]=1, allocatable[half]=2`
    (not 10/20). Confirms the `freshDevices[T]` clamp.
- **Per-device counter consumption (cap-at-1 for counters)**: a pool
  with one shareable partition (`allowMultipleAllocations=true`,
  `consumesCounters: {gpu-0: {memory: 40Gi}}`) reserved by 3 concurrent
  non-AdminAccess claims reports `counterSets[0].counters[memory].consumed`
  = 40Gi (not 120Gi), matching the scheduler's per-device counter
  accounting. The same pool with the typed view reports
  `partitionSummary[half].allocatable` computed against
  `counterAvailable = capacity − 40Gi`, not `capacity − 120Gi`.
- **`partitionSummary[]` validation**: a pool whose grouping
  attribute resolves to two devices of partition type `full` with
  different `ConsumesCounters` costs (one declares 80Gi, the other
  60Gi) produces a per-pool `validationError` instead of an
  inconsistent `allocatable`. A pool where some slices declare
  `PartitionTypeAttribute` and others do not produces a per-pool
  `validationError`. A pool where slices declare different
  `PartitionTypeAttribute` values produces a per-pool
  `validationError`.
- **`counterSets[]` aggregation (fallback view)**: a pool whose
  slice declares
  `sharedCounters: [{name: memory, counters: {memory: {value: 80Gi}}}]`
  but no `PartitionTypeAttribute`, with **two distinct devices** each
  declaring `consumesCounters: {memory: {memory: 30Gi}}` and each
  reserved by exactly one non-AdminAccess claim, reports
  `counterSets[0].counters[memory] = {capacity: 80Gi, consumed: 60Gi, available: 20Gi}`
  (two in-use devices × 30Gi per-device cost) and no
  `partitionSummary[]`.
- **`partitionSummary[]` and `counterSets[]` mutual exclusion**: on
  a pool with `sharedCounters` and a declared
  `PartitionTypeAttribute`, only `partitionSummary[]` is populated;
  `counterSets[]` is absent. On a pool with `sharedCounters` and no
  attribute, only `counterSets[]` is populated.
- **`shareableSummary` aggregation**: a pool with three devices
  (`nic-0`, `nic-1`, `nic-2`, all `allowMultipleAllocations=true`,
  each with `bandwidth=10Gi`), where `nic-0` has two claims totalling
  7Gi and `nic-1` has one claim of 2Gi and `nic-2` has no claims,
  reports
  `shareableSummary = {fullyAvailableDevices: 1, partiallyAvailableDevices: 2,
  capacity: [{name: bandwidth, total: 30Gi, consumed: 9Gi, available: 21Gi}]}`.
- **`shareableSummary` heterogeneous-schema handling**: a pool with
  two devices that carry different capacity keys (`nic-a` has
  `bandwidth=10Gi`, `nic-b` has `packets-per-sec=1M`) produces a
  `capacity[]` with two entries; each entry's `total` only sums
  the device(s) that carry that key.
- **All three sub-objects omitted on plain pools**: a pool with no
  `sharedCounters` and no `allowMultipleAllocations=true` device
  produces a `PoolStatus` with `partitionSummary`, `counterSets`,
  and `shareableSummary` all absent (confirms `omitempty` behaviour
  on two slice fields and one pointer field).
- **`+k8s:maxItems` truncation**: a pool with >32 counter sets or
  >32 distinct capacity keys in `shareableSummary.capacity[]` yields
  a `validationError` rather than silent truncation. The controller
  measures size before populating the field and writes
  `validationError` directly when over-cap, avoiding a rejected
  write against the apiserver.

#### Integration tests

Located at `test/integration/dra/resourcepoolstatusrequest_test.go`. These
verify controller behavior end-to-end against a real apiserver with fake /
in-memory driver data.

Test cases (Alpha 1.36):
1. Controller starts, watches requests, and processes new ones
2. Status populated with correct pool data
3. Processed requests are skipped (one-time processing)
4. Per-pool `validationError` set for incomplete pools; device counts unset
5. `limit` respected and truncation reflected via `poolCount` vs `len(pools)`
6. Immutability after status is set (updates rejected)
7. RBAC: controller can update status; users cannot bypass

Additional cases (Alpha 1.37):

8. **Partitionable end-to-end (typed view)**: seed a pool whose
   slice declares `sharedCounters`, `partitionTypeAttribute`, and
   devices that `consumesCounters` from them grouped by attribute
   value; create allocations; assert `partitionSummary[]` is
   populated with the expected per-type `allocatable` and no
   `counterSets[]` is emitted.
9. **Partitionable end-to-end (fallback view)**: seed the same
   pool without `partitionTypeAttribute`; assert `counterSets[]` is
   populated with the expected `consumed`/`available` and no
   `partitionSummary[]` is emitted.
10. **Pool-level `partitionTypeAttribute` validation**: a pool with
    two slices that declare different `partitionTypeAttribute`
    values (and a separate case where some slices set it and others
    do not) produces a per-pool `validationError`; status is not
    emitted for that pool.
11. **Consumable end-to-end**: seed a pool with at least one
    `allowMultipleAllocations=true` device and multiple claims that
    each set `consumedCapacity`; assert `shareableSummary` is
    populated with the expected `fullyAvailableDevices`,
    `partiallyAvailableDevices`, and per-key `capacity[]` aggregates, and
    that `allocatedDevices` is capped at 1 per device.
12. **AdminAccess invisibility**: in addition to a normal claim,
    create an AdminAccess claim against the same device; assert the
    AdminAccess claim does not appear in any tally.
13. **`unavailableDevices` from a `DeviceTaintRule`** (test must
    enable `DRADeviceTaintRules`, Beta default-off as of 1.36):
    create a matching `DeviceTaintRule`, request status, assert the
    `unavailableDevices` count moves accordingly.
14. **Scale (carries forward as a 1.37 addition)**: ≥100 pools and
    ≥1000 expired requests; assert cleanup completes within the
    10-min interval and apiserver QPS for `delete resourcepoolstatusrequests`
    stays under a sensible bound.

#### e2e tests

E2E tests are added to the existing DRA e2e test suite at `test/e2e/dra/dra.go`,
using the existing test-driver (`test/e2e/dra/test-driver/`) behind
`--feature-gate=DRAResourcePoolStatus`.

Test cases already implemented (Alpha 1.36):
1. Conformance-style resource lifecycle (create / get / update labels /
   delete) for `resource.k8s.io/v1alpha3 ResourcePoolStatusRequest`,
   asserting spec immutability via label-only updates.
2. "should report pool status with correct device counts": create a
   request, wait for the `Complete` condition, and assert that the single
   `network` pool reports `totalDevices=10`, `allocatedDevices=0`,
   `availableDevices=10`, `unavailableDevices=0`, `resourceSliceCount=1`,
   `generation=1`, `nodeName=nil`.
3. "should reflect allocated devices after pod is scheduled": schedule a
   pod that consumes devices, then create a new request and assert the
   updated `allocatedDevices` / `availableDevices`.

Added in Alpha 1.37:

4. "should report partition-type availability on a partitionable
   pool": seed the test driver to publish a pool with
   `sharedCounters`, `partitionTypeAttribute`, and devices that
   `consumesCounters`; schedule a pod; assert `partitionSummary[]`
   shows the expected `total` / `allocatable` per type. A parallel
   case seeds the same pool without `partitionTypeAttribute` and
   asserts the `counterSets[]` fallback view.
5. "should report shareable-device aggregate on a consumable pool":
   seed the test driver with two `allowMultipleAllocations=true`
   devices; schedule two pods that each consume a slice of capacity
   on one of them; assert `shareableSummary` reports
   `fullyAvailableDevices=1`, `partiallyAvailableDevices=1`, the per-key
   `capacity[]` aggregate is consistent with the consumed slice,
   and `allocatedDevices=1` (cap-at-1 verified end-to-end).
6. "should not count AdminAccess claims as consumers": create an
   AdminAccess claim against an otherwise-fully-allocated device;
   assert the AdminAccess claim does not move any counter.
7. Tighten existing assertions per the Pohly review feedback —
   replace `gstruct.IgnoreExtras` with `MatchAllFields` where
   feasible, or move detailed field-by-field checks into the unit
   suite.

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
- Per-pool `validationError` reporting for incomplete pools with
  controller-side requeue
- Full object immutability once `status` is set
- Documentation

#### Alpha (1.37)

A second Alpha cycle is targeted instead of an immediate Beta
graduation. The reasoning, strongest first:

1. **The Alpha API does not correctly describe partitionable or
   consumable devices.** The 1.36 controller increments
   `allocatedDevices` per allocation result, which (a) overcounts
   on devices with `allowMultipleAllocations=true` (consumable) and
   (b) does not reflect shared-counter consumption on partitionable
   devices. The visible symptom is `availableDevices=0` reported on
   pools that actually have free capacity. Fixing this requires new
   API fields (`partitionSummary[]` / `counterSets[]`,
   `shareableSummary`) plus a new optional field on `ResourceSlice`
   (`PartitionTypeAttribute`) to drive the typed view — not just a
   controller patch — and adding new API surface in Beta is exactly
   what Alpha cycles exist to avoid. This is the load-bearing
   reason; the points below are supporting evidence.
2. **No production DRA driver has been validated against yet.**
   The original Beta criteria required out-of-tree validation
   against at least one production DRA driver; no driver-side code
   change is needed (the controller reads existing ResourceSlice /
   allocation fields), but operational validation in a real-driver
   environment cannot be back-filled inside the same release that
   graduates to Beta.
3. **Several Alpha reviewer follow-ups remain open** (batched TTL
   deletes, deterministic metrics tests, e2e assertion tightening).
   Resolving them inside another Alpha cycle is lower-stakes than
   gating a Beta promotion on them.
4. **Limited soak.** Alpha shipped in 1.36
   (kubernetes/kubernetes#137028); only one release has elapsed.
   sig-node has graduated faster in the past, so this is supporting
   evidence rather than a hard blocker — but combined with the
   three points above, additional soak in 1.37 is well-motivated.

Scope of the 1.37 Alpha:

- API stays at `resource.k8s.io/v1alpha3`.
- Feature gate `DRAResourcePoolStatus` stays Alpha, default off.
- **Add `ResourceSlice.Spec.PartitionTypeAttribute`** (new optional
  `*FullyQualifiedName` on `ResourceSliceSpec`, gated behind
  `DRAResourcePoolStatus`) so drivers can declare a per-pool
  grouping attribute for partition types. The slice schema change
  requires sign-off from KEP-4815 since the field lives on the
  `ResourceSlice` type owned by that KEP.
- **Add `partitionSummary[]` to `PoolStatus`** (`type`, `total`,
  `allocatable` per partition type) for partitionable pools whose
  slices declare `PartitionTypeAttribute`. `allocatable` nets out
  shared-counter consumption directly via
  `min(freshDevices[type], min over c of floor(counterAvailable[c] / cost[type][c]))`,
  so this view replaces the raw counter dump in the common case.
- **Add `counterSets[]` to `PoolStatus`** as a fallback view, used
  only on pools that have `sharedCounters` but no declared
  `PartitionTypeAttribute`. The two are mutually exclusive per pool.
- **Add `shareableSummary` to `PoolStatus`** (`fullyAvailableDevices`,
  `partiallyAvailableDevices`, plus per-capacity-key `total`/`consumed`/`available`
  aggregates) for pools that contain at least one device with
  `allowMultipleAllocations=true`.
- **Cap the per-device contribution to `allocatedDevices` at 1**,
  fixing the consumable overcount.
- **Skip AdminAccess allocations** in all device, counter, and
  shareable-device tallies.
- **Compute `unavailableDevices`** from real device taints
  (`NoSchedule` / `NoExecute`), replacing the Alpha 1.36 hard-coded
  `0`.
- **Batch / pace TTL-delete sweeps** so a large cleanup does not
  spike apiserver QPS (Alpha reviewer follow-up).
- **Deterministic metrics tests** using a `synctest` bubble,
  replacing Alpha 1.36's string-dump assertions (Alpha reviewer
  follow-up).
- **Tighten e2e assertions** — replace `gstruct.IgnoreExtras` with
  `MatchAllFields` where feasible, or move detailed field-by-field
  checks into unit tests (Alpha reviewer follow-up).
- **Scale validation** at ≥100 pools with ≥1000 expired requests
  via an integration benchmark.
- **Best-effort production DRA driver validation** (out-of-tree) —
  coordinate with at least one driver maintainer if one is available
  within the 1.37 window. Not a hard gate for the second Alpha; a
  hard gate for the eventual Beta promotion.

#### Beta

Beta criteria will be revisited after the Alpha 1.37 work lands
(`partitionSummary[]` / `counterSets[]` / `shareableSummary`,
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
  (`partitionSummary[]` / `counterSets[]`, `shareableSummary`) are
  populated by the 1.37 controller when the source data warrants it.
  Older clients ignore the unknown fields.
- `ResourceSlice.Spec.PartitionTypeAttribute` (new in `resource.k8s.io/v1`,
  gated behind `DRAResourcePoolStatus`) is an additive optional
  field. Slices written by 1.36 leave it unset, so the 1.37
  controller emits the `counterSets[]` fallback view for those
  pools. Drivers that adopt the convention opt in slice by slice;
  partial adoption per pool is rejected with a per-pool
  `validationError` to avoid mixed-view confusion.
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
  both 1.36 and 1.37, so both components must opt in explicitly.
- **1.36 ↔ 1.37 skew:** Status API is `resource.k8s.io/v1alpha3` in
  both releases. A 1.37 KCM serving a 1.36 apiserver may emit
  `partitionSummary[]` / `counterSets[]` / `shareableSummary` on
  objects whose 1.36 apiserver storage understands them as opaque
  optional fields — no compatibility issue. A 1.36 KCM serving a
  1.37 apiserver simply does not populate the new fields.
- **`ResourceSlice.Spec.PartitionTypeAttribute` skew:** the field
  lives in served `resource.k8s.io/v1`. A 1.37 apiserver with the
  feature gate disabled (default) drops the field on write, so
  drivers that set it on a gate-disabled cluster see it silently
  cleared — same shape as other gated optional fields. A 1.36
  apiserver does not know the field and rejects it on write. The
  controller-side check (cross-slice consistency) tolerates the
  unset case via the fallback view.
- **Older kubectl** can create/read objects via the standard
  `v1alpha3` endpoint without changes.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: DRAResourcePoolStatus
  - Components: kube-apiserver, kube-controller-manager

###### Does enabling the feature change any default behavior?

No. Users must explicitly create ResourcePoolStatusRequest objects.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disable the feature gate. Existing requests become inaccessible but
no workload impact.

###### What happens if we reenable the feature if it was previously rolled back?

Existing requests (if any) become visible again. Unprocessed requests will
be processed by the controller.

###### Are there any tests for feature enablement/disablement?

Yes, integration tests verify behavior with feature gate on/off.

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

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No, the controller will expose the standard metrics listed above.

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
`DRAResourcePoolStatus`). It is a single, bounded string per slice
and is omitted unless the driver opts in, so per-slice size is
effectively unchanged on existing clusters.

Alpha 1.37 also adds optional `partitionSummary[]` (`+k8s:maxItems=32`,
provisional), `counterSets[]` (`+k8s:maxItems=32`, provisional —
emitted as fallback only), and `shareableSummary` (a fixed-shape
sub-object with an inner `capacity[]` capped at `+k8s:maxItems=32`)
to each `PoolStatus`. All three are omitted on plain pools, so the
typical response size is unchanged; on partitionable or consumable
pools the response grows by a bounded, small amount
(`partitionSummary[]` and `shareableSummary` are much smaller than
the per-device list they replace, and `partitionSummary[]` is in
turn typically smaller than the fallback `counterSets[]` dump it
supersedes).

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No impact on scheduling or pod startup.

###### Will enabling / using this feature result in non-negligible increase of resource usage?

Minimal:
- etcd: Small objects, bounded by built-in TTL cleanup (Alpha: 1h completed / 24h pending)
- KCM: Reuses existing `resource.k8s.io/v1` informers for ResourceSlice and ResourceClaim, adds a small controller with its own work queue
- API server: Standard API operations
- Response size: Bounded by the required `driver` field (one driver's pools), the `limit` field (default 100, max 1000), the `+k8s:maxItems=1000` constraint on `status.pools`, and (for Alpha 1.37) `+k8s:maxItems=32` on each of `partitionSummary[]`, `counterSets[]`, and `shareableSummary.capacity[]` per pool

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
| Incomplete pool data | Fewer slices published than `ResourceSliceCount` declared by the driver | `pools[].validationError` set; count fields unset; controller requeues up to 5 times | Ensure driver fully publishes slices; retry by recreating request | Inspect `status.pools[].validationError`; check driver logs | Covered by unit and integration tests |
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
- 1.37 (Alpha, planned): second Alpha cycle on `v1alpha3` to
  correctly handle partitionable and consumable devices — see
  "Alpha (1.37)" in Graduation Criteria.

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
