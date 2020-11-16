---
title: Graduate Server-side Get and Partial Objects to GA
authors:
  - "@smarterclayton"
owning-sig: sig-api-machinery
participating-groups:
  - sig-cli
reviewers:
  - "@lavalamp"
  - "@soltysh"
  - "@liggitt"
approvers:
  - "@liggitt"
  - "@pwittrock"
editor: TBD
creation-date: 2019-03-22
last-updated: 2019-03-22
status: implementable
see-also:
  - "https://github.com/kubernetes/community/blob/master/contributors/design-proposals/api-machinery/server-get.md"
replaces:
superseded-by:
---

# Graduate Server-side Get and Partial Objects to GA

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [1.15](#115)
  - [1.16](#116)
  - [1.19](#119)
  - [Implementation Details](#implementation-details)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

Server-side columnar formatting and partial object metadata has been in beta since Kube 1.10 and as of 1.15 is consistently implemented and in wide use as part of `kubectl` and other web interfaces. This document outline required steps to graduate it to GA. 

## Motivation

The user of server-side printing with CRDs and Aggregated APIs is common and is a key part of providing equivalence between built-in and extension APIs for administrator usability. We have not needed to update the schema since 1.10, proving that it is ready to ship. Promoting it to GA in 1.15 will allow us to declare the kubectl portion feature complete and remove the legacy printers, and also update controllers that would benefit from use of PartialObjectMetadata (garbage collector, namespace controller, and quota counter) without fear of deprecation. PartialObjectMetadata allows these controllers to perform protobuf list actions efficiently when retrieving only the core object metadata.

### Goals

Server-side printing has no outstanding feature requests now that full [WATCH support has been implemented in 1.15](https://github.com/kubernetes/kubernetes/pull/71548). It is ready to move to GA by promoting the resources.

PartialObjectMetadata exposes our full ObjectMeta interface and no API changes are anticipated. However, to prove their value one of the dynamic controllers should be ported in 1.15 to use PartialObjectMetadata instead of Unstructured objects to demonstrate the gains in performance. If successful PartialObjectMetadata would also be candidate for GA in 1.15.

### Non-Goals

* Changes to the `Table` object not directly concerned with supporting `kubectl` or other consumers
* Changes to `PartialObjectMetadata` that are not related to backend implementation.

## Proposal

### 1.15

* Copy `Table`, `PartialObjectMetadata` and `PartialObjectMetadataList` to `meta/v1` and expose the transformations in the API server.
  * Update the serialization of `PartialObjectMetadataList` to use protobuf id `1` for `ListMeta` (it was added late in v1beta1)

### 1.16

* Update controllers to use `PartialObjectMetadata` `v1`.
  * In the garbage collector, we will remove the need to call `Update` and use a partial object metadata client/informer
  * In the namespace controller, we will use a partial object metadata informer
  * In the quota counting code, we will use a partial object metadata informer
* Announce deprecation of `v1beta1` objects and removal in 1.19 
* `kubectl` should switch to using `meta.k8s.io/v1` `Table` (supporting 1.15+ clusters)

### 1.19

* Remove `meta.k8s.io/v1beta1`

### Implementation Details

A new dynamic client variant capable of supporting read and write operations on PartialObjectMetadata
should be created that hides whether the server supports PartialObjectMetadata. 

Currently `v1beta1.Table` does not support Protobuf and the generators do not trivially support the
serialization of the cells. We need to decide on a serialization format for the Protobuf cells and
ensure generators can be made to support it. This work does not need to block `v1` for Protobuf
because the clients that access `Tables` are almost exclusively JSON clients (web consoles and CLIs).
`PartialObjectMetadata*` will have a full protobuf implementation.

### Risks and Mitigations

The primary risk is that we identify a `v1beta1.Table` schema change post freeze. Wider use might
gather that feedback, but the schema is deliberately left simple to allow us to grow in the future.

## Graduation Criteria

The following code changes must be made to take `Table` GA

* Move API objects to `v1` and support conversion internally
* Update REST infra to support transforming objects at that version

The following code changes must be made to take `PartialObjectMetadata` GA

* Move API objects to `v1` and support conversion internally
* Update REST infra to support transforming objects at that version

The following code changes should be made before `PartialObjectMetadata` is GA to get feedback

* Update all of (GC, Namespace, Quota counting) to use a new PartialObjectMetadata specific typed client using protobuf.

### Version Skew Strategy

We will support N-1 for both `kubectl` and participating controllers by introducing GA first and then deprecating and removing v1beta1 in the subsequent release. There are no serialized forms of these objects and so on-disk format is not a concern.

## Implementation History

* First version of this proposal merged.
* Server-side GET objects moved to v1 in 1.15