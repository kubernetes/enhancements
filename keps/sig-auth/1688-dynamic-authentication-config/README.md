<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up.  KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please ensure to complete all
  fields in that template.  One of the fields asks for a link to the KEP.  You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "title", "authors", "owning-sig",
  "status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary", and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG that are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly.  The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved.  Any KEP
marked as a `provisional` is a working document and subject to change.  You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused.  If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement", for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example.  If there are
new details that belong in the KEP, edit the KEP.  Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable` or significant changes once
it is marked `implementable` must be approved by each of the KEP approvers.
If any of those approvers is no longer appropriate than changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross cutting KEPs).
-->
# KEP-1688: Dynamic Authentication Config

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
  - [User Stories (optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Notes/Constraints/Caveats (optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Restrictions on asserted identities](#restrictions-on-asserted-identities)
    - [Avoiding ambiguity](#avoiding-ambiguity)
    - [Ordering of authentication modes](#ordering-of-authentication-modes)
    - [Source of identity](#source-of-identity)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha implementation](#alpha-implementation)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

- [ ] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] KEP approvers have approved the KEP status as `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

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

Authentication in Kubernetes is quite flexible and unopinionated.  There is no
requirement for the API server to understand where an identity originates from.
The API server supports a variety of command line flags that enable authentication
via x509 certificates, OpenID Connect ID tokens, arbitrary bearer tokens via a
token webhook, etc.  Dynamic authentication config expands this capability to a
Kubernetes REST API.

## Motivation

