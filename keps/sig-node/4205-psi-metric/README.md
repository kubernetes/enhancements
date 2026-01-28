# KEP-4205: Expose PSI Metrics
<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
      - [CPU](#cpu)
      - [Memory](#memory)
      - [IO](#io)
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

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
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

This KEP proposes adding support in kubelet to read Pressure Stall Information (PSI) metric pertaining to CPU, Memory and IO resources exposed from cAdvisor and runc.

## Motivation

[PSI metric](https://www.kernel.org/doc/Documentation/accounting/psi.txt) provides a quantifiable way to see resource pressure increases as they develop, with a new pressure metric for three major resources (memory, CPU, IO). These pressure metrics are useful for detecting resource shortages and provide nodes the opportunity to respond intelligently - by updating the node condition.

In short, PSI metric are like barometers that provide fair warning of impending resource shortages on the node, and enable nodes to take more proactive, granular and nuanced steps when major resources (memory, CPU, IO) start becoming scarce.

### Goals

This proposal aims to:
1. Enable the kubelet to have the PSI metric of cgroupv2 exposed from cAdvisor and Runc.
2. Enable the pod level PSI metric and expose it in the Summary API.

### Non-Goals

* Invest in more opportunities to further use PSI metric for pod evictions,
userspace OOM kills, and so on, for future KEPs.

## Proposal

### User Stories (Optional)

#### Story 1

Today, to identify disruptions caused by resource crunches, Kubernetes users need to
install node exporter to read PSI metric. With the feature proposed in this enhancement, 
PSI metric will be available for users in the Kubernetes metrics API.

### Risks and Mitigations

There are no significant risks associated with integrating
the PSI metric in kubelet from either from cadvisor runc libcontainer library or kubelet's CRI runc libcontainer implementation which doesn't involve any shelled binary operations.

## Design Details

1. Add new Data structures PSIData and PSIStats corresponding to the PSI metric output format as following:

```
some avg10=0.00 avg60=0.00 avg300=0.00 total=0
full avg10=0.00 avg60=0.00 avg300=0.00 total=0
```

```go
// PSI data for an individual resource.
type PSIData struct {
	// Total time duration for tasks in the cgroup have waited due to congestion.
	// Unit: nanoseconds.
	Total  uint64 `json:"total"`
	// The average (in %) tasks have waited due to congestion over a 10 second window.
	Avg10  float64 `json:"avg10"`
	// The average (in %) tasks have waited due to congestion over a 60 second window.
	Avg60  float64 `json:"avg60"`
	// The average (in %) tasks have waited due to congestion over a 300 second window.
	Avg300 float64 `json:"avg300"`
}

// PSI statistics for an individual resource.
type PSIStats struct {
	// PSI data for some tasks in the cgroup.
	Some PSIData `json:"some,omitempty"`
	// PSI data for all tasks in the cgroup.
	Full PSIData `json:"full,omitempty"`
}
```

2. Summary API includes stats for both system and kubepods level cgroups. Extend the Summary API to include PSI metric data for each resource obtained from cadvisor. 
Note: if cadvisor-less is implemented prior to the implementation of this enhancement, the PSI
metric data will be available through CRI instead.

##### CPU
```go
type CPUStats struct { 
	// PSI stats of the overall node
	PSI *PSIStats `json:"psi,omitempty"`
}
```

##### Memory
```go
type MemoryStats struct {
	// PSI stats of the overall node
	PSI *PSIStats `json:"psi,omitempty"`
}
```

##### IO
```go
// IOStats contains data about IO usage.
type IOStats struct {
	// The time at which these stats were updated.
	Time metav1.Time `json:"time"`

	// PSI stats of the overall node
	PSI *PSIStats `json:"psi,omitempty"`
}

type NodeStats struct {
	// Stats about the IO pressure of the node
	IO *IOStats `json:"io,omitempty"`
}
```

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

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
-->
- `k8s.io/kubernetes/pkg/kubelet/server/stats`: `2023-10-04` - `74.4%`
- `k8s.io/kubernetes/pkg/kubelet/stats`: `2025-06-10` - `77.4%`

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

Within Kubernetes, the feature is implemented solely in kubelet. Therefore a Kubernetes integration test doesn't apply here.

Any identified external user of either of these endpoints (prometheus, metrics-server) should be tested to make sure they're not broken by new fields in the API response. 

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- `test/e2e_node/summary_test.go`: `https://storage.googleapis.com/k8s-triage/index.html?test=test%2Fe2e_node%2Fsummary_test.go`

### Graduation Criteria

#### Alpha

- PSI integrated in kubelet behind a feature flag.
- Unit tests to check the fields are populated in the 
  Summary API response.

#### Beta

- Feature gate is enabled by default.
- Extend e2e test coverage.
- Allowing time for feedback.
- Performance testing to verify:
    - Verification enabling PSI on nodes doesn't introduce excessive CPU or memory usage in the kernel
    - PSI metrics collection doesn't introduce excessive CPU or memory usage increase in the kubelet

#### GA
- Quantify the cAdvisor and kubelet-level overhead of PSI metric collection, especially where PSI is disabled at the kernel level.
- Validate with SIG Node that collection overhead is acceptable for general use cases, or include opt-out knobs.
- Exoanded stress testing with diverse environments and scenarios, while maintining acceptable minimal resource consumption like outlined in Beta perf testing.
- Gather evidence of real-world usage from beta users.
- No major issues reported.

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

### Upgrade / Downgrade Strategy

No impact. Runc will be upgraded to 1.2.0 version as a prerequisite for this feature,
and all the other components will already be at expected levels. Hence there shouldn't
be a problem in upgrading or downgrading. Besides, it's always possible to upgrade/downgrade
to a different kubelet version.

### Version Skew Strategy

N/A

PSI stats will be available only after CRI and cadvisor have been updated to use runc 1.2.0
in K8s 1.29. Since `PSI Based Node Conditions` is dependent on kubelet version, and CRI and kubelet are generally updated in tandem, Version skew strategy is not applicable.

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

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: KubeletPSI
  - Components depending on the feature gate: kubelet

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->
No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->
Yes, but starting in v1.36 where this feature graduates to GA, the KubeletPSI feature gate will be locked to true and will no longer be disable-able.

###### What happens if we reenable the feature if it was previously rolled back?
No PSI metrics will be available in kubelet Summary API nor Prometheus metrics if the
feature was rolled back.

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
Unit tests

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

The PSI metrics in kubelet Summary API and Prometheus metrics are for monitoring purpose,
and are not used by Kubernetes itself to inform workload lifecycle decisions. Therefore it should
not impact running workloads.

If there is a bug and kubelet fails to serve the metrics during rollout, the kubelet Summary API
and Prometheus metrics could be corrupted, and other components that depend on those metrics could
be impacted. Disabling the feature gate / rolling back the feature should be safe.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

PSI metrics exposed at kubelet `/metrics/cadvisor` endpoint:

```
container_pressure_cpu_stalled_seconds_total
container_pressure_cpu_waiting_seconds_total
container_pressure_memory_stalled_seconds_total
container_pressure_memory_waiting_seconds_total
container_pressure_io_stalled_seconds_total
container_pressure_io_waiting_seconds_total
```

kubelet Summary API at the `/stats/summary` endpoint.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

Test plan:
- Create pods when the feature is alpha and disabled
- Upgrade kubelet so the feature is beta and enabled
  - Pods should continue to run
  - PSI metrics should be reported in kubelet Summary API and Prometheus metrics
- Roll back kubelet to previous version
  - Pods should continue to run
  - PSI metrics should no longer be reported

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No

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
Use `kubectl get --raw "/api/v1/nodes/{$nodeName}/proxy/stats/summary"` to call Summary API. If the PSIStats field is seen in the API response,
the feature is available to be used by workloads.

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
  - Details: The feature is only about metrics surfacing. One can know that it is working by reading the metrics.

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

kubelet Summary API and Prometheus metrics should continue serving traffics meeting their originally targeted SLOs

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
Yes, it depends on runc version 1.2.0. This KEP can be implemented only after runc 1.2.0 is released, which is estimated to be released in Q1 2024.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

Yes, PSIStats is the new API type that will be added to Summary API.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. Additional metric i.e. PSI is being read from cadvisor.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

NA

###### How does this feature react if the API server and/or etcd is unavailable?

- NA.


###### What are other known failure modes?

NA

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 2023/09/13: Initial proposal
- 2025/06/10: Drop Phase 2 from this KEP. Phase 2 will be tracked in its own KEP to allow separate milestone tracking
- 2025/06/10: Update the proposal with Beta requirements

## Drawbacks

No drawbacks identified. There's no reason the enhancement should not be
implemented. This enhancement now makes it possible to read PSI metric without installing
additional dependencies

## Infrastructure Needed (Optional)

No new infrastructure is needed.
