# KEP-2339: StorageVersion API for HA API servers

## Table of Contents

<!-- toc -->
- [Overview](#overview)
- [API changes](#api-changes)
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
- [Graduation Plan](#graduation-plan)
- [FAQ](#faq)
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
use different storage versions encoding a resource. The [storageVersionHash][]
in the discovery document does not expose this disagreement. As a result, the
storage migrator may proceed with migration with the false belief that all API
server instances are encoding objects using the same storage version, resulting
in polluted migration.  ([details][]).

[storageVersionHash]:https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/types.go#L979
[details]:https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/35-storage-version-hash.md#ha-masters

## Motivation

==TODO==

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### Goals

- Create an API that allows an external controller to observe what storage versions are in use by API servers in an HA setup

### Non-Goals

- Automatically running storage migration

## Proposal

We propose a way to show what storage versions all API servers are using, so
that the storage migrator can defer migration until an agreement has been
reached.

### Risks and Mitigations

==TODO==

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

### Resource Version API

We introduce a new API `StorageVersion`, in a new API group
`internal.apiserver.k8s.io/v1alpha1`.

```golang
//  Storage version of a specific resource.
type StorageVersion struct {
  TypeMeta
  // The name is <group>.<resource>.
  ObjectMeta

  // Spec is omitted because there is no spec field.
  // Spec StorageVersionSpec

  // API server instances report the version they can decode and the version they
  // encode objects to when persisting objects in the backend.
  Status StorageVersionStatus
}

// API server instances report the version they can decode and the version they
// encode objects to when persisting objects in the backend.
type StorageVersionStatus struct {
  // The reported versions per API server instance.
  // +optional
  ServerStorageVersions []ServerStorageVersion
  // If all API server instances agree on the same encoding storage version,
  // then this field is set to that version. Otherwise this field is left empty.
  // +optional
  AgreedEncodingVersion string

  // The latest available observations of the storageVersion's state.
  // +optional
  Conditions []StorageVersionCondition

}

// An API server instance reports the version it can decode and the version it
// encodes objects to when persisting objects in the backend.
type ServerStorageVersion struct {
  // The ID of the reporting API server.
  // For a kube-apiserver, the ID is configured via a flag.
  APIServerID string

  // The API server encodes the object to this version when persisting it in
  // the backend (e.g., etcd).
  EncodingVersion string

  // The API server can decode objects encoded in these versions.
  // The encodingVersion must be included in the decodableVersions.
  DecodableVersions []string
}


const (
  // Indicates that storage versions reported by all servers are equal.
  AllEncondingVersionsEqual StorageVersionConditionType = "AllEncodingVersionsEqual"
)

// Describes the state of the storageVersion at a certain point.
type StorageVersionCondition struct {
	// Type of the condition.
	Type StorageVersionConditionType
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus
	// The last time this condition was updated.
	// +optional
	LastUpdateTime metav1.Time
	// The reason for the condition's last transition.
	// +optional
	Reason string
	// A human readable message indicating details about the transition.
	// +optional
	Message string
}
```

## Changes to API servers

In this section, we describe how to update and consume the StorageVersion API.

### Curating a list of participating API servers in HA master

API servers need such a list when updating the StorageVersion API. Currently,
such a list is already maintained in the "kubernetes" endpoints, though it is not
working in all flavors of Kubernetes deployments.

We will inherit the existing [mechanism][], but formalize the API and process in
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
filter to the [handler chain][]. Before kube-aggregator, kube-apiserver, and
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

Today, the [storageVersionHash][] in the discovery document in HA setup can
diverge from the actual storage version being used. See the [appendix][] for
details.

[appendix]:#appendix
[storageVersionHash]:https://github.com/kubernetes/kubernetes/blob/c008cf95a92c5bbea67aeab6a765d7cb1ac68bd7/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/types.go#L989

To ensure that the storageVersion.status always shows the actual encoding
versions, the apiextension-apiserver must update the storageVersion.status
before it [enables][] the custom resource handler. This way it does not require
the [filter][] mechanism that is used by the kube-apiserver to ensure the
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
  from the storageState.status.[persistedStorageVersionHashes][],
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

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- `<test>: <link to test coverage>`

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- `<test>: <link to test coverage>`

### Graduation Criteria

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

### Version Skew Strategy

1. Q: if an API server is rolled back when the migrator is in the middle of
   migration, how to prevent corruption? ([original question][])

   A: Unlike the discovery document, the new StorageVersion API is persisted in
   etcd and has the resourceVersion(RV) field, so the migrator can determine if
   the storage version has changed in the middle of migration by comparing the
   RV of the storageVersion object before and after the migration. Also, as an
   optimization, the migrator can fail quickly by aborting the ongoing migration
   if it receives a storageVersion change event via WATCH.

   [original question]:https://github.com/kubernetes/enhancements/pull/1176#discussion_r307977970

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

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

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
- [ ] API .status
  - Condition name: 
  - Other field: 
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

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
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

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

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

Cons: it has many assumptions, see [cons][].
[cons]:https://github.com/kubernetes/enhancements/pull/920/files#diff-a1d206b4bbac708bf71ef85ad7fb5264R339

## Appendix

### Accuracy of the discovery document of CRDs

Today, the storageVersionHash listed in the discovery document "almost"
accurately reflects the actual storage version used by the apiextension-apiserver.

Upon storage version changes in the CRD spec,
* [one controller][] deletes the existing resource handler of the CRD, so that
  a new resource handler is created with the latest cached CRD spec is created
  upon the next custom resource request.
* [another controller][] enqueues the CRD, waiting for the worker to updates the
  discovery document.

[one controller]:https://github.com/kubernetes/kubernetes/blob/1a53325550f6d5d3c48b9eecdd123fd84deee879/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/customresource_handler.go#L478
[another controller]:https://github.com/kubernetes/kubernetes/blob/1a53325550f6d5d3c48b9eecdd123fd84deee879/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/customresource_discovery_controller.go#L258

These two controllers are driven by the [same informer][], so the lag between
when the server starts to apply the new storage version and when the discovery
document is updated is just the difference between when the respective
goroutines finish.
[same informer]:https://github.com/kubernetes/kubernetes/blob/1a53325550f6d5d3c48b9eecdd123fd84deee879/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/apiserver.go#L192-L210

Note that in HA setup, there is a lag between when apiextension-apiserver
instances observe the CRD spec change.

## References
1. Email thread [kube-apiserver: Self-coordination](https://groups.google.com/d/msg/kubernetes-sig-api-machinery/gTS-rUuEVQY/9bUFVnYvAwAJ)
