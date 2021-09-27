# KEP-2091: Add support for ClusterNetworkPolicy resources

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [ClusterNetworkPolicy resource](#clusternetworkpolicy-resource)
  - [ClusterDefaultNetworkPolicy resource](#clusterdefaultnetworkpolicy-resource)
  - [Precedence model](#precedence-model)
  - [User Stories](#user-stories)
    - [Story 1: Deny traffic from certain sources](#story-1-deny-traffic-from-certain-sources)
    - [Story 2: Ensure traffic goes through ingress/egress gateways](#story-2-ensure-traffic-goes-through-ingressegress-gateways)
    - [Story 3: Isolate multiple tenants in a cluster](#story-3-isolate-multiple-tenants-in-a-cluster)
    - [Story 4: Zero-trust default security posture for tenants](#story-4-zero-trust-default-security-posture-for-tenants)
    - [Story 5: Restrict egress to well known destinations](#story-5-restrict-egress-to-well-known-destinations)
  - [RBAC](#rbac)
  - [Key differences between Cluster-scoped policies and Network Policies](#key-differences-between-cluster-scoped-policies-and-network-policies)  
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Future Work](#future-work)
- [Design Details](#design-details)
  - [ClusterNetworkPolicy API Design](#clusternetworkpolicy-api-design)
  - [ClusterDefaultNetworkPolicy API Design](#clusterdefaultnetworkpolicy-api-design)
  - [Shared API Design](#shared-api-design)
    - [AppliedTo](#appliedto)
    - [Namespaces](#namespaces)
  - [Sample Specs for User Stories](#sample-specs-for-user-stories)
    - [Sample spec for Story 1: Deny traffic from certain sources](#sample-spec-for-story-1-deny-traffic-from-certain-sources)
    - [Sample spec for Story 2: Ensure traffic goes through ingress/egress gateways](#sample-spec-for-story-2-ensure-traffic-goes-through-ingressegress-gateways)
    - [Sample spec for Story 3: Isolate multiple tenants in a cluster](#sample-spec-for-story-3-isolate-multiple-tenants-in-a-cluster)
    - [Sample spec for Story 4: Zero-trust default security posture for tenants](#sample-spec-for-story-4-zero-trust-default-security-posture-for-tenants)
    - [Sample spec for Story 5: Restrict egress to well known destinations](#sample-spec-for-story-5-restrict-egress-to-well-known-destinations)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha to Beta Graduation](#alpha-to-beta-graduation)
    - [Beta to GA Graduation](#beta-to-ga-graduation)
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
ClusterNetworkPolicy API and the ClusterDefaultNetworkPolicy API to complement
the developer focused NetworkPolicy API in Kubernetes.

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
1. Error reporting for Network Policy
2. Kubernetes Node policies
3. New policy selectors (services, service accounts, etc.)
4. Tools/CLI to debug/explain NetworkPolicies, cluster-scoped policies and their
   impact on workloads.
5. Cluster external traffic access control.

## Proposal

In order to achieve the two primary broad use cases for a cluster admin to
secure K8s clusters, we propose to introduce the following two new resources
under `netpol.networking.k8s.io` API group:
- ClusterNetworkPolicy
- ClusterDefaultNetworkPolicy

### ClusterNetworkPolicy resource

A ClusterNetworkPolicy resource will help the administrators set strict
security rules for the cluster, i.e. a developer CANNOT override these rules
by creating NetworkPolicies that applies to the same workloads as the
ClusterNetworkPolicy does.

Unlike the NetworkPolicy resource in which each rule represents an allowed
traffic, ClusterNetworkPolicy will enable administrators to set `Empower`,
`Deny` or `Allow` as the action of each rule. ClusterNetworkPolicy rules should
be read as-is, i.e. there will not be any implicit isolation effects for the Pods
selected by the ClusterNetworkPolicy, as opposed to what NetworkPolicy rules imply.

In terms of precedence, the aggregated `Empower` rules (all ClusterNetworkPolicy
rules with action `Empower` in the cluster combined) should be evaluated before
aggregated ClusterNetworkPolicy `Deny` rules, followed by aggregated ClusterNetworkPolicy
`Allow` rules, followed by NetworkPolicy rules in all Namespaces. As such, the
`Empower` rules have the highest precedence, which shall only be used to provide
exceptions to deny rules. The `Empower` rules do not guarantee that the traffic
will not be dropped: it simply denotes that the packets matching those rules can bypass the
ClusterNetworkPolicy `Deny` rule evaluation.

ClusterNetworkPolicy `Deny` rules are useful for administrators to explicitly
block traffic from malicious clients, or workloads that poses security risks.
Those traffic restrictions can only be lifted once the `Deny` rules are deleted
or modified by the admin. In clusters where the admin requires total control over
security postures of all workloads, the `Deny` rules can also be used to deny all
incoming/outgoing traffic in the cluster, with few exceptions that's listed out
by `Empower` rules.

On the other hand, the `Allow` rules can be used to call out traffic in the cluster
that needs to be allowed for certain components to work as expected (egress to
CoreDNS for example). Those traffic should not be blocked when developers apply
NetworkPolicy to their Namespaces which isolates the workloads.

### ClusterDefaultNetworkPolicy resource

A ClusterDefaultNetworkPolicy resource will help the administrators set baseline
security rules for the cluster, i.e. a developer CAN override these rules by creating
NetworkPolicies that applies to the same workloads as the ClusterDefaultNetworkPolicy
does.

ClusterDefaultNetworkPolicy works just like NetworkPolicy except that it is cluster-scoped.
When workloads are selected by a ClusterDefaultNetworkPolicy, they are isolated except
for the ingress/egress rules specified. ClusterDefaultNetworkPolicy rules will not have
actions associated -- each rule will be an 'allow' rule.

Aggregated NetworkPolicy rules will be evaluated before aggregated ClusterDefaultNetworkPolicy
rules. If a Pod is selected by both, a ClusterDefaultNetworkPolicy and a NetworkPolicy,
then the ClusterDefaultNetworkPolicy's effect on that Pod becomes obsolete.
In this case, the traffic allowed will be solely determined by the NetworkPolicy.

### Precedence model

```
         +-----------------------+      
  ---->  |    traffic matches    | -------- Yes
         | a CNP Empower rule?   |           |        
         +-----------------------+           |  
                 |                           |                                     
                 No                          |    Yes ------> [ Allow ]           Yes ------> [ Allow ]
                 |                           |     |                               |
                 V                           v     |                               |
         +------------------+             +-------------------+              +------------------+
         | traffic matches  | --- No -->  | traffic matches   |  --- No -->  | traffic matches  |
         | a CNP Deny rule? |             | a CNP Allow rule? |              | a NetworkPolicy  |
         +------------------+             +-------------------+              | rule?            |
                 |                                                           +------------------+
                 |                                                                 |
                Yes -------> [ Drop ]                                              No
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


CNP = ClusterNetworkPolicy   DNP = ClusterDefaultNetworkPolicy   NP = NetworkPolicy
(*) If a Pod has a ingress NetworkPolicy applied, then any ingress traffic to the Pod that does
    not match the NetworkPolicy's ingress rules, matches NetworkPolicy default isolation
    (the Pod is isolated for ingress). Same applies for egress. Same applies for DNP.

```

The diagram above explains the rule evaluation precedence between ClusterNetworkPolicy,
NetworkPolicy and ClusterDefaultNetworkPolicy.

Consider the following scenario:

- Pod `server` exists in Namespace x. Each Namespace [a, b, c, d] has a Pod named `client`.
The following policy resources also exist in the cluster:
- (1) A ClusterNetworkPolicy `Empower` rule selects Namespace x and makes an exception for ingress traffic from Namespace a.
- (2) A ClusterNetworkPolicy `Deny` rule selects Namespace x and denies all ingress traffic from Namespace a and b.
- (3) A ClusterNetworkPolicy `Allow` rule selects Namespace x and allows all ingress traffic Namespace b and c.
- (4) A NetworkPolicy rule isolates [x/server], only allows ingress traffic from its own Namespace and Namespace a.
- (5) A ClusterDefaultNetworkPolicy rule isolates [x/server], only allows ingress traffic from Namespace d.

Now suppose the client in each Namespace initiates traffic towards x/server.
- a/client -> x/server is affected by rule (1), (2), (4) and (5). Since rule (1) denotes rule (2) should be bypassed, rule (4) applies and the request should be allowed.
- b/client -> x/server is affected by rule (2), (3), (4) and (5). Since rule (2) has highest precedence, the request should be denied.
- c/client -> x/server is affected by rule (3), (4) and (5). Since rule (3) has highest precedence, the request should be allowed.
- d/client -> x/server is affected by rule (4) and (5). Since rule (4) has higher precedence, the request should be denied.

### User Stories

Note: This KEP will focus on East-West traffic, cluster internal, user stories and 
not address North-South traffic, cluster external, use cases, which will be 
solved in a follow-up proposal.

#### Story 1: Deny traffic at a cluster level 

As a cluster admin, I want to explicitly isolate certain pod(s) and(or) 
Namespace(s) from all other cluster internal traffic. 

![Alt text](explicit_isolation.png?raw=true "Explicit Deny")

#### Story 2: Allow traffic at a cluster level

As a cluster admin, I want to explicitly allow traffic to certain pods(s) 
and(or) Namespace(s) from all other cluster internal traffic. 

![Alt text](explicit_allow.png?raw=true "Explicit Allow")

#### Story 3: Explicitly Delegate traffic to existing K8's Network Policy 

As a cluster admin, I want to explicitly delegate traffic to be handled by 
standard namespace scoped network policies. The delegate action can also be 
thought of as a deny exception.  

Note: In the diagram below the ability to talk to the service svc-pub 
in namespace bar-ns-1 is delegated to the k8s network policies 
implemented in foo-ns-1 and foo-ns-2. If no k8's network policies touch the 
delegated traffic the traffic will be allowed. 

![Alt text](delegation.png?raw=true "Delegate")

#### Story 4: Create and Isolate multiple tenants in a cluster

As a cluster admin, I want to build tenants (modeled as Namespace(s))
in my cluster that are isolated from each other by default. Tenancy may be 
modeled as 1:1, where 1 tenant is mapped to a single Namespace, or 1:n, where a 
single tenant may own more than 1 Namespace.

![Alt text](tenants.png?raw=true "Tenants")

#### Story 5: Cluster wide Guardrails 

As a cluster admin, I want all workloads to start with a network/security
model that meets the needs of my company.

In order to follow best practices, an admin may want to begin the cluster
lifecycle with a default zero-trust security model, where in the default policy
of the cluster is to deny traffic. Only traffic that is essential to the cluster
will be opened up with stricter cluster level policies. Namespace owners are therefore
forced to use NetworkPolicies to explicitly allow only known traffic. This follows
a explicit whitelist model which is familiar to many security administrators. 

### RBAC

Cluster-scoped NetworkPolicy resources are meant for cluster administrators.
Thus, access to manage these resources must be granted to subjects which have
the authority to outline the security policies for the cluster. Therefore, by
default, the `admin` and `edit` ClusterRoles will be granted the permissions
to edit the cluster-scoped NetworkPolicy resources.

### Key differences between Cluster-scoped policies and Network Policies

|                          | ClusterNetworkPolicy                                                                                             | K8s NetworkPolicies                                                                | ClusterDefaultNetworkPolicy                                                                                        |
|--------------------------|------------------------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------|
| Target persona           | Cluster administrator or equivalent                                                                              | Developers within Namespaces                                                       | Cluster administrator or equivalent                                                                                |
| Scope                    | Cluster                                                                                                          | Namespaced                                                                         | Cluster                                                                                                            |
| Drop traffic             | Supported with a `Deny` rule action                                                                              | Supported via implicit isolation of target Pods                                    | Supported via implicit isolation of target Pods                                                                    |
| Skip enforcement         | Supported with an `Empower` rule action                                                                          | Not needed                                                                         | Not needed                                                                                                         |
| Allow traffic            | Supported with an `Allow` rule action for ClusterNetworkPolicy                                                   | Default action for all rules is to allow                                           | Default action for all rules is to allow                                                                           |
| Implicit isolation       | No implicit isolation                                                                                            | All rules have an implicit isolation of target Pods                                | All rules have an implicit isolation of target Pods                                                                |
| Rule precedence          | Empower > Deny > Allow action                                                                                    | Rules are additive                                                                 | Rules are additive                                                                                                 |
| Policy precedence        | Enforced before K8s NetworkPolicies                                                                              | Enforced after ClusterNetworkPolicies                                              | Enforced after K8s NetworkPolicies                                                                                 |
| Cluster external traffic | Not supported                                                                                                    | Supported via IPBlock                                                              | Not supported                                                                                                      |
| Namespace selectors      | Supports advanced selection of Namespaces with the use of `namespaces`and label based `namespaceSelector` fields | Supports label based Namespace selection with the use of `namespaceSelector` field | Supports advanced selection of Namespaces with the use of  `namespaces` and label based `namespaceSelector` fields |

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

In addition, in an extreme case where a ClusterNetworkPolicy `Empower` rule,
ClusterNetworkPolicy `Deny` rule, ClusterNetworkPolicy `Allow` rule,
NetworkPolicy rule and ClusterDefaultNetworkPolicy rule applies to an overlapping
set of Pods, users will need to refer to the precedence model mentioned in the
[previous section](#precedence-model) to determine which rule would take effect.
As shown in that section, figuring out how stacked policies affect traffic
between workloads might not be very straightfoward.

To mitigate this risk and improve UX, a tool which reversely looks up affecting
policies for a given Pod and prints out relative precedence of those rules
can be quite useful. The [cyclonus](https://github.com/mattfenwick/cyclonus)
project for example, could be extended to support ClusterNetworkPolicy and
ClusterDefaultNetworkPolicy. This is an orthogonal effort and will not be addressed
by this KEP in particular.

### Future Work

Although the scope of the cluster-scoped policies is wide, the above proposal
intends to only solve the use cases documented in this KEP. However, we would
also like to consider the following set of proposals as future work items:
- **Audit Logging**: Very often cluster administrators want to log every connection
  that is either denied or allowed by a firewall rule and send the details to
  an IDS or any custom tool for further processing of that information.
  With the introduction of `deny` rules, it may make sense to incorporate the
  cluster-scoped policy resources with a new field, say `auditPolicy`, to
  determine whether a connection matching a particular rule/policy must be
  logged or not.
- **Rule identifier**: In order to collect traffic statistics corresponding to
  a rule, it is necessary to identify the rule which allows/denies that traffic.
  This helps administrators figure the impact of the rules written in a
  cluster-scoped policy resource. Thus, the ability to uniquely identify a rule
  within a cluster-scoped policy resource becomes very important.
  This can be addressed by introducing a field, say `name`, per `ClusterNetworkPolicy`
  and `ClusterDefaultNetworkPolicy` ingress/egress rule.
- **Node Selector**: Cluster administrators and developers want to write
  policies that apply to cluster nodes or host network pods. This can be
  addressed by introducing nodeSelector field under `appliedTo` field of the
  `ClusterNetworkPolicy` and `ClusterDefaultNetworkPolicy` spec.
  `ClusterDefaultNetworkPolicy` is a better candidate compared to K8s `NetworkPolicy`
  for introducing this field as nodes are cluster level resources.

## Design Details

### ClusterNetworkPolicy API Design
The following new `ClusterNetworkPolicy` API will be added to the `netpol.networking.k8s.io` API group.

**Note**: Much of the behavior of certain fields proposed below is intentionally aligned with K8s NetworkPolicy
resource, wherever possible. For eg. the behavior of empty or missing fields matches the behavior specified in
the NetworkPolicySpec.

```golang
type ClusterNetworkPolicy struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	// Specification of the desired behavior of ClusterNetworkPolicy.
	Spec ClusterNetworkPolicySpec
}

type ClusterNetworkPolicySpec struct {
	// No implicit isolation of AppliedTo Pods/Namespaces.
	// Required field.
	AppliedTo    AppliedTo
	Ingress      []ClusterNetworkPolicyIngressRule
	Egress       []ClusterNetworkPolicyEgressRule
}

type ClusterNetworkPolicyIngress/EgressRule struct {
	// Action specifies whether this rule must allow traffic or deny traffic. Deny rules take
	// precedence over allow rules. Any exception to a deny rule must be written as an Empower
	// rule which takes highest precedence. i.e. Empower > Deny > Allow
	// Required field for any rule.
	Action       RuleAction
	// List of ports for incoming/outgoing traffic.
	// Each item in this list is combined using a logical OR. If this field is
	// empty or missing, this rule matches all ports (traffic not restricted by port).
	// If this field is present and contains at least one item, then this rule allows/denies
	// traffic only if the traffic matches at least one port in the list.
	// +optional
	Ports        []networkingv1.NetworkPolicyPort
	// List of sources/dest which should be able to access the pods selected for this rule.
	// Items in this list are combined using a logical OR operation. If this field is
	// empty or missing, this rule matches all sources/dest (traffic not restricted by
	// source/dest). If this field is present and contains at least one item, this rule
	// allows/denies traffic only if the traffic matches at least one item in the from/to list.
	// +optional
	From/To      []ClusterNetworkPolicyPeer
}

type ClusterNetworkPolicyPeer struct {
	PodSelector         *metav1.LabelSelector
	// One of NamespaceSelector or Namespaces is required, if a PodSelector is specified.
	// In the same ClusterNetworkPolicyPeer, NamespaceSelector and Namespaces fields are mutually
	// exclusive.
	NamespaceSelector   *metav1.LabelSelector
	Namespaces          *Namespaces
}

const (
	// RuleActionEmpower is the highest priority rules which enable admins to provide exceptions to deny rules.
	RuleActionEmpower   RuleAction = "Empower"
	// RuleActionDeny enables admins to deny specific traffic. Any exception to this deny rule must be overridden by
	// creating a RuleActionEmpower rule.
	RuleActionDeny      RuleAction = "Deny"
	// RuleActionAllow enables admins to specifically allow certain traffic. These rules will be enforced after
	// Empower and Deny rules.
	RuleActionAllow     RuleAction = "Allow"
)
```

For the ClusterNetworkPolicy ingress/egress rule, the `Action` field dictates whether
traffic should be allowed or denied from/to the ClusterNetworkPolicyPeer. This will be a required field.

### ClusterDefaultNetworkPolicy API Design
The following new `ClusterDefaultNetworkPolicy` API will be added to the `netpol.networking.k8s.io` API group:

```golang
type ClusterDefaultNetworkPolicy struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	// Specification of the desired behavior of ClusterDefaultNetworkPolicy.
	Spec ClusterDefaultNetworkPolicySpec
}

type ClusterDefaultNetworkPolicySpec struct {
	// Implicit isolation of AppliedTo Pods.
	AppliedTo   AppliedTo
	Ingress     []ClusterDefaultNetworkPolicyIngressRule
	Egress      []ClusterDefaultNetworkPolicyEgressRule
}

type ClusterDefaultNetworkPolicyIngress/EgressRule struct {
	// List of ports for incoming/outgoing traffic.
	// Each item in this list is combined using a logical OR. If this field is
	// empty or missing, this rule matches all ports (traffic not restricted by port).
	// If this field is present and contains at least one item, then this rule allows
	// traffic only if the traffic matches at least one port in the list.
	// +optional
	Ports       []networkingv1.NetworkPolicyPort
	// List of sources/dest which should be able to access the pods selected for this rule.
	// Items in this list are combined using a logical OR operation. If this field is
	// empty or missing, this rule matches all sources/dest (traffic not restricted by
	// source/dest). If this field is present and contains at least one item, this rule
	// allows traffic only if the traffic matches at least one item in the from/to list.
	// +optional
	From/To     []ClusterDefaultNetworkPolicyPeer
}

type ClusterDefaultNetworkPolicyPeer struct {
	PodSelector       *metav1.LabelSelector
	// One of NamespaceSelector or Namespaces is required, if a PodSelector is specified
	// In the same ClusterDefaultNetworkPolicyPeer, NamespaceSelector and Namespaces
	// fields are mutually exclusive.
	NamespaceSelector *metav1.LabelSelector
	Namespaces        *Namespaces
}
```

### Shared API Design
The following structs will be added to the `netpol.networking.k8s.io` API group and
shared between `ClusterNetworkPolicy` and `ClusterDefaultNetworkPolicy`:

```golang
type AppliedTo struct {
	// required if a PodSelector is specified
	NamespaceSelector   *metav1.LabelSelector
	// optional
	PodSelector         *metav1.LabelSelector
}

// Namespaces define a way to select Namespaces in the cluster.
type Namespaces struct {
	Scope       NamespaceMatchType
	// Labels are set only when scope is "SameLabels".
	Labels      []string
	// Selector is only set when scope is "Selector".
	// Namespaces.Selector has the same effect as NamespaceSelector.
	Selector    *metav1.LabelSelector
}

// NamespaceMatchType describes Namespace matching strategy.
type NamespaceMatchType string

// This list can/might get expanded in the future (i.e. NotSelf etc.)
const (
	NamespaceMatchSelf          NamespaceMatchType = "Self"
	NamespaceMatchSelector      NamespaceMatchType = "Selector"
	NamespaceMatchSameLabels    NamespaceMatchType = "SameLabels"
)

```

#### AppliedTo
The `AppliedTo` field in Cluster-scoped network policies is what `Spec.PodSelector` field is to K8s NetworkPolicy spec,
as means to specify the target Pods that this cluster-scoped policy (either `ClusterNetworkPolicy` or
`ClusterDefaultNetworkPolicy`) applies to.
Since the policy is cluster-scoped, the `NamespaceSelector` field is required.
An empty `NamespaceSelector` (namespaceSelector: {}) selects all Namespaces in the Cluster.

#### Namespaces
The `Namespaces` field complements `NamespaceSelector` in peers, as means to specify the Namespaces of
ingress/egress peers for cluster-scoped policies.
The scope of the Namespaces to be selected is specified by the matching strategy chosen.
For selecting Pods from specific Namespaces, the `Selector` scope works exactly as `NamespaceSelector`.
The `Self` scope is added to satisfy the specific needs for cluster-scoped policies:

__Self:__
This is a special strategy to indicate that the rule only applies to the Namespace for
which the ingress/egress rule is currently being evaluated upon. Since the Pods
selected by the ClusterNetworkPolicy `appliedTo` could be from multiple Namespaces,
the scope of ingress/egress rules whose `scope=self` will be the Pod's
own Namespace for each selected Pod.
Consider the following example:

- Pods [a1, b1] exist in Namespace x, which has labels `app=a` and `app=b` respectively.
- Pods [a2, b2] exist in Namespace y, which also has labels `app=a` and `app=b` respectively.

```yaml
apiVersion: netpol.networking.k8s.io/v1alpha1
kind: ClusterDefaultNetworkPolicy
spec:
  appliedTo:
    namespaceSelector: {}
  ingress:
    - from:
      - namespaces:
          scope: self
        podSelector:
          matchLabels:
            app: b
```

The above ClusterDefaultNetworkPolicy should be interpreted as: for each Namespace in
the cluster, all Pods in that Namespace should only allow traffic from Pods in
the _same Namespace_ who has label app=b. Hence, the policy above allows
x/b1 -> x/a1 and y/b2 -> y/a2, but denies y/b2 -> x/a1 and x/b1 -> y/a2.

__SameLabels:__
This is a special strategy to indicate that the rule only applies to the Namespaces
which share the same label value. Since the Pods selected by the ClusterNetworkPolicy `appliedTo`
could be from multiple Namespaces, the scope of ingress/egress rules whose `scope=samelabels; labels: [tenant]`
will be all the Pods from the Namespaces who have the same label value for the "tenant" key.
Consider the following example:

- Pods [a1, b1] exist in Namespace t1-ns1, which has label `tenant=t1`.
- Pods [a2, b2] exist in Namespace t1-ns2, which has label `tenant=t1`.
- Pods [a3, b3] exist in Namespace t2-ns1, which has label `tenant=t2`.
- Pods [a4, b4] exist in Namespace t2-ns2, which has label `tenant=t2`.

```yaml
apiVersion: netpol.networking.k8s.io/v1alpha1
kind: ClusterDefaultNetworkPolicy
spec:
  appliedTo:
    namespaceSelector:
      matchExpressions: {key: "tenant"; operator: Exists}
  ingress:
    - from:
      - namespaces:
          scope: samelabels
          labels:
            - tenant
```

The above ClusterDefaultNetworkPolicy should be interpreted as: for each Namespace in
the cluster who has a label key set as "tenant", all Pods in that Namespace should only allow traffic
from all Pods in the Namespaces who has the same label value for key `tenant`. Hence, the policy above
allows all Pods in Namespaces labeled `tenant=t1` i.e. t1-ns1 and t1-ns2, to reach each other,
similarly all Pods in Namespaces labeled `tenant=t2` i.e. t2-ns1 and t2-ns2, are allowed
to talk to each other, however it does not allow any Pod in t1-ns1 or t1-ns2 to reach Pods in
t2-ns1 or t2-ns2.

### Sample Specs for User Stories

![Alt text](user_story_diagram.png?raw=true "User Story Diagram")

#### Sample spec for Story 1: Deny traffic from certain sources

n/a

#### Sample spec for Story 2: Ensure traffic goes through ingress/egress gateways
As a cluster admin, I want to ensure that all traffic coming into (going out of)
my cluster always goes through my ingress (egress) gateway.

```yaml
apiVersion: netpol.networking.k8s.io/v1alpha1
kind: ClusterNetworkPolicy
metadata:
  name: ingress-egress-gateway
spec:
  appliedTo:
    namespaceSelector:
      matchLabels:
        type: tenant  # assuming all tenant namespaces will be created with this label
  ingress:
    - action: Empower
      from:
      - namespaceSelector:
          matchExpressions:
            {Key: kubernetes.io/metadata.name, Operator: In, Values: [kube-system, dmz]}
      - namespaces:
          scope: Self
    - action: Deny
      from:
      - namespaceSelector: {}
  egress:
    - action: Empower
      from:
      - namespaceSelector:
          matchExpressions:
            {Key: kubernetes.io/metadata.name, Operator: In, Values: [kube-system, egress-gw]}
      - namespaces:
          scope: Self
    - action: Deny
      to:
      - namespaceSelector: {}
```

#### Sample Spec for Story 3: Isolate multiple tenants in a cluster

As a cluster admin, I want to isolate all the tenants (modeled as Namespaces)
on my cluster from each other by default.

```yaml
apiVersion: netpol.networking.k8s.io/v1alpha1
kind: ClusterNetworkPolicy
metadata:
  name: namespace-isolation # strictly deny inter-namespace traffic for tenant namespaces with the exception
                            # of system namespaces and intra-namespace traffic. Tenants are empowered with
                            # the control of intra-namespace traffic and from system namespaces like kube-system.
spec:
  appliedTo:
    namespaceSelector:
      matchLabels:
        type: tenant      # assuming all tenant namespaces will be created with this label
  ingress:
    - from:
      - namespaces:
          scope: Self
        action: Empower   # add exception to deny rule for intra-namespace traffic
      - namespaceSelector:
          matchLabels:
            app: system
        action: Empower   # add exception to deny rule for system namespaces like kube-system
      - action: Deny      # deny traffic from all namespaces except for exceptions provided by Empower rules
        namespaceSelector: {}
```

In this policy, tenant isolation (Namespace being the boundary) is strictly enforced.
Tenants can however allow or deny intra-namespace traffic and traffic from "kube-system"
Namespace depending on their needs. They cannot, however, overwrite Allow and Deny
rules which cluster admins listed out as guardrails (dns must be allowed, egress
to some IPs must be denied etc.)

#### Sample Spec for Story 4: Zero-trust default security posture for tenants

As a cluster admin, I want all workloads to start with a network/security
model that meets the needs of my company.

```yaml
apiVersion: netpol.networking.k8s.io/v1alpha1
kind: ClusterDefaultNetworkPolicy
spec:
  appliedTo:
    namespaceSelector:
      matchLabels:
        type: tenant  # assuming all tenant namespaces will be created with this label
  # By default, allow no ingress traffic for tenant namespaces
  ingress:
  # By default, allow no egress traffic for tenant namespaces
  egress:
```

__Note:__ The above policy ensures that, by default, all tenant Namespaces do not have the
ability to ingress/egress, except for any traffic that is allowed by a stricter
ClusterNetworkPolicy (see sample [story 3](#sample-spec-for-story-3-isolate-multiple-tenants-in-a-cluster)).
Tenants may override the default deny behavior by explicitly opening up traffic which suit their
needs with the help of K8s NetworkPolicies. However, they will be bound by the ClusterNetworkPolicy rules.

#### Sample spec for Story 5: Restrict egress to well known destinations

n/a

### Test Plan

- Add e2e tests for ClusterNetworkPolicy resource
  - Ensure `Empower` rules can provide exceptions to the `Deny` rules.
  - Ensure `Deny` rules override all allowed traffic in the cluster, except for `Empower` traffic.
  - Ensure `Allow` rules override K8s NetworkPolicies
  - Ensure that in stacked ClusterNetworkPolicies/K8s NetworkPolicies, the following precedence is maintained
    aggregated `Deny` rules > aggregated `Allow` rules > K8s NetworkPolicy rules
- Add e2e tests for ClusterDefaultNetworkPolicy resource
  - Ensure that in absence of ClusterNetworkPolicy rules and K8s NetworkPolicy rules, ClusterDefaultNetworkPolicy rules are observed
  - Ensure that K8s NetworkPolicies override ClusterDefaultNetworkPolicies by applying policies to the same workloads
  - Ensure that stacked ClusterDefaultNetworkPolicies are additive in nature
- e2e test cases must cover ingress and egress rules
- e2e test cases must cover port-ranges, named ports, integer ports etc
- e2e test cases must cover various combinations of `podSelector` in `appliedTo` and ingress/egress rules
- e2e test cases must cover various combinations of `namespaceSelector` in `appliedTo`
- e2e test cases must cover various combinations of `namespaces` in ingress/egress rules
  - Ensure that namespace matching strategies work as expected
- Add unit tests to test the validation logic which shall be introduced for cluster-scoped policy resources
  - Ensure that `self` field cannot be set along with `selector` within `namespaces`
  - Test cases for fields which are shared with NetworkPolicy, like `endPort` etc.
- Ensure that only administrators or assigned roles can create/update/delete cluster-scoped policy resources

### Graduation Criteria

#### Alpha to Beta Graduation

- Gather feedback from developers and surveys
- At least 2 CNI provider must provide the implementation for the complete set
  of alpha features
- Evaluate "future work" items based on feedback from community

#### Beta to GA Graduation

- At least 4 CNI providers must provide the implementation for the complete set
  of beta features
- More rigorous forms of testing
  — e.g., downgrade tests and scalability tests
- Allowing time for feedback
- Completion of all accepted "future work" items

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
Creating a ClusterNetworkPolicy/ClusterDefaultNetworkPolicy does have an effect on
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
the proposed two resources, ClusterNetworkPolicy and ClusterDefaultNetworkPolicy. This alternate
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

An alternative approach is to combine `ClusterNetworkPolicy` and `ClusterDefaultNetworkPolicy`
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
use-cases of both `ClusterNetworkPolicy` and `ClusterDefaultNetworkPolicy`

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
