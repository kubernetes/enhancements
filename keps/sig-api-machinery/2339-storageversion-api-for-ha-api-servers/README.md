# KEP-2339: StorageVersion API for HA API servers

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Resource Version API](#resource-version-api)
- [Changes to API servers](#changes-to-api-servers)
  - [Curating a list of participating API servers in HA master](#curating-a-list-of-participating-api-servers-in-ha-master)
  - [Updating StorageVersion](#updating-storageversion)
  - [Garbage collection](#garbage-collection)
  - [CRDs](#crds)
  - [Aggregated API servers](#aggregated-api-servers)
- [Consuming the StorageVersion API](#consuming-the-storageversion-api)
- [StorageVersion API vs. StorageVersionHash in the discovery document](#storageversion-api-vs-storageversionhash-in-the-discovery-document)
- [Backwards Compatibility](#backwards-compatibility)
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
  - [Letting API servers vote on the storage version](#letting-api-servers-vote-on-the-storage-version)
  - [Letting the storage migrator detect if API server instances are in agreement](#letting-the-storage-migrator-detect-if-api-server-instances-are-in-agreement)
- [Appendix](#appendix)
  - [Accuracy of the discovery document of CRDs](#accuracy-of-the-discovery-document-of-crds)
- [References](#references)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

During the rolling upgrade of an HA master, the API server instances may
use different storage versions encoding a resource. The [storageVersionHash]
in the discovery document does not expose this disagreement. As a result, the
storage migrator may proceed with migration with the false belief that all API
server instances are encoding objects using the same storage version, resulting
in polluted migration.  ([details]).

[storageVersionHash]:https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/types.go#L979
[details]:https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/35-storage-version-hash.md#ha-masters

## Motivation

The lack of automated storage migration creates unfixable technical debt in the
Kubernetes project as we are unable to drop old versions of APIs, even after the
associated REST endpoints have been disabled.  This also blocks more nuanced
use cases of storage migration such as encryption key rotation, which also needs
a reliable mechanism for understanding when API servers are in agreement.

### Goals

- Create an API that allows an external controller to observe what storage versions are in use by API servers in an HA setup

### Non-Goals

- Automatically running storage migration

## Proposal

We propose a way to show what storage versions all API servers are using, so
that the storage migrator can defer migration until an agreement has been
reached.

### Risks and Mitigations

Writes to most Kubernetes resources must be prevented until the API server has
had a chance to emit its encoding versions for all resources.  This requires us to
have a carefully written handler that only lets a very specific set of requests through.
Mistakes in this code could easily prevent the API server from functioning.  This
handler will be carefully tested via unit and integration tests.

## Design Details

### Resource Version API

We introduce a new API `StorageVersion`, in a new API group
`internal.apiserver.k8s.io/v1alpha1`.

```go
// Storage version of a specific resource.
type StorageVersion struct {
  metav1.TypeMeta `json:",inline"`
  // The name is <group>.<resource>.
  metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

  // Spec is an empty spec. It is here to comply with Kubernetes API style.
  Spec StorageVersionSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`

  // API server instances report the version they can decode and the version they
  // encode objects to when persisting objects in the backend.
  Status StorageVersionStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

// StorageVersionSpec is an empty spec.
type StorageVersionSpec struct{}

// API server instances report the versions they can decode and the version they
// encode objects to when persisting objects in the backend.
type StorageVersionStatus struct {
  // The reported versions per API server instance.
  // +optional
  // +listType=map
  // +listMapKey=apiServerID
  StorageVersions []ServerStorageVersion `json:"storageVersions,omitempty" protobuf:"bytes,1,opt,name=storageVersions"`
  // If all API server instances agree on the same encoding storage version,
  // then this field is set to that version. Otherwise this field is left empty.
  // API servers should finish updating its storageVersionStatus entry before
  // serving write operations, so that this field will be in sync with the reality.
  // +optional
  CommonEncodingVersion *string `json:"commonEncodingVersion,omitempty" protobuf:"bytes,2,opt,name=commonEncodingVersion"`

  // The latest available observations of the storageVersion's state.
  // +optional
  // +listType=map
  // +listMapKey=type
  Conditions []StorageVersionCondition `json:"conditions,omitempty" protobuf:"bytes,3,opt,name=conditions"`
}

// An API server instance reports the version it can decode and the version it
// encodes objects to when persisting objects in the backend.
type ServerStorageVersion struct {
  // The ID of the reporting API server.
  APIServerID string `json:"apiServerID,omitempty" protobuf:"bytes,1,opt,name=apiServerID"`

  // The API server encodes the object to this version when persisting it in
  // the backend (e.g., etcd).
  EncodingVersion string `json:"encodingVersion,omitempty" protobuf:"bytes,2,opt,name=encodingVersion"`

  // The API server can decode objects encoded in these versions.
  // The encodingVersion must be included in the decodableVersions.
  // +listType=set
  DecodableVersions []string `json:"decodableVersions,omitempty" protobuf:"bytes,3,opt,name=decodableVersions"`
}

type StorageVersionConditionType string

const (
  // Indicates that encoding storage versions reported by all servers are equal.
  AllEncodingVersionsEqual StorageVersionConditionType = "AllEncodingVersionsEqual"
)

type ConditionStatus string

const (
  ConditionTrue    ConditionStatus = "True"
  ConditionFalse   ConditionStatus = "False"
  ConditionUnknown ConditionStatus = "Unknown"
)

// Describes the state of the storageVersion at a certain point.
type StorageVersionCondition struct {
  // Type of the condition.
  // +required
  Type StorageVersionConditionType `json:"type" protobuf:"bytes,1,opt,name=type"`
  // Status of the condition, one of True, False, Unknown.
  // +required
  Status ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status"`
  // If set, this represents the .metadata.generation that the condition was set based upon.
  // +optional
  ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,3,opt,name=observedGeneration"`
  // Last time the condition transitioned from one status to another.
  // +required
  LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,4,opt,name=lastTransitionTime"`
  // The reason for the condition's last transition.
  // +required
  Reason string `json:"reason" protobuf:"bytes,5,opt,name=reason"`
  // A human readable message indicating details about the transition.
  // +required
  Message string `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
}
```

## Changes to API servers

In this section, we describe how to update and consume the StorageVersion API.

### Curating a list of participating API servers in HA master

API servers need such a list when updating the StorageVersion API. Currently,
such a list is already maintained in the "kubernetes" endpoints, though it does
not work in all flavors of Kubernetes deployments.

We will inherit the existing [mechanism], but formalize the API and process in
another KEP. In this KEP, we assume all API servers have access to the list of
all participating API servers via some API.

[mechanism]:https://github.com/kubernetes/community/pull/939

### Updating StorageVersion

During bootstrap, for each resource, the API server
* gets the storageVersion object for this resource, or creates one if it does
  not exist yet,
* gets the list of participating API servers,
* updates the storageVersion locally. Specifically,
  * creates or updates the .status.serverStorageVersions, to express this API
    server's decodableVersions and encodingVersion.
  * removes .status.serverStorageVersions entries whose server ID is not present
    in the list of participating API servers, such entries are stale.
  * checks if all participating API servers agree on the same storage version.
    If so, sets the version as the status.agreedEncodingVersion. If not, sets
    the status.agreedEncodingVersion to empty. The "AllEncodingVersionsEqual"
    status.condition is updated accordingly as well.
* updates the storageVersion object, using the rv in the first step
  to avoid conflicting with other API servers.
* installs the resource handler.

The above mentioned process requires an API server to update the storageVersion
before accepting API requests. If we don't enforce this order, data encoded in
an unexpected version can sneak into etcd. For example, an API server persists a
write request encoded in an obsoleted version, then it crashes before it can
update the storageVersion. The storage migrator has no way to detect this write.

For the cmd/kube-apiserver binary, we plan to enforce this order by adding a new
filter to the [handler chain]. Before kube-aggregator, kube-apiserver, and
apiextension-apiserver have registered the storage version of the built-in
resources they host, this filter only allows the following requests to pass:
1. a request sent by the loopbackClient and is destined to the storageVersion
   API.
2. the verb of the request is GET.
3. the request is for an API that is not persisted, e.g.,
   SubjectAccessReview and TokenReview. [Here] is a complete list.
4. the request is for an aggregated API, because the request is handled by the
   aggregated API server.
5. the request is for a custom resource, because the apiextension apiserver
   makes sure that it updates the storage version before it serves the CR (see
   [CRDs](#crds)).

The filter rejects other requests with a 503 Service Unavailable response code.

[handler chain]:https://github.com/kubernetes/kubernetes/blob/fc8f5a64106c30c50ee2bbcd1d35e6cd05f63b00/staging/src/k8s.io/apiserver/pkg/server/config.go#L639
[Here]:https://github.com/kubernetes/kubernetes/blob/709a0c4f7bfec2063cb856f3cdf4105ce984e247/pkg/master/storageversionhashdata/data.go#L26

One concern is that the storageVersion API becomes a single-point-of-failure,
though it seems inevitable in order to ensure the correctness of the storage
migration.

We will also add a post-start hook to ensure that the API server reports not
ready until the storageVersions are up-to-date and the filter is turned off.

### Garbage collection

There are two kinds of "garbage":

1. stale storageVersion.status.serverStorageVersions entries left by API servers
   that have gone away;
2. storageVersion objects for resources that are no longer served.

We can't rely on API servers to remove the first kind of stale entries during
bootstrap, because an API server can go away after other API servers bootstrap,
then its stale entries will remain in the system until one of the other API
servers reboots.

Hence, we propose a leader-elected control loop in API server to clean up the
stale entries, and in turn clean up the obsolete storageVersion objects. The
control loop watches the list of participating API servers, upon changes, it
performs the following actions for each storageVersion object:

* gets a storageVersion object
* gets the list of participating API servers,
* locally, removes the stale entries (1st kind of garbage) in
  storageVersion.status.serverStorageVersions,
  * after the removal, if all participating API servers have the same
    encodingVersion, then sets storageVersion.status.AgreedEncodingVersion and
    status.condtion.
* checks if the storageVersion.status.serverStorageVersions is empty,
  * if empty, deletes the storageVersion object (2nd kind of garbage),
  * otherwise updates the storageVersion object,
  * both the delete and update operations are preconditioned with the rv in the
    first step to avoid conflicting with API servers modifying the object.

An API server needs to establish its membership in the list of participating API
servers before updating storageVersion, otherwise the above control loop can
mistake a storageVersion.status.serverStorageVersions entry added by a new API
server as a stale entry.

### CRDs

Today, the [storageVersionHash] in the discovery document in HA setup can
diverge from the actual storage version being used. See the [appendix] for
details.

[appendix]:#appendix
[storageVersionHash]:https://github.com/kubernetes/kubernetes/blob/c008cf95a92c5bbea67aeab6a765d7cb1ac68bd7/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/types.go#L989

To ensure that the storageVersion.status always shows the actual encoding
versions, the apiextension-apiserver must update the storageVersion.status
before it [enables] the custom resource handler. This way it does not require
the [filter] mechanism that is used by the kube-apiserver to ensure the
correct order.

[enables]:https://github.com/kubernetes/kubernetes/blob/220498b83af8b5cbf8c1c1a012b64c956d3ebf9b/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/customresource_handler.go#L703
[filter]:#updating-storageversion

### Aggregated API servers

Most code changes will be done in the generic apiserver library, so aggregated
API servers using the library will get the same behavior.

If an aggregated API server does not use the API, then the storage migrator does
not manage its API.

## Consuming the StorageVersion API

The consumer of the StorageVersion API is the storage migrator. The storage
migrator
* starts migration if the storageVersion.status.agreedEncodingVersion differs
  from the storageState.status.[persistedStorageVersionHashes],
* aborts ongoing migration if the storageVersion.status.agreedEncodingVersion is
  empty.

[persistedStorageVersionHashes]:https://github.com/kubernetes-sigs/kube-storage-version-migrator/blob/60dee538334c2366994c2323c0db5db8ab4d2838/pkg/apis/migration/v1alpha1/types.go#L164

## StorageVersion API vs. StorageVersionHash in the discovery document

We do not change how the storageVersionHash in the discovery document is
updated. The only consumer of the storageVersionHash is the storage migrator,
which will convert to use the new StorageVersion API. After the StorageVersion
API becomes stable, we will remove the storageVersionHash from the discovery
document, following the standard API deprecation process.

## Backwards Compatibility

There is no change to the existing API, so there is no backwards compatibility
concern.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

The code related to this change is mostly new:

- `k8s.io/apiserver/pkg/storageversion`
- `k8s.io/kubernetes/pkg/controller/storageversiongc`
- `k8s.io/apiserver/pkg/endpoints/filters/storageversion.go`

##### Integration tests

The wiring of the new code into existing code paths will be tested via:

- `TestStorageVersionBootstrap`: https://github.com/kubernetes/kubernetes/blob/ea0764452222146c47ec826977f49d7001b0ea8c/test/integration/storageversion/storage_version_filter_test.go#L148
- `TestStorageVersionGarbageCollection`: https://github.com/kubernetes/kubernetes/blob/0a7f45d5baa88dbe4d71102abe3a751a829d87a2/test/integration/storageversion/gc_test.go#L50

##### e2e tests

No E2E tests are required for this enhancement as the functionality can be completely
tested via unit and integration tests (the kubelet is not relevant to this change).

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial unit and integration tests completed and enabled
- Storage version support for Kuberentes API server and aggregated API servers

#### Beta

- Storage version support for API extensions API server (custom resources)
- Add tests for feature enablement/disablement based on the feature flag

#### GA

- ==TODO==

#### Deprecation

- See discussion regarding `StorageVersionHash` above.

### Upgrade / Downgrade Strategy

No specific changes to upgrade and downgrade strategies are required with regards to this
feature (see enablement and rollback section below).  Once this feature is widely available
and deployed, it will make future upgrades easier because it enables fully automated storage
migration without the risk of polluted migrations.

### Version Skew Strategy

1. Q: if an API server is rolled back when the migrator is in the middle of
   migration, how to prevent corruption? ([original question])

   A: Unlike the discovery document, the new StorageVersion API is persisted in
   etcd and has the resourceVersion(RV) field, so the migrator can determine if
   the storage version has changed in the middle of migration by comparing the
   RV of the storageVersion object before and after the migration. Also, as an
   optimization, the migrator can fail quickly by aborting the ongoing migration
   if it receives a storageVersion change event via WATCH.

   [original question]:https://github.com/kubernetes/enhancements/pull/1176#discussion_r307977970

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: StorageVersionAPI
  - Components depending on the feature gate: kube-apiserver, kube-controller-manager

###### Does enabling the feature change any default behavior?

Enabling the feature will enable a new API in the apiserver, but this should not change
any behaviors for existing workloads in the cluster.  The new filter on the API server's
handler chain may result in failed requests if traffic is routed to the API server before
it has reported ready.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, like any other alpha API, it can be disabled with runtime config.
Cluster admins should be mindful of any existing consumers of this API before rolling back.
Given this is an internal API, we don't anticipate many external consumers though.

###### What happens if we reenable the feature if it was previously rolled back?

Such like any alpha API, disabling the feature will prevent kube-apiserver from serving the API,
but will not remove objects from etcd. Re-enabling the feature will allow apiserver to serve the
API again. If the feature is re-enabled against a newer API version, it should be converted after a roundtrip.

###### Are there any tests for feature enablement/disablement?

There are several tests that require the feature gate to be enabled [test/integration/storageversion/storage_version_filter_test.go](https://github.com/kubernetes/kubernetes/blob/ea0764452222146c47ec826977f49d7001b0ea8c/test/integration/storageversion/storage_version_filter_test.go)
and [test/integration/storageversion/gc_test.go](https://github.com/kubernetes/kubernetes/blob/0a7f45d5baa88dbe4d71102abe3a751a829d87a2/test/integration/storageversion/gc_test.go).

However, there are no tests that test feature enablement/disablement based on the gate, these should be added prior to Beta.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Some potential rollout/rollback failures:
* bugs in conversion code could make re-enabling the API on newer versions unsafe
* bugs in the API / controllers could falsely report the desired storage version of an API, causing undesired storage migration from external consumers (e.g. kube-storage-version-migrator).
* a paused/stuck upgrade could block writes to resources where the agreed encoding version has not reached consensus.
* rolling back the feature and disabling the API could break clients of a cluster that depended on the API

###### What specific metrics should inform a rollback?

Recently added [apiserver health check metrics](https://github.com/kubernetes/kubernetes/pull/112741) would be useful here:
  - kubernetes_healthcheck{name="poststarthook/built-in-resources-storage-version-updater",type="healthz"}
  - kubernetes_healthcheck{name="poststarthook/built-in-resources-storage-version-updater",type="readyz"}
  - kubernetes_healthcheck{name="poststarthook/built-in-resources-storage-version-updater",type="livez"}

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet, but manual validation should be complete prior to Beta. Steps for manual validation should be reported in this section of the KEP before Beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Operators can check the existence of the StorageVersion API, as well as apiserver
health check metrics to check that the feature is working as expected.
Operations against the StorageVersion API can be checked via audit logs to see
if the API is in use by any controllers in the cluster.

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason:
- [X] API .status
  - Condition name:
  - Other field: `.status.storageVersions`, `.status.commonEncodingVersion`
- [X] Other (treat as last resort)
  - Details: audit logs showing operations against StorageVersion API

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Regarding availability/latency of serving the new API, this should follow the [existing latency SLOs](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/slos.md#steady-state-slisslos)
for serving mutating or read-only API calls.

Reasonable SLOs specific to the StorageVersion API could be:
- storageVersion.status.serverStorageVersions is accurately updated for an apiserver within 1 minute of start-up.
- storageVersion.status.agreedEncodingVersion is accurately updated for the cluster within 1 minute of completing an upgrade.

1 minute seems reasonable since this feature depends on the `APIServerIdentity` feature, that relies on a heart beat from each apiserver
to create a new Lease object obtaining an ID that will be used in the `storageVersion.status.serverStorageVersions[*].apiServerID` field.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - Metric name: `kubernetes_healthcheck{name="poststarthook/built-in-resources-storage-version-updater"}`
  - Components exposing the metric: kube-apiserver

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

The following metrics could be useful, but are likely not practical due to cardinality issues or complexity of the implementation:
- The latency for a single apiserver to update it's encoding version after start-up
- The latency for all apiservers to reach a consensus on the agreed encoding version for a storage version

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No, but it does depend on the `APIServerIdentity` feature in kube-apiserver.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes, there would be new API calls for the StorageVersion API. We don't anticipate
a lot of traffic for this API since the only anticipated consumer would be controllers
such as [kube-storage-version-migrator](https://github.com/kubernetes-sigs/kube-storage-version-migrator).

###### Will enabling / using this feature result in introducing new API types?

Yes, the StorageVersion API. There will be 1 object per API group.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

SLIs/SLOs for API calls could be impacted in cases where kube-apiservers have not reached consensus on the encoding version for a resource
for a long duration of time. This could be possible during version skew of kube-apiservers or other reasons that pause an upgrade.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The API will not be served if kube-apiserver or etcd is unavailable.
Since this feature is tightly coupled to kube-apiserver and etcd already,
unavailability of either component would make the feature inaccessible.

In some cases where only etcd is unavailable, the desired storage version reported
by apiserver could be stale. However this should not be a big concern since
objects during this time would not be receiving writes anyways and the storage version would
be updated eventually when etcd is available again. In no situation can the agreed
encoding version flip back to an old version when etcd is unavailable, since this
would only happen once all kube-apiserver's have successfully reported the new version.

###### What are other known failure modes?

- [StorageVersion API is not being updated]
  - Detection: `kubernetes_healthcheck{name="poststarthook/built-in-resources-storage-version-updater",type="healthz"}`
      indicates that the storage version updater is not running.
  - Mitigations: Check kube-apiserver logs for specific errors. Restarting may be helpful.
  - Diagnostics: Errors from the [storage version hanndler chain](https://github.com/kubernetes/kubernetes/blob/e11f23eb9712c9b4bebf8dd85dcb04441b4fb705/staging/src/k8s.io/apiserver/pkg/endpoints/filters/storageversion.go#L110)
  - Testing: There can be many reasons for why kube-apiserver can't update an API that are hard to reproduce in a test.

- [Pending storage migrations can block writes to resources]
  - Detection: kube-apiserver returns 503 on writes with error `wait for storage version registration to complete for resource: <resource>`
  - Mitigations: Check kube-apiserver logs for specific errors and check etcd health
  - Diagnostics: Errors from the [storage version hanndler chain](https://github.com/kubernetes/kubernetes/blob/e11f23eb9712c9b4bebf8dd85dcb04441b4fb705/staging/src/k8s.io/apiserver/pkg/endpoints/filters/storageversion.go#L110)
  - Testing: Updates to storage version can fail for many reasons (e.g. connectivity to etcd) that are hard to reproduce in a test.

###### What steps should be taken if SLOs are not being met to determine the problem?

As a last resort, cluster admins can check kube-apiserver logs for errors that may indicate
why the StorageVersion is not being updated. In many cases when there are issues with StorageVersion,
operators should check the health of etcd in their clusters.

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

==TODO==

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

==TODO==

## Alternatives

### Letting API servers vote on the storage version

See [#1201](https://github.com/kubernetes/enhancements/pull/920)

The voting mechanism makes sure all API servers in an HA cluster always use the
same storage version, and the discovery document always lists the selected
storage version.

Cons:
* The voting mechanism adds complexity. For the storage migrator to work
  correctly, it is NOT necessary to guarantee all API server instances always
  use the same storage version.

### Letting the storage migrator detect if API server instances are in agreement

See [#920](https://github.com/kubernetes/enhancements/pull/920)

Cons: it has many assumptions, see [cons].
[cons]:https://github.com/kubernetes/enhancements/pull/920/files#diff-a1d206b4bbac708bf71ef85ad7fb5264R339

## Appendix

### Accuracy of the discovery document of CRDs

Today, the storageVersionHash listed in the discovery document "almost"
accurately reflects the actual storage version used by the apiextension-apiserver.

Upon storage version changes in the CRD spec,
* [one controller] deletes the existing resource handler of the CRD, so that
  a new resource handler is created with the latest cached CRD spec is created
  upon the next custom resource request.
* [another controller] enqueues the CRD, waiting for the worker to updates the
  discovery document.

[one controller]:https://github.com/kubernetes/kubernetes/blob/1a53325550f6d5d3c48b9eecdd123fd84deee879/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/customresource_handler.go#L478
[another controller]:https://github.com/kubernetes/kubernetes/blob/1a53325550f6d5d3c48b9eecdd123fd84deee879/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/customresource_discovery_controller.go#L258

These two controllers are driven by the [same informer], so the lag between
when the server starts to apply the new storage version and when the discovery
document is updated is just the difference between when the respective
goroutines finish.
[same informer]:https://github.com/kubernetes/kubernetes/blob/1a53325550f6d5d3c48b9eecdd123fd84deee879/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/apiserver.go#L192-L210

Note that in HA setup, there is a lag between when apiextension-apiserver
instances observe the CRD spec change.

## References
1. Email thread [kube-apiserver: Self-coordination](https://groups.google.com/d/msg/kubernetes-sig-api-machinery/gTS-rUuEVQY/9bUFVnYvAwAJ)
