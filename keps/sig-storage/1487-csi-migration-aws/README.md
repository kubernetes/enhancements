# In-tree Storage Plugin to CSI Migration - AWS EBS Design Doc

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
used as as described in its parent KEP. For all other contents, please refer to the parent KEP.

### New Feature Gates

- CSIMigrationAWS
  - As describe in [CSI Migration](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/625-csi-migration), 
  when this feature flag && the `CSIMigration` is enabled at the same time, the in-tree volume 
  plugin `kubernetes.io/aws-ebs` will be redirect to use the corresponding CSI driver. From a 
  user perspective, nothing will be noticed.
- InTreePluginAWSUnregister
  - This flag technically is not part of CSI Migration design. But it happens to be related and helps with 
  CSI Migration. The name speaks for itself, when this flag is enabled, kubernetes will not register the 
  `kubernetes.io/aws-ebs` as one of the in-tree storage plugin provisioners. This flag standalone 
  can work out of CSI Migration features.
  - However, when all `InTreePluginAWSUnregister`, `CSIMigrationAWS` and `CSIMigration` feature 
  flags are enabled at the same time. The kube-controller-manager will skip the feature flag checking 
  on kubelet and treat AWS EBS CSI migration as already complete. And directly redirect traffic to CSI 
  driver for all aws ebs related operations.

## Design Details

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

No prerequisite tests are needed.

##### Unit tests

See https://github.com/kubernetes/csi-translation-lib/blob/master/plugins/aws_ebs_test.go.

- `k8s.io/csi-translation-lib/plugins/aws_ebs.go`: `2022-06-17` - `52.3`

##### Integration tests

N/A

##### e2e tests

The [Kubernetes storage e2e
tests](https://github.com/kubernetes/kubernetes/tree/master/test/e2e/storage/external)
are run by an
[aws-ebs-csi-driver](https://github.com/kubernetes-sigs/aws-ebs-csi-driver/tree/master/tests/e2e-kubernetes)
job: https://testgrid.k8s.io/provider-aws-ebs-csi-driver#pull-migration-test.
The tests create in-tree volume plugin objects like StorageClasses with
`provisioner: kubernetes.io/aws-ebs` and validate using metrics that all volume
operations went to the CSI driver.

## Production Readiness Review Questionnaire

Please refer to the [CSI Migration Production Readiness Review Questionnaire](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/625-csi-migration#production-readiness-review-questionnaire).

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.

- 2022-01-08 KEP created

Major milestones for AWS EBS in-tree plugin CSI migration:

- 1.14
  - AWS EBS CSI migration to Alpha

- 1.17
  - AWS EBS CSI migration to Beta, off by default.

- 1.23
  - AWS EBS CSI migration to Beta, on by default.

- 1.25
  - AWS EBS CSI migration to Stable
