# KEP-6151: Pod-level Image Pull Duration Metric

## Summary

This KEP proposes adding a new structured field, `ImagePullDuration`, directly to the core `v1.PodStatus` API.

The Kubelet will calculate the precise net duration spent actively pulling images for a pod (properly accounting for overlapping and sequential container pulls). Once all images for the pod's containers are pulled, the Kubelet writes this calculated duration to the `ImagePullDuration` field in `PodStatus`. External observability systems can then watch for this field transitioning from nil to a non-nil value to retrieve the exact image pull latency directly through the Kubernetes API.

## Motivation

### Large Images in AI/LLM Workloads
In artificial intelligence and large language model (LLM) environments, container image sizes frequently exceed tens or hundreds of gigabytes. These massive images are the largest driver of pod startup latency. To optimize startup, solutions like image streaming (using remote snapshotters) are employed to start containers as soon as critical files are loaded.

### The Necessity of Pod-Level Metrics
Since the pod is the fundamental unit of deployment and scaling in Kubernetes, platform engineers and developers need a pod-level metric to track total image pulling time. Without a standardized, API-exposed pod-level milestone, it is difficult to isolate image loading bottlenecks from other startup phases (e.g., sandbox creation, volume mounts, init container runs) or to measure the real-world impact of streaming optimizations.

## Design Details

### API Schema Additions
We propose adding `ImagePullDuration` directly to `v1.PodStatus`.

#### Core API Types Modification (staging/src/k8s.io/api/core/v1/types.go)
```go
type PodStatus struct {
    ...
    // ImagePullDuration is the net duration spent pulling images for the pod's containers.
    // +optional
    ImagePullDuration *metav1.Duration `json:"imagePullDuration,omitempty" protobuf:"bytes,22,opt,name=imagePullDuration"`
}
```

### Kubelet Logic
The Kubelet's startup latency tracker tracks individual container image pull sessions. It merges overlapping sessions and excludes periods when no container is actively pulling an image. Once all image pulling for the pod is complete, it computes the final net active duration.

*   **Existing Support:** Kubelet's tracker already records sessions via `RecordImageStartedPulling` and `RecordImageFinishedPulling`, and provides `calculateImagePullingTime` to merge overlapping parallel sessions.
*   **Newly Added:**
    *   Expose the calculated duration through a new tracker method `GetImagePullDuration`.
    *   Integrate Kubelet's `generateAPIPodStatus` to populate this field upon pod status updates.

## Implementation Plan

1.  **API Schema Definition:**
    *   Add `ImagePullDuration` to external `v1.PodStatus` in `staging/src/k8s.io/api/core/v1/types.go`.
    *   Add `ImagePullDuration` to internal `PodStatus` in `pkg/apis/core/types.go`.
    *   Run codegen: `./hack/update-codegen.sh`.

2.  **Kubelet Core Tracking Logic (pkg/kubelet/util/pod_startup_latency_tracker.go):**
    *   Extend `podStartupLatencyTracker` interface with `GetImagePullDuration(pod *v1.Pod) *time.Duration`.

3.  **Status Generation (kubelet_pods.go):**
    *   In `generateAPIPodStatus`, retrieve and populate `PodStatus.ImagePullDuration`.

## Alternatives Considered

### Why not use ContainerStatus fields?
1.  **No Condition Arrays:** `ContainerStatus` does not have a generic condition list.
2.  **Lack of Fine-Grained Milestones:** `ContainerStatus.State` only represents Waiting, Running, or Terminated. Waiting only contains unstructured Reason and Message.
3.  **Ephemeral State:** The Waiting state is transient and deleted once the container runs.

### Why not use simple Pod Condition timestamp subtraction (T_end - T_start)?
1.  **The Sequential Pull Challenge:** If images are pulled sequentially with gaps (e.g., for container initialization), $T_{end} - T_{start}$ incorrectly includes those gaps.
2.  **API Semantics Violations:** Shifting timestamps backwards to match active duration violates core Kubernetes API conventions.