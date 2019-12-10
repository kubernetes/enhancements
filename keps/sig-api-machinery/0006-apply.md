---
title: Apply
authors:
  - "@lavalamp"
owning-sig: sig-api-machinery
participating-sigs:
  - sig-api-machinery
  - sig-cli
reviewers:
  - "@pwittrock"
  - "@erictune"
approvers:
  - "@bgrant0607"
editor: TBD
creation-date: 2018-03-28
last-updated: 2018-03-28
status: implementable
see-also:
  - n/a
replaces:
  - n/a
superseded-by:
  - n/a
---

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
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Testing Plan](#testing-plan)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
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
* Instead of using a last-applied annotation, the control plane will track a
  "manager" for every field.
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
This happens through [PrepareForUpdate](https://github.com/kubernetes/kubernetes/blob/bc1360ab158d524c5a7132c8dd9dc7f7e8889af1/staging/src/k8s.io/apiserver/pkg/registry/rest/update.go#L49)
for update and [PrepareForCreate](https://github.com/kubernetes/kubernetes/blob/bc1360ab158d524c5a7132c8dd9dc7f7e8889af1/staging/src/k8s.io/apiserver/pkg/registry/rest/create_update.go#L37) for create.

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

When provided with a field set, FieldManager behavior depends on the operation:

- On `apply`, the received patch gets stripped of all `resetFields`.
If the removal results in the patch to change, the FieldManager rejects the request with an error returning the invalid fields to the user.

```go
staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/structuredmerge.go @ Apply()

func (f *structuredMergeManager) Apply(liveObj, patchObj runtime.Object, managed Managed, manager string, force bool) (runtime.Object, Managed, error) {
  ...
  if f.resetFields != nil {
    resetObjTyped := patchObjTyped.Remove(f.resetFields)
    diff, err := patchObjTyped.Compare(resetObjTyped)
    if err != nil {
      return nil, nil, fmt.Errorf("failed to compare typed patch to its reset: %v", err)
    }

    if !diff.IsSame() {
      // TODO: define good error message
      return nil, nil, fmt.Errorf("apply contained fields a user should not change:\n%s", diff)
    }
  }
  ...
}
```

- On `update`, the received object gets stripped of all `resetFields`.
Then the flow continues, but the user won't own the removed fields.

```go
# staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/structuredmerge.go @ Update()

func (f *structuredMergeManager) Update(liveObj, newObj runtime.Object, managed Managed, manager string) (runtime.Object, Managed, error) {
  ...
  if f.resetFields != nil {
    newObjTyped = newObjTyped.Remove(f.resetFields)
  }
  ...
}
```

##### Alternatives

We looked at a way to get the fields affected by status wiping without defining them separately.
Mainly by pulling the reset logic from the strategies `PrepareForCreate` and `PrepareForUpdate` methods into a new method `ResetFields` implementing an `ObjectResetter` interface.

This approach did not work as expected, because the strategy works on internal types while the FieldManager handles external api types.
The conversion between the two and creating the diff was complex and would have caused a notable amount of allocations.

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
- [x] Path element confersion will fail if a known qualifier's value is invalid. [link](https://github.com/kubernetes/kubernetes/blob/6b2e4682fe883eebcaf1c1e43cf2957dde441174/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/pathelement_test.go#L63-L84)
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

## Graduation Criteria

An alpha version of this is targeted for 1.14.

This can be promoted to beta when it is a drop-in replacement for the existing
kubectl apply, and has no regressions (which aren't bug fixes). This KEP will be
updated when we know the concrete things changing for beta.

This will be promoted to GA once it's gone a sufficient amount of time as beta
with no changes. A KEP update will precede this.

## Implementation History

* Early 2018: @lavalamp begins thinking about apply and writing design docs
* 2018Q3: Design shift from merge + diff to tracking field managers.
* 2019Q1: Alpha.

(For more details, one can view the apply-wg recordings, or join the mailing list
and view the meeting notes. TODO: links)

## Drawbacks

Why should this KEP _not_ be implemented: many bugs in kubectl apply will go
away. Users might be depending on the bugs.

## Alternatives

It's our belief that all routes to fixing the user pain involve
centralizing this functionality in the control plane.
