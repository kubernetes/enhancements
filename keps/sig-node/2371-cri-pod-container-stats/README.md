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
        - [ContainerMetrics additions](#containermetrics-additions)
      - [Kubelet](#kubelet)
      - [cAdvisor](#cadvisor)
    - [cAdvisor Metrics Endpoint](#cadvisor-metrics-endpoint)
      - [CRI implementations](#cri-implementations)
      - [cAdvisor](#cadvisor-1)
      - [Windows](#windows)
    - [Test Plan](#test-plan)
        - [Prerequisite testing updates](#prerequisite-testing-updates)
        - [Unit tests](#unit-tests)
        - [Integration tests](#integration-tests)
        - [e2e tests](#e2e-tests)
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
   
|Top level object              |`/stats/summary` Field|`/metrics/cadvisor` field                       |Level Needed in `/stats/summary`|Currently provided by:|Proposed to be provided by:|Cgroup v1 stat:                         |Cgroup v2 stat:              |
|------------------------------|----------------------|------------------------------------------------|--------------------------------|----------------------|---------------------------|----------------------------------------|---------------------------|
|InterfaceStats (Network)      |RxBytes               |container_network_receive_bytes_total           |Pod                             |cAdvisor              |CRI                        |/sys/class/net/eth0/statistics/rx_bytes |/sys/class/net/eth0/statistics/rx_bytes
|                              |RxErrors              |container_network_receive_errors_total          |Pod                             |cAdvisor              |CRI                        |/sys/class/net/eth0/statistics/rx_errors|/sys/class/net/eth0/statistics/rx_errors
|                              |TxBytes               |container_network_transmit_bytes_total          |Pod                             |cAdvisor              |CRI                        |/sys/class/net/eth0/statistics/tx_bytes| /sys/class/net/eth0/statistics/tx_bytes
|                              |TxErrors              |container_network_transmit_errors_total         |Pod                             |cAdvisor              |CRI                        |/sys/class/net/eth0/statistics/tx_errors|/sys/class/net/eth0/statistics/tx_errors
|                              |N/A                   |container_network_receive_packets_dropped_total |N/A                             |cAdvisor              |CRI or N/A                 |/sys/class/net/eth0/statistics/rx_dropped|/sys/class/net/eth0/statistics/rx_dropped
|                              |N/A                   |container_network_receive_packets_total         |N/A                             |cAdvisor              |CRI or N/A                 |/sys/class/net/eth0/statistics/rx_packets|/sys/class/net/eth0/statistics/rx_packets
|                              |N/A                   |container_network_transmit_packets_dropped_total|N/A                             |cAdvisor              |CRI or N/A                 |/sys/class/net/eth0/statistics/tx_dropped|/sys/class/net/eth0/statistics/tx_dropped
|                              |N/A                   |container_network_transmit_packets_total        |N/A                             |cAdvisor              |CRI or N/A                 |/sys/class/net/eth0/statistics/tx_packets|/sys/class/net/eth0/statistics/tx_packets
|CPUStats                      |UsageNanoCores        |N/A                                             |Pod and Container               |cAdvisor              |CRI or Kubelet             |
|                              |UsageCoreNanoSeconds  |N/A                                             |Pod and Container               |CRI                   |CRI                        |
|                              |N/A                   |container_cpu_cfs_periods_total                 |N/A                             |cAdvisor              |CRI or N/A                 | (cpu.stat) nr_periods      |  (cpu.stat) nr_periods
|                              |N/A                   |container_cpu_cfs_throttled_periods_total       |N/A                             |cAdvisor              |CRI or N/A                 | (cpu.stat) nr_throttled    |  (cpu.stat) nr_throttled
|                              |N/A                   |container_cpu_cfs_throttled_seconds_total       |N/A                             |cAdvisor              |CRI or N/A                 | (cpu.stat) throttled_time  |  (cpu.stat) throttled_usec
|                              |N/A                   |container_cpu_load_average_10s                  |N/A                             |cAdvisor              |Removing this metric (not in v2)
|                              |N/A                   |container_cpu_system_seconds_total              |N/A                             |cAdvisor              |CRI or N/A                 | (cpuacct.stat) system      |  (cpu.stat) system_usec
|                              |N/A                   |container_cpu_usage_seconds_total               |N/A                             |cAdvisor              |CRI or N/A                 | (cpuacct.usage)            |  (cpu.stat) usage_usec
|                              |N/A                   |container_cpu_user_seconds_total                |N/A                             |cAdvisor              |CRI or N/A                 | (cpuacct.stat) user        |  (cpu.stat) user_usec
|MemoryStats                   |AvailableBytes        |N/A                                             |Pod and Container               |cAdvisor              |CRI                        | 
|                              |UsageBytes            |container_memory_usage_bytes                    |Pod and Container               |cAdvisor              |CRI                        |  memory.usage_in_bytes     |  memory.current
|                              |WorkingSetBytes       |container_memory_working_set_bytes              |Pod and Container               |CRI                   |CRI                        |  memory.usage_in_bytes (extra if logic) | memory.usage_in_bytes (extra if logic)
|                              |RSSBytes              |container_memory_rss                            |Pod and Container               |cAdvisor              |CRI                        | (memory.stat) total_rss    |  (memory.stat) anon
|                              |PageFaults            |N/A                                             |Pod and Container               |cAdvisor              |CRI                        | (memory.stat) pgfault      |  (memory.stat) pgfault 
|                              |MajorPageFaults       |N/A                                             |Pod and Container               |cAdvisor              |CRI                        | (memory.stat) pgmajfault   |  (memory.stat) pgmajfault
|                              |N/A                   |container_memory_cache                          |N/A                             |cAdvisor              |CRI or N/A                 | (memory.stat) cache        |  (memory.stat) file
|                              |N/A                   |container_memory_failcnt                        |N/A                             |cAdvisor              |CRI or N/A                 |  memory.failcnt                N/A
|                              |N/A                   |container_memory_failures_total                 |N/A                             |cAdvisor              |CRI or N/A                 | (memory.stat) pg_fault && pg_maj_fault |
|                              |N/A                   |container_memory_mapped_file                    |N/A                             |cAdvisor              |CRI or N/A                 | (memory.stat) mapped_file  |  (memory.stat) file_mapped
|                              |N/A                   |container_memory_max_usage_bytes                |N/A                             |cAdvisor              |CRI or N/A                 | memory.max_usage_in_bytes  |  memory.max
|                              |N/A                   |container_memory_swap                           |N/A                             |cAdvisor              |CRI or N/A                 | (memory.stat) swap  |  memory.swap.current - memory.current
|ProcessStats                  |ProcessCount          |container_processes                             |Pod                             |cAdvisor              |CRI                        |  Process
|AcceleratorStats              |Make                  |N/A (too lazy to find the mapping)              |Container                       |cAdvisor              |cAdvisor or N/A            |  accelerators/nvidia.go    | accelerators/nvidia.go
|                              |Model                 |N/A (too lazy to find the mapping)              |Container                       |cAdvisor              |cAdvisor or N/A            |  accelerators/nvidia.go    | accelerators/nvidia.go
|                              |ID                    |N/A (too lazy to find the mapping)              |Container                       |cAdvisor              |cAdvisor or N/A            |  accelerators/nvidia.go      |accelerators/nvidia.go
|                              |MemoryTotal           |N/A (too lazy to find the mapping)              |Container                       |cAdvisor              |cAdvisor or N/A            |  accelerators/nvidia.go      |accelerators/nvidia.go
|                              |MemoryUsed            |N/A (too lazy to find the mapping)              |Container                       |cAdvisor              |cAdvisor or N/A            |  accelerators/nvidia.go      |accelerators/nvidia.go 
|                              |DutyCycle             |N/A (too lazy to find the mapping)              |Container                       |cAdvisor              |cAdvisor or N/A            |  accelerators/nvidia.go      |accelerators/nvidia.go 
|VolumeStats                   |All Fields            |N/A                                             |Pod                             |Kubelet               |Kubelet                    |
|Ephemeral Storage             |All Fields            |N/A                                             |Pod                             |Kubelet               |Kubelet                    |
|Rootfs.FsStats                |AvailableBytes        |N/A                                             |Container                       |cAdvisor or N/A       |CRI or N/A                 |
|                              |CapacityBytes         |container_fs_limit_bytes                        |Container                       |cAdvisor or N/A       |CRI or N/A                 |/proc/diskstats | /proc/diskstats
|                              |UsedBytes             |container_fs_usage_bytes                        |Container                       |CRI                   |CRI                        |/proc/diskstats | /proc/diskstats
|                              |InodesFree            |container_fs_inodes_free                        |Container                       |cAdvisor or N/A       |CRI or N/A                 |/proc/diskstats | /proc/diskstats
|                              |Inodes                |container_fs_inodes_total                       |Container                       |cAdvisor or N/A       |CRI or N/A                 |/proc/diskstats | /proc/diskstats
|                              |InodesUsed            |N/A                                             |Container                       |CRI                   |CRI                        |
|                              |N/A                   |container_fs_io_current                         |N/A                             |cAdvisor              |CRI or N/A                 |/proc/diskstats | /proc/diskstats
|                              |N/A                   |container_fs_io_time_seconds_total              |N/A                             |cAdvisor              |CRI or N/A                 |/proc/diskstats | /proc/diskstats
|                              |N/A                   |container_fs_io_time_weighted_seconds_total     |N/A                             |cAdvisor              |CRI or N/A                 |/proc/diskstats | /proc/diskstats
|                              |N/A                   |container_fs_read_seconds_total                 |N/A                             |cAdvisor              |CRI or N/A                 |/proc/diskstats | /proc/diskstats
|                              |N/A                   |container_fs_reads_bytes_total                  |N/A                             |cAdvisor              |CRI or N/A                 |/proc/diskstats | /proc/diskstats
|                              |N/A                   |container_fs_reads_merged_total                 |N/A                             |cAdvisor              |CRI or N/A                 |/proc/diskstats | /proc/diskstats
|                              |N/A                   |container_fs_reads_total                        |N/A                             |cAdvisor              |CRI or N/A                 |/proc/diskstats | /proc/diskstats
|                              |N/A                   |container_fs_sector_reads_total                 |N/A                             |cAdvisor              |CRI or N/A                 |/proc/diskstats | /proc/diskstats
|                              |N/A                   |container_fs_sector_writes_total                |N/A                             |cAdvisor              |CRI or N/A                 |/proc/diskstats | /proc/diskstats
|                              |N/A                   |container_fs_write_seconds_total                |N/A                             |cAdvisor              |CRI or N/A                 |/proc/diskstats | /proc/diskstats
|                              |N/A                   |container_fs_writes_bytes_total                 |N/A                             |cAdvisor              |CRI or N/A                 |/proc/diskstats | /proc/diskstats
|                              |N/A                   |container_fs_writes_merged_total                |N/A                             |cAdvisor              |CRI or N/A                 |/proc/diskstats | /proc/diskstats
|                              |N/A                   |container_fs_writes_total                       |N/A                             |cAdvisor              |CRI or N/A                 |/proc/diskstats | /proc/diskstats
|UserDefinedMetrics            |All Fields            |N/A                                             |Container                       |cAdvisor              |CRI or N/A                 |
|No Equivalent in Stats Summary|N/A                   |container_scrape_error                          |N/A                             |cAdvisor              |CRI or N/A                 | error returning metrics | error returning metrics
|                              |N/A                   |container_sockets                               |N/A                             |cAdvisor              |CRI or N/A                 | cgroup.procs manipulation | cgroup.procs manipulation
|                              |N/A                   |container_spec_cpu_period                       |N/A                             |cAdvisor              |CRI or N/A                 | N/A                         |            cpu.max (2nd val)
|                              |N/A                   |container_spec_cpu_quota                        |N/A                             |cAdvisor              |CRI or N/A                 | N/A                         |         cpu.max (1st val)
|                              |N/A                   |container_spec_cpu_shares                       |N/A                             |cAdvisor              |CRI or N/A                 |                              |         cpu.weight
|                              |N/A                   |container_spec_memory_limit_bytes               |N/A                             |cAdvisor              |CRI or N/A                 | memory.        limit_in_bytes                                      memory.max
|                              |N/A                   |container_spec_memory_reservation_limit_bytes   |N/A                             |cAdvisor              |CRI or N/A                 | memory.soft_limit_in_bytes |                                     memory.high
|                              |N/A                   |container_spec_memory_swap_limit_bytes          |N/A                             |cAdvisor              |CRI or N/A                 | memory.memsw.limit_in_bytes  |                                    memory.swap.max
|                              |N/A                   |container_start_time_seconds                    |N/A                             |cAdvisor              |CRI or N/A                 | creation time of container | creation time of container
|                              |N/A                   |container_tasks_state                           |N/A                             |cAdvisor              |CRI or N/A                 |
|                              |N/A                   |container_threads                               |N/A                             |cAdvisor              |CRI or N/A                 | pids.curent | pids.curent
|                              |N/A                   |container_threads_max                           |N/A                             |cAdvisor              |CRI or N/A                 | pids.max | pids.max
|                              |N/A                   |container_file_descriptors                      |N/A                             |cAdvisor              |CRI or N/A                 | cgroup.procs manipulation | cgroup.procs manipulation
|                              |N/A                   |container_last_seen                             |N/A                             |cAdvisor              |CRI or N/A                 | now.Now().Unix()             | now.Now().Unix()
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
2. cAdvisor should be updated to support no longer collecting stats that are duplicated with CRI implementation, and omit them from the report sent to `/metrics/cadvisor`.
3. The precise endpoint can change, but all the fields should be duplicated (so custom rules can be maintained).
4. Kubelet does not collect nor expose pod and container level metrics that were formally collected for and exposed by `/metrics/cadvisor`.
5. Kubelet should broadcast the endpoint from the CRI, similarly to how it does for `/metrics/cadvisor`.

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
    For the alpha implementation, [PR #102789](https://github.com/kubernetes/kubernetes/pull/102789) added a Linux specific field
    `LinuxPodSandboxStats`. It also added Windows Specific field called  `WindowsPodSandboxStats` but left it blank.  

    Using the `WindowsPodSandboxStats` stats struct we will create new Windows specific fields that make sense for Windows stats.  The
    motivation behind this is Windows has differences in stats that are specific to its OS and doesn't currently fill certain fields (in some
    cases cannot such as `rss_bytes`). By adopting a Windows Specific set of stats it will allow for flexibity and customization in the
    future. 
    
    A challenge with new `WindowsPodSandboxStats` with custom fields is that the current calls to CRI endpoint `ListContainerStats` use the
    exiting `ContainerStats` object which is not Windows/Linux specific.  In the case of the CRI call `ListPodSandboxStats` it would
    currently return the new Windows specific stats propose here.  There will be a miss match in names of the fields but the underlying
    values will be the same.  The biggest difference initially will be that fields that were left blank in `ContainerStats` will not be on
    the new Windows specific structs.

    Making changes to `ListContainerStats` in backwards compatible way to use the Windows Specific stat structs is not in scope for this KEP.

    Alternatives include:

    - Add fields to `PodSandboxStats` which will be used by both Linux and Windows.  Windows and
    Linux would both use this field filling in stats that make most sense for them.  Windows is currently missing several stats but can
    reasonably fill in many of the missing fields (see the [Windows](#windows) section below).  This would be similar approach that is use
    today for the current CRI `ListContainerStats` call and this would make logic in kubelet more straight forward. This may have made sense
    if `LinuxPodSandboxStats` wasn't already created but would require re-work in kubelet and doesn't provide the flexibility for future customization based on OS.  
    - Leave the two fields `LinuxPodSandboxStats` and `WindowsPodSandboxStats` but use same fields.  This might be the easiest but is counter-intuitive and still adds complexity to the Kubelet implementation.  We also end up with Linux specific fields in the `WindowsPodSandboxStats`.

    See more information in the [ContainerStats](#containerstats-additions) and [Windows](#windows) sections below which gives details on proposal and Windows stats differences.

### Risks and Mitigations

- To properly move to CRI stats, it is likely there are some metrics that we'll want to not support (Accelerator/UserDefined). We should be careful to not break entities that rely on these metrics.
- A large part of this work is changing the source of the Summary API metrics. In doing so, there is a risk that collecting from a new source will change how the stats look in aggregate (and risk bugs popping up in new areas).
- cAdvisor has a long history of collecting these stats. There is a risk that changing the source of the stats to the CRI implementation can cause performance regressions
as cAdvisor is fine tuned to perform in an adequate manner.
    - CRI implementations should do performance regression analyses to ensure the change does not regress too much.

## Design Details

### Stats Summary API

#### CRI Implementation
The CRI implementation will need to be extended to support reporting the full set of container-level from the [Summary API](#summary-container-stats-object). A new gRPC call will also be added to the CRI that allows reporting for metrics currently exported by cAdvisor, but are outside the scope of the Summary API. This new gRPC call will return a Prometheus metric based response which Kubelet can export. Additionally, `PodAndContainerStatsFromCRI` feature gate support will be added to only report Prometheus based metrics from the CRI when calling `/metrics/cadvisor` endpoint when the feature gate is enabled. The additional metrics we support will need to be added to the individual container runtimes.
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
  // CPU usage gathered for the pod sandbox.
  WindowsCpuUsage cpu = 1;
  // Memory usage gathered for the pod sandbox.
  WindowsMemoryUsage memory = 2;
  // Network usage gathered for the pod sandbox
  WindowsNetworkUsage network = 3;
  // Stats pertaining to processes in the pod sandbox.
  WindowsProcessUsage process = 4;
  // Stats of containers in the measured pod sandbox.
  repeated WindowsContainerStats containers = 5;
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

// Windows specific fields. Many of these will look the same initially 
// this leave the ability to customize between Linux and Windows in the future
// Adding only fields we currently populate
message WindowsCpuUsage {
    int64 timestamp = 1;
    UInt64Value usage_core_nano_seconds = 2;
    UInt64Value usage_nano_cores = 3;
}

// MemoryUsage provides the memory usage information.
message WindowsMemoryUsage {
    int64 timestamp = 1;
    // The amount of working set memory in bytes.
    UInt64Value working_set_bytes = 2;
    UInt64Value available_bytes = 3;
    UInt64Value page_faults = 6;
}

message WindowsNetworkUsage {
    // The time at which these stats were updated.
    int64 timestamp = 1;
    WindowsNetworkInterfaceUsage default_interface = 2;
    repeated WindowsNetworkInterfaceUsage interfaces = 3;
}

message WindowsNetworkInterfaceUsage {
    string name = 1;
    UInt64Value rx_bytes = 2;
    UInt64Value rx_packets_dropped = 3;
    UInt64Value tx_bytes = 4;
    UInt64Value tx_packets_dropped = 5;
}

message WindowsProcessUsage {
    int64 timestamp = 1;
    UInt64Value process_count = 2;
}

message WindowsContainerStats {
    ContainerAttributes attributes = 1;
    WindowsCpuUsage cpu = 2;
    WindowsMemoryUsage memory = 3;
    WindowsFilesystemUsage writable_layer = 4;
}

message WindowsFilesystemUsage {
    int64 timestamp = 1;
    FilesystemIdentifier fs_id = 2;
    UInt64Value used_bytes = 3;
}

```

##### ContainerMetrics additions
For stats that are outside the scope of `/stats/summary` but are still reported by cAdvisor, we will return these as unstructured metrics in Prometheus format. The Kubelet will then implement collect methods and descriptors to fetch these metrics from the CRI and export them in Prometheus format. This is done via `ListPodSandboxMetrics` RPC call.

```
// ListPodSandboxMetrics gets pod sandbox metrics from CRI Runtime
rpc ListPodSandboxMetrics(ListPodSandboxMetricsRequest) returns (ListPodSandboxMetricsResponse) {}

message ListPodSandboxMetricsRequest {} 

message ListPodSandboxMetricsResponse {
    repeated PodSandboxMetrics pod_metrics = 1;
}

message PodSandboxMetrics {
    string pod_sandbox_id = 1;
    repeated Metric metrics = 2;
    repeated ContainerMetrics container_metrics = 3;
}

message ContainerMetrics {
    string container_id = 1;
    repeated Metric metrics = 2;
}

message Metric {
    //timestamp=0 indicates the metrics returned are cached
    int64 timestamp = 1;
    repeated LabelPair labels = 2;
    MetricType metric_type = 3;
    Int64Value value = 4;
}

message LabelPair {
    string name = 1;
    string value = 2;
}

enum MetricType {
    COUNTER = 0;
    GAUGE = 1;
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

1. The Alpha release will focus solely on `/stats/summary` endpoint, and `/metrics/cadvisor` support will follow in Beta.
2. For the Beta release, add initial support for CRI implementations to report these metrics
    - Initial research on the set of metrics required should be done. This will, possibly, allow the community to declare metrics that are not required to be moved to the CRI implementations.
    - Testing on how performant cAdvisor+Kubelet are today should be done, to find a target, acceptable threshold of performance for the CRI implementations
    - Creation of tests verifying the metrics are reported correctly should be created and verified with the existing cAdvisor implementation.
3. For the GA release, the CRI implementation should be the source of truth for all pod and container level metrics that external parties rely on (no matter how many endpoints the Kubelet advertises).

#### cAdvisor

As a requirement for the Beta stage, cAdvisor must support optionally collecting and broadcasting these metrics, similarly to the changes needed for summary API.

#### Windows

Windows currently does a best effort at filling out the stats in `/stats/summary` and misses some stats either because those are not exposed or they are not supported. 

Another aspect for Windows to consider is that work is being done to create [Hyper-v containers](https://github.com/containerd/containerd/issues/6862). We will want to make sure we have an intersection of stats that support Hyper-v as well.
It was [discussed](https://github.com/kubernetes/kubernetes/pull/110754#issuecomment-1176531055) if we want a separate set of stats specific for Window Hyper-v vs process isolated but decided that 
these stats should be generic and not expose implementation details of the pod sandbox.  
More detailed stats could be collected by external tools if required.

The current set of stats that are used by windows in the `ListContainerStats` API:

**cpu usage** - https://github.com/microsoft/hcsshim/blob/master/cmd/containerd-shim-runhcs-v1/stats/stats.proto			

| field                      | type   | process isolated field | hyperv filed     | notes                                                                                                                                                                                                                |
| -------------------------- | ------ | ---------------------- | ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| timestamp                  | int64  | ✅                      | ✅                |                                                                                                                                                                                                                      |
| usage\_core\_nano\_seconds | uint64 | TotalRuntimeNS         | TotalRuntimeNS   | // Cumulative CPU usage (sum across all cores) since object creation.                                                                                                                                                |
| usage\_nano\_cores         | uint64 | calculated value       | calculated value | calculated value done runtime (containerd or kubelet)<br><br> // Total CPU usage (sum of all cores) averaged over the sample window.  <br> // The "core" unit can be interpreted as CPU core-nanoseconds per second. |

**Memory usage** - https://github.com/microsoft/hcsshim/blob/master/cmd/containerd-shim-runhcs-v1/stats/stats.proto			

| field               | type   | process isolated field                      | hyperv filed         | notes                                                                                                                                                               |
| ------------------- | ------ | ------------------------------------------- | -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| timestamp           | int64  | ✅                                           | ✅                    |                                                                                                                                                                     |
| working\_set\_bytes | uint64 | memory\_usage\_private\_working\_set\_bytes | working\_set\_bytes? | // The amount of working set memory in bytes.<br><br>Is hyper-v working\_set\_bytes same as private\_working\_set\_bytes                                            |
| available\_bytes    | uint64 |                                             | available\_memory    |   We should be able to return this. It is the limit set on the job object.  https://docs.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-jobobject_extended_limit_information <br><br>  // Available memory for use. This is defined as the memory limit - workingSetBytes.<br><br> |
| usage\_bytes        | uint65 | ?                                           | ?                    | // Total memory in use. This includes all memory regardless of when it was accessed. Is this cumultive memory usage?   |
| rss\_bytes          | uint66 | n/a                                         | n/a                  | windows doesn't have rss. Cannot report rss                                                                                                                                              |
| page\_faults        | uint67 |                                             |                      | not reported. It may be possible use `TotalPageFaultCount` from https://docs.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-jobobject_basic_accounting_information                            |
| major\_page\_faults | uint68 |                                             |                      | not reported. Windows does not make a distinction here                                                                                                          |

**Process isolated** also has

- memory_usage_commit_bytes
- memory_usage_commit_peak_bytes 

**Hyperv** also has 

- virtual_node_count
- available_memory_buffer
- reserved_memory 
- assigned_memory 
- slp_active 
- balancing_enabled
- dm_operation_in_progress 

These are not used currently but are be very specific to each implementation and could be collected by specialized tools on a as needed basis.

**Network stats**
| field      | type   | process isolated field | hyperv filed | notes                                                                                                                                                 |
| ---------- | ------ | ---------------------- | ------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| timestamp  | int64  | ✅                      | ✅            |                                                                                                                                                       |
|  name      | uint64 |      EndpointID                  |              | same values can be used for hyperv                                                                                                                                                    |
| rx\_bytes  | uint65 | BytesReceived          |              |   same values can be used for hyperv                                                                                                                                                   |
| rx\_errors | uint66 |                        |              | can we use DroppedPacketsIncoming. https://github.com/microsoft/hcsshim/blob/949e46a1260a6aca39c1b813a1ead2344ffe6199/internal/hns/hnsendpoint.go#L65 |
| tx\_bytes  | uint67 | BytesSent              |              |  same values can be used for hyperv                                                                                            |
| tx\_errors | uint68 |                        |              | should use DroppedPacketsOutgoing                                                                                                                                |

Based on the above settings, we should stay conservative and expose the existing set of working overlapping stats.  This is what is proposed in the [changes cri](#cri-implementation)

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

- `pkg/kubelet/server/stats`: 06-15-2022 - 74.9

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- Internally in the Kubelet, there should be integration tests verifying that information gotten from the two sources is not too different.
- Each CRI implementation should do regression testing on performance to make sure the gathering of these stats is reasonably efficient.
- Any identified external user of either of these endpoints (prometheus, metrics-server) should be tested to make sure they're not broken by API changes.


##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- A test using the CRI stats feature gate with enabled CRI implementations should be used with cri_stats_provider to ensure the stats reported are conformant.

### Graduation Criteria
#### Alpha implementation

- CRI should be extended to provide required stats for `/stats/summary`
- Kubelet should be extended to provide the required stats from CRI implementation for `/stats/summary`.
- This new behavior will be gated by a feature gate to prevent regressions for users that rely on the old behavior.
- cAdvisor should be able to optionally not report the metrics needed for both summary API and `/metrics/cadvisor`. This behavior will be toggled by the Kubelet feature gate.
- Kubelet will query the CRI implementation for endpoints to broadcast from its own server.
	- This will allow the CRI to broadcast `/metrics/cadvisor` through the Kubelet's HTTP server.

#### Alpha -> Beta Graduation

- Conduct research to find the set of metrics from `/metrics/cadvisor` that compliant CRI implementations must expose.
- Conformance tests for the fields in `/metrics/cadvisor` should be created.
- Validate performance impact of this feature is within allowable margin (or non-existent, ideally).
	- The CRI stats implementation should perform better than they did with CRI+cAdvisor.
- cAdvisor stats provider may be marked as deprecated (depending on stability of new CRI based implementations).

#### Beta -> GA Graduation

- The CRI stats provider in the Kubelet should be fully formed, and able to satisfy all the needs of downstream consumers
- cAdvisor stats provider will likely be marked as deprecated (depending on dockershim deprecation).
- Feature gate removed and the CRI stats provider will no longer rely on cAdvisor for container/pod level metrics.

### Upgrade / Downgrade Strategy

- There needs to be a way for the Kubelet to verify the CRI provider is capable of providing the correct metrics.
  Upon upgrading to a version that relies on this new behavior (assuming the feature gate is enabled),
  Kubelet should fallback to use cAdvisor if the CRI implementation won't report the expected metrics.
- For Beta/GA releases, components that rely on `/metrics/cadvisor` should take the decided action (use `/stats/summary`, or use the CRI provided replacement for `/metrics/cadvisor`).

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

If the CRI implementation doesn't support the required metrics, and cAdvisor has container metrics collection turned off,
it is possible the node comes up with no metrics about pods and containers. This should be mitigated by making sure that
the kubelet probes the CRI implementation and enables cAdvisor metrics collection even if the feature gate is on.

* **What specific metrics should inform a rollback?**

The lack of any metrics reported for pods and containers is the worst case scenerio here, and would require either a rollback or for the feature gate to be disabled.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.

The source of the metrics is a private matter between the kubelet, CRI implementation and cAdvisor. Since cAdvisor
in embedded in the kubelet, the two pieces that could move disjointly are kubelet and CRI implementation. The
quality of the metrics reported by the kubelet/CRI are dependent solely on the Kubelet's configuration at runtime. In other
words, rolling back and upgrading should have no affect--if the upgrade broke metrics because the CRI didn't support them
(and measures weren't taken to cause kubelet to fallback to cAdvisor), then a rollback (or toggling of the feature gate)
would return the metrics from cAdvisor.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

A piece of work for Beta is moving the source of the contents of `/metrics/cadvisor`. If users toggle the feature gate,
prometheus collectors will have to move the URL. However, it's an expressed intention of the implementation to have the CRI
report metrics previously reported by cAdvisor, so the contents should not change.


### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

The source of the pod and container metrics previously reported to Prometheus by `/metrics/cadvisor` is the CRI implementation, not cAdvisor.
Further, if the CRI implementation was using the old CRI stats provider, then the memory usage of the cgroup the kubelet and runtime
were in should go down--as some duplicated work should be unduplicated.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [x] Metrics
    - Metric name:
        - all pod and container level stats coming from cAdvisor `container_*`
    - Components exposing the metric:
        - Previously cAdvisor, now CRI implementation.
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

- Reduction of CPU and memory usage between kubelet and CRI (if previously using CRI stats provider).
- Minimal (< 2%) of performance hit between CPU and memory between CRI and kubelet (if previously using cAdvisor stats provider).

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


  - CRI implementation
    - Usage description:
      - Impact of its outage on the feature: The feature, as well as many other pieces of Kubernetes, would not work, as the CRI implementation is vital to the creation and running of Pods.
      - Impact of its degraded performance or high-error rates on the feature: All Kuberetes operations will slow down if the CRI spends too much energy in getting the stats.


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
  - The CRI implementation may scrape the metrics less efficiently than cAdvisor currently does. This should be measured and evaluated as a requirement of Beta.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**
  - Should not change.
* **What are other known failure modes?**
  - Kubelet should fall back to using cAdvisor if errors are detected with version skew. Nothing else should be affected.

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

2021-01-27: KEP opened
2021-05-12: KEP merged, targeted at Alpha in 1.22
2021-07-08: KEP deemed not ready for Alpha in 1.22
2021-12-07: KEP successfully implemented at Alpha in 1.23
2022-01-25: KEP targeted at Beta in 1.24
2022-04-20: KEP deemed not ready for Beta in 1.24
2022-06-13: Move some Beta criteria to Alpha criteria in 1.25

## Drawbacks

CRI runtimes will each have to implement additional interface to support full stats, rather than all metric collection being unified by cAdvisor.
Note: This is by design as this will enable to decouple runtime implementation details further from Kubelet.

Support for full /metrics/cadvisor endpoint is not enforced, and individual container runtimes can return different metrics as they see fit.

Greater complexity as opposed to adding these unstructured metrics directly into the CRI, and additional overhead with RPC call and converting between Prometheus, CRI, and back. 

## Alternatives

- Instead of teaching CRI how to do *everything* cAdvisor does, we could instead have cAdvisor not do the work the CRI stats end up doing (specifically when reporting disk stats, which are the most expensive operation to report).
    - However, this doesn't address the anti-pattern of having multiple parties confusingly responsible for a wide array of metrics and other issues described.
- Have cAdvisor implement the summary API. A cAdvisor daemonset could be a drop-in replacement for the summary API.
- Don't keep supporting the summary API. Replace it with a "better" format, like prometheus. Or help users migrate to equivalent APIs that container runtimes already expose for monitoring.
