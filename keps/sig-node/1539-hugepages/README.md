# HugePages

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories [optional]](#user-stories-optional)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    - [Feature Gate](#feature-gate)
    - [Node Specification](#node-specification)
    - [Pod Specification](#pod-specification)
    - [CRI Updates](#cri-updates)
    - [Cgroup Enforcement](#cgroup-enforcement)
    - [Limits and Quota](#limits-and-quota)
    - [Scheduler changes](#scheduler-changes)
    - [cAdvisor changes](#cadvisor-changes)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Huge pages as shared memory](#huge-pages-as-shared-memory)
    - [NUMA](#numa)
- [Graduation Criteria](#graduation-criteria)
- [Graduation Criteria for HugePageStorageMediumSize](#graduation-criteria-for-hugepagestoragemediumsize)
- [Test Plan](#test-plan)
- [Test Plan for HugePageStorageMediumSize](#test-plan-for-hugepagestoragemediumsize)
- [Production Readiness Review Questionnaire for HugePageStorageMediumSize](#production-readiness-review-questionnaire-for-hugepagestoragemediumsize)
  - [Monitoring requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
  - [Version 1.8](#version-18)
  - [Version 1.9](#version-19)
  - [Version 1.14](#version-114)
  - [Version 1.18](#version-118)
  - [Version 1.19](#version-119)
  - [Version 1.22](#version-122)
- [Release Signoff Checklist](#release-signoff-checklist)
<!-- /toc -->

## Summary

A proposal to enable applications running in a Kubernetes cluster to use huge
pages.

A pod may request a number of huge pages.  The `scheduler` is able to place the
pod on a node that can satisfy that request.  The `kubelet` advertises an
allocatable number of huge pages to support scheduling decisions. A pod may
consume hugepages via `hugetlbfs` or `shmget`.  Huge pages are not
overcommitted.

## Motivation

Memory is managed in blocks known as pages.  On most systems, a page is 4Ki. 1Mi
of memory is equal to 256 pages; 1Gi of memory is 256,000 pages, etc. CPUs have
a built-in memory management unit that manages a list of these pages in
hardware. The Translation Lookaside Buffer (TLB) is a small hardware cache of
virtual-to-physical page mappings.  If the virtual address passed in a hardware
instruction can be found in the TLB, the mapping can be determined quickly.  If
not, a TLB miss occurs, and the system falls back to slower, software based
address translation.  This results in performance issues.  Since the size of the
TLB is fixed, the only way to reduce the chance of a TLB miss is to increase the
page size.

A huge page is a memory page that is larger than 4Ki.  On x86_64 architectures,
there are two common huge page sizes: 2Mi and 1Gi.  Sizes vary on other
architectures, but the idea is the same.  In order to use huge pages,
application must write code that is aware of them.  Transparent Huge Pages (THP)
attempts to automate the management of huge pages without application knowledge,
but they have limitations.  In particular, they are limited to 2Mi page sizes.
THP might lead to performance degradation on nodes with high memory utilization
or fragmentation due to defragmenting efforts of THP, which can lock memory
pages. For this reason, some applications may be designed to (or recommend)
usage of pre-allocated huge pages instead of THP.

Managing memory is hard, and unfortunately, there is no one-size fits all
solution for all applications.

### Goals

This proposal only includes pre-allocated huge pages configured on the node by
the administrator at boot time or by manual dynamic allocation.    

### Non-Goals

This proposal defers issues relating to NUMA. It does not discuss how the
cluster could dynamically attempt to allocate huge pages in an attempt to find a
fit for a pod pending scheduling.  It is anticipated that operators may use a
variety of strategies to allocate huge pages, but we do not anticipate the
kubelet itself doing the allocation. Allocation of huge pages ideally happens
soon after boot time.

## Proposal

### User Stories [optional]

The class of applications that benefit from huge pages typically have
- A large memory working set
- A sensitivity to memory access latency

Example applications include:
- database management systems (MySQL, PostgreSQL, MongoDB, Oracle, etc.)
- Java applications can back the heap with huge pages using the
  `-XX:+UseLargePages` and `-XX:LagePageSizeInBytes` options.
- packet processing systems (DPDK)
- VMs running on top of Kubernetes infrastructure (libvirt, QEMU, etc.)

Applications can generally use huge pages by calling
- `mmap()` with `MAP_ANONYMOUS | MAP_HUGETLB` and use it as anonymous memory
- `mmap()` a file backed by `hugetlbfs`
- `shmget()` with `SHM_HUGETLB` and use it as a shared memory segment (see Known
  Issues).

1. A pod can use huge pages with any of the prior described methods.
1. A pod can request huge pages.
1. A scheduler can bind pods to nodes that have available huge pages.
1. A quota may limit usage of huge pages.
1. A limit range may constrain min and max huge page requests.

### Implementation Details/Notes/Constraints [optional]

#### Feature Gate

The proposal introduces huge pages as an Alpha feature.

It must be enabled via the `--feature-gates=HugePages=true` flag on pertinent
components pending graduation to Beta.

#### Node Specification

Huge pages cannot be overcommitted on a node.

A system may support multiple huge page sizes. For each supported huge page
size, the node will advertise a resource of the form `hugepages-<hugepagesize>`.
On Linux, supported huge page sizes are determined by parsing the
`/sys/kernel/mm/hugepages/hugepages-{size}kB` directory on the host. Kubernetes
will expose a `hugepages-<hugepagesize>` resource using binary notation form.
It will convert `<hugepagesize>` into the most compact binary notation using
integer values.  For example, if a node supports `hugepages-2048kB`, a resource
`hugepages-2Mi` will be shown in node capacity and allocatable values.
Operators may set aside pre-allocated huge pages that are not available for user
pods similar to normal memory via the `--system-reserved` flag.

There are a variety of huge page sizes supported across different hardware
architectures.  It is preferred to have a resource per size in order to better
support quota.  For example, 1 huge page with size 2Mi is orders of magnitude
different than 1 huge page with size 1Gi.  We assume gigantic pages are even
more precious resources than huge pages.

Pre-allocated huge pages reduce the amount of allocatable memory on a node. The
node will treat pre-allocated huge pages similar to other system reservations
and reduce the amount of `memory` it reports using the following formula:

```
[Allocatable] = [Node Capacity] - 
 [Kube-Reserved] - 
 [System-Reserved] - 
 [Pre-Allocated-HugePages * HugePageSize] -
 [Hard-Eviction-Threshold]
```

The following represents a machine with 10Gi of memory.  1Gi of memory has been
reserved as 512 pre-allocated huge pages sized 2Mi.  As you can see, the
allocatable memory has been reduced to account for the amount of huge pages
reserved.

```
apiVersion: v1
kind: Node
metadata:
  name: node1
...
status:
  capacity:
    memory: 10Gi
    hugepages-2Mi: 1Gi
  allocatable:
    memory: 9Gi
    hugepages-2Mi: 1Gi
...  
```

#### Pod Specification

Containers in a pod can cunsume huge pages by requesting pre-allocated
huge pages. In order to request huge pages, A pod spec must specify a certain
amount of huge pages using the resource `hugepages-<hugepagesize>` in container
object. The quantity of huge pages must be a positive amount of memory in bytes.
The specified amount must align with the `<hugepagesize>`; otherwise, the pod
will fail validation. For example, it would be valid to request
`hugepages-2Mi: 4Mi`, but invalid to request `hugepages-2Mi: 3Mi`.

The request and limit for `hugepages-<hugepagesize>` must match.  Similar to
memory, an application that requests `hugepages-<hugepagesize>` resource is at
minimum in the `Burstable` QoS class.

If multiple containers consume huge pages in the same pod, the request must be
made for each container. Similar to memory setting in cgroup sandbox, the sum of
huge page limits across the pod sets on pod cgroup sandbox, and the limit of
containers also sets on container cgroup sandboxes individually.

If a pod consumes huge pages via `shmget`, it must run with a supplemental group
that matches `/proc/sys/vm/hugetlb_shm_group` on the node.  Configuration of
this group is outside the scope of this specification.

A pod may consume multiple huge page sizes backed by the `hugetlbfs` in a single
pod spec. In this case it must use `medium: HugePages-<hugepagesize>` notation
for all volume mounts.

A pod may use `medium: HugePages` only if it requests huge pages of one
size.

In order to consume huge pages backed by the `hugetlbfs` filesystem inside the
specified container in the pod, it is helpful to understand the set of mount
options used with `hugetlbfs`.  For more details, see "Using Huge Pages" here:
https://www.kernel.org/doc/Documentation/vm/hugetlbpage.txt

```
mount -t hugetlbfs \
	-o uid=<value>,gid=<value>,mode=<value>,pagesize=<value>,size=<value>,\
	min_size=<value>,nr_inodes=<value> none /mnt/huge
```

The proposal recommends extending the existing `EmptyDirVolumeSource` to satisfy
this use case.  A new `medium=HugePages[-<hugepagesize>]` options would be
supported.  To write into this volume, the pod must make a request for huge
pages.  The `pagesize` argument is inferred from the medium of the mount if
`medium: HugePages-<hugepagesize>` notation is used. For `medium: HugePages`
notation the `pagesize` argument is inferred from the resource request
`hugepages-<hugepagesize>`.

The existing `sizeLimit` option for `emptyDir` would restrict usage to the
minimum value specified between `sizeLimit` and the sum of huge page limits of
all containers in a pod. This keeps the behavior consistent with memory backed
`emptyDir` volumes whose usage is ultimately constrained by the pod cgroup
sandbox memory settings.  The `min_size` option is omitted as its not necessary.
The `nr_inodes` mount option is omitted at this time in the same manner it is
omitted with `medium=Memory` when using `tmpfs`.

The following is a sample pod that is limited to 1Gi huge pages of size 2Mi and
2Gi huge pages of size 1Gi. It can consume those pages using `shmget()` or via
`mmap()` with the specified volume.

```
apiVersion: v1
kind: Pod
metadata:
  name: example
spec:
  containers:
...
    volumeMounts:
    - mountPath: /hugepages-2Mi
      name: hugepage-2mi
    - mountPath: /hugepages-1Gi
      name: hugepage-1gi
    resources:
      requests:
        hugepages-2Mi: 1Gi
        hugepages-1Gi: 2Gi
      limits:
        hugepages-2Mi: 1Gi
        hugepages-1Gi: 2Gi
  volumes:
  - name: hugepage-2mi
    emptyDir:
      medium: HugePages-2Mi
  - name: hugepage-1gi
    emptyDir:
      medium: HugePages-1Gi
```

For backwards compatibility, a pod that uses one page size should pass
validation if a volume emptyDir `medium=HugePages` notation is used.

The following is an example of a pod backward compatible with the
current implementation. It uses `medium: HugePages` notation and
requests hugepages of one size.

```
apiVersion: v1
kind: Pod
metadata:
  name: example
spec:
  containers:
...
    volumeMounts:
    - mountPath: /hugepages
      name: hugepage
    resources:
      requests:
        hugepages-2Mi: 1Gi
      limits:
        hugepages-2Mi: 1Gi
  volumes:
  - name: hugepage
    emptyDir:
      medium: HugePages
```

A pod that requests more than one page size should fail validation if a volume
emptyDir medium=HugePages is specified.

This is an example of an invalid pod that requests huge pages of two
differfent sizes, but doesn't use `medium: Hugepages-<size>` notation:

```
apiVersion: v1
kind: Pod
metadata:
  name: example
spec:
  containers:
...
    volumeMounts:
    - mountPath: /hugepages
      name: hugepage
    resources:
      requests:
        hugepages-2Mi: 1Gi
        hugepages-1Gi: 2Gi
      limits:
        hugepages-2Mi: 1Gi
        hugepages-1Gi: 2Gi
  volumes:
  - name: hugepage
    emptyDir:
      medium: HugePages
```

A pod that requests huge pages of one size and uses another size in a
volume emptyDir medium should fail validation.
This is an example of such an invalid pod. It requests hugepages of 2Mi
size, but specifies 1Gi in `medium: HugePages-1Gi`:

```
apiVersion: v1
kind: Pod
metadata:
  name: example
spec:
  containers:
...
    volumeMounts:
    - mountPath: /hugepages
      name: hugepage
    resources:
      requests:
        hugepages-2Mi: 1Gi
      limits:
        hugepages-2Mi: 1Gi
  volumes:
  - name: hugepage
    emptyDir:
      medium: HugePages-1Gi
```

Also, it is important to note that emptyDir usage is not required if pod
consumes huge pages via shmat/shmget system calls or mmap with MAP_HUGETLB.
This is an example of the pod that consumes 1Gi huge pages of size 2Mi and
2Gi huge pages of size 1Gi without using emptyDir:

```
apiVersion: v1
kind: Pod
metadata:
  name: example
spec:
  containers:
...
    resources:
      requests:
        hugepages-2Mi: 1Gi
        hugepages-1Gi: 2Gi
      limits:
        hugepages-2Mi: 1Gi
        hugepages-1Gi: 2Gi
```

The following is an example of the pod that requests multiple sizes of
huge pages for multiple containers. It requests 1Gi huge pages of size 1Gi and
2Mi for the container1 and 1Gi huge pages of size 2Mi for the container2 with
emptyDir backing. Note that `hugetlbfs` offers `size` mount option to specify
the maximum amount of memory for the mount, but huge pages medium does not use
the option to set limits so that the huge pages usage of containers will be
controlled by container cgroup sandboxes individually:
```
apiVersion: v1
kind: Pod
metadata:
  name: example
spec:
  containers:
  - name: container1
    volumeMounts:
    - mountPath: /hugepage-2Mi
      name: hugepage-2mi
    - mountPath: /hugepage-1Gi
      name: hugepage-1gi
    resources:
      requests:
        hugepages-2Mi: 1Gi
        hugepages-1Gi: 1Gi
      limits:
        hugepages-2Mi: 1Gi
        hugepages-1Gi: 1Gi
  - name: container2
    volumeMounts:
    - mountPath: /hugepage-2Mi
      name: hugepage-2mi
    resources:
      requests:
        hugepages-2Mi: 1Gi
      limits:
        hugepages-2Mi: 1Gi
  volumes:
  - name: hugepage-2mi
    emptyDir:
      medium: HugePages-2Mi
  - name: hugepage-1gi
    emptyDir:
      medium: HugePages-1Gi
```

#### CRI Updates

The `LinuxContainerResources` message should be extended to support specifying
huge page limits per size.  The specification for huge pages should align with
opencontainers/runtime-spec.
see:
https://github.com/opencontainers/runtime-spec/blob/master/config-linux.md#huge-page-limits

The runtime-spec provides the object `hugepageLimits` as an array of objects to
represent `hugetlb` controller. `hugepageLimits` allows specifying `limit` per
`pageSize`. The following is an example of the `hugepageLimits` object,
which has limits per 2MB and 64KB page size:

```
    "hugepageLimits": [
        {
            "pageSize": "2MB",
            "limit": 209715200
        },
        {
            "pageSize": "64KB",
            "limit": 1000000
        }
   ]
```

The `LinuxContainerResources` message can be extended to specify multiple sizes
to align with the runtime-spec in this way:
```
message LinuxContainerResources {
    ...
    string cpuset_mems = 7;
    // List of HugepageLimits to limit the HugeTLB usage of container per page size. Default: nil (not specified).
    repeated HugepageLimit hugepage_limits = 8;
}

// HugepageLimit corresponds to the file`hugetlb.<hugepagesize>.limit_in_byte` in container level cgroup.
// For example, `PageSize=1GB`, `Limit=1073741824` means setting `1073741824` bytes to hugetlb.1GB.limit_in_bytes.
message HugepageLimit {
    // The value of PageSize has the format <size><unit-prefix>B (2MB, 1GB),
    // and must match the <hugepagesize> of the corresponding control file found in `hugetlb.<hugepagesize>.limit_in_bytes`.
    // The values of <unit-prefix> are intended to be parsed using base 1024("1KB" = 1024, "1MB" = 1048576, etc).
    string page_size = 1;
    // limit in bytes of hugepagesize HugeTLB usage.
    uint64 limit = 2;
}
```

#### Cgroup Enforcement

To use this feature, the `--cgroups-per-qos` must be enabled.  In addition, the
`hugetlb` cgroup must be mounted.

The `kubepods` cgroup is bounded by the `Allocatable` value.

The QoS level cgroups are left unbounded across all huge page pool sizes.

The pod level cgroup sandbox is configured as follows, where `hugepagesize` is
the system supported huge page size(s).  If no request is made for huge pages of
a particular size, the limit is set to 0 for all supported types on the node.

```
pod<UID>/hugetlb.<hugepagesize>.limit_in_bytes = sum(pod.spec.containers.resources.limits[hugepages-<hugepagesize>])
```

If the container runtime supports specification of huge page limits, the
container cgroup sandbox will be configured with the specified limit.

The `kubelet` will ensure the `hugetlb` has no usage charged to the pod level
cgroup sandbox prior to deleting the pod to ensure all resources are reclaimed.

#### Limits and Quota

The `ResourceQuota` resource will be extended to support accounting for
`hugepages-<hugepagesize>` similar to `cpu` and `memory`.  The `LimitRange`
resource will be extended to define min and max constraints for `hugepages`
similar to `cpu` and `memory`.

#### Scheduler changes

The scheduler will need to ensure any huge page request defined in the pod spec
can be fulfilled by a candidate node.

#### cAdvisor changes

cAdvisor will need to be modified to return the number of pre-allocated huge
pages per page size on the node.  It will be used to determine capacity and
calculate allocatable values on the node.

### Risks and Mitigations

#### Huge pages as shared memory

For the Java use case, the JVM maps the huge pages as a shared memory segment
and memlocks them to prevent the system from moving or swapping them out.

There are several issues here:
- The user running the Java app must be a member of the gid set in the
  `vm.huge_tlb_shm_group` sysctl
- sysctl `kernel.shmmax` must allow the size of the shared memory segment
- The user's memlock ulimits must allow the size of the shared memory segment
- `vm.huge_tlb_shm_group` is not namespaced.

#### NUMA

NUMA is complicated.  To support NUMA, the node must support cpu pinning,
devices, and memory locality.  Extending that requirement to huge pages is not
much different.  It is anticipated that the `kubelet` will provide future NUMA
locality guarantees as a feature of QoS.  In particular, pods in the
`Guaranteed` QoS class are expected to have NUMA locality preferences.

## Graduation Criteria

- Reports of successful usage of the feature for isolating huge page resources.
- E2E testing validating its usage.
-- https://k8s-testgrid.appspot.com/sig-node-kubelet#node-kubelet-serial&include-filter-by-regex=Feature%3AHugePages

## Graduation Criteria for HugePageStorageMediumSize

- Reports of successful usage of the hugepage-<size> resources
- E2E testing validating its usage
-- https://k8s-testgrid.appspot.com/sig-node-kubelet#kubelet-serial-gce-e2e-hugepages

## Test Plan

- A test plan will consist of the following tests
  - Unit tests
    - Each unit test for enhancement will be implemented.
  - E2E tests
    - There is a test suit for huge pages in e2e test, it will be extended to validate enhancements.
    - here: https://github.com/kubernetes/kubernetes/blob/master/test/e2e_node/hugepages_test.go
  - cri-tools
    - Test case will be added to cri-tools to be used in CRI runtime' test(CI).
    - here: https://github.com/kubernetes-sigs/cri-tools

## Test Plan for HugePageStorageMediumSize

- Promote existing HugePages E2E tests to conformance

## Production Readiness Review Questionnaire for HugePageStorageMediumSize
### Monitoring requirements

* **How can an operator determine if the feature is in use by workloads?**
An operator could use hugepages-<size> resource limits and emptydir
mounts with medium: HugePage-<size> as described in the Kubernetes
documentation at https://kubernetes.io/docs/tasks/manage-hugepages/scheduling-hugepages

* **What are the SLIs (Service Level Indicators) an operator can use to determine.
the health of the service?**
  - [ ] Metrics
    - Metric name:
      `kube_pod_resource_request` and `kube_pod_resource_limit` for hugepages-<size> resources indicates usage.
    - Components exposing the metric: kube-scheduler

Workload performance can be measured by existing system metrics provided by Kubernetes components and e.g. [node_exporter](https://github.com/prometheus/node_exporter)

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

These will be set individually by application developers. This feature allows them to tune the performance of their workloads. See e.g. [Linux Huge Pages and virtual memory (VM) tuning](https://blog.yannickjaquier.com/linux/linux-hugepages-and-virtual-memory-vm-tuning.html)

* **Are there any missing metrics that would be useful to have to improve observability.
of this feature?**
No.

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
No

### Scalability

* **Will enabling / using this feature result in any new API calls?**
No.

* **Will enabling / using this feature result in introducing new API types?**
No

* **Will enabling / using this feature result in any new calls to the cloud.
provider?**
No

* **Will enabling / using this feature result in increasing size or count of.
the existing API objects?**
No

* **Will enabling / using this feature result in increasing time taken by any.
operations covered by [existing SLIs/SLOs]?**
No

* **Will enabling / using this feature result in non-negligible increase of.
resource usage (CPU, RAM, disk, IO, ...) in any components?**
No

### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?**
No impact.

* **What are other known failure modes?**
Not applicable.

* **What steps should be taken if SLOs are not being met to determine the problem?**
A cluster admin can tune the HugePage requests allocated to a workload by changing the available sizes, use the default HugePages configuration, or disable HugePages on the workload entirely.

## Implementation History

### Version 1.8

Initial alpha support for huge pages usage by pods.

### Version 1.9

Beta support for huge pages

### Version 1.14

GA support for huge pages proposed based on feedback from user community
using the feature without issue.

### Version 1.18

Extending of huge pages feature to support container isolation of huge pages and multiple sizes of huge pages.

### Version 1.19

Extending of huge pages test suite of E2E tests and cri-tools for enhancements after GA.

### Version 1.22

GA support of multiple huge page sizes proposed based on feedback from
user community using the feature without issue.

## Release Signoff Checklist
- \[x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- \[ ] KEP approvers have set the KEP status to `implementable`
  - The KEP is already implemented/GA.
- \[x] Design details are appropriately documented
- \[x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- \[x] Graduation criteria is in place
- \[x] "Implementation History" section is up-to-date for milestone
- \[x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- \[x] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes
