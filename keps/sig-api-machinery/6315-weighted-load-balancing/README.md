# KEP-6315: Weighted Load Balancing for Kube-Apiserver

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
    - [Protocol Standard &amp; Industry Adoption](#protocol-standard--industry-adoption)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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

In High-Availability (HA) Kubernetes clusters (e.g. 3-node control planes), traffic is distributed across `kube-apiserver` replicas through external or internal load balancers. Today, these load balancers operate with unweighted routing algorithms (such as round-robin or random distribution). However, control plane load across apiserver instances is inherently asymmetric. While on average traffic on nodes is distributed equally, leader elected controllers connect to usually only one `kube-apiserver`, establishing dozens of informers, persistent watch streams, and heavy write loops against single backend. 

Because unweighted load balancers have no visibility into server load, they continue sending equal shares of external and cluster traffic to the already-saturated instance. This causes severe "hot node" degradation, latency spikes, API Priority and Fairness (APF) request rejections (HTTP 429), and potential node crashes, while other control plane replicas remain underutilized.

This KEP introduces **Weighted Load Balancing** support for `kube-apiserver`. By adopting the open **Open Request Cost Aggregation (ORCA)** standard, `kube-apiserver` communicates its real-time capacity and utilization to upstream load balancers via HTTP response headers. This standed is supported by many layer 7 load balancers (such as Envoy Proxy, HAProxy, and cloud load balancers) allowing them to execute **Weighted Round Robin (WRR)** or **Weighted Least Request (WLR)** algorithms, automatically routing traffic to control plane nodes with available capacity and balancing utilization across the cluster.

## Motivation

By enabling **Weighted Round Robin (WRR)** and **Weighted Least Request (WLR)** load balancing, the load balancer dynamically scales the routing weight of each `kube-apiserver` proportionally to its available headroom. When one apiserver is heavily loaded by active controllers, its weight is reduced, steering external traffic (from kubelets, CI/CD, user requests, and cluster add-ons) toward underutilized nodes and maintaining balanced control plane performance.

### Goals

- Adopt an open telemetry standard to expose real-time utilization metrics to load balancers.
- Select a minimal set of utilization metrics that are validated to improve load distribution.
- Improve load distribution for unary requests like POST, PATCH, DELETE, GET, LIST. 
- Provide a path to expand signals, if we see an opportunity to improve load distribution for other request types.

### Non-Goals

- Expose all possible resource and APF signals.
- Implement client-side load balancing.

## Proposal

We propose supporting **Weighted Load Balancing for `kube-apiserver`** by exposing real-time CPU and inflight request count via the Open Request Cost Aggregation (ORCA) telemetry standard using HTTP headers.. 

When enabled via the `RequestCostAggregation` feature gate kube-apiserver will:
1. Enable background goroutine to collect real time metrics, we expect around 100ms sampling rate should be a good balance of accuracy and overhead.
2. Install a HTTP middleware that will attach standard ORCA headers (`endpoint-load-metrics: TEXT cpu_utilization=0.3, mem_utilization=0.8, rps_fractional=10.0`) to outgoing responses.

### Notes/Constraints/Caveats

- **out-of-band reporting**: For L7 load balancers that cannot inspect response or L4 load balancer, ORCA supports out-of-band reporting. Those usecases will be addressed at later Beta stage.
- **Header Size Overhead**: The ORCA HTTP header adds a negligible payload (approx. 40–80 bytes) per HTTP response.

### Risks and Mitigations

TODO

## Design Details

#### Protocol Standard & Industry Adoption
The **Open Request Cost Aggregation (ORCA)** specification is an open, vendor-neutral standard created within the CNCF ecosystem (developed collaboratively by Envoy Proxy, gRPC, and Istio communities). ORCA defines standard schemas and protocols for backend services to communicate utilization and request costs to load balancers.

ORCA is natively supported by:
- **Envoy Proxy**: via built-in load balancing policies.
- **gRPC**: via the ORCA OpenRcaService and out-of-band / in-band load report filters.
- **Istio / Service Meshes**: via dynamic weighted routing based on backend capacity.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests


##### Integration tests

##### e2e tests

### Graduation Criteria

#### Alpha
- Feature gate `RequestCostAggregation` added (disabled by default).
- Comprehensive unit and integration test coverage.

#### Beta
- Feature gate `RequestCostAggregation` enabled by default.
- Benchmark validation showing zero measurable throughput or latency regression in 5000-node scalability test suites.

#### GA
- Feature gate `RequestCostAggregation` locked to true (unconditionally enabled).

### Upgrade / Downgrade Strategy

To make use of the enhancement, an operator has to do two independent things:
enable the `RequestCostAggregation` feature gate on `kube-apiserver`, and
configure their L7 load balancer to consume ORCA load reports and use a weighted
policy. Either step can be reverted independently, and reverting either one
returns the cluster to unweighted balancing.

### Version Skew Strategy

Telemetry will just be emitted from apiserver has the feature enabled.

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

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `RequestCostAggregation`
  - Components depending on the feature gate: `kube-apiserver`

Enabling or disabling the gate requires a `kube-apiserver` restart.


###### Does enabling the feature change any default behavior?

Yes, but only in that responses served by `kube-apiserver` carry an additional
`endpoint-load-metrics` response header. No request handling, authorization,
serialization, or response body changes in any way.


###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Not applicable.

###### What happens if we reenable the feature if it was previously rolled back?

Not applicable.

###### Are there any tests for feature enablement/disablement?

No, not planned.

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

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No. Nothing is persisted in etcd; the load report exists only on the wire.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No measurable increase is expected. The report is serialized once per sampling
interval by a background goroutine and cached; the request path only performs an
atomic load and sets one header, with no serialization and no allocation on the
hot path.

This will be validated by benchmarks and by scalability tests before Beta.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

- CPU: one background goroutine sampling process/cgroup CPU and memory counters at
  the proposed ~100ms interval; this is a small constant cost independent of QPS.
- RAM: a single cached, pre-serialized report (tens of bytes) plus the sampler
  state.
- Network: the header adds roughly 40-80 bytes per response when sent
  uncompressed. `kube-apiserver` serves HTTP/2, where HPACK indexes the header
  name once and indexes each distinct value, so the marginal cost for most
  responses is a few bytes; a new literal value is only sent once per sampling
  interval. At 10k QPS this is on the order of 0.1 MB/s, which is negligible
  relative to the volume of API response bodies and watch traffic at that scale.

The sampling interval is the main knob controlling both the sampler cost and the
frequency of new HPACK literals, and the exact default will be validated during
Alpha.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. The feature adds a single goroutine and no new file descriptors, sockets, or
connections.

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

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->