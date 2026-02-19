# KEP-5922: Conformance Tests for Out-of-Tree Networking Features

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Conformance tests for out-of-tree features](#story-1-conformance-tests-for-out-of-tree-features)
    - [Story 2: Conformance tests for existing-but-untested functionality](#story-2-conformance-tests-for-existing-but-untested-functionality)
    - [Story 3: Rolling out new features that may confuse existing components](#story-3-rolling-out-new-functionality-that-may-confuse-existing-components)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
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

Networking is unusual among Kubernetes features in that while it is
required for conformance, much of it is implemented outside of
`kubernetes/kubernetes`, by people who are not always Kubernetes
developers, on schedules that are not always in sync with the
Kubernetes release cycle.

This makes it problematic to add new conformance requirements for
Kubernetes networking, since in many cases the conformance test won't
just be validating code that we already implemented in-tree, it will
be immediately imposing a requirement on third parties to have
implemented the feature in their own code before the next Kubernetes
release.

This KEP proposes a process for formally declaring that an e2e test
will become a conformance test in a specific future release, so that
third-party networking implementations will know they are required to
implement that behavior, and will have a reasonable amount of time to
do so.

(In theory the process here is not specific to SIG Network, but AFAIK
SIG Node is the only other SIG that has a component that is required
for conformance but which is externally developed with multiple
implementations (container runtimes) and they already have their own
rule for dealing with that. However, the [third user
story](#story-3-rolling-out-new-functionality-that-may-confuse-existing-components)
below might be applicable to another SIG at some point.)

## Motivation

According to [the CNCF's Conformance page], "Users want consistency
when interacting with any installation of Kubernetes". It seems clear
that SIG Network is _not_ delivering this:

<!-- This is roughly "all SIG Network KEPs that describe a feature
which must be implemented at least partially outside of k/k which are
`status: implemented` and `stage: stable` as of 1.35". The
"Implemented by" column is intentionally vague but is based on a
survey of major implementations. -->

|KEP                                           |GA in|Implemented by?                               |Status           |
|----------------------------------------------|-----|----------------------------------------------|:---------------:|
|[KEP-2447] `service-proxy-name` label         |1.14 |Most service proxies                          |:neutral_face:   |
|[KEP-614]  `SCTPSupport`                      |1.20 |Some pod networks, some service proxies       |:fearful:        |
|[KEP-752]  `EndpointSliceProxying`            |1.21 |All service proxies?                          |:smile:          |
|[KEP-563]  `IPv6DualStack`                    |1.23 |Most pod networks, most service proxies?      |:neutral_face:   |
|[KEP-1138] IPv6 single-stack                  |1.23 |Most pod networks, most service proxies?      |:neutral_face:   |
|[KEP-2365] `IngressClassNamespacedParams`     |1.23 |(unknown%) ingress controllers                |:thinking:       |
|[KEP-2079] `NetworkPolicyEndPort`             |1.25 |Most NetworkPolicy implementations            |:neutral_face:   |
|[KEP-1435] `MixedProtocolLBSVC`               |1.26 |Few cloud load balancers                      |:rage:           |
|[KEP-2086] `ServiceInternalTrafficPolicy`     |1.26 |Some service proxies                          |:fearful:        |
|[KEP-1669] `ProxyTerminatingEndpoints`        |1.28 |Most service proxies                          |:neutral_face:   |
|[KEP-3836] `KubeProxyDrainingTerminatingNodes`|1.31 |Few service proxies, few cloud load balancers?|:rage:           |
|[KEP-1860] `LoadBalancerIPMode`               |1.32 |Few service proxies, some cloud load balancers|:fearful:        |
|[KEP-1880] `MultiCIDRServiceAllocator`        |1.33 |???                                           |:exploding_head: |
|[KEP-2433] `TopologyAwareHints`               |1.33 |Some service proxies                          |:fearful:        |
|[KEP-4444] `ServiceTrafficDistribution`       |1.33 |Some service proxies                          |:fearful:        |
|[KEP-3015] `PreferSameTrafficDistribution`    |1.35 |Few service proxies                           |:rage:           |

[the CNCF's Conformance page]: https://www.cncf.io/training/certification/software-conformance/#benefits

[KEP-563]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/563-dual-stack
[KEP-614]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/614-SCTP-support
[KEP-752]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/0752-endpointslices
[KEP-1138]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/1138-ipv6
[KEP-1435]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/1435-mixed-protocol-lb
[KEP-1669]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/1669-proxy-terminating-endpoints
[KEP-1860]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/1860-kube-proxy-IP-node-binding
[KEP-1880]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/1880-multiple-service-cidrs
[KEP-2079]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/2079-network-policy-port-range
[KEP-2086]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/2086-service-internal-traffic-policy
[KEP-2365]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/2365-ingressclass-namespaced-params
[KEP-2433]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/2433-topology-aware-hints
[KEP-2447]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/2447-Make-kube-proxy-service-abstraction-optional
[KEP-3015]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/3015-prefer-same-node
[KEP-3836]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/3836-kube-proxy-improved-ingress-connectivity-reliability
[KEP-4444]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/4444-service-traffic-distribution

### Goals

- Agree on a process for adding new conformance tests for behavior
  which is implemented by out-of-tree components.

- Make any necessary changes to the Kubernetes e2e framework, and to
  sonobuoy and hydrophone, to support the new process.

- Update the conformance documentation (both internal and CNCF) to
  explain the new process.

### Non-Goals

- Actually promoting any new e2e tests to conformance; that will be
  handled independently of the KEP.

## Proposal

### User Stories

#### Story 1: Conformance tests for out-of-tree features

As a SIG Network KEP author, I want users to be able to use the
feature that I developed, in all conforming Kubernetes clusters.

#### Story 2: Conformance tests for existing-but-untested functionality

As a SIG Network Lead, I want to promote the e2e test `"should support
named targetPorts that resolve to different ports on different
endpoints"` to conformance ([kubernetes #132954]), since this has been
documented as an important feature of Services since [the earliest
online version of our docs]. However, I don't want to abruptly break
conformance in clusters using certain out-of-tree service proxy
implementations that are known to currently fail that test.

[kubernetes #132954]: https://github.com/kubernetes/kubernetes/issues/132954
[the earliest online version of our docs]: https://github.com/kubernetes/website/blob/c8dd8b8831db5cd7c862ac5631c4414a53ac021c/docs/user-guide/services/index.md

#### Story 3: Rolling out new features that may confuse existing components

As a SIG Network Lead, I want users in all clusters to be able to use
the `ServiceCIDR` API ([KEP-1880]) without it breaking third-party
networking components in their cluster.

(Being able to extend the service CIDR range in a running cluster has
long been a requested feature, and it is now _theoretically_ possible.
However, some existing external networking components don't handle the
service CIDRs being changed after cluster install time (since this was
previously not possible), and would end up mis-routing traffic if new
service CIDRs were added later. While these components can be fixed to
take the `ServiceCIDR` API into account, the `ServiceCIDR` API itself
has no way of determining whether a given cluster includes components
that are incompatible with it. For now, we document how cluster
operators can disable the `ServiceCIDR` API via
`ValidatingAdmissionPolicy` if they know it won't work in their
cluster; it would be good if we could require components to eventually
support it.)

### Risks and Mitigations

The entire KEP is about reducing risk:

  - Creating a well-defined process for adding new conformance
    requirements for third-party networking components reduces the
    risk that third party implementers will be caught off guard and
    not have enough time to implement the necessary features.

  - Having a mandatory lag time between announcing the new conformance
    tests and having them actually become required reduces the risk
    that we will accidentally introduce new conformance requirements
    that are impossible for some third parties to implement (like the
    old `timeoutSeconds` parameter for service session affinity, which
    we had to demote from conformance after realizing it was too
    specific to the kube-proxy `iptables` implementation ([kubernetes
    #112806])).

  - Making it less risky to add networking conformance tests means SIG
    Network is likely to add more of them in the future, which will
    increase compatibility between various Kubernetes environments,
    and decrease risk to users when migrating between different
    providers.

[kubernetes #112806]: https://github.com/kubernetes/kubernetes/pull/112806

## Design Details

All conformance tests are labelled with the version of Kubernetes in
which the test first became part of conformance. I propose that we
allow adding conformance tests that are tagged with *future* release
numbers. This would be used to indicate that, while the test is not
required for conformance in the current release, it is intended to
become a conformance requirement in the indicated future release.

For purposes of Kubernetes CI, these "future conformance" tests would
be treated no different from "present-day conformance" tests: all
Kubernetes CI jobs that run `[Conformance]` tests would begin running
them immediately, and the job would fail if the test failed.

However, for people doing conformance testing of Kubernetes
distributions, failures in the "future conformance" tests would merely
result in warnings in the conformance test results, not failures. The
warnings should be obvious to the user, and should indicate in which
release the test is intended to become required for conformance.

There are no explicit requirements for promotion to "future
conformance" beyond the usual [conformance test requirements].
However, the fact that the test would already have to be able to pass
all existing conformance CI jobs would imply that:

  - To promote a pod networking-related feature or behavior to "future
    conformance", it would have to already be implemented correctly by
    both `kindnet` and "GKE Dataplane v1".

  - To promote a service proxying feature or behavior to "future
    conformance", it would have to already be implemented correctly by
    `kube-proxy` (specifically, the `iptables` mode of `kube-proxy`,
    for the moment).

  - To promote a service DNS feature or behavior to "future
    conformance", it would have to already be implemented correctly by
    both `kube-dns` and `CoreDNS`.

(NetworkPolicy, cloud load balancers, Ingress, and Gateway are
considered optional features, and are not covered by conformance.)

When promoting pre-existing e2e tests, or otherwise adding conformance
tests for "old" functionality where we believe most external
implementations already implement it correctly, the future release
must be at least 2 releases in the future.

When promoting tests of new features, where we assume that some
external implementations have not yet implemented the feature at the
time the test is promoted, the future release must be at least
4 releases in the future.

```
<<[UNRESOLVED]>>

FIXME. I just made up those numbers above.

<<[/UNRESOLVED]>>
```

If necessary, a test that was marked for "future conformance" could be
demoted back to non-conformance before the release where it would have
become required.

[conformance test requirements]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md#conformance-test-requirements

### Implementation details

TBD

As proposed above, the "future conformance" tests behave differently
in Kubernetes CI (where they are just treated as present-day
requirements) and in formal conformance testing, despite the fact that
the formal conformance testing process mostly involves running the
conformance test suite in exactly the same way that Kubernetes CI
does...

This may require some combination of changes to:

  - The k/k e2e framework
  - The tools used for testing conformance (sonobuoy and hydrophone)
  - The official conformance testing documentation

There will also need to be a conforming way to run the conformance
suite _without_ including "future conformance" tests, in case there is
a situation where running one of the "future conformance" tests causes
the current version of a third-party component to crash (or otherwise
misbehave).

Given that, the _simplest_ approach would just be to tell people to
run the full present-and-future conformance suite first, and if it
passes, submit those results, but if it fails, re-run just the present
conformance suite, and submit the results of that.

### Test Plan

This is proposing a change to testing itself. Other than perhaps some
unit tests of the changes to `conformance.yaml` generation, there is
unlikely to be any automated testing associated with it. Instead, if
there are any changes to our e2e infrastructure, we will need to just
manually confirm that they have the expected result.

### Graduation Criteria

Not really applicable; the new "future conformance" feature would be
GA as soon as it was fully implemented. Additionally, the primary
changes are to the conformance-testing process, *not* to Kubernetes
itself, so they can be added (or reverted) outside of the release
cycle.

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

Assuming that the KEP requires changes to at least two components out
of (a) kubernetes, (b) sonobuoy/hydrophone, (c) CNCF documentation, we
will need to think about what will happen when someone has a mix of
old and new pieces. (For example, if proper testing ends up requiring
passing a new flag to sonobuoy, then people reading the old CNCF
documentation might end up not passing that flag, and thus not getting
the expected results.)

## Production Readiness Review Questionnaire

N/A: the KEP does not describe a change to the runtime behavior of
Kubernetes.

## Implementation History

- 2026-02-15: Initial proposal

## Drawbacks

Although it would be possible to abuse the process proposed here, it
seems like "being able to add new networking conformance requirements
in a way that is friendly to third party implementations" is strictly
better than "not being able to add new networking conformance
requirements in a way that is friendly to third party
implementations".

## Alternatives

The obvious alternatives are (a) never add new conformance
requirements, and (b) add new conformance requirements whenever we
want to, without worrying about third party implementations. Neither
of these is a good alternative. (One could argue that all out-of-tree
networking features added since conformance was first defined in 1.9
are inherently "optional" and thus not subject to conformance, but
that does not match the way that we document those APIs.)

We could implement the same general idea as proposed here, but with no
formal infrastructure, by just having a rule like "if you want to
promote a test of an out-of-tree feature to conformance, you have to
write a blog post about it on the developer blog 1 year before you do
it so everyone will know". That would be simpler, but I don't think it
would be better.

For features implemented by container runtimes, SIG Node uses the rule
that a required out-of-tree feature can be depended on once both
containerd and cri-o implement it. I don't think we could adopt a rule
like that for networking components. There are many more third-party
networking components than there are container runtimes (including
some important platform-specific ones in the "long tail"), and it
would probably not be either statistically or politically valid to try
to bless a specific small group of networking implementations as
"first among equals" in the way that SIG Node has blessed the two
major container runtimes.
