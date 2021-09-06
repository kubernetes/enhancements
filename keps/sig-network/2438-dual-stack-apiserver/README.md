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
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [kube-apiserver Command-Line Arguments](#kube-apiserver-command-line-arguments)
  - [Pod environment / client-go](#pod-environment--client-go)
  - [Service and EndpointSlice Reconciling](#service-and-endpointslice-reconciling)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Handling of the &quot;<code>kubernetes</code>&quot; Service](#handling-of-the--service)
    - [Handling of &quot;<code>kubernetes</code>&quot; Endpoints and EndpointSlices](#handling-of--endpoints-and-endpointslices)
    - [Service vs EndpointSlice Sync Issues](#service-vs-endpointslice-sync-issues)
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

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

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

### kube-apiserver Command-Line Arguments

```
<<[UNRESOLVED apiserver-arguments ]>>

Needs to be figured out, but probably:

  - If `--bind-address` or `--advertise-address` is specified as a
    single IP, then the apiserver is single-stack, even if dual
    `--service-cluster-ip-range`s are specified.

  - If `--bind-address` or `--advertise-address` is specified as a
    pair of IPs, and `--service-cluster-ip-range` is specified as a
    pair of CIDRs, then the apiserver is dual-stack, and should fail
    if it cannot bind sockets for both address families.

  - (Incompatible values for `--bind-address`, `--advertise-address`,
    and `--service-cluster-ip-range` will result in kube-apiserver
    exiting with an error.)

  - If neither `--bind-address` nor `--advertise-address` is
    specified, then if `--service-cluster-ip-range` is dual-stack,
    then it will try to bind sockets for both address families. If it
    succeeds, it will proceed as dual-stack; if it fails, it will log
    a warning and proceed as single-stack.

<<[/UNRESOLVED]>>
```

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

If the dual-stack apiserver changes were to take effect in a cluster
all at once when the feature went to beta, then this could cause
Service/EndpointSlice flapping in dual-stack clusters during the
upgrade when the `DualStackAPIServer` feature gate becomes enabled,
because the older apiservers would be trying to enforce a single-stack
`kubernetes.default` service, while the newer apiservers would be
trying to make it be dual-stack.

(Note that the problem is only with `Service` and `EndpointSlice`, not
with `Endpoints` because `Endpoints` is inherently single-stack, so
both old and new apiservers will always agree on which IPs should be
listed there.)

To prevent this, we can make some of the dual-stack apiserver support
code be active even when the feature gate is disabled; the feature
gate will control whether an apiserver in a dual-stack cluster tries
to advertise *itself* as dual-stack, but not whether it allows *other*
apiservers to advertise themselves as dual-stack. Thus:

  - API server does not know about `DualStackAPIServer` (eg,
    Kubernetes 1.21, regardless of single/dual stack configuration):

      - Advertises its own single-stack endpoint IP to etcd

      - Forces the `kubernetes` Service to be single-stack when it
        starts up.

      - Syncs Endpoints and the primary-IP-family EndpointSlice.
        Mistakenly ignores the secondary-IP-family EndpointSlice
        ([kubernetes #101070]) and so could leave behind a stale
        secondary-IP-family EndpointSlice on downgrade from dual-stack
        apiserver.

  - API server knows about `DualStackAPIServer` but either the feature
    is not enabled on this apiserver, or else this apiserver has a
    single-stack configuration:

      - Advertises its own *single-stack* endpoint IP to etcd

      - Forces the `kubernetes` Service to single-stack if there are
        no dual-stack apiservers in etcd. Leaves the Service unchanged
        if it is currently dual-stack and there are also dual-stack
        apiservers in etcd.

      - Syncs Endpoints the primary-IP-family EndpointSlice. If there
        are no dual-stack apiservers in etcd then it will delete any
        secondary-IP-family EndpointSlice.

  - API server knows about `DualStackAPIServer`, the feature gate is
    enabled, and the apiserver has a dual-stack
    `--service-cluster-ip-range`:

      - Advertises its own dual-stack endpoint IPs to etcd

      - Forces the `kubernetes` Service to be dual-stack on startup

      - Syncs Endpoints and dual-stack EndpointSlices

As long as the "`DualStackAPIServer`-aware" code goes in two releases
before the feature gate is enabled by default, then upgrades and
downgrades should proceed smoothly. People who enable the feature gate
manually during the alpha period may encounter problems with
Service/EndpointSlice flapping during upgrades, and may need to do
some manual cleanup on downgrade.

(Note that the middle case covers both "upgrading from
`DualStackAPIServer=false` to `DualStackAPIServer=true`" _and_
"upgrading the cluster from single stack to dual stack (after
`DualStackAPIServer` goes GA)". In both cases, some apiservers will
begin advertising dual-stack endpoints before other apiservers are
aware of the configuration change, but the un-upgraded apiservers will
allow the upgrade to proceed smoothly.)

[kubernetes #101070]: https://github.com/kubernetes/kubernetes/issues/101070

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

- We must be alpha for at least 2 releases, to ensure we don't need to
  support upgrades from pre-Alpha to Beta.

#### Beta -> GA Graduation

- Feedback from users

- 2 examples of distributors/developers/providers making use of
  dual-stack Endpoints

### Upgrade / Downgrade Strategy

As discussed under ["Service and EndpointSlice
Reconciling"](#service-and-endpointslice-reconciling), we will modify
the apiserver behavior when the feature gate is off to make it
cooperate better with apiservers where the feature gate is on (while
still having no change in behavior in a cluster where _all_ apiservers
have the feature gate off). Assuming that the feature remains alpha /
disabled-by-default for at least 2 releases, this should ensure that
all non-early-adopters get smooth upgrades and downgrades.

For early adopters, or if we want to only have the feature be alpha
for a single release, and not have completely smooth behavior for
users with N-2 version skew, there are a few potential problems:

#### Handling of the "`kubernetes`" Service

kube-apiserver attempts to force the "`kubernetes`" Service definition
to match its expectations on startup, but does not attempt to
reconcile it after that point. Old/current apiservers always expect
the Service to be single-stack.

Thus, during an upgrade in which some (new) apiservers are trying to
use the `DualStackAPIServer` feature gate, and some (old) apiservers
predate the new feature, whichever apiserver started last will
determine whether the Service is currently single or dual stack.

In a simple upgrade or downgrade, this would result in the Service
definition changing when the first apiserver was upgraded/downgraded,
and then not changing after that. In a more complicated
upgrade/downgrade scenario, there might be some flapping; eg, the
first master upgrades and changes the Service to dual-stack, but then
the second master reboots before upgrading, causing the old apiserver
to restart and revert the Service definition back to its single-stack
form, before then being upgraded and changing it back to dual-stack
again.

At no point would the Service be advertised as dual-stack when there
were no dual-stack apiservers, so the Service ought to always be
usable; it just may sometimes be single-stack when it _could_ have
been dual-stack.

(The one way that something could go wrong would be if a spurious
apiserver were to start up after the upgrade/downgrade completed. Eg,
after upgrading all apiservers to a dual-stack release, if an old
single-stack-only apiserver were to start up, but then exit, it might
spuriously change the Service back to single-stack before exiting.
It's not clear why this would happen, and at any rate, it would be
fixed as soon as any of the real apiservers restarted again.)

#### Handling of "`kubernetes`" Endpoints and EndpointSlices

For the "`kubernetes`" `Endpoints` (_not_ `EndpointSlice`), there
should be no flapping; both old and new apiservers will agree at all
times about what endpoints are available _in the cluster's primary IP
family_, and thus will always agree on the contents of the
single-stack `Endpoints` resource.

For `EndpointSlice`, current versions of kube-apiserver create a
single `EndpointSlice` object named "`kubernetes`", and assume that
other apiservers will do the same; they don't look at any other
`EndpointSlice` objects when reconciling. To create dual-stack
EndpointSlices, a new apiserver would need to create two objects with
different names. Even if new apiservers use the same name as the old
apiserver does for the primary-IP-family slice, old apiservers would
never see the slice it creates for the secondary IP family. This means
there would be no flapping, but also means that a stale
secondary-IP-family `EndpointSlice` could be left behind after a
downgrade if the new apiserver exited uncleanly. (However, in this
scenario the Service ought to end up as single-stack at the end as
well, so the secondary-IP-Family `EndpointSlice` ought to be ignored
anyway.)

#### Service vs EndpointSlice Sync Issues

Given that the Service is updated only on apiserver restart, but the
EndpointSlice is updated periodically by each apiserver, then whenever
there is a mix of single-stack and dual-stack apiservers, it is
possible to see any combination of service and endpoint
dual-stack-ness:

  - Single-stack Service, single-stack EndpointSlice: in this case,
    the service will work fine; everyone will just ignore the
    secondary IP family.

  - Single-stack Service, dual-stack EndpointSlices: kube-proxy will
    ignore the secondary IP family endpoints, since it has no
    secondary-family ClusterIP to match them up with. If other clients
    try to use the secondary-IP-family EndpointSlice values directly,
    they should succeed, except in the "stale EndpointSlice" case
    discussed above.

  - Dual-stack Service, single-stack EndpointSlice: kube-proxy will
    create working mappings for the primary ClusterIP, and a "broken"
    (ie, `-j REJECT`) mapping for the secondary ClusterIP. Clients
    explicitly attempting to connect to the secondary ClusterIP will
    fail, but clients attempting to connect to
    `kubernetes.default.svc.cluster.local` should succeed, even if
    they try the secondary ClusterIP first, because they should fall
    back to the primary ClusterIP after the secondary one fails.

  - Dual-stack Service, dual-stack EndpointSlices: works fine. Any
    still-single-stack apiservers will not be advertising an IP in the
    secondary family, so the secondary-family EndpointSlice may only
    point to a subset of masters.

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

Nothing unexpected

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

- Initial proposal: 2020-04-13
