# KEP-2610: AllPorts Services

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Supported service types](#supported-service-types)
  - [Usage](#usage)
  - [Service Transitions](#service-transitions)
  - [Life of a request](#life-of-a-request)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
    - [Open Questions to be resolved:](#open-questions-to-be-resolved)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Today, a Kubernetes Service accepts a list of ports to be exposed by it.
It is possible to specify any number of ports (as long as the service is within the max object size limit), by listing them in the service spec.
This can be tedious if the service needs a large number of ports.
This KEP proposes to add a new field to the Service spec to allow exposing the entire port range(1 to 65535).

## Motivation

There are several applications like SIP/RTP/Gaming servers that need a lot(1000+) of ports to run multiple calls or media streams.
Currently the only option is to specify every port in the Service Spec. A request for port ranges in Services has been open in https://github.com/kubernetes/kubernetes/issues/23864. Implementing port ranges are challenging since iptables/ipvs do not support remapping port ranges. Also, in order to specify several non-contiguous port ranges, the user will have to expose the entire valid port range. Hence, this proposal to set a single field in order to expose the entire port range and implement the service clients and endpoints accordingly.
[A survey](https://docs.google.com/forms/d/1FOOG2ZoQsnJLYAjnhEtSPYmUULWFNe88iXR7gtFcP7g/edit) was sent out to collect the use-cases for AllPorts support - [results.](http://tiny.cc/allportsslides)

### Goals

* Allow users to optionally expose the entire port range via a Service (of Type LoadBalancer or ClusterIP).

### Non-Goals

* Supporting Port Ranges in a Service.
* Changing the default behavior of Service ports.

## Proposal

The proposal here is to introduce an `allPorts` boolean(*bool) field to the [service API.](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.21/#servicespec-v1-core)
A service that sets this field to true will be reachable on any valid port [1 to 65535]. The backend pods have to be configured accordingly. The backend pods will receive requests with the same port number - in other words, port remapping will not be possible.

The [APIServer validation](https://github.com/kubernetes/kubernetes/blob/b9ce4ac212d150212485fa29d62a2fbd783a57b0/pkg/apis/core/validation/validation.go#L4162) that disallows empty `ports` will be relaxed, when `allPorts` is set.

The value of `allPorts` field can be toggled on supported service types.

### Supported service types

Setting this field will be supported for:

* ServiceType=ClusterIP
* ServiceType=LoadBalancer

This field is not applicable to ExternalName services.

NodePort services are not supported.
A NodePort service accepts traffic on a given port on the Node, redirecting it to the specified targetPort of the service endpoints. Support AllPorts for a NodePort service means -
traffic to any port on the node, will be forwarded to its endpoints on the same port. This could potentially break networking on the node, if traffic for, say, port 22 got forwarded to \<endpoint IP\>:22.

Headless services are not supported either. Headless services do not have a ClusterIP, nor can they be LoadBalancer services. SRV records for endpoints with empty port array will not be created. Hence, supporting AllPorts on Headless services has little value.

Setting `allPorts` to true is not supported on services that specify [ExternalIPs in the spec.](https://kubernetes.io/docs/reference/kubernetes-api/service-resources/service-v1/#ServiceSpec)

### Usage

In order to expose the entire port range on a supported service, a user needs to:
1) Create a LoadBalancer/ClusterIP Service and set `allPorts` to True.
2) Leave the `ports` field empty. Populating the ports array is invalid when setting `allPorts` to True.

There will be no NodePort allocation for the service in this case. The only exception is the NodePort for HealthChecks for LoadBalancers using `ExternalTrafficPolicy: Local`.


### Service Transitions

Consider the following transitions:

* Changing from ClusterIP to LoadBalancer type, or vice versa
  Preserving `allPorts` value as well as toggling is supported.

* Setting an ExternalIP on a ClusterIP that has `allPorts` true
  `allPorts` value has to be unset.

* Changing from ClusterIP/LoadBalancer to NodePort type
  `allPorts` value has to be unset.
 
* Changing from ClusterIP/LoadBalancer to NodePort type
  Once the service has been changed to ClusterIP/LoadBalancer type, `allPorts` field can be set.
 
 
Transitioning from non-headless to headless and vice versa are not permitted by the API today.


### Life of a request

1) A service with ClusterIP a.b.c.d is configured with `allPorts` set to true. This service has 2 endpoints - pod1, pod2.

2) A client(1.2.3.4) sends a request to a.b.c.d:8888 - it is received on a cluster node where:

  a) firewall rules allow it

  b) kube-proxy iptables rules DNAT it to one of the backend pod IPs - p.q.r.s (pod1)

  c) request is received on pod1 with source IP - 1.2.3.4 and destination p.q.r.s:8888

  d) pod1 responds directly to 1.2.3.4.

The path taken by the request is similar in case of a LoadBalancer service. If the LoadBalancer implementation uses a proxy(instead of Direct Server Return), the proxy should be able to receive requests on all ports as well.


### User Stories

* A user wants to expose ports 20,000 to 50,000 for a web-conferencing application that is exposed as a LoadBalancer service.

The user can now create a LoadBalancer service with `allPorts` set to true. This will enable clients to connect on <LB VIP>: <port>, where port is any value between 20,000 and 50,000.


### Risks and Mitigations

* Allowing empty `ports` array in the Service object could break clients that watch Services and expect a valid port value.
These include kube-proxy, kube-dns/CoreDNS, other controllers. To mitigate this risk, the API validation change that allows empty `ports`,
will be feature-gated and soaked in Alpha stage for 2-3 releases. That way, when this feature is Beta,
the supported node-version(upto 2 releases behind) will have the kube-proxy changes that handle empty ports.

* A user could accidentally expose the entire port range on their cluster nodes, by enabling `allPorts` for a Service.
To avoid this, users should make sure that their firewall implementation only permits traffic to LoadBalancerIP:\<servicePort\> and not NodeIP:\<servicePort\>.
This is mostly applicable to LoadBalancer services, that are typically
accessible from outside the cluster.
Currently, kube-proxy adds the right rules to only allow traffic to the ServiceIP/Port combination specified in the service.
When using AllPorts, kube-proxy will allow all traffic for the given serviceIP/LoadBalancerIP. Kube-Proxy rules alone cannot drop traffic to NodeIP:\<servicePort\>.

* LoadBalancing at IP-level could have regression in behavior.
For example, IPVS supports loadbalancing without specifying ports, for TCP and UDP services.
However, traffic to the same 3-tuple(dest IP, dest Port, protocol) will be sent to the same backend.
In other words, this is similar to setting `sessionAffinity: ClientIP`, but it will be the default(and only) behavior with AllPorts + IPVS.
In contrast, when service ports are specified, backend pods are selected at random, unless `sessionAffinity: ClientIP` is specified.
This could be mitigated by using iptables to implement the AllPorts logic.

* A known issue with [host services being accessible via ClusterIP in ipvs mode](https://github.com/kubernetes/kubernetes/issues/72236) could be mitigated by AllPorts support.
If a ClusterIP service is created using `allPorts` set to true and sshd on the host listens on 0.0.00:22, traffic to `<ClusterIP>:22` will only go to backend pods.
If the service were not using `allPorts`, and did not specify port 22 in the Port list, sshd would be exposed by connecting to `<ClusterIP>:22`.
This is because the clusterIP is assigned to an ipvs interface in the host namespace.

## Design Details

Changes are required to APIServer validation, kube-proxy and controllers that use the ServicePort field.

1) New Validation checks:

   * `allPorts` can be set to True only for ClusterIP(non-headless) and LoadBalancer services.
   * The `ports` array should be empty when `allPorts` is set to True.

2) Kube-Proxy should configure iptables/ipvs rules by skipping port/protocol filter, if `allPorts` is true.

