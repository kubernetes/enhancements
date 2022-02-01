# KEP-2727: Add GRPC Probe

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Goals](#goals)
- [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Alternative Considerations](#alternative-considerations)
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
- [Implementation History](#implementation-history-1)
  - [Alpha](#alpha-1)
  - [Beta](#beta-1)
- [Alternatives](#alternatives)
- [References](#references)
<!-- /toc -->


## Release Signoff Checklist

- [X] Enhancement issue in release milestone, which links to KEP dir in
  [kubernetes/enhancements] (not the initial KEP PR)
- [X] KEP approvers have approved the KEP status as `implementable`
- [X] Design details are appropriately documented
- [X] Test plan is in place, giving consideration to SIG Architecture
      and SIG Testing input
- [X] Graduation criteria is in place
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in
  [kubernetes/website], for publication to [kubernetes.io]
- [X] Supporting documentation e.g., additional design documents,
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
standard healthcheck service (`Check` method) to determine the health of the
container. Using `Watch` method of the healthcheck service is not supported,
but may be considered in future iterations.
As spec'd, the `kubelet` probe will not allow use of client
certificates nor verify the certificate on the container.  We do not
support other protocols for the time being (unencrypted HTTP/2, QUIC).

The healthcheck request will be identified with the following gRPC
`User-Agent` metadata. This user agent will be statically defined (not
configurable by the user):

```
User-Agent: kube-probe/K8S_MAJOR_VER.K8S_MINOR_VER
```

Example:

```
User-Agent: kube-probe/1.23
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

### Alternative Considerations

Note that `readinessProbe.grpc.service` may be confusing, some
alternatives considered:

- `serviceName`
- `healthCheckServiceName`
- `grpcService`
- `grpcServiceName`

There were no feedback on the selected name being confusing in the context of a probe definition.

### Graduation Criteria

#### Alpha

- Implement the feature.
- Add unit and e2e tests for the feature.

#### Beta

- Solicit feedback from the Alpha.
- Ensure tests are stable and passing.

Depending on skew strategy:

- kubelet version skew ensures all (kubelet ver, cluster ver) support
  the feature.

#### GA

- Address feedback from beta usage
- Validate that API is appropriate for users. There are some potential tunables:
  - `User-Agent`
  - connect timeout
  - protocol (HTTP, QUIC)
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

###### How can an operator determine if the feature is in use by workloads?

When gRPC probe is configured, Pod must be scheduled and, the metric
`probe_total` can be observed to see the result of probe execution.

###### How can someone using this feature know that it is working for their instance?

When gRPC probe is configured, Pod must be scheduled and, the metric
`probe_total` can be observed to see the result of probe execution.

Event will be emitted for the failed probe and logs available in `kubelet.log`
to troubleshoot the failing probes.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Probe must succeed whenever service has returned the correct response
in defined timeout, and fail otherwise.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

The metric `probe_total` can be used to check for the probe result. Event and
`kubelet.log` log entries can be observed to troubleshoot issues.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Creation of a probe duration metric is tracked in this issue:
https://github.com/kubernetes/kubernetes/issues/101035 and out of scope for this
KEP.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Adds < 200 bytes to Pod.Spec, which is consistent with other probe types.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

The overhead of executing probes is consistent with other probe types.

We expect decrease of disk, RAM, and CPU use for many scenarios where the https://github.com/grpc-ecosystem/grpc-health-probe
was used to probe gRPC endpoints.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No dependency on etcd availability.

###### What are other known failure modes?

None

###### What steps should be taken if SLOs are not being met to determine the problem?

- Make sure feature gate is set
- Make sure configuration is correct and gRPC service is reacheable by kubelet.
  This may be different when migrating off https://github.com/grpc-ecosystem/grpc-health-probe
  and is covered in feature documentation.
- `kubelet.log` log must be analyzed to understand why there is a mismatch of
  service response and status reported by probe.

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

### Alpha

Alpha feature was implemented in 1.23.

### Beta

Feature is promoted to beta in 1.24.

## Alternatives

* 3rd party solutions like https://github.com/grpc-ecosystem/grpc-health-probe

## References

* GRPC healthchecking: https://github.com/grpc/grpc/blob/master/doc/health-checking.md
