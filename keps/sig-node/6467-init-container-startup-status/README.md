# KEP-6467: Improve Init Container Startup Status Visibility

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
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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

This KEP is provisional. SIG sponsorship, approvers, the rollout plan, and a
release target are not yet agreed.

- [ ] (R) Enhancement issue in release milestone, which links to KEP directory
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are agreed
- [ ] (R) Test plan is agreed
  - [ ] e2e coverage for the changed status behavior
  - [ ] Conformance requirements reviewed with SIG Testing
- [ ] (R) Graduation criteria are agreed
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] Implementation history is up to date
- [ ] User-facing documentation is ready
- [ ] Supporting discussions and implementation links are recorded

## Summary

**Users should be able to tell whether an init container is being prepared to
start, is running, or is waiting for earlier init containers, using
the same status semantics as regular containers.**

Use `ContainerCreating` for an init container being prepared to start and retain
`PodInitializing` for containers waiting for earlier init containers. A running
init container keeps its existing `state.running` status and `Init:N/M` display.

## Motivation

Today, `kubectl get pods` can show `Init:0/2` both while the first init container
is being prepared and while it is running. Its waiting reason can remain
`PodInitializing`, so users must inspect Events to understand the delay.

In [the original report][original-issue], an init container image pull took more
than 50 minutes without a useful distinction in Pod status.

### Goals

Distinguish startup preparation, execution, and dependency waiting in
`status.initContainerStatuses`, using existing container states and reasons.
Expose that distinction in the Pod table display.

### Non-Goals

Add a `Pulling` reason, report download percentages, or change container execution
order, readiness, restart rules, or Pod phases. Image-pull progress APIs are
outside this KEP.

## Proposal

When an init container has no more specific state or error, kubelet uses
`PodInitializing` if an earlier init container still blocks it under the existing
startup rules. Otherwise, it uses `ContainerCreating` as the default waiting
reason.

For a Pod with one regular container and two ordinary init containers, the
expected status excerpt while preparing `init-0` is:

```yaml
status:
  initContainerStatuses:
    - name: init-0
      state:
        waiting:
          reason: ContainerCreating
    - name: init-1
      state:
        waiting:
          reason: PodInitializing
```

Using the existing server-side Pod printer, the output is:

```console
$ kubectl get po demo
NAME   READY   STATUS                   RESTARTS   AGE
demo   0/1     Init:ContainerCreating    0          2m
```

The following examples assume no startup errors or restarts:

| Stage | Expected display |
| --- | --- |
| Preparing `init-0` | `Init:ContainerCreating` |
| Running `init-0`; `init-1` still waiting | `Init:0/2` |
| `init-0` succeeded; preparing `init-1` | `Init:ContainerCreating` |
| Running `init-1` | `Init:1/2` |

### Risks and Mitigations

Scripts, alerts, and controllers matching `state.waiting.reason` or Pod table
output may need updates. Use `state.running` and `state.terminated` to identify
execution states. Checks for these waiting defaults must accept both values
during rollout. Test the change with the gate enabled and disabled.

`ContainerCreating` is a general startup waiting reason. It can include image
pulling; it does not prove that a CRI `CreateContainer` call is in progress.
Users still need Events for the exact operation. Status can lag runtime activity,
as it does for regular containers.

## Design Details

Kubelet changes only the default waiting reason. Runtime states and specific
errors, including `ErrImagePull`, `ImagePullBackOff`, and `CrashLoopBackOff`,
keep their existing precedence. Regular containers retain their existing
status behavior.

Startup eligibility follows kubelet's existing initialization progress.
An ordinary init container must exit successfully before the next starts.
A restartable init container must first satisfy the existing `started` rules:
it is running and any configured startup probe has succeeded. A later sidecar
restart alone must preserve initialization progress already made in the same sandbox.
Sandbox recreation and init resets follow existing kubelet rules; stale status
must not bypass them.

No new Pod fields or CRI interfaces are needed. The existing Pod printer already
displays `Init:ContainerCreating` for this waiting reason. No new table column
or progress format is proposed.

The proposed kubelet feature gate is `InitContainerStartupStatus`, disabled by
default in alpha. Its name and rollout plan need SIG Node agreement. This is a
proposed gate, not an existing Kubernetes option.

### Test Plan

##### Prerequisite testing updates

Update assertions that assume every waiting init container uses
`PodInitializing`. Keep explicit coverage for the gate-disabled behavior.

##### Unit tests

Cover status generation and Pod printing for single and multiple init
containers, preparation-to-running transitions, successful completion, and
startup errors. Test restartable init containers with a pending startup probe,
restarts, and sandbox recreation. Ensure blocked containers are not reported as
eligible to start.

##### Integration tests

Use the node/e2e coverage below to check kubelet status publication through the
API server. No new API endpoint is added; separate API-server integration tests
are not proposed.

##### e2e tests

Use a controlled registry response to hold an uncached image pull long enough
to observe the waiting reason. Verify API status and server-side Pod table
output before releasing the pull, while the init container runs, and after it
finishes. Include multiple init containers and a restartable init container
whose startup probe blocks the next container. Check error reasons and both
feature-gate settings. Use bounded waits rather than a large public image.

