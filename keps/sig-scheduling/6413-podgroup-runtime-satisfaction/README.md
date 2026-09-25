# KEP-6413: PodGroup runtime satisfaction condition

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [What the scheduler reports today](#what-the-scheduler-reports-today)
  - [What controllers do today](#what-controllers-do-today)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: an operator asks whether a replica is healthy](#story-1-an-operator-asks-whether-a-replica-is-healthy)
    - [Story 2: a workload controller decides whether to recreate a group](#story-2-a-workload-controller-decides-whether-to-recreate-a-group)
    - [Story 3: a disruption budget counts group replicas](#story-3-a-disruption-budget-counts-group-replicas)
    - [Story 4: a dashboard shows degraded gangs](#story-4-a-dashboard-shows-degraded-gangs)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API changes](#api-changes)
  - [Condition semantics](#condition-semantics)
  - [Where the condition is computed and written](#where-the-condition-is-computed-and-written)
  - [Interaction with other Workload-Aware Scheduling features](#interaction-with-other-workload-aware-scheduling-features)
  - [Consumers](#consumers)
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
- [Future work: a recovery policy for gangs](#future-work-a-recovery-policy-for-gangs)
  - [Use cases](#use-cases)
  - [Shape, if it is pursued](#shape-if-it-is-pursued)
- [Open Questions](#open-questions)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure that all e2e tests have been promoted to conformance for GA
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation (e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes)

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

A gang `PodGroup` carries exactly one condition, `PodGroupInitiallyScheduled`,
and that condition is terminal by design: once it is `True` it never reverts,
"even if pods are subsequently evicted and group constraints are no longer
met". Nothing else in the API reports how many members the group currently
has. A workload controller, a disruption controller or an operator asking "is
this gang currently intact" has no answer to read, and each one re-derives it
from the member pods instead.

This KEP adds one condition, `PodGroupSatisfied`, reporting whether the group
currently has at least `minCount` scheduled members. It transitions in both
directions for the life of the group. It is computed from per-group state the
scheduler cache already maintains, and it changes no scheduling behavior.

`PodGroup` already carries a `[]metav1.Condition`, so this adds no field and
no schema change. The whole feature is a new condition value written by
kube-scheduler behind a feature gate.

A second question, whether the owner of a gang should be able to choose how
replacement members are admitted after the group has been placed, is
deliberately **not** part of this KEP. It is recorded under
[Future work](#future-work-a-recovery-policy-for-gangs) with the use cases
that motivate it, so that the observability half can land on its own.

## Motivation

### What the scheduler reports today

`PodGroupInitiallyScheduled` is documented in
`staging/src/k8s.io/api/scheduling/v1beta1/types.go` as terminal: "Once this
condition transitions to True, it serves as a terminal state and will never
revert to False, even if pods are subsequently evicted and group constraints
are no longer met." The scheduler enforces that with an explicit guard in
`updatePodGroupCondition` (`pkg/scheduler/schedule_one_podgroup.go`) that
refuses to regress the condition.

[KEP-4671] is explicit that this is a known gap rather than a decision against
live status. It records, as a possible future extension, "a more robust status
lifecycle mechanism capable of reflecting live post-scheduling state changes,
including current pod counts", and notes that "a new, separate component"
might own it. It also states that "scheduled pods may later be evicted or
impacted by node failures, but the `PodGroup` status will not track these
post-scheduling disruptions". No issue or KEP has picked that up, and
[kubernetes/kubernetes#136334] is where the need was most recently raised.

The information itself is not missing, only unexposed. The scheduler cache
keeps a `podGroupState` per known group
(`pkg/scheduler/backend/cache/podgroupstate.go`), updated from the pod event
handlers through `AddPodGroupMember`, `UpdatePodGroupMember` and
`RemovePodGroupMember`. `ScheduledPodsCount()` on that state is exactly the
number the gang plugin compares against `minCount` when admitting members.
Nothing publishes it.

### What controllers do today

Every consumer that needs group health derives it independently, and at least
one derives it wrongly.

**LeaderWorkerSet** infers group health from Pod phase. Its restart handling
guards on `pendingPodsInGroup` (`pkg/controllers/pod_controller.go`), which
treats "any member is Pending" as "the group is still starting up, do not
interfere". Under gang scheduling, Pending is also the steady state of a group
waiting for capacity, so one signal now carries two opposite meanings. The
native Workload integration in [kubernetes-sigs/lws#979] additionally lists
`PodGroup.status.conditions[type=PodGroupInitiallyScheduled]` as a surface
users inspect, and reuses a replica's `PodGroup` across a leader restart. Once
groups are reused, that condition reads `True` for a replica whose members are
all gone.

**The multi-pod PodDisruptionBudget KEP** ([kubernetes/enhancements#5671])
defines a replica as healthy "if and only if `healthy_pods >= minCount`" and
has the eviction path fetch the `PodGroup`, read `minCount` out of its spec,
and count member pods itself, because the group does not report it.

**JobSet** records "recover one failed component" as future work in its gang
KEP ([kubernetes-sigs/jobset#1253]): "partial eviction and rescheduling of an
individual ReplicatedJob remain future work".

**StatefulSet and Deployment** integrations ([KEP-6277], [KEP-6276]) bring
gang policies to in-tree controllers, which widens the set of consumers that
will ask this question.

Each of these re-implements a count that the scheduler already has, using a
different definition of "scheduled" than the scheduler uses, with no way to
agree during the window between assume and bind.

### Goals

- Report, in `PodGroup` status, whether the group currently has at least
  `minCount` scheduled members, in a form that transitions in both directions.
- Give that signal a single definition of "scheduled", the one the gang plugin
  itself uses, so consumers agree with the scheduler and with each other.
- Change no scheduling behavior and no API schema.

### Non-Goals

- **A recovery policy for gangs.** Whether a gang owner can choose how
  replacement members are admitted after initial placement is a separate
  question, deferred to
  [Future work](#future-work-a-recovery-policy-for-gangs) pending agreement on
  the use cases.
- Rescheduling or replacing members. This KEP creates no pods, evicts no pods
  and moves no pods. Those remain controller responsibilities.
- Changing what `minCount` means, or how initial placement works.
- Changing the meaning or the terminal nature of
  `PodGroupInitiallyScheduled`.
- Workload-aware handling of planned disruption, owned by [KEP-4563] and
  `disruptionMode`.
- Reporting satisfaction for `CompositePodGroup` hierarchies. Alpha scopes the
  condition to leaf `PodGroup`s; see [Open Questions](#open-questions).

## Proposal

Add a condition type `PodGroupSatisfied` to `PodGroup.status.conditions`.

It is `True` when the number of scheduled members, counting assumed plus
assigned members exactly as `permitPodGroup` does, is at least the effective
`minCount`. It is `False` otherwise. Unlike `PodGroupInitiallyScheduled` it
transitions in both directions for the life of the group.

The condition is observational. No scheduler decision reads it.

### User Stories

#### Story 1: an operator asks whether a replica is healthy

An operator runs a tensor-parallel model as a four-pod gang. A node is
drained and one worker is evicted. Today `kubectl describe podgroup` shows
`PodGroupInitiallyScheduled=True` and nothing else, which is the same output
as a fully healthy group. With this KEP the group also shows
`PodGroupSatisfied=False`, reason `BelowMinCount`, message `3 of 4 members
scheduled`, and it flips back to `True` when the replacement lands.

#### Story 2: a workload controller decides whether to recreate a group

A controller wants to distinguish "the group has not formed yet" from "the
group formed and has since degraded", because the two call for different
handling. Today LeaderWorkerSet approximates the first with "any member is
Pending", which is wrong under gang scheduling. With both conditions present
the distinction is explicit: `PodGroupInitiallyScheduled` answers whether the
group ever formed, `PodGroupSatisfied` answers whether it is intact now.

#### Story 3: a disruption budget counts group replicas

The multi-pod PDB KEP needs to know whether each `PodGroup` replica is
healthy, and currently derives `healthy_pods >= minCount` in the eviction
path. With the condition it can read one scheduler-maintained signal, which
also removes the possibility of the eviction path and the scheduler
disagreeing about what counts as scheduled.

#### Story 4: a dashboard shows degraded gangs

A platform team wants an alert for gangs that have been below `minCount` for
more than a few minutes, which today requires joining pod-level metrics
against `PodGroup` specs. A condition with a `LastTransitionTime`, plus the
transition metric this KEP adds, makes that a single query.

### Notes/Constraints/Caveats

- The scheduler writes `PodGroup` status only from inside a group scheduling
  cycle today. A group that loses members is not being scheduled, so the
  condition needs a write path outside the cycle. See
  [Where the condition is computed and written](#where-the-condition-is-computed-and-written).
- Members leave the cache on the pod delete event, not when a deletion
  timestamp is set, so a terminating member still counts as scheduled until
  it is gone. The condition inherits that definition. It is stated here
  because a controller that recreates groups will observe the window, and
  because any other choice would make the condition disagree with the
  admission behavior it describes.
- `minCount` is mutable, so the condition can flip without any pod changing.

### Risks and Mitigations

**Status write amplification.** A large group whose members churn could
generate frequent status writes.

*Mitigation:* write only on transitions across the `minCount` boundary, not on
every count change; coalesce same-status updates; route writes through the
asynchronous API-call machinery from [KEP-5229] when it is available so that
informer event handlers never block on the API. The scalability section
quantifies the resulting write rate.

**Consumers treat an observational condition as authoritative for eviction
safety.** A controller could read `PodGroupSatisfied=True` and skip its own
checks in a window where the cache is briefly stale.

*Mitigation:* document the condition as eventually consistent with the same
staleness characteristics as pod status, and keep fail-closed behavior in
consumers such as the PDB path.

**A beta API gains a condition.** `PodGroup` is `v1beta1` in v1.37 and targets
GA in v1.38 ([kubernetes/enhancements#6349]).

*Mitigation:* conditions are a `[]metav1.Condition` list with no enumeration
of valid types, so this adds no schema change and no validation change. The
condition is only written when the feature gate is on, and consumers that
predate it ignore an unknown condition type.

## Design Details

### API changes

No schema change. `PodGroupStatus.Conditions` already exists. This KEP adds
one known condition type and two reasons as Go constants next to the existing
ones in `staging/src/k8s.io/api/scheduling/v1beta1/types.go`, and documents
them in the `Conditions` field comment:

```go
const (
	// PodGroupSatisfied reports whether the PodGroup currently has at least
	// minCount scheduled members. Unlike PodGroupInitiallyScheduled, which is
	// terminal, this condition transitions in both directions for the life of
	// the PodGroup.
	PodGroupSatisfied string = "PodGroupSatisfied"

	// BelowMinCount reason in the PodGroupSatisfied condition indicates that
	// fewer than minCount members are currently scheduled.
	BelowMinCount string = "BelowMinCount"

	// MinCountSatisfied reason in the PodGroupSatisfied condition indicates
	// that at least minCount members are currently scheduled.
	MinCountSatisfied string = "MinCountSatisfied"
)
```

Because no field is added, kube-apiserver needs no change and the feature gate
is scheduler-only.

### Condition semantics

"Scheduled" means assumed plus assigned members, the same count
`permitPodGroup` compares against `minCount`. The effective `minCount` is the
gang policy's value.

| Event | Result |
|---|---|
| Group created, no members scheduled | Condition absent until the first transition is computed. |
| Scheduled count reaches `minCount` | `True`, reason `MinCountSatisfied`, message `N of M members scheduled`. Written alongside `PodGroupInitiallyScheduled=True` on initial placement. |
| A member is deleted, evicted or preempted, dropping the count below `minCount` | `False`, reason `BelowMinCount`. |
| A replacement member is scheduled, restoring the count | `True`, reason `MinCountSatisfied`. |
| `minCount` raised above the current count | `False`, reason `BelowMinCount`, with no pod change. |
| `minCount` lowered to or below the current count | `True`. |
| Scheduler restarts | Recomputed once the informer cache syncs; no transition is recorded if the value is unchanged. |

`LastTransitionTime` changes only on `True`/`False` transitions. The message
may be refreshed on same-status writes, and implementations should coalesce
those.

In alpha the condition is written for `PodGroup`s with a gang scheduling
policy. Whether to write it for `basic` policy groups, where the effective
`minCount` is 1, is an open question.

### Where the condition is computed and written

The count already exists in the scheduler cache. The work is publishing it.

1. The cache records a pending transition for a group when a member operation
   moves `ScheduledPodsCount()` across the `minCount` boundary, or when a
   `PodGroup` update moves `minCount` across the current count.
2. A status writer in the scheduler consumes pending transitions and patches
   the condition through the same strategic merge patch path
   `updatePodGroupCondition` uses. With `SchedulerAsyncAPICalls` enabled the
   patch goes through the asynchronous API-call queue ([KEP-5229]);
   otherwise it is issued from a dedicated goroutine, never from an informer
   event handler.
3. The existing in-cycle writer sets `PodGroupSatisfied=True` alongside
   `PodGroupInitiallyScheduled=True` when a gang is first placed, so the two
   conditions appear together rather than one lagging the other.

The scheduler is the right writer because it is the only component that
already distinguishes assumed from assigned members. A separate controller
listing pods would disagree with the scheduler during the assume-to-bind
window and would duplicate the cache; see [Alternatives](#alternatives).

### Interaction with other Workload-Aware Scheduling features

- **Gang scheduling ([KEP-4671]).** No behavior change. The condition
  describes the same relation the plugin enforces, without altering it, and
  does not touch `PodGroupInitiallyScheduled`.
- **Workload-aware preemption ([KEP-5710]).** A group preempted as a whole
  drops below `minCount` and reports `False`. Victim selection is unchanged.
- **Topology-aware workload scheduling ([KEP-5732]).** Unaffected; the
  condition counts members and says nothing about placement.
- **CompositePodGroup ([KEP-6012]).** Alpha scopes the condition to leaf
  `PodGroup`s. `minGroupCount` raises the same runtime question one level up,
  which this KEP does not answer.
- **EvictionRequest ([KEP-4563]) and `disruptionMode`.** Orthogonal. Those
  govern whether members may be disrupted; this reports what happened after
  they were.

### Consumers

| Consumer | What it gains |
|---|---|
| LeaderWorkerSet ([kubernetes-sigs/lws#979]) | A correct source for group health, replacing inference from Pod phase, and a correction to the observability guidance that currently points at the terminal condition |
| Multi-pod PDB ([kubernetes/enhancements#5671]) | Replaces the eviction path's own `healthy_pods >= minCount` derivation with the scheduler's own count |
| JobSet ([kubernetes-sigs/jobset#1253]) | A signal for the deferred "recover one failed component" work |
| StatefulSet ([KEP-6277]) and Deployment ([KEP-6276]) integrations | Group health in status for gang-enabled sets |
| Operators and dashboards | A direct answer to "is this gang intact", with a transition timestamp |

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

None identified. The gang plugin and the `PodGroup` status path already have
unit and integration coverage that this KEP extends.

##### Unit tests

- `pkg/scheduler/backend/cache`: transition detection when members are added,
  updated to assigned, and removed, and when `minCount` changes in either
  direction.
- Status writer: coalescing of same-status updates, no write without a
  transition, no write when the gate is off.
- Condition message formatting for counts and for `minCount` changes.

##### Integration tests

- A gang of four is placed; assert `PodGroupSatisfied=True` appears with
  `PodGroupInitiallyScheduled=True`. Delete one member; assert `False` with
  message `3 of 4 members scheduled`. Let the replacement be placed; assert
  `True` again.
- Raise `minCount` above the scheduled count; assert the condition flips with
  no pod change. Lower it again; assert it flips back.
- Restart the scheduler with a degraded group; assert the condition is
  unchanged and no spurious transition is recorded.
- Feature gate disabled: no condition is written, and an existing condition
  from a previous run is left untouched.

##### e2e tests

- One test in the scheduling e2e suite mirroring the first integration case
  against a real cluster, to be promoted with beta graduation.

### Graduation Criteria

#### Alpha

- Feature gate `PodGroupRuntimeSatisfaction` off by default, kube-scheduler
  only.
- Condition implemented, written on transitions, and documented in the API
  field comment.
- Unit and integration tests above.
- At least one consumer with a design that reads it.

#### Beta

- Gate on by default.
- Transition metric in place, and the write-rate measurement in the
  scalability section confirmed on a large cluster.
- Feedback from at least two consumers.
- Decisions recorded on `basic` policy groups and on CompositePodGroup scope.
- e2e test stable.

#### GA

- Two releases at beta with no semantic changes.
- Conformance test for the condition on a gang `PodGroup`.

### Upgrade / Downgrade Strategy

Enabling the gate starts writing the condition for existing gang groups at the
next transition, or on scheduler start once the informer cache syncs.
Disabling it stops the writes; a stale `PodGroupSatisfied` condition is left
on the object, is ignored by the scheduler, and disappears when the group is
deleted. No object needs migration in either direction.

### Version Skew Strategy

Only kube-scheduler is involved, so there is no skew between components for
this feature. Conditions are free-form `metav1.Condition` entries, so a
scheduler with the gate on can write the condition against an older
kube-apiserver. Consumers that predate the KEP ignore an unknown condition
type.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `PodGroupRuntimeSatisfaction`
  - Components depending on the feature gate: kube-scheduler

###### Does enabling the feature change any default behavior?

No. The condition is additive and no scheduling decision reads it.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Writes stop; a stale condition is inert.

###### What happens if we reenable the feature if it was previously rolled back?

The scheduler resumes writing from current cache state.

###### Are there any tests for feature enablement/disablement?

Yes, planned as unit tests on the status writer and one integration test that
toggles the gate.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A defect in the status writer could produce excessive writes or a wrong
condition value. Neither touches running pods, since nothing consumes the
condition inside the scheduler.

###### What specific metrics should inform a rollback?

`scheduler_podgroup_satisfaction_transitions_total` rising without bound on a
stable cluster, or apiserver write latency on `podgroups/status`.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

To be done before beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Gang `PodGroup` objects carry a `PodGroupSatisfied` condition.

###### How can someone using this feature know that it is working for their instance?

- [x] API .status
  - Condition name: `PodGroupSatisfied`
  - Other field: message of the form `N of M members scheduled`

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The condition reflects cache state within the scheduler's normal event
processing latency, comparable to existing pod status updates.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `scheduler_podgroup_satisfaction_transitions_total`
  - Components exposing the metric: kube-scheduler

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A histogram of time spent below `minCount` per group would help alerting and
is deferred to beta.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. It builds on the `PodGroup` API and the `GenericWorkload` feature from
[KEP-4671].

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes: one `podgroups/status` patch per transition across the `minCount`
boundary. In a stable cluster transitions are rare, bounded by member churn,
and at most two per group per disruption event (one down, one back up).

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

One additional condition entry per gang `PodGroup`.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. Transition detection is a comparison against a count the cache already
maintains, and the write is off the scheduling path.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Negligible: one pending-transition set in the scheduler cache.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Status writes fail and are retried. Scheduling is unaffected, since nothing in
the scheduling path reads the condition.

###### What are other known failure modes?

- The condition lags reality after a scheduler restart until the informer
  cache syncs.
- A terminating member counts as scheduled until the delete event arrives.

###### What steps should be taken if SLOs are not being met to determine the problem?

Inspect `PodGroup` conditions and scheduler logs at verbosity 4 for the
transition writer.

## Implementation History

- 2026-09-12: Initial draft.
- 2026-09-15: Scope narrowed to the condition after discussion on
  [kubernetes/kubernetes#136334]; the recovery policy moved to future work
  with its use cases.

## Drawbacks

- A second condition sits next to one that already confuses readers. The names
  and the field documentation have to make the difference obvious, and the
  website documentation should present them together.
- The condition is eventually consistent, so a consumer that treats it as a
  hard interlock will eventually be wrong.

## Alternatives

**Let each consumer derive it from pods.** This is the status quo. Three
consumers already do it, LeaderWorkerSet does it wrongly under gang
scheduling, and none of them can agree with the scheduler during the
assume-to-bind window.

**A separate controller in kube-controller-manager.** [KEP-4671] floated "a
new, separate component" for live status. Such a controller would list member
pods and recompute the count, duplicating the scheduler cache and disagreeing
with it during assume-to-bind. The scheduler is the only component that
already knows the answer.

**Make `PodGroupInitiallyScheduled` non-terminal.** This would answer the
question with no new condition, but it breaks the documented contract of a
beta field, changes the meaning of existing data, and loses the "did this
group ever form" signal that controllers separately need.

**Expose the count without a condition, as a status field.** A
`scheduledMembers` integer would be more precise than a boolean condition, but
it is a schema change to a beta API, it invites write amplification on every
count change rather than on transitions, and it does not fit the conventional
way Kubernetes reports state. A condition carrying the count in its message
was preferred; see [Open Questions](#open-questions).

## Future work: a recovery policy for gangs

This is recorded for the discussion on [kubernetes/kubernetes#136334] and is
explicitly out of scope above. No decision is asked for in this KEP.

The question is what happens to *replacement* members after a group has been
placed. kube-scheduler admits them only when scheduled plus newly feasible
members reach `minCount` again, at all three enforcement points in the gang
plugin. Koordinator does the opposite, and states it plainly in its design:
"when once Gang has been satisfied, all subsequent resource allocations are no
longer constrained by Gang rules, and their performance is similar to ordinary
pod." Both behaviors ship in production, and a `PodGroup` owner cannot choose
between them.

### Use cases

**A documented per-pod restart contract that gang scheduling silently
overrides.** LeaderWorkerSet's `restartPolicy: None` is documented in its API
as "only the failed pod will be restarted on failure and other pods in the
group will not be impacted". Nothing in LWS or in its gang KEP
([kubernetes-sigs/lws#979], whose non-goals list restricts only
`startupPolicy: LeaderReady` and `RecreateGroupAfterStart`) prevents combining
that restart policy with a gang `PodGroup`. When one member is lost the
promise holds, because three scheduled plus one feasible replacement reaches
`minCount`. When a node carrying two members of the same group is drained and
capacity exists for only one replacement, that replacement is withheld and the
documented per-pod behavior silently becomes all-or-nothing. The user
configured per-pod recovery and gang admission, and got neither.

**Serving workloads that tolerate reduced capacity.** A data-parallel
inference deployment gang-schedules to guarantee it starts with enough
replicas, but once running, a replica short is reduced throughput rather than
an outage. Under the current rule, a multi-member loss keeps every feasible
replacement Pending until the whole shortfall can be satisfied at once, which
is exactly when the cluster is most capacity-constrained.

**Training workloads that must not proceed degraded.** The counter-case, and
the reason the current behavior must remain the default. A synchronous
data-parallel training job makes no progress below `minCount`, and a
replacement that lands early only occupies an accelerator while waiting.

**Production evidence on both sides.** Koordinator shipped once-satisfied as
its only behavior and tracks it with a `ResourceSatisfied` flag exposed as
`onceResourceSatisfied` in its debug API. kube-scheduler shipped the opposite
as its only behavior. Neither community appears to have argued the other is
wrong, which suggests the real user set is non-empty on both sides.

### Shape, if it is pursued

An optional `recoveryPolicy` on the gang scheduling policy with
`RequireMinCount` (today's behavior, the default when unset) and
`OnceSatisfied`. Under `OnceSatisfied`, once `PodGroupInitiallyScheduled` is
`True`, the three enforcement points skip the count relation while every other
constraint, including topology, still applies. Switching on the durable
condition rather than in-memory state means a scheduler restart does not
forget that the group was satisfied once. Initial placement is all-or-nothing
under both values.

The third behavior named in #136334, `OnlyWaiting`, where replacements form a
new gang among themselves and surviving members are not counted, has no
identified consumer and is not proposed.

## Open Questions

1. Should the condition be written for `basic` policy groups, where the
   effective `minCount` is 1, or only for gang policies?
2. Is the count in the condition message sufficient, or should a follow-up add
   a `scheduledMembers` status field? A message is not machine-readable, and a
   consumer that wants the number has to parse it or count pods anyway.
3. Should a member with a deletion timestamp still count as scheduled? Keeping
   it matches the admission behavior the condition describes; dropping it
   would report degradation earlier, which is what a controller reacting to
   node failure would want.
4. Does `CompositePodGroup` need an equivalent condition for `minGroupCount`,
   and should that be this KEP's beta scope or a separate one?
5. Naming: `PodGroupSatisfied` versus something that does not invite confusion
   with `PodGroupInitiallyScheduled`, such as `PodGroupMinCountSatisfied`.

[KEP-4563]: /keps/sig-node/4563-eviction-request-api/README.md
[KEP-4671]: /keps/sig-scheduling/4671-gang-scheduling/README.md
[KEP-5229]: /keps/sig-scheduling/5229-asynchronous-api-calls-during-scheduling/README.md
[KEP-5710]: /keps/sig-scheduling/5710-workload-aware-preemption/README.md
[KEP-5732]: /keps/sig-scheduling/5732-topology-aware-workload-scheduling/README.md
[KEP-6012]: /keps/sig-scheduling/6012-composite-podgroup-api/README.md
[KEP-6276]: https://github.com/kubernetes/enhancements/issues/6276
[KEP-6277]: https://github.com/kubernetes/enhancements/issues/6277
[kubernetes/kubernetes#136334]: https://github.com/kubernetes/kubernetes/issues/136334
[kubernetes/enhancements#5671]: https://github.com/kubernetes/enhancements/pull/5671
[kubernetes/enhancements#6349]: https://github.com/kubernetes/enhancements/pull/6349
[kubernetes-sigs/lws#979]: https://github.com/kubernetes-sigs/lws/pull/979
[kubernetes-sigs/jobset#1253]: https://github.com/kubernetes-sigs/jobset/pull/1253
