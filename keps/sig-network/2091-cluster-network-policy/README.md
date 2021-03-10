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

# KEP-2091: Add support for ClusterNetworkPolicy resources

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

Introduce new set of APIs to express an administrator's intent in securing their K8s cluster.
This doc proposes two new set of resources, ClusterNetworkPolicy API and the DefaultNetworkPolicy API
to complement the developer focused NetworkPolicy API in Kubernetes.

## Motivation

Kubernetes provides NetworkPolicy resources to control traffic within a cluster.
NetworkPolicies focus on expressing a developers intent to secure their applications.
Thus, in order to satisfy the needs of a security admin, we propose to introduce new set of APIs
that capture the administrators intent.

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### Goals

The goals for this KEP are to satisfy two key user stories:
1. As a cluster administrator, I want to enforce irrevocable guardrails that all workloads must adhere to in order to guarantee the safety of my clusters.
2. As a cluster administrator, I want to deploy a default set of policies to all workloads that may be overridden by the developers if needed.

There are several unique properties that we need to add in order accomplish the user stories above.
1. Deny rules and, therefore, hierarchical enforcement of policy
2. Semantics for a cluster-scoped policy object that may include namespaces/workloads that have not been created yet.
3. Backwards compatibility with existing Kubernetes Network Policy API

### Non-Goals

Our mission is to solve the most common use cases that cluster admins have. That is, we don't want to solve for every possible policy permutation a user can think of. Instead, we want to design an API that addresses 90-95% use cases while keeping the mental model easy to understand and use.

Additionally, this proposal is squarely focused on solving the needs of the Cluster Administrator. It is not intended to solve:
1. Logging / error reporting for Network Policy
2. Kubernetes Node policies
3. New policy selectors (services, service accounts, etc.)

## Proposal

In order to achieve the two primary broad use cases for a cluster admin to secure K8s clusters,
we propose to introduce the following two new resources under `networking.k8s.io` API group:
- ClusterNetworkPolicy resource
- DefaultNetworkPolicy resource

### ClusterNetworkPolicy resource

A ClusterNetworkPolicy resource will help the administrators set strict security rules for the cluster,
i.e. a developer CANNOT override these rules by creating NetworkPolicies that applies to the same workloads
as the ClusterNetworkPolicy does.

Unlike the NetworkPolicy resource in which each rule represents a whitelisted traffic, ClusterNetworkPolicy
will enable administrators to set `Allow` or `Deny` as the action of each rule. ClusterNetworkPolicy rules
should be read as-is, i.e. there will not be any implicit isolation effects for the Pods selected by the
ClusterNetworkPolicy, as opposed to what NetworkPolicy rules imply.

In terms of precedence, the aggregated `Deny` rules (all ClusterNetworkPolicy rules with action `Deny` in
the cluster combined) should be evaluated before aggregated ClusterNetworkPolicy `Allow` rules, followed
by aggregated NetworkPolicy rules in all Namespaces. As such, the `Deny` rules have the highest precedence,
which prevents them to be unexpectedly overwritten.

ClusterNetworkPolicy `Deny` rules are useful for administrators to explicitly block traffic from malicious
clients, or workloads that poses security risks. Those traffic restrictions can only be lifted once the
`Deny` rules are deleted or modified. On the other hand, the `Allow` rules can be used to call out traffic
in the cluster that needs to be allowed for certain components to work as expected (egress to CoreDNS for
example). Those traffic cannot be blocked when developers apply NetworkPolicy to their Namespaces which
isolates the workloads.

### DefaultNetworkPolicy resource

A DefaultNetworkPolicy resource will help the administrators set baseline security rules for the cluster,
i.e. a developer CAN override these rules by creating NetworkPolicies that applies to the same workloads
as the DefaultNetworkPolicy does.

DefaultNetworkPolicy works just like NetworkPolicy except that it is cluster-scoped. When workloads are
selected by a DefaultNetworkPolicy, they are isolated except for the ingress/egress rules whitelisted.
DefaultNetworkPolicy rules will not have actions associated -- each rule will be an 'allow' rule.

Aggregated NetworkPolicy rules will be evaluated before aggregated DefaultNetworkPolicy rules.
If a Pod is selected by both a DefaultNetworkPolicy and a NetworkPolicy, then the DefaultNetworkPolicy's
effect on that Pod becomes obsolete. The traffic allowed will be solely determined by the NetworkPolicy.

### Precedence model
(TODO: Add a diagram to explain the precedence)

