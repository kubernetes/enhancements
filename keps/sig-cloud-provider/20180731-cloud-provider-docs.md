---
title: Cloud Provider Documentation
authors:
  - "@d-nishi"
  - "@hogepodge"
  - "@andrewsykim"
owning-sig: sig-cloud-provider
participating-sigs:
  - sig-docs
  - sig-cluster-lifecycle
reviewers:
  - "@andrewsykim"
  - "@calebamiles"
  - "@hogepodge"
  - "@jagosan"
approvers:
  - "@andrewsykim"
  - "@hogepodge"
  - "@jagosan"
editor: TBD
creation-date: 2018-07-31
last-updated: 2019-02-12
status: implementable
---

## Documentation Requirements for Kubernetes Cloud Providers

### Table of Contents

<!-- toc -->
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [Proposed Format for Documentation](#proposed-format-for-documentation)
    - [Requirement 1: Example Manifests](#requirement-1-example-manifests)
    - [Requirement 2: Resource Management](#requirement-2-resource-management)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Graduation Criteria](#graduation-criteria)
  - [Alternatives [optional]](#alternatives-optional)
- [Implementation History](#implementation-history)
<!-- /toc -->

### Summary
This KEP describes the documentation requirements for both in-tree and out-of-tree cloud providers.
These requirements are meant to ensure the usage and integration of cloud providers with the rest of the Kubernetes ecosystem is concise and clear and that documentation layout across providers is consistent.

### Motivation
Currently documentation for cloud providers for both in-tree and out-of-tree managers is limited in both scope, consistency, and quality. This KEP describes requirements to create and maintain consistent documentation across all cloud providers. By establishing these standards, SIG Cloud Provider will benefit the user-community by offering a single discoverable source of reliable documentation while relieving the SIG-Docs team from the burden of maintaining content from various cloud providers across many Kubernetes versions.

#### Goals

* Create a single source of truth that outlines what documentation from each provider should cover.
* Ensure all Kubernetes cloud providers adhere to this doc format to provide a consistent experience for all users.
* Ensure SIG Docs can confidently link to documentation by any Kubernetes cloud provider on any future releases.

#### Non-Goals

* Where in the Kubernetes website the cloud provider documentation will be hosted. This should be a decision made by SIG Docs based on the content given by each cloud provider.
* Cloud provider documentation outside the scope of Kubernetes.

### Proposal

#### Proposed Format for Documentation

The following is a proposed format of how docs should be added to `k8s.io/cloud-provider-*` repositories.

#### Requirement 1: Example Manifests

Provide validated manifests for every component required for both in-tree and out-of-tree versions of your cloud provider. The contents of the manifest should contain a DaemonSet resource that runs these components on the control plane nodes or a systemd unit file. The goal of the example manifests is to provide enough details on how each component in the cluster should be configured. This should provide enough context for users to build their own manifests if desired.

```
cloud-provider-foobar/
├── ...
├── ...
├── docs
│   └── example-manifests
│       └── in-tree/
│           ├── apiserver.manifest                 # an example manifest of apiserver using the in-tree integation of this cloud provider
│           ├── kube-controller-manager.manifest   # an example manifest of kube-controller-manager using the in-tree integration of this cloud provider
│           ├── kubelet.manifest                   # an example manifest of kubelet using the in-tree integration of this cloud provider
│       └── out-of-tree/
│           ├── apiserver.manifest                 # an example manifest of apiserver using the out-of-tree integration of this cloud provider
│           ├── kube-controller-manager.manifest   # an example manifest of kube-controller-manager using the out-of-tree integration of this cloud provider
│           ├── cloud-controller-manager.manifest  # an example manifest of cloud-controller-manager using the out-of-tree integration of this cloud provider
│           ├── kubelet.manifest                   # an example manifest of kubelet using out-of-tree integration of this cloud provider
```

#### Requirement 2: Resource Management

List the latest annotations and labels that are cloud-provider dependent and can be used by the Kubernetes administrator. The contents of these files should be kept up-to-date as annotations/labels are deprecated/removed/added/updated. Labels and annotations should be grouped based on the resource they are applied to. For example the label `beta.kubernetes.io/instance-type` is applied to nodes so it should be added to `k8s.io/cloud-provider-foobar/docs/node/labels.md`.

```
cloud-provider-foobar/
├── ...
├── ...
├── docs
│   └── resources
│       └── node/
│           ├── labels.md        # outlines what annotations that can be used on a Node resource
│           ├── annotations.md   # outlines what annotations that can be used on a Node resource
│           ├── README.md        # outlines any other cloud provider specific details worth mentioning regarding Nodes
│       └── service/
│           ├── labels.md        # outlines what annotations that can be used on a Service resource
│           ├── annotations.md   # outlines what annotations that can be used on a Service resource
│           ├── README.md        # outlines any other cloud provider specific details worth mentioning regarding Services
│       └── persistentvolumes/
│           ├── ...
│       └── ...
│       └── ...
```

### Implementation Details/Notes/Constraints [optional]

The requirements above lists the bare minimum documentation that any cloud provider in the Kubernetes ecosystem should have. Cloud providers may choose to add more contents under the `docs/` directory as they see fit.

### Risks and Mitigations

There are no risks and mitigation with respect to adding documentation for cloud providers. If there are any, they would already exist in the various places the cloud provider docs exists today and implementing this KEP would not increase those risks.

### Graduation Criteria

* All cloud providers have written docs that adhere to the format specified in this KEP.
* SIG Docs is consuming docs written by each provider in a way that is easily consumable for Kubernetes' users.

### Alternatives [optional]

* SIG Docs can be solely responsible for writing documentation around integrating Kubernetes with the existing cloud providers. This alternative would not be efficient because it would require SIG Docs to understand the context/scope of any cloud provider work that happens upstream. Developers who work with cloud provider integrations are most fit to write the cloud provider integration docs.

## Implementation History

- July 31st 2018: KEP is merged as a signal of acceptance. Cloud providers should now be looking to add documentation for their provider according to this KEP.
- Nov 19th 2018: KEP has been in implementation stage for roughly 4 months with Alibaba Cloud, Azure, DigitalOcean, OpenStack and vSphere having written documentation for their providers according to this KEP.
- Feb 12th 2019: KEP has been updated to state the implementation details and goals more clearly

