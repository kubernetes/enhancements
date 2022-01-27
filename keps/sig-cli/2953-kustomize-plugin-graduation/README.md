[Catalog KEP]: https://github.com/kubernetes/enhancements/pull/2908
[Composition KEP]: /keps/sig-cli/2299-kustomize-plugin-composition
[Generators and Transformers KEP]: /keps/sig-cli/993-kustomize-generators-transformers/README.md
[KRM Functions Specification]:https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md

<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [x] **Create an issue in kubernetes/enhancements**
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
- [x] **Create a PR for this KEP.**
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
# KEP-2953: Kustomize plugin graduation

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
  - [Key terminology](#key-terminology)
- [Motivation](#motivation)
  - [Legacy plugins](#legacy-plugins)
  - [KRM Function plugins](#krm-function-plugins)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Plugin execution restriction](#plugin-execution-restriction)
  - [Supported runtimes](#supported-runtimes)
  - [Plugin developer SDK](#plugin-developer-sdk)
  - [Plugin configuration](#plugin-configuration)
  - [Legacy plugin migration](#legacy-plugin-migration)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Rollout Plan](#rollout-plan)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
- [Appendix](#appendix)
  - [Current Plugin Compatibility Grid](#current-plugin-compatibility-grid)
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

This KEP proposes converging Kustomize's various alpha extension mechanisms into a single KRM-driven feature that has an enhanced story around plugin distribution, discovery and trust. A central goal of this KEP is to equip Kustomize with an extension story that will be acceptable for full inclusion in `kubectl kustomize`. This KEP provides an overarching vision that ties together several existing proposals that go into more detail on specific features this KEP considers important to plugin graduation.



### Key terminology

*KRM*: [Kubernetes Resource Model](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/resource-management.md)

*Plugin*: User-authored generator, transformer or validator (refers to both the "provider" program that implements it and the "config" YAML required to use it).

*Plugin provider*: The program that implements the plugin (e.g. container, script, Go program). Analogous to the controller for a custom resource.

*Plugin config*: The KRM-style YAML document that declares the desired state the plugin implements. In some plugin styles, it must also specify the plugin provider to execute and the specification to follow in doing so. Analogous to a custom resource object.

## Motivation

Kustomize currently supports two distinct categories of plugins, both of which are in alpha. These inconsistent enduring alphas cause [confusion and frustration](https://github.com/kubernetes-sigs/kustomize/issues/2721) for users and are a source of complexity for maintainers. See also [Plugin Compatibility Grid](#current-plugin-compatibility-grid).

### Legacy plugins

These are Kustomize's original plugin style, which was first proposed in the [Generators and Transformers KEP].  

**kustomize/kubectl support**: These plugins work in recent versions of `kubectl kustomize` if the `--enable-alpha-plugins` is used.

**Plugin authoring**: This style supports writing plugins as [executable scripts](https://kubectl.docs.kubernetes.io/guides/extending_kustomize/#exec-plugins) or as [Go plugins](https://kubectl.docs.kubernetes.io/guides/extending_kustomize/#go-plugins). Documentation and examples are available but not comprehensive and are somewhat out of date. No SDK is available.

**Plugin distribution/discovery**: No mechanism exists or has been proposed.

**Provider lookup**: Plugin config + environment + convention. The APIVersion and Kind are used to form a path suffix (e.g. `/someteam.example.com/v1/myplugin/`) and filename (e.g. `MyPlugin` or `MyPlugin.so`). A series of environment variables are consulted in an attempt to find a directory in which that path exists on disk (e.g. `$KUSTOMIZE_PLUGIN_HOME/someteam.example.com/v1/myplugin/`).

**Provider distribution**: End users must each manually install all the plugins their Kustomization requires at the expected locations, and may need to configure their environment accordingly. This process is always completely out of band.

**Plugin config**: A KRM-style object in which the APIVersion and Kind are interpreted as hints for looking up the plugin provider. For exec plugins, the config has additional reserved fields `argsFromFile` and `argsOneLiner` that are used to change the arguments used in invoking the provider. 

**Plugin provider trust model:** Though the plugin root may be arbitrary, the provider lookup requires the executable to be located at a very specific subpath that is unlikely to exist by accident. Plugin execution is also gated by an alpha flag. The plugin itself is not sandboxed in any way. The user is deemed to have accepted the risks by installing the provider and using the flag in their Kustomize invocation.

**Additional notes**:

* [Go plugins are really hard to use](https://kubectl.docs.kubernetes.io/guides/extending_kustomize/goplugincaveats/) and require _end users_ to compile both the plugin and Kustomize itself, which sometimes still yields confusing incompatibilities.
* The provider discovery model means that providers must be 1:1 with GVKs. For example, if a provider supports multiple versions of a given virtual resource, the end user need to install separate copies of it for each one.
* End users necessarily need to know about and manage provider versions (plugin authors cannot implement an upgrade story).



### KRM Function plugins

These are a newer plugin style that is not covered by the Kustomize documentation, but follows a more mature open specification: the [KRM Functions Specification](https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md), which is also used by kpt. Docs are in the Kustomize repo and on the kpt site, but not the Kustomize site.

**kustomize/kubectl support**: The container subtype works in recent versions of `kubectl kustomize` if the `--enable-alpha-plugins` flag is used. The other subtypes work only in standalone `kustomize`, and only if additional flags (`--enable-exec` / `--enable-star`) are provided as well.

**Plugin authoring:** Plugins can be written in any language and packaged as containers or executable scripts, or they can be [written in Starlark](https://github.com/kubernetes-sigs/kustomize/tree/master/plugin/someteam.example.com/v1/starlarkmixer) and executed directly. For Go, an extensive alpha SDK is available in Kustomize's kyaml module: the [functions framework package](https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml/fn/framework). Since these plugins follow the functions specification, any SDK developed for that spec can also be used (e.g. [kpt's Typescript SDK](https://kpt.dev/book/05-developing-functions/03-developing-in-Typescript)).

**Plugin distribution/discovery**: The [Catalog KEP] proposes a mechanism for declaratively specifying groups of KRM Function style plugins. This could also be used to drive discovery, e.g. via an official Kustomize, SIG-CLI or company-published catalog.

**Provider lookup**: The location of the provider must be explicitly declared in an annotation on the plugin config resource. For exec and starlark, paths on the user's machine can be used with no restrictions.

**Provider distribution**: Container-based plugins are distributed via docker registries. Starlark and exec plugins are distributed alongside the Kustomization (i.e. typically committed in git and referred to by relative path).

**Plugin config**: A KRM-style object bearing a [config.kubernetes.io/function annotation](https://github.com/kubernetes-sigs/kustomize/blob/f61b075d3bd670b7bcd5d58ce13e88a6f25977f2/cmd/config/docs/api-conventions/functions-impl.md#input) that points to the plugin provider and optionally includes provider invocation information. The [Catalog KEP] proposes making the annotation optional when a Catalog is used. The object itself otherwise resembles a standard custom resource: its APIVersion and Kind have no special significance and it has no top-level reserved fields.

**Plugin provider trust model:** Currently, the user assumes the risk associated with running the third party code by using the `--enable-alpha-plugins` flag. Starklark and exec plugins are not sandboxed in any way, and require additional `--enable-exec` /  `--enable-star` flags to invoke. Container-based plugins are run without disk or network access by default, but if a plugin requires it, the user must enable it both in the plugin config object (which they may not have written themselves) and on the command line via the `--mount`, `--network` and `--network-name` flags. The [Catalog KEP] proposes a way to trust a published set of plugins as well as a way for Kustomize to validate the integrity of that the providers it fetches.

### Goals

1. Kustomize's extension story MUST be consistent between standalone Kustomize and kustomize-in-kubectl. For example, if any flags are used to enable or configure Kustomize, the usage of those flags must be identical in both the standalone and embedded distributions.
1. Plugin configuration MUST be fully declarative.
1. Plugin usage MUST NOT require out-of-band imperative installation steps. 
1. Kustomize SHOULD support declarative configuration of a set of trusted plugins.  (See the [Catalog KEP])
1. Kustomize SHOULD provide an API that enables plugins to be orchestrated on equal footing with built-in transformers, including with respect to ordering. (See the [Composition KEP])
1. Kustomize SHOULD provide an API that facilitates sane layering in plugin-centric workflows where the plugin configuration itself must be manipulated. (See the [Composition KEP])
1. As a measure of plugin UX, Kustomize itself SHOULD be able to convert existing transformers (e.g. the Helm transformer) to "official" plugins without significantly degrading the user experience.

### Non-Goals

1. Make any change to the Configuration Functions Specification, a low-level spec shared with other orchestrators such as kpt. No such changes are required by this KEP.
1. Create a catalog of "official" plugins. Although such a thing would be a natural extension to this KEP, it is out of scope.
1. Convert currently compiled in transformers to plugins. Although this KEP should make this possible, the work to do so is out of scope.

## Proposal

This KEP proposes to deprecate the legacy style plugins in favour of the KRM Functions style, with some refinements. The KRM Functions style plugin mechanism is the most consistent with Kustomize's philosophy and most readily able to be adapted to meet the requirements outlined in the Goals section. It is also based on an open standard that has been adopted by at least one other tool (kpt), increasing the value of plugins built in this way.

This KEP proposes implementing both the [Composition KEP] and the [Catalog KEP] as a prerequisite for plugin GA in Kustomize. Those two KEPs solve key discovery, distribution and UX problems described in the goals section.

### Plugin execution restriction

The flags used to restrict plugin execution will be replaced with a new set that reflects the GA plugin feature set and is consistent between standalone and embedded Kustomize.

This KEP proposes using the concept of a "trusted plugin catalog" from the [Catalog KEP] as a convenient, concise way for end users to express their intent to execute a constrained set of plugins. With Catalog, the list of plugins being authorized is explicit and reviewable.

**Plugins from a user-specified trusted catalog**

When using plugins with a trusted catalog, only one flag will be required, regardless of the runtime type of the provider: `--trusted-plugin-catalog=""` .

This new repeatable flag will be introduced by the [Catalog KEP]. When that flag is present, plugins in the referenced catalogs will be executed without the need for any additional flags. The integrity of the plugin provider will be validated against the Catalog entry before execution. This all applies whether the providers are defined explicitly in the Kustomization (and the explicit entry matches the catalog) or looked up dynamically in the catalog.

If a containerized plugin needs network or disk access, its catalog entry MUST specify it. Redundant inline provider config cannot be used to grant it. When the catalog does specify it, no additional flags or config fields will be required to grant it the required access.

**Plugins from a compiled-in trusted catalog**

Optionally, it should be possible for a user to compile a bespoke distribution of Kustomize that embeds a trusted catalog, such that the plugins it references can be used without any additional flags on the end user's part. The target persona for this scenario is a platform maintainer at a large organization that distributes its own Kustomize internally. This approach could also be used in the future to extract Kustomize built-ins to plugins to reduce compiled in dependencies while minimizing UX impact.

**Plugins NOT in a trusted catalog**

*Recommended Option: no such thing*

Trusted catalogs are always required to use plugins; uncatalogued plugins will not be executed. Kustomizations that require ad-hoc plugins must be accompanied by a `Catalog` that the end user can reference locally, e.g. `kustomize build dir/ --trusted-plugin-catalog=dir/catalog.yaml`. There will be no additional plugin-related flags. The end user will be expected to inspect this catalog before trusting it, which must always be done explicitly on the command line.

A `kustomize edit generate-catalog` command will be implemented to streamline local plugin workflows. That command will take a Kustomization, collect all the uncatalogued plugins into a list, and build that list into the catalog resource format, using local references relative to the root Kustomization (plugins outside the root will result in an error). Optionally, this command can extract the provider references from the Kustomization, since they are most likely redundant once the catalog has been constructed. Optionally, this command can localize all required entries from referenced remote catalogs into the generated catalog.

While this does introduce some friction, it is far less than is required to install and use plugins today. It also has the advantage of making the plugins required highly reviewable in a gitops workflow, and of not requiring the large suite of CLI flags needed for plugin execution today.

*Alternative: many flags*

By default, plugins not listed in a trusted catalog will not be executed and an error will be printed. The providers for these plugins must be listed explicitly in each plugin config, and additional flags are always required to use them. Which flags are required depends on the provider type.

For plugins distributed as containers, the `--enable-container-plugins` flag will need to be provided. The containers will continue to be denied disk and network access by default. To enable access, the existing network/mount restriction flags will be retained but renamed to clarify what they govern:  `--plugin-container-networks=[STRING]` `--plugin-container-mounts=[STRING]`.

For plugins distributed as executables (typically committed in git along with the Kustomization), every provider the Kustomization needs must be explicitly named in a new flag:  `--trust-embedded-exec-plugins=base/plugin/transformer.sh,overlay/scripts/gen.sh`. By using this flag, the user demonstrates they are aware of what will be executed and accept the risk of running each one. To make the list less painful to construct, the error message thrown when this list is not provided or is incomplete will inform the user of which are missing.

**Additional constraints for exec providers**

The location of exec providers that are local will be subject Kustomize's load restrictor, i.e. the LoadRestrictionsRootOnly policy will apply to them by default (it does not today). If this is deemed insufficient, we could take it a step further and force the LoadRestrictionsRootOnly policy even when a different loader policy is provided by flag.

### Supported runtimes

Container and exec KRM Function plugins will continue to be supported. 

Starlark support will be deprecated and removed. This is because the dependencies the Starlark runtime brings in were deemed unacceptable for inclusion in kubectl, which would lead to a permanent discrepancy between kustomize distributions (an anti-goal of this KEP). No Starlark SDK has been developed to date, and Starlark scripts can still be used, like any other script, in the context of a container. This is the path kpt has chosen as well with [StarlarkRun](https://kpt.dev/book/05-developing-functions/04-executable-configuration?id=starlark-function).

Go plugins are currently only supported by legacy plugins, not by the KRM Function plugin style. Although these have advantages in terms of execution speed and ease of testing, they have many shortcomings and cannot provide the level of user experience one would expect from a "plugin". Most notably, they are not portable, requiring end users to undertake compilation steps themselves. For these reasons, this KEP does not recommend porting support to KRM Function plugins as part of plugin graduation. Nothing prevents it from being added in the future if there is sufficient interest. 

### Plugin developer SDK

The [functions framework package](https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml/fn/framework) will be promoted to stable along with or shortly after KRM Function plugin graduation to GA.

A plugin developer guide will be added to the Kustomize documentation, replacing the current plugin docs. The new documentation will include guidelines on how to use plugins to build a case for inclusion as a built in (similar to the process recommended for new kubectl features), and when/how new built-in transformers will be accepted. It will also extensively document Catalog use, and include guidance for migrating alpha plugins.

### Plugin configuration

All plugins must be configured using KRM-style custom-resource-like objects.

The `config.kubernetes.io/function` annotation that currently contains a nested JSON blob of plugin provider configuration will be graduated to a reserved top-level field called `provider` (ALTERNATIVE: it could be nested as `metadata.provider`, but that would make the object meta non-standard). That field will be made optional by the [Catalog KEP].

The `metadata.name` field is optional as long as the config's GVK uniquely identifies it. In Kustomization, it is not required unless the config is being handled as a resource. In Composition, it is always recommended, because transformer config is always handled as a resource before execution and uniqueness must be global across all Compositions in the tree. 

The `env` field for container plugins will be deprecated. In line with Kustomize's principles, plugin configuration should be passed in the KRM object and should not be affected by the execution environment. The network and storage mount options will be retained, as they are needed to support some common generator plugin use cases. Guidance on their appropriate use will be added to the plugin developer documentation.

### Legacy plugin migration

Although legacy plugins are still in alpha, the alpha has been around in standalone Kustomize for quite a while. Documentation on the migration process will be provided. See the [Rollout Plan](#rollout-plan) section for more detail on deprecation and removal of the existing alphas.

The exec plugin conversion process should be relatively straightforward: copy the provider in-tree and update the configuration that referred to it to have a Provider stanza pointing to the new location. `argsFromFile` and `argsOneLiner` can be straightforwardly converted to the `args` subfield the KRM Functions style already supports.

For authors who build plugins for distribution (as opposed to bespoke plugins where embedding within an end-user Kustomization is an appropriate solution), the recommendation will be to upgrade to the Catalog model. A Catalog guide will be added to the Kustomize website and will address the scenario of migrating a legacy style plugin.



### User Stories (Optional)

#### Story 1

As an end user, I want to develop a simple plugin for my particular Kustomization.

1. Write a script that adheres to the [KRM Functions Specification] and place it within the Kustomization root.
2. Write a corresponding plugin config and reference it from the Kustomization's generator or transformer field as appropriate.
3. Run `kustomize edit generate-catalog`
3. Run `kustomize build --trusted-plugin-catalog=catalog.yaml`

```bash
# Possible directory structure
.
⊢ Kustomization.yaml
⊢ input.yaml
⊢ reorder.yaml
⊢ plugin-catalog.yaml
∟ plugins
  ∟ sorter
    ⊢ sorter.rb
    ∟ sorter_test.rb
```

```yaml
# Example Kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- input.yaml
transformers:
- reorder.yaml
```

```yaml
# Example reorder.yaml
apiVersion: local-config.example.co/v1
kind: CustomSorter
provider:
  exec: 
  	path: plugins/sorter/sorter.rb
spec:
  order: NsGKVN
```

```yaml
# Example catalog.yaml (auto-generated)

apiVersion: kustomize.io/v1
kind: Catalog
metadata:
  name: "local-plugins"
spec:
  modules:
  - apiVersion: local-config.example.co/v1
    kind: CustomSorter
    provider:
      exec:
        - path: plugins/sorter/sorter.rb
          sha256: [a hash]
```


#### Story 2

As an end user, I want to use one or more plugins published by a third party I trust.

1. Write the config objects for the plugins you want to use, and reference them from your Kustomization's generators or transformers field as appropriate.
2. Run `kustomize build --trusted-plugin-catalog=https://catalog.kpt.dev/1.2.3.json`


```bash
# Possible directory structure
.
⊢ Kustomization.yaml
⊢ input.yaml
⊢ conditionally-add-annotations.yaml
∟ name-substring.yaml
```

```yaml
# Example Kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
catalogs:
- https://catalog.kpt.dev/1.2.3.json
resources:
- input.yaml
transformers:
- name-substring.yaml
- conditionally-add-annotations.yaml
```

```yaml
# Example conditionally-add-annotations.yaml
# Excerpted from https://catalog.kpt.dev/starlark/v0.2/
apiVersion: fn.kpt.dev/v1alpha1
kind: StarlarkRun
metadata:
  name: conditionally-add-annotations
params:
  toMatch:
    config.kubernetes.io/local-config: "true"
  toAdd:
    configmanagement.gke.io/managed: disabled
source: |
  def conditionallySetAnnotation(resources, toMatch, toAdd):
  ...
```

```yaml
# Example name-substring.yaml
# Excerpted from https://catalog.kpt.dev/ensure-name-substring/v0.1/
apiVersion: v1
kind: EnsureNameSubstring
metadata:
  name: my-fn-config
substring: dev-
```



#### Story 3

As a platform developer for a large organization, I want to provide internal users with a suite of trusted abstraction-based plugins.

1. Develop the plugins in accordance with the [KRM Functions Specification] and publish them to one or more supported locations.
2. Publish a Catalog (ideally using automation) to a supported location.
3. Build a bespoke Kustomize distribution for internal use, e.g. `DEFAULT_TRUSTED_CATALOGS=kustomize.example.co/catalog/1.1.json make kustomize` (that hypothetical env var is build-time-only)
4. Internal customers can then invoke `kustomize build` with no further flags and the plugins will be invoked as seamlessly as built-ins.

```yaml
# Example end-user Composition
kind: Composition
modules:
- apiVersion: kustomize.example.co/v1
  kind: JavaApplication
  metadata:
    name: my-app
  spec:
    application: team/my-app
    version: v1.0
- apiVersion: kustomize.example.co/v1
  kind: Logger
  logFormat: json
  collectPaths: ["/path/to/logs"]
```



### Notes/Constraints/Caveats (Optional)

Individual Kustomizations/Compositions can contain catalog references. These references are treated as informational regarding the dependencies of the Kustomization and will not result in Catalog trust for plugin execution purposes. If Kustomize encounters a plugin config with an unknown provider in a Kustomization/Composition with a catalog reference, it will emit an error message suggesting that the catalog in question needs to be trusted for the build to succeed.

### Risks and Mitigations

* A risk of the proposed ability to compile in a trusted catalog is that a user could be unaware that they are trusting any plugins when invoking Kustomize. This risk and attack vector are no different from the risk of using an arbitrarily modified version of Kustomize downloaded or built from a non-canonical source today.

* Even if the Catalog contains checksums Kustomize can use to validate providers prior to execution, there is a risk of MITM attack on the Catalog retrieval itself. As a mitigation, we could change the trust flag to include checksums for each Catalog being trusted, and emit a warning if they are not provided. Or, we could always require them, and emit an error if they are not provided. In either case, we may want to consider a user-specific Kustomize configuration file that lists a default set of trusted catalogs along with checksums. 

* Although the trust model with Catalog is more granular than the current `--enable-alpha-plugins` flag, end users can still use this feature to execute arbitrary malicious third party code. To mitigate this, the word "trust" is explicitly used in the flag, and the user actually invoking Kustomize must always configure the trust themselves (e.g. the plugins listed in a Kustomization downloaded from a git URL are never automatically trusted via internal reference).

* Exec plugins are inherently dangerous. Currently, even `kubectl kustomize` will execute the legacy style based on the fact that the out of band installation process (which this KEP considers very problematic for usability) proves the user is aware of what will be executed. This KEP proposes two options for new ways the user can prove to use that they understand what they are doing. No matter what we do in this regard, the unrestricted access exec plugins imply will always come with a risk that some user will enable an exec plugin irresponsibly, that the plugin in question will be malicious, and that it will cause the user harm. The only way to fully mitigate this is to deprecate all forms of exec support, and require that all plugin authors use containers. Doing so would seriously hinder the usefulness of Kustomize plugins for individual ad-hoc use cases.

## Design Details

Example execution flow for `kustomize build --trusted-plugin-catalog=company.com/foo.json`:

1. Kustomize retrieves all trusted catalogs specified on the command line.
1. Kustomization is read. If it includes a `catalogs` field, that field is compared to the trusted catalogs from the command line. If any required catalogs have not been trusted, a warning is printed with the details of the missing catalog (which remains untrusted).
1. When preparing to execute a plugin, Kustomize looks for a corresponding entry in the trusted catalogs. If none is found, an error is thrown an execution halts. If one is found, the plugin provider configuration (e.g. network, storage) listed in the plugin config (if any) is compared to the requirements declared in the catalog entry. If they are not a strict subset, an error is thrown and execution halts.
1. Having successfully identified the catalog entry for the given plugin, Kustomize attempts to retrieve the plugin provider from the specified location.
1. Having successfully retrieved the plugin provider, Kustomize hashes it and compares the result to the catalog entry. If it is not an exact match, an error is thrown and execution halts.
1. Having successfully verified the provider's identity, Kustomize invokes it with the user's plugin config.

As an optimization, Kustomize may implement caching for both catalogs and plugin providers that are specified using content-addressable URLs.
   

### Test Plan

Kustomize already has a test harness capable of running plugins, but coverage of KRM-Function-style plugins is not comprehensive. Additional test cases will be added. The Catalog and Composition features described in related KEPs will also require their own extensive tests. End to end tests proving that safeguards related to plugin and catalog trust will be particularly important.

### Rollout Plan

#### Alpha

- Composition implemented with an alpha GV and available in both standalone and embedded Kustomize.
- Catalog implemented with an alpha GV and enabled in standalone Kustomize only behind a  `--alpha-trusted-plugin-catalog` flag.
- `--enable-alpha-plugins` remains required for legacy plugin use, as well as non-catalog KRM Function plugin use (in other words, if you aren't using catalog, the flags that work today still work the same way during alpha)
- Deprecation warnings emitted for all plugin-related flags that are being removed or renamed.
- KRM exec plugin locations subjected to loader restrictions.
- Starlark and legacy go/exec plugin support deprecated
- `kubectl kustomize` also emits deprecation warnings about replaced flags, legacy exec and legacy go (starlark never worked there).
- Documentation for KRM Function-style plugins added to Kustomize website and updated in repo. Container providers will be highlighted and best documented at this stage.

#### Beta

- Beta GVs for Composition and Catalog introduced
- `kustomize edit fix` supports migrating legacy exec to non-KRM exec plugins
- "alpha" removed from `--trusted-plugin-catalog` flag name
- Deprecated plugin-related flags removed.
- Starlark, legacy exec and Go plugin support removed.
- Plugin provider identity verification implemented.
- New plugin developer guide published. Exec provider documentation enhanced.
- `kubectl kustomize` released with beta Catalog support, beta Composition support, identical build flags to standalone kustomize, and full beta plugins support (including KRM exec support)

#### GA

- Two kubectl releases included the beta.
- kyaml fn framework Go module published as stable



### Upgrade / Downgrade Strategy

n/a

### Version Skew Strategy

n/a

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

n/a - client-side only

###### Does enabling the feature change any default behavior?

Enabling only affects the feature itself as described.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

n/a - client-side only

###### What happens if we reenable the feature if it was previously rolled back?

n/a - client-side only

###### Are there any tests for feature enablement/disablement?

n/a - client-side only

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

n/a - client-side only

###### What specific metrics should inform a rollback?

n/a - client-side only

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

n/a - client-side only

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

Yes, it is deprecating several alpha flags, one of which appears in `kubectl kustomize`. It also proposes the replacement of a client-side-only annotation with a client-side-only reserved field in client-side-only CR equivalents. It proposes removing support for three styles of plugins entirely, including the fields and annotation subfields currently used to configure them within client-side-only CR equivalents. Kustomization fields (`generators`, `transformers` and `validators`) are not themselves affected.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?

n/a - client-side only

###### How can someone using this feature know that it is working for their instance?

n/a - client-side only

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

n/a - client-side only

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

Not directly, though plugins could be implemented that make calls to the API server.

###### Will enabling / using this feature result in introducing new API types?

Yes, but only on the client side, and specific to the user.

###### Will enabling / using this feature result in any new calls to the cloud provider?

Not directly, though plugins could be implemented that make calls to cloud providers.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Not in any server-side components. Container plugins are known to have a performance impact that can vary widely depending on factors like container size, mounts being enabled, and of course the time the plugin code itself takes to run.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

n/a

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
    -->

###### What steps should be taken if SLOs are not being met to determine the problem?

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

**Why should this KEP _not_ be implemented?**
It makes it easier for Kustomize users to invoke third party code, which can be malicious, and no safeguards will be perfect. See [Risks and Mitigations](#risks-and-mitigations).

## Alternatives

* Graduate the plugins this KEP calls "legacy" instead of the ones it calls "KRM Function style" (which would be deprecated and removed). TL;DR gaps remain that would require a lot of investment to fill, and that investment would be in a direction that is misaligned with other community efforts in the configuration plugin space.
  * Either containers would not be supported, or the experience would be inconsistent across container plugins and exec/go (it does not make sense to look up containers at a local path).
  * Kustomize would fail to share in the ecosystem of [KRM Function Specification]-based plugins it helped start, and Kustomize plugin authors could not benefit from the plugin SDKs built around that spec.
  * Both legacy plugin styles lack a distribution story, which seriously impedes adoption. They both rely on manual installation to a user-specific path as part of their trust model. In the case of Go plugins, the end user (who may not be a Golang user at all) has to compile Kustomize, another serious barrier to adoption.
  * Because legacy plugin styles require out of band installation to an independent path, Kustomizations that require them are not fully encapsulated/distributable.
* Graduate all existing plugin alphas as-is. TL;DR too much complexity.
  * It is too complicated to use, maintain and document so many different approaches to plugins. The existence of two different ways to run executables is particularly glaring.
  * There would be a permanent difference between kubectl and standalone distributions, since the Starlark and KRM Exec are not acceptable for inclusion in kubectl as they stand today.
  * Only container-based plugins currently have a distribution story viable enough to recommend.
* Graduate "KRM Function style" plugins as described (including removing starlark and further restricting exec), but do not introduce Catalog or Composition.
  * This would be a less usable and useful plugin system than the one proposed, but better than nothing. Catalog and Composition are encapsulated enough that plugin GA can proceed without them if they fail or lose contributor bandwidth before completion. Changes to the flag strategy would be required.

## Infrastructure Needed (Optional)

None. This will be implemented within the existing Kustomize repo.


## Appendix

### Current Plugin Compatibility Grid

| Feature               | Flags                                  | Kustomize (4.2)                                              | Kubectl Kustomize (1.21)                                     |
| --------------------- | -------------------------------------- | ------------------------------------------------------------ | ------------------------------------------------------------ |
| legacy go plugins     | `--enable-alpha-plugins`               | Fails loudly without the plugins flag                        | Fails loudly without the plugins flag                        |
| legacy exec plugins   | `--enable-alpha-plugins`               | Surprisingly, no exec flag required                          | Surprisingly, works (because no exec flag required)          |
| KRM starlark plugins  | `--enable-alpha-plugins --enable-star` | Fails loudly without the plugins flag and silently without the star flag | DISABLED. Throws error that plugins must be enabled, then fails silently if `--enable-alpha-plugins` is passed. The `--enable-star` flag does not exist. |
| KRM exec plugins      | `--enable-alpha-plugins --enable-exec` | Fails loudly without the plugins flag and silently without the exec flag | DISABLED. Throws error that plugins must be enabled, then fails silently if `--enable-alpha-plugins` is passed. The `--enable-exec` flag does not exist. |
| KRM container plugins | `--enable-alpha-plugins`               | `--mount` `--network` and `--network-name` further configure these types of plugins specifically. | All flags appear to be available.                            |

