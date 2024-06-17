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
# KEP-4580: Deprecate & remove Kubelet RunOnce mode

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
  - [KubeletConfiguration Change: KubeletConfiguration](#kubeletconfiguration-change-kubeletconfiguration)
  - [kubelet flag Change](#kubelet-flag-change)
  - [Implement warning logging for RunOnce mode usage](#implement-warning-logging-for-runonce-mode-usage)
  - [Introduction LegacyNodeRunOnceMode feature gate](#introduction-legacynoderunoncemode-feature-gate)
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
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Deprecate and remove kubelet support for RunOnce mode, and mark the `RunOnce` field in `KubeletConfiguration` and the `--runonce` flag of kubelet as deprecated,  finally remove the `--runonce` flag.

## Motivation

* RunOnce mode has been broken and does not work.
* RunOnce mode doesn't support many newer pod features (init containers).
* RunOnce mode does not apply to the pod lifecycle we describe in the documentation, e.g. it does not support any volumes.
* RunOnce only provides some unit tests, without any e2e or integration tests, which makes us unable to guarantee whether it is usable.

### Goals

* Mark the `RunOnce` field in `KubeletConfiguration` and the `runonce` flag of kubelet as deprecated,  and finally remove the `runonce` flag.
* Remove kubelet support for RunOnce mode.

### Non-Goals

Immediate removal: the deprecation and removal process will be gradual and feature gate to increase awareness among potential users.

## Proposal

The RunOnce mode of kubelet will exit the kubelet process after spawning pods from the local manifests or remote URL. It is suitable for scenarios where one-time tasks need to be run on the node, this proposal outlines plans to deprecate and remove RunOnce mode in kubelet.

### Risks and Mitigations

Some people may still rely on this feature, but podman addresses the same use case with more well-supported way, ref: https://docs.podman.io/en/latest/markdown/podman-kube.1.html. Affected users can migrate to *podman kube subcommand* on demand.

For Docker users, Docker does not officially provide a subcommand similar to podman-kube-play to create containers with Kubernetes YAML, and there is currently no mature and reliable third-party tool to translate Kubernetes YAML into Docker Compose files, but they can manually perform this process and run containers in the form of Docker Compose.

## Design Details

### KubeletConfiguration Change: KubeletConfiguration

Mark the `RunOnce` field as deprecated.

### kubelet flag Change

make the `--runonce` flag as deprecated, and remove it in GA version.

### Implement warning logging for RunOnce mode usage

Starting in 1.31, during kubelet startup, if running in RunOnce mode, the kubelet will log a warning message, for example:

```
klog.ErrorS(nil, "RunOnce mode is deprecated, please migrate to the podman kube subcommand.")
```

### Introduction LegacyNodeRunOnceMode feature gate

With the introduction of the `LegacyNodeRunOnceMode` feature gate, Kubernetes aims to guide users through the deprecated RunOnce mode. Unless this feature gate is enabled, kubelet will refuse to start when the `--runonce` command line flag is set.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- N/A

##### Integration tests

- N/A

##### e2e tests

- N/A

### Graduation Criteria

#### Alpha

- Feature gate `LegacyNodeRunOnceMode` is introduced, is enable by default. Disable this feature gate will fail the kubelet on startup with RunOnce mode enable.
- Mark the `RunOnce` field in `KubeletConfiguration` as deprecated.

#### Beta

- `LegacyNodeRunOnceMode` feature gate is disable by default.
- Failed when starting kubelet in RunOnce mode.

#### GA

- We make the `LegacyNodeRunOnceMode` feature gate disable by default and cannot be enable.
- Comment the `RunOnce` field in KubeletConfiguration as 'no longer has any effect', and remove the kubelet's `--runonce` flag.
- Remove kubelet RunOnce mode.

### Upgrade / Downgrade Strategy

### Version Skew Strategy

- N/A

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: LegacyNodeRunOnceMode
  - Components depending on the feature gate: kubelet
  - Will enabling / disabling the feature require downtime of the control
    plane? Yes. Flag must be set on kubelet start. To disable, kubelet must be restarted. Hence, there would be brief control component downtime on a given node.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? Yes. See above; disabling would require brief node downtime.

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Using the feature gate is the only way to enable/disable this feature.

###### What happens if we reenable the feature if it was previously rolled back?

Re-enabling the feature will make the RunOnce functionality available in the kubelet. 

###### Are there any tests for feature enablement/disablement?

N/A

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

In the alpha `stage`, this feature is enable by default.

Cluster operators can test the behavior by enabling the feature gate.

In the beta `stage`, this feature is enable by default. With this feature disabled, the kubelet will refuse to start if is still using RunOnce mode.

Cluster operators can reinstate the mode by explicitly enabling the feature gate.

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

We will deprecate and remove the `--runonce` flag of kubelet and the `RunOnce` field in `KubeletConfiguration`.

### Monitoring Requirements

* N/A

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- \- 2024-04-17: Initial draft KEP

## Drawbacks

## Alternatives

* Fix RunOnce mode and add e2e tests and integration tests.
* Make RunOnce mode work and support volumes.

## Infrastructure Needed (Optional)
