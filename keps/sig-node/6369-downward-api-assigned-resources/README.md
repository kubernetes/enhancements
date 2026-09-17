# KEP-6369: Pod Assigned Resource Exposure via Downward API

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
  - [Use Cases](#use-cases)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Implementation](#implementation)
    - [<code>NodeDeclaredFeatures</code> Integration](#nodedeclaredfeatures-integration)
    - [Resource Field Extensions](#resource-field-extensions)
    - [Downward API Volume Exposure](#downward-api-volume-exposure)
    - [Downward API Environment Variable Exposure](#downward-api-environment-variable-exposure)
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
  - [1. Read the cgroup files from inside the container](#1-read-the-cgroup-files-from-inside-the-container)
  - [2. Query the kubelet pod resources endpoint](#2-query-the-kubelet-pod-resources-endpoint)
  - [3. Expose the assignments in the pod status](#3-expose-the-assignments-in-the-pod-status)
  - [4. Volume files only, without environment variables](#4-volume-files-only-without-environment-variables)
  - [5. Include the amount of memory in <code>assigned.memset</code>](#5-include-the-amount-of-memory-in-assignedmemset)
  - [6. Expose the assignments through DRA](#6-expose-the-assignments-through-dra)
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
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
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
-->

This proposal extends the Downward API to expose the exclusive CPUs and the memory NUMA nodes assigned to a container, both as volume files and as environment variables. This extension is controlled by the new `DownwardAPIAssignedResources` feature gate.

This KEP was split from [KEP-6122](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/6122-configurable-scaling-delay) because the two features (configurable scale-down delay and downward API exposure of assigned resources) are functionally independent. This separation improves document readability and reduces complexity, making each feature easier to understand and review.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Latency-sensitive applications often require exclusive CPUs to achieve predictable performance and resource isolation. These applications commonly use CPU affinity to minimize performance degradation caused by CPU migration.

However, when scaling down guaranteed QoS pods, containers need to know in advance which CPUs will be removed from their cpuset. Together with KEP-6122 (which introduces configurable scale-down delay), this feature allows latency-sensitive applications to obtain the assigned cpuset in advance via the downward API. This enables workloads to take preparatory actions — such as migrating workloads away from affected CPUs — and avoid performance degradation caused by CPU migration, core sharing, and sudden CPU loss during the removal of active CPUs.

Memory assignments are exposed from the start here. They were part of the original KEP-6122 draft, were dropped from its Alpha scope for timing reasons, and remained an Alpha2 criterion there; @kad's approval was explicitly conditioned on treating CPU and memory consistently. Covering both in this KEP's Alpha settles that, and the compact list representation described below is the one @ffromani asked for.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

* Expose CPU and Memory assignments to containers via the downward API with `DownwardAPIAssignedResources` feature gate enabled:
   + `assigned.cpuset`: The desired exclusive cpuset (Linux cpuset format, e.g. `0-3,7,12-15`). Empty string when no exclusive CPUs are assigned.
   + `assigned.memset`: The assigned memory NUMA nodes, in the same list format (e.g. `0-1`). Empty string when no memory is assigned.
* Integrate with `NodeDeclaredFeatures` so that a node declares whether it can serve these values, and the scheduler keeps pods requesting them off nodes that cannot.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

* Allow containers to specify which CPUs to remove during scale-down.
* Let a pod influence which CPUs or memory NUMA nodes it is assigned. These values report a decision, they do not take part in making it.
* Expose anything beyond what is assigned to the container itself, such as the assignments of other pods or the node's full topology.
* Guarantee a window in which a workload can react to a change. That is the subject of [KEP-6122](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/6122-configurable-scaling-delay).

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

This proposal extends the Downward API with two new values of the existing `ResourceFieldRef.Resource` field: `assigned.cpuset`, carrying the CPU Manager's cpuset for the container, and `assigned.memset`, carrying the Memory Manager's set of memory NUMA nodes. Exposure of both is gated by the `DownwardAPIAssignedResources` feature gate.

**Note:** The feature supports both downward API volume files and environment variables. Volume file values are updated while the container runs, including during a resize, while an environment variable is evaluated when the container is created and keeps that value until the container is recreated.

### Use Cases

1. A DPDK-style workload is given four exclusive CPUs and pins its worker threads to exactly those cores. Today it has to discover them from inside the container by reading its cgroup files. With this feature it reads `assigned.cpuset` instead — from an environment variable if it only needs the value at startup, or from a volume file if it wants to follow later changes.
2. The same workload allocates its buffers on the memory NUMA nodes it was actually assigned, read from `assigned.memset`, rather than inferring them from whichever CPUs it happens to be running on.
3. Together with [KEP-6122](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/6122-configurable-scaling-delay), the workload also learns the cpuset it is about to be given: during a scale-down the volume file is updated with the new set before that set is applied to the container, so the workload can move work off the CPUs being removed. Without KEP-6122 the value still changes, but there is no guaranteed window in which to react.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

**Architectural Coupling Concern:** The Downward API was designed to expose fields of the pod spec and status — declarative state the user wrote — whereas `assigned.cpuset` and `assigned.memset` are node-local runtime state computed by the CPU Manager and the Memory Manager. Exposing them this way couples the Downward API to those implementations, which could constrain how CPU management evolves, in particular alongside a DRA CPU driver. This was raised by @dchen1107 during the review of KEP-6122 and accepted there as a non-blocker for Alpha.

**Mitigation:** The coupling is bounded by the fact that the kubelet's static CPU policy and the DRA CPU driver cannot run on the same node, so the two mechanisms do not compete for the same workloads. Rather than being specific to the CPU Manager, the exposed file path is intended as a shared contract that a DRA driver can publish to as well; this was the direction proposed by @pravk03, the dra-driver-cpu maintainer, and it is tracked in [dra-driver-cpu#181](https://github.com/kubernetes-sigs/dra-driver-cpu/issues/181). If the coupling has to be undone later, the pod status alternative in [Alternatives](#alternatives) covers the same use cases without naming a resource manager.

**Security Considerations:** No new information is disclosed. A container can already read both values from its own cgroup files — `cpuset.cpus` for the CPUs and `cpuset.mems` for the memory NUMA nodes — so this KEP changes how the values are delivered, not who can see them. The set of memory NUMA nodes does reveal part of the node's topology, but only the part already assigned to that container and already readable by it. No new data recipients are created: the volume file and the environment variable are visible to exactly the processes that can read the container's cgroup files today.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Implementation

The feature builds on the existing Downward API framework. The `DownwardAPIAssignedResources` feature gate has a different job in each component: in kube-apiserver it decides whether a pod referencing `assigned.cpuset` or `assigned.memset` is accepted, and in the kubelet it decides whether those values are produced and whether the node declares the feature.

#### `NodeDeclaredFeatures` Integration

This feature integrates with the Node Declared Features framework ([KEP-5328](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/5328-node-declared-features), GA since v1.37), so that a pod asking for these values is only placed on a node able to serve it.

When the `DownwardAPIAssignedResources` feature gate is enabled, the kubelet declares `DownwardAPIAssignedResources` in `node.status.declaredFeatures` during bootstrap. The declaration depends on the feature gate alone, not on which resource manager policies are configured: it states that the node understands these values, and an empty value is the correct answer for a container with no exclusive CPUs or no assigned memory.

The scheduler infers that a pod referencing `assigned.cpuset` or `assigned.memset` requires the feature and only places it on nodes that declare it. This keeps such a pod off a kubelet that predates the feature, where the downward API setup for the container would fail. A pod that reaches a node without going through the scheduler — a static pod, or one placed by a custom scheduler — is not covered by this filtering; see [Version Skew Strategy](#version-skew-strategy) for what the kubelet does then.

Once the feature graduates to GA and the feature gate is removed, every kubelet serves these values and the declared feature is no longer needed. Declared features are temporary by design in KEP-5328 and are removed as part of the post-GA cleanup.

#### Resource Field Extensions

Two new values, `assigned.cpuset` and `assigned.memset`, are added to the existing `ResourceFieldRef.Resource` field:

ResourceFieldRef.Resource
* resource: limits.cpu
   + A container's CPU limit
* resource: requests.cpu
   + A container's CPU request
* resource: limits.memory
   + A container's memory limit
* resource: requests.memory
   + A container's memory request
* resource: limits.hugepages-*
   + A container's hugepages limit
* resource: requests.hugepages-*
   + A container's hugepages request
* resource: limits.ephemeral-storage
   + A container's ephemeral-storage limit
* resource: requests.ephemeral-storage
   + A container's ephemeral-storage request
* **resource: assigned.cpuset** *(NEW)*
   + **A container's desired set of exclusive CPUs.**
* **resource: assigned.memset** *(NEW)*
   + **A container's desired set of assigned memory NUMA nodes.**

Both values use the Linux list format — `0-3,7,12-15` for CPUs, `0-1` for NUMA nodes. They mirror the `cpuset.cpus` and `cpuset.mems` pair of the cgroup cpuset controller, where `cpuset.mems` is the set of memory NUMA nodes and is the established counterpart of `cpuset.cpus`. The amount of memory assigned is deliberately not part of `assigned.memset`, since the Downward API already exposes it through `limits.memory`.

#### Downward API Volume Exposure

The Volume manager gets the CPU state from CPU manager, and writes it to the Downward API volume file `assigned.cpuset`, which exposes the CPUSet to the container:
  - If the container has exclusive CPUs assigned, the value exposes the exclusive cpuset:
    - during a scale-down, the newly allocated cpuset, before it is applied to the container,
    - otherwise, the cpuset the container currently holds.
  - Otherwise, the value is empty ("").

The Volume manager gets the Memory state from Memory manager, and writes it to the Downward API volume file `assigned.memset`, which exposes the set of memory NUMA nodes to the container:
  - If the container has memory assigned, the value exposes those NUMA nodes (e.g. `0-1`).
  - Otherwise, the value is empty ("").

#### Downward API Environment Variable Exposure

The environment variable for CPU exposure gets the CPU state from the CPU manager when the container is created:
  - If the container has exclusive CPUs assigned, the value exposes the exclusive cpuset.
  - Otherwise, the value is empty ("").

The environment variable for Memory exposure gets the Memory state from the Memory manager when the container is created:
  - If the container has memory assigned, the value exposes those NUMA nodes (e.g. `0-1`).
  - Otherwise, the value is empty ("").

**Note** An environment variable is evaluated when the container is created and is not updated afterwards, so it does not follow a resize; it is re-evaluated only when the container is recreated. Volume file values, in contrast, are updated whenever the assignment changes. A workload that needs the value to follow a resize must read the volume file.

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

We plan on adding or extending tests in the following files.

API validation:
- `pkg/apis/core/validation/validation_test.go`: the two new values accepted when the gate is enabled and rejected when it is disabled, for both a downward API volume item and a container environment variable.
- `pkg/api/pod/util_test.go`: wiring the gate into the pod validation options, and the ratcheting rule that keeps a value already in use permitted once the gate is off.

Producing the values:
- `pkg/volume/downwardapi/downwardapi_test.go`: writing `assigned.cpuset` and `assigned.memset` into the volume, an empty value when the container has no assignment, and an empty value when the gate is disabled.
- `pkg/kubelet/kubelet_pods_test.go`: the environment variable path, including that the value is taken at container creation and not refreshed afterwards.

Node Declared Features:
- `staging/src/k8s.io/component-helpers/nodedeclaredfeatures/features`: a new package for this feature and its registration, following the per-feature packages already present there.

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

- **Validation ratcheting**: with the gate disabled in kube-apiserver, creating a pod that references these values is rejected, while updating a pod that already references them — including through a resize — is accepted.
- **Scheduler filtering**, in `test/integration/scheduler/filters/`:
  - a pod referencing these values is scheduled to a node that declares `DownwardAPIAssignedResources`;
  - nodes that do not declare it are filtered out, representing older kubelets or nodes with the gate disabled;
  - with no declaring node available, the pod stays `Pending` with a `FailedScheduling` event naming the missing feature.

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

- **Volume exposure**: a container referencing `assigned.cpuset` and `assigned.memset` in a downward API volume sees the CPUs and the memory NUMA nodes it was assigned.
- **Volume follows a resize**: after a scale-down the volume file shows the newly allocated cpuset.
- **Environment variable exposure**: a container referencing the same values as environment variables sees them at startup, and they do not change after a resize.
- **No assignment**: a container with no exclusive CPUs and no assigned memory sees empty values.
- **Gate disabled on the node**: the container keeps running and sees empty values rather than failing.
- **Gate rollback and rollout**: with the gate disabled, creating a pod referencing these values is rejected while a pod already using them keeps running and can still be updated; after re-enabling, the volume files are filled in again.


### Graduation Criteria

#### Alpha

* Two new values, `assigned.cpuset` and `assigned.memset`, are accepted by kube-apiserver behind the `DownwardAPIAssignedResources` feature gate and rejected when it is disabled.
* The kubelet produces both values for downward API volume files and for container environment variables.
* With the gate disabled the kubelet produces an empty value rather than failing the container.
* The kubelet declares `DownwardAPIAssignedResources` in `node.status.declaredFeatures`, and the scheduler filters on it.
* Unit, integration and e2e tests as described in the test plan.

#### Beta

* No unresolved critical bugs, and bugs reported by users have been addressed.
* Alternatives to the coupling between the Downward API and the resource managers have been evaluated, together with the DRA work tracked in [dra-driver-cpu#181](https://github.com/kubernetes-sigs/dra-driver-cpu/issues/181).

#### GA

* Allow time for feedback (6+ months).
* Make sure all risks have been addressed.

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

**Upgrade.** No change is required of an existing cluster. Nothing exposes these values unless a pod asks for them. To use the feature, enable `DownwardAPIAssignedResources` on kube-apiserver and on the kubelets, and reference `assigned.cpuset` or `assigned.memset` from a downward API volume item or a container environment variable.

**Changing what a pod exposes.** Downward API volume items and container environment variables are part of the pod spec and cannot be changed on a running pod, so adding or removing these values means recreating the pod.

**No state is stored.** The values are computed from the CPU Manager's and the Memory Manager's current state every time a volume is written, so there is nothing to migrate, and nothing a newer kubelet could leave behind that an older one would have to read.

The two downgrade paths differ in whether the target version knows these values at all.

**The target version knows the values, with the feature gate disabled.** The values are preserved on existing pods and rejected on new ones, so a pod already using them keeps running and can still be updated. Its volume files become empty, while its environment variables keep whatever they were given when the container was created.

**The target version does not know the values.** kube-apiserver rejects pods that reference them: these are new values of an existing field, so validation checks them against a closed set of allowed names and fails with an unsupported container resource error. Operators should remove these references from pod specs before such a downgrade.

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

This feature involves coordination between kube-apiserver (field validation), the kubelet (producing the values) and the scheduler (node filtering via Node Declared Features).

**New apiserver, older kubelet.** The apiserver accepts `assigned.cpuset` and `assigned.memset`. An older kubelet does not know these resource names at all, so it cannot produce a value for them and the downward API setup for such a container fails. Node Declared Features prevents this from being reached: such a kubelet does not declare `DownwardAPIAssignedResources`, so the scheduler does not place these pods on it.

**Old apiserver, newer kubelet.** Not a supported configuration, since the [version skew policy](https://kubernetes.io/releases/version-skew-policy/#kubelet) requires that the kubelet not be newer than kube-apiserver. Were it to occur anyway, the apiserver would reject the pod: these are new values of an existing field, and an apiserver that does not know them fails validation with an unsupported container resource error.

**Apiserver ON, kubelet OFF.** The pod is admitted, but the node does not declare the feature and the scheduler avoids it. If such a pod runs there anyway — a gate flip under a running pod, or a pod placed without the scheduler — the kubelet exposes an empty value instead of failing the pod. Unlike the older kubelet above, this one has the code and can degrade gracefully.

**Apiserver OFF, kubelet ON.** New pods using these values are rejected by the apiserver, so they never reach the kubelet. A pod that already uses them keeps them and is still served by the kubelet, since validation permits a value already in use. Static pods bypass the apiserver, so a static pod using these values is served regardless of the gate there.

**Both ON.** Full behavior: the apiserver validates the values, the kubelet declares the feature and produces the files and environment variables, and the scheduler places these pods only on nodes that declare it.

**Both OFF.** Feature disabled, existing behavior.

In clusters with mixed node versions the Node Declared Features framework ([KEP-5328](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/5328-node-declared-features)) handles the skew on its own: only nodes declaring the feature receive pods that use these values. The operator therefore does not have to upgrade every kubelet before enabling the gate, and a pod that no node can serve stays unschedulable instead of failing on a node that cannot produce the values.

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

This feature requires enabling the following feature gate

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `DownwardAPIAssignedResources`
    - Components depending on the feature gate: kube-apiserver, kubelet

The gate has a different job in each component. In kube-apiserver it decides whether a pod referencing `assigned.cpuset` or `assigned.memset` is accepted. In the kubelet it decides whether those values are produced and whether the node declares the feature. It has to be enabled on both for the feature to work; [Version Skew Strategy](#version-skew-strategy) describes what happens when only one of them has it.

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No. Enabling the gate changes nothing on its own: the values are produced only for pods that reference `assigned.cpuset` or `assigned.memset`, and a pod that references neither behaves exactly as before.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes, and no workload is disrupted by it.

**Disabling on kube-apiserver:** new pods referencing `assigned.cpuset` or `assigned.memset` are rejected, while pods that already reference them keep the reference and keep running, since validation permits a value already in use.

**Disabling on kubelet:** the kubelet writes an empty value into the volume files and leaves the environment variables as they were. Containers keep running; they only stop being told what they were assigned. The node also stops declaring the feature, so the scheduler will not place further pods needing it there.

###### What happens if we reenable the feature if it was previously rolled back?

**On kube-apiserver:** new pods referencing these values are accepted again.

**On the kubelet:** the volume files of pods already referencing them are filled in again at the next write. Environment variables are not, because they are evaluated when the container is created; a container has to be recreated to pick up a correct value there.

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

Yes. Unit tests exercise the feature gate switch itself: that the validation option follows the gate, and that a value already in use in the old spec stays permitted once the gate is off, so that disabling it does not break updates of running pods. On the kubelet side, unit tests cover that a container is given an empty value when the gate is off, rather than the downward API setup failing.

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

**Highly available control plane.** During a rollout the gate may be enabled on some apiservers and not others. Creating a pod that references these values then succeeds or fails depending on which apiserver serves the request. The failure is an explicit validation error rather than silent acceptance, and it disappears once the rollout completes. Updates of pods that already reference them are unaffected, because the validation option is derived from the old spec and does not depend on the gate state of the apiserver handling the request.

**Rolling the gate out across nodes.** A kubelet starts declaring `DownwardAPIAssignedResources` once the gate is enabled on it. Until enough nodes declare it, a pod referencing these values stays `Pending` with a scheduling event, rather than running somewhere that cannot serve it. That is a visible and recoverable state.

**Already running workloads are not affected.** A rollback does not kill pods. With the gate disabled the kubelet writes an empty value into the volume file and leaves the environment variables as they were, so a container keeps running and only loses the information. Nothing about the CPU or memory assignment itself changes — this feature only reports it.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

No dedicated metrics are added in Alpha. The signals to watch are pods that stay `Pending` because no node declares the feature, and pod creation failures caused by an apiserver that still has the gate disabled. Both are visible without access to the nodes.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

Local testing plan.

**Feature gate enable → disable → enable**

1. Cluster with the gate enabled on kube-apiserver and the kubelet. Create a pod with a downward API volume and an environment variable for `assigned.cpuset` and `assigned.memset`.
   - Verify the volume files and the environment variables carry the assigned CPUs and memory NUMA nodes.
   - Resize the pod down and verify the volume files follow the new assignment while the environment variables keep their original values.
2. Disable the gate on kube-apiserver and the kubelet and restart both.
   - Verify the pod keeps running and can still be updated, since the values are already in use.
   - Verify the volume files are now empty.
   - Verify that creating a new pod using these values is rejected.
3. Re-enable the gate on both and restart.
   - Verify the volume files carry the assigned values again.
   - Verify a newly created pod using these values is admitted.


###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

N/A

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

No metric is added in Alpha. The references live in the pod spec, under `spec.volumes[].downwardAPI.items[].resourceFieldRef.resource` and `spec.containers[].env[].valueFrom.resourceFieldRef.resource`, so an operator determines use by listing pods whose spec names `assigned.cpuset` or `assigned.memset`. Because the references are nested in arrays, this needs a JSON query rather than a field selector. The nodes able to serve them are those declaring `DownwardAPIAssignedResources` in `node.status.declaredFeatures`.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

For a single pod this is visible from inside the container, without access to node logs or metrics. Read the value and compare it with the container's own cgroup files: a working setup gives the same set in `assigned.cpuset` as in `cpuset.cpus`, and in `assigned.memset` as in `cpuset.mems`. An empty value for a container that does hold exclusive CPUs or assigned memory means the node is not serving the feature — see [Troubleshooting](#troubleshooting).

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

These are the guarantees this KEP can make on its own. The window in which a workload can act on an upcoming change belongs to [KEP-6122](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/6122-configurable-scaling-delay), not here.

- A non-empty value matches the assignment the resource manager currently holds for that container.
- When an assignment changes, the volume file is updated within one kubelet sync period.
- While a change has been computed but not yet applied to the container, the volume file shows the new assignment rather than the old one.
- An environment variable reflects the assignment as of container creation and is not updated afterwards. This is a property of the mechanism, not a failure.
- This feature never delays a resize; it only reports.


###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- Time from a resource manager computing a new assignment to the corresponding volume file being written.
- Number of volume files whose contents disagree with the assignment the resource manager holds, which should be zero.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

Yes. A histogram of the delay between an assignment changing and the volume file being written, and a counter of failed writes, would let an operator check the SLOs above without inspecting individual pods. Neither is implemented in Alpha.

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

No new in-cluster or external services. The feature relies on the following, all in-tree:

- CPU Manager `static` policy (`--cpu-manager-policy=static`)
  - Usage description: `assigned.cpuset` has a value only for containers holding exclusive CPUs, which exist only under this policy.
    - Impact of its outage on the feature: under any other policy no container has exclusive CPUs, so `assigned.cpuset` is always empty.
    - Impact of its degraded performance or high-error rates on the feature: N/A, a kubelet configuration.
- Memory Manager `Static` policy (`--memory-manager-policy=Static`)
  - Usage description: `assigned.memset` has a value only for containers the Memory Manager has assigned memory to, which happens only under this policy. The value is read through the Memory Manager's existing `GetMemory` interface; the Memory Manager itself is unchanged.
    - Impact of its outage on the feature: with `None` no container has assigned memory, so `assigned.memset` is always empty.
    - Impact of its degraded performance or high-error rates on the feature: N/A, a kubelet configuration.
- Node Declared Features ([KEP-5328](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/5328-node-declared-features), GA since v1.37)
  - Usage description: the kubelet declares `DownwardAPIAssignedResources` whenever the feature gate is enabled, and the scheduler uses it to keep pods requesting these values off nodes that would not understand them.
    - Impact of its outage on the feature: a pod may be placed on a node that cannot serve the request — it then receives empty values, or, on a kubelet predating the feature, the downward API setup for the container fails. The same applies to pods placed without the scheduler, such as static pods.
    - Impact of its degraded performance or high-error rates on the feature: a stale node status could misroute pods for as long as the declared features are out of date, with the same bounded consequence.

Neither resource manager policy gates the declaration: a node declares the feature whenever the feature gate is enabled. Nothing is lost by that, because a node not running the static policies has no exclusive assignments to report in the first place, so an empty value is the accurate answer rather than a degraded one. The declaration states that the node understands these fields, not that it currently has assignments to report.

No new container runtime capability is required: the kubelet writes the values into the Downward API volume and the container environment itself, without involving CRI.

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

No. The kubelet reports its declared features as part of the node status it already sends, and the values themselves are written locally into the pod's volume, without involving the API server.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No new API types, and no new field. Two new values, `assigned.cpuset` and `assigned.memset`, become valid for the existing `ResourceFieldRef.Resource` field; they are listed in [Resource Field Extensions](#resource-field-extensions).

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

Only for pods that use the feature, and no differently from any other downward API reference: such a pod carries one volume item or one environment variable entry naming the new value, which costs the same as naming `limits.cpu` does today. No new objects are created.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No. Producing a value is an in-memory lookup in the CPU Manager or the Memory Manager, and it is written into a volume the pod already mounts, so nothing is added to the pod startup path beyond what any downward API item costs.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No. One small file per reference, in a volume the pod already mounts.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

No. The number of files is bounded by the number of containers referencing these values, exactly as for any other downward API item.

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

The values come from the kubelet's own CPU Manager and Memory Manager state, not from the control plane, so the downward API volume files of a running pod keep being updated while the apiserver is unreachable. Creating a pod that uses these values requires the apiserver, as any pod creation does.

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

- A container is given an empty value although the node is expected to support the feature
  - Detection: the volume file or the environment variable is empty while the container does have exclusive CPUs or assigned memory.
  - Mitigations: enable the gate on that node, or move the pod to a node that declares the feature.
  - Diagnostics: kubelet logs, and `node.status.declaredFeatures` on the node in question.
  - Testing: the unit and e2e tests for the gate being disabled.
- A pod using these values stays `Pending`
  - Detection: the pod has no node assigned and the scheduler reports that no node satisfies its required features.
  - Mitigations: enable the gate on at least one node, or remove the fields from the pod.
  - Diagnostics: scheduling events from `kubectl describe pod`, and the declared features of the candidate nodes.
  - Testing: the scheduler filtering integration test.
- An environment variable is stale after a resize
  - Detection: the environment variable disagrees with the corresponding volume file.
  - Mitigations: none. This is by design; workloads that need the value to follow a resize must read the volume file.
  - Diagnostics: compare the environment variable with the volume file for the same resource.
  - Testing: the e2e case verifying that volume files follow a resize while environment variables keep their original values.

###### What steps should be taken if SLOs are not being met to determine the problem?

The SLOs concern the accuracy and the freshness of the exposed values.

If a value disagrees with the container's actual assignment, first check whether it is an environment variable. Those are set when the container is created and are expected to be stale after a resize, which is not a violation. For a volume file, compare it with the container's `cpuset.cpus` and `cpuset.mems`, and look in the kubelet log for failures writing the volume.

If a value is empty where an assignment does exist, the node is not serving the feature: check that the gate is enabled there, that the node declares `DownwardAPIAssignedResources`, and that the relevant resource manager policy is not `none`.

If a volume file lags an assignment change by much more than a kubelet sync period, check the kubelet's sync configuration first, then whether the pod is being synced at all.

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

- 2026-09-15: KEP created by splitting the Downward API exposure out of [KEP-6122](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/6122-configurable-scaling-delay) ([#6370](https://github.com/kubernetes/enhancements/pull/6370))
- 2026-09-17: Memory exposure brought into the Alpha scope, with `assigned.memset` defined as the set of assigned memory NUMA nodes

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

### 1. Read the cgroup files from inside the container

* **Description**: The container already sees its own `cpuset.cpus` and `cpuset.mems` through cgroupfs, so a workload could parse them instead of being told.
* **Why Rejected**: Those files show the set that is currently applied, never the one that is about to be applied, so they cannot serve the advance-notice use case that motivates this KEP together with KEP-6122. They are also an implementation detail rather than a contract: the paths differ between cgroup v1 and v2, depend on whether a cgroup namespace is in use, and are not guaranteed across runtimes. Every workload would carry its own fragile parser for something Kubernetes can state plainly.

### 2. Query the kubelet pod resources endpoint

* **Description**: The kubelet already reports concrete CPU and memory assignments through the gRPC service at `/var/lib/kubelet/pod-resources/kubelet.sock` ([KEP-2043](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2043-pod-resource-concrete-assigments)).
* **Why Rejected**: That endpoint is designed for node monitoring agents, not for workloads. Using it from an ordinary container means mounting a host socket into the pod, which grants visibility into every pod on the node — a privilege escalation that is hard to justify for a workload that only needs to know about itself. The Downward API exists precisely to give a pod information about itself without such access.

### 3. Expose the assignments in the pod status

* **Description**: Report the assigned CPUs and memory NUMA nodes in the pod status, for example as `pod.status.resourceAssignments`, and let the workload watch its own pod. Suggested by @dchen1107 and @ffromani during the review of KEP-6122 as a way to decouple the exposure from the CPU Manager implementation.
* **Why Rejected for Alpha**: This requires API credentials and RBAC inside the workload, plus a watch per pod on the apiserver, to deliver information the node already has locally. It also makes the value's freshness depend on the control plane being reachable, whereas a Downward API volume is written from local state. It remains the strongest candidate should the coupling discussed in [Risks and Mitigations](#risks-and-mitigations) need to be undone, since it would cover CPU, memory and future resource types through one field.

### 4. Volume files only, without environment variables

* **Description**: Expose the values only as Downward API volume files, since an environment variable cannot be updated after the container has started.
* **Why Rejected**: A workload that needs the value only at startup, to pin its threads once, is better served by an environment variable than by mounting a volume. Offering the values in one form but not the other would also make `assigned.cpuset` and `assigned.memset` behave unlike every other `ResourceFieldRef` value, which is available in both. The staleness is real and is recorded as a known failure mode in [Troubleshooting](#troubleshooting).

### 5. Include the amount of memory in `assigned.memset`

* **Description**: Report both the memory NUMA nodes and the assigned memory size, for example `memory:2097152000,NUMA:[0-1]`.
* **Why Rejected**: The size is already available through `limits.memory` in the same Downward API, so it would be redundant. It would also force a composite encoding into a field whose every other value is a plain list or quantity, and make `assigned.memset` structurally unlike `assigned.cpuset` for no gain.

### 6. Expose the assignments through DRA

* **Description**: Have the Dynamic Resource Allocation framework provide the abstraction instead, for example by publishing the assignment through CDI. Suggested by @SergeyKanzhelev during the review of KEP-6122.
* **Why Deferred**: The DRA CPU driver and the kubelet's static CPU policy cannot run on the same node today, so DRA cannot serve the workloads this KEP targets. Exposing DRA allocations the same way is tracked in [dra-driver-cpu#181](https://github.com/kubernetes-sigs/dra-driver-cpu/issues/181), and the intent is for the driver to use the same file path, which makes this a future extension of the contract rather than a competing design.

### 7. Inject the assignments as files, the way DRA device attributes are

* **Description**: Rather than letting a pod request these values through `ResourceFieldRef`, have the component that knows them inject them. [KEP-5304](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/5304-dra-attributes-downward-api) does this for DRA device attributes: the driver's metadata is bind-mounted into the container through a CDI spec and appears at a well-known path. Applied here, the CPU Manager and the Memory Manager would inject their assignments the same way, with nothing in the pod spec asking for them. Suggested by @liggitt in [kubernetes/kubernetes#136015](https://github.com/kubernetes/kubernetes/pull/136015#issuecomment-5683234122), on the grounds that these values are not part of the Pod API and therefore sit oddly in an API meant to project pod fields downward.

* **Why Rejected for Alpha**: Its main attraction is that it would remove this KEP's API change altogether, but three things do not carry over. KEP-5304's path and CDI spec are keyed by a claim and a request, and exclusive CPUs managed by the kubelet have neither, so the convention would have to be reinvented along with a way to emit container edits without a DRA driver. Nothing in the pod spec would reference the assignment, so there would be no anchor for the scheduler to filter on and no way for a workload to state that it needs the value — KEP-5304 has the ResourceClaim for that, and this KEP has no equivalent. And a file is then the only possible form, so the environment variable in [Use Cases](#use-cases) would be lost. This remains the most direct answer to the coupling concern in [Risks and Mitigations](#risks-and-mitigations) and a candidate for revisiting.
