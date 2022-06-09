# KEP-2091: Add support for AdminNetworkPolicy resources

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [AdminNetworkPolicy resource](#adminnetworkpolicy-resource)
    - [Actions](#actions)
    - [Priority](#priority)
    - [Rule Names](#rule-names)
  - [User Stories](#user-stories)
    - [Story 1: Deny traffic at a cluster level](#story-1-deny-traffic-at-a-cluster-level)
    - [Story 2: Allow traffic at a cluster level](#story-2-allow-traffic-at-a-cluster-level)
    - [Story 3: Explicitly Delegate traffic to existing K8s Network Policy](#story-3-explicitly-delegate-traffic-to-existing-k8s-network-policy)
    - [Story 4: Create and Isolate multiple tenants in a cluster](#story-4-create-and-isolate-multiple-tenants-in-a-cluster)
    - [Story 5: Cluster Wide Default Guardrails](#story-5-cluster-wide-default-guardrails)
  - [RBAC](#rbac)
  - [Key differences between AdminNetworkPolicies and NetworkPolicies](#key-differences-between-adminnetworkpolicies-and-networkpolicies)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigation](#risks-and-mitigation)
  - [Future Work](#future-work)
- [Design Details](#design-details)
  - [AdminNetworkPolicy API Design](#adminnetworkpolicy-api-design)
    - [General Notes on the AdminNetworkPolicy API](#general-notes-on-the-adminnetworkpolicy-api)
    - [Further examples utilizing the self field for <code>NamespaceSet</code> objects](#further-examples-utilizing-the-self-field-for--objects)
  - [Sample Specs for User Stories](#sample-specs-for-user-stories)
    - [Sample spec for Story 1: Deny traffic at a cluster level](#sample-spec-for-story-1-deny-traffic-at-a-cluster-level)
    - [Sample spec for Story 2: Allow traffic at a cluster level](#sample-spec-for-story-2-allow-traffic-at-a-cluster-level)
    - [Sample spec for Story 3: Explicitly Delegate traffic to existing K8s Network Policy](#sample-spec-for-story-3-explicitly-delegate-traffic-to-existing-k8s-network-policy)
    - [Sample spec for Story 4: Create and Isolate multiple tenants in a cluster](#sample-spec-for-story-4-create-and-isolate-multiple-tenants-in-a-cluster)
    - [Sample spec for Story 5: Cluster Wide Default Guardrails](#sample-spec-for-story-5-cluster-wide-default-guardrails)
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
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [NetworkPolicy v2](#networkpolicy-v2)
  - [Empower, Deny, Allow action based CRD](#empower-deny-allow-action-based-crd)
    - [ClusterDefaultNetworkPolicy resource](#clusterdefaultnetworkpolicy-resource)
  - [Single CRD with DefaultRules field](#single-crd-with-defaultrules-field)
  - [Single CRD with IsOverrideable field](#single-crd-with-isoverrideable-field)
  - [Single CRD with BaselineAllow as Action](#single-crd-with-baselineallow-as-action)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
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
their K8s cluster. This doc proposes the AdminNetworkPolicy API to complement
the developer focused NetworkPolicy API in Kubernetes.

## Motivation

Kubernetes provides the NetworkPolicy resource to control traffic within a
cluster. NetworkPolicy focuses on expressing a developer's intent to secure
their applications. However, it was not intended to be used for cluster scoped
administrative traffic control, which is reflected by its design:
- NetworkPolicy uses a "implicit isolation" model, which means that once a policy
  is applied to certain workloads, they are automatically isolated (in the direction
  specified by the policy) and anything allowed needs to be explicitly called out.
- It has no concept of explicit "deny" rules, because the application deployer can
  simply refrain from allowing the things they want to deny.
- The commutative nature of NetworkPolicy can make certain filtering intents difficult
  to express.
Thus, in order to satisfy the needs of a cluster admin, we propose to introduce
a new API that captures the administrator's intent.

### Goals

The goals for this KEP are to satisfy the following key user stories:

1. As a cluster administrator, I want to enforce irrevocable in-cluster guardrails
   that all workloads must adhere to in order to guarantee the safety of my clusters.
   In particular I want to enforce certain network level access controls that are
   cluster scoped and cannot be overridden or bypassed by namespace scoped
   NetworkPolicies.

   Example: I would like to explicitly allow all pods in my cluster to reach
   kubeDNS.

2. As a cluster administrator, I want to have the option to enforce in-cluster network level
   access controls that facilitate network multi-tenancy and strict network level
   isolation between multiple teams and tenants sharing a cluster via use of namespaces
   or groupings of namespaces per tenant.

   Example: I would like to define two tenants in my cluster, one composed of the pods
   in `foo-ns-1` and `foo-ns-2` and the other with pods in `bar-ns-1`, where inter-tenant
   traffic is denied.

3. As a cluster administrator, I want to optionally also deploy an additional default
   set of policies to all in-cluster workloads that may be overridden by the developers
   if needed

   Example: I would like to explicitly delegate the restriction of traffic destined
   for cluster monitoring pods to the developer, allowing them to setup network policy
   to deny or allow the traffic from/to their application.

There are several unique properties that we need to add in order accomplish the
user stories above.
1. Deny rules and, therefore, hierarchical enforcement of policy
2. Semantics for a cluster-scoped policy object that may include
   namespaces/workloads that have not been created yet.
3. Interoperability with existing Kubernetes Network Policy API

### Non-Goals

Our mission is to solve the most common use cases that cluster admins have.
That is, we don't want to solve for every possible policy permutation a user
can think of. Instead, we want to design an API that addresses 90-95% use cases
while keeping the mental model easy to understand and use.
The focus of this KEP is on cluster scoped controls for east-west traffic within
a cluster, meaning that an AdminNetworkPolicyPeer is _always_ defined as a set of
in cluster objects. Cluster scoped controls for north-south traffic may be addressed via
future versions of the api resources introduced in this or other future KEPs.
For the time being, the AdminNetworkPolicy resource introduced by this KEP will
never affect north-south traffic, and thus also don't override or bypass NetworkPolicies
with ipBlock rules that select external traffic.

## Proposal

In order to achieve the three primary broad use cases for a cluster admin to
secure K8s clusters, we propose to introduce the following resources
under `policy.networking.k8s.io` API group:
- AdminNetworkPolicy
- BaselineAdminNetworkPolicy

### AdminNetworkPolicy resource

An AdminNetworkPolicy (ANP) resource will help the administrators set strict security
rules for the cluster, i.e. a developer CANNOT override these rules by creating
NetworkPolicies that applies to the same workloads as the AdminNetworkPolicy does.

#### Actions

Unlike the NetworkPolicy resource in which each rule represents an allowed
traffic, AdminNetworkPolicy will enable administrators to set `Pass`,
`Deny` or `Allow` as the action of each rule. AdminNetworkPolicy rules should
be read as-is, i.e. there will not be any implicit isolation effects for the Pods
selected by the AdminNetworkPolicy, as opposed to what NetworkPolicy rules imply.

- Pass: Traffic that matches a `Pass` rule will skip all further rules from all
  lower precedenced ANPs, and instead be enforced by the K8s NetworkPolicies.
  If there is no K8s NetworkPolicy rule match, and no BaselineAdminNetworkPolicy
  rule match (more on this in the [priority section](#priority)), traffic will be
  governed by the implementation. For most implementations, this means "allow",
  but there may be implementations which have their own policies outside of the
  standard Kubernetes APIs.
- Deny: Traffic that matches a `Deny` rule will be dropped.
- Allow: Traffic that matches an `Allow` rule will be allowed.

AdminNetworkPolicy `Pass` rules allows an admin to delegate security posture for
certain traffic to the Namespace owners by overriding any lower precedenced Allow
or Deny rules. For example, intra-tenant traffic management can be delegated to
tenant admins explicitly with the use of `Pass` rules.

AdminNetworkPolicy `Deny` rules are useful for administrators to explicitly
block traffic with malicious in-cluster clients, or workloads that pose security
risks. Those traffic restrictions can only be lifted once the `Deny` rules are
deleted, modified by the admin, or overridden by a higher priority rule.

On the other hand, the `Allow` rules can be used to call out traffic in the cluster
that needs to be allowed for certain components to work as expected (egress to
CoreDNS for example). Those traffic should not be blocked when developers apply
NetworkPolicy to their Namespaces which isolates the workloads.

#### Priority

The policy instances will be ordered based on the numeric priority assigned to each
ANP. `Priority` is a 32 bit integer value, where a smaller number corresponds to
a higher precedence. The lowest numeric priority value is "0", which corresponds
to the highest precedence. Larger numbers have lower precedence.
Any ANP will have higher precedence over the namespaced NetworkPolicy instances
in the cluster. If traffic matches both an ANP rule and a NetworkPolicy rule, the
only case where the NetworkPolicy rule will be evaluated instead of ANP rule, is
when there is a third higher-precedence ANP `Pass` rule that allows it to bypass
any lower-precedence ANP rules.

The relative precedence of the rules within a single ANP object (all of which
share a priority) will be determined by the order in which the rule is written.
Thus, a rule that appears at the top of the ingress/egress rules would take the
highest precedence.

For alpha, this API defines "1000" as the maximum numeric value for priority, but
this may be revisited as the proposal advances. For future-safety, clients may assume
that higher values will eventually be allowed, and simply treat it as an int32.
Also for alpha, each ANP is limited to 100 ingress rules and 100 egress rules,
which is subject to change (to a greater number) in the future as well.

Conflict resolution: Two policies are considered to be conflicting if they are assigned
the same `priority` and apply to the same resources or a union of resources. In order
to avoid such conflicts, we propose to include tooling for ANP resources to help alert
the admin to potentially ambiguous ANP priority scenarios, more details in [risks and mitigation](#risks-and-mitigation).
However, ultimately it will be the job of the network policy implementation to decide
how to handle overlapping priority situations.

#### Rule Names

In order to help future proof the ANP API, a built in mechanism to identify each
allow/deny/pass rule is required. Such a mechanism will help administrators organize
and identify individual rules within an AdminNetworkPolicy resource.
We propose to introduce a new string field, called `name`, in each `AdminNetworkPolicy`
ingress/egress rule. Currently the `name` of a rule is optional and is most useful
if it is unique within an ANP instance. The max length for the rule name
string is restricted to 100 characters, which provides flexibility for long generated
names.

### BaselineAdminNetworkPolicy

An BaselineAdminNetworkPolicy (BANP) resource will help the administrators set
baseline security rules that describes default connectivity for cluster workloads,
which CAN be overridden by developer NetworkPolicies if needed.

The BaselineAdminNetworkPolicy spec will look almost identical to that of an ANP,
except for two important differences:
1. There is no `Priority` field associated with BaselineAdminNetworkPolicy.
Note that in writing a BaselineAdminNetworkPolicy, admins can still stack a narrower
`Allow` on top of a wider `Deny` rule for example, by positioning the `Allow` rule
higher in the ingress/egress rule list, which will result it to be evaluated
first. However, the authors of this KEP did not have a valid usecase for creating
multiple BaselineAdminNetworkPolicies in a cluster with distinct policy-level
priorities. BANPs are intended for setting cluster default security postures, and
in most cases the subject of such policy should be the entire cluster.
2. There is no `Pass` action for BaselineAdminNetworkPolicy rules.

### User Stories

Note: This KEP will focus on East-West traffic, cluster internal, user stories and
not address North-South traffic, cluster external, use cases, which will be
solved in a follow-up proposal.

#### Story 1: Deny traffic at a cluster level

As a cluster admin, I want to apply non-overridable deny rules
to certain pod(s) and(or) Namespace(s) that isolate the selected
resources from all other cluster internal traffic.

For Example: In this diagram there is a AdminNetworkPolicy applied to the
`sensitive-ns` denying ingress from all other in-cluster resources for all
ports and protocols.

![Alt text](explicit_deny.png?raw=true "Explicit Deny")

#### Story 2: Allow traffic at a cluster level

As a cluster admin, I want to apply non-overridable allow rules to  
certain pods(s) and(or) Namespace(s) that enable the selected resources
to communicate with all other cluster internal entities.  

For Example: In this diagram there is a AdminNetworkPolicy applied to every
namespace in the cluster allowing egress traffic to `kube-dns` pods, and ingress
traffic from pods in `monitoring-ns` for all ports and protocols.

![Alt text](explicit_allow.png?raw=true "Explicit Allow")

#### Story 3: Explicitly Delegate traffic to existing K8s Network Policy

As a cluster admin, I want to explicitly delegate traffic so that it
skips any remaining cluster network policies and is handled by standard
namespace scoped network policies.

For Example: In the diagram below egress traffic destined for the service svc-pub
in namespace bar-ns-1 on TCP port 8080 is delegated to the k8s network policies
implemented in foo-ns-1 and foo-ns-2. If no k8s network policies touch the
delegated traffic the traffic will be allowed.

![Alt text](delegation.png?raw=true "Delegate")

#### Story 4: Create and Isolate multiple tenants in a cluster

As a cluster admin, I want to build tenants in my cluster that are isolated from
each other by default. Tenancy may be modeled as 1:1, where 1 tenant is mapped
to a single Namespace, or 1:n, where a single tenant may own more than 1 Namespace.

For Example: In the diagram below two tenants (Foo and Bar) are defined such that
all ingress traffic is denied to either tenant.  

![Alt text](tenants.png?raw=true "Tenants")

#### Story 5: Cluster Wide Default Guardrails

As a cluster admin I want to change the default security model for my cluster,
so that all intra-cluster traffic (except for certain essential traffic) is
blocked by default. Namespace owners will need to use NetworkPolicies to
explicitly allow known traffic. This follows a whitelist model which is
familiar to many security administrators, and similar
to how [kubernetes suggests network policy be used](https://kubernetes.io/docs/concepts/services-networking/network-policies/#default-policies).

For Example: In the following diagram all Ingress traffic to every cluster
resource is denied by a baseline deny rule.

![Alt text](baseline.png?raw=true "Default Rules")

### RBAC

AdminNetworkPolicy resources are meant for cluster administrators.
Thus, access to manage these resources must be granted to subjects which have
the authority to outline the security policies for the cluster. Therefore, by
default, the `cluster-admin` ClusterRole will be granted the permissions
to edit the AdminNetworkPolicy resources.

### Key differences between AdminNetworkPolicies and NetworkPolicies

|                          | AdminNetworkPolicy                                                                                               | K8s NetworkPolicies                                                                |
|--------------------------|------------------------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------|
| Target persona           | Cluster administrator or equivalent                                                                              | Developers within Namespaces                                                       |
| Scope                    | Cluster                                                                                                          | Namespaced                                                                         |
| Drop traffic             | Supported with a `Deny` rule action                                                                              | Supported via implicit isolation of target Pods                                    |
| Skip enforcement         | Supported with an `Pass` rule action                                                                             | Not needed                                                                         |
| Allow traffic            | Supported with an `Allow` rule action                                                                            | Default action for all rules is to allow                                           |
| Implicit isolation       | No implicit isolation                                                                                            | All rules have an implicit isolation of target Pods                                |
| Rule precedence          | Depends on the order in which they appear within a ANP                                                           | Rules are additive                                                                 |
| Policy precedence        | Depends on `priority` field among ANPs. Enforced before K8s NetworkPolicies if positive numeric priority value   | Enforced after numeric-priority ClusterNetworkPolicies, before baseline-priority AdminNetworkPolicy |
| Matching pod selection   | Can apply different rules to multiple groups of Pods                                                             | Applies rules to a single group of Pods                                            |
| Rule identifiers         | Name per rule in string format. Unique within a ANP                                                              | Not supported                                                                      |
| Cluster external traffic | Not supported                                                                                                    | Partially supported via IPBlock                                                    |
| Namespace selectors      | Supports advanced selection of Namespaces with the use of `namespaceSet`                                         | Supports label based Namespace selection with the use of `namespaceSelector` field |

Note that AdminNetworkPolicy can also apply to Pods in Namespaces that don't
exist yet, and will automatically apply to a new Namespace as long as the new
Namespace's labels match the AdminNetworkPolicy rule's appliedTo selection
criteria. NetworkPolicies, on the contrary, only apply to Pods in the Namespace
they are created in.

### Notes/Constraints/Caveats

It is important to note that the controller implementation for cluster-scoped
policy APIs will not be provided as part of this KEP. Such controllers which
realize the intent of these APIs will be provided by individual network policy
providers, as is the case with the NetworkPolicy API.

### Risks and Mitigation

To understand why traffic between a pair of Pods is allowed or denied, a list of
NetworkPolicy resources in both Pods' Namespace used to be sufficient (considering
no other CRDs in the cluster tries to alter traffic behavior). With the introduction
of AdminNetworkPolicy this is no longer the case, and users could face difficulty
in determining why NetworkPolicies did not take effect.

For example, in the case where a positive priority (non-zero) AdminNetworkPolicy rule,
NetworkPolicy rule and "0" priority AdminNetworkPolicy rule apply to an overlapping
set of Pods, users will need to refer to the priority associated with the
rule to determine which rule would take effect. Figuring out how stacked policies
affect traffic between workloads might not be very straightforward.

To mitigate this risk and improve usability, some additional in-tree tooling
for both the Admin and Developer will need to be created. For the Admin, it is
safe to assume they will have the correct RBAC roles to list all the NetworkPolicies
and AdminNetworkPolicies in a cluster. Therefore, the Admin oriented tooling should
be able to both alert the Admin to any overriding of NetworkPolicies that may occur if a
new AdminNetworkPolicy is to be created and provide a warning if there is another ANP
with the same priority. For the Developer, who usually can only list the NetworkPolicies
in a given namespace, the tooling should simply alert if a given NetworkPolicy would
be overridden by any of the ANPs in a cluster. The aforementioned tooling will not
be a primary development goal during the alpha version of this API, and will most
likely be completed during the beta development cycle.

### Future Work

Although the scope of the AdminNetworkPolicies is extensive, the above proposal
intends to only solve the documented use cases. However, we would
also like to consider the following set of proposals as future work items:
- **Audit Logging**: Very often cluster administrators want to log every connection
  that is either denied or allowed by a firewall rule and send the details to
  an IDS or any custom tool for further processing of that information.
  With the introduction of `deny` rules, it may make sense to incorporate the
  cluster-scoped policy resources with a new field, say `auditPolicy`, to
  determine whether a connection matching a particular rule/policy must be
  logged or not.

## Design Details

### AdminNetworkPolicy API Design

The following new `AdminNetworkPolicy` API will be added to the `policy.networking.k8s.io`
API group:

```golang

// AdminNetworkPolicy describes cluster-level network traffic control rules
type AdminNetworkPolicy struct {
  metav1.TypeMeta
  metav1.ObjectMeta

  // Specification of the desired behavior of AdminNetworkPolicy.
  Spec     AdminNetworkPolicySpec

  // Status is the status to be reported by the implementation, this is not
  // standardized in alpha and consumers should report what they see fit in
  // relation to their AdminNetworkPolicy implementation.
  // +optional
  Status  AdminNetworkPolicyStatus
}

type AdminNetworkPolicyStatus struct {
  Conditions   []metav1.Condition
}

// AdminNetworkPolicySpec provides the specification of AdminNetworkPolicy
type AdminNetworkPolicySpec struct {
  // Priority is a value from 0 to 1000. Rules with lower numeric priority values
  // have higher precedence, and are checked before rules with higher numeric
  // priority values. All AdminNetworkPolicy rules have higher precedence than
  // NetworkPolicy or BaselineAdminNetworkPolicy rules.
  Priority    *int32

  // Subject defines the objects to which this AdminNetworkPolicy applies.  
  Subject     AdminNetworkPolicySubject

  // List of Ingress rules to be applied to the selected objects.
  // A total of 100 rules will be allowed per each network policy instance,
  // this rule count will be calculated as the total summation of the
  // Ingress and Egress rules in a single AdminNetworkPolicy Instance.
  Ingress     []AdminNetworkPolicyIngressRule

  // List of Egress rules to be applied to the selected objects.
  // A total of 100 rules will be allowed per each network policy instance,
  // this rule count will be calculated as the total summation of the
  // Ingress and Egress rules in a single AdminNetworkPolicy Instance.
  Egress      []AdminNetworkPolicyEgressRule
}

// AdminNetworkPolicySubject defines what objects the policy selects.
// Exactly one of the `Namespaces` or `Pods` pointers should be set.
type AdminNetworkPolicySubject struct {
  Namespaces    *metav1.LabelSelector
  Pods          *NamespacedPodSubject
}

// AdminNetworkPolicyIngressRule describes an action to take on a particular
// set of traffic destined for pods selected by an AdminNetworkPolicy's
// Subject field. The traffic must match both ports and from.
type AdminNetworkPolicyIngressRule struct {
  // Name is an identifier for this rule, that should be no more than 100 characters
  // in length.
  // +optional
  Name         string

  // Action specifies whether this rule must pass, allow or deny traffic.
  // Allow: allows the selected traffic
  // Deny: denies the selected traffic
  // Pass: allows the selected traffic to skip and remaining positive priority (non-zero)
  // ANP rules and be delegated by K8's Network Policy.
  Action       AdminNetPolRuleAction

  // Ports allows for matching on traffic based on port and protocols.
  Ports        []AdminNetworkPolicyPort

  // List of sources from which traffic will be allowed/denied/passed to the entities
  // selected by this AdminNetworkPolicyRule. Items in this list are combined using a logical OR
  // operation. If this field is empty, this rule matches no sources.
  // If this field is present and contains at least one item, this rule
  // allows/denies/passes traffic from the defined AdminNetworkPolicyPeer(s)
  From         []AdminNetworkPolicyPeer
}

// AdminNetworkPolicyEgressRule describes an action to take on a particular
// set of traffic originating from pods selected by a AdminNetworkPolicy's
// Subject field. The traffic must match both ports and to.
type AdminNetworkPolicyEgressRule struct {
  // Name is an identifier for this rule, that should be no more than 100 characters
  // in length.
  // +optional
  Name         string

  // Action specifies whether this rule must pass, allow or deny traffic.
  // Allow: allows the selected traffic
  // Deny: denies the selected traffic
  // Pass: allows the selected traffic to skip and remaining positive priority (non-zero)
  // ANP rules and be delegated by K8's Network Policy.
  Action       AdminNetPolRuleAction

  // Ports allows for matching on traffic based on port and protocols.
  Ports        []AdminNetworkPolicyPort

  // List of destinations to which traffic will be allowed/denied/passed from the entities
  // selected by this AdminNetworkPolicyRule. Items in this list are combined using a logical OR
  // operation. If this field is empty, this rule matches no destinations.
  // If this field is present and contains at least one item, this rule
  // allows/denies/passes traffic to the defined AdminNetworkPolicyPeer(s)
  To           []AdminNetworkPolicyPeer
}

const (
  // RuleActionPass enables admins to provide exceptions to ClusterNetworkPolicies and delegate this rule to
  // K8s NetworkPolicies.
  AdminNetPolRuleActionPass    AdminNetPolRuleAction = "Pass"
  // RuleActionDeny enables admins to deny specific traffic.
  AdminNetPolRuleActionDeny    AdminNetPolRuleAction = "Deny"
  // RuleActionAllow enables admins to specifically allow certain traffic.
  AdminNetPolRuleActionAllow   AdminNetPolRuleAction = "Allow"
)
```

The following new `BaslineAdminNetworkPolicy` API will also be added to the `policy.networking.k8s.io`
API group:

```golang

type BaselineAdminNetworkPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	// Specification of the desired behavior of BaselineAdminNetworkPolicy.
	Spec BaselineAdminNetworkPolicySpec `json:"spec"`

	// Status is the status to be reported by the implementation.
	// +optional
	Status BaselineAdminNetworkPolicyStatus `json:"status,omitempty"`
}

// BaselineAdminNetworkPolicyStatus defines the observed state of
// BaselineAdminNetworkPolicy.
type BaselineAdminNetworkPolicyStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
}

// BaselineAdminNetworkPolicySpec defines the desired state of
// BaselineAdminNetworkPolicy.
type BaselineAdminNetworkPolicySpec struct {
	// Subject defines the pods to which this BaselineAdminNetworkPolicy applies.
	Subject AdminNetworkPolicySubject `json:"subject"`

	// Ingress is the list of Ingress rules to be applied to the selected pods
	// if they are not matched by any AdminNetworkPolicy or NetworkPolicy rules.
	// A total of 100 Ingress rules will be allowed in each BANP instance.
	// BANPs with no ingress rules do not affect ingress traffic.
	// +optional
	// +kubebuilder:validation:MaxItems=100
	Ingress []BaselineAdminNetworkPolicyIngressRule `json:"ingress,omitempty"`

	// Egress is the list of Egress rules to be applied to the selected pods if
	// they are not matched by any AdminNetworkPolicy or NetworkPolicy rules.
	// A total of 100 Egress rules will be allowed in each BANP instance. BANPs
	// with no egress rules do not affect egress traffic.
	// +optional
	// +kubebuilder:validation:MaxItems=100
	Egress []BaselineAdminNetworkPolicyEgressRule `json:"egress,omitempty"`
}

// BaselineAdminNetworkPolicyIngressRule describes an action to take on a particular
// set of traffic destined for pods selected by a BaselineAdminNetworkPolicy's
// Subject field.
type BaselineAdminNetworkPolicyIngressRule struct {
	// Name is an identifier for this rule, that may be no more than 100 characters
	// in length. This field should be used by the implementation to help
	// improve observability, readability and error-reporting for any applied
	// BaselineAdminNetworkPolicies.
	// +optional
	// +kubebuilder:validation:MaxLength=100
	Name string `json:"name,omitempty"`

	// Action specifies the effect this rule will have on matching traffic.
	// Currently the following actions are supported:
	// Allow: allows the selected traffic
	// Deny: denies the selected traffic
	Action BaselineAdminNetworkPolicyRuleAction `json:"action"`

	// From is the list of sources whose traffic this rule applies to.
	// If any AdminNetworkPolicyPeer matches the source of incoming
	// traffic then the specified action is applied.
	// This field must be defined and contain at least one item.
	// +kubebuilder:validation:MinItems=1
	From []AdminNetworkPolicyPeer `json:"from"`

	// Ports allows for matching traffic based on port and protocols.
	// If Ports is not set then the rule does not filter traffic via port.
	// +optional
	// +kubebuilder:validation:MaxItems=100
	Ports *[]AdminNetworkPolicyPort `json:"ports,omitempty"`
}

// AdminNetworkPolicyEgressRule describes an action to take on a particular
// set of traffic originating from pods selected by a BaselineAdminNetworkPolicy's
// Subject field.
type BaselineAdminNetworkPolicyEgressRule struct {
	// Name is an identifier for this rule, that may be no more than 100 characters
	// in length. This field should be used by the implementation to help
	// improve observability, readability and error-reporting for any applied
	// BaselineAdminNetworkPolicies.
	// +optional
	// +kubebuilder:validation:MaxLength=100
	Name string `json:"name,omitempty"`

	// Action specifies the effect this rule will have on matching traffic.
	// Currently the following actions are supported:
	// Allow: allows the selected traffic
	// Deny: denies the selected traffic
	Action BaselineAdminNetworkPolicyRuleAction `json:"action"`

	// To is the list of destinations whose traffic this rule applies to.
	// If any AdminNetworkPolicyPeer matches the destination of outgoing
	// traffic then the specified action is applied.
	// This field must be defined and contain at least one item.
	// +kubebuilder:validation:MinItems=1
	To []AdminNetworkPolicyPeer `json:"to"`

	// Ports allows for matching traffic based on port and protocols.
	// If Ports is not set then the rule does not filter traffic via port.
	// +optional
	// +kubebuilder:validation:MaxItems=100
	Ports *[]AdminNetworkPolicyPort `json:"ports,omitempty"`
}

// BaselineAdminNetworkPolicyRuleAction string describes the BaselineAdminNetworkPolicy
// action type.
// +enum
type BaselineAdminNetworkPolicyRuleAction string

const (
	// BaselineAdminNetworkPolicyRuleActionDeny enables admins to deny traffic.
	BaselineAdminNetworkPolicyRuleActionDeny BaselineAdminNetworkPolicyRuleAction = "Deny"
	// BaselineAdminNetworkPolicyRuleActionAllow enables admins to allow certain traffic.
	BaselineAdminNetworkPolicyRuleActionAllow BaselineAdminNetworkPolicyRuleAction = "Allow"
)
```

The following types are common to the `AdminNetworkPolicy` and `BaselineAdminNetworkPolicy`
resources:

```golang

// NamespacedPodSubject allows the user to select a given set of pod(s) in
// selected namespace(s)
type NamespacedPodSubject struct {
  // This field follows standard label selector semantics; if empty,
  // it selects all Namespaces.  
  NamespaceSelector  metav1.LabelSelector

  // Used to explicitly select pods within a namespace; if empty,
  // it selects all Pods.  
  PodSelector        metav1.LabelSelector
}

// AdminNetworkPolicyPort describes how to select network ports on pod(s).
// Exactly one field must be set.
// +kubebuilder:validation:MaxProperties=1
// +kubebuilder:validation:MinProperties=1
type AdminNetworkPolicyPort struct {
	// Port selects a port on a pod(s) based on number.
	// +optional
	PortNumber *Port `json:"portNumber,omitempty"`

	// NamedPort selects a port on a pod(s) based on name.
	// +optional
	NamedPort *string `json:"namedPort,omitempty"`

	// PortRange selects a port range on a pod(s) based on provided start and end
	// values.
	// +optional
	PortRange *PortRange `json:"portRange,omitempty"`
}

type Port struct {
	// Protocol is the network protocol (TCP, UDP, or SCTP) which traffic must
	// match. If not specified, this field defaults to TCP.
	Protocol v1.Protocol `json:"protocol"`

	// Number defines a network port value.
	Port int32 `json:"port"`
}

// PortRange defines an inclusive range of ports from the the assigned Start value
// to End value.
type PortRange struct {
	// Protocol is the network protocol (TCP, UDP, or SCTP) which traffic must
	// match. If not specified, this field defaults to TCP.
	Protocol v1.Protocol `json:"protocol,omitempty"`

	// Start defines a network port that is the start of a port range, the Start
	// value must be less than End.
	Start int32 `json:"start"`

	// End defines a network port that is the end of a port range, the End value
	// must be greater than Start.
	End int32 `json:"end"`
}

// AdminNetworkPolicyPeer defines an in-cluster peer to allow traffic to/from.
// Exactly one of the selector pointers should be set for a given peer.
type AdminNetworkPolicyPeer struct {
	// Namespaces defines a way to select a set of Namespaces.
	// +optional
	Namespaces *NamespacedPeer `json:"namespaces,omitempty"`
	// Pods defines a way to select a set of pods in
	// in a set of namespaces.
	// +optional
	Pods *NamespacedPodPeer `json:"pods,omitempty"`
}

type NamespaceRelation string

const (
	NamespaceSelf    NamespaceRelation = "Self"
	NamespaceNotSelf NamespaceRelation = "NotSelf"
)

// NamespacedPeer defines a flexible way to select Namespaces in a cluster.
// Exactly one of the selectors must be set.  If a consumer observes none of
// its fields are set, they must assume an unknown option has been specified
// and fail closed.
// +kubebuilder:validation:MaxProperties=1
// +kubebuilder:validation:MinProperties=1
type NamespacedPeer struct {
	// Related provides a mechanism for selecting namespaces relative to the
	// subject pod. A value of "Self" matches the subject pod's namespace,
	// while a value of "NotSelf" matches namespaces other than the subject
	// pod's namespace.
	// +optional
	Related *NamespaceRelation `json:"related,omitempty"`

	// NamespaceSelector is a labelSelector used to select Namespaces, This field
	// follows standard label selector semantics; if present but empty, it selects
	// all Namespaces.
	// +optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`

  // SameLabels is used to select a set of Namespaces that share the same values
  // for a set of labels.
	// To be selected a Namespace must have all of the labels defined in SameLabels,
	// and they must all have the same value as the subject of this policy.
	// If Samelabels is Empty then nothing is selected.
  // +optional
	SameLabels []string `json:"sameLabels,omitempty"`

	// NotSameLabels is used to select a set of Namespaces that do not have a set
	// of label(s). To be selected a Namespace must have none of the labels defined
	// in NotSameLabels. If NotSameLabels is empty then nothing is selected.
	// +optional
	NotSameLabels []string `json:"notSameLabels,omitempty"`
}

// NamespacedPodPeer defines a flexible way to select Namespaces and pods in a
// cluster. The `Namespaces` and `PodSelector` fields are required.
type NamespacedPodPeer struct {
	// Namespaces is used to select a set of Namespaces.
	Namespaces NamespacedPeer `json:"namespaces"`

	// PodSelector is a labelSelector used to select Pods, This field is NOT optional,
	// follows standard label selector semantics and if present but empty, it selects
	// all Pods.
	PodSelector metav1.LabelSelector `json:"podSelector"`
}
```

#### General Notes on the AdminNetworkPolicy API

- Much of the proposed behavior is intentionally not aligned with
K8s NetworkPolicy resource, especially in regards to the behavior of empty fields.
Specifically this api is designed to be verbose and explicit. Please pay attention
to the comments above each field for more information.

- For the AdminNetworkPolicy ingress/egress rule, the `Action` field dictates whether
traffic should be allowed/denied/passed from/to the AdminNetworkPolicyPeer. This will be a required field.

- The `AdminNetworkPolicySubject` and `AdminNetworkPolicyPeer` types are explicitly
designed to allow for future extensibility with a focus on the addition of new types
of selectors. Specifically it will allow for failing closed in the event an implementation
does not implement a defined selector. For example, If a new type (`ServiceAccounts`)
was added to the `AdminNetworkPolicyPeer` struct, and an implementation had not
yet implemented support for such a selector, an ANP using the new selector would
have no effect since the implementation would simply see an empty `AdminNetworkPolicyPeer`
object.

#### Further examples utilizing the self field for `NamespacedPeer` objects

__Self:__
This is a special strategy to indicate that the rule only applies to the Namespace for
which the ingress/egress rule is currently being evaluated upon. Since the Pods
selected by the AdminNetworkPolicy `subject` could be from multiple Namespaces,
the scope of ingress/egress rules whose `namespaces.related=self` will be the Pod's
own Namespace for each selected Pod.
Consider the following example:

- Pods [a1, b1], with labels `app=a` and `app=b` respectively, exist in Namespace x.
- Pods [a2, b2], with labels `app=a` and `app=b` respectively, exist in Namespace y.

```yaml
apiVersion: policy.networking.k8s.io/v1alpha1
kind: AdminNetworkPolicy
spec:
  priority: 10
  subject:
    namespaceSelector: {}
  ingress:
  - action: Allow
    from:
    - pods:
        namespaces:
          related: self
        podSelector:
          matchLabels:
            app: b
```

The above AdminNetworkPolicy should be interpreted as: for each Namespace in
the cluster, all Pods in that Namespace should strictly allow traffic from Pods in
the _same Namespace_ who has label app=b at all ports. Hence, the policy above allows
x/b1 -> x/a1 and y/b2 -> y/a2, but does not allow y/b2 -> x/a1 and x/b1 -> y/a2.

__SameLabels:__
This is a special strategy to indicate that the rule only applies to the Namespaces
which share the same label value. Since the Pods selected by the AdminNetworkPolicy `subject`
could be from multiple Namespaces, the scope of ingress/egress rules whose `namespaces.samelabels=tenant`
will be all the Pods from the Namespaces who have the same label value for the "tenant" key.
Consider the following example:

- Pods [a1, b1] exist in Namespace t1-ns1, which has label `tenant=t1`.
- Pods [a2, b2] exist in Namespace t1-ns2, which has label `tenant=t1`.
- Pods [a3, b3] exist in Namespace t2-ns1, which has label `tenant=t2`.
- Pods [a4, b4] exist in Namespace t2-ns2, which has label `tenant=t2`.

```yaml
apiVersion: policy.networking.k8s.io/v1alpha1
kind: AdminNetworkPolicy
spec:
  priority: 20
  subject:
    namespaceSelector:
      matchExpressions: {key: "tenant"; operator: Exists}
  ingress:
    - action: Pass
      from:
      - namespaces:
          sameLabels:
          - tenant
```

The above AdminNetworkPolicy should be interpreted as: for each Namespace in
the cluster who has a label key set as "tenant", traffic for all Pods in that Namespace
from all Pods in the Namespaces who has the same label value for key `tenant` is delegated to the Namespace
admins, i.e such traffic will not be subject to any ANP (`priority` > 50) rules and be evaluated by K8s NetworkPolicies.
Hence, the policy above delegates traffic from all Pods in Namespaces labeled `tenant=t1` i.e. t1-ns1 and t1-ns2,
to reach each other, to K8s NetworkPolicies, similarly traffic for all Pods in Namespaces labeled `tenant=t2`
i.e. t2-ns1 and t2-ns2, to talk to each other is delegated to K8s NetworkPolicies as well, however it does not
delegate traffic from any Pod in t1-ns1 or t1-ns2 to reach Pods in t2-ns1 or t2-ns2, such traffic is still subject
to ANP rules.

### Sample Specs for User Stories

#### Sample spec for Story 1: Deny traffic at a cluster level

```yaml
apiVersion: policy.networking.k8s.io/v1alpha1
kind: AdminNetworkPolicy
metadata:
  name: cluster-wide-deny-example
spec:
  priority: 10
  subject:
    namespaces:
      matchLabels:
        kubernetes.io/metadata.name: sensitive-ns
  ingress:
  - action: Deny
    from:
    - namespaces:
        namespaceSelector: {}
```

#### Sample spec for Story 2: Allow traffic at a cluster level

```yaml
apiVersion: policy.networking.k8s.io/v1alpha1
kind: AdminNetworkPolicy
metadata:
  name: cluster-wide-allow-example
spec:
  priority: 30
  subject:
    namespaces: {}
  ingress:
  - action: Allow
    from:
    - namespaces:
        namespaceSelector:
          matchLabels:
            kubernetes.io/metadata.name: monitoring-ns
  egress:
  - action: Allow
    to:
    - pods:
        namespaces:
          namespaceSelector:
            matchlabels:
              kubernetes.io/metadata.name: kube-system
        podSelector:   
          matchlabels:
            app: kube-dns
```

#### Sample spec for Story 3: Explicitly Delegate traffic to existing K8s Network Policy

```yaml
apiVersion: policy.networking.k8s.io/v1alpha1
kind: AdminNetworkPolicy
metadata:
  name: pub-svc-delegate-example
spec:
  priority: 20
  subject:
    namespaces: {}
  egress:
  - action: Pass
    to:
    - pods:
        namespaces:
          namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: bar-ns-1
        podSelector:
          matchLabels:
            app: svc-pub
    ports:
      port:
       - protocol: TCP
         number: 8080
```

#### Sample spec for Story 4: Create and Isolate multiple tenants in a cluster

```yaml
apiVersion: policy.networking.k8s.io/v1alpha1
kind: AdminNetworkPolicy
metadata:
  name: tenant-creation-example
spec:
  priority: 50
  subject:
    namespaces:
      matchExpressions: {key: "tenant"; operator: Exists}
  ingress:
  - action: Deny
    from:
    - namespaces:
        notSameLabels:
          - tenant
```

Note: the above AdminNetworkPolicy can also be written in the following fashion:
```yaml
apiVersion: policy.networking.k8s.io/v1alpha1
kind: AdminNetworkPolicy
metadata:
  name: tenant-creation-example
spec:
  priority: 50
  subject:
    namespaces:
      matchExpressions: {key: "tenant"; operator: Exists}
  ingress:
  - action: Pass
    from:
    - namespaces:
        sameLabels:
        - tenant
  - action: Deny   # Deny everything else other than same tenant traffic
    from:
    - namespaces:
        namespaceSelector: {}
```
The difference is that in the first case, traffic within tenant Namespaces will fall
through, and be evaluated against lower-priority ClusterNetworkPolicies, and then
NetworkPolicies. In the second case, the matching packet will skip all AdminNetworkPolicy
evaluation (except for AdminNetworkPolicy priority=0), and only match
against NetworkPolicy rules in the cluster. In other words, the second AdminNetworkPolicy
specifies intra-tenant traffic must be delegated to the tenant Namespace owners.

#### Sample spec for Story 5: Cluster Wide Default Guardrails

```yaml
apiVersion: policy.networking.k8s.io/v1alpha1
kind: BaselineAdminNetworkPolicy
metadata:
  name: baseline-rule-example
spec:
  subject:
    namespaces: {}
  ingress:
  - action: Deny   # zero-trust cluster default security posture
    from:
    - namespaces:
        namespaceSelector: {}
  egress:
  - action: Deny
    to:
    - namespaces:
        namespaceSeletor: {}

```

### Test Plan

- Add e2e tests for AdminNetworkPolicy resource
  - Ensure `Pass` rules are delegated and are not subject to ANP rules.
  - Ensure `Deny` rules drop traffic.
  - Ensure `Allow` rules allow traffic.
  - Ensure that in stacked ClusterNetworkPolicies/K8s NetworkPolicies, precedence is maintained
    as per the `priority` set in ANP.
- e2e test cases must cover ingress and egress rules.
- e2e test cases must cover port-ranges, named ports, integer ports etc.
- e2e test cases must cover various combinations of `namespaceSet*s` in each ingress/egress rule.
- Ensure that namespace matching strategies work as expected.
- Add unit tests to test the validation logic which shall be introduced for cluster-scoped policy resources.
  - Ensure exactly one selector has to be set in an `Subject` section.
  - Ensure exactly one selector has to be set in an `AdminNetworkPolicyPeer` section.
  - Test cases for fields which are shared with NetworkPolicy, like `endPort` etc.
- Ensure that only administrators or assigned roles can create/update/delete cluster-scoped policy resources.
- Ensure smooth integration with existing Kubernetes NetworkPolicy.
  - Ensure all positive priority (non-zero) ANP rules are evaluated before any NetworkPolicy rules.
  - Ensure ANP rules with priority="0" are evaluated after any NetworkPolicy rules.

### Graduation Criteria

#### Alpha to Beta Graduation

- Gather feedback from developers and surveys
- At least 2 implementors must provide a functional and scalable implementation
  for the complete set of alpha features.
    - Specifically,  ensure that only selecting E/W cluster traffic is plausible
      at scale.
- Evaluate the need for multiple `Subject`s per ANP.
- Evaluate "future work" items based on feedback from community and challenges
  faced by implementors.
- Ensure extensibility of adding new fields. i.e. adding new fields do not "fail-open"
  traffic for older clients.
- Revisit the topic of whether this API should cover north-south traffic.

#### Beta to GA Graduation

- At least 4 implementors providers must provide a scalable implementation for the
  complete set of beta features
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
policy rules by other means, such that no unintended traffic is allowed, and all
intended traffic is allowed.

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

<!--
This section must be completed when targeting alpha to a release.
-->

N/A for `alpha` release.

NOTE: for `alpha` this resource will be implemented as a CRD following the
precedence set by the gateway API.

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.
-->

N/A for `alpha` release.

NOTE: for `alpha` this resource will be implemented as a CRD following the
precedence set by the gateway API.

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

Creating a AdminNetworkPolicy does have an effect on the cluster, however they
must be specifically created, which means the administrator is aware of the impact.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

For `alpha` there will be no feature gate so this is N/A.

###### What happens if we reenable the feature if it was previously rolled back?

For `alpha` there will be no feature gate so this is N/A.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->

Not in-tree, generally the implementations should have unit tests covering this
scenario.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

N/A for `alpha`.

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

N/A for `alpha`.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

The AdminNetworkPolicy API has a `Status` field which should be used by the
implementation to report weather or not the rules were correctly programmed.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

This will be tested once implementations of the API have been completed.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No.

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

Since the controller for this feature will not be implemented in-tree, it will be
the responsibility of the implementations to report metrics as they see fit.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [x] API .status
  - Condition name: The Condition name will not be standardized in `alpha` however
  implementations are given the `status` field to report what they see fit.

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

Specific SLOs will be determined by the implementations.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [x] Other (treat as last resort)
  - Details: N/A since the indicators will vary based on the implementation.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

A metric describing the time it takes for the implementation to program the rules
defined in an AdminNetworkPolicy could be useful.  However, some implementations
may struggle to report such a metric.

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

No.

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

No.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

- API Type: AdminNetworkPolicy
- Supported number of objects per cluster: The total number of AdminNetworkPolicies
will not be limited. However, it is important to remember that the only users
creating ANPs will be Cluster-Admins, of which there should only be a handful. This
will help limit the total number ANPs being deployed at any given time.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

This depends on the implementation, specifically based on the API used to program
the AdminNetworkPolicy rules into the data-plane.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

Not in any in-tree components, resource efficiency will need to be monitored by the
implementation.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

N/A for `alpha` release.

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

N/A for `alpha` release.

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A for `alpha` release.

## Implementation History

- 2021-02-18 - Created initial PR for the KEP

## Drawbacks

Securing traffic for a cluster for administrator's use case can get complex.
This leads to introduction of a more complex set of APIs which could confuse
users.

## Alternatives

Following alternative approaches were considered as this KEP has been iterated upon:

### NetworkPolicy v2

A new version for NetworkPolicy, v2, was evaluated to address features and use cases
documented in this KEP. Since the NetworkPolicy resource already exists, it would be
a low barrier to entry and can be extended to incorporate admin use cases.
However, this idea was rejected because the NetworkPolicy resource was introduced
solely to satisfy a developers intent. Thus, adding new use cases for a cluster admin
would be contradictory. In addition to that, the administrator use cases are mainly
scoped to the cluster as opposed to the NetworkPolicy resource, which is `namespaced`.

### Empower, Deny, Allow action based CRD

Alternatively, AdminNetworkPolicy can have `Empower` (as opposed to `Pass`),
`Deny` or `Allow` as the action of each rule.

In terms of precedence, the aggregated `Empower` rules (all AdminNetworkPolicy
rules with action `Empower` in the cluster combined) should be evaluated before
aggregated AdminNetworkPolicy `Deny` rules, followed by aggregated AdminNetworkPolicy
`Allow` rules, followed by NetworkPolicy rules in all Namespaces. As such, the
`Empower` rules have the highest precedence, which shall only be used to provide
exceptions to deny rules. The `Empower` rules do not guarantee that the traffic
will not be dropped: it simply denotes that the packets matching those rules can
bypass the AdminNetworkPolicy `Deny` rule evaluation. This idea was outvoted
by the `Pass` action during sig-networkpolicy meetings, as most members find the
`Empower` keyword confusing, and using an 'action' to provide exceptions to certain
rule feels counter-intuitive.

#### ClusterDefaultNetworkPolicy resource

Instead of using the `Baseline` action to set cluster default rules, the authors
of this KEP also considered using an entirely separate resource named
ClusterDefaultNetworkPolicy. A ClusterDefaultNetworkPolicy resource will help the
administrators set baseline security rules for the cluster, i.e. a developer CAN
override these rules by creating NetworkPolicies that applies to the same workloads
as the ClusterDefaultNetworkPolicy does.

ClusterDefaultNetworkPolicy works just like NetworkPolicy except that it is cluster-scoped.
When workloads are selected by a ClusterDefaultNetworkPolicy, they are isolated except
for the ingress/egress rules specified. ClusterDefaultNetworkPolicy rules will not have
actions associated -- each rule will be an 'allow' rule.

Aggregated NetworkPolicy rules will be evaluated before aggregated ClusterDefaultNetworkPolicy
rules. If a Pod is selected by both, a ClusterDefaultNetworkPolicy and a NetworkPolicy,
then the ClusterDefaultNetworkPolicy's effect on that Pod becomes obsolete.
In this case, the traffic allowed will be solely determined by the NetworkPolicy.

This idea was eventually abandoned due to several reasons:
1. Two separate resources make it harder to reason about effect of aggregated rules.
2. It is confusing that one cluster level resource has implicit isolation and
   the other does not.

### Single CRD with DefaultRules field

This alternate proposal was a hybrid approach, where in the AdminNetworkPolicy
resource (introduced in the proposal) would include additional fields called `defaultIngress`
and `defaultEgress`. These defaultIngress/defaultEgress fields would be similar in structure
to the ingress/egress fields, except that the default rules will not have `action` field.
All default rules will be "allow" rules only, similar to K8s NetworkPolicy. Presence of
at least one `defaultIngress` rule will isolate the `appliedTo` workloads from accepting
any traffic other than that specified by the policy. Similarly, the presence of at least
one `defaultEgress` rule will isolate the `appliedTo` workloads from accessing any other
workloads other than those specified by the policy. In addition to that, the rules specified
by `defaultIngress` and `defaultEgress` fields will be evaluated to be enforced after the
K8s NetworkPolicy rules, thus such default rules can be overridden by a developer written
K8s NetworkPolicy.

### Single CRD with IsOverrideable field

Another alternative for separating non-overridable guardrail rules and overridable
baseline rules is to introduce a `IsOverridable` field in ANP ingress/egress rules:

```golang
type AdminNetworkPolicyIngress/EgressRule struct {
	Action        RuleAction
	IsOverridable bool
	Ports         []networkingv1.NetworkPolicyPort
	From/To       []networkingv1.AdminNetworkPolicyPeer
}
```

If `IsOverridable` is set to false, the rules will take higher precedence than the
Kubernetes Network Policy rules. Otherwise, the rules will take lower precedence.
Note that both overridable and non overridable cluster network policy rules have explicit
allow/ deny rules. The precedence order of the rules is as follows:

`AdminNetworkPolicy` Deny (`IsOverridable`=false) > `AdminNetworkPolicy` Allow (`IsOverridable`=false) > K8s `NetworkPolicy` > `AdminNetworkPolicy` Allow (`IsOverridable`=true) > `AdminNetworkPolicy` Deny (`IsOverridable`=true)

As the semantics for overridable Cluster NetworkPolicies are different from
K8s Network Policies, cluster administrators who worked on K8s NetworkPolicies
will have hard time writing similar policies for the cluster. Also, modifying
a single field (`IsOverridable`) of a rule will change the priority in a
non-intuitive manner which may cause some confusion. For these reasons, we
decided not go with this proposal.

### Single CRD with BaselineAllow as Action

We evaluated another single CRD approach with an additional `RuleAction` to cover
use-cases of both `AdminNetworkPolicy` and `ClusterDefaultNetworkPolicy`

In this approach, we introduce a `BaselineRuleAction` rule action.

```golang
type AdminNetworkPolicyIngress/EgressRule struct {
	Action       RuleAction
	Ports        []networkingv1.NetworkPolicyPort
	From/To      []networkingv1.AdminNetworkPolicyPeer
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
