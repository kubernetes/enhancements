# KEP-2527: Clarify if/how controllers can use status to track non-observable state

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Part 1: Loosen and clarify <code>status</code>](#part-1-loosen-and-clarify-status)
    - [Examples](#examples)
  - [Part 2: Clarify when it makes sense to use 2 objects](#part-2-clarify-when-it-makes-sense-to-use-2-objects)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Alternative 1: Add a new top-level stanza to spec/status resources](#alternative-1-add-a-new-top-level-stanza-to-specstatus-resources)
    - [Examples](#examples-1)
    - [Tradeoffs](#tradeoffs)
  - [Alternative 2: Sub-divide spec](#alternative-2-sub-divide-spec)
    - [Examples](#examples-2)
    - [Tradeoffs](#tradeoffs-1)
    - [Notes](#notes)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
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

Since basically the beginning of Kubernetes, we've had this sort of "litmus
test" for status fields: "If I erased the whole status struct, would everything
in it be reconstructed (or at least be reconstructible) from observation?". The
goal was to ensure that the delineation between "what I asked for" and "what it
actually is" was clear and to encourage active reconciliation of state.

Another reason for this split was an idea which, as far as I know, has never
been implemented by anyone: that an object's spec and status blocks could be
stored in different etcd instances and the status could have a TTL.  At this
point in the project, I expect that wiping status out like this would end in
fireworks, and not the fun kind.  Status is, effectively, as durable as spec.

Many of our APIs pass this test (sometimes we fudge it and say yes "in
theory"), but not all of them do.  This KEP proposes to clarify or remove this
guidance, especially as it pertains to state that is not derived from
observation.

One of the emergent uses of the spec/status split is access control.  It is
assumed that, for most resources, users own (can write to) all of spec and
controllers own all of status, and not the reverse.  This allows patterns like
Services which set `spec.type: LoadBalancer`, where the controller writes the
LB's IP address to status, and kube-proxy can trust that IP address (because it
came from a controller, not a user).  Compare that with Services which use
`spec.externalIPs`.  The behavior in kube-proxy is roughly the same, but
because non-trusted users can write to `spec.externalIPs` and that does not
require a trusted controller to ACK, that behavior was declared a CVE.

This KEP further proposes to add guidance for APIs that want to implement an
"allocation" or "binding" pattern which requires trusted ACK.

## Motivation

As an API reviewer, I have seen many different patterns in use.  I have shot
down APIs that run afoul of the rebuild-from-observation test, and forced the
implementations to be more complicated.  I no longer find this to be useful to
our APIs, and in fact I find it to be a detriment.

I suspect that several APIs would have come out differently if not for this.

### Goals

* Clarify or remove the from-observation rule to better match reality.
* Provide guidance for APIs on how to save controller results.

### Non-Goals

* Retroactively apply this pattern to change existing GA APIs.
* Necessarily apply this pattern to change pre-GA APIs (though they may choose
  to follow it).
* Provide arbitrary storage for controller state.
* Provide distinct access control for subsets of spec or status.

## Proposal

This KEP proposes to:

1) Get rid of the [if it got lost](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#spec-and-status)
litmus test for status fields
2) Acknowledge that spec/status are a useful and meaningful split for access control
3) Document one or more patterns for APIs that need a trusted acknowledgement
   of the spec

Depending on feedback, there may be other approaches to solving the same
problems.

### Risks and Mitigations

The current guidance exists for a reason.  It does encourage a sort of
ideological cleanliness.  However, it is not universally adhered to and it
neglects the reality of these request/acknowledge APIs.  On the whole, this KEP
posits that the current guidance is a net negative.

## Design Details

### Part 1: Loosen and clarify `status`

Remove the idea that status fields _must_ be from observation.  Allow controllers
to write values to status that represent allocations or acknowledged requests.
Document that status fields are best when they represent observed state, but do
not _require_ it.

This has [already happened](https://github.com/kubernetes/enhancements/pull/2308/files#r567809465)
at least once.

Additionally, discuss the risks of using `status` for non-observation data -
specifically how to think about write sequencing with regards to observers and
how to think about idempotency.  These problems are not new to this revised
definition of status, but "status is from observation" tends to be more
one-way, and relaxing that will likely lead to more variety of controller
patterns.

#### Examples

Some of these are variants of existing APIs and some are hypothetical.

1) User creates a Service and sets `Service.spec.type = LoadBalancer`.  The
cloud controller sets `Service.status.loadBalancer.ingress.ip` in response.  If
the user sets `Service.spec.loadBalancerIP` to a specific value, the cloud
controller will either successfully use that IP and and set it as before (ACK),
or it will not set any value in `Service.status.loadBalancer.ingress` (NAK).

2) Given a Pod, user patches `Pod.spec.containers[0].resources.requests[cpu]`
to a new value.  Kubelet sees this as a request and, if viable, sets
`Pod.status.containerStatuses[0].resources.requests[cpu]` to the same value.

3) User creates a `Bucket`, setting `Bucket.spec.instance = artifacts-prod`.
The bucket controller verifies that the namespace is allowed to use the
artifacts-prod bucket and, if so, sets `Bucket.status.instance` to the same
value.

### Part 2: Clarify when it makes sense to use 2 objects

Offer API developers guidance on when it makes sense to use 2 objects instead
of status.  The exact wording is TBD, but it has to consider authorization
granularity vs. many small objects, cardinality, cohesion, and extensibility.

### Test Plan

N/A (docs)

### Graduation Criteria

N/A (docs)

### Upgrade / Downgrade Strategy

N/A (docs)

### Version Skew Strategy

N/A (docs)

## Production Readiness Review Questionnaire

N/A (docs)

## Implementation History

Feb 22, 2021: First draft
Jun 16, 2021: Incorporate feedback, pick a preferred model

## Drawbacks

There is some value in keeping `status` purely from-observation.  Unfortunately
that ideal does not seem to be surviving contact with the real world.

Using `status` as an RBAC scope is coarse, and this makes that worse by
allowing more uses of `status` (and thereby more controllers RBAC'ed to write
to it).

## Alternatives

### Alternative 1: Add a new top-level stanza to spec/status resources

Keep and strengthen the idea that status fields must be from observation.
Segregate controller-owned fields to a new stanza, parallel to `spec` and
`status`, which can be RBAC'ed explicitly.  For the sake of this doc, let's
call it `control`.

To make this viable, we would need to add `control` as a "standard" subresource
and apply similar rules to `spec` and `status` with regards to writes (can't
write to `control` through the main resource, can't write to `spec` or `status`
through the `control` subresource).  We would also need to add it to CRDs as an
available subresource.

#### Examples

1) User creates a Service and sets `Service.spec.type = LoadBalancer`.  The
cloud controller sets `Service.control.loadBalancer.ingress.ip` in response.  If
the user sets `Service.spec.loadBalancerIP` to a specific value, the cloud
controller will either successfully use that IP and and set it as before (ACK),
or it will not set any value in `Service.control.loadBalancer.ingress` (NAK).

2) Given a Pod, user patches `Pod.spec.containers[0].resources.requests[cpu]`
to a new value.  Kubelet sees this as a request and, if viable, sets
`Pod.control.containerStatuses[0].resources.requests[cpu]` to the same value.

3) User creates a `Bucket`, setting `Bucket.spec.instance = artifacts-prod`.
The bucket controller verifies that the namespace is allowed to use the
artifacts-prod bucket and, if so, sets `Bucket.control.instance` to the same
value.

#### Tradeoffs

Pro: Clarifies the meaning of status.

Pro: Possibly clarifies the roles acting on a resource.

Con: Requires a lot of implementation and possibly steps on existing uses of the
field name.

Con: Net-new concept requires new documentation and socialization.

Con: Incompatible with existing uses of status for this.

### Alternative 2: Sub-divide spec

Keep and strengthen the idea that status fields must be from observation.
Segregate controller-owned fields to a sub-stanza of spec.  Create a new
access-control mechanism (or extend RBAC) to provide field-by-field access.

#### Examples

1) User creates a Service and sets `Service.spec.type = LoadBalancer`.  The
cloud controller sets `Service.spec.control.loadBalancer.ingress.ip` in
response.  If the user sets `Service.spec.loadBalancerIP` to a specific value,
the cloud controller will either successfully use that IP and and set it as
before (ACK), or it will not set any value in
`Service.spec.control.loadBalancer.ingress` (NAK).

2) Given a Pod, user patches `Pod.spec.containers[0].resources.requests[cpu]`
to a new value.  Kubelet sees this as a request and, if viable, sets
`Pod.spec.control.containerStatuses[0].resources.requests[cpu]` to the same
value.

3) User creates a `Bucket`, setting `Bucket.spec.instance = artifacts-prod`.
The bucket controller verifies that the namespace is allowed to use the
artifacts-prod bucket and, if so, sets `Bucket.spec.control.instance` to the same
value.

#### Tradeoffs

Pro: Retains purity of status.

Con: Confuses the meaning of spec.

Con: Can collide with existing uses of the field name.

Con: Needs a whole new access model.

Con: Likely needs a new subresource.

#### Notes

This model is included for completeness.  I do not expect ANYONE to endorse it.
