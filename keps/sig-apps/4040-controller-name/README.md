# KEP-4040: Controller Name

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Upgrade](#upgrade)
    - [Downgrade](#downgrade)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This feature unblocks custom handling for selected standard
Kubernetes API objects, while still allowing the default
behavior for other API objects. For now it focuses only on Job and HPA, but other controllers can reuse this pattern.

## Motivation

Currently Kubernetes users, who need custom behavior for
some API objects, can either:

* disable the corresponding controller in controller-manager
and implement a new one that provides both modified(for marked
objects) and original(for all other) functionality.

* duplicate/fork the API as CRD.

However, on managed environments (like, for example, GKE),
only the second option is available as cloud providers often
don't allow to modify `controller-manager` configuration.

### Goals

* Establish a simple and common method to mark an API object as
to be handled by a non default controller, in a similar fashion
as pods can specify their custom scheduler.

* Apply it and provide implementation for Job and HPA.

### Non-Goals

* Add controller name to all possible K8S objects at once.


* While the KEP enables providing custom handling for standard
K8S APIs,
it doesn't focus on any customization or additional
configuration that might be possible using it. Additional
configuration options may be subject for the following
enhancement proposals.


## Proposal

Add `controllerName` at the top level of Spec of API objects to
specify which controller should handle them. If the field
value is `”default”` or the field is not specified at all,
then the default controller from controller manager handles the
object in the usual way. If the specified  value is different,
then the controller, which identifies itself with the given name, should handle the object and all others ignore it.

Initially add the field only to Job and HPA. If the feedback is good, consider expansion to other objects.

### User Stories (Optional)

#### Story 1

I want to run a modified, custom version of  ControllerManager in
cluster and handle some (not all) of the Jobs with it.

#### Story 2

I want to provide a significantly different implementation of
HPA and gradually test it in production.

### Notes/Constraints/Caveats (Optional)

Other controller owners may decide whether and when to introduce
the field. Allowing it on security-related APIs may be more
controversial than on `HorizontalPodAutoscaler` or `Job`.

### Risks and Mitigations

Objects handled by custom controllers may have different field value
combinations than handled by the default controller manager. These
combinations, while correct from the API validation perspective,
may project logical error or simply be not understood by dependant
code, and, as a result, potentially crash some external components.
For example, a job with .status.failed higher than .spec.backoffLimit
should be marked as Failed, but with a custom controller there is
not guarantee it will be.

Some of the APIs are in GA and in K8S conformance tests. With a
custom controller, there is no guarantee that the object will
behave in GA or test-conformant manner.

Adding the field on security-related APIs may potentially
mislead the system administrators and, as a result misconfiguration risks.

To minimize the risks we can start with supporting the field
on just 2 APIs - HPA and Job, but the same field can be added to other objects.

There may be two controllers identifying with the same name, just like
there can be two misconfigured schedulers reacting for the same name.
Similarly, there will be no migitiation for this problem.

## Design Details

The supported API controllers will simply skip API objects that have
non-empty and non-default `controllerName` field. API objects with
empty `controllerName` will be defaulted to `"default"`.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

No additional tests need to be added prior to implementing this enhancement.

##### Unit tests

Regular unit tests for the feature should suffice. The implementation
should be quite straightforward (skip object if `controllerName != “default”`)
and most likely doesn't need extensive unit tests.

##### Integration tests

Integration test should check whether the supported api object is
processed when the `controllerName` field is set.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag for the supported APIs
(proposal: apps, batch and autoscaling).
- Unit and integration tests are passing.

#### Beta

- Feature gate enabled by default.
- Agreed the list of additionally supported APIs.
- E2E tests are passing.

#### GA

- Enough user feedback is gathered.
- No bugs/issues reported.

### Upgrade / Downgrade Strategy

#### Upgrade

No previous behavior is broken.

#### Downgrade

The feature stops working. As a result, the some API object may be
processed by 2 controllers, the default one and the custom one. So,
before the downgrade the additional controller should be stopped.
The objects that cannot be processed by the default controller should
be removed.

### Version Skew Strategy

N/A.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ControllerName
  - Components depending on the feature gate: kube-controller-manager, apiserver
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

API objects with the proposed field set will be handled differently.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, although disabling will require user attention. Objects, with the
proposed field, expect some different handling and may need to be
deleted before the feature is turned off.


###### What happens if we re-enable the feature if it was previously rolled back?

If there are any objects with the `controllerName` field set they will start to behave in a different way.

###### Are there any tests for feature enablement/disablement?

No.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

After the rollback the field value may be lost.

###### What specific metrics should inform a rollback?

N/A.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No, nothing is deprecated on the rollout.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

They can list objects having non-default/non-empty `controllerName` field.

###### How can someone using this feature know that it is working for their instance?

The assumption is that annotated objects should still, more-or-less, do their
job, but in a slightly different fashion. The custom controllers are expected to update
status, conditions, etc.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Not directly. The new, designated controller may however issue new API calls
on its own to handle the marked objects.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

Not directly. The new, designated controller may however issue new cloud
provider calls on its own to handle the marked objects.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Not directly. The new, designated controller may however create additional API objects
on its own to handle the marked objects.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Not directly.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Not directly.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

Not directly.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Users will not be able to set the field. Custom controllers may not work.

###### What are other known failure modes?

N/A.

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History


## Drawbacks


## Alternatives

* Disable controllers in `controller-manager` and provide a
complete replacement. It will not work on managed K8S
environments where `controller-manager` flags/configuration are
not exposed to the users.

* Generic label/annotation `controller.kubernetes.io/name`.

* Controller specific label/annotation like `”job.batch.kubernetes.io/controller-name”`.

## Infrastructure Needed (Optional)

