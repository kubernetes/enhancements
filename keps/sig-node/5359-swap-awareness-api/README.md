# KEP-5359: Swap Awareness API

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Use Case 1: Swap-Disabled](#use-case-1-swap-disabled)
    - [Use Case 2: Explicit limits on swap usage](#use-case-2-explicit-limits-on-swap-usage)
    - [Use Case 3: Swap in Guaranteed pods](#use-case-3-swap-in-guaranteed-pods)
  - [Notes / Constraints / Caveats](#notes--constraints--caveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Node Configuration](#node-configuration)
  - [Proposed Design: Limits-Only Model](#proposed-design-limits-only-model)
  - [Swap limit semantics](#swap-limit-semantics)
  - [NodeInfo Exposure](#nodeinfo-exposure)
  - [User Experience Examples](#user-experience-examples)
    - [Use Case 1: Swap-Disabled Workload](#use-case-1-swap-disabled-workload)
    - [Use Case 2: Swap-Enabled Workload](#use-case-2-swap-enabled-workload)
    - [Use Case 3: Unlimited Swap](#use-case-3-unlimited-swap)
- [Test Plan](#test-plan)
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
  - [Using Dynamic Resource Allocation](#using-dynamic-resource-allocation)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Summary

This KEP proposes a new API to give users control over how much swap a
container can use. The current swap behavior in Kubernetes is implicit, which can
lead to under-utilization of swap provisioned on a node. Explicit API
control for swap enables Kubernetes users to directly manage swap for their
workloads, eliminating assumptions about their requirements. This proposal also
removes existing swap restrictions for features like In-Place
Pod Resize and for Guaranteed pods (e.g. those with CPU pinning), allowing these
workloads to benefit from swap. This KEP introduces a
"WorkloadControlledSwap" mode where swap usage is explicitly
defined by the user for each container. This allows for better resource management
and safer overcommitment of swap resources on a node.

## Motivation

This KEP aims to give Kubernetes workloads greater control over swap usage,
addressing limitations of the current "LimitedSwap" mode. This allows the
application owners to disable or provision a larger swap space for their
containers as best fitting their needs, enhancing swap management for these
applications. Enabling workloads to define swap limits promotes safer, more
efficient swap usage, balancing performance, cost and OOM protection.  

### Goals

To effectively manage swap utilization in workloads, the primary goals of this
KEP are to

-  provide an API that allows application owners to specify the degree of
    swap an application can use.
-  offer the ability to disable swap entirely for a container by setting
    `swap.limit=0`.
-  enable workloads to declare the maximum _acceptable_ swap limits for
    their containers.
-  enable users to configure swap for containers of any QoS class (including
    `Guaranteed` and `BestEffort`), removing QoS-based restrictions on swap
    while maintaining the safe default of swap being disabled.
-  allow safely overcommit on swap to fully leverage available node capacity.
-  facilitate kubernetes node features like in-place pod resize and CPU pinning on
    swap enabled nodes by eliminating implicit swap assumptions on pods.

### Non Goals

-  define new swap scheduling behavior for workloads; this is managed by a
    separate KEP for placement control
-  change eviction behavior for swap enabled nodes; this will be
    investigated with a separate future KEP if improvements are needed. 

## Proposal

This proposal introduces a new `swapBehavior` mode in the `kubeletConfiguration`
called `WorkloadControlledSwap`. When this mode is enabled on a node, swap usage
is no longer implicitly calculated (as in `LimitedSwap` mode) but is instead
explicitly defined by the user on a per-container basis.

This is achieved by introducing a new `swap` resource field under
`resources.limits` for a container. This "limits-only" model allows users to
specify the maximum amount of swap a container can use. If this limit is not
specified, the container will not be allowed to use swap, providing a safe
default.

This explicit per-container limit allows for:

1. Disabling swap for specific containers by setting `swap: "0"`
1. Granting specific swap allowances to containers that can benefit from it
    eg: `swap: "1Gi"`
1. Enabling swap for QoS classes that were previously incompatible, like
    Guaranteed pods, because the user intent is now explicit.
1. Removing restrictions on In-Place Pod Resize feature on swap-enabled
    nodes, as resize on memory limits no longer has any side-effects on swap.
1. Safer overcommitment of swap on a node as the control is granular.

### User Stories 

#### Use Case 1: Swap-Disabled

 A user wants to run a workload that should never use swap.

#### Use Case 2: Explicit limits on swap usage

Modern applications with multiple containers often have varying swap
requirements. eg: a log uploader might have more swap tolerance than a main
web-server. 

#### Use Case 3: Swap in Guaranteed pods

A user has a Guaranteed pod (with CPU pinning) that runs a memory-intensive
process. They want to allow this pod to use a small, fixed amount of swap as a
safety net against OOM kills, which was previously not possible.

### Notes / Constraints / Caveats

1. **Why is swap not an allocatable resource?**

Swap is not modeled as a conventional / allocatable resource as swap is only
consumed when memory pressure occurs. If swap space were 'accounted for' without
being actively used, it could result in scenarios where swap is reserved
unnecessarily, leading to underutilization of other available resources. If
there are use-cases for `resources.swap` rise in the future it could be
discussed.

1. **The "swap:0" placement problem**

A key question is whether `swap: "0"` controls placement or just usage. This
proposal adopts the position that limits control usage, not placement.

-  The swap limits are managed at container level and placements are
    determined at pod level. A "swap:0" container can be co-existing with
    another workload utilizing swap.
-  If workload separation for swap is desired, explicit placement controls
    like taints or nodeSelector should be the preferred option, separating API
    concerns of workload placement from resource usage.
-  ‘limits' should not overload the meaning of "swap:0" to mean "I require a
    non-swap node". Swap aware scheduling is investigated as a separate KEP
    (xref: [#5424](https://github.com/kubernetes/enhancements/issues/5424)). 

### Risks and Mitigations

<<[UNRESOLVED kannon@]>>

1. **Risk: Discuss the implications of overcommitting swap further**

-  what k8s should do to make sure node doesn't end up in a place where all
    the pods have swap provisioned but cannot utilize anymore.

> This can be addressed by the user responding with configuring an
additional swap. K8s cannot react to swap is full as swap is a node
resource; with better observability story this concerns could be reduced.  
  

-  better observability story; user should be able to know when there is
    overcommit of swap or there will be swap capacity crunch.

> We already have swap metrics for capacity and usage at node, pod and
container level
> - Would a new metric for `kubelet_node_swap_allocated_bytes `address this
concern?
> - This would be sum of all `resources.limits.swap` for all containers
running on that node. This can help operators to create precise alert for their
risk for ( allocated / capacity ).  
> - New Condition for `SwapPressure` in NPD.

<<[/UNRESOLVED]>>

1. Risk: User confusion between `LimitedSwap` and `WorkloadControlledSwap`
    modes.

Mitigation: Swap behavior will be exposed as a field in node-info to be
observable by the user.
## Design Details

### Node Configuration

A new `swapBehavior` is introduced in the `kubeletConfiguration`

```
kubeletConfiguration:
  memorySwap:
    swapBehavior: "WorkloadControlledSwap" # Node-level swap enabled, but workloads control usage
```
### Proposed Design: Limits-Only Model

Swap limits are configured per container for a cleaner resource model. This
avoids the ambiguity of swap requests. To enforce this, API-level validation
will be added to forbid non-zero values for `requests.swap`.

-  **Rationale:** "policy" fits per pod, swap "limits" are container specific
    as swap is treated by kernel per process. Starting with ‘container' limits
    first gives us flexibility for unambiguous design. If we start with pod
    limits first, this implies all containers and we will need to reconsider
    how to support individual container limits in the future. (eg: will it
    override?)
-  This also avoids handling conflicts with current `PodLevelResource`
    behavior of applying limit as request and using for admission time.   

```yaml
resources:
  limits:
    memory: "2Gi"
    swap: "1Gi"    # Maximum swap this container can use
  requests:
    memory: "1Gi"
    # No swap ‘requests' as this doesn't make sense
```

### Swap limit semantics

The default behavior for all pods in "WorkloadControlledSwap" mode is "No swap"
(`swap=0)`.

<table>
  <thead>
    <tr>
      <th><em>mode</em>:<br>
<br>
<em>workload behavior</em>:</th>
      <th>NoSwap</th>
      <th>LimitedSwap</th>
      <th>WorkloadControlledSwap</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>No explicit swap limit -  Burstable QoS</td>
      <td>will not swap</td>
      <td>swap as per calculated limit</td>
      <td>will not swap (default)</td>
    </tr>
    <tr>
      <td>No explicit swap limit - Guaranteed/ BestEffort</td>
      <td>will not swap</td>
      <td>will not swap</td>
      <td>will not swap (default)</td>
    </tr>
    <tr>
      <td><code>swap.limit</code> set</td>
      <td>will not swap (No effect)</td>
      <td>swap as per calculated limit (user limit will have no effect)</td>
      <td>maximum swap as per user request. </td>
    </tr>
    <tr>
      <td><code>swap.limit=0</code> (disable)</td>
      <td>will not swap</td>
      <td>swap as per calc limit for Burstable</td>
      <td>will not swap</td>
    </tr>
  </tbody>
</table>

**Note on user experience:** If a pod with `resources.limits.swap` set is
scheduled on a node where the kubelet is configured with `NoSwap` or
`LimitedSwap`, the pod will be admitted, but a Pod event will be generated to
indicate that the node does not support the requested swap configuration. The
container will run, but the specified swap limit will have no effect. This
approach avoids disrupting pods that have already been scheduled. For explicit
placement, users should use node labels and selectors to ensure pods are
scheduled on nodes with the appropriate `swapBehavior`.

**Note on placement:** For cases, when explicit limit is set, the
`NodeDeclaredFeatures` KEP is an option to explore for implicit scheduling
control – a separate Swap Scheduling KEP is exploring this further. Explicit
limits show clear user intent for these workloads to work with nodes in
`WorkloadControlledSwap` mode, and the scheduler could place them right. This
also can protect the conflicting case by not placing explicitly disabled swap
workloads on a `LimitedSwap` node.

**Note on coexistence:**  Kubernetes cannot support ‘built-in' protection when users
want to have some nodes in `LimitedSwap` and some nodes in
`WorkloadControlledSwap` within a cluster. This placement control can be
achieved with taints or label selectors. NFD (Node Feature Discovery) is seen as
the path to work with swap-labels, which will help with grouping swap nodes for
maintenance or migration. Existing workloads in `LimitedSwap` will continue to
work to protect existing behavior of swap enabled nodes. 

<<[UNRESOLVED skanzhelev@]>>  
When do we even need LimitedSwap? As WorkloadControlledSwap is more powerful,
why do we need limited mode? Should we have these as a different mode or have
these as implicit behavior of swap design?

With LimitedSwap already swap getting adoption in production by many users,
overriding to change behavior may not be preferred, WorkloadControlledSwap would
enable the additional usecases and can coexist with current behavior.   
<<[/UNRESOLVED]>>

### NodeInfo Exposure

Swap behavior will be exposed in the `Node` status to enable monitoring and
selection: 

```
nodeInfo:
  ...
  swap:
    behavior: WorkloadControlledSwap
    capacity: 53687087104
```

This will enable field selection for monitoring:

### User Experience Examples

#### Use Case 1: Swap-Disabled Workload

Disabling swap can be achieved by setting `swap: "0"`. The `nodeSelector` is
used for explicit placement preference with NFD.

```yaml
# I don't want swap, prefer non-swap nodes
spec:
  nodeSelector:
    feature.node.kubernetes.io/memory-swap: "false"
  containers:
  - resources:
      limits:
        memory: "2Gi"
        swap: "0"
```

#### Use Case 2: Swap-Enabled Workload

```yaml
# I want swap capability, place only in a swap-enabled node with LimitedSwap
spec:
  nodeSelector:
    feature.node.kubernetes.io/memory-swap: "true"
    feature.node.kubernetes.io/memory-swap.behavior: LimitedSwap
  containers:
  - resources:
      limits:
        memory: "2Gi"
        swap: "1Gi"
```

#### Use Case 3: Unlimited Swap

```yaml
# I want as much swap as the node allows
spec:
  nodeSelector:
    feature.node.kubernetes.io/memory-swap: "false"
    feature.node.kubernetes.io/memory-swap.behavior: WorkloadControlledSwap
  containers:
  - resources:
      limits:
        memory: "2Gi"
        swap: "8Gi"    # Large limit = effectively unlimited
```

## Test Plan

1.  I/we understand the owners of the involved components may require
    updates to existing tests to make this code solid enough prior to
    committing the changes necessary to implement this enhancement.

**Unit Tests**

- `k8s.io/apis/core`
- `k8s.io/apis/core/v1/validations`
- `k8s.io/features`
- `k8s.io/kubelet`
- `k8s.io/kubelet/container`

**Integration Tests**

Unit and E2E tests provide sufficient coverage for the feature. Integration
tests may be added to cover any gaps that are discovered in the future.

**e2e tests**
    -  Verify pod with explicit swap on `WorkloadControlledSwap` node uses swap.
    -  Verify pod with no limit on `WorkloadControlledSwap` node does not use swap.
    -  Verify pod with `swap:"0"` on `WorkloadControlledSwap` node does not use swap.
    -  Verify that a Guaranteed pod with explicit swap set on `WorkloadControlledSwap`
        node uses swap.

## Graduation Criteria

### Alpha

-  Feature implemented behind a feature flag `WorkloadControlledSwap`
-  Initial e2e tests completed and enabled.
-  Public documentation on workload controlled swap is updated.

### Beta

-  API controlled swap functionality is running behind feature flag for at least one release.
-  No major bugs reported and user feedback is positive.

### GA

-  No major bugs reported for three months.

## Upgrade / Downgrade Strategy

API server should be upgraded before Kubelets. Kubelets should be downgraded
before the API server.

## Version Skew Strategy

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
-->

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `WorkloadControlledSwap`
  - Components depending on the feature gate: kubelet, kube-apiserver

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->
Yes. KEP introduces safe default with WorkloadControlledSwap - if explicitly specified use the limits for swap, otherwise set it as 0 (no swap). To ensure backward compatibility, this change will be a new node behavior, so existing users who are working with the LimitedSwap swap behavior will not be impacted. The api set limits are not applicable in LimitedSwap configured nodes.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes. To roll back, the feature gate should be disabled in the API server and
kubelets, and components should be restarted. If a Pod was created with a
`resources.limits.swap` field while the gate was enabled, those will be ignored by
kubelets once the feature is disabled.

###### What happens if we reenable the feature if it was previously rolled back?

If the feature is re-enabled, the kubelet will once again recognize and enforce
the swap limits for any Pods that have the field defined.

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

- Unit test for the API's validation with the feature enabled and disabled.
- Unit test for the kubelet with the feature enabled and disabled.
- Unit test for API on the new field. First enable the feature gate, create a Pod with a container including `resources.limits.swap` field, validation should pass and the Pod API should match the expected result. Second, disable the feature gate, validate the Pod API should still pass and it should match the expected result. Lastly, re-enable the feature gate, validate the Pod API should pass and it should match the expected result.

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

If this feature is being actively used in a cluster that has this feature
partially enabled on some nodes, pods on nodes with WorkloadControlledSwap
enabled may configure different swap limits than pods on nodes without this 
feature.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

Swap is not configured on the workload even when limits are specified.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

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

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

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

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

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

No.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

Enabling this feature will introduce a new field `resources.limits.swap` to the [Container](https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/core/types.go#L2601) API spec.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

This feature adds a new key-value pair to the resources.limits map within the [v1.Container](https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/core/types.go#L2601) spec for each container that specifies a swap limit. Key: "swap" (4 bytes) and Value: a string like "1Gi" (3 bytes) or "500Mi" (5 bytes). The total increase per container could be 10-15 bytes per container.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

When the workload is configured with swap and node is under memory pressure, swap utilization may result in increased CPU and I/O usage to offload memory (RAM) to disk.


###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

Enabling this feature will add swap utilization for the workload and can result in resource exhaustion of 'swap resource' if swap is overcommitted.

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

###### What steps should be taken if SLOs are not being met to determine the problem?

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

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

### Using Dynamic Resource Allocation

Another possible way to realize this KEP is to leverage Dynamic Resource Allocation (DRA) framework to manage swap. In this model, "swap" could be defined as a ResourceClass, and pods would use a ResourceClaim to request a specific swap limit. DRA requires a full ecosystem of CRDs, a node-level driver, and Kubelet plugins. This is massive overhead for what is ultimately setting a single cgroup value (`memory.swap.max`). The simplicity of the `resources.limits` approach is preferable over the complex DRA approach.


## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->


