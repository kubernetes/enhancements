# KEP-4963: Kube-proxy Services Acceleration

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
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

This KEP proposes utilizing the flowtable infrastructure within the Linux kernel's netfilter module to create an accelerated path for Kubernetes Service traffic using kube-proxy.

## Motivation

Services traffic acceleration can significantly reduce latency and increase throughput improving application response times and user experience.
It will also reduce the CPU and memory overhead associated with Service traffic processing, improving efficiency and reducing cost.

### Goals

- Provide an option for kube-proxy users to enable Service traffic acceleration.

### Non-Goals

- Provide acceleration in any traffic that is not related to kube-proxy, per example, Pod to Pod traffic.

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

Enabling the flowtable fastpath requires to use nftables and only needs to create a flowtable on the corresponding network interface and add one rule in the forward chain.

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

Kube-proxy will allow users to opt-in for Service acceleration, since it is not possible to guarantee that a feature that completely bypass the kernel has unintended consequences in some environments or incompatibility with other networking components.

The opt-in behavior will be based on network interface names and attributes and will use CEL expressions, similar to how we use `--pod-interface-name-prefix` in [KEP-2450](https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/2450-Remove-knowledge-of-pod-cluster-CIDR-from-iptables-rules). The powerful matching expression allow users to deal with more complex environments: nodes with multiple interfaces and only some of them with hardware offload, matches on alias to select only some specific kind of pods, ...

Some example expressions are:

- `interface.name.startsWith(\"eth\")` selects all interfaces with prefix `eth`
- `interface.type == \"veth\"` selects all interfaces with type `veth`
- `interface.alias == \"offload\"` selects all interfaces with alias set to `offload`

This expressions can also be concatenated, see https://github.com/google/cel-go for more information.

Kernel bypassing will only be applied once the connection is ESTABLISHED, the rationale behind this is that, once the connection is accelerated, it bypass the network stack, by waiting for the connection to be established we ensure the integration with other network applications (network policies, service meshes, ... ) in the node and we avoid possible performance problems or possible DOS attacks by trying to accelerate every single connection on a node.

### User Stories (Optional)

#### Story 1

I want to use Service traffic acceleration with kube-proxy to improve the performance of my applications.

### Risks and Mitigations

Once the network traffic moves to the datapath it completely bypass the kernel stack, so
any other network applications that depend on the packets going through the network stack (monitoring per example) we'll not be able to see the connection data. The feature will only
apply the fast path on established connections, since most of the network applications are statefuls, usually is safe to think that once a connection is established no additional operations are required on it.

## Design Details

This feature will only work with kube-proxy nftables mode.

Users will be able to opt-in to Service traffic acceleration by passing a CEL expression using the flag `--accelerated-interface-expression` or the configuration option `AcceleratedInterfaceExpression` to match the network interfaces in the node that are subjet to Service traffic acceleration. The absence of a CEL expression disables the feature.

Kube-proxy will create a `flowtable` in the kube-proxy table with the name `kube-proxy-flowtable` and will monitor the network interfaces in the node to populate the `flowtable` with the interfaces that match the configured CEL expression.

Kube-proxy will insert a rule to offload all Services established traffic in the `filter-forward` chain:

```
	tx.Add(&knftables.Rule{
		Chain: filterForwardChain,
		Rule: knftables.Concat(
			"ct original", ipX, "daddr", "@", clusterIPsSet,
			"ct state established",
			"flow offload", "@", serviceFlowTable,
		),
	})
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

- eBPF was considered, but this was discarded because of the complexity, the increase on node resources consumption and the lack of support for old kernels.
