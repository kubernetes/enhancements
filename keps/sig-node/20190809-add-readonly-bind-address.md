---
title: KEP Template
authors:
  - "@jfbai"
owning-sig: sig-node
participating-sigs:
  - sig-cluster-lifecycle
reviewers:
  - "@mtaufen"
  - "@mattjmcnaughton"
approvers:
  - TBD
editor: TBD
creation-date: 2019-08-09
last-updated: 2019-08-09
status: implementable
---

# Introduce a new field ReadOnlyBindAddress to kubelet configuration

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
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

This KEP intends to introduce a new field `ReadOnlyBindAddress` to `KubeletConfiguration`, so that kubelet serves on insecure readonly http server on a different address, rather than apply the same address which users set by `KubeletConfiguration.Address`.

**Note:** It isn't an alternative to someday removing the insecure server.

## Motivation

Currently, by setting a positive value (port) to `readOnlyPort`, users can enable the insecure readonly http server in kubelet. This server exposes endpoints like

- `/pods` 
- `/stats`
- `/metrics`

This server binds the address which is the same as the address for secure serving server.

```go
	if kubeCfg.ReadOnlyPort > 0 {
		go k.ListenAndServeReadOnly(net.ParseIP(kubeCfg.Address), uint(kubeCfg.ReadOnlyPort), enableCAdvisorJSONEndpoints)
	}
```

Sometimes, users want to bind secure serving server on `0.0.0.0`, but bind insecure serving server on `127.0.0.1` for giving access to the local agents.

In order to meet this requirement, we'd like to introduce a new configuration field to seprate the address into different address.

### Goals

- add a new configuration field `ReadOnlyBindAddress` to kubelet configuration
- add a new flag `--read-only-bind-address` to kubelet cmd

## Proposal

Introduce a new field `ReadOnlyBindAddress` to `KubeletConfiguration`, and the default value is "" (empty string). If the `ReadOnlyBindAddress` is not set by users, it will apply the value of `Address`. Otherwise, it will apply the value set by users.

Then, the insecure readonly server will bind on `ReadOnlyBindAddress`.

```go
	if kubeCfg.ReadOnlyPort > 0 {
		go k.ListenAndServeReadOnly(net.ParseIP(kubeCfg.ReadOnlyBindAddress), uint(kubeCfg.ReadOnlyPort), enableCAdvisorJSONEndpoints)
	}
```

## Design Details

### Test Plan

- Test if the value is valid ip
- Test if the insecure server is binding on address set via `Address` when `ReadOnlyBindAddress` is not set.
- Test if the insecure server is binding on address set via `ReadOnlyBindAddress` when `ReadOnlyBindAddress` is not set.

