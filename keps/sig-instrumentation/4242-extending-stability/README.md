## Table of Contents

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
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Monitoring Requirements](#monitoring-requirements)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The metric stability framework was originally introduced with the intent of safeguarding significant metrics from being broken downstream. Metrics could be deemed `stable` or `alpha`, and only `stable` metrics would have stability guarantees. 

This KEP intends to propose additional stability classes to extend on the existing metrics stability framework, such that we can achieve parity with the various stages of the feature release cycle.

## Motivation

It's become more obvious recently that we need additional stability classes, particularly in respect to various stages of feature releases. This has become more obvious with the advent of PRR (production readiness reviews) and mandated production readiness metrics. 

### Goals

Introduce two more metric classes: `beta`, corresponding to the `beta` stage of feature release, and `internal` which corresponds to internal development related metrics. 


### Non-Goals

- establishing if specific metrics fall into a stability class, this exercise is left for component owners, who own their own metrics


## Proposal

We're proposing adding additional metadata fields to Kubernetes metrics. Specifically we want to add the following stability levels:

- `Internal` - representing internal usages of metrics (i.e. classes of metrics which do not correspond to features)
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


### Graduation Criteria


#### Alpha

- Kubernetes metrics framework will be enhanced to support additional stability classes
- The static analysis pipeline of the metrics framework will be enhanced to understand how to parse more things (these are listed above)
- All instances of `Alpha` metrics will be converted to `Internal`

#### Beta

- Kubernetes metrics framework will be enhanced to support marking `Alpha` and `Beta` metrics with a date. The semantics of this are yet to be determined. This date will be used to statically determine whether or not that metric should be decrepated automatically or promoted.
- Kubernetes metrics framework will be enhanced with a script to auto-deprecate metrics which have passed their window of existence as an `Alpha` or `Beta` metric
- It is at this point, we will determine the longevity rules for `Alpha` and `Beta` metrics
- The `beta` stage for this framework will be a few releases. During this time, we will evaluate the utility and the ergonomics of the framework, making adjustments as necessary

#### GA


## Production Readiness Review Questionnaire

During the `alpha` stage of this KEP, we will not be making any user facing changes, except marking metrics as `Internal` which were previously `Alpha`. The stability guarantees of `Internal` metrics is the same as `Alpha` currently and therefore there will not be any changes to what users can expect from the metrics they are using. 

### Feature Enablement and Rollback

We can revert our changes if it breaks the metrics framework. But we will be adding testing coverage as we enhance the framework, so it is unlikely that this will need to occur.

###### How can this feature be enabled / disabled in a live cluster?

Metrics stability framework is an internal feature of Kubernetes. 

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, this can be rolled back.

###### What happens if we reenable the feature if it was previously rolled back?

Metrics will be annotated with `Internal` instead of `Alpha` and vice versa.

###### Are there any tests for feature enablement/disablement?

No.


### Monitoring Requirements

Well, this is a meta-monitoring improvement, so it's a strange thing to monitor. But I suppose we can add metrics around how many metrics are registered divided by stability-level and metric name.


###### How can someone using this feature know that it is working for their instance?

You will see metrics from your component. 


## Implementation History


## Drawbacks

This introduces complexity to metrics stability levels, however this has been asked for by various community members over the past few years. And we, as a community, are moving towards requiring metrics as a prerequisite for KEPs, which this should basically align with. 

## Alternatives

Doing nothing is a viable alternative. However, we end up in a weird spot with feature metrics, where they have no guarantees or are fully stable. 

