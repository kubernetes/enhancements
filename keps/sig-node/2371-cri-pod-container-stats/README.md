<!-- toc -->
- [cAdvisor-less, CRI-full Container and Pod Stats](#cadvisor-less-cri-full-container-and-pod-stats)
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
    - [Current State of These Metrics](#current-state-of-these-metrics)
      - [Current Fulfiller of Metrics Endpoints &amp; Future Proposal](#current-fulfiller-of-metrics-endpoints--future-proposal)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [Summary API](#summary-api)
    - [/metrics/cadvisor](#metricscadvisor)
    - [User Stories [optional]](#user-stories-optional)
      - [Story 1](#story-1)
      - [Story 2](#story-2)
    - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
      - [History/Past Conversations](#historypast-conversations)
      - [Open Questions](#open-questions)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Design Details](#design-details)
    - [Stats Summary API](#stats-summary-api)
      - [CRI Implementation](#cri-implementation)
        - [ContainerStats additions](#containerstats-additions)
        - [PodStats CRI additions](#podstats-cri-additions)
      - [Kubelet](#kubelet)
      - [cAdvisor](#cadvisor)
    - [cAdvisor Metrics Endpoint](#cadvisor-metrics-endpoint)
      - [CRI implementations](#cri-implementations)
      - [cAdvisor](#cadvisor-1)
    - [Test Plan](#test-plan)
    - [Graduation Criteria](#graduation-criteria)
      - [Alpha implementation](#alpha-implementation)
      - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
      - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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
<!-- /toc -->

# cAdvisor-less, CRI-full Container and Pod Stats

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
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

There are two main APIs that consumers use to gather stats about running containers and pods: [summary API][summary-api] and [/metrics/cadvisor][metrics-cadvisor].
The Kubelet is responsible for implementing the summary API, and cadvisor is responsible for fulfilling `/metrics/cadvisor`.

The [CRI API](https://github.com/kubernetes/cri-api) currently does not provide enough metrics to fully supply all the fields for either endpoint, however is used to fill some fields of the summary API.
This results in an unclear origin of metrics, duplication of work done between cAdvisor and CRI, and performance implications. 

This KEP aims to enhance CRI implementations to be able to fulfill all the stats needs of Kubernetes.
At a high level, there are two pieces of this:
- enhance the CRI API with enough metrics to be able to supplement the pod and container fields in the summary API directly from CRI.
- enhance the CRI implementations to broadcast the required metrics to fulfill the pod and container fields in the `/metrics/cadvisor` endpoint.

[summary-api]: https://github.com/kubernetes/kubernetes/blob/release-1.20/staging/src/k8s.io/kubelet/pkg/apis/stats/v1alpha1/types.go
[metrics-cadvisor]: https://github.com/google/cadvisor/blob/master/docs/storage/prometheus.md#prometheus-container-metrics

### Current State of These Metrics

Summary API has two interfaces:
* [cAdvisor stats provider](https://github.com/kubernetes/kubernetes/blob/release-1.20/pkg/kubelet/stats/cadvisor_stats_provider.go)
    * Calls cAdvisor directly to obtain node, pod, and container stats
* [CRI stats provider](https://github.com/kubernetes/kubernetes/blob/release-1.20/pkg/kubelet/stats/cri_stats_provider.go#L54)
    * Calls CRI implementation to provide some minimum container level stats, but still relies on cAdvisor for the majority of container level stats + pod level stats + node level stats

#### Current Fulfiller of Metrics Endpoints & Future Proposal 

Below is a table describing which stats come from what source now, as well a proposal of which should come from where in the future. It also includes which fields roughly correspond to fields in the `/metrics/cadvisor` endpoint, some of which will not come from the CRI for the first iteration of this KEP. See more below.

|Top level object              |`/stats/summary` Field|`/metrics/cadvisor` field                       |Level Needed in `/stats/summary`|Currently provided by:|Proposed to be provided by:|
|------------------------------|----------------------|------------------------------------------------|--------------------------------|----------------------|---------------------------|
|InterfaceStats (Network)      |RxBytes               |container_network_receive_bytes_total           |Pod                             |cAdvisor              |CRI                        |
|                              |RxErrors              |container_network_receive_errors_total          |Pod                             |cAdvisor              |CRI                        |
|                              |TxBytes               |container_network_transmit_bytes_total          |Pod                             |cAdvisor              |CRI                        |
|                              |TxErrors              |container_network_transmit_errors_total         |Pod                             |cAdvisor              |CRI                        |
|                              |N/A                   |container_network_receive_packets_dropped_total |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_network_receive_packets_total         |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_network_transmit_packets_dropped_total|N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_network_transmit_packets_total        |N/A                             |cAdvisor              |CRI or N/A                 |
|CPUStats                      |UsageNanoCores        |N/A                                             |Pod and Container               |cAdvisor              |CRI or Kubelet             |
|                              |UsageCoreNanoSeconds  |N/A                                             |Pod and Container               |CRI                   |CRI                        |
|                              |N/A                   |container_cpu_cfs_periods_total                 |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_cpu_cfs_throttled_periods_total       |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_cpu_cfs_throttled_seconds_total       |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_cpu_load_average_10s                  |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_cpu_system_seconds_total              |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_cpu_usage_seconds_total               |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_cpu_user_seconds_total                |N/A                             |cAdvisor              |CRI or N/A                 |
|MemoryStats                   |AvailableBytes        |N/A                                             |Pod and Container               |cAdvisor              |CRI                        |
|                              |UsageBytes            |container_memory_usage_bytes                    |Pod and Container               |cAdvisor              |CRI                        |
|                              |WorkingSetBytes       |container_memory_working_set_bytes              |Pod and Container               |CRI                   |CRI                        |
|                              |RSSBytes              |container_memory_rss                            |Pod and Container               |cAdvisor              |CRI                        |
|                              |PageFaults            |N/A                                             |Pod and Container               |cAdvisor              |CRI                        |
|                              |MajorPageFaults       |N/A                                             |Pod and Container               |cAdvisor              |CRI                        |
|                              |N/A                   |container_memory_cache                          |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_memory_failcnt                        |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_memory_failures_total                 |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_memory_mapped_file                    |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_memory_max_usage_bytes                |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_memory_swap                           |N/A                             |cAdvisor              |CRI or N/A                 |
|ProcessStats                  |ProcessCount          |container_processes                             |Pod                             |cAdvisor              |CRI                        |
|AcceleratorStats              |Make                  |N/A (too lazy to find the mapping)              |Container                       |cAdvisor              |cAdvisor or N/A            |
|                              |Model                 |N/A (too lazy to find the mapping)              |Container                       |cAdvisor              |cAdvisor or N/A            |
|                              |ID                    |N/A (too lazy to find the mapping)              |Container                       |cAdvisor              |cAdvisor or N/A            |
|                              |MemoryTotal           |N/A (too lazy to find the mapping)              |Container                       |cAdvisor              |cAdvisor or N/A            |
|                              |MemoryUsed            |N/A (too lazy to find the mapping)              |Container                       |cAdvisor              |cAdvisor or N/A            |
|                              |DutyCycle             |N/A (too lazy to find the mapping)              |Container                       |cAdvisor              |cAdvisor or N/A            |
|VolumeStats                   |All Fields            |N/A                                             |Pod                             |Kubelet               |Kubelet                    |
|Ephemeral Storage             |All Fields            |N/A                                             |Pod                             |Kubelet               |Kubelet                    |
|Rootfs.FsStats                |AvailableBytes        |N/A                                             |Container                       |cAdvisor or N/A       |CRI or N/A                 |
|                              |CapacityBytes         |container_fs_limit_bytes                        |Container                       |cAdvisor or N/A       |CRI or N/A                 |
|                              |UsedBytes             |container_fs_usage_bytes                        |Container                       |CRI                   |CRI                        |
|                              |InodesFree            |container_fs_inodes_free                        |Container                       |cAdvisor or N/A       |CRI or N/A                 |
|                              |Inodes                |container_fs_inodes_total                       |Container                       |cAdvisor or N/A       |CRI or N/A                 |
|                              |InodesUsed            |N/A                                             |Container                       |CRI                   |CRI                        |
|                              |N/A                   |container_fs_io_current                         |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_fs_io_time_seconds_total              |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_fs_io_time_weighted_seconds_total     |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_fs_read_seconds_total                 |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_fs_reads_bytes_total                  |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_fs_reads_merged_total                 |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_fs_reads_total                        |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_fs_sector_reads_total                 |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_fs_sector_writes_total                |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_fs_write_seconds_total                |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_fs_writes_bytes_total                 |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_fs_writes_merged_total                |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_fs_writes_total                       |N/A                             |cAdvisor              |CRI or N/A                 |
|UserDefinedMetrics            |All Fields            |N/A                                             |Container                       |cAdvisor              |CRI or N/A                 |
|No Equivalent in Stats Summary|N/A                   |container_scrape_error                          |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_sockets                               |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_spec_cpu_period                       |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_spec_cpu_quota                        |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_spec_cpu_shares                       |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_spec_memory_limit_bytes               |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_spec_memory_reservation_limit_bytes   |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_spec_memory_swap_limit_bytes          |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_start_time_seconds                    |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_tasks_state                           |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_threads                               |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_threads_max                           |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_file_descriptors                      |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_last_seen                             |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |                      |                                                |                                |cAdvisor              |CRI or N/A                 |

## Motivation

We want to avoid using cAdvisor for container & pod level stats and move metric collection to the CRI implementation for the following reasons:

* cAdvisor and metric dependency: CRI mission is not fully fulfilled - container runtime is not fully plugable.
* Break the monolithic design of cAdvisor, which needs to be aware of the underlying container runtime.
* Duplicate stats are collected by both cAdvisor and the CRI runtime, which can lead to:
    * Different information from different sources
    * Confusion from unclear origin of a given metric
    * Performance degradations (increased CPU / Memory / etc) [xref][perf-issue]
* Stats should be reported by the container runtime which knows behavior of the container/pod the best.
* cAdvisor only supports runtimes that run processes on the host, not e.g. VM based runtime like Kata Containers.
* cAdvisor only supports linux containers, not Windows ones.
* cAdvisor has big list of external dependencies since it needs to support multiple container runtimes itself - leads to large support surface.

[perf-issue]: https://github.com/kubernetes/kubernetes/issues/51798


### Goals

* Rely on a single component to provide the container and pod level metrics for the Summary API.
* Improve performance and reduce confusion on metrics collection in the Kubelet.
* Do not introduce breaking changes to the Summary API.
* Eliminate dependencies on container runtime clients used by cAdvisor.
* Enhance CRI implementations to provide metrics analogous to the existing metrics provided by `/metrics/cadvisor`.

### Non-Goals

- Have CRI support Volume Plugin stats (responsibility of the Kubelet).
- Have CRI support Ephemeral Storage stats (responsibility of the Kubelet).
- Have CRI support metrics of the host filesystem (responsibility of cAdvisor).
- Have CRI support Accelerator metrics stats (deprecated already in summary API).
- Have CRI support UserDefinedMetrics.
- Redefine which stats are reported in the Summary API (minus those that are being deprecated or not worth supporting in CRI)
- Propose alternatives to the Summary API
- Drop support for the fields in `/metrics/cadvisor`
- Support `/metrics/cadvisor` from the Kubelet longterm.

## Proposal

This KEP encompasses two related pieces of work (summary API and `/metrics/cadvisor`), and will require changes in three different components (CRI implementation, Kubelet, cAdvisor).
As such, this proposal will be broken up into what needs to be done for each of those pieces for each of those three component.

### Summary API

1. The [`ContainerStats`](https://github.com/kubernetes/cri-api/blob/release-1.20/pkg/apis/runtime/v1/api.proto#L1303-L1312) CRI message needs to be extended to include all of the fields from Summary's API's [`ContainerStats`](#summary-container-stats-object).
2. Add CRI Pod Level Stats message to CRI protobuf that includes all [Pod Level Stats](#summary-pod-stats-object) metrics from Summary API.
3. Add support for the new CRI additions in supported container runtimes (CRI-O and containerd).
4. Switch Kubelet's CRI stats provider from querying container and pod level stats from cAdvisor to newly added CRI pod and container level stats
5. cAdvisor should be updated to support no longer collecting stats that are duplicated with CRI implementation. Any client that requires the metrics that are reported by the CRI should gather them from the CRI instead of cAdvisor.

This will be described in more detail in the [design details section](#design-details)

[summary-pod-stats-object]: https://github.com/kubernetes/kubernetes/blob/release-1.20/staging/src/k8s.io/kubelet/pkg/apis/stats/v1alpha1/types.go#L102-L132

### /metrics/cadvisor

1. Expose the metric fields provided in `/metrics/cadvisor` in an analogous Prometheus endpoint directly from the CRI implementation.
5. cAdvisor should be updated to support no longer collecting stats that are duplicated with CRI implementation, and omit them from the report sent to `/metrics/cadvisor`.
3. The precise endpoint can change, but all the fields should be duplicated (so custom rules can be maintained).
4. Kubelet does not collect nor expose pod and container level metrics that were formally collected for and exposed by `/metrics/cadvisor`.

### User Stories [optional]

#### Story 1
As a cluster admin, I would like my node to be as performant as possible, and not waste resources by duplicating stats collection by CRI and cAdvisor.

#### Story 2
As a node maintainer, I would like to clarify the sources of truth for each stat the Kubelet reports to the Summary API, so that I can better find which component is to investigate for any issues.

### Notes/Constraints/Caveats (Optional)

#### History/Past Conversations

There have been conversations in the past on [deprecating the stats Summary API](#core-proposal), in favor of allowing an on-demand model of stats gathering/reporting.
This KEP does not attempt to implement that. These stats are largely considered [a part of Kubernetes](#keep-summary). The stated goal of this KEP is to reduce the overhead
on two entities reporting metrics, not totally changing what stats the Kubelet reports (with the exception of Accelerator and UserDefinedMetrics).

Thus, this KEP largely the plan described [here](#plan), with some changes:

- The CRI implementation will be responsible for the fields in the `/metrics/cadvisor` endpoint, though the name of the endpoint and location may change.
- CRI API is used for all of the monitoring endpoints related to Containers and Pods (except Volume and Ephemeral Storage)
- CRI API is used to provide metrics for eviction (as it relies on the summary API, which will be populated by the CRI implementation)

[core-proposal]: https://github.com/kubernetes/community/blob/master/contributors/design-proposals/instrumentation/core-metrics-pipeline.md#proposed-core-metrics
[keep-summary]: https://github.com/kubernetes/kubernetes/issues/68522#issuecomment-666636130
[plan]: https://github.com/kubernetes/kubernetes/issues/68522#issuecomment-724928827

#### Open Questions
1. For the newly introduced CRI API fields, should there be Windows and Linux specific fields?


### Risks and Mitigations

- To properly move to CRI stats, it is likely there are some metrics that we'll want to not support (Accelerator/UserDefined). We should be careful to not break entities that rely on these metrics.
- A large part of this work is changing the source of the Summary API metrics. In doing so, there is a risk that collecting from a new source will change how the stats look in aggregate (and risk bugs popping up in new areas).
- cAdvisor has a long history of collecting these stats. There is a risk that changing the source of the stats to the CRI implementation can cause performance regressions
as cAdvisor is fine tuned to perform in an adequate manner.
    - CRI implementations should do performance regression analyses to ensure the change does not regress too much.

## Design Details

### Stats Summary API

#### CRI Implementation
The CRI implementation will need to be extended to support reporting the full set of container-level from the [Summary API](#summary-container-stats-object).

##### ContainerStats additions
Currently, the CRI endpoints `{,List}ContainerStats` report the following fields for each container:
- CPU
    - `usage_core_nano_seconds`
- Memory
    - `working_set_bytes`
- Filesystem
    - `inodes_used`
    - `used_bytes`

These correspond to some fields of the [ContainerStats](#summary-container-stats-object) object reported by cAdvisor. Each of the CRI Stats fields will have to be extended to support the remaining fields. Here are some concrete additions the CRI will need, as well as notes showing what fields correspond to what between CRI and cAdvisor (note: a table version of this can be seen above):
```
=// CpuUsage provides the CPU usage information.
=message CpuUsage {
=    // Timestamp in nanoseconds at which the information were collected. Must be > 0.
+    // Corresponds to Stats Summary API CPUStats Time field
=    int64 timestamp = 1;
=    // Cumulative CPU usage (sum across all cores) since object creation.
+    // Corresponds to Stats Summary API CPUStats UsageCoreNanoSeconds
=    UInt64Value usage_core_nano_seconds = 2;
+    // Total CPU usage (sum of all cores) averaged over the sample window.
+    // The "core" unit can be interpreted as CPU core-nanoseconds per second.
+    UInt64Value usage_nano_cores = 3;
=}

=// MemoryUsage provides the memory usage information.
=message MemoryUsage {
=    // Timestamp in nanoseconds at which the information were collected. Must be > 0.
+    // Corresponds to Stats Summary API MemoryStats Time field
=    int64 timestamp = 1;
=    // The amount of working set memory in bytes.
+    // Corresponds to Stats Summary API MemoryStats WorkingSetBytes field
=    UInt64Value working_set_bytes = 2;
+    // Available memory for use. This is defined as the memory limit - workingSetBytes.
+    UInt64Value available_bytes = 3;
+    // Total memory in use. This includes all memory regardless of when it was accessed.
+    UInt64Value usage_bytes = 4;
+    // The amount of anonymous and swap cache memory (includes transparent hugepages).
+    UInt64Value rss_bytes = 5;
+    // Cumulative number of minor page faults.
+    UInt64Value page_faults = 6;
+    // Cumulative number of major page faults.
+    UInt64Value major_page_faults = 7;
=}
```

Notes:
- In Stats Summary API ContainerStats object, there's a timestamp field. We do not need such a field, as each struct in the ContainerStats object
  has its own timestamp, allowing CRI implementations flexibility when they collect which metrics.
- Notice the omission of the Stats Summary API ContainerStats Accelerators field.
  With its [deprecation](#accelerator-deprecation), it was deemed not worth implementing in the CRI.

[summary-container-stats-object]: https://github.com/kubernetes/kubernetes/blob/release-1.20/staging/src/k8s.io/kubelet/pkg/apis/stats/v1alpha1/types.go#L135-L160
[accelerator-deprecation]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/1867-disable-accelerator-usage-metrics/README.md

##### PodStats CRI additions
Previously, pod level stats were deemed the sole responsibility of the Kubelet. Currently, the Kubelet fetches pod level metrics by making use of cAdvisor to lookup the pod sandbox stats.
Instead of doing so, this KEP proposes introducing a new set of CRI API endpoints, structured similarly to the `ContainerStats` endpoints, dedicated to obtaining `PodSandbox` stats.

They will be defined as follows:

```
// Runtime service defines the public APIs for remote pod runtimes
service RuntimeService {
    ...
    // PodSandboxStats returns stats of the pod. If the pod sandbox does not
    // exist, the call returns an error.
    rpc PodSandboxStats(PodSandboxStatsRequest) returns (PodSandboxStatsResponse) {}
    // ListPodSandboxStats returns stats of the pods matching a filter.
    rpc ListPodSandboxStats(ListPodSandboxStatsRequest) returns (ListPodSandboxStatsResponse) {}
    ...
}
...
message PodSandboxStatsRequest {
    // ID of the pod sandbox for which to retrieve stats.
    string pod_sandbox_id = 1;
}

message PodSandboxStatsResponse {
    PodSandboxStats stats = 1;
}

// PodSandboxStatsFilter is used to filter the list of pod sandboxes to retrieve stats for.
// All those fields are combined with 'AND'.
message PodSandboxStatsFilter {
    // ID of the pod sandbox.
    string id = 1;
    // LabelSelector to select matches.
    // Only api.MatchLabels is supported for now and the requirements
    // are ANDed. MatchExpressions is not supported yet.
    map<string, string> label_selector = 2;
}

message ListPodSandboxStatsRequest {
    // Filter for the list request.
    PodSandboxStatsFilter filter = 1;
}

message ListPodSandboxStatsResponse {
    // Stats of the pod sandbox.
    repeated PodSandboxStats stats = 1;
}

// PodSandboxAttributes provides basic information of the pod sandbox.
message PodSandboxAttributes {
    // ID of the pod.
    string id = 1;
    // Metadata of the pod.
    PodSandboxMetadata metadata = 2;
    // Key-value pairs that may be used to scope and select individual resources.
    map<string,string> labels = 3;
    // Unstructured key-value map holding arbitrary metadata.
    // Annotations MUST NOT be altered by the runtime; the value of this field
    // MUST be identical to that of the corresponding PodSandboxStatus used to
    // instantiate the PodSandbox this status represents.
    map<string,string> annotations = 4;
}

// PodSandboxStats provides the resource usage statistics for a pod.
// The linux or windows field will be populated depending on the platform.
message PodSandboxStats {
    // Information of the pod.
    PodSandboxAttributes attributes = 1;
    // Stats from linux.
    LinuxPodSandboxStats linux = 2;
    // Stats from windows.
    WindowsPodSandboxStats windows = 3;
}

// LinuxPodSandboxStats provides the resource usage statistics for a pod sandbox on linux.
message LinuxPodSandboxStats {
    // CPU usage gathered for the pod sandbox.
    CpuUsage cpu = 1;
    // Memory usage gathered for the pod sandbox.
    MemoryUsage memory = 2;
    // Network usage gathered for the pod sandbox
    NetworkUsage network = 3;
    // Stats pertaining to processes in the pod sandbox.
    ProcessUsage process = 4;
    // Stats of containers in the measured pod sandbox.
    repeated ContainerStats containers = 5;
}

// WindowsPodSandboxStats provides the resource usage statistics for a pod sandbox on windows
message WindowsPodSandboxStats {
    // TODO: Add stats relevant to windows.
}

// NetworkUsage contains data about network resources.
message NetworkUsage {
    // The time at which these stats were updated.
    int64 timestamp = 1;
    // Stats for the default network interface.
    NetworkInterfaceUsage default_interface = 2;
    // Stats for all found network interfaces, excluding the default.
    repeated NetworkInterfaceUsage interfaces = 3;
}

// NetworkInterfaceUsage contains resource value data about a network interface.
message NetworkInterfaceUsage {
    // The name of the network interface.
    string name = 1;
    // Cumulative count of bytes received.
    UInt64Value rx_bytes = 2;
    // Cumulative count of receive errors encountered.
    UInt64Value rx_errors = 3;
    // Cumulative count of bytes transmitted.
    UInt64Value tx_bytes = 4;
    // Cumulative count of transmit errors encountered.
    UInt64Value tx_errors = 5;
}

// ProcessUsage are stats pertaining to processes.
message ProcessUsage {
    // The time at which these stats were updated.
    int64 timestamp = 1;
    // Number of processes.
    UInt64Value process_count = 2;
}
```

#### Kubelet

Once all required CRI changes are completed, Kubelet can update its CRI stats provider to stop fetching metrics from cAdvisor and instead obtain the metrics from the CRI for container and pods.

To do so, we propose to add a feature gate, that, when set, modifies the existing CRI stats provider by removing all usage of cAdvisor for pod and container level stats.
It will also configure cAdvisor to not report these stats.

As a note on that point: if users enable this behavior in alpha, and rely on `/metrics/cadvisor`, they would need to enable cAdvisor as a daemonset on the node.
There is no plan for the alpha iteration of this KEP to support `/metrics/cadvisor` coming from the built-in cAdvisor (when the feature gate is set).

Since all internal entities rely solely on the Summary API (eviction, preemption, metrics server), their needs will be satisfied by using the information gathered from the CRI.

For users that rely on `/metrics/cadvisor`, see the details below.

Additional work may be required to evaluate other kubelet components (e.g. eviction, preemption, etc) that may be relying on container or pod level metrics.
Ideally all components will rely on summary API thereby alleviating need for cAdvisor for container and pod level stats.
This is also a requirement to be able to disable cAdvisor container metrics collection.

#### cAdvisor

Once CRI and Kubelet stats provider level changes are in place, we can evaluate disabling cAdvisor from collecting container and pod level stats.
We may need to introduce new ability in cAdvisor to disable on-demand collection for certain cgroup hierarchies to ensure we can continue using cAdvisor for only node/machine level stats. 

### cAdvisor Metrics Endpoint

At this point, the `/metrics/cadvisor` endpoint (and the fields/labels contained within) are effectively part of the Kubernetes API.
There is a plethora of tooling built on the assumption that this information is present, and this cannot go away.

Simultaneously, it must move away from being broadcasted by cAdvisor. Otherwise, the community gains nothing from the advantages of moving away from cAdvisor for the summary API.

As such, the proposal is to move the fulfiller of the *fields* of `/metrics/cadvisor` to the CRI implementations, away from cAdvisor.

#### CRI implementations

Fulfilling `/metrics/cadvisor` poses a couple of issues with API stability and standardization.
Primarily, up until this proposal, the CRI API has been the source of common functionality between the CRI implementations.
However, for efficiency and simplicity, this proposal has deliberately chosen to *not* send the metrics required for `/metrics/cadvisor` (that aren't needed for the summary API) over the CRI.

Thus, a lot of the work needed to support these metrics directly from the CRI will involve deciding on how to standardize these metrics,
so users can rely on them as a plug-and-play interface between the different implementations.

The table above describes the various metrics that are in this endpoint.

Each compliant CRI implementation must:
- Have a location broadcasted about where these metrics can be gathered from. The endpoint name must not necessarily be `/metrics/cadvisor`, nor be gathererd from the same port as it was from cAdvisor
- Implement *all* metrics within the set of metrics that are decided on.
    - **TODO** How will we decide this set? We could support all, or take polls from the community and come up with a set of sufficiently useful metrics.
- Pass a set of tests in the critest suite that verify they report the correct values for *all* supported metrics labels (to ensure continued conformance and standardization).

Below is the proposed strategy for doing so:

1. The Alpha release will strictly cover research, performance testing and the creation of conformance tests.
    - Initial research on the set of metrics required should be done. This will, possibly, allow the community to declare metrics that are not required to be moved to the CRI implementations.
    - Testing on how performant cAdvisor+Kubelet are today should be done, to find a target, acceptable threshold of performance for the CRI implementations
    - Creation of tests verifying the metrics are reported correctly should be created and verified with the existing cAdvisor implementation.
2. For the Beta release, add initial support for CRI implementations to report these metrics
    - This set of metrics will be based on the research done in alpha
    - Each will be validated against the conformance and performance tests created in alpha.
3. For the GA release, the CRI implementation should be the source of truth for all pod and container level metrics that external parties rely on (no matter how many endpoints the Kubelet advertises).

#### cAdvisor

As a requirement for the Beta stage, cAdvisor must support optionally collecting and broadcasting these metrics, similarly to the changes needed for summary API.


### Test Plan

- Internally in the Kubelet, there should be integration tests verifying that information gotten from the two sources is not too different.
- Each CRI implementation should do regression testing on performance to make sure the gathering of these stats is reasonably efficient.
- Any identified external user of either of these endpoints (prometheus, metrics-server) should be tested to make sure they're not broken by API changes.

### Graduation Criteria
#### Alpha implementation

- CRI should be extended to provide required stats for `/stats/summary`
- Kubelet should be extended to provide the required stats from CRI implementation for `/stats/summary`.
- cAdvisor should be updated to support no longer collecting stats that are duplicated with CRI implementation.
- This new behavior will be gated by a feature gate to prevent regressions for users that rely on the old behavior.
- Conduct research to find the set of metrics from `/metrics/cadvisor` that compliant CRI implementations must expose.
- Conformance tests for the fields in `/metrics/cadvisor` should be created
- Performance tests for CPU/memory usage between Kubelet/cAdvisor/CRI implementation should be added.
#### Alpha -> Beta Graduation

- CRI implementations should report any difficulties collecting the stats, and by Beta the CRI stats implementation should perform better than they did with CRI+cAdvisor.
- CRI implementations should support their equivalent of `/metrics/cadvisor`, passing the performance and conformance suite created in Alpha.
- cAdvisor stats provider may be marked as deprecated (depending on stability of new CRI based implementations).
- cAdvisor should be able to optionally not report the metrics needed for both summary API and `/metrics/cadvisor`. This behavior will be toggled by the Kubelet feature gate.

#### Beta -> GA Graduation
- The CRI stats provider in the Kubelet should be fully formed, and able to satisfy all the needs of downstream consumers
- cAdvisor stats provider will likely be marked as deprecated (depending on dockershim deprecation).
- Feature gate removed and the CRI stats provider will no longer rely on cAdvisor for container/pod level metrics.

### Upgrade / Downgrade Strategy

- There needs to be a way for the Kubelet to verify the CRI provider is capable of providing the correct metrics.
  Upon upgrading to a version that relies on this new behavior (assuming the feature gate is enabled),
  Kubelet should fail early if the CRI implementation won't report the expected metrics.
- For Beta/GA releases, components that rely on `/metrics/cadvisor` should take the decided action (use `/stats/summary`, or use the Kubelet provided `/metrics/cadvisor`).

### Version Skew Strategy

- Breaking changes between versions will be mitigated by the FeatureGate.
    - By the time the FeatureGate is deprecated, it is expected the transition between CRI and cAdvisor is complete, and CRI has had at least one release to expose the required metrics (to allow for `n-1` CRI skew).
- In general, CRI should be updated in tandem with or before the Kubelet.

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

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: PodAndContainerStatsFromCRI
    - Components depending on the feature gate: Kubelet

* **Does enabling the feature change any default behavior?**
  Any change of default behavior may be surprising to users or break existing
  automations, so be extremely careful here.
  Enabling this behavior means some stats endpoints will not be filled:
  - some entries in `/metrics/cadvisor`
  - Accelerator and UserDefinedMetrics in `/stats/summary`

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes, assuming the Kubelet is restarted.

* **What happens if we reenable the feature if it was previously rolled back?**
  There should be no problem with this.

* **Are there any tests for feature enablement/disablement?**
  It will need to be (at least manually) tested against enabling/disabling on a live Kubelet.

Note: enabling/disabling feature gate will require cAdvisor is restarted. The most graceful way to make this happen is require the Kubelet restarts to apply these changes.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At a high level, this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide comprehensive guidance, but at the very
  high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
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


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  It should not.

* **Will enabling / using this feature result in introducing new API types?**
  Describe them, providing:
  - There will be new CRI API types, described above. These are to be agreed upon by Kubelet and the CRI implementation.

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**
  - No.
* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
  Describe them, providing:
  - There are no changes that affect objects stored in the database.
  - There are changes to the CRI API, which will have to be coordinated between CRI implementation and Kubelet.
 
* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.
  - The process of collecting and reporting the metrics should not differ too much between cAdvisor and the CRI implementation:
    - At a high level, both need to watch the changes to the stats (from cgroups, disk and network stats)
    - Once collected, the CRI implementation will need to report them (both through the CRI and eventually through the prometheus endpoint).
    - Both of these steps are already done by cAdvisor, so the work is changing hands, but not fundamentally changing.
  - It is possible the Alpha iteration of this KEP may affect CPU/memory usage on the node:
    - This may come because cAdvisor's performance has been fine-tuned, and changing the location of work may loose some optimizations.
    - However, it is explicitly stated that a requirement for transition from Alpha->Beta is little to no performance degradation.
    - The existence of the feature gate will allow users to mitigate this potential blip in performance (by not opting-in).
* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  - It most likely will reduce resource utilization. Right now, there is duplicate work being done between CRI and cAdvisor.
    This will not happen anymore.
  - The CRI implementation may scrape the metrics less efficiently than cAdvisor currently does. This should be measured and evaluated before Beta.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**
  - Should not change.
* **What are other known failure modes?**
  - Kubelet should fail early if problems are detected with version skew. Nothing else should be affected.

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

2021-1-27: KEP opened

## Drawbacks

CRI runtimes will each have to implement additional interface to support full stats, rather than all metric collection being unified by cAdvisor.
Note: This is by design as this will enable to decouple runtime implementation details further from Kubelet.

## Alternatives

- Instead of teaching CRI how to do *everything* cAdvisor does, we could instead have cAdvisor not do the work the CRI stats end up doing (specifically when reporting disk stats, which are the most expensive operation to report).
    - However, this doesn't address the anti-pattern of having multiple parties confusingly responsible for a wide array of metrics and other issues described.
- Have cAdvisor implement the summary API. A cAdvisor daemonset could be a drop-in replacement for the summary API.
- Don't keep supporting the summary API. Replace it with a "better" format, like prometheus. Or help users migrate to equivalent APIs that container runtimes already expose for monitoring.
