# KEP-4786: Direct cgroup stats collection on Node

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
  - [Current Use of cAdvisor](#current-use-of-cadvisor)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Root Filesystem Stats](#root-filesystem-stats)
  - [Cgroup Stats](#cgroup-stats)
  - [Windows](#windows)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

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

The [cAdvisor] is still started and running behind the scene by the Kubelet
after [KEP-2371] to gather system cgroup stats. Running cAdvisor brings in extra
overhead and complexity. However, the stat collection of system cgroups is
trivial to implement with a low overhead as there are only a handful of system
cgroups.

This KEP aims to eliminate the need to run cAdvisor with enablement of KEP-2371
for better performance and simplicity.

[cAdvisor]: https://github.com/google/cadvisor
[KEP-2371]: https://github.com/kubernetes/enhancements/edit/master/keps/sig-node/2371-cri-pod-container-stats/README.md

### Current Use of cAdvisor

After enabling feature `PodAndContainerStatsFromCRI`, only
[summary API][summary-api] invokes cAdvisor for stats of:

*   Root filesystem
*   Root cgroup (CPU, Memory, Swap and Network stats)
*   System cgroup (CPU, Memory and Swap)
    *   Kubelet cgroup
    *   Runtime cgroup
    *   Pod Root cgroup
    *   Misc cgroup

[summary-api]: https://github.com/kubernetes/kubernetes/blob/release-1.30/staging/src/k8s.io/kubelet/pkg/apis/stats/v1alpha1/types.go

## Motivation

Running cAdvisor is a visible performance overhead and there are work to tune
the cAdvisor parameters to minimize the overhead such as [#18044], [#124520] and
[#107960]. The overhead would be much higher on Nodes of high pod density as
there are many more cgroups to collect stats, which is also redandunt when
`PodAndContainerStatsFromCRI` is enabled since pod stats are collected via CRI
APIs.

[#18044]: https://github.com/kubernetes/kubernetes/issues/18044
[#124520]: https://github.com/kubernetes/kubernetes/pull/124520
[#107960]: https://github.com/kubernetes/kubernetes/pull/107960

### Goals

*   Improve performance in Kubelet without running cAdvisor.
*   Do not introduce breaking changes to the Summary API or eviction function.

### Non-Goals

*   Enhance or implement any items from KEP-2371

## Proposal

This KEP only involves changes in Kubelet so it's trivial to move logic from
cAdvisor to collect system stats. Simply put, Kubelet will collect those system
stats directly instead of calling cAdvisor APIs.

Prior to this KEP, cadviosr collects all metrics (CPU, Memory, Disk, Network,
Process etc) from following cgroups:

*   Root cgroup
*   All non-pod cgroups
    *   Including non-pod cgroups as specified in `--runtime-cgroups`,
        `--system-cgroups` and `--kubelet-cgroups`
*   Root cgroup for pods (e.g. /sys/fs/cgroup/kubepods.slice/)
*   Pod cgroups

After this KEP, Kubelet collects subset of above metrics from following cgroups:

*   Root cgroup
*   Non-pod cgroups as specified in `--runtime-cgroups`, `--system-cgroups` and
*   Root cgroup for pods (e.g. /sys/fs/cgroup/kubepods.slice/)
    `--kubelet-cgroups`

The overhead reduction comes from fewer cgropus and metrics to collect.

### User Stories (Optional)

#### Story 1

#### Story 2

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

*   There might be subtle discrepancy between stats collected by cAdvisor and
    Kubelet so we should carefully use existing E2E tests to catch them and run
    manual tests if no tests available already to vet the discrepancy.

## Design Details

### Root Filesystem Stats

The Kubelet gets the root filesystem stats via the cAdvisor client, which is
wrapped as `cadvisorClient` in Kubelet's own codebase. So it's convenient and
transparent to upper callers to get the stats directly in `cadvisorClient`
without accessing cadvisor. To keep consistency, the new logic will still invoke
cAdvisor functions to produce the filesystem stats.

### Cgroup Stats

Similarly, the Kubelet is capable of collecting cgroup stats directly by taking
advantage of cadvisor library --
`github.com/google/cadvisor/container/libcontainer`. Note, the root cgroup
collects network stats additionally.

### Windows

Windows currently does a best effort at filling out the stats in
`/stats/summary` and misses some stats either because those are not exposed or
they are not supported. To lower the risk and complexity of this KEP, the
changes won't be implemented for Windows platform.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

-   `pkg/kubelet/server/stats`: `<date>` - `<test coverage>`

##### Integration tests

-   <test>: <link to test coverage>

##### e2e tests

-   test/e2e_node/summary_test.go

### Graduation Criteria

#### Alpha

-   Feature implemented behind the feature gate `KubeletNodeCgroupStats`
-   Initial e2e tests completed and enabled

#### Beta

-   `KubeletNodeCgroupStats` feature gate is enabled by default
-   Gather feedback and ensure no breaking changes

#### GA

-   Feature gate removed

### Upgrade / Downgrade Strategy

### Version Skew Strategy

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

-   [x] Feature gate (also fill in values in `kep.yaml`)
    -   Feature gate name: KubeletNodeCgroupStats
    -   Components depending on the feature gate: Kubelet

###### Does enabling the feature change any default behavior?

When relying on cAdvisor collecting cgroup stats, cAdvisor collects them on
asynchronously and returns stats from most recent collections. While this KEP
will make Kubelet collect them on deman so customers might notice subtle
difference in timestamp of responded stat data, which should have little impact.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

###### What happens if we reenable the feature if it was previously rolled back?

There should be no problems

###### Are there any tests for feature enablement/disablement?

Following tests should be passed for both feature enabled and disabled.

-   The existing summary api e2e test
-   Disk eviction tests

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

There might be workloads depending on Node stats and/or system cgroup stats
returned by `/stats/summary`. If the correctness of those stats are critical to
those workloads, then it might impact them.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

The lack or discrenpancy of stat data returned by `/stats/summary`

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

NA

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

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

-   [ ] Events
    -   Event Reason:
-   [ ] API .status
    -   Condition name:
    -   Other field:
-   [ ] Other (treat as last resort)
    -   Details:

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

-   [ ] Metrics
    -   Metric name:
    -   [Optional] Aggregation method:
    -   Components exposing the metric:
-   [ ] Other (treat as last resort)
    -   Details:

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

No
<!-- Describe them, providing: - API call type (e.g. PATCH pods) - estimated throughput - originating component(s) (e.g. Kubelet, Feature-X-controller) Focusing mostly on: - components listing and/or watching resources they didn't before - API calls that may be triggered by changes of some Kubernetes resources (e.g. update of object X triggers new updates of object Y) - periodic API calls to reconcile state (e.g. periodic fetching state, heartbeats, leader election, etc.) -->

###### Will enabling / using this feature result in introducing new API types?

No
<!-- Describe them, providing: - API type - Supported number of objects per cluster - Supported number of objects per namespace (for namespace-scoped objects) -->

###### Will enabling / using this feature result in any new calls to the cloud provider?

No <!-- Describe them, providing: - Which API(s): - Estimated increase: -->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No
<!-- Describe them, providing: - API type(s): - Estimated increase in size: (e.g., new annotation of size 32B) - Estimated amount of new objects: (e.g., new Object X for every existing Pod) -->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No <!-- Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between (e.g.
need to do X to start a container), etc. Please describe the details.

\[existing SLIs/SLOs]:
https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

-   Instead, it most likely will reduce resource usage since it will remove
    running cadvisor tasks. <!-- Things to keep in mind include: additional
    in-memory state, additional non-trivial computations, excessive access to
    disks (including increased log volume), significant amount of data sent
    and/or received over network, etc. This through this both in small and large
    cases, again with respect to the [supported limits].

\[supported limits]:
https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No <!-- Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming
resources, etc.). If any of the resources can be exhausted, how this is
mitigated with the existing limits (e.g. pods per node) or new limits added by
this KEP?

Are there any tests that were run/should be run to understand performance
characteristics better and validate the declared limits? -->

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

NA

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

*   Tunning cAdvisor parameter to have comparable performance.
*   Optimizing cAdvisor metric collection.

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
