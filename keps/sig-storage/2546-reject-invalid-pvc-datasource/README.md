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