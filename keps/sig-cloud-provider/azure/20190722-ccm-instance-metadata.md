---
title: Support Instance Metadata Service with Cloud Controller Manager
authors:
  - "@feiskyer"
owning-sig: sig-cloud-provider
participating-sigs:
  - sig-node
reviewers:
  - "@andrewsykim"
  - "@jagosan"
approvers:
  - "@andrewsykim"
editor: "@feiskyer"
creation-date: 2019-07-22
last-updated: 2019-12-15
status: implementable
see-also:
  - "/keps/sig-cloud-provider/20180530-cloud-controller-manager.md"
---

# Support Instance Metadata Service with Cloud Controller Manager

## Table of Contents

<!-- TOC -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Alternatives](#alternatives)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Examples](#examples)
      - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
      - [Beta -&gt; GA Graduation](#beta---ga-graduation)
      - [Removing a deprecated flag](#removing-a-deprecated-flag)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
<!-- /TOC -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

With [cloud-controller-manager](https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/)(CCM), Kubelet won’t initialize itself from Instance Metadata Service (IMDS). Instead,  CCM would get the node information from cloud APIs. This would introduce more cloud APIs invoking and more possibilities to get throttled, especially for large clusters.

This proposal aims to add instance metadata service (IMDS) support with CCM. So that, all the nodes could still initialize themselves and reconcile the IP addresses from IMDS.

## Motivation

Before CCM, kubelet supports getting Node information by the cloud provider's instance metadata service. This includes:

- NodeName
- ProviderID
- NodeAddresses
- InstanceType
- AvailabilityZone

Instance metadata service could help to reduce API throttling issues and the node's initialization duration. This is especially helpful for large clusters. But with CCM, this is not possible anymore because the above functionality has been moved to CCM.

Take Azure cloud provider for example:

- According to Azure documentation [here](https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-manager-request-limits), for each Azure subscription and tenant, Resource Manager allows up to **12,000 read requests per hour** and 1,200 write requests per hour. That means, on average only 200 read requests could be sent per minute.
- For different Azure APIs, there’re also additional rate limits based on different durations. For example, there are 3Min and 30Min read limits for VMSS APIs (the numbers below are only for reference since they’re not officially [documented](https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-manager-request-limits)):
  - Microsoft.Compute/HighCostGetVMScaleSet3Min;200
  - Microsoft.Compute/HighCostGetVMScaleSet30Min;1000
  - Microsoft.Compute/VMScaleSetVMViews3Min;5000

Based on those rate limits, getting node’s information for a 5000 cluster may need hours.  Things would be much worse for multiple clusters in the same tenant and subscription.

So the proposal aims to add IMDS support back with CCM, so that the kubernetes cluster could still be scaled to a large number of nodes.

### Goals

- Allow nodes to be initialized from IMDS.
- Allow nodes to reconcile the node addresses from IMDS.

### Non-Goals

- Authentication and authorization for each provider implementations.
- API throttling [issue](https://github.com/kubernetes/kubernetes/issues/60646) on route controller.

## Proposal

Same as kube-controller-manager, the [cloud provider interfaces](https://github.com/kubernetes/cloud-provider/blob/master/cloud.go#L43) could be split into two parts:

- Instance-level interfaces: Instances and Zones
- Control-plane interfaces, e.g. LoadBalancer and Routes

The control-plane interfaces are still kept in CCM (who’s name is `cloud-controller-manager`), and deployed on masters. For instance-level interfaces, a new daemonsets would be introduced and implement instance-level interfaces (who’s name is `cloud-node-manager`).

With these changes, the whole node initialization workflow would be:

- Kubelet specifying `--cloud-provider=external` will add a taint `node.cloudprovider.kubernetes.io/uninitialized` with an effect NoSchedule during initialization.
- `cloud-node-manager` would initialize the node again with `Instances` and `Zones`.
- `cloud-controller-manager` then take care of the rest things, e.g. configure the Routes and LoadBalancer for the node.

After node initialized, cloud-node-manager would reconcile the IP addresses from IMDS periodically.

Considering some providers may not require IMDS, cloud-node-manager cloud be enabled optionally by a new option `--enable-node-controller` on cloud-controller-manager. With this new option, there would be three node initialization modes after this proposal:

- 1) Centrally via cloud-controller-manager. All the node initialization, node IP address reconciling and other cloud provider operations are done in CCM.
  - `cloud-controller-manager --enable-node-controller=true`
- 2) Using IMDS with cloud-node-manager.
  - cloud-node-manager running as a daemonset on each node
  - `cloud-controller-manager enable-node-controller=false`
- 3) Arbitrary via custom controllers. Customers may also choose their own controllers, which implement the same functions in cloud provider interfaces. The design and deployments are out of this proposal's scope.

## Alternatives

Since there are already a lot of plugins in Kubelet, e.g. CNI, CRI, and CSI. An alternative way is introducing another cloud-provider plugin, e.g. Cloud Provider Interface (CPI).

When Kubelet starts, the cloud provider plugin may register itself into Kubelet, and then Kubelet invokes cloud provider plugin to initialize the node.

One problem is the deployment of those plugins. If daemonsets is used to deploy those cloud provider plugins, then they should be schedulable before kubelet fully initialized the nodes. That means Kubelet may need to initialize itself two times:

- Register the node into Kubernetes without any cloud-specific information.
- Wait for cloud provider plugins registered and then invoke the plugin to add the cloud-specific information.

The problem of this way is cloud provider plugin would block node’s initialization, while the plugin itself could be scheduled to that node. Although taint _node.cloudprovider.kubernetes.io/uninitialized_ with an effect NoSchedule could still be applied to solve this issue, separating it to cloud-node-manager would make the whole architecture more clear.

## Design Details

### Test Plan

**Note:** *Section not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:

- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy.
Anything that would count as tricky in the implementation and anything particularly challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage expectations).
Please adhere to the [Kubernetes testing guidelines][testing-guidelines] when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md

### Graduation Criteria

**Note:** *Section not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. Initial KEP should keep
this high-level with a focus on what signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:

- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning),
or by redefining what graduation means.

In general, we try to use the same stages (alpha, beta, GA), regardless how the functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

#### Examples

These are generalized examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

##### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

##### Beta -> GA Graduation

- N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least 2 releases between beta and GA/stable, since there's no opportunity for user feedback, or even bug reports, in back-to-back releases.

##### Removing a deprecated flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality which deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include [conformance tests].**

[conformance tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/conformance-tests.md

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
