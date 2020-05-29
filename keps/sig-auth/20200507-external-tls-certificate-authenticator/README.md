# KEP-20200507: External TLS certificate authenticator

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
  - [External signer vs existing authenticators using TLS certificates](#external-signer-vs-existing-authenticators-using-tls-certificates)
  - [Monolithic vs modular architecture](#monolithic-vs-modular-architecture)
  - [RPC vs exec](#rpc-vs-exec)
  - [Independent external plugin configuration vs passing configuration parameters from kubectl/client-go](#independent-external-plugin-configuration-vs-passing-configuration-parameters-from-kubectlclient-go)
  - [Stdin vs program arguments vs environment variables](#stdin-vs-program-arguments-vs-environment-variables)
  - [Multiple key-value pairs vs a single JSON string](#multiple-key-value-pairs-vs-a-single-json-string)
  - [FIDO U2F](#fido-u2f)
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

This enhancement proposes adding support for authentication via external TLS
certificate signers, what would enable usage of Hardware Security Modules (HSMs)
- also known as smartcards, cryptographic processors or, by a popular brand
  name, YubiKeys(tm) via the PKCS#11 standard. This enhancement allows
  developers or automation pipelines to authenticate with the Kubernetes
  cluster, without requiring access to the client key, hence improving
  compliance and security.

## Motivation

A very common way for authenticating with a Kubernetes cluster is via private
keys. Even if other authentication methods are used, such as OpenID, private
keys are still necessary for break-glass scenarios. Some companies' key
management policy -- e.g., based on ISO 27001 Annex A.10 -- require delegating
all digital key operations to specialized [Hardware Security Modules
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

- kubectl can authenticate to a Kubernetes cluster with an external TLS
  certificate signer, for example a PKCS#11-compatible HSM, such as
  [SoftHSM](https://github.com/opendnssec/SoftHSMv2) or
  [YubiKey](https://www.yubico.com/)
- kubectl has no access to client key data

### Non-Goals

- HSM support on the server-side, i.e., kubernetes-apiserver
- Improving PKCS#11 support in the Go runtime or in a Go library
- FIDO U2F (see [Alternatives](#Alternatives))

## Proposal

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
by providing an external signer that implements the proposed API.

### User Stories

#### Story 1

As a developer or an operator, I want to be able to authenticate my API requests
using a client certificate without a need of providing direct access to my
private key data so that I can improve compliance and security of the whole
system.

To authenticate against the API:

- The user issues a `kubectl` command, for example `kubectl get pods`.
- `kubectl` calls the external signer to obtain a client certificate and
  signature.
- The external signer may suggest `kubectl` to show the user a prompt (i.e.,
  string) directing the user on what actions are necessary for allowing the
  signature.
- Depending on various factors, such as the external signer implementation, HSM
  support and company policy, the user may have to type a PIN in a model
  graphical pop-up window, touch the HSM or type a PIN on a special keyboard
  attached to the HSM.
- The external signer returns a client certificate and a signature to client-go
  via the authentication provider.
- The API server verifies the signature and processes the request.

### Notes/Constraints/Caveats

* The solution does not require `kubectl` with CGO.
* The solution does not require new secrets in KUBECONFIG.
* The solution does not invoke executables (executables in KUBECONFIG are
  considered insecure, due to the risk of distributing mallicious KUBECONFIGs).
* The solution should work with PKCS#11-based HSM.

### Risks and Mitigations

To reduce the risk of breaking the existing private key authentication for
kubectl users, the solution should be minimally intrusive to existing code
paths.

## Design Details

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

### FIDO U2F

Universal 2nd Factor (U2F) is a rather new standard proposed by the FIDO
Alliance. It is meant to complement user and password authentication with a
cryptographic signature produced by a cryptographic device, such as an HSM. In
fact, many HSM support both PKCS#11 and U2F.

U2F can readily be used against many OpenID providers, including Google, GitHub,
GitLab and others. However, even with strong authentication using OpenID, it is
still desirable to allow private key authentication to the Kubernetes cluster in
break-glass scenarios.