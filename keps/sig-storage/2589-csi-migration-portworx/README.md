# KEP-2589: In-tree Storage Plugin to CSI Migration - Portworx Design Doc


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
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist


Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [x] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [x] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This document present as a vendor specific KEP for the parent KEP
[CSI Migration](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/625-csi-migration)

This inherits all the contents from its parent KEP. It will introduce two new feature gates to be 
used as described in its parent KEP. For all other contents, please refer to the parent KEP.

## Motivation

Currently the Portworx volume provisioning happens through Portworx in-tree driver. As part of the parent KEP [CSI Migration](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/625-csi-migration), the Portworx in-tree driver logic needs to be "migrated" to use Portworx CSI driver instead.

### Goals

 - To migrate Portworx in-tree plugin to CSI


### Non-Goals

  - This doesn't target the core in-tree to CSI migration code in k/k.

## Proposal

The in-tree to CSI migration feature is already in place in k/k. We just need to enable Portworx specific feature gates for it to work for Portworx driver.


### Risks and Mitigations

 - Portworx CSI driver needs to be already deployed before enabling this feature.

## Design Details

The in-tree to CSI migration feature is already in place in k/k. We just need to enable vendor specific feature gates for it to work for each vendor. Below are the feature gates we need to enable:

- CSIMigrationPortworx
  - As describe in [CSI Migration](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/625-csi-migration), 
  when this feature flag && the `CSIMigration` is enabled at the same time, the in-tree volume 
  plugin `kubernetes.io/portworx-volume` will be redirected to use the corresponding CSI driver. From a user perspective, nothing will be noticed.
- InTreePluginPortworxUnregister
  - This flag technically is not part of CSI Migration design. But it happens to be related and helps with 
  CSI Migration. The name speaks for itself, when this flag is enabled, kubernetes will not register the 
  `kubernetes.io/portworx-volume` as one of the in-tree storage plugin provisioners. This flag standalone 
  can work out of CSI Migration features.
  - However, when all `InTreePluginPortworxUnregister`, `CSIMigrationPortworx` and `CSIMigration` feature 
  flags are enabled at the same time. The kube-controller-manager will skip the feature flag checking 
  on kubelet and treat Portworx CSI migration as already complete. And directly redirect traffic to CSI driver for all portworx related operations.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

No additional tests are needed, rather the issue is orchestrating CSI driver
deployment for prow jobs. This has been complicated by the storage provider
extraction work, which no longer permits storage provider specific orchestration
in the k/k repository. This means that it is not possible to run any test for
portworx-volume in k/k.

##### Unit tests

- `k8s.io/csi-translation-lib/plugins/portworx.go`: `6-20` - `80.6`

##### Integration tests

N/A

##### e2e tests

- `sig-storage` `Driver: portworx-volume` To ensure the implementation correctness, I/we have manually run the e2e tests, [located in the main k8s repository](https://github.com/kubernetes/kubernetes/blob/master/test/e2e/storage/drivers/in_tree.go). Test results are attached to the pull requests 


### Graduation Criteria

* Alpha in 1.23 provided all tests are passing.
* All functionality is guarded by alpha `CSIMigrationPortworx` feature gate.
* Portworx CSI migration to Beta, off by default in 1.25. e2e test results are provided in the PR.
* Beta in 1.31 with design validated by customer deployments
  (non-production)
* Manual testing with in-tree Portworx volumes should be passing.
* GA in 1.33, with `CSIMigrationPortworx` feature gate graduating to GA.


### Upgrade / Downgrade Strategy

When `CSIMigrationPortworx` feature gate gets enabled and customers are not using Portworx security feature, the upgrade/downgrade will work without any changes to cluster objects or configurations.
In case of downgrade, it will revert back to the existing behavior of using in-tree driver.

With Portworx security feature enabled, customers will have to add certain annotations to in-tree PVs mentioning the CSI secret name/namespace which the kubelet or CSI sidecar containers can use(using `csi-translation-lib`) to pass secret contents to Portworx CSI driver for operations on in-tree PVs. The annotations to be added will be documented in Portworx documentation.
The downgrade will work without any changes.

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire
Please refer to the [CSI Migration Production Readiness Review Questionnaire](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/625-csi-migration#production-readiness-review-questionnaire).

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: CSIMigrationPortworx
  - Components depending on the feature gate: kubelet, A/D controller

###### Does enabling the feature change any default behavior?

It will switch the control plane volume operations from in-tree driver to CSI driver.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate will revert back to the existing behavior of using in-tree driver.

###### What happens if we reenable the feature if it was previously rolled back?
If reenabled, any subsequent CSI operations will go through Portworx CSI driver.

###### Are there any tests for feature enablement/disablement?
  We will need to create unit tests that enable this feature.


### Rollout, Upgrade and Rollback Planning


###### How can a rollout or rollback fail? Can it impact already running workloads?
  No, a rollout should not impact running workloads, since the default behavior
  remains the same to use in-tree driver.


###### What specific metrics should inform a rollback?
  No known rollback criteria.


###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

The upgrade->downgrade->upgrade path should work fine as it's already handled in the parent KEP [CSI Migration](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/625-csi-migration)

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

It deprecates the use of in-tree Portworx driver.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?
Check the `migrated-plugins` annotation on `CSINode` object, which will have the list of plugins for which in-tree to CSI migration feature is turned on.

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as last resort)
  - Details:
  The PV object will have a `migrated-to` annotation on it.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?
- No increased failure rates during mounting a volume created using in-tree driver.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Other (treat as last resort)
  - Details:
  We can use the SLIs for parent KEP [CSI Migration](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/625-csi-migration), if any.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No additional metrics needed.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

It has a pre-requisite of Portworx CSI driver to be already deployed in the cluster.

### Scalability

###### Will enabling / using this feature result in any new API calls?

There will be no new API calls.

###### Will enabling / using this feature result in introducing new API types?

There are no new API types.


###### Will enabling / using this feature result in any new calls to the cloud provider?

There should be no new calls to the cloud providers.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

There will be no increase in size or count of existing API objects.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?
The CSI operations like mounting a volume will fail.

###### What are other known failure modes?
 - Volume mount failing for in-tree PVs when Portworx security feature is enabled.
  - Detection: The pods using in-tree PVs will not be in running state, and the error would mention that the authorization token is missing.
  - Mitigations: Customers will have to add certain annotations to in-tree PVs mentioning the CSI secret name/namespace which the kubelet or CSI sidecar containers can use(using `csi-translation-lib`) to pass secret contents to Portworx CSI driver for operations on in-tree PVs. 
  Please note that adding these annotations for such PVs will be automatically done by Portworx platform upon upgrade, and it will also be documented in the Portworx documentation.
  - Diagnostics: The pod using the in-tree PV will not be in running state, and the error would mention that the authorization token is missing.
  - Testing: Manual testing has been done with in-tree Portworx volumes migrating to CSI.

###### What steps should be taken if SLOs are not being met to determine the problem?

We need to check kubelet logs to determine any failures in Kubernetes while mounting volumes. Additionally, we can also check Portworx logs if there is any error originating from Portworx side.

## Implementation History

- 2021-09-08 KEP created

Major milestones for Portworx in-tree plugin CSI migration:

- 1.23
  - Portworx CSI migration to Alpha
- 1.25
  - Portworx CSI migration to Beta, off by default
- 1.31
  - Portworx CSI migration to Beta, on by default
- 1.33
  - Portworx CSI migration to Stable
- 1.36
  - Cleanup Portworx CSI migration feature gates

## Drawbacks

N/A

## Alternatives

N/A

