---
title: Different protocols in the same Service definition with type=LoadBalancer
authors:
  - "@janosi"
owning-sig: sig-network
participating-sigs:
  - sig-cloud-provider
reviewers:
  - "@thockin"
  - "@dcbw"
  - "@andrewsykim"
approvers:
  - "@thockin"
editor: TBD
creation-date: 2020-01-03
last-updated: 2020-01-03
status: provisional
see-also:
replaces:
superseded-by:
---

# different protocols in the same service definition with type=loadbalancer

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories [optional]](#user-stories-optional)
    - [Story 1](#story-1)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    - [Alibaba](#alibaba)
    - [AWS](#aws)
    - [Azure](#azure)
    - [GCE](#gce)
    - [IBM Cloud](#ibm-cloud)
    - [OpenStack](#openstack)
    - [Oracle Cloud](#oracle-cloud)
    - [Tencent Cloud](#tencent-cloud)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Option Control Alternatives](#option-control-alternatives)
    - [Annotiation in the Service definition](#annotiation-in-the-service-definition)
    - [Merging Services in CPI](#merging-services-in-cpi)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Examples](#examples)
      - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
      - [Beta -&gt; GA Graduation](#beta---ga-graduation)
      - [Removing a deprecated flag](#removing-a-deprecated-flag)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
- [Infrastructure Needed [optional]](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR) https://github.com/kubernetes/enhancements/issues/1435
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This feature enables the creation of a LoadBalancer Service that has different port definitions with different protocols. 

## Motivation

The ultimate goal of this feature is to support users that want to expose their applications via a single IP address but different L4 protocols with a cloud provided load-balancer. 
The following issue and PR shows considerable interest from users that would benefit from this feature:
https://github.com/kubernetes/kubernetes/issues/23880
https://github.com/kubernetes/kubernetes/pull/75831

### Goals

The goals of this KEP are:

- To analyze the impact of this feature with regard to current implementations of cloud-provider load-balancers
- define how the activation of this feature could be made configurable if certain cloud-provider load-balancer implementations do not want to provide this feature

### Non-Goals


## Proposal

The first thing proposed here is to lift the hardcoded limitation in Kubernetes that currently rejects Service definitions with different protocols if their type is LoadBalancer. Kubernetes would not reject Service definitions like this from that point:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: mixed-protocol
spec:
  type: LoadBalancer
  ports:
    - name: dns-udp
      port: 53
      protocol: UDP
    - name: dns-tcp
      port: 53
      protocol: TCP
  selector:
    app: my-dns-server
  ```

Once that limit is removed those Service definitions will reach the Cloud Provider LB controller implementations. The logic of the particular Cloud Provider LB controller and of course the actual capabilities and architecture of the backing Cloud Load Balancer services determines how the actual exposure of the application really manifests. For this reason it is important to understand the capabilities of those backing services and to design this feature accordingly.

### User Stories [optional]

#### Story 1

As a Kubernetes cluster user I want to expose an application that provides its service on different protocols with a single cloud provider load balancer IP. In order to achieve this I want to define different `ports` with mixed protocols in my Service definition of `type: LoadBalancer`

### Implementation Details/Notes/Constraints [optional]

#### Alibaba

The Alibaba Cloud Provider Interface Implementation (CPI) supports TCP, UDP and HTTPS in Service definitions and can configure the SLB listeners with the protocols defined in the Service.

The Alibaba SLB supports TCP, UDP and HTTPS listeners. A listener must be configured for each protocol, and then those listeners can be assigned to the same SLB instance. 

The number of listeners does not affect SLB pricing.
https://www.alibabacloud.com/help/doc-detail/74809.htm

A user can ask for an internal TCP/UDP Load Balancer via a K8s Service definition that also has the annotation `service.beta.kubernetes.io/alibaba-cloud-loadbalancer-address-type: "intranet"`. Internal SLBs are free.

Summary: Alibaba CPI and SLB seems to support mixed protocol Services, and the pricing is not based on the number of protocols per Service.

#### AWS

The AWS CPI does not support mixed protocols in the Service definition since it only allows TCP for load balancers. The AWS CPI looks for annotations on the Service to determine whether TCP, TLS or HTTP(S) listener should be created in the AWS ELB for a configured Service port.

AWS Classic LB supports TCP,TLS, and HTTP(S) protocols behind the same IP address. 

AWS Network LB supports TCP/TLS and UDP protocols behind the same IP address. As we can see, UDP cannot be utilized currently, due to the limitation in the AWS CPI.

The usage of TCP+HTTP or UDP+HTTP on the same LB instace behind the same IP address is not possible in AWS.

From a pricing perspective the AWS NLB and the CLB have the following models:
https://aws.amazon.com/elasticloadbalancing/pricing/
Both are primarily usage based rather than charging based on the number of protocol, however NLBs have separate pricing unit quotas for TCP, UDP and TLS.

A user can ask for an internal Load Balancer via a K8s Service definition that also has the annotation `service.beta.kubernetes.io/aws-load-balancer-internal: 0.0.0.0/0`. So far the author could not find any difference in the usage and pricing of those when compared to the external LBs - except the pre-requisite of a private subnet on which the LB can be deployed.

Summary: AWS CPI is the current bottleneck with its TCP-only limitation. As long as it is there the implementation of this feature will have no effect on the AWS bills.

#### Azure

Azure CPI LB documentation: https://github.com/kubernetes-sigs/cloud-provider-azure/blob/master/docs/services/README.md

The Azure CPI already supports the usage of both UDP and TCP protocols in the same Service definition. It is achieved with the CPI specific annotation `service.beta.kubernetes.io/azure-load-balancer-mixed-protocols`. If this key has value of `true` in the Service definition, the Azure CPI adds the other protocol value (UDP or TCP) to its internal Service representation. Consequently, it also manages twice the amount of load balancer rules for the specific frontend. 

Only TCP and UDP are supported in the current mixed protocol configuration.

The Azure Load Balancer supports only TCP and UDP as protocols. HTTP support would require the usage of the Azure Application Gateway. I.e. HTTP and L4 protocols cannot be used on the same LB instance/IP address

Basic Azure Load Balancers are free.

Pricing of the Standard Azure Load Balancer is based on load-balancing rules and outbound rules.  There is a flat price up to 5 rules, on top of which every new forwarding rule has additional cost. 
https://docs.microsoft.com/en-us/azure/load-balancer/load-balancer-overview#pricing

A user can ask for an internal Azure Load Balancer via a K8s Service definition that also has the annotation `service.beta.kubernetes.io/azure-load-balancer-internal: true`. There is not any limitation on the usage of mixed service protocols with internal LBs in the Azure CPI.

Summary: Azure already has mixed protocol support for TCP and UDP, and it already affects the bills of the users. The implementation of this feature may require some work in Azure CPI.

#### GCE

The GCE CPI supports only TCP and UDP protocols in Services.

GCE/GKE creates Network Load Balancers based on the K8s Services with type=LoadBalancer. In GCE there are "forwarding rules" that define how the incoming traffic shall be forwarded to the compute instances. A single forwarding rule can support either TCP or UDP but not both. In order to have both TCP and UDP forwarding rules we have to create separate forwarding rule instances for those. Two or more forwarding rules can share the same external IP if
- the network load balancer type is External
- the external IP address is not ephemeral but static

There is a workaround in GCE, please see this comment from the original issue: https://github.com/kubernetes/kubernetes/issues/23880#issuecomment-269054735 If the external IP is static the user can create two Service instances, one for UDP and another for TCP and the user has to specify that static external IP address in those Service definitions in `loadBalancerIP`.

HTTP protocol support: there is a different LB type in GCE for HTTP traffic: HTTP(S) LB. I.e. just like in the case of AWS it is not possible to have e.g. TCP+HTTP or UDP+HTTP behind the same LB/IP address.

Forwarding rule pricing is per rule: there is a flat price up to 5 forwarding rule instances, on top of which every new forwarding rule has additional cost. 

https://cloud.google.com/compute/network-pricing#forwarding_rules_charges

[GCP forwarding_rules_charges](https://cloud.google.com/compute/network-pricing#forwarding_rules_charges) suggest that the same K8s Service definition would result in the creation of 2 forwarding rules in GCP. This has the same fixed price up to 5 forwarding rule instances, and each additional rule results in extra cost.

A user can ask for an internal TCP/UDP Load Balancer via a K8s Service definition that also has the annotation `cloud.google.com/load-balancer-type: "Internal"`. Forwarding rules are also part of the GCE Internal TCP/UDP Load Balancer architecture, but in case of Internal TCP/UDP Load Balancer it is not supported to define different forwarding rules with different protocols for the same IP address. That is, for Services with type=LoadBalancer and with annotation `cloud.google.com/load-balancer-type: "Internal"` this feature would not be supported.

Summary: The implementation of this feature can affect the bills of GCP users. However the following perspectives are also observed:
- if a user wants to have UDP and TCP ports behind the same NLB two Services with must be defined, one for TCP and one for UDP. As the pricing is based on the number of forwarding rules this setup also means the same pricing as with the single Service instance.
- if a user is happy with 2 NLB (Service) instances for TCP and UDP still the user has two more forwarding rules to be billed - i.e. it has the same effect on pricing as if those TCP and UDP endpoints were behind the same NLB instance
- already now the bills of users is affected if they have more than 5 forwarding rules as the result of their current Service definitions (e.g. 6 or more port definitions in a single Serice, or if the number of all port definitions in different Services is 6 or more, etc.)

That is, if we consider the "single Service with 6 ports" case the user has to pay more for that single Service ("single LB instance") than for another Service (another LB instance) with 5 or less ports already now. It is not the number of LBs that matters. This phenomenon is already there with the current practice, and the enabling of mixed protocols will not change it to the worse.

#### IBM Cloud

The IBM Cloud CPI implementation supports TCP, UDP, HTTP protocol values in K8s Services, and it supports mutiple protocols in a Service. The IBM Cloud CPI creates VPC Load Balancer in a VPC based cluster and NLB in a classic cloud based cluster.

The VPC Load Balancer supports TCP and HTTP, and it is possible to create TCP and HTTP listeners for the same LB instance. UDP is not supported. 

The VPC LB pricing is time and data amount based, i.e. the number of protocols on the same LB instance does not affect it.

The NLB supports TCP and UDP on the same NLB instance. The usage of NLB does not have pricing effects, it is part of the IKS basic package.

Summary: once this feature is implemented IBM Cloud VPC LB can use TCP and HTTP ports from a single Service definition. NLB can use TCP and UDP ports from a single Service definition.

#### OpenStack

The OpenStack CPI supports TCP, UDP, HTTP(S) in Service definitions and can configure the Octavia listeners with the protocols defined in the Service.
OpenStack Octavia supports TCP, UDP and HTTP(S) on listeners, an own listener must be configured for each protocol, and different listeners can be used on the same LB instance.

Summary: the OpenStack based clouds that use Octavia v2 as their LBaaS seems to support this feature once implemented. Pricing is up to their model.

#### Oracle Cloud

Oracle Cloud supports TCP, HTTP(S) protocols in its LB solution. The Oracle CPI also supports the protocols in he K8s Service definitions.

The pricing is based on time and capacity: https://www.oracle.com/cloud/networking/load-balancing-pricing.html I.e. the amount of protocols on a sinlge LB instance does not affect the pricing.

#### Tencent Cloud

The Tencent Cloud CPI supports TCP, UDP and HTTP(S) in Service definitions and can configure the CLB listeners with the protocols defined in the Service.
The Tencent Cloud CLB supports TCP, UDP and HTTP(S) listeners, an own listener must be configured for each protocol. 
The number of listeners does not affect CLB pricing. CLB pricing is time (day) based and not tied to the number of listeners.
https://intl.cloud.tencent.com/document/product/214/8848

A user can ask for an internal Load Balancer via a K8s Service definition that  has the annotation `service.kubernetes.io/qcloud-loadbalancer-internal-subnetid: subnet-xxxxxxxx`. Internal CLBs are free.


### Risks and Mitigations

The goal of the current restriction on the K8s API was to prevent an unexpected extra charging for Load Balancers that were created based on Services with mixed protocol definitions.
If the current limitation is removed without any option control we introduce the same risk. Let's see which clouds are exposed:
- Alibaba: the pricing here is not protocol or forwarding rule or listener based. No risk.
- AWS: there is no immediate impact on the pricing side as the AWS CPI limits the scope of protocols to TCP only.
- Azure: Azure pricing is indeed based on load-balancing rules, but this cloud already supports mixed protocols via annotations. There is another risk for Azure, though: if the current restriction is removed from the K8s API, the Azure CPI must be prepared to handle Services with mixed protocols.
- GCE: here the risk is valid once the user exceeds the threshold of 5 forwarding rules. Though, as we mentioned above, it is already possible now without this feature
- IBM Cloud: no risk
- OpenStack: here the risk is that there is almost no chance to analyze all the public OpenStack cloud providers with regard to their pricing policies
- Oracle: no risk
- Tencent Cloud: no risk

The other risk is in the introduction of this feature without an option control mechanism, i.e. as a general change in Service handling. In that case there is the question whether this feature should be a part of the conformance test set, because it can affect the conformance of cloud providers.

A possible mitigation is to put the feature behind option control. 

## Design Details

The implementation of the basic logic is ready in this PR:
https://github.com/kubernetes/kubernetes/pull/75831

Currently a feature gate is used to control its activation status. Though if we want to keep this feature behind option control even after it reaches its GA state we should come up with a different solution, as feature gates are used to control the status of a feature as that graduates from alpha to GA, and they are not meant for option control for features with GA status.

### Option Control Alternatives

#### Annotiation in the Service definition

In this alternative we would have a new annotation in the `kubernetes.io` annotation namespace, as it was planned in the original PR. Example: `kubernetes.io/mixedprotocol`. If this annotation is applied on a Service definition the Service would be accepted by the K8s API.

Pro: 
- a kind of "implied conduct" from the user's side. The user explicitly defines with the usage of the annotation that the usage of multiple protocols on the same LoadBalancer is accepted
- Immediate feedback from the K8s API if the user configures a Service with mixed protocol set but without this annotation
Con:
- Additional configuration task for the user
- Must be executed for each and every Service definitions that are to define different protocols for the same LB

#### Merging Services in CPI

This one is not really a classic option control mechanism. The idea comes from the current practice implemented in MetalLB: https://metallb.universe.tf/usage/#ip-address-sharing
The same works in GCE as a workaround.

I.e. if a cloud provider wants to support this feature the CPI must have a logic to apply the Service definitions with a common key value (for example loadBalancerIP) on the same LoadBalancer instance. If the CPI does not implement this support it will work as it does currently.
Pro:
- the cloud provider can decide when to support this feature, and until that it works as currently for this kind of config
Con:
- the users must maintain two Service instances
- Atomic update can be a problem - the key must be such a Service attribute that cannot be patched

### Test Plan

### Graduation Criteria

#### Examples

##### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

##### Beta -> GA Graduation

- N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

##### Removing a deprecated flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality which deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include [conformance tests].**

[conformance tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md

### Upgrade / Downgrade Strategy

If applicable, how will the component be upgraded and downgraded? Make sure this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to make use of the enhancement?

### Version Skew Strategy

If applicable, how will the component handle version skew with other components? What are the guarantees? Make sure
this is in the test plan.

Consider the following in developing a version skew strategy for this enhancement:
- Does this enhancement involve coordinating behavior in the control plane and in the kubelet? How does an n-2 kubelet without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI, CRI or CNI may require updating that component before the kubelet.

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded

## Drawbacks [optional]

## Alternatives [optional]

## Infrastructure Needed [optional]
