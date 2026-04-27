# KEP-5000: Configurable cAdvisor Metrics Collection

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Implementation](#implementation)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements]
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website]

## Summary

This KEP proposes adding a `cadvisor` configuration section to KubeletConfiguration that allows operators to selectively disable expensive cAdvisor metric collectors. The primary use case is disabling `ProcessMetrics` collection, which scans `/proc` for every thread in every container and causes extreme CPU overhead on high-density nodes.

## Motivation

Kubelet's embedded cAdvisor collects container metrics including process/thread information. On nodes with high pod density (100+ pods), the `ProcessMetrics` collector causes significant CPU overhead by scanning `/proc` for every thread in every container.

### Production Evidence

Testing at Wix on EKS 1.31 clusters with ~200 pods/node showed:

| Metric | Before | After (ProcessMetrics disabled) | Improvement |
|--------|--------|--------------------------------|-------------|
| Kubelet CPU | 1332% | 8% | **99.4% reduction** |
| Node Total CPU | 44% | 13% | 70% reduction |
| Cores freed | - | ~10 cores | Per node |

The `ProcessMetrics` collector scans:
- Every process in every container
- Every thread in every process  
- Every file descriptor in every process

With 200 pods × 200 threads/pod = 40,000+ `/proc` reads per housekeeping cycle (every 10 seconds).

### Community Interest