3) LoadBalancer controllers should create LoadBalancer resources with the appropriate port values.

4) Endpoints and EndpointSlices controller should create Endpoints with empty port values.

5) DNS Providers should handle empty ports in Services and Endpoints.
   There will be no DNS SRV Records for Services with `allPorts` set.
   coreDNS handles empty ports in [services](https://github.com/coredns/coredns/blob/09b63df9c1584bb5389d1b681698631bcd7c19e1/plugin/kubernetes/kubernetes.go#L577) and [endpoints.](https://github.com/coredns/coredns/blob/09b63df9c1584bb5389d1b681698631bcd7c19e1/plugin/kubernetes/kubernetes.go#L559)
   kube-dns also handles empty ports in [services](https://github.com/kubernetes/dns/blob/077a43e83e648ba5f04bae18ffcb824edc9db967/pkg/dns/dns.go#L506) and [endpoints.](https://github.com/kubernetes/dns/blob/077a43e83e648ba5f04bae18ffcb824edc9db967/pkg/dns/dns.go#L541)
   There is a warning in kube-dns for [empty port services](https://github.com/kubernetes/dns/blob/077a43e83e648ba5f04bae18ffcb824edc9db967/pkg/dns/dns.go#L320) that could be removed after GA of AllPorts.

6) There will be no environment variable of the form "_SERVICE_PORT" for services with `allPorts` set. This [codepath](https://github.com/kubernetes/kubernetes/blob/e1f971d5c2a1002c4e90471d064f87f297740aba/pkg/kubelet/envvars/envvars.go#L48) currently assumes non-zero ports and that will be updated.

#### Open Questions to be resolved:

1) How should IPVS implementation be handled?
Options are to a) use iptables for DNAT b) use ipvs rules that results in same 5-tuple requests being sent to the same backend pod. c) create allPorts services as "fwmark-service" and assign a unique mark for each of them.

