---
title: Service Fallback Selector
authors:
  - "@rabbitfang"
owning-sig: sig-network
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2020-03-13
last-updated: 2020-03-13
status: provisional
---

# Service Fallback Selector

This is the title of the KEP.
Keep it simple and descriptive.
A good title can help communicate what the KEP is and should be considered as part of any review.

The title should be lowercased and spaces/punctuation should be replaced with `-`.

To get started with this template:
1. **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking up.
  KEPs should not be checked in without a sponsoring SIG.
1. **Make a copy of this template.**
  Copy this template into the owning SIG's directory (or KEP root directory, as appropriate) and name it `YYYYMMDD-my-title.md`, where `YYYYMMDD` is the date the KEP was first drafted.
1. **Fill out the "overview" sections.**
  This includes the Summary and Motivation sections.
  These should be easy if you've preflighted the idea of the KEP with the appropriate SIG.
1. **Create a PR.**
  Assign it to folks in the SIG that are sponsoring this process.
1. **Create an issue in kubernetes/enhancements, if the enhancement will be targeting changes to kubernetes/kubernetes**
  When filing an enhancement tracking issue, please ensure to complete all fields in the template.
1. **Merge early.**
  Avoid getting hung up on specific details and instead aim to get the goal of the KEP merged quickly.
  The best way to do this is to just start with the "Overview" sections and fill out details incrementally in follow on PRs.
  View anything marked as a `provisional` as a working document and subject to change.
  Aim for single topic PRs to keep discussions focused.
  If you disagree with what is already in a document, open a new PR with suggested changes.

The canonical place for the latest set of instructions (and the likely source of this file) is [here](/keps/YYYYMMDD-kep-template.md).

The `Metadata` section above is intended to support the creation of tooling around the KEP process.
This will be a YAML section that is fenced as a code block.
See the KEP process for details on each of these items.

## Table of Contents

A table of contents is helpful for quickly jumping to sections of a KEP and for highlighting any additional information provided beyond the standard KEP template.

Ensure the TOC is wrapped with <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code> tags, and then generate with `hack/update-toc.sh`.

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [API Changes](#api-changes)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
    - [Story 4](#story-4)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Examples](#examples)
      - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
      - [Beta -&gt; GA Graduation](#beta---ga-graduation)
      - [Removing a deprecated flag](#removing-a-deprecated-flag)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

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

This KEP proposes an extension on the existing Service API to specify a fallback mechanism for the existing selector field,
so when the Service's selector doesn't match any ready pods, an alternate set of pods can be selected instead.

## Motivation

The current behavior implemented by `kube-proxy` when a `ClusterIP` Service has no ready endpoints is to reject all connections.
This results in a poor user experience, especially for services pointing at an HTTP application where the user's browser displays an unhelpful "connection refused" error message or other generic "service unavailable" page. This proposal enables cluster users to customize the behavior that occurs when a Service has no ready pods.

### Goals

- Define a flexible mechanism by which Kubernetes users can adjust the behavior for when no ready pods are matched by a Service's selector.
- The mechanism should have the same effect as adjusting the service's `spec.selector` field, even if no such change is actually made.

### Non-Goals

- Extend the capabilities of Service beyond Layer 4 (e.g. making `kube-proxy` HTTP-aware).
- Provide cross-namespace routing capabilities (e.g. for a single "server down" page shared by multiple namespaces)
- Change the behavior/API of topology-aware service routing or EndpointSlice.
- Support partial fallback, or balancing between two sets of selectors.

## Proposal

### API Changes

*TODO*

### User Stories

Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of the system.
The goal here is to make this feel real for users without getting bogged down.

#### Story 1

An application developer wants to set up a branded "service unavailable" page as an alternative to the generic "connection refused" page displayed by their browser or CDN provider if (for whatever reason) all instances of their application fail.
They configure a highly-available (multiple replicas, PodDisruptionBudget, etc.) deployment to serve a single static page that they are able to customize as desired (e.g. with company branding, links to status pages, etc).
They add a fallback selector to the service to point at the error-page pods.
If an outage occurs in their application, end users will see this branded error page instead of the harsh generic page they would otherwise see.

#### Story 2

A company has a large development environment where running applications consumes a lot of CPU and memory resources.
To save money, the company would like to be able to scale down this environment outside of normal business hours when usage is generally zero.
They implement a mechanism to scale down deployments outside of business hours,
  but they do not like the unhelpful "connection refused" response that occurs when they visit these environments when they are scaled down,
  since they do occasionally use these environments outside of business hours.
They implement a simple application that provides a simple "turn on" button that, when clicked, scales the relevant deployment back up.
Using a fallback selector on the service, they can have this application be served automatically when the primary application is offline.

#### Story 3

An application has in front of it a caching layer to provide cached responses to clients much faster than the application would be able to,
  even though the application is still capable of handling the full load of traffic.
That is, the caching layer is there to provide a better user experience, not to reduce the load on the application.
Developers configure the application's service to point at the caching layer,
  but if the caching layer ever failed,
  to fallback to sending traffic directly to the main application.

#### Story 4

An application developer wants to create a canary deployment of an application.
They create a new Service that points only at the canary pods.
Because they would like the canary Service to still function even if no canary is active,
  they configure the canary Service to fall back onto the primary application pods when no canary pods are available.

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

[conformance tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md

### Upgrade / Downgrade Strategy

If applicable, how will the component be upgraded and downgraded? Make sure this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to make use of the enhancement?

### Version Skew Strategy

This feature does not require any changes in any components other than the API server and controller manager.

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

- If used improperly, a service can rapidly (every few seconds) switch between the primary and fallback selectors if the pods matched by the primary selector flap between ready and not ready.
  If either the primary or the fallback selector match many pods,
  the size of the changes to the Endpoint and/or EndpointSlice objects would be large.
  This would result in higher CPU and network loads to both the API server, the controller manager, and kube-proxy.

## Alternatives

- Instead of extending the Service API, this feature could be achieved using an annotation on the Service object or with a CRD and a controller that updates the Service's selector field.
