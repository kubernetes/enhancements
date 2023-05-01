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
  override both node IPs on a node in a dual-stack cluster.

- Define how kubelet will communicate this new intent to cloud
  providers.

- Update the code in `k8s.io/cloud-provider/node/helpers` to implement
  the needed algorithms for the new behaviors.

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

- Improving node IP handling in any other ways. The original version
  of this KEP proposed some other `--node-ip` features to help with
  single-stack IPv6 and dual-stack clusters but they turned out to be
  insufficient and would require a larger redesign of node IP handling
  in order to fix.

- Renaming `alpha.kubernetes.io/provided-node-ip` annotation. This was
  also proposed in the original version of this KEP, but is no longer
  planned as part of this feature.

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

We will allow comma-separated dual-stack `--node-ip` values in
clusters using external cloud providers (but _not_ in clusters using
legacy cloud providers).

No other changes to `--node-ip` handling are being made as part of
this KEP.

### Changes to the `provided-node-ip` annotation

Currently, if the user passes an IP address to `--node-ip` which is
not recognized by the cloud provider as being a valid IP for that
node, kubelet will set that value in the `provided-node-ip`
annotation, and the cloud provider will see it, realize that the node
IP request can't be fulfilled, log an error, and leave the node in the
tainted state.

It makes sense to have the same behavior if the user passes a
dual-stack `--node-ip` value to kubelet, but the cloud provider does
not recognize the new syntax and thus can't fulfil the request.
Conveniently, we can do this just by passing the dual-stack
`--node-ip` value in the existing annotation; the old cloud provider
will try to parse it as a single IP address, fail, log an error, and
leave the node in the tainted state, which is exactly what we wanted
it to do if it can't interpret the `--node-ip` value correctly.

Therefore, we do not need a new annotation for the new `--node-ip`
values; we can continue to use the existing annotation, assuming
existing cloud providers will treat unrecognized values as errors.

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

Notes:

  - In the `--node-ip ::` case, kubelet will be expecting a
    single-stack IPv6 or dual-stack IPv6-primary setup and so would
    get slightly confused in this case since the cloud gave it a
    dual-stack IPv4-primary configuration. (In particular, you would
    have IPv4-primary nodes but IPv6-primary pods.)

  - `--node-ip 1.2.3.4,abcd::ef01` was previously valid syntax when
    using no `--cloud-provider`, but was not valid for cloud kubelets.

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

In this case, kubelet would be even more confused in the
`--node-ip ::` case, and some things would likely not work.

Due to backward-compatibility constraints, it is not possible to end up
with a cluster of every type (single-stack/dual-stack,
IPv4-primary/IPv6-primary) in all cases. For example, given the initial
NodeAddress list:

```yaml
addresses:
  - type:    InternalIP
    address: 10.0.0.1
  - type:    InternalIP
    address: 10.0.0.2
  - type:    InternalIP
    address: fd00::1
  - type:    InternalIP
    address: fd00::2
  - type:    ExternalIP
    address: 192.168.0.1
```

You can request to get a single-stack IPv4 cluster with any of the
three IPv4 IPs as the node IP (`--node-ip 10.0.0.1`, `--node-ip
10.0.0.2`, `--node-ip 192.168.0.1`); a dual-stack IPv4-primary cluster
with any combination of the IPv4 and IPv6 IPs (`--node-ip
10.0.0.2,fd00::2`, etc); or a dual-stack IPv6-primary cluster with any
combination of IPs (`--node-ip fd00::1,192.168.0.1`, etc).

But there is no way to get a single-stack IPv6 cluster, because passing
`--node-ip fd00::1` results in a _dual-stack_ cluster, because the
current, backward-compatible semantics of single-valued `--node-ip`
values means that the IPv4 `ExternalIP` will be preserved.

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

(None.)

##### Unit tests

Most of the changes will be in `k8s.io/cloud-provider/node/helpers`.
There will also be small changes in kubelet startup.

- `k8s.io/kubernetes/pkg/kubelet`: `2023-01-30` - `66.9`
- `k8s.io/kubernetes/pkg/kubelet/nodestatus`: `2023-01-30` - `91.2`
- `k8s.io/kubernetes/vendor/k8s.io/cloud-provider/node/helpers`: `2023-01-30` - `31.7`
- `k8s.io/kubernetes/vendor/k8s.io/cloud-provider/node/helpers/address.go`: `2023-01-30` - `100`

##### Integration tests

Given unit tests for the new `--node-ip` parsing and
`node.status.addresses`-setting code, and e2e tests of some sort, we
probably don't need integration tests.

##### e2e tests

