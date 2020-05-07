<!--
**Note:** When your KEP is complete, all of these comment blocks should be
removed.

To get started with this template:

- [ ] **Pick a hosting SIG.** Make sure that the problem space is something the
  SIG is interested in taking up.  KEPs should not be checked in without a
  sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements** When filing an enhancement
  tracking issue, please ensure to complete all fields in that template.  One of
  the fields asks for a link to the KEP.  You can leave that blank until this
  KEP is filed, and then go back to the enhancement and add the link.
- [ ] **Make a copy of this template directory.** Copy this template into the
  owning SIG's directory and name it `NNNN-short-descriptive-title`, where
  `NNNN` is the issue number (with no leading-zero padding) assigned to your
  enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.** At minimum, you
  should fill in the "title", "authors", "owning-sig", "status", and
  date-related fields.
- [ ] **Fill out this file as best you can.** At minimum, you should fill in the
  "Summary", and "Motivation" sections. These should be easy if you've
  preflighted the idea of the KEP with the appropriate SIG(s).
- [ ] **Create a PR for this KEP.** Assign it to people in the SIG that are
  sponsoring this process.
- [ ] **Merge early and iterate.** Avoid getting hung up on specific details and
  instead aim to get the goals of the KEP clarified and merged quickly.  The
  best way to do this is to just start with the high-level sections and fill out
  details incrementally in subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved.  Any KEP
marked as a `provisional` is a working document and subject to change.  You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused.  If you disagree with what is already in a document, open a new PR with
suggested changes.

One KEP corresponds to one "feature" or "enhancement", for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example.  If there are
new details that belong in the KEP, edit the KEP.  Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source of
this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable` or significant changes once
it is marked `implementable` must be approved by each of the KEP approvers. If
any of those approvers is no longer appropriate than changes to that list should
be approved by the remaining approvers and/or the owning SIG (or SIG
Architecture for cross cutting KEPs).
-->

# KEP-20200507: External TLS certificate authenticator

<!--
This is the title of your KEP.  Keep it short, simple, and descriptive.  A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with <code>&lt;!-- toc --&rt;&lt;!-- /toc
  --&rt;</code> tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Configuration](#configuration)
  - [API specs](#api-specs)
    - [Obtaining a certificate](#obtaining-a-certificate)
      - [Certificate request](#certificate-request)
      - [Certificate response](#certificate-response)
    - [Signing](#signing)
      - [Sign request](#sign-request)
      - [Sign response](#sign-response)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [External signer vs existing authenticators using TLS
    certificates](#external-signer-vs-existing-authenticators-using-tls-certificates)
  - [Monolithic vs modular architecture](#monolithic-vs-modular-architecture)
  - [RPC vs exec](#rpc-vs-exec)
  - [Independent external plugin configuration vs passing configuration
    parameters from
    kubectl/client-go](#independent-external-plugin-configuration-vs-passing-configuration-parameters-from-kubectlclient-go)
  - [Stdin vs program arguments vs environment
    variables](#stdin-vs-program-arguments-vs-environment-variables)
  - [Multiple key-value pairs vs a single JSON
    string](#multiple-key-value-pairs-vs-a-single-json-string)
- [Infrastructure Needed (optional)](#infrastructure-needed-optional)
  <!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement
Freeze](https://git.k8s.io/sig-release/releases) of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

- [ ] Enhancement issue in release milestone, which links to KEP dir in
  [kubernetes/enhancements] (not the initial KEP PR)
- [ ] KEP approvers have approved the KEP status as `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG
  Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for
  publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to
  mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every
time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/ [kubernetes/enhancements]:
https://git.k8s.io/enhancements [kubernetes/kubernetes]:
https://git.k8s.io/kubernetes [kubernetes/website]: https://git.k8s.io/website

## Summary

<!--
This section is incredibly important for producing high quality user-focused
documentation such as release notes or a development roadmap.  It should be
possible to collect this information before implementation begins in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself.  KEP editors, SIG Docs, and SIG PM
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.
-->

