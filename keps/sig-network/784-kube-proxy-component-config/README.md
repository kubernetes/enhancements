# kube-proxy component config graduation proposal

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Re-encapsulate mode specific options](#re-encapsulate-mode-specific-options)
    - [Example](#example)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [X] KEP approvers have set the KEP status to `implementable`
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

This document is intended to propose a process and desired goals by which kube-proxy's component configuration is to be graduated to beta.

## Motivation

kube-proxy is a component, that is present in almost all Kubernetes clusters in existence.
Historically speaking, kube-proxy's configuration was supplied by a set of command line flags. Over time, the number of flags grew and they became unwieldy to use and support. Thus, kube-proxy gained component config.
Initially this was just a large flat object, that was representing the command line flags. However, over time new features were added to it, all while staying as v1alpha1.

This resulted in a configuration format, that had various different options grouped together in ways, that made them hard to specify and understand. For example:

- Instance local options (such as host name override, bind address, etc.) are in the same flat object as shared between instances options (such as the cluster CIDR, config sync period, etc.).
- Platform specific options are mixed together. For example, the IPTables rule sync fields are used by the Windows HNS backend for the same purpose.
- Again, the IPTables rule sync options are used for the Linux legacy user mode proxy, but not for the IPVS mode (where a set of identical options exist, despite the fact, that it too uses some other fields, designed for IPTables).

Clearly, this made the configuration both hard to use and to maintain. Therefore, a plan to restructure and stabilize the config format is needed.

### Goals

- To cleanup the existing config format.
- To provide config structure, that is easier for users to understand and use.
- To distinguish between instance local and shared settings.
- To allow for the persistence of settings for different platforms (such as Linux and Windows) in a manner that reduces confusion and the possibility of an error.
- To allow for easier introduction of new proxy backends.
- To provide users with flexibility, especially with regards to the config source.

### Non-Goals

- To change or implement additional features in kube-proxy.
- To deal with graduation of any other component of kube-proxy, other than its configuration.
- To remove most or even all of the command line flags, that have corresponding component config options.

## Proposal

The idea is to conduct the process of graduation to beta in small steps in the span of at least one Kubernetes release cycle. This will be done by creating one or more alpha versions of the config with the last alpha version being copied as v1beta1 after the community is happy with it.
Each of the sub-sections below can result in a separate alpha version release, although it will be better for users to have no more than a couple of alpha versions past v1alpha1.
After each alpha version release, the community will gather around for new ideas on how to proceed in the graduation process. If there are viable proposals, this document is updated with an appropriate section(s) below and the new changes are introduced in the form of new alpha version(s).
The proposed process is similar to the already successfully used one for kubeadm.

### Re-encapsulate mode specific options

The current state of the config has proven that:
- Some options are deemed as mode specific, but are in fact shared between all modes.
- Some options are placed directly into KubeProxyConfiguration, but are in fact mode specific ones.
- There are options that are shared between some (but not all) modes. Specific features of the underlying implementation are common and this happens only within the boundaries of the platform (iptables and ipvs modes for example).
- Although legacy Linux and Windows user mode proxies are separate code bases, they have a common set of options.

With that in mind, the following measures are proposed:
- Mode specific structs are consolidated to not use fields from other mode specific structs.
- Introduce a single combined legacy user mode proxy struct for both Linux and Windows backends.

#### Example

```yaml
commonSetting1: ...
commonSetting2: ...
...
modeA: ...
modeB: ...
modeC: ...
```

### Risks and Mitigations

So far, the following risks have been identified:
- Deviation of the implementation guidelines and bad planning may have the undesired effect of producing bad alpha versions.
- Bad alpha versions will need good alpha versions to fix them. This will create too many iterations over the API and users may get confused.
- New and redesigned kube-proxy API versions may cause confusion among users who are used to the v1alpha1 relatively flat, single document design. In particular, multiple YAML documents and structured (as opposed to flat) objects can create confusion as to what option is placed where.

The mitigations to those risks:
- Strict following of the proposals in this document and planning ahead for a release and config cycle.
- Support reading from the last couple of API versions released. When the beta version is released, support the last alpha version for one or two release cycles after that.
- Documentation on the new APIs and how to migrate to them.
- Provide optional migration tool for the APIs.

## Design Details

### Test Plan

Existing test cases throughout the kube-proxy code base should be adapted to use the latest config version.
If required, new test cases should also be created.

### Graduation Criteria

The config should be considered graduated to beta if it:
- is well structured with clear boundaries between different proxy mode settings.
- allows for easy multi-platform use with less probability of an error.
- allows for easy distinguishment between instance local and shared settings.
- is well covered by tests.
- is well documented. Especially with regards of migrating to it from older versions.
