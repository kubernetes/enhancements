# KEP-1669: Proxy Terminating Endpoints

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Additions to EndpointSlice](#additions-to-endpointslice)
  - [kube-proxy](#kube-proxy)
  - [Test Plan](#test-plan)
    - [Unit Tests](#unit-tests)
    - [E2E Tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
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

- [X] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] KEP approvers have approved the KEP status as `implementable`
- [X] Design details are appropriately documented
- [X] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes some enhancements to kube-proxy to handle terminating endpoints in an effort to improve the traffic engineering capabilities and overall relability of Kubernetes.
These changes will depend on recent changes to the EndpointSlice API as part of KEP-1672 to include terminating pods in the EndpointSlice API.

## Motivation

Historically, Kubernetes has ignored terminating Pods from both the Endpoints and EndpointSlice API. KEP-1672 recently introduced an API change in EndpointSlice
where terminating endpoints are now included in EndpointSlice, with the addition of two new endpoint conditions "Serving" and "Terminating". Even though the EndpointSlice
API now includes terminating endpoints, kube-proxy strictly forwards traffic to only "Ready" pods that are not terminating. There are several scenarios where not handling
terminating endpoints can lead to traffic loss. It's worth diving into one specific scenario described in [this issue](https://github.com/kubernetes/kubernetes/issues/85643):

When using Service Type=LoadBalancer w/ externalTrafficPolicy=Local, the availability of node backend is determined by the healthCheckNodePort served by kube-proxy.
Kube-proxy returns a 200 on this endpoint if there is a local ready endpoint for a Serivce, otherwise it returns 500 signalling to the load balancer that the node should be removed
from the backend pool. Upon performing a rolling update of a Deployment, there can be a small window of time where old pods on a node are terminating (hence not "Ready") but the load balancer
has not probed kube-proxy's healthCheckNodePort yet. In this event, there is traffic loss because the load balancer is routing traffic to a node where the proxy rules will blackhole
the traffic due to a lack of local endpoints. The likihood of this traffic loss is impacted by two factors: the number of local endpoints on the node and the interval between health checks
from the load balancer. The worse case scenario is a node with 1 local endpoint and a load balancer with a long health check interval.

Currently there are several workarounds that users can leverage:
* Use Kubernetes scheduling/deployment features such that a node would never only have terminating endpoints. For example, always scheduling two pods on a node and only allowing 1 pod to update a time.
* Reducing the load balancer health check interval to very small time windows. This may not alwyays be possible based on the load balancer implementation.
* Use a preStop hook in the Pod to delay the time between a Pod terminating and the process receiving SIGTERM.

While some of these solutions help, there's more that Kubernetes can do to handle this complexity for users.

### Goals

* Reduce potential traffic loss that occurs on rolling updates because trafffic is sent to Pods that are terminating.

### Non-Goals

* Changing the behavior of how pods terminate.

## Proposal

This KEP proposes that if all endpoints for a given Service scoped to its traffic policy are terminating (i.e. pod.DeletionTimestamp != nil), then all traffic should be sent to
terminating Pods that are still Ready. Note that the EndpointSlice API introduced a new condition called "Serving" which is semantically equivalent to "Ready" except that the Ready condition
must always be "False" for terminating pods for compatibility reasons. For consumers of the EndpointSLice API that want to route traffic strictly based on a Pod's readiness ignoring
it's terminating state, they should be reading the Serving condition going forward. Below are some examples to help illustrate the proposed behavior:

### Example: only some endpoints terminating when traffic policy is "Cluster"

When the traffic policy is "Cluster" and some endpoints are terminating, all traffic should be routed to the ready endpoints that are not terminating..

### Example: only some endpoints terminating on a node when traffic policy is "Local"

When the traffic policy is "Local" and some endpoints are terminating within a single node, traffic should be routed to ready endpoints on that node that are not terminating.

### Example: all endpoints terminating and traffic policy is "Cluster"

When the traffic policy is "Cluster" and all endpoints are terminating, then traffic should be routed to any terminating endpoint that is ready.

### Example: all endpoints terminating on a node when traffic policy is "Local"

When the traffic policy is "Local" and all endpoints are terminating within a single node, then traffic should be routed to any terminating endpoint that is ready on that node.


### Handling terminating endpoints that are not ready.

It is worth noting that traffic should not be routed to terminating pods if their readiness probe is failing, even if it is the only endpoints remaining. This is to give workloads
the flexibility/control to opt out of this behavior by either exiting immediately or failing the readiness probe when receiving SIGTERM from kubelet. This would also be counter-intuitive
to the current understanding of readiness probes.


### User Stories (optional)

#### Story 1

As a user I would like to do a rolling update of a Deployment fronted by a Service Type=LoadBalancer with ExternalTrafficPolicy=Local.
If a node that has only 1 pod of said deployment goes into the `Terminating` state, all traffic to that node is dropped until either a new pod
comes up or my cloud provider removes the node from the loadbalancer's node pool. Ideally the terminating pod should gracefully handle traffic to this node
until either one of the conditions are satisfied.

### Risks and Mitigations

There are scalability implications to tracking termination state in EndpointSlice. For now we are assuming that the performance trade-offs are worthwhile but
future testing may change this decision. See KEP 1672 for more details.

## Design Details

### Additions to EndpointSlice

This work depends on the `Terminating` condition existing on the EndpointSlice API (see KEP 1672) in order to check the termination state of an endpoint.

### kube-proxy

Updates to kube-proxy when watching EndpointSlice:
* update kube-proxy endpoints info to track terminating endpoints based on endpoint.condition.terminating in EndpointSlice.
* update kube-proxy endpoints info to track endpoint readiness based on endpoint.condition.ready in EndpointSlice
* within the scope of the traffic policy for a Service, iterate the following prioritized set of endpoints, picking the first set that has at least 1 ready endpoint:
  * ready endpoints that are not terminating
  * ready endpoints that are terminating

In addition, kube-proxy's node port health check should fail if there are only `Terminating` endpoints, regardless of their readiness in order to:
* remove the node from a loadbalancer's node pool as quickly as possible
* gracefully handle any new connections that arrive before the loadbalancer is able to remove the node
* allow existing connections to gracefully terminate

### Test Plan

#### Unit Tests

kube-proxy unit tests:

* Unit tests will validate the correct behavior when there are only local terminating endpoints.
* Unit tests will validate the changein behavior against the matrix of possible Service configurations using both internalTrafficPolicy and externalTrafficPolicy.
* Existing unit tests will validate that terminating endpoints are only used when there are no ready endpoints, otherwise ready && !terminating endpoints are used.
* Unit tests will validate health check node port succeeds only when there are ready && !terminating endpoints.

#### E2E Tests

E2E tests will be added to validate that no traffic is dropped during a rolling update for a Service. E2E tests should cover all permutations of externalTrafficPolicy
and internalTrafficPolicy.

All existing E2E tests for Services should continue to pass.

### Graduation Criteria

#### Alpha

* kube-proxy internally tracks the `terminating` and `serving` condition from EndpointSlice
* kube-proxy falls back to terminating endpoints if and only if they are the only available endpoints.
* feature is only enabled if the feature gate `ProxyTerminatingEndpoints` is on.
* unit tests in kube-proxy.

#### Beta

* E2E tests are in place, exercising all permutations of internalTrafficPolicy and externalTrafficPolicy.
* Metrics to publish how many Services/Endpoints are routing traffic to terminating endpoints.

### Upgrade / Downgrade Strategy

Behavioral changes to terminating endpoints will apply once kube-proxy is upgraded to v1.19 and the `EndpointSlice`/`EndpointSliceProxying` feature gates are enabled.
On downgrade, the worse case scenario is that kube-proxy falls back to the existing behavior. See [Version Skew Strategy](#version-skew-strategy) below.

### Version Skew Strategy

The worse case version skew scenario is that kube-proxy falls back to the existing behavior today where traffic does not fall back to terminating endpoints.
This would either happen if a version of the control plane was not aware of the additions to EndpointSlice or if the version of kube-proxy did not know to consume the additions to EndpointSlice.

There's not much risk involved as the worse case scenario is falling back to existing behavior.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ProxyTerminatingEndpoints
  - Components depending on the feature gate: kube-proxy

###### Does enabling the feature change any default behavior?

Yes, when there are only terminating (and ready) endpoints, kube-proxy will route traffic to those endpoints. Before this change, kube-proxy
dropped or disallowed this traffic instead.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

###### What happens if we reenable the feature if it was previously rolled back?

kube-proxy will no longer drop traffic if only terminating endpoints are available.

###### Are there any tests for feature enablement/disablement?

Yes, there will be unit tests in kube-proxy with the feature gate enabled and disabled.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?
-->

TBD for beta.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

TBD for beta.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

TBD for beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

TBD for beta.

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

TBD for beta.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

TBD for beta.

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

TBD for beta.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

TBD for beta.

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

TBD for beta.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

TBD for beta.

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

TBD for beta.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

TBD for beta.

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

- [x] 2020-04-23: KEP accepted as implementable for v1.19

## Drawbacks

* scalability: this KEP (and KEP 1672) would add more writes per endpoint to EndpointSlice as each terminating endpoint adds at least 1 and at
most 2 additional writes - 1 write for marking an endpoint as "terminating" and another if an endpoint changes it's readiness during termination.
* complexity: an additional corner case is added to kube-proxy adding to it's complexity.

## Alternatives

Some users work around this issue today by adding a preStop hook that sleeps for some duration. Though this may work in some scenarios, better handling from kube-proxy
would alleviate the need for this work around altogether.

