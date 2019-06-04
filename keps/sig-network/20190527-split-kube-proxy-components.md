---
title: split-kube-proxy-components
authors:
  - "@vllry"
owning-sig: sig-network
participating-sigs:
  - sig-network
reviewers:
  - "@cmluciano"
  - "@danwinship"
  - "@luxas"
approvers:
  - "@andrewsykim"
  - "@thockin"
editor: "@vllry"
creation-date: 2019-05-27
last-updated: 2019-05-27
status: provisional
---

# split-kube-proxy-components

## Table of Contents

A table of contents is helpful for quickly jumping to sections of a KEP and for highlighting any additional information provided beyond the standard KEP template.
[Tools for generating][] a table of contents from markdown are available.

Table of Contents
=================

   * [split-kube-proxy-components](#split-kube-proxy-components)
      * [Table of Contents](#table-of-contents)
      * [Release Signoff Checklist](#release-signoff-checklist)
      * [Summary](#summary)
      * [Motivation](#motivation)
         * [Goals](#goals)
         * [Non-Goals](#non-goals)
      * [Proposal](#proposal)
         * [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
         * [Risks and Mitigations](#risks-and-mitigations)
      * [Design Details](#design-details)
         * [Test Plan](#test-plan)
         * [Graduation Criteria](#graduation-criteria)
         * [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
         * [Version Skew Strategy](#version-skew-strategy)
      * [Implementation History](#implementation-history)
      * [Drawbacks](#drawbacks)
      * [Alternatives](#alternatives)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

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

Replace kube-proxy configuration flags with ComponentConfig.
Use one ComponentConfig for each ProxyMode.

In the background, split up per-ProxyMode code to be independent.

## Motivation

* Implement WG-component-standard for config.
* Clean up mode-specific code in kube-proxy.
    * kube-proxy has a history of bugs caused by overlap between ProxyMode functionality,
    EG https://github.com/kubernetes/kubernetes/issues/75360
    * Most ProxyMode code (like Windows mode) is already logically completely separate from the others.

### Goals

* Replace kube-proxy flag configuration with ComponentConfig.
Make ComponentConfig be distinct per-mode (EG iptables config vs IPVS config).
* Split the startup and run code for each ProxyMode into distinct packages.
    * ProxyMode code should not import code from other ProxyModes.

### Non-Goals

* Moving kube-proxy out of k/k (to be considered in a future enhancement).
* Splitting kube-proxy into multiple binaries by mode (to be considered in a future enhancement).

## Proposal

* Separate ProxyMode code internally into subpackages.
    * Move any common logic into shared subpackages.
* Introduce a ComponentConfig for each ProxyMode.
* Shift users to using ComponentConfig and away from flags.

### Implementation Details/Notes/Constraints

TBD

### Risks and Mitigations

Testing must account for distinct configuration handling between each ProxyMode.
A side effect of deliberately duplicating calls/lines-of-code is the increase in combinations of possible behavior.

## Design Details

### Test Plan

TBD

### Graduation Criteria

TBD

### Upgrade / Downgrade Strategy

TBD

### Version Skew Strategy

TBD

## Implementation History

## Drawbacks

TBD

## Alternatives

* Adopt a single kube-proxy ComponentConfig, for use with any mode.
* Split kube-proxy out of kubernetes/kubernetes first.
