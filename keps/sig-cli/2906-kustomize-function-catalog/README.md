<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [x] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [x] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [x] **Fill out this file as best you can.**
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
# KEP-2906: Kustomize Plugin Catalog

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
  - [Key terminology](#key-terminology)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
    - [Story 4](#story-4)
    - [Story 5](#story-5)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Determining the Plugin Provider to Execute](#determining-the-plugin-provider-to-execute)
  - [Use of OCI Artifacts](#use-of-oci-artifacts)
  - [OCI Artifacts](#oci-artifacts)
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
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

Introduce a new API (`kind`) that will provide a mechanism to improve distribution and discovery of Kustomize plugins, for use with `Kustomization`, `Components`, and `Composition` resources.

This new API will provide a standardized way to define a collection of one or more Kustomize plugins, as well as supporting KRM-style configuration resources, that can be consumed by Kustomize in order to automate the use of plugins and eliminate manual out-of-band discovery and installation steps, regardless of the packaging format. All Kustomize configuration objects (i.e. Kustomization, Component and Composition) will support plugin source configuration via this new kind. Ideally, we would like the new API to become a standard that other KRM-style transformer orchestrators such as KPT can adopt as well.

### Key terminology

*KRM*: [Kubernetes Resource Model](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/resource-management.md)

 *Plugin*: User-authored generator, transformer or validator (refers to both the "provider" program that implements it and the "config" YAML required to use it).

 *Plugin provider*: The program that implements the plugin (e.g. container, script, Go program). Analogous to the controller for a custom resource.

 *Plugin config*: The KRM-style YAML document that declares the desired state the plugin implements well as the plugin provider to execute and the specification to follow in doing so. Analogous to a custom resource object.
 

## Motivation

The use of Kustomize plugins today is cumbersome, both in terms of discovery and the use of plugins within a Kustomization. The introduction of the `Composition` API will improve plugin workflows, but challenges remain surrounding plugin distribution and discovery. This KEP is motivated by this need to improve the distribution and discovery of plugins, for use in `Composition` or other Kustomize resources. 

In order to use Kustomize plugins today, an end user must explicitly provide a reference to the plugin implementation. For example, consider the use of a plugin with a Kustomization. First, the user would define a resource configuration as follows:

```yaml
apiVersion: team.example.com/v1alpha1
kind: HTTPLoadBalancer
metadata:
  name: lb
  annotations:
    config.kubernetes.io/function: |
      container:
        image: docker.example.com/kustomize-modules/lb:v0.1.1
spec:
  selector:
    matchLabels:
      app: nginx-example
  expose:
    serviceName: nginx
    port: 80
```

This is then referenced from a Kustomization in the following way:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- lb.yaml
```

This explicit configuration requires the user to include two separate pieces of configuration to use the function: 
* they must provide the API Version and Kind
* they must provide an explicit reference to the Docker container that will be used 

When using a container based plugin provider with the `Composition` API, the user must still specify this information:

```yaml
# app/composition.yaml
kind: Composition

modules:
# generate resources for a Java application
- apiVersion: example.com/v1
  kind: JavaApplication
  provider: {container: {image: example/module_providers/java:v1.0.0}}
  spec:
    application: team/my-app
    version: v1.0
```

Once the explicit container reference is provided, Kustomize is able to download and run this image as part of the Kustomize build step, by invoking the user installed Docker client and leveraging local images or OCI registries. In addition to container based plugins, ad-hoc functions can also currently be written using the [Starlark programming language](https://github.com/bazelbuild/starlark) and other non-container based mechanisms. Unlike container based plugins, these plugins do not currently have an associated registry concept and they must be stored locally. Discovery and installation of these providers are both currently left to the users. 

In addition to these user defined plugins, this KEP is also partially motivated by a need to change how Kustomize provides officially supported functionality. Currently, to support a given piece of functionality officially a built-in function must be created (and typically also added to the Kustomization API). Some of these features would be better implemented as extensions for security reasons or to limit the dependency graph of Kustomize and the integration with kubectl. No mechanism currently exists, however, to support official distribution of these capabilities, so they are instead built into Kustomize.

As an example, the Helm functionality built into Kustomize currently relies on the user to install Helm as a separate step. This in turn requires the use of special flags on invocation to use, for security reasons. If the Helm integration was instead built and distributed as a container based plugin, the implementation could instead use the Helm Go packages and be built independently of Kustomize and distributed to users of Kustomize through an official channel. 

### Goals

Develop an API (`kind`) for Kustomize that is focused on plugin provider discovery, as well as guidelines and recommendations for distribution of plugin providers and associated resources, such as schema definitions.

A successful implementation of this API should have the following characteristics:

1. Plugin based workflows are driven by seamless invocation of sets of plugin providers without individual out-of-band discovery or installation steps for specific providers.
1. The new API is integrated with the existing Kustomize tool (i.e. `kustomize build`) through references provided in Kustomization, Component, and Composition resources.
1. Eligible Kustomize functionality could be extracted and distributed as official extensions. This won't be completed as part of this KEP but the required changes to support this will be implemented. 

### Non-Goals

1. Support anything other than KRM-style plugins that follow the [functions spec](https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md)
1. Directly implement capabilities to publish plugin providers or other resources to OCI registries 

## Proposal

In order to standardize plugin discovery, introduce a `Catalog` API `kind` recognized by `kustomize build`. One or more `Catalog` resources can be referenced from any Kustomize kind as either a local file or remote reference available as from an HTTP(s) endpoint or as an [OCI artifact](https://github.com/opencontainers/artifacts).

A `Catalog` will contain a collection of one or more modules that can be used with a Kustomize resource. 

A minimal example is shown below:

```yaml
apiVersion: kustomize.io/v1
kind: Catalog
metadata: 
  name: "example-co-plugins"
spec: 
  modules: 
  - apiVersion: example.com/v1
    kind: JavaApplication
    description: "A Kustomize plugin provider that can handle Java apps"
    provider: 
      container: 
        image: example/module_providers/java:v1.0.0
```

This will enable a users to use plugin providers with provider information omitted, such as:

```yaml
# app/kustomization.yaml
kind: Kustomization
catalogs:
  - https://example.com/plugins/catalog.yaml
resources:
- javaapp.yaml
```

```yaml
# app/javaapp.yaml
apiVersion: example.com/v1
kind: JavaApplication
spec:
  application: team/my-app
  version: v1.0
```

When this Kustomization is processed by `kustomize build`, the referenced catalog (or catalogs) will be used to locate a plugin provider that supports the apiVersion `example.com/v1` and kind `JavaApplication`. If found in one of the referenced catalogs, kustomize can determine the provider configuration without the need for the user to specify it in the kustomization resources directly. The catalogs will be searched in order specified. If more than one catalog defines the target apiVersion and kind, the first will be selected. 

In addition to the new catalog kind, `kustomize build` will accept a repeatable flag `--trusted-plugin-catalog=""`. When present, this flag instructs `kustomize build` to automatically fetch and execute plugins that are defined by the catalog and referenced within the Kustomization, Component or Composition. When a resource is processed by `kustomize build` and a catalog is referenced but not specified using the `--trusted-plugin-catalog=""` flag, an error will occur. Kustomize will provide a built in Catalog for supporting official extensions, published to a well publicized endpoint. This catalog will _NOT_ require the user to explicitly trust it. Users can provide the `apiVersion` and `kind` of the official extensions in kustomize resource and these will be resolved by the official catalog.

In addition to container based plugin providers, the `Catalog` will support discovery of Starlark and Exec based plugin providers, via an HTTP(s), Git, or OCI reference as illustrated below: 

```yaml
apiVersion: kustomize.io/v1
kind: Catalog
metadata: 
  name: "example-co-plugins"
spec: 
  modules: 
  - apiVersion: example.com/v1
    kind: GroovyApplication
    description: "A Kustomize plugin provider that can handle groovy apps"
    provider:  
      starlark: https://example.co/module_providers/starlark-func:v1.0.0
```

This concept can be extended later to support additional plugin provider packaging, but is out of scope for the current proposal.

When HTTP(s) references are used, the HTTP(s) endpoint must support anonymous access for reading resources. Resources will be expected to be stored as a single file, such as `catalog.yaml`. Git or OCI references can be authenticated or anonymous, and will use appropriate configuration from the users file-system. 

 
### User Stories (Optional)

#### Story 1

As a platform developer at enterprise company Example Co, I want to publish a catalog of Kustomize plugin providers that represent custom capabilities important for our internal Kubernetes platform. I have built and published several Kustomize plugins, packaged as Docker images, to our internal Docker registry: `docker.example.co`.

To do this, I build a new `Catalog` API resource: 

```yaml
# catalog.yaml
apiVersion: kustomize.io/v1
kind: Catalog
metadata: 
  name: "example-co-plugins"
spec: 
  modules: 
  - apiVersion: example.com/v1
    kind: JavaApplication
    description: "A Kustomize plugin provider that can handle Java apps"
    provider: 
      container: 
        image: docker.example.co/plugins/java:v1.0.0
  - apiVersion: example.com/v1
    kind: Logger
    description: "A Kustomize plugin provider adds our bespoke logging"
    provider: 
      container: 
        image: docker.example.co/plugins/logger:v1.0.0
  - apiVersion: example.com/v1
    kind: SecretSidecar
    description: "A Kustomize plugin provider adds our bespoke secret sidecar"
    provider: 
      container: 
        image: docker.example.co/plugins/secrets:v1.0.0
```

I then publish this catalog to https://example.co/kustomize/catalog.yaml, for use by Example Co Kustomize users.

#### Story 2

As an application developer at Example Co, I want to use the published Example Co Catalog in the `Kustomization` for my application, after locating the published location of the catalog.

While building my `Kustomization`, I don't want to care about the provider configuration and I want Kustomize to figure things out for me, based on the catalog.


```yaml
# app/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
catalogs:
  - https://example.co/kustomize/catalog.yaml
resources:
- java.yaml
- secrets.yaml
```

```yaml
# app/java.yaml
apiVersion: example.com/v1
kind: JavaApplication
spec:
  application: team/my-app
  version: v1.0
```

```yaml
# app/secrets.yaml
apiVersion: example.com/v1
kind: SecretSidecar
spec:
  key: my.secret.value
  path: /etc/secrets
```

I then run `kustomize build app/ --trusted-plugin-catalog=https://example.co/kustomize/catalog.yaml`. 

When this command is run, Kustomize detects the use of `example.com/v1/JavaApplication` and `example.com/v1/SecretSidecar`. As these are not built in transformers and there was no explicit provider configuration specified, Kustomize will check for any referenced catalogs. It will see that I have specified `https://example.co/kustomize/catalog.yaml` and allowed it as a trusted plugin catalog. It will then fetch the catalog and attempt to resolve these two transformers. It will match the specified apiVersion and Kinds with entries in the catalog and utilize the referenced docker images for the provider configuration.

#### Story 3

As an application developer at Example Co, I want to use the published Example Co Catalog in the `Composition` for my application, but I want to pin to a specific version of the `SecretSidecar` module . While building my `Composition`, I don't want to care about the provider configuration, except for `SecretSidecar` and I want Kustomize to figure the rest of the providers out for me, based on the catalog.

```yaml
# app/composition.yaml
kind: Composition
catalogs:
  - https://example.co/kustomize/catalog.yaml
modules:
# generate resources for a Java application
- apiVersion: example.com/v1
  kind: JavaApplication
  spec:
    application: team/my-app
    version: v1.0
- apiVersion: example.com/v1
  kind: Logger
  metadata:
    name: my-logger
  spec:
    logPath: /var/logs
- apiVersion: example.com/v1
  kind: SecretSidecar
  provider: {container: {image: docker.example.co/module_providers/secrets:v0.9.0}}
  metadata:
    name: my-secrets
  spec:
    key: my.secret.value
    path: /etc/secrets
```

Unlike the previous execution of Kustomize, Kustomize will not use the catalog to resolve the `SecretSidecar`, as the provider configuration was specified. 

#### Story 4

As a platform operator at Example Co, I want to provide an easier mechanism to enable use of the official Example Co Catalog. To do this, I compile a version of Kustomize with a built in reference to the Example Co official catalog. Example Co users can then simply reference our plugins, without specifying the catalog:

```yaml
# app/composition.yaml
kind: Composition
modules:
# generate resources for a Java application
- apiVersion: example.com/v1
  kind: JavaApplication
  spec:
    application: team/my-app
    version: v1.0
- apiVersion: example.com/v1
  kind: Logger
  spec:
    logPath: /var/logs
- apiVersion: example.com/v1
  kind: SecretSidecar
  spec:
    key: my.secret.value
    path: /etc/secrets
```

#### Story 5

As a Kustomize developer, I want to build an `official` Helm extension module and publish it via the Kustomize official extension catalog. I build and publish the extension to the official Kustomize `gcr.io` project (or alternative, such as Github Package Registry). I then update the official Kustomize extension catalog:

```yaml
apiVersion: kustomize.io/v1
kind: Catalog
metadata: 
  name: "official-kustomize-plugins"
spec: 
  modules: 
  - apiVersion: kustomize.io/v1
    kind: Helm
    description: "A Kustomize plugin that can handle Helm charts"
    provider: 
      container: 
        image:  k8s.gcr.io/kustomize/helm-plugin:v1.0.0
```

This official extension will be embedded within the Kustomize binary and require subsequent releases to Kustomize when official extensions are updated. 

Alternatively, this could be published as an external resource that can be pulled by Kustomize as would any other catalog. This would decouple the release cadence of Kustomize and the official extensions, but would introduce extra latency for the end user.

### Notes/Constraints/Caveats (Optional)

Not all registries currently support OCI Artifacts, which will constrain the use of that capability. Most major cloud providers and several open source projects, however, support this:

* https://aws.amazon.com/blogs/containers/oci-artifact-support-in-amazon-ecr/
* https://cloud.google.com/artifact-registry
* https://docs.microsoft.com/en-us/azure/container-registry/container-registry-oci-artifacts
* https://github.com/features/packages
* https://github.com/goharbor/harbor/releases/tag/v2.0.0
* https://github.com/distribution/distribution

This proposal does not suggest adding any OCI artifact publishing capabilities to Kustomize, and would instead rely upon the [ORAS](https://oras.land) project to handle publishing and fetching of artifacts for now. Built in capabilities could be added by including ORAS as a library, but an analysis of the dependencies introduced will be needed.

### Risks and Mitigations

This proposal introduces extension capabilities to Kustomize that may expose users to external content. As with `Composition`, it must be made clear to users that use of a `Catalog` may represent untrusted/unvalidated content and they should only use `Catalogs` that they trust. When `Catalog` and other resources are stored as OCI artifacts, users can get extra assurance of content by using `digest` references. Additionally, the [cosign](https://github.com/sigstore/cosign) project could be used to provide signing and validation capabilities. The guidance around executing plugins, as outlined in the [Composition](../2290-kustomize-plugin-composition/README.md) KEP remain applicable when combined with `Catalog` resources.

## Design Details

The catalog kind will have a YAML representation. This representation will contain metadata about the catalog, such as name labels, as well as a collection of plugins. Each plugin entry will contain an apiVersion and kind, along with one or more references to a plugin provider, as well as an optional information, such as Open API v3 definitions. Provider references can contain https, git, or OCI references. Additionally, a provider can declare that it requires the use of network or storage mounts, which would otherwise be prohibited. The use of these requires [additional flags](https://github.com/kubernetes-sigs/kustomize/blob/1e1b9b484a836714b57b25c2cd47dda1780610e7/api/types/pluginrestrictions.go#L51-L55) on the command line. 

When using OCI references, either a tag or digest reference can be provided. Exec plugins should include an sha256 hash for verification purposes, although this can also be done by using an OCI digest reference. When a hash verification fails, Kustomize will emit an error to inform the user. When a hash is not provided for verification, Kustomize will emit a warning to inform the user that validation could not be performed. 

A complete representation is shown below:

```yaml
apiVersion: kustomize.io/v1
kind: Catalog
metadata: 
  name: "example-co-plugins"
  labels:
    author: ExampleCo
spec: 
  modules: 
  - apiVersion: example.com/v1
    kind: JavaApplication
    description: "A Kustomize plugin provider that can handle Java apps"
    definition: "https://example.com/java/definition"
    provider: 
      container: 
        image: example/module_providers/java:v1.0.0
        requireNetwork: true
        requireFilesystem: true
      starlark:
        path: oci://docker.example.com/javaapp/provider:v1.0.0
      exec:
        - os: darwin
          arch: arm64
          path: https://example.com/java
          sha256: [a hash] 
        - os: darwin
          arch: amd64
          path: https://example.com/java
          sha256: [a hash]
  - apiVersion: example.com/v1
    kind: SiderCarInjector
    description: "A Kustomize plugin provider that injects our custom sidecar"
    provider: 
      starlark:
        path: https://github.example.com/functions-catalog.git/providers/starlark@sidecar/v1.0
```

### Determining the Plugin Provider to Execute

When a `Composition`, or other Kustomize resource that utilizes plugins is loaded, Kustomize will leverage the `Catalog` to determine plugin providers that should be run. The order in which plugins are resolved is determined by the type of resource being processed, and will be clearly addressed in user facing documentation. 

For a `Kustomization`, a catalog reference is local to a given layer and individual layers could reference different catalog or provider versions. As the layer is processed, Kustomize will evaluate any catalog references and select the appropriate version based on the referenced catalog. 

For a `Composition`, on the other hand, `Kustomize` will consolidate the modules defined in the `Composition` and it's imports into a finalized list of modules. Next, it will consolidate the list of `Catalog`s in the module and it's imports to build a finalized list of `Catalog`s. Each of the `Catalog` resources will be fetched and used to build a unified catalog representation. When two catalogs define the same module, the first definition will be used.

Once the module list and the catalogs for the resolved composition have been generated, the following steps will be performed in order to determine the `provider` to execute:

* If a KRM-style resource includes the `provider` field, that will be used
* The `provider` field will continue to support `container.image`, `starlark.path`, and `exec.path` options, for the time being. This short-circuits `Catalog` for the given plugin, and the flags currently required for plugin execution (`--enable-alpha-plugins`, `--enable-exec`) will continue to be required in this case.
* If the `provider` field is absent, the configured `Catalog` resources will be used to determine the provider to execute, based on the `kind` and `apiVersion` fields of the resource specification. If an official catalog has been created for Kustomize, it will be checked first.
* If there is no matching module, the processing of the resource will result in an error.

### Use of OCI Artifacts

### OCI Artifacts

While this proposal is largely focused on the introduction of the new Catalog `kind`, the introduction of this kind enables additional distribution and trust mechanisms for non container based plugin providers and associated resources, like Open API v3 schemas through the use of OCI Artifacts. 

When OCI references are used, either a `tag` or `digest` reference can be used. This proposal does not address publishing plugins or the `Catalog` resource to an OCI registry but will define the following media types, based on guidance in the [OCI Artifacts](https://github.com/opencontainers/artifacts/blob/master/artifact-authors.md#defining-a-unique-artifact-type) documentation:

| Description                 | Media Type                                                 | 
|-----------------------------|------------------------------------------------------------|
| Kustomize Catalog           | application/vnd.cncf.kubernetes.krm-plugin-catalog.layer.v1+yaml      |
| Kustomize Plugin Definition | application/vnd.cncf.kubernetes.krm-plugin-definition.layer.v1+yaml   |
| Kustomize Plugin (Starlark) | application/vnd.cncf.kubernetes.krm-plugin-provider-starlark.layer.v1 |
| Kustomize Plugin (Exec)     | application/vnd.cncf.kubernetes.krm-plugin-provider-starlark.layer.v1 |

The [ORAS](https://oras.land) library and CLI can be used to publish these artifacts and can be used to build specific publishing tooling, but Kustomize will not be changed to add publishing capabilities. Instead, appropriate user documentation and examples will be provided. 

In order to support pulling these resources, the ORAS library could be included as a dependency to support automatic fetching of OCI artifacts, however this will introduce a number of dependencies and could be undesirable. Alternatively, the ORAS binary can be installed by the user and used as the locally installed Docker client is used today. When an OCI artifact is referenced and fetched using ORAS, it will be stored locally within the file-system and can then be used within the `kustomize build` step. 

Kustomize plugin providers that are packaged as OCI images will continue to use the existing OCI media types. 

While out of scope of this KEP, the use of OCI artifacts enables additional verification use cases, like the signing and verification of plugin providers, definitions, and the catalog itself.

### Test Plan

 Kustomize already has a test harness capable of running plugins, so this will be leveraged. Unit tests and end to end tests related to plugin and catalog retrieval, evaluation, and trust will be implemented, covering major workflows such as:
 * Retrieval of catalog resources
 * Resolving plug-in configuration references
 * Multiple catalog vs single catalog references
 * Use of containerized and non-containerized plugins 

### Graduation Criteria

TBD

#### Alpha

- Feature integrated with the `kustomize build` command, and all currently implemented Kustomize kinds.
- Initial e2e tests completed and enabled
- Container based plugin provider support.
- HTTPs and Git based catalog storage

#### Beta

- Gather feedback from developers and surveys
- Starlark and Exec plugin support
- OCI artifact support for catalog and provider distribution

#### GA

- TBD


#### Deprecation

N/A 

### Upgrade / Downgrade Strategy

NA -- not part of the cluster

### Version Skew Strategy

NA -- not part of the cluster

## Production Readiness Review Questionnaire

NA -- not part of the cluster

### Feature Enablement and Rollback

This enhancement will only be available in the standalone kustomize command while it is in the alpha state. 

Integration and rollout to kubectl will not occur until the beta phase. 

### Rollout, Upgrade and Rollback Planning

NA -- distributed as a client-side binary

### Monitoring Requirements

NA -- distributed as a client-side binary

### Dependencies

The ability for kustomize, and by extension kubectl, to pull plugin providers specified in the catalog may introduce additional compile time or runtime dependencies. For example, pulling OCI artifacts is a net new capability and will either require the use of something like ORAS as a compile-time library dependency, or as a run-time client dependency like Docker. This will be examined as part of the work to move this enhancement to the beta state and a decision will be made based on an analysis of the dependencies that would be introduced.

This enhancement also introduces a dependency on catalogs and providers that may not be on the users local environment. When a catalog is unavailable and required for successful execution of kustomize build, an error will occur. If the catalog is referenced, but not actually required for the execution of kustomize build (i.e. no plugins are actually used in a given kustomization or composition), the operation will complete successfully.

While most testing can occur without community infrastructure, we will require a place to publish and host catalog resources, along with plugins, for testing purposes. We will investigate the use of Github registries or other Kubernetes community infrastructure to enable this. 

### Scalability

End users will encounter a cold-start period related to pulling the catalog resources and associated providers, if they do not exist locally on the file-system.

### Troubleshooting

NA -- distributed as a client-side binary

## Implementation History

2021-08-XX: Proposal submitted https://github.com/kubernetes/enhancements/pull/XXXX

## Drawbacks

The discovery mechanism imposed by this KEP does expose an additional layer of indirection/complexity for plugin users. Additionally, this will introduce some dependence on community infrastructure and may introduce some operational burden for plugin authors.  

## Alternatives

* Do nothing - as outlined in the drawbacks section, users can continue to leverage explicit references to providers, at the cost of module discoverability and ease of use.
* An alternative tool could be developed outside of kustomize that supports the catalog resource and installation of plugin providers, much like Krew does for kubectl plugins. Such a tool would provide a less integrated experience and would require users to execute steps outside of the `kustomize build` flow and would need to modify the local Kustomize resources to add explicit provider configuration.

## Infrastructure Needed (Optional)

When Kustomize publishes an official `Catalog` and any associated plugins, the Kubernetes community GCR and GAR (if available) infrastructure will be needed to host resources.  
