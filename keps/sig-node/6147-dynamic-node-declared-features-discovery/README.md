# KEP-6147: Dynamic node declared features Discovery

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
    - [Story 1: Container Runtime Hot Upgrade](#story-1-container-runtime-hot-upgrade)
    - [Story 2: Workloads Continue Running After Kubelet Rollback](#story-2-workloads-continue-running-after-kubelet-rollback)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Dynamic Runtime Feature Discovery](#dynamic-runtime-feature-discovery)
  - [Kubelet Admission Policy Change](#kubelet-admission-policy-change)
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

This KEP proposes two improvements to [KEP-5328 Node Declared Features](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/5328-node-declared-features):

1. The kubelet currently discovers container runtime-provided features only at startup. If the container runtime is upgraded after the kubelet has started, new runtime capabilities are not reflected in `node.status.declaredFeatures` until the kubelet is restarted. This proposal makes the kubelet dynamically re-discover runtime features during each `updateRuntimeUp` cycle, without requiring a kubelet restart.

2. KEP-5328 currently specifies that when the kubelet restarts with fewer declared features, any running Pod whose requirements are no longer met will be transitioned to the `Failed` phase. This behavior is inconsistent with how other kubelet feature gates handle rollbacks—the established best practice is to allow already-running workloads to continue, rather than actively terminating them. This proposal restricts the admission check to newly scheduled Pods only, and skips it for Pods already running on the node at the time of kubelet restart.

## Motivation

KEP-5328 defines a class of declared features that depend not only on a kubelet feature gate being enabled, but also on the container runtime (e.g., containerd) supporting the corresponding capability. Runtime capabilities are expressed as boolean fields in the CRI `RuntimeFeatures` message (e.g., `UserNamespacesHostNetwork`). The kubelet reads these via `runtimeState.runtimeFeatures()` inside `discoverNodeDeclaredFeatures()` to decide which features to include in `node.status.declaredFeatures`.

However, the current implementation has two problems:

**Problem 1: Runtime features are discovered only once, at kubelet startup.**

`discoverNodeDeclaredFeatures()` is called once in `NewMainKubelet` and the result is never refreshed. This means:

- If a cluster administrator upgrades containerd in-place without restarting the kubelet, newly supported runtime features (e.g., `UserNamespacesHostNetwork` in a newer containerd release) will not appear in `node.status.declaredFeatures` until the next kubelet restart.
- Restarting the kubelet is not without risk. The community has filed multiple PRs and KEPs specifically addressing container state inconsistencies that arise from kubelet restarts. Requiring administrators to restart the kubelet just to pick up a new runtime capability creates unnecessary operational risk.

**Problem 2: The admission behavior is inconsistent with other feature gate rollbacks.**

The [Declared Feature Changes on Existing Nodes](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/5328-node-declared-features#declared-feature-changes-on-existing-nodes) section of KEP-5328 states:

> If a kubelet restarts with a different configuration (e.g., a feature gate is disabled), its declared features may change. Upon restart, the kubelet re-evaluates all existing pods for admission, and if a running pod requires a declared feature that the node no longer provides, the kubelet's admission check will fail. Consequently, the pod will not be started and will be transitioned to a `Failed` phase.

For typical kubelet feature gate rollbacks—where the kubelet is downgraded to a version that does not support a given feature—already-running workloads continue to run uninterrupted. The `NodeDeclaredFeatures` framework deviates from this established behavior, exposing administrators to unexpected workload disruption when performing minor kubelet rollbacks or toggling feature gates.

**Scheduler perspective.**

The `kube-scheduler` does not observe whether the kubelet has restarted; it only watches for changes to `node.status.declaredFeatures`. Therefore, whether a new feature is added to `node.status.declaredFeatures` via a kubelet restart or via dynamic discovery makes no difference to the scheduler. This confirms that dynamic runtime feature discovery has no impact on scheduling logic.

### Goals

1. Enable the kubelet to dynamically re-discover container runtime-provided declared features and update `node.status.declaredFeatures` in response to runtime status updates, without requiring a kubelet restart.
2. Change the kubelet admission policy so that the declared features check applies only to **newly scheduled Pods** and is skipped for Pods already running on the node when the kubelet restarts.

### Non-Goals

1. Modifying declared feature handling in `kube-scheduler` or `kube-apiserver`.
2. Changing scheduling behavior triggered by newly added runtime features (the scheduler will automatically observe `node.status.declaredFeatures` updates).
3. Providing an active migration mechanism for running Pods whose feature dependencies are no longer satisfied—this proposal simply avoids terminating them.
4. Changing how the `NodeDeclaredFeatures` framework handles declared features driven by static configuration (feature gates); this proposal is scoped to features whose discovery depends on runtime-reported capabilities.

## Proposal

This KEP updates the Node Declared Features kubelet behavior in two ways:

1. When both `NodeDeclaredFeatures` and `NodeDeclaredFeaturesRuntimeDiscovery`
   are enabled, the kubelet re-discovers runtime-backed declared features during
   the existing runtime status update path and updates `node.status.declaredFeatures`
   when the discovered set changes.
2. The kubelet declared feature admission check continues to apply to newly
   admitted Pods, but is skipped for Pods that were already running on the node
   when the kubelet restarts.

### User Stories (Optional)

#### Story 1: Container Runtime Hot Upgrade

A cluster administrator upgrades containerd in-place on a node. The new version of containerd adds support for `UserNamespacesHostNetwork`.

**Current behavior:**
- After upgrading containerd, `UserNamespacesHostNetwork` does not appear in `node.status.declaredFeatures`.
- The administrator must restart the kubelet to trigger feature discovery and update node status.
- Restarting the kubelet carries potential container state inconsistency risk.

**Expected behavior:**
- After upgrading containerd, the kubelet detects the runtime capability change during the next `updateRuntimeUp` cycle (default: ~10 seconds).
- The kubelet re-discovers declared features and adds `UserNamespacesHostNetwork` to `node.status.declaredFeatures`.
- The scheduler observes the new feature and begins placing Pods that require it onto the node.
- No kubelet restart is required throughout this process.

#### Story 2: Workloads Continue Running After Kubelet Rollback

A cluster administrator needs to roll back a kubelet feature by setting a feature gate to false. This feature gate influences which features are included in `node.status.declaredFeatures`.

**Current behavior:**
- After restarting the kubelet, all existing Pods are re-admitted.
- Running Pods that require a feature the node no longer declares are transitioned to `Failed`.
- Workloads are disrupted unexpectedly.

**Expected behavior:**
- After rolling back the feature, the kubelet does not perform declared feature admission checks against already-running Pods (consistent with other feature gate rollback behavior).
- Running Pods continue to run, though the feature may no longer be fully supported.
- No new Pods that depend on the removed feature will be scheduled onto this node.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

**Risk 1: Incompatible Pods running long-term after a rollback**

After a kubelet rollback, a running Pod may depend on a feature that is no longer supported, potentially leading to runtime errors.

**Mitigation:** This is consistent with the behavior of other feature gate rollbacks. There is no active mitigation; if running workloads are affected, users can manually evict them.

## Design Details

### Dynamic Runtime Feature Discovery

**Current behavior (KEP-5328):**

`discoverNodeDeclaredFeatures()` is called once in `NewMainKubelet` and its result is cached in `nodeDeclaredFeatures` and `nodeDeclaredFeaturesSet`, which are never updated afterward.

**Proposed change:**

Feature discovery is moved into a new `updateNodeDeclaredFeatures()` method that is called both during initialization and on every `updateRuntimeUp()` invocation. This ensures that runtime capability changes (such as a new feature becoming available after a containerd upgrade) are promptly reflected in `node.status.declaredFeatures` without a kubelet restart. If the re-discovered feature set is identical to the previous one, no `node.status` write is triggered. Since `updateRuntimeUp()` runs concurrently with node status reporting and admission checks, access to the relevant fields is protected by a `sync.RWMutex`, exposed to callers through getter functions.

### Kubelet Admission Policy Change

**Current behavior (KEP-5328):**

On kubelet restart, all existing Pods on the node (including already-running ones) are re-admitted. If a Pod's required declared features are no longer present on the node, `declaredFeaturesAdmitHandler` fails the admission check, and the Pod is transitioned to `Failed`.

**Proposed change:**

`declaredFeaturesAdmitHandler.Admit()` distinguishes between two categories of Pods:

1. **Newly scheduled Pods:** `pod.spec.nodeName` has just been set by the scheduler and the Pod has not yet run on this node. These Pods go through the full declared feature admission check.
2. **Re-admitted running Pods:** The Pod is already running on this node (recorded in the local pod cache) and the kubelet is simply reloading it after a restart. These Pods skip the `declaredFeaturesAdmitHandler` check.

**Distinguishing criterion:**

The distinction is made by inspecting the Pod's status. If `pod.Status.Phase` is `Running` (or the Pod is present in `podCache`), it indicates the Pod has previously passed admission on this node and begun running, so the declared feature check is skipped.

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

##### Unit tests

- `pkg/kubelet/`: 2026-06-04 - 92.2%
- - `pkg/kubelet/lifecycle`: 2026-06-04 - 63.7%

##### Integration tests

N/A. The e2e behavior is validated through the e2e tests of concrete features that use this framework (e.g., `UserNamespacesHostNetwork`).

##### e2e tests

Dedicated e2e tests are out of scope for this KEP. e2e coverage is provided by existing e2e tests for features that depend on the `NodeDeclaredFeatures` framework (e.g., `UserNamespacesHostNetwork`).

### Graduation Criteria

#### Alpha

- A new feature gate `NodeDeclaredFeaturesRuntimeDiscovery` is introduced, disabled by default, gating only the kubelet.
- Both improvements (dynamic runtime discovery and the updated admission policy) are protected by `NodeDeclaredFeaturesRuntimeDiscovery` and take effect only when `NodeDeclaredFeatures` is also enabled.

#### Beta

- Feature gate `NodeDeclaredFeaturesRuntimeDiscovery` is enabled by default.
- Upgrade/downgrade tests verify the stability of the new admission policy (running Pods remain running after a kubelet restart with reduced declared features).
- No significant regressions in kubelet performance have been observed.

#### GA

- Feature gate `NodeDeclaredFeaturesRuntimeDiscovery` is locked to enabled and subsequently removed as part of the standard feature gate cleanup process.
- At least two releases have passed since Beta with no regressions or critical bug reports.


### Upgrade / Downgrade Strategy

### Version Skew Strategy

All changes in this KEP are kubelet-only and do not involve any changes to control plane components.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `NodeDeclaredFeaturesRuntimeDiscovery`
  - Components depending on the feature gate: kubelet
  - Prerequisite: `NodeDeclaredFeatures` (introduced by KEP-5328) must also be enabled.

###### Does enabling the feature change any default behavior?

Yes, in two ways:
1. The kubelet periodically (rather than only at startup) updates `node.status.declaredFeatures` to reflect the latest runtime capabilities.
2. On kubelet restart, Pods already in the `Running` phase are no longer transitioned to `Failed` due to a reduction in declared features.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Setting `NodeDeclaredFeaturesRuntimeDiscovery=false` and restarting the kubelet reverts both behaviors:
- Runtime feature discovery reverts to startup-only.
- On restart, all Pods are re-admitted against the current declared feature set (KEP-5328 behavior).

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

Manual testing will also be conducted during the beta stage, and the testing process will be documented here.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

- Rollout: Existing workloads are not affected. Dynamic discovery is purely additive; if re-discovery fails (e.g., CRI call timeout), the kubelet retains the last successfully discovered feature set.
- Rollback: Running workloads may be affected on the first kubelet restart after the downgrade, because the older kubelet re-admits all Pods.

###### What specific metrics should inform a rollback?

No.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This will be validated via manual testing. 

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements


###### How can an operator determine if the feature is in use by workloads?

Inspect `node.status.declaredFeatures` to verify that expected runtime-capability-dependent features (e.g., `UserNamespacesHostNetwork`) are present.

###### How can someone using this feature know that it is working for their instance?

- [x] API .status
  - Condition name: 
  - Other field: Other field: `node.status.declaredFeatures` is updated within ~10 seconds of a container runtime upgrade, without restarting the kubelet.


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

No.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- [KEP-5328 Node Declared Features](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/5328-node-declared-features)
  - Usage description: `NodeDeclaredFeatures` feature gate must be enabled; this KEP improves the discovery and admission behavior within that framework.
  - Impact of its absence: `NodeDeclaredFeaturesRuntimeDiscovery` has no effect if `NodeDeclaredFeatures` is disabled.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The kubelet continues to re-discover runtime features locally. The updated feature set will be written to `node.status` once the API server becomes available again, consistent with the existing node heartbeat behavior.

###### What are other known failure modes?

No.

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 2026-06-07: KEP draft created.

## Drawbacks

The admission policy change means that, after a kubelet feature rollback, Pods that depend on a feature no longer supported by the node may continue running and potentially encounter runtime errors. This is an intentional trade-off that prioritizes avoiding unexpected workload disruption.

## Alternatives

**Require a Kubelet Restart to Discover New Runtime Features:**

The simplest alternative is to maintain the current behavior: after a container runtime upgrade, require the cluster administrator to manually restart the kubelet to trigger a one-time feature discovery and update `node.status.declaredFeatures`.

**Why rejected:**

- Restarting the kubelet is not a zero-risk operation. The community has filed multiple PRs and KEPs (e.g., [KEP-5532](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/5532-restart-all-containers-on-container-exits)) specifically to address container state inconsistencies that arise from kubelet restarts.
