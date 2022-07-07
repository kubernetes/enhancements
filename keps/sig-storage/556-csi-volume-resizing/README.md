# Support for CSI volume resizing

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [External resize controller](#external-resize-controller)
  - [Expansion on Kubelet](#expansion-on-kubelet)
    - [Offline volume resizing on kubelet:](#offline-volume-resizing-on-kubelet)
    - [Online volume resizing on kubelet:](#online-volume-resizing-on-kubelet)
    - [Supporting per-PVC secret refs](#supporting-per-pvc-secret-refs)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Test Plan](#test-plan)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

To bring CSI volumes in feature parity with in-tree volumes we need to implement support for resizing of CSI volumes.

## Motivation

We recently implemented volume resizing support in CSI specs. This proposal implements this feature for Kubernetes.
Any CSI volume plugin that implements necessary part of CSI specs will become resizable.

### Goals

To enable expansion of CSI volumes used by `PersistentVolumeClaim`s that support volume expansion as a plugin capability.

### Non-Goals

The expansion capability of a CSI plugin will not be validated by using CSI RPC call when user edits the PVC(i.e existing resize admission controller will not make CSI RPC call).
The responsibility of
actually enabling expansion for certains storageclasses still falls on Kubernetes admin.

## Proposal

The design of CSI volume resizing is made of two parts.


### External resize controller

To support resizing of CSI volumes an external resize controller will monitor all changes to PVCs.  If a PVC meets following criteria for resizing, it will be added to
controller's workqueue:

- The driver name disovered from PVC(via corresponding PV) should match name of driver currently known(by querying driver info via CSI RPC call) to external resize controller.
- Once it notices a PVC has been updated and by comparing old and new PVC object, it determines more space has been requested by the user.

Once PVC gets picked from workqueue, the controller will also compare requested PVC size with actual size of volume in `PersistentVolume`
object. Once PVC passes all these checks, a CSI `ControllerExpandVolume` call will be made by the controller if CSI plugin implements `ControllerExpandVolume`
RPC call.

If `ControllerExpandVolume` call is successful and plugin implements `NodeExpandVolume`:
- if `ControllerExpandVolumeResponse` returns `true` in `node_expansion_required` then `FileSystemResizePending` condition will be added to PVC and `NodeExpandVolume` operation will be queued on kubelet. Also volume size reported by PV will be updated to new value.
- if `ControllerExpandVolumeResponse` returns `false` in `node_expansion_required` then volume resize operation will be marked finished and both `pvc.Status.Capacity` and `pv.Spec.Capacity` will report updated value.

If `ControllerExpandVolume` call is successful and plugin does not implement `NodeExpandVolume` call then volume resize operation will be marked as finished and both `pvc.Status.Capacity` and `pv.Spec.Capacity` will report updated value.

If `ControllerExpandVolume`  call fails:
- Then PVC will retain `Resizing` condition and will have appropriate events added to the PVC.
- Controller will retry resizing operation with exponential backoff, assuming it corrects itself.

A general mechanism for recovering from resize failure will be implemented via: https://github.com/kubernetes/kubernetes/issues/73036

### Expansion on Kubelet

A CSI volume may require expansion on the node to finish volume resizing. In some cases - the entire resizing operation can happen on the node and
plugin may choose to not implement `ControllerExpandVolume` CSI RPC call at all.

Currently Kubernetes supports two modes of performing volume resize on kubelet. We will describe each mode here. For more information , please refer to original volume resize proposal - https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/grow-volume-size.md.


#### Offline volume resizing on kubelet:

This is the default mode and in this mode `NodeExpandVolume` will only be called when volume is being mounted on the node(or `MountVolume` operation in `operation_executor.go`). In other words, pod that was using the volume must be re-created for expansion on node to happen.

When a pod that is using the PVC is started, kubelet will compare `pvc.spec.resources.requests.storage` and `pvc.Status.Capacity`. It also compares PVC's size with `pv.Spec.Capacity` and if it detects PV is reporting same size as pvc's spec but PVC's status is still reporting smaller value then it determines -
a volume expansion is pending on the node.  At this point if plugin implements `NodeExpandVolume` RPC call then, kubelet will call it and:

If `NodeExpandVolume` is successful:
- It will update `pvc.Status.Capacity` with latest value and remove all resizing related conditions from PVC.

If `NodeExpandVolume` failed:
- It will add a event to both PVC and Pod about failed resizing and resize operation will be retried. This will prevent pod from starting up.


#### Online volume resizing on kubelet:

More details about online resizing can be found in [Online resizing design](https://github.com/kubernetes/enhancements/pull/737) but essentially if
`ExpandInUsePersistentVolumes` feature is enabled then kubelet will periodically poll all PVCs that are being used on the node and compare `pvc.spec.resources.requests.storage` and `pvc.Status.Capacity`(also `pv.Spec.Capacity`) and make similar determination about whether node expansion is required for the volume.

In this mode `NodeExpandVolume` can be called while pod is running and volume is in-use. Using aformentioned check if kubelet determines that
volume expansion is needed on the node and plugin implements `NodeExpandVolume` RPC call then, kubelet will call it(provided volume has already been node staged and published on the node) and:

If `NodeExpandVolume` is successful:
- It will update `pvc.Status.Capacity` with latest value and remove all resizing related conditions from PVC.

If `NodeExpandVolume` failed:
- It will add a event to both PVC and Pod about failed resizing and resize operation will be retried.

#### Supporting per-PVC secret refs

To support per-PVC secrets for volume resizing, similar to CSI attach and detach - this proposal expands `CSIPersistentVolumeSource` object to contain `ControllerExpandSecretRef` and `NodeExpandSecretRef`. This API change will be gated by `ExpandCSIVolumes` feature gate currently in Beta:

```
type CSIPersistentVolumeSource struct {
    ....
    // ControllerPublishSecretRef is a reference to the secret object containing
    // sensitive information to pass to the CSI driver to complete the CSI controller publish
    ControllerPublishSecretRef *SecretReference

    // ControllerExpandSecretRef is a reference to secret object containing sensitive
    // information to pass to the CSI driver to complete CSI controller expansion
    ControllerExpandSecretRef *SecretReference

    // NodeExpandSecretRef is a reference to secret object containing sensitive
    // information to pass to the CSI driver to complete CSI node expansion
    NodeExpandSecretRef *SecretReference
}
```

Secrets will be fetched from StorageClass with parameters `csi.storage.k8s.io/controller-expand-secret-name` and `csi.storage.k8s.io/controller-expand-secret-namespace`, `csi.storage.k8s.io/node-expand-secret-name` and `csi.storage.k8s.io/node-expand-secret-namespace`. Resizing secrets will support same templating rules as attach and detach as documented - https://kubernetes-csi.github.io/docs/secrets-and-credentials.html#controller-publishunpublish-secret .

Starting from 1.15 it is expected that all CSI volumes that require secrets for expansion will have `ControllerExpandSecretRef` field set. If not set
`ControllerExpandVolume` CSI RPC call will be made without secret. Existing validation of `PersistentVolume` object will be relaxed to allow
setting of `ControllerExpandSecretRef` for the first time so as CSI volume expansion can be supported for existing PVs.

Starting from 1.23 it is expected that all CSI volumes  that require secrets for online expansion will have `NodeExpandSecretRef` field set. If not set `NodeExpandVolume` CSI RPC call will be made without secret. Existing validation of `PersistentVolume` object will be relaxed to allow setting of `NodeExpandSecretRef` for the first time so as CSI volume expansion can be supported for existing PVs.

### Risks and Mitigations

Before this feature goes GA - we need to handle recovering https://github.com/kubernetes/kubernetes/issues/73036.

## Test Plan

* Unit tests for external resize controller.
* Add e2e tests in Kubernetes that use csi-mock driver for volume resizing.
  - (positive) Give a plugin that supports both control plane and node size resize, CSI volume should be resizable and able to complete successfully.
  - (positive) Given a plugin that only requires control plane resize, CSI volume should be resizable and able to complete successfully.
  - (positive) Given a plugin that only requires node side resize, CSI volume should be resizable and able to complete successfully.
  - (positive) Given a plugin that support online resizing, CSI volume should be resizable and online resize operation be able to complete successfully.
  - (negative) If control resize fails, PVC should have appropriate events.
  - (neative) if node side resize fails, both pod and PVC should have appropriate events.

## Graduation Criteria

Once implemented CSI volumes should be resizable and in-line with current in-tree implementation of volume resizing.

* *Alpha* :
  - Kubernetes - 1.14:  Initial support for CSI volume resizing. Released code will include an external CSI volume resize controller and changes to Kubelet. Implementation will have unit tests and csi-mock driver e2e tests.
  - Kubernetes - 1.15:  Add e2e tests that use real drivers(`gce-pd`, `ebs` at minimum). Add metrics for volume resize operations. Support per-PVC secret refs.
* *Beta*  : More robust support for CSI volume resizing, handle recovering from resize failures.
* *GA* : CSI resizing in general will only leave GA after existing [Volume expansion](https://github.com/kubernetes/enhancements/issues/284) feature leaves GA. Online resizing of CSI volumes depends on [Online resizing](https://github.com/kubernetes/enhancements/pull/737) feature and online resizing of CSI volumes will be available as a GA feature only when [Online resizing feature](https://github.com/kubernetes/enhancements/pull/737) goes GA.

Hopefully the content previously contained in [umbrella issues][] will be tracked in the `Graduation Criteria` section.

[umbrella issues]: https://github.com/kubernetes/kubernetes/issues/62096

## Implementation History

- 1.14 Implement CSI volume resizing as an alpha feature.
- 1.11 Move in-tree volume expansion to beta.
- 1.11 Implement online resizing feature for in-tree volume plugins as an alpha feature.
- 1.8 Implement in-tree volume expansion an an alpha feature.
- 1.23 Implement online resizing with secret for csi volume plugins as an beta feature.
