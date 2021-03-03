# KEP-2539: Continuously Deploy K8s Prow

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
    - [Prow Users](#prow-users)
    - [Prow Oncall](#prow-oncall)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [Breaking Changes in Prow](#breaking-changes-in-prow)
- [Design Details](#design-details)
    - [Automated Merging of Prow Autobump PRs](#automated-merging-of-prow-autobump-prs)
    - [Roll Back Process](#roll-back-process)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
    - [A new tool merges autobump PRs](#a-new-tool-merges-autobump-prs)
      - [Pros:](#pros)
      - [Cons:](#cons)
<!-- /toc -->

## Release Signoff Checklist

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


[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This document proposes to change deployment of k8s prow from manual to be automated continuously.

## Motivation

Currently, deploying k8s prow consists of following steps:

1. Updates made in prow are released as container images.
1. Automated process creates PRs updating the prow images tags.
1. Oncall inspects prow logs to make sure prow is safe to upgrade.
1. Oncall inspects PRs created in step #2, approve them, and post message on Slack.
1. Oncall waits until #4 and a postsubmit job applying the changes on prow cluster.
1. Oncall waits until #5 is done, do several manual inspections to make sure prow works.

This is a very time consuming process (Roughly 3 hours per week for oncall), especially the context switching between the waits makes oncall hard to focus on their day job. Thus it’s desired to streamline the process so that we can save time.

Historically, the biggest reasons why the manual processes were introduced are:

- Errors in prow were not easily discoverable, most of time were reported by prow users, which was bad.

This problem has been largely solved by the introduction of prow monitoring + alerting by grafana, prometheus, and prometheus alertmanager stack. And based on our experience in the past quarter, we haven’t had a single case where prow errors were discovered by humans earlier than by prow alerts. This fact gives us reasonable confidence to proceed with continuous delivery, and the following assumptions should hold:

- Prow is stable as long as there is no alert. (Indicate that no need to inspect prow logs before bumping)
- Errors caused by prow upgrades are discovered by prow alerts in a timely manner.


### Goals

The proposed change switches from daily manual deployment to hourly automated deployment.

#### Prow Users

Shouldn’t see any change, prow breakage should be discovered by prow monitoring system and rollback will be performed. The chance of prow being break is almost identical to what we have today(Assume there are not more than a single breaking change every day).

#### Prow Oncall

- What’s Not Changed
  - React to prow alerts and take actions.
- What’s Changed
  - No more manual inspecting prow healthiness.
  - No more manual lgtm/approve/retest autobump PRs.
  - No more manual Slack posting.


### Non-Goals

Change how prow is released.


## Proposal

Prow autobump PRs are automatically merged every hour, only on working hours of working days.

### Notes/Constraints/Caveats (Optional)

#### Breaking Changes in Prow
Breaking changes in prow will require manual intervention. Currently prow isn’t able to handle these intelligently, as it was not designed with the mindset of API versions and thus kubernetes conversion webhook can not help coping with breaking changes among major APIs.
One possible way of dealing with breaking changes, is:
- Prow oncall inspects prow logs and breaking changes announcements once per week, and take actions based on deprecation warnings from prow logs and breaking changes from ANNOUNCEMENTS.md.
- [Stretch Goal][Push left] Discover breaking changes, especially configs or flag changes in prow integration test (This requires prow integration test use the same set of deployment configs as prod)
- [Optional] (This is not very reliable) Prow TLs inspects new PRs, manually identifies possible breaking changes and informs oncall for awareness. Either prow TL or oncall can take deeper look at the new PR and decide whether to take actions or  not.

## Design Details

#### Automated Merging of Prow Autobump PRs

- Prow autobump job is already configured to run on work days only, change it to at least one hour apart, so that it doesn’t bump more frequently than one hour.
- Tide blindly trusts PRs from the bot that does autobump, merging the PR as long as tests all pass. Flaky tests will be covered by auto-pushing of new autobump jobs later.

This approach uses tide auto-merge feature, so that no need to worry about repo requirements such as need more than one approver etc.

```
<<[UNRESOLVED (spiffxp) ]>>
Suggestion: how to keep slack reports on each automated bump.
<<[/UNRESOLVED]>>
```

#### Roll Back Process

When prow stopped functioning after a bump, prow oncall should:
- Stop auto-deploying by commenting `/hold` on latest autobump PR.
- Manually create rollback PR for rolling back to known good version.
- Manually apply the changes from rollback PR.

```
<<[UNRESOLVED]>>
Which version to roll back. This is generally not a problem due to low release volume of prow. @alvaroaleman suggested 6 hours intervals.
<<[/UNRESOLVED]>>
```

## Implementation History


## Alternatives


#### A new tool merges autobump PRs
This method is independent of tide, which makes sure it works on every prow instance.

##### Pros:
Not relying on tide, works really well with prow instances that don't have tide.

##### Cons:
Probably have significantly divergent code paths for finding and approving PRs on Gerrit vs PRs on GitHub.
