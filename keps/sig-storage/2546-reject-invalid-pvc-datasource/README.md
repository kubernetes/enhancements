# Reject Invalid PVC DataSource

## Table of Contents

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
- [Alternatives](#alternatives)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
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

[Volume Populators KEP]: ../1495-volume-populators/README.md

While developing the [Volume Populators KEP] it became obvious that the way the admission
controller handles the `DataSource` field of PVC specs is contrary to everyone's intentions.

The existing behavior, which has been in place since 1.12, is to clear the field if the user
supplies a value that is not one of the supported values. This results in Kubernetes behaving
as if the user requested an empty volume, and indeed the user is given an empty volume in
that case (absent any other errors).

Developers never intended the `DataSource` field to work this way, and it's clearly not good
practice to ignore user input. Most likely this was an accidental behavior resulting from the
need to support disabling feature gates for specific kinds of data sources.

Users cannot possibly depend on this behavior, because for every invalid request that we are
currently incorrectly treating as valid, there is an actual valid request that results in the
exact same result -- specifically a PVC with an empty data source.

## Motivation

### Goals

- Fix the buggy admission controller behavior
- Generate a clearly visible release note for users in the unlikely event that anyone is
  affected.

### Non-Goals

- This change makes sense regardless of the future of the Volume Populators KEP. As such
  is it not a goal of this KEP to advance the Volume Populators feature, although both
  KEPs together to make good sense.

## Proposal

We propose to correct this bad behavior by making the admission controller reject invalid
data sources instead of silently changing them. This would cause the admission controller
to reject PVCs that specify core object other than PVCs as data sources when the
AnyVolumeDataSource feature gate is enabled, and to reject PVCs that specify anything
other than the two supported data sources (PVCs and VolumeSnapshots) when the feature gate
is disabled.

No existing workflows would be affected by this change, unless they are accidentally
specifying a data source when they wanted an empty volume. Correcting those workflows would
be trivial -- users would simply need to remove the data source if they wanted an empty
volume. Valid data sources would be completely unaffected.

### Risks and Mitigations

The main risk is if a user had some preconfigured workflow that involved creation of
PVCs with the `DataSource` field set to an invalid value. The workflow would be yielding
empty volumes today, and if that was acceptable (most likely because the user forgot
they ever put contents in the `DataSource` field) then it could go on unnoticed. Fixing
this bug will cause that user's workflow to suddenly break.

The workaround for the user would be trivial, they would just need to clear the
`DataSource` field in their requests to obtain the old behavior. One can imagine
situations where changing the workflow is non-trivial, but it's not unheard of for users
to have to take actions upon Kubernetes upgrades so that we can correct bad decisions
from our past.

The real question is the liklihood of users making the particular mistake outlined above.
Given that the only supported contents of that field was other PVCs and VolumeSnapshots,
and both of those types of data sources are GA today, it seems impossible that users
who wanted to clone a PVC or a VolumeSnapshot would be doing so incorrectly, because the
fact that they were receiving an empty volume would make them fix their workflow. It's
almost impossible to imagine a case where a user put something else in that field and
expected it to work, except for the handful of developers working on the volume populators
feature. The only remaining possibility is users who were intentionally trying out
something they expected not to work, and were surprised that it did work, and for some
reason continued to use the malformed workflow. This seems surpassinly unlikely.

## Design Details

### Test Plan

* Add test case that creates a PVC with a `DataSource` pointing to a core object and expect
creation to fail.
* Add test case that creates a PVC with a `DataSource` pointing to a definitely-invalid 
object and expect creation to fail. (This test case becomes obsolete after volume populator
modifies this.)

## Alternatives

The only viable alternative that I can see is to forever enshrine the existing
behavior and accept that it's gross.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [X] Other
  - Describe the mechanism: Cannot be disabled
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

Use of invalid datasources, which would previously have been ignored and yielded
an empty volume, will now properly result in an error.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

No

###### What happens if we reenable the feature if it was previously rolled back?

n/a

###### Are there any tests for feature enablement/disablement?

No, the change will always be on.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout fail? Can it impact already running workloads?

As this is effectively a bug fix, the rollout cannot fail. Also, because the change
alters PVC creation, already-running workloads won't be affected.

###### What specific metrics should inform a rollback?

Rollback is not possible.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

n/a

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

This is a bug fix, so not applicable.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [X] Other (treat as last resort)
  - Details: It's not a service, and doesn't have a health status per se. One could track
    failed PVC creations to notice if users were relying on the previously buggy behavior.

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

The expectation is that nobody would rely on this particular bug, because if they wanted
an empty volume, it's much easier to leave the data source field empty. So we don't
anticipate the error case ever occurring.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Failed PVC creations.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No, it's a change to kube-api-server.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No impact, because the change is in API server itself.

###### What are other known failure modes?

The only way this change can "fail" is if a user was relying on the old buggy behavior.
The best remediation in that case is for the user to stop filling in the `DataSource`
field of the PVC with an invalid value and leave it empty isntead.

###### What steps should be taken if SLOs are not being met to determine the problem?

Fix the user's PVC definitions.
