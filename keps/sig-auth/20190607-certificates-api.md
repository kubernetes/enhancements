---
title: Certificates API
authors:
  - "@mikedanese"
owning-sig: sig-auth
reviewers:
  - "@liggitt"
  - "@smarterclayton"
approvers:
  - "@liggitt"
  - "@smarterclayton"
creation-date: 2019-06-07
last-updated: 2019-09-04
status: implementable
---

# Certificates API

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [Sequence of an Issuance](#sequence-of-an-issuance)
  - [CertificateSigningRequest API Definition](#certificatesigningrequest-api-definition)
  - [Manual CSR Approval With Kubectl](#manual-csr-approval-with-kubectl)
  - [Automatic CSR Approval Implementations](#automatic-csr-approval-implementations)
  - [Automatic Signer Implementations](#automatic-signer-implementations)
- [Graduation Criteria](#graduation-criteria)
  - [Beta -&gt; GA Graduation](#beta---ga-graduation)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

The Certificates API enables automation of
[x509](https://tools.ietf.org/html/rfc5280) credential provisioning by providing
a programmatic interface for clients of the Kubernetes API to request and obtain
x509 certificates from a Certificate Authority (CA).

## Motivation

The security of the Kubernetes platform is underpinned by a public key
infrastructure (PKI). Each Kubernetes cluster has a root certificate authority.
This CA is used to secure communication between cluster components. The
Certificates API was originally merged to support [Kubelet TLS
Bootstrap](https://github.com/kubernetes/community/blob/1fd524165bcf54d4bef99adb8332df72f4f88d5c/contributors/design-proposals/cluster-lifecycle/kubelet-tls-bootstrap.md)
but users have also begun to use this API to provision certificates for PKI
needs out of core.

### Goals

- Provide a generic API for PKI issuance to be consumed by both core Kubernetes
  components as well as user workloads running in the cluster.
- Support extensions that allow for specialized certificate issuance.

### Non-Goals

- Build in support for specialized certificate issuance (e.g.
  [LetsEncrypt](https://letsencrypt.org/)).

## Proposal

We intend to provision initial and renewed certificates in Kubernetes (often
without operator intervention). The crux of this process is how to make (and
automate) the decision of whether to approve or deny a particular certificate
signing request.

The role of a Registration Authority (referred to in this design as the
approver) is to verify that a CSR satisfies two requirements:

1. Authentication: The subject of the CSR is the origin of the CSR.
1. Authorization: The subject of the CSR is authorized to act in the requested
   context (e.g. has authority over requested Subject Alternative Names, etc).

Iff these two requirements are met, the approver should approve the CSR and
otherwise should deny the CSR. Once the CSR is approved a Certificate Authority
(referred to in this design as the signer) should construct a certificate from
the CSR and return the certificate to the requester.

The Certificates API provides a generic means of communication exposed via the
Kubernetes resource model over which a certificate requestor, approver and
signer can interact.

## Design Details

A client requesting a certificate post a CertificateSigningRequest to the
Certificates API. The client may only provide the encoded [Certificate
Request](https://tools.ietf.org/html/rfc2986) and usages of the certificate in
the spec (and the standard object metadata) on the initial creation of the
CertificateSigningRequest. The kube-apiserver also asserts authentication
attributes of the requestor in the CertificateSigningRequest spec before
committing it to storage so that they can be used later during CSR approval. The
information contained in the spec is immutable after the request is created.

The approver updates approval status of the CertificateSigningRequest via the
CertificateSigningRequestStatus. The approval condition can only be updated via
the `/approval` subresource allowing approval permission to be authorized
independently of other operations on the CertificateSigningRequest.

Contingent on approval, the signer posts a signed certificate to the status. The
certificate field of the status can only be updated via the `/status`
subresource allowing signing permission to be authorized independently of other
operations on the CertificateSigningRequest.

The API is designed to support the standard asynchronous controller model of
Kubernetes where the approver and signer act as independent controllers of the
Certificates API. Since issuance is asynchronous, the approver can perform
out-of-band verification of the CSR before making an authorization decision.

The approver is designed to be explicitly independent of the signer. This
separates concerns of authorization and certificate minting and also allows the
signer to operate in a separate, more restrictive environment than the approver.
This is typical of many PKI architectures.

### Sequence of an Issuance

A typical successful issuance proceeds as follows.

![CSR](/keps/sig-auth/csr.png)

1. The requestor generates a private key, builds a certificate signing request,
   and submits the `CertificateSigningRequest` to the Kubernetes certificates
   API.
1. The approver controller observes the newly submitted request, validates and
   authorizes the request and if all goes well, approves the request.
1. The signer observes the approval, mints a new certificate and stores it in
   the `.Status.Certificate` field.
1. The requestor observes the update, and stores the certificate locally.

### CertificateSigningRequest API Definition

```go
// This information is immutable after the request is created. Only the Request
// and Usages fields can be set on creation, other fields are derived by
// Kubernetes and cannot be modified by users.
type CertificateSigningRequest struct {
  Spec   CertificateSigningRequestSpec
  Status CertificateSigningRequestStatus
}


type CertificateSigningRequestSpec struct {
  // Base64-encoded PKCS#10 CSR data
  Request []byte

  // usages specifies a set of usage contexts the key will be
  // valid for.
  // See: https://tools.ietf.org/html/rfc5280#section-4.2.1.3
  //      https://tools.ietf.org/html/rfc5280#section-4.2.1.12
  Usages []KeyUsage

  // Information about the requesting user.
  // See user.Info interface for details.
  Username string
  // UID information about the requesting user.
  // See user.Info interface for details.
  UID string
  // Group information about the requesting user.
  // See user.Info interface for details.
  Groups []string
  // Extra information about the requesting user.
  // See user.Info interface for details.
  Extra map[string]ExtraValue
}

// ExtraValue masks the value so protobuf can generate
type ExtraValue []string

type CertificateSigningRequestStatus struct {
  // Conditions applied to the request, such as approval or denial.
  Conditions []CertificateSigningRequestCondition

  // If request was approved, the controller will place the issued
  // certificate here.
  Certificate []byte
}

type CertificateSigningRequestCondition struct {
  // request approval state, currently Approved or Denied.
  Type RequestConditionType
  // brief reason for the request state
  Reason string
  // human readable message with details about the request state
  Message string
}

type RequestConditionType string

// These are the possible conditions for a certificate request.
const (
  CertificateApproved RequestConditionType = "Approved"
  CertificateDenied   RequestConditionType = "Denied"
)
```

### Manual CSR Approval With Kubectl

A Kubernetes administrator (with appropriate permissions) can manually approve
(or deny) Certificate Signing Requests by using the `kubectl certificate
approve` and `kubectl certificate deny` commands.

### Automatic CSR Approval Implementations

The kube-controller-manager ships with an in-built
[approver](https://github.com/kubernetes/kubernetes/blob/32ec6c212ec9415f604ffc1f4c1f29b782968ff1/pkg/controller/certificates/approver/sarapprove.go)
for Kubelet TLS Bootstrap that delegates various permissions on CSRs for node
credentials to authorization. It does this by posting subject access reviews to
the API server. It punts on TLS certificates for server authentication of the
Kubelet API because verifying IP SANs for Kubelets in a generic way poses
challenges.

The GCP controller manager replaces the in-built approver with an
[approver](https://github.com/kubernetes/cloud-provider-gcp/blob/08fa1e3260ffb267682762e24ba93692000e3be8/cmd/gcp-controller-manager/csr_approver.go)
that handles TLS certificates for server authentication and support for
verification of CSRs with proof of control of GCE vTPM.

An external project, [kapprover](https://github.com/coreos/kapprover), does
policy based approval of kubelet CSRs.

### Automatic Signer Implementations

The kube-controller-manager ships with an in-built
[signer](https://github.com/kubernetes/kubernetes/blob/32ec6c212ec9415f604ffc1f4c1f29b782968ff1/pkg/controller/certificates/signer/cfssl_signer.go)
that signs all approved certificates with a local key. This key is generally
configured to be the key of the cluster's root certificate authority, but is not
required to be.

The GCP controller manager implements a
[signer](https://github.com/kubernetes/cloud-provider-gcp/blob/08fa1e3260ffb267682762e24ba93692000e3be8/cmd/gcp-controller-manager/csr_signer.go)
that uses a webhook to sign all approved CSRs. This allows the root certificate
authority secret material to be stored and maintained outside of the Kubernetes
control plane.

## Graduation Criteria

### Beta -> GA Graduation

- TBD

## Implementation History

- The Certificates API was merged as Alpha in 1.4
- The Certificates API was promoted to Beta in 1.6
