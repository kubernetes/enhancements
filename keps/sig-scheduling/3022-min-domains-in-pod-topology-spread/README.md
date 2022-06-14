# KEP-3022: min domains in Pod Topology Spread

<!-- toc -->

- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Story](#user-story)
- [Design Details](#design-details)
  - [API](#api)
  - [Implementation details](#implementation-details)
  - [How user stories are addressed](#how-user-stories-are-addressed)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (v1.24):](#alpha-v124)
    - [Beta (v1.25):](#beta-v125)
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
  - [Support <code>minDomains</code> in ScheduleAnyway as well](#support--in-scheduleanyway-as-well)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required _prior to targeting to a milestone / release_.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

A new field `minDomains` is introduced to `PodSpec.TopologySpreadConstraint[*]` to limit
the minimum number of topology domains.
`minDomains` can be used only when `whenUnsatisfiable=DoNotSchedule`.

## Motivation

Pod Topology Spread has [`maxSkew` parameter](https://github.com/kubernetes/enhancements/tree/11a976c74e1358efccf251d4c7611d05ce27feb3/keps/sig-scheduling/895-pod-topology-spread#maxskew), which control the degree to which Pods may be unevenly distributed.
But, there isn't a way to control the number of domains over which we should spread.
In some cases, users want to force spreading Pods over a minimum number of domains and, if there aren't enough already present, make the cluster-autoscaler provision them.

### Goals

- Users can specify `minDomains` to limit the number of domains when using `WhenUnsatisfiable=DoNotSchedule`.

### Non-Goals

- Add new field to limit the maximum number of topology domains.
- Users can use it as a best-efforts manner with `WhenUnsatisfiable=ScheduleAnyway`.

## Proposal

### User Story

I am using cluster autoscaler and I want to force spreading a deployment over at least 5 Nodes.

## Design Details

Users can define a minimum number of domains with `minDomains` parameter.
This parameter only applies when `whenUnsatisfiable=DoNotSchedule`.

Pod Topology Spread has the semantics of "global minimum", which means the minimum number of pods that match the label selector in a topology domain.

However, the global minimum is only calculated for the nodes that exist and match the node affinity. In other words, if a topology domain was scaled down to zero (for example, because of low utilization), this topology domain is unknown to the scheduler, thus it's not considered in the global minimum calculations.

The new `minDomains` field can help with this problem.

When the number of domains with matching topology keys is less than `minDomains`,
Pod Topology Spread treats "global minimum" as 0; otherwise, "global minimum"
is equal to the minimum number of matching pods on a domain.

As a result, when the number of domains is less than `minDomains`, scheduler doesn't schedule a matching Pod to Nodes on the domains that have the same or more number of matching Pods as `maxSkew`.

`minDomains` is an optional parameter. If `minDomains` is nil, the constraint behaves as if MinDomains is equal to 1.

### API

New optional parameter called `MinDomains` is introduced to `PodSpec.TopologySpreadConstraint[*]`.

```go
type TopologySpreadConstraint struct {
......
	// MinDomains indicates a minimum number of eligible domains.
	// When the number of eligible domains with matching topology keys is less than minDomains,
	// Pod Topology Spread treats "global minimum" as 0, and then the calculation of Skew is performed.
	// And when the number of eligible domains with matching topology keys equals or greater than minDomains,
	// this value has no effect on scheduling.
	// As a result, when the number of eligible domains is less than minDomains,
	// scheduler won't schedule more than maxSkew Pods to those domains.
	// If value is nil, the constraint behaves as if MinDomains is equal to 1.
	// Valid values are integers greater than 0.
	// When value is not nil, WhenUnsatisfiable must be DoNotSchedule.
	//
	// For example, in a 3-zone cluster, MaxSkew is set to 2, MinDomains is set to 5 and pods with the same
	// labelSelector spread as 2/2/2:
	// +-------+-------+-------+
	// | zone1 | zone2 | zone3 |
	// +-------+-------+-------+
	// |  P P  |  P P  |  P P  |
	// +-------+-------+-------+
	// The number of domains is less than 5(MinDomains), so "global minimum" is treated as 0.
	// In this situation, new pod with the same labelSelector cannot be scheduled,
	// because computed skew will be 3(3 - 0) if new Pod is scheduled to any of the three zones,
	// it will violate MaxSkew.
	//
	// This is an alpha field and requires enabling MinDomainsInPodTopologySpread feature gate.
	// +optional
  MinDomains *int32
}
```

### Implementation details

In Filter of Pod Topology Spread, current filtering criteria is

```
('existing matching num' + 'if self-match (1 or 0)' - 'global min matching num') <= 'maxSkew'
```

- `existing matching num` denotes the number of current existing matching Pods on the domain.
- `if self-match` denotes if the labels of Pod matches with selector of the constraint.
- `global min matching num` denotes the minumun number of matching Pods.

For `whenUnsatisfiable: DoNotSchedule`, Pod Topology Spread will treat `global min matching num` as 0
when the number of domains with matching topology keys is less than `minDomains`.

We can calculate the number of domains with matching topology keys in PreFilter, along with the calculation of [`TpPairToMatchNum`](https://github.com/kubernetes/kubernetes/blob/0153febd9f0098d4b8d0d484927710eaf899ef40/pkg/scheduler/framework/plugins/podtopologyspread/filtering.go#L49).
This extra calculation doesn't increase the complexity of the preFilter logic.
Pod Topology Spread will be able to use the number of domains to determine the value of `global min matching num` when we calculate filtering criteria.

### How user stories are addressed

Users can set `MinDomains` and `whenUnsatisfiable: DoNotSchedule` to achieve it.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  selector:
    matchLabels:
      app: nginx
  replicas: 10
  template:
    metadata:
      labels:
        foo: bar
    spec:
      containers:
        - name: nginx
          image: nginx:1.14.2
          ports:
            - containerPort: 80
      topologySpreadConstraints:
        - maxSkew: 2
          minDomains: 5
          topologyKey: kubernetes.io/hostname
          whenUnsatisfiable: DoNotSchedule
          labelSelector:
            matchLabels:
              foo: bar
```

Considering the case that we have 3 Nodes which can schedule Pods to.

6 Pods will be scheduled to that Nodes, and the rest 4 Pods can only be scheduled when 2 more Node join the cluster.

With the flow, this deployment will be spread over at least 5 Nodes while protecting the constraints of `maxSkew`.

### Test Plan

To ensure this feature to be rolled out in high quality. Following tests are mandatory:

- **Unit Tests**: All core changes must be covered by unit tests.
- **Integration Tests / E2E Tests:** Tests to ensure the behavior of this feature must
  be covered by either integration tests or e2e tests.
- **Benchmark Tests:** We can bear with slight performance overhead if users are
  using this feature, but it shouldn't impose penalty to users who are not using
  this feature. We will verify it by designing some benchmark tests.

### Graduation Criteria

#### Alpha (v1.24):

- [x] Add new parameter `MinDomains` to `TopologySpreadConstraint` and feature gating.
- [x] Filter extension point implementation.
- [x] Implement all tests mentioned in the [Test Plan](#test-plan).

#### Beta (v1.25):

- [ ] This feature will be enabled by default as a Beta feature in v1.25.

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

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `MinDomainsInPodTopologySpread`
  - Components depending on the feature gate: `kube-scheduler`, `kube-apiserver`

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

The feature can be disabled in Alpha and Beta versions
by restarting kube-apiserver and kube-scheduler with feature-gate off.
In terms of Stable versions, users can choose to opt-out by not setting the
`pod.spec.topologySpreadConstraints.minDomains` field.

###### What happens if we reenable the feature if it was previously rolled back?

Scheduling of new Pods is affected.

###### Are there any tests for feature enablement/disablement?

No - unit and integration tests will be added.

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

It shouldn't impact already running workloads. It's an opt-in feature,
and users need to set `pod.spec.topologySpreadConstraints.minDomains` field to use this feature. 

When this feature is disabled by the feature flag, the already created Pod's `pod.spec.topologySpreadConstraints.minDomains` field is preserved,
but, the newly created Pod's `pod.spec.topologySpreadConstraints.minDomains` field is silently dropped.


###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

- A spike on metric `schedule_attempts_total{result="error|unschedulable"}` when pods using this feature are added.
- A spike on metric `plugin_execution_duration_seconds{plugin="PodTopologySpread"}` or `scheduling_algorithm_duration_seconds` when pods using this feature are added. 

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

Yes. The behavior is changed as expected.

Test scenario:
1. start kube-apiserver that `MinDomains` feature is enabled. 
2. create three nodes and pods spread across nodes as 2/2/1
3. create new Pod that has a TopologySpreadConstraints: maxSkew is 1, topologyKey is `kubernetes.io/hostname`, and minDomains is 4 (larger than the number of domains (= 3)).
4. the Pod created in (3) isn't scheduled because of `MinDomain`.
5. delete the Pod created in (3).
6. recraete kube-apiserver that `MinDomains` feature is disabled.
7. create the same Pod as (3).
8. the Pod created in (7) is scheduled because `MinDomain` is disabled.
9. delete the Pod created in (7).
10. recreate kube-apiserver that `MinDomains` feature is enabled.
11. create the same Pod as (3).
12. the Pod created in (11) isn't scheduled because of `MinDomain`.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

The operator can query pods with `pod.spec.topologySpreadConstraints.minDomains` field set.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [x] Other (treat as last resort)
  - Details: 
              The feature MinDomains in Pod Topology Sprad plugin doesn't cause any logs, any events, any pod status updates.
              If a Pod using `pod.spec.topologySpreadConstraints.minDomains` was successfully assigned a Node,
              nodeName will be updated. 
              And if not, `PodScheduled` condition will be false and an event will be recorded with a detailed message
              describing the reason including the failed filters. (Pod Topology Spread plugin could be one of them.)

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

- Metric `plugin_execution_duration_seconds{plugin="PodTopologySpread"}` <= 100ms on 90-percentile.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

  - [x] Metrics
    - Component exposing the metric: kube-scheduler
      - Metric name: `plugin_execution_duration_seconds{plugin="PodTopologySpread"}`
      - Metric name: `schedule_attempts_total{result="error|unschedulable"}`

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

Yes. It would be useful if we could see more details related to scheduler's decisions in metrics.

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

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Describe them, providing:

- API type(s): Pod
- Estimated increase in size: new field `.Spec.topologySpreadConstraint.MinDomains` about 4 bytes (int32)

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. The performance degradation on scheduler is not expected.

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

The scheduler have to process `MinDomains` parameter which may result in some small increase in CPU usage.

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

The feature isn't affected because Pod Topology Spread plugin doesn't communicate with kube-apiserver or etcd
during Filter phase.

###### What are other known failure modes?

N/A

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

- Check `plugin_execution_duration_seconds{plugin="PodTopologySpread"}` to see if latency increased. 
  - In this case, the metrics showes literally the feature is slow.
  - You should stop using `MinDomains` in your Pods and may need to disable `MinDomains` feature by feature flag `MinDomainsInPodTopologySpread`.
- Check `schedule_attempts_total{result="error|unschedulable"}` to see if the number of attempts increased.
  - In this case, your use of `MinDomains` may be incorrect or not appropriate for your cluster.

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

 - 2021-11-02: Initial KEP sent for review
 - 2022-01-14: Initial KEP is merged. 
 - 2022-03-16: The implementation PRs are merged.
 - 2022-05-03: The MinDomain feature is released as alpha feature with Kubernetes v1.24 release.

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

### Support `minDomains` in ScheduleAnyway as well

When the number of domains with matching topology keys is less than `minDomains` and `whenUnsatisfiable` equals to `ScheduleAnyway`,
Pod Topology Spread will give low scores to Nodes on the domains which have the same or more number of matching Pods as `maxSkew`.

In Pod Topology Spread, the higher the score from Score, the lower will be the normalized score calculated by Normalized Score. So, Pod Topology Spread should give high scores to non-preferred Nodes in Score.

When the number of domains with matching topology keys is less than `minDomains`,
Pod Topology Spread doubles that score for the constraint in Score (so that normalized score will be a lower score) if this criteria is met:

```
('existing matching num' + 'if self-match (1 or 0)' - 'global min matching num') > 'maxSkew'
```

- `existing matching num` denotes the number of current existing matching Pods on the domain.
- `if self-match` denotes if the labels of Pod matches with selector of the constraint.
- `global min matching num` denotes the minumun number of matching Pods.

This `minDomains` in ScheduleAnyway is decided not to support because of the following reasons:

- To support this, we need to calculate the number of domains with matching topology keys and the minimum number of matching Pods in preScore like preFilter, so that Pod Topology Spread can determine the evaluation way with them.

  This extra calculation may affect the performance of the preScore, because the current preScore only see Nodes which have passed the Filter, but to calculate them, Pod Topology Spread needs to see all Nodes (includes Nodes which haven't passed the Filter).

- `minDomains` is supported mainly for [the above user story](#user-story), which using the cluster autoscaler.

  The scoring results of scheduler doesn't affect the cluster-autoscaler. So, it is not worth supporting with the performance degradation.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
