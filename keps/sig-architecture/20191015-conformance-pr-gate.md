---
title: Conformance Coverage PR Gate
authors:
  - "@hh"
owning-sig: sig-architecture
participating-sigs:
  - sig-testing
reviewers:
  - "@timothysc"
  - "@alejandrox1"
  - "@johnschnake"
approvers:
  - "@bgrant0607"
  - "@smarterclayton"
editor: TBD
creation-date: 2019-10-15
last-updated: 2019-10-15
status: draft
---

# Conformance Coverage PR Gate

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Representation of Behaviors](#representation-of-behaviors)
  - [Behavior and Test Generation Tooling](#behavior-and-test-generation-tooling)
    - [Handwritten Behaviour Scenarios](#handwritten-behaviour-scenarios)
  - [Coverage Tooling](#coverage-tooling)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Phase 1](#phase-1)
    - [Tying tests back to behaviors](#tying-tests-back-to-behaviors)
    - [kubetestgen](#kubetestgen)
  - [Phase 2](#phase-2)
  - [Graduation Criteria](#graduation-criteria)
  - [Future development](#future-development)
    - [Complex Storytelling combined with json/yaml](#complex-storytelling-combined-with-jsonyaml)
    - [Example patch test scenario](#example-patch-test-scenario)
    - [Generating scaffolding from Gherkin .feature files](#generating-scaffolding-from-gherkin-feature-files)
    - [Autogeneration of Test Scaffolding](#autogeneration-of-test-scaffolding)
    - [Combining gherkin with existing framework](#combining-gherkin-with-existing-framework)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Annotate test files with behaviors](#annotate-test-files-with-behaviors)
  - [Annotate existing API documentation with behaviors](#annotate-existing-api-documentation-with-behaviors)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary


We have a policy requiring Conformance Tests before promoting beta APIs to GA.
This policy needs automation to ensure we eventually have coverage for
everything. The proposal provides a method of measuring Conformance coverage and
placing a `/hold` on any PR which fails to maintain or increase said coverage.


## Motivation


Currently PRs are merging that do not meet the Conformance Test policy
requirement or accidentally decrease coverage by changes to how tests or
frameworks are written.


### Goals

* run per PR against kubernetes/kubernetes
* feedback with list of newly promoted API objects/fields lacking tests
* feedback with list of currently tested API objects/fields not tested
* soft gate with '/hold' (or similar merge blocking tag)

### Non-Goals

* hard gating
* definition conformance coverage

## Proposal

The proposal consists of four deliverables:
* prow-job that runs periodically/continously against master for baseline
* prow-job that waits for PR e2e test jobs to finish for comparision
* automated workflow for tagging / failing as a test for all PRs that reduce coverage percentage

### Baseline Job against master
### Comparision Job per PR
### Tagging / Failing PRs reducing coverage

### Risks and Mitigations

The behavior definitions may not be properly updated if a change is made to a
feature, since these changes are made in very different areas in the code.
However, given that the behaviors defining conformance are generally stable,
this is not a high risk.

## Design Details

Delivery of this KEP shall be done in the following phases:

### Phase 1

In Phase 1, we will:
* Create a periodic prow-job giving us a baseline for coverage.

### Phase 2

In Phase 2, we will:
* Create a per PR prow-job comparing a limited subset of PRs touching tests/e2e/*

### Phase 3

In Phase 3, we will:
* Implement the tagging / feedback workflow
* Open up to all PRs hitting k/k

### Graduation Criteria

As this is a tooling component and is not user facing, it does not follow the
ordinary alpha/beta/GA process. In 1.17, the intent is to implement Phases 1 and 2,
without disruption to any feature development. The acceptance criteria here
are that the deliverables described in Phase 1 are complete, and that no
developers other than those writing or promoting conformance tests are
affected by the changes introduced in this KEP.


## Implementation History

- 2019-10-14: Created

## Drawbacks

* Automating policy is good overall, not seeing any immediate drawbacks.

## Alternatives

### Manual / Periodic Testing

This option is essentially what we are doing now.


*Pros*
* It's already in place.

*Cons*
* We find out about drops in coverage after the fact.
* Identifying which PR created the drop in coverage is difficult

