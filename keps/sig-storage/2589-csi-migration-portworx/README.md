# In-tree Storage Plugin to CSI Migration - Portworx Design Doc

## Table of Contents

<!-- toc -->
- [Summary](#summary)
  - [New Feature Gates](#new-feature-gates)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
- [Implementation History](#implementation-history)
- [Design details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
<!-- /toc -->


## Summary

This document present as a vendor specific KEP for the parent KEP
[CSI Migration](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/625-csi-migration)

This inherits all the contents from its parent KEP. It will introduce two new feature gates to be 
used as described in its parent KEP. For all other contents, please refer to the parent KEP.

### New Feature Gates

- CSIMigrationPortworx
  - As describe in [CSI Migration](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/625-csi-migration), 
  when this feature flag && the `CSIMigration` is enabled at the same time, the in-tree volume 
  plugin `kubernetes.io/portworx-volume` will be redirected to use the corresponding CSI driver. From a 
  user perspective, nothing will be noticed.
- InTreePluginPortworxUnregister
  - This flag technically is not part of CSI Migration design. But it happens to be related and helps with 
  CSI Migration. The name speaks for itself, when this flag is enabled, kubernetes will not register the 
  `kubernetes.io/portworx-volume` as one of the in-tree storage plugin provisioners. This flag standalone 
  can work out of CSI Migration features.
  - However, when all `InTreePluginPortworxUnregister`, `CSIMigrationPortworx` and `CSIMigration` feature 
  flags are enabled at the same time. The kube-controller-manager will skip the feature flag checking 
  on kubelet and treat Portworx CSI migration as already complete. And directly redirect traffic to CSI 
  driver for all portworx related operations.


## Production Readiness Review Questionnaire

Please refer to the [CSI Migration Production Readiness Review Questionnaire](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/625-csi-migration#production-readiness-review-questionnaire).

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.

- 2021-09-08 KEP created

Major milestones for Portworx in-tree plugin CSI migration:

- 1.23
  - Portworx CSI migration to Alpha
- 1.25
  - Portworx CSI migration to Beta, off by default

## Design details

### Test Plan

 I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

No additional tests are needed, rather the issue is orchestrating CSI driver
deployment for prow jobs. This has been complicated by the storage provider
extraction work, which no longer permits storage provider specific orchestration
in the k/k repository. This means that it is not possible to run any test for
portworx-volume in k/k.

##### Unit tests

See tests in the [https://github.com/kubernetes/csi-translation-lib/blob/master/plugins/portworx_test.go](https://github.com/kubernetes/csi-translation-lib/blob/master/plugins/portworx_test.go)

##### Integration tests

N/A

##### e2e tests

To ensure the implementation correctness, I/we have manually run the e2e tests, [located in the main k8s repository](https://github.com/kubernetes/kubernetes/blob/master/test/e2e/storage/drivers/in_tree.go). Test results are attached to the pull requests 
