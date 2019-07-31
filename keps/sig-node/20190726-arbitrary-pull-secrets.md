---
title: Arbitrary image pull secrets
authors:
  - "@sashayakovtseva"
owning-sig: sig-node
creation-date: 2019-07-26
status: provisional
---

# Arbitrary image pull secrets

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Implementation Details](#implementation-details)
- [Alternatives ](#alternatives)
<!-- /toc -->

## Summary

Currently in order to pull an image from a private registry one can use `imagePullSecrets` (as described
[here](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/)). 
However, it is tightly integrated with Docker and there is no other way to support different on-prem
registries or different types of authorization (e.g. private tokens).

This proposal suggests changes that will fully enable arbitrary private image registry support.

## Motivation

Kubernetes introduced CRI making it possible to integrate any container runtime with it.
CRI contains `ImageService` that is responsible for any image-related operations performed by kubelet. 

Whenever a new container is created and base image is not present on node, `PullImage` method of `ImageService`
is called. `PullImageRequest` contains `AuthConfig`, so no changes should be make in CRI to enable support
for private image pulls.   

It is common to have private container images or on-prem image registries when working with Kubernetes. To use
those, one should either perform node configuration or leverage `imagePullSecrets` for pod that will be passed
to the CRI implementation in `AuthConfig`.
 
However, even though Kubernetes has CRI, `imagePullSecrets` are respected only when the secret type is
`kubernetes.io/dockerconfigjson` or `kubernetes.io/dockercfg`. This is not aligned with CRI concept since CRI should
enable support for arbitrary container image registries (via `ImageService`) and arbitrary container runtimes (via
`RuntimeService`). 

### Goals

- Introduce a new secret type, e.g. `kubernetes.io/regcredentials`.
It should contain data that can be mapped to the CRI's `AuthConfig` subset of fields directly.
  
- Do not interpret passed credentials as Docker’s when a secret of type `kubernetes.io/regcredentials`
is used in `imagePullSecrets`. Pass it directly to the CRI implementation and let it decide how to handle them instead.

## Proposal

### User Stories 

#### Story 1
As a cluster user, I want to create a secret with credentials to an image registry other than Docker.

#### Story 2
As a cluster user, I want to pull my private image from a non-Docker registry with my credentials.

#### Story 3
As a cluster user, I want to pull my images from my on-prem image registry.

### Implementation Details 

```
apiVersion: v1
kind: Secret
metadata:
  ...
  name: regcred
  ...
data:
  serverAddress: my.custom-registry.com
  username: john-smith
  password: <base64-encoded-password>
  identityToken: <identity-token-here>
  registryToken: <registry-token-here>
type: kubernetes.io/regcredentials
```

## Alternatives 

A cluster admin can configure each node or set up some kind of a proxy that will contain registry credentials.
However, that is not convenient and not scalable. Also images from a various private/on-prem registries can be
requested, and credentials themselves can be updated.