2) Identfy how Service Mesh(Istio)/Calico/MetalLB can support AllPorts.

### Test Plan

Unit tests:

* To verify API validation of the `allPorts` and `ports` fields.
* To verify that all users(kube-proxy, kubelet, various controllers) of Service/Endpoints can handle empty ports.
* To check the default value of `allPorts` on each type of Service.

E2E tests:

* To verify that default behavior of `allPorts` does not break any existing e2e tests.
* Test setting `allPorts` on a new Service and connecting to the service VIP on a few different ports.
* Test setting `allPorts` on an existing Service and connecting to the service VIP on a few different ports.
* Test unsetting `allPorts` on a service and specifying a single port allows traffic only for that port.
* Test setting `allPorts` explicitly to false.

### Graduation Criteria

#### Alpha

- Add a new field `allPorts` to Service, but it can only be set when the feature gate is on.


#### Alpha -> Beta Graduation

- Ensure that the main clients of Service/Endpoints API - kube-proxy, kubelet, loadbalancer controllers have added support to handle empty ports. All supported node versions for the given master version(that graduates `allPorts` to Beta) should handle this case correctly.
- Tests are in Testgrid and linked in KEP
- Demonstrated community adoption of this feature.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?
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
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
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

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

<!--
At a high level, this usually will be in the form of "high percentile of SLI
per day <= X". It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code
-->

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

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

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

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

* Create a new service type - IPOnly which does not accept any port entries.
  Traffic redirection to endpoints from service VIP will be IP-based, without a port or protocol filter.

  This requires modifying existing LoadBalancer controller to also provision resources for this type of Service.
  Adding a new service type is quite a bit of API overhead. This would be a better fit in the [Gateway API.](https://gateway-api.sigs.k8s.io/)

*  Restrict AllPorts support to LoadBalancer services only. Allow LoadBalancer services to be headless [by relaxing the validation the check.](https://github.com/kubernetes/kubernetes/blob/036cab71a6faefa84b10a199a61bcdc38e3572c3/pkg/apis/core/validation/validation.go#L4177)
   Default behavior is to allow traffic for all Protocols. A list of allowed protocols can be specified in a Protocols list(new field) in the ServiceSpec.

   This approach breaks the assumption that all LoadBalancer services are valid ClusterIP services. It also does not provide AllPorts support for ClusterIP services which was desirable based on the [survey results.](https://docs.google.com/presentation/d/1FO9H55-gnDh2RIqOZMDoP4OPVbaIaPhKl9C5to4uNNE/edit#slide=id.gdf6ff10943_0_30)

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
