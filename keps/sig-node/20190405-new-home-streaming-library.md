---
kep-number: TBD (37?)
title:
authors:
  - "@derekwaynecarr"
  - "@sjenning"
  - "@PiotrProkop"
owning-sig: sig-node
participating-sigs:
  - sig-architecture
reviewers:
  - "@dims"
approvers:
  - "@derekwaynecarr"
editor: "@alok87"
creation-date: 2019-04-05
last-updated: 2019-04-05
status: drafting
see-also:
replaces:
superseded-by:
---

# Streaming library moving out of kubernetes repository

Table of Contents
=================

   * [Streaming library moving out of kubernetes repository](#streaming-library-moving-out-of-kubernetes-repository)
      * [Summary](#summary)
      * [Motivation](#motivation)
         * [Goals](#goals)
         * [Non-Goals](#non-goals)
      * [Proposal](#proposal)
         * [Why do we need to move out streaming library?](#why-do-we-need-to-move-out-streaming-library)
         * [What are the other dependencies that needs to be isolated if streaming is moved out?](#what-are-the-other-dependencies-that-needs-to-be-isolated-if-streaming-is-moved-out)
         * [Where do we move out streaming library?](#where-do-we-move-out-streaming-library)
            * [staging/cri-api/pkg/streaming](#stagingcri-apipkgstreaming)
            * [staging/cri-helpers](#stagingcri-helpers)
            * [staging/cri-streaming](#stagingcri-streaming)

Created by [gh-md-toc](https://github.com/ekalinin/github-markdown-toc)

## Summary

A proposal for moving out [streaming library](https://github.com/kubernetes/kubernetes/tree/master/pkg/kubelet/server/streaming) from kubernetes repository. This proposal aims to discuss the package structure and location where the movement should be done for it.

## Motivation

[Issue link](https://github.com/kubernetes/kubernetes/issues/75828)

Streaming library is used by many CRI implementations. [cri-api](https://github.com/kubernetes/cri-api) was moved to a [new staging repository](https://github.com/kubernetes/kubernetes/pull/75531). This was done so that CRI developers do not need to vendor kubernetes/kubernetes. On doing the above task it was found that [streaming library](https://github.com/kubernetes/kubernetes/tree/master/pkg/kubelet/server/streaming) is also one of the candidates that should be moved out. This should be done since a lot CRI developers are the prime consumers of this library and they still need to vendor `kubernetes/kubernetes` just for this library.

### Goals

Moving streaming library out of kubernetes must:
* reduce vendor dependencies for [cri-api](https://github.com/kubernetes/cri-api) on `k8s.io/kubernetes`
* should not increase the dependency tree for [cri-api](https://github.com/kubernetes/cri-api) or any other component.
* make it easier for the community to make changes in this library.
* find the right place for the streaming library outside the kubernetes repository.

### Non-Goals

The proposal does not aim to discuss movement of any other component except the streaming library and its dependencies.

## Proposal

### Why do we need to move out streaming library?

Streaming library is a library and is only consumer is [cri-api](https://github.com/kubernetes/cri-api) as of now. Since [cri-api](https://github.com/kubernetes/cri-api) has moved out of kubernetes repository, moving streaming library out of kubernetes repository is the right thing to do. As vendoring kubernetes is hard and vendoring kubernetes only for this library should be avoided.

### What are the other dependencies that needs to be isolated if streaming is moved out?

On moving streaming library out of kubernetes, below mentioned dependencies needs to be moved out of kubernetes repository

* `k8s.io/kubernetes/pkg/kubelet/server/portforward`
* `k8s.io/kubernetes/pkg/kubelet/server/remotecommand`
* `k8s.io/kubernetes/pkg/apis/core` would be replaced by `k8s.io/api/blob/master/core/v1` in the streaming library files.

### Where do we move out streaming library?

Based on discussions with community we have following homes to move the streaming library with its dependencies:
- [issue](https://github.com/kubernetes/kubernetes/issues/75828)
- [slack-discussion](https://kubernetes.slack.com/archives/C0BP8PW9G/p1554395402082400)
- [slack-discussion](https://kubernetes.slack.com/archives/C0BP8PW9G/p1553777162006400)

1. Inside the [cri-api](https://github.com/kubernetes/cri-api) staging repository: `staging/cri-api/pkg/streaming`
2. Inside a new staging repository [cri-helpers](httpss://github.com/kubernetes/cri-helpers): `staging/cri-helpers`
3. Inside a new staging repository [cri-streaming](httpss://github.com/kubernetes/cri-helpers): `staging/cri-streaming`

#### `staging/cri-api/pkg/streaming`

* **Pros**
  * Since [cri-api](https://github.com/kubernetes/cri-api) is the only consumer of streaming library at present moving this to the same repository as cri-api makes sense.
  * Prevents us from maintaining one more staging repository.
* **Cons**
  * Adding streaming library to the [cri-api](https://github.com/kubernetes/cri-api) greatly increases the dependency tree due to the following staging dependencies streaming library has. [More info](https://github.com/kubernetes/kubernetes/issues/75828#issuecomment-479951324)
  ```
  k8s.io/api
  k8s.io/apimachinery
  k8s.io/apiserver
  k8s.io/client-go
  ```
  * A developer needs to download all these large transitive dependencies for just importing some cri types. But if the cri-api already has this dependencies then it is not a con.

[Implementation](https://github.com/kubernetes/kubernetes/pull/76090)

#### `staging/cri-helpers`

* **Pros**
  * Since we already have a cli-runtime repo which has a set of helpers for implementers, we could use that same pattern. [More Info](https://github.com/kubernetes/kubernetes/issues/75828#issuecomment-477594093)
  * `/streaming`, `/remotecommand`, and `/portforward` are libraries for streaming connections between kubelet and a kubelet container runtimes implementing the cri so moving all of them under a common helper repostiory seems right. [More Info](https://github.com/kubernetes/kubernetes/issues/75828#issuecomment-477609283)
  * A developer does not need to download and depend on streaming library dependencies for importing something in the cri-api (but if the cri-api already have those dependencies downloaded, this point should not be under "pros")

* **Cons**
  * One more staging repository to vendor and maintain

[Implementation](to be made)

Note: Please note the name here is still open for discussion. Few more options are:
```
cri-server
cri-runtime
```

#### `staging/cri-streaming`

* **Pros**
  * Since `streaming`, `portforward` and `remotecommand` are all related to streaming connections, it also makes sense to make a new staging repository specifically for streaming.
  * A developer does not need to download and depend on streaming library dependencies for importing something in the cri-api (but if the cri-api already have those dependencies downloaded, this point should not be under "pros")

* **Cons**
  * One more staging repository to vendor and maintain.
  * If later more cri helper libraries needs to be moved out of kubernetes then a new repository for it also needs to be made as it will not fit into the "streaming" staging repository

[Implementation](to be made)
