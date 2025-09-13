<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
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
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: AI/ML Workload with Data-Ingestion Sidecar](#story-1-aiml-workload-with-data-ingestion-sidecar)
    - [Story 2: Workload with a Device-Specific Infrastructure Container](#story-2-workload-with-a-device-specific-infrastructure-container)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Topology Manager](#topology-manager)
    - [Policies](#policies)
    - [Policy Options](#policy-options)
    - [Scopes](#scopes)
      - [<code>pod</code> Scope](#pod-scope)
      - [<code>container</code> Scope](#container-scope)
  - [CPU Manager](#cpu-manager)
    - [Pod Scope Allocation and Partitioning Algorithm](#pod-scope-allocation-and-partitioning-algorithm)
  - [Memory Manager](#memory-manager)
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

## Summary

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.
-->

This KEP proposes extending the Kubelet's Topology, CPU, and Memory Managers to
support pod-level resource specifications. Currently, these managers operate on
a per-container basis. This enhancement will enable them to make NUMA alignment
and resource allocation decisions for a pod as a single unit, based on
`pod.spec.resources`. This change introduces a more flexible and powerful
resource management model, particularly for performance-sensitive workloads. All
functionality will be controlled by a new `PodLevelResourceManagers` feature
gate.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

The introduction of Pod-Level Resources
([KEP-2837](https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/2837-pod-level-resource-spec/README.md))
allows users to define a resource budget for an entire pod. However, the
Kubelet's key resource management components—Topology, CPU, and Memory
Managers—are not yet able to leverage this pod-level information. They continue
to operate on a per-container basis, which limits the potential performance
gains for workloads that benefit from having all their resources co-located on
the same NUMA node.

For high-performance computing (HPC), AI/ML, and network function virtualization
(NFV) workloads, minimizing memory and CPU latency is critical. By making the
resource managers aware of the pod's total resource budget, we can ensure the
entire pod is treated as a single NUMA-aligned unit, significantly improving
performance.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

1.  **Enable Pod-Scope Alignment:** Update the Topology Manager's `pod` scope to
    use `pod.spec.resources` for NUMA alignment decisions.
2.  **Implement Pod-Level Allocation:** Enhance the CPU and Memory managers to
    allocate a single, unified resource pool for a pod based on its pod-level
    requests.
3.  **Support Resource Partitioning:** For the `pod` scope, implement a model
    where the pod's allocated resource pool is partitioned. Containers that
    individually qualify for `Guaranteed` QoS will receive exclusive "slices"
    from this pool, while all other containers will share the remainder.
4.  **Ensure Container-Scope Compatibility:** Allow the `container` scope to
    work with pods that use pod-level resources to achieve a `Guaranteed` QoS,
    enabling a mix of containers that receive exclusive, NUMA-aligned resources
    and containers that do not.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

1.  Changing the fundamental Kubernetes QoS model. This KEP leverages the
    existing QoS calculation, which prioritizes `pod.spec.resources` when
    present.
2.  Introducing new user-facing APIs or policy options. This enhancement focuses
    on extending the behavior of existing policies and scopes.
3.  Supporting non-Guaranteed pods for exclusive resource allocation.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

We will introduce a new feature gate, `PodLevelResourceManagers`, to enable the
desired functionality. When this gate is active, the Topology, CPU, and Memory
managers will be updated to recognize and act upon `pod.spec.resources`.

The core of the proposal is to enable two flexible resource management models:

1.  **Pod Scope:** The Topology Manager will secure a single NUMA-aligned
    resource pool for the entire pod. The CPU and Memory managers will then
    partition this pool, carving out exclusive slices for containers with
    specific `Guaranteed` requests and creating a shared pool for all other
    containers from the remainder.
2.  **Container Scope:** This scope will continue to manage resources on a
    per-container basis. With the new feature gate, it will correctly handle
    `Guaranteed` pods that use pod-level resources, allowing some containers to
    receive exclusive NUMA-aligned resources while others in the same pod use
    the node's shared pool.

This hybrid approach provides significant flexibility, allowing users to
co-locate critical, performance-sensitive containers with less critical sidecars
or helper containers, all within a single, NUMA-aligned pod.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1: AI/ML Workload with Data-Ingestion Sidecar

As a machine learning engineer, I am deploying a pod that contains two
containers: a primary training container that requires significant, guaranteed
CPU resources for performance, and a data-ingestion sidecar that has minimal,
burstable resource needs. I want the entire pod to be NUMA-aligned to ensure the
training container has fast access to memory, but I don't want to overprovision
resources for the sidecar.

By setting `pod.spec.resources` to make the pod `Guaranteed` and specifying
`static` CPU policy with `pod` scope, I can define a large resource budget for
the pod. I then set specific, `Guaranteed` requests on my training container.
The system will allocate a NUMA-aligned pool for the pod, carve out an exclusive
slice for my training container, and allow the data-ingestion sidecar to use the
remaining shared portion of the pod's resources.

#### Story 2: Workload with a Device-Specific Infrastructure Container

As a platform engineer, I am deploying a pod with a main workload and an
infrastructure sidecar that needs access to a specific hardware device (e.g., a
high-performance NIC) that is physically located on a particular NUMA node. The
main workload should run on a different NUMA node to avoid resource contention.

While this KEP does not make the Topology Manager aware of device-to-NUMA
locality, it provides the foundation for such a feature. A future enhancement
could allow the device plugin to provide a hint for a specific NUMA node. With
the `container` scope, the Topology Manager could then ensure the infrastructure
sidecar is placed on that specific NUMA node, while the main workload could be
placed on another, satisfying the complex alignment requirements.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

-   This feature is dependent on the `PodLevelResources` feature gate being
    enabled.
-   The functionality is only implemented for the `static` CPU Manager policy
    and the `Static` Memory Manager policy. Other policies like `none` are
    unaffected as they do not perform NUMA-aware allocations.
-   For the `pod` scope, the sum of all container-level resource requests must
    not exceed the pod-level resource budget defined in `pod.spec.resources`.
    This is enforced by Kubelet admission control.
-   The `container` scope will ignore `pod.spec.resources` for its allocation
    decisions, focusing only on per-container requests.
-   Interaction with In-Place Pod Vertical Scaling is not part of this KEP. The
    focus is on the initial allocation of resources.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

-   **Risk:** Incorrect resource accounting could lead to over- or
    under-allocation of resources.
    -   **Mitigation:** The implementation will be accompanied by extensive unit
        and e2e tests covering all policy and scope combinations to validate the
        new partitioning and allocation logic.
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

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Topology Manager

When the `topology-manager-scope` is set to `pod`, the manager will use the
pod's `spec.resources` to determine the NUMA affinity for the entire pod. When
the scope is `container`, it will continue to operate on a per-container basis,
but it will now correctly handle `Guaranteed` pods that use pod-level resources,
allowing for a mix of aligned and non-aligned containers.

The behavior of existing Topology Manager policies and policy options will be
extended to support pod-level resources.

#### Policies

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

##### `pod` Scope

When the Kubelet is configured with `--topology-manager-scope=pod`, the resource
managers will offer alignment and exclusive allocation for the entire pod. If a
pod specifies both pod-level and container-level resources, a partitioning model
is used.

Pod-level resources | Container resources | CPU and Memory manager behavior
:------------------ | :------------------ | :------------------------------
Unset               | Set                 | Current behavior (alignment and exclusive allocation based on container resources)
Set                 | Unset               | Alignment and exclusive allocation for the overall pod. All containers share the resulting resource pool.
Set                 | Set                 | Alignment and exclusive allocation for the overall pod. The pod's resource pool is partitioned into exclusive slices for `Guaranteed` containers and a shared pool for the rest.

When using pod level resources, the QoS will be completely determined by them,
however the scope behavior will be determined by both pod and container level,
requiring an additional and similar in essence to the QoS check for the
container level resources to determine which containers are eligible for
exclusive allocation.

New cases that need to be covered:

-   Pod level request/limit set, Container level request/limit unset
    -   Scope is set to pod, only pod level resources are specified. The
        alignment will be done using pod level resources. Allocation will be
        done to the overall pod, and the obtained resource pool will be
        propagated to each container.
-   Pod level request/limit set, Container level request/limit set
    -   The overall pod will receive the corresponding alignment and allocation
        for the requested resources, which will be propagated to the containers
        that did not fulfill the Guaranteed QoS check, while all the individual
        containers that properly specify to fulfill the QoS check will be
        allocated exclusive resources subtracted from the overall pod level
        resource pool.

Pod Scope                 | Spec                                                                               | Hint Generation | Allocation
:------------------------ | :--------------------------------------------------------------------------------- | :-------------- | :---------
Current behavior          | Container 1: 3 CPU <br> Container 2: 1 CPU <br> Container 3: 1 CPU                 | {1,2,3,4,5}     | Container 1: {1, 2, 3} <br> Container 2: {4} <br> Container 3: {5}
New behavior <br> Full    | Pod: 5 CPU <br> Container 1: 3 CPU <br> Container 2: 1 CPU <br> Container 3: 1 CPU | {1,2,3,4,5}     | Container 1: {1, 2, 3} <br> Container 2: {4} <br> Container 3: {5}
New behavior <br> Partial | Pod: 5 CPU <br> Container 1: 3 CPU <br> Container 2 <br> Container 3               | {1,2,3,4,5}     | Container 1: {1, 2, 3} <br> Container 2: {4, 5} <br> Container 3: {4, 5}

###### Examples

The examples focus on CPU resource behavior, however the memory behavior is
analogous.

**Example 1: Shared Pool Only**

This example shows a `Guaranteed` pod with a pod-level resource budget. Neither
container requests individual resources, so they will both run in the pod's
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
      cpu: "2"
      memory: "2Gi"
    limits:
      cpu: "2"
      memory: "2Gi"
  containers:
  - name: container-1
    image: example-image
  - name: container-2
    image: example-image
```

<!-- mdformat on -->

**Expected Allocation:**

-   The Topology Manager will secure a NUMA-aligned pool of 2 CPUs and 2Gi of
    memory.
-   Since there are no containers with exclusive requests, the pod's shared pool
    is the entire 2-CPU pool.
-   Both `container-1` and `container-2` will be assigned the same 2-CPU CPU
    set.

**Example 2: Mixed Allocation (Exclusive and Shared)**

This example shows a `Guaranteed` pod with a 4-CPU budget. `container-2`
requests an exclusive 2-CPU slice, while `container-1` will use the remainder.

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
  - name: container-2
    image: example-image
    resources:
      requests:
        cpu: "2"
        memory: "2Gi"
      limits:
        cpu: "2"
        memory: "2Gi"
```
<!-- mdformat on -->

**Expected Allocation:**
- The Topology Manager will secure a NUMA-aligned pool of 4 CPUs.
- `container-2` will be allocated an exclusive 2-CPU slice from the pool.
- The pod's shared pool will be the remaining 2 CPUs.
- `container-1` will be assigned the 2-CPU shared pool.

##### `container` Scope

When the Kubelet is configured with `--topology-manager-scope=container`, the
resource managers will offer alignment and exclusive allocation on a
per-container basis.

Pod-level resources | Container resources | CPU and Memory manager behavior
:------------------ | :------------------ | :------------------------------
Unset               | Set                 | Current behavior (alignment and exclusive allocation based on container resources)
Set                 | Unset               | No alignment or exclusive allocation. Containers run in the node's shared pool.
Set                 | Set                 | Alignment and exclusive allocation for containers that specify `Guaranteed` resources; pod-level resources are ignored for allocation decisions.

The same way as the pod scope, the QoS will be only determined by the pod level
resources, regardless of having container level resources specified, thus,
requiring to make a per-container QoS check for all containers within the pods
too, to effectively determine if any of them will be eligible for alignment and
allocation.

Cases that need to be considered:

- Pod level request/limit set, Container level request/limit unset
  - No possible alignment, container requests/limits do not exist.
- Pod level request/limit set, Container level request/limit set
  - The overall pod will not receive any alignment and allocation for the
  requested resources, only the individual containers that properly specify and
  fulfill the Guaranteed QoS will be allocated exclusive resources.

Container Scope           | Spec                                                                               | Hint Generation                       | Allocation
:------------------------ | :--------------------------------------------------------------------------------- | :------------------------------------ | :---------
Current behavior          | Container 1: 3 CPU <br> Container 2: 1 CPU <br> Container 3: 1 CPU                 | {1,2,3}, {4}, {5}                     | Container 1: {1, 2, 3} <br> Container 2: {4} <br> Container 3: {5}
New behavior <br> Full    | Pod: 5 CPU <br> Container 1: 3 CPU <br> Container 2: 1 CPU <br> Container 3: 1 CPU | {1,2,3}, {4}, {5}                     | Container 1: {1, 2, 3} <br> Container 2: {4} <br> Container 3: {5}
New behavior <br> Partial | Pod: 5 CPU <br> Container 1: 3 CPU <br> Container 2 <br> Container 3               | {1,2,3}, not preferred, not preferred | Container 1: {1,2,3} <br> Container 2: no guaranteed resources <br> Container 3: no guaranteed resources

###### Examples

The examples focus on CPU resource behavior, however the memory behavior is
analogous.

**Example 1: Mixed Allocation (Exclusive and Node Shared)**

This example shows a `Guaranteed` pod using pod-level resources. Because the scope is `container`, the managers will evaluate each container individually. `container-2` gets an exclusive allocation, while `container-1` runs in the node-wide shared pool.

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
      cpu: "2"
      memory: "2Gi"
    limits:
      cpu: "2"
      memory: "2Gi"
  containers:
  - name: container-1
    image: example-image
  - name: container-2
    image: example-image
    resources:
      requests:
        cpu: "1"
        memory: "1Gi"
      limits:
        cpu: "1"
        memory: "1Gi"
```

<!-- mdformat on -->

**Expected Allocation:**

-   The Topology Manager will evaluate `container-2` and generate hints for its
    1 CPU request, resulting in an exclusive, NUMA-aligned allocation for it.
-   The Topology Manager will evaluate `container-1` and, seeing no resource
    requests, will generate a "no preference" hint.
-   `container-2` will be assigned its own exclusive 1-CPU CPU set.
-   `container-1` will run in the general, node-wide shared pool, along with
    containers from other pods.

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
      cpu: "2"
      memory: "2Gi"
    limits:
      cpu: "2"
      memory: "2Gi"
  containers:
  - name: container-1
    image: example-image
  - name: container-2
    image: example-image
```

<!-- mdformat on -->

**Expected Allocation:**

-   The Topology Manager evaluates each container individually. Since neither
    `container-1` nor `container-2` has its own resource requests, "no
    preference" hints are generated for both.
-   The pod-level resources are **not** considered for allocation or NUMA
    alignment by the managers in this scope.
-   Both containers will run in the general, node-wide shared pool. No exclusive
    resources will be allocated for this pod.

### CPU Manager

With the `static` policy enabled, the CPU manager will:

-   **Pod Scope:** Allocate a single CPU set for the entire pod based on
    `pod.spec.resources`. This CPU set will then be partitioned. Containers with
    `Guaranteed` CPU requests will be assigned exclusive CPUs from this set. The
    remaining CPUs will form a shared pool for all other containers in the pod.
-   **Container Scope:** Continue to allocate exclusive CPUs on a per-container
    basis for containers in `Guaranteed` pods. It will ignore
    `pod.spec.resources` for its allocation decisions.

#### Pod Scope Allocation and Partitioning Algorithm

1.  **Initial Pod Allocation:** The `Allocate` function is called for the first
    container of the pod. The manager calculates the total pod budget from
    `pod.spec.resources`. Guided by the Topology Manager's hint, it allocates a
    single, NUMA-aligned CPU set of that size.
2.  **Calculation of Shared Pool:** The manager calculates the `podSharedPool`
    by taking the total allocated pool and subtracting the sum of all exclusive
    CPU requests from the pod's `Guaranteed` containers (both init and app
    containers).
3.  **Handling Init Containers:**
    -   **Standard Init Containers:** An exclusive slice is carved out from the
        pool. Upon completion, these CPUs are added to a per-pod `reusable`
        pool, making them available to subsequent app containers within the
        pool.
    -   **Restartable Init Containers (Sidecars):** An exclusive slice is carved
        out and reserved for the entire pod lifecycle. These CPUs are *not*
        added to the reusable pool.
4.  **Allocation and State Update:** The manager iterates through all
    containers, assigning either their reserved slice or the `podSharedPool`.
    These individual assignments are written to the state file. For all
    subsequent `Allocate` calls for containers in this pod, the manager will
    simply read the pre-calculated assignment from the state.

### Memory Manager

With the `Static` policy enabled, the Memory Manager will mirror the CPU
Manager's behavior:

-   **Pod Scope:** Allocate a set of memory blocks on a single NUMA node for the
    entire pod. This memory will be partitioned into exclusive allocations for
    `Guaranteed` containers and a shared pool for the rest. The same logic for
    handling standard and restartable init containers will apply.
-   **Container Scope:** Allocate exclusive memory on a per-container basis for
    containers in `Guaranteed` pods.

### Feature Gate

All code changes will be guarded by a new feature gate named
`PodLevelResourceManagers`, which will depend on the existing
`PodLevelResources` gate.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.

- `<package>`: `<date>` - `<test coverage>`
-->

-   `pkg/kubelet/cm/topologymanager/scope_pod_test.go`: Add tests for pod scope
    with pod-level resources.
-   `pkg/kubelet/cm/topologymanager/scope_container_test.go`: Add tests for
    container scope with pod-level resources.
-   `pkg/kubelet/cm/cpumanager/policy_static_test.go`: Add extensive tests for
    the new partitioning logic in the `Allocate` function and the updated
    `podGuaranteedCPUs` function.
-   `pkg/kubelet/cm/memorymanager/policy_static_test.go`: Add extensive tests
    for the new partitioning logic and the updated `getPodRequestedResources`
    function.
-   `pkg/kubelet/allocation/state/state_mem_test.go`: Add tests for the updated
    `GetContainerResources` logic.

##### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->


<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)
-->

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
-   Run with different Topology, CPU, and Memory manager policies and scopes.
-   Verify that resources are correctly aligned and allocated by inspecting the
    cgroup filesystem on the node (e.g., `CPU set.cpus`).
-   Use the `kind-config.yaml` provided in the design document to configure the
    Kubelet with the necessary feature gates and manager policies.

###### Specific Test Scenarios

-   **Pod Scope, Shared Only:** A `Guaranteed` pod with only pod-level resources
    and multiple containers, none of which have individual requests. Verify all
    containers are assigned the same shared CPU set.
-   **Pod Scope, Exclusive Only:** A `Guaranteed` pod with pod-level resources
    and multiple containers, all of which have individual `Guaranteed` requests.
    Verify each container gets a unique, exclusive CPU set.
-   **Pod Scope, Mixed:** A `Guaranteed` pod with pod-level resources, one
    container with exclusive requests, and one container with no requests Verify
    the first container gets an exclusive CPU set and the second gets the
    remainder of the pod's allocation.
-   **Pod Scope, Init Containers:** A `Guaranteed` pod with a standard init
    container with exclusive requests. Verify its resources are reused by the
    app containers.
-   **Pod Scope, Sidecar:** A `Guaranteed` pod with a restartable init container
    with exclusive requests. Verify its resources are not reused and are
    reserved for the pod's lifetime.
-   **Container Scope, Mixed:** A `Guaranteed` pod using pod-level resources,
    with one container requesting exclusive resources and another with no
    requests. Verify the first container gets an exclusive, NUMA-aligned CPU set
    and the second runs in the node's shared pool.
-   **Failure Case:** A pod where the sum of container-level requests exceeds
    the pod-level budget. Verify the pod is rejected at admission.

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
-   Support for `pod` and `container` scopes with `static` CPU and Memory
    manager policies is implemented.

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

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

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

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?
-->

-   [X] Feature gate (also fill in values in `kep.yaml`)
    -   Feature gate name: `PodLevelResourceManagers`
    -   Components depending on the feature gate: Kubelet

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No. The feature is opt-in. A user must enable the feature gate (together with
the `PodLevelResources` feature gate), configure the Topology Manager with a
`pod` or `container` scope, enable the CPU and Memory managers, and specify pod
level resources in the pod. Existing workloads and configurations are
unaffected.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes. Disabling the feature gate and restarting the Kubelet will revert the
system to the default container-level resource management behavior. Pods that
were successfully scheduled using the `pod` scope will continue to run with
their allocated resources, but new pods will be subject to the old logic. A pod
that requires pod-level alignment would still be admitted, however no aligment
nor exclusive allocation will be provided.

###### What happens if we reenable the feature if it was previously rolled back?

Re-enabling the feature gate and restarting the Kubelet will restore the
pod-level resource management capabilities. The Kubelet will once again be able
to admit and manage pods according to the `pod` and enhanced `container` scope
logic.

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

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

A rollout could fail if a bug in the new allocation logic prevents `Guaranteed`
pods from starting. This would not impact already running workloads, but it
would prevent new `Guaranteed` pods from being aligned and allocated on the
affected node. A rollback (disabling the feature gate and restarting the
Kubelet) would mitigate this for new pods.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

A significant increase in the following metrics after enabling the feature would
be a strong indicator that a rollback is needed:

-   `topology_manager_admission_errors_total`: Indicates that the Topology
    Manager is rejecting pods that it would have previously admitted.
-   `cpu_manager_pinning_errors_total`: Indicates failures in the CPU allocation
    logic.
-   `memory_manager_pinning_errors_total`: Indicates failures in the Memory
    allocation logic.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

Manual testing of the upgrade/downgrade path will be performed as part of the
alpha development cycle.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

N/A.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

An operator can determine if the feature is in use by inspecting the Kubelet
configuration on each node to see if the `PodLevelResourceManagers` feature gate
is enabled and if the `--topology-manager-scope` is set to `pod` or `container`.
Additionally, inspecting the logs for messages related to pod-level allocation
would indicate usage.

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
-   [X] Other (treat as last resort)
    -   Details: An operator can check the `cpu_manager_pinning_requests_total`
        and `memory_manager_pinning_requests_total` metrics from the Kubelet to
        see if pinning requests are being made. For end-users, after a pod is
        scheduled, a user with access to the node can inspect the cgroup
        filesystem to verify resource pinning. For CPUs, this can be done by
        checking the `cpuset.cpus` file in the container's cgroup directory. For
        memory, the `cpuset.mems` file will show the NUMA node from which memory
        is allocated.

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

<!--
Pick one more of these and delete the rest.

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:
-->

-   [X] Metrics
    -   Metric name: topology_manager_admission_duration_seconds
        -   Aggregation method: histogram
        -   Components exposing the metric: Kubelet
    -   Metric name: topology_manager_admission_errors_total
        -   Aggregation method: sum()
        -   Components exposing the metric: Kubelet
    -   Metric name: cpu_manager_pinning_errors_total
        -   Aggregation method: sum()
        -   Components exposing the metric: Kubelet
    -   Metric name: memory_manager_pinning_errors_total
        -   Aggregation method: sum()
        -   Components exposing the metric: Kubelet

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

For the `pod` scope, it would be beneficial to have metrics that track the
number of pods with partitioned resources and the number of containers with
exclusive slices versus those in the shared pool. This would provide better
insight into how the feature is being used.

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

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

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

N/A. The feature is entirely node-local.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

N/A.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

N/A.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

N/A.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

Yes, pod startup latency will be slightly impacted as the Kubelet performs the
additional resource alignment calculations at admission time. However, this is a
one-time cost at pod creation, and the performance benefits for the running
workload are expected to outweigh this initial latency. The
`topology_manager_admission_duration_seconds` metric will be used to monitor
this.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No. The additional state managed by the Kubelet is minimal and should not result
in a non-negligible increase in resource usage.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

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

-   **v1.35**: Target for initial alpha implementation.

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

The primary drawback is the introduction of additional complexity into the
Kubelet's resource management logic. The new partitioning model for the `pod`
scope is more complex than the existing per-container allocation. This increases
the surface area for potential bugs and requires more extensive testing.
Additionally, the different behaviors of the `pod` and `container` scopes when
handling pod-level resources could be a source of confusion for users if not
documented clearly.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

The primary alternative considered was a simpler model for the `pod` scope where
the entire pod-level resource allocation would be assigned to every container in
the pod. This was rejected because it does not support the critical use case of
having mixed-criticality containers within the same pod (e.g., a primary
application with guaranteed resources and a sidecar with shared resources). The
chosen partitioning model provides much greater flexibility and more efficient
resource utilization.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

N/A.