There is now a `[cloud-provider-kind]` that can be used in kind-based
clusters to implement cloud-provider-based functionality.

By modifying this provider to allow manually overriding the default
node IPs, we should be able to create an e2e test job that brings up a
kind cluster with nodes having various IPs, and then tests different
`kubelet --node-ip` arguments to ensure that they have the expected
effect.

[cloud-provider-kind]: https://github.com/kubernetes-sigs/cloud-provider-kind

### Graduation Criteria

#### Alpha

- Dual-stack `--node-ip` handling implemented behind a feature flag

- Unit tests updated

#### Beta

- Positive feedback / no bugs

- At least one release after Alpha

- Implement e2e test using `cloud-provider-kind`.

- Upgrade/rollback have been tested manually

#### GA

- Positive feedback / no bugs

### Upgrade / Downgrade Strategy

No behavioral changes will happen automatically on upgrade, or
automatically on feature enablement; users must opt in to the feature
by changing their kubelet configuration after upgrading the cluster to
a version that supports the new feature.

On downgrade/disablement, it is necessary to revert the kubelet
configuration changes before downgrading kubelet, or kubelet will fail
to start after downgrade.

### Version Skew Strategy

- Old kubelet / new cloud provider: Kubelet will set the annotation.
  The cloud provider will read it and will interpret it in the same
  way an old cloud provider would (because all `--node-ip` values
  accepted by an old kubelet are interpreted the same way by both old
  and new cloud providers). Everything works.

- New kubelet, single-stack `--node-ip` value / old cloud provider:
  Kubelet will set the annotations. The cloud provider will read it.
  Everything works, because this is an "old" `--node-ip` value, and
  the old cloud provider knows how to interpret it correctly.

- New kubelet, dual-stack `--node-ip` value / old cloud provider:
  Kubelet will set the annotation. The cloud provider will read it,
  but it will _not_ know how to interpret it because it's a "new"
  value. So it will log an error and leave the node tainted. (This is
  the desired result, since the cloud provider is not able to obey the
  `--node-ip` value the administrator provided.)

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: CloudDualStackNodeIPs
  - Components depending on the feature gate:
    - kubelet
    - cloud-controller-manager

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, as long as you also roll back the kubelet configuration (to no
longer use the new feature) either earlier or together with the feature
disablement.

###### What happens if we reenable the feature if it was previously rolled back?

It works.

###### Are there any tests for feature enablement/disablement?

No; enabling/disabling the feature gate has no immediate effect, and
changing between a single-stack and dual-stack `--node-ip` value is no
different, code-wise, than changing from one single-stack `--node-ip`
value to a new single-stack value.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

Assuming no drastic bugs (eg, the cloud provider assigns Node X's IP
to Node Y for no obvious reason), just restarting the cloud provider
or kubelet with the new feature enabled-but-unused cannot fail.

