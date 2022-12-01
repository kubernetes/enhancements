# KEP-3866: Add an nftables-based kube-proxy backend

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [The iptables kernel subsystem has unfixable performance problems](#the-iptables-kernel-subsystem-has-unfixable-performance-problems)
  - [Upstream development has moved on from iptables to nftables](#upstream-development-has-moved-on-from-iptables-to-nftables)
  - [The <code>ipvs</code> mode of kube-proxy will not save us](#the--mode-of-kube-proxy-will-not-save-us)
  - [The <code>nf_tables</code> mode of <code>/sbin/iptables</code> will not save us](#the--mode-of--will-not-save-us)
  - [The <code>iptables</code> mode of kube-proxy has grown crufty](#the--mode-of-kube-proxy-has-grown-crufty)
  - [We will hopefully be able to trade 2 supported backends for 1](#we-will-hopefully-be-able-to-trade-2-supported-backends-for-1)
  - [Writing a new kube-proxy mode will help to focus our cleanup/refactoring efforts](#writing-a-new-kube-proxy-mode-will-help-to-focus-our-cleanuprefactoring-efforts)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Functionality](#functionality)
    - [Compatibility](#compatibility)
    - [Security](#security)
- [Design Details](#design-details)
  - [High level design](#high-level-design)
  - [Low level design](#low-level-design)
    - [Tables](#tables)
    - [Communicating with the kernel nftables subsystem](#communicating-with-the-kernel-nftables-subsystem)
    - [Notes on the sample rules in this KEP](#notes-on-the-sample-rules-in-this-kep)
    - [Versioning and compatibility](#versioning-and-compatibility)
    - [NAT rules](#nat-rules)
      - [General Service dispatch](#general-service-dispatch)
      - [Masquerading](#masquerading)
      - [Session affinity](#session-affinity)
    - [Filter rules](#filter-rules)
      - [Dropping or rejecting packets for services with no endpoints](#dropping-or-rejecting-packets-for-services-with-no-endpoints)
      - [Dropping traffic rejected by <code>LoadBalancerSourceRanges</code>](#dropping-traffic-rejected-by-)
      - [Forcing traffic on <code>HealthCheckNodePort</code>s to be accepted](#forcing-traffic-on-s-to-be-accepted)
    - [Future improvements](#future-improvements)
  - [Changes from the iptables kube-proxy backend](#changes-from-the-iptables-kube-proxy-backend)
    - [Localhost NodePorts](#localhost-nodeports)
    - [NodePort Addresses](#nodeport-addresses)
    - [Behavior of service IPs](#behavior-of-service-ips)
    - [Defining an API for integration with admin/debug/third-party rules](#defining-an-api-for-integration-with-admindebugthird-party-rules)
    - [Rule monitoring](#rule-monitoring)
    - [Multiple instances of <code>kube-proxy</code>](#multiple-instances-of-)
  - [Switching between kube-proxy modes](#switching-between-kube-proxy-modes)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
    - [Scalability &amp; Performance tests](#scalability--performance-tests)
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
  - [Continue to improve the <code>iptables</code> mode](#continue-to-improve-the--mode)
  - [Fix up the <code>ipvs</code> mode](#fix-up-the--mode)
  - [Use an existing nftables-based kube-proxy implementation](#use-an-existing-nftables-based-kube-proxy-implementation)
  - [Create an eBPF-based proxy implementation](#create-an-ebpf-based-proxy-implementation)
<!-- /toc -->

## Release Signoff Checklist

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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The default kube-proxy implementation on Linux is currently based on
iptables. IPTables was the preferred packet filtering and processing
system in the Linux kernel for many years (starting with the 2.4
kernel in 2001). However, problems with iptables led to the
development of a successor, nftables, first made available in the 3.13
kernel in 2014, and growing increasingly featureful and usable as a
replacement for iptables since then. Development on iptables has
mostly stopped, with new features and performance improvements
primarily going into nftables instead.

This KEP proposes the creation of a new official/supported nftables
backend for kube-proxy. While it is hoped that this backend will
eventually replace both the `iptables` and `ipvs` backends and become
the default kube-proxy mode on Linux, that replacement/deprecation
would be handled in a separate future KEP.

## Motivation

There are currently two officially supported kube-proxy backends for
Linux: `iptables` and `ipvs`. (The original `userspace` backend was
deprecated several releases ago and removed from the tree in 1.26.)

The `iptables` mode of kube-proxy is currently the default, and it is
generally considered "good enough" for most use cases. Nonetheless,
there are good arguments for replacing it with a new `nftables` mode.

### The iptables kernel subsystem has unfixable performance problems

Although much work has been done to improve the performance of the
kube-proxy `iptables` backend, there are fundamental
performance-related problems with the implementation of iptables in
the kernel, both on the "control plane" side and on the "data plane"
side:

  - The control plane is problematic because the iptables API does not
    support making incremental changes to the ruleset. If you want to
    add a single iptables rule, the iptables binary must acquire a lock,
    download the entire ruleset from the kernel, find the appropriate
    place in the ruleset to add the new rule, add it, re-upload the
    entire ruleset to the kernel, and release the lock. This becomes
    slower and slower as the ruleset increases in size (ie, as the
    number of Kubernetes Services grows). If you want to replace a large
    number of rules (as kube-proxy does frequently), then simply the
    time that it takes `/sbin/iptables-restore` to parse all of the
    rules becomes substantial.

  - The data plane is problematic because (for the most part), the
    number of iptables rules used to implement a set of Kubernetes
    Services is directly proportional to the number of Services. And
    every packet going through the system then needs to pass through
    all of these rules, slowing down the traffic.

IPTables is the bottleneck in kube-proxy performance, and it always
will be until we stop using it.

### Upstream development has moved on from iptables to nftables

In large part due to its unfixable problems, development on iptables
in the kernel has slowed down and mostly stopped. New features are not
being added to iptables, because nftables is supposed to do everything
iptables does, but better.

Although there is no plan to remove iptables from the upstream kernel,
that does not guarantee that iptables will remain supported by
_distributions_ forever. In particular, Red Hat has declared that
[iptables is deprecated in RHEL 9] and is likely to be removed
entirely in RHEL 10, a few years from now. Other distributions have
made smaller steps in the same direction; for instance, [Debian
removed `iptables` from the set of "required" packages] in Debian 11
(Bullseye).

The RHEL deprecation in particular impacts Kubernetes in two ways:

  1. Many Kubernetes users run RHEL or one of its downstreams, so in a
     few years when RHEL 10 is released, they will be unable to use
     kube-proxy in `iptables` mode (or, for that matter, in `ipvs` or
     `userspace` mode, since those modes also make heavy use of the
     iptables API).

  2. Several upstream iptables bugs and performance problems that
     affect Kubernetes have been fixed by Red Hat developers over the
     past several years. With Red Hat no longer making any effort to
     maintain iptables, it is less likely that upstream iptables bugs
     that affect Kubernetes in the future would be fixed promptly, if
     at all.

[iptables is deprecated in RHEL 9]: https://access.redhat.com/solutions/6739041
[Debian removed `iptables` from the set of "required" packages]: https://salsa.debian.org/pkg-netfilter-team/pkg-iptables/-/commit/c59797aab9

### The `ipvs` mode of kube-proxy will not save us

Because of the problems with iptables, some developers added an `ipvs`
mode to kube-proxy in 2017. It was generally hoped that this could
eventually solve all of the problems with the `iptables` mode and
become its replacement, but this never really happened. It's not
entirely clear why... [kubeadm #817], "Track when we can enable the
ipvs mode for the kube-proxy by default" is perhaps a good snapshot of
the initial excitement followed by growing disillusionment with the
`ipvs` mode:

  - "a few issues ... re: the version of iptables/ipset shipped in the
    kube-proxy container image"
  - "clearly not ready for defaulting"
  - "complications ... with IPVS kernel modules missing or disabled on
    user nodes"
  - "we are still lacking tests"
  - "still does not completely align with what [we] support in
    iptables mode"
  - "iptables works and people are familiar with it"
  - "[not sure that it was ever intended for IPVS to be the default]"

Additionally, the kernel IPVS APIs alone do not provide enough
functionality to fully implement Kubernetes services, and so the
`ipvs` backend also makes heavy use of the iptables API. Thus, if we
are worried about iptables deprecation, then in order to switch to
using `ipvs` as the default mode, we would have to port the iptables
parts of it to use nftables anyway. But at that point, there would be
little excuse for using IPVS for the core load-balancing part,
particularly given that IPVS, like iptables, is no longer an
actively-developed technology.

[kubeadm #817]: https://github.com/kubernetes/kubeadm/issues/817
[not sure that it was ever intended for IPVS to be the default]: https://en.wikipedia.org/wiki/The_Fox_and_the_Grapes

### The `nf_tables` mode of `/sbin/iptables` will not save us

In 2018, with the 1.8.0 release of the iptables client binaries, a new
mode was added to the binaries, to allow them to use the nftables API
in the kernel rather than the legacy iptables API, while still
preserving the "API" of the original iptables binaries. As of 2022,
most Linux distributions now use this mode, so the legacy iptables
kernel API is mostly dead.

However, this new mode does not add any new _syntax_, and so it is not
possible to use any of the new nftables features (like maps) that are
not present in iptables.

Furthermore, the compatibility constraints imposed by the user-facing
API of the iptables binaries themselves prevent them from being able
to take advantage of many of the performance improvements associated
with nftables.

(Additionally, the RHEL deprecation of iptables includes
`iptables-nft` as well.)

### The `iptables` mode of kube-proxy has grown crufty

Because `iptables` is the default kube-proxy mode, it is subject to
strong backward-compatibility constraints which mean that certain
"features" that are now considered to be bad ideas cannot be removed
because they might break some existing users. A few examples:

  - It allows NodePort services to be accessed on `localhost`, which
    requires it to set a sysctl to a value that may introduce security
    holes on the system. More generally, it defaults to having
    NodePort services be accessible on _all_ node IPs, when most users
    would probably prefer them to be more restricted.

  - It implements the `LoadBalancerSourceRanges` feature for traffic
    addressed directly to LoadBalancer IPs, but not for traffic
    redirected to a NodePort by an external LoadBalancer.

  - Some new functionality only works correctly if the administrator
    passes certain command-line options to kube-proxy (eg,
    `--cluster-cidr`), but we cannot make those options be mandatory,
    since that would break old clusters that aren't passing them.

A new kube-proxy mode, which existing users would have to explicitly opt
into, could revisit these and other decisions. (Though if we expect it
to eventually become the default, then we might decide to avoid such
changes anyway.)

### We will hopefully be able to trade 2 supported backends for 1

Right now SIG Network is supporting both the `iptables` and `ipvs`
backends of kube-proxy, and does not feel like it can ditch `ipvs`
because of perceived performance issues with `iptables`. If we create a new
backend which is as functional and non-buggy as `iptables` but as
performant as `ipvs`, then we could (eventually) deprecate both of the
existing backends and only have one Linux backend to support in the future.

### Writing a new kube-proxy mode will help to focus our cleanup/refactoring efforts

There is a desire to provide a "kube-proxy library" that third parties
could use as a base for external service proxy implementations
([KEP-3786]). The existing "core kube-proxy" code, while functional,
is not very well designed and is not something we would want to
support other people using in its current form.

Writing a new proxy backend will force us to look over all of this
shared code again, and perhaps give us new ideas on how it can be
cleaned up, rationalized, and optimized.

[KEP-3786]: https://github.com/kubernetes/enhancements/issues/3786

### Goals

- Design and implement an `nftables` mode for kube-proxy.

    - Consider various fixes to legacy `iptables` mode behavior.

        - Do not enable the `route_localnet` sysctl.

        - Add a more restrictive startup mode to kube-proxy, which
          will error out if the configuration is invalid (e.g.,
          "`--detect-local-mode ClusterCIDR`" without specifying
          "`--cluster-cidr`") or incomplete (e.g.,
          partially-dual-stack but not fully-dual-stack).

        - (Possibly other changes discussed in this KEP.)

        - Ensure that any such changes are clearly documented for
          users.

        - To the extent possible, provide metrics to allow `iptables`
          users to easily determine if they are using features that
          would behave differently in `nftables` mode.

    - Document specific details of the nftables implementation that we
      want to consider as "API". In particular, document the
      high-level behavior that authors of network plugins can rely
      on. We may also document ways that third parties or
      administrators can integrate with kube-proxy's rules at a lower
      level.

- Allowing switching from the `iptables` (or `ipvs`) mode to
  `nftables`, or vice versa, without needing to manually clean up
  rules in between.

- Document the minimum kernel/distro requirements for the new backend.

- Document incompatible changes between `iptables` mode and `nftables`
  mode (e.g. localhost NodePorts, firewall handling, etc).

- Do performance testing comparing the `iptables`,
  `ipvs`, and `nftables` backends in small, medium, and large
  clusters, comparing both the "control plane" aspects (time/CPU usage
  spent reprogramming rules) and "data plane" aspects (latency and
  throughput of packets to service IPs).

- Help with the clean-up and refactoring of the kube-proxy "library"
  code.

- Although this KEP does not include anything post-GA (e.g., making
  `nftables` the default backend, or changing the status of the
  `iptables` and/or `ipvs` backends), we should have at least the
  start of a plan for the future by the time this KEP goes GA, to
  ensure that we don't just end up permanently maintaining 3 backends
  instead of 2.

### Non-Goals

- Falling into the same traps as the `ipvs` backend, to the extent
  that we can identify what those traps were.

- Removing the iptables `KUBE-IPTABLES-HINT` chain from kubelet; that
  chain exists for the benefit of any component on the node that wants
  to use iptables, and so should continue to exist even if no part of
  the kubernetes core uses iptables itself. (And there is no need to
  add anything similar for nftables, since there are no bits of host
  filesystem configuration related to nftables that containerized
  nftables users need to worry about.)

## Proposal

### Notes/Constraints/Caveats

At least three nftables-based kube-proxy implementations already
exist, but none of them seems suitable either to adopt directly or to
use as a starting point:

- [kube-nftlb]: This is built on top of a separate nftables-based load
  balancer project called [nftlb], which means that rather than
  translating Kubernetes Services directly into nftables rules, it
  translates them into nftlb load balancer objects, which then get
  translated into nftables rules. Besides making the code more
  confusing for users who aren't already familiar with nftlb, this
  also means that in many cases, new Service features would need to
  have features added to the nftlb core first before kube-nftld could
  consume them. (Also, it has not been updated since November 2020.)

- [nfproxy]: Its README notes that "nfproxy is not a 1:1 copy of
  kube-proxy (iptables) in terms of features. nfproxy is not going to
  cover all corner cases and special features addressed by
  kube-proxy". (Also, it has not been updated since January 2021.)

- [kpng's nft backend]: This was written as a proof of concept and is
  mostly a straightforward translation of the iptables rules to
  nftables, and doesn't make good use of nftables features that would
  let it reduce the total number of rules. It also makes heavy use of
  kpng's APIs, like "DiffStore", which there is not consensus about
  adopting upstream.

[kube-nftlb]: https://github.com/zevenet/kube-nftlb
[nftlb]: https://github.com/zevenet/nftlb
[nfproxy]: https://github.com/sbezverk/nfproxy
[kpng's nft backend]: https://github.com/kubernetes-sigs/kpng/tree/master/backends/nft

### Risks and Mitigations

#### Functionality

The primary risk of the proposal is feature or stability regressions,
which will be addressed by testing, and by a slow, optional, rollout
of the new proxy mode.

The most important mitigation for this risk is ensuring that rollback
from `nftables` mode back to `iptables`/`ipvs` mode works reliably.

#### Compatibility

Many Kubernetes networking implementations use kube-proxy as their
service proxy implementation. Given that few low-level details of
kube-proxy's behavior are explicitly specified, using it as part of a
larger networking implementation (and in particular, writing a
NetworkPolicy implementation that interoperates with it correctly)
necessarily requires making assumptions about (currently-)undocumented
aspects of its behavior (such as exactly when and how packets get
rewritten).

While the `nftables` mode is likely to look very similar to the
`iptables` mode from the outside, some CNI plugins, NetworkPolicy
implementations, etc, may need updates in order to work with it. (This
may further limit the amount of testing the new mode can get during
the Alpha phase, if it is not yet compatible with popular network
plugins at that point.) There is not much we can do here, other than
avoiding *gratuitous* behavioral differences.

#### Security

The `nftables` mode should not pose any new security issues relative
to the `iptables` mode.

## Design Details

### High level design

At a high level, the new mode should have the same architecture as the
existing modes; it will use the service/endpoint-tracking code in
`k8s.io/kubernetes/pkg/proxy` to watch for changes, and update rules
in the kernel accordingly.

### Low level design

Some details will be figured out as we implement it. We may start with
an implementation that is architecturally closer to the `iptables`
mode, and then rewrite it to take advantage of additional nftables
features over time.

#### Tables

Unlike iptables, nftables does not have any reserved/default tables or
chains (eg, `nat`, `PREROUTING`). Instead, each nftables user is
expected to create and work with its own table(s), and to ignore the
tables created by other components (for example, when firewalld is
running in nftables mode, restarting it only flushes the rules in the
`firewalld` table, unlike when it is running in iptables mode, where
restarting it causes it to flush _all_ rules).

Within each table, "base chains" can be connected to "hooks" that give
them behavior similar to the built-in iptables chains. (For example, a
chain with the properties `type nat` and `hook prerouting` would work
like the `PREROUTING` chain in the iptables `nat` table.) The
"priority" of a base chain controls when it runs relative to other
chains connected to the same hook in the same or other tables.

An nftables table can only contain rules for a single "family" (`ip`
(v4), `ip6`, `inet` (both IPv4 and IPv6), `arp`, `bridge`, or
`netdev`). We will create a single `kube-proxy` table in the `ip`
family, and another in the `ip6` family. All of our chains, sets,
maps, etc, will go into those tables.

(In theory, instead of creating one table each in the `ip` and `ip6`
families, we could create a single table in the `inet` family and put
both IPv4 and IPv6 chains/rules there. However, this wouldn't really
result in much simplification, because we would still need separate
sets/maps to match IPv4 addresses and IPv6 addresses. (There is no
data type that can store/match either an IPv4 address or an IPv6
address.) Furthermore, because of how Kubernetes Services evolved in
parallel with the existing kube-proxy implementation, we have ended up
with a dual-stack Service semantics that is most easily implemented by
handling IPv4 and IPv6 completely separately anyway.)

#### Communicating with the kernel nftables subsystem

We will use the `nft` command-line tool to read and write rules, much
like how we use command-line tools in the `iptables` and `ipvs`
backends.

However, the `nft` tool is mostly just a thin wrapper around
`libnftables`, so any golang API that wraps the `nft` command-line
could easily be rewritten to use `libnftables` directly (via a cgo
wrapper) in the future if that seemed like a better idea. (In theory
we could also use netlink directly, without needing cgo or external
libraries, but this would probably be a bad idea; `libnftables`
implements quite a bit of functionality on top of the raw netlink
API.)

The nftables command-line tool allows either a single command per
invocation (as with `/sbin/iptables`):

```
$ nft add table ip kube-proxy '{ comment "Kubernetes service proxying rules"; }'
$ nft add chain ip kube-proxy services
$ nft add rule ip kube-proxy services ip daddr . ip protocol . th dport vmap @service_ips
```

or multiple commands to be executed in a single atomic transaction (as
with `/sbin/iptables-restore`, but more flexible):

```
$ nft -f - <<EOF
add table ip kube-proxy { comment "Kubernetes service proxying rules"; }
add chain ip kube-proxy services
add rule ip kube-proxy services ip daddr . ip protocol . th dport vmap @service_ips
EOF
```

The syntax for the two modes is the same, other than the need to
escape shell meta characters in the former case.

When reading data from the kernel (`nft list ...`), `nft` outputs the
data in a nested "object" form:

```
$ nft list table ip kube-proxy
table ip kube-proxy {
  comment "Kubernetes service proxying rules";

  chain services {
    ip daddr . ip protocol . th dport vmap @service_ips
  }
}
```

(It is possible to pass data to `nft -f` in this form as well, but
this wouldn't be useful for us, since we would have to pass the entire
contents of `table ip kube-proxy` rather than just adding, removing,
and updating the particular rules, sets, etc, that we wanted to
change.)

`nft` also has a JSON API, which would theoretically be a better
option for programmatic use than the "plain text" API. Unfortunately,
the representation of rules in this mode is vastly different from the
representation of rules in "plain text" mode:

```
$ nft --json list table ip kube-proxy | jq .
...
    {
      "rule": {
        "family": "ip",
        "table": "kube-proxy",
        "chain": "services",
        "handle": 19,
        "expr": [
          {
            "vmap": {
              "key": {
                "concat": [
                  {
                    "payload": {
                      "protocol": "ip",
                      "field": "daddr"
                    }
                  },
                  {
                    "payload": {
                      "protocol": "ip",
                      "field": "protocol"
                    }
                  },
                  {
                    "payload": {
                      "protocol": "th",
                      "field": "dport"
                    }
                  }
                ]
              },
              "data": "@service_ips"
            }
          }
        ]
      }
...
```

While it's clear how this _particular_ rule would be converted back
and forth between the two forms, there is no way to be able to map
_all_ rules back and forth without having separate code for every rule
type. Furthermore, the JSON syntax of individual rules is poorly
documented, and essentially all examples on the web (including the
nftables wiki, random blog posts, etc) use the non-JSON syntax. So if
we used the JSON syntax in kube-proxy, it would make the code harder
to understand and to maintain.

As a result, the plan is that for our internal nftables API:

- When passing data _to_ `nft`, we will use the "plain text" API. In
  particular, this means that all `add rule ...` commands will use the
  well-documented plain text rule form.

- When reading data back _from_ `nft`, we will use the JSON API, to
  ensure that the results are unambiguously parseable (rather than
  having to make assumptions about the exact whitespace, punctuation,
  etc, that `nft` will output in particular cases in the plain text
  mode).

This means that our internal nftables API would not be able to support
reading back rules in a "legible" form. However, this is not expected
to be a problem, given that our internal iptables API
(`pkg/util/iptables`) also does not explicitly support this, and it's
not a problem for the iptables backend.

#### Notes on the sample rules in this KEP

The examples below all show data in the plain text "object" form, but
this is just for reader convenience, and does not correspond to either
the form we would be writing the data in (the multi-command
transaction form) or the form we would be reading it back in (JSON).
(Likewise, note that the `#`-prefixed comments would be ignored by
`nft` and are only there for the benefit of the KEP reader, whereas
the `comment "..."` comments are actual object metadata that would be
stored in nftables, as with iptables `--comment "..."`. Every table,
chain, set, map, rule, and set/map element can have its own comment,
so there is a lot of opportunity for us to make the ruleset
self-documenting, if we want to.)

The examples below are also all IPv4-specific, for simplicity. When
actually writing out rules for nft, we will need to switch between,
e.g., "`ip daddr`" and "`ip6 daddr`" appropriately, to match an IPv4
or IPv6 destination address. This will actually be fairly simple
because the `nft` command lets you create "variables" (really
constants) and substitute their values into the rules. Thus, we can
just always have the rule-generating code write "`$IP daddr`", and
then pass either "`-D IP=ip`" or "`-D IP=ip6`" to `nft` to fix it up.)

The per-service/per-endpoint chain names below use hashed strings to
shorten the names, as in the `iptables` backend (e.g.,
"`svc_4SW47YFZTEDKD3PK`", where that hash was copied out of the
existing `iptables` unit tests and happens to represent
"`ns4/svc4:p80tcp`"). However, it turns out that nftables chain names
can be much longer than iptables chain names (256 characters rather
than 30), so we ought to be able to create more recognizable chain
names in the `nftables` backend.

The multi-word names in the examples are also inconsistent about the
use of underscores vs hyphens; underscores are standard in most
nftables documentation, but hyphens are more
`iptables`-kube-proxy-like. We should eventually settle on one or the
other.

(Also, most of the examples below have not actually been tested and
may have syntax errors. Caveat lector.)

#### Versioning and compatibility

Since nftables is subject to much more development than iptables has
been recently, we will need to pay more attention to kernel and tool
versions.

The `nft` command has a `--check` option which can be used to check if
a command could be run successfully; it parses the input, and then
(assuming success), uploads the data to the kernel and asks the kernel
to check it (but not actually act on it) as well. Thus, with a few
`nft --check` runs at startup we should be able to confirm what
features are known to both the tooling and the kernel.

It is not yet clear what the minimum kernel or `nft` command-line
versions needed by the `nftables` backend will be. The newest feature
used in the examples below was added in Linux 5.6, released in March
2020 (though they could be rewritten to not need that feature).

It is possible some users will not be able to upgrade from the
`iptables` and `ipvs` backends to `nftables`. (Certainly the
`nftables` backend will not support RHEL 7, which some people are
still using Kubernetes with.)

#### NAT rules

##### General Service dispatch

For ClusterIP and external IP services, we will use an nftables
"verdict map" to store the logic about where to dispatch traffic,
based on destination IP, protocol, and port. We will then need only a
single actual rule to apply the verdict map to all inbound traffic.
(Or it may end up making more sense to have separate verdict maps for
ClusterIP, ExternalIP, and LoadBalancer IP?) Either way, service
dispatch will be roughly **O(1)** rather than **O(n)** as in the
`iptables` backend.

Likewise, for NodePort traffic, we will use a verdict map matching
only on destination protocol / port, with the rules set up to only
check the `nodeports` map for packets addressed to a local IP.

```
map service_ips {
  comment "ClusterIP, ExternalIP and LoadBalancer IP traffic";

  # The "type" clause defines the map's datatype; the key type is to
  # the left of the ":" and the value type to the right. The map key
  # in this case is a concatenation (".") of three values; an IPv4
  # address, a protocol (tcp/udp/sctp), and a port (aka
  # "inet_service"). The map value is a "verdict", which is one of a
  # limited set of nftables actions. In this case, the verdicts are
  # all "goto" statements.

  type ipv4_addr . inet_proto . inet_service : verdict;

  elements {
    172.30.0.44 . tcp . 80 : goto svc_4SW47YFZTEDKD3PK,
    192.168.99.33 . tcp . 80 : goto svc_4SW47YFZTEDKD3PK,
    ...
  }
}

map service_nodeports {
  comment "NodePort traffic";
  type inet_proto . inet_service : verdict;

  elements {
    tcp . 3001 : goto svc_4SW47YFZTEDKD3PK,
    ...
  }
}

chain prerouting {
  jump services
  jump nodeports
}

chain services {
  # Construct a key from the destination address, protocol, and port,
  # then look that key up in the `service_ips` vmap and take the
  # associated action if it is found.

  ip daddr . ip protocol . th dport vmap @service_ips
}

chain nodeports
  # Return if the destination IP is non-local, or if it's localhost.
  fib daddr type != local return
  ip daddr == 127.0.0.1 return

  # If --nodeport-addresses was in use then the above would instead be
  # something like:
  #   ip daddr != { 192.168.1.5, 192.168.3.10 } return

  # dispatch on the service_nodeports vmap
  ip protocol . th dport vmap @service_nodeports
}

# Example per-service chain
chain svc_4SW47YFZTEDKD3PK {
  # Send to random endpoint chain using an inline vmap
  numgen random mod 2 vmap {
    0 : goto sep_UKSFD7AGPMPPLUHC,
    1 : goto sep_C6EBXVWJJZMIWKLZ
  }
}

# Example per-endpoint chain
chain sep_UKSFD7AGPMPPLUHC {
  # masquerade hairpin traffic
  ip saddr 10.180.0.4 jump mark_for_masquerade

  # send to selected endpoint
  dnat to 10.180.0.4:8000
}
```

##### Masquerading

The example rules above include

```
  ip saddr 10.180.0.4 jump mark_for_masquerade
```

to masquerade hairpin traffic, as in the `iptables` proxier. This
assumes the existence of a `mark_for_masquerade` chain, not shown.

nftables has the same constraints on DNAT and masquerading as iptables
does; you can only DNAT from the "prerouting" stage and you can only
masquerade from the "postrouting" stage. Thus, as with `iptables`, the
`nftables` proxy will have to handle DNAT and masquerading at separate
times. One possibility would be to simply copy the existing logic from
the `iptables` proxy, using the packet mark to communicate from the
prerouting chains to the postrouting ones.

However, it should be possible to do this in nftables without using
the mark or any other externally-visible state; we can just create an
nftables `set`, and use that to communicate information between the
chains. Something like:

```
# Set of 5-tuples of connections that need masquerading
set need_masquerade {
  type ipv4_addr . inet_service . ipv4_addr . inet_service . inet_proto;
  flags timeout ; timeout 5s ;
}

chain mark_for_masquerade {
  update @need_masquerade { ip saddr . th sport . ip daddr . th dport . ip protocol }
}

chain postrouting_do_masquerade {
  # We use "ct original ip daddr" and "ct original proto-dst" here
  # since the packet may have been DNATted by this point.

  ip saddr . th sport . ct original ip daddr . ct original proto-dst . ip protocol @need_masquerade masquerade
}
```

This is not yet tested, but some kernel nftables developers have
confirmed that it ought to work. We should test to make sure that
having a potentially-high-churn `need_masquerade` set will not be a
performance problem.

##### Session affinity

Session affinity can be done in roughly the same way as in the
`iptables` proxy, just using the more general nftables "set" framework
rather than the affinity-specific version of sets provided by the
iptables `recent` module. In fact, since nftables allows arbitrary set
keys, we can optimize relative to `iptables`, and only have a single
affinity set per service, rather than one per endpoint. (And we also
have the flexibility to change the affinity key in the future if we
want to, eg to key on source IP+port rather than just source IP.)

```
set affinity_4SW47YFZTEDKD3PK {
  # Source IP . Destination IP . Destination Port
  type ipv4_addr . ipv4_addr . inet_service;
  flags timeout; timeout 3h;
}

chain svc_4SW47YFZTEDKD3PK {
  # Check for existing session affinity against each endpoint
  ip saddr . 10.180.0.4 . 80 @affinity_4SW47YFZTEDKD3PK goto sep_UKSFD7AGPMPPLUHC
  ip saddr . 10.180.0.5 . 80 @affinity_4SW47YFZTEDKD3PK goto sep_C6EBXVWJJZMIWKLZ

  # Send to random endpoint chain
  numgen random mod 2 vmap {
    0 : goto sep_UKSFD7AGPMPPLUHC,
    1 : goto sep_C6EBXVWJJZMIWKLZ
  }
}

chain sep_UKSFD7AGPMPPLUHC {
  # Mark the source as having affinity for this endpoint
  update @affinity_4SW47YFZTEDKD3PK { ip saddr . 10.180.0.4 . 80 }

  ip saddr 10.180.0.4 jump mark_for_masquerade
  dnat to 10.180.0.4:8000
}

# likewise for other endpoint(s)...
```

```
<<[UNRESOLVED session affinity ]>>

Decide if we want to stick with iptables-like affinity on sourceIP
only, switch to ipvs-like sourceIP+sourcePort affinity, add a new
`v1.ServiceAffinity` value to disambiguate, or something else.

(See also https://github.com/kubernetes/kubernetes/pull/112806, which
removed session affinity timeouts from conformance, and claimed that
"Our plan is to deprecate the current affinity options and re-add
specific options for various behaviors so it's clear exactly what
plugins support and which behavior (if any) we want to require for
conformance in the future.")

(FTR, the nftables backend would have no difficulty implementing the
existing timeout behavior.)

<<[/UNRESOLVED]>>
```

#### Filter rules

The `iptables` mode uses the `filter` table for three kinds of rules:

##### Dropping or rejecting packets for services with no endpoints

As with service dispatch, this is easily handled with a verdict map:

```
map no_endpoint_services {
  type ipv4_addr . inet_proto . inet_service : verdict
  elements = {
    192.168.99.22 . tcp . 80 : drop,
    172.30.0.46 . tcp . 80 : goto reject_chain,
    1.2.3.4 . tcp . 80 : drop
  }
}

chain filter {
  ...
  ip daddr . ip protocol . th dport vmap @no_endpoint_services
  ...
}

# helper chain needed because "reject" is not a "verdict" and so can't
# be used directly in a verdict map
chain reject_chain {
  reject
}
```

##### Dropping traffic rejected by `LoadBalancerSourceRanges`

The implementation of LoadBalancer source ranges will be similar to
the ipset-based implementation in the `ipvs` kube proxy: we use one
set to recognize "traffic that is subject to source ranges", and then
another to recognize "traffic that is _accepted_ by its service's
source ranges". Traffic which matches the first set but not the second
gets dropped:

```
set firewall {
  comment "destinations that are subject to LoadBalancerSourceRanges";
  type ipv4_addr . inet_proto . inet_service
}
set firewall_allow {
  comment "destination+sources that are allowed by LoadBalancerSourceRanges";
  type ipv4_addr . inet_proto . inet_service . ipv4_addr
}

chain filter {
  ...
  ip daddr . ip protocol . th dport @firewall jump firewall_check
  ...
}

chain firewall_check {
  ip daddr . ip protocol . th dport . ip saddr @firewall_allow return
  drop
}
```

Where, eg, adding a Service with LoadBalancer IP `10.1.2.3`, port
`80`, and source ranges `["192.168.0.3/32", "192.168.1.0/24"]` would
result in:

```
add element ip kube-proxy firewall { 10.1.2.3 . tcp . 80 }
add element ip kube-proxy firewall_allow { 10.1.2.3 . tcp . 80 . 192.168.0.3/32 }
add element ip kube-proxy firewall_allow { 10.1.2.3 . tcp . 80 . 192.168.1.0/24 }
```

##### Forcing traffic on `HealthCheckNodePort`s to be accepted

The `iptables` mode adds rules to ensure that traffic to NodePort
services' health check ports is allowed through the firewall. eg:

```
-A KUBE-NODEPORTS -m comment --comment "ns2/svc2:p80 health check node port" -m tcp -p tcp --dport 30000 -j ACCEPT
```

(There are also rules to accept any traffic that has already been
tagged by conntrack.)

This cannot be done reliably in nftables; the semantics of `accept`
(or `-j ACCEPT` in iptables) is to end processing _of the current
table_. In iptables, this effectively guarantees that the packet is
accepted (since `-j ACCEPT` is mostly only used in the `filter`
table), but in nftables, it is still possible that someone would later
call `drop` on the packet from another table, causing it to be
dropped. There is no way to reliably "sneak behind the firewall's
back" like you can in iptables; if an nftables-based firewall is
dropping kube-proxy's packets, then you need to actually configure
_that firewall_ to accept them instead.

However, this firewall-bypassing behavior is somewhat legacy anyway;
the `iptables` proxy is able to bypass a _local_ firewall, but has no
ability to bypass a firewall implemented at the cloud network layer,
which is perhaps a more common configuration these days anyway.
Administrators using non-local firewalls are already required to
configure those firewalls correctly to allow Kubernetes traffic
through, and it is reasonable for us to just extend that requirement
to administrators using local firewalls as well.

Thus, the `nftables` backend will not attempt to replicate these
`iptables`-backend rules.

#### Future improvements

Further improvements are likely possible.

For example, it would be nice to not need a separate "hairpin" check for
every endpoint. There is no way to ask directly "does this packet have
the same source and destination IP?", but the proof-of-concept [kpng
nftables backend] does this instead:

```
set hairpin {
  type ipv4_addr . ipv4_addr;
  elements {
    10.180.0.4 . 10.180.0.4,
    10.180.0.5 . 10.180.0.5,
    ...
  }
}

chain ... {
  ...
  ip saddr . ip daddr @hairpin jump mark_for_masquerade
}
```

More efficiently, if nftables eventually got the ability to call eBPF
programs as part of rule processing (like iptables's `-m ebpf`) then
we could write a trivial eBPF program to check "source IP equals
destination IP" and then call that rather than needing the giant set
of redundant IPs.

If we do this, then we don't need the per-endpoint hairpin check
rules. If we could also get rid of the per-endpoint affinity-updating
rules, then we could get rid of the per-endpoint chains entirely,
since `dnat to ...` is an allowed vmap verdict:

```
chain svc_4SW47YFZTEDKD3PK {
  # FIXME handle affinity somehow

  # Send to random endpoint
  random mod 2 vmap {
    0 : dnat to 10.180.0.4:8000
    1 : dnat to 10.180.0.5:8000
  }
}
```

With the current set of nftables functionality, it does not seem
possible to do this (in the case where affinity is in use), but future
features may make it possible.

It is not yet clear what the tradeoffs of such rewrites are, either in
terms of runtime performance, or of admin/developer-comprehensibility
of the ruleset.

[kpng nftables backend]: https://github.com/kubernetes-sigs/kpng/tree/master/backends/nft

### Changes from the iptables kube-proxy backend

Switching to a new backend which people will have to opt into gives us
the chance to break backward-compatibility in various places where we
don't like the current iptables kube-proxy behavior.

However, if we intend to eventually make the `nftables` mode the
default, then differences from `iptables` mode will be more of a
problem, so we should limit these changes to cases where the benefit
outweighs the cost.

#### Localhost NodePorts

Kube-proxy in `iptables` mode supports NodePorts on `127.0.0.1` (for
IPv4 services) by default. (Kube-proxy in `ipvs` mode does not support
this, and neither mode supports localhost NodePorts for IPv6 services,
although `userspace` mode did, in single-stack IPv6 clusters.)

Localhost NodePort traffic does not work cleanly with a DNAT-based
approach to NodePorts, because moving a localhost packet to network
interface other than `lo` causes the kernel to consider it "martian"
and refuse to route it. There are various ways around this problem:

  1. The `userspace` approach: Proxy packets in userspace rather than
     redirecting them with DNAT. (The `userspace` proxy did this for
     all IPs; the fact that localhost NodePorts worked with the
     `userspace` proxy was a coincidence, not an explicitly-intended
     feature).

  2. The `iptables` approach: Enable the `route_localnet` sysctl,
     which tells the kernel to never consider IPv4 loopback addresses
     to be "martian", so that DNAT works. This only works for IPv4;
     there is no corresponding sysctl for IPv6. Unfortunately, enabling
     this sysctl opens security holes ([CVE-2020-8558]), which
     kube-proxy then needs to try to close, which it does by creating
     iptables rules to block all the packets that `route_localnet`
     would have blocked _except_ for the ones we want (which assumes
     that the administrator [didn't also change certain other sysctls]
     that might have been safe to change had we not set
     `route_localnet`, and which according to some reports [may block
     legitimate traffic] in some configurations).

  3. The Cilium approach: Intercept the connect(2) call with eBPF and
     rewrite the destination IP there, so that the network stack never
     actually sees a packet with destination `127.0.0.1` / `::1`. (As
     in the `userspace` kube-proxy case, this is not a special-case
     for localhost, it's just how Cilium does service proxying.)

  4. If you control the client, you can explicitly bind the socket to
     `127.0.0.1` / `::1` before connecting. (I'm not sure why this
     works since the packet still eventually gets routed off `lo`.) It
     doesn't seem to be possible to "spoof" this after the socket is
     created, though as with the previous case, you could do this by
     intercepting syscalls with eBPF.

In discussions about this feature, only one real use case has been
presented: it allows you to run a docker registry in a pod and then
have nodes use a NodePort service via `127.0.0.1` to access that
registry. Docker treats `127.0.0.1` as an "insecure registry" by
default (though containerd and cri-o do not) and so does not require
TLS authentication in this case; using any other IP would require
setting up TLS certificates, making the deployment more complicated.
(In other words, this is basically an intentional exploitation of the
security hole that CVE-2020-8558 warns about: enabling
`route_localnet` may allow someone to access a service that doesn't
require authentication because it assumed it was only accessible to
localhost.)

In all other cases, it is generally possible (though not always
convenient) to just rewrite things to use the node IP rather than
localhost (or to use a ClusterIP rather than a NodePort). Indeed,
since localhost NodePorts do not work with `ipvs` mode or with IPv6,
many places that used to use NodePorts on `127.0.0.1` have already
been rewritten to not do so (eg [contiv/vpp#1434]).

So:

  - There is no way to make IPv6 localhost NodePorts work with a
    NAT-based solution.

  - The way to make IPv4 localhost NodePorts work with NAT introduces
    a security hole, and we don't necessarily have a fully-generic way
    to mitigate it.

  - The only commonly-argued-for use case for the feature involves
    deploying a service in a configuration which its own documentation
    describes as insecure and "only appropriate for testing".

      - The use case in question works by default against cri-dockerd
        but not against containerd or cri-o with their default
        configurations.

      - cri-dockerd, containerd, and cri-o all allow additional
        "insecure registry" IPs/CIDRs to be configured, so an
        administrator could configure them to allow non-TLS image
        pulling against a ClusterIP.

Given this, I think we should not try to support localhost NodePorts
in the `nftables` backend.

```
<<[UNRESOLVED dnat-but-no-route_localnet ]>>

As a possible compromise, we could make the `nftables` backend create
appropriate DNAT and SNAT rules for localhost NodePorts (when
`--nodeport-addresses` includes `127.0.0.1`), but _not_ change
`route_localnet`. In that case, we could document that administrators
could enable `route_localnet` themselves if they wanted to support
NodePorts on `127.0.0.1`, but then they would also be responsible for
mitigating any security holes they had introduced.

<<[/UNRESOLVED]>>
```

[CVE-2020-8558]: https://nvd.nist.gov/vuln/detail/CVE-2020-8558
[didn't also change certain other sysctls]: https://github.com/kubernetes/kubernetes/pull/91666#issuecomment-640733664
[may block legitimate traffic]: https://github.com/kubernetes/kubernetes/pull/91666#issuecomment-763549921
[contiv/vpp#1434]: https://github.com/contiv/vpp/pull/1434

#### NodePort Addresses

In addition to the localhost issue, iptables kube-proxy defaults to
accepting NodePort connections on all local IPs, which has effects
varying from intended-but-unexpected ("why can people connect to
NodePort services from the management network?") to clearly-just-wrong
("why can people connect to NodePort services on LoadBalancer IPs?")

The nftables proxy should default to only opening NodePorts on a
single interface, probably the interface with the default route by
default. (Ideally, you really want it to accept NodePorts on the
interface that holds the route to the cloud load balancers, but we
don't necessarily know what that is ahead of time.) Admins can use
`--nodeport-addresses` to override this.

#### Behavior of service IPs

```
<<[UNRESOLVED unused service IP ports ]>>

@thockin has suggested that service IPs should reject connections on
ports they aren't using. (This would most easily be implemented by
adding a `--service-cidr` flag to kube-proxy so we could just "reject
everything else", but even without that we could at least reject
connections on inactive ports of active service IPs.)

<<[/UNRESOLVED]>>
```

```
<<[UNRESOLVED service IP pings ]>>

Users sometimes get confused by the fact that service IPs do not
respond to ICMP pings, and perhaps this is something we could change.

<<[/UNRESOLVED]>>
```

#### Defining an API for integration with admin/debug/third-party rules

Administrators sometimes want to add rules to log or drop certain
packets. Kube-proxy makes this difficult because it is constantly
rewriting its rules, making it likely that admin-added rules will be
deleted shortly after being added.

Likewise, external components (eg, NetworkPolicy implementations) may
want to write rules that integrate with kube-proxy's rules in
well-defined ways.

The existing kube-proxy modes do not provide any explicit "API" for
integrating with them, although certain implementation details of the
`iptables` backend in particular (e.g. the fact that service IPs in
packets are rewritten to endpoint IPs during iptables's `PREROUTING`
phase, and that masquerading will not happen before `POSTROUTING`) are
effectively API, in that we know that changing them would result in
significant ecosystem breakage.

We should provide a stronger definition of these larger-scale "black
box" guarantees in the `nftables` backend. NFTables makes this easier
than iptables in some ways, because each application is expected to
create their own table, and not interfere with anyone else's tables.
If we document the `priority` values we use to connect to each
nftables hook, then admins and third party developers should be able
to reliably process packets before or after kube-proxy, without
needing to modify kube-proxy's chains/rules.

In cases where administrators want to insert rules into the middle of
particular service or endpoint chains, we would have the same problem
that the `iptables` backend has, which is that it would be difficult
for us to avoid accidentally overwriting them when we update rules.
Additionally, we want to preserve our ability to redesign the rules
later to take better advantage of nftables features, which would be
impossible to do if we were officially allowing users to modify the
existing rules.

One possibility would be to add "admin override" vmaps that are
normally empty but which admins could add `jump`/`goto` rules to for
specific services to augment/bypass the normal service processing. It
probably makes sense to leave these out initially and see if people
actually do need them, or if creating rules in another table is
sufficient.

```
<<[UNRESOLVED external rule integration API ]>>

It will be easier to figure out what the right thing to do here is
once we actually have a working implementation.

<<[/UNRESOLVED]>>
```

#### Rule monitoring

Given the constraints of the iptables API, it would be extremely
inefficient to do [a controller loop in the "standard" style]:

```
for {
    desired := getDesiredState()
    current := getCurrentState()
    makeChanges(desired, current)
}
```

(In particular, the combination of "`getCurrentState`" and
"`makeChanges`" is slower than just skipping the "`getCurrentState`"
and rewriting everything from scratch every time.)

In the past, the `iptables` backend *did* rewrite everything from
scratch every time:

```
for {
    desired := getDesiredState()
    makeChanges(desired, nil)
}
```

but [KEP-3453] "Minimizing iptables-restore input size" changed this,
to improve performance:

```
for {
    desired := getDesiredState()
    predicted := getPredictedState()
    if err := makeChanges(desired, predicted); err != nil {
        makeChanges(desired, nil)
    }
}
```

That is, it makes incremental updates under the assumption that the
current state is correct, but if an update fails (e.g. because it
assumes the existence of a chain that didn't exist), kube-proxy falls
back to doing a full rewrite. (It also eventually falls back to a full
update after enough time passes.)

Proxies based on iptables have also historically had the problem that
system processes (particularly firewall implementations) would
sometimes flush all iptables rules and restart with a clean state,
thus completely breaking kube-proxy. The initial solution for this
problem was to just recreate all iptables rules every 30 seconds even
if no services/endpoints had changed. Later this was changed to create
a single "canary" chain, and check every 30 seconds that the canary
had not been deleted, and only recreate everything from scratch if the
canary disappears.

NFTables provides a way to monitor for changes without doing polling;
you can keep a netlink socket open to the kernel (or a pipe open to an
`nft monitor` process) and receive notifications when particular kinds
of nftables objects are created or destroyed.

However, the "everyone uses their own table" design of nftables means
that this should not be necessary. IPTables-based firewall
implementations flush all iptables rules because everyone's iptables
rules are all mixed together and it's hard to do otherwise. But in
nftables, a firewall ought to only flush _its own_ table when
restarting, and leave everyone else's tables untouched. In particular,
firewalld works this way when using nftables. We will need to see what
other firewall implementations do.

[a controller loop in the "standard" style]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/controllers.md
[KEP-3453]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/3453-minimize-iptables-restore/README.md

#### Multiple instances of `kube-proxy`

```
<<[UNRESOLVED multiple instances ]>>

@uablrek has suggested various changes aimed at allowing multiple
kube-proxy instances on a single node:

  - Have the top-level table name by overridable.
  - Allow configuring the chain priorities.
  - Allow configuring which interfaces to process traffic on.

This can be revisited once we have a basic implementation.

<<[/UNRESOLVED]>>
```

### Switching between kube-proxy modes

In the past, kube-proxy attempted to allow users to switch between the
`userspace` and `iptables` modes (and later the `ipvs` mode) by just
restarting kube-proxy with the new arguments. Each mode would attempt
to clean up the iptables rules used by the other modes on startup.

Unfortunately, this didn't work well because the three modes all used
some of the same iptables chains, so, e.g., when kube-proxy started up
in `iptables` mode, it would try to delete the `userspace` rules, but
this would end up deleting rules that had been created by `iptables`
mode too, which mean that any time you restarted kube-proxy, it would
immediately delete some of its rules and be in a broken state until it
managed to re-sync from the apiserver. So this code was removed with
[KEP-2448].

However, the same problem would not apply when switching between an
iptables-based mode and an nftables-based mode; it should be safe to
delete all `iptables` and `ipvs` rules when starting kube-proxy in
`nftables` mode, and to delete all `nftables` rules when starting
kube-proxy in `iptables` or `ipvs` mode. This will make it easier for
users to switch between modes.

Since rollback from `nftables` mode is most important when the
`nftables` mode is not actually working correctly, we should do our
best to make sure that the cleanup code that runs when rolling back to
`iptables`/`ipvs` mode is likely to work correctly even if the rest of
the `nftables` code is broken. To that end, we can have it simply run
`nft` directly, bypassing the abstractions used by the rest of the
code. Since our rules will be isolated to our own tables, all we need
to do to clean up all of our rules is:

```
nft delete table ip kube-proxy
nft delete table ip6 kube-proxy
```

In fact, this is simple enough that we could document it explicitly as
something administrators could do if they run into problems while
rolling back.

[KEP-2448]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/2448-Remove-kube-proxy-automatic-clean-up-logic

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

We will add unit tests for the `nftables` mode that are equivalent to
the ones for the `iptables` mode. In particular, we will port over the
tests that feed Services and EndpointSlices into the proxy engine,
dump the generated ruleset, and then mock running packets through the
ruleset to determine how they would behave.

Since virtually all of the new code will be in a new directory, there
should not be any large changes either way to the test coverage
percentages in any existing directories.

As of 2023-09-22, `pkg/proxy/iptables` has 70.6% code coverage in its
unit tests. For Alpha, we will have comparable coverage for
`nftables`. However, since the `nftables` implementation is new, and
more likely to have bugs than the older, widely-used `iptables`
implementation, we will also add additional unit tests before Beta.

##### Integration tests

Kube-proxy does not have integration tests.

##### e2e tests

Most of the e2e testing of kube-proxy is backend-agnostic. Initially,
we will need a separate e2e job to test the nftables mode (like we do
with ipvs). Eventually, if nftables becomes the default, then this
would be flipped around to having a legacy "iptables" job.

The test "`[It should recreate its iptables rules if they are
deleted]`" tests (a) that kubelet recreates `KUBE-IPTABLES-HINT` if it
is deleted, and (b) that deleting all `KUBE-*` iptables rules does not
cause services to be broken forever. The latter part is obviously a
no-op under `nftables` kube-proxy, but we can run it anyway. (We are
currently assuming that we will not need an nftables version of this
test, since the problem of one component deleting another component's
rules should not exist with nftables.)

(Though not directly related to kube-proxy, there are also other e2e
tests that use iptables which should eventually be ported to nftables;
notably, the ones using [`TestUnderTemporaryNetworkFailure`].)

For the most part, we should not need to add any nftables-specific e2e
tests; the `nftables` backend's job is just to implement the Service
proxy API to the same specifications as the other backends do, so the
existing e2e tests already cover everything relevant. The only
exception to this is in cases where we change default behavior from
the `iptables` backend, in which case we may need new tests for the
different behavior.

We will eventually need e2e tests for switching between `iptables` and
`nftables` mode in an existing cluster.

[It should recreate its iptables rules if they are deleted]: https://github.com/kubernetes/kubernetes/blob/v1.27.0/test/e2e/network/networking.go#L550
[`TestUnderTemporaryNetworkFailure`]: https://github.com/kubernetes/kubernetes/blob/v1.27.0-alpha.2/test/e2e/framework/network/utils.go#L1078

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

#### Scalability & Performance tests

```
<<[UNRESOLVED perfscale ]>>

- For the control plane side, the existing scalability tests are
  probably reasonable, assuming we implement the same
  `NetworkProgrammingLatency` metric as the existing backends.

- For the data plane side, there are tests of
  `InClusterNetworkLatency`, but no one is really looking at the
  results yet and they may need work before they are useable.

- We should also make sure that other metrics (CPU, RAM, I/O, etc)
  remain reasonable in an `nftables` cluster.

<<[/UNRESOLVED]>>
```

### Graduation Criteria

#### Alpha

- `kube-proxy --proxy-mode nftables` available behind a feature gate

- nftables mode has unit test parity with iptables

- An nftables-mode e2e job exists, and passes

- Documentation describes any changes in behavior between the
  `iptables` and `ipvs` modes and the `nftables` mode.

- Documentation explains how to manually clean up nftables rules in
  case things go very wrong.

#### Beta

- At least two releases since Alpha.

- The nftables mode has seen at least a bit of real-world usage.

- No major outstanding bugs.

- nftables mode better unit test coverage than iptables mode
  (currently) has. (It is possible that we will end up adding
  equivalent unit tests to the iptables backend in the process.)

- A "kube-proxy mode-switching" e2e job exists, to confirm that you
  can redeploy kube-proxy in a different mode in an existing cluster.
  Rollback is confirmed to be reliable.

- An nftables e2e periodic perf/scale job exists, and shows
  performance as good as iptables and ipvs.

- Documentation describes any changes in behavior between the
  `iptables` and `ipvs` modes and the `nftables` mode. Any warnings
  that we have decide to add for `iptables` users using functionality
  that behaves differently in `nftables` have been added.

- No UNRESOLVED sections in the KEP. (In particular, we have figured
  out what sort of "API" we will offer for integrating third-party
  nftables rules.)

#### GA

- At least two releases since Beta.

- The nftables mode has seen non-trivial real-world usage.

- The nftables mode has no bugs / regressions that would make us
  hesitate to recommend it.

- We have at least the start of a plan for the next steps (changing
  the default mode, deprecating the old backends, etc).

### Upgrade / Downgrade Strategy

The new mode should not introduce any upgrade/downgrade problems,
excepting that you can't downgrade or feature-disable a cluster using
the new kube-proxy mode without switching it back to `iptables` or
`ipvs` first. (The older kube-proxy would refuse to start if given
`--proxy-mode nftables`, and wouldn't know how to clean up stale
nftables service rules if any were present.)

When rolling out or rolling back the feature, it should be safe to
enable the feature gate and change the configuration at the same time,
since nothing cares about the feature gate except for kube-proxy
itself. Likewise, it is expected to be safe to roll out the feature in
a live cluster, even though this will result in different proxy modes
running on different nodes, because Kubernetes service proxying is
defined in such a way that no node needs to be aware of the
implementation details of the service proxy implementation on any
other node.

### Version Skew Strategy

The feature is isolated to kube-proxy and does not introduce any API
changes, so the versions of other components do not matter.

Kube-proxy has no problems skewing with different versions of itself
across different nodes, because Kubernetes service proxying is defined
in such a way that no node needs to be aware of the implementation
details of the service proxy implementation on any other node.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

The administrator must enable the feature gate to make the feature
available, and then must run kube-proxy with the
`--proxy-mode=nftables` flag.

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: NFTablesProxyMode
  - Components depending on the feature gate:
      - kube-proxy
- [X] Other
  - Describe the mechanism:
      - kube-proxy must be restarted with the new `--proxy-mode`.
  - Will enabling / disabling the feature require downtime of the control
    plane?
      - No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
      - No

###### Does enabling the feature change any default behavior?

Enabling the feature gate does not change any behavior; it just makes
the `--proxy-mode=nftables` option available.

Switching from `--proxy-mode=iptables` or `--proxy-mode=ipvs` to
`--proxy-mode=nftables` will likely change some behavior, depending
on what we decide to do about certain un-loved kube-proxy features
like localhost nodeports. Whatever differences in behavior exist will
be explained clearly by the documentation; this is no different from
users switching from `iptables` to `ipvs`, which initially did not
have feature parity with `iptables`.

(Assuming we eventually make `nftables` the default, then differences
in behavior from `iptables` will be more important, but making it the
default is not part of _this_ KEP.)

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, though it is necessary to clean up the nftables rules that were
created, or they will continue to intercept service traffic. In any
normal case, this should happen automatically when restarting
kube-proxy in `iptables` or `ipvs` mode, however, that assumes the
user is rolling back to a still-new-enough version of kube-proxy. If
the user wants to roll back the cluster to a version of Kubernetes
that doesn't have the nftables kube-proxy code (i.e., rolling back
from Alpha to Pre-Alpha), or if they are rolling back to an external
service proxy implementation (e.g., kpng), then they would need to
make sure that the nftables rules got cleaned up _before_ they rolled
back, or else clean them up manually. (We can document how to do
this.)

(By the time we are considering making the `nftables` backend the
default in the future, the feature will have existed and been GA for
several releases, so at that point, rollback (to another version of
kube-proxy) would always be to a version that still supports
`nftables` and can properly clean up from it.)

###### What happens if we reenable the feature if it was previously rolled back?

It should just work.

###### Are there any tests for feature enablement/disablement?

The actual feature gate enablement/disablement itself is not
interesting, since it only controls whether `--proxy-mode nftables`
can be selected.

We will need an e2e test of switching a node from `iptables` (or
`ipvs`) mode to `nftables`, and vice versa. The Graduation Criteria
currently list this e2e test as being a criterion for Beta, not Alpha,
since we don't really expect people to be switching their existing
clusters over to an Alpha version of kube-proxy anyway.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

The operator is the one who would enable the feature, and they would
know it is in use by looking at the kube-proxy configuration.

###### How can someone using this feature know that it is working for their instance?

- [X] Other (treat as last resort)
  - Details: If Services still work then the feature is working

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

TBD.

We should implement the existing "programming latency" metrics that
the other backends implement (`NetworkProgrammingLatency`,
`SyncProxyRulesLastQueuedTimestamp` / `SyncProxyRulesLastTimestamp`,
and `SyncProxyRulesLatency`). It's not clear if there will be a
distinction between "full syncs" and "partial syncs" that works the
same way as in the `iptables` backend, but if there is, then the
metrics related to that should also be implemented.

It's not clear yet what sort of nftables-specific metrics will be
interesting. For example, in the `iptables` backend we have
`sync_proxy_rules_iptables_total`, which tells you the total number of
iptables rules kube-proxy has programmed. But the equivalent metric in
the `nftables` backend is not going to be as interesting, because many
of the things that are done with rules in the `iptables` backend will
be done with maps and sets in the `nftables` backend. Likewise, just
tallying "total number of rules and set/map elements" is not likely to
be useful, because the entire point of sets and maps is that they have
more-or-less **O(1)** behavior, so knowing the number of elements is
not going to give you much information about how well the system is
likely to be performing.

- [X] Metrics
  - Metric names:
      - `network_programming_duration_seconds` (already exists)
      - `sync_proxy_rules_last_queued_timestamp_seconds` (already exists)
      - `sync_proxy_rules_last_timestamp_seconds` (already exists)
      - `sync_proxy_rules_duration_seconds` (already exists)
      - ...
  - Components exposing the metric:
      - kube-proxy

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

If we change any functionality relative to `iptables` mode (e.g., not
allowing localhost NodePorts by default), it would be good to add
metrics to the `iptables` mode, allowing users to be aware of whether
they are depending on these features.

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

It may require a newer kernel than some current users have. It does
not depend on anything else in the cluster.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

Probably not; kube-proxy will still be using the same
Service/EndpointSlice-monitoring code, it will just be doing different
things locally with the results.

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

It is not expected to...

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

The same way that kube-proxy currently does; updates stop being
processed until the apiserver is available again.

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- Initial proposal: 2023-02-01

## Drawbacks

Adding a new officially-supported kube-proxy implementation implies
more work for SIG Network (especially if we are not able to deprecate
either of the existing backends soon).

Replacing the default kube-proxy implementation will affect many
users.

However, doing nothing would result in a situation where, eventually,
many users would be unable to use the default proxy implementation.

## Alternatives

### Continue to improve the `iptables` mode

We have made many improvements to the `iptables` mode, and could make
more. In particular, we could make the `iptables` mode use IP sets
like the `ipvs` mode does.

However, even if we could solve literally all of the performance
problems with the `iptables` mode, there is still the looming
deprecation issue.

(See also "[The iptables kernel subsystem has unfixable performance
problems](#the-iptables-kernel-subsystem-has-unfixable-performance-problems)".)

### Fix up the `ipvs` mode

Rather than implementing an entirely new `nftables` kube-proxy mode,
we could try to fix up the existing `ipvs` mode.

However, the `ipvs` mode makes extensive use of the iptables API in
addition to the IPVS API. So while it solves the performance problems
with the `iptables` mode, it does not address the deprecation issue.
So we would at least have to rewrite it to be IPVS+nftables rather
than IPVS+iptables.

(See also "[The <code>ipvs</code> mode of kube-proxy will not save
us](#the--mode-of-kube-proxy-will-not-save-us)".)

### Use an existing nftables-based kube-proxy implementation

Discussed in [Notes/Constraints/Caveats](#notesconstraintscaveats).

### Create an eBPF-based proxy implementation

Another possibility would be to try to replace the `iptables` and
`ipvs` modes with an eBPF-based proxy backend, instead of an an
nftables one. eBPF is very trendy, but it is also notoriously
difficult to work with.

One problem with this approach is that the APIs to access conntrack
information from eBPF programs only exist in the very newest kernels.
In particular, the API for NATting a connection from eBPF was only
added in the recently-released 6.1 kernel. It will be a long time
before a majority of Kubernetes users have a kernel new enough that we
can depend on that API.

Thus, an eBPF-based kube-proxy implementation would initially need a
number of workarounds for missing functionality, adding to its
complexity (and potentially forcing architectural choices that would
not otherwise be necessary, to support the workarounds).

One interesting eBPF-based approach for service proxying is to use
eBPF to intercept the `connect()` call in pods, and rewrite the
destination IP before the packets are even sent. In this case, eBPF
conntrack support is not needed (though it would still be needed for
non-local service connections, such as connections via NodePorts). One
nice feature of this approach is that it integrates well with possible
future "multi-network Service" ideas, in which a pod might connect to
a service IP that resolves to an IP on a secondary network which is
only reachable by certain pods. In the case of a "normal" service
proxy that does destination IP rewriting in the host network
namespace, this would result in a packet that was undeliverable
(because the host network namespace has no route to the isolated
secondary pod network), but a service proxy that does `connect()`-time
rewriting would rewrite the connection before it ever left the pod
network namespace, allowing the connection to proceed.

The multi-network effort is still in the very early stages, and it is
not clear that it will actually adopt a model of multi-network
Services that works this way. (It is also _possible_ to make such a
model work with a mostly-host-network-based proxy implementation; it's
just more complicated.)

