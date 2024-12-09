# KEP-267: Kubelet server certificate bootstrap and rotation

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kubelet uses client certificates to authenticate to kube-apiserver. A kubelet also can use serving certificates. The kubelet itself exposes an https endpoint for certain features. To secure these, the kubelet can do one of:

- use provided key and certificate, via the --tls-private-key-file and --tls-cert-file flags
- create self-signed key and certificate, if a key and certificate are not provided
- request serving certificates from the cluster server, via the CSR API

## Motivation

The client certificate provided by TLS bootstrapping is signed, by default, for client auth only, and thus cannot be used as serving certificates, or server auth.

This proposal covers a process for generating a key locally and then issuing a
Certificate Signing Request to the cluster API server to get an associated
certificate signed by the cluster Certificate Authority. Also, as certificates
approach expiration, the same mechanism will be used to request an updated
certificate.

### Goals

- Allow Nodes/Kubelet to request a certificate to an external authority instead of using self signed certificates during bootstrap.
- Allow Nodes/Kubelet to request renewals of their certificates to an external authority.


## Design Details

Kubernetes v1.8 and higher kubelet implements features for enabling rotation of its client and/or serving certificates.

You can configure the kubelet to rotate its client certificates by creating new CSRs as its existing credentials expire. To enable this feature, user can enable the rotateCertificates field of kubelet configuration file.

The CSR approving controllers implemented in core Kubernetes do not approve node serving certificates for security reasons. To use RotateKubeletServerCertificate operators need to run a custom approving controller, or manually approve the serving certificate requests.

A deployment-specific approval process for kubelet serving certificates should typically only approve CSRs which:

1. are requested by nodes (ensure the spec.username field is of the form system:node:<nodeName> and spec.groups contains system:nodes)

2. request usages for a serving certificate (ensure spec.usages contains server auth, optionally contains digital signature and key encipherment, and contains no other usages)

3. only have IP and DNS subjectAltNames that belong to the requesting node, and have no URI and Email subjectAltNames (parse the x509 Certificate Signing Request in spec.request to verify subjectAltNames)

### Test Plan


[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates


##### Unit tests

- `k8s.io/kubernetes/pkg/kubelet/certificate/kubelet.go`: `2024-09-12` - `57.7`
- k8s.io/kubernetes/pkg/kubelet/certificate/transport.go: `2024-09-12` - `64.1`
- k8s.io/kubernetes/pkg/kubelet/certificate/bootstrap/bootstrap.go: `2024-09-12` - `50`

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

##### e2e tests

Any cluster running e2e tests with the feature enabled will be exercising the feature.

A job using the built-in CSR approver will be added exercising all the Conformance e2e tests.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Real world usage

#### GA

- Real world usage
- Opt-in built-in Node CSR approver on cloud-controller-manager so users does not have to depend on external components to use this feature
- No outstanding bugs or limitation


### Upgrade / Downgrade Strategy

The feature is beta enabled by default since v1.12 and no changes are required for GA, only add a new opt-in builtin CSR approver that does not impact the upgrade downgrade strategy

### Version Skew Strategy

The feature is beta enabled by default since v1.12 and no changes are required for GA, only add a new opt-in builtin CSR approver that does not impact the version skew trategy.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?


- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: RotateKubeletServerCertificate
  - Components depending on the feature gate: kubelet
- [x] Other
  - Describe the mechanism: Kubelet config option `ServerTLSBootstrap`
  - Will enabling / disabling the feature require downtime of the control
    plane? Yes, in case the control plane nodes use the existing feature
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? Yes

###### Does enabling the feature change any default behavior?

It has to be opted-in

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

The feature is per node, it requires to restart the kubelet

###### What happens if we reenable the feature if it was previously rolled back?

The feature is tied to the lifecycle of the node, restarting the kubelet allows to disable or reenable without any problem.

###### Are there any tests for feature enablement/disablement?

This feature is in beta since 1.12, with more than 20 releases running in production.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

This feature adds a dependency on an external approver for the certificate generated by the node, communication failures or certificate problems betweeen the external certificate approver can cause problems with the Node communication if the certificates are not signed or expire.

###### What specific metrics should inform a rollback?

  - kubelet_server_expiration_renew_errors
  - kubelet_certificate_manager_server_rotation_seconds
  - kubelet_certificate_manager_server_ttl_seconds
  - kubelet_client_expiration_renew_errors

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This feature is in beta since 1.12, with more than 20 releases running in production.


###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

`kubectl get csr -o json|jq '.items[]|select(.spec.signerName == "kubernetes.io/kubelet-serving")`

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [x] Other (treat as last resort)
  - Details:
    - `kubectl get -o json|jq '.items[]|select(.spec.signerName == "kubernetes.io/kubelet-serving")|`

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

99.9% of Certificate Signing Request from nodes per day are accepted

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: kubelet_server_expiration_renew_errors
  - Components exposing the metric: kubelet
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?


### Dependencies


###### Does this feature depend on any specific services running in the cluster?

It requires a certificate approver for the Certificate Signing Requests from the Nodes

### Scalability

###### Will enabling / using this feature result in any new API calls?

The node startup is impacted since the bootstrap now requires a handshake to sign the Certificate Request of each node

###### Will enabling / using this feature result in introducing new API types?

NO, all API types required are already GA

###### Will enabling / using this feature result in any new calls to the cloud provider?

Yes, it requires a CSR approver

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

New CSR objects will be created per Nde

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Node readiness/startup

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Nodes will not be able to bootstrap or renew the certificate, but independently of the feature, this is required to communicate with the control plane that in this scenario will be unavailable.

###### What are other known failure modes?

Problems with the CSR approver will impact node boostrap and certificate renewal, causing issues with the communication between nodes and control plane.

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History
- Alpha in [Certificate rotation for kubelet server certs. #45059](https://github.com/kubernetes/kubernetes/pull/45059)
- Beta in [kubelet: Move RotateCertificates to the KubeletConfiguration struct #63912](https://github.com/kubernetes/kubernetes/pull/63912)

## Drawbacks


## Alternatives
