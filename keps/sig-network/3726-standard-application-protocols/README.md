# KEP-3726: Standard Application Protocols

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

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
  - [New Standard Protocols](#new-standard-protocols)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Adding new protocols](#adding-new-protocols)
  - [Followup work](#followup-work)
  - [Documentation change](#documentation-change)
  - [Test Plan](#test-plan)
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
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [x] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [x] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

There are cases where implementations implement different things for the same application protocol names. That has already caused issues for certain implementations to interoperate, with the possibility of more in the future. (See the example with GKE and Istio under [Motivation](#motivation))

This KEP suggests a different description for the `AppProtocol` field and proposes to declare standard Kubernetes protocol names for common protocols that are not IANA service names.


## Motivation

The lack of direct support for specifying application protocols for ports and the widespread use of implementation-specific annotations to mitigate it had led Kubernetes to add the `AppProtocol` field to the port spec.

While this is a good solution - we never came with recommended standards other than [IANA standard service names](https://www.iana.org/assignments/service-names) or a domain prefixed protocol.

This loose definition has led us to;
1. Have instances where implementations do different things for common protocols.
2. Have no support for implementations interoperability with different domain prefixed protocols (or a mix domain prefixed and non prefixed protocol) for the same port.

One good example for the first bullet is `HTTP2`. 
* In GKE you can use `appProtocol: HTTP2` and it will describe HTTP2 over TLS (https://cloud.google.com/kubernetes-engine/docs/how-to/secure-gateway#load-balancer-tls).
* While in Istio it will be h2c (HTTP2 over cleartext).

That creates a problem where users with GKE and Istio in their cluster can have very different behaviors for the same `appProtocol` value.


### Goals

* Rephrase AppProtocol field description
* Build consensus around how common (non IANA service names) should be implemented
* Help the broader community and specifically implementations to interoperate better
* Provide short and clear documentation for how AppProtocol values should be implemented
  * Update the appProtocol user documentation respectively


### Non-Goals

* Validate appProtocol values
* Monitor appProtocol implementations
* Support multiple AppProtocols values for the same port to improve interoperability (suggested as a followup work though)


## Proposal

~~Kubernetes supports the `appProtocol` field to provide a way to specify an application protocol for each Service port.~~

Kubernetes `appProtocol` field is used as a hint for implementations to configure the protocol used between the implementation and the application it exposes.

The [documentation](https://kubernetes.io/docs/concepts/services-networking/service/#application-protocol) for this field says that:

```Values should either be IANA standard service names or domain prefixed names such as mycompany.com/my-custom-protocol.```

This KEP proposes to declare standard Kubernetes protocol names for common protocols that are not IANA standard service names.

Those common protocols will be well defined strings prefixed with ‘k8s.io’. 
`k8s.io/h2c` as an example.

### New Standard Protocols
- 'k8s.io/http2'
- 'k8s.io/grpc'
- 'k8s.io/tcp'

### Risks and Mitigations

There are no real “risks”, primary concerns are:
1. End users will not migrate to `k8s.io/<>` values
2. It will take long time to implementations align with the new standards
3. We have no easy way to monitor who is aligned and who is not


## Design Details

At first, the collection of standard protocols is going to live in `ServicePort` and `EndpointPort` as part of the AppProtocol description.

We might revisit this decision in the future and suggest an alternative location based on the number of standard protocols we support.

Proposed changes to the API spec:

```go
type ServicePort struct {
  ... 
  ...

  // Used as a hint for implementations to
  // configure the protocol used between the
  // implementation and the application it exposes.
  // This field follows standard Kubernetes label syntax.
  // Valid values are either:
  //
  // * Un-prefixed protocol names - reserved for IANA standard service names (as per
  // RFC-6335 and https://www.iana.org/assignments/service-names).
  //
  // * Kubernetes standard names:
  //   * 'k8s.io/http2' - http2 over cleartext, aka 'h2c'. https://www.rfc-editor.org/rfc/rfc7540
  //   * 'k8s.io/grpc' - grpc traffic - see https://github.com/grpc/grpc/blob/v1.51.1/doc/PROTOCOL-HTTP2.md
  //   * 'k8s.io/tcp' - plain tcp traffic
  //
  // * Other protocols should use prefixed names such as
  // mycompany.com/my-custom-protocol.
  // +optional
  AppProtocol *string
```

same wording for type `EndpointPort`

### Adding new protocols

In order to be included in the collection, a new protocol must:
* Not be an [IANA standard service name](https://www.iana.org/assignments/service-names)
* Run on top of L4 protocol supported by Kubernetes Service
* Be supported in two or more implementations
* Be well defined and broadly used

<<[UNRESOLVED sig-network ]>>

if we require two implementations to support a protocol before we include it in the standard collection (i.e k8s.io prefixed collection) we create a situation where we force them to support their own domain prefixed values and k8s.io-prefixed values, have their users migrate to the k8s.io values once they are included, and also the k8s.io ones might end up not be quite the same definition as the implementation specific ones (as we see in the GKE & Istio example).

The proposed followup work might address this problem also when we turn the field into a list

<<[/UNRESOLVED]>>


### Followup work
To support implementations interoperability with different domain prefixed protocols (or a mix domain prefixed and non prefixed protocol) for the same port we need to turn `AppProtocol` to a list.

It is likely to be an API change but design details TBD.

### Documentation change

[kubernetes website](https://github.com/kubernetes/website/blob/main/content/en/docs/concepts/services-networking/service.md#application-protocol) will be changed accordingly

### Test Plan

N/A

This KEP does not plan to change code, just documentation.

### Graduation Criteria

### Upgrade / Downgrade Strategy

N/A

This KEP does not plan to change code, just documentation.

### Version Skew Strategy

N/A

This KEP does not plan to change code, just documentation.

## Production Readiness Review Questionnaire


### Feature Enablement and Rollback

N/A

This KEP does not plan to change code, just documentation.


### Rollout, Upgrade and Rollback Planning

N/A

This KEP does not plan to change code, just documentation.

### Monitoring Requirements

N/A

This KEP does not plan to change code, just documentation.

### Dependencies

N/A

This KEP does not plan to change code, just documentation.

### Scalability

N/A

This KEP does not plan to change code, just documentation.

### Troubleshooting

N/A

This KEP does not plan to change code, just documentation.

## Implementation History

N/A

## Drawbacks

* The collection of the standard protocols can become stale fairly quick when new protocols are implemented before we decide to declare them as part of k8s.io common collection. That can lead to a the current state again where implementations already implement support without a prefix (although they should not) OR with a domain prefix.


## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->
