# KEP-1898: Hardening exec endpoints against SSRF

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Kubelet API](#kubelet-api)
    - [1. Remove the <code>/run</code> endpoint](#1-remove-the-run-endpoint)
    - [2. Require a <code>POST</code> request to the streaming endpoints](#2-require-a-post-request-to-the-streaming-endpoints)
    - [3. Remove the <code>UID</code> versions of the endpoints](#3-remove-the-uid-versions-of-the-endpoints)
    - [4. Require the request options to be provided in the request body](#4-require-the-request-options-to-be-provided-in-the-request-body)
  - [API Server Changes](#api-server-changes)
    - [1. Require options to be included in the POST request body for exec requests.](#1-require-options-to-be-included-in-the-post-request-body-for-exec-requests)
    - [2. Match new Kubelet request requirements.](#2-match-new-kubelet-request-requirements)
    - [3. Require the websocket protocol for GET requests.](#3-require-the-websocket-protocol-for-get-requests)
  - [Client Changes](#client-changes)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha Criteria](#alpha-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Mitigate the potential for privilege escalations via SSRF vulnerabilities by requiring a POST method for
most exec requests, and limiting control of the executed command via URL parameters. These are
**breaking** changes.

## Motivation

A number of SSRF vulnerabilities have been discovered in Kubernetes recently, and I expect there to
be more. Notable examples include:

- [CVE-2020-8559](http://issues.k8s.io/92914): Client-side redirect
- [CVE-2018-1002102](http://issues.k8s.io/85867): Unvalidated kubelet redirect

In both these cases, the potential of the vulnerability is greatly increased by the ability for an
attacker to execute arbitrary commands specified by a URL parameter in a GET request. Removing this
attacker capability will greatly reduce the impact of future vulnerabilities in which an attacker
can control the URL of an authenticated request.

### Goals

Reduce the potential impact of vulnerabilities in which an attacker can control the URL of an
authenticated request by:

- Eliminating the ability to control an executed command with a URL parameter.
- Limiting the ability to execute commands with a GET method, without breaking websocket connections.

### Non-Goals

- Protecting proxy requests
- Maintaining backwards compatibility of the Kubelet API, which is only intended for consumption by
  the kube-apiserver and only requires forwards-compatibility
- Maintaining long-term backwards compatibility of SSRF-prone exec GET requests

## Proposal

### Kubelet API

The changes to the Kubelet API are:

1. Remove the `/run` endpoint, which is unused in core Kubernetes.
2. Require a `POST` request for the `/exec`, `/attach` and `/portForward` endpoints.
3. Remove the `/{UID}/` versions of the endpoints.
4. Require the exec/attach/port-forward options and pod reference to be provided in the request body.

**Note on backwards compatibility:** This proposal assumes that the only client that should be
talking to these endpoints on the Kubelet is the kube-apiserver. Therefore, while the changes must
support API server version skew, endpoints and request types that are unused by the kube-apiserver
can be removed immediately. The [version skew
policy](https://kubernetes.io/docs/setup/release/version-skew-policy/#kubelet) requires that the
Kubelet not be newer than the kube-apiserver, so backwards compatibility with older apiservers is
not required.

In case there are non-apiserver clients, all the backwards-incompatible Kubelet changes described
below will be guarded by the feature gate `DeprecatedKubeletStreamingAPI`. This feature gate will
start in the default-disabled state and `Deprecated` prerelease channel, and serves to provide a
temporary escape hatch for these API changes.

#### 1. Remove the `/run` endpoint

The `run` endpoint provides the option to run a command in a container with a synchronous
request. This endpoint is not used by any Kubernetes components, and should be removed to reduce the
attack surface.

#### 2. Require a `POST` request to the streaming endpoints

Exec, attach, and port-forward (and run) all respond to either GET or POST requests. Currently the
kube-apiserver uses the same request method when calling the Kubelet as the incoming client request
(e.g. a GET exec request to the apiserver results in a GET exec request to the kubelet). The API
server will need to be updated to always POST to these endpoints, regardless of the incoming client
request.

Since we do not support Kubelets newer than the kube-apiserver, we can safely remove support for GET
requests to these endpoints in the same release that the apiserver moves to unconditional POST.

Although attach & port-forward do not have the same risks of arbitrary code execution as exec, they
share a lot of the same code and should match the exec logic to reduce complexity.

#### 3. Remove the `UID` versions of the endpoints

The exec, attach, and port-forward handlers all support 2 URL path formats:

1. `/exec/{podNamespace}/{podID}/{containerName}`
2. `/exec/{podNamespace}/{podID}/{uid}/{containerName}`

The second path format includes the pod's UID, and is never called by the kube-apiserver. We should
remove these endpoints as they're unused, to reduce code complexity.

#### 4. Require the request options to be provided in the request body

Currently the request options for exec, attach, port-forward and run are provided through the
URL. The pod and container are referenced by the request path, and the command and streaming options
are provided as query parameters. This is the case regardless of whether the request is a GET or
POST.

Instead, the client should be required to provide the options in the request body. Requiring the pod
to be referenced in the body mitigates vulnerabilities that reuse the request body (e.g. a 307
redirect). For backwards compatibility, during the transition period the kube-apiserver will need to
provide the options in both the body and the URL. Newer Kubelets should verify the options match,
and older Kubelets will just ignore the body.

### API Server Changes

The changes in the apiserver are:

1. Require options to be included in the request body for POST exec requests.
2. Match new kubelet request requirements.
3. Require the websocket protocol for GET requests.

#### 1. Require options to be included in the POST request body for exec requests.

The kube-apiserver should read options from the POST request body for exec requests. The pod
reference should be added for the same reasons as described in [4. Require the request options to be
provided in the request body](#4-require-the-request-options-to-be-provided-in-the-request-body).

Initially, this should be optional for backwards compatibility:

- If only body options are provided, use those.
- If only query options are provided, use those.
- If both body and query options are provided, require that they be identical.

Eventually, body options can be required. The requirement will be controlled by the
`HardenedExecRequests` feature gate. The feature will remain in alpha for TBD releases after
supported clients have been updated with the new request format.

<<[UNRESOLVED]>>

OPEN QUESTION: How many releases should we wait before graduating `HardenedExecRequests` to Beta?

<<[/UNRESOLVED]>>

To minimize risk of breakage, attach and port-forward requests should be left unchanged, as they do
not share the same risks of arbitrary code execution as exec.

#### 2. Match new Kubelet request requirements.

The API server must be updated to unconditionally send POST requests to the Kubelet (for
exec/attach/port-forward), and provide the request options in the body. See [2. Require a
<code>POST</code> request to the streaming
endpoints](#2-require-a--request-to-the-streaming-endpoints) and [4. Require the request options to
be provided in the request body](#4-require-the-request-options-to-be-provided-in-the-request-body).

#### 3. Require the websocket protocol for GET requests.

We cannot completely remove support for GET exec requests without breaking websockets. However, we
can require the websocket protocol be used to perform a GET exec. This requirement is guarded by the
`HardenedExecRequests` feature gate.

**Risk:** This is a breaking change for non-websocket clients, and leaves websocket clients exposed
to SSRF risks.

<<[UNRESOLVED]>>

OPEN QUESTION: How common is it for non-websocket clients to use GET for exec requests? `kubectl`
does not. Do we need to do a gradual rollout of this change?

<<[/UNRESOLVED]>>

<<[UNRESOLVED]>>

Alternative options for protecting websockets at the expense of introducing a breaking change:

1. As extra protection against redirect attacks, we could use a similar trick as is used for
providing the bearer token as an extra sub-protocol:
https://github.com/kubernetes/kubernetes/blob/21953d15ea48972f20a8de29d58bd5ce6d913914/staging/src/k8s.io/apiserver/pkg/authentication/request/websocket/protocol.go#L37-L38
In this case, we could take a hash of the request options and provide the hash as an additional
sub-protocol. Then, the exec handler would need to validate that the request options matched the
hash value in the sub-protocol.

2. Approve the exec params with a separate POST request. For example, the client issues a POST to
   `.../exec/token` with the exec params included in the request body. The response includes a token
   that must be included in the GET exec params. Implementation options include caching a single-use
   token (similar to how CRI streaming requests work), or signing the request params.

3. The initial websocket request only opens the websocket protocol stream, and a subsequent request
   must be sent over the websocket to initiate the actual exec action.

<<[/UNRESOLVED]>>

### Client Changes

Kubernetes clients (except websocket clients) will need to be updated to provide the request options
in the POST body before the `HardenedExecRequests` feature can graduate to Beta. Clients must
continue providing parameters as query parameters in addition to the request body for backwards
compatibility with older API servers. See [1. Require options to be included in the POST request
body](#1-require-options-to-be-included-in-the-post-request-body) for the requirements.

### Risks and Mitigations

The proposed changes are **breaking changes**. To mitigate breakage, requiring the new request
parameter format will be protected by the `HardenedExecRequests` feature gate and rolled out
gradually over a long period. In the interim, updated clients can use the new request format for
immediate protection (unless we delay the GET request changes). Requiring POST for non-websocket
exec requests is also a breaking change. See above.

## Design Details

### API

There is already an API type for exec request options:
https://github.com/kubernetes/kubernetes/blob/21953d15ea48972f20a8de29d58bd5ce6d913914/staging/src/k8s.io/api/core/v1/types.go#L4998-L5035

This type will be augmented with a reference to the target pod:

```golang
type PodExecOptions struct {
	metav1.TypeMeta `json:",inline"`

    // Pod is the pod in which the command is to be executed.
    Pod *ObjectReference `json:"pod,omitempty"`

    ...
}
```

### Test Plan

Although `exec` requests are used extensively across E2E tests, the dedicated test coverage is
severely lacking, and there is no test coverage of `attach` and `portForward`. Tests will be added
covering:

1. Parameters provided via query parameters
2. Parameters provided via request body
3. Request via client-go
4. Websocket request (already covered for exec)

For (1) and (2) the request will need to be crafted manually using the go HTTP client. In the case
of exec, (1) will need 2 variants controlled with a `[Feature:HardenedExecRequests]` tag, either
expecting success or failure depending on the status of the `HardenedExecRequests` feature gate.

The tests should be implemented under `test/e2e/common` for inclusion in the `e2e_node` test suites.

### Graduation Criteria

#### Alpha Criteria

- Update `PodExecOptions` with pod reference
- Update Kubelet API (guarded by `DeprecatedKubeletStreamingAPI`)
  - Remove the kubelet's `/run` and UID-specific endpoints
  - Require POST request for kubelet streaming endpoints
  - Require options in request body
- Update kube-apiserver:
  - Always use POST for streaming requests to Kubelet
  - Send options in request body (but also query params)
  - Require POST with request body for non-websocket `exec` requests, guarded by **alpha** `HardenedExecRequests`
- Update go-client to send exec POST requests with options in the body (and also in query params)
- Expand E2E test coverage - https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/1898-hardened-exec#test-plan

#### Alpha -> Beta Graduation

- Clients have been updated for a sufficient amount of time.
- Announcements of breaking changes have been sent out.
- No major ecosystem projects or tools are known to be broken by this.

#### Beta -> GA Graduation

- Sufficient time has passed (amount TBD) for breakages to be resolved.

### Version Skew Strategy

The only version skew risk is between the apiserver and Kubelet. Since the updated apiserver will
send both query & body parameters, both old and new Kubelets will accept the requests.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - Feature gate `HardenedExecRequests`
    - Components depending on the feature gate: kube-apiserver
    - Description: Guards new backwards-incompatible requirements on pod exec requests to the
      kube-apiserver
  - Feature gate: `DeprecatedKubeletStreamingAPI`
    - Components depending on the feature gate: kubelet
    - Description: Enables the unused (by kube-apiserver) kubelet streaming APIs. Default disabled.

* **Does enabling the feature change any default behavior?**
  Yes, enabling `HardenedExecRequests` alters the pod exec API by adding additional constraints.
  Disabling `DeprecatedKubeletStreamingAPI` (default) also removes several request paths from the
  Kubelet API in addition to further constraining several remaining APIs. These APIs are only
  intended for use by the kube-apiserver, and are only required to be forwards-compatible by the
  [Kubernetes version skew
  policy](https://kubernetes.io/docs/setup/release/version-skew-policy/#kubelet).

* **Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?**
  Yes, both features are stateless and can be enabled or disabled without any consequence outside of
  those explicitly controlled by the feature gates.

* **What happens if we reenable the feature if it was previously rolled back?**
  Nothing. See previous question.

* **Are there any tests for feature enablement/disablement?**
  See [Test Plan](#test-plan)

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**

Any request counts for the following metrics indicate that the deprecated Kubelet APIs are in use,
and clients may be broken by disabling `DeprecatedKubeletStreamingAPIs` (`*` indicates any value):

- `kubelet_http_requests_total{long_running=*,method="GET",path="exec",server_type="*"}`
- `kubelet_http_requests_total{long_running=*,method="GET",path="attach",server_type="*"}`
- `kubelet_http_requests_total{long_running=*,method="GET",path="portforward",server_type="*"}`
- `kubelet_http_requests_total{long_running=*,method=*,path="run",server_type="*"}` _Note the method wildcard_

There are no metrics identifying requests missing body parameters, or metrics that break out the UID
sub-paths. These requests, along with reject requests can be identified in the Kubelet's logs.

Unfortunately there are no metrics recording Kubelet response status, so requests that are broken by
disabling `DeprecatedKubeletStreamingAPIs` will need to be detected client-side. See
https://github.com/kubernetes/kubernetes/issues/95307.

On the API server side, GET exec requests can be identified from the audit logs, which can also be
used to identify the client. An increase in 400 Bad Request codes to exec may indicate a breakage by
`HardenedExecRequests`.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At a high level, this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide comprehensive guidance, but at the very
  high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
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


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  No.

* **Will enabling / using this feature result in introducing new API types?**
  No.

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**
  No.

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
  This will expand the `PodExecOptions` API, but this API is not stored.

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  No.

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  No.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**
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

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

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

This proposal includes breaking changes. See [Risks and Mitigations](#risks-and-mitigations).

## Alternatives

- Do nothing - We're likely to see more vulnerabilities related to exec requests in the future, but
  it's possible that we won't.
- Drop support for websockets - If we dropped support for websockets, we could completely eliminate
  exec-via-GET, at the expense of dropping a supported feature. This is probably a non-starter.
- Verify ExecOptions for websockets through a sub-protocol - See [3. Require the websocket protocol
  for GET requests.](#3-require-the-websocket-protocol-for-get-requests).
