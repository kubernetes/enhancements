# KEP-XXXX: Container OOM Kill Mode Configuration

<!-- toc -->

- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Multi-process Web Application](#story-1-multi-process-web-application)
    - [Story 2: Database Container](#story-2-database-container)
    - [Story 3: Batch Processing Workload](#story-3-batch-processing-workload)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
    - [Container API](#container-api)
  - [Kubelet Implementation](#kubelet-implementation)
  - [Validation Rules](#validation-rules)
  - [CRI Changes](#cri-changes)
  - [Interaction with Existing Features](#interaction-with-existing-features)
    - [OOMScoreAdj](#oomscoreadj)
    - [Pod QoS Classes](#pod-qos-classes)
    - [Memory QoS](#memory-qos)
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
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required _prior to targeting to a milestone / release_.

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

This KEP proposes adding a per-container `oomKillMode` field to configure whether the OOM killer terminates a single process or all processes in a container. The field extends the existing per-node `singleProcessOOMKill` kubelet flag (v1.31) to provide container-level granularity, addressing production issues reported when cgroup v2 changed the default OOM behavior from single-process to group-kill semantics.

## Motivation

The transition from cgroup v1 to cgroup v2 changed OOM killer behavior significantly:

- **cgroup v1**: Kills only the process that triggered OOM (single process kill)
- **cgroup v2**: Kills all processes(group kill) or a process in the container cgroup via `memory.oom.group`

The kubelet flag `singleProcessOOMKill` (added in [PR #126096](https://github.com/kubernetes/kubernetes/pull/126096) for v1.31) provides node-level control, but different workloads require different behaviors:

- **Multi-process applications** (e.g., PHP-FPM, Apache, Python Gunicorn/uWSGI) benefit from single-process kills, allowing the application to continue running with remaining healthy processes
- **Single-process or tightly-coupled applications** (e.g., databases, stateful services) require group kills to maintain consistency

The current all-or-nothing approach at the node level forces administrators to choose a single behavior that may not be optimal for all processes, potentially leading to:

- Unnecessary downtime for resilient multi-process applications
- Inconsistent state in applications that expect full container restarts
- Difficulty in migrating diverse workloads from cgroup v1 to v2 environments

### Real-World Impact

The transition to cgroup v2's default group kill behavior has had significant production impact, as documented in [PR #117793](https://github.com/kubernetes/kubernetes/pull/117793) and [PR #122813](https://github.com/kubernetes/kubernetes/pull/122813):

- Multiple organizations reported severe impact on large containers where group kills caused unnecessary service disruptions[^1]
- Production environments experienced issues where group kills bypassed SIGTERM/SIGKILL handlers, preventing graceful shutdown
- Reports of potential issues affecting customers due to the behavioral change
- Multi-process workloads experienced complete container restarts when only a single process needed to be terminated[^2]

These reports led to [PR #122813](https://github.com/kubernetes/kubernetes/pull/122813), which attempted to add a kubelet flag but was closed after community discussion concluded that container-level configuration was the proper solution[^3]. The consensus was that node-level configuration cannot adequately address the needs of heterogeneous workloads.

### Goals

- Add a per-container `oomKillMode` field to allow container-level OOM behavior configuration
- Maintain full backward compatibility with the existing `singleProcessOOMKill` kubelet configuration during migration

### Non-Goals

- Provide cgroups v1 and Windows support
- Pod-level/WholePod OOM behavior is out of scope; we may revisit it based on demand and feedback from operators.

## Proposal

Add an `oomKillMode` field to the Container specification in the core v1 API. This design choice is deliberate and based on several key considerations:

### Why Container-level Configuration?

1. **Granular Control**: Different containers in the same pod may have different OOM requirements:
   - A sidecar container might benefit from single-process kills
   - The main application container might require group kills for consistency

2. **Natural Alignment**: OOM behavior is inherently a container-level concern:
   - Memory limits are set per container
   - OOMScoreAdj is calculated per container based on QoS
   - cgroups are managed at the container level

3. **Consistency with Existing APIs**: Follows the pattern of other container-level resource configurations:
   - Resources (limits/requests) can be set at both pod and container level
   - SecurityContext can be set at both pod and container level
   - The existing `singleProcessOOMKill` implementation operates per container

### API Design

```go
type Container struct {
    // ... existing fields ...

    // Compute Resources required by this container.
    // Cannot be updated.
    // More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
    // +optional
    Resources ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,8,opt,name=resources"`

    // OOMKillMode specifies how the OOM killer behaves for this container.
    // - "Single": only the process that triggered OOM is killed
    // - "Group": all processes in the container are killed (cgroup v2 default)
    // If not specified, the behavior is determined by the kubelet configuration.
    // This field requires the ContainerOOMKillMode feature gate to be enabled.
    // +featureGate=ContainerOOMKillMode
    // +optional
    OOMKillMode *OOMKillMode `json:"oomKillMode,omitempty" protobuf:"bytes,26,opt,name=oomKillMode"`

    // ... rest of fields ...
}

// OOMKillMode defines how the OOM killer behaves when the container exceeds its memory limit
// +enum
type OOMKillMode string

const (
    // OOMKillModeSingle kills only the process that triggered the OOM condition
    OOMKillModeSingle OOMKillMode = "Single"

    // OOMKillModeGroup kills all processes in the container when OOM is triggered
    OOMKillModeGroup OOMKillMode = "Group"
)
```

The optional `oomKillMode` field will be wired through every Kubernetes container surface:

- `pod.spec.containers` (regular and sidecar containers)
- `pod.spec.initContainers`
- `ephemeralContainers`

The existing `EphemeralContainer` struct will gain the same optional field so that debugging workflows stay consistent with long-running containers.

### Configuration Hierarchy

The OOM kill behavior follows this precedence:

1. **Container-level `oomKillMode`** (highest priority) - if explicitly set
2. **Kubelet `singleProcessOOMKill` flag** - node-level default when `oomKillMode` is unset and the flag is set
3. **System default** - cgroup v2 defaults to group kill, cgroup v1 defaults to single process

This hierarchy ensures backward compatibility while enabling smooth migration from node-level to container-specific configuration.

### User Stories

#### Story 1: Multi-process Web Application

Multi-process web servers (PHP-FPM, Python Gunicorn/uWSGI, Ruby Unicorn) run a master process with multiple worker processes. When a worker consumes excessive memory due to a memory leak or processing a large request, only that problematic worker should be killed. This allows the application to continue serving requests with remaining healthy workers while the master spawns a replacement, ensuring high availability without full service disruption.

#### Story 2: Database Container

Databases like PostgreSQL require all processes to be killed together on OOM to ensure data consistency. Partial kills could leave the database in an inconsistent state.

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
    - name: multi-process-app
      oomKillMode: Single # Explicitly set for containers needing it
    - name: database # No setting needed - will use the node default
```

### Notes/Constraints/Caveats

- **cgroup v1 limitations**: On systems using cgroup v1, the `Group` mode cannot be enforced as cgroup v1 lacks the `memory.oom.group` mechanism. Kubelet will fail container creation with a clear event if `Group` is requested on such nodes, prompting the workload to be rescheduled on cgroup v2 capacity.
- **Windows incompatibility**: This feature is Linux-specific. Pods that target Windows nodes (`spec.os.name=windows`) and set `oomKillMode` will be rejected during admission with a validation error.
- **Container runtime support**: Requires container runtime support for setting `memory.oom.group`. Most modern runtimes (containerd, CRI-O) support this on cgroup v2.

### Risks and Mitigations

**Risk**: Inconsistent behavior between cgroup v1 and v2 environments.

- **Mitigation**: Clear documentation about platform limitations, validation warnings when `Group` is used on cgroup v1.

**Risk**: User confusion about the interaction between container and kubelet settings.

- **Mitigation**: Comprehensive documentation with examples, clear precedence rules, kubectl describe showing effective OOM mode.

**Risk**: Performance impact from additional cgroup configuration.

- **Mitigation**: The setting is applied once at container creation, no runtime overhead.

## Design Details

### API Changes

#### Container API

The changes will be made to the Container struct in `staging/src/k8s.io/api/core/v1/types.go`:

```go
// Container represents a single container that is expected to be run on the host.
type Container struct {
    // ... existing fields ...

    // OOMKillMode specifies how the OOM killer behaves for this container.
    // +featureGate=ContainerOOMKillMode
    // +optional
    OOMKillMode *OOMKillMode `json:"oomKillMode,omitempty" protobuf:"bytes,26,opt,name=oomKillMode"`
}
```

The same field applies to `EphemeralContainer` for consistency.

### Kubelet Implementation

The kubelet will be modified in `pkg/kubelet/kuberuntime/kuberuntime_container_linux.go` to respect the container-level setting:

```go
func (m *kubeGenericRuntimeManager) generateLinuxContainerResources(
    ctx context.Context,
    pod *v1.Pod,
    container *v1.Container,
    enforceMemoryQoS bool,
) *runtimeapi.LinuxContainerResources {
    // ... existing code ...

    // Optional info when a node-level default is configured
    if utilfeature.DefaultFeatureGate.Enabled(features.ContainerOOMKillMode) &&
       m.singleProcessOOMKill != nil {
        klog.V(2).Info("Using kubelet --single-process-oom-kill as node-level default; " +
                       "container oomKillMode overrides when set.")
    }

    // Determine OOM kill behavior
    var useGroupKill bool

    if container.OOMKillMode != nil {
        // Container-level setting takes precedence
        switch *container.OOMKillMode {
        case v1.OOMKillModeSingle:
            useGroupKill = false
        case v1.OOMKillModeGroup:
            if !isCgroup2UnifiedMode() {
                klog.Warningf("Container %s/%s requests Group mode but system uses cgroup v1, falling back to single process",
                    pod.Name, container.Name)
                useGroupKill = false
            } else {
                useGroupKill = true
            }
        }
    } else {
        // Fall back to kubelet configuration when set; otherwise use system default
        if utilfeature.DefaultFeatureGate.Enabled(features.ContainerOOMKillMode) &&
           m.singleProcessOOMKill != nil {
            useGroupKill = isCgroup2UnifiedMode() && !ptr.Deref(m.singleProcessOOMKill, false)
        } else {
            useGroupKill = isCgroup2UnifiedMode()
        }
    }

    // Apply the configuration
    if isCgroup2UnifiedMode() && useGroupKill {
        resources.Unified = map[string]string{
            "memory.oom.group": "1",
        }
    }

    return &resources
}
```

### Validation Rules

API validation will be added in `pkg/apis/core/validation/validation.go`:

```go
func ValidateContainer(podSpec *core.PodSpec, container *core.Container, path *field.Path) field.ErrorList {
    allErrs := field.ErrorList{}
    // ... existing validation ...

    if container.OOMKillMode != nil {
        validModes := sets.NewString(
            string(core.OOMKillModeSingle),
            string(core.OOMKillModeGroup),
        )
        if !validModes.Has(string(*container.OOMKillMode)) {
            allErrs = append(allErrs, field.Invalid(
                path.Child("oomKillMode"),
                *container.OOMKillMode,
                fmt.Sprintf("must be one of %v", validModes.List()),
            ))
        }

        if podSpec.OS != nil && podSpec.OS.Name == core.WindowsOS {
            allErrs = append(allErrs, field.Forbidden(
                path.Child("oomKillMode"),
                "oomKillMode is not supported for Windows pods",
            ))
        }
    }

    return allErrs
}
```

Platform-specific validation in `pkg/kubelet/apis/config/validation/validation_linux.go`:

```go
func validateOOMKillMode(kc *kubeletconfig.KubeletConfiguration, container *v1.Container) error {
    if container.OOMKillMode != nil && *container.OOMKillMode == v1.OOMKillModeGroup {
        if !isCgroup2UnifiedMode() {
            return fmt.Errorf("oomKillMode=Group requires cgroup v2 support on the node")
        }
    }
    return nil
}
```

### CRI Changes

No CRI changes are required.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

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
-->

- Test cases:
  - Valid values (`Single`, `Group`)
  - Invalid values rejected
  - Container setting overrides kubelet flag
  - Fallback to kubelet flag when nil
  - cgroup v1 fallback behavior

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow testing interactions between controllers.

This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- Feature gate enable/disable: verify field is ignored when disabled
- Container and kubelet setting interaction
- cgroup configuration verification

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- Single-process kill behavior with memory-intensive workload
- Group kill behavior with multi-process container
- Multi-container pods with different OOM modes
- cgroup v1/v2 compatibility
- Upgrade/downgrade scenarios

### Graduation Criteria

#### Alpha (v1.35)

- [ ] Implement the `oomKillMode` field behind the `ContainerOOMKillMode` feature gate
- [ ] Manual testing on both cgroup v1 and v2 systems
- [ ] Documentation of the feature in k/website
- [ ] Integration with existing `singleProcessOOMKill` flag verified
- [ ] Feature works with containerd and CRI-O runtimes
- [ ] Documentation clarifies precedence between container-level config and the kubelet flag

#### Beta (v1.36)

- [ ] Feature gate enabled by default (can still be disabled)
- [ ] Comprehensive e2e tests including:
  - Multi-container pods with mixed OOM modes
  - Behavior verification under actual OOM conditions
  - cgroup v1 fallback scenarios
- [ ] No critical bugs reported during Alpha (1 release cycle)
- [ ] kubectl support for displaying effective OOM mode in `describe pod`

#### GA (v1.38)

- [ ] Feature in Beta for at least 2 releases (v1.36, v1.37)
- [ ] Feature gate locked to enabled
- [ ] Re-evaluate deprecation status of `singleProcessOOMKill` based on adoption and feedback

### Upgrade / Downgrade Strategy

**Upgrade**:

- New field is optional and defaults to existing behavior
- Existing pods continue to work unchanged
- Feature gate allows gradual rollout

**Downgrade**:

- Field is ignored by older kubelet versions
- Pods continue to run with node-default OOM behavior
- No data loss or corruption

### Version Skew Strategy

- **API Server ↔ Kubelet**: API server can be newer than kubelet. Older kubelet ignores the new field.
- **Kubelet ↔ Container Runtime**: No new CRI fields required, uses existing Unified map.
- **kubectl ↔ API Server**: Older kubectl won't show the field but doesn't break.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- **Feature gate name**: ContainerOOMKillMode
- **Components depending on the feature gate**: kubelet
- **Will enabling / disabling the feature require downtime?**: No
- **Are there any tests for feature enablement/disablement?**: Yes, unit and e2e tests

###### Does enabling the feature change any default behavior?

No. The feature only takes effect when the new field is explicitly set.

###### Can the feature be disabled once it has been enabled?

Yes, via the feature gate. Existing pods with the field set will have it ignored.

###### What happens if we reenable a previously rolled back feature?

The field will be respected again for new pods. Existing pods are unaffected until recreated.

###### Are there any tests for feature enablement/disablement?

Yes, tests will verify that:

- With feature disabled, field is ignored
- With feature enabled, field is respected
- Toggling doesn't affect running pods

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail?

- Misconfiguration of the field value (caught by validation)
- Incompatibility with cgroup v1 (falls back gracefully with warning)

###### What specific metrics should inform a rollback?

- Increased pod restart rates
- OOM kill events not matching expected behavior
- Container runtime errors related to cgroup configuration

###### Were upgrade and rollback tested?

Will be tested in Beta phase with cluster upgrade/downgrade scenarios.

###### Is the rollout accompanied by any deprecations and/or removals?

No immediate removals. The `singleProcessOOMKill` kubelet flag remains supported as a node-level default.

### Monitoring Requirements

###### How can an operator determine if this feature is in use?

- Check pods: `kubectl get pods -A -o json | jq '.items[].spec.containers[] | select(.oomKillMode != null)'`
- Metrics: `container_oom_kill_mode_total{mode="Single|Group"}`
- Node inspection: `crictl inspect <container-id> | grep "memory.oom.group"`

###### How can someone using this cluster tell that this feature is working?

- **Configuration**: `kubectl describe pod <pod-name>` shows `OOM Kill Mode` field
- **Runtime**: Check `/sys/fs/cgroup/memory/<container-cgroup>/memory.oom.group` (0=single, 1=group)
- **Behavior**: Monitor whether single process or all processes are killed on OOM

###### What are the reasonable SLOs?

- **Configuration Accuracy**: 100% of containers must have correct `memory.oom.group` setting matching their configuration
- **Performance Impact**: <1ms additional latency for container creation
- **Feature Reliability**: 99.99% of OOM events handled according to configured mode

###### What are the SLIs?

```promql
# Percentage of containers with correct OOM configuration
(container_oom_kill_mode_configured_total / container_total) * 100

# OOM events by mode (for behavior validation)
rate(container_oom_events_by_mode[5m])

# Configuration errors
rate(container_oom_config_errors_total[5m])

# Time to apply OOM configuration
histogram_quantile(0.99, container_oom_config_duration_seconds)
```

###### Are there any missing metrics?

New metrics to be added:

- `container_oom_kill_mode_total`: Number of containers by OOM kill mode
- `container_oom_events_by_mode`: OOM kill events by effective kill mode
- `container_oom_config_errors_total`: Total number of OOM configuration errors

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No additional services required.

###### Does this feature depend on any specific runtime features?

- cgroup v2 for full functionality
- Container runtime support for setting cgroup parameters (already standard)

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new API calls. The field is part of the Pod spec.

###### Will enabling / using this feature result in introducing new API types?

Only a new enum type `OOMKillMode` with two values.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of objects?

Minimal increase in Container object size (one optional field).

###### Will enabling / using this feature result in increasing time taken by any operation?

No measurable impact. Configuration is applied once at container creation.

###### Will enabling / using this feature result in non-negligible increase of resource usage?

No additional resource usage.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No impact. The configuration is already part of the pod spec on the node.

###### What are other known failure modes?

- cgroup v1 incompatibility: Logged as warning, falls back to single process
- Invalid field value: Rejected at API validation

###### What steps should be taken if SLOs are not being met?

1. Check logs for cgroup configuration errors
2. Verify container runtime support
3. Disable feature gate if necessary

## Implementation History

TBD

## Drawbacks

- Adds complexity to the Container API
- Potential for user confusion about OOM behavior
- Platform-specific limitations (cgroup v1, Windows)
- Another field for users to understand and configure

## Alternatives

- **Pod-level configuration**: Rejected as too coarse-grained for multi-container pods

- **Annotation-based approach**:

  ```yaml
  annotations:
    node.kubernetes.io/oom-kill-mode: "Single"
  ```

  Rejected: Annotations are not the proper place for functional configuration

- **NRI plugin**: Provide an NRI plugin that sets `memory.oom.group` for target containers
  Rejected: Requires node-level plugin deployment and runtime support, adds operational complexity, and is not portable across clusters.

- **Privileged init container**: Configure `memory.oom.group` from a privileged init container
  Rejected: Requires privileged access, is fragile across runtimes, and bypasses Kubernetes API validation.

- **Extend existing Resources field**:

  ```yaml
  resources:
    limits:
      memory: "1Gi"
    oomPolicy:
      mode: "Single"
  ```

  Rejected: Mixes resource quantities with behavior configuration

- **Do nothing**: Continue with node-level configuration only
  Rejected: Doesn't meet the needs of diverse workloads

## Infrastructure Needed(Optional)

None

## References

[^1]: https://github.com/kubernetes/kubernetes/pull/117793#issuecomment-1843606901

[^2]: https://github.com/kubernetes/kubernetes/pull/117793#issuecomment-2059012668

[^3]: https://github.com/kubernetes/kubernetes/pull/122813#issuecomment-1953290374

[^4]: https://github.com/kubernetes/kubernetes/pull/117793#issuecomment-1551382249
