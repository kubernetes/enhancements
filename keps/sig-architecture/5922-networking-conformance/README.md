# KEP-5922: Explicitly Indicating Future Conformance Tests

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Adding new out-of-tree networking features to conformance](#story-1-adding-new-out-of-tree-networking-features-to-conformance)
    - [Story 2: Tightening conformance requirements on already-required out-of-tree features](#story-2-tightening-conformance-requirements-on-already-required-out-of-tree-features)
    - [Story 3: Ratcheting in a breaking behavior change](#story-3-ratcheting-in-a-breaking-behavior-change)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Criteria for Future Conformance](#criteria-for-future-conformance)
    - [Networking-specific criteria for Future Conformance](#networking-specific-criteria-for-future-conformance)
    - [Linux vs. Windows](#linux-vs-windows)
  - [Picking the Version for Future Conformance](#picking-the-version-for-future-conformance)
  - [Implementation details](#implementation-details)
    - [Test metadata](#test-metadata)
    - [Test Framework Changes](#test-framework-changes)
    - [Sonobuoy / Hydrophone Changes](#sonobuoy--hydrophone-changes)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
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
  - [Continue to never add new networking conformance requirements](#continue-to-never-add-new-networking-conformance-requirements)
  - [YOLO conformance](#yolo-conformance)
  - [&quot;Future conformance&quot; without a formal process](#future-conformance-without-a-formal-process)
  - [&quot;The feature becomes required for conformance once there are 3 out-of-tree implementations&quot;](#the-feature-becomes-required-for-conformance-once-there-are-3-out-of-tree-implementations)
  - [Requiring manual promotion from <code>[FutureConformance]</code> to <code>[Conformance]</code>](#requiring-manual-promotion-from-futureconformance-to-conformance)
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

This KEP proposes a process for formally declaring that an e2e test
will become a conformance test in a specific future release.

While this KEP is being written specifically with SIG Network and
out-of-tree networking features in mind, the overall process is
SIG-agnostic, and could potentially be used for other situations in
which immediately promoting a feature to conformance could break
existing conformant clusters; see the [User Stories](#user-stories).

## Motivation

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

While, in theory, nothing stops us from just announcing that some
externally-implemented feature will be required for conformance in the
future, the fact is that we _haven't_ done that (and, as noted in [the
second user story below], we have been actively avoiding promoting a
specific test that we want to promote, because of concerns about
external breakage).

According to [the CNCF's Conformance page], "Users want consistency
when interacting with any installation of Kubernetes". It seems clear
that SIG Network is currently _not_ delivering this:

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

[the second user story below]: (#story-2-tightening-conformance-requirements-on-already-required-out-of-tree-features)
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

- Add a process for tagging an e2e test as `[FutureConformance]` for a
  few releases before switching it to `[Conformance]`.

- Update sonobuoy and hydrophone to support optionally running future
  conformance tests.

- Update the conformance documentation (both internal and CNCF) to
  explain future conformance (and why you would or wouldn't care).

### Non-Goals

- Discussing which existing e2e tests to promote to future
  conformance; that will be handled independently of the KEP (though
  the Graduation Criteria below require that at least one e2e test be
  promoted before this KEP becomes GA).

## Proposal

### User Stories

#### Story 1: Adding new out-of-tree networking features to conformance

As a SIG Network KEP author, I want all users to be able to use the
new feature that I developed, rather than having it work with some
network plugins but not with others.

#### Story 2: Tightening conformance requirements on already-required out-of-tree features

As a SIG Network Lead, I want to promote the e2e test `"should support
named targetPorts that resolve to different ports on different
endpoints"` to conformance ([kubernetes #132954]), since this has been
documented as an important feature of Services since [the earliest
online version of our docs] (despite not having been tested until
1.35). However, I don't want to abruptly break conformance in clusters
using certain out-of-tree service proxy implementations that are known
to currently fail that test.

[kubernetes #132954]: https://github.com/kubernetes/kubernetes/issues/132954
[the earliest online version of our docs]: https://github.com/kubernetes/website/blob/c8dd8b8831db5cd7c862ac5631c4414a53ac021c/docs/user-guide/services/index.md

```
<<[UNRESOLVED] non-networking example >>

Is there a non-network example of something that works in some
currently-conformant clusters, but fails in other currently-conformant
clusters, that we might want to add conformance requirements around?
Maybe something to do with how Nodes can be slightly different on
different cloud platforms?

<<[/UNRESOLVED]>>
```

#### Story 3: Ratcheting in a breaking behavior change

There are security problems in the default configuration of
Kubernetes, which can't be fixed (in the default configuration)
because doing so would break some existing clusters on upgrade. As a
result, we have [an entire page of documentation] describing how to
switch a cluster from _default_ behaviors to _recommended_ ones.

Future conformance could be another solution. Rather than just telling
users "you really ought to enable Pod Security Admission", we could
declare that Pod Security Admission will be enabled by default and
required for conformance as of some future release.

[an entire page of documentation]: https://kubernetes.io/docs/tasks/administer-cluster/securing-a-cluster/

### Risks and Mitigations

The entire KEP is about reducing risk:

  - Creating a well-defined process for adding new conformance
    requirements for externally-implemented features reduces the risk
    that third party implementers will be caught off guard and not
    have enough time to implement the necessary features.

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

Marking a test for future conformance would cause it to have the
`[FutureConformance]` label and _not_ the `[Conformance]` label. But
we would update our own conformance CI jobs to run the
future conformance tests along with the actual conformance tests. (At
least on Linux.)

People doing conformance testing of Kubernetes distributions would, by
default, _not_ run the "future conformance" tests, but we would
provide instructions on how they could do so by running an alternative
command. We assume this option would mostly be used by developers of
out-of-tree network components (though it might also be used by
distributors, to be aware when components they are distributing are in
danger of falling out of conformance).

### Criteria for Future Conformance

There are no explicit requirements for promotion to "future
conformance" beyond the usual [conformance test requirements]. At a
minimum, this means that a feature proposed for future conformance
must have an existing e2e test that passes in all of the `always_run:
true, optional: false` k/k presubmits, as well as any
release-informing periodics, and it must already be demonstrably
non-flaky at the time it is proposed for future conformance.

[conformance test requirements]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md#conformance-test-requirements

#### Networking-specific criteria for Future Conformance

For networking-related features, the requirement that Future
Conformance features have pre-existing e2e tests implies that:

  - To promote a pod networking-related feature or behavior to "future
    conformance", it would have to already be implemented correctly by:

      - `kind`'s internal version of `kindnet`

      - the pod network implementation used by GCP CI, combining the
        `ptp` CNI plugin and the inter-node routing set up by
        `cloud-provider-gcp`

      - (for HostPort-related changes), the `portmap` CNI plugin

  - To promote a service proxying feature or behavior to "future
    conformance", it would have to already be implemented correctly by
    both the `iptables` and `nftables` backends of `kube-proxy`.

  - To promote a service DNS feature or behavior to "future
    conformance", it would have to already be implemented correctly by
    `CoreDNS`.

(NetworkPolicy, cloud load balancers, Ingress, and Gateway are
considered optional features, and are not covered by conformance,
though if we wanted to make NetworkPolicy a required feature in the
future, we could potentially do that by adding the e2e tests to future
conformance.)

#### Linux vs. Windows

For Linux CI, conformance jobs run in a subset of the configurations
that the required CI jobs in general run in, so anything that already
passes in all required CI jobs should be guaranteed to pass in the
conformance jobs as well.

For Windows, there are currently no CI jobs that run all (or even
most) of the non-conformance e2e tests, and we have had problems in
the past where promoting tests to conformance broke the Windows
conformance jobs (for example, [kubernetes #132019]). Perhaps the best
approach on Windows will be to _not_ automatically run
`[FutureConformance]` tests as part of the Windows conformance jobs,
but instead add a new Windows "future-conformance" job. SIG Windows
will then know when new tests are going to be promoted that will break
Windows, and they can choose to either implement the required
functionality, or else proactively mark those tests as skipped in the
Windows conformance-testing jobs.

```
<<[UNRESOLVED] windows conformance >>

... or we could just run the [FutureConformance] tests in the Windows
conformance job and just make SIG Windows handle the breakage.

<<[/UNRESOLVED]>>
```

[kubernetes #132019]: https://github.com/kubernetes/kubernetes/pull/132019#issuecomment-2922602026

### Picking the Version for Future Conformance

While every case is different, there should be some baseline rules for
adding future conformance criteria:

  - A test using the "future conformance" mechanism should not become
    a conformance requirement until at least 1 year after it is first
    tagged for future conformance.

  - E2e tests of externally-implemented features associated with KEPs
    which go through the alpha → beta → GA cycle, should not become
    conformance requirements until at least:

      - 2 years after the e2e test is first merged to k/k (presumably
        behind an Alpha feature gate).

      - 1 year after the KEP for the feature becomes `status:
        implemented`.

  - New e2e tests of externally-implemented features/behaviors not
    associated with KEPs, or pre-existing e2e tests of
    externally-implemented features/behaviors, should not become
    conformance requirements until at least:

      - 1 year after the e2e test for the feature/behavior is first
        merged to k/k.

  - No new tests for externally-implemented features/behaviors should
    become conformance requirements until at least 1 year after
    *this* KEP becomes `status: implemented`.

(The requirements are stated in terms of years, but as proposed below,
would be implemented in terms of release versions. If the release
cadence changes in the future, it may be necessary to retroactively
adjust the target releases of existing future conformance tests to
cause them to still be promoted at the right time according to the new
release cadence.)

If necessary, a test that was marked for "future conformance" could be
demoted back to non-conformance before the release where it would have
become required.

### Implementation details

#### Test metadata

We want `area/conformance` approval to be required at the point when
the test is first proposed for future conformance, not just when it
goes from future conformance to actual conformance. The easiest way to
do this is to require that something in
`k/k/test/conformance/testdata` change when promoting a test to future
conformance.

To avoid confusion for anyone currently consuming `conformance.yaml`,
we will create a separate `future-conformance.yaml` file for future
conformance tests.

#### Test Framework Changes

We will add a new method,
`framework.WithConformanceVersion(string...)`, which will replace
`framework.ConformanceIt()` / `framework.WithConformance()`, as well
as the `Release` tags in the conformance test doc comments. e.g.:

```diff
 	/*
-		Release: v1.21, v1.35
 		Testname: EndpointSlice, "empty" Service
 		Description: The EndpointSlice controller should create and delete empty EndpointSlices for a Service that matches no pods.
 	*/
-	framework.ConformanceIt("should create and delete EndpointSlices for a Service with a selector that matches no pods", func(ctx context.Context) {
+	ginkgo.It("should create and delete EndpointSlices for a Service with a selector that matches no pods", framework.WithConformanceVersion("1.21", "1.35"), func(ctx context.Context) {
```

`WithConformanceVersion` will compare the (first) indicated version
against the current static compile-time version
(`DefaultKubeBinaryVersion`), and add either a `[Conformance]` or a
`[FutureConformance]` label to the test accordingly, along with a
ginkgo component version constraint (e.g., `[Conformance: [>=1.21]]`;
the version constraint is always for "`Conformance`", whether it
indicates a past or future version).

After this change, the process for immediate conformance promotion and
"future conformance" promotion would be the same:

  - Ensure that the existing e2e test has been non-flaky.

  - Add a conformance doc comment to the test with `Testname` and
    `Description` tags.

  - Add a call to `framework.WithConformanceVersion`, indicating
    either the upcoming release version (for immediate promotion to
    conformance), or a future release version for future promotion.

All `[FutureConformance]` tests targeting a particular release would
flip to `[Conformance]` as part of the PR that updates
`DefaultKubeBinaryVersion` at the start of that release cycle (e.g.
[kubernetes #138548]). The change to the labels on the test would mean
that `update-conformance-yaml.sh` would have to be run as part of that
PR, and the PR would thus need conformance approval. Note that this
shouldn't have any chance of breaking CI, since the required
conformance CI jobs would already have been running the test when it
was `[FutureConformance]` anyway.

[kubernetes #138548]: https://github.com/kubernetes/kubernetes/pull/138548

#### Sonobuoy / Hydrophone Changes

In their existing conformance-testing modes, sonobuoy and hydrophone
both filter the e2e tests to those containing exactly the string
`[Conformance]`.

We can add an additional mode to each of them that will run both
`[Conformance]` and `[FutureConformance]` tests. For sonobuoy, this
would presumably look like

```console
$ sonobuoy run --mode future-conformance
```

while for hydrophone it would be

```console
$ hydrophone --future-conformance
```

If the authors of either program wished to, they could add additional
arguments to allow picking a particular `Conformance` version to
test against, using the associated ginkgo component version
constraint.

### Test Plan

This is proposing a change to testing itself. Other than perhaps an
addition to `test/conformance/walk_test.go` to test the updates to
`conformance.yaml`/`future-conformance.yaml` generation, there is
unlikely to be any automated testing associated with it. Instead, we
will need to manually confirm that all related changes to our e2e
infrastructure have the expected results.

### Graduation Criteria

```
<<[UNRESOLVED] Alpha/Beta/GA >>

The relevant components (`e2e.test`, `sonobuoy`, etc) do not have
feature gates, and the feature is effectively "disabled by default"
even when "GA" anyway, so it seems like this feature can just become
GA in a single release.

We could perhaps have an Alpha/Beta period where the feature was fully
implemented but nobody was allowed to add any additional
future-conformance tests besides the initial one(s) being used to
test/demonstrate the feature, but I don't think there's any real
advantage to that.

<<[/UNRESOLVED]>>
```

#### GA

  - (All UNRESOLVED questions are resolved.)

  - k/k e2e framework has been updated to support `[FutureConformance]`

  - At least 1 e2e test has been marked for `[FutureConformance]`, so
    that the functionality can be tested/demonstrated. (Probably
    [kubernetes #132954].)

  - Linux conformance jobs have been updated to run
    `[FutureConformance]` tests, and Windows conformance jobs have
    been updated based on UNRESOLVED decisions.

  - Conformance test runners have been updated to support future
    conformance testing.

  - External conformance documentation is updated to explain future
    conformance.

### Upgrade / Downgrade Strategy

N/A: the KEP does not describe a change to the runtime behavior of
Kubernetes.

### Version Skew Strategy

The skew-able components here are:

  - The `e2e.test` binary
  - The test runner (sonobuoy / hydrophone)
  - The official ["How to submit conformance results"] instructions

All combinations of old and new will either work correctly or else
error out in an appropriate way:

  - **Old instructions, old or new test runner, old or new `e2e.test`**:
    If the user follows the old instructions (`sonobuoy run
    --certified-conformance` or `hydrophone --conformance`), they will
    always get a standard conformance run.

  - **New instructions, old runner, old or new `e2e.test`**: If the
    user tries to do a future conformance run as documented in the new
    instructions, the old test runner will error out because it won't
    recognize the arguments they gave it. The user should realize that
    they have not successfully tested future conformance (and the
    instructions can provide explicit help for this case).

  - **New instructions, new runner, old `e2e.test`**: If the user
    tries to do a future conformance run as documented in the new
    instructions, the new test runner will select all tests tagged
    `[Conformance]` or `[FutureConformance]`, but there won't be any
    tests tagged `[FutureConformance]`, so the result will be
    identical to a standard conformance run, which is the correct
    result for that version of the tests.

  - **New instructions, new runner, new `e2e.test`**: The user can do
    either a normal conformance run or a future conformance run, and
    either will work as expected.

["How to submit conformance results"]: https://github.com/cncf/k8s-conformance/blob/master/instructions.md

## Production Readiness Review Questionnaire

N/A: the KEP does not describe a change to the runtime behavior of
Kubernetes.

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

N/A

###### Does enabling the feature change any default behavior?

N/A

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

N/A

###### What happens if we reenable the feature if it was previously rolled back?

N/A

###### Are there any tests for feature enablement/disablement?

N/A

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

N/A

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

N/A

### Monitoring Requirements

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

N/A

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

- 2026-02-15: Initial proposal
- 2026-03-07, 2026-05-25: Updated for comments
- 2026-08-28: Further updates and moved from sig-network to sig-architecture

## Drawbacks

Although it would be possible to abuse the process proposed here, it
seems like "being able to add new networking conformance requirements
in a way that is friendly to third party implementations" is strictly
better than "not being able to add new networking conformance
requirements in a way that is friendly to third party
implementations".

## Alternatives

### Continue to never add new networking conformance requirements

Lame.

One could argue that all out-of-tree networking features added since
conformance was first defined in 1.9 are inherently "optional" and
thus not subject to conformance, but that does not match the way that
we document those APIs. More importantly, it mocks the entire premise
of Conformance ("Users want consistency when interacting with any
installation of Kubernetes").

### YOLO conformance

We could add new conformance requirements whenever we want to, without
worrying about whether third party implementations have enough time to
implement them. This would be a good approach if our goal was to make
lots of enemies.

### "Future conformance" without a formal process

We could implement the same general idea as proposed here, but with no
formal infrastructure, by just having a rule like "if you want to
promote a test of an out-of-tree feature to conformance, you have to
write a blog post about it on the developer blog 1 year before you do
it so everyone will know". That would be simpler, but I don't think it
would be better. It would require a lot of boring blog posts...

### "The feature becomes required for conformance once there are 3 out-of-tree implementations"

For features implemented by container runtimes, SIG Node uses the rule
that a required out-of-tree feature can be depended on once both
containerd and cri-o implement it. I don't think we could adopt any rule
like that for networking components. There are many more third-party
networking components than there are container runtimes (including
some important platform-specific ones in the "long tail"), and it
would probably not be either statistically or politically valid to try
to bless a specific small group of networking implementations as
"first among equals" in the way that SIG Node has blessed the two
major container runtimes.

(Note that this doesn't imply we would end up promoting conformance
requirements that *didn't* have 3 out-of-tree implementations or
whatever: if we declare a feature to be required for future
conformance, but nobody implements it, then presumably this would
result in push-back on SIG Network to demote the feature.)

### Requiring manual promotion from `[FutureConformance]` to `[Conformance]`

Rather than having future-conformance tests _automatically_ flip to
full conformance when their targeted release arrives, we could make it
manual instead. In that case, we would probably not replace
`ConformanceIt` with `WithConformanceVersion` as described above, but
would instead add something like `WithFutureConformance(string)` to
set the `[FutureConformance]` label. (Alternatively, we could have
both `WithConformanceVersion` for "graduated" conformance tests and
`WithFutureConformanceVersion` for future ones.) To promote the test
from "future conformance" to "conformance", someone would have to file
a PR to update the test. (Perhaps there could be a release-informing
job pointing out when there were tests that were overdue for
promotion.)

Requiring explicit manual promotion would make it sort of like the
feature gate process, but that's a misleading analogy, because feature
gate promotion _has to_ be manual. We might claim that a particular
feature is going to go to Beta or GA in 1.40, but that is a
prediction, contingent on meeting certain graduation criteria from the
KEP, and there is no way to verify those criteria without human
intervention. In contrast, in the "future conformance" case, the idea
is that all of the (k/k) work has already been done at the point when
we mark it for future conformance, and the promotion from future
conformance to actual conformance is thus inevitable (sort of like
automatically publishing a blog entry once we reach the date it has
been assigned to). So in that case, requiring manual promotion is just
unnecessary extra work (and leaves open the possibility of tests
getting stranded un-promoted).
