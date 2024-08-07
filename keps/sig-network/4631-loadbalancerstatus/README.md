# KEP-4631: LoadBalancer Service Status Improvements

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Expected Behavior When the Cloud Provider Knows It Can't Provide a Working Load Balancer At All](#expected-behavior-when-the-cloud-provider-knows-it-cant-provide-a-working-load-balancer-at-all)
  - [Expected Behavior When the Cloud Provider Knows It Can't <em>Fully</em> Implement a Load Balancer](#expected-behavior-when-the-cloud-provider-knows-it-cant-fully-implement-a-load-balancer)
  - [Expected Behavior When the Cloud Provider <em>Doesn't Know</em> That It Can't Implement a Load Balancer](#expected-behavior-when-the-cloud-provider-doesnt-know-that-it-cant-implement-a-load-balancer)
    - [Option 1: &quot;That Can't Happen&quot;](#option-1-that-cant-happen)
    - [Option 2: Make the Cloud Provider Indicate Statically Which Features It Implements](#option-2-make-the-cloud-provider-indicate-statically-which-features-it-implements)
    - [Option 3: Make the Cloud Provider Indicate Dynamically Which Features It Has Implemented](#option-3-make-the-cloud-provider-indicate-dynamically-which-features-it-has-implemented)
    - [Option 4: Make the Cloud Provider Indicate Dynamically Which Features It <em>Didn't</em> Implement](#option-4-make-the-cloud-provider-indicate-dynamically-which-features-it-didnt-implement)
    - [Option 5: Make the Cloud Provider Respond to an Explicit User-Provided Set of Required Features](#option-5-make-the-cloud-provider-respond-to-an-explicit-user-provided-set-of-required-features)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Risks from losing test coverage](#risks-from-losing-test-coverage)
    - [Risks to users](#risks-to-users)
- [Design Details](#design-details)
  - [<code>Service.Status.Conditions</code>](#servicestatusconditions)
    - [The <code>LoadBalancerServing</code> Condition](#the-loadbalancerserving-condition)
    - [The <code>LoadBalancerProvisioning</code> Condition](#the-loadbalancerprovisioning-condition)
    - [Terminating Condition](#terminating-condition)
    - [Service Lifecycle and Interaction of the Conditions](#service-lifecycle-and-interaction-of-the-conditions)
    - [Well-Known <code>Reason</code> Values for Load Balancer Conditions](#well-known-reason-values-for-load-balancer-conditions)
  - [FIXME Unsupported Features](#fixme-unsupported-features)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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

While updating the e2e load balancer tests after the final [removal of
in-tree cloud providers], we have run into three problems:

  1. The tests have hard-coded timeouts (that sometimes differ per
     cloud provider) for deciding how long to wait for the cloud
     provider to update the service. It would make much more sense for
     the cloud provider to just provide information about its status
     on the Service object, so the tests could just monitor that.

  2. The tests recognize that not all cloud providers can implement
     all load balancer features, but in the past this was handled by
     hard-coding the information into the individual tests. (e.g.,
     `e2eskipper.SkipUnlessProviderIs("gce", "gke", "aws")`) These
     skip rules no longer work in the providerless tree, and this
     approach doesn't scale anyway. OTOH, we don't want to have to
     provide a separate `Feature:` tag for each load balancer
     subfeature, or have each cloud provider have to maintain their
     own set of `-ginkgo.skip` rules. It would be better if the e2e
     tests themselves could just figure out, somehow, whether they
     were running under a cloud provider that intends to implement the
     feature they are testing, or a cloud provider that doesn't.

  3. In some cases, because the existing tests were only run on
     certain clouds, it is not clear what the expected semantics are
     on other clouds. For example, since `IPMode: Proxy` load
     balancers can't preserve the client source IP in the way that
     `ExternalTrafficPolicy: Local` expects, should they refuse to
     provision a load balancer at all, or should they provision a load
     balancer that fails to preserve the source IP?

This KEP proposes new additions to `service.Status.LoadBalancer` and
`service.Status.Conditions` to allow cloud providers to better
communicate the status of load balancer support and provisioning, and
new guidelines on how cloud providers should handle load balancers for
services that they cannot fully support.

(Note: although I repeatedly refer to service load balancers as being
implemented by "cloud providers" here, everything also applies to
non-cloud-provider components load balancers via the
[`LoadBalancerClass`] field.)

[removal of in-tree cloud providers]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-cloud-provider/2395-removing-in-tree-cloud-providers/
[`LoadBalancerClass`]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-cloud-provider/1959-service-lb-class-field/

## Motivation

### Goals

- Allow cloud providers to indicate that they are working on
  provisioning load balancer infrastructure, so that
  users/operators/tests can distinguish the case of "it is taking a
  while for the cloud to provision the load balancer" from "the cloud
  has failed to provision a load balancer" and "there is no cloud
  provider so load balancers don't work".

- Allow cloud providers to indicate explicitly when they are unable to
  provision a particular `LoadBalancer` service, and why.

- Allow cloud providers to indicate when they have provided an
  "imperfect" load balancer that the user may or may not consider to
  be "good enough" (e.g., providing a single-stack load balancer for a
  dual-stack Service).

- Allow users and e2e tests to either prevent or reliably recognize
  the case where the cloud provider has provisioned a load balancer
  without support for a particular feature the user/test needs (even
  when that is a feature the cloud provider doesn't know about).

- Add appropriate helper functions to `k8s.io/cloud-provider`, update
  at least [cloud-provider-gcp] and [cloud-provider-kind] to fully
  support the new status reporting functionality, and encourage other
  providers to follow.

- Update the e2e load balancer tests to use the new functionality to
  determine which tests to skip, so that any cloud provider that
  conforms to the recommendations in this KEP will be able to run all
  `[Feature:LoadBalancer]` tests, now and in the future, without
  needing to have any per-provider exceptions in the test code, or
  having to manually tell the test runner to skip any of the tests.

  To avoid ambiguous test results (and wasted resources), each test
  will have at most one auto-skip, at the earliest possible point
  where the presence or absence of necessary functionality becomes
  apparent. (Some of the existing load balancer e2e tests are very
  long and "kitchen sink"-y, but each one fundamentally only tests a
  single feature (e.g., UDP, or source ranges support) so they could
  be made to bail out early on unsupported clouds.)

### Non-Goals

- Requiring all cloud providers to implement all existing
  load-balancer-related features.

[cloud-provider-gcp]: https://github.com/kubernetes/cloud-provider-gcp/
[cloud-provider-kind]: https://github.com/kubernetes-sigs/cloud-provider-kind/

## Proposal

It seems uncontroversial that the cloud provider should provide
feedback to the user when it is in the process of provisioning a load
balancer, or when it has conclusively failed to do so.

```
<<[UNRESOLVED split the KEP? ]>>

Should we split the presumably-uncontroversial
`LoadBalancerProvisioning` and `LoadBalancerServing` Conditions out
into a separate KEP?

<<[/UNRESOLVED]>>
```

That leaves three problems to solve.

### Expected Behavior When the Cloud Provider Knows It Can't Provide a Working Load Balancer At All

A cloud provider might find itself unable to provide a load balancer
for a Service *at all* because:

  - The Service uses exclusively UDP (or SCTP) ports, but the
    cloud/load balancer only supports TCP, so a load balancer would
    be unable to route any traffic to the Service.

  - The Service is single-stack IPv6 in a cloud (or cluster) that only
    supports IPv4 load balancers, or vice versa, so a load balancer
    would be unable to route any traffic to the Service.

  - The Service has `AllocateLoadBalancerNodePorts: false`, but the
    cloud only supports NodePort-based load balancing.

  - The Service is `ExternalTrafficPolicy: Local` but the cloud does
    not implement the `HealthCheckNodePort` aspect of local traffic
    policy (presumably because it cannot implement client source IP
    preservation anyway), so if it created a load balancer, it would
    end up randomly misrouting traffic to nodes that won't accept it.

Currently there is no way for a cloud provider to report failure back
to the user, and existing implementations vary between "do nothing
(except log an error)" and "provide a load balancer that won't
actually work".

We should allow cloud providers to indicate to the user that they
can't (or won't) provide a load balancer for a given service, and
require load balancers to do that in cases where they know they can't
provide at least a minimally-functional load balancer.

(In theory, a cloud provider could pick a non-default load balancer
type so as to be able to support the features that the user requested.
Given that there is not currently any information about load balancer
types in `ServiceSpec`, this KEP does not add any to `ServiceStatus`.)

```
<<[UNRESOLVED "broken" condition? ]>>

Should there be a condition to explicitly indicate
brokenness/unsupportedness?

In particular, consider the case where an older version of a cloud
provider has already incorrectly provisioned a load balancer, where
the newer version would choose to reject it instead. Clearly it
shouldn't retroactively deprovision the load balancer on upgrade.
Should it be able to indicate that the service is
`LoadBalancerServing=True, LoadBalancerSupported=False`?

Or perhaps this is a use case for `LoadBalancerServing=Unknown`? ("I
don't know whether you would count this as 'serving' or not.")

<<[/UNRESOLVED]>>
```

### Expected Behavior When the Cloud Provider Knows It Can't *Fully* Implement a Load Balancer

There are also situations where a cloud provider may be able to
partially or imperfectly provide load balancing for a Service:

  - The Service uses both TCP and UDP ports, but the cloud/load
    balancer only supports single-protocol load balancers, so it would
    only be able to serve one of the two protocols.

  - The Service is dual-stack, but the cloud (or cluster) only
    supports single-stack load balancers, so it would only be able to
    serve clients of one IP family.

  - The Service is `ExternalTrafficPolicy: Local` but the cloud cannot
    implement client source IP preservation, though it *does*
    implement `HealthCheckNodePort` tracking anyway, so it can still
    provide "efficient"/"one-hop" load balancing *without* source IP
    preservation (which might actually be all the user wanted anyway).

  - The Service uses `LoadBalancerSourceRanges` but the cloud does not
    implement that feature.

  - The Service uses `SessionAffinityConfig` but the cloud does not
    implement that feature.

(The "mixed TCP and UDP" situation was discussed in [KEP-1435], which
provided a Service Condition (`LoadBalancerPortsError`) that could be
used to indicate non-support for this functionality. While the KEP
required that "All of the major clouds support \[mixed protocols] or
indicate non-support properly" as a condition for graduating to
Beta, it seems that only `cloud-provider-gcp` was ever updated to set
this condition, and that was several months after the feature went
GA.)

In the past, in general, due to the inability to negotiate load
balancer functionality or report errors, most cloud providers have
erred on the side of providing *some* functionality in these cases
rather than just refusing to provision a load balancer.

If we were going to provide official guidance, then it doesn't seem
like a one-size-fits-all rule will work; having a cloud provider
ignore `IPFamilyPolicy` may actually be what the user wants, but
having it ignore `LoadBalancerSourceRanges` is probably not.

What seems most important is that *if* the cloud provider does choose
to provide an "imperfect" load balancer, it must be able to explain
this to the user.

[KEP-1435]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/1435-mixed-protocol-lb/

### Expected Behavior When the Cloud Provider *Doesn't Know* That It Can't Implement a Load Balancer

The final (potential) problem is when a Service uses a feature that is
newer than the cloud provider codebase, resulting in a situation where
the cloud provider cannot implement a proper load balancer for the
Service, but does not realize that this is the case. (As a concrete
example, at the time of writing, [KEP-4444]'s `TrafficDistribution`
field is available as an Alpha API, but no cloud providers are yet
aware of it, and thus they will likely not distribute traffic
correctly for Services that are using this feature.)

[KEP-4444] https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/4444-service-routing-preference/

```
<<[UNRESOLVED the unknown unknowns ]>>

Pick one of the options below (or another?). Current TL;DR is:

Option 1: Doesn't fully solve the e2e testing problem. Possibly unrealistic.
Option 2: "Un-Kubernetes-Like"
Option 3: Maybe verbose and annoying but no one has argued against it yet?
Option 4: Sounds clever but probably isn't.
Option 5: Inconvenient for user? Makes skew more of a problem.

@thockin suggested that 5 is inconvenient ("So the user has to specify
things in 2 different places? Blech") but that's only in the case
where the user wants the load balancer to be rejected if the cloud
can't provide all the requested functionality, and in that case,
options 3 and 4 are even more inconvenient, because then the user has
to first create the LB and then after it's provisioned check to see if
the cloud provided the right features. So if we don't like 5 because
it's inconvenient, then the options are 1 or 2.)

(@thockin also said that this begs the question of "why we don't do
this for EVERYTHING" and that may be worth thinking about; do we think
we'd eventually want a similar thing for NetworkPolicy or service
proxy implementations, and if so, what sort of API would we want
there, so we can choose a similar API here.)

Options 3 and 4 also make the e2e tests slower, because they require
us to provision a load balancer and then find out belatedly that it
doesn't do what we want. (We can't cache supported-ness results
between e2e tests either, since the provider might theoretically
switch between `Proxy` and `VIP` style LBs depending on the details of
the Service.) Options 1 and 2 let us know ahead of time whether the
cloud supports a given feature, so we'd be able to skip the
unimplemented tests without having to create a load balancer. Option 5
makes the cloud fail before it actually tries to allocate a load
balancer, so it's also fast.

<<[/UNRESOLVED]>>
```

#### Option 1: "That Can't Happen"

One possibility would be to declare by fiat that this not a problem.
In particular, we could say that all cloud providers are required to
know about all features in the same release that they are added as
Alpha, and while they do not need to necessarily *implement* those
features immediately, they are at least responsible for indicating to
the user that they don't implement them, when needed (as per the
previous two sections).

(Requiring them to only do this for Beta or GA APIs might be more
implementor-friendly but it's not clear how well that will work with
version skew during upgrades?)

However, while this may solve "real world" usage problems, it doesn't
really solve the e2e testing problem; there will necessarily be at
least a small gap between when the API for a new feature is first
merged, and when _all_ cloud providers have been updated to know about
it, so we would not reliably be able to enable `[Feature:LoadBalancer]`
in alpha-feature-testing jobs.

#### Option 2: Make the Cloud Provider Indicate Statically Which Features It Implements

The cloud provider could publish, on some API object, a list of the
features it currently implements, which users, operators, and e2e
tests could use to decide what sorts of load balancers to create. (In
the case of clouds that support multiple load balancer types, they
could perhaps even provide separate feature lists for each load
balancer type...)

Currently, Kubernetes networking does not have anything like this, and
so this feels "un-Kubernetes-like", though at the same time, it's not
like the current Kubernetes networking configuration situation is
really great, and there has been some discussion of trying to provide
more explicit and well-defined cluster networking configuration in the
past (e.g., "[Kubernetes Network Configuration]")

[Kubernetes Network Configuration]: https://docs.google.com/document/d/1Dx7Qu5rHGaqoWue-JmlwYO9g_kgOaQzwaeggUsLooKo/edit

#### Option 3: Make the Cloud Provider Indicate Dynamically Which Features It Has Implemented

Rather than indicating ahead of time what features it implements, a
cloud provider could indicate in the `LoadBalancerStatus` of a Service
which features it had implemented *on that load balancer*.

For example, an e2e test for the traffic distribution feature might
create a load balancer service, and then expect to see:

```yaml
status:
  loadBalancer:
    ingress:
      - ip: 1.2.3.4
    features:
      - IPv4
      - TCP
      - TrafficDistribution
```

(indicating that the cloud provider has provisioned a load balancer
that serves TCP over IPv4 and obeys the `TrafficDistribution` field).
If instead it sees just `features: ["IPv4", "TCP"]`, then it knows
that the cloud provider does not currently implement traffic
distribution, and so it should skip the test.

#### Option 4: Make the Cloud Provider Indicate Dynamically Which Features It *Didn't* Implement

*In theory*, a cloud provider could notice that it has been asked to
implement features that it doesn't know about by detecting that the
`Service` object it received from the API server contains fields that
aren't present in the version of `k8s.io/api` that it was compiled
against. In that case, the shared cloud provider code could
*automatically* return a list of fields that the cloud provider
doesn't know about (augmented with a list from the provider-specific
code of fields that it *does* know about but is ignoring anyway).
Thus, in the unimplemented-KEP-4444 case, the resulting Service status
might look something like:

```yaml
status:
  loadBalancer:
    ingress:
      - ip: 1.2.3.4
    unimplementedFeatures:
      - trafficDistribution
```

While at first glance, this is more user-friendly since it is explicit
about what wasn't implemented, it runs into problems given its
auto-generated-ness; since the provider doesn't know *anything* about
these fields, it may report that it has failed to implement them even
when that failure has no actual consequences (e.g., claiming to have
not implemented `allocateLoadBalancerNodePorts` for a Service which is
`allocateLoadBalancerNodePorts: true`). This in turn makes it *less*
user-friendly (though this is not a problem for automated use from e2e
tests, since the tests would know when it does and doesn't matter that
a particular feature was unimplemented).

#### Option 5: Make the Cloud Provider Respond to an Explicit User-Provided Set of Required Features

Another alternative is that if the user actually wants to know, for
sure, that the load balancer implements a particular feature, they can
request that explicitly:

```
# The load balancer must be able to support mixed TCP and UDP, and
# must support KEP-4444
requiredFeatures:
  - TCP
  - UDP
  - TrafficDistribution
```

Then the cloud provider could scan the list to see if there are any
features it either doesn't implement or simply doesn't recognize, and
refuse to create a load balancer if so.

In that case, the e2e tests can just always require the specific
features that they are testing, and skip themselves if the cloud
provider returns a `LoadBalancerServing` with the value `False` and a
`Reason` of `UnsupportedRequiredFeature` or the like.

(This feature would also be useful in other cases: for example, if the
user wants a load balancer using `LoadBalancerSourceRanges` and wants
to ensure that the cloud provider doesn't create a load balancer if it
can't correctly implement the source range filtering.)

(Unlike all of the other options, this one potentially has [version
skew](#version-skew-strategy) problems, since this field is not purely
informational.)

### Risks and Mitigations

#### Risks from losing test coverage

Up to 1.30, Kubernetes CI included GCE-based testing of the Service
load balancer APIs. This has regressed somewhat due to the removal of
provider-specific code from k/k, and we need to make sure we
un-regress before 1.31 ships. This is being done partially by getting
back to full coverage on GCE, and partially by adding new testing
based on `cloud-provider-kind`, to allow more generic testing of the
API surface and of the kube-proxy end of load balancer implementation.

(This un-regressing will happen with or without this KEP, but without
the new APIs in this KEP, it will be hacky, and it won't work well for
_other_ cloud providers that want to run the e2e test suite.)

#### Risks to users

Using the new features should not introduce any new risks.

They should hopefully _improve_ security for users. Currently, you
sometimes get load balancers that aren't what you wanted (e.g.,
failing to implement source ranges or session affinity). This KEP aims
to prevent that (or at least make it more visible when it happens).

(If a cloud provider implemented the API incorrectly and claimed
support for features that it didn't actually support, then that could
lead to users being more confident when deploying broken load
balancers than they are now, but the cloud provider would fail the e2e
tests in that case, so this shouldn't happen.)

## Design Details

### `Service.Status.Conditions`

The problem of indicating provisioning progress and provisioning
failure can easily be solved with conditions on the Service.

```
<<[UNRESOLVED sync condition names with Gateway? ]>>

@thockin suggested we should make sure to keep our condition names in
sync with Gateway, but it seems like Gateway doesn't currently have
any conditions that overlap with the conditions suggested here?

<<[/UNRESOLVED]>>
```

#### The `LoadBalancerServing` Condition

The new `LoadBalancerServing` condition, when `True`, indicates that
the service's load balancer is handling connections via the IP(s) or
hostname(s) indicated in `.Status.LoadBalancer`.

The cloud provider should set a `LoadBalancerServing` condition
immediately upon first becoming aware of a `Type: LoadBalancer`
Service. At all times, the condition should be `True` if the load
balancer is handling connections, and `False` if it is not. (There are
no defined semantics for an `Unknown` value of this condition.)

(This means that at construction time, the `LoadBalancerServing`
condition is redundant with the `.Status.LoadBalancer` field; it
should switch from `False` to `True` at exactly the same time as the
`.Status.LoadBalancer` field is filled in. However, if the Service is
updated later, the cloud provider may temporarily switch it to
`LoadBalancerServing=False` while reprovisioning the Service, and it
would not generally clear `.Status.LoadBalancer` during this time.)

Any time the load balancer behavior changes in an externally-visible
way, the cloud provider must update at least the `LastTransitionTime`
of the `LoadBalancerServing` condition. (For example, if the user
changes the service's `LoadBalancerSourceRanges`, and the cloud
provider updates the load balancer for this, it must update the
`LoadBalancerServing` condition's `LastTransitionTime`, even if the
value of `LoadBalancerServing` did not toggle from `True` to `False`
and back to `True`.)

#### The `LoadBalancerProvisioning` Condition

The new `LoadBalancerProvisioning` condition indicates that the cloud
provider is currently performing a time-consuming
resource-provisioning step for the load balancer. When
`LoadBalancerProvisioning` is `True`, the load balancer is either not
handling connections, or else is still handling connections according
to the semantics of a previous state of the Service (with the value of
the `LoadBalancerServing` condition distinguishing these two cases).

The cloud provider should set a `LoadBalancerProvisioning` condition
immediately upon first becoming aware of a `Type: LoadBalancer`
Service. At all times, the condition should be `True` if the cloud
provider is waiting for the completion of an operation to bring the
load balancer's behavior in line with the current state of the Service
object, and `False` if it not. (There are no defined semantics for an
`Unknown` value of this condition.)

Any time any load-balancer-related field (or provider-specific
annotation) of the Service object changes, the cloud provider must
update at least the `LastTransitionTime` of the
`LoadBalancerProvisioning` condition, to indicate that it has seen the
update, even if the value of `LoadBalancerProvisioning` remains
`False`.

#### Terminating Condition

```
<<[UNRESOLVED termination ]>>

Do we need/want a `LoadBalancerTerminating` condition? And/or, should
the cloud provider set `LoadBalancerProvisioning` when it is
*de*-provisioning a load balancer?

There is possibly no need for this when deleting a Service, but it may
help in the case of converting a Service from `Type: LoadBalancer` to
`Type: ClusterIP` (which is tested by the e2es).

<<[/UNRESOLVED]>>
```

#### Service Lifecycle and Interaction of the Conditions

When a load balancer Service is first created (or a non-load-balancer
Service is converted to `Type: LoadBalancer`), one of three things
should immediately happen:

  1. The cloud provider sets `LoadBalancerProvisioning=False`,
     `LoadBalancerServing=False`, with appropriate `Reason` values
     (discussed below), indicating that it cannot or will not
     provision a load balancer for the service.

  2. The cloud provider sets `LoadBalancerProvisioning=False`,
     `LoadBalancerServing=True`, (and fills in `.Status.LoadBalancer`)
     indicating that the load balancer for the service is already
     available.

  3. The cloud provider sets `LoadBalancerProvisioning=True`,
     `LoadBalancerServing=False`, with appropriate `Reason` values,
     indicating that it is provisioning the load balancer.

In the third case, this would later be followed up by the cloud
provider setting `LoadBalancerProvisioning` to `False` and
`LoadBalancerServing` to either `True` or `False`, depending on
whether the provisioning succeeded or failed.

If the `LoadBalancerServing` and `LoadBalancerProvisioning` conditions
do not appear on the Service within a reasonable amount of time after
the Service is created (or modified to be `Type: LoadBalancer`), the
user should assume that no cloud provider is running, and that
`LoadBalancer` Services are not supported.

(Of course, in the short term, users can't actually assume that,
because not all cloud providers will support the new conditions.
However, since supporting these conditions will be a prereq for
running the `[Feature:LoadBalancer]` e2e tests, we assume that
eventually all cloud providers will support them.)

When the user updates load-balancer-relevant fields in a Service that
already has `LoadBalancerProvisioning` and `LoadBalancerServing`
conditions, the cloud provider should update the conditions in mostly
the same way as above, except that if the load balancer is able to
continue handling connections according to the original Service
semantics while it is being reprovisioned, the cloud provider should
leave the `LoadBalancerServing` condition set to `True`.

Thus, the four possible combinations of the two conditions are:

- `Provisioning=False`, `Serving=False`: The load balancer is in an
  error state; it is not handling connections and not expected to.

- `Provisioning=True`, `Serving=False`: The load balancer is being
  provisioned or reprovisioned, and is not handling connections while
  it does so.

- `Provisioning=True`, `Serving=True`: The load balancer is being
  reprovisioned, but is continuing to handle traffic according to an
  older state of the Service during the reprovisioning step.

- `Provisioning=False`, `Serving=True`: The load balancer is
  functioning normally and handling connections according to the
  last-seen update to the Service.

#### Well-Known `Reason` Values for Load Balancer Conditions

When provisioning or reprovisioning a load balancer, the
`LoadBalancerProvisioning` condition should have the value `True` with
`Reason=InProgress`. If `LoadBalancerProvisioning` completes
successfully, it should be changed to `False` with `Reason=Complete`.

When the load balancer is not serving because it is being provisioned
or reprovisioned, the `LoadBalancerServing` condition should have the
value `False` with `Reason=Provisioning`. When the load balancer is
serving (regardless of whether it is also simultaneously
reprovisioning) the `LoadBalancerServing` condition should be `True`
with `Reason=Serving`.

If the cloud provider has not provisioned a load balancer because it
cannot acceptably implement the semantics of the service, it should
set both `LoadBalancerProvisioning` and `LoadBalancerServing` to
`False`, with `Reason=Unsupported` and a `Message` indicating why it
is not supported. (See also the `.Features` field below.)

If the cloud provider has not provisioned a load balancer because of
problems with the underlying cloud infrastructure (e.g., no more load
balancer IPs are available), it should set both
`LoadBalancerProvisioning` and `LoadBalancerServing` to `False`, with
`Reason=Infrastructure` and a `Message` indicating what has gone
wrong.

### FIXME Unsupported Features

```
<<[UNRESOLVED unsupported features ]>>

Figure out an option for handling unsupported features

<<[/UNRESOLVED]>>
```

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

None / already complete.

##### Unit tests

There will be a few unit tests for validation/defaulting of the new
API, but this feature can really only be tested via e2e tests, since
it depends on external components.

##### Integration tests

As above, this feature can really only be tested via e2e tests, since
it depends on external components.

##### e2e tests

Most of the new functionality exists to _support_ the e2e tests, so it
is difficult to also test the functionality itself.

(Perhaps we should add some support to cloud-provider-kind for
intentionally failing at various load balancer tasks, based on
annotations or something?)

In order to not regress testing coverage while this KEP is still
Alpha, we will need to fork the `[Feature:LoadBalancer]` tests into
two versions: the existing tests that only really support
cloud-provider-gcp and cloud-provider-kind, and new tests that use the
new status/conditions to work with any load balancer that supports the
new functionality.

In the new tests:

  1. Load Balancer provisioning will assume support for the new
     conditions, and will expect a condition to be set on the Service
     within 30 seconds. (That matches
     `e2eservice.KubeProxyEndpointLagTimeout`, the maximum lag between
     updating a Pod or Service and seeing those updates reflected in
     EndpointSlices, which involves the same "e2e.test to apiserver to
     controller to apiserver to e2e.test" path.) Once the condition is
     set we will still use the existing
     `e2eservice.GetServiceLoadBalancerCreateTimeout()` value (based
     on cluster size) to wait for provisioning to complete.

  2. Once the conditions indicate that the load balancer is serving,
     `IP` based load balancers will be expected to work immediately
     (and the test will fail if they do not). `Hostname` based load
     balancers will be allowed some lag time for DNS propagation, but
     will be expected to work immediately once the DNS name resolves.

  3. Based on whatever decision is made about feature detection, the
     tests will skip tests that aren't expected to pass on the current
     cloud, but will fail tests that were expected to pass but didn't.

Specifically:

- [`LoadBalancers should be able to change the type and ports of a TCP
  service`](https://github.com/kubernetes/kubernetes/blob/67e0c519e3/test/e2e/network/loadbalancer.go#L133).
  This is an annoyingly large test, but everything it does should be
  supported by all cloud providers that support load balancers at all,
  so it will be strictly pass/fail with no skips.

- [`LoadBalancers should be able to change the type and ports of a UDP
  service`](https://github.com/kubernetes/kubernetes/blob/67e0c519e3/test/e2e/network/loadbalancer.go#L271).
  As above, but will be skipped if the cloud does not support UDP load
  balancers.

    - As currently written, this test (and the previous one) do
      "create a `ClusterIP` service, test it, convert to `NodePort`,
      test it, convert to `LoadBalancer`, test it, convert back to
      `ClusterIP`, test it. We don't actually need to test UDP
      ClusterIP/NodePort support (that should already be tested
      elsewhere), but we do need to test "the cloud can enable/disable
      LoadBalancing on an existing service". So we should rewrite this
      to "create a `LoadBalancer` service, maybe skip, test it,
      convert to `ClusterIP`, test it, convert back to `LoadBalancer`,
      test it. (Though this means we'll have to wait for LB
      provisioning twice during the test run.)

- [`LoadBalancers should only allow access from service loadbalancer
  source ranges`](https://github.com/kubernetes/kubernetes/blob/67e0c519e3/test/e2e/network/loadbalancer.go#L422).
  Needs to be skipped if the cloud does not support source ranges
  _and_ is `Proxy` style. (`VIP` style LBs can fall back to using
  kube-proxy's source ranges support, so they essentially
  automatically support this.)

- [`LoadBalancers should have session affinity work for LoadBalancer
  service with ( Local | Cluster ) traffic
  policy`](https://github.com/kubernetes/kubernetes/blob/67e0c519e3/test/e2e/network/loadbalancer.go#L504).
  Needs to be skipped if the cloud does not support session affinity.
  The "Local" variant needs to be skipped if the cloud does not
  support Local traffic policy (if we allow that as an implementation
  choice).

- [`LoadBalancers should be able to switch session affinity for
  LoadBalancer service with ( Local | Cluster ) traffic
  policy`](https://github.com/kubernetes/kubernetes/blob/67e0c519e3/test/e2e/network/loadbalancer.go#L514).
  As above. The test should also be flipped around, to test "has
  affinity" first, then maybe skip, then test "doesn't have affinity".

- [`LoadBalancers should handle load balancer cleanup finalizer for
  service`](https://github.com/kubernetes/kubernetes/blob/67e0c519e3/test/e2e/network/loadbalancer.go#L549).
  This should pass on all clouds that support load balancers.

- [`LoadBalancers should be able to create LoadBalancer Service
  without NodePort and change it`](https://github.com/kubernetes/kubernetes/blob/67e0c519e3/test/e2e/network/loadbalancer.go#L581).
  Should be skipped for `Proxy` mode load balancers (or for `VIP` mode
  load balancers that don't yet support
  `AllocateLoadBalancerNodePorts: false`).

- [`LoadBalancers should be able to preserve UDP traffic when server
  pod cycles for a LoadBalancer service ( on different nodes | on the
  same node)`](https://github.com/kubernetes/kubernetes/blob/67e0c519e3/test/e2e/network/loadbalancer.go#L646).
  Should be skipped for clouds that don't support UDP load balancers.

- [`LoadBalancers should not have connectivity disruption during
  rolling update with externalTrafficPolicy=( Cluster | Local )`](https://github.com/kubernetes/kubernetes/blob/67e0c519e3/test/e2e/network/loadbalancer.go#L910).
  "Local" variant needs to be skipped if the cloud does not support
  Local traffic policy (if we allow that as an implementation choice).

- [`LoadBalancers ExternalTrafficPolicy: Local should work for
  type=LoadBalancer`](https://github.com/kubernetes/kubernetes/blob/67e0c519e3/test/e2e/network/loadbalancer.go#L956).
  Should be skipped if the cloud does not support the source IP
  preservation aspect of Local traffic policy.

- [`LoadBalancers ExternalTrafficPolicy: Local should work for
  type=NodePort`](https://github.com/kubernetes/kubernetes/blob/67e0c519e3/test/e2e/network/loadbalancer.go#L1013).
  This isn't actually a load balancer test, and should be moved out of
  `[Feature:LoadBalancers]`.

- [`LoadBalancers ExternalTrafficPolicy: Local should only target
  nodes with endpoints`](https://github.com/kubernetes/kubernetes/blob/67e0c519e3/test/e2e/network/loadbalancer.go#L1045).
  Should be skipped if the cloud does not support Local traffic policy
  at all (if we allow that as an implementation choice).

- [`LoadBalancers ExternalTrafficPolicy: Local should work from pods`](https://github.com/kubernetes/kubernetes/blob/67e0c519e3/test/e2e/network/loadbalancer.go#L1121).
  Should be skipped if the cloud does not support the source IP
  preservation aspect of Local traffic policy (and the load balancers
  are `Proxy` mode, since for `VIP` mode, pod-to-LB traffic won't
  hit the actual load balancer).

- [`LoadBalancers ExternalTrafficPolicy: Local should handle updates
  to ExternalTrafficPolicy field`](https://github.com/kubernetes/kubernetes/blob/67e0c519e3/test/e2e/network/loadbalancer.go#L1179).
  There are several problems with this test and it needs to be split
  into two tests anyway:

    - "HealthCheckNodePorts for LoadBalancer Services are implemented
      correctly". This requires creating a `type: LoadBalancer`
      Service (because only LoadBalancer Services have
      HealthCheckNodePorts), but it _doesn't_ actually require cloud
      load balancer support (because we aren't testing the LBs
      themselves, we're testing the service proxy functionality that
      gets _consumed_ by the cloud provider), so it can be moved out
      of `[Feature:LoadBalancers]`.

    - "LoadBalancers handle updates to ExternalTrafficPolicy". Should
      be skipped if the cloud does not support Local traffic policy at
      all (if we allow that as an implementation choice). Currently
      the test also tests that source IPs are preserved with Local
      traffic policy and _not_ preserved with Cluster traffic policy,
      but the former is redundant with other tests, and the latter is
      not actually required (e.g., Cilium sometimes preserves client
      IP even for Cluster traffic policy services). So we can just
      remove all the source IP bits from this test.

### Graduation Criteria

#### Alpha

- Condition handling implemented in `k8s.io/cloud-provider`

- cloud-provider-kind and cloud-provider-gcp set conditions/status
  appropriately when the feature gate is enabled.

- New version of e2e load balancer tests that use the
  conditions/status, which pass on GCE and kind when the feature gate
  is enabled.

#### Beta

- At least two additional cloud providers (one of which must be
  cloud-provider-aws or cloud-provider-azure) have been updated to use
  the new functionality, and pass/skip the e2e tests as expected for
  those platforms.

- As discussed in "[Version Skew Strategy](#version-skew-strategy)"
  below, since the new API fields are informational-only, we do not
  actually need to wait for "skew-safety" between Alpha and Beta.

#### GA

- At least two additional cloud providers (one of which must be a
  non-public-cloud provider like cloud-provider-vsphere or
  cloud-provider-openstack) have been updated to use the new
  functionality, and pass/skip the e2e tests as expected for those
  platforms.

- Allowing time for feedback...?

### Upgrade / Downgrade Strategy

When upgrading/enabling the feature gate, cloud providers should
update existing already-provisioned load balancers with appropriate
status information.

When downgrading/disabling, the cloud provider does not need to do
anything.

### Version Skew Strategy

Unless we go with the `requiredFeatures` ([Option
5](#option-5-make-the-cloud-provider-respond-to-an-explicit-user-provided-set-of-required-features))
implementation of feature detection, then the new fields are purely
informational, and are write-only from the perspective of the
cloud-provider.

If the cloud provider is newer than the apiserver, and tries to write
to status fields that don't exist, those writes will simply be
ignored, and the result will usually be the same as with using an old
cloud provider. One catch is that if the cloud provider changes its
behavior to take advantage of the new possibility of error reporting
(e.g., failing to provision load balancers that request session
affinity), then it might end up exhibiting that changed behavior, but
failing to actually report the error to the user correctly. This might
be surprising to the user, but would not really be an API break. (The
cloud provider just changed from one buggy behavior ("silently giving
the user a broken load balancer"), to another ("silently failing to
give the user any load balancer").

If we use `requiredFeatures` (or any similar approach where there is a
new field written by the client), then version skew could result in a
situation where the client thinks it has required a particular
feature, but the cloud provider does not know that it required that.
In particular, in the case where the client and the cloud provider are
both "new" and the apiserver is "old", the cloud provider would
succeed in setting the `LoadBalancerServing` condition (since that is
a pre-existing API field), leaving the client to believe that the
cloud provider had asserted full support for the feature. This could
be mitigated by requiring the cloud provider to set an
`implementedFeatures` field matching the client's `requiredFeatures`
field, though that might be gratuitous if it serves no purpose other
than preventing an obscure problem during version skew.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: LoadBalancerStatusImprovements
  - Components depending on the feature gate:
     - cloud-controller-manager

###### Does enabling the feature change any default behavior?

Yes. Cloud providers that have been updated to support the new
functionality will set new status fields on Services of `Type:
LoadBalancer`.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

###### What happens if we reenable the feature if it was previously rolled back?

It works.

###### Are there any tests for feature enablement/disablement?

Not planned, because the enabled/disabled transition itself is not
particularly interesting.

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

Individual cloud providers might choose to change their behavior to
take advantage of the new possibility for error reporting, by
henceforth refusing to create load balancers that they previously
would have created incorrectly. However, they should not actually
modify the behavior of _existing_ load balancers.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

By looking at the Status of load balancer Services. (If the feature is
enabled, and there is an updated cloud provider, and there are load
balancer services in the cluster, then the feature is automatically in
use.)

###### How can someone using this feature know that it is working for their instance?

- [X] API .status
  - Condition name: `LoadBalancerProvisioning`, `LoadBalancerServing`
  - Other field: TBD

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This KEP mostly does not aim to change the SLOs around load balancer
provisioning, it just aims to make the process more visible.

The time it takes for a load balancer Service to be updated with a
"provisioning" status after being created should be the same as for
other "simple" controller interactions. For e2e testing we will use 30
seconds, which matches the amount of time we wait to see an
EndpointSlice change from the EndpointSlice controller after updating
a Pod.

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

This is a cloud-provider feature, but it does not depend on anything
other than that.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

The cloud provider will make more updates to `Type: LoadBalancer`
Services, to update the conditions and new status fields. Components
that watch Services will then also get more updates (though none of
them would have any reason to react to the new updates).

Overall impact should be very low.

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No... the cloud provider will be doing more, but not because of
additional calls.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

It will add 2 new conditions to Services of `Type: LoadBalancer`, and
possibly additional status fields TBD.

Non-load-balancer Services should not be changed.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No; the additional status reporting by the cloud provider is all
associated with things that it is already doing anyway (and just not
reporting on the status of).

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
