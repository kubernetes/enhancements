# KEP-3705: Cloud Dual-Stack --node-ip Handling

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Current behavior](#current-behavior)
  - [Changes to <code>--node-ip</code>](#changes-to-)
  - [Changes to the <code>provided-node-ip</code> annotation](#changes-to-the--annotation)
  - [Changes to cloud providers](#changes-to-cloud-providers)
  - [Example of <code>--node-ip</code> possibilities](#example-of--possibilities)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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

Kubelet supports dual-stack `--node-ip` values for clusters with
no cloud provider (eg, "bare metal" clusters), but not for clusters
using a cloud provider. This KEP proposes to fix that.

## Motivation

### Goals

- Allow administrators of clusters using external cloud providers to
  override the cloud-selected node IPs in a more dual-stack-aware
  manner:

    - Allow administrators to override a single node IP on a node in a
      dual-stack cluster without causing the node to become
      single-stack.

    - Allow administrators to override both node IPs on a node in a
      dual-stack cluster.

    - Allow administrators to override the order of node IPs on a node
      in a dual-stack cluster (ie, requesting that nodes be
      IPv6-primary dual stack rather than IPv4-primary dual stack, or
      vice versa).

    - Allow administrators to configure nodes to be single-stack when
      the cloud provider suggests dual-stack, without needing to
      provide a specific IPv4 or IPv6 node IP.

- Define how kubelet will communicate these new intents to cloud
  providers.

- Update the code in `k8s.io/cloud-provider/node/helpers` to implement
  the needed algorithms for the new behaviors.

- Rename the `alpha.kubernetes.io/provided-node-ip` annotation, which
  has not been alpha for a long time.

### Non-Goals

- Changing the behavior of nodes using legacy cloud providers.

- Changing the node-IP-selection behavior in any currently-existing
  Kubernetes cluster. This means that the default behavior when
  `--node-ip` is not specified will remain the same, and the behavior
  of any currently-allowed `--node-ip` value will remain the same. New
  behavior will only be triggered by `--node-ip` values that would
  have been rejected by kubelet in older clusters.

- Adding the ability for nodes in clusters with cloud providers to use
  node IPs that are not already known to the cloud provider. (In
  particular, this implies that we will continue to not support
  dual-stack nodes in clouds that do not themselves support
  dual-stack.)

- Improving the behavior of autodetecting the node IP in any other
  ways. (Eg, being able to pass a CIDR rather than an IP.) There was
  some discussion of ways we might do this in the abandoned [KEP-1664]
  (which this KEP is a spiritual successor to), but if we want that,
  it can be done later, independently of these changes. (These changes
  were mostly desired in the context of non-cloud-provider clusters
  anyway.)

[KEP-1664]: https://github.com/kubernetes/enhancements/issues/1664

## Proposal

### Risks and Mitigations

As the intention is to not change the user-visible behavior except in
clusters where administrators explicitly make use of the new
functionality, there should be no risk of breaking existing clusters,
nor of surprising administrators by suddenly exposing node services on
unexpected IPs.

## Design Details

### Current behavior

Currently, when `--cloud-provider` is passed to kubelet, kubelet
expects `--node-ip` to be either unset, or a single IP address. (If it
is unset, that is equivalent to passing `--node-ip 0.0.0.0`, which
means "autodetect an IPv4 address, or if there are no usable IPv4
addresses, autodetect an IPv6 address".)

If `--cloud-provider` and `--node-ip` are both specified (and
`--node-ip` is not "`0.0.0.0`" or "`::`"), then kubelet will add an
annotation to the node, `alpha.kubernetes.io/provided-node-ip`. Cloud
providers expect this annotation to conform to the current expected
`--node-ip` syntax (ie, a single value); if it does not, then they
will log an error and not remove the
`node.cloudprovider.kubernetes.io/uninitialized` taint from the node,
causing the node to remain unusable until kubelet is restarted with a
valid (or absent) `--node-ip`.

When `--cloud-provider` is not passed, the `--node-ip` value can also
be a comma-separated pair of dual-stack IP addresses. However, unlike
in the single-stack case, the IPs in the dual-stack case are not
currently allowed to be "unspecified" IPs (ie `0.0.0.0` or `::`); you
can only make a (non-cloud) node be dual-stack if you explicitly
specify both IPs that you want it to use.

### Changes to `--node-ip`

The most obvious required change is that we need to allow
comma-separated dual-stack `--node-ip` values in clusters using
external cloud providers (but _not_ in clusters using legacy cloud
providers).

Additionally, the fact that kubelet does not currently pass
"`0.0.0.0`" and "`::`" to the cloud provider creates a compatibility
problem: we would like for administrators to be able to say "use an
IPv6 node IP but I don't care which one" in cloud-provider clusters
like they can in non-cloud-provider clusters, but for compatibility
reasons, we can't change the existing behavior of "`--cloud-provider
external --node-ip ::`" (which doesn't do what it's "supposed to", but
does have possibly-useful side effects that have led some users to use
it anyway; see [kubernetes #111695].)

So instead, we will add new syntax, and allow administrators to say
"`--node-ip IPv4`" or "`--node-ip IPv6`" if they want to explicitly
require that the cloud provider pick a node IP of a specific family.
(This also improves on the behavior of the existing "`0.0.0.0`" and
"`::`" options, because you almost never actually want the "or fall
back to the other family if there are no IPs of this family" behavior
that "`0.0.0.0`" and "`::`" imply.)

Additionally, we will update the code to allow including "`IPv4`" and
"`IPv6`" in dual-stack `--node-ip` values as well (in both cloud and
non-cloud clusters). This code will have to check the status of the
feature gate until the feature is GA.

[kubernetes #111695]: https://github.com/kubernetes/kubernetes/issues/111695

### Changes to the `provided-node-ip` annotation

Currently, if the user passes an IP address to `--node-ip` which is
not recognized by the cloud provider as being a valid IP for that
node, kubelet will set that value in the `provided-node-ip`
annotation, and the cloud provider will see it, realize that the node
IP request can't be fulfilled, log an error, and leave the node in the
tainted state.

It makes sense to have the same behavior if the user passes a "new"
(eg, dual-stack) `--node-ip` value to kubelet, but the cloud provider
does not recognize the new syntax and thus can't fulfil the request.
Conveniently, we can do this just by passing the dual-stack
`--node-ip` value in the existing annotation; the old cloud provider
will try to parse it as a single IP address, fail, log an error, and
leave the node in the tainted state, which is exactly what we wanted
it to do if it can't interpret the `--node-ip` value correctly.

Therefore, we do not _need_ a new annotation for the new `--node-ip`
values; we can continue to use the existing annotation, assuming
existing cloud providers will treat unrecognized values as errors.

That said, the existing annotation name is
`alpha.kubernetes.io/provided-node-ip` but it hasn't been "alpha" for
a long time. We should fix this. So:

  1. When `--node-ip` is unset, kubelet should delete both the legacy
     `alpha.kubernetes.io/provided-node-ip` annotation and the new
     `kubernetes.io/provided-node-ip` annotation (regardless of
     whether the feature gate is enabled or not, to avoid problems
     with rollback and skew).

  2. When the `CloudNodeIPs` feature is enabled and `--node-ip` is
     set, kubelet will set both the legacy annotation and the new
     annotation. (It always sets them both to the same value, even if
     that's a value that old cloud providers won't understand).

  2. When the `CloudNodeIPs` feature is enabled, the cloud provider
     will use the new `kubernetes.io/provided-node-ip` annotation _if
     the legacy alpha annotation is not set_. (But if both annotations
     are set, it will prefer the legacy annotation, so as to handle
     rollbacks correctly.)

  3. A few releases after GA, kubelet can stop setting the legacy
     annotation, and switch to unconditionally deleting it, and
     setting/deleting the new annotation depending on whether
     `--node-ip` was set or not. Cloud providers will also switch to
     only using the new annotation, and perhaps logging a warning if
     they see a node with the old annotation but not the new
     annotation.

Kubelet will preserve the existing behavior of _not_ passing
"`0.0.0.0`" or "`::`" to the cloud provider, even via the new
annotation. This is needed to preserve backward compatibility with
current behavior in clusters using those `--node-ip` values. However,
it _will_ pass "`IPv4`" and/or "`IPv6`" if they are passed as the
`--node-ip`.

### Changes to cloud providers

Assuming that all cloud providers use the `"k8s.io/cloud-provider"`
code to handle the node IP annotation and node address management, no
cloud-provider-specific changes should be needed; we should be able to
make the needed changes in the `cloud-provider` module, and then the
individual providers just need to revendor to the new version.

### Example of `--node-ip` possibilities

Assume a node where the cloud has assigned the IPs `1.2.3.4`,
`5.6.7.8`, `abcd::1234` and `abcd::5678`, in that order of preference.

("SS" = "Single-Stack", "DS" = "Dual-Stack")

| `--node-ip` value    | New? | Annotation             | Resulting node addresses |
|----------------------|------|------------------------|--------------------------|
| (none)               | no   | (unset)                | `["1.2.3.4", "5.6.7.8", "abcd::1234", "abcd::5678"]` (DS IPv4-primary) |
| `0.0.0.0`            | no   | (unset)                | `["1.2.3.4", "5.6.7.8", "abcd::1234", "abcd::5678"]` (DS IPv4-primary) |
| `::`                 | no   | (unset)                | `["1.2.3.4", "5.6.7.8", "abcd::1234", "abcd::5678"]` (DS IPv4-primary *) |
| `1.2.3.4`            | no   | `"1.2.3.4"`            | `["1.2.3.4"]` (SS IPv4) |
| `9.10.11.12`         | no   | `"9.10.11.12"`         | (error, because the requested IP is not available) |
| `abcd::5678`         | no   | `"abcd::5678"`         | `["abcd::5678"]` (SS IPv6) |
| `1.2.3.4,abcd::1234` | yes* | `"1.2.3.4,abcd::1234"` | `["1.2.3.4", "abcd::1234"]` (DS IPv4-primary) |
| `IPv4`               | yes  | `"IPv4"`               | `["1.2.3.4"]` (SS IPv4) |
| `IPv6`               | yes  | `"IPv6"`               | `["abcd::1234"]` (SS IPv6) |
| `IPv4,IPv6`          | yes  | `"IPv4,IPv6"`          | `["1.2.3.4", "abcd::1234"]` (DS IPv4-primary) |
| `IPv6,5.6.7.8`       | yes  | `"IPv6,5.6.7.8"`       | `["abcd::1234", "5.6.7.8"]` (DS IPv6-primary) |
| `IPv4,abcd::ef01`    | yes  | `"IPv4,abcd::ef01"`    | (error, because the requested IPv6 IP is not available) |

Notes:

  - In the `--node-ip ::` case, kubelet will be expecting a
    single-stack IPv6 or dual-stack IPv6-primary setup and so would
    get slightly confused in this case since the cloud gave it a
    dual-stack IPv4-primary configuration. (In particular, you would
    have IPv4-primary nodes but IPv6-primary pods.)

  - `--node-ip 1.2.3.4,abcd::ef01` was previously valid syntax when
    using no `--cloud-provider`, but was not valid for cloud kubelets.

<<[UNRESOLVED multiple-addresses ]>>

In the examples above, `--node-ip IPv4` means "pick any one IPv4
address", not "use all IPv4 addresses". This seemed to me to be
consistent with the behavior when specifying a specific IP address (in
which case all other IPs of the same NodeAddressType are removed from
the list), but is inconsistent with the behavior of `--node-ip
0.0.0.0` / `--node-ip ::` with legacy cloud providers and bare metal
(where that affects the _sorting_ of IPs but does not do any
_filtering_).

In past discussions (eg https://kep.k8s.io/1664,
https://issues.k8s.io/95768) no one has ever been entirely sure what
the point of having multiple node.status.addresses values is, and
there has never been a way to get multiple node.status.addresses
values (of the same IP family) on "bare metal", so that would tend to
prevent people from designing features that depended on multiple node
addresses. So it's not clear that there's really any downside in
filtering out the "unused" node IPs...

<<[/UNRESOLVED]>>

If the cloud only had IPv4 IPs for the node, then the same examples would look like:

| `--node-ip` value    | New? | Annotation             | Resulting node addresses |
|----------------------|------|------------------------|--------------------------|
| (none)               | no   | (unset)                | `["1.2.3.4", "5.6.7.8"]` (SS IPv4) |
| `0.0.0.0`            | no   | (unset)                | `["1.2.3.4", "5.6.7.8"]` (SS IPv4) |
| `::`                 | no   | (unset)                | `["1.2.3.4", "5.6.7.8"]` (SS IPv4 *) |
| `1.2.3.4`            | no   | `"1.2.3.4"`            | `["1.2.3.4"]` (SS IPv4) |
| `9.10.11.12`         | no   | `"9.10.11.12"`         | (error, because the requested IP is not available) |
| `abcd::5678`         | no   | `"abcd::5678"`         | (error, because the requested IP is not available) |
| `1.2.3.4,abcd::1234` | yes* | `"1.2.3.4,abcd::1234"` | (error, because the requested IPv6 IP is not available) |
| `IPv4`               | yes  | `"IPv4"`               | `["1.2.3.4"]` (SS IPv4) |
| `IPv6`               | yes  | `"IPv6"`               | (error, because no IPv6 IPs are available) |
| `IPv4,IPv6`          | yes  | `"IPv4,IPv6"`          | (error, because no IPv6 IPs are available) |
| `IPv6,5.6.7.8`       | yes  | `"IPv6,5.6.7.8"`       | (error, because no IPv6 IPs are available) |
| `IPv4,abcd::ef01`    | yes  | `"IPv4,abcd::ef01"`    | (error, because the requested IPv6 IP is not available) |

In this case, kubelet would be even more confused in the
`--node-ip ::` case, and some things would likely not work.
By contrast, with `--node-ip IPv6`, the user would get a clear error.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

Most of the changes will be in `k8s.io/cloud-provider/node/helpers`.
There will also be small changes in kubelet startup.

- `k8s.io/kubernetes/pkg/kubelet`: `2023-01-30` - `66.9`
- `k8s.io/kubernetes/pkg/kubelet/nodestatus`: `2023-01-30` - `91.2`
- `k8s.io/kubernetes/vendor/k8s.io/cloud-provider/node/helpers`: `2023-01-30` - `31.7`
- `k8s.io/kubernetes/vendor/k8s.io/cloud-provider/node/helpers/address.go`: `2023-01-30` - `100`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

```
<<[UNRESOLVED integration ]>>

I don't know what existing integration testing of kubelet and cloud
providers there is. Given unit tests for the new `--node-ip` parsing
and `node.status.addresses`-setting code, and e2e tests of some sort,
we probably don't need integration tests.

<<[/UNRESOLVED]>>
```

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

```
<<[UNRESOLVED e2e ]>>

I'm not sure how we currently handle cloud-provider e2e. GCP does not
support IPv6 and `kind` does not use a cloud provider, so we cannot
test the new code/behavior in any of the "core" e2e jobs.

@aojea suggests we _could_ use `kind` with an external cloud provider.
This would presumably have to be a bespoke dummy cloud provider, but
since we are only doing this to test the new `k/cloud-provider` code,
having a mostly-dummied-out cloud provider should not be a problem? It
could perhaps have a default `NodeAddresses` return value of
"<intentionally-wrong-IPv4-IP>, <correct-IPv6-IP>,
<intentionally-wrong-IPv6-IP>, <correct-IPv4-IP>", and then we could
pass `--node-ip <correct-IPv4-IP>,IPv6` to kubelet. If the cloud
provider did not receive and interpret the `--node-ip` value
correctly, then we would end up with node IPs that didn't work and the
test job should fail.

<<[/UNRESOLVED]>>
```

- <test>: <link to test coverage>

### Graduation Criteria

#### Alpha

- New `--node-ip` handling and annotation implemented behind a feature flag
- Unit tests updated

- Initial e2e tests completed and enabled? Unless maybe the e2e tests
  will take a bunch of development work on either kind or the
  cloud-provider CI infrastructure, in which case we might decide to
  pass on e2e testing in Alpha?

#### Beta

- Positive feedback / no bugs
- At least one release after Alpha
- Decision made about whether `--node-ip IPv4` means 1 IPv4 IP or all IPv4 IPs
- e2e tests definitely completed and enabled if they weren't already in Alpha
- e2e test that new annotation gets cleared on rollback

#### GA

- Positive feedback / no bugs
- Two releases after Beta to allow time for feedback

#### Deprecation

- Two releases after GA, kubelet will stop setting the legacy
  `alpha.kubernetes.io/provided-node-ip` annotation (and start
  deleting it). Cloud providers will stop honoring the legacy
  annotation.

- (Two releases after that, we can remove the code to delete the
  legacy annotation when present.)

### Upgrade / Downgrade Strategy

No behavioral changes will happen automatically on upgrade, or
automatically on feature enablement; users must opt in to the feature
by changing their kubelet configuration after upgrading the cluster to
a version that supports the new feature.

On downgrade/disablement, it is necessary to revert the kubelet
configuration changes before downgrading kubelet, or kubelet will fail
to start after downgrade.

### Version Skew Strategy

- Old kubelet / old cloud provider: Kubelet will only set the old
  annotation, and the cloud provider will only read the old
  annotation. Everything works (even if there's a stale new annotation
  left over from a cluster rollback).

- Old kubelet / new cloud provider: Kubelet will only set the old
  annotation. The cloud provider will read the old annotation, and
  will interpret it in the same way an old cloud provider would
  (because all `--node-ip` values accepted by an old kubelet are
  interpreted the same way by both old and new cloud providers).
  Everything works (even if there's a stale new annotation left over
  from a kubelet rollback, because the new cloud provider still
  prefers the old annotation).

- New kubelet, single-stack `--node-ip` value / old cloud provider:
  Kubelet will set both annotations (to the same value). The cloud
  provider will read the old annotation. Everything works, because
  this is an "old" `--node-ip` value, and the old cloud provider knows
  how to interpret it correctly.

- New kubelet, dual-stack `--node-ip` value / old cloud provider:
  Kubelet will set both annotations (to the same value). The cloud
  provider will read the old annotation, but it will _not_ know how to
  interpret it because it's a "new" value. So it will log an error and
  leave the node tainted. (This is the desired result, since the cloud
  provider is not able to obey the `--node-ip` value the administrator
  provided.)

- New kubelet / new cloud provider: Kubelet will set both annotations
  (to the same value). The cloud provider will read the old
  annotation, and will know how to interpret it regardless of whether
  it's an "old" or "new" value. Everything works.

- Future (post-GA) kubelet / GA cloud provider: Kubelet will set
  the new annotation, and delete the old annotation if present. The
  cloud provider will see that only the new annotation is set, and so
  will use that. Everything works (even if there had been a stale old
  annotation left over from the upgrade).

- GA kubelet / Future (post-GA) cloud provider: Kubelet will set
  both annotations (to the same value). The cloud provider will only
  look at the new annotation. Everything works.

- Future (post-GA) kubelet / Future (post-GA) cloud provider: Kubelet
  will set the new annotation, and delete the old annotation if
  present. The cloud provider will only look at the new annotation.
  Everything works.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: CloudNodeIPs
  - Components depending on the feature gate:
    - kubelet
    - cloud-controller-manager

###### Does enabling the feature change any default behavior?

Kubelet will begin setting two annotations on each node rather than
one. However, cloud providers will still prefer the old annotation and
will only read the new annotation if the old one isn't there.

So assuming no bugs, the only visible change after (just) enabling the
feature is the duplicate annotation.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, as long as you also roll back the kubelet configuration to no
longer use the new feature.

One edge case: if a user who was not previously setting `--node-ip`
upgrades, enables the feature, starts using `--node-ip`, and then
rolls back the kubelet and its configuration but _not_ the cloud
provider, then they will end up with a node that has a stale "new"
annotation (with the `--node-ip` value they had been using while
upgraded), but no "old" annotation (because the rolled-back kubelet
configuration no longer specifies `--node-ip` so kubelet will delete
the old annotation). The (not-rolled-back) cloud provider will then
mistake this as being the future rather than the past, assuming
kubelet intentionally set only the new annotation and not the old
annotation, and so it will obey the (stale) new annotation.

That could be prevented by stretching out the feature enablement over
a few more releases, and not having the cloud-provider switch from
"always use the old annotation" to "prefer the old annotation when
it's set but use the new annotation if it's the only one" until a few
releases later. If we were going to do that, then perhaps we should
split the annotation renaming into a separate KEP from the dual-stack
`--node-ip` handling, so that the edge cases of the annotation
renaming don't have to slow down the dual-stack `--node-ip` adoption.

###### What happens if we reenable the feature if it was previously rolled back?

It works.

###### Are there any tests for feature enablement/disablement?

TBD; we should have a test that the annotation is cleared on rollback

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

The operator is the one who would be using the feature (and they can
tell by looking at the kubelet configuration to see if a "new"
`--node-ip` value was passed).

###### How can someone using this feature know that it is working for their instance?

The Node will have the IPs they expect it to have.

- [X] API .status
  - Other field: node.status.addresses

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A. This changes the startup behavior of Nodes (and does not affect
startup speed). There is no ongoing "service".

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A. This changes the startup behavior of Nodes (and does not affect
startup speed). There is no ongoing "service".

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

The feature depends on kubelet/cloud provider communication, but it is
just an update to an existing feature that already depends on
kubelet/cloud provider communication. It does not create any
additional dependencies, and it does not add any new failure modes if
either component fails.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Not really. Kubelet will now create two node-IP-related annotations on
each Node rather than just one, and the value of the annotation may
be slightly larger (eg because it now contains two IP addresses rather
than one), and some users may change their `--node-ip` to take
advantage of the new functionality in a way that would cause more node
IPs to be exposed in `node.status.addresses` than before.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

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

It does not add any new failure modes. (The kubelet and cloud provider
use an annotation and an object field to communicate with each other,
but they _already_ do that. And the failure mode there is just
"updates don't get processed until the API server comes back".)

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

N/A: there are no SLOs.

## Implementation History

- Initial proposal: 2022-12-30

## Drawbacks

The status quo is slightly simpler, but some people need the
additional functionality.

## Alternatives

In the discussion around [KEP-1664] there was talk of replacing
`--node-ip` with a new `--node-ips` (plural) field, but this was
mostly because the behavior of `--node-ip` in clusters with external
cloud providers was incompatible with the behavior of `--node-ip` in
clusters with legacy or no cloud providers and I wasn't sure if we
could get away with changing it to make it consistent. However,
[kubernetes #107750] did just that already, so there's now nothing
standing in the way of synchronizing the rest of the behavior.

[KEP-1664]: https://github.com/kubernetes/enhancements/issues/1664
[kubernetes #107750]: https://github.com/kubernetes/kubernetes/pull/107750
