# KEP-3488: CEL for Admission Control

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

Currently the way custom policies are enforced are via admission
webhooks. Admission webhooks are extremely flexible, but have a few drawbacks as
compared to in-process policy enforcement:

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
AdmissionRequest to include it.

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

Also, making it possible (and convenient) to declare zero trust policies is
important to compliance. i.e., it should be possible for policies governing 
resources like namespace and roles to default to the most restrictive state
when first created.

## Proposal

### Phase 1

Introduce a new `ValidatingAdmissionPolicy` kind to the
admissionregistration.k8s.io group. (suggestions welcome on exact name to use
for kind)

At a high level, the API will support:

- Request matching (similar to webhook Rules, NamespaceSelector,
  ObjectSelector)
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
severity, levels, failure policies, type safety, ...) that will be discussed
in detail further on in this proposal.

#### API

Before getting into all the individual fields and capabilities, let's look at the
general "shape" of the API.

We will separate policy definition from policy configuration. This has a couple advantages:

- Access to, and delegation of, policy configuration is more managable with this separation of concerns.
- Without the separation, all configuration would need to be encoded into a single resource could become
  very large and it prone to running into the resource size limits.

`ValidatingAdmissionPolicy` will define a admission control policy. It
will contain the CEL expressions for the vallidation rule and declare how a policy is
configured.

For example:

```yaml
# Policy definition
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: "validate-xyz.example.com"
spec:
  match:
  - rules:
      resources: ["deployments"]
  config:
    group: rules.example.com
    kind: ReplicaLimit
    version: v1
  validations:
    - expression: "self.spec.xyz == config.x"
      # ...other rule related fields here...
```

To configure a policy, custom resources of the Kind referenced by
the `spec.config` field of the `ValidatingAdmissionPolicy` are used:

For example:

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
      values: ["prod", "staging"]
  maxReplicas: 3
```

##### Matching

Policies are only enforced at admission for to the resources that are matched.

Both policy definitions (`ValidatingAdmissionPolicy`) and policy configurations contain match criteria.

Policy definition matching (i.e. the match rules in `ValidatingAdmissionPolicy`):

- Establish what resources the policy applies to across the entire cluster. Even when no policy
  configurations match the resource, if the policy definition matches, the admission policy is enforced.
  This enables zero trust policies-- policies that default to their most restrictive state. 
- Govern what data a CEL expression has access to. If the rules match a single Group, Version and Kind (GVK) then
  the CEL expression is allowed to access all fields of that kind, otherwise the CEL expression is only
  allowed to access metadata.  Accessing the scale subresource of multiple resources still results in a match
  against a single GVK, so if both deployment/scale and replicaset/scale are matched, the CEL expression has access
  to all the scale resource fields.

Policy configuration matching (e.g. the match rules in `ReplicaLimit` custom resource in the above example):

- Determines which resources are validated using the configuration of the particular policy.
- Is constrainted matching criteria of the policy definition. For example, if the policy definition matches
  v1.Pods, then policy configuration may only further constrain the match with additional matching
  rules (e.g. a namespaceSelector), but it will never match anything other than v1.Pods.
- Differs depending on the scope of the policy configuration resource:
  - If the configuration resource is namespace scoped, it matches resources only in the namespace it is in.
  - If the configuration resource is cluster scoped, it match all resources (namespace and cluster scoped) 
    unless it contains are additonal match rules that constrain their matching to a specific namespace.
- May overlap. If two policy configurations match a resource, it is validated against both configurations.

TODO: What happens when the policy definition has no match rules. It should be possible to have an "abstract"
policy definition that does not match anything but allows policy configurations to apply it anywhere. Is
it reasonable to have this behavior if the policy definition has not match rules?

TODO: How to match ONLY cluster scoped resources? If matching a single kind, then using a cluster scoped
policy configuration is sufficient. But I don't think it is possible all cluster scoped resources with
what we've defined here.

TODO: Example: Zero trust policy

TODO: Example: Policy that is abstract/defaults to off

TODO: Example: Single resource policy

##### Configuration

The policy definition may provide a default configuration.

Policy configurations may override the default configuration.

If the policy definition declares a configuration kind, all possible resource matches are
required to have a configuration. This includes resources that match the policy
definitions but not any policy configurations.

TODO: Explain how policy configuration is handled for combinations of policy definition
{has config, does not have config} and policy configuration {has config, does not have config}.

TODO: what if all the fields of a policy configuration are
optional? does that impact the above requirements, or do we still require a `config: {}` to state
explicitly that an empty configuration is desired.

If a policy definition does not declare a configuration kind, then it
may not have any configuration (making the policy definition resource
and entirely self contained policy).

TODO: Example: Configuration override

TODO: Example: Abstract policy that must be configured by each policy configuration (no default). (same as above "abstract/defaults to off" example?)

#### API Capabilities

In addition to the basic API "shape" proposed above, there are quite
a few other capatiblities that will be included in the API:

We expect that the design and implementation of some of these will need to
be deferred to future phases. We have ordered these by priority of
implementation, where we are prioritizing capatiblities that impact the system
as a whole over those that are more isolated and could be added independantly
with minimal impact on other aspects of the API.

##### Failure Policy

- For in-process validation there is no remote request, so errors should be
  deterministic.
- Cases we should consider:
  - What if a CEL rule evaluates to an error?
  - What if a CEL rule is for a CRD kind and the CRD changes such that the CEL
    rule fails type checking? (or the CRD does not exist?)

Because failure policy is most often selected based on the need to guarantee
enforcement, we propose defaulting failure policy to "fail" and allowing it to
be configured on a per-rule basis:

```yaml
  validations:
    - rule: "self.spec.xyz == configuration.x"
      failurePolicy: (Fail|Ignore)
