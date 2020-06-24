# KEP-617: Enhance the KEP implementation

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [1) Standardize an intra-KEP denotation for &quot;not resolved&quot;](#1-standardize-an-intra-kep-denotation-for-not-resolved)
  - [2) Change the names of KEPs](#2-change-the-names-of-keps)
  - [3) Move KEPs from flat-files to a directory structure](#3-move-keps-from-flat-files-to-a-directory-structure)
  - [4) Metadata split from KEP text](#4-metadata-split-from-kep-text)
  - [5) Remove some fields from metadata](#5-remove-some-fields-from-metadata)
  - [6) Formalize one-enhancement == one-KEP](#6-formalize-one-enhancement--one-kep)
  - [7) Adjust GitHub tooling for KEPs](#7-adjust-github-tooling-for-keps)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

- [ ] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] KEP approvers have approved the KEP status as `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Drive KEP adoption through improved process, documentation, visibility, and
automation.

KEP artifacts are difficult to navigate.  It's hard to find which KEPs are in
which states and to know what is considered undecided.  This proposal aims to
define several small, incremental steps that can be taken to make these
problems a bit better.

## Motivation

The KEP process is the standardized structure for proposing changes to the
Kubernetes project.  This KEP seeks to define actionable / delegable items to
move the process forward.

Additionally, a group of people from sig-arch started a KEP reading group.  We
found it very difficult to keep track of what was happening with KEPs.  We
identified a number of specific suggestions to make consuming KEPs more
friendly.  History can be found in [this thread](https://groups.google.com/forum/#!msg/kubernetes-sig-architecture/SZ7UcFuOtCI/PQlTeSCICAAJ).

### Goals

To make it easier for humans and automation to use, find, reference, and
consume KEPs.

### Non-Goals

- API Review process
- Feature request triage
- Developer guide

## Proposal

This proposal is somewhat omnibus.  Individual items are detailed below.

### 1) Standardize an intra-KEP denotation for "not resolved"

There is a perception that "merge and iterate" can be used to avoid dissent.
If one topic is under debate, but other parts are acceptable, it should be
possible to merge a KEP and iterate on the debatable point.  However, without
reading the entire history of PRs on a KEP, exactly what is "under debate" is
not clear.

Requirements:
  - not an HTML comment (so it is visible in the rendered markdown)
  - unambiguous
  - can wrap large bodies of text

Parts of a KEP that are being merged without consensus shall be denoted as
follows:

```
<<[UNRESOLVED optional text]>>
Stuff that is under debate.
<<[/UNRESOLVED]>>
```

The "optional text" can include a list of names or other context, but should be
short.

### 2) Change the names of KEPs

The date-format in the name is not very human-friendly and doesn't do anything
for machines.  Instead, we propose to use the issue number of the enhancement
issue as the KEP number.  KEP names would now be:
`1234-short-human-friendly-string`.  This makes them easier to reference and
remember: "KEP 1234" is easier than "KEP 20190903".  We will not retroactively
change existing KEPs, so as to not break external links to them.

### 3) Move KEPs from flat-files to a directory structure

It's sometimes useful to have secondary documents—for example,
image files—alongside a KEP.  To make this cleaner, new KEPs will be created in their own
sub-directories of the SIG-specific directories. As per the above, a KEPs
directory would be named as `1234-short-human-friendly-string`.  Inside that
directory, the body of the KEP would be `README.md`.  Supporting documents and
images would go in that directory.

As part of this update, we will move all existing KEPs into this new structure,
and leave links to the new locations in the old files.  This will preserve
in-bound links and tell users where to find the information.

### 4) Metadata split from KEP text

Today, KEP metadata is a header on the markdown file.  This proposal moves that
metadata to a new `kep.yaml` in each KEP directory (see above).  This file will
contain the same information that was previously in the YAML front-matter.

### 5) Remove some fields from metadata

Almost no KEPs actually set `editor` at all.  This proposes to remove it from
the template.

The `last-updated` field can be derived from git history.

The `superseded-by` field is never set and can be implied by the corresponding
`replaces` field.

### 6) Formalize one-enhancement == one-KEP

There has been debate about whether a new KEP is needed to move a feature from beta
to GA.  This proposal defines that one feature or enhancement gets one KEP,
which is updated over its whole lifecycle.

### 7) Adjust GitHub tooling for KEPs

- [x] Move existing KEPs into [k/features]
- [x] Create a `kind/kep` label for [k/community] and [k/features]
  - For `k/community`:
    - [ ] Label incoming KEPs as `kind/kep`
    - [ ] Enable searches of `org:kubernetes label:kind/kep`, so we can identify active PRs to `k/community` and reroute the PR authors to `k/enhancements` (depending on the state)
  - For `k/enhancements` (fka `k/features`):
    - [ ] Label incoming KEPs as `kind/kep`
    - [ ] Classify KEP submissions / tracking issues as `kind/kep`, differentiating them from `kind/feature`
- [x] Move existing design proposals into [k/features]
- [ ] Move existing architectural documents into [k/features] (process TBD)
- [ ] Deprecate design proposals
- [x] Rename [k/features] to [k/enhancements]
- [ ] Create tombstones / redirects to [k/enhancements]
- [ ] Prevent new KEPs and design proposals from landing in [k/community]
- [ ] Remove `kind/kep` from [k/community] once KEP migration is complete
- [x] Correlate existing feature tracking issues with links to KEPs
- [x] Fix [KEP numbering races] by using the GitHub issue number of the KEP tracking issue
- [ ] Coordination of existing KEPs to use new directory structure

[k/community]: http://git.k8s.io/community
[k/enhancements]: http://git.k8s.io/enhancements
[k/features]: http://git.k8s.io/features

### Risks and Mitigations

Changing conventions makes for inconsistency.  We believe the quality-of-life
improvements make these changes worthwhile anyway.

## Design Details

The above proposal covers most of the details.

We will need tooling to move existing KEPs into the new structure.

### Test Plan

N/A

### Graduation Criteria

Throughout implementation, we will reach out across the project to SIG
leadership, approvers, and reviewers to capture feedback.

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

N/A

## Implementation History

- 2018-08-20: (@timothysc) Issue filed about repo separation: https://github.com/kubernetes/community/issues/2565
- 2018-08-30: SIG Architecture meeting mentioning the need for a clearer KEP process - https://youtu.be/MMJ-zAR_GbI
- 2018-09-06: SIG Architecture meeting agreeing to move forward with a KEP process improvement effort to be co-led with SIG PM (@justaugustus / @jdumars) - https://youtu.be/fmlXkN4DJy0
- 2018-09-10: KEP-1a submitted for review
- 2018-09-25: Rationale discussion in SIG PM meeting
- 2018-09-28: Merged as `provisional`
- 2018-09-29: KEP implementation started
- 2018-09-29: [KEP Implementation Tracking issue] created
- 2018-09-29: [KEP Implementation Tracking board] created
- 2018-09-29: Submitted as `implementable`
- 2020-02-07: Add UNRESOLVED and simplify dir structure

## Drawbacks

Changing conventions makes for inconsistency.  We believe the quality-of-life
improvements make these changes worthwhile anyway.

## Alternatives

There are many partial solutions to these problems.

Leaving unresolved text out completely or HTML-commented could work, but we
feel it is more useful to actually show that text and emphasize its
non-resolvedness.

Renaming KEPs without making subdirectories incurs most of the same pain with
less gain.

Not fixing tooling will incur more of the same pain we suffer today.

We have already tried a "surrogate" KEP number mechanism, which gave way to the
current date mechanism.

## Infrastructure Needed (optional)

N/A
