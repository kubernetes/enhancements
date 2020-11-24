# KEP-1659: Standard Topology Labels

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Reserve a label prefix](#reserve-a-label-prefix)
  - [Defining the meaning of existing labels](#defining-the-meaning-of-existing-labels)
  - [Redefining kubernetes.io/hostname](#redefining-kubernetesiohostname)
  - [Defining a third key (or not)](#defining-a-third-key-or-not)
  - [Followup work (or optionally part of this)](#followup-work-or-optionally-part-of-this)
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
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

- [x] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] KEP approvers have approved the KEP status as `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kubernetes has always taken the position that "topology is arbitrary", and
designs dealing with topology have had to take that into account.  Even so, the
project has two commonly assumed labels - `topology.kubernetes.io/region` and
`topology.kubernetes.io/zone` - which are used in many components, generally
hard-coded and not extensible.  Those labels have relatively well understood
meanings, and (so far) have been sufficient to represent what most people need.

This KEP proposes to declare those labels, and possibly one more, as "standard"
and give them more well-defined meanings and semantics.  APIs that handle
topology can still handle arbitrary topology keys, but these common ones may be
handled automatically.

## Motivation

As we consider problems like cross-zone network traffic being a chargeable
resource in most public clouds, we started to build an API for topology in
Services.  We tried to think through how that API would map to existing
load-balancer implementations which may already understand topology, and we
realized 3 things.

  1) Cloud-ish load-balancers do not have arbitrary topology APIs and can't
     easily adapt to that.
  2) Other systems have standardized on two or three levels of topology (e.g. the [Envoy locality API]).
  3) Nobody is really complaining about this.

In trying to simplify the way Service topology might work, we are proposing
that standardizing on a small set of well-defined topology concepts will be a
net win for the project at almost no cost to what users are actually doing with
Kubernetes.

[Envoy locality API]: https://www.envoyproxy.io/docs/envoy/latest/api-v3/config/core/v3/base.proto#envoy-v3-api-msg-config-core-v3-locality

### Goals

The goals of this KEP are to:
   * build consensus that the two topology lables that ALREADY EXIST in Kubernetes are enough for most users
   * determine whether a third level of topology is required or not
   * produce short, descriptive, canonical documentation for theses labels

### Non-Goals

This KEP does NOT seek to:
   * add new functionality that uses topology
   * change existing functionality that uses topology
   * solve the service topology problem

## Proposal

Kubernetes has always taken the position that "topology is arbitrary", and
designs dealing with topology have had to take that into account.  Even so, the
project has two commonly assumed labels - `topology.kubernetes.io/region` and
`topology.kubernetes.io/zone` - which are used in many components, generally
hard-coded and not extensible.  Those labels have relatively well understood
meanings, and (so far) have been sufficient to represent what most people need.

This KEP proposes to document those labels as "standard" and give them more
rigorous definitions.  This also proposes that we discuss and decide whether a
third level of topology is needed and if so, define it in the same manner as
the existing labels.

The resulting definitions should be specific enough that users and implementors
understand what they mean, but not so rigid that they can not map them to the
nearest constructs available in most environments.

### Risks and Mitigations

The primary risks here are:

1) That we define these too loosely, such that users can not derive sufficient
value from their use.

2) That we define these too specifically, such that implementors can not use
them to represent natural concepts in their environments.

3) That we define these in a way that is incompatible with the ways they are
alredy being used.

4) That we preclude or design-out other uses of topology that users are using
today.

## Design Details

### Reserve a label prefix

Label prefixes allow us to group labels on common origin and meaning.  We
propose to document somewhere (TBD) that the prefix "topology.kuberntes.io" is
explicitly reserved for use in defining metadata about the physical or logical
connectivity and grouping of Kubernetes nodes (and other things), and the
associated behavioral and failure properties of those groups.

This prefix is already in use.  This KEP just aims to formalize it.

### Defining the meaning of existing labels

This KEP proposes to define the meaning and semantics of the following labels:

    * topology.kubernetes.io/region
    * topology.kubernetes.io/zone

The exact wording is TBD, but it must be specific enough to be useful to users
and loose enough to allow implementors sufficient freedom.

This will also include defining that "region" and "zone" are strictly
hierarchical ("zones" are subsets of "regions") and that zone names are unique
across regions.  For example AWS documents "us-east-1a" as a zone under region
"us-east-1".

This will also define that, while labels are generally mutable, the topology
labels should be assumed immutable and that any changes to them may be ignored
by downstream consumers of topology.

### Redefining kubernetes.io/hostname

The widely-known label "kubernetes.io/hostname" might be better as
"topology.kubernetes.io/node", but that change is considered out of scope for
this KEP.  We may or may not choose to tackle it at a later time.

### Defining a third key (or not)

Some systems define topology in two levels (e.g. public clouds) and others use three
levels (e.g. Envoy adds "sub-zone").  This KEP proposes that we standardize on
two levels for now, while reserving the right to expand that to three (or more)
if and when we have strong demand.

### Followup work (or optionally part of this)

For a Pod to know its own topology today, it must be authorized to look at
Nodes.  This is somewhat tedious, when we have downward-API support for labels
already, and we know that these topology labels are not likely to change at
run-time.

If we standardize topology keys, it would be reasonable to copy those
well-known keys from the Node to the Pod at startup, so Pods could extract that
information without bouncing through a Node object.

As long as "topology is arbitrary", we need more information about which keys to
copy, which makes this feature request less feasible.

### Test Plan

NOT APPLICABLE.

This KEP does not plan to change code, just documentation.

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning),
or by redefining what graduation means.

In general, we try to use the same stages (alpha, beta, GA), regardless how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least 2 releases between beta and
GA/stable, since there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Removing a deprecated flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality which deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include [conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
-->

### Upgrade / Downgrade Strategy

NOT APPLICABLE.

This KEP does not plan to change code, just documentation.

### Version Skew Strategy

NOT APPLICABLE.

This KEP does not plan to change code, just documentation.

## Production Readiness Review Questionnaire

### Feature enablement and rollback

NOT APPLICABLE.

This KEP does not plan to change code, just documentation.

### Rollout, Upgrade and Rollback Planning

NOT APPLICABLE.

This KEP does not plan to change code, just documentation.

### Monitoring requirements

NOT APPLICABLE.

This KEP does not plan to change code, just documentation.

### Dependencies

NOT APPLICABLE.

This KEP does not plan to change code, just documentation.

### Scalability

NOT APPLICABLE.

This KEP does not plan to change code, just documentation.

### Troubleshooting

NOT APPLICABLE.

This KEP does not plan to change code, just documentation.
## Implementation History

* 2020-03-31: First draft

## Drawbacks

Topology being arbitrary has a certain abstract elegance to it, and it forces
consumers of topology to be flexible in their designs.  Moving away from that
brings risks of over-specifying and missing the mark for some users.

## Alternatives

The main alternative is status quo - topology is arbitrary.  The main drivers
for abandoning this are described above under "Motivation".
