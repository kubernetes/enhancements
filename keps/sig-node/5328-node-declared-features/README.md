# KEP-5328: Node Declared Features


<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Existing Mechanisms and Limitations](#existing-mechanisms-and-limitations)
- [Proposal](#proposal)
  - [Node Declared Features Requirements](#node-declared-features-requirements)
- [User Stories](#user-stories)
  - [Feature Rollout Challenges with Version Skew](#feature-rollout-challenges-with-version-skew)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
    - [Declared Feature Semantics](#declared-feature-semantics)
  - [Kubelet Changes](#kubelet-changes)
  - [Shared Feature Matching Library](#shared-feature-matching-library)
    - [Multi Cluster Support](#multi-cluster-support)
  - [kube-scheduler Changes](#kube-scheduler-changes)
    - [Plugin Implementation](#plugin-implementation)
    - [Performance Considerations](#performance-considerations)
    - [Node Autoscaling Integration](#node-autoscaling-integration)
      - [Scaling based in existing nodes (Scale from &gt; 0)](#scaling-based-in-existing-nodes-scale-from--0)
      - [Scaling based on specifications (Scale from Zero)](#scaling-based-on-specifications-scale-from-zero)
  - [Admission Controller Changes](#admission-controller-changes)
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
    - [Node Autoscaling Integration - Bypass Scheduler Filter for &quot;Scale-from-0&quot; Scenarios](#node-autoscaling-integration---bypass-scheduler-filter-for-scale-from-0-scenarios)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [] (R) Production readiness review completed
- [] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes a **Node Declared Features** framework for nodes to declare the availability of specific, feature-gated Kubernetes features. This would then be used by control plane components (such as the kube-scheduler, admission controllers, or the API server itself) to make informed decisions, primarily to manage version skew. For scheduling, the kube-scheduler would utilize these declared features to ensure pods are only placed on nodes that possess the necessary features to run them successfully. For API request validation, admission controllers would prevent operations on nodes that lack the required feature support. The intent is to streamline cluster operations by reducing the reliance on manual configurations like taints, tolerations, and complex node labeling schemes.

## Motivation

The primary motivation for this KEP is to solve scheduling and validation problems that arise from version skew. When a new feature is enabled on the control plane, a mismatch often occurs because nodes are upgraded gradually or are simply running older Kubelet versions, as is permitted by the Kubernetes version skew policy. This creates a window where the scheduler might place a pod requiring a new feature onto a node that does not support it.

By making the scheduler aware of the features on the node, this proposal ensures that incompatibilities are handled correctly and proactively instead of failing later with a runtime error or a Kubelet admission failure on the node. A pod without a matching node remains `Pending` with an event that details the unmet feature requirement, providing actionable user feedback ([slack discussion](https://kubernetes.slack.com/archives/C5P3FE08M/p1741867194258139)).

### Goals

1. Define a standard mechanism for nodes to declare features that are tied to the lifecycle of new Kubernetes features to manage version skew.
2. Introduce a shared library to encapsulate the logic for inferring a pod's feature requirements and matching them against a node's declared features, ensuring consistency between control plane components that depend on this mechanism.
3. Enhance the kube-scheduler to filter nodes based on the pod's requirements.
4. Enable API admission controllers to validate requests for operations against a node's actual feature support.
5. Enable Kubelet admission plugin to check if the Pod is compatible with the node's features.
6. Provide an API in the shared library for autoscalers (Eg: Cluster Autoscaler, Karpenter) to deterministically resolve node declared features based on static node configuration.

### Non-Goals

1. Replace Taints/Tolerations or Node Labels/Selectors/Affinity.
2. Serve as a reporting mechanism for permanent static node attributes (like architecture, or specific hardware).
3. The feature declaration and matching mechanism is designed to support only new features introduced after this framework is in place. It is not applicable to Kubernetes features that are already implemented.
4. Implementation of integration logic within node autoscalers. The shared library provides the API to determine declared features, but the actual integration logic remain out of scope for this KEP.

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

This proposal introduces a new field `declaredFeatures` to node's `.status` to expose information which the kube-scheduler, admission controller or the API server would use to make more informed decisions. The Kubelet is primarily responsible for discovering, consolidating, and declaring features to the API server.

### Node Declared Features Requirements

1. Every feature added in a Node's `status.declaredFeatures` is temporary and must be associated with a Kubernetes feature graduating through the Alpha/Beta/GA process. This is to ensure that declared features are not used as permanent node attributes and are removed as part of the standard post-GA feature cleanup process.
2. The NodeDeclaredFeatures framework must be used for new features. Onboarding features that have already graduated to Beta or GA must be avoided as the control plane would be unable to differentiate between a node that has the feature but is not declaring it, and a node that genuinely lacks the feature.
3. Kubelet must determine the list of declared features once during its bootstrap and update the node's `status.declaredFeatures`. This list must be derived solely from feature gates and node's static configuration. Reporting new or changed feature gates requires a Kubelet restart.
4. A new feature should only be declared if a control plane component (like kube-scheduler, admission controllers, or the API server) can use it to make a decision. Examples include filtering nodes for pod scheduling, validating an API request, or altering component interaction with the node.
5. Features must not be declared if they are only selectively dependent on a feature gate (i.e., the feature gate is required in certain node configurations but the feature works by default in others).
    *   For example, in-place CPU resizing support for Guaranteed QoS pods is not a suitable declared feature. This is because kubelet support for guaranteed pod resize depends on the `InPlacePodVerticalScalingExclusiveCPUs` feature gate only when static CPU policy is enabled in kubelet config. Using declared features here would cause the control plane to incorrectly block valid requests to older nodes with static CPU policy disabled.

## User Stories

### Feature Rollout Challenges with Version Skew

To maintain cluster stability and enable safer rollouts, cluster administrators perform gradual upgrades. This inherently creates a mixed-version Kubernetes cluster, where nodes can be on different versions than the control plane. The primary concerns in such clusters are:

1. How do we prevent new pods that might use newer features from being scheduled on older, incompatible nodes?
  *   Example: [Pod Level Resources](https://github.com/kubernetes/enhancements/blob/63d4f6f2aa0e2eb0b83067b067c4949643b1b24c/keps/sig-node/2837-pod-level-resource-spec/README.md?plain=1#L4)
    *   The existing beta feature lacks explicit version-skew management, requiring it to be enabled on all components (scheduler, Kubelet, API server) to function correctly. If enabled only on the control plane, a pod can be scheduled on an incompatible node where the Kubelet will reject it during admission. This rejection puts the pod into a `Failed` state, preventing it from being retried on other newer (compatible) nodes in the cluster.
    *   For new enhancements, like adding new container restart rules ([KEP-5532](https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/5532-restart-all-containers-on-container-exits/README.md)), ensuring a pod that needs the feature lands on a compatible node requires significant manual coordination, such as custom node labels, taints, and corresponding pod selectors/tolerations.

2. How do we block API calls that target pods on nodes that don't support newer features?
  *   Example: [In-Place Pod Resizing](https://kubernetes.io/docs/tasks/configure-pod-container/resize-container-resources/)
    *   For an existing beta feature, there currently exists a [workaround](https://github.com/kubernetes/kubernetes/blob/23258f104d74c6f27fd4db94940d745d9d463a8f/pkg/apis/core/validation/validation.go#L5796) to handle this version skew by looking for alternate signals from the pod spec. However such workarounds may be complex and not always feasible for new features.
    *   For a new feature enhancement like supporting In-Place Resize with Pod Level Resources ([KEP-5419](https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/5419-pod-level-resources-in-place-resize/README.md)), an API request to modify a running pod must be validated. The operation should be rejected if the pod resides on an older node that does not support the feature.

3. How can control plane components discover and adapt to node-level features ?
  *   Example: Transitioning from SPDY to WebSockets ([KEP-4006])(https://github.com/kubernetes/enhancements/issues/4006)
    *   When introducing WebSocket support between the API server and Kubelet, the API server needs to know if the target Kubelet can handle WebSockets.
    *   A node can declare a feature like `ExtendWebSocketsToKubelet`. The API server can check for this feature and use WebSockets if available, falling back to SPDY otherwise. This allows gradual rollout of the new protocol.

The Node Declared Features framework addresses these issues through a unified mechanism without requiring any intervention (like adding Node Labels, selectors etc.). The scheduler uses the declared feature information to correctly filter nodes for new pods, the admission controller validates operations against the declared features of a pod's current node, and other components like the API server can adjust their behavior based on node features. This ensures that feature incompatibilities are handled proactively at the control plane.

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
    *   Every registered feature, must have at least one Kubernetes feature gate associated.

**Example:**

```
declaredFeatures:
   - InPlacePodLevelResourcesVerticalScaling
   - RestartAllContainersOnContainerExits
```

### Kubelet Changes

Kubelet has the primary responsibility of discovering the enabled features relevant for control plane during its bootstrap and updating `node.status.declaredFeatures`. It determines its full, static set of declared features **once** upon startup based on its configuration. This feature set is then populated into the `node.status.declaredFeatures` field during the Kubelet's periodic node status update cycle. The Kubelet sends a patch to the API server only when there is a change in the overall `NodeStatus` object or when the status reporting period expires. The Kubelet is the authoritative source of truth for its own declared features. If an external controller or user were to modify the `node.status.declaredFeatures` field on the Node object, the Kubelet's next periodic status update would automatically overwrite and revert those changes.

In addition to reporting, Kubelet will also need to validate the pod. Before admitting a pod to be run on the node, the Kubelet's pod admission handlers check the pod's feature requirements (as inferred from its spec). It will validate these requirements against its own in-memory list of declared features. If a feature required by the pod is not present, the Kubelet will reject the pod, transition it to a `Failed` state, and post an appropriate event to the API server detailing the missing feature requirement. This secondary validation (along with kube-scheduler filtering) is necessary to handle node restart scenarios where a feature that was enabled during pod scheduling is no longer enabled (feature gate flip with node restart).

As a part of the feature's graduation to GA and eventual removal of the feature gate, the Kubelet must stop declaring the corresponding feature in `node.status.declaredFeatures`. This ensures that obsolete information is removed from the `node.status` object as part of the standard feature cleanup process.

### Shared Feature Matching Library

To avoid code duplication and ensure that all components make decisions based on the same logic, a new shared library is introduced in the `k8s.io/component-helpers/nodedeclaredfeatures` staging repository. This library encapsulates all the logic for feature registration, discovery on nodes, inferring pod feature requirements, and matching pod requirements against a node's declared features.

**Feature Registration**

*   Each new feature to be declared in `node.status.declaredFeatures` must implement the `Feature` interface. This interface defines how the presence of a feature on a node is discovered. For features where compatibility
    depends on the pod's configuration, the interface includes methods to Infer requirements from the pod (e.g., InferForScheduling, InferForUpdate). These inference methods are typically used by the scheduler and admission controllers. Other control plane components like the API server might directly check for the presence of a declared feature in `node.status.declaredFeatures` to adjust their behavior based on node features, without needing to infer anything from a pod.
*   The `Discover` method for each feature receives a `NodeConfiguration` struct. This struct provides the context needed for a feature to determine if it's active. The function must determine declared features solely on the Node's static configuration must not depend on dynamic runtime states, hardware or topology, as these cannot be predicted by Autoscalers during scale up simulations. The inputs to this function are:
    *   **`Version`**: The Kubelet's binary version.
    *   **`FeatureGates`**: This allows the `Discover` method to checks the status of any standard Kubernetes feature gates. Every registered feature should have at least one feature gate associated with it.
    *   **`StaticConfig`**: Static configuration fields from the node used for feature discovery. Includes relevant fields from [Kubelet config](https://kubernetes.io/docs/reference/config-api/kubelet-config.v1beta1/) (Eg - static CPU policy). Any new configurations added must be statically determinable by the autoscaler. A declared feature cannot be derived from static configuration alone; it must also be associated with a feature gate. This constraint is enforced by the library to ensure all declared features remain temporary ([declared feature requirements](#node-declared-features-requirements)).
*   The `Feature` interface includes a `MaxVersion()` method. This specifies the upper bound of Kubernetes versions for which the feature is an active scheduling constraint. This should typically be set to the version where the feature becomes GA and is expected to be available on all nodes, considering version skew (e.g., `GA_VERSION + SKEW`). A `nil` return value from `MaxVersion()` indicates no upper version bound.

```go
// Feature encapsulates all logic for a given declared feature.
type Feature interface {
	// Name returns the feature's well-known name.
	Name() string

	// Discover checks if a node provides the feature based on its configuration.
	Discover(cfg *NodeConfiguration) (bool, error)

	// InferForScheduling checks if pod scheduling requires the feature.
	InferForScheduling(podInfo *PodInfo) bool

	// InferForUpdate checks if a pod update requires the feature.
	InferForUpdate(oldPodInfo, newPodInfo *PodInfo) bool

	// MaxVersion specifies the upper bound Kubernetes version (inclusive) for this feature's relevance
	// as a scheduling factor. Should be set based on the feature's GA version
	// and the cluster's version skew policy. Nil means no upper version bound.
	MaxVersion() *version.Version

  // Requirements returns the feature's feature gate and static config dependencies.
	Requirements() *FeatureRequirements
}

// PodInfo is an extensible data structure that wraps the pod object.
type PodInfo struct {
	// Spec is the Pod's specification.
	Spec *v1.PodSpec
	// Status is the Pod's current status.
	Status *v1.PodStatus
  // Add other ancillary resources here in the future as needed.
	// Example: ResourceClaims []*v1.ResourceClaim
}

// StaticConfiguration provides a view of a node's static configuration.
type StaticConfiguration struct {
  // Limited to parameters that are statically determinable from node configuration
  // Example: CPUManagerPolicy string - configured CPU manager policy in kubelet.
  // Any new parameter included must be statically determinable by the autoscaler.
}

// NodeConfiguration provides a view of a node's static configuration.
// This struct contains all inputs required to determine a node's declared features.
type NodeConfiguration struct {
	FeatureGates FeatureGate
	StaticConfig StaticConfiguration
	Version      *version.Version
}

// FeatureRequirements lists the potential dependencies of a feature.
type FeatureRequirements struct {
	// EnabledFeatureGates lists feature gate strings that the feature depends on.
	EnabledFeatureGates []string
	// StaticConfig lists keys from StaticConfiguration that the feature depends on and their expected values.
	StaticConfig map[string]string
}

```

**Framework**

```go
// DiscoverNodeFeatures determines which features from the registry are enabled
// for a specific node configuration. Returns a sorted, unique list of feature names.
// The returned declared feature list is deterministic based solely on the provided NodeConfiguration input.
func (f *Framework) DiscoverNodeFeatures(cfg *NodeConfiguration) []string

// InferForPodScheduling determines which features are required by a pod for scheduling.
func (f *Framework) InferForPodScheduling(podInfo *PodInfo, targetVersion *version.Version) (FeatureSet, error)

// InferForPodUpdate determines which features are required by a pod update operation.
func (f *Framework) InferForPodUpdate(oldPodInfo, newPodInfo *PodInfo, targetVersion *version.Version) (FeatureSet, error)

// MatchNode checks if a node's declared features satisfy the pod's pre-computed requirements.
func MatchNode(requiredFeatures FeatureSet, node *v1.Node) (*MatchResult, error)

// MatchNodeFeatureSet compares a set of required features against a set of features present on a node.
func MatchNodeFeatureSet(requiredFeatures FeatureSet, nodeFeatures FeatureSet) (*MatchResult, error)

// GetFeatureRequirements returns the known dependencies for a given feature name.
func (f *Framework) GetFeatureRequirements(name string) (*FeatureRequirements, error)
```

1.    The `DiscoverNodeFeatures` function takes the node's current `NodeConfiguration`. This configuration includes the status of all relevant Kubernetes feature gates and the values of key static settings on the node. Each registered feature's `Discover` method uses this configuration to determine if the feature is enabled on this particular node.
2.    Inferring Requirements: The library will provide functions  (`InferForPodScheduling`, `InferForPodUpdate`) to inspect `PodInfo` and return a `FeatureSet` of its feature requirements. The inference functions iterate over all registered features. For each feature, they check if the `targetVersion` has exceeded the feature's `MaxVersion()`. If `targetVersion > MaxVersion()`, the feature requirement is silently ignored, as the feature is assumed to be universally available. Otherwise, the feature's `InferForScheduling` or `InferForUpdate` method is called.
3.    Matching Requirements: The library provides functions  (`MatchNode`, `MatchNodeFeatureSet`) to compare the pod's inferred features against a node's declared features.
4.    The `GetFeatureRequirements` returns the feature gate and configuration dependencies for a given feature name. This helps the consumers of the library understand when a node would declare a specific feature. For instance, the cluster autoscaler can use this information to determine the appropriate node group to scale up when a pod goes pending, or help diagnose what feature is missing on existing nodes leading to pod scheduling failures.

This design provides a clear and extensible interface for consumers:
*   **Kubelet:** Uses `DiscoverNodeFeatures` at startup to determine its declared features. The Kubelet's admission handler uses `InferForPodScheduling` and `MatchNodeFeatureSet` to validate pods.
*   **kube-scheduler:** The `NodeDeclaredFeatures` plugin uses `InferForPodScheduling` in the `PreFilter` stage to get pod requirements and `MatchNode` in the `Filter` stage to check against each node. The plugin also implements `EnqueueExtensions` to provide queueing hints for events on Nodes (declared feature changes) and Pods (changes affecting feature requirements).
*   **Admission Controller:** The `NodeDeclaredFeatureValidator` plugin uses `InferForPodUpdate()` to determine requirements for a pod update and `MatchNode` to validate against the node the pod is on.

#### Multi Cluster Support

 To support external controllers that interact with multiple Kubernetes clusters of different versions, the shared library's inference logic is version-aware. The `targetVersion` parameter (the version of the component making the call, e.g., kube-scheduler or kube-apiserver) is passed to the inference functions. The framework uses this `targetVersion` in conjunction with each registered feature's `MaxVersion()` to determine if a feature is still a relevant constraint in that version. Features whose `MaxVersion` is less than the `targetVersion` are considered universally available and are not included in the inferred requirements.

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
    *   **Pod Events:** The scheduler plugin registers for Pod `Update` event. A queue hint is generated if a pending pod is updated in a way that its feature requirements have changed.

This design ensures that the `NodeDeclaredFeatures` scheduler plugin remains generic and does not require modification when new features are added. All feature-specific inference and matching logic is encapsulated within the shared library.

#### Performance Considerations

Introducing this plugin adds a new step to the scheduling cycle, which may have an impact on scheduling throughput. However, this trade-off is acceptable because it prevents the greater inefficiency of scheduling a pod onto a node where it cannot run. To minimize the performance impact, a bitmap-based approach will be implemented for faster matching of node declared features against pod requirements.

#### Node Autoscaling Integration

For the Node Declared Features feature to be fully effective, it must be supported by node autoscalers to enable fully feature-aware node autoscaling decisions. This includes both scaling existing node groups and provisioning entirely new nodes based on specifications (scale-from-zero).

##### Scaling based in existing nodes (Scale from > 0)

This scenario applies when an autoscaler scales up a node group that already has active nodes. The autoscaler can sample a real node from the group to create a template node used for scheduler simulation. Since the real node already has `node.status.declaredFeatures` populated by its running Kubelet, the template inherits these features automatically. 

##### Scaling based on specifications (Scale from Zero)

This scenario applies when an autoscaler scales a node group with zero active nodes or provisions a new node directly from a specification. In these cases, there is no existing `node.status.declaredFeatures` to clone. To support this the autoscaler must populate the `declaredFeatures` on the node template. The requirement for this integration is that list of declared features for a node must be deterministically obtained solely based only on the node's static configuration (kubelet version, Feature Gates, static configurations). This allows autoscalers to predict the features of a potential node before it is provisioned.

The autoscaler determines the inputs for the new node, specifically the `NodeConfiguration` (which includes kubelet version, feature gates, static configuration) and passes this to `DiscoverNodeFeatures()` to get the list of declared features. The autoscaler populates the `node.status.declaredFeatures` field on the template Node object before passing to the scheduler simulation.

### Admission Controller Changes

To enable the validation of API requests against Node Declared Features, this KEP proposes the introduction of a new admission controller plugin, `NodeDeclaredFeatureValidator`, that will be enabled when the `NodeDeclaredFeatures` feature gate is active. For its initial scope, this admission controller will focus on validating Pod `UPDATE` requests to prevent modifications that are incompatible with the node a pod is running on. It only validates updates to the main pod spec or the `resize` subresource. This admission controller cross-references the objects, i.e., looking up the Node object that a Pod is bound to (`spec.nodeName`) and runs the validation checks.

The admission controller workflow will be as follows:
*   Inspect an incoming pod `UPDATE` request and verifies that the pod is already bound to a node by checking that `pod.spec.nodeName` is set. If not, it takes no action.
*   Retrieves the Node object corresponding to `pod.spec.nodeName`.
*   Call the shared library to infer the feature requirements based on the changes between the old and new `PodSpec`. The admission controller uses its own component version (i.e., the kube-apiserver version) as the `targetVersion` argument for the inference function. If the function indicates an issue (e.g., a feature is not known in this version), the admission controller rejects the request.
*   Call the shared library's `MatchNode` function with the inferred feature requirements and the `node.status.declaredFeatures`. If the check fails, the admission controller rejects the request.

### Declared Feature Lifecycle

The lifecycle stages are as follows:

* Introduction (Alpha/Beta): A new string is introduced in `node.status.declaredFeatures` alongside a new feature. Kubelet begins reporting this on nodes where the feature is enabled and active. Control plane components, via the shared library, start using this declared feature to make decisions.
* Graduation (GA): When the feature graduates to GA, the Kubelet continues to declare the feature. This is necessary to manage version skew, allowing the control plane to correctly identify older nodes that do not yet have the GA feature.
* Deprecation and Cleanup (Post-GA): The feature declaration and inference logic is removed from the codebase and can coincide with the removal of feature gate itself.

### Walkthrough

This walkthrough demonstrates the end-to-end lifecycle of the [In-Place Resize with Pod Level Resources](https://github.com/kubernetes/enhancements/issues/5419) feature using the Node Declared Features framework.

**Phase 1: Feature Development**

  * NodeDeclaredFeatures library
    *   A new declared feature with feature key `InPlacePodLevelResourcesVerticalScaling` is registered.
        *   The `Discover()` method checks for the feature gate status.
        *   The `InferForUpdate()` method checks the old and new pod spec to determine if a pod with pod level resources is being resized.
  *   Kubelet: No specific changes required; automatically declares all registered and enabled features in `node.Status`.
  *   NodeDeclaredFeatureValidator: No specific changes required; automatically validates pod updates against all registered features.

**Phase 2: Rollout**
  *   Cluster administrator enables the new feature `InPlacePodLevelResourcesVerticalScaling` on a NodePool.
  *   A user requests an in-place CPU/Memory resize on a pod with pod level requests.
  *   The `NodeDeclaredFeatureValidator` admission controller intercepts the update. It calls the shared library's `InferForPodUpdate` function. The `InferForPodUpdate` function will loop through all the registered features and call their corresponding infer functions. The infer function added for `InPlacePodLevelResourcesVerticalScaling` identifies the need feature requirement.
  *   The `NodeDeclaredFeatureValidator` admission controller calls `MatchNode` function with the inferred pod feature requirements and the Node object from `pod.spec.nodeName`.
  *   The request is admitted only if the node's `declaredFeatures` list **contains** `InPlacePodVerticalScalingExclusiveCPUs`.

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

*   **Kubelet:**
    *   Verify that the Kubelet adds the `declaredFeatures` field in `node.status` when the feature gate is enabled and omits it when disabled.
    *   Verify that the Kubelet's pod admission handler (`declaredFeaturesAdmitHandler`) correctly rejects a pod that requires a declared feature the node does not have, using the shared library.
*   **kube-apiserver:**
    *   Verify the `declaredFeatures` field in `node.status` is correctly served (e.g., on GET, LIST).
    *   Verify API validation rules (regex format, uniqueness, sorting) are enforced for `node.status.declaredFeatures` on updates.
    *   Verify that all the registered features are associated with a feature gate.
*   **Shared Feature Matching Library (`k8s.io/component-helpers/nodedeclaredfeatures`):**
    *   Verify the library accurately infers a pod's feature requirements for scheduling and updates.
    *   Verify the library correctly matches a pod's requirements against a node's declared features.
    *   Verify the library silently ignores a feature requirement if the `targetVersion` is greater than the feature's registered `MaxVersion()`.
    *   Verify the conditional discovery logic for each registered feature (e.g., feature gate status, static node configuration).
    *   Verify that `GetFeatureRequirements()` correctly returns the associated feature gates and static configuration keys for a given declared feature name.
    *   Verify that every registered feature is associated with a feature gate and its being used during discovery.
*   **kube-scheduler (`NodeDeclaredFeatures` scheduler plugin):**
    *   Verify the plugin correctly calls the shared library to infer requirements in `PreFilter` stage and match nodes in `Filter` stage.
    *   Verify event handling for requeuing in `EnqueueExtensions`.
*   **Admission Controller (`NodeDeclaredFeatureValidator`):**
    *   Verify that for a Pod update request (e.g., resize), the validator correctly fetches the declared features of the node specified in `pod.spec.nodeName`.
    *   Verify the plugin correctly uses the shared library to match pod requirements against node declared features, and correctly admits or rejects the request based on the library's result.
    *   Verify feature gate enablement and resource/subresource filtering.

** Test Coverage:**

1. Shared library
- `k8s.io/component-helpers/nodedeclaredfeatures`: `20260115` - `88.2`
- `k8s.io/component-helpers/nodedeclaredfeatures/features/inplacepodresize`: `20260115` - `84`
- `k8s.io/component-helpers/nodedeclaredfeatures/features/restartallcontainers`: `20260115` - `84.6`

2. kube-scheduler
- `pkg/scheduler/framework/plugins/nodedeclaredfeatures`: `20260115` - `64.1`

3. Admission controller
- `plugin/pkg/admission/nodedeclaredfeatures`: `20260115` - `71.6`

4. kubelet
- `pkg/kubelet/kubelet_node_declared_features.go`: `20260115` - `100`
- `pkg/kubelet/lifecycle/handlers.go`: `20260115` - `84.4`

##### Integration tests

*   kube-scheduler (for the `NodeDeclaredFeatures` scheduler plugin):
    *   **Alpha:**
        *   Integration tests in `test/integration/scheduler/filters/filters_test.go`: Verify the plugin's PreFilter and Filter logic, ensuring pods are scheduled only on nodes with matching `declaredFeatures`.
            *  [TestNodeDeclaredFeaturesFilter](https://github.com/kubernetes/kubernetes/blob/f4f3e5f92c38d8f3005996201bd2cdccd16629bc/test/integration/scheduler/filters/filters_test.go#L3171C6-L3171C36):[integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master&include-filter-by-regex=filters): [triage search](https://storage.googleapis.com/k8s-triage/index.html?date=2026-01-15&test=NodeDeclaredFeatures)
        *   Integration tests in `test/integration/scheduler/queueing/queueinghint/queue_test.go`: Verify the Enqueue extensions, ensuring pods are re-queued correctly when relevant Node or Pod updates occur.
            *   [TestCoreResourceEnqueue](https://github.com/kubernetes/kubernetes/blob/f4f3e5f92c38d8f3005996201bd2cdccd16629bc/test/integration/scheduler/queueing/queueinghint/queue_test.go#L32C6-L32C29):[integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master&include-filter-by-regex=queueinghint): [triage search](https://storage.googleapis.com/k8s-triage/index.html?date=2026-01-15&test=Enqueue)
    *   **Beta:**
        *   Performance tests would be introduced in `test/integration/scheduler_perf` to measure the plugin's impact on scheduling throughput and latency.
*   Admission Controller (for the `NodeDeclaredFeatureValidator` admission controller):
    *   **Alpha:**
        *   Integration tests in `test/integration/pods/pods_test.go`: Verify that the admission plugin correctly admits or rejects Pod `UPDATE` operations based on the `declaredFeatures` of the Node the Pod is bound to.
            *   [TestNodeDeclaredFeatureAdmission](https://github.com/kubernetes/kubernetes/blob/f4f3e5f92c38d8f3005996201bd2cdccd16629bc/test/integration/pods/pods_test.go#L1504C6-L1504C38):[integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master&include-filter-by-regex=pods): [triage search](https://storage.googleapis.com/k8s-triage/index.html?date=2026-01-15&test=NodeDeclared)


##### e2e tests

Dedicated E2E tests for this framework are not being added because they would need to rely on specific features being declared. Since features using this framework are pre-GA and stop declaring the feature after GA, any such E2E test would break when the underlying feature graduates. The end-to-end functionality is validated through other features leveraging the node declared features framework. Currently, three alpha features ([InPlacePodVerticalScalingExclusiveCPUs](https://github.com/kubernetes/kubernetes/blob/5f4adaf57935eaeb0d7b924c60ffe4abdde32007/staging/src/k8s.io/component-helpers/nodedeclaredfeatures/features/inplacepodresize/guaranteed_cpu_resize.go), [InPlacePodLevelResourcesVerticalScaling](https://github.com/kubernetes/kubernetes/blob/5f4adaf57935eaeb0d7b924c60ffe4abdde32007/staging/src/k8s.io/component-helpers/nodedeclaredfeatures/features/inplacepodresize/pod_level_resource_resize.go), [RestartAllContainersOnContainerExits](https://github.com/kubernetes/kubernetes/blob/5f4adaf57935eaeb0d7b924c60ffe4abdde32007/staging/src/k8s.io/component-helpers/nodedeclaredfeatures/features/restartallcontainers/restart_all_containers.go)) that depend on this framework for version skew management.

Integration tests are added to provide coverage for `NodeDeclaredFeatures` scheduler plugin and `NodeDeclaredFeatureValidator` admission controller. These tests validate that the features declared by the Kubelet in `node.status.declaredFeatures` are correctly considered by the scheduler when evaluating a node for a pod and by the admission controller when validating pod updates.

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

*   Feature gate `NodeDeclaredFeatures` is enabled by default.
*   Enhance the shared library to support Node Autoscaler integration. This include:
    *  Providing an API to deterministically derive declaredFeatures from static node configuration
    *  Providing an API to determine the configuration dependencies (e.g., feature gates) required for a specific declared feature.
*   Unit coverage for the new changes in the shared library.
*   Integration test coverage for the changes added for cluster autoscaler support.
*   Performance tests for the scheduler plugin are implemented to measure scheduling throughput and latency impact, ensuring no significant regressions.

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

###### How can a rollout or rollback fail? Can it impact already running workloads?

*  The behavior when the feature is enabled only on the control plane but not on nodes is covered in the [Version Skew Strategy](#version-skew-strategy) section.
*  Existing workload should not be impacted by rollout. Only new pods that require a feature using this framework would remain pending if none of the nodes support the feature.

###### What specific metrics should inform a rollback?

*   A sudden increase in the `scheduler_pending_pods` metric in the kube-scheduler could point to potential issues in `NodeDeclaredFeatures` plugin. The pods would be stuck in `Pending` state with scheduler events indicating `FailedScheduling` due to `NodeDeclaredFeatures`.
*   An increase in `kubelet_admission_rejections_total` metric with `PodFeatureUnsupported` reason would point to kubelet admission failures.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

Testing plan for beta:

**Initial State:**
*  Create a 1.35 cluster with `NodeDeclaredFeatures` feature gate enabled. Create a 1.34 node in the cluster. 
*  Enable a feature that is using the framework to manage version skew. Eg: `RestartAllContainersOnContainerExits` ([restart_all_containers.go](https://github.com/kubernetes/kubernetes/blob/c6ba23521ce78e90a6765abd7431f75d2cc58966/staging/src/k8s.io/component-helpers/nodedeclaredfeatures/features/restartallcontainers/restart_all_containers.go#L34)).
*  Create `testpod1` requiring `RestartAllContainersOnContainerExits`. The pod should remain pending as the 1.34 node does not support the feature.

**Upgrade:**
*  Upgrade the node to 1.35 and enable `NodeDeclaredFeatures` and `RestartAllContainersOnContainerExits` feature gates. The node should start reporting the feature in `node.status.declaredFeatures`
* `testpod1` should now get scheduled on the node and start running.
*  Flip `NodeDeclaredFeatures` to `false` first on the node and then on the control plane. `testpod1` should remain running. 
*  Flip `NodeDeclaredFeatures` back to `true`.
*  Flip `RestartAllContainersOnContainerExits` to `false` and restart node. `testpod1` transitions to `Failed` after node restart due to kubelet admission failure.
*  Flip `RestartAllContainersOnContainerExits` back to `true` and restart node. `testpod1` should start running.


**Downgrade:**
*  Downgrade the node to 1.34. The node stops reporting any declared features.
* `testpod1` transitions to `Failed` since 1.34 does not support the feature that the pod requires.
*  Delete `testpod1` and recreate it again. Now `testpod1` should remain `Pending`.

**Upgrade:**
*  Upgrade the node to 1.35 and re-enable the feature gates. 
* `testpod1` should now get scheduled on the node and start running.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

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

This feature is designed to be mostly invisible to end users during normal operation. The user may become aware of this if there is an incompatible node, and the pod remains pending.

- [x] Events
    - Event Reason: FailedScheduling
    - Details: When a user creates a pod that requires a feature not available on any node in the cluster, the pod will remain `Pending`. The scheduler updates the pod's status condition to `Unschedulable`. By running kubectl describe pod <pod-name>, the user will see an event with a message clearly stating the reason, for example: 0/5 nodes are available: 5 node(s) did not match node declared features: InPlacePodVerticalScalingExclusiveCPUs.

- [x] API .status
    - Other field: node.status.declaredFeatures
    - Details: User can verify that a feature is being declared by a specific node by inspecting its API object.

- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Existing kube-scheduler SLOs continue to apply since this feature implicitly affects the node filtering when a pod is being scheduled.

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
- [x] Metrics
  - Metric name: scheduler_pending_pods
  - Metric name: scheduler_plugin_execution_duration_seconds{plugin="NodeDeclaredFeatures"}
  - Components exposing the metric: kube-scheduler

  - Metric name: kubelet_admission_rejections_total{reason="PodFeatureUnsupported"}
  - Components exposing the metric: kube-scheduler


###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No

### Dependencies

No

###### Does this feature depend on any specific services running in the cluster?

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

Yes, an increase is possible for pod scheduling operations for pods relying on a declared feature. This potential increase is because kube-scheduler will need to extract feature requirements specified in the Pod Spec and match them against the features declared by the node. However, this additional processing overhead is expected to be comparable to that of existing scheduling predicates like taint/toleration or Node Label/Selector matching.

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

*   Kubelet will not be able to report `Node.Status` including any changes to `declaredFeatures`.
*   The scheduler relies on the API server to fetch node information. Without access to the API server, the scheduler would continue scheduling 
    based on the cached node status information which may result in incorrect scheduling decisions. This is consistent with how the scheduler handles 
    all status information and is not specific to this feature.

###### What are other known failure modes?

- Pods `pending` due to feature mismatch
    - Detection: Increase in `scheduler_pending_pods` metric. Pod events will show `FailedScheduling` with messages indicating "node(s) did not match node declared features". Operators can see by running `kubectl describe pod <pod-name>`.
    - Mitigations: This happens because of configuration issue where a pod requires features which are not enabled on the nodes. Mitigation is to update the node configuration enabling necessary feature gates or adjusting pod requirements.
    - Diagnostics: Kube-scheduler logs will have details about why nodes were not selected. Node status can be checked to ensure if the necessary feature is being declared.
    - Testing: Covered in integration tests which ensures pods are not scheduled on nodes lacking required declared features.

- API requests rejected during admission
    - Detection: Client side errors on Pod `UPDATE` requests. The metric `apiserver_admission_controller_admission_duration_seconds_count` would show an increase in validation errors for label `NodeDeclaredFeatureValidator`.
    - Mitigations: Mitigation is to update the node configuration enabling necessary feature gates or adjusting pod requirements.
    - Diagnostics: API server logs will contain the errors returned by `NodeDeclaredFeatureValidator` plugin which would indicate the missing feature requirements on the node.
    - Testing:  Covered in integration tests which ensures for the admission controller plugin rejects invalid requests.

*   Kubelet rejects pod during admission:
    -   Detection: `kubelet_admission_rejections_total{reason="PodFeatureUnsupported"}` metric increase. Pod events showing Kubelet rejection.
    -   Mitigation: This would only happen if kubelet was restarted with a feature gate disabled and the feature that was enabled during pod scheduling is no longer enabled. In this case the rejection would be valid. The mitigation would be to adjust pod requirements to not require disabled features.
    -   Diagnostics: Kubelet logs for feature discovery and validation errors.
    -   Testing: Covered in unit tests for Kubelet pod admission handler.

###### What steps should be taken if SLOs are not being met to determine the problem?

*   Review Kube-scheduler logs for errors or delays related to the `NodeDeclaredFeatures` scheduling plugin.
*   Review API Server logs for errors related to the `NodeDeclaredFeatureValidator` admission plugin.
*   Review Kubelet logs for errors related to feature discovery or pod admission.

## Implementation History

- 2025-05-09: [Proposal draft](https://docs.google.com/document/d/1vSDlAA3o0riVq0EcmGBOYUJUVF4tN2Ib7VJg3o1LvBw/edit?tab=t.0) and discussion.
- 2025-05-13: Initial discussion in SIG Node meeting.
- 2025-06-12: KEP discussion in SIG Scheduling meeting.
- 2025-06-26: KEP discussion in SIG Architecture meeting.
- 2025-11-09: Alpha Implementation merged.

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
* Supports node-restricted labels. For labels in restricted domains (e.g., kubernetes.io/), the admission controller prevents the Kubelet from modifying them.

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

#### Node Autoscaling Integration - Bypass Scheduler Filter for "Scale-from-0" Scenarios

We will use Cluster Autoscaler (CA) as our first reference implementation.

###### Scaling a node group with existing nodes (Scale from > 0)

The Cluster Autoscaler can effectively utilize Node Declared Features when scaling node groups for which it has created a template based on a real node in the cluster. This includes:
*   Node groups with active nodes. CA samples a real node from the group to create a template.
*   Node groups with currently no active nodes, but had nodes in the past. CA caches the template previously created from a real node.
In both cases, the template inherits the `node.status.declaredFeatures` and the `NodeDeclaredFeatures` scheduler plugin will correctly consider this in the CA simulator.

###### Scaling Empty Node Groups (Scale from 0)
    
If a node group has no nodes, Cluster Autoscaler cannot create a template based on an existing node. Instead, the cloud provider template is used and this lacks declared feature information (`node.status.declaredFeatures`) as they are populated later by kubelet during bootstrap.
When a pending pod requires a declared feature and CA runs the scheduler simulation, the `NodeDeclaredFeatures` scheduler plugin (filter) will fail to match the node to the pod because the features are missing in the template. As a result, Cluster Autoscaler will not scale up any node group, even if a real node in the group supports the required features.

To ensure compatibility with node autoscaling mechanisms and prevent the scheduler from blocking scale-up in scale-from-zero scenarios, the `NodeInfo` interface in `k8s.io/kube-scheduler/framework/types.go` is extended.

```go
// NodeInfo interface in k8s.io/kube-scheduler/framework/types.go
type NodeInfo interface {
    // ... existing methods like Node(), GetPods(), etc.

    // GetNodeDataOrigin returns the origin of the node info.
    GetNodeDataOrigin() NodeDataOrigin

    // SetNodeDataOrigin sets the origin of node info
    SetNodeDataOrigin(origin NodeDataOrigin)
}

// NodeDataOrigin indicates the source and nature of the Node data.
type NodeDataOrigin string

const (
  ClusterNode NodeDataOrigin = "ClusterNode"
  FromSpecification NodeDataOrigin = "FromSpecification"
)
```

Using this new API, the `NodeDeclaredFeatures` plugin can distinguish between `NodeInfo` objects derived from real nodes and those based on cloud-provider specifications.
*   By default, the node origin is set to `ClusterNode` and is used by the kube-scheduler implementation. The `NodeDeclaredFeatures` scheduler plugin's behavior would remain unchanged for `ClusterNode`.
*   Cluster Autoscaler will be modified to set the origin to
    *  `FromSpecification` when creating a `NodeInfo` from a cloud provider template (i.e., when scaling from zero).
    *  `ClusterNode` when creating `NodeInfo` from a real node ( which has `node.status.declaredFeatures`) sampled from a node group.
*   The `NodeDeclaredFeatures` scheduler plugin's `Filter` method checks this origin. If it's `FromSpecification`, the plugin bypasses the feature check, allowing the simulation to succeed and the scale-up to proceed. This is done because of the absence of declared features in the template node and doesn't necessarily mean the feature will be missing on the actual node.

**Drawback:** 

*   The above solution only unblocks a node group from scaling up. This might lead to scaling up node groups even if the nodes created from the template lack the features required by the pod. The pod will remain pending, as the kube-scheduler will correctly filter out the real node created from the template. This can result in unnecessary node creation and churn.
*   Autoscaling solutions that rely only on specification (like Karpenter) would always bypass the scheduler plugin.