---
title: structured output
authors:
  - "@ibelle"
owning-sig: sig-cli
participating-sigs:
  - sig-cli
reviewers:
  - TBD
  - "@pwittrock"
approvers:
  - TBD
  - "@pwittrock"
creation-date: 2019-12-05
last-updated: 2019-12-05
status: provisional
see-also:
  - n/a
replaces:
  - n/a
superseded-by:
  - n/a
---

# Kubernetes Structured Output

## Table of Contents
<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
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

Define a protocol for emitting machine parseable structured output format (e.g. `-o structured`) from Kubernetes project tools.
This protocol would provide an API for better command integration & composition of tools in the Kubernetes ecosystem.

## Motivation

While many Kubernetes project tools provide for structured output as JSON or YAML (e.g. `-o json` or `-o yaml`) this is insufficient for
integrating with other tools in the ecosystem, like IDE's and higher-level CLI tool chains that wrap multiple project specific tools. These other
tools often require information beyond pure KRM object output, including:

* Clear separation of resource object output vs. non-resource (un-structured) output
* Details on errors that may occur.
* Execution trace for errors that may occur.
* Log level at which a message was generate (e.g. debug, info, critical etc.)

Encapsulating all of this data in a single structured format will greatly reduce overhead of programmatically parsing and interpreting
the output of the various Kubernetes project tools.

### Goals

* Define a schema for writing structured output to both stdout and stderr that allows all CLI tools across the Kubernetes ecosystem to be more easily integrated into programmatic contexts: IDE's, automated scripts, higher-level CLI tool chains etc.
* Define usage semantics for how Kubernetes tools should use this schema to generate their output.
* Define a standard output flag option which all Kubernetes CLI tools should support to enable structured output.

### Non-Goals
* Replace any existing output options or flags.

## Proposal

Details of protocol specification and implementation are captured as api-conventions under the kustomize repo here:

- [Structured Output for Kubernetes CLI Tools](https://github.com/ibelle/kustomize/blob/master/cmd/config/docs/api-conventions/structured_output.md#structured-output-for-kubernetes-cli-tools)

## Implementation History
- 12-05-2019 KEP Created