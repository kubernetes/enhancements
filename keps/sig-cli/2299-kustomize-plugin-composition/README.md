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
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Security of running containers locally](#security-of-running-containers-locally)
- [Design Details](#design-details)
  - [Key terminology](#key-terminology)
  - [API schema](#api-schema)
  - [Built-in modules](#built-in-modules)
  - [Function invocation](#function-invocation)
  - [Composition evaluation](#composition-evaluation)
  - [Function packaging and distribution](#function-packaging-and-distribution)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Introduces a new Kustomize API (`kind`) that is oriented around Kustomize plugins, making them a first-class expression of a resource configuration bundle.

This new API will provide new sophisticated capabilities for using plugins as composable
units and include support for automatic discovery and installation of plugins.  It will showcase
plugins as a way to implement composable, declarative client-side abstractions.

## Motivation

The `Kustomization` API was designed around built-in Kustomize transformer and generator
operations applied to Kustomize bases. This API is suboptimal for workflows that
are primarily composed of Kustomize plugins.  Challenges with the current approach include:

1. Packaging, distribution and installation of plugins is immature and non-declarative.
2. Orchestration, ordering and dependencies is overly complex due to its integration
with the orchestration of built-in operations. For example, `Kustomization` requires
generators and transformers to be specified and executed separately, whereas a given
plugin may do both (or neither, as in a validator plugin).
3. Plugin execution happens during the evaluation of the `Kustomization` where it is
specified. Overlays cannot modify plugin values before they are evaluated,
which seriously hinders plugin usability for developing composable abstractions.
4. The `Kustomization.yaml` format does not elegantly allow plugins to be
specified inline. Instead, they are defined in separate files, which obfuscates
the holistic user intent in workflows primarily driven by plugins.

### Goals

Develop an API (`kind`) for Kustomize that is focused on plugin-based workflows.

A successful implementation of this API should have the following characteristics:

1. The orchestration model used to evaluate the API must be simplified in a way
that is optimized for plugins. Notably, it must be possible for lists of plugins to be
recursively composed, and for overlay instances of the API to modify the
configuration of plugins they import.
1. The API format must be optimized for plugins. This means most or all of its
top-level fields should configure plugin orchestration in some way.
Support for existing Kustomize operations should be compiled in, but
expressed in the same way as extensions plugin operations.
1. The machinery for the new API must enable seamless invocation of sets of approved
plugins, and must not require out-of-band imperative installation steps.
1. The new API must cleanly integrate with the existing Kustomize tool (i.e. `kustomize build`).

### Non-Goals

- Replace Kustomize as "the way" to do anything.
- Expand the scope of the existing `Kustomization` API.
- Directly integrate with `kubectl apply`, `kubectl diff`, etc. The new API will
 be compatible with those and other tools via evaluation into a resource list
 suitable for use in gitops workflows.
- Pull resources or files from a remote git source.

## Proposal

Introduce `Composition` as a new API `kind` recognized by `kustomize build`.

The Composition API enables users to:

- Define a list of Kubernetes-style configuration objects called **modules**.
A module is a client-side resource that expresses the desired state implemented by
a Kustomize plugin.
- Import modules from another Composition and add them to the list.
- Override an imported module's fields with new values.
- Reorder the list of modules prior to execution.

Once a user has written a Composition, all they need to do is run `kustomize build`
to turn it into a list of Kubernetes resources that they can commit to git.

To provide this experience, Kustomize will need to do the following during builds:

- Consolidate the Composition and its imports into a finalized list of modules.
- Automatically fetch and execute the plugin that implements each module.
- Pass the output of each plugin invocation as the input to the next plugin.

### User Stories (Optional)

#### Story 1

As a user, I can define my configuration as a sequence of declarative abstractions
implemented by plugins that are automatically discovered and installed.

```yaml
# app/composition.yaml
kind: Composition

modules:
# generate resources for a Java application
- apiVersion: example.com/v1
  kind: JavaApplication
  provider: {container: {image: example/module_providers/java:v1.0.0}}
  metadata:
    name: my-app
  spec:
    application: team/my-app
    version: v1.0
# transform resources to inject a logger
- apiVersion: example.com/v1
  kind: Logger
  provider: {container: {image: example/module_providers/logger:v1.0.3}}
  metadata:
    name: logger
```

#### Story 2

As a user, I can reuse and extend an existing Composition by importing its modules and
optionally overriding imported modules' fields.

```yaml
# staging/composition.yaml
kind: Composition

modulesFrom:
# import the JavaApplication and Logger modules
- path: ../app/composition.yaml
  importMode: prepend

moduleOverrides:
# override the JavaApplication version before the function is run
- apiVersion: example.com/v1
  kind: JavaApplication
  metadata:
    name: my-app
  spec:
    version: v1.1-beta

modules:
# transform resources from imported modules to inject metrics
- apiVersion: example.com/v1
  kind: Prometheus
  provider: {container: {image: example/module_providers/prometheus:v1.0.2}}
  metadata:
    name: metrics
  spec:
    prefix: my-app-
```

### Notes/Constraints/Caveats (Optional)

Although direct integration with the existing `Kustomization` API could be done,
it is outside the scope of this proposal and carries risks that must be considered.
For example, given that `Kustomization` is integrated with existing workflows such as `kubectl apply -k`,
the introduction of automatic plugin installation and execution may be undesirable.

### Risks and Mitigations

#### Security of running containers locally

Composition has similar capabilities to Kustomization.

The key difference for security is that it executes plugins automatically (no alpha flag
required) and does not require them to be installed in advance / out of band. This difference is
key to the user experience, and it does decrease user awareness of exactly what is
being executed. As such, it must be made clear to users that Compositions are not inherently
secure and that they should only build Compositions they trust.

This risk will be mitigated in the following ways:

1. Container-based functions will be run without network or volume access. This does
limit the functionality they can implement, but it also encourages plugins
to be developed such that the module resources that configure them express
the entire desired state. Should this prove unacceptably limiting, network or volume access
may be introduced in a future version, but likely with a more explicit and restrictive
permissions model than plugins have today. The same is true for exec plugins: they
will not be initially supported, and additional restrictions will likely be placed on them
if support is introduced in the future.

1. Even if network or volume access support is added, containers will still run
without it unless the user's Composition explicitly grants it. If exec support
is added, a checksum may be required in addition to the binary location.

1. Plugins will not be trusted by default. The execution framework will feature
a pluggable trust model that enables users and organizations to allowlist plugin
sources. It should be possible for large organizations to distribute binaries with
specific allowlists compiled in, and for additional sources to be allowlisted at
invocation time.

## Design Details

### Key terminology

This terminology is not set in stone at this early stage, but should be
used consistently within this proposal.

  * **Composition**: a new plugin-oriented API (`kind`) to be understood by `kustomize build`.
  * **module**: a client-side resource that expresses the desired state implemented by
a Kustomize plugin. Modules are analogous to server-side custom resource instances.
      * **Built-in module**: a module whose implementation is provided by Kustomize itself.
      * **Extensions module**: a module whose implementation is provided by a plugin.
  * **plugin/function**: a Starlark or container function that complies with
the existing [Configuration Functions Specification](https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md). Functions are analogous to server-side controllers.
Although they power the new API, functions are not the center of the user experience, the same way
controller code is not the focus of (or even referred to in) server-side resource expressions.
  * **module definition**: an OpenAPIv3 schema describing a module. Analogous
  to a server-side CRD.

### API schema

Example showing all fields:

```yaml
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Composition

# `modulesFrom` imports module lists from other Compositions.
# The target Compositions' own imports, overrides and reorderings are applied before it is imported.
modulesFrom:
- path: ../app/composition.yaml
  importMode: prepend # this is the default; append is also possible

# `moduleOverrides` allows fields of modules imported via `modulesFrom` to be changed before execution.
# Entries are treated as strategic merge patches.
moduleOverrides:
- apiVersion: example.com/v1
  kind: JavaApplication
  metadata:
    name: my-app
  spec:
    application: team/my-app
    version: v3.1.3

# `modules` adds new modules to the list.
modules:
- apiVersion: example.com/v1
  kind: Prometheus
  provider:
    container:
      image: example/prometheus:v1.0.2
  metadata:
    name: metrics-injector

# `moduleOrder` allows advanced users to reorder the merged modules list explicitly.
# modules can be referred to by name alone as long as names are unique.
moduleOrder:
  - name: my-app
  - name: logger
  - name: metrics
```

<details>
  <summary>Full schema</summary>

```yaml
definitions:
  module:
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
  modulesFrom:
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
  modules:
    type: array
    items:
      "$ref": "#/definitions/module"
  moduleOverrides:
    type: array
    items:
      "$ref": "#/definitions/module"
  moduleOrder:
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

### Built-in modules

The Composition implementation itself should provide a select few modules that
implement essential functionality, notably functionality that is available through
Kustomization built-ins today.

Essential starter set:

1. `kind: StaticResources`: a way to load custom-written Kubernetes resources from the
filesystem for processing. Equivalent to the `resources` field in Kustomization, but
not capable of fetching from remote sources.
1. `kind: Kustomize`: support basic transformations via fields such as `spec.commonLabels`,
as well as processing a Kustomization.yaml referred to by path.

The following example shows how this core Kustomize functionality would be expressed
as modules that fit in elegantly alongside plugin-driven modules:

```yaml
...
modules:
- metadata:
    name: local-resources
  kind: StaticResources
  apiVersion: kustomize.config.k8s.io/v1alpha1
  spec:
    paths:
    - ../resources
- metadata:
    name: prod-customizations
  kind: Kustomize
  apiVersion: kustomize.config.k8s.io/v1alpha1
  spec:
    commonLabels:
      env: production
    namespace: my-app-prod
```

Other possibilities include:

- A validation module.
- A Helm module that would enable existing Helm charts to be used as a starting point
for interoperable customization. It would do the equivalent of running `helm template`.

### Function invocation

To work with Composition, functions need to comply with the existing [Configuration Functions Specification](https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md).

Function invocation machinery will be built using the existing `sigs.k8s.io/kustomize/kyaml/fn/runtime` package.
It will work largely the same way function invocation does in Kustomization, with
the following notable exceptions:
* The function to be invoked will be specified in a new reserved field called `provider`
rather than being embedded in module metadata.
* The `provider` field will only support the `container.image` and `starlark.path` options,
at least at first.
* Composition will _always_ use the `ResourceList` input format when
invoking functions, and will _always_ include the `functionConfig` field.
That field will contain the module.

### Composition evaluation

Composition topologically sorts imported modules with its own modules
and then runs the combined list sequentially. This means that arbitrary depths of
imports always result in a single consolidated Composition prior to execution.

For use cases that require resources to be evaluated eagerly before further processing,
for example diamond-shaped configuration, the built-in `StaticResources` module can be given
a Composition to render into resources.

### Function packaging and distribution

The canonical way to build functionality for Composition will be to publish
functions as container images that are semantically versioned using tags,
and to include the image registry in Composition's allowlist. The format of
that list is TBD, but ideally it can be provided at compile time for organizational
use as well as at runtime.

Function authors will be encouraged to label their containers with module definitions.
Doing so will unlock additional functionality, such as making the `provider` field
optional in cases where a module's APIVersion+Kind resolve uniquely within the registry.

Ad hoc functions can be written in Starlark and stored locally with the
Composition. If remote Starlark functions are supported in the future,
a similar "registry" model will be needed for them.

### Test Plan

Testing can be done purely on the client-side without the need for a cluster.

### Graduation Criteria

Not yet required.

### Upgrade / Downgrade Strategy

NA -- not part of the cluster

### Version Skew Strategy

NA -- not part of the cluster

## Production Readiness Review Questionnaire

NA -- not part of the cluster

### Feature Enablement and Rollback

NA -- distributed as a client-side binary

### Rollout, Upgrade and Rollback Planning

NA -- distributed as a client-side binary

### Monitoring Requirements

NA -- distributed as a client-side binary

### Dependencies

NA -- distributed as a client-side binary

### Scalability

NA -- distributed as a client-side binary

### Troubleshooting

NA -- distributed as a client-side binary


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

* The new API provides another way to run plugins, and may be confusing to users
trying to decide which to use. Creating separate APIs for Deployment, StatefulSet,
DaemonSet have similar drawbacks.

* Since this new API is oriented entirely around plugins, it will likely lead to
workflows in which a considerable number of plugins must be invoked to render
the resource list. Especially for container plugins, this may not be performant.
Although the same issue would exist in today's plugins via Kustomization,
it may become more apparent if plugins get a substantial increase
in ease of use.

## Alternatives

* Implement these capabilities in the `Kustomization` API. All of the fields proposed
in the new API could instead be added as top-level fields, or grouped fields, in
`Kustomization`. However, the result would not be very elegant. In particular,
one of the primary proposed features--enabling module configuration to be overridden--requires
a completely different evaluation model. Namely, a topological sort is applied to the
entire tree of modules prior to evaluation, as opposed to Kustomization's current
model of executing generators, transformers and plugins in each level in isolation.
Combining these two mental models within a single API would be difficult and likely
lead to considerable confusion among users. Adding these features to `Kustomization`
would also preclude excluding them from workflows like `kubectl apply -k` unless they
permanently require extra enablement flags.
* Implement this as an independent tool. This is certainly a possibility, but
this proposal is effectively an improved way of exposing existing Kustomize features.
As such, it would be great if the Kustomize community could leverage it directly.
