<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
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

# KEP-2572: Defining the Kubernetes Release Cadence

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
    - [Enhance determinism](#enhance-determinism)
    - [Reduce risk](#reduce-risk)
    - [Collecting data](#collecting-data)
    - [Creating a policy](#creating-a-policy)
  - [Non-Goals](#non-goals)
    - [Long-term support (LTS) releases](#long-term-support-lts-releases)
    - [Changing enhancements graduation](#changing-enhancements-graduation)
    - [Architecture changes](#architecture-changes)
    - [Modifying SIG Architecture policies](#modifying-sig-architecture-policies)
    - [Accelerated release cycles](#accelerated-release-cycles)
    - [Establishing maintenance/stability releases](#establishing-maintenancestability-releases)
    - [Determining an upper bound for Release Team shadows](#determining-an-upper-bound-for-release-team-shadows)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [End User](#end-user)
    - [Distributors and downstream projects](#distributors-and-downstream-projects)
    - [Contributors](#contributors)
    - [SIG Release members](#sig-release-members)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Concentrating risk](#concentrating-risk)
    - [Attention to tests](#attention-to-tests)
    - [Attention to dependencies](#attention-to-dependencies)
- [Design Details](#design-details)
  - [Schedule Policy](#schedule-policy)
  - [Feedback survey](#feedback-survey)
- [Implementation History](#implementation-history)
  - [Leads meeting feedback session](#leads-meeting-feedback-session)
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
- [ ] (R) Graduation criteria is in place
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

With this KEP, SIG Release proposes to change the current Kubernetes release
cadence from 4 down to 3 releases per year.

## Motivation

Discussions around changing the release cadence for Kubernetes, which currently
releases 4 times per year, are ongoing in the community.

The extended release schedule for 1.19 resulted in only three minor Kubernetes
releases for 2020. As a result, SIG Release received several questions across a
variety of platforms and communication channels about whether the project
intends to only have three minor releases/year.

### Goals

#### Enhance determinism

With the current release cadence we already achieve a deterministic schedule
for every year. The goal of this KEP is to increase this even further by
providing a lightweight policy around creating the release schedule. Going
down to 3 releases provides additional room for triage, development, and
explicit breaks, which should result in better overall planning and more
predictability.

#### Reduce risk

With higher predictability we can reduce the overall risk of changing the
release schedule. The planning overhead of SIG Release gets reduced, while
users of Kubernetes gain more time to adapt to the latest release.

The current Kubernetes release cadence is so fast that most organizations
cannot keep up with regularly making a minor version update every 3 months or
going out of security support in a year. While releasing more frequently
theoretically reduces the churn and risk of each release, this is only true if
end users are actually able to apply the upgrades.

#### Collecting data

After this KEP is in place, SIG Release will follow up with a survey to collect
feedback about the new release cadence.

#### Creating a policy

The outcome of this KEP is a policy for creating release schedules for
Kubernetes.
This allows the release team, as well as users, to follow a set of simple rules
when it comes to knowing when and how Kubernetes releases will be scheduled.

### Non-Goals

#### Long-term support (LTS) releases

The LTS Working Group was
[disbanded](https://github.com/kubernetes/community/pull/5240) on October 20,
2020.

The outcome of their conversations was the proposal which established a
[yearly support period][/keps/sig-release/1498-kubernetes-yearly-support-period/README.md]
for minor releases of the project.

While we may revisit the idea in the future, for now we trust the 2+ years of
thoughtful deliberation by the working group enough to conclude that the
project is not currently in a place to support long-term support releases.

#### Changing enhancements graduation

This KEP will not change the way that enhancements are being graduated. It's
the responsibility of SIGs to keep track of their enhancements and graduate
them in the provided constraints of SIG Architecture.

The new release schedule will add room for only a few more weeks of
development.
SIGs should focus on using those additional weeks to enhance documentation and
testing (stability)—not on adding more features. These decisions are not part
of any SIG Release planning and will therefore be considered out of scope.

#### Architecture changes

Changing any architecture of Kubernetes—for example, decoupling its core
components from the k/k repository—is outside the scope of this KEP.

#### Modifying SIG Architecture policies

Any policy change made by SIG Architecture is out of scope of this KEP. This
non-goal corresponds partially to the [Changing enhancements
graduation](#changing-enhancements-graduation) section.

#### Accelerated release cycles

The intent of this proposal is to create more opportunities to provide a
high-value experience for Kubernetes consumers.

The Kubernetes community faces a reasonable amount of tech debt across
infrastructure, testing, policy, and documentation. This KEP proposes that we
spend more time paying down that debt.

SIG Release currently produces releases at the following cadence:

- patch releases (`x.y.Z`): [monthly][https://git.k8s.io/sig-release/releases/patch-releases.md]
- minor releases (`x.Y.z`): [every four months][https://git.k8s.io/sig-release/release-engineering/versioning.md]
- pre-releases (`x.y.0-(alpha|beta|rc).N`): every 1-3 weeks during active
  development cycles ([example](https://git.k8s.io/sig-release/releases/release-1.21/README.md#timeline))

At the time of writing, SIG Release considers these to be reasonable cadences
for patch and pre-releases.

If you'd like to provide suggestions on longer-term improvements that could
potentially accelerate production of releases, please join the discussion
[here](https://github.com/kubernetes/sig-release/discussions/1495).

#### Establishing maintenance/stability releases

Establishing a shorter maintenance/stability release at the end of the year has
been casually discussed at several points in the project, with the most recent
(at the time of writing) occurrence being
[here](https://github.com/kubernetes/sig-release/issues/809).

Nothing compelling has emerged from previous conversations to give cause to
establish maintenance/stability releases.

Fixing bugs, stabilizing components, adding/deflaking tests, improving
documentation, and graduating features are activities that can and should
happen in a reasonably consistent manner throughout the year.

#### Determining an upper bound for Release Team shadows

It was noted that fewer releases for the year would lead to fewer opportunities
to participate on the Release Team.

This will be discussed and addressed in
https://github.com/kubernetes/sig-release/issues/1494.

## Proposal

### User Stories

Kubernetes releases are made by real people. The technical aspects—for example,
the release automation—reflects only a tiny part of the complete cycle. This
means we will mainly focus on the human aspects and their corresponding roles
when deciding to move to a 3-releases-per-year cadence.

#### End User

Most companies are facing issues upgrading Kubernetes 4 times a year. Providing
only 3 releases per year will relax this situation.

#### Distributors and downstream projects

Downstream projects assemble their solution from different projects. Having
fewer upgrades helps them to reduce complexity. For example, cloud providers
will gain more room for upgrading their infrastructure.

#### Contributors

With a lower release cadence, contributors will gain more time for project
enhancements, feature development, planning, and testing. It will provide more
room for maintaining their mental health and prepare for events like KubeCon.

Through this proposal SIG Release's aim is to give contributors more
flexibility to decide how to invest their time. It is explicitly *not* to push
contributors in doing more.

#### SIG Release members

By applying a cadence of 3 releases per year, SIG Release members will gain a
reduced management overhead. There are also only 3 patch releases to maintain,
which right now can overlap up to 4. SIG Release will gain more time to ensure
a seamless transition from the previous release team to the next one. It is
also possible to include more shadows if the role leads conclude that this is
appropriate.

### Risks and Mitigations

#### Concentrating risk

In theory a reduced release cadence will cause more changes for every release.
This means that there will be an increased risk, which would usually be split
up into 4 dedicated milestones rather than 3.

SIG Release cannot mitigate this risk directly, but is able to track and
influence it during each release cycle. It's the responsibility of SIG Release,
together with SIG Testing and SIG Architecture, to identify new gaps and issues
in the release cadence and mitigate them on a case-by-case basis.

#### Attention to tests

This KEP does not propose any change to the release cycle itself and assumes
that the same periods for Code and Test Freeze. Assuming that, there is an
increased risk for flakes and test failures. It will be the responsibility of
SIG Release to mitigate this, together with the CI signal role. If we speak
about an overall release cycle enhancement of 3-4 weeks, then we believe that
SIG Release is able to mitigate this risk over multiple releases.

#### Attention to dependencies

Having fewer releases will introduce the risk of missing dependencies—for
example, Golang upgrades. This has to be mitigated on a case-by-case basis, in
the same way as it is being done right now.

## Design Details

### Schedule Policy

The exact details of the release schedule are created by the team leads before
the actual cycle begins. SIG Release only provides a lightweight policy, which
is defined as:

1. The first Kubernetes release of a year should start at the second or third
   week of January to provide people more room after coming back from the
   Christmas holidays.

2. The last Kubernetes release of a year should be finished by the middle of
   December.

3. A Kubernetes release cycle has a length of of ~15 weeks.

4. Events like KubeCon will be considered as blocked from development or
   decision-making. SIG Release will also consider the week before and after
   the event in the same way.

5. An explicit break of at least two weeks between each release cycle will be
   enforced.

   This does not mean that zero development can happen during that time.
   Rather, SIG Release will use this time to do the release retrospective and
   plan for the next cycle.

### Feedback survey

Each minor Kubernetes release will be an experience survey, which will include
questions around the release cadence.

Survey contents are to be determined, but we welcome content suggestions to
continually improve the process.

Post-release surveys will close after the `.2` patch release to allow the team
sufficient time to process and incorporate feedback.

Using Kubernetes v1.19 date to provide an example of the survey timeline:

- 2020-08-26: v1.19.0 released (survey would go out)
- 2020-09-09: v1.19.1 released
- 2020-09-16: v1.19.2 released (survey would close)

With this example, the survey would have been open for three weeks.
With an extended release cycle, post-release surveys would be open for around
three to six weeks (depending on the patch release schedule).

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

### Leads meeting feedback session

Already captured above, but you can find meeting notes [here](https://docs.google.com/document/d/1Jio9rEtYxlBbntF8mRGmj6Q1JAdzZ9fTDo3ru1HK_LI/edit#bookmark=id.val5alfdahlr).

## Drawbacks

The main drawbacks of this KEP have been covered in the [Risks and
Mitigations](#risks-and-mitigations) section.

## Alternatives

The alternative approaches have been discussed in the [Non-goals](#non-goals)
section.
