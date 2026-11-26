# KEP-6472: Show Pod extended resources and ResourceClaims in kubectl describe node

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Relationship to KEP-5677](#relationship-to-kep-5677)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Explicit ResourceClaims](#explicit-resourceclaims)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

This KEP is provisional. SIG sponsorship, reviewers, release target, and release plan are not yet agreed.

Tracking issue: [#6472](https://github.com/kubernetes/enhancements/issues/6472).

- [ ] (R) Enhancement issue in release milestone, which links to KEP directory
- [ ] (R) KEP approved as `implementable`
- [ ] (R) Design and test plan agreed
- [ ] (R) Graduation criteria agreed
- [ ] (R) Production readiness review completed and approved
- [ ] Implementation history updated
- [ ] User-facing documentation and discussion links added

## Summary

Show which Pods on a node request extended resources or use explicit Dynamic Resource Allocation (DRA) ResourceClaims. Add two tables to `kubectl describe node`: `Extended Resources by Pod` and `ResourceClaims by Pod`. Keep the existing tables and accounting. No server changes or new APIs are needed.

## Motivation

A node can report `nvidia.com/gpu: 8` under `Allocated resources`, but the `Non-terminated Pods` table only shows CPU and memory. Users cannot tell which Pods requested those eight resource units from this output. For Pods using explicit ResourceClaims, users must look up the referenced Claims separately.

[PR #46079] added the node totals. It left out Pod details because long resource names made the Pod table hard to format. A separate table can show these details without adding columns to the existing table.

### Goals

- Show each listed Pod's extended resource requests and limits, regardless of the allocation backend.
- Show explicit Claim requests and whether each Claim has an allocation.
- Keep existing node accounting and avoid vendor-specific rules.

### Non-Goals

Report device IDs, runtime use, device health, or free capacity. Change scheduling, resource accounting, or `kubectl top`.

## Proposal

After `Allocated resources`, print each new table only if it has rows. For extended resources:

```text
Extended Resources by Pod:
  Namespace  Name                    Resource                   Requests  Limits
  ---------  ----                    --------                   --------  ------
  public     pod-using-2-gpu          nvidia.com/gpu              2         2
  public     pod-using-mig            nvidia.com/mig-1g.10gb       1         1
  public     pod-using-hami           nvidia.com/gpu              1         1
  public     pod-using-hami           nvidia.com/gpucores         50        50
  public     pod-using-hami           nvidia.com/gpumem           8192      8192
  public     pod-using-gpu-and-rdma   nvidia.com/gpu              1         1
  public     pod-using-gpu-and-rdma   rdma/rdma_shared_device_a   1         1
```

These are sample values, not a report from one cluster. Each row describes one Pod and one resource. The number of columns stays the same as resource types increase. Nodes without such requests or limits get no extended resource table.

The last Pod requests both GPU and shared RDMA resources. It has two rows with the same namespace and Pod name, one for each resource.

For Pods that reference explicit ResourceClaims:

```text
ResourceClaims by Pod:
  Namespace  Name           Claim/Request     DeviceClass  Requested  Claim Allocation
  ---------  ----           -------------     -----------  ---------  ----------------
  demo       pod-using-dra  job-devices/gpu   gpu-class    2          allocated
  demo       pod-using-dra  job-devices/rdma  rdma-class   1          allocated
  demo       pod-using-all  all-gpus/gpu      gpu-class    All        allocated
```

Each row shows one Pod and one request in a Claim. Keep the same Pod order and sort by Claim and request within each Pod. `Requested` describes the Claim request, not a count of devices held by the Pod. `Claim Allocation` reports whether the Claim has an allocation; it does not indicate whether the Pod can use the devices yet. If Pods share a Claim, it can appear under each Pod. Do not sum these rows or add them to the existing node totals. A Pod that uses both request paths can appear in both tables.

### Relationship to KEP-5677

[KEP-5677] adds `ResourcePoolStatusRequest` to report DRA pool availability. A controller reads ResourceSlices and ResourceClaims and returns a snapshot. Its result is a pool summary, rather than a list of Pod requests. Users read it through `kubectl get resourcepoolstatusrequest`.

| | This KEP | KEP-5677 |
| --- | --- | --- |
| Question | Which Pods request how much? | How much DRA pool capacity is available? |
| Row | Pod and extended resource or Claim request | Driver and pool, with capacity summaries |
| Input | Pod requests and referenced ResourceClaims | ResourceSlices and ResourceClaims |
| Change | kubectl output | API and controller |
| Resource path | Extended resources and explicit Claim requests | DRA pools |

This KEP does not depend on KEP-5677 or calculate DRA availability. It shows scalar requests, including keys served by DRA, and explicit Claim requests. It does not turn Claim allocations into device counts. Existing node-local ResourceSlice summaries stay unchanged.

### Risks and Mitigations

The output grows by one row per nonzero Pod/resource pair and per referenced Pod/Claim/request. If a Claim cannot be read, a single row shows the reference and the reason details are unavailable. A request for eight units of one resource still adds one row. Pods with several resource keys or Claim requests add several rows. Show all matching rows without silent truncation. Existing rows and columns keep their format. Scripts that parse the whole text may need updates; scripts should use structured API output.

A resource unit may mean a device, a partition, or a share. Keep the full resource name and quantity. Do not label the value as physical GPU count, convert vendor units, or add percentages.

## Design Details

Both tables reuse the Pod list read by `NodeDescriber.Describe` in `staging/src/k8s.io/kubectl/pkg/describe/describe.go`. Include Pods bound to this node, even if they are still starting. Exclude unscheduled Pods and Pods in `Succeeded` or `Failed` phase. Keep non-terminal Pods being deleted and API-visible mirror Pods. Node, Pod, and Claim reads are not an atomic snapshot.

For extended resources:

1. Reuse each Pod's `PodRequests` and `PodLimits` results and options. This keeps init container, sidecar, overhead, and status handling consistent.
2. From the union of those maps, select names using the same rule as the current `Allocated resources` table: `!kubectlresourcehelper.IsStandardContainerResourceName(string(name)) && name != corev1.ResourcePods`. This excludes CPU, memory, huge pages, ephemeral storage, and `pods`.
3. Keep a pair if either quantity is nonzero. A missing quantity prints as `0`, as in the existing tables. Format quantities with `Quantity.String()`.
4. Keep the same Pod order as the current table. Sort resource names within each Pod. Print full namespace, Pod, and resource names on every row.

Ephemeral containers have no separate resource requests or limits; use the existing helpers without adding special accounting.

The extended resource rows and `Allocated resources` totals use the same Pod list and helper options. For each resource shown in both sections, the row sums must match the totals. Keys absent from Node allocatable can appear only in the per-Pod table.

Collect rows during the existing Pod loop and print them after the current totals. Do not calculate each Pod's resources again for the new table. Use the existing `PrefixWriter` and tabwriter; do not set a fixed line width or truncate names.

The `Resource` column uses keys from these maps. New extended resource names need no kubectl registration or code changes.

Keep DeviceClass request keys such as `deviceclass.resource.kubernetes.io/example`. A general check that drops all names under `kubernetes.io` would lose these requests.

Read resource names from the Pod maps, not only from Node allocatable. Some scheduler extensions use resource dimensions that the Node does not publish. Such requests remain useful to show, but do not prove that a device or that amount of capacity was assigned.

GPU, NPU, MIG, HAMi, and RDMA requests use the same path when expressed as extended resources. There is no vendor list or plugin detection. DRA-backed extended resources stay in the first table; do not look up the implicit Claim created for them by [KEP-5004].

### Explicit ResourceClaims

Match the regular and init containers' `resources.claims` entries to the Pod's `spec.resourceClaims`. A named request selects that request; an empty request name selects all requests in the Claim. Merge references from the same Pod so repeated container references do not produce duplicate rows. Show the same Claim separately for each Pod that references it.

Use a direct Claim name or the generated name in `status.resourceClaimStatuses`, following the existing DRA name resolution and ownership checks. Check generated Claims against the Pod or its matching PodGroup using the Pod fields; no PodGroup read is needed. Skip Claims that DRA intentionally did not create. Do not guess a generated name from a template. Read only the named Claims in the Pod namespaces, with the caller's credentials. Use the `resource.k8s.io/v1` API. Cache each lookup result, including errors, by namespace and name for the node description. No cluster-wide Claim list is needed.

For a readable Claim, show the DeviceClass and requested count for `ExactCount`, or `All` for that allocation mode. For `firstAvailable`, match the allocation result's request and subrequest names to the referenced parent request. Show the selected subrequest's class and count only when this match identifies it. Until then show `Alternatives` and do not guess a class or count. Report the Claim as `allocated` or `unallocated` based on `status.allocation`. This is the Claim's state, not the Pod phase or reservation status.

If the Claim name cannot yet be resolved, keep the Pod's Claim reference with `unresolved`. Keep the reference with `not found` for an object-level NotFound, `forbidden` for a denied read, or `unavailable` for an unsupported API or other read error. If ownership cannot be verified, report details as `unavailable`. Leave unknown class and count cells blank. Keep successful rows and the existing node output even when Claim details are incomplete. If the user cannot list Pods, keep the current access-denied behavior and skip both tables.

### Test Plan

- [x] The authors understand that component owners may require more tests.

##### Prerequisite testing updates

Extend the existing node describer fixtures. No GPU hardware is required.

##### Unit tests

Cover no extended resources; multiple vendors and resource names; multiple containers; ordinary and restartable init containers; zero and missing values; resource keys absent from Node allocatable; full names, quantities, and ordering. Include one Pod that requests both GPU and shared RDMA resources, a `deviceclass.resource.kubernetes.io/example` request, and many keys across many Pods. A Claim-only Pod produces Claim rows without a scalar row; a mixed Pod produces both. Cover named requests, template names and ownership checks, duplicate container references, a shared Claim, `All`, and selected and unresolved `firstAvailable` alternatives.

At the `NodeDescriber.Describe` level, use the real tabwriter with long Pod and resource names and check that existing tables and totals stay unchanged. Use fake-client reactors for multiple Pod pages, Forbidden, and other API errors. Verify that all listed non-terminated Pods are considered, terminated Pods are excluded, non-terminal Pods being deleted and API mirror Pods are included, row sums match the totals for shared keys, denied access still prints `Pods: not authorized`, and Claim reads are skipped when no listed Pod references an explicit Claim. Check one lookup per unique Claim, missing or forbidden reads, unresolved template names, and an unavailable Claim API. Claim read errors must leave the original tables and successful rows intact.

##### Integration tests

No new server integration test is proposed. The command uses the existing Pod and ResourceClaim APIs.

##### e2e tests

Create a synthetic Node and a Pending Pod with `spec.nodeName` set to that Node. The Pod does not need to run. Check the Pod/resource row and its absence without such requests. Where the Claim API is served, create a Claim and check its row for a bound Pod, and its absence for unbound and terminal Pods. Clean up the fixtures; no real node, GPU, or device plugin is needed.

### Graduation Criteria

The client-only release plan needs SIG CLI agreement. No release is targeted. The two tables may be implemented in separate PRs under this KEP.

#### Alpha

Agree on the default table and pass the proposed tests.

#### Beta

Review output compatibility and large-node output before wider release.

#### GA

Confirm test coverage and publish the command documentation.

#### Deprecation

None planned.

### Upgrade / Downgrade Strategy

Use the new kubectl to get the tables. Returning to an older supported kubectl removes them. Cluster state and workloads do not change.

### Version Skew Strategy

Follow the supported kubectl/server version skew. The extended resource table uses existing Pod fields. Claim details require the ResourceClaim v1 API and read access. When details are unavailable, keep the reference and the extended resource output.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

The new tables appear by default when there are matching rows. There is no feature gate or stored state. Rolling back the client removes them.

### Rollout, Upgrade and Rollback Planning

No server restart or workload change is needed. Check old and new client output, script compatibility, and client time before release. Rollback testing has not been done. No API, field, or flag is removed.

### Monitoring Requirements

Compare extended resource rows with the Pod's effective requests and limits, and Claim rows with the referenced Claim. Check command success, output, and client time with large fixtures. No new metric is proposed.

### Dependencies

The existing node and Pod reads are unchanged. Claim details require `get` permission on the referenced ResourceClaims. No monitoring agent or device driver is queried.

### Scalability

Claim lookups add at most one GET per unique referenced Claim per node description. No Claim reads occur on the scalar-only path. Keep lookup concurrency bounded and test many distinct Claims as well as shared Claims. No new API types, stored objects, or node-side work are added.

### Troubleshooting

Keep current node and Pod error handling. Check the Claim name and read permission if details are unavailable. Long names can still wrap in a narrow terminal.

## Implementation History

## Drawbacks

The output is longer and repeats Pod names. Claim reads add latency and may return incomplete details. The tables do not show actual device use or free capacity.

## Alternatives

- Add columns to the current table: long resource names and many resource types make it too wide, as discussed in PR #46079.
- Put all resources in one extra cell: HAMi and mixed-device requests make the cell long and hard to read.
- Put both request types in one table: Claim requests have no limits, and their allocation state does not apply to scalar requests.
- Show only scalar requests: avoids Claim reads, but leaves Claim-only Pods out of the device request view.
- Require a flag or plugin: keeps the default short, but users must know the extra command to find basic request details.

Separate tables keep a fixed column count and preserve the meaning of each field.

## Infrastructure Needed (Optional)

None.

[PR #46079]: https://github.com/kubernetes/kubernetes/pull/46079
[KEP-5677]: /keps/sig-node/5677-dra-resource-availability-visibility/README.md
[KEP-5004]: /keps/sig-scheduling/5004-dra-extended-resource/README.md
