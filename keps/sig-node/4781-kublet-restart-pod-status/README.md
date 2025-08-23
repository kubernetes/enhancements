# KEP-4781: Restarting kubelet does not change pod status

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
      - [Inconsistency with other Kubernetes components](#inconsistency-with-other-kubernetes-components)
      - [Delayed Health Check Updates](#delayed-health-check-updates)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [deprecated](#deprecated)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
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

When the kubelet restarts in a short period, it actually won't affect the status of nodes and pods, and the service are expected to remain availability. However, the Pod's Started and Ready statuses are both set to False by default currently, which can disrupt services that were previously functioning normally. This KEP proposes improving Pod readiness management in the kubelet to ensure that the status of Pods is preserved during kubelet restarts.

## Motivation

Ensuring high availability and minimizing service disruptions are critical considerations for Kubernetes clusters. 
When the kubelet restarts, it resets the Start and Ready states of all containers to False by default. This means that any successful probe statuses that were previously established are lost upon the restart. As a result, services may be inaccurately flagged as unavailable, despite having been operational prior to the kubelet's restart. This reset can lead to erroneous perceptions of service health and negatively impact the overall performance of the cluster, potentially triggering unnecessary alerts or load balancing changes. 
It's essential to implement strategies to ensure that the service states accurately reflect their operational status, even during kubelet interruptions.

### Goals

- Ensure consistency in container start and ready states across kubelet restarts.
- Minimize unnecessary service disruptions caused by temporary ready state changes.

### Non-Goals

- If the kubelet fails to renew its lease beyond the nodeMonitorGracePeriod due to an excessively long restart interval, the Ready status of the containers in the pods on the node will be set to false. In this situation, we should not manually set the Ready status back to true. Instead, it should remain false, waiting for the probe to execute again and restore it.
- Modify the fundamental logic of how readiness probes work.

## Proposal

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1
As a user of Kubernetes, I want the container's Ready state to remain consistent across kubelet restarts so that my services do not experience unnecessary downtime.
However, currently, a kubelet restart causes a brief "Not Ready" storm, where the state of all Pods is set to Not Ready, impacting the availability of my services.

### Risks and Mitigations

##### Inconsistency with other Kubernetes components
If other parts of Kubernetes (e.g., the API server, controllers) expect certain behavior regarding container readiness states, these changes might cause inconsistencies.

##### Delayed Health Check Updates
By preserving the old state without immediate health checks, there is a delay in recognizing containers that have become unhealthy during or after kubelet's downtime. Services relying on Pod readiness for service discovery might continue directing traffic to Pods with containers that are no longer healthy but are still reported as Ready.
We plan to immediately trigger a probe after that to reduce the risk caused by such delays.

## Design Details

1. We will retrieve the `Started` field from the container status in the Pod via the API server. After the Kubelet restarts, during the first entry into `SyncPod`, we will propagate this value to the newly generated container status.

2. We ensure that if the `Started` field in the container status is true, the container is considered started (since the startupProbe only runs during container startup and will not execute again once completed).

3. If the Kubelet restart occurs within the `nodeMonitorGracePeriod` and the Pod’s Ready condition is set to false, we will set the container’s ready status to false. It will remain in this state until subsequent probes reset it to true.

4. We will modify the logic in the `doProbe` function. When it detects a container that was already running before the Kubelet restarted (for the first time after restart), it will skip marking an initial Failure status. This allows the probe `result` to retain the default `Success` status. If the container’s state changes during the Kubelet restart period and causes the probe to return an abnormal result, the status will be updated to a non-Success state in the next probe cycle. Subsequent syncPod operations will then set the container’s Ready status to false.

**Before the Changes:**
If kubelet restarts, the pod status transition process is as follows:

1. Kubelet uses `SyncPod` to reconcile the pod state. During the first execution of `SyncPod`, the pod has not yet been added to the `probeManager`. At this point, `SyncPod` assumes the pod has no probes configured (note: if it is a newly created pod, the first execution of `SyncPod` does not go through this step). Therefore, it sets the container's `Ready` status to true and updates it to the APIserver.

2. After updating the container status, `SyncPod` adds the pod to the `probeManager`. The pod then begins executing probes.

3. During the first execution of `doProbe`, `doProbe` sets the result of all probes to their `initialValue`. The `initialValue` for `readinessProbe` is `Failure`, and for `startupProbe` it is `Unknown`. Based on the probe results, it updates the `Started` and `Ready` fields of the container status in the APIserver to false.

**After the Changes:**  
**Scenario 1:**
After the changes, if kubelet restarts, the pod status transition process is as follows:

* (The first two steps are the same as before the changes and are omitted here.)
3. During the first execution of `doProbe`, if the pod's creation time is earlier than kubelet's start time and the container's Ready status is true, `doProbe` skips the step of setting all probe results to their `initialValue` and proceeds with subsequent probe steps. This ensures that kubelet can immediately probe whether the container is still functioning properly after restarting, avoiding a situation where the container becomes unhealthy during kubelet restart but kubelet fails to update the container's `Ready` fields to false in a timely manner.

**Scenario 2:**
After the changes, if kubelet restarts for a sufficiently long time such that the pod's `Ready condition` is set to false:

1. `Kubelet` uses `SyncPod` to reconcile the pod state. During the first execution of `SyncPod`, if kubelet detects that the pod's `Ready condition` is false, it directly sets the container's `Ready` fields to `false` and updates it to the APIserver.

2. After updating the container status, `SyncPod` adds the pod to the `probeManager`. The pod then begins executing probes.

3. The logic here is the same as in Scenario 1. Since the container's `Ready` fields is false, doProbe sets the result of all probes to their `initialValue` and updates the `Started` and `Ready` fields of the container status in the API server to false based on the probe results. Subsequent executions of `doProbe` will then transition the pod status to the desired state.

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

- `pkg/kubelet/prober`: `2025-08-25` - `77.4%`
- `k8s.io/kubernetes/pkg/kubelet`: `2025-08-25` - `71.2%`

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

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

### Graduation Criteria

#### deprecated

Implement the code and add the `ConsistentPodStatusOnRestart` feature gate.
Add e2e tests to ensure the functionality meets expectations.

#### GA

During the Deprecated phase, no issues were reported by users.

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

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ConsistentPodStatusOnRestart`
  - Components depending on the feature gate: `kubelet`

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->
Yes, currently, when a kubelet restarts, the state of Pods and containers are reported as Not Ready. This feature changes the behavior to inherit the last state of Pods and containers, thus avoiding service inconsistencies, but may introduce delayed updates to the Not Ready state.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Due to the use of a feature gate, the feature can be disabled by setting the gate to false.

###### What happens if we reenable the feature if it was previously rolled back?
If reenabled, the functionality of this KEP will take effect. Other than that, there will be no changes, the modifications in this KEP are fully compatible with previous behavior.


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

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

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

No

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

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No


###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

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

If a container becomes unhealthy during the kubelet restart, the kubelet may still report a Ready status until the Readiness probe completes its check. This can lead to other Kubernetes components making decisions based on stale information, such as directing traffic to an unhealthy Pod, resulting in service degradation or failed user requests.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->