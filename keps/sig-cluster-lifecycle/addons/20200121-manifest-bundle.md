---
title: Manifest Bundle
authors: ["@ecordell"]
owning-sig: sig-cluster-lifecycle
participating-sigs: ["sig-cluster-lifecycle", "sig-api-machinery"]
reviewers: ["TBD"]
approvers: ["TBD"]
editor: TBD
creation-date: 2020-01-21
last-updated: 2020-02-06
status: provisional
see-also: ["/keps/sig-cluster-lifecycle/addons/0035-20190128-addons-via-operators.md"]
---

# Manifest Bundle

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
- [Goals](#goals)
- [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [User Stories](#user-stories)
  - [Build, Push, Pull Manifest Bundle](#build-push-pull-manifest-bundle)
- [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [Examples](#examples)
    - [Operator Bundles](#operator-bundles)
    - [Plain manifests](#plain-manifests)
    - [kops](#kops)
    - [kubeadm](#kubeadm)
  - [Expose a Manifest bundle](#expose-a-manifest-bundle)
    - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints-1)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

This enhancement proposes standards and conventions for storing kubernetes manifests and metadata associated with an operator as container images in OCI-compliant container registries, and to associate metadata-only images with standard, runnable images.

## Motivation

There is no standard way to associate and transmit kube and kube-like manifests and metadata between clusters, or to associate a set of manifests with one or more runnable container images.

Existing non-standard methods include:

* git repositories
* operator-registry directory “bundles”
* helm charts
* appregistry

We would like to be able to talk about a set of metadata and manifests, outside the context of a cluster, as representing a particular application or service (in this case, an operator).

By standardizing on a container format for this data, we get many other features for free, such as: identity, distribution, replication, deduplication, signing, and cluster ingress.

## Goals

* Define a convention for storing manifests and metadata with a container image
* Build and push metadata using standard container tooling (docker cli, podman, etc) to common container image registries (Artifactory, docker hub, quay, google's registry, amazon's registry)
* No union filesystem should be required to consume metadata
* Have a simple mechanism to apply a bundle to a kubernetes cluster

## Non-Goals

* Require OCI registries that support any non-standard media types (i.e. anything other than docker and layer image oci)
* Build on-cluster tooling to interact with bundles

## Proposal

We would like to make a distinction between:

* **Manifests** * data which can be applied to a cluster, either with or without pre-processing. Examples: json or yaml manifests, helm charts, kustomize bundles.
* **Metadata** * information which will not be directly applied to a cluster, but which will otherwise affect the processing or application of a bundle. 

## User Stories

### Build, Push, Pull Manifest Bundle

As a manifest author, I would like to store manifests and metadata in a format compatible with container registries.

Constraints:

* An operator bundle (including both manifests and metadata) should be identifiable using a single versioned identifier.

## Implementation Details/Notes/Constraints

The initial implementation target will be Docker v2-2 manifests, manifest-lists, and client support, for maximum compatibility with existing tooling. OCI is a future target, but avoided for now due to lack of tooling support. 

Labels are used to identify the contents. The approach is similar to OCI mediatypes, but is supported by existing tooling.

The following label convention is used to annotate the bundle image:

* `<namespace>.mediatype.<version>=<identifier>` is used to identify the top-level format of the bundle. For example, this may indicate that the bundle contains a kustomization file and kustomize manifests.
* `<namespace>.bundle.manifests.<version>=<path>` reflects the path in the image to a directory that contains manifests. 
* `<namespace>.bundle.metadata.<version>=<path>` reflects the path in the image to a directory that contains metadata.
* Any additional `<namespace>.bundle.<identifier>=<value>` may be used to indicate additional properties of the manifest bundle. It may be useful to denormalize information that would otherwise be stored in the metadata directory, so that tooling can read it without unpacking a full image.
These labels should also be replicated in a well-known location within the image, metadata/annotations.yaml:

```yaml
annotations:
  <namespace>.mediatype.<version>: <identifier>
  <namespace>.bundle.manifests.<version>: <path>
  <namespace>.bundle.metadata.<version>: <path>
  <namespace>.bundle.<identifier>: <value>
```

This is important so that on-cluster tools can have access to the labels. There is currently no way for kubernetes to expose image labels to a consumer. In the case of a conflict, the labels in the annotations file are preferred.

### Examples

#### Operator Bundles

The operator framework is [piloting the use of this format](https://github.com/operator-framework/operator-registry/blob/master/docs/design/operator-bundle.md) for transmitting operator bundles. The annotations used in this example are:

```yaml
annotations:
  operators.operatorframework.io.bundle.mediatype.v1: "registry+v1"
  operators.operatorframework.io.bundle.manifests.v1: "manifests/"
  operators.operatorframework.io.bundle.metadata.v1: "metadata/"
  operators.operatorframework.io.bundle.package.v1: "test-operator"
  operators.operatorframework.io.bundle.channels.v1: "beta,stable"
  operators.operatorframework.io.bundle.channel.default.v1: "stable"
```

#### Plain manifests

Kubernetes manifests may be packaged minimally with:

```yaml
annotations:
  k8s.io.bundle.mediatype.v1: "manifests"
  k8s.io.bundle.manifests.v1: "manifests/"
  k8s.io.bundle.metadata.v1: "metadata/"
```

and kubectl could easily be extended to support applying a bundle formatted in this way.

#### kops

There is ongoing work in `kops` to improve support for addons. One current prototype uses `kops create cluster --add=file.yaml --add=file2.yaml`, where each `--add` flag indicates some manifest required for an addon. Metadata for kops may look something like:

```yaml
annotations:
  kops.k8s.io.bundle.mediatype.v1: "kops+v1"
  operators.operatorframework.io.bundle.manifests.v1: "manifests/"
  operators.operatorframework.io.bundle.metadata.v1: "metadata/"
```

#### kubeadm

The [proposed kep](https://github.com/kubernetes/enhancements/pull/1308) for addon support for kubeadm mentions kustomize as a target bundle format and OCI artifacts as a potential solution. The current KEP would offer similar benefits to the OCI approach but without some of the current limitations.

### Expose a Manifest bundle

As a kubernetes user, I would like to use kubelet to download the manifest bundle and expose it to me for consumption.

Clusters are often configured with specific rules and credentials for pulling images into a cluster. Relying on kubelet to pull manifest bundles ensures that configuration comes through the same channel as runnable images.

#### Implementation Details/Notes/Constraints

A tool should be able to pull down the manifest bundle and expose it to a cluster. Examples of such extractors are:

* Extract and store manifests as entries in a configmap or secret for further processing on-cluster
* Read and apply the manifests, according to rules configured by the metadata.

Tools that extract bundles on a cluster work by:

* Running a Pod with an initContainer containing the extraction tool and a container referencing the bundle.
* On start, the tool from the init container is copied into the running environment

The tool is run in a pod which is now populated with manifests. It will read metadata/annotations.json to know how to process the other files in the bundle.

Such a [tool](https://github.com/operator-framework/operator-registry/blob/master/cmd/opm/alpha/bundle/extract.go) has been written for the operator-framework pilot and can easily be generalized.

## Alternatives

This proposal is aligned with the use cases and requirements of [OCI Artifacts](https://github.com/opencontainers/artifacts). This proposal could be seen as a way to implement OCI artifacts with standard container images.

There are several reasons that artifacts are not a good solution _at the moment_:

* Not all registries support OCI yet.
* Of registries that do support the OCI spec, not all support OCI artifacts in particular.
* Of registries that do support OCI artifacts, not all allow arbitrary media types, or provide a mechanism to register new media types. There is currently no standard set of artifact media types for registries to support (something of a chicken-and-egg problem with mediatypes).
* There are few tools that can build OCI artifacts.
* There is currently no way for kubelet to pull an OCI artifact into a cluster. This limits the ways in which manifest bundles can be used to offcluster. (Other, non-kubelet tools within a cluster could pull them, but such pulls would lack the auditing and cluster-wide configuration that comes from relying on kubelet for image ingress).
* There is currently no way to read labels (annotations) from an image on a cluster.

None of these problems are insurmountable, and it would be desirable to move to OCI artifacts when these roadblocks are removed. There should be a straightforward migration path to OCI artifacts when that happens.
