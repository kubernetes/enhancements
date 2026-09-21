# KEP-6404: API Server Write Throughput: Reducing Allocations

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [1. The Allocation Ceiling](#1-the-allocation-ceiling)
  - [2. Sticky Degradation on Disruptions](#2-sticky-degradation-on-disruptions)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Serve All GETs from the Watch Cache](#serve-all-gets-from-the-watch-cache)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Watch Cache Lock Contention](#watch-cache-lock-contention)
- [Design Details](#design-details)
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
- [Potential Future Extensions](#potential-future-extensions)
  - [Avoiding Whole-Object Decoding and Encoding](#avoiding-whole-object-decoding-and-encoding)
  - [Storage Versioning Metadata and Open Challenges](#storage-versioning-metadata-and-open-challenges)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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

Following recent SIG Scalability investigations into `kube-apiserver` write throughput using realistic resource sizes (>10KB pods), memory allocations were identified as the primary bottleneck limiting write scale and causing sticky performance degradation during disruptions.

This KEP proposes to address both sources of allocations on happy path and remove some `kube-apiserver` fallbacks that lead to allocations amplifaction causing unrecoverable state degradations. 

## Motivation

Following the SIG Scalability proposals to introduce Resource Size as a new dimension ([kubernetes/kubernetes#134375](https://github.com/kubernetes/kubernetes/issues/134375)) and establish a realistic "Pod Shape" ([kubernetes/kubernetes#138415](https://github.com/kubernetes/kubernetes/issues/138415)), write throughput benchmarks revealed that **memory allocations** are the primary limiting factor for `kube-apiserver` scalability.

### 1. The Allocation Ceiling

**The API server allocates over 1MB of memory to apply a trivial patch on a 10KB pod object—a >100x allocation factor on the *happy path*** (which includes storing the object in etcd, observing it on watch, decoding and transforming it, storing it in the watch cache, and re-encoding/propagating it to external and internal watchers).

When running [k8s write throughput benchmark] on a `c4-highmem-144` machine, we discovered that Go runtime hits a hard ceiling around 6GB/s of memory allocations. At this ceiling, Compare-And-Swap (CAS) operations during the GC sweep phase consume >50% of CPU, saturating the node. While tuning `GOGC` provides temporary relief (up to ~12GB/s), while we reported the issue to Go team, a fundamental reduction in per-operation allocations within `kube-apiserver` is required.

[k8s write throughput benchmark]: https://github.com/kubernetes/perf-tests/tree/master/clusterloader2/testing/write-throughput

Sources of these allocations include:
- **Inefficient PATCH validation and merging:** Specifically `managedFields` patching and Server-Side Apply (SSA) tracking, which account for over half of write-path allocations (tracked separately in [kubernetes/kubernetes#142228](https://github.com/kubernetes/kubernetes/issues/142228)).
- **Repeated Whole-Object Decoding and Encoding:** Even when a request only modifies tens of bytes on a 10KB object (for example, a Pod `PATCH` that sets `spec.nodeName` or updates `status`), `kube-apiserver` operates on the object as a whole rather than only on the relevant bytes that changed. While the watch cache helps serialize objects once for synced watchers, each operation on an object serializes the entire object from scratch.

While `managedFields` overhead is being optimized directly in [kubernetes/kubernetes#142228](https://github.com/kubernetes/kubernetes/issues/142228), avoiding redundant whole-object encoding and decoding has nontrivial interactions with Kubernetes's data model (defaulting, unknown field stripping, and storage decoration) and is documented in [Potential Future Extensions](#potential-future-extensions).

### 2. Sticky Degradation on Disruptions

The >100x allocation factor only accounts for the steady-state happy path. On a traffic burst or short disruption, the API server can enter a degraded state where the fallback path is more costly than the normal path, preventing recovery. Common causes of such a state include:
- **Watch breaking:** When a watch connection breaks, re-establishing watch on non-zero RV doesn't benefit from serialization caching.
- **Admission Informer:** The API server includes built-in informers that on short disruption can relist, further straining the process.
- **Conflicting TXNs:** When optimistic concurrency conflicts occur, the API server falls back to fetching the latest object directly from etcd and decoding it from scratch.

In these scenarios, **allocations easily double, turning a brief disruption into a persistent, sticky degradation.** Hitting the allocation ceiling requires dropping client throughput by 2x–3x before `kube-apiserver` can recover. Consequently, `kube-apiserver` cannot tolerate sudden bursts in write QPS without entering a degraded loop.

As part of this KEP we would like to tackle the problem of conflicting TXN fallbacks by serving all `GET`s from the watch cache.

### Goals

- **Reduce per-operation memory allocations in `kube-apiserver`** to raise the write-throughput ceiling and prevent sticky performance degradation during disruptions.
- **Serve all `GET` requests from the watch cache**, including consistent `GET`s (`resourceVersion=""`) and internal `GET`s on write/patch transaction (`TXN`) conflicts.

### Non-Goals


## Proposal

### Serve All GETs from the Watch Cache

Currently, while `LIST` requests with `resourceVersion=""` use the consistent read-from-cache mechanism ([KEP-2340](/keps/sig-api-machinery/2340-Consistent-reads-from-cache)) and historical `LIST`s use B-tree cache snapshots ([KEP-4988](/keps/sig-api-machinery/4988-snapshottable-api-server-cache)), many `GET` paths still bypass the watch cache and hit etcd directly:
- Consistent `GET` requests (`resourceVersion=""`).
- Internal `GET` requests executed by `GuaranteedUpdate` when an optimistic concurrency transaction (`TXN`) fails due to a conflict.

We propose routing **all** `GET` requests through the watch cache:
- For consistent `GET`s and `TXN` conflict retries, the watch cache will ensure freshness up to the required etcd revision (using watch progress notifications or the revision returned by the failed TXN response) and return the already-decoded cached object from the immutable B-tree snapshot.
- Because the watch cache in v1.37+ relies on immutable B-tree snapshots and avoids copying under the lock, serving all `GET`s from cache scales concurrently without lock contention and eliminates both etcd load spikes and expensive per-retry decode allocations during write conflicts.

### Risks and Mitigations

#### Watch Cache Lock Contention
- **Risk:** Routing all `GET`s (including `GuaranteedUpdate` conflict `GET`s) through the watch cache could increase lock contention on the watch cache mutex.
- **Mitigation:** Major watch cache refactors in v1.37 replaced copying under locks with lock-free reads over immutable B-tree snapshots. `GET` lookups only acquire a brief pointer to the latest snapshot.

## Design Details

1. **Serving All `GET`s from Cache (`ConsistentGetFromCache`):**
   - For client consistent `GET`s (`resourceVersion=""`), `cacher.Get` determines the required revision from etcd (or uses the latest known revision) and invokes `WaitUntilFreshAndGet(ctx, requiredRV, key)` against the watch cache B-tree snapshot.
   - For optimistic concurrency (`TXN`) conflicts in `GuaranteedUpdate`, instead of fetching and decoding the conflicting object directly from etcd, `GuaranteedUpdate` waits for the watch cache to reach the revision returned in the failed `TxnResponse` and retrieves the already-decoded object from the watch cache.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

- Unit tests in `k8s.io/apiserver/pkg/storage/cacher` and `k8s.io/apiserver/pkg/storage/etcd3` covering consistent `GET`s and `GuaranteedUpdate` conflict retries served from cache with `ConsistentGetFromCache` enabled and disabled.

##### Integration tests

- Integration tests in `test/integration/apiserver/correctness` verifying `GET` consistency and `GuaranteedUpdate` conflict handling across `ConsistentGetFromCache` feature gate toggles.

##### e2e tests

- Existing Kubernetes e2e and conformance suites exercise all `GET`, `CREATE`, `UPDATE`, and `PATCH` operations.

### Graduation Criteria

#### Alpha
- Implement `ConsistentGetFromCache` feature gate to serve all `GET`s (including `GuaranteedUpdate` conflict `GET`s) from the watch cache.

#### Beta
- Enable `ConsistentGetFromCache` by default.

#### GA
- Graduate `ConsistentGetFromCache` to GA and lock to default-on.

### Upgrade / Downgrade Strategy

- `ConsistentGetFromCache` is a purely in-memory feature in `kube-apiserver` with no persistent storage format changes.
- Upgrading or downgrading `kube-apiserver`, or toggling the feature gate, requires no data migration.

### Version Skew Strategy

- Purely internal to each `kube-apiserver` instance.
- Safe under HA mixed-version control planes (`v1.(N-1)` and `v1.N`), as each `kube-apiserver` independently serves its `GET` requests from its own watch cache or etcd.

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

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ConsistentGetFromCache`
  - Components depending on the feature gate: `kube-apiserver`

###### Does enabling the feature change any default behavior?

- **`ConsistentGetFromCache`:** Consistent `GET` requests (`resourceVersion=""`) and internal `GET` requests triggered on `GuaranteedUpdate` transaction conflicts are served from the watch cache. While the API semantics and consistency guarantees remain identical puts watch cache on critical path. This means that availability is dependent on the health of the watch cache and latency of those requests will be impacted by watch cache delay. However the incurrent latency is intentional to remove the costly fallback and allow APF to properly protect apiserver and etcd from overload.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. `ConsistentGetFromCache` can be disabled by setting the feature gate to `false` and restarting `kube-apiserver`, which immediately reverts `kube-apiserver` to serving consistent `GET`s and `TXN` conflict retries directly from etcd.

###### What happens if we reenable the feature if it was previously rolled back?

`kube-apiserver` immediately resumes serving all `GET` requests from the watch cache.

###### Are there any tests for feature enablement/disablement?

ConsistentGetFromCache is purely in-memory feature, so enablement/disablement tests boil down to feature tests.

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

No. Serving `GET`s from the watch cache reduces direct `Range` (`GET`) requests to etcd (especially during `TXN` conflicts), replacing full object fetches with lightweight watch progress checks when needed.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. It is designed to reduce `GET`, `PATCH`, and `UPDATE` latency and prevent degraded-state latency spikes under high write throughput.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. It significantly reduces CPU and memory allocations in `kube-apiserver` and reduces read load on etcd.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

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

- 2026-09-18: Initial write-throughput investigation and proposal filed in [kubernetes/kubernetes#142223](https://github.com/kubernetes/kubernetes/issues/142223).
- 2026-09-21: KEP-6404 drafted to serve all `GET`s from cache (`ConsistentGetFromCache`) and document future serialization/deserialization allocation optimizations.

## Drawbacks

- Puts the watch cache on the critical path for consistent `GET`s (`resourceVersion=""`) and `GuaranteedUpdate` conflict retries, making their latency and availability dependent on watch cache freshness. As noted in the PRR, this tradeoff is intentional to eliminate the expensive etcd decode fallback and allow APF to protect `kube-apiserver` and etcd from overload.

## Alternatives

- **Relying solely on `GetOnFailure` in etcd `TxnResponse` for `GuaranteedUpdate` conflicts:** While etcd `TxnResponse` can return the conflicting key-value pair directly without an extra network round-trip, `kube-apiserver` still has to deserialize and default those raw bytes into a fresh Go struct on every conflict. Serving the conflicting object from the watch cache returns the already-decoded object in memory, eliminating the per-conflict decode allocations.

## Potential Future Extensions

### Avoiding Whole-Object Decoding and Encoding

`kube-apiserver` has an ingrained bottleneck of operating on API objects as a whole rather than only on the relevant bytes that changed. While network transfer of large objects is rarely the bottleneck because wire payloads compress well, **encoding and decoding whole 10–100KB objects** consumes a massive share of CPU and memory allocations.

Consider a `PATCH` request to a Pod that only sets `spec.nodeName` or updates `status`: even though the change is only tens of bytes, `kube-apiserver` must serialize the entire 10–100KB Pod object. If `kube-apiserver` knew that only `status` was changing, it could serialize `status` alone and reuse the existing serialization of `spec`—reducing serialization cost from `O(object size)` to `O(subtree that changed)` at an appropriate subtree granularity. Similarly, on the read/watch path, `kube-apiserver` could reuse bytes already serialized in etcd instead of re-encoding unchanged subtrees.

### Storage Versioning Metadata and Open Challenges

Most optimizations that reuse bytes stored in etcd share a common prerequisite: knowing that the bytes persisted in etcd are already up to date with the current `kube-apiserver` representation.

Although `kube-apiserver` already applies field defaulting on the write path (when decoding the incoming client request), it also applies defaulting on the read path to handle cluster upgrades: when a cluster upgrades from `v1.(N-1)` to `v1.N` and `v1.N` introduces a new field with a new default value, objects written before the upgrade do not have that default persisted in etcd. Because nothing in the etcd record indicates which Kubernetes version wrote the object, `kube-apiserver` currently must decode and default every object on read.

One candidate approach we explored was extending the etcd storage envelope metadata to record the **Kubernetes minor version** (`k8sMinorVersion = max(storedVersion, currentVersion)`) that wrote the object, using `stored.k8sMinorVersion >= currentAPIServer.minorVersion` to determine whether the stored serialization is up to date (analogous to how runtimes like Java fast-load bytecode produced by a compatible version without re-verification).

However, review identified several reasons why a Kubernetes minor-version stamp alone is insufficient to safely reuse or pass through undecoded bytes from etcd:
1. **Feature-gate-controlled defaulting:** Default values can depend on whether specific feature gates are enabled on a given `kube-apiserver` instance, not just the binary's minor version.
2. **Unknown field stripping:** Decoding into Go structs strips unknown fields, ensuring `kube-apiserver` only returns known fields to clients. Passing through or reusing undecoded bytes could preserve unknown fields and cause observable behavioral differences.
3. **Storage-level decoration:** `rest.Storage` supports read-time `Decorator` hooks (e.g., `store.Decorator = rest.defaultOnRead` used in PVC and Service storage) that mutate objects on read.
4. **Target version, format, and serialization options:** Clients frequently request conversions across API versions, wire encodings (`JSON` vs. `Protobuf`), or serialization options.

Alternative approaches—such as **zero-allocation defaulting on the read path** or **subtree-level manipulation directly on Protobuf wire bytes**—could reduce serialization/deserialization allocations from `O(object size)` toward `O(changed fields)` while respecting Kubernetes defaulting and unknown-field semantics, and will be explored in future proposals.

## Infrastructure Needed (Optional)
