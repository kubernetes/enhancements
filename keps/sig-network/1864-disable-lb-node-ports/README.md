# KEP-1864: Optionally Disable Node Ports for Service Type=LoadBalancer

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
  - [Service Type=LoadBalancerWithoutNodePorts](#service-typeloadbalancerwithoutnodeports)
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

Today, a Kubernetes Service Type=LoadBalancer will always allocate a node port for every service port. Though most implementations of Service Type=LoadBalancer do require node ports, there are several implementations that do not. This KEP proposes to add a new field to Service to opt out of node port allocation for loadbalancers.

## Motivation

There are several implementations of Service Type=LoadBalancer API today that do not consume the node ports automatically allocated by Kubernetes. Some examples include but are not limited to:
* [MetalLB](https://github.com/danderson/metallb)
* [kube-router](https://github.com/cloudnativelabs/kube-router)

The drawbacks of allocating unused node ports are that:
* the number of load balancers is now limited to the number of available node ports. Less if each load balancer consumes multiple node ports.
* node ports are exposed for an LB even though they are not used -- unecessarily exposed ports can fail regulatory/compliance requirements

For clusters that have integrations for Service Type=LoadBalancer but don't require node ports should have the option to disable node port allocation.

### Goals

* Allow users to optionally disable node port for Service Type=LoadBalancer
* Service LoadBalancer node ports can safely be re-enabled in a cluster

### Non-Goals

* Changing default values/behavior for Service Type=LoadBalancer

## Proposal

Add a new field `allocateLoadBalancerNodePorts` to `Service.Spec` that allows a user to opt out of node ports for Service Type=LoadBalancer.
  - if `allocateLoadBalancerNodePort: true`, allocate node ports for Service Type=LB, this is the existing behavior today and will be the default value.
  - if `allocateLoadBalancerNodePort: false`, stop allocating new node ports but don't deallocate existing node ports.
  - if `allocateLoadBalancerNodePort: false`, and a user wants to disable node ports on existing Services, they need to remove the nodePort field explicitly.

When a user relies on `kubectl apply` to fill in the node port field, the node port would never be deallocated since the existing node ports from the
server will always be merged prior to update. A user must send an explicit update request for the Service using something like `kubectl edit svc` or
building a controller to update all Services in a cluster.

### Risks and Mitigations

A user may unknowingly disable node ports while it is serving traffic for their pods. The chances of this should be significantly reduced since
node ports are not automatically deallocated when a user sets `allocateLoadBalancerNodePort: false`. The additional step to disable node ports
should ensure users are aware of the consequences of this change.

## Design Details

API changes to Service:
* Add a new field `spec.allocateLoadBalancerNodePorts: true|false`.
* `allocateLoadBalancerNodePorts` defaults to true, preserving existing behavior for Service Type=LoadBalancer.
* On create, if `allocateLoadBalancerNodePorts: false`, don't allocate node ports.
* On update, if `allocateLoadBalancerNodePorts: false` don't allocate new node ports but do not deallocate existing node ports if set.
* On update, set `nodePort: 0` (by removing the field) to deallocate a node port, a new port will not be re-allocated if `allocateLoadBalancerNodePorts: false`.
* On delete, node ports are deallocated regardless of `allocateLoadBalancerNodePorts`.

### Test Plan

Unit tests:
* unit tests for the allocate/deallocation of node ports based on the value of `allocateLoadBalancerNodePorts`.
* validate the default value for `allocateLoadBalanceNodePorts` on Service.

E2E tests:
* The default behavior for `allocateLoadBalancerNodePorts` does not break any existing e2e tests.
* e2e tests for disabling node ports on a Service (testing the network data path may be difficult since testing platforms uses node ports).
* e2e tests for disabling node ports on an existing Service, ensure the node ports are preserved.
* e2e tests to explicitly disable node ports with `allocateLoadBalancerNodePorts: false`.
* e2e tests to re-enable `allocateLoadBalancerNodeports` for a Service with node ports disabled.

### Graduation Criteria

### Alpha

* Adds new field `allocateLoadBalancerNodePorts` to Service, but the field is dropped unless an existing Service has the field set already.
* Only allow the field `allocateLoadBalancerNodePorts` to be set when the feature gate is on.
* There are sufficient unit tests exercising API strategy  with the feature gate enabled / disabled.

### Beta

* E2E tests checking that node ports do not get allocated when `service.spec.allocateLoadBalancerNodePorts=false`.
* Feature gate is on by default.

### GA

* Feature gate is on by default and locked.
* To safely handle rollback, there has been at least 1 release prior where apiserver understands the new field (covered in alpha).

### Upgrade / Downgrade Strategy

Upgrade should be trivial since kube-proxy's behavior of the Service is determined by whether a node port is set.
For existing Services that are upgraded, `nodePort` will continue to be set (even if `allocateLoadBalancerNodePort: false`).

On downgrade, if `allocateLoadBalancerNodePorts: false`, the worse case is that a Service which was intended to have
node ports disabled will now have node ports re-enabled. Assuming the loadbalancer implementation never relied on the node port,
re-enabling node port should not cause any traffic disruptions.

### Version Skew Strategy

Version skew from the control plane to kube-proxy should be trivial since kube-proxy's behavior is driven by the `nodePort` field
and not the `allocateLoadBalancerNodePorts` field.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: ServiceLBNodePortControl
    - Components depending on the feature gate: kube-apiserver
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**

No, enabling the feature gate but not setting `spec.allocateLoadBalancerNodePorts` will not
change any default behaviors in Service.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**

  Yes, if the feature gate is disabled, new Services cannot use the new field, but existing Services
  already using the field will continue to have it set. Updates to existing fields are allowed.

* **What happens if we reenable the feature if it was previously rolled back?**

  The existing value for `spec.allocateLoadBalancerNodePorts` will remain intact since API strategy
  will not drop fields if existing resources have it set.

* **Are there any tests for feature enablement/disablement?**

  Yes, there will be unit tests for the Service API strategy which exercises the behavior
  with the feature gate enabled and disabled.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**

* By default this should not impact any existing Services since we are not changing any default behaviors.
* Enabling this feature on new clusters can impact workloads if load balancers depend on node ports without users
being aware.

* **What specific metrics should inform a rollback?**

Metrics for node port counts will vary for Service LoadBalancers that set `spec.allocateLoadBalancerNodeports=false`.
If load balancers are misbehaving at the same time node port allocation metric is decreasing, the user may want to
consider rolling back this feature.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

No, upgrade->downgrade->upgrade has not been tested yet. Like any new API field, on downgrade
any existing Services using the field will continue to have the field set. For these Services,
they will not have node ports allocated. New Services cannot use the new field unless the feature
gate is enabled in the old version when the feature was alpha.

Manual validation of this behavior should be done prior to promoting this feature to beta.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**

No.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**

Service should have `spec.allocateLoadBalancerNodePorts=false` and Service LoadBalancers will not have node ports allocated.

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
should only be used if the load balancer implementation does not need node ports for the load balancer
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
the feature is to disable an existing behavior with node ports.

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

Not any different from when node ports are used for load balancers.

* **What are other known failure modes?**

If `service.spec.allocateLoadBalancerNodePorts=false` but the load balancer implementation does depend on node ports.

* **What steps should be taken if SLOs are not being met to determine the problem?**

In a scenario where a user sets `service.spec.allocateLoadBalancerNodePorts=false` but the load balancer does require node ports,
the user can re-enable node ports for a Service by setting `service.spec.allocateLoadBalancerNodePorts` back to `true`.
This will trigger node port allocation from kube-apiserver.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- 2020-06-17: KEP is proposed as implementable

## Drawbacks

More fields == more complexity in Service.

## Alternatives

### APIServer Flag

We can add an apiserver flag that toggles the node port behavior for Service Type=LoadBalancer. This requires the cluster admin to
understand the Service Type=LoadBalancer implementation and set this flag accordingly. In general we should try to avoid global flags
if it's feasible to drive the behavior through the API.

### Service Type=LoadBalancerWithoutNodePorts

A new Service Type like `LoadBalancerWithoutNodePorts` can be added, which would have similar semantics as Service Type=LoadBalancer
without node ports. This wouldn't be a good experience for existing consumers of Service Type=LoadBalancer that now have to watch a new Type.

