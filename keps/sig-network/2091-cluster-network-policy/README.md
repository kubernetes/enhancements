# KEP-2091: Add support for ClusterNetworkPolicy resources

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [ClusterNetworkPolicy resource](#clusternetworkpolicy-resource)
  - [DefaultNetworkPolicy resource](#defaultnetworkpolicy-resource)
  - [Precedence model](#precedence-model)
  - [User Stories](#user-stories)
    - [Story 1: Deny traffic from certain sources](#story-1-deny-traffic-from-certain-sources)
    - [Story 2: Funnel traffic through ingress/egress gateways](#story-2-funnel-traffic-through-ingressegress-gateways)
    - [Story 3: Isolate multiple tenants in a cluster](#story-3-isolate-multiple-tenants-in-a-cluster)
    - [Story 4: Enforce network/security best practices](#story-4-enforce-networksecurity-best-practices)
    - [Story 5: Restrict egress to well known destinations](#story-5-restrict-egress-to-well-known-destinations)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Future Work](#future-work)
- [Design Details](#design-details)
  - [ClusterNetworkPolicy API Design](#clusternetworkpolicy-api-design)
  - [Except Field Semantics](#except-field-semantics)
  - [DefaultNetworkPolicy API Design](#defaultnetworkpolicy-api-design)
  - [Shared API Design](#shared-api-design)
    - [AppliedTo](#appliedto)
    - [Namespaces](#namespaces)
    - [IPBlock](#ipblock)
  - [Sample Specs for User Stories](#sample-specs-for-user-stories)
    - [Story 1: Deny traffic from certain sources](#story-1-deny-traffic-from-certain-sources-1)
    - [Story 2: Funnel traffic through ingress/egress gateways](#story-2-funnel-traffic-through-ingressegress-gateways-1)
    - [Story 3: Isolate multiple tenants in a cluster](#story-3-isolate-multiple-tenants-in-a-cluster-1)
    - [Story 4: Enforce network/security best practices](#story-4-enforce-networksecurity-best-practices-1)
    - [Story 5: Restrict egress to well known destinations](#story-5-restrict-egress-to-well-known-destinations-1)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha to Beta Graduation](#alpha-to-beta-graduation)
    - [Beta to GA Graduation](#beta-to-ga-graduation)
    - [Removing a Deprecated Flag](#removing-a-deprecated-flag)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Upgrade considerations](#upgrade-considerations)
    - [Downgrade considerations](#downgrade-considerations)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Scalability](#scalability)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [NetworkPolicy v2](#networkpolicy-v2)
  - [Single CRD with DefaultRules field](#single-crd-with-defaultrules-field)
  - [Single CRD with IsOverrideable field](#single-crd-with-isoverrideable-field)
  - [Single CRD with BaselineAllow as Action](#single-crd-with-baselineallow-as-action)
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

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Introduce new set of APIs to express an administrator's intent in securing
their K8s cluster. This doc proposes two new set of resources,
ClusterNetworkPolicy API and the DefaultNetworkPolicy API to complement the
developer focused NetworkPolicy API in Kubernetes.

## Motivation

Kubernetes provides NetworkPolicy resources to control traffic within a
cluster. NetworkPolicies focus on expressing a developers intent to secure
their applications. Thus, in order to satisfy the needs of a security admin,
we propose to introduce new set of APIs that capture the administrators intent.

### Goals

The goals for this KEP are to satisfy two key user stories:
1. As a cluster administrator, I want to enforce irrevocable guardrails that
   all workloads must adhere to in order to guarantee the safety of my clusters.
2. As a cluster administrator, I want to deploy a default set of policies to all
   workloads that may be overridden by the developers if needed.

There are several unique properties that we need to add in order accomplish the
user stories above.
1. Deny rules and, therefore, hierarchical enforcement of policy
2. Semantics for a cluster-scoped policy object that may include
   namespaces/workloads that have not been created yet.
3. Backwards compatibility with existing Kubernetes Network Policy API

### Non-Goals

Our mission is to solve the most common use cases that cluster admins have.
That is, we don't want to solve for every possible policy permutation a user
can think of. Instead, we want to design an API that addresses 90-95% use cases
while keeping the mental model easy to understand and use.

Additionally, this proposal is squarely focused on solving the needs of the
Cluster Administrator. It is not intended to solve:
1. Logging / error reporting for Network Policy
2. Kubernetes Node policies
3. New policy selectors (services, service accounts, etc.)
4. Tools/CLI to debug/explain NetworkPolicies, cluster-scoped policies and their
   impact on workloads.

## Proposal

In order to achieve the two primary broad use cases for a cluster admin to
secure K8s clusters, we propose to introduce the following two new resources
under `networking.k8s.io` API group:
- ClusterNetworkPolicy
- DefaultNetworkPolicy

### ClusterNetworkPolicy resource

A ClusterNetworkPolicy resource will help the administrators set strict
security rules for the cluster, i.e. a developer CANNOT override these rules
by creating NetworkPolicies that applies to the same workloads as the
ClusterNetworkPolicy does.

Unlike the NetworkPolicy resource in which each rule represents a allowed
traffic, ClusterNetworkPolicy will enable administrators to set `Allow` or
`Deny` as the action of each rule. ClusterNetworkPolicy rules should be read
as-is, i.e. there will not be any implicit isolation effects for the Pods
selected by the ClusterNetworkPolicy, as opposed to what NetworkPolicy rules imply.

In terms of precedence, the aggregated `Deny` rules (all ClusterNetworkPolicy
rules with action `Deny` in the cluster combined) should be evaluated before
aggregated ClusterNetworkPolicy `Allow` rules, followed by aggregated
NetworkPolicy rules in all Namespaces. As such, the `Deny` rules have the
highest precedence, which prevents them to be unexpectedly overwritten.

ClusterNetworkPolicy `Deny` rules are useful for administrators to explicitly
block traffic from malicious clients, or workloads that poses security risks.
Those traffic restrictions can only be lifted once the`Deny` rules are deleted
or modified. On the other hand, the `Allow` rules can be used to call out
traffic in the cluster that needs to be allowed for certain components to work
as expected (egress to CoreDNS for example). Those traffic should not be blocked
when developers apply NetworkPolicy to their Namespaces which isolates the workloads.

### DefaultNetworkPolicy resource

A DefaultNetworkPolicy resource will help the administrators set baseline security
rules for the cluster, i.e. a developer CAN override these rules by creating
NetworkPolicies that applies to the same workloads as the DefaultNetworkPolicy does.

DefaultNetworkPolicy works just like NetworkPolicy except that it is cluster-scoped.
When workloads are selected by a DefaultNetworkPolicy, they are isolated except
for the ingress/egress rules specified. DefaultNetworkPolicy rules will not have
actions associated -- each rule will be an 'allow' rule.

Aggregated NetworkPolicy rules will be evaluated before aggregated DefaultNetworkPolicy rules.
If a Pod is selected by both, a DefaultNetworkPolicy and a NetworkPolicy, then
the DefaultNetworkPolicy's effect on that Pod becomes obsolete.
In this case, the traffic allowed will be solely determined by the NetworkPolicy.

### Precedence model

```

                 Yes -------> [ Drop ]            Yes -------> [ Allow ]          Yes -------> [ Allow ]
                  |                                |                               |
                  |                                |                               |
         +------------------+             +-------------------+              +------------------+
  ---->  | traffic matches  | --- No -->  | traffic matches   |  --- No -->  | traffic matches  |
         | a CNP Deny rule? |             | a CNP Allow rule? |              | a NetworkPolicy  |
         +------------------+             +-------------------+              | rule?            |
                                                                             +------------------+
                                                                                   |
                                                                                   No
                                                                                   |
                                                                                   V
         +------------------+             +-------------------+              +------------------+
  <----  | traffic matches  | <--- No --- | traffic matches   |  <--- No --- | traffic matches  |
         | DNP default      |             | a DNP Allow rule? |              | NP default       |
         | isolation(*)?    |             +-------------------+              | isolation(*)?    |
         +------------------+                      |                         +------------------+
                |                                  |                                |
                |                                 Yes -------> [ Allow ]            |
               Yes ------> [ Drop ]                                                Yes ------> [ Drop ]


CNP = ClusterNetworkPolicy   DNP = DefaultNetworkPolicy   NP = NetworkPolicy
(*) If a Pod has a ingress NetworkPolicy applied, then any ingress traffic to the Pod that does
    not match the NetworkPolicy's ingress rules, matches NetworkPolicy default isolation
    (the Pod is isolated for ingress). Same applies for egress. Same applies for DNP.

```

The diagram above explains the rule evaluation precedence between ClusterNetworkPolicy,
NetworkPolicy and DefaultNetworkPolicy.

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

### User Stories

![Alt text](user_story_diagram.png?raw=true "User Story Diagram")

#### Story 1: Deny traffic from certain sources

As a cluster admin, I want to explicitly deny traffic from certain source IPs
that I know to be bad.

Many admins maintain lists of IPs that are known to be bad actors, especially
to curb DoS attacks. A cluster admin could use ClusterNetworkPolicy to codify
all the source IPs that should be denied in order to prevent that traffic from
accidentally reaching workloads. Note that the inverse of this (allow traffic
from well known source IPs) is also a valid use case.

#### Story 2: Funnel traffic through ingress/egress gateways

As a cluster admin, I want to ensure that all traffic coming into (going out of)
my cluster always goes through my ingress (egress) gateway.

It is common practice in enterprises to setup checkpoints in their clusters at
ingress/egress. These checkpoints usually perform advanced checks such as
firewalling, authentication, packet/connection logging, etc.
This is a big request for compliance reasons, and ClusterNetworkPolicy can ensure
that all the traffic is forced to go through ingress/egress gateways.

#### Story 3: Isolate multiple tenants in a cluster

As a cluster admin, I want to isolate all the tenants (modeled as Namespaces)
on my cluster from each other by default.

Many enterprises are creating shared Kubernetes clusters that are managed by a
centralized platform team. Each internal team that wants to run their workloads
gets assigned a Namespace on the shared clusters. Naturally, the platform team
will want to make sure that, by default, all intra-namespace traffic is allowed
and all inter-namespace traffic is denied.

#### Story 4: Enforce network/security best practices

As a cluster admin, I want all workloads to start with a baseline network/security
model that meets the needs of my company.

A platform admin may want to factor out policies that each namespace would have
to write individually in order to make deployment and auditability easier.
Common examples include allowing all workloads to be able to talk to the cluster
DNS service and, similarly, allowing all workloads to talk to the logging/monitoring
pods running on the cluster.

#### Story 5: Restrict egress to well known destinations

As a cluster admin, I want to explicitly limit which workloads can connect to well
known destinations outside the cluster.

This user story is particularly relevant in hybrid environments where customers
have highly restricted databases running behind static IPs in their networks
and want to ensure that only a given set of workloads is allowed to connect to
the database for PII/privacy reasons. Using ClusterNetworkPolicy, a user can
write a policy to guarantee that only the selected pods can connect to the database IP.

### Notes/Constraints/Caveats

It is important to note that the controller implementation for cluster-scoped
policy APIs will not be provided as part of this KEP. Such controllers which
realize the intent of these APIs will be provided by individual CNI providers,
as is the case with the NetworkPolicy API.

### Risks and Mitigations

A potential risk of the ClusterNetworkPolicy resource is, when it's stacked on
top of existing NetworkPolicies in the cluster, some existing allowed traffic
patterns (which were regulated by those NetworkPolicies) may become blocked by
ClusterNetworkPolicy `Deny` rules, while some isolated workloads may become
accessible instead because of ClusterNetworkPolicy `Allow` rules.

Developers could face some difficulties figuring out why the NetworkPolicies
did not take effect, even if they know to look for ClusterNetworkPolicy rules
that can potentially override these policies:
To understand why traffic between a pair of Pods is allowed/denied, a list of
NetworkPolicy resources in both Pods' Namespace used to be sufficient
(considering no other CRDs in the cluster tries to alter traffic behavior).
The same Pods, on the other hand, can appear as an AppliedTo, or an ingress/egress
peer in any ClusterNetworkPolicy. This makes looking up policies that affect a
particular Pod more challenging than when there's only NetworkPolicy resources.

In addition, in an extreme case where a ClusterNetworkPolicy `Deny` rule,
ClusterNetworkPolicy `Allow` rule, NetworkPolicy rule and DefaultNetworkPolicy
rule applies to an overlapping set of Pods, users will need to refer to the
precedence model mentioned in the [previous section](precedence-model) to
determine which rule would take effect. As shown in that section, figuring out
how stacked policies affect traffic between workloads might not be very straightfoward.

To mitigate this risk and improve UX, a tool which reversely looks up affecting
policies for a given Pod and prints out relative precedence of those rules
can be quite useful. The [cyclonus](https://github.com/mattfenwick/cyclonus) project
for example, could be extended to support ClusterNetworkPolicy and DefaultNetworkPolicy.
This is an orthogonal effort and will not be addressed by this KEP in particular.

### Future Work

Although the scope of the cluster-scoped policies is wide, the above proposal
intends to only solve the use cases documented in this KEP. However, we would
also like to consider the following set of proposals as future work items:
- **Logging**: Very often cluster administrators want to log every connection
  that is either denied or allowed by a firewall rule and send the details to
  an IDS or any custom tool for further processing of that information.
  With the introduction of `deny` rules, it may make sense to incorporate the
  cluster-scoped policy resources with a new field, say `loggingPolicy`, to
  determine whether a connection matching a particular rule/policy must be logged or not.
- **Rule identifier**: In order to collect traffic statistics corresponding to
  a rule, it is necessary to identify the rule which allows/denies that traffic.
  This helps administrators figure the impact of the rules written in a
  cluster-scoped policy resource. Thus, the ability to uniquely identify a rule
  within a cluster-scoped policy resource becomes very important.
  This can be addressed by introducing a field, say `name`, per `ClusterNetworkPolicy`
  and`DefaultNetworkPolicy` ingress/egress rule.
- **Node Selector**: Cluster administrators and developers want to write
  policies that apply to cluster nodes or host network pods. This can be
  addressed by introducing nodeSelector field under `appliedTo` field of the
  `ClusterNetworkPolicy` and `DefaultNetworkPolicy` spec. `DefaultNetworkPolicy`
  is a better candidate compared to K8s `NetworkPolicy` for introducing this
  field as nodes are cluster level resources.

## Design Details

### ClusterNetworkPolicy API Design
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
	Except       []networkingv1.ClusterNetworkPolicyExcept
}

type ClusterNetworkPolicyPeer struct {
	PodSelector  *metav1.LabelSelector
	// required if a PodSelector is specified
	Namespaces   *networkingv1.Namespaces
	IPBlock      *IPBlock
}

type ClusterNetworkPolicyExcept struct {
	PodSelector  *metav1.LabelSelector
	Namespaces   *networkingv1.Namespaces
}

const (
	RuleActionDeny  RuleAction = "Deny"
	RuleActionAllow RuleAction = "Allow"
)
```

For the ClusterNetworkPolicy ingress/egress rule, the `Action` field dictates whether
traffic should be allowed or denied from/to the ClusterNetworkPolicyPeer.
This will be a required field.
An optional `Except` field can be used by policy writers to add exclusions to the
`ClusterNetworkPolicyPeer`s selected. This is especially useful in policies that,
for example, intend to deny ingress from everywhere except a few specific
Namespaces, such as `kube-system`.

### Except Field Semantics
ClusterNetworkPolicy does not validate that the Pods selected by the `Except`
list is subset of `From/To`: the final peers selected are simply, the set of
Pods selected by `ClusterNetworkPolicyPeer`s, subtracting the set of Pods selected
by `ClusterNetworkPolicyExcept`s. This implies that if the set of Pods selected
by `ClusterNetworkPolicyPeer`s does not intersect with the `ClusterNetworkPolicyExcept`
set, then the `Except` list will be completely disregarded.
IPBlock is not allowed in `ClusterNetworkPolicyExcept` since IPBlock already
provides a way to add except CIDRs.

### DefaultNetworkPolicy API Design
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
	// required if a PodSelector is specified
	Namespaces   *networkingv1.Namespaces
	IPBlock      *IPBlock
}
```

Most structs above are very similar to NetworkPolicy and quite self-explanatory.
One detail to notice is that in the DefaultNetworkPolicy Ingress/Egress rule spec,
the peers are created in a field named `OnlyFrom`/`OnlyTo`, as opposed to `To`/`From`
in ClusterNetworkPolicy. We chose this naming to better hint policy writers about
the isolation effect of DefaultNetworkPolicy on the workloads it applies to.

### Shared API Design
The following structs will be added to the `networking.k8s.io` API group and
shared between `ClusterNetworkPolicy` and `DefaultNetworkPolicy`:

```golang
type AppliedTo struct {
	// required if a PodSelector is specified
	NamespaceSelector   *metav1.LabelSelector
	// optional
	PodSelector         *metav1.LabelSelector
}

type Namespaces struct {
	// Self and Selector are mutually-exclusive
	Self       bool
	Selector   *metav1.LabelSelector
}
```

#### AppliedTo
The `AppliedTo` field replaces `PodSelector` in NetworkPolicy spec, as means to specify
the target Pods that this cluster-scoped policy (either `ClusterNetworkPolicy` or
`DefaultNetworkPolicy`) applies to.
Since the policy is cluster-scoped, the `NamespaceSelector` field is required.
An empty `NamespaceSelector` (namespaceSelector: {}) selects all Namespaces in the Cluster.

#### Namespaces
The `Namespaces` field replaces `NamespaceSelector` in NetworkPolicyPeer, as
means to specify the Namespaces of ingress/egress peers for cluster-scoped policies.
For selecting Pods from specific Namespaces, the `Selector` field under `Namespaces`
works exactly as `NamespaceSelector`. The `Self` field is added to satisfy the
specific needs for cluster-scoped policies:

__Self:__ An optional field, which evaluates to false by default.
When `self: true` is set, no selectors can be present concurrently. This is a
special keyword to indicate that the rule only applies to the Namespace for
which the ingress/egress rule is currently being evaluated upon. Since the Pods
selected by the ClusterNetworkPolicy appliedTo could be from multiple Namespaces,
the scope of ingress/egress rules whose `namespace.self=true` will be the Pod's
own Namespace for each selected Pod.
Consider the following example:

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

The above DefaultNetworkPolicy should be interpreted as: for each Namespace in
the cluster, all Pods in that Namespace should only allow traffic from Pods in
the _same Namespace_ who has label app=b. Hence, the policy above allows
x/b1 -> x/a1 and y/b2 -> y/a2, but denies y/b2 -> x/a1 and x/b1 -> y/a2.

#### IPBlock

The `ClusterNetworkPolicyPeer` and `DefaultNetworkPolicyPeer` both allow the
ability to set an `IPBlock` as a peer. The usage of this field is similar to
how it is used in the NetworkPolicyPeer. However, we should also explicitly
note that the IPBlock set in this field could belong to the cluster locally, or
could be cluster external. For example, the peer could be set with a subnet
or an IP which maps to a Pod existing in the cluster. This means that a Pod
in a cluster-scoped NetworkPolicy could be identified by either the labels
applied on the Pod, or its IP address. In case of multiple conflicting rules
targeting the same Pod, but identified in different ways, such as via labelSelector
or via PodIP set in IPBlock, the net effect of the rules will be determined
by the `action` associated with the rule and/or the resource in which the
rule is set in, i.e. ClusterNetworkPolicy rule takes precedence over a
DefaultNetworkPolicy rule.

### Sample Specs for User Stories

![Alt text](user_story_diagram.png?raw=true "User Story Diagram")

#### Story 1: Deny traffic from certain sources
As a cluster admin, I want to explicitly deny traffic from certain source IPs
that I know to be bad.

```yaml
apiVersion: networking.k8s.io/v1alpha1
kind: ClusterNetworkPolicy
metadata:
  name: deny-bad-ip
spec:
  appliedTo:
    # if there's an ingress gateway in the cluster, applying the policy to
    # gateway namespace will be sufficient
    namespaceSelector: {}
  ingress:
    - action: Deny
      from:
      - ipBlock:
          cidr: 62.210.0.0/16  # blacklisted addresses
```

#### Story 2: Funnel traffic through ingress/egress gateways
As a cluster admin, I want to ensure that all traffic coming into (going out of)
my cluster always goes through my ingress (egress) gateway.

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
      - ipBlock:
          cidr: 0.0.0.0/0
      except:
      - namespaces:
          self: true
      - namespaces:
          selector:
            matchLabels:
              kubernetes.io/metadata.name: dmz  # ingress gateway
  egress:
    - action: Deny
      to:
      - ipBlock:
          cidr: 0.0.0.0/0
      except:
      - namespaces:
          self: true
      - namespaces:
          selector:
            matchLabels:
              kubernetes.io/metadata.name: istio-egress  # egress gateway
```

__Note:__ The above policy is very restrictive, i.e. it rejects ingress/egress
traffic between tenant Namespaces and `kube-system`. For `coredns` etc. to work,
`kube-system` Namespace or at least the `coredns` pods needs to be added into
the Deny `except` list.

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

__Note:__ The above policy will take no effect if applied together with
`ingress-egress-gateway`, since both policies apply to the same Namespaces, and
ClusterNetworkPolicy rules have higher precedence than DefaultNetworkPolicy rules.

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

__Note:__ The above policy only ensures that traffic from `app=system` Namespaces
will not be blocked, if developers create NetworkPolicy which isolates the Pods in
tenant Namespaces. When there's a ClusterNetworkPolicy like `ingress-egress-gateway`
present in the cluster, the above policy will be overridden as `Deny` rules have
higher precedence than `Allow` rules. In that case, the `app=system` Namespaces need
to be added to the Deny `except` list of `ingress-egress-gateway`.

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
  - Ensure `Deny` rules override all allowed traffic in the cluster
  - Ensure `Allow` rules override K8s NetworkPolicies
  - Ensure that in stacked ClusterNetworkPolicies/K8s NetworkPolicies, the following precedence is maintained
    aggregated `Deny` rules > aggregated `Allow` rules > K8s NetworkPolicy rules
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

#### Alpha to Beta Graduation

- Gather feedback from developers and surveys
- At least 1 CNI provider must provide the implementation for the complete set
  of alpha features
- Evaluate "future work" items based on feedback from community

#### Beta to GA Graduation

- At least 2 CNI providers must provide the implementation for the complete set
  of alpha features
- More rigorous forms of testing
  — e.g., downgrade tests and scalability tests
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

#### Upgrade considerations

As such, the cluster-scoped policy resources are new and shall not exist prior
to upgrading to a new version. Thus, there is no direct impact on upgrades.

#### Downgrade considerations

Downgrading to a version which no longer supports cluster-scoped policy APIs
must ensure that appropriate security rules are created to mimick the cluster-scoped
policy rules by other means, such that no unintended traffic is allowed.

### Version Skew Strategy

n/a

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

1.22:
- disable by default
- allow gate to enable the feature
- release note

1.24:
- enable by default
- allow gate to disable the feature
- release note

1.26:
- remove gate

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ClusterNetworkPolicy
  - Components depending on the feature gate: kube-apiserver

###### Does enabling the feature change any default behavior?

Enabling the feature by itself has no effect on the cluster.
Creating a ClusterNetworkPolicy/DefaultNetworkPolicy does have an effect on
the cluster, however they must be specifically created, which means the
administrator is aware of the impact.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Once enabled, the feature can be disabled via feature gate. However, disabling
the feature may cause created cluster-scoped policy resources to be deleted,
which may impact the security of the cluster. Administrators must make provision
to secure their cluster by other means before disabling the feature gate.

###### What happens if we reenable the feature if it was previously rolled back?

n/a

###### Are there any tests for feature enablement/disablement?

n/a

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

Enabling this feature by itself will have no impact on the cluster and no
new API calls will be made.

###### Will enabling / using this feature result in introducing new API types?

Enabling this feature will introduce new API types as described in the [design](#design-details)
section. The supported number of objects per cluster will depend on the individual
CNI providers who will be responsible to provide the implementation to realize
these resources.

###### Will enabling / using this feature result in any new calls to the cloud provider?

n/a

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

n/a

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

n/a

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

n/a

## Implementation History

- 2021-02-18 - Created initial PR for the KEP

## Drawbacks

Securing traffic for a cluster for administrator's use case can get complex.
This leads to introduction of a more complex set of APIs which could confuse
users.

## Alternatives

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
into a single CRD with an additional overrideable field in Ingress/ Egress rule
as shown below.

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

As the semantics for overridable Cluster NetworkPolicies are different from
K8s Network Policies, cluster administrators who worked on K8s NetworkPolicies
will have hard time writing similar policies for the cluster. Also, modifying
a single field (`IsOverridable`) of a rule will change the priority in a
non-intuitive manner which may cause some confusion. For these reasons, we
decided not go with this proposal.

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

RuleActionDeny and RuleActionAllow are used to specify rules that take higher
precedence than Kubernetes NetworkPolicies whereas RuleActionBaselineAllow is
used to specify the rules that take lower precedence Kubernetes NetworkPolicies.
The RuleActionBaselineAllow rules have same semantics as Kubernetes NetworkPolicy
rules but defined at cluster level.

One of the reasons we did not go with this approach is the ambiguity of the term
`BaselineAllow`. Also, the semantics around `RuleActionBaselineAllow` is
slightly different as it involves implicit isolation compared to explicit
Allow/ Deny rules with other `RuleActions`.
