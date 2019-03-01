---
title: Separating a CRI for docker from Kubelet 
authors:
  - "@resouer"
  - "@dims"
  - "@zhangxiaoyu-zidif"
owning-sig: sig-node
reviewers:
  - "@yujuhong"
  - "@dchen1107"
  - "@derekwaynecarr"
  - "@PatrickLang"
approvers:
  - "@DawnChen"
  - "@yujuhong"
creation-date: 2019-02-26
last-updated: 2019-02-26
status: provisional
---

# Separating a CRI for Docker from Kubelet

## Table of Contents

- [Terms](#terms)
- [Summary](#summary)
- [Motivation](#motivation)
  * [Pros](#pros)
  * [Cons](#cons)
  * [Goals](#goals)
  * [Non-Goals](#non-goals)
- [Proposal](#proposal)
  * [Dockershim deprecation plan](#dockershim-deprecation-plan)
  * [Dockershim deprecation criteria](#dockershim-deprecation-criteria)
  * [Test Plan](#test-plan)
  * [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)

## Terms

- **CRI:** Container Runtime Interface – a plugin interface which enables kubelet to use a wide variety of container runtimes, without the need to recompile.


## Summary

CRI for docker (i.e. dockershim) is currently a built-in container runtime in kubelet code base. This proposal aims at a concrete deprecation and migration plan for separating dockershim from kubelet to out-of-tree without breaking current production users and WIP engineering efforts.

## Motivation

In Kubernetes, CRI is the used as the "default" container runtime, while currently the CRI of docker (a.k.a. dockershim) is part of kubelet code and coupled with kubelet's lifecycle. 

This is not ideal as kubelet then has dependency on specific container runtime which leads to maintenance burden for not only developers in sig-node, but also cluster administrators when critical issues (e.g. runc CVE) happen to container runtimes. The pros of moving dockershim to out-of-tree is straightforward:

### Pros
- Docker is not special and just a CRI just like every other CRI in our ecosystem.
- Currently, dockershim "enjoys" some backdoors for various reasons. Deprecating these "features" should eliminate maintenance burden of kubelet.
- A cri-dockerd can be maintained independently.
- Over time we can remove vendored docker dependencies in kubelet.

Having said that, cons of deprecation built-in dockershim requires lots of attentions:

### Cons
- Deployment pain with a new binary in addition to kubelet.
  - An additional component may aggravate the complexity currently. It may be relieved with docker version evolutions.
- The number of affected users maybe large.
  - Users must change existing use experience when using Kubernetes and docker.
  - Users have to change their existing workflows to adapt to this new changes.
  - And other unrecorded stuff.
- Updating all the eco-system tools to support the new cri-dockerd.
- Many people use the built in dockershim for in-cluster image build. While that may not be something we recommend for a variety of reasons, it will be a breaking change for these users.
- CRI is still in alpha，should probably get a 1.0 out there splitting out dockershim completely from kubelet.
- Existing CNI and CSI plugins may also be affected.
  - Current dockershim has independent module interacting with CNI plugins. After migrating dockershim out of Kubelet, it may affect some processes between dockershim and CNI plugins.
- cri-dockerd will vendor kubernetes/kubernetes, that may be tough.
- cri-dockerd as an independent software running on node should be allocated enough resource to guarantee its availability.

> You can check [the discussion in sig-node mailing list](https://groups.google.com/forum/#!msg/kubernetes-sig-node/0qVzfugYhro/l6Au216XAgAJ) for more details. 

Based on all the discussion, we agree that we should not rush to immediate decision. At the same time, it's the right time to start designing and documenting dockershim deprecation criteria and plan, which will be the main content of rest of this KEP.

### Goals

- A concrete dockershim deprecation criteria.
- A brief plan to deprecate dockershim spanning multiple releases.

### Non-Goals

- Deprecation of dockershim immediately without consideration for users and WIP efforts depending on it.
- Refactoring or re-design of dockershim itself due to deprecation.

## Proposal

### Dockershim deprecation criteria

- CRI itself is beta.
- kubelet has no dependency on dockershim/docker in its whole lifecycle. 
- All node related features are CRI generic and have no "back door" dependency on dockershim/docker.
- Deprecate and remove, or replace all Docker-specific features.
- Reasonable benchmark result of performance degradation after moving dockershim to out-of-tree.
- A out-of-tree CRI for docker is implemented and well maintained, and become to beta.
- E2E test framework has been updated with fully support of out-of-tree CRI container runtime.

### Dockershim deprecation plan

Step 1: Stabilize in-tree dockershim and decouple dockershim from kubelet (but still in-tree).

Target releases: 1.15, 1,16, 1.17

Actions:

- Mark in-tree dockershim as "maintenance mode":
  - CRI generic changes/features can continue on dockershim.
  - WIP efforts on dockershim can continue and go to complete.
  - dockershim/docker specific changes/features should be rejected.
- Deprecate the legacy features of dockershim in kubelet by providing a specific timeline. Currently, kubelet still has:
  - vendored dockershim 
  - flags that are used to configure dockershim.
  - support to get container logs when docker uses journald as the driver.
  - logic of moving docker processes to a given cgroup
  - TBD anything else?
- Package in-tree dockershim is separated from kubelet and provide a "option" to enable/disable it. And the original in-tree dockershim will be remained there currently and depreciated gradually.

- Ensure e2e/Node e2e test framework is CRI generic and test cases are independent of container runtime.

Step 2: Work out a out-of-tree CRI for docker

Target releases: 1.18

Actions:

- Design & implement a out-of-tree CRI for docker, it can be "copied" from dockershim as beginning.
- Re-direct dockershim related features/changes to this out-of-tree CRI for docker.


Step 3: Completely deprecate in-tree dockershim from kubelet.

Target releases: TBD, we probably need to continue keeping in-tree dockershim for 3 releases as grace period.

Actions:

- Refactoring e2e/Node e2e test framework to include CRI for docker installation (or use other CRI container runtime).
  - Ensure cluster/node e2e are 100% CRI focused.
  - Ensure test-infra install CRI-docker or Containerd binary in e2e machines. Currently, they install Docker only.
- Document and announce migration guide.
- Delete in-tree dockershim code from kubelet after certain "grace period".


### Test Plan

_To be filled until targeted at a release._

### Graduation Criteria

_To be filled until targeted at a release._

## Implementation History

- 2019-02-28: Initial KEP sent out for discussion & reviewing.