Consider the following scenario:

- Pods [a, b, c, d] exist in Namespace x. Another Pod `client` exist in Namespace y. Namespace z has no Pods.
The following policy resources also exist in the cluster:
- (1) A ClusterNetworkPolicy `Deny` rule selects [x/a] and denies all ingress traffic from Namespace y.
- (2) A ClusterNetworkPolicy `Allow` rule selects [x/a, x/b] and allows all ingress traffic Namespace y.
- (3) A NetworkPolicy rule isolates [x/b], only allows ingress traffic from Namespace z.
- (4) A NetworkPolicy rule isolates [x/c], only allows ingress traffic from Namespace y.
- (5) A DefaultNetworkPolicy rule isolates [x/c, x/d], only allows ingress traffic from Namespace z.

Now suppose Pod y/client initiates traffic towards x/a, x/b, x/c and x/d.
- y/client -> x/a is affected by rule (1) and (2). Since rule (1) has higher precedence, the request should be denied.
- y/client -> x/b is affected by rule (2) and (3). Since rule (2) has higher precedence, the request should be allowed.
- y/client -> x/c is affected by rule (4) and (5). Since rule (4) has higher precedence, the request should be allowed.
- y/client -> x/d is affected by rule (5) only, The request should be denied.

### User Stories (Optional)

TODO: insert image

#### Story 1: Deny traffic from certain sources

As a cluster admin, I want to explicitly deny traffic from certain source IPs that I know to be bad.

Many admins maintain lists of IPs that are known to be bad actors, especially to curb DoS attacks.
A cluster admin could use ClusterNetworkPolicy to codify all the source IPs that should be denied in order to prevent
that traffic from accidentally reaching workloads. Note that the inverse of this (allow traffic from well known source
IPs) is also a valid use case.

#### Story 2: Funnel traffic through ingress/egress gateways

As a cluster admin, I want to ensure that all traffic coming into (going out of) my cluster always goes through my
ingress (egress) gateway.

It is common practice in enterprises to setup checkpoints in their clusters at ingress/egress.
These checkpoints usually perform advanced checks such as firewalling, authentication, packet/connection logging, etc.
This is a big request for compliance reasons, and ClusterNetworkPolicy can ensure that all the traffic is forced to go
through ingress/egress gateways.

#### Story 3: Isolate multiple tenants in a cluster

As a cluster admin, I want to isolate all the tenants (modeled as Namespaces) on my cluster from each other by default.

Many enterprises are creating shared Kubernetes clusters that are managed by a centralized platform team. Each internal
team that wants to run their workloads gets assigned a Namespace on the shared clusters. Naturally, the platform team
will want to make sure that, by default, all intra-namespace traffic is allowed and all inter-namespace traffic is denied.

#### Story 4: Enforce network/security best practices

As a cluster admin, I want all workloads to start with a baseline network/security model that meets the needs of my company.

A platform admin may want to factor out policies that each namespace would have to write individually in order to make
deployment and auditability easier. Common examples include allowing all workloads to be able to talk to the cluster DNS
service and, similarly, allowing all workloads to talk to the logging/monitoring pods running on the cluster.

#### Story 5: Restrict egress to well known destinations

As a cluster admin, I want to explicitly limit which workloads can connect to well known destinations outside the cluster.

This user story is particularly relevant in hybrid environments where customers have highly restricted databases running
behind static IPs in their networks and want to ensure that only a given set of workloads is allowed to connect to the
database for PII/privacy reasons. Using ClusterNetworkPolicy, a user can write a policy to guarantee that only the selected
pods can connect to the database IP.


### Notes/Constraints/Caveats (Optional)

It is important to note that the controller implementation for cluster-scoped policy APIs will
not be provided as part of this KEP. Such controllers which realize the intent of these APIs
will be provided by individual CNI providers, as is the case with the NetworkPolicy API.

### Risks and Mitigations

A potential risk of the ClusterNetworkPolicy resource is, when it's stacked on top of existing
NetworkPolicies in the cluster, some existing allowed traffic patterns (which were regulated by
those NetworkPolicies) may become blocked by ClusterNetworkPolicy `Deny` rules, while some isolated
workloads may become accessible instead because of ClusterNetworkPolicy `Allow` rules.

