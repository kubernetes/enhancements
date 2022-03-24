# KEP-1972: Kubelet Exec Probe Timeouts

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kubelet today does not respect exec probe timeouts. This is considered a bug we should fix since
the timeout value is supported in the Container Probe API. Because exec probe timeouts
were never respected by kubelet, a new feature gate `ExecProbeTimeout` will be introduced.
With this feature, nodes can be configured to preserve the current behavior while the proper
timeouts are enabled for exec probes.

## Motivation

Kubelet not respecting the probe timeout is a bug and should be fixed.

### Goals

* treat exec probe timeouts as probe failures in kubelet

### Non-Goals

* ensuring exec processes that timed out have been killed by kubelet.
* introducing CRI errors for handling scenarios such as time outs.

## Proposal

### Risks and Mitigations

* existing workloads on Kubernetes that relied on this bug may unexpectedly see their probes timeout

## Design Details

Changes to kubelet:
* Ensure kubelet handles timeout errors and registers them as failing probes.
* Add feature gate `ExecProbeTimeout` that is GA and on by default.
* If the feature gate `ExecProbeTimeout` is disabled and an exec probe timeout is reached, add warning event to inform users that exec probes are timing out.
* Introduce the [probe duration metric](https://github.com/kubernetes/kubernetes/issues/101035)
  * metric dimension cardinality must be reviewed and approved by SIG Instrumentation
* Re-enable existing exec liveness probe e2e test.
* Add new exec readiness probe e2e test.

### Test Plan

E2E tests:
* re-enable [existing exec liveness probe e2e test](https://github.com/kubernetes/kubernetes/blob/ea1458550077bdf3b26ac34551a3591d280fe1f5/test/e2e/common/container_probe.go#L210-L227) that is currently being skipped
* add new exec readiness probe e2e test.
* exec probe tests are promotes to Conformance ([#97619](https://github.com/kubernetes/kubernetes/pull/97619)).

### Graduation Criteria

This is a bug fix so the feature gate will be GA and on by default from the start.

Documentation on the migration steps must be provided at kubernetes
documentation site offering tips on detecting and updating affected workloads.

The feature flag should be kept available till we get a sufficient evidence of people not being
affected by this bug fix - either directly (adjusting the timeouts in pod definition), or
indirectly, when the timeout is not specified in some third party templates and products
that cannot be easily fixed by end user.

Tentative timeline is to lock the feature flag to `true` in 1.25.

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

N/A

## Implementation History

* 2020-09-08 - the KEP was merged as implementable for v1.20
* 2020-12-08 - Timeout is respected in [Kubernetes 1.20: The Raddest Release](https://kubernetes.io/blog/2020/12/08/kubernetes-1-20-release-announcement/),
  and can be disabled with the feature flag


## Drawbacks

* Existing workloads may depend on the fact that exec probe timeouts were never respected. Introducing
the timeout now may result in unexpected behavior for some workloads.

## Alternatives

Some alternatives that were considered:

1. Increasing the default timeout for exec probes
2. Continuing to ignore the exec probe timeout

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ExecProbeTimeouts`
  - Components depending on the feature gate: kubelet

###### Does enabling the feature change any default behavior?

Yes, all workloads that were not accounting for the timeout affect the probe
behavior will experience the problem.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, by resetting the feature gate back.

###### What happens if we reenable the feature if it was previously rolled back?

Behavior will restore back immediately.

###### Are there any tests for feature enablement/disablement?

N/A, trivial

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollout and rollback are straightforward and are not expected to fail.

###### What specific metrics should inform a rollback?

Pods entering crashloopbackoff because of exec timeout failure.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A, trivial

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

The only mechanism currently implemented is warning logs in kubelet.
The KEP was updated to introduce the warning events for the cases when timeout
was exceeded. With these events, operator may ensure that no workloads are
affected by this bug currently by analyzing events.

###### How can an operator determine if the feature is in use by workloads?

Before migration, analyze events indicating that the timeout was exceeded by exec probe.
There is no way to determine if exceed timeout failure of exec probes were intentional
or not once the feature gate was enabled.

###### How can someone using this feature know that it is working for their instance?

No, there is no way to determine if exceed timeout failure of exec probes were intentional
or not once the feature gate was enabled.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

SLO of the feature: exec probes must fail when timeout is exceeded. This can be
checked by reviewing that Probe duration metric not exceeding significantly
the timeout value.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `probe_duration_seconds`

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

The metric [probe duration metric](https://github.com/kubernetes/kubernetes/issues/101035)
was not implemented yet.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

### Troubleshooting

Kubelet.log may be used for all the probes behavior troubleshooting.

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

None

###### What steps should be taken if SLOs are not being met to determine the problem?

None. It is a core functionality of kubelet
