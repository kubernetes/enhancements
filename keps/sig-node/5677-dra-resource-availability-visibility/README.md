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
    - [Reusing Existing Informers](#reusing-existing-informers)
  - [kubectl Integration](#kubectl-integration)
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
│     resourcepoolstatusrequest    --for=condition=Complete  rpsr/my-check    │
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
│  │    driver: example.com/gpu    ───►    observationTime: <timestamp>    │  │
│  │    poolName: node-1                   pools:                          │  │
│  │                                       - driver: example.com/gpu       │  │
│  │                                         poolName: node-1              │  │
│  │                                         totalDevices: 4               │  │
│  │                                         allocatedDevices: 3           │  │
│  │                                         availableDevices: 1           │  │
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
│  │  2. Skip if status.observationTime already set (one-time processing)   │ │
│  │  3. Read ResourceSlices matching spec filters (driver, poolName)       │ │
│  │  4. Read ResourceClaims to determine allocations                       │ │
│  │  5. Compute availability summary per pool                              │ │
│  │  6. Write result to status with timestamp                              │ │
│  │  7. Set condition Complete=True                                        │ │
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
apiVersion: resource.k8s.io/v1alpha1
kind: ResourcePoolStatusRequest
metadata:
  name: check-gpus-$(date +%s)
spec:
  driver: example.com/gpu
EOF
resourcepoolstatusrequest.resource.k8s.io/check-gpus-1707300000 created

# Wait for processing
$ kubectl wait --for=condition=Complete rpsr/check-gpus-1707300000 --timeout=30s
resourcepoolstatusrequest.resource.k8s.io/check-gpus-1707300000 condition met

# View results
$ kubectl get rpsr/check-gpus-1707300000 -o yaml
apiVersion: resource.k8s.io/v1alpha1
kind: ResourcePoolStatusRequest
metadata:
  name: check-gpus-1707300000
spec:
  driver: example.com/gpu
status:
  observationTime: "2026-02-07T10:30:00Z"
  pools:
  - driver: example.com/gpu
    poolName: node-1
    nodeName: node-1
    totalDevices: 4
    allocatedDevices: 3
    availableDevices: 1
  - driver: example.com/gpu
    poolName: node-2
    nodeName: node-2
    totalDevices: 4
    allocatedDevices: 1
    availableDevices: 3
  conditions:
  - type: Complete
    status: "True"
    reason: CalculationComplete
    lastTransitionTime: "2026-02-07T10:30:00Z"

# Cleanup
$ kubectl delete rpsr/check-gpus-1707300000
```

#### Story 2: Developer Debugging Resource Allocation

As a developer, when my pod fails to schedule because "insufficient DRA
resources", I want to understand what resources are available.

**Workflow:**
```bash
# Quick one-liner to check GPU availability
$ kubectl create -f - <<EOF && sleep 2 && \
  kubectl get rpsr/debug-check -o jsonpath='{.status.pools[*]}'
apiVersion: resource.k8s.io/v1alpha1
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
apiVersion: resource.k8s.io/v1alpha1
kind: ResourcePoolStatusRequest
metadata:
  name: $REQUEST_NAME
spec:
  driver: $DRIVER
EOF

# Wait and get result
kubectl wait --for=condition=Complete rpsr/$REQUEST_NAME --timeout=60s
AVAILABLE=$(kubectl get rpsr/$REQUEST_NAME -o jsonpath='{.status.pools[*].availableDevices}' | tr ' ' '+' | bc)

# Alert if low
if [ "$AVAILABLE" -lt 5 ]; then
  echo "ALERT: Only $AVAILABLE devices available cluster-wide"
fi

# Cleanup
kubectl delete rpsr/$REQUEST_NAME
```

### Notes/Constraints/Caveats

1. **Asynchronous operation**: Unlike SubjectAccessReview (synchronous), this
   uses the CSR pattern where user must wait for controller processing.

2. **One-time calculation**: Each request is processed once. To get updated
   data, delete and recreate the request.

3. **Object persists in etcd**: Requests are stored until deleted. Users should
   clean up old requests or use unique names with timestamps.

4. **Controller processing delay**: Status is not immediate - controller must
   process the request. Typically completes within seconds.

5. **RBAC controls access**: Users need RBAC permission to create/read
   ResourcePoolStatusRequest objects to use this feature.

6. **Partitionable devices**: Device counts may be misleading for partitionable
   devices (e.g., a single GPU that can be split into 15 mutually exclusive
   partitions). The controller counts devices as listed in ResourceSlices
   without understanding partition relationships. Future versions may add
   driver-provided metadata to indicate partitioning.

7. **Generation handling**: ResourceSlices with older pool generations are
   ignored during computation (not counted as errors). Drivers are expected
   to delete old-generation slices eventually. The `generation` field in
   status reflects the highest generation observed.

### Risks and Mitigations

#### Scaling Risks

| Risk | Mitigation |
|------|------------|
| Request accumulation in etcd | Document cleanup; consider TTL for Beta |
| Large status objects (many pools) | Required `driver` field bounds response; `limit` field caps further |
| Controller processing spike | Work queue with rate limiting |
| Simultaneous request flood | Rate limiting per user (Beta) |

**Alpha approach:** For Alpha, the required `driver` field naturally bounds
response size to one driver's pools, with `limit` as an additional cap. We rely
on documentation and controller-side rate limiting. Cluster administrators
can enforce object count limits using admission webhooks (e.g., Gatekeeper,
Kyverno) if needed for their environment. This is sufficient for initial
adoption while we gather feedback on usage patterns.

**Beta improvements:** TTL-based auto-cleanup (`ttlSecondsAfterComplete`) and
per-user rate limiting will be added. If Alpha feedback indicates a need, we
may also add a built-in cluster-wide object limit (similar to how some APIs
cap total objects).

#### Operational Risks

| Risk | Mitigation |
|------|------------|
| Stale data if not recalculated | Timestamp shows age; recreate for fresh |
| Controller not running | Condition stays pending; user can detect |
| Feature gate mismatch | Document both apiserver and KCM required |

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

```yaml
apiVersion: resource.k8s.io/v1alpha1
kind: ResourcePoolStatusRequest
metadata:
  name: my-request
  # Cluster-scoped (no namespace)
spec:
  # Driver is REQUIRED - bounds response to one driver's pools
  driver: example.com/gpu

  # Filter by pool name (optional)
  poolName: node-1

  # Limit number of pools returned (optional)
  # Default and max values will be determined at implementation time
  # based on max-size calculations to ensure the object fits in etcd
  limit: 100

status:
  # Timestamp when calculation was performed
  observationTime: "2026-02-07T10:30:00Z"

  # List of pools matching the filter
  pools:
  - driver: example.com/gpu
    poolName: node-1
    nodeName: node-1              # Empty for non-node-local pools
    totalDevices: 4
    allocatedDevices: 3
    availableDevices: 1
    unavailableDevices: 0         # Devices with constraints
    sliceCount: 1                 # ResourceSlices in this pool
    generation: 5                 # Pool generation observed

  # Conditions indicating processing status
  conditions:
  - type: Complete
    status: "True"
    reason: CalculationComplete
    message: "Processed 2 pools"
    lastTransitionTime: "2026-02-07T10:30:00Z"

  # Validation errors found (max 10 entries, max 256 chars each)
  validationErrors:
  - "pool node-1: device gpu-0 appears in multiple slices"

  # True if more pools exist but were not included due to limit
  truncated: false

  # Total number of pools matching filter (even if truncated)
  totalMatchingPools: 2
```

#### Spec Fields

The spec is **immutable after creation**, following the CSR pattern. Updates
to the spec fields will be rejected by API validation. To query with different
filters or get fresh data, delete and create a new request.

| Field | Description |
|-------|-------------|
| `driver` | Driver name (**required**) - bounds response to one driver's pools |
| `poolName` | Filter by pool name (optional) |
| `limit` | Max pools to return (optional, values determined at implementation based on size calculations) |

#### Status Fields

| Field | Description |
|-------|-------------|
| `observationTime` | Timestamp when calculation was performed |
| `pools[]` | List of pools matching filter (up to limit) |
| `pools[].driver` | DRA driver name |
| `pools[].poolName` | Pool name from ResourceSlice |
| `pools[].nodeName` | Node name (for node-local pools) |
| `pools[].totalDevices` | Total devices across all slices |
| `pools[].allocatedDevices` | Devices allocated to claims |
| `pools[].availableDevices` | Devices available for allocation |
| `pools[].unavailableDevices` | Devices unavailable (constraints) |
| `pools[].sliceCount` | Number of ResourceSlices in pool |
| `pools[].generation` | Pool generation observed |
| `conditions[Complete]` | Processing completed successfully |
| `validationErrors` | Pool consistency issues (max 10, 256 chars each) |
| `truncated` | True if more pools exist beyond limit |
| `totalMatchingPools` | Total pools matching filter |

### Controller Implementation

#### Controller in KCM

The controller is added to kube-controller-manager as a separate controller
with its own client and QPS limits (not part of the resourceclaim controller).
This ensures client-side throttling does not impact scheduling.

The controller:
1. Watches ResourcePoolStatusRequest objects via informer
2. Maintains a rate-limited work queue for processing
3. Reuses existing ResourceSlice and ResourceClaim informers from KCM
4. Uses `UpdateStatus` to write results to the status subresource

#### One-time Processing

Following the CSR pattern, the controller processes each request exactly once:

1. When a new ResourcePoolStatusRequest is created, it is added to work queue
2. Controller checks if `status.observationTime` is already set
3. If set, the request was already processed - controller skips it
4. If not set, controller computes pool status and writes to status
5. Once status is written, the request is never processed again

To get fresh data, users delete and recreate the request.

#### Reusing Existing Informers

The controller reuses ResourceSlice and ResourceClaim informers that are
already running in KCM for the device-taint-eviction controller. This adds
minimal overhead since the informers are already cached in memory.

The controller constructor accepts these shared informers rather than
creating its own, following the established KCM pattern.

### kubectl Integration

Standard kubectl commands work:

```bash
# Create request
$ kubectl create -f request.yaml

# Wait for completion
$ kubectl wait --for=condition=Complete rpsr/my-request

# Get status
$ kubectl get rpsr/my-request -o yaml

# List all requests
$ kubectl get rpsr

# Delete request
$ kubectl delete rpsr/my-request
```

Short name `rpsr` is registered for convenience.

### Test Plan

#### Prerequisite testing updates

None required.

#### Unit tests

Coverage targets:
- Pool status computation: 80%+
- Cross-slice validation: 80%+
- Controller logic: 75%+

Test cases:
- Driver only (all pools for that driver)
- Driver and pool name filter
- No matching pools for driver
- Missing driver field (validation error)
- Various allocation states
- Cross-slice validation errors
- Generation handling
- One-time processing (skip if processed)

#### Integration tests

Integration tests verify controller logic using **fake clients and informers**
without requiring a real cluster or DRA driver. These tests focus on the
controller's internal behavior.

Test cases:
1. Controller starts and watches requests
2. New request triggers processing
3. Status updated with correct pool data
4. Processed requests are skipped (one-time processing)
5. Validation errors detected and reported
6. RBAC enforced correctly
7. Limit field respected, truncated flag set correctly

#### e2e tests

E2E tests will be added to the existing DRA e2e test suite at `test/e2e/dra/`,
using the **existing test-driver** (`test/e2e/dra/test-driver/`) that is
already available in CI. This test-driver publishes ResourceSlices and
supports creating ResourceClaims, which is sufficient for testing
ResourcePoolStatusRequest functionality.

Test cases:
1. Create ResourcePoolStatusRequest via kubectl
2. Wait for condition Complete using `kubectl wait`
3. Verify status contains expected pools via `kubectl get`
4. Create ResourceClaim, create new request, verify updated counts
5. Delete and recreate request, verify fresh data
6. Test with multiple pools from the test-driver
7. Delete request via `kubectl delete`, verify cleanup

Note: Testing with production DRA drivers (e.g., GPU drivers) is outside
the scope of CI and would be validated separately by driver vendors.

### Graduation Criteria

#### Alpha

- API defined and implemented
- Controller in KCM behind feature gate
- Basic kubectl workflow works
- Unit and integration tests
- Documentation

#### Beta

- E2E tests passing in CI (using test-driver)
- Validated with at least one production DRA driver (out-of-tree testing)
- Performance validated at scale (100+ pools)
- User feedback incorporated
- Add TTL field for automatic cleanup (`ttlSecondsAfterComplete`)
- Add per-user rate limiting for request creation
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
- Check controller metrics: `resourcepoolstatus_requests_processed_total > 0`

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: N/A (no events emitted)
- [x] API .status
  - Condition name: `Complete` (status: "True" when processing finished)
  - Other field: `status.observationTime` is set when calculation is performed
- [ ] Other (Alarm, К8s resources status)

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- Request processing: 99% of requests complete within 30 seconds
- No impact on existing scheduling or pod startup SLOs

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `resourcepoolstatus_request_processing_duration_seconds`
    - Aggregation method: histogram
    - Components exposing the metric: kube-controller-manager
  - Metric name: `resourcepoolstatus_request_processing_errors_total`
    - Aggregation method: counter
    - Components exposing the metric: kube-controller-manager
  - Metric name: `resourcepoolstatus_requests_processed_total`
    - Aggregation method: counter
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

Note: Objects are intended to be short-lived. Users should delete requests after reading
the status. TTL-based auto-cleanup will be added in Beta.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No impact on scheduling or pod startup.

###### Will enabling / using this feature result in non-negligible increase of resource usage?

Minimal:
- etcd: Small objects, users should clean up (TTL in Beta)
- KCM: Reuses existing informers, adds small controller
- API server: Standard API operations
- Response size: Bounded by required `driver` field (one driver's pools) and optional `limit` field (values determined at implementation based on size calculations)

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
| Controller not running | ResourcePoolStatusRequest controller in KCM is not running or crashed | Requests stay in pending state (no `status.observationTime`), `resourcepoolstatus_requests_processed_total` metric stays at 0 | Restart KCM, check KCM logs | Check KCM logs for controller startup errors, verify feature gate enabled | Covered by integration tests |
| Informers not synced | ResourceSlice or ResourceClaim informers have not completed initial sync | Controller logs warning, requests delayed | Wait for informer sync, check API server connectivity | Check KCM logs for informer sync status | Covered by integration tests |
| Request accumulation | Users create many requests without cleanup | etcd storage grows, `kubectl get rpsr` shows many objects | Delete old requests, implement cleanup automation | List requests with `kubectl get rpsr`, check etcd metrics | Documented, TTL planned for Beta |

###### What steps should be taken if SLOs are not being met?

1. Check KCM logs for controller errors
2. Check controller metrics
3. Verify informers are synced
4. Check for excessive request volume

## Implementation History

- 2025-12-20: KEP created in provisional state
- 2026-01-15: Design revision - ResourceSlice status to ResourcePool
- 2026-02-07: Design revision - in-tree CSR-like pattern per API review

## Drawbacks

1. **Asynchronous operation**: User must wait for controller, unlike sync APIs
   - Mitigation: Processing is fast (seconds); kubectl wait helps

2. **Objects persist in etcd**: Users must clean up old requests
   - Mitigation: Document cleanup; consider TTL in future

3. **Not real-time**: Shows point-in-time snapshot, not live data
   - Mitigation: Timestamp shows age; recreate for fresh data

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
