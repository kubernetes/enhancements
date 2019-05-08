---
title: Graduate Ingress to GA
authors:
  - "@bowei"
owning-sig: sig-network
participating-sigs:
reviewers:
  - "@aledbf"
approvers:
  - "@thockin"
  - "@caseydavenport"
editor:
creation-date: 2018-01-25
last-updated: 2018-04-25
status: implementable
see-also:
replaces:
superseded-by:
---

# Graduate Ingress to GA

## Summary

* Move the Ingress resource from the current API group
  (extensions.v1beta1) to networking.v1beta1.
* Graduate the Ingress API with bug fixes to GA.

## Motivation

The `extensions` API group is considered deprecated.  Ingress is the
last non-deprecated API in that group.  All other types have been
migrated to other permanent API groups.  Such an API group migration
takes three minor version cycles (~9 months) to ensure
compatibility. This means any API group movement should be started
sooner rather than later.

The Ingress resource has been in a beta state for a *long* time (first
commit was in Fall 2015). While the interface [is not
perfect][survey], there are many [independent implementations][ingress-docs]
in active use.

We have a couple of choices (and non-choices, see appendix) for the
current resource:

1.  We can delete the current resource from extensions.v1beta1 in
    anticipation that an improved API can replace it.

1.  We can copy the API as-is (or with minor changes) into
    networking.v1beta1, preserving/converting existing data (following
    the same approach taken with all other extensions.v1beta1
    resources). This will allow us to start the cleanup of the
    extensions API group. This also prepares the API for GA.

Option 1 does not seem realistic in a short-term time frame (a new API
will need to be taken through design, alpha/beta/ga phases). At the
same time, there are enough users that the existing API cannot be
deleted out right.

In terms of moving the API towards GA, the API itself has been
available in beta for so long that it has attained defacto GA status
through usage and adoption (both by users and by load balancer /
ingress controller providers). Abandoning it without a full
replacement is not a viable approach.  It is clearly a useful API and
captures a non-trivial set of use cases.  At this point, it seems more
prudent to declare the current API as something the community will
support as a V1, codifying its status, while working on either a V2
Ingress API or an entirely different API with a superset of features.

### Goals

A detailed list of the changes being proposed is given in the Design
section below.

* Move Ingress to a permanent API group. (status: implemented)
* Make changes to the Ingress API be in a GA-ready state. (status:
  proposal).
  * Clean up the Ingress API (fix ambiguities, API spec bugs).
  * Promote commonly supported annotations to proper API fields.
  * Create a suite of conformance tests to validate existing
    implementations.
* Make Ingress GA. (status: proposal).

## Design

This section describes the API fixes proposed for GA.

### Summary of the proposed changes

1. Add path as a prefix and make regex support optional. The current spec states
   that the path is a regular expression and arbitrary regular expression, but
   support varies across providers. In addition, regex matching is not supported
   by many of the cloud provider implementations.
1. Fix API field naming:
   1. `spec.backend` should be called `spec.defaultBackend`.
1. Hostname wildcard matching. We currently allow for creation of
   `*.foo.com` and this seems to be a commonly suppported host match,
   but this is not part of the spec.
1. Specify healthcheck behavior and be able to configure the healthcheck path
   and timeout.
1. Improve the Ingress status field to be able to include additional
   information. The current status currently only contains provisions the IP
   address of the load balancer.
1. Formalize the Ingress class annotation into a field and an associated
   `IngressClass` resource.
1. Add support for non-Service Backend types.

Note: this is a proposed list. Please discuss in the comments.

### Path as a prefix

The [current APIs][ingress-api] state that the path is a regular expression
using the [POSIX IEEE Std 1003.1 standard][posix-regex]. However, this is not
consistent with the syntax supported by any of the common proxy vendors:

| Platform | Syntax                   |
|----------|--------------------------|
| nginx    | [PCRE][nginx-re]         |
| haproxy  | [PCRE/PCRE2][haproxy-re] |
| envoy    | [ECMAscript][envoy-re]   |

Among cloud providers, there is also inconsistent levels of support
for regular expression-based path matching. See the load-balancer
documentation for [aws][aws-re], [gcp][google-re], [azure][azure-re].

It is also the case that our [documentation][ingress-docs] (and most
Ingress providers) treats the path match as a prefix match. For
example, a narrow interpretation of the specification would require
all paths to end with `".*$"`.

#### Proposal

1. Clarify behavior of the existing path regex match and identify that use of
   path regex is non-portable.
1. Support prefix matching behavior as the portable alternative.

Change specification of `ingress.spec.rules.http.paths.path` from the current
text to indicate the non-portability of the value:

> The matching behavior of Path is implementation specific. Path is either a
> prefix or regular expression match.