### Graduation Criteria

These are proposed criteria; no release is targeted yet.

#### Alpha

Agree on the status semantics, gate, and approvers. Complete PRR and implement
the gate with unit and node/e2e coverage.

#### Beta

Enable the gate by default after alpha feedback is addressed. Complete
documentation, compatibility checks, and upgrade/downgrade testing. Resolve
known correctness, observability, and test issues. Verify that the status change
does not affect startup or restart behavior.

#### GA

After at least two releases in beta, review user feedback and confirm sustained
use without outstanding serious regressions. Agree on any conformance coverage
before graduation.

#### Deprecation

Not applicable. This KEP does not deprecate an API or feature.

### Upgrade / Downgrade Strategy

With the gate disabled, kubelet reports the existing waiting reasons. Enabling
it changes default waiting reasons on subsequent status updates. Disabling it
restores the previous behavior on subsequent updates without restarting Pods.
Downgrading kubelet restores the older behavior. No stored-data migration is
needed.

### Version Skew Strategy

Existing API servers can store these reasons, and existing Pod printers can
display `Init:ContainerCreating`. Older kubelets continue reporting the older
defaults. A mixed cluster can therefore show different startup reasons across
nodes; this does not change container execution.

## Production Readiness Review Questionnaire

These answers describe the proposal. Implementation and rollout tests are still
pending.

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

Through the proposed kubelet gate `InitContainerStartupStatus`. Changing kubelet
gate configuration may require restarting kubelet.

###### Does enabling the feature change any default behavior?

It changes default waiting reasons for eligible init containers and their Pod
table display. Container execution is unchanged.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Subsequent status updates use the previous waiting reasons.

###### What happens if we reenable the feature if it was previously rolled back?

Subsequent status updates use the proposed reasons again. No data migration is
needed.

###### Are there any tests for feature enablement/disablement?

Both settings are required in the proposed test plan; tests are not complete.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Incorrect dependency detection could misreport waiting containers. The change
must not feed back into startup decisions or alter running containers. Test
restart and sandbox-recreation paths before rollout.

###### What specific metrics should inform a rollback?

Watch for regressions in Pod startup latency, kubelet sync errors, and container
restarts. Incorrect status reports also warrant rollback. No new metric is
proposed.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet. This is required before beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Check kubelet gate configuration and waiting init-container reasons. The change
applies to all Pods on an enabled node, without per-Pod opt-in.

###### How can someone using this feature know that it is working for their instance?

An eligible init container being prepared to start reports `ContainerCreating`;
a later blocked container reports `PodInitializing`.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Existing Pod startup and status-publication expectations apply. No separate SLO
is proposed.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Use existing kubelet health and Pod startup measurements, together with the
API status checks described above.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No new metric is proposed for this status-only change.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No additional service is needed. Existing kubelet, runtime, and API-server
status paths are used. Image-pull progress APIs are not a dependency.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No additional calls or status-publication loop is proposed.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Only the length of an existing reason string changes. Object count is unchanged.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No material increase is expected. Eligibility checks should avoid repeated
container-status scans; verify this during implementation review.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No material increase is expected. Any lookup data should be bounded by the
number of containers in the Pod.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No new processes, sockets, files, or persistent caches are proposed.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Status updates can be delayed through the existing failure path. Container
startup continues under existing kubelet behavior.

###### What are other known failure modes?

Status can lag runtime activity. Incorrect dependency detection can label a
blocked init container `ContainerCreating`. Neither reason gives image-download
progress or proves that startup is making progress.

###### What steps should be taken if SLOs are not being met to determine the problem?

Compare Pod status with Events and kubelet/runtime logs. Check the gate setting
and reproduce with it disabled if the status change is suspected.

## Implementation History

- 2026-10-13: Initial provisional proposal, based on [issue #141686][original-issue]
  and [implementation PR #141693][implementation-pr]. That PR predates this KEP
  and must be aligned with the agreed design before it can proceed.

## Drawbacks

The Pod table still shows one status, so `Init:ContainerCreating` does not also
show the completion count. Users can inspect individual container statuses for
details. Image pulling and other preparation work remain grouped together.

## Alternatives

Keeping the current reasons leaves preparation and execution indistinguishable
in the default table output. Using `ContainerCreating` for every waiting init
container would mislabel containers blocked by earlier ones. A new `Pulling`
reason would require a separate design for detecting and publishing that state.

[KEP-5334: Image Pull Progress][image-pull-progress] proposes streaming progress
APIs. It addresses a related need but does not define this waiting-reason change.
This proposal can proceed without those APIs. SIG Node and SIG CLI can decide
whether to keep it separate or add this scope to KEP-5334.

## Infrastructure Needed (Optional)

No production infrastructure is needed. The e2e test needs a controllable test
registry.

[original-issue]: https://github.com/kubernetes/kubernetes/issues/141686
[implementation-pr]: https://github.com/kubernetes/kubernetes/pull/141693
[image-pull-progress]: https://github.com/kubernetes/enhancements/issues/5334
