# KEP-266: Kubelet client certificate bootstrap and rotation

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [During Kubelet Boot Sequence](#during-kubelet-boot-sequence)
  - [As Expiration Approaches](#as-expiration-approaches)
  - [Certificate Approval](#certificate-approval)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

- [x] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] KEP approvers have approved the KEP status as `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [x] "Implementation History" section is up-to-date for milestone
- [x] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes
- [x] User-facing documentation has been created at https://kubernetes.io/docs/tasks/tls/certificate-rotation/

## Summary

Currently, a kubelet has a certificate/key pair that authenticates the kubelet to the kube-apiserver.
The certificate is supplied to the kubelet when it is first booted, via an out of cluster mechanism.
This proposal covers a process for obtaining the initial cert/key pair and rotating it as expiration 
of the certificate approaches.

### Goals

1. Enable exchange of a short-lived credential for a client certificate on a new node when starting a Kubelet for the first time.
2. Enable renewal/rotation of a Kubelet certificate as it approaches expiration.

## Proposal

### During Kubelet Boot Sequence

1. Look on disk for an existing cert/key pair managed by the certificate manager.
1. If there is an existing cert/key pair, load them.
1. If there is no existing cert/key pair, look for cert/key data specified in the kubelet config file:
    - values encoded in the kubeconfig (CertData, KeyData, CAData)
    - file references in the kubeconfig (CertFile, KeyFile, CAFile)
1. If the certificate is a bootstrap certificate, use the certificate to:
    - generate a key
    - create a certificate signing request
    - request a signed certificate from the API server
    - wait for a response
    - Store the new cert/key pairs in the Kubelet's certificate directory

### As Expiration Approaches

Rotating a Kubelet client certificate will work by generating a new private key,
issuing a new Certificate Signing Request to the API Server, safely updating the
cert/key pair on disk, begin using the new cert/key pair.

1. Store the new cert/key pairs in the kubelet's certificate directory.
   This will allow the kubelet to have a place for storing the multiple 
   cert/key pairs that it might have available at any given moment 
   (because of a rotation in progress).
    - When cert/key files are specified in the kubeconfig, these will be used if
      a newer, rotated, cert/key pair does not exist.
2. There will be a kubelet configuration option (`--rotate-certificates`) which must be set to true
   to enable certificate bootstrapping and rotation, and a feature gate (`RotateKubeletClientCertificate`)
   which must be enabled prior to the feature's graduation to GA
4. Centralize certificate access within the kubelet code to the CertificateManager.
   Whenever rotation is enabled, CertificateManager will be responsible for:
    - Providing the correct certificate to use for establishing TLS connections.
    - Generating new private keys and requesting new certificates when the
      current certificate approaches expiry.
    - Since certificates can rotate at any time, all other parts of the kubelet
      should ask the CertificateManager for the correct certificate each time a
      certificate is used. No certificate caching except by the CertificateManager.
        - TLS connections should prefer to set the
          [`GetCertificate`](https://golang.org/pkg/crypto/tls/#Config) and
          [`GetClientCertificate`](https://golang.org/pkg/crypto/tls/#Config)
          callbacks so that the connection dynamically requests the certificate
          as needed.
    - Recovering from kubelet crashes or restarts that occur while certificate
      transitions are in flight (request issued, but not yet signed, etc)
5. The RBAC `system:node` role and the Node authorizer will permit nodes to create and read CertificateSigningRequests.
6. The CertificateManager repeats the request process as certificate expiration approaches.
    - New certificates will be requested when the configured duration threshold has been exceeded.
    - Crash-safe file structure:
        - A private key is generated and reused until a successful CSR is received.
        - The name of the created CSR is based on a hash of the public key,
          so a kubelet restart allows it to resume waiting for a previously created CSR.
        - When the corresponding signed certificate is received,
          the temporary private key is removed, and the the cert/key pair is written to a single file,
          e.g. `kubelet-client-<timestamp>.pem`.
        - Replace the `kubelet-client-current.pem` symlink to point to the new cert/key pair.

### Certificate Approval

With the kubelet requesting certificates be signed as part of its boot sequence,
and on an ongoing basis, certificate signing requests from the kubelet need to
be auto approved to make cluster administration manageable. Certificate signing
request approval is complete, and covered by [this design]
(https://github.com/kubernetes/enhancements/blob/master/keps/sig-auth/20190607-certificates-api.md).

### Test Plan

Unit tests:
* Certificate comparison
* Key generation
* Rotation expiration logic
* Error handling waiting for certificate issuance
* Writing rotated certificates to disk
* Error handling writing rotated certificates to disk

E2E tests:
* Enable bootstrap/rotation in e2e test clusters

### Graduation Criteria

* Test plan is completed
* CertificateSigningRequest API is at v1
* Certificate manager uses the v1 CertificateSigningRequest API if available
* Actively used in production clusters by multiple distributions for at least 3 releases

### Upgrade / Downgrade Strategy

Client certificate bootstrap and rotation are opt-in features, controlled by the Node deployer.
Kubelet upgrades that do not opt-in continue using the provided credentials.

Any upgrade that opts the kubelet into bootstrap/rotation must also provide 
the kubelet with credentials authorized to obtain an initial client certificate.

Any downgrade that reverts the kubelet to a version that does not support bootstrap/rotation,
or which opts out of the feature, must also provide the kubelet with credentials with
sufficient longevity and authorization to function properly.

### Version Skew Strategy

The Kubernetes skew policy requires API servers to be as new or newer than Kubelets.
This means Kubelets can use current versions of the CertificateSigningRequest API.
Any changes to the CertificateSigningRequest in kube-apiserver for issuing kubelet
client certificates must remain backwards compatible for at least n-2 releases.

## Implementation History

- 2017-06-27 - Design proposed at https://github.com/kubernetes/community/pull/768
- v1.4 - CertificateSigningRequest API v1alpha1 released
- v1.5 - CertificateSigningRequest API v1beta1 released
- v1.7 - gce enables kubelet client certificate bootstrap
- v1.8 - RotateKubeletClientCertificate feature released as beta
- v1.8 - kubeadm enables kubelet client certificate rotation
- v1.11 - openshift enables kubelet client certificate rotation
- v1.18 - CertificateSigningRequest API adds support for specific kubelet client certificate signers
- v1.19 - Legacy proposal converted to KEP
