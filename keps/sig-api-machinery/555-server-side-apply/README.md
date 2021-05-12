# Apply

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    - [API Topology](#api-topology)
      - [Lists](#lists)
      - [Maps and structs](#maps-and-structs)
    - [Kubectl](#kubectl)
      - [Server-side Apply](#server-side-apply)
    - [Status Wiping](#status-wiping)
      - [Current Behavior](#current-behavior)
      - [Proposed Change](#proposed-change)
      - [Alternatives](#alternatives)
      - [Implementation History](#implementation-history)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Testing Plan](#testing-plan)
- [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Upgrade from kubectl Client-Side to Server-Side Apply](#upgrade-from-kubectl-client-side-to-server-side-apply)
      - [Avoiding Conflicts from Client-Side Apply to Server-Side Apply](#avoiding-conflicts-from-client-side-apply-to-server-side-apply)
    - [Downgrade from kubectl Server-Side to Client-Side Apply](#downgrade-from-kubectl-server-side-to-client-side-apply)
    - [Downgrade the API Server](#downgrade-the-api-server)
- [Implementation History](#implementation-history-1)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives-1)
<!-- /toc -->

## Summary

`kubectl apply` is a core part of the Kubernetes config workflow, but it is
buggy and hard to fix. This functionality will be regularized and moved to the
control plane.

## Motivation

Example problems today:

* User does POST, then changes something and applies: surprise!
* User does an apply, then `kubectl edit`, then applies again: surprise!
* User does GET, edits locally, then apply: surprise!
* User tweaks some annotations, then applies: surprise!
* Alice applies something, then Bob applies something: surprise!

Why can't a smaller change fix the problems? Why hasn't it already been fixed?

* Too many components need to change to deliver a fix
* Organic evolution and lack of systematic approach
  * It is hard to make fixes that cohere instead of interfere without a clear model of the feature
* Lack of API support meant client-side implementation
  * The client sends a PATCH to the server, which necessitated strategic merge patch--as no patch format conveniently captures the data type that is actually needed.
  * Tactical errors: SMP was not easy to version, fixing anything required client and server changes and a 2 release deprecation period.
* The implications of our schema were not understood, leading to bugs.
  * e.g., non-positional lists, sets, undiscriminated unions, implicit context
  * Complex and confusing defaulting behavior (e.g., Always pull policy from :latest)
  * Non-declarative-friendly API behavior (e.g., selector updates)

### Goals

"Apply" is intended to allow users and systems to cooperatively determine the
desired state of an object. The resulting system should:

* Be robust to changes made by other users, systems, defaulters (including mutating admission control webhooks), and object schema evolution.
* Be agnostic about prior steps in a CI/CD system (and not require such a system).
* Have low cognitive burden:
  * For integrators: a single API concept supports all object types; integrators
    have to learn one thing total, not one thing per operation per api object.
    Client side logic should be kept to a minimum; CURL should be sufficient to
    use the apply feature.
  * For users: looking at a config change, it should be intuitive what the
    system will do. The “magic” is easy to understand and invoke.
  * Error messages should--to the extent possible--tell users why they had a
    conflict, not just what the conflict was.
  * Error messages should be delivered at the earliest possible point of
    intervention.

Goal: The control plane delivers a comprehensive solution.

Goal: Apply can be called by non-go languages and non-kubectl clients. (e.g.,
via CURL.)

### Non-Goals

* Multi-object apply will not be changed: it remains client side for now
* Some sources of user confusion will not be addressed:
  * Changing the name field makes a new object rather than renaming an existing object
  * Changing fields that can’t really be changed (e.g., Service type).

## Proposal

(Please note that when this KEP was started, the KEP process was much less well
defined and we have been treating this as a requirements / mission statement
document; KEPs have evolved into more than that.)

A brief list of the changes:

* Apply will be moved to the control plane.
  * The [original design](https://goo.gl/UbCRuf) is in a google doc; joining the
    kubernetes-dev or kubernetes-announce list will grant permission to see it.
    Since then, the implementation has changed so this may be useful for
    historical understanding. The test cases and examples there are still valid.
  * Additionally, readable in the same way, is the [original design for structured diff and merge](https://goo.gl/nRZVWL);
    we found in practice a better mechanism for our needs (tracking field
    managers) but the formalization of our schema from that document is still
    correct.
* Apply is invoked by sending a certain Content-Type with the verb PATCH.
* Instead of using a `kubectl.kubernetes.io/last-applied-configuration` annotation,
the control plane will track a "manager" for every field.
* Apply is for users and/or ci/cd systems. We modify the POST, PUT (and
  non-apply PATCH) verbs so that when controllers or other systems make changes
  to an object, they are made "managers" of the fields they change.
* The things our "Go IDL" describes are formalized: [structured merge and diff](https://github.com/kubernetes-sigs/structured-merge-diff)
* Existing Go IDL files will be fixed (e.g., by [fixing the directives](https://github.com/kubernetes/kubernetes/pull/70100/files))
* Dry-run will be implemented on control plane verbs (POST, PUT, PATCH).
  * Admission webhooks will have their API appended accordingly.
* An upgrade path will be implemented so that version skew between kubectl and
  the control plane will not have disastrous results.

The linked documents should be read for a more complete picture.

### Implementation Details/Notes/Constraints [optional]

(TODO: update this section with current design)

#### API Topology

Server-side apply has to understand the topology of the objects in order to make
valid merging decisions. In order to reach that goal, some new Go markers, as
well as OpenAPI extensions have been created:

##### Lists

Lists can behave in mostly 3 different ways depending on what their actual semantic
is. New annotations allow API authors to define this behavior.

- Atomic lists: The list is owned by only one person and can only be entirely
replaced. This is the default for lists. It is defined either in Go IDL by
pefixing the list with `// +listType=atomic`, or in the OpenAPI
with `"x-kubenetes-list-type": "atomic"`.

- Sets: the list is a set (it has to be of a scalar type). Items in the list
must appear at most once. Individual actors of the API can own individual items.
It is defined either in Go IDL by pefixing the list with `//
+listType=set`, or in the OpenAPI with
`"x-kubenetes-list-type": "set"`.

- Associative lists: Kubernetes has a pattern of using lists as dictionary, with
"name" being a very common key. People can now reproduce this pattern by using
`// +listType=map`, or in the OpenAPI with `"x-kubernetes-list-type": "map"`
along with `"x-kubernetes-list-map-keys": ["name"]`, or `// +listMapKey=name`.
Items of an associative lists are owned by the person who applied the item to
the list.

For compatibility with the existing markers, the `patchStrategy` and
`patchMergeKey` markers are automatically used and converted to the corresponding `listType`
and `listMapKey` if missing.

##### Maps and structs

Maps and structures can behave in two ways:
- Each item in the map or field in the structure are independent from each
other. They can be changed by different actors. This is the default behavior,
but can be explicitly specified with `// +mapType=granular` or `//
+structType=granular` respectively. They map to the same openapi extension:
`"x-kubernetes-map-type": "granular"`.
- All the fields or item of the map are treated as one unit, we say the map/struct is
atomic. That can be specified with `// +mapType=atomic` or `//
+structType=atomic` respectively. They map to the same openapi extension:
`"x-kubernetes-map-type": "atomic"`.

#### Kubectl

##### Server-side Apply

Since server-side apply is currently in the Alpha phase, it is not
enabled by default on kubectl. To use server-side apply on servers
with the feature, run the command
`kubectl apply --experimental-server-side ...`.

If the feature is not available or enabled on the server, the command
will fail rather than fall-back on client-side apply due to significant
semantical differences.

As the feature graduates to the Beta phase, the flag will be renamed to `--server-side`.

The long-term plan for this feature is to be the default apply on all
Kubernetes clusters. The semantical differences between server-side
apply and client-side apply will make a smooth roll-out difficult, so
the best way to achieve this has not been decided yet.

#### Status Wiping

##### Current Behavior

Right before being persisted to etcd, resources in the apiserver undergo a preparation mechanism that is custom for every resource kind.
It takes care of things like incrementing object generation and status wiping.
This happens through [PrepareForUpdate](https://github.com/kubernetes/kubernetes/blob/bc1360ab158d524c5a7132c8dd9dc7f7e8889af1/staging/src/k8s.io/apiserver/pkg/registry/rest/update.go#L49) and [PrepareForCreate](https://github.com/kubernetes/kubernetes/blob/bc1360ab158d524c5a7132c8dd9dc7f7e8889af1/staging/src/k8s.io/apiserver/pkg/registry/rest/create_update.go#L37).

The problem status wiping at this level creates is, that when a user applies a field that gets wiped later on, it gets owned by said user.
The apply mechanism (FieldManager) can not know which fields get wiped for which resource and therefor can not ignore those.

Additionally ignoring status as a whole is not enough, as it should be possible to own status (and other fields) in some occasions. More conversation on this can be found in the [GitHub issue](https://github.com/kubernetes/kubernetes/issues/75564) where the problem got reported.

##### Proposed Change

Add an interface that resource strategies can implement, to provide field sets affected by status wiping.

```go
# staging/src/k8s.io/apiserver/pkg/registry/rest/rest.go
// ResetFieldsProvider is an optional interface that a strategy can implement
// to expose a set of fields that get reset before persisting the object.
type ResetFieldsProvider interface {
  // ResetFieldsFor returns a set of fields for the provided version that get reset before persisting the object.
  // If no fieldset is defined for a version, nil is returned.
  ResetFieldsFor(version string) *fieldpath.Set
}
```

Additionally, this interface is implemented by `registry.Store` which forwards it to the corresponding strategy (if applicable).
If `registry.Store` can not provide a field set, it returns nil.

An example implementation for the interface inside the pod strategy could be:

```go
# pkg/registry/core/pod/strategy.go
// ResetFieldsFor returns a set of fields for the provided version that get reset before persisting the object.
// If no fieldset is defined for a version, nil is returned.
func (podStrategy) ResetFieldsFor(version string) *fieldpath.Set {
  set, ok := resetFieldsByVersion[version]
  if !ok {
    return nil
  }
  return set
}

var resetFieldsByVersion = map[string]*fieldpath.Set{
  "v1": fieldpath.NewSet(
    fieldpath.MakePathOrDie("status"),
  ),
}
```

When creating the handlers in [installer.go](https://github.com/kubernetes/kubernetes/blob/3ff0ed46791a821cb7053c1e25192e1ecd67a6f0/staging/src/k8s.io/apiserver/pkg/endpoints/installer.go) the current `rest.Storage` is checked to implement the `ResetFieldsProvider` interface and the result is passed to the FieldManager.

```go
# staging/src/k8s.io/apiserver/pkg/endpoints/installer.go
var resetFields *fieldpath.Set
if resetFieldsProvider, isResetFieldsProvider := storage.(rest.ResetFieldsProvider); isResetFieldsProvider {
    resetFields = resetFieldsProvider.ResetFieldsFor(a.group.GroupVersion.Version)
}
```

When provided with a field set, the FieldManager strips all `resetFields` from incoming update and apply requests.
This causes the user/manager to not own those fields.

```go
...
if f.resetFields != nil {
  patchObjTyped = patchObjTyped.Remove(f.resetFields)
}
...
```

##### Alternatives

We looked at a way to get the fields affected by status wiping without defining them separately.
Mainly by pulling the reset logic from the strategies `PrepareForCreate` and `PrepareForUpdate` methods into a new method `ResetFields` implementing an `ObjectResetter` interface.

This approach did not work as expected, because the strategy works on internal types while the FieldManager handles external api types.
The conversion between the two and creating the diff was complex and would have caused a notable amount of allocations.

##### Implementation History

- 12/2019 [#86083](https://github.com/kubernetes/kubernetes/pull/86083) implementing a poc for the described approach

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

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: [ServerSideApply](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/features/kube_features.go#L100)
    - Components depending on the feature gate: kube-apiserver

* **Does enabling the feature change any default behavior?**

  While this changes how objects are modified and then stored in the database, all the changes should be strictly backward compatible, and shouldn’t break existing automation or users. The increase in size can possibly have adverse, surprising consequences including increased memory usage for controllers, increased bandwidth usage when fetching objects, bigger objects when displaying for users (kubectl get -o yaml). We’re trying to mitigate all of these with the addition of a new header.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Also set `disable-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g., if this is a runtime
  feature, can it break the existing applications?).

  Yes. The consequence is that managed fields will be reset for server-side applied objects (requiring a read/write cycle on the impacted resources).

* **What happens if we reenable the feature if it was previously rolled back?**

  The feature will be restored. Server-side applied objects will have lost their “set” which may cause some surprising behavior (fields might not be removed as expected).

* **Are there any tests for feature enablement/disablement?**
  The e2e framework does not currently support enabling or disabling feature
  gates. However, unit tests in each component dealing with managing data, created
  with and without the feature, are necessary. At the very least, think about
  conversion tests if API types are being modified.

  Tests are in place for upgrading from client side to server side apply and vice versa.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?
There is no specific way that the rollout can fail. The rollout can't impact existing workload.
* **What specific metrics should inform a rollback?**

  The feature shouldn't affect any existing behavior. A surprisingly high number of modification rejections could be a sign that something is not working properly.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

  Because the feature doesn't affect existing behavior, rollback and upgrades haven't be specifically tested.
The feature is being used by the cluster role aggregator though. Upgrading/downgrading/upgrading, which
could result in the managedFields being removed, wouldn't cause any problems since the `Rules` field
filled by the controller is `atomic`, and thus doesn't depend on the current state of the managedFields.

  The new `managedFields` field is cleared when it is incorrect. That protects us from having invalid data inserted by a potential bad upgrade.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?** No
No.
### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

  Any existing metric split by request verb will record the [APPLY](https://github.com/kubernetes/kubernetes/blob/8f6ffb24df989608b87451f89b8ac9fc338ed71c/staging/src/k8s.io/apiserver/pkg/endpoints/metrics/metrics.go#L507-L509) verb if the feature is in use.

  Additionally, the OpenAPI spec exposes the available media-type for each individual endpoint. The presence of the `apply` type for the PATCH verb of a endpoints indicates whether the feature is enabled for that specific resource, e.g.
```json
...
"patch": {
  "consumes": [
     "application/json-patch+json",
     "application/merge-patch+json",
     "application/strategic-merge-patch+json",
     "application/apply-patch+yaml"
   ],
    ...
}
...
```

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**

  There is no specific metric attached to server side apply. All PATCH requests that utilize SSA will use the verb APPLY when logging metrics. API Server metrics that are split by verb automatically include this. They include `apiserver_request_total`, `apiserver_longrunning_gauge`, `apiserver_response_sizes`, `apiserver_request_terminations_total`, `apiserver_selfrequest_total`
    - Components exposing the metric: kube-apiserver

  Apply requests (`PATCH` with `application/apply-patch+yaml` mime type) have the same level of SLIs as other types of requests.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?** n/a
Apply requests (`PATCH` with `application/apply-patch+yaml` mime type) have the same level of SLOs as other types of requests.
* **Are there any missing metrics that would be useful to have to improve observability
of this feature?** n/a

### Dependencies

* **Does this feature depend on any specific services running in the cluster?** No

### Scalability

* **Will enabling / using this feature result in any new API calls?** No

* **Will enabling / using this feature result in introducing new API types?**
  Describe them, providing: No

* **Will enabling / using this feature result in any new calls to the cloud
provider?** No

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?** Objects applied using server side apply will have their managed fields metadata populated. `managedFields` metadata fields can represent up to 60% of the total size of an object, increasing the size of objects.

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?** No

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?** Since objects are larger with the new `managedFields`, caches as well as network bandwidth requirement will increase.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

  The feature is part of of the API server and will not function without it

* **What are other known failure modes?**
  For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node? Apply requests (`PATCH` with `application/apply-patch+yaml` mime type) have the same level of SLIs as other types of requests.
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads? This shouldn't affect running workloads, and this feature shouldn't alter the behavior of previously existing mechanisms like PATCH and PUT.
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue? The feature uses very little logging, and errors should be returned directly to the user.
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? Failure modes are tested exhaustively both as unit-tests and as integration tests.

* **What steps should be taken if SLOs are not being met to determine the problem?** n/a

### Risks and Mitigations

We used a feature branch to ensure that no partial state of this feature would
be in master. We developed the new "business logic" in a
[separate repo](https://github.com/kubernetes-sigs/structured-merge-diff) for
velocity and reusability.

### Testing Plan

The specific logic of apply will be tested by extensive unit tests in the
[structured merge and diff](https://github.com/kubernetes-sigs/structured-merge-diff)
repo. The integration between that repo and kubernetes/kubernetes will mainly
be tested by integration tests in [test/integration/apiserver/apply](https://github.com/kubernetes/kubernetes/tree/master/test/integration/apiserver/apply)
and [test/cmd](https://github.com/kubernetes/kubernetes/blob/master/test/cmd/apply.sh),
as well as unit tests where applicable. The feature will also be enabled in the
[alpha-features e2e test suite](https://k8s-testgrid.appspot.com/sig-release-master-blocking#gce-cos-master-alpha-features),
which runs every hour and everytime someone types `/test pull-kubernetes-e2e-gce-alpha-features`
on a PR. This will ensure that the cluster can still start up and the other
endpoints will function normally when the feature is enabled.

Unit Tests in [structured merge and diff](https://github.com/kubernetes-sigs/structured-merge-diff) repo for:

- [x] Merge typed objects of the same type with a schema. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/typed/merge_test.go)
- [x] Merge deduced typed objects without a schema (for CRDs). [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/typed/deduced_test.go)
- [x] Convert a typed value to a field set. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/typed/toset_test.go)
- [x] Diff two typed values. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/typed/symdiff_test.go)
- [x] Validate a typed value against it's schema. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/typed/validate_test.go)
- [x] Get correct conflicts when applying. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/merge/conflict_test.go)
- [x] Apply works for deduced typed objects. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/merge/deduced_test.go)
- [x] Apply works for leaf fields with scalar values. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/merge/leaf_test.go)
- [x] Apply works for items in associative lists of scalars. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/merge/set_test.go)
- [x] Apply works for items in associative lists with keys. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/merge/key_test.go)
- [x] Apply works for nested schemas, including recursive schemas. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/merge/nested_test.go)
- [x] Apply works for multiple appliers. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/9f6585cadf64c6b61b5a75bde69ba07d5d34dc3f/merge/multiple_appliers_test.go#L31-L685)
- [x] Apply works when the object conversion changes value of map keys. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/9f6585cadf64c6b61b5a75bde69ba07d5d34dc3f/merge/multiple_appliers_test.go#L687-L886)
- [x] Apply works when unknown/obsolete versions are present in managedFields (for when APIs are deprecated). [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/merge/obsolete_versions_test.go)

Unit Tests for:

- [x] Apply strips certain fields (like name and namespace) from managers. [link](https://github.com/kubernetes/kubernetes/blob/8a6a2883f9a38e09ae941b62c14f4e68037b2d21/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/fieldmanager_test.go#L69-L139)
- [x] ManagedFields API can be round tripped through the structured-merge-diff format. [link](https://github.com/kubernetes/kubernetes/blob/4394bf779800710e67beae9bddde4bb5425ce039/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/managedfields_test.go#L30-L156)
- [x] Manager identifiers passed to structured-merge-diff are encoded as json. [link](https://github.com/kubernetes/kubernetes/blob/4394bf779800710e67beae9bddde4bb5425ce039/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/managedfields_test.go#L158-L202)
- [x] Managers will be sorted by operation, then timestamp, then manager name. [link](https://github.com/kubernetes/kubernetes/blob/4394bf779800710e67beae9bddde4bb5425ce039/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/managedfields_test.go#L204-L304)
- [x] Conflicts will be returned as readable status errors. [link](https://github.com/kubernetes/kubernetes/blob/69b9167dcbc8eea2ca5653fa42584539920a1fd4/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/conflict_test.go#L31-L106)
- [x] Fields API can be round tripped through the structured-merge-diff format. [link](https://github.com/kubernetes/kubernetes/blob/0e1d50e70fdc9ed838d75a7a1abbe5fa607d22a1/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/fields_test.go#L29-L57)
- [x] Fields API conversion to and from the structured-merge-diff format catches errors. [link](https://github.com/kubernetes/kubernetes/blob/0e1d50e70fdc9ed838d75a7a1abbe5fa607d22a1/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/fields_test.go#L59-L109)
- [x] Path elements can be round tripped through the structured-merge-diff format. [link](https://github.com/kubernetes/kubernetes/blob/6b2e4682fe883eebcaf1c1e43cf2957dde441174/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/pathelement_test.go#L21-L54)
- [x] Path element conversion will ignore unknown qualifiers. [link](https://github.com/kubernetes/kubernetes/blob/6b2e4682fe883eebcaf1c1e43cf2957dde441174/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/pathelement_test.go#L56-L61)
- [x] Path element conversion will fail if a known qualifier's value is invalid. [link](https://github.com/kubernetes/kubernetes/blob/6b2e4682fe883eebcaf1c1e43cf2957dde441174/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/pathelement_test.go#L63-L84)
- [x] Can convert both built-in objects and CRDs to structured-merge-diff typed objects. [link](https://github.com/kubernetes/kubernetes/blob/42aba643290c19a63168513bd758822e8014a0fd/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/typeconverter_test.go#L40-L135)
- [x] Can convert structured-merge-diff typed objects between API versions. [link](https://github.com/kubernetes/kubernetes/blob/0e1d50e70fdc9ed838d75a7a1abbe5fa607d22a1/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/versionconverter_test.go#L32-L69)

Integration tests for:

- [x] Creating an object with apply works with default and custom storage implementations. [link](https://github.com/kubernetes/kubernetes/blob/1b8c8f1daf4b1ed6d17ee1d2f40d62c8ecec0e15/test/integration/apiserver/apply/apply_test.go#L55-L121)
- [x] Create is blocked on apply if uid is provided. [link](https://github.com/kubernetes/kubernetes/blob/1b8c8f1daf4b1ed6d17ee1d2f40d62c8ecec0e15/test/integration/apiserver/apply/apply_test.go#L123-L154)
- [x] Apply has conflicts when changing fields set by Update, and is able to force. [link](https://github.com/kubernetes/kubernetes/blob/1b8c8f1daf4b1ed6d17ee1d2f40d62c8ecec0e15/test/integration/apiserver/apply/apply_test.go#L156-L239)
- [x] There are no changes to the managedFields API. [link](https://github.com/kubernetes/kubernetes/blob/1b8c8f1daf4b1ed6d17ee1d2f40d62c8ecec0e15/test/integration/apiserver/apply/apply_test.go#L241-L341)
- [x] ManagedFields has no entries for managers who manage no fields. [link](https://github.com/kubernetes/kubernetes/blob/1b8c8f1daf4b1ed6d17ee1d2f40d62c8ecec0e15/test/integration/apiserver/apply/apply_test.go#L343-L392)
- [x] Apply works with custom resources. [link](https://github.com/kubernetes/kubernetes/blob/b55417f429353e1109df8b3bfa2afc8dbd9f240b/staging/src/k8s.io/apiextensions-apiserver/test/integration/apply_test.go#L34-L117)
- [x] Run kubectl apply tests with server-side flag enabled. [link](https://github.com/kubernetes/kubernetes/blob/81e6407393aa46f2695e71a015f93819f1df424c/test/cmd/apply.sh#L246-L314)

E2E and Conformance tests will be added for GA.

## Graduation Criteria

An alpha version of this is targeted for 1.14.

This can be promoted to beta when it is a drop-in replacement for the existing
kubectl apply, and has no regressions (which aren't bug fixes). This KEP will be
updated when we know the concrete things changing for beta.

A GA version of this is targeted for 1.22.

- E2E tests are created and graduate to conformance
- [Apply for client-go's typed client](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/2144-clientgo-apply) is implemented and at least one kube-controller-manager uses that client
- Outstanding bugs around status wiping and scale subresource are fixed

### Upgrade / Downgrade Strategy

#### Upgrade from kubectl Client-Side to Server-Side Apply

With client-side `kubectl apply`, the annotation
`kubectl.kubernetes.io/last-applied-configuration` tracks ownership for a single
shared field manager. With server-side `kubectl apply --server-side`, the
`.metadata.managedFields` field tracks ownership for multiple field managers.

Users who wish to start using server-side apply for objects managed with
client-side apply would encounter a field manager conflict: the field set that
the user now wants to manage with server-side apply will be owned by the
client-side apply field manager.

If we don't specifically handle this case, then users would need to force
conflicts with `kubectl apply --server-side --force-conflicts`. This extra step
is not desirable for users who wish to onboard to server-side apply.

However we know that users' intent is to take ownership of client-side apply
fields when upgrading, which we can do for them while avoiding the conflict.

##### Avoiding Conflicts from Client-Side Apply to Server-Side Apply

We'll use the `kubectl` user-agent and the client-side apply
`last-applied-configuration` annotation to identify when to do the upgrade.

When server-side apply is run with `kubectl apply --server-side` on an object
with a `last-applied-configuration` annotation for client-side apply, then the
annotation will be upgraded to the managed fields server-side apply notation.

To upgrade the `last-applied-configuration` annotation, the following procedure
will be used.

1.  Identify if the server-side apply is from the `kubectl` user-agent
1.  Identify if the server-side apply would result in a conflict
1.  Create a fieldset from the `last-applied-configuration` annotation.
1.  Remove all fields from the `last-applied-configuration` annotation that are
    added, missing, or different than the corresponding field of the live
    object. Because the fields have changed, client-side apply does not own
    them.
1.  Compare the "last-applied" fieldset to the conflict fieldset. Take the
    difference as the new conflict fieldset. If the conflict fieldset is empty,
    then the conflicts are allowed and we force the server-side apply. If the
    conflict fieldset is not empty, then return the conflict fieldset.

#### Downgrade from kubectl Server-Side to Client-Side Apply

Client-side `kubectl apply` users can incrementally upgrade to a version of
`kubectl` that can send a server-side apply

We can sync the intent between server-side and client-side apply by keeping the
`last-applied-configuration` annotation up-to-date with the `.managedFields`
field.

Client-side apply will continue to work.

#### Downgrade the API Server

When downgrading the API server with server-side apply disabled, then
`.metadata.managedFields` field will be cleared since the API server doesn't
know about this field. A server-side apply will fail with a content-type unknown
error.

A client-side apply would succeed because the `last-applied-configuration`
annotation is preserved and up-to-date as described in the downgrade above.

## Implementation History

* Early 2018: @lavalamp begins thinking about apply and writing design docs
* 2018Q3: Design shift from merge + diff to tracking field managers.
* 2019Q1: Alpha.
* 2019Q3: Beta.

(For more details, one can view the apply-wg recordings, or join the mailing list
and view the meeting notes. TODO: links)

## Drawbacks

Why should this KEP _not_ be implemented: many bugs in kubectl apply will go
away. Users might be depending on the bugs.

## Alternatives

It's our belief that all routes to fixing the user pain involve
centralizing this functionality in the control plane.
