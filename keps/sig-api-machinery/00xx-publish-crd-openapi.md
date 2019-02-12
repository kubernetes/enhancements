---
kep-number: 0
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
last-updated: 2019-02-11
status: provisional
see-also:
  - [Validation for CustomResources design doc](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/api-machinery/customresources-validation.md)
---

# Publish CRD OpenAPI

## Table of Contents

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
* [Proposal](#proposal)
* [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)

## Summary

In CustomResourceDefinition (CRD) we allow CRD author to define OpenAPI v3 schema, to
enable server-side [validation for CustomResources (CR)](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/api-machinery/customresources-validation.md).
The validation schema format is compatible for creating OpenAPI documentation for CRs,
which can be used by clients like kubectl to perform client-side validation
(e.g. `kubectl create` and `kubectl apply`),
schema explanation (`kubectl explain`), and client generation.
This KEP purposes using the OpenAPI v3 schema to create and publish OpenAPI
documentation for CRs.

## Motivation

Publishing CRD OpenAPI enables client-side validation, schema explanation and
client generation for CRs. It covers the gap between CRD and native Kubernetes
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
* For CRDs without schema, the CRD object definition will be as
  complete as possible while still maintaining compatibility with the openapi
  spec and with supported kubernetes components

## Proposal

Ref: https://github.com/kubernetes/kubernetes/pull/71192

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

We will add a feature gate CustomResourceOpenAPI, which can be turned off to stop
openapi creation & aggregation & publishing from happening. The feature gate
will be defaulted to be True. The feature gate will be Beta in 1.14 and Stable
in 1.15.

## Graduation Criteria

In 1.14, publishing CRD OpenAPI should be e2e-tested to ensure:

1. for CRD with validation schema

* client-side validation (`kubectl create` and `kubectl apply`) works to:
  * allow requests with known and required properites
  * reject requests with unknown properties when disallowed by the schema
  * reject requests without required properties

* `kubectl explain` works to:
  * explain CRD properties
  * explain CRD properties recursively
  * return error when explain is called on property that doesn't exist

2. for CRD without validation schema

* client-side validation (`kubectl create` and `kubectl apply`) works to:
  * allow requests with any unknown properties

* `kubectl explain` works to:
  * returns an understandable error or message when asked to explain a CRD with no available schema

3. for any CRD with/without validation schema, we do not regress existing
  behavior for prior version of kubectl:

* `kubectl create` client-side validation works to:
  * allow requests with any unknown properties

* `kubectl explain` works to:
  * returns an understandable error or message when asked to explain a CRD with no available schema

4. for multiple CRDs
  * CRDs in different groups (two CRDs) show up in OpenAPI documentation
  * CRDs in the same group but different version (one multiversion CRD or two
    CRDs) show up in OpenAPI
    documentation
  * CRDs in the same group version but different kind (two CRDs) show up in OpenAPI
    documentation

5. for publishing latency and cpu load
  * does not consume significant delay to update OpenAPI Paths and Definitions
    after CRD gets created/updated/deleted
  * does not measurably increase server load at steady state

6. for `kubectl apply`
  * verify no regression using apply with CRDs with and without schemas, on the current and previous version of kubectl

## Implementation History

