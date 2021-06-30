# KEP-2185: Random Pod Selection on ReplicaSet Downscale

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
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
  - [Make downscale heuristic an option](#make-downscale-heuristic-an-option)
  - [Compare pods using their distribution in the failure domains](#compare-pods-using-their-distribution-in-the-failure-domains)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The current downscaling algorithm for ReplicaSets prefers to delete Pods that
have been running for the least amount of time. This heuristic attempts to
minimize disruption under the premise that newer Pods are likely to be serving a
lesser amount of clients. However, the heuristic can be detrimental to high
availability requirements: when a ReplicaSet lands in an imbalanced state across
failure domains, the heuristic tends to preserve the imbalance after repeated
up and down scales.

We propose a randomized approach to the downscale Pod victim selection algorithm
of the ReplicaSet controller to mitigate ReplicaSet imbalance across failure
domains.

## Motivation

There are scenarios where a ReplicaSet can reach an imbalanced state across
failure domains. See [user stories](#user-stories) to see one such scenario
and how a randomized approach solves the issue.

### Goals

- A randomized algorithm for Pod selection when downscaling ReplicaSets.
- Softly honor the heuristic that prefers to downscale newer Pods first.
- Validate that the approach is able to get a ReplicaSet out of an imbalanced
  state.

### Non-Goals

- Provide guarantees of preserving balance during scale down. This introduces
  a violation of separation of concerns between the ReplicaSet controller and
  kube-scheduler.
- Preserve the existing behavior that always downscales the newer Pods first.
  This order was never guaranteed in the API or user documentation.

## Proposal


### User Stories

#### Story 1

This story shows an imbalance cycle after a failure domain fails or gets
upgraded.

1. Assume a ReplicaSet has 2N pods evenly distributed across 2 failure domains,
   thus each has N pods.
2. An upgrade happens adding a new available domain and the ReplicaSet is upscaled
   to 3N. The new domain now holds all the youngest pods due to scheduler spreading.
3. ReplicaSet is downscaled to 2N again. Due to the downscaling preference, all
   the Pods from one domain are removed, leading to imbalance. The situation
   doesn't improve with repeated upscale and downscale steps. Instead, a
   randomized approach leaves about 2/3*N nodes in each
   failure domain.

### Notes/Constraints/Caveats

The original heuristic, dowscaling the youngest Pods first, has its benefits.
Newer Pods might not have finished starting up (or warming up) and are likely
to have less active connections than older Pods. However, this distinction
doesn't generally apply once Pods have been running steadily for some time.

A purely randomized approach would break those assumptions, potentially
leading to services disruption. Choosing a heuristic could be left to the user.
On the other hand, certain workloads take a long time to warm up and, at the
same time, require high availability.

### Risks and Mitigations

Certain users might be relaying in the existing downscaling heuristic. However,
there are a number of reasons why we don't need to preserve such behavior as is:

- The behavior is not documented in the API or user docs, only in code.
- The heuristic is applied last, after other criteria have been applied. In
  particular, the heuristic doesn't apply when multiple Pods are in the same
  Node or if a particular Pod has had several container restarts. While users
  can enforce one Pod per Node with certain scheduling features, there is no
  workaround for other criteria.
- We are not proposing to entirely remove the heuristic, just make it more
  laxed.
- We are introducing a [related feature](git.k8s.io/enhancements/keps/sig-apps/2255-pod-cost)
  that provides higher guarantees for downscaling order that users can migrate
  to.
  
  <!--
  If not merged yet, the KEP can be found in https://github.com/kubernetes/enhancements/pull/1828
  -->

## Design Details

We propose a randomized approach to the algorithm for Pod victim selection
during ReplicaSet downscale:

1. Sort ReplicaSet pods by pod UUID. The purpose of this is to obtain a
pseudo-random shuffle of the pods (this also does not necessarily have to
be the first step, it is just another comparison criteria).
2. Obtain wall time, and add it to [`ActivePodsWithRanks`](https://github.com/kubernetes/kubernetes/blob/dc39ab2417bfddcec37be4011131c59921fdbe98/pkg/controller/controller_utils.go#L815)
3. Call sorting algorithm with a modified time comparison for start and
   creation timestamp.


Instead of directly comparing timestamps, the algorithm compares the elapsed
times since the creation and ready timestamps until the current time but in a
logarithmic scale, floor rounded. These serve as sorting criteria.
This has the effect of treating elapsed times as equals when they
have the same scale. That is, Pods that have been running for a few nanoseconds
are equal, but they are different from pods that have been running for a few
seconds or a few days.

For example, let's assume the base 10 is used, then we have the following
mapping for different durations:

| Duration | Scale |
|----------|-------|
| 5ns      | 0     |
| 23ns     | 1     |
| 71ns     | 1     |
| 1ms      | 6     |
| 8ms      | 6     |
| 50ms     | 7     |
| 2m       | 11    |
| 11m      | 11    |

An alternative interpretation for the base 10 is that, if a Pod has been running
for more than 10 times the time as another Pod, then the second Pod would be
deleted first.

While base 10 is quite intuitive, it might be too aggressive on bucketing
timestamps together. A base of 2 could be similarly intuitive and provide a
better bucketing. But if documentation is not a problem, the natural base is
a good choice as well.

### Test Plan

Unit and e2e tests will be helpful to ensure continuing performance of the
intended behavior. However, due to the random nature of downscale selection
within each rounded bucket it will be important to keep in mind the difficulty
in expecting a specific pod to be deleted when multiple could still be valid candidates.
Understanding this while writing tests will significantly reduce flakes.

Specific test cases could include something similar to what is described 
above in the [user stories](#user-stories), where a balanced cluster state 
is created, then downscaled, then upscaled (to rebalance), and finally downscaled 
again. The expectation is that after the final downscale, the nodes should still 
be relatively balanced.

### Graduation Criteria

Alpha (v1.21): 
- Add LogarithmicScaleDown feature gate to kube-controller-manager 
(disabled by default).
- Unit and e2e tests

Beta (v1.22): 
- Enable LogarithmicScaleDown feature gate by default
- Enable `sorting_deletion_age_ratio` metric

Stable (v1.23):
- Remove LogarithmicScaleDown feature gate
- Make this behavior standard

### Upgrade / Downgrade Strategy

There should be no issues during upgrades and downgrades since this does not affect
any APIs or user-exposed behavior. If there are cluster components that currently
assume or depend on the existing behavior this change should be clearly communicated
to work on an acceptable solution during development of this change.

### Version Skew Strategy

Version skew should have minimal effect with this feature for similar reasons to the
upgrade/downgrade strategy. The lack of exposure or documentation around the current
behavior reduces the risk that it is an expectation from other components.


## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: LogarithmicScaleDown
    - Components depending on the feature gate: kube-controller-manager
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**
  Yes, this changes the default assumption that the youngest pod in a replica set 
  will always be the one evicted. However, it still groups pods by their age and picks 
  from the youngest group.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes. Existing workloads should see no change when disabling this feature.

* **What happens if we reenable the feature if it was previously rolled back?**
  Assumptions that the newest pod will be deleted first may break.

* **Are there any tests for feature enablement/disablement?**
  Tests for feature disablement shouldn't be necessary, as this is already an assumed 
  (but not documented) controller behavior.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  This should not affect running workloads, though there is the possibility that the logic 
  panics which would cause kube-controller-manager to crash

* **What specific metrics should inform a rollback?**
  Increased pod deletions could indicate runaway/hot-loop failures in the scaledown logic.
  Availability of applications may also be affected. Though the intent of this is to provide 
  better available through more distributed victim selection, in cases of desired binpacking 
  pods may remain running on undesired nodes.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  This will be manually tested before the graduation to beta

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  No

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  The scaledown behavior of all replicasets will be affected by this featuregate being 
  enabled, so somehow monitoring them will be necessary to determine it

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [x] Metrics
    - Metric name: sorting_deletion_age_ratio
    - [Optional] Aggregation method:
    - Components exposing the metric: kube-controller-manager
  - [ ] Other (treat as last resort)
  
The metric `sorting_deletion_age_ratio` will provide a histogram of the ratio between the 
chosen `deleted pod`'s age over the current `youngest pod`'s age, for pods where the sort 
algorithm falls back to age. (Pod age is the final criteria in the sorting algorithm, so we don't 
want to measure this ratio for deletions which don't use this feature, as those may validly fall 
outside the desired range).

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  There should be no values `>2` in the above metric when the Pod Cost annotation is unset 
  (see https://github.com/kubernetes/enhancements/tree/master/keps/sig-apps/2255-pod-cost) and 
  the pod's deletion was based on a timestamp comparison (rather than, for example, pod state).

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  No, it is part of the controller-manager

### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  No

* **Will enabling / using this feature result in introducing new API types?**
  No

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**
  No

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
  No

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  No

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  No, perhaps minimal increase in calculating the buckets for pod age

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**
  N/a - this is not a feature of running workloads. The main controller will not work and 
  be unable to scale up or down if API or etcd are unavailable.

* **What are other known failure modes?**
n/a

* **What steps should be taken if SLOs are not being met to determine the problem?**
n/a

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- 2021-01-06: Initial KEP submitted
- 2021-05-07: Updated KEP for graduation to beta

## Drawbacks

The first drawback to this is that assumptions that the newest pod will always be the 
first deleted may break. However, the number of users affected by this should be small 
and acceptable due to the current behavior being undocumented.

This may also introduce slightly more work for the controller manager as it requires 
additional calculations before making a selection of which pod to downscale.

## Alternatives

### Make downscale heuristic an option

Choosing between a random or newest-first downscale heuristic can be left out to
the user, but this has 2 problems:

- Both heuristics optimize for different things, and they might be useful
  together.
- Leaving the decision to the user hurts usability. Given the different
  comparison criteria in the downscale algorithm, it might be hard to describe
  the heuristic in a way that users can take an informed decision.

### Compare pods using their distribution in the failure domains

Pods can express spreading constraints when scheduling via
`.spec.topologySpreadConstraints`. The constraints include the failure domains
to be used and skew tolerations.

This API could be used to calculate spreading skew and inform the downscaling
algorithm to preserve a minimum skew. However, this has 2 problems:

- Calculating the skew might be expensive. The controller needs to track Nodes
  to obtain their topology information.
- Violates separation of concerns. The replication controller needs to implement
  or reuse scheduling algorithms. It also opens the question of whether
  other scheduling features need to be respected during downscale.

