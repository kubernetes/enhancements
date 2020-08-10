<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up.  KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please ensure to complete all
  fields in that template.  One of the fields asks for a link to the KEP.  You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "title", "authors", "owning-sig",
  "status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary", and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG that are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly.  The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

**Note:** Any PRs to move a KEP to `implementable` or significant changes once
it is marked `implementable` must be approved by each of the KEP approvers.
If any of those approvers is no longer appropriate than changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross cutting KEPs).
-->
# KEP-1664: Better Support for Dual-Stack Node Addresses

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
    - [Background: <code>Node.Status.Addresses</code>, <code>--node-ip</code>, and &quot;Primary Node IP&quot;](#background---and-primary-node-ip)
    - [IPv6 Addresses in <code>Node.Status.Addresses</code>](#ipv6-addresses-in-)
    - [Proposal: <code>--node-ips</code>, Primary Node IP, Secondary Node IP](#proposal--primary-node-ip-secondary-node-ip)
    - [Updated Use of <code>Node.Status.Addresses</code>](#updated-use-of-)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Implementation](#implementation)
    - [With a Cloud Provider (External or Built-In)](#with-a-cloud-provider-external-or-built-in)
    - [On Bare Metal](#on-bare-metal)
  - [Shared Code](#shared-code)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [<code>--node-ip</code> deprecation](#-deprecation)
    - [IPv6 <code>Node.Status.Addresses</code> Gotchas](#ipv6--gotchas)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Deprecate <code>Node.Status.Addresses</code>](#deprecate-)
  - [Add <code>Node.Status.NodeIPs</code>](#add-)
  - [Don't Deprecate <code>--node-ip</code>](#dont-deprecate-)
- [Appendices](#appendices)
  - [Cloud Provider Survey](#cloud-provider-survey)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

- [ ] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] KEP approvers have approved the KEP status as `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

There are problems with the contents and interpretation of
`Node.Status.Addresses` that currently break or complicate some IPv6
and dual-stack scenarios.

An earlier version of this KEP proposed deprecating
`Node.Status.Addresses` in favor of a new field, but various issues
make this complicated and to some extent pointless.

This KEP now proposes deprecating kubelet's `--node-ip` argument and
replacing it with a new slightly-different `--node-ips` argument which
will result in `Node.Status.Addresses` reliably indicating IPv4 and
IPv6 addresses according to the user's wishes.

## Motivation

### Goals

1. Assign dual-stack `Pod.Status.PodIPs` to host-network Pods on nodes
   that have both IPv4 and IPv6 IPs, so they can be targeted as
   endpoints of IPv4, IPv6, or dual-stack Services. (This is
   independent of the question of _how_ the node ends up with both
   IPv4 and IPv6 IPs in `Node.Status.Addresses`.)

2. Make the necessary changes to kubelet to allow bare-metal clusters
   to have dual-stack node IPs (either auto-detected or specified on
   the command line) rather than limiting them to a single node IP.

3. Define how cloud providers should handle IPv4 and IPv6 node IPs in
   different cluster types (single-stack IPv4, single-stack IPv6,
   dual-stack) so as to enable IPv6/dual-stack functionality in
   clusters that want it without accidentally breaking old IPv4-only
   clusters. Update at least a few cloud providers to obey the new
   rules. File issues against the remaining cloud providers pointing
   out what needs to be done.

4. Make built-in cloud providers and external cloud providers behave
   the same way with respect to detecting and overriding the Node
   IP(s). Allow administrators to override both IPv4 and IPv6 Node IPs
   in dual-stack clusters.

5. Find a home for the node-address-handling code which is shared
   between kubelet and external cloud providers. (eg,
   `PatchNodeStatus`, which is currently duplicated between
   `k8s.io/kubernetes/pkg/util/node` and
   `k8s.io/cloud-provider/node/helpers`.) Note that even after all
   built-in cloud providers are deprecated in favor of external ones,
   this code will still be used by both kubelet and external cloud
   providers, because kubelet handles the bare metal case.

### Non-Goals

- Drastically changing the semantics of `Node.Status.Addresses`.

- Requiring cloud providers to implement support for IPv6 / dual-stack
  beyond just returning IPv6 node addresses.

- Updating `Pod.Status.HostIP` for dual-stack. It might be useful to
  add `HostIPs` as with `PodIPs` but that is not entirely necessary,
  and can be done fairly trivially later if we decide it's useful.
  (Though FTR [kubernetes #85443] requests having access to dual-stack
  hostIPs via downward API, which would presumbaly imply having them
  in the actual API as well.)

- Being able to set multiple IPs of the same IP family on bare-metal
  nodes ([kubernetes #42125]).

[kubernetes #85443]: https://github.com/kubernetes/kubernetes/issues/85443
[kubernetes #42125]: https://github.com/kubernetes/kubernetes/issues/42125

## Proposal

### Notes/Constraints/Caveats

#### Background: `Node.Status.Addresses`, `--node-ip`, and "Primary Node IP"

Traditionally, a node has had an implicitly-defined "Primary Node IP",
which is, among other things, used to set `Pod.Status.HostIP` on all
Pods, and `Pod.Status.PodIP` for host-network Pods. It is defined
(independently in several places) as the first `InternalIP` address in
`Node.Status.Addresses`, unless there are no `InternalIP` addresses,
in which case the first `ExternalIP` address. (If the node has neither
`InternalIP` nor `ExternalIP` addresses then the node has no "Primary
Node IP". Kubelet itself does not currently consider this a fatal
error, but other code (like some e2e tests) does.)

On clusters using an external cloud provider, `Node.Status.Addresses`
is set by the cloud provider with no input from kubelet. (Passing
`--node-ip` will require that the indicated IP _exists_, but it won't
cause it to become primary.) In practice, this means that external
cloud providers must always return IPv4 addresses first, since
otherwise single-stack IPv4 clusters would not work correctly. In
turn, that means that single-stack IPv6 clusters _won't_ work
correctly with external cloud providers.

When using a built-in cloud provider, the same list of
cloud-provider-provided addresses is used as a starting point, but:

  1. If the user passes `--node-ip IP` to kubelet, and the provided IP
     is one of the addresses returned by the cloud provider, then all
     other IPs of the same `v1.NodeAddressType` are removed from
     `Node.Status.Addresses` to ensure that the passed-in one becomes
     the Primary Node IP. (If the provided IP is not found, then
     kubelet errors out.)

  2. Otherwise, if the user passes `--node-ip ::` (or `--node-ip
     0.0.0.0`) to kubelet, it will sort the cloud-provider-provided
     addresses by IP family (with the given IP family first). Assuming
     that the cloud returned at least one IP of the intended family,
     this will result in that becoming the Primary Node IP.

  3. (If the user passes no `--node-ip` the list of addresses from the
     cloud provider is used unchanged.)

On bare metal, if an explicit `--node-ip` has been passed, then
kubelet will create a single `InternalIP` address with that IP.
Otherwise it will try:

  1. Parsing the node's hostname as an IP address
  2. Looking up the node's node name (not hostname?!) in DNS
  3. Finding an IP address on the node's default NIC

If `--node-ip ::` was passed, then IPv6 DNS/interface addresses will
be preferred; otherwise IPv4 addresses will be preferred.

Although these rules can end up choosing an IPv6 Primary Node IP when
no `--node-ip` was passed in some situations, this will not actually
result in a fully-functioning node, since kubelet will set up
IPv4-only iptables rules if no `--node-ip` was passed.

#### IPv6 Addresses in `Node.Status.Addresses`

Most cloud providers do not currently include IPv6 addresses in
`Node.Status.Addresses`. (See the "[Cloud Provider
Survey](#cloud-provider-survey)" at the end.) To some extent this is
just because they haven't been updated for IPv6/Dual-Stack support,
but this is also because people are worried about confusing existing
clients by adding IPv6 addresses to a field that historically only
held IPv4 addresses. (Something we have also worried about in other
places. eg, Endpoints.)

Some cloud providers (notably Azure) are already returning IPv6 node
IPs when they exist, without apparent problems. OTOH, the fact that
there haven't been problems with clusters on Azure (a relatively new
platform) may not mean that there wouldn't be problems with clusters
on AWS if the AWS CloudProvider suddenly started returning IPv6
addresses.

On the other other hand, CloudProviders will not be able to return
IPv6 addresses unless the Node actually _has_ IPv6 addresses, and on
most platforms it probably won't unless the user explicitly chose to
have them. So in that case, making CloudProviders suddenly start
returning IPv6 addresses when they exist still won't cause problems,
even in clusters where there is old software that only handles IPv4
addresses in `Node.Status.Addresses`, since the clusters running the
old software presumably don't have any IPv6 node addresses anyway.

If we are concerned about confusing old software, there are three
approaches we could take to fix it:

  1. Add one or more new `v1.NodeAddressType` values to use for IPv6
     addresses. eg, `InternalIPv6` and `ExternalIPv6`. One problem
     with this is that there is code scattered across many components
     (and, eg, network plugins) that assumes that all Nodes have
     either an `InternalIP` address or an `ExternalIP` address, so
     using different types for IPv6 addresses would likely break
     single-stack IPv6 clusters.

  2. Provide CloudProvider-level configuration options to say whether
     IPv6 node addresses should be returned or not. The OpenStack and
     vSphere providers do this currently.

  3. Filter `Node.Status.Addresses` based on desired cluster
     configuration; if the user wants IPv6 addresses to be removed,
     they must configure kubelet to request an obligatorily
     single-stack IPv4 cluster.

     (Of course, they only actually need to do that if they're running
     their obligatorily single-stack IPv4 cluster on
     actually-dual-stack hosts, which they presumably aren't.)

The rest of this assumes the last approach.

#### Proposal: `--node-ips`, Primary Node IP, Secondary Node IP

To allow more consistent behavior without breaking existing users, we
will deprecate `--node-ip` and replace it with a new `--node-ips`
argument with more powerful and more-consistent semantics, and change
the way that we generate, filter, and sort `Node.Status.Addresses` to
more reliably choose and indicate the Primary Node IP, and (when
available) the preferred Secondary Node IP of the opposite IP family.

`--node-ips` can be either a single element describing the node IP
configuration for a single family (indicating an
obligatorily-single-stack node), or else a pair of elements describing
both the IPv4 and IPv6 configurations. Each element can be:

  - a specific IP address, eg, `10.0.0.1` or `fd01::1`, indicating
    that the node _must_ have that IP address for that IP family

  - the string "`ipv4`" or "`ipv6`", indicating that the node can have
    _any_ address of that family (but doesn't necessarily have to).

The default value is `ipv4,ipv6`, which allows for any of single-stack
IPv4, single-stack IPv6, or dual stack (with IPv4 primary). By
contrast, `--node-ips ipv4` would mean single-stack IPv4 only (and
would cause IPv6 addresses to be removed from `Node.Status.Addresses`)
(Note that there is no way to say "dual-stack only".)

```
<<[UNRESOLVED optional-vs-required ]>>

If we wanted to be able to have both "optionally dual-stack" and
"mandatory dual-stack" we could have separate "optional ipv4/6" and
"required ipv4/6" flags, but that seems to result in a lot of
complication for not much benefit.

<<[/UNRESOLVED]>>
```

If `--node-ips` is not specified but `--node-ip` is, then `--node-ips`
is defaulted to an equivalent value as follows:

  - If `--node-ip` is `0.0.0.0` then `--node-ips` is set to `ipv4`.

  - If `--node-ip` is `::` then `--node-ips` is set to `ipv6`.

  - Otherwise (when `--node-ip` is a specific IP address) `--node-ips`
    is set to the same value as `--node-ip`.

It is an error to pass both `--node-ip` and `--node-ips`.

As discussed under [Implementation](#implementation),
`Node.Status.Addresses` will then be set correctly to reflect this.
Once `Node.Status.Addresses` has been set, the rule for actually
finding the Primary Node IP is still the same as it used to be; we
will just adjust `Node.Status.Addresses` to ensure that someone
following the traditional rule will find the IP we want them to.

There is now also a "Secondary Node IP", which uses the same rule as
for Primary Node IP, but only matching IPs of the opposite IP family.
(Thus, the Secondary Node IP is the first `InternalIP` address of the
opposite IP family from the Primary Node IP, or the first such
`ExternalIP` if there is no such `InternalIP`.)

#### Updated Use of `Node.Status.Addresses`

When setting Pod status, kubelet will continue to use the Primary Node
IP for `Pod.Status.HostIP`.

In clusters with the `IPv6DualStack` feature gate enabled (or
post-dual-stack GA), host-network Pods will now get both the Primary
Node IP and the Secondary Node IP as `Pod.Status.PodIPs` (allowing
host-network Pods to be endpoints of both IPv4 and IPv6 Services).

Some methods in `utilnode` such as `GetNodeHostIP` and
`GetPreferredNodeAddress` may be updated to have dual-stack versions
that return multiple IPs. The existing versions are still useful on
their own though.

### Risks and Mitigations

<!--
What are the risks of this proposal and how do we mitigate.  Think broadly.
For example, consider both security and how this will impact the larger
kubernetes ecosystem.

How will security be reviewed and by whom?

How will UX be reviewed and by whom?

Consider including folks that also work outside the SIG or subproject.
-->

TODO

## Design Details

### Implementation

As before, when using an external cloud provider, the external
provider deals with setting `Node.Status.Addresses`, and when using a
built-in cloud provider or bare metal, kubelet deals with setting it.

#### With a Cloud Provider (External or Built-In)

When using an external cloud provider, kubelet will pass the value of
`--node-ips` (explicit or defaulted) to the provider via a Node
annotation. (If the provider finds that kubelet has not set the
`--node-ips` annotation, then it should assume it's running against an
old kubelet, and behave as it would have in previous releases.)

Kubelet or the external cloud provider will now get the provisional
list of IPv4 and IPv6 node addresses from the cloud provider-specific
code (which will hopefully have been updated to return IPv6 node IPs)
and then modify the list of node addresses as follows. ("Kubelet"
below really means "Either kubelet or the external cloud provider".)

  1. When using a built-in cloud provider, if the deprecated
     `--node-ip` argument was passed, then kubelet will first do the
     filtering and sorting associated with that.

  2. If `--node-ips` has a single element, then all addresses of the
     non-matching family will be removed from the list.

  3. If there are no longer any `InternalIP` or `ExternalIP` elements
     in the list, then the provided `--node-ips` is invalid, and
     kubelet will error out.

  4. If either element of `--node-ips` is a specific IP address, and
     that IP address does not appear in the address list, then the
     provided `--node-ips` is invalid and kubelet will error out.

     Likewise, if the indicated IP could not possibly be chosen as the
     Primary/Secondary Node IP because it is an `ExternalIP` and would
     always be ignored in favor of an `InternalIP`, then the provided
     `--node-ips` is invalid and kubelet will error out.

  5. At this point kubelet will determine what the Primary Node IP and
     (if relevant) Secondary Node IP would be based on the current
     address list. If the results are compatible with what is
     specified in `--node-ips`, then we are done.

  6. Otherwise, kubelet will look for an element of the address list
     that satisfies the first element of `--node-ips`, and (if
     `--node-ips` has two elements) an element that satisfies the
     second element of `--node-ips`, and move it/them to the front of
     the list, in the correct order. It is guaranteed at this point
     that at least one of those two elements exists, and that moving
     it/them will result in a list of addresses that matches
     `--node-ips`.

  7. The resulting list is then assigned to `Node.Status.Addresses`.

Eg, suppose `--node-ips` is `ipv4,ipv6` and the initial list of
addresses is `[ fd01::1, fd01::2, 10.0.0.1, 10.0.0.2 ]`. In that case
steps 1-4 would have no effect. Step 5 would determine that the
Primary Node IP was `fd01::1` and the Secondary Node IP was
`10.0.0.1`, which does not match `ipv4,ipv6`. (Had we said `ipv6,ipv4`
instead we'd be done though.) Thus, step 6 would reorder the list to
`[ 10.0.0.1, fd01::1, fd01::2, 10.0.0.2 ]`, and then the Primary Node
IP would be `10.0.0.1` and the Secondary would be `fd01::1`, and we
would match.

With the same list of addresses but `--node-ips 10.0.0.2`, step 2
would remove the two IPv6 addresses, step 4 would confirm that
`10.0.0.2` was in the list, step 5 would find the wrong Primary Node
IP, and step 6 would move the desired IP to the front of the list,
giving `[ 10.0.0.2, 10.0.0.1 ]`.

With `--node-ips ipv4,ipv6` and initial list of addresses `[ 10.0.0.1,
10.0.0.2 ]`, steps 1-4 would have no effect, and step 5 would find
that the Primary Node IP was `10.0.0.1` and there was no Secondary
Node IP. This _is_ compatible with `--node-ips ipv4,ipv6` (since both
elements are optional), so we would keep the list of node addresses
unchanged. Note that we would get exactly the same result with
`--node-ips ipv6,ipv4`.

```
<<[UNRESOLVED reordering ]>>

If we wanted to avoid ever rearranging `Node.Status.Addresses`, we
could add new `NodeAddressType` values `PrimaryIP` and `SecondaryIP`,
and prepend new elements to the list. This would also allow fix the
problem where it's currently impossible to make an `ExternalIP` become
the primary IP.

The down side to this would be that older components using the
original "Primary Node IP" algorithm would not agree with newer
clients about what the node's primary IP was.

Perhaps we could handle that by phasing in the new behavior over
several releases: at first we would add the new addresses _and_ also
rearrange the old ones, and then eventually we could stop doing the
rearrangement.

This would also be slightly annoying for clients though because they
may have to continue dealing with un-updated cloud providers that
don't set the new fields for a long time.

<<[/UNRESOLVED]>>
```

#### On Bare Metal

On bare metal, `Node.Status.Addresses` will be set by kubelet
basically as before, based on some combination of explicitly-provided
IPs, DNS lookups, and IPs from the default interface, except that now
it is based on `--node-ips` rather than `--node-ip`, and it will
potentially add two `InternalIP` addresses rather than just one.

### Shared Code

Currently kubelet and the external cloud provider code each have their
own separate copy of `PatchNodeStatus` (with the code to work around
the strategic patch annotation bug). A straightforward implementation
of the plan above would result in them each also needing their own
copy of the (non-trivial) code to sort/filter addresses based on
`--node-ips`.

The cloud provider cannot use kubelet's version of these functions
because it doesn't have access to `"k8s.io/kubernetes/pkg/util/node"`.
In theory, kubelet could use the cloud provider's version in
`"k8s.io/cloud-provider/node/helpers"`, although this seems wrong,
since the code is also used in the bare metal case, so it shouldn't be
owned by the cloud-provider module.

It would make more sense to move the functions to a lower-level module
where they could be shared by both kubelet and cloud-provider.
`"k8s.io/apimachinery"` seems like the most likely bet.

```
<<[UNRESOLVED code-duplication ]>>

Or we could just have duplicate copies of the code... But in
particular, note that the duplication doesn't go away when all of the
built-in cloud providers are moved out-of-tree, since kubelet still
needs all of that code for bare metal.

<<[/UNRESOLVED]>>
```

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy.  Anything
that would count as tricky in the implementation and anything particularly
challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations).  Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

This is impossible to do full e2e testing of since that would require
setting up multiple cloud environments with a mix of IPv4, IPv6, and
dual-stack. We can add more unit tests though...

TODO

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning),
or by redefining what graduation means.

In general, we try to use the same stages (alpha, beta, GA), regardless how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least 2 releases between beta and
GA/stable, since there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include [conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
-->

TODO

### Upgrade / Downgrade Strategy

#### `--node-ip` deprecation

This can happen in the usual way for deprecating command-line
arguments.

#### IPv6 `Node.Status.Addresses` Gotchas

The proposed behavior has one upgrade caveat:

  - If you currently have a single-stack IPv4 cluster,
  - _and_ aren't passing any `--node-ip` value to kubelet,
  - _and_ have software in/around the cluster that will fail if it
    sees an IPv6 address in `Node.Status.Addresses` even when that
    address comes after the Primary Node IP.
  - _and_ are running on a cloud that supports IPv6
  - _and_ are running on hosts that have IPv6 addresses assigned to
    them even though you weren't using them (at least for Kubernetes),
  - then you will need to update your kubelet configuration to include
    `--node-ips ipv4` to force kubelet to be single-stack-IPv4-only in
    the future.

This should hopefully be uncommon enough to be essentially
non-existent. And also, this does not _technically_ constitute an API
break, since `Node.Status.Addresses` was always theoretically
dual-stack.

The alternative to this would be to make it so that the default value
for `--node-ips` was `ipv4` rather than `ipv4,ipv6`, so that if you
pass no `--node-ips` you always get a single-stack IPv4 cluster, even
in a dual-stack environment. But this seems non-future-friendly.

We can also phase this functionality in over several releases if we
wanted. In the first release, we only return IPv6 addresses if you
explicitly request them via `--node-ips` (or if there are _no_ IPv4
addresses), and we warn if we are filtering out IPv6 addresses that
would be kept by a future release. In the next release, we keep IPv6
addresses unless you explicitly request single-stack IPv4, and we warn
if this is causing previously-filtered-out IPv6 addresses to appear.
In the release after that, we drop the warning.


### Version Skew Strategy

TODO

## Implementation History

<!--
Major milestones in the life cycle of a KEP should be tracked in this section.
Major milestones might include
- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

TODO

## Alternatives

### Deprecate `Node.Status.Addresses`

An earlier draft of this KEP proposed much larger changes, including a
more complicated `NodeIPs` field that was a replacement for
`Node.Status.Addresses` rather than a complement to it. For various
reasons, trying to deprecate `Node.Status.Addresses` turns out to be a
bad idea (and if we were going to do it, we would probably want to get
rid of more of it than I had proposed getting rid of).

### Add `Node.Status.NodeIPs`

Originally I wanted to avoid modifying `Node.Status.Addresses`, and so
I suggested having kubelet create `Node.Status.NodeIPs` showing the
IPs it had picked as primary.

However, since we have to modify `Node.Status.Addresses` to get rid of
IPv6 addresses in single-stack IPv4 clusters, there's less of an
argument for not making other modifications to it.

### Don't Deprecate `--node-ip`

Instead of deprecating `--node-ip` and replacing it with `--node-ips`,
we could extend `--node-ip` to allow multiple values instead. I
decided against primarily because `--node-ip` currently behaves
inconsistently between built-in cloud providers and external cloud
providers, and we would have to either keep that inconsistency or else
change behavior in one of the two cases. (eg, if we made `--node-ip`
work for changing the Primary Node IP when using an external cloud
provider, that could potentially break clusters where the admin was
previously passing a `--node-ip` argument that _wasn't_ changing the
Primary Node IP.)

## Appendices

### Cloud Provider Survey

From the official list of [Cloud Providers]:

- **Alibaba Cloud**: APIs used by CloudProvider imply that IPv6 addresses
exist but it is only fetching IPv4 ones
(https://github.com/denverdino/aliyungo/blob/master/metadata/client.go#L22).

- **AWS**: CloudProvider currently only fetches IPv4 addresses. A PR
to add IPv6 addresses to the built-in provider has been open for
several months (https://github.com/kubernetes/kubernetes/pull/86918).

- **Azure**: Built-in CloudProvider returns IPv6 addresses if they exist.
External CloudProvider is out of sync with the built-in provider and is
IPv4-only.

- **Baidu Cloud**: ??? `kubernetes-sigs/cloud-provider-baiducloud`
only supports a single IP per node, but it appears to depend on a
non-open-source (or at least, non-fetchable-from-the-US today) SDK, so
it's not clear if it's possible for that IP to be IPv6. No useful
English-language documentation.

- **Cloudstack**: Cloudstack itself supports IPv6. Kubernetes CloudProvider
seems very agnostic about IPv4-vs-IPv6 so probably just returns IPv6
node addresses if they exist? Can't find any docs confirming either
the existence or non-existence of IPv6 on Kubernetes on CloudStack.

- **DigitalOcean**: Intentionally IPv4-only for now
(https://github.com/digitalocean/digitalocean-cloud-controller-manager/blob/master/cloud-controller-manager/do/droplets.go#L49).
The underlying cloud supports IPv6 so presumably the CloudProvider
could support IPv6 in the future.

- **GCP**: IPv4-only, no underlying cloud support for IPv6

- **Huawei Cloud**: already offering dual-stack kubernetes as a (beta) product!
https://support.huaweicloud.com/en-us/cce_faq/cce_faq_00222.html

- **IBM Cloud**: can't find CloudProvider source. Googling suggests it does
not support IPv6.

- **OpenStack**: supports IPv6 by default but provides a
CoudProvider-specific config flag to disable it
(https://github.com/kubernetes/cloud-provider-openstack/blob/master/docs/using-openstack-cloud-controller-manager.md#networking)

- **Tencent Cloud**: APIs used by CloudProvider imply that IPv6 addresses
exist but it is only fetching IPv4 ones
(https://github.com/TencentCloud/tencentcloud-cloud-controller-manager/blob/master/vendor/github.com/dbdd4us/qcloudapi-sdk-go/metadata/client.go#L18).

- **vSphere**: does not support IPv6 by default but provides a
CloudProvider-specific flag to enable it
(https://github.com/kubernetes/cloud-provider-vsphere/blob/master/docs/book/cloud_config.md#global)


[Cloud Providers]: https://kubernetes.io/docs/concepts/cluster-administration/cloud-providers/

