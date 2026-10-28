# KEP-6472: Show Pod extended resource requests in kubectl describe node

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

Show which Pods request extended resources on a node, and how much each Pod requests, in `kubectl describe node`.

Add a separate `Extended Resources by Pod` table with namespace, Pod name, resource name, requests, and limits. Keep the existing tables unchanged. Use the Pod data that the command already reads. No API change is needed.

## Motivation

A node can report `nvidia.com/gpu: 8` under `Allocated resources`, but the `Non-terminated Pods` table only shows CPU and memory. Users cannot tell which Pods requested those eight resource units from this output.

[PR #46079] added the node totals. It left out Pod details because long resource names made the Pod table hard to format. A separate table can show these details without adding columns to the existing table.

### Goals

- Show extended resource requests and limits for each listed Pod.
- Support all extended resource names without vendor-specific rules.
- Keep the current API calls, access rules, and resource calculations.

### Non-Goals

Report physical device IDs, runtime use, device health, or free capacity. Read Dynamic Resource Allocation (DRA) ResourceClaims, device-plugin sockets, or vendor annotations. Change scheduling, resource accounting, or `kubectl top`.

## Proposal

After `Allocated resources`, print the new table if at least one Pod has a nonzero extended resource request or limit. For example:

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

These are sample values, not a report from one cluster. Each row describes one Pod and one resource. The number of columns stays the same as resource types increase. Nodes without such requests or limits get no new table.

The last Pod requests both GPU and shared RDMA resources. It has two rows with the same namespace and Pod name, one for each resource.

If any listed Pod has entries in `spec.resourceClaims`, also print this note, even when the scalar table has no rows:

```text
ResourceClaim requests are not expanded here. Inspect the Pods and their ResourceClaims for details.
```

This check uses the Pod data already read. A mixed Pod has its scalar requests shown in the table and its Claim requests covered by the note. The output is not a full list of device allocations.

### Relationship to KEP-5677

[KEP-5677] adds `ResourcePoolStatusRequest` to report DRA pool availability. A controller reads ResourceSlices and ResourceClaims and returns a snapshot. Its result is a pool summary, rather than a list of Pod requests. Users read it through `kubectl get resourcepoolstatusrequest`.

| | This KEP | KEP-5677 |
| --- | --- | --- |
| Question | Which Pods request how much? | How much DRA pool capacity is available? |
| Row | Pod and extended resource | Driver and pool, with capacity summaries |
| Input | Existing Pod requests and limits | ResourceSlices and ResourceClaims |
| Change | kubectl output | API and controller |
| Resource path | Extended resource requests | DRA pools |

This KEP does not depend on KEP-5677 or calculate DRA availability. It shows scalar keys already in the Pod's resource maps, including keys served by DRA. It does not turn Claim allocations into rows or counts. Existing node-local ResourceSlice summaries stay unchanged.

### Risks and Mitigations

The output grows by one row per nonzero Pod/resource pair. A request for eight units of one resource adds one row; a Pod with several resource keys adds several rows. GPU memory/core limits and separate network resource pools can add keys. Show all matching rows without silent truncation. Test many keys per Pod and many Pods per node. Existing rows and columns keep their format. Scripts that parse the whole text may need updates; scripts should use structured API output.

A resource unit may mean a device, a partition, or a share. Keep the full resource name and quantity. Do not label the value as physical GPU count, convert vendor units, or add percentages.

## Design Details

Change `describeNodeResource` in `staging/src/k8s.io/kubectl/pkg/describe/describe.go`:

1. Use the same non-terminated Pod list as the current table. This includes Pods bound to the node, even if they are not running yet.
2. Reuse each Pod's `PodRequests` and `PodLimits` results and options. This keeps init container, sidecar, overhead, and status handling consistent.
3. From the union of those maps, select names using the same rule as the current `Allocated resources` table: `!kubectlresourcehelper.IsStandardContainerResourceName(string(name)) && name != corev1.ResourcePods`. This excludes CPU, memory, huge pages, ephemeral storage, and `pods`.
4. Keep a pair if either quantity is nonzero. A missing quantity prints as `0`, as in the existing tables. Format quantities with `Quantity.String()`.
5. Keep the same Pod order as the current table. Sort resource names within each Pod. Print full namespace, Pod, and resource names on every row.

Exclude `Succeeded` and `Failed` Pods, as today. Include other phases and Pods being deleted until they reach a terminal phase. Static Pods are visible only through their API mirror Pods. Ephemeral containers have no separate resource requests or limits; use the existing helpers without adding special accounting.

The new rows and `Allocated resources` totals use the same Pod list and helper options. For each resource shown in both tables, the row sums must match the totals. The Node and Pods are read separately, so this is not an atomic snapshot of node capacity and Pod requests. Keys absent from Node allocatable can appear only in the new table.

Collect rows during the existing Pod loop and print them after the current totals. Do not calculate each Pod's resources again for the new table. Use the existing `PrefixWriter` and tabwriter; do not set a fixed line width or truncate names.

The `Resource` column uses keys from these maps. New extended resource names need no kubectl registration or code changes.

Keep DeviceClass request keys such as `deviceclass.resource.kubernetes.io/example`. A general check that drops all names under `kubernetes.io` would lose these requests.

Read resource names from the Pod maps, not only from Node allocatable. Some scheduler extensions use resource dimensions that the Node does not publish. Such requests remain useful to show, but do not prove that a device or that amount of capacity was assigned.

GPU, NPU, MIG, HAMi, and RDMA requests use the same path when expressed as extended resources. There is no vendor list or plugin detection. DRA-only Pods do not receive a row with a made-up zero device count. Check `spec.resourceClaims` in the same loop to decide whether the Claim note is needed; no Claim API lookup is made.

If the user cannot list Pods, keep the current access-denied behavior and do not print a table that suggests no resources were requested. Add no API calls, API fields, flags, feature gates, or new access requirements.

### Test Plan

- [x] The authors understand that component owners may require more tests.

##### Prerequisite testing updates

Extend the existing node describer fixtures. No GPU hardware is required.

##### Unit tests

Cover no extended resources; multiple vendors and resource names; multiple containers; ordinary and restartable init containers; zero and missing values; resource keys absent from Node allocatable; full names, quantities, and ordering. Include one Pod that requests both GPU and shared RDMA resources, a `deviceclass.resource.kubernetes.io/example` request, and many keys across many Pods. A Claim-only Pod produces the note without a scalar row; a mixed Pod produces both; a Pod with no Claim reference adds no note.

At the `NodeDescriber.Describe` level, use the real tabwriter with long Pod and resource names and check that existing tables and totals stay unchanged. Use fake-client reactors for multiple Pod pages, Forbidden, and other API errors. Verify that all listed non-terminated Pods are considered, terminated Pods are excluded, non-terminal Pods being deleted and API mirror Pods are included, row sums match the totals for shared keys, denied access still prints `Pods: not authorized`, and no extra API requests are made.

##### Integration tests

No new server integration test is proposed. This change adds no control-plane behavior or API calls.

##### e2e tests

Create a synthetic Node and a Pending Pod with `spec.nodeName` set to that Node. The Pod does not need to run. Check the Pod/resource row and its absence without such requests. Clean up both objects; no real node, GPU, or device plugin is needed.

### Graduation Criteria

The client-only release plan needs SIG CLI agreement. No release is targeted.

#### Alpha

Agree on the default table and pass the proposed tests.

#### Beta

Review output compatibility and large-node output before wider release.

#### GA

Confirm test coverage and publish the command documentation.

#### Deprecation

None planned.

### Upgrade / Downgrade Strategy

Use the new kubectl to get the table. Returning to an older supported kubectl removes the table. Cluster state and workloads do not change.

### Version Skew Strategy

Follow the supported kubectl/server version skew. The table uses existing Pod fields and does not require a new server, DRA API, or feature gate.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

The new table appears by default with the new kubectl. There is no feature gate or stored state. Rolling back the client removes it; using the new client again restores it. Tests cover table presence and absence.

### Rollout, Upgrade and Rollback Planning

No server restart or workload change is needed. Check old and new client output, script compatibility, and client time before release. Rollback testing has not been done. No API, field, or flag is removed.

### Monitoring Requirements

Compare the rows with the Pod's effective requests and limits. Check command success, correct output, and client time with large fixtures. No new service SLO or metric is proposed.

### Dependencies

Only the API access already needed by `describe node`. No monitoring agent or device driver is queried.

### Scalability

No new API calls, API types, cloud calls, or stored objects. Only client work and output grow with the number of nonzero Pod/resource pairs. Reuse existing calculations and check the cost of large fixtures. No node-side work is added.

### Troubleshooting

Keep current API and permission error handling. Long names can still wrap in a narrow terminal. Claim-only Pods have no scalar request rows; the Claim note explains this limit. If output or client time is wrong, compare old and new clients using the same Pod data and permissions.

## Implementation History

## Drawbacks

The output is longer and repeats Pod names for Pods with several resources. It shows requests and limits, not actual device use or available capacity.

## Alternatives

- Add columns to the current table: long resource names and many resource types make it too wide, as discussed in PR #46079.
- Put all resources in one extra cell: HAMi and mixed-device requests make the cell long and hard to read.
- Require a flag or plugin: keeps the default short, but users must know the extra command to find basic request details.

The separate table keeps a fixed column count and uses existing data.

## Infrastructure Needed (Optional)

None.

[PR #46079]: https://github.com/kubernetes/kubernetes/pull/46079
[KEP-5677]: /keps/sig-node/5677-dra-resource-availability-visibility/README.md
