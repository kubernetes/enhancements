# KEP-2136: NetworkPolicy Versioning and Status

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1 - Testing Features of Different NetworkPolicy Implementations](#story-1---testing-features-of-different-networkpolicy-implementations)
    - [Story 2 - More Reliable NetworkPolicy Test Cases](#story-2---more-reliable-networkpolicy-test-cases)
    - [Story 3 - Understanding Whether New Features Are Supported](#story-3---understanding-whether-new-features-are-supported)
    - [Story 4 - Reporting NetworkPolicy Errors](#story-4---reporting-networkpolicy-errors)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
    - [Determining Whether a Network Plugin Implements NetworkPolicyStatus](#determining-whether-a-network-plugin-implements-networkpolicystatus)
    - [Distributed and Delegating NetworkPolicy Implementations](#distributed-and-delegating-networkpolicy-implementations)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
  - [Semantics](#semantics)
    - [The &quot;Supported&quot; Condition](#the-supported-condition)
    - [The &quot;Enforcing&quot; Condition](#the-enforcing-condition)
    - [The &quot;Problem&quot; Condition](#the-problem-condition)
    - [Other Conditions](#other-conditions)
  - [Test Plan](#test-plan)
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
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
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

Because NetworkPolicy is implemented by components outside of the
Kubernetes core, the existing versioning mechanisms in Kubernetes do
not work well for it; instead of having a two-way version skew between
"what features the user wants to use" and "what features the cluster
supports", there is a three-way split between "what features the user
wants to use", "what features the apiserver knows about", and "what
features the NetworkPolicy implementation supports". Especially, there
is often a lag between when new NetworkPolicy features are defined and
when they are widely implemented. This means that in some cases a user
cannot know if a cluster will correctly process a given NetworkPolicy
(and makes it difficult to add new NetworkPolicy features if we want
to ensure they won't be catastrophically misinterpreted by older
implementations).

```
<<[UNRESOLVED snowflake ]>>

Just how special/unusual is NetworkPolicy in this regard? Have any
other SIGs already dealt with similar problems? Or if not are there
any other APIs that are similarly failing to deal with the same
problems?

<<[/UNRESOLVED]>>
```

Somewhat related to this, there is currently no way for a
NetworkPolicy implementation (ie, the network plugin) to indicate the
status of NetworkPolicy processing to the user who created the policy.

This KEP attempts to solve both of these problems.

(Although it is not technically required that NetworkPolicy be
implemented by the network plugin, this is almost universally the case,
and so I will refer to NetworkPolicy implementations as "network
plugins" through the rest of this document.)

## Motivation

### Goals

- Provide a better mechanism for dealing with version skew with
  respect to NetworkPolicy features. Specifically:

    - Allow the apiserver to indicate to the network plugin what
      version of the NetworkPolicy featureset a particular policy
      requires (eg, whether it uses combined `podSelector` and
      `namespaceSelector`), so that network plugins can reliably
      recognize when a NetworkPolicy makes use of a feature that the
      NetworkPolicy implementation does not support.

    - Allow NetworkPolicy implementations to indicate back to the user
      when they are aware that they cannot correctly implement a given
      NetworkPolicy.

- Define rules for dealing with feature gates and alpha APIs in
  NetworkPolicy that work well with the 3-way versioning split.

- Allow network plugins to indicate when a NetworkPolicy has been fully
  "programmed" into the cluster network, to allow clients (including
  humans, scripts, and cluster components) to get more reliable behavior
  when adding new policies.

- (Maybe) Allow network plugins to provide some sort of global status
  object with certain information about the implementation.

### Non-Goals

- Deprecating (or requiring) the use of feature gates to indicate the
  stability / "certainty" of NetworkPolicy API additions; whether a
  feature is API-stable or not is distinct from whether it is widely
  implemented by plugins or not (though there is generally bidirectional
  influence between the two).

- Defining "metric-like" NetworkPolicy status information (eg, how long
  a particular rule took to implement).

## Proposal

### User Stories

#### Story 1 - Testing Features of Different NetworkPolicy Implementations

As a Kubernetes developer (who might be named Jay), I want to create a
NetworkPolicy test suite, and test the NetworkPolicy implementations of
various network plugins, but I don't want to have to manually keep track
of which plugins (and which versions of which plugins) are expected to
correctly implement which NetworkPolicy features.

#### Story 2 - More Reliable NetworkPolicy Test Cases

As a developer, I want NetworkPolicy e2e test cases that work reliably,
meaning I want the test code to be able to tell at what point a
newly-created NetworkPolicy is expected to be in effect, so that I don't
time out the test case before the network plugin has managed to process
the policy.

#### Story 3 - Understanding Whether New Features Are Supported

As a user, I want to be able to tell whether the cluster I am working
in implements certain new NetworkPolicy features, such as
[NetworkPolicy port ranges] or [selecting Namespaces by name].

[NetworkPolicy port ranges]: https://github.com/kubernetes/enhancements/pull/2090
[selecting Namespaces by name]: https://github.com/kubernetes/enhancements/pull/2113

#### Story 4 - Reporting NetworkPolicy Errors

As a network plugin developer, I want to be able to report problems with
processing a NetworkPolicy back to the user, so that they can see when
problems exist, and know how to fix them.

For example, if a policy uses [incorrect CIDR notation], I could add a
warning to the `NetworkPolicyStatus` indicating how the CIDR strings
were actually parsed.

[incorrect CIDR notation]: https://github.com/kubernetes/kubernetes/pull/94484

#### Story 5 - Changing an Alpha NetworkPolicy API

As a kubernetes developer, I want to make a change to a previously-added
alpha NetworkPolicy feature, in a way that does not cause problems for
users in clusters with varying apiserver and network plugin versions.

### Notes/Constraints/Caveats

#### Feature Gates and NetworkPolicy

Alpha APIs create an additional problem for NetworkPolicy, due to the
three-way (user/apiserver/plugin) versioning split; if an alpha field
is added in 1.21 but then changed in 1.22, then a 1.22 cluster with a
network plugin implementing the 1.21 version would probably be unable
to use either version of the feature. We might want to establish a
somewhat more restrictive set of alpha API rules for NetworkPolicy
changes than are used for other API types.

#### Determining Whether a Network Plugin Implements NetworkPolicyStatus

If a user is waiting to see the status of a newly-created NetworkPolicy,
there is no entirely-reliable way to distinguish "the plugin has not yet
set `status` but will soon" from "the plugin doesn't know about `status`
and is never going to set it".

It's not clear how big a problem this is, especially if we suggest that
implementations should create an "empty" `status` right away if it's
going to take them a while to determine the final `status`.

#### Distributed and Delegating NetworkPolicy Implementations

Some NetworkPolicy implementations do not operate from a single
controller, but instead operate independently on every node (much like
how kube-proxy implements Service rules). In this case it is not clear
who would be responsible for updating `NetworkPolicyStatus`, and it may
not be easy for any part of the system to know with certainty that a
given NetworkPolicy was fully in effect across the cluster.

In other cases, a plugin may implement NetworkPolicy in terms of some
other lower-level policy engine (implemented by a cloud, perhaps), and
may work by simply translating NetworkPolicy rules into rules for this
other engine. As in the distributed case, it may not be possible for
such a NetworkPolicy implementation to know for sure when a policy was
fully in effect.

### Risks and Mitigations

## Design Details

### API

Add a new field to `networkingv1.NetworkPolicySpec`:

```
type NetworkPolicySpec struct {
	...

	// minVersion indicates the Kubernetes release version corresponding to
	// the NetworkPolicy functionality required by this policy. If this is
	// explicitly specified by the user, the apiserver will require that the policy
	// conforms to the specified version. If it is not specified, the apiserver
	// will fill in the correct minVersion based on the features used by the policy.
	// +optional
	MinVersion NetworkPolicyVersion `json:"minVersion,omitempty" protobuf:"bytes,5,name=minVersion"`

	...
}

type NetworkPolicyVersion string

const (
	// Kubernetes 1.3 introduced the NetworkPolicy v1beta1 API, supporting
	// ingress-only policies containing podSelectors and namespaceSelectors,
	// and TCP and UDP ports.
	NetworkPolicyVersion1_3 NetworkPolicyVersion = "1.3"

	// Kubernetes 1.8 brought the NetworkPolicy API to v1, adding egress policies
	// and ipBlocks.
	NetworkPolicyVersion1_8 NetworkPolicyVersion = "1.8"

	// Kubernetes 1.9 added alpha IPv6 support. Plugins implementing this version are
	// assumed to recognize IPv6 addresses, even if they don't implement them.
	NetworkPolicyVersion1_9 NetworkPolicyVersion = "1.9"

	// Kubernetes 1.11 added support for specifying podSelector and namespaceSelector
	// together in a peer.
	NetworkPolicyVersion1_11 NetworkPolicyVersion = "1.11"

	// Kubernetes 1.12 added alpha SCTP support. Plugins implementing this version are
	// assumed to recognize `protocol: SCTP`, even if they don't implement it.
	NetworkPolicyVersion1_12 NetworkPolicyVersion = "1.12"
)

```

And add a new `Status NetworkPolicyStatus` field to `NetworkPolicy`, defined as:

```
// NetworkPolicyStatus contains information about the processing of a NetworkPolicy
type NetworkPolicyStatus struct {
	// conditions associated with the policy. These may include a "Supported"
	// condition indicating whether the features used by the policy are supported,
	// an "Enforcing" condition indicating whether the policy is actively being
	// enforced on network traffic yet, and a "Problem" condition indicating some
	// non-fatal problem with the policy.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []NetworkPolicyStatusCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,3,rep,name=conditions"`
}

// NetworkPolicyStatusConditionType is a valid value of NetworkPolicyStatusCondition.Type
type NetworkPolicyStatusConditionType string

const (
	// NetworkPolicyStatusSupported - The plugin supports this NetworkPolicy.
	NetworkPolicyStatusSupported NetworkPolicyStatusConditionType = "Supported"

	// NetworkPolicyStatusProblem - The NetworkPolicy has a problem
	NetworkPolicyStatusProblem NetworkPolicyStatusConditionType = "Problem"

	// NetworkPolicyStatusEnforcing - The plugin is currently enforcing the rules
	// specified by this NetworkPolicy
	NetworkPolicyStatusEnforcing NetworkPolicyStatusConditionType = "Enforcing"
)

// NetworkPolicyStatusCondition contains details about the state of NetworkPolicy processing
type NetworkPolicyStatusCondition struct {
	Type   NetworkPolicyStatusConditionType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=NetworkPolicyStatusConditionType"`
	Status ConditionStatus                  `json:"status" protobuf:"bytes,2,opt,name=status,casttype=ConditionStatus"`
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,4,opt,name=lastTransitionTime"`
	// Short, machine understandable string that gives the reason for condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,5,opt,name=reason"`
	// Human-readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
}
```

### Semantics

When a user creates a NetworkPolicy, and is using a network plugin that
implements this specification:

  - If the NetworkPolicy uses API fields which are not known to the
    apiserver, then the apiserver (or is it kubectl?) will reject the
    policy.

  - If the NetworkPolicy specifies an explicit `minVersion` but uses
    features newer than that version, then the apiserver will reject the
    policy as invalid. If the NetworkPolicy does not specify an explicit
    `minVersion` then the apiserver will default it based on the content
    of the policy.

  - If the NetworkPolicy is accepted by the apiserver, then it will
    eventually be seen by the network plugin:

      - If the policy's `minVersion` indicates that it uses features not
        known to the network plugin then the plugin will set appropriate
        status conditions, as described below, and then ignore the
        policy.

      - If the policy uses featuers which are _known_ to the network
        plugin but not fully _supported_ by it, the plugin will set
        appropriate status conditions, as described below, indicating
        that it may not have handled all of the policy's features in the
        way the user intended.

      - Otherwise the network plugin knows that it can correctly
        implement the policy, and does so.

#### The "Supported" Condition

The `"Supported"` condition indicates whether the plugin supports the
features used by a NetworkPolicy. Note that having a `{ type:
"Supported", status: "False" }` condition does not necessarily mean that
the policy is _completely_ unsupported. Unless there is also a `{ type:
"Enforcing", status: "False" }` condition, it is possible that the
policy may be partially-enforced.

  - If the policy's `spec.minVersion` is a version that the plugin does
    not recognize, then the plugin should set a status condition with `{
    type: "Supported", status: "False", reason: "Version" }` and an
    appropriate `message` (and likewise it should set an appropriate
    `"Enforcing"` condition indicating that it is not enforcing the
    policy).

  - If the policy has a recognized `minVersion`, but the plugin can see
    that it uses features which the plugin does not support, then the
    plugin should set a status condition with `{ type: "Supported",
    status: "False", reason: "Unimplemented" }` and an appropriate
    `message`. It is up to the plugin's discretion whether it should
    also set a `"False"` `"Enforcing"` status like above, or if it
    instead chooses to implement as much of the NetworkPolicy as it can.

  - If a policy has a recognized `minVersion` and uses only supported
    features, the plugin _may_ set a status condition with `{ type:
    "Supported", status: "True" }`. However, this can also be assumed as
    the default value if there is no `"Supported"` condition.

Note the distinction between the first two cases; if the `minVersion` is
higher than the highest version the plugin knows about, then the plugin
has no way of telling whether it can implement the policy or not, and no
way of knowing for sure if it's correctly interpreting even the parts
that it can see. (For example, if a network plugin that only implemented
version "1.9" received a policy with a combined `podSelector` and
`namespaceSelector`, it might misinterpret it by reading the
`namespaceSelector` but not the `podSelector`, and thereby allowing
access to/from more pods than was intended.) Thus, it is best for it to
not even try.

On the other hand, if the plugin knows about a feature but doesn't
implement it, they may still be able to implement the other parts of the
policy safely. (For example, a plugin that does not implement egress
policies at all might choose to still implement the ingress side of a
mixed ingress/egress policy.)

#### The "Enforcing" Condition

The `"Enforcing"` condition indicates whether the plugin is currently
enforcing the policy described by a NetworkPolicy.

- If the plugin is intentionally not enforcing the policy (eg, because
  it depends on unsupported features), it should set a `{ type: "Enforcing",
  status: "False" }` condition, with an appropriate `message` and the
  same `reason` as the `"Supported"` condition.

- A plugin which is implementing a policy but is not able to determine
  when the policy is fully in effect should set a status condition with
  `{ type: "Enforcing", status: "Unknown" }` and an appropriate `reason`
  and `message`. This indicates to the user that the enforcing status is
  not known and is not going to become known.

- A plugin which is able to determine when a NetworkPolicy is fully in
  effect _may_ set an initial status condition with `{ type:
  "Enforcing", status: "False", reason: "Pending" }`, and an appropriate
  `message`. In particular, it should do this if it thinks there may be
  a noticeable delay before it is able to set the condition to `"True"`.

- A plugin which is able to determine when a NetworkPolicy is fully in
  effect should set a status condition with `{ type: "Enforcing",
  status: "True" }`, and an appropriate `reason` and `message`, once it
  knows the NetworkPolicy is fully in effect.

  "Fully in effect" means that for any pod which ought to be subject
  to the policy, either (a) the pod was subject to the policy at the time
  when the `"Enforcing"` condition became `"True"`, or (b) the pod was not
  yet "Ready" at the time the `"Enforcing"` condition became `"True"`,
  but will not become "Ready" until it is subject to the policy.

```
<<[UNRESOLVED pod-readiness ]>>

I kind of just required that plugins that implement "Enforcing" also
properly set up pod ReadinessConditions for NetworkPolicy...

<<[/UNRESOLVED]>>
```

#### The "Problem" Condition

A `"Problem"` condition with a status of `"True"` indicates some problem
with the policy that the network plugin wants to report, which is not
covered by any of the other conditions. eg:

```
conditions:
  - type: Problem
    status: True
    reason: AmbiguousCIDR
    message: "Interpreting 192.168.1.5/24 as 192.168.1.0/24 rather than 192.168.1.5/32"
```

#### Other Conditions

```
<<[UNRESOLVED matching-conditions ]>>

One of the requests that came up in the NP WG was to be able to tell
whether a NetworkPolicy matches anything. We could have a "TargetMatch"
condition, to allow the plugin to explicitly confirm that there is/is
not at least one pod matched by the policy's `spec.podSelector`, and a
"TrafficMatch" condition, to confirm that there is/is not at least one
source/destination pod or IP range matched by the `ingress`/`egress`
sections.

(Providing more fine-grained matching info (eg, the exact pods being
matched, or even the exact number of pods matched) would presumably
require too many updates to the NetworkPolicyStatus and would be better
implemented in some other way.)

<<[/UNRESOLVED]>>
```

### Test Plan


<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy. Anything
that would count as tricky in the implementation, and anything particularly
challenging to test, should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Removing a Deprecated Flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include 
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
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
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness/README.md.

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

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name:
    - Components depending on the feature gate:
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**
  Any change of default behavior may be surprising to users or break existing
  automations, so be extremely careful here.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Also set `disable-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g., if this is a runtime
  feature, can it break the existing applications?).

* **What happens if we reenable the feature if it was previously rolled back?**

* **Are there any tests for feature enablement/disablement?**
  The e2e framework does not currently support enabling or disabling feature
  gates. However, unit tests in each component dealing with managing data, created
  with and without the feature, are necessary. At the very least, think about
  conversion tests if API types are being modified.

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

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At a high level, this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide comprehensive guidance, but at the very
  high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
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


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
  focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)

* **Will enabling / using this feature result in introducing new API types?**
  Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
  Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data sent and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits].

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**
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

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

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




If `spec.minVersion` is set on a policy, apiserver validation will
enforce that it is one of the known versions, rejecting the object if
not. 

  - The apiserver will also enforce that the policy does not use any
    features not supported by its `minVersion`. Eg, if `minVersion` is
    `1.3`, then egress policies would not be allowed.

  - We could also make `minVersion` be _required_ for all future
    features (ie, features added after this KEP).

