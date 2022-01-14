# KEP-3138: Increasing Reliability Bar

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [[Milestone 1] Existing tests failures/flakiness](#milestone-1-existing-tests-failuresflakiness)
  - [[Milestone 2] Ensuring reasonable test coverage](#milestone-2-ensuring-reasonable-test-coverage)
  - [[Milestone 3] Addressing long-standing reliability issues](#milestone-3-addressing-long-standing-reliability-issues)
  - [[Milestone 4] New reliability investments](#milestone-4-new-reliability-investments)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
- [Implementation History](#implementation-history)
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
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

With Kubernetes project being more and more mature and being widely used for
critical production usecases, it’s increasingly important to ensure the high
quality bar for all its releases and contributions.

In the spirit of that, we would like to make the reliability of Kubernetes
a priority for everyone in the community. To make the project sustainable,
we need to prioritize that over new features.

To achieve that, we propose splitting the effort into four milestones:
1. Reducing test flakiness in a sustainable way
1. Increasing test coverage and tests quality
1. Addressing long-standing reliability issues
1. Investing into new reliability-oriented efforts

In order to achieve the goals across the whole project and all SIGs, we need
to ensure that the principles will not be ignored. So while we hope that building
the appropriate culture and appropriate encouragement will be enough, we also
define the enforcement mechanisms that could be used if everything else fails.


## Motivation

As Kubernetes is becoming more and more mature, and widely used in production,
we need to increase the quality bar to reduce the amount of outages and issues
our users are facing.

While companies/cloud-providers do not generally publish the data about outages,
we already have a lot of data that proves the above. The last [Production
Readiness Survey][] (performed by Production Readiness subproject of SIG
Architecture) proved that there are tens of examples where users had to rollback
their cluster due to issues with examples of many different reasons, including:
broken feature, failing components, scalability issues or even failing clusters.
Being forced to rollback clearly means a significant reliability issue.
There are also many examples described in different blog posts or Kubecon talks,
e.g. a recent [How to Break your Cluster with networking][]).
However, many issues that were discussed even years ago during other Kubecon
talks (e.g. [1][] or [2][]) also weren’t addressed. And these aren’t just
hypothetical - many of us have seen them in real production.

However, it’s not just about those huge outages. It’s also about smaller incidents
(some of them even self-recovering, but still causing issues in the meantime).
By having a better testing discipline we could catch a lot of them. But what’s
even worse, we aren’t catching some even though they are exposed by our tests,
because they are simply getting lost/oversight amongst a huge number of test
flakes. Networking is a great example - back then around 1.20-1.21 releases
their tests were extremely flaky. Thanks to a huge and organized effort by the
SIG network they reduced their flakiness by orders of magnitude. While many of the
problems were test-related, addressing them allowed them to uncover real system
issues, like [this example][], which must have affected many real production
clusters before.

To make Kubernetes (or any other project) sustainable, we should ensure that
existing architecture, individual features, tests etc. really works as expected.
If there is work needed to improve reliability (or stability) of those, it
should come before new features.
This is often not the case currently. We significantly rely on the heroic
efforts of very few individuals that are keeping us on the surface with new bugs,
issues, test flakes etc. This approach isn’t sustainable and doesn’t scale.
We need to make it a business of everyone in the whole community.

[Production Readines Survey]: https://datastudio.google.com/reporting/2e9c7439-202b-48a9-8c57-4459e0d69c8d/page/GC5HB
[How to Break your Cluster with networking]: https://www.youtube.com/watch?v=7qUDyQQ52Uk)
[1]: https://www.youtube.com/watch?v=QKI-JRs2RIE
[2]: https://www.youtube.com/watch?v=6sDTB4eV4F8
[this example]: https://github.com/kubernetes/kubernetes/pull/98305


### Goals

- Define the high-level areas of investments
- Define the sustainable process to ensure reliability investments
- Define the details of the first milestone as a pilot (reducing test flakiness)


### Non-Goals

- Providing exact list of long-standing reliability issues (milestone 3) or new
  reliability investments (milestone 4) - those will be handled by
  Reliability WG separately


## Proposal

This document outlines steps to improve the quality and reliability of Kubernetes.
Given the maturity level the project is reaching and the non-negligible amount of
technical debt accumulated over the years, the most important aspect of the
proposal is to build the culture which prioritizes certain types of contributions
over purely feature work.

We have to acknowledge that fixing, and especially debugging, reliability or
stability issues is the kind of work that very few people like doing - for the
majority it’s something that they would never do if they wouldn’t have to.
This is especially true when it comes to debugging flaky tests.
Moreover, while in companies there are multiple ways of enforcing that certain
types of work is actually happening (e.g. oncall rotations, dedicated sprints, etc.),
it’s much harder to achieve in the projects driven by the open-source community.

As a result, we’re proposing introducing a process that will allow us to ensure
that this kind of work that generally attracts fewer developers  will be really
happening. This means, that for every piece of such work, we will introduce a two
phase approach:
1. announce the reliability-related issue to the SIG asking for prioritizing it
   with a realistic (and non-conservative) deadline.
   *We really hope that this will be effective in the majority of cases and
   we will never be forced to use the point (2) special powers.*
1. if the issue (reported above) is not addressed before the deadline, block
   new features from the owning SIG until the issue will get addressed


### Risks and Mitigations

**We acknowledge that it may hurt the overall productivity and velocity of the
community in the short-term. However, this is necessary to increase both
reliability and our velocity medium and longer term.

To mitigate the risk, we will provide the necessary tooling to ensure that any
single SIG will not get surprised by the consequences of conscious
deprioritization of reliability-related work.**


Another risk is that due to pulling engineers away from the most interesting
feature work towards non-fancy work, we may see attritions and decrease in
contributor base. This is why it is critical to actually measure the benefits
from this effort. Most of the metrics will turn out to be lagging indicators,
so they will rather be used to measure the impact and reflect on the process
periodically, not necessarily guide immediate future actions.

The exact metrics are to-be-decided, but the current candidates include:
- number of bugs (ideally weighed by severity) reported per SIG for already
  released versions:
  - for cross-cutting issues, they will be counted towards “owning SIG”
- number of cherrypicks per SIG for a given minor release (targeting low
  numbers meaning that SIG doesn’t have any issues that need to be addressed).
  - for cross-cutting fixes, they will be counted towards “owning SIG”
- test flakiness (as observed in https://sippy.k8s.io/?release=kube-master)
- test coverage (the details of how to measure it are to-be-decided)
- [ideal but unlikely due to inability to get the data] number of outages
  reported by Kubernetes providers


## Design Details

As mentioned above, we propose to divide the effort into 4 milestones.
Below we describe these milestones, focusing on the first milestone that
will be executed first as a pilot.

### [Milestone 1] Existing tests failures/flakiness

When a test fails (especially the end-to-end test), the reason for the failure
is often unclear. All of the following reasons are theoretically possible:
- test-related issue (e.g. test written in non-deterministic way, incorrect
  CI job configuration, etc.)
- a bug in the exercised component
- a bug in another component (potentially owned by a different SIG)
- some underlying infrastructure-related issues (e.g. cloud-provider outage)

For sustainability, we should figure out a path towards understanding them.
Obviously we’re extremely far from being able to do that now, but without any
work the amount of necessary work to get there can only increase - it will
not decrease on its own.

To make the effort trackable and sustainable we propose the following:
1. We will limit the effort only to the test jobs that are part of
   “master release blocking” and “master release informing” tabs.
   - If really needed we may consider extending the set of jobs in scope,
     but this is not a plan of record as of now.

1. We will use [Sippy][] as our source of truth for determining the health
   of the individual jobs and tests.
   We’ve already set up [Kubernetes Sippy][] that is monitoring the jobs
   described above.
   We will focus on the “Top failing tests” part of it.

1. People may use https://storage.googleapis.com/k8s-triage/index.html for
   finding instances of the failing tests that they can use for debugging purposes.

1. We will be periodically looking at the “Top failing parts” to:
   - Open issues (labeled with “kind/reliability-issue” and appropriate SIG)
     for the worst one such that:
     - There should be at most 3 open issues for a given SIG, with the caveat
       of significant regressions - if the test pass rate goes down by more
       than 10% week over week, that justifies opening additional ones
     - We considered automatic issue filing but decided not to start with it due to:
       - CI signal experience with flakes (e.g. system-level issues, handling
         duplicates etc.)
       - people tending to ignore bot-originating issues
     - The proper labeling of issues would mean that people would be able to easily
       find issues currently opened for a given SIG, via a simple github query like
       this `https://github.com/kubernetes/kubernetes/issues?q=is%3Aopen+is%3Aissue+label%3Asig%2Ffoo+label%3Akind%2Freliability-issue`
   - Notify the SIG on Slack asking for explicit acknowledgement on the issue
   - Start a timer on the issue: 2 months (half of the release) is the proposed default

1. Even though each test is generally assigned to a SIG, tests may be failing
   due to bugs or changes done by other SIGs. SIG labels are designed to be
   easy to apply and change to ensure a collaborative nature of issue triage.
   If a SIG will show that the bug is elsewhere, they should notify that
   other SIG, reassign the bug and the new owning SIG is allowed to restart
   the clock.

1. The issues can’t simply be closed with “test is now passing”. Even if in
   the meantime the failure rate dropped, it should still be investigated
   and debugged to ensure that this won’t spike again in the future for the
   same reason.
   If the test can’t be debugged due to lack of data, the expected step
   forward is to increase our observability by adding logs, metrics, events,
   etc. to allow debugging the future instances of the flake.

1. Once debugged and fixed, the issue should be labeled with the root cause
   category from the predefined set, e.g. (cause/test-issue, cause/system-issue,
   cause/infra) for the purpose of tracking it at the higher level.

1. We acknowledge that some issues might be hard to debug, so the deadline
   for a bug may be increased by WG Reliability if the bug was actively worked
   on since it was reported.
   This includes situations where larger test(s) redesign is the desired way
   of addressing the problem, in which case SIG may apply for extending the
   deadline to have time to execute on that and avoid investing into current
   tests that eventually will be replaced.

1. When the issue reaches the actual (potentially extended) deadline, the SIG
   gets added to the list of SIGs that are not allowed to graduate any feature
   to Alpha or Beta (Beta to GA graduations are still allowed as they serve
   increased stability of the system)
   - Depending on the lifecycle of the release, it may mean not being able to
     target a particular KEP for a release or not being able to change the
     value of a feature gate [for that we may want to adjust OWNERS filed in
     features directory].
   - The SIG will be removed from the list when all issues opened (for them)
     at the moment of adding SIG to the denylist will get resolved [which is
     consciously a more strict requirement then for getting added to the list].

Why do we believe that eliminating flakes is possible? Purely based on the
empirical experience in the project - around 1.20 - 1.21 release, networking
tests were extremely flaky. However, thanks to organized (and huge!) effort
from within the SIG network, we are pretty much not observing any flakes,
even though many of those depend on cloud-provider infrastructure. This
clearly proves we can achieve great results if we just invest enough capacity
in it.

[Sippy]: https://github.com/openshift/sippy
[Kubernetes Sippy]: https://sippy.k8s.io/?release=kube-master


### [Milestone 2] Ensuring reasonable test coverage

Even if our tests are perfectly green, to ensure the quality of our features
we also need to provide a reasonable test coverage.

To achieve this, we propose that each KEP should explicitly list not just a
set of integration and e2e tests required for promotion of a feature to Beta
or Stable stages, but even more importantly, the missing links to them will
be blocking promotion to the next stage. In other words - proving their
existence (e.g. by linking from kep.yaml) will be a requirement for promotion
to the next stage.

We will work with the enhancements team to figure out the details and exact
path to get there. This may also require changing our practices and processes
and ensure that test coverage is actually required for merging PRs (to avoid
too many post-merge/pre-deadline issues). However, exact details for this
milestone will come later.

At the same time, we will be working with SIG testing on measuring unit test
coverage. Once that data exists, we will be filing issues for individual
SIG for test coverage improvement. The exact details are still to be figured
out, but the procedure is expected to be very similar as for flaky tests.

We will also work with SIG testing on ensuring that our overarching test
philosophy and test implementation guidance is up-to-date and reflects the
standards we would like to have across the project. Based on that we will
also figure out a strategy for how to backfill those standards to already
existing features and tests. However, the details of that are out of scope
for this document.


### [Milestone 3] Addressing long-standing reliability issues

This category covers system issues that cannot be easily fixed with 1 or 2 PRs,
but rather require significant changes in the system (we can probably
characterize them as requiring a KEP). A good recent example of something
in this category may be CronJob (which was staying in Beta for a long time
due to them).

Given the nature of the problem, as well as the fact that many of them
exist in Kubernetes for years, we can’t suddenly block contributions from
a SIG until the problem is addressed. But we really need a way to ensure
that those will get addressed.

As a result we propose a similar pattern to the one for flaky test:
1. The initial list of such issues will be prepared by WG Reliability and
   presented for approval to SIG Architecture (not started yet). Over time,
   new issues in this category may be added to the list (again with SIG
   Architecture approval), e.g. as a result of investigations/debugging of
   test flakes.
1. For each such issue, we will open a tracking issue in the k/k repo and
   tag it appropriately with kind/reliability.
1. From then, a SIG has two releases to present an approved and implementable
   KEP addressing the problem with a clear execution plan on it (by default
   Alpha in the upcoming release (N), Beta in the next release (N+1) and
   Stable in N+3).
1. If a particular SIG will have more than N (say N=2) such issues opened,
   the SIG may limit themselves to work on any chosen 2 issues.
1. If either the KEP itself or its execution will fall over the expected
   timelines, similarly as above this will block any Alpha or Beta
   graduations for features owned by this SIG.


### [Milestone 4] New reliability investments

This category covers investments that we should make towards increasing the
quality and/or reliability of Kubernetes. This area includes (but is not
limited to):
- creating new types of tests (e.g. chaos tests)
  - New tests that demonstrate a real issue will be merged and run in their
   dedicated test suite. Their results will prove the issue from [Milestone 3]
   category and will be prioritized together with other issues in that category.
- features to harden Kubernetes (Priority & Fairness is a good current example)
- architectural improvements leading to better reliability

The list of such investments will be created by the Reliability WG as one of
the artifacts they should produce.

Given this category is generally quite similar to the previous one, we’re
going to use the exact same process (including SIG Architecture approval)
for addressing issues from that list.


### Test Plan

N/A


### Graduation Criteria

We're going to proceed with this process.

However, the agreement on the process doesn't mean it's set in stone forever.
While in effect, the concerns about it should be brought up and we will be
trying to address them by adjusting the process.


## Production Readiness Review Questionnaire

This proposal is just a process proposal - PRR doesn't apply.


## Implementation History

11/2020 - First draft of the proposal
11/2021 - Revised v2 version of the proposal
01/2022 - KEP opened based on the proposal
