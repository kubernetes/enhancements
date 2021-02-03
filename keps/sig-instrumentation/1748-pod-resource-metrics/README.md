<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up.  KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please ensure to complete all
  fields in that template.  One of the fields asks for a link to the KEP.  You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "title", "authors", "owning-sig",
  "status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary", and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG that are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly.  The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved.  Any KEP
marked as a `provisional` is a working document and subject to change.  You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused.  If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement", for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example.  If there are
new details that belong in the KEP, edit the KEP.  Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable` or significant changes once
it is marked `implementable` must be approved by each of the KEP approvers.
If any of those approvers is no longer appropriate than changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross cutting KEPs).
-->
# KEP-1748: Expose Pod Resource Request Metrics

<!--
This is the title of your KEP.  Keep it short, simple, and descriptive.  A good
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
  - [Report a new multi-dimension pod_resources metric](#report-a-new-multi-dimension-pod_resources-metric)
  - [Describe the pod resource model](#describe-the-pod-resource-model)
    - [The Kubernetes resource model](#the-kubernetes-resource-model)
  - [User Stories (optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Cardinality growth of metrics](#cardinality-growth-of-metrics)
- [Design Details](#design-details)
  - [Expose new metrics](#expose-new-metrics)
  - [Add recording rules consistent with this metric to describe actual resource usage](#add-recording-rules-consistent-with-this-metric-to-describe-actual-resource-usage)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

- [x] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] KEP approvers have approved the KEP status as `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The current Prometheus metrics exposed by the cluster have gaps that make building accurate capacity alerts, dashboards, and ad-hoc queries more difficult than necessary and complicate the average user’s understanding of the resource model. The Kubernetes resource model is fundamental to capacity planning and error triage. It should be easy to visualize and alert on core usage and capacity through a simple set of metrics. Administrators and integrators should be able to easily graph and calculate the available and consumed resources within the cluster at any time.


## Motivation

<!--
This section is for explicitly listing the motivation, goals and non-goals of
this KEP.  Describe why the change is important and the benefits to users.  The
motivation section can optionally provide links to [experience reports][] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### Goals

Kubernetes should define a set of standard metrics that allow a user who works with the Kubernetes resource model to easily answer via instrumentation several important queries:

* What is the current remaining schedulable capacity of all my nodes
* Which components are using less than they request
* Which components on a node are the most likely to be evicted
* Which resources on my cluster are most contended
* Is a set of pods requesting more resources than any node has to offer

We should attempt to provide sufficient “out of the box” metrics for capacity planning for most cluster administrators, and present the resource model that Kubernetes uses for easy consumption. Integrations that extend the resource model with our supported pattern should also be visible in these metrics as the default resource model would interpret them.

We should use consistent terminology, patterns, and descriptions of resource request, consumption, and availability across our API, metrics, component code, and documentation to ensure Kubernetes admins see a coherent view of the Kubernetes resource model.

### Non-Goals

* Full coverage all fields on pods in metrics
* Container level representations of these metrics
* Changes to the pod lifecycle API definition

## Proposal

### Report a new multi-dimension pod_resources metric

A component with clear alignment to our resource consumption model should report a standard metric for pod consumption that takes into account lifecycle, resource type, units, and is consistent with the existing node_resources metric exposed by KSM.  This metric should be a pod level synthesis (resource decisions are pod level) of the same fundamental calculations the scheduler or kubelet make. It is recommended the scheduler expose this since it already has the relevant data, and custom schedulers may wish to expose similar metrics for their subset of usage. While the metrics are expected to cost significantly less than the set of metrics exposed by kube-state-metrics today for similar purposes (cardinality likely an order of magnitude lower), these metrics should be opt-in on the scheduler for those who have alternate or pre-existing resource metrics. Because scheduling is done at a pod level, this proposals eschews container level granularity which is available via the pod API if needed.

### Describe the pod resource model

The Kubernetes resource model on a pod is described in a number of places - this section updates the definition to use consistent terminology and serve as a full overview. If subsequent changes occur this definition will be updated and continue to be suitable for use in documentation.

#### The Kubernetes resource model

