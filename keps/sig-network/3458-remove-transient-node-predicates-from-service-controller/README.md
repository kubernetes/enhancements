# KEP-3458: Remove transient node predicates from KCCM's service controller

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Risk](#risk)
    - [Mitigations](#mitigations)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
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

The service controller in the Kubernetes cloud controller manager (KCCM)
currently adds/removes Nodes from the load balancers' node set in the following
cases:

a) When a node gets the taint `ToBeDeletedByClusterAutoscaler` added/removed
b) When a node goes `Ready` / `NotReady`

b) however only applies to services with `externalTrafficPolicy: Cluster`. In
both cases: removing the Node in question from the load balancers' node set will
cause all connections on that node to get terminated instantly. This can be
considered a bug / sub-optimal behavior for nodes which are experiencing
transient readiness state or for terminating nodes, since connections are not
allowed to drain in those cases, even though the load balancer might support
that. Moreover: on large clusters with a lot nodes and entropy, re-syncing load
balancers like this can lead to rate-limiting by the cloud provider due to an
excessive amount of update calls.

As to enable connection draining, reduce cloud provider API calls and simplify
the KCCMs sync loop: this KEP proposes that the service controller stops
synchronizing the load balancer node set in these cases. Seeing as how this has
always been the case, a new feature gate `StableLoadBalancerNodeSet` will
be introduced, which will be used to enable the more optimal behavior.

## Motivation

Abruptly terminating connections in the cases defined by a) and b) above can be
seen as buggy behavior and should be improved. By enabling connection draining,
applications are allowed profit from graceful shutdown / termination, for what
concerns cluster ingress connectivity. Users of Kubernetes will also see a
reduction in the amount of cloud API calls, for what concerns calls stemming
from syncing load balancers with the Kubernetes cluster state.

Addressing b) is not useful for ingress load balancing. A load balancer needs to
know if the networking data plane is running fine and this is determined by the
configured health check. Cloud providers define their own health check, and no
one does the same. The following describes what the health check looks like on
the three major public cloud providers:

- GCP: probes port 10256 (Kube-proxy's healthz port)
- AWS: if ELB; probes the first `NodePort` defined on the service spec
- Azure: probes all `NodePort` defined on the service spec.
  
All clouds take an approach of trying to ascertain if traffic can be forwarded
to the endpoint, which is a completely valid health check for load balancer
services. There are drawbacks to all of these ways of doing - but cloud
providers themselves are deemed best suited for what concerns: determining what
is the best mechanism to use for their load balancers / cloud's mode of
operation. Their mechanism is beyond the scope of this KEP, i.e: this KEP does
not attempt to "align them".

### Goals

- Stop re-configuring the load balancers' node set for cases a) and b) above

### Non-Goals

- Stop re-configuring the load balancers' node set for fully deleted /
  newly added cluster nodes, or for nodes which get annotated with
  `node.kubernetes.io/exclude-from-external-load-balancers`.
- Enable load balancer connection draining while Node is draining. This requires
  health check changes.

## Proposal

### Risks and Mitigations

#### Risk

1. Cloud providers which do not allow VM deletion when the VM is referenced by
  other constructs, will block the cluster auto-scaler (CA) from deleting the VM
  upon downscale. This will result in reduced downscale performance by the CA,
  or completely block VM deletion from happening - this is because the service
  controller will never proceed to de-reference the VM from the load balancer
  node set until the Node is fully deleted in the API server, which will never
  occur until the VM is deleted. The three major cloud providers (GCP/AWS/Azure)
  do however support this, and it is not expected that other providers don't.
2. Cloud providers which do not configure their load balancer health checks to
  target the service proxy's healthz, alternatively: constructs which validate
  the endpoint's reachability across the data plane; risk experiencing
  regressions as a consequence of the removal of b). This would happen if a node
  is faced with a terminal error which does impact the Node's network
  connectivity. Doing this is considered incorrect, and therefor not expected to
  be the case.
