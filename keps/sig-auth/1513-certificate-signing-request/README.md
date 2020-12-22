# Certificates API

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [Sequence of an Issuance](#sequence-of-an-issuance)
  - [Signers](#signers)
    - [Limiting approval and signer powers for certain signers.](#limiting-approval-and-signer-powers-for-certain-signers)
  - [API changes between v1beta1 and v1](#api-changes-between-v1beta1-and-v1)
  - [CertificateSigningRequest API Definition](#certificatesigningrequest-api-definition)
  - [Manual CSR Approval With Kubectl](#manual-csr-approval-with-kubectl)
  - [Automatic CSR Approval Implementations](#automatic-csr-approval-implementations)
  - [Automatic Signer Implementations](#automatic-signer-implementations)
  - [Test Plan](#test-plan)
- [Graduation Criteria](#graduation-criteria)
  - [Beta to GA Graduation](#beta-to-ga-graduation)
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
Request](https://tools.ietf.org/html/rfc2986), usages of the certificate in
the spec, the standard object metadata, and the requested signer on the initial creation of the
CertificateSigningRequest. The kube-apiserver also asserts authentication
attributes of the requestor in the CertificateSigningRequest spec before
committing it to storage so that they can be used later during CSR approval. The
information contained in the spec is immutable after the request is created.

An approver updates approval status of the CertificateSigningRequest via the
CertificateSigningRequestStatus. `Approved`/`Denied` conditions can only be set via
the `/approval` subresource, allowing approval permission to be authorized
independently of other operations on the CertificateSigningRequest.
`Approved`/`Denied` conditions are mutually exclusive, cannot be removed,
cannot have a `False` or `Unknown` status, and are treated as `True` if present.

Once an `Approved` condition is present, a signer posts a signed certificate to the status. The
certificate field of the status can only be updated via the `/status`
subresource allowing signing permission to be authorized independently of other
operations on the CertificateSigningRequest.

Terminal signer failures (such as a refusal to sign the CSR) can be indicated with a `Failed` condition.
A `Failed` condition cannot be removed, cannot have a `False` or `Unknown` status, and is treated as `True` if present.

The API is designed to support the standard asynchronous controller model of
Kubernetes where the approver and signer act as independent controllers of the
Certificates API. Since issuance is asynchronous, an approver can perform
out-of-band verification of the CSR before making an authorization decision.

The approver is designed to be explicitly independent of the signer. This
separates concerns of authorization and certificate minting and also allows the
signer to operate in a separate, more restrictive environment than the approver.
This is typical of many PKI architectures.

### Sequence of an Issuance

A typical successful issuance proceeds as follows:

![CSR](/keps/sig-auth/1513-certificate-signing-request/csr.png)

1. The requestor generates a private key, builds a certificate signing request,
   and submits the `CertificateSigningRequest` to the Kubernetes certificates
   API.
1. The approver controller observes the newly submitted request, validates and
   authorizes the request and adds an `Approved` condition via the `/approval` subresource.
1. The signer observes the approval, mints a new certificate and stores it in
   the `.Status.Certificate` field via the `/status` subresource.
1. The requestor observes the update, and stores the certificate locally.

Approver failure sequence:

1. The requestor generates a private key, builds a certificate signing request,
   and submits the `CertificateSigningRequest` to the Kubernetes certificates
   API.
1. The approver controller observes the newly submitted request, determines the
   request should be denied, and adds a `Denied` condition via the `/approval` subresource.
1. The requestor observes the `Denied` condition.

Signer failure sequence:

1. The requestor generates a private key, builds a certificate signing request,
   and submits the `CertificateSigningRequest` to the Kubernetes certificates
   API.
1. The approver controller observes the newly submitted request, validates and
   authorizes the request and adds an `Approved` condition via the `/approval` subresource.
1. The signer observes the approval, but encounters a terminal failure during the signing process,
   and adds a `Failed` condition via the `/status` subresrouce.
1. The requestor observes the `Failed` condition.

### Signers

CSRs have a `signerName` field which is used to specify which signer the CSR creator wants to sign the certificate.
To support migration from v1beta1 to v1, this required field will be defaulted in v1beta1 (optional in openapi), but
not defaulted and required in v1 :
 1. If it's a kubelet client certificate, it is assigned "kubernetes.io/kube-apiserver-client-kubelet".
 2. If it's a kubelet serving certificate, it is assigned "kubernetes.io/kubelet-serving". 
 see https://github.com/kubernetes/kubernetes/blob/release-1.10/pkg/controller/certificates/approver/sarapprove.go#L211-L223 for details.
 3. Otherwise, it is assigned "kubernetes.io/legacy-unknown".

There will be field selector support to make approvers and signers easier to write.

All signers should provide information about how they work so that clients can predict what will happen to their CSRs.
This includes:
 1. Trust distribution - how trust (ca bundles) are distributed.
 2. Permitted subjects - (any? specific subtree?) and behavior when a disallowed subject is requested.
 3. Permitted x509 extensions - (IP SANs? DNS SANs? Email SANs? URI SANs? others?) and behavior when a disallowed
 extension is requested.
 4. Permitted key usages / extended key usages - (client only? server only? any? signer-determined? CSR-determined?) and
 behavior when usages different than the signer-determined usages are specified in the CSR.
 5. Expiration/cert lifetime - (fixed by signer? configurable by admin? CSR-determined?) and behavior when an expiration
 different than the signer-determined expiration is specified in the CSR.
 6. CA bit allowed/disallowed - and behavior if a CSR contains a request a for a CA cert when the signer does not permit it.
 7. (optional) Information about the meaning of additional CERTIFICATE PEM blocks in `status.certificate`, if different from the
 standard behavior of treating the additional certificates as intermediates, and presenting them in TLS handshakes.


sig-auth reserves all `kubernetes.io/*` `signerNames` and more may be added in the future.
Kubernetes provides the following well-known signers.  Today, failures for all of these are only reported in kube-controller-manager logs:
 1. kubernetes.io/kube-apiserver-client - signs certificates that will be honored as client-certs by the kube-apiserver.
    Never auto-approved by kube-controller-manager.
    1. Trust distribution: signed certificates must be honored as client-certificates by the kube-apiserver.  The CA bundle
       is not distributed by any other means.
    2. Permitted subjects - no subject restrictions, but approvers and signers may choose not to approve or sign.
       Certain subjects like cluster-admin level users or groups vary between distributions and installations, but deserve
       additional scrutiny before approval and signing.  An admission plugin is available to restrict system:masters, but
       it is often not the only cluster-admin subject in a cluster.
    3. Permitted x509 extensions - URI, DNS, IP, and Email SAN extensions are honored. Other extensions are discarded.
    4. Permitted key usages - must include `[]string{"client auth"}`.  Must not include key usages beyond `[]string{"digital signature", "key encipherment", "client auth"}`
    5. Expiration/cert lifetime - minimum of CSR signer or request.  Sanity of the time is the concern of the signer.
    6. CA bit allowed/disallowed - not allowed.
 2. kubernetes.io/kube-apiserver-client-kubelet - signs client certificates that will be honored as client-certs by the kube-apiserver.
    May be auto-approved by kube-controller-manager.
    1. Trust distribution: signed certificates must be honored as client-certificates by the kube-apiserver.  The CA bundle
       is not distributed by any other means.
    2. Permitted subjects - organizations are exactly `[]string{"system:nodes"}`, common name starts with `"system:node:"`
    3. Permitted x509 extensions - URI, DNS, IP, and Email SAN extensions are not allowed. Other extensions are discarded.
    4. Permitted key usages - exactly `[]string{"key encipherment", "digital signature", "client auth"}`
    5. Expiration/cert lifetime - minimum of CSR signer or request.  Sanity of the time is the concern of the signer.
    6. CA bit allowed/disallowed - not allowed.
 3. kubernetes.io/kubelet-serving - signs serving certificates that are honored as a valid kubelet serving certificate 
    by the kube-apiserver, but has no other guarantees.  Never auto-approved by kube-controller-manager.
    1. Trust distribution: signed certificates must be honored by the kube-apiserver as valid to terminate connections to a kubelet.
       The CA bundle is not distributed by any other means.
    2. Permitted subjects - organizations are exactly `[]string{"system:nodes"}`, common name starts with `"system:node:"`
    3. Permitted x509 extensions - DNS and IP SANs are allowed, and at least one DNS or IP SAN must be present. URI and Email SANs are not allowed. Other extensions are discarded.
    4. Permitted key usages - exactly `[]string{"key encipherment", "digital signature", "server auth"}`
    5. Expiration/cert lifetime - minimum of CSR signer or request.
    6. CA bit allowed/disallowed - not allowed.
 4. kubernetes.io/legacy-unknown - has no guarantees for trust at all.  Some distributions may honor these as client
    certs, but that behavior is not standard kubernetes behavior.  Never auto-approved by kube-controller-manager.
    This signerName is only permitted in CertificateSigningRequest objects created via the v1beta1 API.
    1. Trust distribution: None.  There is no standard trust or distribution for this signer in a kubernetes cluster.
    2. Permitted subjects - any
    3. Permitted x509 extensions - URI, DNS, IP, and Email SAN extensions are honored. Other extensions are discarded.
    4. Permitted key usages - any
    5. Expiration/cert lifetime - minimum of CSR signer or request.  Sanity of the time is the concern of the signer.
    6. CA bit allowed/disallowed - not allowed.

Distribution of trust happens out of band for these signers.  Any trust outside of those described above are strictly
coincidental.  For instance, some distributions may honor kubernetes.io/legacy-unknown as client-certificates for the
kube-apiserver, but this is not a standard.
None of these usages are related to ServiceAccount token secrets `.data[ca.crt]` in any way.  That ca-bundle is only
guaranteed to verify a connection the kube-apiserver using the default service.

To support HA upgrades, the kube-controller-manager will duplicate defaulting code for an empty `signerName` for one
release.

#### Limiting approval and signer powers for certain signers.
Given multiple signers which may be implemented as "dumb" controllers that sign if the CSR is approved, there is benefit
to providing a simple way to subdivide approval powers through the API.  We will introduce an admission plugin that requires
 1. verb == `approve`
 2. resource == `signers`
 3. name == `<.spec.signerName>` 
 4. group == `certificates.k8s.io`
 
To support a use-case that wants a single rule to allow approving an entire domain (example.com in example.com/cool-signer),
there will be a second check for
 1. verb == `approve`
 2. resource == `signers`
 3. name == `<.spec.signerName domain part only>/*` 
 4. group == `certificates.k8s.io`

There are congruent check for providing a signature that use the verb=="sign" instead of "approve" above.

For migration, we will provide three bootstrap cluster-roles defining authorization rules needed to approve CSRs for the kubernetes.io signerNames.
Cluster admins can either:
1. grant signer-specific approval permissions using roles they define
2. grant signer-specific approval permissions using the bootstrap roles starting in 1.18
3. disable the approval-authorizing admission plugin in 1.18 (if they don't care about partitioning approver rights)

### API changes between v1beta1 and v1

`spec.signerName` validation:

* required, not defaulted
* `kubernetes.io/legacy-unknown` is disallowed when creating new CSR objects

`status.certificate` validation:

* cannot be unset or changed once set
* must contain one or more PEM blocks
* all PEM blocks must have the "CERTIFICATE" label, contain no headers, and the encoded data
  must be a BER-encoded ASN.1 Certificate structure as described in section 4 of RFC5280.
* non-PEM content may appear before or after the CERTIFICATE PEM blocks and is unvalidated,
  to allow for explanatory text as described in section 5.2 of RFC7468.

`status.conditions` validation:

* type is required
* status is required and must be `True`, `False`, or `Unknown`

`/status` subresource:

* allows modifying `status.conditions` other than `Approved`/`Denied`
  (`Approved`/`Denied` conditions are still limited to the `/approval` subresource
  to ensure version-independent authorization policy grants consistent permissions
  across v1beta1 and v1 versions)

### CertificateSigningRequest API Definition

```go
type CertificateSigningRequest struct {
  // spec information is immutable after the request is created.
  // Only the request, usages, and signerName fields can be set on creation,
  // other fields are derived by Kubernetes and cannot be modified by users.
  Spec   CertificateSigningRequestSpec
  Status CertificateSigningRequestStatus
}

type CertificateSigningRequestSpec struct {
  // requested signer for the request up to 571 characters long.  It is a qualified name in the form: `scope-hostname.io/name`.  
  // In v1beta1, it will be defaulted:
  //  1. If it's a kubelet client certificate, it is assigned "kubernetes.io/kube-apiserver-client-kubelet".  This is determined by 
  //     Seeing if organizations are exactly `[]string{"system:nodes"}`, common name starts with `"system:node:"`, and
  //     key usages are exactly `[]string{"key encipherment", "digital signature", "client auth"}`
  //  2. Otherwise, it is assigned "kubernetes.io/legacy-unknown".
  // In v1, it will not be defaulted, is required, and `kubernetes.io/legacy-unknown` is not permitted when creating new requests.
  // Distribution of trust for signers happens out of band. 
  // The following signers are known to the kube-controller-manager signer.
  //  1. kubernetes.io/kube-apiserver-client - signs certificates that will be honored as client-certs by the kube-apiserver. Never auto-approved by kube-controller-manager.
  //  2. kubernetes.io/kube-apiserver-client-kubelet - signs client certificates that will be honored as client-certs by the kube-apiserver. May be auto-approved by kube-controller-manager.
  //  3. kubernetes.io/kubelet-serving - signs serving certificates that are honored as a valid kubelet serving certificate by the kube-apiserver, but has no other guarantees.
  //  4. kubernetes.io/legacy-unknown - has no guarantees for trust at all.  Some distributions may honor these as client certs, but that behavior is not standard kubernetes behavior.
  // None of these usages are related to ServiceAccount token secrets `.data[ca.crt]` in any way.
  // You can select on this field using `.spec.signerName`.
  SignerName string

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
  // Conditions applied to the request, such as approval, denial, or failure.
  Conditions []CertificateSigningRequestCondition

  // Certificate is populated by the signer with the issued certificate in PEM format.
  // In JSON and YAML output, this entire field is base64-encoded, so it consists of:
  // base64(
  // -----BEGIN CERTIFICATE-----
  // MII...Pb7Yu/E=
  // -----END CERTIFICATE-----
  // optional intermediate certificate blocks
  // -----BEGIN CERTIFICATE-----
  // MII...MGKB
  // -----END CERTIFICATE-----
  // -----BEGIN CERTIFICATE-----
  // MII...AY1M
  // -----END CERTIFICATE-----
  // )
  //
  // In v1beta1, this field is unvalidated.
  //
  // In v1, modified content in this field is validated:
  // * Content must contain one or more PEM blocks
  // * All PEM blocks must have the "CERTIFICATE" label, contain no headers, and the encoded data
  //   must be a BER-encoded ASN.1 Certificate structure as described in section 4 of RFC5280.
  // * Non-PEM content may appear before or after the CERTIFICATE PEM blocks and is unvalidated,
  //   to allow for explanatory text as described in section 5.2 of RFC7468.
  //
  // If more than one PEM block is present, and the definition of the requested spec.signerName
  // does not indicate otherwise, the first block is the issued certificate,
  // and subsequent blocks should be treated as intermediate certificates and presented in TLS handshakes.
  Certificate []byte
}

type CertificateSigningRequestCondition struct {
  // conditions, including ones that indicate request approval ("Approved", "Denied") and failure ("Failed").
  // Approved/Denied conditions are mutually exclusive.
  // Approved, Denied, and Failed conditions cannot be removed once added.
  // Only one condition of a given type is allowed.
  Type RequestConditionType
  // status of the condition. Valid values are True, False, and Unknown.
  // Approved, Denied, and Failed conditions are limited to a value of True.
  // In v1beta1, this is optional and defaults to True.
  // In v1, this is required and not defaulted.
  Status v1.ConditionStatus
  // brief reason for the request state
  Reason string
  // human readable message with details about the request state
  Message string
  // lastUpdateTime timestamp for the last update to this condition.
  // If unset, when a CSR is written the server defaults this to the current time.
  // +optional
  LastUpdateTime metav1.Time
  // lastTransitionTime is the time the condition last transitioned from one status to another.
  // If unset, when a new condition type is added or an existing condition's status is changed,
  // the server defaults this to the current time.
  // +optional
  LastTransitionTime metav1.Time
}

type RequestConditionType string

// These are well-known conditions types for a certificate request.
const (
  CertificateApproved RequestConditionType = "Approved"
  CertificateDenied   RequestConditionType = "Denied"
  CertificateFailed   RequestConditionType = "Failed"
)
```

### Manual CSR Approval With Kubectl

A Kubernetes administrator (with appropriate permissions) can manually approve
(or deny) Certificate Signing Requests by using the `kubectl certificate approve`
and `kubectl certificate deny` commands.

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

### Test Plan

- unit tests of:
  - API defaulting
  - API validation
  - API round-tripping
  - Authorizing admission plugin
  - Admission plugin limiting system:master client CSRs
  - Auto-approving controller
  - Auto-signing controller
  - v1beta1 behavior validation
    - `Approved`/`Denied` conditions can be added via `/approval` subresource, modifications are ignored via `/status` subresource
    - `Failed` condition can be added via `/approval` or `/status` subresource
    - `status.certificate` can be set via `/status` subresource, modification is ignored via `/approval` subresource
    - `status.certificate` can be set to arbitrary content
  - v1 behavior validation
    - `Approved`/`Denied` conditions can be added via `/approval` subresource, modifications are rejected via `/status` subresource
    - `Failed` condition can be added via `/approval` or `/status` subresource
    - `status.certificate` can be set via `/status` subresource, modification is rejected via `/approval` subresource
    - `status.certificate` content is validated when modified

- integration tests of:
  - auto-approve and signing of kubelet client CSR
  - manual-approve and auto-signing of kubelet serving CSR
  - manual-approve and auto-signing of API client CSR
  - manual-approve and auto-signing of legacy CSR

- e2e tests (and conformance tests, once v1) of:
  - CSR API presence, CRUD behavior
  - status subresource get/update behavior
  - creating, approving, and signing a CSR for a custom signerName (e.g. `k8s.example.com/e2e`)

## Graduation Criteria	

### Beta to GA Graduation

- Multi-signer API, admission plugins, and controller-handling for `kubernetes.io/*` signers is implemented and tested
- Test plan is completed
- e2e tests acceptable as conformance tests exist for all v1 endpoints+verbs

## Implementation History

- 1.4: The Certificates API was merged as Alpha
- 1.6: The Certificates API was promoted to Beta
- 1.18: 2020-01-15: Multi-signer design added
- 1.18: 2020-01-21: status.certificate field format validation added
- 1.19: 2020-05-07: v1 details added
- 1.19: 2020-08-26: v1 implementation released
