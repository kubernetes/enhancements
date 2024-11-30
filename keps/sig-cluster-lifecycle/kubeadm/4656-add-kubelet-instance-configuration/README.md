<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-4656: Add kubelet instance configuration to configure CRI socket for each node

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

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
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The proposal is to no longer set the Container Runtime Interface (CRI) socket annotation  named `kubeadm.alpha.kubernetes.io/cri-socket` from the Kubernetes Node object, which is currently added during the `kubeadm init upload-config` phase. This annotation is used to specify the CRI socket endpoint used by the kubelet on each node for communication with the container runtime.

Instead of relying on the annotation, this KEP proses creating an instance config per node and overriding the `ContainerRuntimeEndpoint` in the kubelet config when calling kubeadm commands. This will eliminate the need for kubeadm to store CRI socket configuration on each Node object.

## Motivation

Currently, kubeadm adds a CRI socket annotation to the Node object during the `init upload-config` phase, which specifies the endpoint for the CRI that is being used by the kubelet on each node. This annotation is persistent on the Node object, even if the kubelet is updated or the CRI is changed.

After migrating the container runtime endpoint flag to the instance configuration, we can use it
Set the CRI socket by overriding the `ContainerRuntimeEndpoint` field in `/var/lib/kubelet/config.yaml`.

### Goals

* kubeadm currently adds an annotation with the key `kubeadm.alpha.kubernetes.io/cri-socket` to each Node object. We will deprecate and no longer set it.
* Provide an instance configuration file named `/var/lib/kubelet/instance-config.yaml` for each node, in which the `ContainerRuntimeEndpoint` field is defined. During the `kubeadm init/join/upgrade` process, the instance configuration will be read and the `ContainerRuntimeEndpoint` field in `/var/lib/kubelet/config.yaml` will be overwritten.
* The `--container-runtime-endpoint` flag is no longer written to the `/var/lib/kubelet/kubeadm-flags.env` file.

### Non-Goals

- Continue maintaining CRI socket paths on Node objects.

## Proposal

We will add a new file `/var/lib/kubelet/instance-config.yaml` to customize the CRI socket of each node. This file will be merged with `/var/lib/kubelet/config.yaml` in the process of kubeadm init/join by using the `kubeletconfiguration` patch target. If the user uses the `kubeletconfiguration` with `--patches`, the patch file provided by the user will be given priority.

For different subcommands, there are the following changes:

* kubeadm init: If the CRI socket provided in the kubeadm configuration is set, it will take precedence and generate the `/var/lib/kubelet/instance-config.yaml` configuration file based on it; if the CRI socket is not specified, the container runtime endpoint will be automatically detected and generate the `/var/lib/kubelet/instance-config.yaml` file.

* kubeadm join: If the CRI socket provided in the kubeadm configuration is set, it will take precedence and generate the `/var/lib/kubelet/instance-config.yaml` configuration file based on it. If no CRI socket is specified, the socket is automatically detected on the node and `/var/lib/kubelet/instance-config.yaml` is generated based on it.

* kubeadm upgrade: future versions of `kubeadm upgrade apply/node` will only check `/var/lib/kubelet/instance-config.yaml`.

### Risks and Mitigations

## Design Details

**We will add a new `NodeLocalCRISocket` feature gate. In the Alpha phase, the feature gate is disabled by default. If feature gate is disabled, kubeadm subcommands will not be changed, when the feature gate is enabled, the kubeadm subcommands change as follows:** 

kubeadm init:

* No longer need to write the `--container-runtime-endpoint` to `/var/lib/kubelet/kubeadm-flags.env`.
* No longer need to add the `kubeadm.alpha.kubernetes.io/cri-socket` annotation.
* If the CRI socket provided in the kubeadm configuration is set, it is used first and the `/var/lib/kubelet/instance-config.yaml` configuration file is generated based on it. If the CRI socket is not set, the container runtime endpoint is automatically detected and generate the `/var/lib/kubelet/instance-config.yaml` file.

kubeadm join:

* No longer need to add the `kubeadm.alpha.kubernetes.io/cri-socket` annotation.
* If the CRI socket provided in the kubeadm configuration is set, it is used first and the `/var/lib/kubelet/instance-config.yaml` configuration file is generated based on it. If no CRI socket is specified, the socket is automatically detected on the node and `/var/lib/kubelet/instance-config.yaml` is generated based on it.

kubeadm reset:

* There is no need to do anything, according to the existing process, we get CRISocketPath before deleting the /var/lib/kubelet directory, and after deleting the `/var/lib/kubelet` directory, `/var/lib/kubelet/instance-config.yaml` will also be cleaned up.

