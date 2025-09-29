# KEP-4569: Moving cgroup v1 support into maintenance mode

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Introduction of cgroup version metric](#introduction-of-cgroup-version-metric)
  - [Implementing a warning log and an event for cgroup v1 usage](#implementing-a-warning-log-and-an-event-for-cgroup-v1-usage)
  - [Introduce a kubelet flag to disable cgroup v1 support](#introduce-a-kubelet-flag-to-disable-cgroup-v1-support)
  - [Code modifications for default cgroup assumptions](#code-modifications-for-default-cgroup-assumptions)
  - [Separation of cgroup v1 and cgroup v2 Code Paths](#separation-of-cgroup-v1-and-cgroup-v2-code-paths)
  - [API Changes](#api-changes)
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

Move cgroup v1 support in Kubernetes into maintenance mode, aligning with the industry's move towards cgroup v2 as the default for Linux kernel resource management and isolation.

## Motivation

The Linux kernel community has made cgroup v2 the focus for new features, offering better functionality, a more consistent interface, and improved scalability. As a result, major Linux distributions and projects like systemd are phasing out support for cgroup v1. This trend puts pressure on Kubernetes to follow suit.

By shifting cgroup v1 support to maintenance mode, Kubernetes can stay in line with these changes, ensuring compatibility and taking advantage of the improvements in cgroup v2. This transition encourages the use of a more secure and efficient technology while acknowledging that the broader ecosystem, including essential components like the Linux kernel and systemd, is moving beyond cgroup v1.

For those needing long-term support for cgroup v1, it's important to note that this KEP reflects a broader shift in the ecosystem. The reality is that many critical dependencies are moving to cgroup v2, making it necessary for Kubernetes to adapt accordingly.

### Goals

1. **Feature Freeze**: No new features will be added to the cgroup v1 support code. The existing functionality of cgroup v1 will be considered complete and stable.

2. **e2e Testing**: Maintain a set of e2e tests to ensure ongoing validation of cgroup v1 for the currently supported features

3. **Security Maintenance**: Kubernetes community may provide security fixes for Critical and Important CVEs related to cgroup v1  as long as the release is not in end of life.

4. **Best-Effort Bug Fixes**:
   - Address critical security vulnerabilities in cgroup v1 on priority.
   - Major bugs in cgroup v1 will be evaluated and potentially fixed if a feasible solution exists.
   - Acknowledging that some bugs, particularly those requiring substantial changes, may not be resolvable given the constraints around maintaining cgroup v1 support, some issues may need fixes in the kernel or other dependencies that may not happen and so will not be fixed.

5. **Migration Support**: Provide clear and practical migration guidance for users using cgroup v1, facilitating a smoother transition to cgroup v2.

6. **Enhancing cgroup v2 Support**: Address all known pending bugs in Kubernetes’ cgroup v2 support to ensure it reaches a level of reliability and functionality that encourages users to transition from cgroup v1.

### Non-Goals

Removing cgroup v1 support. Deprecation and removal will be addressed in a future KEP.

## Proposal

The proposal outlines a plan to move cgroup v1 support in Kubernetes into maintenance mode, encouraging the community and users to transition to cgroup v2.

### Risks and Mitigations

The primary risk involves potential disruptions for users who migrate to cgroup v2 with incompatible workloads.

Users depending on the following technologies will need to ensure they are using the specified versions or later, which support cgroup v2:

- OpenJDK / HotSpot: jdk8u372, 11.0.16, 15 and later
- NodeJs 20.3.0 or later
- IBM Semeru Runtimes: jdk8u345-b01, 11.0.16.0, 17.0.4.0, 18.0.2.0 and later
- IBM SDK Java Technology Edition Version (IBM Java): 8.0.7.15 and later
- If users run any third-party monitoring and security agents that depend on the cgroup file system, they need to update the agents to a version that supports cgroup v2.

Mitigations include providing clear documentation on the migration process, offering community support for common issues encountered during migration, and keeping the cgroup v1 support in the maintenance mode for allowing users additional time to switch to cgroup v2 without any major disruptions.


## Design Details

This enhancement outlines the steps required to transition existing cgroup v1 support into maintenance mode, and as such, no new feature gates are proposed in this document.

### Introduction of cgroup version metric

A new metric, `kubelet_cgroup_version`, is proposed. This metric will report values `1` or `2`, indicating whether the host is utilizing cgroup version `1` or `2`, respectively. The kubelet will assess the host's cgroup version at startup and emit this metric accordingly.

The introduction of this metric aims to streamline the process for cluster administrators in determining the cgroup version deployed across their hosts. This metric removes the need for manual node inspection, providing a clear insight into the cgroup version each node operates on.

### Implementing a warning log and an event for cgroup v1 usage

Starting from 1.31, during kubelet startup if the host is running on cgroup v1, kubelet will log a warning message like,

```golang
klog.Warning("cgroup v1 detected. cgroup v1 support has been transitioned into maintenance mode, please plan for the migration towards cgroup v2. More information at https://git.k8s.io/enhancements/keps/sig-node/4569-cgroup-v1-maintenance-mode")
```
and also emit a corresponding event,
```golang
eventRecorder.Event(pod, v1.EventTypeWarning, "CgroupV1", fmt.Sprint("cgroup v1 detected. cgroup v1 support has been transitioned into maintenance mode, please plan for the migration towards cgroup v2. More information at https://git.k8s.io/enhancements/keps/sig-node/4569-cgroup-v1-maintenance-mode"))
```
### Introduce a kubelet flag to disable cgroup v1 support

A new boolean kubelet flag, `--fail-cgroupv1`, will be introduced. By default, this flag will be set to `false` to ensure users can continue to use cgroup v1 without any issues. The primary objective of introducing this flag is to set it to `true` in CI, ensuring that all blocking and new CI jobs use only cgroup v2 by default (unless the job explicitly wants to run on cgroup v1).


### Code modifications for default cgroup assumptions

Code segments that default to cgroup v1 logic will be inverted to assume cgroup v2 as the default. This shift underscores the transition to cgroup v2.

Original Code Snippet:
```golang
	memLimitFile := "memory.limit_in_bytes"
	if libcontainercgroups.IsCgroup2UnifiedMode() {
		memLimitFile = "memory.max"
	}
```
Revised Code Snippet:
```golang
	memLimitFile := "memory.max"
	if !libcontainercgroups.IsCgroup2UnifiedMode() {
		memLimitFile = "memory.limit_in_bytes"
	}
```

### Separation of cgroup v1 and cgroup v2 Code Paths

Within [cgroup manager](https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/cm/cgroup_manager_linux.go) in the kubelet cgroup v1 and v2 code is intertwined. To maintain clear separation and facilitate focused maintenance, [CgroupManager](https://github.com/kubernetes/kubernetes/blob/b2a8ac15a0db0d3f2c7ae6c221ed56e2e3cde7fb/pkg/kubelet/cm/types.go#L60) interface will have cgroup v1 and v2 specific implementations respectively. The [existing](https://github.com/kubernetes/kubernetes/blob/b2a8ac15a0db0d3f2c7ae6c221ed56e2e3cde7fb/pkg/kubelet/cm/cgroup_manager_linux.go#L156) common implementation of `CgroupManager` will be split and a common code will be moved the helper functions.

### API Changes

N/A


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

All existing test jobs that use cgroup v2 should continue to pass without any flakiness. Additionally, test jobs for cgroup v1 must also continue to pass, as we will be modifying a significant amount of kubelet code and could inadvertently break v1 as well.

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

The respective kubelet subcomponents should have unit tests cases to handle cgroup v1 and v2.

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

N/A

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

1. Monitor both cgroup v1 and v2 CI jobs.

2. Ensure all features coverage by running all tests on cgroup v2 (while some may still run on cgroup v1 to test back compatibility)

3. Make cgroup v2 host mandatory for new e2e and node e2e tests.

### Graduation Criteria

#### Alpha

This feature won't follow the normal cycle of alpha->beta->GA, and will instead be all implemented in GA

#### Beta

This feature won't follow the normal cycle of alpha->beta->GA, and will instead be all implemented in GA

#### GA
- Kubelet detects the host using cgroup v1, it will not only log a warning message but also generate an event to highlight the cgroup v1 moving to maintenance mode.
- Introduce a new metric, `kubelet_cgroup_version`, to provide insights into the cgroup version utilized by the hosts.
- Introduce a boolean kubelet flag `--fail-cgroupv1` and set it to `false` by default.
- Blog post on advantages of using cgroup v2 with kubernetes.
- Code modifications for to assume cgroup v2 by default. Check [this]( #code-modifications-for-default-cgroup-assumptions) section for details.
- Ensure all features coverage by running all tests on cgroup v2 (while some may still run on cgroup v1 to test back compatibility)
- Make cgroup v2 host mandatory for new e2e and node e2e jobs. Set `--fail-cgroupv1` to `true` for those jobs.
- Fix all pending known bugs in cgroup v2 support in kubernetes.
- Separation of cgroup v1 and cgroup v2 Code Paths. Check [this](#separation-of-cgroup-v1-and-cgroup-v2-code-paths) section for details.

- Mark cgroup v1 support in maintenance mode in the [documenatation](https://kubernetes.io/docs/home/).

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

- For clusters upgrading to a version of Kubernetes where cgroup v1 is in maintenance mode, administrators should ensure that all nodes are compatible with cgroup v2 prior to upgrading. This might include operating system upgrades or workload configuration changes.
- Downgrading and switching to cgroup v1 requires careful consideration. If users rely on features that only work with cgroup v2, such as swap support, they will need to either discontinue using those features or keep their systems on cgroup v2.

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

Kubernetes components that interact with node cgroups should be tolerant of both cgroup v1 and cgroup v2. This includes kubelet, the container runtime interface (CRI) implementations, and any cloud-provider-specific agents running on the node.

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

N/A

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.
-->

N/A

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

N/A

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

N/A

###### What happens if we reenable the feature if it was previously rolled back?

N/A

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->

N/A.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?
-->

N/A

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

`kubelet_cgroup_version` metric should be used to make determine the cgroup version on the cluster nodes.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can someone using this feature know that it is working for their instance?

A Warning log as well as an event will be emitted about cgroup v1 maintenance mode when the hosts are still using cgroup v1 from 1.31 onwards.

User will also be able to probe the cgroup version on the hosts using the metric `kubelet_cgroup_version`.

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

Operators can use `kubelet_cgroup_version` metric to determine the version of the cgroup on the cluster hosts. They can also monitor the log and event as described in this [section](#implementing-a-warning-log-and-an-event-for-cgroup-v1-usage).

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

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

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

N/A

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

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

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

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

No.

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


Kubernetes components are compatible with both cgroup v1 and cgroup v2. The failure can occur within workload if it depends on the cgroup version and does not support the version used by the host. But such workload related failures are outside the scope of kubernetes.

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

- **2024-04-05:** KEP for moving cgroup v1 to maintenance mode.

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

Moving cgroup v1 to maintenance mode presents transitional challenges, including:

1. Operational Overhead: Migrating to cgroup v2 requires updating underlying hosts, imposing significant operational efforts.

2. Compatibility Concerns: Workloads or tools not yet adapted for cgroup v2 may experience compatibility issues, despite ongoing community efforts to ensure broad compatibility.


## Alternatives

An alternative to moving cgroup v1 into maintenance mode would be to continue its full support. However, this approach would prevent users from accessing the improvements and features available in cgroup v2. Additionally, it would expose them to risks as key subsystems such as systemd and major operating systems like RHEL 9 have already deprecated or are planning to deprecate cgroup v1 support. This could lead to compatibility and maintenance issues in the future.

Another option could be deprecating cgroup v1 support with the eventual goal of removing it altogether. While this approach might accelerate the adoption of cgroup v2, it could significantly impact users who rely on legacy versions of operating systems or kernels that remain supported under long-term support plans, potentially creating substantial challenges for maintaining their systems.