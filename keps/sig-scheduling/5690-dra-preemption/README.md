<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-5690: DRA Preemption

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
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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
Dynamic Resource Allocation (DRA) was promoted to stable in v1.35, but one major capability—preemption of pods
utilizing DRA resources—is currently not supported. Under the current state, lower-priority workloads that
happen to be scheduled early can occupy scarce devices indefinitely, leaving higher-priority workloads
pending. By introducing native preemption support in DRA, we equip the scheduler with the necessary
capabilities to ensure that the most critical workloads gain access to scarce hardware resources.

## Motivation
Acquisition and allocation of specialized hardware (e.g., GPUs and TPUs) is a primary operational concern for
AI/ML users. Because accelerator resources are frequently constrained, it is essential that Kubernetes
provides robust tools to run high-priority workloads on the available hardware. Introducing preemption
support for DRA ensures that the scheduler can reclaim resources from lower-priority workloads to satisfy
scheduling requirements of higher-priority tasks.

### Goals
* Enable the scheduler to preempt pods consuming node-local DRA devices.

### Non-Goals
* Support preemption of workloads utilizing PodGroup-level ResourceClaims.
* Support preemption of workloads using multi-node or network-attached devices.

## Proposal
We will implement the `fwk.PreFilterExtensions` interface in the `dynamicresources` scheduler plugin, which
requires implementing the `AddPod` and `RemovePod` methods. These functions are invoked by the core
`DefaultPreemption` plugin to incrementally simulate the removal or recovery (reprieval) of candidate victim
pods during the preemption planning loop.
Implementing these functions requires updating the internal `stateData` structure in the `dynamicresources`
plugin to track state transitions and maintain local capacity maps transactionally throughout the preemption
simulation.
The scheduler must correctly handle several advanced DRA features during this simulation, or explicitly
bypass preemption if those features are in use.

Features that we will support but require careful implementation:
* **Partitionable Devices**: Freeing up the exact device partition requested by a higher-priority workload
  may require preempting multiple lower-priority pods that collectively occupy fractional partitions
  (represented as counters) on the same physical device, so that the raw capacity can be consolidated
  and re-partitioned.
* **Consumable Capacity**: We might need to free up just a subset of the capacity on a device to satisfy
  the request of a higher-priority pod. We need to make sure the remaining capacity on a device is correctly
  tracked during preemption simulations, and that we don't overcommit.
* **Device Binding Conditions**: Pods might be waiting for a resource binding condition, so we want to make
  sure this is handled correctly during preemption simulations.

Features/scenarios that we will not support:
* **ResourceClaims that span multiple nodes**: The default preemption algorithm only considers preempting
  pods on a single node, meaning that it will not be able to free up devices that are allocated to claims
  referenced by pods spanning multiple nodes.
* **ResourceClaims for PodGroups**: These claims are allocated to a PodGroup, meaning that all pods in the
  group must be scheduled together. The dynamicresources plugin will not support preemption for
  ResourceClaims allocated at the PodGroup level.


### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
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

<!--
Generated with:
go test -cover ./pkg/scheduler/framework/plugins/dynamicresources/... | sed -e 's/.*\(k8s.io[a-z/-]*\).*coverage: \(.*\) of statements/- `\1`: \2/' | sort
-->

- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/dynamicresources`: 82.4%

##### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)
-->

Integration tests will be added that covers the normal functionality of preemption with DRA, with
specific tests covering all the special scenarios called out above.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.
-->

E2e tests will be added to cover the basic scenarios for preemption with DRA. The more complicated
scenarios will be handled by integration tests.

### Graduation Criteria

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Additional tests are in Testgrid and linked in KEP

#### GA

- 2 examples of real-world usage
- Allowing time for feedback


### Upgrade / Downgrade Strategy

Standard upgrade/downgrade strategies may be used, no special configuration
changes are needed. The changes are local to the scheduler.

### Version Skew Strategy

This feature only impacts the scheduler, so there will be no version skew issues.

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

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: DRAPreemption
  - Components depending on the feature gate:
    - kube-scheduler


###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

Yes, it will cause the scheduler to preempt pods referencing ResourceClaims to
make room for higher priority pods. There is no API change for this feature, so it
will potentially impact any Pod in the cluster when the feature is enabled.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, disabling the feature will prevent the scheduler from preempting pods that
reference ResourceClaims.

###### What happens if we reenable the feature if it was previously rolled back?

That it was previously rolled back have no impact. Reenabling it just means that
preemption of pods referencing ResourceClaims will again be considered.

###### Are there any tests for feature enablement/disablement?

We will cover this scenario in both unit tests and integration tests.

### Rollout, Upgrade and Rollback Planning


###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout will immediately make pods referencing ResourceClaims eligible for
preemption. So if there are higher priority pods that can't be scheduled due
to pending ResourceClaims, the result will be that lower priority pods will
get preempted.

Similarly, a rollback means that pods referencing ResourceClaims will no longer
be eligible for preemption. But already preempted workload will remain in the
pending state until sufficient resources are freed up.

###### What specific metrics should inform a rollback?

The specific signal that would suggest this feature should be rolled back, would
be if pods are being preempted when they shouldn't be. This means there is a bug
somewhere in the implementation.

If the `scheduler_preemption_victims` increases significantly when the feature is enabled, but we don't 
see a corresponding increase in pods being scheduled, we should investigate. It would suggest
that pods are being preempted incorrectly and the higher-priority pods are not actually
being scheduled.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

It will be tested by bringing up a KinD cluster and enabling the feature gate.
We will then run a simple workload with higher priority pods and then enable
the feature gate to verify that the lower priority pods are preempted. We will then 
disable the feature gate and verify that another pod with an even higher priority
does not preempt the existing pod.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

It is enabled on a cluster-level, so if it is enabled, it is in use. It will only impact
pods that are referencing ResourceClaims.

###### How can someone using this feature know that it is working for their instance?

When a Pod is preempted, it can be observed in the following ways:

- [x] Events
  - Event Reason: Preempted
- [x] API .status
  - Condition name: DisruptionTarget

Seeing this on a Pod that has references ResourceClaims whows that the feature is working.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Kubernetes does not have SLOs for preemption, so neither will this enhancement.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [x] Metrics
  - Metric name: `scheduler_preemption_victims` for pod that references ResourceClaims
  - Components exposing the metric: kube-scheduler

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No

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

No, the preemption simulation will run using the existing state for the
dynamicresources scheduler plugin.

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. It might lead to increase time taken to check if lower priority pods can
be preempted to make room for a higher priority pod, since we don't do that
today.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

It can lead to some additional work in the scheduler, since we are enabling
preemption simulation for a new scheduler plugin.

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

This feature doesn't add any additional calls to the API server compared to
what the dynamicresources scheduler plugin already does. So if the API server
is unavailable, the scheduler will behave as it would without this feature
enabled.

###### What are other known failure modes?

None known at this point.

###### What steps should be taken if SLOs are not being met to determine the problem?

There are no SLOs for this feature.

## Implementation History

1.37: first KEP revision and implementation directly to Beta

## Drawbacks

It complicates the logic in the dynamicresources plugin and can lead
to slower preemption when pods are using DRA.

## Alternatives

None, Kubernetes has a well-established framework for doing preemption.

## Infrastructure Needed (Optional)

No