In Kubernetes components perform calculations on the requested resources of a pod to determine whether a pod fits within the limits of a node or namespace resource quota.  The scheduler is responsible for placing a pod onto a node, the kubelet is responsible for consistently enforcing only pods that fit within its allowed requests are admitted, and the quota subsystem tracks and rejects creations or updates that exceed fixed limits by performing consistent calculations. At the [current time resource reservations are immutable](../../sig-node/20181106-in-place-update-of-pod-resources.md) once the pod is created.

The reservation of a pod for a given resource is

    max(max_over_init_containers(resource), sum_over_containers(resource)) + resource_overhead

The reservation of a pod is the union of all resource calculations of this form. Resources are considered individually when making decisions about admission or rejection from nodes or quota, and any one resource being exceeded will prevent execution or admission.

For instance, the pod:

```yaml
metadata:
  name: nginx
spec:
  initContainers:
  - name: copy-files
    ...
    resources:
      limits:
        cpu: 100m
  - name: generate-config
    ...
    resources:
      requests:
        cpu: 300m
  containers:
  - name: proxy
    ...
    resources:
      requests:
        cpu: 50m
  - name: sidecar-logging
    ...
    resources:
      requests:
        cpu: 100m
```

will be handled by the system in the following ways:

* Is considered to consume `300m` cores of CPU by the scheduler (the node must be able to run the largest init container by itself, which is larger than the sum of the proxy and sidecar containers at runtime)
* The kubelet will not admit the pod if `(the node's allocatable CPU) - (the sum of all runnable pods)` is less than `300m`
* The kubelet will create a cgroup for the pod that expects to get roughly `300m` cores, but the container cgroup created for `copy-files` will allow `100m` while that init container is running (without limit set, it would be given `300m` as shares which is the pod default)
* The quota subsystem will block this pod from being created if there is less than `300m` available.

Once a pod is created, it passes through three high level lifecycle states as seen by the total Kubernetes system. These states are defined in terms of an outside observer of the API - because Kubernetes is a distributed system individual components may report status such as the pod phase after the transition has already occurred, and it is important to clarify what assumptions an API observer may make about those states. The naming is chosen to align with the phases reported by the pod, but we clarify the difference between the Kubelet's view of phase and how that outside observer should view that state without changing the meaning of the pod's status fields.

The first state is `Schedulable` - the time before the pod is scheduled to a node (this is denoted by the nodeName field being set, not by the status update made by the Kubelet to the phase). A pod is `Schedulable` after a successful create via the API. The second state is `Runnable` and occurs immediately after `nodeName` is set, when the Kubelet may begin initializing resources on the node and may start or restart process, and continues until the pod is gracefully deleted via the API, reaches a terminal state of success or failure when `restartPolicy` is `Never` or `OnFailure`, or is rejected by the Kubelet and put in the terminal state of failure. The final state is `Completed` which means the pod has no running containers, all resources have been released or cleaned up, and the pod will never again consume those resources. A pod transitions to the `Completed` terminal state when pod status phase has reached Succeeded or Failed (as observed by the Kubelet and then recorded into the API), or the pod is marked for deletion and all containers are reported stopped via status to the API. A pod in the `Completed` state may be referred to as a "terminal pod" or a "terminated pod". A pod that has been marked for termination via the DELETE API call but not yet reached a terminal state may be referred to as a "gracefully deleted pod" (or "deleted pod" for short) or a "terminating pod".

Because Kubernetes is a distributed system we must be aware of how the system views these states, not just individual components. The Kubelet may be arbitrarily delayed reporting pod status phase changes while simultaneously allocating resources. We may observe a `Runnable` pod to have already reached completion but the Kubelet has not yet reported that to the API. Once a pod is scheduled to the node Kubernetes considers the pod the responsibility of that node. Because the Kubernetes model does not require the Kubelet to update the status of the pod before it launches the process (does not require a synchronization point with the API), clients must assume that the processes within the pod may have been started and so we consider the `Schedulable` lifecycle phase at a system level to end the moment a pod is bound to the node and the `Runnable` phase to begin immediately at that time, although pod status may report the pending phase for a potentially unbounded period of time. The transition to the `Completed` state occurs when the Kubelet records the terminal state of the pod in the apiserver by setting the Succeeded or Failed phase in status, or when the pod is marked for deletion and all containers are given a status and termination state. A node retains ownership of the pod within Kubernetes until the pod is fully deleted, although a client may safely assume that any pod in the `Completed` state will never again become `Runnable` or `Schedulable` without being fully deleted and recreated (UID changes).

