---
title: New Cronjob Concurrency Policy - CatchUp
authors:
  - "@kolorful"
owning-sig: sig-apps
participating-sigs:
  - TBD
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-07-16
last-updated: 2019-07-16
status: provisional
see-also:
replaces:
superseded-by:
---

# New Cronjob Concurrency Policy - CatchUp

## Table of Contents
<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
  - [Implementation History](#implementation-history)
  - [Drawbacks [optional]](#drawbacks-optional)
  - [Alternatives](#alternatives)
- [Infrastructure Needed [optional]](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

This document proposes a plan to add a new concurrency policy `CatchUp` to Cronjob,
such that missed jobs are not skipped and will be scheduled in sequential order.
Additionally, two derived minor improvements are proposed to better support this feature: 
(1) Expose the original planned run time to Jobs.
(2) Only conduct the "maximum 100 missed schedules" check while concurrency policy is set to `Allow`.

## Motivation

Currently if any schedule misses, cronjob controller will skipped it immediately no matter what concurrency policy is in use.
This behavior, however, does not fit for cronjobs that need to know the exact time range of the data they are going to process.
For example, I have a cronjob that runs hourly and only processes the logs within the previous hour.
Say the job now takes 3 hours to run due to a bug, then many jobs will be skipped sporadically,
and users have to manually find out and backfill the missing pieces in the middle.

In fact this type of stateful behavior is yet supported in Kubernetes at the moment,
and it causes us not able to migrate all of our cronjobs to Kubernetes.
Since it is a pretty common practice (e.g. [Airflow](https://airflow.apache.org/scheduler.html#backfill-and-catchup)),
we would like to enable it with minimum intrusion of the system. Therefore, the original planned run time approach is proposed.

Additionally, existing "maximum 100 missed schedules" limit seems a bit problematic.
First of all, [the scenario it tries to prevent](https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/cronjob/utils.go#L131)
does not exist in current code logic. According to the [code](https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/cronjob/cronjob_controller.go#L280),
regardless of the concurrency policy and number of unmet schedules,
the controller only picks the latest one. Secondly, the goal of `CatchUp` policy is to keep up missed schedules,
in other words, if there are 101 missed schedules for a cronjob, we should eventually run them all sequentially.

### Goals

* New `CatchUp` concurrency policy will allow missed jobs to be scheduled sequentially instead of being skipped.
* Let the jobs know when they ware suppose to run.
* "Maximum 100 missed schedules" will only be checked when concurrency policy is set to `Allow`.

### Non-Goals

Making "maximum missed schedules" configurable. This can be its own KEP since it is not directly related to this proposal.

## Proposal

We will add a new concurrency policy: `CatchUp` in addition to `Allow`, `Forbid`, `Replace`. In `CatchUp` mode,
when cronjob controller sees a list of missed schedules, it will schedule them sequentially from the oldest to the latest.

When cronjob controller creates a Job, it will pass the original planned run time to it. This could be as simple as an annotation.
Users can then implement a webhook to compute and inject the desired start/end time information to Job.


We will also update the cronjob controller so that it only checks
whether a cronjob has exceeds the maximum allowed missed schedules if the concurrency policy is set to `Allow`.
This means:
* If concurrency policy is set to `Allow`, no new behavior is introduced.
* If concurrency policy is set to `Forbid`, and say there are 200 missed schedules.
Instead of complaining about `Cannot determine if job needs to be started. Too many missed start time (> 100). Set or decrease .spec.startingDeadlineSeconds or check clock skew.`
It will skip all missed jobs and fast forward to run the latest.
* If concurrency policy is set to `Replace`, the behavior is similar to `Forbid` except it will replace the existing job with latest.

### User Stories

#### Story 1
Users can leverage the exposed original scheduled run time info to set up more sophisticated cronjob that needs to know
the exact time range of the data it is going to process.

#### Story 2
Users do not need to worry about whether there are any missing jobs, nor to go through the hassle of backfilling, 
since cronjob controller will catch up all the missed ones automatically.

### Implementation Details/Notes/Constraints

Adding a new option for concurrency policy, need to update validation as well.

Expose Job's original planned run time by adding an annotation on Job like `job-planned-start-time: 1564309800`.

While making cronjob controller only check "maximum missed schedules" on `concurrencyPolicy: Allow`,
we should not compute a full list of unmet schedules as current implementation since it is very inefficient.

### Risks and Mitigations
This is a pretty extreme case. If the last scheduled time of a cronjob is accidentally set to 1970/01/01, and it runs every minute.
When concurrency policy is set to `Forbid` or `Repalce`,
cronjob controller needs to compute and skip roughly 25 million (49 years * 365 days * 24 hours * 60 minutes) dates,
but it should only happen once for that cronjob.

## Design Details

### Test Plan

This feature will be tested with a combination of unit, integration and e2e
tests. In particular:

* Field validation
* Scheduling behavior correctness validation
* TBA

### Graduation Criteria
TBA

### Upgrade / Downgrade Strategy
N/A

### Version Skew Strategy
N/A

### Implementation History
TBA

### Drawbacks [optional]
N/A

### Alternatives

We have considered to add a boolean `catchUp` in Cronjob's spec, but it does not make sense with `concurrencyPolicy: Replace`.

## Infrastructure Needed [optional]
N/A
