# KEP-2365: IngressClass Namespaced Params

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
    - [Unit Tests](#unit-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha release](#alpha-release)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This KEP proposes adding new Scope and Namespace fields to the IngressClass
ParametersRef field.

## Motivation

After the initial release of IngressClass, a number of use cases called for the
ability to reference namespace-scoped Parameters. For example, one could use a
GatewayClass parameters CR to describe how and where a controller should be
provisioned. This same thought process was also happening in the Service APIs
subproject. It was ultimately deemed worthwhile for GatewayClass if we could
also gain approval for a parallel API change to IngressClass.

### Goals

- Allow referencing namespace-scoped Parameters resources.

### Non-Goals

- Requiring all Parameters resources to be namespace-scoped.

## Proposal

Add new Scope and Namespace fields to the IngressClass ParametersRef field.

### Risks and Mitigations

The option to reference namespace-scoped Parameters resources could lead to
confusion. It is relatively rare for resource references to be able to target
both cluster-scoped and namespace-scoped resources. We believe that the
advantages of this KEP outweigh this potential confusion.

## Design Details

This will result in adding a new `IngressClassParametersReference` type that
closely mirrors the existing `TypedLocalObjectReference` type that is currently
in use.

```golang
// IngressClassParametersReference identifies an API object. This can be used
// to specify a cluster-scoped or namespace-scoped resource.
type IngressClassParametersReference struct {
  // APIGroup is the group for the resource being referenced. If APIGroup is not
  // specified, the specified Kind must be in the core API group. For any other
  // third-party types, APIGroup is required.
  // +optional
  APIGroup *string
  // Kind is the type of resource being referenced.
  Kind string
  // Name is the name of resource being referenced.
  Name string
  // Scope represents if this refers to a cluster or namespace scoped resource.
  // This may be set to "cluster" or "namespace".
  // Default: "cluster"
  Scope string
  // Namespace is the namespace of the resource being referenced. This field is
  // required when scope is set to "namespace".
  // +optional
  Namespace *string
}
```

Use of these new `Scope` and `Namespace` fields will be guarded by a new
`IngressClassNamespacedParams` feature gate.

### Test Plan

#### Unit Tests
- When feature gate is disabled:
  - Ensure that namespace and scope fields can not be set on a newly created
    IngressClass resource.
  - Ensure that namespace and scope field can not be changed if it is not
    already set on an IngressClass resource.
  - Ensure that namespace and scope field can be changed if it is already set on
    an IngressClass resource.
- When feature gate is enabled:
  - Ensure that namespace and scope field can be set on a newly created
    IngressClass resource.
  - Ensure that namespace and scope field can be changed if it is not already
    set on an IngressClass resource.
  - Ensure that namespace and scope field can be changed if it is already set on
    an IngressClass resource.

### Graduation Criteria

#### Alpha release

- Implementation complete
- Test plan complete
- Documentation added covering how params resources should and should not be
  used

#### Alpha -> Beta Graduation

- Existed in alpha for at least 1 minor release

#### Beta -> GA Graduation

- Existed in beta for at least 1 minor release

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

See unit tests above.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: IngressClassNamespacedParams
    - Components depending on the feature gate: API Server

* **Does enabling the feature change any default behavior?**
  A new API field can be set. This may enable new behavior for Ingress
  controllers that support the field.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes.

* **What happens if we reenable the feature if it was previously rolled back?**
  The fields becomes accessible again.

* **Are there any tests for feature enablement/disablement?**
  Yes.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**
  N/A

* **What specific metrics should inform a rollback?**
  N/A

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  N/A

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**
  No.

### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?**
  N/A

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**
  N/A

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  N/A

* **Are there any missing metrics that would be useful to have to improve
  observability of this feature?**
  No.

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
  No

### Scalability

* **Will enabling / using this feature result in any new API calls?**
  No

* **Will enabling / using this feature result in introducing new API types?**
  Yes, IngressClassParametersReference.

* **Will enabling / using this feature result in any new calls to the cloud
  provider?**
  No

* **Will enabling / using this feature result in increasing size or count of the
  existing API objects?**
  Will very slightly increase the size of the IngressClass resource. Generally
  less than 10 of these resources should exist in a cluster.

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs]?**
  No

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  No

### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?**
  N/A

* **What are other known failure modes?**
  N/A

* **What steps should be taken if SLOs are not being met to determine the problem?**
  N/A

## Implementation History

- January 28, 2021: KEP written

## Drawbacks

Potential for confusion with a params reference that can point to both namespace
scoped and cluster scoped resources.

## Alternatives

- Each controller could assume all parameters were in a predefined namespace.
  This would likely lead to more confusion since it would be different for each
  implementation.

- We could not support namespace-scoped parameters references. This would be
  simplest but would rule out some compelling use cases.