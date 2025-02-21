<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-4358: Custom Resource Field Selectors

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Background: Field selection as it exists before this enhancement](#background-field-selection-as-it-exists-before-this-enhancement)
  - [Plan](#plan)
    - [API](#api)
      - [Validation rules](#validation-rules)
    - [OpenAPI Discovery](#openapi-discovery)
    - [Implementation](#implementation)
  - [User Stories](#user-stories)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Future work](#future-work)
    - [Declarative field selector definitions on built-in types](#declarative-field-selector-definitions-on-built-in-types)
    - [Index selected fields](#index-selected-fields)
    - [CEL as an accelerated predicate language](#cel-as-an-accelerated-predicate-language)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Increases watch cache resource utilization](#increases-watch-cache-resource-utilization)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
- [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [OpenAPI Discovery](#openapi-discovery-1)
    - [Declare selectable fields to the fieldSelector path parameter of OpenAPI](#declare-selectable-fields-to-the-fieldselector-path-parameter-of-openapi)
    - [Declare selectable fields by annotating schema fields as being selectable, e.g.:](#declare-selectable-fields-by-annotating-schema-fields-as-being-selectable-eg)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes adding [field selector](https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/)
support to custom resources.

## Motivation

All Kubernetes APIs support [field
selection](https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/)
for metadata.name, metadata.namespace.

Additionally, built-in APIs support field selection on specific fields such as
status.phase for Pods, but CustomResourceDefinitions lack equivalent
functionality.

Without this enhancement, an extension author that wants to be be able to filter
resources might choose to:

  - Organize data into a label even though it should be a spec or status field.
  - Double writing data both to a label and a spec or status field.

Both of these should be avoided. We should enable extension authors structure
their APIs according to best practices, not based on data access patterns.

### Goals

- Add an allow list of selectable fields to CustomResourceDefinitions.
- Performance and resource consumption characteristics are roughtly equivalent
  to label selectors.

### Non-Goals

- Arbitrary field selection.
   - This proposal is to support field selection for a small allow list of
     fields. Expanding support to include all fields is complicated by:
     - Fields contained in, or nested under, lists or maps.
     - The costs to apiserver of evaluating filters on objects or maintaining
       indices.
     - Lack of direct support for filtering in etcd.
- Support for conversions, casts or other transforms of field values beyond
  the automatic cast to string required for field selection.
- More sophisticated field selection logic. Possible approaches include: CEL
  based field selection, enhancements to field selection such as set based
  operators like provided for labels ("in", "not in"), OR clauses, pattern
  matching... See the "CEL as an accelerated predicate language" section "Future
  Work" for more details on what makes this challenging.


## Proposal

Add `selectableFields` to the versions of CustomResourceDefinition:

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: selector.stable.example.com
spec:
  group: stable.example.com
  versions:
    - name: v1
      selectableFields:
      - jsonPath: .spec.color 
      - jsonPath: .spec.size 
      additionalPrinterColumns:
      - jsonPath: .spec.color
        name: Color
        type: string
      - jsonPath: .spec.size
        name: Size
        type: string
...
```

(Alternatively, `selectableFields` could contain `fieldPath: spec.color` to
align more closely with `validationRules.fieldPath`. This will be discussed API
review).

Clients may then use the field selector to filter resources:

```sh
$ kubectl get selector.stable.example.com
NAME       COLOR  SIZE
example1   blue   S
example2   blue   M
example3   green  M

$ kubectl get selector.stable.example.com --field-selector spec.color=blue
NAME       COLOR  SIZE
example1   blue   S
example2   blue   M
$ kubectl get selector.stable.example.com --field-selector spec.color=green,spec.size=M

NAME       COLOR  SIZE
example2   blue   M
```


### Background: Field selection as it exists before this enhancement

Field selection is string equality based. [Set-based operators](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#set-based-requirement) are
not supported (in, notin, exists). Field selectors may be ["chained"](https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/#chained-selectors) (ANDed together).

All resources have two field selectors available for metadata:

| Kind                                      | Field                              | Field type       | field selector                 | Conversions/Notes                                             |
|-------------------------------------------|------------------------------------|------------------|--------------------------------|---------------------------------------------------------------|
| *                                         | metadata.name                      | string           | metadata.name                  |                                                               |
| *                                         | metadata.namespace                 | string           | metadata.namespace             | Absent on cluster scoped resources                            |

Some resources have additional field selectors:

| Kind                                      | Field                              | Field type       | field selector                 | Conversions/Notes                                             |
|-------------------------------------------|------------------------------------|------------------|--------------------------------|---------------------------------------------------------------|
| ReplicaSet / RC                           | status.replicas                    | int              | status.replicas                | itoa(int)                                                     |
| Job                                       | *status.succeeded*                 | int              | *status.successful*            | itoa(int), *field selector/name mismatch*                     |
| CertificateSigningRequest                 | spec.signerName                    | string           | spec.signerName                | -                                                             |
| Event                                     | involvedObject.kind                | string           | involvedObject.kind            | -                                                             |
| Event                                     | involvedObject.namespace           | string           | involvedObject.namespace       | -                                                             |
| Event                                     | involvedObject.name                | string           | involvedObject.name            | -                                                             |
| Event                                     | involvedObject.uid                 | UID              | involvedObject.uid             | string(UID)                                                   |
| Event                                     | involvedObject.apiVersion          | string           | involvedObject.apiVersion      | -                                                             |
| Event                                     | involvedObject.resourceVersion     | string           | involvedObject.resourceVersion | -                                                             |
| Event                                     | involvedObject.fieldPath           | string           | involvedObject.fieldPath       | -                                                             |
| Event                                     | reason                             | string           | reason                         | -                                                             |
| Event                                     | reportingComponent                 | string           | reportingComponent             | -                                                             |
| Event                                     | source.component                   | string           | source                         | *Set to reportingController if field omitted*, *field selector/name mismatch* |
| Event                                     | type                               | string           | type                           | -                                                             |
| Namespace/PV/PVC                          | metadata.name [^1]                 | string           | name [^1]                      | *field selector/name mismatch*, *metadata.name already selectable* |
| Namespace                                 | phase                              | string           | status.phase                   | -                                                             |
| Node                                      | spec.unschedulable                 | boolean          | spec.unschedulable             | fmt.Sprint(bool)                                              |
| Pod                                       | spec.nodeName                      | string           | spec.nodeName                  | -                                                             |
| Pod                                       | spec.restartPolicy                 | enum             | spec.restartPolicy             | string(enum)                                                  |
| Pod                                       | spec.schedulerName                 | enum             | spec.schedulerName             | string(enum)                                                  |
| Pod                                       | spec.serviceAccountName            | enum             | spec.serviceAccountName        | string(enum)                                                  |
| Pod                                       | *spec.securityContext.hostNetwork* | optional boolean | spec.hostNetwork               | strconv.FormatBool(bool) *Set to "false" if field is omitted* |
| Pod                                       | status.phase                       | string           | phase                          | *field selector/name mismatch*                                |
| Pod                                       | *status.podIP*                     | slice            | *status.podIPs*                | *Set to "" if field is omitted*                               |
| Pod                                       | status.nominatedNodeName           | string           | status.nominatedNodeName       |                                                               |
| Secret                                    | type                               | enum             | type                           | string(enum)                                                  |

[^1]: This field selector is commented with: "This is a bug, but we need to support it for backward compatibility.".

Note that:

- Resources always provide a value for a selectable field. Notice that in the
  above table, *all* field selectors for optional fields provide a fallback if the
  field is omitted.
- Field selectors only exist for strings, enums, integers and booleans.
- Field selectors for name and namespace are provided automatically.
- Field selectors that do not match the path name of the actual field do exist,
  for historical reasons, but the best practice is for field selectors to match
  the exact path name of the field.

Also note that we do not currently publish any field selector information into
OpenAPIv3, and we do not document in APIs or on the Kubernetes website which
fields may be selected. Because of this, finding the field selectors available for built-in types, to construct the above table, required
locating [`GetAttrs/*ToSelectableFields` functions in the Kubernetes source code](https://github.com/kubernetes/kubernetes/blob/master/pkg/registry/core/pod/strategy.go#L340).

### Plan

#### API

```go
type CustomResourceDefinitionVersion struct {
  
  // selectableFields specifies paths to fields that may be used as field selectors.
  // TODO: A max limit will be decided during API review
	// See https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors
	SelectableFields []SelectableField `json:"selectableFields,omitempty" protobuf:"bytes,9,rep,name=selectableFields"`
}

  // SelectableField specifies the JSON path of a field that may be used with field selectors.
type SelectableField struct {
	// jsonPath is a simple JSON path which is evaluated against each custom resource to produce a
  // field selector value.
  // Only JSON paths without the array notation are allowed.
  // Must point to a field of type string, boolean or integer. Types with enum values
  // and strings with formats are allowed.
  // If jsonPath refers to absent field in a resource, the jsonPath evaluates to an empty string.
  // Must not point to metdata fields.
	JSONPath string `json:"jsonPath" protobuf:"bytes,1,opt,name=jsonPath"`
}
```

If a default value is need, the CustomResouceDefiniton author is recommended to
set the default on the field in the OpenAPI schema that the jsonPath refers to.

If the jsonPath refers to an absent field that is absent in a resource and
does not have a default, the selectable field value will be an empty string.

##### Validation rules

- `selectableFields[].jsonPath` must be a "simple path" (similar to
  `additionalPrinterColumns[].jsonPath` except validated properly). I.e.
    property names separated by '.', e.g. `.spec.x.y` (The path may not contain
    map or array indexing)
- The schema type referenced by `selectableFields[].jsonPath` must be one of:
  string, integer, boolean (enums and fields with string formats are allowed).
- There is a limit on the maximum `selectableFields` per CRD version that are allowed.
- `selectableFields` may not refer to metadata fields.
- `selectableFields` may not contain duplicates.


#### OpenAPI Discovery

We will add discovery details in to OpenAPIv3. For example:

```json
"org.example.v1.CustomResource": {
  "type": "object",
  "x-kubernetes-group-version-kind": [ ... ],
  "x-kubernetes-selectable-fields": [
    { "fieldPath": "metadata.name"},
    { "fieldPath": "metadata.namespace"},
    { "fieldPath": "spec.myField"},
  ]
},
```

#### Implementation

Because `generic.StoreOptions` consumes selectors from built-in and CRDs in the
same way, we would extend the `GetAttrs` of CRDs to offer fields for selection.
https://github.com/jpbetz/kubernetes/tree/crd-object-filters contains a working
prototype.

We would leverage the fieldPath validation supported added for CRD Validation
Rules to validate paths.

We would leverage the jsonpath library in the same way as used by
additionalPrinterColumns to retrieve values from custom resources using
jsonpaths.

The majority of the remaining work will be to update the API to add the
field and to validate it.

### User Stories

- CustomResourceDefinition author wishes to provide filtered access by a field
  - Author adds the field to an allow list in the CustomResourceDefinition
- User uses `--field-selector` in kubectl to filter a list response.
  - User first needs to know what field selectors are available and what values the selectable fields might have.
    - Places user might look: discovery, CRD resource.
  - User filter by:
    - a single field
    - multiple fields
    - both fields and labels
- Controller watches custom resources while filtering with field selectors.
  - Controller provides a FieldSelector string in ListOptions 

### Notes/Constraints/Caveats (Optional)

Discussions about limitations of field selectors:

- https://github.com/kubernetes/kubernetes/issues/32946
- https://github.com/kubernetes/kubernetes/issues/72196
- https://github.com/kubernetes/kubernetes/issues/107053

Problems with escaping:

- https://github.com/kubernetes/kubernetes/pull/28112

### Future work

#### Declarative field selector definitions on built-in types

There are two aspects of this:

- built-in selectableFields converted to declarative go tags
- built-in selectableFields included in OpenAPIv3

To make it easier and safer to add field selector definitions for built-in types
in the future. We will support defining field selectors on API types using
go struct tags.

```go
type PodSpec struct {
  // NodeName ...
  //
  // +selectableField
  NodeName string `json:...`

  // Host networking ...
  //
  // +k8s:schema:selectableField:fieldPath="spec.hostNetwork",
  // +k8s:schema:selectableField:default="false"
  // +k8s:schema:selectableField:description="Set to the value of the spec.securityContext.hostNetwork field"
  // +optional
  HostNetwork bool `json:...`
}
```

String, bool and integer typed fields, and typerefs to these types will be
supported and automatically cast to string.

`default` is needed to handle some of the existing builtin cases that provide
a value that is not an empty string if the field is unset.

`fieldSelector` should only be set for legacy cases where the field selector
name does not match the field path. When set, a description will be required.

`description` is provided for the legacy cases that use `fieldSelector` and
require further clarification. For all other cases, it should not be needed.

`Pod.status.podIP[0]` is a special legacy case. We will handle it in code. We
may need to add special go tag part to allow us to include declarative
information in the go struct for discovery purposes while handling the
implementation imperatively.

The OpenAPI would also include descriptions for built-in types. This is needed
to explain some of the legacy field selectors that have surprising behavior.

```json
"io.k8s.api.core.v1.Pod": {
        "description": "Pod is a collection of containers that can run on a host. This resource is created by clients and scheduled onto hosts.",
        "properties": { ... },
        "type": "object",
        "x-kubernetes-group-version-kind": [ ... ],
        "x-kubernetes-selectable-fields": [
          { "fieldPath": "metadata.name"},
          { "fieldPath": "metadata.namespace"},
          { "fieldPath": "spec.nodeName"},
          { "fieldPath": "spec.restartPolicy"},
          { "fieldPath": "spec.schedulerName"},
          { "fieldPath": "spec.serviceAccountName"},
          { "fieldPath": "spec.hostNetwork", "default": "false", "description": "Set to the value of the spec.securityContext.hostNetwork field"},
          { "fieldPath": "status.phase"},
          { "fieldPath": "status.podIP", "default": "", "description": "Set to the value of status.podIPs[0]"},
          { "fieldPath": "status.nominatedNodeName"},
        ]
      },
```

#### Index selected fields

Label and field selectors may be accelerated in the future. Possible approaches:

- etcd adds filter/index/symlink support in the future and Kubernetes leverages it
- the watch cache is updated to index labels and field selectors

#### CEL as an accelerated predicate language

I had drafted a [CEL based field selection proposal in the past](https://docs.google.com/document/d/1-61zFxrhjywPlPteaU22_Q3FOxA_kJ7acVRvBlgPG84/edit#heading=h.zy7gz3toxt1) and discussed it in a [larger thread about field selection](https://groups.google.com/g/kubernetes-sig-cli-feature-requests/c/hUvQhfJxHls).

One of the biggest problems is that there is a real possibility that label and
field selectors will indexed in the future (as described above), and the
community would like to avoid introducing changes to label or field selectors
that prevent this from happening. The naive use of CEL--evaluate objects
one-by-one in a full scan--would be very expensive in comparison to indexed
selectors.

But there is a way CEL expressions could be accelerated, so long as the
expressions only contain references to indexed fields.

Imagine a CEL expression like `spec.x == 'a' && spec.y == 'b'`.

This expression is conceptually very similar to a AND WHERE clause in SQL,
which can be optimized SQL query engine so long as the ANDed fields are indexed.

With CEL, an expression's AST could be inspected and evaluated using an approach
similar to those used by query engines.  For example, for `spec.x == 'a' &&spec.y == 'b'`
the field index for `spec.x` and `spec.y` could be inner joined to evaluate
the expression.

For cases where this acceleration technique works:
- filtered lists can be served w/o a full scan
- filtered watches be optimized using [content-based subscription approaches](https://groups.google.com/g/kubernetes-sig-cli-feature-requests/c/hUvQhfJxHls/m/eeNPw-PSBQAJ)

Operations that canot be accelerated this could be assigned high costs,
restricting the use of the CEL expressions entirely, or limiting their use.

CEL operations that could potentially be accelerated might include:

- boolean logic: `&&`, `J||`, `!`
- set operations: `in`, [`sets.intersects()`](https://github.com/google/cel-go/blob/master/ext/sets.go)
- some string operations: `string.endsWith()` (requires more advanced prefix matching indices)

I do realize this implies the introduction of a CEL query engine for Kubernetes.
But not all query engines are large or complex. If indexed, label selectors
would require a small query engine that supports AND, NOT and IN. A simple CEL
query engine might introduce OR and some other operations, but could be kept
small and lean and still offer a substantial boost to the query options
available today. A major benefit of CEL is that it provides a stable
grammar that is already used elsewhere in Kubernetes.

### Risks and Mitigations

#### Increases watch cache resource utilization

At first glance, the resource utilization cost to the watch cache will be
roughly equivalent to the cost of using labels, suggesting that this enhancement
doesn't introduce any fudamentally new risks.

But there is still a potential problem. CustomResourceDefinition authors might
choose to add a large number of fields to `selectableFields`.  This is plausible
because it is so easy to add fields to `selectableFields`. For comparison,
double writing data both to labels and to spec or status fields is significantly
significantly more complicated and error prone. (Note, however, that once
MutatingAdmissionPolicy is available writing data to labels for selector
purposes will become easier and safer, so it might be good to get this
enhancement in sooner than later).

Mitigations: 
 - Limit `selectableFields` per CustomResourceDefinition to a small number (e.g. 8).
   We will pick the exact limit emperically after measuring the change in
   resource utilization.
 - Clearly document that each selectable field has associated resource costs.

## Design Details

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- staging/src/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/validation/validation_test.go
  - [x] duplicate selectableFields not allowed
  - [x] invalid path not allowed
  - [x] path to invalid type not allowed
  - [x] too many selectableFields not allowed
  - [x] empty string is set if field may be absent

- staging/src/k8s.io/apiextensions-apiserver/pkg/registry/customresource/strategy_test.go
  - [ ] ensure field paths are not dereferenced if not used, or make it fast and benchmark it
  - [ ] all possible type cases, including string format cases with special type bindings (duration, intOrString)

- staging/src/k8s.io/apiextensions-apiserver/pkg/registry/customresourcedefinition/strategy_test.go
  - [x] all drop field cases

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

For Alpha: 

- test/integration/controlplane/crd_test.go
  - [x] selectableFields cannot be written feature gate disabled
  - [x] feature disablement testing
  - [x] Test list selectable field {with, without default}, {with single field, with multiple fields, with combined use of label selectors} for {list, watch}
  - [ ] Use of a field selector that does not exist
  - [x] test with multiple versions where the field moves from field path 1 to field path 2 across serializations and the REST request does not match the serialization stored in etcd
  - Test watch selectable field
    - [x] Only field selected resources are observed
    - [ ] When a watched item changes values for its selection. Does a delete
          get issued to the watch?
    - [ ] Use of a field selector that does not exist
  - [ ] Test DeleteCollection with a field selector
  - [x] Test reading selectableField data from OpenAPI v2 and v3


##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

For Beta:

- test/e2e/apimachinery/custom_resource_definition.go
  - TODO: Test selectable fields with combined multiple fields and combined with label selectors for {list, watch}

- <test>: <link to test coverage>

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

--->

#### Alpha

- CRD selectableFields implemented
- CRD selectableField data available in OpenAPI discovery
- Feature implemented behind a feature flag
- integration tests completed and enabled

#### Beta

- Optimize GetAttrs to only incur JSONPath lookup cost for selectableFields when actually used for field selection
- e2e tests completed and enabled

## GA

- Add test coverage for informers
- Promote e2e tests to conformance

<!--
#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: CustomResourceFieldSelectors
  - Components depending on the feature gate: kube-apiserver

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes

The enhancement's field selection will become unavailable on CRDs when disabled.
When this happens, any clients depending on the new field selectors will receive
an HTTP `400 Bad Request` response with a message like:

```
"Unable to find \"stable.example.com/v1, Resource=selector\" that match label selector \"\", field selector \"spec.colorx=blue\": field label not supported: spec.colorx"
```

This is idential to the behavior before the enhancement is enabled.

selectableFields are removed from write requests to CRDs that do
not already have selectableFields.

No other system behavior is impacted.

###### What happens if we reenable the feature if it was previously rolled back?

Any selectableFields already written to CRDs are reactivated and requests to select
by the fields will be filtered correctly.

selectableFields can be added to CRDs again.

###### Are there any tests for feature enablement/disablement?

Yes

Unit tests: staging/src/k8s.io/apiextensions-apiserver/pkg/registry/customresourcedefinition/strategy_test.go:

```
=== RUN   TestDropDisabledFields/SelectableFields,_For_create,_FG_disabled,_SelectableFields_in_update,_dropped
=== RUN   TestDropDisabledFields/SelectableFields,_For_create,_FG_enabled,_no_SelectableFields_in_update,_no_drop
=== RUN   TestDropDisabledFields/SelectableFields,_For_create,_FG_enabled,_SelectableFields_in_update,_no_drop
=== RUN   TestDropDisabledFields/SelectableFields,_For_update,_FG_disabled,_oldCRD_has_SelectableFields,_SelectableFields_in_update,_no_drop
=== RUN   TestDropDisabledFields/SelectableFields,_For_update,_FG_disabled,_oldCRD_does_not_have_SelectableFields,_no_SelectableFields_in_update,_no_drop
=== RUN   TestDropDisabledFields/SelectableFields,_For_update,_FG_disabled,_oldCRD_does_not_have_SelectableFields,_SelectableFields_in_update,_dropped
=== RUN   TestDropDisabledFields/SelectableFields,_For_update,_FG_enabled,_oldCRD_has_SelectableFields,_SelectableFields_in_update,_no_drop
=== RUN   TestDropDisabledFields/SelectableFields,_For_update,_FG_enabled,_oldCRD_does_not_have_SelectableFields,_SelectableFields_in_update,_no_drop
=== RUN   TestDropDisabledFields/pre-version_SelectableFields,_For_update,_FG_disabled,_oldCRD_does_not_have_SelectableFields,_SelectableFields_in_update,_dropped
```

Integration tests: staging/src/k8s.io/apiextensions-apiserver/test/integration/fieldselector_test.go

```
=== RUN   TestFieldSelectorDropFields
```


### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

This feature can only be used when enabled and does not persist any state.

Rollout could only fail if a defect in the implementation were to somehow impact
list/watch code paths not using this feature.

Once rolled out, rollback would only impact clients using the feature.

###### What specific metrics should inform a rollback?

[request_total](https://github.com/kubernetes/kubernetes/blob/a97f4b7a3123c9768ec7136b6ca32be926e16cd6/staging/src/k8s.io/apiserver/pkg/endpoints/metrics/metrics.go#L81). This metric can be monitored for non-200 response codes.


###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Yes. Note however, that a upgrade after a downgrade does not matter for this feature since it does not persist any state.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Check if the "has_field_selector" label (plan is to add this for beta) on [request_total](https://github.com/kubernetes/kubernetes/blob/a97f4b7a3123c9768ec7136b6ca32be926e16cd6/staging/src/k8s.io/apiserver/pkg/endpoints/metrics/metrics.go#L81) is used on any CRDs.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [x] Other (treat as last resort)
  - Details: Use the feature to filter when listing CRDs.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This feature reduces the expected sizes of list responses, and so falls under the SLOs for read responses.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name:
    - [request_total](https://github.com/kubernetes/kubernetes/blob/a97f4b7a3123c9768ec7136b6ca32be926e16cd6/staging/src/k8s.io/apiserver/pkg/endpoints/metrics/metrics.go#L81)
    - [request_duration_seconds](https://github.com/kubernetes/kubernetes/blob/a97f4b7a3123c9768ec7136b6ca32be926e16cd6/staging/src/k8s.io/apiserver/pkg/endpoints/metrics/metrics.go#L99)

Elevated HTTP 400 response codes in `request_total` for list and watch is a SLI for potential problems
with requests using label selectors.

High `request_duration_seconds` for list requests may indicate a performance problem with label selectors.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Yes, we will add "has_label_selector" and "has_field_selector" labels to the request_total and request_duration_seconds. We intend to keep it simple (for low cardinality) and only use a true/false label value.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No
-->

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

Note that this feature does not fundamentally enable capabilities not already available.  Today, users add labels to resources
to enable filtering. This feature merely eliminates the need to "labelize" resources in this way.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The feature is provided by the API server and is unavailable if the API server is unavailable.

Requests served by etcd are also unavailable when etcd is unavailable. (Watch cache served requests remain available).

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

Check the request_duration_seconds metric where 'has_field_selector=true' on CRD types to identify if filtered
list requests exceed the SLO. Further narrow the SLO down to a specific 'component' or inspect apiserver logs
to identify the exact requests exceeding the SLO.

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

### OpenAPI Discovery

#### Declare selectable fields to the fieldSelector path parameter of OpenAPI

Path parameters for `fieldSelector` are arguably the correct place in OpenAPI to
document selectable fields.

We decided against this option because, pragmatically, it is less convenient for clients
to leverage since clients have significant tooling to process and interpret OpenAPI schemas
but have very little tooling for parameters.

```json
 "/api/v1/namespaces/{namespace}/pods": {
      ...
      "parameters": [
        ...
        {
          "description": "A selector to restrict the list of returned objects by their fields. Defaults to everything.",
          "in": "query",
          "name": "fieldSelector",
          "schema": {
            "type": "string",
            "uniqueItems": true,
            "x-kubernetes-selectable-fields": [
              { "fieldPath": "metadata.name"},
              { "fieldPath": "metadata.namespace"},
              { "fieldPath": "spec.nodeName"},
              { "fieldPath": "spec.restartPolicy"},
              { "fieldPath": "spec.schedulerName"},
              { "fieldPath": "spec.serviceAccountName"},
              { "fieldPath": "spec.hostNetwork", "default": "false", "description": "Set to the value of the spec.securityContext.hostNetwork field"},
              { "fieldPath": "status.phase"},
              { "fieldPath": "status.podIP", "default": "", "description": "Set to the value of status.podIPs[0]"},
              { "fieldPath": "status.nominatedNodeName"},
              ...
            ]
          }
        },
        ...
      ...
 }
```

#### Declare selectable fields by annotating schema fields as being selectable, e.g.:

```
"hostNetwork": {
            "description": "...",
            "type": "boolean",
            "x-kubernetes-selectable": {
                "default": "false",
            }
          },
```

Problem swith this approach:

- Field selection is a property of an API, not a field.  Some types are composed
  into multiple APIs and if we were to put field selection onto fields, then
  anywhere the type is resused, field selection would be enabled, which is not
  what we want.
- For legacy reasons, not all selectable fields use an identifier that has a
  corresponding field in the schema. For example, but pods don't have a
  `status.podIP` field. Pods have a `status.podIPs` which is *almost* the same,
  but not quite. There are 6+ fields like this, see the table in the proposal
  section for full field selectors for more detail.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