<!--
This section is for explicitly listing the motivation, goals and non-goals of
this KEP.  Describe why the change is important and the benefits to users.  The
motivation section can optionally provide links to [experience reports][] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Configuring authentication via command line flags has some limitations.  Some
forms of config are simply easier to specify and understand in a structured
manner such as via a Kubernetes API or file.  The existing authentication flags
for OIDC and token webhook limit the user to only one of each of these types of
authentication modes as there is no way to specify the set of flags multiple
times [#71162].  These flag based configs also require an API server restart to
take effect.

Since there is no way to configure authentication other than via flags, hosted
environments have no consistent way to expose this functionality [#166].  To get
around this limitation, users install authentication proxies that run as
cluster-admin and impersonate the desired user [kube-oidc-proxy] [teleport].
Other than being a gross abuse of the impersonation API, this opens the API
server to escalation bugs caused by the proxy (such as improper handling of
incoming request headers).  This proxy also intercepts network traffic which
poses privacy and scalability concerns.

As a simple case study, it should be possible to deploy a simple identity
provider (ex: [dex]) directly onto a running cluster and be able to use that IDP
to authenticate users in the order of minutes without restarting the API server.
No proxying of network traffic should occur and the impersonation API must not
be used.

[#71162]: https://github.com/kubernetes/kubernetes/issues/71162
[#166]: https://github.com/aws/containers-roadmap/issues/166
[kube-oidc-proxy]: https://aws.amazon.com/blogs/opensource/consistent-oidc-authentication-across-multiple-eks-clusters-using-kube-oidc-proxy
[teleport]: https://aws.amazon.com/blogs/opensource/authenticating-eks-github-credentials-teleport
[dex]: https://github.com/dexidp/dex

### Goals

<!--
List the specific goals of the KEP.  What is it trying to achieve?  How will we
know that this has succeeded?
-->

1. Create a new Kubernetes REST API that allows configuration of authentication
  - x509
  - OIDC
  - Token Webhook
2. Changes made via the REST API should be active in the order of minutes without
requiring a restart of the API server
3. Allow the use of a custom authentication stack in hosted Kubernetes offerings

### Non-Goals

<!--
What is out of scope for this KEP?  Listing non-goals helps to focus discussion
and make progress.
-->

1. Changing the internal behavior of the authentication modes (ex: using
different fields in x509 certificates to build the `user.Info` object)
2. Restricting the REST API to the exact structure of the CLI flags
3. Any changes to the authentication CLI flags (deprecation, removal, etc)

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation.  The "Design Details" section below is for the real
nitty-gritty.
-->

This change aims to add a new Kubernetes REST API called `AuthenticationConfig`.
It is similar to the `ValidatingWebhookConfiguration` API in that it will be
watched by the API server and used to construct a struct.  In this case, this
struct with be an `authenticator.Request`, i.e. a dynamic authenticator.  This
authenticator will be unioned with the other authenticators.

### User Stories (optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system.  The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

Alice wishes to use Dex on an existing cluster.  She follows Dex's instructions
to deploy the IDP onto the cluster.  She then create a matching
`AuthenticationConfig` object with the `spec.type` set to `oidc` along with other
relevant options.  This allows the API server to validate ID tokens issued by Dex.
Alice configures the `kubectl` OIDC integration and uses a Dex provided identity
to authenticate to the cluster.

#### Story 2

Bob creates an `AuthenticationConfig` object with `spec.type` set to `x509`.  He
is then able to create a custom signer for use with the CSR API.  It can issue
client certificates that are valid for authentication against the Kube API.

#### Story 3

Charlie creates an `AuthenticationConfig` object with `spec.type` set to `webhook`.
This webhook is configured to honor GitHub tokens.  He configures an `exec`
credential plugin to make it easy for him to get tokens.  He is then able to
authenticate to the cluster using `kubectl` via GitHub tokens.

#### Story 4

(continues on from Story 3)

There is a service running in Charlie's cluster: `metrics.cluster.svc`.  This
service exposes some metrics about the cluster.  The service is assigned the
`system:auth-delegator` role and uses the `tokenreviews` API to limit access to
the data (any bearer token that can be validated via the `tokenreviews` API is
sufficient to be granted access).  Charlie uses his GitHub token to authenticate
to the service.  The API server calls the dynamic authentication webhook and is
able to validate the GitHub token.  Charlie is able to access the service.

#### Story 5

Dan wants to configure multiple `AuthenticationConfig` objects while guaranteeing
that they will not assert overlapping identities.  He configures each object with
a unique `spec.prefix`.

#### Story 6

Eve wants to use `gitlab.com` as an `oidc` `AuthenticationConfig`.  To prevent
every GitLab user from being able to authenticate to her cluster, she sets the
`spec.requiredGroups` field to her company's GitLab corporation which is
included as a group in the ID token.  Only users from her company are able to
authenticate to the cluster.

#### Story 7

Frank is exploring different options for authentication in Kubernetes.  He browses
various repos on GitHub.  He finds a few projects that are of interest to him.
He is able to try out the functionality using `kubectl apply` to configure his
cluster to use the custom authentication stacks.  He finds a solution that he
likes.  He uses the appropriate `kubectl apply` command to update his existing
clusters to the new authentication stack.

### Notes/Constraints/Caveats (optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

TBD.

### Risks and Mitigations

<!--
What are the risks of this proposal and how do we mitigate.  Think broadly.
For example, consider both security and how this will impact the larger
kubernetes ecosystem.

How will security be reviewed and by whom?

How will UX be reviewed and by whom?

Consider including folks that also work outside the SIG or subproject.
-->

#### Restrictions on asserted identities

To prevent confusion with identities that are controlled by Kubernetes, the
`system:` prefix will be disallowed in the username and groups contained in the
`user.Info` object.  A disallowed username will cause authentication to fail.
All disallowed groups will be filtered out.

#### Avoiding ambiguity

Since authorization modes such as RBAC are unaware of the origin of an identity,
care must be taken to avoid ambiguity between identities asserted by different
authenticators.  The `AuthenticationConfig` API will require username and groups
prefix config to be explicitly opted out of by the cluster admin.

#### Ordering of authentication modes

To prevent interference with other authentication modes configured on the API
server, all `AuthenticationConfig` objects will be considered after the other
authenticators have had a chance to run.  This prevents issues such as a
misbehaving authenticator attempting to authenticate a Kubernetes service
account bearer token.  It also prevents performance issues caused by invoking
a token webhook before the built-in authentication methods.

The ordering of the `AuthenticationConfig` objects in relation to each other is
(this is based on the performance of each type):

1. `x509` type
2. `oidc` type
3. `webhook` type

`AuthenticationConfig` objects of the same type will be ordered lexicographically.
No explicit control over ordering will be provided to the cluster admin.  All
`AuthenticationConfig` objects are equally trusted to assert arbitrary identities
and there is no concept of priority among them.

#### Source of identity

One important characteristic of an impersonation based proxy is that it is easy
to trace when it asserts an identity by looking at the API server's audit logs.
This property will be maintained by setting audit annotations upon successful
authentication.

The `authenticationconfigs.authentication.k8s.io/name` key will be set to the
`metadata.name` of the `AuthenticationConfig` object that asserted the `user.Info`.

The `authenticationconfigs.authentication.k8s.io/type` key will be set to the
`spec.type` of the `AuthenticationConfig` object that asserted the `user.Info`.

More audit annotations may be added in the future to aid in tracing the source
of the asserted identity.  For example, with the `oidc` type it may be helpful
to include the issuer URL and client ID.

#### Naming restrictions

The `metadata.name` field of all `AuthenticationConfig` objects must adhere to
the following validation rules:

1. `<hostname>:<identifier>` (two segments separated by a colon)
2. `hostname` must be at least two DNS1123 labels separated by `.`
3. `identifier` must be one or more DNS1123 subdomains separated by `.`
4. Max length of `571` characters

The validation rules are based off of the CSR API's `spec.signerName` validation
rules expect that `:` is used as the separator since `/` cannot be used in a the
`metadata.name` field due to etcd naming restrictions.

The `kubernetes.io` and `k8s.io` domains and all subdomains thereof are reserved
for future Kubernetes use and will result in a validation error.

`company.com:v1` is an example of a valid `metadata.name`.

#### Webhook restrictions

The aforementioned naming restrictions primarily exist to increase the security
and performance of token webhook implementations.  While webhook implementations
are responsible for understanding what tokens are meant for them, there is a
desire not to "leak" credentials to webhooks.  In particular, authentication
always falls-through to the next authenticator in the chain.  This can be
problematic when an authenticator suffers a transient failure.  Instead of
stopping early, later authenticators in the chain are attempted.  In the case of
dynamic webhook authenticators, this could result in service account tokens and
tokens meant for other webhooks being sent "to the wrong place."  Note that
these concerns do not apply to the `x509` and `oidc` types as the verification
process is performed locally by the API server.

To address these concerns, a bearer token will only be sent to a `webhook`
authenticator for verification if the token itself is prefixed with the name of
the `AuthenticationConfig` object followed by a `/`.  Thus if a webhook
authenticator is created with the name `company.com:v1`, only tokens with the
prefix `company.com:v1/` will be sent to it for verification.  The API server
will strip the prefix from the token before sending it to the webhook.  For
example, if `company.com:v1` and `panda.io:trees` are configured as webhook
authenticators and the token `company.com:v1/D2tIR6OugyAi70do2K90TRL5A` is
bared to the API server, the `panda.io:trees` authenticator will be ignored and
the token `D2tIR6OugyAi70do2K90TRL5A` will be sent to the `company.com:v1`
webhook for verification.

This guarantees that a JWT like the ones used for service account tokens and
with `oidc` are never sent to a `webhook`, increasing both performance and
preventing any leakage of these credentials.  Furthermore, only one `webhook`
will be invoked for a given token, thus cross webhook credential leakage cannot
occur.  Performance will also be increased by limiting the number of webhooks
that can be invoked on a given request.

It is the responsibility of the token issuer, the webhook, and the client to
coordinate on how tokens should be correctly prefixed.  For example, the issuer
could prefix the tokens automatically if it knows the name of the webhook.  Or
the client could perform the prefixing via an `exec` credential plugin.  In most
situations it should be possible to perform the prefixing without any end-user
involvement.

Question: should we support a simple form of wildcard by allowing a token such
as `company.com:*/D2tIR6OugyAi70do2K90TRL5A` to be sent to all webhooks that
have a `metadata.name` that starts with `company.com:`?

#### Cluster-admin only

Control over `AuthenticationConfig` objects allows the manufacture of arbitrary
identities.  As such, great care must be taken to limit access to this API to
cluster-admins.  Thus an escalation check similar to the one that RBAC performs
during a mutation of the `aggregationRule` field will be performed when an
`AuthenticationConfig` object is mutated.  The requirements are:

1. Group membership in `system:masters`, or
2. Subject access review:
    verb `escalate`
    name `metadata.name`
    namespace `""` (empty string since this API is cluster scoped)
    group `authentication.k8s.io`
    version `*`
    resource `authenticationconfigs`
    subresource `spec.type:<value>` (to allow for distinct authorization per type)
   or
3. Subject access review:
    verb `*`
    name `""` (empty string)
    namespace `""` (empty string)
    group `*`
    version `*`
    resource `*`
    subresource `""` (empty string)
   and
    verb `*`
    path `*`

Read level access will be controlled through standard authorization mechanisms.

#### Recovering from mistakes

Similar to the RBAC API, it is possible for the cluster-admin to lock themselves
out by deleting or incorrectly configuring the `authenticationconfigs` API.  The
`system:masters` group can be used as the explicit break glass mechanism but the
identity must be asserted by a configuration via the CLI flags.  This config is
required to set up the `authenticationconfigs` API in the first place, and thus
the simple recommendation would be to not alter the CLI config even after the
`authenticationconfigs` API is in use.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable.  This may include API specs (though not always
required) or even code snippets.  If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

The proposed functionality will be gated behind a new feature flag called
`DynamicAuthenticationConfig`.

The proposed API will reside in the `authentication.k8s.io` group at version
`v1alpha1` and resource `authenticationconfigs`.  The `AuthenticationConfig`
`Kind` is defined as:

```golang
type AuthenticationConfig struct {
  metav1.TypeMeta `json:",inline"`
  // +optional
  metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

  Spec AuthenticationConfigSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

type AuthenticationConfigSpec struct {
  Type AuthenticationConfigType `json:"type" protobuf:"bytes,1,opt,name=type"`

  // this generic username and group prefix config is always required
  // but the fields can be explicitly set to "-" to override it
  Prefix *PrefixConfig `json:"prefix" protobuf:"bytes,2,opt,name=prefix"`

  // TODO determine if this is the best approach to make "public" IDPs safe to use
  // the asserted user must be a member of at least one of these groups to authenticate
  // set to ["*"] to disable this check
  // required (cannot be empty)
  RequiredGroups []string `json:"requiredGroups" protobuf:"bytes,3,opt,name=requiredGroups"`

  X509 *X509Config `json:"x509,omitempty" protobuf:"bytes,4,opt,name=x509"`

  OIDC *OIDCConfig `json:"oidc,omitempty" protobuf:"bytes,5,opt,name=oidc"`

  Webhook *WebhookConfig `json:"webhook,omitempty" protobuf:"bytes,6,opt,name=webhook"`
}

type AuthenticationConfigType string

const (
  AuthenticationConfigTypeX509    AuthenticationConfigType = "x509"
  AuthenticationConfigTypeOIDC    AuthenticationConfigType = "oidc"
  AuthenticationConfigTypeWebhook AuthenticationConfigType = "webhook"
)

type X509Config struct {
  // caBundle is a PEM encoded CA bundle used for client auth (x509.ExtKeyUsageClientAuth).
  // +listType=atomic
  // Required
  CABundle []byte `json:"caBundle" protobuf:"bytes,1,opt,name=caBundle"`
}

type OIDCConfig struct {
  // this type is defined in admission registration
  Issuer admissionregistrationv1.WebhookClientConfig `json:"issuer" protobuf:"bytes,1,opt,name=issuer"`

  ClientID string `json:"clientID" protobuf:"bytes,2,opt,name=clientID"`

  UsernameClaim string `json:"usernameClaim" protobuf:"bytes,3,opt,name=usernameClaim"`

  GroupsClaim string `json:"groupsClaim" protobuf:"bytes,4,opt,name=groupsClaim"`
}

type WebhookConfig struct {
  // this type is defined in admission registration
  ClientConfig admissionregistrationv1.WebhookClientConfig `json:"clientConfig" protobuf:"bytes,1,opt,name=clientConfig"`

  // controls how the API server authenticates to the webhook (client cert, bearer token, etc)
  // this is just a placeholder as https://github.com/kubernetes/enhancements/pull/658
  // is tracking the work required to allow for this functionality
  // it may end up in the WebhookClientConfig struct instead of a new struct
  // ServerAuthentication *ServerAuthentication `json:"serverAuthentication,omitempty" protobuf:"bytes,2,opt,name=serverAuthentication"`
}

type PrefixConfig struct {
  UsernamePrefix string `json:"usernamePrefix" protobuf:"bytes,1,opt,name=usernamePrefix"`

  GroupsPrefix string `json:"groupsPrefix" protobuf:"bytes,2,opt,name=groupsPrefix"`
}
```

The `spec.type` field serves as a union discriminator to determine which config
struct to read.

Most of the functionality required to create the dynamic authenticator already
exists in the API server (to read and process the CLI flags).  It simply needs
to be wired up to an informer that triggers a rebuild of an authenticator that
will be stored in an `atomic.Value`.

A new `authenticatorfactory.CAContentProvider` will be required to support
dynamic certificate based authentication on the API server (via
`BuiltInAuthenticationOptions`) and the kubelet (via `DelegatingAuthenticatorConfig`).
This `CAContentProvider` will look similar to the dynamic authenticator except
that it will build and maintain a concatenation of all of the CA bundles from
the `AuthenticationConfig` objects with `spec.type` equal to `x509`.

Question: should we add white listing support to only allow certain identities?

Question: on removal of an `x509` config, should we close open connections?
What does the API server do today when it reloads a file based CA bundle?

Question: should we allow an `x509` config with an empty bundle that is a no-op
with the presumption that it will be updated later with a CA bundle?

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
expectations).  Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

Unit tests will be used to confirm that the logic of transforming the list of
`AuthenticationConfig` into a `authenticator.Request` works as expected.  Care
will be taken to ensure that:

1. Token caching semantics work (ex: cache is not thrown away on informer resync)
2. Initialization works (ex: `oidc` public key cache population)
3. Rebuilding the authenticator is not disruptive (ex: making changes to the
  `AuthenticationConfig` object should not result in transient authentication
  failures)
4. Audit annotations are correctly set

Integration tests will be used to assert that the overall wiring for the new
authenticator works as expected.

The kubelet will require a similar wiring test, but this will likely have to be
an end-to-end test.

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
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning),
or by redefining what graduation means.

In general, we try to use the same stages (alpha, beta, GA), regardless how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least 2 releases between beta and
GA/stable, since there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Removing a deprecated flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality which deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include [conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
-->

#### Alpha implementation

- API server support for `x509` and `oidc` `AuthenticationConfig` types
- Basic unit tests

#### Alpha -> Beta Graduation

- API server support for `webhook` `AuthenticationConfig` type
- Kubelet support for `x509` `AuthenticationConfig` type
- Complete server side validation of `AuthenticationConfig` REST API
- Complete unit and integration tests

#### Beta -> GA Graduation

- 3 examples of real world usage
- Gather feedback from example usages to confirm API structure is sufficient
- Associated documentation
- End-to-end test that deploys dex on cluster

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade in order to make use of the enhancement?
-->

No actions are required on upgrade as the `AuthenticationConfig` API is inert
until used.

If a cluster is downgraded to a version that does not support the
`AuthenticationConfig` API, all objects will become inert.  Thus care should be
taken before downgrading to ensure that some form of authentication will still
be available to the user after the downgrade is complete.

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

An n-2 kubelet may not support x509 authentication via the `AuthenticationConfig`
API.  This can be mitigated by explicitly configuring the kubelet via the
`--client-ca-file` flag.

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

2020-04-15: Initial KEP draft created
2020-05-15: KEP updated to address various comments

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

TBD.

## Alternatives

<!--
What other approaches did you consider and why did you rule them out?  These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

An officially supported impersonation proxy.  Abusing the impersonation API and
requiring all user traffic to pass through the proxy does not feel like an
approach that Kubernetes should endorse.

## Infrastructure Needed (optional)

<!--
Use this section if you need things from the project/SIG.  Examples include a
new subproject, repos requested, github details.  Listing these here allows a
SIG to get the process for these resources started right away.
-->

None.
