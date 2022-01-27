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
  - [Scalability](#scalability)
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
  It has no concept of explicit "deny" rules, because the application deployer can
  simply refrain from allowing the things they want to deny.
- It doesn't include priorities, or any concept of one user "outranking" another
  and being able to override their policies, because a given application is expected
  to only have policies created by a single user (the one deploying the application).
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
secure K8s clusters, we propose to introduce the following resource
under `policy.networking.k8s.io` API group:
- AdminNetworkPolicy

### AdminNetworkPolicy resource

An AdminNetworkPolicy (ANP) resource will help the administrators:
1. Set strict security rules for the cluster, i.e. a developer CANNOT override
these rules by creating NetworkPolicies that applies to the same workloads as the
AdminNetworkPolicy does.
2. Set baseline security rules that describes default connectivity for cluster
workloads, which CAN be overridden by developer NetworkPolicies if needed.

#### Actions

Unlike the NetworkPolicy resource in which each rule represents an allowed
traffic, AdminNetworkPolicy will enable administrators to set `Pass`,
`Deny` or `Allow` as the action of each rule. AdminNetworkPolicy rules should
be read as-is, i.e. there will not be any implicit isolation effects for the Pods
selected by the AdminNetworkPolicy, as opposed to what NetworkPolicy rules imply.

- Pass: Traffic that matches a `Pass` rule will skip all further rules from all 
  positive priority ANPs and instead be enforced by the K8s NetworkPolicies. 
  If there is no K8s NetworkPolicy rule match, and no ANP priority "0" rule 
  match, traffic will be allowed.
- Deny: Traffic that matches a `Deny` rule will be dropped. 
- Allow: Traffic that matches an `Allow` rule will be allowed.

AdminNetworkPolicy `Deny` rules are useful for administrators to explicitly
block traffic with malicious in-cluster clients, or workloads that pose security risks.
Those traffic restrictions can only be lifted once the `Deny` rules are deleted, 
modified by the admin, or overridden by a higher priority rule.

On the other hand, the `Allow` rules can be used to call out traffic in the cluster
that needs to be allowed for certain components to work as expected (egress to
CoreDNS for example). Those traffic should not be blocked when developers apply
NetworkPolicy to their Namespaces which isolates the workloads.

AdminNetworkPolicy `Pass` rules allows an admin to delegate security posture for
certain traffic to the Namespace owners by overriding any lower priority Allow or 
Deny rules. For example, allowing intra-tenant traffic can be delegated to tenant 
admins explicitly by the cluster admin with the use of `Pass` rules.

#### Priority

The policy instances will be ordered based on the numeric priority assigned to the
ANP. `Priority` is a 32 bit integer value, where a larger number corresponds to 
a higher precedence. The lowest "regular" value (see below for 
special cases) is "1", which corresponds to the lowest "importance". High values 
have higher importance. For alpha, this API defines "1000" as the maximum value, 
but this may be revisited as the proposal advances. For future-safety, clients may 
assume that higher values will eventually be allowed, and simply treat it as an int32.
Any positive priority will have higher precedence over the namespaced NetworkPolicy 
instances in the cluster, unless the traffic matches on a `Pass` rule that allows 
it to bypass any lower positive priority ANP rules.

Additionally, the special priority "0" can be used in the priority field to indicate
that the rules in that policy instance shall be created at a priority lower than the
Namespaced NetworkPolicies. 

The relative precedence of the rules within a single ANP object (all of which 
share a priority) will be determined by the order in which the rule is written. 
Thus, a rule that appears at the top of the ingress/egress rules would take the 
highest precedence. The maximum number of rules, which will be calculated as the 
total summation of the AdminNetworkPolicyIngressRules and AdminNetworkPolicyEgressRules
in a single ANP instance, will be 100.

Conflict resolution: Two policies are considered to be conflicting if they are assigned the
same `priority` and apply to the same resources or a union of resources. In order to avoid such conflicts, 
we propose to include tooling for ANP resources to help alert the admin to potentially ambiguous ANP
priority scenarios, more details in [risks and mitigation](#risks-and-mitigation). However, ultimately
it will be the job of the network policy implementation to decide how to handle overlapping priority 
situations. 

#### Rule Names

In order to help future proof the ANP api, a built in mechanism to identify each
allow/deny/pass rule is required. Such a mechanism will help administrators organize 
and identify individual rules within an AdminNetworkPolicy resource.
We propose to introduce a new string field, called `name`, in each `AdminNetworkPolicy`
ingress/egress rule. Currently the `name` of a rule is optional and does not need to 
be unique, due to the fact that there's no easy way to validate this in-tree. It 
is suggested that implementations provide a mechanism to validate any name 
duplications.

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

|                          | AdminNetworkPolicy                                                                                             | K8s NetworkPolicies                                                                |
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

For example, in the case where a positive priority AdminNetworkPolicy rule,
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
API group.  

```golang

// AdminNetworkPolicy describes cluster-level network traffic control rules
type AdminNetworkPolicy struct {
  metav1.TypeMeta
  metav1.ObjectMeta

  // Specification of the desired behavior of AdminNetworkPolicy.
  Spec     AdminNetworkPolicySpec

  // ANPStatus is the status to be reported by the implementation, this is not 
  // standardized in alpha and consumers should report what they see fit in 
  // relation to their AdminNetworkPolicy implementation
  // +optional 
  Status  AdminNetworkPolicyStatus
}

type AdminNetworkPolicyStatus struct { 
  conditions   []metav1.Condition  
}

// AdminNetworkPolicySpec provides the specification of AdminNetworkPolicy
type AdminNetworkPolicySpec struct {
  // Priority is an int32 value bound to 0 - 1000, the lowest positive priority, 
  // "1" corresponds to the lowest importance, while higher priorities have 
  // higher importance. An ANP with a priority of "0" will be evaluated after all 
  // positive priority AdminNetworkPolicies and standard NetworkPolicies.
  // The Priority for an ANP must be set 
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
// Exactly one of the selector pointers should be set.  
type AdminNetworkPolicySubject struct {
  NamespaceSelector         *metav1.LabelSelector
  NamespaceAndPodSelector   *NamespaceAndPodSelector
}

// NamespaceAndPodSelector allows the user to select a Namespace(s) and optionally 
// select a given set of pod(s) in that namespace(s)
type NamespaceAndPodSelector struct {
  // This field follows standard label selector semantics; if present but empty, 
  // it selects all Namespaces.  
  NamespaceSelector   metav1.LabelSelector

  // Used to explicitly select pods within a namespace; if present but empty, 
  // it selects all Pods.  
  PodSelector         metav1.LabelSelector
}

// AdminNetworkPolicyIngressRule describes an action to take on a particular 
// set of traffic destined for pods selected by an AdminNetworkPolicy's 
// Subject field. The traffic must match both l4Selector and from. 
type AdminNetworkPolicyIngressRule struct {
  // Name is an identifier for this rule.
  // +optional
  Name         string

  // Action specifies whether this rule must pass, allow or deny traffic. 
  // Allow: allows the selected traffic 
  // Deny: denies the selected traffic
  // Pass: allows the selected traffic to skip and remaining positive priority ANP rules 
  // and be delegated by K8's Network Policy. 
  Action       AdminNetPolRuleAction
  
  // L4Selector allows for matching traffic based on L4 constructs. 
  L4Selector   AdminNetworkPolicyL4Selector

  // List of sources from which traffic will be allowed/denied/passed to the entities 
  // selected by this AdminNetworkPolicyRule. Items in this list are combined using a logical OR 
  // operation. If this field is empty, this rule matches no sources. 
  // If this field is present and contains at least one item, this rule
  // allows/denies/passes traffic from the defined AdminNetworkPolicyPeer(s)
  From         []AdminNetworkPolicyPeer
}

// AdminNetworkPolicyEgressRule describes an action to take on a particular 
// set of traffic originating from pods selected by a AdminNetworkPolicy's 
// Subject field. The traffic must match both l4Selector and to. 
type AdminNetworkPolicyEgressRule struct {
  // Name is an identifier for this rule. 
  // +optional
  Name         string

  // Action specifies whether this rule must pass, allow or deny traffic. 
  // Allow: allows the selected traffic 
  // Deny: denies the selected traffic
  // Pass: allows the selected traffic to skip and remaining positive priority AMP rules 
  // and be delegated by K8's Network Policy. 
  Action       AdminNetPolRuleAction

  // L4Selector allows for matching on traffic based on L4 constructs. 
  L4Selector   AdminNetworkPolicyL4Selector

  // List of destinations to which traffic will be allowed/denied/passed from the entities 
  // selected by this AdminNetworkPolicyRule. Items in this list are combined using a logical OR 
  // operation. If this field is empty, this rule matches no destinations. 
  // If this field is present and contains at least one item, this rule
  // allows/denies/passes traffic to the defined AdminNetworkPolicyPeer(s)
  To           []AdminNetworkPolicyPeer
}

// AdminNetworkPolicyL4Selector handles selection of traffic for based on L4 
// constructs. Exactly one of the fields must be defined.
type AdminNetworkPolicyL4Selector struct { 
  // AllPorts cannot be "false" when it is set 
  // AllPorts allows the user to select all ports for all protocols, thus not 
  // selecting traffic based on L4 principles.
  // If "true" then all ports are selected for the all protocols
  AllPorts   *bool

  // List of ports for outgoing traffic.
  // Each item in this list is combined using a logical OR. If this field is
  // empty or missing, this rule matches no ports (traffic not restricted by port).
  // If this field is present and contains at least one item, then this rule 
  // allows/denies/passes traffic only if the traffic matches at least one port in the list.
  // +optional
  Ports      []AdminNetworkPolicyPort 
}


// AdminNetworkPolicyPort describes a port to select
type AdminNetworkPolicyPort struct {
  // The protocol (TCP, UDP, or SCTP) which traffic must match. If not specified, this
  // field defaults to TCP.
  // +optional
  Protocol   *v1.Protocol

  // The port on the given protocol. This can either be a numerical or named
  // port on a pod. If this field is not provided, this matches no port names and
  // numbers.
  // If present, only traffic on the specified protocol AND port will be matched.
  // +optional
  Port       *intstr.IntOrString
    
  // If set, indicates that the range of ports from port to endPort, inclusive,
  // should be allowed by the policy. This field cannot be defined if the port field
  // is not defined or if the port field is defined as a named (string) port.
  // The endPort must be equal or greater than port.
  // This feature is in Beta state and is enabled by default.
  // It can be disabled using the Feature Gate "NetworkPolicyEndPort".
  // +optional
  EndPort     *int32 
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

// AdminNetworkPolicyPeer defines an in-cluster peer to allow traffic to/from. 
// Exactly one of the selector pointers should be set for a given peer. 
type AdminNetworkPolicyPeer struct {
  Namespaces       *NamespaceSet
  NamespacedPods   *NamespaceAndPodSet
}

// NamespaceSet defines a flexible way to select Namespaces in a cluster.
// Exactly one of the selectors should be set.  If a consumer observes none of 
// its fields are set, they should assume an option they are not aware of has 
// been specified and fail closed.
type NamespaceSet struct {
  // Self cannot be "false" when it is set.  
  // If Self is "true" then all pods in the subject's namespace are selected.
  Self                *bool
  // NotSelf cannot be "false" when it is set.  
  // if NotSelf is "true" then all pods not in the subject's Namespace are selected.
  NotSelf             *bool
  // NamespaceSelector is a labelSelector used to select Namespaces, This field 
  // follows standard label selector semantics; if present but empty, it selects 
  // all Namespaces. 
  NamespaceSelector   *metav1.LabelSelector
  // SameLabels is used to select a set of Namespaces that share the same label(s).  
  // To be selected a Namespace must have all of the labels defined in SameLabels, 
  // If Samelabels is Empty then nothing is selected
  SameLabels          []string
  // NotSameLabels is used to select a set of Namespaces that do not have a set 
  // of label(s). To be selected a Namespace must have none of the labels defined 
  // in NotSameLabels. If NotSameLabels is empty then nothing is selected
  NotSameLabels       []string
}

// PodSet defines a flexible way to select pods in a cluster. Exactly one of the 
// selectors should be set.  If a consumer observes none of its fields are set, 
// they should assume an option they are not aware of has been specified and fail closed.
type PodSet struct { 
  // PodSelector is a labelSelector used to select Pods, This field 
  // follows standard label selector semantics; if present but empty, it selects 
  // all Pods.  
  PodSelector     *metav1.LabelSelector   
  // SameLabels is used to select a set of Pods that share the same label(s).  
  // To be selected a Pod must have all of the labels defined in SameLabels, 
  // If Samelabels is Empty then nothing is selected
  SameLabels      []string
  // NotSameLabels is used to select a set of Pods that do not have a set 
  // of label(s). To be selected a Pod must have none of the labels defined 
  // in NotSameLabels. If NotSameLabels is empty then nothing is selected
  NotSameLabels   []string
}

// NamespaceSetAndPod defines a flexible way to select Namespaces and pods in a 
// cluster.
type NamespaceAndPodSet struct { 
  // Namespaces is used to select a set of Namespaces.  It must be defined 
  Namespaces   NamespaceSet 
  // Namespaces is used to select a set of Pods in the set of Namespaces. It must 
  // must be defined 
  Pods         PodSet 
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
of selectors. Specifically it will allow for failing closed in the event a CNI does 
not implement a defined selector.  For example, If a new type (`NamespaceSetAndService`) 
was added to the `AdminNetworkPolicyPeer` struct, and an implementation had not 
yet implemented support for such a selector, an ANP using the new selector would 
have no effect since the implementation would simply see an empty `AdminNetworkPolicyPeer` 
object. 

#### Further examples utilizing the self field for `NamespaceSet` objects

__Self:__
This is a special strategy to indicate that the rule only applies to the Namespace for
which the ingress/egress rule is currently being evaluated upon. Since the Pods
selected by the AdminNetworkPolicy `subject` could be from multiple Namespaces,
the scope of ingress/egress rules whose `self=true` will be the Pod's
own Namespace for each selected Pod.
Consider the following example:

- Pods [a1, b1], with labels `app=a` and `app=b` respectively, exist in Namespace x.
- Pods [a2, b2], with labels `app=a` and `app=b` respectively, exist in Namespace y.

```yaml
apiVersion: policy.networking.k8s.io/v1alpha1
kind: AdminNetworkPolicy
spec:
  priority: 100
  subject:
    namespaceSelector: {}
  ingress:
  - action: Allow
    from:
    - namespacedPods:
        namespaces: 
          self: true 
        pods: 
          podSelector:
            matchLabels:
              app: b
    l4Selector:
      allPorts: true 
```

The above AdminNetworkPolicy should be interpreted as: for each Namespace in
the cluster, all Pods in that Namespace should strictly allow traffic from Pods in
the _same Namespace_ who has label app=b at all ports. Hence, the policy above allows
x/b1 -> x/a1 and y/b2 -> y/a2, but does not allow y/b2 -> x/a1 and x/b1 -> y/a2.

__SameLabels:__
This is a special strategy to indicate that the rule only applies to the Namespaces
which share the same label value. Since the Pods selected by the AdminNetworkPolicy `subject`
could be from multiple Namespaces, the scope of ingress/egress rules whose `scope=samelabels; labels: [tenant]`
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
  priority: 50
  subject:
    namespaceSelector:
      matchExpressions: {key: "tenant"; operator: Exists}
  ingress:
    - action: Pass
      from:
      - namespaces:
          sameLabels: 
          - tenant
      l4Selector:
        allPorts: true
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
    namespaceSelector: 
      matchLabels: 
        kubernetes.io/metadata.name: sensitive-ns
  ingress:
    - action: Deny
      from:
      - namespaces:
         namespaceSelector: {}
      l4Selector: 
        allPorts: true
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
    namespaceSelector: {}
  ingress:
    - action: Allow
      from:
      - namespaces: 
          namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: monitoring-ns
      l4Selector: 
        allPorts: true
  egress: 
    - action: Allow
      to: 
      - namespacedPods:
          namespaces: 
            namespaceSelector:
              matchlabels: 
                kubernetes.io/metadata.name: kube-system 
          pods:   
            podSelector: 
              matchlabels: 
                app: kube-dns
      l4Selector: 
        allPorts: true
        
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
    namespaceSelector: {}
  egress:
  - action: Pass
    to:
    - namespacedPods:
        namespaces:
          namespaceSelector: 
            matchLabels:
              kubernetes.io/metadata.name: bar-ns-1 
        pods: 
          podSelector: 
            matchLabels:
              app: svc-pub
    l4Selector: 
      ports: 
       - protocol: TCP 
         port: 8080
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
    namespaceSelector:
      matchExpressions: {key: "tenant"; operator: Exists}
  ingress:
    - action: Deny
      from:
      - namespaces:
          notSameLabels:
          - tenant
      l4Selector: 
        allPorts: true 
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
    namespaceSelector:
      matchExpressions: {key: "tenant"; operator: Exists}
  ingress:
    - action: Pass
      from:
      - namespaces:
          sameLabels:
          - tenant
      l4Selector: 
        allPorts: true 
    - action: Deny   # Deny everything else other than same tenant traffic
      from:
      - namespaces:
          namespaceSelector: {}
      l4Selector: 
        allPorts: true 
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
kind: AdminNetworkPolicy
metadata: 
  name: baseline-rule-example
spec:
  priority: 0
  subject:
    namespaceSelector: {}
  ingress:
    - action: Deny   # zero-trust cluster default security posture
      from:
      - namespaces:
          namespaceSelector: {}
      l4Selector: 
        allPorts: true 
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
  - Ensure all positive priority ANP rules are evaluated before any NetworkPolicy rules. 
  - Ensure ANP rules with priority="0" are evaluated after any NetworkPolicy rules.

### Graduation Criteria

#### Alpha to Beta Graduation

- Gather feedback from developers and surveys
- At least 2 CNI providers must provide a functional and scalable implementation 
  for the complete set of alpha features.
    - Specifically,  ensure that only selecting E/W cluster traffic is plausible 
      at scale 
- Evaluate the need for multiple `Subject`s per ANP 
- Evaluate "future work" items based on feedback from community and challenges 
  faced by implementors.
- Ensure extensibility of adding new fields. i.e. adding new fields do not "fail-open" 
  traffic for older clients. 

#### Beta to GA Graduation

- At least 4 CNI providers must provide a scalable implementation for the complete set
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

1.24:
- disable by default
- allow gate to enable the feature
- release note

1.26:
- enable by default
- allow gate to disable the feature
- release note

1.28:
- remove gate

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: AdminNetworkPolicy
  - Components depending on the feature gate: kube-apiserver

###### Does enabling the feature change any default behavior?

Enabling the feature by itself has no effect on the cluster.
Creating a AdminNetworkPolicy does have an effect on
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

While focusing only on E/W traffic for now prevents functional overloading of the 
AdminNetworkPolicy API, it also creates some concerns around scalability.  Specifically, 
the implementations will need to keep track of all in cluster IPs in order to only 
select cluster-internal entities while ignoring everything else.  Specific graduation 
requirements have been constructed to ensure this does not become an issue for 
future versions and implementations. 

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
