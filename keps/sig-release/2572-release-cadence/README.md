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
    - [Feature graduation](#feature-graduation)
- [Design Details](#design-details)
  - [Schedule Policy](#schedule-policy)
  - [Feedback survey](#feedback-survey)
- [Implementation History](#implementation-history)
  - [Leads meeting feedback session](#leads-meeting-feedback-session)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

With this KEP, SIG Release proposes to change the current Kubernetes release
cadence from 4 down to 3 releases per year. This would start with the 
Kubernetes 1.22 release cycle.

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
down to 3 releases provides additional room for triage, development, conference
and release cycle preparations, which should result in better overall planning
and more predictability.

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

After this KEP is in place and the first three minor (`1.x.0`) versions have
been released, SIG Release will follow up with a survey to collect feedback
about the new release cadence.

#### Creating a policy

The outcome of this KEP is a policy for creating release schedules for
Kubernetes.
This allows the release team, as well as users, to follow a set of simple rules
when it comes to knowing when and how Kubernetes releases will be scheduled.

### Non-Goals

#### Long-term support (LTS) releases

The LTS Working Group was
[disbanded](https://github.com/kubernetes/community/pull/5240) on October 20, 2020.

The outcome of their conversations was the proposal which established a
[yearly support period](/keps/sig-release/1498-kubernetes-yearly-support-period/readme.md)
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
testing (stability) - not on adding more features. These decisions are not part
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

- patch releases (`x.y.Z`): [monthly](https://git.k8s.io/sig-release/releases/patch-releases.md)
- minor releases (`x.Y.z`): [every four months](https://git.k8s.io/sig-release/release-engineering/versioning.md)
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

Most end user organizations find it difficult to match Kubernetes release
cadence - only 3 releases per year will relax this situation.

#### Distributors and downstream projects

Downstream projects assemble their solution from different projects. Having
fewer upgrades helps them to reduce complexity. For example, cloud providers
will gain more room for upgrading their infrastructure.

#### Contributors

With a lower release cadence, contributors will gain more time for project
enhancements, feature development, planning, and testing. It will provide more
room for maintaining their mental health, prepare for events like KubeCon or
work on the downstream integration.

Through this proposal SIG Release's aim is to give contributors more
flexibility to decide how to invest their time. It is explicitly _not_ to push
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

Having fewer releases will introduce the risk of missing dependencies — for
example, Golang upgrades. This has to be mitigated on a case-by-case basis, in
the same way as it is being done right now.

#### Feature graduation

Research discovered that only 5% of Kubernetes features advanced from Alpha to
GA in the minimum 3 releases. However, the same research showed that reminders
from the Release Team played a critical role in advancement of more than 50% of
features. With an increased release cycle, this reminder activity can be
expected to slow down. As such, advancement will need to be mitigated by making
sure that SIGs keep track of their feature enhancement in more detail.

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
   decision-making from the SIG release perspective. SIG Release will also
   consider the week before and after the event in the same way.

5. An explicit SIG release break of at least two weeks between each cycle will
   be enforced.

   This does not mean that zero development can happen during that time.
   Rather, SIG Release will use this time to do the release retrospective and
   plan for the next cycle.

### Feedback survey

SIG Release will draft an experience survey after the first three releases from
which the new cadence has been applied. This survey which will include questions
around the release cadence and how it impacted end users.

Survey contents are to be determined, but we welcome content suggestions to
continually improve the process.

## Implementation History

### GitHub Discussion

Prior to opening this KEP, a [Github Discussion](https://github.com/kubernetes/sig-release/discussions/1290) was opened to solicit community feedback, which was used as the basis for this KEP. 

### Leads meeting feedback session

Already captured above, but you can find meeting notes [here](https://docs.google.com/document/d/1Jio9rEtYxlBbntF8mRGmj6Q1JAdzZ9fTDo3ru1HK_LI/edit#bookmark=id.val5alfdahlr).

## Drawbacks

The main drawbacks of this KEP have been covered in the [Risks and
Mitigations](#risks-and-mitigations) section.

## Alternatives

The alternative approaches have been discussed in the [Non-goals](#non-goals)
section.
