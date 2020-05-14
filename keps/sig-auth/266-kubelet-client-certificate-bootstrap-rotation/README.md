# KEP-266: Kubelet client certificate bootstrap and rotation

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [During Kubelet Boot Sequence](#during-kubelet-boot-sequence)
  - [As Expiration Approaches](#as-expiration-approaches)
  - [Implementation notes](#implementation-notes)
  - [Certificate Approval](#certificate-approval)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
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
        - Atomically replace the `kubelet-client-current.pem` symlink to point to the new cert/key pair.

### Implementation notes

* On startup, obtaining a certificate (via bootstrap or certificate renewal) blocks
  the goroutine that contacts the API server to fetch pods the kubelet should run
* On startup, obtaining a certificate (via bootstrap or certificate renewal) does *not*
  block the goroutine that runs static pods. This ensures that a kubelet self-hosting the
  kube-apiserver it speaks to can successfully start.
* The kubelet will wait up to five minutes to obtain a valid certificate on startup,
  then restarts itself. On restart, the last private key used to generate a CSR is reused,
  so it can resume waiting for an existing CSR without creating another one.
* On startup, when configured with both a bootstrap kubeconfig and a kubeconfig file,
  the kubelet validates the credentials in the kubeconfig exist, load successfully, and are not expired.
  If any of those checks fail, it reruns the bootstrap step using the bootstrap kubeconfig.

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

## Production Readiness Review Questionnaire

### Feature enablement and rollback

* **How can this feature be enabled / disabled in a live cluster?**

  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: RotateKubeletClientCertificate
    - Components depending on the feature gate: kubelet
  - [x] Other
    - Describe the mechanism:
      - kubelet flag: `--rotate-certificates=true`
      - kubelet config field: `rotateCertificates: true`
    - Will enabling / disabling the feature require downtime of the control plane?
      - No
    - Will enabling / disabling the feature require downtime or reprovisioning of a node?
      - This feature is generally expected to be configured on new nodes
      - This feature is opt-in even after GA, so upgrades of existing nodes with existing credential provisioning strategies are unaffected

* **Does enabling the feature change any default behavior?**

  * Default behavior does not change
  * This feature is opt-in, even after GA, so upgrades of existing nodes with existing credential provisioning strategies are unaffected

* **Can the feature be disabled once it has been enabled (i.e. can we rollback the enablement)?**

  * A cluster can run with a mix of nodes that use the feature and nodes that do not
  * The feature can be turned off on a node after it has been enabled
  * The node deployer must provide a valid and sufficiently long-lived `--kubeconfig` credential (as before this feature)

* **What happens if we reenable the feature if it was previously rolled back?**

  * The kubelet will resume attempting to obtain and rotate client credentials

* **Are there any tests for feature enablement/disablement?**

  * Enabling/disabling this feature does not affect API availability
  * The CertificateSigningRequests created by this feature have a maximum lifetime of 24 hours before they are removed by the CSR controller

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**

  * The recommendation is to enable this feature on new nodes, which would prevent impact to existing workloads
  * Ensure kube-controller-manager has a cluster signer cert/key that matches the client CA bundle given to kube-apiserver
  * Ensure Kubelets have permission to create CertificateSigningRequests
  * Ensure Kubelets have permission to be approved for node client certificates

* **What specific metrics should inform a rollback?**

  - `kubelet_certificate_manager_client_expiration_seconds - now()` on nodes using this feature should be > 5% of cluster expiration duration
  - `kubelet_certificate_manager_client_expiration_renew_errors` on nodes using this feature should not be increasing

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**

  * Started kubelet without this feature enabled with `--kubeconfig=static.kubeconfig`
    * verified it registered successfully with the API server
  * Restarted the kubelet with `--rotate-certificates=true`,
    `--bootstrap-kubeconfig=bootstrap.kubeconfig` containing valid bootstrap credentials,
    and `--kubeconfig=dynamic.kubeconfig` pointed at a non-existent file
    * verified a CSR was created
    * verified `dynamic.kubeconfig` was created
    * verified certificate was created in `/var/lib/kubelet/pki`
  * Restarted the kubelet without the feature enabled with `--kubeconfig=static.kubeconfig`
    * verified it established contact with the API server

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?**
  
  * No

### Monitoring requirements

* **How can an operator determine if the feature is in use by workloads?**

  * This feature is used by kubelets, not workloads

* **What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?**

  - Metric: `kubelet_certificate_manager_client_expiration_seconds`
    - Description: Gauge of the lifetime of the active certificate. The value is the date the certificate will expire in seconds since January 1, 1970 UTC.
    - Components exposing the metric: kubelet

  - Metric: `kubelet_certificate_manager_client_expiration_renew_errors`
    - Description: Counter of certificate renewal errors.
    - Components exposing the metric: kubelet

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

  * `kubelet_certificate_manager_client_expiration_seconds - now()` should be >= 5% of cluster signing duration

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
  
  * The `certificates.k8s.io` API group must be enabled
  * An approver for kubelet client certificate requests must be running.
    A default implementation is built into kube-controller-manager in the `csrapproving`
    controller loop that uses authorization checks to determine if a request should be approved.
  * A signer for kubelet client certificate requests must be running.
    A default implementation is built into kube-controller-manager in the `csrsigning`
    controller loop that signs approved CSRs with the signing certificate passed to
    kube-controller-manager using `--cluster-signing-{cert,key}-file` flags.

### Scalability

* **Will enabling / using this feature result in any new API calls?**

  * For each attempt by a kubelet to obtain a certificate, it will:
    * Create a CertificateSigningRequest API object
    * Watch the single CertificateSigningRequest API object until it is approved, denied, or deleted
    * The kubelet attempts to obtain a certificate:
      * on first bootstrap startup
      * on certificate expiration (at 80% of lifetime, jittered +/- 10%), lifetime controlled by
        `--experimental-cluster-signing-duration`, defaults to 1 year
  
  * The kube-controller-manager will list/watch CertificateSigningRequest objects using an informer
    * For each kubelet client certificate CertificateSigningRequest,
      the approving controller will make 1 update request to approve or deny
    * For each approved kubelet client certificate CertificateSigningRequest,
      the signing controller will make 1 update request to add the issued certificate
    * For each CertificateSigningRequest, the csrcleaner controller will make
      1 delete request to clean up the API object

* **Will enabling / using this feature result in introducing new API types?**

  * CertificateSigningRequest API type is introduced for this feature.
  * The lifetime of these objects is limited to 24 hours by the csrcleaner controller,
    or 1 hour for requests that have been approved or denied.
  * 1 CertificateSigningRequest in existence per node is expected.

* **Will enabling / using this feature result in any new calls to cloud
  provider?**

  * Not in the default kube-controller-manager approver/signer implementation

* **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**

  * Existing types are not modified

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**

  * Time to bootstrap the initial credential is added to first Kubelet startup,
    but node setup time is not a component of existing SLIs/SLOs.
    Once credentials are established, Kubelet -> API server communication is unchanged.

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**

  * `kube-controller-manager` becomes responsible for signing certificate requests for nodes.
    Adding large numbers of nodes to a cluster simultaneously can increase CPU of the signer.

### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?**

  * Kubelet client credentials allow it to authenticate to the API server.
    If the API server is unavailable, it will be unable to bootstrap/rotate those credentials,
    but since the only use for those credentials is speaking with the API server,
    no other aspects of the kubelet are interrupted.

* **What are other known failure modes?**

  * If a kubelet cannot renew a client credential, it retries in the background.
    - Detection:
      - `kubelet_certificate_manager_client_expiration_renew_errors` metric with non-zero value
      - `kubelet_certificate_manager_client_expiration_seconds` less than 5% of expiration limit
      - Persistent kubelet client CertificateSigningRequests that are not approved or are not issued
    - Mitigations:
      - Replace the node
      - Reconfigure the node with client cert bootstrap/rotation off
    - Diagnostics: 
      - kubelet logs:
        - certificate_manager and certificate_store errors when rotation fails, diagnostic messages at level 2 verbosity
      - kube-controller-manager logs
        - certificate_controller errors when sync fails, diagnostic messages at level 4 verbosity
      - watching CertificateSigningRequest API objects
      - audit logs for CertificateSigningRequest API requests
    - Testing:
      - certificate manager unit testing
  
  * On startup, if a kubelet cannot obtain a client credential, it restarts after 5 minutes.
    - Detection:
      - Kubelet process restart count
      - Presence of kubelet client CertificateSigningRequests that are not approved or are not issued older than 5 minutes
    - Mitigations: 
      - Replace the node
      - Reconfigure the node with client cert bootstrap/rotation off
    - Diagnostics: 
      - kubelet logs:
        - certificate_manager and certificate_store errors when rotation fails, diagnostic messages at level 2 verbosity
      - kube-controller-manager logs
        - certificate_controller errors when sync fails, diagnostic messages at level 4 verbosity
    - Testing: 
      - certificate manager unit testing for resuming a pre-existing CSR watch at startup

* **What steps should be taken if SLOs are not being met to determine the problem?**

  * Determine which step in certificate renewal is not successful:
    * CertificateSigningRequest creation
      * Symptom: No CertificateSigningRequest objects are created by a given node
      * Troubleshooting: kubelet logs
    * CertificateSigningRequest approval
      * Symptom: Kubelet client CertificateSigningRequest objects for a given node exist, but are unapproved
      * Troubleshooting: kube-controller-manager logs, permissions granted to CertificateSigningRequest requester
    * CertificateSigningRequest signing
      * Symptom: Kubelet client CertificateSigningRequest objects for a given node exist and are approved, but are not issued
      * Troubleshooting: kube-controller-manager logs, `--cluster-signing-{cert,key}-file` setup
    * CertificateSigningRequest use
      * Symptom: Kubelet client CertificateSigningRequest objects for a given node exist and are approved and issued, but Kubelet API requests are Unauthorized
      * Troubleshooting: kubelet logs, verify `--cluster-signing-cert-file` given to kube-controller-manager is included in `--client-ca` bundle given to kube-apiserver

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

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
