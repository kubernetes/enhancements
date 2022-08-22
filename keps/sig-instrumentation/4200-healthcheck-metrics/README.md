<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ X ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ X ] **Create an issue in kubernetes/enhancements**
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
# KEP-4200: Healthcheck metrics

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
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Monitoring Requirements](#monitoring-requirements)
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
so that monitoring agents can consume healthz/livez/readyz data and create SLOs/alerts off
of these inputs.

## Motivation

Healthchecking data is currently surfaced in unstructured format and is scraped by monitoring
agents (as well as the kubelet), which must be configured to interpret the health data and
to act upon it. This process does not lend itself readily to the creation of availability SLOs,
since we basically require an outside agent to parse the health endpoint and convert this signal
into an SLI. 

### Goals

- Create a uniform interface by which we can consume health checking information
- Allow availability SLIs to be created without a specialized monitoring agent


### Non-Goals

- Creation of SLOs are out of scope. This KEP specifically targets the creation of signals which can be used as SLIs.

## Proposal

We are proposing to add a new endpoint on Kubernetes components `/health/metrics` which returns
health check data in prometheus format.


### Risks and Mitigations

This is a separate endpoint, it does not have to be used. The risk of adding metrics is generally cardinality, but in this
case we are proposing known dimensions to the metrics, specifically:

1. status - one of `Success`, `Error`, `Pending`
1. type - one of `livez`, `readyz`, `healthz`
1. name - the known name of the health check. AFAIK, these are all static strings in the Kubernetes codebase, therefore bounded in cardinality


## Design Details


### Test Plan


##### Unit tests

- [ ] ensure that healthcheck states are reset for gauges on write
- [ ] ensure that counters properly retain state of all seen healthchecks

##### Integration tests

- [ ] ensure existence of healthcheck endpoint

##### e2e tests


- [ ] ensure existence of healthcheck endpoint


### Graduation Criteria


## Production Readiness Review Questionnaire


### Feature Enablement and Rollback

We can feature gate the api and the registry for this metrics endpoint, such that it is disabled by default and toggleable only by feature-gate. Given the lazy
nature of the Kubernetes metric framework, this will ensure that we do not encounter any memory leaks.


###### How can this feature be enabled / disabled in a live cluster?

Via feature gate.


###### Does enabling the feature change any default behavior?

Yes, it exposes an endpoint which can then be scraped.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, but it will remove the endpoint.

### Monitoring Requirements

I am proposing a metric/series of metrics here.

### Scalability

We already hit the readyz/healthz/livez endpoints of our control-plane components frequently, this KEP only 
adds instrumentation of these endpoints' results.

###### Will enabling / using this feature result in any new API calls?

Yes, we are proposing that this health metrics are surfaced in each component under `/health/metrics` which 
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

No/

### Troubleshooting


###### How does this feature react if the API server and/or etcd is unavailable?

If apiserver is unavailable, then you will not be able to ingest the metrics from the apiserver.
However, the failure of etcd should allow you to scrape the metrics from apiserver, so long as 
it is otherwise healthy. 


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
