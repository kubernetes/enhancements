# KEP-1553: Issue Triage Workflow and Automation

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
  - [Note to Reviewers](#note-to-reviewers)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Group Leads](#group-leads)
    - [Reviewers and Approvers](#reviewers-and-approvers)
    - [Contributors](#contributors)
  - [Notes/Constraints/Caveats (optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Graduation Criteria](#graduation-criteria)
  - [Phase 0](#phase-0)
    - [needs-triage and triage/accepted labels](#needs-triage-and-triageaccepted-labels)
    - [Remove or rename unused triage/** labels](#remove-or-rename-unused-triage-labels)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed](#infrastructure-needed)
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
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

To ease the burden of SIG/area reviewers/approvers, we would like to prescribe
a triage workflow and supporting automation.

### Note to Reviewers

For this `provisional` phase, the details of this KEP are left intentionally
light. There was significant discussion across issues, mailing list threads,
and PRs that didn't lead to forward progress because it seemed we were trying
to solve everything at once.

Here we attempt to scope a single deliverable before moving on to discussing
workflow, label states, and entry/exit criteria.

## Motivation

At present, there are [2,155 open issues and 902 pull requests](https://github.com/kubernetes/kubernetes/issues?utf8=%E2%9C%93&q=is%3Aopen) open in
`kubernetes/kubernetes`.

- [356 are categorized as `lifecycle/stale`](https://github.com/kubernetes/kubernetes/labels/lifecycle%2Fstale)
- [246 are categorized as `lifecycle/rotten`](https://github.com/kubernetes/kubernetes/labels/lifecycle%2Frotten)
- [690 are categorized as `lifecycle/frozen`](https://github.com/kubernetes/kubernetes/labels/lifecycle%2Ffrozen)

This makes for about 20% of open issues/PRs that are in some state of staleness.
If we consider items marked as `lifecycle/frozen`, then we're looking at around
42% of issues/PRs that could potentially require attention.

### Goals

See [User Stories](#user-stories).

<<[UNRESOLVED]>>

### Non-Goals

- Prescribing an issue triage workflow for all projects within Kubernetes orgs

<<[/UNRESOLVED]>>

## Proposal

### User Stories

#### Group Leads

As a SIG Chair/Technical Lead, WG/UG organizer, or subproject owner, I want to
be able to easily triage my group's backlog of open issues and PRs to accelerate
merge velocity and issue resolution.

#### Reviewers and Approvers

As a reviewer/approver, I want a simple system to categorize issues/PRs in my
purview, so that I can prioritize what to review first.

#### Contributors

As a contributor, I want to be able to submit issues or PRs and:

- understand how the community works to address new submissions
- have some assurance that they will be routed to the correct group
- have them addressed in a timely manner

<<[UNRESOLVED]>>

### Notes/Constraints/Caveats (optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they releate.
-->

<<[/UNRESOLVED]>>

<<[UNRESOLVED]>>

### Risks and Mitigations

<!--
What are the risks of this proposal and how do we mitigate.  Think broadly.
For example, consider both security and how this will impact the larger
kubernetes ecosystem.

How will security be reviewed and by whom?

How will UX be reviewed and by whom?

Consider including folks that also work outside the SIG or subproject.
-->

<<[/UNRESOLVED]>>

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable.  This may include API specs (though not always
required) or even code snippets.  If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*
-->

_Not required in `provisional` state._

### Phase 0

#### needs-triage and triage/accepted labels

We currently have a Prow plugin called `require-matching-label`, which requires
specific labels to be set on a issue or pull request.

Examples in [kubernetes/kubernetes] include:

- `needs-kind` (needs a `kind/foo` label)
- `needs-priority` (needs a `priority/bar` label)
- `needs-sig` (needs a `sig/baz` label)

Here we propose [introducing two new labels](https://github.com/kubernetes/test-infra/pull/16298):

- `needs-triage`
- `triage/accepted`

The `needs-triage` label would be automatically applied to incoming issues.

Current contributors would then need to inspect issues with the `needs-triage`
label and attempt to triage them.

Upon determining that an issue is ready to be actively worked on, an org member
can apply the `triage/accepted` label using the following bot command:

```shell
/triage accept
```

(Alternatively, contributors with write access to `kubernetes/kubernetes` would be
able to manually apply the `triage/accepted` label.)

**Considerations:**

- `triage/accepted` should be considered a placeholder name for the
  label in this provisional state, depending on what we decide is the most
  appropriate name.
- There will be an expectation that only SIG members designated to triage
  issues are applying triage labels. We're considering limiting application of
  this label to the [`milestone-maintainers` GitHub team](https://github.com/orgs/kubernetes/teams/milestone-maintainers).

From here, a group can search the open items labeled with
`triage/accepted` and proceed to work on them.

A nice example of a written SIG workflow is the [grooming document from
SIG Cluster Lifecycle](https://git.k8s.io/community/sig-cluster-lifecycle/grooming.md).

#### Remove or rename unused triage/** labels

As we're considering reviving the `triage/**` labels in this workflow, it's
important that all labels with this affix are up-to-date, removing any
`triage/**` label deemed to be unused.

The current list of `triage/**` labels is as follows:

- `triage/duplicate`
- `triage/needs-information`
- `triage/not-reproducible`
- `triage/support`
- `triage/unresolved`

We propose here removing the following labels:

- `triage/duplicate`
- `triage/not-reproducible`
- `triage/unresolved`

We rename the following labels:

- `triage/support` --> `kind/support`: Issue has been identified as a support
  question, which should be routed to the appropriate forum and closed
  immediately afterwards
- `triage/needs-information` --> `lifecycle/needs-information`: Issue requires
  more information (from submitter or SIG) in order to work on it

Which leaves the following label and accompanying definition:

- `triage/accepted`: Issue has been triaged by a SIG representative and
  is ready to be worked

## Implementation History

- 2020-02-15 - [Issue Triage KEP submitted as `provisional`](https://github.com/kubernetes/enhancements/pull/1554)
- 2020-02-15 - [Issue Triage Enhancement issue](https://github.com/kubernetes/enhancements/issues/1553) opened
- 2020-02-14 - [Carry PR opened](https://github.com/kubernetes/test-infra/pull/16298) for `needs-triage` and `triage/accepted` labels
- 2020-02-14 - [Carry PR opened](https://github.com/kubernetes/test-infra/pull/16299) to rename `triage/**` labels to `close/**`
- 2019-05-30 - [Original PR opened](https://github.com/kubernetes/test-infra/pull/12814) to rename `triage/**` labels to `close/**`
- 2019-05-01 - ["Deadlines are horrible"](https://groups.google.com/d/topic/kubernetes-sig-release/dGVBrlkOXQo/discussion) discussion
- 2019-03-18 - [Original PR opened](https://github.com/kubernetes/test-infra/pull/11818) for `needs-triage` and `lifecycle/ready` labels
- 2019-03-18 - [k/k-wide triage workflow improvements](https://github.com/kubernetes/community/issues/3456) issue opened
- 2019-03-08 - ["Issue triage again"](https://groups.google.com/d/topic/kubernetes-sig-contribex/BvGmOQ0v5f0/discussion) discussion
- 2016-11-14 - [Initial mailing list](https://groups.google.com/d/topic/kubernetes-sig-contribex/mI7kuTFa_I4/discussion) discussion on issue triage

## Drawbacks

N/A

<<[UNRESOLVED]>>

## Alternatives

<!--
What other approaches did you consider and why did you rule them out?  These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

<<[/UNRESOLVED]>>

## Infrastructure Needed

At the current stage of this proposal, Prow already supports the behaviors we require:

- Adding new labels
- Enforcing label requirements on issues/PRs
