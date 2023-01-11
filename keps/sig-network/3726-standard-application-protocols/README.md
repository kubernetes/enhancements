\<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
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
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
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

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

There are cases where implementations implement different things for the same application protocol names. That already causes(and can cause more in the future) issues for certain implementations to interoperate. (see example with GKE and Istio under [Motivation](#motivation))

This KEP proposes to declare standard Kubernetes protocol names for common protocols that are not IANA service names.


## Motivation

The lack of direct support for specifying application protocols for ports and the widespread use of implementation-specific annotations to mitigate it had led k8s to add the `AppProtocol` field to the port spec.

While this is a good solution - we never came with recommended standards other than [IANA standard service names](https://www.iana.org/assignments/service-names) or a domain prefixed protocol.

This loose definition has led us to have instances where implementations do different things for common protocols.

One good example is `HTTP2`. 
* In GKE you can use `appProtocol: HTTP2` and it will describe HTTP2 over TLS (https://cloud.google.com/kubernetes-engine/docs/how-to/secure-gateway#load-balancer-tls).
* While in Istio it will be h2c (HTTP2 over cleartext).

That creates a problem where users with GKE and Istio in their cluster can have very different behaviors for the same `appProtocol` value.



### Goals

* Build consensus around how common (non IANA service names) should be implemented
* Help the broader community and specifically implementations to interoperate better
* Provide short and clear documentation for how appProtocol values should be implemented
  * Update the appProtocol user documentation respectively


### Non-Goals

* Validate appProtocol values
* Monitor `appProtocol implementations


## Proposal

Kubernetes supports the `appProtocol` field to provide a way to specify an application protocol for each Service port.

The [documentation](https://kubernetes.io/docs/concepts/services-networking/service/#application-protocol) for this field says that:

```Values should either be IANA standard service names or domain prefixed names such as mycompany.com/my-custom-protocol.```

This KEP proposes to declare standard Kubernetes protocol names for common protocols that are not IANA standard service names.

Those common protocols will be well defined strings prefixed with ‘k8s.io’. 
`k8s.io/h2c` as an example.

### list of protocols
- 'k8s.io/h2c'
- 'k8s.io/h2'
- 'k8s.io/grpc'
- 'k8s.io/grpc-web'
- 'k8s.io/tls'
- 'k8s.io/tcp'

### Risks and Mitigations

There are no real “risks”, primary concerns are:
1. End users will not migrate to `k8s.io/<>` values
2. It will take long time to implementations align with the new standards
3. We have no easy way to monitor who is aligned and who is not


## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

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

* The list of protocols can become stale fairly quick when new protocols are implemented before we decide to declare them as part of k8s.io common list. That can lead to a the current state again where implementations already implement support without a prefix (although they should not) OR with a domain prefix.


## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->
