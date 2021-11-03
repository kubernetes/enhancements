# KEP-3037: Client-go Alternative Services

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [List of available control-plane nodes](#list-of-available-control-plane-nodes)
  - [Client side HA](#client-side-ha)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
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

Kubernetes clusters architecture have a "hub-and-spoke" pattern and all the components are required to communicate
with the API servers. Kubernetes components use client-go for these communications.

Creating high available clusters requires to use multiple control-plane nodes, however, clients need to use `Services` abstractions or external load balancers for bootstrapping and/or HA.

This proposal outlines a new feature in client-go that would allow clients to not requires external load balancers or `Services` for bootstrapping and/or HA.

## Motivation

Currently, when deploying high available clusters, clients can only benefit of it by using external load balancers or mechanisms that balance the connection to the corresponding control-plane.
This also has the "chicken and egg" problem on some components like kube-proxy and kubenet and "in-cluster" clients, that require `Services` to load balance the connection.

Ref: https://github.com/kubernetes/kubernetes/issues/18174

### Goals

- Allow users to don't require external load balancers between cluster and control-plane for HA setups.
- Allow components to bootstrap without any dependency on external load balancers or `Services` implementations.
- Increase the resilience of the cluster.

### Non-Goals

- The goal is to offer the opt-in functionality, any component change to use it is out of scope.
- Provide a lot of nines of availability.
- Work outside of the cluster network.
- Provide a client-side load balancing solution to distribute the load between the available API servers.

## Proposal

The proposal is to extend client-go to allow to connect to multiple control-plane nodes, and for API servers to announce the list of available servers to the clients.

### User Stories (Optional)

#### Story 1

As a Kubernetes admin I want to deploy my cluster in HA without having to use external load balancers for the cluster communication.

#### Story 2

As a Kubernetes developer I'd like to be able to develop HA applications and controllers without requiring external components.

#### Story 3

As a Kubernetes architect I'd like the cluster components to be completely autonomous.

### Notes/Constraints/Caveats (Optional)

Network load balancing is complicated, there are many edge cases and there is no universal solution, the requirements also change and evolve with the time (see HTTP1.1 vs HTTP2).

Client-go is the library used by all the cluster components to communicate with the API servers, any change or regression there can have a great impact on the project.

The implementation of this feature should not modify any of current behaviors and be completely pluggable, so users can opt-in safely and the project can be sure there will not create any regression.

### Risks and Mitigations

The feature will be opt-in, users can disable it and keep having the same behavior.

To avoid possible security issues, this feature will require HTTP2 and HTTPS in order to work.

The alternative services IPs exposed are the same published in the Endpoints for the `kubernetes.default` services, the Alt-Svc headers will not be present on requests
that are not authenticated.

## Design Details

There are two problems we have to solve:

- get a list of available control-plane nodes
- client capability to be able to connect to multiple control-planes

### List of available control-plane nodes

This problem can be solved in two different ways:

1. Configuring on the client the list of available control-plane nodes

This can be done extending the client-go configuration with an additional field that allows users to include a list of alternative control-plane nodes that the client can use.

`k8s.io/client-go/rest/config.go`
```go
type Config struct {
	// Host must be a host string, a host:port pair, or a URL to the base of the apiserver.
	// If a URL is given then the (optional) Path of that URL represents a prefix that must
	// be appended to all request URIs used to access the apiserver. This allows a frontend
	// proxy to easily relocate all of the apiserver endpoints.
	Host string
  // AlternativeHosts must be a comma separated list if hosts, host:port pairs or URLs to the base of
  // different apiservers. The client can use any of them to access the apiserver.
	AlternativeHosts string
```

2. RFC7838 HTTP Alternative Services

This specification defines a new concept in HTTP, "Alternative Services", that allows an origin server to nominate additional means of interacting with it on the network.

All conformant clusters are required to publish a list of endpoints with the apiserver addresses.In the default implementation, this list of endpoints is generated by a reconcile loop that guarantees that only ready apiserver addresses are present, see https://github.com/kubernetes/kubernetes/tree/master/pkg/controlplane/reconcilers.

The proposal is for API servers to have an option to enable the generation of the Alternative Services headers based on the list of Endpoints created for the `kubernetes.default` service, this requires that the users of this feature belong to the cluster network. TODO define a way of filtering the headers in the answer, per example, by source IP, certificate name, ... 


```sh
1103 12:30:59.066469 1558329 round_trippers.go:454] GET https://127.0.0.1:44267/api/v1/namespaces/default/pods?limit=500 200 OK in 1 milliseconds
I1103 12:30:59.066484 1558329 round_trippers.go:460] Response Headers:
I1103 12:30:59.066491 1558329 round_trippers.go:463]     Cache-Control: no-cache, private
I1103 12:30:59.066502 1558329 round_trippers.go:463]     Alt-Svc: h2="10.0.0.2:6443", h2="10.0.0.3:6443", h2="10.0.0.4:6443
```

The Alternative Services RFC defines options to [implement caching](https://datatracker.ietf.org/doc/html/rfc7838#section-3.1),
however, it doesn't seem necessary because:

- kubernetes control-plane is not a high dynamic environment.
- apiserver already has another mechanisms to warn clients using http codes.
- if the alt-service has gone you will have a connection error, the client will expire that entry from the cache and will try another alt-svc.

### Client side HA

Client-go creates a layer on top of the golang net/http library abstracting the communications against the apiservers using [Requests](https://github.com/kubernetes/client-go/blob/master/rest/request.go).

The golang net/http library allows to modify a request in different points:
- RoundTripper/Transport
- Dialer
- DNS dialer

The client-go base-URL is immutable after creation, that guarantees that Requests will not be able to modify it externally.

Client-go already implements some [custom RoundTrippers](https://github.com/kubernetes/client-go/blob/master/transport/round_trippers.go) for some functionalities, like debugging, authentication, impersonation, ...

The proposal is to implement a new round tripper that has a local cache with the list of available apiservers.
This list can be created manually via configuration (comma-separated list on the Config object) or automatically via the Alt-Svc headers published.

The round tripper behavior, if there are alternative services, will be:

1. Prefer local (same system as the client) alternative services.
2. Prefer the one in the original request if is listed as an alternative service.
3. If one alternative service is chosen and succeed, stick to it, only change on network failures.
4. If the connection against an alternative service fails, block that alternative service for a specified timeout so it will not be retried, retry against the other available alternative services, if everything fails fall back to the original host.

It retries only in case of network failures, it operates only at the network L2/L3/L4 level, that guarantees the idempotency of the request and any possible conflict with others roundtrippers and the client-go retry logic.

The round tripper modifies the original `URL.Host` field with the alternative service `Host` and sets the http2 pseudo header `:authority` field and the SNI TLS server name to `kubernetes.default`
If the connection against the server fails because of a certificate error, that host is blocked forever and will not be retried anymore.


Example:

```
https://apiserver.lb:6443/path?query
```

is converted to:

```
https://10.0.0.2:6443/path?query
Host: kubernetes.default
SNI TLS.ServerName = kubernetes.default
```

Alternative Hosts received on the requests will take precedence and replace the configured ones in the cache.

### Test Plan

- Unit tests
- Integration tests for:
  - client round tripper and request/response behavior against different scenarios simulating errors
  - apiserver correct headers generation for alternative-services
  - client failover behavior against multiple-master using alternative-services headers and preconfigured alternative hosts
- e2e tests for:
  - client failover behavior against multiple-master using alternative-services headers and preconfigured alternative hosts

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag (using apiserver flag)
- Initial unit, integration and e2e tests completed and enabled

#### Beta

- Gather feedback from developers and distributions
#### GA

- 2 examples of real-world usage

#### Deprecation

### Upgrade / Downgrade Strategy

### Version Skew Strategy

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

- Use a custom dialer https://github.com/aojea/client-go-multidialer
- Use an embedded in-memory DNS server to resolve API endpoints https://github.com/aojea/kubernetes/pull/1

The alternatives require a first connection to succeed, and create an additional request to fetch the Endpoints
object with the API server advertises addresses.

In addition to the additional requests, the code is harder to maintain in the longer term and can interfere
with some of the current 

## Infrastructure Needed (Optional)
