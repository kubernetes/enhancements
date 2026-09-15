# KEP-6371: External Node Liveness Detection

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Node lifecycle controller: <code>--node-liveness-source</code>](#node-lifecycle-controller---node-liveness-source)
  - [External liveness detector contract](#external-liveness-detector-contract)
  - [Kubelet: <code>enableNodeLease</code>](#kubelet-enablenodelease)
  - [User Stories (Optional)](#user-stories-optional)
    - [An operator with a custom liveness detector](#an-operator-with-a-custom-liveness-detector)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Controller changes](#controller-changes)
  - [Kubelet changes](#kubelet-changes)
  - [Feature gate](#feature-gate)
  - [Metrics](#metrics)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure that all e2e tests are run in Kubernetes CI and are not skipped
  - [ ] (R) Minimum Two Week Soak in Kubernetes CI - 'This is required for all new features but some may have a longer period. Discussion for a longer period should be done with the CI Signal lead for the release.'
  - [ ] (R) The following are in place for all tests
    - [ ] Tests have proper linters and tools used to detect flaky tests
    - [ ] Tests are consistently passing
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md)
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

This KEP makes node liveness detection replaceable with a custom implementation.
A cluster operator can turn off the detector built into the node lifecycle
controller, disable kubelet Lease heartbeats, and supply an external detector
that writes `Ready=Unknown`. The node lifecycle controller retains
responsibility for all other lifecycle management, such as eviction and zone
disruption handling.

This is implemented by a `--node-liveness-source=external` flag on the
kube-controller-manager, which stops the controller from checking liveness and
changes how it reacts to `Ready=Unknown` when the condition is written by an
external node liveness detector. An `enableNodeLease: false` field is added to
the kubelet for configurations that do not need node lease heartbeats.

## Motivation

Node lease writes can be expensive. At scale, they are the largest single class
of writes to the API server (~29% of all writes in a 5000-node scalability run),
and the cost grows with every node added.

In many environments, external systems can detect whether a node is healthy,
and in some cases they do so faster than node leases.

Today, the node lifecycle controller does not cooperate well with external
sources of node liveness. If an external system writes `Ready=Unknown`, the node
lifecycle controller taints and evicts the node but never marks the node's pods
NotReady. This is because `monitorNodeHealth` compares the Ready condition
before and after its own update within a single pass, so a transition written by
anyone else is never seen as a transition.

This gap has been reported repeatedly (kubernetes/kubernetes#125618, #112733,
#135205) and is the reason forks of the controller exist (for example,
OpenYurt).

Lastly, today there is no way to disable kubelet heartbeats. Every kubelet
renews its Lease every 10s. This makes it difficult to introduce an alternative
node liveness mechanism, because there is no good way to disable the built-in
one.

### Goals

- Provide a way for a cluster operator to declare that node liveness is handled
  by a custom mechanism.
- Define the contract an external writer must follow.
- Let an operator turn off the kubelet's node Lease heartbeat.

### Non-Goals

- Changing default behavior.
- Shipping an external liveness component.
- Changing what the kubelet writes in node status.
- Changing how `Ready=False` is handled (usually written by the kubelet). Pods
  are still marked `NotReady` only on transitions to `Unknown`
  (kubernetes/kubernetes#125618).

## Proposal

### Node lifecycle controller: `--node-liveness-source`

`--node-liveness-source` is a new kube-controller-manager option with two
values: `kubelet` (default, today's behavior) and `external`. It is backed by
`NodeLifecycleControllerConfiguration.NodeLivenessSource`.

In `external` mode the controller:

- Does not mark Ready, MemoryPressure, DiskPressure, or PIDPressure Unknown when
  `--node-monitor-grace-period` expires. `--node-startup-grace-period` still
  applies to nodes that have never posted a Ready condition, since an external
  writer has nothing to observe for them.
- Detects Ready transitions. It tracks the Ready condition it observed in the
  previous pass. A node whose Ready condition transitions to Unknown is handled
  the same way a controller-detected failure is handled today: a NodeNotReady
  event is recorded and `MarkPodsNotReady` is called for the node's pods.
- On controller restart or leader failover, treats a node whose Ready condition
  is not True as needing `MarkPodsNotReady`.

When a kubelet recovers, it continues to write `Ready=True` as it does today,
and the controller treats that as recovery.

### External liveness detector contract

An external liveness detector:

- Is responsible for ensuring that a failure signal persists for some interval
  before writing Unknown. Today the node lifecycle controller fetches the node's
  Lease live from the API server and, if it is newer than the cached one, aborts
  and waits for the informer cache to catch up. By taking over node liveness, an
  external writer takes responsibility for safety measures of this kind.
- To record a node failure, sets the node's `Ready` condition to `Unknown` with
  any reason other than `NodeStatusUnknown` and sets `lastTransitionTime`, but
  does not modify `lastHeartbeatTime`.
- Treats the `Ready=True` condition as an indication that the kubelet is
  recovering and is careful not to interfere with the core controllers' handling
  of that condition. To avoid flapping, it does not overwrite a Ready condition
  that is newer than its own last write.

Note that external writers must be granted write permission on `nodes/status`.

### Kubelet: `enableNodeLease`

`enableNodeLease` is a new optional `KubeletConfiguration` field that defaults
to `true`. When `false`, the kubelet does not create or renew its node Lease.

`nodeLeaseDurationSeconds` continues to bound the heartbeat client timeout and
remains validated as positive.

Node status reporting is not affected.

### User Stories (Optional)

#### An operator with a custom liveness detector

An operator runs an agent that decides node liveness from out-of-band signals.
They set `--node-liveness-source=external` on the kube-controller-manager, then
set `enableNodeLease: false` on the kubelets. When the agent detects a node
failure, it writes `Ready=Unknown`. The core controllers handle the rest of the
node lifecycle, including taints and eviction.

### Notes/Constraints/Caveats (Optional)

The two settings must be enabled in order: the kube-controller-manager first,
then the kubelets. See [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy).

### Risks and Mitigations

- **Ecosystem tools consuming node Leases.** Gardener, Deckhouse fencing,
  node-undertaker, kwatch, kube-state-metrics, `kubectl describe node`, and the
  NodeConformance test "kubelet should create and update a lease" all assume the
  Lease exists. See [Drawbacks](#drawbacks).
- **The external liveness detector fails.** This is a fundamental risk of this
  design. It is the responsibility of the external liveness detector and the
  cluster operator to ensure that the external system is robust and reliable.

## Design Details

### Controller changes

All changes are in `pkg/controller/nodelifecycle/node_lifecycle_controller.go`.

- **Ready condition:** The controller keeps a `nodeHealthData` record for each
  node. We will add a `lastReady` field that holds the Ready condition seen on
  the previous pass, and update it on every pass. The existing `status` field
  cannot be used for this. It is only updated when `lastHeartbeatTime` changes,
  and external detectors do not change `lastHeartbeatTime`.
- **External change detection:** Today, `tryUpdateNodeHealth` copies the
  Ready condition at the start of a pass and compares against that copy at the
  end. Only the controller's own writes show up as a change. In `external` mode,
  compare against `lastReady` instead. A change made by an external detector
  between passes now shows up as a change.
- **Timeouts:** In `external` mode, skip the block that marks a node
  Unknown when its `probeTimestamp` is older than the grace period. Keep running
  it for nodes that have no Ready condition at all, so the startup grace period
  still works.
- **Update `monitorNodeHealth`:** Today, `monitorNodeHealth` records a
  NodeNotReady event and calls `MarkPodsNotReady` when Ready goes from True to
  anything else during a pass. In `external` mode, also do this when Ready has
  become Unknown since the previous pass, from either True or False. On the
  first pass after a restart or leader change there is no `lastReady`. If Ready
  is not True on that pass, add the node to `nodesToRetry` so its pods are
  marked NotReady.
- **Update `processPod`:** `processPod` marks a pod NotReady when it is assigned
  to a node that is already not Ready. It reads the node's Ready condition from
  `status`. Read it from `lastReady` instead, for the same reason as above.

The controller still watches Leases, but ignores them for liveness in `external`
mode.

### Kubelet changes

The kubelet constructs and runs its lease controller only when `enableNodeLease`
is true.

### Feature gate

Both settings are opt-in and controlled by the cluster administrator. Feature
gates are unnecessary.

### Metrics

A new gauge, `node_collector_liveness_source{source="kubelet"|"external"}`, is
set to 1 for the configured source.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

Node lifecycle controller in `external` mode (`k8s.io/kubernetes/pkg/controller/nodelifecycle`):

- A node that stops heartbeating is not marked Unknown.
- When a node's Ready condition becomes Unknown, its pods are marked NotReady and
  one NodeNotReady event is recorded. This holds whether the previous value was
  True or False, and whether or not `lastHeartbeatTime` changed.
- When a node's Ready condition becomes False, nothing happens (unchanged from
  today).
- When a node's Ready condition returns to True, the node recovers.
- After a controller restart, pods on nodes that are not Ready are marked
  NotReady.
- A node that never reported a Ready condition is marked Unknown after the
  startup grace period.

Node lifecycle controller in `kubelet` mode: behavior is unchanged.

Kubelet (`k8s.io/kubernetes/pkg/kubelet`): with `enableNodeLease: false`, no
Lease is created or renewed.

##### Integration tests

Controller in `external` mode (`test/integration/node/lifecycle_test.go`):

- A test writer sets `Ready=Unknown` on one node. Within one monitor period, that
  node's pods lose readiness and the `unreachable` taint appears. Other nodes are
  unaffected.
- The kubelet then writes `Ready=True`. The node recovers, and exactly one
  NodeNotReady event was recorded.

Wrong rollout order: with leases disabled and the controller still in `kubelet`
mode, nodes flap between Unknown and Ready as described in
[Version Skew Strategy](#version-skew-strategy).

##### e2e tests

None for alpha.

### Graduation Criteria

#### Alpha

- Unit and integration tests above.

#### Beta

- At least one external liveness detector outside the test suite reported in
  use.
- An e2e test with a test writer, running in CI.
- The NodeConformance lease test is skipped in jobs that disable leases.
- Upgrade, downgrade, and rollback tested.
- A metric that identifies nodes whose Ready condition has not been written for
  several status report periods, so that a stalled external detector is visible.
- Decision on per-node opt-in.

#### GA

- Two releases at beta with no reported defects.
- Documentation on kubernetes.io for the flag, the field, and the external
  detector contract.

### Upgrade / Downgrade Strategy

To enable, set `--node-liveness-source=external` on every kube-controller-manager
replica first, then set `enableNodeLease: false` on the kubelets. The controller
must change first because leader election can move the controller to any
replica, and a replica still in `kubelet` mode will treat lease-less nodes as
dead.

To disable, reverse the order: re-enable kubelet leases first, then set every
kube-controller-manager replica back to `kubelet` mode.

A kube-controller-manager older than 1.38 rejects `--node-liveness-source` as an
unknown flag and fails to start, so the flag must be removed before downgrading.
A kubelet older than 1.38 fails strict decoding of `enableNodeLease`, logs a
warning, falls back to lenient decoding, and heartbeats as it does today, so
kubelet leases must be re-enabled before downgrading kubelets.

### Version Skew Strategy

The only unsafe combination is a kubelet with `enableNodeLease: false` running
against a kube-controller-manager in `kubelet` mode. The controller sees no
Lease and only the 5-minute status report, so it marks the node Unknown after
`--node-monitor-grace-period` (50s by default), marks its pods NotReady, and
starts eviction. The live kubelet then re-asserts Ready=True within
`--node-status-update-frequency` (10s by default), the controller treats that
as recovery, and the cycle repeats roughly once a minute for every such node.

Because `kubelet` mode is the default, and because kube-controller-managers
older than 1.38 have no other mode, kubelet leases must never be disabled until
every kube-controller-manager replica is running 1.38 or later in `external`
mode. The two components do not otherwise interact beyond the Node and Lease
objects they already use.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [ ] Feature gate
- [x] Other
  - Describe the mechanism: `--node-liveness-source=external` on
    kube-controller-manager and `enableNodeLease: false` in KubeletConfiguration.
    Both are opt-in and default to today's behavior.
  - Will enabling / disabling the feature require downtime of the control plane?
    No. Changing the flag requires a kube-controller-manager restart, which is a
    rolling restart in HA deployments.
  - Will enabling / disabling the feature require downtime or reprovisioning of
    a node? Requires a kubelet restart.

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Turn kubelet leases back on, then set every kube-controller-manager back to `kubelet` mode.

###### What happens if we reenable the feature if it was previously rolled back?

The settings take effect again. There is no persisted state.

###### Are there any tests for feature enablement/disablement?

Yes. The controller unit and integration tests cover both `kubelet` and
`external` modes, and the kubelet unit tests cover `enableNodeLease` set to
`true` and `false`.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

To be completed for beta. See [Version Skew Strategy](#version-skew-strategy)
for the known failure mode.

### Monitoring Requirements

To be completed for beta.

### Dependencies

To be completed for beta.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. It only reduces the number of writes to the API server when the kubelet Lease is disabled.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### What are other known failure modes?

To be completed for beta.

## Implementation History

- 2026-09-15: KEP opened, targeting alpha in v1.38.

## Drawbacks

- In `external` mode, failure detection is only as good as the writer, and Kubernetes cannot verify
  that one exists.
- The mode is cluster-wide. Node pools with different liveness sources cannot be expressed.
- `enableNodeLease: false` breaks tools that read the kubelet Lease: Gardener's node health checks,
  Deckhouse's fencing controller, node-undertaker, kwatch, kube-state-metrics lease metrics,
  `kubectl describe node`, and the NodeConformance lease test.
- On managed control planes the option is available only if the provider exposes it.

## Alternatives

**Use the existing knobs.** Set `nodeLeaseDurationSeconds` and `--node-monitor-grace-period` very
large and write Ready=Unknown from outside. The controller still never marks pods NotReady for a
transition it did not write, and the Lease is still written.

## Infrastructure Needed (Optional)

None.
