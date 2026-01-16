<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md

To get started with this template:

- [X] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [X] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [X] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [X] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [X] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [X] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->

# KEP-5526: Pod Level Resource Managers

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Glossary](#glossary)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: AI/ML Workload with Data-Ingestion Sidecar](#story-1-aiml-workload-with-data-ingestion-sidecar)
    - [Story 2: Workload with a Device-Specific Infrastructure Container](#story-2-workload-with-a-device-specific-infrastructure-container)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Topology Manager](#topology-manager)
    - [Policies](#policies)
    - [Policy Options](#policy-options)
    - [Scopes](#scopes)
      - [Pod Scope](#pod-scope)
      - [Container Scope](#container-scope)
  - [Pod Scope Allocation and Partitioning Algorithm for CPU and Memory managers](#pod-scope-allocation-and-partitioning-algorithm-for-cpu-and-memory-managers)
  - [CPU Manager](#cpu-manager)
    - [Policies](#policies-1)
    - [Policy Options](#policy-options-1)
    - [Pod Scope Allocation and Partitioning Algorithm](#pod-scope-allocation-and-partitioning-algorithm)
    - [State Management and Container Removal](#state-management-and-container-removal)
    - [Interaction with CPU Quota Management](#interaction-with-cpu-quota-management)
  - [Memory Manager](#memory-manager)
    - [Policies](#policies-2)
    - [Pod Scope Allocation and Partitioning Algorithm](#pod-scope-allocation-and-partitioning-algorithm-1)
    - [State Management and Container Removal](#state-management-and-container-removal-1)
  - [Future Enhancements and Long-Term Vision](#future-enhancements-and-long-term-vision)
  - [Feature Gate](#feature-gate)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
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

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone /
release*.

-   [ ](R) Enhancement issue in release milestone, which links to KEP dir in
    [kubernetes/enhancements](not the initial KEP PR)
-   [ ](R) KEP approvers have approved the KEP status as `implementable`
-   [ ](R) Design details are appropriately documented
-   [ ](R) Test plan is in place, giving consideration to SIG Architecture and
    SIG Testing input (including test refactors)
    -   [ ] e2e Tests for all Beta API Operations (endpoints)
    -   [ ](R) Ensure GA e2e tests meet requirements for
        [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
    -   [ ](R) Minimum Two Week Window for GA e2e tests to prove flake free
-   [ ](R) Graduation criteria is in place
    -   [ ](R)
        [all GA Endpoints](https://github.com/kubernetes/community/pull/1806)
        must be hit by
        [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
        within one minor version of promotion to GA
-   [ ](R) Production readiness review completed
-   [ ](R) Production readiness review approved
-   [ ] "Implementation History" section is up-to-date for milestone
-   [ ] User-facing documentation has been created in [kubernetes/website], for
    publication to [kubernetes.io]
-   [ ] Supporting documentation—e.g., additional design documents, links to
    mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Glossary

-   **Pod Level Resources**: The resource budget defined at the pod level in
    `pod.spec.resources`, which specifies the collective requests and limits for
    the entire pod.
-   **Guaranteed Container**: Within the context of this KEP, a container is
    considered `Guaranteed` if it specifies resource requests equal to its
    limits for both CPU and Memory. This status makes it eligible for exclusive
    resource allocation from the resource managers, provided the topology scope
    allows it.
-   **Pod Shared Pool**: The subset of a pod's allocated resources that remains
    after all exclusive slices have been reserved. These resources are shared by
    all containers in the pod that do not receive an exclusive allocation.
-   **Exclusive Slice**: A dedicated portion of resources (e.g., specific CPUs
    or memory pages) allocated solely to a single container, ensuring isolation
    from other containers.

## Summary

This KEP proposes extending the Kubelet's Topology, CPU, and Memory Managers to
support pod-level resource specifications. This enhancement evolves the resource
managers from a strictly per-container allocation model to a pod-centric one,
that enables them to use `pod.spec.resources` to perform NUMA alignment for the
pod as a whole, and introduces a partitioning scheme to manage resources for
containers within that pod-level grouping. This change introduces a more
flexible and powerful resource management model, particularly for
performance-sensitive workloads.

## Motivation

The introduction of Pod-Level Resources
([KEP-2837](https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/2837-pod-level-resource-spec/README.md))
allows users to define a resource budget for an entire pod. However, the
Kubelet's key resource management components—Topology, CPU, and Memory
Managers—are not yet able to use this pod-level specification as the direct
basis for NUMA alignment decisions. While the `pod` scope can align a pod based
on the aggregate of its container requests, it cannot leverage the pod-level
budget directly. This limits performance and flexibility.

For high-performance computing (HPC), AI/ML, and network function virtualization
workloads, minimizing memory and CPU latency is critical. This is best achieved
by ensuring all of a pod's resources are allocated from the same NUMA node.
Furthermore, many of these workloads consist of a primary, performance-sensitive
container alongside less critical sidecars. The current per-container model
lacks the flexibility to manage such mixed-criticality pods as a single,
NUMA-aligned unit with internal resource divisions.

This KEP addresses these gaps by enabling the resource managers to use
`pod.spec.resources` to NUMA-align the entire pod. It also introduces a
partitioning model that allows the pod's resources to be divided between
containers that require exclusive access and those that can share resources, all
within a single NUMA alignment.

### Goals

1.  **Pod-level Aware Hint Generation:** The CPU and Memory managers, acting as
    hint providers for the Topology Manager, will be updated to recognize and
    use `pod.spec.resources`. When the scope is `pod`, they will generate a
    single hint for the entire pod's resource budget. When the scope is
    `container`, they will continue to generate hints on a per-container basis.
2.  **Implement Hybrid Allocation Logic:** The CPU and Memory managers will
    implement a hybrid allocation strategy that is determined by the Topology
    Manager's scope:
    *   **Pod Scope:** The Topology Manager will perform a NUMA alignment for
        the entire pod based on `pod.spec.resources`, respecting the
        `topology-manager-policy`, which may result in a single or multi-NUMA
        node allocation. The resulting pod-wide allocation is then partitioned
        by the CPU and Memory managers, exclusive slices are carved out for
        `Guaranteed` containers, while the remainder forms a shared pool for all
        other containers. This enables a single pod to host a mix of containers
        with and without dedicated resource guarantees.
    *   **Container Scope:** The existing per-container allocation logic will be
        preserved. This logic will be enhanced to support pods that achieve a
        `Guaranteed` QoS class via `pod.spec.resources`, enabling a powerful
        mixed-model: individual containers that are `Guaranteed` can receive
        exclusive, NUMA-aligned resources, while other containers in the same
        pod can run in the node's shared pool, with their overall resource
        consumption enforced by the pod's `pod.spec.resources` budget.

### Non-Goals

1.  **Changing the Kubernetes QoS Model:** This KEP leverages the existing QoS
    calculation, which correctly prioritizes `pod.spec.resources` when the
    `PodLevelResources` feature is active. We are not proposing any changes to
    this fundamental model.
2.  **Introducing New User-Facing APIs or Policy Options:** This enhancement
    extends the behavior of the existing `static` (CPU Manager) and `Static`
    (Memory Manager) policies, along with the `pod` and `container` scopes. It
    does not introduce new policies (e.g., a "shared" policy) or policy options.
3.  **Supporting Non-Guaranteed Pods for Exclusive Allocation:** The mechanisms
    for exclusive, NUMA-aligned resources described in this KEP apply only to
    pods that have a `Guaranteed` QoS class.
4.  **Supporting In-Place Pod Vertical Scaling:** This KEP is focused on the
    initial allocation of resources at pod startup. The interaction between
    pod-level resource management and the In-Place Pod Vertical Scaling feature
    is out of scope.

## Proposal

This proposal updates the Topology, CPU, and Memory managers to recognize and
act upon `pod.spec.resources`, enabling two flexible resource management models.
Both models support `Guaranteed` pods that contain a mix of `Guaranteed` and
non- `Guaranteed` containers:

1.  **Pod Scope:** The Topology Manager will secure a single NUMA-aligned
    resource pool for the entire pod based on `pod.spec.resources`. The CPU and
    Memory managers will then partition this pool, carving out exclusive slices
    for containers with specific `Guaranteed` requests and creating a shared
    pool for all other containers from the remainder.
2.  **Container Scope:** This scope will continue to manage resources on a
    per-container basis, ignoring `pod.spec.resources` for allocation decisions.
    With the new feature gate, it will correctly handle `Guaranteed` pods that
    use pod-level resources, allowing containers with their own `Guaranteed`
    requests to receive exclusive NUMA-aligned resources while others in the
    same pod use the node's shared pool.

This approach provides significant flexibility. The `pod` scope is ideal for
workloads that benefit from having all containers co-located on the same NUMA
node, such as a primary application and its sidecars. It allows users to
co-locate containers that require exclusive, `Guaranteed` resources with
containers that do not, all within a single, NUMA-aligned pod.

The `container` scope, in turn, offers a key benefit over the `pod` scope: it
provides the flexibility to manage containers with different resource
requirements independently within a single `Guaranteed` pod. This enables a
powerful mixed-model where a `Guaranteed` container can receive an exclusive,
NUMA-aligned allocation, while a `non-Guaranteed` container in the same pod can
run in the node's shared pool. This was not possible before, as all containers
in a pod were required to be `Guaranteed` to enable exclusive allocations for
any of them.

### User Stories

#### Story 1: AI/ML Workload with Data-Ingestion Sidecar

As a machine learning engineer, I am deploying a pod that contains multiple
containers, a primary training container that requires significant, guaranteed
CPU resources for performance, and several sidecars for tasks like
data-ingestion, logging, and monitoring, all of which have minimal, burstable
resource needs. I want the entire pod to be NUMA-aligned to ensure the training
container has fast access to memory, but I don't want to overprovision resources
for the less critical sidecars.

By setting `pod.spec.resources` to make the pod `Guaranteed` and specifying
`static` CPU policy with `pod` scope, I can define a large resource budget for
the pod. I then set specific, `Guaranteed` requests on my training container.
The system will allocate a NUMA-aligned pool for the pod, carve out an exclusive
slice for my training container, and allow the data-ingestion, logging, and
monitoring sidecars to efficiently share the rest of the resources within the
pod's exclusive shared pool.

#### Story 2: Workload with a Device-Specific Infrastructure Container

As a platform engineer, I am deploying a pod with a main workload and an
infrastructure sidecar that needs access to a specific hardware device (e.g., a
high-performance NIC), and a few other utility containers. For optimal
performance, the sidecar requires exclusive, `Guaranteed` resources for NUMA
alignment. My main workload and the utility containers also have resource
requirements, but they are different from the sidecar's, and I need the
flexibility to manage them independently within the same pod.

This KEP enables a new mixed-resource model within the `container` scope that
addresses this. By setting `pod.spec.resources` to make the entire pod
`Guaranteed`, I can now provide `Guaranteed` requests for my infrastructure
sidecar, ensuring it receives its own independent, NUMA-aligned slice of
resources. This allows the sidecar to be aligned to a specific NUMA node (a
critical step for device co-location), while the main workload, and the other
utility containers can co-exist in the same pod and utilize other available
resources.

While this KEP does not make the Topology Manager aware of device-to-NUMA
locality, this ability to grant per-container alignment within a mixed-resource
pod is a foundational enabler, as discussed in the
[Future Enhancements and Long-Term Vision](#future-enhancements-and-long-term-vision)
section.

### Notes/Constraints/Caveats

-   This feature is dependent on the `PodLevelResources` feature gate being
    enabled.
-   If the `PodLevelResourceManagers` feature gate is disabled, pods utilizing
    pod-level resources remain eligible for admission; however, they will not
    receive NUMA-aligned or exclusive resource allocations from the CPU and
    Memory managers, even if the pod's QoS class is `Guaranteed`.
-   The functionality is only implemented for the `static` CPU Manager policy
    and the `Static` Memory Manager policy.
    -   Other policies like `none` are unaffected as they do not perform
        NUMA-aware allocations.
    -   The `BestEffort` memory policy is also unaffected, as it's a
        Windows-only feature and pod-level resources are not supported on
        Windows.
-   For the `pod` scope, the sum of all container-level resource requests must
    not exceed the pod-level resource budget defined in `pod.spec.resources`.
    This is enforced by Kubelet admission control.
-   When the `container` scope is active, the resource managers will ignore
    `pod.spec.resources` for their allocation decisions, focusing only on
    per-container requests. However, `pod.spec.resources` remains critical: it
    determines the pod's QoS class, and its `limits` are enforced on a pod-wide
    cgroup. This means that while `non-Guaranteed` containers run in the node's
    shared pool, their collective resource usage is still constrained by the
    pod-level limit at runtime.
-   Interaction with In-Place Pod Vertical Scaling is not part of this KEP. The
    focus is on the initial allocation of resources.

### Risks and Mitigations

-   **Risk:** Incorrect resource accounting in the new partitioning model could
    lead to the over- or under-allocation of resources, potentially causing
    unexpected container throttling or resource contention within the pod's
    shared pool.
    -   **Mitigation:** The implementation will be accompanied by extensive unit
        and e2e tests covering all policy and scope combinations to validate the
        new partitioning and allocation logic. These tests will include
        scenarios with mixed-criticality containers, init containers, and
        sidecars to ensure the resource accounting is correct in all cases.
-   **Risk:** Changes to core Kubelet components could introduce performance
    regressions or instability.
    -   **Mitigation:** All new functionality will be protected by the
        `PodLevelResourceManagers` feature gate, which will be disabled by
        default in its initial alpha release. This allows for safe testing and a
        clear rollback path (disabling the gate and restarting the Kubelet).
-   **Risk:** User confusion between `pod` and `container` scopes could lead to
    suboptimal performance. A user might use the `pod` scope for a workload that
    has containers requiring placement on different NUMA nodes (e.g., for device
    access), inadvertently forcing them onto the same NUMA node.
    -   **Mitigation:** Clear and detailed user-facing documentation is
        essential. The documentation must include explicit examples for both
        scopes, guiding users to select the `pod` scope for co-location and the
        `container` scope for independent, per-container NUMA alignment.
        Additionally, the system will provide **informative log messages** when
        resource managers make decisions that might be unexpected (e.g., a
        container in a `Guaranteed` pod not receiving exclusive allocation due
        to non-integral requests). **New metrics** will also be exposed to help
        operators identify patterns of suboptimal resource utilization or
        misconfiguration across the cluster.

## Design Details

### Topology Manager

The Topology Manager's core logic for merging hints remains unchanged. However,
its overall behavior will be updated through modifications to its `pod` and
`container` scopes. These scopes will be enhanced to interact with hint
providers (CPU and Memory Managers) that are now aware of `pod.spec.resources`,
enabling more flexible and accurate NUMA alignment for pods utilizing this
feature.

#### Policies

The behavior of existing Topology Manager policies and policy options will be
supported for pod-level resources.

-   **`none`**: This policy makes the Topology Manager idle. No changes are
    needed.
-   **`best-effort`**: This policy will admit pods regardless of whether the
    NUMA alignment is preferred or not. It will be supported for pods with
    pod-level resources.
-   **`restricted`**: This policy will only admit pods if a preferred NUMA
    alignment can be achieved. It will be supported for pods with pod-level
    resources.
-   **`single-numa-node`**: This policy will only admit pods if all of their
    resources can be allocated from a single NUMA node. It will be supported for
    pods with pod-level resources.

#### Policy Options

-   **`prefer-closest-numa-nodes`**: This option makes the Topology Manager
    aware of NUMA distances when making alignment decisions. It will be
    supported for pods with pod-level resources.
-   **`max-allowable-numa-nodes`**: This policy allows the Topology manager to
    work on nodes with more than eight NUMA nodes, and it will be supported by
    pod level resources.

#### Scopes

##### Pod Scope

When the `topology-manager-scope` is set to `pod`, the Topology Manager will
invoke the `GetPodTopologyHints` function on its hint providers. This triggers
the providers to generate a single set of hints based on the total resource
budget defined in `pod.spec.resources`. This process ensures the entire pod,
including containers with and without individual `Guaranteed` resource requests,
can be aligned to a single NUMA node or a set of NUMA nodes.

Pod-level resources | Container resources | CPU and Memory manager behavior
:------------------ | :------------------ | :------------------------------
Unset               | Set                 | Current behavior (alignment and exclusive allocation based on container resources)
Set                 | Unset               | Alignment and exclusive allocation for the overall pod. All containers share the resulting resource pool.
Set                 | Set                 | Alignment and exclusive allocation for the overall pod. The pod's resource pool is partitioned into exclusive slices for `Guaranteed` containers and a shared pool for the rest.

When `pod.spec.resources` are defined, they are the sole determinant of the
pod's QoS class. However, the resource managers will still check each
container's eligibility for exclusive resource allocation. To qualify, a
container must have `requests` equal to its `limits` for CPU and memory.
Additionally, for CPU resources, the requested quantity must be a positive
integer. To aid debugging, the results of this check will be surfaced via log
message.

New cases that need to be covered:

-   **Pod-level resources set, Container-level resources unset:**
    -   The alignment decision is made for the pod as a single unit, based on
        `pod.spec.resources`. The resulting NUMA-aligned resource pool is then
        managed for the pod, and this entire pool is treated as the shared pool
        for all containers within it.
-   **Pod-level resources set, Container-level resources set:**
    -   The alignment decision is made for the pod as a single unit, based on
        `pod.spec.resources`. The resulting NUMA-aligned resource pool is then
        partitioned. Containers that individually meet `Guaranteed` criteria are
        allocated exclusive slices from the pool, while all other containers are
        allocated the remaining shared portion of the pool.

Already existing logic for standard and restartable init containers is
compatible with the changes presented in this KEP. Standard init containers will
be allocated exclusive slices from the pod's managed resource pool. Because they
run sequentially, their resources are made reusable for subsequent app
containers. Restartable init containers (sidecars) that are `Guaranteed` will
also be allocated exclusive slices, but their resources are reserved for the
entire pod lifecycle and are not returned to the pool for reuse. This results in
a progressively decreasing `podSharedPool` as more sidecars with exclusive
resource requirements are defined.

Additionally, if a standard init container does not specify resource requests
and limits (placing it in the shared pool), it is granted access to the entire
pod-level resource budget, minus only the resources exclusively allocated to any
`Guaranteed` sidecars running up to that point. This ensures that init
containers can burst to utilize maximum available resources during
initialization.

In contrast, sidecars that run in the shared pool are assigned the final
`podSharedPool`. This pool consists of the resources remaining after all
exclusive allocations (for both `Guaranteed` sidecars and `Guaranteed`
application containers) have been reserved. This ensures that shared sidecars do
not interfere with the exclusive resources of application containers once they
start.

Pod Scope                                                             | Spec                                                                               | Hint Generation | Allocation
:-------------------------------------------------------------------- | :--------------------------------------------------------------------------------- | :-------------- | :---------
Current behavior                                                      | Container 1: 3 CPU <br> Container 2: 1 CPU <br> Container 3: 1 CPU                 | {NUMA node 0}   | Container 1: {1, 2, 3} <br> Container 2: {4} <br> Container 3: {5}
New behavior <br> Pod Level Resources <br> All Guaranteed containers  | Pod: 5 CPU <br> Container 1: 3 CPU <br> Container 2: 1 CPU <br> Container 3: 1 CPU | {NUMA node 0}   | Container 1: {1, 2, 3} <br> Container 2: {4} <br> Container 3: {5}
New behavior <br> Pod Level Resources <br> Some Guaranteed containers | Pod: 5 CPU <br> Container 1: 3 CPU <br> Container 2 <br> Container 3               | {NUMA node 0}   | Container 1: {1, 2, 3} <br> Container 2: {4, 5} <br> Container 3: {4, 5}
New behavior <br> Pod Level Resources <br> No Guaranteed containers   | Pod: 5 CPU <br> Container 1 <br> Container 2 <br> Container 3                      | {NUMA node 0}   | Container 1: {1,2,3,4,5} <br> Container 2: {1,2,3,4,5} <br> Container 3: {1,2,3,4,5}
New behavior <br> Pod Level Resources <br> Admission Failure          | Pod: 5 CPU <br> Container 1: 3 CPU <br> Container 2: 2 CPU <br> Container 3        | Admission Error | Pod Rejected

###### Rejected Configurations

**Empty podSharedPool with non-Guaranteed containers**

A critical aspect of the `pod` scope is ensuring that every container within the
pod can be allocated valid resources from the pod's NUMA-aligned resource pool.
To enforce this, the Kubelet performs an admission check to prevent
configurations that would lead to an empty pod shared pool for one or more
containers. A pod will be **rejected** at admission time if it meets the
following criteria:

1.  The `topology-manager-scope` is set to `pod`.
2.  The sum of resource requests from all containers that are `Guaranteed`
    exactly equals the total resource budget defined in `pod.spec.resources`.
3.  There is at least one other container in the pod that does not qualify for
    exclusive allocation and therefore requires placement in the pod's shared
    pool.

This validation is necessary to prevent a silent but critical failure of
resource isolation. If such a pod were admitted, the calculated `podSharedPool`
would be empty. Assigning an empty `cpuset` to a container, makes it run on the
node's shared pool. This behavior would break the pod's NUMA affinity and
violate the exclusivity guarantees of its sibling containers. The admission
check prevents this scenario by failing with a clear error. An example
configuration can be found in:
[Example 3](#example-3-admission-failure-empty-shared-pool).

###### Special configurations

**Underutilization of Pod-Level Resources**

It should be noted that a pod will **not** be rejected if the sum of all
exclusive container-level requests is less than the total pod-level resource
budget.

In such a scenario, the resource managers will still align the full amount of
resources specified in `pod.spec.resources` for the pod. The corresponding
resources requested by the individual containers will be allocated as exclusive
slices. Any remaining resources within the pod's allocation will be reserved for
the pod but will remain unused by any container.

This behavior is intentional. The pod-level specification is treated as the
source of truth for the total resource reservation. The admission control logic
is narrowly focused on preventing functional failures, such as the empty shared
pool scenario, not on enforcing that all reserved resources are fully utilized
by the containers within the pod.

###### Examples

The examples focus on CPU resource behavior, however the memory behavior is
analogous.

**Example 1: Shared Pool Only**

This example shows a `Guaranteed` pod with a pod-level resource budget. Neither
container requests individual resources, so all three will run in the pod's
shared resource pool.

<!-- mdformat off() -->

**Pod Spec:**
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: pod-scope-shared
spec:
  resources:
    requests:
      cpu: "4"
      memory: "4Gi"
    limits:
      cpu: "4"
      memory: "4Gi"
  containers:
  - name: container-1
    image: example-image
  - name: container-2
    image: example-image
  - name: container-3
    image: example-image
```

<!-- mdformat on -->

**Expected Allocation:**

-   The Topology Manager will secure a NUMA-aligned pool of 4 CPUs and 4Gi of
    memory.
-   Since there are no containers with exclusive requests, the pod's shared pool
    is the entire 4 CPU pool.
-   All `container-1`, `container-2` and `container-3` will be assigned the same
    4 CPU set.

**Example 2: Mixed Allocation (Exclusive and Shared)**

This example shows a `Guaranteed` pod with a 4 CPU budget. `container-1`
requests an exclusive 2 CPU slice, while `container-2` and `container-3` will
use the remainder.

<!-- mdformat off() -->

**Pod Spec:**
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: pod-scope-mixed
spec:
  resources:
    requests:
      cpu: "4"
      memory: "4Gi"
    limits:
      cpu: "4"
      memory: "4Gi"
  containers:
  - name: container-1
    image: example-image
    resources:
      requests:
        cpu: "2"
        memory: "2Gi"
      limits:
        cpu: "2"
        memory: "2Gi"
  - name: container-2
    image: example-image
  - name: container-3
    image: example-image
```

<!-- mdformat on -->

**Expected Allocation:**

-   The Topology Manager will secure a NUMA-aligned pool of 4 CPUs.
-   `container-1` will be allocated an exclusive 2 CPU slice from the pool.
-   The pod's shared pool will be the remaining 2 CPUs, which will be assigned
    to `container-2` and `container-3`.

**Example 3: Admission Failure (Empty Shared Pool)**

This example shows a `Guaranteed` pod where the sum of exclusive container
requests exactly matches the pod-level budget, but another container exists that
requires a shared pool. This configuration is invalid and will be rejected at
admission.

<!-- mdformat off() -->

**Pod Spec:**
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: pod-scope-admission-failure
spec:
  resources:
    requests:
      cpu: "5"
      memory: "5Gi"
    limits:
      cpu: "5"
      memory: "5Gi"
  containers:
  - name: container-1
    image: example-image
    resources:
      requests:
        cpu: "3"
        memory: "3Gi"
      limits:
        cpu: "3"
        memory: "3Gi"
  - name: container-2
    image: example-image
    resources:
      requests:
        cpu: "2"
        memory: "2Gi"
      limits:
        cpu: "2"
        memory: "2Gi"
  - name: container-3
    image: example-image
```

<!-- mdformat on -->

**Expected Allocation:**

-   The pod will be **rejected** at admission time.
-   **Reason:** The sum of exclusive CPU requests (`3 + 2 = 5`) equals the pod's
    total budget (`5`). This leaves an empty shared pool for `container-3`.
    Attempting to assign an empty resource set would cause the container to
    default to the node-wide shared pool, for a detailed explanation of this
    situarion, refer to [Rejected Configurations](#rejected-configurations).

##### Container Scope

When the `topology-manager-scope` is set to `container`, the Topology Manager
will continue to invoke the `GetTopologyHints` function on its hint providers
for each container individually. The providers will generate hints based on each
container's specific resource requests, ignoring `pod.spec.resources` for this
purpose. This allows for independent NUMA alignment for each container.

It is important to note that while the resource managers ignore
`pod.spec.resources` for NUMA alignment and runtime allocation decisions in this
scope, the pod-level budget is still enforced by Kubelet admission control. The
sum of all individual container requests cannot exceed the limits defined in
`pod.spec.resources`. This enables a mix where some containers receive exclusive
resources while others run in the node's shared pool, but are still collectively
bound by the pod's overall resource limit.

Pod-level resources | Container resources | CPU and Memory manager behavior
:------------------ | :------------------ | :------------------------------
Unset               | Set                 | Current behavior (alignment and exclusive allocation based on container resources)
Set                 | Unset               | No alignment or exclusive allocation. Containers run in the node's shared pool.
Set                 | Set                 | Alignment and exclusive allocation for containers that specify `Guaranteed` resources; pod-level resources are ignored for allocation decisions.

As with the `pod` scope, the pod's QoS class is determined solely by
`pod.spec.resources`. The resource managers will then perform a QoS-like check
on each container to determine which are eligible for exclusive resource
allocation. To aid debugging, the results of this check will be surfaced to the
via log message.

New cases that need to be covered:

-   **Pod-level resources set, Container-level resources unset:**
    -   No possible alignment, container requests/limits do not exist.
-   **Pod-level resources set, Container-level resources set:**
    -   The overall pod will not receive any alignment and allocation for the
        requested resources, only the individual containers that properly
        specify and fulfill the `Guaranteed` QoS will be allocated exclusive
        resources.

Already existing logic for standard and restartable init containers is
compatible with the changes presented in this KEP. They will continue be
allocated exclusive resources from the node's allocatable resources. The
resources from standard init containers will be made reusable by the app
containers.

Container Scope                                                       | Spec                                                                               | Hint Generation                                            | Allocation
:-------------------------------------------------------------------- | :--------------------------------------------------------------------------------- | :--------------------------------------------------------- | :---------
Current behavior                                                      | Container 1: 3 CPU <br> Container 2: 1 CPU <br> Container 3: 1 CPU                 | {NUMA node 0}, {NUMA node 0}, {NUMA node 0}                | Container 1: {1, 2, 3} <br> Container 2: {4} <br> Container 3: {5}
New behavior <br> Pod Level Resources <br> All Guaranteed containers  | Pod: 5 CPU <br> Container 1: 3 CPU <br> Container 2: 1 CPU <br> Container 3: 1 CPU | {NUMA node 0}, {NUMA node 0}, {NUMA node 0}                | Container 1: {1, 2, 3} <br> Container 2: {4} <br> Container 3: {5}
New behavior <br> Pod Level Resources <br> Some Guaranteed containers | Pod: 5 CPU <br> Container 1: 3 CPU <br> Container 2 <br> Container 3               | {NUMA node 0}, No NUMA preference, No NUMA preference      | Container 1: {1,2,3} <br> Container 2: no exclusive allocation <br> Container 3: no exclusive allocation
New behavior <br> Pod Level Resources <br> No Guaranteed containers   | Pod: 5 CPU <br> Container 1 <br> Container 2 <br> Container 3                      | No NUMA preference, No NUMA preference, No NUMA preference | Container 1: no exclusive allocation <br> Container 2: no exclusive allocation <br> Container 3: no exclusive allocation

###### Examples

The examples focus on CPU resource behavior, however the memory behavior is
analogous.

**Example 1: Mixed Allocation (Exclusive and Node Shared)**

This example shows a `Guaranteed` pod using pod-level resources. Because the
scope is `container`, the managers will evaluate each container individually.
`container-1` gets an exclusive allocation, while `container-2` and
`container-3` run in the node-wide shared pool.

<!-- mdformat off() -->

**Pod Spec:**
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: container-scope-mixed
spec:
  resources:
    requests:
      cpu: "4"
      memory: "4Gi"
    limits:
      cpu: "4"
      memory: "4Gi"
  containers:
  - name: container-1
    image: example-image
    resources:
      requests:
        cpu: "2"
        memory: "2Gi"
      limits:
        cpu: "2"
        memory: "2Gi"
  - name: container-2
    image: example-image
  - name: container-3
    image: example-image
```

<!-- mdformat on -->

**Expected Allocation:**

-   The Topology Manager will evaluate `container-1` and generate hints for its
    2 CPU request, resulting in an exclusive, NUMA-aligned allocation for it.
-   The Topology Manager will evaluate `container-2` and `container-3`, seeing
    no resource requests, will generate a "no preference" hint.
-   `container-1` will be assigned its own exclusive 2 CPU set.
-   `container-2` and `container-3` will run in the general, node-wide shared
    pool, along with containers from other pods.

**Example 2: Pod-Level Resources Only (No Exclusive Allocation)**

This example shows a `Guaranteed` pod that only specifies pod-level resources.
Because the scope is `container` and no containers have individual resource
requests, all containers will run in the node's shared pool. The pod-level
resources are ignored for allocation and alignment purposes.

<!-- mdformat off() -->

**Pod Spec:**
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: container-scope-pod-only
spec:
  resources:
    requests:
      cpu: "4"
      memory: "4Gi"
    limits:
      cpu: "4"
      memory: "4Gi"
  containers:
  - name: container-1
    image: example-image
  - name: container-2
    image: example-image
  - name: container-3
    image: example-image
```

<!-- mdformat on -->

**Expected Allocation:**

-   The Topology Manager evaluates each container individually. Since neither
    `container-1`, `container-2` nor `container-3` has its own resource
    requests, `no preference` hints are generated for both.
-   The pod-level resources are **not** considered for allocation or NUMA
    alignment by the managers in this scope.
-   All three containers will run in the general, node-wide shared pool. No
    exclusive resources will be allocated for this pod.

### Pod Scope Allocation and Partitioning Algorithm for CPU and Memory managers

When the `topology-manager-scope` is `pod`, the CPU and Memory managers adopt a
resource partitioning model for the entire pod. This model ensures that a pod's
total NUMA-aligned resource budget is effectively divided among its containers
based on their individual requirements.

1.  **[New Logic] One-Time Pod-Level Calculation:** The manager determines the
    total pod resource budget from `pod.spec.resources`. It then retrieves the
    NUMA affinity hint from the Topology Manager. This hint is used to identify
    the best set of NUMA nodes from which to allocate the pod's total resource
    budget, ensuring alignment according to the configured topology policy.

2.  **[New Logic] Partitioning into Exclusive and Shared Pools:** The total
    pod-wide allocation is partitioned into two types of resource pools:

    *   **Exclusive Slices:** For each container that individually qualifies as
        `Guaranteed`, a dedicated, exclusive slice of resources is reserved from
        the pod's total allocation.
    *   **Pod Shared Pool:** The remaining resources after all exclusive slices
        have been reserved form the `podSharedPool`. This pool is shared among
        all other containers in the pod that do not receive an exclusive
        allocation.

    A fundamental concept in this new functionality is the `podSharedPool`. This
    represents the set of resources within the pod's total NUMA-aligned budget
    that are available to be shared by all containers that do not have their own
    exclusive `Guaranteed` resource requests. This mechanism is key to the
    partitioning model, as it ensures that critical containers receive their
    reserved resources while allowing all other containers to efficiently share
    the remainder of the pod's budget.

3.  **[New Logic] Handling Init Containers:** The existing, well-established
    logic for handling init container resources is reused and applied within the
    new pod-level resource pool. The allocation is pre-calculated based on the
    predictable, sequential lifecycle of init containers.

    -   **Standard Init Containers:** If individually `Guaranteed`, they are
        allocated an exclusive slice. These resources are added to a per-pod
        reusable set upon completion, making them available for subsequent app
        containers. If they are `non-Guaranteed` (shared), they are granted
        access to the entire pod-level resource budget minus any resources
        exclusively allocated to `Guaranteed` sidecars running at that time.
        This allows standard init containers to burst during initialization.
    -   **Restartable Init Containers (Sidecars):** If individually
        `Guaranteed`, they receive an exclusive slice reserved for the entire
        pod lifecycle. If they are `non-Guaranteed` (shared), they are assigned
        the final `podSharedPool`. This pool consists of resources remaining
        after all exclusive allocations (for both sidecars and application
        containers) have been carved out. This ensures that shared sidecars
        never compete with the exclusive resources of application containers.

    This model results in a progressively decreasing shared resource set for
    standard init containers as more `Guaranteed` sidecars are started,
    eventually settling into the final `podSharedPool` used by all application
    containers and shared sidecars.

4.  **[New Logic] Pre-computation and State Update:** The manager pre-computes
    and writes the specific assignment for every container to the state file.
    `Guaranteed` containers are assigned their exclusive slices, and all other
    containers are assigned the `podSharedPool`. To ensure state consistency and
    support accurate recovery after restarts, a new top-level property is added
    to the state file to track the aggregate pod-level resource allocation. For
    CPU Manager, this is `PodCPUAssignments` (a map of pod UIDs to `PodEntry`
    containing the `CPUSet`). For Memory Manager, this is `PodMemoryAssignments`
    (a map of pod UIDs to `PodEntry` containing the `MemoryBlocks`). This entry
    serves as the source of truth for the pod's total NUMA-aligned resource
    reservation.

5.  **Container-Level Allocation:** For the current container (and all
    subsequent containers in the pod), the `Allocate` function reads that
    container's specific, pre-computed assignment from the state and applies it
    to the container's cgroup.

6.  **[New Logic] Container and Pod Removal:** When a container is removed, its
    individual assignment is deleted from the state. The pod-level resource
    allocation persists to maintain alignment for any remaining containers.
    Resources exclusively allocated to a container within a pod are not returned
    to the pod's shared pool upon that container's termination; they remain
    reserved for the pod until it is fully decommissioned. Only when the last
    container of the pod is removed are the pod's resources released back to the
    node's general pool and the aggregate pod-level entry is deleted from the
    state.

### CPU Manager

The CPU Manager will be updated to support pod-level resources when its policy
is set to `static`. The core changes will affect how it determines the scope of
resource management and how it allocates CPUs to containers within that scope.

-   **Pod Scope:** When the Topology Manager's scope is `pod`, the CPU Manager
    will manage a single CPU set for the entire pod, based on
    `pod.spec.resources`. This set will then be partitioned, containers with
    `Guaranteed` CPU requests will be allocated exclusive CPUs from this set,
    while the remaining CPUs will form a shared pool to be allocated to all
    other containers in the pod.
-   **Container Scope:** When the Topology Manager's scope is `container`, the
    CPU Manager will continue to allocate exclusive CPUs on a per-container
    basis. If a pod is `Guaranteed` due to `pod.spec.resources`, the CPU manager
    will support a mix of containers, those that are individually `Guaranteed`
    will receive exclusive CPUs, while others will run in the node's shared
    pool. The CPU manager will ignore `pod.spec.resources` for its allocation
    decisions.

#### Policies

The behavior of existing CPU Manager policies and policy options will be
supported for pod-level resources. In the `pod` scope, these options will
operate on the total `pod.spec.resources`. For example, the `full-pcpus-only`
option will validate that the pod's total CPU request is a multiple of the SMT
level.

-   **`none`**: This policy makes the Topology Manager idle. No changes are
    needed.
-   **`static`**: This policy enables the CPU manager functionality, and
    contains all logic for the policy behavior, including alignment and
    allocation, which need to be updated to support pod level resources.

#### Policy Options

All existing `static` policy options for the CPU Manager are compatible with the
introduction of pod-level resource management. When the `topology-manager-scope`
is set to `pod`, these options will apply to the total resource budget defined
in `pod.spec.resources`. For the initial alpha release, the e2e test plan will
focus on explicitly validating the options that have graduated to General
Availability (GA) to ensure a stable foundation. While all options are expected
to function correctly, comprehensive e2e validation for Beta and Alpha policy
options will be added as this feature progresses towards its own Beta
graduation.

-   **`distribute-cpus-across-numa`**: Distributing CPUs across NUMA nodes will
    have the same behavior as the one for container level resources, the total
    number of pod level CPUs will be evenly distributed (using modulo, any
    remaining will be stripped across the NUMA nodes subset) across all
    available NUMA nodes. It is important to mention that this policy focuses on
    the distribution, not in CPU exclusivity, facilitating parallel algorithms
    to run more efficiently.
-   **`distribute-cpus-across-cores`**: This policy is analogous to the
    distribute-cpus-across-numa policy option, because of spreading the CPU
    allocations out, rather than packing them together. However, in this case
    the total number of pod level CPUs will be spread across cores, and this
    policy relies on changing the sorting algorithm for sockets, cores and CPUs.
    This policy has the purpose of leveraging L2 Cache.
-   **`full-pcpus-only`**: This policy is highly beneficial for multi-tenant
    pods requiring inter-pod isolation, as it helps prevent hyperthread
    contention. The pod level CPUs will be aligned and allocated in full
    physical cores, with a shared CPU pool for its containers.
-   **`align-by-socket`**: This policy ensures all a pod's CPUs remain on the
    same socket when possible, reducing inter-socket latencies and benefiting
    containers that share L3 cache or communicate frequently. The pod level CPU
    pool will be aligned and allocated in the same socket.
-   **`strict-cpu-reservation`**: This policy is compatible and crucial for
    guaranteed workloads, preventing interference from burstable and best-effort
    pods. The pod level CPUs will be reserved for the overall pod, creating a
    shared, reserved and isolated CPU pool for the containers.
-   **`prefer-align-cpus-by-uncorecache`**: This policy optimizes CPU allocation
    across uncore cache groups, enhancing shared cache locality for the overall
    pod level CPU pool.

#### Pod Scope Allocation and Partitioning Algorithm

The CPU Manager implements the pod-scope resource partitioning model. Its
resource-specific responsibilities include:

-   Managing CPU resources as a `cpuset.CPUSet`.
-   The final allocation and partitioning are performed in a single step by the
    `takeByTopology` method. This method selects the total `cpuSet` for the pod
    and simultaneously carves out exclusive CPU slices, respecting all
    configured policy options (e.g., `full-pcpus-only`, `align-by-socket`).

-   **[New Logic]** Defining the `podSharedPool` as the `cpuset.CPUSet` of
    remaining CPUs after all exclusive slices have been taken.

#### State Management and Container Removal

The in-memory state and on-disk checkpoint file will be updated to facilitate
the transition from a container-only model to a pod-aware model. This is
achieved by introducing a new top-level field in the checkpoint file:
`PodCPUAssignments`. This map uses pod UIDs as keys and the new `PodEntry`
struct as values.

The `PodEntry` struct is explicitly designed as an extensible container for all
state information associated with a pod-level resource allocation. While its
initial implementation only encapsulates the overall `CPUSet` assigned to the
pod (`CPUSet cpuset.CPUSet`), using a struct instead of a raw type is a
strategic choice for future-proofing. It allows for the seamless addition of new
fields—such as the original `TopologyHint` used for alignment, the original
resource request, or other metadata—without requiring breaking changes to the
state file schema or complex migration logic for existing checkpoints.

The state management logic is designed to support upgrade state compatibility,
as detailed in the
[Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
section. This ensures that the Kubelet can transparently migrate from
container-only checkpoints to the new pod-level aware format.

```go
// PodCPUAssignments contains pod-level CPU assignments.
type PodCPUAssignments map[string]PodEntry

// PodEntry represents pod-level CPU assignments for a pod
type PodEntry struct {
    CPUSet cpuset.CPUSet `json:"cpuSet,omitempty"`
}
```

The implementation also considers the potential need to enhance the state
representation to support features like the correct handling of CPU quotas, as
described in the "Interaction with CPU Quota Management" section.

When a container is removed, the `RemoveContainer` function first deletes the
container-level assignment. For pods using the `pod` scope, the manager then
checks if any containers from that pod remain in the state. Only when the last
container is removed does the manager release the pod's aggregate `CPUSet` back
to the node's default pool and delete the entry from `PodCPUAssignments`. This
ensures that the NUMA-aligned pool remains reserved for the pod as long as it is
running.

#### Interaction with CPU Quota Management

For `Guaranteed` pods, the `static` CPU manager policy allocates exclusive CPUs
to containers with integer CPU requests. The correct and expected behavior for
these containers is to have their CFS CPU quota disabled (`cpu.cfs_quota_us` set
to `-1`). This prevents the kernel from performing unintended throttling on
workloads that have been guaranteed exclusive access to hardware resources,
which is essential for the performance of latency-sensitive applications.

This behavior was implemented to fix a long-standing bug, and while it is
guarded by the `DisableCPUQuotaWithExclusiveCPUs` feature gate, that gate was
intended as a temporary escape hatch for users whose tooling might have depended
on the old, buggy behavior. Therefore, disabling the CPU quota for exclusively
allocated CPUs must be considered the standard, non-optional behavior that this
KEP must uphold.

**The Conflict:**

The current heuristic for identifying an exclusive allocation is the presence of
an explicit `cpuset` assignment in the CPU Manager's state. The new `pod` scope
logic introduces a conflict with this heuristic. Containers that are *not*
individually `Guaranteed` are assigned the `podSharedPool` as an explicit
`cpuset`. The existing quota logic would incorrectly identify these shared
containers as exclusive and disable their CPU quotas. This would break resource
fairness, allowing a single container in the shared pool to consume all of the
shared CPUs, starving its peers.

This issue is specific to the `pod` scope. The `container` scope is unaffected
because it does not assign an explicit `cpuset` to `non-Guaranteed` containers.

**Required Outcome:**

The implementation of the `pod` scope must include a mechanism to differentiate
between a truly exclusive CPU assignment and a shared-but-isolated assignment
(the `podSharedPool`). The logic that disables the CPU quota must be updated to
use this more granular information. This is not an optional compatibility
measure; it is a mandatory requirement to ensure that:

1.  Truly exclusive containers continue to benefit from the correct, quota-free
    behavior.
2.  Containers within the new `podSharedPool` have their quotas correctly
    enforced to maintain resource fairness between them.

To achieve this, the CPU Manager performs a check to determine the
`ResourceIsolationLevel` for each container. This logic evaluates the
container's resource assignment and its eligibility for exclusivity to
distinguish between:

-   **`ResourceIsolationContainer`**: Resources are exclusive to the container
    (e.g., standard Guaranteed containers). In this case, the CFS quota is
    disabled to allow full, unthrottled access to the exclusive CPUs.
-   **`ResourceIsolationPod`**: Resources are isolated from other pods but
    shared among containers within the same NUMA-aligned pod budget (the
    `podSharedPool`). The CFS quota remains enabled to ensure fairness and
    prevent noisy neighbors within the pod's shared pool.
-   **`ResourceIsolationHost`**: Resources are drawn from the node's general
    shared pool. The CFS quota remains enabled as per standard behavior.

The `ContainerHasExclusiveCPUs` check is updated to return `true` only when the
calculated isolation level is `ResourceIsolationContainer`, preventing the
unintended disabling of quotas for containers in the `podSharedPool`.

```go
// ResourceIsolationLevel defines the level of isolation for a resource.
type ResourceIsolationLevel string

const (
    // ResourceIsolationHost implies the resource is shared with other containers on the host.
    ResourceIsolationHost ResourceIsolationLevel = "host"
    // ResourceIsolationPod implies the resource is isolated from other pods but shared within the pod.
    ResourceIsolationPod ResourceIsolationLevel = "pod"
    // ResourceIsolationContainer implies the resource is exclusive to the container.
    ResourceIsolationContainer ResourceIsolationLevel = "container"
)
```

### Memory Manager

The Memory Manager will be updated to support pod-level resources when its
policy is set to `Static`. The core changes will affect how it determines the
scope of resource management and how it allocates memory to containers within
that scope.

-   **Pod Scope:** When the Topology Manager's scope is `pod`, the Memory
    Manager will manage a set of memory blocks on a single NUMA node for the
    entire pod, based on `pod.spec.resources`. This memory will then be
    partitioned, containers with `Guaranteed` memory requests will be allocated
    exclusive memory blocks from this set, while the remaining memory will form
    a shared pool to be allocated to all other containers in the pod.
-   **Container Scope:** When the Topology Manager's scope is `container`, the
    Memory Manager will continue to allocate exclusive memory on a per-container
    basis. If a pod is `Guaranteed` due to `pod.spec.resources`, the Memory
    Manager will support a mix of containers, those that are individually
    `Guaranteed` will receive exclusive memory blocks, while others will run in
    the node's shared pool. The Memory Manager will ignore `pod.spec.resources`
    for its allocation decisions.

#### Policies

The behavior of existing Memory Manager policies will be supported for pod-level
resources.

-   **`none`**: This policy makes the CPU Manager idle. No changes are needed.
-   **`Static`**: This policy enables the Memory manager functionality, and
    contains all logic for the policy behavior, including alignment and
    allocation, which need to be updated to support pod level resources.
-   **`BestEffort`**: This policy will not be supported, only available on
    Windows nodes. In addition to Windows pods not being supported by pod level
    resources.

#### Pod Scope Allocation and Partitioning Algorithm

The Memory Manager implements the pod-scope resource partitioning model. Its
resource-specific responsibilities include:

-   Managing memory and huge pages as `state.Block` objects.
-   Performing validation after retrieving the initial hint from the Topology
    Manager, which may involve extending the hint or running NUMA violation
    checks before determining the final allocation for the pod.

-   **[New Logic]** Defining the `podSharedPool` as the remaining quantity of
    memory available after exclusive blocks have been reserved.

#### State Management and Container Removal

The in-memory state and on-disk checkpoint file will be updated to facilitate
the transition from a container-only model to a pod-aware model. This is
achieved by introducing a new top-level field in the checkpoint file:
`PodMemoryAssignments`. This map uses pod UIDs as keys and the `PodEntry` struct
as values, mirroring the pattern established in the CPU Manager.

The `PodEntry` struct encapsulates the pod-level memory assignment, currently
containing the overall `MemoryBlocks` slice assigned to the pod. As with the CPU
Manager, using a struct here is a forward-looking design choice. It allows for
the future inclusion of additional metadata—such as the original topology hint
or request details—without disrupting the state file schema or requiring complex
migrations.

The state management logic is designed to support upgrade state compatibility,
as detailed in the
[Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
section. This ensures that the Kubelet can transparently migrate from
container-only checkpoints to the new pod-level aware format.

```go
// PodMemoryAssignments stores memory assignments of pods
type PodMemoryAssignments map[string]PodEntry

// PodEntry represents pod-level memory assignments for a pod
type PodEntry struct {
    MemoryBlocks []Block `json:"memoryBlocks,omitempty"`
}
```

When a container is removed, the `RemoveContainer` function first deletes the
container-level assignment. For pods using the `pod` scope, the manager then
checks if any containers from that pod remain in the state. Only when the last
container is removed does the manager release the pod's aggregate `MemoryBlocks`
back to the node's free pool and delete the entry from `PodMemoryAssignments`.
This ensures that the NUMA-aligned memory remains reserved for the pod as long
as it is running.

### Future Enhancements and Long-Term Vision

This KEP provides a critical foundation for more advanced, device-aware
scheduling and resource management in the future. While this KEP does not make
the Topology Manager aware of device-to-NUMA locality, the enhancements to the
`container` scope are a key enabler for this functionality.

**Enabling Device-Aware, Per-Container NUMA Alignment**

A common challenge for high-performance workloads is the need to co-locate a
container with a specific hardware device (e.g., a high-performance NIC or a
GPU) that resides on a particular NUMA node. At the same time, other containers
in the same pod may need to be placed on different NUMA nodes to avoid resource
contention.

This KEP enables this scenario by allowing a pod to be `Guaranteed` via
`pod.spec.resources`, which in turn allows for a mix of `Guaranteed` and
`non-Guaranteed` containers within that pod. This is the key that unlocks
per-container NUMA alignment for the containers that need it, without forcing
all containers in the pod to be `Guaranteed`. In a future enhancement, a device
plugin could be updated to provide a hint to the Topology Manager, indicating
the NUMA node of the device it is allocating.

With the `container` scope enabled, the Topology Manager could then use this
hint to ensure that the container requiring the device is placed on the correct
NUMA node. Other containers in the same pod, without such a device dependency,
could then be placed on other NUMA nodes, satisfying the complex, per-container
alignment requirements of the workload.

This capability is a crucial step towards a more holistic and device-aware
resource management model in Kubernetes.

### Feature Gate

All code changes will be guarded by a new feature gate named
`PodLevelResourceManagers`, which will depend on the existing
`PodLevelResources` gate.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

-   `k8s.io/kubernetes/pkg/kubelet/cm/topologymanager`: `20250929` - `92.3%`
-   `k8s.io/kubernetes/pkg/kubelet/cm/cpumanager`: `20250929` - `86.8%`
-   `k8s.io/kubernetes/pkg/kubelet/cm/memorymanager`: `20250929` - `81.2%`
-   `k8s.io/kubernetes/pkg/kubelet/allocation/state`: `20250929` - `48.8%`

The following files will have test coverage added or updated: -
`pkg/kubelet/cm/topologymanager/scope_pod_test.go`: - Add tests for pod scope
with pod-level resources. -
`pkg/kubelet/cm/topologymanager/scope_container_test.go`: - Add tests for
container scope with pod-level resources. -
`pkg/kubelet/cm/cpumanager/policy_static_test.go`: - Add extensive tests for the
new partitioning logic in the `Allocate` function and the updated
`podGuaranteedCPUs` function. -
`pkg/kubelet/cm/memorymanager/policy_static_test.go`: - Add extensive tests for
the new partitioning logic and the updated `getPodRequestedResources`
function. - `pkg/kubelet/allocation/state/state_mem_test.go`: - Add tests for
the updated `GetContainerResources` logic.

Additionally, dedicated state restoration tests will be implemented in:

-   `pkg/kubelet/cm/cpumanager/policy_static_restore_test.go`
-   `pkg/kubelet/cm/memorymanager/policy_static_restore_test.go`

These tests will validate that pod-level resource assignments
(`PodCPUAssignments` and `PodMemoryAssignments`) will be correctly persisted to
the checkpoint file and restored accurately after a Kubelet restart, ensuring
alignment consistency across the pod lifecycle.

##### Integration tests

N/A. The functionality is confined to the Kubelet and does not require
interaction with other control plane components. Node e2e tests provide more
effective coverage.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)
-->

New e2e node tests will be created under `test/e2e_node/` to validate the
end-to-end behavior. These tests will:

-   Create pods with various combinations of pod-level and container-level
    resources.
-   Run with different Topology manager scopes, and CPU, and Memory manager
    policies and policy options.
-   Verify that resources are correctly aligned and allocated by inspecting the
    cgroup filesystem on the node (e.g., `CPU set.cpus`).

Additional e2e tests will be implemented to validate the correctness of
pod-level resource accounting and the lifecycle of aggregate resource
allocations.

-   **Pod-Level Accounting for Shared Containers:** These tests will create
    multiple pods, each requesting a significant portion of the node's
    allocatable resources (CPU or Memory) at the pod level, but containing
    multiple `non-Guaranteed` containers. These tests verify that accounting is
    correctly performed against the pod-level budget. If the system incorrectly
    summed container-level resources (which are unset), the pods would be
    rejected; instead, the tests will ensure both pods are admitted and run
    concurrently based on their aggregate pod-level specifications.
-   **Pod Shared Pool Resource Cleanup:** These tests will verify that resources
    will be correctly released from the pod shared pool, even in cases where no
    containers are explicitly linked to it at the time of removal. This
    demonstrates that the container manager correctly identifies and cleans up
    the pod-level aggregate allocation from the state when the pod's container
    lifecycle ends.
-   **Sequential Resource Recycle:** These tests will demonstrate that pod-level
    resources will be correctly released and made available for subsequent
    workloads. It will run two `Guaranteed` pods sequentially, each requesting a
    large portion of the node's resources. The test will verify that the second
    pod will be successfully allocated the same physical resources as the first
    pod, proving that the managers correctly clean up and return the pod-level
    bubble to the node's assignable pool once the pod terminates.

###### Specific Test Scenarios

-   **Pod Scope, Shared Only:** A `Guaranteed` pod with only pod-level resources
    and multiple containers, none of which have individual requests. Verify all
    containers are assigned the same shared CPU set.
-   **Pod Scope, Exclusive Only:** A `Guaranteed` pod with pod-level resources
    and multiple containers, all of which have individual `Guaranteed` requests.
    Verify each container gets a unique, exclusive CPU set.
-   **Pod Scope, Mixed:** A `Guaranteed` pod with pod-level resources, one
    container with exclusive requests, and one container with no requests.
    Verify that the first container is allocated an exclusive CPU set and the
    second is allocated the pod's shared CPU pool.
-   **Pod Scope, Init Containers:** A `Guaranteed` pod with a standard init
    container with exclusive requests. Verify the CPUs allocated to the init
    container are reused in the pod's shared pool for the app containers.
-   **Pod Scope, Sidecar:** A `Guaranteed` pod with a restartable init container
    with exclusive requests. Verify the CPUs allocated to the sidecar are not
    reused and are reserved for the pod's lifetime.
-   **Container Scope, Mixed:** A `Guaranteed` pod using pod-level resources,
    with one container requesting exclusive resources and another with no
    requests. Verify the first container gets an exclusive, NUMA-aligned CPU set
    and the second runs in the node's shared pool.
-   **Failure Case:** A pod where the sum of container-level requests exceeds
    the pod-level budget. Verify the pod is rejected at admission.

-   **GA Policy Options Interaction Test Scenarios:**

    -   **Pod Scope with `full-pcpus-only`:**
        -   A `Guaranteed` pod with a pod-level CPU request that is a multiple
            of the SMT level (i.e., requests full physical cores). Verify that
            the pod is admitted and the allocated `cpuset` for the pod consists
            of full physical cores.
        -   A `Guaranteed` pod with a pod-level CPU request that is *not* a
            multiple of the SMT level. Verify the pod is rejected at admission
            with an SMT alignment error.

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].
-->

#### Alpha

<!-- - Feature implemented behind a feature flag
- Initial e2e tests completed and enabled -->

-   Feature implemented behind the `PodLevelResourceManagers` feature gate,
    disabled by default.
-   Initial unit and e2e tests are completed and running in CI.
-   Support for `pod` and `container` Topology scopes with `static` CPU and
    Memory manager policies is implemented.

<!--
#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- N examples of real-world usage
- N installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation
-->

<!--
- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

This feature is controlled by a feature gate and Kubelet configuration flags.

**Note:** The `PodLevelResources` feature gate must also be enabled for this
feature to function.

**Upgrade:**

1.  Enable the `PodLevelResourceManagers` feature gate on a node.
2.  Set the `--topology-manager-scope` Kubelet flag to `pod` or `container`.
3.  Restart the Kubelet. Existing workloads will not be affected. New pods will
    be subject to the new resource management logic.

**Downgrade:**

1.  Disable the `PodLevelResourceManagers` feature gate.
2.  Restart the Kubelet. The Kubelet will revert to the default container-level
    resource management behavior. Pods that were running with pod-level
    allocations will continue to run, but they will not be re-admitted with
    aligment and exclusive allocation under the old logic if they are restarted.

### Version Skew Strategy

This feature is entirely local to the Kubelet. There is no dependency on the
control plane or other components, so there is no version skew to consider.

## Production Readiness Review Questionnaire

<!--
Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

-   [X] Feature gate (also fill in values in `kep.yaml`)
    -   Feature gate name: `PodLevelResources`
        -   Components depending on the feature gate: Kubelet
        -   Note: Already existing feature for `KEP-2837: Pod Level Resource
            Specifications`
    -   Feature gate name: `PodLevelResourceManagers`
        -   Components depending on the feature gate: Kubelet
        -   Note: New feature gate, dependent on `PodLevelResources`

###### Does enabling the feature change any default behavior?

No. The feature is opt-in. A user must enable the feature gate (together with
the `PodLevelResources` feature gate), configure the Topology Manager with a
`pod` or `container` scope, enable the CPU and Memory managers, and specify pod
level resources in the pod. Existing workloads without pod level resources are
unaffected.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate and restarting the Kubelet will revert the
system to the default container-level resource management behavior. Pods that
were successfully scheduled using the `pod` scope will continue to run with
their allocated resources, but new pods will be subject to the old logic. A pod
that requires pod-level alignment would still be admitted, however no aligment
nor exclusive allocation will be provided.

###### What happens if we reenable the feature if it was previously rolled back?

Re-enabling the feature gate and restarting the Kubelet will restore the
pod-level resource management capabilities. The Kubelet will once again be able
to align and allocate pod level resources pods according to the `pod` and
enhanced `container` scope logic.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

Unit tests will be added to verify that the Kubelet's resource and topology
managers correctly handle the feature gate being enabled or disabled.
Specifically, tests will cover the conditional logic in the `Admit` and
`Allocate` functions.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout or rollback does not impact already running workloads, as their cgroup
settings are already configures. The primary risk involves the Kubelet restart
required for the change and the compatibility of the on-disk state files (e.g.,
`/var/lib/kubelet/cpu_manager_state`), which affects new or restarting pods, as
they will be subject to the new (or old) admission logic and could fail to come
back up if an issue is present.

*   **Rollout (Enabling the Feature):** This is a low-risk operation.

    *   **Behavior:** Before this feature, pods with `pod.spec.resources` were
        ignored by the resource managers. After enabling the feature, new or
        restarted pods will be correctly aligned, and new entries will be added
        to the state file.
    *   **Mitigation (Forward Compatibility):** The new Kubelet version will be
        implemented to transparently read the old state file format. When it
        admits a new pod using this feature, it will write the new pod-aware
        entries, ensuring a seamless upgrade path without manual intervention.

*   **Rollback (Disabling the Feature):** This is a higher-risk operation.

    *   **Failure Mode:** The main risk is state file incompatibility. An older
        Kubelet may fail to parse the new state file format, leading to a crash
        loop or preventing any Guaranteed pods from being admitted on the node.
    *   **Mitigation (Backward Compatibility):** The new state file will be
        structured to maintain backward compatibility. New pod-level fields will
        be added in a way that allows older Kubelet versions to safely ignore
        them. However, to completely eliminate any risk of parsing errors, the
        safest and recommended rollback procedure involves draining the node and
        deleting the state file.

**Recommended Operator Process for Rollout and Rollback**

To ensure a safe rollout and rollback, cluster administrators should follow
these distinct procedures.

*   **Rollout Process (Enabling the Feature)**

    1.  **Step 1: Canary Rollout:** Perform the upgrade on a small, isolated
        group of "canary" nodes first. This involves enabling both the
        `PodLevelResources` and `PodLevelResourceManagers` feature gates in the
        Kubelet configuration and restarting the Kubelet on those nodes. This
        limits the blast radius of any potential negative impact.
    2.  **Step 2: Intensive Monitoring:** During the canary rollout, closely
        monitor the key metrics identified in the "What specific metrics should
        inform a rollback?" section. A spike in errors like
        `pod_topology_manager_pinning_errors_total` or a significant increase in
        `topology_manager_admission_duration_seconds` are critical indicators of
        a problem.
    3.  **Step 3: Halt on Regression:** If monitoring on the canary nodes
        detects a regression (e.g., admission errors exceed a predefined
        threshold), halt the rollout immediately and proceed to the Rollback
        Process.

*   **Rollback Process (Disabling the Feature)**

    1.  **Step 1: Kubelet Configuration Change:** On the affected nodes (either
        the canary group or all nodes if the rollout was wider), update the
        Kubelet configuration to disable the feature gate
        (`--feature-gates=PodLevelResourceManagers=false`). The
        `PodLevelResources` gate can remain enabled if other cluster features
        depend on it.
    2.  **Step 2: Kubelet Restart:** Restart the Kubelet service on the nodes to
        apply the configuration change.
    3.  **Step 3: (If Necessary) Drain Node and Delete State File:** If the
        Kubelet enters a crash loop or fails to start due to a state file
        parsing error, the safest mitigation is to drain the node, manually
        delete the state file (e.g., `/var/lib/kubelet/cpu_manager_state`), and
        then restart the Kubelet. This ensures the Kubelet starts with a clean
        state.
    4.  **Step 3: Understand the Impact:**
        *   **Running Pods:** Pods that were successfully allocated using the
            feature will continue to run with their existing cgroup settings.
            They are not affected.
        *   **New/Restarted Pods:** Any new pods, or existing pods that are
            restarted, will now be subject to the old logic. Pods that rely on
            the `pod` scope for NUMA alignment will no longer receive it and
            will run in the node's shared pool.
    5.  **Step 4: Verification:** Verify the rollback by inspecting the Kubelet
        logs on a rolled-back node to ensure it has started without parsing
        errors from the state file. An operator can also confirm that new pods
        requiring pod-level alignment are no longer receiving it by inspecting
        their cgroups on the node.

###### What specific metrics should inform a rollback?

When the `PodLevelResourceManagers` feature is enabled, operators should closely
monitor the following metrics. A sustained or significant increase in any of
these, particularly the new pod-level error metrics, would be a strong indicator
that a rollback is necessary:

-   **`resource_manager_allocation_errors_total{source="pod"}`**: This is the
    most critical signal. A spike in this new metric for either
    `resource_name="cpu"` or `resource_name="memory"` indicates that the new
    pod-level partitioning and allocation logic for exclusive resources is
    failing. This is a strong reason to initiate a rollback.
-   **`topology_manager_admission_errors_total`**: While an existing metric, a
    significant increase after enabling the feature could indicate that the
    Topology Manager is rejecting pods due to issues introduced by the new
    pod-level resource management, even if the specific
    `pod_topology_manager_pinning_errors_total` is not yet high.
-   **`cpu_manager_pinning_errors_total`**: An increase in this existing metric
    could indicate broader CPU pinning issues, potentially exacerbated or
    triggered by the new pod-level logic.
-   **`memory_manager_pinning_errors_total`**: An increase in this existing
    metric could indicate broader memory pinning issues, potentially exacerbated
    or triggered by the new pod-level logic.

Additionally, general system health metrics such as node CPU/memory utilization,
pod startup latency, and overall application error rates should be monitored.
Any unexpected degradation in these, correlated with the feature enablement,
would also warrant investigation and potential rollback.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Manual testing of the upgrade/downgrade path will be performed as part of the
alpha development cycle. As the feature matures towards Beta, dedicated e2e
tests will be implemented to automate this validation. These tests will cover
the full `upgrade->downgrade->upgrade` path, verifying the Kubelet's ability to
handle state file transitions gracefully and manage pod lifecycles correctly
during version changes.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

N/A.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

An operator can determine if the feature is in use by monitoring the new Kubelet
metrics exposed when the feature is enabled. A non-zero value for any of the
following metrics is a definitive signal that workloads are actively using the
pod-level resource management capabilities on a given node:

-   `resource_manager_allocations_total{source="pod"}`: A non-zero value for
    this metric series indicates that the resource managers have successfully
    performed at least one exclusive allocation using the new pod-level logic.
-   `resource_manager_container_assignments{assignment_type="shared_from_pod"}`:
    A non-zero value for this metric series indicates that one or more
    containers are currently running in a pod-level shared pool.

By querying these metrics, an operator can get a clear picture of the overall
adoption and specific usage patterns of the feature across the cluster.

As a secondary method, an operator can inspect the Kubelet configuration on a
node to confirm that the `PodLevelResourceManagers` feature gate is enabled and
that the `--topology-manager-scope` is set to `pod`. However, the metrics
provide the most direct and real-time confirmation of active usage by workloads.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.

- [ ] Events
  - Event Reason:
- [ ] API .status
  - Condition name:
  - Other field:
- [ ] Other (treat as last resort)
  - Details:

-->

-   [X] Events
    -   Event Reason: A pod failing admission due to a `TopologyAffinityError`
        will generate an event on the pod, visible via `kubectl describe pod`.
-   [ ] Metrics
    -   Operators can monitor the new Kubelet metrics to see if pod-level
        resource management is active and healthy.
        -   A steady increase in
            `resource_manager_allocations_total{source="pod"}` confirms that the
            new pod-scope logic is successfully performing exclusive
            allocations.
        -   The `resource_manager_container_assignments` gauge provides a
            real-time view of how many containers are running in each state
            (`exclusive_from_pod`, `shared_from_pod`, etc.).
        -   An increase in
            `resource_manager_allocation_errors_total{source="pod"}` would
            indicate failures in the new logic.
-   [X] Other (treat as last resort)
    -   Details: For end-users, after a pod is scheduled, a user with access to
        the node can inspect the cgroup filesystem to verify resource pinning.
        For CPUs, this can be done by checking the `cpuset.cpus` file in the
        container's cgroup directory. For memory, the `cpuset.mems` file will
        show the NUMA node from which memory is allocated.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

The pod admission latency should not significantly increase. The
`topology_manager_admission_duration_seconds` metric can be used to monitor
this. A reasonable SLO would be to keep the 99th percentile of this duration
within a small margin of the baseline without the feature enabled.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

An operator can determine the health of this feature by monitoring a combination
of existing Kubelet metrics and new metrics introduced specifically for
pod-level resource management.

-   [ ] New Metrics

    -   Metric name: `resource_manager_allocations_total`
        -   Type: `CounterVec`
        -   Labels: `resource_name` ("cpu", "memory"), `source` ("pod", "node")
        -   Description: Counts the total number of exclusive resource
            allocations performed by a manager. The `source` label distinguishes
            between allocations drawn from the node-level pool (`node`) versus a
            pre-allocated pod-level pool (`pod`).
    -   Metric name: `resource_manager_allocation_errors_total`
        -   Type: `CounterVec`
        -   Labels: `resource_name` ("cpu", "memory"), `source` ("pod", "node")
        -   Description: Counts errors encountered during exclusive resource
            allocation, distinguished by the intended allocation source.
    -   Metric name: `resource_manager_container_assignments`
        -   Type: `CounterVec`
        -   Labels: `resource_name` ("cpu", "memory"), `assignment_type`
            ("node_exclusive", "pod_exclusive", "pod_shared")
        -   Description: Counts the total number of containers that will be
            granted a specific type of resource assignment. This provides
            visibility into how many containers are running with exclusive
            resources (from the node or pod pool) versus the pod-level shared
            pool.

-   [X] Already existing Metrics

    -   Metric name: `topology_manager_admission_requests_total`
        -   Aggregation method: `sum()`
    -   Metric name: `topology_manager_admission_errors_total`
        -   Aggregation method: `sum()`
    -   Metric name: `topology_manager_admission_duration_ms`
        -   Aggregation method: `histogram_quantile`
    -   Metric name: `cpu_manager_pinning_errors_total`
        -   Aggregation method: `sum()`
    -   Metric name: `memory_manager_pinning_errors_total`
        -   Aggregation method: `sum()`

All of the previous metrics are exposed by the Kubelet. If not, it will be
explicitly mentioned.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

The proposed metrics provide excellent observability into both allocation events
and the current state of container assignments. No additional metrics are deemed
necessary for the initial alpha release.

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

N/A. This feature is entirely local to the Kubelet.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

N/A. The feature is entirely node-local.

###### Will enabling / using this feature result in introducing new API types?

N/A.

###### Will enabling / using this feature result in any new calls to the cloud provider?

N/A.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

N/A.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

A negligible increase in pod admission time is expected, but it should not
impact any existing SLIs/SLOs in a meaningful way.

The additional logic for pod-level resource management is executed entirely
within the Kubelet's existing admission control flow. The changes primarily
involve extra in-memory checks and calculations to determine the pod's total
resource budget and partition it among its containers. Crucially, this process
does not introduce any new API calls, network requests, or other high-latency
operations that would significantly affect performance.

The computational overhead is comparable to the existing logic for
container-level resource alignment. The hint generation and merging process for
a single pod-level request is computationally similar to handling multiple
container-level requests. Therefore, any increase in the
`topology_manager_admission_duration_seconds` metric is expected to be almost
non-existent.

This is a one-time cost incurred during pod creation, and the performance
benefits of proper NUMA alignment for the running workload are expected to far
outweigh this negligible admission latency.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The additional state managed by the Kubelet is minimal and should not result
in a non-negligible increase in resource usage.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. This feature only affects the assignment of CPU and memory resources and
does not interact with PIDs, sockets, or inodes.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

As a node-local feature, it is not impacted by the availability of the API
server or etcd. The Kubelet will continue to make allocation decisions for pods
based on its local state.

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

-   [Pod admission failure due to lack of NUMA alignment]
    -   Detection: The `topology_manager_admission_errors_total` metric will
        increase. The pod will enter a `Terminated` state with a
        `TopologyAffinityError` reason.
    -   Mitigations: This is expected behavior. The pod must be rescheduled on a
        node that can satisfy its NUMA alignment requirements.
    -   Diagnostics: Kubelet logs will contain messages from the Topology
        Manager indicating which resource's hints could not be satisfied.
    -   Testing: This is a core part of the e2e test plan.

###### What steps should be taken if SLOs are not being met to determine the problem?

If the `topology_manager_admission_duration_seconds` SLO is not being met, an
operator should inspect the Kubelet logs on the affected node to look for
performance bottlenecks in the hint generation or merging logic. Profiling the
Kubelet may be necessary in extreme cases.

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

-   **v1.36**: Target for initial alpha implementation.

## Drawbacks

The primary drawback is the introduction of additional complexity into the
Kubelet's resource management logic. The new partitioning model for the `pod`
scope is more complex than the existing per-container allocation. This increases
the surface area for potential bugs and requires more extensive testing.
Additionally, the different behaviors of the `pod` and `container` scopes when
handling pod-level resources could be a source of confusion for users if not
documented clearly.

## Alternatives

The primary alternative considered was a simpler model for the `pod` scope where
the entire pod-level resource allocation would be assigned to every container in
the pod. This was rejected because it does not support the critical use case of
having mixed-criticality containers within the same pod (e.g., a primary
application with guaranteed resources and a sidecar with shared resources). The
chosen partitioning model provides much greater flexibility and more efficient
resource utilization.

## Infrastructure Needed (Optional)

N/A.
