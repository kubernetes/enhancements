---
title: New cgroup driver: "none" (for Rootless)
authors:
  - "@AkihiroSuda"
owning-sig: sig-node
reviewers:
  - TBD
approvers:
  - TBD
creation-date: 2019-06-04
last-updated: 2019-06-04
status: implementable
---

# New cgroup driver: "none" (for Rootless)

## Table of Contents

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->


- [Summary](#summary)
- [Motivation](#motivation)
  - [Future: cgroup2](#future-cgroup2)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details](#implementation-details)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
- [Feature Gate](#feature-gate)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Summary

Add "none" cgroup driver to kubelet so that kubelet (as well as CRI/OCI runtimes) can be executed without write permissions to cgroups.

Docker [v19.03+](https://github.com/moby/moby/pull/38050), containerd/CRI [20190104+](https://github.com/containerd/cri/pull/970), and CRI-O [v1.12+](https://github.com/cri-o/cri-o/pull/1729) already support running without cgroup permissions. So we only need to update kubelet.

Draft patch for kubelet is available in [Usernetes project](https://github.com/rootless-containers/usernetes): https://github.com/rootless-containers/usernetes/blob/2fd0ba0594673e0c5ea04a894191bfaa3b117377/src/patches/kubernetes/0003-kubelet-new-feature-gate-SupportNoneCgroupDriver.patch

An old version of our patch has been also used by Rancher's [k3s](https://github.com/rancher/k3s/commit/e397cb4e5b017805588ff223a63f42bff1f42d3a).

## Motivation

The motivation of adding "none" cgroup driver is to allow running the entire Kubernetes components (`kubelet`, CRI, OCI, CNI, and all `kube-*`) as a non-root user (w/o write access to `/sys/fs/cgroup`) on the host. ("Rootless mode").

The motivations of Rootless mode are:
* To protect the host from potential container-breakout vulnerabilities
* To allow users of shared machines (especially HPC) to run Kubernetes without the risk of breaking their colleagues' environments
* Safe Kubernetes-on-Kubernetes

Without cgroups, kubelet cannot support restricting and monitoring CPU/memory resources for pods, but pod processes can still spontaneously impose traditional `setrlimit` restrictions on themselves.

See [FOSDEM 2019 talk "Rootless Kubernetes"](https://www.slideshare.net/AkihiroSuda/rootless-kubernetes) by [@AkihiroSuda](https://github.com/AkihiroSuda) and [@giuseppe](https://github.com/giuseppe) for the overview of Rootless mode.

### Future: cgroup2

In future, we will be able to safely delegate cgroup permission to unprivileged users by using [cgroup2 unified-mode](https://systemd.io/CGROUP_DELEGATION.html#delegation).
So most users will no longer need to use the "none" cgroup driver.

However, the migration to cgroup2 is likely to take a couple of years or even more, because [cgroup1 controllers and cgroup2 controllers cannot be used simultaneously](https://systemd.io/CGROUP_DELEGATION.html#three-different-tree-setups-).
([Fedora 31](https://www.phoronix.com/scan.php?page=news_item&px=Fedora-31-Cgroups-V2-Default) is already planning to use cgroup2 by default, but anyway Kubernetes users will need to
change the GRUB configuration to use cgroup1, as Kubelet and CRI runtimes will not be able to support cgroup2 by the releae of Fedora 31.)

Also, even after the migration to cgroup2, containerized Kubernetes might not get support for cgroup2, because it is likely that [cgroup2 will only be supported via systemd](https://github.com/containers/libpod/issues/1429#issuecomment-465420444). So the "none" cgroup driver will still remain useful.

### Goals

* Support running kubelet without real cgroup access

### Non-Goals

* Support restricting and monitoring CPU/memory usage without cgroup

## Proposal

### Implementation Details

A new implementation of `pkg/kubelet/cm.CgroupManager` will be added:

```go
type noneCgroupManager struct {
	names map[string]struct{}
}

func (m *noneCgroupManager) Create(c *CgroupConfig) error {
	name := m.Name(c.Name)
	m.names[name] = struct{}{}
	return nil
}

func (m *noneCgroupManager) Destroy(c *CgroupConfig) error {
	name := m.Name(c.Name)
	delete(m.names, name)
	return nil
}

func (m *noneCgroupManager) Update(c *CgroupConfig) error {
	name := m.Name(c.Name)
	m.names[name] = struct{}{}
	return nil
}

func (m *noneCgroupManager) Exists(cgname CgroupName) bool {
	name := m.Name(cgname)
	_, ok := m.names[name]
	return ok
}

func (m *noneCgroupManager) Name(cgname CgroupName) string {
	return cgname.ToCgroupfs()
}

func (m *noneCgroupManager) CgroupName(name string) CgroupName {
	return ParseCgroupfsToCgroupName(name)
}

func (m *noneCgroupManager) Pids(_ CgroupName) []int {
	return nil
}

func (m *noneCgroupManager) ReduceCPULimits(cgroupName CgroupName) error {
	return nil
}

func (m *noneCgroupManager) GetResourceStats(name CgroupName) (*ResourceStats, error) {
	return &ResourceStats{
		MemoryStats: &MemoryStats{},
	}, nil
}
```

### Risks and Mitigations

Pods might be able to exhaust the host CPU/memory resources.
So it is recommended to restrict the total resource usage of the systemd user slice, e.g. `sudo systemctl set-property user-$UID.slice CPUQuota=...`.

The resource exhaustion could be also mitigated by modifying the Pod manifest to use traditional `setrlimit`.

## Design Details

### Test Plan

We should modify e2e test suites to support the "none" cgroup driver.

## Feature Gate

A new feature gate `SupportNoneCgroupDriver=true` will be needed to specify "none" as the cgroup driver.

## Implementation History

* 2018-07-20: Early POC implementation in [Usernetes project](https://github.com/rootless-containers/usernetes)
* 2018-08-17: [CRI-O got support for running without cgroup](https://github.com/cri-o/cri-o/pull/1729)
* 2019-01-04: [containerd/CRI got support for running without cgroup](https://github.com/containerd/cri/pull/970)
* 2019-02-04: [Docker got support for running without cgroup](https://github.com/moby/moby/pull/38050)
* 2019-04-10: [k3s kubelet got support for running without cgroup, by adopting an earlier version of the Usernetes patch](https://github.com/rancher/k3s/pull/195)
* 2019-06-04: present KEP to sig-node

## Drawbacks

* Pods will look like as if they are not consuming CPU and memory at all
* `spec.containers[].resources.limits` will be just ignored

## Alternatives

* `/sys/fs/cgroup` could be faked with FUSE (which can be unprivileged since kernel 4.18), but it seems impossible to fake `/proc/$PID/cgroup` with FUSE. So anyway we would need to add some special trick to kubelet implementation.
* [seccomp-trap-to-userspace (introduced in kernel 5.0)](https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/commit/?id=6a21cc50f0c7f87dae5259f6cfefe024412313f6) might be usable for faking `/sys/fs/cgroup` as well as `/proc/$PID/cgroup`, but it still seems difficult and too new. Also, its performance overhead needs to be evaluated.