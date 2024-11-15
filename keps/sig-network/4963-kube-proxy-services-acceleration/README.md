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
  - [Default Value](#default-value)
    - [Default value evaluation](#default-value-evaluation)
  - [Mechanics](#mechanics)
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
    - [Alternative 1: Adding a new field to the Service API](#alternative-1-adding-a-new-field-to-the-service-api)
    - [Alternative 2: Adding a new configuration option to select network interfaces on nodes](#alternative-2-adding-a-new-configuration-option-to-select-network-interfaces-on-nodes)
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

Kube-proxy manages Service traffic by manipulating iptables/nftables rules. This approach can introduce performance overhead, particularly for services with high throughput or a large number of connections. The kernel's flowtables offer a more efficient alternative for handling established connections, bypassing the standard netfilter processing pipeline.

### Goals

- Provide an option for kube-proxy users to enable Service traffic acceleration.

### Non-Goals

- Separation of Concerns: Kube-proxy's primary responsibility is to manage Service traffic. Extending the flowtable offloading functionality to non-Service traffic will potentially introduce unintended side effects. It's better to keep the feature focused on its core purpose.

## Proposal

The kernel [Netfilter’s flowtable infrastructure](https://docs.kernel.org/networking/nf_flowtable.html) allows to define a fastpath through the flowtable datapath. This infrastructure also provides hardware offload support.

```
                                       userspace process
                                        ^              |
                                        |              |
                                   _____|____     ____\/___
                                  /          \   /         \
                                  |   input   |  |  output  |
                                  \__________/   \_________/
                                       ^               |
                                       |               |
    _________      __________      ---------     _____\/_____
   /         \    /          \     |Routing |   /            \
-->  ingress  ---> prerouting ---> |decision|   | postrouting |--> neigh_xmit
   \_________/    \__________/     ----------   \____________/          ^
     |      ^                          |               ^                |
 flowtable  |                     ____\/___            |                |
     |      |                    /         \           |                |
  __\/___   |                    | forward |------------                |
  |-----|   |                    \_________/                            |
  |-----|   |                 'flow offload' rule                       |
  |-----|   |                   adds entry to                           |
  |_____|   |                     flowtable                             |
     |      |                                                           |
    / \     |                                                           |
   /hit\_no_|                                                           |
   \ ? /                                                                |
    \ /                                                                 |
     |__yes_________________fastpath bypass ____________________________|

             Fig.1 Netfilter hooks and flowtable interactions
```

Enabling the flowtable fastpath requires to use nftables. It only needs to create a flowtable and add the corresponding network interfaces whose traffic will be offloaded.
The traffic to be offloaded will be selected by a flowtable rule in the forward chain.

Example configuration:

```
table inet x {
        flowtable f {
                hook ingress priority 0; devices = { eth0, eth1 };
        }
        chain y {
                type filter hook forward priority 0; policy accept;
                ip protocol tcp flow add @f
                counter packets 0 bytes 0
        }
}
```

We propose introducing a new kube-proxy feature that allows users to use the flowtable fastpath for Service traffic:

- **Sane Defaults:** Kubernetes should provide the best possible performance out of the box and simplify the user experience, this feature will be enabled by default with a sensible threshold, so most users will immediately experience the performance benefits without any manual configuration.
- **Flexibility and Safeguards:** Users will have the ability to adjust the behavior or completely disable it. This provides flexibility and "escape hatches" for users who may encounter compatibility issues, require fine-grained control over their network traffic, or need to optimize for specific workload characteristics on a node.

### User Stories

#### Story 1

As a Kubernetes user, I want Kubernetes to automatically optimize the network performance of my applications and minimize resource consumption by default, without requiring manual configuration.

#### Story 2

As a cluster administrator managing a cluster where services typically handle small, short-lived connections, I want to be able to easily configure or disable the flowtable offloading feature to prevent potential performance degradation and maintain control over my network environment.

### Risks and Mitigations

Once the network traffic moves to the fastpath it completely bypass the kernel stack, so
any other network applications that depend on the packets going through the network stack (monitoring per example) will not be able to see the connection details. The feature will only apply the fastpath based on a defined threshold, that will also allow to disable the feature.

Flowtables netfilter infrastructure is not well documented and we need to validate assumptions to avoid unsupported or suboptimal configurations. Establishing good relations and involve netfilter maintainers in the design will mitigate these possible problems.

## Design Details

This feature will only work with kube-proxy nftables mode. We will add a new configuration option to kube-proxy to enable Service traffic offload based on a number of packets threshold per connection.

The packet threshold approach offers several advantages over the [alternatives](#alternatives):

- It directly targets the need to apply offloading on large connections focusing on the size of connections (number of packets) while avoiding potential performance penalties for small connections.
- It's a straightforward and easily understandable metric for users.
- It allows for fine-grained control over which connections are offloaded.

A configurable parameter (--offload-packet-threshold) will determine the minimum number of packets a connection must exchange before being considered for offloading.

### Default Value

The default value for the --offload-packet-threshold will be carefully chosen based to ensure optimal performance for a wide range of applications. The traffic that will get more benefit from this feature will be the so called [elephant flows](https://en.wikipedia.org/wiki/Elephant_flow), so we'll obtain our default value based on that.

The [elephant flow detection is a complex topic with a considerable number of literature about it](https://scholar.google.pt/scholar?q=elephant+flow+detection&hl=es&as_sdt=0&as_vis=1&oi=scholart). For our use case we proposa a more simplistic approach based on the number of packets, so it can give us good trade off between performance improvement and safety, we want to avoid complex heuristics and have a more predictible and easy to think about behavior based on the lessones learned from [KEP-2433 Topology aware hints](https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/2433-topology-aware-hints#proportional-cpu-heuristic).

We chose 20 as the number of packets as the threshold to offload, based on existing thresold used by Cisco systems in [Cisco Application Centric Infrastructure](https://scholar.google.com/scholar_lookup?hl=en&publication_year=2018&author=G.+Tam&title=Cisco+Application+Centric+Infrastructure), Cisco referred to an elephant flow if the flow contains more than 15 packet sizes, i.e., short flow is less than 15 packets, we add 5 packets of buffer to be on the safe side. This means that, using TCP as an example, and assuming an MTU of 1500 bytes and removing the overhead of the TCP headers (that can vary from 20-60 bytes, use 40 for this example), offloading will benefit workloads that transmit more than: TCP Payload * Number of packets = 1460 bytes/packet * 20 = 29200 bytes.

#### Default value evaluation

We can use `netperf` to evaluate the impact and benefits of this feature for large streams of data. The scenario will be one client Pod running `netperf` and a second Pod running `netserver` behind a Service. It is important to mention that `netperf` requires two connections, one for the control plane and other for the data plane so the Service MUST expose two different ports for each of them. An example of this setup can be done with:

```sh
$ kubectl run client --image=cilium/netperf
$ kubectl run server --image=cilium/netperf
```

And creating a Service with the following manifest:

```yaml
apiVersion: v1
kind: Service
metadata:
  labels:
    run: server
  name: server
spec:
  ports:
  - name: netperf-udp
    port: 5201
    protocol: UDP
    targetPort: 5201
  - name: netperf-tcp
    port: 5202
    protocol: TCP
    targetPort: 5202
  - name: netperf-ctl
    port: 12865
    protocol: TCP
    targetPort: 12865
  selector:
    run: server
  type: ClusterIP
```

Connecting to the `client` Pod we can run test the dataplane performance:

- Service traffic without flowtables:

```sh
$ netperf -H 10.244.0.10 -fm -tTCP_STREAM -i10
MIGRATED TCP STREAM TEST from 0.0.0.0 (0.0.0.0) port 0 AF_INET to 10.244.0.10 (10.244.) port 0 AF_INET : +/-2.500% @ 99% conf.
Recv   Send    Send
Socket Socket  Message  Elapsed
Size   Size    Size     Time     Throughput
bytes  bytes   bytes    secs.    10^6bits/sec

131072  16384  16384    10.00    13046.26
```

- Service traffic with flowtables without threshold:

```sh
$ netperf -H 10.96.142.98 -fm -tTCP_STREAM -i10 -- -P 10001,5201
MIGRATED TCP STREAM TEST from 0.0.0.0 (0.0.0.0) port 10001 AF_INET to 10.96.142.98 (10.96.1) port 5201 AF_INET : +/-2.500% @ 99% conf.
Recv   Send    Send
Socket Socket  Message  Elapsed
Size   Size    Size     Time     Throughput
bytes  bytes   bytes    secs.    10^6bits/sec

131072  16384  16384    10.00    16572.83
```

- Service traffic with flowtables with default threshold 20:

```sh
$ netperf -H 10.96.142.98 -fm -tTCP_STREAM -l10 -- -P 10001,5202 -R 1
MIGRATED TCP STREAM TEST from 0.0.0.0 (0.0.0.0) port 10001 AF_INET to 10.96.142.98 (10.96.1) port 5202 AF_INET
Recv   Send    Send
Socket Socket  Message  Elapsed
Size   Size    Size     Time     Throughput
bytes  bytes   bytes    secs.    10^6bits/sec

131072  16384  16384    10.00    16292.78

```

The performance impact is still significant, but avoids the penalty on short lived connections, that are impacted in the latency as we can see with the [netperf CRR test](https://hewlettpackard.github.io/netperf/doc/netperf.html#TCP_005fCRR) that emulate short lived HTTP connections:


- Service traffic without flowtables:

```sh
$ netperf -H 10.244.0.10 -fm -tTCP_CRR -i10 -- -o min_latency,mean_latency,max_latency,stddev_latency,transaction_rate
MIGRATED TCP Connect/Request/Response TEST from 0.0.0.0 (0.0.0.0) port 0 AF_INET to 10.244.0.10 (10.244.) port 0 AF_INET : +/-2.500% @ 99% conf.
Minimum Latency Microseconds,Mean Latency Microseconds,Maximum Latency Microseconds,Stddev Latency Microseconds,Transaction Rate Tran/s
41,56.63,848,11.14,17730.719
```

- Service traffic with flowtables no threshold:

```sh
netperf -H 10.96.142.98 -fm -tTCP_CRR -i10 -- -P 10001,5201 -o min_latency,mean_latency,max_latency,stddev_latency,transaction_rate
MIGRATED TCP Connect/Request/Response TEST from 0.0.0.0 (0.0.0.0) port 10001 AF_INET to 10.96.142.98 (10.96.1) port 5201 AF_INET : +/-2.500% @ 99% conf.
Minimum Latency Microseconds,Mean Latency Microseconds,Maximum Latency Microseconds,Stddev Latency Microseconds,Transaction Rate Tran/s
40,57.64,2244,11.85,17388.759
```

Since we have a default threshold of 20 packet, the offload will not impact negatively short lived connections as we can see in the previous results.

### Mechanics

Users will have the ability to adjust the --offload-packet-threshold value or completely disable flowtable offloading if desired. This provides flexibility and "escape hatches" for users who may encounter compatibility issues, require fine-grained control over their network traffic, or need to optimize for specific workload characteristics on a node.

Kube-proxy will create a `flowtable` in the kube-proxy table with the name `kube-proxy-flowtable` and will monitor the network interfaces in the node to populate the `flowtable` with the interfaces on the Node.

Kube-proxy will insert a rule to offload all Services established traffic in the `filter-forward` chain:

```go
	// Offload the connection after the defined number of packets
	if proxier.fastpathPacketThreshold > 0 {
		tx.Add(&knftables.Flowtable{
			Name: serviceFlowTable,
		})
		tx.Add(&knftables.Rule{
			Chain: filterForwardChain,
			Rule: knftables.Concat(
				"ct original", ipX, "daddr", "@", clusterIPsSet,
				"ct packets >", proxier.fastpathPacketThreshold,
				"flow offload", "@", serviceFlowTable,
			),
		})
	}
```

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

To be added

- `<package>`: `<date>` - `<test coverage>`

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

- eBPF was considered, but this was discarded because of the complexity, the increase on node resources consumption and the lack of support for old kernels.

### API Design 

#### Alternative 1: Adding a new field to the Service API

- This approach would involve extending the Service API to include a field that indicates whether flowtable offloading should be enabled for that Service.
- Why it was discarded:
  - A Service-level setting lacks the granularity to differentiate between large and small connections within that Service. The goal is to offload large connections by default, and a Service-level setting can't achieve this.
  - The Service abstraction represents a logical grouping of pods, not a specific traffic pattern. Using it to control a low-level network optimization like flowtable offloading creates an abstraction mismatch.
  - Adding a new field to the Service API increases complexity for users and implementers.

#### Alternative 2: Adding a new configuration option to select network interfaces on nodes

- This would allow users to specify which network interfaces on the nodes should have flowtable offloading enabled.
- Why it was discarded:
  - Mapping network interfaces to specific connections or traffic types is complex and can be confusing for users. Interface names are often tied to pods, and Services can route traffic to multiple pods dynamically.
  - This approach doesn't directly address the core problem of identifying large connections. It relies on an indirect mapping between interfaces and traffic patterns, which can be unreliable and difficult to manage.
  - This option wouldn't be applicable in scenarios where multiple services share the same network interface.
