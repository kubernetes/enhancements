# KEP-5707: Deprecate Service.spec.externalIPs

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Migration Guidance](#migration-guidance)
    - [Recommended Alternatives](#recommended-alternatives)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Deprecation (v1.36)](#deprecation-v136)
    - [Feature disabled - (Feature Gate Default to false)](#feature-disabled---feature-gate-default-to-false)
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
  - [Keep externalIPs with Enhanced Security](#keep-externalips-with-enhanced-security)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes to deprecate the `Service.spec.externalIPs` field, along with deprecating and finally
removing kube-proxy's implementation of this feature. The externalIPs feature has long been considered a security risk and
architectural problem, as it allows non-privileged users to claim arbitrary IP addresses without proper authorization or
validation, potentially enabling man-in-the-middle attacks and other security exploits.[^1]

[^1]: [CVE-2020-8554](https://nvd.nist.gov/vuln/detail/cve-2020-8554)

This deprecation will proceed through three phases:

1. Mark the API field as deprecated and emit warnings when the field is used in a create or update of a Service
2. Introduce a feature gate named AllowServiceExternalIPs (default: false) that disables kube-proxy support
3. Remove all associated code in kube-proxy and remove the "DenyServiceExternalIPs" admission controller

## Motivation

The `Service.spec.externalIPs[]` field was introduced early in Kubernetes history as a way to expose services on specific external IP addresses. However, this feature has several fundamental problems:

1. **Security Risk**: Any user with permission to create or modify Services can claim arbitrary IP addresses, including IPs that may be in use by other systems or services. This enables potential man-in-the-middle attacks and unauthorized traffic interception.

2. **Lack of Validation**: There is no built-in validation to ensure that the specified IP addresses are actually available, routable, or authorized for use by the cluster.

3. **Operational Complexity**: The feature requires manual configuration and coordination outside of Kubernetes to ensure that traffic to the specified IPs is properly routed to the cluster nodes.

4. **Better Alternatives Exist**: Modern Kubernetes provides better alternatives for exposing services, including LoadBalancer services (with cloud provider integration) and Ingress/Gateway resources.

The community has already recognized these issues, as evidenced by [KEP-2200](https://github.com/kubernetes/enhancements/tree/08fe93397de28e3cfa1c2cb5b2a8488d8b3b1121/keps/sig-network/2200-externalips-admission) (Block service ExternalIPs via admission),
which introduced admission control to deny the use of externalIPs. This KEP takes the next step by formally deprecating the field and removing the implementation.

### Goals

- Deprecate (via a warning) the `Service.spec.externalIPs` field in the Kubernetes API
- Removal of kube-proxy's support for `Service.spec.externalIPs`
- Removal of the `DenyServiceExternalIPs` admission controller
- Updating docs to reflect the deprecation/removal.

### Non-Goals

- Providing a direct replacement feature within the Service API (users should migrate to LoadBalancer services, Ingress, or Gateway API)

## Proposal

This KEP proposes a phased deprecation and removal of the `Service.spec.externalIPs` field and its implementation:

1. **Phase 1 - Deprecation Warnings**

Tentative version: 1.36

- Create a blog post describing the timeline of the removal of the feature
- Mark the field as deprecated in API documentation and emit warnings when used. Document security issues and migration alternatives.
- Log an error when the DenyServiceExternalIPs admission controller is enabled, explaining of its upcoming deprecation and removal.
- Introduce the `AllowServiceExternalIPs` feature gate to kube-proxy (default: true). When set to false, this will:
  - Cause kube-proxy to stop programming iptables/nftables/ipvs rules for externalIPs

2. **Phase 2 - Disable kube-proxy support**

4 releases after Phase 1 (tentatively in Kubernetes 1.40):

- Switch the `AllowServiceExternalIPs` feature gate to false
- Create e2e test ensuring that externalIPs is no longer used by kube-proxy

Clusters administrators can enable the feature gate if needed to restore the externalIPs functionality

3. **Phase 3 - Lock feature gate and remove kube-proxy functionality**

3 releases after Phase 2 (tentatively in Kubernetes 1.43):

- Lock `AllowServiceExternalIPs` feature gate to false
- Remove the feature gate checks and all related code from kube-proxy
- Remove any tests (unit, integration or e2e) of the feature
- Update docs to reflect that the feature no longer exists
- Promote the e2e to a conformance test

- Do not remove any code from kube-apiserver or its unit tests
- For simplicity, do not remove the DenyServiceExternalIPs admission controller.

4. **Phase 4 - Code and feature gate removal**

After 3 releases since phase 3, remove the `AllowServiceExternalIPs` feature gate and `DenyServiceExternalIPs` admission controller

### Risks and Mitigations

User who use this feature may not be aware of the upcoming deprecation and feature removal, and they may not have alternative options.
This can be mitigrated by:

1. Updating feature documentation indicating that the feature is insecure and will be going away, and provide users with alternatives
1. Create a blog post describing the timeline of the removal of the feature

## Design Details

Design details are covered in the proposal section

### Migration Guidance

Users must migrate to supported alternatives:

#### Recommended Alternatives

1. **LoadBalancer Services with Cloud Provider Integration**
   - Use `type: LoadBalancer` with cloud provider's load balancer
   - Provides proper IP allocation and authorization
   - Supported by all major cloud providers

2. **Gateway API**
   - Modern, extensible API for cluster ingress
   - Supports advanced routing and multiple implementations
   - Recommended for new deployments

3. **Ingress Controllers**
   - Traditional approach for HTTP/HTTPS traffic
   - Wide ecosystem of implementations
   - May be combined with LoadBalancer services

### Test Plan

[x] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

##### Prerequisite testing updates

No new functionality is being added, so existing test coverage for the Service API should be sufficient. Tests that create Services with externalIPs should be updated to expect deprecation warnings.

##### Unit tests

- Update existing kube-proxy unit tests to handle disabling and enabling of the `AllowServiceExternalIPs` feature gate

- `pkg/proxy/iptables`: `2026-02-03` - `80.5%`
- `pkg/proxy/nftables`: `2026-02-03` - `77.9%`
- `pkg/proxy/ipvs`: `2026-02-03` - `61.9%`

##### Integration tests

No integration tests to be added or updated

##### e2e tests

Existing e2e tests that use externalIPs should be updated to handle when gate is enabled and disabled
Create e2e test ensuring that externalIPs is no longer used by kube-proxy, promote this e2e test to conformance test once the feature gate is locked.

### Graduation Criteria

#### Deprecation (v1.36)

- API documentation updated to mark externalIPs as deprecated
- Deprecation warnings emitted when externalIPs is used
- Migration guide with alternatives published

#### Feature disabled - (Feature Gate Default to false)

- No major issues with deprecation warnings
- Feature gate `AllowServiceExternalIPs` defaults to `false`
- Documentation for opting out if needed

### Upgrade / Downgrade Strategy

Users who require additional time to move off of ExternalIPs may need to enable the AllowServiceExternalIPs feature gate for the feature to continue to work (for a limited time).

### Version Skew Strategy

No concerns with version skew, the warning from api-server and feature not working in kube-proxy are independent.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: AllowServiceExternalIPs
  - Components depending on the feature gate: kube-proxy

###### Does enabling the feature change any default behavior?

Enabling of this feature will cause ExternalIPs to stop working.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Rollback is allowed by setting `AllowServiceExternalIPs` to true, however, this is not possible after the gate has been locked.

###### What happens if we reenable the feature if it was previously rolled back?

Services with ExternalIPs will start to be served again.

###### Are there any tests for feature enablement/disablement?

No

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

N/A

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No - the code doesn't exist at time of writing

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

Yes, this KEP is the deprecation of `Service.spec.externalIPs`

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Yes, using:

```
kubectl get svc -A -o jsonpath='{range .items[?(@.spec.externalIPs)]}{.metadata.namespace}{"\t"}{.metadata.name}{"\t"}{.spec.externalIPs}{"\n"}{end}'
```

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

N/A

### Scalability

###### Will enabling / using this feature result in any new API calls?

N/A

###### Will enabling / using this feature result in introducing new API types?

N/A

###### Will enabling / using this feature result in any new calls to the cloud provider?

N/A

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

N/A

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

N/A

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

N/A

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

### Troubleshooting


###### How does this feature react if the API server and/or etcd is unavailable?

N/A

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

- Users currently relying on externalIPs will need to migrate their workloads
- Some users in restricted environments (bare metal, air-gapped) may have limited alternatives
- Tooling and operators that programmatically create Services with externalIPs will need updates
- The deprecation warnings may create noise in logs for users who cannot immediately migrate

## Alternatives

### Keep externalIPs with Enhanced Security

Instead of deprecating the field, we could enhance it with better security controls:

- Require explicit RBAC permissions for using externalIPs
- Add VAP for IP validation and authorization

**Rejected**: This approach would require significant engineering effort to secure a feature that has better alternatives. The fundamental architectural issues with externalIPs make it a poor fit for modern Kubernetes. The security issues are inherent to the design.
