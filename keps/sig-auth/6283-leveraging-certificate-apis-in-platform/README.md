# KEP-6283: Leveraging new certificate APIs in the platform

<!--
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
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Using ClusterTrustBundles directly as CA in API resources](#using-clustertrustbundles-directly-as-ca-in-api-resources)
  - [Platform-provided signer for serving certificates](#platform-provided-signer-for-serving-certificates)
    - [Referencing Services for the kubernetes.io/service-ca signer](#referencing-services-for-the-kubernetesioservice-ca-signer)
    - [On Pod IPs](#on-pod-ips)
  - [Use the 'kubernetes.io/kube-apiserver-serving' signer in workloads for kube-apiserver trust](#use-the-kubernetesiokube-apiserver-serving-signer-in-workloads-for-kube-apiserver-trust)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
    - [Unit tests](#unit-tests)
    - [Integration tests](#integration-tests)
    - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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

Two KEPs that significantly improve Kubernetes ability to work with X.509
certificates - 3257-ClusterTrustBundles and 4317-PodCetificates. The lay out
the APIs their backing logic, but don't directly make use of these. In this KEP,
we will make use of the principles these APIs brought.

## Motivation

We introduced mechanisms that are supposed to simplify people's experience with
X.509 certificates and bring their own signers but we don't currently use these
in places where they would make sense. This KEP does just that.

### Goals

- Direct use of ClusterTrustBundle references in API resources that mandate trust to
  a CA
- Use 'kubernetes.io/kube-apiserver-serving' signer to mount kube-apiserver trust in
  pods' ServiceAccount volumes
- Introduce a new easy-to-use signer for server identies for Pods and Services
- ? `request-header-ca`, `client-ca` from `extension-apiserver-authentication` ?

### Non-Goals

- remove the 'kube-root-ca.crt' ConfigMap from cluster namespaces

## Proposal

Allow users to use ClusterTrustBundle references in API resources that
otherwise require a CA PEM inlined.

Add a serving signer that anyone can use in their workload to get a
serving certificate trusted locally in the cluster.

Wire the 'kubernetes.io/kube-apiserver-serving' signer as the source of
trust for kube-apiserver serving cert in the automounted ServiceAccount
Pod volumes.

### User Stories (Optional)

#### Story 1

Instead of pasting the PEM bundle in API objects that require it so that
kube-apiserver can use it to trust a remote service, I would like to use
a reference to a signer defined by a ClusterTrustBundle that is easier
to maintain.

#### Story 2

I would like to have an easy-to-use way to get a cluster-local server
identity for my pods so that I can securely serve content within the
cluster.

#### Story 3

I would rather all pods used a single source for the ca.crt in their
auto-injected ServiceAccount volumes. I like the positive outlook that
we might one day be able to get rid of the `kube-root-ca.crt` ConfigMaps
from all the namespaces and thus making life a little easier for our
etcd databases.

### Notes/Constraints/Caveats (Optional)

This enhancement removes the need for the existence of the `kube-root-ca.crt`
ConfigMap as it uses a ClusterTrustBundle for its original purpose.
However, we cannot just remove the ConfigMap as there might be people relying
on it.

As for the serving signer, only a single signer exists for the whole cluster.
If app developers need their own trust domain, they still need to create
their own signer that would sign their PodCertificateRequests.

### Risks and Mitigations

\<TBD\>
Risk - People will love these features so much they will want more.
Mitigation - Make the next feature a little bit less awesome.
\</TBD\>

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

### Using ClusterTrustBundles directly as CA in API resources

This feature is controlled by the `ClusterTrustBundleSelector` feature gate that
is off-by-default during alpha and beta, but becomes on-by-default when the feature
reaches GA.

There are a few cases of API resources where we allow injecting
a PEM-formatted CA certificate, typically for the kube-apiserver to
be able to use it while connecting to remote services. Namely, this occurs
in these resources:

- `APIService` - `.spec.caBundle`
- `{Mutating,Validating}WebhookConfig` - `.webhooks[].clientConfig.caBundle`
- [`kubectl config`](https://github.com/kubernetes/kubernetes/blob/c72b65de137c544fe5ea1010dbf4738fd0ecea04/staging/src/k8s.io/client-go/tools/clientcmd/api/types.go#L41)
    - these do not make sense to replace in normal kubeconfig, but there might
      be *some* value when for API servers' static webhook configs
    - `.clusters[].certificate-authority{,-data}`
        - these should likely not be dynamically retrievable and should keep their
          current static form <-- TODO: get agreement on that
    - `.users[].client-certificate{,-data}`
        - does not make sense to replace, we still need to provide the key somehow

For this purpose, we will create a new API structure that we put beside the
API fields that expect CA PEM bundle strings. These fields will be mutually
exclusive.

```go
// ClusterTrustBundleSelector allows to configure trust by poiting to
// a selection of ClusterTrustBundles
type ClusterTrustBundleSelector struct {
    // name selects a single ClusterTrustBundle by name.
    // 
    // Mutually-exclusive with `signerName` and `labelSelector`.
    // +optional
    Name *string  `json:"name,omitempty"`
    
    // signerName selects all ClusterTrustBundles for a signer with
    // matching name.
    // The selection can be narrowed down by using `labelSelector`.
    // The contents of all selected ClusterTrustBundles will be
    // unified and deduplicated.
    //
    // Mutually-exclusive with `name`.
    // +optional
    SignerName *string `json:"signerName,omitempty"`
    
    // labelSelector allows to narrow down the selection of
    // ClusterTrustBundles for a signer with a given `signerName`.
    // If unset, interpreted as "match nothing". If set but empty,
    // interpreted as "match everything".
    //
    // Mutually-exclusive with `name`.
    // +optional
    LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
}
```

The logic that handles the normalization of the resulting bundle MUST be
shared with the [relevant kubelet code](https://github.com/kubernetes/kubernetes/blob/8c078bc32e90f0ea72f57169846b533e20b1ca71/pkg/kubelet/clustertrustbundle/clustertrustbundle_manager.go#L304-L335)
that handles ClusterTrustBundles volume projection.

### Platform-provided signer for serving certificates

This feature is controlled by the `KubeServingCertificatesSigner` feature gate that
is off-by-default during alpha and beta, but becomes on-by-default when the feature
reaches GA.

A new certificate signer is introduced to handle PodCertificateRequests for
serving certificates for Pods and Services - `kubernetes.io/service-ca`. The signer
is represented by a controller in the Kubernetes Controller Manager that publishes
a ClusterTrustBundle with the signer's trust anchor, and another controller to issue
certificates for incoming PodCertificateRequests that target this signer.

To be able to mint certificates on the service layer, the Kube Controller Manager
receives new options:

- `--service-ca-key` and `--service-ca-cert` that point to the signer's key and
  certificate, respectively. These must be two different files.
- `--cluster-service-domain` that will be appended for Pod/Service hostnames, e.g.
my-pod.my-namespace<strong>.svc.cluster.local</strong> for service domain **cluster.local**

The signer expects at least one of the following parameters in PodCertificateRequest
`unverifiedUserAnnotations`:
- `kubernetes.io/add-pod-fqdn`
    - adds the Pod FQDN to the SAN DNS extension of the resulting certificate
    - the pod must set its `spec.hostname` and a headless Service whose name
      matches the Pod's `spec.subdomain` must exist in the namespace and the
      Service's selector must match the Pod
    <!-- these rules are based on https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-hostname-and-subdomain-field -->
    - valid values: "true"
- `kubernetes.io/service-names`
    - adds parameters from Kubernetes Service to the certificate
    - valid values: comma-separated list of Service names in the Pod's namespace
	  that match the Pod labels in the Service's label selector

The attributes of the Pod requesting the certificate are mapped into the certificate
subject like this:
- `CN` is the Pod name
- `O` is the Pod namespace
- `uid` is the Pod UID (this uses the [standardized](https://www.rfc-editor.org/rfc/rfc4519.html#section-2.39)
`uid` OID 0.9.2342.19200300.100.1.1, not the Kubernetes-specific UID for client certificates)

#### Referencing Services for the kubernetes.io/service-ca signer

The `kubernetes.io/service-names` Service references require the following to be
considered valid:
- the referenced Service must exist in the Pod's namespace
- the `spec.selector` of the Service must match the Pod's labels

If the above conditions are fulfilled, parameters of the Service are added into
the resulting certificate.

For **headless services** the certificate gets SAN DNS names in the form of
`<svc-name>.<svc-namespace>.svc.<cluster-domain>` and `<svc-name>.<svc-namespace>.svc`.

For **cluster IP services** the certificate gets SAN DNS names same as for
headless services, but the Service's IP is also added to the certificate as SAN
IP address.

#### On Pod IPs

It would seem natural to also be able to add Pod IPs in the certificate
by using the same mechanisms as described above. Unfortunately, the Pod
network is only set up once all the volumes are successfully mounted, specifically:

1. Pod only gets `.status.PodIPs` before the volumes are handled if it is
   using [host network](https://github.com/kubernetes/kubernetes/blob/3e6b313d8c882d2fd36a779d3ea601fcbe0a8cb1/pkg/kubelet/kubelet.go#L2114-L2120)
2. Pod repeatedly [waits for all volumes to be mounted](https://github.com/kubernetes/kubernetes/blob/3e6b313d8c882d2fd36a779d3ea601fcbe0a8cb1/pkg/kubelet/kubelet.go#L2232)
   properly
3. control is handed over to the [container runtime](https://github.com/kubernetes/kubernetes/blob/3e6b313d8c882d2fd36a779d3ea601fcbe0a8cb1/pkg/kubelet/kubelet.go#L2267)
   kubelet handler
4. here the pod sandbox [gets created](https://github.com/kubernetes/kubernetes/blob/3e6b313d8c882d2fd36a779d3ea601fcbe0a8cb1/pkg/kubelet/kuberuntime/kuberuntime_manager.go#L1572-L1573)
   and later [Pod IPs are determined](https://github.com/kubernetes/kubernetes/blob/3e6b313d8c882d2fd36a779d3ea601fcbe0a8cb1/pkg/kubelet/kuberuntime/kuberuntime_manager.go#L1668-L1672)
   for the freshly created sandbox

As observed, Pod IPs are configured only after the volumes are set up. The codebase
currently is not ready to handle situations where Pod network might be needed
for some volumes, in fact this is likely a design choice.

### Use the 'kubernetes.io/kube-apiserver-serving' signer in workloads for kube-apiserver trust

This feature is controlled by the `KubeAPIServerWorkloadsTrust` feature gate that
is off-by-default during alpha and beta, but becomes on-by-default when the feature
reaches GA.

Kubernetes automatically mounts a ServiceAccount token to pods via a
projected volume. A part of that projected volume is a CA certificate
to trust the kube-apiserver's serving certificate.

This is done via [admission](https://github.com/kubernetes/kubernetes/blob/8c078bc32e90f0ea72f57169846b533e20b1ca71/plugin/pkg/admission/serviceaccount/admission.go#L418-L484)
that is using configmaps [distributed by the kube-controller-manager](https://github.com/kubernetes/kubernetes/blob/8c078bc32e90f0ea72f57169846b533e20b1ca71/pkg/controller/certificates/rootcacertpublisher/publisher.go#L189-L236)
for each namespace.

The ClusterTrustBundles KEP introduced a new signer for kube-apiserver
serving trust - 'kubernetes.io/kube-apiserver-serving', along with a
ClusterTrustBundle for it.

This KEP proposes to modify the admission plugin mentioned above to
use the ClusterTrustBundle instead of the configmaps. In the future
we may be able to remove the configmaps altogether but that is outside
the scope of this proposal.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

#### Prerequisite testing updates

\-

#### Unit tests

- k8s.io/apiserver/pkg/util/webhook: 2026-08-14 - 56.2%
- k8s.io/kube-aggregator/pkg/apiserver/handler_proxy.go: 2026-08-14 - 78.7%
- k8s.io/kubernetes/pkg/controller/certificates/rootcacertpublisher: 2026-08-14 - 69.1%

#### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

#### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

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
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- N examples of real-world usage
- N installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

<!--
- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

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

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

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

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ClusterTrustBundleSelector`
  - Components depending on the feature gate: `kube-apiserver`
- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `KubeServingCertificatesSigner`
  - Components depending on the feature gate: `kube-controller-manager`
- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `KubeAPIServerWorkloadsTrust`
  - Components depending on the feature gate: `kube-apiserver`
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

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

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

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

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

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

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

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

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

