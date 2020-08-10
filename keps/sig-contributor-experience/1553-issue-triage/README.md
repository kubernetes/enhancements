# KEP-1553: Issue Triage Workflow and Automation

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [New Workflow](#new-workflow)
  - [Re-categorize the triage/support label](#re-categorize-the-triagesupport-label)
  - [User Stories](#user-stories)
    - [Group Leads](#group-leads)
    - [Reviewers and Approvers](#reviewers-and-approvers)
    - [Contributors](#contributors)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [needs-triage and triage/accepted labels](#needs-triage-and-triageaccepted-labels)
  - [Rename the triage/support label](#rename-the-triagesupport-label)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

- [ ] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] KEP approvers have approved the KEP status as `implementable`
- [ ] Design details are appropriately documented
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] Contributor documentation has been created in [kubernetes/community]

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

To ease the burden of SIG/area reviewers/approvers, we would like to prescribe
a triage workflow and supporting automation.

## Motivation

As of 09-08-2020, there are [3,004 open issues and 975 pull requests](https://github.com/kubernetes/kubernetes/issues?utf8=%E2%9C%93&q=is%3Aopen) open in
`kubernetes/kubernetes`.

- [197 are categorized as `lifecycle/stale`](https://github.com/kubernetes/kubernetes/labels/lifecycle%2Fstale)
- [200 are categorized as `lifecycle/rotten`](https://github.com/kubernetes/kubernetes/labels/lifecycle%2Frotten)
- [756 are categorized as `lifecycle/frozen`](https://github.com/kubernetes/kubernetes/labels/lifecycle%2Ffrozen)

This makes for about 13% of open issues/PRs that are in some state of staleness.
If we consider items marked as `lifecycle/frozen`, then we're looking at around
38% of issues/PRs that could potentially require attention.

### Goals

Streamline the issue triage process used within the [kubernetes/kubernetes]
repository.

- Create new workflow centered around `needs-triage` label auto-applied to new
  issues.

See [User Stories](#user-stories) for more information.


### Non-Goals

- Prescribing an issue triage workflow for all projects within Kubernetes orgs.
- Requiring all community groups within Kubernetes to adhere to the issue triage
  workflow introduced in this KEP.

## Proposal

### New Workflow

An additional [required label] called `needs-triage` will be applied automatically
to issues created within the [kubernetes/kubernetes] repository similar to the
current `needs-sig` or `needs-kind` labels. This serves as a boolean signal to
community group members that the issue has not yet been triaged.

After the issue has been evaluated, an org member can apply one of the `triage`
labels. If there is enough information or supporting evidence in the issue a
member can signal that it is ready for work by using the `/triage accepted` bot
command to apply the `triage/accepted` label. If the issue is a duplicate or
lacks supporting evidence one of the other [triage labels] can be applied.

In either condition the `needs-triage` label will be removed.


### Re-categorize the triage/support label

The label `triage/support` will become `kind/support`. `kind` better reflects
the class or or type of issue.

**Note:** This is a revert of the previous decision to move the `support` label
from `kind/*` to `triage/*` introduced in [kubernetes/test-infra#7598]. The
goal at that time, was to use the label as a signal that the issue was something
that should be closed. This is still true for [kubernetes/kubernetes], but the
`triage/support` label's usage has grown outside of the [kubernetes/kubernetes]
repo.


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


### Risks and Mitigations


**The new labels and process are either ignored or fall into dis-use.**

There are close to 200 labels associated with the [kubernetes/kubernetes]
repository. While this removes some of the labels it does introduce new ones and
adds an additional process. Both of these could potentially go unused or ignored
without effort made by the community groups to use them appropriately.

When the new process is ready to be put into place, the [upstream marketing team]
will be engaged to ensure there is clear communication regarding the changes.
Their message will be backed by updated documentation regarding the new
[issue triage process].


## Design Details

### needs-triage and triage/accepted labels

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
/triage accepted
```

(Alternatively, contributors with write access to `kubernetes/kubernetes` would be
able to manually apply the `triage/accepted` label.)

**Considerations:**

Any Org member may use the `/triage` command in good faith that the issue is
valid and belongs to the owning community group associated with the issue. Should
it be widely misused, the command may be restricted to the [`milestone-maintainers`]
or another to-be-determined GitHub team in the future.

From here, a group can search the open items labeled with
`triage/accepted` and proceed to work on them.

A nice example of a written SIG workflow is the [grooming document from
SIG Cluster Lifecycle](https://git.k8s.io/community/sig-cluster-lifecycle/grooming.md).


### Rename the triage/support label

The label `triage/support` will become `kind/support`. `kind` better reflects
the class or or type of issue.

For more information on this decision see the [Proposal](#proposal).


### Graduation Criteria

This is a workflow change that will be rolled out at one time. It is not expected
to go through the `alpha` / `beta` / `stable` stages and will instead go right
to stable.



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

- It is yet-another-label for our contributors to keep track of.
- A new contributor may not understand why it is applied and who can resolve it.

## Alternatives

Potential workflows could be built with the current set of labels in use. For
example, issue triage teams could apply `triage/unresolved` to open issues that
have not yet been acted upon. This process is done currently for `sig/network`
issues automatically by [Athenabot].

## Infrastructure Needed

At the current stage of this proposal, Prow already supports the behaviors we require:

- Adding new labels
- Enforcing label requirements on issues/PRs

[triage labels]: https://github.com/kubernetes/kubernetes/labels?q=triage
[kubernetes/test-infra#7598]: https://github.com/kubernetes/test-infra/issues/7598
[kubernetes/community]: https://git.k8s.io/community
[required label]: http://git.k8s.io/test-infra/prow/plugins/require-matching-label/require-matching-label.go
[issue triage process]: https://git.k8s.io/community/contributors/guide/issue-triage.md
[Athenabot]: https://github.com/athenabot/k8s-issues#what-it-does
[`milestone-maintainers`]: https://github.com/orgs/kubernetes/teams/milestone-maintainers
