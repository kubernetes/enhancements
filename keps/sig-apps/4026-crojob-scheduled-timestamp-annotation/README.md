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
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Currently, there is no supported way to get the original/expected initial scheduled timestamp for the job created from a cronjob. This KEP proposes to set the original scheduled time as an annotation in the job metadata.

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

CronJobs are always working with the assumption that the changes apply only to newly created jobs after the change. Therefore, the change will be to inject the annotation for newly created Jobs from CronJobs for when the feature is on. This will nicely play with downgrade and doesn't introduce unnecessary complexity.

## Design Details

The CronJob controller will only need a minor update to the [getJobFromTemplate2](https://github.com/kubernetes/kubernetes/blob/7024beeeeb1f2e4cde93805a137cd7ad92fec466/pkg/controller/cronjob/utils.go#L188) function, to add the job scheduled timestamp as the job annotation `batch.kubernetes.io/cronjob-scheduled-timestamp`. The scheduled timestamp is represented in `RFC3339`.

For the scheduled timestamp's timezone, the initial thought was to use `UTC` as it's used as the primary one for less confusion. However, since the `job` object has a `spec.timeZone`, it was a better to use the same timezone within the same object. If the job `spec.timeZone` is not set or `nil`, the annotation will use the `UTC` timezone as a default.

### Test Plan

- [X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates


##### Unit tests

- `k8s.io/kubernetes/pkg/controller/cronjob`: `09/24/2023` - `71.2%`

##### Integration tests

- No integration tests are planned for this feature.

##### e2e tests

- [CronJob should set the cronjob-scheduled-timestamp annotation](https://github.com/kubernetes/kubernetes/blob/4aeaf1e99e82da8334c0d6dddd848a194cd44b4f/test/e2e/apps/cronjob.go#L264-L287): [test coverage](https://storage.googleapis.com/k8s-triage/index.html?test=.*CronJob%20should%20set%20the%20cronjob-scheduled-timestamp%20annotation.*)

### Graduation Criteria

The feature will be released directly in Beta state since there is no benefit in having an alpha release, since we are simply adding a new annotation so there is very little risk.

#### Beta

- Feature implemented behind the `CronJobsScheduledAnnotation` feature gate.
- Unit and e2e tests passing.

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
  - Feature gate name: `CronJobCreationAnnotation`
  - Components depending on the feature gate: `kube-controller-manager`
- [ ] Other
  - Describe the mechanism: N/A.
  - Will enabling / disabling the feature require downtime of the control
    plane? No
  - Will enabling / disabling the feature require downtime or re-provisioning of a node? No

###### Does enabling the feature change any default behavior?

The jobs newly created by cronjob controller will contain a new annotation `CronJobsScheduledAnnotation`.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. If the feature gate is disabled, the CronJob controller will not add the
scheduled timestamp as an annotation.

###### What happens if we reenable the feature if it was previously rolled back?

The CronJob controller will begin adding the scheduled timestamp as an annotation to jobs created while the feature is enabled, and existing jobs will be unaffected.

###### Are there any tests for feature enablement/disablement?

Given the feature results in adding an annotation only to newly created objects, those tests won't really be different from the actual feature tests.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

This change will not impact the rollout or rollback fail. It also will not impact the already running workloads.

###### What specific metrics should inform a rollback?

- Users can monitor CronJobs metrics `job_creation_skew_duration_seconds` and `cronjob_controller_rate_limiter_use`, `cronjob_job_creation_skew`.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

The following manual upgrade->downgrade->upgrade scenario was performed:

1. Create a v1.27 cluster where the feature is not available, yet.
2. Create a CronJob and wait for jobs to be created. Verify the newly created job
   does NOT have the `batch.kubernetes.io/cronjob-scheduled-timestamp` annotation.
3. Upgrade cluster to v1.28, where the feature was available as beta, iow.
   on by default. Verify the newly created job from a CronJob created in 2nd step
   has the `batch.kubernetes.io/cronjob-scheduled-timestamp` annotation with
   planned time, when a job was to be created.
4. Downgrade cluster to v1.27, where the feature was NOT available. Verify the
   newly created job from a CronJob created in 2nd step does NOT have the
   `batch.kubernetes.io/cronjob-scheduled-timestamp` annotation.

During the tests no problems were identified with cronjobs or jobs.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements


###### How can an operator determine if the feature is in use by workloads?

Randomly checking the CronJobs annotation `batch.kubernetes.io/cronjob-scheduled-timestamp` is sufficient. For monitoring purposes, we can rely on pre-existing metrics which monitor both the cronjob queue and the job creation skew, which should provide sufficient signal if the controller is working as expected. For small clusters, checking  the annotation will determine the feature is used.

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason:
- [X] API .metadata
  - Condition name:
  - Other field:
    - `.metadata.annotations['batch.kubernetes.io/cronjob-scheduled-timestamp']`

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- 99% percentile over day for Job syncs is <= 15s for a client-side 50 QPS limit.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - Metric name: cronjob_job_creation_skew
  - Components exposing the metric: kube-controller-manager
  - Metric name: job_creation_skew_duration_seconds
  - Components exposing the metric: kube-controller-manager

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, each job created by a cronjob-controller will have an additional annotation containing `RFC3339` timestamp, which together with annotation name results in ~70B per job object.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No change comparing to existing failure modes.

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

The new annotation shouldn't cause any unforeseen issues with the cronjob controller.
In the event of issues with meeting SLOs, cluster admins are advised to consult
[troubleshooting overview document](https://kubernetes.io/docs/tasks/debug/).

## Implementation History

- 2023-06-06: KEP published
- 2024-09-24: KEP updated for stable promotion

## Drawbacks

## Alternatives

- Add label instead of annotation
  - Labels are unnecessary as we need to pass data that won't be used with search or satisfy certain conditions.

- Add a status field
  - The object already has the `CreationTimestamp` field, but it will get overridden with the time the CronJob will start. The point of the new annotation is to pass the original/expected scheduled timestamp information.

## Infrastructure Needed (Optional)

N/A
