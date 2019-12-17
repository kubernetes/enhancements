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

## Table of contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Design](#design)
  - [Summary of the proposed changes](#summary-of-the-proposed-changes)
    - [Potential features for post V1](#potential-features-for-post-v1)
  - [Path as a prefix](#path-as-a-prefix)
    - [Paths proposal](#paths-proposal)
      - [Defaults](#defaults)
      - [Path matching semantics](#path-matching-semantics)
      - [<code>Exact</code> match](#-match)
      - [<code>Prefix</code> match](#-match-1)
      - [<code>ImplementationSpecific</code> match](#-match-2)
      - [Examples](#examples)
  - [<code>backend</code> to <code>defaultBackend</code>](#-to-)
  - [Hostname wildcards](#hostname-wildcards)
    - [Hostname proposal](#hostname-proposal)
      - [Hostname match examples](#hostname-match-examples)
  - [Status](#status)
  - [Ingress class](#ingress-class)
    - [Ingress class proposal](#ingress-class-proposal)
      - [Interoperability with previous annotation](#interoperability-with-previous-annotation)
  - [Alternative backend types](#alternative-backend-types)
    - [Backend types proposal](#backend-types-proposal)
      - [Backend types examples](#backend-types-examples)
      - [Supporting custom backends (non-normative)](#supporting-custom-backends-non-normative)
- [Proposed roadmap](#proposed-roadmap)
  - [1.14](#114)
    - [Test plan](#test-plan)
  - [1.15](#115)
  - [1.16](#116)
  - [1.17](#117)
  - [1.18](#118)
- [Graduation Criteria](#graduation-criteria)
  - [API group move to <code>networking.k8s.io/v1beta1</code>](#api-group-move-to-)
  - [GA](#ga)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
- [Appendix](#appendix)
  - [Design discussions](#design-discussions)
  - [Non-options](#non-options)
  - [Future design: Healthchecks](#future-design-healthchecks)
    - [Healthchecks proposal](#healthchecks-proposal)
  - [Potential pre-GA work](#potential-pre-ga-work)
  - [Rejected designs](#rejected-designs)
    - [Portable regex for Path](#portable-regex-for-path)
<!-- /toc -->

## Summary

- Move the Ingress resource from the current API group
  (extensions.v1beta1) to networking.v1beta1.
- Graduate the Ingress API with bug fixes to GA.

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

1. We can delete the current resource from extensions.v1beta1 in
  anticipation that an improved API can replace it.

1. We can copy the API as-is (or with minor changes) into
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

- Move Ingress to a permanent API group. (status: implemented)
- Make changes to the Ingress API be in a GA-ready state. (status:
  proposal).
  - Clean up the Ingress API (fix ambiguities, API spec bugs).
  - Promote commonly supported annotations to proper API fields.
  - Create a suite of conformance tests to validate existing
    implementations.
- Make Ingress GA. (status: proposal).

## Design

This section describes the API fixes proposed for GA.

### Summary of the proposed changes

1. Add path as a prefix and make regex support optional. The current spec
  states that the path is a regular expression, but support for the flavor
  defined in the spec varies across providers. In addition, regex matching
  is not supported by many popular provider implementations.
1. Fix API field naming:
   1. `spec.backend` should be called `spec.defaultBackend`.
1. Hostname wildcard matching. We currently allow for creation of
   `*.foo.com` and this seems to be a commonly supported host match,
   but this is not part of the spec.
1. Formalize the Ingress class annotation into a field and an associated
   `IngressClass` resource.
1. Add support for non-Service Backend types.

#### Potential features for post V1

These are features that were discussed but not part of this discussion:

1. (**POST GA**) Specify healthcheck behavior and be able to configure
   the healthcheck path and timeout.
1. (**POST GA**) Improve the Ingress status field to be able to
   include additional information. The current status currently only
   contains the provisioned IP address(es) of the load balancer.

### Path as a prefix

The [current APIs][ingress-api] state that the path is a regular expression
using the [POSIX IEEE Std 1003.1 standard][posix-regex]. However, this is not
consistent with the syntax supported by any of the common proxy vendors:

| Platform | Syntax                   |
|----------|--------------------------|
| nginx    | [PCRE][nginx-re]         |
| haproxy  | [PCRE/PCRE2][haproxy-re] |
| envoy    | [ECMAscript][envoy-re]   |
| skipper  | re2                      |

Among cloud providers, there is also inconsistent levels of support
for regular expression-based path matching. See the load-balancer
documentation for [AWS][aws-re], [GCP][google-re], [Azure][azure-re],
[Skipper][skipper-link].

[skipper-link]: https://github.com/zalando/skipper

It is also the case that our [documentation][ingress-docs] (and most
Ingress providers) treats the path match as a prefix match. For
example, a narrow interpretation of the specification would require
all paths to end with `".*$"`.

A detailed discussion of this issue can be found
[here](https://github.com/kubernetes/ingress-nginx/issues/555).

#### Paths proposal

1. Explicitly state the match mode of the path.
1. Support the existing implementation-specific behavior.
1. Support a portable prefix match and future expansion of behavior.

Add a field `ingress.spec.rules.http.paths.pathType` to indicate
the desired interpretation of the meaning of the `path`:

```golang
 type HTTPIngressPath struct {
   ...
  // Path to match against. The interpretation of Path depends on
  // the value of PathType.
  //
  // Defaults to "/" if empty.
  //
  // +Optional
  Path string

  // PathType determines the interpretation of the Path
  // matching. PathType can be one of the following values:
  //
  // Exact  - matches the URL path exactly.
  //
  // Prefix - matches based on a URL path prefix split
  // by '/'. [insert description of semantics described below]
  //
  // ImplementationSpecific - interpretation of the Path
  // matching is up to the IngressClass. Implementations
  // are not required to support ImplementationSpecific matching.
  //
  // +Optional
  PathType string
  ...
 }
 ```

V1 validation

Note: default value are permitted between API versions
([reference][api-conv-versions]).

[api-conv-versions]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#defaulting

##### Defaults

The `PathType` field will default to a value of `ImplementationSpecific` to
provide backwards compatibility. 

##### Path matching semantics

For `Prefix` and `Exact` paths:

1. Let `[p_1, p_2, ..., p_n]` be the list of Paths for a specific host.
1. Every Path `p_i` must be syntactically valid:
    1. Must begin with the `'/'` character (relative paths are not allowed by [RFC-7230][rfc7230]).
    1. Must not contain consecutive `'/'` characters (e.g. `/foo///`, `//`).
1. A trailing `'/'` character in the Path is ignored, e.g. `/abc` and `/abc/`
   specify the same match. The rest of the discussion below assumes
   trailing `'/'` are removed from the paths.
1. If there is more than one potential match:
   1. `Exact` match is preferred to a `Prefix` match.
   1. For multiple prefix matches, the longest Path `p_i` will be the
      matching path.
   1. If an `ImplementationSpecific` match exists in the spec, then the
      preference depends on the implementation.
1. If there is no matching path, then the `defaultBackend` for the host will be
   used.
1. If there is not a match for the host, then the overall `defaultBackend` for
   the Ingress will be selected.

##### `Exact` match

Path must be exactly the same as the request path.

##### `Prefix` match

Matching is done on a path element by element basis. A path element refers
is the list of labels in the path split by the `'/'` separator. A request
is a match for path `p` if every `p` is an element-wise prefix of `p` of the
request path. Note that if the last element of the path is a substring of
the last element in request path, it
is *not* a match (e.g. `/foo/bar` matches `/foo/bar/baz`, but does not
match `/foo/barbaz`).

##### `ImplementationSpecific` match

Interpretation of the implementation-specific behavior is defined by the
associated `IngressClass`. Implementations are not required to support this
type of match. If the match type is not supported, then the controller MAY
raise this error as an asynchronous Event to the user.

##### Examples

| Kind   | Path                            | Request path                  | Matches?                           |
|--------|---------------------------------|-------------------------------|------------------------------------|
| Exact  | `/foo`                          | `/foo`                        | Yes                                |
| Exact  | `/foo`                          | `/bar`                        | No                                 |
| Prefix | `/`                             | (all paths)                   | Yes                                |
| Prefix | `/aaa/bbb`                      | `/aaa/bbb`                    | Yes                                |
| Prefix | `/aaa/bbb/`                     | `/aaa/bbb`                    | Yes, ignores trailing slash        |
| Prefix | `/aaa/bbb`                      | `/aaa/bbb/`                   | Yes,  matches trailing slash       |
| Prefix | `/aaa/bbb`                      | `/aaa/bbb/ccc`                | Yes, matches subpath               |
| Prefix | `/aaa/bbb`                      | `/aaa/bbbxyz`                 | No, does not match string prefix   |
| Prefix | `/`, `/aaa`, `/aaa/bbb`         | `/aaa/ccc`                    | Yes, matches `/aaa` prefix     |
| Prefix | `/`, `/aaa`, `/aaa/bbb`         | `/aaa/bbb`                    | Yes, matches `/aaa/bbb` prefix, not `/` or `/aaa` |
| Prefix | `/`, `/aaa`, `/aaa/bbb`         | `/ccc`                        | Yes, matches `/` prefix            |
| Prefix | `/aaa`                          | `/ccc`                        | No, uses default backend           |
| Mixed  | `/foo` (Prefix), `/foo` (Exact) | `/foo`                        | Yes, prefers Exact                 |

[rfc7230]: https://tools.ietf.org/html/rfc7230#section-5.3.1

### `backend` to `defaultBackend`

These are straightforward one-to-one renames for better semantic
meaning.

| v1beta1 field  | v1                    | rationale                   |
|----------------|-----------------------|-----------------------------|
| `spec.backend` | `spec.defaultBackend` | Explicitly mentions default |

Add comment clarifying behavior:

> It is up to the controller to resolve conflicts between the defaultBackend's
> for multiple Ingress definitions that are served from the same
> VIP if this is possible.

### Hostname wildcards

Most platforms support wildcards for host names, e.g. syntax such as
`*.foo.com` matches names `app1.foo.com`, `app2.foo.com`. The current
spec states that `spec.rules.host` must be an exact FQDN match of a
network host.

#### Hostname proposal

Add support for a single wildcard `*` as the first label in the hostname.

The `IngressRule.Host` specification would be changed to:

> `Host` can be "precise" which is an domain name without the
> terminating dot of a network host (e.g. "foo.bar.com") or
> "wildcard", which is a domain name prefixed with a single wildcard
> label (e.g. `"*.foo.com"`).
>
> Requests will be matched against the `Host` field in the following
> way:
>
> If `Host` is precise, the request matches this rule if the http host
> header is equal to `Host`.
>
> If `Host` is a wildcard, then the request matches this rule if the
> http host header is to equal to the suffix (removing the first
> label) of the wildcard rule.
>
> - The wildcard character `'*'` must appear by itself as the first
>   DNS label and matches only a single label.
> - You cannot have a wildcard label by itself (e.g. `Host == "*"`).

##### Hostname match examples

- `"*.foo.com"` matches `"bar.foo.com"` because they share an the same
   suffix `"foo.com"`.
- `"*.foo.com"` does not match `"aaa.bbb.foo.com"` as the wildcard only
   matches a single label.
- `"*.foo.com"` does not match `"foo.com"`, as the wildcard must match a
   single label.

Note: label refers to a "DNS label", i.e. the strings separated by the dots "."
in the domain name.

### Status

As this is strictly additive, this could be punted to post-GA to reduce
the size of the change.

### Ingress class

The `kubernetes.io/ingress.class` annotation is required for selecting between
multiple Ingress providers. As support for this annotation is universal, this
concept should be promoted to an actual field.

#### Ingress class proposal

Promoting the annotation as it is currently defined as an opaque string is the
most direct path but precludes any future enhancements to the concept.

An alternative is to create a new resource `IngressClass` to take its place.
This resource will serve a couple of purposes:

- Define the set of valid classes available to the user. Gives operators control
  over allowed classes.
- Allow us to evolve the API to express concepts such a levels of service
  associated with a given Ingress controller.

Add a field to `ingress.spec`:

```golang
type IngressSpec struct {
  ...
  // Class is the name of the IngressClass cluster resource. This defines
  // which controller(s) will implement the resource.
  Class string
  ...
}

...

// IngressClass represents the class of the Ingress, referenced by the
// ingress.spec. IngressClass will be a non-namespaced Cluster resource.
type IngressClass struct {
  metav1.TypeMeta
  metav1.ObjectMeta

  // Controller is responsible for handling this class. This should be
  // specified as a domain-prefixed path, e.g. "acme.io/ingress-controller".
  //
  // This allows for different "flavors" that are controlled by the same
  // controller. For example, you may have different Parameters for
  // the same implementing controller.
  Controller string

  // Parameters is a link to a custom resource configuration for
  // the controller. This is optional if the controller does not
  // require extra parameters.
  //
  // +optional
  Parameters *TypeLocalObjectReference
}
```

##### Interoperability with previous annotation

The Ingress class set in the annotation takes priority to the
`Spec.Class` field. The controller MAY emit a warning event if the
user sets conflicting (different) values for the annotation and
`Spec.Class`.

### Alternative backend types

The Ingress resource is an L7 description of a composite set of
services. It currently supports only Kubernetes Services as a
backends. However, there are many use cases where a portion of the
HTTP requests could be routed to a different kind of resource. For
example, serving content from an object storage ([S3][s3-backend],
[GCS][gcs-backend]) is a commonly requested feature.

At the same time, we do not expect to enumerate all possible backends
that could arise, nor do we expect that naming of the resources will
be uniform in schema, parameters etc. Similarly, many of the resources
will be implementation-specific.

#### Backend types proposal

Add a field to the `IngressBackend` struct with an object reference:

```golang
type IngressBackend struct {
  // Only one of the following fields may be specified.

  // Service references a Service as a Backend. This is specially
  // called out as it is required to be supported AND to reduce
  // verbosity.
  // +optional
  Service *ServiceBackend

  // Resource is an ObjectRef to another Kubernetes resource in the namespace
  // of the Ingress object.
  // +optional
  Resource *v1.TypedLocalObjectReference
}

// ServiceBackend references a Kubernetes Service as a Backend.
type ServiceBackend struct {
  // Service is the name of the referenced service. The service must exist in
  // the same namespace as the Ingress object.
  // +optional
  Name string

  // Port of the referenced service. If unspecified and the ServiceName is
  // non-empty, the Service must expose a single port.
  // +optional
  Port ServiceBackendPort
}

// ServiceBackendPort is the service port being referenced.
type ServiceBackendPort struct {
  // Number is the numerical port number (e.g. 80) on the Service.
  Number int
  // Name is the name of the port on the Service.
  Name string
}
```

Support for non-`Service` type `Resource`s is
implementation-specific. Implmentations MUST support Kubernetes
Service. Support for other types is OPTIONAL.

##### Backend types examples

Ingress routing everything to `foo-app`:

```yaml
kind: Ingress
spec:
  class: acme-lb
  backend:
    service:
	  name: foo-app
	  port:
	    number: 80
```

Ingress routing everything to the ACME storage bucket:

```yaml
kind: Ingress
spec:
  class: acme-lb
  backend:
    resource:
	  apiGroup: acme.io/networking
	  kind: storage-bucket
	  name: foo-bucket
```

Invalid configuration (uses both resource and service):

```yaml
kind: Ingress
spec:
  class: acme-lb
  backend:
    service:
	  name: foo-app
	  port:
	    number: 80
    resource: # INVALID!
	  apiGroup: acme.io/networking
	  kind: storage-bucket
	  name: foo-bucket
```

##### Supporting custom backends (non-normative)

As a sketch, an object bucket can be named with a CRD. NOTE: this
example is non-normative and for illustration purposes only.

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

- [x] Copy the Ingress API to `networking.k8s.io/v1beta1` (preserving
  existing data and round-tripping with the extensions Ingress API,
  following the approach taken for all other `extensions/v1beta1`
  resources).
- [x] Develop a set of planned changes and GA graduation criteria with
  sig-network (intent is to target a minimal set of bugfixes and
  non-breaking changes)
- [x] Announce `extensions/v1beta1` Ingress as deprecated (and
      announce plan for GA)

#### Test plan

- Copy existing Ingress tests, changing the resource type to the new
  group. Keep existing tests as is.

### 1.15

- [x] Update API server to persist in networking.k8s.io/v1beta1 kubernetes/kubernetes#77139
- [x] Update in-tree controllers, examples, and clients to target kubernetes/kubernetes#77617
  `networking.k8s.io/v1beta1`
- [x] Update Ingress controllers in the kubernetes org to target
  `networking.k8s.io/v1beta1`
  - [x] [ingress-nginx](https://github.com/kubernetes/ingress-nginx/pull/4127)
  - [x] [ingress-gce](https://github.com/kubernetes/ingress-gce/issues/770)
- [x] Update documentation to recommend new users start with kubernetes/website#14239
  networking.k8s.io/v1beta1, but existing users stick with
  `extensions/v1beta1` until `networking.k8s.io/v1` is available.
- [x] Update documentation to reference `networking.k8s.io/v1beta1` kubernetes/website#14239

### 1.16

- [ ] Meet graduation criteria and promote API to `networking.k8s.io/v1`
- [ ] Implement API changes to GA version.
- [ ] Announce `networking.k8s.io/v1beta1` Ingress as deprecated

### 1.17

- [ ] Update API server to persist in `networking.k8s.io/v1`.
- [ ] Update in-tree controllers, examples, and clients to target
  `networking.k8s.io/v1`.
- [ ] Update Ingress controllers in the kubernetes org to target
  `networking.k8s.io/v1`.
- [ ] Update documentation to reference `networking.k8s.io/v1`.
- [ ] Evangelize availability of v1 Ingress API to out-of-org Ingress
      controllers

### 1.18

- [ ] Remove ability to serve `extensions/v1beta1` and
  `networking.k8s.io/v1beta1` Ingress resources (preserve ability to
  read existing `extensions/v1beta1` Ingress objects from storage and
  serve them via the `networking.k8s.io/v1` API)

## Graduation Criteria

### API group move to `networking.k8s.io/v1beta1`

- [x] 1.14: Ingress API exists and has parity with existing
  `extensions/v1beta1` API
- [x] 1.14: `extensions/v1beta1` Ingress tests are replicated against
  `networking.k8s.io`
- [x] 1.15: all in-tree use and in-org controllers switch to
  `networking.k8s.io` API group
- [ ] 1.15: documentation and examples are updated to refer to
  networking.k8s.io API group `networking.k8s.io/v1`

### GA

- [ ] 1.17: API finalized and implemented on the branch.
- [ ] 1.XX: Ingress spec and conformance tests finalized and running against branch.
- [ ] 1.XX: API changes merged into the main API, with tests from v1beta1 pointing to GA.

## Implementation History

- 1.14: Copied Ingress API to the networking API group.

## Alternatives

See motivation section.

## Appendix

### Design discussions

- Kubecon EU 2019 [sig-network meetup][kubecon-eu-2019].

[kubecon-eu-2019]: https://docs.google.com/document/d/1x8KoNWLKA9JEDD-z88A8Kb1yzHJ8YDhDCoBZsQxMM6Q/edit#bookmark=id.60xvqkshg3z4

### Non-options

One suggestion was to move the API into a new API group, defined as a
CRD.  This does not work because there is no way to do round-trip of
existing Ingress objects to a CRD-based API.

### Future design: Healthchecks

The current spec does not have any provisions to customize
healthchecks for referenced backends. Many users already have a
healthcheck URL that is lightweight and different from the HTTP root
(i.e. `/`).

One obvious question that arises is why the Ingress healthcheck
configuration is (a) is needed and (b) is different from the current
Pod readiness and liveness checks. The Ingress healthcheck represents
an end-to-end check from the proxy server to the backend.  The
Kubelet-based service health check operates only within the VM and
does not include the network path. A minor point is that it is also
the case that some providers require a healthcheck to be specified as
part of load balancing.

An option that has been explored is to infer the healthcheck URL from
the Readiness/Liveness probes on the Pods of the Service. This method
has proven to be unworkable: Every Pod in a Service can have a
different Readiness probe definition and therefore it's not clear
which one should be used. Furthermore, the behavior is implicit and
creates action-at-a-distance relationship between the Ingress and Pod
resources.

#### Healthchecks proposal

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

### Potential pre-GA work

Note: these items are NOT the main focus of this KEP, but recorded
here for reference purposes. These items came up in discussions on the
KEP (roughly sorted by practicality):

- Spec path as a prefix, maybe as a new field
- Rename `backend` to `defaultBackend` or something more obvious
- Be more explicit about wildcard hostname support (I can create *.bar.com but
  in theory this is not supported)
- Add health-checks API
- Specify whether to accept just HTTPS or also allow bare HTTP
- Better status
- Formalize Ingress class
- Reference a secret in a different namespace?  Use case: avoid copying wildcard
  certificates (generated with cert-manager for instance)
- Add non-required features (levels of support)
- Some way to have backends be things other than a service (e.g. a GCS bucket)
- Some way to restrict hostnames and/or URLs per namespace
- HTTP to HTTPS redirects
- Explicit sharing or non-sharing of external IPs (e.g. GCP HTTP LB)
- Affinity
- Per-backend timeouts
- Backend protocol
- Cross-namespace backends

### Rejected designs

This section contains rejected design proposals for future reference.

#### Portable regex for Path

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

Maintaining a regular expression subset is not worth the complexity and
is likely impossible across the [many implementations][regex-survey].

<!-- References -->

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
