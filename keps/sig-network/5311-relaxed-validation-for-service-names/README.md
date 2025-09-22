<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [X] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [X] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [X] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [X] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [X] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-5311: Relaxed validation for Services names

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
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
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

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [X] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This document proposes a relaxation of the Service name validation, in order bring it in line
with the validation requirements of names of other resources in Kubernetes.

## Motivation

At time of writing, Service name validation is stricter than most of the other Kubernetes resource names.
By losening the validation of Services, it simplifies the Kubernetes code base slightly, by removing the
`apimachineryvalidation.NameIsDNS1035Label` validation, which is only used by Service names.

Additionally, this change also allows users to name their Services with the same conventions
as other of their resources, ie: Service names can now start with a digit.

### Goals

- Allow Service names to be created using `apimachineryvalidation.NameIsDNSLabel` validation

### Non-Goals

- Change validation for other Kubernetes resource types
- Removal of the `apimachineryvalidation.NameIsDNS1035Label` function

## Proposal

At time of writing Service names are validated with `apimachineryvalidation.NameIsDNS1035Label`.
The proposal is to change this validation to `apimachineryvalidation.NameIsDNSLabel`, allowing Service names to start with a digit.

### Risks and Mitigations

1. Services are responsible for creating DNS records (ie: `<service-name>.<namespace>.svc.cluster.local>`). To confirm that downstream systems will support the new validation, we will conduct compatibility testing by:
   - Verifying that DNS providers used in Kubernetes clusters can handle the new service name format.
   - Running integration tests to ensure that dependent components such as Ingress controllers and service discovery mechanisms function correctly with the updated validation.
2. The Ingress resource references Service and will also need a relaxed validation on its reference to the Service
3. Downstream applications may perform validation on fields relating to Service names (ie: [ingress-nginx](https://github.com/kubernetes/ingress-nginx/blob/d3ab5efd54f38f2b7c961024553b0ad060e2e916/internal/ingress/annotations/parser/validators.go#L190-L197) and will also need updating.

## Design Details

Introduce a new feature gate named RelaxedServiceNameValidation, which is disabled by default in alpha.

When enabled, the feature gate will use the [NameIsDNSLabel](https://github.com/kubernetes/kubernetes/blob/3196c9946355c1d20086f66c22e9e5364fb0a56f/staging/src/k8s.io/apimachinery/pkg/api/validation/generic.go#L44-L50) validation for new Services.

Since the relaxed check allows previously invalid values, care must be taken to support cluster downgrades safely. To accomplish this, the validation will distinguish between new resources and updates to existing resources:

When the feature gate is disabled:

- Creation of Services will use the previous `NameIsDNS1035Label()` validation for `.metadata.name`
- Updates of Services will no longer validate `metadata.name`, since the field is immutable, so the existing value can be assumed to be valid.
- Creation of Ingress will use the previous `NameIsDNS1035Label()` validation for `.spec.rules[].http.paths[].backend.service.name`
- Updates of Ingress will use the previous `NameIsDNS1035Label()` validation for `.spec.rules[].http.paths[].backend.service.name` if that field changes, otherwise, there is no validation

When the feature gate is enabled:

- Creation of Services will use new `NameIsDNSLabel()` validation for `.metadata.name`
- Creation and update of Ingress will use the new `NameIsDNSLabel()` validation for `.spec.rules[].http.paths[].backend.service.name`

### Test Plan


[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates


##### Unit tests

Tests which validate Service creation/update and Ingress creation/update to be updated.

- `pkg/apis/core/validation`: `2025-05-24` - `84.7%`
- `pkg/apis/networking/validation`: `2025-05-24` - `91.9%`

##### Integration tests

**Alpha:**

1. With the feature gate enabled, test that Services can be created with both new and previous validation
1. With the feature gate disabled, test that Services can be created with the previous validation, and fail when using the new validation
1. Disable the feature gate and ensure that the Service can be edited without a validation error being returned

**Beta:**

Tests have been written: https://github.com/kubernetes/kubernetes/blob/v1.34.0/test/integration/service/service_test.go#L1219-L1309

##### e2e tests

**Alpha:**

- Create a Service that requires the new validation and test if a DNS lookup works for it

**Beta:**

An e2e exists: https://github.com/kubernetes/kubernetes/blob/v1.34.0/test/e2e/network/dns.go#L659-L686

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- E2E and Integration tests completed and enabled.

#### GA

- Time passes, no major objections
- Promote e2e test to conformance

### Upgrade / Downgrade Strategy

### Version Skew Strategy

Not applicable - only a single component is being changed.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: RelaxedServiceNameValidation
  - Components depending on the feature gate:
    - kube-apiserver

###### Does enabling the feature change any default behavior?

No, as this feature is for validation only.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, via the feature gate.

###### What happens if we reenable the feature if it was previously rolled back?

The new relaxed validation will be enabled again.

###### Are there any tests for feature enablement/disablement?

The integration test will test disabling of the feature

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

An initial rollout cannot fail because the feature only changes the
behavior when creating new Services and doesn't affect already-created
Services.

A rollback will always succeed if the user hasn't created any Services that depend
on the new validation.

A rollback to a version that has the feature gate turned on will work, even if there are
Services that depend on the new validation, since any version of Kubernetes with
this feature gate on will allow existing Services to have `NameIsDNSLabel` names. If a
user rolls back to a version without the feature gate on, or even earlier not having this functionality, and has Services that depend
on the new validation, they will be unable to modify those services, and will
probably need to delete them and recreate them with new names.

###### What specific metrics should inform a rollback?

The following metrics could indicate that this feature is failing:

1. `apiserver_request_total{code=500, version=v1, resource=service, verb=POST}`
1. `apiserver_request_total{code=500, version=v1, resource=service, verb=PATCH}`

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Yes, using the following steps:

1. Installed a 1.34 Kubernetes cluster with the `RelaxedServiceNameValidation` feature gate disabled
1. Attempted to create a service with a name starting in a digit - it failed as expected
1. Upgraded to a custom built 1.35 Kubernetes cluster with the `RelaxedServiceNameValidation` feature gate enabled
1. Attempted to create a service with a name starting in a digit - it succeeded as expected
1. Downgraded back to 1.34 with the `RelaxedServiceNameValidation` feature gate disabled
1. Edited that same service, and it succeeded as expected

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Existence of any service which has a name that begins with a digit.

###### How can someone using this feature know that it is working for their instance?

If they are able to apply a service resource with a name that begings with a digit.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. This is a change to API validation.

### Scalability

N/A

###### Will enabling / using this feature result in any new API calls?

No. This is a change to validation of existing API calls.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No, the new validation doesn't change the maximum length of the field.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A. This is a change to validation within the API server.

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

- [x] Alpha
  - [x] KEP (`k/enhancements`) update PR(s):
    - https://github.com/kubernetes/enhancements/pull/5315
  - [x] Code (`k/k`) update PR(s):
    - https://github.com/kubernetes/kubernetes/pull/132339
  - [ ] Docs (`k/website`) update PR(s):
- [x] Beta
  - [ ] KEP (`k/enhancements`) update PR(s):
    - ...
  - [ ] Code (`k/k`) update PR(s):
    - ...

## Drawbacks

3rd party tooling being incompatible with the new validation could introduce issues for users

## Alternatives

N/A
