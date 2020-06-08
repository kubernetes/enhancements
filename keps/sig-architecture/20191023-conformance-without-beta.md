---
title: Ensure Conformance Tests Do Not Require Beta APIs or Features
authors:
  - "@liggitt"
owning-sig: sig-architecture
participating-sigs:
  - sig-testing
  - sig-api-machinery
reviewers:
  - "@deads2k"
  - "@bentheelder"
  - "@timothysc"
  - "@smarterclayton"
  - "@johnbelamaric"
approvers:
  - "@timothysc"
  - "@smarterclayton"
  - "@johnbelamaric"
creation-date: 2019-10-23
last-updated: 2019-10-23
status: implementable
see-also:
  - "/keps/sig-architecture/20190412-conformance-behaviors.md"
---

# Ensure Conformance Tests Do Not Require Beta REST APIs or Features

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
- [Design Details](#design-details)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Infrastructure Needed](#infrastructure-needed)
- [References](#references)
<!-- /toc -->

## Release Signoff Checklist

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [x] "Implementation History" section is up-to-date for milestone
- [x] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes
- [x] KEP approvers have set the KEP status to `implementable`
- [ ] ~~User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]~~

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

The KEP describes a process for ensuring that the Kubernetes components and the Kubernetes conformance tests have no dependencies on beta REST APIs or features.

## Motivation

As the Kubernetes project matures, and is used as the foundation for other projects,
production distributions are expected to increase in stability, reliability, and consistency.
The Kubernetes conformance project is an important aspect of ensuring consistency across distributions.
A production distribution should not be required to enable non-GA features or REST APIs in order to pass conformance tests.

Beta dependencies can be grouped into several categories:
* Kubernetes components requiring beta REST APIs with no stable alternatives (kubelet's use of the beta CertificateSigningRequest endpoints for cert rotation is a borderline example; cert rotation is not a required feature, but all our setup and test tools make use of it)
* Kubernetes components requiring beta REST APIs with stable alternatives (for example, kube-scheduler accidentally switched to using the beta Events REST API in 1.16)
* Kubernetes components requiring behavior enabled by beta feature gates
* Kubernetes conformance tests requiring calls to beta REST APIs
* Kubernetes conformance tests exercising behavior enabled by beta feature gates

For each category, we should identify and resolve existing dependencies, and prevent future occurrences.

### Goals

* Identify existing beta REST APIs and features required by Kubernetes components so they can be graduated or the dependencies removed
* Identify existing beta REST APIs and features required by conformance tests so the tests can be rewritten to remove those dependencies
* Prevent new beta REST APIs and features from being required by Kubernetes components or conformance tests
* Demonstrate passing conformance tests on a cluster with all beta REST APIs and features disabled

### Non-Goals

* Resolve questions about conformance profiles (see https://github.com/kubernetes/community/issues/2651)
* Resolve questions about GA-but-optional features (see https://github.com/kubernetes/community/issues/3997)
* Resolve code-level dependencies in Kubernetes source code on pre-v1.0.0 components
* Forbid beta REST APIs or features from being enabled in conformant clusters by distributors

## Proposal

### User Stories

#### Story 1

As a Kubernetes component developer, I cannot accidentally make changes
that would require a production cluster to enable a beta API or feature
in order to pass conformance tests.

Attempts to make such a change would be caught/blocked by a CI job.

#### Story 2

As a Kubernetes conformance test author, I cannot accidentally write
(or promote to conformance) tests that require use of beta REST APIs.

Attempts to make such a change would be caught/blocked by a CI job.

#### Story 3

As a Kubernetes distribution maintainer, I can pass conformance tests
without enabling beta REST APIs or features.

A canonical CI job running against Kubernetes master and release branches
would demonstrate that it is possible to pass conformance tests with no
beta REST APIs or features enabled.

## Design Details

1. Make it possible to easily disable all beta REST APIs and features
  * Add support for disabling built-in REST API versions of the form `v[0-9]+beta[0-9]+` to the Kubernetes API server with `--runtime-config api/beta=false`
    * Parallels existing use of `--runtime-config api/all=false`
    * For completeness, we can also add support for disabling built-in REST API versions of the form `v[0-9]+alpha[0-9]+` to the Kubernetes API server with `--runtime-config api/alpha=false`
  * Add support for disabling beta feature gates to all components with `--feature-gates AllBeta=false`
    * Parallels existing use of `--feature-gates AllAlpha=false`
2. Identify existing beta REST APIs/features required by Kubernetes components or conformance tests
  * Iteratively run a conformance test with all beta REST APIs and features disabled
  * Identify failures due to uses of beta features/REST APIs
  * Open issues for owners of the relevant tests or components to remove the beta dependency
  * Construct an exemption list of beta features/REST APIs currently required to pass conformance
3. Prevent introduction of new required beta dependencies in Kubernetes components or conformance tests
  * Set up a merge-blocking CI job running conformance tests with all beta REST APIs and features disabled except the exemptions constructed in step 2.
4. Resolve existing dependencies on beta REST APIs/features required by Kubernetes components or conformance tests
  * Track issues opened in step 2
  * As each dependency is resolved, remove it from the exemption list

### Graduation Criteria

Phase 1:
* All beta REST APIs and features required to pass conformance tests are identified
* Blocking presubmit and periodic CI jobs ensure no additional beta dependencies are introduced

Phase 2:
* All identified dependencies on beta REST APIs and features to pass conformance are resolved
* Blocking presubmit and periodic CI jobs ensure no beta dependencies are introduced

Phase 3:
* All GA APIs required to pass conformance tests are identified
* Blocking presubmit and periodic CI jobs ensure no dependencies on optional GA APIs are introduced as required into conformance tests

## Implementation History

- 2019-10-23: KEP created
- 2019-11-01: KEP marked implementable

## Infrastructure Needed

A pre-submit and periodic CI job per release branch, configured with beta REST APIs and features disabled

## References

Existing guidelines that conformance tests should only make use of GA APIs:
* https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md#conformance-test-requirements

Past issues attempting to diagnose use of beta APIs by conformance tests:
* https://github.com/kubernetes/kubernetes/issues/78605
* https://github.com/kubernetes/kubernetes/issues/78613
