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
      - [Policy Definitions](#policy-definitions)
      - [Policy Configuration](#policy-configuration)
      - [Match Criteria](#match-criteria)
    - [Decisions and Enforcement](#decisions-and-enforcement)
    - [Failure Policy](#failure-policy)
    - [Safety measures](#safety-measures)
    - [Singleton Policies](#singleton-policies)
    - [Limits](#limits)
  - [Phase 2](#phase-2)
    - [Informational type checking](#informational-type-checking)
    - [Enforcement Actions](#enforcement-actions)
    - [Audit Annotations](#audit-annotations)
    - [Audit Events](#audit-events)
    - [Namespace scoped policy binding](#namespace-scoped-policy-binding)
    - [CEL Expression Composition](#cel-expression-composition)
      - [Use Cases](#use-cases)
      - [Match Conditions](#match-conditions)
      - [Variables](#variables)
    - [Secondary Authz](#secondary-authz)
    - [Access to namespace metadata](#access-to-namespace-metadata)
    - [Transition rules](#transition-rules)
    - [Resource constraints](#resource-constraints)
    - [Safety Features](#safety-features)
    - [Aggregated API servers](#aggregated-api-servers)
    - [CEL function library](#cel-function-library)
    - [Audit Annotations](#audit-annotations-1)
    - [Client visibility](#client-visibility)
    - [Metrics](#metrics)
  - [User Stories](#user-stories)
    - [Use Case: Singleton Policy](#use-case-singleton-policy)
    - [Use Case: Shared Parameter Resource](#use-case-shared-parameter-resource)
    - [Use Case: Principle of least privilege policy](#use-case-principle-of-least-privilege-policy)
    - [Use Case: Validating native type with new field (version skew case)](#use-case-validating-native-type-with-new-field-version-skew-case)
    - [Use Case: Multiple policy definitions for different versions of CRD](#use-case-multiple-policy-definitions-for-different-versions-of-crd)
    - [Use Case: Prevent admission webhooks from matching a reserved namespace](#use-case-prevent-admission-webhooks-from-matching-a-reserved-namespace)
    - [Use Case: Fine grained control of enforcement](#use-case-fine-grained-control-of-enforcement)
    - [Use Case: Migrating from validating webhook to validation policy](#use-case-migrating-from-validating-webhook-to-validation-policy)
    - [Use Case: Pre-existing Deployment triggers rollout long after Pod policy is changed](#use-case-pre-existing-deployment-triggers-rollout-long-after-pod-policy-is-changed)
    - [Use Case: Rollout of a new validation expression to an existing policy](#use-case-rollout-of-a-new-validation-expression-to-an-existing-policy)
    - [Use Case: Canary-ing a policy](#use-case-canary-ing-a-policy)
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
    - [Parameter CRD Versioning](#parameter-crd-versioning)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
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
- [Future Work](#future-work)
- [Alternatives](#alternatives)
  - [Type checking alternatives](#type-checking-alternatives)
  - [Policy definition and configuration separation alternatives](#policy-definition-and-configuration-separation-alternatives)
    - [Alternative: Duck Typed CRDs](#alternative-duck-typed-crds)
    - [Alternative: OpenAPIv3 <code>$ref</code> in CRDs](#alternative-openapiv3--in-crds)
    - [Alternative: <code>/matchRules</code> subresource](#alternative--subresource)
    - [Alternative: <code>PolicyConfiguration</code> kind with config embedded](#alternative--kind-with-config-embedded)
    - [Alternative: Generate CRDs](#alternative-generate-crds)
  - [CEL variables alternatives](#cel-variables-alternatives)
    - [Alternative: Scopes](#alternative-scopes)
  - [Message formatting alternatives](#message-formatting-alternatives)
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
configuration, but CEL is designed for evaluations over structure input, such as
configuration data, and the alternative of generating a CEL expression for each
possible configuration would be sub-optimal from an evaluation and maintenance
standpoint.

### Migration

With webhooks already in large scale use in the Kubernetes ecosystem, we intend
to prioritize capabilities that ease migration. As a concrete example, when
migrating, having fine grained control of what validation messages are returned
and how they are formatted can make a migration far more seamless.

### Compliance

In-process admission control has fundamental advantages over webhooks: it is far
safer to use in a "fail closed" mode because it removes the network as a
possible failure domain. With webhooks, using "fail closed" can negatively
impact cluster availability. But "fail closed" is very valuable when enforcing
compliance (and security). We intend to prioritize capabilities that make "fail
closed" a safe mode of operation. As a concrete example, only allowing CEL
expressions that pass compilation and type checking significantly reduces the
opportunities for runtime errors.

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
  but with access to the data in
  [AdmissionRequest](https://github.com/kubernetes/kubernetes/blob/2ac6a4121f5b2a94acc88d62c07d8ed1cd34ed63/staging/src/k8s.io/api/admission/v1/types.go#L40))
- Version conversion support (similar to admission webhook's MatchPolicy)
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

This API separates policy _definition_ from policy _configuration_ by splitting
responsibilities across resources. The resources involved are:

- Policy definitions (ValidatingAdmissionPolicy)
- Policy bindings (ValidatingAdmissionPolicyBinding)
- Policy param resources (custom resources or config maps)

![Relatinships between policy resources](erd.png)

This separation has already been demonstrated successfully by multiple policy
frameworks (see the survey further down in this KEP). It has a few key
properties:

- Reduces total amount of resource data needed to manage policies:
  - Params can be shared across multiple policies instead of copied. Multiple
    policies can be enforcing different aspects of a "no external connections",
    for example, but can all share the configuration.
  - Policies can be configured in different ways for different use cases without
    having to copy the policy definition.
  - Rollouts and canary-ing can be managed largely via bindings without having
    to copy policy definitions or params.
- Ownership of resources aligns well with typical separation of roles for policy
  management.
- Existing policy frameworks can leverage this design far more easily because it
  aligns with how separation of concerns is expressed by most policy frameworks.

Each `ValidatingAdmissionPolicy` resource defines a admission control policy.
The resource contains the CEL expressions to validate the admission policy and
declares how the admission policy may be configured for use.

For example:

```yaml
# Policy definition
apiVersion: admissionregistration.k8s.io/v1alpha1
kind: ValidatingAdmissionPolicy
metadata:
  name: "replicalimit-policy.example.com"
spec:
  paramKind:
    group: rules.example.com
    kind: ReplicaLimit
    version: v1
  matchConstraints:
    resourceRules:
    - apiGroups:   ["apps"]
      apiVersions: ["v1"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["deployments"]
  validations:
    - name: max-replicas
      expression: "object.spec.replicas <= params.maxReplicas"
      messageExpression: "'object.spec.replicas must be no greater than %d'.format([params.maxReplicas])"
      reason: Invalid
      # ...other rule related fields here...
```

The `spec.paramKind` field of the `ValidatingAdmissionPolicy` specifies the
kind of resources used to parameterize this policy. For this example, it is
configured by `ReplicaLimit` custom resources. Note in this example how the CEL
expression references to the parameters via the CEL `params` variable, e.g.
`params.maxReplicas`.

`spec.matchConstraints` specifies what resources this policy is designed to
validate. This also guides type-checking, see the "Informational type checking"
section for details.

The `spec.validations` fields contain CEL expressions. If an expression
evaluates to false, the validation check is enforced according to the
`enforcement` field.

This is a "Bring Your Own CRD" design. The admission policy definition author is
responsible for providing the `ReplicaLimit` parameter CRD.

To configure an admission policy for use in a cluster, a binding and parameter
resource are created. For example:

```yaml
# Policy binding
apiVersion: admissionregistration.k8s.io/v1alpha1
kind: ValidatingAdmissionPolicyBinding
metadata:
  name: "replicalimit-binding-test.example.com"
spec:
  policyName: "replicalimit-policy.example.com"
  paramRef:
   name: "replica-limit-test.example.com"
   namespace: "default"
  matchResources:
    namespaceSelectors:
    - key: environment,
      operator: In,
      values: ["test"]
```

```yaml
# Policy parameters
apiVersion: rules.example.com/v1
kind: ReplicaLimit
metadata:
  name: "replica-limit-test.example.com"
maxReplicas: 3
```

This policy parameter resource limits deployments to a max of 3 repliacas in all
namespaces in the test environment.

An admission policy may have multiple bindings. To bind all other environments
environment to have a maxReplicas limit of 100, create another `PolicyBinding`:

```yaml
apiVersion: aadmissionregistration.k8s.io/v1alpha1
kind: ValidatingAdmissionPolicyBinding
metadata:
  name: "replicalimit-binding-nontest"
spec:
  policyName: "replicalimit-policy.example.com"
  paramRef:
    name: "replica-limit-test.example.com"
    namespace: "default"
  matchResources:
    namespaceSelectors:
    - key: environment,
      operator: NotIn,
      values: ["test"]
```

```yaml
apiVersion: rules.example.com/v1
kind: ReplicaLimit
metadata:
  name: "replica-limit-clusterwide.example.com"
maxReplicas: 100
```

Bindings can have overlapping match criteria. The policy is evaluated for each
matching binding. In the above example, the "nontest" policy binding could
instead have been defined as a global policy:

```yaml
apiVersion: aadmissionregistration.k8s.io/v1alpha1
kind: ValidatingAdmissionPolicyBinding
metadata:
  name: "replicalimit-binding-global"
spec:
  policyName: "replicalimit-policy.example.com"
  paramRef:
   name: replica-limit-clusterwide.example.com"
  matchResources:
    namespaceSelectors:
    - key: environment,
      operator: Exists
```

With this binding, the test and global policy bindings overlap. Resources
admitted to test environment would then be checked against both policy
configurations.

##### Policy Definitions

Policy definitions are responsible for:

- Defining what validations the policy enforces and how violations are reported 
- Defining how a policy may be configured

Each `ValidatingAdmissionPolicy` resource contains a `spec.matchConstraints` to
declare what resources it validates. This field is required.

- `spec.matchConstraints` constrains which resources this policy can be applied
  to. Policy bindings each have match rules with further narrow this constraint,
  but cannot expand it. This allows the CEL expressions to make safe
  assumptions. E.g. a CEL expression that is constrained to CREATES and UPDATES
  of resources is guaranteed the root `object` variable is never null, but a CEL
  expression that might need to evaluate a DELETE must handle the root `object`
  variable being null. See below "Match criteria" section for how match criteria
  is described.
- `spec.matchConstraints` guides CEL expression type checking, see the below
  "Type safety" section for more details. 

CEL expressions have access to the contents of the `AdmissionReview` type,
organized into CEL variables as well as some other useful variables:

- 'object'
- 'oldObject'
- 'request'
  - 'requestResource' (GVR)
  - 'resource' (GVR)
  - 'name'
  - 'namespace'
  - 'operation'
  - 'userInfo'
  - 'dryRun'
  - 'options'
- 'params' - referred params object, maybe null if no object is referred

See below "Decisions and Enforcement" for more detail about how the
`spec.validations` field works and how violations are reported.

##### Policy Configuration

`ValidatingAdmissionPolicyBinding` resources and parameter CRDs together define how cluster
administrators configure policies for clusters.

Each `ValidatingAdmissionPolicyBinding` contains:

- `spec.policyName` - A reference to the policy being configured
- `spec.matchResources` - Match criteria for which resources the policy should
  validate
- `spec.paramKind` - Reference to the custom resource containing the params to use
  when validating resources 

Example:

```yaml
apiVersion: aadmissionregistration.k8s.io/v1alpha1
kind: ValidatingAdmissionPolicyBinding
metadata:
  name: "xyzlimit-scale.example.com"
spec:
  policyName: xyzlimit-scale.example.com
  paramRef:
    name: xyzlimit-scale-settings.example.com
  matchResources:
    namespaceSelectors:
    - key: environment,
      operator: Exists
```

Each parameter CRD defines the custom resources that are referenced by the
`spec.params` field of `PolicyBinding` resources. Example:

```yaml
apiVersion: rules.example.com/v1
kind: XyzLimit
metadata:
  name: "xyzlimit-settings.example.com"
allowed: ["a", "b", "c"]
banned: ["x", "y", "z"]
xyz:
  fuzzFactor: 0.8
  reticulate: true
```

We will recommend tag / label / annotation on every CRD that is used for policy
params, so that the CRDs to be easily identified/queried.

Note that this API design simplifies well to basic cases:

- For policies that require no parameterization, only the `PolicyBinding` is
needed.
- For policies that are global, see the below "Default Validations" section
  and "singleton policies" for how a policy can be created using a single resource.

See "Alternatives considered" section for rejected alternatives. This design was
selected because:

- The param CRD schema is owned entirely by the policy author.
- Matching criteria is fully defined and validated in the builtin `PolicyBinding` type.
- Type checking is straight forward.
- Policy parameterization is separated from the policy binding, allowing for
  well abstracted parameterization types to be used by applied in different ways
  by multiple policies.
- Make some rollouts easier. E.g. adding a new validation rule to a policy.

Details:

- Namespace Collisions: 
  - This design is similar to `RoleBindings` which use a `roleRef`. If We are to
    support both cluster and namespace scoped policy definitions, we need the
    same structure as `roleRef` for `policyRef`.
  - To address we will require parameter CR names of the form
    `<identifier>.<resourceName>.<apiGroup>`
  - Fix: Require the name to include the parameter type's group and resource.
- Invalid Configurations:
  - With 4 different resources involved in each validating admission policy
    check (policy, binding, parameter CRD, parameter CR), there are many
    combinations of the states of these resources that are invalid. E.g.:
    - parameter CRD does not exist
    - binding refers to policy or parameter resource that does not exist
    - policy CEL expressions references fields that do not exist in parameter
      CRD
  - If a policy binding is in any of these states, it is identified as "invalid"
    and the failure policy is applied.
- Privileges to access policy bindings implies control of policy configurations:
  - To address this, bindings resource will have extra auth check to verify that
    anyone modifying the binding is also permitted to modify the parameters
    resource.
  - We should consider using a verb for secondary authz check that policy
    binding editor has policy parameter edit roles? (tallclair suggested this,
    it has nice properties).

API details:

- Name this `ClusterPolicyBinding` so we can add `PolicyBinding` (namespace
scoped) in the future?

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
- Match criteria is available on policy bindings and allows the binding author
  to further constrain what resources the particular binding applies to.

We did consider not having any "YAML matching" for this feature and instead pushing all matching into CEL. The main deciders for me were:

- Kubernetes already has resource matching as a well established concept
- Match criteria can be built indexed/accelerated/built into decision trees
- Match criteria can evaluate only to true/false. There is no 'error' case to consider
- Match criteria can be used to guide static typing. If the match is for
  v1.Deployment, we know ahead of runtime what type the object variable is

Matching is performed in quite a few systems across Kubernetes:

| Match type                               | Usages in existing matchers  | Support?                               |
| ---------------------------------------- | ---------------------------- | -------------------------------------- |
| namespace                                | Audit, P&F                   | phase 1                                |
| namespace label selectors                | WH                           | phase 1                                |
| label selectors                          | WH                           | phase 1                                |
| annotations                              |                              | No                                     |
| apiGroup + resource                      | WH/Audit/P&F/RBAC            | phase 1                                |
| apiVersion                               | WH                           | phase 1                                |
| resource name                            | Audit/RBAC                   | phase 1                                |
| scope (cluster\|namespace)               | WH/P&F                       | phase 1                                |
| operation (HTTP verb)                    | WH/Audit/P&F                 | phase 1                                |
| exclude                                  | Audit (level=None)           | phase 1                                |
| apiVersion + kind                        |                              | phase 1                                |
| NonResourceURLs                          | Audit/RBAC/P&F               | No                                     |
| user/userGroup                           | Audit                        | phase 1 - request.userInfo.groups      |
| user.Extra                               | (in WH AdmissionReview)      | phase 1 - request.userInfo.extra       |
| permissions (RBAC verb)                  | RBAC                         | phase 2 see "Secondary Authz" section  |

WH = Admission webhooks, P&F = Priority and Fairness

Match criteria must be declared in the `spec.matchResources` field of
`PolicyBinding` resources (see `ReplicaLimit` in the above example) and will be
declared with API types in a format similar to admission webhooks, P&F, RBAC and
Audit, but with improved support for exclude matching.

*Excluding*:

Exclude support makes it possible to do things like validate all namespaces
except `kube-system`. This is difficult to support without direct exclude
support, particularly for 3rd party policy enforcements systems that cannot
assume permission to set labels on kube-system.

Exclude matching will be offered adding `exclude<fieldname>` for match fields
where exclude is appropriate. Fields such as `namespaceSelector` that already
offer exclusion (e.g. via the `NotIn` operator) do not need a corresponding
`exclude<fieldname>`.

E.g.:

```yaml
  excludeNamespaces: ["kube-system"] # excludeNamespaces and namespaces are mutually exclusive
  excludePermissions: ["all-the-superpowers"]
  namespaceSelector: # Already has exclude support via NotIn and DoesNotExist
  - keys: "xyz"
    operator: NotIn
    values: ["1"]
  excludeResourceRules: # excludeResourceRules takes precedent over resourceRules
  - apiGroups:        ["apps"]
    apiVersions:      ["*"]
    operations:       ["*"]
    resources:        ["deployments"]
  resourceRules:
  - apiGroups:        ["apps"]
    apiVersions:      ["*"]
    operations:       ["*"]
    resources:        ["*"]
  ...
```

*Special case: apiGroup + resource + operation matching*

For admission webhooks, at least one `spec.rules` must be declared to state
which apiGroup + resource + operations the webhook operates on. To configure a
webhook to match everything (which is a very bad idea), a match rule would need
to be written to state that, e.g.:

```yaml
spec:
  rules:
  - apiGroups:   ["*"]
    apiVersions: ["*"]
    operations:  ["*"]
    resources:   ["*"]
```

This forces the webhook configuration author to explicitly declare what
they intend to match.

The same principle applies here but with one major difference-- if a policy
definition has a match rules for apiGroup + resource + operation, then all
bindings of that policy are already constrainted to that apiGroup + resource +
operation match.

Take, for example:

```yaml
apiVersion: admissionregistration.k8s.io/v1alpha1
kind: ValidatingAdmissionPolicy
...
spec:
  paramSource: ...
  matchConstraints:
    resourceRules:
    - apiGroups:   ["apps"]
      apiVersions: ["v1"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["deployments"]
```

Since this policy is constrainted to create/update of deployments. Policy
bindings don't need to repeat this constraint. This would be sufficient:

```yaml
apiVersion: admissionregistration.k8s.io/v1alpha1
kind: PolicyBinding
...
spec:
  matchResources:
    namespaceSelectors:
    - key: environment
      operator: Exists
 ...
```

We can enforce this by requiring:

- Policy definitions match rules are validated to match one or more apiGroup +
  resource + operation.
- Policy bindings are not required to match apiGroup + resource + operation
  since the policy definition is already required to do this. But they may
  further narrow down the match. This is useful when rolling out policies. E.g.
  when transitioning a policy from Warn to Deny, being able to narrow down the
  resource match allows for more fine grained rollout steps.

This encourages policy definition authors to consider the assumptions that the
CEL expressions make. If the expressions unconditionally access `object` without
a `has(object)` check, the expression will only ever work on `CREATE` and
`UPDATE` and would fail at runtime on a `DELETE`. It is quite difficult to write
CEL expression that handle all admission requests well, so we want to guide
policy authors toward matching only the requests that they intend to support.
The Kubernetes admission chain also scales better, and is more resiliant, when
matching is precise and the validation expressions don't need to do any
post-matching checks that could have been handled by matching.

Note that if a policy binding is to be applied as broadly as possible (i.e.
everywhere allowed by the policy definition) it must do so by using a wildcard
match rule.

Match Policy:

`MatchPolicy` will work the same as for admission webhooks. It will default to
`Equivalent` but may be set to `Exact`. See "Use Case: Multiple policy
definitions for different versions of CRD" for an explanation of why we need
`MatchPolicy`.

xref:

- https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#matching-requests-rules
- https://kubernetes.io/docs/tasks/debug/debug-cluster/audit/
- https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#resourcepolicyrule-v1beta1-flowcontrol-apiserver-k8s-io
- https://github.com/open-policy-agent/gatekeeper/blob/a1add93b0beb5c48eb92a6a2eb5ee7d21551a1b6/pkg/mutation/match/match.go#L22-L56

#### Decisions and Enforcement

This section focuses on how policies make decisions, how those decisions are
enforced, and how decisions are reported back to the client.

Goals:

- Feature parity with `AdmissionReview`
  - Support allow/deny result, warnings, audit annotations
  - Support for reasons/codes
- Ability to format message strings

Policy definitions:

- Each validation may define a message:
  - `message` - plain string message
  - `messageExpression: "<cel expression>"` (mutually exclusive with `message`)
    - As part of [the KEP update to add expression composition](https://github.com/kubernetes/enhancements/pull/3669/files),
      expressions defined under `variables` will be accessible from `messageExpression`
    - `messageExpression` is a CEL expression and thus factors into the runtime cost limit.
    If the runtime cost limit is exceeded during `messageExpression` execution, then this is logged.
    Whether or not the action is admitted after that depends upon failure policy.
  - If `message` and `messageExpression` are absent, `expression` and `name`
    will be included in the failure message
  - If `messageExpression` results in an error: `expression` and `name` will be
    included in the failure message plus the arg evaluation failure
  - `reason` and/or `code` - these fields have same semantics as admission
      review; the reason clarifies the code but does not override it. If
      `reason` is well known (.e.g "Unauthorizied" is well known to be `code` 401), then the code will be inferred from the `reason` and use of a different code will not be allowed.

Example policy definition:

```yaml
# Policy definition
apiVersion: admissionregistration.k8s.io/v1alpha1
kind: ValidatingAdmissionPolicy
metadata:
  name: "validate-xyz.example.com"
spec:
  ...
  validations:
    - expression: "self.name.startsWith('xyz-')"
      name: name-prefix
      message: "self.name must start with xyz-"
      reason: Unauthorized
    - expression: "self.name.contains('bad')"
      name: bad-name
      message: "name contains 'bad' which is discouraged due to ..."
      code: 400
      reason: Invalid
    - expression: "self.name.contains('suspicious')"
      name: suspicious-name
      message: "'self.name contains suspicious'"
      code: 400
      reason: Invalid
```

xref:

- https://open-policy-agent.github.io/gatekeeper/website/docs/next/violations/

#### Failure Policy

Because failure policy is most often selected based on the need to guarantee
enforcement, we will default failure policy to "fail" and allow it to
be configured on a per-policy basis:

```yaml
apiVersion: admissionregistration.k8s.io/v1alpha1
kind: ValidatingAdmissionPolicy
spec:
  ...
  failurePolicy: Ignore # The default is "Fail"
  validations:
    - expression: "object.spec.xyz == params.x"  
```

#### Safety measures

To prevent clusters from being put into a unustable state that cannot be
recoverd from via the API, admission webhooks are not allowed to match
`ValidatingWebhookConfiguration` and `MutatingWebhookConfiguration` kinds.

We will extend this approach:

- `ValidatingAdmissionPolicy` cannot match
  `ValidatingAdmissionPolicy`/`PolicyBinding`/param resources.
- `ValidatingWebhookConfiguration` cannot match `MutatingWebhookConfiguration`
  or `ValidatingAdmissionPolicy`/`PolicyBinding`/param resources.

Note that this does allow `ValidatingAdmissionPolicy` to match
`ValidatingWebhookConfiguration`.

Note: In the future we may further loosen this up and allow admission
configuration to intercept/guard writes to admission configuration while
preventing deadlock - Add feature to configure a set of webhooks to intercept
other webhooks https://github.com/kubernetes/kubernetes/issues/101794.

Alternative considered: Each `ValidatingAdmissionPolicy` has a "level", a
`ValidatingAdmissionPolicy` can match another `ValidatingAdmissionPolicy` of a
higher level. This could be added later.

#### Singleton Policies

For simple policies that does not refer to a param, a policy can be authored using a
single `ValidatingAdmissionPolicy` resource without a `paramKind` field.

This is only available for cases where there is no need to have multiple
bindings, and where all params can be inlined in CEL.

A "singleton" (aka standalone) policy can be defined as:

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
...
spec:
  # no paramKind
  matchConstraints: ...
  validations:
  - expression: "object.spec.replicas < 100"
```

Note that:

- `spec.paramKind` must be absent
- validation expressions may not refer `params`

Safety features:

- This field may only be set when the policy is created. it may not be set on
  existing policies.
- Any bindings assigned to a singleton policy are considered "misconfigured" and
  apply the `FailurePolicy`.

Reporting/debugging/analysis implications:

- Violations for non-singleton policies will always be reported using a {policy
  definition, policy binding} identifier pair. To be consistent with this,
  singleton policies can be use a {policy definition, policy definition}
  identifier pair. This is a bit verbose but keeps reporting consistent and
  makes tracing a violation back to resources that produced it straight-forward.

#### Limits

We will put limits on:

- Max policy bindings per policy definition
- Max lengths for all lists in match criteria (resourceRules, namespaceSelectors, labelSelectors, ...)
- Per expression CEL cost limits
- Per policy CEL evaluation cost limits

### Phase 2

All these capabilities are required before Beta, but will not be implemented in
the first alpha release of this enhancement due to the size and complexity of
this enhancement.

#### Informational type checking

Some advantages of strongly typed objects and expressions over treating everything as unstructured are:

- Checks against missing/misspelled fields.
  The user may write an expression that refers to a missing/misspelled field that
  does not exist in their test cases but appears later. A type check can detect this kind of error while an evaluation-time check may not.
- Checks against type confusions. Similarly, the user may confuse the type of field but their test cases never touch wrongly typed fields.
- Guard against short-circuit evaluation. The user may make a mistake of one of the mentioned above but the code path is never covered in their test cases;
- Support Kubernetes extensions. For example, IntOrString and map lists.

However, enforcing types for every expression and object is not feasible because of:

- Version skew
- CRDs
- Aggregated API servers

Problem examples:

| Problem                                          | Summary                                                                                                               |
|--------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------|
| version skew: ephemeralContainers case           | New pod field, need to be able to validate in same was containers and initContainers if field exists and is populated |
| version skew: Migration from annotation to field | Need to be able to validate annotation (if present) or field (if it exists and is populated)                          |
| CRD is deleted                                   | Nothing to type check against, but also means there are no coresponding custom resources                              |
| CRD is in multiple clusters, but schema differs  | If policy author is aware of the schema variations, can they write policies that work for all the variations?         |
| Validation of an aggregated API server type      | Main API server does not have type definitions                                                                        |

Until the design is extended to handle these situations, the type checking will remain informational.

Informational type checking will be performed against all expressions where a GVK can be resolved to
type check against. The result of type checking will be part of the status of the performed policy.

For example, accessing an unknown field will result a warning like this.

```yaml
...
status:
  expressionWarnings:
    - expression: "object.replicas > 1" # should be "object.spec.replicas > 1"
      warning: "no such field 'replicas'"
```

#### Enforcement Actions

`ValidatingAdmissionPolicyBinding` resources may control how admission is
enforced. This is performed using a single field. E.g.:

```yaml
apiVersion: admissionregistration.k8s.io/v1alpha1
kind: ValidatingAdmissionPolicyBinding
...
spec:
  validationActions: [Warn, Audit] # required field
```

The enum options will be:

- `Deny`: Validation failures result in a denied request.
- `Warn`: Validation failures are reported as warnings to the client. (xref: [Admisssion Webhook Warnings](https://kubernetes.io/blog/2020/09/03/warnings/#admission-webhooks))
- `Audit`: Validation failures are published as audit events (see below Audit
  Annotations section for details).

If, in the future, `ValidatingAdmissionPolicy` also introduces enforcement
action fields, this effective enforcement will be the set to the intersection of the
the policy enforcement actions and the binding enforcement actions.

Systems that need to aggregate validation failures may implement an [audit
webhook
backend](https://kubernetes.io/docs/tasks/debug/debug-cluster/audit/#webhook-backend). See
below "Audit Events" for details.

For singleton policies, since there is no separate binding resource, the
`validationActions` field will be set on the policy definition in the same way
that other binding fields are.

Metrics will include validation action so that cluster administrators can monitor the
validation failures of a binding before setting `validationActions` to `Deny`.

This enables the following use cases:

- A policy framework captures enforcement violations during dry run and
  aggregates them. (E.g. When in DryRun mode, OPA Gatekeeper aggregates
  violations and records them to the status of the constraint resource).
  Including validation failures in audit events makes this possible to do
  using a audit webhook backend.
- Cluster admin would like to rollout policies, sometimes in bulk, without
  knowing all the details of the policies. During rollout the cluster admin
  needs a state where the policies being rolled out cannot result in admission
  rejection. With the enforcement field on bindings, cluster admins can decide
  which initial actions to enable and then add actions until `Deny` is enabled.
  The cluster admin may monitoring metrics, warnings and audit events along the
  way.
- A policy framework needs different enforcement actions at different
  enforcement points. Since this API defines the behavior of only the admission
  enforcement point, higher level constructs can map to the actions of this
  enforcement point as needed.

Future work:

- ValidatingAdmissionPolicy resources might, in the future, add a `warnings`
  field adjacent to the `validations` and `auditAnnotations` fields to declare
  expressions only ever result in warnings. This would allow
  ValidatingAdmissionPolicy authors to declare a expression as non-enforcing
  regardless of `validationActions`.

- ValidatingAdmissionPolicy resources, might, in the future, offer
  per-expression enforcement actions (instead of a separate `warnings` field)
  and combine these enforcement actions with the
  ValidatingAdmissionPolicyBinding enforcement action to determine the effective
  enforcement. This would be designed to simplify the workflow required to add
  or update expression on an existing ValidatingAdmissionPolicy.

#### Audit Annotations

`ValidatingAdmissionPolicy` may declare [Audit
  annotations](https://github.com/kubernetes/kubernetes/blob/97bbf07d3f3f20332912ee411fdf75ce84425e28/staging/src/k8s.io/api/admission/v1/types.go#L142)
  in the policy definition. E.g.:

```yaml
apiVersion: admissionregistration.k8s.io/v1alpha1
kind: ValidatingAdmissionPolicy
...
spec:
  ...
  validations:
    - expression: <expression>
  auditAnnotations:
    - key: "my-audit-key"
      valueExpression: <expression that evaluates to a string (and is recorded) or null (and is not recorded)>
```

`auditAnnotations` are independent of `validations`. A `ValidatingAdmissionPolicy`
may contain only `validations`, only `auditAnnotations` or both.

Auudit annotations are recorded regardless of whether a
ValidatingAdmissionPolicyBinding's `validationActions` include `Audit`.

The published annotation key will be of the form `<ValidatingPolicyDefinition
name>/<auditAnnotation key>` and will be validated as a
[QualifiedName](https://github.com/kubernetes/kubernetes/blob/dfa4143086bf504c6c72d5eee8a2210b8ed41b9a/staging/src/k8s.io/apimachinery/pkg/util/validation/validation.go#L43).

The validation rule will be: `len(key) < QualifierName.maxLength - len(policy
name) - 1` to accommodate the `<ValidatingPolicyDefinition
name>/<auditAnnotation key>` audit annotation key format.

If `valueExpression` returns a string, the audit annotation is published.  If
`valueExpression` returns null, the audit annotation is omitted. No other return
types will be supported.

#### Audit Events

All audit event keys are prefixed by `<ValidatingPolicyDefinition name>/`.

At Metadata audit level or higher, when a validating admission binding fails,
and the binding's `validationActions` includes `Audit`, any validation
expression, details are included in the audit annotations for the audit event
under the key `validation_failures`. E.g.:

```yaml
# the audit event recorded
{
    "kind": "Event",
    "apiVersion": "audit.k8s.io/v1",
    "annotations": {
        "ValidatingAdmissionPolicy/mypolicy.mygroup.example.com/validation_failure": "{\"expression\": 1, \"message\": \"x must be greater than y\", \"enforcement\": \"Deny\", \"binding\": \"mybinding.mygroup.example.com\"}"
        # other annotations
        ...
    }
    # other fields
    ...
}
```

Also, at Metadata audit level or higher, any audit annotations declared by the policy definition
are included with the key provided. E.g.:

```yaml
# the audit event recorded
{
    "kind": "Event",
    "apiVersion": "audit.k8s.io/v1",
    "annotations": {
        "ValidatingAdmissionPolicy/mypolicy.mygroup.example.com/myauditkey": "my audit value"
        # other annotations
        ...
    }
    # other fields
    ...
}
```

#### Namespace scoped policy binding

For phase 1, policy bindings were only allowed to be cluster scoped. We can
support namespace scoped policy bindings as follows:

- Add a `NamespacePolicyBinding` resource.
- If the parameter resource is namespace scoped, it implicitly matches
  resources only in the namespace it is in, but may further constrain what
  resources it matches with additional match criteria.

Benefits: Allows policy of a namespace to be controlled from within the
namespace. As an example, ResourceQuota works this way.

Details to consider:

- Should a policy support both cluster scoped and namespace scoped binding? If
  so how? It would need two different parameter CRDs (since a CRD must either be
  cluster scoped or namespace scoped, not both).

#### CEL Expression Composition

##### Use Cases

###### Code re-use for complicated expressions

A CEL expression may not be computationally expensive, but could still be
intricate enough that copy-pasting could prove to be a bad decision later on
in time. With the addition of the `messageExpression` field, more copy-pasting
is expected as well. If a sufficiently complex expression ended up copy-pasted everywhere,
and then needs to be updated somehow, it will need that update in every place
it was copy-pasted. A variable, on the other hand, will only need to be updated
in one place.

###### Reusing/memoizing an expensive computation

For a CEL expression that runs in O(n^2) time or worse (or otherwise
takes a significant amount of time to execute), it would be nice to only run
it when necessary. For instance, if multiple validation expressions used the
same expensive expression, that expression could be refactored out into a
variable.

##### Match Conditions

Note that the syntax of the `matchConditions` resource is intended to
align with the [Admission Webhook Match Conditions KEP #3716](https://github.com/kubernetes/enhancements/pull/3717),
so that KEP should be controlling with regard to deviations in the schema.
This section is focused specifically on how the `matchConditions` concept can
be applied to in-process admission.

The match criteria in bindings are not expected to be able to cover all possible
ways users may want to scope their policies. For example, there is no way to
match off of kind, only resource. To provide extensibility for the match criteria
without requiring modifying every validation rule individually, a global predicate
system is needed. These predicates contain CEL statements that must be satisfied, otherwise
the policy will be ignored. In order to keep bindings language-agnostic and to support
singleton policies, the logic should live in the policy definition resource. To enable
customization per-binding, the CEL statements should have access to the parameter resource.

Here is an example of a policy definition using match conditions (under the `matchConditions` field):

```yaml
apiVersion: admissionregistration.k8s.io/v1alpha1
kind: ValidatingAdmissionPolicy
metadata:
  name: "replicalimit-policy.example.com"
Spec:
  failurePolicy: Fail
  paramKind:
    apiVersion: rules.example.com/v1
    kind: ReplicaLimit
  matchConstraints:
    resourceRules:
    - apiGroups:   ["apps"]
      apiVersions: ["v1"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["deployments"]
  matchConditions:
    - name: 'is-deployment'
      expression: 'metadata.kind == "Deployment"'
    - name: 'not-in-excluded-namespaces'
      expression: '!(metadata.namespace in params.excludedNamespaces)'
  validations:
    - expression: "object.spec.replicas <= params.maxReplicas"
      reason: Invalid
```

For demonstration purposes, we assume `match` has no support for `excludedNamespaces`.

Note that `matchConditions` and `validations` look similar, but `matchConditions` entries only have the
`expression` field: their only function is to gate whether the expressions in `validations` are evaluated.

`matchConditions` has the following behaviors:

* Only the request object and parameters are accessible (no referential lookup)
* All match conditions must be satisfied (evaluate to `true`) before `validations` are tested
* If there is an error executing a match condition, the failure policy for the (definition, binding) tuple is invoked


##### Variables

Each CEL "program" is a single expression. There is no support for variable
assignment. This can result in redundant code to traverse maps/arrays or
dereference particular fields.

We can support this in much the same way as cel-policy-template `terms`. These
can be lazily evaluated while the validation expressions are evaluated
(cel-policy-template does this).

A policy can include an additional `variables` section. This is an array
containing one or more `name` and `expression` pairs, which can be used/re-used by
the policy's validation expressions. These results are memoized on a
per-validation basis, so if multiple expressions use the same spec variables,
the expression that calculates the variable's value will only run once.

The variables can be accessed as members of `variables`, which is an object
that is exposed to CEL expressions (both validation expressions as well as
other variables).

For example:

```yaml
  variables:
    - name: metadataList
      expression: "spec.list.map(x, x.metadata)"
    - name: itemMetadataNames
      expression: "variables.metadataList.map(m, m.name)"
  validations:
    - expression: "variables.itemMetadataNames.all(name, name.startsWith('xyz-'))"
    - expression: "variables.itemMetadataNames.exists(name, name == 'required')"
```

Variable names must be valid CEL names. What constitutes a
valid CEL name can be found at CEL's [language definition](https://github.com/google/cel-spec/blob/master/doc/langdef.md#syntax) under `IDENT`.
This validity is checked at write time.

For per-policy runtime cost limit purposes, variables count towards the runtime cost limit
once per policy. The cost of each variable is computed when it is first evaluated in an
expression, mirroring how the cost limit would be calculated if the variable's
expression was embedded verbatim.  If the runtime cost limit is exceeded in the
process, then evaluation halts. No individual variable or expression will be listed as the
cause in the resulting message. Whether or not the request actually fails depends on the failure policy,
however. For subsequent uses, inclusion of the variable has zero effect on the runtime
cost limit. If the variable evaluates to an array or some other iterable, and some expression
iterates on it, that of course contributes to the cost limit, but simply including the variable does
not add the underlying expression's cost again.

Variables are also subject to the per-expression runtime cost limit. Exceeding the per-expression
runtime cost limit is always attributed to the variable, unlike the per-policy limit.

Variables can only reference other variables that
have been previously defined in the `variables` section, so circular references
are not allowed.

If an error ocurrs during variable evaluation, then the expression
that caused it to be evaluated (since variable are always
lazily-evaluated) also finishes with an error. Evaluation for that
variable is not attempted again during the same validation; if any other
expressions attempt to evaluate a variable that already failed an evaluation
attempt, they will also be considered to have failed.

#### Secondary Authz

We will support admission control use cases requiring permission checks:

- Validate that only a user with a specific permission can set a particular field.
- Validate that only a controller responsible for a finalizer can remove it from the finalizers
  field.

To depend on an authz decision, validation expressions can use the `authorizer`
variable, which performs authz checks for the admission request user (the same
use as identified by `request.userInfo`), and which will be bound at evaluation
time to an Authorizer object supporting receiver-style function overloads:

| Symbol      | Type                                                                    | Description                                                                                     |
|-------------|-------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------|
| path        | Authorizer.(path string) -> PathCheck                                   | Defines a check for an non-resource request path (e.g. /healthz)                                |
| check       | PathCheck.(httpRequestVerb string) -> Decision                          | Checks if the user is authorized for the HTTP request verb on the path                          |
| resource    | Authorizer.(kind string, group string, version string) -> ResourceCheck | Defines a check for API resources                                                               |
| subresource | ResourceCheck.(subresource string) -> ResourceCheck                     | Specifies thath the check is for a subresource                                                  |
| namespace   | ResourceCheck.(namespace string) -> ResourceCheck                       | Specifies that the check is for a namespace (if not called, the check is for the cluster scope) |
| name        | ResourceCheck.(name string) -> ResourceCheck                            | Specifies that the check is for a specific resource name                                        |
| check       | ResourceCheck.(apiVerb string) -> Decision                              | Checks if the admission request user is authorized for the API verb on the resource             |
| allowed     | Decision.() -> bool                                                     | Is the admission request user authorized?                                                       |
| denied      | Decision.() -> bool                                                     | Is the admission request user denied authorization?                                             |

xref: https://kubernetes.io/docs/reference/access-authn-authz/authorization/#review-your-request-attributes for a details on
authorization attributes.

Example expressions using `authorizer`:

- `authorizer.resource('signers', 'certificates.k8s.io', '*').name(oldObject.spec.signerName).check('approve').allowed()`
- `authorizer.path('/metrics').denied('get')`

Note that this API:

- Produces errors at compilation time when parameters are missing or improperly combined
  (e.g. subresource with non-resource path, missing path or resource).
- Is open to the addition of future knobs (e.g. impersonation).
- Will have a limit on the number of authz checks generated during expression
  evaluation, this will be enforced by setting a CEL evaluation cost to
  performing each authz check that is high enough serve as an effective limit to
  the total authz checks that can be performed. The limit will be picked
  emperically by evaluating the resource costs (CPU cost primarily) of authz
  checks. This will need to be set high enough to handle policies like "You must
  be authorized to read every secret your pod mounts".

Other considerations:

- Since authorization decisions depend on an apiserver's authorizer, and the
  ValidatingAdmissionPolicy plugin is supported on [aggregated API
  servers](#aggregated-api-servers), the evaluation result of any validation expression involving
  secondary authz inherently depends on which apiserver evalutes it.
- A validation expression could be written such that it allows malicious clients to probe
  permissions. Policy authors are responsible for careful use of client-provided inputs when
  constructing authz checks.
- Decisions can be stored either as a [variable](#variables) or as part of the implementation of the
  object bound to the exposed `authorizer` object (i.e. caching decisions for identical checks
  within a single policy evaluation) in order to prevent duplicate checks across multiple validation
  expressions.
- Authorization checks that require information from resources other than the
  resource being admitted are possible but will be limited by eventual
  consistency. Information from other resources can be accumulated by a controller
  and written to a custom resources which can then be referenced by the `paramSource`
  of a policy binding and accessed in CEL expressions via the `params` variable.


CertificateApproval use case:

```yaml
apiVersion: admissionregistration.k8s.io/v1alpha1
kind: ValidatingAdmissionPolicy
metadata:
  name: "certificate-approval-policy.example.com"
spec:
  matchConstraints:
    resourceRules:
    - apiGroups:   ["certificates.k8s.io"]
      apiVersions: ["*"]
      operations:  ["UPDATE"]
      resources:   ["certificatesigningrequests/approval"]
  validations:
  - expression: "authorizer.resource('signers', 'certificates.k8s.io', '*').name(oldObject.spec.signerName).check('approve').allowed() || authorizer.resource('signers', 'certificates.k8s.io', '*').name([oldObject.spec.signerName.split('/')[0], '*'].join('/')).check('approve').allowed()"
    reason: Forbidden
    messageExpression: "user not permitted to approve requests with signerName %s".format([oldObject.spec.signerName])"
```

Other use cases in existing admission plugins:

- PodSecurityPolicy (kube)
- CertificateSigning (kube)
- OwnerReferencesPermissionEnforcement (kube)
- network.openshift.io/ExternalIPRanger
- route.openshift.io/IngressAdmission
- scheduling.openshift.io/PodNodeConstraints
- network.openshift.io/RestrictedEndpointsAdmission
- security.openshift.io/SecurityContextConstraint
- security.openshift.io/SCCExecRestrictions

Restricted Service Account use case (from deads2k):

> Note that user.Extra in AdmissionReview has pod claims, which are valuable.

> sig-auth has previous talked about trying to find a way to restrict access
> from a daemonset pod to a customresource/foo that has Foo.spec.NodeName set to
> the Node.metadata.name of the pod bound to the particular SA token. This is
> tantalizingly close because user.Extra contains
> authentication.kubernetes.io/pod-uid to locate a pod, determine a
> Pod.spec.NodeName.

> A built-in that does that may be well received and unlock many use-cases.
> Exploring the idea may be useful. If most also require controlled read
> permission, then its probably better to create something specifically for the
> purpose.

This enhancement makes
`request.userInfo.extra['authentication.kubernetes.io/pod-uid']` available to
admission policies. Looking up the pod (or any other additional resources) is
makes this use case challenging. A controller that accumulates a `pod-uid ->
node-name` map in a custom resource by watching all pods could then make the
mapping available in a custom resource for the admission policy to consume using
`paramSource`. 

This would result in a CEL expression like:

```
object.spec.NodeName == params.nodeNamebyPodUID[request.userInfo.extra['authentication.kubernetes.io/pod-uid']]
```

But this would not scale well to clusters with large pod counts.

If we were to offer a way to lookup arbitrary other resources, or even if
we provided selective access to just some resources, this might become
easier. This can explored as future work.

#### Access to namespace metadata

We have general agreement to include this as a feature, but need to provide
a concrete design.

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

#### Resource constraints

We will leverage the design and implementation of CRD Validation Rules [Resource
Constraints](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/2876-crd-validation-expression-language#resource-constraints), which provides:

- CEL estimated cost limits
- CEL runtime cost limits
- Go context cancellation as a way of halting CEL execution if the request
  context is canceled for any reason.

 Estimated cost is, unfortunately, not something we can offer for admission with
 any kind of guarantees attached due to the already listed issues that have
 prevented use from enforcing type safety. We could instead compute estimated
 for the same cases where we provide informational type checking, in which case
 we can report any cost limit violations in the same way we report type checking
 violations. Note that for built-in types, where `max{Length,Items,Properties}`
 value validations are not available, estimated cost calculations will not be
 nearly as helpful or actionable. We do not plan to enforce any estimated
 cost calculations on ValidatingAdmissionPolicy.

 Runtime cost limits should be established and enforced. Exceeding the cost
 limit will trigger the `FailurePolicy`, so this will need to be documented, but
 unlike webhooks, runtime cost is deterministic (it is purely a function of the
 input data and the CEL expression and is independent of underlying hardware or
 system load), making it less of a concern for control plane availability than
 webhook timeouts.

 The request's Go context will be passed in to all CEL evaluations such that
 cancellation halts CEL evaluation, if, for any reason, the context is canceled.

#### Safety Features

Additional safety features we should consider:

- Configurable admission blocking write requests made internally in
  kube-apiserver during server startup (like RBAC default policy reconciliation)
  making it impossible for a server to start up healthy. (This is not specific to CEL?)
- Ability to skip specific resource types - Admission Controller Webhook
  configuration rule cannot exclude specific resources:
  https://github.com/kubernetes/kubernetes/issues/92157

#### Aggregated API servers

Main complications (provided by @liggitt):

- The API server validating/persisting the ValidatingAdmissionPolicy
  instances isn't the same one serving aggregated types, so wouldn't necessarily
  have schema info to check type safety.

- The aggregated API server is responsible for enforcing admission on its custom
  types, so the implementation that reads ValidatingAdmissionPolicy
  instances and enforces them would have to live in k8s.io/apiserver and be
  active in aggregated API servers to enforce admission on aggregated types
  effectively (same as admission webhooks today).

Plan:

- Do not offer type checking for aggregated types.
- Support ValidatingAdmissionPolicy in aggregated API servers.

#### CEL function library

To consider:

- labelSelector evaluation functions or other match evaluator functions ([original comment thread](https://github.com/kubernetes/enhancements/pull/3492#discussion_r981747317))

To implement:

- `string.format` into CEL upstream ([tracking PR](https://github.com/google/cel-go/pull/617)) (TODO @DangerOnTheRanger: add tracking cel-go issue once available)

#### Audit Annotations

To consider: Would audit support in this enhancement become redundant if
[Audit](https://kubernetes.io/docs/tasks/debug/debug-cluster/audit/) were also
extended to support CEL? If so, which should we invest in?

Admission webhooks are able to include an associative array of audit annotations
in a review response. If we intend to provide parity with webhooks we would
also want to support audit.

Rough plan:

- Each validation has a `name`. If the enforcement is `Audit` the name can be
  used as the audit annotation key.
- Can add an `audit` option next to the `deny` and `warn` enforcement options.

#### Client visibility

In order to make `DryRun` more visible to clients we will add a client
visibility option to policy bindings.

This is largely focused at making deployment/rollout more manageable.

It _might_ be generalized to control visibility of enforced violations.

#### Metrics

Goals:

- Parity with admission webhook metrics
  - Should include counter of deny, warn and audit violations
  - Label by {policy, policy binding, validation expression} identifiers 
- Counters for number of policy defintions and policy bindings in cluster
  - Label by state (active vs. error), enforcement action (deny, warn)

Granularity:

Latency metrics should be per {policy, validation expression}. The next level of
granularity would be {policy, binding, validation expression}, but that
the number of biindings can become quite large, so let's limit it to
{policy, validation expression} for now.

- xref: [Metrics Provided by OPA Gatekeeper](https://open-policy-agent.github.io/gatekeeper/website/docs/metrics/)
- xref: [Admission Webhook Metrics](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#admission-webhook-metrics)

### User Stories

In addition to "User Stores", see below "Potential Applications" for a list of
known applications and their use case requirements.

#### Use Case: Singleton Policy

User wishes to define a simple policy that required no parameters. They don't
want to create parameter CRD since what they're doing can be expressed quite
simply in a single CEL expression.

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: "validate-xyz.example.com"
spec:
  singletonPolicy: true
  match:
    resourceRules:
    - apiGroups:   ["apps"]
      apiVersions: ["v1"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["deployments"]
  defaultValidations:
  - expression: "object.spec.replicas < 100"
```

#### Use Case: Shared Parameter Resource

User wishes to define a CRD for a list of banned words that may not be used in
any of a wide range of identifiers in the cluster (resource names, container
names, ...).

- Parameter CRD is defined to hold the list of banned words.
- Multiple policies are defined for different resources. The policies all
  reference the same parameter CRD. 
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
    resourceRules:
    - apiGroups:   ["*"]
      apiVersions: ["*"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["pods"]
  validations:
  - expression: "!object.name in params.bannedWords"
```

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: "policy2.example.com"
spec:
  match:
    resourceRules:
    - apiGroups:   [""]
      apiVersions: ["v1"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["pods"]
  validations:
  - expression: "!object.spec.containers.any(c, c.name in params.bannedWords)"
  - expression: "!object.spec.initContainers.any(c, c.name in params.bannedWords)"
```

Both policies can use a trivial `PolicyBinding` to enable the same parameter
resource for both policies.

Similar Use Case: A cluster administrator wishes to use a single policy
configuration to manage a network policy that must be enforced across multiple
Kubernetes kinds that contain relevant networking fields. It is possible to
implement by having multiple `ValidatingAdmissionPolicy` resources that all
reference the same `spec.params` CRD but that each enforce the policy for a
different Kubernetes network kind.

#### Use Case: Principle of least privilege policy

A cluster administrator would like disallow the use of a list of reserved labels
by default, but allow use of the labels in specific namespaces so long as the
label values are valid. 

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
...
spec:
  paramSource:
    group: rules.example.com
    kind: ReservedLabels
    version: v1
  match:
    ...
  validations:
    expression: "['reserved1', ...].exists(r, object.metadata.labels.contains(r) && !params.allowedLabels.contains(r))"
  defaultValidations:
  - expression: "['reserved1', ...].exists(r, object.metadata.labels.contains(r)"
```

#### Use Case: Validating native type with new field (version skew case)

Policy author wants to write a policy that validates a property of all
containers in a pod, including ephemeralContainers for versions of
Kubernetes where ephemeralContainers are available.

```yaml
  validations:
    - expression: "object.spec.containers.all(c, c.name.startsWith('xyz-'))"
    - expression: "!has(object.spec.initContainers) || object.spec.initContainers.all(c, c.name.startsWith('xyz-'))"
    - expression: "!has(object.spec.ephemeralContainers) || object.spec.ephemeralContainers.all(c, c.name.startsWith('xyz-'))"
```

This does not work if type checking is strict. The ephemeralContainers field
will be reported as unrecognized.

Note: This is the sort of policy where cluster managers would ideally be able to
register the policy to validate ephemeralContainers _before_ upgrading to the
version of Kubernetes where ephemeralContainers are available for use.

Annotation to field migration example: https://github.com/open-policy-agent/gatekeeper-library/blob/master/library/pod-security-policy/seccomp/template.yaml

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
    resourceRules:
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
    excludeResourceRules:
    - apiGroups:   ["example.com"]
      apiVersions: ["v1"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["myCRD"]
    - apiGroups:   ["example.com"]
      apiVersions: ["*"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["myCRD"]
      matchPolicy: Equivalent #is the default
  validations:
  - expression: "object.v2fieldname == 'xyz'"
```

#### Use Case: Prevent admission webhooks from matching a reserved namespace

Cluster administrator wishes to prevent admission webhooks from matching
requests to specific namespaces. E.g. kube-system or some other namespace that
is critical to the cluster.

Let's assume for this example that the namespace is `kube-system`.

One approach would be to require kube-system contain a special label and the 1st
match rule of a admission webhook use a namespaceSelector:

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: "policy1.example.com"
spec:
  match:
    resourceRules:
    - apiGroups:   ["admissionregistration"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["ValidatingAdmissionWebhook", "MutatingAdmissionWebhook"]
  validations:
  - expression: >
      has(object.namespaceSelectors) && object.namespaceSelectors.size() > 0 && 
      object.namespaceSelectors[0].key = 'webhook-restricted' &&
      object.namespaceSelectors[0].namespaceSelector.operator = 'In' &&
      object.namespaceSelectors[0].namespaceSelector.values = ['true']
    message: "The 1st namespaceSelector or ValidatingAdmissionWebhook and MutatingAdmissionWebhooks must be: {key: webhook-restricted, operator: In, values: ['true']}"
    reason: Forbidden
```

This approach would pair well with a admission mutation that adds the rule to
exclude kube-system to all admission webbhooks. This would require CEL mutating
admission support.

More general types of validations like this would benefit from CEL support for
functions like `labelSelector.match()`.

#### Use Case: Fine grained control of enforcement

Policy author wishes to define a policy where the cluster administrator is able
to configure how a policy is enforced by defining a series of progressively
stricter levels.

Multiple copies of the same expression can be used, each guarded by a params
check:

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: "policy1.example.com"
spec:
  match: ...
  validations:
  - expression: "!(params.enforceLevel > 2) || <cel expression>"
    reason: Invalid
  - expression: "!(params.enforceLevel > 1) || <cel expression>"
    reason: Invalid
  - expression: "<cel expression>"
    reason: Invalid
```


#### Use Case: Migrating from validating webhook to validation policy

Steps:

1. Webhook is configured and in-use.
2. `ValidatingAdmissionPolicy` created with `FailPolicy: Ignore`
3. `ValidatingAdmissionPolicy` is monitored to ensure it behaves the same as the webhook (logs or audit annotations can be used)
4. `ValidatingAdmissionPolicy` is updated to `FailPolicy: Fail`
5. Verify the webhook never denies any requests. If the admission policy is
   equivalent, then policy will be run first and deny the request before
   webhooks are called.
6. Webhook is configured with `FailPolicy: Ignore` (optional)
7. Webhook configuration is deleted

#### Use Case: Pre-existing Deployment triggers rollout long after Pod policy is changed

- User creates a Deployment
- User uses the Deployment to roll out a ReplicaSet
- A validation policy is introduced for Pods, it is set to `Warn`
- Because deployments can be infrequent, a Deployment that would create pods
  violating the policy is not noticed
- validation policy is set to `Deny`
- problem Deployment is used to roll out a new ReplicaSet, pods fail policy
  validation

Note that even if all existing objects in the system are checked against the new
policy, the Deployment will not be noticed to violate it unless the PodTemplate
is checked. And if the policy is for Pods specifically, there is not automatic
checking of the PodTemplate.

xref: https://kyverno.io/docs/writing-policies/autogen/

#### Use Case: Rollout of a new validation expression to an existing policy

1. Policy definition A exists in cluster with policy bindings X1..Xn
1. "temporary" policy definition B is created with the new validation, it has
   the same settings as policy definition A otherwise (e.g. it uses the same
   param CR)
1. Policy bindings X1..Xn are replicated as Y1..Yn but modified to use policy
   definition B and set enforcement to warning only
1. Cluster administrators observe violations (via metrics, audit logs or logged warnings)
1. Cluster administrator determines new validation is safe
1. Policy bindings X1..Xn are set enforcement to deny
1. If anything goes wrong, revert enforcement back to warning only
1. Policy definition A is updated to include the new validation
1. Policy definition B and policy bindings Y1..Yn are deleted

#### Use Case: Canary-ing a policy

1. New policy definition is created
1. Any needed param CRs are created
1. policy bindings are created and set enforcement to warning only
1. Cluster administrators observe violations (via metrics, audit logs or logged warnings)
1. Cluster administrator determines new policy is safe
1. policy bindings are set enforcement to deny

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

This enhancement proposes using status of `ValidatingAdmissionPolicy` to
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
This avoids repeated updates for the same generation and potential fights of API
server in HA environments even without leader elected controllers.

We will use a similar approach. The complication here is that for this case we
must consider up to 3 resources:

- `ValidatingAdmissionPolicy` resource
- Parameter CRD
- The CRD for the kind-under-test, if it is a CRD (and not a built-in type)

In order to be able to know how old the three resources were when the status was
last written, we must track additional information in the status:

```yaml
apiVersion: "admissionregisteration/v1alpha1"
kind: "ValidatingAdmissionPolicy"
metadata:
  name: "myPolicy"
  generation: 2
  ...
status:
  paramKind:
    apiVersion: "example.com/v1"
    kind: "fooLimits"
    generation: 5
    resourceVersion: 10100
  matchedCustomResource:
    apiVersion: "example.com/v1"
    kind: "foo"
    generation: 100
    resourceVersion: 10200
```

Whenever an apiserver is performing a sync (of any of these three resources), it
takes the latest state is has of all three resources and checks if they represent
forward progress compared to a consistent read of the resource.

Forward progress is (1) no resource state is older than used for the last status
update (2) at least one resource state is newer.

For spec.generation, this is trival, just compare the generation from the
observed with the current state.

For referenced CRDs, it is more involved:

- Forward progress requires comparing both the apiVersion/kind and the
  generation/resourceVersion.
- If the apiVersion/kind does not match the CRD from the spec, it can safely be
  considered older without checking the generation. Goal is to converge with
  what is in the spec.
- Generation and resourceVersion comparison:
    - observed resourceVersion > existing status resourceVersion: older.
    - observedresourceVersion > existing status resourceVersion && observed generation == status generation: same.
    - observedresourceVersion > existing status resourceVersion && observed generation > status generation: newer.

If the controller has observed forward progress it updates the entire status,
including any conditions and error information:

```yaml
status:
  ...
  conditions:
    type: "Available" # TODO: pick an appropriate type for broken policies
    status: "False"
    reason: Misconfigured
    message: "Validation expressions contain errors. Param custom resource definition not found."
    ...
  validationErrors:
    - expression: "object.baz > params.min"
      errors:
        - "illegal ..."
        - "no such field ..."
  paramSourceErrors:
    - "paramSource custom resource definition not found"
```

Note that write conflicts do not require a retry since the write that caused the
conflict will result in another sync once it is observed.

Alternative Considered: Use Leader election

Pro:

- Reconciliation loop becomes noticibly simpler

Con:

- Implementation difficulty- I suspect an entire KEP could be dedicated to using
  leader election for this purpose.

### Versioning

#### Policy Definition Versioning

As a built-in type, `ValidatingAdmissionPolicy` follows Kubernetes API guidelines.

#### Parameter CRD Versioning

A parameter CFD may offer a new version using the existing CRD schema
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

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->
N/A

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
In the first alpha phase, the integration tests are expected to be added for:
- The behavior with feature gate and API turned on/off and mix match
- The happy path with everything configured and validation proceeded successfully
- Validation with different enforcement policies
- Validation with different failure policies
- Validation with different Match Criteria
- Validation violations for different reasons including type checking failures, misconfiguration, failed validation, etc and formatted messages
- Singleton policy
- Validation limit check including cost limit, max policy binding limit, max length, etc.


##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->
We will test the edge cases mostly in integration test and unit test.
We may add e2e test for spot check of the feature presence.

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
#### Alpha

- Feature implemented behind a feature flag
- Ensure proper tests are in place.

#### Beta

- benchmark and resolve optimization issues, including:
  - add tests which registers a validation policy for everything and iterates through all groups/versions/resources/subresources 
  and ensures they get intercepted and work properly with a CEL validation policy([comment](https://github.com/kubernetes/kubernetes/pull/113314#discussion_r1013596456))
  - set `paramKind` in a ValidatingAdmissionPolicy results in starting a new informer 
  that watches all instances of that object using a new unstructured informer which is inefficient([comment](https://github.com/kubernetes/kubernetes/pull/113314#discussion_r1013331221))
  - switch to a lock-free implementation to address lock having to wait for all existing admission evaluations 
  to complete and blocking all new admission evaluations until this completes.([comment1](https://github.com/kubernetes/kubernetes/pull/113314#discussion_r1013318103),[comment2](https://github.com/kubernetes/kubernetes/pull/113314#discussion_r1013305167))
  - Perform minimal possible number of conversions when evaluating multiple admission policies for a request resource. 
  If multiple admission policies require the same conversion, convert only once. 
  From @liggitt: "webhook code loops up one level, first accumulates all the validation webhooks we'll run, then converts to the versions needed by those webhooks then evaluates in parallel"
- authz check to the specific resource referenced in the policy's paramKind. ([comment](https://github.com/kubernetes/kubernetes/pull/113314#discussion_r1013135860))

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
In alpha, no changes are required to maintain previous behavior. And the feature gate
and the required API could be turned on to make use of the enhancement.

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
N/A

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
  - Feature gate name: CELValidatingAdmission
  - Components depending on the feature gate: kube-apiserver
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
No, default behavior is the same.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->
Yes, disabling the feature will result in validation expressions being ignored.

###### What happens if we reenable the feature if it was previously rolled back?
The validatingAdmissionPolicy will be enforced again.

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
Unit test and integration test will be introduced in alpha implementation.

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
The existing workload could potentially fail the validation and cause unexpected failures if the validation is misconfigured.
 
While rollout, the cluster administrator could configure the feature with `enforcement: [Warn, Audit]` or similar,
and wait until being comfortable to switch the enforcement level to `Deny`. In this way it will minimize the effect on the running workloads.

Note that if the request was in while the feature is off, when the feature is turned back on, 
the validation policy might prevent it from editing if validation rules applied.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->
On a cluster that has not yet opted into ValidatingAdmissionPolicy, non-zero counts for either of the following metrics mean 
the feature is not working as expected:
- cel_admission_validation_total
- cel_admission_validation_errors

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->
Upgrade and rollback will be tested before the feature goes to Beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->
No.

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

## Future Work

- cel-policy-template [`range`](https://github.com/google/cel-policy-templates-go/blob/master/test/testdata/map_ranges/template.yaml) or equivalent.
- Default validations?
- Short circuiting of validation (right now all are always evaluated)?
- CEL based matching support?
- kubectl support for this feature could show information about a policy and how
  it is applied? could be really useful pre-GA to help users

## Alternatives

### Type checking alternatives

Alternatives are summarized here and discussed in more detail below:

| Design Alternative                                | Summary                                                                                                                              |
| ------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Typesafe CEL expressions and scopes               | Expressions and the schema paths of expression scopes are fully typechecked, any type errors trigger the `failureMode` of the policy |
| Typesafe CEL expressions, dynamic scopes          | Expressions are typesafe, but scopes are dynamically typed, easing version skew cases                                               |
| Informational type checking                       | Expressions and scopes are typechecked, but only to report warnings, evaluation is dynamic                                           |
| No typechecking                                   | Expressions and scopes are evaluated dynamically                                                                                     |
 
Alternative: Typesafe CEL expressions and scopes

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
   resourceRules:
    - apiGroups:   ["*"]
      apiVersions: ["*"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["*"]
  validations:
    # minReadySeconds is not accessible because this resource matches multiple types
  - expression: "object.spec.minReadySeconds > 60" # ERROR! Not such field "minReadySeconds".
    # metadata is always accessible
  - expression: "object.name.startsWith('xyz')"
```

Pros:

- All CEL expressions are type checked,

Cons:

- Does not support all above cases, in particular: version skew cases,
  validation of an aggregated API server type.
- Typechecking happens quite late for some operations, lots of failure modes to
  reason through (policy created before CRD, CRD updated/deleted/recreated, CRD
  schema differs across clusters, incompatible CRD change, version skew, ...)

Alternative Considered: Typesafe CEL expressions, dynamic scopes

Idea is to use "CEL expression scoping" (see section below) in such a way
that missing schema fields due to version skew or CRD changes/inconsistencies
can be tolerated.

Scope schema paths are typechecked, but if there are any fields are missing:

- the scoped expression skips validation
- warnings are reported, but no error states are triggered

Example usage:

```yaml
# Policy definition
apiVersion: admissionregistration.k8s.io/v1alpha1
kind: ValidatingAdmissionPolicy
metadata:
  name: "validate-xyz.example.com"
spec:
  paramKind:
    group: rules.example.com
    kind: ReplicaLimit
    version: v1
  match:
    resourceRules:
    - apiGroups:   ["apps"]
      apiVersions: ["v1"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["deployments"]
  validations:
    - expression: "self.name.startsWith('xyz-')"
      scopes: ["spec.containers[*]", "initContainers[*]", "spec.ephemeralContainers[*]"]
status:
  expressionWarnings:
    - expression: "self.name.startsWith('xyz-')"
      scope: "spec.ephemeralContainers[*]"
      # For Kubernetes versions that pre-date the ephemeralContainers field:
      warnings: ["spec.ephemeralContainers[*] is not a valid schema path"]
```

Pros:

- Retains typechecking of CEL expressions while still supporting version skew
  cases and CRD changes/inconsistencies via the dynamic evalution of expression scopes.
- Possible to have a policy definition suppress expression scope warnings. E.g. 
  `suppressWarning: { type: MissingField, field: spec.ephemeralContainers, reason: 'Field is only available in Kubernetes 1.x+' }`

Cons:

- Does not handle aggregated API server case.
- Strange mix of type safety and dynamic typing. Difficult to explain, document, justify.

Alternative: Informational type checking

All CEL expressions are evaluated dynamically.

Type checking is still performed for all expressions where a GVK can be matched
to type check against, resulting in warnings, e.g.:

```yaml
...
status:
  expressionWarnings:
    - expression: "object.foo"
      warning: "no such field 'foo'"
```

Pros:

- Can handle all use cases listed.
- Does not depend on implementing "CEL expression scoping" to support listed use
  cases.
- Policy definition authors can still opt-in to take full advantage of type
  checking at development time.
- Cluster administrators can check if a policy passes type checking before
  enabled it.
- Possible to have a policy definition suppress warnings. E.g. 
  `suppressWarning: { type: MissingField, field: spec.ephemeralContainers, reason: 'Field is only available in Kubernetes 1.x+' }`

Cons:

- Type errors that would have prevented production issues can be ignored.

Alternative Considered: No typechecking

Pros:

- Possible to handle all cases dynamically.

Cons:

- No opportunity to benefit from type checking.

### Policy definition and configuration separation alternatives

#### Alternative: Duck Typed CRDs

This is the alternative shown in the initial examples of this KEP.

Policy authors write a CRD to define how each policy is configured. E.g.:

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: replicalimit.rules.example.com
  annotations:
    admission.kubernetes.io/is-policy-configuration-definition: "true"
spec:
  group: example.com
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                config:
                  type: object
                    maxReplicas:
                      type: int
  scope: Cluster
  names:
    kind: ReplicaLimit
```

The `admission.kubernetes.io/is-policy-configuration-definition` annotation
means "inject the correct .spec.match during admission and keep it up to date
with a controller". (Suggested by deads2k). This minimizes version skew if new
match criteria is added to `spec.match` and also minimizes development effort by
removing the need to manually declare the fields in CRDs.

The main challenge with this alternative is dealing with mismatches between how
CRDs declare the `spec.match` schema and what the apiserver expects, even with a
controller keeping it in sync, it can briefly become out of sync. For this our
plan is:

- When consuming configuration CRDs OR policy configuration resources used for
  policy configuration:
  - If any unrecognized fields, missing required fields, or incorrectly typed
    fields are found under `spec.match`:
    - For configuration CRDs:
      - Set the `ValidatingAdmissionPolicy` state to "misconfigured" in the
        status (via a Condition, I believe).
      - Trigger the `FailurePolicy` on all admission validations.
      - Add a detailed error in the status of the `ValidatingAdmissionPolicy`.
    - For policy configuration resources:
      - Track in the status of `ValidatingAdmissionPolicy` that some policy
        configurations are misconfigured. (also via a Condition?).
      - Add a detailed error in the status of the `ValidatingAdmissionPolicy`.
      - Trigger the `FailurePolicy` on admission for resources that match the
        policy configuration.
  - If the CRD is deleted:
    - Set state to "misconfigured
    - Trigger `FailurePolicy` on all admission validations
    - Add a detailed error in the status of the `ValidatingAdmissionPolicy`.

A partial `spec.match` schema (subset of the full schema) is okay so long as
only optional fields are omitted. But any unrecognized field in the `spec.match`
would not be allowed. 

Proposed annotation: 

> Example: `admission.kubernetes.io/is-policy-configuration-definition: "true"`
> 
> Used on: CustomResourceDefinition
> 
> What a CustomResourceDefinition has the annotation set to "true", the
> OpenAPIv3 schema of all versions of this resource is modified and then
> kept-in-sync by a controller to always contain the expected schema fields of
> admission policy configuration resources.

xref: https://kubernetes.io/docs/reference/labels-annotations-taints/

Pros:

- Concise. A single resource configures both match criteria and configuration
  params.
- If later Kubernetes OpenAPIv3 supports `$ref`, there is a migration path from
  this approach to the `$ref` approach (below)

Cons:

- A single resource for both configuration params and matching rules is
  a problem when using the same configuration with multiple polices, each
  that need different matching rules.
- API server must check for a wide range of error conditions and define how
  exactly it handles each of them.
- If the `spec.match` schema is incorrectly defined, CRD author might not
  realize it since they need to check the status of the corresponding
  `ValidatingAdmissionPolicy` for any errors.
- Changing this schema in the future could be extremely difficult. CRD schemas
  are atomic from a server-side-apply perspective (spec.versions on down).


#### Alternative: OpenAPIv3 `$ref` in CRDs

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: replicalimit.rules.example.com
spec:
  group: example.com
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                config:
                  type: object
                    maxReplicas:
                      type: int
                match:
                  $ref: "#/components/schemas/matchrules"
  scope: Cluster
  names:
    kind: ReplicaLimit
```

Pros:

- Match rule schema is owned by Kubernetes, so as it evolves, CRDs automatically
  pick up changes.

Cons:

- CRDs do not yet support OpenAPIv3 `$ref`s, so support would need to be added,
  presumably this would require a separate KEP.

#### Alternative: `/matchRules` subresource

The idea of this alternative is to require a configuration CRD to declare that
it provides match criteria by exposing a `/matchRules` subresource:

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: replicalimit.example.com
spec:
  group: example.com
  versions:
    ... # configuration schema(s) go here
  subresources:
    matchRules:
      matchRulesPath: .spec.match
```

Pros:

- CRD explicitly opt-in to providing match criteria in a structured way.
- Follows pattern used by scale subresource to provide polymorphism across CRDs

Cons:

- One of the primary purposes of subresources is accessing/modifying a portion
  of a resource independently, which is not what we're trying to achieve.
- Kubernetes development/maintenance perspective, subresources, particularly for
  CRDs, are expensive to introduce and maintain.

#### Alternative: `PolicyConfiguration` kind with config embedded

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: PolicyConfiguration
metadata:
  name: "replica-limit-prod.example.com"
spec:
  match:
    namespaceSelectors:
    - key: environment,
      operator: NotIn,
      values: ["test"]
  config:
    apiVersion: rules.example.com/v1
    kind: ReplicaLimit
    spec:
      maxReplicas: 100
```

Pros:

- Matching criteria is fully defined and validated in a builtin type

Cons:

- Embedded config needs the same treatment as custom resource, but
  reimplementing it all on an embedded resource is, at best, highly impractical
  in the apiserver as it exists today. E.g.. there is not automatic validation
  of the embedded resource.

#### Alternative: Generate CRDs

The `ValidatingAdmissionPolicy` contains a OpenAPIv3 schema defining how the
policy is configured. The schema need only contains the policy specific
configuration (it does not need to contain match rules or anything else that is
standard).

```yaml
apiVersion: admissionregistration.k8s.io/v1alpha1
kind: ValidatingAdmissionPolicy
metadata:
  name: "validate-xyz.example.com"
spec:
  config:
    group: rules.example.com
    kind: ReplicaLimit
    version: v1
    openAPIV3Schema:
      type: object
      properties:
        spec:
          type: object
          properties:
            maxReplicas:
              type: int
```

This allows the apiserver to combine the configuration schema with the match
schema and then generate a CRD. The apiserver then can run a control loop to
always keep the CRD match stanzas in sync with what is expected.

This could alternatively be kept separate from the policy definition by having
something like:

```yaml
apiVersion: admissionregistration.k8s.io/v1alpha1
kind: ValidatingAdmissionConfigurationDefinition
metadata:
  name: "validate-configuration-xyz.example.com"
spec:
  configSchema:
    group: rules.example.com
    kind: ReplicaLimit
    version: v1
    openAPIV3Schema:
      type: object
      properties:
        spec:
          type: object
          properties:
            maxReplicas:
              type: int
```

Pros:

- Policy author doesn't need to define the CRD. It can instead be generated for
  them.

Cons:

- Most of the cons of "Duck Typed CRDs" alternative apply, since ultimately a
  CRD is created and behaves the same.
- This implies that the policy owns the configuration. But we have use cases
  where multiple policies share the same configuration. For this model it seems
  misleading and potentially problematic to have both policies attempting to
  define the same configuration.

### CEL variables alternatives

#### Alternative: Scopes

Imagine that a policy validation uses a CEL expression find an invalid value in
a list somewhere nested in a resource.  E.g. `spec.initContainers.<listitem>.name`.

How do we:

- Including the invalid value in a message?
  - Using a second expression to build a message that then must again find the
    validation error in the list duplicates a lot of code, and is inefficient.
- Include the field path of the problem field in the message?
  - It can be hard coded in the message string for basic cases, but for more
    complex cases (where the are map keys or array indices involved) it becomes
    messy and complicated to reconstruct.
- Traverse across fields safely? CEL offers the `all()` macro, so traversals
  like `spec.initContainers.all(c, c.name)` are possible, but can be subtly
  incorrect because of optional fields (`initContainers` in this case), which
  require `has()` checks, e.g.: `!has(spec.initContainers) || spec.initContainers.all(c, c.name)`.

CRD Validation Rules solved these problems by allowing validation rules to
attached to any schema in the OpenAPIv3. The validation rules are scoped to
whatever location in the OpenAPIv3 they are attached.

We propose using a simple schema path format. The purpose of the path is to
uniquely identify a schema from the root of a CRDs OpenAPIv3 schema.

Example:

```
spec.initContainers         # Schema of initContainers array
spec.initContainers[*]      # Schema of the items of the initContainers array
spec.initContainers[*].name # Schema of the name of initContainers
```

For example, to validate all containers:

```yaml
  validations:
    - scope: "spec.containers[*]"
      expression: "scope.name.startsWith('xyz-')"
      message: "scope.name does not start with 'xyz'"
```

To make it possible to access the path information in the scope, we can offer a
way to bind varables to the map and list indices in the path, e.g.:

```
spec.x[xKey].y[yIndex].field
```

```yaml
  validations:
    - scope: "x[xKey].y[yIndex].field"
      expression: "scope.startsWith('xyz-')"
      messageExpression: "'%s, %d: some problem'.format([scopePath.xKey, scopePath.yIndex])"
```

Prior art:

- cel-policy-template's offer a
  [`range`](https://github.com/google/cel-policy-templates-go/blob/master/test/testdata/map_ranges/template.yaml)
  feature that allows a CEL expression to be scoped to each entry of a map or
  item of an array. Multiple ranges can be combined to traverse a complex
  object.
- Kyverno [foreach
  declarations](https://kyverno.io/docs/writing-policies/validate/#foreach), use
  [JMESPath](https://jmespath.org/) to query for the elements that are then
  validated.

Note: We considered extending to a list of scopes, e.g.:

```yaml
  validations:
    - scopes: ["spec.containers[*]", "initContainers[*]", "spec.ephemeralContainers[*]"]
      expression: "scope.name.startsWith('xyz-')"
      message: "scope.name does not start with 'xyz'"
```

But feedback was this is signficantly more difficult to understand.

### Message formatting alternatives

Alternative: CEL args

```yaml
- expression: "..."
  message: "{1} is less than {2}"
  messageArgs: ["spec.value", "spec.max"]
```

Cons:

- How all types are converted to string becomes the responsibility of this API.
  Hard to please everyone and may end up needing to reimplementing `fmt.Sprintf`.
  In which case this is probably best handled from within CEL.

Alternative: Inline CEL expressions

Single `message` field but it supports templating, e.g.:

```
"{{object.int1}} is less than {{object.int2}}"
```

Cons:

- Must defining escaping rules in string for including `{{` or `}}` as a literal
- CEL expressions must be properly escaped

Alternative: Inline JSON path

```
"{{.object.int1}} is less than {{.object.int2}}"
```

Cons:

- (Same as above "Inline CEL expressions")
- Author must switch between using CEL and JSON Path in adjacent fields
- JSON Path is less expressive than CEL (both a pro and a con)

Alternative: CEL expressions, separate args from format string

```yaml
- expression: "..."
  message: "{1} is less than {2}"
  messageArgs: ["", "object.int2"]
```

Note "%s is less than %s" is also viable, but CEL can always preformat and emit
a string for cases where developer needs more control.

Cons:

- Slightly more verbose format (but avoid all the escaping problems)


## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