- [kubernetes/kubernetes#123340](https://github.com/kubernetes/kubernetes/issues/123340) - Open, triage/accepted, 34+ comments
- [kubernetes/kubernetes#99183](https://github.com/kubernetes/kubernetes/issues/99183) - Similar request
- [kubernetes/kubernetes#101079](https://github.com/kubernetes/kubernetes/issues/101079) - Similar request

### Goals

1. Allow operators to disable `ProcessMetrics` collection via KubeletConfiguration
2. Reduce kubelet CPU usage on high-density nodes
3. Maintain backward compatibility (all metrics enabled by default)
4. Provide a foundation for future metric configurability

### Non-Goals

1. Deprecating cAdvisor (separate effort)
2. Configuring all cAdvisor parameters (scope limited to metrics selection)
3. Configuring housekeeping interval (hardcoded, separate issue)
4. CRI stats provider configuration (separate KEP)

## Proposal

Add a new `CAdvisorConfiguration` struct to KubeletConfiguration with an `includedMetrics` field that allows operators to selectively disable metric collectors.

### User Stories

**Story 1: High-Density Node Operator**

As a platform operator running 200+ pods per node, I want to disable ProcessMetrics collection so that kubelet doesn't consume 10+ CPU cores scanning `/proc`.

```yaml
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
cadvisor:
  includedMetrics:
    processMetrics: false
```

**Story 2: Metrics-Aware Operator**

As an operator who doesn't use `container_threads`, `container_processes`, or `container_file_descriptors` metrics, I want to disable their collection to reduce overhead without affecting other metrics.

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Users disable metrics they need | Default: all enabled. Document which metrics are affected. |
| Confusion with CRI stats | Clear documentation. Feature gate for visibility. |
| API surface expansion | Minimal API - single struct with boolean fields. |

## Design Details

### API Changes

Add to `pkg/kubelet/apis/config/types.go`:

```go
// KubeletConfiguration contains the configuration for the Kubelet
type KubeletConfiguration struct {
    // ... existing fields ...
    
    // CAdvisor contains configuration for the embedded cAdvisor.
    // +optional
    CAdvisor CAdvisorConfiguration
}

// CAdvisorConfiguration contains settings for cAdvisor metrics collection.
type CAdvisorConfiguration struct {
    // IncludedMetrics specifies which cAdvisor metric collectors are enabled.
    // All collectors are enabled by default for backward compatibility.
    // +optional
    IncludedMetrics CAdvisorIncludedMetrics
}

// CAdvisorIncludedMetrics specifies which cAdvisor metric collectors to enable.
// All fields default to true for backward compatibility.
type CAdvisorIncludedMetrics struct {
    // ProcessMetrics enables collection of process/thread metrics.
    // These metrics scan /proc for every thread in every container.
    // Disabling significantly reduces kubelet CPU on high-density nodes.
    // Affected metrics: container_processes, container_threads,
    // container_file_descriptors, container_sockets, container_ulimits_soft/hard.
    // Default: true
    // +optional
    ProcessMetrics *bool
}
```

### Implementation

Modify `pkg/kubelet/cadvisor/cadvisor_linux.go`:

```go
func New(imageFsInfoProvider ImageFsInfoProvider, rootPath string, cgroupRoots []string, 
         usingLegacyStats, localStorageCapacityIsolation bool,
         cadvisorConfig *kubeletconfig.CAdvisorConfiguration) (Interface, error) {
    
    includedMetrics := cadvisormetrics.MetricSet{
        cadvisormetrics.CpuUsageMetrics:     struct{}{},
        cadvisormetrics.MemoryUsageMetrics:  struct{}{},
        cadvisormetrics.CpuLoadMetrics:      struct{}{},
        cadvisormetrics.DiskIOMetrics:       struct{}{},
        cadvisormetrics.NetworkUsageMetrics: struct{}{},
        cadvisormetrics.AppMetrics:          struct{}{},
        cadvisormetrics.OOMMetrics:          struct{}{},
    }
    
    // ProcessMetrics - default enabled for backward compatibility
    if cadvisorConfig == nil || 
       cadvisorConfig.IncludedMetrics.ProcessMetrics == nil ||
       *cadvisorConfig.IncludedMetrics.ProcessMetrics {
        includedMetrics[cadvisormetrics.ProcessMetrics] = struct{}{}
    }
    
    // ... rest of function
}
```

### Test Plan

#### Unit Tests

- `pkg/kubelet/cadvisor/cadvisor_linux_test.go`:
  - Test ProcessMetrics enabled by default (nil config)
  - Test ProcessMetrics enabled explicitly (true)
  - Test ProcessMetrics disabled (false)
  - Test backward compatibility with no config

#### Integration Tests

- Test kubelet starts with ProcessMetrics disabled
- Test metrics endpoint doesn't include process metrics when disabled

#### e2e Tests

- Test that disabling ProcessMetrics reduces kubelet CPU usage
- Test that other metrics still work when ProcessMetrics disabled

### Graduation Criteria

#### Alpha (v1.33)

- Feature gate `ConfigurableCAdvisorMetrics` disabled by default
- KubeletConfiguration API added
- Unit tests passing
- Documentation added

#### Beta (v1.34)

- Feature gate enabled by default
- E2e tests passing
- Gather feedback from early adopters
- No major issues reported

#### GA (v1.35)

- Feature gate locked to enabled
- At least 2 releases of beta usage
- Production usage documented

### Upgrade / Downgrade Strategy

**Upgrade**: No action required. All metrics remain enabled by default.

**Downgrade**: If downgrading from a version where ProcessMetrics was disabled, the metrics will be collected again. No data loss or corruption.

### Version Skew Strategy

This feature only affects kubelet. No coordination with other components required.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

**How can this feature be enabled / disabled in a live cluster?**

- Feature gate: `ConfigurableCAdvisorMetrics`
- Component: kubelet
- Requires kubelet restart

**Does enabling the feature change any default behavior?**

No. All metrics remain enabled by default.

**Can the feature be disabled once it has been enabled?**

Yes. Remove the configuration and restart kubelet. Metrics will be collected again.

### Scalability

**Will enabling / using this feature result in any new API calls?**

No.

**Will enabling / using this feature result in non-negligible increase of resource usage?**

No. This feature *reduces* resource usage.

## Implementation History

- 2026-01-09: KEP created

## Alternatives

### Alternative 1: Command-line flag

Add `--disable-cadvisor-metrics=process` flag.

**Rejected**: KubeletConfiguration is the preferred way to configure kubelet. Flags are being deprecated.

### Alternative 2: Expose all cAdvisor flags

Pass through all cAdvisor flags like `--disable_metrics`.

**Rejected**: Too broad API surface. cAdvisor is being phased out.

### Alternative 3: Wait for CRI stats

Wait for CRI stats provider to fully replace cAdvisor.

**Rejected**: CRI stats doesn't cover all metrics yet. Users need relief now.
