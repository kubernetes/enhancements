# KEP-1860 Make Kubernetes aware of the LoadBalancer behaviour

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Design Details](#design-details)
- [Test Plan](#test-plan)
- [Graduation Criteria](#graduation-criteria)
  - [Alpha](#alpha)
  - [Beta/GA](#betaga)
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
<!-- /toc -->

## Summary

The different kube-proxy implementations (at least ipvs and iptables), as of today, are binding the External IPs of the LoadBalancer Service to each node. (iptables are creating some rules to redirect packets directly to the service and ipvs is binding the IP to one interface on the node). This feature exists because:

1. To redirect pod traffic that is intended for LoadBalancer IP directly to the service, bypassing the loadbalancer for path optimization.
2. Some LBs are sending packet with the destination IP equal to the LB IP, so the IP needs to be routed directly to the right service (otherwise looping back and forth)

This enhancement proposes a way to make these features configurable, giving the cloud provider, a way to disable the behaviour of kube-proxy.
The best way would be to set a default behaviour at the cloud controller level.

## Motivation

There are numerous problems with the current behaviour:

1. Some cloud providers (Scaleway, Tencent Cloud, ...) are using the LB's external IP (or a private IP) as source IP when sending packets to the cluster. This is a problem in the ipvs mode of kube-proxy since the IP is bounded to an interface and healthchecks from the LB are never coming back.
2. Some cloud providers (DigitalOcean, Scaleway, ...) have features at the LB level (TLS termination, PROXY protocol, ...). Bypassing the LB means missing these features when the packet arrives at the service (leading to protocols errors).

So, giving options to these cloud providers to disable the actual behaviour would be very valuable.

Currently there is some hacky workaround that set the `Hostname` on the `Service` in order to bypass kube-proxy binding (AWS and DigitalOcean for instance), but as stated, it's a bit hacky. This KEP should also give a solution to this hack.

### Goals

* Expose an option for the different controllers managing Load Balancer (e.g. the Cloud Controller Manager), to choose the actual behaviour of kube-proxy for all services.

### Non-Goals

* Deprecate the default behaviour of kube-proxy
* Change the default behaviour of kube-proxy

## Proposal

The solution would be to add a new field in the `loadBalancer` field of a `Service`'s `status`, like `ipMode`. This new field will be used by kube-proxy in order to not bind the Load Balancer's External IP to the node (in both IPVS and iptables mode). The value `VIP` would be the default one (if not set for instance), keeping the current behaviour. The value `Proxy` would be used in order to disable the the shortcut.

Since the `EnsureLoadBalancer` returns a `LoadBalancerStatus`, the Cloud Controller Manager can optionally set the `ipMode` field before returning the status.

### User Stories

#### Story 1

User 1 is a Managed Kubernetes user. There cluster is running on a cloud provider where the LB's behaviour matches the `VIP` `ipMode`.
Nothing changes for them since the cloud provider manages the Cloud Controller Manager.

#### Story 2

User 2 is a Managed Kubernetes user. There cluster is running on a cloud provider where the LB's behaviour matches the `Proxy` `ipMode`.
Almost nothing changes for them since the cloud provider manages the Cloud Controller Manager.
The only difference is that pods using the load balancer IP may observe a higher response time since the datapath will go through the load balancer. 

#### Story 3

User 3 is a developer working on a cloud provider Kubernetes offering. 
On the next version of `k8s.io/cloud-provider`, the cloud provider can optionally set the `ipMode` field as part of `LoadBalancerStatus`, and reflect the cloud provider load balancer behaviour.

### Risks and Mitigations

1. The first risk is when using the `Proxy` `ipMode` for pod using the load balancer IP. In this case the packets will not bypass the load balancer anymore.
However if a user wants to to keep using the in cluster datapath, he can still use the ClusterIP of the service.

## Drawbacks

1. Performance hit, see the risk 1. above.

## Alternatives

A viable alternative may be to use a flag directly on `kube-proxy`.
When running on a cloud provider managed Kubernetes this solution is viable since the cloud provider will be able to set the right value on `kube-proxy`.
When running a self hosted cluster, the user needs to be aware of how the cloud's load balancers works and need to set the flag on `kube-proxy` accordingly.

## Design Details

API changes to Service:
- Add a new field `status.loadBalancer.ingress[].ipMode: VIP | Proxy`.
- `ipMode` defaults to VIP if the feature gate is enabled, `nil` otherwise, preserving existing behaviour for Service Type=LoadBalancer.
- On create and update, it fails if `ipMode` is set without the `ip` field.

## Test Plan

Unit tests:
- unit tests for the ipvs and iptables rules
- unit tests for the validation

Integration tests:
- https://github.com/kubernetes/kubernetes/pull/125237

E2E tests:
- The default behavior for `ipMode` does not break any existing e2e tests
- The default `VIP` value is already tested
- https://github.com/kubernetes/kubernetes/pull/124876

## Graduation Criteria

### Alpha

Adds new field `ipMode` to Service, which is used when `LoadBalancerIPMode` feature gate is enabled, allowing for rollback.

### Beta/GA

`LoadBalancerIPMode` is enabled by default.

### Upgrade / Downgrade Strategy

On upgrade, while the feature gate is disabled, nothing will change. Once the feature gate is enabled, 
all the previous LoadBalancer service will get an `ipMode` of `VIP` by the defaulting function when we get them from kube-apiserver(xref https://github.com/kubernetes/kubernetes/pull/118895/files#r1248316868).
If `kube-proxy` was not yet upgraded: the field will simply be ignored.
If `kube-proxy` was upgraded, and the feature gate enabled, it will stil behave as before if the `ipMode` is `VIP`, and will behave accordingly if the `ipMode` is `Proxy`.

On downgrade, the feature gate will simply be disabled, and as long as `kube-proxy` was downgraded before, nothing should be impacted.

### Version Skew Strategy

Version skew from the control plane to `kube-proxy` should be trivial since `kube-proxy` will simply ignore the `ipMode` field.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: LoadBalancerIPMode
  - Components depending on the feature gate: kube-proxy, kube-apiserver, cloud-controller-manager

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, by disabling the feature gate. Disabling it in kube-proxy is necessary and sufficient to have a user-visible effect.

###### What happens if we reenable the feature if it was previously rolled back?

It works. The forwarding rules for services which have the value of `ipMode` had been set to "Proxy" will be removed by kube-proxy.

###### Are there any tests for feature enablement/disablement?

Yes. It is tested by `TestUpdateServiceLoadBalancerStatus` in pkg/registry/core/service/storage/storage_test.go.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

As the rollout will enable a feature not being used yet, there is no possible failure
scenario as this feature will then need to be also enabled by the cloud provider on
the services resources.

In case of a rollback, kube-proxy will also rollback to the default behavior, switching 
back to VIP mode. This can fail for workloads that may be already relying on the 
new behavior (eg. sending traffic to the LoadBalancer expecting some additional
features, like PROXY and TLS Termination as per the Motivations section).

###### What specific metrics should inform a rollback?

If using kube-proxy, looking at metrics `sync_proxy_rules_duration_seconds` and 
`sync_proxy_rules_last_timestamp_seconds` may help identifying problems and indications
of a required rollback.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Because this is a feature that depends on CCM/LoadBalancer controller, and none yet 
implements it, the scenario is simulated with the upgrade/downgrade/upgrade path being
enabling and disabling the feature flag, and doing the changes on services status subresources.

There is a LoadBalancer running on the environment (metallb) that is responsible for doing the proper 
LB ip allocation and announcement, but the rest of the test is related to kube-proxy programming
or not the iptables rules based on this enablement/disablement path

* Initial scenario
  * Started with a v1.29 cluster with the feature flag enabled
  * Created 3 Deployments:
    * web1 - Will be using the new feature
    * web2 - Will NOT be using the new feature
    * client -  "the client"
  * Created the loadbalancer for the two web services. By default both LBs are with the default `VIP` value
```yaml
status:
  loadBalancer:
    ingress:
    - ip: 172.18.255.200
      ipMode: VIP
```
  * With the feature flag enabled but no change on the service resources, tested and both
web deployments were accessible
    * Verified that the iptables rule for both LBs exists on all nodes
* Testing the feature ("upgrade")
  * Changed the `ipMode` from first LoadBalancer to `Proxy`
    * Verified that the iptables rule for the second LB still exists, while the first one didn't
  * Because the LoadBalancer of the first service is not aware of this new implementation (metallb), it is 
  not accessible anymore from the client Pod
  * The second service, which `ipMode` is `VIP` is still accessible from the Pods
* Disable the feature flag ("downgrade")
  * Edit kube-apiserver manifest and disable the feature flag
  * Edit kube-proxy configmap, disable the feature and restart kube-proxy Pods
  * Confirmed that both iptables rules are present, even if the `ipMode` field was still 
  set as `Proxy`, confirming the feature is disabled. Both accesses are working

Additionally, an apiserver and kube-proxy upgrade test was executed as the following:
* Created a KinD cluster with v1.28
* Created the same deployments and services as bellow
  * Both loadbalancer are accessible
* Upgraded apiserver and kube-proxy to v1.29, and enabled the feature flag
* Set `ipMode` as `Proxy` on one of the services and execute the same tests as above
  * Observed the expected behavior of iptables rule for the changed service 
  not being created
  * Observed that the access of the changed service was not accessible anymore, as
  expected
* Disable feature flag 
* Rollback kube-apiserver and kube-proxy to v1.28
* Verified that both services are working correctly on v1.28
* Upgraded again to v1.29, keeping the feature flag disabled
  * Both loadbalancers worked as expected, the field is still present on 
  the changed service.


###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

If the LB IP works correctly from pods, then the feature is working

###### How can someone using this feature know that it is working for their instance?

- [X] API .status
  - Condition name:
  - Other field: `.status.loadBalancer.ingress.ipMode` not null
- [X] Other:
  - Details: To detect if the traffic is being directed to the LoadBalancer and not 
    directly to another node, the user will need to rely on the LoadBalancer logs, 
    and the destination workload logs to check if the traffic is coming from one Pod
    to the other or from the LoadBalancer.


###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The quality of service for clouds using this feature is the same as the existing 
quality of service for clouds that don't need this feature

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

* On kube-proxy, a metric containing the count of IP programming vs service type would be useful
to determine if the feature is being used, and if there is any drift between nodes

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- cloud controller manager /  LoadBalancer controller
  - If there is an outage of the cloud controller manager, the result is the same 
  as if this feature wasn't in use; the LoadBalancers will get out of sync with Services
- kube-proxy or other Service Proxy that implements this feature
  - If there is a service proxy outage, the result is the same as if this feature wasn't in use

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- API type: v1/Service
- Estimated increase size: new string field. Supported options at this time are max 6 characters (`Proxy`)
- Estimated amount of new objects: 0

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.


###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Same for any loadbalancer/cloud controller manager, the new IP and the new status will not be 
set.

kube-proxy reacts on the IP status, so the service LoadBalancer IP and configuration will be pending.

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?
N/A

## Implementation History

- 2020-06-18: Initial KEP
- 2023-12-14: Alpha release with Kubernetes 1.29
- 2024-04-18: Beta release with Kubernetes 1.30
- 2024-12-11: GA release with Kubernetes 1.32
