# KEP-2829: Migrate Gateway API to k8s.io Group

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

*Note: This KEP is primarily for tracking purposes. The code for this feature is
all out-of-tree, but the using the k8s.io API group requires API review. Most
sections of this KEP will be marked as not applicable.*

This KEP aims to transition [Gateway API](https://gateway-api.sigs.k8s.io) from
an experimental API (x-k8s.io) to an official one (k8s.io) as part of the
v1alpha2 release. The APIs will continue to be represented as CRDs, but will be
subject to the standard [Kubernetes API review
process](https://github.com/kubernetes/community/blob/master/sig-architecture/api-review-process.md).

## Motivation

Although Gateway API started as an experimental API, we are preparing a v1alpha2
release that we believe will be a significant step towards stabilizing the API.
The number of [implementations of the
API](https://gateway-api.sigs.k8s.io/implementations/) continues to
grow, and we believe that transitioning to an official Kubernetes API will help
provide greater quality and stability going forward.

### Goals

* Formalize transition of Gateway API from x-k8s.io API group to k8s.io.
* Organize full Kubernetes API reviews as part of the transition.

## Proposal

This is a large API, it would not be practical to include it within this KEP.
Instead, the API review process will happen through a separate PR or set of PRs
on the [Gateway API repo](https://github.com/kubernetes-sigs/gateway-api).
Subproject maintainers will review PRs and API-changes on a per patch basis,
while upstream k8s api-reviewers will review API changes on a release by release
basis. In addition to this, subproject maintainers could involve k8s
api-reviewers on a case-by-case as needed.

### Risks and Mitigations

N/A

## Design Details

N/A

### Graduation Criteria

#### Alpha

- Approval from subproject owners + KEP reviewers

#### Beta

- v1alpha2 APIs are implemented by several implementations
- Approval from subproject owners + KEP reviewers
- Initial conformance tests are in place
- Validating webhook for advanced validation
- We know users of the API are deploying apps with this API and exercising most
  of the API surface

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

N/A

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

N/A

###### What happens if we reenable the feature if it was previously rolled back?

N/A

###### Are there any tests for feature enablement/disablement?

N/A

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

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

###### How can someone using this feature know that it is working for their instance?

N/A

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

## Implementation History

* November 2020 - v1alpha1@0.1.0 released
* February 2021 - v1alpha1@0.2.0 released
* April 2021 - v1alpha1@0.3.0 released

## Drawbacks

Transitioning to a new API group will mean that resources in each group will be
entirely unique. If users have both new and old APIs installed, even a simple
operation like `kubectl get gateway` would only return results from one of the
installed API groups. This is unfortunate, but represents a reason to make this
change before the API becomes more stable and has a larger userbase.

## Alternatives

This API could stay as an experimental API indefinitely, but that would be very
confusing for users.