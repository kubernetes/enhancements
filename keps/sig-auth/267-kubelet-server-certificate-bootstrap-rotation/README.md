# KEP-267: Kubelet Server TLS Certificate Rotation

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This is retrospective KEP description for the feature `RotateKubeletServerCertificate`.

The feature gate `RotateKubeletServerCertificate` allows to configure kubelet to
get a server certificate for the kubelet from the Certificate Signing Request
API instead of generating one self signed and auto rotates the certificate as
expiration approaches.

## Motivation

The feature is heavily used and is a part of [CIS Kubernetes benchmark](https://www.cisecurity.org/benchmark/kubernetes):

> 1.3.6	Ensure that the `RotateKubeletServerCertificate` argument is set to true

This indicates feature maturity and high production readiness.

The outstanding work of generalizing the approach certificate requests got
approved will require a new KEP.


### Goals

- ability to configure kubelet to get a server certificate for the kubelet from
  the Certificate Signing Request API instead of generating one self signed
- auto-rotate the certificate nearing expiration using Certificate Signing
  Request API

### Non-Goals

- built-in support for approving CSRs nodes request (not currently done because
  there is not a way to determine if a node controls a given IP or DNS name)
- ability to limit the addresses/hostnames a node requests in its cert
  (currently, nodes request all addresses reported by the cloud provider, and
  the selection of which address to use is done on the API server side)

## Proposal

Documentation for the feature can be found here:

https://kubernetes.io/docs/reference/access-authn-authz/kubelet-tls-bootstrapping/#certificate-rotation

### Notes/Constraints/Caveats (Optional)

The note about default implementation that will not approve certificate requests
is already a part of documentation. See the Note on this page: https://kubernetes.io/docs/reference/access-authn-authz/kubelet-tls-bootstrapping/#certificate-rotation

### Risks and Mitigations

The feature was in heavy production use for a while. Likely no risks will emerge
with locking the feature gate.

## Design Details

Here are limited design definition for the retrospective KEP.

Implementation PRs:
- Feature gate introduced: [#45059](https://github.com/kubernetes/kubernetes/pull/45059)
- Feature promoted to GA: [#51045](https://github.com/kubernetes/kubernetes/pull/51045)

### Test Plan

Configuration testing is already done.

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None

##### Unit tests

There is a unit test coverage for the certificate manager.

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

None found.

- <test>: <link to test coverage>

##### e2e tests

None found.

- <test>: <link to test coverage>

### Graduation Criteria

#### Alpha

N/A: this is retroactive KEP, feature is already in Beta

#### Beta

N/A: this is retroactive KEP, feature is already in Beta

#### GA

- [x] Feature is actively used with the positive feedback
- [ ] Metrics promoted to GA:
  - `certificate_manager_server_rotation_seconds`
  - `certificate_manager_server_ttl_seconds`
  - `server_expiration_renew_errors`
- [ ] There are conformance tests for the certificates API endpoint, functionality
  itself do not require covering with conformance tests
- [ ] Documentation created: https://github.com/kubernetes/website/issues/30575

### Upgrade / Downgrade Strategy

Feature exists for a long time, no risk for Upgrade/Downgrade.

### Version Skew Strategy

Feature exists for a long time, no risk for version skew.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

Feature exists for a long time, no risk for Enablement and Rollback.

###### How can this feature be enabled / disabled in a live cluster?

Configuration settings must be used - command line argument
(`--rotate-server-certificates`: deprecated) or Node config flag
`serverTLSBootstrap`.

###### Does enabling the feature change any default behavior?

No, unless kubelet uses special config.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes

###### What happens if we reenable the feature if it was previously rolled back?

Kubelet will not start unless configuration will be changed to useself-signed
certificate.

###### Are there any tests for feature enablement/disablement?

N/A

### Rollout, Upgrade and Rollback Planning

N/A

###### How can a rollout or rollback fail? Can it impact already running workloads?

N/A


###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

N/A

### Monitoring Requirements

N/A

###### How can an operator determine if the feature is in use by workloads?

Yes, using metrics 

###### How can someone using this feature know that it is working for their instance?

N/A

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Metrics can be used to determine the speed of cert update.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

- Certificate Signing Request API. This API is GA.
- Configuration to approve signing requests.

This feature only works when properly configured.

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

Certificate signing requests for rotating the cert are executed on TTL
expiration. This should not introduce any scalability challenges.

###### Will enabling / using this feature result in any new API calls?

Yes, certificate signing requests.

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

Failure to get certificate can be troubleshooted using logs.

###### How does this feature react if the API server and/or etcd is unavailable?

- If it's a bootstrap, kubelet will not become functional.
- If it is a rotation, kubelet will become non-functional unless will be able to
  renew the certificate.

###### What are other known failure modes?

None

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- alpha: "v1.7"
- beta: "v1.12"


## Drawbacks

None

## Alternatives

N/A
