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
# KEP-3136: Beta APIs Are Off by Default

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
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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

From the Kubernetes release where this change is introduced, and onwards, beta APIs will not be enabled in clusters by default.
Existing beta APIs and new versions of existing beta APIs, will continue to be enabled by default: 
if v1beta.some.group is currently enabled by default and we create v1beta2.some.group, v1beta2.some.group will still be enabled by default.

## Motivation

Beta APIs are not considered stable and reliance upon APIs in this state leads to exposure to bugs,
guaranteed migration pain for users when the APIs move to stable, and the risk that dependencies will
grow around unfinished APIs.
Enabling beta APIs by default, exacerbates these problems by making them on in nearly every cluster.
We observed these problems as we removed long-standing beta APIs and the PRR survey tells us that over
90% of cluster-admins leave production clusters with these APIs enabled.
Unsuitability for production use is documented at https://kubernetes.io/docs/reference/using-api/#api-versioning 
("The software is not recommended for production uses"), but defaulting on means they are present in nearly every 
production cluster.
By disabling beta APIs by default, a cluster-admin can opt-in for specific APIs without having every
incomplete API present in the cluster.
This is now practical to do since conformance no longer relies on non-stable APIs.

### Goals

1. Disable new beta APIs by default.
2. Continue enabling existing beta APIs and new version of existing beta APIs by default:
   if v1beta.some.group is currently enabled by default and we create v1beta2.some.group, v1beta2.some.group will still be enabled by default.
3. Allow enabling specific resources in beta.  Enable coolnewjobtype.v1beta1.batch.k8s.io without enabling other-neat-job.v1beta1.batch.k8s.io

### Non-Goals

1. Change feature gate defaults.
   Feature gates control new features (not just new APIs) and they are on by default for beta features.
   This KEP is not changing the lifecycle flow for feature gates.
   It is currently alpha=off-by-default, beta=on-by-default, stable=locked-to-on.

## Proposal

New beta APIs will be placed into the `DisableVersions` stanza instead of the `EnableVersions` stanza (see [DefaultAPIResourceConfigSource](https://github.com/kubernetes/kubernetes/blob/0669da445fa8c1ae07c15c0827f0e83da11cbe58/pkg/controlplane/instance.go#L643)).
The `--runtime-config` flag will be extended to allow `group/version/resource=true`, to enable specific resources.
To enable a beta API, a cluster-admin will have to add the appropriate `--runtime-config` flags.

### User Stories (Optional)

#### Story 1

As a cluster-admin I want to enable the coolnewjobtype.v1beta1.batch.k8s.io API in my cluster.

To do this I call `kube-apiserver --runtime-config=batch.k8s.io/v1beta1/coolnewjobtype`.

#### Story 2

As a cluster-admin I want to enable all beta APIs as in past releases.

To do this I call `kube-apiserver --runtime-config=api/beta=true`.
This already exists and will continue to function.


### Notes/Constraints/Caveats (Optional)

Installers, utilities, controllers, etc that need to know if a certain beta API is present can continue to use the
existing discovery mechanisms (example: kubectl's api-resources sub command or the `/api/apps/v1` REST API).

### Risks and Mitigations

Adoption of beta features will slow.
Given how kubernetes is now treated, this is a good thing, not a bad thing.
Those users that want to move quickly and get new features can do so by enabling all beta feature
or just enabling those that are important for their workload.
The [PRR survey](https://datastudio.google.com/reporting/2e9c7439-202b-48a9-8c57-4459e0d69c8d/page/Cv5HB) shows that 
over 30% of cluster-admins have enabled alpha features on at least some production clusters, so cluster-admins are willing and able to enable features
that are not on by default when they are desired.

If two or more APIs are tightly coupled together, it will now be possible to enable them independently.
This can lead to unanticipated failure modes, but should only impact beta APIs with beta dependencies.
While this is a risk, it is not very common and components should fail safe as a general principle.

If beta APIs are off by default, it's possible that fewer clients will use them and provide feedback.
This is a risk, but early adopters are able to enable these features and have a history of enabling alpha features.
When moving from beta to GA, it will be important for sigs to explicitly seek feedback.
We will address this by extending the PRR questionnaire to include a GA-targeted question to validate that the feature
was reasonably validated in production use-cases.

If beta APIs are off by default, it is possible that sigs don't treat taking an API as an indication of a "mostly-baked" API.
If this happens, then more transformation may be required.
Keeping our beta API rules consistent and continuing to enforce easy to use APIs seems to be the best option.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Test Plan

Integration tests will be written to ensure that no new beta APIs are enabled in the kube-apiserver by default.
Unit tests will be written to ensure that the new flag functionality works as expected.

### Graduation Criteria

This KEP is a policy KEP, not a feature KEP.  It will start as GA.

#### GA

- Integration and unit tests from above.
- updating the enablement docs for beta
  - https://kubernetes.io/docs/reference/using-api/#api-versioning
  - https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/#using-a-feature 
    Even though that is talking about feature gates, it is likely worth calling out there that new beta REST APIs are no
    longer enabled by default)
- email to dev@kubernetes.io to explain the new policy
- blog post explaining change in time for 1.24 release
- CI configuration updated to have a testing mode that enables beta APIs, likely set using `kube-apiserver --runtime-config=api/beta=true`
- extend the PRR questionnaire to include a GA-targeted question to validate that the feature was reasonably validated in production use-cases.

### Upgrade / Downgrade Strategy

The additional command line flag format for `--runtime-config` will not be recognized on older levels of kubernetes.
This means that when downgrading, cluster-admins will have to adjust their CLI arguments if they opted into a new beta API.
This is congruent to flag handling for new features today.
Because this only impacts new beta APIs, there is no behavior change for existing APIs on upgrade.

### Version Skew Strategy

Because this only impacts new beta APIs, there is no novel skew risk.

## Production Readiness Review Questionnaire

Not applicable because this is a policy KEP.

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

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
