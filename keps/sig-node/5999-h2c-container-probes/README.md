
# KEP-5999: HTTP/2 cleartext (h2c) container probes



<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1 (Optional)](#story-1-optional)
    - [Story 2 (Optional)](#story-2-optional)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Design](#api-design)
    - [Kubelet Probe Execution](#kubelet-probe-execution)
    - [HTTP/2 Cleartext Transport](#http2-cleartext-transport)
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
  - [Option B: Add a dedicated <code>h2cGet</code> probe handler](#option-b-add-a-dedicated-h2cget-probe-handler)
  - [Option C: Extend <code>httpGet</code> with an <code>http2Cleartext</code> boolean](#option-c-extend-httpget-with-an-http2cleartext-boolean)
  - [Option D: Add <code>HTTP2_CLEARTEXT</code> as a new <code>httpGet.scheme</code> value](#option-d-add-http2_cleartext-as-a-new-httpgetscheme-value)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist



Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
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


HTTP/2 cleartext (h2c) is widely used where TLS terminates at the edge while
workloads speak plain HTTP/2 on the pod network (e.g. gRPC). The IETF HTTP/2
spec defines cleartext prior-knowledge connections, and mainstream HTTP
libraries already support h2c. Kubernetes probes should speak that protocol
instead of forcing a separate HTTP/1.1-only listener.

This KEP adds a `protocol` field to the existing `httpGet` probe handler,
allowing operators to select the HTTP version independently of the URI scheme.
When `protocol: HTTP2` is set (with the default `scheme: HTTP`), the kubelet
performs the HTTP GET over HTTP/2 with prior knowledge instead of HTTP/1.1.
Operators are no longer forced to run a second HTTP/1.1-only probe port or
fall back to a `tcpSocket` probe that only confirms the socket is open rather
than a valid HTTP-level response.

## Motivation


H2c is a mature, IETF-standardized protocol supported by all major HTTP
libraries, and it is already widely used in-cluster wherever TLS terminates
outside the pod. That makes it a natural fit for first-class probe support.

Today the kubelet supports `httpGet`, `tcpSocket`, `grpc`, and `exec`, but
none of these can speak HTTP/2 to a cleartext port. Operators must either bolt
on an HTTP/1.1 listener or settle for a `tcpSocket` probe that gives no
application-level health signal.

Adding h2c does introduce an HTTP/2 client dependency, but the cost is
justified: it matches how services already listen, mirrors what `https` probes
already do after TLS, and keeps health checks declarative rather than forcing
workloads into `exec` workarounds.

See the [upstream discussion](https://github.com/kubernetes/kubernetes/issues/125599)
for community experience reports motivating this enhancement.

### Goals


Enable HTTP/2 cleartext (h2c) container probes so apps are not forced to add a
separate HTTP/1.1-only probe port or rely on TCP.

### Non-Goals

- This KEP does not alter gRPC probes or ingress behavior.
- This KEP does not introduce a new top-level probe handler type.

## Proposal


Add a `Protocol *HTTPProtocol` field to the existing `HTTPGetAction`
struct. When set to `HTTP2` (with the default `scheme: HTTP`), the kubelet
performs the HTTP GET over HTTP/2 cleartext with prior knowledge (h2c)
instead of the default HTTP/1.1. When nil or unset, behavior is unchanged
(HTTP/1.1), preserving full backward compatibility. The field is gated by
the `H2CContainerProbe` feature gate on both kube-apiserver (API validation)
and kubelet (probe execution).

This approach reuses the existing `httpGet` path, port, and header semantics
that operators already know, and is naturally extensible to future protocols
(e.g., HTTP/3).


### User Stories (Optional)


#### Story 1 (Optional)

As a platform engineer operating Kubernetes behind a TLS-terminating load
balancer, I want to configure liveness/readiness probes that speak HTTP/2
cleartext (h2c) to my app's main port, so that I don't have to run a second
HTTP/1.1-only port (with extra ingress rules and hardening) just to make probes
succeed.

#### Story 2 (Optional)

As an application developer whose service listens with HTTP/2 without TLS inside
the cluster, I want the kubelet to perform a real HTTP health check
(status code on a path) over h2c, so that I am not forced to use a `tcpSocket`
probe that only proves the port is open and does not confirm a valid HTTP
response.

### Notes/Constraints/Caveats (Optional)

- `protocol: HTTP2` is only valid with `scheme: HTTP` (the default). This
  KEP scopes to cleartext h2c only.
- When `protocol: HTTP2`, `host` must be empty (probe always targets pod IP,
  same rationale as `grpc`). Can be relaxed later if needed.

### Risks and Mitigations

- **Risk:** A user sets `protocol: HTTP2` against a server that only speaks
  HTTP/1.1. The probe will fail repeatedly, causing unnecessary container
  restarts until the misconfiguration is corrected.
- **Mitigation:** Probe failures surface clearly in pod events and
  `prober_probe_total{result="failure"}` metrics. The behavior is identical
  to existing misconfiguration scenarios (e.g., wrong port, wrong path).
  Documentation will emphasize that `protocol: HTTP2` requires the target
  server to accept h2c connections.

## Design Details


### API Design

A new `Protocol *HTTPProtocol` field is added to `HTTPGetAction`. When
set to `HTTP2` (with the default `scheme: HTTP`), the kubelet performs the
HTTP GET over HTTP/2 cleartext with prior knowledge (h2c). When nil,
existing HTTP/1.1 behavior is preserved.

```go
// HTTPProtocol selects the wire protocol for the HTTP probe,
// independently of the URI scheme.
type HTTPProtocol string

const (
    // HTTPProtocolHTTP1 uses HTTP/1.1 (the existing default).
    HTTPProtocolHTTP1  HTTPProtocol = "HTTP1"
    // HTTPProtocolHTTP2 uses HTTP/2.
    // Currently, only cleartext with prior knowledge (h2c) is supported, and must be used with scheme HTTP.
    HTTPProtocolHTTP2 HTTPProtocol = "HTTP2"
)

type HTTPGetAction struct {
    Path        string             `json:"path,omitempty" protobuf:"bytes,1,opt,name=path"`
    Port        intstr.IntOrString `json:"port" protobuf:"bytes,2,opt,name=port"`
    Host        string             `json:"host,omitempty" protobuf:"bytes,3,opt,name=host"`
    Scheme      URIScheme          `json:"scheme,omitempty" protobuf:"bytes,4,opt,name=scheme,casttype=URIScheme"`
    HTTPHeaders []HTTPHeader       `json:"httpHeaders,omitempty" protobuf:"bytes,5,rep,name=httpHeaders"`
    // Protocol selects the wire protocol. Nil defaults to HTTP1.
    // +optional
    // +default="HTTP1"
    Protocol    *HTTPProtocol `json:"protocol,omitempty" protobuf:"bytes,6,opt,name=protocol,casttype=HTTPProtocol"`
}
```

**`ProbeHandler` is unchanged**, the existing `httpGet` field carries the new
`protocol` sub-field.

**Example probe manifest:**

```yaml
readinessProbe:
  httpGet:
    port: 8080
    path: /readyz
    protocol: HTTP2
    httpHeaders:
      - name: Custom-Header
        value: my-value
  initialDelaySeconds: 5
  periodSeconds: 10
```

**Validation rules** (enforced only when `H2CContainerProbe` gate is on):

- `protocol` must be one of `HTTP1`, `HTTP2`, or nil (defaults to `HTTP1`).
- `protocol: HTTP2` requires `scheme: HTTP` (the default). Setting
  `protocol: HTTP2` with `scheme: HTTPS` is rejected during validation.
- When `protocol: HTTP2` is set, `host` must be empty. The probe always
  targets `status.podIP`. This is rejected during validation with a descriptive
  error. `protocol: HTTP1` (or nil) does not restrict `host`.
- When the gate is off, the protocol field is silently dropped during object creation
  (PrepareForCreate). For updates (PrepareForUpdate), the field is dropped unless it was
  already set in the existing object, ensuring existing pods are not corrupted during a
  rollback.

**Kubelet behavior:**

- When executing an `httpGet` probe with `protocol: HTTP2` and the
  `H2CContainerProbe` gate is off, the kubelet ignores the unknown field
  and falls back to HTTP/1.1.
- The h2c client uses HTTP/2 with prior knowledge (no HTTP/1.1 Upgrade
  negotiation), implemented via the standard library's `net/http` transport.
- A 2xx response code is success; any other response or connection error is
  failure, consistent with existing `httpGet` probe semantics.
- Probe timeout and period settings apply identically to other probe types.

#### Kubelet Probe Execution

In `pkg/kubelet/prober/prober.go`, the kubelet reads the `Protocol` field:

```go
useHTTP2 := p.HTTPGet.Protocol != nil && *p.HTTPGet.Protocol == v1.HTTPProtocolHTTP2
```

This boolean is passed to the HTTP prober to select the transport.

#### HTTP/2 Cleartext Transport

In `pkg/probe/http/http.go`, an h2c transport is created alongside the
existing HTTP/1.1 transport:

```go
tr := &http.Transport{}
tr.Protocols = new(http.Protocols)
tr.Protocols.SetUnencryptedHTTP2(true)
client := &http.Client{Transport: tr}
```

`SetUnencryptedHTTP2(true)` enables HTTP/2 over cleartext (h2c) using the
standard library's built-in HTTP/2 support.


### Test Plan

- [x] I/we understand the owners of the involved components may require updates
  to existing tests to make this code solid enough prior to committing the
  changes necessary to implement this enhancement.

##### Prerequisite testing updates


No prerequisite test updates are expected.

##### Unit tests

- `pkg/probe/http` (existing package, extended):
  - Build GET URL and headers correctly from `HTTPGetAction` with
    `protocol: HTTP2`.
  - Probe a local h2c test server: 200 -> success, 500 -> failure, timeout ->
    failure, connection refused -> failure.
  - Verify HTTP/2 is actually used on the wire (not HTTP/1.1 downgrade).
  - `protocol: nil` -> unchanged HTTP/1.1 behavior.

- `pkg/apis/core/validation`:
  - Valid `httpGet` with `protocol: HTTP2`, numeric port, path, and headers ->
    allowed.
  - `protocol: HTTP2` + `scheme: HTTPS` -> rejected during validation.
  - `protocol: HTTP2` + `host` set -> rejected (host override not allowed
    with HTTP2).
  - `H2CContainerProbe` gate off -> `protocol` field silently dropped.
  - `protocol: nil` + `host` set → allowed (existing behavior unchanged).

- `pkg/kubelet/prober`:
  - Gate off + `protocol: HTTP2` set -> kubelet ignores the field, falls back
    to HTTP/1.1.
  - Gate on + `protocol: HTTP2` -> kubelet uses h2c transport for the probe.
  - Gate on/off toggle: object admitted with gate on, gate turned off ->
    kubelet ignores the field and falls back to HTTP/1.1.


- `pkg/probe/http`: `2026-05-26` - `78.7%`
- `pkg/kubelet/prober`: `2026-05-26` - `79.8%`
- `pkg/apis/core/validation`: `2026-05-26` - `85.3%`

##### Integration tests

- Create a pod with `protocol: HTTP2` when `H2CContainerProbe` gate is on ->
  field is accepted and persisted.
- Create a pod with `protocol: HTTP2` when `H2CContainerProbe` gate is off ->
  field is silently dropped, pod is created without it.
- Create a pod with `protocol: HTTP2` + `scheme: HTTPS` -> rejected during
  validation.
- Create a pod with `protocol: HTTP2` + `host` set -> rejected during
  validation.
- Update a pod that has `protocol: HTTP2` when gate is on -> field is
  preserved.
- Update a pod that has `protocol: HTTP2` when gate is off -> field is
  preserved on existing pod (backward compatibility).

##### e2e tests

- Liveness probe with `protocol: HTTP2` against an h2c server succeeds ->
  container is **not** restarted (happy path, using agnhost `h2c-server`).
- Liveness probe with `protocol: HTTP1` against an HTTP/1.1 server succeeds ->
  container is **not** restarted.
- Liveness probe with `protocol: HTTP2` against an HTTP/1.1-only server fails
  -> container **is** restarted (protocol mismatch).
- Liveness probe with `protocol: HTTP2` targeting a wrong port fails ->
  container **is** restarted (connection refused).



### Graduation Criteria


#### Alpha

- API field implemented and functional
- Unit and integration tests passing.
- Documentation available

#### Beta

- No major bugs reported during alpha
- Gather feedback from users

#### GA

- Stable for at least two releases
- No major issues reported

### Upgrade / Downgrade Strategy


**Upgrade:** Opt-in only. Existing pods are unaffected (`protocol` defaults to
nil = HTTP/1.1). Enable the `H2CContainerProbe` gate on both kube-apiserver
and kubelet to use the feature.

**Downgrade:** On downgrade to a version that does not know the `protocol`
field, the field is silently ignored by both the apiserver and kubelet. The
kubelet falls back to HTTP/1.1 for all probes. Pods whose servers only
accept HTTP/2 will experience probe failures, which is the same behavior
that existed before this feature. No manual pod spec changes are required
to roll the cluster back.

### Version Skew Strategy


- **New apiserver (gate on), old kubelet:** Pod is admitted but the old kubelet
  ignores `protocol` and probes with HTTP/1.1. This is the expected backward
  compatible behavior, the feature behaves as if it did not exist.
- **New apiserver (gate off):** `protocol` is silently dropped during validation. No pods
  with the field reach any kubelet.
- **Both new, gate on apiserver / gate off kubelet:** Kubelet ignores the
  `protocol` field and falls back to HTTP/1.1.
- **Both new, gate on everywhere:** Fully operational.

**Recommended:** Upgrade all components first (following the
[standard cluster upgrade order](https://kubernetes.io/docs/tasks/administer-cluster/cluster-upgrade/)),
then enable the feature gate.

## Production Readiness Review Questionnaire


### Feature Enablement and Rollback



###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `H2CContainerProbe`
  - Components depending on the feature gate:
    - `kube-apiserver` (validation, field dropping)
    - `kubelet` (probe execution)


###### Does enabling the feature change any default behavior?

No. The `protocol` field defaults to nil, which preserves existing HTTP/1.1
probe behavior. Only pods that explicitly set `protocol: HTTP2` are affected.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the gate on the apiserver prevents new pods from setting the
`protocol` field. The kubelet silently ignores the unknown field and falls
back to HTTP/1.1. Pods whose servers only accept HTTP/2 will experience
probe failures, which is the pre-feature behavior. No manual pod spec
changes are required.

###### What happens if we reenable the feature if it was previously rolled back?

Pods that still carry `protocol: HTTP2` in their spec (admitted before rollback)
will resume h2c probing automatically. No data is lost or corrupted; the field
is purely declarative.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests in `pkg/apis/core/validation` and `pkg/kubelet/prober` cover:
- Gate off: `protocol` field silently dropped from new pods, preserved on
  existing pods for backward compatibility.
- Gate on then off: kubelet ignores the `protocol` field and falls back to
  HTTP/1.1.



### Rollout, Upgrade and Rollback Planning



###### How can a rollout or rollback fail? Can it impact already running workloads?



###### What specific metrics should inform a rollback?

A spike in `prober_probe_total{result="failure"}` for pods using HTTP probes
after enabling the gate may indicate misconfigured h2c probes and could
warrant a rollback.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?




### Monitoring Requirements


###### How can an operator determine if the feature is in use by workloads?

Operators can monitor the existing `prober_probe_total` metric. An increase in
HTTP probe executions after enabling the gate, combined with pods whose specs
set the `protocol` field, indicates the feature is in use. A dedicated label
(e.g., `protocol="HTTP2"`) on `prober_probe_total` could be added in beta to
make this easier to observe.

###### How can someone using this feature know that it is working for their instance?



###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?



###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?


- [x] Metrics
  - Metric name: `prober_probe_total`
  - Components exposing the metric: kubelet

###### Are there any missing metrics that would be useful to have to improve observability of this feature?



### Dependencies


###### Does this feature depend on any specific services running in the cluster?


No external cluster services are required.

### Scalability


###### Will enabling / using this feature result in any new API calls?

No. This feature doesn't introduce any new API calls. The kubelet already makes probe requests directly to the pod (not through the API server). 

###### Will enabling / using this feature result in introducing new API types?

Yes, `HTTPProtocol` (a string enum type) and a new `Protocol` field on
`HTTPGetAction`. Not a standalone API resource.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Negligible increase. The protocol field adds a small optional string ("HTTP2") to the HTTPGetAction struct inside the Pod spec.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. Probe execution time is determined by the target container's response time, not the wire protocol.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The h2c transport is lightweight, it is a standard http.Transport with SetUnencryptedHTTP2(true). It uses the same connection pattern as existing HTTP/1.1 probes.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. The feature uses the same one-connection-per-probe model as existing HTTP probes. 

### Troubleshooting



###### How does this feature react if the API server and/or etcd is unavailable?



###### What are other known failure modes?



###### What steps should be taken if SLOs are not being met to determine the problem?


## Implementation History

- 2026-04-07: KEP created


## Drawbacks



No significant drawbacks beyond the added complexity noted in Risks and Mitigations.

## Alternatives



### Option B: Add a dedicated `h2cGet` probe handler

This approach adds a new `h2cGet` field to `ProbeHandler`, modeled after the
`grpc` probe type: numeric-only port, no host override, and a fixed h2c
protocol.

```go
type H2CGetAction struct {
    Port        int32        `json:"port" protobuf:"varint,1,opt,name=port"`
    Path        string       `json:"path,omitempty" protobuf:"bytes,2,opt,name=path"`
    HTTPHeaders []HTTPHeader `json:"httpHeaders,omitempty" protobuf:"bytes,3,rep,name=httpHeaders"`
}

type ProbeHandler struct {
    Exec      *ExecAction      `json:"exec,omitempty" ...`
    HTTPGet   *HTTPGetAction   `json:"httpGet,omitempty" ...`
    TCPSocket *TCPSocketAction `json:"tcpSocket,omitempty" ...`
    GRPC      *GRPCAction      `json:"grpc,omitempty" ...`
    H2CGet    *H2CGetAction    `json:"h2cGet,omitempty" ...`
}
```

```yaml
readinessProbe:
  h2cGet:
    port: 8080
    path: /readyz
```

**Why this approach was not adopted:**

1. h2c is just a transport variant, not a distinct protocol like gRPC. A
   dedicated handler sets a precedent for every future transport option
   (HTTP/3, TLS-without-ALPN, etc.) to need its own handler.
2. It duplicates most `httpGet` semantics into a parallel struct, increasing
   maintenance burden.
3. Users must learn a new handler and rewrite probes; the `protocol` field
   lets them keep existing `httpGet` probes and add one field.
4. Any validation constraints (e.g. disallowing named ports) can be enforced
   on the `protocol` field instead.

### Option C: Extend `httpGet` with an `http2Cleartext` boolean

This approach would add an optional `http2Cleartext *bool` field to the existing
`HTTPGetAction` struct:

```go
type HTTPGetAction struct {
    Path           string             `json:"path,omitempty" ...`
    Port           intstr.IntOrString `json:"port" ...`
    Host           string             `json:"host,omitempty" ...`
    Scheme         URIScheme          `json:"scheme,omitempty" ...`
    HTTPHeaders    []HTTPHeader       `json:"httpHeaders,omitempty" ...`
    // +optional
    HTTP2Cleartext *bool              `json:"http2Cleartext,omitempty" ...`
}
```

**Why this approach was not adopted:**

1. A boolean is not extensible — future transport variants (e.g. HTTP/3)
   would require additional booleans and combinatorial validation.
2. The `protocol` enum handles this and future variants in a single field.

### Option D: Add `HTTP2_CLEARTEXT` as a new `httpGet.scheme` value

Instead of a new field, the existing `scheme` field in `HTTPGetAction` could
gain a third enum value:

```yaml
readinessProbe:
  httpGet:
    scheme: HTTP2_CLEARTEXT
    port: 8080
    path: /readyz
```

**Why this approach was not adopted:**

1. `scheme` maps to URI schemes (`HTTP`, `HTTPS`), not transport encodings —
   adding a wire-level value like `HTTP2_CLEARTEXT` creates a semantic mismatch.
2. Probe URL construction derives the prefix from `scheme`; a transport-only
   variant that doesn't change the URL complicates that code path.
3. Combinations like `scheme: HTTP2_CLEARTEXT` with `host` overrides or named
   ports are undefined, expanding the validation surface.


## Infrastructure Needed (Optional)



No new infrastructure is needed.
