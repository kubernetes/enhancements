(**FIXME:** Left here as a guideline, remove it before sending the PR)

> **Note:** When your KEP is complete, all of these comment blocks should be removed.
>
> To get started with this template:
>
> - [ ] **Pick a hosting SIG.**
>   Make sure that the problem space is something the SIG is interested in taking
>   up.  KEPs should not be checked in without a sponsoring SIG.
> - [ ] **Create an issue in kubernetes/enhancements**
>   When filing an enhancement tracking issue, please ensure to complete all
>   fields in that template.  One of the fields asks for a link to the KEP.  You
>   can leave that blank until this KEP is filed, and then go back to the
>   enhancement and add the link.
> - [ ] **Make a copy of this template directory.**
>   Copy this template into the owning SIG's directory and name it
>   `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
>   leading-zero padding) assigned to your enhancement above.
> - [ ] **Fill out as much of the kep.yaml file as you can.**
>   At minimum, you should fill in the "title", "authors", "owning-sig",
>   "status", and date-related fields.
> - [ ] **Fill out this file as best you can.**
>   At minimum, you should fill in the "Summary", and "Motivation" sections.
>   These should be easy if you've preflighted the idea of the KEP with the
>   appropriate SIG(s).
> - [ ] **Create a PR for this KEP.**
>   Assign it to people in the SIG that are sponsoring this process.
> - [ ] **Merge early and iterate.**
>   Avoid getting hung up on specific details and instead aim to get the goals of
>   the KEP clarified and merged quickly.  The best way to do this is to just
>   start with the high-level sections and fill out details incrementally in
>   subsequent PRs.
>
> Just because a KEP is merged does not mean it is complete or approved.  Any KEP
> marked as a `provisional` is a working document and subject to change.  You can
> denote sections that are under active debate as follows:
>
> ```
> <<[UNRESOLVED optional short context or usernames ]>>
> Stuff that is being argued.
> <<[/UNRESOLVED]>>
> ```
>
> When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
> focused.  If you disagree with what is already in a document, open a new PR
> with suggested changes.
>
> One KEP corresponds to one "feature" or "enhancement", for its whole lifecycle.
> You do not need a new KEP to move from beta to GA, for example.  If there are
> new details that belong in the KEP, edit the KEP.  Once a feature has become
> "implemented", major changes should get new KEPs.
>
> The canonical place for the latest set of instructions (and the likely source
> of this file) is [here](/keps/NNNN-kep-template/README.md).
>
> **Note:** Any PRs to move a KEP to `implementable` or significant changes once
> it is marked `implementable` must be approved by each of the KEP approvers.
> If any of those approvers is no longer appropriate than changes to that list
> should be approved by the remaining approvers and/or the owning SIG (or
> SIG Architecture for cross cutting KEPs).

# Kustomize Components

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Story](#user-story)
  - [Notes/Constraints/Caveats (optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

(**FIXME:** Left here as a guideline, remove it before sending the PR)

**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

(**FIXME:** Explain what is the current situation and then roughly explain what this feature does)

> This section is incredibly important for producing high quality user-focused
> documentation such as release notes or a development roadmap.  It should be
> possible to collect this information before implementation begins in order to
> avoid requiring implementors to split their attention between writing release
> notes and implementing the feature itself.  KEP editors, SIG Docs, and SIG PM
> should help to ensure that the tone and content of the `Summary` section is
> useful for a wide audience.
>
> A good summary is probably at least a paragraph in length.

## Motivation

(**FIXME:** Explain that we don't want to copy-paste stuff, list issues that wanted it at some form)

> This section is for explicitly listing the motivation, goals and non-goals of
> this KEP.  Describe why the change is important and the benefits to users.  The
> motivation section can optionally provide links to [experience reports][] to
> demonstrate the interest in a KEP within the wider Kubernetes community.
>
> [experience reports]: https://github.com/golang/go/wiki/ExperienceReports

### Goals

(**FIXME:** Reusable configuration, anything else?)

> List the specific goals of the KEP.  What is it trying to achieve?  How will we
> know that this has succeeded?

### Non-Goals

(**FIXME:** What to add here?)

> What is out of scope for this KEP?  Listing non-goals helps to focus discussion
> and make progress.

## Proposal

(**FIXME:** High-level explanation of how it works)

> This is where we get down to the specifics of what the proposal actually is.
> This should have enough detail that reviewers can understand exactly what
> you're proposing, but should not include things like API designs or
> implementation.  The "Design Details" section below is for the real
> nitty-gritty.

### User Story

(**FIXME:** Write here the example from the PR, minus the section for the current situation)

> Detail the things that people will be able to do if this KEP is implemented.
> Include as much detail as possible so that people can understand the "how" of
> the system.  The goal here is to make this feel real for users without getting
> bogged down.


### Notes/Constraints/Caveats (optional)

(**FIXME:** Write here the issue with files in the component directory)

> What are the caveats to the proposal?
> What are some important details that didn't come across above.
> Go in to as much detail as necessary here.
> This might be a good place to talk about core concepts and how they relate.

### Risks and Mitigations

(**FIXME:** The threat model is the same as with overlays, so there should not be any new risk)

> What are the risks of this proposal and how do we mitigate.  Think broadly.
> For example, consider both security and how this will impact the larger
> kubernetes ecosystem.
>
> How will security be reviewed and by whom?
>
> How will UX be reviewed and by whom?
>
> Consider including folks that also work outside the SIG or subproject.

## Design Details

(**FIXME:** Explain in greater detail how the feature works under the hood)

> This section should contain enough information that the specifics of your
> change are understandable.  This may include API specs (though not always
> required) or even code snippets.  If there's any ambiguity about HOW your
> proposal will be implemented, this is the place to discuss them.


### Test Plan

(**FIXME:** Mention that we will add unit tests)

> **Note:** *Not required until targeted at a release.*
>
> Consider the following in developing a test plan for this enhancement:
> - Will there be e2e and integration tests, in addition to unit tests?
> - How will it be tested in isolation vs with other components?
>
> No need to outline all of the test cases, just the general strategy.  Anything
> that would count as tricky in the implementation and anything particularly
> challenging to test should be called out.
>
> All code is expected to have adequate tests (eventually with coverage
> expectations).  Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
> when drafting this test plan.
>
> [testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md

### Graduation Criteria

(**FIXME:** Maybe have two major releases of Kustomize for alpha -> beta, write
user-facing documentation, etc. See
[here](https://github.com/arrikto/kubernetes-enhancements/blob/master/keps/sig-cli/20200115-kubectl-diff.md#graduation-criteria)
for an example)

> **Note:** *Not required until targeted at a release.*
>
> Define graduation milestones.
>
> These may be defined in terms of API maturity, or as something else. The KEP
> should keep this high-level with a focus on what signals will be looked at to
> determine graduation.
>
> Consider the following in developing the graduation criteria for this enhancement:
> - [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
> - [Deprecation policy][deprecation-policy]
>
> Clearly define what graduation means by either linking to the [API doc
> definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning),
> or by redefining what graduation means.
>
> In general, we try to use the same stages (alpha, beta, GA), regardless how the
> functionality is accessed.
>
> [maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
> [deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

## Implementation History

(**FIXME:** This section should be left empty, but not be removed)

> Major milestones in the life cycle of a KEP should be tracked in this section.
> Major milestones might include
> - the `Summary` and `Motivation` sections being merged signaling SIG acceptance
> - the `Proposal` section being merged signaling agreement on a proposed design
> - the date implementation started
> - the first Kubernetes release where an initial version of the KEP was available
> - the version of Kubernetes where the KEP graduated to general availability
> - when the KEP was retired or superseded

## Drawbacks

(**FIXME:** Unclear when to use components vs overlays, anything else?)

> Why should this KEP _not_ be implemented?

## Alternatives

(**FIXME:** Mention generators/transformers, sharing patches, possibly
@monopole's suggestion for merge behavior in resources, possibly @jbrette's
suggestion as well)

> What other approaches did you consider and why did you rule them out?  These do
> not need to be as detailed as the proposal, but should include enough
> information to express the idea and why it was not acceptable.
