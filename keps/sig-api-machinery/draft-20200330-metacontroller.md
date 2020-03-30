---
title: Metacontroller As A Subproject
authors:
  - "@AmitKumarDas"
  - "@alaimo"
owning-sig: sig-api-machinery
participating-sigs:
  - sig-api-machinery
reviewers:
  - "@DirectXMan12"
  - "@enisoc"
  - "@mikebryant"
  - "@piersharding"
  - "@luisdavim"
  - "@rosenhouse"
  - "@kmova"
  - “@droot”
approvers:
  - "@DirectXMan12"
editor: "@AmitKumarDas"
creation-date: 2020-03-30
last-updated: 2020-03-30
status: provisional
see-also:
  - "https://github.com/GoogleCloudPlatform/metacontroller/issues/184"
  - "https://github.com/kubernetes/kubernetes/issues/80948"
---

# Metacontroller As A Subproject

<!-- toc -->
- [Summary](#Summary)
  - [What Is Metacontroller?](#What-Is-Metacontroller?)
  - [Why Should Metacontroller Become A Sig-API-Machinery Subproject?](#Why-Should-Metacontroller-Become-A-Sig-API-Machinery-Subproject?)
- [Goals](#Goals)
- [Proposal](#Proposal)
- [Risks and Mitigations](#Risks-and-Mitigations)
- [Graduation Criteria](#Graduation-Criteria)
- [Alternatives](#Alternatives)
  - [Absorb Metacontroller Into Another Project](#Absorb-Metacontroller-Into-Another-Project)
  - [Metacontroller Continues As A Non-Sig Project](#Metacontroller-Continues-As-A-Non-Sig-Project)
  - [Abandon Metacontroller](#Abandon-Metacontroller)

<!-- /toc -->

## Summary

### What Is Metacontroller?

[Metacontroller](https://github.com/GoogleCloudPlatform/metacontroller/) is an add-on for Kubernetes that makes it easy for developers and administrators to write and deploy custom controllers. Custom controllers define the behavior of a new extension to the Kubernetes API. The Metacontroller APIs make it easy to define behavior for new API extensions or add custom behavior to existing APIs.

In the Kubernetes world there are web hooks that deal with validating, conversion, defaulting, and mutating logic for both custom and native resources. Metacontroller enables web hooks of a different nature. It provides webhooks to implement reconciliation logic for any Kubernetes resource (custom or native). It expects developers to understand the feedback loop that Kubernetes reconciliation is based on. Developers only need to ensure that their logic is idempotent during synchronization stages. Metacontroller abstracts away all low level Kubernetes aspects from this reconcile logic.

Metacontroller is language agnostic. Business logic for controllers and operators are delegated to web hook(s) which can be implemented in almost any modern programming language. This ensures a low barrier of entry for those looking to extend the Kubernetes platform. Developers do not need to understand much about controllers, Golang, or Kubernetes client libraries to start producing viable controllers or operators.

Metacontroller simplifies unit testing. It enables developers to easily test their controller’s business logic in the absence of a Kubernetes cluster. Additionally, with Metacontroller, developers are not required to maintain any generated code. Developers only maintain the code that is necessary to execute their business requirements. Using this approach, developers can achieve better test coverage just by following idiomatic language guidelines.

### Why Should Metacontroller Become A Sig-API-Machinery Subproject?

Metacontroller has not been actively maintained for more than 9 months. Its last commit dates back to 25th March, 2019. End users have inquired about the future of the project and shown significant interest in seeing the project continue to be maintained [here](https://github.com/GoogleCloudPlatform/metacontroller/issues/184).

Users of Metacontroller feel that the Kubernetes' Github organization will be the best neutral home for the project. The Kubernetes organization provides a sense of continuity and clear [governance rules](https://github.com/kubernetes/community/blob/master/committee-steering/governance/sig-governance.md) that are in the best interests of Metacontroller’s users and future maintainers. Joining the Kubernetes organisation as subproject would provide a wealth of information and resources that will contribute to the success, viability, and longevity of the Metacontroller project going forward.

## Goals

Establish a new repository for Metacontroller at github.com/kubernetes-sigs/metacontroller to be maintained by the open source Kubernetes community and governed as a subproject of sig-api-machinery.

## Proposal

Donate Metacontroller to sig-api-machinery as a subproject under the guidelines specified [here](https://github.com/kubernetes/community/blob/master/github-management/kubernetes-repositories.md#rules-for-donated-repositories). If possible, the original Metacontroller repository will be transferred directly, otherwise it will be archived. The new source control location will become the authoritative source of truth for all issues and pull requests. As an independent subproject under sig-api-machinery, Metacontroller will maintain the same Apache license hosted [here](https://github.com/kubernetes-sigs). 

The following group of community members will serve as initial maintainers of the new repository:

* @luisdavim
* @alaimo 
* @AmitKumarDas

Maintainers will devote time to transfer, maintain, and release the Metacontroller code base in a time bound manor. Maintainers will document features, blog, evangelize, and respond to the community on slack, groups, forums, etc. Maintainers will serve as the initial owners of the subproject.

Once a new home for the repository has been established, the maintainers will institute a "cool-off" period to:

* Establish a common understanding amongst the new maintainers
* Evaluate and merge changes from new and outstanding pull requests.
* Triage, classify, and transfer open issues accordingly
* Build or enhance unit tests, integration tests, CI CD pipelines, etc.
* Document additional examples that simulate end user adoption
* Establish formal procedures for governance

Following the cool-off period, Metacontroller will be developed and maintained under the newly established and compliant governance rules.

## Risks and Mitigations

 Reductions in end-user adoption and project momentum are the primary risks during the transition. These can be mitigated by migrating the repository to a permanent community owned and maintained home as quickly as possible. This includes transferring ownership of the codebase from GCP to CNCF and any related privately held assets (e.g. https://metacontroller.app/). Progress must also be regularly communicated to the Metacontroller community.

## Graduation Criteria

* Metacontroller is adopted as subproject of a sig-api-machenery
* Formal procedures for governance as documented and enacted
* Ownership of resources is transfered to CNCF
* Project members and management are established
* Core committers are established and inducted
* Permanent home for the repository is established and communicated
* All documentation, source control, tests and project roadmaps are updated and inline with sig standards

## Alternatives

### Absorb Metacontroller Into Another Project
At present, only the [KubeBuilder](https://github.com/kubernetes-sigs/kubebuilder) project has been suggested as a parent that could absorb Metacontroller. So far, the current community response has been that Metacontroller is sufficiently different in its objectives, approach, and customer base such that it should not be absorbed into KubeBuilder. It should instead remain a standalone project. Additionally, while not a deciding factor, it should be noted that Metacontroller is not presently built with [controller runtime](https://github.com/kubernetes-sigs/controller-runtime). 

### Metacontroller Continues As A Non-Sig Project
This is undesired since it is unlikely that Metacontroller will find an alternative home that will provide as much confidence or exposure to end users as the sig-api-machinery will. Alternatively, private sponsorship could be an option, but it is the opinion of the authors of this proposal that it would restrict innovation and potentially lead back to an unmaintained repository. An alternative foundation like the Apache Software Foundation could be a potential alternative to CNCF, but given the objectives of Metacontroller, sig-api-machinery is a natural fit and should remain the primary goal.

### Abandon Metacontroller
There is significant community interest for the Metacontroller project to continue on. Organisations have invested in production software powered by Metacontroller. Well known alternatives for bootstrapping Kubernetes controllers like KubeBuilder and The Operator Framework do not provide equivalent declarative, language agnostic feature sets. It is the opinion of the community members contributing to this proposal that abandoning Metacontroller should be avoided.