This enhancement proposes adding support for authentication via external TLS
certificate signers, what would enable usage of Hardware Security Modules (HSMs)
- also known as smartcards, cryptographic processors or, by a popular brand
name, YubiKeys(tm) via the PKCS#11 standard. This enhancement allows developers
or automation pipelines to authenticate with the Kubernetes cluster, without
requiring access to the client key, hence improving compliance and security.

## Motivation

<!--
This section is for explicitly listing the motivation, goals and non-goals of
this KEP.  Describe why the change is important and the benefits to users.  The
motivation section can optionally provide links to [experience reports][] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

<!-- How would you react if your laptop was stolen? Are you worried about 
attackers performing a cold boot attack to extract your Kubernetes 
credentials?  -->
Are you worried about someone getting access to your Kubernetes credentials?
(For example, by extracting private keys from your laptop using malware or by
performing a cold boot attack.) Do you already use a YubiKey for SSH and GPG,
and wonder why you cannot use it with kubectl? If yes, then this enhancement is
for you!

Highly regulated environments, such as FinTech, require delegating all digital
key operations to specialized [Hardware Security Modules
(HSMs)](https://en.wikipedia.org/wiki/Hardware_security_module). Amongst others,
HSMs increase security by storing digital keys without allow them to be
extracted. Authentication, encryption and signing is performed via a standard
such as the PKCS#11 on the HSM directly. In fact, many regulated environments
already require developers and operators to store SSH and GPG keys on the
[YubiKey](https://en.wikipedia.org/wiki/YubiKey), a popular HSM connected via
USB.

Unfortunately, as of today, kubectl lacks support for PKCS#11 (see [Issue
#64783](https://github.com/kubernetes/kubernetes/issues/64783)). Indeed, kubectl
requires direct access to the client key data, which can either be stored in the
kubeconfig or provided via a [credentials
plugin](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#client-go-credential-plugins).
Because of this, some security bloggers have even argued for [not using
certificates](https://www.tremolosecurity.com/kubernetes-dont-use-certificates-for-authentication/)
in kubectl at all.

### Goals

<!--
List the specific goals of the KEP.  What is it trying to achieve?  How will we
know that this has succeeded?
-->

- kubectl can authenticate to a Kubernetes cluster with an external TLS
  certificate signer, for example a PKCS#11-compatible HSM, such as
  [SoftHSM](https://github.com/opendnssec/SoftHSMv2) or
  [YubiKey](https://www.yubico.com/)
- kubectl has no access to client key data

### Non-Goals

<!--
What is out of scope for this KEP?  Listing non-goals helps to focus discussion
and make progress.
-->

- HSM support on the server-side, i.e., kubernetes-apiserver (although a
  follow-up enhancement would be cool!)
- Improving PKCS#11 support in the Go runtime or in a Go library

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what you're
proposing, but should not include things like API designs or implementation.
The "Design Details" section below is for the real nitty-gritty.
-->

This KEP proposes introducing a new authentication provider to enable a secure
and complient usage of TLS client certificates within kubectl/client-go, by
delegating digital key operations to extarnal processes, for example HSMs. The
proposed authentication provider is similar to the [credentials
plugin](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#client-go-credential-plugins)
in that it also uses an external process during a part of the authentication
sequence, but contrary to the credentials plugin, it performs the sign operation
externally, and returns only the product of it - the signature.

In order to achieve a general solution, which can support various HSMs with
different protocols, this KEP proposes to split the authenticator into two
components and establish an API for the communication between them. The internal
component provides only the general primitives for the authentication process,
while leaving the implementation of a particular HSM protocol (for example
PKCS#11) out of kubectl/client-go. Support of a specific protocol can be enabled
by providing an external plugin that implements the proposed API.

### User Stories

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system.  The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

As a developer or an operator, I want to be able to authenticate my API requests
using a client certificate without a need of providing direct access to my
private key data so that I can improve compliance and security of the whole
system.

To authenticate against the API:

- The user issues a `kubectl` command, for example `kubectl get pods`.
- Authentication provider calls the external plugin to obtain a client
  certificate and signature.
- External plugin prompts the user for a PIN to perform signing operation (if
  the PIN is not provided in the configuration file).
- External plugin returns a client certificate and a signature to client-go via
  the authentication provider.
- API server verifies the signature and processes the request.

<!-- - Credential plugin prompts the user for LDAP credentials, exchanges credentials with external service for a token.
- Credential plugin returns token to client-go, which uses it as a bearer token against the API server.
- API server uses the webhook token authenticator to submit a TokenReview to the external service.
- External service verifies the signature on the token and returns the user’s username and groups. -->

<!-- #### Story 2 -->

### Notes/Constraints/Caveats

<!--
What are the caveats to the proposal? What are some important details that
didn't come across above. Go in to as much detail as necessary here. This might
be a good place to talk about core concepts and how they relate.
-->

 It was requested by the sig-auth community to come up with a solution that does
 not require CGO during the build process and preferably does not add new
 dependencies to kubectl.

### Risks and Mitigations

<!--
What are the risks of this proposal and how do we mitigate.  Think broadly. For
example, consider both security and how this will impact the larger kubernetes
ecosystem.

How will security be reviewed and by whom?

How will UX be reviewed and by whom?

Consider including folks that also work outside the SIG or subproject.
-->

## Design Details

<!--
This section should contain enough information that the specifics of your change
are understandable.  This may include API specs (though not always required) or
even code snippets.  If there's any ambiguity about HOW your proposal will be
implemented, this is the place to discuss them.
-->

The complete solution for external TLS certificate signing includes two
components:

- an authentication provider within kubectl/client-go, and
- an external plugin, which provides the actual certificate and signing
  services.

![UML sequence diagram](uml_sequence_diagram.png)

The main role of the authentication provider is to pass along requests for
obtaining a TLS certificate and signing to the external plugin, retrieve the
responses and return them to the initial caller.

Users would be required to install an external plugin (possibly along with
dependencies) on their workstation.

### Configuration

Configuration of the both components (authentication provider and external
plugin) is specified in the kubeconfig files as part of the user fields. The
authentication provider is responsible for reading the configuration from the
kubeconfig file and exposing the configuration parameters to the external
plugin.

The internal authentication provider requires only a single parameter, the path
to the external plugin `pathExec`.

<!-- TODO: Relative command paths are interpreted as relative to the directory of the config file. If KUBECONFIG is set to /home/jane/kubeconfig and the exec command is ./bin/example-client-go-exec-plugin, the binary /home/jane/bin/example-client-go-exec-plugin is executed. -->

The remaining of the parameters are specific for the external plugin and
protocol in use. In order to increase flexibility of the solution and support
for multiple protocols, the authentication provider will pass all the
configuration parameters specified in the kubeconfig file to the external plugin
in a form of string-to-string mapping (key-value pairs).

In case of the PKCS#11 protocol, the following parameters are in use:

- `pathLib` - a path to the library used by the authentication protocol
  (mandatory),
- `slotId` - an identifier of the slot (mandatory),
- `objectId` - an identifier of the object (mandatory),
- `pin` - a PIN code used when accessing the private key during the signing
  operation (optional, if not provided, the user will be asked to provide the
  PIN during each signing operation).

An excerpt from an exemplary kubeconfig file:

```yaml
apiVersion: v1
kind: Config
users:
- name: my-user
  user:
    auth-provider:
      name: externalSigner
      config:
        pathExec: /path/to/externalSigner  
        pathLib: /path/to/library.so        # library used by the external plugin
        objectId: "2"                       # PKCS#11 specific configuration
        slotId: "0"
        pin: "123456"                       # (optional)
```

### API specs
<!-- Input and output formats -->

Communication between the authentication provider and the external plugin is
bidirectional. The authentication provider initiates the communication by
sending a request for performing an operation (obtaining a client certificate or
signing) to the external plugin and the external plugin replies by sending a
response with a product of the respective operation (a client certificate or
signature).

All messages (requests and responses) are in the JSON format. The resources
(certificates and signatures) are Base64 encoded.

A request message is passed from the authentication provider to the external
plugin as an environment variable. The external plugin is expected to return the
response message by printing them to `stdout`. Moreover, the external plugin has
access to `stdin` for interacting with the user (for example providing a PIN)
and `stderr` for printing diagnostic information.

#### Obtaining a certificate

##### Certificate request

The authentication provider sends a request message in the JSON format of
`CertificateRequest` kind containing the plugin configuration parameters as an
environment variable.

```json
{
  "apiVersion":"external-signer.authentication.k8s.io/v1alpha1",
  "kind":"CertificateRequest",
  "configuration":{
    "objectId":"2",
    "pathExec":"/path/to/externalSigner",
    "pathLib":"/path/to/library.so",
    "pin":"123456",
    "slotId":"0"
  }
}
```

##### Certificate response

The external plugin returns a response massage in the JSON format of
`CertificateResponse` kind containing a Base64-encoded client certificate by
printing it to `stdout`.

```json
{
  "apiVersion":"external-signer.authentication.k8s.io/v1alpha1",
  "kind":"CertificateResponse",
  "certificate":"(CERTIFICATE BASE64 ENCODED)"
}
```

`k8s.io/client-go` uses the returned client certificate in the `certificate`.

#### Signing

##### Sign request

The authentication provider sends a request message as an environment variable
in the JSON format of `SignRequest` kind containing:

- the digest,
- plugin configuration parameters,
- the type of signer options (`rsa.PSSOptions` by default),
- signer options as a key-value pairs (`SaltLength` and `Hash` by default).

```json
{
  "apiVersion":"external-signer.authentication.k8s.io/v1alpha1",
  "kind":"SignRequest",
  "digest":"TqRUvJjLvlp3g9B3elpfzfgrSbukXBP5txkBLIkCSs4=",
  "configuration":{
    "objectId":"2",
    "pathExec":"/path/to/externalSigner",
    "pathLib":"/path/to/library.so",
    "pin":"123456",
    "slotId":"0"
    },
  "signerOptsType":"*rsa.PSSOptions",
  "signerOpts":"{\"SaltLength\":-1,\"Hash\":5}"
}
```

##### Sign response

The external plugin returns a response massage in the JSON format of
`SignResponse` kind containing a Base64-encoded signature by printing it to
`stdout`.

```json
{
  "apiVersion":"external-signer.authentication.k8s.io/v1alpha1",
  "kind":"SignResponse",
  "signature":"(SIGNATURE BASE64 ENCODED)"
}
```

`k8s.io/client-go` authenticates against the Kubernetes API using the signed
certificate returned in the `signature`.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy.  Anything
that would count as tricky in the implementation and anything particularly
challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations).  Please adhere to the [Kubernetes testing
guidelines][testing-guidelines] when drafting this test plan.

[testing-guidelines]:
https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

This enhancement includes unit and integration tests:

- Unit tests in `pkg/client/auth/externalsigner` to:
  - test that the external singer authentication provider APIs follow the format
    defined in the specification,
  - test the internal mechanisms of the authentication provider, such as
    caching, and
  - test handling of certificates and signatures data (including (un)marshalling
    the messages and en/decoding values).

- Integration tests in `test/integration/auth/externalsigner_test.go` to:
  - attempt an execution of a `kubectl` command with authentication using the
    external singer authentication provider.

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this
enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning),
or by redefining what graduation means.

In general, we try to use the same stages (alpha, beta, GA), regardless how the
functionality is accessed.

[maturity-levels]:
https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]:
https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity
levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least 2 releases between beta and GA/stable,
since there's no opportunity for user feedback, or even bug reports, in
back-to-back releases.

#### Removing a deprecated flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality which deprecates the
  flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]:
https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
-->

#### Alpha -> Beta Graduation

- Gather feedback regarding the API from developers of external plugins

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure this
is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade in order to make use of the enhancement?
  -->

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other components?
What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI, CRI
  or CNI may require updating that component before the kubelet.
  -->

## Implementation History

<!--
Major milestones in the life cycle of a KEP should be tracked in this section.
Major milestones might include
- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
  -->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider and why did you rule them out?  These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

### External signer vs existing authenticators using TLS certificates

First of all, it should be noted that Kubernetes already offers authentication
using client certificates, enabled by passing the `--client-ca-file=SOMEFILE`
option to API server, or by using the client-go [credential
plugins](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#client-go-credential-plugins)
(`exec`), which executes an arbitrary command that returns
`clientCertificateData` and `clientKeyData`. However, both of these alternatives
require kubectl to obtain a direct access to the client key data, what can be
considered unacceptable in some regulated environments and therefore does not
fullfil our requirements.

### Monolithic vs modular architecture

An alternative approach to enable HSM support within kubectl would be to create
a monolithic authentication provider, which implements the PKCS#11 protocol. The
drawback of this approach is that the support of PKCS#11 by definition requires
making C calls to the .so library, which makes CGO mandatory during the build
process of the whole kubectl. Such a dependency has been considered unacceptable
by the sig-auth community ([see the Slack
thread](https://kubernetes.slack.com/?redir=%2Farchives%2FC0EN96KUY%2Fp1582313541102300%3Fthread_ts%3D1582308815.101400%26cid%3DC0EN96KUY)).

### RPC vs exec

Similarly as in case of [external credential
providers](../541-external-credential-providers), external plugin could be
exposed as a network endpoint. Instead of executing a binary and passing
request/response over arguments/`stdout`, client could open a network connection
and send request/response over that.

The downsides of this approach compared to exec model are:

- if external signer is remote, design for client authentication is required
  (aka "chicken-and-egg problem"),
- external signer must constantly run, consuming resources; clients refresh
  their credentials infrequently.

### Independent external plugin configuration vs passing configuration parameters from kubectl/client-go

The external plugin may require configuration with some protocol specific
parameters, for example the path to a library implementing the communication
with an HMS, an identifier of HMS unit, etc. This communication could be stored
and loaded outside of the kubectl/client-go process, directly by the external
provider. However, that would require adding additional logic to the external
provider, which is already available within kubectl/client-go, as well as, would
spread configuration of authentication process into multiple files.

### Stdin vs program arguments vs environment variables

The authentication provider within the kubectl/client-go process sends to the
external plugin two categories of data:

- general configuration of the external plugin (as described above), and
- parameters for the sign operation (digest and signer options).

These data could be passed in various ways, for example:

- writing directly to the `stdin` of the external plugin,
- as program arguments set when spawning the extenal plugin process by the
  authentication provider,
- as environment variables set by the authentication provider and read by the
  external plugin.

We have decided to reserve `stdin` for the interactive usage of the external
plugin (for example, entering a PIN code). Since program arguments can be
visible for other processes running on the same host and we can be passing some
sensitive date (for example, a PIN code) we decided not to use them. Therefore,
we use environment variables to pass data from the authentication provider to
the external plugin.

### Multiple key-value pairs vs a single JSON string

The data passed from the authentication provider to the external plugin could be
arranged in various way, for example:

- as multiple environment variables, each forming a key-value pair,
- as a single string in JSON format, containing all the data.

Since some of the configuration parameters have a complex structure (maps) we
have decided to marshal them to JSON format and pass as a single environment
variable, instead of creating multiple environment variables.
<!-- Moreover, this approach fits well also the case of sign operation parameters, which are coming directly from within kubectl/client-go. -->

## Infrastructure Needed (optional)

<!--
Use this section if you need things from the project/SIG.  Examples include a
new subproject, repos requested, github details.  Listing these here allows a
SIG to get the process for these resources started right away.
-->
