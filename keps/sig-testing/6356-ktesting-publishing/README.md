<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md

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
# KEP-6356: publishing ktesting

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Using ktesting in k8s.io/dynamic-resource-allocation](#using-ktesting-in-k8siodynamic-resource-allocation)
    - [Moving pkg/scheduler to staging](#moving-pkgscheduler-to-staging)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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
  - [Publishing as separate repo, usage in Kubernetes](#publishing-as-separate-repo-usage-in-kubernetes)
  - [Publishing as separate repo, no usage in Kubernetes](#publishing-as-separate-repo-no-usage-in-kubernetes)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [X] e2e Tests for all Beta API Operations (endpoints) (not applicable and hence done)
  - [X] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) (not applicable and hence done)
  - [X] (R) Minimum Two Week Window for GA e2e tests to prove flake free (not applicable and hence done)
- [X] (R) Graduation criteria is in place
  - [X] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA (not applicable and hence done)
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io] (not applicable, docs are automatically provided via `go docs` and https://pkg.go.dev, hence done)
- [X] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

ktesting is a set of Go packages that support writing unit, integration and E2E
tests. Two of them are currently located in k8s.io/kubernetes/test/utils:
- k8s.io/kubernetes/test/utils/ktesting: abstraction around Go testing and Ginkgo
- k8s.io/kubernetes/test/utils/client-go/ktesting: tailored towards testing Kubernetes as a client
  with client-go calls

The proposal in this KEP is to publish k8s.io/kubernetes/test/utils/ktesting in
a new k8s.io/ktesting staging repo and
k8s.io/kubernetes/test/utils/client-go/ktesting as a k8s.io/client-go/ktesting
sub-package.

## Motivation

The main motivation is to enable the use of ktesting in all code developed in
the kubernetes/kubernetes repository, including staging repos. Currently it can
be used only in core Kubernetes tests, not staging repos, because that would
cause an import cycle.

A secondary motivation is to enable use outside of Kubernetes. Maybe being able
to use it will motivate other developers to contribute to it.

### Goals

- publish the core ktesting package as k8s.io/ktesting
- move the client-go support to k8s.io/client-go/ktesting

### Non-Goals

- define a policy when to use or not to use ktesting
- large-scale conversion of tests which aren't using ktesting yet; this can be
  decided by code owners on a case-by-case basis

## Proposal

First the new k8s.io/ktesting staging repo needs to be created. Then a single
PR will move the existing code and adapt package names to keep everything
compiling without errors.

No further work is needed. The ktesting packages are fully documented, so `go
doc` and https://pkg.go.dev will provide the necessary documentation for
developers.

The publicly available ktesting packages will have the same
[Go API stability goals](https://github.com/kubernetes/community/blob/main/contributors/devel/sig-architecture/go_api_changes.md#stability-goals)
as other staging repos:
- no plan to stabilize a v1 package API
- API breaks will be avoided as much as possible, but may happen if needed

### User Stories

#### Using ktesting in k8s.io/dynamic-resource-allocation

ktesting was developed alongside DRA and is used in various in-tree tests. It
is jarring that the same functionality is not available when working on the DRA
code in staging, k8s.io/dynamic-resource-allocation, because tests have to be
written differently and behave differently.

Once ktesting is available for use in staging repos, tests in
k8s.io/dynamic-resource-allocation can be updated to be consistent.

#### Moving pkg/scheduler to staging

Some code under pkg/scheduler uses test/utils/ktesting. To move that code into
k8s.io/kube-scheduler, either tests must be rewritten or ktesting must be made
usable from k8s.io/kube-scheduler.

### Risks and Mitigations

Outside usage of ktesting might lead to an expectation of Go API stability.
ktesting is designed as much as possible to allow future extensions without
breaking Go APIs (concrete type instead of interface, functional configuration
parameters, etc.). Should the need arise to break an API, then the existing
documentation makes it clear that this is allowed. As a staging repo, such
a change can be made in a single atomic PR.

Outside usage of ktesting might cause feature requests or bug reports. Bug
reports are useful because they help improving the code. Feature requests may
provide useful ideas, but there is no obligation to implement new features.
Pull requests which implement new features may get reviewed and merged, but as
always only if reviewers have the time and motivation to do so.

Because ktesting is never used in production code, there is no risk that it'll
ever cause a security issue in components using it. A CI might be affected, but
this is not different from running tests developed by a contributor.

## Design Details

For further information about the design and features of ktesting,
see [test/utils/ktesting docs](https://github.com/kubernetes/kubernetes/blob/ae161646fec55fa74209c42e58bcce30f53f3fe6/test/utils/ktesting/doc.go#L17-L19)
or better (once available)
https://pkg.go.dev/k8s.io/kubernetes@v1.38.0-alpha.1/test/utils/ktesting

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `k8s.io/kubernetes/test/utils/ktesting`: 2026-09-14 - 89.6%
- `k8s.io/kubernetes/test/utils/ktesting/format`: 2026-09-14 - 80.0%
- `k8s.io/kubernetes/test/utils/client-go/ktesting`: 2026-09-14 - 80.9%

##### Integration tests

e2e framework

##### e2e tests

Not applicable because ktesting itself does not run in a cluster.

### Graduation Criteria

#### GA

- ktesting published as proposed

#### Deprecation

None.

### Upgrade / Downgrade Strategy

Not applicable.

### Version Skew Strategy

Not applicable.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

Not applicable.

###### How can this feature be enabled / disabled in a live cluster?

Not applicable.

###### Does enabling the feature change any default behavior?

Not applicable.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Not applicable.

###### What happens if we reenable the feature if it was previously rolled back?

Not applicable.

###### Are there any tests for feature enablement/disablement?

Not applicable.

### Rollout, Upgrade and Rollback Planning

Not applicable.

###### How can a rollout or rollback fail? Can it impact already running workloads?

Not applicable.

###### What specific metrics should inform a rollback?

Not applicable.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not applicable.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

Not applicable.

### Monitoring Requirements

Not applicable.

###### How can an operator determine if the feature is in use by workloads?

Not applicable.

###### How can someone using this feature know that it is working for their instance?

Not applicable.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Not applicable.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Not applicable.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Not applicable.

### Dependencies

The core ktesting has a [tightly controlled](https://github.com/kubernetes/kubernetes/blob/ae161646fec55fa7/test/utils/ktesting/.import-restrictions)
set of dependencies besides the standard library:
- github.com/go-logr
- github.com/onsi/gomega
- go.uber.org/goleak (test-only)
- k8s.io/klog/v2/ktesting
- sigs.k8s.io/yaml

This should make it usable in a wide variety of other modules. If some of these
dependencies are not acceptable for a module, then ktesting cannot be used
there.

client-go currently depends on all of these already, except for Gomega. Instead it
depends on testify, which has worse indirect dependencies than Gomega. In
particular, testify depends on go-spew, an unwanted dependency in Kubernetes.

It's under debate whether test-only dependencies should be considered (see
["analyze production dependencies"](https://github.com/kubernetes/kubernetes/pull/138838)).
If we care, then replacing testify with Gomega might be a good next step.
See ["Assertions"](https://github.com/kubernetes/kubernetes/blob/ae161646fec55fa74209c42e58bcce30f53f3fe6/test/utils/ktesting/doc.go#L287-L401)
for an explanation how and why ktesting integrates support for Gomega.

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

Not applicable because not used in production code.

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

Not applicable.

###### How does this feature react if the API server and/or etcd is unavailable?

Not applicable.

###### What are other known failure modes?

None.

###### What steps should be taken if SLOs are not being met to determine the problem?

Not applicable.

## Implementation History

ktesting was born out of the frustration of having to extend all functions
along a deep call tree such they pass through a generic client in addition to
several other client-go related parameters and the context. The actual code
then created arbitrary objects from YAML files, with automatic cleanup and
reporting errors as test failures. This would have been useful for both Go and
Ginkgo tests, except that the lack of a common API prevented the code reuse.

Nowadays ktesting provides that common API and access to everything tests need
(context, access to client-go clients) with a single parameter. It has already
been cleaned up to make it directly suitable for publishing.

Development of ktesting was covered in:
- KubeCon Contributor Summit 2024: ["Unified framework for unit, integration and E2E testing​"](https://youtu.be/VCG559w9gzo?si=GEl_FGXrFJD7rNOy)
- [Jan 7, 2025 SIG Testing meeting](https://docs.google.com/document/d/1z8MQpr_jTwhmjLMUaqQyBk1EYG_Y_3D4y4YdMJ7V1Kk/edit?tab=t.0#heading=h.ysuoj1k3i9v3) (no YouTube recording available)

## Drawbacks

None. Development of ktesting will continue as before.

## Alternatives

### Publishing as separate repo, usage in Kubernetes

This would be viable. However, it is less desirable than a staging repo because
future changes will be harder to make: first the out-of-tree repo needs to be
updated, then the update needs to be vendored, and only then can tests inside
Kubernetes use a new feature.

Setting up test jobs which ensure that updates don't break Kubernetes will
cause further work and will be slightly more costly than running more tests in
existing jobs because of the cold Go caches.

A separate repo would have the advantage that it would be possible to version
releases separately, possibly even with a v1. However, such a v1 and this kind
of manual release control are not goals. If they become one, then we can still
switch to modifying and releasing directly in the github.com/kubernetes/ktesting repo,
with no visible difference for consumers (same import path).

### Publishing as separate repo, no usage in Kubernetes

Besides having to rewrite existing tests before this is even feasible, it would
also lead to the same undesirable situation as with sigs.k8s.io/e2e-framework:
SIG Testing would directly or indirectly own code that doesn't benefit the
development of Kubernetes itself.

## Infrastructure Needed

A staging repo.
