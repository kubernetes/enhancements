# KEP-6361: Leader Election Recovery

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [The write gate](#the-write-gate)
  - [The election loop keeps running](#the-election-loop-keeps-running)
  - [The recovery deadline](#the-recovery-deadline)
  - [User stories](#user-stories)
    - [Apiserver or etcd unavailability](#apiserver-or-etcd-unavailability)
    - [The leader is network-partitioned](#the-leader-is-network-partitioned)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Configuration](#configuration)
  - [Implementation Notes](#implementation-notes)
  - [Health checks](#health-checks)
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

This KEP is provisional. No release milestone has been selected.

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation, such as additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kubernetes client-go based controller managers use Lease-based leader election
to provide mutual exclusion.

When Lease renewal fails for longer than client-go's `RenewDeadline`, the
controller managers exit. Exiting guarantees that all controllers halts
reconciliation, which preserves the mutual exclusion guarantee. As a
consequence, temporary kube-apiserver or etcd unavailability amplifies into
larger interruptions requiring controller managers to restart and rebuild
caches.

The goal of this KEP is to minimize the interruption.

We propose a client-go **write gate** that is only open when the controller
manager is the leader. It is enabled by an opt-in recovery mode:

- The write gate is a client-go transport wrapper. While the gate is closed,
  write requests are rejected and any in-flight write requests are cancelled.
- client-go closes the write gate at the same time as the `OnStoppedLeading`
  callback (the callback no longer forces the process to exit).
- Instead of returning, the election loop keeps running with the same identity.
  If a later renewal succeeds, client-go reopens the gate. The controllers never
  stopped, so reconciliation simply resumes with the existing caches.

Keeping the process alive allows for fast, low-overhead recovery. In many cases,
the informer caches remain warm.

This feature will be gated by a `LeaderElectionRecovery` client-go feature gate.

## Motivation

Today, if the kube-apiserver or etcd is unavailable for a period of time, the
system amplifies the interruption by exiting every controller loop. This forces
every controller to rebuild its caches and restart its reconciliation.

### Goals

- Controller managers keep their caches when renewal fails and resume
  reconciliation quickly when the lease is renewed.
- Enforce "no reconciliation writes while not leading" in the client-go
  transport layer, independently of how quickly controller loops stop. This
  preserves mutual exclusion at the framework level. Controller developers do
  not need to change their controllers to benefit.

### Non-Goals

- Protecting against misbehaved Lease writers. The leader election protocol
  continues to assume lease candidates are coordinated and cooperative.

## Proposal

### The write gate

Implement the write gate as a `transport.WrapperFunc` in `LeaderElector`. The
controller manager applies it to the `rest.Config` its controllers use.

| Gate state | Write requests (POST, PUT, PATCH, DELETE) | Read requests (GET, including list and watch) |
| :--- | :--- | :--- |
| Closed | Rejected immediately with a distinct error. Requests already in flight are cancelled. | Pass through. |
| Open | Pass through. | Pass through. |

The gate classifies requests by HTTP method. This is conservative: POST-based
read-like APIs such as `SubjectAccessReview` are also blocked while the gate is
closed. Informers continue to run, so caches stay current while the process is
not leading.

The election client uses an ungated configuration, so it can renew while the
gate is closed.

The gate opens at the time of `OnStartedLeading` and closes at the time of
`OnStoppedLeading`.

### The election loop keeps running

Today, `Run` returns after `OnStoppedLeading`. In recovery mode, the elector
instead keeps attempting renewal at the existing retry period, with the same
identity, using the existing read-then-conditional-update path. The next transition is one of:

1. **Renewal succeeds:** No other replica took over. client-go opens the gate.
   Reconciliation resumes with the existing caches.
2. **Another lease candidate takes over:** Its conditional update commits first,
   and the holder's next attempt observes the new holder. `OnNewLeader` reports
   the new identity as it does today, and the elector exits. (We don't strictly
   need to exit, but it keeps the resource utilization of the system similar to
   how it is today, where only the active leader maintains an informer cache.)
3. **The recovery deadline passes:** The elector stops trying and exits, exactly
   as it does today, just later.

### The recovery deadline

A holder that cannot renew is not necessarily silent. If its requests are slow
but still commit, every committed renewal restarts the other candidates' wait,
while the gate stays closed because no response arrives within the renew
deadline. Today that holder exits after one renew deadline and stops writing.

The recovery deadline is the one setting this KEP adds. It bounds how long the
holder keeps trying before it exits. A larger value rides through longer
outages; a smaller value lets another candidate take over sooner from a holder
that is unhealthy but still writing.

### User stories

#### Apiserver or etcd unavailability

During a short outage, the leader's renewals fail. When `RenewDeadline` passes,
the write gate closes. The controllers keep running, but their writes fail and
are retried by their workqueues. When the outage ends, a successful lease renewal
reopens the gate and reconciliation resumes without a restart or a cache
rebuild.

#### The leader is network-partitioned

Only the leader loses connectivity. Its gate closes after `RenewDeadline`, and
it retries without success. A standby observes no change to the Lease for a full
lease duration and takes over, exactly as today.

### Risks and Mitigations

- **Active reconciliation attempts while the gate is closed incur extra cost.**
  The main cost is maintaining informer caches on multiple controller manager
  instances at the same time. This is mitigated by making the feature opt-in
  and by offering a recovery deadline.

## Design Details

### Configuration

```go
type LeaderElectionConfig struct {
	// ... existing fields unchanged ...

	// Recovery configures how a controller manager handles API unavailability.
	Recovery *RecoveryConfig
}

// RecoveryConfig configures how a controller manager handles API unavailability.
type RecoveryConfig struct {
	// RecoveryDeadline limits how long to attempt to claim the lease after losing
	// it due to API unavailability. When the limit is reached, Run returns and
	// callers exit.
	//
	// Regardless of the deadline, if another candidate claims the lease, Run
	// returns and callers exit.
	//
	// If unset, there is no limit.
	RecoveryDeadline time.Duration
}
```

### Implementation Notes

The write gate is a client-go transport wrapper:

```go
// WriteGate returns a transport wrapper that blocks write requests whenever
// this elector is not leading.
func (le *LeaderElector) WriteGate() transport.WrapperFunc
```

### Health checks

Today, `LeaderElector.Check` reports failure when the elector believes it is the
leader but has not observed a renewal for a lease duration plus the configured
tolerance. This will need to be updated to report failure only when the election
loop has stopped making attempts.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

- Write gate:
  - writes rejected while closed
  - reads pass in both states
  - in-flight writes cancelled on close
  - writes that completed before close are unaffected
  - reopen after close
- Election loop:
  - gate closes and `OnStoppedLeading` fires at `RenewDeadline`
  - gate reopens on a later successful renewal
  - the `OnStartedLeading` context is cancelled only when `Run` returns
- Health check:
  - an inactive holder is healthy while attempts continue

##### Integration tests

- A gated client's write is rejected while inactive and succeeds after the gate reopens

##### e2e tests

- A controller manager continues to run (does not restart) across an API
  server interruption
- A gated client's write is rejected while inactive and succeeds after the gate reopens

### Graduation Criteria

#### Alpha

- Implement recovery mode behind `LeaderElectionRecovery`, disabled by default.
- Unit and integration tests above.

#### Beta

TODO

#### GA

TODO

#### Deprecation

Not applicable.

### Upgrade / Downgrade Strategy

Safe. Externally observed behavior is unchanged, just more efficient at recovery.

### Version Skew Strategy

None needed. The Lease API is unchanged.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `LeaderElectionRecovery`
  - Components depending on the feature gate: client-go, and controller
    managers that opt in to recovery mode.

The gate is alpha and disabled by default.

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

###### What happens if we reenable the feature if it was previously rolled back?

This is safe. There is no persisted state.

###### Are there any tests for feature enablement/disablement?

Unit tests will cover the gate on and off, with and without the opt-in.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A controller manager that leaves a client ungated can write while not leading.
That is caught by its tests and reverted by disabling recovery mode. Other
replicas and the API server are unaffected.

###### What specific metrics should inform a rollback?

If `leader_election_master_status` stays at 0 across all controller managers while the API
server is reachable, leader election is failing.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

We will add a write-gate rejection counter to client-go that can be observed.

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as last resort)
  - Details: After an API server or etcd interruption shorter than the recovery
    deadline, the controller manager does not restart and
    `leader_election_master_status` returns to 1 on the same replica.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Time to first successful reconciliation after an interruption. We will measure this in alpha
at different levels of scale.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric names: `leader_election_master_status` and a write-gate
    rejection counter (name TBD).
  - Components exposing the metric: controller managers

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Yes, the write-gate rejection counter that we plan to add.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

Lease API - GA.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Yes. An inactive holder retains its informer caches for up to the recovery deadline,
where today it would have exited.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The Proposal section above describes this in detail.

###### What are other known failure modes?

TODO

###### What steps should be taken if SLOs are not being met to determine the problem?

Check `leader_election_master_status` and the write-gate rejection counter.

## Implementation History

- 2026-09-14: Initial provisional proposal for KEP-6361.

## Drawbacks

TODO

## Alternatives

- **Continue exiting after renewal failure.** This remains the default, at the
  cost of a cold restart and cache rebuild after every interruption.
- **Increase lease durations.** This tolerates longer interruptions but slows
  failover from a failed holder and does not avoid the restart when an
  interruption exceeds the renew deadline.
- **Rely on context cancellation alone.** Ask controllers to stop on
  cancellation and trust them, without a gate. This makes safety depend on every
  controller loop honoring cancellation promptly, which process exit did not
  require. The write gate keeps that property.

## Infrastructure Needed (Optional)

None.
