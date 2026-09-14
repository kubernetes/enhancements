<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-4438: Restarting sidecar containers during Pod termination

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [Alpha limitations](#alpha-limitations)
    - [Pod termination and In-Place Pod Restart interaction](#pod-termination-and-in-place-pod-restart-interaction)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Scope and worker transitions](#scope-and-worker-transitions)
  - [Observations and scheduling](#observations-and-scheduling)
  - [Desired actions and ordering](#desired-actions-and-ordering)
  - [Deadlines and cancellation](#deadlines-and-cancellation)
  - [Recovery](#recovery)
  - [Completion, resources and status](#completion-resources-and-status)
  - [Lifecycle invariants and review requirements](#lifecycle-invariants-and-review-requirements)
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

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->
Sidecar containers should be restarted if they exit prematurely when a Pod is terminated to ensure that they
are running while the containers that should terminate prior to its termination are still running.
This feature was originally planned for beta release in the initial KEP, but after further analysis, 
we decided to postpone and introduce a separate feature gate for it.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->
The reason for this KEP is that restarting sidecar containers during Pod termination requires
fundamental changes to the Pod lifecycle, and such changes cannot be introduced in beta
(enabled by default) without risking disruption to users.

On the other hand, the code for sidecar containers is already well tested, and we are confident
that many use cases will benefit from them even without the restart during Pod termination.

For this reason, we are introducing a separate feature gate for the sidecar containers KEP to decouple 
the two features and allow users to use sidecar containers without the refactoring required for
the restart during Pod termination.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->
The following behaviors should be maintained during pod termination:
- sidecar containers restarting
- liveness, readiness and startup probing
- container lifecycle hooks running
- service account token rotation
- secret and configmap volume updates

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->
The proposal is to introduce a new feature gate for the sidecar containers KEP to decouple the sidecar feature
from the restart during Pod termination feature and allow users to use sidecar containers without the refactoring
required for the restart during Pod termination.

Please refer to the original KEP for the details of the sidecar containers feature:
https://git.k8s.io/enhancements/keps/sig-node/753-sidecar-containers

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

#### Story 2

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

#### Alpha limitations

The proposed Alpha implementation targets v1.38 behind
`SidecarsRestartableDuringPodTermination`, disabled by default. It restarts a
previously started sidecar while application containers or later sidecars still
need it, and stops the current instance at its ordered turn. The lifecycle design
below is proposed for review alongside [kubernetes/kubernetes#140133]; that PR is
not a merged implementation or evidence of design approval.

- **Probes:** liveness and startup probes are stopped when termination begins.
  Probe workers are not reattached to replacement instances. Restarts are driven
  by observed exits, not probe failures. Probe support remains Beta work.
- **Recovery:** API deletion deadlines survive kubelet restart through
  `DeletionTimestamp`. Local eviction and static-pod termination requests have no
  durable termination checkpoint; their original intent and deadline are not
  guaranteed to survive kubelet restart. See [Recovery](#recovery).
- **Hooks:** `postStart` uses the normal start path. `preStop` is scheduled for
  each observed running instance, including replacements, while grace remains.
  Hook completion is not persisted, so a hook may execute again after kubelet
  restart. Hooks must tolerate replay.
- **Deadline expiry:** no restart is attempted with at most one second remaining.
  At expiry, ordering and unfinished hooks no longer delay forced stops. The
  proposed zero-grace behavior differs from the legacy minimum stop grace and
  requires explicit review; see [Deadlines and cancellation](#deadlines-and-cancellation).
- **Availability:** a deadline bounds the requested grace, not the time at which
  an unavailable runtime or hung node physically stops a process. Failed runtime
  observations keep termination pending and retain resources.
- **Configuration:** pull secrets and image volumes are resolved through the
  normal start path. This does not add service-account token rotation or secret
  and configmap volume refresh during termination.
- **Observability:** pod status is refreshed on each successful observation,
  including replacement IDs and restart counts. API publication remains
  asynchronous. The restart counter resets on kubelet restart and counts
  successful start-path completions, not starts whose CRI response was lost.

#### Pod termination and In-Place Pod Restart interaction

`ShouldAllContainersRestart` returns false for an API pod with a
`DeletionTimestamp`. Once the worker enters `TerminatingPod`, it no longer calls
`SyncPod`, including for local termination requests. Only the termination
reconciler may restart eligible sidecars; it never starts application containers
or triggers `RestartAllContainers`.

### Risks and Mitigations

Changing `SyncTerminatingPod` from a one-shot operation to reconciliation changes
when other kubelet subsystems may release resources. A successful RPC, an expired
deadline, or a missing cache entry must not independently authorize cleanup.
Tests exercise the worker completion signal, runtime observations, final status,
and DRA unprepare boundary.

Lost CRI responses can leave a created or running replacement behind. A fresh
runtime observation precedes each reconciliation, and created replacements are
removed before retrying. Outstanding stop requests are deduplicated by container
ID. Replays rely on CRI's idempotent `StopContainer` and `RemoveContainer`
contracts. Tests with the fake CRI cover failure before and after side effects;
real-runtime node tests remain necessary to validate cancellation and ordering.

Restarting a sidecar continues using pod resources during termination, including
images and credentials. The feature does not extend their validity or the pod's
grace budget. Alpha remains opt-in. Approval of the lifecycle decisions and
passing node tests are release requirements, separate from unit-test success.

## Design Details

### Scope and worker transitions

The kubelet enables reconciliation only for a pod with restartable init
containers when `SidecarsRestartableDuringPodTermination` is enabled. Ordinary
pods, gate-disabled pods, sandbox replacement in `SyncPod`, and runtime-only
orphan cleanup retain their existing kill paths. No API fields or CRI methods
are added. The generic one-shot `KillPod` path does not contain a restart watcher.

The pod worker remains the sole owner of lifecycle transitions for a pod UID:

| Current state | Result | Next action |
| --- | --- | --- |
| `SyncPod` | Termination requested or normal execution finished | Enter `TerminatingPod`; stop normal setup |
| `TerminatingPod` | `complete=false, err=nil` | Keep resources and kill waiters; schedule another reconciliation |
| `TerminatingPod` | Error | Keep resources and kill waiters; retry with backoff |
| `TerminatingPod` | `complete=true, err=nil` | Notify kill waiters and allow `SyncTerminatedPod` cleanup |
| `TerminatedPod` | Cleanup succeeds | Finish the worker under the existing cleanup contract |

The completion boolean is explicit: returning nil error does not mean the pod
has stopped. An expired deadline also does not imply completion.

### Observations and scheduling

Each invocation obtains the runtime pod and its container status directly, with
one two-second context budget for the observation. The terminating worker skips
`podCache.GetNewerThan`: that wait has no timeout and could otherwise prevent the
worker's retry timer from firing when PLEG stops advancing. A failed observation
returns an error; it is not interpreted as an empty pod.

PLEG continues to supply ordinary observations and wakeups. It owns no desired
termination state. The worker also owns a retry timer, normally one second for
pending work. Errors use the existing worker backoff, capped by the time remaining
until the pod deadline. After expiry, retries continue. The timer reuses the
worker's last pod specification, so eviction and removed static pods do not
require another update from podManager. A shorter grace request cancels the
current worker context and supplies an earlier deadline on the next invocation.

A slow start may occupy the worker until its context ends. Starts use the pod
deadline and the worker cancellation context. The design depends on CRI and hook
implementations honoring context cancellation; it does not promise progress
through an indefinitely hung runtime call.

### Desired actions and ordering

The reconciler derives desired actions from the pod specification, absolute
deadline, and latest runtime observation:

1. Application containers and non-restartable init containers are never started.
   Their observed live instances must stop.
2. Walk restartable init containers in reverse specification order. A sidecar is
   still needed while any application container, non-restartable init container,
   or later sidecar is observed non-exited. Unknown state is conservatively live.
3. A needed sidecar can restart only if it has previously started, is now exited
   (or has an unstarted replacement from a partial start), has a ready sandbox,
   and has more than one second left. Normal restart backoff applies. Missing
   status for a never-started sidecar does not authorize starting it.
4. Once a sidecar's turn arrives, stop its current observed instance. Do not
   restart a sidecar that exited at or after its turn. At the deadline, stop all
   remaining instances regardless of ordering and remove unstarted instances.

Starts use `startContainer`, including image pull secrets, image volumes and
`postStart`. Secret and configmap managers are registered during termination so
configuration can be resolved after kubelet restart. This registration does not
restore volume-update or token-rotation behavior deferred from Alpha.

Runtime-manager records track outstanding hook and stop calls per container ID,
and successful replacements per exited ID. They suppress duplicate work across
repeated observations but do not define which containers should run. Calls
complete through buffered channels; only the pod worker accesses these records.

### Deadlines and cancellation

For a newly terminating worker, the local deadline is termination start plus the
effective grace period. If the pod has a `DeletionTimestamp`, use the earlier of
that timestamp and the local deadline. A shorter grace request can move the
deadline earlier to request time plus the new grace. Repeated or longer requests
never move it later within that worker's lifetime.

`preStop` begins as soon as a running instance is observed during termination,
including sidecars whose ordered stop is still pending. Each observed replacement
gets its own hook. Hooks and ordering consume the same absolute grace budget.
Hook completion is retained per instance for the lifetime of the runtime manager.

Hooks and stops run asynchronously so another reconciliation can observe exits
and restart eligible sidecars. A stop passes the rounded-up remaining grace to
CRI. Its RPC context allows two additional seconds for transport completion.
When grace is shortened, a superseded stop context is cancelled and a new stop
is issued for the same ID with the shorter grace. Cancellation is not evidence
that the original server-side operation was rolled back or that the container
stopped. Subsequent runtime observations determine progress.

At expiry, hooks are cancelled, new starts are forbidden, created instances are
removed, and remaining containers receive a zero-grace stop. Each retry after
expiry has a bounded stop RPC context. That transport allowance does not add
container shutdown grace or authorize cleanup.

<<[UNRESOLVED deadline compatibility]>>
The proposed implementation does not preserve the legacy kill path's minimum
two-second container grace after a long `preStop` or ordering wait. Reviewers must
choose whether an absolute deadline should force immediately, as implemented, or
whether a single bounded shutdown extension is required. If an extension is
chosen, its recovery and shortening rules must be designed so retries cannot
renew it. The deadline/hook tests currently assert zero-grace stops at expiry.
<<[/UNRESOLVED]>>

### Recovery

No new checkpoint is introduced in Alpha. On kubelet restart, spec and runtime
status reconstruct ordering and restart eligibility. The API deletion timestamp
reconstructs the original deadline even when it has already expired. Backoff and
operation-deduplication records are in memory and may reset.

| Interrupted operation | Observation after restart | Recovery |
| --- | --- | --- |
| Create did not take effect | Previous sidecar exited | Retry through the normal start path |
| Create committed; start did not | Created replacement with a restart attempt | Remove it and retry only while eligible; remove at expiry |
| Start committed; response lost | Running replacement | Keep that instance; do not create another |
| Stop pending or response lost | Container still running | Reissue idempotent stop using the remaining grace |
| Stop committed | Container exited | Advance ordering without waiting for the old RPC result |
| `preStop` interrupted or completed | Same instance still running | Hook may replay while grace remains |

These decisions assume the runtime reports committed operations and enforces
container identity/name reservations for overlapping creation attempts. The
in-memory records are not an exactly-once transaction across kubelet and CRI.

Local evictions and static-pod removals retain their deadline while the same
worker exists. Without an API deletion timestamp, kubelet restart can lose the
original local kill intent and deadline, as with existing local termination.
If only a runtime pod remains and its spec is unavailable, the existing
`SyncTerminatingRuntimePod` path stops it without sidecar restart.

<<[UNRESOLVED local termination recovery]>>
Alpha proposes retaining the existing lack of durable local kill intent. Before
claiming deadline preservation for all termination sources, design a checkpoint
for both the intent and the absolute deadline, including eviction policy,
static-pod replacement, and checkpoint cleanup. Persisting only a timestamp would
not resolve recovery of the intent. Reviewers must explicitly accept this Alpha
scope or require that work before Alpha.
<<[/UNRESOLVED]>>

### Completion, resources and status

The runtime reconciler returns pending while any observed container is non-exited,
including unknown or created instances. Completed stop calls alone do not imply
completion. Once observations show no remaining active containers, the kubelet
stops the sandbox through the existing kill path, reads final runtime status,
and checks that no containers remain running before unpreparing DRA resources
and publishing final status. Only then may the worker transition to
`TerminatedPod` and allow foreground cleanup and runtime removal. Existing status
callbacks, such as eviction marking a pod Failed, may run before containers stop;
API phase alone is not the authorization for resource removal.

Pod status is refreshed during reconciliation, so replacement IDs and restart
counts are observable through the API subject to status publication latency.
Liveness and startup probes stop when termination begins; all probe workers are
removed after the pod stops. The
`kubelet_sidecar_restarts_during_termination_total` counter increments after a
successful restart path and resets when kubelet restarts.

### Lifecycle invariants and review requirements

The implementation and tests must preserve these invariants:

- No application-container restart, or sidecar restart after its turn or deadline.
- Repeated observations and recovery of committed starts do not duplicate a live
  sidecar instance.
- A worker's deadline never extends; API deletion preserves it across kubelet
  restart. Local recovery is limited as described above.
- PLEG silence does not block reconciliation. Runtime errors retain resources
  and schedule retries instead of reporting termination complete.
- Stop response, timer expiry, and cancellation alone never release resources
  or kill waiters. Observed termination and final cleanup checks are required.
- Gate-disabled pods, pods without sidecars, and runtime-only cleanup retain
  their existing lifecycle behavior.

The unresolved compatibility and recovery decisions require KEP approver review.
Unit tests and this document do not imply that review has occurred. The node test
suite must also run against a supported node/runtime before release.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

Tests accompanying [kubernetes/kubernetes#140133] exercise both the state machine
and component boundaries. Fault-injection tests use a fake CRI; they do not
replace execution of the node suite against a real runtime.

##### Prerequisite testing updates

The worker's pending result, timer, and resource-removal predicates must be tested
with the actual pod cache. A fake cache that always returns fresh status cannot
expose a PLEG wait that blocks the deadline.

##### Unit tests

| Invariant or failure | Test in `k8s.io/kubernetes` |
| --- | --- |
| Stalled PLEG; API deletion, eviction and static-pod removal; shorter grace | `pkg/kubelet/pod_workers_termination_test.go`: `TestTerminatingPodProgressesWithStalledPLEG` |
| Pending work retains waiters and runtime resources | Same file: `TestTerminatingPodRequeuesWithoutCompleting` |
| Deadline monotonicity and expired API deadline recovery | Same file: `TestTerminationDeadlineDoesNotReset` |
| New worker reconstructs an expired API deadline | Same file: `TestTerminatingPodRecoversExpiredAPIDeadline` |
| Runtime-only orphan cleanup never restarts sidecars | Same file: `TestTerminatingRuntimePodDoesNotRestartSidecars` |
| Error backoff cannot delay the next attempt beyond remaining grace | Same file: `TestTerminationRetryCannotPassDeadline` |
| Fresh runtime observation supersedes stale cache; DRA and final-status guard | `pkg/kubelet/kubelet_termination_test.go`: `TestSyncTerminatingPodObservesRuntimeBeforeCleanup` |
| Runtime observation failure remains pending and bounded | Same file: `TestSyncTerminatingPodObservationFailureRetainsResources` |
| Gate controls the lifecycle path | Same file: `TestSyncTerminatingPodGateControlsReconciliation` |
| Lost create/start responses before or after side effects; new runtime manager | `pkg/kubelet/kuberuntime/kuberuntime_termination_restart_test.go`: `TestSyncTerminatingPodRecoversInterruptedStart` |
| Stop completion or lost response is insufficient without an observation | Same file: `TestSyncTerminatingPodWaitsForObservedStop` |
| Rejected stop is retried without renewing grace | Same file: `TestSyncTerminatingPodRetriesFailedStop` |
| Shorter grace replaces an outstanding stop request | Same file: `TestSyncTerminatingPodShortensOutstandingStop` |
| Partial start cannot leave a created container past expiry | Same file: `TestSyncTerminatingPodRemovesPartialStartAtDeadline` |
| Earlier sidecar restarts while a later one drains; current instance stops in order | Same file: `TestSyncTerminatingPodOrdersMultipleSidecars` |
| Long hooks do not block reconciliation or renew grace | Same file: `TestSyncTerminatingPodPreStopDoesNotBlockReconciliation`, `TestSyncTerminatingPodDeadlineCancelsHooks` |
| Restart eligibility, deduplication, backoff and retry | Same file: `TestSyncTerminatingPodDoesNotStartIneligibleSidecars`, `TestSyncTerminatingPodRestartsAndDeduplicates`, `TestSyncTerminatingPodRestartBackoff`, `TestSyncTerminatingPodRetriesPartialStart` |
| Unknown container is stopped at expiry despite missing spec | Same file: `TestSyncTerminatingPodDeadlineStopsUnknownContainer` |
| No pod-wide restart during API deletion | `pkg/kubelet/container/helpers_test.go`: `TestShouldAllContainersRestart` |

##### Integration tests

The worker/cache and kubelet/runtime/resource-manager tests above run in package
unit suites and exercise those component boundaries. No separate
`test/integration` suite is claimed for this change. Real kubelet restart,
API deletion and CRI process behavior are exercised by the node tests below.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.

- <test>: <link to test coverage>
-->

###### Alpha implementation tests

`test/e2e_node/sidecar_termination_restart_test.go`, gated by
`SidecarsRestartableDuringPodTermination`, contains these scenarios:

- A sidecar exits during application shutdown, its restart count increases while
  the application is still running, and the pod subsequently terminates.
- A later deletion with shorter grace overrides an ongoing termination; CRI
  observations confirm running containers disappear within the shortened budget.
- Kubelet stops during termination, the original sidecar exits while it is down,
  and the restarted kubelet replaces it and completes within the original API
  deletion deadline (`Serial`, `Disruptive`).

These scenarios require execution on a supported node. Compilation and package
fault-injection tests do not establish containerd or CRI-O behavior, nor prove
that the disruptive kubelet-restart scenario passes.

###### Existing tests

- should respect termination grace period seconds
- should respect termination grace period seconds with long-running preStop hook https://github.com/kubernetes/kubernetes/blob/fbb2e6293fb0c8c107ae48b8b8ae488325c59598/test/e2e_node/container_lifecycle_test.go#L536
- should call the container's preStop hook and terminate it if its startup probe fails https://github.com/kubernetes/kubernetes/blob/master/test/e2e_node/container_lifecycle_test.go#L616
- should call the container's preStop hook and terminate it if its liveness probe fails https://github.com/kubernetes/kubernetes/blob/fbb2e6293fb0c8c107ae48b8b8ae488325c59598/test/e2e_node/container_lifecycle_test.go#L683

###### Beta (planned)

The Alpha node scenarios above cover restart and grace-period behavior. Beta
adds probe, hook replay, and configuration-lifetime assertions on real runtimes.
Probe scenarios depend on implementing probe reattachment. Hook execution already
uses the Alpha lifecycle paths; end-to-end validation must cover replacement
instances and kubelet restart, not assume exactly-once delivery. Service-account
token work (#116481, #122568) remains tracked separately.

Probes:
- Readiness probes are still running while in preStop
- Readiness status is beings updated for the container and the Pod while in preStop
- Liveness probes are NOT running for regular containers while the Pod is terminating 
- SIDECAR: Liveness probes DO run for sidecar containers while the Pod is terminating 
- SIDECAR: sidecar container will be restarted when liveness probe failed during Pod termination

Not fully started containers:
- preStop will not be executed for the container that hasn’t started yet
- preStop will be called on the container even if postStart is still running
- postStart hook CONTINUE EXECUTE even if container started termination
- postStart hook will stop once pod passed it’s graceful termination period

Re-terminating the Pod:
- When the Pod is terminating, another request with greater grace must not extend the deadline
- BUGFIX: Service account token gets invalidated while terminating pod is re-deleted · Issue #122568

Pre-stop vs. SIGTERM traps:
- Same as existing and above tests, need to validate that the container that traps the SIGTERM behaves the same way as with preStop:
- Respect the grace period
- Liveness probes are not running
- Readiness probes are running

Test what is available for during preStop:
- BUGFIX: While the Pod is terminating, service account tokens are rotated Kubelet stops rotating service account tokens when pod is terminating, breaking preStop hooks · Issue #116481
- BUGFIX: Service account token is valid if the terminating Pod was deleted again Service account token gets invalidated while terminating pod is re-deleted · Issue #122568

Eviction and OOM kills:
- preStop is called when Pod is evicted
- preStop is NOT called when Container is OOMkilled

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

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

#### Alpha

- Feature implemented behind `SidecarsRestartableDuringPodTermination`, disabled
  by default, targeting v1.38.
- KEP approvers accept the worker reconciliation contract and explicitly resolve
  the deadline-compatibility and local-recovery scope decisions above.
- The lifecycle invariants have passing package tests, including fault injection
  at component boundaries and gate-disabled regression coverage.
- Node restart, shorter-grace, and ordered shutdown scenarios execute successfully
  against a supported runtime. Test results are linked during implementation
  review; merely adding or compiling the tests is insufficient.

#### Beta

- Resolve remaining Alpha limitations, including probe reattachment and its
  interaction with restart backoff during termination.
- Decide whether to persist local termination intent/deadlines based on the Alpha
  recovery scope; test any durable recovery behavior before promising it.
- Validate hook replay, replacement hooks, configuration lifetime, runtime
  cancellation and ambiguous CRI outcomes on real nodes.
- Node tests pass in Testgrid without flakes for two consecutive releases.
- Feedback from Alpha adopters is addressed.

#### GA

TBD

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->
This feature only concerns the kubelet, so the upgrade and downgrade strategy is limited to the kubelet.
Moreover, the Pod spec is not altered, so no changes are required for existing workloads to make use of the feature.
Likewise, no changes are required for these workloads to revert to previous behavior.

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->
There is no version skew strategy for this feature.
The kubelet is the only component that needs to be updated to make use of this feature.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: SidecarsRestartableDuringPodTermination
  - Components depending on the feature gate:
    - kubelet
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->
Enabling the feature will change the behavior of the kubelet when terminating a Pod with sidecar containers.
Sidecar containers that exit prematurely will be restarted during the termination of the Pod to ensure they are running
until the main containers that should terminate prior to the sidecar containers are still running.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->
Yes, the feature can be disabled once it has been enabled.
There is no alteration to the Pod spec, so existing workloads will be terminated according to the current behavior,
after the kubelet is restarted with the feature gate disabled.

###### What happens if we reenable the feature if it was previously rolled back?
No side effect, the feature can be switched on or off.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->
Yes, unit tests will be added to ensure the feature can be enabled and disabled.
The KEP will be updated with the details of the tests as they are added.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

The `kubelet_sidecar_restarts_during_termination_total` counter (per node) is incremented every
time a sidecar is restarted during pod termination. A non-zero and increasing value indicates
the feature is enabled and actively restarting sidecars for terminating pods.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [X] API .status
  - Fields: `initContainerStatuses[].containerID`, `restartCount`, and `state`
  - Publication is asynchronous while the terminating pod still exists.
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [X] Metrics
  - Metric name: `kubelet_sidecar_restarts_during_termination_total`
  - Components exposing the metric: kubelet
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

Terminating sidecar pods add direct runtime status reads on each reconciliation.
Pending timer retries normally occur once per second; pod updates can trigger
additional reconciliations. This trades additional CRI reads for independence
from stalled PLEG and stale observations after partial starts. Runtime latency
and concurrent terminating-pod load need measurement before Beta.

Each pod retains one worker timer, per-instance operation records, and bounded
hook/stop calls. Restart backoff limits repeated failing starts; deadline expiry
forbids further starts. Replacements still consume normal pod resources, and the
feature does not increase pod resource limits. A runtime that ignores
cancellation can retain server-side work beyond a client timeout; real-runtime
fault tests must cover this limitation.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

- 2024-01-30: `Summary` and `Motivation` sections merged
- 2024-02-08: `Proposal` section merged, KEP marked as `implementable`
- 2026-09-07: Proposed worker reconciliation design and lifecycle regression
  tests in [kubernetes/kubernetes#140133], targeting v1.38 Alpha. KEP lifecycle
  review and real-node validation remain release requirements.

[kubernetes/kubernetes#140133]: https://github.com/kubernetes/kubernetes/pull/140133

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

The main drawback of this KEP is that it introduces a new feature gate for the sidecars,
which can be confusing for users. However, we believe that the current behavior of sidecar containers
is already useful and that the restart during Pod termination feature is not critical for many use cases.
This is why this feature is introduced as a separate feature gate, so that KEP-753 can reach GA faster.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

The alternative would be to introduce KEP-753 with the restart during Pod termination feature.
However, this would have required a significant refactoring of the kubelet
and the Pod lifecycle, which would introduce a risk of disruption to users. This is why we decided to
introduce a separate feature gate for the sidecar containers feature.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
