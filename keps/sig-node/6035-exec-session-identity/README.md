# KEP-6035: Exec Session Identity Propagation

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
- [Design Details](#design-details)
  - [The Exec Call Chain](#the-exec-call-chain)
  - [CRI Primitive: ExecRequest.envs](#cri-primitive-execrequestenvs)
  - [Environment Variable Precedence](#environment-variable-precedence)
  - [Scope](#scope)
  - [Hooks and Probes](#hooks-and-probes)
  - [Audit ID Propagation](#audit-id-propagation)
  - [Validation](#validation)
  - [Silent Degradation](#silent-degradation)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
      - [CRI conformance tests](#cri-conformance-tests)
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
  - [OTel Baggage Propagation](#otel-baggage-propagation)
  - [Keeping the CRI Primitive as a Separate KEP](#keeping-the-cri-primitive-as-a-separate-kep)
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
- [ ] Supporting documentation, e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

When a user runs `kubectl exec`, the API server authenticates the caller and records their identity in the audit log, but this information is never forwarded to the kubelet or the container runtime. This makes it impossible for runtime security tooling to attribute in-container activity to the Kubernetes user who initiated the session. This KEP proposes propagating the API server's audit request ID, a UUID generated per-request and recorded in the audit log, through the kubelet and into each exec session, so that runtime security agents can correlate in-container process activity with the originating request. Raw user credentials (such as usernames or tokens) are not transmitted to the node.

Delivering the audit ID into the exec'd process requires a reliable CRI-level mechanism for environment variable injection. The existing workaround of prepending the exec command with the `env` binary is unreliable: it fails silently on distroless or minimal images, and carries no runtime contract. This KEP therefore also adds a `repeated KeyValue envs` field to `ExecRequest` in the CRI protobuf, establishing a contract that the runtime must inject the provided key-value pairs into the OCI process spec before handing off to the low-level runtime.

## Motivation

Kubernetes audit logs record who opened an exec session and when. Runtime security agents (such as Falco or Tetragon) record what commands and syscalls are executed inside containers. Today, there is no reliable mechanism to link these two sources of information.

The gap is structural: the `ExecRequest` protobuf message in the CRI specification contains only six fields (`container_id`, `cmd`, `tty`, `stdin`, `stdout`, `stderr`). There is no field for request metadata. The API server's `Connect` handler proxies the exec request to the kubelet using the API server's own credentials, not the original caller's. The audit ID that was present in the request context is dropped and never forwarded.

When multiple exec sessions overlap on the same pod, it becomes impossible to determine which user ran a specific command. Time-based correlation between audit logs and runtime events only works when sessions do not overlap, a condition that cannot be guaranteed in production.

This is not a tooling limitation at the transport layer. No runtime security agent can recover request information that was never forwarded, because the data is lost at the protocol level before it reaches the runtime. This gap has been independently identified by the Falco community: [falcosecurity/falco#2895](https://github.com/falcosecurity/falco/issues/2895).

The Security Profiles Operator (SPO) currently works around the identity gap by using a mutating webhook to inject the API server's request UID as an environment variable (`SPO_EXEC_REQUEST_UID`) into exec sessions. This approach requires an external operator, and relies on prepending the exec command with the `env` binary, a workaround that is unreliable because the `env` binary is not guaranteed to be present in distroless or minimal images, and inline `KEY=VALUE cmd` is a shell feature that is not interpreted by `execve()` directly. There is no reliable CRI-level mechanism to inject environment variables into exec sessions today.

Solving the identity correlation gap requires addressing two distinct problems:

- **The transport problem**: how does the audit ID travel from the API server to the container runtime through the Kubernetes stack? This KEP solves the transport problem.
- **The attribution problem**: once it arrives, how do we ensure it cannot be tampered with by the caller? This is explicitly out of scope and deferred to follow-up work. See [Bridging the Kubernetes Exec Identity Gap](https://blog.sigtrapd.dev/posts/bridging-k8s-exec-identity-gap/) for background on out-of-tree approaches to this problem.

### Goals

- Add `repeated KeyValue envs` to `ExecRequest` in the CRI protobuf, establishing a clear runtime contract for environment variable injection at exec time.
- Add `Env []EnvVar` to `PodExecOptions` as the Kubernetes API surface for this capability, allowing user agents (such as `kubectl exec --env`) to reliably inject environment variables into exec'd processes without depending on any binary or shell being present inside the container image.
- Propagate the API server audit request ID, not raw user identity, end-to-end as the first consumer of the CRI primitive.
- Enable runtime security agents to correlate in-container process activity with the originating exec request, using the audit ID as a join key against the API server audit log.

### Non-Goals

- Solving the attribution problem: ensuring the propagated audit ID cannot be tampered with by the caller. Deferred to follow-up work (e.g., eBPF-based agents capturing the value at `execve` time into kernel-managed storage before any userspace code runs).
- Identity propagation for lifecycle hooks and probes via `ExecSyncRequest`. Probes and hooks do not originate from API server requests, so there is no audit ID to propagate. Adding the `envs` field to `ExecSyncRequest` is deferred until a concrete consumer for it is identified.
- Modifying RBAC policies or admission control. This KEP does not change who is allowed to exec into a pod.
- Propagating raw user credentials (usernames, tokens, groups) to nodes.

## Proposal

This KEP makes two coordinated changes.

The first is a CRI API change: add `repeated KeyValue envs` to `ExecRequest` in the CRI protobuf. The contract: the runtime must mechanically inject the provided key-value pairs into the exec'd process's environment, without filtering or modification. This is a general-purpose primitive. Any future caller (user-facing env injection, tracing metadata, etc.) that needs to surface key-value data into exec sessions can use this field without further API changes.

The second change is the first consumer of that primitive: the API server reads the audit request ID from the request context and appends it as `?env=KUBERNETES_EXEC_AUDIT_ID=<value>` on the HTTP request to the kubelet. The kubelet parses the `env` query parameter and forwards it to the CRI runtime via `ExecRequest.envs`. The runtime injects it as an environment variable into the exec'd process, where runtime security agents can capture it.

The audit ID (a UUID such as `f4a3b2c1-...`) is a per-request identifier assigned by the API server and present in audit log entries. It is not the user's name or credentials: it is an opaque correlation key. A security agent that captures `KUBERNETES_EXEC_AUDIT_ID` from the exec'd process can look up that UUID in the API server audit log to find the full user context (username, groups, UID) associated with that exec request.

### User Stories

#### Story 1

As a security engineer investigating a production incident, I need to determine which Kubernetes user ran a specific command inside a container when multiple exec sessions were active on the same pod simultaneously. Today, the API server audit log tells me who opened sessions and when, and my runtime security agent tells me what commands were executed, but there is no way to link a specific command to a specific user when sessions overlap.

With this KEP, each exec session carries the API server's audit request ID. My eBPF-based runtime agent captures the `KUBERNETES_EXEC_AUDIT_ID` environment variable at `execve` time and includes it in its process event records. I join that UUID against the API server audit log to find the requesting user's identity.

#### Story 2

As a platform security engineer writing runtime detection rules, I want to alert when a specific user runs commands in a privileged pod. Today, runtime agents can only match on process-level attributes (command, container, namespace) because Kubernetes user context is not available at the runtime layer. With this KEP, the propagated audit ID serves as an indirect reference to the requesting user that my detection rules can use.

#### Story 3

As a cluster operator, I want to inject environment variables into exec sessions without depending on any binary or shell being present in the container image. Today, the only option is to prepend `env KEY=VALUE` to the exec command, which fails silently on distroless images. With this KEP, `kubectl exec --env KEY=VALUE` provides a reliable mechanism to inject metadata at exec time without any in-container dependency.

### Risks and Mitigations

- **Environment variable tamperability.** Once injected into the exec'd process, userspace code inside the exec session can overwrite or unset the `KUBERNETES_EXEC_AUDIT_ID` value before a security agent reads it. Consumers should treat it as a correlation hint rather than a security assertion. Tamper-proofing via kernel-space eBPF capture is deferred to follow-up work.
- **Log exposure.** Env vars are passed as `?env=KEY=VALUE` query parameters on the HTTP request to the kubelet and can appear in access logs and other request logging at the API server and kubelet. Values injected via this mechanism should not be sensitive. Future use cases requiring sensitive values are out of scope for this KEP; a natural extension would be to allow `PodExecOptions.Env` entries to reference Kubernetes Secrets or ConfigMaps, similar to how `Container.env` supports `valueFrom`.
- **Sensitive environment variable injection (PATH, LD_PRELOAD, etc.).** Exposing `PodExecOptions.Env` as a first-class API field raises the question of whether callers could inject dangerous keys such as `PATH` or `LD_PRELOAD` to influence the exec'd process. However, this risk already exists today: any caller with exec permission can achieve the same effect via the `env` binary on containers that have it. The appropriate mitigation is at the RBAC level: exec permission should not be granted to untrusted users. Additionally, by surfacing env vars as a structured `PodExecOptions.Env` field rather than burying them inside the command array, admission controllers (Gatekeeper, Kyverno) can now inspect and enforce policies against specific keys, which was not possible before.
- **CRI runtime compatibility.** Runtimes that have not yet implemented `ExecRequest.envs` will silently ignore the field (standard protobuf behaviour for unrecognised fields). The feature degrades gracefully. Beta graduation requires at least two CRI implementations (containerd and CRI-O) to have implemented and tested the field.

## Design Details

This KEP uses two feature gates with distinct responsibilities:

- **`ExecEnvVar`** (kube-apiserver and kubelet): gates the end-to-end env var wiring, namely `PodExecOptions.Env` parsing at the API server, `env` query parameter forwarding at the kubelet, and `ExecRequest.envs` population to the CRI runtime. This is the general-purpose primitive and must be enabled for any env injection to work.
- **`ExecRequestID`** (kube-apiserver): gates the audit ID injection specifically, namely the API server reading the audit ID from context and injecting it as `KUBERNETES_EXEC_AUDIT_ID` into the `env` query parameter. Requires `ExecEnvVar` to be enabled.

The CRI contract (`ExecRequest.envs` and kubelet wiring) and the Kubernetes API object changes (`PodExecOptions.Env`) are designed to be independently landable. The audit ID propagation uses the `env` query parameter on the kubelet HTTP request and does not depend on `PodExecOptions.Env`, so the exec identity gap can be closed even if the API object changes face review delays.

### The Exec Call Chain

The exec request flows through the following steps:

1. The API server receives the exec request, generates an audit ID, and records it in the audit context for this request.
2. With the feature gate enabled, the API server reads the audit ID from `audit.AuditContext(ctx).AuditID()` and appends it as `?env=KUBERNETES_EXEC_AUDIT_ID=<value>` to the HTTP request URL sent to the kubelet.
3. The kubelet receives the request, parses the `env` query parameter, and makes a gRPC `Exec` call to the CRI runtime with `ExecRequest.envs` populated from the parsed values.
4. The CRI runtime injects the provided env vars into the exec'd process's environment and returns a streaming URL.
5. The kubelet dials the streaming URL, hijacks both ends, and splices them together.
6. The exec session runs. Runtime security agents on the node (e.g., eBPF-based agents) observe the `KUBERNETES_EXEC_AUDIT_ID` env var at `execve` time and record it alongside process events.

The audit ID must be delivered at step 3, the gRPC call. By step 5, the connection is a raw byte splice with no opportunity to inject metadata into the container environment.

### CRI Primitive: ExecRequest.envs

Add `repeated KeyValue envs` to `ExecRequest` in `staging/src/k8s.io/cri-api/pkg/apis/runtime/v1/api.proto`:

```protobuf
message ExecRequest {
    string container_id = 1;
    repeated string cmd = 2;
    bool tty = 3;
    bool stdin = 4;
    bool stdout = 5;
    bool stderr = 6;
    repeated KeyValue envs = 7;  // new field
}
```

The contract established by this change: the runtime must inject all provided env vars into the exec'd process's environment. Values are injected literally; no expansion or interpolation is performed by the runtime, regardless of whether a value contains shell-style references such as `$VAR` or `${VAR}`. Callers that need dynamic values must resolve them before placing them in `ExecRequest.envs`.

### Environment Variable Precedence

The final environment of the exec'd process is constructed so that earlier entries take precedence on collision:

```
final_envp[] = runtime_envs + kubelet_envs + apiserver_envs + caller_envs + container_envs
```

Highest to lowest precedence:

1. Runtime-injected envs (envs the container runtime adds for its own internal needs)
2. Kubelet-injected envs (reserved for future node-level use)
3. API server-injected envs (e.g., `KUBERNETES_EXEC_AUDIT_ID`)
4. Caller-provided envs (e.g., from `kubectl exec --env`)
5. Container baseline envs (from OCI image config and `CreateContainerRequest`)

Runtime-injected envs sit at the top because the container runtime may need to inject envs for its own internal purposes (e.g., to make the runtime function correctly), and `ExecRequest.envs` should not be able to override these. No widely-used CRI runtime injects such envs today, but reserving the slot at the top of the precedence order keeps the contract safe for any future runtime that does.

This ordering is otherwise intentionally counter-intuitive: conventionally, the layer closest to the process would take highest precedence. Here it is reversed for the kubelet, API server, and caller layers because keys like `KUBERNETES_EXEC_AUDIT_ID` must not be overridable by the exec caller or the container's base environment. If a caller passes `KUBERNETES_EXEC_AUDIT_ID=fake` via `PodExecOptions.Env`, the request is rejected by API server validation before reaching the kubelet (see [Validation](#validation)).

If a key appears multiple times within a single precedence tier (e.g., two entries with `LOG_LEVEL` in `ExecRequest.envs`), the last entry wins.

Caller-provided envs outranking the container baseline is intentional and useful. A user running `kubectl exec --env LOG_LEVEL=debug` should be able to override a `LOG_LEVEL` defined in the container spec for the duration of the exec session. Similarly, a user may want to point a tool at a different endpoint by overriding a `SERVICE_URL` env var that was baked into the image. This only applies to envs explicitly set at exec time; the container's default environment remains unchanged for any keys the caller does not override.

### Scope

- `ExecRequest` (streaming exec) is in scope. This covers `kubectl exec` and all programmatic exec calls.
- `ExecSyncRequest` (used by lifecycle hooks and probes) is out of scope. There is no audit ID or other concrete consumer for these paths in this KEP, so the field is not added (see [Hooks and Probes](#hooks-and-probes)).
- `AttachRequest` is explicitly out of scope. Attach connects to an already-running process; modifying the environment of a running process would require `ptrace`, which is invasive and beyond this proposal's scope. Attach sessions are traceable via API server audit logs.

### Hooks and Probes

Lifecycle hooks (`postStart`, `preStop`) and liveness/readiness/startup probes use `ExecSync` (`ExecSyncRequest`), which executes synchronously and does not go through the streaming exec path. These calls do not originate from API server requests, so there is no audit ID to propagate. Extending env injection to `ExecSyncRequest` would require a different consumer (e.g., kubelet-supplied metadata) and is deferred until a concrete use case is identified.

### Audit ID Propagation

The audit request ID is a UUID assigned by the API server to each incoming request and recorded in audit log entries. To be precise: the audit ID is **not** the user's name or group; it is an opaque identifier. To look up the user associated with an exec session, a security agent joins the captured `KUBERNETES_EXEC_AUDIT_ID` value against the `auditID` field in the API server audit log, where the full user context (username, groups, UID) is available.

**API server** (`pkg/registry/core/pod/rest/subresources.go:Connect`)

Read `audit.AuditContext(ctx).AuditID()` and append `?env=KUBERNETES_EXEC_AUDIT_ID=<value>` to the kubelet request URL before forwarding the exec request. If the audit context is absent or returns an empty ID (e.g., audit logging is disabled), inject a fixed placeholder value (e.g., `KUBERNETES_EXEC_AUDIT_ID=unknown`) so the variable is always present when the feature gate is enabled. Caller-supplied `KUBERNETES_EXEC_AUDIT_ID` entries in `PodExecOptions.Env` are rejected at validation time (see [Validation](#validation)).

A Prometheus counter `apiserver_exec_audit_id_injected_total` is incremented on each exec request, with a `result` label distinguishing `injected` (real audit ID), `placeholder` (fallback value), and `skipped` (feature gate disabled). This metric is the primary signal for operators to confirm the feature is in use and to detect regressions.

**`PodExecOptions` API type**

Add `Env []EnvVar` to both the internal and versioned `PodExecOptions` type, with the same field semantics as `Container.Env`. This is a new versioned Kubernetes API field and triggers the standard API review, feature gate, and graduation process. It generalises to arbitrary key-value injection into exec sessions for future use cases without further API changes. This requires deepcopy/conversion/OpenAPI regeneration.

**Kubelet** (`pkg/kubelet/server/server.go:getExec`)

Parse the `env` query parameters from the incoming HTTP request and forward them to the CRI runtime as `ExecRequest.envs`. When the feature gate is disabled, the `env` query parameters are ignored and `ExecRequest.envs` is left empty.

**kubectl** (`staging/src/k8s.io/kubectl/pkg/cmd/exec`)

Add an `--env KEY=VALUE` flag (repeatable) to `kubectl exec`. The values are populated into `PodExecOptions.Env` on the request to the API server, which then includes them in the serialised `PodExecOptions` query on the kubelet request. The flag is gated on the `ExecEnvVar` feature gate at the server side; older API servers will reject unknown fields.

**CRI-O and containerd**

Read `ExecRequest.envs` and inject the provided key-value pairs into the exec'd process's environment, following the precedence rules in [Environment Variable Precedence](#environment-variable-precedence).

### Validation

`PodExecOptions.Env` is validated by the API server in `pkg/apis/core/validation` before the exec request is forwarded to the kubelet. Validation rules align with the existing rules for `Container.Env` where applicable:

- **`Name` is required and must match `[-._a-zA-Z][-._a-zA-Z0-9]*`.** This is the same pattern Kubernetes already uses for `Container.Env` names. Empty names are rejected.
- **`Value` may be empty.** Setting a key with an empty string is a valid use case.
- **Duplicate keys are rejected at the API level.** A request containing two entries with the same `Name` returns a validation error. This prevents ambiguity at the source; the `last entry wins` rule in [Environment Variable Precedence](#environment-variable-precedence) applies only when entries arrive from different layers (e.g., kubelet vs. API server).
- **Maximum 256 entries per request.** Prevents unbounded URL/audit log growth.
- **Maximum 32KB total size across all entries.** Prevents oversized requests that could exceed practical URL length limits or bloat audit logs.
- **Reserved keys are rejected with a clear error.** `KUBERNETES_EXEC_AUDIT_ID` and any other reserved keys (defined in a constant list) cannot be supplied by callers; the API server returns a validation error such as `"KUBERNETES_EXEC_AUDIT_ID is reserved by the API server"`. This gives users immediate feedback rather than silently stripping the entry.

### Silent Degradation

If a runtime receives `ExecRequest.envs` populated but has not yet implemented forwarding, the field is silently ignored (standard protobuf zero-value behaviour for unrecognised fields). Exec sessions continue to work normally; the audit ID is simply not injected into the process environment. This is consistent with how other new CRI fields have been handled historically and is acceptable during the rollout period.

Rather than relying purely on protobuf zero-value behaviour, the kubelet can negotiate runtime support explicitly via the existing CRI runtime features mechanism (`RuntimeStatus`). A two-phase rollout is under consideration:

- **Alpha**: detect-and-warn. The kubelet queries runtime features and, if `ExecRequest.envs` is unsupported, logs an event and emits a metric (e.g., `kubelet_exec_envs_unsupported_total`) but allows the exec to proceed. Operators get visibility without breakage.
- **Beta**: detect-and-optionally-deny. A kubelet config flag (e.g., `--require-exec-env-support`) defaulting to off lets operators with strict audit/compliance requirements reject exec on unsupported runtimes with a clear error.

The exact shape of this negotiation will be finalised during implementation. Beta graduation explicitly requires at least two runtimes to have implemented the field.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

Unit tests will cover the following packages:

- `pkg/registry/core/pod/rest`: API server appends `?env=KUBERNETES_EXEC_AUDIT_ID=<value>` to the kubelet request URL when the feature gate is enabled; does not append when disabled.
- `pkg/kubelet/server`: kubelet parses `env` query parameters and populates `ExecRequest.envs`; omits when the feature gate is disabled.
- `pkg/apis/core/validation`: validation of `PodExecOptions.Env` entries.
- CRI client stub tests: `ExecRequest.envs` is populated correctly from the parsed query parameters.

##### Integration tests

Integration tests (spanning the API server without running the kubelet or CRI runtime) will cover:

- The API server appends `?env=KUBERNETES_EXEC_AUDIT_ID=<value>` to the outbound kubelet exec request URL when the feature gate is enabled.
- The API server does not append env params when the feature gate is disabled.

##### e2e tests

The following e2e tests will be added:

**Audit ID propagation (requires both feature gates enabled):**

1. Run `kubectl exec` against a pod on a CRI runtime that supports `ExecRequest.envs`.
2. Verify that `KUBERNETES_EXEC_AUDIT_ID` is present in the exec'd process's environment.
3. Verify the `KUBERNETES_EXEC_AUDIT_ID` value matches the `auditID` field in the API server audit log entry for that exec request.

**User-provided env injection (requires `ExecEnvVar` gate enabled):**

1. Run `kubectl exec --env LOG_LEVEL=debug` against a pod where `LOG_LEVEL` is already set in the container spec.
2. Verify that `LOG_LEVEL=debug` is present in the exec'd process's environment, confirming caller-provided envs override the container baseline.

**Concurrent sessions produce distinct audit IDs:**

1. Open two concurrent `kubectl exec` sessions against the same pod simultaneously.
2. Verify that each session carries a distinct `KUBERNETES_EXEC_AUDIT_ID` value, confirming the feature correctly attributes overlapping sessions.

**Spoofing prevention:**

1. Attempt to pass `KUBERNETES_EXEC_AUDIT_ID=fake` as a user-provided env via `kubectl exec --env`.
2. Verify that the request is rejected by API server validation with a clear error indicating the key is reserved.

##### CRI conformance tests

This KEP introduces a new CRI contract that must be validated across runtimes. The following test cases should be added to `kubernetes-sigs/cri-tools` (`critest`) as conformance requirements for beta. Precedence resolution between kubelet-injected, caller-provided, and container baseline envs happens at the kubelet; from the CRI runtime's perspective the contract is simpler: whatever arrives in `ExecRequest.envs` must take precedence over the container baseline:

- **Env injection**: verify that env vars provided in `ExecRequest.envs` are present in the exec'd process's environment.
- **Precedence over container baseline**: verify that `ExecRequest.envs` entries override conflicting keys from the container's baseline environment, not the other way around.
- **No variable expansion against container baseline**: verify that env vars inherited from the container baseline are not expanded or interpolated when `ExecRequest.envs` entries are added. Values must be injected literally.
- **No variable expansion within ExecRequest.envs**: send `ExecRequest.envs = [{FOO=bar}, {BAZ=$FOO}]` and verify that `BAZ` is set to the literal string `$FOO`, not `bar`. Repeat with `${FOO}` and `%FOO%` patterns to confirm no interpolation is performed regardless of syntax.

### Graduation Criteria

#### Alpha

- `ExecEnvVar` feature gate implemented (kube-apiserver and kubelet): end-to-end env var wiring from `PodExecOptions.Env` through `env` query parameter to `ExecRequest.envs`.
- `ExecRequestID` feature gate implemented (kube-apiserver): audit ID injection as `?env=KUBERNETES_EXEC_AUDIT_ID=<value>` on kubelet exec requests, including placeholder for absent audit context.
- `repeated KeyValue envs` added to `ExecRequest` in the CRI protobuf.
- `Env []EnvVar` added to `PodExecOptions` (internal and versioned).
- `kubectl exec --env KEY=VALUE` flag added (repeatable), populating `PodExecOptions.Env`.
- API-level validation for `PodExecOptions.Env` (name format, no duplicates, count/size limits, reserved-key rejection).
- `apiserver_exec_audit_id_injected_total` counter exposed by the API server with `result` label (`injected` / `placeholder` / `skipped`).
- Initial unit tests completed and passing.
- Initial e2e tests completed and enabled.

#### Beta

- No major bugs reported during alpha.
- At least two CRI implementations (containerd and CRI-O) support `ExecRequest.envs` and have been validated end-to-end.
- CRI conformance tests added to `kubernetes-sigs/cri-tools` (`critest`) covering env injection, precedence over container baseline, and no variable expansion.
- Integration tests linked in KEP and passing in Testgrid.
- e2e tests stable in Testgrid with no flakes in the last month.
- All monitoring requirements completed.
- All known alpha issues resolved.

#### GA

- Stable for at least two releases with no major issues reported.

### Upgrade / Downgrade Strategy

The feature is additive. On upgrade, no configuration changes are required to maintain previous behavior. To use the feature, enable the `ExecEnvVar` and `ExecRequestID` feature gates on the API server and kubelet, and ensure the CRI runtime supports `ExecRequest.envs`. On downgrade, the API server stops appending `env` query parameters and the kubelet stops populating `ExecRequest.envs`. Exec sessions continue to work normally. No persistent state is created, so there is nothing to clean up on downgrade.

### Version Skew Strategy

This enhancement involves the API server, kubelet, and CRI runtime. All skew scenarios degrade gracefully:

- **Newer API server, older kubelet:** The API server appends `?env=KUBERNETES_EXEC_AUDIT_ID=<value>` to the kubelet request URL. If the kubelet does not recognise the `env` query parameter, it is ignored and `ExecRequest.envs` is not populated. The exec session works; the audit ID is not injected.
- **Newer kubelet, older CRI runtime:** The kubelet populates `ExecRequest.envs`. If the runtime has not implemented the field, it is silently ignored (protobuf zero-value behaviour). The exec session works; the env var is not injected.
- **Older API server, newer kubelet/runtime:** The API server does not append `env` query parameters. The kubelet sends an empty `ExecRequest.envs`. The runtime injects nothing additional. Normal exec behaviour is unchanged.

No coordination between components is required for rollout.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ExecEnvVar` (gates end-to-end env var wiring at the API server and kubelet)
  - Components depending on the feature gate:
    - kube-apiserver
    - kubelet
  - Feature gate name: `ExecRequestID` (gates audit ID injection at the API server; requires `ExecEnvVar`)
  - Components depending on the feature gate:
    - kube-apiserver
    - kubelet

###### Does enabling the feature change any default behavior?

Yes, when the feature gates are enabled: the API server starts appending `?env=KUBERNETES_EXEC_AUDIT_ID=<value>` to kubelet exec request URLs, and the kubelet starts forwarding `env` query parameters to the CRI runtime via `ExecRequest.envs`. CRI runtimes that support the field will inject the `KUBERNETES_EXEC_AUDIT_ID` environment variable into exec'd processes. Both gates are disabled by default at alpha, so there is no change to default cluster behavior. Existing exec sessions and workloads are not affected.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling `ExecEnvVar` stops the kubelet from forwarding `env` query parameters to `ExecRequest.envs`. Disabling `ExecRequestID` stops the API server from appending `?env=KUBERNETES_EXEC_AUDIT_ID=<value>` to kubelet requests. No existing workloads are affected. In-flight exec sessions continue without interruption since identity propagation applies only at session creation time.

###### What happens if we reenable the feature if it was previously rolled back?

New exec sessions will again include the audit ID in `ExecRequest.envs`. There is no persistent state, so re-enablement has no side effects.

###### Are there any tests for feature enablement/disablement?

Yes. The observable signal is the `KUBERNETES_EXEC_AUDIT_ID` environment variable in the exec'd process: present when both feature gates are enabled, absent when either is disabled. The e2e test covers both cases. There is no separate unit-level gate test; the gate simply controls whether the code path that appends the `env` query parameter is exercised.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

###### What specific metrics should inform a rollback?

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?

The `apiserver_exec_audit_id_injected_total` counter on the API server tracks every exec request that passes through the audit ID injection path. A non-zero `injected` count indicates the feature is in active use. A non-zero `placeholder` count indicates the feature is enabled but the audit context is missing for some requests (worth investigating). A non-zero `skipped` count indicates the feature gate is disabled.

###### How can someone using this feature know that it is working for their instance?

- [x] Metrics
  - Metric name: `apiserver_exec_audit_id_injected_total{result="injected"}`
  - Components exposing the metric: kube-apiserver
- [x] Other
  - Details: Run `kubectl exec <pod> -- env | grep KUBERNETES_EXEC_AUDIT_ID`. If the variable is present and contains a UUID, the feature is functioning correctly for that exec session. The UUID should match the `auditID` field in the API server audit log entry for that exec request.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

When both feature gates are enabled, every exec request reaching the API server should result in either an `injected` or `placeholder` increment of `apiserver_exec_audit_id_injected_total`. The ratio of `placeholder` to `injected` should remain near zero in clusters with audit logging enabled; a sustained non-zero `placeholder` rate indicates audit context is missing and warrants investigation.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `apiserver_exec_audit_id_injected_total`
  - Aggregation method: counter, summed by `result` label (`injected` / `placeholder` / `skipped`)
  - Components exposing the metric: kube-apiserver

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

None at alpha. A kubelet-side counter for `ExecRequest.envs` population could be added at beta if operators need node-level visibility separately from the control plane signal.

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

This feature requires the CRI runtime to support `ExecRequest.envs`. Runtimes that do not support the field will silently ignore it, and the feature degrades gracefully. The feature is inert without runtime support but does not impact cluster operation.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. The feature adds a field to an existing exec proxy request. No new API calls are introduced.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No. The audit ID is propagated transiently as a query parameter on the kubelet HTTP request and via the CRI `ExecRequest.envs` field. It is never stored in etcd. No persistent API objects are modified in size or count.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No measurable increase. Reading a UUID from the audit context and adding a single `KeyValue` to an existing protobuf message is negligible.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The feature adds a single `KeyValue` entry (~60 bytes) to an existing exec request. No additional in-memory state, disk I/O, or network traffic beyond that.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. The feature does not create any new processes, sockets, or files. It only adds metadata to an existing exec request flow.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

The feature is part of the exec request path. If the API server is unavailable, exec requests fail entirely as they do today. The feature does not introduce additional failure modes.

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics?
    - Mitigations: What can be done to stop the bleeding?
    - Diagnostics: What are the useful log messages and their required logging levels?
    - Testing: Are there any tests for failure mode?
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 2026-04-28: KEP-6035 created (exec session identity propagation via audit ID)
- 2026-05-18: KEP-6090 created (CRI exec environment variable injection primitive)
- 2026-05-19: Proposal presented at SIG Node meeting ([recording](https://www.youtube.com/watch?v=bl8OX2nlEfM&t=900s), discussion starts at 15:00)
- 2026-06-15: KEP-6035 and KEP-6090 merged into this KEP per reviewer feedback; CRI primitive and audit ID propagation are presented as a single coherent proposal

## Drawbacks

Adding `ExecRequest.envs` extends the CRI surface and requires updates to CRI implementations. The additional implementation cost per runtime is minimal: each runtime must populate the `env` array in the OCI process spec it constructs before handing off to the low-level runtime (e.g., runc).

## Alternatives

### OTel Baggage Propagation

The audit ID could be propagated out-of-band using [W3C Baggage](https://www.w3.org/TR/baggage/). The API server would inject the audit ID as a baggage member (`audit-id=<value>`) on the outbound HTTP request to the kubelet. The kubelet HTTP server already has an `otelrestful` filter that extracts W3C Baggage from incoming HTTP headers into the Go `context.Context`. That context flows to the gRPC `Exec` call, where an `otelgrpc` stats handler injects it as gRPC metadata. The CRI runtime extracts the audit ID from the incoming metadata and appends it to the environment variables before caching the `ExecRequest`.

CRI-O and containerd each require a one-line change to pass the gRPC handler context through to `GetExec`, plus an interface change in the `cri-streaming` library.

However, the contract (that the W3C Baggage key `audit-id` becomes the environment variable `KUBERNETES_EXEC_AUDIT_ID` in the exec process) is implicit in the runtime's baggage-reading code rather than visible in the CRI API or Kubernetes API. The mechanism is specific to out-of-band metadata propagation and does not generalise to arbitrary env injection for future callers. A first-class CRI field is preferred for inspectability, testability, and generality.

### Keeping the CRI Primitive as a Separate KEP

KEP-6090 was initially proposed as a standalone KEP for the CRI `ExecRequest.envs` primitive, with KEP-6035 as the first consumer. Reviewers noted that KEP-6035 implicitly depended on KEP-6090 without stating it, and requested that the two be compressed into one. Landing a CRI-only change with no user-visible consumer in the same release is also harder to justify to SIGs. The merged KEP presents a coherent end-to-end story with a clear motivation, from the structural gap in the API server through to the CRI contract and runtime injection.

## Infrastructure Needed (Optional)
