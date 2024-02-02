# KEP-4444: Routing Preference for Services

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
    - [Story 4](#story-4)
    - [Story 5](#story-5)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Standard Heuristic Implementation (kube-proxy dataplane)](#standard-heuristic-implementation-kube-proxy-dataplane)
    - [<code>Default</code> and <code>Spread</code>](#default-and-spread)
    - [<code>Zone</code>](#zone)
  - [Changes within kube-proxy](#changes-within-kube-proxy)
  - [Status Reporting](#status-reporting)
    - [Condition usage by other implementations](#condition-usage-by-other-implementations)
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
    - [Complementary Use of Pod Topology Spread Constraints and routingPreference](#complementary-use-of-pod-topology-spread-constraints-and-routingpreference)
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

This KEP proposes introducing a new field, `routingPreference`, to the
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
  * Responds the annotation service.kubernetes.io/topology-mode. When this
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
`routingPreference` field, the design allows for the potential introduction of
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

* **Strict Routing Guarantees:** The `routingPreference` field will not
  enforce deterministic routing paths. It serves as a mechanism for expressing
  hints and preferences that implementations can consider when making routing
  decisions.

* **Mandatory and Uniform Implementation Support:** Kubernetes implementations
  are not required to support all standard heuristics (e.g., `Zone`, `Spread`).
  Even when standard heuristics are supported, their precise behavior and
  interpretation might vary across implementations.

* **Replacement of Traffic Policies:** The new field is complementary to
  InternalTrafficPolicy and ExternalTrafficPolicy. It does not aim to substitute
  their role in enforcing strict traffic locality.

* **Immediate Support for All Possible Heuristics:** The initial implementation
  focuses on a core set of heuristics. Addition of new heuristics (like
  `Local` for Node local preference) could be explored in future
  refinements.

## Proposal

Add a new field, `routingPreference`, to the Service specification. This field
will serve as preference or hint for the underlying implementation to consider
while making routing decisions. It does not offer strict routing guarantees.

The field will support the following initial values:

* `Default`: Indicates no specific routing preference. The user delegates the
  routing decision to the implementation, allowing it to apply its best-effort
  strategy.
* `Spread`: Encourages an equal distribution of traffic across
  endpoints, potentially spanning multiple zones (or regions).
* `Zone`: Encourages routing traffic to endpoints within the same zone as
  the client. If no endpoints are available within the zone, traffic should be
  routed to other zones.

Implementations are strongly encouraged to support the standard values. While
some flexibility in interpretation is permitted, implementations should aim to
align their behavior with the described intent of these preferences as closely
as possible.

Implementations may support additional routing heuristics using values of the
form `<domain>/<heuristicName>`. Heuristics without a domain prefix will be
reserved for potential future standardization.

NOTE: Implementations reserve the right to refine the behavior associated with
  any heuristic, including standard heuristics. This means the behavior enabled
  by values such as `Default` or `Zone` might evolve over time, and some
  evolutions might interpret the heuristic goals slightly differently. For
  example, in the case of `Zone`, an implementation might initially route
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
* **Solution:** Set `routingPreference=Default` (or leave the field unset)
* **Effect:** The Kubernetes implementation will apply its best-effort routing
  strategy based on its design. This strategy might change over time as the
  implementation evolves. It may load balance across zones or regions.

#### Story 2
* **Requirement:** I want my application to primarily receive traffic from
  endpoints within the same zone for performance or cost reasons. However,
  I want to avoid connection failures if no local endpoints are available.
* **Solution:** Set `routingPreference=Zone`
* **Effect:** The Kubernetes implementation will aim to prioritize routing
  traffic to endpoints in the same zone as the client. If no endpoints are
  available within the zone, traffic will be routed to other zones. It's
  possible that traffic patterns could lead to endpoints within the preferred
  zone becoming overloaded. Consider other routing strategies or scaling up
  resources within the zone if this becomes a concern.

#### Story 3
* **Requirement:** I prioritize application availability and want to minimize the
  risk of outages due to localized overload. I'm willing to accept potentially
  higher costs associated with cross-zone traffic distribution.
* **Solution:** Set `routingPreference=Spread`
* **Effect:** The Kubernetes implementation will try to distribute traffic as
  equally as possible across endpoints, potentially spanning multiple zones or
  regions. This can improve resilience but might lead to increased network
  traffic costs.

#### Story 4
* **Requirement:** As a developer of a widely deployed cluster-addon, I want to
  be able to provide users an easy way to configure my Helm chart and/or
  deployment configuration to enable same-zone routing behavior in a portable
  way that works across many different environments.
* **Solution:** Using one of the available standard values in the
  `routingPreference` field will allow the customer to use the same Helm Chart
  configurations with greater confidence regardless of the underlying Kubernetes
  implementation. This simplifies their deployment process and reduces the
  complexity of managing cross-cluster applications.

#### Story 5
* **Requirement:** I have some other precise preferences for how traffic should
  be routed, and I know that my chosen implementation supports the desired
  preference. 
* **Solution:** Set `routingPreference=<domain>/<heuristicName>` (where
  `<domain>` and `<heuristicName>` are provided by your implementation).
* **Effect:** The Kubernetes implementation will apply the specified routing
  heuristic. It's important to note that the precise behavior of
  implementation-specific heuristics might vary.

### Notes/Constraints/Caveats (Optional)

This proposal is our third attempt at an API revolving around such a
configuration. There's a non-zero chance that we may need to revisit this again.

### Risks and Mitigations

* **Risk:** Having a routing preference like `Zone` comes at the risk of
  endpoints in certain zones being overloaded if the originating traffic is
  skewed towards a particular zone.

  **Mitigation:**
    * Emphasize in the documentation that the `Zone` preference is
      designed for low-latency or monetory-cost reasons, with the understanding
      that it can lead to overload within zones.
    * Recommend approaches like having deployments per zone which can scale
      independently of other zones.

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
as the control plane) will support the three standard routing preferences
(`Default`, `Spread`, `Zone`).

#### `Default` and `Spread`
* Initially, kube-proxy will treat the `Default` preference the same as
  `Spread`
* This leverages existing implementation, requiring no major changes.

#### `Zone`
* This preference will be implemented by the use of Hints within EndpointSlices.
* We already use Hints to implement `service.kubernetes.io/topology-mode=Auto`
  Similarly, we’ll use the same Hints within the EndpointSlice to implement the
  `Zone` heuristic – the hints will match the zone of the endpoint itself.
* While it may seem redundant to populate the hints here since kube-proxy can
  already derive the zone hint from the endpoints zone (as they would be the
  same), we will still use this for implementation simply because of the reason
  that it’s easier to implement and provides a better design. Consider an
  alternative implementation where kube-proxy reads
  `routingPreference=Zone` field and then constructs the route rules
  accordingly. This means some extra logic needs to be baked into the kube-proxy
  which could have just as easily been implemented by an already existing
  extensibility mechanism (i.e. EndpointSlice hints)

Although this is not an explicit design goal, an implication of the above
implementation choice means that:
* The control plane (kube-controller-manager in this case) is only concerned
  with the field `routingPreference` and populates EndpointSlice hints based
  on its value 
* The data plane (kube-proxy in this case) is only concerned with the Hints
  populated in the EndpointSlice and the fields `internal/externalTrafficPolicy`
  to make routing decisions.
* Neither the control plane nor the data plane looks at the others field.

NOTE: The fact that EndpointSlice hints are not expected to be implemented by
  all data planes is not of concern here, because `routingPreference` is anyways
  supposed to be implementation dependent. Since we know that the kube-proxy
  data plane will respect the Hints, it’s sufficient for the kube-proxy and
  kube-controller-manager based implementations to say that we do support the
  standard heuristics.

### Changes within kube-proxy

**Present behaviour:** kube-proxy only considers EndpointSlice hints for route
programming if the `service.kubernetes.io/topology-aware-hints` annotation is
set to "Auto"

**New behaviour:** Irrespective of what the annotation
`service.kubernetes.io/topology-aware-hints` or field `routingPreference` are
set to (or even if they are not set at all), kube-proxy will always consider
EndpointSlice hints (assuming this feature-gate is enabled).

NOTE: The expectation remains that *all* endpoints within an EndpointSlice must
  have corresponding hints for kube-proxy to utilize them. This avoids scenarios
  with partial hints. The reason for this requirement is the same one highlighted in [KEP-2433 Topology
  Aware
  Hints](https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/2433-topology-aware-hints/README.md#kube-proxy), i.e. _"This is to provide safer transitions between enabled and disabled states. Without this fallback, endpoints could easily get overloaded as hints were being added or removed from some EndpointSlices but had not yet propagated to all of them."_

### Status Reporting

To provide clear status updates about the routing preferences to the user, the
EndpointSlice controller (which is acting as the control plane) will update the
Service status with the following conditions (inspired by Gateway API conditions)

* RoutingPreferenceAccepted
  * **Type:** `RoutingPreferenceAccepted` 
  * **Description:** 
    * Indicates whether a control plane component recognized and successfully
      parsed the `routingPreference` value. A `False` status typically suggests
      a configuration issue (e.g., an unsupported or malformed preference).
    * The EndpointSlice controller will set this status to `True` if it
      recognizes the `routingPreference` value as one it explicitly supports. 

* RoutingPreferenceProgrammed
  * **Type:**  `RoutingPreferenceProgrammed`
  * **Description:** 
    * This condition indicates whether they `routingPreference` has generated
      some configuration that is assumed to be ready soon in the underlying data
      plane.
    * The EndpointSlice controller will set this status to `True` if it
      successfully populated the EndpointSlice hints based on the
      `routingPreference`.

Note that the EndpointSlice controller and kube-proxy implementation does not
currently provide a condition denoting acknowledgment from the dataplane. In the
future when this is possible, another condition like the following could be
used:

* RoutingPreferenceReady
  * **Type:** `RoutingPreferenceReady`
  * **Description:** Confirms that the dataplane has received the hints,
    acknowledged them, and configured itself accordingly.

#### Condition usage by other implementations

Other implementations supporting `routingPreference` **should** adopt a domain
prefixing strategy for their condition types. This means prefixing condition
types with a domain string (e.g., `my.domain.io/RoutingPreferenceAccepted`) to
prevent conflicts when multiple control planes (like the default EndpointSlice
controller) are present.

<<[UNRESOLVED Name for the field is being discussed]>>

### Choice of field name
The name `routingPreference` is meant to capture the highly
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
* Use of words like “traffic” and “policy” for names like
  trafficRoutingPreference was not favored so that it’s clearly different from
  the `internalTrafficPolicy` and `externalTrafficPolicy` fields which have a
  more fixed behavior.
* Use of the word "selection" for a name like `endpointSelection` were avoided
  so as not to confuse with the actual process of selecting the complete set of
  pods backing a service.

<<[/UNRESOLVED]>>

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
    <th>routingPreference</th>
    <th>External Result</th>
    <th>Internal Result</th>
  </tr>
</thead>
<tbody>
  <tr>
    <td>-</td>
    <td>-</td>
    <td>Auto</td>
    <td>routingPreference=Auto</td>
    <td>routingPreference=Auto</td>
  </tr>
  <tr>
    <td>Local</td>
    <td>-</td>
    <td>Auto</td>
    <td>ExternalTrafficPolicy=Local</td>
    <td>routingPreference=Auto</td>
  </tr>
  <tr>
    <td>Local</td>
    <td>Local</td>
    <td>Auto</td>
    <td>ExternalTrafficPolicy=Local</td>
    <td>InternalTrafficPolicy=Local</td>
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

* Verify that if both the annotation `service.kubernetes.io/topology-mode=Auto`
  and field `routingPreference=Zone` are configured, precedence is given
  to the annotation.

##### e2e tests

* Verify that EndpointSlice hints are correctly populated when
  `routingPreference=Zone`
* Verify through probes that when `routingPreference=Zone`, requests
  originating from a zone which has service pods get sent to a pod in the same
  zone. For requests originating from zones with no service pods, requests
  should not get blackholed and should rather be forwarded to any service pod
  from the cluster.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

### Upgrade / Downgrade Strategy

No special considerations are required for upgrade / downgrade:

Upon upgrade of the EndpointSlice controller (within kube-controller-manager)
and kube-proxy:
* If a Service had both annotation `service.kubernetes.io/topology-mode` and
  field `routingPreference`, then the annotation will take precedence and the
  field will be ignored. 
* If a Service only had the annotation `service.kubernetes.io/topology-mode`,
  then field `routingPreference`'s value will be assumed to be `Default` and the
  behaviour will be the same as above with the annotation taking precedence.

Upon downgrade of EndpointSlice controller (within kube-controller-manager) and
kube-proxy, the `routingPreference` will simply not be considered in any
decisions.

### Version Skew Strategy

Version skews should naturally get handled as per the following behaviour.

* **kube-apiserver:** [Kubernetes Version Skew
  Policies](https://kubernetes.io/releases/version-skew-policy/#supported-version-skew)
  require that kube-apiserver is atleast at the version of kube-proxy or
  kube-controller-manager. The only valid version skew would mean that a newer
  kube-apiserver serves the new `routingPreference` field but the older
  kube-proxy and kube-controller-manager would silently ignore this field. (No
  adverse affect, behaviour equivalent to feature being disabled)

* **New kube-controller-manager (EndpointSlice controller) / Old kube-proxy:**

  1. **Both `service.kubernetes.io/topology-mode` and `routingPreference` are
     set:** For EndpointSlice controller, since the annotation takes precedence
     it will have the same behaviour as the old version, which will naturally
     work with the old kube-proxy.

  2. **Only `routingPreference` is set:** EndpointSlice controller configure
     EndpointSlice hints for the new routing preference, but kube-proxy still
     sees that `service.kubernetes.io/topology-mode` is unset (i.e. disabled)
     hence ignores the hints.

  3. **Only `service.kubernetes.io/topology-mode` is set:** Same as scenario 1.

* **Old kube-controller-manager (EndpointSlice controller) / New kube-proxy:**

  1. **Both `service.kubernetes.io/topology-mode` and `routingPreference` are
     set:** Old EndpointSlice controller programs hints as per
     `service.kubernetes.io/topology-mode`. kube-proxy sees that the
     `routingPreference` is set to takes the hints into account (it's not a
     problem that the hints were programmed according to the annotation)

  2. **Only `routingPreference` is set:** EndpointSlice controller does NOT
     configure EndpointSlice hints for the new routing preference. kube-proxy
     recognizes the new routing preference and checks for any hints. Since no
     hints are set, it configures routes as it would without any hints.

  3. **Only `service.kubernetes.io/topology-mode` is set:** Same as scenario 1,
     because if `routingPreference` is not set, kube-proxy would think it's set
     to the `Default` value.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ServiceRoutingPreference`
  - Components depending on the feature gate: kube-controller-manager, kube-proxy, kube-apiserver

###### Does enabling the feature change any default behavior?

No. (since we are giving precedence to the existing
`service.kubernetes.io/topology-mode` over the new `routingPreference` field)

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

Yes we will have unit tests covering the same.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

_This section will be completed when targeting beta to a release._

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

For implementations like kube-proxy which supported the topology annotation
(which was a beta feature), its functionality will persist and will have
precedence over the new field. The annotation will not support any new
heuristics (and only support the existing `Auto`/`Disabled` keywords)

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

_This section will be completed when targeting beta to a release._

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

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

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

_This section will be completed when targeting beta to a release._


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
which field would be appropriate to support a heuristic like `Zone`. If we were
to in fact use this approach we would be faced with the dilemma of choosing
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

  * **Problem:** Introducing routing preferences like `Zone` would dilute this
    clear semantic meaning and could create potential misinterpretations. Using
    a separate field dedicated to routing preferences avoids this confusion and
    maintains consistency.

* Become inflexible or rigid

  * Alternatively, if we introduce `Zone` without diluting the meaning of
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
like `routingPreference` might be a better option is:

* Introducing numerous configuration options within the Service API (or a
  separate API type) could be sacrificing some of the core simplicity of the
  Service API. Future, more complex needs could (and should) be explored within
  the Gateway API.

* The `routingPreference` field elegantly balances control and abstraction.
  Users can influence behavior with high-level heuristics (`Zone`, `Spread`)
  while implementations handle the underlying complexity. Heuristics can flag
  potential issues and guide users towards safe configurations. Using
  independent fields increases the risk of unintended consequences, as
  interactions between seemingly unrelated settings can create unexpected and
  potentially damaging routing behavior. Additionally, even simple routing
  adjustments might require tweaking multiple fields, adding complexity for the
  user.

* Rigid API contracts with granular fields can hinder an implementation's
  ability to introduce innovative routing strategies that don't fit the
  predefined mold. `routingPreference` encourages flexibility by treating
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

A dedicated `routingPreference` fields gets us:

* **Clear Intent:** The `routingPreference` field provides an explicit way for
  users to signal desired traffic patterns, focusing solely on traffic
  distribution.

* **Implementation Flexibility:** Implementations can intelligently incorporate
  Topology Spread Constraints information (if desired) alongside other factors
  like latency, load, or custom heuristics to optimize routing decisions.

#### Complementary Use of Pod Topology Spread Constraints and routingPreference

Rather than having to choose between the two, Pod Topology Spread Constraints
and `routingPreference` can offer slightly better and resilient traffic
distribution when used in conjunction.

* Users can set `routingPreference` to `Zone` to express the preference for
  keeping traffic within the same zone as the client.
* Then, they can configure Pod Topology Spread Constraints to Ensure balanced
  pod distribution across zones, maximizing the likelihood that the
  `routingPreference` can be satisfied and reduce (although not completely
  eliminate) chances of overload for a single zone.


## Infrastructure Needed (Optional)

N/A
