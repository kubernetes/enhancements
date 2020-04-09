---
title: Less object serializations
authors:
  - "@wojtek-t"
owning-sig: sig-api-machinery
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
last-updated: 2019-10-10
status: implemented
see-also:
  - TODO
replaces:
  - n/a
superseded-by:
  - n/a
---

# Less object serializations

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
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [Bake-in caching objects into apimachinery](#bake-in-caching-objects-into-apimachinery)
  - [LRU cache](#lru-cache)
  - [Smart objects](#smart-objects)
<!-- /toc -->

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

First, we will extend [runtime.Encoder][] interface with the Identifier() method:
```go
// Identifier represents an identifier.
// Identitier of two different objects should be equal if and only if for every
// input the output they produce is exactly the same.
type Identifier string

type Encoder interface {
	...
	// Identifier returns an identifier of the encoder.
	// Identifiers of two different encoders should be equal if and only if for every input
	// object it will be encoded to the same representation by both of them.
	Identifier() Identifier
}
```

With that, we will introduce a new interface:
```go
// CacheableObject allows an object to cache its different serializations
// to avoid performing the same serialization multiple times.
type CacheableObject interface {
	// CacheEncode writes an object to a stream. The <encode> function will
	// be used in case of cache miss. The <encode> function takes ownership
	// of the object.
	// If CacheableObject is a wrapper, then deep-copy of the wrapped object
	// should be passed to <encode> function.
	// CacheEncode assumes that for two different calls with the same <id>,
	// <encode> function will also be the same.
	CacheEncode(id Identifier, encode func(Object, io.Writer) error, w io.Writer) error

	// GetObject returns a deep-copy of an object to be encoded - the caller of
	// GetObject() is the owner of returned object. The reason for making a copy
	// is to avoid bugs, where caller modifies the object and forgets to copy it,
	// thus modifying the object for everyone.
	// The object returned by GetObject should be the same as the one that is supposed
	// to be passed to <encode> function in CacheEncode method.
	// If CacheableObject is a wrapper, the copy of wrapped object should be returned.
	GetObject() Object
```

We will add support for `CacheableObject` for all existing Encoders. This is
basically as simple as:
```go
func (e *Encoder) Encode(obj Object, stream io.Writer) error {
	if co, ok := obj.(CacheableObject); ok {
		return co.CacheEncode(s.Identifier(), s.doEncode, stream)
	}
	return s.doEncode(obj, stream)
}

func (e *Encoder) doEncode(obj Object, stream io.Writer) error {
	// Existing encoder logic.
}
```

Necessary generic tests will be created to ensure it is supported correctly.

[runtime.Encoder]: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/runtime/interfaces.go#L52

With those (relatively mechanical) changes, we will introduce an internal type
in package `cacher` implementing both `runtime.Object` and `CacheableObject`
interfaces. The idea behind it is that it will be encapsulating the original
object and additionally it will be able to accumulate its serialized versions.
It will look like this:
```go
// serializationResult captures a result of serialization.
type serializationResult struct {
	// once should be used to ensure serialization is computed once.
	once sync.Once

	// raw is serialized object.
	raw []byte
	// err is error from serialization.
	err error
}

// metaRuntimeInterface implements runtime.Object and
// metav1.Object interfaces.
type metaRuntimeInterface interface {
	runtime.Object
	metav1.Object
}

// cachingObject is an object that is able to cache its serializations
// so that each of those is computed exactly once.
//
// cachingObject implements the metav1.Object interface (accessors for
// all metadata fields). However, setters for all fields except from
// SelfLink (which is set lately in the path) are ignored.
type cachingObject struct {
	lock sync.RWMutex

	// Object for which serializations are cached.
	object metaRuntimeInterface

	// serializations is a cache containing object`s serializations.
	// The value stored in atomic.Value is of type serializationsCache.
	// The atomic.Value type is used to allow fast-path.
	serializations atomic.Value
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

Based on the implementation, we observed the following gains:
- eliminating kube-apiserver unresponsiveness in case of write of
a single huge Endpoints object: [#75294#comment-472728088][]
- ~5% lower cpu-usage
- ~15% less memory allocations

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
- 2019-10-10: KEP updated to reflect the implementation.
- v1.17: Implemented

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
