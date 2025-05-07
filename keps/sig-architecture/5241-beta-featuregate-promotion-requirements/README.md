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
# KEP-5241: Beta Feature Gate Promotion Requirements

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
    - [What if I need to add capability to my feature?](#what-if-i-need-to-add-capability-to-my-feature)
    - [Who will make sure that new KEPs follow the promotion rules?](#who-will-make-sure-that-new-keps-follow-the-promotion-rules)
  - [Graduation Criteria](#graduation-criteria)
- [Drawbacks](#drawbacks)
  - [This may slow the rate that new features are promoted.](#this-may-slow-the-rate-that-new-features-are-promoted)
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

Features gates must include all functional, security, monitoring, and testing requirements along with
resolving all issues and gaps identified prior to being enabled by default.
The only valid GA criteria are “all issues and gaps identified as feedback during beta are resolved”.

## Motivation

Features gates that are enabled by default are enabled in every production Kubernetes cluster in the world.
We must avoid making every production cluster into unstable or incomplete feature testing clusters.
Even feature gates that make flags accessible, but require a secondary configuration to use must be
stable, because it is unrealistic to expect everyone to understand the graduation stages of various flags
for each release: the only stages that really matter are "takes enabling an explicit alpha feature gate"
and "my production cluster accepts this as valid by default".

### Goals

* Features gates must include all functional, security, monitoring, and testing requirements along with
  resolving all issues and gaps identified prior to being enabled by default.
* The only valid GA criteria are “all issues and gaps identified as feedback during beta are resolved”.

### Non-Goals

* Changing beta APIs off by default rules.
* Change the imperfect mechanisms we have for API evolution.

## Proposal

Kubernetes feature gates have three levels: GA (locked on), GA (disable-able), Beta, and Alpha.
1. GA (locked-on) means that a feature gate is unconditionally enabled in all production kubernetes clusters and
   that feature cannot be disabled.
2. GA (disable-able) is only for features gates that include a new API serialization that cannot be enabled by default 
   until the API reaches stable.  This means that the first time the API is enabled in production, the feature will
   be GA, but also can be disabled.  This is a less common state and does not apply to most features.
3. Beta means that a feature gate is usually enabled in all production Kubernetes clusters by default
   and that feature can be disabled.
   Exceptions exist for entirely new APIs and some node features, but this broadly the case.
4. Alpha means that a feature gate is disabled in all production Kubernetes clusters by default and
   can be optionally enabled by setting a `--feature-gate` command line argument.

Making the jump to GA (cannot be disabled), without actual field experience is irresponsible.
The first time we take a feature gate enabled by default in production Kubernetes clusters, we must
have a way to disable the feature in case of unexpected stability, performance, or security issues.

Enabling incomplete features in production Kubernetes clusters by default is irresponsible.
Features that are known to be incomplete naturally bring with them additional stability, performance, and security issues.
Once a feature has been enabled in a production Kubernetes cluster by default, adding to it carries
greater risk to upgrading clusters and the ecosystem.
The feature can easily have become relied upon by workloads and other platform extensions.
If an accident happens in adding those capabilities with stability, performance, and security the
cost to disable those features in a cluster becomes significantly greater and breaks existing
clusters, workloads and use-cases.
This posture makes upgrades higher risk than necessary.

To balance these concerns, we are changing how we evaluate Beta and GA stability criteria.
The only valid GA criteria are “all issues and gaps identified as feedback during beta are resolved”.
Promotion from Beta to GA must be zero-diff for the release.
This means that Beta criteria must include all functional, security, monitoring, and testing requirements along
with resolving all issues and gaps identified prior to beta.

Phasing in larger features over time can be done by bringing separate feature gates through alpha, beta, and GA.
Each feature gate needs to meet the beta and GA criteria for completeness, functional, security, monitoring, and testing.
After meeting the criteria for enabled by default, and at the SIG's discretion, the new feature gate could be 
set to enabled by default in the release it is introduced.
Importantly, the features need to behave in a way that allows old and new clients to interoperate and new additions
to larger features able to be independently disablable with their own path for GA.

### Risks and Mitigations

#### What if I need to add capability to my feature?
To handle this situation, we described above how to add second feature gate for the new behavior.
This provides a mechanism for adding needed capability, but ensures that
cluster-admins never end up stuck after upgrade because they rely on v1.Y-1 behavior that new capability
in v1.Y broke under the same feature gate.

#### Who will make sure that new KEPs follow the promotion rules?
We'll adjust the KEP template to indicate the allowed criteria, so authors should notice.
SIG approvers should enforce those standards.
PRR approvers can be a final backstop.

### Graduation Criteria

This document is our new position once merged until it is superceded by another position statement.

## Drawbacks

### This may slow the rate that new features are promoted.
For this to be true, that would mean that we previously enabled feature gates in production that were knowingly
incomplete for functional, security, monitoring, testing, or known bugs.
We hope this was not the common case, but if it was the common enough to have an impact, we're pleased that
the result is preventing incomplete feature gates from being enabled in production clusters.

## Alternatives

None proposed so far.
