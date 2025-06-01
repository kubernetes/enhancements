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

# KEP-5359: Pod-Level Swap Control

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

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
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
    - [Example](#example)
    - [Validation Scenarios](#validation-scenarios)
    - [Kubelet Changes](#kubelet-changes)
    - [Backward Compatibility](#backward-compatibility)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
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

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
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

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

This KEP proposes introducing a pod-level API field to control swap memory usage for individual pods.
This enhancement aims to complement the existing node-level swap support by providing a per-workload swap configuration.

The initial focus is to allow pods to explicitly disable swap, even if swap is enabled at the node level
(via the `LimitedSwap` setting introduced in [KEP-2400](https://github.com/kubernetes/enhancements/issues/2400)).
In the future, the pod-level swap control API field can be extended to allow more granular control over swap usage,
or to provide hints to kubelet on how to configure swap for individual pods, depending on the configured "swapBehavior".

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Kubernetes has introduced node-level swap support (KEP-2400), currently Beta3 in 1.33.
KEP-2400 allows nodes to utilize swap memory by configuring kubelet to use the `LimitedSwap` swap behavior.
The `LimitedSwap` mode is currently purely automatic and implicit.
Swap limits are automatically assigned to
pods, [based on various factors](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2400-node-swap#steps-to-calculate-swap-limit),
while users cannot specifically configure swap for individual pods.
For example, `LimitedSwap` restricts Guaranteed and high-priority pods from swap for performance reasons,
and users cannot specifically exclude other critical pods from potential swap usage.

While node-level swap can improve node stability and resource utilization in certain scenarios, it presents a challenge
to some latency-sensitive applications.
For these applications, performance degradation with any swap activity may be undesirable.
Currently, if a node has swap enabled, application owners do not have a standard Kubernetes API mechanism to express
their workload's intolerance to swap.

This KEP addresses this gap by providing a pod-level swap control API.
In the first phase, we'll focus on the ability to disable swap for all its containers irrespective of underlying node
swap behavior.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

* Introduce a new `swapPolicy` field in PodSpec to allow users to configure swap for an individual pod.
* Initially, provide a way to disable swap under the `LimitedSwap` swap behavior.
* Maintain backward compatibility: existing pods that run on swap-enabled nodes should behave as they do by default.
* Provide a mechanism to alleviate concerns regarding "all-or-nothing" nature of node-level swap enablement, potentially
  unblocking KEP-2400’s path to GA.
* Open the door to future enhancements, both at the kubelet-level swap behaviors and the new pod-level swap control API
  field.
  For example, a more sophisticated declarative node-level swap behavior that relies on hints provided by pod owners,
  or a swap behavior that allows users to explicitly specify swap limits for individual pods.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

* To introduce fine-grained swap controls at the pod or container level (e.g., setting a specific swap limit other than
  zero) in this initial KEP.
* To change the fundamental behavior of LimitedSwap for Guaranteed, Burstable, or BestEffort pods when the pod-level
  swap setting is not specified.
* To define how swap contributes to pod eviction or schedulable resource limits beyond disabling its usage.
* To enhance pod scheduling to allow users to specify swap-capable node preferences. This will be achievable with
  features such
  as [node-capabilities](https://docs.google.com/document/d/1vSDlAA3o0riVq0EcmGBOYUJUVF4tN2Ib7VJg3o1LvBw/edit?tab=t.0).
* To support cgroupv1 for this feature, as Kubernetes swap support (KEP 2400) is focused on cgroup v2.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

As the owner of a latency-sensitive application,
I want to ensure my pod’s memory pages never swap out to help avoid performance degradation in a multi-tenant environment.

Since swap is enabled on the node,
the kernel may still swap other pods' pages in and out to reclaim memory, potentially impacting performance.

#### Story 2

As an application owner managing swap tolerant batch jobs, I do not set pod.spec.swapPolicy, allowing it to use any swap
as available if the node-policy permits.

#### Story 3

As a cluster administrator, I enable LimitedSwap on nodes for cost optimization. The pod.spec.swapPolicy allows specific
workloads to opt-out.

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

1. Ability to disable swap at workload level may hide other concerns of swap such as noisy-neighbor issues, where a
   heavy swap reliant workload could cause cpu or I/O contention with another workload that doesn’t use swap.
    1. `PodSwapAwareness` is not a replacement for node level swap isolation for workloads.
       `PodSwapAwareness` feature should protect workloads that naturally fit in the same nodes as some swap requiring
       workloads,
       but cannot afford the performance cost.
       The alternative for these workloads is to also use swap or be OOM killed due to memory pressure (when no swap is
       present).
    2. When node-capability based filtering is available, users can utilize it to select swap-disabled nodes during
       scheduling for clear-isolation, managing explicit node-swap preferences.
2. User confusion about interactions between API and node-level configuration.
    1. A potential mitigation would be to improve pod-level visibility to clearly indicate whether a workload is
       actively utilizing swap memory.
    2. Clear documentation on pod-level, node-level and QoS dependencies on swap will be added.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### API Changes

We propose adding a new field swapPolicy to pod.spec.

```go
// PodSpec is a description of a pod
type PodSpec struct {

// .. existing fields..

// SwapPolicy defines the desired swap memory policy for this pod. This field
// is immutable after the pod has been created.
//
// If unspecified, the default value is "NoPreference", which means the pod's swap behavior is determined by the node's swap 
// configuration (KEP-2400).
// If mode is set to "Disabled", swap will be disabled for this pod, irrespective of the node's swap configuration.
// 
// +featuregate="PodSwapAwareness"
// +optional
SwapPolicy *PodSwapPolicy `json:"swapPolicy,omitempty"`
}

// PodSwapPolicy defines the swap memory policy for a pod.
type PodSwapPolicy struct {
// Mode defines the desired swap behavior mode for the pod.
// This field is immutable after the pod has been created.
// Two modes are supported:
// - "Disabled": Swap will be disabled for this pod.
// - "NoPreference": The pod adheres to the node’s default swap behavior. This is the default if swapPolicy is unset.
// 
// +optional
Mode SwapPolicyMode `json:"mode,omitempty"`
}

// SwapPolicyMode defines the possible values for mode in pod.swapPolicy.
type SwapPolicyMode string

const (
// SwapPolicyModeDisabled explicitly disables swap for the pod. 
SwapPolicyModeDisabled SwapPolicyMode = "Disabled"
// SwapPolicyModeNoPreference states the pod should follow node-level swap configuration.
SwapPolicyModeNoPreference SwapPolicyMode = "NoPreference"
)
```

Notes:

1. The swapPolicy field will be optional.
2. For Alpha, the only accepted modes will be `NoPreference` or `Disabled`.
3. If the swapPolicy field is not set or empty `swapPolicy.mode`, it will default to `NoPreference` swap mode,
   meaning the node's configured swap behavior (e.g. `LimitedSwap`) will determine how swap is utilized by the pod and
   ensures backward compatibility.
4. The field `pod.spec.swapPolicy` will be immutable after pod creation.

#### Example

A pod that wants to disable swap would be configured like this:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: no-swap-pod
spec:
  swapPolicy:
    mode: Disabled
  containers:
    - name: my-app
  image: test-image  
```

A pod that explicitly wants to follow node's default behavior would look like this:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: swap-tolerant-pod
spec:
  swapPolicy:
    mode: NoPreference
  containers:
    - name: my-app
  image: test-image
```

#### Validation Scenarios

The following pod types are currently excluded from swap usage with `LimitedSwap` (KEP-2400):

* Guaranteed and BestEffort QoS class pods.
* System-priority pods (system-node-critical, system-cluster-critical).

When any of the above pod has 'Disabled' swap-behavior:

1. Kubelet disables the pod's swap cgroup control explicitly.
2. This action is consistent with the node swap policy for the excluded pod categories.
3. Setting the pod swap limit to `0` maintains existing behavior and will not be rejected at the API level.

These node-swap policy exclusions will not have explicit validations because they align with the `Disabled`
swap-behavior.

#### Kubelet Changes

Kubelet will recognize and act upon the `pod.spec.swapPolicy` field as follows:

1. If the kubelet configuration `memorySwap.swapBehavior` is `NoSwap`, kubelet will disable system-level swap and
   `pod.spec.swapPolicy` has no effect (existing behavior).
2. When kubelet is in `LimitedSwap` mode and the `PodSwapAwareness` feature gate is disabled, the `pod.spec.swapPolicy`
   field is ignored. In this configuration, burstable pod containers will get a swap proportion based on the memory
   requested for these containers (existing behavior).
3. Pod Swap Behavior Enforcement:

   With PodSwapAwareness feature-gate enabled,
    1. If the node has swap configured (e.g. `LimitedSwap`) and `pod.spec.swapPolicy.mode` is set to Disabled, Kubelet
       will
       configure the pod’s cgroupv2 swap control memory.swap.max=0 to prevent swap usage.
       Their containers will not get swap allocation.
    2. If `pod.spec.swapPolicy` is unset or `pod.spec.swapPolicy.mode` set to `NoPreference`, the pod adheres to
       existing
       node-level swap policy and QoS rules.

#### Backward Compatibility

1. The new field is optional. API server will enforce the immutability of this field after it is created.
2. Pods created without swapPolicy will continue to function as they do today.
   Existing pods utilizing swap on enabled nodes will continue to benefit from swap without any change.
3. Older kubelets will ignore the field. Newer kubelets will act on `pod.spec.swapPolicy` when the feature gate is
   enabled.

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

- `kubernetes/kubernetes/tree/master/pkg/kubelet`: `<date>` - `<test coverage>`
  Parsing `PodSwapAwareness`, feature-gate, `memory.swap.max` manipulation.
- `kubernetes/kubernetes/tree/master/pkg/apis/core/v1/validation`: `<date>` - `<test coverage>`
  Validation of new API field for immutability.

##### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

* Ensure kubelet handles combinations of node swap settings: cgroupv2 x swapPolicy x QoS. Verify correct swap cgroup
  controls are set.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

* Existing Swap tests must pass fine even after introducing the `PodSwapAwareness` api.
* swapPolicy: "Disabled" on `LimitedSwap` nodes: verify no swap.
* swapPolicy: "NoPreference" on `LimitedSwap` nodes: verify existing QoS rules.
* Feature-gate disabled continues existing behavior.
* Pods with `swapPolicy` configured on a cgroupv1 node: kubelet will report an error event but will ignore the setting.

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
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved 

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- N examples of real-world usage
- N installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

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

* Feature implemented behind feature flag PodSwapAwareness (default false).
* Existing node e2e tests around swap must pass.
* New e2e tests to ensure kubelet enforces swapPolicy.mode: "Disabled".
* Documentation: swapPolicy field, node swap interaction.

#### Beta

* Feature gate PodswapPolicy default to true.
* Consider other "swapPolicy" values for scheduling preferences based on user feedback. Future expansions for improved
  scheduling awareness could include options like "Required", "Preferred" and "Avoid" to express varying affinity and
  anti-affinity rules for workloads.

#### GA

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

Upgrade: Feature gate is off by default in Alpha.
No changes until enabled on nodes and swapPolicy field used on pods.

Downgrade: swapPolicy is ignored by older kubelets.
Pods set with swapPolicy.mode: "Disabled", will revert to node-level behavior.

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

Standard n-2 skew.

Newer API / Older Kubelet: Kubelet ignores field
Older API / Newer Kubelet: new field is not settable.

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

- [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: PodSwapAwareness.
    - Components depending on the feature gate: kubelet.
- [X] Other
    - Describe the mechanism: Kubelet needs to be restarted when enabling/disabling this feature.
    - Will enabling / disabling the feature require downtime of the control
      plane? No.
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? Yes.

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

This feature introduces a new user facing behavior for swap.
However, the default behavior of pods is not changed with this feature.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Requires kubelet restart. See Rollout, Upgrade and Rollout planning.

###### What happens if we reenable the feature if it was previously rolled back?

When the feature-flag is disabled, the new swapPolicy field will be ignored by kubelet.
Re-enabling the feature-flag will enable this behavior, and any pods that start later on this node will adhere to the
swapPolicy configured at the pod spec.

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

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
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

The following alternative was considered and ruled out, because people seem to favor predictability in this area.
In any case, I'll keep this here for reference.

One potential concern with the `Disabled` mode is that it forces the API to allow pod owners to disable swap, regardless
of the `swapBehavior` setting configured by the admin at the kubelet level.

This is problematic because swap is fundamentally a system-level resource that impacts node stability and cannot be
isolated per pod, as is possible with CPU and memory.
In practice, disabling swap could negatively affect other pods, since the kernel would be unable to swap out memory
pages
from the pod with `swapPolicy.mode` set to `Disabled`, even if those pages are never actively used.

Looking ahead, we might want to introduce more sophisticated kubelet-level swap behaviors that can make dynamic
decisions based on pod hints and real-time memory usage. Locking into the `Disabled` mode now could limit future
flexibility.

An alternative approach could be renaming `Disabled` to `PreferNoSwap`, signaling a preference rather than an absolute
restriction. Under the `LimitedSwap` behavior, this would effectively opt the pod out of swap, while leaving room for
future behaviors that allow selective swap usage for pods that prefer to avoid it.

However, if the goal is to consistently allow pod owners to opt out of swap across all swap behaviors, keeping the
`Disabled` mode may be the appropriate choice.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
