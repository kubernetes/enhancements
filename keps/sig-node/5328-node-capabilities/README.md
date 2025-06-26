
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
  - [Story 1](#story-1)
  - [Story 2](#story-2)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Kubelet Changes](#kubelet-changes)
    - [Capability Reporting Semantics](#capability-reporting-semantics)
  - [kube-scheduler Changes](#kube-scheduler-changes)
    - [Plugin Implementation](#plugin-implementation)
    - [Performance Considerations](#performance-considerations)
  - [Admission Controller Changes](#admission-controller-changes)
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
  - [Kubelet Feature Graduation](#kubelet-feature-graduation)
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
  - [Using Node Labels and Node Affinity with SemVer comparison](#using-node-labels-and-node-affinity-with-semver-comparison)
  - [Introducing a <code>NodeCapabilities</code> API](#introducing-a-nodecapabilities-api)
- [Case Study](#case-study)
  - [Retrospective on RuntimeClass](#retrospective-on-runtimeclass)
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

This KEP proposes a new **Node Capabilities** framework for Kubernetes nodes to declaratively inform the control plane about the static capabilities and features they support, such as specific Kubelet, kernel, and runtime configurations. This information is designed to help control plane components like kube-scheduler and/or admission controllers to make better decisions.

For **scheduling**, the `kube-scheduler` would utilize these advertised Node Capabilities by matching a workload's requirements (inferred from the pod spec) against them. This ensures pods are only placed on nodes that definitively possess the necessary features to run them successfully.

For **API request validation**, admission controllers would leverage these Node Capabilities. This allows them to validate incoming API requests and prevent operations if a node lacks the required static features.

The intent of this Node Capabilities framework is to prevent both the scheduling of pods onto incompatible nodes and the execution of unsupported operations or configurations on specific nodes. This, in turn, aims to streamline cluster operations by reducing the reliance on manual configurations like taints, tolerations, and complex node labeling schemes for managing node-specific features.

## Motivation

This proposal includes "Node Capabilities" in Kubernetes to ensure pods are scheduled and can operate reliably on nodes while reducing the operational burden. It provides a standardized way for Kubelet to advertise specific node features and configurations, decreasing reliance on manual Taints or Node Labels.

Node Capabilities aims to prevent pods from being scheduled on incompatible nodes - those missing necessary features because of version skew between control plane and the node or unsupported runtime/kernel configurations ([slack discussion](https://kubernetes.slack.com/archives/C5P3FE08M/p1741867194258139)). Making the scheduler aware of specific node capabilities will enable more reliable pod placement and ensure that incompatibilities are proactively identified as scheduling failures.

### Goals

1. Define a standard mechanism for nodes to expose Kubelet, Runtime, and Kernel configurations that are pertinent to workload scheduling and/or improve API Request validation. 
2. Enhance the kube-scheduler to understand pod requirements and match them against Node capabilities and place pods on compatible nodes. 
3. Enable API validation mechanisms to utilize Node Capabilities to verify support for requested operations (e.g., in-place resize) before they are processed.

### Non-Goals

1. Replace Taints/Tolerations or Node Labels/Selectors/Affinity.
2. This KEP focuses on introducing the Node Capabilities as a part of NodeStatus API. The exact details of how specific capabilities should be mapped to workload requirements is use case specific and out of scope for this KEP.
3. Modifications to the Cluster Autoscaler. For the Alpha stage, this proposal does not include any changes to the Cluster Autoscaler's logic to consider Node Capabilities. Defining an integration strategy is deferred as a [future enhancement](#cluster-autoscaler-integration). 


## Existing Mechanisms and Limitations

The Kubernetes scheduler currently uses two primary mechanisms to control pod placement onto specific nodes:

1. Taints and Tolerations 

    Primarily used to **restrict** which pods can schedule onto specific nodes commonly to manage specialized hardware resources. 

    **Standard Usage Pattern:**
*   Cluster Administrators apply taints to nodes equipped with special capabilities; cloud providers may also automate this tainting for certain NodePool configurations.
*   Developers add corresponding tolerations in their Pod specifications for workloads to be able to run on these nodes. Alternatively, cluster administrator or cloud provider could also inject tolerations through admission webhooks. 

2. Node Labels and Node Selectors/Affinity

    Primarily used to **attract** specific nodes for pods based on the node's characteristics. By applying specific labels to nodes (reflecting Kubelet features, OS version, etc.), we can enable pods to use selectors or affinity to ensure they run on specific nodes.

    **Standard Usage Pattern:**
*   Cluster Administrators apply specific Labels to nodes to indicate the presence of certain features. This involves applying descriptive labels (Eg: `kubelet.config.k8s.io/some-alpha-feature=enabled`, `node.kubernetes.io/gvisor-enabled=true`)
*   Optionally, for well-defined features may need to create other resources (Eg: [RunTimeClass](https://kubernetes.io/docs/concepts/containers/runtime-class/)) that bundle these feature requirements with a node selector targeting the corresponding node labels.
*   Developers specify their workload's dependency on these features in the PodSpec either directly (spec.nodeSelector) or other abstractions enabling the scheduler to match them to capable nodes.

**Drawbacks**:

1. Operational Overhead: Cluster administrators have to add the necessary taints/labels to nodes as indirect signals of features and resources, and workloads should use corresponding tolerations/selectors. This needs to be done manually or automations built (webhooks, controllers) to handle standard usage patterns. 
2. Scheduling constraints are encoded indirectly rather than being implicitly understood by kube-scheduler. 
3. Incorrect configurations can lead to scheduling failures or suboptimal placements.

## Proposal

This proposal introduces a new field `Capabilities` to `Node.Status` to expose information which the kube-scheduler and/or the admission controller would use to make more informed decisions. The Kubelet is primarily responsible for discovering, consolidating, and reporting `Capabilities` to the API server. 

**Node Capabilities Requirements:**

1. Must be derived from node's static configuration, which the Kubelet evaluates during bootstrap. Reporting new or changed capabilities generally requires a Kubelet restart to take effect. 
2. Do not indicate operational health or Node Readiness.[Node Readiness Gates](https://github.com/kubernetes/enhancements/issues/5233) are better suited for such dynamic readiness signals.
3. A capability is considered "relevant" for inclusion in `node.status.capabilities` only if it can be used in the control plane to: \
    i.   Scheduling: Filter nodes in kube-scheduler by matching a pod's needs (from its spec) to Node Capabilities. \
    ii.  Validation: Validating API requests in an admission controller (e.g., for in-place resize or ephemeral containers). \
4. Must be validated by the Kubelet at runtime. This is to prevent errors from post-scheduling capability changes. \

The data exposed as capabilities falls into these categories:

1. **Kubelet Configuration:** Includes Kubelet feature gates and configurations. A single logical capability can represent multiple underlying settings.
2. **Runtime Configurations:** Includes Runtime details like supported handlers and features, obtained via the existing [Status RPC](https://github.com/kubernetes/cri-api/blob/79a12c13b6e0a0e049d166d72b0f352057d9e41b/pkg/apis/runtime/v1/api.proto#L117) between Kubelet and CRI.
3. **Kernel Configurations:** Details kernel support necessary for certain pod features, which may be optional or available only in newer kernel versions.

## User Stories 

### Story 1

Effectively manage version skew during new feature introductions, and enable faster feedback loops with safer rollouts for Kubelet/Runtime features by scheduling Pods to nodes possessing these specific capabilities, especially in mixed-version clusters.

**Examples**: 

*   Alpha features like [PodLevelResources](https://kubernetes.io/docs/tasks/configure-pod-container/assign-pod-level-resources/) which may be enabled on selected Node Pools for evaluation. Instead of Node labels and selectors, the kube-scheduler can use` pod.spec.resources` to map pods needing the feature to nodes advertising the capability
*   Beta features like [InPlacePodResizing](https://kubernetes.io/docs/tasks/configure-pod-container/resize-container-resources/) may be enabled in the control plane but not on all the nodes due to version skew. There currently exists a [workaround](https://github.com/kubernetes/kubernetes/blob/23258f104d74c6f27fd4db94940d745d9d463a8f/pkg/apis/core/validation/validation.go#L5796) to handle this version skew by looking for alternate signals from the pod spec. But having such workarounds may not always be possible. Alternatively, we could use` pod.spec.nodeName` and validate if the node has the required capability.


### Story 2

Prevent Pods from being scheduled on nodes lacking their required Kubelet, Kernel, or Runtime features, avoiding errors, degraded performance, or silent failures.

**Examples:**

*  [PodLevelResources](https://kubernetes.io/docs/tasks/configure-pod-container/assign-pod-level-resources/) is currently not supported on Windows nodes. The capability advertised by kubelet can be dependent on both the feature gate and the operating system which the kube-scheduler can filter out all Windows nodes.

## Design Details

### API Changes

 \
Add a `Capabilities` field as type `map[string]string` to the `Node.Status` structure. 

```
type NodeStatus struct {
    // ... existing fields
    // Capabilities provides a structured way to report various capabilities of the node. Keys are typically DNS-style and values are strings.
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

**Example:**

```
Capabilities:
   feature.kubernetes.io/nodeCapabilities: "true"
   feature.kubernetes.io/inPlacePodResize: "true"
   feature.kubernetes.io/podLevelResourcesSupport: "true"
   feature.kubernetes.io/swapSupport: "true"
```

**Note:**  

*   Any new capability being introduced is considered a formal API change and must go through the API review process. This governance will be enforced in `kubernetes/kubernetes` codebase by protecting the list of capabilities with the `api-approvers` OWNERS file.
*   We currently have [Node Features](https://github.com/kubernetes/api/blob/e8d4d542f6a9a16a694bfc8e3b8cd1557eecfc9d/core/v1/types.go#L6279) and [Node Runtime Features](https://github.com/kubernetes/api/blob/e8d4d542f6a9a16a694bfc8e3b8cd1557eecfc9d/core/v1/types.go#L6251) which publish some runtime features through Node Status. They are too narrowly scoped and currently not used for scheduling pods. We can deprecate those fields and introduce them into NodeCapabilities if required. 


### Kubelet Changes

Kubelet has the primary responsibility of discovering the capabilities during bootstrap and updating `node.status.capabilities`.

#### Capability Reporting Semantics

1.  Combine Interdependent Settings
    * If multiple settings are required to enable a feature, they must be collapsed into a single, logical capability.
    * This simplifies decision-making for control plane components like the scheduler, which should not need to understand multiple interdependent settings.

2.  Presence of a Capability
    * A capability should only be present in the `node.status.capabilities` map if the feature is enabled and functional.
    * If a feature is disabled or unsupported, the Kubelet must remove the corresponding capability key from the map.
    * This approach ensures consistency and simplifies logic for the control plane, treating nodes that don't know about a feature the same as those that have it disabled.

3.  Validation Rules
    * Keys and values within the `node.status.capabilities` map must adhere to the same validation rules as standard Kubernetes labels.
    * The Kubelet validates each capability and will discard any invalid key-value pair, logging an error.
    * When introducing a new capability that requires a Kubelet change, developers must ensure that the new keys and values meet these validation rules. This would also be enforced by [unit tests](#unit-tests).

4.  Opt-in Signal
    * When the `NodeCapabilities` feature gate is enabled on Kubelet, it automatically adds the capability `feature.kubernetes.io/nodeCapabilities: "true"`.
    * This explicitly signals to the control plane that the node is participating in the Node Capabilities framework.
    * This helps distinguish a node that has opted-in from an older node or one where the feature gate is off. It also clarifies the status of a node that has opted-in but has no other specific capabilities to publish.

In addition to reporting, Kubelet will also need to validate the pod. Before admitting a pod to be run on the node, the Kubelet's internal sync loop must check if the pod requires any specific capabilities (as inferred from its spec). It will validate these requirements against its own live, in-memory map of supported capabilities. If a required capability is not present, the Kubelet will reject the pod, transition it to a Failed phase, and post an appropriate event to the API server detailing the missing capability. This secondary validation (along with kube-scheduler capability based filtering) is necessary to handle node restart scenerios where a capability that existed during pod scheduling does not exists anymore (feature gate flip with node restart).


### kube-scheduler Changes

To enable capability-based scheduling, this KEP proposes a new scheduler plugin named NodeCapabilityFilter.

#### Plugin Implementation

`NodeCapabilityFilter` plugin would be enabled if the feature gate is enabled in the kube-scheduler and would implement two extension points in the [scheduling framework](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/). 

1. Prefilter
    *  In this stage, the plugin will inspect the PodSpec to infer the set of required NodeCapability key-value pairs. This computed set of requirements is then stored in the prefilter state.
2. Filter: 
    *   For each node being evaluated, this phase retrieves the pre-computed requirements and checks them against the node's `node.status.capabilities` map.
    *   The capabilities filtering logic will run for nodes with `feature.kubernetes.io/nodeCapabilities: "true"`. The nodes that have not opted in the capabilities framework (older nodes or feature gate off on the node) will just return Success status from their Filter.
    *   If any required capability is not present or the value does not match on the node, the plugin will filter out the node, making it unschedulable for the pod. If no nodes satisfy the capability requirements, the pod will remain in a pending state with a  FailedScheduling event detailing the reason. 
#### Performance Considerations

Introducing this plugin adds a new step to the scheduling cycle, which may have an impact on scheduling throughput. However, this trade-off is acceptable because it prevents the greater inefficiency of scheduling a pod onto a node where it cannot run.


### Admission Controller Changes

To enable the validation of API requests against Node Capabilities, this KEP proposes the introduction of a new admission controller plugin `NodeCapabilityValidator `that will be enabled when the NodeCapabilities feature gate is active. This admission controller cross-references the objects, i.e., looking  up the `Node` object that a `Pod` is bound to (spec.nodeName) and runs the validation checks.

The admission controller would implement the below logic
*   Inspects incoming pod update request and infer if it depends on any node capability. 
*   It verifies that `pod.spec.nodeName` is set. If the pod is not yet bound to a node, the validator takes no action.
*   Retrieve the `Node` object corresponding to` pod.spec.nodeName` and check the `node.status.capabilities` map of the fetched node object.
*   If the capability is missing or has a non-matching value, the request is marked as invalid

###  Walkthrough

**New Features and Version Skew Management**

1. When developing a new Kubelet feature Eg: [In-Place Pod Resize for Guaranteed QoS pods](https://github.com/kubernetes/enhancements/issues/5294), the version skew management strategy involves
    *   Modifying the Kubelet to advertise the feature as a Node Capability `feature.kubernetes.io/guaranteedQOSPodCPUResize`. This capability is advertised in the below conditions
        * `InPlacePodResize` and `InPlacePodVerticalScalingExclusiveCPUs` feature gates are enabled and CPU manager policy is set to `static`. 
        * `InPlacePodResize` feature gate is enabled and CPU Manager Policy is `none`.
    *   Updating the `NodeCapabilityValidator` to check if the node specified in the pod's `spec.nodeName` possesses this capability before the resize request is admitted.
2. Cluster administrator enables  `InPlacePodVerticalScaling` and `InPlacePodVerticalScalingExclusiveCPUs` feature gates  and `static` CPU Manager Policy on a NodePool. 
3. The Kubelet on the node detects this and advertises its support by setting `feature.kubernetes.io/guaranteedQOSPodCPUResize: true` in `status.capabilities`.
4. An Application Owner has a Guaranteed QoS Pod running on the node and requests an in-place CPU resize. 
5. When the API server processes the resize request, its validation logic for pod updates is invoked in the admission controller. The request is admitted only if the node supports the feature.
6. Following a feature's GA release and the supported version skew period, the Kubelet can stop reporting that capability, and the admission controller can stop doing the capability check.

**Workload Requirements during Scheduling**

1. The feature developer will enhance the Kubelet to conditionally advertise capabilities for `PodLevelResources`. Kubelet will check that both the feature gate is enabled and the underlying OS is Linux and publishes them using a capability  `feature.kubernetes.io/podLevelResourcesSupport: "true"` capability to the node's status.
2. An Application Owner deploys a Pod that requires pod-level resource management by defining resources directly in the pod spec (e.g., `spec.resources.requests`).
3. The `NodeCapabilityFilter` in kube-scheduler inspects the `PodSpec`. Upon detecting the `spec.resources` field, it infers that the pod requires the `podLevelResourcesSupport` capability. The kube-scheduler plugin will filter out nodes that do not advertise this capability, ensuring the pod is only placed on a compatible Linux nodes.

### Capability Changes on Existing Nodes

Node capabilities are checked by the scheduler during scheduling and then validated again by the Kubelet before a pod's containers can start. If a node's capability is removed (kubelet restart with a feature gate disabled) while a pod is already running, the pod is not automatically removed. Kubelet will re-validate the pod's requirements against the node's current state upon container or node restart, and if a required capability is now missing, it will prevent the container from starting, causing the pod to enter a failed state.

### Integration with Existing Mechanisms

We should ideally have one signal to express scheduling intent, but during transition we might end up having multiple active mechanisms to achieve Pod-to-Node matching in kube-scheduler. 

**Scenario 1**  
If a certain node characteristic or constraint is already managed through existing Kubernetes mechanisms (taints/tolerations/node labels/affinity etc.), and now we have a similar (or overlapping) signal through Node Capabilities. 
*   The kube-scheduler changes for Node Capabilities do not alter any logic related to the existing mechanisms. In such cases, the scheduling restrictions would be additive in nature i,e., we need the node and the pod to satisfy both capability and the existing mechanism to be schedulable on the node. 

**Scenario 2**  
There might not be anything preventing pods from being scheduled but we introduce a new capability detection logic in kube-scheduler that filters out all the nodes that a pod could use. 
*   This is a valid use case for Node Capabilities. If a node lacks a required capability, it should effectively be unscheduled for pods that need it, similar to how a missing label prevents a nodeSelector match.


### Test Plan

[x] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

The examples described in the [Example Walkthrough](#example-walkthrough) section can be used to demonstrate and test Node Capabilities.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->
##### Unit tests

*   kubelet
    *   Verify that the kubelet adds the `capabilities` field in `node.status` when the feature gate is enabled and omits it when disabled.
    *   Verify that the Kubelet correctly applies validation rules (same rules as node labels) to capabilities and discards invalid key-value pairs.
    *   Test the conditional logic
        *   Verify `node.status.capabilities` accurately reflects the state of the `InPlacePodResizeExclusiveCpus` feature gate.
        *   Verify the `podLevelResources` capability is reported only if both the feature gate is enabled and the OS is Linux.
*   kube-apiserver
    *   Verify the capabilities field in NodeStatus is correctly served (e.g., on GET, LIST) when the feature gate is enabled and omitted when the feature gate is disabled.
*   kube-scheduler (for the `NodeCapabilityFilter` plugin):
    *   Verify that the plugin correctly parses the `node.status.capabilities` field from `Node` object.
    *   Test the filtering logic
        *   Given a pod that is inferred to require `PodLevelResources`, the filter must accept nodes that report this capability and reject nodes that do not with an appropriate error message.
*   Admission Controller (for the `NodeCapabilityValidator`):
    *   Verify that for a Pod update request (a resize), the validator correctly fetches the capabilities of the node specified in `pod.spec.nodeName`.
    *   Test the validation logic 
        *   The validator admits the request if the node reports the `InPlacePodResizeExclusiveCpus` capability and rejects the request with an appropriate error message if the node does not report the capability.

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

*   In a cluster with both Linux and Windows nodes, a pod that uses `PodLevelResources` (by having `spec.resources` defined) should only be scheduled on Linux nodes.
*   In a cluster with `[InPlacePodResizeExclusiveCpus](http://kubernetes.io/feature/inPlacePodResizeExclusiveCpus)` disabled, the admission controller should reject CPU resize requests for guaranteed QOS pods.

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
*   The API server correctly serves and validates the `capabilities` field when the feature gate is enabled.
*   The `kube-scheduler` and `admission-controller` plugins are introduced.
*   All unit tests outlined in the Test Plan are implemented and verified.

**Beta:**
*   Revisit the [Explicit Capability Request](#explicit-capability-request) as a beta graduation criteria.

### Upgrade / Downgrade Strategy

#### Upgrade

Users can enable the Node Capabilities feature gate after upgrading to a Kubernetes version that supports it. For the feature to be fully effective, both the control plane and Kubelets must be upgraded to this version. Existing workload should not after the upgrade. The [Version Skew Strategy](#version-skew-strategy) section details behavior when only the control plane is upgraded while nodes are still running older Kubelet versions without the feature.


#### Downgrade

Downgrading both kubelet and control plane components to a version without the Node Capabilities feature means subsequent scheduling decisions and API request validations will no longer utilize these capabilities. The [Version Skew Strategy](#version-skew-strategy) section details behavior when only the kubelet is downgraded to a version lacking the feature.

### Version Skew Strategy

Scenario 1: Node Capabilities enabled in the control plane (v1.X or feature gate ON), not enabled in kubelet  (v1.X-1 or feature gate OFF). 

*   This presents a challenge when existing features begin to be reported through the new Node Capabilities mechanism. For such features, we would need a phased capability reliance in the control plane. The presence of the capability `feature.kubernetes.io/nodeCapabilities: "true"` would indicate if the node itself has opted into the capability framework. Capability-based filtering will only be applied to nodes that have opted into the framework (i.e., those reporting `feature.kubernetes.io/nodeCapabilities: "true"`). Nodes that have not opted not be filtered out, thus retaining the existing scheduling behavior for them 


Scenario 2 : kube-scheduler is looking for certain capability (a new feature added in v1.X) but nodes are running older kubelet versions (v1.X-2) that do not have the capability. Node capabilities feature is GA in v1.X-2 so all nodes are reporting their capabilities.

*   This is a valid use case that Node Capabilities is trying to address. The pod that relies on the feature being enabled should remain pending with a valid status message. 
*   This prevents pods from failing or running with degraded functionality on incompatible nodes, which can happen today if feature-specific node labels and selectors are not properly applied.

### Kubelet Feature Graduation 

For Node Capabilities that report the status of specific feature gates, reporting must continue after the feature graduates to GA to support the version skew policy. This ensures a control plane can correctly manage older nodes where the feature may not be available. After the feature's GA version + supported version skew release, the feature would become universally available on all the nodes in the cluster and at that point, the Kubelet can stop reporting and the control plane can stop consuming the capability. 

### Future Considerations 

#### Explicit Capability Request 

For the Alpha implementation, the pod's capability requirements are inferred by the scheduler plugin. This implicit model requires a code change in the `NodeCapabilityFilter` plugin for each new capability. A suggestion from the SIG Scheduling meeting was to explore a mechanism to keep the scheduler logic generic and avoid having case-specific inference logic.

#### Cluster Autoscaler Integration

The cluster-autoscaler makes scale-up decisions based on `NodeGroup` templates, which are abstract representations of a node. Since node capabilities are reported by a running Kubelet, they are not present on these templates. This creates a prediction gap: the autoscaler cannot know what capabilities a new node will have, making it unable to scale up a node pool from zero nodes to satisfy a capability-specific pending pod.

Considered approaches:

1.  Have the autoscaler inspect a running node in the target node pool and assume all new nodes will be identical. This would work only if a running node exists and fails for the "scale-from-zero" conditions.
2.  This problem is fundamentally the same as what [kubernetes/autoscaler#7799](https://github.com/kubernetes/autoscaler/issues/7799) is tracking to support DRA use cases. The cluster-autoscaler currently does not consider DRA resources while scaling up and the long term solution would likely involve a new API surface to specify and/or modify autoscaler predictions.


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
*Note:* Enabling the NodeCapabilities feature gate alone doesn't alter default behavior; it only activates the underlying framework. Actual changes to behaviors like pod scheduling or admission happen once nodes start publishing new capabilities and control plane components start consuming them.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. During rollback, the nodes with the feature disabled would stop reporting the capabilities `feature.kubernetes.io/nodeCapabilities: "true"`. If this capability is absent, the control plane components (kube-scheduler and admission controller) should not apply capability filtering on downgraded nodes. This behavior is crucial as it enables a standard node-first rollback strategy, allowing the control plane to ignore capability checks for those nodes and thus retaining the current behavior for them.

###### What happens if we reenable the feature if it was previously rolled back?

If the feature is re-enabled after being previously disabled, the control plane components would start using capabilities for new pod scheduling or request validation for nodes that are reporting the capabilities (`feature.kubernetes.io/nodeCapabilities: "true"`).

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
* Kube-scheduler
    * When the feature gate is toggled from on to off, verify that the kube-scheduler plugin becomes inactive and capability-based filtering is bypassed.
    * When the feature gate is toggled from off to on, verify that the kube-scheduler plugin becomes active and correctly applies capability-based filtering to nodes with the capability `feature.kubernetes.io/nodeCapabilities: "true"`.
    * When the feature gate is enabled on the control plane and toggled on the nodes, capability based filtering is applied only on the nodes with the capability `feature.kubernetes.io/nodeCapabilities: "true"`.
* Admission controller
    * When the feature gate is toggled from on to off, verify that the plugin becomes inactive and capability-based API request validation is bypassed.
    * When the feature gate is toggled from off to on, verify that the plugin becomes active and correctly applies capability-based validation if the node has the capability `feature.kubernetes.io/nodeCapabilities: "true"`.
    * When the feature gate is enabled on the control plane and toggled on the nodes, capability based validation is applied only when the node has the capability `feature.kubernetes.io/nodeCapabilities: "true"`.

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

* 2025-05-09: [Proposal draft](https://docs.google.com/document/d/1vSDlAA3o0riVq0EcmGBOYUJUVF4tN2Ib7VJg3o1LvBw/edit?tab=t.0) and discussion.  
* 2025-05-13: Initial discussion in SIG Node meeting.
* 2025-06-12: KEP discussion in SIG Scheduling meeting.

## Drawbacks

1. **Difficult Ad-Hoc Overrides:** Labels/Taints/Tolerations can easily be added, removed, or modified to quickly influence scheduling. Capabilities reported by Kubelet and auto-detected from pod spec makes it not suitable for overrides. 
2. **Greater Scheduler Complexity:** The scheduler plugins needed to interpret and match these capabilities against pod requirements (which may be inferred) could be more complex than existing label or taint matching.
3. Capabilities can make it easier to support a diverse set of runtime features and could lead to runtimes supporting an arbitrary subset of k8s features. This might lead to more heterogeneity in Node configurations which is harder to support.
4. This would make the `NodeStatus` object larger and less focused on just the operational status.
    *   To ensure the number of capabilities remains manageable, the design requires that every capability be directly actionable by a control plane component.
5. Updating static capabilities frequently with `NodeStatus` is an inefficient use of network resources.

## Alternatives

### Using Node Labels and Node Affinity with SemVer comparison

This approach leverages the existing node affinity mechanism to control pod placement based on node features. Node labels can be introduced by cluster admin, kubelet (well-known labels) or tool like Node Feature Discovery (NFD), which runs on each node, discovers its hardware and software characteristics, and automatically applies them as labels. Specifically for kubelet features, we could have Kubelet version as a node label. A user, cluster administrator, or admission webhook would introduce a nodeAffinity rule in the Pod specification based on the features it needs to target nodes with the appropriate kubelet versions. The also requires the node affinity kube-scheduler plugin to be enhanced to support SemVer comparison. 

**Pro's**

* No core API Change. Leverages an existing and well-established Kubernetes mechanism for attaching metadata.
* Supports node-restricted labels. For labels in restricted domains (e.g., kubernetes.io/),the admission controller prevents the Kubelet from modifying them.

**Con's** 

* Node labels often become de facto APIs for controllers and other components, making them difficult to change or deprecate once related features reach GA.
* Operational overhead: In order to create correct affinity rules, the user or cluster administrator should have a detailed understanding of features needed by the workload and what kubelet versions have those features enabled.
* A node's Kubelet version indicates if the feature is present, but there is no way to know if the feature is actually enabled via a feature gate on the Node. This would make the signal unreliable for features under active development (non-default).
* This approach would not work if there is a dependency on kernel or runtime configurations.

### Introducing a `NodeCapabilities` API

**Pro's**

* Keeps node capabilities information separate from the operational status in `NodeStatus.`
* Updates can be infrequent and only when the capability changes.
* Allows for evolving the features API independently. Easier to extend with new feature types and fields.

**Con's** 
* Increased API server load - kube-scheduler should start watching the new core API object.
* Additional etcd load due to creating, updating, and reading these new objects.
* Increased scheduling latency as the kube-scheduler now needs to reconcile additional objects to make scheduling decisions.

## Case Study

### Retrospective on RuntimeClass

This is a case study for how NodeCapabilities, had the framework existed when the `RunTimeClass` feature was being designed, could have offered a simpler, more automated approach.

**How RuntimeClass Works Today:**

1.  Manual Labeling: A cluster administrator must manually apply a label like `runtime.kubernetes.io/gvisor-enabled: "true"` to all nodes that have a specific runtime like gVisor installed and configured.
2.  `RunTimeClass` Object: The administrator creates a `RuntimeClass` object. This object contains a handler name `gvisor` and, a scheduling section with a nodeSelector that targets the label from step 1. 

    Example RuntimeClass Object:
    ```
    apiVersion: node.k8s.io/v1
    kind: RuntimeClass
    metadata:
      name: gvisor
    handler: gvisor
    scheduling:
      nodeSelector:
        "runtime.kubernetes.io/gvisor": "true"
    ```

3.  Pod Consumption: A user creates a Pod and sets `spec.runtimeClassName: "gvisor"` in the pod spec. The scheduler then uses the `nodeSelector` from the `RuntimeClass` object to place the pod on a correctly labeled node.

    Example Pod Spec
    ```
    apiVersion: v1
    kind: Pod
    metadata:
      name: my-gvisor-pod
    spec:
      runtimeClassName: gvisor
      # ... rest of pod spec
    ```

**A Hypothetical Streamlined Approach with Node Capabilities:**

1.  Automated Capability Reporting
    *  On a node, the Kubelet during bootstrap would use the existing [Status RPC](https://github.com/kubernetes/cri-api/blob/79a12c13b6e0a0e049d166d72b0f352057d9e41b/pkg/apis/runtime/v1/api.proto#L117) to get a list of handlers supported by the CRI.
    *  Each supported non-default handler is advertised as a capability on the Node object. 
    *  For default handlers like runc, for Pods that do not set `spec.runtimeClassName`, no capability based filtering is needed. However, to support Pods that request the default handler through `spec.runtimeClassName`, we have 2 options
       *  Option 1: Kubelet to advertise a capability for every supported handler, including the default. While this means, every node would have one extra capabity with a default runtime handler, we would not need any special handling the scheduler.
       *  Option 2: The scheduler plugin could have special logic for the default-handler requesting through `spec.runtimeClassName` and avoid node filtering. This would require the scheduler to make assumptions about the node environment, which is less robust.

    Example Node Status Snippet:
    ```
    status:
      # ... other status fields
      capabilities:
        # ... other capabilities
        "runtime.kubernetes.io/handler-gvisor": "true"
        "runtime.kubernetes.io/handler-runc": "true"
    ```

2. Simplified `RunTimeClass` Object: With scheduling handled by capabilities, the RuntimeClass object no longer needs the `scheduling` field.

  Example Object:
    ```
    apiVersion: node.k8s.io/v1
    kind: RuntimeClass
    metadata:
      name: gvisor
    handler: gvisor
    # The 'scheduling' field is no longer necessary!
    ```

3. Scheduling
    * When a user creates a Pod with a `spec.runtimeClassName`, the `NodeCapabilityFilter` scheduler plugin finds the handler and infers that the Pod requires the corresponding capability (`runtime.kubernetes.io/handler-gvisor: "true"`) and filters nodes.
 
 **Benefits of the Node Capabilities Approach:**

1.  Reduced Operational Overhead: Administrators would no longer need to label the nodes. 
2.  Increased Reliability: The capability is the "source of truth" reported directly by the Kubelet that has verified the runtime handler exists, making it more reliable than a manually applied label.
3.  A Simpler and More Focused API: The `RuntimeClass` object itself could be simplified. This creates a cleaner separation of concerns: the RuntimeClass object defines "what a runtime is" at a cluster level, while NodeCapabilities handles "which nodes can support it" at the node level.
4.  A Consistent Framework: `RuntimeClass` had to implement its own mechanism for node-to-pod matching. With NodeCapabilities, handling runtime handlers becomes just another instance of a standard, unified pattern for managing all types of node-specific features.

