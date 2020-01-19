---
title: Support Out-of-Tree Azure Cloud Provider
authors:
  - "@andrewsykim"
  - "@dstrebel"
  - "@feiskyer"
owning-sig: sig-cloud-provider
reviewers:
  - "@dstrebel"
  - "@justaugustus"
  - "@khenidak"
  - "@feiskyer"
approvers:
  - "@feiskyer"
  - "@khenidak"
  - "@hogepodge"
  - "@jagosan"
editor: "@feiskyer"
creation-date: 2019-01-29
last-updated: 2019-05-06
status: provisional
---

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
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Technical Leads are members of the Kubernetes Organization](#technical-leads-are-members-of-the-kubernetes-organization)
- [Subproject Leads](#subproject-leads)
- [Meetings](#meetings)
<!-- /toc -->

## Release Signoff Checklist

- [X] k/enhancements issue in release milestone and linked to KEP (https://github.com/kubernetes/enhancements/issues/667)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

Build support for the out-of-tree Azure cloud provider. This involves a well-tested version of the cloud-controller-manager
that has feature parity to the kube-controller-manager.

## Motivation

Motivation for supporting out-of-tree providers can be found in the [Cloud Controller Manager KEP](/keps/sig-cloud-provider/20180530-cloud-controller-manager.md).
This KEP is specifically tracking progress for the Azure cloud provider.

### Goals

- Develop/test/release the Azure cloud-controller-manager
- Kubernetes clusters running on Azure should be running the cloud-controller-manager.

### Non-Goals

- Removing in-tree Azure cloud provider code, this effort falls under the [KEP for removing in-tree providers](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/2019-01-25-removing-in-tree-providers.md).

## Proposal

We propose a set of repositories from the Kubernetes organization to host our cloud provider implementation. Since AzureFile/AzureDisk are also depending on Azure cloud provider, three new projects would be setup:

- [kubernetes/cloud-provider-azure](https://github.com/kubernetes/cloud-provider-azure) would be the main repository for Azure cloud controller manager.
- [kubernetes-sigs/azuredisk-csi-driver](https://github.com/kubernetes-sigs/azuredisk-csi-driver) would be the repository for AzureDisk CSI plugin.
- [kubernetes-sigs/azurefile-csi-driver](https://github.com/kubernetes-sigs/azurefile-csi-driver) would be the repository for AzureFile CSI plugin.

Those projects would be subprojects under [SIG Azure](https://github.com/kubernetes/community/tree/master/sig-azure#subprojects).

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

- The core of Azure cloud provider would be moved to [kubernetes/cloud-provider-azure](https://github.com/kubernetes/cloud-provider-azure).
- The storage drivers would be moved to [kubernetes-sigs/azuredisk-csi-driver](https://github.com/kubernetes-sigs/azuredisk-csi-driver) and [kubernetes-sigs/azurefile-csi-driver](https://github.com/kubernetes-sigs/azurefile-csi-driver).
- The credential provider is still under discussion on [kubernetes/cloud-provider#13](https://github.com/kubernetes/cloud-provider/issues/13).

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

This issue is being tracked on [kubernetes/cloud-provider#13](https://github.com/kubernetes/cloud-provider/issues/13). Its status would be updated later when it's discussed through sig cloud-provider.

## Design Details

### Test Plan

Azure Cloud Controller provider is reporting conformance test results to TestGrid as per the [Reporting Conformance Test Results to Testgrid KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/0018-testgrid-conformance-e2e.md).

See [report](https://testgrid.k8s.io/sig-azure-master#Summary) for more details.

### Graduation Criteria

- Azure cloud controller manager is moving to GA
  - Feature compatible with KCM
  - Conformance tests are passed and published to testgrid
- CSI drivers for AzureDisk/AzureFile are moving to GA
  - Feature compatible with KCM
  - Conformance tests are passed and published to testgrid
- Azure credential provider is still supported in Kubelet
  - Feature compatible with KCM
  - Conformance tests are passed and published to testgrid

### Upgrade / Downgrade Strategy

Upgrade/Downgrade Azure cloud controller manager and CSI drivers together with other master components. The versions for Azure cloud controller manager and CSI drivers should be chosen according to the version skew strategy below.

### Version Skew Strategy

For each Kubernetes minor releases (e.g. v1.15.x), dedicated Azure cloud controller manager would be released. For CSI drivers, however, they would be released based on [CSI specification versions](https://kubernetes-csi.github.io/docs/).

- The version matrix for Azure cloud controller manager would be documented on [kubernetes/cloud-provider-azure](https://github.com/kubernetes/cloud-provider-azure/blob/master/README.md#current-status).
- The version matrix for CSI drivers would be documented on [kubernetes-sigs/azuredisk-csi-driver](https://github.com/kubernetes-sigs/azuredisk-csi-driver#container-images--csi-compatibility) and [kubernetes-sigs/azurefile-csi-driver](https://github.com/kubernetes-sigs/azurefile-csi-driver#container-images--csi-compatibility).

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