A `Schedulable` state pod is suitable for scheduling and consumes resources for the purposes of quota. A `Runnable` state pod consumes resources for the purposes of quota and from its node. A `Completed` state pod consumes no resources.

Future features may alter this model, such as the pod resizing proposal, and those rules will be automatically reflected in the resources handled by the system.


### User Stories (optional)

#### Story 1

As an administrator of a Kubernetes cluster, I can easily see the requested resources for the pods on the cluster and compare those to the actual usage of those pods with our instrumentation pipeline.

#### Story 2

As an extender of Kubernetes, my extended resources used by the scheduler should be easily queryable by an administrator without requiring Kubernetes code changes or adding new components to the system.

### Story 3

An administrator of the Kubernetes cluster should be able to see an aggregate representation of actual resource consumption metrics that is consistent with the resource request and scheduling model in order to measure resources that are imprecisely sized.

### Notes/Constraints/Caveats (optional)

The `kube-state-metrics` component of Kubernetes already exposes a lower fidelity model of resource requests from pods. However, because it is focused on exposing the attributes of the pod resource model rather than their calculated meaning, it captures an imprecise representation of the metrics Kubelet and the scheduler use to make decisions. This representation is not suitable for making complete decisions on *why* a resource is not schedulable, or to accurately capture how the scheduler views the resource model. In order to practically model these resources, a number of high cardinality metrics would need to be added and a fairly complicated calculation would have to be performed in the metrics aggregator that would subtly differ from the decisions made by the Kubelet or scheduler. Instead we recommend closely matching the implementation the scheduler uses to make decisions (documented and part of our API, but not explicitly modelled outside of code) to represent the key metrics for the resource model at significantly reduced overall cardinality. For this reason in kube-state-metrics the resource metrics might be deprecated and removed at some point in the future, but not tied to this implementation.

### Risks and Mitigations

#### Cardinality growth of metrics