kubeadm upgrade:

**In the Alpha phase, the feature gate is disabled by default. If feature gate is enabled, the kubeadm subcommands change as follows:** 
* `kubeadm upgrade node/apply` will check the `--container-runtime-endpoint` flag in the `/var/lib/kubelet/kubeadm-flags.env` file and generate `/var/lib/kubelet/instance-config.yaml` based on it. The flag `--container-runtime-endpoint` will be then removed from `/var/lib/kubelet/kubeadm-flags.env` .

**In the Beta phase, the feature gate is enabled by default. If feature gate is disabled, kubeadm subcommands will not be changed, when the feature gate is enabled, the kubeadm subcommands change as follows:** 

* `kubeadm upgrade apply/node`  will use `/var/lib/kubelet/instance-config.yaml`, and override the `ContainerRuntimeEndpoint` field to `/var/lib/kubelet/config.yaml`.

**In the GA phase, the feature gate is enabled by default and cannot be disabled. the kubeadm subcommands change as follows:** 

* `kubeadm upgrade apply/node` will use `/var/lib/kubelet/instance-config.yaml` override the `ContainerRuntimeEndpoint` field to `/var/lib/kubelet/config.yaml` only.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

At least the following kubeadm packages will require updates and new unit tests:

* `cmd/kubeadm/app/cmd/phases/init`

* `cmd/kubeadm/app/phases/kubelet`

* `cmd/kubeadm/app/phases/upgrade`

* `cmd/kubeadm/app/cmd/phases/join`

##### Integration tests

- N/A

##### e2e tests

* A new e2e test will be added by using the kinder tool.

### Graduation Criteria

#### Alpha

- Use `NodeLocalCRISocket` feature gate to implement features.
- Add corresponding e2e tests.
- Added documentation for feature gates.

#### Beta

* Make feature gate to be enabled by default.

- Gather feedback from developers and surveys.
- Update the feature gate documentation.
- Implement changes in kubeadm upgrade apply/node Beta phase.

#### GA

- Gather feedback from developers and surveys.
- Implement changes in kubeadm upgrade apply/node GA phase.
- Update the phases documentation.
- Remove kubeadm.alpha.kubernetes.io/cri-socket annotation from https://kubernetes.io/docs/reference/labels-annotations-taints page.
- Update https://kubernetes.io/docs/tasks/administer-cluster/migrating-from-dockershim/change-runtime-containerd/ page and replace update annotation with update instance-config.

### Upgrade / Downgrade Strategy

**Alpha**: Users can patch their `ClusterConfiguration` in the `kube-system/kubeadm-config` ConfigMap to enable the `NodeLocalCRISocket` feature gate before calling kubeadm upgrade apply, which will allow a `/var/lib/kubelet/instance-config.yaml` to be generated and overwrite the `ContainerRuntimeEndpoint` field in `/var/lib/kubelet/config.yaml` with it.

**Beta**: Users can modify `ClusterConfiguration` to disable the feature gate during upgrades. This will allow them to continue using the CRI socket annotation on nodes.

**GA**: Users can no longer patch ClusterConfiguration to opt out of the feature and it will be locked to be enabled by default.

### Version Skew Strategy

kubeadm will continue to skew from kubelet for three versions. The `ContainerRuntimeEndpoint` field in `KubeletConfiguration` was [introduced](https://github.com/kubernetes/kubernetes/pull/112136) in v1.27, so when we overwrite the `ContainerRuntimeEndpoint` field in `/var/lib/kubelet/config.yaml` through `/var/lib/kubelet/instance-config.yaml`, it will be supported on all versions of the kubelet within the skew.

## Implementation History

- 2024-05-23: Initial draft KEP.
- 2024-10-03: KEP marked as implementable.
- 2024-11-30: Modify KEP based on implemented PR.

## Drawbacks

* This KEP will bring a breaking change. some users do read / write the `kubeadm.alpha.kubernetes.io/cri-socket` annotation or the `/var/lib/kubelet/kubeadm-flags.env` file to declare the CRI socket endpoint on the Node, because many users are familiar with them.

## Alternatives

* We can avoid providing feature gates and ensure the compatibility of kubeadm by implementing it in multiple versions, but we should improve user awareness by adding a feature gate.
* Do nothing, continue to use the`/var/lib/kubelet/kubeadm-flags.env` file, but kubelet has deprecated the `--container-runtime-endpoint` args.

## Infrastructure Needed (Optional)
