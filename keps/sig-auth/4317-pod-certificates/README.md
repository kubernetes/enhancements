<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
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
# KEP-4317: PodIdentity certificates, PodCertificate volumes, and in-cluster kube-apiserver client certificates

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
- [Summary and Motivation](#summary-and-motivation)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [X.509 PodIdentity Extension](#x509-podidentity-extension)
  - [PodIdentity Extension Node Restriction](#podidentity-extension-node-restriction)
  - [PodCertificate Projected Volume Sources](#podcertificate-projected-volume-sources)
    - [API Object Diff](#api-object-diff)
    - [Example certificate bundle written to the pod filesystem](#example-certificate-bundle-written-to-the-pod-filesystem)
  - [API Server PodIdentity Extension Support](#api-server-podidentity-extension-support)
  - [API Server Pod Client Certificate Approver / Signer](#api-server-pod-client-certificate-approver--signer)
  - [Client-go InClusterConfig Enhancements](#client-go-inclusterconfig-enhancements)
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
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary and Motivation

The certificates.k8s.io API group provides a flexible, pluggable mechanism to request X.509 certificates within a Kubernetes cluster, but actually making certificates available to your workloads is left as an exercise for the reader.

This KEP defines three pieces of fundamental machinery:
* An X.509 extension for expressing pod identity
* Node restriction support for certificate requests that use the PodIdentity
  extension.
* A PodCertificate projected volume source that instructs the kubelet to handle
  provisioning a key and certificate chain on behalf of a pod.  
Taken together, this machinery makes it feasible to securely and automatically
deliver X.509 certificates to every pod in a cluster, without imposing an
unreasonable burden on application developers or cluster administrators.  

As a first application of this machinery, this KEP defines additional machinery that allows pods to automatically receive mTLS client certificates suitable for authenticating to kube-apiserver, iwth the same attributes provided by bound service account tokens:
* A new TLS user-mapping capability in kube-apiserver to understand client
  certificates that embed PodIdentity extensions.
* A new approver/signer pair, shipped in kube-controller-manager, to issue
  compatible client certificate to pods.
* An enhancement to client-go's InClusterConfig that automatically loads a
  credential bundle from a well-known filesystem path and uses is as a client
  certificate.

There is a draft implementation of the design outlined in this KEP:
https://github.com/kubernetes/kubernetes/pull/121596

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### Goals

**Ensure that certificate issuance is accomplished without relying on bearer tokens at any step**:  One of the most significant benefits of mTLS authentication is that the credential is split into two pieces (private key and certificate), only one of which is ever transmitted to the counterparty (and thus subject to eavesdropping or theft by the counterparty).  In order to preserve this guarantee, it's important that bearer tokens are not involved *at any step*, otherwise the security of the scheme devolves to just being a bearer token.  In this KEP, as long as both kubelet and the signer implementation authenticate to kube-apiserver using their own separately-bootstrapped client certificates, then no bearer tokens are involved in certificate issuance.

**Make it easy to ship secure third-party signer implementations**:  Securely implementing and operating a third-party signer implementation today requires intimate familiarity with Kubernetes' security model, otherwise your signer may negate Kubernetes' node restriction security boundary.  This KEP moves responsibilty for enforcing node restriction on CSRs into kube-apiserver, so that signer implementations do not need to consider it.

### Non-Goals

**Specify a solution for pod-to-pod mTLS in core Kubernetes**: Pod-to-pod mTLS (aka service mesh) is a rapidly-evolving area.  While the PodCertificate volume and PodIdentity X.509 extension are expected (and designed to be) useful for service mesh implementations, it doesn't make sense to try and standardize pod-to-pod mTLS at the core Kubernetes layer today.

**Support TPM-backed private keys**: In this proposal, private keys are
contained to the node (held only in kubelet memory and tmpfs projected volumes),
but they can still be extracted from a compromised workload (or compromised
node).  Storing the private keys in TPMs closes this loophole, but will require
an additional KEP to define how kubelet can use the Linux (or other platform's)
virtual TPM subsystem to provide virtual TPMs to pods

**Be 100% compatible with software in the field today**: The primary example in this KEP is that the private key and certificate chain are delivered in a single credential-bundle file.  This is based on implementation and operation experience of a CSI-based certificates solution that delivered private keys and certificate chains in separate files, as expected by most commodity webservers today.  While this is an attractive idea, automatically rotating these separate files is fraught.  Even if both file's content is updated atomically, skewed reads in the workload can result in hard-to-diagnose errors if the workload reads the key before rotation and the certificate chain after rotation (or vice-versa).

**Support private keys shared across multiple pods**: Sharing private keys and certificates across multiple pods may be required in certain applications.  However, this use case is well-served by Secret projected volume sources.

**Support human-in-the-loop approval for certificate requests**: This KEP assumes that CSR approval and issuance is fully-automated, because this process blocks pod startup for pods that use PodCertificate projected volume sourcs.

**Standardize on using the SPIFFE SVID format in core Kubernetes**: The certificates issued by `kubernetes.io/kube-apiserver-client-pod` can be made compatible with SPIFFE in the future, since they currently don't include any Subject Alternate Name entries.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### User Stories (Optional)

#### Story 1

I'm an application developer building a Kubernetes controller that I want to deploy into my cluster.  I want to make sure that my controller securely authenticates to kube-apiserver using mTLS.

I add a new projected volume to my pod, containing the information that client-go expects at a well-known path.  I set `noAutomountServiceAccountToken` because I want to ensure that my pod cannot use bearer tokens for authentication.
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
      mountPath: /var/run/secrets/kubernetes.io/api-mtls-access
  volumes:
  - name: kube-apiserver-client-certificate
    projected:
      sources:
      - podCertificate:
          signerName: "kubernetes.io/kube-apiserver-client-pod"
          credentialBundlePath: client-credentials.pem
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

I ensure that my pod uses client-go's InClusterConfig, and client-go automatically uses the mTLS client certificate to authenticate to kube-apiserver.

### Risks and Mitigations

**Scalability**: In the current design and draft implementation, kubelet holds
the private keys for each pod on the node in memory.  If kubelet restarts, then
the private keys are lost, and kubelet will need to immediately generate new
keys and request new certificates for each volume on the pod.  If kubelet
crashloops, this can generate high CertificateSigningRequest traffic on
kube-apiserver.  This can potentially be mitigated by giving kubelet the
capability to read back the existing key from the projected volume on startup.

## Design Details

### X.509 PodIdentity Extension

The CNCF has an existing ASN.1 Private Enterprise Number registered for use by
the Kubernetes project (`1.3.6.1.4.1.57683`).  In order to cleanly embed pod
identity information into an X.509 certificate or certificate request, we can
define a new ASN.1 OID (currently `1.3.6.1.4.1.57683.1`, though we need to
define a management policy for the PEN) and specify that it contains the
following DER-encoded structure:

```go
type podIdentityASN1 struct {
	Namespace          string `asn1:"utf8"`
	ServiceAccountName string `asn1:"utf8"`
	PodName            string `asn1:"utf8"`
	PodUID             string `asn1:"utf8"`
	NodeName           string `asn1:"utf8"`
}
```

Certificates and certificate requests that embed this extension do not need to include any data in the standard X.509 name or subjectAlternateName fields. The CertificateSigningRequests generated by the kubelet will not, nor will the certificates issued by `kubernetes.io/kube-apiserver-client-pod`.

Utility functions for embedding and extracting this extension from `crypto/x509`
Certificate and CertificateRequest objects are defined in
`k8s.io/component-helpers/kubernetesx509`.  They are intended to be used both by
core Kubernetes components and third-party software that needs to interact with
these certificates.

### PodIdentity Extension Node Restriction

The noderestriction admission plugin will check the consistency of PodIdentity extensions in CertificateSigningRequest objects created by `system:node:` identities.  The creation request will only be admitted if the PodIdentity extension refers to a pod that is currently running on the node that requested the certificate.

This upholds the Kubernetes node isolation guarantee by ensuring that
compromising a single node only results in compromises of workloads that have
actually been scheduled onto that node.

<<[UNRESOLVED]>>
Is it necessary or desirable to forbid non-kubelet identities from creating CertificateSigningRequest objects that embed a PodIdentity extension?
<<[/UNRESOLVED]>>

During Alpha, this capability will be behind the PodIdentityX509Support feature gate.

### PodCertificate Projected Volume Sources

The PodCertificate projected volume source instructs kubelet to inject a
credential bundle consisting of a private key and certificate chain into the
container fileystem.  kubelet handles generating the key, requesting a
certificate from the named signer, and renewing the certificate before it
expires.

The kubelet supports only a few key generation strategies, each named by a string
that encapsulates both the key type and any parameters necessary (for example,
RSA modulus size).  The intention is to offer a a tasting menu of reasonable key
choices, rather than offering a flexibility to a wide variety of parameters.
Key types supported by kubelet are "RSA2048", "RSA3072", "RSA4096", "ECDSAP256",
and "ECDSAP384".  If no key type is specified, kubelet defaults to "ECDSAP256".

<<[UNRESOLVED]>>
Should RSA2048 be offered?  @ahmedtd inclined to say no.  Are there any other key types that should be offered?
<<[/UNRESOLVED]>>

Once the key is generated, kubelet creates a CertificateSigningRequest that
contains an X.509 certificate request with no data in the name or
subjectAlternateName fields, but instead embeds an X.509 PodIdentity extension
with the details of the pod that mounted the volume.  The signer is intended to
treat this certificate request merely as an indication that it should issue a
certificate for the named pod, rather than trying to issue a certificate that
follows the structure of the certificate request.

The kubelet then starts a single-object informer on the CertificateSigningRequest object it just created, and waits until the CSR is Approved and Issued.  The kubelet then takes the private key it generated, plus the issued certificate chain, and writes them to the filesystem of the relevant Pod volume, concatenated into a single PEM-encoded file.  The first block in the PEM file contains the private key, and the remaining blocks contain the certificate chain in the same order they were in the CSR.

The kubelet strips any inter-block text comments and / or intra-block headers
that were present in the issued certificate chain from the signer.

The kubelet always specifies a 24-hour TTL on its certificate requests, although of course the signer is free to ignore the requested TTL and return a certificate with any TTL.  The kubelet will use the issued certificate unless the TTL is less than 1 hour from the current time.

During the initial volume mount, before the pod has started, any errors returned during the certificate issuance process will block pod startup.

The kubelet will begin trying to rotate the certificate once less than 50% of its TTL is remaining, or after 24 hours, whichever is sooner.  Errors in the certificate issuance process are treated as non-fatal (logged but not returned from the volume SetUp function) while the original certificate is still valid.  Once the original certificate has expired, errors are will be returned from the volume SetUp function, because it's better for the pod to noisily go unhealthy than to continue to run with an expired certificate.

<<[UNRESOLVED]>>
The rotation/renewal behavior is minimally tunable.  Is greater
tunability needed?  For example, is it necessary to let the pod author set the
TTL Kubelet passes in the CSR object, or to let the pod author control the
rotation safety threshold?
<<[/UNRESOLVED]>>

#### API Object Diff

```diff
+// PodCertificateProjection provides a private key and X.509 certificate in
+// a combined file.
+type PodCertificateProjection struct {
+	// Kubelet's generated CSRs will be addressed to this signer.
+	SignerName string `json:"signerName,omitempty" protobuf:"bytes,1,rep,name=signerName"`
+
+	// The type of keypair Kubelet will generate for the pod.
+	//
+	// Valid values are "RSA2048", "RSA3072", "RSA4096", "ECDSAP256", and
+	// "ECDSAP384".  If left empty, Kubelet defaults to "ECDSAP256".
+	KeyType string `json:"keyType,omitempty" protobuf:"bytes,2,rep,name=keyType"`
+
+	// Write the credential bundle at this path in the projected volume.
+	CredentialBundlePath string `json:"credentialBundlePath,omitempty" protobuf:"bytes,3,rep,name=credentialBundlePath"`
+}

// ...

// Projection that may be projected along with other supported volume types
type VolumeProjection struct {
  //...

	ServiceAccountToken *ServiceAccountTokenProjection `json:"serviceAccountToken,omitempty" protobuf:"bytes,4,opt,name=serviceAccountToken"`
+
+	// Projects an auto-rotating credential bundle (private key and certificate
+	// chain) that the pod can use either as a TLS client or server.
+	//
+	// Kubelet generates a private key and uses it to send a
+	// CertificateSigningRequest to the named signer.  Once the signer approves
+	// the request and issues a certificate chain, Kubelet writes the key and
+	// certificate chain to the pod filesystem.  The pod does not start until
+	// certificates have been issued for each podCertificate projected volume
+	// source in its spec.
+	//
+	// Kubelet will begin trying to rotate the certificate after 50% of the leaf
+	// certificate's lifetime has elapsed, or 24 hours, whichever is shorter.
+	//
+	// The credential bundle is a single file in PEM format.  The first PEM
+	// entry is the private key, and the remaining PEM entries are the
+	// certificate chain issued by the signer (typically, signers will return
+	// their certificate chain in leaf-to-root order).
+	//
+	// The named signer controls chooses the format of the certificate it
+	// issues; consult the signer implementation's documentation to learn how to
+	// use the certificates it issues.
+	//
+	// +featureGate=PodCertificateProjection
+ // +optional
+	PodCertificate *PodCertificateProjection `json:"podCertificate,omitempty" protobuf:"bytes,6,opt,name=podCertificate"`
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

### API Server PodIdentity Extension Support

The kube-apiserver is extended to authenticate TLS client certificates that:

* Are signed by the existing `--client-ca-file` CA,
* Contain empty Name and SubjectAlternateName data, and
* Contain an X.509 PodIdentity extension.

These certificates are translated to an authenticator.Response that has:
* `User.Name` set to
  `system:serviceaccount:{PodIdentity.Namespace}:{PodIdentity.ServiceAccountName}`.
* `User.Extra["authentication.kubernetes.io/pod-name"]` set to
  `{PodIdentity.PodName}`
* `User.Extra["authentication.kubernetes.io/pod-uid"]` set to
  `{PodIdentity.PodUID}`
* `User.Extra["authentication.kubernetes.io/node-name"]` set to
  `{PodIdentity.NodeName}`

The net effect is that authenticating via a PodIdentity-bearing cert has the
same object-bound behavior as authenticating via a bound service account token.

During Alpha, this feature will be gated by the PodIdentityX509Support feature
gate.

### API Server Pod Client Certificate Approver / Signer

The kube-controller-manager is extended to support a new certficate signer,
`kubernetes.io/kube-apiserver-client-pod`.

The approver for this signer name automatically approves
CertificateSigningRequests that:
* Contain no Name or Subject Alternate Name data, 
* Embed a PodIdentity extension, and
* Are requested by node identities.

(The approver relies on the noderestriction plugin to verify that the pod and
node in the extension are consistent.)

The signer for this signer name issues certificates for all approved CSRs
addressed to the signer name.  The certificates it issues contain no Name or
Subject Alternate Name data, embed a PodIdentity extension, and have key usages
consistent with the certificate being used as a TLS client certificate. It uses
the same CA key and certificates as `kubernetes.io/kube-apiserver-client`.

During Alpha, this feature is gated by the
`IssueKubeAPIServerPodClientCertificates` feature gate.

### Client-go InClusterConfig Enhancements

<<[UNRESOLVED not prototyped yet ]>>
  TBD
<<[/UNRESOLVED]>>

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

- <test>: <link to test coverage>

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

- Feature Gate: PodIdentityX509Support
  - Components depending on the feature gate: kube-apiserver
- Feature Gate: PodCertificateProjection
  - Components depending on the feature gate: kubelet
- Feature Gate: IssueKubeAPIServerPodClientCertificates
  - Components depending on the feature gate: kube-controller-manager


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
