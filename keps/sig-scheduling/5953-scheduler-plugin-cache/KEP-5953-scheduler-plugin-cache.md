# KEP-5953: Caching Mechanism for InterPodAffinity and PodTopologySpread Plugins

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Generic Cache Framework](#generic-cache-framework)
  - [CachePlugin: Reserve-Phase Cache Coordination](#cacheplugin-reserve-phase-cache-coordination)
  - [InterPodAffinity Caching](#interpodaffinity-caching)
    - [Scoring Cache](#scoring-cache)
    - [Filtering Cache](#filtering-cache)
  - [PodTopologySpread Caching](#podtopologyspread-caching)
  - [Cache Invalidation Strategy](#cache-invalidation-strategy)
  - [Feature Gate](#feature-gate)
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
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Summary

This KEP proposes adding a caching mechanism to the `InterPodAffinity` and `PodTopologySpread` scheduler plugins to reduce redundant computations during the PreFilter/Filter and PreScore/Score phases. By memoizing expensive affinity term evaluations, topology spread constraint calculations, and pod-matching results, the scheduler can avoid recomputing these values for pods with identical affinity or topology spread configurations. This is particularly beneficial in clusters with large numbers of pods sharing the same `pod-template-hash` (e.g., Deployments, ReplicaSets) and complex affinity rules.

## Motivation

In large Kubernetes clusters, the `InterPodAffinity` and `PodTopologySpread` plugins are among the most CPU-intensive scheduler plugins. During each scheduling cycle, these plugins must:

1. **InterPodAffinity Scoring**: For each node, iterate over all existing pods, evaluate affinity/anti-affinity term matching, and compute topology-based scores. This involves label matching, namespace label lookups, and topology key evaluations — all of which are repeated for every pod being scheduled.

2. **InterPodAffinity Filtering**: For each node, iterate over all existing pods to evaluate required affinity/anti-affinity constraints against the incoming pod and existing pods' anti-affinity against the incoming pod.

3. **PodTopologySpread Scoring**: For each topology constraint, count matching pods across all nodes, which requires iterating over all pods on every node and evaluating label selectors.

When scheduling pods from the same Deployment or ReplicaSet, these computations are largely identical because the pods share the same labels, affinity rules, and topology spread constraints. The current implementation recomputes everything from scratch for every scheduling cycle, leading to:

- High CPU usage on the scheduler, especially in clusters with 5,000+ nodes
- Increased scheduling latency (P99) during burst creation of pods
- Redundant iterations over the same pod sets with the same selectors

By caching the intermediate results (topology score maps, pod-matching counts) and incrementally updating them when pods are added, removed, or rescheduled, we can significantly reduce the computational overhead.

### Goals

- Reduce CPU overhead in `InterPodAffinity` PreScore/PreFilter by caching topology score maps and affinity term evaluation results
- Reduce CPU overhead in `PodTopologySpread` PreScore by caching per-constraint pod counts
- Maintain scheduling correctness by invalidating and incrementally updating cache entries on relevant cluster events (Pod create/update/delete, Node label changes, Namespace label changes)
- Integrate with the scheduler's Reserve/Unreserve lifecycle to ensure cache consistency during pod binding
- Provide a safe opt-in mechanism via feature gate

### Non-Goals

- Caching results for the `NodeAffinity` or other plugins
- Changing the scheduling algorithm, scoring formulas, or filtering logic
- Modifying the scheduler framework interfaces
- Optimizing the scheduler's data snapshot mechanism
- Providing user-facing APIs or configuration for cache tuning

## Proposal

We propose introducing a generic cache framework (`cacheplugin` package) and dedicated caches for the `InterPodAffinity` and `PodTopologySpread` plugins. The key idea is that pods from the same workload (identified by `pod-template-hash` or namespace+label set) share identical affinity/topology configurations, so their scheduling computations can be reused.

### Cache Architecture

```
                    CachePlugin (ReservePlugin)
                    +-----------------------+
                    |  Reserve / Unreserve  |
                    +----------+------------+
                               | notifies all caches
              +----------------+----------------+
              v                v                v
    InterPodAffinity   InterPodAffinity   PodTopologySpread
    Scoring Caches     Filtering Caches   Scoring Cache
    +--------------+   +---------------+  +---------------+
    | Incoming Pod |   | Incoming Pod  |  | preCalRes     |
    | Existing Pod |   | Existing Pod  |  | cachedPods    |
    +--------------+   +---------------+  +---------------+
              |                |                |
              +-------- Informers -------------+
                  Pod / Node / Namespace
```

1. **Generic Cache (`CacheImpl[T, V]`)**: An LRU cache with read-write lock protection, configurable size, automatic expiration (1-minute TTL), and a rate-limited work queue for asynchronous event processing.

2. **CachePlugin**: A `ReservePlugin` implementation that coordinates cache updates during the Reserve and Unreserve phases. When a pod is assumed to be bound to a node, all caches are notified to update their pre-computed results.

3. **Plugin-specific caches**: Each plugin maintains caches keyed by pod identity (e.g., `pod-template-hash` for incoming pod caches, `namespace/labels` for existing pod caches).

### User Stories

#### Story 1: Large Deployment Rollout

As a cluster administrator performing a rolling update of a 1,000-replica Deployment across a 5,000-node cluster with pod anti-affinity rules, I want the scheduler to efficiently schedule new pods without re-evaluating the same affinity terms for each identical replica. With caching, only the first pod's scheduling cycle performs the full computation; subsequent pods reuse the cached topology scores with incremental updates for pods that have been scheduled since.

#### Story 2: Complex Topology Spread with High Pod Churn

As a platform team running a multi-tenant cluster with per-namespace topology spread constraints, I want the scheduler to handle high pod creation rates without CPU saturation. When 100 pods from the same ReplicaSet are created simultaneously, the topology spread pod counts should be computed once and reused, with incremental updates as pods are bound.

#### Story 3: Scheduler Performance at Scale

As an SRE managing a 10,000-node cluster, I want the scheduler's CPU usage to remain reasonable even with complex affinity and topology spread rules. The caching mechanism should reduce the O(nodes * pods) computation per scheduling cycle to an amortized O(1) lookup for pods with cached results.

### Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Cache staleness leading to incorrect scores | High | Incremental cache updates via Informer event handlers for Pod/Node/Namespace changes; cache entries invalidated on any label change; Reserve/Unreserve integration for in-flight scheduling decisions |
| Memory pressure from cache entries | Medium | Configurable cache size (default 10 entries per cache); LRU eviction policy; 1-minute TTL automatic cleanup |
| Concurrency issues with cache reads/writes | High | Read-write locks on cache entries; separate lock for cache-level operations; atomic operations for counter updates |
| Cache key collisions (different pods mapping to same key) | Low | Incoming pod cache keyed on `pod-template-hash` (unique per ReplicaSet revision); existing pod cache keyed on `namespace/label-set` (precise identity) |
| Performance regression when cache miss rate is high | Low | On cache miss, falls back to the original uncached computation path; no overhead beyond the cache lookup |

## Design Details

### Generic Cache Framework

The `cacheplugin` package provides a generic LRU cache implementation:

```go
type CacheImpl[T any, V any] struct {
    rwlock        sync.RWMutex
    Name          string
    Size          int
    PriorityQueue []string         // LRU ordering
    OriginMap     map[string]T     // key -> original input
    HashMap       map[string]*ItemInfo[V]  // key -> cached result
    KeyFunc       func(T) string   // key derivation
    wq            workqueue.RateLimitingInterface
    podEvHandle   func(key NamespaceedNameNode, t V, logger logr.Logger)
}
```

Key operations:
- **Read(t T) V**: Look up a cached result; on hit, moves the entry to the back of the LRU queue and updates access time
- **Write(t T, v V)**: Insert or update a cache entry; evicts the least-recently-used entry if the cache is full
- **AddIfNotPresent(t T, v V)**: Insert only if not already cached
- **Forget(f func(T, V) bool)**: Remove entries matching a predicate (used for invalidation)
- **Process(f func(T, V))**: Iterate all cache entries (used for bulk updates)
- **ProcessReservePod / ProcessUnreservePod**: Handle Reserve/Unreserve events by updating all cached entries
- **ProcessUpdatePod**: Enqueue a pod event to the rate-limited work queue for asynchronous processing

The cache runs background workers that process pod events from the work queue and a periodic cleanup goroutine that removes entries older than 1 minute.

### CachePlugin: Reserve-Phase Cache Coordination

A `CachePlugin` implementing `ReservePlugin` is registered as a scheduler plugin. It maintains a registry of all active caches:

```go
type CachePlugin struct {
    lock   sync.Mutex
    caches []CacheForPlugin
}
```

During Reserve, it notifies all caches that a pod has been assumed on a node, allowing them to update their pre-computed results (e.g., incrementing topology counts). During Unreserve, it notifies caches to revert these updates.

### InterPodAffinity Caching

#### Scoring Cache

Two caches are introduced for the PreScore phase:

1. **IncomingPodCacheProxy**: Caches the topology score contributions from the incoming pod's preferred affinity/anti-affinity terms against all existing pods.
   - **Key**: `pod-template-hash` annotation value (pods from the same ReplicaSet revision share identical affinity terms)
   - **Value**: `IncomingPodAffinityTermDetailedState` containing:
     - `preCalRes scoreMap` — pre-computed topology scores: `map[topologyKey]map[topologyValue]int64`
     - `cachedPods cachedPodsMap` — per-pod score contributions for incremental updates
     - `affinity / antiaffinity` — the affinity terms being cached

2. **ExisingPodCacheProxy**: Caches the topology score contributions from existing pods' affinity/anti-affinity terms against the incoming pod.
   - **Key**: `namespace/label-set` of the incoming pod (existing pods' terms match based on the incoming pod's labels)
   - **Value**: `ExistingPodAffinityTermDetailedState` containing:
     - `preCalRes scoreMap` — pre-computed topology scores
     - `cachedPods cachedPodsMap` — per-pod score contributions
     - `namespace / labels / namespaceLabels` — immutable identity fields

When a cache hit occurs during PreScore, the cached `preCalRes` is directly merged into the `topologyScore` state, skipping the expensive per-node/per-pod iteration. When both incoming and existing caches hit, the entire PreScore computation is skipped.

#### Filtering Cache

Similarly, two caches are introduced for the PreFilter phase:

1. **FilteringIncomingPodCacheProxy**: Caches `affinityCounts` and `antiAffinityCounts` (`topologyToMatchedTermCount`) computed from the incoming pod's required affinity/anti-affinity terms.

2. **FilteringExisingPodCacheProxy**: Caches `existingAntiAffinityCounts` computed from existing pods' required anti-affinity terms against the incoming pod.

These allow the `preFilterState` to be populated from cache instead of iterating over all nodes and pods.

### PodTopologySpread Caching

A single cache is introduced for the PreScore phase:

**PodTopologySpreadCacheProxy**: Caches per-constraint pod counts.
- **Key**: `pod-template-hash` annotation value
- **Value**: `PodTopologySpreadState` containing:
  - `preCalRes []map[string]*int64` — per-constraint map of topology-value to pod-count (using `*int64` for atomic updates)
  - `cachedPods cachedPodsMap` — per-pod topology value contributions
  - `constraints` — the topology spread constraints being cached
  - `requiredNodeAffinity` — node affinity for node inclusion filtering
  - `namespace` — pod namespace for matching

On cache hit, the pre-computed `TopologyValueToPodCounts` is directly populated from `preCalRes`, and the entire PreScore per-node iteration is skipped.

### Cache Invalidation Strategy

Cache consistency is maintained through multiple mechanisms:

1. **Pod Events (Informer)**:
   - **Add**: When a pod is bound to a node (`Spec.NodeName != ""`), its contribution is computed and added to all relevant cache entries via the work queue.
   - **Update**: If pod labels change, the pod's old contribution is removed and a new one computed.
   - **Delete**: The pod's contribution is removed from all cache entries.

2. **Node Events (Informer)**:
   - When a node's labels change, all cache entries are notified to recompute scores for pods on that node (since topology keys are derived from node labels).

3. **Namespace Events (Informer)**:
   - When a namespace's labels change, cache entries for that namespace are invalidated (for existing pod caches) or all entries are cleared (for incoming pod caches, since namespace label selectors may now match differently).

4. **Reserve/Unreserve**:
   - When the scheduler assumes a pod on a node (Reserve), caches update their precomputed results to include the assumed pod.
   - When the assumption is reverted (Unreserve), the changes are rolled back.

5. **TTL Expiration**:
   - Cache entries not accessed for 1 minute are automatically removed by a background cleanup goroutine.

### Feature Gate

A new feature gate `SchedulerPluginCache` is added:

```go
// Enables caching mechanism for InterPodAffinity and PodTopologySpread plugins.
// When enabled, the scheduler caches intermediate computation results (topology scores,
// pod-matching counts) and incrementally updates them on cluster events, reducing
// redundant computation for pods with identical affinity/topology configurations.
SchedulerPluginCache featuregate.Feature = "SchedulerPluginCache"
```

Default settings:
- Alpha (v1.36): Disabled by default
- Beta (v1.37): Enabled by default
- GA (v1.39): Locked to enabled

### Test Plan

#### Prerequisite testing updates

- Existing unit tests for `InterPodAffinity` and `PodTopologySpread` plugins are verified to pass with caching both enabled and disabled.
- Concurrency safety is validated by adding locks to `preFilterState` in both plugins.

#### Unit tests

The following test scenarios are covered:

1. **Cache event handling (`cacheplugin_test.go`)**: Verify generic cache operations (Read, Write, AddIfNotPresent, Forget, LRU eviction, TTL expiration)
2. **InterPodAffinity cache (`cache_test.go`)**: Verify that cached scoring results match uncached computation for various affinity configurations
3. **InterPodAffinity filtering cache (`filtering_map_test.go`)**: Verify that cached filtering counts match uncached computation for required affinity/anti-affinity
4. **PodTopologySpread cache (`cache_test.go`)**: Verify that pod event handling correctly increments/decrements topology counts
5. **Pod add/update/delete events**: Verify cache entries are correctly updated when pods are bound, labels change, or pods are deleted
6. **Node label change**: Verify that cache entries are recomputed when node labels change
7. **Namespace label change**: Verify that cache entries are invalidated when namespace labels change
8. **Reserve/Unreserve integration**: Verify that pre-computed results reflect assumed pods

Test files:
- `pkg/scheduler/framework/plugins/cacheplugin/cacheplugin_test.go`
- `pkg/scheduler/framework/plugins/interpodaffinity/cache_test.go`
- `pkg/scheduler/framework/plugins/interpodaffinity/filtering_map_test.go`
- `pkg/scheduler/framework/plugins/podtopologyspread/cache_test.go`
- `pkg/scheduler/framework/plugins/podtopologyspread/scoringwithcache_test.go`

#### Integration tests

- Run the scheduler with caching enabled under workloads with complex affinity and topology spread rules
- Verify scheduling decisions are identical with and without caching
- Measure scheduling throughput improvement and CPU reduction

#### e2e tests

- Run existing scheduler e2e tests with the feature gate enabled
- No new e2e tests are required as this is an internal optimization that must produce identical scheduling outcomes

### Graduation Criteria

#### Alpha

- [x] Feature implemented behind `SchedulerPluginCache` feature gate
- [x] Unit tests for generic cache framework
- [x] Unit tests for InterPodAffinity scoring and filtering caches
- [x] Unit tests for PodTopologySpread scoring cache
- [x] Concurrency safety verified (locks added to preFilterState)

#### Beta

- [ ] Feature enabled by default
- [ ] Production workloads tested showing no scheduling correctness regressions
- [ ] Performance benchmarks demonstrating measurable improvement (target: >30% CPU reduction for InterPodAffinity-heavy workloads)
- [ ] Cache hit rate metrics exposed
- [ ] No critical bugs reported for 2 releases

#### GA

- [ ] Feature used in production for at least 2 releases
- [ ] Comprehensive performance data across different cluster sizes
- [ ] All known issues resolved
- [ ] Feature gate locked to enabled

### Upgrade / Downgrade Strategy

**Upgrade**:
- When upgrading to a version with this feature, the cache is disabled by default (Alpha).
- No user action required. Existing scheduler behavior is preserved when the feature is disabled.
- When enabled, caches are populated lazily on first scheduling cycle; no warm-up period needed.

**Downgrade**:
- When downgrading, if the feature was enabled, caches are simply not initialized.
- No state is persisted; all cache data is in-memory and transient.
- The scheduler reverts to full computation for every scheduling cycle with no correctness impact.

### Version Skew Strategy

This feature is internal to the kube-scheduler and does not affect API objects, inter-component communication, or the scheduler configuration API. Version skew is not a concern.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: SchedulerPluginCache

###### Does enabling the feature change any default behavior?

No. The scheduling decisions (which node a pod is assigned to) must remain identical. The feature only changes the internal computation path by reusing cached results instead of recomputing from scratch. Scoring values and filtering decisions are the same.

###### Can the feature be disabled once it has been enabled (roll back)?

Yes, by disabling the feature gate. The scheduler will revert to the uncached computation path. In-memory caches will be garbage collected. No persistent state needs cleanup.

###### What happens if we reenable the feature if it was previously rolled back?

The scheduler will reinitialize empty caches and begin populating them on the next scheduling cycles. There is no state carryover from previous enablement.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests verify correct behavior with caching both enabled and disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollout/rollback is safe:
- Enabling: Caches are lazily populated; first scheduling cycles behave like uncached; no impact on running workloads
- Disabling: Caches are abandoned; scheduler reverts to full computation; no impact on running workloads
- No API objects or persistent state involved

###### What specific metrics should inform a rollback?

- `scheduler_schedule_attempts_total`: Should not decrease
- `scheduler_pod_scheduling_duration_seconds`: P99 latency should not increase
- `scheduler_pending_pods`: Should not grow unbounded
- Scheduler CPU usage: Should decrease (if not, cache hit rate may be too low)

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This will be tested before Beta graduation.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Check if the feature gate is enabled on the kube-scheduler:
```bash
kubectl get pods -n kube-system -l component=kube-scheduler -o yaml | grep SchedulerPluginCache
```

Scheduler logs at verbosity level 3+ will show cache hit/miss messages.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- Scheduling correctness: 100% identical decisions with and without caching
- Scheduling throughput: Should improve or remain the same
- P99 scheduling latency: Should improve or remain the same

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- `scheduler_schedule_attempts_total`: Rate of scheduling attempts
- `scheduler_pod_scheduling_duration_seconds`: Scheduling latency distribution
- `scheduler_pending_pods`: Number of pending pods by queue
- Scheduler process CPU usage

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. This is an internal scheduler optimization. The caches use existing Informer-based watchers already present in the scheduler.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. The feature is designed to reduce time taken by scheduling operations. In the worst case (100% cache miss), there is minimal overhead from cache lookup.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

- **Memory**: Each cache stores up to `cacheSize` entries (default 10). Each entry contains pre-computed score maps and per-pod contributions. For a 5,000-node cluster with 100k pods, a single cache entry might use ~1-5 MB. With 10 entries across 5 caches, total memory overhead is ~50-250 MB, which is acceptable for the CPU savings.
- **CPU**: Background workers (default 4 per cache) consume CPU for event processing, but this is amortized over scheduling cycles. Net CPU usage should decrease.
- **Goroutines**: Each cache spawns `workerSize` (default 4) background workers plus 1 cleanup goroutine. With 5 caches, this adds ~25 goroutines.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

This feature does not interact with API server or etcd directly. Informer event handlers may stop receiving events if the API server is unavailable, which could lead to stale cache entries. However, the 1-minute TTL cleanup and the existing scheduler behavior of retrying scheduling cycles mitigate this.

###### What are other known failure modes?

| Failure Mode | Description | Mitigation |
|---|---|---|
| Stale cache entries | Cache not updated due to missed Informer events | 1-minute TTL auto-cleanup; Reserve/Unreserve integration; pod event handlers with rate-limited retry |
| Memory growth | Too many cache entries or large per-entry data | Configurable cache size with LRU eviction; TTL cleanup |
| Lock contention on cache | High-frequency reads/writes contending on cache locks | Read-write locks (multiple readers allowed); separate per-entry locks for fine-grained updates |
| Incorrect scores from cache | Bug in incremental update logic | Comprehensive unit tests; fallback to uncached path on feature disable |

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check if the feature gate is enabled
2. Examine scheduler logs at verbosity 3+ for cache hit/miss patterns
3. Monitor scheduler CPU usage and memory usage
4. If scheduling decisions appear incorrect, disable the feature immediately
5. Collect scheduler pprof profiles for analysis

## Implementation History

- 2025-03-10: KEP created
- 2025-03-10: Alpha implementation merged (commit 7806a9a)

## Drawbacks

1. **Memory Overhead**: Each cache entry stores pre-computed results and per-pod contributions, which can consume significant memory in very large clusters.
2. **Code Complexity**: The caching layer adds substantial complexity (3,600+ lines of new code) with concurrent data structures, Informer event handlers, and incremental update logic.
3. **Cache Consistency Risk**: Any bug in the cache invalidation or incremental update logic could lead to incorrect scheduling decisions, which are difficult to diagnose.
4. **Limited Applicability**: The caching is most effective for workloads with identical `pod-template-hash` (Deployments/ReplicaSets). Standalone pods or Jobs with unique labels will not benefit.

## Alternatives

1. **Scheduler Framework Snapshot Caching**: Cache at the framework snapshot level rather than per-plugin. This would benefit all plugins but requires significant changes to the scheduler framework interfaces.

2. **Pod Grouping at Queue Level**: Group identical pods at the scheduling queue level and schedule them as a batch with shared computation. This is more invasive but could yield even greater performance improvements.

3. **Precomputed Scoring Tables**: Maintain a global scoring table updated by Informers, instead of per-pod caches. This avoids cache miss overhead but requires more memory and complex synchronization.

4. **Parallelization Improvements**: Instead of caching, further parallelize the per-node computation within PreScore. This is simpler but does not eliminate the redundant computation across scheduling cycles.

5. **Profile-Guided Optimization**: Use pprof data to identify and optimize the specific hot paths in affinity/topology spread evaluation without introducing a caching layer. This is less invasive but provides smaller improvements.

The chosen approach (per-plugin LRU caches with Informer-driven incremental updates) provides a good balance of performance improvement and implementation complexity, while being opt-in and reversible.

## Infrastructure Needed

No additional infrastructure needed. This is a code-only change to the kube-scheduler.
