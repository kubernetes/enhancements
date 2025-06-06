# KEP-4622: New TopologyManager Policy which configure the value of maxAllowableNUMANodes

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
- [Goals](#goals)
- [Non-Goals](#non-goals)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [X] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [X] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [X] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [X] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

In this KEP, we propose a new TopologyManager Policy Option called `max-allowable-numa-nodes` to configure the value of maxAllowableNUMANodes in the TopologyManager. The current hard-coded value of 8 was added as a stop-gap 4 years ago to mitigate the state explosion that occurs when trying to enumerate the possible NUMA affinities and generating their hints. By making this setting configurable, we give users the ability to increase this limit when appropriate.

## Goals

- This proposal does not aim to modify the existing TopologyManager Policies. It focuses solely on introducing a new policy option to let users configure the maximum supported number of NUMA nodes.
- Support high-end CPUs with more than 8 NUMA nodes.

## Non-Goals

- It does not address other resource allocation or management aspects within Kubernetes.
- It does not attempt to remove the state explosion that still exists in the TopologyManager.

### User Stories (Optional)

#### Story 1

As a developer in the AI space, I want to use AI accelerators "super chips" which expose ARM cores with more than 8 NUMA nodes.  

#### Story 2

As a user in the high-performance-computing space, I want to enable the sub-NUMA or NUMA-per-socket option of my high-end x86 CPU, which will bring  
the count of NUMA nodes to exceed 8.  

#### Story 3

As administrator of edge nodes, I want to use power-efficient yet massively parallel ARM chips which expose more than 8 NUMA nodes.

### Notes/Constraints/Caveats (Optional)

Setting values higher than the current default may cause performance degradation at admission time. Users must either be willing to accept this or know that they won't actually be affected by it in their particular setup. Fixing this will require rearchitecting the Topology Manager and it is thus out of scope of this KEP.

### Risks and Mitigations

The risk associated with implementing this new proposal is minimal. It pertains only to a distinct policy option within the `TopologyManager` and is safeguarded by the option's inherent security measures, in addition to the default deactivation of the `TopologyManagerPolicyBetaOptions` feature gate.

| Risk                                             | Impact | Mitigation |
| -------------------------------------------------| -------| ---------- |
| Set a value lower 8 causes kubelet crash         | High   | the minimum value legal value should be the current hardcoded value(8), If not, we should log it and fail |
| Set a value too high                             |  Low   | add a log when starting. If possible, we should mark the node Degraded somehow because allocation performance could be significantly slow |

## Design Details

Users can configure the value of maxAllowableNUMANodes in the TopologyManager when the kubelet starts up, It will fail and abort if the user sets the value is lower than the current hardcoded default (8).

```go
  case MaxAllowableNUMANodes:
   optValue, err := strconv.Atoi(value)
   if err != nil {
    return opts, fmt.Errorf("bad value for option %q: %w", name, err)
   }
   opts.MaxAllowableNUMANodes = optValue
      ...

  if opts.MaxAllowableNUMANodes < defaultMaxAllowableNUMANodes {
    return opts, fmt.Errorf("value for option %q is lower than defaultMaxAllowableNUMANodes: %d", MaxAllowableNUMANodes, opts.MaxAllowableNUMANodes)
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

- `k8s.io/kubernetes/pkg/kubelet/cm/topologymanager`: `20240405` - `91.5%`

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

No new integration tests for kubelet are planned.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

For beta:

- Verify the input validation with the existing e2e tests(e.g. 9 or 10 or something bigger than the current default but not "too big")

### Graduation Criteria

#### Beta

- Feature implemented behind the existing static policy feature flag
- Initial unit tests completed and coverage is improved
- Documents is improved and enough guidance and examples can be given to potential users.
- Add a e2e test to verify the input validation.

#### GA

- An existing metric: `kubelet_topology_manager_admission_time` can be used.

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

We anticipate no repercussions. The new policy option is voluntary and operates independent of the existing options.

### Version Skew Strategy

No changes needed.
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
1.31:

- enable by default
- allow gate to disable the feature
- release note

1.33:

- promote to GA
- cannot be disabled
- release note

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
    - `TopologyManagerPolicyBetaOptions`
    - `TopologyManagerPolicyOptions`
  - Components depending on the feature gate: `kubelet`
- [X] Change the kubelet configuration to set a TopologyManager policy of static and a TopologyManager policy option of `max-allowable-numa-nodes`
  - Will enabling / disabling the feature require downtime of the control plane?
    No
  - Will enabling / disabling the feature require downtime or reprovisioning of a node? (Do not assume Dynamic Kubelet Config feature is enabled).
    Yes -- kubelet restart is required.

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

Yes, When it is disabled once (i.e. no value is set), this falls back to the default behavior.

###### What happens if we reenable the feature if it was previously rolled back?

Running containers won't be affected by the rollback of the feature, only newly created will.

###### Are there any tests for feature enablement/disablement?

This new `TopologyManager` policy option will start immediately from beta stage. The unit tests will test whether the configured value of `max-allowable-numa-nodes` is as expected and whether it is the default recommended value when it is not configured.

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

When feature a is not enabled or configured, its value is the default value. and the feature is fully contained in the kubelet, has no dependencies and rollback and upgrades both will affect only newly created pods.

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollout or rollout fail do not impact already running workloads, only impact the new workloads.

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

We have an existing metric which records the topology manager admission time: `kubelet_topology_manager_admission_time`.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

Rollout or upgrade do not impact already running workloads. We plan to add an e2e test for this in the furture.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->
No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->
An existing metric: `kubelet_topology_manager_admission_time` for kubelet  can be used to check if the setting is causing unacceptable performance drops.

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

Examine the kubelet configuration of a node to verify the existence of the feature gate and the utilization of the new policy option. we can use the following command to check the feature if it is enabled:

```
kubectl get --raw "/api/v1/nodes/<nodename>/proxy/configz" | jq '.kubeletconfig.TopologyManagerPolicyOptions'
```

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
- [x] Other (treat as last resort)
  - Details: If their system has more than 8 NUMA nodes, the TopologyManager is turned on and the kubelet is not crashing, then the feature is working.

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

The value of max-allowable-numa-nodes does not (in and of itself) affect the latency of pod admission. With the TopologyManager enabled, the time to admit a pod is tied to the number of NUMA nodes on the physical machine. In the past, this was hard-coded at 8 to ensure that pod admission always completed in a reasonable amount of time. If a machine had more than 8 NUMA nodes, the kubelet would crash with a log message stating that the ToplogyManager is unsupported on machines with more than 8 NUMA nodes. With the new max-allowable-numa-nodes option, admins now have the ability to allow nodes with more than 8 NUMA nodes to run with the TopologyManager enabled. However, it is unknown exactly how much this will slow down pod admission on any given system. This feature is therefore to be used at-your-own-risk until we have a proper solution in place to reduce the state explosion that causes pod admission time to slow down as the number of NUMA nodes increases.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [X] Metrics
  - Metric name: kubelet_topology_manager_admission_time
  - Components exposing the metric: kubelet

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

The feature is not used by workloads in any way shape or form. and it only (potentially) impacts how long it takes for the kubelet to start a workload. We can easily check if this feature is enabled by looking at the kubelet config, example:

```shell
kubectl get --raw "/api/v1/nodes/<nodename>/proxy/configz" | jq '.kubeletconfig.TopologyManagerPolicyOptions'
```

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

N/A

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

No. It doesn't rely on other Kubernetes components.

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
No

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->
No

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->
No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

It will slow down pod admission/start time on the node, and the slowdown occurs because the kubelet's TopoolgyManager now has more combinations it needs to consider when deciding where a cpus and devices can be allocated in an aligned way, and the slowdown affects only node configured with the feature, there is not any cluster impact as the feature is at node-level.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

It will increase the kubelet's CPU usage time. If your system has more than 8 NUMA nodes, then you will not be able to run kubernetes on it without this feature. so
the purpose is then to provide an escape hatch for those that are OK paying the price of increased latency for pod admission (and its associated CPU/RAM costs) in order to allow the kubelet to run on such a node.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

Same answer as above.

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

N/A

###### What are other known failure modes?

Setting a value lower 8 causes kubelet crash.

###### What steps should be taken if SLOs are not being met to determine the problem?

 As a cluster administrator you should know the number of NUMA nodes on your nodes and adjust the value of the kubelet's topologyManager options or turn it off.

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

- 2024-05-08 - initial KEP draft created
- 2024-06-06 - updates per review feedback
- 2025-02-12 - promote it to GA

## Drawbacks

- increased kubelet's CPU/Memory usage time
- increase in pod start time

Before this feature: the kubelet would crash.
With this feature: you get a potential slowdown, but at least the kubelet will run.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

Adding a new kubelet configuration option.

<!--
## Infrastructure Needed (Optional)
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->