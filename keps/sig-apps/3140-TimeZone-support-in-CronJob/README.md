# KEP-3140: TimeZone support in CronJob

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [CronJob API](#cronjob-api)
  - [CronJob controller](#cronjob-controller)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [x] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [x] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

CronJob creates Jobs based on the schedule specified by the author, but the Time
Zone used during the creation depends on where kube-controller-manager is running.
This proposal aims to extend CronJob resource with the ability for a user to
define the TimeZone when a Job should be created.

## Motivation

Not long after the [introduction of CronJob in kubernetes](https://github.com/kubernetes/kubernetes/pull/11980)
a [request was raised to support setting time zones](https://github.com/kubernetes/kubernetes/issues/47202).
The initial [response from SIG-Apps and SIG-Architecture](https://github.com/kubernetes/kubernetes/issues/47202#issuecomment-360820586)
at the time was that introducing such functionality would require cluster operator
to manually include TimeZone database since golang did not have one.
Starting from [golang 1.15](https://go.dev/doc/go1.15) `time/tzdata` package can
be embedded in the binary thus removing the requirement for external database.
At that time, the majority of the focus was towards [moving CronJob to GA](https://github.com/kubernetes/enhancements/issues/19),
so the effort to support TimeZone was again delayed. Now that we have CronJob
fully GA-ed it's time to satisfy the original request.

### Goals

- Add the field `.spec.timeZone` which allows specifying a valid TimeZone name

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

## Proposal

Add the field `.spec.timeZone` to the CronJob resource. The cronjob controller
will take the field into account when scheduling the next Job run. In case the
field is not specified or is empty, the controller will maintain the current
behavior, which is to rely on the time zone of the kube-controller-manager
process.

### Notes/Constraints/Caveats (Optional)

The current mechanism, which will still be the default behavior, heavily relies
on the time zone of the kube-controller-manager process, which is hard for a
regular user to figure out. Exposing an explicit field for setting a time zone
will allow CronJob authors to simplify the user experience when creating CronJobs.

### Risks and Mitigations

- Outdated time zone in the golang might lead to wrong schedule times

This problem can be mitigated with a fresh build of kube-controller-manager with
updated golang version, but it heavily relies on the go community to keep the
time zone database up-to-date.

- Malicious user can create multiple CronJobs with different time zone which can
actually trigger Jobs at the exact same time

Cluster administrators should enforce quota to ensure not that many Jobs and
CronJobs can be created per user.


## Design Details

### CronJob API

The `.spec` for a CronJob is expanded with a new `timeZone` field which allows
specifying the name of the time zone to be used, the list of valid time zones
can be found [in tz database](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones).
Missing or empty value of the field indicates the current behavior, which relies
on the time zone of the kube-controller-manager process.

In the API code, that looks like:

```golang

type CronJobSpec struct {

    // The schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.
    Schedule string

    // Time zone for the above schedule
    TimeZone *string

}
```

The value provided in `TimeZone` field will be validated against the embedded golang
timezone database, which will result in the `kube-apiserver` and `kube-controller-manager`
binaries growing by roughly extra 500kB.

### CronJob controller

CronJob controller will use non-nil, non-empty value from `TimeZone` field when
parsing the schedule and during scheduling the next run time. Additionally, the
time zone will be reflected in the `.status.lastSuccessfulTime` and `.status.lastScheduleTime`.
In all other cases the controller will maintain the current behavior.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

1. Add tests ensuring that case insensitive location loading is properly handled.
   See [beta requirements](#beta) for more details.
2. Add at least integration and optionally e2e covering TimeZone usage.

##### Unit tests

- `k8s.io/kubernetes/pkg/apis/batch/validation`: `2022-06-09` - `94.4%`
- `k8s.io/kubernetes/pkg/controller/cronjob`: `2022-06-09` - `50.8%`
- `k8s.io/kubernetes/pkg/registry/batch/cronjob`: `2022-06-09` - `61.8%`

##### Integration tests

None.

##### e2e tests

None.

### Graduation Criteria

#### Alpha

- Functionality implemented behind feature gate:
  - TimeZone field added to API (kube-apiserver)
  - CronJob controller reacting to new field (kube-controller-manager)

#### Beta

- Solve issue with case insensitive location loading:
  - Test skipped on MacOS (https://github.com/kubernetes/kubernetes/pull/109218)
  - Golang issue (https://github.com/golang/go/issues/21512)

#### GA

TBD

<!---
#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

### Upgrade / Downgrade Strategy

- Upgrades

When upgrading from a release without this feature to a release with `TimeZone`
users should not notice any change in behavior. The default value for `TimeZone`
is nil, which is equal to current behavior for backwards compatibility.

- Downgrades

When downgrading from a release with this feature, to a release without `TimeZone`
there are a few cases:
  1. If TimeZone feature gate was enabled and user specified a TimeZone, the newly
  created Jobs after downgrade will return to previous behavior, as if no TimeZone
  was ever specified.
  2. If TimeZone feature gate was enabled and user did not specify TimeZone, there
  should be no change in behavior.
  3. If TimeZone feature gate was not enabled there should be no change in behavior.

Irrespectively of the option, cluster administrator should monitor `cronjob_job_creation_skew`
which reports the skew between schedule and actual job creation.

### Version Skew Strategy

This feature has no node runtime implications.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: CronJobTimeZone
  - Components depending on the feature gate: kube-apiserver, kube-controller-manager

###### Does enabling the feature change any default behavior?

No, the default behavior is maintained independently of the feature gate.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the feature can be disabled. The outcome of disabling will depend if the
new field was set by the user:

1. If there was no value set in `TimeZone` field there will be no change in behavior.
2. If the user set a valid `TimeZone`, the newly created Jobs will be triggered
as if the field was never set.

###### What happens if we reenable the feature if it was previously rolled back?

The controller will start reading the new field.

###### Are there any tests for feature enablement/disablement?

Yes, both units and integration tests for enablement, disablement and transitions.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

An upgrade flow can be vulnerable to the enable, disable, enable if you have
a lease that is acquired by a new kube-controller-manager, then an old
kube-controller-manager, then a new kube-controller-manager.

###### What specific metrics should inform a rollback?

Increased `cronjob_job_creation_skew` which tracks how much a job creation
is delayed compared to requested time slot.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Upgrade->downgrade->upgrade path was manually tested. No issues were found during tests.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

There's no explicit metric for TimeZone but operator should monitor `cronjob_job_creation_skew`,
ensuring the job creation skew is not increasing.

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event Reason: `UnknownTimeZone` when specified TimeZone is not correct

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

99th percentile of cron_job_creation_skew <= 5 seconds per cluster-day.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `cronjob_controller_rate_limiter_use`
  - Components exposing the metric: `kube-controller-manager`
  - Metric name: `cron_job_creation_skew`
  - Components exposing the metric: `kube-controller-manager`


###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

None.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new API calls are expected.

###### Will enabling / using this feature result in introducing new API types?

Yes, `.spec.timeZone` will be present in CronJob API.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No new calls to cloud provider are expected.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes.

- API type(s): CronJob
- Estimated increase in size: new field in CronJob spec up to 50 bytes.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No increase it existing SLIs/SLOs is expected.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Additional CPU and memory increase in the kube-controller-manager is negligible
since the current schedule parsing is already covering time zone specification.
We're not using it, yet.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

- [Incorrect TimeZone]
  - Detection: `UnknownTimeZone` events being reported for a CronJob.
  - Mitigations: Fix the TimeZone or suspend a CronJob.
  - Diagnostics: Logs containing `TimeZone` phrase.
  - Testing: A set of unit tests is ensuring that invalid TimeZone is properly
    handled both in the apiserver and in the controller itself, reporting to
    user the problem.


###### What steps should be taken if SLOs are not being met to determine the problem?

If possible increase the log level for kube-controller-manager and check cronjob's
controller logs looking for warnings and errors which might point where the problem
lies.

## Implementation History

- *2022-01-14* - Initial KEP draft
- *2022-06-09* - Updated KEP for beta promotion.

## Drawbacks

Using TimeZone might be simpler for users working with a cluster in different
TimeZones, but adds additional complexity to the code and to the operator
who will need to re-calculate when an actual CronJob will be creating a Job
when `.spec.timeZone` is set.

## Alternatives

Another approach was to specify time zone as an offset to UTC, but using the
name instead seems more user friendly.

## Infrastructure Needed (Optional)

None.
