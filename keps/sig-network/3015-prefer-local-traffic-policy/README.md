# KEP-3015: PreferLocal Traffic Policy

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [<code>PreferLocal</code> External Traffic Policy as a More-Reliable &quot;Local&quot;](#-external-traffic-policy-as-a-more-reliable-local)
    - [<code>PreferLocal</code> Internal Traffic Policy as a More-Reliable <code>Local</code>](#-internal-traffic-policy-as-a-more-reliable-)
    - [<code>PreferLocal</code> as a More-Efficient <code>Cluster</code>](#-as-a-more-efficient-)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [External Traffic Policy](#external-traffic-policy)
  - [Internal Traffic Policy](#internal-traffic-policy)
  - [Interaction with Topology](#interaction-with-topology)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
    - [apiserver vs kube-proxy skew](#apiserver-vs-kube-proxy-skew)
    - [kube-proxy vs kube-proxy skew](#kube-proxy-vs-kube-proxy-skew)
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
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

The `Local` policy for `service.spec.externalTrafficPolicy` and
`service.spec.internalTrafficPolicy` allows configuring services so
that they will always deliver to local endpoints, avoiding an
unnecessary hop to another node when the request could just as easily
be processed locally.

However, they both fail entirely if there are no endpoints for the
service on the node. This tradeoff is unacceptable for some use cases.

This KEP adds a `PreferLocal` policy to both `externalTrafficPolicy`
and `internalTrafficPolicy`, which minimizes hops when possible, but
falls back when not.

## Motivation

### Goals

- Allow configuring a service so that connections from outside the
  cluster will be delivered to a local endpoint when possible, and a
  remote endpoint if not.

- Allow configuring a service so that connections from inside the
  cluster will be delivered to a local endpoint when possible, and a
  remote endpoint if not.

### Non-Goals

N/A

## Proposal

### User Stories

#### `PreferLocal` External Traffic Policy as a More-Reliable "Local"

As an application developer, I want incoming service traffic from
outside the cluster to be delivered to my pods as efficiently as
possible (as with `Local` traffic policy), but without interruptions
when endpoints are restarted or moved between nodes.

The [Terminating Endpoints] feature was intended to make
`externalTrafficPolicy: Local` services more robust when endpoints
change, but it doesn't close the race conditions entirely: in
particular, it only works if the time it takes the endpoint pod to
terminate is longer than the amount of time between load balancer
health checks.

[Terminating Endpoints]: ../1669-graceful-termination-local-external-traffic-policy/

#### `PreferLocal` Internal Traffic Policy as a More-Reliable `Local`

As a cluster administrator, I plan to run a DNS pod on each node, and
would like DNS requests from other pods to always go to the local DNS
pod, for efficiency. However, if no local DNS pod is available, DNS
should just go to a remote pod instead so it keeps working.

(This is a modified version of a user story from the [Internal Traffic
Policy KEP]. The original implies (by omission) that is is acceptable
to have DNS lookups fail when there is no local DNS pod.)

[Internal Traffic Policy KEP]: ../2086-service-internal-traffic-policy/

#### `PreferLocal` as a More-Efficient `Cluster`

As an application owner, I would like kube-proxy to worry primarily
about distributing traffic to my service _efficiently_, and let me
take care of ensuring that it is distributed _evenly_. Thus, when it
is possible to deliver service traffic to a local pod, kube-proxy
should do that, even if that means it ends up sending all traffic to
the same pod. (When there are no local pods, kube-proxy would ideally
forward the connection to a nearby pod rather than a far away pod, so
in the fallback case, it should take advantage of Topology Aware
Hints, etc, to ensure efficiently delivery.)

### Risks and Mitigations

## Design Details

### External Traffic Policy

Our [public documentation on externalTrafficPolicy] emphasizes the
fact that `externalTrafficPolicy: Local` preserves the client source
IP more than the fact that it prevents extra hops. A `PreferLocal`
policy would be unable to preserve the source IP in all cases, since
sometimes it will need to bounce the connection to another node.

Since someone using `externalTrafficPolicy: PreferLocal` cannot _rely_
on always having the correct source IP, it would probably be least
confusing to just say that `PreferLocal` services _never_ preserve the
source IP, just like `externalTrafficPolicy: Cluster` services don't.
Thus, the behavior would be:

- `Cluster`: Prioritizes load-spreading. A node receiving a connection
  to a `Cluster` service is equally likely to send it to any of the
  service's endpoints (with no preference given to local endpoints).
  Use this when the clients or intermediaries do not balance
  connections between nodes themselves. (eg, perhaps you are using a
  `NodePort` service, but only advertising a single node IP for
  clients to connect to). Does not preserve source IP.

- `Local`: Prioritizes efficiency and preservation of the source IP. A
  node receiving a connection to a `Local` service will send it to one
  of the service's endpoints on the same node; if there are no
  endpoints on the same node then the packets will be discarded.
  (Hopefully the client will try again and hit a valid node the next
  time.) Since packets never leave the node, the original source IP
  can be preserved. Depends on an upstream load balancer to distribute
  connections evenly between nodes (and to not send connections to
  nodes with no endpoints).

- `PreferLocal`: Prioritizes efficiency and reliability by combining
  aspects of the other two policies. A node receiving a connection to
  a `PreferLocal` service will send it to one of the service's
  endpoints on the same node, if possible; if there are no local
  endpoints then it will send it to a random remote endpoint, as with
  `Cluster`. Like `Local` it depends on an upstream load balancer to
  distribute connections evenly, but it is slightly more reliable in
  the face of changing endpoints, at the expense of not preserving the
  source IP (even when delivering locally).

```
<<[UNRESOLVED renaming ]>>

Should we add a new alias for `Local`, such as `PreserveSourceIP`, and
deprecate the old name?

<<[/UNRESOLVED]>>
```

[public documentation on externalTrafficPolicy]: https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/#preserving-the-client-source-ip

### Internal Traffic Policy

This is easy; source IP is always preserved for internal traffic, so
`PreferLocal` can be implemented in the obvious way with no caveats.

```
<<[UNRESOLVED internal-traffic-policy-local ]>>

Given `PreferLocal`, is there any reason to keep the behavior of
`internalTrafficPolicy: Local` with no fallback? Is there actually a
use case for "either send locally or drop the traffic"?

The KEP gives the use case:

    As a platform owner, I want to create a Service that always
    directs traffic to a logging daemon on the same node. Traffic
    should never bounce to a daemon on another node.

But there's an implied "and if the logging daemon is unavailable, then
logs should be dropped". Are there really cases where you want that?
And the answer can't be "the administrator will just make sure that
the logging daemon is always available", because (a) you can't do
that, and (b) if you could do that then `PreferLocal` would give the
same result as `Local` anyway, since it would never need to fall back.

The strongest argument for keeping distinct `Local` and `PreferLocal`
in the internal traffic case is that it makes it parallel with the
external traffic case (other than the source IP stuff). But I feel
like we really need to document "you almost never want
`internalTrafficPolicy: Local`".

If we rename `externalTrafficPolicy: Local` then there's a stronger
argument for deprecating `internalTrafficPolicy: Local` along with it.

<<[/UNRESOLVED]>>
```

### Interaction with Topology

The [Topology Aware Hints KEP] says "Both ExternalTrafficPolicy and
InternalTrafficPolicy will be given precedence over topology aware
routing." (By which it really means that _`Local`_ traffic policy has
precedence over topology, and topology only applies to `Cluster`
traffic policy.)

When delivering packets locally, `PreferLocal` policy will have to
ignore topology, just like `Local` policy does.

When delivering packets remotely, it makes the most sense to have
`PreferLocal` policy make use of topology. (This is explicitly wanted
in the "`PreferLocal` as Node-Level Topology Routing" use case, and
not unwanted in the other cases.) This also keeps things simple
because it means a `PreferLocal` service always either sends to the
same endpoints it would have sent to if it was `Local`, or to the same
endpoints it would have sent to if it was `Cluster`; there's no need
to add a third endpoint-filtering case.

[Topology Aware Hints KEP]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/2433-topology-aware-hints

### Test Plan

E2E tests will be added similar to existing traffic policy tests, to
cover the new options.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Unit tests for api enablement

#### Beta

- Initial e2e tests completed and enabled.

#### GA

- Time passes, no major objections

### Upgrade / Downgrade Strategy

Existing kube-proxy code only checks for the traffic policy value
`Local`, meaning that any service that isn't `externalTrafficPolicy:
Local` is assumed to be `externalTrafficPolicy: Cluster`, and likewise
for internal traffic policy.

Thus, after downgrading, an old kube-proxy would treat `PreferLocal`
services as `Cluster`. This would preserve all externally-visible
behavior, but would not preserve the efficiency gains of
`PreferLocal`. A user could manually change their `PreferLocal`
services to `Local` after downgrading if they were willing to live
with the problems of `Local`.

### Version Skew Strategy

#### apiserver vs kube-proxy skew

If the apiserver knows about the `PreferLocal` option but kube-proxy
does not, then users can create `PreferLocal` services that kube-proxy
wouldn't know how to handle. But this would just look the same as the
downgrade case; kube-proxy would treat any newly-created `PreferLocal`
services as `Cluster`, which would still have the correct
externally-visible semantics.

#### kube-proxy vs kube-proxy skew

If NewNode's kube-proxy understands `PreferLocal` and OldNode's
kube-proxy does not, then:

  - If a connection arrives at OldNode, then OldNode will treat the
    service as though it was `Cluster`, and forward the connection to
    an endpoint IP somewhere in the cluster, which may be on OldNode
    or NewNode or somewhere else. (The efficiency promises of
    `PreferLocal` are not necessarily met, but the externally-visible
    semantics are correct.)

  - If a connection arrives at NewNode, and NewNode has an endpoint
    for the service, and then NewNode will forward the connection to
    that local endpoint. (`PreferLocal` is handled correctly.)

  - If a connection arrives at NewNode, and NewNode does not have an
    endpoint for the service, the NewNode will forward the connection
    to an endpoint IP somewhere else in the cluster, which may be on
    OldNode or somewhere else. (`PreferLocal` is handled correctly; we
    only fall back to a remote pod because we have to.)

No kube-proxy ever forwards a service connection to another node
without first rewriting it to point to an endpoint IP rather than a
service IP, so it's always the initial node's kube-proxy that
determines whether `PreferLocal` is obeyed or not.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: PreferLocalTrafficPolicy
  - Components depending on the feature gate:
    - apiserver
    - kube-proxy

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes; if the user disables the feature after creating services with
`PreferLocal` policy, those services will still be handled in a
basically reasonable manner, as discussed above under Downgrades.

###### What happens if we reenable the feature if it was previously rolled back?

It starts working again.

###### Are there any tests for feature enablement/disablement?

Not yet.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

An initial rollout cannot fail and won't impact already-running
workloads, because at the time of the initial rollout, there cannot
already be any `PreferLocal` services.

A rollback has reasonable fallback behavior (as with downgrades), and
a re-rollout just updates the behavior of existing `PreferLocal`
services in the expected way.

In particular, no service ever gets switched from `Cluster` or
`PreferLocal` behavior to `Local` behavior after any combination of
rollouts and rollbacks, so no service ever becomes less available /
more racey. They just become more or less efficient.

###### What specific metrics should inform a rollback?

There are no metrics that would inform anyone that the feature was
failing (discussed more below).

If an end user converts a service to `PreferLocal`, and that service
starts failing mysteriously, then that might indicate a bug in the
`PreferLocal` feature, but in that case the user just has to switch
the service back to `Local` or `Cluster`. They don't need to have the
operator roll back the feature enablement.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No removals. TBD on deprecations.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

By checking if any Service has `externalTrafficPolicy: PreferLocal` or
`internalTrafficPolicy: PreferLocal`

###### How can someone using this feature know that it is working for their instance?

When moving an existing `Local` service to `PreferLocal`, they can
know that fallback is occurring because they will no longer see
sporadic client failures when moving/restarting endpoints.

There is no easy way to confirm that the "Local" part of the feature
is working correctly other than by looking at the logs of each
endpoint to confirm that they are receiving the expected connections
and not receiving unexpected connections. Or alternatively, packet
sniffing the cluster network.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- [X] Other

A `PreferLocal` service with endpoints on all nodes where connections
could possibly arrive will have the same "availability" SLOs as a
`Cluster` service, and the same "response time" SLOs as a `Local`
service.

A `PreferLocal` service where some connections may arrive on nodes
with no endpoints will still have the same "availability" SLOs as a
`Cluster` service, but will have "response time" SLOs somewhere
between those of `Local` and `Cluster` services.

| Type                      | Availability              | Response Time                              |
| ------------------------- | ------------------------- | ------------------------------------------ |
| `Cluster`                 | 100%-ish                  | Not As Fast                                |
| `Local`                   | observably less than 100% | Fast                                       |
| `PreferLocal` as fallback | 100%-ish                  | normally "Fast", occasionally "Not As Fast |
| `PreferLocal` as optimiz. | 100%-ish                  | Between "Not As Fast" and "Fast"           |

("`PreferLocal` as fallback" refers to the first two user stories
where most connections get routed locally, but some fall back to
remote. "`PreferLocal` as optimiz." refers to the third use case where
even in the normal case it's expected that some connections will be
routed remotely, but local endpoints are used as an optimization.)

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Other

There are no existing metrics for determining the availability or
response time of individual Services, and this KEP does not add any.
In general, the operator cannot know what response times the user
expects from their Services anyway.

The developer may be able to generate their own metrics based on logs
from the endpoint pods and/or logs generated by the service's clients;
but this involves user-owned code, not kubernetes code, on both ends.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Theoretically there could be metrics about how many Service
connections are refused, etc, but this information is not just lying
around somewhere where we could gather it; adding such a metric would
require a KEP of its own to figure out the implementation details (and
may not be generically implementable).

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

It depends on a service proxy which recognizes the new traffic policy
values. We will update `kube-proxy` ourselves, but network plugins /
kubernetes distributions that ship their own alternative service
proxies will also need to be updated to support the new value before
their users can make use of it.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

`PreferLocal` is more bytes than either `Cluster` or `Local`, so
technically yes.

But No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No change from existing service/proxy behavior.

###### What are other known failure modes?

None known

###### What steps should be taken if SLOs are not being met to determine the problem?

Assuming that the administrator didn't just forget to mark the
services as `PreferLocal`, an SLO failure (eg, a given service is
consistently faster when `Local` than when `PreferLocal`) would have
to indicate a bug in the implementation of the feature, so they should
file a bug.

## Implementation History

- Initial proposal: 2021-10-21
- Updated: 2022-01-15

## Drawbacks

The distinction between `Local` and `PreferLocal` will be somewhat
confusing. This could be mitigated by renaming `Local`.

## Alternatives

None considered.
