# KEP-2558: Publish versioning information in OpenAPI

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Example](#example)
  - [Constraints and Caveats](#constraints-and-caveats)
    - [Constraints](#constraints)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
- [x] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes a way for publishing field-level versioning information
in OpenAPI using comment tags on API fields for built-in types.

Specifically, the field-level versioning information would include the
prerelease status, minimum Kubernetes version for the prerelease status
and the feature gate associated with field.

## Motivation

**Feature gate**

When a new field is added to an existing API version, a feature gate
is added to control the enablement of the new field. Sometimes feature
gates are also used when a new `Kind` is added.

The feature gates are defined in [`kube_features.go`]. The comments
for each of these feature gates in [`kube_features.go`] lists
the prerelease status of the feature gate and the minimum Kubernetes
version for the particular prerelease. These comments do not follow
a pattern and are not parseable.

Currently, when a new field is added, there is no way to map and document
the field with the feature gate that controls it. Some fields mention
the feature gates in a comment but these comments are not parseable either.

**Field version**

Currently, when a new field is added to a stable API version, there
is no way to expose the version of the new field in OpenAPI.
One way to identify alpha API fields is by comparing the OpenAPI schema
generated from an apiserver with all APIs enabled and the one generated
from an apiserver with only stable and beta APIs enabled.
However, this process is cumbersome and not always feasible because not all
fields are gated at the API group level.

Since the OpenAPI definition is the source of truth for API documentation,
this KEP proposes a way to publish versioning and feature gate
information via OpenAPI.

As a follow-up, this information can also be used to:
- update the Kubernetes API reference documentation
- surface [server-side warnings] in client-go and kubectl when a deprecated
field or alpha field is being used.
- surface warnings next to the field name in IDE when a deprecated
field or alpha field is being used.

### Goals

- Publish feature gate associated with an API field for built-in API types
in OpenAPI to document the mapping of API fields with their feature gates.
- Document the current prerelease status of an API field by publishing
the prerelease status of the field and the minimum Kubernetes version for
the prerelease in OpenAPI.
- Validate that the feature gate mentioned in the API field comment
is a valid feature gate.

### Non-Goals

- Support publishing versioning information for built-in `Kind`s.
This may be added in the future but is not in scope in the alpha status of this KEP.
- Support publishing field-level versioning for custom resources in OpenAPI.
- Publish the default status of a feature gate (enabled/disabled by default)
in OpenAPI.
- Allow parsing comment tags on feature gates in [`kube_features.go`].
- Update the Kubernetes API reference docs to include the field-level
versioning information (out of scope of this KEP, can be done in a follow up).

## Proposal

This KEP proposes changes to the kube-openapi generator to add a new
vendor extension field `x-kubernetes-api-lifecycle` in OpenAPI.
This extension specifies the prerelease status of an API field,
the minimum Kubernetes version for the prerelease and the
feature gate associated with the field.

This information is derived from comment tag on API fields with the format:

```
// +lifecycle:kubernetes:minVersion=<k8s-version>,status=<prelease-status>,featureGate=<featuregate-name>
```

The following Kubernetes-specific constraints are validated by a linter:

- The `+lifecycle:kubernetes:minVersion=<k8s-version>` tag denotes that the field is
in the prerelease since the specified Kubernetes version. It must map to a
Kubernetes release (e.g. `v1.20`).
- The `status` value must be one of `alpha`, `beta`, or `deprecated`.
- The `featureGate` value must be a valid feature gate as defined in [`kube_features.go`].

When an API field moves to GA, the feature gate and the comment tag for the API field
are removed. This will also remove the `x-kubernetes-api-lifecycle` vendor extension for
the field in OpenAPI.

This method also provides us the extensibility to support CRDs in the future.
For CRDs, multiple `// +lifecycle` comment lines may be added for custom types
to denote the project's specific version.

For example, an additional line `// +lifecycle:istio:minVersion=v3.0.0` may be added.
This allows capturing both the Kubernetes and the project's (istio) versions accurately.
The specific syntax for CRDs will be defined later.

Note: This KEP does not propose changing what API fields are published in the OpenAPI
schema. It only adds extensions to fields for which the comment tag is specified.

### Example

Consider adding a new alpha API field `Width` to the stable API `Frobber`
in `v1.20`.

```go
type Frobber struct {
  Height *int32 `json:"height"
  Param  string `json:"param"
  // width indicates how wide the object is.
  // This field is alpha-level and is only honored by servers that enable the Frobber2D feature.
  // +optional
  Width  *int32 `json:"width,omitempty"
}
```

The field `Width` is controlled by the `Frobber2D` feature gate defined
in [`kube_features.go`].

```go
// owner: @you
// alpha: v1.20
//
// Add multiple dimensions to frobbers.
Frobber2D utilfeature.Feature = "Frobber2D"

var defaultKubernetesFeatureGates = map[utilfeature.Feature]utilfeature.FeatureSpec{
  ...
  Frobber2D: {Default: false, PreRelease: utilfeature.Alpha},
}
```

To link the feature gate with the API field, the following comment line
is added for the `Width` field:

```go
type Frobber struct {
  Height *int32 `json:"height"
  Param  string `json:"param"
  // width indicates how wide the object is.
  // +optional
  // +lifecycle:kubernetes:minVersion=v1.20,status=alpha,featureGate=Frobber2D
  Width  *int32 `json:"width,omitempty"
}
```

This will generate an OpenAPI schema with an additional vendor extension field:

```go
"width": {
  VendorExtensible: spec.VendorExtensible{
    Extensions: spec.Extensions{
      "x-kubernetes-api-lifecycle": map[string]interface{}{
        "kubernetes":  map[string]interface{
          "minVersion": "v1.20",
          "status": "alpha",
          "featureGate": "Frobber2D",
        },
      },
    },
  },
```

This shows that the field is alpha from Kubernetes version v1.20 and 
is controlled by the `Frobber2D` feature gate.

### Constraints and Caveats

#### Constraints

1. The source of truth for versioning information for a feature gate
will be the comment tags on the respective API fields because
these comment tags are used for publishing to OpenAPI.
A linter ensures that comments for feature gates defined in
[`kube_features.go`] do not diverge from the comment lines on API fields.

2. Additionally, the following constraints are applied and validated for
comment tags on API fields:

- The `+lifecycle:kubernetes:minVersion` value must satisfy the regular expression `^v[1-9][0-9]*\.(0|[1-9][0-9]*)$`.
- The `status` value must be one of `alpha`, `beta`, or `deprecated`.
- The `featureGate` value is a valid feature gate as defined in
[`kube_features.go`].

### Test Plan

The constraints for the comment tags listed above are validated using
a linter and relevant tests for introducing a new OpenAPI extension
are added to kube-openapi.

### Graduation Criteria

This feature is currently in the alpha status.

For the feature to move to the beta status, feedback will be collected
on usage of the `x-kubernetes-api-lifecycle` extension in OpenAPI in
various places like documentation, IDE support and [server-side warnings]
in client-go and kubectl.

Depending on the feedback, support for publishing versioning information
may be added to new `Kind`s as well.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

N/A

###### Does enabling the feature change any default behavior?

N/A

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

N/A

###### What happens if we reenable the feature if it was previously rolled back?

N/A

###### Are there any tests for feature enablement/disablement?

N/A

### Rollout, Upgrade and Rollback Planning

###### How can a rollout fail? Can it impact already running workloads?

N/A

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

N/A

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

### Troubleshooting

N/A

###### How does this feature react if the API server and/or etcd is unavailable?

N/A

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

- 2021-02-05: Initial [discussion on the SIG Architecture mailing list]
- 2021-03-07: KEP proposed

## Alternatives

1. The [prerelease-lifecycle-gen] generator allows specifying metadata about
the prerelease status for packages (e.g. `cronjobs.v1beta1.batch`).

Field-level versioning information support is added in the kube-openapi
generator and not added to the [prerelease-lifecycle-gen] generator
because:

- [prerelease-lifecycle-gen] only generates and documents metadata
about when each _package_ was introduced, deprecated or removed.
It is not meant to work for individual API types or fields.
- Additionally, even if field-level support was added to it,
this information would not be surfaced in OpenAPI which makes it
harder to be consumed by other tools (e.g. for conformance and documentation).

2. An alternative was considered to define the prerelease status, minVersion and
featureGate value on a single comment line:

```
// +k8s:openapi-gen:prerelease=alpha,minVersion=v1.20,featureGate=Frobber2D
```

This approach was not used because `+k8s:openapi-gen` would limit the field metadata
to only be used by the openapi generator. Additionally, splitting into multiple comment
lines keeps the syntax simpler and makes it easier to debug.

3. The initial draft of this KEP allowed specifying multiple prerelease statuses
in comment tags for an API field. For example:

```
// +k8s:openapi-gen:prerelease=alpha,minVersion=v1.20,featureGate=Frobber2D
// +k8s:openapi-gen:prerelease=beta,minVersion=v1.21,featureGate=Frobber2D
```

However, this introduces risk of vastly increasing the size of the OpenAPI schema.
CRDs with multiple embedded `PodTemplateSpec`s can easily blow past the 1MB etcd limit with
this approach.

4. An alternative was considered to add comment tags to all types that are not GA.
Existing and new GA types would be listed as [exceptions].
This ensures that `types.go` files are not spammed with comment tags an
tooling can interpret absence of a comment tag as an error instead of
assuming the types to be GA.

Depending on the feedback of the initial proposal for fields, this option
may be considered in the future.

[`kube_features.go`]: https://github.com/kubernetes/kubernetes/blob/master/pkg/features/kube_features.go
[server-side warnings]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/1693-warnings
[exceptions]: https://github.com/kubernetes/kubernetes/tree/master/api/api-rules
[`hack/verify-description.sh`]: https://github.com/kubernetes/kubernetes/blob/master/hack/verify-description.sh
[prerelease-lifecycle-gen]: https://github.com/kubernetes/kubernetes/tree/master/staging/src/k8s.io/code-generator/cmd/prerelease-lifecycle-gen
[discussion on the SIG Architecture mailing list]: https://groups.google.com/g/kubernetes-sig-architecture/c/UmPwm-J3ztE
[#99307]: https://github.com/kubernetes/kubernetes/pull/99307