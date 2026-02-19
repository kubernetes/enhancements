# KEP-4963: Kube-proxy Services Acceleration

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
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Service API Extension](#service-api-extension)
  - [FastPath Controller](#fastpath-controller)
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
  - [Technology](#technology)
  - [API Design](#api-design)
    - [Alternative 1: Adding a new configuration option to select network interfaces on nodes](#alternative-1-adding-a-new-configuration-option-to-select-network-interfaces-on-nodes)
    - [Alternative 2: Offloading all the service connections with packets greater than configurable threshold](#alternative-2-offloading-all-the-service-connections-with-packets-greater-than-configurable-threshold)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
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

This KEP proposes utilizing the flowtable infrastructure within the Linux kernel's netfilter module to offload Kubernetes Service traffic using kube-proxy to the fastpath.

## Motivation

Every packet entering the Linux kernel is evaluated against all rules attached to the netfilter hooks, even for established connections. These rules may be added by CNIs,
system administrators, firewalls, or kube-proxy, and together they define how packets are filtered, routed, or rewritten. As a result, packets continue to traverse the
full netfilter processing path, which can add unnecessary overhead for long-lived or high-throughput connections.

A connection becomes established only after the initial packets successfully pass through all applicable rules without being dropped or rejected. Once established,
packets associated with a Kubernetes service can be offloaded to the kernel fast path using flowtables. This allows subsequent service packets to bypass the full
netfilter stack, accelerating kube-proxy traffic and reducing CPU usage.

### Goals

- Provide an option for kube-proxy users to enable traffic acceleration for TCP and UDP services.

### Non-Goals

- Separation of Concerns: Kube-proxy's primary responsibility is to manage Service traffic. Extending the flowtable offloading functionality to non-Service traffic will
- potentially introduce unintended side effects. It's better to keep the feature focused on its core purpose.
- Supporting service acceleration for SCTP services.

## Proposal

We propose adding a new controller to KubeProxy to manage service acceleration using flowtables. This controller will install rules specifically for flowtable offloading.
This approach allows us to utilize flowtables for all linux backends (iptables, ipvs and nftables) without modifying their core logic. 


Only long-lived connections will benefit from the fastpath optimization, short-lived connections may see overall performance degradation due to overhead of offloading flows
for a small number of packets. To ensure the optimization is applied selectively for flowtables offloading, we propose extending the Service API with a new enum, 
`TrafficHint`. The field indicates expected traffic semantics of a Service. Only connections with `TrafficHint: LongLived` will be considered for fastpath offload.

### User Stories

#### Story 1

As a Kubernetes user, I want Kubernetes to automatically optimize the network performance of my applications and minimize resource consumption by default, without requiring
much manual configuration.

#### Story 2

As a cluster administrator managing a cluster where services typically handle small, short-lived connections, I want to be able to easily configure or disable the flowtable
offloading feature for services to prevent potential performance degradation and maintain control over my network environment.

### Risks and Mitigations

Moving network traffic to the fastpath causes packets to bypass the standard netfilter hooks after the ingress hook. Flowtables operate at the ingress hook, and packets still
traverse taps, so tools like tcpdump and Wireshark will continue to observe traffic. However, any applications that rely on hooks or rules evaluated after the ingress hook
may not observe or process these packets as expected. To mitigate this, fastpath offload will be applied selectively based on a configurable threshold, and users will have 
the option to disable the feature entirely.

Flowtables netfilter infrastructure is not well documented and we need to validate assumptions to avoid unsupported or suboptimal configurations. Establishing good relations
and involve netfilter maintainers in the design will mitigate these possible problems.

## Design Details

### Service API Extension

Users can set the `TrafficHint` field to opt in to fastpath acceleration for a Service. In the future, this field can also be leveraged to enable other dataplane optimizations
such as GRO-based UDP packet aggregation, reduced conntrack timeouts, or traffic distribution towards local endpoints. These optimizations are workload-dependent and only
beneficial for certain traffic patterns. The `TrafficHint` field provides kube-proxy with contextual information about the expected traffic semantics, allowing it to program
the dataplane with suitable optimizations for the Service.

```go
// +enum
// TrafficHint indicates the expected traffic behavior of a Service.
// Service implementations may use this hint to apply dataplane optimizations, which can vary
// depending on the type of optimization and the technology used to program the dataplane.
type TrafficHint string

// These are the valid TrafficHint values for a Service.
const (
	// LongLived indicates that the Service is expected to handle long-lived flows.
	// Dataplane implementations may use this hint to enable fastpath optimizations
	// for long-lived connections, reducing per-packet CPU overhead and improving
	// traffic throughput.
	LongLived TrafficHint = "LongLived"
)

// ServiceSpec describes the attributes that a user creates on a service.
type ServiceSpec struct {

	// TrafficHint describe the expected service traffic semantics.
    TrafficHint *TrafficHint
    
    // ...
}
```

### FastPath Controller

The controller will run as an independent go routine, separate from existing proxier backends. The controller will create a dedicated table `kube-proxy-flowtable`, in the `inet`
family to manage fastpath offloading. The controller will create `fastpath` flowtable and will continuously reconcile all the opted services and node network interfaces for offloading.

Fastpath offloading is only supported in the `forward` hook. Since the forward hook cannot operate at a [priority urgent than DNAT](https://people.netfilter.org/pablo/nf-hooks.png),
the  controller must rely on conntrack to retrieve the original destination IP and port to selectively offload service traffic. Currently, nftables does not support matching on sets
with data type [`inet_service` using `ct original proto-dst` expression](https://github.com/kubernetes/kubernetes/issues/131765#issuecomment-2994343818). As a result, packets must
be identified for offload in the `prerouting` hook, and then offloaded to the flowtables in the `forward` hook.

The controller will create following three sets in `kube-proxy-flowtable` for tracking Service that have opted into fastpath:
1. `service-nodeports`  (Protocol and NodePort)
2. `service-ip4-ports`  (Protocol, ServiceIP and ServicePort)
3. `service-ip6-ports`  (Protocol, ServiceIP and ServicePort)

Two primary chains will be added in `kube-proxy-flowtable`:
1. `fastpath-mark`
   Attached to `prerouting` hook with a priory urgent than DNAT. This chains marks the packets belonging to the Services listed in the sets above.
2. `fastpath`
   Attached to `forward` hook with filter priority. This chain offloads the marked packets to `fastpath` flowtable and clears the mark afterward.

```nft
table inet kube-proxy-fastpath {
	set service-nodeport {
		type inet_proto . inet_service
		elements = { tcp . 31147 }
	}

	set service-ip4-ports {
		type inet_proto . ipv4_addr . inet_service
		elements = { tcp . 10.96.128.18 . 5001 }
	}

	set service-ip6-ports {
		type inet_proto . ipv6_addr . inet_service
		elements = { tcp . fd00:10:96::c284 . 5001 }
	}

	flowtable fastpath {
		hook ingress priority filter
		devices = { eth0, tunl0 }
	}

	chain fastpath-mark {
		type filter hook prerouting priority dstnat - 10; policy accept;
		meta l4proto . th dport @service-nodeport meta mark set meta mark | 0x00008000
		meta l4proto . ip daddr . th dport @service-ip4-ports meta mark set meta mark | 0x00008000
		meta l4proto . ip6 daddr . th dport @service-ip6-ports meta mark set meta mark | 0x00008000
	}

	chain fastpath {
		type filter hook forward priority filter; policy accept;
		meta mark 0x00008000 flow add @fastpath
		meta mark set meta mark & 0x00007fff
	}
}
```

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

There will addition of new tests and modification of existing ones in the following packages:
- `k8s.io/kubernetes/pkg/proxy`: `2025-09-07` - `89.8%`

##### Integration tests

Not needed

##### e2e tests

- Create one Service with one backend running an iperf Service
- Run an iperf client against the Service without acceleration
- Run an iperf client against the Service with acceleration.
- The service with acceleration has to show a significant statistically difference on the throughput results.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers

#### GA

- No bug reported
- Feedback from developers and users

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md


### Upgrade / Downgrade Strategy

kube-proxy reconcile the nftables rules so the rules will be reconciled during startup and added or removed depending on how kube-proxy is configured.

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: KubeProxyAcceleration
  - Components depending on the feature gate: kube-proxy
- [x] Other
  - Describe the mechanism: kube-proxy configuration option
  - Will enabling / disabling the feature require downtime of the control
    plane? No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? No

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, disable the feature gate or the configuration in kube-proxy and restart it.

###### What happens if we reenable the feature if it was previously rolled back?

Kube-proxy reconciles at startup, so no problems can happen during rolls backs.

###### Are there any tests for feature enablement/disablement?

This is an opt-in feature in kube-proxy behind a feature gate, manual test will be performed for validation.

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

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No, on the contrary it is expected CPU consumption to be decreased.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

This feature consumes entries on the network device flowtable, that are per device.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Is not impacted, kube-proxy will not be able to add new Services to the datapath, but existing traffic for Services will be unaffected.

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

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

### Technology

- eBPF was considered, but this was discarded because of the complexity, the increase on node resources consumption and the lack of support for 
old kernels.

### API Design 

#### Alternative 1: Adding a new configuration option to select network interfaces on nodes

- This would allow users to specify which network interfaces on the nodes should have flowtable offloading enabled.
- Why it was discarded:
    - Mapping network interfaces to specific connections or traffic types is complex and can be confusing for users. Interface names are often
tied to pods, and Services can route traffic to multiple pods dynamically.
    - This approach doesn't directly address the core problem of identifying large connections. It relies on an indirect mapping between interfaces
and traffic patterns, which can be unreliable and difficult to manage.
    - This option wouldn't be applicable in scenarios where multiple services share the same network interface.

#### Alternative 2: Offloading all the service connections with packets greater than configurable threshold

- This approach would involve adding a new field to KubeProxy configuration that determines when the connections should be offloaded. Only connections
with packets greater than threshold will be offloaded.
- Why it was discarded:
    - Defining an appropriate threshold is a challenge.
    - This provides only global control, doesn't allow fine-grained control to enable offloading for selective services.
