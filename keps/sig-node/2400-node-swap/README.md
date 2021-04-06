# KEP-2400: Node system swap support

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
- [Motivation](#motivation)
  - [Scenarios](#scenarios)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Improved Node Stability](#improved-node-stability)
    - [Long-running applications that swap out startup memory](#long-running-applications-that-swap-out-startup-memory)
    - [Memory Flexibility](#memory-flexibility)
    - [Local development and systems with fast storage](#local-development-and-systems-with-fast-storage)
    - [Low footprint systems](#low-footprint-systems)
    - [Virtualization management overhead](#virtualization-management-overhead)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
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
  - [Just set <code>--fail-swap-on=false</code>](#just-set-)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
- [ ] (R) Graduation criteria is in place
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

Kubernetes currently does not support the use of [swap memory](https://en.wikipedia.org/wiki/Paging#Linux) on Linux, as it is difficult to provide guarantees and account for pod memory utilization when swap is involved. As part of Kubernetes’ earlier design, [swap support was considered out of scope](https://github.com/kubernetes/kubernetes/issues/7294).

However, there are a [number of use cases](#user-stories) that would benefit from Kubernetes nodes supporting swap. Hence, this proposal aims to add swap support to nodes in a controlled, predictable manner so that Kubernetes users can perform testing and provide data to continue building cluster capabilities on top of swap.

## Motivation

There are two distinct types of user for swap, who may overlap:
- node administrators, who may want swap available for node-level performance tuning and stability/reducing noisy neighbour issues
- application developers, who have written applications that would benefit from using swap memory

There are hence a number of possible ways that one could envision swap use on a node.

### Scenarios

1. Swap is enabled only at the system level. The CRI does not permit user workloads to use swap. (This scenario is a prerequisite for the following use cases.)
1. Swap is enabled at the node level. The CRI can be globally configured to permit user workloads scheduled on the node to use some quantity of swap.
1. Swap is set on a per-workload basis. The CRI sets permitted swap utilization on each individual workload.

This KEP will be limited in scope to the first two scenarios. The third can be addressed in a follow-up KEP. The enablement work that is in scope for this KEP will be necessary to implement the third scenario.


### Goals

- On Linux systems, when swap is provisioned and available, Kubelet can start up with swap on.
- Configuration is available for CRI to set swap utilization available to Kubernetes workloads, defaulting to 0 swap.
- Cluster administrators can enable and configure CRI swap utilization on a per-node basis.

### Non-Goals

- Provisioning swap. Swap must already be available on the system.
- Allocating swap on a per-workload basis with accounting (e.g. pod-level specification of swap). If desired, this should be designed and implemented as part of a follow-up KEP. This KEP is a prerequisite for that work.
- Supporting zram, zswap, or other memory types like SGX EPC. These could be addressed in a follow-up KEP, and are out of scope.

## Proposal

I propose that, when swap is provisioned and available on a node, we allow cluster administrators to configure the Kubelet and CRI such that:

- The kubelet can start with swap on.
- The CRI is updated such that by default, workloads will use 0 swap.
- The CRI will have configuration available such that swap utilization can be configured for the entire node (e.g. as a percentage of pod memory requests).

This proposal enables scenarios 1 and 2 above, but not 3.

### User Stories

#### Improved Node Stability

cgroupsv2 improved memory management algos, such as oomd, currently require swap. Hence, having a small amount of swap available on nodes could improve better resource pressure handling and recovery.

- https://www.kernel.org/doc/html/latest/admin-guide/cgroup-v2.html#id1
- https://chrisdown.name/2018/01/02/in-defence-of-swap.html
- https://media.ccc.de/v/ASG2018-175-oomd
- https://github.com/facebookincubator/oomd/blob/master/docs/production_setup.md#swap

This user story is addressed by scenario 1 and 2, and could benefit from 3.

#### Long-running applications that swap out startup memory

- Applications such as the Java and Node runtimes rely on swap for optimal performance https://github.com/kubernetes/kubernetes/issues/53533#issue-263475425
- Initialization logic of applications can be safely swapped out without affecting long-running application resource usage https://github.com/kubernetes/kubernetes/issues/53533#issuecomment-615967154

This user story is addressed by scenario 2, and could benefit from 3.

#### Memory Flexibility

This user story addresses cases in which cost of additional memory is prohibitive, or elastic scaling is impossible (e.g. on-premise/bare metal deployments).

- Occasional cron job with high memory usage and lack of swap support means cloud nodes must always be allocated for maximum possible memory utilization, leading to overprovisioning/high costs https://github.com/kubernetes/kubernetes/issues/53533#issuecomment-354832960
- Lack of swap support would require provisioning 3x the amount of memory as required with swap https://github.com/kubernetes/kubernetes/issues/53533#issuecomment-617654228
- On-premise deployment can’t horizontally scale available memory based on load https://github.com/kubernetes/kubernetes/issues/53533#issuecomment-637715138
- Scaling resources is technically feasible but cost-prohibitive, swap provides flexibility at lower cost https://github.com/kubernetes/kubernetes/issues/53533#issuecomment-553713502

This user story is addressed by scenario 2, and could benefit from 3.

#### Local development and systems with fast storage

Local development or single-node clusters and systems with fast storage may benefit from using available swap (e.g. NVMe swap partitions, one-node clusters).

- Single node, local Kubernetes deployment on laptop https://github.com/kubernetes/kubernetes/issues/53533#issuecomment-361748518
- Linux has optimizations for swap on SSD, allowing for performance boosts https://github.com/kubernetes/kubernetes/issues/53533#issuecomment-589275277

This user story is addressed by scenarios 1 and 2, and could benefit from 3.

#### Low footprint systems

For example, edge devices with limited memory.

- Edge compute systems/devices with small memory footprints (\<2Gi) https://github.com/kubernetes/kubernetes/issues/53533#issuecomment-751398086
- Clusters with nodes \<4Gi memory https://github.com/kubernetes/kubernetes/issues/53533#issuecomment-751404417

This user story is addressed by scenario 2, and could benefit from 3.

#### Virtualization management overhead

This would apply to virtualized Kubernetes workloads such as VMs launched by kubevirt.

Every VM comes with a management related overhead which can sporadically be pretty significant (memory streaming, SRIOV attachment, gpu attachment, virtio-fs, …). Swap helps to not request much more memory to deal with short term worst-case scenarios.

With virtualization, clusters are typically provisioned based on the workloads’ memory consumption, and any infrastructure container overhead is overcommitted. This overhead could be safely swapped out.

- Required for live migration of VMs https://github.com/kubernetes/kubernetes/issues/53533#issuecomment-754878431

This user story is addressed by scenario 2, and could benefit from 3.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

Having swap available on a system reduces predictability. When swap is available to workloads, and is not accounted for on an individual workload-by-workload basis

First, this risk is mitigated by preventing any workloads from using swap by default, even if it is enabled on a system. This will allow a cluster administrator to test swap utilization just at the system level without introducing unpredictability to workload resource utilization.

Additionally, we mitigate this risk by quantifying system stability and then gathering test and production data to determine if system stability remains the same or is improved when swap is available to the system and/or workloads.

Since swap provisioning is out of scope of this proposal, this enhancement poses little risk to Kubernetes clusters that will not enable swap.

## Design Details

\[In progress\]

Need to add specifics here for:

- Changes to `--fail-on-swap` flag
- CRI config details
- Where changes will need to be made so that dockershim and the CRI are consistent with swap control

### Test Plan

For alpha:

- Swap scenarios are enabled in test-infra for at least two Linux distributions. e2e suites will be run against them.
- Data should be gathered from a number of use cases to guide beta graduation and further development efforts.

Once this data is available, additional test plans should be added for the next phase of graduation.

### Graduation Criteria

#### Alpha

- Kubelet can be started with swap enabled.
- KubeletConfig allows CRI to be configured with a percentage of swap available to workloads. This will default to 0.
- e2e test jobs are configured for Linux systems with swap enabled.


#### Beta

(Tentative.)

- Determine a set of metrics for node QoS in order to evaluate the performance of nodes with and without swap enabled.
- Collect feedback from test user cases.
- Improve coverage for appropriate scenarios in testgrid.

#### GA

- Remove feature flag.

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

No changes are required on upgrade to maintain previous behaviour.

It is possible to downgrade a kubelet on a node that was using swap, but this would require disabling the use of swap and setting `swapoff` on the node.

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

Feature flag will apply to kubelet only, so version skew strategy is N/A.

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
-->

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: NodeSwapEnabled
  - Components depending on the feature gate: Kubelet
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No. If the feature flag is enabled, the user must still set `--fail-swap-on=false` to adjust the default behaviour.

A node must have swap provisioned and available for this feature to work. If there is no swap available, but the feature flag is set to true, there will still be no change in existing behaviour.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

No. I don’t think it makes much sense to be able to provide users a meaningful ability to disable the feature flag at runtime, as this would be highly disruptive to workloads and difficult to implement. To turn this off, the kubelet would need to be restarted.

###### What happens if we reenable the feature if it was previously rolled back?

N/A

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->

N/A. This should be tested separately for scenarios with the flag enabled and disabled.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?
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
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
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

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

<!--
At a high level, this usually will be in the form of "high percentile of SLI
per day <= X". It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code
-->

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

No.

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

No.

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

The KubeletConfig API object may slightly increase in size due to new config fields.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

It is possible for this feature to affect performance of some worker node-level SLIs/SLOs. We will need to monitor for differences, particularly during beta testing, when evaluating this feature for beta and graduation.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

Yes. It will permit the utilization of swap memory (i.e. disk) on nodes. This is expected, as this enhancement is enabling cluster administrators to access this resource.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

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

- **2015-04-24:** Discussed in [#7294](https://github.com/kubernetes/kubernetes/issues/7294).
- **2017-10-06:** Discussed in [#53533](https://github.com/kubernetes/kubernetes/issues/53533).
- **2021-01-05:** Initial design discussion document for swap support and use cases.
- **2021-04-05:** Alpha KEP drafted for initial node-level swap support and implementation (KEP-2400).

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

When swap is enabled, particularly for workloads, the kubelet’s resource accounting may become much less accurate. This may make cluster administration more difficult and less predictable.

Currently, there exists an unsupported workaround, which is setting the kubelet flag `--fail-swap-on` to false.

## Alternatives

### Just set `--fail-swap-on=false`

This is insufficient for most use cases because there is inconsistent control over how swap will be used by various container runtimes. Dockershim currently sets swap available for workloads to 0. The CRI does not restrict it at all. This inconsistency makes it difficult or impossible to use swap in production, particularly if a user wants to restrict workloads from using swap when using the CRI rather than dockershim.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

We may need Linux VM images built with swap partitions for e2e testing in CI.