Add a field `ingress.spec.rules.http.paths.pathPrefix`:

```golang
type HTTPIngressPath struct {
  ...
  // PathPrefix matches the HTTP request if the request path begins with
  // this string. For example, PathPrefix = "/foo" will match "/foo", "/foox",
  // "/foo/bar".
  //
  // Both Path and PathPrefix cannot be set (non-empty) at the same time.
  PathPrefix string
}
```

Interoperability between v1beta and v1 will work as follows:

v1beta1 gets the `PathPrefix" field. v1 gets an annotation
`ingress.kubernetes.io/non-portable-path-semantics`. If we create a v1 object
from a v1beta1 where `path` was used, we set the annotation. We keep both Path
and PathPrefix in the API.

<!--
1) v1beta1 gets a "PathPrefix" field, specced as a strict prefix, no
trailing "*" or ".*" needed.  Document that Path is under-specified
and implementation-specific.  Document that all implementations are
expected to support PathPrefix with consistent semantics.  Path and
PathPrefix can be mutually exclusive.  v1 gets either Path or
PathPrefix (debatable) and an annotation
"ingress.kubernetes.io/beta-path" which maps the new field to the old.
That satisfies round-trip requirement but is kind of horrible.

2) v1beta1 gets a "PathPrefix" field as above.  v1 gets an annotation
"ingress.kubernetes.io/beta-path-semantics".  If we create a v1 object
from a v1beta1 where `path` was used, we set the annotation.
Validation logic will check the annotation to decide how to interpret
and validate paths.  We'll have to keep it for a long while.  Maybe we
can prevent new objects from being created with that annotation?  Now
what if implementations want to offer "extended" semantics here?  Same
validation problem.

3)  v1beta1 gets a "PathPrefix" field as above.  v1 gets an annotation
"ingress.kubernetes.io/non-portable-path-semantics".  If we create a
v1 object from a v1beta1 where `path` was used, we set the annotation.
We keep it forever.

4) Same as 2 but we just don't validate Path at all.
-->

#### Rejected alternatives

##### Portable regex behavior

The safest route for specifying the regex would be to state a limited
subset that can be used in a portable way. Any expressions outside of
the subset will have implementation specific behavior.

Regular expression subset (derived from [re2][re2-syntax] syntax page)

| Expression | description             |
|------------|-------------------------|
| `.`        | any character           |
| `[xyz]`    | character class         |
| `[^xyz]`   | negated character class |
| `x*`       | 0 or more x's           |
| `x+`       | 1 or more x's           |
| `xy`       | x followed by y         |
| `x|y`      | x or y (prefer x)       |
| `(abc)`    | grouping                |

Maintaining a regular expression subset is likely not worth the complexity and
is likely impossible across the [many implementations][regex-survey].

### API field naming

These are straightforwarding one-to-one renames to for consistency.

| v1beta1 field  | ga field              | rationale                   |
|----------------|-----------------------|-----------------------------|
| `spec.backend` | `spec.defaultBackend` | Explicitly mentions default |

### Hostname wildcards

Most platforms support wildcarding for hostnames, e.g. syntax such as
`*.foo.com` matches names `app1.foo.com`, `app2.foo.com`. The current spec
states that `spec.rules.host` must be a FQDN of a network host. 

#### Proposal

This proposal would be a limited support for adding a single wildcard `*` as the
first host label.

The `IngressRule.Host` comment would be changed to:

> `Host` can be a "precise host" which is an fully-qualified domain name without
> the terminating dot of a network host (e.g. "foo.bar.com") or a "wildcard
> host" domain name prefixed with a single wildcard label ("*.foo.com").
> Requests will be matched against the `Host` field in the following way:
> 
> If `Host` is precise, the request matches the rule if the http host header
> is equal to `Host`.
> 
> If `Host` is a wildcard, then request matches the rule if the http host header
> is to equal to the suffix (removing the first label) of the wildcard rule.
> E.g. wildcard "*.foo.com" matches "bar.foo.com" because they share an equal
> suffix "foo.com".

### Healthchecks

The current spec does not have any provisions to customize appropriate
healtchecks for referenced backends. Many users already have a healthcheck
URLthat is lightweight and different from the HTTP root (e.g. `/`).

One option that has been explored is to infer the healthcheck URL from the
Readiness probes on the Pods of the Service. This method has proven to be quite
unstable: Every Pod in a Service can have a different Readiness probe definition
and it's not clear which one should be used. Furthermore, the behavior is quite
implicit and creates action-at-a-distance relationship between the Ingress and
Pod resources.

#### Proposal

Add the following fields to `IngressBackend`:

```golang
type IngressBackend struct {
  ...
  // Healthcheck defines custom healthcheck for this backend.
  // +optional
  Healthcheck *IngressBackendHealthcheck
}

