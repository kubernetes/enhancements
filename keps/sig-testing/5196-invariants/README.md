# KEP-5196: Invariant Signal Collection for Kubernetes Testing

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

This proposal defines a system to gather and analyze invariant signals about the Kubernetes cluster during test execution. We want to see how key parts of the system are behaving in a way that's similar to how real users would see it. This will help us find problems that normal tests might miss, and give us a better picture of how stable and reliable Kubernetes is.

## Motivation

Even with extensive testing, it's impossible to catch every potential problem in Kubernetes. Some issues only show up when the system is used in real-world scenarios. We want to use our testing setup to act like a group of users, watching for critical signs that something might be wrong. These signs, or "invariants," can tell us if the cluster is healthy and working as expected. By collecting and analyzing this information, we can find and fix problems before they affect real users.

### Goals

- To build a system that collects and reports on how well defined invariants are behaving during our tests.
- To get a better understanding of how these invariants perform in a situation that looks like real user usage, so we can find potential problems early.
- To create a way to look at and understand this invariant information, so we can make better decisions about the system.

### Non-Goals

- To make invariant checks part of the regular pass/fail tests.
- To change the way our tests work now.
- To collect invariant data from places that normal users can't easily see.
- To build a complex system; we'll focus on collecting and looking at the information first.

## Proposal

We will add to our current testing system so it can collect and report invariant data during tests. This data will be stored in a central place where we can look at it later. We will use existing tools whenever possible.

1. Defining Invariants:

We will create a file that lists all the invariants we want to watch, with clear descriptions and how to get the data.
These definitions will need approval from SIG Architecture and Testing Leads to make sure they are good and useful.

- (TBD) Similar to existing conformance.yaml , it has to be part of test/e2e/invariants/data.yaml or similart

2. Collecting Invariant Data:

Our tests will collect the invariant data at the end of each run, using the methods we defined.
We will only collect invariant data from places that are easily accessible.

- (TBD) Ginkgo Reporting AfterSuite + Metrics Gatherer

3. Storing and Processing Invariant Data:

We will use existing tools to store the invariant data in a database that is easy to search.

- https://github.com/kubernetes/test-infra/tree/master/kettle#readme

4. (Future) Looking at the Invariant Data:

We will create dashboards to show the invariant data in a way that's easy to understand.

- https://github.com/kubernetes/test-infra/tree/master/metrics 