The cloud provider will not "re-taint" an existing working node if its
node IP annotation becomes invalid. Thus, if the administrator
accidentally rolls out a kubelet config that does something completely
wrong (eg, specifying a new secondary node IP value which is not
actually one of that node's IPs) then the only effect would be that
the cloud provider will log "Failed to update node addresses for node"
for that node.

The most likely failure mode would be that in the process of
reconfiguring nodes to use the new feature, the administrator
reconfigures them _incorrectly_. (In particular, if nodes were
previously using auto-detected primary node IPs, and the administrator
needs to switch them to manually-specified dual-stack node IPs, they
might end up manually specifying wrong (but valid) primary IPs.) In
that case, the cloud provider would accept the new node IP value and
update the node's addresses, possibly resulting in disruption.
However, this would only happen as each kubelet was reconfigured and
restarted, so as long as the administrator did not roll out the new
configurations to every node simultaneously without any
sanity-checking, they should only break a single node.

###### What specific metrics should inform a rollback?

There are no relevant metrics; however, the feature will only affect
nodes that have been reconfigured to use the new feature, so it should
be obvious to the administrator if the feature is not working
correctly.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

TODO; do a manual test

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

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

N/A. This changes the startup behavior of kubelet (and does not affect
startup speed). There is no ongoing "service".

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A. This changes the startup behavior of kubelet (and does not affect
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

No

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

The original version of this KEP proposed further changes to
`--node-ip` handling:

> Additionally, the fact that kubelet does not currently pass
> "`0.0.0.0`" and "`::`" to the cloud provider creates a compatibility
> problem: we would like for administrators to be able to say "use an
> IPv6 node IP but I don't care which one" in cloud-provider clusters
> like they can in non-cloud-provider clusters, but for compatibility
> reasons, we can't change the existing behavior of "`--cloud-provider
> external --node-ip ::`" (which doesn't do what it's "supposed to", but
> does have possibly-useful side effects that have led some users to use
> it anyway; see [kubernetes #111695].)
> 
> So instead, we will add new syntax, and allow administrators to say
> "`--node-ip IPv4`" or "`--node-ip IPv6`" if they want to explicitly
> require that the cloud provider pick a node IP of a specific family.
> (This also improves on the behavior of the existing "`0.0.0.0`" and
> "`::`" options, because you almost never actually want the "or fall
> back to the other family if there are no IPs of this family" behavior
> that "`0.0.0.0`" and "`::`" imply.)
> 
> Additionally, we will update the code to allow including "`IPv4`" and
> "`IPv6`" in dual-stack `--node-ip` values as well (in both cloud and
> non-cloud clusters). This code will have to check the status of the
> feature gate until the feature is GA.

[kubernetes #111695]: https://github.com/kubernetes/kubernetes/issues/111695

As well as to the Node IP annotation:

> That said, the existing annotation name is
> `alpha.kubernetes.io/provided-node-ip` but it hasn't been "alpha" for
> a long time. We should fix this. So:
> 
>   1. When `--node-ip` is unset, kubelet should delete both the legacy
>      `alpha.kubernetes.io/provided-node-ip` annotation and the new
>      `kubernetes.io/provided-node-ip` annotation (regardless of
>      whether the feature gate is enabled or not, to avoid problems
>      with rollback and skew).
> 
>   2. When the `CloudDualStackNodeIPs` feature is enabled and `--node-ip` is
>      set, kubelet will set both the legacy annotation and the new
>      annotation. (It always sets them both to the same value, even if
>      that's a value that old cloud providers won't understand).
> 
>   2. When the `CloudDualStackNodeIPs` feature is enabled, the cloud provider
>      will use the new `kubernetes.io/provided-node-ip` annotation _if
>      the legacy alpha annotation is not set_. (But if both annotations
>      are set, it will prefer the legacy annotation, so as to handle
>      rollbacks correctly.)
> 
>   3. A few releases after GA, kubelet can stop setting the legacy
>      annotation, and switch to unconditionally deleting it, and
>      setting/deleting the new annotation depending on whether
>      `--node-ip` was set or not. Cloud providers will also switch to
>      only using the new annotation, and perhaps logging a warning if
>      they see a node with the old annotation but not the new
>      annotation.
> 
> Kubelet will preserve the existing behavior of _not_ passing
> "`0.0.0.0`" or "`::`" to the cloud provider, even via the new
> annotation. This is needed to preserve backward compatibility with
> current behavior in clusters using those `--node-ip` values. However,
> it _will_ pass "`IPv4`" and/or "`IPv6`" if they are passed as the
> `--node-ip`.

However, trying to implement this behavior turned up problems:

> While implementing the above behavior, it became clear that retaining
> backward compatibility with old `--node-ip` values means the overall
> behavior is idiosyncratic and full of loopholes. For example, given
> the initial NodeAddress list:
> 
> ```yaml
> addresses:
>   - type:    InternalIP
>     address: 10.0.0.1
>   - type:    InternalIP
>     address: 10.0.0.2
>   - type:    InternalIP
>     address: fd00::1
>   - type:    InternalIP
>     address: fd00::2
>   - type:    ExternalIP
>     address: 192.168.0.1
> ```
> 
> You can request to get a single-stack IPv4 cluster with any of the
> three IPv4 IPs as the node IP (`--node-ip 10.0.0.1`, `--node-ip
> 10.0.0.2`, `--node-ip 192.168.0.1`); a dual-stack IPv4-primary cluster
> with any combination of the IPv4 and IPv6 IPs (`--node-ip
> 10.0.0.2,fd00::2`, etc); a dual-stack IPv6-primary cluster with any
> combination of IPs (`--node-ip fd00::1,192.168.0.1`, etc); or an IPv6
> single-stack cluster using the _first_ IPv6 IP (`--node-ip IPv6`).
> 
> But there is no way to get a single-stack IPv6 cluster using the
> second IPv6 IP, because passing `--node-ip fd00::2` results in a
> _dual-stack_ cluster, because the current, backward-compatible
> semantics of single-valued `--node-ip` values means that the IPv4
> `ExternalIP` will be preserved.

In the discussion around [KEP-1664] there was talk of replacing
`--node-ip` with a new `--node-ips` (plural) field with
new-and-improved semantics, and I think this is what we're going to
have to do if we want to make this better. But that will have to wait
for another KEP.

[KEP-1664]: https://github.com/kubernetes/enhancements/issues/1664
