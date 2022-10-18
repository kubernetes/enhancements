# KEP-3622: Optionally Disable ClusterIP for Service Type=LoadBalancer

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
  - [APIServer Flag](#apiserver-flag)
  - [Service Type=LoadBalancerWithoutClusterIP](#service-typeloadbalancerwithoutip)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [X] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Today, a Kubernetes Service Type=LoadBalancer will always allocate a clusterIP for every service. Though most implementations of Service Type=LoadBalancer do require clusterIP, there are several implementations that do not. When a large number of Type=LoadBalancer services(in one clsuter) do not need to allocate cluster IP, it will cause a waste of cluster IP resources, and the number of loadBalancer services created in this way is also limited by the number of ClusterIPs in the cluster. 

This KEP proposes to add a new field to Service to opt out of clusterIP allocation for loadbalancers.
## Motivation

There are several implementations of Service Type=LoadBalancer API today that do not consume the clusterIP automatically allocated by Kubernetes. 

Creating a loadbalancer service in this way brings a lot of limitations:
* the number of load balancers is now limited to the number of available clusterIP. 
* clusterIP are allocated for an LB even though they are not used.

For clusters that have integrations for Service Type=LoadBalancer but don't require clusterIP should have the option to disable clusterIP allocation.

### Goals

* Allow users to optionally disable clusterIP for Service Type=LoadBalancer

### Non-Goals

* Changing default values/behavior for Service Type=LoadBalancer
* Removing existing ClusterIP from Service Type=LoadBalancer.

## Proposal

Add a new field `allocateLoadBalancerClusterIP` to `Service.Spec` that allows a user to opt out of clusterIP for Service Type=LoadBalancer.
  - if `allocateLoadBalancerClusterIP: true`, allocate clusterIP for Service Type=LB, this is the existing behavior today and will be the default value.
  - if `allocateLoadBalancerClusterIP: false`, do not allocate clusterIP for Service Type=LB.


### Risks and Mitigations

A user may unknowingly disable clusterIP while it is serving traffic for their pods. The chances of this should be significantly reduced since
clusterIP are not automatically deallocated when a user sets `allocateLoadBalancerClusterIP: false`. The additional step to disable clusterIP
should ensure users are aware of the consequences of this change.

## Design Details

API changes to Service:
* Add a new field `spec.allocateLoadBalancerClusterIP: true|false`.
* `allocateLoadBalancerClusterIP` defaults to true, preserving existing behavior for Service Type=LoadBalancer.
* On create, if `allocateLoadBalancerClusterIP: false`, don't allocate clusterIP.
* On update, set `allocateLoadBalancerClusterIP: true` allocate clusterIP for Service Type=LoadBalancer.
* On update, if `allocateLoadBalancerClusterIP: false` deny the update request. Like the existing behavior, once the clusterIP has been assigned, it refuses to be modified.
* On delete, clusterIP are deallocated regardless of `allocateLoadBalancerClusterIP`.

### Test Plan

Unit tests:
* unit tests for the allocate of clusterIP based on the value of `allocateLoadBalancerClusterIP`.
* validate the default value for `allocateLoadBalancerClusterIP` on Service.

E2E tests:
* The default behavior for `allocateLoadBalancerClusterIP` does not break any existing e2e tests.
* e2e tests to explicitly disable clusterIP with `allocateLoadBalancerClusterIP: false`.
* e2e tests to re-enable `allocateLoadBalancerClusterIP` for a Service with clusterIP disabled.

### Graduation Criteria

### Alpha

* Adds new field `allocateLoadBalancerClusterIP` to Service, but the field is dropped unless an existing Service has the field set already.
* Only allow the field `allocateLoadBalancerClusterIP` to be set when the feature gate is on.
* There are sufficient unit tests exercising API strategy  with the feature gate enabled / disabled.

### Beta

* E2E tests checking that clusterIP do not get allocated when `service.spec.allocateLoadBalancerClusterIP=false`.
* Feature gate is on by default.

### GA

* Feature gate is on by default and locked.
* To safely handle rollback, there has been at least 1 release prior where apiserver understands the new field (covered in alpha).

### Upgrade / Downgrade Strategy

Upgrade should be trivial since kube-proxy's behavior of the Service is determined by whether a clusterIP is set.
For existing Services that are upgraded, `clusterIP` will continue to be set (even if `allocateLoadBalancerClusterIP: false`).

On downgrade, if `allocateLoadBalancerClusterIP: false`, the worse case is that a Service which was intended to have
clusterIP disabled will now have clusterIP re-enabled. Assuming the loadbalancer implementation never relied on the clusterIP,
re-enabling clusterIP should not cause any traffic disruptions.

### Version Skew Strategy

Version skew from the control plane to kube-proxy should be trivial since kube-proxy's behavior is driven by the `clusterIP` field
and not the `allocateLoadBalancerClusterIP` field.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: ServiceLBClusterIPControl
    - Components depending on the feature gate: kube-apiserver
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**

No, enabling the feature gate but not setting `spec.allocateLoadBalancerClusterIP` will not
change any default behaviors in Service.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**

  Yes, if the feature gate is disabled, new Services cannot use the new field, but existing Services
  already using the field will continue to have it set. Updates to existing fields are allowed.

* **What happens if we reenable the feature if it was previously rolled back?**

  The existing value for `spec.allocateLoadBalancerClusterIP` will remain intact since API strategy
  will not drop fields if existing resources have it set.

* **Are there any tests for feature enablement/disablement?**

  Yes, there will be unit tests for the Service API strategy which exercises the behavior
  with the feature gate enabled and disabled.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**

* By default this should not impact any existing Services since we are not changing any default behaviors.
* Enabling this feature on new clusters can impact workloads if load balancers depend on clusterIP without users
being aware.

* **What specific metrics should inform a rollback?**

None.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

No, upgrade->downgrade->upgrade has not been tested yet. Like any new API field, on downgrade
any existing Services using the field will continue to have the field set. For these Services,
they will not have clusterIP allocated. New Services cannot use the new field unless the feature
gate is enabled in the old version when the feature was alpha.

Manual validation of this behavior should be done prior to promoting this feature to beta.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**

No.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**

Service should have `spec.allocateLoadBalancerClusterIP=false` and Service LoadBalancers will not have clusterIP allocated.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**

N/A

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

N/A

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**

N/A

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**

This feature is dependent on the Service LoadBalancer implementation of a cluster. This feature
should only be used if the load balancer implementation does not need clusterIP for the load balancer
data path.


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  Describe them, providing:

No, enabling this feature should actually reduce the number of operations, since
the feature is to disable an existing behavior with clusterIP.

* **Will enabling / using this feature result in introducing new API types?**

No

* **Will enabling / using this feature result in any new calls to the cloud
provider?**

No

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**

No

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**

No

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**

No

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

Not any different from when clusterIP are used for load balancers.

* **What are other known failure modes?**

If `service.spec.allocateLoadBalancerClusterIP=false` but the load balancer implementation does depend on clusterIP.

* **What steps should be taken if SLOs are not being met to determine the problem?**

In a scenario where a user sets `service.spec.allocateLoadBalancerClusterIP=false` but the load balancer does require clusterIP,
the user can re-enable clusterIP for a Service by setting `service.spec.allocateLoadBalancerClusterIP` back to `true`.
This will trigger clusterIP allocation from kube-apiserver.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- 2020-06-17: KEP is proposed as implementable

## Drawbacks

More fields == more complexity in Service.

## Alternatives

### APIServer Flag

We can add an apiserver flag that toggles the clusterIP behavior for Service Type=LoadBalancer. This requires the cluster admin to
understand the Service Type=LoadBalancer implementation and set this flag accordingly. In general we should try to avoid global flags
if it's feasible to drive the behavior through the API.

### Service Type=LoadBalancerWithoutClusterIP

A new Service Type like `allocateLoadBalancerClusterIP` can be added, which would have similar semantics as Service Type=LoadBalancer
without clusterIP. This wouldn't be a good experience for existing consumers of Service Type=LoadBalancer that now have to watch a new Type.

