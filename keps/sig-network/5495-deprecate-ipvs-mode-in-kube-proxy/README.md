# KEP-5495: Deprecate IPVS mode in kube-proxy

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Stage 1 (1.35)](#stage-1-135)
    - [Stage 2 (1.37)](#stage-2-137)
    - [Stage 3 (1.40)](#stage-3-140)
    - [Stage 4 (1.43)](#stage-4-143)
    - [Cleanup (1.46)](#cleanup-146)
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

## Summary

This KEP proposes deprecation of `ipvs` in kube-proxy.

## Motivation

At time of writing, kube-proxy has four supported backends (`iptables`, `ipvs`, `nftables`, and `winkernel`).

sig-network currently lacks maintainers who are familar with the `ipvs` backend code, and as such, has been encouraging
users who report `ipvs` bugs to move to using the `nftables` backend mode, where they can. (ie: [first example], [second example])

`ipvs` was introduced in [KEP-265] to solve performance problems in large clusters.
[KEP-3866] was created to introduce a new `nftables` mode to kube-proxy. The goal of this new backend mode
has always been to eventually replace `ipvs` and `iptables`[^1], as it solve the performance issues of iptables
and already solves many of the bugs present in the `ipvs` mode.

[First example]: https://github.com/kubernetes/kubernetes/issues/132689#issuecomment-3031585314
[second example]: https://github.com/kubernetes/kubernetes/issues/132068#issuecomment-2945169346
[^1]: See https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/3866-nftables-proxy/README.md#we-will-hopefully-be-able-to-trade-2-supported-backends-for-1

[KEP-265]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/265-ipvs-based-load-balancing/README.md
[KEP-3866]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/3866-nftables-proxy/README.md

### Goals

- Deprecate and eventually remove the `ipvs` mode of kube-proxy

### Non-Goals

N/A

## Proposal

First, we will make the `ipvs` backend log a warning when it is run,
and update the documentation to reflect its deprecation. (This
happened in 1.35.)

We had previously talked about moving the `ipvs` backend to a separate
repo (e.g. `kubernetes-sigs/kube-proxy-ipvs`), and this was going to
be handled in [KEP-5343]. However, we would have no intention of
maintaining this repo, and wouldn't want to be responsible for doing
security fixes for it, so we would essentially have to create the repo
and then immediately archive it.

Additionally, since we started this deprecation, we've realized that
the `nftables` kernel requirement situation isn't quite as dire as we
had originally thought, and all of the kernels that are too old to
support kube-proxy's `nftables` mode will be out of LTS by the end of
2026. At that point, there will be even less of an argument for
continuing to run `ipvs` rather than switching to `nftables`.

Thus, we will simply remove the IPVS proxy from the tree, and not copy
it anywhere else. This will be done via a `Deprecated` feature gate.

### Risks and Mitigations

N/A

## Design Details

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

N/A - This KEP is only removing code, not adding new code.

##### Integration tests

N/A - This KEP is only removing code, not adding new code.

##### e2e tests

N/A - This KEP is only removing code, not adding new code.

### Graduation Criteria

#### Stage 1 (1.35)

- The Kubernetes web site has been updated with deprecation notices for the `ipvs` mode of kube-proxy
- Kube-proxy prints a warning when a user starts kube-proxy in `ipvs` mode
- All nftables-mode bugfixes have been backported to 1.34 and 1.33, to ensure that `ipvs` users on older releases can still migrate to `nftables`.

#### Stage 2 (1.37)

- We have added a `KubeProxyIPVS` feature gate, `Default: true,
  Prerelease: featuregate.GA`.

- Docs and warning messages are updated to indicate that the plan is
  for the feature gate to go `Default: false` in 1.40 and
  `LockToDefault: true` in 1.43 (after which `ipvs` mode will no
  longer be available).

- We've done a blog post with information about migrating from `ipvs`
  to `iptables` or `nftables`, including an explanation of why IPVS
  schedulers aren't actually useful in Kubernetes, which a lot of
  `ipvs` users don't seem to realize.

#### Stage 3 (1.40)

- The feature gate is flipped to `Default: false`, and the docs
  updated. (If you try to run kube-proxy in `ipvs` mode without
  overriding the feature gate, kube-proxy will exit with an error
  listing the valid modes (`iptables` and `nftables`).)

#### Stage 4 (1.43)

- The feature gate is flipped to `LockToDefault: false` and
  `pkg/proxy/ipvs` is removed. Remaining mentions of IPVS mode in the
  docs are removed.

#### Cleanup (1.46)

- The feature gate is removed.

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

N/A

###### Does enabling the feature change any default behavior?

N/A

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

N/A

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

N/A

### Rollout, Upgrade and Rollback Planning

N/A

###### How can a rollout or rollback fail? Can it impact already running workloads?

N/A

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

The rollout *is* a deprecation/removal of features.

We should probably keep the IPVS CLI/config options as no-ops (with
warnings), in case anyone migrates to another backend but forgets to
remove some of the IPVS flags and doesn't notice.

### Monitoring Requirements

N/A

###### How can an operator determine if the feature is in use by workloads?

N/A

###### How can someone using this feature know that it is working for their instance?

N/A

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

N/A

### Scalability

N/A

###### Will enabling / using this feature result in any new API calls?

N/A

###### Will enabling / using this feature result in introducing new API types?

N/A

###### Will enabling / using this feature result in any new calls to the cloud provider?

N/A

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

N/A

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

N/A

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

N/A

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

N/A

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

N/A

## Drawbacks

Users may be running version of the kernel which isn't new enough to support the `nftables` backend mode in kube-proxy.

## Alternatives

Getting active maintainers for `ipvs` may be a short term alternative, see [The ipvs mode of kube-proxy will not save us] (from [KEP-3866]) for details

[The ipvs mode of kube-proxy will not save us]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/3866-nftables-proxy/README.md#the-ipvs-mode-of-kube-proxy-will-not-save-us

## Infrastructure Needed (Optional)

N/A
