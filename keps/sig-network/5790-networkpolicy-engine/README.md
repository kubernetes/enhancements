# KEP-5790: NetworkPolicy Controller Name Label

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Policies on Multus Interfaces](#policies-on-multus-interfaces)
    - [OVN-Kubernetes User Defined Networks](#ovn-kubernetes-user-defined-networks)
    - [Other Network Plugins](#other-network-plugins)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [<code>policy-controller-name</code> Label](#policy-controller-name-label)
  - [<code>policy-controller-name: none</code>?](#policy-controller-name-none)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
    - [Unit tests](#unit-tests)
    - [Integration tests](#integration-tests)
    - [e2e tests](#e2e-tests)
    - [ClusterNetworkPolicy conformance tests](#clusternetworkpolicy-conformance-tests)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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

In experimental / non-standard multi-network Kubernetes clusters, the
secondary networks often have network policy enforcement needs that
are very similar to the functionality provided by the standard
NetworkPolicy APIs. It's possible to create an alternative
"[`MultiNetworkPolicy` API]" that just mirrors NetworkPolicy, but this
is annoying for multiple reasons (and implies also needing
"`ClusterMultiNetworkPolicy`"). But using plain NetworkPolicy or
ClusterNetworkPolicy objects to describe the semantics of
non-cluster-default pod networks is non-portable and problematic if
other components in the cluster don't understand what's going on.

By creating a way to mark a NetworkPolicy or ClusterNetworkPolicy as
"non-standard", we can allow people to reuse the existing types with
slightly-altered semantics, without confusing other implementations
that don't understand/expect those semantics.

[`MultiNetworkPolicy` API]: https://github.com/k8snetworkplumbingwg/multi-networkpolicy

## Motivation

### Goals

- Allow marking a NetworkPolicy or ClusterNetworkPolicy as belonging
  to a particular implementation, such that other implementations (and
  tools, such as `policy-assistant`) will know to ignore it.

- Update `kube-network-policies` to ignore policies marked as
  non-standard.

### Non-Goals

- Implementing any non-standard behavior in `kube-network-policies`.

- Changing the behavior of any core Kubernetes components (for example,
  making `kubectl get np` show the new information by default).

## Proposal

### User Stories

#### Policies on Multus Interfaces

As a user, I want to use Multus to add interfaces to some pods, and
then enforce policy on those pods. Currently this is possible with
something like [multi-networkpolicy-nftables], which uses its own
`MultiNetworkPolicy` CRD that is mostly just a clone of NetworkPolicy,
but there is currently no multi-network equivalent of
ClusterNetworkPolicy, and anyway, it's awkward to clone the types just
to add multi-network semantics, when we want things to otherwise be
exactly the same as NP/CNP for familiarity.

So in the future, it would nice if I could create NetworkPolicies and
ClusterNetworkPolicies for the Multus interfaces, and have both
multi-networkpolicy-nftables and the pod network implementation
understand which policies are meant to apply to the pod network, and
which are meant to apply to Multus interfaces.

[multi-networkpolicy-nftables]: https://github.com/k8snetworkplumbingwg/multi-networkpolicy-nftables

#### OVN-Kubernetes User Defined Networks

As a user, I would like to use [OVN-Kubernetes User Defined Networks]
and have NetworkPolicy-like and ClusterNetworkPolicy-like policies on
those networks. (As with the Multus example, this feature currently
supports NetworkPolicy-like semantics, using the same
MultiNetworkPolicy CRD, but does not currently support
ClusterNetworkPolicy.)

[OVN-Kubernetes User Defined Networks]: https://ovn-kubernetes.io/features/user-defined-networks/user-defined-networks/

#### Other Network Plugins

As an OpenShift developer, I would like to ensure that whatever APIs
Multus and OVN-Kubernetes offer for non-standard network policy do not
confuse other network plugins. For example, if a user is running
Calico in an OpenShift cluster, and tries to apply policies to Multus
interfaces (or accidentally ends up with some UDN policies installed),
Calico should not mistakenly try to apply those policies to the
cluster-default pod network.

### Risks and Mitigations

The big risk is that users will try to use the new feature in a
cluster where some or all components don't implement it yet. For
example, a user might try to use a NetworkPolicy with the new label to
specify policies for pods connected via a secondary network created by
Multus, but if the primary pod network implementation doesn't know to
ignore those policies, then the effect may not be what the user
intended. There is no easy mitigation for this, other than that the
user needs to be careful to not do that.

## Design Details

### `policy-controller-name` Label

We will add a new well-known label, `networking.k8s.io/policy-controller-name`,
which can be added to NetworkPolicy or ClusterNetworkPolicy objects.
(Compare `service.kubernetes.io/service-proxy-name`.)

```
<<[UNRESOLVED label name ]>>

Is that name OK?

CNP uses the API group `policy.networking.k8s.io`, so my first thought
here was to use that, but I feel like it makes more sense with
"policy" in the post-"/" part of the name, and I didn't want to
duplicate it.

<<[/UNRESOLVED]>>
```

```
<<[UNRESOLVED label value ]>>

What should the values be? e.g., `ovn-kubernetes` or
`ovn-kubernetes.k8s.ovn.org`

<<[/UNRESOLVED]>>
```

When that label is set, NetworkPolicy / ClusterNetworkPolicy
implementations other than the one named by the label's value should
_completely ignore_ the NetworkPolicy / ClusterNetworkPolicy object,
as though the object _did not exist at all_. This is different from
"the object exists but has no enforceable rules". For example:

  - If a NetworkPolicy that selects a Pod has a
    `networking.k8s.io/policy-controller-name` label that the implementation
    does not recognize, then that policy must not cause the
    implementation to treat the Pod as "isolated".

  - If there are 2 ClusterNetworkPolicy objects with the same `tier`
    and `priority` which both could apply to the same packet, but one
    object has a `networking.k8s.io/policy-controller-name` value that the
    implementation does not recognize, and the other object has no
    `networking.k8s.io/policy-controller-name` label, then the implementation
    _must_ always apply the rules from the latter policy, even if the
    implementation's normal rule for resolving `priority` conflicts
    would have picked the former policy.

(A simple way to implement this correctly is to just create an
informer that filters out objects with unknown values of the label, so
that the implementation never even sees the policies it needs to
ignore.)

Likewise, policy management or debugging tools should ignore (or at
least be wary of) policies marked with this label, as they may not
have the effects that the tool thinks they do.

When a policy is marked as being specific to a particular
implementation, that implementation may give the policy arbtirary
alternative semantics (possibly controlled by additional
implementation-specific labels or annotations). However,
implementations should be careful to not change the semantics of
marked NetworkPolices and ClusterNetworkPolicies in ways that are
likely to lead to user confusion.

```
<<[UNRESOLVED non-standardness ]>>

Should we be more specific about what you shouldn't do? I don't want
to say "this is only for multi-network"...

<<[/UNRESOLVED]>>
```

### `policy-controller-name: none`?

There is an open issue ([kubernetes #112560]) suggesting that it
should be possible to disable a NetworkPolicy. Similarly, there is an
issue suggesting a "dry-run" mode for ClusterNetworkPolicy
([network-policy-api #230]), where it was also suggested that, given
that there is no standard for logging/auditing in NP/CNP anyway,
perhaps it could be implemented as the combination of a "disable" flag
plus a vendor-specific annotation.

So, perhaps we could make `networking.k8s.io/policy-controller-name: none` be a
standard way of disabling a NetworkPolicy / ClusterNetworkPolicy?

```
<<[UNRESOLVED policy-controller-name: none ]>>

Do we want this? Is "none" right or should it be "disabled", etc?

<<[/UNRESOLVED]>>
```

[kubernetes #112560]: https://github.com/kubernetes/kubernetes/issues/112560
[network-policy-api #230]: https://github.com/kubernetes-sigs/network-policy-api/issues/230

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

#### Prerequisite testing updates

N/A

#### Unit tests

None. (There is no implementation of NetworkPolicy in k/k, so there is
nothing to unit test beyond "labels work".)

#### Integration tests

None. (There is no implementation of NetworkPolicy in k/k, so there is
nothing to integration-test with.)

#### e2e tests

A test will be added to the NetworkPolicy e2e tests, ensuring that the
NetworkPolicy implementation ignores policies with the label set to a
nonsense value.

```
<<[UNRESOLVED test labeling ]>>

Since this feature is not implemented by any in-tree component, it
seems like it doesn't make sense to add a feature gate for it. But
that may be the only supported way to tag an e2e test as "Alpha" these
days? OTOH, there are no jobs that run both `[Feature:NetworkPolicy]`
and `Alpha` or `Beta` anyway, so tagging the test with its stability
level would simply ensure that nobody ran it.

<<[/UNRESOLVED]>>
```

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

#### ClusterNetworkPolicy conformance tests

As with the NetworkPolicy e2e test, we will add a test to the
ClusterNetworkPolicy conformance suite, ensuring that the
ClusterNetworkPolicy implementation ignores policies with the label
set to a nonsense value.

### Graduation Criteria

#### Alpha

- Label defined in `k8s.io/api/networking/v1`
- e2e tests added

#### Beta

- Multiple implementations

#### GA

- Enough time has passed since Beta

```
<<[UNRESOLVED graduation ]>>

I'm not sure how long we need to wait between Alpha and Beta, and Beta
and GA.

In particular, waiting N releases to handle skew/downgrade issues
doesn't necessarily make sense, since the implementations are
out-of-tree and may not track k8s releases anyway.

<<[/UNRESOLVED]>>
```

### Upgrade / Downgrade Strategy

N/A. The feature is only implemented by external components, and there
are no real upgrade/downgrade considerations anyway, beyond the usual
"don't downgrade to a version that will do bad things with objects
that are using this feature if you have objects using this feature".

### Version Skew Strategy

N/A; the feature is implemented entirely in a single (external) component.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

N/A. The feature is only implemented by external components, and it is
up to them how/if it can be enabled/disabled.

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Assuming it can be disabled at all, yes.

###### What happens if we reenable the feature if it was previously rolled back?

Nothing unexpected.

###### Are there any tests for feature enablement/disablement?

N/A. The feature is only implemented by external components, and it is
up to them how/if it can be enabled/disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rolling back the NetworkPolicy implementation to a version that
doesn't understand the label, while having objects that make use of
the label, may cause those objects to be misinterpreted. In the worst
case, a cluster might contain policies that would appear to block all
traffic, which are not intended to actually block all traffic, but
which will end up blocking all traffic.

###### What specific metrics should inform a rollback?

N/A / implementation-specific.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

By examining all NetworkPolicy and ClusterNetworkPolicy objects.

###### How can someone using this feature know that it is working for their instance?

- [X] Other (treat as last resort)
  - Details: The policy is interpreted correctly by the expected
    implementation and not interpreted by other implementations.

NetworkPolicy does not currently have a `.status` field. (It briefly
did in the past, but it was removed when the associated KEP was
withdrawn.)

In theory, we could re-add `.status` and `.status.conditions` and
recommend that implementations add a condition indicating when they
are enforcing a policy. This would allow people to detect when a
labeled policy _was_ being enforced by the expected implementation, but would
not allow them to detect when it was mistakenly also being enforced
(incorrectly) by a non-compliant implementation.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Implementing the feature should have no impact on SLOs.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A / no change; the feature should not impact the health of the
service implementing it.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A. The feature is only implemented by external components.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

It requires a NetworkPolicy implementation to implement it.

### Scalability

###### Will enabling / using this feature result in any new API calls?

It shouldn't.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No. (It will result in some users adding new API objects which use the
feature.)

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Implementation-specific, but unlikely to be any different from now.

###### What are other known failure modes?

The only likely failure mode is having two NetworkPolicy /
ClusterNetworkPolicy implementations in a cluster, where only 1
correctly implements the feature. (But such a configuration would
never have worked _without_ this feature either, so this would only
happen if an administrator was over-eager about trying to use the
feature with un-upgraded components.)

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A.

## Implementation History

- Initial proposal: 2026-01-13

## Drawbacks

Labels and annotations are not good APIs. Also, the primary use case
for this API is hopefully temporary, in that there should eventually
be better explicit multi-network support (which will presumably
include explicit multi-network support in NetworkPolicy).

However, the proposed feature is very small, and won't be a burden for
implementations to maintain in the future even if a better
multi-network API is added. Also, there is precedent, in the form of
`service.kubernetes.io/service-proxy-name`.

## Alternatives

The alternative would be an explicitly-multi-network-aware API, which
could happen in one of two ways.

First, we could wait for [KEP-3698] "Multi-Network" to be completed,
which is expected to include appropriate APIs for Services and
NetworkPolicies on secondary networks. However, the Multi-Network
subproject has not made much progress on finishing this KEP (which is
currently closed, having by lifecycled out by the bot), so this is not
likely to happen any time soon.

An alternative approach would be to add an API for _referring to_
secondary networks without actually having a way to _define_ them. The
Multi-Network subproject is currently working on a [`NetworkClass`
API] which would allow this. The user could create a `NetworkClass`
object indicating the Group/Version/Kind of an implementation-specific
secondary network type, and then we could add API to NetworkPolicy and
ClusterNetworkPolicy to allow you to pick a network by NetworkClass,
optional namespace, and name.

While this API could likely become stable much sooner than a full
Multi-Network API, it is not clear at this point if we actually want
that API.

[`NetworkClass` API]: https://docs.google.com/document/d/1R2LDPZzstUKJVC5old9TfR6Db6gp-eAcpCcZtqG52Ac/edit
