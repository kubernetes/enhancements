---
title: Require Transition from Beta
authors:
  - "@deads2k"
owning-sig: sig-architecture
participating-sigs:
reviewers:
  - "@bgrant0607"
  - "@liggitt"
  - "@smarterclayton"
approvers:
  - TBD
editor: TBD
creation-date: 2019-10-01
last-updated: 2019-10-01
status: implementable
see-also:
replaces:
superseded-by:
---

# Require Transition from Beta

## Table of Contents

A table of contents is helpful for quickly jumping to sections of a KEP and for highlighting any additional information provided beyond the standard KEP template.

Ensure the TOC is wrapped with <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code> tags, and then generate with `hack/update-toc.sh`.

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
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

APIs and features should not languish in beta.  They should take feedback and progress towards GA by either
1. meeting GA criteria and getting promoted, or
2. having a new beta and deprecating the previous beta
  
This must happen within six months (two releases).  If it does not,
the API will be deprecated with an announced intent to remove the feature per the deprecation policy.

## Motivation

When a feature reaches beta, it is turned on by default.  This is great for getting feedback, but it can also lead to state
where users and vendors start building important infrastructure against APIs that are not considered stable.
In addition, once a feature is on by default, the incentive to further stabilize appears to diminish.
See the features that have been beta for a long time: CSRs and Ingresses as examples.
If we're honest with ourselves, a single actor has been cleaning up behind a lot of the project to unstick perma-beta APIs.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports

### Goals

1. Prevent APIs and features from being in a single beta version for more than six months.
2. Prevent beta APIs from being treated as GA by users and vendors.

### Non-Goals

1. Promote APIs to GA before they are ready.

## Proposal

Once an API or feature reaches beta, it has six months to 
1. reach GA and deprecate the beta or 
2. have a new beta version and deprecate the previous beta.

If neither of those conditions met, the beta API/feature is deprecated in the second release with a stated intent to remove the feature entirely.
To avoid removal, the feature must create a new beta version (it cannot go directly from deprecated to GA).

By regularly having new beta versions, we can ensure that consumers will not grow long running dependencies on particular betas which could pin design decisions.
It will also create an incentive for feature authors to push their features to GA instead of letting them live in a permanent beta state. 

### Risks and Mitigations

What are the risks of this proposal and how do we mitigate.
Think broadly.
For example, consider both security and how this will impact the larger kubernetes ecosystem.

How will security be reviewed and by whom?
How will UX be reviewed and by whom?

Consider including folks that also work outside the SIG or subproject.

## Drawbacks [optional]

Why should this KEP _not_ be implemented.

## Alternatives [optional]

Similar to the `Drawbacks` section the `Alternatives` section is used to highlight and record other possible approaches to delivering the value proposed by a KEP.

