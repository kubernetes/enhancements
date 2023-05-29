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
# 3929: Remove CRI Socket Annotation from Node Object

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
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [init: upload a global kubelet configuration with cri socket](#init-upload-a-global-kubelet-configuration-with-cri-socket)
  - [join: can override it using --config](#join-can-override-it-using---config)
  - [upgrade: re-download global one, but should use local kubelet configuration firstly](#upgrade-re-download-global-one-but-should-use-local-kubelet-configuration-firstly)
  - [other proposal: respect a list of configuration in local kubelet configuration, and in v1.27, CRI socket is the only one](#other-proposal-respect-a-list-of-configuration-in-local-kubelet-configuration-and-in-v127-cri-socket-is-the-only-one)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
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
[kubernetes/website]: https://git.k8s.io/website

## Summary

The proposal is to remove the Container Runtime Interface (CRI) socket annotation
from the Node object in Kubernetes, which is currently added during the
"init upload-config" phase in Kubeadm. This annotation is used to specify the CRI
socket endpoint used by the kubelet on each node for communication with the container
runtime. Instead of relying on this annotation, the proposal suggests using a global
kubelet configuration with a CRI socket specified, as well as providing the ability to
override this configuration during kubeadm join using the --config flag. This would
eliminate the need for kubeadm to store CRI socket configuration on each node, and
instead rely on the Kubernetes configuration files for specifying this information.

## Motivation

Currently, kubeadm adds a CRI socket annotation to the Node object during the
"init upload-config" phase, which specifies the endpoint for the CRI that is being
used by the kubelet on each node. This annotation is persistent on the Node object,
even if the kubelet is updated or the CRI is changed.

After migration of container runtime endpoint flag to kubelet config, we can set
cri socket in kubelet configuration.

### Goals

- Remove the use of CRI socket annotation on Node object
- We will prioritize respecting the local setting over the global one.
- [Not decided yet] For further node customized kubelet configuration, it can be saved locally
  on disk with file path `/var/lib/kubelet/kubeadm-config.yaml`. (If not used, I will move it to Non-Goals)

### Non-Goals

- Update the CRI socket annotation on Node object to be the latest

## Proposal

1. init: upload a global kubelet configuration with cri socket.
   - the cri socket will take `--cri-socket` value and if the flag is empty, kubeadm will auto-detect it.
   - After seting or auto-detecting, it will be set in the global kubelet configuration.
2. join: it will use the global confugration.
   - if it is not set in the global configuration, it will use `--cri-socket` value
   - if it is still empty, kubeadm will auto-detect it.
   - join will not change the global configuration, and if it is different with the global,
     kubeadm will save it in `/var/lib/kubelet/kubeadm-config-instance.yaml`
3. upgrade: re-download global one, but should use local kubelet configuration firstly in `kubeadm-config-instance.yaml`

### User Stories (Optional)

#### Story 1

#### Story 2

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

## Design Details

### init: upload a global kubelet configuration with cri socket

- `kubeadm init` will not add the annotation to node any more.
- `kubeadm init` will check the customized `--config` at first and if no cri socket is set, it will
  auto-detect it and save it global configuration.
  if `--cri-socket` is specified, we will use it in the local kubelet configuration and `kubeadm-config-instance.yaml`,
  but it will not be saved to the global configuration.

### join: can override it using --config

- `kubeadm join` will not add the annotation to node.
- `kubeadm join` will download the kubelet configuration from apiserver and the customized `--config`
  at first and auto-detect will work only if not set. Auto-detect may log a warning message if it may
  be misconfigured and log a general debug log if there is multi CRI-sockets.
  if `--cri-socket` is specified, we will use it in the local kubelet configuration and `kubeadm-config-instance.yaml`,
  but it will not be saved to the global configuration.

### upgrade: re-download global one, but should use local kubelet configuration firstly

- `kubeadm upgrade` will download the kubelet configuration from apiserver and respect local one.
- in v1.28-1.29, for backward compatibility, when `kubeadm upgrade apply`, we will read the `cri` annotation(if no annotation, we autodetect it)
  and then patch it to the global configuration. `kubeadm upgrade node` is similar, and it will never change global configuration.
- in v1.30+, `kubeadm upgrade apply` will not read the cri annotation any more.
- in v1.28, for other nodes, `kubeadm upgrade node` will check if the cri annotation is diffent with the global setting.
  if `cri-socket` is different, we will use it in the local kubelet configuration and `kubeadm-config-instance.yaml`,
  but it will not be saved to the global configuration.
- in v1.29 or later, `kubeadm upgrade node` will check `kubeadm-config-instance.yaml` at first and then check annoation like v1.28.
- in v1.30+, `kubeadm upgrade node` will check `kubeadm-config-instance.yaml` and then global configuration only.

### other proposal: respect a list of configuration in local kubelet configuration, and in v1.27, CRI socket is the only one

During `kubeadm upgrade`, kubeadm will read the local kubelet configuration in `/var/lib/kubelet/config.yaml`.
kubeadm also download the kubelet configuration from configmap and replace the `containerRuntimeEndpoint` and
`imageServiceEndpoint`(This maybe empty and I prefer to respect it as well) with the local configuration.

A node-specific kubelet configuration list should be maintained in kubeadm code.

- containerRuntimeEndpoint
- imageServiceEndpoint

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

Install/Join/Upgrade test in <https://testgrid.k8s.io/sig-cluster-lifecycle-kubeadm>

- upgrade v1.(n-1) to v1.n.
- upgrade v1.n to v1.n.

##### Prerequisite testing updates

##### Unit tests

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

- <test>: <link to test coverage>

##### e2e tests

- <test>: <link to test coverage>

### Graduation Criteria

#### Alpha

- The upgrade will still respect the CRI annotation
- Initial e2e tests completed and enabled

#### Beta

- Use the local kubelet configuration or global configuration, ignore the CRI annotation
- Gather feedback from developers and surveys
- Additional tests are in Testgrid and linked in KEP

#### GA

- Remove the CRI annotation during upgrade(this may be not urgent or have to)
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

### Upgrade / Downgrade Strategy

See above.

### Version Skew Strategy

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: No
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane? No.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? No.

###### Does enabling the feature change any default behavior?

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

###### What specific metrics should inform a rollback?

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason:
- [ ] API .status
  - Condition name:
  - Other field:
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

### Scalability

###### Will enabling / using this feature result in any new API calls?

###### Will enabling / using this feature result in introducing new API types?

###### Will enabling / using this feature result in any new calls to the cloud provider?

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

## Drawbacks

## Alternatives

## Infrastructure Needed (Optional)
