<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

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
# KEP-NNNN: Your short, descriptive title

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
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
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

This KEP intends to allow us to emit health check data in a structured and consistent way,
so that monitoring agents can consume healthz/livez/readyz data and create SLOs (service level 
objectives) and alerts off of these SLIs (service level indicators).

## Motivation

Healthchecking data is currently surfaced in unstructured format and is scraped by monitoring
agents (as well as the kubelet), which must be configured to interpret the health data and
to act upon it. This process does not lend itself readily to the creation of availability SLOs,
since we basically require an outside agent to parse the health endpoint and convert this signal
into an SLI. 

### Goals

- Create a uniform interface by which we can consume health checking information
- Allow availability SLIs to be created without a specialized monitoring agent
- Allow for increased granularity (by configuring a more frequent interval) of health check metric data
- Minimize the diff involved for each Kubernetes component


### Non-Goals

- Creation of SLOs are out of scope. This KEP specifically targets the creation of signals which can be used as SLIs.

## Proposal

We are proposing to add a new endpoint on Kubernetes components `/metrics/health` which returns
health check data in prometheus format.


### Risks and Mitigations

This is a separate endpoint, it does not have to be used. The risk of adding metrics is generally cardinality, but in this
case we are proposing known dimensions to the metrics, specifically:

1. status - one of `Success`, `Error`, `Pending`
1. type - one of `livez`, `readyz`, `healthz`
1. name - the known name of the health check. AFAIK, these are all static strings in the Kubernetes codebase, therefore bounded in cardinality


## Design Details

When healthz/livez/readyz paths are accessed (not on a timer), they will record whatever they return
in a gauge metric. Admittedly, this has the downside of staleness though, since the health check 
data can be as stale as the length of the kubelet scrape interval. However, given our e2e tests 
configure [apiserver to 1s intervals](https://github.com/kubernetes/kubernetes/blob/master/cluster/gce/manifests/kube-apiserver.manifest#L58), it is reasonable to assume that other cloud-providers 
likely configure similar small scrape intervals, which means staleness should not realistically 
be much of an issue. However, in the case that the kubelet gets stuck, one can alert off of the
counter that we expose; if the counter stops incrementing, then we know that the health endpoint
is not getting hit and that our gauge data is too stale. It would therefore be prudent to set a
staleness alert off of the counter.

We considered fetching metric data when the metrics endpoint was hit, but this would introduce 
extra load against the health endpoint, which we took care to avoid. Alternatively, we considered
periodic polling of the metrics endpoint such that the metrics would be incremented only during this
periodic poll, but that change would be larger and would need to be implemented in each component
for each of their health check endpoints. 

Using a gaugeFunc would also preclude making the metric `stable`, since gaugeFuncs are
dynamic by nature and therefore cannot be parsed at compile time by the stability framework. Since
these metrics are intended to be used as component health SLIs, we want them to be able to
be promoted to `stable` status.

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

N/A Our exisiting feature is already thoroughly tested and has been GA for several years now
without any issues.

##### Unit tests

- [ X ] ensure that healthcheck states are reset for gauges on write
- [ X ] ensure that counters properly retain state of all seen healthchecks

- `staging/src/k8s.io/component-base/metrics/prometheus/health/metrics_test.go`: `09-21-2022` - `existing battery of tests for testing the metrics endpoint`

##### Integration tests

- [ ] ensure existence of healthcheck endpoint

- <test>: <link to test coverage>

##### e2e tests

- [ ] ensure existence of healthcheck endpoint

- <test>: <link to test coverage>

### Graduation Criteria



<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].
-->

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled
- Feature implemented for apiserver

#### Beta

- Gather feedback from developers 
- Feature implemented for other Kubernetes Components

#### GA

- Several cycles of bake-time
- Graduation of metrics to stable status


#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

### Upgrade / Downgrade Strategy

This is a new metrics endpoint and should not affect upgrade/downgrade strategy with the exception that
if you are scraping this endpoint, downgrading may remove this endpoint from the Kubernetes components
and you may end up missing these metrics.

### Version Skew Strategy

We do not plan to modify these metrics, so it should be safe for version skew. 

## Production Readiness Review Questionnaire


### Feature Enablement and Rollback

We will target this feature behind a flag `ComponentHealthSLIsFeatureGate`

###### How can this feature be enabled / disabled in a live cluster?


- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ComponentHealthSLIsFeatureGate`
  - Components depending on the feature gate:
    + apiserver
    + kubelet
    + scheduler
    + controller-manager
    + kube-proxy


###### Does enabling the feature change any default behavior?

Yes it will expose a new metrics endpoint.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. But it will remove the metrics endpoint

###### What happens if we reenable the feature if it was previously rolled back?

It will expose the metrics endpoint again

###### Are there any tests for feature enablement/disablement?

We intend to add them with our e2e tests.

### Rollout, Upgrade and Rollback Planning

I'm not sure these are super relevant for this feature.

###### How can a rollout or rollback fail? Can it impact already running workloads?

This feature should not cause rollout failures. If it does, we can disable the feature.

###### What specific metrics should inform a rollback?

I mean, we're literally introducing health metrics so those can be used to inform a rollback.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This should not be necessary, we're adding a new metrics endpoint with no dependencies. The rollback
simply removes the endpoint, so if scrapes were happening, they will just fail.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

I am proposing a metric/series of metrics here.

###### How can an operator determine if the feature is in use by workloads?

They can check their prometheus scrape configs.

###### How can someone using this feature know that it is working for their instance?

They can curl the apiserver's `metrics/health` endpoint. 

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This is intended to allow people to establish SLOs.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

This KEP introduces SLIs.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Yes, the exact metrics that this KEP proposes.

### Dependencies

Prometheus client and the Kubernetes metrics framework.

###### Does this feature depend on any specific services running in the cluster?

Yes, it depends on the Kubernetes components running in the cluster.

### Scalability

We already hit the readyz/healthz/livez endpoints of our control-plane components frequently, this KEP only 
adds instrumentation of these endpoints' results.

###### Will enabling / using this feature result in any new API calls?

Yes, we are proposing that this health metrics are surfaced in each component under `/metrics/health` which 
will have to be consumed for the feature to be useful. However, this should be relatively innocuous since 
it will an isolated endpoint strictly for the purpose of surfacing health metrics.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. This should, in theory, reduce calls to health endpoints since SLIs need to be calculated currently by directly hitting
and parsing out the results of our existing health check endpoints, which adds to the total number of calls (since kubelet 
also hits these endpoints). 

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

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

If apiserver is unavailable, then you will not be able to ingest the metrics from the apiserver.
However, the failure of etcd should allow you to scrape the metrics from apiserver, so long as 
it is otherwise healthy. 

###### What are other known failure modes?

If the metric is unbounded, then it can cause a memory leak. However, we are only propsing using
bounded label values so this should not be a problem. 

###### What steps should be taken if SLOs are not being met to determine the problem?

I mean this makes it possible to establish Kubernetes Component Health SLOs...

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

Slight increase of memory usage for components (i.e. the breadth of the prometheus metric label values).

## Alternatives

Status quo. Which also means you basically need to implement this in an external component.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
