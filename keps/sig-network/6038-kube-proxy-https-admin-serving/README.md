# KEP-6038: HTTPS (TLS) for kube-proxy admin endpoints

## Table of Contents

- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
- [Proposal](#proposal)
    - [User Stories](#user-stories)
    - [Notes / Constraints / Caveats](#notes--constraints--caveats)
    - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
    - [Configuration API](#configuration-api)
    - [Implementation Sketch](#implementation-sketch)
    - [Security Model](#security-model)
    - [Interaction with Cluster Lifecycle](#interaction-with-cluster-lifecycle)
    - [Test Plan](#test-plan)
    - [Graduation Criteria](#graduation-criteria)
    - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements](https://github.com/kubernetes/enhancements)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation in [kubernetes/website](https://github.com/kubernetes/website)
- [ ] Supporting documentation (issues, mailing list threads, relevant PRs)

## Summary

Today kube-proxy serves its **administrative** HTTP endpoints (health checks, Prometheus metrics, optional profiling and debug handlers, configz/flagz, etc.) over **cleartext HTTP** on:

- the **metrics** listener (`MetricsBindAddress`, default `127.0.0.1:10249` on IPv4 nodes), and
- the **health** listener (`HealthzBindAddress`, default `0.0.0.0:10256` / `[::]:10256` depending on configuration).

This KEP proposes **optional TLS (HTTPS)** for those listeners so that operators who expose these ports beyond localhost (or who require encryption-in-transit for compliance) can do so using Kubernetes-consistent certificate configuration, without changing how kube-proxy implements **data-plane** forwarding (iptables/IPVS/nftables/kernelspace).

## Motivation

- **Confidentiality and integrity**: Metrics and health responses can reveal cluster topology, timing, and operational state. On shared networks or when bind addresses are widened, cleartext HTTP is vulnerable to passive observation and tampering.
- **Alignment with other control-plane and node components**: Many Kubernetes components already expose privileged operational endpoints over TLS or are moving that direction; kube-proxy remains a notable exception for its metrics/health servers.
- **Policy / compliance**: Some environments mandate TLS for any management interface, even on the node loopback, when traffic crosses certain boundaries (e.g., sidecars, host networking, or compliance frameworks).

### Goals

- Allow kube-proxy to serve **metrics** and **healthz/livez** (and the other handlers co-located on those servers today) over **HTTPS** when configured.
- Support **operator-provided** certificates (and optional rotation via file reload patterns consistent with other k8s binaries, where feasible).
- Preserve **backward compatibility**: default behavior remains HTTP unless TLS is explicitly enabled.
- Document **Prometheus / kubelet / probes** migration (e.g., `scheme: https`, `insecureSkipVerify` vs proper CA, or mTLS where chosen).

### Non-Goals

- TLS termination for **Service** traffic, **NodePort**, **ClusterIP**, or proxy data paths.
- Replacing kube-proxy with a different dataplane design.
- **Authentication/authorization** for metrics comparable to full Kubernetes API aggregation (unless explicitly extended in a follow-up; initial scope is TLS for the transport).
- Changing default bind addresses or ports (those remain separate decisions).

## Proposal

Introduce explicit **TLS configuration** for:

1. **Metrics / diagnostics server** — today started from `serveMetrics` in `cmd/kube-proxy/app/server.go`, multiplexing `healthz`, SLI metrics, `/metrics`, profiling, `configz`, `flagz`, optional `statusz`, etc., on `MetricsBindAddress`.
2. **Proxy health server** — today `ProxyHealthServer.Run` in `pkg/proxy/healthcheck/proxy_health.go`, serving `/healthz` and `/livez` on `HealthzBindAddress`.

When TLS is **disabled** (default), behavior is unchanged: plain HTTP.

When TLS is **enabled** for a given listener, kube-proxy wraps the TCP listener with `tls.NewListener` (or equivalent) using configured cert/key material and optional client authentication settings.

### User Stories

1. **As a cluster operator**, I bind metrics on a non-loopback interface for scraping; I want **HTTPS** with a cluster-issued or corporate CA so scrapers verify the server identity.
2. **As a security reviewer**, I want kube-proxy’s **health** port on `10256` to use TLS so that JSON health payloads are not sent in cleartext on the node network segment.
3. **As a distro packager**, I want configuration analogous to other components (cert file paths, min TLS version, cipher suites) so we can apply a single security baseline.

### Notes / Constraints / Caveats

- **Dual listeners**: Metrics and health are separate today (see existing TODO in code: healthz and metrics on the same port). This KEP keeps **two** logical servers but allows **each** to enable TLS independently so operators can stagger rollout (e.g., TLS health first for kubelet probes, metrics later for Prometheus).
- **Loopback default**: The default `MetricsBindAddress` is loopback-only, which limits exposure but does not satisfy all compliance wording; TLS remains valuable when operators change binds or when traffic is observable within the node.
- **Probes**: Anything that currently assumes `http://` for `:10256` or `:10249` must be updated when TLS is turned on (kubelet HTTP probes, static manifests, monitoring config).

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| **Breakage of scrapers/probes** when TLS is enabled | TLS is **opt-in**; clear release notes; document Prometheus `tls_config` and kubelet probe `httpsGet` / lifecycle changes. |
| **Certificate expiry** without rotation | Support **file-based** certs with documented rotation (reload strategy aligned with other components—see Design Details). |
| **Cipher / TLS version** drift | Reuse shared helpers / validation patterns from `k8s.io/component-base/cli/flag` or apiserver serving options where practical; document allowed configurations. |
| **Operational complexity** | Start with **one-way TLS** (server auth); document optional **client cert** (mTLS) as a later sub-feature if not in the first milestone. |

## Design Details

### Configuration API

Extend **`KubeProxyConfiguration`** (`pkg/proxy/apis/config`, all served versions, defaults, validation, conversion) with nested structures similar in spirit to other components’ secure serving options, for example:

```yaml
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
metricsBindAddress: 0.0.0.0:10249
healthzBindAddress: 0.0.0.0:10256
# New fields (illustrative names — finalize in API review)
metricsServerTLS:
  enable: true
  certFile: /var/lib/kube-proxy/pki/metrics.crt
  keyFile: /var/lib/kube-proxy/pki/metrics.key
  # optional: clientCAFile for mTLS
healthzServerTLS:
  enable: true
  certFile: /var/lib/kube-proxy/pki/healthz.crt
  keyFile: /var/lib/kube-proxy/pki/healthz.key
```

**Fields to consider** (exact names subject to API review):

- `enable` (bool): when false, HTTP (current behavior).
- `certFile`, `keyFile` (strings): PEM paths; required when `enable` is true.
- Optional: `clientCAFile` for client verification; `cipherSuites`, `minVersion`, `curvePreferences` aligned with existing Kubernetes flags elsewhere.
- Optional future: `certDirectory` / auto-generated self-signed certs for development only (mirrors some binaries); **not** required for MVP if it complicates security posture.

**Validation rules (draft):**

- If `enable` is true, `certFile` and `keyFile` must both be set and readable at startup (or follow dynamic cert loading semantics if chosen).
- TLS must not be silently “half-enabled”; invalid combinations fail `kube-proxy` validation with clear errors.
- Document interaction with **`BindAddressHardFail`**: TLS handshake or cert load failures should respect existing fatal-bind semantics where applicable.

### Implementation Sketch

**Metrics server** (`serveMetrics`):

- After `netutils.MultiListen`, if metrics TLS is enabled, construct `tls.Config` from the new config, attach to `http.Server` via `TLSConfig` **or** wrap listener with `tls.NewListener`, matching patterns used elsewhere in the tree for static file-based certs.
- Keep handler mux unchanged.

**Health server** (`ProxyHealthServer`):

- Extend construction / `Run` to accept optional TLS config; apply to the listener used in `server.Serve(listener)` today.
- Log line should say HTTPS when TLS is enabled (avoid claiming “HTTP server” when it is not).

**Shared code**:

- Prefer consolidating TLS construction in a small helper under `cmd/kube-proxy/app/` or `pkg/proxy` to avoid duplication between metrics and health paths.

**Feature gate** (recommended):

- e.g. `KubeProxyAdminTLS` (name subject to review) — gates new config fields and TLS code paths until beta/GA criteria are met.

**Files likely touched** (non-exhaustive):

- `cmd/kube-proxy/app/server.go` — metrics listener and server setup
- `pkg/proxy/healthcheck/proxy_health.go` — health server
- `pkg/proxy/apis/config/types.go` (+ generated code), `v1alpha1` defaults/validation
- `cmd/kube-proxy/app/options.go` — flags mirroring config if still exposed via CLI
- `cmd/kubeadm` — if kubeadm emits kube-proxy ConfigMaps/static pods, document or optionally wire cert paths when SIG Cluster Lifecycle agrees
- Tests under `cmd/kube-proxy/...`, `pkg/proxy/healthcheck/...`

### Security Model

- **Baseline**: Server TLS provides confidentiality and integrity for admin HTTP traffic; operators supply certs trusted by their scrapers and probe clients.
- **Client authentication**: Optional CA for client certs is desirable for high-security environments; if deferred from alpha, call that out explicitly in the KEP revision.
- **Profiling / debug**: When profiling is enabled, TLS does not remove the sensitivity of those endpoints; RBAC-style protection is out of scope for this KEP—operators should still restrict network access and disable profiling in production where appropriate.

### Interaction with Cluster Lifecycle

- **Static pods / DaemonSet**: Document how to mount secrets or host paths for cert/key into kube-proxy pods.
- **Prometheus**: ServiceMonitor or PodMonitor must use HTTPS and `tls_config` when TLS is enabled; optional bearer token is **not** introduced by this KEP unless extended later.
- **Cloud provider / vendor distros**: Vendors may generate certs using their existing node PKI; keep configuration file-first so vendors can template paths.

### Test Plan

- **Unit tests**: TLS on/off; invalid cert/key; handler still reachable over HTTPS in tests using ephemeral certs.
- **Integration**: Optional—if there is an existing kube-proxy integration harness; otherwise extend `cmd/kube-proxy` tests with `httptest`/`crypto/tls` servers.
- **e2e (optional for alpha)**: Node e2e that starts kube-proxy with TLS-enabled config in a test-only configuration, or conformance-adjacent tests if graduation requires it.

### Graduation Criteria

**Alpha**

- Feature gate disabled by default; TLS configurable for metrics and/or health.
- Documented configuration and known limitations (reload behavior, mTLS if not implemented).

**Beta**

- Sufficient production signal or soak; reload semantics defined and tested; user docs on kubernetes.io.

**GA**

- Feature gate graduation per project policy; no regressions in default (HTTP) behavior; PRR complete.

### Upgrade / Downgrade Strategy

- Upgrading: new fields ignored by old kube-proxy; old manifests work unchanged.
- Enabling TLS: rolling DaemonSet update with coordinated scraper/probe updates, or enable per-node with canaries.
- Downgrading: disable TLS in config before rolling back to a binary that lacks the fields (or keep fields but gate prevents use—document clearly).

### Version Skew Strategy

- Skew between **kube-proxy versions** on nodes is independent; each node’s config controls TLS.
- **API server** is unaffected.
- **Monitoring agents** must understand HTTPS per node when enabled.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

- **How is this feature enabled?** Via `KubeProxyConfiguration` (and optional CLI flags if retained) plus feature gate during alpha/beta.
- **Can it be rolled back?** Yes—set `enable: false` or roll back kube-proxy version per node; restore HTTP probes/scrapes.

### Rollout, Upgrade and Rollback Planning

- **Configurable per listener** reduces blast radius (metrics vs health).
- Document a **canary** procedure: enable TLS on one node, fix scrape/probe config, then expand.

### Monitoring Requirements

- **How can an operator verify it works?** Successful HTTPS scrapes of `/metrics`; kubelet `/livez`/`/healthz` over HTTPS when configured; kube-proxy logs listener mode (HTTP vs HTTPS) at startup.
- **Failure modes**: Misconfigured TLS surfaces as probe failures or scrape errors—document troubleshooting (wrong SAN, expired cert, wrong CA in client).

### Dependencies

- None on new cluster-level services; depends on cert material provisioned by the operator or host integration.

### Scalability

- TLS handshake CPU cost is negligible relative to kube-proxy’s normal work; no additional API server load.

### Troubleshooting

- Common issues: probe still using HTTP, Prometheus missing `tls_config`, hostname/SAN mismatch when scraping by IP.

## Implementation History

- 2026-04-29: KEP drafted (provisional) by authors listed above.

## Drawbacks

- Increased **configuration and operational** burden (cert issuance, rotation, monitoring updates).
- **Two servers** means potentially **two cert pairs** unless operators use the same files for both.

## Alternatives

1. **Status quo**: Rely on network policies, loopback-only binds, and SSH tunnels—rejected for environments that mandate TLS on management ports.
2. **Sidecar TLS terminator**: A per-node sidecar in front of kube-proxy ports adds complexity and PID/network namespace coupling; native TLS is simpler for most deployments.
3. **Merge metrics and health onto one TLS listener**: Desirable long-term (there is an existing TODO in kube-proxy) but larger refactor; could be a follow-up once TLS exists on both listeners.

## Infrastructure Needed

- None beyond standard CI and optional e2e jobs once implemented.

---

## References (codebase)

Current HTTP serving paths this KEP targets include:

- Metrics mux and `http.Server` in `cmd/kube-proxy/app/server.go` (`serveMetrics`).
- Health server in `pkg/proxy/healthcheck/proxy_health.go` (`ProxyHealthServer.Run`).

Configuration types today: `KubeProxyConfiguration` in `pkg/proxy/apis/config/types.go` (`metricsBindAddress`, `healthzBindAddress`, etc.).

For comparison, components such as **kube-scheduler** combine secure serving, metrics, and health on a **TLS** listener with authentication/authorization (`cmd/kube-scheduler/app/server.go`). kube-proxy may adopt a **smaller** subset (TLS only, no full API authz) unless SIG Auth and SIG Network jointly expand scope later.
