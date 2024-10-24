# KEP-4444: Traffic Distribution for Services

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Topology Keys](#topology-keys)
  - [Topology Aware Routing and InternalTrafficPolicy](#topology-aware-routing-and-internaltrafficpolicy)
- [Goals](#goals)
- [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Standard Heuristic Implementation (kube-proxy dataplane)](#standard-heuristic-implementation-kube-proxy-dataplane)
    - [Default (i.e. <code>trafficDistribution</code> is not configured)](#default-ie-trafficdistribution-is-not-configured)
    - [<code>PreferClose</code>](#preferclose)
  - [Changes within kube-proxy](#changes-within-kube-proxy)
  - [Choice of field name](#choice-of-field-name)
  - [Intersection with internal/externalTrafficPolicy](#intersection-with-internalexternaltrafficpolicy)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Possible future expansions](#possible-future-expansions)
  - [Status Reporting](#status-reporting)
    - [Condition usage by other implementations](#condition-usage-by-other-implementations)
  - [Implementation specific heuristics](#implementation-specific-heuristics)
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
  - [Repurpose the existing topology annotation to recognize additional values](#repurpose-the-existing-topology-annotation-to-recognize-additional-values)
  - [Reuse the fields internal/externalTrafficPolicy to offer these routing preferences](#reuse-the-fields-internalexternaltrafficpolicy-to-offer-these-routing-preferences)
  - [Granular Routing Controls](#granular-routing-controls)
  - [Reuse Pod Topology Spread Constraints for Traffic Distribution](#reuse-pod-topology-spread-constraints-for-traffic-distribution)
    - [Complementary Use of Pod Topology Spread Constraints and trafficDistribution](#complementary-use-of-pod-topology-spread-constraints-and-trafficdistribution)
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

This KEP proposes introducing a new field, `trafficDistribution`, to the
Kubernetes Service spec. It will supersede the functionality currently provided
by the `service.kubernetes.io/topology-mode` annotation and it’s precursor
`topologyKeys` field (which has been deprecated since Kubernetes 1.21)

## Motivation

To be able to understand the motivations for introducing this new field, it’s
important to understand the precursors that shaped this KEP.


### Topology Keys

* This early mechanism allowed users to specify ordered routing preferences (e.g.,
prioritize local nodes, then zone, then anywhere). 

* **Limitations:** 
  * It lacked flexibility for implementations to make smart decisions on its
    own, like incorporating feedback-based routing optimizations (e.g., avoiding
    overloaded endpoints). 
  * The name `topologyKeys` might be perceived as overly restrictive, given the
    desire to incorporate non-topology factors into routing decisions.
  * The arbitrary ordering, lengths, and keys made it difficult to integrate
    with systems that have more standard settings like preferring same zone
    routing. With `topologyKeys`, a user could theoretically request that an
    entirely arbitrary topology key be given preference over zone or region,
    which might not be possible or extremely difficult for an implementation to
    achieve.

### Topology Aware Routing and InternalTrafficPolicy

Topology aware routing together with `internalTrafficPolicy` were meant to be
the successors of `topologyKeys` and allow implementations to be more flexible.

* TopologyAwareRouting: 
  * Responds the annotation `service.kubernetes.io/topology-mode`. When this
    annotation is set to Auto, an implementation specific heuristic is used to
    route the traffic. 
  * **Goal:** The aim with Auto was to allow implementations to be as smart as
    possible and choose what it considered to be the best routing criteria. The
    field didn’t support any other value by design with the thought that the
    implementation should be able to make the best possible choice for the user.
  * **Limitation:** 
      * While designed for intelligent routing, this mode offers less user
        control and can be less predictable. Users filed issues reporting that
        hints weren't being applied or didn't function as expected.
      * The design attempted to strike a balance between safety and preferential
        zone-based routing, but may have fallen short in achieving both
        effectively. Some users prioritize predictability for performance
        optimization, even when accepting potential endpoint overload in certain
        situations.

* InternalTrafficPolicy 
  * The spec.internalTrafficPolicy field with the "Local" setting restricts
    traffic to endpoints on the same node as the originating Pod. 
  * **Goal:** With the deprecation of topologyKeys, one of the driving reasons was
    to satisfy the semantic use cases of strictly routing the traffic to the
    same node.
  * **Limitation:** Lacks failover; traffic is dropped if no local endpoint exists.

Note that while the initial proposal of InternalTrafficPolicy proposed a
`PreferLocal` policy, it was dropped later on. This meant that now
TopologyAwareRouting in conjunction with InternalTrafficPolicy didn’t exactly
allow users to express a much desired use case from topologyKeys which is
"prefer node-local, failover to same zone, then route anywhere" While this
specific behavior is a non-goal for the initial implementation of the
`trafficDistribution` field, the design allows for the potential introduction of
such a preference in future refinements.

## Goals

* **Enhanced User Control:** Provide users with a knob to specify a preference that
  MAY influence the traffic routing of a Kubernetes Service.

* **Guidance for Implementations:** Offer standard routing preference values
  that implementations can choose to recognize and support, promoting a degree
  of commonality.

* **Flexibility:** Allow implementations to interpret and support standard
  heuristics in a manner that aligns with their capabilities and design.

* **Extensibility:** Enable Kubernetes implementations to introduce and support
  innovative routing strategies (for example, those based on topology, latency,
  or other custom heuristics) that can evolve alongside users needs and
  infrastructure capabilities.

## Non-Goals

* **Strict Routing Guarantees:** The `trafficDistribution` field will not
  enforce deterministic routing paths. It serves as a mechanism for expressing
  hints and preferences that implementations can consider when making routing
  decisions.

* **Replacement of Traffic Policies:** The new field is complementary to
  `internalTrafficPolicy` and `externalTrafficPolicy`. It does not aim to
  substitute their role in enforcing strict traffic locality.

* **Immediate Support for All Possible Heuristics:** The initial implementation
  focuses on a core set of heuristics. Addition of new heuristics (like
  `Local` for Node local preference) could be explored in future
  refinements.

## Proposal

Add a new field, `trafficDistribution`, to the Service specification. This field
will serve as preference or hint for the underlying implementation to consider
while making routing decisions. It does not offer strict routing guarantees.

The field will support the following initial values:


* `PreferClose`: Indicates a preference for routing traffic to endpoints that
  are topologically proximate to the client. The interpretation of
  "topologically proximate" may vary across implementations and could encompass
  endpoints within the same node, rack, zone, or even region.

The absence of a value indicates no specific routing preference. In this case,
the user delegates the routing decision to the implementation, allowing it to
apply its best-effort strategy.

Implementations SHOULD support the standard values. While some flexibility in
interpretation is permitted, implementations should aim to align their behavior
with the described intent of these preferences as closely as possible.

NOTE: Implementations reserve the right to refine the behavior associated with
  any heuristic, including standard heuristics. This means the behavior enabled
  by values such as `PreferClose` might evolve over time, and some
  evolutions might interpret the heuristic goals slightly differently. For
  example, in the case of `PreferClose`, an implementation might initially route
  traffic within the zone without considering endpoint overload, while a future
  refinement could introduce feedback mechanisms to detect overload and route
  traffic outside the zone when necessary, optimizing overall performance. The
  decision of what constitutes an "improvement" remains at the discretion of the
  implementation.

### User Stories

#### Story 1

* **Requirement:** I don't have strong preferences for how my application
  traffic is routed. I prioritize simplicity and trust my Kubernetes
  implementation to optimize traffic distribution.
* **Solution:** Leave the `trafficDistribution` field unset.
* **Effect:** The Kubernetes implementation will apply its best-effort routing
  strategy based on its design. This strategy might change over time as the
  implementation evolves. It may load balance across zones or regions.

#### Story 2
* **Requirement:** I want my application to primarily send traffic to endpoints
  that are topologically close for performance or cost reasons. However, I want
  to avoid connection failures if no sufficiently close endpoints are available.
* **Solution:** Set `trafficDistribution: PreferClose`
* **Effect:** The Kubernetes implementation will aim to prioritize routing
  traffic to endpoints in the same zone as the client. If no endpoints are
  available within the zone, traffic will be routed to other zones. It's
  possible that traffic patterns could lead to endpoints within the preferred
  zone becoming overloaded. Consider other routing strategies or scaling up
  resources within the zone if this becomes a concern.

#### Story 3
* **Requirement:** As a developer of a widely deployed cluster-addon, I want to
  be able to provide users an easy way to configure my Helm chart and/or
  deployment configuration to enable same-zone routing behavior in a portable
  way that works across many different environments.
* **Solution:** Using one of the available standard values in the
  `trafficDistribution` field will allow the customer to use the same Helm Chart
  configurations with greater confidence regardless of the underlying Kubernetes
  implementation. This simplifies their deployment process and reduces the
  complexity of managing cross-cluster applications.

### Notes/Constraints/Caveats

This proposal is our third attempt at an API revolving around such a
configuration. There's a non-zero chance that we may need to revisit this again.

### Risks and Mitigations

* **Risk:** Having a routing preference like `PreferClose` comes at the risk of
  endpoints in certain locality being overloaded if the originating traffic is
  skewed towards a particular locality.

  **Mitigation:**
    * Emphasize in the documentation that the `PreferClose` preference is
      designed for low-latency or monetory-cost reasons, with the understanding
      that it can lead to overload within that locality.
    * Recommend approaches like having deployments per locality, (like a zone
      locality when using kube-proxy as the data-plane), which can scale
      independently of other localities.

* **Risk:** Users migrating from the `service.kubernetes.io/topology-mode`
  annotation might encounter differences in exact routing behavior:
  * The new field doesn't support a routing preference that is exactly similar
    to using `service.kubernetes.io/topology-mode=Auto` from the old annotation.
  * If both field and the annotation are set, the annotation will take
    precedence. (However, this behavior is temporary as the annotation will be
    deprecated and removed in future releases)

  **Mitigation:** Properly document the suggested migration paths with
  limitations.

## Design Details

### Standard Heuristic Implementation (kube-proxy dataplane)

kube-proxy (along with EndpointSlice controller, within kube-controller-manager
as the control plane) will start with supporting two distinct behaviors based on
the value configured for `trafficDistribution`

#### Default (i.e. `trafficDistribution` is not configured)
* **Meaning:** When `trafficDistribution` is not used, kube-proxy would match
  it's existing behaviour of having an equal distribution of traffic across
  endpoints (potentially spanning multiple zones or regions)
* This leverages existing implementation, requiring no major changes.

#### `PreferClose`
* **Meaning:** Attempts to route traffic to endpoints within the same zone as
  the client. If no endpoints are available within the zone, traffic would be
  routed to other zones.
* This preference will be implemented by the use of Hints within EndpointSlices.
* We already use Hints to implement `service.kubernetes.io/topology-mode: Auto`
  In a similar manner, the EndpointSlice controller will now also populate hints
  for `trafficDistribution: PreferClose` -- although in this case, the zone hint will
  match the endpoint of the zone itself.
* While it may seem redundant to populate the hints here since kube-proxy can
  already derive the zone hint from the endpoints zone (as they would be the
  same), we will still use this for implementation simply because of the reason
  that it’s easier to implement and provides a better design. Consider an
  alternative implementation where kube-proxy reads
  `trafficDistribution: PreferClose` field and then constructs the route rules
  accordingly. This means some extra logic needs to be baked into the kube-proxy
  which could have just as easily been implemented by an already existing
  extensibility mechanism (i.e. EndpointSlice hints)

Although this is not an explicit design goal, an implication of the above
implementation choice means that:
* The control plane (kube-controller-manager in this case) is only concerned
  with the field `trafficDistribution` and populates EndpointSlice hints based
  on its value 
* The data plane (kube-proxy in this case) is only concerned with the Hints
  populated in the EndpointSlice and the fields `internal/externalTrafficPolicy`
  to make routing decisions.
* Neither the control plane nor the data plane looks at the others field.

NOTE: The fact that EndpointSlice hints are not expected to be implemented by
  all data planes is not of concern here, because `trafficDistribution` is anyways
  supposed to be implementation dependent. Since we know that the kube-proxy
  data plane will respect the Hints, it’s sufficient for the kube-proxy and
  kube-controller-manager based implementations to say that we do support the
  standard heuristics.

### Changes within kube-proxy

**Present behaviour:** kube-proxy only considers EndpointSlice hints for route
programming if the `service.kubernetes.io/topology-aware-hints` annotation is
set to "Auto"

**New behaviour:** Irrespective of what the annotation
`service.kubernetes.io/topology-aware-hints` or field `trafficDistribution` are
set to (or even if they are not set at all), kube-proxy will always consider
EndpointSlice hints (assuming this feature-gate is enabled).

NOTE: The expectation remains that *all* endpoints within an EndpointSlice must
  have corresponding hints for kube-proxy to utilize them. This avoids scenarios
  with partial hints. The reason for this requirement is the same one highlighted in [KEP-2433 Topology
  Aware
  Hints](https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/2433-topology-aware-hints/README.md#kube-proxy), i.e. _"This is to provide safer transitions between enabled and disabled states. Without this fallback, endpoints could easily get overloaded as hints were being added or removed from some EndpointSlices but had not yet propagated to all of them."_

### Choice of field name
The name `trafficDistribution` is meant to capture the highly
implementation-specific nature of this field and how it affects the routing of
traffic
* Field names that include the word "topology", like `topologyPreference` and
  `topologyRoutingPreference`, were avoided because:
  * Topology word might be a bit vague
  * The actual heuristic might not specifically be tied to the geographical or
    network topology, but might also be considering factors like latency, active
    connections, cpu usage of the backends, etc.
* Use of the word “hint” for names like `trafficRoutingHint` and
  `customRoutingHint` was avoided so as not to confuse with the hint field
  within EndpointSlices which has previously been discussed in relation to such
  routing heuristics. In a sense, the hint field within EndpointSlices is just
  an implementation detail.
* Use of words like “policy” for names like trafficPolicy was not favored so
  that it’s clearly different from the `internalTrafficPolicy` and
  `externalTrafficPolicy` fields which have a more fixed behavior.
* Use of the word "selection" for a name like `endpointSelection` were avoided
  so as not to confuse with the actual process of selecting the complete set of
  pods backing a service.

### Intersection with internal/externalTrafficPolicy

The intersection of the field with `internalTrafficPolicy` and
`externalTrafficPolicy` fields remains the same as the annotation. The following
table borrowed from [KEP-2086: Service Internal Traffic
Policy](https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/2086-service-internal-traffic-policy)
captures the precedence

<table>
<thead>
  <tr>
    <th>ExternalTrafficPolicy</th>
    <th>InternalTrafficPolicy</th>
    <th>trafficDistribution</th>
    <th>External Result</th>
    <th>Internal Result</th>
  </tr>
</thead>
<tbody>
  <tr>
    <td>-</td>
    <td>-</td>
    <td>Auto</td>
    <td>trafficDistribution: Auto</td>
    <td>trafficDistribution: Auto</td>
  </tr>
  <tr>
    <td>Local</td>
    <td>-</td>
    <td>Auto</td>
    <td>ExternalTrafficPolicy: Local</td>
    <td>trafficDistribution: Auto</td>
  </tr>
  <tr>
    <td>Local</td>
    <td>Local</td>
    <td>Auto</td>
    <td>ExternalTrafficPolicy: Local</td>
    <td>InternalTrafficPolicy: Local</td>
  </tr>
</tbody>
</table>

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

No updates required.

##### Unit tests

The main packages that will see major changes due to this enhancement are:
- `k8s.io/kubernetes/vendor/k8s.io/endpointslice/topologycache`: `2024-01-26` -
  `74.8`
- `k8s.io/kubernetes/vendor/k8s.io/endpointslice`: `2024-01-26` - `80.7`

There will be some code refactoring into newer packages which may prevent exact
comparisons, but the updated packages will aim to provide equivalent coverage.

The following packages will also see minor changes:
- `k8s.io/kubernetes/pkg/controller/endpointslice`: `2024-01-26` - `61.4`
- `k8s.io/kubernetes/pkg/proxy`: `2024-01-26` - `69.5`

##### Integration tests

* Verify that if both the annotation `service.kubernetes.io/topology-mode: Auto`
  and field `trafficDistribution: PreferClose` are configured, precedence is given to
  the annotation.

##### e2e tests

* Verify that EndpointSlice hints are correctly populated when
  `trafficDistribution` is set to `PreferClose`.
* Verify through probes that for `trafficDistribution: PreferClose`, requests originating
  from a zone which has service pods get sent to a pod in the same zone. For
  requests originating from zones with no service pods, requests should not get
  blackholed and should rather be forwarded to any service pod from the cluster.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature gate
- Initial e2e tests completed and enabled

### Upgrade / Downgrade Strategy

No special considerations are required for upgrade / downgrade:

Upon upgrade of the EndpointSlice controller (within kube-controller-manager)
and kube-proxy:
* If a Service had both annotation `service.kubernetes.io/topology-mode` and
  field `trafficDistribution`, then the annotation will take precedence and the
  field will be ignored. 
* If a Service only had the annotation `service.kubernetes.io/topology-mode`,
  then field `trafficDistribution` was absent, then behaviour will be the same
  as above with the annotation taking precedence.

Upon downgrade of EndpointSlice controller (within kube-controller-manager) and
kube-proxy, the `trafficDistribution` will simply not be considered in any
decisions.

### Version Skew Strategy

Version skews should naturally get handled as per the following behaviour.

* **kube-apiserver:** [Kubernetes Version Skew
  Policies](https://kubernetes.io/releases/version-skew-policy/#supported-version-skew)
  require that kube-apiserver is at least at the version of kube-proxy or
  kube-controller-manager. The only valid version skew would mean that a newer
  kube-apiserver serves the new `trafficDistribution` field but the older
  kube-proxy and kube-controller-manager would silently ignore this field. (No
  adverse affect, behaviour equivalent to feature being disabled).

* **New kube-controller-manager (EndpointSlice controller) / Old kube-proxy:**

  1. **Both `service.kubernetes.io/topology-mode` and `trafficDistribution` are
     set:** For EndpointSlice controller, since the annotation takes precedence
     it will have the same behaviour as the old version, which will naturally
     work with the old kube-proxy.

  2. **Only `trafficDistribution` is set:** EndpointSlice controller configure
     EndpointSlice hints for the new routing preference, but kube-proxy still
     sees that `service.kubernetes.io/topology-mode` is unset (i.e. disabled)
     hence ignores the hints.

  3. **Only `service.kubernetes.io/topology-mode` is set:** Same as scenario 1.

* **Old kube-controller-manager (EndpointSlice controller) / New kube-proxy:**

  1. **Both `service.kubernetes.io/topology-mode` and `trafficDistribution` are
     set:** Old EndpointSlice controller programs hints as per
     `service.kubernetes.io/topology-mode`. kube-proxy sees that the
     `trafficDistribution` is set to takes the hints into account (it's not a
     problem that the hints were programmed according to the annotation)

  2. **Only `trafficDistribution` is set:** EndpointSlice controller does NOT
     configure EndpointSlice hints for the new routing preference. kube-proxy
     recognizes the new routing preference and checks for any hints. Since no
     hints are set, it configures routes as it would without any hints.

  3. **Only `service.kubernetes.io/topology-mode` is set:** Same as scenario 1,
     because if `trafficDistribution` is not set, the annotation
     `service.kubernetes.io/topology-mode` will take precedence.

## Possible future expansions

Based on user feedback, we **may** consider adding support for the following in
future iterations.

### Status Reporting

To provide clear status updates about the routing preferences to the user, the
EndpointSlice controller (which is acting as the control plane) will update the
Service status with the following conditions (inspired by Gateway API conditions)

* TrafficDistributionAccepted
  * **Type:** `TrafficDistributionAccepted` 
  * **Description:** 
    * Indicates whether a control plane component recognized and successfully
      parsed the `trafficDistribution` value. A `False` status typically suggests
      a configuration issue (e.g., an unsupported or malformed preference).
    * The EndpointSlice controller will set this status to `True` if it
      recognizes the `trafficDistribution` value as one it explicitly supports. 

* TrafficDistributionProgrammed
  * **Type:**  `TrafficDistributionProgrammed`
  * **Description:** 
    * This condition indicates whether they `trafficDistribution` has generated
      some configuration that is assumed to be ready soon in the underlying data
      plane.
    * The EndpointSlice controller will set this status to `True` if it
      successfully populated the EndpointSlice hints based on the
      `trafficDistribution`.

Note that the EndpointSlice controller and kube-proxy implementation does not
currently provide a condition denoting acknowledgment from the dataplane. In the
future when this is possible, another condition like the following could be
used:

* TrafficDistributionHonored
  * **Type:** `TrafficDistributionHonored`
  * **Description:** Confirms that the dataplane has received the hints,
    acknowledged them, and configured itself accordingly.

#### Condition usage by other implementations

Other implementations supporting `trafficDistribution` **should** adopt a domain
prefixing strategy for their condition types. This means prefixing condition
types with a domain string (e.g., `my.domain.io/TrafficDistributionAccepted`) to
prevent conflicts when multiple control planes (like the default EndpointSlice
controller) are present.

### Implementation specific heuristics

Implementations may support additional routing heuristics using values of the
form `<domain>/<heuristicName>`. Heuristics without a domain prefix will be
reserved for potential future standardization.

This can enable supporting the following user story:

* **Requirement:** I have some other precise preferences for how traffic should
  be routed, and I know that my chosen implementation supports the desired
  preference. 
* **Solution:** Set `trafficDistribution: <domain>/<heuristicName>` (where
  `<domain>` and `<heuristicName>` are provided by your implementation).
* **Effect:** The Kubernetes implementation will apply the specified routing
  heuristic. It's important to note that the precise behavior of
  implementation-specific heuristics might vary.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ServiceTrafficDistribution`
  - Components depending on the feature gate: kube-controller-manager, kube-proxy, kube-apiserver

###### Does enabling the feature change any default behavior?

No. (since we are giving precedence to the existing
`service.kubernetes.io/topology-mode` over the new `trafficDistribution` field)

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature would mean that the new field (if set) will be
ignored. Additionally, introducing the new field through the standard process of
[Adding Unstable Features to Stable Versions
](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md#adding-unstable-features-to-stable-versions)
will ensure safe rollback/disablement for kube-apiserver.

###### What happens if we reenable the feature if it was previously rolled back?

This would be equivalent to enabling the feature for the first time. Refer
[Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)

###### Are there any tests for feature enablement/disablement?

We have tests at several layers which cover how the system behaves with and
without the feature being enabled:
- kube-proxy:
  https://github.com/kubernetes/kubernetes/blob/073ce0e34bc6529d9dc81fa98a9b3fc75d90f40d/pkg/proxy/topology_test.go#L134-L162
- kube-controller-manager (EndpointSliceController):
  https://github.com/kubernetes/kubernetes/blob/073ce0e34bc6529d9dc81fa98a9b3fc75d90f40d/staging/src/k8s.io/endpointslice/reconciler_test.go#L2031-L2129

Tests which exercise the "switch" of the feature gate itself (i.e. what happens
if I disable a feature gate after having objects written with the new field) are
missing and will be added.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Partial rollouts and rollbacks which result in some version skew between
kube-apiserver, kube-controller-manager and kube-proxy should get handled as per
described in [Version Skew Strategy](#version-skew-strategy). 

Running workloads should not get affected any differently then how they would in
the absence of this feature.

###### What specific metrics should inform a rollback?

- The metric `endpoint_slice_controller_syncs` (within kube-controller-manager)
  tracks the success and failures of reconciliations performed by the
  EndpointSlice reconciler. Relative increase in the failures reported by this
  metric should serve as a signal for rollback.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Yes, testing was done using the following steps:

1. Create a v1.30.0 Kind cluster with the `ServiceTrafficDistribution` feature-gate:

```bash
kind create cluster --name=traffic-dist --config=<(cat <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
featureGates:
  ServiceTrafficDistribution: true
nodes:
- role: control-plane
  image: kindest/node:v1.30.0
- role: worker
  image: kindest/node:v1.30.0
  kubeadmConfigPatches:
  - |
    kind: JoinConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "topology.kubernetes.io/zone=zone-a"
- role: worker
  image: kindest/node:v1.30.0
  kubeadmConfigPatches:
  - |
    kind: JoinConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "topology.kubernetes.io/zone=zone-b"
EOF
)
```

2. Create a Service using the new `trafficDistribution` field.

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: demo-svc
spec:
  type: ClusterIP
  trafficDistribution: PreferClose
  ports:
  - name: tcp
    port: 80
    protocol: TCP
    targetPort: 8080
  selector:
    app: demo-app
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: demo-app
  name: demo
spec:
  replicas: 5
  selector:
    matchLabels:
      app: demo-app
  template: 
    metadata:
      labels:
        app: demo-app
    spec:
      containers:
      - name: agnhost
        image: gcr.io/kubernetes-e2e-test-images/agnhost:2.8
        args: ["serve-hostname", "--port", "8080"]
EOF
```

3. Verify that the endpointslice has the correct hints:

```bash
kubectl get endpointslice -l kubernetes.io/service-name=demo-svc -o yaml
```

4. Rollback kube-apiserver to v1.29.0

```bash
docker exec -it traffic-dist-control-plane /bin/bash

# Edit file, remove feature flag and downgrade image to v1.29.0
```

5. Verify that the endpointslice are still there but no longer have any hints:

```bash
kubectl get endpointslice -l kubernetes.io/service-name=demo-svc -o yaml
```

6. Upgrade kube-apiserver back to v1.30.0

```bash
docker exec -it traffic-dist-control-plane /bin/bash

# Edit file:
# - Add feature flag: "--feature-gates=ServiceTrafficDistribution=true"
# - Upgrade image to v1.30.0
```

7. Verify that the service has the `trafficDistribution` field visible again
   (since it persisted in etcd) and the hints are back in the EndpointSlices:

```bash
kubectl get svc demo-svc -o yaml
kubectl get endpointslice -l kubernetes.io/service-name=demo-svc -o yaml
```

8. Exec into one of the worker nodes and verify that kube-proxy has correctly
   programmed the iptable rules for the service. The rules should only contain
   endpoints which are local to that zone:

```bash
docker exec -it traffic-dist-worker /bin/bash
iptables-save
```

8. Now downgrade the kube-proxy pods to v1.29.0

```bash
# Edit:
# - DaemonSet by changing the image to v1.29.0
# - ConfigMap by removing the ServiceTrafficDistribution feature-flag.
kubectl edit -n kube-system ds/kube-proxy cm/kube-proxy
```

9. Observe the iptable rules within some worker node. This time around, the
   rules should contain all endpoints for the service.

```bash
docker exec -it traffic-dist-worker /bin/bash
iptables-save
```

10. Although kube-proxy was downgraded, the Service should still have the
    `trafficDistribution` field set and similarly the EndpointSlices should
    still have the hints.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

For implementations like kube-proxy which supported the topology annotation
(which was a beta feature), its functionality will persist and will have
precedence over the new field. The annotation will not support any new
heuristics (and only support the existing `Auto`/`Disabled` keywords)

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

A new metric `endpoint_slice_controller_services_count_by_traffic_distribution`
is exposed by kube-controller-manager which can be used to determine if some
Service is using the `trafficDistribution` field. The metric label
`traffic_distribution` can further be used to drill down on the number of
Services using some specific `trafficDistribution`.

###### How can someone using this feature know that it is working for their instance?

- [X] Events
  - A failed reconciliation in the EndpointSlice controller should be visible as
    as Event on the respective Service object.
- [ ] API .status
  - Condition name: 
  - Other field: 
- [X] Other (treat as last resort)
  - Details: A successful reonciliation by the EndpointSlice controller should
    be visible by checking the EndpointSlices and verifying that is has the
    `hints` populated. All endpoints within the EndpointSlice must have hints
    for kube-proxy to consider them (when programming routing rules).

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

It's challenging to provide exact figures without specific data, but a "normal"
quality of service should ensure the following:

- The EndpointSlice controller (control-plane) consistently and accurately
  configures EndpointSlices (including hints).
- kube-proxy (data-plane) successfully establishes routing rules based on those
  EndpointSlices.

Any significant increase in errors within these processes could indicate a
degradation in the feature's quality of service.

The following section outlines the metrics that can provide a general indication
of the quality of service for this feature.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics

  - Metric name: `endpoint_slice_controller_syncs`

    * [Optional] Aggregation method: Counter (incremented each time EndpointSliceReconciler reconciles an EndpointSlice)

    * Components exposing the metric: kube-controller-manager (or more precisely, the EndpointSlice controller)

    * Detail: The count of this metric for `success` and `failure` label values
      serves as a useful indicator

  - Metric name: `kubeproxy_sync_proxy_rules_last_timestamp_seconds`

    * [Optional] Aggregation method: Gauge (Updated on each kube-proxy sync)

    * Components exposing the metric: kube-proxy

    * Detail: An unusually old timestamp may signal a problem with kube-proxy's
      ability to update routing rules based on EndpointSlice information.
     

- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- Users using the new field will see an see an increase in the Service object of
  ~10 bytes
- Also, in situations when the status for this new field is reported, size
  equivalent to the addition of two Condition types may be observed.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Unavailability of those components implies that users will not be able to
create/update any Services. For already existing Services, they should continue
to serve traffic in accordance with the value of their `trafficDistribution`
field.

###### What are other known failure modes?

**Feature Usage Failure Modes:**

- Ensure the `trafficDistribution` field is present in the Service spec.
- Check for any events related to the Service that might indicate failures in
  the EndpointSlice controller's reconciliation process.
- Confirm that EndpointSlices are being created for the Service.
- Ensure that the EndpointSlices have the appropriate hints populated. For
  `PreferClose`, the hints should align with the endpoint zones, which can be
  verified within the EndpointSlice itself.
- Having confirmed that ALL endpoints have some hint should ensure that
  kube-proxy accepts the hints and programs the routing accordingly.
- If traffic patterns remain unexpected, examine the kube-proxy logs for any
  error messages that might indicate issues.

###### What steps should be taken if SLOs are not being met to determine the problem?

To determine the problem, the logs for the EndpointSlice controller (within
kube-controller-manager) and kube-proxy can be checked.

In terms of mitigation, there are several options:
- Remove the `trafficDistribution` field from the Services.
- Disable the `ServiceTrafficDistribution` feature flag (this should work for
  now since the feature is in beta and the feature-flag should still exist)
- Downgrade to a lower version.

## Implementation History

- First merged version of the KEP: https://github.com/kubernetes/enhancements/pull/4445
- Changes released in alpha as part of Kubernetes 1.30
- KEP updated to rename field names with the choices made during implementation.
- KEP updated with PRR sections filled, targeting beta release in Kubernetes 1.31

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

### Repurpose the existing topology annotation to recognize additional values

The historical reason for having a topology annotation instead of a field was
because the annotation just supported a single (non-disabled) value of Auto.
Given this and the fact that the behavior of the topology annotation was
supposed to be open to interpretation by the implementation, an annotation
seemed to be the right fit. The [choice of having a field](https://groups.google.com/g/kubernetes-sig-network/c/IxtD8KYsMeA) was still kept open for
the future.

Now that we plan on allowing additional values than Auto, it only feels
appropriate that we turn this into a field. A field would offer improvements
along the following lines:

* **Improved Structure and Validation:** Dedicated field allows for better type
  checking, validation, and more structured data representation within the API.
* **Clearer API Semantics:** A first-class field can make the purpose and behavior
  of the configuration more explicit. 
* The meaning of the field will also be more discoverable through spec
  documentation and commands like kubectl explain

### Reuse the fields internal/externalTrafficPolicy to offer these routing preferences

This has been a major topic of discussion in the past, with questions around
which field would be appropriate to support a heuristic like `PreferZone`. If we
were to in fact use this approach we would be faced with the dilemma of choosing
between two less-than-ideal options:

* Dilute purpose and sacrifice semantic expectation 

  * One of the primary purposes of `internalTrafficPolicy` and
    `externalTrafficPolicy` is to enforce strict traffic locality requirements
    for semantic correctness. Quoting [KEP-2086: Service Internal Traffic
    Policy](https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/2086-service-internal-traffic-policy)
    for an example of semantic incorrectness: “As a platform owner, I want to
    create a Service that always directs traffic to a logging daemon or metrics
    agent on the same node. Traffic should never bounce to a daemon on another
    node since the logs would then report an incorrect log source.”. Values like
    "Local" mandate that traffic must remain within the Node boundary. 

  * **Problem:** Introducing routing preferences like `PreferZone` would dilute
    this clear semantic meaning and could create potential misinterpretations.
    Using a separate field dedicated to routing preferences avoids this
    confusion and maintains consistency.

* Become inflexible or rigid

  * Alternatively, if we introduce `PreferZone` without diluting the meaning of
    the existing fields, we'd need to create extremely specific and inflexible
    rules for how it works across all implementations.

  * **Problem:** This would limit future innovation (like optimizing routing based
    on real-time feedback) and make it difficult to adapt to different
    infrastructure needs.

Given the above, introducing a new dedicated field seems to be better than
picking one of the two bad options.

### Granular Routing Controls

One approach to routing control would be introducing numerous configuration
fields, either directly in the Service API or within a separate, dedicated API.
This offers users maximum precision in defining routing behaviors based on
factors like location, weighted preferences, and other criteria. This approach
can be seen as a revisited, and potentially expanded, version of the
`topologyKeys` concept (and hence would suffer from some of the downsides of
`topologyKeys`, as stated previously.)

In some sense, the approach is indeed very tempting. The reason why an option
like `trafficDistribution` might be a better option is:

* Introducing numerous configuration options within the Service API (or a
  separate API type) could be sacrificing some of the core simplicity of the
  Service API. Future, more complex needs could (and should) be explored within
  the Gateway API.

* The `trafficDistribution` field elegantly balances control and abstraction.
  Users can influence behavior with high-level heuristics (like `PreferClose`) while
  implementations handle the underlying complexity. Heuristics can flag
  potential issues and guide users towards safe configurations. Using
  independent fields increases the risk of unintended consequences, as
  interactions between seemingly unrelated settings can create unexpected and
  potentially damaging routing behavior. Additionally, even simple routing
  adjustments might require tweaking multiple fields, adding complexity for the
  user.

* Rigid API contracts with granular fields can hinder an implementation's
  ability to introduce innovative routing strategies that don't fit the
  predefined mold. `trafficDistribution` encourages flexibility by treating
  preferences as hints, allowing for sophisticated, implementation-specific
  algorithms that can evolve over time.

### Reuse Pod Topology Spread Constraints for Traffic Distribution

[Pod Topology Spread
Constraints](https://kubernetes.io/docs/concepts/scheduling-eviction/topology-spread-constraints/)
offer a powerful mechanism to influence how Kubernetes schedules pods across
topology domains (like zones, nodes, or regions). Pod Topology Spread
Constraints can be re-used to guide traffic to stay within the same topology
domain as the originating pod.

Challenges:

* **Conflicting Domains:** Services often span multiple pods, which might belong to
different topology domains (e.g., a Service with pods constrained to both
node-level and zone-level Pod Topology Spread Constraints). Resolving routing
conflicts in such scenarios would require complex decision-making. 

* **Data Plane Overhead:** Informing data planes like kube-proxy of detailed Topology Spread
Constraints information for each pod could significantly increase the complexity
of communication between the control plane and data plane. This might
necessitate changes to resources like `EndpointSlices` to communicate this extra
information to the data plane (or alternatively, have the data-palne watching
all pods across all nodes, which also tends to be a bad idea.)

The potential benefits of traffic routing based solely on Topology Spread
Constraints might not outweigh the added implementation and configuration
complexity. 

A dedicated `trafficDistribution` fields gets us:

* **Clear Intent:** The `trafficDistribution` field provides an explicit way for
  users to signal desired traffic patterns, focusing solely on traffic
  distribution.

* **Implementation Flexibility:** Implementations can intelligently incorporate
  Topology Spread Constraints information (if desired) alongside other factors
  like latency, load, or custom heuristics to optimize routing decisions.

#### Complementary Use of Pod Topology Spread Constraints and trafficDistribution

Rather than having to choose between the two, Pod Topology Spread Constraints
and `trafficDistribution` can offer slightly better and resilient traffic
distribution when used in conjunction.

* Users can set `trafficDistribution` to `PreferClose` to express the preference for
  keeping traffic within the same zone as the client.
* Then, they can configure Pod Topology Spread Constraints to Ensure balanced
  pod distribution across zones, maximizing the likelihood that the
  `trafficDistribution` can be satisfied and reduce (although not completely
  eliminate) chances of overload for a single zone.