```

##### Type safety

- How can CEL rules know what types they are validating? This gets complicated if the
  ValidatingAdmissionConfiguration matches multiple GVKs.
- What happens when a ValidatingAdmissionConfiguration is created in a cluster
  where either CRDs or native types are incompatible with CEL rules and the CEL
  rules fail to compile?

To keep failure policy easy to reason about, and to continue to use CEL in a type-safe way
we propose:

- If a ValidatingAdmissionConfiguration matches:
  - ...a single GVK, then the CEL expression is allowed access to the full object in a typesafe way.
  - ...multiple GVKs, then the CEL expression is allowed access to the metadata only.
- If there are any type checking errors (or if the CRD for the matched GVK does not exist) 
  all runtime evaluations of the rule fail immediately, without the CEL expression being
  evaluated.
- All type check errors are reported in the status of the ValidatingAdmissionConfiguration.

The main disadvantage of this approach is that CEL rules that would have evaluated successfully
dynamtically (where type checking is skipped), will instead fail. By favoring type safety here
were are trading off avaiability for consistent and perdictable behavior.

Example: Typesafe access to object 

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionConfiguration
metadata:
  name: "validate-xyz.example.com"
spec:
  match:
  - rules:
      apiGroups: ["apps"]
      apiVersions: ["v1"]
      resources: ["deployments"]
validations:
    # replicas is accessible because this resource matches only v1 deployments
  - rule: "self.spec.replicas < 100"
```

Example: Typesafe access only to metadata

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionConfiguration
metadata:
  name: "validate-xyz.example.com"
spec:
  match:
  - rules:
      apiGroups: ["apps"]
      apiVersions: ["v1"]
      resources: ["deployments", "statefulSet"]
validations:
    # minReadySeconds is not accessible because this resource matches multiple types
  - rule: "self.spec.minReadySeconds > 60" # ERROR! Not such field "minReadySeconds".
    # metadata is always accessible
  - rule: "self.name.startsWith('xyz')"
```

##### Rule scope

- Dereferencing to deeply nested fields in CEL can be burdonsome. A field
  containing a field path would allow a CEL rule to be scope to some node in
  the schema. E.g. `spec.containers[*].image`
- More ambitiously, the idea of named scopes (e.g. "ALL_CONTAINERS") would allow a single
  rule to be applied to all container definitions in a resource (e.g. both
  "containers" and "initContainers" in a pod)

##### Transition rules

- Should we use "self" and "oldSelf"? How do transition rules work for delete?

##### Messages

We need:

- Formatted messages (need support for string formatting. Allow a CEL expression
  to be used to format the whole string? Or allow CEL expressions to evaluate
  template values?). Note that CRD Validation Rules do NOT support formatted
  messages (but if we add support for formatted message for admission control,
  we should consider adding the feature to CRD validation as well).
- severity (e.g. "Warning")
- status types (e.g. Forbidden can be returned from a webhook using
  HTTP 403) (can a match rule override this? or can it be selected via a CEL
  expression?)
- Similar to how each match has configuration, can each match also select
  the severity of messages? It make it possible to configure a policy as
  warning only when first enabling it. Some policy frameworks support this.

##### Access to namespace labels

- Most heavily needed fields not directly available in the resource being
  validated. Note that namespaceSelectors already allow matches to examine
  namespace levels.

##### RBAC checks

kube-apiserver authorizer checks (aka Secondary-authz checks) have been proposed as a way of doing things like:

- Validate that only a user with a specific permission can set an enum to the "HOLD" value.
- Validate that only a controller responsible for a finalizer can remove it from the finalizers field.
 
 Concerns: "Is joe authorized to do this"? That only works for the objects joe creates, but not objects that get created on joe's behalf by a controller. Ditto for updates. I heard someone cite PSP as an example for why it's needed, but IMO that was an anti-pattern of PSP, and one that we explicitly decided to omit from PSA

##### Decision Rules

- Are overlapping matches combined using an OR or AND? Can this be configured?
- Can a validation rule depend on another validation rule (only evaluate if the
  other rule is true?)
- How to make it so that an "not authorized" error prevents all other validation
  messages from being returned (to prevent exfiltration)?

##### Composition utilities

- Ability to define a sub-expression and then use it in validation rules (xref
  cel-policy-template "terms")

### Phase 2

TBD. This enhancement is large enough that we anticipating the alpha implementation
will happen over multiple releases.

### User Stories

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
