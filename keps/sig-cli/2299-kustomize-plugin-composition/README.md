# KEP-2299: Kustomize Plugin Composition API

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [Integration with Kustomization](#integration-with-kustomization)
    - [Integration with Kubectl Kustomize](#integration-with-kubectl-kustomize)
    - [Plugin execution flags](#plugin-execution-flags)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Key terminology](#key-terminology)
  - [API schema](#api-schema)
  - [Built-in transformers](#built-in-transformers)
  - [Composition evaluation](#composition-evaluation)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in
  [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to
  [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list
  discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website[Configuration Functions Specification]:
https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md

## Summary

Introduces a new Kustomize API (`kind`) that is oriented around Kustomize plugins, making them a
first-class expression of a resource configuration bundle.

This new API will provide new sophisticated capabilities for using plugins as composable units. It
will showcase plugins as a way to implement composable, declarative client-side abstractions.

## Motivation

The `Kustomization` API was designed around built-in Kustomize transformer, generator and validator
operations applied to Kustomize bases. While plugin transformers can be invoked via the appropriate
field, `Kustomization` is suboptimal for workflows that are primarily composed of Kustomize
plugins. Challenges with the current approach include:

1. **Sequencing of built-in and plugin transformers is static**. The end user can choose to
categorize a plugin as a generator, transformer or validator, but can only control relative
execution order within those fields. Especially in the case of abstraction-based plugins, the user
may not know which category they fall in, and indeed the plugin may change what it does based on
input. This can be addressed by placing all plugins under `transformers`. However, doing so means
they will all be executed after built-in transformers. The result is that an additional
`Kustomization` layer is typically required to use built-ins fruitfully in abstraction-based
workflows.

1. **Plugin config transformation must be isolated**. Although it is possible to
transform plugin configuration as resources before execution, it is not possible to have a sensible
distribution of concerns in configuration sets composed this way. Notably, plugin configuration
must be finalized (e.g. tailored to an environment) before it can be combined with built-in
configuration at all, and built-in transformers cannot be specified in bases, or else they will not
apply to the results of the plugins. [example/kustomization](example/kustomization) demonstrates this difficulty.

The sequencing issue could be solved within `Kustomization`; it is the difficulty with plugin
config transformation that motivates an entirely new Kind.

### Goals

Develop an API (`kind`) for Kustomize that is focused on plugin-based workflows.

A successful implementation of this API should have the following characteristics:

1. It must be easy to recursively compose and modify lists of plugins in a way that yields an
overall configuration structure that reflects the user's intent.

1. Plugin orchestration must be
central to the new API format. Built-in and plugin transformer config should be treated the same
way in the new API.

1. The new API must cleanly integrate with the existing Kustomize tool
(i.e. `kustomize build`) as well as existing Kustomize kinds.

### Non-Goals

- Replace Kustomization for use cases that involve few or no plugins.
- Expand the scope of the existing `Kustomization` API.
- Introduce plugin distribution or discovery mechanisms.
- Change the flags and parameters used for invoking plugins in any way.

## Proposal

Introduce `Composition` as a new API `kind` recognized by `kustomize build`. A `Composition` must be
defined in a `composition.yaml` file. Inclusion of both `composition.yaml` and `kustomization.yaml`
in the target directory of a build will be considered an error.

The `Composition` API enables users to:

- Define a list of transformer configuration that intermingles built-in and plugin transformers.
- Import transformers from another `Composition` and add them to the list.
- Override an imported transformer's fields with new values.
- Reorder the list of transformers prior to execution.

Once a user has written a `Composition`, they can run `kustomize build` to turn it into a list of
Kubernetes resources that they can commit to git, the same way they would with `Kustomization`.

To provide this experience, Kustomize will need to do the following during builds:

- Consolidate the `Composition` and its imports into a finalized list of transformers (which may be
  plugins).
- Execute each transformer in sequence (no changes to plugin execution are proposed by this KEP).
- Pass the output of each transformer as the input to the next.

### User Stories (Optional)

#### Story 1

As a user, I can define my configuration as a sequence of declarative abstractions implemented by
plugins.

```yaml
# app/composition.yaml
kind: Composition

transformers:
# generate resources for a Java application
- apiVersion: example.com/v1
  kind: JavaApplication
  provider: {container: {image: docker.example.co/kustomize_transformers/java:v1.0.0}}
  metadata:
    name: my-app
  spec:
    application: team/my-app
    version: v1.0
# transform resources to inject a logger
- apiVersion: example.com/v1
  kind: Logger
  provider: {container: {image: docker.example.co/kustomize_transformers/logger:v1.0.3}}
  metadata:
    name: logger
```

#### Story 2

As a user, I can reuse and extend an existing `Composition` by importing its transformers and
optionally overriding imported transformers' fields.

```yaml
# staging/composition.yaml
kind: Composition

transformersFrom:
# import the JavaApplication and Logger transformers
- path: ../app/composition.yaml
  importMode: prepend

transformerOverrides:
# override the JavaApplication version before the function is run
- apiVersion: example.com/v1
  kind: JavaApplication
  metadata:
    name: my-app
  spec:
    version: v1.1-beta

transformers:
# transform resources from imported transformers to inject metrics
- apiVersion: example.com/v1
  kind: Prometheus
  provider: {container: {image: docker.example.co/kustomize_transformers/prometheus:v1.0.2}}
  metadata:
    name: metrics
  spec:
    prefix: my-app-
```

### Notes/Constraints/Caveats (Optional)

#### Integration with Kustomization

By beta at the latest, `Kustomization`s should be able to reference `Composition`s in various fields
as follows:
- the `resources` and `generators` fields: the `Composition` will be executed with empty input to the
  first transformer in the consolidated `Composition`, and the results will be appended to the
  resource list.
- the `transformers` field: the resource list will be provided as input to the first transformer in
  the consolidated `Composition`, and the output will replace the resource list.
- the `validators` field: the resource list will be provided as input to the first transformer in
  the consolidated `Composition`, and the output (if any) will be discarded.

#### Integration with Kubectl Kustomize

`Composition` support will neither be required nor suppressed in `kubectl kustomize` during alpha. In
other words, if a `kubectl kustomize` release happens during alpha, it can include the feature, but
graduation to beta does not require this. `Composition` support MUST be included in at least two
`kubectl kustomize` releases during beta.

#### Plugin execution flags

This KEP does not propose any changes to the flags used to gate plugins, or to the parameters
available to configure container/exec/starlark plugin execution. All plugin execution from either
Kind will remain gated behind `--enable-alpha-plugins`, as well as existing additional flags for
particular plugin provider types. Any changes to plugin gating should apply identically to both
Kinds. This does limit the usefulness of `Composition` within `kubectl -k` for the time being, since
only a subset of plugins can be enabled in that context.

### Risks and Mitigations

This proposal effectively takes a feature set that exists in Kustomize today and makes it much more
usable in certain workflows. As such, it does not come with any novel technical risks.

## Design Details

### Key terminology

This terminology is not set in stone at this early stage, but should be used consistently within
this proposal.

  * **Composition**: a new plugin-oriented API (`kind`) to be understood by `kustomize build`.
  * **Transformer**: a program that generates, manipulates and/or validates a stream of Kubernetes
  resources, along with the corresponding YAML config object used to configure that program.
    * **Plugin transformer**: A user-authored transformer.
    * **Built-in transformer**: A transformer whose configuration spec and implementation are part
        of Kustomize itself.
  * **Plugin provider**: The program that implements the plugin (e.g. container, script, Go
      program). Analogous to the controller for a custom resource.
  * **Plugin/transformer config**: The KRM-style YAML document that declares the desired state the
      transformer implements. In the case of a plugin, it includes the provider to execute and the
      specification to follow in doing so. Analogous to a custom resource object.

### API schema

Example showing all fields:

```yaml
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Composition

# `transformersFrom` imports transformer lists from other Compositions.
# The target Compositions' own imports, overrides and reorderings are applied before it is imported.
transformersFrom:
- path: ../app/composition.yaml
  importMode: prepend # this is the default; append is also possible

# `transformerOverrides` allows fields of transformers imported via `transformersFrom` to be changed
# before execution.
# Entries are treated as strategic merge patches.
transformerOverrides:
- apiVersion: example.com/v1
  kind: JavaApplication
  metadata:
    name: my-app
  spec:
    application: team/my-app
    version: v3.1.3

# `transformers` adds new transformers to the list.
transformers:
- apiVersion: example.com/v1
  kind: Prometheus
  provider:
    container:
      image: example/prometheus:v1.0.2
  metadata:
    name: metrics-injector

# `transformerOrder` allows advanced users to reorder the merged transformers list explicitly.
# transformers can be referred to by name alone as long as names are unique.
transformerOrder:
  - name: my-app
  - name: logger
  - name: metrics
```

Notes:
- One notable difference from `Kustomization` is that in addition to being able to provide a path to
  a file containing the transformer config, you can specify the config itself (not strigified)
  inline.
- A second difference is the introduction of a top-level reserved `provider` field, replacing the
  JSON annotation currently used for this information.
- The `metadata.name` field is defaulted to `Kind` in kebab case. However, `metadata.name` MUST be
specified when using multiple independent instances of the same GVK within a `Composition` tree.

<details>
  <summary>Full schema</summary>

```yaml
definitions:
  transformer:
    type: object
    additionalProperties: true
    required:
      - apiVersion
      - kind
      - metadata
    properties:
      kind:
        type: string
        minLength: 1
      apiVersion:
        type: string
        pattern: "^[a-z0-9][a-z0-9\\.]*\\/[a-z0-9]+$"
      metadata:
        type: object
        required:
          - name
        additionalProperties: false
        properties:
          name:
            type: string
            minLength: 1
            maxLength: 253
            pattern: "^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$"
      provider:
        type: object
        additionalProperties: false
        properties:
          container:
            type: object
            required:
            - image
            additionalProperties: false
            properties:
              image:
                type: string
                minLength: 1
          starlark:
            type: object
            required:
            - path
            additionalProperties: false
            properties:
              path:
                type: string
                minLength: 1
type: object
required:
  - apiVersion
  - kind
additionalProperties: false
properties:
  apiVersion:
    type: string
    enum: ["kustomize.config.k8s.io/v1alpha1"]
  kind:
    type: string
    enum: ["Composition"]
  transformersFrom:
    type: array
    items:
      type: object
      required:
        - path
      additionalProperties: false
      properties:
        path:
          type: string
          minLength: 1
        importMode:
          type: string
          enum: ["append", "prepend"]
  transformers:
    type: array
    items:
      "$ref": "#/definitions/transformer"
  transformerOverrides:
    type: array
    items:
      "$ref": "#/definitions/transformer"
  transformerOrder:
    type: array
    items:
      type: object
      required:
      - name
      additionalProperties: false
      properties:
        kind:
          type: string
          minLength: 1
        apiVersion:
          type: string
          pattern: "^[a-z0-9][a-z0-9\\.]*\\/[a-z0-9]+$"
        name:
          type: string
          minLength: 1
          maxLength: 253
          pattern: "^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$"
```
</details>

### Built-in transformers

[everything is already a transformer]:
https://kubectl.docs.kubernetes.io/references/kustomize/kustomization/#everything-is-a-transformer
[accumulateResources]:
https://github.com/kubernetes-sigs/kustomize/blob/bf6b207cc997b689887c5b1cfc9a4f81f3012b12/api/internal/target/kusttarget.go#L346

In Kustomize, [everything is already a transformer]. In a Composition, users can include any of the
existing built-in transformers (including generators and validators) in their transformers list.

```yaml
transformers:
- apiVersion: builtin
  kind: PrefixSuffixTransformer
  metadata:
    name: myFancyNamePrefixer
  prefix: bob-
  fieldSpecs:
  - path: metadata/name
```

One important exception to this is the `resources:` field, which is not already a generator. Work
will need to be done to expose [accumulateResources] as one.

```yaml
transformers:
- apiVersion: builtin
  kind: ResourceAccumulator
  paths:
  - path/to/file.yaml
```


In addition to that, `kind: Kustomization` itself may be inlined as a transformer. Because it is a
transformer in this context, resources from previous transformers will be written to a temp file
that will be automatically prepended to the `Kustomization`'s `resources` field.

```yaml
transformers:
- apiVersion: kustomize.config.k8s.io/v1beta1
  kind: Kustomization
  commonLabels:
    foo: bar
  resources:
  - config.yaml
```

### Composition evaluation

Composition topologically sorts imported transformers with its own transformers and then runs the
combined list sequentially. This means that arbitrary depths of imports always result in a single
consolidated Composition prior to execution. Once that consolidated Composition has been compiled,
its transformers will be invoked the same way they are today from the `transformers` field in
`Kustomization`.

For use cases that require resources to be evaluated eagerly before further processing, for example
diamond-shaped configuration, the built-in `ResourceAccumulator` can be given a Composition to render
into resources.

If it is deemed useful for debugging, a command for consolidating Composition without executing it
may be introduced to the `kustomize cfg` command group (or equivalent, if `cfg` is replaced).

### Test Plan

Testing can be done purely on the client-side without the need for a cluster.

### Graduation Criteria

#### Alpha

- `Composition` implemented with an alpha GV and supported by `kustomize build`
  - Compatibility with existing built-in transformers
  - `ResourceAccumulator` extracted as a transformer
  - `transformer`, `transformersFrom` and `transformerOverrides` fields implemented.
  - reserved `provider` field
- Basic documentation added to the Kustomize website.

#### Beta

- `Composition` beta GV supported by `kustomize build`
  - `Kustomization` transformer implemented
  - `transformerOrder` field implemented
- `Composition` supported by the `resources`, `transformers`, `generators` and `validators` fields
  in `Kustomization` and `Component`
- Thorough documentation and examples published on the Kustomize website.

#### GA

- At least two releases of `kubectl kustomize` have included `Composition` support, to allow for
  feedback from the wider community.
- Inline transformer config support in the `transformers`, `generators` and `validators` fields of
  `Kustomization`
- TBD

#### Deprecation

As with any Kustomize feature, deprecation post-alpha would need to take the `kubectl kustomize`
release cycle into consideration as well as Kustomize's own, to ensure all users are adequately
warned before removal.

### Upgrade / Downgrade Strategy

N/A -- feature is client-side-only

### Version Skew Strategy

N/A -- feature is client-side-only

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

No special flags will be needed to use `Composition`, which will have a GV indicative of its
maturity level. The same flags required to use plugins in existing contexts will be required to use
them via `Composition`.

###### How can this feature be enabled / disabled in a live cluster?

N/A -- feature is client-side-only

###### Does enabling the feature change any default behavior?

N/A -- feature is client-side-only

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

N/A -- feature is client-side-only

###### What happens if we reenable the feature if it was previously rolled back?

N/A -- feature is client-side-only

###### Are there any tests for feature enablement/disablement?

N/A -- no feature flag required

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

N/A -- feature is client-side-only

###### What specific metrics should inform a rollback?

N/A -- feature is client-side-only

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A -- feature is client-side-only

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

N/A -- feature is client-side-only

###### How can an operator determine if the feature is in use by workloads?

N/A -- feature is client-side-only

###### How can someone using this feature know that it is working for their instance?

N/A -- feature is client-side-only

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A -- feature is client-side-only

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A -- feature is client-side-only

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A -- feature is client-side-only

### Dependencies

This feature does not require any new Golang dependencies.

###### Does this feature depend on any specific services running in the cluster?

No -- feature is client-side-only

### Scalability

This feature is not expected to have significantly different performance characteristics than the
use of transformers in Kustomization. Performance in practice will depend largely on the type and
implementation of plugins used.

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

Yes, but only client-side within Kustomize

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

N/A -- feature is client-side-only

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A -- feature is client-side-only

###### What are other known failure modes?

Kustomize plugins are immature (various alpha mechanisms) and when they fail to execute, it is not
always graceful. Golang plugins in particular come with a long list of caveats. KRM Exec and
Starlark plugins are not currently enabled in `kubectl kustomize` at all.

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A -- feature is client-side-only

## Implementation History

- April 27, 2021: Provisional KEP merged
- August 2021: Proposal updated and marked implementable.

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section. Major milestones might
include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

* The new API provides another way to run plugins, and may be confusing to users trying to decide
  which to use. Creating separate APIs for Deployment, StatefulSet, DaemonSet have similar
  drawbacks.

* Since this new API is oriented entirely around plugins, it will likely lead to workflows in which
  a considerable number of plugins must be invoked to render the resource list. Especially for
  container plugins, this may not be performant. Although the same issue would exist in today's
  plugins via Kustomization, it may become more apparent if plugins get a substantial increase in
  ease of use.

## Alternatives

* Implement these capabilities in the `Kustomization` API. All of the fields proposed in the new API
  could instead be added as top-level fields, or grouped fields, in `Kustomization`. However, the
  result would not be very elegant. In particular, one of the primary proposed features--enabling
  transformer configuration to be overridden--requires a completely different evaluation model. Namely,
  a topological sort is applied to the entire tree of transformers prior to evaluation, as opposed to
  Kustomization's current model of executing generators, transformers and plugins in each level in
  isolation. Combining these two mental models within a single API would be difficult and likely
  lead to considerable confusion among users.
* Implement this as an independent tool. This is certainly possible; `Composition` itself is a
  transformer, and could be implemented as a plugin. However, this proposal solves a usability
  problem faced by existing Kustomize features. As such, it would be great if the Kustomize
  community could leverage it directly.
