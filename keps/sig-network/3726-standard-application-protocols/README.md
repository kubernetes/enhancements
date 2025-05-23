# KEP-3726: Standard Application Protocols

<!-- toc -->
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

Those common protocols will be well defined strings prefixed with ‘kubernetes.io’. 
`kubernetes.io/h2c` as an example.

### New Standard Protocols
- 'kubernetes.io/h2c'
- 'kubernetes.io/ws'
- 'kubernetes.io/wss'

### Risks and Mitigations

There are no real “risks”, primary concerns are:
1. End users will not migrate to `kubernetes.io/<>` values
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

  // The application protocol for this port.
  // This is used as a hint for implementations to offer richer behavior for protocols that they understand.
  // This field follows standard Kubernetes label syntax.
  // Valid values are either:
  //
  // * Un-prefixed protocol names - reserved for IANA standard service names (as per
  // RFC-6335 and https://www.iana.org/assignments/service-names).
  //
  // * Kubernetes-defined prefixed names:
  //   * 'kubernetes.io/h2c' - HTTP/2 prior knowledge over cleartext as described in https://www.rfc-editor.org/rfc/rfc9113.html#name-starting-http-2-with-prior-
  //   * 'kubernetes.io/ws'  - WebSocket over cleartext as described in https://www.rfc-editor.org/rfc/rfc6455
  //   * 'kubernetes.io/wss' - WebSocket over TLS as described in https://www.rfc-editor.org/rfc/rfc6455
  //
  // * Other protocols should use implementation-defined prefixed names such as
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

if we require two implementations to support a protocol before we include it in the standard collection (i.e kubernetes.io prefixed collection) we create a situation where we force them to support their own domain prefixed values and kubernetes.io-prefixed values, have their users migrate to the kubernetes.io values once they are included, and also the kubernetes.io ones might end up not be quite the same definition as the implementation specific ones (as we see in the GKE & Istio example).

The proposed followup work might address this problem also when we turn the field into a list

<<[/UNRESOLVED]>>


### Followup work
- To support implementations interoperability with different domain prefixed protocols (or a mix domain prefixed and non prefixed protocol) for the same port we need to turn `AppProtocol` to a list.

It is likely to be an API change but design details TBD.

- Some implementations are trying to guess the application protocol in absence of `appProtocol`. We should consider adding a new protocol like `kubernetes.io/raw` to the collection that would instructs implementations to process requests as raw.
We need to look at combination of `appProtocol: kubernetes.io/raw` and the different supported port protocols (TCP, UDP, SCTP).  

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

* The collection of the standard protocols can become stale fairly quick when new protocols are implemented before we decide to declare them as part of kubernetes.io common collection. That can lead to a the current state again where implementations already implement support without a prefix (although they should not) OR with a domain prefix.


## Alternatives
