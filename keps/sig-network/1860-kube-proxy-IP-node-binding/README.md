# KEP-1860 Make Kubernetes aware of the LoadBalancer behaviour

## Table of Contents

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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


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

E2E tests:
- The default behavior for `ipMode` does not break any existing e2e tests
- The default `VIP` value is already tested

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

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

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