Developers could face some difficulties figuring out why the NetworkPolicies did not take effect,
even if they know to look for ClusterNetworkPolicy rules that can potentially override these policies:
To understand why traffic between a pair of Pods is allowed/denied, a list of NetworkPolicy resources
in both Pods' Namespace used to be sufficient (considering no other CRDs in the cluster tries to alter
traffic behavior). The same Pods, on the other hand, can appear as an AppliedTo, or an ingress/egress
peer in any ClusterNetworkPolicy. This makes looking up policies that affect a particular Pod more
challenging than when there's only NetworkPolicy resources.

In addition, in an extreme case where a ClusterNetworkPolicy `Deny` rule, ClusterNetworkPolicy `Allow`
rule, NetworkPolicy rule and DefaultNetworkPolicy rule applies to an overlapping set of Pods, users
will need to refer to the precedence model mentioned in the [previous section](precedence-model) to
determine which rule would take effect. As shown in that section, figuring out how stacked policies
affect traffic between workloads might not be very straightfoward.

To mitigate this risk and improve UX, A tool which reversely looks up affecting policies for a given
Pod and prints out relative precedence of those rules can be quite useful.
(TODO: link to NP explainer/future work?)

### Future work

Although the scope of the cluster-scoped policies is wide, the above proposal intends to only
solve the use cases documented in this KEP. However, we would also like to consider the following
set of proposals as future work items:
- **Logging**: Very often cluster administrators want to log every connection that is either denied
  or allowed by a firewall rule and send the details to an IDS or any custom tool for further
  processing of that information. With the introduction of `deny` rules, it may make sense to
  incorporate the cluster-scoped policy resources with a new field, say `loggingPolicy`, to
  determine whether a connection matching a particular rule/policy must be logged or not.
- **Rule identifier**: In order to collect traffic statistics corresponding to a rule, it is
  necessary to identify the rule which allows/denies that traffic. This helps administrators
  figure the impact of the rules written in a cluster-scoped policy resource. Thus, the ability
  to uniquely identify a rule within a cluster-scoped policy resource becomes very important.
  This can be addressed by introducing a field, say `name`, per `ClusterNetworkPolicy` and
  `DefaultNetworkPolicy` ingress/egress rule.
- **Node Selector**:  Cluster administrators and developers want to write policies that apply to
  cluster nodes or host network pods. This can be addressed by introducing nodeSelector field under
  `appliedTo` field of the `ClusterNetworkPolicy` and `DefaultNetworkPolicy` spec. `DefaultNetworkPolicy`
  is a better candidate compared to K8s `NetworkPolicy` for introducing this field as nodes are
  cluster level resources.

## Design Details

The following new `ClusterNetworkPolicy` API will be added to the `networking.k8s.io` API group:

```golang
type ClusterNetworkPolicy struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec ClusterNetworkPolicySpec
}

type ClusterNetworkPolicySpec struct {
	// No implicit isolation of AppliedTo Pods.
	AppliedTo    AppliedTo
	Ingress      []ClusterNetworkPolicyIngressRule
	Egress       []ClusterNetworkPolicyEgressRule
}

type ClusterNetworkPolicyIngress/EgressRule struct {
	Action       RuleAction
	Ports        []networkingv1.NetworkPolicyPort
	From/To      []networkingv1.ClusterNetworkPolicyPeer
}

type ClusterNetworkPolicyPeer struct {
	PodSelector  *metav1.LabelSelector
	Namespaces   *networkingv1.Namespaces
	IPBlock      *IPBlock
}

const (
	RuleActionDeny  RuleAction = "Deny"
	RuleActionAllow RuleAction = "Allow"
)
```

The following new `DefaultNetworkPolicy` API will be added to the `networking.k8s.io` API group:

```golang
type DefaultNetworkPolicy struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec DefaultNetworkPolicySpec
}

type DefaultNetworkPolicySpec struct {
	// Implicit isolation of AppliedTo Pods.
	AppliedTo   AppliedTo
	Ingress     []DefaultNetworkPolicyIngressRule
	Egress      []DefaultNetworkPolicyEgressRule
}

type DefaultNetworkPolicyIngress/EgressRule struct {
	Ports            []networkingv1.NetworkPolicyPort
	OnlyFrom/OnlyTo  []networkingv1.DefaultNetworkPolicyPeer
}

type DefaultNetworkPolicyPeer struct {
	PodSelector  *metav1.LabelSelector
	Namespaces   *networkingv1.Namespaces
	IPBlock      *IPBlock
}
```
Most structs above are very similar to NetworkPolicy and quite self-explanatory. One detail to
notice is that in the DefaultNetworkPolicy Ingress/Egress rule spec, the peers are created in a
field named `OnlyFrom`/`OnlyTo`, as opposed to `To`/`From` in ClusterNetworkPolicy. We chose this
naming to better hint policy writers about the isolation effect of DefaultNetworkPolicy on the
workloads it applies to.

