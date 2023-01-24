# KEP-3775: Mutating Admission with CEL

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Examples](#examples)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
  - [Type Definitions and Checking](#type-definitions-and-checking)
  - [Bindings and Parameter Objects](#bindings-and-parameter-objects)
  - [Return Values for Mutating Expressions](#return-values-for-mutating-expressions)
  - [Unresolved Items and Future Considerations](#unresolved-items-and-future-considerations)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
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

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP introduces a new API, MutatingAdmissionPolicy, that enables CEL-based mutation of Kubernetes objects,
leveraging capabilities and best practices learned from ValidatingAdmissionPolicy (introduced in v1.26).

## Motivation

CEL for Admission Control has proven that CEL expressions can be used for validating objects in the kube-apiserver admission chain.
This KEP proposes to expand usage of CEL to mutate objects, reducing the infrastructure footprint that would often be required when
using mutating admission webhooks. Many of the same benefits and use-cases for ValidatingAdmissionPolicy also apply for MutatingAdmissionPolicy.


### Goals

* Leverage CEL to perform mutations against Kubernetes objects
* Reduce use-cases requiring mutating admission webhooks

### Non-Goals

* Wholly replacing mutating admission webhooks

## Proposal

MutatingAdmissionPolicy will reuse and leverage many existing fields from ValidatingAdmissionPolicy,
specifically for matching resources and binding parameter objects. Defer to KEP “CEL for Admission Control”
for more details.

For alpha, mutations will be supported via CEL expressions that resolve to JSONPatch type definitions,
which are then applied against matched resources. JSON Patch serves as a great starting point for mutations for the following reasons:

1. JSON Patch is familiar – it is already supported by kubectl and used under the hood in many places (including mutating admission webhooks)
2. Reduces required fields in the initial API since operations and fields to mutate are specified directly in the expression.
3. JSON Patch is robust and can support a wide range of use-cases out of the box.

Future versions of this API will very likely evolve away from CEL expressions that simply resolve to JSONPatch definitions.
Direct use of JSON Patch in CEL expressions will serve as a means to build a minimal viable API for users to experiment and provide early feedback.

### Examples

Below are some examples to illustrate the high level design before diving into the implementation details.

The first policy contains two expressions that mutate Pods. The first expression sets pod.spec.dnsPolicy
field to None and the second defaults pod.spec.dnsConfig to a config pointing to a custom nameserver:

```yaml
apiVersion: admissionregistration.k8s.io/v1alpha1
kind: MutatingAdmissionPolicy
metadata:
 name: "demo-policy.example.com"
spec:
 matchConstraints:
   resourceRules:
   - apiGroups:   [""]
     apiVersions: ["v1"]
     operations:  ["CREATE", "UPDATE"]
     resources:   ["pods"]
 mutations:
   - expression: 'JSONPatch[{ op: "replace", path: "/spec/dnsPolicy", value: "None" }]'
     condition: "object.spec.dnsPolicy != "None"
   - expression: 'JSONPatch[{ op: "replace", path: "/spec/dnsConfig", value: { "nameservers": ["1.2.3.4"], "searches": ["ns1.svc.cluster-domain.example", "my.dns.search.suffix"] } }]'
```

Here is another example of a policy that defaults all Deployments to have 5 replicas:
```yaml
apiVersion: admissionregistration.k8s.io/v1alpha1
kind: MutatingAdmissionPolicy
metadata:
 name: "demo-policy.example.com"
spec:
 matchConstraints:
   resourceRules:
   - apiGroups:   ["apps"]
     apiVersions: ["v1"]
     operations:  ["CREATE", "UPDATE"]
     resources:   ["deployments"]
 mutations:
   - expression: 'JSONPatch[{ op: "replace", path: "/spec/replicas", value: 5 }]'
```

Lastly, here is an example policy that sets the imagePullPolicy for all containers to “Always”:
```yaml
apiVersion: admissionregistration.k8s.io/v1alpha1
kind: MutatingAdmissionPolicy
metadata:
 name: "demo-policy.example.com"
spec:
 matchConstraints:
   resourceRules:
   - apiGroups:   ["apps"]
     apiVersions: ["v1"]
     operations:  ["CREATE", "UPDATE"]
     resources:   ["deployments"]
 mutations:
   - expression: '[1…size(object.spec.containers)].map(i, JSONPatch[{op: “replace”, path: /spec/containers/i/imagePullPolicy, value: “Always”}])'
```

NOTE: the expression `[1...size(object.spec.containers)]` is not supported by CEL yet but there are on-going discussions to add
a new macro and type to support this use-case.

### User Stories (Optional)

#### Story 1

As a cluster admin, I would like to have a mutating admission policy that adds a HTTP_PROXY
and HTTPS_PROXY environment variable to every container in my cluster. Deploying a mutating
admission webhook seems cumbersome since the proxy URL is static.

#### Story 2

As a cluster admin, I would like to have a mutating admission policy that ensures all
`ValidatingWebhookConfiguration` and `MutatingWebhookConfiguration` objects exclude
`leases` and all objects in the `kube-system` namespace. This is currently not possible
today since `MutatingWebhookConfiguration` cannot mutate `ValidatingWebhookConfiguration` or
`MutatingWebhookConfiguration` objects.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

* Users can find new ways to break clusters using CEL expressions. While the risks
are similar to risks with mutating webhooks, this would expose more avenues for those risks.

## Design Details

### API

```go
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MutatingAdmissionPolicy describes the definition of an admission validation policy that mutates an object.
type MutatingAdmissionPolicy struct {
  metav1.TypeMeta
  // Standard object metadata; More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata.
  // +optional
  metav1.ObjectMeta
  // Specification of the desired behavior of the MutatingAdmissionPolicy.
  Spec MutatingAdmissionPolicySpec
}

// MutatingAdmissionPolicySpec is the specification of the desired behavior of the MutatingAdmissionPolicy.
type MutatingAdmissionPolicySpec struct {
  // ParamKind specifies the kind of resources used to parameterize this policy.
  // ...
  // +optional
  ParamKind *ParamKind

  // MatchConstraints specifies what resources this policy is designed to validate.
  // ...
  // Required.
  MatchConstraints *MatchResources

  // Mutations contain CEL expressions which are used to apply mutations.
  // A minimum of one mutation is required for a policy definition.
  // Required.
  Mutations []Mutation

  // FailurePolicy defines how to handle failures for the admission policy.
  // ...
  // Allowed values are Ignore or Fail. Defaults to Fail.
  // +optional
  FailurePolicy *FailurePolicyType
}

// Mutation specifies the CEL expression which is used to apply the mutation.
type Mutation struct {
  // Expression represents a CEL expression used for patching objects
  // <...>
  // +optional
  Expression string
  // Condition is a CEL expression used to evalulate whether this mutation should apply.
  // <...>
  Condition string
  // Message represents the message displayed when mutations fail.
  // ...
  // +optional
  Message string
  // Reason represents a machine-readable description of why this mutation failed.
  // ...
  // +optional
  Reason *metav1.StatusReason
  // reinvocationPolicy indicates whether this expression should be called multiple times as part of a single admission evaluation.
  // <...>
  // +optional
  ReinvocationPolicy *ReinvocationPolicyType
}
```

### Type Definitions and Checking

Mutating CEL expressions will support a `JSONPatch` type that can be referenced directly in the expression (see examples above).
CEL supports static type-checking as long as a CEL program does not use dynamically expanded messages (see [Gradual Type Checking](https://github.com/google/cel-spec/blob/d6c4afd0655cf8a7f85a1f117d50b0507dc965ab/doc/langdef.md#gradual-type-checking)).
Since the JSON Patch objects are well-defined to a few fields, we can leverage Gradual Type Checking to support static type checking of expressions.

### Bindings and Parameter Objects

MutatingAdmissionPolicy will follow the same API patterns used to support bindings and parameter objects in ValidatingAdmissionPolicy.
Any future changes to ValidatingAdmissionPolicy in that regard should also be applied for MutatingAdmissionPolicy.

### Return Values for Mutating Expressions

Mutating CEL expressions will support multiple return values. Expressions should resolve to `JSONPatch` types if a mutation is desired,
but it can also return `null` to indicate no-op or `true` / `false` to indiciate acceptance or rejection of the admission request.
Type checking will be done in validation to ensure mutating expressions must return one of these types.

### Unresolved Items and Future Considerations

* Support for additional types: `JSONMergePatch`, `StrategicMergePatch` and `ApplyConfiguration`
* CEL does not support indexing of lists yet, which may block progress on some key use-cases
* Preventing recursion in mutations

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

TBD once package structure is defined.

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

For alpha, integration tests should cover the following use-cases:
* defaulting fields
* mutating fields for all elements in an array
* idempotency
* expressions that are no-op (resolve to null)
* expressions that resolve to bools

##### e2e tests

TBD

### Graduation Criteria

#### Alpha

* Mutating CEL expressions support JSONPatch definition
* Mutating CEL expressions are type-checked during validation of MutatingAdmissionPolicy
* Mutating CEL expressions can support the following use-cases:
    * expressions can default fields of various types (string, int, JSON objects, etc)
    * expressions can mutate the same field for all elements in an array (e.g. imagePullPolicy=Always in pod.spec.containers[*])
    * expressions can be idempotent
    * expressions can return bool values to indicate admission accept/deny
    * expressions can return null to indicate no-op
* MutatingAdmissionPolicy supports reinvocationPolicy, similar to MutatingWebhookConfiguration
* Unit and integration tests

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