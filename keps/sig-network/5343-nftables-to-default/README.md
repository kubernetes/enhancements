# KEP-5343: Make nftables the default kube-proxy backend

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Kernel support](#kernel-support)
  - [Stability and performance](#stability-and-performance)
  - [Feature compatibility](#feature-compatibility)
  - [Transitioning the default proxy <code>mode</code> in existing clusters](#transitioning-the-default-proxy-mode-in-existing-clusters)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (1.37)](#alpha-137)
    - [Beta (1.39)](#beta-139)
    - [GA (1.40)](#ga-140)
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

[KEP-3866], which introduced the `nftables` kube-proxy backend, noted
that we would probably eventually want to make it the default backend,
as part of some other KEP. This is that KEP.

[KEP-3866]: https://github.com/kubernetes/enhancements/issues/3866

## Motivation

### Goals

- Declare `nftables` to be the default backend.

- Avoid breaking existing users.

### Non-Goals

None

## Proposal

When we created the `nftables` backend, we assumed it would
*eventually* be a better choice for everyone than the `iptables` or
`ipvs` backend. However, there were a few reasons for not changing the
default right away.

### Kernel support

The `nftables` backend requires a much newer kernel (5.13) than the
`iptables` and `ipvs` backends.

There is no official minimum required kernel version for Kubernetes,
though `kubeadm` will warn if you are using a kernel that is both
"old" (older than 6.0 currently) and not under LTS upstream. Currently
it allows 5.4, 5.10, 5.15, and 6.0 or later, though the last update to
that list was 9 months ago (September 2025), and 5.4 has ended LTS
since then. 5.10 and 5.15 will end LTS in December 2026, so assuming
we update the list again at that point, Kubernetes 1.39 would
semi-officially only support kernels new enough to support the
`nftables` backend.

In terms of distributions:

  - Debian 12 (bookworm) has a 6.1 kernel (and 11 will be out of LTS
    by then anyway).

  - Ubuntu 22.04 has a 5.15 kernel (and will be the oldest
    _fully_-supported release at that point).

  - RHEL 9 nominally has a 5.14 kernel, though it's probably
    effectively newer than 6.1 anyway. (RHEL 8's nominally 4.18 kernel
    probably does not have enough backports to be able to support
    `nftables` mode.)

  - SLE15 SP4 and SP5 have 5.14, and SP6 and SLE16 have 6.x kernels.

Thus, it seems reasonable to assume that as of Kubernetes 1.39, _most_
users will have a new-enough kernel to support `nftables`.

### Stability and performance

The new `nftables` backend needed to get some use to convince us that
it was stable and performant enough to replace `iptables`. At this
point, with it having been GA for 3 releases, we have gotten enough
bug reports to know that people are using it, and few enough to be
confident that it mostly works.

There have been some performance fixes since GA. One problem we know
existed in the past that may still be there is that it definitely used
to take a long time to start up a new node in a cluster with many
services/endpoints. It is possible that [kubernetes #135800] and
[kubernetes #136499] may have fixed this. (They definitely fixed a
closely-related issue.) If not, then we should probably fix it to do
very large syncs in multiple stages rather than all in a single
transaction.

The biggest remaining stability problem is the fact that `nft` itself
crashes in some circumstances, notably when it encounters rules
created by someone else using a much newer version of `nft` than the
kube-proxy image contains. We are working on fixing this now by
building our own patched version of nft that should be more stable
([kubernetes #136786]). (We can't simply switch to using a newer
version of nft, because then we would be inflicting crashes on users
in the opposite situation, with a host filesystem containing a much
older `nft`.)

[kubernetes #135800]: https://github.com/kubernetes/kubernetes/pull/135800
[kubernetes #136499]: https://github.com/kubernetes/kubernetes/pull/136499
[kubernetes #136786]: https://github.com/kubernetes/kubernetes/issues/136786

### Feature compatibility

Another thing that may have prevented some people from migrating to
`nftables` mode is the lack of support for localhost NodePorts. This
is being addressed by [KEP-6032], which is expected to be Beta in 1.38
and GA in 1.40.

[KEP-6032]: https://github.com/kubernetes/enhancements/issues/6032

### Transitioning the default proxy `mode` in existing clusters

Many users presumably run kube-proxy with no `--proxy-mode` argument
or config-file `mode`. We do not want to switch them from `iptables`
to `nftables` without warning.

Starting in 1.37, we will warn users (via logs and events) when the
proxy mode gets defaulted to `iptables` (rather than having been
explicitly set to `iptables`). After enough releases of this, we can
switch the default.

(Alternatively, we could remove the default altogether, and say that
in the future, everybody will have to always explicitly specify the
proxy mode that they want. That avoids having anyone get accidentally
switched to `nftables`, but it means they accidentally completely
break their cluster instead, which is not an improvement.)

### Risks and Mitigations

The big risk of making `nftables` the default is that it might turn
out to not be ready. Of course, if this turned out to be the case, we
could just un-promote it and go back to `iptables` as default. (The
mitigation is the fact that we will continue to support the `iptables`
backend for a while still.)

## Design Details

See [Graduation Criteria](#graduation-criteria) for the timeline.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None

##### Unit tests

Probably none; there's not going to be much new/modified code here.

##### Integration tests

None; kube-proxy doesn't have integration tests.

##### e2e tests

We could possibly do further testing of switching kube-proxy modes,
but we may not need to.

### Graduation Criteria

This isn't really Alpha/Beta/GA, but let's pretend it is anyway just
to structure the work.

#### Alpha (1.37)

  - Begin warning users who get `iptables` by default that the default
    mode will switch to `nftables` in 1.40 so they should request
    `iptables` explicitly. (Or migrate now.)

      - Update kubeadm to specify `iptables` explicitly in new configs
        if it's not already doing that.

  - ([KEP-5495] should declare the `ipvs` backend to be deprecated.)

#### Beta (1.39)

  - Update kubeadm's `system-validator` to no longer accept 5.4 and
    5.10 kernels, which will now be out of LTS, but to accept 5.14 and
    5.15 (which aren't LTS upstream but are used by still-supported
    LTS distros).

  - Update the kube-proxy docs to say something more like "nftables
    doesn't work with very old kernels" rather than "nftables requires
    a very recent kernel".

  - ([KEP-6032] should have a Beta implementation of localhost
    nodeports for `nftables` by now.)

#### GA (1.40)

  - Change the default proxy mode to `nftables`.

      - This will be pointed out in an action-required release note.

  - Make `iptables` log about the fact that it is no longer the
    default and that `nftables` is now preferred.

  - Update kubeadm to default to `nftables`.

  - ([KEP-5495] should update the `KubeProxyIPVS` feature gate to
    `Default: false`.)

  - Update the docs to reflect that `nftables` is the default,
    `iptables` is legacy, and `ipvs` is on the way out.

[KEP-5495]: https://github.com/kubernetes/enhancements/issues/5495
[KEP-6032]: https://github.com/kubernetes/enhancements/issues/6032

### Upgrade / Downgrade Strategy

Users who already explicitly set the proxy mode will see no change on
upgrade or downgrade. Hopefully, by the time the default actually
changes, *everybody* will be explicitly setting the proxy mode.

For people who ignore the warnings and the release notes and update to
1.40 without explicitly setting the proxy mode, they will have a
change in behavior on upgrade, and a reversion to that change on
rollback. But, "don't do that then".

### Version Skew Strategy

N/A: kube-proxy is the only component that is changing. While it is
possible that some people will end up with a mix of nodes running
`iptables` and nodes running `nftables` during an upgrade, this
configuration is supported.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

The change to the default value will happen automatically at the given
release (planned for 1.40). There is no reason to put the change
behind a feature gate, since no functionality is being added or
removed: anyone who wants to "opt in" to the new default value before
the default changes can just set `mode: nftables` explicitly, and
anyone who wants to "opt out" of the new default can just set `mode:
iptables` explicitly (before or after the default changes).

For anyone who is explicitly setting a `mode`, this KEP will have
absolutely no effect. The rest of this section and the next are only
considering the case of people who ignore the warnings and let their
cluster get switched from `iptables` to `nftables` automatically when
they upgrade to 1.40.

###### How can this feature be enabled / disabled in a live cluster?

It can't. The change to the default value happens automatically in
1.40.

###### Does enabling the feature change any default behavior?

The feature *is* a change to default behavior.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

If you downgrade back to before 1.40, the default will go back to
`iptables`. Alternatively, you can explicitly set `mode: iptables`
after upgrading to undo the effect of the defaults change.

###### What happens if we reenable the feature if it was previously rolled back?

As expected.

###### Are there any tests for feature enablement/disablement?

No, though we have manually tested changing the proxy mode to/from
`nftables` on a running node in the past.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

If someone running on a kernel that does not support kube-proxy in
`nftables` mode upgrades to 1.40 without having explicitly set `mode:
iptables` in their config, then kube-proxy will try to start in
`nftables` mode and fail, leaving the node broken. (Again, "don't do
that then".)

In theory, we could have kube-proxy try to detect this case, and have
the default mode be `iptables` on older hosts and `nftables` on newer
ones, but that would be confusing for users and hard to document
clearly.

###### What specific metrics should inform a rollback?

If the `nftables` mode is not working, it is likely that the cluster
will fail catastrophically (e.g. because of `kubernetes.default` not
being accessible), and metrics will be beside the point.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No new testing has been done since the mode-changing testing that was
done when the `nftables` backend was first added.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

Switching from `iptables` mode to `nftables` mode means that some of
the CLI and config options the user is passing to kube-proxy might now
have no effect (and the user is presumably not setting the
corresponding `nftables`-specific options).

We give some notes on [migrating from `iptables` mode to `nftables`]
in the online documentation, which points out other things that may
not work right if the proxy mode gets changed out from under the user.

[migrating from `iptables` mode to `nftables`]: https://kubernetes.io/docs/reference/networking/virtual-ips/#migrating-from-iptables-mode-to-nftables

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The changed default value is automatically in effect for all clusters
after the release in which it changes.

An operator can determine whether a cluster will be / was affected by
the changed default by examining the kube-proxy configuration to see
if a mode is explicitly specified or not.

###### How can someone using this feature know that it is working for their instance?

N/A. Anybody who knows that the change in behavior is coming should
know that they don't want to allow the default to be changed out from
under them, and thus they should explicitly set a kube-proxy mode to
prevent that from happening. (And they can know that *that* is working
by the fact that kube-proxy will no longer log an event at them.)

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

We could add a metric for "number of warnings logged about the fact
that you're not explicitly setting `mode`", but anybody who would look
at metrics ought to see the event anyway?

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

No; switching from `iptables` to `nftables` should result in less CPU
and RAM being used.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A / the same as kube-proxy already does

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

Changing the default value is inherently risky, but we don't want
people getting stuck with the `iptables` backend forever.

## Alternatives

None.
