# In-tree Storage Plugin to CSI Migration - GCE Design Doc

## Table of Contents

<!-- toc -->
- [Summary](#summary)
  - [New Feature Gates](#new-feature-gates)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
- [Implementation History](#implementation-history)
<!-- /toc -->


## Summary

This document present as a vendor specific KEP for the parent KEP
[CSI Migration](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/625-csi-migration)

This inherits all the contents from its parent KEP. It will introduce two new feature gates to be 
used as as described in its parent KEP. For all other contents, please refer to the parent KEP.

### New Feature Gates

- CSIMigrationGCE
  - As describe in [CSI Migration](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/625-csi-migration), 
  when this feature flag && the `CSIMigration` is enabled at the same time, all operations related to the 
  in-tree volume plugin `kubernetes.io/gce-pd` will be redirect to use the corresponding CSI driver. From a 
  user perspective, nothing will be noticed.
- InTreePluginGCEUnregister
  - This flag technically is not part of CSI Migration design. But it happens to be related and helps with 
  CSI Migration. The name speaks for itself, when this flag is enabled, kubernetes will not register the 
  `kubernetes.io/gce-pd` as one of the in-tree storage plugin provisioners. This flag standalone can work out 
  of CSI Migration features.
  - However, when all `InTreePluginGCEUnregister`, `CSIMigrationGCE` and `CSIMigration` feature 
  flags are enabled at the same time. The kube-controller-manager will skip the feature flag checking 
  on kubelet and treat GCE PD CSI migration as already complete. And directly redirect traffic to CSI 
  driver for all gce pd related operations.


## Production Readiness Review Questionnaire

Please refer to the [CSI Migration Production Readiness Review Questionnaire](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/625-csi-migration#production-readiness-review-questionnaire).

## Design Details

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

No additional tests are needed, rather the issue is orchestrating CSI driver
deployment for prow jobs. This has been complicated by the cloud provider
extraction work, which no longer permits cloud provider specific orchestration
in the k/k repository. This means that it is not possible to run any test for
gce-pd in k/k. All such tests have been moved to the [cloud-provider-gcp repo and
test infrastructure](https://github.com/kubernetes/cloud-provider-gcp).

##### Unit tests

See tests in [`k8s.io/csi-translation-lib/plugins/gce_pd_test`](https://github.com/kubernetes/csi-translation-lib/blob/master/plugins/gce_pd_test.go).

##### Integration tests

N/A

##### e2e tests

Support for tests after gce-pd migration have been [added to
cloud-provider-gcp](https://github.com/kubernetes/cloud-provider-gcp/pull/265). These
tests [have been removed from the k/k test
jobs](https://github.com/kubernetes/test-infra/pulls?q=is%3Apr+author%3Aleiyiz+gcepd).

The e2e tests are now covered in [cloud-provider-gcp](https://testgrid.k8s.io/provider-gcp-presubmits#cloud-provider-gcp-e2e-full&include-filter-by-regex=gcepd).

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.

- 2021-09-08 KEP created

Major milestones for GCE PD in-tree plugin CSI migration:

- 1.14
  - GCE PD CSI migration to Alpha

- 1.17
  - GCE PD CSI migration to Beta, off by default.

- 1.23
  - GCE PD CSI migration to Beta, on by default.

- 1.25
  - GCE PD CSI migration to GA