type IngressBackendHealthcheck struct {
  // HTTP defines healthchecks using the HTTP protocol.
  HTTP *IngressBackendHTTPHealthcheck
}

// IngressBackendHTTPHealthcheck is a healthcheck using the HTTP protocol.
type IngressBackendHTTPHealthcheck struct {
  // Host header to send when healthchecking. If empty, the host header will be
  // implementation specific.
  Host string
  // Path to use for the HTTP healthcheck. If empty, the root '/' path will be
  // used for healthchecking.
  Path string
  // TimeoutSeconds for the healthcheck. Failure to respond with a success code
  // within TimeoutSeconds will be counted towards the FailureThreshold.
  TimeoutSeconds int
  // FailureThreshold is the number of consecutive failures necesseary to
  // indicate a backend failure.
  FailureThreshold int
}
```

If `Healthcheck` is nil, then the implementation default healthcheck will be
configured, healthchecking the root `/` path. If `Healthcheck` is specfied,
then the backend health will be checked using the parameters listed above. 

### Status

!!! TODO

### Ingress class

The `kubernetes.io/ingress.class` annotation is required for interoperation
between various Ingress providers. As support for this annotation is universal,
this concept should be promoted to an actual field.

#### Proposal

Promoting the annotation as is as an opaque string the most direct path, but
precludes any future enhancements to the concept.

An alternative is to create a new resource `IngressClass` to take its place.
This resource will serve a couple of purposes:

* Define the set of valid classes available to the user. Gives operators control
  over allowed classes.
* Allow us to evolve the API to express concepts such a levels of service
  associated with a given Ingress controller.

Add a field to `ingress.spec`:
```golang
type IngressSpec struct {
  ...
  // Class references an IngressClass resource in kube-system. This is used
  // by the cluster Ingress controllers to determine which controller operates
  // on this resource.
  Class string
  ...
}
```

```golang
// IngressClass represents the class of the Ingress, referenced by the
// ingress.spec.
type IngressClass struct {
  metav1.TypeMeta
  metav1.ObjectMeta
  Spec IngressClassSpec
}

type IngressClassSpec struct {
  // This is currently empty.
}
```

### Alternative backend types

The Ingress resource is an L7 description of a composite set of services. It
current supports Kubernetes Services as a backends. However, there are many use
cases where a portion of the HTTP requests could be routed to a different kind
of resource. For example, serving content from an object storage
([S3][s3-backend], [GCS][gcs-backend]) is a commonly requested feature.

At the same time, we do not expect to enumerate all possible backends that could
arise, nor do we expect that naming of the resources will be uniform in schema,
parameters etc. Similarly, many of the resources will be platform specific.

#### Proposal

Add a field to the `IngressBackend` struct with an object reference:

```golang
type IngressBackend struct {
  // Specifies the name of the referenced service. The service must exist in
  // the same namespace as the Ingress object. The API server will
  // automatically populate the Resource field below.
  // +optional
  ServiceName string

  // Specifies the port of the referenced service. If unspecfied and the
  // ServiceName is non-empty, the Service must expose a single port.
  // +optional
  ServicePort intstr.IntOrString

  // Resource is an ObjectRef to another Kubernetes resource in the namespace
  // of the Ingress object. If Resource is a Service, then ServiceName will
  // populated to match the Resource. If both ServiceName and Resource are
  // specified, then they must reference the same Service object.
  // +optional
  Resource v1.TypedLocalObjectReference
}
```

Support for non-`Service` type `Resource`s will be implementation specific. This
can take advantage of standardized way of reference external object stores. As a
sketch, an object bucket can be named with a CRD:

```golang
type Bucket struct {
  metav1.TypeMeta
  metav1.ObjectMeta
  Spec BucketSpec
}

type BucketSpec struct {
  Bucket string
  Path   string 
}
```

The associated `IngressBackend` referencing the bucket would be:

```yaml
backend:
  resource:
    apiGroup: bucket.io
    kind: bucket
    name: my-bucket
