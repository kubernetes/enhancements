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
  - [Controller Implementation](#controller-implementation)
    - [Controller in KCM](#controller-in-kcm)
    - [One-time Processing](#one-time-processing)
    - [Incomplete-Pool Handling and Requeue](#incomplete-pool-handling-and-requeue)
    - [Reusing Existing Informers](#reusing-existing-informers)
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
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and
  SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests]
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
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

6. **Partitionable devices**: Device counts may be misleading for partitionable
   devices (e.g., a single GPU that can be split into 15 mutually exclusive
   partitions). The controller counts devices as listed in ResourceSlices
   without understanding partition relationships. Future versions may add
   driver-provided metadata to indicate partitioning.

7. **Incomplete pools**: When a pool's observed ResourceSlice count is less
   than `ResourceSliceCount` declared by the driver, the pool is reported
   with `validationError` set and device-count fields left unset. The
   controller requeues the request (up to 5 attempts) to give drivers time
   to publish remaining slices.

8. **Generation handling**: ResourceSlices with older pool generations are
   ignored during computation (not counted as errors). Drivers are expected
   to delete old-generation slices eventually. The `generation` field in
   each PoolStatus reflects the highest generation observed.

9. **`unavailableDevices` in Alpha**: Currently always `0`. Inspection of
   device taints/conditions to populate this field is planned for Beta.

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
  poolCount: 2

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
    unavailableDevices: 0         # Always 0 in Alpha
  - driver: example.com/gpu
    poolName: node-2
    generation: 5
    # validationError is set when a pool is incomplete (observed < expected
    # slice count). When set, count fields are unset. Max 256 bytes.
    validationError: "pool example.com/gpu/node-2 is incomplete: observed 1/2 slices at generation 5"

  # Conditions indicating processing status.
  # Known types: "Complete" (True when processed successfully) and
  # "Failed" (True when the request could not be processed). Max 10 entries.
  conditions:
  - type: Complete
    status: "True"
    reason: CalculationComplete
    message: "Calculated status for 2 pools (1 incomplete)"
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
| `pools[].unavailableDevices` | `*int32` (optional) | Devices not available due to taints/conditions. **Always 0 in Alpha** (not yet computed). |
| `pools[].validationError` | `*string` (optional, max 256 bytes) | Set when the pool's data could not be fully validated (e.g., incomplete slice publication). When set, count fields above may be unset. |
| `conditions[]` | map list by `type`, max 10 | `Complete` (True when processed) or `Failed` (True on error). |

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

Test cases:
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

#### Integration tests

Located at `test/integration/dra/resourcepoolstatusrequest_test.go`. These
verify controller behavior end-to-end against a real apiserver with fake /
in-memory driver data.

Test cases:
1. Controller starts, watches requests, and processes new ones
2. Status populated with correct pool data
3. Processed requests are skipped (one-time processing)
4. Per-pool `validationError` set for incomplete pools; device counts unset
5. `limit` respected and truncation reflected via `poolCount` vs `len(pools)`
6. Immutability after status is set (updates rejected)
7. RBAC: controller can update status; users cannot bypass

#### e2e tests

E2E tests are added to the existing DRA e2e test suite at `test/e2e/dra/dra.go`,
using the existing test-driver (`test/e2e/dra/test-driver/`) behind
`--feature-gate=DRAResourcePoolStatus`.

Test cases (already implemented):
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

#### Beta

- E2E tests passing in CI (using test-driver)
- Validated with at least one production DRA driver (out-of-tree testing)
- Performance validated at scale (100+ pools)
- User feedback incorporated
- Compute `unavailableDevices` from real device taints/conditions
  (currently always 0 in Alpha)
- Consider adding a `rpsr` short name
- Consider per-user rate limiting for request creation
- Consider configurable TTLs
- Consider namespace-scoped variant if requested

#### GA

- At least 2 releases as beta
- Validated at scale (1000+ pools)
- kubectl plugin for better UX (optional)
- Documentation complete

### Upgrade / Downgrade Strategy

**Upgrade:**
- Enable feature gate
- New API becomes available
- No migration needed

**Downgrade:**
- Disable feature gate
- Existing requests become inaccessible
- No impact on workloads

### Version Skew Strategy

- API server and KCM must both have feature enabled
- Older kubectl can still create/read objects (standard API)
- No special version skew concerns

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

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No impact on scheduling or pod startup.

###### Will enabling / using this feature result in non-negligible increase of resource usage?

Minimal:
- etcd: Small objects, bounded by built-in TTL cleanup (Alpha: 1h completed / 24h pending)
- KCM: Reuses existing `resource.k8s.io/v1` informers for ResourceSlice and ResourceClaim, adds a small controller with its own work queue
- API server: Standard API operations
- Response size: Bounded by required `driver` field (one driver's pools), the `limit` field (default 100, max 1000), and the `+k8s:maxItems=1000` constraint on `status.pools`

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
- 1.36 (Alpha): feature gate `DRAResourcePoolStatus` (default off)

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
