<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [X] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [X] **Create an issue in kubernetes/enhancements**  (Issue #4006).
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [X] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [X] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [X] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [X] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [X] **Merge early and iterate.**
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
# KEP-4006: Transition from SPDY to WebSockets

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Background: Streaming Protocol Basics](#background-streaming-protocol-basics)
  - [Background: <code>RemoteCommand</code> Subprotocol](#background--subprotocol)
  - [Background: API Server and Kubelet <code>UpgradeAwareProxy</code>](#background-api-server-and-kubelet-)
  - [Proposal: <code>kubectl</code> WebSocket Executor and Fallback Executor](#proposal--websocket-executor-and-fallback-executor)
  - [Proposal: <code>K8s-Websocket-Protocol: stream-translate</code> Header](#proposal--header)
  - [Proposal: API Server <code>StreamTranslatorProxy</code>](#proposal-api-server-)
  - [Beta: Port Forward Subprotocol](#beta-port-forward-subprotocol)
  - [Pre-GA: Kubelet <code>StreamTranslatorProxy</code>](#pre-ga-kubelet-)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta (RemoteCommand)](#beta-remotecommand)
    - [Beta (PortForward)](#beta-portforward)
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

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
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

Some Kubernetes clients need to communicate with the API Server using a bi-directional
streaming protocol, instead of the standard HTTP request/response mechanism. A streaming
protocol provides the ability to read and write arbitrary data messages between the
client and server, instead of providing a single response to a client request.
For example, the commands `kubectl exec`, `kubectl attach`, and `kubectl port-forward`
all benefit from a bi-directional streaming protocol (`kubectl cp` is build on top
of `kubectl exec` primitives so it utilizes streaming as well). Currently,
the bi-directional streaming solution for these `kubectl` commands is SPDY/3.1. For
the communication leg between `kubectl` and the API Server, this enhancement transitions
the bi-directional streaming protocol to WebSockets from SPDY/3.1.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

The SPDY streaming protocol has been deprecated since 2015, and by now
many proxies, gateways, and load-balancers do not support SPDY. Our effort to modernize
the streaming protocol between Kubernetes clients and the API Server using WebSockets
is necessary to enable the aforementioned intermediaries. WebSockets is a currently
supported standardized protocol (https://www.rfc-editor.org/rfc/rfc6455) that guarantees
compatibility and interoperability with the different components and programming
languages. Finally, WebSockets is preferrable to HTTP/2.0 because the updated HTTP
standard does not support streaming well. The decision to forego HTTP/2.0 is discussed
at greater length in the [Alternatives Section](##Alternatives).

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

1. Transition the bi-directional streaming protocol from SPDY/3.1 to WebSockets for
`kubectl exec`, `kubectl attach`, and `kubectl cp` for the communication leg
between `kubectl` and the API Server.

2. Transition the bi-directional streaming protocol from SPDY/3.1 to WebSockets
for `kubectl port-forward` for the communication leg between `kubectl` and the API
Server (alpha in v1.29).

3. Extend the WebSockets communication leg from the API Server to Kubelet
*before* the current leg goes GA (probably in v1.30). After this extension, WebSockets
streaming will occur between `kubectl` and Kubelet (proxied through the API Server).
This plan is described at [Pre-GA: Kubelet](#pre-ga-kubelet-).

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

1. We will not make *any* changes to current WebSocket based browser/javascript clients.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

Currently, the bi-directional streaming protocols (either SPDY or WebSockets) are 
initiated from clients, proxied by the API Server and Kubelet, and terminated at
the Container Runtime (e.g. containerd or CRI-O). This enhancement proposes to 1)
modify `kubectl` to request a WebSocket based streaming connection, and to 2) modify
the current API Server proxy to translate the `kubectl` WebSockets data stream to
a SPDY upstream connection. In this way, the cluster components upstream from the
API Server will not initially need to be changed. We intend to extend the communication
path for WebSockets streaming from `kubectl` to Kubelet once the the initial leg
is proven to work (i.e. that it goes GA).

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

The functionality of this KEP will allow `kubectl` users to leverage L7 proxies and
gateways that support WebSockets but not SPDY. Usually, the setup for these intermediaries
is specific to a cloud provider or cluster operator. For example, to use the
`Anthos Connect Gateway` to communicate with (some) Google clusters, users must
run `gcloud` specific commands which update the `kubeconfig` file to point to the
gateway. Afterwards, users can run streaming commands such as `kubectl exec ...`,
and the commands will transparently use the now supported WebSockets protocol.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

Initial work on the `PortForward` subprotocol over WebSockets will begin in the
next release (v1.29). While the streaming `kubectl` commands are similar in the
eyes of the users, the streamed data messages are significantly different. Work
on moving the `PortForward` subprotocol to WebSockets from SPDY was started in 2017
with the following [Support websockets from client portforwarding #50428](https://github.com/kubernetes/kubernetes/pull/50428)
PR. But this PR was abandoned and closed when the author realized the significant
scope of the effort. For this reason, we have staggered the development of the subprotocols
by initially prioritizing the `RemoteCommand` subprotocol.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

- TBD

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

**Current SPDY Streaming Architectural Diagram**

![Current SPDY Streaming Architectural Diagram](./spdy-architechtural-diagram.png)

### Background: Streaming Protocol Basics

`kubectl` bi-directional streaming connections are created by upgrading an
initial HTTP/1.1 request. By adding two headers (`Connection: Upgrade`, `Upgrade: SPDY/3.1`),
the request can initiate the streaming upgrade. And when the response returns status
`101 Switching Protocols` signalling success, the connection can then be kept open
for subsequent streaming. An example of an upgraded HTTP Request/Response for `kubectl exec`
could look like:

**HTTP Request**
```
POST /api/v1/…/pods/nginx/exec?command=<CMD>... HTTP/1.1
Connection: Upgrade
Upgrade: SPDY/3.1
X-Stream-Protocol-Version: v4.channel.k8s.io
X-Stream-Protocol-Version: v3.channel.k8s.io
X-Stream-Protocol-Version: v2.channel.k8s.io
X-Stream-Protocol-Version: v1.channel.k8s.io
```

**HTTP Response**
```
HTTP/1.1 101 Switching Protocols
Connection: Upgrade
Upgrade: SPDY/3.1
X-Stream-Protocol-Version: v4.channel.k8s.io
```

If the upgrade is successful, one of the requested subprotocol versions is chosen
and returned in the response. In this instance, the chosen version of the subprotocol
is: `v4.channel.k8s.io`.

### Background: `RemoteCommand` Subprotocol

![Remote Command Subprotocol](./remote-command-subprotocol.png)

Once the connection is upgraded to a bi-directional streaming connection, the
client and server can exchange data messages. These messages are interpreted with
agreed upon standards which are called subprotocols. The three `kubectl` commands
(`exec`, `attach`, and `cp`) communicate using the `RemoteCommand` subprotocol. Basically,
this subprotocol provides command line functionality from the client to a running
container in the cluster. By multiplexing `stdin`, `stdout`, `stderr`, and `tty`
resizing over a streaming connection, this subprotocol supports clients executing
and interacting with commands executed on a container in the cluster. An example of
`kubectl exec` running the `date` command on an `nginx` pod/container is:

```
$ kubectl exec nginx -- date
Tue May 16 03:34:04 PM PDT 2023
```

The `RemoteCommand` Subprotocol has iterated through four different versions, where the
transmitted data has changed. The second version of the subprotocol (`v2.channel.k8s.io`)
included `stdin`, `stdout`, and `stderr`, and the third version (`v3.channel.k8s.io`)
added support for terminal resizing with the `tty` stream. The fourth and most
current version, `v4.channel.k8s.io` added a structured error stream, which includes
the process exit code from the container. All of these streams are multiplexed over
a single connnection by prepending a stream identififer byte to the data message.
For example, a `stdout` data message sent over the connection will have the `stdout`
file descriptor (1) prepended to the data message.

### Background: API Server and Kubelet `UpgradeAwareProxy`

In order to route the data streamed between the client and the container, both the
API Server and Kubelet must proxy these data messages. Both the API Server and the
Kubelet provide this functionality with the `UpgradeAwareProxy`, which is a reverse
proxy that knows how to deal with the connection upgrade handshake.

### Proposal: `kubectl` WebSocket Executor and Fallback Executor

This enhancement proposes adding a `WebSocketExecutor` to `kubectl`, implementing
the WebSocket client using the latest subprotocol version (`v4.channel.k8s.io`).
Additionally, we propose creating a `FallbackExecutor` to address client/server version
skew. The `FallbackExecutor` first attempts to upgrade the connection with the
`WebSocketExecutor`, then falls back to the legacy `SPDYExecutor`, if the upgrade is
unsuccessful. Note that this mechanism can require two request/response trips instead
of one. While the fallback mechanism may require an extra request/response if the
initial upgrade is not successful, we believe this possible extra roundtrip is
justified for the following reasons:

1. The upgrade handshake is implemented in low-level SPDY and WebSocket libraries,
and it is not easily exposed by these libraries. If it is even possible to modify
the upgrade handshake, the added complexity would not be worth the effort.
2. The streaming is already IO heavy, so another roundtrip will not substantially
affect the perceived performance.
3. As releases increment, the probablity of a WebSocket enabled `kubectl` communicating
with an older non-WebSocket enabled API Server decreases.

### Proposal: `K8s-Websocket-Protocol: stream-translate` Header

In addition to the current SPDY-based clients, there are other current WebSocket clients,
including a javascript/browser-based client. In order to distinguish these older
WebSocket clients from the new stream-translated WebSocket clients, we propose adding
a new header `K8s-Websocket-Protocol: stream-translate`. As described further in the
next `Proposal` section, this header allows newer clients to delegate to the
`StreamTranslatorProxy` to translate WebSockets data messages to SPDY.

### Proposal: API Server `StreamTranslatorProxy`

![Stream Translator Proxy](./stream-translator-proxy-2.png)

Currently, the API Server role within client/container streaming is to proxy the
data stream using the `UpgradeAwareProxy`. This enhancement proposes to modify the
SPDY data stream between `kubectl` and the API Server by conditionally adding a
`StreamTranslatorProxy` at the API Server. If the request is for a WebSocket upgrade
with the header `K8s-Websocket-Protocol: stream-translate`, the `UpgradeAwareProxy`
will delegate to the `StreamTranslatorProxy`. This translation proxy terminates the
WebSocket connection, and it de-multiplexes the various streams in order to pass the
data on to a SPDY connection, which continues upstream (to Kubelet and eventually
the container runtime).

### Beta: Port Forward Subprotocol

This KEP addresses only the `RemoteCommand` subprotocol, but the intent is to immediately
follow on with a new `PortForward` subprotocol over WebSockets. Even though the subprotocols
are completely different, in the eyes of the users, the `kubectl` streaming commands (`exec`,
`attach`, `cp`, and `port-forward`) are very similar. Our plan is to go alpha for `RemoteCommand`
in v1.28. For v1.29, we will create an alpha for `PortForward` and go beta for `RemoteCommand`.
For v1.30, we will go beta for `PortForward`, and we will not go GA for either subprotocol
unless both subprotocols are ready for GA.

### Pre-GA: Kubelet `StreamTranslatorProxy`

The eventual plan is to incrementally transition all SPDY communication legs to WebSockets.
After the WebSocket communication leg from `kubectl` to the API Server is proven
to work, the next communication leg to transition is the one from the API Server to
the Kubelet. Both the API Server and the Kubelet stream data messages using the
`UpgradeAwareProxy`. Since the initial plan is to modify the `UpgradeAwareProxy`
in the API Server to delegate to the `StreamTranslatorProxy`, it will be straightforward
to transition this next communication leg by moving the integrated `StreamTranslatorProxy`
from the API Server to the Kubelet. This communication leg will upgraded to WebSockets
*before* the first let goes GA.

The final communication leg to transition from SPDY to WebSockets will be the one
from Kubelet to the Container Runtimes. Since this communication happens within a
node (using Unix domain sockets), this path is not as critical. But this effort
will be more work, since it will require modifying not just Kubelet, but **all**
Container Runtimes.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

The following packages (including current test coverage) will be modified to implement
this SDPY to WebSockets migration.

- `k8s.io/kubernetes/staging/src/k8s.io/client-go/tools/remotecommand`: `2023-05-31` - `57.3%`
- `k8s.io/kubernetes/staging/src/k8s.io/client-go/transport`: `2023-05-31` - `57.7%`
- `k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/util/httpstream`: `2023-05-31` - `76.7%`
- `k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/util/proxy`: `2023-05-31` - `59.1%`
- `k8s.io/kubernetes/staging/src/k8s.io/kubectl/pkg/cmd/attach`: `2023-06-05` - `43.4%`
- `k8s.io/kubernetes/staging/src/k8s.io/kubectl/pkg/cmd/cp`: `2023-06-05` - `66.3%`
- `k8s.io/kubernetes/staging/src/k8s.io/kubectl/pkg/cmd/exec`: `2023-06-05` - `70.0%`
- `k8s.io/kubernetes/staging/src/k8s.io/kubectl/pkg/cmd/portforward`: `2023-06-05` - `76.5%`


##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

- `<test>: <link to test coverage>`

-->

An important integration test for this migration will be a **loopback** test, exercising the
WebSocket client and the StreamTranslator proxy. This test creates two test servers: a
proxy server handling the stream translation, and a SPDY server which sends received data
from one stream (e.g. stdin) back down another stream (e.g. stdout). This test will
send random data from the WebSocket client to the StreamTranslator proxy, which then
sends the data to the test SPDY server.

WebSocket client  <->  Proxy Server (StreamTranslator)  <->  SPDY Server

Once the data is received back at the WebSocket client on the separate stream, it
is compared to the data that was sent to ensure the data is the same. This **loopback**
test has been implemented in a proof-of-concept PR, validating the WebSocket client
and the StreamTranslator proxy.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.

- `<test>: <link to test coverage>`
-->

While there are already numerous current e2e tests for `kubectl exec, cp, attach`,
we will enhance these tests with the permutations of the feature flags for `kubectl`
and the API Server. We will add e2e test coverage for flags and arguments that are
not already covered for these commands.

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

#### Alpha

- `WebSocketExecutor` and `FallbackExecutor` completed and functional behind the `kubectl`
  environment variable KUBECTL_REMOTE_COMMAND_WEBSOCKETS which is **OFF** by default.
- `StreamTranslatorProxy` successfully integrated into the `UpgradeAwareProxy`
  behind an API Server feature flag which is off by default.
- Initial unit tests completed and enabled.
- Initial integration tests completed and enabled.
- Initial e2e tests completed and enabled.

#### Beta (RemoteCommand)

- `WebSocketExecutor` and `FallbackExecutor` completed and functional behind the `kubectl`
  environment variable KUBECTL_REMOTE_COMMAND_WEBSOCKETS which is **ON** by default.
- `StreamTranslatorProxy` successfully integrated into the `UpgradeAwareProxy`
  behind an API Server feature flag which is **on** by default.
- Implement the alpha version of the `PortForward` subprotocol, and surface the new
  `kubectl port-forward` behind a `kubectl` environment variable which is **OFF** by default.
- `PortForwardProxy` successfully integrated into the `UpgradeAwareProxy`
  behind an API Server feature flag which is off by default.
- Additional unit tests completed and enabled.
- Additional integration tests completed and enabled.
- Additional e2e tests completed and enabled.

#### Beta (PortForward)

- Implement the beta version of the `PortForward` subprotocol, and surface the new
  `kubectl port-forward` behind a `kubectl` environment variable which is **ON**
  by default.
- `PortForwardProxy` successfully integrated into the `UpgradeAwareProxy`
  behind an API Server feature flag which is **on** by default.
- Additional unit tests completed and enabled.
- Additional integration tests completed and enabled.
- Additional e2e tests completed and enabled.

#### GA

- Conformance tests for `RemoteCommand` and `PortForward` completed and enabled.
- Conformance tests for `RemoteCommand` and `PortForward` have been stable and
  non-flaky for two weeks.

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

Upgrade requires both the kubectl environment variable and API Server feature flags
to be enabled. Downgrade requires one of the kubectl environment variable **or** API
Server feature flags to be disabled.

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

This feature needs to take into account the following version skew scenarios:

1. A newer WebSockets enabled `kubectl` communicating with an older API Server that
does not support the newer `StreamTranslator` proxy.

In this case, the initial upgrade request for `WebSockets/RemoteCommand` will
fail, and the `FallbackExecutor` will follow up with a legacy upgrade request for
`SDPY/RemoteCommand`. The streaming functionality in this case will work exactly
as it has for the last several years.


2. A legacy non-WebSockets enabled `kubectl` communicating with a newer API Server that
supports the newer `StreamTranslator` proxy.

The legacy `kubectl` will successfully request an upgrade for `SPDY/RemoteCommand - V4`,
just as it has for the last several years.


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

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name(s): KUBECTL_REMOTE_COMMAND_WEBSOCKETS, ClientRemoteCommandWebsockets
  - Components depending on the feature gate: kubectl, API Server

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

Enabling the feature gate on the API Server will allow the streaming mechanism
to be WebSockets instead of SPDY for communication between `kubectl` and the API
Server. The `kubectl` client must also have the KUBECTL_REMOTE_COMMAND_WEBSOCKETS
environment variable set to **ON**, so it will request the newer WebSockets streaming
feature. These modifications, however, will be transparent to the user unless the
`kubectl`/API Server communication is communicating through an intermediary such
as a proxy (which is the whole reason for the feature).

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

The feature can be disabled for a single user by setting the `kubectl` environment
variable associated with the feature to **OFF**. Or the feature can be turned off
for all `kubectl` users communicating with a cluster by turning off the feature flag
for the API Server.

###### What happens if we reenable the feature if it was previously rolled back?

The feature does not depend on state, and can be disabled/enabled at will.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

- There will be unit tests for the `kubectl` environment variable KUBECTL_REMOTE_COMMAND_WEBSOCKETS.
- There will be unit tests in the API Server which exercise the feature gate within
  the `UpgradeAwareProxy`, which conditionally delegates to the `StreamTranslator`
  proxy (depending on the feature gate and the upgrade parameters).

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

- TBD (complete for beta)

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

For highly-available clusters with different versions of API Servers, there
should not be any impact on this feature. The bi-directional streaming protocol
(either SPDY or WebSockets) is only proxied through one instance of the API Server,
which does not change throughout the entirety of the `kubectl` command.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

The most straightforward signal indicating a problem for the feature is failures
for `kubectl exec, cp, attach` commands.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

- TBD (complete for beta)

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

- TBD (complete for beta)

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

- TBD (complete for beta)


###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [X] Other (treat as last resort)
  - Details:
    - `kubectl exec -v=7 <POD|CONTAINER> -- date`
	- One of the request headers will be `K8s-Websocket-Protocol: stream-translate`
	if using the new WebSockets streaming.

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

- [X] Metrics
  - Metric name: TBD (complete for beta)
  - Components exposing the metric: kube-apiserver

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

- TBD (complete for beta)

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

- TBD (complete for beta)

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

No.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

The proposed design envisions a fallback mechanism when a new `kubectl` communicates
with an older API Server. The client will initially request an upgrade to WebSockets,
but it will fallback to the legacy SPDY if it is not supported. In this version
skew scenario where the client implements the new functionality but the server does
not, there is an extra request/response. Since bi-directional streaming already is
very IO intensive, this extra request/response should not be significant. Additionally,
as releases are incremented, the probability of the version skew will continually
decrease.

The newer WebSockets streaming mechanism will also include heartbeat messages,
which will require network IO. But this heartbeat mechanism should contain no
more messages than the current SPDY heartbeat mechanism.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

`kubectl exec, cp, attach` commands spawn container runtime processes, so there is
the danger of node resource exhaustion. This feature, however, does not change the
current legacy mechanism for how these container runtime processes execute or
communicate, except for the communication leg between `kubectl` and the API Server.
There should be no more risk of node resource exhaustion than already exists.

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

- TBD (complete for beta)

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

- TBD (complete for beta)

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

- First Kubernetes release where initial version of KEP available: v1.28

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

The main motivation for taking the risk to change the streaming protocol from SPDY
to WebSockets is to support proxies or gateways in between Kubernetes clients and the
API Server. If we do not believe it is worth it to support these intermediaries
with a modern bi-directional streaming protocol, then we should re-consider this
effort.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

The only currently supported bi-directional streaming protocol is WebSockets.
When HTTP/2.0 was initially proposed, many believed it would provide streaming functionality;
this belief appears to have been misplaced. Ironically, HTTP/2.0 is based on SPDY.
But the upgraded HTTP/2.0 standard did not surface streaming functionality.
For example, HTTP/2.0 specifically does not support `Upgrade` requests to
create a streamable connection. In the golang standard library, HTTP/2.0 requests
with the `Upgrade` header return an error code.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

N/A
