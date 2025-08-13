
<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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

This KEP proposes adding a `WritableCgroups` field to the container SecurityContext in Kubernetes to allow unprivileged containers to have writable access to cgroup interfaces on cgroup v2 systems.

## Motivation

In containerd v2.1.0, the [writable_cgroups](https://github.com/containerd/containerd/blob/75cb2b7193e4e490e9fbdc236c0e811ccaba3376/internal/cri/config/config.go#L109) configuration option was introduced to ensure that, when cgroup v2 is enabled, the cgroup interface (`/sys/fs/cgroup`) is mounted with read-write permissions for containers. With cgroup v2's secure delegation model, unprivileged containers can safely manage their own cgroup subtree without compromising system security. Currently, updating this setting requires stopping running workloads, modifying the containerd configuration, and restarting the service on every node. Additionally the configuration enables writable cgroups for all Pods running on the node.

To simplify adoption, and given that some runtimes already support a similar `cgroups_writable` setting in their configuration, this field can be exposed through the the Kubernetes API and CRI interface. Container runtimes will then be updated to support this setting via CRI.

Related Issues:
- https://github.com/containerd/containerd/issues/10924
- https://github.com/kubernetes/kubernetes/issues/121190

### Goals

- Add a `WritableCgroups` boolean field to the container SecurityContext object
- CRI API changes
- Integrate with Pod Security Standards to ensure appropriate security policy enforcement
- Container runtime modifications - Containerd
- Maintain backward compatibility with existing workloads

## Proposal

Add a new `WritableCgroups *bool` field to the `SecurityContext` struct in the core v1 API. When set to `true` on cgroup v2 systems, this field will instruct the container runtime to mount the cgroup filesystem with write permissions for the container's own cgroup subtree.

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
      WritableCgroups: true
```

### User Stories (Optional)

#### Story 1: Container-in-Container Development

As a developer, I need to run Docker-in-Docker for testing and development purposes. Currently, I need to use privileged containers which exposes unnecessary security risks. With `WritableCgroups`, I can run these nested containers securely while still allowing them to manage their own resource constraints.

#### Story 2: Dynamic Resource Management

As an application developer building a resource-intensive application, I want my application to be able to monitor its memory usage via `/sys/fs/cgroup/memory.current` and dynamically adjust its memory limits via `/sys/fs/cgroup/memory.max` based on workload patterns, without requiring privileged access or manual operator intervention.

### Notes/Constraints/Caveats (Optional)

- **cgroup v2 Only**: This feature requires cgroup v2 and will be ignored on cgroup v1 systems
- **Linux Only**: The field is only valid on Linux containers and will be validated accordingly
- **Runtime Support**: Requires container runtime support (Containerd, CRI-O)
- **Security Context Integration**: Must work cohesively with other SecurityContext fields

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| **Security Bypass**: Containers gaining unauthorized access to system cgroups | Only allow write access to container's own cgroup subtree. cgroup v2 delegation model provides isolation |
| **Resource Exhaustion**: Containers setting inappropriate resource limits | Kubernetes resource quotas and limit ranges still apply. Container cannot exceed pod-level limits |
| **Pod Security Policy Bypass**: Feature being used in restricted environments | Integration with Pod Security Standards to block in restricted profiles |
| **Intra-Pod Resource Starvation**: A container in a Burstable or BestEffort Pod could modify its cgroup limits to consume resources intended for other containers in the same Pod, as no Pod-level cgroup enforces a ceiling. | Enforce that any container setting WritableCgroups: true must belong to a Pod that qualifies for the Guaranteed QoS class (where all containers have CPU and memory requests and limits set, and they are equal). This can be done via a validating admission policy or potentially in the normal validation code. This ensures Kubelet creates a pod-level cgroup that acts as a hard boundary for all containers within the pod. | 
| **Runtime Incompatibility**: Feature not working with older runtimes | Graceful degradation - field ignored if runtime doesn't support it |

## Design Details

### API Changes

#### Core API Types

Add `WritableCgroups *bool` field to `SecurityContext` in both internal and external API:

**File**: `pkg/apis/core/types.go`
```go
type SecurityContext struct {
    // ... existing fields ...
    
    // WritableCgroups controls whether the container has write access to cgroup interfaces.
    // This allows unprivileged containers to manage their own cgroup hierarchies on cgroup v2 systems.
    // Only effective on Linux containers with cgroup v2.
    // +optional
    WritableCgroups *bool
}
```

#### CRI API Changes

**File**: `cri-api/pkg/apis/runtime/v1/api.proto`
```protobuf
message LinuxContainerSecurityContext {
    // ... existing fields ...
    
    // writable_cgroups controls whether the container has write access to cgroup interfaces.
    // Only effective with cgroup v2.
    bool writable_cgroups = 18;
}
```

### Implementation Flow

#### Kubelet to Container Runtime Flow

```mermaid
sequenceDiagram
    participant Kubelet
    participant KubeRuntime as "KubeRuntime"
    participant CRI as "CRI API"
    participant Containerd
    Kubelet->>KubeRuntime: CreatePodSandbox()
    KubeRuntime->>CRI: RunPodSandbox(RunPodSandboxRequest)
    CRI->>Containerd: gRPC call
    Containerd-->>CRI: RunPodSandboxResponse
    CRI-->>KubeRuntime: PodSandbox created
    
    loop For each container
        Kubelet->>KubeRuntime: CreateContainer(containerConfig)
        KubeRuntime->>KubeRuntime: convertToRuntimeSecurityContext()<br/>Map WritableCgroups to CRI
        KubeRuntime->>CRI: CreateContainer(CreateContainerRequest)<br/>SecurityContext.Linux.WritableCgroups = true
        CRI->>Containerd: CreateContainer RPC
        Containerd->>Containerd: Generate OCI spec with<br/>writable cgroup configuration
        Containerd-->>CRI: CreateContainerResponse
        CRI-->>KubeRuntime: Container created
        
        KubeRuntime->>CRI: StartContainer(StartContainerRequest)
        CRI->>Containerd: StartContainer RPC
        Containerd-->>CRI: StartContainerResponse
        CRI-->>KubeRuntime: Container started
    end
    
    KubeRuntime-->>Kubelet: All containers running
    Kubelet->>APIServer: Update status: Running
```

### Validation and Defaulting

**File**: `pkg/apis/core/validation/validation.go`
```go
func ValidateSecurityContext(sc *core.SecurityContext, fldPath *field.Path) field.ErrorList {
    allErrs := field.ErrorList{}
    
    if sc.WritableCgroups != nil {
        // WritableCgroups is Linux-only
        if sc.WindowsOptions != nil {
            allErrs = append(allErrs, field.Invalid(fldPath.Child("WritableCgroups"), 
                sc.WritableCgroups, "cannot be set when WindowsOptions is specified"))
        }
    }
    
    return allErrs
}
```

**QoS Class Validation**:

```go
func ValidatePodSpec(spec *core.PodSpec, podMeta metav1.Object, fldPath *field.Path, opts PodValidationOptions) field.ErrorList {
    allErrs := field.ErrorList{}
    
    // Check if any container has WritableCgroups: true
    if hasWritableCgroups {
        tempPod := &core.Pod{Spec: *spec}
        qosClass := qos.ComputePodQOS(tempPod)
        if qosClass != core.PodQOSGuaranteed {
            allErrs = append(allErrs, field.Invalid(fldPath, spec, 
                "WritableCgroups=true requires Guaranteed QoS class (equal CPU/memory requests and limits for all containers)"))
        }
    }
    
    return allErrs
}
```

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

- Update existing SecurityContext validation tests to include WritableCgroups field
- Ensure backward compatibility with existing SecurityContext tests

##### Unit tests

Coverage for new and existing packages:

- `k8s.io/kubernetes/pkg/apis/core/validation`:  Unit tests for WritableCgroups validation logic, Linux-only constraints
- `k8s.io/kubernetes/pkg/kubelet/kuberuntime`: Security context conversion tests including WritableCgroups mapping
- `k8s.io/pod-security-admission/policy`: Pod Security Standards policy enforcement tests
- `k8s.io/kubernetes/pkg/apis/core/v1`:  API defaulting and conversion tests

##### Integration tests

- `TestWritableCgroupsSecurityContextValidation`: API server validation integration test ensuring Linux-only enforcement
- `TestPodSecurityStandardsWritableCgroups`: Pod Security Standards admission controller integration test
- `TestKubeletSecurityContextConversion`: Kubelet CRI conversion integration test

##### e2e tests

- `test/e2e_node/writable_cgroup_test.go`: Node E2E tests covering:
  - Basic functionality (writable vs read-only cgroups)
  - Multi-container pods with mixed settings
  - Integration with other SecurityContext fields
  - cgroup v2 requirement validation
  - Runtime compatibility checks

### Graduation Criteria

#### Alpha (v1.35)

- Feature implemented behind a feature gate `WritableCgroups`
- Basic API, validation, and kubelet implementation complete
- CRI API and Containerd Changes
- Unit and integration tests implemented
- Pod Security Standards integration complete
- Node E2E tests passing on v2 systems

#### Beta (v1.36)

- Feature gate enabled by default
- E2E tests stable and passing consistently
- At least one runtime (containerd) supporting the feature in a released version
- User feedback incorporated from alpha testing

#### GA (v1.37)

- Feature stable and ready for production use
- Conformance tests implemented where applicable
- Documentation completed

### Upgrade / Downgrade Strategy

Enable/disable the feature gate

**Upgrade**: 
- New field is optional and defaults to `nil` (no change in behavior)
- Existing workloads continue to function without modification

**Update Flow**:
- `WritableCgroups` field is **immutable** after pod creation.
- Changes to `WritableCgroups` require pod recreation (delete + create)

**Downgrade**:

*Two scenarios depending on downgrade type:*

**Feature Gate Disabled (same Kubernetes version):**
- New pods with `WritableCgroups: true` will have the field silently dropped to `nil`
- Existing pods with `WritableCgroups: true` continue to work
- No errors occur - field is accepted but ignored

**True Version Downgrade (to Kubernetes version without WritableCgroups field):**
- Pods with `WritableCgroups` field will be **rejected** with strict decoding error: `unknown field "spec.containers[0].securityContext.writableCgroups"`
- Remove the field from pod specs before downgrading

### Version Skew Strategy

**kubelet vs Container Runtime**:
- On unsupported runtimes, the feature is silently ignored because the WritableCgroups field is not used.
- Runtime version check not enforced to avoid breaking existing setups


## Production Readiness Review Questionnaire



### Feature Enablement and Rollback


###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `WritableCgroups`
  - Components depending on the feature gate: kubelet, kube-apiserver
- [ ] Other

The feature can be controlled via:
1. **Feature Gate**: `--feature-gates=WritableCgroups=true/false`
2. **Runtime Support**: Requires compatible container runtime
3. **Pod Specification**: Per-container `securityContext.WritableCgroups` field

- Will enabling / disabling the feature require downtime of the control plane? **Yes**
- Will enabling / disabling the feature require downtime or reprovisioning of a node? **Yes** (kubelet restart required for feature gate changes)

###### Does enabling the feature change any default behavior?

**No**. The feature only affects containers that explicitly set `securityContext.WritableCgroups: true`. Default behavior remains unchanged - containers continue to have read-only access to cgroup filesystem.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

**Yes**. Disabling the feature gate will:
- Prevent new pods with `WritableCgroups: true` from being created
- Existing running containers continue with their current cgroup permissions until restart
- API server will reject new pods with the field set

###### What happens if we reenable the feature if it was previously rolled back?

New pods with `WritableCgroups: true` can be created again. No data loss or corruption occurs during disable/enable cycles.

###### Are there any tests for feature enablement/disablement?

**Yes**. E2E tests will verify:
- Feature gate disabled: API rejects pods with writableCgroups field
- Feature gate enabled: API accepts and kubelet processes the field correctly
- Runtime compatibility testing with and without feature support



### Rollout, Upgrade and Rollback Planning



###### How can a rollout or rollback fail? Can it impact already running workloads?



###### What specific metrics should inform a rollback?


###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?



###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?



### Monitoring Requirements



###### How can an operator determine if the feature is in use by workloads?



###### How can someone using this feature know that it is working for their instance?



- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?



###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?



- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?



### Dependencies



###### Does this feature depend on any specific services running in the cluster?


### Scalability


###### Will enabling / using this feature result in any new API calls?



###### Will enabling / using this feature result in introducing new API types?



###### Will enabling / using this feature result in any new calls to the cloud provider?



###### Will enabling / using this feature result in increasing size or count of the existing API objects?



###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?



###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?


###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?



### Troubleshooting



###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?



###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- **2025-08-25**: KEP written and proposed
- **TBD**: Alpha implementation targeting v1.35
- **TBD**: Beta implementation targeting v1.36
- **TBD**: GA implementation targeting v1.37

## Drawbacks

1. **Runtime Dependency**: Feature requires specific container runtime versions.
3. **Security Complexity**: Adds another security dimension that may need to be considered in Pod Security Standards

## Alternatives

### Alternative 1: Runtime-specific Annotations
Similar to CRI-O's `io.kubernetes.cri-o.cgroup2-mount-hierarchy-rw` annotation approach.

**Pros**: No API changes required
**Cons**: Runtime-specific, not portable, harder to enforce security policies

## Infrastructure Needed (Optional)

- **Runtime Support**: Coordination with container runtime projects to adopt CRI API changes
- **Documentation**: Updates to Kubernetes documentation and security guides


