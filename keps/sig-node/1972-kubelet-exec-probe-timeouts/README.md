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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
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
* If the feature gate `ExecProbeTimeout` is disabled and an exec probe timeout is reached, add warning logs to inform users that exec probes are timing out.
* Re-enable existing exec liveness probe e2e test.
* Add new exec readiness probe e2e test.

### Test Plan

E2E tests:
* re-enable [existing exec liveness probe e2e test](https://github.com/kubernetes/kubernetes/blob/ea1458550077bdf3b26ac34551a3591d280fe1f5/test/e2e/common/container_probe.go#L210-L227) that is currently being skipped
* add new exec readiness probe e2e test.
* exec probe tests are promotes to Conformance ([#97619](https://github.com/kubernetes/kubernetes/pull/97619)).

### Graduation Criteria

This is a bug fix so the feature gate will be GA and on by default from the start.

The feature flag should be kept available till we get a sufficient evidence of people not being
affected by this bug fix - either directly (adjusting the timeouts in pod definition), or
indirectly, when the timeout is not specified in some third party templates and products
that cannot be easily fixed by end user.

Tentative timeline is to lock the feature flag to `true` in 1.22.

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
