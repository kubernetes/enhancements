---
kep-number: 0
title: Move Ingress to the networking.v1beta1 API group
authors:
  - "@bowei"
owning-sig: sig-network
participating-sigs:
reviewers:
  - "@aledbf"
approvers:
  - "@thockin"
  - "@caseydavenport"
editor:
creation-date: 2018-01-25
last-updated: 2018-02-01
status: implementable
see-also:
replaces:
superseded-by:
---

# Move Ingress to the networking.v1beta1 API group

## Summary

Copy the Ingress resource from the current API group (extensions.v1beta1) to
networking.v1beta1 and prepare the resource to go to GA as-is as the Ingress.V1.

## Motivation

The `extensions` API group is considered deprecated.  Ingress is the last
non-deprecated API in that group.  All other types have been migrated to other
permanent API groups.  Such an API group migration takes three minor version
cycles (~9 months) to ensure compatibility. This means any API group movement
should be started sooner rather than later.

The Ingress resource has been in a beta state for a *long* time (first commit
was in Fall 2015). While the interface [is not perfect][survey], there are many
[independent implementations][impls] in active use.

[survey]: https://github.com/bowei/k8s-ingress-survey-2018
[impls]: https://kubernetes.io/docs/concepts/services-networking/ingress/#ingress-controllers

We have a couple of choices (and non-choices, see appendix) for the current
resource:

1.  We can delete the current resource from extensions.v1beta1 in anticipation
    that an improved API can replace it.

1.  We can copy the API as-is (or with minor changes) into networking.v1beta1,
    preserving/converting existing data (following the same approach taken with
    all other extensions.v1beta1 resources). This will allow us to start the
    cleanup of the extensions API group. This also prepares the API for GA.

Option 1 does not seem realistic in a short-term time frame (a new API will need
to be taken through design, alpha/beta/ga phases). At the same time, there are
enough users that the existing API cannot be deleted out right.

In terms of moving the API towards GA, the API itself has been available in beta
for so long that it has attained defacto GA status through usage and adoption
(both by users and by load balancer / ingress controller providers). Abandoning
it without a full replacement is not a viable approach.  It is clearly a useful
API and captures a non-trivial set of use cases.  At this point, it seems more
prudent to declare the current API as something the community will support as a
V1, codifying its status, while working on either a V2 Ingress API or an
entirely different API with a superset of features.

### Goals

* Move Ingress to a permanent API group.
* Position the Ingress API in a place towards making progress towards GA.

## Proposal

### 1.14

* Copy the Ingress API to `networking.k8s.io/v1beta1` (preserving existing data
  and round-tripping with the extensions Ingress API, following the approach
  taken for all other `extensions/v1beta1` resources)
* Develop a set of planned changes and GA graduation criteria with sig-network
  (intent is to target a minimal set of bugfixes and non-breaking changes)
* Announce `extensions/v1beta1` Ingress as deprecated (and announce plan for GA)

### 1.15

* Update API server to persist in networking.k8s.io/v1beta1
* Update in-tree controllers, examples, and clients to target
  `networking.k8s.io/v1beta1`
* Update Ingress controllers in the kubernetes org to target
  `networking.k8s.io/v1beta1`
* Update documentation to recommend new users start with
  networking.k8s.io/v1beta1, but existing users stick with `extensions/v1beta1`
  until `networking.k8s.io/v1` is available.

* Update documentation to reference `networking.k8s.io/v1beta1`
* Meet graduation criteria and promote API to `networking.k8s.io/v1`
* Announce `newtorking.k8s.io/v1beta1` Ingress as deprecated

### 1.16

* Update API server to persist in `networking.k8s.io/v1`.
* Update in-tree controllers, examples, and clients to target
  `networking.k8s.io/v1`.
* Update Ingress controllers in the kubernetes org to target
  `networking.k8s.io/v1`.
* Update documentation to reference `networking.k8s.io/v1`.
* Evangelize availability of v1 Ingress API to out-of-org Ingress controllers

### 1.18

* Remove ability to serve `extensions/v1beta1` and `networking.k8s.io/v1beta1`
  Ingress resources (preserve ability to read existing `extensions/v1beta1`
  Ingress objects from storage and serve them via the `networking.k8s.io/v1`
  API)

## Graduation Criteria

`networking.k8s.io/v1beta1`

* 1.14: Ingress API exists and has parity with existing `extensions/v1beta1` API
* 1.14: `extensions/v1beta1` Ingress tests are replicated against
  `networking.k8s.io`
* 1.15: all in-tree use and in-org controllers switch to `networking.k8s.io` API
  group
* 1.15: documentation and examples are updated to refer to networking.k8s.io
  API group `networking.k8s.io/v1`

* TBD based on plans developed by sig-network in 1.14 timeframe

## Implementation History

TBD

## Alternatives

See motivation section.

## Appendix

### Non-options

One suggestion was to move the API into a new API group, defined as a CRD.  This
does not work because there is no way to do round-trip of existing Ingress
objects to a CRD-based API.


### Potential pre-GA work

Note: these items are NOT the main focus of this KEP, but recorded here for
reference purposes. These items came up in discussions on the KEP (roughly
sorted by practicality):

* Spec path as a prefix, maybe as a new field
* Rename `backend` to `defaultBackend` or something more obvious
* Be more explicit about wildcard hostname support (I can create *.bar.com but
  in theory this is not supported)
* Add health-checks API
* Specify whether to accept just HTTPS or also allow bare HTTP
* Better status
* Formalize Ingress class
* Reference a secret in a different namespace?  Use case: avoid copying wildcard
  certificates (generated with cert-manager for instance)
* Add non-required features (levels of support)
* Some way to have backends be things other than a service (e.g. a GCS bucket)
* Some way to restrict hostnames and/or URLs per namespace
* HTTP to HTTPS redirects
* Explicit sharing or non-sharing of external IPs (e.g. GCP HTTP LB)
* Affinity
* Per-backend timeouts
* Backend protocol
* Cross-namespace backends
