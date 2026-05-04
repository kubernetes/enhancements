# KEP-6053: Kubelet Image Garbage Collection Observability
<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Metrics](#metrics)
  - [Trigger Attribution](#trigger-attribution)
  - [Result Semantics](#result-semantics)
  - [User Stories](#user-stories)
    - [Story 1: Alert on ineffective Image GC](#story-1-alert-on-ineffective-image-gc)
    - [Story 2: Detect runtime or filesystem stat issues](#story-2-detect-runtime-or-filesystem-stat-issues)
    - [Story 3: Understand GC latency regressions](#story-3-understand-gc-latency-regressions)
- [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
    - [Unit tests](#unit-tests)
    - [Integration tests](#integration-tests)
    - [e2e tests](#e2e-tests)
- [Graduation Criteria](#graduation-criteria)
  - [Alpha](#alpha)
  - [Beta](#beta)
  - [Stable](#stable)
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
  - [Rely on kubelet logs](#rely-on-kubelet-logs)
  - [Add Kubernetes events for every Image GC attempt](#add-kubernetes-events-for-every-image-gc-attempt)
  - [Extend the existing image deletion counter only](#extend-the-existing-image-deletion-counter-only)
  - [Add per-image labels](#add-per-image-labels)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in kubernetes/enhancements
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in kubernetes/website

## Summary

Improve kubelet Image Garbage Collection (Image GC) observability by adding
structured metrics that describe each Image GC attempt, its trigger, result,
duration, requested reclaim target, reclaimed bytes, and image deletion
outcomes.

Today, kubelet exposes `kubelet_image_garbage_collected_total{reason}`, which
counts successfully removed images by broad reason. Operators can observe that
some images were removed, but cannot reliably answer operational questions such
as:

- Is Image GC running when image filesystem usage crosses the configured high
  threshold?
- How long does Image GC take on each node?
- How many bytes did kubelet attempt to reclaim, and how many bytes were
  actually reclaimed?
- Did Image GC fail because image filesystem stats were unavailable, no images
  were eligible, image deletion failed, or the reclaim target could not be met?
- Are images skipped because they are in use, too new, pinned, or otherwise not
  eligible?

This KEP proposes kubelet metrics that make those answers available without
changing Image GC policy, kubelet configuration, CRI APIs, pod behavior, or the
eviction API.

## Motivation

Image GC is a critical node-local mechanism for maintaining available storage on
the image filesystem. When it does not run, runs slowly, or cannot reclaim
enough space, nodes may enter disk pressure and workloads may be disrupted by
evictions or image pull failures.

The existing kubelet metric only counts successfully deleted images and does not
describe failed or no-op GC attempts. This leaves operators dependent on kubelet
logs and node-local investigation during storage pressure incidents. Logs are
often sampled, rotated, or unavailable in centralized monitoring systems, and
they are difficult to aggregate across a fleet.

### Goals

- Add low-cardinality kubelet metrics for Image GC attempts, latency, reclaim
  target, reclaimed bytes, and deletion outcomes.
- Expose enough labels to distinguish periodic threshold-based Image GC,
  age-based cleanup, and eviction-manager-triggered cleanup.
- Expose failure and no-op outcomes in metrics, not only successful image
  deletions.
- Keep all metrics node-local, scrapeable from the existing kubelet metrics
  endpoint, and compatible with existing Kubernetes metric stability policy.
- Avoid labels containing image names, image IDs, pod names, namespaces,
  container names, runtime handlers, filesystem paths, or error strings.

### Non-Goals

- Change the Image GC policy or thresholds.
- Add new kubelet configuration fields.
- Add or change Kubernetes APIs.
- Add or change CRI APIs.
- Replace kubelet events or logs.
- Provide per-image, per-pod, per-namespace, or per-runtime-handler GC metrics.
- Guarantee that the runtime actually freed the exact number of bytes reported
  by image metadata after an image deletion.

## Proposal

Add ALPHA kubelet metrics emitted by the Image GC manager.

### Metrics

`kubelet_image_gc_attempts_total`

- Type: Counter
- Labels:
  - `operation`: `garbage_collect` or `delete_unused_images`
  - `trigger`: `periodic`, `eviction`, or `unknown`
  - `result`: `success`, `error`, `partial`, or `noop`
  - `reason`: `threshold`, `age`, `all_unused`, `stats_error`,
    `invalid_capacity`, `delete_error`, `insufficient_freed`,
    `detect_error`, or `unknown`
- Description: Number of Image GC attempts by operation, trigger, result, and
  reason.

`kubelet_image_gc_duration_seconds`

- Type: Histogram
- Labels:
  - `operation`
  - `trigger`
  - `result`
- Description: Duration of Image GC attempts.

`kubelet_image_gc_freed_bytes`

- Type: Histogram
- Labels:
  - `operation`
  - `trigger`
  - `result`
  - `reason`
- Description: Bytes kubelet estimated as reclaimed by deleted images during an
  Image GC attempt.

`kubelet_image_gc_target_bytes`

- Type: Histogram
- Labels:
  - `operation`
  - `trigger`
- Description: Bytes kubelet attempted to reclaim. For
  `delete_unused_images`, this records the unbounded cleanup request as a
  sentinel-free value by using the total size of eligible images, not
  `math.MaxInt64`.

`kubelet_image_gc_image_deletions_total`

- Type: Counter
- Labels:
  - `operation`
  - `trigger`
  - `reason`: `space`, `age`, or `all_unused`
  - `result`: `success` or `error`
- Description: Number of image deletion attempts and their result.

This metric supersets the existing
`kubelet_image_garbage_collected_total{reason}`. The existing metric remains in
place for compatibility. Implementations should increment both metrics for
successful image deletions during the deprecation-free compatibility period.

`kubelet_image_gc_skipped_images_total`

- Type: Counter
- Labels:
  - `operation`
  - `trigger`
  - `reason`: `in_use`, `min_age`, `pinned`, or `other`
- Description: Number of images considered by Image GC but skipped because they
  were not eligible for deletion.

`kubelet_image_gc_image_fs_usage_percent`

- Type: Gauge
- Labels:
  - `phase`: `before`
- Description: Image filesystem usage percentage observed by kubelet before
  threshold-based Image GC.

This KEP intentionally does not add a post-GC image filesystem usage gauge. The
current Image GC code estimates reclaimed bytes from image metadata and does not
re-query image filesystem stats after deletion. Adding a post-GC gauge would
require an additional stats call on every GC attempt. That can be reconsidered
later if operators need it.

### Trigger Attribution

Kubelet currently calls Image GC from different paths:

- Periodic kubelet Image GC policy enforcement calls `GarbageCollect`.
- Eviction manager node reclaim calls `DeleteUnusedImages`.
- Tests or future internal call sites may call either method directly.

The implementation should preserve the current Image GC interface where
possible. If trigger attribution cannot be inferred reliably from existing call
sites, kubelet may add an internal, unexported context value or a small internal
options struct to pass `trigger` from known callers. Unknown callers must be
reported with `trigger="unknown"` rather than introducing high-cardinality
labels.

### Result Semantics

- `success`: Image GC completed and met the requested reclaim objective, or
  age-based cleanup completed without deletion errors.
- `partial`: Image GC deleted one or more images but did not meet the requested
  reclaim objective.
- `noop`: Image GC completed without deleting images because no cleanup was
  needed or no images were eligible.
- `error`: Image GC failed before completing, or image deletion errors occurred.

### User Stories

#### Story 1: Alert on ineffective Image GC

A cluster operator receives disk pressure alerts on a subset of nodes. They
query kubelet metrics and see increasing
`kubelet_image_gc_attempts_total{result="partial",reason="insufficient_freed"}`
and low `kubelet_image_gc_freed_bytes` values. This indicates that kubelet is
attempting Image GC but cannot reclaim enough bytes, likely because disk usage
is dominated by active images, logs, volumes, or data outside image storage.

#### Story 2: Detect runtime or filesystem stat issues

An operator sees nodes where image filesystem usage is high, but
`kubelet_image_gc_attempts_total{reason="stats_error"}` is increasing. They can
identify kubelet or CRI stats collection as the reason Image GC cannot make a
threshold decision.

#### Story 3: Understand GC latency regressions

After a runtime upgrade, an operator observes increased
`kubelet_image_gc_duration_seconds` on nodes under image churn. They can compare
latency distributions before and after rollout and decide whether to roll back
the runtime change.

## Risks and Mitigations

Metric cardinality risk is mitigated by using only bounded labels. The proposal
does not include image IDs, image names, runtime handlers, pod names,
namespaces, paths, node names, or raw error messages.

Behavioral regression risk is mitigated by making instrumentation passive. The
metrics are emitted from existing decision points and do not alter policy,
thresholds, deletion order, or runtime calls.

Accounting accuracy risk exists because kubelet currently estimates freed bytes
from image metadata. The metric name and help text should state that freed bytes
are kubelet-estimated reclaimed bytes, not a filesystem-level measurement.

Latency overhead is expected to be negligible. Metrics are updated once per GC
attempt and once per considered or deleted image. No additional CRI calls are
required by this proposal.

## Design Details

Implementation should instrument the following paths in
`pkg/kubelet/images/image_gc_manager.go`:

- `GarbageCollect`: record attempt start, threshold decision, filesystem usage,
  target bytes, duration, outcome, and freed bytes.
- `DeleteUnusedImages`: record attempt start, target bytes derived from
  eligible image sizes, duration, outcome, and freed bytes.
- `freeSpace`: return structured summary data in addition to current return
  values, or update a per-attempt accumulator that records skipped images,
  deletion attempts, deletion successes, deletion failures, and freed bytes.
- `freeOldImages`: record age-based deletion outcomes and skipped image counts
  where applicable.
- `freeImage`: record deletion success and failure for the new deletion metric,
  while preserving the existing `ImageGarbageCollectedTotal` metric on
  successful deletions.

The first implementation should keep metrics in
`pkg/kubelet/metrics/metrics.go` unless SIG Node and SIG Instrumentation prefer
co-locating Image Manager metrics under `pkg/kubelet/images/metrics.go`.

### Test Plan

- [ ] I/we understand the owners of the involved components may require updates
  to existing tests to make this code solid enough prior to committing the
  changes necessary to implement this enhancement.

#### Prerequisite testing updates

None.

#### Unit tests

- `pkg/kubelet/images/image_gc_manager_test.go`: add assertions for metrics
  emitted by:
  - below-threshold no-op GC
  - stats provider failure
  - invalid image filesystem capacity
  - successful threshold-based cleanup
  - insufficient freed bytes
  - image deletion error
  - age-based cleanup
  - `DeleteUnusedImages`
- `pkg/kubelet/metrics/metrics_test.go`: add metric registration and stability
  assertions for the new kubelet metrics.

#### Integration tests

No new integration tests are required. The feature is passive kubelet
instrumentation and is covered by unit tests around the Image GC manager.

#### e2e tests

No new e2e tests are required for Alpha. Existing node e2e tests for image
garbage collection and eviction continue to validate behavior. If SIG Node
requires end-to-end signal validation for Beta, add a node e2e test that forces
Image GC and verifies the new metrics are exposed by kubelet.

## Graduation Criteria

### Alpha

- Metrics are implemented as ALPHA metrics.
- Unit tests cover success, failure, partial, and no-op Image GC outcomes.
- Existing `kubelet_image_garbage_collected_total` behavior is preserved.
- Documentation lists the new kubelet metrics and label values.

### Beta

- Metrics have been available for at least one release.
- SIG Node and SIG Instrumentation review feedback from real cluster usage.
- Label names and values are confirmed to be sufficiently low-cardinality.
- Any naming or semantic adjustments required by metric stability review are
  completed before Beta.

### Stable

- Metrics satisfy Kubernetes metric stability requirements.
- No unresolved scalability or cardinality concerns remain.
- Documentation and troubleshooting guidance are complete.

## Upgrade / Downgrade Strategy

Upgrading kubelet adds new ALPHA metrics. No configuration or API migration is
required.

Downgrading kubelet removes the new metrics from the kubelet metrics endpoint.
Existing workloads and Image GC behavior are unaffected.

## Version Skew Strategy

The feature is entirely node-local kubelet instrumentation. Kubelets at
different versions may expose different metric sets during a rolling upgrade.
Control plane version skew does not affect this feature.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

How can this feature be enabled / disabled in a live cluster?

- The feature is enabled by running a kubelet version that includes the metrics.
- No feature gate is proposed because the change is passive observability.
- Rollback is performed by rolling back kubelet to a version without the
  metrics.

Does enabling the feature change any default behavior?

- No.

Can the feature be disabled once it has been enabled?

- The metrics can be removed by rolling back kubelet. Image GC behavior is
  unchanged.

What happens if we reenable the feature if it was previously rolled back?

- Metrics begin being emitted again. Counter continuity follows normal
  Prometheus process restart behavior.

Are there any tests for feature enablement/disablement?

- No feature gate is proposed. Unit tests verify metric emission.

### Rollout, Upgrade and Rollback Planning

How can a rollout or rollback fail? Can it impact already running workloads?

- The rollout can fail only through normal kubelet rollout failure modes. The
  metrics themselves do not affect running workloads.

What specific metrics should inform a rollback?

- Unexpected kubelet CPU or memory regression.
- Unexpected scrape cardinality or Prometheus ingestion growth attributed to the
  new metrics.

Were upgrade and rollback tested?

- Not yet.

Is the rollout accompanied by any deprecations and/or removals?

- No. The existing `kubelet_image_garbage_collected_total` metric remains.

### Monitoring Requirements

How can an operator determine if the feature is in use by workloads?

- This feature is not used by workloads. Operators can determine availability by
  checking whether the new metrics exist on kubelet's metrics endpoint.

How can someone using this feature know that it is working for their instance?

- Scrape kubelet metrics and query `kubelet_image_gc_attempts_total` after
  kubelet has performed Image GC.

What are the reasonable SLOs for the enhancement?

- The enhancement has no service SLO. It provides observability for an existing
  kubelet maintenance operation.

What are the SLIs an operator can use to determine the health of the service?

- `rate(kubelet_image_gc_attempts_total{result="error"}[5m])`
- `rate(kubelet_image_gc_attempts_total{result="partial"}[5m])`
- `histogram_quantile(0.99, rate(kubelet_image_gc_duration_seconds_bucket[5m]))`
- `rate(kubelet_image_gc_freed_bytes_sum[5m])`
- `kubelet_image_gc_image_fs_usage_percent{phase="before"}`

Are there any missing metrics that would be useful to have to improve
observability of this feature?

- This KEP defines the missing metrics.

### Dependencies

Does this feature depend on any specific services running in the cluster?

- No. It depends only on kubelet's existing metrics endpoint and current Image
  GC implementation.

### Scalability

Will enabling / using this feature result in any new API calls?

- No.

Will enabling / using this feature result in introducing new API types?

- No.

Will enabling / using this feature result in any new calls to the cloud
provider?

- No.

Will enabling / using this feature result in increasing size or count of the
existing API objects?

- No.

Will enabling / using this feature result in increasing time taken by any
operations covered by existing SLIs/SLOs?

- No meaningful increase is expected. Metrics are updated during Image GC, which
  is already outside request-serving paths.

Will enabling / using this feature result in non-negligible increase of resource
usage in any components?

- No. The metrics have bounded cardinality and are updated at low frequency.

Can enabling / using this feature result in resource exhaustion of some node
resources?

- No.

### Troubleshooting

How does this feature react if the API server and/or etcd is unavailable?

- It is unaffected. Image GC and kubelet metrics are node-local.

What are other known failure modes?

- If kubelet metrics scraping is unavailable, operators cannot use these
  metrics.
- If the CRI or stats provider cannot return image filesystem stats, kubelet
  records attempts with `reason="stats_error"` where possible.
- If image deletion fails, kubelet records deletion errors and Image GC attempt
  errors.

What steps should be taken if SLOs are not being met to determine the problem?

- Check `kubelet_image_gc_attempts_total` by `result` and `reason`.
- Check Image GC duration histograms for latency regressions.
- Compare `kubelet_image_gc_target_bytes` and
  `kubelet_image_gc_freed_bytes`.
- Inspect kubelet logs for concrete runtime errors after metrics identify the
  failing node and failure category.
- Inspect node filesystem usage to determine whether disk pressure is caused by
  active images, logs, volumes, or non-Kubernetes data.

## Implementation History

- 2026-05-01: Initial provisional draft.

## Drawbacks

- Adds more kubelet metrics and documentation surface.
- Requires careful metric naming and label review to avoid committing to
  unclear semantics.
- Estimated freed bytes may be misinterpreted as filesystem-measured freed
  bytes unless documented clearly.

## Alternatives

### Rely on kubelet logs

Kubelet logs already contain useful Image GC messages, but logs are harder to
aggregate, alert on, and retain consistently across nodes. Metrics provide a
lower-friction operational signal.

### Add Kubernetes events for every Image GC attempt

Events are useful for exceptional conditions but are not appropriate for
high-volume or periodic operational telemetry. Metrics are a better fit for
fleet-wide aggregation.

### Extend the existing image deletion counter only

Adding labels to `kubelet_image_garbage_collected_total` would not capture
failed attempts, no-op attempts, latency, reclaim targets, or skipped images.
Keeping the existing metric unchanged avoids compatibility risk.

### Add per-image labels

Per-image labels would answer detailed forensic questions but would create
unbounded cardinality. This KEP intentionally avoids them.

## Infrastructure Needed

None.
