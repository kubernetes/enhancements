# KEP-2727: Add GRPC Probe

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Goals](#goals)
- [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
- [References](#references)
<!-- /toc -->


## Release Signoff Checklist

- [ ] Enhancement issue in release milestone, which links to KEP dir in
  [kubernetes/enhancements] (not the initial KEP PR)
- [ ] KEP approvers have approved the KEP status as `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture
      and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in
  [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents,
  links to mailing list discussions/SIG meetings, relevant PRs/issues,
  release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Goals

Enable gRPC probe natively from Kubelet without requiring users to package a
gRPC healthcheck binary with their container.

- https://github.com/grpc-ecosystem/grpc-health-probe
- https://github.com/grpc/grpc/blob/master/doc/health-checking.md

## Non-Goals

Add gRPC support in other areas of K8s (e.g. Services).

## Proposal

Add the follow configuration to the `LivenessProbe`, `ReadinessProbe`
and `StartupProbe`. Example:

```yaml
    readinessProbe:
      grpc:                     #+
        port: 9090              #+
        service: my-service     #+
      initialDelaySeconds: 5
      periodSeconds: 10
```

This will result in the use of gRPC (using HTTP/2 over TLS) to use the
standard healthcheck service to determine the health of the
container. As spec'd, the `kubelet` probe will not allow use of client
certificates nor verify the certificate on the container.  We do not
support other protocols for the time being (unencrypted HTTP/2, QUIC).

Note that `readinessProbe.grpc.service` may be confusing, some
alternatives:

- `serviceName`
- `healthCheckServiceName`
- `grpcService`
- `grpcServiceName`

These options can be added in Beta with user feedback.

The healthcheck request will be identified with the following gRPC
`User-Agent` metadata. This user agent will be statically defined (not
configurable by the user):

```
User-Agent: kubernetes/K8S_MAJOR_VER.K8S_MINOR_VER
```

Example:

```
User-Agent: kubernetes/1.22
```

### Risks and Mitigations

1. Adds more code to Kubelet and surface area to Pod.Spec. *Response*: we
   expect that this will be generally useful given broad gRPC adoption in the
   industry.

## Design Details

```go
// core/v1/types.go

type Handler struct {
  // ...
  TCPSocket *TCPSocketAction `json...`

  // GRPC specifies an action involving a TCP port. //+
  // +optional                                      //+
  GRPC *GRPCAction `json...`                        //+

  // ...
}

type GRPCAction struct {                                                         //+
  // Port number of the gRPC service. Number must be in the range 1 to 65535.    //+
  Port int32 `json:"port" protobuf:"bytes,1,opt,name=port"`                      //+
                                                                                 //+
  // Service is the name of the service to place in the gRPC HealthCheckRequest  //+
  // (see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).      //+
  //                                                                             //+
  // The service name can be the empty string (i.e. "").                         //+
  Service string `json:"service" protobuf:"bytes,2,opt,name=service"`            //+
                                                                                 //+
  // Host is the host name to connect to, defaults to the Pod's IP.              //+
  Host string `json,omitempty", protobuf:"bytes,3,opt,name=host"`                //+
}                                                                                //+
```

Note that `GRPCAction.Port` is an int32, which is inconsistent with
the other existing probe definitions. This is on purpose -- we want to
move users away from using the (portNum, portName) union type.

### Test Plan

- Unit test: Add unit tests to `pkg/kubelet/prober/...`
- e2e: Add test case and conformance test to `e2e/common/node/container_probe.go`.

### Graduation Criteria

#### Alpha

- Implement the feature.
- Add unit and e2e tests for the feature.

#### Beta

- Solicit feedback from the Alpha. Validate that API is appropriate
  for users. There are some potential tunables:
  - `User-Agent`
  - connect timeout
  - protocol (HTTP, QUIC)
- Ensure tests are stable and passing.

Depending on skew strategy:

- kubelet version skew ensures all (kubelet ver, cluster ver) support
  the feature.

#### GA

- Address feedback from beta
- Close on any remaining open issues & bugs

### Upgrade / Downgrade Strategy

Upgrade: N/A

Downgrade: gRPC probes will not be supported in a downgrade from Alpha.

### Version Skew Strategy

- We may not be able to graduate this widely until all kubelet version
  skew supports the probe type.


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

Feature enablement will be guarded by a feature gate flag.

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.
-->

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `GRPCContainerProbe`
  - Components depending on the feature gate: `kubelet` (probing), API
    server (API changes).

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. This would require restarting kubelet, and so probes for existing
Pods would no longer run.

###### What happens if we reenable the feature if it was previously rolled back?

It becomes enabled again after the `kubelet` restart.

###### Are there any tests for feature enablement/disablement?

Y
es, unit tests for the feature when enabled and disabled will be
implemented in both kubelet and api server.

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

No

### Monitoring Requirements

TODO for Beta.

<!--

###### How can an operator determine if the feature is in use by workloads?

TODO for Beta.

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
->

###### How can someone using this feature know that it is working for their instance?

TODO for Beta.

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
->

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
->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
->

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

Beta TODO

<!--
This section must be completed when targeting beta to a release.
->

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

Adds < 200 bytes to Pod.Spec.


###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

### Troubleshooting

Beta TODO.
<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
->

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
->

###### What steps should be taken if SLOs are not being met to determine the problem?

-->

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

## Implementation History

* Original PR for k8 Prober: https://github.com/kubernetes/kubernetes/pull/89832
* 2020-04-04: MR for k8 Prober
* 2021-05-12: Cloned to this KEP to move the probe forward.
* 2021-05-13: Updates.

## Alternatives

* 3rd party solutions like https://github.com/grpc-ecosystem/grpc-health-probe

## References

* GRPC healthchecking: https://github.com/grpc/grpc/blob/master/doc/health-checking.md
