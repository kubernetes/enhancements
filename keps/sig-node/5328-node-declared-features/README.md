# KEP-5328: Node Declared Features


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
    - [Declared Feature Semantics](#declared-feature-semantics)
  - [Kubelet Changes](#kubelet-changes)
  - [Shared Feature Matching Library](#shared-feature-matching-library)
    - [Multi Cluster Support](#multi-cluster-support)
  - [kube-scheduler Changes](#kube-scheduler-changes)
    - [Plugin Implementation](#plugin-implementation)
    - [Performance Considerations](#performance-considerations)
  - [Admission Controller Changes](#admission-controller-changes)
  - [Cluster Autoscaler Integration](#cluster-autoscaler-integration)
  - [Declared Feature Lifecycle](#declared-feature-lifecycle)
  - [Walkthrough](#walkthrough)
  - [Declared Feature Changes on Existing Nodes](#declared-feature-changes-on-existing-nodes)
  - [Integration with Existing Mechanisms](#integration-with-existing-mechanisms)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade/Downgrade Strategy](#upgradedowngrade-strategy)
    - [Upgrade](#upgrade)
    - [Downgrade](#downgrade)
  - [Version Skew Strategy](#version-skew-strategy)
  - [Future Considerations](#future-considerations)
    - [Explicit Declared Feature Request](#explicit-declared-feature-request)
    - [Cluster Autoscaler Scale-From-Zero Integration](#cluster-autoscaler-scale-from-zero-integration)
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
  - [Using a Meta Opt-In Signal](#using-a-meta-opt-in-signal)
  - [Automatic Declared Feature Deprecation](#automatic-declared-feature-deprecation)
  - [Using Node Labels and Node Affinity with SemVer comparison](#using-node-labels-and-node-affinity-with-semver-comparison)
  - [Using a <code>map[string]string</code> for <code>DeclaredFeatures</code>](#using-a-mapstringstring-for-declaredfeatures)
  - [Introducing a <code>NodeCapabilities</code> API](#introducing-a-nodecapabilities-api)
  - [Alternative Naming Conventions](#alternative-naming-conventions)
    - [Alternative Names for the field in <code>Node.Status</code>](#alternative-names-for-the-field-in-nodestatus)
    - [Alternative Naming Conventions for Declared Feature Keys](#alternative-naming-conventions-for-declared-feature-keys)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
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

This KEP proposes a **Node Declared Features** framework for nodes to declare the availability of specific, feature-gated Kubernetes features. This would then be used by the control plane to make scheduling and admission decisions, primarily to manage version skew. For scheduling, the kube-scheduler would utilize these declared features to ensure pods are only placed on nodes that possess the necessary features to run them successfully. For API request validation, admission controllers would prevent operations on nodes that lack the required feature support. The intent is to streamline cluster operations by reducing the reliance on manual configurations like taints, tolerations, and complex node labeling schemes.

## Motivation

The primary motivation for this KEP is to solve scheduling and validation problems that arise from version skew. When a new feature is enabled on the control plane, a mismatch often occurs because nodes are upgraded gradually or are simply running older Kubelet versions, as is permitted by the Kubernetes version skew policy. This creates a window where the scheduler might place a pod requiring a new feature onto a node that does not support it.

By making the scheduler aware of the features on the node, this proposal ensures that incompatibilities are handled correctly and proactively instead of failing later with a runtime error or a Kubelet admission failure on the node. A pod without a matching node remains `Pending` with an event that details the unmet feature requirement, providing actionable user feedback ([slack discussion](https://kubernetes.slack.com/archives/C5P3FE08M/p1741867194258139)).

### Goals

1. Define a standard mechanism for nodes to declare features that are tied to the lifecycle of new Kubernetes features to manage version skew.
2. Introduce a shared library to encapsulate the logic for inferring a pod's feature requirements and matching them against a node's declared features, ensuring consistency between control plane components that depend on this mechanism.
3. Enhance the kube-scheduler to filter nodes based on the pod's requirements.
4. Enable API admission controllers to validate requests for operations against a node's actual feature support.
5. Enable Kubelet admission plugin to check if the Pod is compatible with the node's features.
6. Enable Cluster Autoscaler to consider declared features while scaling up existing node groups with active nodes.

### Non-Goals

1. Replace Taints/Tolerations or Node Labels/Selectors/Affinity.
2. Serve as a reporting mechanism for permanent static node attributes (like architecture, or specific hardware).
3. The feature declaration and matching mechanism is designed to support only new features introduced after this framework is in place. It is not applicable to Kubernetes features that are already implemented.
4. Cluster Autoscaler integration for scale-from-zero scenarios is a non-goal for this KEP and needs to be solved as a part of larger autoscaling strategy. This is discussed in [Future Considerations](#cluster-autoscaler-scale-from-zero-integration) section.


## Existing Mechanisms and Limitations

The Kubernetes scheduler currently uses two primary mechanisms to control pod placement onto specific nodes:

1. Taints and Tolerations
    Primarily used to **restrict** which pods can schedule onto specific nodes. Commonly used to manage specialized hardware resources.

  **Standard Usage Pattern:**
    *   Cluster Administrators apply taints to nodes equipped with special capabilities; cloud providers may also automate this tainting for certain NodePool configurations.
    *   Workload authors add corresponding tolerations in their Pod specifications for workloads to be able to run on these nodes. Alternatively, cluster administrator or cloud provider could also inject tolerations through admission webhooks.

2. Node Labels and Node Selectors/Affinity

    Primarily used to **attract** specific nodes for pods based on the node's characteristics. By applying specific labels to nodes (reflecting Kubelet features, OS version, etc.), we can enable pods to use selectors or affinity to ensure they run on specific nodes.

  **Standard Usage Pattern:**
    *   Cluster Administrators apply specific Labels to nodes to indicate the presence of certain features. This involves applying descriptive labels (e.g., `kubelet.config.k8s.io/some-alpha-feature=enabled`, `node.kubernetes.io/gvisor-enabled=true`)
    *   Optionally, for well-defined features, it may need to create other resources (e.g., [RuntimeClass](https://kubernetes.io/docs/concepts/containers/runtime-class/)) that bundle these feature requirements with a node selector targeting the corresponding node labels.
    *   Developers specify their workload's dependency on these features in the PodSpec either directly (spec.nodeSelector) or other abstractions enabling the scheduler to match them to capable nodes.

**Drawbacks**:

1. Operational Overhead: Cluster administrators have to add the necessary taints/labels to nodes as indirect signals of features and resources, and workloads should use corresponding tolerations/selectors. This needs to be done manually or automations built (webhooks, controllers) to handle standard usage patterns.
2. Scheduling constraints are encoded indirectly rather than being implicitly understood by kube-scheduler.
3. Incorrect configurations can lead to scheduling failures or suboptimal placements.
4. Reduced workload portability. The taints, tolerations, and labels are defined by specific administrators or distribution providers, and may vary across organizations or even clusters. This means that manifests on one cluster may behave differently on another cluster.

## Proposal

This proposal introduces a new field `DeclaredFeatures` to `Node.Status` to expose information which the kube-scheduler and/or the admission controller would use to make more informed decisions. The Kubelet is primarily responsible for discovering, consolidating, and declaring features to the API server.

**Node Declared Features Requirements:**

1. Every feature added in `node.status.declaredFeatures` is temporary and must be associated with a Kubernetes feature graduating through the Alpha/Beta/GA process. This ensures declared features are not used as permanent node attributes and are removed as part of the standard post-GA feature cleanup process.
2. The NodeDeclaredFeatures framework must be used for new features introduced after the framework. Onboarding existing features must be avoided as it would create ambiguity; the control plane would be unable to differentiate between a node that has the feature but is not declaring it, and a node that genuinely lacks the feature.
3. Must be derived from node's static configuration and determined at startup. Kubelet must determine all declared features during its bootstrap sequence, before its admission handlers are active. Reporting new or changed features requires a Kubelet restart to take effect.
4. Must be actionable by the control plane. A declared feature is only relevant if it can be used by a control plane component (like kube-scheduler or an admission controller) to make a decision, such as filtering a node or validating an API request.

## User Stories

### Story 1: Feature Rollout Challenges with Version Skew

To maintain cluster stability and enable safer rollouts, cluster administrators perform gradual upgrades. This inherently creates a mixed-version Kubernetes cluster, where nodes can be on different versions than the control plane. This environment introduces significant challenges for both pod scheduling and API request validation. The primary concerns in such clusters are:

1. How do we prevent new pods that might use newer features from being scheduled on older, incompatible nodes?
  *   Example: [Pod Level Resources](https://github.com/kubernetes/enhancements/blob/63d4f6f2aa0e2eb0b83067b067c4949643b1b24c/keps/sig-node/2837-pod-level-resource-spec/README.md?plain=1#L4)
    *   The existing beta feature lacks explicit version-skew management, requiring it to be enabled on all components (scheduler, Kubelet, API server) to function correctly. If enabled only on the control plane, a pod can be scheduled on an incompatible node where the Kubelet will reject it during admission. This rejection puts the pod into a `Failed` state, preventing it from being retried on other newer (compatible) nodes in the cluster.
    *   For new enhancements, like adding support for CPU alignment (static CPU policy support), there is no way to make sure a guaranteed QOS pod using pod level resources lands on a node supporting this new feature.

2. How do we block API calls that target pods on nodes that don't support newer features?
  *   Example: [In-Place Pod Resizing](https://kubernetes.io/docs/tasks/configure-pod-container/resize-container-resources/)
    *   For an existing beta feature, there currently exists a [workaround](https://github.com/kubernetes/kubernetes/blob/23258f104d74c6f27fd4db94940d745d9d463a8f/pkg/apis/core/validation/validation.go#L5796) to handle this version skew by looking for alternate signals from the pod spec. However such workarounds may be complex and not always feasible for new features.
    *   For a new feature enhancement like [In-Place Pod Resize for Guaranteed QoS pods](https://github.com/kubernetes/enhancements/issues/5294), an API request to modify a running pod must be validated. The operation should be rejected if the pod resides on an older node that does not support the feature.

The Node Declared Features framework addresses both issues through a unified mechanism without requiring any intervention (like adding Node Labels, selectors etc.). The scheduler uses the declared feature information to correctly filter nodes for new pods, while the admission controller validates operations against the declared features of a pod's current node. This ensures that feature incompatibilities are handled proactively at the control plane.

## Design Details

### API Changes

Add a `DeclaredFeatures` field as type `[]string` to the `Node.Status` structure.

```
type NodeStatus struct {
    // ... existing fields
    // DeclaredFeatures provides a list of features declared by the node.
    // The list is sorted alphabetically and contains no duplicate entries.
    // +optional
    // +listType=atomic
    // +featureGate=NodeDeclaredFeatures
    DeclaredFeatures []string `json:"declaredFeatures,omitempty"`
}


// Node object remains unchanged in spec, only status is modified.
type Node struct {
    ...
    Status NodeStatus `json:"status,omitempty"`
}

```

**Note:**

*   Any new feature being introduced in `node.status.declaredFeatures` is considered a formal API change and must go through the API review process. This governance will be enforced by protecting the list of declared features with `api-approvers` in the OWNERS file.
*   We currently have [Node Features](https://github.com/kubernetes/api/blob/e8d4d542f6a9a16a694bfc8e3b8cd1557eecfc9d/core/v1/types.go#L6279) and [Node Runtime Features](https://github.com/kubernetes/api/blob/e8d4d542f6a9a16a694bfc8e3b8cd1557eecfc9d/core/v1/types.go#L6251) which publish some runtime features through Node Status. They are too narrowly scoped and currently not used for scheduling pods.


#### Declared Feature Semantics

1.  Combine Interdependent Settings
    *   If multiple settings are required to enable a feature, they must be collapsed into a single, logical declared feature.
    *   This simplifies decision-making for control plane components like the scheduler, which should not need to understand multiple interdependent settings.

2.  Presence of a Declared Feature
    *   A feature should only be present in `node.status.declaredFeatures` if it is enabled and functional.
    *   If a feature is disabled or unsupported, its corresponding identifier must not exist in `node.status.declaredFeatures`.
    *   This approach ensures consistency and simplifies logic for the control plane, treating nodes that don't know about a feature the same as those that have it disabled.

3.  Validation Rules
    *   Strings within the `node.status.declaredFeatures` slice should follow a CamelCase convention and is case-sensitive.
    *   For features with sub-components, a `FeatureName/SubFeature` format can be used.
    *   Each identifier can have a maximum length of 253 characters (DNS1123SubdomainMaxLength).
    *   The list must be sorted alphabetically and must not contain duplicate entries.
    *   Kubelet validates each identifier and will discard any invalid or duplicate entries, logging an error. These validation rules will also be enforced by [unit tests](#unit-tests).


**Example:**

```
declaredFeatures:
   - GuaranteedQoSPodCPUResize
```

### Kubelet Changes

Kubelet has the primary responsibility of discovering the enabled features relevant for control plane during its bootstrap and updating `node.status.declaredFeatures`. It determines its full, static set of declared features **once** upon startup based on its configuration. This feature set is then populated into the `node.status.declaredFeatures` field during the Kubelet's periodic node status update cycle. The Kubelet sends a patch to the API server only when there is a change in the overall `NodeStatus` object or when the status reporting period expires. The Kubelet is the authoritative source of truth for its own declared features. If an external controller or user were to modify the `node.status.declaredFeatures` field on the Node object, the Kubelet's next periodic status update would automatically overwrite and revert those changes.

In addition to reporting, Kubelet will also need to validate the pod. Before admitting a pod to be run on the node, the Kubelet's pod admission handlers check the pod's feature requirements (as inferred from its spec). It will validate these requirements against its own in-memory list of declared features. If a feature required by the pod is not present, the Kubelet will reject the pod, transition it to a `Failed` state, and post an appropriate event to the API server detailing the missing feature requirement. This secondary validation (along with kube-scheduler filtering) is necessary to handle node restart scenarios where a feature that was enabled during pod scheduling is no longer enabled (feature gate flip with node restart).

As a part of the feature's graduation to GA and eventual removal of the feature gate, the Kubelet must stop declaring the corresponding feature in `node.status.declaredFeatures`. This ensures that obsolete information is removed from the `node.status` object as part of the standard feature cleanup process.

### Shared Feature Matching Library

To avoid code duplication and ensure that all components make decisions based on the same logic, a new shared library will be introduced in the `k8s.io/component-helpers/nodedeclaredfeatures` staging repository. This library encapsulates all the logic for inferring a pod's feature requirements and matching them against a node's declared features.

**Feature Registration**

*   Each new feature that needs to be declared in `node.status.declaredFeatures` is registered along with its specific inference logic by implementing a `CreateInferrer` or `UpdateInferrer` interface. This registration will also include `MinVersion` and `MaxVersion` fields which define the inclusive bounds of Kubernetes versions for which the feature is an active scheduling constraint.
    *   **`MinVersion`** is the first Kubernetes version in which the feature is introduced.
    *   **`MaxVersion`** should be the version at which the feature is considered universally available in the cluster (e.g., `GA_VERSION + SKEW`). This can be defined when the feature goes GA.
*   This feature, along with its inferrer, is then registered with the central registry. This makes the framework pluggable, allowing new features to be added without modifying the core components.

**Core Functionality**

The `NodeDeclaredFeatureHelper` will expose functions that logically separate pod inspection and per-node matching.
1. Inferring Requirements: The library will provide functions  (`InferPodCreateRequirements`, `InferPodUpdateRequirements`) to analyze the PodSpec and return a list of its feature requirements. The infer functions do not contain any feature specific logic themselves, instead they iterate over all features registered in the central registry and execute the corresponding `Infer` method for each one. This design is key to the framework's extensibility. These functions validate the pod's inferred requirements against the `targetVersion`. Passing the target version in the inference functions is necessary to support [multi-cluster support](#multi-cluster-support)
   *  If the pod requires a feature where `targetVersion < MinVersion`, the function returns an `IncompatibleFeatureError`. 
   *  If the pod requires a feature where `targetVersion > MaxVersion`, the specific feature requirement is silently ignored, as the feature is assumed to be universally available in the cluster and is no longer a scheduling constraint.
2. Matching Requirements: A second function (`MatchNode`) will take the pre-computed requirements and check them against a specific node.

```go
// PodInfo is an extensible data structure that wraps the pod object
// and can be expanded in the future to include ancillary resources
// like ResourceClaims or PVCs.
type PodInfo struct {
	Pod *v1.Pod
	// Add other ancillary resources here in the future as needed.
	// Example: ResourceClaims []*v1.ResourceClaim
}

// CreateInferrer is an interface for inferring feature requirements from a new pod.
// It returns the list of feature requirements, or an empty list if none are needed.
type CreateInferrer interface {
  Infer(ctx context.Context, podInfo *PodInfo, targetVersion string) []string
}

// UpdateInferrer is an interface for inferring feature requirements from a pod update.
// It returns the list of feature requirements, or an empty list if none are needed.
type UpdateInferrer interface {
  Infer(ctx context.Context, oldPodInfo, newPodInfo *PodInfo, targetVersion string) []string
}

// InferPodCreateRequirements inspects a new pod and returns a list of feature requirements required by the pod
// for a specific target Kubernetes version.
// If the pod requires a feature where `targetVersion < MinVersion`, the function returns an `IncompatibleFeatureError`.
// If the pod requires a feature where `targetVersion > MaxVersion`, the feature is not added to the list as it's assumed to be universally available in the cluster.
func (h *NodeDeclaredFeatureHelper) InferPodCreateRequirements(ctx context.Context, podInfo *PodInfo, targetVersion string) ([]string, error)

// InferPodUpdateRequirements inspects the change between an old and new pod spec
// and returns a list of feature requirements required by the pod update operation
// for a specific target Kubernetes version.
// If the pod requires a feature where `targetVersion < MinVersion`, the function returns an `IncompatibleFeatureError`.
// If the pod requires a feature where `targetVersion > MaxVersion`, the feature is not added to the list as it's assumed to be universally available in the cluster.
func (h *NodeDeclaredFeatureHelper) InferPodUpdateRequirements(ctx context.Context, oldPodInfo, newPodInfo *PodInfo, targetVersion string) ([]string, error)

// MatchResult encapsulates the result of a feature match check.
type MatchResult struct {
	// IsMatch is true if the node satisfies all feature requirements.
	IsMatch bool
	// UnsatisfiedFeatureRequirements lists the specific feature requirements that were not met.
	// This field is only populated if IsMatch is false.
	UnsatisfiedFeatureRequirements []string
}

// MatchNode checks if the node's declared features satisfy the pre-computed feature requirements for a pod.
// It returns a MatchResult with the outcome of the match.
func (h *NodeDeclaredFeatureHelper) MatchNode(ctx context.Context, reqs []string, node *v1.Node) (*MatchResult, error)
```

This design provides a clear and extensible interface for consumers:
*   kube-scheduler would call `InferPodCreateRequirements` once during the `PreFilter` and `Enqueue Extension` stages and call `MatchNode` for each node during the Filter stage.
*   Admission controller when validating a pod update, would call `InferPodUpdateRequirements()` with the current and the new PodSpec to infer the requirement and then call `MatchNode()` with the node object from `pod.spec.nodeName`.

#### Multi Cluster Support

 To support external controllers that interact with multiple Kubernetes clusters of different versions, the shared library's inference logic must be version-aware. The set of available declared features and the rules for inferring them from a `PodSpec` can change between versions. The library must provide an unambiguous signal to these controllers.

### kube-scheduler Changes

We propose a new scheduler plugin named `NodeDeclaredFeatures` to infer a pod's feature requirements and match them against a node's declared features.

#### Plugin Implementation

`NodeDeclaredFeatures` scheduler plugin would be enabled if the feature gate is enabled in the kube-scheduler and would implement the below extension points in the [scheduling framework](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/).

1. PreFilter
    *   In this stage, the plugin will pass the pod spec to the shared library to compute a set of its feature requirements. This result will be cached in the scheduler's prefilter state.
2. Filter:
    *   For each node being evaluated, the plugin will retrieve the cached requirements and pass them along with the Node object to a matching function in the shared library.
    *   The library itself is responsible for handling all complex logic. The scheduler plugin simply acts on the boolean result returned by the library to either filter the node or pass it.
    *   The library also returns a list of the specific feature requirements that were not satisfied which can be used to generate a `UnschedulableAndUnresolvable` status with a clear, informative message. The scheduler framework then records this message in the Pod's status and events, making the reason for the scheduling failure directly visible to the user.
3. Enqueue Extension:
    *   To make the scheduler responsive to cluster changes, the plugin implements queueing hints to optimize re-queueing.
    *   **Node Events:** The scheduler plugin registers for Node `Add` and `Update` events. In the case of updates, a new narrowly-scoped `UpdateNodeDeclaredFeature` action type will be introduced to trigger the hint function when `node.status.declaredFeatures` is modified. A queue hint is generated if a new node is added or an existing node's declared features are updated in a way it satisfies the pod's feature requirements.
    *   **Pod Events:** The scheduler plugin registers for Pod `Update` event. A new narrowly-scoped `UpdatePodFeatureRequirement` action type will be introduced and a queue hint is generated if a pending pod is updated in a way that its feature requirements have changed.

This design ensures that the `NodeDeclaredFeatures` scheduler plugin remains generic and does not require modification when new features are added. All feature-specific inference and matching logic is encapsulated within the shared library.

#### Performance Considerations

Introducing this plugin adds a new step to the scheduling cycle, which may have an impact on scheduling throughput. However, this trade-off is acceptable because it prevents the greater inefficiency of scheduling a pod onto a node where it cannot run. If the performance impact is determined to be non-negligible, bitmap-based approach can be considered for faster matching.


### Admission Controller Changes

To enable the validation of API requests against Node Declared Features, this KEP proposes the introduction of a new admission controller plugin, `NodeDeclaredFeatureValidator`, that will be enabled when the `NodeDeclaredFeatures` feature gate is active. For its initial scope, this admission controller will focus on validating Pod `UPDATE` requests to prevent modifications that are incompatible with the node a pod is running on. This admission controller cross-references the objects, i.e., looking up the Node object that a Pod is bound to (`spec.nodeName`) and runs the validation checks.

The admission controller workflow will be as follows:
*   Inspect an incoming pod `UPDATE` request and verifies that the pod is already bound to a node by checking that `pod.spec.nodeName` is set. If not, it takes no action.
*   Retrieves the Node object corresponding to `pod.spec.nodeName`.
*   Call the shared library to infer the feature requirements based on the changes between the old and new `PodSpec` along with the kubelet version `node.status.nodeInfo.kubeletVersion`.  If the function returns an `IncompatibleFeatureError`, the admission controller rejects the request.
*   Call the shared library's `MatchNode` function with the inferred feature requirements and the `node.status.declaredFeatures`. If the check fails, the admission controller rejects the request.

### Cluster Autoscaler Integration

For the Node Declared Features feature to be fully effective, it must be integrated with the Cluster Autoscaler (CA). The CA makes scaling decisions by simulating the scheduling of pending pods on template nodes representing what a new node from a node group would look like. The integration has two distinct scenarios depending on how this template is created:

1.  **Scaling a node group with existing nodes:** A node group is an abstraction for a set of nodes with the same configuration. If a node group has active nodes, the CA can create a template for a new node based on the configuration of an existing node. This template will include the `node.status.declaredFeatures` reported by the running node. The CA's scheduling simulator can then use this information, in conjunction with a simulated `NodeDeclaredFeatures` scheduler plugin, to correctly predict that a new node will satisfy a pending pod's feature requirements. This approach relies on the assumption that all nodes within a node group are homogeneous and will declare the same set of features. This scenario is in scope for the Alpha release.

2.  **Scaling a node group from zero nodes (scale-from-zero):** This approach fails in scale-from-zero scenarios. When a node group has no active nodes, the CA must create the node template from the cloud provider's configuration (e.g., GCE Instance Template). Since Node Declared Features is dynamically set by the Kubelet only after a node has been created, the CA's simulation cannot account for this during scale-from-zero nodes. Handling this is not in scope for Alpha or Beta. A long-term solution to solve the scale-from-zero problem is discussed in the [Future Considerations](#cluster-autoscaler-scale-from-zero-integration) section.

### Declared Feature Lifecycle

The lifecycle stages are as follows:

* Introduction (Alpha/Beta): A new string is introduced in `node.status.declaredFeatures` alongside a new feature. Kubelet begins reporting this on nodes where the feature is enabled and active. Control plane components, via the shared library, start using this declared feature to make decisions.
* Graduation (GA): When the feature graduates to GA, the Kubelet continues to declare the feature. This is necessary to manage version skew, allowing the control plane to correctly identify older nodes that do not yet have the GA feature.
* Deprecation and Cleanup (Post-GA): The feature declaration and inference logic is removed from the codebase and can coincide with the removal of feature gate itself.

### Walkthrough

This walkthrough demonstrates the end-to-end lifecycle of the [In-Place Pod Resize for Guaranteed QoS pods](https://github.com/kubernetes/enhancements/issues/5294) feature using the Node Declared Features framework.

**Phase 1: Feature Development**
  * Kubelet Changes
    *   A new feature key `GuaranteedQoSPodCPUResize` is defined and registered with the shared library along with a function to infer if the feature is required by the pod. 
    *   The Kubelet is updated to report `GuaranteedQoSPodCPUResize` in `node.status.declaredFeatures` if either of the following conditions are met:
        *   The `InPlacePodResizeExclusiveCPUs` feature gate is enabled AND the CPU Manager Policy is set to static.
        *   The CPU Manager Policy is set to none.
  *   No code changes are needed in the  `NodeDeclaredFeatureValidator` admission plugin itself. It calls the shared library with the pod spec and will use the inferrer provided along with the registration.

**Phase 2: Rollout**
  *   Cluster administrator enables the new feature `InPlacePodResizeExclusiveCPUs` and `static` CPU Manager Policy on a NodePool.
  *   A user requests an in-place CPU resize for a pod on an upgraded node.
  *   The `NodeDeclaredFeatureValidator` admission controller intercepts the update. It calls the shared library's `InferPodUpdateRequirements` function. The `InferPodUpdateRequirements` function will loop through all the registered features and call their corresponding infer functions. The infer function added for `GuaranteedQoSPodCPUResize` identifies the need for the `GuaranteedQoSPodCPUResize` feature requirement.
  *   The `NodeDeclaredFeatureValidator` admission controller calls `MatchNode` function with the inferred pod feature requirements and the Node object from `pod.spec.nodeName`.
  *   The request is admitted only if the node's `declaredFeatures` list **contains** `GuaranteedQoSPodCPUResize`.

**Phase 3: Post-GA Cleanup**
  *   After the feature graduates to GA and is eventually deprecated, the feature gate is removed.
  *   As part of the same code removal, the node declaration logic is removed from the Kubelet and the inference logic is removed from the shared library.

### Declared Feature Changes on Existing Nodes

Node declared features are checked by the scheduler during scheduling and then validated again by the Kubelet's admission handlers before a pod's containers can start. This validation is critical for handling cases where a node's declared features change after a pod has been scheduled.

If a Kubelet restarts with a different configuration (e.g., a feature gate is disabled), its declared features may change. Upon restart, the Kubelet re-evaluates all existing pods for admission and if a running pod requires a declared feature that the node no longer provides, the Kubelet's admission check will fail. Consequently, the pod will not be started and will be transitioned to a `Failed` phase. This ensures that a pod does not run on a node that cannot support its feature requirements. This behavior is consistent with the best practice of draining a node before making significant configuration changes, such as toggling feature gates.

### Integration with Existing Mechanisms

We should ideally have one signal to express scheduling intent, but during transition we might end up having multiple active mechanisms to achieve Pod-to-Node matching in kube-scheduler.

**Scenario 1**
If there is an existing mechanism (like node labels and selectors) and now we introduce a new node declared feature to manage feature availability on the node, the scheduling restrictions are additive. The `NodeDeclaredFeatures` filter does not override other filters; it works alongside them. A pod must satisfy the requirements of all active filters.

**Scenario 2**
If a pod could previously schedule on any node, the new `NodeDeclaredFeatures` kube-scheduler filter may now proactively filter out nodes that lack a required feature. This is the intended behavior of the feature. It ensures that pods are not scheduled on nodes where they would fail, which is analogous to how a missing label prevents a nodeSelector match.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

The example described in the [Example Walkthrough](#example-walkthrough) section can be used to demonstrate and test Node Declared Features.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->
##### Unit tests

*   kubelet
    *   Verify that the kubelet adds the `declaredFeatures` field in `node.status` when the feature gate is enabled and omits it when disabled.
    *   Verify that the Kubelet correctly applies validation rules to declared features.
    *   Verify that the Kubelet's pod admission handler correctly rejects a pod that requires a declared feature the node does not have.
    *   Test the conditional logic
        *   Verify `node.status.declaredFeatures` accurately reflects the state of the `InPlacePodResizeExclusiveCPUs` feature gate.
*   kube-apiserver
    *   Verify the declaredFeatures field in `node.status` is correctly served (e.g., on GET, LIST) when the feature gate is enabled and omitted when the feature gate is disabled.
* Shared Feature Matching Library
    *   Verify the library accurately infers a pod's feature requirements from its specification.
    *   Verify the library correctly matches a pod's requirements against a node's declared features.
    *   Verify the library returns an IncompatibleFeatureError if a pod requires a feature where targetVersion is less than the feature's registered MinVersion.
    *   Verify the library silently ignores a feature requirement if the targetVersion is greater than the feature's registered MaxVersion.
    *   Verify the library correctly infers requirements when the targetVersion is within the valid [MinVersion, MaxVersion] range.
* kube-scheduler (`NodeDeclaredFeatures` scheduler plugin):
    *   Verify the plugin correctly calls the shared library to infer requirements in `PreFilter` stage and match nodes based on pre-computed requirements in `Filter` stage.
* Admission Controller (`NodeDeclaredFeatureValidator`):
    *   Verify that for a Pod update request (e.g., resize), the validator correctly fetches the declared features of the node specified in `pod.spec.nodeName`.
    *   Verify the plugin correctly uses the shared library to match pod requirements against node declared features, and correctly admits or rejects the request based on the library's result.

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

*   kube-scheduler (for the `NodeDeclaredFeatures` scheduler plugin):
    *   Integration tests would be added in `test/integration/scheduler/filters`. They will verify the scheduler plugin's PreFilter and Filter by creating `v1.Node` objects, patching their `status.declaredFeatures` field, and scheduling `v1.Pod` objects whose specs imply a requirement for those declared features.
    *   Tests for queueing hints would be introduced in `test/integration/scheduler/queueing`. These tests will verify the conditional logic to requeue a pending pod if its feature requirements are satisfied after the node is added or an existing node's declared features are updated, or pod is updated in a way its required declared features change.
    *   Performance tests would be introduced in `test/integration/scheduler_perf` to measure the plugin's impact on scheduling throughput and latency.
*   Admission Controller (for the `NodeDeclaredFeatureValidator` admission controller):
    *   New tests in `test/integration/apiserver/admissionwebhook` will enable the `NodeDeclaredFeatureValidator` plugin and verify that Pod `UPDATE` operations are correctly admitted or rejected based on the declared features of the Node the Pod is bound to.

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
*   The `NodeDeclaredFeatures` feature gate is implemented and disabled by default.
*   The Kubelet correctly discovers and declares features to the `node.status.declaredFeatures` list when the feature gate is enabled.
*   A shared library is implemented with the initial logic for feature inference and matching.
*   The API server correctly serves and validates the `declaredFeatures` field when the feature gate is enabled.
*   The `kube-scheduler` and `admission-controller` plugins are introduced and integrated with the shared library.
*   At least one feature is integrated with and uses the Node Declared Features framework. 
    * [In-Place Pod Resize with Static CPU policy](https://github.com/kubernetes/enhancements/issues/5554) will be introduced along with the framework changes.
*   All unit and integration tests outlined in the Test Plan are implemented and verified.

**Beta:**
*   Cluster Autoscaler support for existing node pools with active nodes. 
*   Revisit the [Explicit Declared Feature Request](#explicit-declared-feature-request) as a beta graduation criteria.

### Upgrade/Downgrade Strategy

#### Upgrade

Users can enable the `NodeDeclaredFeatures` feature gate after upgrading to a Kubernetes version that supports it. For the feature to be fully effective, both the control plane and Kubelets must be upgraded to this version. Existing workloads should not be affected after the upgrade. The [Version Skew Strategy](#version-skew-strategy) section details behavior when only the control plane is upgraded while nodes are still running older Kubelet versions without the feature.

#### Downgrade

Downgrading both kubelet and control plane components to a version without the feature means subsequent scheduling decisions and API request validations will no longer utilize node declared features. The [Version Skew Strategy](#version-skew-strategy) section details behavior when only the kubelet is downgraded to a version lacking the feature.

### Version Skew Strategy

1. When the `NodeDeclaredFeatures` feature gate is enabled on the control plane (e.g., v1.X) but not on an older Kubelet (e.g., v1.X-1).
    *  If the control plane is upgraded and begins processing requests that use a new feature, it will correctly identify older nodes as incompatible. The scheduler will filter these nodes, causing pods with the feature requirement to remain Pending. Similarly, for API validation, an operation will be rejected if the target pod resides on an older node that lacks the necessary feature.
    *  This strict filtering is reliable because the NodeDeclaredFeatures framework is scoped to new features only. This prevents ambiguous situations where a feature might be present on a node but is not being reported because the node is too old. The absence of a declared feature is a definitive signal for the absence of a feature.

2. When the `NodeDeclaredFeatures` feature gate is disabled on the control plane but enabled on the Kubelet:
    *  The scheduler and admission controller will ignore any features declared by the node. This reverts to the behavior where scheduling decisions are made without considering node declared features. 

### Future Considerations

#### Explicit Declared Feature Request

For the Alpha implementation, a pod's feature requirements are inferred by the shared library based on its `PodSpec`. While this centralizes the logic, this implicit model requires a code change for each new feature that is introduced. To make the framework more scalable and generic, we should explore an explicit feature request mechanism. In this model, a pod would declare its requirements directly in its specification. This would simplify the control plane's role, particularly the scheduler, reducing its task to a straightforward comparison between the features a pod requests and those a node provides, eliminating the need for complex, case-by-case inference logic.

#### Cluster Autoscaler Scale-From-Zero Integration

The Cluster Autoscaler (CA) makes scale-up decisions based on `NodeGroup` templates, which are abstract representations of a node. Since node declared features are set by a running Kubelet, they are not present on these templates. This creates an information gap: CA cannot know what declared features a new node will have, making it unable to scale up a node pool to satisfy a feature-specific pending pod in scale-from-zero scenarios.

A long-term solution to solve the scale-from-zero problem requires that declared feature information be made available as part of the node group's template providing an authoritative signal to the CA. This problem is similar to the one faced by Dynamic Resource Allocation (DRA), where the CA also does not work well if a pod is pending because it requires nodes with specific ResourceSlices. This problem was discussed [here](https://github.com/kubernetes/enhancements/pull/5347#discussion_r2132302773) and is being tracked in [kubernetes/autoscaler#7799](https://github.com/kubernetes/autoscaler/issues/7799). In both the DRA and Node Declared Features use cases, critical scheduling information is determined after a node is created, making it invisible to the CA's template-based simulation. 

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
  - Feature gate name: NodeDeclaredFeatures
  - Components depending on the feature gate: kubelet, kube-scheduler, kube-apiserver

###### Does enabling the feature change any default behavior?

Yes. While enabling `NodeDeclaredFeatures` feature gate itself does not affect existing workloads, it changes the default behavior for any pod that uses a subsequent feature built upon this framework. For such pods, scheduling and admission will be automatically restricted to compatible nodes. This is a change in default behavior because it occurs without the user explicitly adding any scheduling constraints.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the feature can be disabled by setting the `NodeDeclaredFeatures` feature gate to false. The impact of a rollback is limited to workloads that rely on new features declared through this mechanism.

* If disabled on the control plane: The scheduler and admission controller plugins will bypass the node declared features based check and revert to the existing scheduling behavior.
* If disabled on the node: Kubelet will stop declaring the features. If the control plane still has the feature enabled, it will now see the node as incompatible for any pod requiring those features. The node remains available for all other workloads.

###### What happens if we reenable the feature if it was previously rolled back?

* Re-enabled on control plane: Re-enabling the feature on the control plane immediately resumes scheduling and admission checks. If nodes still have the feature disabled, they will not declare any features, and any pod requiring those features will become unschedulable. 
* Re-enabled on nodes: Kubelets will begin declaring features in node.status. If the control plane has the feature enabled, the scheduler will see these nodes as eligible for pods requiring those features. If the control plane feature is disabled, there is no scheduling impact.

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

* Kubelet: When the feature gate is toggled, verify that the kubelet correctly populates or clears `node.status.declaredFeatures`.
* Control Plane (kube-scheduler & admission-controller):
    * When the feature gate is toggled from on to off, verify that the declared features based filtering and validation is bypassed. This will be tested by confirming that a pod requiring a specific declared feature is not filtered from an incompatible node.
    * When the feature gate is toggled from off to on, verify that the declared features based filtering and validation is active and is correctly applied to nodes.

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

N/A. Workloads cannot explicitly request this feature.

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

This feature is designed to be mostly invisible to end users during normal operation. The user may become aware of this if there is an incompatible , and the pod remains pending.

- [x] Events
    - Event Reason: FailedScheduling
    - Details: When a user creates a pod that requires a feature not available on any node in the cluster, the pod will remain `Pending`. The scheduler updates the pod's status condition to `Unschedulable`. By running kubectl describe pod <pod-name>, the user will see an event with a message clearly stating the reason, for example: 0/5 nodes are available: 5 node(s) did not match node declared features: GuaranteedQoSPodCPUResize.

- [x] API .status
    - Other field: node.status.declaredFeatures
    - Details: User can verify that a feature is being declared by a specific node by inspecting its API object.

- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

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
N/A

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
No

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

No. Declared features are published as part of the existing Node Status updates.

###### Will enabling / using this feature result in introducing new API types?

No. This feature will not introduce new top-level API types. Instead, it will modify the existing Node API type by adding a new field node.status.declaredFeatures, to convey declared feature information.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes. The size of the `Node` object is expected to increase as more declared features are introduced. However the number of declared features exported will be limited as the node declarations would be removed as a part of the standard post-GA feature cleanup process.


###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Yes, an increase is possible for pod scheduling operations. This potential increase is because kube-scheduler will need to extract feature requirements specified in the Pod Spec and match them against the features declared by the node. However, this additional processing overhead is expected to be comparable to that of existing scheduling predicates like taint/toleration or Node Label/Selector matching.

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

The scheduler relies on the API server to fetch node information. Without access to the API server, the scheduler would continue scheduling based on the cached node status information which may result in incorrect scheduling decisions. This is consistent with how the scheduler handles all status information and is not specific to this feature.

###### What are other known failure modes?

* Pods will remain in a `pending` state if no nodes are declaring a required feature. This is an expected behavior. By running `kubectl describe pod <pod-name>`, the user will see an event which would indicate the missing features.

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

N/A

## Implementation History

- 2025-05-09: [Proposal draft](https://docs.google.com/document/d/1vSDlAA3o0riVq0EcmGBOYUJUVF4tN2Ib7VJg3o1LvBw/edit?tab=t.0) and discussion.
- 2025-05-13: Initial discussion in SIG Node meeting.
- 2025-06-12: KEP discussion in SIG Scheduling meeting.
- 2025-06-26: KEP discussion in SIG Architecture meeting.

## Drawbacks

1. **Difficult Ad-Hoc Overrides:** Labels/Taints/Tolerations can easily be added, removed, or modified to quickly influence scheduling. The fact that declared features are set by the Kubelet and auto-detected from the pod spec makes them unsuitable for ad-hoc overrides.
2. **Greater Scheduler Complexity:** The scheduler plugins needed to interpret and match these declared features against pod requirements (which may be inferred) could be more complex than existing label or taint matching.
3. Declared features can make it easier to support a diverse set of runtime features and could lead to runtimes supporting an arbitrary subset of k8s features. This might lead to more heterogeneity in Node configurations,  which is harder to support.
4. This would make the `NodeStatus` object larger and less focused on just the operational status.
    *   To ensure the number of declared features remains manageable, the design requires that every declared feature be directly actionable by a control plane component.
5. Updating static declared features frequently with `NodeStatus` is an inefficient use of network resources.

## Alternatives

### Using a Meta Opt-In Signal

An alternative design considered using `node-declared-features` as an explicit opt-in signal in `node.status.declaredFeatures`. In this model, control plane components would first check for this signal before applying declared feature-based filtering. If the signal was absent, the filtering logic would be bypassed, making the node eligible for any workload, even one requiring a declared feature the node did not have. This approach was intended to safely ignore older nodes or nodes where the feature was rolled back, allowing them to revert to legacy behavior.

**Drawbacks:**

The primary issue is that the "bypass" logic makes the framework unreliable for managing version skew for any new Alpha or Beta features. If a non-participating node is considered eligible by the filter, a pod requiring a new declared feature could be scheduled onto that older, incompatible node, leading to the exact runtime failures this KEP aims to prevent. Consequently, any new feature could not safely depend on this framework until the NodeDeclaredFeatures feature itself was GA and universally enabled across the cluster.

### Automatic Declared Feature Deprecation

An earlier version of this KEP proposed an automated deprecation mechanism where declared features were removed after a fixed period post-GA. This approach was not adopted for the following reasons:

1.  **Inflexible Lifetime for Kernel/Runtime Dependencies:** The primary concern was that a declared feature with dependencies on the kernel or a container runtime might need a lifetime that extends beyond the standard Kubernetes support window. A rigid, automatic deprecation would not work in such cases.
2.  **Disincentivizes Proper Code Cleanup:** Automating the behavioral change could lead to developers neglecting to remove the underlying declared feature logic from the codebase, creating technical debt.

### Using Node Labels and Node Affinity with SemVer comparison

This approach leverages the existing node affinity mechanism to control pod placement based on node features. Node labels can be introduced by cluster administrator, Kubelet (well-known labels) or tools like Node Feature Discovery (NFD), which runs on each node, discovers its hardware and software characteristics, and automatically applies them as labels. Specifically for kubelet features, we could have Kubelet version as a node label. A user, cluster administrator, or admission webhook would introduce a nodeAffinity rule in the Pod specification based on the features it needs to target nodes with the appropriate kubelet versions. This also requires the node affinity kube-scheduler plugin to be enhanced to support SemVer comparison.

**Pros**

* No core API Change. Leverages an existing and well-established Kubernetes mechanism for attaching metadata.
* Supports node-restricted labels. For labels in restricted domains (e.g., kubernetes.io/),the admission controller prevents the Kubelet from modifying them.

**Cons**

* Node labels often become de facto APIs for controllers and other components, making them difficult to change or deprecate once related features reach GA.
* Operational overhead: To create correct affinity rules, the user or cluster administrator should have a detailed understanding of features needed by the workload and what kubelet versions have those features enabled.
* A node's Kubelet version indicates if the feature is present, but there is no way to know if the feature is actually enabled via a feature gate on the node. This would make the signal unreliable for features under active development (non-default).
* This approach would not work if there is a dependency on kernel or runtime configurations.

### Using a `map[string]string` for `DeclaredFeatures`

Instead of a simple `[]string`, a `map[string]string` could be used. This approach uses a key-value map, where the key is the feature name and the value provides additional information. For simple boolean features, the value would be `"true"`.

**Pros:**

*   **Flexible:** This is the primary advantage. While the initial use case is for boolean "enabled/disabled" features, this structure can accommodate more complex scenarios in the future without an API change if we decide to extend the framework beyond version-skew usecases.
*   **Efficient Lookup:** Checking for the presence of a feature is a direct O(1) hash map lookup, which is more efficient than the O(n) linear scan required for a string slice. The performance impact should also be negligible in practice as the number of `declaredFeatures` is expected to be small. This is also easy to optimize; consumers can convert the slice to a hash set and pass it to the shared library's `MatchNode()` function, which accepts a map input.

**Cons:**

*   **Increased Verbosity and Object Size:** For the common use case of boolean features targeted towards version-skew, the value (`"true"`) is redundant. This leads to larger Node objects which consumes more etcd storage.

### Introducing a `NodeCapabilities` API

**Pros**

* Keeps node capabilities information separate from the operational status in `NodeStatus`.
* Updates can be infrequent and only when the capability changes.
* Allows for evolving the features API independently. Easier to extend with new feature types and fields.

**Cons**
* Increased API server load - kube-scheduler should start watching the new core API object.
* Additional etcd load due to creating, updating, and reading these new objects.
* Increased scheduling latency as the kube-scheduler now needs to reconcile additional objects to make scheduling decisions.

### Alternative Naming Conventions

#### Alternative Names for the field in `Node.Status`

An earlier version of this proposal used the more generic term `capabilities` for the field in `node.status`. However, the decision was made to adopt a more specific name to better reflect the intended scope. The name `declaredFeatures` strongly implies a connection to Kubernetes Features and their lifecycle. Another key advantage of `declaredFeatures` is that it has a consistent meaning, both when a feature is new and may not be available on all the nodes and after it is assumed to be universally available in the cluster.

Several other names were considered:

  * FeatureReadiness: avoided to prevent confusion with other readiness concepts within Kubernetes.
  * EligibleFeatures / AvailableFeatures / CompatibleFeatures: These were considered, but the concern is that they may be interpreted as an exhaustive list of features enabled on the Node, which is not the intended scope.
  
#### Alternative Naming Conventions for Declared Feature Keys

The initial proposal required all feature keys to use a DNS-style prefix, such as `compatibility.kubernetes.io/`. The goal was to signal the temporary nature of the declared features while also making it extensible for more capability types being introduced in the future. A decision was made to drop the prefix because there are no current plans to make these declared features extensible. If extensibility is required in the future, the un-prefixed keys can be reserved for core declared features and prefixes can be introduced for the new extensions. The temporary nature of these declared features is intended to be conveyed by the field name in `node.status`, rather than encoded in each key. This keeps the declared feature keys simple and direct.
