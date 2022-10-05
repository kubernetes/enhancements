# KEP-1965: kube-apiserver identity

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Caveats](#caveats)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [Alternative 1: new API + storage TTL](#alternative-1-new-api--storage-ttl)
  - [Alternative 2: using storage interface directly](#alternative-2-using-storage-interface-directly)
  - [Alternative 3: storage interface + Lease API](#alternative-3-storage-interface--lease-api)
  - [Alternative 4: storage interface + new API](#alternative-4-storage-interface--new-api)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
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

In a HA cluster, each kube-apiserver has an ID. Controllers have access to the
list of IDs for living kube-apiservers in the cluster.

## Motivation

The [dynamic coordinated storage version API](https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/20190802-dynamic-coordinated-storage-version.md#curating-a-list-of-participating-api-servers-in-ha-master)
needs such a list to garbage collect stale records. The
[API priority and fairness feature](https://github.com/kubernetes/kubernetes/pull/91389)
needs a unique identifier for an apiserver reporting its concurrency limit.

Currently, such a list is already maintained in the “kubernetes” endpoints,
where the kube-apiservers’ advertised IP addresses are the IDs. However it is
not working in all flavors of Kubernetes deployments. For example, if there is a
load balancer for the cluster, where the advertise IP address is set to the IP
address of the load balancer, all three kube-apiservers will have the same
advertise IP address.

### Goals

* Provide a mechanism in which controllers can uniquely identify kube-apiserver's in a cluster.

### Non-Goals

* improving the availability of kube-apiserver

## Proposal

We will use “hostname+PID+random suffix (e.g. 6 base58 digits)” as the ID.

Similar to the [node heartbeats](https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/589-efficient-node-heartbeats),
a kube-apiserver will store its ID in a Lease object. All kube-apiserver Leases
will be stored in a special namespace `kube-apiserver-lease`. The Lease creation
and heart beat will be managed by a controller that is started in kube-apiserver's
post startup hook. A separate controller in kube-controller-manager will be responsible
for garbaging collecting expired Leases.

### Caveats

In this proposal we focus on kube-apiservers. Aggregated apiservers don’t have
the same problem, because their record is already exposed via the service. By
listing the pods selected by the service, an aggregated server can learn the
list of living servers with distinct podIPs. A server can get its own IDs via
downward API.

We prefer that expired Leases remain for a longer duration as opposed to
collecting them quickly, because in the latter case, if a Lease is falsely
collected by accident, it can do more damage than the former case. Take the
storage version API scenario as an example, if a kube-apiserver accidentally
missed a heartbeat and got its Lease garbage collected, its StorageVersion can
be falsely garbage collected as a consequence. In this case, the storage
migrator won’t be able to migrate the storage, unless this kube-aipserver gets
restarted and re-registers its StorageVersion. On the other hand, if a
kube-apiserver is gone and its Lease still stays around for an hour or two, it
will only delay the storage migration for the same period of time.

## Design Details

The [kubelet heartbeat](https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/589-efficient-node-heartbeats)
logic [already written](https://github.com/kubernetes/kubernetes/tree/master/pkg/kubelet/nodelease)
will be re-used. The heartbeat controller will be added to kube-apiserver in a
post-start hook.

Each kube-apiserver will refresh its Lease every 10s by default. A GC controller
will watch the Lease API using an informer, and periodically resync its local
cache. On processing an item, the controller will delete the Lease if the last
`renewTime` was more than `leaseDurationSeconds` ago (default to 1h). The
default `leaseDurationSeconds` is chosen to be way longer than the default
refresh period, to tolerate clock skew and/or accidental refresh failure. The
default resync period is 1h. By default, assuming negligible clock skew, a Lease
will be deleted if the kube-apiserver fails to refresh its Lease for one to two
hours. The GC controller will run in kube-controller-manager, to leverage leader
election and reduce conflicts.

The refresh rate, lease duration will be configurable through kube-apiserver
flags. The resync period will be configurable through a kube-controller-manager
flag.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `staging/src/k8s.io/apiserver/pkg/endpoints`

##### Integration tests

[apiserver_identity_test.go](https://github.com/kubernetes/kubernetes/blob/24238425492227fdbb55c687fd4e94c8b58c1ee3/test/integration/controlplane/apiserver_identity_test.go)
- integration test for creating the Namespace and the Lease on kube-apiserver startup
- integration test for not creating the StorageVersions after creating the Lease
- integration test for garbage collecting a Lease that isn't refreshed
- integration test for not garbage collecting a Lease that is refreshed

##### e2e tests

Proposed e2e tests:
- an e2e test that validates the existence of the Lease objects per kube-apiserver
- an e2e test that restarts a kube-apiserver and validates that a new Lease is created
  with a newly generated ID and the old lease is garbage collected

### Graduation Criteria

Alpha should provide basic functionality covered with tests described above.

#### Alpha -> Beta Graduation

  - Appropriate metrics are agreed on and implemented
  - Sufficient integration tests covering basic functionality of this enhancement.
  - e2e tests outlined in the test plan are implemented

#### Beta -> GA Graduation

N/A

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

### Version Skew Strategy

  - This feature is proposed for the control plane internal use. Master-node skew is
    not considered.
  - During a rolling update, an HA cluster may have old and new masters. Old masters
    won't create Leases, nor garbage collect Leases.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: APIServerIdentity
  - Components depending on the feature gate: kube-apiserver

###### Does enabling the feature change any default behavior?

A namespace "kube-apiserver-lease" will be used to store kube-apiserver identity Leases.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Stale Lease objects will remain stale (renewTime won't get updated)

###### What happens if we reenable the feature if it was previously rolled back?

Stale Lease objects will be garbage collected.

###### Are there any tests for feature enablement/disablement?

Yes, see [apiserver_identity_test.go](https://github.com/kubernetes/kubernetes/blob/24238425492227fdbb55c687fd4e94c8b58c1ee3/test/integration/controlplane/apiserver_identity_test.go).

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Existing workloads should not be impacteded by this feature, unless they were
looking for Lease objects in the `kube-apiserver-lease` namespace.

###### What specific metrics should inform a rollback?

Recently added [healthcheck metrics for apiserver](https://github.com/kubernetes/kubernetes/pull/112741), which includes
the health of the post start hook can be used to inform rollback, specifically `kubernetes_healthcheck{poststarthook/start-kube-apiserver-identity-lease-controller}`

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Manual testing for upgrade/rollback will be done prior to Beta. Steps taken for manual tests will be updated here.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The existence of the `kube-apiserver-lease` namespace and Lease objects in the namespace
will determine if the feature is working. Operators can check for clients that are accessing
the Lease object to see if workloads or other controllers are relying on this feature.

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason:
- [X] API .status
  - Condition name:
  - Other field:
- [X] Other (treat as last resort)
  - Details: audit logs for clients that are reading the Lease objects

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

A rough SLO here is that kube-apiserver updates leases at the same frequency as kubelet node heart beats,
since the same mechanism is being used.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - Metric name: kubernetes_healthcheck
  - [Optional] Aggregation method: name="poststarthook/start-kube-apiserver-identity-lease-controller"
  - Components exposing the metric: kube-apiserver

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Yes, heart beat latency could be useful.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes, kube-apiserver will be making new API calls as part of the lease controller.

###### Will enabling / using this feature result in introducing new API types?

No, the feature will use the existing Lease API.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, it will increase the number of Leases in a cluster by the number of control plane VMs.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

The lease controller may use additional resources in kube-apiserver, but it is likely negligible.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Lease objects for a given kube-apiserver may become stale if the kube-apiserver or etcd is non-responsive. Clients should
be able to respond accordingly by checking the lease expiration.

###### What are other known failure modes?

* lease objects can become stale if etcd is unavailable and clients do not check lease expiration.
* kube-apiserver heart beats consuming too many resources (unlikely but possible)

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 2020-09-18: KEP introduced
- 2022-10-05: KEP updated with Beta criteria and all PRR questions answered.

## Alternatives

### Alternative 1: new API + storage TTL

We define a new API for kube-apiserver identity. Similar to [Event](https://github.com/kubernetes/kubernetes/blob/9062c43b76c8562062e454a190a948f1370f8eb3/pkg/registry/core/rest/storage_core.go#L128),
we make the storage path for the new object type [tack on the TTL](https://github.com/kubernetes/kubernetes/blob/9062c43b76c8562062e454a190a948f1370f8eb3/staging/src/k8s.io/apiserver/pkg/registry/generic/registry/store.go#L1173).
Etcd will delete objects who don’t get their TTL refreshed in time.

  - Pros:
    - We don’t need to write a controller to garbage collect expired records, nor
    worry about client-server clock skew.
    - We can extend the API in future to include more information (e.g. version,
    feature, config)
  - Cons:
    - We need a new dedicated API

Note that the proposed solution doesn't prevent us from switching to a new API
in future. Similar to node heartbeats switched from node status to leases.

### Alternative 2: using storage interface directly

The existing “kubernetes” Endpoints [mechanism](https://github.com/kubernetes/community/pull/939)
can be inherited to solve the kube-apiserver identity problem. There are two
parts of the mechanism:
  1. Each kube-apiserver periodically writes a lease of its ID (address) with a
  TTL to etcd through the storage interface. The lease object itself is an
  Endpoints. Leases will be deleted by etcd for servers who fail to refresh the
  TTL in time.
  2. A controller reads the leases through the storage interface, to collect the
  list of IP addresses. The controller updates the “kubernetes” Endpoints to
  match the IP address list.

We inherit the first part of the existing mechanism (the etcd TTL lease), but
change the key and value. The key will be the new ID. All the keys will be
stored under a special prefix “/apiserverleases/” (similar to the [existing mechanism](https://github.com/kubernetes/kubernetes/blob/14a11060a0775ed609f0810898ebdbe737c59441/pkg/master/master.go#L265)).
The value will be a Lease object. A kube-apiserver obtains the list of IDs by
directly listing/watching the leases through the storage interface.

  - Cons:
    - We depend on a side-channel API, which is against Kubernetes philosophy
    - Clients like the kube-controller-manager cannot access the storage
    interface. For the storage version API, if we put the garbage collector in
    kube-apiserver instead of kube-controller-manager, the lack of leader
    election may cause update conflicts.

### Alternative 3: storage interface + Lease API

The kube-apiservers still write the master leases to etcd, but a controller will
watch the master leases and update an existing public API (e.g. store it in a
defined way in a Lease). Note that we cannot use the endpoints API like the
“kubernetes” endpoints, because the endpoints API is designed to store a list of
addresses, but our IDs are not IP addresses.

  - Cons:
    - We depend on a side-channel API, which is against Kubernetes philosophy

### Alternative 4: storage interface + new API

Similar to Alternative 1, the kube-apiservers write the master leases to etcd,
and a controller watches the master leases, but updates a new public API
specifically designed to host information about the API servers, including its
ID, enabled feature gates, etc.

  - Cons:
    - We depend on a side-channel API, which is against Kubernetes philosophy
