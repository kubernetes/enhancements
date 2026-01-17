# KEP-5808: Native Histogram Support for Kubernetes Metrics

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Platform Engineer Optimizing Monitoring Costs](#story-1-platform-engineer-optimizing-monitoring-costs)
    - [Story 2: SRE Detecting Performance Regressions](#story-2-sre-detecting-performance-regressions)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Dual Exposition Strategy](#dual-exposition-strategy)
  - [Implementation Phases](#implementation-phases)
  - [Prometheus Version Compatibility](#prometheus-version-compatibility)
    - [Special Concern: Prometheus 2.x Users](#special-concern-prometheus-2x-users)
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
  - [Increase Classic Histogram Bucket Count](#increase-classic-histogram-bucket-count)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
- [References](#references)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website


## Summary

This KEP proposes adding support for [Prometheus Native Histograms](https://prometheus.io/docs/specs/native_histograms/) to Kubernetes component metrics. Starting with Prometheus v3.8.0, native histograms are supported as a stable feature. Native histograms use exponential bucket boundaries instead of fixed boundaries, providing significant storage efficiency (~10x reduction in time series count per histogram), improved query performance, and finer-grained visibility into distributions while maintaining full backward compatibility with existing monitoring infrastructure through a dual exposition strategy.

The implementation introduces a feature gate (`NativeHistograms`) to provide safe rollout and rollback capabilities. When enabled, Kubernetes components will expose histogram metrics in both classic and native formats simultaneously, ensuring existing dashboards and alerts continue to function while users can migrate to native histograms at their own pace. Rollback is handled primarily through Prometheus-side configuration (for Prometheus 3.x users) or via the K8s feature gate.

## Motivation

Kubernetes exposes hundreds of histogram metrics across its control plane components. These metrics are essential for monitoring cluster health, debugging performance issues, and ensuring service level objectives are met. However, classic Prometheus histograms have inherent limitations:

1. **Storage overhead**: Each classic histogram creates multiple time series (one per bucket plus `_count` and `_sum`), leading to high storage costs at scale
2. **Fixed bucket boundaries**: Predefined buckets may not align well with actual data distributions, causing accuracy issues and rendering bucket boundaries useless. For example, if a histogram uses default buckets like `[0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]` seconds, a request completing in 1µs (0.000001s) falls into the same `le="0.005"` bucket as a request completing in 4ms—a 4000x difference in latency becomes indistinguishable. Similarly, all requests between 1s and 2.5s are grouped together, hiding important performance variations

Prometheus Native Histograms, introduced in Prometheus 2.40, address these limitations using exponential bucket boundaries with automatic adjustment. Kubernetes should support this modern, more efficient histogram format.

### Goals

1. Enable Kubernetes components to expose metrics in Prometheus Native Histogram format
2. Maintain full backward compatibility with existing monitoring infrastructure
3. Provide a safe, gradual rollout path with extended testing periods

### Non-Goals

1. Removing classic histogram exposition format
2. Remove existing histogram metrics

## Proposal

Add native histogram support to the `component-base/metrics` package with:

1. **Feature Gate**: A new `NativeHistograms` feature gate controlling whether K8s exposes native histogram format
2. **Extended HistogramOpts**: New fields in `HistogramOpts` for native histogram configuration
3. **Dual Exposition**: When enabled, expose both classic and native histogram formats

The control model is intentionally simple:
- K8s-side: Feature gate controls whether native histograms are exposed
- Prometheus-side: Per-job `scrape_native_histograms` ([ref](https://prometheus.io/docs/specs/native_histograms/#scrape-configuration)) controls what Prometheus ingests (Prometheus 3.x)

### User Stories

#### Story 1: Platform Engineer Optimizing Monitoring Costs

As a platform engineer managing a large Kubernetes fleet, I want to reduce the storage costs of my Prometheus infrastructure. With native histograms, I can achieve ~10x reduction in time series count for histogram metrics, significantly reducing storage and improving query performance without changing my existing dashboards.

#### Story 2: SRE Detecting Performance Regressions

As an SRE responsible for cluster reliability, I need to detect performance regressions accurately. With classic histograms, a latency regression from 1ms to 50ms might go unnoticed because both values fall into the same `le="0.1"` bucket. Native histograms' exponential buckets provide much finer resolution, enabling me to reliably detect even small performance regressions and set precise SLO thresholds.

### Notes/Constraints/Caveats

1. **External Dependency**: Native histogram support in Kubernetes depends on Prometheus server scrape settings. Users must configure the following in their Prometheus scrape config during transition to receive both formats
   1. `scrape_native_histograms: true`
   2. `always_scrape_classic_histograms: true`


### Risks and Mitigations

1. **Silent Dashboard/Alert Failures on Upgrade**: When upgrading to a Kubernetes version where `NativeHistograms` feature gate becomes default ON, users with `scrape_native_histograms: true` in Prometheus who forget to also set `always_scrape_classic_histograms: true` can experience silent failures:
   - Classic `_bucket`, `_count`, `_sum` metrics will no longer be ingested
   - Existing dashboards using `histogram_quantile(..._bucket...)` queries will show no data or stale data
   - Alerts based on classic histogram queries will stop firing


The dashboard breakage risk depends on a combination of Prometheus settings:

| `scrape_native_histograms` | `always_scrape_classic_histograms` | Result                                                           |
| -------------------------- | ---------------------------------- | ---------------------------------------------------------------- |
| `false` (default)          | N/A                                | **SAFE**: Classic only                                           |
| `true`                     | `true`                             | **SAFE**: Both formats (recommended during migration)            |
| `true`                     | `false` (default)                  | **RISK**: Native only (safe only after full dashboard migration) |

**Migration workflow:**

The `always_scrape_classic_histograms` setting addresses a chicken-and-egg problem: users cannot migrate dashboards to native histogram queries without first enabling native histogram ingestion, but enabling ingestion without classic format would break existing dashboards.

Recommended approach:
1. **Enable both formats**: Set `scrape_native_histograms: true` AND `always_scrape_classic_histograms: true`
2. **Migrate dashboards/alerts**: Update queries from classic (`histogram_quantile(..._bucket...)`) to native histogram functions
3. **Verify in staging**: Ensure all dashboards and alerts work with native histogram queries
4. **Disable classic scraping**: Once migration is complete and verified, set `always_scrape_classic_histograms: false` to reduce storage overhead

**Mitigation:**
   - Verify Prometheus version (3.x recommended for per-job control)
   - Set `always_scrape_classic_histograms: true` for all K8s scrape jobs during migration
   - Test dashboard queries in staging before production upgrade
   - Documentation must clearly state that users enabling `scrape_native_histograms` should **also** set `always_scrape_classic_histograms: true` until dashboard migration is complete

## Design Details

Kubernetes metrics use the `component-base/metrics` package which wraps `prometheus/client_golang`. Currently:

- `HistogramOpts` only supports classic `Buckets []float64`
- No configuration path for native histogram options
- Hundreds of histogram metrics across control plane components

### Dual Exposition Strategy

When native histograms are enabled, Kubernetes will expose **BOTH** formats:

```
# Classic format (always present)
apiserver_request_duration_seconds_bucket{le="0.005"} 1000
apiserver_request_duration_seconds_bucket{le="0.01"} 2000
...
apiserver_request_duration_seconds_bucket{le="+Inf"} 10000
apiserver_request_duration_seconds_count 10000
apiserver_request_duration_seconds_sum 450.5

# Native format (when enabled, requires protobuf exposition)
apiserver_request_duration_seconds{} <native histogram encoding>
```

This ensures:
- Existing dashboards continue to work
- Users can migrate queries at their own pace
- Prometheus stores whichever format it's configured for

### Implementation Phases

We will extend the `HistogramOpts` struct in the `component-base/metrics` histogram wrapper to include the native histogram configuration options required by `prometheus/client_golang`. We will set sensible global defaults (e.g., bucket factor of 1.1, max 160 buckets) that apply to all histograms when the feature is enabled, while allowing per-metric overrides if needed:

```go
type HistogramOpts struct {
    // Existing fields...
    Namespace         string
    Subsystem         string
    Name              string
    Help              string
    ConstLabels       map[string]string
    Buckets           []float64
    DeprecatedVersion string
    StabilityLevel    StabilityLevel
    
    // New: Native histogram configuration (used when feature enabled)
    // If nil, uses global defaults when native histograms are enabled.
    NativeHistogramBucketFactor     *float64
    NativeHistogramZeroThreshold    *float64
    NativeHistogramMaxBucketNumber  *uint32
}
```

We will update the conversion function to pass native histogram options to the underlying Prometheus library when the feature gate is enabled:

```go
func (o *HistogramOpts) toPromHistogramOpts() prometheus.HistogramOpts {
    opts := prometheus.HistogramOpts{
        Namespace:   o.Namespace,
        Subsystem:   o.Subsystem,
        Name:        o.Name,
        Help:        o.Help,
        ConstLabels: o.ConstLabels,
        Buckets:     o.Buckets,  // Always keep classic buckets
    }
    
    if utilfeature.DefaultFeatureGate.Enabled(features.NativeHistograms) {
        // Use defaults, allow per-metric override if specified
        factor := 1.1  // Default bucket growth factor
        if o.NativeHistogramBucketFactor != nil {
            factor = *o.NativeHistogramBucketFactor
        }
        
        opts.NativeHistogramBucketFactor = factor
        opts.NativeHistogramMaxBucketNumber = 160  // Default max buckets
        opts.NativeHistogramZeroThreshold = prometheus.DefNativeHistogramZeroThreshold
    }
    
    return opts
}
```

### Prometheus Version Compatibility

Native histogram support and configuration varies significantly across Prometheus versions:

**Prometheus < 2.40:**
- Cannot ingest native histograms at all
- K8s exposing native histograms has no effect—Prometheus ignores them
- Classic `_bucket`, `_count`, `_sum` metrics continue to work
- **Action needed:** None, but no benefit from native histograms

**Prometheus 2.40 - 2.x:**
```bash
# Enable native histogram support globally
prometheus --enable-feature=native-histograms

# This is all-or-nothing: ALL scrape jobs will attempt to ingest native histograms
# No per-job control available
```
- Higher risk: Cannot selectively enable for K8s while keeping classic for other targets
- If enabled, dashboards for ALL targets using classic histogram queries may break

**Prometheus 3.0 - 3.7:**
```yaml
# Per-job configuration (recommended)
scrape_configs:
  - job_name: 'kubernetes-apiservers'
    scrape_native_histograms: true
    always_scrape_classic_histograms: true  # Keep classic during transition

# OR use feature flag for global default (still supported)
# prometheus --enable-feature=native-histograms
```
- Per-job control available
- Can enable native histograms for K8s while keeping other jobs on classic only
- Feature flag still works as global default

**Prometheus 3.8:**
```yaml
# Per-job configuration (required for fine-grained control)
scrape_configs:
  - job_name: 'kubernetes-apiservers'
    scrape_native_histograms: true
    always_scrape_classic_histograms: true

# Feature flag now ONLY sets scrape_native_histograms default to true
# prometheus --enable-feature=native-histograms  # Sets default, not recommended
```
- Feature flag's only remaining effect: sets `scrape_native_histograms: true` as default
- Per-job settings override the default
- Transition period: migrate from flag to explicit per-job config

**Prometheus 3.9+:**
```yaml
# Per-job configuration (only method)
scrape_configs:
  - job_name: 'kubernetes-apiservers'
    scrape_native_histograms: true
    always_scrape_classic_histograms: true
```
- Feature flag fully deprecated and removed
- Must use per-job `scrape_native_histograms` and `always_scrape_classic_histograms`
- Default for both settings is `false`

#### Special Concern: Prometheus 2.x Users

Prometheus 2.x users with `--enable-feature=native-histograms` enabled are in a **difficult position**:

**Scenario:**
- User has `prometheus --enable-feature=native-histograms` (maybe enabled for other workloads that benefit from native histograms)
- K8s upgrades start exposing native histograms

**Problem:**
- Global flag means ALL jobs ingest native format
- No way to keep native for app X but classic for K8s

**Mitigation options:**

1. **Turn off Prometheus feature flag**
   - Loses native histograms for ALL workloads (not just K8s)

2. **Disable K8s feature gate:** `--feature-gates=NativeHistograms=false`
   - Requires K8s component restarts (may be slow/disruptive)
   - For managed K8s, may not be possible

3. **Upgrade to Prometheus 3.x**
   - Major version upgrade, may not be quick/easy
   - Then can use per-job `scrape_native_histograms: false`

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

Existing histogram metric tests should be extended to verify dual exposition behavior when the feature gate is enabled.

##### Unit tests

- `staging/src/k8s.io/component-base/metrics`: Test `toPromHistogramOpts()` with feature gate enabled/disabled
- Test per-metric native histogram option overrides
- Test that classic buckets are always present

##### Integration tests

- Verify metrics endpoint serves both formats when enabled
- Verify classic buckets are always present regardless of feature state

##### e2e tests

- Scrape metrics with Prometheus (native histogram support enabled)
- Verify both formats are queryable

### Graduation Criteria

#### Alpha

- Feature implemented behind `NativeHistograms` feature flag
- Initial unit tests completed and enabled
- Basic documentation available

#### Beta

- Gather feedback from early adopters
- Comprehensive integration tests in place
- E2E tests covering upgrade/downgrade scenarios
- Documentation updated with:
  - Migration guide
  - Troubleshooting procedures
  - Prometheus configuration examples
  - Clear Prometheus 2.x limitations
- Performance benchmarks completed showing no regression

#### GA

- TBD based on feedback

### Upgrade / Downgrade Strategy

**Upgrade:**
1. Kubernetes upgrade does not change monitoring behavior if feature gate is off
2. When feature gate is enabled:
   - Classic histogram format continues to be exposed
   - Native format is additionally exposed

**Enabling Native Histograms (Opt-in):**

```bash
# 1. Ensure Prometheus is ready (Prometheus 3.x)
# prometheus.yml
scrape_configs:
  - job_name: 'kubernetes-apiservers'
    scrape_native_histograms: true           # Ingest native histograms
    always_scrape_classic_histograms: true   # CRITICAL: Keep classic during transition

# For older Prometheus (2.40-2.x), use global feature flag:
# --enable-feature=native-histograms

# 2. Enable feature gate in Kubernetes
--feature-gates=NativeHistograms=true
```

### Version Skew Strategy

Native histogram support is independent per component. Each component's metrics are independent, no coordination required.
Some components may expose native histograms while others don't. This is acceptable as Prometheus scrapes each target independently.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `NativeHistograms`
  - Components depending on the feature gate: kube-apiserver, kube-controller-manager, kube-scheduler, kubelet, kube-proxy
- [x] Other
  - Describe the mechanism: Prometheus 3.x per-job `scrape_native_histograms: false` stops ingestion without K8s changes
  - Will enabling / disabling the feature require downtime of the control plane? For feature gate changes, yes (component restart required). For Prometheus config changes, no.
  - Will enabling / disabling the feature require downtime or reprovisioning of a node? No

###### Does enabling the feature change any default behavior?

When enabled, the metrics endpoint will expose an additional native histogram encoding alongside the existing classic histogram format. The classic format (`_bucket`, `_count`, `_sum`) remains unchanged and always present.

Users with Prometheus configured to prefer native histograms will see the data stored in native format. Users without native histogram support enabled in Prometheus see no change.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling can be done via:
1. Prometheus config: `scrape_native_histograms: false` per job (fastest, Prometheus 3.x only, no K8s restart)
2. Feature gate: `--feature-gates=NativeHistograms=false` (requires component restart)

When K8s feature is disabled, only classic histogram format is exposed. When Prometheus stops ingesting native histograms, it resumes scraping classic format on next scrape interval. No data loss occurs; historical data in Prometheus remains queryable.

###### What happens if we reenable the feature if it was previously rolled back?

Native histogram exposition resumes. No special handling required. Prometheus will begin storing native histograms again if configured to do so.

###### Are there any tests for feature enablement/disablement?

Yes, unit tests will verify:
- `toPromHistogramOpts()` returns correct configuration based on feature gate state
- Toggling feature gate changes histogram configuration appropriately
- Classic buckets are always present regardless of feature gate state

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

**Rollout failure scenarios:**
1. Prometheus too old to understand native histograms - Prometheus ignores native format, stores classic
2. Dashboard queries not updated - Dashboards continue to work with classic format
3. Memory pressure from additional histogram storage - Configurable via `--native-histogram-max-buckets`

**Impact on workloads:** None. This feature only affects metrics exposition, not workload behavior.

###### What specific metrics should inform a rollback?

- Prometheus scrape errors increasing for Kubernetes targets
- Significant increase in `process_resident_memory_bytes` for control plane components
- Increase in `/metrics` endpoint latency
- Dashboard queries failing 

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be tested as part of beta graduation.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No. Classic histogram format is not deprecated.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- Check component logs for "native histograms enabled" message
- Query `kubernetes_feature_enabled` metric with label `name=NativeHistograms` (value 1 = enabled)
- Scrape `/metrics` endpoint with protobuf format and verify native histogram encoding is present

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as last resort)
  - Details: Query the metrics endpoint with `Accept: application/vnd.google.protobuf` header and verify native histogram encoding is present for histogram metrics. Note: Native histograms are only supported in protobuf exposition format, not in text-based formats.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- Metrics endpoint latency should not increase significantly when native histograms are enabled
- All existing classic histogram queries continue to function

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `process_resident_memory_bytes` (existing, monitor for unexpected increases)
  - Components exposing the metric: All control plane components

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No cluster services required. However, to utilize native histograms:

- **Prometheus 2.40+** (experimental) or **Prometheus 3.0+** (stable)
  - Usage description: Required to scrape and store native histogram format
  - Configuration:
    - Prometheus 2.40-2.x: `--enable-feature=native-histograms` (global)
    - Prometheus 3.x: `scrape_native_histograms: true` per scrape job (recommended)

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

The `/metrics` endpoint may take slightly longer to serialize when exposing both formats. This will be benchmarked during alpha/beta.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

**Memory:** Small increase for native histogram bucket storage. Bounded by `--native-histogram-max-buckets` (default: 160).

**CPU:** Negligible increase for histogram operations.

**Network:** Slight increase in `/metrics` response size when exposing both formats.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. This feature only affects in-memory histogram representation and metrics endpoint output.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No impact. Metrics exposition is independent of API server and etcd availability (except for the API server's own metrics).

###### What are other known failure modes?

- **Failure mode: Prometheus too old**
  - Detection: Prometheus logs errors about unknown metric format
  - Mitigations: Upgrade Prometheus to 2.40+ or disable native histograms
  - Diagnostics: Check Prometheus version; verify `--enable-feature=native-histograms` is set

- **Failure mode: Memory pressure from histogram storage**
  - Detection: `process_resident_memory_bytes` increasing; OOMKilled events
  - Mitigations: Disable native histogram ingestion in Prometheus; disable K8s feature gate
  - Diagnostics: Compare memory usage before/after enabling feature

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check if native histograms are enabled
2. Compare memory usage with baseline
3. Check `/metrics` endpoint latency
4. If issues detected, disable via Prometheus config (`scrape_native_histograms: false`) or K8s feature gate
5. File issue with memory/latency profiles

## Implementation History

- 2026-01-16: Initial KEP created

## Drawbacks

1. **Increased complexity**: Two histogram formats to maintain and test
2. **External dependency**: Full benefit requires Prometheus upgrade by users
3. **Memory overhead**: Small additional memory for native histogram storage

## Alternatives

### Increase Classic Histogram Bucket Count

Instead of adopting native histograms, we could increase the number of buckets in classic histograms to achieve finer granularity.

**Cons**:
- Each additional bucket creates a new time series, significantly increasing cardinality
- This directly increases Prometheus storage costs, memory usage, and query latency
- The cardinality explosion from more classic buckets negates any observability benefits

## Infrastructure Needed (Optional)

None.

## References

- [Prometheus Configuration - scrape_native_histograms](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#scrape_config)
- [prometheus/client_golang Native Histogram Support](https://github.com/prometheus/client_golang)
- [Prometheus Feature Flags](https://prometheus.io/docs/prometheus/latest/feature_flags/)
