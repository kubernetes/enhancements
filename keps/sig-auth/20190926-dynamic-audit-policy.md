---
title: Dynamic Audit Policy
authors:
  - "@shturec"
owning-sig: sig-auth
participating-sigs:
  - sig-apimachinery
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-09-23
status: provisional
see-also:
  - "https://github.com/kubernetes/enhancements/blob/master/keps/sig-auth/0014-dynamic-audit-configuration.md"
  - "https://docs.google.com/document/d/1Mp1E4fEbFiCBmSrAJxZuPrbwjYMZ4dAZC0K4XvSyZFA/edit?usp=sharing"
  - "https://docs.google.com/document/d/12uYuvykipkG96EJ4PsFtvhzgMY6mmsBgX8kNLuOq_RE/edit?usp=sharing"
---

# Dynamic Audit Policy

## Table of Contents
<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Multiple audit stakeholder roles](#multiple-audit-stakeholder-roles)
  - [Insight into cluster activities on the spot](#insight-into-cluster-activities-on-the-spot)
  - [Separation of concerns](#separation-of-concerns)
  - [Comprehensible policies](#comprehensible-policies)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Policy and Rules API Resources](#policy-and-rules-api-resources)
    - [Audit Level](#audit-level)
    - [Stages](#stages)
    - [Request Selection Rules](#request-selection-rules)
  - [Dynamic Audit Policy Runtime](#dynamic-audit-policy-runtime)
    - [Getting Ready for Audit](#getting-ready-for-audit)
    - [Runtime Control on References](#runtime-control-on-references)
    - [Request Auditing](#request-auditing)
  - [Summary of proposed API changes](#summary-of-proposed-api-changes)
  - [Summary of differences to file audit policy and schema](#summary-of-differences-to-file-audit-policy-and-schema)
  - [Compatibility with file audit policy](#compatibility-with-file-audit-policy)
  - [Design requirements and constraints](#design-requirements-and-constraints)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Privilege Escalation](#privilege-escalation)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
- [Implementation History](#implementation-history)
- [API Type Reference](#api-type-reference)
- [Examples](#examples)
  - [Example 1](#example-1)
- [References](#references)
<!-- /toc -->

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

The proposal is to enhance the [Dynamic Audit Backend](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/#dynamic-backend) feature with **audit policy** that has the following characteristics:
- policies are defined with the `auditregistration.k8s.io/v1alpha1` [`AuditSink`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#auditsink-v1alpha1-auditregistration) API, which has a [`Policy`](https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/auditregistration/types.go#L77) structure for that purpose.
- Policies define the default level of audit detail for requests to the API server.
- A level can be assigned also to a subset of all requests that are selected with rules.
- Rules are decoupled from policies into dedicated `auditregistration.k8s.io/v1alpha1` `AuditClass` API objects. 
- An `AuditClass` object can be *referenced* by multiple policies.
- Rules in `AuditClass` resources are typed, conforming to the kinds of requests to the API server (e.g. for resource, cluster resource, url, etc.).
- Each rule is a set of selectors for properties of the request (e.g. users, verbs, namespaces, etc)

The newly proposed policy model is specifically tailored to the dynamic audit backend use case, promoting separation of concerns and policy design efficiency. It retains semantic interoperability with the existing `audit.k8s.io/v1` `Policy` model.

> Note, that this KEP focuses exclusively on dynamic audit policies. For other concerns related to the dynamic audit sink registration feature, please refer to the corresponding [KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-auth/0014-dynamic-audit-configuration.md).

## Motivation

The use of the `kube-apiserver` audit policy command-line flag `--audit-policy-file` to set cluster audit policy is suited primarily for scenarios where exclusive auditing control for the cluster provider is either sufficient or required. The user scenarios and context elaborated in the [design proposal draft](https://docs.google.com/document/d/12uYuvykipkG96EJ4PsFtvhzgMY6mmsBgX8kNLuOq_RE/edit?usp=sharing) reveal actual use cases where this approach has shortcomings. 

### Multiple audit stakeholder roles

In environments, such as managed clusters (e.g. GKE), there are multiple roles that are interested in controlling certain audit aspects that do not necessarily overlap, or at least not completely. When a single policy controlled only by one of the roles is not the actual goal, it is a limitation that cannot address adequately the interests of all auditing stakeholders.

### Insight into cluster activities on the spot

The auditing mechanism is used to monitor and explore the requests of extensions (for example) to the API server. While this is not its primary purpose, it has become a popular solution among developers for debugging for a limited period of time. Other examples for scenarios that involve temporarily increased insight into API server requests are active suspicious activity monitoring and policy tuning. The changes that these scenarios require server restarts and access to the API server, which may not be possible, as discussed in the previous section.

### Separation of concerns

The separation of concerns is important in clusters where multiple roles are involved. The current auditing mechanism is not designed for that and has the problem that too many competencies are implied in a single role. In reality, sometimes neither a single policy is sufficient to meet each stakeholder demands and domain responsibilities, nor a single policy provider can address them efficiently.

### Comprehensible policies

The current policy model is generic, leaving a lot of implicit constraints and rules between the lines and thus being error prone to develop and hard to reason about. It also implies a deep understanding of how request handling works internally, which adds further to the complexity.

### Goals

- A consistent audit policy API designed for dynamic audit backends
- Efficient policy design
- Improved development and operations experience

### Non-Goals

- Remove support for file policy configured statically in the api server
- Automatic migration from statically configured API server policy (`audit.k8s.io/v1` `Policy`) file to the dynamic audit policy API.

## Proposal

> This KEP touches the most important points in the requirements and design of the dynamically registered audit policies feature. For a more comprehensive documentation, please refer to [Dynamic Audit Policy - Design Proposal](https://docs.google.com/document/d/1Mp1E4fEbFiCBmSrAJxZuPrbwjYMZ4dAZC0K4XvSyZFA/edit?usp=sharing#heading=h.he1xvyttymtw). For convenience, there are links in the KEP referencing the corresponding sections in this document.

### User Stories

> Complete reference available at [Design Proposal - User Stories](https://docs.google.com/document/d/1Mp1E4fEbFiCBmSrAJxZuPrbwjYMZ4dAZC0K4XvSyZFA/edit?usp=sharing#heading=h.he1xvyttymtw)

The user stories considered in the proposal elaborate the [ones](https://github.com/kubernetes/enhancements/blob/master/keps/sig-auth/0014-dynamic-audit-configuration.md#user-stories) in the Dynamic Audit Control KEP further into the dynamic policy domain. They address cases in both self-hosted and managed kubernetes clusters (provided e.g. by [GKE](https://cloud.google.com/kubernetes-engine), [AKS](https://azure.microsoft.com/en-us/services/kubernetes-service), [EKS](https://aws.amazon.com/eks), and/or with [Rancher](https://rancher.com), [Gardener](https://gardener.cloud), etc.), taking account for the more fine-grained breakdown of roles in the latter. For example, *Cluster Provider* and *Cluster Administrator* are distinct roles representing respectively the managed service provider and its consumer with highest privileges on the consumers side. Other roles that are considered as audit policy stakeholders are *Operator* and *Developer*.

The user stories that define the scope can be clustered in two epics:
- Dynamic registration
  - parallel audit facilities with independent policies
  - separation of concerns between roles with stake in auditing (either concerning the audit targets or the contribution to setting up a dynamic audit facility)
  - non-disruptive, dynamic operations, ad-hoc auditing
- Policy design
  - manageable rules by decoupling and reuse 
  - policy compositions by reference
  - separation of concerns between roles that design policies, deliver rules and enable audit facilities

### Policy and Rules API Resources

> Complete reference available at [Design Proposal - Audit Policy API](https://docs.google.com/document/d/1Mp1E4fEbFiCBmSrAJxZuPrbwjYMZ4dAZC0K4XvSyZFA/edit?usp=sharing#heading=h.aykhdqf4hni7)

There are two main concepts in the proposed API - *policies* and *rule sets*. Policies are an integral part of the `AuditSink` API supported by the [`Policy`](https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/auditregistration/types.go#L77) structure. A policy assigns level of audit detail to audit events generated requests to the API server. To assign a different level to a subset of the requests, they must be selected using rules. Rules are decoupled from policies into the `AuditClass` kind of resources. An `AuditClass` is a logical grouping of rules for selecting requests. It can be referenced by multiple policies and potentially assigned a different audit level in each.

```yaml
apiVersion: auditregistration.k8s.io/v1alpha1
kind: AuditSink
metadata:
  name: mysink
spec:
  policy:
    level: None # applies to all requests unmatched by rules
    rules:
    - withAuditClass: sensitive-things
      level: Metadata
---
apiVersion: auditregistration.k8s.io/v1alpha1
metadata:
  name: sensitive-things 
kind: AuditClass
spec:
  rules:
  - subjects: 
    - type: UserGroup
      names: ["system:masters"]
    verbs: ["create","patch","update","delete"]
```

> A more comprehensive [example](#example-1) policy and rules manifests designed with the proposed API are available in the Examples section.

The rules decoupling method is similar to RBAC, which binds a request selection rule set (`Role`) to a `Subject` and namespace via a `RoleBinding` runtime object. In the same manner, an audit level is bound (assigned) to a rule set (`AuditClass`) by a `Policy`. 

Decoupling rules enables them to be managed separately and potentially by a different role than the one managing the policies, and with the following benefits:
- **Reuse and maintainability**   
Decoupled rules can be referenced by multiple policies in the same cluster. Compared to the inline policy rule statements and considering the potential multiplicity of the dynamic policies this reduces redundancy and maintenance effort greatly.
- **Repurpose**   
The same `AuditClass` can be assigned different levels in different policies, enabling it to support various scenarios, potentially in parallel. it is the policy designer's decision how the class `AuditClass` is going to be used.
- **Separation of concerns**  
A practical benefit could be that rules can be supplied as part of an application/extension resource manifests bundle directly by the domain knowledge experts, instead of inferred and figured out by the solution composer or operator.

#### Audit Level

The processing of requests to the API server goes through several defined stages, generating audit events. The purpose of a policy is to assign level of audit detail to requests, which is considered when the audit events are generated. The levels range is `None`, `Metadata`, `Request`, `RequestResponse`, having the same semantics as in the [policy file](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/#audit-policy).

In contrast to the policy file schema, the policy API considers a global level of a policy. It is interpreted as a default assignment of audit level to requests not matched by rules. In that way there is always a default *catch-all* rule using the global level assignment. A minimal `auditregistration.k8s.io/v1alpha1` policy therefore looks like this:
```yaml
policy:
  level: Metadata
```
, which translates in `audit.k8s.io/v1` `Policy` terms into:
```yaml
policy:
  rules:
  - level: Metadata
```

#### Stages

Stages refer to defined milestones in the request processing when an audit event is generated with the assigned level of detail. The concept for stages however presents certain challenges to policy designers.

Understanding stages requires deep understanding of the various request processing options and details. The stages are not strictly sequential. For example, request processing enters the `ResponseStarted` stage only for long running operations, which applies only to request with verbs `watch` and/or `proxy` and/or urls such as `pprof` and/or other conditions that are not externalized.
 
Using certain stages with level combination could produce cumbersome results. For example, switching off emit of audit events to a sink could be achieved both by defining `level: None`, or by not providing `stages`. It was easy to produce incoherent, contradicting presets such as `level` that is different from `None` and no `stages` or `stages` set and `level` set to `None`.

The proposed API takes a simpler and less error-prone approach and treats stages strictly *internal*. In contrast to the file policy schema, stages are specified neither in policies, nor in rules. Instead, every rule is *implicitly* assigned a whitelist of stages (`ResponseStarted`, `ResponseComplete`, `Panic`), excluding `RequestReceived`. 

`RequestReceived` is generally not applicable to dynamic audit webhook backends, because they are designed for *non-blocking* (batch) scenarios. `RequestReceived` was added to guarantee an audit event was logged even in a "query of doom" scenario. However, that guarantee only holds if auditing is in *synchronous* mode to ensure the query of doom is not processed until the event is already successfully recorded. In batch mode, the `RequestReceived` event is put in the buffer, but when the API server is taken offline it is never delivered.

The effective stage, for which an audit event was generated is provided by the [`Stage`](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/apis/audit/v1/types.go#L81) property of the `Event` type. 

#### Request Selection Rules

> For a more comprehensive reference to rules concept support, please refer to the [Dynamic Audit Policy - Design Proposal](https://docs.google.com/document/d/1Mp1E4fEbFiCBmSrAJxZuPrbwjYMZ4dAZC0K4XvSyZFA/edit?usp=sharing#heading=h.i02gvql48q0x).

Rules defined by `AuditClass` objects are used to select requests for audit level assignment defined for that class of requests in an `AuditSink` policy. There must be at least one rule, or an error is yielded.

```yaml
apiVersion: auditregistration.k8s.io/v1alpha1
kind: AuditClass
metadata:
  name: sample
spec:
  rules:
  - groupResourceSelectors:
    # Select requests for secrets in namespace “vault” 
    - group: ""
      resources:
      - kind: secrets
      scope: Namespaced
      namespaces:  ["vault"]
```

The request matching rules resemble the same concept in [dynamic admission control](https://kubernetes.io/docs/reference/access-authn-authz/rbac/), but usually target a more broad spectre of resources and includes the subject as selection criteria. Audit rules are also very close to the same concept in [RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/), but in a different use case and treat subjects as yet another selection criteria, equally important to the others.

A rule is a composition of request attribute selectors, each matching a single value or a set of whitelist values to a corresponding request attribute at runtime. A rule is evaluated to select a request only if _all_ its defined attribute selectors have matched successfully the corresponding attributes.

Attribute selectors are four types: subject, verb, resource and non-resource selectors. Requests can be selected by subject and/or verb only. But quite often these two categories are used together either with resource or non-resource selectors. This is why subject and verb selectors are also considered *base* selectors. The two other non-base selector types are specific variants of request selectors that cannot be used together. 
- Subjects selector   
The subjects selector matches the request user or group to a set of subject kinds and whitelists for their names.
subjects:
  ```yaml
  subjects:
  - type: UserGroup 
    names: ["system:masters"]
  - type: User 
    names: ["admin"]
  ```
- Verbs selector   
Verbs selector matches request verb to a set of whitelist verb names. This could be API verbs such as:
  ```yaml
  verbs: ["create","patch","update","delete"]
  ```
  or normal HTTP verbs such as:
  ```yaml
  verbs: ["get","post"]
  ```
- Group resource selectors   
Resource selectors are a set of composite selector definitions that match attributes from resource requests
  ```yaml
  groupResourceSelectors:
  - group: "" # core API group
    resources: 
    - kind: endpoints
    - kind: services
      objectNames: ["my-service"]
    scope: Namespaced
    namespaces: ["my-namespace"]
  ```
  The resources in the selector are a whitelist of resource kinds in the group, that can be specified with scope (namespace, cluster or anywhere) and potentially comprising also a whitelist of object names or subresources per kind.
- Non-resource selectors   
Non resource selectors are designed to be used for anything that does not apply to group resource selectors. They contain a whitelist of url templates that match the request URI path.
  ```yaml
  - nonResourceSelectors:
    urls:
    - "/api*" # Wildcard matching.
    - "/version"
  ```

For usage examples of the selectors types and Golang type reference, refer to the [Examples](#examples) or [API Type Reference](#api-type-reference) sections in this document, or the  [Examples](https://docs.google.com/document/d/1Mp1E4fEbFiCBmSrAJxZuPrbwjYMZ4dAZC0K4XvSyZFA/edit?usp=sharing#heading=h.th00tafe36h) section in the supplementary design proposal.

In contrast to the existing `audit.k8s.io/v1` `Policy` where non-resource request selectors are specified with the same rule model as resource requests selectors, in `AuditClass` objects they are no longer mixed. Having a dedicated selector model makes `AuditClass` objects easier to comprehend and less error-prone, by sparing them from implicit rules such as invalid coexistence of `nonResourceUrl` and `namespace`/`resource` elements.

### Dynamic Audit Policy Runtime

#### Getting Ready for Audit

When an `AuditSink` configuration is registered, its policy is compiled into a list of rules composed of the rules from the referenced `AuditClass` objects with their effective levels assigned. The compiled policy is then applied to the dynamic backend, maintained by the API server audit plugin for the corresponding `AuditSink`. At runtime, the `AuditSink` dynamically adapts to changes in the `AuditSink` policy or the `AuditClass` objects that it references, by recompiling and applying the new effective policy.

> The compiled effective policy is an internal model and the `audit.k8s.io/v1` `Policy` was chosen to play that role. More details on how that dynamic audit machinery plugs in the server auditing are available in the Dynamic Control KEP's [Policy Enforcement](https://github.com/kubernetes/enhancements/blob/f1a799d5f4658ed29797c1fb9ceb7a4d0f538e93/keps/sig-auth/0014-dynamic-audit-configuration.md#policy-enforcement) section. 

#### Runtime Control on References

Introducing a dependency in a `AuditSink` policy to another runtime object (`AuditClass`) by reference, raises concerns about maintaining the consistency of this relationship. For example, what should be the status of an `AuditSink` comprising a policy that refers to a non-existing `AuditClass`. How is this addressed when it is created and when the `AuditClass` is deleted at runtime? 

This proposal borrows conceptually from the approach of `Pod` resources towards used resources, such as secrets. `AuditSink` objects are created even if referenced `AuditClass` objects are not yet available at the time, but the corresponding dynamic backend does not send events to that sink until all references are available. Deleting an `AuditClass` object without removing existing policy references to it is blocked.
Updates to `AuditClass` objects that are referenced in policies at runtime are not blocked. Automatic propagation of changes to referenced policies may actually be an advantage. If that is not a desirable scenario, then controlled access (RBAC) to modifying operations such as `update` could be a viable approach.

#### Request Auditing

Upon an API server request, the compiled policy rules are enforced and in a sequential evaluation, the level of the first rule to match the request applies to the generated audit event. If no defined rule matches the request, the global level applies. Another way to put that is that the global level applies to an implicit default *catch-all* rule.

### Summary of proposed API changes

> The scope of the changes includes the  `auditregistration.k8s.io/v1alpha1` API only
- New `AuditSink` `rules` property   
- New `AuditClass` resource   
- `stages` removed from the API (supported internally only)
- The `RequestStarted` stage is not supported
- Auditing of requests is switched off only with `level: None`

### Summary of differences to file audit policy and schema

- Policies are API server resources (part of the `AuditSink` resource)
- Rules are decoupled form policies into distinct API objects of kind `AuditClass` 
- Rules are typed
- No `omitStages`
- Global policy `level` applying to all unselected requests
- Default, implicit *catch-all* rule that is assigned the global policy level

### Compatibility with file audit policy

The proposed policy is compatible with the existing file policy. Both can co-exist in the same cluster, each assigned to its audit sink type. The file policy is applied to the audit sink that is statically configured with command-line flags for the API server. The dynamic policies are part of the dynamically registered `AuditSink` resources and apply to the webhook backends defined alongside. 

The audit plugin in the API server ensures that each active backend, regardless of its type, will be able to perform auditing operations, independent of the others. Technically, this is achieved by [building](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/server/options/audit.go#L290-L371) a [union](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/audit/union.go#L27-L34) of all configured backends that is seen as a single [Backend](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/audit/types.go#L32-L46) from the audit system sending events for processing to it. In fact, the union backend delegates sequentially the event processing to the wrapped backends. In this way, effectively, it is possible to enable log, webhook and dynamic backends altogether or any combinations of the three.

Currently, the event processing by the union Backend is performed sequentially, which may be revised in the context of isolation of backends. That would rather be done in the scope of [Dynamic Audit Control KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-auth/0014-dynamic-audit-configuration.md).


### Design requirements and constraints

The dynamic audit policy model must conform to the following requirements:
- is concise and easy to reason about at a glance, without implicit constraints, sparing its consumers from any internal details that do not directly support their goals.
- reduces the rule maintenance cost and minimizes the rules redundancy across multiple cluster policies by decoupling and sharing.
- is tailored specifically to the dynamic audit backend context.
- is future proof and can seamlessly adapt to potential changes in the request model or its handling.
- is bi-directionally interoperable with the `audit.k8s.io/v1` `Policy`.
- is a standard server API that can be protected with RBAC to prevent privileged users from hiding malicious actions or DoS the audit endpoint with excessive policies.
- relies on the AuditSink API [design](https://github.com/kubernetes/enhancements/blob/master/keps/sig-auth/0014-dynamic-audit-configuration.md#policy-enforcement), for isolation support in multi-policy clusters (incl. the static policy file). Changes in policy affect only the audit sink, to which it applies even when using shared rules.
- relies on the AuditSink API [design](https://github.com/kubernetes/enhancements/blob/master/keps/sig-auth/0014-dynamic-audit-configuration.md#cluster-scoped-configuration) to prevent disruption of the API server normal operations, when remote mechanisms are involved, by enforcing batching webhook mode for dynamic backends.
- relies on the AuditSink API [design](https://github.com/kubernetes/enhancements/blob/master/keps/sig-auth/0014-dynamic-audit-configuration.md#performance) for protection of the API server against policies that might lead to abusive usage of the feature, e.g. to put excessive load on the API server.
  
The dynamic audit backend sinks operate in non-blocking, asynchronous, batching mode. This leads to the following limitations:
- Restarts of the server pods void the in-memory event batches that have not been sent to the remote webhooks yet, i.e. generated events will be lost for the dynamic sinks.
- The dynamic audit backend cannot block server requests. It is meant primarily to record the activities asynchronously.
- As a consequence of the above two statements, the dynamic audit backend will not record query of doom events, because when the server dies, the batched events are lost anyway. Therefore, the `RequestReceived` stage that is meant to address this case is not considered (on purpose) when generating events for dynamic policies.

### Risks and Mitigations

Already documented in [Dynamic Audit Control KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-auth/0014-dynamic-audit-configuration.md#risks-and-mitigations) and in additon:

#### Privilege Escalation
- DoS the audit endpoint with too many parallel dynamic sinks   
  Mitigation: 
  - limit the access to the dynamic audit API resources
  - server flag for maximum number of audit sinks 
- Privileged user changing policy or rules to hide (leave unaudited) activities.   
  Mitigation:
  - Static policy file is not accessible via API and audits activities on dynamic audit API if that is applicable (e.g. same organization or by contract)

## Design Details

### Test Plan

- Unit tests for functional changes
- Scalability tests for defining the optimal operational conditions and confirming the design

### Graduation Criteria

An alpha version of this is targeted for 1.17.

#### Alpha -> Beta graduation

- Scope confirmed. No significant changes to the context and use cases.
- The API can address a minimal viable part of the scope:
  - Policies with rules control audit events sent to associated sinks, unaffected by and not affecting the statically configured server audit.
  - `AuditClass` rule sets shared in multiple policies.
  - CRUD ops on `AuditClass` and `AuditSink` referencing `AuditClass`es
  - RBAC on `AuditClass` and `AuditSink` for various scenarios
- Functional unit tests:
  - Policies with rules control audit events sent to associated sinks.
  - Conversion between `audit.k8s.io/v1` `Policy` and `auditregistration.k8s.io/v1alpha1`
- Risks assessment
- Plan for post-graduation development and no known showstoppers

Optional (to be discussed):
- Initial policy scalability tests:
  - Addressing policy cases in `AuditSink` scalability tests
  - `AuditClass` scalability concerns

The KEP will be updated when we know the concrete things changing for beta.

#### Beta -> GA Graduation

The GA graduation criteria will be updated upon beta milestone achieved to reflect the potential changes. Preliminary criteria: 
- Examples of real-world usage for the user stories
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

### Upgrade / Downgrade Strategy

The proposal addresses the `auditregistration.k8s.io/v1alpha1` API and targets the 1.17 release. The upgrade / downgrade requirements listed here are only policy related concerns that are acceptable for an alpha API. For a general dynamic audit backend feature upgrade/downgrade please refer to the corresponding [KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-auth/0014-dynamic-audit-configuration.md).

The following changes need to be considered when upgrading:
- Remove the `stages` property from any existing `AuditSink` policies. Stages will be implicitly considered for the effective policy and are fixed to the `Panic`, `ResponseStarted`, `ResponseCompleted` range.

The following changes need to be considered when downgrading:
- Add `stages` with a valid range of values explicitly to any existing `AuditSink` policies, or no audit events will be generated. 
- Remove `rules` from `AuditSink` policies and `AuditClass` objects, if any. Rules will be lost. They are not supported in prior releases.

## Implementation History

- 26-09-2019: First KEP version proposed

## API Type Reference

> The [AuditSink](https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/auditregistration/types.go#L63) type is omitted for brevity. No changes to it.

```go
type AuditSinkSpec struct {
  Policy Policy
  Webhook Webhook
}
type Policy struct {
  // +optional
  Level Level
  // +optional
  Rules  []PolicyRule
}
type PolicyRule struct {
  Level Level
  WithAuditclass string
}
type AuditClass struct {
  metav1.TypeMeta
  // +optional
  metav1.ObjectMeta
  // Spec defines the audit class spec
  Spec AuditSinkSpec
}
type AuditClassSpec struct {
  Rules []Rule
}
type Rule struct {
  BaseSelector
  RequestSelector
}
type BaseSelector struct {
  // +optional
  Subjects []SubjectSelector
  // +optional
  Verbs []string
}
type SubjectSelector struct {
  Type SubjectType 
  Names []string
}
type SubjectType string
const (
  SubjectTypeUser = “User”
  SubjectTypeUserGroup  = “Group”
)
// union
type RequestSelector struct {
  // +optional
  GroupResourceSelectors []GroupResourceSelector
  // +optional
  NonResourceSelectors []NonResourceSelector
}
type GroupResourceSelector struct {
  // +optional
  Group string
  // +optional
  Resources []ResourceSelector
  // +optional
  ScopeSelector
}
type ResourceSelector struct {
  Kind string
  // +optional
  Subresources []string
  // +optional
  ObjectNames []string
}
// union
type ScopeSelector struct {
  // +unionDiscriminator
  Scope ScopeType
  // +optional
  Namespaces []NamespaceSelector
}
type ScopeType string
const (
  ScopeTypeAny = “Any” // default
  ScopeTypeCluster = “Cluster”
  ScopeTypeNamespaces = “Namespaced”
)
type NamespaceSelector struct {
  Name string
}
type NonResourceSelector struct {
  // +optional
  URLs []string
}
```

## Examples

### Example 1
```yaml
apiVersion: auditregistration.k8s.io/v1alpha1
kind: AuditSink
metadata:
  name: mysink
spec:
  policy:
    level: Request # applies to all requests unmatched by rules
    rules:
    - withAuditClass: sensitive-things
      level: Metadata      
    - withAuditClass: noisy-lowrisk-things
      level: None
---
apiVersion: auditregistration.k8s.io/v1alpha1
metadata:
  name: sensitive-things 
kind: AuditClass
spec:
  rules:
  - subjects:
    - type: UserGroup
      names: ["system:masters"]
    verbs: ["create","patch","update","delete"]
  - groupResourceSelectors:
    - group: ""
      resources:
      - kind: secrets
      - kind: configmaps
      scope: Namespaced
      namespaces:
      - name: kube-system
---
apiVersion: auditregistration.k8s.io/v1alpha1
kind: AuditClass
metadata:
  name: noisy-lowrisk-things
spec:
  rules:
  - groupResourceSelectors:
    - group: ""
      resources: 
      - kind: configmaps
        objectNames: ["controller-leader"]
  - subjects:
    - type: Users
      names: ["system:kube-proxy"]
    verbs: ["watch"]
    groupResourceSelectors:
    - group: ""
      resources: 
      - kind: endpoints
      - kind: services
  - subjects:
    - type: UserGroup
      names: ["system:authenticated"]
    nonResourceSelectors:
      urls:
      - "/api*"
      - "/version"
```

## References

- [Dynamic Audit Policy Design Proposal](https://docs.google.com/document/d/1Mp1E4fEbFiCBmSrAJxZuPrbwjYMZ4dAZC0K4XvSyZFA/edit?usp=sharing)
- [Dynamic Audit Policy Design Draft (with comments)](https://docs.google.com/document/d/12uYuvykipkG96EJ4PsFtvhzgMY6mmsBgX8kNLuOq_RE/edit?usp=sharing)
- [Dynamic Audit Control KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-auth/0014-dynamic-audit-configuration.md)
- [Dynamic Audit Backend documentation at kubernetes.io](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/#dynamic-backend)
- [`auditregistration.k8s.io/v1alpha1` `AuditSink` API reference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#auditsink-v1alpha1-auditregistration-k8s-io)
- [`auditregistration.k8s.io/v1alpha1` Go types reference](https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/auditregistration/types.go)
- [`audit.k8s.io/v1` policy file schema Go types reference](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/apis/audit/v1/types.go#L182-L257)
- [Audit decorator implementation for API server endpoint request handlers](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/endpoints/filters/audit.go#L37-L112)
- [Static policy runtime rule checker (matches policy rules to request)](https://github.com/kubernetes/kubernetes/blob/34db57b0071aa62f546020ad4d7cb603196dd0d7/staging/src/k8s.io/apiserver/pkg/audit/policy/checker.go#L78-L115)
- [Dynamic Audit Policy API out-of-tree implementation](https://github.com/kubernetes/kubernetes/compare/master...shturec:dynamicpolicyapi)