The following structs will be added to the `networking.k8s.io` API group and shared between
`ClusterNetworkPolicy` and `DefaultNetworkPolicy`:

```golang
type AppliedTo struct {
	PodSelector         *metav1.LabelSelector
	NamespaceSelector   *metav1.LabelSelector
}

type Namespaces struct {
	Self       bool
	Selector   *metav1.LabelSelector
	Except     []*metav1.LabelSelector
}
```

### AppliedTo
The `AppliedTo` field replaces `PodSelector` in NetworkPolicy spec, as means to specify the target
Pods that this cluster-scoped policy (either `ClusterNetworkPolicy` or `DefaultNetworkPolicy`)
applies to. Since the policy is cluster-scoped, the `NamespaceSelector` field is required. An empty
`NamespaceSelector` (namespaceSelector: {}) selects all Namespaces in the Cluster.

### Namespaces
The `Namespaces` field replaces `NamespaceSelector` in NetworkPolicyPeer, as means to specify the
Namespaces of ingress/egress peers for cluster-scoped policies. For selecting Pods from specific
Namespaces, the `Selector` field under `Namespaces` works exactly as `NamespaceSelector`. The two
other fields `Self` and `Except` are added to satisfy the specific needs for cluster-scoped policies:

__Self:__ An optional field, which evaluates to false by default. When `self: true` is set, no selectors
or `Except` can be present concurrently. This is a special keyword to indicate that the rule only applies
to the Namespace for which the ingress/egress rule is currently being evaluated upon. Since the Pods
selected by the ClusterNetworkPolicy appliedTo could be from multiple Namespaces, the scope of
ingress/egress rules whose `namespace.self=true` will be the Pod's own Namespace for each selected Pod.
Consider the following exmaple:

- Pods [a1, b1] exist in Namespace x, which has labels `app=a` and `app=b` respectively.
- Pods [a2, b2] exist in Namespace y, which also has labels `app=a` and `app=b` respectively.

```yaml
apiVersion: networking.k8s.io/v1alpha1
kind: DefaultNetworkPolicy
spec:
  appliedTo:
    namespaceSelector: {}
  ingress:
    - onlyFrom:
      - namespaces:
          self: true
        podSelector:
          matchLabels:
            app: b
```
The above DefaultNetworkPolicy should be interpreted as: for each Namespace in the cluster, all
Pods in that Namespace should only allow traffic from Pods in the _same Namespace_ who has label
app=b. Hence, the policy above allows x/b1 -> x/a1 and y/b2 -> y/a2, but denies y/b2 -> x/a1 and
x/b1 -> y/a2.

__Except:__ An optional list, which needs to be accompanied by `Namespaces.selector` if specified.
It provides policy writers with the ability to add exclusions to the Namespaces selected. This is
especially useful in policies that, for example, intend to deny ingress from everywhere except a
few specific Namespaces, such as `kube-system`.


### Sample Specs for User Stories

#### Story 1: Deny traffic from certain sources
```yaml
apiVersion: networking.k8s.io/v1alpha1
kind: ClusterNetworkPolicy
metadata:
  name: deny-bad-ip
spec:
  appliedTo:
    # if there's an ingress gateway in the cluster, applying the policy to gateway namespace will be sufficient
    namespaceSelector: {}
  ingress:
    - action: Deny
      from:
      - ipBlock:
          cidr: 62.210.0.0/16  # blacklisted addresses
```

#### Story 2: Funnel traffic through ingress/egress gateways
```yaml
apiVersion: networking.k8s.io/v1alpha1
kind: ClusterNetworkPolicy
metadata:
  name: ingress-egress-gateway
spec:
  appliedTo:
    namespaceSelector:
      matchLabels:
        type: tenant  # assuming all tenant namespaces will be created with this label
  ingress:
    - action: Deny
      from:
      - namespaces:
          selector: {}
          except:
          - matchLabels:
              k8s.metadata.io/name: dmz  # ingress gateway
  egress:
    - action: Deny
      to:
      - namespaces:
          selector: {}
          except:
          - matchLabels:
              k8s.metadata.io/name: istio-egress  # egress gateway
```
__Note:__ The above policy is very restrictive, i.e. it rejects ingress/egress traffic between tenant
Namespaces and `kube-system`. For `core-dns` etc. to work, `kube-system` needs to be added into the
Deny `except` list.

