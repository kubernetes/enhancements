# KEP-4939: TLS Credentials in gRPC Probe

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
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Design](#api-design)
  - [Feature Gate Behavior](#feature-gate-behavior)
  - [Kubelet Probe Execution](#kubelet-probe-execution)
  - [gRPC Transport Credentials](#grpc-transport-credentials)
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

The new gRPC health probe enables developers to probe [gRPC health servers](https://github.com/grpc-ecosystem/grpc-health-probe) from the node. This allows them to stop using workarounds such as this grpc-health-probe paired with `exec` probes.

It allows natively running health checks on gRPC services without deploying additional binaries as well as other benefits outlined in [the announcement](https://kubernetes.io/blog/2022/05/13/grpc-probes-now-in-beta/).

A limitation in the current implementation is that it only supports gRPC servers that do not leverage TLS connections. Even if they are not concerned about certificate verification for the health check, a connection cannot be established at all if the server is expecting TLS and the client is not.

This enhancement aims to add configuration options to enable TLS on the gRPC probe.

## Motivation

We often deploy internal gRPC services on our cluster. These deployments provide internal services and it's simple to add the health server to them so they can be verified through a single interface.

It's also worth noting we have an internal CA that signs certs for communicating with these servers and all of them use TLS.

Currently, we are using the exec probe for readiness and liveness configured as:

```yaml
livenessProbe:
  exec:
    command:
      - "/bin/grpc_health_probe"
      - "-addr=:8443"
      - "-tls"
      - "-tls-no-verify"
```
We would really like to switch to the gRPC probes introduced in 1.24 but are unable to do so since there is no way to configure it to use a TLS connection when reaching out to the health server.

Instead we must continue to rely on the exec probe and cannot reap the benefits described in [the announcement](https://kubernetes.io/blog/2022/05/13/grpc-probes-now-in-beta/).


### Goals

The primary goal is to support TLS connections when using the grpc probe. The probe will use TLS but not verify the certificate.

### Non-Goals

It is not a goal of this KEP to support providing a certificate to verify the TLS connection.

## Proposal

Add a new optional `mode` field alongside `port` and `service` in the
[Probe GRPCAction](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#grpcaction-v1-core).
It indicates whether the probe should connect using TLS or plaintext, and
serves as a basis for future TLS-related probe functionality if desired.

### User Stories

#### Story 1

As a platform engineer running internal gRPC services with TLS enabled
(via an internal CA), I want to configure native gRPC liveness and readiness
probes with `mode: TLS` so that I can stop bundling `grpc_health_probe` in
every container image and relying on `exec` probes just to health-check a
TLS endpoint.

#### Story 2

As an application developer whose gRPC server only accepts TLS connections,
I want the kubelet's built-in gRPC probe to connect over TLS so that my
healthy containers are not marked unhealthy due to a TLS handshake failure.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

- The threat model is similar to existing probes hitting a node local endpoint.
  The TLS connection uses `InsecureSkipVerify: true` because the probe connects
  to the pod's own IP / localhost where certificate verification is impractical.
  This does not weaken security compared to existing plaintext gRPC probes.

- Adds more code to kubelet and surface area to `Pod.Spec`.

## Design Details

### API Design

A new optional `mode` field is added to the existing `GRPCAction` struct.
The field is a pointer to a `GRPCProbeMode` enum (`nil` preserves existing
plaintext behavior). Gated by the `GRPCContainerProbeTLS` feature gate on
both kube-apiserver (validation / field-dropping) and kubelet (probe execution).

**New enum `GRPCProbeMode`:**

```go
// +enum
type GRPCProbeMode string

const (
    GRPCProbeModePlaintext GRPCProbeMode = "Plaintext"
    GRPCProbeModeTLS       GRPCProbeMode = "TLS"
)
```

**Updated `GRPCAction`:**

```go
type GRPCAction struct {
    Port    int32    `json:"port" protobuf:"varint,1,opt,name=port"`
    Service *string  `json:"service" protobuf:"bytes,2,opt,name=service"`
    // +featureGate=GRPCContainerProbeTLS
    // +optional
    Mode *GRPCProbeMode `json:"mode,omitempty" protobuf:"bytes,3,opt,name=mode,casttype=GRPCProbeMode"`
}
```

**Pod spec example: TLS probe:**

```yaml
livenessProbe:
  grpc:
    port: 8443
    mode: TLS
```

**Pod spec example: explicit plaintext:**

```yaml
livenessProbe:
  grpc:
    port: 50051
    mode: Plaintext
```

**Pod spec example: default (nil mode, plaintext, backward compatible):**

```yaml
livenessProbe:
  grpc:
    port: 50051
```

### Feature Gate Behavior

When `GRPCContainerProbeTLS` is **disabled**:

- The `mode` field is **silently dropped** from new and updated pods by
  `dropDisabledGRPCContainerProbeTLS` during the strategy phase (PrepareForCreate /
  PrepareForUpdate). Users do not see an error, the pod is created without the field.
- Validation acts as a defense-in-depth safety net: if the field somehow survives
  the drop (e.g., due to a code bug), validation rejects the pod with a `Forbidden`
  error. In normal operation this path is never hit.
- If an existing pod already has `mode` set (created while the gate was enabled),
  the field is **preserved** in etcd for backward compatibility so that
  read-modify-write cycles do not lose data.

### Kubelet Probe Execution

In `pkg/kubelet/prober/prober.go`, the kubelet reads the `Mode` field:

```go
useTLS := p.GRPC.Mode != nil && *p.GRPC.Mode == v1.GRPCProbeModeTLS
```

This boolean is passed to the gRPC prober.

### gRPC Transport Credentials

In `pkg/probe/grpc/grpc.go`, transport credentials are selected based on the
`useTLS` flag:

```go
var transportCreds credentials.TransportCredentials
if useTLS {
    transportCreds = credentials.NewTLS(&tls.Config{
        InsecureSkipVerify: true,
    })
} else {
    transportCreds = insecure.NewCredentials()
}
```

`InsecureSkipVerify: true` is used because the probe connects to the pod's own
IP / localhost where certificate verification is impractical.

### Test Plan

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

The following unit tests have been added:

- **`pkg/api/pod`**
  - `TestDropGRPCContainerProbeTLS`: Verifies that `mode` is stripped from all container types (regular, init, ephemeral) when the feature gate is disabled, and preserved when enabled or when the field is already persisted on an existing pod.
  - `TestGRPCContainerProbeTLSValidationOptions`: Verifies that `AllowGRPCContainerProbeTLS` is set correctly based on gate state and old pod spec.

- **`pkg/apis/core/validation`**
  - `TestValidateGRPCAction`: Verifies that `mode: TLS` and `mode: Plaintext` are accepted when the gate is enabled, rejected with `Forbidden` when disabled, and unsupported values like `"Verify"` are rejected with `NotSupported`.

- **`pkg/apis/core/v1`**
  - `TestSetDefaultProbeGRPCMode`: Verifies that `mode: TLS`, `mode: Plaintext`, and `nil` mode are all preserved through round-trip defaulting with no unwanted mutation.

- **`pkg/probe/grpc`**
  - `TestGrpcProber_Probe`: Existing plaintext probe tests updated to pass `useTLS=false`.
  - `TestGrpcProber_Probe_TLS`: Verifies TLS probe succeeds against a TLS server, plaintext probe fails against a TLS server, and TLS probe fails against a plaintext server.

##### Integration tests

Integration tests will be added.

##### e2e tests

The following e2e tests have been added in `test/e2e/common/node/container_probe.go`:

- **`should *not* be restarted with a GRPC liveness probe with TLS mode`**: Creates a pod with a TLS-enabled gRPC health server and a liveness probe using `mode: TLS`. The probe should succeed and the restart count must remain zero.
- **`should be restarted with a GRPC liveness probe when not using TLS against a TLS server`**: Creates a pod with a TLS-enabled gRPC health server and a plaintext liveness probe (no `mode` set). The probe should fail the TLS handshake and the container must be restarted.
- **`should be restarted with a GRPC liveness probe with TLS mode when endpoint returns not healthy`**: Creates a pod with a TLS-enabled gRPC service that returns NOT_SERVING after a delay. The liveness probe uses `mode: TLS`. The probe should detect the unhealthy response and restart the container.
- **`should be restarted with a GRPC liveness probe with TLS mode on wrong port`**: Creates a pod with a TLS-enabled gRPC service on port 5000. The liveness probe uses `mode: TLS` but targets a wrong port where nothing is listening. The probe should fail and restart the container.

All tests are gated by `framework.WithFeatureGate(features.GRPCContainerProbeTLS)`.

### Graduation Criteria

#### Alpha

- API field implemented and functional
- Unit tests passing
- Documentation available

#### Beta

- No major bugs reported during alpha
- Gather feedback from users

#### GA

- Stable for at least two releases
- No major issues reported


### Upgrade / Downgrade Strategy

No special upgrade steps are required. The `mode` field defaults to `nil`,
which preserves the existing plaintext behavior. Existing pods are unaffected
on upgrade.

On downgrade (or if the `GRPCContainerProbeTLS` feature gate is disabled):

- The API server drops the `mode` field from new or updated pods.
- Existing pods that had `mode` set retain the field in etcd, but the kubelet
  on the older version ignores it and falls back to plaintext.
- Pods relying on `mode: TLS` to reach a TLS-only server will begin failing
  probes, which is the same behavior that existed before this feature.

No data migration is needed. The feature is purely additive and opt-in.

### Version Skew Strategy

This feature requires the `GRPCContainerProbeTLS` feature gate on both
kube-apiserver and kubelet.

- **API server newer than kubelet:** The API server accepts `mode: TLS`,
  but the older kubelet ignores the field and dials plaintext. TLS-only
  servers will fail probes, identical to pre-feature behavior.
- **Kubelet newer than API server:** The older API server drops the `mode`
  field, so the kubelet never sees it and dials plaintext.

Both components must have the gate enabled for TLS probes to function.
Partial enablement degrades gracefully to plaintext with no crashes or
unexpected behavior.

Note: the kubelet intentionally does **not** check the feature gate at probe
execution time. It relies on the apiserver as the source of truth, if `mode`
is present in the pod spec, it was persisted by an apiserver that had the gate
enabled. This avoids a confusing state where the field is set in the spec but
silently ignored at runtime.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `GRPCContainerProbeTLS`
  - Components depending on the feature gate:
    - `kube-apiserver` (validation, field dropping)
    - `kubelet` (probe execution)

###### Does enabling the feature change any default behavior?

No. The `mode` field defaults to `nil`, which preserves the existing plaintext
behavior. Only pods that explicitly set `mode: TLS` are affected.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the `GRPCContainerProbeTLS` feature gate and restarting
kube-apiserver and kubelet will cause the `mode` field to be dropped from new
or updated pods. Existing pods that had `mode` set retain the field in etcd,
but the kubelet will ignore it and fall back to plaintext. Pods relying on
`mode: TLS` to reach a TLS-only server will begin failing probes, which is
the same behavior that existed before this feature.

**Recommended rollback procedure for workloads already using `mode: TLS`:**

1. Ensure the gRPC services in affected containers can accept plaintext
   connections (configure dual-mode or plaintext-only listeners).
2. Update Deployments / StatefulSets to remove the `mode` field from probe
   specs (or set `mode: Plaintext`) and roll out the change.
3. Verify all pods are healthy with plaintext probes.
4. Disable the `GRPCContainerProbeTLS` feature gate on kube-apiserver and
   kubelet, then restart both components.

If steps 1–2 are skipped, pods with TLS-only servers will experience probe
failures and restarts once the gate is disabled.

###### What happens if we reenable the feature if it was previously rolled back?

Pods that still have `mode: TLS` persisted in etcd will start using TLS
probes again. New pods can set `mode: TLS` as expected.

###### Are there any tests for feature enablement/disablement?

Yes. `TestDropGRPCContainerProbeTLS` verifies the `mode` field is dropped
when the gate is disabled and preserved when enabled or when the old pod
already uses it. `TestGRPCContainerProbeTLSValidationOptions` verifies
validation allows or rejects the field based on gate state.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

###### What specific metrics should inform a rollback?

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Query pods for `.spec.containers[*].livenessProbe.grpc.mode`,
`.spec.containers[*].readinessProbe.grpc.mode`, or
`.spec.containers[*].startupProbe.grpc.mode` being set.

###### How can someone using this feature know that it is working for their instance?

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. The `mode` field is part of the existing `PodSpec` and is read by kubelet
from the pod spec it already has locally. No additional API calls are made.
The probe execution happens entirely on the node.

###### Will enabling / using this feature result in introducing new API types?

No. This adds a single optional string field (`mode`) to the existing
`GRPCAction` struct. No new API types are introduced.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No. The TLS connection is made locally from kubelet to the container's gRPC
health server on the node. No cloud provider APIs are involved.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, minimally. Pods that use `mode: TLS` will have an additional field in
their `GRPCAction` spec. The increase is approximately 10–15 bytes per probe
that uses it. This is negligible.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

The TLS handshake adds a small amount of latency to each gRPC probe
execution compared to plaintext. This does not affect pod startup SLOs because
probes run after the container is started. The existing probe timeout
configuration already accounts for execution time. No existing SLI/SLO is
impacted.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Negligible increase. The TLS handshake requires a small amount of additional
CPU for cryptographic operations and memory for short-lived TLS session state
on kubelet. At scale (thousands of pods with TLS probes on a single node), the
overhead remains minimal because probes run sequentially per-pod and the TLS
session is torn down immediately after the health check.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No more than existing gRPC probes. Each probe already opens a TCP socket to
the container. TLS adds a handshake on that same socket but does not open
additional connections. If a TLS handshake hangs, the existing probe timeout
applies and the connection is closed. The per-node pod limit already bounds
the maximum number of concurrent probes.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 2026-05-21: KEP created

## Drawbacks

No significant drawbacks beyond the added complexity noted in Risks.

## Alternatives

**Nested `tls` struct with `mode` field.** The
[Previous KEP proposal](https://github.com/kkoch986/enhancements/blob/0e2ba3bb95e73aaed31e5dfb60aa2061424de265/keps/sig-node/4939-tls-in-grpc-probe/README.md)
added a `tls` sub-struct to `GRPCAction` with a `mode` field (`NoVerify`).
The presence of the struct indicated TLS should be used. This was rejected by
reviewers because the probe will never validate certificates (it connects to
localhost/pod IP), so a nested struct adds unnecessary complexity. A flat
`mode` field on `GRPCAction` is simpler and sufficient.

**Boolean `tls` flag.** A simple `tls: true/false` boolean was
[discussed](https://github.com/kubernetes/enhancements/pull/5029#discussion_r1936341743).
This was rejected in favor of a string enum (`mode`) to allow explicit naming
of the connection type (`"TLS"`, `"Plaintext"`) and to leave room for future
values without a breaking API change.

## Infrastructure Needed (Optional)