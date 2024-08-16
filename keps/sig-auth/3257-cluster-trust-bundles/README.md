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
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [ClusterTrustBundle Object](#clustertrustbundle-object)
    - [API Object Definition](#api-object-definition)
    - [Access Control](#access-control)
    - [Well-known ClusterTrustBundles](#well-known-clustertrustbundles)
  - [ClusterTrustBundle Projected Volume Source](#clustertrustbundle-projected-volume-source)
    - [Configuration Object Definition](#configuration-object-definition)
    - [Volume Content Generation and Refresh](#volume-content-generation-and-refresh)
  - [Canarying Changes to a ClusterTrustBundle](#canarying-changes-to-a-clustertrustbundle)
  - [Publishing the kube-apiserver-serving Trust Bundle](#publishing-the-kube-apiserver-serving-trust-bundle)
    - [The kube-apiserver-serving signer](#the-kube-apiserver-serving-signer)
    - [Kubelet and KCM API discovery](#kubelet-and-kcm-api-discovery)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
  - [Beta](#beta)
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
  - [Do nothing in-tree; solve trust distribution with CRD/CSI driver](#do-nothing-in-tree-solve-trust-distribution-with-crdcsi-driver)
  - [Use a ConfigMap rather than defining a new type](#use-a-configmap-rather-than-defining-a-new-type)
  - [Support for other certificate formats beyond PEM-wrapped DER-formatted X.509](#support-for-other-certificate-formats-beyond-pem-wrapped-der-formatted-x509)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
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

A trust anchor is an X.509 certificate that forms a root of trust &mdash; if you
are presented with a leaf certificate, you decide whether or not to trust it
based on whether or not you can form a chain of intermediate certificates from
the leaf to a trust anchor.  This might be referred to as a “root certificate”
in other contexts, but nothing actually forces a trust anchor to be a **root**;
your trust anchor might be an intermediate certificate.  Additionally, note that
trust anchors are contextual; depending on the identity that a client or server
presents to you, you might wish to select a different trust anchor set.

This KEP introduces the certificates.k8s.io ClusterTrustBundle cluster-scoped
object, a vehicle for in-cluster certificate signers to communicate their trust
anchors to workloads.

Additionally, this KEP introduces a new `clusterTrustBundle` kubelet projected
volume source that gives workloads easy filesystem-based access to trust anchor
sets that they might need.

A new ClusterTrustBundle with a new signer `kubernetes.io/kube-apiserver-serving` also gets created
for all clusters. In combination with the new projected volume source,
this trust bundle can be eventually used to replace the use of `kube-root-ca.crt` configMaps
that nowadays live in all namespaces.
## Motivation

In the absence of a standard mechanism for trust distribution, a few different
approaches have emerged:

* The trust anchors necessary for connecting to kube-apiserver are distributed
  using a ConfigMap that is automatically injected into every namespace.

* Kubernetes webhooks and aggregated API servers are configured with a static
  trust anchor set embedded in their config (which complicates rotation).

* At least one signer implementation uses a [cluster-scoped
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

### Non-Goals

* Distribute a default set of "system" trust anchors.  (This is a natural
  extension, though).

* Handle trust anchors expressed in other forms than PEM-wrapped, DER-formatted
  X.509.

* Have the kube-apiserver consume ClusterTrustBundles as a part of service/webhook APIs.
  This enhancement does not specify a revocation mechanism for a trust represented
  by a ClusterTrustBundle. Having this mechanism would be a natural follow-up
  candidate to this KEP.

## Proposal

This proposal is centered around a new cluster-scoped ClusterTrustBundle
resource, initially in the certificates/v1alpha1 API group.  The
ClusterTrustBundle object can be thought of as a specialized configmap tailored
to the X.509 trust anchor use case. Introducing a dedicated type allows us to
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

To enable consumption by workloads, the new `clusterTrustBundle` Kubelet
projected volume source writes the certificates from a ClusterTrustBundle into
the container filesystem, with the contents of the projected files updating as
the corresponding trust anchor sets are updated.

Finally, the ClusterTrustBundle API is exercised to create an object for
distributing the serving trust to the kube-apiserver, currently represented by
the `kube-root-ca.crt`configMap that's synced into every namespace.

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
  name: example.com:server-tls:foo
  labels:
    kubernetes.io/cluster-trust-bundle-version: live
spec:
  signerName: example.com/server-tls
  trustBundle: "<... PEM DATA ...>"
```

As the set of trust anchors for my CA changes due to rotations and revocations,
my controller updates the `spec` of the ClusterTrustBundle object.

I then update my client pod to add a `clusterTrustBundle` volume:
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
+      - clusterTrustBundle:
+          signerName: example.com/server-tls
+          labelSelector:
+            kubernetes.io/cluster-trust-bundle-version: live
+          path: ca_certificates.pem
```

Kubelet selects all ClusterTrustBundles that match the signer name and label selector, merges their trustBundle fields (discarding duplicate certificates and applying a stable but arbitrary ordering) and writes them to `ca_certificates.pem`.

The mechanics of projected volumes prevent my pod's containers from starting until `ca_certificates.pem` is present and non-empty.

I ensure that my client application is capable of reading a set of trust anchors
from a file containing PEM-wrapped DER-formatted X.509 certificates at the
location I've specified
(`/var/run/example-com-server-tls-trust-anchors/ca_certificates.pem`).

I ensure that my client application notices changes to `ca_certificates.pem`
(using inotify or periodic polling), and properly updates its TLS stack when
that happens.

### Risks and Mitigations

Scalability: In the limit, ClusterTrustBundle objects will be used by every pod
in the cluster, which will require one ClusterTrustBundle watch from each
Kubelet in the cluster.  When they are updated, workloads will need to receive
the updates fairly quickly (within 5 minutes across the whole cluster), to
accommodate emergency rotation of trust anchors for a private CA.

Security: Should individual trust anchor set entries designate an OCSP endpoint
to check for certificate revociation?  Or should we require the URL to be
embedded in the issued certificates?  Note: This question is deferred from the 1.30 beta scope, and will be discussed as an addition to the beta scope in 1.31.

## Design Details

### ClusterTrustBundle Object

The ClusterTrustBundle object allows signers to publish their trust anchors for
consumption by workloads, whether directly, via Kubelet, or otherwise.

ClusterTrustBundle objects may optionally be associated with a signer using the
`.spec.signerName` field.  Setting the `.spec.signerName` field enables
additional admission control and access control for the ClusterTrustBundle
object, as detailed in the [Access Control](#access-control) section.

ClusterTrustBundles that are associated with a signer have a required naming
prefix:  take the signer name, replace slashes (`/`) with colons (`:`), and
append a colon at the end.  This must then be followed by an additional
freely-chosen name that does not contain colons.  For example, valid
ClusterTrustBundle names for the signer `example.com/mysigner` would be
`example.com:mysigner:foo` and `example.com:mysigner:live`.

ClusterTrustBundles that are not associated with a signer may be freely named,
as long as their name does not contain a colon.

Multiple ClusterTrustBundle objects may be associated with a single signer.
While each object is independent at the API level, consumers (mostly Kubelet)
will select trust anchors by a combination of field selector on signer name, and
a label selector.  Signer controllers may follow the convention of making the
label selector `kubernetes.io/cluster-trust-bundle-version=live` correspond to a
meaningful set of trust anchors.  In general, users are expected to read the documentation for their signer controller implementation in order to determine which label selectors to use for their needs, including [canarying](#canarying-changes-to-a-clustertrustbundle).

ClusterTrustBundle objects support `.metadata.generation`.

ClusterTrustBundle creates and updates that result in an empty
`.spec.pemTrustAnchors` will be rejected during validation.

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
  TrustBundle string `json:"trustBundle"`
}
```

#### Access Control

All ClusterTrustBundle objects are assumed to be publicly-readable within the
cluster, due to the fact that the kubelet trustAnchors volume type will allow
any pod to mount any ClusterTrustBundle.  This will be backed up by a built-in
grant of get, list, and watch for ClusterTrustBundle objects to the
`system:serviceaccounts` and `system:nodes` groups.

For ClusterTrustBundle objects without `.spec.signerName` specified, no
nonstandard access control is applied.  Due to the `clusterTrustBundle` projected
volume, any workload in the cluster will be able to load the contents of any
ClusterTrustBundle.

For ClusterTrustBundle objects with `.spec.signerName` specified, some
additional admission checks (similar to those on CertificateSigningRequest
objects) are applied to creates and updates.  Creates and updates are rejected
unless the requester has permission for the `attest` verb on group
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
  - attest
```

`resourceNames` can also be specified in the wildcard form `example.com/*` to
grant access to all ClusterTrustBundles under a given signer name domain prefix.

(Note that this permission does not need to be granted via RBAC; it could be
granted via any of Kubernetes' supported authorization methods.)

Updates to ClusterTrustBundle objects that attempt to change `.spec.signerName`
are always rejected.

#### Well-known ClusterTrustBundles

Signer names with the domain prefix `kubernetes.io` (including any subdomains) are reserved by the Kubernetes project, and thus by extension the ClusterTrustBundle prefix name `kubernetes.io:` is reserved.

(Targeting Kubernetes 1.31) Built-in signers should export their roots where it
makes sense.  For example, the root certificate(s) for clients to validate the
kube-apiserver serving certificate should be made available in a singular
ClusterTrustBundle, replacing the current per-namespace ConfigMaps.

### ClusterTrustBundle Projected Volume Source

The ClusterTrustBundle projected volumes source is responsible for writing the contents of ClusterTrustBundle objects into the container filesystem as a single file containing trust anchor sets in PEM format.

It has two modes:
* ClusterTrustBundles with no signer name are selected by object name.  Only a
  single ClusterTrustBundle can be selected in a single projected volume source.
* ClusterTrustBundles with a signer name are selected by the intersection of a
  field selector on signer name and a label selector.

All of the new fields described in this section are gated by the
ClusterTrustBundleProjection kubelet feature gate.  Unless the feature gate is
enabled, usage of these alpha objects and fields will be rejected by kubelet.

#### Configuration Object Definition

```go
// ClusterTrustBundleProjection describes how to select a set of
// ClusterTrustBundle objects and project their contents into the pod
// filesystem.
type ClusterTrustBundleProjection struct {
	// Select a single ClusterTrustBundle by object name.  Mutually-exclusive
	// with signerName and labelSelector.
	// +optional
	Name *string `json:"name,omitempty" protobuf:"bytes,1,rep,name=name"`

	// Select all ClusterTrustBundles that match this signer name.
	// Mutually-exclusive with name.  The contents of all selected
	// ClusterTrustBundles will be unified and deduplicated.
	// +optional
	SignerName *string `json:"signerName,omitempty" protobuf:"bytes,2,rep,name=signerName"`

	// Select all ClusterTrustBundles that match this label selector.  Only has
	// effect if signerName is set.  Mutually-exclusive with name.  If unset,
	// interpreted as "match nothing".  If set but empty, interpreted as "match
	// everything".
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty" protobuf:"bytes,3,rep,name=labelSelector"`

	// If true, don't block pod startup if the referenced ClusterTrustBundle(s)
	// aren't available.  If using name, then the named ClusterTrustBundle is
	// allowed not to exist.  If using signerName, then the combination of
	// signerName and labelSelector is allowed to match zero
	// ClusterTrustBundles.
	// +optional
	Optional *bool `json:"optional,omitempty" protobuf:"varint,5,opt,name=optional"`

	// Relative path from the volume root to write the bundle.
	Path string `json:"path" protobuf:"bytes,4,rep,name=path"`
}
```

#### Volume Content Generation and Refresh

For each ClusterTrustBundle projected volume source in the pod spec:

1. If the ClusterTrustBundle source uses Name:
    1. Kubelet reads the individual certificates out of spec.trustBundle
       (discarding any PEM data side channels like headers and inter-block data).
    2. Kubelet deduplicates the individual certificates.
2. If the ClusterTrustBundle source uses SignerName and LabelSelector:
    1. Kubelet selects all ClusterTrustBundle objects matching SignerName and
       LabelSelector.
    2. Kubelet collects all individual certificates from each of the matching
       ClusterTrustBundle's spec.trustBundle fields (discarding and PEM data
       side channels like headers and inter-block data).
3. If no certificates were collected, Kubelet throws an error.
4. Kubelet deduplicates the collected certificates.
5. Kubelet orders the certificates in an arbitrary but stable ordering.
6. Kubelet writes the certificates to the designated file in PEM format.
7. If any error was thrown during the process, the volume mount process will
   fail.

Once the pod has started, Kubelet will continue to periodically update the
contents of the designated file by re-running the logic above.  Any errors will
cause Kubelet to hold the file at the last known-good content, periodically
emitting an event describing the error.

### Canarying Changes to a ClusterTrustBundle

Because ClusterTrustBundle objects can potentially impact TLS traffic across the
cluster, it's important to be able to canary changes to them, so that problems
do not break traffic cluster-wide.  This KEP does not impose any particular
canarying strategy (or require the use of canarying at all), but it does provide
the tools to do so.

Human operators or controllers may use unique names and labels to maintain different versions of the trust anchor set for a given signer, and coordinate their workloads to canary trust anchor changes across different workloads.

For example, if I maintain `example.com/my-signer`, I can use the following strategy:
* I maintain one ClusterTrustBundle named `example.com:my-signer:live`, labeled
  `kubernetes.io/cluster-trust-bundle-version=live` (the object name is mostly
  irrelevant).
* I maintain an additional ClusterTrustBundle named
  `example.com:my-signer:canary`, labeled
  `kubernetes.io/cluster-trust-bundle-version=canary`.
* I have coordinated some fraction of my workloads to use the canary label
  selector, while the bulk of them use the live label selector
* When I want to perform a root rotation or other trust change, I edit the
  canary object first, and assess the health of the canary workloads.
* Once I am satisfied that the change is safe, I edit the live object.

### Publishing the kube-apiserver-serving Trust Bundle

Today, the trust bundle that allows verifying kube-apiserver serving certificate(s)
at its internal endpoints is distributed into every namespace using a configMap.
This is so that it can be mounted along with the ServiceAccount token in order
for the workloads to be able to communicate with the kube-apiserver.

In the future, we should be able to replace mounting these configMaps in pods for
for kube-apiserver trust with the projected volume from this feature, and so a
ClusterTrustBundle API object will be created for all clusters:

```yaml
apiVersion: v1beta1
kind: ClusterTrustBundle
metadata:
  generateName: kubernetes.io:kube-apiserver-serving:
spec:
  signerName: kubernetes.io/kube-apiserver-serving
  trustBundle: "<... PEM CA ...>"
```

This object is managed by the Kubernetes Controller Manager in the existing
`root-ca-certificate-publisher-controller`. It serves the same purpose
and contains the same content as the `ca.crt` data in the `kube-root-ca.crt` configMap - to verify internal kube-apiserver
endpoints. There is currently no in-tree signer designated for these purposes,
and so the signer with name `kubernetes.io/kube-apiserver-serving` is introduced
along with this bundle.

This behavior is feature-gated by the `ClusterTrustBundle` KCM feature gate.

#### The kube-apiserver-serving signer

The signer signs certificates that can be used to verify kube-apiserver serving
certificates. Signing and approval are handled outside kube-controller-manager.

**Trust distribution** - signed certificates are used by the kube-apiserver for TLS
server authentication. The CA bundle is distributed using a ClusterTrustBundle object
identifiable by the `kubernetes.io/kube-apiserver-serving` signer name.
**Permitted subjects** - "Subject" itself is deprecated for TLS server authentication. However,
it should still follow the same rules on DNS/IP SANs from the "Permitted x509 extensions" section
below.
**Permitted x509 extensions** - honors subjectAltName and key usage extensions. At
least one DNS or IP subjectAltName must be present. The SAN DNS/IP of the certificates
must resolve/point to kube-apiserver's hostname/IP.
**Permitted key usages** - ["key encipherment", "digital signature", "server auth"] or ["digital signature", "server auth"].
**Expiration/certificate lifetime** - The recommended maximum lifetime is 30 days.
**CA bit allowed/disallowed** - not recommended.

#### Kubelet and KCM API discovery

Functionalities in both the kubelet and KCM depend on the presence of the ClusterTrustBundle
API. If the `ClusterTrustBundleProjection` (kubelet) and `ClusterTrustBundle` (KCM) feature
gates are enabled, the kubelet and the KCM perform API discovery at startup to check for the
API presence at the version they need. If the API is not present, neither kubelet nor KCM
will enable the new behavior, and the check won't be performed until they restart again.

If the API gets disabled on the kube-apiserver side, both the kubelet and KCM must be
restarted in order for the feature to be disabled there, too.

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

| Package | Date | Coverage |
| ------- | ---- | -------- |
| k8s.io/api/                                                            | 2023-06-15 | (not listed) |
| k8s.io/api/certificates/v1alpha1                                       | 2023-06-15 | (not listed) |
| k8s.io/api/core/v1                                                     | 2023-06-15 | (not listed) |
| k8s.io/kubernetes/cmd/kube-apiserver/app                               | 2023-06-15 | 32% |
| k8s.io/kubernetes/pkg/api/pod                                          | 2023-06-15 | 78%|
| k8s.io/kubernetes/pkg/apis/certificates/install                        | 2023-06-15 | (not listed) |
| k8s.io/kubernetes/pkg/apis/certificates                                | 2023-06-15 | (not listed) |
| k8s.io/kubernetes/pkg/apis/certificates/v1alpha1                       | 2023-06-15 | (not listed) |
| k8s.io/kubernetes/pkg/apis/certificates/validation                     | 2023-06-15 | 97% |
| k8s.io/kubernetes/pkg/apis/core                                        | 2023-06-15 | 80% |
| k8s.io/kubernetes/pkg/apis/core/validation                             | 2023-06-15 | 84% |
| k8s.io/kubernetes/pkg/controller/volume/attachdetach                   | 2023-06-15 | 65% |
| k8s.io/kubernetes/pkg/controller/volume/expand                         | 2023-06-15 | 30% |
| k8s.io/kubernetes/pkg/controller/volume/persistentvolume               | 2023-06-15 | 80% |
| k8s.io/kubernetes/pkg/controlplane/                                    | 2023-06-15 | (not listed) |
| k8s.io/kubernetes/pkg/features                                         | 2023-06-15 | (not listed) |
| k8s.io/kubernetes/pkg/kubeapiserver                                    | 2023-06-15 | (not listed) |
| k8s.io/kubernetes/pkg/kubeapiserver/options                            | 2023-06-15 | 79% |
| k8s.io/kubernetes/pkg/kubelet                                          | 2023-06-15 | (not listed) |
| k8s.io/kubernetes/pkg/kubelet/clustertrustbundle                       | 2023-06-15 | (newly added) |
| k8s.io/kubernetes/pkg/printers/internalversion                         | 2023-06-15 | 71% |
| k8s.io/kubernetes/pkg/registry/certificates/clustertrustbundle/storage | 2023-06-15 | (newly added) |
| k8s.io/kubernetes/pkg/registry/certificates/clustertrustbundle         | 2023-06-15 | (newly added) |
| k8s.io/kubernetes/pkg/registry/certificates/rest                       | 2023-06-15 | (not listed) |
| k8s.io/kubernetes/pkg/registry/registrytest                            | 2023-06-15 | (not listed) |
| k8s.io/kubernetes/pkg/volume                                           | 2023-06-15 | 46% |
| k8s.io/kubernetes/pkg/volume/projected                                 | 2023-06-15 | 69% |
| k8s.io/kubernetes/pkg/volume/testing                                   | 2023-06-15 | (not listed) |
| k8s.io/kubernetes/plugin/pkg/admission/certificates/ctbattest          | 2023-06-15 | (newly added) |
| k8s.io/kubernetes/plugin/pkg/admission/serviceaccount                  | 2023-06-15 | 89% |
| k8s.io/kubernetes/plugin/pkg/auth/authorizer/rbac/bootstrappolicy/     | 2023-06-15 | (not listed) |

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

The following scenarios will need to be tested using integration tests:

* kube-apiserver forbids creation and update of a ClusterTrustBundle targeted if
  the requester doesn't have the `attest` verb on the corresponding signer.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

The following end-to-end tests will be needed:

* (Alpha) A pod that uses a ClusterTrustBundle projected volume that names
  a well-formed ClusterTrustBundle can start up and has the expected content in
  the targeted file.
* (Beta) Happy path: A pod selecting multiple ClusterTrustBundles by signer
  name and label selector.
* (Beta) A pod that selects one or more ClusterTrustBundles that don't exist
  still starts up if the `optional` flag is set on the volume.
* (Beta) A pod that selects one or more ClusterTrustBundles that don't exist is
  blocked from starting if the `optional` flag is not set on the volume.

### Graduation Criteria

#### Alpha

In alpha, all new fields and objects are guarded by the ClusterTrustBundle and
ClusterTrustBundleProjection feature gates.

The feature will be covered by unit, integration, and E2E tests as described above.

### Beta

For Beta, the ClusterTrustBundle type is moved to the certificates.k8s.io/v1beta1 API group.

The feature is still guarded by the ClusterTrustBundle and ClusterTrustBundleProjection feature gates.

The feature is covered by unit, integration, and E2E tests.  E2E test coverage is expanded from the current "happy path" test.

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

The feature is gated by these feature flags:
- kube-apiserver
    - `ClusterTrustBundle` controls availability of the ClusterTrustBundle API and
      presence of the relevant rules in cluster roles for the kubelet and KCM.
    - `ClusterTrustBundleProjection` controls availability of the `ClusterTrustBundle`
      projected volume source in the Pod API.
- kubelet
  - `ClusterTrustBundleProjection` controls the availability of the kubelet being
    able to mount the volumes. If disabled, kubelet will error out on any attempt to
    mount a ClusterTrustBundle projected volume.
- KCM - `ClusterTrustBundle` controls the availability of the kube-apiserver-serving's
  signer ClusterTrustBundle.

The proper order at which the feature should be enabled is to start with the
kube-apiserver's feature flags. Aside from enabling the API, the `ClusterTrustBundle`
feature gate also creates the necessary rules in the `system:node` cluster role.

Once the kube-apiserver feature gates are enabled, the order of enabling the feature
at kubelet or KCM does not matter.

### Version Skew Strategy

The ClusterTrustBundle volume projection was implemented in 1.29 and kubelet would fail to mount CTB
volumes if it was requested via the Pod API while the feature gate is disabled on
kubelet side. This means that pods will fail to become ready in version-skewed environments where the
`ClusterTrustBundleProjection` kubelet feature gate is disabled, independently of the
API version.

If the `ClusterTrustBundleProjection` kubelet feature gate is enabled but the API is at a different
version than the kubelet expects, the kubelet will behave as if the feature gate was disabled.
This will cause pods trying to mount a ClusterTrustBundle to fail to become ready as kubelet
won't be able to create the mount. If the API eventually appears at the desired version, the kubelet
must be restarted in order to enable the new behavior.

Enabling the `ClusterTrustBundle` feature gate at KCM while a different-than-KCM-expected
API version is being served will make the KCM to act as if the feature gate was disabled.
If the API eventually appears at the desired version, the KCM must be restarted in order
to enable the new behavior.

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

The ClusterTrustBundle feature gate controls whether or not kube-apiserver will
accept operations on ClusterTrustBundle objects.

The ClusterTrustBundleProjection feature gate controls whether or not kubelet
will accept pemTrustAnchors projected volume sources in pod specs.

###### Does enabling the feature change any default behavior?

Enabling the feature does not change any default behavior.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, if the ClusterTrustBundle feature gate is disabled, the cluster will
largely continue to function properly.

Any pods that specify a ClusterTrustBundle projected volume source will
encounter errors as kubelet will no longer process their spec.  Existing pods
will stop seeing updates to their PEM files, and new pods will be rejected.

Any ClusterTrustBundle objects in the cluster will stop being served, but be
retained in etcd storage.

###### What happens if we reenable the feature if it was previously rolled back?

Existing ClusterTrustBundle objects that are still in etcd will reappear.

Any pods that use pemTrustAnchors projected volume sources will stop being
rejected by kubelet.

###### Are there any tests for feature enablement/disablement?

Unit tests will exercise the feature gates.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollout consists of enabling the ClusterTrustBundle and
ClusterTrustBundleProjection feature gates, then restarting kube-apiserver and
all kubelets.  

Initial rollout should cause no changes in the cluster, beyond each kubelet
establishing an additional watch on ClusterTrustBundle objects.  During rollout
in HA clusters, there may be a period where workloads that use
ClusterTrustBundles can be scheduled to a node, but be unable to begin running
because either kubelet or a particular kube-apiserver replica does not yet
understand clustertrustbundles.

Rollback consists of disabling the ClusterTrustBundle and ClusterTrustBundleProjection feature gates, then restarting kube-apiserver and all kubelets.

Rollback risks are primarily that workloads using ClusterTrustBundles will become nonfunctional.  Existing pods will stop seeing updates to projected ClusterTrustBundles, and new pods created by workload controllers will be rejected.

Rollfoward consists of enabling the ClusterTrustBundle and ClusterTrustBundleProjection feature gates, then restart kube-apiserver and all kubelets.

Rollforward has the same risks as rollout, with the additional complication that there may be "sleeper" workloads in the cluster: ClusterTrustBundle objects in etcd, and deployments or other workload controllers trying to create Pods with ClusterTrustBundle projected volume sources.  These workloads will immediately light up on rollforward, which could cause a step change in the number of pods running and Kubelet memory usage.

###### What specific metrics should inform a rollback?

Kubelet memory usage is the primary risk factor, since kubelet holds all ClusterTrustBundle objects in the cluster in an informer cache.  This is tracked by the `clustertrustbundle_informer_cache_size` metric.

Otherwise, the feature is opt-in: only workloads that mount ClusterTrustBundle projected volume sources will interact with ClusterTrustBundles.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Manual upgrade and rollback have not yet been tested, but will be tested during the 1.30 development cycle.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Use of existing metrics to track operations against certificates.k8s.io/v1alpha1
ClusterTrustBundle objects will provide a good proxy for the amount of active
use of the feature.

Kubelet will export the following metrics, which will help gauge usage and
health of ClusterTrustBundle projection into pods:
- `clustertrustbundle_informer_cache_size`: Gauge measuring the total memory usage in bytes of the ClusterTrustBundle informer cache.  Exported from kubelet.
  - `projectedvolumesource_instances`: Gauge counting current usage of projected volumes sources on a particular node. Exported from kubelet.  Tags:
    - `projected_volume_source_type`: ClusterTrustBundle, ServiceAccountToken, etc.
  - `projectedvolumesource_refresh_count`: Counter tracking the number of calls to the projected volume source refresh logic.  Exported from kubelet.  Tags:
    - `projected_volume_source_type`: ClusterTrustBundle, ServiceAccountToken, etc.
    - `status`: OK, or error code.
  - `projectedvolumesource_refresh_duration`: Histogram of times spent refreshing the content of projected volumes sources,.  Exported from kubelet.  Tags:
    - `projected_volume_source_type`: ClusterTrustBundle, ServiceAccountToken, etc.
    - `status`: OK, or error code.


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
- [x] Other (treat as last resort)
  - Users can see that pods that use ClusterTrustBundle projected volume sources are able to begin running.
  - This doesn't cover showing that running pods are having their mounted trust bundles updated properly, so we need to think about how to cover them with events or conditions.
  - Users can see that a ClusterTrustBundle for the signer `kubernetes.io/kube-apiserver-serving` exists in the cluster

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Operations on ClusterTrustBundle objects are covered by the existing [API call
latency
SLOs](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/api_call_latency.md).

Usage of ClusterTrustBundle projected volumes are covered by the existing [pod
startup latency
SLOs](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/pod_startup_latency.md).

We may need a new SLO to govern the latency between an update to a
ClusterTrustBundle's certificate set and that change being made available to
pods using ClusterTrustBundle volumes.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?


- Metrics
  - `clustertrustbundle_informer_cache_size`: Gauge measuring the total memory usage in bytes of the ClusterTrustBundle informer cache.  Exported from kubelet.
  - `projectedvolumesource_instances`: Gauge counting current usage of projected volumes sources on a particular node. Exported from kubelet.  Tags:
    - `projected_volume_source_type`: ClusterTrustBundle, ServiceAccountToken, etc.
  - `projectedvolumesource_refresh_count`: Counter tracking the number of calls to the projected volume source refresh logic.  Exported from kubelet.  Tags:
    - `projected_volume_source_type`: ClusterTrustBundle, ServiceAccountToken, etc.
    - `status`: OK, or error code.
  - `projectedvolumesource_refresh_duration`: Histogram of times spent refreshing the content of projected volumes sources,.  Exported from kubelet.  Tags:
    - `projected_volume_source_type`: ClusterTrustBundle, ServiceAccountToken, etc.
    - `status`: OK, or error code.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

If we can find a low-cardinality way for Kubelet to report what object
generation of each ClusterTrustBundle object that it is currently exporting to
pods, that would help implement the two new SLOs proposed above.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

None beyond kube-apiserver.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes.

Kubelet will open a watch on ClusterTrustBundle objects.  This watch will be
low-throughput. A similar watch is also opened from the KCM side.

###### Will enabling / using this feature result in introducing new API types?

ClusterTrustBundle (no per-cluster limit enforced)

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

A new API ClusterTrustBundle API object is created for the new `kubernetes.io/kube-apiserver-serving` signer, and there are
additional pod fields that the user sets to make use of the feature.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Use of the ClusterTrustBundle projected volume type will impact the [existing
SLOs](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/pod_startup_latency.md)
that cover startup latency for stateless schedulable pods (similar to the
existing effect from).

The definition of "stateless" will also need to be updated to include pods that
use ClusterTrustBundle volumes, unless it is intentional that those SLOs exclude
projected token volumes.

(Kubelet projected volumes seem problematic in general for the startup latency
SLOs --- if I create a pod that projects many large configmaps, I can introduce
arbitrary startup latency for my pod and cause an SLO breach.)

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Use of the ClusterTrustBundle objects by themselves should have negligible
resource impact.

When the ClusterTrustBundleProjection feature gate is enabled, Kubelet will open
a watch on all ClusterTrustBundle objects in the cluster.  We expect there to be
a low number of ClusterTrustBundle objects that does not scale with the number
of nodes or workloads in the cluster, although individual ClusterTrustBundle
objects could be large.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

When a user specifies a ClusterTrustBundle projected volume source, this places several files and links within the projected volume (one main file, but the atomic update package also places symlinked folders with versioned copies of the file).

On Linux, each projected volume is an independent tmpfs filesystem, so this is unlikely to lead to overall exhaustion of inodes on the node.

On Windows, "tmpfs" volumes appear to be translated to plain folders in the host filesystem, so there may be a risk of exhausting some node-wide filesystem resource.  However, this would still require the user to create many pods, each with thousands or more projected volume sources.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

If kube-apiserver becomes unavailable, then existing pods that have ClusterTrustBundle projected volume sources will continue to run with stale data in their projected files.  The impact should be minimal.

Even if Kubelet restarts containers or pods during the outage, their volume mounting code will continue to use stale data from the local informer cache.

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

None known.

###### What steps should be taken if SLOs are not being met to determine the problem?

If the API call latency SLO is being breached for operations on
ClusterTrustBundle operations, the operator can roll back the ClusterTrustBundle
and ClusterTrustBundleProjection feature gates, at the cost of losing access to
the feature.

If the pod startup latency SLO is being breached for pods that use
ClusterTrustBundle projected volume sources, the operator can roll back the
ClusterTrustBundle and ClusterTrustBundleProjection feature gates, at the cost
of losing access to the feature.

Detailed investigation of success rate and timings of ClusterTrustBundle
projected volume source operations can use the
`projectedvolumesource_refresh_count` and
`projectedvolumesource_refresh_duration` kubelet metrics.

## Implementation History

* 1.27 --- ClusterTrustBundle objects went to alpha behind a feature gate.
* 1.29 --- ClusterTrustBundle projected volume sources went to alpha behind a feature gate.
* 1.30 --- ClusterTrustBundles and ClusterTrustBundle projected volumes sources targeting beta behind a feature gate.

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

Item 1 requires a core object to describe trust anchors.  Item 2 could be
accomplished using CRDs and CSI drivers, but we eventually want these to be
standardized features.

### Use a ConfigMap rather than defining a new type

ConfigMaps have two problems in this application:

* They are namespace-scoped, whereas signers are cluster-scoped.
* They have poor human factors.  They are not often considered to contain
  security-critical data, and so may have wide update permissions assigned to
  them.

### Support for other certificate formats beyond PEM-wrapped DER-formatted X.509

PEM-wrapped DER-formatted X.509 is the lingua franca for TLS stacks to load
certificates.  The only exception I'm aware of is Java, which requires the
creation of a Java-specific keystore file.  In the future, we could add support
to the Kubelet clusterTrustBundle projected volume source to emit certificates
in the Java keystore format.

## Infrastructure Needed (Optional)

No additional infrastructure is needed.