#### Story 3: Isolate multiple tenants in a cluster
```yaml
apiVersion: networking.k8s.io/v1alpha1
kind: DefaultNetworkPolicy
metadata:
  name: namespace-isolation
spec:
  appliedTo:
    namespaceSelector:
      matchLabels:
        type: tenant  # assuming all tenant namespaces will be created with this label
  ingress:
    - onlyFrom:
      - namespaces:
          self: true
```
__Note:__ The above policy will take no effect if applied together with `ingress-egress-gateway`, since
both policies apply to the same Namespaces, and ClusterNetworkPolicy rules have higher precedence than
DefaultNetworkPolicy rules.

#### Story 4: Enforce network/security best practices
```yaml
apiVersion: networking.k8s.io/v1alpha1
kind: ClusterNetworkPolicy
spec:
  appliedTo:
    namespaceSelector:
      matchLabels:
        type: tenant  # assuming all tenant namespaces will be created with this label
  ingress:
    - action: Allow
      from:
      - namespaces:
          selector:
            matchLabels:
                app: system  # which can include kube-system and logging/monitoring namespaces
```
__Note:__ The above policy only ensures that traffic from `app=system` Namespaces will not be blocked,
if developers create NetworkPolicy which isolates the Pods in tenant Namespaces. When there's a
ClusterNetworkPolicy like `ingress-egress-gateway` present in the cluster, the above policy will be
overriden as `Deny` rules have higher precedence than `Allow` rules. In that case, the `app=system`
Namespaces need to be added to the Deny `except` list of `ingress-egress-gateway`.

#### Story 5: Restrict egress to well known destinations
```yaml
apiVersion: networking.k8s.io/v1alpha1
kind: ClusterNetworkPolicy
metadata:
  name: restrict-egress-to-db
spec:
  appliedTo:
    namespaceSelector: {}
    podSelector:
      matchExpressions:
        - {key: app, operator: NotIn, values: [authorized-client]}
  egress:
    - action: Deny
      to:
      - ipBlock:
          cidr: 10.220.0.8/32  # restricted database running behind static IP
```

### Test Plan

- Add e2e tests for ClusterNetworkPolicy resource
  - Ensure `deny` rules override all allowed traffic in the cluster
  - Ensure `allow` rules override K8s NetworkPolicies
  - Ensure that in stacked ClusterNetworkPolicies/K8s NetworkPolicies, the following precedence is maintained

    aggregated `deny` rules > aggregated `allow` rules > K8s NetworkPolicy rules
- Add e2e tests for DefaultNetworkPolicy resource
  - Ensure that in absence of ClusterNetworkPolicy rules and K8s NetworkPolicy rules, DefaultNetworkPolicy rules are observed
  - Ensure that K8s NetworkPolicies override DefaultNetworkPolicies by applying policies to the same workloads
  - Ensure that stacked DefaultNetworkPolicies are additive in nature
- e2e test cases must cover ingress and egress rules
- e2e test cases must cover port-ranges, named ports, integer ports etc
- e2e test cases must cover various combinations of `podSelector` in `appliedTo` and ingress/egress rules
- e2e test cases must cover various combinations of `namespaceSelector` in `appliedTo`
- e2e test cases must cover various combinations of `namespaces` in ingress/egress rules
  - Ensure that `except` field works as expected
  - Ensure that `self` field works as expected
- Add unit tests to test the validation logic which shall be introduced for cluster-scoped policy resources
  - Ensure that `self` field cannot be set along with `selector` within `namespaces`
  - Test cases for fields which are shared with NetworkPolicy, like `ipBlock`, `endPort` etc.
- Ensure that only administrators or assigned roles can create/update/delete cluster-scoped policy resources

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

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- At least 1 CNI provider must provide the implementation for the complete set of alpha features
- Evaluate "future work" items based on feedback from community

#### Beta -> GA Graduation

- At least 2 CNI providers must provide the implementation for the complete set of alpha features
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback
- Completion of all accepted "future work" items

#### Removing a Deprecated Flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

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
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?
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

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
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

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

<!--
At a high level, this usually will be in the form of "high percentile of SLI
per day <= X". It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code
-->

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

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

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

Following alternative approaches were considered:

### NetworkPolicy v2

