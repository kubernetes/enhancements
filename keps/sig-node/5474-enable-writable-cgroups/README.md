
<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: Container-in-Container Development](#story-1-container-in-container-development)
    - [Story 2: Dynamic Resource Management](#story-2-dynamic-resource-management)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [cpuset Isolation](#cpuset-isolation)
  - [Verified Behavior](#verified-behavior)
    - [Descendant Cgroup Memory Accounting](#descendant-cgroup-memory-accounting)
    - [Kubelet Stats Collection](#kubelet-stats-collection)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
    - [Core API Types](#core-api-types)
    - [CRI API Changes](#cri-api-changes)
      - [Descendant and Depth Limits (Pod-level, no CRI changes)](#descendant-and-depth-limits-pod-level-no-cri-changes)
  - [Implementation Details](#implementation-details)
    - [CgroupOptions Implementation Flow](#cgroupoptions-implementation-flow)
    - [Validation](#validation)
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
  - [Alternative 1: Runtime-specific Annotations](#alternative-1-runtime-specific-annotations)
  - [Alternative 2: Boolean Field Instead of Struct](#alternative-2-boolean-field-instead-of-struct)
  - [Alternative 3: Runtime Auto-Detection (Implicit Behavior)](#alternative-3-runtime-auto-detection-implicit-behavior)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist


Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes adding a `CgroupOptions` struct field to the container SecurityContext in Kubernetes to allow unprivileged containers to have writable access to cgroup interfaces on cgroup v2 systems. This feature relies on the kernel's `nsdelegate` mount option to ensure containers can only manage their own cgroup subtrees, preventing unauthorized access to system resources. The explicit API field is required to provide visibility, policy enforcement, and defense-in-depth for this capability.

## Motivation

 With cgroup v2's secure delegation model, unprivileged containers can safely manage their own cgroup subtree without compromising system security. To support this, a configuration option can be introduced to ensure that, when cgroup v2 is enabled, the cgroup interface (/sys/fs/cgroup) is mounted with read-write permissions for containers.

By exposing the `CgroupOptions` field through the Kubernetes API and CRI interface, container runtimes can be updated to honor the setting via CRI, enabling unprivileged containers to take advantage of writable cgroups in a secure manner.

While the `nsdelegate` mount option makes this safe from a kernel isolation perspective, it represents a significant change in the container's capabilities. Requiring an explicit opt-in via the API ensures this capability is visible to cluster administrators, can be restricted via policy (e.g., Pod Security Standards), and maintains the principle of defense-in-depth by keeping the default secure and restricted.

Related Issues:
- https://github.com/containerd/containerd/issues/10924
- https://github.com/kubernetes/kubernetes/issues/121190

### Goals

- Add a `CgroupOptions` struct field to the container SecurityContext object for extensibility
- CRI API changes
- Integrate with Pod Security Standards to ensure appropriate security policy enforcement
- Container runtime modifications
- Maintain backward compatibility with existing workloads

## Proposal

Add a new `CgroupOptions` struct field to the `SecurityContext` in the core v1 API. This struct-based approach provides extensibility for future cgroup-related configurations. When `mountMode` is set to `Writable` on cgroup v2 systems, it instructs the container runtime to mount the cgroup filesystem with write permissions for the container's own cgroup subtree.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: writable-cgroups-example
spec:
  containers:
  - name: app
    image: myapp:latest
    securityContext:
      cgroupOptions:
        mountMode: Writable
```

### User Stories (Optional)

#### Story 1: Container-in-Container Development

As a developer, I need to run Docker-in-Docker for testing and development purposes. Currently, I need to use privileged containers which exposes unnecessary security risks. With `CgroupOptions.MountMode: Writable`, I can run these nested containers securely while still allowing them to manage their own resource constraints.

#### Story 2: Dynamic Resource Management

As a developer, I might want to create sub-cgroups for finer granularity and concise control over the different in-pod processes. For instance, distributed AI/ML frameworks like Ray can create sub-cgroups for each worker process and dynamically adjust CPU and memory limits based on workload patterns, enabling better resource utilization and isolation without requiring privileged access.
Another example is, [KubeVirt](https://github.com/kubevirt/kubevirt) runs a hypervisor and a guest VM in the same unprivileged pod which sometimes demands to divide the resources between the management layer (hypervisor + management processes) and the guest layer (e.g. vCPU processes). Currently the management of sub-cgroups is done by a privileged component in a somewhat hacky way, but with writable cgroups the unprivileged pod itself could create and manage these cgroups.


### Notes/Constraints/Caveats (Optional)

- **cgroup v2 Only**: This feature requires cgroup v2
- **Linux Only**: The field is only valid on Linux containers and will be validated accordingly
- **Runtime Support**: Requires container runtime support
- **Node Configuration**: The host's cgroup v2 filesystem must be mounted with `nsdelegate`, and the kubelet must manage a cgroup per Pod (`--cgroups-per-qos`).
- **Descendant Limits**: When `mountMode: Writable` is enabled, the kubelet applies descendant and depth limits to the Pod cgroup before any container starts; see [Descendant and Depth Limits](#descendant-and-depth-limits-pod-level-no-cri-changes).
- **Security Context Integration**: Must work cohesively with other SecurityContext fields

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| **Security Bypass**: Containers gaining unauthorized access to system cgroups | Only allow write access to container's own cgroup subtree. cgroup v2 delegation model provides isolation |
| **Resource Exhaustion**: Containers setting inappropriate resource limits | Kubernetes resource quotas and limit ranges still apply. Container cannot exceed pod-level limits |
| **Pod Security Policy Bypass**: Feature being used in restricted environments | Integration with Pod Security Standards to block in restricted profiles |
| **Runtime Incompatibility**: Feature not working with older runtimes | The scheduler excludes unsupported nodes, and kubelet admission rejects Pods with an explicit cgroup mount mode on those nodes. |
| **cpuset Isolation**: Containers could modify `cpuset.cpus` to access CPUs allocated to other workloads by CPU Manager. | The `nsdelegate` mount option for cgroup v2 prevents containers from modifying their own resource limits (like `cpuset.cpus`). They can only create and manage sub-cgroups within their allocated constraints. |
| **Cgroup Descendant Exhaustion**: A container could create many descendant cgroups, exhausting node-level resources (memory and `inotify` watches were observed in the experiment) that are not counted against the container's `memory.max` limit, causing the node to become `NotReady`. | The kubelet applies per-Pod limits on the number and depth of live descendants; see [Descendant and Depth Limits](#descendant-and-depth-limits-pod-level-no-cri-changes) for scope and failure behavior. |

### cpuset Isolation

**Problem**: If a container has write access to its cgroup directory, it might attempt to modify sensitive resource limits like `cpuset.cpus` to access CPUs allocated to other workloads.

**Mitigation**:

The `nsdelegate` mount option for cgroup v2 provides kernel-level protection. When `/sys/fs/cgroup` is mounted with `nsdelegate`, and the container is in its own cgroup namespace, the kernel prevents the container from modifying its own resource limits. It can only create subdirectories (child cgroups) and manage resources within those sub-cgroups.

This feature relies on `nsdelegate` being supported and configured on the host. If the runtime cannot ensure this isolation (e.g., missing `nsdelegate` support), it MUST NOT enable writable cgroups for the container.

### Verified Behavior

#### Descendant Cgroup Memory Accounting

A container with `cgroupOptions.mountMode: Writable` and `memory.max: 128Mi` running on a GKE node (COS 125, kernel 6.12.68+, cgroup v2) was able to create approximately 42,000 sibling cgroups before any `mkdir` failed.

During the test:

- Container `memory.current` grew from 97 MB to 122 MB (within its limit).
- Node `Slab` grew from 200 MB to over 800 MB.
- Node `MemAvailable` dropped from 14.3 GB to ~20 MB.
- Before `NotReady`, the node raised a `ResourceExhausted` condition for `inotify-pressure` (100% of one user's watch quota, 12,288 watches).
- The node entered `NotReady`.

The container stayed within its own `memory.max` while the node ran out of memory and exhausted the inotify watch quota of one user, so the container's memory limit alone does not bound node-level resource consumption from descendant cgroups. To bound live descendants, the kubelet applies `cgroup.max.descendants` and `cgroup.max.depth` on the Pod-level cgroup it already manages when writable cgroups is enabled for the Pod.

#### Kubelet Stats Collection

By default, the kubelet's cAdvisor discovers and watches every cgroup under the
Pod cgroups, including cgroups that containers create. On a kind node running
v1.37.0, a container created 200 cgroups inside its own cgroup. The kubelet
gained 200 inotify watches, 200 housekeeping goroutines, and 200 series on
`/metrics/cadvisor`. The kubelet released all of them when the container removed
the cgroups. The Pod descendant limit caps each Pod subtree at 250 cgroups, and
therefore at 250 watches.

When the `PodAndContainerStatsFromCRI` feature gate
([KEP-2371](../2371-cri-pod-container-stats/README.md)) is enabled and the
runtime implements `PodSandboxStats`, the kubelet disables cAdvisor's container
discovery and does not watch cgroups that containers create. The runtime reports
Pod and container stats from the Pod cgroup and each container's cgroup. These
stats include the usage of cgroups that containers create because cgroup v2
accounts usage hierarchically.

## Design Details

### API Changes

#### Core API Types

Add `CgroupOptions` struct field to `SecurityContext` in both internal and external API:

**File**: `pkg/apis/core/types.go`
```go
type SecurityContext struct {
    // ... existing fields ...
    
    // CgroupOptions controls cgroup filesystem access and configuration.
    // This allows unprivileged containers to manage their own cgroup hierarchies on cgroup v2 systems.
    // Only effective on Linux containers with cgroup v2.
    // +optional
    CgroupOptions *CgroupOptions
}

// CgroupOptions defines options for cgroup filesystem access.
type CgroupOptions struct {
    // MountMode controls whether the cgroup filesystem is mounted as writable.
    // If not specified, the container runtime's default mount mode is used.
    // +optional
    MountMode *CgroupMountMode
}

// CgroupMountMode defines the mount mode for cgroup filesystem.
type CgroupMountMode string

const (
    // CgroupMountModeReadOnly mounts cgroup filesystem as read-only
    CgroupMountModeReadOnly CgroupMountMode = "ReadOnly"
    
    // CgroupMountModeWritable mounts cgroup filesystem as writable,
    // allowing containers to manage their own cgroup subtree
    CgroupMountModeWritable CgroupMountMode = "Writable"
)
```

**Verify Node Support:**

The kubelet declares `CgroupOptions` in `node.status.declaredFeatures` when the
feature gate is enabled and the node meets these prerequisites:

- The CRI implementation advertises `cgroup_mount_mode` in `RuntimeFeatures`.
- The host runs cgroup v2 with `nsdelegate`.
- The kubelet manages a cgroup per Pod (`--cgroups-per-qos`).

The runtime capability is node-level and independent of runtime handlers. It
reports only that the runtime accepts `cgroup_mount_mode`. The kubelet checks the
host prerequisites itself at startup.

Both explicit mount modes require these prerequisites. An omitted `mountMode`,
including an empty `cgroupOptions` object, uses the runtime default and does not
require the node to declare `CgroupOptions`. Descendant and depth limits apply
only when a container requests `Writable`.

The scheduler excludes nodes that do not declare the feature for Pods requesting
`mountMode: ReadOnly` or `mountMode: Writable`. Kubelet admission rejects such
Pods with `PodFeatureUnsupported` if they reach an unsupported node directly. See
[KEP-5328](../5328-node-declared-features/README.md).

When the kubelet creates a container that requests `Writable`, it checks that
the host has cgroup v2, `nsdelegate`, and a Pod cgroup. If the host lacks any of
them, the container fails with `CreateContainerConfigError`.

#### CRI API Changes

**File**: `cri-api/pkg/apis/runtime/v1/api.proto`
```protobuf
message LinuxContainerSecurityContext {
    // ... existing fields ...
    
    // cgroup_mount_mode controls how the cgroup filesystem is mounted in the
    // container. Only effective with cgroup v2.
    CgroupMountMode cgroup_mount_mode = 18;
}

// CgroupMountMode defines how the cgroup filesystem is mounted in a container.
enum CgroupMountMode {
    // CGROUP_MOUNT_MODE_UNSPECIFIED uses the runtime's default mount mode.
    CGROUP_MOUNT_MODE_UNSPECIFIED = 0;
    // CGROUP_MOUNT_MODE_READ_ONLY mounts the cgroup filesystem read-only.
    CGROUP_MOUNT_MODE_READ_ONLY = 1;
    // CGROUP_MOUNT_MODE_WRITABLE allows the container to manage its cgroup subtree.
    CGROUP_MOUNT_MODE_WRITABLE = 2;
}

// RuntimeFeatures (node-level, independent of runtime handlers) feeds node feature discovery.
message RuntimeFeatures {
    // ... existing fields (e.g. supplemental_groups_policy) ...

    // cgroup_mount_mode is set to true if the runtime supports the
    // cgroup_mount_mode field of LinuxContainerSecurityContext.
    bool cgroup_mount_mode = 4;
}
```

When `mountMode` is omitted, the kubelet leaves `cgroup_mount_mode` at
`CGROUP_MOUNT_MODE_UNSPECIFIED`, its zero value. The runtime uses its default
mount mode. For example, containerd's `cgroup_writable` configuration field
([internal/cri/config/config.go:108-109](https://github.com/containerd/containerd/blob/v2.3.5/internal/cri/config/config.go#L108-L109))
can make this default writable. An explicit `ReadOnly` maps to
`CGROUP_MOUNT_MODE_READ_ONLY` and overrides the runtime's default.

A runtime must fail container creation if it cannot honor
`CGROUP_MOUNT_MODE_WRITABLE`, including when `nsdelegate` is absent.

Each `CgroupOptions` option has its own `RuntimeFeatures` capability.

##### Descendant and Depth Limits (Pod-level, no CRI changes)

When a Pod opts into writable cgroups, the kubelet applies `cgroup.max.descendants`
of 250 and `cgroup.max.depth` of 50 to the Pod cgroup it manages. These limits cover
live descendants throughout the Pod subtree, including container cgroups. No CRI
changes are required.

The limits apply separately to each Pod. After a cgroup directory is removed, its
kernel resources can remain allocated until cleanup completes.

A Pod that requests `Writable` never runs a container without the limits. The
kubelet writes them when it creates the Pod cgroup. If the write fails, the kubelet
removes the Pod cgroup, records a `FailedToCreatePodContainer` event on the Pod,
and retries on the next sync.

### Implementation Details

#### CgroupOptions Implementation Flow

```mermaid
sequenceDiagram
    participant Kubelet as "Kubelet"
    participant KubeRuntime as "KubeRuntime"
    participant Container Runtime

    Note over Kubelet,Container Runtime: System & Runtime Validation

    Kubelet->>Kubelet: Admit pod against node.status.declaredFeatures
    Note right of Kubelet: Require CgroupOptions for an explicit mount mode

    alt Node does not declare CgroupOptions
        Kubelet->>Kubelet: Reject pod
        Note right of Kubelet: Reason: PodFeatureUnsupported
    else Node declares CgroupOptions

        Note over Kubelet,Container Runtime: Pod Cgroup Setup

        Kubelet->>Kubelet: EnsureExists(pod): create Pod-level cgroup
        Note right of Kubelet: Apply cgroup.max.descendants and cgroup.max.depth<br/>on the Pod cgroup; a write failure fails the Pod sync

        Note over Kubelet,Container Runtime: Container Creation
        
        KubeRuntime->>KubeRuntime: convertToRuntimeSecurityContext()
        Note right of KubeRuntime: Map CgroupOptions.MountMode to CRI

        KubeRuntime->>Container Runtime: CreateContainer(CgroupMountMode=Writable)
        
        Container Runtime->>Container Runtime: Generate OCI Spec with writable cgroups
        Note right of Container Runtime: Mount /sys/fs/cgroup as read-write

        Container Runtime-->>KubeRuntime: Container created
        
        KubeRuntime->>Container Runtime: StartContainer()
        Container Runtime-->>KubeRuntime: Container running with writable cgroups
    end
```

#### Validation

**Node Support Validation**

The scheduler and kubelet admission enforce the prerequisites described under
[API Changes](#api-changes).

**System Validation**:

`cgroupOptions.mountMode` accepts `ReadOnly` and `Writable`. The API rejects
`cgroupOptions` when `spec.os.name` is `windows`, following the other Linux-only
securityContext fields.

Ephemeral containers cannot set `cgroupOptions`. Allowing them to request writable
cgroups would require applying descendant limits to an existing Pod cgroup when
the Pod did not initially request them.

**File**: `pkg/kubelet/kuberuntime/security_context.go`

The container-create check described under [API Changes](#api-changes) runs
during CRI security context conversion.

**Pod-Level Descendant and Depth Limits**

The kubelet includes the descendant and depth limits in the cgroup v2 `Unified`
parameters when creating the Pod cgroup. See
[Descendant and Depth Limits](#descendant-and-depth-limits-pod-level-no-cri-changes)
for the values and failure behavior.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

- Update existing SecurityContext validation tests to include CgroupOptions field
- Ensure backward compatibility with existing SecurityContext tests

##### Unit tests

Coverage for new and existing packages:

- `k8s.io/kubernetes/pkg/apis/core/validation`:  Unit tests for CgroupOptions validation logic, Linux-only constraints, and ephemeral-container exclusion
- `k8s.io/kubernetes/pkg/kubelet/kuberuntime`: Security context conversion tests including CgroupOptions mapping
- `k8s.io/kubernetes/pkg/kubelet/cm`: Pod cgroup limit configuration and failure behavior when the limits cannot be applied
- `k8s.io/pod-security-admission/policy`: Pod Security Standards policy enforcement tests
- `k8s.io/kubernetes/pkg/apis/core/v1`:  API defaulting and conversion tests
- `k8s.io/component-helpers/nodedeclaredfeatures/features/cgroupoptions`: Node feature discovery and Pod requirement inference

##### Integration tests

- API server validation, including Linux-only enforcement and ephemeral-container exclusion
- Feature gate handling: field removal on create and preservation on existing Pods
- Pod Security Standards admission controller integration
- Scheduling excludes nodes that do not declare `CgroupOptions` for Pods with an explicit cgroup mount mode

##### e2e tests

- `test/e2e_node/cgroup_options_test.go`: Node E2E tests covering:
  - Basic functionality (writable vs read-only cgroups)
  - Multi-container pods with mixed settings
  - Integration with other SecurityContext fields
  - Node support requirements during kubelet admission
  - Containers not able to "escape" the resource limits set by the Pod
  - Runtime compatibility checks
  - Pod descendant and depth limits
- `critest`: a CRI conformance test in [cri-tools](https://github.com/kubernetes-sigs/cri-tools) exercising `cgroup_mount_mode` and its `RuntimeFeatures` advertisement, so container runtimes implementing the CRI API can validate conformance independently of Kubernetes.

### Graduation Criteria

#### Alpha

- **Alpha-1**
  - Feature implemented behind a feature gate `CgroupOptions`
  - Basic API, validation, and kubelet implementation completed

- **Alpha-2**
  - CRI API and container runtime changes
  - At least one runtime (containerd or CRI-O) implementing the CRI API merged to its dev branch or marked experimental, with an e2e test alongside, per the [CRI API dev policy](https://www.kubernetes.dev/docs/code/cri-api-dev-policies/#same-maturity-level-for-alpha)
  - Unit and integration tests implemented
  - Pod Security Standards integration complete
  - Node E2E tests passing on v2 systems

#### Beta

- Feature gate enabled by default
- E2E tests stable and passing consistently
- Both containerd and CRI-O support the feature in a released version, per the [CRI API dev policy](https://www.kubernetes.dev/docs/code/cri-api-dev-policies/#same-maturity-level-for-beta-and-ga)
- Re-evaluate the Pod descendant and depth limits with alpha feedback: the defaults of 250 and 50, and failing Pod startup when they cannot be applied
- Evaluate exposing the limits in `KubeletConfiguration` for per-node tuning
- Document that the kubelet watches cgroups that containers create unless `PodAndContainerStatsFromCRI` is enabled and the runtime implements `PodSandboxStats`. Recommend enabling the gate on nodes that run Pods with writable cgroups

#### GA

- Feature stable and ready for production use
- `PodAndContainerStatsFromCRI` is enabled by default; see [Kubelet Stats Collection](#kubelet-stats-collection)
- Conformance tests implemented where applicable
- Documentation completed

### Upgrade / Downgrade Strategy

Enable/disable the feature gate

**Upgrade**:

- If `securityContext.cgroupOptions` is omitted, its API value is `nil`. If
  `mountMode` is omitted, including in an empty `cgroupOptions` object, kubelet
  uses `CGROUP_MOUNT_MODE_UNSPECIFIED`, and the runtime uses its existing
  default cgroup mount mode.
- Enabling `CgroupOptions` does not change the cgroup mount of running
  containers. Pods that omit the field remain compatible. A normal node
  upgrade can still recreate individual Pods during drain and replacement.

**Update Flow**:

- `CgroupOptions` is immutable after Pod creation. Setting, changing, or
  clearing it requires Pod recreation.
- Ephemeral containers cannot use `cgroupOptions`. When `CgroupOptions` is
  enabled on kube-apiserver, the API rejects any non-nil value, including an
  empty object. When the gate is disabled, the field is dropped before
  validation.

**Downgrade**:

*Two scenarios depending on downgrade type:*

**Feature Gate Disabled (same Kubernetes version):**

The `CgroupOptions` gates on kube-apiserver and kubelet are configured
independently. Disabling the gate on one component does not change the
configuration of the other.

- When `CgroupOptions` is disabled on kube-apiserver, the apiserver drops
  `cgroupOptions` from newly created Pods. It preserves the field on existing
  Pods. A new Pod whose template requested an explicit mount mode therefore
  uses the runtime default mount mode.
- When `CgroupOptions` is disabled on kubelet and kubelet is restarted, the
  node no longer declares `CgroupOptions`. Kubelet rejects an existing Pod that
  explicitly requests `ReadOnly` or `Writable` with the
  `PodFeatureUnsupported` reason. The Pod becomes `Failed`, and kubelet stops
  its running containers, as specified by the
  [Node Declared Features policy](../5328-node-declared-features/README.md#declared-feature-changes-on-existing-nodes).
  Pods with an omitted mount mode, including an empty `cgroupOptions` object,
  are not affected.
- A replacement Pod created through kube-apiserver uses the runtime default
  when the kube-apiserver gate remains disabled, even if its workload template
  requested an explicit mount mode. If the kube-apiserver gate is enabled, a
  Pod that explicitly requests `ReadOnly` or `Writable` requires a node that
  declares `CgroupOptions`. `NodeDeclaredFeatures` prevents normal scheduling
  to unsupported nodes; kubelet also rejects the Pod if it is bound directly
  to one.

**True Version Downgrade (to Kubernetes version without CgroupOptions field):**

The following describes compatibility behavior during a manual true version
downgrade; it is not a kubeadm-managed workflow.

- Before a planned true version downgrade, workload templates should omit
  `cgroupOptions`, and affected Pods should be recreated. The `cgroupOptions`
  field is immutable and cannot be removed by updating an existing Pod spec.
- An older apiserver does not recognize `cgroupOptions`. When it reads a Pod
  stored by a newer apiserver, it ignores the unknown field and omits it from
  API responses. This read does not modify etcd data. If the older apiserver
  later processes a write for that Pod, it can serialize and persist its older
  representation, which does not contain `cgroupOptions`. The field can then
  be lost from stored Pod data, even if the requested change only updates
  metadata. A verified pre-downgrade etcd snapshot provides a recovery point
  containing the original stored Kubernetes objects.
- Containers that started before the downgrade keep their existing cgroup
  mount.
- An older kubelet does not recognize or receive `cgroupOptions`. When it
  recreates a container, the runtime then uses its default mount mode.
- Requests to an older apiserver that include `cgroupOptions` follow standard
  unknown-field validation behavior: strict validation rejects the request;
  non-strict validation drops the field, with a warning in `Warn` mode.
  `kubectl` uses strict validation by default.

### Version Skew Strategy

Before `CgroupOptions` is enabled on kube-apiserver, the scheduler must recognize its node requirement.
Otherwise, the scheduler can place Pods on older kubelets that ignore the field.

**kubelet vs Container Runtime**:
A node whose runtime does not advertise `cgroup_mount_mode` does not declare
`CgroupOptions`. The scheduler excludes it for Pods with an explicit cgroup mount mode.

**kube-scheduler vs kubelet**:
The scheduler excludes nodes whose kubelet predates the feature or has its gate
disabled. API field handling is described under
[Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy).

## Production Readiness Review Questionnaire



### Feature Enablement and Rollback


###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `CgroupOptions`
  - Components depending on the feature gate: kubelet, kube-apiserver
- [ ] Other

The feature can be controlled via:
1. **Feature Gate**: `--feature-gates=CgroupOptions=true/false`
2. **Runtime Support**: Requires compatible container runtime
3. **Pod Specification**: Per-container `securityContext.cgroupOptions.mountMode` field

- Will enabling / disabling the feature require downtime of the control plane? **Yes**
- Will enabling / disabling the feature require downtime or reprovisioning of a node? **Yes** (kubelet restart required for feature gate changes)

###### Does enabling the feature change any default behavior?

**No**. Containers that omit `securityContext.cgroupOptions.mountMode` continue to use the runtime's default mount mode. Explicit `ReadOnly` and `Writable` requests select the corresponding mount mode.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

**Yes**. Restarting kubelet with the gate disabled terminates Pods with an explicit
cgroup mount mode. Disabling it on kube-apiserver drops the field from new Pods.
See [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy).

###### What happens if we reenable the feature if it was previously rolled back?

New Pods with an explicit `ReadOnly` or `Writable` mount mode can be created again. Pods marked `Failed` during rollback must be replaced.

###### Are there any tests for feature enablement/disablement?

Tests will verify:

- Feature gate disabled: the apiserver drops `cgroupOptions` from new Pods and preserves it on existing Pods
- Feature gate enabled: API accepts and kubelet processes the field correctly
- Runtime compatibility testing with and without feature support



### Rollout, Upgrade and Rollback Planning

This section complements [Feature Enablement and Rollback](#feature-enablement-and-rollback) above with rollout-specific failure modes.

###### How can a rollout or rollback fail? Can it impact already running workloads?

Possible rollout failure modes:

- **Version skew (apiserver enabled, kubelet not)**: With a scheduler that recognizes the feature, Pods with an explicit cgroup mount mode remain `Pending` if no compatible node is available.
- **Runtime missing CRI support**: The node does not declare `CgroupOptions`. The scheduler excludes it for Pods with an explicit cgroup mount mode.
- **Host missing cgroup v2, `nsdelegate`, or per-Pod cgroups**: The node does not declare `CgroupOptions`. The scheduler excludes it, and kubelet admission rejects opted-in Pods that reach it directly.

Rollback (disabling the feature gate):

- Disabling the gate on kube-apiserver drops `cgroupOptions` from new Pods, which then use the runtime's default cgroup mount mode. Operations that require writable cgroups fail if that default is read-only.
- Restarting kubelet with the gate disabled terminates Pods with an explicit cgroup mount mode; see [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy).

No impact on workloads that do not opt in to the feature.

###### What specific metrics should inform a rollback?

No dedicated metrics for alpha. Existing kubelet pod-startup counters (`kubelet_started_pods_errors_total`, `kubelet_runtime_operations_errors_total`) do not carry a feature-level label that can isolate this feature's impact, so operators should:

- Audit which pods opt in (see the kubectl query under [Monitoring Requirements](#monitoring-requirements)).
- Watch scheduler events for opted-in Pods stuck `Pending`, and `PodFeatureUnsupported` events for kubelet admission failures.
- Track restart rates for opted-in pods.

Whether to add a feature-specific dimension to existing counters is deferred to beta, contingent on observed adoption.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

TODO: requires the alpha implementation in kubernetes/kubernetes and a container runtime supporting the new CRI field. Plan: implement in kubernetes and containerd, then exercise the upgrade -> downgrade -> upgrade flow in `kind` (cheap to swap node images and flip feature gates). Cases to cover:

- Enable feature gate, create pod with `cgroupOptions.mountMode: Writable`, confirm container has writable `/sys/fs/cgroup`.
- Disable the gate on kube-apiserver, confirm it drops the field from new Pods and preserves it on existing Pods.
- Disable the gate on kubelet and restart it. Confirm existing Pods with an explicit cgroup mount mode enter `Failed` and their containers are terminated.
- Re-enable feature gate, confirm new pods can be created again.

True version downgrade behavior (to a kubernetes version without the field) is described under [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy).

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.


### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

By inspecting Pod specs:

```
kubectl get pods -A -o json \
  | jq '.items[] | select([.spec.containers[]?, .spec.initContainers[]?] | any(.securityContext.cgroupOptions.mountMode != null)) | {namespace: .metadata.namespace, name: .metadata.name}'
```

This follows the alpha pattern used by KEP-3857 (recursive read-only mounts) and KEP-4639 (OCI volume source). A dedicated metric can be considered at beta if there is demand.

###### How can someone using this feature know that it is working for their instance?

From inside the container:

- `mount | grep '^cgroup2'` should show `/sys/fs/cgroup` mounted with `rw`.
- `mkdir /sys/fs/cgroup/test && rmdir /sys/fs/cgroup/test` should succeed.

Check `node.status.declaredFeatures` for `CgroupOptions`. The scheduler excludes nodes that do not declare it for Pods with an explicit cgroup mount mode. Kubelet admission rejects such Pods if they reach an unsupported node directly.

- [x] Events
  - Event Reason: `PodFeatureUnsupported` when kubelet admission rejects a Pod with an explicit cgroup mount mode on an unsupported node.
- [ ] API .status
- [ ] Other (treat as last resort)

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

No dedicated SLOs at alpha. Existing pod startup SLOs apply.

Beta will revisit whether a startup-latency SLO scoped to opted-in pods is warranted.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

For alpha, existing kubelet SLIs apply:

- `kubelet_pod_start_duration_seconds`
- `kubelet_started_pods_errors_total`

- [ ] Metrics
- [x] Other (treat as last resort)
  - Details: For alpha, rely on existing kubelet metrics combined with pod-spec audit (above), scheduler events, and `PodFeatureUnsupported` events for per-pod failure attribution.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

None planned for alpha. The dominant pattern for SecurityContext-adjacent alpha features (KEP-3857, KEP-4639, KEP-2400) is to defer dedicated metrics until beta. If beta uptake reveals a need for fleet-wide visibility, candidates to consider are: a per-feature label on existing pod-startup counters, or a dedicated `kubelet_writable_cgroup_pods` gauge (matching the KEP-127 pattern, which added `started_user_namespaced_pods_total` at beta).


### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No new in-cluster services. The feature depends on:

- A Linux kernel with cgroup v2 support.
- The host's `/sys/fs/cgroup` mounted with the `nsdelegate` option (the default in modern systemd; required for safe operation, see [cpuset Isolation](#cpuset-isolation)).
- A container runtime that supports the new CRI `cgroup_mount_mode` field and advertises it in the node-level `RuntimeFeatures`.
- A kubelet managing a cgroup per Pod (`--cgroups-per-qos`).
- Node Declared Features enabled in kube-scheduler and kubelet.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No new top-level types. The feature adds a nested `CgroupOptions` struct and a `CgroupMountMode` string enum inside the existing `SecurityContext`.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Marginal. Pods opting in to the feature add a small nested object containing a single string enum to their spec. Pods that do not opt in are unchanged.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No measurable impact. The scheduler filters nodes using Node Declared Features, and kubelet admission checks the Pod requirements against the node's declared features. Host and runtime support are discovered at kubelet startup.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

The feature does not add controllers. By default, the kubelet's cAdvisor adds an inotify watch and a housekeeping goroutine for each cgroup that a container creates, up to the Pod descendant limit. With `PodAndContainerStatsFromCRI` enabled and a runtime that implements `PodSandboxStats`, the kubelet does not watch these cgroups. See [Kubelet Stats Collection](#kubelet-stats-collection).

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

Yes. Descendant cgroups consume kernel memory and inotify watches that `memory.max` does not count (see [Verified Behavior](#verified-behavior)). The Pod limits bound live descendants; see [Descendant and Depth Limits](#descendant-and-depth-limits-pod-level-no-cri-changes) for scope and failure behavior.

Mitigations:

- The kubelet applies `cgroup.max.descendants` and `cgroup.max.depth` on the Pod-level cgroup when `mountMode: Writable` is enabled for the Pod.
- With `PodAndContainerStatsFromCRI` enabled and a runtime that implements `PodSandboxStats`, the kubelet does not watch cgroups that containers create; see [Kubelet Stats Collection](#kubelet-stats-collection).
- The feature is opt-in via `cgroupOptions.mountMode: Writable`, so cluster administrators can restrict usage via Pod Security Standards or admission policies.


### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No different from existing pod creation. If the apiserver is unavailable, no new pods (including those opting in to writable cgroups) can be created. Pods already running with writable cgroups are unaffected; the feature has no runtime dependency on the control plane after pod startup.

###### What are other known failure modes?

| Failure | Detection | Mitigations | Diagnostics | Testing |
|---|---|---|---|---|
| Container runtime does not support the CRI field | The node does not declare `CgroupOptions`: the scheduler excludes it, and kubelet admission rejects Pods that reach it directly with `PodFeatureUnsupported` | Use a runtime advertising `cgroup_mount_mode`; or remove the field from the pod spec | `node.status.declaredFeatures`; scheduler and Pod events | unit + integration tests |
| Host is on cgroup v1 | The node does not declare `CgroupOptions`; scheduler exclusion and `PodFeatureUnsupported` as in the first row | Migrate the node to cgroup v2; or remove the field from the pod spec | `node.status.declaredFeatures`; scheduler and Pod events | node support tests |
| Host's `/sys/fs/cgroup` is not mounted with `nsdelegate` | The node does not declare `CgroupOptions`; scheduler exclusion and `PodFeatureUnsupported` as in the first row. The runtime refuses writable mounts if `nsdelegate` is removed after kubelet startup | Mount `/sys/fs/cgroup` with `nsdelegate` and restart kubelet after changing the mount options | `findmnt /sys/fs/cgroup`; runtime logs | runtime contract; node support tests |
| Kubelet cannot apply the Pod descendant and depth limits | `FailedToCreatePodContainer` event on the Pod; its containers do not start and the kubelet retries each sync | Remove the field from the pod spec; or fix the cgroup write error reported in the event | Pod events; kubelet logs | unit tests |
| Pod reaches its descendant limit | `mkdir` returns `EAGAIN` when `cgroup.max.descendants` is reached | Remove unused cgroups inside the container; the limit is per Pod and bounds node resource use | container logs; Pod cgroup `cgroup.stat` | e2e for descendant bound |

###### What steps should be taken if SLOs are not being met to determine the problem?

No SLOs at alpha. If pod startup latency degrades after enabling the feature gate:

1. Check scheduler events for Pods stuck `Pending` and kubelet events for `PodFeatureUnsupported`.
2. Confirm the node lists `CgroupOptions` under `status.declaredFeatures` via `kubectl get node -o yaml`.
3. Confirm the host has cgroup v2 with `nsdelegate` (`findmnt /sys/fs/cgroup`).
4. Follow [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy) when disabling the gate; restarting kubelet terminates Pods with an explicit cgroup mount mode.

## Implementation History

- **2025-08-25**: KEP written and proposed
- **2026-02-05**: KEP refined to focus strictly on `nsdelegate` for isolation and justify API opt-in requirements
- **2026-06-08**: Per SIG-Node discussion, descendant/depth exhaustion is mitigated by the kubelet setting `cgroup.max.descendants`/`cgroup.max.depth` defaults on the Pod-level cgroup rather than via new CRI fields
- **TBD**: Alpha implementation targeting v1.38
- **TBD**: Beta implementation targeting v1.39
- **TBD**: GA implementation, target release TBD

## Drawbacks

1. **Runtime Dependency**: Feature requires specific container runtime versions.
1. **Security Complexity**: Adds another security dimension that may need to be considered in Pod Security Standards

## Alternatives

### Alternative 1: Runtime-specific Annotations
Similar to CRI-O's `io.kubernetes.cri-o.cgroup2-mount-hierarchy-rw` annotation approach.

**Pros**: No API changes required
**Cons**: Runtime-specific, not portable, harder to enforce security policies

### Alternative 2: Boolean Field Instead of Struct

Use a simple `WritableCgroups *bool` field instead of the `CgroupOptions` struct:

```go
type SecurityContext struct {
    // WritableCgroups controls whether the container has write access to cgroup interfaces.
    // +optional
    WritableCgroups *bool
}
```

```yaml
securityContext:
  writableCgroups: true
```

**Pros**: 
- Simpler API

**Cons**: 
- Not extensible for future cgroup-related configurations

The struct-based approach (`CgroupOptions`) was chosen to allow future extensibility.

### Alternative 3: Runtime Auto-Detection (Implicit Behavior)

Instead of a new API field, container runtimes could automatically detect if the host has `nsdelegate` enabled and, if so, mount cgroups as read-write.

**Pros**:
- No API changes required.
- "It just works" for configured nodes.

**Cons**:
- **Lack of Visibility**: Cluster administrators cannot easily identify which workloads are using this capability.
- **Policy Enforcement**: Admission controllers and security policies cannot restrict usage since it's not in the Pod spec.
- **Defense in Depth**: It removes a layer of defense. Even if `nsdelegate` is safe, keeping the default restricted protects against potential kernel bugs or implementation flaws.

## Infrastructure Needed (Optional)

- **Runtime Support**: Coordination with container runtime projects to adopt CRI API changes
- **Documentation**: Updates to Kubernetes documentation and security guides
