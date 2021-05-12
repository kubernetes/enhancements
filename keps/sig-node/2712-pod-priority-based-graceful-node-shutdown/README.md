# KEP-2712: Pod Priority Based Graceful Node Shutdown

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Implementation](#implementation)
  - [Migration from the Node graceful shutdown feature](#migration-from-the-node-graceful-shutdown-feature)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha Graduation](#alpha-graduation)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->


## Release Signoff Checklist


Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kubelet graceful shutdown should take the pod priority values into account to
determine the order in which the pods are stopped.

## Motivation

The [node graceful shutdown KEP](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2000-graceful-node-shutdown) added support to the kubelet to detect
that a node is shutting down and making sure that the pods are gracefully
stopped before allowing the shutdown to proceed. 

The feature added flags to specify the total time for shutdown and the
time to reserve for shutting down critical pods.

However, there is a need to allow more fine grained control over the
pod shutdown order beyond _critical_ and _regular_ pods.

Also, in general, kubernetes API design discourages hard coding anything by instances
of name. 

Instead of looking at the pod priority class names, we can instead
look at the pod priority class values to allow more control over pod
shutdown order.

### Goals

*   Make the kubelet use shutdown configuration based on pod priority values for
    graceful shutdown.

### Non-Goals

*   Non-Linux hosts aren't supported
*   Let users modify or change existing pod lifecycle or introduce new inner
    pod depencides / shutdown ordering
*   Provide guarantee to handle all cases of graceful node shutdown, for
    example abrupt shutdown or sudden power cable pull can’t result in graceful
    shutdown

## Proposal


### User Stories (Optional)


#### Story 1

*   As a cluster administrator, I can configure the nodes in my cluster to
    allocate different graceful shutdown durations for different pod priority
    value ranges to terminate them gracefully during node shutdown


### Implementation

This implementation builds on top of the node graceful shutdown feature
by introducing additional configuration. A new feature flag called
`PodPriortityBasedGracefulShutdown` will be added to control the behavior
of the kubelet.

We will describe the configuration by using an example. Say, the
following custom pod priority classes are created in a cluster:

|Pod priority class name|Pod priority class value|
|-----------------------|------------------------|
|custom-class-a         | 100000                 |
|custom-class-b         | 10000                  |
|custom-class-c         | 1000                   |
|regular/unset          | 0                      |

We could set kubelet configuration to stop the pods as:

|Pod priority class value|Shutdown period|
|------------------------|---------------|
| 100000                 |300 seconds    |
| 10000                  |180 seconds    |
| 1000                   |120 seconds    |
| 0                      |60 seconds     |

The above table implies that any pod with priority value >= 100000 will get
300 seconds to stop, any pod with value >= 10000 and < 100000 will get 180
seconds to stop, any pod with value >= 1000 and < 10000 will get 120 seconds to stop. 
Finally, all other pods will get 60 seconds to stop.

Note: We use priority values instead of names because k8s API design discourages
using names and the values are more portable as well.

One doesn't have to specify values corresponding to all of the classes. For
e.g. the config could also be

|Pod priority class value|Shutdown period|
|------------------------|---------------|
| 100000                 |300 seconds    |
| 1000                   |120 seconds    |
| 0                      |60 seconds     |


In the above case, the pods with custom-class-b will go into the same bucket
as custom-class-c for shutdown.

If there are no pods in a particular range, then the kubelet does not wait
for pods in that priority range. Instead, the kubelet immediately skips to the
next priority class value range.

If this feature is enabled and no configuration is provided, then no ordering
action will be taken. The rationale is to allow some users to opt out of this
if they are on a non-systemd distribution or have an older version of systemd
with which this feature won't work.

The feature relies on systemd inhibitor locks that were introduced in 
systemd version [183](https://lwn.net/Articles/499480/).

### Migration from the Node graceful shutdown feature

If a user configures `ShutdownGracePeriod` to 300 seconds and `ShutdownGracePeriodCriticalPods`
to 120 seconds, then it could be migrated to (note that the non-critical pods will
get the difference of total time and critical pods time):

|Pod priority class value|Shutdown period|
|------------------------|---------------|
| 2000000000             |180 seconds    |
| 0                      |120 seconds    |

Kubelet will be modified to only work with the config proposed in this KEP or the
Node shutdown KEP. If both are specified, then it will be treated as a configuration
error. If neither are specified, then Graceful Node Shutdown feature is disabled.

### Risks and Mitigations

Same as the graceful shutdown KEP.

## Design Details

The configuration will be controlled by a new Kubelet Config setting,
`kubeletConfig.PodPriorityShutdownGracePeriods`:

```
type PodPriorityShutdownGracePeriod struct {
	Priority int32
	ShutdownGracePeriodSeconds int64
}

type KubeletConfiguration struct {
  PodPriorityShutdownGracePeriods []PodPriorityShutdownGracePeriod
}
```

### Test Plan

*   Unit tests for kubelet of handling shutdown event in pod priority order.
*   New E2E tests to validate node graceful shutdown in pod priority order.

### Graduation Criteria

#### Alpha Graduation

* Implemented the feature for Linux (systemd) only
* Unit tests
  * Unit tests will mock out system components (i.e. systemd, inhibitors) for
    alpha

#### Alpha -> Beta Graduation

* Addresses feedback from alpha testers
* Sufficient E2E and unit testing

#### Beta -> GA Graduation

* Addresses feedback from beta
* Sufficient number of users using the feature
* Confident that no further API / kubelet config configuration options changes are needed
* Close on any remaining open issues & bugs

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

n/a

### Version Skew Strategy

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

n/a

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/20190731-production-readiness-review-process.md.

The production readiness review questionnaire must be completed for features in
v1.19 or later, but is non-blocking at this time. That is, approval is not
required in order to be in the release.

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

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: `PodPriorityBasedGracefulNodeShutdown`
    - Components depending on the feature gate:
      - `kubelet`
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
      - no
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
      - yes (will require restart of kubelet)

* **Does enabling the feature change any default behavior?**
  Any change of default behavior may be surprising to users or break existing
  automations, so be extremely careful here.

    * The main behavior change is that during a node shutdown, pods running on
      the node will be terminated gracefully. Note that the pod authors won't be
      able to control the graceful shutdown time of the node as it will be bounded
      by the config proposed in the KEP.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Also set `disable-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g., if this is a runtime
  feature, can it break the existing applications?).

    * Yes, the feature can be disabled by either disabling the feature gate. The kubelet
      could be restarted with the feature gate disabled without having to evict the 
      running pods.

* **What happens if we reenable the feature if it was previously rolled back?**

    * Kubelet will attempt to perform graceful termination of pods during a
        node shutdown using pod priority configuration.

* **Are there any tests for feature enablement/disablement?**
  The e2e framework does not currently support enabling or disabling feature
  gates. However, unit tests in each component dealing with managing data, created
  with and without the feature, are necessary. At the very least, think about
  conversion tests if API types are being modified.

    *   N/A

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?

This feature should not impact rollouts.

* **What specific metrics should inform a rollback?**

N/A.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.

The feature is part of kubelet config so updating kubelet config should
enable/disable the feature; upgrade/downgrade is N/A.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

No.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

Check if the feature gate and kubelet config settings are enabled on a node.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

N/A

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At a high level, this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide comprehensive guidance, but at the very
  high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

N/A.

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).

N/A.

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

No, this feature doesn't depend on any specific services running the cluster.
It only depends on systemd running on the node itself.

### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
  focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)

No.

* **Will enabling / using this feature result in introducing new API types?**
  Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)

No.

* **Will enabling / using this feature result in any new calls to the cloud
provider?**

No.

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**
  Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)

No.

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.

No.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data sent and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits].

No.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

The feature does not depend on the API server / etcd.

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

N/A.

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


## Infrastructure Needed (Optional)

<!-- Use this section if you need things from the project/SIG. Examples include
a new subproject, repos requested, or GitHub details. Listing these here allows
a SIG to get the process for these resources started right away.  -->
