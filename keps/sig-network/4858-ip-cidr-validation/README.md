# KEP-4858: IP/CIDR Validation Improvements

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Updated IP/CIDR validity criteria](#updated-ipcidr-validity-criteria)
  - [Affected Fields](#affected-fields)
  - [Canonicalization of Kubernetes-controlled values](#canonicalization-of-kubernetes-controlled-values)
  - [API Warnings](#api-warnings)
  - [Updated Validation](#updated-validation)
    - [Ratcheting validation for pre-existing objects](#ratcheting-validation-for-pre-existing-objects)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
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
  - [Ratcheting validation for immutable fields](#ratcheting-validation-for-immutable-fields)
  - [Fixing pre-existing invalid values](#fixing-pre-existing-invalid-values)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [X] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [X] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
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

    - This will initially be behind a feature gate.

- Before that, do a few releases where IP/CIDR values that would be
  invalid according to the new rules cause apiserver warnings. (This
  is [already done for IPs and CIDRs in Service].)

- For all new IP/CIDR-valued API fields, require that IPv6 addresses
  be in canonical form (e.g., `"fc99::123"`, not `"FC99:0:0::0123"`).
  ("New fields" here effectively means "since v1.27", because all of
  the IP/CIDR-valued fields added since then already had that
  validation requirement anyway.) For "legacy" fields, add apiserver
  warnings for IPv6 addresses in non-canonical form.

- Update the [CEL IP/CIDR validation helpers] to use the same code as
  the new core API validation. (The CEL helpers already have the
  correct semantics; this is just about not having two separate
  implementations.)

[already done for IPs and CIDRs in Service]: https://github.com/kubernetes/kubernetes/blob/v1.31.1/pkg/api/service/warnings.go
[CEL IP/CIDR validation helpers]: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/cel/library/ip.go

### Non-Goals

- Updating/replacing `netutils.ParseIPSloppy` and
  `netutils.ParseCIDRSloppy`. (There is a separate plan for this.)

- Providing `netip.Addr`/`netip.Prefix`-based utilities or increasing
  the usage of those types. (There is a separate plan for this.)

- Adding cluster-level warnings/logs/events when invalid IP/CIDR
  values are seen in old objects, so admins can ensure that they
  eventually get fixed. (This was originally listed as a Goal, but was
  not implemented.)

- Allowing immutable fields to be fixed if they are invalid. (E.g., if
  an existing Service has `clusterIP: 172.030.099.099`, allow changing
  it to `clusterIP: 172.30.99.99` even though `clusterIP` is normally
  immutable. This was originally a Goal, but there was not a
  consensus that it was a good idea _in general_, and it doesn't seem
  particularly important for the specific fields that would actually
  be affected.)

- Fixing invalid IPs/CIDRs in existing objects in etcd. (We could
  still potentially do this later in another KEP.)

- Requiring IPv6 addresses in "legacy" fields to be in canonical form.
  While this would be nice (because it lets you reliably
  compare/hash/sort values as strings rather than needing to parse
  them first), it only really works if we forcibly fix up all legacy
  non-canonical values as well, and we decided against that.

- Any other "drive-by" IP/CIDR-related validation improvements, such
  as:

    - fixing `LoadBalancerSourceRanges` to not allow whitespace around
      the CIDRs

    - fixing Pod `HostAliases` to correctly validate that the
      `Hostnames` field *doesn't* contain IP addresses.

    - expanding the use of `ValidateNonSpecialIP` and making various
      fields stricter about whether they accept loopback, multicast,
      etc.

- Any changes to IP/CIDR validation of CLI arguments / component
  configs. We may want to do some follow-up work here, though in
  general things are a lot better there than with API fields (in that
  CLI/config values generally just get immediately used by the
  component that receives them, and are never seen by any other
  components that might interpret them differently).

## Proposal

### Updated IP/CIDR validity criteria

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

  **The updated validation will reject IPv4 IPs with leading "0"s, and
  CIDR values that contain such IPs.**

- They allow "IPv4-mapped IPv6 addresses", e.g., "`::ffff:1.2.3.4`".
  This form of IP address was part of an old IETF plan to simplify
  dual-stack transition (by allowing an IPv4-only program to be
  quickly converted to a syntactically-IPv6-only program that was
  nonetheless semantically dual-stack capable). But they were never
  widely used, and at this point mostly exist to cause confusion. This
  confusion could, again, allow for smuggling IP addresses past
  validation checks.

  **The updated validation will reject IPv4-mapped IPv6 addresses, and
  CIDR values that contain such IPs.**

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
  plugins, it is not currently used in any Kubernetes API object
  except for the very-CNI-influenced DRA `NetworkDeviceData`.
  Nonethless, `net.ParseCIDR` and `netip.ParsePrefix` accept both
  types, and so we currently consider the "ifaddr" form to be valid in
  fields where we are looking for a subnet (e.g.
  `node.status.podCIDRs`) or a mask (e.g., a NetworkPolicy `ipBlock`).
  If a string such as "`192.168.1.5/24`" is accepted by Kubernetes and
  then passed unmodified to another API, it is not always clear
  whether that API will end up treating it as the equivalent of
  `192.168.1.0/24` or `192.168.1.5/32`, so again, this could result in
  problems.

  **The updated validation will have separate validation functions for
  subnet/mask-like CIDRs and ifaddr-like CIDRs.**

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

### Affected Fields

As of 1.32, the IP/CIDR-valued fields in `k8s.io/api` are:

  - in `core`:
    - `endpoints.subsets[].addresses[].ip`
    - `endpoints.subsets[].notReadyAddresses[].ip`
    - `node.spec.podCIDR`
    - `node.spec.podCIDRs[]`
    - `pod.spec.dnsConfig.nameservers[]`
    - `pod.spec.hostAliases[].ip`
    - `pod.status.hostIP`
    - `pod.status.hostIPs[].ip`
    - `pod.status.podIP`
    - `pod.status.podIPs[].ip`
    - `service.spec.clusterIP`
    - `service.spec.clusterIPs[]`
    - `service.spec.externalIPs[]`
    - `service.spec.loadBalancerSourceRanges[]`
    - `service.status.loadBalancer.ingress[].ip`

  - in `networking`:
    - `ingress.status.loadBalancer.ingress[].ip`
    - `ipaddr.metadata.name`
    - `networkpolicy.spec.egress[].to[].ipBlock.cidr`
    - `networkpolicy.spec.egress[].to[].ipBlock.except[]`
    - `networkpolicy.spec.ingress[].from[].ipBlock.cidr`
    - `networkpolicy.spec.ingress[].from[].ipBlock.except[]`
    - `servicecidr.spec.cidrs[]`

  - in `discovery`:
    - `endpointslice.endpoints[].addresses[]`

  - in `resource`:
    - `resourceclaim.status.devices[].networkData.ips[]`

`ipaddr.metadata.name` and `servicecidr.spec.cidrs[]` already have the
semantics we propose here, so while their validation functions will be
updated to make better use of shared code, their semantics will not
actually change.

Similarly, `resourceclaim.status.devices[].networkData.ips[]` is part
of a feature that was still Alpha when implementation of this KEP
began, so we were able to just make it unconditionally require the
stricter validation rather than having to ratchet it forward.

### Canonicalization of Kubernetes-controlled values

Some of the fields above are normally set by Kubernetes-internal code:

  - Set by the Endpoints Controller
    - `endpoints.subsets[].addresses[].ip`
    - `endpoints.subsets[].notReadyAddresses[].ip`

  - Set by the EndpointSlice and EndpointSlice Mirroring controllers:
    - `endpointslice.endpoints[].addresses[]`

  - Set by the NodeIPAM controller:
    - `node.spec.podCIDR`
    - `node.spec.podCIDRs[]`

  - Set by kubelet:
    - `pod.status.hostIP`
    - `pod.status.hostIPs[].ip`
    - `pod.status.podIP`
    - `pod.status.podIPs[].ip`

  - Set by kube-apiserver:
    - `service.spec.clusterIP`
    - `service.spec.clusterIPs[]`

  - Set by the cloud-provider framework code:
    - `service.status.loadBalancer.ingress[].ip`

We will make sure that the code paths that set these values always set
them to fully-canonical values, even if they initially receive them in
non-canonical form from some other component.

However, we will still need stricter validation for these fields as
well, since all of them are either mutable, or else sometimes set by
external components.

### API Warnings

For several releases before changing the validation rules, the
apiserver will return warnings when anyone sets any of the above
fields to an invalid or non-canonical IP value:

```
spec.dnsConfig.nameservers[1]: non-standard IP address "05.06.07.08" will be considered invalid in a future Kubernetes release: use "5.6.7.8"

spec.hostAliases[0].ip: non-standard IP address "::ffff:1.2.3.4" will be considered invalid in a future Kubernetes release: use "1.2.3.4"

spec.loadBalancerSourceRanges[0]: CIDR value "192.12.2.8/24" is ambiguous in this context (should be "192.12.2.0/24" or "192.12.2.8/32"?)

spec.clusterIPs[1]: IPv6 address "2001:db8:0:0::2" should be in RFC 5952 canonical format ("2001:db8::2")
```

(The IP/CIDR fields in Service have had these warnings since 1.27;
they will be added to other objects in the Alpha phase of this KEP.)

### Updated Validation

After several releases with the warnings, we will update the
validation code to reject new invalid values. (For "legacy" fields, we
will still only warn, rather than reject, on non-canonical IPv6
addresses.)

All in-tree API types have already been modified to use appropriate
`utilvalidation` methods for IP/CIDR validation ([kubernetes #122931],
[kubernetes #123174]), so we just need to update the validation
functions themselves to allow only unambiguous values.

[kubernetes #122931]: https://github.com/kubernetes/kubernetes/pull/122931
[kubernetes #123174]: https://github.com/kubernetes/kubernetes/pull/123174

#### Ratcheting validation for pre-existing objects

We will apply [ratcheting validation] to existing resources containing
IP and CIDR values.

In the case of the four fields above which are not arrays themselves,
and not contained within arrays, this is simple: if `pod.spec.hostIP`,
`pod.spec.podIP`, `node.status.podCIDR`, or `service.spec.clusterIP`
contains an invalid value, and an update the the Pod/Service does not
change the value of that field, then the invalid value should be
allowed to remain.

In the case of array-valued fields, this becomes more complicated; if
a service has `clusterIPs: ["001.002.003.004"]`, should it be possible
to update it to `clusterIPs: ["001.002.003.004", "fd00::1234"]`? Does
that count as not changing the original invalid IP or not?

The approach we have used for all types except Endpoints and
EndpointSlice is: when doing an update, collect the IPs/CIDRs from the
old version of the object, and pass them to the validation function as
a `validOldIPs`/`validOldCIDRs` argument. The validation function will
then allow any of those values to appear anywhere in the array,
without worrying about whether they are new values or were just moved
around. Any IPs/CIDRs that are not in that array will be required to
be valid.

Endpoints and EndpointSlice are treated
differently, because (a) they are large enough that doing additional
address-by-address validation on them could be noticeably slow, (b) in
most cases, they are generated by controllers that only write out
valid IPs anyway, (c) in the case of EndpointSlice, if we were going
to allow moving bad IPs around within a slice without revalidation,
then we ought to allow moving them between related slices too, which
we can't really do.

So for Endpoints and EndpointSlice, the rule is that invalid IPs
are only allowed to remain unfixed if the update leaves the entire
top-level `.subsets` / `.endpoints` unchanged. So you can edit the
labels or annotations without fixing invalid endpoint IPs, but you
can't add a new IP while leaving existing invalid IPs in place.

[ratcheting validation]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md#ratcheting-validation

### Risks and Mitigations

The new validation should increase the security of Kubernetes by
making it impossible to have ambiguously-interpretable IPs/CIDRs in
the future. (At least in _new_ clusters; if we do not ever forcibly
replace bad values in existing objects, then clients will need to
protect against bad values forever.)

The most obvious risk to users is that by tightening validation, we
might break existing clusters. Especially, the fact that we are
enforcing tighter validation for new objects of existing types means
that we might break some users' existing workflows / automation that
were generating now-invalid values. This will be mitigated by having
API warnings for a few releases before we flip the switch, and by
wrapping the new validation in a feature gate that users could disable
if needed while they update their infrastructure.

The new validation logic (to allow legacy invalid values) will be more
complicated than the old logic, and thus potentially more likely to
have bugs.

## Design Details

Assuming all of the UNRESOLVED sections are ignored, most of the
updated validation (not including a feature gate) is already
implemented, in [kubernetes #122550] and [kubernetes #128786]; the KEP
is being written retroactively to make sure we have agreement on that
plan (or not).

[kubernetes #122550]: https://github.com/kubernetes/kubernetes/pull/122550
[kubernetes #128786]: https://github.com/kubernetes/kubernetes/pull/128786

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

The already-merged changes to make IP/CIDR validation more consistent
also included updates to unit tests both for the generic functions
(e.g. `utilvalidation.IsValidIP`) and for some specific fields (e.g.
the weird edge cases of `LoadBalancerSourceRanges`.)

##### Unit tests

We will add tests for the new IP/CIDR warning validation functions,
and for the fields that use them. We will test the behavior of being
able to make unrelated changes to objects with pre-existing invalid
values, and the ability to fix invalid fields of immutable objects, to
the extent agreed on above.

- `k8s.io/apimachinery/pkg/util/validation`: `2024-10-29` - `82.3`
- `k8s.io/kubernetes/pkg/api/pod/warnings.go`: `2024-10-29` - `97.4`
- `k8s.io/kubernetes/pkg/api/service/warnings.go`: `2024-10-29` - `96.8`
- `k8s.io/kubernetes/pkg/apis/core/validation`: `2024-10-29` - `84.4`
- `k8s.io/kubernetes/pkg/apis/discovery/validation`: `2024-10-29` - `98.8`
- `k8s.io/kubernetes/pkg/apis/networking/validation`: `2024-10-29` - `91.4`
- `k8s.io/kubernetes/pkg/registry/core/endpoint`: `2024-10-29` - `0.0`
- `k8s.io/kubernetes/pkg/registry/core/node`: `2024-10-29` - `50.0`
- `k8s.io/kubernetes/pkg/registry/core/pod`: `2024-10-29` - `64.0`
- `k8s.io/kubernetes/pkg/registry/core/service`: `2024-10-29` - `73.8`
- `k8s.io/kubernetes/pkg/registry/discovery/endpointslice`: `2024-10-29` - `79.2`
- `k8s.io/kubernetes/pkg/registry/networking/ingress`: `2024-10-29` - `64.5`
- `k8s.io/kubernetes/pkg/registry/networking/networkpolicy`: `2024-10-29` - `76.9`

##### Integration tests

No new tests, and once the feature gate is locked on, we will remove
`test/integration/apiserver/cve_2021_29923_test.go`, which tests that
it is possible to create new objects with invalid IP values (since it
won't be possible any more).

##### e2e tests

No new tests.

The test `test/e2e/network/funny_ips.go`, which tests the behavior of
kube-proxy when it sees Service/EndpointSlice objects with invalid
IPs, was recently removed, because it was (necessarily) a very
weirdly-designed test, and because it would break once it was no
longer possible to create objects with invalid IPs.

For kube-proxy, there are already [equivalent unit tests] in the
iptables and nftables backends, so there is no loss in coverage. (In
fact, the unit tests caught a bug that the e2e test had missed.)

However, we will lose testing for out-of-tree service proxy
implementations (and out-of-tree NetworkPolicy implementations, etc),
but there doesn't seem to be anything we can do about this: we can
only have an e2e test of "what does the implementation do with bad IP
values?" if we can create Pods / Services / EndpointSlices /
NetworkPolicies / etc with bad IP values. But the whole point of this
KEP is to make that impossible.

(I guess possibly we might be able to set up the e2e test to be able
to write the object directly into etcd, bypassing the apiserver? Or
maybe add an "e2e test run" mode to the apiserver, that makes it
possible to request that it skip validation for a particular
Create/Update? Alternatively, this is an argument for eventually
forcibly fixing all remaining invalid values in etcd, since it will no
longer be possible to test behavior regarding them.)

[equivalent unit tests]: https://github.com/kubernetes/kubernetes/pull/126203

### Graduation Criteria

#### Alpha

- API warnings added.

- Mandatory strict validation for `NetworkDeviceData`.

- All kube-internal controllers only write out canonical IPs/CIDRs.

- Updated validation available behind a feature gate, but disabled by
  default.

- Unit tests added/updated.

#### Beta

- All UNRESOLVED items are resolved (which may possibly result in
  extending Alpha longer, or moving some items out into a separate
  KEP).

- Updated validation available behind a feature gate, enabled by
  default.

- Unit tests added/updated, old e2e and integration tests removed.

#### GA

- Allow time for feedback, etc

- Remove the integration test that now tests unreachable behavior.

### Upgrade / Downgrade Strategy

Admins and users should address any warnings they receive about
objects during the Alpha period, to ensure that they do not run into
problems trying to modify or recreate those objects after the stricter
validation is enabled. In particular, admins should not upgrade to
Beta if they have local tooling/automation that creates objects that
trigger the warnings about values that "will be considered invalid in
a future Kubernetes release".

Admins should also ensure that they have no third-party components in
their cluster that try to create objects with invalid IP/CIDR values.
(Note that we are not aware of any third-party components that have
this problem, and weren't aware of any even before the work on this
KEP began. Indeed, the entire point of the KEP is that many
third-party components already had stricter validation than the
apiserver, so upgrading is more likely to increase compatibility with
third-party components than to decreate it.)

### Version Skew Strategy

As discussed
[above](#canonicalization-of-kubernetes-controlled-values), some
Kubernetes controllers currently fill in API fields with IP/CIDR
values provided by external code that may provide invalid values. (For
example, kubelet sets `pod.status.podIPs` based on the IPs provided by
CRI, which in turn normally come from a CNI plugin.) These components
will be fixed in Alpha to always canonicalize these values before
writing them out. Thus, we cannot go to Beta until 3 releases have
passed since Alpha, to ensure that we do not enable the feature gate
by default while some components may still be writing out invalid
values.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: StrictIPCIDRValidation
  - Components depending on the feature gate:
    - kube-apiserver

###### Does enabling the feature change any default behavior?

Yes, it changes the validation of multiple fields.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes

###### What happens if we reenable the feature if it was previously rolled back?

The same thing as happened when it was enabled the first time.

###### Are there any tests for feature enablement/disablement?

There are unit tests in the `core`, `networking`, and `discovery`
validation code for both the enabled and disabled states. For
instance, [`TestValidateServiceCreate`] checks that an invalid
`.spec.clusterIP` is [allowed with the feature gate disabled] and
[causes an error with the feature gate enabled].

(The [corresponding validation tests in `resource`] are not feature
gated, since `ResourceClaim` always requires valid IPs. There are no
e2e tests for the functionality because it is not possible to create
bad IPs to test with when the feature gate is enabled, as discussed
[above](#e2e-tests).)

[`TestValidateServiceCreate`]: https://github.com/kubernetes/kubernetes/blob/v1.35.0/pkg/apis/core/validation/validation_test.go#L16422
[allowed with the feature gate disabled]: https://github.com/kubernetes/kubernetes/blob/v1.35.0/pkg/apis/core/validation/validation_test.go#L16580-L16586
[causes an error with the feature gate enabled]: https://github.com/kubernetes/kubernetes/blob/v1.35.0/pkg/apis/core/validation/validation_test.go#L16588-L16593
[corresponding validation tests in `resource`]: https://github.com/kubernetes/kubernetes/blob/v1.35.0/pkg/apis/resource/validation/validation_resourceclaim_test.go#L1461

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollback should not be able to fail.

Rollout would fail if the cluster contains objects with invalid
IPs/CIDRs that must be re-created as part of the rollout/upgrade. For
example, if kube-proxy was deployed via a DaemonSet whose template
contained an invalid IP in `.spec.dnsConfig.nameservers[]`, then it
would not be possible to recreate the pod after the apiserver was
updated.

The warnings in the Alpha period will hopefully avert this problem.

Rolling out the feature in a skewed cluster that contains pre-Alpha
components could fail if those pre-Alpha components try to set invalid
IP/CIDR values.

###### What specific metrics should inform a rollback?

None; if anything fails, it is likely to involve user workloads. There
are not any metrics for validation failures (but see the "missing
metrics" question below).

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No.

The feature does not do anything during the actual upgrade/downgrade,
so it can be tested adequately against "static" clusters.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

The rollout makes API validation stricter for several objects.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

It is automatically in use in any cluster where it is enabled.

###### How can someone using this feature know that it is working for their instance?

"Things didn't break"

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

It should not noticeably affect the SLOs of the API server (or any
other component).

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Existing API server SLIs.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

We could add a metric to kube-apiserver for when a Create/Update gets
an IP/CIDR-related warning. That might help admins decide whether it
was safe to upgrade, though it wouldn't especially help them figure
out _why_. A more useful approach might be to make sure that the
warnings get logged, with information about specific objects, which
would be more useful in helping to track down the offending
users/controllers.

(Neither of these was done. There is no real reason why people would
have been creating objects with invalid IPs anyway, and we have not
seen any evidence that anyone is actually encountering the warnings
about the impending validation change, so any additional
metrics/logging seems unnecessary.)

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

The KEP will add new code to the Create/Update path (to check objects
and generate the warnings) for certain types (including Pod and
Service), but it is no heavier than any other increased validation.

The intent is to try to minimize additional validation on Endpoints
and EndpointSlice, since those objects can be particularly large (and
generally don't need extra validation).

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

It shouldn't.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A; it is part of the API server.

###### What are other known failure modes?

Anything that requires an object to be recreated could potentially run
into problems starting in Beta, if it is trying to recreate the object
from a template containing an invalid IP or CIDR. This could happen
automatically (as with the DaemonSet example in the "How can a rollout
or rollback fail?" section above), or as a result of some external
controller or user action (for example, trying to recreate a Service
object with a bad IP after the original is deleted).

In general, there is little reason why users would have intentionally
been creating objects with these invalid values before, so this is not
likely to be common, and hopefully where it does happen, users will
see warnings during Alpha and have a chance to fix things.

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A.

## Implementation History

- 2024-01-01: Proposed as a PR ([kubernetes #122550])
- 2024-10-02: Proposed as a KEP
- 2025-04-25: Alpha in Kubernetes v1.33

## Drawbacks

The drawback of implementing this is that it may require some users to
do some work to comply with the tightened validation. But this is
necessary to fix the security problems.

## Alternatives

A few ideas from the original KEP proposal were not implemented.

### Ratcheting validation for immutable fields

Four of the fields listed above are immutable:

  - `pod.spec.dnsConfig.nameservers[]`
  - `pod.spec.hostAliases[].ip`
  - `service.spec.clusterIP`
  - `service.spec.clusterIPs[]`

It has been suggested that we should allow fixing invalid values in
such fields. That is, you would be allowed to modify them if:

  - the old value does not pass current validation rules, _and_
  - the new value is the canonical representation of the old value.

So given `clusterIP: 172.030.099.099`, you would be allowed to modify
it to `clusterIP: 172.30.99.99`, but not to any other value. (For
example, you could not modify it to `clusterIP: 172.30.99.099`,
because while that is less wrong than the original value, it is still
wrong, and not the canonical representation of that IP.)

This allows people to fix bad values in existing objects, but adds
complexity to validation, and potentially could cause problems with
code that was expecting a more literal definition of "immutable".

We decided against implementing this, because people weren't sure that
it was a good idea in general, and for these specific fields:

  - Even if we allowed changing `pod.spec.dnsConfig.nameservers[]` or
    `pod.spec.hostAliases[].ip` after a Pod was created, the change
    would not actually be reflected into the Pod's environment.

  - Pods are intended to be ephemeral anyway, so it shouldn't be too
    problematic to simply delete the bad Pods and recreate them with
    fixed IP values.

  - For Service `clusterIPs`, in most cases the values are assigned
    (correctly) by the Service controller anyway, so Services with
    invalid `clusterIPs` should be rare (and there has been a warning
    about them at creation time for quite a while anyway).

### Fixing pre-existing invalid values

We would eventually like to be able to know for sure that there are no
invalid IP/CIDR values anywhere in any Kubernetes clusters after a
certain release. If we don't do this, then theoretically all
Kubernetes users have to continue dealing correctly with legacy
IP/CIDR values literally forever (but also it's impossible to have e2e
tests of your legacy IP/CIDR handling behavior because there's no way
to create the legacy values in a new cluster).

Fixing existing invalid values up ourselves, either by updating them
in etcd, or by fixing them on read, is tricky and could possibly break
some users in weird ways.

One way to mitigate the potential damage is to get users to fix up the
invalid values themselves. This could be done by having warnings,
logs, or events that are triggered when the apiserver notices objects
with existing invalid values.

We did not take any action here.
