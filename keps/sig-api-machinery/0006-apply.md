---
kep-number: 6
title: Apply
authors:
  - "@lavalamp"
owning-sig: sig-api-machinery
participating-sigs:
  - sig-api-machinery
  - sig-cli
reviewers:
  - "@pwittrock"
  - "@erictune"
approvers:
  - "@bgrant0607"
editor: TBD
creation-date: 2018-03-28
last-updated: 2018-03-28
status: implementable
see-also:
  - n/a
replaces:
  - n/a
superseded-by:
  - n/a
---

# Apply

## Table of Contents

- [Apply](#apply)
   - [Table of Contents](#table-of-contents)
   - [Summary](#summary)
   - [Motivation](#motivation)
      - [Goals](#goals)
      - [Non-Goals](#non-goals)
   - [Proposal](#proposal)
      - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
      - [Risks and Mitigations](#risks-and-mitigations)
   - [Graduation Criteria](#graduation-criteria)
   - [Implementation History](#implementation-history)
   - [Drawbacks](#drawbacks)
   - [Alternatives](#alternatives)

## Summary

`kubectl apply` is a core part of the Kubernetes config workflow, but it is
buggy and hard to fix. This functionality will be regularized and moved to the
control plane.

## Motivation

Example problems today:

* User does POST, then changes something and applies: surprise!
* User does an apply, then `kubectl edit`, then applies again: surprise!
* User does GET, edits locally, then apply: surprise!
* User tweaks some annotations, then applies: surprise!
* Alice applies something, then Bob applies something: surprise!

Why can't a smaller change fix the problems? Why hasn't it already been fixed?

* Too many components need to change to deliver a fix
* Organic evolution and lack of systematic approach
  * It is hard to make fixes that cohere instead of interfere without a clear model of the feature
* Lack of API support meant client-side implementation
  * The client sends a PATCH to the server, which necessitated strategic merge patch--as no patch format conveniently captures the data type that is actually needed.
  * Tactical errors: SMP was not easy to version, fixing anything required client and server changes and a 2 release deprecation period.
* The implications of our schema were not understood, leading to bugs.
  * e.g., non-positional lists, sets, undiscriminated unions, implicit context
  * Complex and confusing defaulting behavior (e.g., Always pull policy from :latest)
  * Non-declarative-friendly API behavior (e.g., selector updates)

### Goals

"Apply" is intended to allow users and systems to cooperatively determine the
desired state of an object. The resulting system should:

* Be robust to changes made by other users, systems, defaulters (including mutating admission control webhooks), and object schema evolution.
* Be agnostic about prior steps in a CI/CD system (and not require such a system).
* Have low cognitive burden:
  * For integrators: a single API concept supports all object types; integrators
    have to learn one thing total, not one thing per operation per api object.
    Client side logic should be kept to a minimum; CURL should be sufficient to
    use the apply feature.
  * For users: looking at a config change, it should be intuitive what the
    system will do. The “magic” is easy to understand and invoke.
  * Error messages should--to the extent possible--tell users why they had a
    conflict, not just what the conflict was.
  * Error messages should be delivered at the earliest possible point of
    intervention.

Goal: The control plane delivers a comprehensive solution.

Goal: Apply can be called by non-go languages and non-kubectl clients. (e.g.,
via CURL.)

### Non-Goals

* Multi-object apply will not be changed: it remains client side for now
* Some sources of user confusion will not be addressed:
  * Changing the name field makes a new object rather than renaming an existing object
  * Changing fields that can’t really be changed (e.g., Service type).

## Proposal

(Please note that when this KEP was started, the KEP process was much less well
defined and we have been treating this as a requirements / mission statement
document; KEPs have evolved into more than that.)

A brief list of the changes:

* Apply will be moved to the control plane.
  * The [original design](https://goo.gl/UbCRuf) is in a google doc; joining the
    kubernetes-dev or kubernetes-announce list will grant permission to see it.
    Since then, the implementation has changed so this may be useful for
    historical understanding. The test cases and examples there are still valid.
  * Additionally, readable in the same way, is the [original design for structured diff and merge](https://goo.gl/nRZVWL);
    we found in practice a better mechanism for our needs (tracking field
    managers) but the formalization of our schema from that document is still
    correct.
* Apply is invoked by sending a certain Content-Type with the verb PATCH.
* Instead of using a last-applied annotation, the control plane will track a
  "manager" for every field.
* Apply is for users and/or ci/cd systems. We modify the POST, PUT (and
  non-apply PATCH) verbs so that when controllers or other systems make changes
  to an object, they are made "managers" of the fields they change.
* The things our "Go IDL" describes are formalized: [structured merge and diff](https://github.com/kubernetes-sigs/structured-merge-diff)
* Existing Go IDL files will be fixed (e.g., by [fixing the directives](https://github.com/kubernetes/kubernetes/pull/70100/files))
* Dry-run will be implemented on control plane verbs (POST, PUT, PATCH).
  * Admission webhooks will have their API appended accordingly.
* An upgrade path will be implemented so that version skew between kubectl and
  the control plane will not have disastrous results.

The linked documents should be read for a more complete picture.

### Implementation Details/Notes/Constraints [optional]

(TODO: update this section with current design)

What are the caveats to the implementation?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they releate.

### Risks and Mitigations

We used a feature branch to ensure that no partial state of this feature would
be in master. We developed the new "business logic" in a
[separate repo](https://github.com/kubernetes-sigs/structured-merge-diff) for
velocity and reusability.

### Testing Plan

The specific logic of apply will be tested by extensive unit tests in the 
[structured merge and diff](https://github.com/kubernetes-sigs/structured-merge-diff)
repo. The integration between that repo and kubernetes/kubernetes will mainly
be tested by integration tests in [test/integration/apiserver/apply](https://github.com/kubernetes/kubernetes/tree/master/test/integration/apiserver/apply)
and [test/cmd](https://github.com/kubernetes/kubernetes/blob/master/test/cmd/apply.sh),
as well as unit tests where applicable. The feature will also be enabled in the
[alpha-features e2e test suite](https://k8s-testgrid.appspot.com/sig-release-master-blocking#gce-cos-master-alpha-features),
which runs every hour and everytime someone types `/test pull-kubernetes-e2e-gce-alpha-features`
on a PR. This will ensure that the cluster can still start up and the other
endpoints will function normally when the feature is enabled.

## Graduation Criteria

An alpha version of this is targeted for 1.14.

This can be promoted to beta when it is a drop-in replacement for the existing
kubectl apply, and has no regressions (which aren't bug fixes). This KEP will be
updated when we know the concrete things changing for beta.

This will be promoted to GA once it's gone a sufficient amount of time as beta
with no changes. A KEP update will precede this.

## Implementation History

* Early 2018: @lavalamp begins thinking about apply and writing design docs
* 2018Q3: Design shift from merge + diff to tracking field managers.
* 2019Q1: Alpha.

(For more details, one can view the apply-wg recordings, or join the mailing list
and view the meeting notes. TODO: links)

## Drawbacks

Why should this KEP _not_ be implemented: many bugs in kubectl apply will go
away. Users might be depending on the bugs.

## Alternatives

It's our belief that all routes to fixing the user pain involve
centralizing this functionality in the control plane.
