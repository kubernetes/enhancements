# KEP-1923: Prefer Nominated Node

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
- [Design Details](#design-details)
  - [Implementation Details](#implementation-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (v1.21):](#alpha-v121)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes to change the scheduling cycle such that nominated node of a pod is evaluated first
and schedule the pod on that node if it fits. If the nominated node doesn't fit the pod, only then the
scheduling cycle continues with the standard logic of evaluating the rest of the nodes in the cluster.

## Motivation

If the scheduler fails to fit an incoming pod on any node, it will try to preempt lower priority pods
running on a selected node to make room for the pod. The name of this node will be set in the
pod's `.status.nominatedNodeName`.

The Node is called *Nominated* to indicate the intent for the Pod to be scheduled on it once preemption
of other Pods finishes. However, the Pod's `.status.nominatedNodeName` information is not fully utilized
in the Pod's following scheduling attempts.

Pod scheduling is split into two phases, the scheduling cycle and the binding cycle, the scheduling cycle
primarily includes filtering and scoring.

When preemption happens in a previous scheduling cycle, there is a high chance that the nominated node is
the *only* node that satisfies the filters for the unscheduled Pod that triggered preemption.

In real production environment, pods can have different priorites due to business needs, the preemption
could happen to make sure higher priority pods could get scheduled.

In cluster with large number of computing nodes, evaluating all nodes when scheduling a pod is time consuming.

### Goals

Prefer scheduling a pod to its `.status.nominatedNodeName` if set, if the nominated node doesn't fit the pod,
the scheduling cycle will continue to evaluate the rest of the nodes in the cluster just like we do today.


## Proposal

### User Stories (Optional)

Users want faster scheduling. Since it is highly likely the pod will only fit on the nominated node, the improvement
in scheduling latency will come at negligible cost (the cost being placing the pod on a less optimal node).

### Notes/Constraints/Caveats (Optional)

When this feature is enabled the preemptor Pod might not be dispatched to the best candidated node in some corner case,
e.g. another node releases the resources and becomes the best candidate while the victim pods got removed from the
nominated node.

## Design Details

### Implementation Details

1. In filtering phase, which is currently implemented in the method of `findNodesThatFitPod`, check the nominated node
   first if the incoming pod has the `.status.nominatedNodeName` defined and the feature gate is enabled.

2. In case the nominated node doesn't suit for the incoming pod anymore, get `err` from `findNodesThatPassFilters` where
   `NominatedNode` is firstly evaluated, the `err` will be padded with more information to tell that scheduler is evaluating
   the feasibility of `NominatedNode` and failed on that node.

   If no error is returned but `NominatedNode` cannot pass all the filtering, this is possibly caused by the resource that
   claims to be removed but has not been fully released yet.

   For both of above cases, scheduler will continue to evaluate the rest of nodes to check if there is any node already
   available for the coming pod.

### Test Plan

Following tests will be covered or considered:

- **Unit Tests**: All core changes must be covered by unit tests.
- **Integration Tests**: Integration test will be provided if necessary, for example,
  - enable the feature
  - preempt the victim pods on the nominated node
  - check pod will be scheduled on the nominated node
- **Benchmark Tests**: A benchmark test which compares the performance before and after the change.
  The performance improvement is visible by benchmark of `scheduling_algorithm_predicate_evaluation_seconds`.
  Other benchmark will be created on-demand along with the code review process.


### Graduation Criteria

#### Alpha (v1.21):

- [x] New feature gate proposed to enable the feature.
- [x] Implementation of the new feature in scheduling framework.
- [x] Test cases mentioned in the [Test Plan](#test-plan).

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: PreferNominatedNode
    - Components depending on the feature gate: kube-scheduler

* **Does enabling the feature change any default behavior?**
  Yes. If the coming pod has the nominated node set, then the nominated node will be evaluated first in any
  scheduling cycle, this is only the default process logic that is handled by scheduler, end-user will not
  and need not aware of any difference.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes. It can be disabled by restarting scheduler with feature gate turned off.

* **What happens if we reenable the feature if it was previously rolled back?**
  The feature will start working again when scheduling pods.

* **Are there any tests for feature enablement/disablement?**
  unittest will switch the feature gate manually to enable the feature, and compare the different behavior.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At a high level, this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide comprehensive guidance, but at the very
  high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
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

### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  Describe them, providing:
  No.

* **Will enabling / using this feature result in introducing new API types?**
  Describe them, providing:
  No.

* **Will enabling / using this feature result in any new calls to the cloud
provider?**
  No.

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**
  No.

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**
  No.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  No.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**
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

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- 2020-09-29: Initial KEP sent out for review https://github.com/kubernetes/enhancements/pull/2026
- 2020-12-17: Mark the KEP as implementable
