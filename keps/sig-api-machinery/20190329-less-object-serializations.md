---
title: Less object serializations
authors:
  - "@wojtek-t"
owning-sig: sig-apimachinery
participating-sigs:
  - sig-scalability
reviewers:
  - "@jpbetz"
  - "@justinsb"
  - "@smarterclayton"
approvers:
  - "@deads2k"
  - "@lavalamp"
creation-date: 2019-03-27
last-updated: 2019-07-19
status: implementable
see-also:
  - TODO
replaces:
  - n/a
superseded-by:
  - n/a
---

# Less object serializations

## Table of Contents

* [Less object serializations](#less-object-serializations)
   * [Table of Contents](#table-of-contents)
   * [Release Signoff Checklist](#release-signoff-checklist)
   * [Summary](#summary)
   * [Motivation](#motivation)
      * [Goals](#goals)
      * [Non-Goals](#non-goals)
   * [Proposal](#proposal)
      * [Risks and Mitigations](#risks-and-mitigations)
   * [Design Details](#design-details)
      * [Test Plan](#test-plan)
      * [Graduation Criteria](#graduation-criteria)
      * [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
      * [Version Skew Strategy](#version-skew-strategy)
   * [Implementation History](#implementation-history)
   * [Drawbacks](#drawbacks)
   * [Alternatives](#alternatives)

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

Scalability and performance of kube-apiserver is crucial for scalability
of the whole Kubernetes cluster. Given that kube-apiserver is cpu-intensive
process, scaling a single instance of it translates to optimizing amount
of work is needed to process a request (cpu cycles and amount of allocated
memory, as memory management is significant part of work done be
kube-apiserver).

This proposal is aiming to significantly reduce amount of work spent on
serializing objects as well as amount of allocated memory to process that.

## Motivation

Running different types of scalability tests and analyzing large production
clusters proves that large number of watches watching the same set of objects
may cause significant load on kube-apiserver. An extreme example of it is
[#75294][], where creation of a single large Endpoints object (almost 1MB of
size, due to 5k pods backing it) in 5k-node cluster can completely overload
kube-apiserver for 5 seconds.

The main reason for that is that for every watcher (Endpoints are being watched
by kube-proxy running on every one) kube-apiserver independently serializes
(which also requires deep-copy) every single object being send via this watch.

While this problem is extremely visible for watch, the situation looks the same
for regular GET/LIST operations - reading the same object N times will result
in serializing that N times independently.

This proposal presents a solution for that problem.

[#75294]: https://github.com/kubernetes/kubernetes/issues/75294

### Goals

- Reduce load on kube-apiserver and number of memory allocations, by avoiding
serializing the same object multiple times for different watchers.

### Non-Goals

- Change overall architecture of the system, by changing what data is being
read/watched by different components.

## Proposal

This proposal does not introduce any user-visible changes - the proposed changes
are purely implementation details of kube-apiserver.

The first observation is that a given object may be serialized to multiple
different formats, based on it's:
- group and version
- subresource
- output media type

However, group, version and subresource are reflected in the `SelfLink` of
returned object. As a result, for a given (potentially unversioned) object,
we can identify all its possible serializations by (SelfLink, media-type)
pairs, and that is what we will do below.

We propose to extend [WithVersionEncoder][] by adding ObjectConvertor to it:
```
type WithVersionEncoder struct {
	Version GroupVersioner
	Encoder
	ObjectConvertor
	ObjectTyper
}
```

On top of that, we propose introducing `CustomEncoder` interface:
```
type CustomEncoder interface {
	InterceptEncode(encoder WithVersionEncoder, w io.Writer) error
}
```

With that, we will change existing serializers (json, protobuf and versioning),
to check if the to-be-serialized object implements that interface and if so simply
call its `Encode()` method instead of using existing logic.

[WithVersionEncoder]: https://github.com/kubernetes/kubernetes/blob/990ee3c09c0104cc1045b343040fe76082862d73/staging/src/k8s.io/apimachinery/pkg/runtime/helper.go#L215

With those (very local and small) changes, we will introduce an internal type
in package `cacher` implementing both `runtime.Object` and `CustomEncoder`
interfaces. The idea behind it is that it will be encapsulating the original
object and additionally it will be able to accumulate its serialized versions.
It will look like this:
```
// TODO: Better name is welcome :)
type CachingObject struct {
	// Object is the object (potentially in the internal version)
	// for which serializations should be cached for future reuse.
	Object runtime.Object

	// FIXME: We may want to change that during performance experiments
	// e.g. to use sync.Map or slice or something different, also to
	// allow some fast-path.
	lock sync.Mutex
	Versioned map[runtime.GroupVersioner]*CachingVersionedObject
}

// TODO: Better name is welcome :)
type CachingVersionedObject struct {
	// Ojbect is the versioned object for which serializations
	// should be cached.

	// FIXME: We may want to change that during performance experiments
	// e.g. to use sync.Map or slice or something different, also to
	// allow some fast-path.
	lock sync.Mutex
	serialization map[cachingKey]*serializationResult
}

type cachineKey struct {
	// encoder is a proxy for mediaType - given we don't have access to it
	// (Serializer interface doesn't expose it) and we want to minimize
	// changes to apimachinery, we take runtime.Encoder which is a singleton
	// for a given mediaType in apiserver.
	encoder runtime.Encoder
	// selfLink of the serialized object identifying endpoint
	// (e.g. group, version, subresource)
	selfLink string
}

// TODO: Better name is welcome :)
type serializationResult struct {
	once sync.Once

	// raw is serialized object.
	raw []byte
	// err is error from serialization.
	err error
}
```

In the initial attempt, watchCache when receiving an event via watch from
etcd will be opaquing it into `CachingObject` and operating on object of
that type later.

That means that we won't have gains from avoid serialization for any GET/LIST
requests server from cache as well as for `init event` that we process when
initializing a new watch, but that seems good enough for the initial attempt.
The obvious gain from it is that the memory used for caching is used only
for a very short period of time (when delivering this watch to watchers) and
quickly released, which means we don't need to be afraid about increased
memory usage.
We may want to revisit that decision later if we would need more gains
from avoiding serialization and deep-copies of objects in watchcache.

Note that based on a [POC][] (slightly different that above design),
the gains of implementing it include:
- eliminating kube-apiserver unresponsiveness in case of write of
a single huge Endpoints object: [#75294#comment-472728088][]
- ~7% lower cpu-usage
- XX% less memory allocations

[POC]: https://github.com/kubernetes/kubernetes/pull/60067
[#75294#comment-472728088]: https://github.com/kubernetes/kubernetes/issues/75294#issuecomment-472728088

### Risks and Mitigations

The proposal doesn't introduce any user visible change - the only risk is
related to bugs in implementation. Even though, the serialization code is
widely user by all end-to-end tests and bugs should be catched by those
or unit tests of newly added logic, we will try to mitigate the risk by
introducing a feature gate and hiding the logic of using the newly introduced
object behind this feature gate.

## Design Details

### Test Plan

* Unit tests covering all corner cases of logic of newly introduced objects.
* Unit test to detect races of newly introduced objects
* Regular e2e tests are passing.

### Graduation Criteria

* All existing e2e tests are passing.
* Scalability tests confirm gains of that change.

We're planning to enable this feature by default, but a feature gate to
disable it is the mitigation strategy if bugs will be discovered after
release.

### Upgrade / Downgrade Strategy

This feature doesn't change any persistent state of the cluster, just the
in-memory representation of objects, upgrade/downgrade strategy is not
relevant to this feature.

### Version Skew Strategy

The feature is only changing in-memory representation of objects only in
kube-apiserver, so version skew strategy is not relevant.

## Implementation History

- 2019-03-27: KEP Created
- 2019-07-18: KEP Merged
- 2019-07-19: KEP updated with test plan and moved to implementaable state.

## Alternatives

### Bake-in caching objects into apimachinery
We considered making objects above part of apimachinery.

Pros:
- Expose ability to use it for others

Cons:
- Complicated code hits apimachinery

### LRU cache
We considered using simple LRU cache to store serialized objects.

Pros:
- performance gains also for reads served from etcd (though these
doesn't seem to be huge based on experiments)

Cons:
- potentially significant point of contention
- no-control over what is still cached (e.g. for frequently changing
resources, we still keep them in cache, even if they will never be
served again)

### Smart objects
We also considered using `smart objects` - an object that carries the
serialized format of object from etcd with itself.

Pros:
- very clear encapsulation

Cons:
- We need an ability to in-place add fields to serialized object
(i.e. SelfLink) - very tricky and error-prone
- This doesn't work across different (group, version) pairs. As
an example, if at some point we will be migrating `Endpoints`
object to the new API, this will stop working for the whole
migration period (i.e. at least one release).
