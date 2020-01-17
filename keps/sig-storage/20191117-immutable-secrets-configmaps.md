---
title: Immutable Secrets and ConfigMaps
authors:
  - "@wojtek-t"
owning-sig: sig-storage
participating-sigs:
  - sig-apps
  - sig-node
  - sig-scalability
reviewers:
  - "@yujuhong"
  - "@lavalamp"
  - "@msau42"
approvers:
  - "@saad-ali"
creation-date: 2019-11-17
last-updated: 2019-12-09
status: implementable
see-also:
replaces:
superseded-by:
---

# Immutable ephemeral volumes

## Table of Contents

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
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [Define immutability at VolumeSource level](#define-immutability-at-volumesource-level)
  - [Optimize watches](#optimize-watches)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

The most popular and the most convinient way of consuming Secrets and
ConfigMaps by Pods is consuming it as a file. However, any update to
a Secret or ConfigMap object is quickly (roughly within a minute)
reflected in updates of the file mounted for *all* Pods consuming them.
That means, that a bad update (push) of Secret and/or ConfigMap can
very quickly break the whole appplication.

The recommended approach for upgrading applications are obviously
rolling upgrades. For Secrets/ConfigMaps this is translating to
creating a new object and updating PodTemplate with the reference
to it. However, that doesn't protect users from outages caused by
accidental bad updates of existing Secrets and/or ConfigMaps.

Moreover, the feature of updating Secrets/ConfigMaps for running
Pods is also expensive from scalability/performance point of view.
Every Kubelet has to watch (default) or periodically poll every
unique Secret and ConfigMap referenced by any of the Pods it is
running.

## Motivation

In this KEP, we are proposing to introduce an ability to specify
that contents of a particular Secret/ConfigMap should be immutable
for its whole lifetime. For those Secrets/ConfigMap, Kubelets will
not be trying to watch/poll for changes to updated mounts for their
Pods.
Given there are a lot of users not really taking advantage of automatic
updates of Secrets/ConfigMaps due to consequences described above, this
will allow them to:
- protect themselves better for accidental bad updates that could cause
  outages of their applications
- achieve better performance of their cluster thanks to significant
  reduction of load on apiserver

### Goals

- introduce protection mechanism to avoid outages due to accidental
  updates of existing Secrets/ConfigMaps
- improve cluster performance by reducing load on Kubernetes control
  plane (mostly kube-apiserver) consumed by a feature many people
  would be willing to tradeoff for better scale/performance

### Non-Goals

- change the default behavior of consumption of Secrets/ConfigMaps

## Proposal

We propose to extend `ConfigMap` and `Secret` types with an additional
field:
```go
  Immutable *bool
```

If set, the machinery in apiserver will reject any updates of the object
trying to change anything different than ObjectMetadata.

Note that will NOT reject all updates of the object, as we need to allow
e.g. for mutating ObjectMetadata (to not break object lifecycle, e.g. by
introducing a deadlock if Finalizers are set) or to allow rotating
certificates used for encryption at rest. We will only reject requests
that are explicitly changing keys and/or values stored in Secrets and/or
ConfigMaps.

Based on the value of `Immutable` field Kubelet will or will not:
- start a watch (or periodic polling) of a given Secret/ConfigMap
- perform updates of files mounted to a Pod based on updates of
  the Kubernetes object

### Risks and Mitigations

Given how closely the implementation of the feature will be related to
the implementation of automatic updates of Secrets/ConfigMaps, there is
a risk for introducing a bug and breaking that feature. The existing
unit and e2e tests should catch that, but we will audit them and add
new ones to cover the gaps if needed. Additionally, we will try to hide
the new logic behind the feature gate.

## Design Details

### Test Plan

For `Alpha`, we will add e2e tests verifying that contents of Secrets and
ConfigMaps marked as immutable really can't be updated. Additionally, these
will check if the metadata can be modified.

Additionally, unit tests will be added in Kubelet codebase to ensure that
the newly added logic to not watch immutable Secrets/ConfigMaps works as
expected.

For `Beta`, we will also extend scalability tests with a number of immutable
`Secrets` and `ConfigMaps` to validate the performance impact (for `Alpha`
only manual scalability tests will be performed).

### Graduation Criteria

Alpha:
- All tests describe above for `Alpha` are implemented and passing.
- Manual scalability tests prove the expected impact.

Beta:
- Scalability tests are extended to mount an immutable Secret and ConfigMap
for every single Pod, and that doesn't violate existing SLOs.

GA:
- No complaints about the API and user bug reports for 2 releases.

### Upgrade / Downgrade Strategy

No upgrade/downgrade concerns.

### Version Skew Strategy

On Nodes in versions on supporting the feature, Kubelet will still be watching
immutable Secrets and/or ConfigMaps. That said, this is purely a performance
improvement and doesn't have correctness implications. So those clusters will
simple have worse scalability characteristic.

## Implementation History

2019-11-18: KEP opened
2019-12-09: KEP marked implementable

## Alternatives

### Define immutability at VolumeSource level

Instead of making an object immutable, we could define immutability
per mount in VolumeSource.

Pros:
- higher granularity
Cons:
- unclear/tricky semantic on Kubelet restarts and Pod restarts/updates

### Optimize watches

We could potentially address scalability/performance aspect by optimizing
apimachinery. However, the bottlenecks seem to be mainly at the level of
Golang memory allocations.

Pros:
- no additional API
Cons:
- doesn't protect against unexpected bad updates causing outages
- unclear to what extent we can push the limits here (if at all)
