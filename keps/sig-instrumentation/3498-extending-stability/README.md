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
# KEP-3498: Extending Metrics Stability

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

The metric stability framework was originally introduced with the intent of safeguarding significant metrics from being broken downstream. Metrics could be deemed `stable` or `alpha`, and only `stable` metrics would have stability guarantees. 

This KEP intends to propose additional stability classes to extend on the existing metrics stability framework, such that we can achieve parity with the various stages of the feature release cycle.

## Motivation

It's become more obvious recently that we need additional stability classes, particularly in respect to various stages of feature releases. This has become more obvious with the advent of PRR (production readiness reviews) and [mandated production readiness metrics](https://github.com/kubernetes/community/blob/cf715a4404c4cefcfb52278bc128b7a765373fc7/sig-architecture/production-readiness.md#common-feedback-from-reviewers). 

### Goals

Introduce two more metric classes: `beta`, corresponding to the `beta` stage of feature release, and `internal` which corresponds to internal development related metrics. 

### Non-Goals

- establishing if specific metrics fall into a stability class, this exercise is left for component owners, who own their own metrics

## Proposal

We're proposing adding additional metadata fields to Kubernetes metrics. Specifically we want to add the following stability levels:

- `Internal` - representing internal usages of metrics (i.e. classes of metrics which do not correspond to features) or low-level metrics that a typical operator would not understand (or would not be able to react to them properly).
- `Beta` - representing a more mature stage in a feature metric, with greater stability guarantees than alpha or internal metrics, but less than `Stable`

We also propose amending the semantic meaning of an `Alpha` metric such that it represents the nascent stage of a KEP-proposed feature, rather than the entire class of metrics without stability guarantees. 

Additionally we propose forced upgrades of metrics stability classes in the similar vein that features are not allowed to languish in `alpha` or `beta` stages, but this feature will not be available until the beta version of this KEP. For the alpha version of this KEP, we will implement the necessary changes to Kubernetes metrics framework, such that it supports the additional classes of metrics, without making changes to any existing metrics or their stability levels. As such, this KEP proposes changes to the metrics pipeline and the static analysis pieces of Kubernetes metrics framework. 


### Risks and Mitigations

The primary risk is that these changes break our existing (and working) metrics infrastructure. The mitigation should straightfoward, i.e. rollback the changes to the metrics framework. 
## Design Details

Our plan is to add functionality to our static analysis framework which is hosted in the main `k8s/k8s` repo, under `test/instrumentation`. Specifically, we will need to support:

- parsing variables
- multi-line strings
- evaluating buckets
- buckets which are defined via variables and consts
- evaluation of simple consts
- evaluation of simple variables

We will not attempt to parse metrics which:

- are constructed dynamically, i.e. through function calls which use function arguments as parameters in metric definitions, since some of those cannot be resolved until runtime. 
- are constructed using custom prometheus collectors, for the same reasons as above. 

As an aside, much of this work has already been done, but is stashed in a local repo. 

### Test Plan

We have static analysis testing for stable metrics, we will extend our test coverage 
to include metrics which are `ALPHA` and `BETA` while ignoring `INTERNAL` metrics.

[ X ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

We already have thorough testing for the stability framework which has been GA for years.

##### Unit tests

[ ] parsing variables
[ ] multi-line strings
[ ] evaluating buckets
[ ] buckets which are defined via variables and consts
[ ] evaluation of simple consts
[ ] evaluation of simple variables

- `test/instrumentation`: `09/20/2022` - `full coverage of existing stability framework`

##### Integration tests

We will test the static analysis parser on a test directory with all permutations of metrics which we expect to parse (and variants we expect not to be able to parse)

##### e2e tests

The statis analysis tooling runs in a precommit pipeline and is therefore exempt from runtime tests.

### Graduation Criteria


#### Alpha

- Kubernetes metrics framework will be enhanced to support additional stability classes
- The static analysis pipeline of the metrics framework will be enhanced to understand how to parse more things (these are listed above)

#### Beta

- All instances of `Alpha` metrics will be converted to `Internal`
- Kubernetes metrics framework will be enhanced to support marking `Alpha` and `Beta` metrics with a date. The semantics of this are yet to be determined. This date will be used to statically determine whether or not that metric should be decrepated automatically or promoted.
- Kubernetes metrics framework will be enhanced with a script to auto-deprecate metrics which have passed their window of existence as an `Alpha` or `Beta` metric
- We will determine the semantics for `Alpha` and `Beta` metrics
- The `beta` stage for this framework will be a few releases. During this time, we will evaluate the utility and the ergonomics of the framework, making adjustments as necessary

#### GA

- We will allow bake time before promoting this feature to GA
- At this stage, we will promote our meta-metric for registered metrics to Stable

#### Deprecation

- This section will pertain to the deprecation policy of deprecated `Alpha` and `Beta` metrics which will be determined in the `Beta` version of this KEP.


### Upgrade / Downgrade Strategy

The static analysis code does not run in Kubernetes runtime code, with the exception of the registered_metrics metric. 

### Version Skew Strategy

This feature does not require a version skew strategy.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

This feature cannot be enabled or rolled back. It is built into the infrastructure of metrics, which will support two additional values for the enumeration of stable classes of metrics. 

###### How can this feature be enabled / disabled in a live cluster?

It cannot. This is purely infrastructure based and requires adding additional enumeration values to metrics stability classes. 

###### Does enabling the feature change any default behavior?

It will cause metrics previously annotated as `Alpha` metrics to be denoted as `Internal`.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

No.

###### What happens if we reenable the feature if it was previously rolled back?

N/A

###### Are there any tests for feature enablement/disablement?

No. 

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

This should not affect rollout. It could affect workloads that depended on `Alpha` metrics, which will be recagetorized as `Internal`. But to be fair, we've already explicitly laid out the fact that `Alpha` metrics do not have stability guarantees. 

###### What specific metrics should inform a rollback?

`registered_metrics_total` summing to zero. 

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This should not affect upgrade/rollback paths.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

`Alpha` metrics will be recategorized as `Internal`.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

You can determine this by seeing if workloads depend on any Kubernetes control-plane metrics. If they do, they are using this feature.

###### How can someone using this feature know that it is working for their instance?

They will be able to see metrics. 


###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This tooling runs in precommit. It does not affect runtime SLOs.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

`registered_metrics_total` will be used to calculate the number of registered stable metrics.

### Dependencies

Prometheus and the Kubernetes metric framework.

###### Does this feature depend on any specific services running in the cluster?

In order to ingest these metrics, one needs a prometheus scraping agent and some backend to persist the metric data.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No. 

###### Will enabling / using this feature result in any new calls to the cloud provider?

No. 

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No. 

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. 

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. 

### Troubleshooting


###### How does this feature react if the API server and/or etcd is unavailable?

Apiserver needs to be available to scrape metrics, if etcd is not available, you may still be able to scrape metrics from the apiserver. 

###### What are other known failure modes?

Runaway cardinality of metrics, but that is orthogonal to the scope of this KEP.

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

This introduces complexity to metrics stability levels, however this has been asked for by various community members over the past few years. And we, as a community, are moving towards requiring metrics as a prerequisite for KEPs, which this should basically align with. 

## Alternatives

Doing nothing is a viable alternative. However, we end up in a weird spot with feature metrics, where they have no guarantees or are fully stable. 

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
