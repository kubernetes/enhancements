# KEP-3178: Cleaning up IPTables Chain Ownership

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
    - [The Chains in Question and Their Purpose](#the-chains-in-question-and-their-purpose)
    - [Eliminating <code>KUBE-MARK-DROP</code>](#eliminating-)
    - [External Users of Kubelet's IPTables Chains](#external-users-of-kubelets-iptables-chains)
    - [<code>iptables-wrapper</code>](#)
    - [Martian Packet Blocking](#martian-packet-blocking)
- [Design Details](#design-details)
  - [Implementation](#implementation)
    - [Pre-Alpha](#pre-alpha)
    - [Alpha (1.25)](#alpha-125)
    - [Beta (1.27)](#beta-127)
    - [GA (1.28?)](#ga-128)
    - [GA + 2](#ga--2)
    - [Indeterminate Future](#indeterminate-future)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
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
  - [Use a Feature Gate in Kube-Proxy As Well](#use-a-feature-gate-in-kube-proxy-as-well)
  - [Move <code>KUBE-MARK-DROP</code> to Kube-Proxy Rather Than Removing It](#move--to-kube-proxy-rather-than-removing-it)
  - [Remove <code>KUBE-MARK-MASQ</code> from Kube-Proxy and Let Kubelet Own It](#remove--from-kube-proxy-and-let-kubelet-own-it)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kubelet currently creates some IPTables chains at startup. In the past
it actually used some of them, but with [the removal of dockershim] it
no longer does. Additionally, for historical reasons, it creates some
IPTables chains that are duplicates of chains also created by
kube-proxy, and some that are only used by kube-proxy, but which
kube-proxy requires kubelet to create on its behalf. (The initial
comment in [kubernetes #82125] has more details on the history of how
we got to where we are now, or at least to where we were before
dockershim was removed.)

We should clean this up so that kubelet no longer unnecessarily
creates IPTables chains, and that kube-proxy creates all of the
IPTables chains it needs.

[the removal of dockershim]: ../../sig-node/2221-remove-dockershim
[kubernetes #82125]: https://github.com/kubernetes/kubernetes/issues/82125

## Motivation

### Goals

- Remove most IPTables chain management from kubelet.

    - Remove the `--iptables-drop-bit` and `--iptables-masquerade-bit`
      arguments from kubelet.

    - Kubelet will continue to create at least one IPTables chain as a
      hint to `iptables-wrapper` and other components that need to
      know whether the system is using `legacy` or `nft` iptables.

    - For security-backward-compatibility, kubelet will continue (for
      now) to create a rule to block "martian packets" that would
      otherwise be allowed by `route_localnet` (even though kubelet
      itself does not set `route_localnet` or expect it to be set).
      (See discussion in [Martian Packet
      Blocking](#martian-packet-blocking) below.)

- Update kube-proxy to no longer rely on chains created by kubelet:

    - Rewrite packet-dropping rules to not depend on a
      `KUBE-MARK-DROP` chain created by kubelet.

    - Kube-proxy should create its own copy of kubelet's "martian
      packet" fix, when running in a mode that needs that fix.

- Ensure that the proxy implementations in `kpng` get updated
  similarly.

- Document for the future that Kubernetes's IPTables chains (other
  than the IPTables mode "hint" chain) are meant for its internal use
  only, do not constitute API, and should not be relied on by external
  components.

- Ensure that users and third-party software that make assumptions
  about kubelet's and kube-proxy's IPTables rules have time to get
  updated for the changes in rule creation.

### Non-Goals

- Changing `KUBE-MARK-MASQ` and `KUBE-MARK-DROP` to use the connection
  mark or connection label rather than the packet label (as discussed
  in [kubernetes #78948]).

[kubernetes #78948]: https://github.com/kubernetes/kubernetes/issues/78948

## Proposal

### Notes/Constraints/Caveats

#### The Chains in Question and Their Purpose

Kubelet currently creates five IPTables chains; two to help with
masquerading packets, two to help with dropping packets, and one for
purely-internal purposes:

  - `KUBE-MARK-MASQ` and `KUBE-POSTROUTING`

    `KUBE-MARK-MASQ` marks packets as needing to be masqueraded (by
    setting a configurable bit of the "packet mark").
    `KUBE-POSTROUTING` checks the packet mark and calls `-j
    MASQUERADE` on the packets that were previously marked for
    masquerading.

    These chains were formerly used for HostPort handling in
    dockershim, but are no longer used by kubelet. Kube-proxy (in
    iptables or ipvs mode) creates identical copies of both of these
    chains, which it uses for service handling.

  - `KUBE-MARK-DROP` and `KUBE-FIREWALL`

    `KUBE-MARK-DROP` marks packets as needing to be dropped by setting
    a _different_ configurable bit on the packet mark. `KUBE-FIREWALL`
    checks the packet mark and calls `-j DROP` on the packets that
    were previously marked for dropping.

    These chains have always been created by kubelet, but were only
    ever used by kube-proxy.

    (`KUBE-FIREWALL` also contains a rule blocking certain "martian
    packets"; see below.)

  - `KUBE-KUBELET-CANARY`, which is used by the `utiliptables.Monitor`
    functionality to notice when the iptables rules have been flushed
    and kubelet needs to recreate its rules.

The reason that the `MARK` chains exist is because most of
kube-proxy's service-processing logic occurs in subchains of the
`OUTPUT` and `PREROUTING` chains of the `nat` table (because those are
the only places you can call the `DNAT` target from, to redirect
packets from a service IP to a pod IP), but neither masquerading nor
dropping packets can occur at that point in IPTables processing:
dropping packets can only occur from chains in the `filter` table,
while masquerading can only occur from the `POSTROUTING` chain of the
`nat` table (because the kernel can't know what IP to masquerade the
packet to until after it has decided what interface it will send it
out on, which necessarily can't happen until after it knows if it is
going to DNAT it). To work around this, when Kubernetes wants to
masquerade or drop a packet, it just marks it as needing to be
masqueraded or dropped later, and then one of the later chains handles
the actual masquerading/dropping.

This approach was not _necessary_; kube-proxy could have just
duplicated its matching logic between multiple IPTables chains, eg,
having a rule saying "a packet sent to `192.168.0.11:80` should be
redirected to `10.0.20.18:8080`" in one chain and a rule saying "a
packet whose original pre-NAT destination was `192.168.0.11:80` should
be masqueraded" in another.

But using `KUBE-MARK-MASQ` allows kube-proxy to keep its logic more
efficient, compact, and well-organized than it would be otherwise; it
can just create a pair of adjacent rules saying "a packet sent to
`192.168.0.11:80` should be redirected to `10.0.20.18:8080`, and
should also be masqueraded".

#### Eliminating `KUBE-MARK-DROP`

In theory, `KUBE-MARK-DROP` is useful for the same reason as
`KUBE-MARK-MASQ`, although in practice, it turns out to be
unnecessary. Kube-proxy currently drops packets in two cases:

  - When using the `LoadBalancerSourceRanges` feature on a service,
    any packet to that service that comes from outside the valid
    source IP ranges is dropped.

  - When using `Local` external traffic policy, if a connection to a
    service arrives on a node that has no local endpoints for that
    service, it is dropped.

In the `LoadBalancerSourceRanges` case, it would not be difficult to
avoid using `KUBE-MARK-DROP`. The current rule logic creates a
per-service `KUBE-FW-` chain that checks against each allowed source
range and forwards traffic from those ranges to the `KUBE-SVC-` or
`KUBE-XLB-` chain for the service. At the end of the chain, it calls
`KUBE-MARK-DROP` on any unmatched packets.

To do this without `KUBE-MARK-DROP`, we simply remove the
`KUBE-MARK-DROP` call at the end of the `KUBE-FW-` chain, and add a
new rule in the `filter` table, matching on the load balancer IP and
port, and calling `-j DROP`. Any traffic that matched one of the
allowed source ranges in the `KUBE-FW-` chain would have been DNAT'ed
to a pod IP by this point, so any packet that still has the original
load balancer IP as its destination when it reaches the `filter` table
must be a packet that was not accepted, so we can drop it.

In the external traffic policy case, things are even simpler; we can
just move the existing traffic-dropping rule from the `nat` table to
the `filter` table and call `DROP` directly rather than
`KUBE-MARK-DROP`, and we will get exactly the same effect. (Indeed, we
already do it that way when calling `REJECT`, so this will actually
make things more consistent.)

#### External Users of Kubelet's IPTables Chains

Some users or third-party software may expect that certain IPTables
chains always exist on nodes in a Kubernetes cluster.

For example, the [CNI portmap plugin] provides an
`"externalSetMarkChain"` option that is explicitly intended to be used
with `"KUBE-MARK-MASQ"`, to make the plugin use Kubernetes's iptables
rules instead of creating its own. Although in most cases kube-proxy
will also create `KUBE-MARK-MASQ`, kube-proxy may not start up until
after some other components, and some users may be running a network
plugin that has its own proxy implementation rather than using
kube-proxy. Thus, users may end up in a situation where some software
is trying to use a `KUBE-MARK-MASQ` chain that does not exist.

Most of these external components have simply copied kube-proxy's use
of `KUBE-MARK-MASQ` without understanding why it is used that way in
kube-proxy. But because they are generally doing much less IPTables
processing than kube-proxy does, they could fairly easily be rewritten
to not use the packet mark, and just have slightly-redundant
`PREROUTING` and `POSTROUTING` rules instead.

[CNI portmap plugin]: https://github.com/containernetworking/plugins/tree/master/plugins/meta/portmap

#### `iptables-wrapper`

Another problem is the [iptables-wrapper script] for detecting whether
the system is using iptables-legacy or iptables-nft. Currently it
assumes that kubelet will always have created _some_ iptables chains
before the wrapper script runs, and so it can decide which iptables
backend to use based on which one kubelet used. If kubelet stopped
creating chains entirely, then iptables-wrapper might find itself in a
situation where there were no chains in either set of tables.

To help this script (and other similar components), we should have
kubelet continue to always create at least one IPTables chain, with a
well-known name.

[iptables-wrapper script]: https://github.com/kubernetes-sigs/iptables-wrappers

#### Martian Packet Blocking

In `iptables` mode, kube-proxy sets the
`net.ipv4.conf.all.route_localnet` sysctl, so that it is possible to
connect to NodePort services via 127.0.0.1. (This is somewhat
controversial, and doesn't work under IPv6, but that's a story for
another KEP.) This creates a security hole ([kubernetes #90259]) and
so we add an iptables rule to block the insecure case while allowing
the "useful" case ([kubernetes #91569]). In keeping with historical
confusion around iptables rule ownership, this rule was added to
kubelet even though the behavior it is fixing is in kube-proxy.

Since kube-proxy is the one that is creating this problem, it ought to
be the one creating the fix for it as well, and we should make
kube-proxy create this filtering rule itself.

However, it is possible that users of other proxy implementations (or
hostport implementations) may be setting `route_localnet` based on
kube-proxy's example and may depend on the security provided by the
existing kubelet rule. Thus, we should probably have kubelet continue
to create this rule as well, at least for a while. (The rule is
idempotent, so it doesn't even necessarily require keeping kubelet's
definition and kube-proxy's in sync.) We can consider removing it
again at some point in the future.

[kubernetes #90259]: https://github.com/kubernetes/kubernetes/issues/90259
[kubernetes #91569]: https://github.com/kubernetes/kubernetes/pull/91569

## Design Details

### Implementation

We cannot remove `KUBE-MARK-DROP` from kubelet until we know that
kubelet cannot possibly be running against a version of kube-proxy
that requires it.

Thus, the process will be:

#### Pre-Alpha

Kubelet will begin creating a new `KUBE-IPTABLES-HINT` chain in the
`mangle` table, to be used as a hint to external components about
which iptables API the system is using. (We use the `mangle` chain,
not `nat` or `filter`, because given how the iptables API works, even
just checking for the existence of a chain is slow if the table it is
in is very large.)

(This happened in 1.24: [kubernetes #109059].)

To help ensure that external components have time to remove any
dependency on `KUBE-MARK-DROP` well before this feature goes Beta, we
will document the upcoming changes in a blog post and the next set of
release notes. (And perhaps other places?)

We will also document the new `KUBE-IPTABLES-HINT` chain and its
intended use, as well as the best practices for detecting the system
iptables mode in previous releases.

[kubernetes #109059]: https://github.com/kubernetes/kubernetes/pull/109059

#### Alpha (1.25)

Kube-proxy will be updated to not use `KUBE-MARK-DROP`, as described
above. (This change is unconditional; it is _not_ feature-gated,
because it is more of a cleanup/bugfix than a new feature.) We should
also ensure that kpng gets updated.

Kubelet's behavior will not change by default, but if you enable the
`IPTablesOwnershipCleanup` feature gate, then:

  1. It will stop creating `KUBE-MARK-DROP`, `KUBE-MARK-MASQ`,
     `KUBE-POSTROUTING`, and `KUBE-KUBELET-CANARY`. (`KUBE-FIREWALL`
     will remain, but will only be used for the "martian packet" rule,
     not for processing the drop mark.)

  2. It will warn that the `--iptables-masquerade-bit` and
     `--iptables-drop-bit` flags are deprecated and have no effect.

(Importantly, those kubelet flags will _not_ claim to be deprecated
when the feature gate is disabled, because in that case it is
important that if the user was previously overriding
`--iptables-masquerade-bit`, that they keep overriding it in both
kubelet and kube-proxy for as long as they are both redundantly
creating the chains.)

#### Beta (1.27)

The behavior is the same as alpha, except that the feature gate is
enabled by default

As long as we wait 2 releases between Alpha and Beta, then it is
guaranteed that when the user upgrades to Beta, the currently-running
kube-proxy will be one that does not require the `KUBE-MARK-DROP`
chain to exist, so the upgrade will work correctly even if nodes end
up with an old kube-proxy and a new kubelet at some point.

#### GA (1.28?)

The feature gate is now locked in the enabled state.

Most of the IPTables-handling code in kubelet can be removed (along
with the warnings in kube-proxy about keeping that code in sync with
the kubelet code).

Kubelet will now unconditionally warn that the
`--iptables-masquerade-bit` and `--iptables-drop-bit` flags are
deprecated and have no effect, and that they will be going away soon.

#### GA + 2

The deprecated kubelet flags can be removed.

#### Indeterminate Future

We may eventually remove the "martian packet" blocking rule from
Kubelet, but there is no specific plan for this at this time.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

We discovered a while back that our existing e2e tests do not properly
test the cases that are expected to result in dropped packets. (The
tests still pass even when we don't drop the packets: [kubernetes
#85572]). However, attempting to fix this resulted in the discovery
that [there is not any easy way to test these rules]. In the
`LoadBalancerSourceRanges` case, the drop rule will never get hit on
GCP (unless there is a bug in the GCP CloudProvider or the cloud load
balancers). (The drop rule _can_ get hit in a bare-metal environment
when using a third-party load balancer like MetalLB, but we have no
way of testing this in Kubernetes CI). In the traffic policy case, the
drop rule is needed during periods when kube-proxy and the cloud load
balancers are out of sync, but there is no good way to reliably
trigger this race condition for e2e testing purposes.

However, we can manually test the new rules (eg, by killing kube-proxy
before updating a service to ensure that kube-proxy and the cloud load
balancer will remain out of sync), and then once we are satisfied that
the rules do what we expect them to do, we can use the unit tests to
ensure that we continue to generate the same (or functionally
equivalent) rules in the future.

[kubernetes #85572]: https://github.com/kubernetes/kubernetes/issues/85572
[there is not any easy way to test these rules]: https://github.com/kubernetes/kubernetes/issues/85572#issuecomment-1031733890

##### Unit tests

The unit tests in `pkg/proxy/iptables/proxier_test.go` ensure that we
are generating the iptables rules that we expect to, and the [new
tests already added in 1.25] allow us to assert specifically that
particular packets would behave in particular ways.

Thus, for example, although we can't reproduce the race conditions
mentioned above in an e2e environment, we can at least confirm that if
a packet arrived on a node which it shouldn't have because of this
race condition, that the iptables rules we generate would [route it to
a `DROP` rule], rather than delivering or rejecting it.

- `pkg/proxy/iptables`: `06-21` - `65.1%`

[new tests already added in 1.25]: https://github.com/kubernetes/kubernetes/pull/107471
[route it to a `DROP` rule]: https://github.com/kubernetes/kubernetes/blob/v1.25.0-alpha.1/pkg/proxy/iptables/proxier_test.go#L5974

##### Integration tests

There are no existing integration tests of the proxy code and no plans
to add any.

##### e2e tests

As discussed above, it is not possible to test this functionality via
e2e tests in our CI environment.

### Graduation Criteria

#### Alpha

- Tests and code are implemented as described above

- Documentation of the upcoming changes in appropriate places.

#### Beta

- Two releases have passed since Alpha

- Running with the feature gate enabled causes no problems with any
  core kubernetes components.

- The SIG is not aware of any problems with third-party components
  that would justify delaying Beta. (For example, Beta might be
  delayed if a commonly-used third-party component breaks badly when
  the feature gate is enabled, but the SIG might choose not to delay
  Beta for a bug involving a component which is not widely used, or
  which can be worked around by upgrading to a newer version of that
  component, or by changing its configuration.)

#### GA

- At least one release has passed since Beta

- The SIG is not aware of any problems with third-party components
  that would justify delaying GA. (For example, if the feature breaks
  a third-party component which is no longer maintained and not likely
  to ever be fixed, then there is no point in delaying GA because of
  it.)

### Upgrade / Downgrade Strategy

Other than version skew (below), there are no upgrade / downgrade
issues; kube-proxy recreates all of the rules it uses from scratch on
startup, so for the purposes of this KEP there is no real difference
between starting a fresh kube-proxy and upgrading an existing one.

### Version Skew Strategy

As long as we wait two releases between Alpha and Beta, then all
allowed skewed versions of kubelet and kube-proxy will be compatible
with each other.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in valuesin `kep.yaml`)
  - Feature gate name: IPTablesOwnershipCleanup
  - Components depending on the feature gate:
    - kubelet

###### Does enabling the feature change any default behavior?

When the feature gate is enabled, kubelet will no longer create the
IPTables chains/rules that it used to. This may cause problems with
third-party components in Alpha but these problems are expected to be
ironed out before moving to Beta.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes

###### What happens if we reenable the feature if it was previously rolled back?

Nothing unexpected

###### Are there any tests for feature enablement/disablement?

No... there is no real difference between enabling the feature in an
existing cluster vs creating a cluster where it was always enabled.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

The most likely cause of a rollout failure would be a third-party
component that depended on one of the no-longer-existing IPTables
chains. It is impossible to predict exactly how this third-party
component would fail in this case, but it would likely impact already
running workloads.

###### What specific metrics should inform a rollback?

Any failures would be the result of third-party components being
incompatible with the change, so no core Kubernetes metrics are likely
to be relevant.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

TBD

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

This KEP will eventually remove two kubelet command-line arguments,
but not until after the feature is GA.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?

There is no simple way to do this because if the feature is working
correctly there will be no difference in externally-visible behavior.
(The generated iptables rules will be different, but the _effect_ of
the generated iptables rules will be the same.)

###### How can someone using this feature know that it is working for their instance?

- [X] Other (treat as last resort)

  - Details: As above, the feature is not supposed to have any
    externally-visible effect. If anything is not working, it is
    likely to be a third-party component, so it is impossible to say
    what a failure might look like.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A / Unchanged from previous releases. The expected end result of
this enhancement is that no externally-measurable behavior changes.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Other (treat as last resort)
  - Details: N/A / Unchanged from previous releases.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

This does not change kubelet or kube-proxy's dependencies.

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

No

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

This KEP does not change the way that either kubelet or kube-proxy
reacts to such a scenario.

###### What are other known failure modes?

As above, the only expected failure mode is that a third-party
component expects kubelet to have created the chains that it no longer
does, in which case the third-party component will react in some way
we cannot predict.

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

- Initial proposal: 2022-01-23
- Updated: 2022-03-27, 2022-04-29
- Updated: 2022-07-26 (feature gate rename)

## Drawbacks

The primary drawback is the risk of third-party breakage. But the
current state, where kubelet creates IPTables chains that it does not
use, and kube-proxy depends on someone else creating IPTables chains
that it does use, is clearly wrong.

## Alternatives

### Use a Feature Gate in Kube-Proxy As Well

The description of the Alpha stage above suggests that we will change
the code in kube-proxy _without_ feature-gating it, because the change
to kube-proxy (dropping packets directly rather than depending on
`KUBE-MARK-DROP`) is more of a refactoring/bugfix than it is a
"feature". However, it would be possible to feature-gate this instead.

This would extend the rollout by a few more releases, because we
would not be able to move the kubelet feature gate to Beta until 2
releases after the kube-proxy feature gate went to Beta.

### Move `KUBE-MARK-DROP` to Kube-Proxy Rather Than Removing It

Rather than dropping `KUBE-MARK-DROP`, we could just make kube-proxy
create it itself rather than depending on kubelet to create it. This
would potentially be slightly more third-party-component-friendly (if
there are third-party components using `KUBE-MARK-DROP`, which it's
not clear that there are).

This is a more complicated approach though, because moving
`KUBE-MARK-DROP` requires also moving the ability to override the
choice of mark bit, and if a user is overriding it in kubelet, they
_must_ override it to the same value in kube-proxy or things will
break, so we need to warn users about this in advance before we can
start changing things, and we need separate feature gates moving out
of sync for kubelet and kube-proxy. The end result is that this would
take two release cycles longer than the No-`KUBE-MARK-DROP` approach:

  - First release

      - Kubelet warns users who pass `--iptables-drop-bit` that they
        should also pass the same option to kube-proxy.

        If the `KubeletIPTablesCleanup` feature gate is enabled (which
        it is not by default), Kubelet does not create any iptables
        chains.

      - Kube-proxy is updated to accept `--iptables-drop-bit`. If the
        `KubeProxyIPTablesCleanup` feature gate is disabled (which it
        is by default), kube-proxy mostly ignores the new flag, but it
        does compare the `KUBE-MARK-DROP` rule that kubelet created
        against the rule that it would have created, and warns the
        user if they don't match. (That is, it warns the user if they
        are overriding `--iptables-drop-bit` in kubelet but not in
        kube-proxy.)

        When the feature gate is enabled, kube-proxy creates
        `KUBE-MARK-DROP`, etc, itself (using exactly the same rules
        kubelet would, such that it is possible to run with
        `KubeProxyIPTablesCleanup` enabled but
        `KubeletIPTablesCleanup` disabled).

  - Two releases later...

      - `KubeProxyIPTablesCleanup` moves to Beta.
        (`KubeletIPTablesCleanup` remains Alpha.) Since kubelet and
        kube-proxy have been warning about `--iptables-drop-bit` for 2
        releases now, everyone upgrading to this version should
        already have seen the warnings and updated their kube-proxy
        config if they need to.

        By default (with no manual feature gate overrides), kubelet
        and kube-proxy are now both creating identical
        `KUBE-MARK-DROP` rules.

  - Two releases after that...

      - `KubeProxyIPTablesCleanup` moves to GA

      - `KubeletIPTablesCleanup` moves to Beta. Since kube-proxy has
        been creating its own `KUBE-MARK-DROP` chain for 2 releases
        now, everyone upgrading to this version should already have a
        kube-proxy that creates `KUBE-MARK-DROP`, so there is no
        chance of there temporarily being no `KUBE-MARK-DROP` due to
        version skew during the upgrade.

      - When running with `KubeletIPTablesCleanup` enabled, kubelet
        warns that `--iptables-masquerade-bit` and
        `--iptables-drop-bit` are deprecated.

  - One release after that...

      - `KubeletIPTablesCleanup` moves to GA

      - Kubelet unconditionally warns about the deprecated flags

  - Two releases after that

      - We remove the deprecated flags

### Remove `KUBE-MARK-MASQ` from Kube-Proxy and Let Kubelet Own It

As discussed in [kubernetes #82125], the original plan had been to
move the maintenance of both "mark" chains into kubelet, rather than
into kube-proxy.

This would still get rid of the code duplication, and it would also
let us avoid problems with external components, by declaring that now
they _can_ always assume the existence of `KUBE-MARK-MASQ`, rather
than that they should not.

But kube-proxy already runs into problems with `KUBE-MARK-DROP`, where
if it finds that the chain has been deleted (eg, by a system firewall
restart), it is unable to recreate it properly and thus operates in a
degraded state until kubelet fixes the chain. This problem would be
much worse with `KUBE-MARK-MASQ`, which kube-proxy uses several orders
of magnitude more often than it uses `KUBE-MARK-DROP`.

(There was also other discussion in #82125 about reasons why kubelet
is not the right place to be doing low-level networking setup.)

[kubernetes #82125]: https://github.com/kubernetes/kubernetes/issues/82125
