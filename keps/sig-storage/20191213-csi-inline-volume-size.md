---
title: Size Parameter for CSI Ephemeral Inline Volumes
authors:
  - "@pohly"
owning-sig: sig-storage
reviewers:
  - TBD
approvers:
  - TBD
editor: "@pohly"
creation-date: 2019-12-13
last-updated: 2019-12-13
status: provisional
see-also:
  - "https://github.com/kubernetes/enhancements/pull/1353"
---

# Size Parameter for CSI Ephemeral Inline Volumes

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Ephemeral local volumes](#ephemeral-local-volumes)
  - [Capacity-aware scheduling of pods](#capacity-aware-scheduling-of-pods)
  - [fsSize field](#fssize-field)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
<!-- /toc -->

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links
      to KEP (this should be a link to the KEP location in
      kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in
      [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents,
      links to mailing list discussions/SIG meetings, relevant
      PRs/issues, release notes

## Summary

The [CSIVolumeSource API in Kubernetes
1.16](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#csivolumesource-v1-core)
for [CSI ephemeral inline
volumes](https://kubernetes-csi.github.io/docs/ephemeral-local-volumes.html)
currently supports only two standardized parameters for the volume:
- `readOnly`
- `fsType`

All other parameters for the volume must be passed as
`volumeAttributes` with key/value pairs that are driver-specific and
thus opaque to Kubernetes. This API was chosen because it directly
maps to parameters supported by the underlying
[NodePublishVolumeRequest](https://github.com/container-storage-interface/spec/blob/4731db0e0bc53238b93850f43ab05d9355df0fd9/lib/go/csi/csi.pb.go#L3604-L3652).

One common questions that developers and users have is how the size of
the volume can be specified. The proposal is to introduce a
standardized `fsSize` parameter as answer to that and gradually
replace driver-specific parameters that may have been used before.

## Motivation

### Goals

- Add a `CSIVolumeSource.fsSize` field.
- Pass it to drivers as additional entry in `NodePublishVolumeRequest.VolumeContext`
  if the driver has opted into getting additional fields there.

### Non-Goals

- Validate whether a CSI driver really supports this field. This is
  consistent with `readOnly` and `fsType` which also may be silently
  ignored.

## Proposal

### User Stories

#### Ephemeral local volumes

Local volumes are useful as scratch
space. [PMEM-CSI](https://github.com/intel/pmem-csi) and
[TopoLVM](https://github.com/cybozu-go/topolvm/blob/master/README.md)
are two examples for CSI drivers which dynamically create volumes of a
fixed capacity and thus need a size parameter.

### Capacity-aware scheduling of pods

Scheduling pods with ephemeral inline volumes onto nodes [with
sufficient free storage
capacity](https://github.com/kubernetes/enhancements/pull/1353)
depends (among other information, like free capacity) on knowing the
size of the volumes. If the size is only specified via some
vendor-specific parameter, it's not available to the Kubernetes
scheduler.

### fsSize field

A new field `fsSize` of type `*Quantity` in
[CSIVolumeSource](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#csivolumesource-v1-core)
needs to be added, alongside the existing `fsType`. It must be a
pointer to distinguish between no size and zero size selected.

The new field is under the `CSIInlineVolumeSize` feature
gate.

While `fsType` can (and does get) mapped to
`NodePublishVolumeRequest.volume_capability`, for `fsSize` we need a
different approach for passing it to the CSI driver because there is
no pre-defined field for it in `NodePublishVolumeRequest`. We can
extend the [pod info on
mount](https://kubernetes-csi.github.io/docs/pod-info.html) feature:
if (and only if) the driver enables that, then a new
`csi.storage.k8s.io/size` entry in
`NodePublishVolumeRequest.volume_context` is set to the string
representation of the size quantity. An unset size is passed as empty
string.

This has to be optional because CSI drivers written for 1.16 might do
strict validation of the `volume_context` content and reject volumes
with unknown fields. If the driver enables pod info, then new fields
in the `csi.storage.k8s.io` namespace are explicitly allowed.

Using that new `fsSize` field must be optional. If a CSI driver
already accepts a size specification via some driver-specific
parameter, then specifying the size that way must continue to
work. But if the `fsSize` field is set, a CSI driver should use that
and treat it as an error when both `fsSize` and the driver-specific
parameter are set.

Setting `fsSize` for a CSI driver which ignores the field is not an
error. This is similar to setting `fsType` which also may get ignored
silently.

### Risks and Mitigations

TBD

## Design Details

### Test Plan

TBD

### Upgrade / Downgrade Strategy

TBD

### Version Skew Strategy

TBD

## Implementation History

## Drawbacks

Why should this KEP _not_ be implemented - TBD.
