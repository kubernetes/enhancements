
# KEP-5328: Node Capabilities


<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Existing Mechanisms and Limitations](#existing-mechanisms-and-limitations)
- [Proposal](#proposal)
- [User Stories](#user-stories)
  - [Story 1: Feature Rollout Challenges with Version Skew](#story-1-feature-rollout-challenges-with-version-skew)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
    - [Capability Reporting Semantics](#capability-reporting-semantics)
  - [Kubelet Changes](#kubelet-changes)
  - [Shared Capability Matching Library](#shared-capability-matching-library)
  - [kube-scheduler Changes](#kube-scheduler-changes)
    - [Plugin Implementation](#plugin-implementation)
    - [Performance Considerations](#performance-considerations)
  - [Admission Controller Changes](#admission-controller-changes)
  - [Capability Lifecycle and Deprecation](#capability-lifecycle-and-deprecation)
  - [Walkthrough](#walkthrough)
  - [Capability Changes on Existing Nodes](#capability-changes-on-existing-nodes)
  - [Integration with Existing Mechanisms](#integration-with-existing-mechanisms)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Upgrade](#upgrade)
    - [Downgrade](#downgrade)
  - [Version Skew Strategy](#version-skew-strategy)
  - [Future Considerations](#future-considerations)
    - [Explicit Capability Request](#explicit-capability-request)
    - [Cluster Autoscaler Integration](#cluster-autoscaler-integration)
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
  - [Using a Meta-Capability as an Opt-In Signal](#using-a-meta-capability-as-an-opt-in-signal)
  - [Using Node Labels and Node Affinity with SemVer comparison](#using-node-labels-and-node-affinity-with-semver-comparison)
  - [Introducing a <code>NodeCapabilities</code> API](#introducing-a-nodecapabilities-api)
  - [Alternative Naming Conventions](#alternative-naming-conventions)
    - [Alternative Names for the Capabilities Field](#alternative-names-for-the-capabilities-field)
    - [Alternative Naming Conventions for Capability Keys](#alternative-naming-conventions-for-capability-keys)
<!-- /toc -->

## Release Signoff Checklist


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


[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes a **Node Capabilities** framework for Kubernetes nodes to report capabilities that are tied to the lifecycle of Kubernetes features. The primary goal is to provide a standard mechanism to manage version skew between the control plane and nodes, which helps control plane components make better decisions.

For scheduling, the kube-scheduler would utilize these advertised capabilities to ensure pods are only placed on nodes that possess the necessary features to run them successfully. For API request validation, admission controllers would prevent operations on nodes that lack the required feature support.

The intent is to streamline cluster operations by reducing the reliance on manual configurations like taints, tolerations, and complex node labeling schemes.

## Motivation

The primary motivation for this KEP is to solve scheduling and validation problems that arise from version skew. When a new feature is enabled on the control plane, a mismatch often occurs because nodes are upgraded gradually or are simply running older Kubelet versions, as is permitted by the Kubernetes version skew policy. This creates a window where the scheduler might place a pod requiring a new feature onto a node that does not support it. 

By making the scheduler aware of specific node capabilities,  this proposal ensures that such incompatibilities are handled correctly and proactively. Instead of failing later with a runtime error or a Kubelet admission failure on the node, a pod that cannot be matched to a capable node will be identified as a clear scheduling failure. The pod will remain in the Pending state with an event explaining exactly which capability was missing, providing immediate and actionable feedback to the user ([slack discussion](https://kubernetes.slack.com/archives/C5P3FE08M/p1741867194258139)).

### Goals

1. Define a standard mechanism for nodes to expose capabilities that are tied to the lifecycle of **new Kubernetes features** to manage version skew.
2. Introduce a shared library to encapsulate the logic for inferring a pod's requirements and matching them against node capabilities, ensuring consistency between control plane components that depend on capabilities.
3. Enhance the kube-scheduler to filter nodes based on the pod's requirements.
4. Enable API admission controllers to validate requests for operations against a node's actual feature support.
5. Enable Kubelet admission plugin to check if the Pod is compatible with the Node's features  

### Non-Goals

1. Replace Taints/Tolerations or Node Labels/Selectors/Affinity.
2. Serve as a reporting mechanism for permanent static node attributes (like architecture, or specific hardware).
3. To define the exact mapping of a feature to a capability. This KEP proposes the framework that establishes the mechanism; specific mappings will be defined with the features that use them.
4. This framework will not be applied to Kubernetes features that are already implemented. The capability reporting mechanism is designed to support only new features introduced after this framework is in place.
5. To include full Cluster Autoscaler integration in the initial Alpha stage. The autoscaler makes scaling decisions based on node templates, which lack the capability information. Defining an integration strategy is deferred as a [future enhancement](#cluster-autoscaler-integration).


## Existing Mechanisms and Limitations

The Kubernetes scheduler currently uses two primary mechanisms to control pod placement onto specific nodes:

1. Taints and Tolerations 

    Primarily used to **restrict** which pods can schedule onto specific nodes. Commonly used to manage specialized hardware resources. 

    **Standard Usage Pattern:**
*   Cluster Administrators apply taints to nodes equipped with special capabilities; cloud providers may also automate this tainting for certain NodePool configurations.
*   Developers add corresponding tolerations in their Pod specifications for workloads to be able to run on these nodes. Alternatively, cluster administrator or cloud provider could also inject tolerations through admission webhooks. 

2. Node Labels and Node Selectors/Affinity

    Primarily used to **attract** specific nodes for pods based on the node's characteristics. By applying specific labels to nodes (reflecting Kubelet features, OS version, etc.), we can enable pods to use selectors or affinity to ensure they run on specific nodes.

    **Standard Usage Pattern:**
*   Cluster Administrators apply specific Labels to nodes to indicate the presence of certain features. This involves applying descriptive labels (e.g., `kubelet.config.k8s.io/some-alpha-feature=enabled`, `node.kubernetes.io/gvisor-enabled=true`)
*   Optionally, for well-defined features may need to create other resources (e.g., [RunTimeClass](https://kubernetes.io/docs/concepts/containers/runtime-class/)) that bundle these feature requirements with a node selector targeting the corresponding node labels.
*   Developers specify their workload's dependency on these features in the PodSpec either directly (spec.nodeSelector) or other abstractions enabling the scheduler to match them to capable nodes.

**Drawbacks**:

1. Operational Overhead: Cluster administrators have to add the necessary taints/labels to nodes as indirect signals of features and resources, and workloads should use corresponding tolerations/selectors. This needs to be done manually or automations built (webhooks, controllers) to handle standard usage patterns. 
2. Scheduling constraints are encoded indirectly rather than being implicitly understood by kube-scheduler. 
3. Incorrect configurations can lead to scheduling failures or suboptimal placements.

## Proposal

This proposal introduces a new field `Capabilities` to `Node.Status` to expose information which the kube-scheduler and/or the admission controller would use to make more informed decisions. The Kubelet is primarily responsible for discovering, consolidating, and reporting `Capabilities` to the API server.

**Node Capabilities Requirements:**

1. Every capability must be associated with a Kubernetes feature graduating through the Alpha/Beta/GA process. This ensures capabilities are not used as permanent node attributes and are automatically removed after the feature is stable (after the supported version skew period).
2. The NodeCapabilities framework must be used for new features introduced after the framework. Onboarding existing features must be avoided as it would create ambiguity; the control plane would be unable to differentiate between a node that has the feature but lacks capability reporting for it, and a node that genuinely lacks the feature.
3. Must be derived from node's static configuration and determined at startup. Kubelet must determine all capabilities during its bootstrap sequence, before its admission handlers are active. Reporting new or changed capabilities requires a Kubelet restart to take effect. Capabilities do not indicate operational health or Node Readiness. [Node Readiness Gates](https://github.com/kubernetes/enhancements/issues/5233) are better suited for such dynamic readiness signals.
4. Must be actionable by the control plane. A capability is only relevant if it can be used by a control plane component (like kube-scheduler or an admission controller) to make a decision, such as filtering a node or validating an API request.
5. Must be validated by the Kubelet at runtime. This is to prevent errors from post-scheduling capability changes.

## User Stories 

### Story 1: Feature Rollout Challenges with Version Skew

To maintain cluster stability and enable safer rollouts, cluster administrators perform gradual upgrades. This inherently creates a mixed-version Kubernetes cluster, where nodes can be on different versions than the control plane. This environment introduces significant challenges for both pod scheduling and API request validation. The primary concerns in such clusters are:

1. How do we prevent new pods that might use newer features from being scheduled on older, incompatible nodes?
  * Example: [Pod Level Resources](https://github.com/kubernetes/enhancements/blob/63d4f6f2aa0e2eb0b83067b067c4949643b1b24c/keps/sig-node/2837-pod-level-resource-spec/README.md?plain=1#L4)
    * The existing beta feature lacks explicit version-skew management, requiring it to be enabled on all components (scheduler, kubelet, API server) to function correctly. If enabled only on the control plane, a pod can be scheduled on an incompatible node where the Kubelet will reject it during admission. This rejection puts the pod into a `Failed` state, preventing it from being retried on other newer (compatible) nodes in the cluster.
    * For new enhancements, like adding support for CPU alignment (static CPU policy support) there is no way to make sure a guaranteed QOS pod using pod level resources lands on a node supporting this new feature.

2. How do we block API calls that target pods on nodes that don't support newer features?
  * Example: [In-Place Pod Resizing](https://kubernetes.io/docs/tasks/configure-pod-container/resize-container-resources/)
    * For an existing beta feature, there currently exists a [workaround](https://github.com/kubernetes/kubernetes/blob/23258f104d74c6f27fd4db94940d745d9d463a8f/pkg/apis/core/validation/validation.go#L5796) to handle this version skew by looking for alternate signals from the pod spec. However such workarounds may be complex and not always feasible for new features.
    * For a new feature enhancement like [In-Place Pod Resize for Guaranteed QoS pods](https://github.com/kubernetes/enhancements/issues/5294), an API request to modify a running pod must be validated. The operation should be rejected if the pod resides on an older node that does not support the feature.

The Node Capabilities framework addresses both issues through a unified mechanism without requiring any intervention (like adding Node Labels, selectors etc.). The scheduler uses the capability information to correctly filter nodes for new pods, while the admission controller validates operations against the capabilities of a pod's current node. This ensures that feature incompatibilities are handled proactively at the control plane.

## Design Details

### API Changes

Add a `Capabilities` field as type `map[string]string` to the `Node.Status` structure. 

```
type NodeStatus struct {
    // ... existing fields
    // Capabilities provides a structured way to report various capabilities of the node. Keys are DNS-style and values are strings.
    // +optional 
    // +featureGate=NodeCapabilities
    Capabilities map[string]string `json:"capabilities,omitempty"`
}


// Node object remains unchanged in spec, only status is modified.
type Node struct {
    ...
    Status NodeStatus `json:"status,omitempty" `
}

```

**Note:**  

*   Any new capability being introduced is considered a formal API change and must go through the API review process. This governance will be enforced in `kubernetes/kubernetes` codebase by protecting the list of capabilities with the `api-approvers` OWNERS file.
*   We currently have [Node Features](https://github.com/kubernetes/api/blob/e8d4d542f6a9a16a694bfc8e3b8cd1557eecfc9d/core/v1/types.go#L6279) and [Node Runtime Features](https://github.com/kubernetes/api/blob/e8d4d542f6a9a16a694bfc8e3b8cd1557eecfc9d/core/v1/types.go#L6251) which publish some runtime features through Node Status. They are too narrowly scoped and currently not used for scheduling pods. We can deprecate those fields and introduce them into NodeCapabilities if required. 


#### Capability Reporting Semantics

1.  Naming Convention
    * To make the temporary nature of these capabilities explicit, all capability keys must use the `lifecycle.kubernetes.io/` prefix (few other [alternatives](#alternative-naming-conventions) considered).
    * This naming convention serves as a clear signal to all consumers that the capability is not a permanent node attribute and should not be depended on long-term. It ties the capability to the formal Kubernetes feature lifecycle (Alpha → Beta → GA), which includes a mandatory removal phase ([capability deprecation](#capability-lifecycle-and-deprecation)).

2.  Combine Interdependent Settings
    * If multiple settings are required to enable a feature, they must be collapsed into a single, logical capability.
    * This simplifies decision-making for control plane components like the scheduler, which should not need to understand multiple interdependent settings.

3.  Presence of a Capability
    * A capability should only be present in the `node.status.capabilities` map if the feature is enabled and functional.
    * If a feature is disabled or unsupported, the Kubelet must remove the corresponding capability key from the map.
    * This approach ensures consistency and simplifies logic for the control plane, treating nodes that don't know about a feature the same as those that have it disabled.

4.  Validation Rules
    * Keys and values within the `node.status.capabilities` map must adhere to the same validation rules as standard Kubernetes labels.
    * The Kubelet validates each capability and will discard any invalid key-value pair, logging an error.
    * When introducing a new capability that requires a Kubelet change, developers must ensure that the new keys and values meet these validation rules. This would also be enforced by [unit tests](#unit-tests).


**Example:**

```
capabilities:
   lifecycle.kubernetes.io/guaranteed-qos-pod-cpu-resize: "true"
```

### Kubelet Changes

Kubelet has the primary responsibility of discovering the capabilities during bootstrap and updating `node.status.capabilities`.

In addition to reporting, Kubelet will also need to validate the pod. Before admitting a pod to be run on the node, the Kubelet's internal sync loop must check if the pod requires any specific capabilities (as inferred from its spec). It will validate these requirements against its own live, in-memory map of supported capabilities. If a required capability is not present, the Kubelet will reject the pod, transition it to a Failed phase, and post an appropriate event to the API server detailing the missing capability. This secondary validation (along with kube-scheduler capability based filtering) is necessary to handle node restart scenarios where a capability that existed during pod scheduling does not exist anymore (feature gate flip with node restart).

As part of the feature's graduation to GA and subsequent stabilization, the Kubelet must stop reporting the capability. This will occur after the feature has been GA for a duration that exceeds the supported version skew. This ensures that obsolete information is automatically removed from `node.status` object.

### Shared Capability Matching Library

To avoid code duplication and ensure that all control plane components make decisions based on the same logic, a new shared library will be introduced. This would be added in [component-helpers](https://github.com/kubernetes/component-helpers) staging repository which could be shared by all the components that need capability matching.

**Initialization**

The library will be initialized by each consuming component (e.g., kube-scheduler) and will be provided with two key dependencies:
* The component's version: This allows the library to handle the automatic deprecation of capability checks over time. For example, if a capability is tied to a feature that graduated to GA in v1.35 and the supported version skew is three releases, a component running v1.39 or newer can automatically bypass the check, as the feature is guaranteed to be present.
* A handle to the component's `FeatureGate` object: This allows the library to check if a feature is enabled before enforcing a capability match.

**Core Functionality**

The Node Capability Helper will expose functions that logically separate pod inspection and per-node matching.
1. Inferring Requirements: The library will provide a function to analyze the PodSpec and compute a set of its capability requirements for pod creation or update.
2. Matching Requirements: A second function will take the pre-computed requirements and check them against a specific node.

```
// NodeCapabilityHelper creates a new helper instance.
// It requires the version of the consuming component (e.g., kube-scheduler) and a handle to its feature gate map.
func NodeCapabilityHelper(componentVersion *version.Version, featureGates featuregate.FeatureGate) (*NodeCapabilityHelper, error)

// InferCreateRequirements inspects a new pod and returns the set of capabilities
// required for its initial scheduling.
func (h *NodeCapabilityHelper) InferCreateRequirements(pod *v1.Pod) (*PodRequirements, error)

// InferUpdateRequirements inspects the change between an old and new pod spec
// and returns the set of capabilities required to validate the update operation.
func (h *NodeCapabilityHelper) InferUpdateRequirements(oldPod, newPod *v1.Pod) (*PodRequirements, error)

// MatchNode checks if a node's advertised capabilities satisfy
// the pre-computed requirements for a pod.
func (h *NodeCapabilityHelper) MatchNode(reqs *PodRequirements, node *v1.Node) (bool, error)
```

This design provides a clear interface for consumers:
* kube-scheduler would call `InferCreateRequirements` once during the `PreFilter` stage and then call `MatchNode` for each node during the Filter stage. 
* Admission controller when validating a pod update, would call `InferUpdateRequirements()` with the current and the new pod spec to infer the requirement and then call `MatchNode` with the node object from `pod.spec.nodeName`.

### kube-scheduler Changes

To enable capability-based scheduling, this KEP proposes a new scheduler plugin named `NodeCapabilityFilter`.

#### Plugin Implementation

`NodeCapabilityFilter` plugin would be enabled if the feature gate is enabled in the kube-scheduler and would implement two extension points in the [scheduling framework](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/). 

1. PreFilter
    *  In this stage, the plugin will pass the pod spec to the shared library to compute a set of its capability requirements. This result will be cached in the scheduler's prefilter state.
2. Filter: 
    * For each node being evaluated, the plugin will retrieve the cached requirements and pass them along with the Node object to a matching function in the shared library.
    * The library itself is responsible for handling all complex logic. The plugin simply acts on the boolean result returned by the library to either filter the node or pass it.

#### Performance Considerations

Introducing this plugin adds a new step to the scheduling cycle, which may have an impact on scheduling throughput. However, this trade-off is acceptable because it prevents the greater inefficiency of scheduling a pod onto a node where it cannot run.


### Admission Controller Changes

To enable the validation of API requests against Node Capabilities, this KEP proposes the introduction of a new admission controller plugin `NodeCapabilityValidator` that will be enabled when the `NodeCapabilities` feature gate is active. This admission controller cross-references the objects, i.e., looking  up the `Node` object that a `Pod` is bound to (spec.nodeName) and runs the validation checks.

The admission controller workflow will be as follows:
*   Inspect an incoming pod update request and verifies that the pod is already bound to a node by checking that `pod.spec.nodeName` is set. If not, it takes no action.
*   Call the shared library to infer the capability requirements.
*   Retrieves the Node object corresponding to `pod.spec.nodeName` and pass it along with the inferred requirements  to the shared library's matching function.
*   If the function returns false, the admission controller rejects the request.

### Capability Lifecycle and Deprecation

To prevent the `node.status.capabilities` map from becoming an ever-growing collection of features,  every capability tied to a feature's lifecycle must be automatically removed from the `node.status` object.

The lifecycle stages are as follows:

* Introduction (Alpha/Beta): A new capability that is introduced alongside a new feature. Kubelet begins reporting the capability on nodes where the feature is enabled and active. Control plane components, via the shared library, start using this capability to make decisions.
* Graduation (GA): When the feature graduates to GA, the Kubelet continues to report the capability. This is necessary to manage version skew, allowing the control plane to correctly identify older nodes that do not yet have the GA feature.
* Automated Deprecation (Post-GA): The reporting and checking of a capability are automatically disabled once a feature is old enough to be considered universally present. This logic is executed independently by the Kubelet and the control plane's shared library.
    * Kubelet automatically stops reporting the capability after the feature has been GA for a duration that exceeds the cluster's supported version skew. 
    * Control Plane: The shared library, using the version and FeatureGate object provided by the consuming component (e.g., kube-scheduler) at initialization, bypasses checks for the same capability. For example, if a feature goes GA in v1.38 and the skew is 3 versions, a
  kube-scheduler running v1.42 or later will bypass that feature's capability check.
* Cleanup: The deprecated reporting and inference logic is removed from the codebase and can coincide with the removal of feature gate itself.

### Walkthrough

This walkthrough demonstrates the end-to-end lifecycle of the [In-Place Pod Resize for Guaranteed QoS pods](https://github.com/kubernetes/enhancements/issues/5294) feature using the Node Capabilities framework.

**Phase 1: Feature Development**
  * Kubelet Changes
    *  A new capability key is defined:`lifecycle.kubernetes.io/guaranteed-qos-pod-cpu-resize`. The Kubelet is updated to report this capability as "true" only if all underlying dependencies are met
        *  `InPlacePodResize` and `InPlacePodVerticalScalingExclusiveCPUs` feature gates are enabled and CPU Manager Policy is set to `static`. 
        * Or, `InPlacePodResize` feature gate is enabled and CPU Manager Policy is `none`.
  * Shared Library Changes
    *  The shared library's inference logic is updated with a new rule to check for the `lifecycle.kubernetes.io/guaranteed-qos-pod-cpu-resize` capability if the CPU request of a guaranteed QOS pod is modified.
  * No code changes are needed in the  `NodeCapabilityValidator` admission plugin itself. It already calls the shared library, so it will automatically pick up the new logic.

**Phase 2: Rollout**
  * Cluster administrator enables the new feature `InPlacePodVerticalScalingExclusiveCPUs` along with `InPlacePodVerticalScaling`  and `static` CPU Manager Policy on a NodePool.
  * A user requests an in-place CPU resize for a pod on an upgraded node.
  * The `NodeCapabilityValidator` admission controller intercepts the update. It calls the shared library to infer requirements for the change.
  * The library's newly added logic correctly identifies the need for the `lifecycle.kubernetes.io/guaranteed-qos-pod-cpu-resize` capability. The library then matches this requirement against the node's reported capabilities.
  * The request is admitted only if the node that the pod is currently running on supports the feature.

**Phase 3: Post-GA Deprecation**
  * After the feature graduates to GA and the supported version skew window has passed, the feature is considered default. Kubelet will automatically stop advertising the capability and the shared library will bypass the capability check.
  * Code cleanup - the reporting logic is removed from kubelet and the inference logic is removed from the shared library.

### Capability Changes on Existing Nodes

Node capabilities are checked by the scheduler during scheduling and then validated again by the Kubelet before a pod's containers can start. If a node's capability is removed (kubelet restart with a feature gate disabled) while a pod is already running, the pod is not automatically removed. Kubelet will re-validate the pod's requirements against the node's current state upon container or node restart, and if a required capability is now missing, it will prevent the container from starting, causing the pod to enter a failed state.

### Integration with Existing Mechanisms

We should ideally have one signal to express scheduling intent, but during transition we might end up having multiple active mechanisms to achieve Pod-to-Node matching in kube-scheduler. 

**Scenario 1**  
If there is an existing mechanism (like node labels and selectors) and now we introduce a new node capability to manage feature availability on the node, the scheduling restrictions are additive. The `NodeCapabilityFilter` does not override other filters; it works alongside them. A pod must satisfy the requirements of all active filters.

**Scenario 2** 
If a pod could previously schedule on any node, the new `NodeCapabilityFilter` may now proactively filter out nodes that lack a required capability. This is the intended behavior of the feature. It ensures that pods are not scheduled on nodes where they would fail, which is analogous to how a missing label prevents a nodeSelector match.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

The examples described in the [Example Walkthrough](#example-walkthrough) section can be used to demonstrate and test Node Capabilities.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->
##### Unit tests

*  kubelet
    * Verify that the kubelet adds the `capabilities` field in `node.status` when the feature gate is enabled and omits it when disabled.
    * Verify that the Kubelet correctly applies validation rules (same rules as node labels) to capabilities and discards invalid key-value pairs.
    * Test the conditional logic
        * Verify `node.status.capabilities` accurately reflects the state of the `InPlacePodResizeExclusiveCpus` feature gate.
    * Test the deprecation logic
        * Add tests to verify that the Kubelet stops reporting a capability after its corresponding feature has been GA for longer than the supported version skew.
* kube-apiserver
    * Verify the capabilities field in `node.status` is correctly served (e.g., on GET, LIST) when the feature gate is enabled and omitted when the feature gate is disabled.
* Shared Capability Matching Library
    * Verify the library accurately infers a pod's capability requirements from its specification.
    * Verify the library correctly matches a pod's requirements against a node's capabilities.
    * Verify the library correctly bypasses capability checks for features that are stable (GA + version skew).
* kube-scheduler (`NodeCapabilityFilter` plugin):
    * Verify the plugin correctly calls the shared library to infer requirements in `PreFilter` stage and match nodes based on pre-computed requirements in `Filter` stage.
* Admission Controller (`NodeCapabilityValidator`):
    * Verify that for a Pod update request (e.g., resize), the validator correctly fetches the capabilities of the node specified in `pod.spec.nodeName`.
    * Verify the plugin correctly uses the shared library to match pod requirements against node capabilities, and correctly admits or rejects the request based on the library's result.

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

- `<package>`: `<date>` - `<test coverage>`
-->

##### Integration tests

*   kube-scheduler (for the `NodeCapabilityFilter` plugin):
    *  Integration tests would be introduced in [filters_tests.go](https://github.com/kubernetes/kubernetes/blob/master/test/integration/scheduler/filters/filters_test.go).
    *  Performance tests would be added in [scheduler_perf](https://github.com/kubernetes/kubernetes/tree/master/test/integration/scheduler_perf).

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

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)
-->

##### e2e tests

*   In a cluster with `[InPlacePodResizeExclusiveCpus](http://kubernetes.io/feature/inPlacePodResizeExclusiveCpus)` disabled and static CPU policy enabled, the admission controller should reject CPU resize requests for guaranteed QOS pods.

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

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)
-->

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

**Alpha:**
*   The `NodeCapabilities` feature gate is implemented and disabled by default.
*   The Kubelet correctly discovers and reports capabilities to the `node.status.capabilities` map when the feature gate is enabled.
*   A shared library is implemented with the initial logic for capability inference and matching.
*   The API server correctly serves and validates the `capabilities` field when the feature gate is enabled.
*   The `kube-scheduler` and `admission-controller` plugins are introduced and integrated with the shared library.
*   All unit tests outlined in the Test Plan are implemented and verified.

**Beta:**
*   Revisit the [Explicit Capability Request](#explicit-capability-request) as a beta graduation criteria.

### Upgrade / Downgrade Strategy

#### Upgrade

Users can enable the Node Capabilities feature gate after upgrading to a Kubernetes version that supports it. For the feature to be fully effective, both the control plane and Kubelets must be upgraded to this version. Existing workload should not be affected after the upgrade. The [Version Skew Strategy](#version-skew-strategy) section details behavior when only the control plane is upgraded while nodes are still running older Kubelet versions without the feature.

#### Downgrade

Downgrading both kubelet and control plane components to a version without the Node Capabilities feature means subsequent scheduling decisions and API request validations will no longer utilize node capabilities. The [Version Skew Strategy](#version-skew-strategy) section details behavior when only the kubelet is downgraded to a version lacking the feature.

### Version Skew Strategy

This section describes the behavior when the `NodeCapabilities` feature gate is enabled on the control plane (e.g., v1.X) but not on an older Kubelet (e.g., v1.X-1).
*  If the control plane is upgraded and begins processing requests that use a new feature, it will correctly identify older nodes as incompatible. For scheduling, the scheduler will filter these nodes, causing pods with new requirements to remain Pending until a compatible node is available. Similarly, for API validation, an operation will be rejected if the target pod resides on an older node that does not report the necessary capability.
*  This strict filtering is reliable because the NodeCapabilities framework is scoped to new features only. This prevents ambiguous situations where a feature might be present on a node but is not being reported because the node is too old. The absence of a capability is a definitive signal for the absence of a feature.

### Future Considerations 

#### Explicit Capability Request 

For the Alpha implementation, a pod's capability requirements are inferred by the shared library based on its specification. While this centralizes the logic, this implicit model still requires a code change in the shared library for each new capability that is introduced. A suggestion from the SIG Scheduling meeting was to explore a mechanism to keep the scheduler logic generic and avoid having case-specific inference logic.

#### Cluster Autoscaler Integration

For the Node Capabilities feature to be fully effective, it must be integrated with the Cluster Autoscaler (CA). The CA makes scale-up decisions based on `NodeGroup` templates, which are abstract representations of a node. Since node capabilities are reported by a running Kubelet, they are not present on these templates. This creates an information gap: CA cannot know what capabilities a new node will have, making it unable to scale up a node pool to satisfy a capability-specific pending pod.

* If a node group has active nodes, the CA can determine if a new node would be a valid match by inspecting a running node and evaluating its `node.status.capabilities` with the `NodeCapabilityFilter` plugin in the scheduling simulator. However, this approach fails in scale-from-zero scenarios.
* A long-term solution to solve the scale-from-zero problem requires that capability information be made available as part of the node group's template providing an authoritative signal to the CA.

This problem is similar to the one faced by Dynamic Resource Allocation (DRA), where the CA also does not work well if a pod is pending because it requires a specific `ResourceClaim`. This problem is being tracked in [kubernetes/autoscaler#7799](https://github.com/kubernetes/autoscaler/issues/7799). In both the DRA and Node Capabilities use cases, critical scheduling information is determined after a node is created, making it invisible to the CA's template-based simulation.

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

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: NodeCapabilities
  - Components depending on the feature gate: kubelet, kube-scheduler, kube-apiserver

###### Does enabling the feature change any default behavior?

Yes. Using new capabilities in the control plane can prevent pod scheduling or validation when the nodes cannot satisfy the pod requirements. 

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the feature can be disabled by setting the `NodeCapabilities` feature gate to false.
* If disabled on the nodes: The Kubelets will stop reporting all capabilities. As a result, any pod that requires a capability managed by this framework will become unschedulable, as no nodes will appear to satisfy its requirements. The nodes will remain available for all other workloads.
* If disabled on the control plane: The scheduler and admission controller plugins will bypass the NodeCapabilities based check and revert to the legacy scheduling behavior where node capabilities are not considered. 


###### What happens if we reenable the feature if it was previously rolled back?

When the `NodeCapabilities` feature gate is enabled on the control plane, pods requiring a specific capability will remain in the Pending state until the feature gate is enabled on the nodes. As the feature is subsequently re-enabled on nodes and they begin reporting capabilities in `node.status`, the scheduler will identify these nodes as eligible and schedule the workloads.

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

Yes, feature enablement/disablement tests will be added along with alpha implementation.

* Kubelet: When the feature gate is toggled, verify that the kubelet correctly populates or clears `node.status.capabilities`.
* Kube-scheduler & Admission Controller:
    * When the feature gate is toggled from on to off, verify that the capabilities based filtering and validation is bypassed. This will be tested by confirming that a pod requiring a specific capability is not filtered from an incompatible node.
    * When the feature gate is toggled from off to on, verify that the capabilities based filtering and validation is active and is correctly applied to nodes.

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

No. Capabilities are published as part of the existing NodeStatus updates.

###### Will enabling / using this feature result in introducing new API types?

No. This feature will not introduce new top-level API types. Instead, it will modify the existing Node API type by adding a new field node.status.capabilities, to convey capability information.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes. The size of the `Node` object is expected to increase as more capabilities are introduced. The number of capabilities exported will be limited by strategies such as:
1. Automatically handling feature graduation, which includes ceasing to export a capability once it matures or is no longer needed.
2. Exporting only configurations that are relevant to the control plane.


###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Yes, a marginal increase is possible for pod scheduling operations. This potential increase is because kube-scheduler will need to extract capability requirements specified in the Pod Spec and match them against the capability information reported on Node objects. However, this additional processing overhead is expected to be comparable to that of existing scheduling predicates like taint/toleration or Node Label/Selector matching.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

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

- 2025-05-09: [Proposal draft](https://docs.google.com/document/d/1vSDlAA3o0riVq0EcmGBOYUJUVF4tN2Ib7VJg3o1LvBw/edit?tab=t.0) and discussion.  
- 2025-05-13: Initial discussion in SIG Node meeting.
- 2025-06-12: KEP discussion in SIG Scheduling meeting.
- 2025-06-26: KEP discussion in SIG Architecture meeting.

## Drawbacks

1. **Difficult Ad-Hoc Overrides:** Labels/Taints/Tolerations can easily be added, removed, or modified to quickly influence scheduling. The fact that capabilities are reported by the Kubelet and auto-detected from the pod spec makes them unsuitable for ad-hoc overrides. 
2. **Greater Scheduler Complexity:** The scheduler plugins needed to interpret and match these capabilities against pod requirements (which may be inferred) could be more complex than existing label or taint matching.
3. Capabilities can make it easier to support a diverse set of runtime features and could lead to runtimes supporting an arbitrary subset of k8s features. This might lead to more heterogeneity in Node configurations which is harder to support.
4. This would make the `NodeStatus` object larger and less focused on just the operational status.
    *   To ensure the number of capabilities remains manageable, the design requires that every capability be directly actionable by a control plane component.
5. Updating static capabilities frequently with `NodeStatus` is an inefficient use of network resources.

## Alternatives

### Using a Meta-Capability as an Opt-In Signal

An alternative design considered using a meta-capability, `lifecycle.kubernetes.io/node-capabilities: "true"`, as an explicit opt-in signal from each node. In this model, control plane components would first check for this signal before applying capability-based filtering. If the signal was absent, the filtering logic would be bypassed, making the node eligible for any workload, even one requiring a capability the node did not have. This approach was intended to safely ignore older nodes or nodes where the feature was rolled back, allowing them to revert to legacy behavior.

**Drawbacks:** 

The primary issue is that the "bypass" logic makes the framework unreliable for managing version skew for any new Alpha or Beta features. If a non-participating node is considered eligible by the filter, a pod requiring a new capability could be scheduled onto that older, incompatible node, leading to the exact runtime failures this KEP aims to prevent. Consequently, any new feature could not safely depend on this framework until the NodeCapabilities feature itself was GA and universally enabled across the cluster.

### Using Node Labels and Node Affinity with SemVer comparison

This approach leverages the existing node affinity mechanism to control pod placement based on node features. Node labels can be introduced by cluster admin, Kubelet (well-known labels) or tools like Node Feature Discovery (NFD), which runs on each node, discovers its hardware and software characteristics, and automatically applies them as labels. Specifically for kubelet features, we could have Kubelet version as a node label. A user, cluster administrator, or admission webhook would introduce a nodeAffinity rule in the Pod specification based on the features it needs to target nodes with the appropriate kubelet versions. This also requires the node affinity kube-scheduler plugin to be enhanced to support SemVer comparison. 

**Pros**

* No core API Change. Leverages an existing and well-established Kubernetes mechanism for attaching metadata.
* Supports node-restricted labels. For labels in restricted domains (e.g., kubernetes.io/),the admission controller prevents the Kubelet from modifying them.

**Cons** 

* Node labels often become de facto APIs for controllers and other components, making them difficult to change or deprecate once related features reach GA.
* Operational overhead: In order to create correct affinity rules, the user or cluster administrator should have a detailed understanding of features needed by the workload and what kubelet versions have those features enabled.
* A node's Kubelet version indicates if the feature is present, but there is no way to know if the feature is actually enabled via a feature gate on the node. This would make the signal unreliable for features under active development (non-default).
* This approach would not work if there is a dependency on kernel or runtime configurations.

### Introducing a `NodeCapabilities` API

**Pros**

* Keeps node capabilities information separate from the operational status in `NodeStatus.`
* Updates can be infrequent and only when the capability changes.
* Allows for evolving the features API independently. Easier to extend with new feature types and fields.

**Cons** 
* Increased API server load - kube-scheduler should start watching the new core API object.
* Additional etcd load due to creating, updating, and reading these new objects.
* Increased scheduling latency as the kube-scheduler now needs to reconcile additional objects to make scheduling decisions.

### Alternative Naming Conventions

#### Alternative Names for the Capabilities Field

The generic field name `capabilities` was intentionally chosen for the field within `node.status`. The primary reason is to allow for future extension. While the initial use case is for reporting lifecycle-tied features to manage version skew, a generic name provides the flexibility to report other types of capabilities in the future without requiring a new API field.

Alternatives like `featureGates` or `lifecycleFeatures` were considered but were ultimately deemed too restrictive. The desired lifecycle semantics for the current use case are instead enforced by the mandatory `lifecycle.kubernetes.io/` prefix on the capability keys, keeping the top-level field name itself generic and extensible.

#### Alternative Naming Conventions for Capability Keys

Several naming conventions for the capability keys were considered to signal their temporary nature.

1.  Use-Case Focused Prefixes (e.g., `skew.kubernetes.io` or `compatibility.kubernetes.io` or `lifecycle.kubernetes.io`)  - They directly address the focus on the "version skew" problem. We can have more prefix type for future use cases.
2.  Nature Focused Prefixes (e.g., `transient.kubernetes.io` or `provisional.kubernetes.io`) - They communicate the impermanent nature of the capabilities. The key concern here is that these terms could be ambiguous and might imply a temporary state that will eventually be promoted to a permanent one (e.g., a "provisional" feature becoming a "full" feature).

`lifecycle.kubernetes.io/` was preferred because it best captures the connection to the formal Kubernetes feature graduation process, which indicates a removal phase after the feature becomes stable.
