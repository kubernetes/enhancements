# In-tree Storage Plugin to CSI Migration Design Doc


## Table of Contents

<!-- toc -->
- [Summary](#summary)
  - [Glossary](#glossary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
  - [Alpha -&gt; Beta](#alpha---beta)
  - [Beta -&gt; GA](#beta---ga)
- [Test Plan](#test-plan)
  - [Per-driver migration testing](#per-driver-migration-testing)
  - [Upgrade/Downgrade/Skew Testing](#upgradedowngradeskew-testing)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

This document presents a detailed design for migrating in-tree storage plugins
to CSI. This will be an opt-in feature turned on at cluster creation time that
will redirect in-tree plugin operations to a corresponding CSI Driver.

### Glossary

* ADC (Attach Detach Controller): Controller binary that handles Attach and Detach portion of a volume lifecycle
* Kubelet: Kubernetes component that runs on each node, it handles the Mounting and Unmounting portion of volume lifecycle
* CSI (Container Storage Interface): An RPC interface that Kubernetes uses to interface with arbitrary 3rd party storage drivers
* In-tree: Code that is compiled into native Kubernetes binaries
* Out-of-tree: Code that is not compiled into Kubernetes binaries, but can be run as Deployments on Kubernetes


## Motivation

The Kubernetes volume plugins are currently in-tree meaning all logic and
handling for each plugin lives in the Kubernetes codebase itself. With the
Container Storage Interface (CSI) the goal is to move those plugins out-of-tree.
CSI defines a standard interface for communication between the Container
Orchestrator (CO), Kubernetes in our case, and the storage plugins.

As the CSI Spec moves towards GA and more storage plugins are being created and
becoming production ready, we will want to migrate our in-tree plugin logic to
use CSI plugins instead. This is motivated by the fact that we are currently
supporting two versions of each plugin (one in-tree and one CSI), and that we
want to eventually transition all storage users to CSI.

In order to do this we need to migrate the internals of the in-tree plugins to
call out to CSI Plugins because we will be unable to deprecate the current
internal plugin API’s due to Kubernetes API deprecation policies. This will
lower cost of development as we only have to maintain one version of each
plugin, as well as ease the transition to CSI when we are able to deprecate the
internal APIs.

### Goals

* Compile all requirements for a successful transition of the in-tree plugins to
  CSI
    * As little code as possible remains in the Kubernetes Repo
    * In-tree plugin API is untouched, user Pods and PVs continue working after
      upgrades
    * Minimize user visible changes
* Design a robust mechanism for redirecting in-tree plugin usage to appropriate
  CSI drivers, while supporting seamless upgrade and downgrade between new
  Kubernetes version that uses CSI drivers for in-tree volume plugins to an old
  Kubernetes version that uses old-fashioned volume plugins without CSI.
* Design framework for migration that allows for easy interface extension by
  in-tree plugin authors to “migrate” their plugins.
    * Migration must be modular so that each plugin can have migration turned on
      and off separately

### Non-Goals

* Design a mechanism for deploying  CSI drivers on all systems so that users can
  use the current storage system the same way they do today without having to do
  extra set up.
* Implementing CSI Drivers for existing plugins
* Define set of volume plugins that should be migrated to CSI

## Proposal

### Implementation Details/Notes/Constraints
The detailed design was originally implemented as a [design proposal](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/csi-migration.md)

### Risks and Mitigations

* Performance risks as outlined in design proposal
* ADC and Kubelet synchronization fairly complicated, upgrade path non-trivial - mitigation discussed in design proposal

## Graduation Criteria

### Alpha -> Beta

* All volume operation paths covered by Migration Shim in Alpha for >= 1 quarter
* Tests outlined in design proposal implemented
* Required CRD and driver installation solved generally

### Beta -> GA

* All volume operation paths covered by Migration Shim in Beta for >= 1 quarter without significant issues

## Test Plan

### Per-driver migration testing

We will require *each* plugin/driver provider to set up public CI to run all
existing in-tree plugin driver tests for their migrated driver. The CI should
include all tests for the in-tree driver with a focus on tests labeled `In-tree
Volumes [driver: {inTreePluginName}]` with a cluster that has CSI migration
enabled with feature flags. The driver authors will be expected to prove (using
the tests) that the driver can handle anything the in-tree plugin can including,
but not limited to: dynamic provisioning, pre-provisioned volumes, inline
volumes, resizing, multivolumes, subpath, volume reconstruction. The onus is on
the storage provider to use appropriate infrastructure to run these tests.

If migration is on for that plugin, the test framework will inspect
kube-controller-manager and kubelet metrics to make sure that the CSI driver is
servicing the operations. This enables the test suite to programatically confirm
migration status. The framework must also observe through metrics that none of
the in-tree code is being called.

The above is done by checking that no in-tree plugin code is emitting metrics
when migration is on. We will also confirm that metrics are being emitted in
general by confirming the existence of an indicator metric.

Passing these tests in Public CI is the main graduation criterea for the
`CSIMigration{provider}` flag to Beta.

### Upgrade/Downgrade/Skew Testing

The Kubernetes community will have test clusters brought up that have different
feature flags enabled on different components (ADC and Kubelet). Once these
feature flag skew configurations are brought up the test itself would have to
know what configuration it’s running in and validate the expected result.

Configurations to test:

| ADC               | Kubelet                                            | Expected Result                                                          |
|-------------------|----------------------------------------------------|--------------------------------------------------------------------------|
| ADC Migration On  | Kubelet Migration On                               | Fully migrated - result should be same as “Migration Shim Testing” above |
| ADC Migration On  | Kubelet Migration Off (or Kubelet version too low) | No calls made to driver. All operations serviced by in-tree plugin       |
| ADC Migration Off | Kubelet Migration On                               | Not supported config - Undefined behavior                                |
| ADC Migration Off | Kubelet Migration Off                              | No calls made to driver. All operations service by in-tree plugin        |

Additionally, the community will craft a test where a cluster should be able to
run through all plugin tests, do a complete upgrade to a version with CSI
Migration turned on, then run through all the plugin tests again and verify that
there is no issue.

Running this set of tests is optional for a per-provider basis. We would
recommend it for providing extra confidence but the framework for
upgrade/downgrade is provider agnostic.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.

-->

### Feature Enablement and Rollback


* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: CSIMigration, CSIMigration{cloud-provider}
    - Components depending on the feature gate: kubelet, kube-controller-manager, kube-scheduler
    - Please refer to this design doc on the [Step to enable the feature](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/csi-migration.md#upgradedowngrade-migrateunmigrate-scenarios)

* **Does enabling the feature change any default behavior?**
  Yes and No. If only CSIMigration feature flag is enabled, nothing will change on the cluster behavior. However, if CSIMigration && CSIMigration{cloud-provider} are both enabled, the behavior will change. The in-tree volume plugin that the cloud-provider use will be redirect to use the corresponding CSI driver. But from a user perspective, nothing will be noticed.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes - can be disabled by disabling feature flags. 
  Please refer to the [upgrade/downgrade](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/csi-migration.md#upgradedowngrade-migrateunmigrate-scenarios) sections on how to downgrade the cluster to roll back the enablement.

* **What happens if we reenable the feature if it was previously rolled back?**
The CSI migration feature will start to work again. The out-of-tree CSI driver will start to work instead of in-tree plugin again.

* **Are there any tests for feature enablement/disablement?**
We have CSI Migration e2e test for each plugin that are implemented and maintained by each driver maintainer. 
Specifically, for each in-tree plugin corresponding CSI drivers, it will have 
  - Full k8s storage e2e tests
  - Migration enabled functional e2e tests. 
  - Upgrade/downgrade/version skew tests that test the transition from feature turning on to off.

  For core K8s, we have unit tests including but not limited to:
   - `pkg/volume/csimigration/plugin_manager_test.go`
   - All unit tests in the csi-translation-lib `staging/src/k8s.io/csi-translation-lib/translate_test.go`
   - Controller test with Migration on CSI sidecars: external-provisioner, external-resizer
     - provisioner: pkg/controller/controller_test.go#TestProvisionWithMigration
     - resizer: pkg/resizer/csi_resizer_test.go#TestResizeMigratedPV
  
  We also have [upgrade tests](https://github.com/kubernetes/kubernetes/tree/master/test/e2e/upgrades/storage) for storage in k8s. The test can be used to create a PVC before migration enabled continues to function after upgrade. We will enhance this
  test to add more feature coverage if needed.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**  
  - The rollout can fail if the ordering of CSIMigration{cloud-provider} flag was wrongly enabled on kubelet and kube-controller-manager. Specifically, if on the node side kubelet enables the flag and control-plane side the flag is not enabled, then the volume will not be able to be mounted successfully. 
    - For workloads that running on nodes have not enable CSI migration, those pods will not be impacted. 
    - For any pod that is being deleted by node drain before turning on migration and created on new node that has CSI migration turned on, the volume mount will fail and pod will not come up correctly.
  - Additionally, CSI Migration has a strong dependency on CSI drivers. So if the in-tree corresponding CSI driver is not properly installed, any volume related operation could fail.
  - If feature parity is not guaranteed or if any bug exists in the CSI driver/csi-translation-lib, the rollout could fail because pod using the PV could fail to execute provision/delete/attach/detach/mount/unmount/resize operations depend on the bug itself.

* **What specific metrics should inform a rollback?**
  We have metrics on the CSI sidecar side called `csi_operation_duration_seconds` and core k8s metrics on both kube-controller-manager and kubelet side called `storage_operation_duration_seconds`. 
  Both of them will have a `migrated` field to indicate whether this operation is a migrated PV operation. 
    - For `csi_operation_duration_seconds`, we will have a `grpc_status` field
    - For `storage_operation_duration_seconds`, we will have a `status` field
  
  If the error ratio of these two metrics has an unusual strike or is keeping at a relatively higher level compared to in-tree model, it means something went wrong and we need a rollback.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
To turn it on by default in Beta, we require each in-tree plugin to at least manually test the upgrade->downgrade->upgrade path.
For GA, we require such test exists in each driver's test CI.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**
There will not be API removal in CSI migration itself. But eventually when CSI migration is all finished. We will plan to remove all in-tree plugins.
So we will have in-tree plugin deprecated when CSIMigration{cloud-provider} goes to beta. And code removal will be required eventually.
In addition, some CSI drivers are not able to maintain 100% backwards compatibility, so those drivers need to deprecate certain behaviors. 
- vSphere [kubernetes#98546](https://github.com/kubernetes/kubernetes/pull/98546).
- Azure drivers links TBD.
- Other providers no deprecations are known.


### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?**
  We will have metrics `csi_sidecar_duration_seconds` on the CSI sidecars and `storage_operation_duration_seconds` on the kube-controller-manager and kubelet side to indicate whether this operation is a migrated operation or not. These metrics will have a `migrated` field to indicate if this is a migrated operation.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
  - [x] Metrics
    - Metric name: csi_sidecar_duration_seconds && storage_operation_duration_seconds, these metrics will have a `migrated` field
    - [Optional] Aggregation method:
    - Components exposing the metric: CSI sidecars, kubelet, kube-controller-manager
  - [x] Other (treat as last resort)
    - Details: Pod using PVC that is provisioned by tge in-tree plugin storageclass has failure.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  - SLO with migration on matches the existing plugin's in-tree SLO with offset less than 1%

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**
Node side CSI operation metrics. It will be implemented in the GA phase.

### Dependencies


* **Does this feature depend on any specific services running in the cluster?**

  - Corresponding CSI Driver for in-tree CSI migration enabled plugin
    - Usage description:
      - Impact of its outage on the feature: in-tree plugin stops working without the CSI Driver properly setup
      - Impact of its degraded performance or high-error rates on the feature: Error or performance decrease will be reflected for volume intensive operation


### Scalability

* **Will enabling / using this feature result in any new API calls?**
  Yes. If the CSI driver has already been installed before turning on the CSI migration, the informer related API calls will not be counted as new API calls. If not, the following new calls can be added:
  - For volume attach/detach:
    VolumeAttachment CREATE/DELETE APIs will be called for volume attachment/detachment by kube-controller-manager. VolumeAttachment PATCH API will be called for volume attachment by csi-attacher. One API call for each volume per operation needed.
    VolumeAttachment LIST/WATCH api will be called by csi-attacher to monitor the VolumeAttachment.
  - For volume provision/delete:
    PVC LIST/WATCH apis will be called by csi-provisioner to monitor the PVC status. PV CREATE api will be called by csi-provisioner to create PV. PV DELETE api will be called by csi-provisioner to delete PV. PVC/PV PATCH api will be called by csi-provisioner for updating the object.
    Notice that these new calls from csi-provisioner also mean that we will reduce call from the kube-controller-manager side.
  - When CSI driver is being installed, the deployer will call CSIDriver CREATE api for the object creation. There will also be CSINode PATCH call by kubelet. For each kubelet that installs the driver there will be one PATCH call.
  - csi-provisioner && csi-attacher will call LIST/WATCH api for monitoring CSINode object when provision/attach volume

* **Will enabling / using this feature result in introducing new API types?**
  No

* **Will enabling / using this feature result in any new calls to the cloud provider?**
  After switching to CSI driver model, all the volume operations including volume provision/deletion/attach/detach/mount/unmount/resize will be running through the CSI driver. 
  So depending on how the CSI driver is designed and implemented, it could vary if there is any new calls being added. 
  For example, `gce-pd` driver has the in-tree and CSI version of plugin implementation for all the operation mentioned above, once we switch from in-tree to CSI by CSI migration. 
  If the implementation is the same, then there will not be new calls to the cloud provider. 
  However, it is also possible that the plugin maintainer has different implementation so there might be new calls.

* **Will enabling / using this feature result in increasing size or count of the existing API objects?**
  General objects that are being used by CSI regardless of migration:
  - CSI migration will require CSI driver to be installed in the cluster so it can add CSI related API objects including CSIDriver, CSINode, VolumeAttachment.
  - The existing Node object will include new labels, specifically the CSI topology that are introduced by the CSI driver, e.g. `topology.gke.io/zone=us-central1-b` for GCE PD CSI Driver.
  - PV object will have new annotation `volume.beta.kubernetes.io/storage-provisioner`
  
  CSI migration specific fields:
  - The size of PV will increase with the new annotation `volume.beta.kubernetes.io/migrated-to`.
  - For existing in-line volumes, there will be a new field under `VolumeAttachment.Spec.Source.VolumeAttachmentSource.InlineVolumeSpec` that will be populated if in-line volumes of migrated in-tree plugin is used.

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  Depending on the design and implementation of the CSI Driver, the operation time taken could vary. 
  In general, it might increase the total time spend because for the CSI sidecar to detect the object in the APIServer and do corresponding change through the unix domain socket might add additional traffic compared to the in-tree plugin model.

  The unix domain socket is the mechanism that kubelet use to communicate with CSI drivers.

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  It should not increasing the resource usage in a significant manner. But each CSI driver deployed on the node could take more CPU and RAM depending on the implementation.

### Troubleshooting


* **How does this feature react if the API server and/or etcd is unavailable?**

CSI sidecars will not be able to monitor the status change of the API object. So all volume related operation will fail. The existing running container should not be impacted.
When the feature is not enabled, only provision/deletion/resize should fail.

* **What are other known failure modes?**
  For each of them, fill in the following information by copying the below template:
  - Bug in CSI driver or translation library.
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
      Operators should be able to identify issues regarding migrated PV when the two metrics `csi_sidecar_operation_seconds` and `storage_operation_duration_seconds` showed error ratio spike.
    - Mitigation: What can be done to stop the bleeding, especially for already running user workloads?
      Already running workload should not be impacted except when there is pod movement. To stop the bleeding, turn off the feature gate and bring the pod back to the node without CSI migration.
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Each CSI sidecars and drivers will have their own logging to help debug.
      On the kubelet side, kubelet also contains error messages returned by the CSI drivers call.
      PVC and PV events also show the error messages.
    - Testing: Are there any tests for failure mode? If not, describe why.
      We do not have specific tests for failure mode. Each driver shall have upgrade/downgrade/version skew tests that can verify the migration is working properly.

* **What steps should be taken if SLOs are not being met to determine the problem?**
  - Take the CSI driver log, kube-controller-manager log and kubelet log to analyze why the SLOs are not being met. What is the most error status and why is it error.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.

- 2021-02-04 KEP updated with Production Readiness Review Questionnaire
- 2019-01-29 KEP Created
- 2019-01-05 Implementation started

Major milestones for each in-tree plugin CSI migration:

- 1.21
  - Azurefile CSI migration to Beta
- 1.19
  - vSphere CSI migration to Beta
  - Azuredisk CSI migration to Beta
- 1.17
  - GCE PD CSI migration to Beta
  - AWS EBS CSI migration to Beta
- 1.15
  - Azuredisk CSI migration to Alpha
  - Azurefile CSI migration to Alpha
- 1.14
  - GCE PD CSI migration to Alpha
  - AWS EBS CSI migration to Alpha
