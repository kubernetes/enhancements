# KEP-1965: kube-apiserver identity

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
- [Proposal](#proposal)
  - [Caveats](#caveats)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
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
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [Alternative 1: new API + storage TTL](#alternative-1-new-api--storage-ttl)
  - [Alternative 2: using storage interface directly](#alternative-2-using-storage-interface-directly)
  - [Alternative 3: storage interface + Lease API](#alternative-3-storage-interface--lease-api)
  - [Alternative 4: storage interface + new API](#alternative-4-storage-interface--new-api)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

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

## Proposal

We will use “hostname+PID+random suffix (e.g. 6 base58 digits)” as the ID.

Similar to the [node heartbeat](https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/0009-node-heartbeat.md),
a kube-apiserver will store its ID in a Lease object. All kube-apiserver Leases
will be stored in a special namespace “kube-apiserver-lease”. A controller will
garbage collect expired Leases.

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

The [kubelet heartbeat](https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/0009-node-heartbeat.md)
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

Each kube-apiserver with expose the following metrics for its own ID:

```go
  startTime := metrics.NewGauge(
		&metricsGaugeOpts{
			Namespace:      "apiserver",
			Subsystem:      "identity",
			Name:           "lease_start_time_seconds",
			Help:           "Lease start time seconds, labeled by the kube-apiserver ID in the Lease object, show the the time the Lease was created",
			StabilityLevel: metrics.ALPHA,
		},
  renewTime := metrics.NewGauge(
		&metricsGaugeOpts{
			Namespace:      "apiserver",
			Subsystem:      "identity",
			Name:           "lease_renew_time_seconds",
			Help:           "Lease rente time seconds, labeled by the kube-apiserver ID in the Lease object, show the time the Lease was updated",
			StabilityLevel: metrics.ALPHA,
		},
	success := metrics.NewCounter(
		&metrics.CounterOpts{
			Namespace:      "apiserver",
			Subsystem:      "identity",
			Name:           "lease_success_count",
			Help:           "Lease success count, labeled by the kube-apiserver ID in the Lease object, counts the number of success renewing a Lease",
			StabilityLevel: metrics.ALPHA,
		},
	)
  failures := metrics.NewCounter(
		&metrics.CounterOpts{
			Namespace:      "apiserver",
			Subsystem:      "identity",
			Name:           "lease_failure_count",
			Help:           "Lease failure count, labeled by the kube-apiserver ID in the Lease object, counts the number of failures to renew a Lease",
			StabilityLevel: metrics.ALPHA,
		},
```


The apiserver identity with be exposed using its own metric (ref. https://www.robustperception.io/exposing-the-software-version-to-prometheus)

```go
	buildInfo = metrics.NewGaugeVec(
		&metrics.GaugeOpts{
			Name:           "apiserver_identity",
			Help:           "A metric with a constant '1' value labeled by the kube-apiserver ID",
			StabilityLevel: metrics.ALPHA,
		},
		[]string{"id"},
	)
```

If users find complicated to obtain the apiserver ID using metrics, the information can be added
as a new endpoint, following the example of the `version` field, that is exposed both as an endpoint
`/version` and a metric `kubernetes_build_info`.

### Test Plan

  - integration test for creating the Namespace and the Lease on kube-apiserver
    startup
  - integration test for not creating the StorageVersions after creating the
    Lease
  - integration test for garbage collecting a Lease that isn't refreshed
  - integration test for not garbage collecting a Lease that is refreshed

### Graduation Criteria

Alpha should provide basic functionality covered with tests described above.

#### Alpha -> Beta Graduation

  - Appropriate metrics are agreed on and implemented
  - An e2e test plan is agreed and implemented (e.g. chaosmonkey in a regional
    cluster)

#### Beta -> GA Graduation

  - Conformance tests are agreed on and implemented

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

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: APIServerIdentity
    - Components depending on the feature gate: kube-apiserver

* **Does enabling the feature change any default behavior?**
  A namespace "kube-apiserver-lease" will be used to store kube-apiserver
  identity Leases.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes. Stale Lease objects will remain stale (`renewTime` won't get updated)

* **What happens if we reenable the feature if it was previously rolled back?**
  Stale Lease objects will be garbage collected.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

### Dependencies

_This section must be completed when targeting beta graduation to a release._

### Scalability

* **Will enabling / using this feature result in any new API calls?**
  Describe them, providing:
  - API call type (e.g. PATCH pods): UPDATE leases
  - estimated throughput:
  - originating component(s) (e.g. Kubelet, Feature-X-controller):
    kube-apiserver

  focusing mostly on:
  - components listing and/or watching resources they didn't before:
    kube-controller-manager
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.): kube-apiserver heartbeat every 10s

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
  Describe them, providing:
  - API type(s): leases
  - Estimated amount of new objects: one per living kube-apiserver

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  No.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- 2020-09-18: KEP introduced

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
