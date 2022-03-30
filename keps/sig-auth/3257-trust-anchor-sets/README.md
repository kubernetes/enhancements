<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [x] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [x] **Make a copy of this template directory.**
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
# KEP-3257: Cluster Trust Bundles

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
  - [User Stories](#user-stories)
    - [Trust Anchor Distribution for Private CAs](#trust-anchor-distribution-for-private-cas)
    - [Trust Anchor Configuration for Webhook Backends](#trust-anchor-configuration-for-webhook-backends)
    - [Trust Anchor Configuration for Aggregated API Server Backends](#trust-anchor-configuration-for-aggregated-api-server-backends)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [ClusterTrustBundle Object](#clustertrustbundle-object)
    - [API Object Definition](#api-object-definition)
    - [Access Control](#access-control)
    - [Admission Webhook Integration](#admission-webhook-integration)
    - [Aggregated API Server Integration](#aggregated-api-server-integration)
  - [trustAnchors Projected Volume Source](#trustanchors-projected-volume-source)
    - [Configuration Object Definition](#configuration-object-definition)
    - [Volume Content Generation and Refresh](#volume-content-generation-and-refresh)
  - [Canarying Changes to a ClusterTrustBundle](#canarying-changes-to-a-clustertrustbundle)
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
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
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

Today, the certificates.k8s.io API group provides a flexible, pluggable
mechanism for workloads running within the cluster to request certificates.
However, certificates only have meaning in relation to a trust anchor set.  Most
signers in a Kubernetes cluster don't issue publicly-trusted certificates, but
Kubernetes lacks a standardized mechanism for signers to distribute their trust
anchors to workloads.

A trust anchor is an X.509 certificate that forms a root of trust &mdash; if you are
presented with a leaf certificate, you decide whether or not to trust it based
on whether or not you can form a chain of intermediate certificates from the
leaf to a trust anchor.  This might be referred to as a “root certificate” in
other contexts, but nothing actually forces a trust anchor to be a **root**;
your trust anchor might be an intermediate certificate.  Additionally, note that
trust anchors are contextual; depending on the identity that a client or server
presents to you, you might wish to select a different trust anchor set.

This KEP introduces the certificates.k8s.io ClusterTrustBundle cluster-scoped
object, a vehicle for in-cluster certificate signers to communicate their trust
anchors to workloads.

Additionally, this KEP introduces a new `pemTrustAnchors` kubelet projected
volume source that gives workloads easy filesystem-based access to trust anchor
sets that they might need.

## Motivation

In the absence of a standard mechanism for trust distribution, a few different
approaches have emerged:

* The trust anchors necessary for connecting to kube-apiserver are distributed
  using a ConfigMap that is automatically injected into every namespace.

* Kubernetes webhooks and aggregated API servers are configured with a static
  trust anchor set embedded in their config (which complicates rotation).

* * At least one signer implementation uses a [cluster-scoped
  CRD](https://cloud.google.com/traffic-director/docs/security-proxyless-setup#:~:text=Save%20the%20following%20TrustConfig%20YAML%20configuration%20to%20tell%20your%20cluster%20how%20to%20trust%20the%20issued%20certificates%3A)
  to distribute trust anchors to workloads.

Each of these approaches has downsides:

* Creating a ConfigMap in every namespace requires granting over-broad
  permissions to the responsible controller (create ConfigMaps in **all**
  namespaces).

* Static trust anchor sets embedded directly in configuration complicate the job
  of rotating the backing CA's roots of trust.

* CRDs work well, but are inappropriate for use by core APIs.

Each of these approaches would be simplified by an in-tree mechanism for
distributing trust anchor sets.

### Goals

* Provide a Kubernetes data type that holds X.509 trust anchors for use wherever
  a set of X.509 trust anchors is needed.

* Make it easy for signer implementations (certificates/v1 or otherwise) to
  publish their relevant trust anchors for use within the cluster.

* Make it easy for workloads that need access to the trust anchors that back a
  particular certificates/v1 signer to access them within the container
  filesystem.

* Allow cluster operators to configure webhook and aggregated API server
  backends that use trust anchor objects instead of inline PEM data.

### Non-Goals

* Distribute a default set of "system" trust anchors.  (This is a natural
  extension, though).

* Handle trust anchors expressed in other forms than PEM-wrapped, DER-formatted
  X.509.

## Proposal

This proposal is centered around a new cluster-scoped ClusterTrustBundle
resource, initially in the certificates/v1alpha1 API group.  The
ClusterTrustBundle object can be thought of as a specialized configmap tailored
to the X.509 trust anchor use case.  Introducing a dedicated type allows us to
attach different RBAC policies to ClusterTrustBundle objects, which will
typically be wide-open for reading, but locked-down for writing.

Each ClusterTrustBundle object is a container for a single trust anchor
set.

ClusterTrustBundles may optionally be associated with a signer using their
`.spec.signerName` field.  ClusterTrustBundles that are associated with a signer
have special handling of creates and updates, restricting mutating actions to
those with an RBAC grant on the signer name (similar to the handling of
CertificateSigningRequest objects).

The `.spec.signerName` field will be supported for use in field selectors.

To enable consumption by workloads, the new `pemTrustAnchors` Kubelet projected
volume source writes the certificates from a ClusterTrustBundle into the
container filesystem, with the contents of the projected files updating as the
corresponding trust anchor sets are updated.

In general, ClusterTrustBundle objects are considered publicly-readable within
the cluster.  Read permissions will be granted cluster-wide to the
`system:authenticated` group.

### User Stories

#### Trust Anchor Distribution for Private CAs

In my cluster, I operate a certificates/v1 signer implementation called
`example.com/server-tls`.  This is backed by a private CA hierarchy.  By some
unspecified mechanism, I issue certificates from this CA hierarchy to a server
workload (a pod named `server/server`).  My client workload (a pod named
`client/client`) needs access to the trust anchors for the CA in order to
connect to the server.  The client should be able to continually establish
connections to the server without downtime, even as I rotate the trust anchors
that back the CA.

To do this, I update the controller that implements `example.com/server-tls` to
create a ClusterTrustBundle object like this one:
```yaml
apiVersion: v1alpha1
kind: ClusterTrustBundle
metadata:
  name: example.com--server-tls-live
spec:
  signerName: example.com/server-tls
  pemTrustAnchors: "<... PEM DATA ...>"
```

As the set of trust anchors for my CA changes due to rotations and revocations,
my controller updates the `spec` of the ClusterTrustBundle object.

I then update my client pod to add a `trustAnchors` volume:
```diff
apiVersion: v1
kind: Pod
metadata:
  namespace: client
  name: client
spec:
  containers:
    name: main
    image: my-image
    volumeMounts:
+    - mountPath: /var/run/example-com-server-tls-trust-anchors
+      name: example-com-server-tls-trust-anchors
+      readOnly: true
  volumes:
+  - name: example-com-server-tls-trust-anchors
+    projected:
+      sources:
+      - pemTrustAnchors:
+          clusterTrustBundleName: example.com--server-tls-live
+          path: ca_certificates.pem
```

When my client pod is scheduled onto a node, Kubelet writes the chosen trust
anchor set into the container filesystem of my main container.  My main
container doesn't start until `ca_certificates.pem` is present and non-empty.

I ensure that my client application is capable of reading a set of trust anchors
from a file containing PEM-wrapped DER-formatted X.509 certificates at the
location I've specified
(`/var/run/example-com-server-tls-trust-anchors/ca_certificates.pem`).

I ensure that my client application notices changes to `ca_certificates.pem`
(using inotify or periodic polling), and properly updates its TLS stack when
that happens.

#### Trust Anchor Configuration for Webhook Backends

In my cluster, I operate a collection of validating and mutating admission
webhooks that use a server TLS certificate issued from my company's private CA.
Today, when my private CA introduces a new root as part of a scheduled rotation
(or incident response), I need to track down and adjust all
ValidatingWebhookConfiguration and MutatingWebhookConfiguration objects in my
cluster that refer to my private CA, and update their
`.webhooks[].clientConfig.caBundle` fields.

ClusterTrustBundle objects let me centralize the management of my private CA's
roots within my cluster.

To use it, I create a new ClusterTrustBundle containing the current roots of my
private CA.

```yaml
apiVersion: v1alpha
kind: ClusterTrustBunddle
metadata:
  name: example-com-private-ca
spec:
  signerName: "" # This ClusterTrustBundle isn't associated with a signer, so I
                 # leave this empty.
  pemTrustAnchors: "<... PEM DATA ...>"
```

Then, I update each ValidatingWebhookConfiguration and
MutatingWebhookConfiguration object that currently refer to my private CA,
removing the CABundle field and adding a reference to the ClusterTrustBundle
object I created.

```diff
apiVersion: admissionregistration/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: my-validating-webhook
webhooks:
  name: frobnicator.example.com
  clientConfig:
    url: https://webhooks.example.com/frobnicator
-    caBundle: "< long base64 data>"
+    trustAnchors:
+      clusterTrustBundleName: example-com-private-ca

```

Once I do so, kube-apiserver will use the root certificates it loads from the
designated trust anchor set in order to validate the certificate presented by
the webhook backend.

When I need to perform a root rotation, I only need to modify my single
`example-com-private-ca` ClusterTrustBundle to introduce the new root and retire
the old root.

#### Trust Anchor Configuration for Aggregated API Server Backends

This is the same as for webhook backends, but I use the new alpha `trustAnchors`
field in the APIServiceSpec object.

### Risks and Mitigations

Scalability: In the limit, ClusterTrustBundle objects will be used by
every pod in the cluster, which will require watches from all Kubelets.  When
they are updated, workloads will need to receive the updates fairly quickly
(within 5 minutes across the whole cluster), to accommodate emergency rotation
of trust anchors for a private CA.

Security: Should individual trust anchor set entries designate an OSCP endpoint
to check for certificate revociation?  Or should we require the URL to be
embedded in the issued certificates?

## Design Details

### ClusterTrustBundle Object

The ClusterTrustBundle object allows signers to publish their trust anchors for
consumption by workloads, whether directly, via Kubelet, or otherwise.

ClusterTrustBundle objects may optionally be associated with a signer using the
`.spec.signerName` field.  Setting the `.spec.signerName` field enables
additional admission control and access control for the ClusterTrustBundle
object, as detailed in the [Access Control](#access-control) section.

While multiple ClusterTrustBundle objects may be associated with a signer, each
is a completely independent expression of the roots of trust for the signer.  No
effort is made to unify trust anchors from different ClusterTrustBundle objects.
The primary use case for multiple ClusterTrustBundle objects for a single signer
is [canarying](#canarying-changes-to-a-clustertrustbundle).

ClusterTrustBundle objects will support `.metadata.generation`.

ClusterTrustBundle creates and updates that result in an empty
`.spec.pemTrustAnchors` will be rejected during admission.

All of the new objects and fields described in this section are gated by the
ClusterTrustBundle kube-apiserver feature gate.  Unless the feature gate is
enabled, usage of these alpha objects and fields will be rejected by
kube-apiserver.

#### API Object Definition

```go
// ClusterTrustBundle packages up trust anchors.
//
// Each ClusterTrustBundle object is optionally owned by a particular signer, which
// updates the trust anchors as the roots of trust for that signer change.
//
// A ClusterTrustBundle object wraps a single set of PEM certificates.
type ClusterTrustBundle struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    // The spec of the object.
    Spec ClusterTrustBundleSpec `json:"spec"`
}

// ClusterTrustBundleSpec is the spec subfield of a ClusterTrustBundle object.
type ClusterTrustBundleSpec struct {
  // The name of the associated signer.
  //
  // +optional
  SignerName string `json:"signerName,omitempty"`

  // The individual trust anchors for this bundle.  A PEM bundle of PEM-wrapped,
  // DER-formatted X.509 certificate.
  //
  // The order of certificates within the bundle has no meaning.
  PEMTrustAnchors string `json:"pemTrustAnchors"`
}
```

#### Access Control

All ClusterTrustBundle objects are assumed to be publicly-readable within the
cluster, due to the fact that the kubelet trustAnchors volume type will allow
any pod to mount any ClusterTrustBundle.  This will be backed up by a built-in
grant of get, list, and watch for ClusterTrustBundle objects to
`system:authenticated`.

For ClusterTrustBundle objects without `.spec.signerName` specified, no
nonstandard access control is applied.  Typically, a controller (or human
operator) who creates such a bundle will grant read access to the
ClusterTrustBundle to the `system:authenticated` group (tighter grants are
possible, but are defeated by `pemTrustAnchors` Kubelet volume type, which
allows any pod in the cluster to read any ClusterTrustBundle).

For ClusterTrustBundle objects with `.spec.signerName` specified, some
additional admission checks (similar to those on CertificateSigningRequest
objects) are applied to creates and updates.  Creates and updates are rejected
unless the requester has permission for the `entrust` verb on group
`certificates.k8s.io`, resource `signers`, with resourceName equal to
`.spec.signerName`.

For example, to allow a controller to manage ClusterTrustBundle objects for
`example.com/my-signer`, the following ClusterRole could be used.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: example-com-trustmanager
rules:
- apiGroups:
  - certificates.k8s.io
  resources:
  - signer
  resourceNames:
  - example.com/my-signer
  verbs:
  - entrust
```

(Note that this permission does not need to be granted via RBAC; it could be
granted via any of Kubernetes' supported authorization methods.)

Updates to ClusterTrustBundle objects that attempt to change `.spec.signerName`
are always rejected.

#### Well-known ClusterTrustBundles

ClusterTrustBundles named with the prefix `kubernetes-io--` are reserved for
future use by the Kubernetes project.

When moving the ClusterTrustBundles feature to beta, we will figure out the
semantics for exporting the trust anchors used by the built-in signers under
this prefix.

#### Trust Anchor Configuration for Admission and Conversion Webhook Backends

The `admissionregistration.k8s.io/v1` and `apiextensions.k8s.io/v1`
WebhookClientConfig objects will be extended with an optional reference to a
single trust anchor set within a single ClusterTrustBundle object.  If
ClusterTrustBundleName is non-empty and the ClusterTrustBundle feature gate is
set, ClusterTrustBundleName takes precedence over CABundle.

When TrustAnchors is specified, kube-apiserver will load the certificates from
the named trust anchor set of the named ClusterTrustBundle object, and use those
for validating the certificate presented by the webhook backend.  kube-apiserver
will follow updates to the ClusterTrustBundle object, and use the updated
certificates within a few seconds.

```diff
type WebhookClientConfig struct {
    // ...

    // `caBundle` is a PEM encoded CA bundle which will be used to validate the webhook's server certificate.
    // If unspecified, system trust roots on the apiserver are used.
    // +optional
    CABundle []byte `json:"caBundle,omitempty" protobuf:"bytes,2,opt,name=caBundle"`

+	// A reference to a ClusterTrustBundle that will be used to validate
+	// the webhook's server certificate.
+	//
+	// If set (and the ClusterTrustBundle feature gate is enabled.), takes precedence
+ // over CABundle.
+	//
+	// This field is alpha.  Using it requires setting the ClusterTrustBundle
+	// feature gate.
+	//
+	// +optional
+	ClusterTrustBundleName string `json:"clusterTrustBundleName,omitempty"`
}
```

#### Trust Anchor Configuration for Aggregated API Server Backends

The `apiregistration.k8s.io` v1 APIServiceSpec object will be extended with a
new optional field ClusterTrustBundleName that references a single trust anchor
set within a single ClusterTrustBundle object.  If ClusterTrustBundleName is
non-empty and the ClusterTrustBundle feature gate is set, ClusterTrustBundleName
takes precendence over the CABundle and InsecureSkipTLSVerify fields.

When `TrustAnchors` is specified, kube-apiserver will load the certificates from
the named trust anchor set of the named ClusterTrustBundle object, an use those
for validating the certificate presented by the aggregated API server.
kube-apiserver will follow updates to the ClusterTrustBundle object, and use the
updated certificates within a few seconds.

```diff
type APIServiceSpec struct {
    // ...

    // InsecureSkipTLSVerify disables TLS certificate verification when communicating with this server.
    // This is strongly discouraged.  You should use the CABundle instead.
    InsecureSkipTLSVerify bool `json:"insecureSkipTLSVerify,omitempty" protobuf:"varint,4,opt,name=insecureSkipTLSVerify"`
    // CABundle is a PEM encoded CA bundle which will be used to validate an API server's serving certificate.
    // If unspecified, system trust roots on the apiserver are used.
    // +listType=atomic
    // +optional
    CABundle []byte `json:"caBundle,omitempty" protobuf:"bytes,5,opt,name=caBundle"`
+	// A reference to a ClusterTrustBundle that will be used to validate
+	// the webhook's server certificate.
+	//
+	// Takes precendence over the InsecureSkipTLSVerify and CABundle fields.
+	//
+	// This field is alpha.  Using it requires setting the ClusterTrustBundle
+	// feature gate.
+	//
+	// +optional
+	ClusterTrustBundleName string `json:"clusterTrustBundleName,omitempty"`

    // ...
}
```

### pemTrustAnchors Projected Volume Source

The pemTrustAnchors projected volume source is responsible for writing the
contents of a particular ClusterTrustBundle object into the container filesystem
as a single file containing a verbatim copy of the correct
`.spec.pemTrustAnchors` field.

All of the new fields described in this section are gated by the
ClusterTrustBundle kubelet feature gate.  Unless the feature gate is enabled,
usage of these alpha objects and fields will be rejected by kubelet.

#### Configuration Object Definition

```go
// PEMTrustAnchorsProjection is the top-level configuration object for
// instances of the trustAnchors kubelet projected volume source.
type PEMTrustAnchorsProjection struct {
  // An explicit ClusterTrustBundle to retrieve trust anchor sets from.
  ClusterTrustBundleName string `json:"clusterTrustBundleName"`

  // Where should we write the flat file in the projected volume?
  Path string `json:"path"`
}
```

#### Volume Content Generation and Refresh

For each pemTrustAnchors projected volume source in the pod spec:

1. If the named ClusterTrustBundle exists, kubelet will write
   spec.pemTrustAnchors from the ClusterTrustBundle into the designated file.

2. If the named ClusterTrustBundle does not exists, kubelet will fail the volume
   mount process and emit an event describing the error.

The volume will not be successfully mounted (and thus the pod will not start)
until all certificate files have been written to the container filesystem.

Once the pod has started, Kubelet will continue to update the contents of files
as the associated ClusterTrustBundles change.  If the ClusterTrustBundle backing
a file gets deleted, kubelet will keep the file at the last-available content,
periodically emitting an event documenting the problem.

### Canarying Changes to a ClusterTrustBundle

Because ClusterTrustBundle objects can potentially impact TLS traffic across the
cluster, it's important to be able to canary changes to them, so that problems
do not break traffic cluster-wide.  This KEP does not impose any particular
canarying strategy (or require the use of canarying at all), but it does provide
the tools to do so.

Human operators or controllers may manage multiple ClusterTrustBundle objects
for a given signer, each of which may contain different trust anchors.
ClusterTrustBundle objects are referred to by name from pod specs, allowing
different pods to refer to different ClusterTrustBundles for the same signer.

This allows a root change to be canaried by maintaining a set of versioned trust
bundles for the signer (for example, `example.com--my-signer-v1` and
`example.com--my-signer-v2`).  A few applications can have their podspecs
updated to refer to the new version, and their health can then be assessed.
Once the new version is confirmed to work, all applications can be updated to
point to it.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None determined so far.

##### Unit tests

All changes will have unit test coverage.

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

(I'm not yet sure precisely which packages will need to be adjusted.)

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

The following scenarios will need to be tested using integration tests:

* kube-apiserver forbids creation and update of a ClusterTrustBundle targeted if
  the requester doesn't have the `entrust` verb on the corresponding signer.

* kube-apiserver correctly connects to validating, mutating, and conversion
  webhook backends that use a ClusterTrustBundle for managing their trust
  anchors, and can follow a CA rotation in the corresponding ClusterTrustBundle.

* kube-apiserver correctl connects to aggregated API server backends that use a
  ClusterTrustBundle for managing their trust anchors.

Additionally, it would be nice to test the following kubelet behavior in integration tests if feasible:

* Kubelet correctly writes (and updates) the content of a ClusterTrustBundle
  named in the pod spec into the container fileystem.

* Kubelet blocks pod startup if the pod spec refers to a ClusterTrustBundle that
  doesn't exist.

* Kubelet blocks pod startup if the pod spec refers to a ClusterTrustBundle with
  an empty `.spec.pemTrustAnchors` field.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

The following end-to-end tests will be needed:

* A workload that uses a pemTrustAnchors projected volume that names a
  well-formed ClusterTrustBundle can start up and has the expected content in
  the targeted file.

### Graduation Criteria

#### Alpha

In alpha, all new fields and objects are guarded by the ClusterTrustBundle
feature gate, which is present in both kube-apiserver and kubelet.

The feature will be covered by unit, integration, and E2E tests as described above.

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

### Upgrade / Downgrade Strategy

At present, there are no upgrade / downgrade concerns.  The ClusterTrustBundle
feature gate controls overall availability of the feature.

### Version Skew Strategy

Both kubelet and kube-apiserver will need to be at 1.26 for the full featureset
to be present.  If only kube-apiserver is at 1.26 and kubelet is lower, then the
the pod mounting feature will be cleanly unavailable, but all other aspects of
the feature will work.

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

###### How can this feature be enabled / disabled in a live cluster?

This feature is gated by the ClusterTrustBundle feature gate.  The feature gate controls:

* Whether or not kube-apiserver will accept operations on ClusterTrustBundle
  objects.

* Whether or not kube-apiserver will accept webhook and aggregated API server
  backend configs that refer to ClusterTrustBundle objects.

* Whether or not kubelet will accept pemTrustAnchors projected volume sources in
  pod specs.

###### Does enabling the feature change any default behavior?

Enabling the feature does not change any default behavior.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, if the ClusterTrustBundle feature gate is disabled, the cluster will
largely continue to function properly.

Any pods that specify a pemTrustAnchors projected volume source will encounter
errors as kubelet will no longer process their spec.  Existing pods will stop
seeing updates to their PEM files, and new pods will be rejected.

Any webhook or aggregated API server configuration objects that specify trust
using a ClusterTrustBundle object will revert back to the behavior specified by
their CABundle and InsecureTLSSkipVerify fields, since the
ClusterTrustBundleName field is no longer set.  This is safe, because users are
not required to change those fields when setting the ClusterTrustBundleName
field; instead, ClusterTrustBundleName takes precedence.

Any ClusterTrustBundle objects in the cluster will stop being served, but be
retained in etcd storage.

###### What happens if we reenable the feature if it was previously rolled back?

Existing ClusterTrustBundle objects that are still in etcd will reappear.

Any pods that use pemTrustAnchors projected volume sources will stop being
rejected by kubelet.

Any webhook or aggregated API server configs that refer to a ClusterTrustBundle
object will start working again.

###### Are there any tests for feature enablement/disablement?

Unit tests will exercise the feature gates.

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

###### How can an operator determine if the feature is in use by workloads?

Use of existing metrics to track operations against certificates.k8s.io/v1alpha1
ClusterTrustBundle objects will provide a good proxy for the amount of active
use of the feature.

Kubelet should export a count of pemTrustAnchors volumes mounted / refreshed, and a latency
histogram for time spent specifically in the pemTrustAnchors volume mount
process.

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

Operations on ClusterTrustBundle objects are covered by the existing [API call latency SLOs](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/api_call_latency.md).

Usage of pemTrustAnchors projected volumes are covered by the existing [pod startup latency SLOs](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/pod_startup_latency.md).

We may need a new SLO to govern the latency between an update to a ClusterTrustBundle's certificate set and that change being made available to pods using pemTrustAnchors volumes.

We may need a new SLO to govern the latency between an update to a ClusterTrustBundle's certificate set and that change being made respected by webhook and aggregated API server connections.


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

If we can find a low-cardinality way for Kubelet to report what object
generation of each ClusterTrustBundle object that it is currently exporting to
pods, that would help implement the two new SLOs proposed above.

### Dependencies

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

###### Will enabling / using this feature result in any new API calls?

Yes.

A pod that uses a pemTrustAnchors projected volume will result in an additional
watch on the named ClusterTrustBundle object orginating from Kubelet.  This
watch will be low-throughput.

Similar to the existing kubelet watches on secrets and configmaps, special care
will need to be taken to ensure that kube-apiserver can efficiently choose which
single-ClusterTrustBundle watches to update for a given etcd update.

A webhook or aggregrated API server config that refers to a ClusterTrustBundle
object will result in an additional (kube-apiserver local?) watch on the named
ClustertrustBundle object, originating from kube-apiserver.  This watch will be
low-throughput.

###### Will enabling / using this feature result in introducing new API types?

ClusterTrustBundle (no per-cluster limit enforced)

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No, except for the additional pod, webhook, and aggregated API server fields
that the user sets to make use of the feature.

5 new ClusterTrustBundle objects will be automatically created, corresponding to
the built-in Kubernetes signers.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Use of the pemTrustAnchors projected volume type will impact the [existing
SLOs](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/pod_startup_latency.md)
that cover startup latency for stateless schedulable pods (similar to the
existing effect from).

The definition of "stateless" will also need to be updated to include pods that
use pemTrustAnchors volumes, unless it is intentional that those SLOs exclude
projected token volumes.

(Kubelet projected volumes seem problematic in general for the startup latency
SLOs --- if I create a pod that projects many large configmaps, I can introduce
arbitrary startup latency for my pod and cause an SLO breach.)

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Use of the ClusterTrustBundle objects by themselves should have negligible
resource impact.

Use of ClusterTrustBundle objects in webhook and aggregated API server configs
should have negligible resource impact.

Use of the pemTrustAnchors projected volume type will result in an additional
watch on kube-apiserver for each unique (Node, ClusterTrustBundle) tuple in the
cluster.  This is similar to existing kubelet support for projecting configmaps
and secrets into a pod.  Just like for secrets and configmaps, we will need to
make sure that kube-apiserver has the indexes it needs to efficiently map etcd
watch events to Kubernetes watch channels.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

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

This KEP assumes that TLS and mutual TLS are technologies that we want to
elevate to first-class status within the Kubernetes ecosystem.  This is a
significant commitment, but not out of line with the existing decision to use
OpenID Connect as the foundation for pod identity tokens.

For the problems that TLS and mTLS address (non-exfiltratable identity
certificates), there are no serious alternatives.

## Alternatives

### Do nothing in-tree; solve trust distribution with CRD/CSI driver

This KEP is informed by the experience of implementing and operating
`gke-spiffe`, an Kubernetes-integrated SPIFFE certificate issuance mechanism
that backs several service mesh features.  It's built using the custom resource
/ CSI driver approach, which works well.  However, there is appetite in SIG Auth
for:

1. Fixing parts of the Kubernetes API that embed static trusted root
   certificates, like webhooks and aggregated API servers.  When these backends
   use certificates issued by a private CA, rotating the root certificate of the
   CA requires tracking down all objects that have these embedded roots and
   updating them.  Adding a layer of indirection will allow rotation to be
   performed with a single, central update.
2. Making progress towards a future where pods are issued identity certificates,
   just like they are issued identity tokens today.

Item 1 requires a core object to describe trust anchors.  Item 2 could be accomplished using CRDs and CSI drivers, but we eventually want these to be standardized features.

### Use a ConfigMap rather than defining a new type

ConfigMaps have two problems in this application:

* They are namespace-scoped, whereas signers are cluster-scoped.
* They have poor human factors.  They are not often considered to contain
  security-critical data, and so may have wide update permissions assigned to
  them.

### Use a TrustAnchorSet (singular) object

One possible alternative would be for the new object type to be TrustAnchorSet
(singular), and have each signer correspond to multiple TrustAnchorSet objects
(using a label selector).

There are two problems with this approach.

First: Modification of trust anchors is security-critical.  It must be possible
to restrict a particular signer's controller using RBAC so that it only has
permission to update the trust anchors it is responsible for.  RBAC
ClusterRoleBindings do not have the ability to grant update to objects based on
a label selector, only based on the object name.  This implies that it must be
possible to name all of the TrustAnchorSet objects that correspond to the signer
at the time you write the ClusterRoleBinding.  This removes the flexibility that
label selectors are meant to provide.

Second: When multiple mutual TLS meshes are federating, pods in mesh A
connecting to pods in mesh B will need to be able to look up the appropriate
roots to establish trust.  This requires an efficient mechanism for the pod to
find the correct TrustAnchorSet (singular) object.  The only workable mechanism
I'm aware of is to encode the signer name (to prevent collisions) and the
context-specific trust anchor set name into the name of the TrustAnchorSet
object.

For example, assume I operate pods in a an mTLS mesh corresponding to my company
`a.com`, using certificate issuance machinery implemented by the signer
`example.com/my-mtls-signer`.  If I want to federate my pods with another
company, `b.com`, my pods need to be able to quickly look up the trust anchor
set corresponding to `example.com/my-mtls-signer` and `b.com`.

### Support for other certificate formats beyond PEM-wrapped DER-formatted X.509

PEM-wrapped DER-formatted X.509 is the lingua franca for TLS stacks to load
certificates.  The only exception I'm aware of is Java, which requires the
creation of a Java-specific keystore file.  In the future, we could add support
to the Kubelet trustAnchorSets volume to emit certificates in the Java keystore
format.

## Infrastructure Needed (Optional)

No additional infrastructure is needed.
