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
  PrepareForUpdate). The pod is created without the field, no error is shown.
  Validation (`validateGRPCAction`) is not gate-aware; it only rejects
  unsupported enum values (e.g. `"Verify"`). Gating is enforced by the drop
  step, not validation.
- If an object (e.g. a Pod or Deployment) already has `mode` set, the field is
  **preserved** across further updates to that same object.

### Kubelet Probe Execution

In `pkg/kubelet/prober/prober.go`, the kubelet checks the feature gate live,
on every probe execution, in addition to reading the `Mode` field:

```go
useTLS := utilfeature.DefaultFeatureGate.Enabled(features.GRPCContainerProbeTLS) &&
    p.GRPC.Mode != nil && *p.GRPC.Mode == v1.GRPCProbeModeTLS
```

This boolean is passed to the gRPC prober. Feature gates are read once at
kubelet startup, not hot-reloaded, so a gate change takes effect on the next
kubelet restart. Once that restart happens, it applies immediately to every
already-running pod on that node, not just newly created ones. See
[Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy) for tested
implications.

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
  - `TestDropGRPCContainerProbeTLS`: Verifies that `mode` is stripped from all container types (regular, init, ephemeral) when the feature gate is disabled, and preserved when enabled or when the field is already persisted on the same, existing object.

- **`pkg/apis/core/validation`**
  - `TestValidateGRPCAction`: Verifies that `mode: TLS` and `mode: Plaintext` pass validation, and unsupported values like `"Verify"` are rejected with `NotSupported`. Validation is not gate-aware (see [Feature Gate Behavior](#feature-gate-behavior)); gating is enforced entirely by the drop step, not validation.

- **`pkg/apis/core/v1`**
  - `TestSetDefaultProbeGRPCMode`: Verifies that `mode: TLS`, `mode: Plaintext`, and `nil` mode are all preserved through round-trip defaulting with no unwanted mutation.

- **`pkg/probe/grpc`**
  - `TestGrpcProber_Probe`: Existing plaintext probe tests updated to pass `useTLS=false`.
  - `TestGrpcProber_Probe_TLS`: Verifies TLS probe succeeds against a TLS server, plaintext probe fails against a TLS server, and TLS probe fails against a plaintext server.

##### Integration tests

Integration tests exercise the full REST path (strategy + validation + storage)
against a real kube-apiserver and etcd, in `test/integration/pods`:

- Create a pod with `grpc.mode: TLS` (and, as an additional table case,
  `mode: Plaintext`) when `GRPCContainerProbeTLS` is enabled -> field is
  accepted and persisted.
- Create a pod with `grpc.mode: TLS` when `GRPCContainerProbeTLS` is disabled
  -> field is silently dropped, pod is created without it.
- Create a pod with `grpc.mode` set to an unsupported value (e.g. `"Verify"`)
  -> rejected during validation, independent of gate state.
- Update a Deployment whose pod template already has `grpc.mode: TLS` after
  the gate is disabled -> field is preserved on the existing template
  (`grpcProbeModeInUse(oldPodSpec)`), while a brand-new Deployment created in
  the same disabled state cannot set the field at all. (This preservation
  does not extend to ReplicaSets created from that template, see the Known
  Issue under [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy).)

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

- `GRPCContainerProbeTLS` feature gate defaults to `true`.
- Integration tests added covering create/drop/validation/preserve-on-update
  behavior against a real kube-apiserver and etcd.
- e2e tests stable with the gate on by default.
- Upgrade/downgrade and feature-gate enable/disable manually verified on a
  live cluster, see [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy).
- No major bugs reported against this KEP's own implementation.

#### GA

- Stable for at least two releases
- No major issues reported


### Upgrade / Downgrade Strategy

No special upgrade steps are required. The `mode` field defaults to `nil`,
which preserves the existing plaintext behavior. Existing pods are unaffected
on upgrade.

On downgrade (or if the `GRPCContainerProbeTLS` feature gate is disabled),
manually verified on a live cluster:

- The API server drops the `mode` field from new or updated pods.
- A pod that is already running and not touched again is not affected by the
  gate change alone; the kubelet keeps using the spec it already has cached.
  The change only takes effect once the kubelet process itself restarts. Once
  it restarts, every already-running pod using `mode: TLS` on that node
  immediately starts dialing plaintext and begins failing its probe, without
  needing the pod itself to be recreated.
- Pods relying on `mode: TLS` to reach a TLS-only server begin failing probes
  and restarting, the same behavior that existed before this feature.

No data migration is needed. The feature is purely additive and opt-in.

### Version Skew Strategy

This feature requires the `GRPCContainerProbeTLS` feature gate on both
kube-apiserver and kubelet.

- **API server newer than kubelet:** An older kubelet predating this KEP has
  no `Mode` field in its vendored API types, so it dials plaintext. TLS-only
  servers fail probes, identical to pre-feature behavior. (A kubelet that has
  the code but its own gate set to `false` behaves the same way, see below,
  though that is a distinct case from a genuinely older binary.)
- **Kubelet newer than API server:** The older API server also has no `Mode`
  field, so it drops the field the same way a gate-disabled current-version
  apiserver would; the kubelet never sees it and dials plaintext.

Both components must have the gate enabled for TLS probes to function.
Partial enablement degrades gracefully to plaintext with no crashes.

The kubelet checks the feature gate on every probe (see
[Kubelet Probe Execution](#kubelet-probe-execution)), confirmed by manual
testing: disabling the gate and restarting kubelet made an already-running
pod with `mode: TLS` still in its spec immediately start failing probes.

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
kube-apiserver and kubelet drops the `mode` field from new or updated pods. A
pod that is not touched again keeps `mode: TLS` in its spec and is unaffected
until the kubelet on its node restarts, at which point it immediately starts
failing its probe. Confirmed by manual testing, see
[Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy).

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

Yes. `TestDropGRPCContainerProbeTLS` (unit) verifies the field is dropped
when the gate is disabled and preserved across updates to the same existing
object. Integration tests cover the same behavior against a real apiserver
and etcd. Manual testing on a live cluster additionally verified the
kubelet-restart-dependent runtime behavior, see
[Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy).

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollout: Enabling the gate adds a new optional field. Existing workloads are
unaffected since `mode` defaults to nil (plaintext). The only risk is if a
user sets `mode: TLS` against a server that does not actually serve TLS, in
which case probes will fail and the container will restart, this is
user-misconfiguration, not a rollout failure.

Rollback: Disabling the gate causes the `mode` field to be dropped from new
and updated pods. An already-running pod is unaffected until the kubelet on
its node restarts, at which point it immediately falls back to plaintext and
starts failing probes against a TLS-only server, until the user reconfigures
their services or re-enables the gate.

###### What specific metrics should inform a rollback?

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Yes, manually tested on a live kind cluster ahead of beta:

- Baseline `mode: TLS` behavior confirmed working, with a plaintext probe
  against the same TLS server correctly failing as a negative control.
- Disable-while-running: a pod is unaffected until its node's kubelet
  restarts, then immediately fails probes.
- Re-enable: pods with `mode: TLS` still persisted resume TLS probing once
  the gate is re-enabled and kubelet restarted.
- Version skew (older kubelet against a newer apiserver) was reasoned about
  but not exercised against a genuinely older kubelet binary.

Unit tests (`TestDropGRPCContainerProbeTLS`) additionally verify the field
drop/preserve behavior at the API level across gate transitions.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Query pods for `.spec.containers[*].livenessProbe.grpc.mode`,
`.spec.containers[*].readinessProbe.grpc.mode`, or
`.spec.containers[*].startupProbe.grpc.mode` being set.

###### How can someone using this feature know that it is working for their instance?

The probe result is visible the same way any other liveness/readiness/startup
probe result is: `kubectl describe pod` shows probe failure events if the TLS
handshake or health check fails, and the container's restart count reflects
liveness probe failures. There is no feature-specific status field beyond the
existing probe machinery.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

None beyond the existing SLOs for gRPC probes in general. This feature
changes the transport used by an existing probe type rather than introducing
a new subsystem.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

The existing `prober_probe_total` and `prober_probe_duration_seconds` metrics,
filtered to `probe_type` values covering gRPC probes, apply unchanged. A
sustained increase in `prober_probe_total{result="failure"}` for pods newly
configured with `mode: TLS` indicates a misconfigured TLS backend or, after a
rollback, the gate-disabled fallback-to-plaintext behavior described in
[Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy).

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No new metrics are added or deemed necessary. The existing prober metrics do
not currently distinguish TLS from plaintext gRPC probes; that granularity
was not required for alpha or beta and can be revisited post-GA if operators
request it.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. The gRPC server being probed is provided by the workload itself (the
container being probed); the feature adds no dependency on any additional
in-cluster service.

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

No different from any other probe: the kubelet executes probes based on the
last pod spec it has locally cached, so a transient apiserver/etcd outage does
not stop already-scheduled probes from running. New pods, or updates that
would change `mode`, cannot be created until the apiserver is available again,
consistent with normal Kubernetes behavior.

###### What are other known failure modes?

- **Misconfigured `mode: TLS` against a plaintext-only server:** the TLS
  handshake fails, the probe fails, and the container restarts. This is user
  misconfiguration, not a feature bug.
- **Gate disabled while pods are running with `mode: TLS`:** probes fail once
  the node's kubelet restarts and picks up the new gate value, see
  [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy).

###### What steps should be taken if SLOs are not being met to determine the problem?

Check `kubectl describe pod` for probe failure events and `prober_probe_total`
for a spike in gRPC probe failures. Confirm whether the backend actually
serves TLS on the configured port, whether `mode` matches the backend's
actual protocol, and whether the feature gate state on kube-apiserver and the
node's kubelet agree (mismatched gate state between the two degrades to
plaintext rather than erroring, per [Version Skew Strategy](#version-skew-strategy)).

## Implementation History

- 2026-05-21: KEP created
- 2026-09-09: Beta graduation testing on a live cluster; corrected design
  details around kubelet's feature-gate check and validation behavior.

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