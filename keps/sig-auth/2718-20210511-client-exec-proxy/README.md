<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-2718: Client Executable Proxy

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Proxy Proposal](#proxy-proposal)
  - [Authentication to Proxy](#authentication-to-proxy)
  - [Caching](#caching)
  - [Proxy Shutdown](#proxy-shutdown)
  - [API](#api)
  - [Test Plan](#test-plan)
    - [Integration](#integration)
    - [E2E](#e2e)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
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
  - [Alternative Proposal: Request Replacement](#alternative-proposal-request-replacement)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

Client authentication in Kubernetes can be achieved via one of a handful of
mechanisms: by passing a bearer token directly to kubectl using the `--token`
flag, by inserting a bearer token into the kubeconfig, by inserting a client
certificate and key into the kubeconfig, or by defining an exec-based token
plugin in the kubeconfig which supports calling an arbitrary binary which is
expected to return a bearer token and/or client certificate/key.  The client
exec proxy adds an extension point for additional authentication mechanisms
that are difficult to set up with the current architecture.  Client request
signing, and an external TLS certificate authenticator are two use cases that
would benefit from this proposal.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

The motivation for this KEP stems from the inability of the existing
authentication mechanisms to satisfy the author's requirement to implement a
Kubernetes client that implements AWS's request signing process, known as
Signature Version 4 (sig-v4)
[[1]](https://docs.aws.amazon.com/general/latest/gr/signature-version-4.html)
[[2]](https://github.com/kubernetes/kubernetes/issues/92535). In order to
implement sig-v4, a canonical version of the request must be created, combined
with any required additional metadata, signed by the secret key, and the
signature appended to the request in a header or query string parameter.  In
order to achieve this, the request signer must have access to the entire
request, so the existing exec-based authentication is unsatisfactory.  A
generic solution is desirable to satisfy other future request signing or
request modifying feature requests.

The solution that exists today is to use the existing proxy configuration in
client-go.  The includes configuring a proxy via the environment variables,
HTTP_PROXY or HTTPS_PROXY, or using the explicit proxy URL supported by
kubeconfig [[3]](https://github.com/kubernetes/kubernetes/pull/81443).  The
problem with this solution is that the operator experience of setting up and
managing an external proxy is poor.  For example, to implement a request
signing proxy for kubelet, one has to manage the proxy process with an init
system of some kind, and secure the proxy endpoint, both of which require
significant, non-trivial configuration outside of Kubernetes.

Another use case is using an external TLS certificate signer
[[4]](https://github.com/kubernetes/enhancements/pull/1749).  This would allow
usage of TLS client certificates within client-go, by delegating digital key
operations to external processes, for example, an HSM.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

The goals of this KEP include:
1. Defining a specification which enables Kubernetes clients to do arbitrary
request modification before sending the request to the API server, via an
extensible mechanism to support the following use cases:
    - Client request signing.
    - Using hardware protected keys for request signing or mTLS.
    - Adding or manipulating arbitrary headers to the request.
2. Defining a specification which would provide the ability for the client-side
proxy server to be executed and communicated with according to configuration
solely contained in a kubeconfig file.
3. Defining a specification which is interoperable with an existing client-side
proxy setup, e.g. a proxy configured with the HTTP_PROXY environment variable.
4. Defining a specification for a client-side proxy server that is easy to
secure without taking additional setup steps, by:
    - Using a unix domain socket for the communication between the client and
      the proxy, secured with file permissions and TLS.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

The following are non-goals of this KEP:
1. Solving use cases outside of the above-mentioned client authentication
   workflows.
2. Implementing a specific flavor of request signing (like SigV4), or adding
   support for any hardware security module protocol directly in the Kubernetes
   client libraries.
3. Defining any new mechanisms for validating request signatures server-side.
   This is because they can be validated by a front-proxy, which is already a
   feature of Kubernetes.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

This proposal is to implement an extension in Kubernetes client libraries to
forward requests to a proxy process for request modification.  The proxy will
receive an HTTP request and will have the opportunity to sign the request,
attach headers, use hardware protected keys for mTLS, and/or modify the
request.  The final request will be constructed and forwarded to the apiserver.

It is already possible, by setting the appropriate HTTP_PROXY, HTTPS_PROXY
environment variables, to configure a Kubernetes client to send requests to a
proxy.  While this may work in some situations, it is not ideal for a generic
request signing solution across all client configurations.  Firstly, http
proxies are already commonly integrated in client networking configurations, so
for these clients setting up an additional proxy configuration might not be
possible, or at least not straightforward.  Secondly, this approach requires a
long running external proxy process which introduces operational complexity.
Thirdly, communicating with a proxy for request signing requires a secure
communication channel between the client and the proxy, which means in a case
where multiple users share a client computer (for example, a shared bastion),
configuring and securing a local proxy to prevent signing requests as a
different user becomes more complicated.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

The risks of this proposal are similar to the existing exec credential
provider.  There are risks associated with execing a binary, for example in a
scenario where users are downloading binaries from untrustworthy sources to use
in their cluster.  However, this is not a departure from existing security
posture of the exec credential feature present in k8s.io/client-go today.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Proxy Proposal

The proxy exec plugin is a long-running proxy process which is executed by the
client on first use.  When executed, the exec proxy plugin will bind and listen
on a socket, and will a output URL on stdout, over which the client would proxy
all API server bound requests.  The proxy binary location is defined in the
`ExecConfig` section of a kubeconfig `user`.

The proxy plugin process will be executed from client-go the first time it is
needed, on transport setup.  If the client fails to connect to the socket
because no one is listening, client-go will reattempt to exec the plugin.
Client-go will support both plugins binaries which are themselves the proxy
process, and additionally plugin binaries which complete, but are responsible
for starting a proxy process and returning a URL to the proxy.

The client will send its request to the proxy plugin process, and the proxy
will forward the request to the API server, rather than returning the modified
request to the client to be sent (see alternatives considered).  The downside
of this approach is that the proxy must duplicate the some of the transport
functionality of client packages, like respecting existing proxy configuration,
but allows for more flexibility for proxy plugin developers.

### Authentication to Proxy

Options:

1. The client will authenticate to its proxy using mTLS.  On exec, the proxy
   will pass back client key and certificate authority data for the client to
   use for authentication when establishing a connection.
2. An alternative method for authentication to the proxy would be file
   permission based.  The GID owner of the client process could passed to the
   proxy over stdin, which the proxy should give read/write permissions on the
   socket to.  However, this could be less portable if we need to use a non UDS
   communication mechanism for windows, or if we support TCP/IP in the future.
3. A third mechanism for authentication for windows based systems would be
   using named pipes, which support the Win32 access control model.

An option will be chosen during implemenation of alpha.

### Caching

The connection from the client to the proxy can be cached in the client based
on the cluster details, the existing HTTP_PROXY configuration, the exec-proxy
URL, and the certificates used for mTLS.

### Proxy Shutdown

In the case where the client dies unexpectedly, due to a crash or SIGKILL, the
proxy should also die.  To handle this, the proxy will attempt to take a file
lock that the parent holds and quit if it ever successfully takes the lock.
Before exec, the client should take the lock.  The lock file path will be
provided to the proxy in the ExecCredentialSpec.

Alterntatively, the proxy could quit after a certain amount of idle time.  For
example, proxies that are idle for more than 5 minutes will be required to
quit.  Idle quit is simpler, but the lock approach handles the kubectl use case
better, so it is preferred.

### API

The communication over stdin and stdout with the exec'd binary will be done
with using the existing ExecCredentialSpec and ExecCredentialStatus objects.

```go
// ExecCredentialSpec holds request and runtime specific information provided by
// the transport.
type ExecCredentialSpec struct {
	// Cluster contains information to allow an exec plugin to communicate with the
	// kubernetes cluster being authenticated to. Note that Cluster is non-nil only
	// when provideClusterInfo is set to true in the exec provider config (i.e.,
	// ExecConfig.ProvideClusterInfo).
	// +optional
	Cluster *Cluster `json:"cluster,omitempty"`

	// Interactive declares whether stdin has been passed to this exec plugin.
	Interactive bool `json:"interactive"`

	// New:

	// LockFilePath specifies a lock file to be taken by the client
	// +optional
	LockFilePath string `json:"lockFilePath,omitempty"`
}

// ExecCredentialStatus holds credentials for the transport to use.
//
// Token and ClientKeyData are sensitive fields. This data should only be
// transmitted in-memory between client and exec plugin process. The Exec plugin
// itself should at least be protected via file permissions.
type ExecCredentialStatus struct {
	// ExpirationTimestamp indicates a time when the provided credentials expire.
	// +optional
	ExpirationTimestamp *metav1.Time `json:"expirationTimestamp,omitempty"`
	// Token is a bearer token used by the client for request authentication.
	Token string `json:"token,omitempty" datapolicy:"token"`
	// PEM-encoded client TLS certificates (including intermediates, if any).
	ClientCertificateData string `json:"clientCertificateData,omitempty"`
	// PEM-encoded private key for the above certificate.
	ClientKeyData string `json:"clientKeyData,omitempty" datapolicy:"security-key"`

	// New:

	// ProxyConfig is the config returned by the exec-proxy
	// +optional
	ProxyConfig *ProxyConfig `json:"proxyConfig,omitempty"`
}


// ProxyConfig configures the client to proxy requests via the returned address.
type ProxyConfig struct {
	// URL is the location of the exec proxy listener.
	URL string `json: "url"`

	CertificateAuthorityData []byte `json:"certificate-authority-data"`
}
```

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy. Anything
that would count as tricky in the implementation, and anything particularly
challenging to test, should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing
guidelines][testing-guidelines] when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

Both integration and e2e tests will be required in addition to unit tests.

#### Integration

Integration tests will cover:

* Authentication between client and proxy.
* Connection from client to apiserver through proxy.
* Proxy use with kubectl (possible test location: [command-line integration
  test
  suite](https://github.com/kubernetes/kubernetes/tree/ec560b9737537be8c688776461bc700e8ddedb9d/test/cmd)).
* UDS connections from the client to the proxy will be tested.

#### E2E

The e2e tests will test communication from client to apiserver through proxy in
a real cluster. The e2e tests will require a proxy implementation and
apiserver, etcd, and a client.

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Removing a Deprecated Flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the
  flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
-->

#### Alpha

- Implementation of UDS connections to proxy.

#### Alpha -> Beta Graduation

- Implementation of UDS (or equivalent) on windows.
- Full reference proxy implementation with TLS offload.  This will be built using code from client-go that provides a harness to offload to a Go `crypto.Signer`.
- As part of client-go, it needs to work on macOS, windows and linux.
- Tests that show SPDY based APIs like pods/exec still work.
- Tests that exercise webhooks from API server because they share the same auth code.
- Gather feedback from developers, SIG Auth, SIG CLI.
- Tests are in Testgrid and linked in KEP.

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

For clients which are not currently using an exec plugin, a modification to the
kubeconfig must be made and a proxy binary must be located as specified in the
kubeconfig.  For clients already using an exec plugin without proxy support,
the exec plugin would need to upgraded. For clients moving from one version of
the proxy to another, the proxy binary must be upgraded.

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

The proxy binary may be required to be updated before the client is updated in
situations where the API over stdio between them changes.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness/README.md.

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

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: ClientExecProxy
    - Components depending on the feature gate: kubelet,
      kube-controller-manager, kube-scheduler (for any that need to use the
      proxy--this will be environment dependent).
    - For kubectl, an env var will be used to enable when its an alpha feature.
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**
  Any change of default behavior may be surprising to users or break existing
  automations, so be extremely careful here.

  No.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Also set `disable-supported` to `true` or `false` in `kep.yaml`.  Describe
  the consequences on existing workloads (e.g., if this is a runtime feature,
  can it break the existing applications?).

  Yes, the feature can be disabled.  In the case where it is being used by long
  running clients to talk to the API server, the client would have to be
  reconfigured to stop using the proxy for connections.  Depending on how the
  client is configured, this may require a restart.

* **What happens if we reenable the feature if it was previously rolled back?**

  The feature can be reenabled after being disabled without consequence.

* **Are there any tests for feature enablement/disablement?**
  The e2e framework does not currently support enabling or disabling feature
  gates. However, unit tests in each component dealing with managing data, created
  with and without the feature, are necessary. At the very least, think about
  conversion tests if API types are being modified.

  No.

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
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
  - [ ] Metrics
    - Metric name: `rest_client_exec_plugin_with_proxy_call_total`
    - Components exposing the metric: client-go (kube-controller-manager, kube-scheduler, kubelet, kube-apiserver)
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
  Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
  focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)

* **Will enabling / using this feature result in introducing new API types?**
  Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)

* **Will enabling / using this feature result in any new calls to the cloud
provider?**

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**
  Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data sent and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits].

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

The proxy's behavior when the API server is unavailable will be up to the proxy implementor.

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

- 2021-05-07 KEP PR created, first reviews received.
- 2022-01-27 PR updated, more feedback received.
- 2022-06-22 PR updated to use ExecCredentialSpec/ExecCredentialStatus.

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

It introduces a new feature into client libraries which comes with a
maintenance burden. It also introduces a new extension point which might make
client installs which choose to use this feature slightly more complicated, in
that the user must install an additional binary.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

### Alternative Proposal: Request Replacement

In this proposal, the the proxy would be replaced by a localhost/UDS server.
The client would package the HTTP request into the body of a GRPC or HTTP
request and send to the server, expecting a response back immediately with the
modified request as a payload.  The client would then extract the request,
replace the original request with the modified request, and send it as it
normally would.  This would be done in the client transport.  Otherwise, the
proposal is largely the same--the client uses configuration in the kubeconfig
to start the signing process, and the client receives a unix domain socket or
localhost address from the server process on exec.  The server process listens
at the unix domain socket address, and the client connects to it.  The client
attempts to exec the server binary if it is unable to connect using the
provided address.

This proposal was not preferred to the chosen proposal because, after
discussions at SIG-Auth, the overall feeling was that defining the request
response protocol between the client and the signing process would be more
cumbersome, and be less suited to a generic solution that accommodated any
proxy related use case.  It is also not possible to implement TLS offload with
this approach.

