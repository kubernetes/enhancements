<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-4444: Routing Preference for Services

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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

* **Limitations:** However, it lacked flexibility for implementations to make smart
  decisions on its own, like incorporating feedback-based routing optimizations
  (e.g., avoiding overloaded endpoints). The name `topologyKeys` might be
  perceived as overly restrictive, given the desire to incorporate non-topology
  factors into routing decisions.

### Topology Aware Routing and InternalTrafficPolicy

TopologyAwareRouting together with InternalTrafficPolicy were meant to be the
successors of `topologyKeys` and allow implementations to be more flexible.

* TopologyAwareRouting: 
  * Exposes the annotation service.kubernetes.io/topology-mode. When this
    annotation is set to Auto, an implementation specific heuristic is used to
    route the traffic. 
  * **Goal:** The aim with Auto was to allow implementations to be as smart as
    possible and choose what it considered to be the best routing criteria. The
    field didn’t support any other value by design with the thought that the
    implementation should be able to make the best possible choice for the user.
  * **Limitation:** While designed for intelligent routing, this mode offers less
    user control and can be less predictable. Some users prioritize
    predictability for performance optimization, even when accepting potential
    endpoint overload in certain situations.

* InternalTrafficPolicy 
  * The spec.internalTrafficPolicy field with the "Local" setting restricts
    traffic to endpoints on the same node as the originating Pod. 
  * **Goal:** With the deprecation of topologyKeys, one of the driving reasons was
    to satisfy the semantic use cases of strictly routing the traffic to the
    same node.
  * **Limitation:** Lacks failover; traffic is dropped if no local endpoint exists.

Note that while the initial proposal of InternalTrafficPolicy proposed a
PreferLocal policy, it was dropped later on. This meant that now
TopologyAwareRouting in conjunction with InternalTrafficPolicy didn’t exactly
allow users to express a much desired use case from topologyKeys which is
"prefer node-local, failover to same zone, then route anywhere"

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
  are not required to support all standard heuristics (e.g., PreferZone,
  ProportionalZoneCPU). Even when standard heuristics are supported, their
  precise behavior and interpretation might vary across implementations.

* **Replacement of Traffic Policies:** The new field is complementary to
  InternalTrafficPolicy and ExternalTrafficPolicy. It does not aim to substitute
  their role in enforcing strict traffic locality.

* **Immediate Support for All Possible Heuristics:** The initial implementation
  focuses on a core set of heuristics. Addition of new heuristics (like
  `PreferLocal` for Node local preference) could be explored in future
  refinements.

## Proposal

Add a new field, `routingPreference`, to the Service specification. This field
will serve as preference or hint for the underlying implementation to consider
while making routing decisions. It does not offer strict routing guarantees.

The field will support the following initial values:

* `Default`: Indicates no specific routing preference. The user delegates the
  routing decision to the implementation, allowing it to apply its best-effort
  strategy.
* `PreferEqualSpread`: Encourages an equal distribution of traffic across
  endpoints, potentially spanning multiple zones (or regions).
* `PreferZone`: Encourages routing traffic to endpoints within the same zone as
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
  by values such as `Default` or `PreferZone` might evolve over time. Such
  refinements could improve the implementation's ability to honor the original
  intent of the heuristic, even if the specific mechanisms change. For example,
  in the case of PreferZone, an implementation might initially route traffic
  within a zone with equal probability. A future improvement could introduce
  load-aware routing within the zone to further optimize performance while still
  adhering to the core principle of zonal preference. The decision of what
  constitutes an "improvement" remains at the discretion of the implementation.

### User Stories

#### Story 1

* **Requirement:** I don't have strong preferences for how my application
  traffic is routed. I prioritize simplicity and trust my Kubernetes
  implementation to optimize traffic distribution.
* **Solution:** Set `routingPreference=Default` (or leave the field unset)
* **Effect:** The Kubernetes implementation will apply its best-effort routing
  strategy based on its design. This strategy might change over time as the
  implementation evolves. It may load balance across zones or regions regions.

#### Story 2
* **Requirement:** I want my application to primarily receive traffic from
  endpoints within the same zone for performance or cost reasons. However,
  I want to avoid connection failures if no local endpoints are available.
* **Solution:** Set `routingPreference=PreferZone`
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
* **Solution:** Set `routingPreference=PreferSpread`
* **Effect:** The Kubernetes implementation will try to distribute traffic as
  equally as possible across endpoints, potentially spanning multiple zones or
  regions. This can improve resilience but might lead to increased network
  traffic costs.

#### Story 4
* **Requirement:** As a developer deploying applications across multiple
  Kubernetes environments, I want a consistent way to express my routing
  preferences. 
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

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

### Standard Heuristic Implementation (kube-proxy dataplane)

kube-proxy based dataplane (along with EndpointSlice controller, within
kube-controller-manager as the control plane) will support the three standard
routing preferences (`Default`, `PreferEqualSpread`, `PreferZone`).

#### `Default` and `PreferEqualSpread`
* Initially, kube-proxy will treat the `Default` preference the same as
  `PreferEqualSpread`
* This leverages existing implementation, requiring no major changes.

#### `PreferZone`
* This preference will be implemented by the use of Hints within EndpointSlices.
* We already use Hints to implement `service.kubernetes.io/topology-mode=Auto`
  Similarly, we’ll use the same Hints within the EndpointSlice to implement the
  PreferZone heuristic – the hints will match the zone of the endpoint itself.
* While it may seem redundant to populate the hints here since kube-proxy can
  already derive the zone hint from the endpoints zone (as they would be the
  same), we will still use this for implementation simply because of the reason
  that it’s easier to implement and provides a better design. Consider an
  alternative implementation where kube-proxy reads
  `routingPreference=PreferZone` field and then constructs the route rules
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

## Intersection with internal/externalTrafficPolicy

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

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

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

For implementations like kube-proxy which supported the topology annotation (which was a beta feature), its functionality will persist and will have precedence over the new field. The annotation will not support any new heuristics (and only support the existing `Auto`/`Disabled` keywords)

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

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

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
which field would be appropriate to support a heuristic like PreferZone. If we
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

  * **Problem:** Introducing routing preferences like "PreferZone" would dilute this
    clear semantic meaning and could create potential misinterpretations. Using
    a separate field dedicated to routing preferences avoids this confusion and
    maintains consistency.

* Become inflexible or rigid

  * Alternatively, if we introduce "PreferZone" without diluting the meaning of
    the existing fields, we'd need to create extremely specific and inflexible
    rules for how it works across all implementations.

  * **Problem:** This would limit future innovation (like optimizing routing based
    on real-time feedback) and make it difficult to adapt to different
    infrastructure needs.

Given the above, introducing a new dedicated field seems to be better than
picking one of the two bad options.


## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
