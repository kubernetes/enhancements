# KEP-6386: Ephemeral Storage Support for Pod-Level Resource Specifications

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Feature Gate](#feature-gate)
  - [PodSpec API Changes](#podspec-api-changes)
  - [PodSpec Validation &amp; Defaulting Rules](#podspec-validation--defaulting-rules)
  - [Scheduler Changes](#scheduler-changes)
  - [Eviction Manager](#eviction-manager)
  - [ResourceQuota &amp; LimitRanger](#resourcequota--limitranger)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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
  - [1. Reuse the existing <code>PodLevelResources</code> feature gate](#1-reuse-the-existing-podlevelresources-feature-gate)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [x] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

[KEP-2837: Pod Level Resource Specifications](/keps/sig-node/2837-pod-level-resource-spec/README.md) introduced the `resources` field at the `PodSpec` level (`pod.spec.resources`), allowing users to define aggregate compute resource requests and limits shared across all containers in a Pod. However, KEP-2837 initially scoped support strictly to CPU, memory, and `hugepages-*`, deferring `ephemeral-storage` to a future enhancement.

This KEP proposes extending Pod-Level Resource Specifications to support `ephemeral-storage` (`pod.spec.resources.requests[ephemeral-storage]` and `pod.spec.resources.limits[ephemeral-storage]`), gated by a new dedicated feature gate: `PodLevelResourcesEphemeralStorage`. This aligns ephemeral storage with KEP-2837 and allows multi-container Pods to manage a unified Pod-level ephemeral storage budget across container writable layers, container logs, and disk-backed `emptyDir` volumes.

## Motivation

Kubernetes workloads often consist of multiple containers collaborating within a Pod. While container-level resource specifications allow granular control per container, estimating and allocating `ephemeral-storage` for each individual container in a multi-container Pod is challenging:

1. **Pod-Scoped Shared Storage (`emptyDir`)**: Unlike CPU and memory, ephemeral storage includes disk-backed `emptyDir` volumes (`medium: ""`) that are defined at the Pod level and shared across multiple containers. With only container-level specifications, users cannot naturally attribute shared `emptyDir` storage to the Pod as a whole and are forced to arbitrarily assign or split the storage request and limit across individual containers.
2. **Avoiding Unintended Whole-Pod Limits from Injected Sidecars**: Kubelet's Pod-level ephemeral storage eviction (`podEphemeralStorageLimitEviction`) calculates a Pod's total ephemeral storage limit by summing the declared container-level limits across the Pod. If an injected sidecar declares an `ephemeral-storage` limit while the primary application containers omit it, the sidecar's limit unintentionally becomes the effective ephemeral storage limit for the entire Pod (bounding the main containers and any shared `emptyDir` volumes). Allowing `ephemeral-storage` to be specified directly at the Pod level avoids this pitfall.
3. **Shared Pod-Level Limit Without Per-Container Limit Over-Allocation**: Kubelet enforces container-level `ephemeral-storage` limits against each individual container's writable layer and logs (`containerEphemeralStorageLimitEviction`), in addition to enforcing the sum of container limits against the Pod's total usage. When multiple containers experience staggered peaks in ephemeral storage usage (e.g., two containers each peaking at `10Gi` at different times while combined usage never exceeds `15Gi`), setting `10Gi` container-level limits on each container results in an aggregated `20Gi` Pod limit (and a `20Gi` defaulted Pod request if requests are not separately specified), whereas a Pod-level specification allows bounding and reserving `15Gi` for the Pod as a whole.

Supporting `ephemeral-storage` requests and limits at the Pod level (`pod.spec.resources`) complements existing container-level settings by allowing containers within a Pod to share a unified ephemeral storage budget while bounding the Pod's total disk consumption.

### Goals

1. Relax Pod API validation and update defaulting to allow specifying `ephemeral-storage` requests and limits at the Pod level (`pod.spec.resources`), gated by the `PodLevelResourcesEphemeralStorage` feature gate.
2. Update Pod resource helpers (`k8s.io/component-helpers/resource` and `pkg/apis/core/helper`) and `LimitRanger` to support Pod-level `ephemeral-storage` alongside existing container-level `ephemeral-storage` specifications, `emptyDir.sizeLimit`, and `ResourceQuota`.
3. Wire `PodLevelResourcesEphemeralStorage` into `kube-scheduler` and Kubelet's eviction manager (via `resourcehelper.PodRequests` and `resourcehelper.PodLimits`) so Pod-level `ephemeral-storage` requests and limits are accounted for and enforced.

### Non-Goals

1. **No Changes to Ephemeral Storage Accounting**: Any changes to how `ephemeral-storage` usage is measured or which storage types are accounted against it (container writable layers, container logs, and disk-backed `emptyDir` volumes; excluding memory-backed `tmpfs` `emptyDir` volumes, PVCs, `hostPath`, and CSI volumes) are out of scope.
2. **No In-Place Resizing of `ephemeral-storage`**: In-place resizing of container-level or Pod-level `ephemeral-storage` requests and limits at runtime is out of scope.
3. **No Changes to Pod QoS Class Calculation**: As with container-level `ephemeral-storage`, Pod-level `ephemeral-storage` does not participate in Pod QoS class determination (`Guaranteed`, `Burstable`, `BestEffort`).

## Proposal

When the `PodLevelResourcesEphemeralStorage` and `PodLevelResources` feature gates are enabled, users can specify `ephemeral-storage` in `pod.spec.resources.requests` and `pod.spec.resources.limits`.

### User Stories

#### Story 1

**Multi-Container Pod Sharing Disk-Backed `emptyDir`**: A data processing Pod uses an `initContainer` (`data-fetcher`) to download raw dataset files into a shared disk-backed `emptyDir` volume, a main container (`processor`) that transforms the data and writes intermediate scratch files to its writable layer, and a sidecar (`log-shipper`) that processes logs.

Instead of arbitrarily inflating one container's ephemeral storage request to account for the shared `emptyDir`, the user specifies `ephemeral-storage` directly at the Pod level:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: data-pipeline-pod
spec:
  resources:
    requests:
      ephemeral-storage: "10Gi"
    limits:
      ephemeral-storage: "20Gi"
  volumes:
    - name: shared-data
      emptyDir: {}
  initContainers:
    - name: data-fetcher
      image: fetcher:latest
      volumeMounts:
        - name: shared-data
          mountPath: /data
  containers:
    - name: processor
      image: processor:latest
      volumeMounts:
        - name: shared-data
          mountPath: /data
    - name: log-shipper
      image: shipper:latest
      resources:
        limits:
          ephemeral-storage: "2Gi" # Optional per-container ceiling for shipper's rootfs + logs
```

Here, `kube-scheduler` reserves `10Gi` of `ephemeral-storage` on the node for the entire Pod. At runtime, Kubelet ensures that the aggregate disk usage of all container writable layers, container logs, and the `shared-data` `emptyDir` volume does not exceed `20Gi`, while also enforcing that `log-shipper`'s individual writable layer + logs do not exceed `2Gi`.

#### Story 2

**Bounding Volume Spikes with Individual `emptyDir.sizeLimit` and Pod Limit**: A Pod mounts two disk-backed `emptyDir` volumes (`cache-dir` with `sizeLimit: 5Gi` and `scratch-dir` without `sizeLimit`) and sets `pod.spec.resources.limits[ephemeral-storage]: 8Gi`. Kubelet enforces that `cache-dir` never exceeds `5Gi`, while total Pod ephemeral storage usage (both volumes + container rootfs layers + logs) never exceeds `8Gi`.

### Notes/Constraints/Caveats (Optional)

* **Storage Types Governed**:
  Pod-level `ephemeral-storage` governs:
  1. Container writable layers (e.g., `overlayfs` upperdir).
  2. Container `stdout`/`stderr` logs (`/var/log/pods/<pod-uid>/...`).
  3. Disk-backed `emptyDir` volumes (`medium: ""` on local disk).
  It explicitly excludes memory-backed `emptyDir` volumes (`medium: Memory`) and CSI/PVC persistent volumes.

### Risks and Mitigations

1. **Opt-In Rollout Safety**: Since Pod-level `ephemeral-storage` is an opt-in feature guarded by `PodLevelResourcesEphemeralStorage`, merging and enabling the feature does not affect existing workloads that specify container-level resources.
2. **Third-Party Tools and Custom Controllers**: External tools, custom schedulers, or monitoring agents that calculate Pod ephemeral storage requirements by summing `pod.spec.containers[*].resources` directly (instead of using `pod.spec.resources` or `k8s.io/component-helpers/resource`) may underestimate a Pod's ephemeral storage reservation or limit when only Pod-level resources are specified.
   - *Mitigation*: Update `k8s.io/component-helpers/resource` (`PodRequests` and `PodLimits`) so ecosystem components consuming standard Kubernetes helpers automatically support Pod-level `ephemeral-storage`, and clearly document the behavior in release notes.
3. **Asynchronous Eviction Polling Latency**: Kubelet enforces `pod.spec.resources.limits[ephemeral-storage]` asynchronously via its eviction manager polling loop (`housekeeping-interval`), allowing brief bursts above the limit before eviction.
   - *Mitigation*: This matches existing Kubernetes container-level `ephemeral-storage` limit enforcement behavior.

## Design Details

### Feature Gate

A new feature gate `PodLevelResourcesEphemeralStorage` will be introduced across `kube-apiserver`, `kube-scheduler`, and `kubelet`:

* **Name**: `PodLevelResourcesEphemeralStorage`
* **Default**: `false` (Alpha)
* **Stage**: Alpha in v1.38
* **Dependencies**: `PodLevelResources` (GA in v1.37) and `LocalStorageCapacityIsolation` (GA in v1.25).

### PodSpec API Changes

With `PodLevelResourcesEphemeralStorage` enabled, `ephemeral-storage` is supported under `pod.spec.resources.requests` and `pod.spec.resources.limits`:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: example-pod
spec:
  resources:
    requests:
      ephemeral-storage: "2Gi"
    limits:
      ephemeral-storage: "5Gi"
```

### PodSpec Validation & Defaulting Rules

Pod-level `ephemeral-storage` follows the same validation, defaulting, and precedence rules defined in [KEP-2837](/keps/sig-node/2837-pod-level-resource-spec/README.md#proposed-validation--defaulting-rules) for CPU and memory:

* **Pod-level and Container-level Coexistence**:
  - When Pod-level `ephemeral-storage` requests or limits are set, Pod-level resource calculations (`PodRequests` and `PodLimits`, used for scheduling, `ResourceQuota`, disk-pressure eviction ranking, and total Pod limit eviction) use the Pod-level value directly instead of summing container-level values.
  - At runtime, each container is constrained to the lesser of the Pod-level limit and its container-level limit: if a container specifies a container-level `ephemeral-storage` limit, Kubelet continues to enforce that limit against the container's individual writable layer and logs (`containerEphemeralStorageLimitEviction`), while enforcing the Pod-level limit against the aggregate ephemeral storage usage of the Pod (`podEphemeralStorageLimitEviction`).
* **Validation Rules**:
  - The aggregated container-level `ephemeral-storage` requests cannot exceed the Pod-level request or limit.
  - The sum of container-level `ephemeral-storage` limits may exceed the Pod-level limit, but no single container-level limit can exceed the Pod-level limit.
* **Defaulting Rules**:
  - Container-level and Pod-level defaulting for `ephemeral-storage` follow the exact same rules as CPU and memory defined in [KEP-2837](/keps/sig-node/2837-pod-level-resource-spec/README.md#proposed-validation--defaulting-rules).
  - `emptyDir.sizeLimit` does not participate in Pod-level `ephemeral-storage` defaulting; it remains strictly a per-volume limit enforced at runtime.

### Scheduler Changes

`kube-scheduler` already computes pod resource requests via `resourcehelper.PodRequests`. Once `ephemeral-storage` is added to the supported pod-level resources in `resourcehelper` (gated by `PodLevelResourcesEphemeralStorage`), `kube-scheduler` will automatically use `pod.spec.resources.requests[ephemeral-storage]` when specified and fall back to aggregated container requests otherwise.

### Eviction Manager

Like container-level `ephemeral-storage` today, pod-level `ephemeral-storage` is enforced reactively by Kubelet's eviction manager during its periodic housekeeping loop (rather than via proactive kernel/cgroup write limits). For ephemeral storage eviction and disk pressure signals, the eviction manager already uses `resourcehelper.PodLimits` and `resourcehelper.PodRequests` to compare the pod's ephemeral storage usage against its effective requests and limits: when pod-level `ephemeral-storage` requests or limits are specified, it uses the pod-level values directly instead of aggregating container-level values; otherwise, it falls back to aggregating container-level values.

### ResourceQuota & LimitRanger

* **`ResourceQuota`**: Namespace `requests.ephemeral-storage` and `limits.ephemeral-storage` quotas evaluate pods using their pod-level `ephemeral-storage` requests and limits when specified, falling back to aggregated container requests and limits when unspecified.
* **`LimitRanger`**: For `LimitRange` objects with `type: Pod`, `LimitRanger` validates that pod-level `ephemeral-storage` requests and limits satisfy configured `min`, `max`, and `maxLimitRequestRatio` bounds, and applies pod-level `default` and `defaultRequest` values when unspecified.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None required prior to implementation.

##### Unit tests

Unit tests will be added/updated in the following packages:

- `pkg/apis/core/validation`: Verify validation rules for `pod.spec.resources.requests[ephemeral-storage]` and `limits[ephemeral-storage]` (request $\le$ limit, aggregated container requests $\le$ pod request, individual container limits $\le$ pod limit, and allowing aggregated container limits $>$ pod limit).
- `pkg/apis/core/v1`: Verify defaulting rules for pod-level `ephemeral-storage`.
- `staging/src/k8s.io/component-helpers/resource`: Verify `PodRequests` and `PodLimits` resolve pod-level and container-level `ephemeral-storage` requests and limits.
- `pkg/kubelet/eviction`:
  - Verify pod eviction and event message (`Evicted: Pod ephemeral local storage usage (...) exceeds pod-level limit (...)`) when total pod usage exceeds `pod.spec.resources.limits[ephemeral-storage]`.
  - Verify eviction precedence between `emptyDir.sizeLimit`, container limit, and pod limit.
  - Verify disk pressure eviction ranking uses pod-level `ephemeral-storage` requests when specified.
- `pkg/scheduler/framework/plugins/noderesources`: Verify `NodeResourcesFit` Filter and Score plugins with pod-level `ephemeral-storage`.
- `pkg/quota/v1/evaluator/core` and `plugin/pkg/admission/limitranger`: Verify `ResourceQuota` and `LimitRanger` with pod-level `ephemeral-storage`.

Current unit test coverage of key touched packages:
- `k8s.io/kubernetes/pkg/apis/core/validation`: `2026-09-17` - `~84%`
- `k8s.io/kubernetes/pkg/kubelet/eviction`: `2026-09-17` - `~76%`
- `k8s.io/component-helpers/resource`: `2026-09-17` - `~92%`

##### Integration tests

Integration tests will be added to `test/integration`:
- `test/integration/scheduler`: Verify Pod scheduling with `pod.spec.resources.requests[ephemeral-storage]` across nodes with different allocatable ephemeral storage capacities.
- `test/integration/quota`: Verify namespace `ResourceQuota` admission and usage tracking for `requests.ephemeral-storage` and `limits.ephemeral-storage` with Pod-level specifications.

##### e2e tests

Node e2e and cluster e2e tests will be added under `test/e2e_node/` and `test/e2e/common/node/`:
- **Pod-Level Ephemeral Storage Limit Eviction (`test/e2e_node/eviction_test.go`)**:
  - Launch a multi-container Pod with `pod.spec.resources.limits[ephemeral-storage]: 100Mi` and a shared disk-backed `emptyDir`.
  - Write `60Mi` to container writable layer/logs and `50Mi` to `emptyDir` (total `110Mi` > `100Mi`).
  - Verify Kubelet evicts the Pod with message `Evicted: Pod ephemeral local storage usage (...) exceeds pod-level limit (100Mi)`.
- **Eviction Precedence with `emptyDir.sizeLimit` (`test/e2e_node/eviction_test.go`)**:
  - Launch a Pod with `pod.spec.resources.limits[ephemeral-storage]: 200Mi` and an `emptyDir` with `sizeLimit: 50Mi`.
  - Write `60Mi` to the `emptyDir` and verify eviction triggers specifically for breaching `emptyDir.sizeLimit` (`50Mi`).
- **Disk Pressure Eviction Priority (`test/e2e_node/eviction_test.go`)**:
  - Under simulated node disk pressure, verify a Pod whose usage is below `pod.spec.resources.requests[ephemeral-storage]` is protected relative to a Pod exceeding its Pod-level request.
- **Scheduling Fit e2e (`test/e2e/common/node/pod_level_resources.go`)**:
  - Verify Pods requesting `pod.spec.resources.requests[ephemeral-storage]` exceeding node allocatable capacity remain `Pending` with `Insufficient ephemeral-storage`.

### Graduation Criteria

#### Alpha

- Feature implemented behind `PodLevelResourcesEphemeralStorage` feature gate (disabled by default).
- API validation, defaulting, and resource helpers implemented and unit-tested.
- `kube-scheduler`, Kubelet admission, Kubelet eviction manager (including clear eviction messaging and `emptyDir.sizeLimit` precedence), `LimitRanger`, and `ResourceQuota` updated.
- Unit, integration, and node e2e tests passing in CI.

#### Beta

- `PodLevelResourcesEphemeralStorage` enabled by default.
- Cluster Autoscaler and Vertical Pod Autoscaler (VPA) compatibility verified.
- All e2e tests running flake-free in TestGrid.

#### GA

- `PodLevelResourcesEphemeralStorage` graduated to GA (`stable`) and locked to `true`.
- At least two minor releases in Beta with positive operational feedback and zero critical bugs.
- Conformance and e2e tests stable with a minimum two-week flake-free window.

#### Deprecation

N/A.

### Upgrade / Downgrade Strategy

* **Upgrade**:
  - Existing Pods without `pod.spec.resources.requests/limits[ephemeral-storage]` experience no behavioral change.
  - Once `kube-apiserver`, `kube-scheduler`, and `kubelet` instances have `PodLevelResourcesEphemeralStorage` enabled, users can submit Pods with Pod-level `ephemeral-storage`.
* **Downgrade**:
  - If `PodLevelResourcesEphemeralStorage` is disabled on `kube-apiserver`, new Pod creations or updates specifying Pod-level `ephemeral-storage` are rejected by validation.
  - Already running Pods persisted in etcd continue to run; if Kubelet has the feature gate disabled, Kubelet falls back to container-level `ephemeral-storage` requests and limits.

### Version Skew Strategy

Pod-level `ephemeral-storage` specification is an opt-in feature. For this feature to work correctly, `PodLevelResourcesEphemeralStorage` must be enabled in all parts of the cluster (`kube-scheduler`, `kube-apiserver`, `kubelet`).

When the feature gate is disabled on the control plane, but enabled on `kubelet`, users will not be able to create Pods with `ephemeral-storage` in `pod.spec.resources`.

When the feature gate is enabled on the control plane, but disabled on `kubelet`, `NodeDeclaredFeatures` ([KEP-5328](/keps/sig-node/5328-node-declared-features/README.md)) ensures `kube-scheduler` only schedules Pods with Pod-level `ephemeral-storage` onto nodes that declare `PodLevelResourcesEphemeralStorage` in `node.status.declaredFeatures`, and `kubelet` will reject those Pods at admission if scheduled onto a node where the feature gate is disabled.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `PodLevelResourcesEphemeralStorage`
  - Components depending on the feature gate: `kubelet`, `kube-apiserver`, `kube-scheduler`
  - Will enabling / disabling the feature require downtime of the control plane? No. Once the feature is disabled, control plane components (`kube-apiserver`) reject new Pod creations/updates containing `ephemeral-storage` in `pod.spec.resources`.
  - Will enabling / disabling the feature require downtime or reprovisioning of a node? No. Once the feature is disabled on `kubelet`, Kubelet rejects new Pods with Pod-level `ephemeral-storage` at admission and falls back to container-level `ephemeral-storage` accounting for already admitted Pods.
  - Note: Requires `PodLevelResources` (GA in v1.37) to be enabled.

###### Does enabling the feature change any default behavior?

For Pods that do not specify `pod.spec.resources`, enabling `PodLevelResourcesEphemeralStorage` does not change any default behavior.

For Pods that specify `pod.spec.resources` (e.g., for `cpu` or `memory`) and also specify container-level `ephemeral-storage`, enabling `PodLevelResourcesEphemeralStorage` will cause `DefaultPodLevelResources` to also default `ephemeral-storage` in `pod.spec.resources` (`requests[ephemeral-storage]` to the aggregated container requests, and `limits[ephemeral-storage]` to the aggregated container limits if all containers specify an `ephemeral-storage` limit). Because the defaulted Pod-level values match the aggregated container values, scheduling and runtime eviction behavior for these Pods remains unchanged.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes (`disable-supported: true`). When the feature gate is disabled:
* New Pods are not permitted to specify `ephemeral-storage` in `pod.spec.resources`.
* For existing Pods that were created with Pod-level `ephemeral-storage`, `resourcehelper.PodRequests` and `resourcehelper.PodLimits` ignore `ephemeral-storage` in `pod.spec.resources` and only consider container-level `ephemeral-storage` specifications.

###### What happens if we reenable the feature if it was previously rolled back?

If `PodLevelResourcesEphemeralStorage` is re-enabled after being previously disabled:
* Any new Pods will again be able to specify `ephemeral-storage` in `pod.spec.resources`.
* For existing Pods with Pod-level `ephemeral-storage`, `resourcehelper.PodRequests` and `resourcehelper.PodLimits` will immediately resume using `pod.spec.resources` for `ephemeral-storage` across `ResourceQuota`, `kube-scheduler`, and `kubelet`'s Eviction Manager.

###### Are there any tests for feature enablement/disablement?

Yes, unit and integration tests cover feature gate toggling:
* Validate that users can set `ephemeral-storage` in `pod.spec.resources` only when `PodLevelResourcesEphemeralStorage` is enabled.
* Validate that `kube-scheduler` (`NodeResourcesFit`) and `k8s.io/component-helpers/resource` consider Pod-level `ephemeral-storage` when the gate is enabled, and fall back to container-level aggregation when the gate is disabled.
* Validate that Kubelet's Eviction Manager enforces `pod.spec.resources.limits[ephemeral-storage]` when enabled, and reverts to container-level limits when disabled.
* Confirm that `LimitRanger` applies Pod-level `defaultRequest` and `default` for `ephemeral-storage` when enabled, and ignores them when disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollouts and rollbacks do not impact existing workloads. During mixed-version rollouts, `NodeDeclaredFeatures` ([KEP-5328](/keps/sig-node/5328-node-declared-features/README.md)) ensures Pods using Pod-level `ephemeral-storage` are only scheduled onto nodes with the feature enabled.

###### What specific metrics should inform a rollback?

Any unusual observations in the following metrics should signal a rollback:
* `kubelet_evictions{eviction_signal="ephemeralpodfs.limit"}`: Unexpected spikes in Pod-level ephemeral storage limit evictions after enabling the feature (alongside `ephemeralcontainerfs.limit` and `emptydirfs.limit`).
* `node_collector_evictions_total`: Node-level eviction anomalies.
* `started_pods_errors_total`: Exposed by Kubelet to detect unusual spikes in Pod startup failures.
* `started_containers_errors_total`: Exposed by Kubelet to detect unusual spikes in container startup failures.
* `scheduler_pending_pods`: Number of pending Pods in the `unschedulable` queue to detect scheduling regressions.
* `schedule_attempts_total{result="error|unschedulable"}`: Spikes in scheduling errors or unschedulable attempts.
* `apiserver_rejected_requests`: Unexpected spikes in API validation rejections.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Testing plan for Beta (`upgrade -> downgrade -> upgrade`):
- **Initial State**: Cluster running previous minor version with `PodLevelResourcesEphemeralStorage` disabled. Create a baseline Pod with container-level `ephemeral-storage`.
- **Upgrade Phase**:
  - Upgrade control plane (`kube-apiserver`, `kube-scheduler`) and enable `PodLevelResourcesEphemeralStorage`.
  - Verify existing baseline Pods continue running without issue.
  - Create a test Pod with `pod.spec.resources.limits[ephemeral-storage]`. Verify API server admits it.
  - Upgrade node (`kubelet`) and enable `PodLevelResourcesEphemeralStorage`.
  - Verify Kubelet enforces the Pod-level ephemeral storage limit when total usage (writable layers + logs + `emptyDir`) exceeds the limit.
- **Downgrade Phase**:
  - Disable `PodLevelResourcesEphemeralStorage` on `kubelet` and control plane components.
  - Verify existing running Pods continue running without crashing.
  - Attempt to create a new Pod with `pod.spec.resources.requests[ephemeral-storage]`; verify `kube-apiserver` rejects the creation.
- **Re-upgrade Phase**:
  - Re-enable `PodLevelResourcesEphemeralStorage` on control plane and `kubelet`.
  - Verify existing persisted Pods resume Pod-level ephemeral storage enforcement and new Pods with Pod-level `ephemeral-storage` are admitted and enforced properly.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Operators can use:
* `kubelet_pod_level_resources_admission_total`: Tracks Kubelet admission of Pods using Pod-level resources, categorized by resource configuration strategy (`config_mode`), admission status, and QoS class.
* `kube_pod_level_resource_spec{resource="ephemeral_storage"}` (from `kube-state-metrics`): Identifies specific Pods specifying `ephemeral-storage` at the Pod level and their configured requests/limits.

###### How can someone using this feature know that it is working for their instance?

- [X] Events
  - Event Reason: `Evicted` with message `Evicted: Pod ephemeral local storage usage (<usedQuantity>) exceeds pod-level limit (<limitQuantity>)`.
- [X] API .status
  - Other field: `pod.status.phase == "Failed"`, `pod.status.reason == "Evicted"`, and `pod.status.message` detailing the Pod-level ephemeral storage limit breach.
- [X] Other (treat as last resort)
  - Details: Kubelet `/stats/summary` API (`ephemeral-storage` Pod-level stats: `usedBytes`, `capacityBytes`, `inodesUsed`).

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

* Pod admission and scheduling latency SLOs remain unchanged from standard Kubernetes SLOs.
* Overall node ephemeral storage utilization remains bounded within configured Pod ceilings without unexpected node disk exhaustion.
* Eviction detection latency: Pods breaching `pod.spec.resources.limits[ephemeral-storage]` are evicted within 1–2 Kubelet housekeeping intervals (default `10s`–`20s`).

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - Metric name:
    - `kubelet_pod_level_resources_admission_total`: Total Pods processed during Kubelet admission using Pod-level resources.
    - `kubelet_evictions{eviction_signal="ephemeralpodfs.limit"}`: Total Pods evicted due to exceeding their Pod-level ephemeral storage limit.
    - `kube_pod_level_resource_spec` (`kube-state-metrics`): Exposes configured numeric values of Pod-level resource requests/limits.
    - `schedule_attempts_total{result="error|unschedulable"}`: Scheduler health and fit metrics.
    - `node_collector_evictions_total`: Node eviction tracking.
    - `started_pods_errors_total` / `started_containers_errors_total`: Pod and container startup error rates.
  - Components exposing the metric: `kube-apiserver`, `kubelet`, `kube-scheduler`, `kube-state-metrics`.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No external services are required. The feature depends on internal feature gates:
- `PodLevelResources` (GA in v1.37).
- `LocalStorageCapacityIsolation` (GA in v1.25).

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. It only modifies existing API request/response payloads.

###### Will enabling / using this feature result in introducing new API types?

No. It extends the supported resource keys within the existing `pod.spec.resources` (`v1.ResourceRequirements`) field introduced by KEP-2837.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Negligible.
- API type(s): `v1.Pod` (`v1.ResourceRequirements` and `v1.ResourceList`)
- Estimated increase in size: ~54B increase per Pod when both request and limit are specified:
  - `v1.ResourceRequirements` with `ephemeral-storage` key (`17B`) and quantity value (`~10B`) in `requests` and `limits`: `2 * (17B + 10B) = ~54B` per Pod.
- Estimated amount of new objects: 0 new API objects.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. The computational overhead in `kube-apiserver` validation, `kube-scheduler` fit evaluation, and `kubelet` eviction checking is negligible ($O(1)$ map lookups and comparisons on already-collected filesystem stats).

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

Yes, indirectly via increased Pod density.

Allowing containers within a Pod to share a single Pod-level `ephemeral-storage` pool reduces over-provisioning compared to per-container requests, which can allow `kube-scheduler` to pack more Pods onto a node. Higher Pod density can increase consumption of node resources such as inodes (`nodefs.inodesFree`), PIDs, and network sockets.

This is mitigated by:
1. Kubelet's `maxPods` configuration limiting total Pods per node.
2. Kubelet's existing inode eviction monitoring (`nodefs.inodesFree` and `imagefs.inodesFree` eviction signals).

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Kubelet enforces `pod.spec.resources.limits[ephemeral-storage]` locally using its cached Pod state and local filesystem usage observations (`cadvisor` / CRI stats). If `kube-apiserver` or `etcd` is unavailable, Kubelet still detects Pod-level ephemeral storage limit breaches and kills the offending Pod's containers locally to reclaim disk space and protect node health. The resulting `Evicted` Pod status update is queued and synced once API server connectivity recovers.

###### What are other known failure modes?

- **Pod evicted due to exceeding Pod-level ephemeral storage limit**:
  - *Detection*: `kubelet_evictions{eviction_signal="ephemeralpodfs.limit"}` increments; Pod status transitions to `Failed` (`reason: Evicted`) with message `Evicted: Pod ephemeral local storage usage (<usedQuantity>) exceeds pod-level limit (<limitQuantity>)`.
  - *Mitigations*: Query Kubelet `/stats/summary` to inspect which container writable layer, container log directory, or disk-backed `emptyDir` volume consumed the budget. Increase `pod.spec.resources.limits[ephemeral-storage]`, add log rotation, or set an explicit `emptyDir.sizeLimit` / container limit on noisy sidecars.
  - *Diagnostics*: Kubelet logs at verbosity `V(2)` log detailed per-container and per-volume byte usage when an eviction threshold is triggered.
  - *Testing*: Covered by node e2e eviction tests (`test/e2e_node/eviction_test.go`).

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Inspect Kubelet logs (`journalctl -u kubelet`) for errors in `eviction_manager.go` or `cadvisor` filesystem stats collection delays.
2. Check Kubelet's `housekeeping-interval` flag if asynchronous eviction latency is higher than expected.
3. If an issue specific to `PodLevelResourcesEphemeralStorage` causes regressions, disable the `PodLevelResourcesEphemeralStorage` feature gate on `kube-apiserver`, `kube-scheduler`, and `kubelet` to roll back to container-level ephemeral storage handling.

## Implementation History

- **2026-09-17**: Initial KEP draft created proposing `PodLevelResourcesEphemeralStorage` for Alpha in v1.38.

## Drawbacks

- Enforcement of `pod.spec.resources.limits[ephemeral-storage]` is asynchronous (via Kubelet's eviction manager housekeeping loop) rather than synchronous kernel write blocking (`ENOSPC`). This matches existing Kubernetes container-level `ephemeral-storage` behavior.

## Alternatives

### 1. Reuse the existing `PodLevelResources` feature gate

We considered adding `ephemeral-storage` support directly under the existing `PodLevelResources` feature gate (KEP-2837).
* **Why rejected**: `PodLevelResources` is graduating to GA (`stable`) in v1.37. Introducing a new resource type with distinct Kubelet eviction manager semantics requires a separate feature gate (`PodLevelResourcesEphemeralStorage`) so it can progress through Alpha $\rightarrow$ Beta $\rightarrow$ GA independently without delaying KEP-2837.

## Infrastructure Needed (Optional)

None beyond standard Kubernetes CI / TestGrid infrastructure (`sig-node-presubmits` and node e2e test suites).
