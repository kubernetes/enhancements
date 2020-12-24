---
title: image pull priority when serializing image pulls
authors:
  - "@saberrey"
  - "@pacoxu"
owning-sig: sig-node
participating-sigs:
reviewers:
  - "@odinuge"
  - "@smarterclayton"
approvers:
  - ""

editor: TBD
creation-date: 2020-12-24
last-updated: 2020-12-24
status: provisional
see-also:
replaces:
superseded-by:
---

# image pulls priority

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details](#implementation-details)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

When image pull is serialized, container runtime will pull image one by one. If there are some critical pods and others pods on the same server, image pull may start from no critical pods images.
Hence, we need to make priority for image pull on kubelet.

First of all, --serialize-image-pulls default is true.
> Pull images one at a time. We recommend *not* changing the default value on nodes that run docker daemon with version < 1.9 or an `aufs` storage backend. Issue #10959 has more details. (DEPRECATED: This parameter should be set via the config file specified by the Kubelet's `--config` flag. See https://kubernetes.io/docs/tasks/administer-cluster/kubelet-config-file/ for more information.)

If --serialize-image-pulls=false is set in your env, that would be great. You can ignore this.

## Motivation

From @saberrey,
> To mark a Pod as critical, we can set priorityClassName for that Pod to system-cluster-critical or system-node-critical，and we can also prefer to pull the image of the pod in the specified namespaces.

From @smarterclayton
> Mark nodes as "NotReady" until critical pods from daemonsets are Ready #75890

Other scenarios like 
- storage pod may start before those pods which would use pv
- kube-system pods need pull image before other pods

### Goals

- Improve the image pull order on kubelet
- Pull critical pods' images first.
- By default, pull images for kube-system namespace before other namespaces. 
- Further goals would be pods' images pull priority depends  on namespace's specified annotation. 


### Non-Goals

- when kubelet sets `--serialize-image-pulls=false`, or we may set the default value of --serialize-image-pulls to false in the future as docker daemon 1.9 is really old. And the backend fs can be checked automaticlly.


## Proposal

- critical pod first: priorityClassName of that Pod is system-cluster-critical or system-node-critical
- kube-system first: pull the image of the pod in the specified namespaces, default kube-system
- order others according to namespaces score annotations: namespace annotation `image-pull-priority=1`,default is 0. 

Other Proposal:  default use serialize image pulls false. Images will be pulled at the same time.
- check backend fs. If aufs, disable it
- check docker version before applying.


### Implementation Details

**Publishing Data:**

Beta:  


**Data Command Structure:**


**Example Command:**



### Risks and Mitigations

- Big image is not good practice, but may effect others.

## Graduation Criteria

- a stable order for node to run pods

## Alternatives



