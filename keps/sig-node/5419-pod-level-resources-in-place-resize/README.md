# KEP-5419: In-Place Pod-Level Resources Resize

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non Goals](#non-goals)
- [Proposal](#proposal)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Design Principles](#design-principles)
  - [Components/Features changes](#componentsfeatures-changes)
    - [PodStatus API changes](#podstatus-api-changes)
      - [Resize Restart Policy](#resize-restart-policy)
      - [Implementation Details](#implementation-details)
    - [Surfacing Pod Resource Requirements](#surfacing-pod-resource-requirements)
      - [The Challenge of Determining Effective Pod Resource Requirements](#the-challenge-of-determining-effective-pod-resource-requirements)
      - [Goals of surfacing Pod Resource Requirements](#goals-of-surfacing-pod-resource-requirements)
      - [Implementation Details](#implementation-details-1)
      - [Notes for implementation](#notes-for-implementation)
  - [Test Plan](#test-plan)
    - [Unit tests](#unit-tests)
    - [Integration tests](#integration-tests)
    - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Phase 1: Alpha (target 1.35) [DONE]](#phase-1-alpha-target-135-done)
    - [Phase 2:  Beta (target 1.36)](#phase-2--beta-target-136)
    - [GA (stable)](#ga-stable)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
      - [Upgrade](#upgrade)
      - [Downgrade](#downgrade)
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
- [Future Work](#future-work)
<!-- /toc -->


## Release Signoff Checklist


Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**options** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes extending the existing "In-Place Pod Resize" (IPPR) functionality to support resources specified at the Pod level, building upon the foundations laid by KEP-2837: Pod Level Resource Specifications. Currently, IPPR primarily focuses on dynamically adjusting container-level resource allocations. With the introduction of pod-level resource limits and requests (as proposed in ([KEP#2387](https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/2837-pod-level-resource-spec/README.md)), it becomes essential to enable the in-place resizing of these aggregate pod-level resources without requiring a full pod restart. This enhancement will provide greater flexibility and efficiency in managing the overall resource consumption of multi-container pods

## Motivation

The ability to specify resource requirements at the pod level, as introduced by KEP-2837, offers a more simplified and efficient way to manage resource consumption for multi-container pods. However, without the ability to resize these pod-level allocations in-place, operators would still be forced to recreate pods to adjust their overall resource footprint, negating some of the benefits of dynamic resource management.

Enabling in-place resizing for pod-level resources complements KEP-2837 by allowing:

* Dynamic Adjustment of Aggregate Resources: Operators can scale the total resources
  available to a pod up or down based on changing workload demands, without
  disrupting the services running within.

* Improved Resource Utilization: By dynamically adjusting pod-level limits, clusters
  can respond more efficiently to fluctuations in aggregate resource usage,
  potentially leading to better packing and reduced over-provisioning.

* Reduced Operational Overhead: Eliminating the need for pod recreation for resource
  adjustments simplifies operational tasks and reduces downtime or disruption for
  applications.

### Goals

This proposal aims to:

1. Extend the In-Place Pod Resize (IPPR) functionality to support dynamic
   adjustments of pod-level CPU and Memory resources.
2. Ensure compatibility and proper interaction between pod-level IPPR and existing container-level IPPR mechanisms.
3. Provide clear mechanisms for tracking and reporting the actual allocated
   pod-level resources in PodStatus

### Non Goals
This KEP focuses solely on extending IPPR to pod-level resources, so the non-goals
are largely the same as [IPPR's
non-goals](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/1287-in-place-update-pod-resources#non-goals).
These include:

1. This KEP focuses solely on in-place resizing of core compute resources (CPU and
  Memory) at the pod level. Extending this functionality to other resource types
  (e.g., GPUs, network bandwidth) is outside the current scope.

2. This KEP does not aim to implement dynamic changes to a pod's QoS class based on
   in-place resource resize operations. 

3. No dynamic adjustments for Init Containers that have already finished and can't
    be restarted.

4. No automatic removal of lower-priority pods to make room for a pod that's resizing its resources.

5. This KEP doesn't aim to fix every complex timing issue that can happen between
   the Kubelet and the scheduler during resizes that already exist in [KEP#1287](https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/1287-in-place-update-pod-resources/README.md).

## Proposal
### Notes/Constraints/Caveats

1. cgroupv1 has been moved to maintenance mode since Kubernetes version 1.31.
   Hence this feature will be supported only on cgroupv2. Kubelet will fail to
   admit pods with pod-level resources on nodes with cgroupv1.

2. Currently, in-place pod resizing for pod-level resources will be guarded behind a
   separate feature gate, InPlacePodLevelResourcesVerticalScaling, which will be
   available in alpha from v1.34. InPlacePodLevelResourcesVerticalScaling is an
   **opt-in** feature, and will have no effect on existing deployments.

3. This feature relies on the PodLevelResources, InPlacePodVerticalScaling and InPlacePodLevelResourcesVerticalScaling feature gates being enabled.

### Risks and Mitigations
This KEP focuses solely on extending IPPR to pod-level resources, so the risks
are largely the same as [IPPR's
risks and mitigations](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/1287-in-place-update-pod-resources#risks-and-mitigations)
& [Pod-Level Resources' risks and
mitigations](https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/2837-pod-level-resource-spec/README.md#risks-and-mitigations).
These include:

1. Backward compatibility: For pods with pod-level resources, when Pod.Spec.Resources
   becomes representative of desired state, and Pod's actual resource configurations are
   tracked in Pod.Status.Resources, applications that query PodSpec and rely on
   Resources in PodSpec to determine resource configurations will see values that
   may not represent actual configurations. As a mitigation, this change needs to be
   documented and highlighted in the release notes, and in
   top-level Kubernetes documents.
   
2. Scheduler race condition: If a resize happens concurrently with the scheduler
   evaluating the node where the pod is resized, it can result in a node being
   over-scheduled, which will cause the pod to be rejected with an OutOfCPU or
   OutOfMemory error. Solving this race condition is out of scope for this KEP, but
   a general solution may be considered in the future. 

3. Since Pod Level Resource Specifications is an opt-in feature, merging the feature related changes won't impact existing workloads. Moreover, the feature will be rolled out gradually, beginning with an alpha release for testing and gathering feedback. This will be followed by beta and GA releases as the feature matures and potential problems and improvements are addressed.

4. While this feature doesn't alter the existing cgroups structure, it does change how pod-level cgroup values are determined. Currently, Kubernetes derived these values from the container-level cgroup settings. However, with Pod Level Resource Specifications enabled, pod-level cgroup settings will be directly set based on the values specified in the pod's resource spec stanza, if set. This change in behavior could potentially affect:

Workloads or tools that rely on reading cgroup values: This means that any workloads or tools that depend on reading or interpreting container cgroup values might observe different derived values if pod-level resources are specified without container level settings.

Third-party schedulers or tools that make assumptions about pod-level resource calculation: These tools might require adjustments to accommodate the new way pod-level resources are determined.

To mitigate potential issues, the feature documentation will clearly highlight this change and its potential impact. This will allow users to:

 - Adjust their pod-level and container-level resource settings as needed
 - Modify any custom schedulers or tools to align with new resource calculation method.   

## Design Details

### Design Principles
To ensure this feature is intuitive, and beneficial, we adhered to the
following core design principles:

1. Consistency with Container-Level IPPR: The mechanism for actuating pod-level
   resizes should be as consistent as possible with the existing container-level
   IPPR to minimize new complexities.

2. Observability: Users and cluster components must have clear visibility into the
   actual, allocated pod-level resources post-resize.

### Components/Features changes

#### PodStatus API changes
Extend `PodStatus` to include pod-level analog of the container status resource
fields. Pod-level resource information in `PodStatus` is essential for pod-level
[In-Place Pod Update] (https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/1287-in-place-update-pod-resources/README.md#api-changes)
as it provides a way to track, report and use the actual resource allocation for the
pod, both before and after a resize operation.

```
type PodStatus struct {
...
  // Resources represents the compute resource requests and limits that have been
  // applied at the pod level. If pod-level resources are not explicitly specified,
  // then these will be the aggregate resources computed from containers. If limits are 
  // not defined for all containers (and pod-level limits are also not set), those
  // containers remain unrestricted, and no aggregate pod-level limits will be applied.  
  // Pod-level limit aggregation is only performed, and is meaningful only, when all 
  // containers have defined limits.
  // +featureGate=InPlacePodVerticalScaling
  // +featureGate=PodLevelResources
  // +featureGate=InPlacePodLevelResourcesVerticalScaling
  // +optional
  Resources *ResourceRequirements

  // AllocatedResources is the total requests allocated for this pod by the node.
  // Kubelet sets this to the accepted requests when a pod (or resize) is admitted.
  // If pod-level requests are not set, this will be the total requests aggregated
  // across containers in the pod.
  // +featureGate=InPlacePodVerticalScaling
  // +featureGate=PodLevelResources
  // +featureGate=InPlacePodLevelResourcesVerticalScaling
  // +optional
  AllocatedResources ResourceList
}
```

##### Resize Restart Policy

Pod-level resize policy is not supported in this KEP. While a pod-level resize policy might
 be beneficial for VM-based runtimes like Kata Containers (potentially allowing the hypervisor
to restart the entire VM on resize), this is a topic for future consideration as a separate KEP. 
We plan to engage with the Kata community to discuss this further and will re-evaluate the need for a pod-level
policy in subsequent development stages.

The absence of a pod-level resize policy means that container restarts are
exclusively managed by their individual `resizePolicy` configs. And, if the
containers do not specify `resizePolicy`, `PreferNoRestart` is the default resize
policy.

Note: As stated in [KEP#1287](https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/1287-in-place-update-pod-resources/README.md), PreferNoRestart restart policy for resize (container-level or pod-level) does not guarantee that a container won't be restarted. If the runtime knows a resize will trigger a restart, it should return an error instead, and the Kubelet will retry the resize on the next pod sync. 

The example below of a pod with pod-level resources demonstrates several key
aspects of this behavior, showing how containers without explicit limits (which
inherit pod-level limits) interact with resize policy, and how containers with
specified resources remain unaffected by
pod-level resizes.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: pod-level-resources
spec:
  resources:
    requests:
      cpu: 100m
      memory: 100Mi
    limits:
      cpu: 200m
      memory: 200Mi
  containers:
    - name: c1
      image: registry.k8s.io/pause:latest
      resizePolicy:
        - resourceName: "cpu"
          restartPolicy: "NotRequired"
        - resourceName: "memory"
          restartPolicy: "RestartRequired"
    - name: c2
      image: registry.k8s.io/pause:latest
      resources:
        requests:
          cpu: 50m
          memory: 50Mi
        limits:
          cpu: 100m
          memory: 100Mi
      resizePolicy:
        - resourceName: "cpu"
          restartPolicy: "NotRequired"
        - resourceName: "memory"
          restartPolicy: "RestartRequired"
```

In this example:
* CPU resizes: Neither container requires a restart for CPU resizes, and therefore CPU resizes at neither the container nor pod level will trigger any restarts.
* Container c1 (inherited memory limit): c1 does not define any container level
  resources, so the effective memory limit of the container is determined by the
  pod-level limit. When the pod's limit is resized, c1's effective memory limit
  changes. Because c1's memory resizePolicy is RestartRequired, a resize of the
  pod-level memory limit will trigger a restart of container c1.
* Container c2 (specified memory limit): c2 does define container-level resources,
  so the effective memory limit of c2 is the container level limit. Therefore, a
  resize of the pod-level memory limit doesn't change the effective container limit,
  so the c2 is not restarted when the pod-level memory limit is resized.

##### Implementation Details

###### Allocating Pod-level Resources
Allocation of pod-level resources will work the same as container-level resources.
The allocated resources checkpoint will be extended to include pod-level resources,
and the pod object will be updated with the allocated resources.

Note: Pod-level resize requests will be considered when determining the priority of
the resize operations (ref [#5266](https://github.com/kubernetes/enhancements/pull/5266])).

###### Actuating Pod-level Resource In Place Resize
Pod Level Resources In Place Resize (i.e. Pod Level Resources + IPPR) will be
guarded behind a separate feature gate `InPlacePodLevelResourcesVerticalScaling`,
which will be available in alpha from v1.34. 

The mechanism for actuating pod-level resize remains largely unchanged from the
existing container-level resize process.  When pod-level resource configurations are
applied, the system handles the resize in a similar manner as it does for
container-level resources. This includes extending the existing logic to incorporate
directly configured pod-level resource settings.

The same ordering rules for pod and container resource resizing will be applied for each
resource as needed:
1. Increase pod-level cgroup (if needed)
2. Decrease container resources
3. Decrease pod-level cgroup (if needed)
4. Increase container resources

Note: As part of a resize operation, users will now be permitted to add or modify pod-level resources within the Pod specification. Upon successful update, the Kubelet will ensure its internal checkpoint for the Pod is updated to reflect these new resource definitions. Importantly, to optimize performance and prevent redundant operations, the Kubelet will only trigger an actual cgroup resize for the Pod's sandbox if the specified pod-level resources are not equal to the aggregated sum of the individual container-level resources.

###### Tracking Actual Pod-level Resources
To accurately track actual pod-level resources during in-place pod resizing, several
changes are required that are analogous to the changes made for container-level
in-place resizing:

1. Configuration reading: In Alpha stage of
   `InPlacePodLevelResourcesVerticalScaling`, re-read Pod-level resource config
   in each sync loop. 
   
2. Pod Status Update: Because the pod status is updated before the resize takes
   effect, the status will not immediately reflect the new resource values.  If a
   container within the pod is also being resized, the container resize operation
   will trigger a pod synchronization (pod-sync), which will refresh the pod's
   status.  However, if only pod-level resources are being resized, a pod-sync must
   be explicitly triggered to update the pod status with the new resource
   allocation.

3. [Scoped for Beta] Caching: Actual pod resource data may be cached in memory. This
   cache, if implemented, must be refreshed after each successful pod resize or for
   every cache-miss to ensure that subsequent reads by the kubelet retrieve the
   latest information. The need for and implementation of this caching mechanism
   will be evaluated in the beta phase. Performance benchmarking will be conducted
   to determine if caching is required and, if so, what caching strategy is most
   appropriate.

#### Surfacing Pod Resource Requirements

##### The Challenge of Determining Effective Pod Resource Requirements

Calculating the actual resource requirements of a pod can be quite complex. This
is due to several factors:

* Derived Pod Resources: Without Pod-level resources, the Pod-level resource
  requirements are derived from container level specifications, involving a complex
  formula (as described in [KEP#753](https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/753-sidecar-containers/README.md#resources-calculation-for-scheduling-and-pod-admission)) that considers init containers, sidecar containers, regular containers,
  and Pod Overhead. 

* Defaulting Logic: With Pod-level resources, Kubernetes API server will apply default
  values for resource requests and limits when they are not explicitly
  specified, further adding to the complexity of determining the final resource
  requirements. 

* In-Place Pod Updates: The ability to update the resource requirements without
  restarting the pod
  ([KEP#1287](https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/1287-in-place-update-pod-resources/README.md))
  introduces another layer of complexity in calculating the effective resource
  requirements.
  
  This inherent complexity makes it difficult for users to accurately determine
  the total resources a pod needs. Various components, such as kube-scheduler,
  kubelet, autoscalers, etc, rely on accurate information about a pod's resource
  requirements to function correctly. Expecting users to perform these
  calculations to understand their pod's resource usage and plan their cluster
  capacity is unrealistic and cumbersome. 

  To address this, this KEP proposes reusing the field called `Resources`
  from the Pod Status. This field will provide a clear and readily available
  representation of the total resource requirements for a pending or scheduled
  pod, simplifying resource understanding and capacity planning for users.

##### Goals of surfacing Pod Resource Requirements

* Allow Kubernetes users to quickly identify the effective resource requirements
  for a pending or scheduled pod directly via observing the pod status.
* Provide cluster autoscaler, karpenter etc with direct access resource
  requirement information of unavailable/unschedulable pods, enabling
  them to determine the necessary capacity and add nodes accordingly.
* Allow consuming the Pod Requested Resources via metrics:
  * Make sure kube_pod_resource_requests formula is up to date
  * Consider exposing Pod Requirements as a Pod state metric via
    https://github.com/kubernetes/kube-state-metrics/blob/main/docs/pod-metrics.md 
* Provide a well documented, reusable, exported function to be used to calculate
  the effective resource requirements for a v1.Pod struct.
* Eliminate duplication of the pod resource requirement calculation within
  `kubelet` and `kube-scheduler`.

Note: in order to support the [Downgrade strategy](#downgrade-strategy), scheduler
will ignore the presence of the `PodLevelResources` feature gate when calculating
resources. This will prevent overbooking of nodes when scheduler ignored sidecar
when calculating resources and scheduled too many Pods on the Node that had the `SidecarContainers` feature gate enabled.

##### Implementation Details

To effectively address the needs of both users and Kubernetes components that
rely on accurate pod resource information, the proposed implementation involves
two key changes:

1. The `Resources` field, added to `PodStatus` (as stated in the [api changes](#podstatus-api-changes) section), will serve as the single source of truth
   to represent the effective resource requirements for the pod. It allows
   controller to inspect running and pending pods to quickly determine their
   effective resource requirements. 

PodStatus.Resources field is set in
[generateAPIPodStatus](https://github.com/kubernetes/kubernetes/blob/a668924cb60901b413abc1fe7817bc7969167064/pkg/kubelet/kubelet_pods.go#L1459)
method for now. 

Note: We'll need to revisit this to enable controllers to utilize this field in 1.35. The motivation for defaulting PodStatus.Resources is to allow components, such as admission (e.g., quota controls) or any controllers that run before the pod starts, to use it for calculating pod-level resource totals, rather than relying on a component-helper.

2. Update the
[PodRequestsAndLimitsReuse](https://github.com/kubernetes/kubernetes/blob/dfc9bf0953360201618ad52308ccb34fd8076dff/pkg/api/v1/resource/helpers.go#L64)
function to support the new calculation and, if possible, re-use this
functionality the other places that pod resource requests are needed (e.g.
kube-scheduler, kubelet).  This ensures that components within Kubernetes have an
identical computation for effective resource requirements and will reduce
maintenance effort. Currently this function is only used for the metrics
`kube_pod_resource_request` and `kube_pod_resource_limit` that are exposed by
`kube-scheduler` which align with the values that will also now be reported in 
`Resources` in the pod status.

A key advantage of having a well-defined, exported function for calculating pod
resource requirements is that it can be used by various Kubernetes ecosystem
components, even for pods that don't yet exist (pending pods). For example, an
autoscaler needs to know what the resource requirements will be for DaemonSet
pods when they are created to incorporate them into its calculations if it
supports scale to zero.

##### Notes for implementation

This change could be made in a phased manner:
* Refactor to use the `PodRequestsAndLimitsReuse` function in all situations
  where pod resource requests are needed.
* Reuse the new `Resources` field on `PodStatus` and modify `kubelet` &
  `kube-scheduler` to update the field.

These two changes are independent of the sidecar and in-place resource update
KEPs.  The first change doesn’t present any user visible change, and if
implemented, will in a small way reduce the effort for both of those KEPs by
providing a single place to update the pod resource calculation.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

#### Unit tests

`k8s.io/kubernetes/pkg/kubelet/cm`: `20250618` - 18.4

`k8s.io/kubernetes/pkg/kubelet/kuberuntime`: `20250618` - 69.1

`k8s.io/kubernetes/pkg/apis/core/validation` - `20250618` - 84.7

`k8s.io/kubernetes/pkg/scheduler/framework` - 20250618 - 71.7 

#### Integration tests

e2e tests provide good test coverage of interactions between the new pod-level
resource feature and existing Kubernetes components i.e. API server, kubelet, cgroup
enforcement. We may replicate and/or move some of the E2E tests functionality into 
integration tests before GA using data from any issues we uncover that are not
covered by planned and implemented tests.

#### e2e tests

  * - [Pod Level Resources Resize](https://github.com/kubernetes/kubernetes/blob/master/test/e2e/common/node/pod_level_resources_resize.go): [SIG Node](https://testgrid.k8s.io/sig-node-presubmits#pr-kubelet-e2e-podlevelresources-resize), [triage search](https://storage.googleapis.com/k8s-triage/index.html?ci=0&pr=1&sig=node&job=pr-kubelet-e2e-podlevelresources-resize)


Following scenarios need to be covered:

* Cgroup settings when pod-level resources are set.
* Validate scheduling and admission.
* Validate the containers with no limits set are throttled on CPU when CPU usage reaches Pod level CPU limits.
* Validate the containers with no limits set are OOMKilled when memory usage
  reaches Pod level memory limits.
* Test the correct values in TotalResourcesRequested.

### Graduation Criteria


#### Phase 1: Alpha (target 1.35) [DONE]
* Feature is disabled by default. It is an opt-in feature which can be enabled by
  enabling the InPlacePodLevelResourcesVerticalScaling feature gate and by setting
  the new resources fields in PodSpec at Pod level.
* In Place Pod-Level Resources Resize functionality consistent with In Place
  Container-Level Resource Resize functionality is implemented 
* Support the basic functionality for scheduler to consider pod-level resource requests to find a suitable node.
Support the basic functionality for kubelet to translate pod-level requests/limits to pod-level cgroup settings.
* Unit test coverage.
* E2E tests.

#### Phase 2:  Beta (target 1.36)

* Pod Level Resources Feature moved to beta.
* The semantic of `UpdatePodSandboxResources` is clarified. And there is a way for container runtime to reject the resize of Pod resources via this method or by other means
* Actual pod resource data may be cached in memory, which will be refreshed after
  each successful pod resize or for every cache-miss.
* Coverage for upgrade->downgrade->upgrade scenarios.
* Extend instrumentation from
  [KEP#1287](https://github.com/kubernetes/enhancements/blob/ef7e11d088086afd84d26c9249a4ca480df2d05a/keps/sig-node/1287-in-place-update-pod-resources/README.md)
  for Pod-level resource resize.
* Revisit the decision of which component sets the defaults for PodStatus.Resources.

#### GA (stable)

* VPA Integration of In-Place Resize moved to beta.
* No major bugs reported for 3 months.
* UpdatePodSandboxResources is implemented by containerd & CRI-O

### Upgrade / Downgrade Strategy

##### Upgrade
API Server and Scheduler should be upgraded before the kubelets in that order. 

The existing workloads that don't use pod-level resources won't be affected by
the downgrade. However, if a user wishes to modify or resize the new pod-level
resources for pods created before the cluster was upgraded and this feature became available, it will be necessary to drain all pods from the affected nodes.

##### Downgrade
Kubelets should be downgraded before Scheduler and API server. 

### Version Skew Strategy
InPlacePodLevelResourcesVerticalScaling is an opt-in feature. For this feature to
work correctly, it must be enabled in all parts of the cluster (scheduler, API
server, kubelet).

When the feature gate is disabled on control plane, but enabled on kubelet, users will not be able to resize Pods with the pod-level resources field in the spec.

When the feature gate is enabled on control plane, but disabled on kubelet, the API server might allow a user to update a pod's resource requests, even though the Kubelet on the node won't actually perform the in-place resize. This creates an inconsistent state: the API reflects the desired change, but the pod's resources on the node remain unchanged. 

Therefore, all cluster nodes (controlplane and worker nodes), must be upgraded before the user can deploy pods with the new resources field in the spec.

For users wanting to use this feature, it is user's responsibility to use the correct version of kube-scheduler, kube-apiserver and kubelet otherwise the feature will not work as expected.

## Production Readiness Review Questionnaire


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
  - Feature gate name: InPlacePodLevelResourcesVerticalScaling, PodLevelResources
  - Components depending on the feature gate: kubelet, kube-apiserver, kube-scheduler
  - Will enabling / disabling the feature require downtime of the control
    plane? No. Once the feature PodLevelResources is disabled, the control plane
    components reject the pods with pod-level resources, and when
    InPlacePodLevelResourcesVerticalScaling is disabled the control plane components
    will reject requests to resize the pod-level resources
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? No. Once the feature PodLevelResources is disabled, the kubelet
    rejects the pods with pod-level resources, and when
    InPlacePodLevelResourcesVerticalScaling is disabled the kubelet will reject
    requests to resize the pod-level resources

###### Does enabling the feature change any default behavior?

No. This feature is guarded by InPlacePodLevelResourcesVerticalScaling &
PodLevelResources feature gates, and requires setting Pod level `resource` stanza
explicitly. Existing default behavior does not change if the
feature is not used.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. 
* InPlacePodLevelResourcesVerticalScaling can be disabled without issue in the control plane.

* InPlacePodLevelResourcesVerticalScaling can be disabled on nodes, but if there are
  any pending resizes container resource configurations may be left in an unknown
  state. This can be avoided by draining the node before disabling in-place resize.

* InPlacePodLevelResourcesVerticalScaling can be disabled and reenabled without consequence.

###### What happens if we reenable the feature if it was previously rolled back?

API will once again permit modification of Resources for 'cpu' and 'memory' at pod-level.
Actual resources applied will be reflected in in Pod's PodStatus.

###### Are there any tests for feature enablement/disablement?

Yes, the tests will be added along with alpha implementation. 
* Unit tests verify that feature does not introduce any regression.
* E2E tests run against a local cluster verify that feature works as expected.

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

Since the feature is opt-in feature the rollouts won't likely fail because
of this feature. For the new workloads that want to use this feature, there could be
unexpected interactions with the existing features. This is why support for this
feature will be rolled out in phases to make sure all the cases and interaction with
existing features are covered before making it available in GA.

Rollbacks should be seamless if done after disabling the feature, and recreating 
the running workloads that use the feature.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

Any unusual observations in the following metrics should signal rollback:

* `kube_pod_container_resource_requests`: This metric exposes the resource
  requests (CPU and memory) for each container within a pod. 
* `kube_pod_container_resource_limits`: This metric exposes the resource limits
  (CPU and memory) for each container within a pod. 
* `node_collector_evictions_total`: to check if a pod level resource setting is
  causing to evict more pods than normal.
* `started_pods_errors_total`: exposed by kubelet to check if large number of
  pods are failing unusually.
* `started_containers_errors_total`: exposed by kubelet to check if large number.
  of containers are failing unusually
* `scheduler_pending_pods`: Number of pending pods on `unschedulable` queue to
  see number of pods scheduler failed to schedule. 
* `scheduler_pod_scheduling_attempts`: Any spiked in number of attempts to schedule pods 
* `scheduler_schedule_attempts_total`: Number of attempts to schedule pods.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->
Testing plan:
* Create test pod with pod-level resources
* Upgrade API server
* Attempt resize of test pod's pod-level resources
  * Expected outcome: resize is rejected (see version skew section for details)

* Create upgraded node
* Create second test pod, scheduled to upgraded node
* Attempt resize of second test pod
  * Expected outcome: resize successful
* Delete upgraded node

* Restart API server with feature disabled
* Ensure original test pod is still running
* Attempt resize of original test pod
* Expected outcome: request rejected by apiserver
* Restart API server with feature enabled
* Verify original test pod is still running

Initial manual verification was completed following the Alpha release ([Results](https://docs.google.com/document/d/19dKnTxH34YjSzrQCMqmpNp9iJk4YfWXXf5ytqNvV4c0/edit?usp=sharing)).
Comprehensive automated testing and E2E coverage are slated for implementation prior to GA graduation."

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

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

This feature will be built into kubelet, API server and scheduler. 
In order to determine if the feature is being used by the workloads, 
check the  `resources` field at pod level in the spec and
apiserver_request_total{resource=pods,subresource=resize} from metrics endpoint.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

* If the Kubelet supports InPlacePodLevelResourcesVerticalScaling, it will always
  set the Resources field in Pod status.
* The ResizeStatus in the pod status should converge to the empty value, indicating the resize has completed.
* The Resources in the pod and container statuses should converge to the resized resources, or an approximation of it.

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
Resize requests should succeed (apiserver_request_total{resource=pods,subresource=resize} with non-success code should be low)

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [X] Metrics
  - Metric name:
  - `apiserver_rejected_requests` will indicate any failures (`Bad Request` code=400) related to translation of new `resources` field in PodSpec. 
  - `schedule_attempts_total{result="error|unschedulable"}`
  - `node_collector_evictions_total`: to check if a pod level resource setting is causing to evict more pods than normal
  - `started_pods_errors_total`: exposed by kubelet to check if large number of pods are failing unusually
  - `started_containers_errors_total`: exposed by kubelet to check if large number of containers are failing unusually
  - Components exposing the metric: apiserver, kubelet, scheduler

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->
No 

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
Container runtime compatible with In-Place Pod Updates feature (see (CRI changes)[https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/1287-in-place-update-pod-resources/README.md#cri-changes] )

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
 - API call type (e.g. PATCH pods)
    - One new PATCH PodStatus API call in response to Pod resize request.
    - No additional overhead unless Pod resize is invoked.
  - estimated throughput
    - Proportional to the number of resize requests issued by users or controllers (e.g., VPA). For a typical cluster this is expected to be < 1% of total Pod update traffic.
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
    - Kubelet
  focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->
No

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
Negligible.
- API type(s):
- Estimated increase in size: (e.g., new annotation of size 32B): Each Pod object will grow by approximately
  200-400 bytes due to the addition of `Resources` and `AllocatedResources`
  fields in `PodStatus`, plus the `Resources` stanza in `PodSpec`.
- Estimated amount of new objects: (e.g., new Object X for every existing Pod)
  - type PodStatus has 2 new fields of type v1.ResourceRequirements and v1.ResourceList

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->
No. The computational overhead introduced in API server, scheduler and kubelet due to additional validation for pod level resource specifications should be negligible. Thorough testing and monitoring will ensure the SLIs/SLOs are not impacted.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?
No

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->
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

If the API server or etcd is unavailable, existing pods will continue to run with their last
known resource configurations. No new resize requests can be initiated, and the Kubelet
will be unable to update the `PodStatus` to reflect any locally completed or failed
resizes until connectivity is restored.

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

- **CRI Runtime doesn't support Pod Sandbox Resize**:
  - Detection: `PodStatus.Resize` will be stuck in `InProgress` and Kubelet logs will
    show errors calling `UpdatePodSandboxResources`.
  - Mitigations: Disable the feature gate or upgrade the container runtime to a
    compatible version (e.g., latest containerd/CRI-O).
  - Diagnostics: Kubelet logs (search for `UpdatePodSandboxResources` errors) and
    `kubectl get pod <name> -o yaml` to check `resizeStatus`.
- **Cgroup update failure (OS level)**:
  - Detection: Kubelet will emit an event indicating failure to update cgroups.
  - Mitigations: Revert the resize request in the Pod spec to a known-good value.
  - Diagnostics: Kubelet logs and `dmesg` on the node for potential OOM or cgroup
    permission issues.


###### What steps should be taken if SLOs are not being met to determine the problem?

1. Verify if the `InPlacePodLevelResourcesVerticalScaling` feature gate is enabled
   on all components (apiserver, scheduler, kubelet).
2. Check `apiserver_request_total{resource="pods", subresource="resize"}` to see
   if resize requests are being rejected at the API level.
3. Inspect Kubelet logs for errors related to `UpdatePodSandboxResources` or
  `ResourceCalculation`.
4. Monitor `node_collector_evictions_total` to ensure pod-level limits aren't
  causing unexpected evictions.

## Implementation History

- **2025-06-18:** KEP draft split from (KEP#2387)[https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/2837-pod-level-resource-spec/README.md]
- **2026-01-28:** KEP moved to beta for 1.36 release

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

Detailed in (KEP#1287)[https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/1287-in-place-update-pod-resources/README.md#alternatives]


<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Future Work

**Ephemeral containers with pod-level resources and IPPR**

Previously, assigning resources to ephemeral
containers wasn't allowed because pod resource allocations were immutable. With
the introduction of in-place pod resizing, users could gain more flexibility:

* Adjust pod-level resources to accommodate the needs of ephemeral containers. This
allows for a more dynamic allocation of resources within the pod.
* Specify resource requests and limits directly for ephemeral containers. Kubernetes will
then automatically resize the pod to ensure sufficient resources are available
for both regular and ephemeral containers.

Currently, setting `resources` for ephemeral containers is disallowed as pod
resource allocations were immutable before In-Place Pod Resizing feature. With
in-place pod resize for pod-level resource allocation, users should be able to
either modify the pod-level resources to accommodate ephemeral containers or
supply resources at container-level for ephemeral containers and kubernetes will
resize the pod to accommodate the ephemeral containers.