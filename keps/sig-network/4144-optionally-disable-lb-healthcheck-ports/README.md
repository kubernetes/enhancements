# KEP-4144: Optionally Disable Healthcheck Ports for LoadBalancer-typed Services with ExternalTrafficPolicy=Local

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

As of today, kube-apiserver allocates a HealthCheck port for every LoadBalancer-typed Services with ExternalTrafficPolicy=Local.
These HealthCheck ports are listened by kube-proxy to let frontend load-balancers know whether the node has the backend Pods running on, so the load-balancers can route traffic to the right nodes.
However, there are several scenarios that load-balancers already know where the Pods are, making the HealthCheck ports mechanism is unnecessary.
This KEP proposes to add a new field to Service to opt out of HealthCheck port allocation for these Services.

## Motivation

In the existing LoadBalancer implementations, such as [AWS ELB](https://docs.aws.amazon.com/elasticloadbalancing/latest/network/load-balancer-limits.html),
there are limitations on the number of backends that can be attached to the LoadBalancer.
As clusters grow larger, LoadBalancers may struggle to attach all nodes within a cluster.
To address this issue, some LoadBalancer implementations, like [Alibaba Cloud](https://github.com/kubernetes/cloud-provider-alibaba-cloud/blob/v2.7.0/pkg/controller/service/clbv1/vgroups.go#L429-L437)
and [Huawei Cloud](https://github.com/kubernetes-sigs/cloud-provider-huaweicloud/blob/release-1.17/pkg/cloudprovider/huaweicloud/elb.go#L566-L607),
opt to configure the target Pods (or the only nodes whereas the target Pods are running on) directly as LoadBalancer's backend, rather than attaching every node to LoadBalancer's backend.
In such cases, LoadBalancers don't rely on the HealthCheck mechanism to route traffic.
The allocation of a HealthCheck port becomes unnecessary, and should be eliminated for it wastes node ports.

### Goals

* Allow users to optionally disable HealthCheck port for LoadBalancer-typed Services with ExternalTrafficPolicy=Local
* Disabled HealthCheck ports can safely be re-enabled in a cluster

### Non-Goals

* Changing default values/behavior for LoadBalancer-typed Services with ExternalTrafficPolicy=Local

## Proposal

Add a new field `allocateLoadBalancerHCNodePort` to `Service.Spec` that allows a user to opt out of allocating HealthCheck ports for LoadBalancer-typed Services with ExternalTrafficPolicy=Local.
- if `allocateLoadBalancerHCNodePort: true`, allocate HealthCheck ports, this is the existing behavior today and will be the default value.
- if `allocateLoadBalancerHCNodePort: false`, stop allocating new HealthCheck ports but don't deallocate existing HealthCheck ports.
- if `allocateLoadBalancerHCNodePort: false`, and a user wants to disable HealthCheck ports on existing Services, they need to remove the healthCheckNodePort field explicitly.

When a user relies on `kubectl apply` to fill in the healthCheckNodePort field, the HealthCheck port would never be deallocated since the existing HealthCheck ports from the
server will always be merged prior to update. A user must send an explicit update request for the Service using something like `kubectl edit svc` or
building a controller to update all Services in a cluster.

### User Stories (Optional)

TBD

### Notes/Constraints/Caveats (Optional)

TBD

### Risks and Mitigations

On platforms that do requires the HealthCheck port mechanism, if a user unknowingly disables the HealthCheck port allocation,
LoadBalancer-typed Services with ExternalTrafficPolicy=Local may not able to route traffic.
However, the chance is very low because this requires users to explicitly disable the HealthCheck port allocation.

## Design Details

API changes to Service:
* Add a new field `spec.allocateLoadBalancerHCNodePort: true|false`.
* `allocateLoadBalancerHCNodePort` defaults to true, preserving existing behavior.
* On create, if `allocateLoadBalancerHCNodePort: false`, don't allocate HealthCheck ports.
* On update, if `allocateLoadBalancerHCNodePort: false` don't allocate new HealthCheck ports but do not deallocate existing HealthCheck ports if set.
* On update, set `healthCheckNodePort: 0` (by removing the field) to deallocate a HealthCheck port, a new port will not be re-allocated if `allocateLoadBalancerHCNodePort: false`.
* On delete, HealthCheck ports are deallocated regardless of `allocateLoadBalancerHCNodePort`.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

No.

##### Unit tests

- `pkg/registry/core/service/storage`: `2023-10-10` - `89.9`

More unit tests will be added shortly, mainly covers:
* HealthCheck ports allocation/deallocation under different value of `allocateLoadBalancerHCNodePort`
* the default value of `allocateLoadBalancerHCNodePort`

##### Integration tests

* with the featureGate `ServiceLBHealthCheckNodePortControl` enabled,
a newly-created LoadBalancer-typed Services with ExternalTrafficPolicy=Local and `spec.allocateLoadBalancerHCNodePort=false` configured do not have healthCheckNodePort field set.
* with the featureGate `ServiceLBHealthCheckNodePortControl` enabled,
  a newly-created LoadBalancer-typed Services with ExternalTrafficPolicy=Local and `spec.allocateLoadBalancerHCNodePort=true` configured have healthCheckNodePort field set.

More tests will be added shortly.

##### e2e tests

* e2e tests to test the default behavior for `allocateLoadBalancerHCNodePort` does not break any existing e2e tests.
* e2e tests to enable, disable, disable with explictly unset `healthCheckNodePort` field, and re-enable HealthCheck ports.

More tests will be added shortly.

### Graduation Criteria

TBD

### Upgrade / Downgrade Strategy

TBD

### Version Skew Strategy

TBD

## Production Readiness Review Questionnaire


### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ServiceLBHealthCheckNodePortControl
  - Components depending on the feature gate: kube-apiserver
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, if the feature gate is disabled, new Services cannot use the new field, but existing Services
already using the field will continue to have it set. Updates to existing fields are allowed.

###### What happens if we reenable the feature if it was previously rolled back?

The existing value for `spec.allocateLoadBalancerHCNodePort` will remain intact since API strategy
will not drop fields if existing resources have it set.

###### Are there any tests for feature enablement/disablement?

Yes, there will be unit tests for the Service API strategy which exercises the behavior
with the feature gate enabled and disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

* By default this should not impact any existing Services since we are not changing any default behaviors.
* On platforms that do requires the HealthCheck port mechanism, if a user unknowingly disables the HealthCheck port allocation,
  LoadBalancer-typed Services with ExternalTrafficPolicy=Local may not able to route traffic.

###### What specific metrics should inform a rollback?

If LoadBalancer-typed Services with ExternalTrafficPolicy=Local are not able to route traffic, and have a sign of increased connection failures,
the user may consider rolling back this change.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

TBD

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The featureGate `ServiceLBHealthCheckNodePortControl` is enabled, and the field `spec.allocateLoadBalancerHCNodePort` of LoadBalancer-typed Service can be set to `true|false`.

###### How can someone using this feature know that it is working for their instance?

- [X] API .status
  - Newly-created LoadBalancer-typed Services with ExternalTrafficPolicy=Local and `spec.allocateLoadBalancerHCNodePort=false` configured do not have healthCheckNodePort field set.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

This feature only applies to certain LoadBalancer implementations which do not rely the HealthCheck port mechanism on LoadBalancer-typed Services with ExternalTrafficPolicy=Local.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- API type(s): v1 Service
- Estimated increase in size: new boolean pointer field in v1 Service, 64B for 64-bit machines.
- Estimated amount of new objects: 0

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

If the API server and/or etcd is unavailable, the Service will not be reconciled. Nothing will be changed.

###### What are other known failure modes?

- A user unknowingly disables the HealthCheck port allocation on platforms that do requires the HealthCheck port mechanism
  - Detection: increased LoadBalancer connection failure.
  - Mitigations: setting `spec.allocateLoadBalancerHCNodePort` to true for these LoadBalancer-typed Services with ExternalTrafficPolicy=Local.
  - Diagnostics: N/A
  - Testing: TBD

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 2023-10-08: Initial version

## Drawbacks

The KEP applies to a limited number of LoadBalancer implementations but adds complexity to the Service.

## Alternatives

[Adding an extra flag](https://github.com/kubernetes/kubernetes/pull/119736) on kube-apiserver to control the default behavior of HealthCheck ports allocation.
This is unacceptable due to possibly inconsistency under multi-master scenarios.

## Infrastructure Needed (Optional)

No.
