---
title: kubectl-diff
authors:
  - "@julianvmodesto"
owning-sig: sig-cli
participating-sigs:
  - sig-api-machinery
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2020-01-15
last-updated: 2020-01-15
status: provisional
see-also:
  - "/keps/sig-api-machinery/0015-dry-run.md"
  - "/keps/sig-api-machinery/0006-apply.md"
replaces:
  - n/a
superseded-by:
  - n/a
---

# kubectl-diff

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories [optional]](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

The `Summary` section is incredibly important for producing high quality user-focused documentation such as release notes or a development roadmap.
It should be possible to collect this information before implementation begins in order to avoid requiring implementors to split their attention between writing release notes and implementing the feature itself.
KEP editors, SIG Docs, and SIG PM should help to ensure that the tone and content of the `Summary` section is useful for a wide audience.

A good summary is probably at least a paragraph in length.

## Motivation

This section is for explicitly listing the motivation, goals and non-goals of this KEP.
Describe why the change is important and the benefits to users.
The motivation section can optionally provide links to [experience reports][] to demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports

### Goals

List the specific goals of the KEP.
How will we know that this has succeeded?

### Non-Goals

What is out of scope for this KEP?
Listing non-goals helps to focus discussion and make progress.

## Proposal

This is where we get down to the nitty gritty of what the proposal actually is.

### User Stories [optional]

Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of the system.
The goal here is to make this feel real for users without getting bogged down.

#### Story 1

#### Story 2

### Implementation Details/Notes/Constraints [optional]

What are the caveats to the implementation?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they releate.

### Risks and Mitigations

What are the risks of this proposal and how do we mitigate.
Think broadly.
For example, consider both security and how this will impact the larger kubernetes ecosystem.

How will security be reviewed and by whom?
How will UX be reviewed and by whom?

Consider including folks that also work outside the SIG or subproject.

## Design Details

### Test Plan

In addition to unit tests, there will be integration tests using the
command-line integration test suite:

- [x] [Test `kubectl diff` for multiple resources with the same name](https://testgrid.k8s.io/presubmits-kubernetes-blocking#pull-kubernetes-integration&include-filter-by-regex=test-cmd.run_kubectl_diff_same_names)

### Graduation Criteria

#### Alpha -> Beta Graduation

- [x] At least 2 release cycles pass to gather feedback and bug reports during
  real-world usage
- [x] End-user documentation is written
- [x] The client-side dry-run used to calculate the diff is replaced with the
  server-side dry-run feature to improve correctness and accuracy for this
  feature
- [x] The dependent API server-side dry-run feature is released to beta

#### Beta -> GA Graduation

- [x] At least 2 release cycles pass to gather feedback and bug reports during
  real-world usage
- [ ] Integration tests are in Testgrid and linked in KEP
- [ ] Documentation exists for user stories
- [ ] The dependent API server-side dry-run feature is released to GA

### Upgrade / Downgrade Strategy

This section is not relevant because this is a client-side component only.

### Version Skew Strategy

To check what the merged live object would look like, the `kubectl diff`
command relies on server-side dry-run support for the resource.

If an API server has disabled server-side dry-run or the API server was
downgraded to a version without server-side dry-run, then `kubectl diff` will
fail to get a merged version of the object and not display a diff.

## Implementation History

- *2020-01*: Added KEP
- *2019-01*: Promoted from alpha to beta in 1.13
- *2017-12*: Released as alpha in 1.9

