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
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [APIServer Flag](#apiserver-flag)
  - [Service Type=LoadBalancerWithoutNodePorts](#service-typeloadbalancerwithoutnodeports)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
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

