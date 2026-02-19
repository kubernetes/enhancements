# KEP-5869: Wildcard Toleration Keys

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
  - [Why not explicit tolerations?](#why-not-explicit-tolerations)
  - [Why not global wildcard?](#why-not-global-wildcard)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: Node Readiness Taints (Primary Use Case)](#story-1-node-readiness-taints-primary-use-case)
    - [Story 2: Grouping taints by vendor](#story-2-grouping-taints-by-vendor)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Scheduler Performance](#scheduler-performance)
    - [Ambiguity](#ambiguity)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Implementation](#implementation)
  - [Example](#example)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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

Extend **core/v1 Toleration** to support **wildcard matching** (glob-style) in the `key` field. This would allow a single toleration to match a family of related taints that share a common prefix.

For example, a toleration with key: "readiness.k8s.io/*" and operator: "Exists" would tolerate any taint whose key begins with `readiness.k8s.io/`, such as `readiness.k8s.io/network-ready` or `readiness.k8s.io/gpu-driver-installed`.

This change would significantly improve the operational ergonomics of managing system-critical components, especially in environments where node taints are governed by a dynamic set of conditions.

## Motivation

Many clusters use taints to manage node readiness, hardware grouping, or specific roles. Often, these taints share a common prefix (e.g., `readiness.k8s.io/`). Currently, to tolerate all such taints, a pod must explicitly list every possible taint key or use a global wildcard (`operator: Exists`, `key: empty`) which is too broad as it tolerates *all* taints.

Platform teams want a way to target a *specific family* of taints without having to update every workload whenever a new specific taint in that family is introduced.

### Why not explicit tolerations?

Explicitly listing every toleration creates a tight coupling between the node taints and the workload manifests. If a new readiness condition is added (e.g., `readiness.k8s.io/new-check`), every DaemonSet or system component needs to be patched to tolerate it. This is operationally burdensome and error-prone.

### Why not global wildcard?

Using `operator: Exists` with an empty key tolerates *everything*, including `node.kubernetes.io/unschedulable`, `node.kubernetes.io/not-ready`, and other critical taints that the workload should likely respect. A scoped wildcard (`prefix/*`) provides the right balance of flexibility and safety.

### Goals

- Allow wildcard matching in toleration keys to match families of taints.
- Improve operational ergonomics for system-critical components.
- Maintain backward compatibility.

### Non-Goals

- supporting wildcard in taint keys on Nodes.
- regex support beyond wildcard matching.
- wildcard support in `value` field (only `key` is in scope).
- new operator / api to support for wildcard matching.

## Proposal

Enable wildcard support in toleration keys via a new feature gate. When enabled, the scheduler will interpret the `*` character in a toleration key as a glob wildcard.

### User Stories (Optional)

#### Story 1: Node Readiness Taints (Primary Use Case)

Projects like the node-readiness-controller allow administrators to define fine-grained readiness rules for nodes using a collection of taints and corresponding conditions.

A node might be tainted with the following until its components are fully initialized:

```bash
readiness.k8s.io/cni-ready:NoSchedule
readiness.k8s.io/storage-driver-ready:NoSchedule
readiness.k8s.io/gpu-driver-ready:NoSchedule
```

The DaemonSets responsible for resolving these conditions (e.g., the CNI agents, or device-driver DaemonSets) must be able to run on these tainted nodes. Currently, this requires adding an explicit toleration for every possible readiness taint to each DaemonSet, or use blanket tolerations as ‘NoSchedule:Exists’

With this proposal, a daemonset installing a critical driver could use a single, stable toleration ‘readiness.k8s.io/*’ to run regardless of the specific readiness taints present:

```yaml
tolerations:
- key: "readiness.k8s.io/*"
  operator: "Exists"
  effect: "NoSchedule"
```

This ensures that as new readiness rules are added, the critical system agents continue to function without requiring manual manifest updates.

#### Story 2: Grouping taints by vendor

Often nodes are tainted to describe specific roles or hardware, for example:

```bash
gpu.vendor.com/model-a100:NoSchedule
gpu.vendor.com/model-h100:NoSchedule
```

A maintenance or monitoring workload that is designed to work with any GPU from that vendor, could tolerate all such nodes with one rule:

```yaml
tolerations:
- key: "gpu.vendor.com/*"
  operator: "Exists"
  effect: "NoSchedule"
```

### Notes/Constraints/Caveats (Optional)

- **Validation**: We must ensure that enabling this feature doesn't break existing validation.
    - Taint keys currently follow the `qualifiedName` validation (IsDNS1123Subdomain). `*` is NOT a valid character in DNS subdomains.
    - We need to relax validation to allow `*` in toleration keys.

### Risks and Mitigations

#### Scheduler Performance
**Risk**: Glob matching is slower than exact string equality. We must
  ensure this does not introduce significant regression.
**Mitigation**:
- The number of taints per node is typically small (<20), so linear scan with glob matching
  should be acceptable.
- Only use glob path if `*` is detected in the key.
- Benchmark the `tainttoleration` plugin.

#### Ambiguity
**Risk**: Users might expect full regex support.
**Mitigation**: Explicitly document that only glob (`*`) is supported.


## Design Details

### API Changes

No schema changes to `Toleration` struct.
However, **Validation** logic in `pkg/apis/core/validation` must be updated.

Current validation for `Toleration.Key` enforces it to be a qualified name (unless empty).
We need to allow `*` in `Toleration.Key` when `WildcardTolerationKeys` feature gate is enabled.

### Implementation

1.  **Feature Gate**: Introduce `WildcardTolerationKeys` (Alpha, default=false).
2.  **Validation**: Update `ValidateTolerations` to allow `*` if feature gate is enabled.
3.  **Scheduler**: Update `pkg/scheduler/framework/plugins/tainttoleration` logic.
    -   In `TolerationsTolerateTaint`:
        -   If `toleration.Key` contains `*`, use `path.Match`.
        -   Else, use strict equality (existing behavior).

### Example

Given a pod with this toleration:

```go
toleration := corev1.Toleration{
    Key:      "readiness.k8s.io/*",
    Operator: corev1.TolerationOpExists,
    Effect:   corev1.TaintEffectNoSchedule,
}
```

The tainttoleration scheduler plugin would evaluate it as follows:

- `taint := corev1.Taint{Key: "readiness.k8s.io/network-pending", Effect: "NoSchedule"}` | Match
- `taint := corev1.Taint{Key: "readiness.k8s.io/storage-not-ready", Effect: "NoSchedule"}` | Match
- `taint := corev1.Taint{Key: "other-taint", Effect: "NoSchedule"}` | No Match

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates
N/A

##### Unit tests

- **Validation Tests**: Ensure `*` is allowed in toleration keys only when feature gate is on.
- **Scheduler Plugin Tests**:
  - Test various wildcard patterns (`prefix/*`, `*/suffix`, `*middle*`).
  - Test interactions with different operators.

##### Integration tests

- **Scheduler Integration**: Verify pods with wildcard tolerations are scheduled on tainted nodes.
- **Feature Gate**: Verify behavior when gate is disabled (should fail validation or not match).

##### e2e tests

- **Scheduling/Taints**:
  1. Create nodes with various taints.
  2. Create pods with wildcard tolerations.
  3. Verify placement.
- **Preemption**:
  1. Verify `NoExecute` taints with wildcards correctly evict pods.

### Graduation Criteria

#### Alpha
- Feature implemented behind `WildcardTolerationKeys` feature gate.
- Validation logic updated.
- Unit and Integration tests passed.

#### Beta
- Feature enabled by default.
- Performance benchmarks confirm negligible impact.
- Feedback gathered.

#### GA
- Stable API.
- No major regressions or bugs.

### Upgrade / Downgrade Strategy

- **Upgrade**: Enable feature gate.
    - Pods can start using wildcard keys.
- **Downgrade**: Disable feature gate.
    - Existing pods with `*` keys might fail validation on update.
    - Scheduler will treat `*` literally (unlikely to match anything since taint keys can't have `*`).
    - Pods relying on wildcards will stop tolerating taints and might be evicted (if `NoExecute`) or fail to schedule.

### Version Skew Strategy

- Scheduler and API Server should be version aligned or API Server should be
  upgraded first to allow the validation. If API Server allows it but Scheduler
  is old, Scheduler will treat `*` literally and fail to match, causing Pending
  pods. This is a standard alpha feature skew.


## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `WildcardTolerationKeys`
  - Components depending on the feature gate: `kube-scheduler`, `kube-apiserver`

###### Does enabling the feature change any default behavior?

No. Only pods using the new wildcard syntax in their tolerations will have different behavior. Existing pods with standard tolerations work as before.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, but it is disruptive for workloads using the feature.

If disabled,
1.  Pods utilizing wildcard tolerations to tolerate `NoExecute` taints will no
longer be protected. The kube-controller-manager fail to match the taint with
wildcard toleration, and immediately evict these pods.
2.  Workload controllers (Deployments, DaemonSets, etc.) with wildcard tolerations in their Pod templates will fail to create new Pods. The kube-apiserver validation will reject the `*` character, causing the controller to stall with validation errors.
3.  Pods utilizing wildcard tolerations for `NoSchedule` taints will fail to schedule as the kube-scheduler will treat the wildcard key.

To mitigate these failure modes, administrators must identify and update all their workloads using wildcard tolerations to use explicit keys or standard `Exists` operators before disabling the feature gate.

###### What happens if we reenable the feature if it was previously rolled back?

Validation allows `*` again. Scheduler honors wildcard matching again.

###### Are there any tests for feature enablement/disablement?

Integration tests will cover enabling/disabling the feature gate.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollback Failure Modes:
- Mass Eviction
: As described above, disabling the feature causes immediate eviction of running pods that relied on wildcards to tolerate `NoExecute` taints.
- Stalled Workloads
: Workloads defined with wildcard tolerations will be unable to scale up or replace terminated pods due to API validation failures.

To avoid the impact, rollback should be preceded by cleaning up usage of the
wildcard keys in the cluster.

###### What specific metrics should inform a rollback?

- Increase in pending pods.
- Increase in `FailedScheduling` events.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

To be tested in Beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- **grep for wildcard toleration in use using API Queries**:
  `kubectl get pods -A -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.tolerations[*].key}{"\n"}{end}' | grep "\*"`

###### How can someone using this feature know that it is working for their instance?

Pods with wildcard tolerations successfully schedule on nodes with matching taints.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `scheduler_framework_extension_point_duration_seconds{plugin="TaintToleration"}`
  - Components exposing the metric: `kube-scheduler`

###### Are there any missing metrics that would be useful to have to improve observability of this feature?


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

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Due to glob matching overhead, it could add to the time taken for
scheduler pass, but the impact should be minimal as the number of tolerations
are minimal. 

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

It is expected to be minimal, but increased CPU usage in `kube-scheduler` is possible due to the overhead of glob matching in the critical scheduling path.

To monitor this, we will introduce a metric `scheduler_wildcard_match_duration_seconds` to track the latency of wildcard matching operations. This will allow us to quantify the performance impact during Alpha.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

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

- 2026-02-02: Initial KEP


## Drawbacks

## Alternatives

## Infrastructure Needed (Optional)

No infrastructure needed
