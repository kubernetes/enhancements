---
title: CSI Volume Topology
authors:
  - "@verult"
owning-sig: sig-storage
participating-sigs:
  - sig-storage
reviewers:
  - "@msau42"
  - "@saad-ali"
approvers:
  - "@msau42"
  - "@saad-ali"
editor: TBD
creation-date: 2019-01-24
last-updated: 2020-04-16
status: implemented
see-also:
  - n/a
replaces:
  - n/a
superseded-by:
  - n/a
---

# Title

CSI Volume Topology

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Upgrade/Downgrade Strategy](#upgradedowngrade-strategy)
  - [Deprecations](#deprecations)
- [Version Skew Strategy](#version-skew-strategy)
- [Test Plan](#test-plan)
  - [GA testing](#ga-testing)
- [Graduation Criteria](#graduation-criteria)
  - [Alpha-&gt;Beta](#alpha-beta)
  - [Beta-&gt;GA](#beta-ga)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

This KEP is written after the original design doc has been approved and implemented. Design for CSI Volume Topology Support in Kubernetes is incorporated as part of the [CSI Volume Plugins in Kubernetes Design Doc](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/container-storage-interface.md).

The rest of the document includes required information missing from the original design document: test plan and graduation criteria.

## Upgrade/Downgrade Strategy
When moving to GA in K8s 1.17, the storage/v1 CSINode object will be introduced. 1.17
kubelets will immediately start creating v1 CSINode objects. 1.17 in-tree
controllers (scheduler and attach-detach controller) can switch to using the v1 APIs.

External controllers like the csi-provisioner and csi-attacher can start using
the v1 objects if they are able to coordinate and align their deployment with the
deployment of the API server. However, this may not always be possible, and
newer sidecars should be able to continue to work on older Kubernetes versions.
Therefore, these sidecars need to continue to support both v1beta1 and v1 versions
for the duration of the deprecation period. They can try the v1 API first, and then
fallback to v1beta if the v1 API doesn't exist.

### Deprecations
The v1beta1.CSINode object will be deprecated in K8s 1.17, and can be removed in
1.20 according to the Kubernetes deprecation policy.

What that means is in 1.20, CSI drivers using older versions of CSI sidecars
that are not aware of v1 objects will stop functioning unless they upgrade to
newer versions of the sidecars that are v1-aware.

Similarly, we will deprecate v1beta1.CSINode support in the CSI sidecars in the
sidecar release where we introduce v1 support, and remove v1beta1 support from
the sidecar in the same release corresponding to K8s 1.20. That removal will
require a new major version of the CSI sidecars.

## Version Skew Strategy
CSI sidecars will support the scenario with K8s 1.13 nodes where
CSINode may not have existed until we remove support for v1beta1 in 1.20 and
bump the major version. This behavior also needs to be deprecated when we
release the v1-aware sidecar version.

## Test Plan
* Unit tests around topology logic in kubelet and CSI external-provisioner.
* New e2e tests around topology features will be added in CSI e2e test suites, which test various volume operation behaviors from the perspective of the end user. Tests include:
  * (Positive) Volume provisioning with immediate volume binding and AllowedTopologies set.
  * (Positive) Volume provisioning with delayed volume binding.
  * (Positive) Volume provisioning with delayed volume binding and AllowedTopologies set.
  * (Negative) Volume provisioning with immediate volume binding and pod zone missing from AllowedTopologies.
  * (Negative) Volume provisioning with delayed volume binding and pod zone missing from AllowedTopologies.
Initially topology tests are run against a single CSI driver. As the CSI test suites become modularized they will run against arbitrary CSI drivers.

### GA testing
* Manual e2e testing for upgrade and version skew scenarios.
  * An older sidecar that only understands v1beta1 should continue to work when the cluster is
    upgraded to 1.17. Upgrading the sidecar after cluster upgrade to a version that understands v1 objects should also continue to work.
  * A newer sidecar that supports v1 and v1beta1 should continue to work if Master and Nodes are < 1.17.
  * A newer sidecar that supports v1 and v1beta1 should continue to work if Nodes are < 1.17.
  * A newer sidecar that only supports v1 should work if Master is >= 1.17. Nodes
    can be >= 1.15 and still use the v1beta object.

## Graduation Criteria

### Alpha->Beta

* Feature complete, including:
  * Volume provisioning with required topology constraints
  * Volume provisioning with preferred topology
  * Cluster-wide topology aggregation
  * StatefulSet volume spreading
* Depends on: CSINodeInfo beta or above; Kubelet Plugins Watcher beta or above
* Unit and e2e tests implemented

### Beta->GA

* Depends on: CSINodeInfo GA; Kubelet Plugins Watcher GA
* Stress test: provisioning load tests; node scale tests; component crash tests
* Feature deployed in production and have gone through at least one K8s upgrade.
* Upgrade and version skew testing.

## Implementation History

* K8s 1.12: Alpha
* K8s 1.14: Beta
* K8s 1.17: GA
