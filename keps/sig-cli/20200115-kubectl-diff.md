---
title: kubectl-diff
authors:
  - "@apelisse"
  - "@julianvmodesto"
owning-sig: sig-cli
participating-sigs:
  - sig-api-machinery
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2020-01-15
last-updated: 2020-03-23
status: implemented
see-also:
  - "/keps/sig-api-machinery/0015-dry-run.md"
  - "/keps/sig-api-machinery/0006-apply.md"
---

# kubectl-diff

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [x] KEP approvers have set the KEP status to `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

`kubectl diff` is a feature that has been available in Kubernetes for a
few releases now (since v1.13) and is planning to go GA in 1.18. It
provides a very simple UX to display how the changes to a configuration
will affect the state of the cluster.

## Motivation

Being able to proactively understand how configuration changes affect
the state of the cluster is a key piece of declarative configuration
management. It allows integration into tools, but is equally important
in manual deployments. Without diff, understanding the exact
consequences of one's changes can be hard to grasp, especially when
exposed to defaults and configurable mutating webhooks. It also provides
a basic validation since the command will fail if something is rejected
by the server.

### Goals

The goal of this KEP is mostly retroactive, since the feature has
mostly been implemented before this process existed.

## Proposal

We initially considered multiple ways to "diff" resources, since we can
consider multiple cases:

1. There is the applied configuration, the one in the file,

2. There is the last-applied-configuration, the one in the file that we applied the last time,

3. There is the "current live" configuration, the one describing the existing state of the server,

4. There is the "future" configuration of the server, if we did apply the configuration.

All the combinations of these diffs initially sounded useful, but after
providing such a variant in `kubectl alpha diff`, we realized that the
most relevant is actually "current live" configuration -> "future" configuration.
This describes exactly the changes that are happening on the cluster.

### Risks and Mitigations

Risks are extremely limited, but can exist since the feature depends on
server-side dry-run. If a bug in dry-run ends-up writing a resource by
accident, then `kubectl diff` could be writing to that resource also by
accident.

## Design Details

In order to diff the current configuration with the future
configuration, we do the following:

1. get the current object from the server,
1. prepare an apply request that is sent with the `dryRun` flag and returns
  the hypothetical object,
1. carefully insert the resourceVersion in the patch that we send, in
  order to make sure that we're patching and diffing the exact same
  object version we've seen to avoid diffing unrelated objects,
1. save each of these objects in files with specific names using
  their GVK to avoid collisions,
1. diff these files either using `diff(1)`, or using the binary provided
  in `KUBECTL_EXTERNAL_DIFF`.

### Test Plan

In addition to unit tests, there will be integration tests using the
command-line integration test suite:

- [x] [Test `kubectl diff` for multiple resources with the same name](https://testgrid.k8s.io/presubmits-kubernetes-blocking#pull-kubernetes-integration&include-filter-by-regex=test-cmd.run_kubectl_diff_same_names)

### Graduation Criteria

#### Alpha -> Beta Graduation

- [x] At least 2 release cycles pass to gather feedback and bug reports during
  real-world usage
- [x] End-user documentation is written
- [x] The client-side dry-run used to calculate the diff is replaced with the
  server-side dry-run feature to improve correctness and accuracy for this
  feature
- [x] The dependent API server-side dry-run feature is released to beta

#### Beta -> GA Graduation

- [x] At least 2 release cycles pass to gather feedback and bug reports during
  real-world usage
- [ ] Integration tests are in Testgrid and linked in KEP
- [ ] Documentation exists for user stories
- [ ] The dependent API server-side dry-run feature is released to GA

### Upgrade / Downgrade Strategy

This section is not relevant because this is a client-side component only.

### Version Skew Strategy

To check what the merged live object would look like, the `kubectl diff`
command relies on server-side dry-run support for the resource.

If an API server has disabled server-side dry-run or the API server was
downgraded to a version without server-side dry-run, then `kubectl diff` will
fail to get a merged version of the object and not display a diff.

## Implementation History

- *2020-01*: Added KEP
- *2019-01*: Promoted from alpha to beta in 1.13
- *2017-12*: Released as alpha in 1.9
