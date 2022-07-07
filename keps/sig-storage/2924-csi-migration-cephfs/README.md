# In-tree Storage Plugin to CSI Migration - Ceph Cephfs Design Doc

## Table of Contents

<!-- toc -->
- [Summary](#summary)
  - [New Feature Gates](#new-feature-gates)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
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
used as described in its parent KEP. For all other contents, please refer to the parent KEP.

### New Feature Gates

- CSIMigrationCephfs
  - As describe in [CSI Migration](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/625-csi-migration), 
  when this feature flag && the `CSIMigration` is enabled at the same time, all operations related to the 
  in-tree volume plugin `kubernetes.io/cephfs` will be redirected to use the corresponding CSI driver. From a
  user perspective, nothing will be noticed.
- InTreePluginCephFSUnregister
  - This flag technically is not part of CSI Migration design. But it happens to be related and helps with 
  CSI Migration. The name speaks for itself, when this flag is enabled, kubernetes will not register the 
  `kubernetes.io/cephfs` as one of the in-tree storage plugin provisioners. This flag standalone can work out 
  of CSI Migration features.
  - However, when all `InTreePluginCephfsUnregister`, `CSIMigrationCephfs` and `CSIMigration` feature 
  flags are enabled at the same time. The kube-controller-manager will skip the feature flag checking 
  on kubelet and treat Ceph Cephfs CSI migration as already complete. And directly redirect traffic to CSI 
  driver for all cephfs related operations.

## Design Details

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

No additional tests are needed, rather the issue is orchestrating CSI driver
deployment for prow jobs. This means that it is not possible to run any test for
cephfs in k/k repository.

##### Unit tests

+- `k8s.io/csi-translation-lib/blob/master/plugins/cephfs_test.go`: `2022-06-27` - `69.7`

##### Integration tests

N/A

##### e2e tests

Support for tests after CephFS migration will be performed by the subjected
CephFS csi driver which is available [here](https://github.com/ceph/ceph-csi/).
Addition to above, in-tree CephFS driver tests available [here](https://github.com/kubernetes/kubernetes/blob/master/test/e2e/storage/drivers/in_tree.go#L670)
also cover the e2e part of this feature..

## Production Readiness Review Questionnaire

Please refer to the [CSI Migration Production Readiness Review Questionnaire](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/625-csi-migration#production-readiness-review-questionnaire).

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.

- 2022-01-27 KEP created

Major milestones for Ceph Cephfs in-tree plugin CSI migration:

- 1.25
  - Ceph Cephfs CSI migration to Alpha
