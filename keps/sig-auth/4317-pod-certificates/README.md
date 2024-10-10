# KEP-4317: Pod Certificates

<!-- generate with `hack/update-toc.sh`. -->
<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary and Motivation](#summary-and-motivation)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Design Details](#design-details)
  - [PodCertificateRequest Resource](#podcertificaterequest-resource)
    - [Why a New Type?](#why-a-new-type)
    - [API Definition](#api-definition)
  - [PodCertificate Projected Volume Sources](#podcertificate-projected-volume-sources)
    - [Issuance Flow](#issuance-flow)
    - [API Object Diff](#api-object-diff)
    - [Example certificate bundle written to the pod filesystem](#example-certificate-bundle-written-to-the-pod-filesystem)
  - [Kubernetes API client pod certificates](#kubernetes-api-client-pod-certificates)
    - [X.509 PodIdentity Extension](#x509-podidentity-extension)
    - [API Server PodIdentity Extension Support](#api-server-podidentity-extension-support)
    - [API Server Pod Client Certificate Signer](#api-server-pod-client-certificate-signer)
    - [Client-go Enhancements](#client-go-enhancements)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Using this feature as a third-party signer implementer](#using-this-feature-as-a-third-party-signer-implementer)
  - [Future expansion: HSM support for private keys](#future-expansion-hsm-support-for-private-keys)
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

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary and Motivation

The certificates.k8s.io API group provides a flexible, pluggable mechanism to
request X.509 certificates within a Kubernetes cluster, but actually making
certificates available to your workloads is left as an exercise for the reader.

This KEP defines two pieces of fundamental machinery:
* A PodCertificateRequest API type, analogous to CertificateSigningRequest but
  narrowly scoped to the job of issuing certificates directly to pods.  A
  PodCertificateRequest is fundamentally a statement that "pod X is requesting a
  certificate from signer Y".
* A PodCertificate projected volume source that instructs the kubelet to handle
  provisioning a key and certificate chain on behalf of a pod.
Taken together, this machinery makes it feasible to securely and automatically
deliver X.509 certificates to every pod in a cluster, without imposing an
unreasonable burden on application developers or cluster administrators.  

As a first application of this machinery, this KEP defines additional machinery
that allows pods to automatically receive mTLS client certificates suitable for
authenticating to kube-apiserver, with the same attributes provided by bound
service account tokens:
* A PodIdentity X.509 extension to allow an X.509 certificate to carry all
  details of a pod's identity, including its UIDs and the node it is running on.
* A new TLS user-mapping capability in kube-apiserver to understand client
  certificates that embed PodIdentity extensions.
* A new PodCertificateRequest signer
  (`kubernetes.io/kube-apiserver-client-pod`), shipped in
  kube-controller-manager, to issue compatible client certificate to pods.
* A new client-go configuration, `rest.InClusterPodCertificateConfig()` that can
  be used to load a credential bundle from disk and use it to connect to
  kube-apiserver.

While this first application motivates some of the design decisions in this KEP,
it is explicitly intended that PodCertificateRequests and PodCertificate
projected volume sources will be useful to third-party projects that want to
easily and securely issue X.509 certificates to workloads running in a
Kubernetes cluster.

There is a draft implementation of the design outlined in this KEP:
https://github.com/kubernetes/kubernetes/pull/127251

## Motivation

### Goals

**Ensure that certificate issuance can be accomplished without relying on bearer
tokens at any step**:  One of the most significant benefits of mTLS
authentication is that the credential is split into two pieces (private key and
certificate), only one of which is ever transmitted to the counterparty (and
thus subject to eavesdropping or theft by the counterparty).  In order to
preserve this guarantee, it's important that bearer tokens are not involved *at
any step*, otherwise the security of the scheme devolves to just being a bearer
token.  In this KEP, as long as both kubelet and the signer implementation
authenticate to kube-apiserver using their own separately-bootstrapped client
certificates, then no bearer tokens are involved in certificate issuance.

**Make it easy to ship secure third-party signer implementations**:  Securely
implementing and operating a third-party signer implementation today requires
intimate familiarity with Kubernetes' security model, otherwise your signer may
negate Kubernetes' node restriction security boundary.  This KEP moves
responsibility for enforcing node restriction on PodCertificateRequests into
kube-apiserver, so that signer implementations do not need to consider it.

**Support issuance of both client and server certificates**: While this KEP does
not include a concrete signer that issues dedicated server certificates,
third-party signer implementations should be able to do so on top of the
machinery described in this KEP.

### Non-Goals

**Specify a solution for pod-to-pod mTLS in core Kubernetes**: Pod-to-pod mTLS
(aka service mesh) is a rapidly-evolving area.  While all of the machinery
described in this KEP is expected (and is designed to be) useful for service
mesh implementations, it doesn't make sense to try and standardize pod-to-pod
mTLS at the core Kubernetes layer today.

**Support HSM-backed private keys**: In this proposal, private keys are
contained to the node (held only in kubelet memory and tmpfs projected volumes),
but they can still be extracted from a compromised workload (or compromised
node).  Storing the private keys in an HSM closes this loophole, but will require
an additional interface for kubelet to expose remote signing capabilities to workloads.

**Be 100% compatible with software in the field today**: The primary example in
this KEP is that the private key and certificate chain are delivered in a single
credential-bundle file.  This is based on implementation and operation
experience of a CSI-based certificates solution that delivered private keys and
certificate chains in separate files, as expected by most commodity webservers
today.  While this is an attractive idea, automatically rotating these separate
files is fraught.  Even if both files' content is updated atomically, skewed
reads in the workload can result in hard-to-diagnose errors if the workload
reads the key before rotation and the certificate chain after rotation (or
vice-versa).

**Support private keys shared across multiple pods**: Sharing private keys and
certificates across multiple pods may be required in certain applications.
However, this use case is well-served by Secret projected volume sources.

**Support human-in-the-loop approval for certificate requests**: This KEP
assumes that PodCertificateRequest issuance is fully-automated, because this
process blocks pod startup for pods that use PodCertificate projected volume
sources.  PodCertificateRequests do not separate the approval and issuance
steps, unlike CertificateSigningRequests.

**Standardize on using the SPIFFE SVID format in core Kubernetes**: The
certificates issued by `kubernetes.io/kube-apiserver-client-pod` can be made
compatible with SPIFFE in the future, since they currently don't include any
Subject Alternate Name entries.

**Support identity delegation or impersonation**: We do not intend to support
issuing certificates that workloads can use as intermediate CA certificates in
order to issue further certificates, nor do we intend to support one privileged
workload minting certificates on behalf of other workloads.

## Design Details

### PodCertificateRequest Resource

PodCertificateRequest is a new API type in certificates.k8s.io/v1alpha1.  It's a
stripped-down version of CertificateSigningRequest that is tailored directly to
the pod certificate issuance use case.

PodCertificateRequest is fundamentally a statement that "Pod X is requesting a
certificate from Signer Y".  The signer is in complete control of the form of
the issued certificate.  As such, PodCertificateRequest objects don't contain a
typical X.509 CSR.  Instead, they contain:
* A public key, PKIX-serialized.  This key must be an RSA key with modulus size
  3072 or 4096, an ECDSA key with curve P256 or P384, or an ED25519 key.
* A proof of possession of the corresponding private key.
  * If the key is an RSA key, then the signature is over the ASCII bytes of the
	  pod UID, using RSASSA-PKCS1-V1_5-SIGN from RSA PKCS #1 v1.5 (as implemented
	  by the golang function crypto/rsa.SignPKCS1v15).
  * If the key is an ECDSA key, then the signature is as described by [SEC 1,
	  Version 2.0](https://www.secg.org/sec1-v2.pdf) (as implemented by the golang
	  library function crypto/ecdsa.SignASN1)
  * If the key is an ED25519 key, then the signature is as described by the
    [ED25519 Specification](https://ed25519.cr.yp.to/) (as implemented by
    the golang library crypto/ed25519.Sign).
* Details of the requesting Pod's identity:
  * Namespace (indicated by the namespace the PodCertificateRequest is created in).
  * Pod name
  * Pod UID
  * Service account name
  * Service account UID
  * Node name
  * Node UID
* A requested maximum duration for the certificate lifetime.
* The signer name from which a certificate is being requested.
* (In the status subobject): An issued certificate chain, stored as a single
  PEM-formatted string.  If there is more than one issued certificate, the first
  element of the chain is assumed to be the leaf certificate, followed by any
  necessary intermediates in leaf-to-root order.
* (In the status subobject): A timestamp at which time the requester that made
  the PodCertificateRequest should begin trying to refresh the certificate.

PodCertificateRequest objects have a simplified lifecycle compared to
CertificateSigningRequests.  There are no separate approval and signing stages.
They can only be issued (and thus implicitly approved), denied (represented by a
"Denied" status), or failed (represented by a "Failed" status).

Similar to CertificateSigningRequests, controllers that want to issue a
certificate in response to a PodCertificateRequest need to have two separate
RBAC permissions:
* `update` on `podcertificaterequests/status`.
* `sign` on the requested signer.
Similar to CertificateSigningRequest, we will also support wildcards on the RBAC signer permissions.

Once a signer controller decides that a PodCertificateRequest is addressed to it
by matching on the signer name, it should take one of the following actions:
* Issue the certificate by updating the status subresource of the
  PodCertificateRequest, setting both the `certificateChain` and
  `beginRefreshAt` fields.
* Reject the certificate by updating the status subresource of the
  PodCertificateRequest, adding a "Denied" condition.
* Indicate failure in issuing the certificate by updating the status subresource
  of the PodCertificateRequest, adding a "Failed" condition.

PodCertificateRequest validation logic will:
* Confirm that the public key is one of the supported key types.
* Confirm that the proof-of-possession is valid.
* Confirm that the issued chain (if one is set) consists of valid certificates,
  that do indeed form a chain to each other.
* Confirm that neither or both of the certificateChain and beginRefreshAt
  timestamp are set.
* Ensure that all fields in PodCertificateRequest.Spec are immutable.
* Ensure that all fields in PodCertificateRequest.Status are immutable once the
  request has been issued, denied, or failed.

The `maxExpirationSeconds` field will have the following logic:
* kube-apiserver will be configurable with an optional maximum duration to allow
  for certificates issued via PodCertificateRequests, specified in the
  `--pod-certificate-request-max-duration` flag.  The syntax of this flag will
  support setting different maximum durations for different signers, with some
  level of wildcarding.  If a signer is not covered by the flag value,
  kube-apiserver will assume a maximum duration of 24 hours.
* If the creator of the PodCertificateRequest left `maxExpirationSeconds` at 0,
  then kube-apiserver will default it to 24 hours.
* kube-apiserver will then truncate `maxExpirationSeconds` to the signer's
  configured maximum duration.
* kube-apiserver will enforce a minimum value of 3600 (1 hour) on
  `maxExpirationseconds`.
* When the signer implementation isuses a certificate into the
  PodCertificateRequest, kube-apiserver will enforce that the issued certificate's life span is less than `maxExpirationSeconds`.
This logic ensures that both pod authors and cluster operators have control to restrict that max TTL of certs that are being issued.

The noderestriction admission plugin checks the consistency of the information
in created PodCertificateRequest objects:
* The namespace, pod name/UID, service account name/UID, and node name/UID must
  correspond to a real pod that currently exists in the cluster.
* The pod should be scheduled onto the node named in the PodCertificateRequest.
* The pod should be in the Pending or Running phases.
* The PodCertificateRequest must have been created by the user
  `system:node:foo`, where `foo` is the node named in the PodCertificateRequest
  spec.

This upholds the Kubernetes node isolation guarantee by ensuring that
compromising a single node only results in compromises of workloads that have
actually been scheduled onto that node.

During alpha, the ability to store PodCertificate objects (and the node
restriction admission logic) will be gated by the PodCertificateRequests feature
gate.

#### Why a New Type?

The following constraints led to the adoption of a new API type, as opposed to
extending CertificateSigningRequest:
1. We wanted the pod details to be legible without having to parse an ASN.1 CSR.
2. We wanted the pod details (especially the associated node name) to be
  available for use in an informer selector.  While these could be surfaced as
  labels on CertificateSigningRequest objects, labels mutable.
3. We wanted pod details to be immutable.
4. We wanted to avoid putting Pod Certificate-specific fields in the generic
  CertificateSigningRequest.
5. Avoid confusing existing signer implementations with pod information they
   don't understand.

The available options were:

* Label CertificateSigningRequests with node/pod labels by convention.  This
  didn't meet goal 3.
* Hoist pod fields into CSR spec.  This didn't meet goal 4.
* Embed pod details into CSR spec.request field.  This didn't meet goals 1 and
  2.
* New dedicated PodCertificateRequest type. Meets all goals.

#### API Definition

```go

// PodCertificateRequest encodes a pod requesting a certificate from a given
// signer.
//
// Kubelets use this API to implement podCertificate projected volumes
type PodCertificateRequest struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// spec contains the details about the certificate being requested.
	Spec PodCertificateRequestSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`

	// status contains the issued certificate, and a standard set of conditions.
	// +optional
	Status PodCertificateRequestStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// PodCertificateRequestSpec describes the certificate request.  All fields are
// immutable after creation.
type PodCertificateRequestSpec struct {
	// signerName indicates the request signer.
	SignerName string `json:"signerName" protobuf:"bytes,1,opt,name=signerName"`

	// maxExpirationSeconds is the requested lifetime for the certificate.  This
  // should be treated as a maximum, as both kube-apiserver and the signer 
  // implementation may substitute a shorter expiration.
  //
  // If this field is set to 0 during creation of the PodCertificateRequest, then 
  // kube-apiserver will set it to 24 hours.  kube-apiserver will then shorten
  // the value to the maximum expiration configured for the requested signer.
	MaxExpirationSeconds int32 `json:"expirationSeconds" protobuf:"bytes,2,opt,name=expirationSeconds"`

	// podName is the name of the pod into which the certificate will be mounted.
	PodName string `json:"podName" protobuf:"bytes,3,opt,name=podName"`
	// podUID is the UID of the pod into which the certificate will be mounted.
	PodUID types.UID `json:"podUID" protobuf:"bytes,4,opt,name=podUID"`

	// serviceAccountName is the name of the service account the pod is running as.
	ServiceAccountName string `json:"serviceAccountName" protobuf:"bytes,5,opt,name=serviceAccountName"`

	// serviceAccountUID is the UID of the service account the pod is running as.
	ServiceAccountUID types.UID `json:"serviceAccountUID" protobuf:"bytes,6,opt,name=serviceAccountUID"`

	// nodeName is the name of the node the pod is assigned to.
	NodeName types.NodeName `json:"nodeName" protobuf:"bytes,7,opt,name=nodeName"`

	// nodeUID is the UID of the node the pod is assigned to.
	NodeUID types.UID `json:"nodeUID" protobuf:"bytes,8,opt,name=nodeUID"`

	// pkixPublicKey is the PKIX-serialized public key the signer should issue
	// the certificate to.
	//
	// The key must be one of RSA3072, RSA4096, ECDSAP256, ECDSAP384, or ED25519.
  // Note that this list may be expanded in the future.
	//
	// Signer implementations do not need to support all key types supported by
	// kube-apiserver and kubelet.  If a signer does not support the key type
	// used for a given PodCertificateRequest, it should deny the request, with
	// a reason of UnsupportedKeyType.  It may also suggest a key type that it
	// does support by attaching an additional SuggestedKeyType condition, with
	// its reason field set to the suggested key type identifier.
	PKIXPublicKey []byte `json:"pkixPublicKey" protobuf:"bytes,9,opt,name=pkixPublicKey"`

	// proofOfPossession proves that the requesting Kubelet holds the private
	// key corresponding to pkixPublicKey.
  //
  // It is contructed by signing the ASCII bytes of the pod's UID using 
  // `PKIXPublicKey`.
	//
	// kube-apiserver validates the proof of possession during creation of the
	// PodCertificateRequest.
	//
	// If the key is an RSA key, then the signature is over the ASCII bytes of
	// the pod UID, using RSASSA-PKCS1-V1_5-SIGN from RSA PKCS #1 v1.5 (as
	// implemented by the golang function crypto/rsa.SignPKCS1v15).
	//
	// If the key is an ECDSA key, then the signature is as described by [SEC 1,
	// Version 2.0](https://www.secg.org/sec1-v2.pdf) (as implemented by the
	// golang library function crypto/ecdsa.SignASN1)
  //
  // If the key is an ED25519 key, the the signature is as described by the
  // [ED25519 Specification](https://ed25519.cr.yp.to/) (as implemented by
  // the golang library crypto/ed25519.Sign).
	ProofOfPossession []byte `json:"proofOfPossession" protobuf:"bytes,10,opt,name=proofOfPossession"`
}

type PodCertificateRequestStatus struct {
	// conditions applied to the request. Known conditions are "Denied",
	// "Failed", and "SuggestedKeyType".
  //
  // If the request is denied with `Reason=UnsupportedKeyType`, the signer
  // may have suggested a key type that will work in the `Reason` field of a
  // `SuggestedKeyType` condition.
	//
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" protobuf:"bytes,1,rep,name=conditions"`

	// certificateChain is populated with an issued certificate by the signer.
	// This field is set via the /status subresource. Once populated, this field
	// is immutable.
	//
	// If the certificate signing request is denied, a condition of type
	// "Denied" is added and this field remains empty. If the signer cannot
	// issue the certificate, a condition of type "Failed" is added and this
	// field remains empty.
	//
	// Validation requirements:
  //  1. certificateChain must consist of one or more PEM-formatted certificates.
  //  2. Each entry must be a valid PEM-wrapped, DER-encoded ASN.1 Certificate as
  //     described in section 4 or RFC5280.
	//
	// If more than one block is present, and the definition of the requested
	// spec.signerName does not indicate otherwise, the first block is the
	// issued certificate, and subsequent blocks should be treated as
	// intermediate certificates and presented in TLS handshakes.  When
	// projecting the chain into a pod volume, kubelet will preserve the exact
  // contents of certificateChain.
  //
	// +optional
	CertificateChain string `json:"certificateChain,omitempty" protobuf:"bytes,2,opt,name=certificateChain"`

	// issuedAt is the time at which the signer issued the certificate.  This
	// field is set via the /status subresource.  Once populated, it is
	// immutable.  The signer must set this field at the same time it sets 
  // certificateChain.
	//
	// +optional
	IssuedAt *metav1.Time `json:"issuedAt,omitempty" protobuf:"bytes,3,opt,name=issuedAt"`

	// notBefore is the time at which the certificate becomes valid.  This field
	// is set via the /status subresource.  Once populated, it is immutable.  The 
  // signer must set this field at the same time it sets certificateChain.
	//
	// +optional
	NotBefore *metav1.Time `json:"notBefore,omitempty" protobuf:"bytes,4,opt,name=notBefore"`

	// beginRefreshAt is the time at which the kubelet should begin trying to
	// refresh the certificate.  This field is set via the /status subresource,
	// and must be set at the same time as certificateChain.  Once populated,
	// this field is immutable.
  //
  // This field is only a hint.  Kubelet may start refreshing before or after
  // this time if necessary.
	//
	// +optional
	BeginRefreshAt *metav1.Time `json:"beginRefreshAt,omitempty" protobuf:"bytes,5,opt,name=beginRefreshAt"`

	// notAfter is the time at which the certificate expires.  This field is set
	// via the /status subresource.  Once populated, it is immutable.  The 
  // signer must set this field at the same time it sets certificateChain.
	//
	// +optional
	NotAfter *metav1.Time `json:"notAfter,omitempty" protobuf:"bytes,6,opt,name=notAfter"`
}

// Well-known condition types for PodCertificateRequests
const (
	// Denied indicates the request was denied by the signer.
	PodCertificateRequestConditionTypeDenied string = "Denied"
	// Failed indicates the signer failed to issue the certificate.
	PodCertificateRequestConditionTypeFailed string = "Failed"
	// SuggestedKeyType is an auxiliary condition that a signer can attach if it
	// denied the request due to an unsupported key type.
	PodCertificateRequestConditionTypeSuggestedKeyType string = "SuggestedKeyType"
)

// Well-known condition reasons for PodCertificateRequests
const (
	// UnsupportedKeyType should be set on "Denied" conditions when the signer
	// doesn't support the key type of publicKey.
	PodCertificateRequestConditionUnsupportedKeyType string = "UnsupportedKeyType"
)
```

### PodCertificate Projected Volume Sources

The PodCertificate projected volume source instructs kubelet to inject a
credential bundle consisting of a private key and certificate chain into the
container fileystem.  The kubelet handles generating the key, requesting a
certificate from the named signer, and renewing the certificate before it
expires.

The kubelet supports only a few key generation strategies, each named by a
string that encapsulates both the key type and any parameters necessary (for
example, RSA modulus size).  The intention is to offer a tasting menu of
reasonable key choices, rather than offering a flexibility to a wide variety of
parameters. Key types supported by kubelet are "RSA3072", "RSA4096",
"ECDSAP256", "ECDSAP384", and "ED25519".  If no key type is specified, kubelet
defaults to "ECDSAP256".

Once the key is generated, kubelet creates a PodCertificateRequest with the
details of the pod that is mounting the volume.  The kubelet then waits for the
PodCertificateRequest to be issued, denied, or failed.

If the PodCertificateRequest is issued, then the kubelet writes both the
generated private key and the issued certificate chain into the pod filesystem
as a single file with multiple PEM blocks.  The first block in the PEM file
contains the private key, and the remaining blocks contain the certificate chain
in the same order they were in the PodCertificateRequest.

If the PodCertificateRequest is denied, the kubelet treats this as a fatal error
and returns an error message from the volume setup.  It is assumed that if a
signer marks a PodCertificateRequest as denied, then retrying the request will
not make it change its mind.  The Pod will not start up.

If the PodCertificateRequest is failed, the kubelet also treats this as a
permanent error (although a Failed status does not necessarily indicate a
permanent problem with the chosen signer).  We assume that most pods are created
by a higher-level workload controller (Deployment, Job, DaemonSet), which will
supply the necessary retries by creating a new Pod.

The kubelet will begin trying to rotate the certificate at the time indicated by
the signer via the `PodCertificateRequest.Status.BeginRefreshAt` timestamp. The
kubelet will apply a randomized jitter of up to five minutes, to prevent
multiple volumes from gradually syncing their refresh periods. Errors in the
certificate issuance process are treated as non-fatal (logged but not returned
from the volume SetUp function) while the original certificate is still valid.
Once the original certificate has expired, errors will be returned from the
volume SetUp function, because it's better for the pod to noisily go unhealthy
than to continue to run with an expired certificate.

#### Issuance Flow
```mermaid
sequenceDiagram
    participant kubelet as kubelet
    participant api as kube-apiserver
    participant signer as signer

    note over kubelet: Begin projected volume setup
    kubelet->api: Create PodCertificateRequest X
    note over api: Enforce node restriction
    api->signer: See PodCertificateRequest X
    note over signer: Decide whether or not to issue certificate
    signer->api: Update /status<br>Set CertificateChain, BeginRefreshAt
    kubelet->api: See PodCertificateRequest X issued
    note over kubelet: Write credential bundle to pod filesystem

    note over kubelet: Pod starts
    
    
    loop While pod is running
      note over kubelet: BeginRefreshAt time reached
      kubelet->api: Create PodCertificateRequest Y
      note over api: Enforce node restriction
      api->signer: See PodCertificateRequest Y
      note over signer: Decide whether or not to issue certificate
      signer->api: Update /status<br>Set CertificateChain, BeginRefreshAt
      kubelet->api: See PodCertificateRequest Y issued
      note over kubelet: Write credential bundle to pod filesystem
    end 
```

#### API Object Diff

```go
// PodCertificateProjection provides a private key and X.509 certificate in the
// pod filesystem.
type PodCertificateProjection struct {
	// Kubelet's generated CSRs will be addressed to this signer.
	SignerName string `json:"signerName,omitempty" protobuf:"bytes,1,rep,name=signerName"`

	// The type of keypair Kubelet will generate for the pod.
	//
	// Valid values are "RSA3072", "RSA4096", "ECDSAP256",
	// "ECDSAP384", and "ED25519".  If left empty, Kubelet defaults to "ECDSAP256".
	KeyType string `json:"keyType,omitempty" protobuf:"bytes,2,rep,name=keyType"`

  // The maximum certificate lifetime that the application wants.
  //
  // Kubelet will copy this value into the PodCertificateRequest that it creates.
  //
  // This field is a *request*, as both kube-apiserver and the signer 
  // implementation can choose to set a shorter lifetime on the certificate.
  MaxExpirationSeconds int32 `json:"maxExpirationSeconds,omitempty" protobuf:"bytes,3,rep,name=maxExpirationSeconds"`

	// Write the credential bundle at this path in the projected volume.
	//
	// The credential bundle is a single file that contains multiple PEM blocks.
	// The first PEM block is a PRIVATE KEY block, containing a PKCS#8 private
	// key.
	//
	// The remaining blocks are CERTIFICATE blocks, containing the issued
	// certificate chain from the signer (leaf and any intermediates).
	//
	// Using credentialBundlePath lets your Pod's application code make a single 
	// atomic read that retrieves a consistent key and certificate chain.  If you 
	// project them to separate files, your application code will need to 
	// additionally check that the leaf certificate was issued to the key.
	//
	// Mutually exclusive with keyPath and certificateChainPath.
	CredentialBundlePath string `json:"credentialBundlePath,omitempty" protobuf:"bytes,3,rep,name=credentialBundlePath"`

	// Write the key at this path in the projected volume.
	//
	// Mutually exclusive with credentialBundlePath.
	//
	// When using keyPath and certificateChainPath, your application needs to check 
	// that the key and leaf certificate are consistent, because it is possible to 
	// read the files mid-rotation.
	KeyPath string `json:"keyPath,omitempty" protobuf:"bytes,4,rep,name=keyPath"`

	// Write the certificate chain at this path in the projected volume.
	//
	// Mutually exclusive with credentialBundlePath.
	CertificateChainPath string `json:"certificateChainPath,omitempty" protobuf:"bytes,5,rep,name=certificateChainPath"`
}


// ...

// Projection that may be projected along with other supported volume types
type VolumeProjection struct {
  // ...

	// Projects an auto-rotating credential bundle (private key and certificate
	// chain) that the pod can use either as a TLS client or server.
	//
	// Kubelet generates a private key and uses it to send a
	// PodCertificateRequest to the named signer.  Once the signer approves the
	// request and issues a certificate chain, Kubelet writes the key and
	// certificate chain to the pod filesystem.  The pod does not start until
	// certificates have been issued for each podCertificate projected volume
	// source in its spec.
	//
	// Kubelet will begin trying to rotate the certificate at the time indicated
	// by the signer using the PodCertificateRequest.Status.BeginRefreshAt
	// timestamp.
	//
	// Kubelet can write a single file, indicated by the credentialBundlePath
	// field, or separate files, indicated by the keyPath and
	// certificateChainPath fields.
	//
	// The credential bundle is a single file in PEM format.  The first PEM
	// entry is the private key (in PKCS#8 format), and the remaining PEM
	// entries are the certificate chain issued by the signer (typically,
	// signers will return their certificate chain in leaf-to-root order).
	//
	// Prefer using the credential bundle format, since your application code
	// can read it atomically.  If you use keyPath and certificateChainPath,
	// your application must make two separate file reads. If these coincide
	// with a certificate rotation, it is possible that the private key and leaf
	// certificate you read may not correspond to each other.  Your application
	// will need to check for this condition, and re-read until they are
	// consistent.
	//
	// The named signer controls chooses the format of the certificate it
	// issues; consult the signer implementation's documentation to learn how to
	// use the certificates it issues.
	//
	// +featureGate=PodCertificateProjection
  // +optional
	PodCertificate *PodCertificateProjection `json:"podCertificate,omitempty" protobuf:"bytes,6,opt,name=podCertificate"`
}
```

#### Example certificate bundle written to the pod filesystem

```
-----BEGIN PRIVATE KEY-----
<Omitted, so that automation stops mailing the KEP authors to complain>
-----END PRIVATE KEY-----
-----BEGIN CERTIFICATE-----
MIIDSDCCAjCgAwIBAgIQRleU9YIlOqvluwFKzauVoTANBgkqhkiG9w0BAQsFADAV
MRMwEQYDVQQDEwprdWJlcm5ldGVzMB4XDTIzMTAyOTA1NDU1NVoXDTIzMTAzMDA1
NTAyMlowADCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAMy7zINPgkDc
kCrb2EEuxvt6PLjthjnlq9uxl79dhQGl7EgIEgA4Kc+WKZ6uJsYJlJazhuc6egDY
7nr8s+MATtPV8hV6Rgd/8bMB2ktq1RVhEJ4kRAodSLDLnD4e0IhIZpXSK5DBCXF6
dRotiCEIGpCQmdMTDYZez6zTfN3E1yuhyw7bSKc2OJj6+MYly6oQwE0T7WVDNqoa
Ev2ZdHZjePkEhXvGRvm1fyxK4h0UJsaT38/COUW/Dal7Fx6VcJeY2QHI28W+NYP3
AOdx/7Hc6bbPcYZI/c1HPolUhoX6huiltGMn3yOLsaGjajwXUoE2aKsy3AHxeIkQ
4UlKQSabpb8CAwEAAaOBqDCBpTAOBgNVHQ8BAf8EBAMCA6gwEwYDVR0lBAwwCgYI
KwYBBQUHAwIwHwYDVR0jBBgwFoAUuQ7U2/jMkK4nY5R2rPvjyCiC2SQwXQYJKwYB
BAGDwlMBBFAwTgwHZGVmYXVsdAwHZGVmYXVsdAwHcGVtLXBvZAwkMjc3YjljNDkt
MzljMy00ZjIzLWIwYjEtZTg4OWVmOTBjMGIyDAtraW5kLXdvcmtlcjANBgkqhkiG
9w0BAQsFAAOCAQEAQd1G2CHdGPv9Lu9/SD7HPNXwI0K4JrXsWxJQ7Q16IVAuuWIY
mOdumO9L5PtTrV1widNm99oMEtztQHZoHHIkkypjfW+BB8XwO5er+FpcUCuvZ3w/
g9Zk9cZUqCdWER0UDl4RCLcS+M6QSq0nUg+JTZKRGJ5fjns3vZHA9Nswp4XRvjt8
yf6RY6oEJoa10QxB7YiNVmVkqLifhkQiF0Bmz7tONbsjceZ030Cs/wjdssnNzRGL
UdFfcDKw4QbMGqntJXPFqu86lk0IJwir+aTk/31yDZX1ZyOK0Y/SI7ZVVDeETTA9
k68gmm9HQCdRI3stW7TC3lB0Cd1XSSMUISIP0g==
-----END CERTIFICATE-----
```

### Kubernetes API client pod certificates

#### X.509 PodIdentity Extension

The CNCF has an existing ASN.1 Private Enterprise Number registered for use by
the Kubernetes project (`1.3.6.1.4.1.57683`).  In order to cleanly embed pod
identity information into an X.509 certificate or certificate request, we can
define a new ASN.1 OID (currently `1.3.6.1.4.1.57683.1`, though we need to
define a management policy for the PEN) and specify that it contains the
following DER-encoded structure:

```go
type podIdentityASN1 struct {
	Namespace          string `asn1:"utf8,tag:0"`
  NamespaceUID       string `asn1:"utf8,tag:1"`
	ServiceAccountName string `asn1:"utf8,tag:2"`
	ServiceAccountUID  string `asn1:"utf8,tag:3"`
	PodName            string `asn1:"utf8,tag:4"`
	PodUID             string `asn1:"utf8,tag:5"`
	NodeName           string `asn1:"utf8,tag:6"`
	NodeUID            string `asn1:"utf8,tag:7"`
}
```

This extension is defined by the behavior of the go `encoding/asn1` package in
go 1.23.  A translation to ASN.1 notation is provided for convenience, but in
the case of conflict, the go version controls:
```asn1
PodIdentity SEQUENCE {
  namespace [0] UTF8String,
  namespaceUID [1] UTF8String,
  serviceAccountName [2] UTF8String,
  serviceAccountUID [3] UTF8String,
  podName [4] UTF8String,
  podUID [5] UTF8String,
  nodeName [6] UTF8String,
  nodeUID [7] UTF8String
}
```

Utility functions for embedding and extracting this extension from `crypto/x509`
Certificate objects are defined in `k8s.io/component-helpers/kubernetesx509`.

This extension, and the utility functions for manipulating it, are developed for
use by core Kubernetes components, although third parties could make use of
them.  The extension should only be changed in backwards-compatible ways.

#### API Server PodIdentity Extension Support

The kube-apiserver is extended to authenticate TLS client certificates that
represent service account identities in the following format:

* Are signed by root certificates provided by a new `--pod-client-ca-file` flag
  (with automatic reload when the contents of the file change),
* Contain no SubjectAlternateName data, and
* Contain an X.509 PodIdentity extension.

These certificates are translated to an authenticator.Response that has:
* `User.Name` set to
  `system:serviceaccount:{PodIdentity.Namespace}:{PodIdentity.ServiceAccountName}`.
* `User.UID` set to `PodIdentity.ServiceAccountUID`.
* `User.Extra["authentication.kubernetes.io/pod-name"]` set to
  `{PodIdentity.PodName}`
* `User.Extra["authentication.kubernetes.io/pod-uid"]` set to
  `{PodIdentity.PodUID}`
* `User.Extra["authentication.kubernetes.io/node-name"]` set to
  `{PodIdentity.NodeName}`
* `User.Extra["authentication.kubernetes.io/node-uid"]` set to
  `{PodIdentity.NodeUID}`
* `User.Extra["authentication.kubernetes.io/credential-id"]` set to
  `X509SHA256=<sha256-hash-of-the-certificate>`.

The net effect is that authenticating via a PodIdentity-bearing cert has the
same object-bound behavior as authenticating via a bound service account token.

During Alpha, this feature will be gated by the
`PodCertificateKubeApiserverClientAcceptance` feature gate.

#### API Server Pod Client Certificate Signer

The kube-controller-manager is extended to support a new certificate signer,
`kubernetes.io/kube-apiserver-client-pod`.

The controller for this signer name automatically issues certificates for
PodCertificateRequests that are addressed to it.  It performs no additional
validation on PodCertificateRequests beyond the validation enforced by
kube-apiserver.

The certificates it issues are meant to be understood by kube-apiserver as
described above.  They:
* Have a PodIdentity extension (critical) that contains the information from
  `PodCertificateRequest.Spec`.
* Key usages consistent with the certificate being used as a TLS client
  certificate.
* It uses a CA specified by a new pair of private key / root certificate flags
  (although implementations are free to set this flag to point at the CA as
  another signer).
* It issues certificates that are valid for the lesser of
  `Spec.MaxExpirationSeconds` or 24 hours, and tells Kubelet to begin refreshing
  them at the lesser of `0.5 * Spec.MaxExpirationSeconds` or 12 hours.
* Are backdated by 5 minutes to limit clock skew issues.

During Alpha, this feature is gated by the
`PodCertificateKubeApiserverClientIssuance` feature gate.

Because kube-apiserver will interpret the PodIdentity extension as expressing an
object binding between the presented certificate and the pod named in the
extension, once a pod is no longer running, all certificates that were issued to
it will become invalid.  This, along with the short lifetime of the
certificates, removes the need for a more permanent certificate revocation
mechanism.

#### Client-go Enhancements

Client-go will support a new `rest.InClusterPodCertificateConfig()`
configuration that takes the location of a credential bundle in the filesystem,
and picks up changes to the bundle as credential rotation occur.

### Risks and Mitigations

**Scalability**: In the current design and draft implementation, kubelet holds
the private keys for each pod on the node in memory.  If kubelet restarts, then
the private keys are lost, and kubelet will need to immediately generate new
keys and request new certificates for each volume on the pod.  If kubelet
crashloops, this can generate high CertificateSigningRequest traffic on
kube-apiserver.  This can potentially be mitigated by giving kubelet the
capability to read back the existing key from the projected volume on startup.

### User Stories

#### Story 1

I'm an application developer building a Kubernetes controller that I want to
deploy into my cluster.  I want to make sure that my controller securely
authenticates to kube-apiserver using mTLS.

I add a new projected volume to my pod, containing the information that
client-go expects at a well-known path.  I set
`automountServiceAccountToken=false` because I want to ensure that my pod is not
using bearer tokens for authentication.
```yaml
apiVersion: v1
kind: Pod
metadata:
  namespace: default
  name: pod-certificates-example
spec:
  restartPolicy: OnFailure
  automountServiceAccountToken: false
  containers:
  - name: main
    image: debian
    command: ['sleep', 'infinity']
    volumeMounts:
    - name: kube-apiserver-client-certificate
      mountPath: /var/run/secrets/kubernetes.io/serviceaccount
  volumes:
  - name: kube-apiserver-client-certificate
    projected:
      sources:
      - podCertificate:
          signerName: "kubernetes.io/kube-apiserver-client-pod"
          credentialBundlePath: credentialbundle.pem
      - configMap:
          localObjectReference: kube-root-ca.crt
          items:
          - key: ca.crt
            path: kube-apiserver-root-certificate.pem
      - downwardAPI:
          items:
          - path: namespace
            fieldRef:
              apiVersion: v1
              fieldPath: metadata.namespace
```

I ensure that my pod uses client-go's `rest.InClusterPodCertificateConfig()`,
and point it at the location of my credential bundle in the filesystem.

#### Using this feature as a third-party signer implementer

### Future expansion: HSM support for private keys

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

Unit test list (informed by draft implementation):
* pkg/api/pod/util_test.go
* pkg/apis/certificates/validation_test.go
* pkg/apis/core/validation/validation_test.go
* pkg/controller/certificates/authority/authority_test.go
* pkg/controller/certificates/podcertificaterequest_controller_test.go
* pkg/controller/certificates/signer/signer_test.go
* pkg/kubelet/podcertificate/podcertificate_manager_test.go
* pkg/registry/certificates/podcertificaterequest/storage/storage_test.go
* pkg/registry/certificates/podcertificaterequest/strategy_test.go
* pkg/volume/projected/projected_test.go
* plugin/pkg/admission/noderestriction/admission_test.go
* plugin/pkg/auth/authorizer/rbac/bootstrappolicy/controller_policy_test.go
* plugin/pkg/auth/authorizer/rbac/bootstrappolicy/policy_test.go
* staging/src/k8s.io/apiserver/pkg/authentication/request/x509/x509_test.go
* staging/src/k8s.io/client-go/rest/config_test.go
* staging/src/k8s.io/component-helpers/kubernetesx509/kubernetesx509_test.go

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

- `<package>`: `<date>` - `<test coverage>`

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
-->

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

* test/e2e/auth/projected_podcertificate.go: Checks end-to-end issuance and
  usage of pod certificates issued by `kubernetes.io/kube-apiserver-client-pod`.
  The test will create an agnhost pod with a mounted credential bundle, and
  agnhost will use the credential bundle to connect to kube-apiserver.

### Graduation Criteria

For Alpha:
* The feature will be gated behind feature flags and alpha API enablement.
* An e2e test will exercise the feature in order to prevent regressions.
* Interested users can experiment with the feature using a local Kubernetes
  distribution such as kind.
* Interested platform providers can experiment with integrating the feature into
  their environment.

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

This feature involves interaction among three components:
* kubelet
* kube-apiserver
* signing controllers (hosted in kube-controller-manager or elsewhere)

Initially, the skews we need consider are:
* (`kube-apiserver>=1.32`, `kubelet<1.32`, `kube-controller-manager<1.32`): If a
  pod with a PodCertificate projected volume is created, it may get scheduled
  onto a node where kubelet doesn't understand PodCertificate projected volumes.
  This kubelet will be unaware of the volume, and beginning running the pod
  without mounting any certificates.  If the application inside the pod is
  expecting the certificates to be present, it will probably crash.
* (`kube-apiserver>=1.32`, `kubelet<1.32`, `kube-controller-manager>=1.32`):
  Same as the above case.
* (`kube-apiserver>=1.32`, `kubelet>=1.32`, `kube-controller-manager<1.32`):
  kube-apiserver will admit pod specs with PodCertificate projected volumes.
  When scheduled, kubelet will successfully create a PodCertificateRequest
  object for the pod.  However, these PodCertificateRequests will hang in an
  unissued state forever, which means the pods will remain stuck at volume set
  up. `kubernetes.io/kube-apiserver-client-pod`, kube-controller-manager will
  not act on it.  Workloads that were trying to create these pods will be stuck
  at pod setup.
* (`kube-apiserver>=1.32`, `kubelet>=1.32`, `kube-controller-manager>=1.32`):
  The feature works normally.

Once the feature is launched, we need to consider feature evolution.

One possible example is support for a new key type.  Supporting a new key type
requires updates in all three components:
* Kubelet will need support for generating keys of the new type.
* Kube-apiserver will need support for deserializing the new key type and
  validating the proof-of-possession.
* Kube-controller-manager (and third-party signing controllers) will need
  support for issuing certificates to the new key type.

If support for the new key type is introduced in version N, then we have the
following skew cases:
* (`kube-apiserver>=N`, `kubelet<N`, `kube-controller-manager<N`):
  kube-apiserver will admit pods with PodCertificate projected volumes that
  specify the new key type, but these pods will fail to start when actually
  scheduled onto a node.  Kubelet will continually return an error during volume
  set up, when it validates the specified key type against its list of supported
  key types.
* (`kube-apiserver>=N`, `kubelet<N`, `kube-controller-manager>=N`): Same as the
  previous case.
* (`kube-apiserver>=N`, `kubelet>=N`, `kube-controller-manager<N`):
  kube-apiserver will admit pods with PodCertificate projected volumes that
  specify the new key type.  These pods will still fail to start when scheduled
  onto a node.  Kubelet will successfully create PodCertificateRequest objects,
  but kube-controller-manager will continually reject them because it does not
  understand the new key type.  Kubelet will reflect this by returning a
  permanent error from volume set up.
* (`kube-apiserver>=N`, `kubelet>=N`, `kube-controller-manager>=N`): No
  problems, everything works.

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

At alpha, enabling this feature in a live cluster will require the following steps.

1. Enable the certificates/v1alpha1 API.
2. Enable the PodCertificateRequest feature gate.
3. Enable the PodCertificateProjection feature gate.
4. Enable the PodCertificateAPIClientCertificates feature gate.

At this point, you can begin to create pods that use the features.  These steps
can all happen at once for clusters that don't currently have any workloads that
use PodCertificate projected volumes.

###### Does enabling the feature change any default behavior?

No, no default behavior is changed.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the feature can be disabled safely by following this procedure.

1. Remove any workloads that are currently using PodCertificate projected volumes.
2. Disable the PodCertificateAPIClientCertificates feature gate.
3. Disable the PodCertificateProjection feature gate.
4. Disable the PodCertificateRequest feature gate.
5. Disable the certificates/v1alpha1 API.

###### What happens if we reenable the feature if it was previously rolled back?

If the procedure in the previous section was followed, nothing.  If workloads
using PodCertificate projected volumes remained in the cluster, their behavior
may suddenly change.

###### Are there any tests for feature enablement/disablement?

We will cover this with unit tests on the PodCertificateRequest and Pod storage
layer.

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

Yes, pods that use PodCertificate projected volumes will create and watch
PodCertificateRequest objects at startup and every few hours as they refresh
certificates.

###### Will enabling / using this feature result in introducing new API types?

Yes, PodCertificateRequest objects will be created.  There is no per-cluster
limit, although we anticipate a small constant multiple of the number of pods in
the cluster.  PodCertificateRequest objects are single-use, and are
automatically cleaned up by control loops in kube-controller-manager.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Only a slight increase in the size of pods that specify PodCertificate projected
volume sources.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Not by itself.  If we decide to automatically inject PodCertificate projected
volumes into pods to replace existing projected token volumes, then it will be
on the critical path of all pod startup.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

kube-controller-manager will use more memory, since it will be running an
informer / control loop to handle `kubernetes.io/api-server-client-pod`
PodCertificateRequests.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

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

<<[UNRESOLVED @ahmedtd to flesh these out ]>>
* a new subresource for ServiceAccounts, that accepts a certificate signing request in some form, and (after checking) returns an issued certificate. The kubelet requests that subresource when creating a Pod that uses certificates to authenticate to the Kubernetes API
* HTTP Message Signatures - still draft; good grounds to skip this alternative. But I would still list it for context.
* HMAC-based one-time password tokens for Pods; the token is never retransmitted, but the involved parties (Pod, kubelet, API server) have access to its cleartext, so it's not a big improvement over reused bearer tokens.
<<[/UNRESOLVED]>>

**Using custom Relative Distinguished Names instead of a PodIdentity critical
extension**: We considered encoding pod identity into certificates issued by
`kubernetes.io/kube-apiserver-client-pod` as part of the `Subject` field, using
new Relative Distinguished Names.  We decided to use a critical custom extension
instead, for several reasons:
* This is the approach taken by the SubjectAlternateNames extension (though we
  could find no discussion in the history of those RFCs about why they are using
  extensions).
* We think the ability to set the extension as critical might be important.



## Infrastructure Needed (Optional)

We may want to set up a formal process for managing OIDs under the existing Kubernetes Private Enterprise number.  This could be as simple as a markdown file somewhere in the Kubernetes repository.
