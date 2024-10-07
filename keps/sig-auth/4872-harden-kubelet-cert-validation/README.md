# KEP-4872: Harden Kubelet Serving Certificate Validation in Kube-API server 

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Impact of node impersonation](#impact-of-node-impersonation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Enabling the feature](#enabling-the-feature)
    - [Metrics](#metrics)
  - [TLS insecure](#tls-insecure)
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
- [Infrastructure Needed](#infrastructure-needed)
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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This proposal aims to enhance the security of the Kube API server by validating the Common Name (CN) of the kubelet's serving certificate to ensure it matches the expected node name.
This validation prevents a compromised node that has obtained a certificate for an IP address it does not own from using it to impersonate another node.

## Motivation

In cloud environments, IPs can change rapidly due to the ephemeral nature of the infrastructure.
If IPs or machines rotate faster than the expiration frequency of kubelet serving certificates, a certificate issued to an old node could be used to respond to requests aimed at a new node, provided they share an IP.

In addition, in on-premises environments, verifying that the IP addresses in a Certificate Signing Request (CSR) are owned by the requesting node can be challenging due to the lack of a reliable source of truth for IP ownership.
Even when such a source exists, integration can be complex, leading to unsafe practices like auto-approval of CSRs without a strong guarantee of IP ownership.
This vulnerability can be exploited through ARP poisoning or other routing attacks, allowing a rogue node to obtain a certificate for an IP it does not own and reroute traffic to itself.

When the Kube API server connects to a kubelet, it verifies that the serving certificate is signed by a trusted CA and that the IP or hostname it’s connecting to is included in the certificate's SANs.
If a rogue node obtained a certificate for an IP it does not own and reroute traffic to itself, it would be able to impersonate a Node that reports that IP.

### Impact of node impersonation

Provided an actor with control of a node can impersonate another node, the impact would be:

* Break confidentiality of the requests sent by the Kube-API server to the kubelet (e.g kubectl exec/logs).These are usually user-driven requests. That gives the threat actor the possibility of producing incorrect or mis-leading feedback. In the exec case, it could allow a threat actor to issue prompts for credentials. In addition, the exec commands might contain user secrets.  
* Break confidentiality of credentials if the client uses token based authentication. This is probably more common for non Kube-API server clients, given mTLS is common for Kube-API server to kubelet communication.

### Goals

* Ensure the Kube API server validates that the node’s serving certificate's CN matches the expected node name.  
* Prevent rogue nodes from using certificates issued for IPs they do not own.

### Non-Goals

* This proposal does not address certificate validation for clients other than the Kube API server, such as metrics scrapers. However, we'll consider an implementation in client-go that could be used by those other clients.

## Proposal

We propose that the Kube API server is modified to validate the Common Name (CN) of the kubelet's serving certificate is equal to `system:node:<nodename>`.
`nodename` is the name of the Node object as reported by the kubelet. When the Kube-API server connects to the kubelet server (e.g. for logs, exec, port-forward), it always knows the Node it's connecting to.

### User Stories (Optional)

#### Story 1

As a cluster administrator, I want to ensure that kubelet serving certificates are validated based on the node name, reducing the risk of IP-based impersonation attacks.

#### Story 2

As a cluster administrator using custom serving certificates for the kubelet server, I want to be able to disable the Subject's CN validation.

### Notes/Constraints/Caveats (Optional)

When the kubelet requests a certificate through a CSR, it sets the CN to `system:node:<nodename>`, enforced by the admission controller as per [PR \#126015](https://github.com/kubernetes/kubernetes/pull/126015).

However, certificates issued manually or through other mechanisms may not follow this convention.
With the new validation, any certificate not following this `system:node:<nodename>` convention will be deemed invalid by the Kube API server.
This will require cluster administrators to reissue any non-conforming certificates before enabling this feature.

### Risks and Mitigations

This could disrupt existing clusters that are using custom kubelet serving certificates.
These clusters will need to reissue their certificates before enabling this feature. We will allow to disable the validation through a command-line flag to allow for a smooth transition.

## Design Details

### Enabling the feature

We will introduce a feature flag `KubeletCertCNValidation` that will gate the usage of the new validation.
This gate will start off by default in Alpha, will be turned on by default in Beta and will be removed in GA.

In addition, we will allow to disable the validation through a command-line flag `--disable-kubelet-cert-cn-validation`.
This flag can only be set if the `KubeletCertCNValidation` feature flag is enabled.
This flag will allow cluster administrators to opt-out of this validation if they are using custom kubelet serving certificates that don't follow the `system:node:<nodename>` convention even after the feature gate is removed.

#### Metrics

In order to help cluster administrators determine if it's safe to enable the feature, we propose to add a new metric `kube_apiserver_validation_kubelet_cert_cn_errors` that will track the number of errors due to the new CN validation.
If the feature gate is disabled, we will still add the validation code to the HTTP transport, however, if the validation fails we won't return an error, we will just increment the metric counter.
In addition, we will log the error including the node name, so cluster administrators can identify which nodes are affected and need to reissue their certificates.

We purposefully don't add the node name to the metric to avoid a high cardinality.
The purpose of the metric is to easily/cheaply tell administrators if they can flip the feature on or not. If the answer is no (counter is greater than 0), the rest of the necessary information to detect the offending nodes will come from logs.

Given that running the validation to feed the metric still has a cost, we won't run it if the validation is explicitly disabled with `--disable-kubelet-cert-cn-validation`.

We will remove the metric once the feature is GA.

> TODO: let's discuss this in the review. We could consider adding the node name to the metric or even keeping the metric post GA if it's valuable.

### TLS insecure

Currently, if the Kube-API server is not configured with a `--kubelet-certificate-authority` the TLS client for kubelet server will skip the server certificate validation.
Additionally, `logs` requests allow to configure `InsecureSkipTLSVerifyBackend` per request to skip the server certificate validation.

To align with this behavior, we won't execute the CN validation if `--kubelet-certificate-authority` is not set or if `InsecureSkipTLSVerifyBackend` is set to true.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

Unit tests will be added along with any new code introduced.

Existing test coverage for the packages we anticipate modifying:

- `k8s.io/kubernetes/pkg/kubelet/client`: `2024-10-07` - `28.2`
- `k8s.io/client-go/transport`: `2024-10-07` - `59.4`

##### Integration tests

Integration tests will be added to ensure the following:
* An error is returned if `--disable-kubelet-cert-cn-validation` is set but `KubeletCertCNValidation` feature flag is not enabled.
* Validation for custom certificates works if feature flag is not enabled.
* Validation for custom certificates works if feature flag enabled and `--disable-kubelet-cert-cn-validation` is set to true.
* Validation for custom certificates fails if feature flag enabled and `--disable-kubelet-cert-cn-validation` is set to false or not set.
* Validation for kubernetes issued certificates works if feature flag enabled and `--disable-kubelet-cert-cn-validation` is set to false or not set.

##### e2e tests

End-to-end tests won't be needed as unit and integration tests will cover all the scenarios.

### Graduation Criteria

#### Alpha

* Add feature flag for gating usage, off by default
* Add flag to disable extra validation
* Unit and integration tests

#### Beta
* Address user reviews and iterate if needed
* Feature flag on by default

#### GA
* Remove feature flag

### Upgrade / Downgrade Strategy

Once feature flag is on by default (starting in Beta), administrators using custom serving certs
can use the proposed flag to disable the extra validation and maintain current behavior.
They will be able to use this flag even after the feature flag is removed.

### Version Skew Strategy

Not applicable.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `KubeletCertCNValidation`
  - Components depending on the feature gate: kube-apiserver
- [x] Other
  - Describe the mechanism: kube-apiserver command-line flag `--disable-kubelet-cert-cn-validation`
  - Will enabling / disabling the feature require downtime of the control
    plane? No. But requires restarting the kube-apiserver.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? No.

###### Does enabling the feature change any default behavior?

Yes. If a cluster is using custom kubelet serving certificates that don't follow the same convention as kubernetes issued certificates (CN is `system:node:<node-name>`),
enabling this feature will make any connection initiated by the kube-api server fail (logs, exec and port-forwarding).

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the feature can be disabled once enabled by just setting the command-line flag to true.

###### What happens if we reenable the feature if it was previously rolled back?

You just get back the new behavior with the extra cert validation, no extra considerations needed.

###### Are there any tests for feature enablement/disablement?

We will add integration tests to validate the enablement/disablement flow. Test cases specified in a previous section.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout can fail if the feature flag is not enabled but the command-line flag is set.

Already running workloads won't be impacted but cluster users won't be able to access the control plane if the cluster is single-node.

###### What specific metrics should inform a rollback?

Not applicable.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No. There is no data stored for this feature which persists between upgrade / downgrade, or between enable / disable.
The feature is purely an API server configuration option.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The cluster administrators can check the flags passed to the kube-apiserver if they have access to the control plane nodes.
If the `--disable-kubelet-cert-cn-validation` flag is not set or set to false, the feature is being used.
Alternatively the can check the `kubernetes_feature_enabled` metric.

###### How can someone using this feature know that it is working for their instance?

- [x] Other
  - Details: users can create a Node with a kubelet serving certificate that doesn't meet the CN requirements enforced by this validation (something different than `system:node:<node-name>`).Then run `kubectl logs` for any pod running in that node. If it returns an error for an invalid certificate, the feature is working.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The average `apiserver_request_duration_seconds` for logs/exec/port-forward requests is within reasonable limits.
A raising value after enabling this feature could signal overhead introduced by the extra validation.

> TODO: I expect the overhead to be negligible and probably to fall in within the standard deviation of the current average. Specially for long running requests like port-forward and exec. Is this even valuable to have here?

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `kube_apiserver_pod_logs_backend_tls_failure_total`
  - Components exposing the metric: kube-apiserver

> TODO: should `kube_apiserver_pod_logs_backend_tls_failure_total` reflect errors due to the new CN validation?
> It's technically a TLS failure, but it's not part of the base TLS client validations.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

We could add a metric specific to track the number of requests that failed due to the new CN validation. In addition, we could track the time spent per request on the CN validation.

However, we consider these metrics to not provide enough value to justify the work to maintain them.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. This only affects streaming APIs and these are not covered by SLIs/SLOs.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

Note: depending on the implementation (caching the client-go transport or not) there might be a slight increase in memory (due to one transport per node being cached) or in CPU usage (due to building the transport on the fly for every request). This should be negligible.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

It's part of the API server, so the feature will be unavailable.

###### What are other known failure modes?

- [API server can't connect to Nodes with custom kubelet serving certificates that don't follow the `system:node:<node-name>` convention]
  - Detection: `kubectl logs` returns a certificate validation error. 
  - Mitigations: disable the validation with the `--disable-kubelet-cert-cn-validation` flag.
  - Diagnostics: error is returned by the API server, no additional logging needed.
  - Testing: We will have tests for this, this is basically testing that the feature works. 

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

## Drawbacks

This could disrupt clusters that are using custom kubelet serving certificates. These clusters will need to reissue their certificates before enabling this feature.

## Alternatives

None.

## Infrastructure Needed

None.
