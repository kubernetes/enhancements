---
title: Component Config fields should be optional
authors:
  - "@justinsb"
owning-sig: sig-apimachinery
participating-sigs:
  - sig-cluster-lifecycle
reviewers:
  - @mtaufen
  - @stealthybox
  - @stts
approvers:
  - TBD
editor: TBD
creation-date: 2019-07-26
last-updated: 2019-07-26
status: provisional
---

# Component Config fields should be optional

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Can marshal to yaml](#can-marshal-to-yaml)
    - [Can set fields to zero values](#can-set-fields-to-zero-values)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
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

`ComponentConfig` allows for specification of options for a component in a
structured and versioned way; kubernetes apimachinery objects replace flags.

However, while flags make the distinction between not-specified and
set-to-empty-value, our api types generally do not.  By using pointer-types it
is possible to preserve this difference, this KEP proposes that we should use
pointer-types in our componentconfig.

This also allows for generation of a minimal componentconfig from code, and
should faciliate patching and merging of componentconfig.

ComponentConfig types are used with v1alpha1 versions for kube-scheduler,
kube-proxy, kube-controller-manager, cloud-controller-manager; I intend to make
these nilable.  kubelet uses a ComponentConfig type that is v1beta1, I propose
not changing those at the current time, but rather proving out the approach with
the v1alpha1 types.

## Motivation

I initially raised this in a [mailing list
thread](https://groups.google.com/d/topic/kubernetes-wg-component-standard/hO7Lmi0cwQU/discussion)
with wg-component-standard.  When trying to create componentconfig from code,
all fields were populated, which did not really reflect the intent.  For minimal
output, either simply for hygene or to support kustomize, we would have needed
another type.  We would then have had to keep that type in sync with changes to
componentconfig.  It seemed that componentconfig could better serve as that
type.

Supporting merging of configurations that better reflects intent has come up a
few times: there have been discussions of merging fields with flags and the
precedence when doing so, merging fields set at different levels of a hierarchy
(e.g. cluster-level vs node-level kube-proxy flags).  Recording the distinction
between the zero-value and not-set allows for these operations to better reflect
intent.

### Goals

- Allow generation of compact componentconfig using the same types
- Start to replace some of the "magic sentinel" values that are used to mean "not set"

### Non-Goals

- Deeper refactoring of component config types into more structured objects
  (this is valuable, merely not under the umbrella of this KEP)

## Proposal

In v1.16 we will change the types of the fields in the kube-proxy,
kube-scheduler, kube-controller-manager, cloud-controller-manager
ComponentConfig so that they are all nilable.  We believe we can keep the
version as v1alpha1, because the yaml and json form has not changed (though the
proto would, but componentconfig is not specified using proto).  We believe we
can keep the internal versions with non-nilable fields, and perform the
conversion.

After demonstrating this in v1.16 (ideally in v1.17) we will change the types of
the fields in kubelet so that they are nilable.  This may require introduction
of a v1beta2 (TBD).

### User Stories

#### Can marshal to yaml

When I create an object of type KubeSchedulerConfiguration and marshal it to
yaml/json, only the fields I have set appear in the output (other than "header
fields" like Kind & APIVersion etc).

#### Can set fields to zero values

When I set fields to zero-values in a KubeSchedulerConfiguration (0 for
integers, "" for strings etc), and marshal to yaml/json, those fields do appear
in the output, reflecting my intent.  This enables other tooling such as
kustomize to know that I set these fields.

### Implementation Details/Notes/Constraints

We will begin with kube-scheduler and kube-proxy, as those are the smallest;
there is also more desire to promote those to beta.

Implementation details TBD!

### Risks and Mitigations

We are not initially changing the kubelet componentconfig behaviour (because it
has been promoted to v1beta1), which means there will be some inconsistency.
However, I think this is appropriate, in that this keeps the work smaller, and
allows us to prove the value before we change types with stricter versioning
requirements.

We risk changing the behaviour of an unspecified field in the componentconfig
yaml.  We will certainly need a release note to cover this change, but the
structures are currently in alpha so this does not require full deprecation.  We
can aim not to change the behaviour, and to explicitly enumerate any changed
fields in the release note.

## Design Details

### Test Plan

Create tests reflecting the two user stories above:

* Empty values should result in mostly-empty output
* Can set fields to empty and have the results be empty (fuzz-style testing, ideally)

Create tests specifically around magic-sentinel values where the default field
values were not the zero-value.

### Graduation Criteria

This is proposing a change to alpha APIs, which should hopefully bring them
closer to graduation, but we are not proposing the graduation in this KEP.

### Upgrade / Downgrade Strategy

Because there should be no change to the yaml or json forms of the schema, no
specific strategy is required.

### Version Skew Strategy

Because there should be no change to the yaml or json forms of the schema, no
specific strategy is required.

## Implementation History

2019-07-26: KEP proposed