A new version for NetworkPolicy, v2, was evaluated to address features and use cases
documented in this KEP. Since the NetworkPolicy resource already exists, it would be
a low barrier to entry and can be extended to incorporate admin use cases.
However, this idea was rejected because the NetworkPolicy resource was introduced
solely to satisfy a developers intent. Thus, adding new use cases for a cluster admin
would be contradictory. In addition to that, the administrator use cases are mainly
scoped to the cluster as opposed to the NetworkPolicy resource, which is `namespaced`.

### Single CRD with DefaultRules field

We evaluated the possibility of solving the administrator use cases by introducing a
single resource, similar to the proposed ClusterNetworkPolicy resource, as opposed to
the proposed two resources, ClusterNetworkPolicy and DefaultNetworkPolicy. This alternate
proposal was a hybrid approach, where in the ClusterNetworkPolicy resource (introduced
in the proposal) would include additional fields called `defaultIngress` and
`defaultEgress`. These defaultIngress/defaultEgress fields would be similar in structure to
the ingress/egress fields, except that the default rules will not have `action` field.
All default rules will be "allow" rules only, similar to K8s NetworkPolicy. Presence of
at least one `defaultIngress` rule will isolate the `appliedTo` workloads from accepting
any traffic other than that specified by the policy. Similarly, the presence of at least
one `defaultEgress` rule will isolate the `appliedTo` workloads from accessing any other
workloads other than those specified by the policy. In addition to that, the rules specified
by `defaultIngress` and `defaultEgress` fields will be evaluated to be enforced after the
K8s NetworkPolicy rules, thus such default rules can be overridden by a developer written
K8s NetworkPolicy.

Adding default rules along with the stricter ClusterNetworkPolicy rules allows us to
satisfy all admin use cases with a single resource. Although this might be appealing,
separating the two broad intents of a cluster admin in two different resources makes
the definition of each resource much cleaner and simpler.

### Single CRD with IsOverrideable field

An alternative approach is to combine `ClusterNetworkPolicy` and `DefaultNetworkPolicy`
into a single CRD with an additional overrideable field in Ingress/ Egress rule as shown below.
```golang
type ClusterNetworkPolicyIngress/EgressRule struct {
	Action        RuleAction
	IsOverridable bool
	Ports         []networkingv1.NetworkPolicyPort
	From/To       []networkingv1.ClusterNetworkPolicyPeer
}
```
If `IsOverridable` is set to false, the rules will take higher precedence than the
Kubernetes Network Policy rules. Otherwise, the rules will take lower precedence.
Note that both overridable and non overridable cluster network policy rules have explicit
allow/ deny rules. The precedence order of the rules is as follows:

`ClusterNetworkPolicy` Deny (`IsOverridable`=false) > `ClusterNetworkPolicy` Allow (`IsOverridable`=false) > K8s `NetworkPolicy` > `ClusterNetworkPolicy` Allow (`IsOverridable`=true) > `ClusterNetworkPolicy` Deny (`IsOverridable`=true)

As the semantics for overridable Cluster Network Policies are different from K8s Network Policies,
Cluster administrators who worked on K8s Network policies will have hard time writing similar policies for the cluster.
Also, modifying a single field (`IsOverridable`) of a rule will change the priority in a non-intuitive manner which
may cause some confusion. For these reasons, we decided not go with this proposal.

### Single CRD with BaselineAllow as Action

We evaluated another single CRD approach with an additional `RuleAction` to cover
use-cases of both `ClusterNetworkPolicy` and `DefaultNetworkPolicy`

In this approach, we introduce a `BaselineRuleAction` rule action.
```golang
type ClusterNetworkPolicyIngress/EgressRule struct {
	Action       RuleAction
	Ports        []networkingv1.NetworkPolicyPort
	From/To      []networkingv1.ClusterNetworkPolicyPeer
}
const (
	RuleActionDeny          RuleAction = "Deny"
	RuleActionAllow         RuleAction = "Allow"
	RuleActionBaselineAllow RuleAction = "BaselineAllow"
)
```

RuleActionDeny and RuleActionAllow are used to specify rules that take higher precedence
than Kubernetes Network Policies whereas RuleActionBaselineAllow is used to specify the
rules that take lower precedence Kubernetes Network Policies. The RuleActionBaselineAllow rules
have same semantics as Kubernetes Network Policy rules but defined at cluster level.

One of the reasons we did not go with this approach is the ambiguity of the term `BaselineAllow`.
Also, the semantics around `RuleActionBaselineAllow` is slightly different as it involves
implicit isolation compared to explicit Allow/ Deny rules with other `RuleActions`.
