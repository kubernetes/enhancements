# KEP-6467: Show init container waiting reasons in Pod table output

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

This KEP is provisional. Kubernetes v1.39 is the candidate first release. SIG ownership, approvers, and the release plan need agreement for the printer-only scope. SIG CLI review is requested.

- [ ] (R) Enhancement issue in release milestone, which links to KEP directory
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are agreed
- [ ] (R) Test plan is agreed
- [ ] (R) Graduation criteria are agreed
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] Implementation history is up to date
- [ ] User-facing documentation is ready
- [ ] Supporting discussions and implementation links are recorded

## Summary

Show the existing `PodInitializing` waiting reason alongside the init count in the Pod table used by `kubectl get pods`. A selected init container waiting with this reason displays `Init:N/M PodInitializing`; a running init container keeps the existing count display.

The change is in the server-side Pod printer. Kubelet status generation, Pod API fields, and waiting reasons stay unchanged. No feature gate is proposed.

## Motivation

Today, the Pod printer shows `Init:0/2` both when the first init container has `state.waiting.reason: PodInitializing` and when it has `state.running`. The API already distinguishes these states, but the table hides that distinction.

In [the original report][original-issue], an init container image pull took more than 50 minutes. Showing the existing waiting reason would make it clear that the reported container state is still waiting. It would not identify the exact startup step or show download progress.

### Goals

Show the existing default waiting reason while keeping the init count. Preserve current error displays and init-container selection rules.

### Non-Goals

Change kubelet status generation, startup order, probes, restart rules, or Pod phases. Distinguish image pulling from container creation, or startup preparation from waiting for an earlier init container. Add image-download progress, a new API field, reason, column, or feature gate.

## Proposal

For a Pod with two ordinary init containers and one regular container, the API can already report either of these states for `init-0`:

```yaml
# Waiting, including while an image is being pulled.
status:
  initContainerStatuses:
    - name: init-0
      state:
        waiting:
          reason: PodInitializing
```

```yaml
# Running the init container.
status:
  initContainerStatuses:
    - name: init-0
      state:
        running:
          startedAt: "2026-10-21T00:00:00Z"
```

These are status excerpts. The table would display:

```console
$ kubectl get po demo
NAME   READY   STATUS                       RESTARTS   AGE
demo   0/1     Init:0/2 PodInitializing       0          2m
```

| Reported state | Display |
| --- | --- |
| `init-0` waiting with `PodInitializing` | `Init:0/2 PodInitializing` |
| `init-0` running | `Init:0/2` |
| `init-0` succeeded; `init-1` waiting with `PodInitializing` | `Init:1/2 PodInitializing` |
| `init-1` running | `Init:1/2` |
| Selected init waiting with `ErrImagePull` | `Init:ErrImagePull` |
| Selected init waiting with `ImagePullBackOff` | `Init:ImagePullBackOff` |

Only append `PodInitializing` when that reason is present in `state.waiting`. Missing state or an empty waiting reason retains the current fallback. A count without the suffix is therefore not proof that the container is running; users needing that check should read `state.running`.

### Risks and Mitigations

The longer STATUS value can affect scripts that split output on spaces. It remains one Table cell. Document the change and recommend structured Pod output for state checks.

`PodInitializing` does not identify image pulling, container creation, or dependency waiting. Pod status can lag runtime activity. Keep these limits explicit and retain Events for detailed diagnosis.

## Design Details

Change `printPod` in `pkg/printers/internalversion/printers.go`. In the existing init-container switch, add a case for `state.waiting.reason == "PodInitializing"` that sets the display to `Init:N/M PodInitializing`. Place it after the existing specific waiting-reason branch and before the default count branch. Use the same index and total as the existing count. Keep other branches and their order.

The printer must still skip successful ordinary init containers and restartable init containers with `started: true`. A running restartable init container whose startup probe has not succeeded keeps the count display. Do not infer startup eligibility from the statuses of other containers.

Keep existing error precedence, restart counts, READY counts, and the handling of `Initialized`, `Terminating`, and `Unknown`. Regular-container output is unchanged. Missing init statuses, empty states, and empty reasons must not gain a made-up waiting reason.

`kubectl get pods` normally receives a server-generated `meta.k8s.io/v1` Table. Updating the API server's Pod printer changes that Table output. Upgrading kubectl alone does not enable it. Raw Pod JSON/YAML is unchanged; client printing without a server Table is outside this change.

The existing [ContainerState API][container-state] and [kubelet status conversion][kubelet-status] supply the needed information. No extra API call or runtime query is needed.

### Test Plan

- [x] The authors understand that component owners may require more tests.

##### Prerequisite testing updates

Update Pod printer fixtures that expect only a count for `PodInitializing`. Keep unrelated printer expectations unchanged.

##### Unit tests

Cover waiting and running states for the first and later ordinary init containers, successful completion, specific waiting errors, and failed exits. Include missing statuses, empty states, and empty waiting reasons.