3. By removing b) above we are delaying the removal of the Node from the load
  balancers' node set until the Node is completely deleted in the API server.
  This might have an impact on CA downscaling. The reason for this is: the CA
  deletes the VM and expects the node controller in the KCCM to notice this and
  delete the Node in Kubernetes, as a consequence. If the node controller takes
  a while to sync that and other Node related events trigger load balancer
  reconciliation while this is happening, then the service controller will error
  until the cluster reaches steady-state (because it's trying to sync Nodes for
  which the VM is non-existent). A mitigation to this is presented in
  [Mitigations](#mitigations)

#### Mitigations

- Cloud providers/workloads which do not support the behavior mentioned in
  [Risk](#risk), have the possibility to set the feature flag to false, thus
  default back to the current mechanism.
- For point 3. we could implement the following change in the service
  controller; ensure it only enqueues Node UPDATE on changes to
  `.metadata.labels["node.kubernetes.io/exclude-from-external-load-balancers"]`.
  When processing the sync: list only Node following the existing predicates
  defined for `externalTrafficPolicy: Cluster/Local` services (both currently
  exclude Nodes with the taint `ToBeDeletedByClusterAutoscaler`). This will
  ensure that whatever Nodes are included in the load balancer set, always have
  a corresponding VM. Doing this is however reverting on the goal of the KEP.

## Design Details

- Implement the change in the service controller and ensure it does not add /
  remove nodes from the load balancers' node set for cases a) and b) mentioned
  in (Summary)[#Summary]
- Add the feature gate: `StableLoadBalancerNodeSet`, set it to "on" by
  default and promote it directly to Beta.

### Test Plan

#### Prerequisite testing updates

#### Unit tests

The service controller in the KCCM currently has a set of tests validating
expected syncs caused by Node predicates, these will need to be updated.

- `k8s.io/cloud-provider/controllers/service`: `08/Feb/2023` - `67.7%`

#### Integration tests

Kubernetes is mostly tested via unit tests and e2e, not integration, and this is
not expected to change.

#### e2e tests

Kubernetes in general needs to extended its load balancing test suite with
disruption tests, this might be the right effort we need to get that ball
rolling. Testing would include:

- validation that an application running on a deleting VM benefits from graceful
  termination of its TCP connection.
- validation that Node readiness state changes do not result in load balancer
  re-syncs.

### Graduation Criteria

#### Beta

This is addressing a sub-optimal solution currently existing in Kubernetes, so
the feature gate will be moved to Beta and "on" by default from the start.

The feature flag should be kept available until we get sufficient evidence of
people not being affected by anything mentioned in (Risks)[#Risks] or other.

#### GA

Given the lack of reported issues in Beta: the feature gate will be locked-in in
GA.

Tentative timeline for this is in v1.29. Services of `type: LoadBalancer` are
sufficiently common on any given Kubernetes cluster, that any cloud provider
susceptible to the (Risks)[#Risks] will very likely report issues in Beta.

### Upgrade / Downgrade Strategy

Any upgrade to a version enabling the feature, succeeded by a downgrade to a
version disabling it, is not expected to be impacted in any way. On upgrade: the
service controller will add all existing cluster nodes (bar excluded ones) to
the load balancer set. On downgrade: any nodes `NotReady` / tainted will get
reconciled by the service controller corresponding to the downgraded control
plane version and get removed from the load balancer set - as they should.

### Version Skew Strategy

This change is contained to only the control plane and is therefor not
impacted by any version skew.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `StableLoadBalancerNodeSet`
  - Components depending on the feature gate: Kubernetes cloud controller manager

###### Does enabling the feature change any default behavior?

Yes, Kubernetes Nodes will remain in the load balancers' node set until fully
deleted in the API server, as opposed to the current behavior: which adds /
removes the nodes from the set when the Node experience transient state changes.
Cloud providers which do not support deleting VMs which are still referenced by
load balancers, will be unable to do so upon downscaling by the cluster
auto-scaler when it attempts to delete the VM.  

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, by resetting the feature gate back.

###### What happens if we reenable the feature if it was previously rolled back?

Behavior will be restored back immediately.

###### Are there any tests for feature enablement/disablement?

Not needed since the enablement/disablement is triggered by changing a in-memory boolean
variable.  

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

If a cluster has a lot of Nodes which are currently `NotReady` (in the order of
hundreds) and a rollout is triggered, it is expected that all of these nodes
will be added at once to every load balancer. That might have cloud API rate
limiting impacts on the service controller.

###### What specific metrics should inform a rollback?

Performance degradation by the CA when downscaling / flat out inability to
delete VMs. - this should be informed by the metric `nodesync_error_rate`
mentioned in [Monitoring requirements](#monitoring-requirements)

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Once the change is implemented: the author will work with Kubernetes vendors to
test the upgrade/downgrade scenario in a cloud environment.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

The only mechanism currently implemented, is: events for syncing load balancers
in the KCCM. The events are triggered any time a service is synced or Node
change triggers a re-sync of all services. This will not change and can be used
to monitor the implemented change. The implementation will result in less load
balancer re-syncs.

A new metric `load_balancer_sync_count` will be added for explicitly monitoring
the amount of load balancer related syncs performed by the service controller.
This will include load balancer syncs caused by Service and Node changes.

A new metric `nodesync_error_rate` will be added for explicitly monitoring the
amount of errors produced by syncing Node related events for load balancers. The
goal is have an indicator of if the service controller is impacted by point 3.
mentioned in (Risk)[#Risk], and at which frequency.

###### How can an operator determine if the feature is in use by workloads?

Analyze events stemming from the API server, correlating node state changes
(readiness or addition / removal of the taint: `ToBeDeletedByClusterAutoscaler`)
to load balancer re-syncs. The events should show a clear reduction in re-syncs
post the implementation and rollout of the change.

###### How can someone using this feature know that it is working for their instance?

- By observing no change for the metric `load_balancer_sync_count` when a Node
transitions between `Ready` <-> `NotReady` or when a Node is tainted with
`ToBeDeletedByClusterAutoscaler`. This is because this KEP proposes that we stop
syncing load balancer as a consequence of these events.

- By observing no change w.r.t any active ingress connections for an
  `externalTrafficPolicy: Cluster` service, which is passing through a Node
  which is transitioning between `Ready` <-> `NotReady`. I.e: no impact on new
  or established connections, given that Kube-proxy is healthy when the Node
  transitions state like this and isn't impacted.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

No

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Total amount of load balancer re-syncs should be reduced, leading to less cloud
provider API calls. Also, and more subtle: connections will get a chance to
gracefully terminate when the CA downscales cluster nodes. For services of type
`externalTrafficPolicy: Cluster` "traversing" connections through a "nexthop"
node might not be impacted by that Node's readiness state anymore.

- [X] Metrics
  - Events: The KCCM triggers events when syncing load balancers. The amount of
    these events should be reduced.
  - Metrics: `load_balancer_sync_count`

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

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

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Not any different than today.

###### What are other known failure modes?

None

###### What steps should be taken if SLOs are not being met to determine the problem?

Validate that services of `type: LoadBalancer` exists on the cluster and that
Nodes are experiencing a transitioning readiness state, alternatively that the
CA downscales and deletes VMs.

## Implementation History

- 2023-02-01: Initial proposal

## Drawbacks

## Alternatives
