# KEP-2438: Dual-Stack API Server

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1 - External Clients](#story-1---external-clients)
    - [Story 2 - APIServer Tooling](#story-2---apiserver-tooling)
    - [Story 3 - Wrong-Single-Stack Internal Clients](#story-3---wrong-single-stack-internal-clients)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [kube-apiserver internals](#kube-apiserver-internals)
  - [kube-apiserver Command-Line Arguments](#kube-apiserver-command-line-arguments)
    - [Default / automatic behavior](#default--automatic-behavior)
    - [<code>--bind-address</code>](#--bind-address)
    - [<code>--advertise-address</code>](#--advertise-address)
  - [Pod environment / client-go](#pod-environment--client-go)
  - [Service and EndpointSlice Reconciling](#service-and-endpointslice-reconciling)
    - [Current Behavior](#current-behavior)
    - [New behavior](#new-behavior)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
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

The initial implementation of [dual-stack networking] in Kubernetes
left the apiserver as a single-stack Service. This moves it to
dual-stack (in dual-stack clusters).

[dual-stack networking]: ../563-dual-stack/README.md

## Motivation

### Goals

- Ensure kube-apiserver can listen dual-stack (subject to appropriate
  feature gates):

    - Allow specifying dual-stack `--bind-address` on the command
      line.

    - Autodetect dual-stack bind addresses if possible when no
      explicit `--bind-address` is specified.

- Allow kube-apiserver to advertise dual-stack functionality to other
  components (subject to appropriate feature gates):

    - Allow specifying dual-stack `--advertise-address` on the command
      line (and likewise, autodetecting dual-stack advertise addresses
      from the bind address when `--advertise-address` is not
      explicitly specified).

    - Publish dual-stack `EndpointSlice`s for the "`kubernetes`"
      Service when dual-stack apiserver IPs are available.

    - When configured with both dual-stack bind/advertise addresses
      and dual-stack service CIDRs, create the "`kubernetes`" service as
      `ipFamilyPolicy: RequireDualStack`, and assign it the first IP
      address in both the IPv4 CIDR range and the IPv6 CIDR range.

- Update `rest.InClusterConfig()` to make use of dual-stack IPs, when
  available.

### Non-Goals

- Changing the definition of `KUBERNETES_SERVICE_HOST`

## Proposal

### User Stories

#### Story 1 - External Clients

As a cluster administrator, I want my apiserver to be reachable by
both IPv4-only and IPv6-only clients _outside the cluster_, without
them needing to use a proxy or NAT mechanism.

#### Story 2 - APIServer Tooling

As a Kubernetes add-on developer or Kubernetes service provider, I
want to be able to reliably determine the IP addresses being used by
the apiserver in a user-created cluster, so that I can correctly
create and configure related infrastructure. eg:

  - generating a TLS certificate that includes all of the IP addresses
    that the apiserver is listening on as `subjectAltName`s

  - pointing an external load balancer to the apiserver endpoints

  - pointing an external DNS name to the apiserver endpoints

#### Story 3 - Wrong-Single-Stack Internal Clients

As a developer, I want to run a "wrong-single-stack pod" in a
dual-stack cluster (that is, a pod that is single-stack of the
cluster's secondary IP family), and have it be able to connect to the
apiserver.

This is kind of an odd case, but [it has been seen in the wild], in
the case of a dual-stack IPv6-primary cluster that contains some nodes
that are on an IPv4-only network, which can therefore only host
IPv4-only pods.

[it has been seen in the wild]: https://github.com/kubernetes/enhancements/issues/563#issuecomment-814560270

### Risks and Mitigations

In theory, an administrator might be running an apiserver on a
dual-stack host but not want it to be exposed as a dual-stack service.
However, like most golang-based servers, the apiserver _listens_ on
both IP families by default unless explicitly configured to only
listen on a single IP, so the administrator in this case would already
be losing, and just not realizing it. Making the apiserver advertise
its secondary IP address in this case would arguably be a security
_improvement_, since it would make it easier for the administrator to
notice that it was not configured how they had wanted.

## Design Details

### kube-apiserver internals

Go's network-serving methods don't allow a single HTTP server to
listen on two different addresses. This means that either:

  1. We have to duplicate a bunch of stuff in the apiserver to have
     two listening sockets and two `HTTPServer`s, etc (and, eg, fix
     the shutdown-cleanly code to track both servers, etc).

  2. We only support dual-stack apiservers in the case where
     `BindAddress` is the unspecified address, and you're on Linux
     (where golang can bind a single unspecified-address socket to
     accept both IPv4 and IPv6 connections). (So, eg, you couldn't do
     `--bind-address 127.0.0.1,::1`.)

For alpha, we will implement the second option, and then work with
sig-apimachinery to decide on the best solution for beta and GA.

### kube-apiserver Command-Line Arguments

All of the below assumes that a dual-stack
`--service-cluster-ip-range` has been provided. (Trying to specify a
dual-stack `--bind-address` or `--advertise-address` without
dual-stack `--service-cluster-ip-range` will cause kube-apiserver to
exit with an error.)

#### Default / automatic behavior

When neither `--bind-address` nor `--advertise-address` is explicitly
specified, but there is a dual-stack `--service-cluster-ip-range`, and
the host has both IPv4 and IPv6 addresses, then kube-apiserver will
attempt to bind dual-stack (to `0.0.0.0` and `::`). If this succeeds,
it will create a dual-stack `kubernetes` Service. Otherwise it will
log a warning and fall back to single-stack behavior.

#### `--bind-address`

In a dual-stack cluster, `--bind-address` can now be a comma-separated
pair of IPs rather than only a single IP. (For alpha, the only valid
dual-stack values would be `0.0.0.0,::` and `::,0.0.0.0`.) When
dual-stack bind addresses are specified, kube-apiserver will fail if
it cannot bind both addresses.

If a dual-stack `--advertise-address` is provided but `--bind-address`
is unspecified, then `--bind-address` will default to either
`0.0.0.0,::` or `::,0.0.0.0` (depending on the order of the service
CIDRs) rather than defaulting to only one IP family (and again,
kube-apiserver will fail if it cannot bind both addresses).

(If a dual `--service-cluster-ip-range` is specified, but an explicit
single-stack `--bind-address` is given, then the apiserver will be
single-stack.)

#### `--advertise-address`

In a dual-stack cluster, `--advertise-address` can now be a
comma-separated pair of IPs rather than only a single IP.

If a dual `--advertise-address` is provided but no `--bind-address` is
provided, then `--bind-address` is defaulted as described above.

If a dual `--bind-address` is provided but `--advertise-address` is
single-stack, then the apiserver will accept connections on both
addresses, but the `kubernetes` Service will remain single-stack.

If `--advertise-address` is not specified, but the bind address
(explicit or defaulted) is dual-stack, then the apiserver will
auto-detect both IPv4 and IPv6 addresses to advertise.

### Pod environment / client-go

To fully support the "wrong-single-stack pod" use case, we need to
make `rest.InClusterConfig()` use a dual-stack configuration.
Currently it tries to connect to `KUBERNETES_SERVICE_HOST`.

Changing anything about `KUBERNETES_SERVICE_HOST` would likely break
some existing users, and should be avoided.

Although we could pass a `KUBERNETES_SERVICE_HOST_SECONDARY` variable
to the pod, `rest.InClusterConfig()` cannot make use of two separate
IPs; it must generate a configuration containing a single apiserver
URL.

Another possibility would be for it to use
`https://kubernetes.default.svc.cluster.local` rather than
`https://${KUBERNETES_SERVICE_HOST}`. Presumably though, doing this
unconditionally would break some clusters, so this is also not an
option.

To make this work, `rest.InClusterConfig()` would have to introspect
its own local network setup (eg, using `net.Interfaces()`) to see if
it would be able to reach `KUBERNETES_SERVICE_HOST`, and then use an
alternative (`KUBERNETES_SERVICE_HOST_SECONDARY` or
`kubernetes.default.svc.cluster.local`) only if the primary apiserver
IP is unreachable from that pod.

There was [discussion in an earlier KEP] about whether it is actually
safe to add new `KUBERNETES_*` environment variables, given that we
don't restrict users from using variables with such names for their
own purposes...)

It's possible that there is no solution that is nice enough that it's
actually worth trying to make it work for the admittedly-edge-case-y
scenario of wrong-single-stack pods...

[discussion in an earlier KEP]: https://github.com/kubernetes/enhancements/pull/1418#discussion_r405636588

### Service and EndpointSlice Reconciling

#### Current Behavior

Current kube-apiservers attempt to force the `kubernetes` Service
definition to match its expectations on startup, but do not attempt to
reconcile it after that point. Current apiservers always expect the
Service to be single-stack, so any time a current apiserver reconciles
the `kubernetes` Service, it will set it to be single-stack.

For Endpoints/EndpointSlice, the apiserver can run in two different
"endpoint reconciler" modes: `master-count` (the old algorithm), and
`lease` (the current default, and preferred mode).

In `master-count` mode, each apiserver periodically checks the
`kubernetes` `Endpoints` object, and:

  - If its own IP is not listed there, it adds it.

  - If the total number of IPs listed there is now greater than the
    expected number of masters, it removes a randomly-selected IP
    other than its own.

(The changes to `Endpoints` are likewise synced to `EndpointSlice`.)

This means that when an apiserver is replaced, it may take arbitrarily
long for the `Endpoints` to settle on the correct values (since none
of the remaining apiservers knows which IP is the one that needs to be
removed), but once it reaches the correct values, it will stay correct
until the set of apiservers changes again.

In the `lease` mode, each apiserver periodically writes its IP to a
non-kubernetes-API object in etcd, along with a timestamp and a lease
time. Then it reads all of the other IPs there, and updates the
`Endpoints` and `EndpointSlice` to contain all of the IPs whose leases
have not expired. So in this mode, a stale IP is guaranteed to be
removed once its lease expires, and no apiserver will ever remove a
valid IP from the list.

Since the `master-count` mode is deprecated, and harder to deal with,
we will only support dual-stack apiservers with `lease` mode.

#### New behavior

For ease-of-transition purposes, we will adopt the rule that the
`kubernetes` Service will only be dual-stack when all apiservers
agree that it should be dual-stack. Thus, during upgrades or
single-to-dual-stack migration, there would never be any point where
old and new (or reconfigured and not-yet-reconfigured) apiservers are
fighting over the correct state of the Service.

In the updated `lease` mode, an apiserver that is properly configured
for dual-stack operation will write out two IPs to etcd rather than
just one. If it finds that each of the other (live) apiservers has
also written out two IPs, then it will reconcile the `Service`,
`Endpoints`, and `EndpointSlice` to be dual-stack. Otherwise it will
reconcile them to be single-stack (deleting the secondary-IP-family
`EndpointSlice` if it exists). Thus, the behavior of a dual-stack
apiserver in a cluster containing at least one "old" apiserver will
match the behavior of the old apiserver (except that it will do a
better job of cleaning up stale state than the old apiserver will).

```
<<[UNRESOLVED endpoints-in-etcd ]>>

Need to figure out how dual-stack endpoints will be stored in etcd in
the `lease` mode, so that old apiservers won't get confused by them
(and, eg, try to create a single Endpoints object with both sets of
IPs).

<<[/UNRESOLVED]>>
```

```
<<[UNRESOLVED you-get-an-endpointslice-YOU-get-an-endpointslice-EVERYONE-GETS-AN-ENDPOINTSLICE ]>>

(I still like this idea but I'm thinking not for alpha1 at least.)

Another possibility is that instead of cooperatively maintaining a
single EndpointSlice (or pair of EndpointSlices), each apiserver would
write out its own slice(s) containing only its own IP(s). Clients
would then have to aggregate all of the slices together to get the
full list of active IPs.

The "nice" thing about this is that it means every cluster would be
guaranteed to have at least one multi-slice Service, which means that
every EndpointSlice client would be forced to deal with that
correctly, rather than naively assuming that every Service has at most
one IPv4 and one IPv6 slice.

Assuming we want to do this in single-stack clusters too, it would be
incompatible with the old/current apiserver code, so it would have to
be something that was done only when all of the apiservers were
running the new code. So (combining with the previous UNRESOLVED
point) this would suggest having new apiservers write out two separate
objects to etcd: a legacy object containing just the primary-IP-family
endpoints, and a new object containing all endpoints (even if the
cluster is single-stack and so "all endpoints" is the same as "the
primary-IP-family endpoints"). When all of the IPs in the legacy
object also exist in the new object, then the apiservers would know
that all of the apiservers were running the new code, so they could
switch to the new behavior. (And the legacy object could be dropped a
few releases later.)

<<[/UNRESOLVED]>>
```

### Test Plan

- We will need unit or integration tests of the behavior under skewed
  configurations, as described in ["Service and EndpointSlice
  Reconciling"](#service-and-endpointslice-reconciling). (These tests
  apply both to the present-day case of "enabling the feature gate in
  a cluster which is already dual stack" and the future case of
  "changing the configuration to dual stack in a cluster which already
  has the feature gate enabled", so they'll still be needed even
  post-GA.)

- E2E tests of dual-stack apiserver functionality:

    - If the feature is enabled and the cluster is single-stack, then
      the Service and EndpointSlices should be single-stack

    - If the feature is enabled and the cluster is dual-stack, then
      the Service and EndpointSlices should be dual-stack, and both
      the IPv4 and IPv6 service IPs should accept connections.

### Graduation Criteria

#### Alpha -> Beta Graduation

- Feedback from users

- Unit/integration and e2e tests as described above.

- Confirm that CoreDNS behaves as expected (ie, it is not mistakenly
  special-casing `kubernetes.default` as being obligatorily
  single-stack).

- (Maybe) We must be alpha for at least 2 releases, if we don't want
  to have to support downgrades from Beta to pre-Alpha.

#### Beta -> GA Graduation

- Feedback from users

- 2 examples of distributors/developers/providers making use of
  dual-stack Endpoints

### Upgrade / Downgrade Strategy

Some of the details of upgrades and downgrades are discussed above under
[Service and EndpointSlice Reconciling](#service-and-endpointslice-reconciling).

The current (1.23) apiserver code will not clean up a
secondary-IP-family `EndpointSlice` if one is left behind ([kubernetes
#101070]). Thus, someone downgrading from dual-stack apiservers to a
release with the current code might end up with a stale (but unused)
secondary-IP-family `EndpointSlice`. (This would only happen if all of
the apiserver downgrades after the first happened in less time than
the endpoint reconciling polling interval; if the downgrade took
longer than that, then one of the still-dual-stack apiservers would
see that the first apiserver was no longer dual-stack and delete the
secondary slice.)

Since this is both somewhat unlikely and not that problematic (nothing
should ever look at the secondary EndpointSlice of a single-stack
Service), we can deal with it just by documenting that the
administrator may need to delete the secondary slice by hand after
such a downgrade.

(Alternatively, if the feature remains in alpha for at least 2
releases, then by the time the feature goes beta, it would no longer
be possible to downgrade to an apiserver that didn't support the
feature.)

[kubernetes #101070]: https://github.com/kubernetes/kubernetes/issues/101070

### Version Skew Strategy

The feature should not be affected by the versions of other components
(assuming all of the other components are at least of a version that
fully supports dual-stack).

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: `DualStackAPIServer`
    - Components depending on the feature gate: `kube-apiserver`

* **Does enabling the feature change any default behavior?**

In single-stack clusters, no.

In dual-stack clusters, enabling the feature gate will result in the
apiserver becoming available over a service IP of the cluster's
secondary IP family. Unless the cluster has some "wrong-single-stack"
pods, this will have no real effect. _In theory_ a user could do
something like configure a dual-stack cluster with an IPv4-only
apiserver and IPv6-only pods, to ensure that the pods cannot reach the
apiserver (essentially using the IP family mismatch as a firewall). In
that case, making the apiserver available over IPv6 could be seen as a
security problem.

In environments with an external load balancer or DNS name pointing to
the apiserver, enabling the feature gate may cause the load balancer /
DNS name to become dual-stack. This could, perhaps, be surprising.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**

Yes.

* **What happens if we reenable the feature if it was previously rolled back?**

Nothing unexpected. Even if there is a stale secondary EndpointSlice
lying around, it should be updated at the same time as the Service is
made dual-stack.

* **Are there any tests for feature enablement/disablement?**

No, but there will be tests of Service/EndpointSlice syncing during
skewed enablement (ie, some apiservers have the feature enabled and
others don't).

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?**

Examining the `Endpoints` of the `kubernetes.default` service.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

Whatever SLIs/SLOs already exist for the apiserver.

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**

No

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**

No

### Scalability

* **Will enabling / using this feature result in any new API calls?**

Not directly; using the feature may result in new clients being able
to connect to the apiserver who were not previously able to connect to
it, but this is the entire point of the feature and would be expected.

* **Will enabling / using this feature result in introducing new API types?**

No

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**

No

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**

No

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**

No

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**

No

### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?**

It is an API server feature, so if the API server is unavailable, then
the feature is unavailable.

* **What are other known failure modes?**

At the current time there are no known failure modes.

* **What steps should be taken if SLOs are not being met to determine the problem?**

N/A

## Implementation History

- Initial proposal: 2021-04-13
- Initial proposal merged: 2021-09-06
- Updates: 2022-01-14