Cover restartable init containers waiting with `PodInitializing` and `started` absent or false, with `started: true` as a skip-rule check. Include a running sidecar with a pending startup probe and sidecar restarts after initialization. Check `Initialized`, deletion, node-unreachable handling, READY, and restart counts. Verify that the new value is one Table cell with all other cells unchanged.

##### Integration tests

Exercise the Pod Table API with controlled Pod status updates from waiting to running to completed. Check the STATUS cell for each state and confirm that raw Pod status keeps its supplied reason. Use a Pod with no active kubelet writer to avoid races. Include an error state and a restartable-init case. No real image pull is needed for these printer assertions.

##### e2e tests

Reuse existing init-container lifecycle coverage to check that running and completed init containers keep their current display. Add the waiting-reason assertion where a test already controls that state. Do not depend on a large public image or a transient image-pull window. No new conformance test is proposed.

### Graduation Criteria

Kubernetes v1.39 is a candidate release, subject to SIG agreement. The metadata keeps the provisional alpha target; the release stages for this ungated printer change need review. No gate-based beta rollout or minimum beta duration is proposed.

#### Alpha

Agree on the output, ownership, and PRR. Pass the printer and Table API tests.

#### Beta

If staged delivery is required, review output compatibility and resolve reported display errors. No feature gate is enabled at this stage.

#### GA

Confirm regression coverage and publish the output change and its limits.

#### Deprecation

None planned.

### Upgrade / Downgrade Strategy

Upgrade the API server to receive the new server-generated Table output. Rolling back to the earlier API server version restores its printer behavior. There is no separate enable/disable switch, Pod rewrite, or stored-data migration.

### Version Skew Strategy

Follow supported kubectl/API-server skew. A supported older kubectl receiving the new Table displays its new STATUS cell. A newer kubectl using an older server still receives the old display. During an API-server rolling upgrade, output can vary with the serving API server. Kubelet version does not select the output format.

## Production Readiness Review Questionnaire

These answers describe the proposal. Implementation tests are pending.

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

It is part of the upgraded API server's Pod printer. There is no feature gate or per-Pod opt-in.

###### Does enabling the feature change any default behavior?

The default Pod Table STATUS cell includes `PodInitializing` for the selected waiting init container. Pod state and execution do not change.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Only by reverting the printer change or using the earlier supported API-server version.

###### What happens if we reenable the feature if it was previously rolled back?

The new printer output returns. No data migration is needed.

###### Are there any tests for feature enablement/disablement?

No gate exists. Compare old and new server output during upgrade and rollback testing.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A printer bug can report an incorrect STATUS or fail Table conversion. Scripts may misread the added text. The printer does not control container execution.

###### What specific metrics should inform a rollback?

Incorrect Table output or Table request failures warrant investigation and possible rollback. No new metric is proposed.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet. Verify old and new Table output before release.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Check the serving API-server version and a Table response for a waiting init container. No workload opt-in is needed.

###### How can someone using this feature know that it is working for their instance?

A selected init with `state.waiting.reason: PodInitializing` displays `Init:N/M PodInitializing`, unless an existing higher-priority Pod status overrides it.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Existing API-server request expectations apply. No separate SLO is proposed.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Use existing API-server request success and latency measurements, together with checks of the Table STATUS cell.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No new metric is proposed.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

It uses the existing Pod API and server-side Table printer. Image-pull progress APIs are not a dependency.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Stored Pod objects are unchanged. Only the STATUS string in affected Table responses grows.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No material change is expected. The printer uses the same container-status traversal.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The change adds one reason check and a short suffix per affected row.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No new node-side work or resources are added.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Pod and Table reads follow the existing API failure paths.

###### What are other known failure modes?

Pod status may lag runtime activity. Empty state or reason keeps the old fallback. Neither the suffix nor its absence identifies the exact startup step or proves progress.

###### What steps should be taken if SLOs are not being met to determine the problem?

Compare the Table cell with raw Pod status and check the serving API-server version. Use Events and kubelet/runtime logs for startup diagnosis.

## Implementation History

## Drawbacks

STATUS becomes longer and contains a space. It still shows only one selected init status. Image pulling and other waiting steps remain grouped together, and a count alone does not prove execution.

## Alternatives

Change kubelet's default reason to `ContainerCreating` for an init container eligible to start. This could distinguish startup preparation from dependency waiting in the API, but requires new eligibility logic and reason compatibility work. The existing waiting/running fields are enough for this narrower display change. The earlier [implementation PR][implementation-pr] targets kubelet status generation; it does not implement this printer-only proposal and is not a dependency.

Keep the current printer. This preserves output but continues to hide an existing state distinction.

Add a `Pulling` reason or progress display. This needs separate information and design. [KEP-5334: Image Pull Progress][image-pull-progress] addresses progress APIs and is not required here.

## Infrastructure Needed (Optional)

No new production infrastructure is needed.

[original-issue]: https://github.com/kubernetes/kubernetes/issues/141686
[implementation-pr]: https://github.com/kubernetes/kubernetes/pull/141693
[container-state]: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/api/core/v1/types.go
[kubelet-status]: https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/kubelet_pods.go
[image-pull-progress]: https://github.com/kubernetes/enhancements/issues/5334
