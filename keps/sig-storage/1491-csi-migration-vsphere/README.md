# In-tree Storage Plugin to CSI Migration - vSphere Design Doc

## Table of Contents

<!-- toc -->
- [Summary](#summary)
  - [New Feature Gates](#new-feature-gates)
- [Design Details](#design-details)
  - [Prerequisite testing updates](#prerequisite-testing-updates)
  - [Unit tests](#unit-tests)
  - [Integration tests](#integration-tests)
  - [e2e tests](#e2e-tests)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
- [Implementation History](#implementation-history)
<!-- /toc -->


## Summary

This document present as a vendor specific KEP for the parent KEP
[CSI Migration](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/625-csi-migration)

This inherits all the contents from its parent KEP. It will introduce two new feature gates to be 
used as as described in its parent KEP. For all other contents, please refer to the parent KEP.

### New Feature Gates

- CSIMigrationvSphere
  - As describe in [CSI Migration](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/625-csi-migration), 
  when this feature flag && the `CSIMigration` is enabled at the same time, all operations related to the 
  in-tree volume plugin `kubernetes.io/vsphere-volume` will be redirect to use the corresponding CSI driver. From a 
  user perspective, nothing will be noticed.
- InTreePluginvSphereUnregister
  - This flag technically is not part of CSI Migration design. But it happens to be related and helps with 
  CSI Migration. The name speaks for itself, when this flag is enabled, kubernetes will not register the 
  `kubernetes.io/vsphere-volume` as one of the in-tree storage plugin provisioners. This flag standalone can work out 
  of CSI Migration features.
  - However, when all `InTreePluginvSphereUnregister`, `CSIMigrationvSphere` and `CSIMigration` feature 
  flags are enabled at the same time. The kube-controller-manager will skip the feature flag checking 
  on kubelet and treat vSphere CSI migration as already complete. And directly redirect traffic to CSI 
  driver for all vsphere volume related operations.

## Design Details
[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.
### Prerequisite testing updates
No additional prerequisite tests are needed.


### Unit tests
Unit tests for vSphere CSI Migration is added in the csi-translation-lib repository
 - https://github.com/kubernetes/csi-translation-lib/blob/master/plugins/vsphere_volume_test.go (85.1% coverage)

### Integration tests
 - No Integration tests required for this feature

### e2e tests
 - Existing e2e tests developed for in-tree vSphere volume plugin executed - https://github.com/kubernetes/kubernetes/tree/master/test/e2e/storage/vsphere
 - Additional tests added in the [vSphere CSI Driver repository](https://github.com/kubernetes-sigs/vsphere-csi-driver) to ensure vSphere CSI Migration is production ready.
   - https://github.com/kubernetes-sigs/vsphere-csi-driver/blob/master/tests/e2e/vcp_to_csi_create_delete.go
   - https://github.com/kubernetes-sigs/vsphere-csi-driver/blob/master/tests/e2e/vcp_to_csi_attach_detach.go
   - https://github.com/kubernetes-sigs/vsphere-csi-driver/blob/master/tests/e2e/vcp_to_csi_full_sync.go
   - https://github.com/kubernetes-sigs/vsphere-csi-driver/blob/master/tests/e2e/vcp_to_csi_syncer.go

## Production Readiness Review Questionnaire

Please refer to the [CSI Migration Production Readiness Review Questionnaire](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/625-csi-migration#production-readiness-review-questionnaire).

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.

- 2022-01-11 KEP created

Major milestones for vSphere in-tree plugin CSI migration:

- 1.18
  - vSphere CSI migration to Alpha

- 1.19
  - vSphere CSI migration to Beta, off by default
  
- 1.25
  - vSphere CSI migration to Beta, On by default

- 1.26
  - vSphere CSI migration to Stable