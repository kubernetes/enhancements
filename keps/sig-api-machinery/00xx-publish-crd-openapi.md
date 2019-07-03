---
title: Publish CRD OpenAPI
authors:
  - "@roycaihw"
owning-sig: sig-api-machinery
reviewers:
  - "@apelisse"
  - "@liggitt"
  - "@mbohlool"
  - "@sttts"
approvers:
  - "@liggitt"
  - "@sttts"
editor: TBD
creation-date: 2019-02-07
last-updated: 2019-02-13
status: implementable
see-also:
  - "[Validation for CustomResources design doc](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/api-machinery/customresources-validation.md)"
---

# Publish CRD OpenAPI

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Build Schema/Definition](#build-schemadefinition)
  - [Build Spec](#build-spec)
  - [Aggregate and Publish Spec from apiextensions-apiserver](#aggregate-and-publish-spec-from-apiextensions-apiserver)
  - [Aggregate and Publish Spec from kube-aggregator](#aggregate-and-publish-spec-from-kube-aggregator)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

In CustomResourceDefinition (CRD) we allow CRD author to define OpenAPI v3 schema, to
enable server-side [validation for CustomResources (CR)](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/api-machinery/customresources-validation.md).
The validation schema format is compatible for creating OpenAPI documentation for CRs,
which can be used by clients like kubectl to perform client-side validation
(e.g. `kubectl create` and `kubectl apply`),
schema explanation (`kubectl explain`), and client generation.
This KEP proposes using the OpenAPI v3 schema to create and publish OpenAPI
documentation for CRs.

## Motivation

Publishing CRD OpenAPI enables client-side validation, schema explanation and
client generation for CRs. It covers the gap between CR and native Kubernetes
APIs, which already support OpenAPI documentation.

Publishing CRD OpenAPI is also noted as potential followup in [Validation for CustomResources Implementation Plan](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/api-machinery/customresources-validation.md#implementation-plan).

### Goals

* For every CRD that we serve, publish Paths (operations that we support on
  resource and subresources) and Definitions (for both CR object and CR list
  object) in OpenAPI documentation to fully demonstrate the existence of the
  API.
* For CRDs with schema defined, the CR object Definition should
  include both CRD schema and native Kubernetes ObjectMeta and
  TypeMeta properties.
* For CRDs without schema, the CR object definition will be as
  complete as possible while still maintaining compatibility with the openapi
  spec and with supported kubernetes components

### Non-Goals

* Finalize design on how to aggregate and publish CRD OpenAPI from kube-aggregator

## Proposal

Ref: https://github.com/kubernetes/kubernetes/pull/71192

### Build Schema/Definition

We drop the features that OpenAPI v2 don't support from CRD OpenAPI v3 schema
(listed in this [spread-sheet](https://docs.google.com/spreadsheets/d/1Mkm9L7CXGvRorV0Cr4Vwfu0DH7XRi24YHPiDK1NZWo4/edit?usp=sharing)
by @nikhita and @sttts: oneOf, anyOf and not) (proposed [here](https://github.com/kubernetes/kubernetes/issues/49879#issuecomment-320031200)).

We add documentation in CRD API pointing it out to CRD authors that if they want
a complete and valid OpenAPI spec being generated for their Resource, they should
not use v3 features like `oneOf` (proposed [here](https://github.com/kubernetes/kubernetes/issues/49879#issuecomment-321774254)).

For CRD with schema, we append Kubernetes TypeMeta and ObjectMeta to the schema.

### Build Spec

We template a WebService object for each CRD like what we do for [built-in APIs](https://github.com/kubernetes/kubernetes/blob/8b98e802eddb9f478ff7d991a2f72f60c165388a/staging/src/k8s.io/apiserver/pkg/endpoints/installer.go#L565-L845),
feed the WebService to kube-openapi builder to build OpenAPI spec.

### Aggregate and Publish Spec from apiextensions-apiserver

We add a spec aggregator to apiextensions-apiserver like [kube-aggregator
does](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/kube-aggregator/pkg/controllers/openapi/aggregator/aggregator.go).
The aggregator keeps a {CR GroupVersionKind (GVK) -> OpenAPI spec} mapping,
aggregates specs, and serves the spec through "/openapi/v2" handler from
apiextensions-apiserver.

All the code mentioned above (CRD OpenAPI builder, aggregator and controller)
live within apiextensions-apiserver staging repo and get called within the
same repo.

### Aggregate and Publish Spec from kube-aggregator

We add apiextensions-apiserver into the resync queue in kube-aggregator openapi
controller with a resync period of 1 second. Previously apiextensions-apiserver
(delegation target) was treated as static and never get resynced.

Downloading spec and etag from apiextensions-apiserver is in-process function
call. We skip aggregation when etag doesn't change. So we are not adding much
load.

Alternative solutions discussed in [this doc](https://docs.google.com/document/d/13lBj8Stdwku8BgL0fbT__4Iw97NRh77loJ_MoZuCGwQ/edit#). Finalizing the design is a non-goal for 1.14.

### Risks and Mitigations

Performance: OpenAPI aggregation has performed poorly in the past. Introducing
aggregation that occurs on a higher frequency could cause performance issues in
the apiserver. To mitigate this, improvements in OpenAPI merging are being made
(both in terms of memory and CPU cost), and benchmarking will be done prior to
merge to ensure we do not regress API server performance.

Compatibility: some Kubernetes components (notably kubectl) perform client-side
validation based on published openapi schemas. To ensure that publishing schemas
for additional custom resources does not regress existing function, we will test
creation of custom resources with and without validation with current and prior versions of kubectl.

We will add a feature gate CustomResourcePublishOpenAPI, which can be turned on to start CRD
openapi creation & aggregation & publishing. The feature gate will be Alpha (defaulted to
False) in 1.14, Beta (defaulted to True) in 1.15 and Stable in 1.16.

## Graduation Criteria

In 1.14, publishing CRD OpenAPI should be e2e-tested to ensure:

1. for CRD with validation schema

* client-side validation (`kubectl create` and `kubectl apply`) works to:
  * allow requests with known and required properties
  * reject requests with unknown properties when disallowed by the schema
  * reject requests without required properties

* `kubectl explain` works to:
  * explain CR properties
  * explain CR properties recursively
  * return error when explain is called on property that doesn't exist

2. for CRD without validation schema

* client-side validation (`kubectl create` and `kubectl apply`) works to:
  * allow requests with any unknown properties

* `kubectl explain` works to:
  * returns an understandable error or message when asked to explain a CR with no available schema

3. for any CRD with/without validation schema, we do not regress existing
  behavior for prior version of kubectl:

* `kubectl create` client-side validation works to:
  * allow requests with any unknown properties

* `kubectl explain` works to:
  * returns an understandable error or message when asked to explain a CR with no available schema

4. for multiple CRDs
  * CRs in different groups (two CRDs) show up in OpenAPI documentation
  * CRs in the same group but different version (one multiversion CRD or two
    CRDs) show up in OpenAPI
    documentation
  * CRs in the same group version but different kind (two CRDs) show up in OpenAPI
    documentation

5. for publishing latency and cpu load
  * does not consume significant delay to update OpenAPI Paths and Definitions
    after CRD gets created/updated/deleted
  * does not measurably increase server load at steady state

6. for `kubectl apply`
  * verify no regression using apply with CRs with and without schemas, on the current and previous version of kubectl

## Implementation History

