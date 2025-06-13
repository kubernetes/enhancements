# KEP-4631: LoadBalancer Service Status Improvements

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [<code>Service.Status.Conditions</code>](#servicestatusconditions)
    - [The <code>LoadBalancerServing</code> Condition](#the-loadbalancerserving-condition)
    - [The <code>LoadBalancerProvisioning</code> Condition](#the-loadbalancerprovisioning-condition)
    - [Service Lifecycle and Interaction of the Provisioning/Serving Conditions](#service-lifecycle-and-interaction-of-the-provisioningserving-conditions)
    - [The <code>LoadBalancerDegraded</code> Condition](#the-loadbalancerdegraded-condition)
  - [<code>Service.Spec.RequiredLoadBalancerFeatures</code>](#servicespecrequiredloadbalancerfeatures)
  - [Newly-Required Behavior for Cloud Providers](#newly-required-behavior-for-cloud-providers)
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

This KEP proposes new additions to `service.Status.Conditions` to
allow cloud providers to better communicate the status of load
balancer support and provisioning, and a new addition to
`service.Spec` to allow users (including e2e tests) to better
communicate which pieces of Service functionality are
mandatory-to-implement for the cloud provider.

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

- Allow users and e2e tests to prevent the cloud provider from
  provisioning a load balancer without support for a particular
  feature the user/test needs (even when that is a feature the cloud
  provider doesn't know about).

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

1. Define new conditions for `Service.Status.Conditions` to allow the
   cloud provider to communicate load balancer status better to the
   user.

2. Define clearer rules about what cloud providers are required to pay
   attention to in the Service object, and how they should behave when
   they can't provision a load balancer that exactly implements the
   requested semantics.

3. Define new API to allow the user to specify when they would prefer
   that the cloud fail to provision a load balancer rather than
   partially provisioning it.

### Risks and Mitigations

Using the new features should not introduce any new risks.

They should hopefully _improve_ security for users. Currently, you
sometimes get load balancers that aren't what you wanted (e.g.,
failing to implement source ranges or session affinity). This KEP
makes it possible for users to prevent this from happening.

(If a cloud provider implemented the API incorrectly and claimed
support for features that it didn't actually support, then that could
lead to users being more confident when deploying broken load
balancers than they are now, but the cloud provider would fail the e2e
tests in that case, so this shouldn't happen.)

## Design Details

### `Service.Status.Conditions`

The problem of indicating provisioning progress and provisioning
failure can easily be solved with conditions on the Service.

We will add three new Service conditions:

  - `LoadBalancerServing`, to indicate explicitly that the load
    balancer is (or isn't) serving traffic.

  - `LoadBalancerProvisioning`, to indicate that the cloud provider
    is (or isn't) configuring/provisioning the load balancer.

  - `LoadBalancerDegraded`, to indicate that the load balancer is
    aware that it is not fully implementing the semantics requested by
    the user, in cases where implementing the full semantics is not
    mandatory.

#### The `LoadBalancerServing` Condition

The `LoadBalancerServing` condition, when `True`, indicates that
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

The well-known `Reason` values for the `LoadBalancerServing` condition
are:

  - `Status=True, Reason=Serving`

  - `Status=False, Reason=Provisioning`: The load balancer is not
    currently serving because it is being provisioned /
    re-provisioned.

  - `Status=False, Reason=Unsupported`: No load balancer could be
    provisioned, because the cloud provider doesn't support some
    feature of this Service (explained further in `Message`).

  - `Status=False, Reason=Infrastructure`: No load balancer could be
    provisioned, because of a problem with the underlying cloud
    infrastructure (explained further in `Message`).

#### The `LoadBalancerProvisioning` Condition

The `LoadBalancerProvisioning` condition, when `True`, indicates that
the cloud provider is currently performing a time-consuming
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

The well-known `Reason` values for the `LoadBalancerProvisioning`
condition are:

  - `Status=True, Reason=Provisioning`

  - `Status=False, Reason=Complete`: The cloud provider has finished
    trying to provision the load balancer (whether successfully or
    unsuccessfully, as indicated by the `Message` and by the
    `LoadBalancerServing` condition).

#### Service Lifecycle and Interaction of the Provisioning/Serving Conditions

When a load balancer Service is first created (or a non-load-balancer
Service is converted to `Type: LoadBalancer`), one of three things
should immediately happen:

  1. The cloud provider sets `LoadBalancerProvisioning=False`,
     `LoadBalancerServing=False`, with appropriate `Reason` values,
     indicating that it cannot or will not provision a load balancer
     for the service.

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

#### The `LoadBalancerDegraded` Condition

The `LoadBalancerDegraded` condition, when `True`, indicates that the
cloud provider is aware that it has provisioned the load balancer in a
way that may not fully implement the semantics of the Service. For
example, the service requests session affinity, but the load balancer
does not implement that feature.

(This is not really a great behavior on the cloud provider's part, but
this is how cloud providers have traditionally behaved, and some users
may be relying on this in some cases. The `LoadBalancerDegraded`
condition allows the load balancer to be _explicit_ about the fact
that it is doing this. However, a cloud provider may also decided to
not provision a load balancer in these cases (with
`LoadBalancerServing` set to `False` with `Reason=Unsupported`).)

If the cloud provider is not intentionally providing a degraded load
balancer, it should leave this condition unset, which clients should
treat as meaning `Unknown`. In general it is not possible for a cloud
provider to say definitively that a load balancer is _not_ degraded,
since the Service object may be using API fields newer than the
latest version of the API that the cloud provider knows about.

Most of the well-known `Reason` values for this condition consist of a
`ServiceSpec` field name followed by "`NotSupported`". Thus,
`Reason=IPFamiliesNotSupported` implies that the load balancer is
degraded because of something in `Service.Spec.IPFamilies` that could
not be implemented.

The well-known `Reason` values for the `LoadBalancerDegraded`
condition are:

  - `Status=True, Reason=IPFamiliesNotSupported`: The service is
    dual-stack, but the cloud provider has provisioned a single-stack
    load balancer.

  - `Status=True, Reason=PortsNotSupported`: The service has multiple
    ports, but the cloud provider has provisioned a load balancer that
    does not support all of them (possibly because one of the ports
    uses an unsupported protocol, or because the cloud provider does
    not support mixed-protocol load balancers).

  - `Status=True, Reason=LoadBalancerIPNotSupported`: The service
    requested a specific `LoadBalancerIP`, but the cloud provider does
    not support this functionality or was not able to use the
    requested IP. (Note that `LoadBalancerIP` is deprecated, and is
    not supported by most cloud providers.)

  - `Status=True, Reason=SessionAffinityNotSupported`: The service
    requested session affinity but the cloud provider has provisioned
    a load balancer that does not support it.

  - `Status=True, Reason=ExternalTrafficPolicyNotSupported`: The
    service requested `Local` external traffic policy, but the cloud
    provider can't preserve the source IP for load balancer traffic.

  - `Status=True, Reason=Multiple`: The load balancer is degraded for
    multiple reasons.

The `LoadBalancerDegraded` condition with `Reason=PortsNotSupported`
replaces the `LoadBalancerPortsError` condition from [KEP-1435], which
was never widely implemented or used. (`LoadBalancerPortsError` will
be marked as deprecated in the API, but not removed from anywhere that
is currently using it.)

[KEP-1435]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/1435-mixed-protocol-lb

### `Service.Spec.RequiredLoadBalancerFeatures`

The new `RequiredLoadBalancerFeatures` field on `v1.ServiceSpec` is an
array of strings indicating mandatory-to-implement features of the
Service. If set, it means that if the cloud provider cannot provision
a load balancer that implements these features correctly, it must fail
to provision the load balancer entirely rather returning a load
balancer with the `LoadBalancerDegraded` condition set.

Clients should assume that cloud providers obey
`RequiredLoadBalancerFeatures` if and only if they set the new
load-balancer-related service conditions. If a client sets
`RequiredLoadBalancerFeatures` and the cloud provider provisions a
load balancer _without_ setting the corresponding conditions, then the
client should assume that the cloud provider was unaware of the
required features request.

When `RequiredLoadBalancerFeatures` is not set, it is up to the cloud
provider how to handle these features. Note that in the case of
Service features that are newer than this KEP, it is possible that a
cloud provider may be unaware of the feature entirely, and thus if it
is not requested in `RequiredLoadBalancerFeatures`, the cloud provider
may not even realize that the Service is requesting that feature, and
so it might ignore it without any notification to the user. (As a
concrete example, at the time of writing, [KEP-4444]'s
`TrafficDistribution` field is available as an Alpha API, but no cloud
providers are yet aware of it, and thus they will likely provision
load balancers that ignore it.)

[KEP-4444]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/4444-service-traffic-distribution/

As with the `LoadBalancerDegraded` well-known `Reason` values, these
values are based on the names of relevant `ServiceSpec` fields.

```golang
type ServiceSpec struct {
        ...

        // RequiredLoadBalancerFeatures indicates the
        // load-balancer-related features that are considered
        // mandatory-to-implement for this service. (In general,
        // the values are the names of fields in ServiceSpec.)
        // If this is empty, the cloud provider may provision a load
        // balancer that does not implement every feature of this
        // Service (for example, providing a single-stack load
        // balancer for a dual-stack service, or ignoring session
        // affinity).
        RequiredLoadBalancerFeatures []LoadBalancerFeature
}

// LoadBalancerFeature indicates a feature that might be required
// for a service of type LoadBalancer.
type LoadBalancerFeature string

const (
        // LoadBalancerFeatureIPFamilies indicates that the cloud
        // must provision a load balancer that will accept connections
        // on all of the service's IP families.
        LoadBalancerFeatureIPFamilies LoadBalancerFeature = "IPFamilies"

        // LoadBalancerFeaturePorts indicates that the cloud must
        // provision a load balancer that will accept connections on
        // all of the service's ports and protocols.
        LoadBalancerFeaturePorts LoadBalancerFeature = "Ports"

        // LoadBalancerFeatureSessionAffinity indicates that the cloud
        // must provision a load balancer that supports the requested
        // session affinity.
        LoadBalancerFeatureSessionAffinity LoadBalancerFeature = "SessionAffinity"

        // LoadBalancerFeatureLoadBalancerIP indicates that the cloud
        // must provision a load balancer with the requested
        // LoadBalancerIP. (Note that LoadBalancerIP is deprecated
        // and not supported by most cloud providers.)
        LoadBalancerFeatureLoadBalancerIP LoadBalancerFeature = "LoadBalancerIP"

        // LoadBalancerFeatureExternalTrafficPolicy indicates that the
        // cloud must provision a load balancer that fully supports
        // the requested ExternalTrafficPolicy. In particular, if the
        // traffic policy is `Local`, then the cloud must preserve the
        // original source IP of traffic passing through the load balancer.
        LoadBalancerFeatureExternalTrafficPolicy LoadBalancerFeature = "ExternalTrafficPolicy"
)
```

### Newly-Required Behavior for Cloud Providers

Cloud providers that implement this KEP are required to recognize and
correctly handle most Service features that were GA before this KEP
become Alpha. In particular:

  - The cloud provider must ignore a Service entirely if the Service
    has a non-empty `LoadBalancerClass` and the cloud provider has not
    been configured to handle the indicated class. In this case it
    must neither provision a load balancer for the service, nor modify
    its `.Status.LoadBalancer`, nor set any conditions on it.

  - The cloud provider must refuse to provision a load balancer for a
    Service (setting the `LoadBalancerServing` condition to `False`
    with `Reason=Unsupported`) if:

      - the Service _exclusively_ uses an IP family that the cloud
        provider does not support. (e.g., the Service is single-stack
        IPv6, but the cloud only supports IPv4 load balancers.)

      - the Service _exclusively_ uses L4 protocols that the cloud
        provider does not support. (e.g., all of the Service's ports
        are UDP, but the cloud only supports TCP load balancers.)

      - the Service sets `AllocateLoadBalancerNodePorts: false`, but
        the cloud only supports NodePort-based load balancing.

      - the Service's `RequiredLoadBalancerFeatures` include any
        feature that the cloud provider does not support (or does not
        know about).

  - The cloud provider must either refuse to provision a load balancer
    (as above), or else set the `LoadBalancerDegraded` condition
    appropriately if:

      - the Service requests `SessionAffinity` but the cloud provider
        does not support affinity.

      - the Service requests `LoadBalancerSourceRanges` but the cloud
        provider does not support source range filtering (and does not
        implement load balancing in a way that would allow the service
        proxy to do source range filtering itself).

      - the Service requests a specific `LoadBalancerIP`, but the
        cloud provider does not support that field or else could not
        acquire the requested IP. (Note that this field is deprecated,
        and cloud providers that do not currently support it should
        continue to not support it; they should just make their
        non-support more explicit.)

  - The cloud provider must set the `LoadBalancerDegraded` condition
    appropriately if:

      - the Service requests `ExternalTrafficPolicy: Local`, but the
        cloud provider does not preserve the original source IP of
        load-balanced connections.

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

  1. We will set `RequiredLoadBalancerFeatures` on the Service when
     testing optional / new / not-universally-implemented features.

  2. We will expect a condition to be set on the Service within 30
     seconds. (That matches `e2eservice.KubeProxyEndpointLagTimeout`,
     the maximum lag between updating a Pod or Service and seeing
     those updates reflected in EndpointSlices, which involves the
     same "e2e.test to apiserver to controller to apiserver to
     e2e.test" path.) Once the condition is set we will still use the
     existing `e2eservice.GetServiceLoadBalancerCreateTimeout()` value
     (based on cluster size) to wait for provisioning to complete.

  2. Once the conditions indicate that the load balancer is serving,
     `IP` based load balancers will be expected to work immediately
     (and the test will fail if they do not). `Hostname` based load
     balancers will be allowed some lag time for DNS propagation, but
     will be expected to work immediately once the DNS name resolves.

  4. In general, tests will:

       - **Pass** if the cloud provider provisions a load balancer
         successfully and then passes the e2e test.

       - **Fail** if the cloud provider provisions a load balancer
         successfully and then fails the e2e test.

       - **Fail** if the test times out due to the load balancer not
         being provisioned.

       - **Skip** themselves if the cloud provider sets the
         `LoadBalancerServing` condition on the service to `False`
         with `Reason=Unsupported`.

       - **Skip** themselves if the cloud provider sets the
         `LoadBalancerDegraded` condition on the service to `True`
         with a `Reason` that is appropriate for the given test.

Specifically:

- These tests should pass on all clouds (even ones that haven't been
  updated to support this KEP), because they only use required
  functionality:

    - [`LoadBalancers should be able to change the type and ports of a TCP
      service`](https://github.com/kubernetes/kubernetes/blob/a6b08a8ea4/test/e2e/network/loadbalancer.go#L132).

    - [`LoadBalancers should handle load balancer cleanup finalizer for
      service`](https://github.com/kubernetes/kubernetes/blob/a6b08a8ea4/test/e2e/network/loadbalancer.go#L610).

    - [`LoadBalancers should not have connectivity disruption during
      rolling update with externalTrafficPolicy=Cluster`](https://github.com/kubernetes/kubernetes/blob/a6b08a8ea4/test/e2e/network/loadbalancer.go#L971).

- These tests may be skipped if the cloud provider indicates that the
  feature is unsupported.

    - [`LoadBalancers should be able to create LoadBalancer Service
      without NodePort and change it`](https://github.com/kubernetes/kubernetes/blob/a6b08a8ea4/test/e2e/network/loadbalancer.go#L642).

    - [`LoadBalancers should be able to change the type and ports of a UDP
      service`](https://github.com/kubernetes/kubernetes/blob/a6b08a8ea4/test/e2e/network/loadbalancer.go#L270).

        - The test should be rearranged to create the LB first (so the
          test can be skipped right away if the cloud doesn't support
          UDP).

    - [`LoadBalancers should be able to preserve UDP traffic when server
      pod cycles for a LoadBalancer service ( on different nodes | on the
      same node)`](https://github.com/kubernetes/kubernetes/blob/a6b08a8ea4/test/e2e/network/loadbalancer.go#L707).

- These tests using `ExternalTrafficPolicy: Local` should pass on all
  clouds. The cloud provider may mark the Service as degraded with
  `Reason=ExternalTrafficPolicy` if it does not support source IP
  preservation, but these tests do not care about that:

    - [`LoadBalancers should not have connectivity disruption during
      rolling update with externalTrafficPolicy=Local`](https://github.com/kubernetes/kubernetes/blob/a6b08a8ea4/test/e2e/network/loadbalancer.go#L980).

    - [`LoadBalancers ExternalTrafficPolicy: Local should only target
      nodes with endpoints`](https://github.com/kubernetes/kubernetes/blob/a6b08a8ea4/test/e2e/network/loadbalancer.go#L1077).

    - [`LoadBalancers ExternalTrafficPolicy: Local should target all
      nodes with endpoints`](https://github.com/kubernetes/kubernetes/blob/a6b08a8ea4/test/e2e/network/loadbalancer.go#L1153).

- These tests using `ExternalTrafficPolicy: Local` need to be split
  into two tests each: one that just tests connectivity, and allows a
  degraded load balancer (as above); and one that tests source IP
  preservation as well, and thus uses `RequiredLoadBalancerFeatures:
  ["ExternalTrafficPolicy"]`.

    - [`LoadBalancers ExternalTrafficPolicy: Local should work for
      type=LoadBalancer`](https://github.com/kubernetes/kubernetes/blob/a6b08a8ea4/test/e2e/network/loadbalancer.go#L1012).

    - [`LoadBalancers ExternalTrafficPolicy: Local should work from pods`](https://github.com/kubernetes/kubernetes/blob/a6b08a8ea4/test/e2e/network/loadbalancer.go#L1200).

- These tests will use `RequiredLoadBalancerFeatures: ["SessionAffinity"]`
  and may skip themselves if the cloud provider indicates that the
  feature is unsupported. They do not test source IP preservation and
  thus will not set `"ExternalTrafficPolicy"` as a required feature.

    - [`LoadBalancers should have session affinity work for LoadBalancer
      service with ( Local | Cluster ) traffic
      policy`](https://github.com/kubernetes/kubernetes/blob/a6b08a8ea4/test/e2e/network/loadbalancer.go#L565).

    - [`LoadBalancers should be able to switch session affinity for
      LoadBalancer service with ( Local | Cluster ) traffic
      policy`](https://github.com/kubernetes/kubernetes/blob/a6b08a8ea4/test/e2e/network/loadbalancer.go#L575).

- This test will use `RequiredLoadBalancerFeatures: ["LoadBalancerSourceRanges"]`
  and may skip itself if the cloud provider indicates that the feature
  is unsupported.

    - [`LoadBalancers should only allow access from service loadbalancer
      source ranges`](https://github.com/kubernetes/kubernetes/blob/a6b08a8ea4/test/e2e/network/loadbalancer.go#L422).

- We lack tests of dual-stack and multi-protocol load balancer
  provisioning, and should add those now that it's possible for clouds
  to signal their failure to support it.

- We should add a test for "cloud providers ignore Services with
  `LoadBalancerClass` set".

### Graduation Criteria

#### Alpha

- Helpers for the new API implemented in `k8s.io/cloud-provider`

- cloud-provider-kind and cloud-provider-gcp set conditions
  appropriately when the feature gate is enabled, and obey
  `RequiredLoadBalancerFeatures` when it is set.

- New version of e2e load balancer tests that use the
  conditions/status, which pass on GCE and kind when the feature gate
  is enabled.

- Update documentation to describe what happens when you request a
  load balancer that the cloud provider cannot implement (in part or
  in full).

#### Beta

- At least two additional cloud providers (one of which must be
  cloud-provider-aws or cloud-provider-azure) have been updated to use
  the new functionality, and we pass/skip the e2e tests as expected
  for those platforms.

#### GA

- At least two additional cloud providers (one of which must be a
  non-public-cloud provider like cloud-provider-vsphere or
  cloud-provider-openstack) have been updated to use the new
  functionality, and we pass/skip the e2e tests as expected for those
  platforms.

- Allowing time for feedback...?

### Upgrade / Downgrade Strategy

When upgrading/enabling the feature gate, cloud providers should
update existing already-provisioned load balancers with appropriate
status information.

When downgrading/disabling, the cloud provider does not need to do
anything.

### Version Skew Strategy

`Service.Status.Conditions` already exists, so there are no skew
issues with that. The conditions are not read by any Kubernetes
components (other than the e2e tests), so there are no
upgrade/rollback-breaking situations where a newer client might be
upset that an older cloud provider is not setting the conditions
(though it would be possible for new user workloads to get confused by
this during a rollback).

`Service.Spec.RequiredLoadBalancerFeatures` is a new addition. It is
not set by any Kubernetes components (other than the e2e tests), so
again, any skew problems involving it would not affect Kubernetes
directly, but might confuse new user workloads if they try to use the
field without validating that it exists on the apiserver. (In that
case, the client might think it had set `RequiredLoadBalancerFeatures`
and the cloud provider had set conditions implying it had agreed to
it, when in fact it hadn't seen that field at all.)

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: LoadBalancerStatusImprovements
  - Components depending on the feature gate:
     - cloud-controller-manager
     - kube-apiserver

```
<<[UNRESOLVED feature gate name ]>>

"LoadBalancerStatusImprovements" is not entirely accurate given the
addition of RequiredLoadBalancerFeatures to Service.Spec...

<<[/UNRESOLVED]>>
```

###### Does enabling the feature change any default behavior?

Yes. Cloud providers that have been updated to support the new
functionality will set new status fields on Services of `Type:
LoadBalancer`.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

###### What happens if we reenable the feature if it was previously rolled back?

It works.

###### Are there any tests for feature enablement/disablement?

TODO (we should make sure the cloud provider updates pre-existing
services in the enablement case).

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

The `LoadBalancerPortsError` Service condition from [KEP-1435] is now
considered deprecated (being redundant with the more general-purpose
`LoadBalancerDegraded` condition). However, the condition is not being
removed from the cloud provider(s?) that previously set it; it's just
no longer recommended that other cloud providers should implement it.

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
  - Condition name: `LoadBalancerProvisioning`, `LoadBalancerServing`, `LoadBalancerDegraded`

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
Services, to update the conditions and new status field. Components
that watch Services will then also get more updates (though none of
them would have any reason to react to the new updates).

Overall impact should be very low.

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No... the cloud provider will be doing more, but not because of
additional calls.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

It will add 2 or 3 new conditions to Services of `Type: LoadBalancer`, and
one new optional spec field.

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
