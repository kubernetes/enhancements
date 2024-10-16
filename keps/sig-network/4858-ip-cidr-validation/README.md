# KEP-4858: IP/CIDR Validation Improvements

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Validation Changes](#validation-changes)
    - [Updated validation criteria](#updated-validation-criteria)
    - [Updates to pre-existing objects with pre-existing invalid fields](#updates-to-pre-existing-objects-with-pre-existing-invalid-fields)
    - [Plan for allowing updates to immutable fields](#plan-for-allowing-updates-to-immutable-fields)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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

Kubernetes's validation of IP addresses and CIDR strings (e.g.
"`192.168.0.0/24`") in API fields has historically been too lax,
accepting any string accepted by the underlying functions
(`net.ParseIP` and `net.ParseCIDR`) even though in some cases these
strings had ambiguous meanings that could lead to security problems.
After [golang 1.17 changed the handling of IP addresses with leading
"0"s] to avoid [CVE-2021-29923], there was consensus that we should
eventually update API validation to be stricter as well, but we did
not do this right away, because we didn't want to retroactively
invalidate existing API objects. This KEP sets forth the plan for
finally moving forward on that.

[golang 1.17 changed the handling of IP addresses with leading "0"s]: https://go-review.googlesource.com/c/go/+/325829
[CVE-2021-29923]: https://nvd.nist.gov/vuln/detail/cve-2021-29923

## Motivation

### Goals

- Update the validation of all IP-valued and CIDR-valued API fields in
  core (`k8s.io/api`) types to require stricter validation, to avoid
  potential security problems with ambiguously-parsable IPs/CIDRs.

    - Don't require pre-existing invalid fields to be fixed on
      unrelated updates. E.g., if an existing Service has `clusterIP:
      172.030.099.099`, allow a user to change the Service's
      `selector` without being forced to fix the bad `clusterIP` at
      the same time.

    - Allow immutable fields to be fixed if they are invalid. E.g., if
      an existing Service has `clusterIP: 172.030.099.099`, allow
      changing it to `clusterIP: 172.30.99.99` even though `clusterIP`
      is normally immutable.

- Update the [CEL IP/CIDR validation helpers] to use the same code as
  the new core API validation. (The CEL helpers already have the
  correct semantics; this is just about not having two separate
  implementations.)

```
<<[UNRESOLVED transition ]>>

- MAYBE do a few releases where IP/CIDR values that are invalid
  according to the new rules cause apiserver warnings, before actually
  changing the validation.

- MAYBE put the new validation behind a `Deprecated` feature gate, so
  that people can turn off the new behavior temporarily without having
  to downgrade?

<<[/UNRESOLVED]>>


<<[UNRESOLVED ipv6-canonical-form ]>>

- MAYBE add apiserver warnings when people specify IPv6 addresses in
  non-canonical form. (e.g., `"FC99:0:0::0123"` rather than
  `"fc99::123"`.)

- MAYBE make a plan to require IPv6 addresses to always be in
  canonical form in _new_ APIs

    - Forcing all values to be in canonical form means you can
      compare/sort/uniquify them as strings rather than needing to
      parse them first.

    - (Only relevant to IPv6 because all of the non-canonical forms
      for IPv4 addresses are now invalid.)

    - (It doesn't make sense to force new values of existing API
      fields to be canonical, because the "compare them as strings"
      thing doesn't work if you have to worry about some objects
      having legacy non-canonical values.)

<<[/UNRESOLVED]>>


<<[UNRESOLVED non-ip-fields ]>>

- MAYBE tighten/make-consistent the validation of other
  networking-related fields at the same time. In particular, some
  fields that are supposed to be hostname-valued also accept IPv4 (but
  not IPv6) addresses, because they just use
  `utilvalidation.IsDNS1123Subdomain`, which requires basically
  "alphanumeric separated by dots", which IPv4 addresses are.

<<[/UNRESOLVED]>>


<<[UNRESOLVED non-special-ip ]>>

- MAYBE revisit the use of `ValidateNonSpecialIP`: certain kinds of
  IPs (loopback, multicast, link-local, etc) do not make sense in many
  contexts, but we only prohibit them in a few places.

<<[/UNRESOLVED]>>


<<[UNRESOLVED cli-validation ]>>

- MAYBE change/consistentify the validation of IP/CIDR values in CLI
  args / component configs. There is less room for changing the
  mandatory validation here without badly breaking people, but we
  could at least add warnings.

<<[/UNRESOLVED]>>
```

[CEL IP/CIDR validation helpers]: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/cel/library/ip.go

### Non-Goals

- ...

## Proposal

### Validation Changes

#### Updated validation criteria

IP and CIDR validation is currently handled by
`netutils.ParseIPSloppy` and `netutils.ParseCIDRSloppy`, which are
simply forks of the golang 1.17 versions of `net.ParseIP` and
`net.ParseCIDR`, to preserve the historic semantics. These functions
have various problems:

- They allow IPv4 addresses to have leading "0"s: e.g.,
  "`012.000.001.002`". The problem with this is that they parse these
  by simply ignoring the extra "0"s (i.e., parsing "`012.000.001.002`"
  as "`12.0.1.2`"), but most libc-based code treats the octets as
  _octal_ in this case (i.e., parsing it as "`10.0.1.2`"). If such an
  IP address is validated in golang code but then the original string
  is passed to something libc-based, this may allow bypassing checks
  on the IP range by taking advantage of the difference in parsing.
  (This was [CVE-2021-29923].)

- They allow "IPv4-mapped IPv6 addresses", e.g., "`::ffff:1.2.3.4`".
  This form of IP address was part of an old IETF plan to simplify
  dual-stack transition (by allowing an IPv4-only program to be
  quickly converted to a syntactically-IPv6-only program that was
  nonetheless semantically dual-stack capable). But they were never
  widely used, and at this point mostly exist to cause confusion. This
  confusion could, again, allow for smuggling IP addresses past
  validation checks.

- They accept two different kinds of "CIDR values":

    - subnets/masks (e.g., "`192.168.1.0/24`", meaning "all IP
      addresses whose first 24 bits match the first 24 bits of
      `192.168.1.0`, which is to say, `192.168.1.0 - 192.168.1.255`")

    - "interface addresses" (or "ifaddrs"), individual IP addresses
      that are assigned to a particular network interface and thus
      associated with the subnet attached to that interface (e.g.,
      "`192.168.1.5/24`", meaning "the IP address `192.168.1.5`, which
      is on the same network segment as the rest of `192.168.1.0/24`).

  Although the latter type of CIDR string (where there are bits set in
  the IP address beyond the "prefix length") is often used by CNI
  plugins, it is not currently used in any Kubernetes API object.
  Nonethless, `net.ParseCIDR` and `netip.ParsePrefix` accept both
  types, and so we currently consider the "ifaddr" form to be valid in
  fields where we are looking for a subnet (e.g.
  `node.status.podCIDRs`) or a mask (e.g., a NetworkPolicy `ipBlock`).
  If a string such as "`192.168.1.5/24`" is accepted by Kubernetes and
  then passed unmodified to another API, it is not always clear
  whether that API will end up treating it as the equivalent of
  `192.168.1.0/24` or `192.168.1.5/32`, so again, this could result in
  problems.

- Though not a problem with _existing_ validation, it is also
  important to note that the new `netip.ParseAddr` function accepts
  addresses with "zone identifiers" attached, such as
  "`fe80::1234%eth0`", meaning "the link-local address `fe80::1234` on
  the network attached to `eth0` (as opposed to the same link-local
  address on some other interface)". While specifying zone identifiers
  is important in some contexts, it should not be necessary for any
  existing Kubernetes API objects, and would confuse any code that
  tries to use `netutils.ParseIPSloppy`, so we need to be careful to
  not accept them.

We will update all in-tree API types to use appropriate
`utilvalidation` methods for IP/CIDR validation, and update the
validation functions to allow only unambiguous values.

The fields whose validation will be updated are:

  - in `core`:
    - `endpoints.subsets[].addresses[].ip`
    - `endpoints.subsets[].notReadyAddresses[].ip`
    - `node.spec.podCIDRs[]`
    - `pod.spec.dnsConfig.nameservers[]`
    - `pod.spec.hostAliases[].ip`
    - `pod.spec.hostIP`
    - `pod.spec.hostIPs[]`
    - `pod.spec.podIP`
    - `pod.spec.podIPs[]`
    - `service.spec.clusterIP`
    - `service.spec.clusterIPs[]`
    - `service.spec.externalIPs[]`
    - `service.spec.loadBalancerSourceRanges[]`
    - `service.status.loadBalancer.ingress[].ip`

  - in `networking`:
    - `ingress.status.loadBalancer.ingress[].ip`
    - `networkpolicy.spec.egress[].to[].ipBlock.cidr`
    - `networkpolicy.spec.egress[].to[].ipBlock.except[]`
    - `networkpolicy.spec.ingress[].from[].ipBlock.cidr`
    - `networkpolicy.spec.ingress[].from[].ipBlock.except[]`
    - `serviceCIDR.spec.cidrs[]`

  - in `discovery`:
    - `endpointslice.endpoints[].addresses[]`

#### Updates to pre-existing objects with pre-existing invalid fields

We want to allow making changes to objects containing invalid
IPs/CIDRs, as long as the IPs/CIDRs themselves are not changed.
However, in the case of array-valued fields, it seems best to allow
adding and removing new valid IPs/CIDRs without needing to
simultaneously fix pre-existing invalid IPs/CIDRs.

So, the rule will be rougly: "When validating an Update, any IP/CIDR
that exists in the old version of the field is allowed to exist in the
new version of the field." For NetworkPolicy, that is extended to "any
CIDR that exists anywhere in the old version of the object is allowed
to exist anywhere in the new version of the object" (to allow
inserting new rules into a policy without having to fix invalid CIDRs
in unrelated later rules).

However, `Endpoints` and `EndpointSlice` will be treated differently,
because (a) they are large enough that doing additional validation on
them could be noticeably slow, (b) in most cases, they are generated
by controllers that only write out valid IPs anyway, (c) in the case
of `EndpointSlice`, if we were going to allow moving bad IPs around
within a slice without revalidation, then we ought to allow moving
them between related slices too, which we can't really do.

So for `Endpoints` and `EndpointSlice`, the rule will be that invalid
IPs are only allowed to remain unfixed if the update leaves the entire
`.subsets` / `.addresses` unchanged. So you can edit the labels or
annotations without fixing invalid endpoint IPs, but you can't add a
new IP while leaving existing invalid IPs in place.

#### Plan for allowing updates to immutable fields

Four of the fields listed above are immutable:

  - `pod.spec.dnsConfig.nameservers[]`
  - `pod.spec.hostAliases[].ip`
  - `service.spec.clusterIP`
  - `service.spec.clusterIPs[]`

For these fields we will add the special rule that you are allowed to
modify them if:

  - the old value does not pass current validation rules, _and_
  - the new value is the canonical representation of the old value.

So given `clusterIP: 172.030.099.099`, you would be allowed to modify
it to `clusterIP: 172.30.99.99`, but not to any other value. (For
example, you could not modify it to `clusterIP: 172.30.99.099`,
because while that is less wrong than the original value, it is still
wrong, and not the canonical representation of that IP.)

### Risks and Mitigations

The new validation should increase the security of Kubernetes by
making it impossible to have ambiguously-interpretable IPs/CIDRs in
the future.

The most obvious risk to users is that by tightening validation, we
might break existing clusters. Especially, the fact that we are
enforcing tighter validation for new objects of existing types means
that we might break some users' existing workflows / automation that
were generating now-invalid values. This could be mitigated by having
API warnings for a few releases before we flip the switch, or by
having a `Deprecated` feature gate that users could disable if needed
while they update their infrastructure.

The new validation logic (to allow legacy invalid values) will be more
complicated than the old logic, and thus potentially more likely to
have bugs.

## Design Details

Assuming all of the UNRESOLVED sections are ignored, this is already
implemented, in [PR #122550]; the KEP is being written retroactively
to make sure we have agreement on that plan (or not).

[PR #122550]: https://github.com/kubernetes/kubernetes/pull/122550
### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

We will add validation for the new IP/CIDR validation functions, and
make sure that all of the updated fields have unit tests, and that we
validate the behavior of being able to make unrelated changes to
objects with pre-existing invalid values, and that we validate the
ability to fix invalid fields of immutable objects.

- `k8s.io/apimachinery/pkg/util/validation`: `2024-10-02` - `91.9`
- `k8s.io/kubernetes/pkgs/apis/core/validation`: `2024-10-02` - `84.3`
- `k8s.io/kubernetes/pkgs/apis/discovery/validation`: `2024-10-02` - `98.8`
- `k8s.io/kubernetes/pkgs/apis/networking/validation`: `2024-10-02` - `91.4`

##### Integration tests

No new tests, and we will remove
`test/integration/apiserver/cve_2021_29923_test.go`, which tests that
it is possible to create new objects with invalid IP values (since it
won't be possible any more).

##### e2e tests

No new tests, and we will remove `test/e2e/network/funny_ips.go`,
which tests the behavior of kube-proxy when it sees
Service/EndpointSlice objects with invalid IPs (since it won't be
possible to create such objects any more). We already added [equivalent
unit tests] to the iptables and nftables backends of kube-proxy (which
actually caught a bug in kube-proxy that the e2e test hadn't).

[equivalent unit tests]: https://github.com/kubernetes/kubernetes/pull/126203

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

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

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

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

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

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

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->


```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```
