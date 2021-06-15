# KEP-2784: CSR Duration

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
    - [Scenario 1](#scenario-1)
    - [Scenario 2](#scenario-2)
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

Extend the Certificates API with a mechanism to allow clients to request a
specific duration for the issued certificate.

## Motivation

Certificates issued through the certificate signing requests API are not
revocable today.  Furthermore, clients lack a way to control the duration of the
issued certificate.  Certificate approvers may have trust distinctions for
different clients.  For example, if an approver knows that a client stores their
private key in a HSM, they may be willing to approve for a longer certificate
duration.  Signers lack a per-CSR mechanism to control the certificate duration,
and thus are forced to learn this information out of band.  The built-in signers
use the same duration for all issued certificates, with the default being a one
year certificate lifespan (that is irrevocable without rotating the signer).

To increase the overall security of the Kubernetes ecosystem and allow clients
to request a particular duration for issued certificates, we propose extending
the Kubernetes certificates signing request API with a new optional spec field
that can be used to request a particular duration.

### Goals

- Allow clients to request a duration for requested certificates
- Allow the CSR API to be easily [adopted in the Kubernetes ecosystem](https://github.com/jetstack/cert-manager/pull/3646)

### Non-Goals

- Requiring signers to honor the requested duration

## Proposal

Add a new field to the `spec` of the certificate signing requests API called
`expirationSeconds` that allows a client to request a desired duration for the
issued certificate.

### User Stories (Optional)

#### Story 1

1. Alice creates a certificate signing request with `spec.expirationSeconds` set to `600`
1. The request is approved
1. The signer issues the signed certificate for `600` seconds
1. Alice is able to use the certificate for `10` minutes, after which it expires

#### Story 2

1. A privileged component `X` issues certificates for clients using the CSR API
1. Based on its internal policy, it creates CSRs with `spec.expirationSeconds` set to `900`
1. `X` approves the CSR and fetches the certificate once the signer issues it
1. `X` validates that the signer honored the `spec.expirationSeconds` field
1. `X` gives the certificate to the client
1. The certificate automatically expires in `15` minutes

### Notes/Constraints/Caveats (Optional)

N/A

### Risks and Mitigations

This functionality will serve to reduce risk and increase security in the
Kubernetes ecosystem by helping client migrate away from long lived certificates.

In the worst case scenario, the new field will be ignored, which does not reduce
security from the status quo.

## Design Details

This design is centered around a change to the `CertificateSigningRequestSpec`
structs found in the `k8s.io/api/certificates/v1` and `k8s.io/api/certificates/v1beta1`
packages.

A new optional `ExpirationSeconds` field will be added to this struct.  The go
doc comment describes the behavior of this field.  A `*uint32` is used because
the field is optional, must be positive and must not overflow JSON parsers.

```go
// CertificateSigningRequestSpec contains the certificate request.
type CertificateSigningRequestSpec struct {
  // ... other fields omitted for brevity

  // go doc omitted for brevity
  SignerName string `json:"signerName" protobuf:"bytes,7,opt,name=signerName"`

  // expirationSeconds is the requested duration of validity of the issued
  // certificate. The certificate signer may issue a certificate with a different
  // validity duration so a client must check the delta between the notBefore and
  // and notAfter fields in the issued certificate to determine the actual duration.
  //
  // The v1.22+ in-tree implementations of the well-known Kubernetes signers will
  // honor this field as long as the requested duration is not greater than the
  // maximum duration they will honor per the --cluster-signing-duration CLI
  // flag to the Kubernetes controller manager.
  //
  // Certificate signers may not honor this field for various reasons:
  //
  //   1. Old signer that is unaware of the field (such as the in-tree
  //      implementations prior to v1.22)
  //   2. Signer whose configured maximum is shorter than the requested duration
  //   3. Signer whose configured minimum is longer than the requested duration
  //
  // The minimum valid value for expirationSeconds is 600, i.e. 10 minutes.
  //
  // +optional
  ExpirationSeconds *uint32 `json:"expirationSeconds,omitempty" protobuf:"varint,8,opt,name=expirationSeconds"`

  // go doc omitted for brevity
  Usages []KeyUsage `json:"usages,omitempty" protobuf:"bytes,5,opt,name=usages"`

  // ... other fields omitted for brevity
}
```

The name `expirationSeconds` was chosen to match existing art in the token request
API.  Similarly, the minimum valid duration was chosen to match the token request
API as well.  As this is a security related field, individuals may be encouraged
to set this value to the minimum valid value to maximize security.  Since a certificate
with a short lifetime will require frequent rotation before the current certificate
expires (say at `80%` the lifetime of the certificate given that CSR approval
and signing is asynchronous which necessitates a buffer), `10` minutes seems like an
appropriate minimum to prevent accidental DOS against the CSR API.  Furthermore,
`10` minutes is a short enough lifetime that revocation is not of concern.

### Test Plan

Unit tests covering:

1. Validation logic for minimum duration
2. `pkg/controller/certificates/authority.PermissiveSigningPolicy` updates to handle `expirationSeconds`

Integration test covering:

1. Creating and approving CSRs and asserting that certificate signer controllers such as
   `pkg/controller/certificates/signer.NewKubeAPIServerClientCSRSigningController` honor
   `spec.expirationSeconds` by checking the duration of the issued certificate.

### Graduation Criteria

#### Alpha

- This design will start at the beta phase and the functionality will always be enabled
- There will be no feature gate associated with this design (discussed below)

This design represents a small, optional change to an existing GA API.  Thus it
prioritizes rollout speed to allow clients to start using this functionality
sooner (to reap potential security benefits) at the cost of data durability
during version skews (discussed below).

#### Beta

- Feature fully implemented as described in this design
- Unit tests completed and enabled
- Integration test completed and enabled
- [CSR docs](https://kubernetes.io/docs/reference/access-authn-authz/certificate-signing-requests) updated with details about usage of the `spec.expirationSeconds` field

#### GA

- Confirm with cert-manager that the new functionality addresses their use case
- Wait one release after beta to allow bugs to be reported

The existing conformance tests for the certificates API (`test/e2e/auth/certificates.go`)
are sufficient coverage as the new functionality is optional.

### Upgrade / Downgrade Strategy

Generally speaking, the slow rollout of new fields and features over multiple
releases is (at least partially) required to preserve data durability.  That is,
during upgrade and downgrade scenarios, it is desirable that old and new servers
interpret the data correctly as defined by the feature being implemented.

In the case of this design, data durability is not of concern as clients, no matter
what, have to assume that signers may ignore the requested duration completely
even after they have been updated to understand the field (for example the client
could request a duration of a month but the signer could truncate the duration to
its internal maximum of two weeks).  Thus this design emphasizes feature rollout
speed to aid in ecosystem adoption instead of data durability.  Combined with the
simplicity of implementation and low risk nature of the proposal, the alpha stage
and related feature gate have been omitted from this design.

Clients that do not set the `spec.expirationSeconds` field will observe no change
in behavior, even during upgrades and downgrades.

### Version Skew Strategy

There are three actors we need to consider:

1. API Server
2. Controller manager
3. Clients that create CSRs

As noted above, old clients observe no change in behavior, thus we assume for the
discussion below that all clients have been upgraded and are attempting to set
the `spec.expirationSeconds` field.

Once the API server is upgraded, clients will be able to set the new field.  For
the purpose of this design, upgrading other components before the API server is
of no consequence as it is impossible to set the new field without the API server
knowing of its existence.

#### Scenario 1

1. Upgraded API server
2. Not upgraded (or partially upgraded) controller manager
3. Upgraded client

In this scenario, the requested `spec.expirationSeconds` may be ignored because
the controller manger will not understand this field.  This is harmless and
represents the status quo.

#### Scenario 2

1. Partially upgraded API server
2. Upgraded controller manager
3. Upgraded client

In this scenario, the requested `spec.expirationSeconds` may be ignored because
old API servers will silently drop the field on update.  This is harmless
represents and the status quo.

The CSR API is resilient to split brain scenarios as unknown fields are silently
dropped and the `spec` fields are immutable after creation [1] [2] [3].

[1]: https://github.com/kubernetes/kubernetes/blob/24b716673caae31f070b06a337bc12c97ff1d4cb/pkg/registry/certificates/certificates/strategy.go#L104-L112
[2]: https://github.com/kubernetes/kubernetes/blob/24b716673caae31f070b06a337bc12c97ff1d4cb/pkg/registry/certificates/certificates/strategy.go#L175-L176
[3]: https://github.com/kubernetes/kubernetes/blob/24b716673caae31f070b06a337bc12c97ff1d4cb/pkg/registry/certificates/certificates/strategy.go#L297-L298

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- Other
  - Describe the mechanism:
      As written above, this functionality will be enabled by default with no configuration options.
  - Will enabling / disabling the feature require downtime of the control
    plane?
      N/A
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
      N/A

###### Does enabling the feature change any default behavior?

Existing clients would continue to leave `spec.expirationSeconds` unset and thus
would observe no difference in behavior.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

N/A

###### What happens if we reenable the feature if it was previously rolled back?

N/A

###### Are there any tests for feature enablement/disablement?

Unit and integration tests will cover cases where `spec.expirationSeconds` is
specified and cases where it is left unspecified.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Since it is optional for signers to honor `spec.expirationSeconds`, this design
is fully tolerant of API server and controller manager rollouts/rollbacks that
fail or get wedged in a partial state.  The worse case scenario is that the
`spec.expirationSeconds` field is ignored, which mimics the status quo.  Clients
must always check the duration of the issued certificate to determine if the
requested `spec.expirationSeconds` was honored.

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

// TODO //
<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

N/A

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Use `kubectl` to determine if CSRs with `spec.expirationSeconds` set are being
created.  Audit logging could also be used to determine this.

###### How can someone using this feature know that it is working for their instance?

- API `.status`
  - Condition name: `Approved` `=` `true`
  - Other field:
      Check that the issued certificate in `.status.certificate` has the correct duration

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This design does not make any meaningful change to the SLO of the Kubernetes CSR
API.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- Details: Check the Kubernetes audit log from CRUD operations on CSRs.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- API Server
  - Usage description: hosts the CSR API
    - Impact of its outage on the feature: CSR API will be unavailable
    - Impact of its degraded performance or high-error rates on the feature:
      + Signers may have difficulty issuing certificates
      + Clients may have to wait a long time for certificates to be issued
- etcd
  - Usage description: stores data for the CSR API
    - Impact of its outage on the feature: CSR API will be unavailable
    - Impact of its degraded performance or high-error rates on the feature:
      + Signers may have difficulty issuing certificates
      + Clients may have to wait a long time for certificates to be issued
- Kubernetes controller manager
  - Usage description: hosts the in-tree signer controllers
    - Impact of its outage on the feature: in-tree signers will be unavailable
    - Impact of its degraded performance or high-error rates on the feature:
      + Clients may have to wait a long time for certificates to be issued

### Scalability

###### Will enabling / using this feature result in any new API calls?

An increase in CSR CRUD operations as clients begin requesting shorter certificates
and rotating them more often due to the reduced lifespan.

###### Will enabling / using this feature result in introducing new API types?

N/A

###### Will enabling / using this feature result in any new calls to the cloud provider?

N/A

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

1. The `spec.expirationSeconds` will increase the size of CSR objects by 32 bits
2. Many short lived CSR objects will be created if clients request very short
   durations.  These will be automatically garbage collected via a pre-existing
   controller once the issued certificate has expired or after one hour,
   whichever is shorter.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

N/A

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

If many clients request short lived certificates and rotate them often, the
Kubernetes controller manager will have an increase in CPU usage due to the extra
signing operations.  The signer controllers hosted by KCM are multi-threaded to
quickly handle bursts of concurrent requests.  The API server and etcd will see
higher CPU and IO to process these requests.

### Troubleshooting

To aid in debugging, the printing and describe functionality of `kubectl` will be
enhanced to show the `spec.expirationSeconds` field as a human friendly duration.
As before, issued certificates can be printed via tools such as `openssl`.

###### How does this feature react if the API server and/or etcd is unavailable?

The semantics of the Kubernetes CSR API do not change in this regard.

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

- 1.22: 2021-06-17: KEP written

## Drawbacks

N/A

## Alternatives

An alternative to creating a new field in the Kubernetes CSR REST API would be to
add an optional but critical x509 extension to the PEM encoded x509 CSR contained
in the `spec.request` field.  This extension would then serve to communicate the
desired `expirationSeconds`.  It is unclear what encoding format would be used for
this extension, perhaps ASN.1 or JSON.  This approach offer no benefits over the
design presented above, and would likely make it far more difficult to adopt this
functionality in the Kubernetes ecosystem.

## Infrastructure Needed (Optional)

N/A
