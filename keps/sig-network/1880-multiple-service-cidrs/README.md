# KEP-1880: Multiple Service CIDRs

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
    - [Story 4](#story-4)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Current implementation details](#current-implementation-details)
  - [New allocation model](#new-allocation-model)
    - [The kube-apiserver bootstrap process and the service-cidr flags](#the-kube-apiserver-bootstrap-process-and-the-service-cidr-flags)
    - [The special &quot;default&quot; ServiceCIDRConfig](#the-special-default-servicecidrconfig)
    - [Service IP Allocation](#service-ip-allocation)
    - [Service IP Reservation](#service-ip-reservation)
    - [Edge cases](#edge-cases)
    - [Resizing Service IP Ranges](#resizing-service-ip-ranges)
    - [API](#api)
    - [Allocator](#allocator)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
  - [Upgrade / Downgrade / Version Skew Strategy](#upgrade--downgrade--version-skew-strategy)
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
    - [Alternative 1](#alternative-1)
    - [Alternative 2](#alternative-2)
    - [Alternative 3](#alternative-3)
    - [Alternative 4](#alternative-4)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in
  [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
  (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance
    Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by
    [Conformance
    Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to
  [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list
  discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Allow to dynamically expand the number of IPs available for Services.

## Motivation

Services are an abstract way to expose an application running on a set of Pods. Some type of
Services: ClusterIP, NodePort and LoadBalancer use cluster-scoped virtual IP address, ClusterIP.
Across your whole cluster, every Service ClusterIP must be unique. Trying to create a Service with a
specific ClusterIP that has already been allocated will return an error.

Current implementation of the Services IP allocation logic has several limitations:

- users can not resize or increase the ranges of IPs assigned to Services, causing problems when
  there are overlapping networks or the cluster run out of available IPs.
- the Service IP range is not exposed, so it is not possible for other components in the cluster to
  consume it
- the configuration is per apiserver, there is no consensus and different apiservers can fight and delete
  others IPs.
- the allocation logic is racy, with the possibility of leaking ClusterIPs
  <https://github.com/kubernetes/kubernetes/issues/87603>
- the size of the Service Cluster CIDR, for IPv4 the prefix size is limited to /12, however, for
  IPv6 it is limited to /112 or fewer. This restriction is causing issues for IPv6 users, since /64
  is the standard and minimum recommended prefix length
- only is possible to use reserved a range of Service IP addresses using the feature gate
  "keps/sig-network/3070-reserved-service-ip-range"
  <https://github.com/kubernetes/kubernetes/issues/95570>

### Goals

Implement a new allocation logic for Services IPs that:

- scales well with the number of reservations
- the number of allocation is tunable
- is completely backwards compatible

### Non-Goals

- Any change unrelated to Services
- Any generalization of the API model that can evolve onto something different, like an IPAM API, or
  collide with existing APIs like Ingress and GatewayAPI.
- NodePorts use the same allocation model than ClusterIPs, but it is totally out of scope of this
  KEP. However, this KEP can be a proof that a similar model will work for NodePorts too.
- Change the default IPFamily used for Services IP allocation, the defaulting depends on the
  services.spec.IPFamily and services.spec.IPFamilyPolicy, a simple webhook or an admission plugin
  can set this fields to the desired default, so the allocation logic doesn't have to handle it.
- Removing the apiserver flags that define the service IP CIDRs, though that may be possible in the future.

## Proposal

The proposal is to implement a new allocator logic that uses 2 new API Objects: ServiceCIDRConfig
and IPAddress, and allow users to dynamically increase the number of Services IPs available by
creating new ServiceCIDRConfigs.

The new allocator will be able to "automagically" consume IPs from any ServiceCIDRConfig available, we can
think about this model, as the same as adding more disks to a Storage system to increase the
capacity.

To simplify the model, make it backwards compatible and to avoid that it can evolve into something
different and collide with other APIs, like Gateway APIs, we are adding the following constraints:

- a ServiceCIDRConfig will be immutable after creation (to be revisited before Beta).
- a ServiceCIDRConfig can only be deleted if there are no Service IP associated to it (enforced by finalizer).
- there can be overlapping ServiceCIDRConfigs.
- the apiserver will periodically ensure that a "default" ServiceCIDRConfig exists to cover the service CIDR flags
  and the "kubernetes.default" Service.
- any IPAddress existing in a cluster has to belong to a ServiceCIDRConfig.
- any Service with a ClusterIP assigned is expected to have always a IPAddress object associated.
- a ServiceCIDRConfig which is being deleted can not be used to allocate new IPs

 This creates a 1-to-1 relation between Service and IPAddress, and a 1-to-N relation between
 ServiceCIDRConfig and IPAddress.

 The new allocator logic can be used by other APIs, like Gateway API.

### User Stories

#### Story 1

As a Kubernetes user I want to be able to dynamically increase the number of IPs available for
Services.

#### Story 2

As a Kubernetes admin I want to have a process that allows me to renumber my Services IPs.

#### Story 3

As a Kubernetes developer I want to be able to know the current Service IP range and the IP
addresses allocated.

#### Story 4

As a Kubernetes admin that wants to use IPv6, I want to be able to follow the IPv6 address planning
best practices, being able to assign /64 prefixes to my end subnets.

### Notes/Constraints/Caveats (Optional)

Current API machinery doesn't consider transactions, Services API is kind of special in this regard
since already performs allocations inside the Registry pipeline using the Storage to keep
consistency on an Obscure API Object that stores the bitmap with the allocations. This proposal still
maintains this behavior, but instead of modifying the shared bitmap, it will create a new IPAddress object.

Changing service IP allocation to be async with regards to the service creation would be a MAJOR
semantic change and would almost certainly break clients.

### Risks and Mitigations

Cross validation of Objects is not common in the Kubernetes API, sig-apimachinery should verify that a
bad precedent is not introduced.

Current Kubernetes cluster have a very static network configuration, allowing to expand the Service
IP ranges will give more flexibility to users, with the risk of having problematic or incongruent
network configurations with overlapping. But this is not really a new problem, users need to do a
proper network planning before adding new ranges to the Service IP pool.

Service implementations,like kube-proxy, can be impacted if they were doing assumptions about the ranges assigned to the Services.
Those implementations should implement logic to watch the configured Service CIDRs.

Kubernetes DNS implementations, like CoreDNS, need to know the Service CIDRs for PTR lookups (and to know which PTR look ups they are NOT authoritative on). Those implementations should be impacted, but they can also benefit from the
new API to automatically configure the Service CIDRs.

## Design Details

### Current implementation details

[Kubernetes
Services](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#service-v1-core) need
to be able to dynamically allocate resources from a predefined range/pool for ClusterIPs.

ClusterIP is the IP address of the service and is usually assigned randomly. If an address is
specified manually, is in-range, and is not in use, it will be allocated to the service; otherwise
creation of the service will fail. The Service ClusterIP range is defined in the kube-apiserver with
the following flags:

```
--service-cluster-ip-range string
A CIDR notation IP range from which to assign service cluster IPs. This must not overlap with any IP ranges assigned to nodes or pods. Max of two dual-stack CIDRs is allowed.
```

And in the controller-manager (this is required for the node-ipam controller to avoid overlapping with the podCIDR ranges):

```
--service-cluster-ip-range string
CIDR Range for Services in cluster. Requires --allocate-node-cidrs to be true
```

The allocator is implemented using a bitmap that is serialized and stored in an ["opaque" API
object](https://github.com/kubernetes/kubernetes/blob/b246220/pkg/apis/core/types.go#L5311).

To avoid leaking resources, the apiservers run a repair loop that keep the bitmaps in sync with the current Services allocated IPs.

The order of the IP families defined in the service-cluster-ip-range flag defines the primary IP family used by the allocator.
This default IP family is used in cases where a Service creation doesn't provide the necessary information, defaulting
the Service to Single Stack with an IP chosen from the default IP family defined.

The current implementation doesn't guarantee consistency for Service IP ranges and default IP families across
multiple apiservers.

[![](https://mermaid.ink/img/pako:eNqFU0FuwjAQ_MrKZ_iAD0gpFIlbJKSectnaG1jJsVPbQaKIv9dpEhICtM4tM7OZnXEuQjlNQopAXw1ZRRvGg8eqsJAOqug8ZIAB1p4wEuzJn1hRB9foIyuu0UZ4owPbjvQMLJ2nV2iW7z7QsMbIzj7C--QBD536KWHLlsPx1fQNqaRPM4pemsFytZr6lU-Xsy69cSfy99Sd5cjJ7TfBLoctVmzOUDIZHf7UZcY41X5kbZoQye_yMBia8GDZeRvjkhBisk-H8-P4KSv3lLamrfPTIF6xN97VoDngpyF9sz_YGZmdn7uCJJxmZd3BnWLWmQSKSvd3ykQIjVIU-sDaM-N3Q27NyeSwrXjk36DfLjMJHUQmEJTI5p_J0xsjwVMKKI6SMbQ5zxCGaYPAJZD37d0axFPJzJzVbcTDA5MjFqIiXyHr9CdeWqwQ8UgVFSKphaYSGxMLUdhrojZ1ipreNafVhCwxLb0Q2ES3P1slZPQNDaT-b-5Z1x-6EVE6)](https://mermaid.live/edit#pako:eNqFU0FuwjAQ_MrKZ_iAD0gpFIlbJKSectnaG1jJsVPbQaKIv9dpEhICtM4tM7OZnXEuQjlNQopAXw1ZRRvGg8eqsJAOqug8ZIAB1p4wEuzJn1hRB9foIyuu0UZ4owPbjvQMLJ2nV2iW7z7QsMbIzj7C--QBD536KWHLlsPx1fQNqaRPM4pemsFytZr6lU-Xsy69cSfy99Sd5cjJ7TfBLoctVmzOUDIZHf7UZcY41X5kbZoQye_yMBia8GDZeRvjkhBisk-H8-P4KSv3lLamrfPTIF6xN97VoDngpyF9sz_YGZmdn7uCJJxmZd3BnWLWmQSKSvd3ykQIjVIU-sDaM-N3Q27NyeSwrXjk36DfLjMJHUQmEJTI5p_J0xsjwVMKKI6SMbQ5zxCGaYPAJZD37d0axFPJzJzVbcTDA5MjFqIiXyHr9CdeWqwQ8UgVFSKphaYSGxMLUdhrojZ1ipreNafVhCwxLb0Q2ES3P1slZPQNDaT-b-5Z1x-6EVE6)

### New allocation model

The new allocation mode requires:

- 2 new API objects ServiceCIDRConfig and IPAddress in networking.k8s.io/v1alpha1, see <https://groups.google.com/g/kubernetes-sig-api-machinery/c/S0KuN_PJYXY/m/573BLOo4EAAJ>. Both will be protected with finalizers.
- 1 new allocator implementing current `allocator.Interface`, that runs in each apiserver, and uses the new objects to allocate IPs for Services.
- 1 new controller that participates on the ServiceCIDRConfig deletion, the guarantee that each IPAddresses has
  a ServiceCIDRConfig associated. It also handles the special case for the `default` ServiceCIDRConfig.
- 1 new controller that participates in the IPAddress deletion, that guarantees that each Service IP
  allocated has its corresponding IPAddress object, recreating it if necessary.
- 1 new controller that handles the bootstrap process and the default ServiceCIDRConfig.

[![](https://mermaid.ink/img/pako:eNp9UstqwzAQ_BWxx5IcCqUHUwrFacGXYJLeqh620tpV0SNIcsAk_vfKjpNiJ3QvYmdGs7NIBxBOEmQgNIawUlh7NNyyVAPCtuT3StDhhPWV6yZE8kXJQvTK1jeYwD4-52RRvqFRWlFPjk17Rbel00q07G7av7c7Omm7G-HyYrXJna1UfYm5RkOzfEW5f7iGHifQxL0oX6T0FMJ_rhuqmPv6ScfE4VynbszJONxzYE_H5fL4PDaXIVkCUGsnMFLgMLn4t-DcYj23EM5GVHZwgAUY8gaVTA88LMEhfpMhDr1UUoWNjr2yS9JmJ9PoV6mi85BVqAMtAJvotq0VkEXf0Fk0_pNR1f0CyVS2dw)](https://mermaid.live/edit#pako:eNp9UstqwzAQ_BWxx5IcCqUHUwrFacGXYJLeqh620tpV0SNIcsAk_vfKjpNiJ3QvYmdGs7NIBxBOEmQgNIawUlh7NNyyVAPCtuT3StDhhPWV6yZE8kXJQvTK1jeYwD4-52RRvqFRWlFPjk17Rbel00q07G7av7c7Omm7G-HyYrXJna1UfYm5RkOzfEW5f7iGHifQxL0oX6T0FMJ_rhuqmPv6ScfE4VynbszJONxzYE_H5fL4PDaXIVkCUGsnMFLgMLn4t-DcYj23EM5GVHZwgAUY8gaVTA88LMEhfpMhDr1UUoWNjr2yS9JmJ9PoV6mi85BVqAMtAJvotq0VkEXf0Fk0_pNR1f0CyVS2dw)

#### The kube-apiserver bootstrap process and the service-cidr flags

Currently, the Service CIDRs are configured independently in each kube-apiserver using flags. During
the bootstrap process, the apiserver uses the first IP of each range to create the special
"kubernetes.default" Service. It also starts a reconcile loop, that synchronize the state of the
bitmap used by the internal allocators with the assigned IPs to the Services.

With current implementation, each kube-apiserver can boot with different ranges configured.
There is no conflict resolution, each apiserver keep writing and deleting others apiservers allocator bitmap
and Services.

In order to be completely backwards compatible, the bootstrap process will remain the same, the
difference is that instead of creating a bitmap based on the flags, it will create a new
ServiceCIDRConfig object from the flags (flags configuration removal is out of scope of this KEP)
with a special label `networking.kubernetes.io/service-cidr-from-flags` set to `"true"`.

It now has to handle the possibility of multiple ServiceCIDRConfig with the special label, and
also updating the configuration, per example, from single-stack to dual-stack.

The new bootstrap process will be:

```
at startup:
 read_flags
 if invalid flags
  exit
 run default-service-ip-range controller
 run apiserver

controller:
 watch all ServiceCIDR objects labelled from-flags
  ignore being-deleted ranges
 wait for first sync

controller on_event:
 if no default range matching exactly my flags
  log
  create a ServiceCIDR from my flags
   generateName: "from-flags-"
   from-flags label: "true"
 else if multiple
  log
  if multiple ranges match exactly my flags (or a single-family subset of)
    log
    delete all subsets, leaving the largest set that exactly matches on at least on family
  endif
 endif
 if kubernetes.default does not exist
  create it
```

#### The special "default" ServiceCIDRConfig

The `kubernetes.default` Service is expected to be covered by a valid range. Each apiserver will
ensure that a ServiceCIDRConfig object exists to cover its own flag-defined ranges, so this should
be true in normal cases. If someone were to force-delete the ServiceCIDRConfig covering `kubernetes.default` it
would be treated the same as any Service in the repair loop, which will generate warnings about
orphaned Service IPs.

#### Service IP Allocation

When a a new Service is going to be created and already has defined the ClusterIPs, the allocator will
just try to create the corresponding IPAddress objects, any error creating the IPAddress object will
cause an error on the Service creation. The allocator will also set the reference to the Service on the
IPAddress object.

The new allocator will have a local list with all the Service CIDRs and IPs already allocated, so it will
have to check just for one free IP in any of these ranges and try to create the object.
There can be races with 2 allocators trying to allocate the same IP, but the storage guarantees the consistency
and the first will win, meanwhile the other will have to retry.

Another racy situation is when the allocator is full and an IPAddress is deleted but the allocator takes
some time to receive the notification. One solution can be to perform a List to obtain current state, but
it will be simpler to just fail to create the Service and ask the user to try to create the Service again.

If the apiserver crashes during a Service create operation, the IPAddress is allocated but the
Service is not created, the IPaddress will be orphan. To avoid leaks, a controller will use the
`metadata.creationTimestamp` field to detect orphan objects and delete them.

There is going to be a controller to avoid leaking resources:

- checking that the corresponding parentReference on the IPAddress match the corresponding Service
- cleaning all IPAddress without an owner reference or, if the time since it was created is greater
than 60 seconds (default timeout value on the kube-apiserver )

#### Service IP Reservation

In [keps/sig-network/3070-reserved-service-ip-range](https://github.com/kubernetes/kubernetes/issues/95570) a new feature was introduced that allow to prioritize one part
of the Service CIDR for dynamically allocation, the range size for dynamic allocation depends
ont the size of the CIDR.

The new allocation logic has to be compatible, but in this case we have more flexibility and
there are different possibilities, that should be resolved before Beta.

```
<<[UNRESOLVED keps/sig-network/3070-reserved-service-ip-range]>>
Option 1: Maintain the same formula and behavior per ServiceCIDRConfig
Option 2: Define a new "allocationMode: (Dynamic | Static | Mixed)" field
Option 3: Define a new "allocationStaticThreshold: int" field
<<[/UNRESOLVED]>>
```

#### Edge cases

Since we have to maintain 3 relationships Services, ServiceCIDRConfigs and IPAddresses, we should be able
to handle edge cases and race conditions.

- Valid Service and IPAddress without ServiceCIDRConfig:

This situation can happen if a user forcefully delete the ServiceCIDRConfig, it can be recreated for the
"default" ServiceCIDRConfigs because the information is in the apiserver flags, but for other ServiceCIDRConfigs
that information is no longer available.

Another possible situation is when one ServiceCIDRConfig has been deleted, but the information takes too long
to reach one of the apiservers, its allocator will still consider the range valid and may allocate one IP
from there. To mitigate this problem, we'll set a grace period of 60 seconds on the servicecidrconfig controller
to remove the finalizer, if an IP address is created during this time we'll be able to block the deletion
and inform the user.

[![](https://mermaid.ink/img/pako:eNp1kjFvwyAQhf8KYm3ixlZVtQyRbNwhW9WsLCe4JEg2uBi3qqL890Jw67hNGBDifffuieNIpVVIGe3xfUAjsdawd9AKQ8IC6a0jFYGe1NigR7JF96El8k39xq3Z6X0CO3BeS92B8YRHHDrdBxQdKf8T9ZyoLpuVUeMOYWomTAIqslyv7zi7mYXkz0WWPz5lq2x1XzykqrSXsZbU7I_1qBobrmzMEghoGisjM7nlReLqs0vJfsuTm0oqj-qyYleCqXPiiRvDzPPOqSnTRb_NK_nU_mAHf3USwtAFbdG1oFWY6TE6CeoP2KKgLBwV7mBovKDCnAI6dCrEf1E6vDxlO2h6XFAYvN1-GUmZdwP-QOO_GKnTN31Vsn4)](https://mermaid.live/edit#pako:eNp1kjFvwyAQhf8KYm3ixlZVtQyRbNwhW9WsLCe4JEg2uBi3qqL890Jw67hNGBDifffuieNIpVVIGe3xfUAjsdawd9AKQ8IC6a0jFYGe1NigR7JF96El8k39xq3Z6X0CO3BeS92B8YRHHDrdBxQdKf8T9ZyoLpuVUeMOYWomTAIqslyv7zi7mYXkz0WWPz5lq2x1XzykqrSXsZbU7I_1qBobrmzMEghoGisjM7nlReLqs0vJfsuTm0oqj-qyYleCqXPiiRvDzPPOqSnTRb_NK_nU_mAHf3USwtAFbdG1oFWY6TE6CeoP2KKgLBwV7mBovKDCnAI6dCrEf1E6vDxlO2h6XFAYvN1-GUmZdwP-QOO_GKnTN31Vsn4)

For any Service and IPAddress that doesn't belong to a ServiceCIDRConfig the controller will raise an event
informing the user, keeping the previous behavior

```go
// cluster IP is out of range
c.recorder.Eventf(&svc, nil, v1.EventTypeWarning, "ClusterIPOutOfRange", "ClusterIPAllocation", "Cluster IP [%v]:%s is not within the service CIDR %s; please recreate service",
```

- Valid Service and ServiceCIDRConfig but not IPAddress

It can happen that an user forcefully delete an IPAddress, in this case, the controller will regenerate the IPAddress, as long as a valid ServiceCIDRConfig covers it.

During this time, there is a chance that an apiserver tries to allocate this IPAddress, with a possible situation where
2 Services has the same IPAddress. In order to avoid it, the Allocator will not delete an IP from its local cache
until it verifies that the consumer associated to that IP has been deleted too.

- Valid IPAddress and ServiceCIDRConfig but no Service

The IPAddress will be deleted and an event generated if the controller determines that the IPAddress
is orphan [(see Allocator section)](#Allocator)

- IPAddress referencing recreated Object (different UID)

1. User created Gateway "foo"
2. Gateway controller allocated IP and ref -> "foo"
3. User deleted gateway "foo"
4. Gateway controller doesn't delete the IP (leave it for GC)
5. User creates a new Gateway "foo"
6. apiserver repair loop finds the IP with a valid ref to "foo"

If the new gateway is created before the apiserver observes the delete, apiserver will find that gateway "foo"
still exists and can't release the IP. It can't peek inside "foo" to see if that is the right IP because it is
a type it does not know. If it knew the UID it could see that "foo" UID was different and release the IP.
The IPAddress will use the UID to reference the parent to avoid problems in this scenario.

#### Resizing Service IP Ranges

One of the most common problems users may have is how to scale up or scale down their Service CIDRs.

Let's see an example on how these operations will be implemented.

Assume we have configured a Service CIDR 10.0.0.0/24 and its fully used, we can:

1. Add another /24 Service CIDR 10.0.1.0/24 and keep the previous one
2. Add an overlapping larger Service CIDR 10.0.0.0/23

After 2., the user can now remove the first /24 Service CIDR, since the new Service CIDR covers all the existing IP Addresses

The same applies for scaling down, meanwhile the new Service CIDR contains all the existing IPAddresses, the old
Service CIDR will be safely deleted.

However, there can be a race condition during the operation of scaling down, since the propagation of the
deletion can take some time, one allocator can successfully allocate an IP address out of new Service CIDR (but still
inside of the old Service CIDR).

There is one controller that will periodically check that the 1-on-1 relation between IPAddresses and Services is
correct, and will start sending events to warn the user that it has to fix/recreate the corresponding Service.

#### API

```go
// ServiceCIDRConfig defines a range of IPs using CIDR format (192.168.0.0/24 or 2001:db2::0/64).
type ServiceCIDRConfig struct {
 metav1.TypeMeta   `json:",inline"`
 metav1.ObjectMeta `json:"metadata,omitempty"`

 Spec   ServiceCIDRConfigSpec   `json:"spec,omitempty"`
}


// ServiceCIDRConfigSpec describe how the ServiceCIDRConfig's specification looks like.
type ServiceCIDRConfigSpec struct {
 // IPv4 is an IPv4 block in CIDR notation "10.0.0.0/8" 
 IPv4 string `json:"ipv4"`
 // IPv6 is an IPv6 block in CIDR notation "fd12:3456:789a:1::/64" 
 IPv6 string `json:"ipv6"`
}

// IPAddress represents an IP used by Kubernetes associated to an ServiceCIDRConfig.
// The name of the object is the IP address in canonical format.
type IPAddress struct {
 metav1.TypeMeta   `json:",inline"`
 metav1.ObjectMeta `json:"metadata,omitempty"`

 Spec   IPAddressSpec   `json:"spec,omitempty"`
}

// IPAddressSpec describe the attributes in an IP Address,
type IPAddressSpec struct {
  // ParentRef references the resources (usually Services) that a IPAddress wants to be attached to.
  ParentRef ParentReference
}

type ParentReference struct {
  // Group is the group of the referent.
  Group string
  // Resource is resource of the referent.
  Resource string
  // Namespace is the namespace of the referent
  Namespace string
  // Name is the name of the referent
  Name string
  // UID is the uid of the referent
  UID string
}

```

#### Allocator

A new allocator will be implemented that implements the current Allocator interface in the apiserver.

```go
// Interface manages the allocation of IP addresses out of a range. Interface
// should be threadsafe.
type Interface interface {
 Allocate(net.IP) error
 AllocateNext() (net.IP, error)
 Release(net.IP) error
 ForEach(func(net.IP))
 CIDR() net.IPNet
 IPFamily() api.IPFamily
 Has(ip net.IP) bool
 Destroy()
 EnableMetrics()

 // DryRun offers a way to try operations without persisting them.
 DryRun() Interface
}
```

This allocator will have an informer watching Services, ServiceCIDRConfigs and IPAddresses, so it can have locally the
information needed to assign new IP addresses to Services.

IPAddresses can only be allocated from ServiceCIDRConfigs that are available and not being deleted.

The uniqueness of an IPAddress is guaranteed by the apiserver, since trying to create an IP
address that already exist will fail.

The allocator will set finalizer on the IPAddress created to avoid that there are Services without the corresponding
IP allocated.

It also will add a reference to the Service the IPAddress is associated.

### Test Plan

This is a very core and critical change, it has to be thoroughly tested on different layers:

API objects must have unit tests for defaulting and validation and integration tests exercising the
different operations and fields permutations, with both positive and negative cases: Special
attention to cross-reference validation problems, like create IPAddress reference wrong ServiceCIDRConfig or
invalid or missing

Controllers must have unit tests and integration tests covering all possible race conditions.

E2E test must be added covering the user stories defined in the KEP.

In addition to testing, it will require a lot of user feedback, requiring multiple releases to
graduate.

[x] I/we understand the owners of the involved components may require updates to existing tests to
make this code solid enough prior to committing the changes necessary to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial unit, integration and e2e tests completed and enabled
- Only basic functionality e2e tests implemented
- Define "Service IP Reservation" implementation

#### Beta

- API stability, no changes on new types and behaviors:
  - ServiceCIDR immutability
  - default IPFamily
  - two or one IP families per ServiceCIDRConfig
- Gather feedback from developers and users
- Document and add more complex testing scenarios: scaling out ServiceCIDRConfigs, ...
- Additional tests are in Testgrid and linked in KEP
- Scale test to O(10K) services and O(1K) ranges
- Improve performance on the allocation logic, O(1) for allocating a free address
- Allowing time for feedback

#### GA

- 2 examples of real-world usage
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and GA/stable, because there's
no opportunity for user feedback, or even bug reports, in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include [conformance
tests].**

[conformance tests]:
    https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

### Upgrade / Downgrade / Version Skew Strategy

The source of truth are the IPs assigned to the Services, both the old and new methods have
reconcile loops that rebuild the state of the allocators based on the assigned IPs to the Services,
this allows to support upgrades and skewed clusters.

Clusters running with the new allocation model will not keep running the reconcile loops that
keep the bitmap used by the allocators.

Since the new allocation model will remove some of the limitations of the current model, skewed
versions and downgrades can only work if the configurations are fully compatible, per example,
current CIDRs are limited to a /112 max for IPv6, if an user configures a /64 to their IPv6 subnets
in the new model, and IPs are assigned out of the first /112 block, the old allocator based in
bitmap will not be able to use those IPs creating an inconsistency in the cluster.

It will be required that those Services are recreated to get IP addresses inside the configured
ranges, for consistency, but there should not be any functional problem in the cluster if the
Service implementation (kube-proxy, ...) is able to handle those IPs.

Example:

- flags set to 10.0.0.0/20
- upgrade to 1.25 with alpha gate
- apiserver create a default ServiceCIDRConfig object for 10.0.0.0/20
- user creates a new ServiceCIDR for 192.168.1.0/24
- create a Service which gets 192.168.1.1
- rollback or disable the gate
- the apiserver repair loops will generate periodic events informing the user that the Service with the IP allocated
  is not within the configured range

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ServiceCIDRConfig
  - Components depending on the feature gate: kube-apiserver, kube-controller-manager

###### Does enabling the feature change any default behavior?

The time to reuse an already allocated ClusterIP to a Service will be longer, since the ClusterIPs
now depend on an IPAddress object that is protected by a finalizer.

The bootstrap logic has been updated to deal with the problem of multiple apiservers with different
configurations, making it more flexible and resilient.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

If the feature is disabled the old allocator logic will reconcile all the current allocated IPs
based on the Services existing on the cluster.

If there are IPAddresses allocated outside of the configured Service IP Ranges in the apiserver flags,
the apiserver will generate events referencing the Services using IPs outside of the configured range.
The user must delete and recreate these Services to obtain new IPs within the range.

###### What happens if we reenable the feature if it was previously rolled back?

There are controllers that will reconcile the state based on the current created Services,
restoring the cluster to a working state.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

Tests for feature enablement/disablement will be implemented.
TBD later, during the alpha implementation.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason:
- [ ] API .status
  - Condition name:
  - Other field:
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

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

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

### Scalability

See Drawbacks section

###### Will enabling / using this feature result in any new API calls?

When creating a Service this will require to create an IPAddress object,
previously we updated a bitmap on etcd, so we keep the number of request
but the size is reduced considerable.

###### Will enabling / using this feature result in introducing new API types?

See [API](#api)

###### Will enabling / using this feature result in any new calls to the cloud provider?

N/A

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

See [Drawbacks](#drawbacks)

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

The apiservers will increase the memory usage because they have to keep a local informer with the new objects ServiceCIDRConfig and IPAddress.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

The feature depends on the API server to work.

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

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

The number of etcd objects required scales at O(N) where N is the number of services. That seems
reasonable (the largest number of services today is limited by the bitmap size ~32k). etcd memory
use is proportional to the number of keys, but there are other objects like Pods and Secrets that
use a much higher number of objects. 1-2 million keys in etcd is a reasonable upper bound in a
cluster, and this has not a big impact (<10%). The objects described here are smaller than most
other Kube objects (especially pods), so the etcd storage size is still reasonable.

xref: Clayton Coleman <https://github.com/kubernetes/enhancements/pull/1881/files#r669732012>

## Alternatives

Several alternatives were proposed in the original PR but discarded by different reasons:

#### Alternative 1

<https://github.com/kubernetes/enhancements/pull/1881#issuecomment-672090253>

#### Alternative 2

<https://github.com/kubernetes/enhancements/pull/1881#issuecomment-672135245>

#### Alternative 3

<https://github.com/kubernetes/enhancements/pull/1881#issuecomment-672764247>

#### Alternative 4

<https://github.com/kubernetes/enhancements/pull/1881#issuecomment-755737620>

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