```

## Proposed roadmap

### 1.14

* [x] Copy the Ingress API to `networking.k8s.io/v1beta1` (preserving
  existing data and round-tripping with the extensions Ingress API,
  following the approach taken for all other `extensions/v1beta1`
  resources).
* [ ] Develop a set of planned changes and GA graduation criteria with
  sig-network (intent is to target a minimal set of bugfixes and
  non-breaking changes)
* [ ] Announce `extensions/v1beta1` Ingress as deprecated (and
      announce plan for GA)

#### Test plan

* Copy existing Ingress tests, changing the resource type to the new
  group. Keep existing tests as is.

#### Test plan

* Copy existing Ingress tests, changing the resource type to the new group. Keep
  existing tests as is.

### 1.15

* [ ] Update API server to persist in networking.k8s.io/v1beta1
* [ ] Update in-tree controllers, examples, and clients to target
  `networking.k8s.io/v1beta1`
* [ ] Update Ingress controllers in the kubernetes org to target
  `networking.k8s.io/v1beta1`
* [ ] Update documentation to recommend new users start with
  networking.k8s.io/v1beta1, but existing users stick with
  `extensions/v1beta1` until `networking.k8s.io/v1` is available.
* [ ] Update documentation to reference `networking.k8s.io/v1beta1`
* [ ] Meet graduation criteria and promote API to `networking.k8s.io/v1`
* [ ] Announce `newtorking.k8s.io/v1beta1` Ingress as deprecated

### 1.16

* [ ] Update API server to persist in `networking.k8s.io/v1`.
* [ ] Update in-tree controllers, examples, and clients to target
  `networking.k8s.io/v1`.
* [ ] Update Ingress controllers in the kubernetes org to target
  `networking.k8s.io/v1`.
* [ ] Update documentation to reference `networking.k8s.io/v1`.
* [ ] Evangelize availability of v1 Ingress API to out-of-org Ingress
      controllers

### 1.18

* [ ] Remove ability to serve `extensions/v1beta1` and
  `networking.k8s.io/v1beta1` Ingress resources (preserve ability to
  read existing `extensions/v1beta1` Ingress objects from storage and
  serve them via the `networking.k8s.io/v1` API)

## Graduation Criteria

### API group move to `networking.k8s.io/v1beta1`

* [ ] 1.14: Ingress API exists and has parity with existing
  `extensions/v1beta1` API
* [ ] 1.14: `extensions/v1beta1` Ingress tests are replicated against
  `networking.k8s.io`
* [ ] 1.15: all in-tree use and in-org controllers switch to
  `networking.k8s.io` API group
* [ ] 1.15: documentation and examples are updated to refer to
  networking.k8s.io API group `networking.k8s.io/v1`

### GA

* TODO

## Implementation History

* 1.14: Copied Ingress API to the networking API group.

## Alternatives

See motivation section.

## Appendix

### Non-options

One suggestion was to move the API into a new API group, defined as a
CRD.  This does not work because there is no way to do round-trip of
existing Ingress objects to a CRD-based API.

### Potential pre-GA work

Note: these items are NOT the main focus of this KEP, but recorded
here for reference purposes. These items came up in discussions on the
KEP (roughly sorted by practicality):

* Spec path as a prefix, maybe as a new field
* Rename `backend` to `defaultBackend` or something more obvious
* Be more explicit about wildcard hostname support (I can create *.bar.com but
  in theory this is not supported)
* Add health-checks API
* Specify whether to accept just HTTPS or also allow bare HTTP
* Better status
* Formalize Ingress class
* Reference a secret in a different namespace?  Use case: avoid copying wildcard
  certificates (generated with cert-manager for instance)
* Add non-required features (levels of support)
* Some way to have backends be things other than a service (e.g. a GCS bucket)
* Some way to restrict hostnames and/or URLs per namespace
* HTTP to HTTPS redirects
* Explicit sharing or non-sharing of external IPs (e.g. GCP HTTP LB)
* Affinity
* Per-backend timeouts
* Backend protocol
* Cross-namespace backends

[aws-re]: https://docs.aws.amazon.com/elasticloadbalancing/latest/application/listener-update-rules.html
[azure-re]: https://docs.microsoft.com/en-us/azure/application-gateway/application-gateway-create-url-route-portal
[envoy-re]: https://github.com/envoyproxy/envoy/blob/v1.10.0/api/envoy/api/v2/route/route.proto#L334
[gcs-backend]: TODO
[google-re]: https://cloud.google.com/load-balancing/docs/https/url-map-concepts
[haproxy-re]: http://git.haproxy.org/?p=haproxy-1.9.git;a=blob;f=Makefile;h=0814440e48d57ae53f058ffb3f233c80b63871f2;hb=HEAD#l17
[ingress-api]: https://github.com/kubernetes/api/blob/release-1.14/networking/v1beta1/types.go#L170
[ingress-docs]: https://kubernetes.io/docs/concepts/services-networking/ingress/#ingress-controllers
[nginx-re]: http://nginx.org/en/docs/http/server_names.html#regex_names
[posix-regex]: https://www.boost.org/doc/libs/1_38_0/libs/regex/doc/html/boost_regex/syntax/basic_extended.html
[re2-syntax]: https://github.com/google/re2/wiki/Syntax
[s3-backend]: TODO
[survey]: https://github.com/bowei/k8s-ingress-survey-2018
[regex-survey]: https://en.wikipedia.org/wiki/Comparison_of_regular_expression_engines