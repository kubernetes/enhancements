# Supporting Out-of-Tree Azure Cloud Provider

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Documentation](#documentation)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [Risks and Mitigation](#risks-and-mitigation)
    - [API throttling](#api-throttling)
    - [Azure credential provider](#azure-credential-provider)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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
- [Technical Leads are members of the Kubernetes Organization](#technical-leads-are-members-of-the-kubernetes-organization)
- [Subproject Leads](#subproject-leads)
- [Meetings](#meetings)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements](https://github.com/kubernetes/enhancements/issues/667).
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes-sigs/cloud-provider-azure](https://kubernetes-sigs.github.io/cloud-provider-azure/)
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

Build support for the out-of-tree Azure cloud provider. This involves a well-tested version of the cloud-controller-manager that has feature parity to the kube-controller-manager.

## Motivation

Motivation for supporting out-of-tree providers can be found in the [Cloud Controller Manager KEP](/keps/sig-cloud-provider/20180530-cloud-controller-manager.md).
This KEP is specifically tracking progress for the Azure cloud provider.

### Goals

- Develop/test/release the Azure cloud-controller-manager
- Kubernetes clusters running on Azure should be running the cloud-controller-manager.

### Non-Goals

- Removing in-tree Azure cloud provider code, this effort falls under the [KEP for removing in-tree providers](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/20190125-removing-in-tree-providers.md).

## Proposal

We propose a set of repositories from the Kubernetes organization to host our cloud provider implementation. Since AzureFile/AzureDisk are also depending on Azure cloud provider, three new projects would be setup:

- [kubernetes/cloud-provider-azure](https://github.com/kubernetes/cloud-provider-azure) would be the main repository for Azure cloud controller manager.
- [kubernetes-sigs/azuredisk-csi-driver](https://github.com/kubernetes-sigs/azuredisk-csi-driver) would be the repository for AzureDisk CSI plugin.
- [kubernetes-sigs/azurefile-csi-driver](https://github.com/kubernetes-sigs/azurefile-csi-driver) would be the repository for AzureFile CSI plugin.

Those projects would be subprojects under [SIG Cloud Provider provider-azure](https://github.com/kubernetes/community/tree/master/sig-cloud-provider#provider-azure).

### Documentation

Example manifests, node labels/annotations, service labels/annotations and persistent volumes would be added per [Cloud Provider Documentation KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/20180731-cloud-provider-docs.md):

```
cloud-provider-azure/
└── docs
    │── example-manifests
    |   |── in-tree/
    |   |   ├── apiserver.manifest                 # an example manifest of apiserver using the in-tree integration of this cloud provider
    |   |   ├── kube-controller-manager.manifest   # an example manifest of kube-controller-manager using the in-tree integration of this cloud provider
    |   |   └── kubelet.manifest                   # an example manifest of kubelet using the in-tree integration of this cloud provider
    |   └── out-of-tree/
    |       ├── apiserver.manifest                 # an example manifest of apiserver using the out-of-tree integration of this cloud provider
    |       ├── kube-controller-manager.manifest   # an example manifest of kube-controller-manager using the out-of-tree integration of this cloud provider
    |       ├── cloud-controller-manager.manifest  # an example manifest of cloud-controller-manager using the out-of-tree integration of this cloud provider
    |       └── kubelet.manifest                   # an example manifest of kubelet using out-of-tree integration of this cloud provider
    └── resources
        |── node/
        |   ├── labels.md        # outlines what annotations that can be used on a Node resource
        |   ├── annotations.md   # outlines what annotations that can be used on a Node resource
        |   └── README.md        # outlines any other cloud provider specific details worth mentioning regarding Nodes
        |── service/
        |   ├── labels.md        # outlines what annotations that can be used on a Service resource
        |   ├── annotations.md   # outlines what annotations that can be used on a Service resource
        |   └── README.md        # outlines any other cloud provider specific details worth mentioning regarding Services
        └── persistentvolumes/
            ├── azuredisk
            |   └── README.md    # outlines CSI drivers of AzureDisk and link to CSI repository
            └── azurefile
                └── README.md    # outlines CSI drivers of AzureFile and link to CSI repository
```

### Implementation Details/Notes/Constraints

- The core of Azure cloud provider would be moved to [kubernetes-sigs/cloud-provider-azure](https://github.com/kubernetes-sigs/cloud-provider-azure).
- The storage drivers would be moved to [kubernetes-sigs/azuredisk-csi-driver](https://github.com/kubernetes-sigs/azuredisk-csi-driver) and [kubernetes-sigs/azurefile-csi-driver](https://github.com/kubernetes-sigs/azurefile-csi-driver).
- The credential provider is tracked by out-of-tree credential provider [KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/20191004-out-of-tree-credential-providers.md) and it won't block the progress of this feature.

### Risks and Mitigation

#### API throttling

Before CCM, kubelet supports getting Node information by cloud provider's instance metadata service. This includes:

- NodeName
- ProviderID
- NodeAddresses
- InstanceType
- AvailabilityZone

But with CCM, this is not possible anymore because the above functionalities have been moved to cloud controller manager.

Since API throttling is a main issue for large clusters, we have added caches for Azure resources in KCM. Those caches would also be added to CCM. But even with caches, there would still be API throttling issues on cluster provisioning stages and nodes initialization durations would be much longer than KCM because of throttling.

This issue is being tracked on [kubernetes/cloud-provider#30](https://github.com/kubernetes/cloud-provider/issues/30). Its status would be updated later when it's discussed through sig cloud-provider.

#### Azure credential provider

Azure credential provider is also depending on cloud provider codes. Though Azure Managed Service Identity (MSI) is a way to avoid explicit setting of credentials, MSI is not available on all cases (e.g. MSI may not be authorized to specific ACR repository).

This issue is being tracked on KEP [Support Instance Metadata Service with Cloud Controller Manager](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/azure/20190722-ccm-instance-metadata.md). It has been marked as implementable and would be implemented in cloud-provider-azure.

## Design Details

### Test Plan

Azure Cloud Controller provider is reporting conformance test results to TestGrid as per the [Reporting Conformance Test Results to Testgrid KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/0018-testgrid-conformance-e2e.md).

See [report](https://testgrid.k8s.io/provider-azure-cloud-provider-azure) for more details.

### Graduation Criteria

- Azure cloud controller manager is moving to GA
  - Feature compatible with KCM
  - Conformance tests are passed and published to [testgrid](https://testgrid.k8s.io/provider-azure-cloud-provider-azure)
- CSI drivers for AzureDisk/AzureFile are moving to GA
  - Feature compatible with KCM
  - Features implemented from CSI API SPEC
  - Conformance tests are passed and published to [testgrid](https://testgrid.k8s.io/provider-azure-azuredisk-csi-driver)
- Azure credential provider is still supported in Kubelet
  - Feature compatible with KCM
  - Features implemented from CSI API SPEC
  - Conformance tests are passed and published to [testgrid](https://testgrid.k8s.io/provider-azure-cloud-provider-azure)

#### Alpha -> Beta Graduation

- E2E tests have been added in [testgrid](https://testgrid.k8s.io/provider-azure-cloud-provider-azure)
- The same set of tests have been passed with out-of-tree projects
- All the features from in-tree implementations are still supported

#### Beta -> GA Graduation

- Code changes are decoupled from in-tree cloud provide (e.g. it shouldn't vendor in-tree implementations directly)
- E2E tests have been run stably (e.g. no flaky tests)
- Upgrade tests and scalability tests have been passed

### Upgrade / Downgrade Strategy

Upgrade/Downgrade Azure cloud controller manager and CSI drivers together with other master components. The versions for Azure cloud controller manager and CSI drivers should be chosen according to the version skew strategy below.

### Version Skew Strategy

For each Kubernetes minor releases (e.g. v1.15.x), dedicated Azure cloud controller manager would be released. For CSI drivers, however, they would be released based on [CSI specification versions](https://kubernetes-csi.github.io/docs/).

- The version matrix for Azure cloud controller manager would be documented on [kubernetes/cloud-provider-azure](https://github.com/kubernetes/cloud-provider-azure/blob/master/README.md#current-status).
- The version matrix for CSI drivers would be documented on [kubernetes-sigs/azuredisk-csi-driver](https://github.com/kubernetes-sigs/azuredisk-csi-driver#container-images--csi-compatibility) and [kubernetes-sigs/azurefile-csi-driver](https://github.com/kubernetes-sigs/azurefile-csi-driver#container-images--csi-compatibility).

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: CSIMigrationAzureDisk and CSIMigrationAzureFile
    - Components depending on the feature gate: kube-controller-manager and kubelet
  - [x] Other
    - Describe the mechanism: deploy cloud-controller-manager, cloud-node-manager and CSI drivers in the cluster.
    - Will enabling / disabling the feature require downtime of the control
      plane? `--cloud-provider=external` should be set for kube-controller-manager.
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? --cloud-provider=external` should be set for for kubelet.

* **Does enabling the feature change any default behavior?**

  The default behaviors are still same as before.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**

  Yes. Delete the cloud-controller-manager and cloud-node-manager, then change the `--cloud-provider`
  option back to `azure` would still work. CSI drivers should be kept to ensure CSI-provisioned PVCs are still working.

* **What happens if we reenable the feature if it was previously rolled back?**

  It would still work as expected.

* **Are there any tests for feature enablement/disablement?**

  E2E tests have already been added and results are published on testgrid.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**

  Wrong component configurations may cause rollout fail, and running workloads won't be impacted.

* **What specific metrics should inform a rollback?**

  Couldn't create a LoadBalancer typed service or AzureDisk PVC indicate the rollout needs to rollback.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

  Manually changing the `--cloud-provider` options have been verified. For upgrade->downgrade,
  the volumes provisioned by CSI drivers should continue to be managed by CSI drivers. They're
  not able to migrate to in-tree drivers.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**

  In-tree AzureDisk/AzureFile drivers would be migrated to CSI drivers automatically.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**

  Operation specific metrics (e.g. LoadBalancer creation and route table update) have been added.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
  - [x] Metrics
    - Metric names:
      - cloudprovider_azure_op_duration_seconds
      - cloudprovider_azure_api_request_errors
      - cloudprovider_azure_api_request_throttled_count
      - cloudprovider_azure_op_duration_seconds_bucket
    - Components exposing the metric: cloud-controller-manager and CSI drivers

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

  - 99.5% of read and write ARM requests in the last 5 minutes were successful
  - LoadBalancer service requests in the last 5 minutes are served in 60 seconds @99th percentile
  - Routes for new nodes in the last 5 minutes are served in 90 seconds @99th percentile
  - Disk PVC attach requests in the last 5 minutes are served in 60 seconds @99th percentile

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**

  CSI drivers for AzureDisk/AzureFile are required for out-of-tree cloud provider, 
  and their plans has already been added in above designs.

### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**

  Yes, CSI drivers for AzureDisk/AzureFile would be introduced.

* **Will enabling / using this feature result in introducing new API types?**

  Yes, CSI drivers AzureDisk/AzureFile would be introduced.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

  Same as before.

* **What are other known failure modes?**

  Refer <https://kubernetes-sigs.github.io/cloud-provider-azure/faq>.

* **What steps should be taken if SLOs are not being met to determine the problem?**

  Check the debug logs of cloud-provider-azure since detailed steps are logged in debug level.

## Implementation History

See [kubernetes/cloud-provider-azure#pulls](https://github.com/kubernetes/cloud-provider-azure/pulls?utf8=%E2%9C%93&q=+is%3Apr+), [kubernetes-sigs/azuredisk-csi-driver#pulls](https://github.com/kubernetes-sigs/azuredisk-csi-driver/pulls?utf8=%E2%9C%93&q=is%3Apr++) and [kubernetes-sigs/azurefile-csi-driver#pulls](https://github.com/kubernetes-sigs/azurefile-csi-driver/pulls?utf8=%E2%9C%93&q=is%3Apr++).

## Technical Leads are members of the Kubernetes Organization

The Leads run operations and processes governing this subproject.

- @khenidak
- @feiskyer

## Subproject Leads

- @dstrebel
- @justaugustus

## Meetings

Sig-Azure meetings is expected to have biweekly. SIG Cloud Provider will provide zoom/youtube channels as required. We will have our first meeting after repo has been settled.

Meeting Time: Wednesdays at 09:00 PT (Pacific Time) (biweekly). [Convert to your timezone](http://www.thetimezoneconverter.com/?t=20:00&tz=PT%20%28Pacific%20Time%29).

- [Meeting notes and Agenda](https://docs.google.com/document/d/1SpxvmOgHDhnA72Z0lbhBffrfe9inQxZkU9xqlafOW9k/edit).
- [Meeting recordings](https://www.youtube.com/watch?v=yQLeUKi_dwg&list=PL69nYSiGNLP2JNdHwB8GxRs2mikK7zyc4).
