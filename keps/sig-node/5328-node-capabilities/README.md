
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
  - [Kube-Scheduler Changes](#kube-scheduler-changes)
  - [Admission Controller Changes](#admission-controller-changes)
  - [Example Walkthrough](#example-walkthrough)
  - [Changes on Existing Nodes](#changes-on-existing-nodes)
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

This KEP proposes a new **Node Capabilities** framework for Kubernetes Nodes to declaratively inform the control plane about the static capabilities and features they support, such as specific Kubelet, Kernel, and Runtime configurations. This information is designed to help control plane components like kube-scheduler or admission controllers to make better decisions.

For **scheduling**, the `kube-scheduler` would utilize these advertised Node Capabilities by matching a workload's requirements (inferred from the pod spec) against them. This ensures pods are only placed on nodes that definitively possess the necessary features to run them successfully.

For **API request validation**, admission controllers would leverage these Node Capabilities. This allows them to validate incoming API requests and prevent operations if a Node lacks the required static features.

The intent of this Node Capabilities framework is to prevent both the scheduling of pods onto incompatible nodes and the execution of unsupported operations or configurations on specific nodes. This, in turn, aims to streamline cluster operations by reducing the reliance on manual configurations like taints, tolerations, and complex node labeling schemes for managing node-specific features.

## Motivation

This proposal includes "Node Capabilities" in Kubernetes to ensure pods are scheduled and can operate reliably on Nodes while reducing the operational burden. It provides a standardized way for Kubelet to advertise specific node features and configurations, decreasing reliance on manual Taints or Node Labels.

NodeCapabilities aims to prevent pods from being scheduled on incompatible nodes - those missing necessary features because of version skew between control plane and the Node or unsupported runtime/kernel configurations ([slack discussion](https://kubernetes.slack.com/archives/C5P3FE08M/p1741867194258139)). Making the scheduler aware of specific node capabilities will enable more reliable pod placement and ensure that incompatibilities are proactively identified as scheduling failures.

### Goals

1. Define a standard mechanism for Nodes to expose Kubelet, Runtime, and Kernel configurations that are pertinent to workload scheduling and/or improve API Request validation. 
2. Enhance the kube-scheduler to understand pod requirements and match them against Node capabilities and place pods on compatible nodes. 
3. Enable API validation mechanisms to utilize Node Capabilities to verify support for requested operations (e.g., in-place resize, ephemeral containers) before they are processed.

### Non-Goals

1. Replace Taints/Tolerations or Node Labels/Selectors/Affinity.
2. This KEP focuses on introducing the NodeCapabilities API. The exact details of how specific Node Capabilities should be mapped to workload requirements is use case specific and out of scope for this KEP. 


## Existing Mechanisms and Limitations

The Kubernetes scheduler currently uses two primary mechanisms to control pod placement onto specific nodes:

1. Taints and Tolerations 

    Primarily used to **restrict** which pods can schedule onto specific nodes commonly to manage specialized hardware resources. 

    **Standard usage pattern:**
*   Cluster Administrators apply Taints to nodes equipped with special capabilities; cloud providers may also automate this tainting for certain NodePool configurations.
*   Developers add corresponding Tolerations in their Pod specifications for workloads to be able to run on these nodes. Alternatively, Cluster Administrator or Cloud Provider could also inject tolerations through admission webhooks. 

2. Node Labels and Node Selectors/Affinity

    Primarily used to **attract** specific nodes for pods based on the node's characteristics. By applying specific labels to nodes (reflecting Kubelet features, OS version, etc.), we can enable pods to use selectors or affinity to ensure they run on specific nodes.

    **Standard usage pattern:**
*   Cluster Administrators apply specific Labels to nodes to indicate the presence of certain features. This involves applying descriptive labels (Eg:_ kubelet.config.k8s.io/some-alpha-feature=enabled_, _node.kubernetes.io/gvisor-enabled=true_)
*   Optionally for well defined features like[ RunTimeClass](https://kubernetes.io/docs/concepts/containers/runtime-class/), administrators may create abstractions (RuntimeClass Object) that bundle these feature requirements with a node selector targeting the corresponding node labels (Eg: spec._nodeSelector: { "node.kubernetes.io/runtime.gvisor": "true" })_
*   Developers specify their workload's dependency on these features in the PodSpec either directly (spec.nodeSelector) or other mechanisms (eg: spec.runtimeClassName), enabling the scheduler to match them to capable nodes.

**Drawbacks**:

1. Operational Overhead: Cluster administrators have to add the necessary taints/labels to nodes as indirect signals of features and resources, and workloads should use corresponding tolerations/selectors. This needs to be done manually or automations built (webhooks, controllers) to handle standard usage patterns. 
2. Scheduling constraints are encoded indirectly rather than being implicitly understood by kube-scheduler. Discoverability 
3. Incorrect configurations can lead to scheduling failures or suboptimal placements.

## Proposal

This proposal introduces a new field `NodeCapabilities `to` NodeSpec.NodeStatus `to expose information which the kube-scheduler and/or the admission controller would use to make more informed decisions.` `Kubelet is primarily responsible for discovering, consolidating, and reporting `NodeCapabilities` to the API server. The data exposed as NodeCapabilities falls into these categories:

1. **Kubelet Configuration:** Includes relevant kubelet feature gates and configurations. A single logical capability can represent multiple underlying settings.
2. **Runtime Configurations:** Covers scheduler-relevant runtime details like supported handlers and features, obtained via the existing [Status RPC](https://github.com/kubernetes/cri-api/blob/79a12c13b6e0a0e049d166d72b0f352057d9e41b/pkg/apis/runtime/v1/api.proto#L117) between Kubelet and CRI.
3. **Kernel Configurations:** Details kernel support necessary for certain pod features, which may be optional or available only in newer kernel versions.

**NodeCapabilities Requirements:**

1. Should be derived from static configuration.
2. Do not indicate operational health or Node readiness.[ Node Readiness Gates](https://docs.google.com/document/d/11i2_rewvcbQkFFq1BIwHa7lgIefgZ8Mak-QMZjP7fFs/edit?tab=t.0)<span style="text-decoration:underline;"> </span>are better suited for such dynamic readiness signals.
3. Usable during scheduling or API request validation to:
    *   Filter nodes in Kube-Scheduler by matching a pod's needs (from its spec) to Node Capabilities.
    *   Or, Infer from API requests (Eg: in-place resize or ephemeral containers) at the validation stage. 

## User Stories 

### Story 1

Effectively manage version skew during new feature introductions, and enable faster feedback loops with safer rollouts for Kubelet/Runtime features by scheduling Pods to nodes possessing these specific capabilities, especially in mixed-version clusters.

**Examples**: 

*   Alpha features like[ PodLevelResources](https://kubernetes.io/docs/tasks/configure-pod-container/assign-pod-level-resources/) which may be enabled on selected Node Pools for evaluation. Instead of Node labels and selectors, the kube-scheduler can use` pod.spec.resources` to map pods needing the feature to Nodes advertising the capability
*   Beta features like[ InPlacePodResizing](https://kubernetes.io/docs/tasks/configure-pod-container/resize-container-resources/) may be enabled in the Control Plane but not on all the Nodes due to version skew. There currently exists a[ workaround](https://github.com/kubernetes/kubernetes/blob/23258f104d74c6f27fd4db94940d745d9d463a8f/pkg/apis/core/validation/validation.go#L5796) to handle this version skew by looking for alternate signals from the pod spec. But having such workarounds may not always be possible. Alternatively, we could use` pod.spec.nodeName` and validate if the Node has the required capability.


### Story 2

Prevent Pods from being scheduled on Nodes lacking their required Kubelet, Kernel, or Runtime features, avoiding errors, degraded performance, or silent failures.

**Examples:**

*   Linux kernel can be compiled with support for both AppArmor and SELinux security modules, only one can be actively enforced at any given time. This is a key reason why Kubernetes clusters can exhibit heterogeneity in their security configuration, making it crucial for the `kube-scheduler` to know which Linux Security Module is active on each node (typically via Node Labels) for pod placement. Instead, kubelet can expose this kernel capability via NodeCapabilities and the kube-scheduler can infer this from the pod spec (`securitycontext.seLinuxOptions,securitycontext.appArmorProfile`) and match it to the right Node.
*   When pods request for a specific [RunTimeClass](https://kubernetes.io/docs/concepts/containers/runtime-class/) (`spec.runtimeClassName)`, we currently use node selectors to make sure they are on appropriate nodes . Instead, kube-scheduler can automatically identify nodes that possess the corresponding capability.

## Design Details

### API Changes

 \
Add a field `NodeCapabilities` field as type `map[string]string `to the` NodeSpec.NodeStatus` structure. 

```
type NodeStatus struct {
    // ... existing fields
    // NodeCapabilities provides a structured way to report various capabilities of the node. Keys are typically DNS-style and values are strings.
    // +optional 
    // +featureGate=NodeCapabilities
    NodeCapabilities map[string]string `json:"capabilities,omitempty"`
}


// Node object remains unchanged in spec, only status is modified.
type Node struct {
    ...
    Status NodeStatus `json:"status,omitempty" `
}

```

**Example:**

```
NodeCapabilities:
   // Kubelet configurations
   kubernetes.io/feature/podLevelResources: "true"
   kubernetes.io/feature/inPlacePodResize: "true"
   // Runtime configurations 
   runtime.kubernetes.io/handlers: "runc, runsc"
   // Kernel capabilities
   kernel.kubernetes.io/LSMConfig: "seLinux"   
   // Plugins capabilities
   csi.kubernetes.io/my-driver/feature/encryptionSupport: "true"
```

**Note:**  

*   We currently have[ Node Features](https://github.com/kubernetes/api/blob/e8d4d542f6a9a16a694bfc8e3b8cd1557eecfc9d/core/v1/types.go#L6279) and[ Node Runtime Features](https://github.com/kubernetes/api/blob/e8d4d542f6a9a16a694bfc8e3b8cd1557eecfc9d/core/v1/types.go#L6251) which publish some runtime features through Node Status. They are too narrowly scoped and currently not used for scheduling pods. We can deprecate those fields and introduce them into NodeCapabilities if required. 
*   Other API options are explored in the [Alternatives](?tab=t.0#heading=h.vfsx3j6ygclj) section. \


### Kube-Scheduler Changes

To enable the scheduler to filter nodes based on advertised capabilities, this KEP proposes the introduction of a new [Filter Plugin](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/#filter) - `NodeCapabilityFilter`. This plugin will be part of the default scheduler and will be enabled when the `NodeCapabilities` feature gate is active. Additionally, the filter plugin would skip any capability checks for daemonset pods.

`NodeCapabilityFilter` plugin would 

*   Inspect the PodSpec to infer a set of required NodeCapability key-value pairs.
*   For each node being evaluated, the plugin will access the `node.status.capabilities` map. It will then iterate through the pod's inferred requirements and check if each required capability is present and matches on the node. If any inferred requirement is not met, the plugin will filter the node, making it unschedulable for the pod.
*   On a mismatch, the plugin will return an FailedScheduling status with a reason that includes the capability that was not found on Nodes.


### Admission Controller Changes

To enable the validation of API requests against Node Capabilities, this KEP proposes the introduction of a new admission controller plugin `NodeCapabilityValidator `that will be enabled when the NodeCapabilities feature gate is active. This is necessary because the validation logic that requires cross-referencing objects i,e looking up the `Node` object that a `Pod` is bound to (spec.nodeName). THe admission controller would 

*   Inspects incoming pod update request and infer if it depends on any node capability. 
*   It verifies that `pod.spec.nodeName` is set. If the pod is not yet bound to a node, the validator takes no action.
*   Retrieve the `Node` object corresponding to` pod.spec.nodeName` and check the node.Status.capabilities map of the fetched Node object.
*   If the capability is missing or has a non-matching value, the request is marked as invalid

### Example Walkthrough

**New Features and Version Skew Management**

1. When developing a new Kubelet feature Eg:[ In-Place Pod Resize for Guaranteed QoS pods](https://github.com/kubernetes/enhancements/issues/5294), the skew management strategy involves
    *   Modifying the Kubelet to advertise the feature as a Node Capability if the feature gate `InPlacePodVerticalScalingExclusiveCPUs` is active. Kubelet would add <code>[kubernetes.io/feature/inPlacePodResizeExclusiveCpus](http://kubernetes.io/feature/inPlacePodResizeExclusiveCpus): true </code>in<code> status.capabilities</code>.
    *   Updating the <code>NodeCapabilityValidator</code> to check if the node specified in the pod's <code>spec.nodeName</code> possesses this capability before the request is admitted.
2. An Admin enables the <code>InPlacePodVerticalScalingExclusiveCPUs</code> alpha feature gate on a NodePool. 
3. The Kubelet on the Node detects this and advertises its support by setting <code>[kubernetes.io/feature/inPlacePodResizeExclusiveCpus](http://kubernetes.io/feature/inPlacePodResizeExclusiveCpus): true </code>in<code> status.capabilities</code>.<code> </code>
4. An Application Owner has a Guaranteed QoS Pod running on the Node and requests an in place resize. 
5. When the API server processes the resize request, its validation logic for pod updates is invoked in the admission controller. The request is admitted only if the Node supports the feature. \


**Workload Requirements during Scheduling**

1. Nodes with `appArmor` Linux Security Module enabled and configured by the Admin advertise this via Kubelet `kernel.kubernetes.io/LSMConfig: "appArmor" `in `status.capabilities`.
2. An Application Owner deploys a Pod specifying an AppArmor profile `spec.securitycontext.appArmorProfile`. 
3. The `NodeCapabilityFilter` in kube-scheduler will check for the security profile and filter out nodes that do not have app Armor enabled.

### Changes on Existing Nodes

Node capabilities should generally represent mostly stable foundational configurations of the Node, not frequently changing real-time statuses (Example: Driver health). Node capabilities are primarily assessed by the kube-scheduler during pod scheduling. Post-scheduling, the Node Capability changes (e.g., kubelet restart with configuration or feature gate changes) is intended to not have any impact on the running pods on the Node and should only be relevant for new workload. 

### Integration with Existing Mechanisms

We should ideally have one signal to express scheduling intent, but during transition we might end up having multiple active mechanisms to achieve Pod -> Node matching in kube-scheduler. 

**Scenario 1**  
If a certain Node characteristic or constraint is already managed through existing Kubernetes mechanisms (taints/tolerations/node labels/affinity etc.), and now we have a similar (or overlapping) signal through Node Capabilities. 
*   The kube-scheduler changes for NodeCapabilities does not alter any logic related to the existing mechanisms. In such cases, the scheduling restrictions would be additive in nature i,e we need the Node and the Pod to satisfy both NodeCapability and the existing mechanism to be schedulable on the Node. 

**Scenario 2**  
There might not be anything preventing pods from being scheduled but we introduce a new capability detection logic in kube-scheduler that filters out all the nodes that a pod could use. 
*   This is a valid use case for Node Capabilities. If a node lacks a required capability, it should effectively be unscheduled for pods that need it, similar to how a missing label prevents a nodeSelector match.


### Test Plan

[x] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.


##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->
##### Unit tests

*   Kubelet
    *   Verify kubelet adds the `capabilities` field in Node.Status when the feature gate is enabled and rejects or omits it when disabled. 
*   Kube-apiserver
    *   Verify the capabilities field in NodeStatus is correctly served (e.g., on GET, LIST) when the feature gate is enabled and omitted when the feature gate is disabled. 
*   Kube-scheduler 
    *   Not applicable for this KEP. Future KEP’s/PR’s using the field should add checks to validate if the `node.status.capabilities `field is parsed correctly and mapped to workload requirements.

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

*   Not applicable for this API-only change; future KEPs/PR’s using this field will add E2E tests.

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
- `capabilities` field (map[string]string) added to `node.status` and guarded by `NodeCapabilities` feature gate.
- API server serves the field correctly.
- Unit test coverage to validate the presence/absence of the field in the Node object.


### Upgrade / Downgrade Strategy

#### Upgrade

Users can enable the Node Capabilities feature gate after upgrading to a Kubernetes version that supports it. For the feature to be fully effective, both the control plane and Kubelets must be upgraded to this version. Existing workload should not after the upgrade. The [Version Skew Strategy](#version-skew-strategy) section details behavior when only the control plane is upgraded while nodes are still running older Kubelet versions without the feature.


#### Downgrade

Downgrading both kubelet and control plane components to a version without the Node Capabilities feature means subsequent scheduling decisions and API request validations will no longer utilize these capabilities. The [Version Skew Strategy](#version-skew-strategy) section details behavior when only the kubelet is downgraded to a version lacking the feature.

### Version Skew Strategy

The control plane (kube-scheduler) can be on a higher version and looks for specific capability which is not yet enabled on the node. \
 \
Scenario 1: Node Capabilities feature enabled in kube-scheduler (v1.X and feature gate ON), not enabled in kubelet  (v1.X-1 or feature gate OFF). 

*   This would be a problem only if we need to start reporting existing features through the capability mechanism. We either need to backfill the capability on older versions or the kube-scheduler change would need a fallback mechanism to infer these required capabilities from legacy signals on non-reporting nodes. This backfilling requirement can be a separate feature gate.

Scenario 2 :  Kube-Scheduler is looking for certain kubelet capability (a new feature added in v1.X) but nodes are running older kubelet versions ( v1.X-2) that do not have the capability.

*   This is a valid use case that Node Capabilities is trying to address. The pod that relies on the feature being enabled should remain pending with a valid status message. 
*   This prevents pods from failing or running with degraded functionality on incompatible nodes, which can happen today if feature-specific node labels and selectors are not properly applied.

### Kubelet Feature Graduation 

For NodeCapabilities that report the status of specific feature gates, after a feature graduates to GA, the reporting of that particular feature gate in NodeCapabilities becomes obsolete and can be automatically removed. Kube-scheduler can stop doing the capability check after GA + allowed version skew.

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

No. Enabling the feature itself doesn't alter default operations or automatically add new node capabilities. Specific capabilities only become active once they are individually introduced via further configuration changes.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature would result in Node Capabilities not being considered for pod scheduling or pod request validation. Existing pods that are already running should be unaffected. 


###### What happens if we reenable the feature if it was previously rolled back?

If the feature is re-enabled after being previously disabled, any new pod scheduling or request validation will again start considering Node Capabilities.

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

No. NodeCapabilities are published as part of the existing NodeStatus updates.

###### Will enabling / using this feature result in introducing new API types?

No. This feature will not introduce new top-level API types. Instead, it will modify the existing Node API type by adding a new field node.status.capabilities, to convey capability information.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes. The size of the Node object is expected to increase as more capabilities are introduced. The number of capabilities exported will be limited by strategies such as:
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

* 2020-05-09: [Proposal draft](https://docs.google.com/document/d/1vSDlAA3o0riVq0EcmGBOYUJUVF4tN2Ib7VJg3o1LvBw/edit?tab=t.0) and discussion.  
* 2025-05-13: Initial discussion in SIG Node meeting.

## Drawbacks

1. **Difficult Ad-Hoc Overrides: **Labels/Taints/Tolerations can easily be added, removed, or modified to quickly influence scheduling. NodeCapabilities reported by Kubelet and auto-detected from pod spec makes it not suitable for overrides. 
2. **Greater Scheduler Complexity: **The scheduler plugins needed to interpret and match these `NodeCapabilities` against pod requirements (which may be inferred) could be more complex than existing label or taint matching.
3. Capabilities can make it easier to support a diverse set of runtime features and could lead to runtimes supporting an arbitrary subset of k8s features. This might lead to more heterogeneity in Node configurations which is harder to support.
4. This would make the `NodeStatus` object larger and less focused on just the operational status.
    *   To ensure the number of capabilities remains manageable, the design requires that every capability be directly actionable by a control plane component.
5. Updating static `NodeCapabilities` frequently with `NodeStatus` is an inefficient use of network resources.

## Alternatives

<table>
  <tr>
   <td style="background-color: null"><strong>Alternate Solution</strong>
   </td>
   <td style="background-color: null"><strong>Advantages</strong>
   </td>
   <td style="background-color: null"><strong>Disadvantages</strong>
   </td>
  </tr>
  <tr>
   <td style="background-color: null">Introducing a<strong> <code>NodeCapabilities</code></strong> API
   </td>
   <td style="background-color: null"><ul>

<li>Keeps node capabilities information separate from the operational status in <code>NodeStatus.</code>
<li>Updates can be infrequent and only when the capability changes.
<li>Allows for evolving the features API independently. Easier to extend with new feature types and fields.</li></ul>

   </td>
   <td style="background-color: null"><ul>

<li>Increase API server load - Kube-scheduler should start watching the new core API object.
<li>Additional etcd load due to creating, updating, and reading these new objects.
<li>Increased scheduling latency as the kube-scheduler now needs to reconcile additional objects to make scheduling decisions.</li></ul>

   </td>
  </tr>
  <tr>
   <td style="background-color: null">Adding well defined NodeLabels in Kubelet
   </td>
   <td style="background-color: null"><ul>

<li>No Core API Change. Leverages an existing and well-established Kubernetes mechanism for attaching metadata.
<li>Does not need periodic updates.</li></ul>

   </td>
   <td style="background-color: null"><ul>

<li>Node labels often become de facto APIs for controllers and other components, making them difficult to change or deprecate once related features reach GA.
<li>Operational overhead: Workload needs to be aware of the Node labels and use appropriate Node selectors / affinity.
<li>Node labels are mutable. </li></ul>

   </td>
  </tr>
</table>
