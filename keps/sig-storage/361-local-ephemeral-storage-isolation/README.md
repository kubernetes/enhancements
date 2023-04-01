# Local Ephemeral Storage Capacity Isolation

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
  - [Future Work](#future-work)
- [Proposal](#proposal)
  - [Ephemeral Storage Resource:](#ephemeral-storage-resource)
  - [Eviction Policy and Scheduler Predicates](#eviction-policy-and-scheduler-predicates)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
- [Graduation Criteria](#graduation-criteria)
  - [Phase 1: Alpha (1.8)](#phase-1-alpha-18)
  - [Phase 2: Beta](#phase-2-beta)
  - [Phase 3: GA (1.25)](#phase-3-ga-125)
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
  - [Version 1.8](#version-18)
  - [Version 1.10](#version-110)
  - [Version 1.25](#version-125)
- [Drawbacks [optional]](#drawbacks-optional)
- [Infrastructure Needed [optional]](#infrastructure-needed-optional)
- [References](#references)
<!-- /toc -->
<!-- /toc -->

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

In addition to persistent storage, pods and containers may require
ephemeral or transient local storage for scratch space, caching, and logs.
Unlike persistent storage, ephemeral storage is unstructured and shared,
the space, not the data, between all pods running on a node, in addition
to other uses by the system. 

Local storage capacity isolation feature was
introduced into Kubernetes via
<https://github.com/kubernetes/features/issues/361>.  It provides
support for capacity isolation of shared storage between pods, such
that a pod can be hard limited in its consumption of shared resources by
evicting Pods if its consumption of shared storage exceeds that
limit.  The limits and requests for shared ephemeral-storage are
similar to those for memory and CPU consumption.


## Motivation

Ephemeral local storage is exposed to pods via the container’s writable layer,
logs directory, and EmptyDir volumes. Pods use ephemeral local storage for
scratch space, caching and logs. There are many issues related to the lack
of local storage accounting and isolation, including:

* Pods do not know how much local storage is available to them.
* Pods cannot request “guaranteed” local storage.
* Local storage is a “best-effort” resource. Pods can get evicted due to other
  pods filling up the local storage, after which no new pods will be admitted
  until sufficient storage has been reclaimed.

### Goals

These goals apply only to local ephemeral storage, as described in
<https://github.com/kubernetes/features/issues/361>.

* Support local ephemeral storage (root partition only) as one of the node allocatable resources.
* Support local ephemeral storage (root partition only) isolation at pod and container levels.
* Support resource request/limit settings on local ephemeral storage


### Non-Goals

* Application to storage other than local ephemeral storage.
* Manage storage other than root partition
* Provide enforcement on ephemeral-storage usage under limit other than
* evicting pods. (Pod eviction is the only mechanism supported in this feature
  to limit the resource usage)
* Enforcing limits such that the pod would be restricted to the desired storage
  limit (The limit might be exceeded temporarily before Pod can be evicted).
* Support for I/O isolation using CFS & blkio cgroups.

### Future Work

* Enforce limits on per-volume storage consumption by using
  enforced project quotas.
* Provide other ways to limit storage usage other than evicting the whole Pod

## Proposal

To reduce the confusion and complexity caused by multiple storage APIs design, we use
one storage API to represent root partition only and manage its ephemeral storage isolation.
In this way, the management of local storage is more consistent with memory management
and easy to understand and configure. 

Resource types for storage:	
	// Local ephemeral storage for root partition
	ResourceEphemeralStorage ResourceName = "ephemeral-storage"


### Ephemeral Storage Resource:

* Container-level resource requirement: Currently Kubernetes only supports resource requirements at
  container level. So similar to CPU and memory, container spec can specify the request/limit
  for local storage. All the validation related to the storage resource requirements will be the
  same as memory resource. In the case of two partition, only the storage usage in root partition
  will be isolated across different containers. (In another word, second or more partitions
  will not be counted for container-level eviction management)
* Pod-level resource constraint: Since we haven’t supported pod-level resource API, the sum of the
  resource request/limit is considered as pod-level resource requirement. Similar to memory,
  all local ephemeral storage resource usage is subject to this requirement. For example,
  emptyDir disk usage (also secrets, configuMap, downwardAPI, gitRepo since they are wrapped
  emptyDir volume) plus containers disk usage should not exceed the pod-level resource requirement
  (the sum of all container’s limit). 
* EmptyDir SizeLimit: Because emptyDir volume is a pod-level resource which is managed separately,
  we also add a sizeLimit for emptyDir Volume for additional storage isolation. If this limit is
  set for emptyDir volume (default medium), eviction manager will validate this limit with emptyDir
  usage too in additional to the above pod-level resource constraints. 

### Eviction Policy and Scheduler Predicates

CPU and memory use cgroup for limiting the resource usage. However, cgroup for disk usage is not
available. Our current design is to evict pods when exceeding the limit set for local storage.
The eviction policy is listed as follows.

* If the container writable layer (overlay) usage exceeds its limit set for this container, pod gets evicted.
* If the emptyDir volume usage exceeds the sizeLimit, pod gets evicted.
* If the sum of the usage from emptyDir and all contains exceeds the sum of the container’s local storage
  limits (pod-level limit), pod gets evicted.

Pod Admission

* When scheduler admits pods, it sums up the storage requests from each container and pod can be scheduled
  only if the sum is smaller than the allocatable of the local storage space.

Resource Quota

This feature adds two more resource quotas for storage. The request and limit set constraints on the
total requests/limits of all containers’ in a namespace.

* requests.ephemeral-storage
* limits.ephemeral-storage

LimitRange 

* Similar to CPU and memory, admin could use LimitRange to set default container’s local storage
  request/limit, and/or minimum/maximum resource constraints for a namespace. 

Node Allocatable Resources

* Similar to CPU and memory, ephemeral-storage may be specified to reserve for kubelet or system.
  example,
  --system-reserved=[cpu=100m][,][memory=100Mi][,][ephemeral-storage=1Gi][,][pid=1000]
  --kube-reserved=[cpu=100m][,][memory=100Mi][,][ephemeral-storage=1Gi][,][pid=1000]

### Risks and Mitigations

* This feature introduces CPU and IO overhead due to the monitoring of disk usage.

* Pod will be evited when resource limit is exceeded. This might not be the ideal
  way of handling resource over use. 

* Before Pod is evicted, resource usage might temporally exceed the limit.

* Not all local storage is tracked by this feature. For example, [CSI ephmeral
  volumes](https://kubernetes.io/docs/concepts/storage/ephemeral-volumes/#csi-ephemeral-volumes),
  if they use node resources, will (probably) be accumulated into system
  resources. A KEP has been proposed in this area
  ([#1472](https://github.com/kubernetes/enhancements/issues/1472)). Details
  like CSI ephemeral volumes which share local storage resources across pods
  have not been addressed. We feel this is best tracked with the CSI ephemeral
  volumes feature.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.


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

- `<package>`: `<date>` - `<test coverage>`
eviction: pkg/kubelet/eviction/helpers_test.go#L702
eviction: pkg/kubelet/eviction/eviction_manager_test.go#L452
cm: pkg/kubelet/cm/container_manager_linux_test.go#L243
scheduler: pkg/scheduler/framework/plugins/noderesources/fit_test.go

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>
LocalStorageCapacityIsolationEviction: https://github.com/kubernetes/kubernetes/blob/master/test/e2e_node/eviction_test.go#L289
ephemeral storage resource limits: https://github.com/kubernetes/kubernetes/blob/master/test/e2e/scheduling/predicates.go


## Graduation Criteria

The following criteria applies to
`LocalStorageCapacityIsolation`:

### Phase 1: Alpha (1.8)

- basic support added in node/core APIs 
- support integrated in kubelet
- Alpha-level documentation
- Unit test coverage

### Phase 2: Beta

- node e2e test coverage
- monitoring matric

### Phase 3: GA (1.25)

- user feedback based on production experience

### Upgrade / Downgrade Strategy

Feature is enabled by default since 1.10
If feature is disabled when creating or updating cluster, in case of
request is set for ephemeral storage, scheduler will not take this ephemeral
storage request into consideration when scheduling pod. If limit is set, pod
will not be evicted due to ephemeral storage usage exceeding limit. sizeLimit
for emptyDir will also not be enforced.

If a pod had a limit, but the feature was disabled, then enabling the feature
could cause that pod to be evicted.

### Version Skew Strategy

N/A (Feature is enabled by default since 1.10)

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: LocalStorageCapacityIsolation
  - Components depending on the feature gate: kubelet and kube-apiserver

###### Does enabling the feature change any default behavior?

When LocalStorageCapacityIsolation is enabled for local ephemeral storage, users will
be able to manage ephemeral stoage the same way as other resources, CPU and memory. It
includes set container-level resource request/limit, reserve resources for kubelet and
system use, and also resoure quota.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature is supported (beta) and will fall back to the existing behavior.

###### What happens if we reenable the feature if it was previously rolled back?

The feature can work as expected when it is reenabled.

###### Are there any tests for feature enablement/disablement?

yes, we have unit tests cover this.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

In most cases, the rollout/rollback will not impact running workloads. If the
feature had been disabled, and a pod had a `sizeLimit` set, that pod could be
evicted after an upgrade as the `sizeLimit` starts being enforced.

###### What specific metrics should inform a rollback?

This feature is already beta since 1.10.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This feature is already beta since 1.10. No upgrade or rollback needed to test this.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?**

  - Search for pod spec about ephemeral-storage settings or emptyDir sizeLimit setting.
  For example, check `spec.containers[].resources.limits.ephemeral-storage` of each container.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**

- [x] Metrics
  - storage metrics from summary API
    Node-level: NodeStats {Fs (FsStats) }
    Pod-level: PodStats.VolumeStats {Name, PVCRef, FsStats }
    Container-level: ContainerStats {Rootfs (FsStats), Logs (FsStats) } 
  - Components exposing the metric: kubelet

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

  - 99.9% of volume stats calculation will cost less than 1s or even 500ms.
  It can be calculated by `kubelet_volume_metric_collection_duration_seconds` metrics.

* **Are there any missing metrics that would be useful to have to improve observability of this feature? **

  - Yes, there are no histogram metrics for each volume. The above metric was grouped by volume types because
    the cost for every volume is too expensive.

### Dependencies
* **Does this feature depend on any specific services running in the cluster? **

  - No


### Scalability
* **Will enabling / using this feature result in any new API calls?**
  - No.

* **Will enabling / using this feature result in introducing new API types?**
  - new resource type API "ResourceEphemeralStorage"

* **Will enabling / using this feature result in any new calls to the cloud
provider?**
  - No.

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**
  - No.

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**
  - No.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  - Yes. It will use CPU time and IO during ephemeral storage monitoring.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.
The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

- kubelet can continue to monitor disk usage, but pod will be evicted but not
  rescheduled.

###### What are other known failure modes?

1. If the ephemeral storage limitation is reached, the pod will be evicted by kubelet.

2. It should skip when the image is not configured correctly (unsupported FS or quota not enabled).

3. For "out of space" failure, kublet eviction should be triggered.


###### What steps should be taken if SLOs are not being met to determine the problem?

- Check the volume stats to see whether the current reported storage usage
  exceeds the limit. 


## Implementation History

### Version 1.8

- `LocalStorageCapacityIsolation` implemented at Alpha

### Version 1.10

- `kubelet_volume_metric_collection_duration_seconds` metrics was added
- Promoted to Beta

### Version 1.25

- Plan to promote `LocalStorageCapacityIsolation` to GA

## Drawbacks [optional]

## Infrastructure Needed [optional]

* Decision: who is responsible for quota management of all volume
  types (and especially ephemeral volumes of all types).  At present,
  emptydir volumes are managed by the kubelet and logdirs and writable
  layers by either the kubelet or the runtime, depending upon the
  choice of runtime.  Beyond the specific proposal that the runtime
  should manage quotas for volumes it creates, there are broader
  issues that I request assistance from the SIG in addressing.

* Location of the quota code.  If the quotas for different volume
  types are to be managed by different components, each such component
  needs access to the quota code.  The code is substantial and should
  not be copied; it would more appropriately be vendored.

## References
