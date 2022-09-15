# KEP-3488: CEL for Admission Control

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Background](#background)
- [Considerations](#considerations)
  - [Admission Webhook Parity](#admission-webhook-parity)
  - [Configurability](#configurability)
  - [Migration](#migration)
  - [Compliance](#compliance)
- [Proposal](#proposal)
  - [Phase 1](#phase-1)
    - [API Shape](#api-shape)
      - [Policy Constraints](#policy-constraints)
      - [Configuration](#configuration)
      - [Match Criteria](#match-criteria)
    - [Singleton policies](#singleton-policies)
    - [Type safety](#type-safety)
      - [Type safety and version skew](#type-safety-and-version-skew)
    - [Failure Policy](#failure-policy)
    - [Safety measures](#safety-measures)
    - [Reporting violations](#reporting-violations)
  - [Phase 2](#phase-2)
    - [Namespace scoped policy configuration](#namespace-scoped-policy-configuration)
    - [CEL expression scoping](#cel-expression-scoping)
    - [Secondary Authz](#secondary-authz)
    - [Exclude and default matching](#exclude-and-default-matching)
    - [Access to namespace metadata](#access-to-namespace-metadata)
    - [Transition rules](#transition-rules)
    - [Composition utilities](#composition-utilities)
    - [Safety Features](#safety-features)
    - [Aggregated API servers](#aggregated-api-servers)
  - [User Stories](#user-stories)
    - [Use Case: Standalone Policy](#use-case-standalone-policy)
    - [Use Case: Shared Configuration](#use-case-shared-configuration)
    - [Use Case: Principle of least privilege policy](#use-case-principle-of-least-privilege-policy)
    - [Use Case: Validating native type with new field (version skew case)](#use-case-validating-native-type-with-new-field-version-skew-case)
    - [Use Case: Multiple policy definitions for different versions of CRD](#use-case-multiple-policy-definitions-for-different-versions-of-crd)
    - [Use Case: Migrating from validating webhook to validation policy](#use-case-migrating-from-validating-webhook-to-validation-policy)
  - [Potential Applications](#potential-applications)
    - [Use Case: Build-in admission controllers](#use-case-build-in-admission-controllers)
    - [Use Case: KubeWarden](#use-case-kubewarden)
    - [Use Case: OPA/Gatekeeper](#use-case-opagatekeeper)
    - [Use Case: K-Rail](#use-case-k-rail)
    - [Use Case: Kyverno](#use-case-kyverno)
    - [Use Case: Cloud Provider Extensions](#use-case-cloud-provider-extensions)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [CEL Integration with Kubernetes native types](#cel-integration-with-kubernetes-native-types)
  - [Writing to Status](#writing-to-status)
  - [Versioning](#versioning)
    - [Policy Definition Versioning](#policy-definition-versioning)
    - [Configuration CRD Versioning](#configuration-crd-versioning)
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

This is a proposal for customizable, in-process validation of requests to the
Kubernetes API server as an alternative to validating admission webhooks.

This proposal builds on the capabilities of the [CRD Validation
Rules](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/2876-crd-validation-expression-language)
feature that graduated to beta in 1.25, but with a focus on the policy
enforcement capabilities of validating admission control.

## Motivation

This KEP will lower the infrastructure barrier to enforcing customizable
policies as well as providing primitives that help the community establish and
adhere to best practices of both K8s and its extensions.

Currently the way custom policies are enforced are via admission webhooks.
Admission webhooks are extremely flexible, but have a few drawbacks as compared
to in-process policy enforcement:

- They require building infrastructure to host the admission webhook.
- They contribute to latency by requiring another network hop.
- Due to the extra infrastructure dependencies, webhooks are inherently less
  reliable than in-process webhooks. This forces cluster operators to choose
  between failing closed, which reduces the availability of the cluster as a
  whole and failing open, which limits the efficacy of webhooks for enforcing
  policy.
- Webhooks are operationally burdensome for cluster administrators to
  manage. They must take responsibility for the observability, security and the
  release/rollout/rollback plans for the webhook.

Taking a view of the K8s ecosystem as a whole, it is clear that there is demand
for opinionated policy frameworks.

Pod Security Policies provided this for pods, but encountered a number of
issues. One of which was that it was hard to keep up with community demand for
more control surfaces, and the delay in delivering these control surfaces due to
K8s' rollout period.

Pod Security Admission is a similar solution, but does not attempt to duplicate
the control granularity that PSP provided.

There are numerous in-tree embedded controllers.

The existence of security regimes like the CIS Kubernetes Benchmarks highlight
the values of standardized controls. Automating their enforcement, where
possible, will make it easier for users to lock down their clusters.

With the advent of CRDs, and the drive to make the resources they define
first-class entities, the footprint of Kubernetes extensions is set to grow for
the foreseeable future. This KEP allows authors of such extensions to provide
policy primitives similar to PSP or PSA, putting them on equal footing with
in-tree functionality.

With the reduced infrastructure footprint and demonstrated demand for a
customizable, built-in mechanism for extensible policy, this KEP fills a
community need. It is not intended to replace validating admission webhooks
altogether, however, since these can support functionality that may not make
sense to provide in-tree.

### Goals

- Provide an alternative to webhooks for the vast majority of validating
  admission use cases.
- Provide the in-tree extensions needed to build policy frameworks for
  Kubernetes, again without requiring webhooks for the vast majority of use
  cases.
- Make good use of CEL type checking. This becomes complicated when considering
  that CRD schemas can be changed at any time and that not all fields of built
  in types exist in an older Kubernetes version.
- Provide a polyfill implementation that is supported by the Kubernetes org to
  provide this enhancement functionality to Kubernetes versions where this
  enhancement is not available.
- Provide core functionality as a library so that use cases like GitOps,
  CI/CD pipelines, and auditing can run the same CEL validation checks
  that the API server does.

### Non-Goals

- Build a comprehensive in-tree policy framework. We believe the ecosystem is
  best equipped to explore and develop policy frameworks. We're focusing on
  building an extensible enforcement point into admission control that can be
  used to build policy frameworks.
  - Examples of what policy frameworks might do beyond this enhancement might
    do:
    - Auditing already written resources
    - Building out libraries for code reuse
    - Validating Kubernetes resource YAML files adhere to a policy in a CI/CD pipeline
- Mutation support. While we believe this enhancement could be extended in the
  future to support mutations, we believe it is best handled as a separate
  enhancement. That said, we need to keep mutation in mind in this KEP to ensure
  we design it in such a way that we don't obviously paint ourselves into a
  corner where mutation would be difficult to introduce later.
- Full feature parity with validating admission webhooks. For example, this
  enhancement is not expected to ever support making requests to external
  systems.
- Replace the admission controllers compiled into the API server. 
- Static or on-initialization specification of admission config. This is a
  needed feature but should be solved in a general way and not in this KEP
  (xref: https://github.com/kubernetes/enhancements/issues/1872).

## Background

This is not a new idea, Tristan Swadell (@TristonianJones) explored policy for
Kubernetes using CEL with
[cel-policy-templates-go](https://github.com/google/cel-policy-templates-go),
and Jordan Liggitt (@liggitt) prototyped using CEL for in-process admission
control in Kubernetes in 2020.

[CRD Validation
Rules](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/2876-crd-validation-expression-language)
were implemented as a more constrained subset of this problem and addressed how
to integrate the Kubernetes type system with CEL.

## Considerations

### Admission Webhook Parity

Users of the Kubernetes API are already familiar with
ValidatingWebhookConfigurations. We should strive for consistency with this API
unless there is a good reason to diverge from it.  As a concrete example, we
should provide access to all the information that webhooks have access to (see
[AdmissionRequest](https://github.com/kubernetes/kubernetes/blob/2ac6a4121f5b2a94acc88d62c07d8ed1cd34ed63/staging/src/k8s.io/api/admission/v1/types.go#L40))
and if we provide access to additional information we should extend
AdmissionRequest to include it. (But we need to be careful not to make
AdmissionRequest significantly larger as this will impact the
performance/latency of existing webhooks not leveraging the additional
information.  We should be careful about providing access to cross-object
information like namespace objects to webhooks since they can be stale.)

### Configurability

Consider an admission rule that disallows requests based on a blocklist.

While it is possible to inline a blocklist directly into a CEL expression as a
data literal (`!(object.metadata.name in ['blocked1', 'blocked2'])`), this quickly
becomes problematic:

- Long blocklists become unwieldy quickly in CEL expressions
- A blocklist per scope (e.g. namespace) is inconvenient express and maintain

The need to configure admission rules is common enough (see below use cases)
that we propose configuration be a 1st class concept in the API.

Since all the policy frameworks we have surveyed have configurability as a 1st
class concept, omitting it would result in either the policy frameworks not
adopting this enhancement (and sticking with webhooks) or somehow bypassing the
limitation. One possible approach would be to generate a CEL expression with the
configuration data embedded as a data literal, but we would strongly prefer not
to have policy frameworks generating a CEL expression for each possible
configuration.

### Migration

With webhooks already in large scale use in the Kubernetes ecosystem, we intend
to prioritize capabilities that ease migration. As a concrete example, when
migrating, having fine grained control of what validation messages are returned
and how they are formatted can make a migration far more seamless.

### Compliance

In-process admission control has fundamental advantages over webhooks: it is far
safer to use in a "fail closed" mode. With webhooks, using "fail closed" can
negatively impact cluster availability. But "fail closed" is very valuable when
enforcing compliance (and security). We intend to prioritize capabilities that
make "fail closed" a safe mode of operation. As a concrete example, only
allowing CEL expressions that pass compilation and type checking significantly
reduces the opportunities for runtime errors.

Also, making it possible (and convenient) to declare "zero trust" policies is
important to compliance. By "zero trust", we mean policy rules that apply the
principle of least privilege to newly created resources (e.g. namespaces) where
the policy is initially set to the most restrictive state and can be made less
restrictive via configuration.

## Proposal

Introduce a new `ValidatingAdmissionPolicy` kind to the
admissionregistration.k8s.io group. (suggestions welcome on exact name to use
for kind)

At a high level, the API will support:

- Request matching (similar to the match rules of admission webhooks, RBAC, priority & fairness and Audit)
- CEL rule evaluation (similar to both [CRD Validation
  Rules](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/2876-crd-validation-expression-language)
  and
  [AdmissionRequest](https://github.com/kubernetes/kubernetes/blob/2ac6a4121f5b2a94acc88d62c07d8ed1cd34ed63/staging/src/k8s.io/api/admission/v1/types.go#L40))
- Version conversion support (similar to webhooks MatchPolicy)
- Access the old object (similar to [transition
  rules](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/2876-crd-validation-expression-language#transition-rules)
  and oldObject in AdmissionRequest)
- Configurability, as motivated above.

There are also lots of additional capabilities (response message formatting,
failure policies, type safety, advanced matching rules, ...) that will be discussed
in detail further on in this proposal.

We have divided this proposal into phases, all of which must be completed before
this feature graduates to beta. Our goal is to size the phases so that each
can be completed in a single Kubernetes release cycle.

### Phase 1

#### API Shape

Before getting into all the individual fields and capabilities, let's look at the
general "shape" of the API.

This enhancement introduces a new `ValidatingAdmissionPolicy` kind.

Each `ValidatingAdmissionPolicy` resource defines a admission control policy.
The resource contains the CEL expressions to validate the admission policy and
declares how the admission policy may be configured for use.

For example:

```yaml
# Policy definition
apiVersion: admissionregistration.k8s.io/v1alpha1
kind: ValidatingAdmissionPolicy
metadata:
  name: "validate-xyz.example.com"
spec:
  config:
    group: rules.example.com
    kind: ReplicaLimit
    version: v1
  match:
    rules:
    - apiGroups:   ["apps"]
      apiVersions: ["v1"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["deployments"]
      scope:       "*"
  validations:
    - expression: "object.spec.replicas <= config.maxReplicas"
      # ...other rule related fields here...
```

The `spec.config` field of the `ValidatingAdmissionPolicy` references the
"configuration CRD" for this admission policy. For this example, the
configuration CRD is `ReplicaLimit`.

Note: This is a "Bring Your Own CRD" design. The admission policy definition
author is responsible for providing the `ReplicaLimit` configuration CRD.

`spec.match` specifies what resources this policy is designed to validate. This
also guides type-checking, see below "Type safety" section for details.

To configure an admission policy, "policy configurations" of the configuration
CRD kind are created. For example:

```yaml
# Policy configurations
apiVersion: rules.example.com/v1
kind: ReplicaLimit
metadata:
  name: "replica-limit-test.example.com"
spec:
  match:
    namespaceSelectors:
    - key: environment,
      operator: In,
      values: ["test"]
  maxReplicas: 3
```

This policy configuration limits deployments to a max of 3 repliacas in all
namespaces in the test environment.

An admission policy may have multiple configurations. To configure the "prod"
environment to have a maxReplicas limit of 100, create:

```yaml
# Policy configurations
apiVersion: rules.example.com/v1
kind: ReplicaLimit
metadata:
  name: "replica-limit-prod.example.com"
spec:
  match:
    namespaceSelectors:
    - key: environment,
      operator: In,
      values: ["prod"]
  maxReplicas: 100
```

This design separates admission policy _definition_ from _configuration_. This
has a couple advantages:

- Access to, and delegation of, policy configuration is more manageable. In
  particular, Kubernetes RBAC works well with this design.
- Without the separation, the next most obvious API shape would be to encode
  everything into a single resource, which could easily become very large and
  run into resource size limits.

##### Policy Constraints

Each `ValidatingAdmissionPolicy` resource may optionally set `spec.match`
to constrain the resources it validates. 

- `spec.match` constraints which resources this policy can be applied to. Policy
  configurations each have match rules with further narrow this constraint, but
  cannot expand it. This allows the CEL expressions to make safe assumptions.
  E.g. a CEL expression that is constrained to CREATES and UPDATES of resources
  is guaranteed the root `object` variable is never null, but a CEL expression
  that might need to evaluate a DELETE must handle the root `object` variable
  being null.
- `spec.match` guides CEL expression type checking, see the below "Type safety"
  section for more details.

Alternatives considered:

- Name the constraint field `spec.constraints` instead of `spec.match` to make
  it more distinct from `spec.match` field policy configuration resources. This
  ends up being in inconvenient for singlton policies, which only a single
  `spec.match` field that handles all matching.
- Use a GVK instead of match rules. This works well for type checking, but doesn't
  help establish a the criteria for what resources a policy is designed to apply to.

##### Configuration

The `spec.config` field of the `ValidatingAdmissionPolicy` references the
"configuration CRDs" used to configure the admission policy.

Policy configuration CRDs are the interface between `ValidatingAdmissionPolicy`
authors and the administrators that configure the policies for clusters. The
configuration CRD allows policy definition authors to define how exactly a
`ValidatingAdmissionPolicy` may be configured using a OpenAPIv3 structural
schema.

Policy configuration CRDs have a few restrictions:

- Match criteria fields must be defined and must conform to the expected schema.
  See "Match Criteria" for more details.
- For phase 1, configuration CRDs must be cluster scoped. See phase 2 (below)
  for our plan on namespace scoped configuration CRDs.

If any of the above configuration CRD restrictions are violated, the errors will
be reported in the status of the `ValidatingAdmissionPolicy`. If the match
criteria is malformed, this unfortunately may cause the policy to fail open--
without match criteria, there is no way to know what resources the policy should
match. (See "Writing to Status" for more details about the structure of status
and how it will be updated).

Note that a policy configuration CRD may be referenced by the `spec.config` of
multiple `ValidatingAdmissionPolicy` resources. Each of which may apply
different policy validation rules using the same configuration.  For example,
one `ValidatingAdmissionPolicy` might validate the containers declared in `pods`
while another might validate the containers declared in the `podTemplate` of a
`replicaSet`. As long as both are validating that the same policy configuration
is being enforced, just in different ways, it is reasonable for both to share
the same policy configuration resources.

Alternative considered: Embed the configuration CRD schema directly into
`ValidatingAdmissionPolicy`.

Pros:

- Policy author doesn't need to define the CRD. It can instead be generated for
  them.

Cons:

- This implies that the policy owns the configuration. But we have use cases
  where multiple policies share the same configuration. For this model it seems
  misleading and potentially problematic to have both policies attempting to
  define the same configuration.

##### Match Criteria

During admission, the Kubernetes API server will validate the resource being
admitted against all policy configuration resources that match the resource.

While webhook match rules give a good sense of what types of capabilities might
be needed, they serve a slightly different purpose.  Webhook match rules make it
possible to avoid webhook requests, which incur latency and impact availability,
for resources that don't need to be evaluated by the webhook. CEL expressions
have a comparatively small impact of latency and are in-process (and so do not
have the same impact to availability).

For CEL expressions, the primary benefits of match criteria are:

- Match criteria establishes bounds on what sort of admission requests the CEL
  expressions must consider. The CEL expressions can be written knowing that the
  match criteria has filtered out requests that is does not need to consider.
- Match criteria is available on policy configurations and allows the
  configuration author to further constrain what resources the particular
  configuration applies to.

Matching is performed in quite a few systems across Kubernetes:

| Match type                               | Usages in existing matchers  | Support?                       |
| ---------------------------------------- | ---------------------------- | ------------------------------ |
| namespace selectors                      | WH/Audit/P&F                 | phase 1                        |
| object selectors                         | WH                           | phase 1                        |
| resource apiGroup + resource             | WH/Audit/P&F/RBAC            | phase 1                        |
| apiVersion                               | WH                           | phase 1                        |
| resourceName                             | Audit/RBAC                   | ?                              |
| scope (cluster|namespace)                | WH/P&F                       | phase 1                        |
| operation (HTTP verb)                    | WH/Audit/P&F                 | phase 1                        |
| NonResourceURLs                          | Audit/RBAC/P&F               |                                |
| exclude / skip resource kinds            | Audit (level=None)           | phase 2? see "Exclude and default matching" section |
| default / fallthrough matching           |                              | phase 2? see "Exclude and default matching" section |
| user/userGroup                           | Audit                        | phase 2? see "Secondary Authz" section |
| permissions (RBAC verb)                  | RBAC                         | phase 2? see "Secondary Authz" section |

WH = Admission webhooks, P&F = Priority and Fairness

Match criteria must be declared in the `spec.match` field of policy
configuration resources (see `ReplicaLimit` in the above example) and will be
declared with API types in a format similar to admission webhooks, P&F, RBAC and
Audit. Match criteria will also use ordered list of rules similar to these other
systems.

In order for policy configuration resources to declare match criteria, the
corresponding configuration CRD schema must has a `spec.match` property. This
property must conform to the below "matching schema template". This ensures that
the match criteria is in the format that API server expects (the API server will
be using duck typing here since there is no established way to do polymorphism
across CRDs). The schemas of these fields in the configuration CRD may omit any
optional properties; policy definition authors should only include the parts of
the "match schema template" that are useful for configuring a particular policy.

(Also, by allowing the "matching schema template" in configuration CRDs to be a
omit optional properties, this API is future proofed against the addition of
other match related properties in the future).

"matching schema template":

```go
// TODO: Add this as a struct into the Kubernetes codebase so it can easily be
// imported?
type Match struct {
  Rules []admissionregisterationv1.RuleWithOperations `json:"rules,omitempty" protobuf:"bytes,1,rep,name=rules"`
  NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty" protobuf:"bytes,2,opt,name=namespaceSelector"`
  ObjectSelector *metav1.LabelSelector `json:"objectSelector,omitempty" protobuf:"bytes,3,opt,name=objectSelector"`
  MatchPolicy MatchPolicy // ...
  // TODO: add: exclude, userInfo, permissions? (see above table)
}
```

Example usage:

```go
// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ReplicaLimit struct {
  MaxReplicas int32 `json:"maxReplicas" protobuf:"varint,1,name=maxReplicas"`

  Match Match `json:"match,omitempty" protobuf:"bytes,2,name=match"`
}
```

`MatchPolicy` will work the same as for admission webhooks. It will default to
`Equivalent` but may be set to `Exact`. See "Use Case: Multiple policy
definitions for different versions of CRD" for an explanation of why we need
`MatchPolicy`.

xref:
https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#matching-requests-rules


Alternative considered: Define match criteria with CEL expressions

Pros:

- CEL expressions can be used to declare far more sophisticated match criteria.

Cons:

- CEL expressions can evalaute to an error while API types only evaluate to
  "matches" / "does not match" result.
- We intend to allow policy configuration resources declare match criteria,
  making it difficult to pre-compile or type-check CEL expressions.

#### Singleton policies

For a policy that requires no configuration, we would prefer not to ask users
create configuration CRD and configuration resource that serve no actual
purpose. Instead, we will allow a `ValidatingAdmissionPolicy` to define
a singleton policy, the recipe is:

- Set a `policyType: Singleton` field
- Exclude `spec.config`

For example:

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: "validate-xyz.example.com"
spec:
  policyType: Singleton
  match:
    rules:
    - apiGroups:   ["apps"]
      apiVersions: ["v1"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["deployments"]
      scope:       "*"
  validations:
  - expression: "object.spec.replicas < 100"
```

Adding a `policyType` field makes it easier for a user examining the policy to
observe that the policy is different that normal (configurable) policies. This
is much less obvious if we infer that a policy is a singleton by the state of
the other fields. It also makes validation errors reported when validating a
`ValidatingAdmissionPolicy` much easier to communicate to the policy author.

#### Type safety

To keep failure policy easy to reason about, and to continue to use CEL in a
type-safe way we propose:

- If a ValidatingAdmissionPolicy has a `spec.match` that matches a single GVK,
  the CEL expression is allowed access to the full object in a typesafe way.
  Otherwise, the CEL expression is allowed access to the metadata only.
- If there are any type checking errors (or if the CRD for the matched GVK does
  not exist):
  - When a `ValidatingAdmissionPolicy` is created/update. Any type check errors
    against Kubernetes built-in types result in the create/update request
    failing validation with the type error. 
  - When any CRD a `ValidatingAdmissionPolicy` needs for type chekcing is
    created/updated: The type check errors are detected by an control loop
    watching the CRDs with an informer in the API server, and reported in the
    status of the ValidatingAdmissionPolicy. The policy toggles to a
    "misconfigured" state where all admission requests matching and of the
    policy configurations of the policy fail according to the `FailureMode`.

Example: Typesafe access to object 

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: "validate-xyz.example.com"
spec:
  match:
    expression:
    - apiGroups:   ["apps"]
      apiVersions: ["v1"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["deployments"]
      scope:       "*"
  validations:
    # replicas is accessible because this resource matches only v1 deployments
  - expression: "object.spec.replicas < 100"
```

Example: Typesafe access only to metadata

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: "validate-xyz.example.com"
spec:
  match:
   rules:
    - apiGroups:   ["*"]
      apiVersions: ["*"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["*"]
      scope:       "*"
  validations:
    # minReadySeconds is not accessible because this resource matches multiple types
  - expression: "object.spec.minReadySeconds > 60" # ERROR! Not such field "minReadySeconds".
    # metadata is always accessible
  - expression: "object.name.startsWith('xyz')"
```

Alternative Considered: Do not use CEL type-checking and instead dynamically evaluate
all CEL expressions.

Pros:

- Can write a single CEL expression that processes multiple GVKs
- Possible to write expressions that cope with the absence of fields in schemas

Cons:

- Most errors that would have failed type checking will still result in a
  runtime error when the expression is evaluated. But type-check errors can be
  surfaced much earlier. Also, while type checking is comprehensive, runtime
  errors are dependent on how a CEL expression is evaluated, with some only
  occurring for specific inputs. Operating a policy that uses statically
  type-checked expressions is easier to validate as safe for use with
  `failureMode: Fail` than a dynamicially typed one.

##### Type safety and version skew

Similar to how changes to a CRD schema can impact a policy, changes to
Kubernetes built-in types can also impact policy. If a field is added to a
resource kind at version X, a policy that references the field in expressions
will fail type checking against version X-1.

Use case: ephemeralContainers are added and policy authors would like to apply
the same policy rules to all containers in a pod (containers, initContainer,
ephemeralContainers).

Our plan is to document rollout strategies that minimize risk. Even if we didn't
do type-checking and could check for existence of fields dynamically, good
rollout strategies will be needed.

Alternatives considered:

- Favor dynamic typing to allow CEL expression authors the option of using
  `has(object.field)` checks.
- Guard CEL expressions with match criteria (e.g. only match objects where
  schema has fields {x.y, a.b}). This ends up being a list of all the fields accessed

#### Failure Policy

For in-process validation there is no remote request, so errors should be
deterministic. We believe this significantly reduces the risk of "fail closed"
admission control as compared to webhooks.

Because failure policy is most often selected based on the need to guarantee
enforcement, we propose defaulting failure policy to "fail" and allowing it to
be configured on a per-rule basis:

```yaml
  validations:
    - rule: "object.spec.xyz == configuration.x"
      failurePolicy: Ignore # The default is "Fail"
```

TODO: Also allow for a `failurePolicy` field on policy configuration resources?

TODO: Metric for ignored failures?

#### Safety measures

To prevent clusters from being put into a unustable state that cannot be
recoverd from via the API, admission webhooks are not allowed to match
`ValidatingWebhookConfiguration` and `MutatingWebhookConfiguration` kinds.

We will extend this approach:

- `ValidatingAdmissionPolicy` cannot match `ValidatingAdmissionPolicy`
- `ValidatingWebhookConfiguration` cannot match `ValidatingAdmissionPolicy` or
  `MutatingWebhookConfiguration`.

Note that this does allow `ValidatingAdmissionPolicy` to match
`ValidatingWebhookConfiguration`.

Note: In the future we may further loosen this up and allow admission
configuration to intercept/guard writes to admission configuration while
preventing deadlock - Add feature to configure a set of webhooks to intercept
other webhooks https://github.com/kubernetes/kubernetes/issues/101794.

Alternative considered: Each `ValidatingAdmissionPolicy` has a "level", a
`ValidatingAdmissionPolicy` can match another `ValidatingAdmissionPolicy` of a
higher level. This could be added later.

#### Reporting violations

We need:

- Formatted messages (need support for string formatting. Allow a CEL expression
  to be used to format the whole string? Or allow CEL expressions to evaluate
  template values?). Note that CRD Validation Rules do NOT support formatted
  messages (but if we add support for formatted message for admission control,
  we should consider adding the feature to CRD validation as well).
- severity (e.g. "Warning")
- Audit Annotations - Information recorded to the audit system but not reported
  back in the response.
- status types (e.g. Forbidden can be returned from a webhook using
  HTTP 403) (can a match rule override this? or can it be selected via a CEL
  expression?)
- Similar to how each match has configuration, can each match also select
  the severity of messages? It make it possible to configure a policy as
  warning only when first enabling it. Some policy frameworks support this.

### Phase 2

#### Namespace scoped policy configuration

For phase 1, policy configuration resources were only allowed to be cluster
scoped. We can support namespace scoped policy configuration as follows:

- If the configuration resource is namespace scoped, it implicitly matches
  resources only in the namespace it is in, but may further constrain what
  resources it matches with additional match criteria.

Benefits: Allows policy of a namespace to be controlled from within the
namespace. 

#### CEL expression scoping

CRD validation rules are scoped to the schema at the location in the OpenAPIv3
where they are defined. This make validation rules far easier to author by
eliminating the need to dereference from the root of an object to the field that
needs to be validated. Should we provide something similar?

Use cases:

- Dereferencing to deeply nested fields in CEL. A `scope` field containing a
  field path be sufficient. E.g. `spec.containers[*].image`
- More ambitiously, some way to scope a CEL expression to a type found nested in
  multiple kinds, e.g. "io.k8s.api.core.v1.PodTemplateSpec" or
  "io.k8s.api.core.v1.Container" would help policy authors apply policies more
  broadly and uniformly (e.g. to match initContainers, containers,
  ephemeralContainers).

#### Secondary Authz

kube-apiserver authorizer checks (aka Secondary-authz checks) have been proposed
as a way of doing things like:

- Validate that only a user with a specific permission can set an enum to the
  "HOLD" value.
- Validate that only a controller responsible for a finalizer can remove it from
  the finalizers field.

This could be supported by matching criteria, or via CEL expression access, or both.
 
 Concerns: "Is joe authorized to do this"? That only works for the objects joe
 creates, but not objects that get created on joe's behalf by a controller.
 Ditto for updates. I heard someone cite PSP as an example for why it's needed,
 but IMO that was an anti-pattern of PSP, and one that we explicitly decided to
 omit from PSA

#### Exclude and default matching

Both exclusion rules and a default configuration have useful properties.

- Exclude rules make it easier for policy definition authors to do things like
  exclude kube-system or other special purpose namespaces from policies that apply
  to all "normal" namespaces.
- Default rules make it easier to apply principal of least privilege to
  policies. I.e. ensure that newly created resources default to a least
  privilege state until configured otherwise.

Exclude rules can be modeled after the approach exstablished in Audit where
level=None on a match rule can be used to exlude any rules beneath it. E.g.
`effect: Skip`. (naming TDB).

Default configurations are more difficult. Options:

- Embed the default configuration into the policy definition. Messy to
  implement, less appropriate for use cases where the default configuration is
  intended to be configurable on a per cluster basis since it requires handing
  out the same permissions required to modify the policy definition.
- Policy definition references the default configuration (`defaultConfig: "<configurationResourceName>"`).
  Same permission problem as previous option.
- For cluster scoped policy configuration objects, use a singlton configuration
  resource e.g. "<name-of-policy-definition>-default". Can be combined with
  `defaultConfiguration: true` in place of normal matching criteria to make it
  obvious that resource is a default configuration.
- For namespace scoped policy configuration objects, a singleton doesn't work
  (unless there is a good namespace to put it in that would work for all
  clusters).
  - To always have the default configuration cluster scoped, policy defintions
    could be allowed to reference two configuration CRDs, one for cluster scope
    and another for namespace scope. Having two CRDs is VERY undesirable.

None of these options seem satisfactory to me.

#### Access to namespace metadata

- Namespace labels and annotations are the most commonly needed fields not
  already available in the resource being validated. Note that
  namespaceSelectors already allow matches to examine namespace levels, but we
  also have use cases that need to be able to inspects the fields in CEL
  expressions.

#### Transition rules

- Will provide access to "object" and "oldObject" in CEL expressions. These will
  be the same as in AdmissionReview.
- On CREATE, "oldObject" will be null.
- On DELETE, "object" will be null.

If we add "CEL expression scoping" (see above section), we will also need to
consider how scoped fields are handled for create/update/delete. Note that CRD
validation rules have transition rules which are only evaluated when both "self"
and "oldSelf" are present. 

#### Composition utilities

- Ability to define a sub-expression and then use it in multiple validation
  rules (xref cel-policy-template "terms")

#### Safety Features

- Configurable admission blocking write requests made internally in
  kube-apiserver during server startup (like RBAC default policy reconciliation)
  making it impossible for a server to start up healthy. (This is not specific to CEL?)
- Ability to skip specific resource types - Admission Controller Webhook
  configuration rule cannot exclude specific resources:
  https://github.com/kubernetes/kubernetes/issues/92157

#### Aggregated API servers

TODO: We need to address:

(provided by @liggitt)

- The API server validating/persisting the ValidatingAdmissionPolicy
  instances isn't the same one serving aggregated types, so wouldn't necessarily
  have schema info to check type safety.

- The aggregated API server is responsible for enforcing admission on its custom
  types, so the implementation that reads ValidatingAdmissionPolicy
  instances and enforces them would have to live in k8s.io/apiserver and be
  active in aggregated API servers to enforce admission on aggregated types
  effectively (same as admission webhooks today).

### User Stories

In addition to "User Stores", see below "Potential Applications" for a list of
known applications and their use case requirements.

#### Use Case: Standalone Policy

User wishes to define a simple policy that required no configuration. They don't
want to create configuration CRD since what they're doing can be expressed quite
simply in a single CEL expression.

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: "validate-xyz.example.com"
spec:
  singletonPolicy: true
  match:
    rules:
    - apiGroups:   ["apps"]
      apiVersions: ["v1"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["deployments"]
      scope:       "*"
  validations:
  - expression: "object.spec.replicas < 100"
```

#### Use Case: Shared Configuration

User wishes to define a configuration for a list of banned words that may not be
used in any of a wide range of identifiers in the cluster (resource names,
container names, ...).

- Configuration CRD is defined to hold the list of banned words.
- Multiple policies are defined for different resources. The policies all
  reference the same configuration CRD. 
- Policy must be able to specify it's own matching rules since the configuration
  applies cluster wide.
- A single custom resource is defined with the list of banned words (but has not
  matching rules of its own).


```yaml
apiVersion: rules.example.com/v1
kind: BannedWords
metadata:
  name: "banned-words.example.com"
spec:
  bannedWords:
  - glitter
  - rainbow
```

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: "policy1.example.com"
spec:
  match:
    rules:
    - apiGroups:   ["*"]
      apiVersions: ["*"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["pods"]
      scope:       "*"
  validations:
  - rule: "!object.name in config.bannedWords"
```

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: "policy2.example.com"
spec:
  match:
    rules:
    - apiGroups:   [""]
      apiVersions: ["v1"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["pods"]
      scope:       "*"
  validations:
  - rule: "!object.spec.containers.any(c, c.name in config.bannedNames)"
  - rule: "!object.spec.initContainers.any(c, c.name in config.bannedNames)"
```


Similar Use Case: A cluster administrator wishes to use a single policy
configuration to manage a network policy that must be enforced across multiple
Kubernetes kinds that contain relevant networking fields. It is possible to
implement by having multiple `ValidatingAdmissionPolicy` resources that all
reference the same `spec.config` CRD but that each enforce the policy for a
different Kubernetes network kind.

#### Use Case: Principle of least privilege policy

A cluster administrator would like disallow the use of a list of reserved labels
by default, but allow use of the labels in specific namespaces so long as the
label values are valid. 

- Define the following policy configurations:
  - policy configuration that matches all namespaces with the
    "may-use-reserved-labels" label and checks that the values are valid
  - "default" policy configuration that matches all namespaces without the
    "may-use-reserved-labels" label and disallows any reserved labels.
  - An additional policy can be used validate that only authorized roles set the
    "may-use-reserved-labels".

TODO: can we do better here?

#### Use Case: Validating native type with new field (version skew case)

Policy author wants to write a policy that validates a property of all
containers in a pod, including ephemeralContainers for versions of
Kubernetes where ephemeralContainers are available.

This is the sort of policy where cluster managers would ideally be able to
register the policy to validate ephemeralContainers _before_ upgrading to the
version of Kubernetes where ephemeralContainers are available for use.

TODO: write up the best approach for this. Multiple policies with a clear
 rollout approach to avoid breaking clusters?

#### Use Case: Multiple policy definitions for different versions of CRD

While version conversion allows for single policy definition. Cases for multiple
policy definitions are:

1. A policy author wishes to write a policy for both the v1 and v2 of a CRD
because they wish to avoid incuring a CRD conversion webhook request, which
would happen if they only offered a single policy (at either version).

OR

2. A policy author wishes to write a policy such that it can be evluated
"shift-left" in a pre-submit check.

Proposed solution:

- Use a `matchPolicy: Exact` for the v1 policy.
- use `matchPolicy: Equivalent` and an exclude match rules for `v1` for the
  policy that handles v2+. This way if a v3 is added in the future, the policy
  for v2 applies via version conversion by default.

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: "policy1.example.com"
spec:
  match:
    rules:
    - apiGroups:   ["example.com"]
      apiVersions: ["v1"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["myCRD"]
    matchPolicy: Exact
  validations:
  - rule: "object.v1fieldname == 'xyz'"
```

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: "policy1.example.com"
spec:
  match:
    rules:
    - apiGroups:   ["example.com"]
      apiVersions: ["v1"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["myCRD"]
      exclude: true # TODO: how to support excludes?
    - apiGroups:   ["example.com"]
      apiVersions: ["*"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["myCRD"]
    # matchPolicy: Equivalent is the default
  validations:
  - rule: "object.v2fieldname == 'xyz'"
```

#### Use Case: Migrating from validating webhook to validation policy

Steps:

1. Webhook is configured and in-use.
2. `ValidatingAdmissionPolicy` created with `FailPolicy: Ignore`
3. `ValidatingAdmissionPolicy` is monitored to ensure it behaves the same as te webhook (logs or audit annotations can be used)
4. `ValidatingAdmissionPolicy` is updated to `FailPolicy: Fail`
5. Webhook is configured with `FailPolicy: Ignore` (optional)
5. Webhook configuration is deleted

### Potential Applications

As mentioned earlier, we aim to provide a customizable, in-process validation of
requests to the Kubernetes API server as an alternative to validating admission
webhooks. The current policy enforcement is mainly done through:

- Build-in admission controllers like PSA
- External admission controllers in the ecosystem like K-Rail, Kyverno,
  Kubewarden and OPA/Gatekeeper
- Self developed validating admission webhooks

#### Use Case: Build-in admission controllers

Extending to security use cases beyond what PodSecurityAdmission (replacement of
PSP) provides.

Use cases for extending Pod Security admission:

- Further limitations an CSI volumes
- Limitations on seccomp and AppArmor localhost profiles
- Additional limitations on which UIDs can be used
- Application or namespace specific SELinux restrictions
- Restricting privileged namespaces

#### Use Case: KubeWarden

Kubewarden is a policy engine for Kubernetes. It helps with keeping the
Kubernetes clusters secure and compliant. Kubewarden policies can be written
using regular programming languages or Domain Specific Languages (DSL). Policies
are compiled into WebAssembly modules that are then distributed using
traditional container registries.

- Policy hub for ready to use policies: https://hub.kubewarden.io/
- Policy examples: https://github.com/topics/kubewarden-policy

#### Use Case: OPA/Gatekeeper

Gatekeeper uses the OPA Constraint Framework to describe and enforce policy. A
community-owned library of policies for the OPA Gatekeeper project:
https://github.com/open-policy-agent/gatekeeper-library

#### Use Case: K-Rail

k-rail is a workload policy enforcement tool for Kubernetes.
policy violations examples: https://github.com/cruise-automation/k-rail#viewing-policy-violations

#### Use Case: Kyverno

Kyverno is a policy engine designed for Kubernetes. It can validate, mutate, and
generate configurations using admission controls and background scans.  The
policy examples used in Kyberno: https://kyverno.io/policies/

#### Use Case: Cloud Provider Extensions

PVL Admission controller (which is deprecated) is being replaced by a webhook
(issue, KEP) - requires mutation.

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

### CEL Integration with Kubernetes native types

While implementing [CRD Validation
Rules](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/2876-crd-validation-expression-language),
CEL was integrated with CRD structural schemas and the "unstructured" data
representation. For admission control, we also need CEL to be integrated with
the Kubernetes Go structs used to representative native API types, both for type
checking and for runtime data access.

### Writing to Status

This enhancement proposes using status to of `ValidatingAdmissionPolicy` to
communicate type-checking errors and any other misconfigurations such as CRD not
found errors.

As mentioned in
https://github.com/kubernetes/enhancements/pull/3492#discussion_r964841045,
status on API server configuration objects has been tricky to design in the
past, because of the following:

- multiple active kube-apiservers (sometimes at identical versions, sometimes
  skewed by one version during upgrade)
- multiple active non-kube-apiserver servers (aggregated servers)

As a concrete example, The CRD `NonStructural` status field takes advantage of
the metadata generation field
(https://github.com/kubernetes/kubernetes/commit/2cfc3c69dc7c17b2711af0168f39ed7f515675c2).

We will use a similar approach. we will use `generation` numbers of resources to
determine if an apiserver is observing a state newer than the written status of
a `ValidatingAdmissionPolicy`, and will only update the status if this is true.

An apiserver controller will watch:

- `ValidatingAdmissionPolicy` resources
- All CRDs the CEL expressions need to be type checked against (Configuration
  CRDs and any CRDs matched by the match criteria of the policy)

It will track the last seen `generation` of all these resources in
`ValidatingAdmissionPolicy`:

```
apiVersion: "admissionregisteration/v1alpha1"
kind: "ValidatingAdmissionPolicy"
metadata:
  name: "myPolicy"
  generation: 2
spec:
  ...
  config:
    apiVersion: "example.com/v1"
    kind: "fooLimits"
  ...
status:
  config:
    generation: 5
  matchedCustomResource:
    apiVersion: "example.com/v1"
    kind: "foo"
    generation: 100
```

Any time any of the resources the apiserver controller is watching change, it
will check that:

- last seen ValidatingAdmissionPolicy generation is no older current
- last seen `spec.config` apiVersion/kind if different than current OR last seen
  `status.config.generation` is no older than current
- last seen `status.matchedCustomResource` apiVersion/kind if different than
  current OR last seen `status.matchedCustomResource.generation` is no older
  than current
- At least one of the generations have increased

If all are true, then the controller has observed a forward progress of the
status and should update the status along with any conditions and errors
observed:

```
status:
  ...
  conditions:
    type: "Available" # TODO: pick an appropriate type for broken policies
    status: "False"
    reason: Misconfigured
    message: "Validation expressions contain errors. Config custom resource definition not found."
    ...
  validationErrors:
    - expression: "object.baz > config.min"
      errors:
        - "illegal ..."
        - "no such field ..."
  configCustomResourceDefinitionErrors:
    - "Config custom resource definition not found"
```

Note that write conflicts do not require a retry since the write that caused the
conflict will result in another sync once it is observed.

### Versioning

#### Policy Definition Versioning

As a built-in type, `ValidatingAdmissionPolicy` follows Kubernetes API guidelines.

#### Configuration CRD Versioning

A configuration CFD may offer a new version using the existing CRD schema
versioning and version conversion support. The policy definition can then
migrate from reading the old version to the new version.

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
