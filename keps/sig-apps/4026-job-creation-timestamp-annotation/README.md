# KEP-4026: Add job creation timestamp to job annotations

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

Currently, there is no supported way to get the original/expected initial scheduled timestamp for the job created from a cronjob. This KEP proposes to set the the original scheduled time as an annotation in the job metadata.

## Motivation

### Goals

- Set job scheduled timestamp as an annotation on the job.
- Adding the annotation should not be disruptive to existing workloads.

### Non-Goals

## Proposal

At a high level, the proposal is to modify the CronJob controller to set the job scheduled timestamp as a job annotation. The details of this are outlined in the Design Details section below.

Job scheduled timestamp annotation: `batch.kubernetes.io/cronjob-scheduled-timestamp`

### User Stories (Optional)

#### Story 1

As a user, I would like to get the job's scheduled timestamp that this job was expected to be running.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

One thing that must be considered is how enabling this new feature will interact with existing workloads. There are a couple of options:

1. Only inject the job annotation for *newly created CronJobs*. We can track this by annotating newly created CronJobs to distinguish existing ones from newly created ones.
Using this strategy, either none of the jobs have this annotation, or all of them do, which will provide a more consistent user experience. However, in the case of a cluster downgrade to a version without this feature, new jobs would start getting created without this annotation again.

1. Inject the annotation on *all jobs* (jobs existing prior to feature enablement and jobs created after feature enablement). However, retroactively modifying jobs of existing workloads would risk being too disruptive to existing workloads which may have logic depending on job annotations, so this option should not be considered.

Both options 1 and 2 will not be disruptive to existing workloads. Option 1 is more straightforward and does not risk locking us into adding this somewhat
hacky annotation to jobs indefinitely like Option 2 does. On the other hand, outside of the cluster downgrade edge case, Option 2 will
ensure consistency within a single job and therefore a more predictable user experience.

After considering these trade-offs, we propose to move forward with Option 1 for simplicity and to avoid being stuck adding this annotation to jobs. In addition, the downside of existing workloads having only a subset of jobs with the new annotation will not cause any serious issues.

## Design Details

The CronJob controller will only need a minor update to the [getJobFromTemplate2](https://github.com/kubernetes/kubernetes/blob/7024beeeeb1f2e4cde93805a137cd7ad92fec466/pkg/controller/cronjob/utils.go#L188) function, to add the job scheduled timestamp as the job annotation `batch.kubernetes.io/cronjob-scheduled-timestamp`. The scheduled timestamp is represented in `RFC3339`.

For the scheduled timestamp's timezone, the initial thought was to use `UTC` as it's used as the primary one for less confusion. However, since the `job` object has a `spec.timeZone`, it was a better to use the same timezone within the same object. If the job `spec.timeZone` is not set or `nil`, the annotation will use the `UTC` timezone as a default.

### Test Plan

- [X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates


##### Unit tests

- `k8s.io/kubernetes/pkg/controller/cronjob`: `05/22/2023` - `96.2%`

##### Integration tests

Unit tests will ensure the new annotation is correctly added to jobs, and integration tests will verify that the annotation is only added to jobs from newly created CronJobs, not existing workloads.

##### e2e tests

E2E tests will not provide any additional coverage that isn't already covered by unit + integration tests, since we are simply adding an annotation, so no e2e tests will be necessary for this change.

### Graduation Criteria

The feature will be released directly in Beta state since there is no benefit in having an alpha release, since we are simply adding a new annotation so there is very little risk (unlike removing an
existing annotation which other things may depend on, for example).

#### Beta

- Feature implemented behind the `JobsScheduledAnnotation` feature gate.
- Unit and integration tests passing.

#### GA

Fix any potentially reported bugs.

### Upgrade / Downgrade Strategy

No changes required to existing cluster to use this feature.

### Version Skew Strategy

N/A. This feature doesn't require coordination between control plane components,
the changes to each controller are self-contained.

## Production Readiness Review Questionnaire


### Feature Enablement and Rollback


###### How can this feature be enabled / disabled in a live cluster?


- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: JobScheduledAnnotation
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. If the feature gate is disabled, the CronJob controller will not add the
scheduled timestamp as an annotation.

###### What happens if we reenable the feature if it was previously rolled back?

The CronJob controller will begin adding the scheduled timestamp as an annotation to jobs created while the feature is enabled, and existing jobs will be unaffected.

###### Are there any tests for feature enablement/disablement?

We plan to add unit tests.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

It will not impact already running workloads.

###### What specific metrics should inform a rollback?

- Users can monitor queue related metrics (e.g., queue depth and work duration) to make sure they aren't growing.
- For CronJobs, users can also monitor `job_creation_skew_duration_seconds`.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

The feature will be tested manually prior to beta launch.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

N/A

###### How can an operator determine if the feature is in use by workloads?

- Check if CronJobs have the annotation `batch.kubernetes.io/cronjob-scheduled-timestamp`.

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason:
- [X] API .metadata
  - Condition name:
  - Other field:
    - `.metadata.annotations['batch.kubernetes.io/cronjob-scheduled-timestamp']`
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- 99% percentile over day for Job syncs is <= 15s for a client-side 50 QPS limit.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - Metric name:
  - `job_creation_skew_duration_seconds`.
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

New job annotation of size 34B plus value of size N where N is the number of digits in the job ordinal.
Worst case for N would be the max number of jobs per node * max number of nodes.
Per the docs on [large clusters](https://kubernetes.io/docs/setup/best-practices/cluster-large/), this would be 110 pods/node * 5000 nodes = 550,000 (6 digits).

Hence, max annotation size would be 34 + 6 = 40B.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

N/A

###### How does this feature react if the API server and/or etcd is unavailable?

N/A

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

- 2023-05-22: KEP published

## Implementation History

## Drawbacks

## Alternatives

- Add label instead of annotation
  - Labels are unnecessary as we need to pass data that won't be used with search or satisfy certain conditions.

- Add a status field
  - The object already has the `CreationTimestamp` field, but it will get overridden with the time the CronJob will start. The point of the new annotation is to pass the original/expected scheduled timestamp information.

## Infrastructure Needed (Optional)
