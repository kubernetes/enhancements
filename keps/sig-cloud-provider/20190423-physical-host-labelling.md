---
title: Physical Host Labelling
authors:
  - '@misterikkit'
owning-sig: sig-cloud-provider
participating-sigs:
  - sig-scheduling
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-04-23
last-updated: 2019-04-29
status: implementable
see-also:
  - /keps/sig-scheduling/20190221-even-pods-spreading.md
  - 'https://github.com/kubernetes/kubernetes/issues/64021'
replaces: []
superseded-by: []
---
# Physical Host Labelling

## Table of Contents

<!-- toc -->

- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  * [Goals](#goals)
  * [Non-Goals](#non-goals)
- [Proposal](#proposal)
  * [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
  * [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  * [Test Plan](#test-plan)
  * [Graduation Criteria](#graduation-criteria)
    + [Examples](#examples)
      - [Alpha -> Beta Graduation](#alpha---beta-graduation)
      - [Beta -> GA Graduation](#beta---ga-graduation)
      - [Removing a deprecated flag](#removing-a-deprecated-flag)
  * [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  * [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed [optional]](#infrastructure-needed-optional)

<!-- tocstop -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

Many kubernetes clusters are deployed on virtual machines (also referred to as
"instances"), and information about the physical host for those instances (also
referred to as "instance host") is not consistently available. Instances which
share a host can fail simultaneously if there is a problem with the host. We
propose to label nodes with their instance host ID so that controllers (namely,
the scheduler) can decrease shared fate of pods from the same collection.

## Motivation

This KEP is motivated by the desire to improve reliability of workloads running
in kubernetes clusters. Today, the scheduler will attempt to spread pods from a
given collection (e.g. a deployment) across nodes. Even if every pod ends up on
a different node, there may be multiple pods per instance host. This means that
the deployment is not as reliable as it appears to be.

By adding a physical host label, we can treat physical hosts as a failure
domain, and perform spreading appropriately.

### Goals

- Introduce a well known label for physical host (e.g.
  `topology.kubernetes.io/physical-host`)
- Add a cloud provider API to get physical host ID
- Update cloud-controller-manager to update node labels based on the cloud
  provider value.

### Non-Goals

- Scheduling behavior that considers physical host
- Controller behavior (e.g. deployment, statefulset) that considers physical
  host
- Any decision-making that is based on the physical host label

## Proposal

We will add a new field to the [Zone struct][zone-struct], `PhysicalMachineID`.
If supported, it will be populated by cloud providers whenever returning a zone.

[zone-struct]: https://github.com/kubernetes/cloud-provider/blob/0a4f4cbb5a664deb4639d7d9bf5bbde3bb3603c1/cloud.go#L208-L211

When the `PhysicalMachineID` is supplied, cloud-controller-manager will use it
as the value for the `topology.kubernetes.io/physical-host` label. (Depending on
when this is implemented, this may instead be
`failure-domain.beta.kubernetes.io/physical-host`. See
https://github.com/kubernetes/enhancements/pull/839)

### Implementation Details/Notes/Constraints [optional]

What are the caveats to the implementation?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they releate.

### Risks and Mitigations

What are the risks of this proposal and how do we mitigate.
Think broadly.
For example, consider both security and how this will impact the larger kubernetes ecosystem.

How will security be reviewed and by whom?
How will UX be reviewed and by whom?

Consider including folks that also work outside the SIG or subproject.

## Design Details

### Test Plan

**Note:** *Section not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy.
Anything that would count as tricky in the implementation and anything particularly challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage expectations).
Please adhere to the [Kubernetes testing guidelines][testing-guidelines] when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md

### Graduation Criteria

**Note:** *Section not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. Initial KEP should keep
this high-level with a focus on what signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning),
or by redefining what graduation means.

In general, we try to use the same stages (alpha, beta, GA), regardless how the functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

#### Examples

These are generalized examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

##### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

##### Beta -> GA Graduation

- N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least 2 releases between beta and GA/stable, since there's no opportunity for user feedback, or even bug reports, in back-to-back releases.

##### Removing a deprecated flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality which deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include [conformance tests].**

[conformance tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/conformance-tests.md

### Upgrade / Downgrade Strategy

If applicable, how will the component be upgraded and downgraded? Make sure this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to make use of the enhancement?

### Version Skew Strategy

If applicable, how will the component handle version skew with other components? What are the guarantees? Make sure
this is in the test plan.

Consider the following in developing a version skew strategy for this enhancement:
- Does this enhancement involve coordinating behavior in the control plane and in the kubelet? How does an n-2 kubelet without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI, CRI or CNI may require updating that component before the kubelet.

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded

## Drawbacks

This adds a specific API for a topology type. There already is some demand for
other topology types, and we could end up implementing specific APIs for each
one instead of developing a general topology/node labelling API.

## Alternatives

Instead of an API specific to physical host ID, we could add a general API
allowing cloud providers to add arbitrary topology labels to instances. One
drawback to this alternative is that users could not count on well-defined
topology labels when moving between cloud providers.

There has been some [discussion][1] about defining topologies in CRDs on the
sig-cloud-provider mailing list.

[1]: https://groups.google.com/forum/#!topic/kubernetes-sig-cloud-provider/32N59IYXogY

## Infrastructure Needed [optional]

Use this section if you need things from the project/SIG.
Examples include a new subproject, repos requested, github details.
Listing these here allows a SIG to get the process for these resources started right away.