The proposal will add O(pods*resource_types) series. Currently a larger number of series are exposed via `kube-state-metrics` to help summarize to this data, but this proposal should reduce the need for those metrics and lead to a cardinality reduction in the future. This proposal grows the number of per pod series by a small fraction of the existing amount, described in the [kube-state-metrics project docs](https://github.com/kubernetes/kube-state-metrics/blob/master/docs/pod-metrics.md). In extremely large clusters an administrator may wish to not scrape or gather the metrics described here using the described flag.

<!--
What are the risks of this proposal and how do we mitigate.  Think broadly.
For example, consider both security and how this will impact the larger
kubernetes ecosystem.

How will security be reviewed and by whom?

How will UX be reviewed and by whom?

Consider including folks that also work outside the SIG or subproject.
-->

## Design Details

### Expose new metrics

The scheduler will expose an optional set of metrics series that capture the resources a pod is considered to be "consuming" from a scheduling perspective at the current time. The metrics will be exposed via `/metrics/resources` to allow aggregation to be optional and to allow a separate scrape interval. Only the active scheduler will report metrics series in order to remove the need to downsample the series and to reduce overall cardinality. If no scheduler is currently leading, then these metrics will temporarily not be reported which is consistent with metrics based on populated caches in leader elected components. Instrumentation for Kubernetes should scrape this endpoint for the default scheduler to satisfy the objectives around capacity management - non-default schedulers are free to provide additional metrics that complement this data but are not required to duplicate the reporting or retrieval of this info.

The scheduler was chosen as the primary owner of the mapping of pods to requested resources and because it already manages an internal view of pod resources that matches the external idealized view of pod resources. We also expect that the scheduler will have configuration (gates, tuning, customizations) that would have practical effects on the representation of the resource model and that aligning these metrics with the scheduler will minimize configuration mismatches or vendor drift when things like pod overhead, mutable resource requests, forked schedulers, or other points of deviation arise. While other components could certainly host these metrics, it would be as or more expensive (due to the duplication of the pod cache and maintenance of the vendoring logic for shared resource model code) and would not offer a meaningful benefit to an administrator of Kubernetes. The metrics described in this proposal apply to the idealized view of pods that transition from pending (schedulable) to running (scheduled) to completed (terminal), all of which are critical to the functioning of the scheduler. The metrics require no integration into the internal state of the scheduler or impose any additional requirements on the scheduler except for access to the pod informer and the same fields on pods the scheduler uses - future efficiency gains on the pod informer could be translated to the metrics implementation if necessary.

The metrics will be implemented via the collector pattern - when a scrape is requested the pod cache will be traversed and each metric will be calculated and reported in a streaming fashion, without caching or having to acquire locks. This is the standard pattern within Kubernetes for reporting high cardinality metrics and generally allows us to discount the cost of scrapes (estimate: one scrape every 15s might add 0.1 core of use for 50k pods, and we can control scape interval and reduce cost). The approach will add no additional memory usage overhead except for the transient allocations during iteration of the pod cache. Note that in other components we use this pattern to gather one or two orders of magnitude more metrics proportional to pods, so from a whole system perspective the impact is negligible and should have no impact on the scheduler performance.

The series reported will be consistent with `kube_node_status_allocatable` to simplify queries that compare used capacity and available capacity. After a pod has reached the Completed lifecycle (as described above) state, the series will not be reported. If the value of the resource is zero, that resource will not report a series. They all share the form `kube_pod_resource_(requests|limits)`.

Note on naming: The Prometheus convention for a metric is to allow the sum of a metric to make sense. However, in the case of Kubernetes resources we are attempting to make discoverability of the reported metrics (which are unbounded in number as they are extensible) a key goal for admins. Most admins will understand CPU and memory usage, but would have to resort to metrics name wildcard regexes to find the other values if we give each resource its own metrics name. Also, because Kube extended resources have characters disallowed in metrics names, admins would continually have to transform `myresource.io/foo-bar` into `myresource_io_foo_bar` in order to locate names which places an undue burden on the user. So for the proposed metrics and `kube_node_status_allocatable` we instead place all types resources under the same metrics name with a unit and resource label.  For this metric, the sum of series is only meaningful when filtering on a resource label.

* `kube_pod_resource_requests` contains series for the consumed requested resources for a pod at a given time.
  * The value is the quantity of the request in `spec.containers.resources.requests`.
  * It has the following labels:
    * `pod` - the name of the pod
    * `namespace` - the namespace of the pod
    * `node` - the node the resource is scheduled to, or empty if not yet scheduled.
    * `scheduler_name` - the name of the scheduler in `spec.schedulerName`
    * `priority` - the priority value assigned to a pod in `spec.priority`
    * `resource` - the name of the resource as described in `spec.containers.resources.requests`
    * `unit` - the units of the resource type as inferred from the resource name in `spec.containers.requests`, such as `bytes`, `cores`. Empty if this is a unitless resource or if it is an extension resource that does not have a detectable unit.
* `kube_pod_resource_limits` contains series for the consumed limit resources for a pod at a given time.
  * The value is the quantity of the request in `spec.containers.resources.limits`.
  * It has the following labels:
    * `pod` - the name of the pod
    * `namespace` - the namespace of the pod
    * `node` - the node the resource is scheduled to, or empty if not yet scheduled
    * `scheduler_name` - the name of the scheduler in `spec.schedulerName`
    * `priority` - the priority value assigned to a pod in `spec.priority`
    * `resource` - the name of the resource as described in `spec.containers.resources.limits`
    * `unit` - the units of the resource type as inferred from the resource name in `spec.containers.limits`, such as `bytes`, `cores`. Empty if this is a unitless resource.

This will add `O(pods * resource types)` metrics series, and in general most clusters have 4-8 resource types. The series per pod is `non_zero_resource_count * 2 (requests and limits)`.  This explicitly does not expose containers as a dimension, as the resource model applies rules to containers that cannot be trivially expressed via a formula and does not help answer the core questions.

The `node`, `scheduler_name`, and `priority` labels allow breakdowns of how capacity as divided among schedulers, classes, and consumers of resources. `node` is a one way transition from empty to a node name that divides the `Schedulable` and `Runnable` phases of the overall pod lifecycle and is critical for understanding queued capacity vs allocated capacity.  These labels add only a small amount of overhead and no extra cardinality.

These metrics should roughly correspond to:

* The resources the scheduler will consider the pod to be consuming at the current time
* The resources the kubelet will use to decide whether to admit the pod (nodes will reject requested resources that exceed their actual allocatable capacity)
* The effective reserved capacity as resource quota would observe for the namespace
* The sum of all resources consumed on the cluster
* The size of cgroup constraints applied to pods on a node

The metrics would be served by the scheduler that is holding the active scheduler lease. During a failover, no metrics would be reported. This would prevent duplicate metrics from being scraped and aligns with our decision in the scheduler to not fill caches prior to election for resource usage minimization.

Custom schedulers based on the scheduler code should be able to easily reuse and report these metrics, and the configuration of the cluster instrumentation to retrieve those metrics would be left as an exercise for the integrator.


### Add recording rules consistent with this metric to describe actual resource usage

In order to make these metrics useful for calculation of actual usage, we would represent the current usage in a form that can easily be queried by applying recording rules in the standard Prometheus stack and equivalents in other systems without having to apply complex calculations.

This section is normative and will be incrementally approached during alpha - not all resource metrics must appear in this format.

* `kube_running_pod_resource_usage` - the resources currently consumed by an initializing or running (shorthanded to running) pod.
  * We use "running" because these resources are measured at the node which is authoratative for the state of pod processes (pods can be Runnable but not Running, but never Running but not Runnable)
  * The value would be the unit of consumption for that pod on that node
  * It should report the following labels to the user:
    * `pod` - the name of the pod
    * `namespace` - the namespace of the pod
    * `node` - the node the resource is scheduled to
    * `resource` - the name of the resource as it would be described in the pod's `spec.containers.resources.requests`
    * `unit` - the unit type of the resource type as inferred from the resource name in `spec.containers.limits`, such as `bytes`, `cores`. Empty if this is a unitless resource and no such calculation can be made.

Specific resource implications:

* `cpu` would be reported as the rate of usage over a reasonable window based on the scrape period of the kubelet - no window is perfect, but the summation is valuable to allow requests to be compared like to like.
* `memory` would be reported as the working set bytes since that is the most accurate assessment of the pods usage and will in the majority of scenarios capture what the OOM killer will target. There may be some confusion because this metric may temporarily exceed total limit, but RSS often significantly underestimates the real memory workload usage.

We use the generic metric form instead of a specific name containing the resource so that administrators can easily discover resources under the same metric.

For alpha we would target cpu and memory to gain familiarity with this approach.

### Test Plan

The metrics implementation for lifecycle will be tested with unit tests to verify the correct transformation of pod -> metric for the relevant metrics. An integration test will verify that the active scheduler exposes the metrics on clusters where the scheduler metrics port is reachable. An e2e may be added later for clusters that expose the scheduler HTTP endpoint (which is both optional and may not be available in some configurations).

### Graduation Criteria

The metrics exposed at the new endpoint will be listed as alpha stability in 1.20, and may be changed or removed in subsequent releases. Default installations of Kubernetes should not utilize these metrics except for testing and feedback as they may change without notice

#### Alpha -> Beta Graduation

- Feedback from administrators utilizing these metrics
- Consumption-focused dashboards have been prototyped that demonstrate the metrics satisfy the questions described above
- Reach decision on whether `kube-state-metrics` representations of pod resources at container level will be deprecated

#### Beta -> GA Graduation

- Stability over two releases demonstrating cardinality is reasonable and the metrics remain valuable

### Upgrade / Downgrade Strategy

As a new metrics endpoint, components can opt-in to consuming these metrics and they are listed as being at alpha level of stability. Future upgrade rules will follow metrics stability requirements.

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade in order to make use of the enhancement?
-->

### Version Skew Strategy

Version skew will not be an issue as these metrics are reported on stable v1 API objects.

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->


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

* **How can this feature be enabled / disabled in a live cluster?**
  - [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name:
    - Components depending on the feature gate:
  - [x] Other
    - Describe the mechanism: A metrics collector may scrape the `/metrics/resources` endpoint of all schedulers, as long as the scheduler exposes metrics of the required stability level.
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**

Scraping these metrics does not change behavior of the system.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**

Yes, in order of increasing effort or impact to other areas:

* Administrators may stop scraping the endpoint, which will mean the metrics are not available and any impacted caused by scraping will stop.
* The administrator may change the RBAC permissions on the delegated auth for the metrics endpoint to deny access to clients if a client is excessively targeting metrics and cannot be stopped.
* The administrator may change the HTTP server arguments on the scheduler to disable information about the scheduler via the `--port` arguments, but doing so may require other changes to scheduler configuration as this will disable health checks and standard metrics.

* **What happens if we reenable the feature if it was previously rolled back?**

Metrics will start getting collected.

* **Are there any tests for feature enablement/disablement?**

As an opt-in metrics endpoint enablement is tested from our integration tests.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**

This cannot impact running workloads unless an unlikely performance issue is triggered due to
excessive scraping of the scheduler metrics endpoints (which is already possible today).

Since the new metrics are proportionally less than the metrics an apiserver or node exposes,
it is unlikely that scraping this endpoint would break a metrics collector.

* **What specific metrics should inform a rollback?**

Excessive CPU use from the Kube scheduler when metrics are scraped at a reasonable rate,
although simply disabling optional scraping while waiting for the bug to be fixed would be
a more reasonable path.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

Does not apply.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**

No.

### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?**

This would be up to the metrics collector component whose API is not under the
scope of the Kubernetes project. Some third party software may use these metrics
as part of a control loop or visualization, but that is entirely up to the metrics
collector.

Administrators and visualization tools are the primary target of these metrics and
so polling and canvassing of Kube distributions is one source of feedback.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [x] Other (treat as last resort)
    - Details: Covered by existing scheduler SLIs (health check, CPU use, pod scheduling rate, http request counts).

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

The existing scheduler SLOs should be sufficient and this change should have no measurable impact on the existing SLO.

The metrics endpoint should consume a tiny fraction of the CPU of the scheduler (less than 5% at idle) when scraped
every 15s. The endpoint should return quickly (tens of milliseconds at a P99) when O(pods) is below 10,000. CPU and
latency should be proportional to number of pods only, as the rest of the scheduler, and the metrics endpoint should
scale linearly to that factor.

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**

No

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**

  - Scheduler
    - Hosts the metrics
  - Metrics collector
    - Scrapes the endpoint
    - May run on or off clutser


### Scalability

* **Will enabling / using this feature result in any new API calls?**

No, this pulls directly from the scheduler's informer cache.

* **Will enabling / using this feature result in introducing new API types?**

No.

* **Will enabling / using this feature result in any new calls to the cloud
provider?**

No.

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**

No.

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**

The CPU usage of this feature when activated should have a negligible effect on
scheduler throughput and latency. No additional memory usage is expected.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**

Negligible CPU use is expected and some increase in network transmit when the scheduler
is scraped.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

* **How does this feature react if the API server and/or etcd is unavailable?**

It returns the metrics of the last set of data received by the scheduler, or no
metrics if the scheduler has been restarted since partitioned from the API server.

* **What are other known failure modes?**

  - Panic due to unexpected code path or incomplete API objects returned in watch
    - Detection: The scrape of the component should fail
    - Mitigations: Stop scraping the endpoint
    - Diagnostics: Panic messages in the scheduler logs
    - Testing: We do not inject fake panics because the behavior of metrics endpoints are well known and there is no background processing.

* **What steps should be taken if SLOs are not being met to determine the problem?**

Perform a golang CPU profile of the scheduler and assess the percentage of CPU charged to the functions
that generate the CPU metrics. If they exceed 5% of total usage, identify which methods are hotspots.
Look for unexpected allocations via a heap profile (the metrics endpoint should not generate much if any
allocations onto the heap).


## Implementation History

* 2020/04/07 - [Prototyped](https://github.com/openshift/openshift-controller-manager/pull/90) in OpenShift after receiving feedback that resource metrics were opaque and difficult to alert on
* 2020/04/21 - Discussed in sig-instrumentation and decided to move forward as KEP
* 2020/07/30 - KEP draft
* 2020/11/12 - Merged implementation https://github.com/kubernetes/kubernetes/pull/94866 for 1.20 Alpha

<!--
Major milestones in the life cycle of a KEP should be tracked in this section.
Major milestones might include
- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

This has the potential to increase the cardinality of metrics gathered by the system by a factor proportional to the number of pods. However, we already have a large number of metrics proportional to pods, and the value of these metrics is deemed to outweigh the cost to add them.


## Alternatives

We considered extending `kube-state-metrics` to represent all of the data necessary to perform the pod resource lifecycle calculation as described above. The implementation would require a significant number of new metric cardinality, and would also require a very complex recording rule that would be difficult in some engines and likely not be completely accurate. Since we believe that metrics should be exposed that reflect the intent of the system, and this metric is extremely high value, and we desire to reuse the same logic that the underlying system relies on to minimize drift, we choose to bypass this approach for now.
