---
title: Context support in k8s.io/client-go
authors:
- "@mikedanese"
- "@maleck13"
owning-sig: sig-api-machinery
participating-sigs:
- sig-api-machinery
approvers:
- "@deads2k"
- "@liggitt"
- "@lavalamp"
creation-date: 2020-01-23
last-updated: 2020-01-23
status: implementable
---

# Context support in k8s.io/client-go

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [Cleanup Inconsistent Options Passing](#cleanup-inconsistent-options-passing)
  - [Client Signatures After Refactor](#client-signatures-after-refactor)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Graduation Criteria](#graduation-criteria)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

This proposal outlines a refactoring that would allow propagating of
[context](https://golang.org/pkg/context/) with
[client-go](https://github.com/kubernetes/client-go) with API calls made using
k8s.io/client-go clientsets.

## Motivation

When using client-go, external calls to the Kubernetes API are made in order to
manage resources. Adding support for context propagation along these request
paths enables clean and efficient implementation of cross cutting functionality.
This functionality initially will include:

- Support for request timeout and cancellation that frees the calling thread.
  Due to issues with authentication and authorization performance, this was
  something that was
  [retrofited](https://github.com/kubernetes/kubernetes/pull/83064) in various
  auth\* expansions, but is generally useful.
- Support for distributed tracing. Active
  [efforts](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/0034-distributed-tracing-kep.md#propagating-context-through-objects)
  by SIG Instrumentation to support distributed tracing are either blocked or
  would be substantially simplified with this enhancement.

### Goals

- Allow consumers of k8s.io/client-go to propagate context with API requests
  made using a clientset.
- Cleanup inconsistent \*Option passing in the public API of the clientset.

### Non-Goals

- Plumb context to client-go callsites in core Kubernetes. We believe that by
  making this feature available, this type of refactoring will happen over time.

## Proposal

The recommended approach is to pass the context with each action on a resource.
We will modify the signatures of all client interfaces to accept a
`context.Context` as the first argument.  This is idiomatic, explicit, and
produces the least error prone API.

## Design Details

This is an example change to `Pods.Get` implementation:

```go
func (c *pods) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.Pod, err error) {
	Result = &v1.Pod{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("pods").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}
```

Here we extend the signature to add `context.Context` as the first argument,
then attach the context to the request.

### Cleanup Inconsistent Options Passing

While we are requiring a migration for consumers of the client, now is a good
time to fix any issues we have with the API we are deprecating. While lack of
context support is the primary motivation, clientset methods do not consistently
support passing \*Options in the current API. Inconstencies we want to fix are:

1. Create does not accept a metav1.CreateOptions.
1. Update, UpdateStatus and Patch do not accept metav1.UpdateOptions.
1. Patch does not accept metav1.PatchOptions.
1. Delete methods accept metav1.DeleteOptions by pointer reference.

To fix these inconstencies, we will pass \*Options to all client methods and we
will migrate Delete methods to pass `metav1.DeleteOptions` by value.

### Client Signatures After Refactor

These are the signatures of the existing flunders client API:

```
func (c *flunders) Get(string, metav1.GetOptions) (*v1beta1.Flunder, error)
func (c *flunders) List(metav1.ListOptions) (*v1beta1.FlunderList, error)
func (c *flunders) Watch(metav1.ListOptions) (watch.Interface, error)
func (c *flunders) Create(*v1beta1.Flunder) (*v1beta1.Flunder, error)
func (c *flunders) Update(*v1beta1.Flunder) (*v1beta1.Flunder, error)
func (c *flunders) UpdateStatus(*v1beta1.Flunder) (*v1beta1.Flunder, error)
func (c *flunders) Delete(string, *metav1.DeleteOptions) error
func (c *flunders) DeleteCollection(*metav1.DeleteOptions, metav1.ListOptions) error
func (c *flunders) Patch(string, types.PatchType, []byte, ...string) (*v1beta1.Flunder, error)
```

After all proposed changes, the flunders client will look like:

```
func (c *flunders) Get(context.Context, string, metav1.GetOptions) (*v1beta1.Flunder, error)
func (c *flunders) List(context.Context, metav1.ListOptions) (*v1beta1.FlunderList, error)
func (c *flunders) Watch(context.Context, metav1.ListOptions) (watch.Interface, error)
func (c *flunders) Create(context.Context, *v1beta1.Flunder, metav1.CreateOptions) (*v1beta1.Flunder, error)
func (c *flunders) Update(context.Context, *v1beta1.Flunder, metav1.UpdateOptions) (*v1beta1.Flunder, error)
func (c *flunders) UpdateStatus(context.Context, *v1beta1.Flunder, metav1.UpdateOptions) (*v1beta1.Flunder, error)
func (c *flunders) Delete(context.Context, string, metav1.DeleteOptions) error
func (c *flunders) DeleteCollection(context.Context, metav1.DeleteOptions, metav1.ListOptions) error
func (c *flunders) Patch(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*v1beta1.Flunder, error)
```

### Risks and Mitigations

This is a breaking change that would likely require extensive changes to both
tests and implementation code throughout the impacted code bases when client-go
is upgraded.

Inside the Kubernetes codebase, we will refactor all callsites to pass
`context.TODO()` in the initial pass.

For consumers of cliensets outside of the Kubernetes codebase, we will take
point-in-time snapshots of the following packages.

```
k8s.io/apiextensions-apiserver/pkg/client/{clientset => deprecated}
k8s.io/client-go/{kubernetes => deprecated}
k8s.io/kube-aggregator/pkg/client/clientset_generated/{clientset => deprecated}
k8s.io/metrics/pkg/client/{clientset => deprecated}
```

These snapshots will live in tree for 2 releases. This allows consumers to
initially rewrite imports (e.g. with sed) and incrementally migrate code to the
new API. This gives consumers a 4 release window (as opposed to the standard 2
releases) to migrate to the new API before their client-go version falls out of
support.

### Graduation Criteria

This enhancement will be GA immediately.

## Alternatives

The client resources (Pods, Secrets etc...) each expose a resource specific
interface `PodInterface` for example. This is returned from a `Getter`
interface:

```go
type PodsGetter interface {
	Pods(namespace string) PodInterface
}
```

We can modify these resource interfaces and their underlying concrete types,
`pod` in this case, to add a `WithContext` method. This change would be
backwards compatible for most consumers of the API although it would break any
implementations of `PodsGetter` (e.g. in unit tests) outside of
k8s.io/client-go.

```go
// PodInterface has methods to work with Pod resources.
type PodInterface interface {
  	WithContext(ctx context.Context) PodInterface
    ...
}

type pods struct {
	client rest.Interface
	ns     string
	ctx    context.Context
}

// WithContext allows you to set a context that will be used by the underlying http request
func (c *pods) WithContext(ctx context.Context) PodInterface {
  	c.ctx = ctx
  	return c
}

```

To pass through this context, it would be necessary to change the underlying
client calls to accept the context. Example:

```go
func (c *pods) Get(name string, options meta_v1.GetOptions) (result *v1.Pod, err error) {
	result = &v1.Pod{}
	err = c.client.Get().
		Context(c.ctx).
		Namespace(c.ns).
		Resource("pods").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}
```

This would result in an API that would be interacted with like so if the
consumer wished to pass a context:

```go
   ctx := req.Context()
   pod, err := k8client.CoreV1().Pods(namespace).WithContext(ctx).Get(podName, ...)
```

This approach was rejected because:

- The idea behind using a context with a request is it is only meant to live as
  long as the initial request that created it. It is not meant to be stored. If
  we expose an error-prone API like the one outlined, the consumer would need to
  call `WithContext` on each request if a previous caller had set a context.
  This API allows broken code to compile, but will fail at runtime if multiple
  requests were made with the same client whose context have expired.
- Adding a method to a public interface is still a breaking change.
