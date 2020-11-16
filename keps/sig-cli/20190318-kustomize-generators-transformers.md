---
title: Kustomize Generators and Transformers
authors:
  - "@pwittrock"
owning-sig: sig-cli
participating-groups:
reviewers:
  - "@droot"
  - "@sethpollack"
approvers:
  - "@monopole"
editor: TBD
creation-date: 2019-03-25
last-updated: 2019-04-30
status: implementable
---

# Kustomize Generators and Transformers



## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Background](#background)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
    - [Terms](#terms)
    - [Plugins](#plugins)
    - [Plugins: Generate](#plugins-generate)
    - [Plugins: Transform](#plugins-transform)
      - [Restrictions](#restrictions)
  - [Phases](#phases)
  - [User Stories [optional]](#user-stories-optional)
    - [DeclarativeEcosystem Solution Plugins](#declarativeecosystem-solution-plugins)
    - [Bespoke Abstractions for Organizations](#bespoke-abstractions-for-organizations)
    - [Bespoke Configuration for Organizations](#bespoke-configuration-for-organizations)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
<!-- /toc -->

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

Enable users to extend Kustomize *Transformers* and *Generators* through `exec` plugins.  Kustomize
provides Resource Config to plugins through STDIN, and plugins emit Resource Config back to Kustomize
through STDOUT.

- Add 2 fields to `kustomization.yaml`: `generators` and `transformers`
- Plugins to actuate new fields have the path `$XDG_CONFIG_HOME/kustomize/plugins/<API group>`

Much of the existing Kustomize functionality could be implemented through this plugin mechanism.

Example Built-In Generators:

- `resource`
- `bases`
- `secretGenerators`
- `configMapGenerators`

Example Built-In Transforms:

- `commonAnnotations`
- `commonLabels`
- `images`
- `namePrefix` # not supported in iteration 1 due to transformation restrictions
- `nameSuffix` # not supported in iteration 1 due to transformation restrictions
- `patches`
- `namespace` # not supported in iteration 1 due to transformation restrictions


## Motivation

1. Enable organizations to develop their own authoring solutions bespoke to their requirements, and to integrate
   those solutions with kustomize and kubectl.
1. Enable tools and authoring solutions developed by the ecosystem to natively integrate with kustomize and kubectl
   commands.
1. Enable Kustomize power users to augment Kustomize with their own *Transformers* and *Generators*.

### Background

Many users of Kubernetes require the ability to also write Config through high-level abstractions rather
than only as low-level APIs, or to configure Kubernetes through in-house developed platforms.

Authoring solutions have already been developed:

- In the Ecosystem
  - Helm
  - Ksonnet
- Internally by Organizations
  - [kube-gen] (AirBnB) 

Support for connecting the tools developed by the Ecosystem with kubectl relies on piping commands together,
however pipes are an imperative technique that require their own scripting and tooling.

This KEP proposes a declarative approach for connecting tools built in the Kubernetes ecosystem or bespoke
internal tools developed by organizations.

[kube-gen]: https://events.linuxfoundation.org/wp-content/uploads/2017/12/Services-for-All-How-To-Empower-Engineers-with-Kubernetes-Melanie-Cebula-Airbnb.pdf

### Goals

- Allow Config authoring solutions developed in the ecosystem to be declaratively accessed by kubectl as
  Generators and Transformers
- Allow users and organizations to develop their own custom config authoring solutions and integrate
  them into kubectl
- Allow Kustomize power users to augment Kustomize's Built-Tranformers
- Allow users to perform client-side mutations so they they show up in diffs and are auditable

### Non-Goals

- Rewrite Kustomize on top of plugins
  - This should be possible, but isn't a problem we need to solve right now
- Develop a market place of any sort
  - The ecosystem solutions themselves will have market places.  This allows those ecosystem tools to
    plug their market places into kubectl via Kustomize.

## Proposal

Introduce an executable plugin that has 2 subcommands.

- `$ team.example.com generate`
- `$ team.example.com transform`

Introduce 2 new fields to `kustomization.yaml`

- `generators`
- `transformers`

#### Terms

- *Virtual Resource*
  - Resource that is not a server-side Kubernetes API object, but used to configure tools
  - Examples: kubeconfig, kustomization
- *Generator* Kustomize directive
  - Accepts Virtual Resources as input
  - Emits generated non-Virtual Resources as output
  - Examples: `configMapGenerator`, `secretGenerator`
- *Transformer* Kustomize directive
  - Accepts Virtual Resources
  - Accepts **all** non-Virtual Resources as input
  - Emits modified non-Virtual Resources as output
  - Examples: `commonAnnotations`, `namespace`, `namePrefix`, `secretGenerator` (for updating references to the generated secret)
- *Built-In*
  - Either a Transformer or Generator that is part of the `kustomization.yaml` rather than
    as a separate Virtual Resource.
- *Plugin*
  - Either a Transformer or Generator that is *not* part of the `kustomization.yaml` and comes from a plugin.

#### Plugins

Generators and Transformers are Configured as Virtual Resources.

Plugins implement Generators and Transformers.

- Plugins are installed `$XDG_CONFIG_HOME/kustomize/plugins/
- Plugin executables names match the Virtual Resource API Group the own - e.g. `team.example.com`
- Plugin executables have 2 subcommands - `generate` and `transform`
- `generate`
  - Accepts 0 or more Virtual Resources whose group matches the executable name on STDIN
  - Emits 0 or more Non-Virtual Resources
- `transform`
  - Accepts 0 or more Virtual Resources whose group matches the executable name on STDIN
  - Accepts 0 or more Non-Virtual Resources from all API groups
  - Emits 0 or more Non-Virtual Resources
- Generators and Transformers are configured as Virtual Resources and the `generators` and `transformers` fields
  on `kustomization.yaml`.
- Plugins working directory is the directory of the `kustomization.yaml` with the `generator` or `transformer`.
  - Plugins should never be able to access files outside this directory structure (e.g. only child directories)
- Plugins inherit kustomize process environment variables

Plugin guidelines:

- If executed with an unrecognized subcommand, plugins should exit `127` signalling to kustomize that the
  operation is not supported.
- Plugins should never change state.  They should be able to be executed with `kubectl apply -k --dry-run` or
  `kubectl diff`.
- Plugin output should be idempotent.

#### Plugins: Generate

Generators generate 0 or more Resources for some Virtual Resources.

Example: Kustomize secretGenerators and configMapGenerators generate Secrets and ConfigMaps
from a `kustomize.config.k8s.io/v1beta1/Kustomization` virtual Resource.

Generators have 2 components:

1. Generator Config
  - Virtual Resource
  - Added to the `kustomization.yaml` field `generators`
1. Generator Implementation
  - Executable plugin
  - Reads Virtual Resources
  - Emits non-Virtual Resources

1. Kustomize reads the `generators` Virtual Resources
1. Kustomize maps the Virtual Resources to plugins by their *Group*
   - Group matches the plugin name
   - Exits non-0 if no plugins are found any generator entries
1. For each Virtual Resource *Group*
  1. Kustomize execs the plugin `generate` command
  1. Kustomize writes the Virtual Resources in that Group to the process STDIN
  1. Kustomize reads the set of generated Resources from the process STDOUT
  1. Kustomize reads error messages from the exec process STDERR
  1. Kustomize fails if the plugin exits non-0
  1. Kustomize adds the emitted Resources to its set of Non-Virtual Resources (e.g. from `resources`, `bases`, etc).

The order of plugin execution is arbitrary.

#### Plugins: Transform

Transformers modify existing non-virtual Resources by modifying their fields.

Transformers have 2 components:

1. Transformer Config
  - Virtual Resource
  - Added to the `kustomization.yaml` field `transformers`
  - All `generators` are implicitly invoked as transformers
1. Transformer Implementation
  - Executable plugin
  - Reads Virtual Resources
  - Reads **all** non-Virtual Resources
  - Emits non-Virtual Resources  
  
1. Kustomize reads the `transformers` *and* `generators` Virtual Resources
  - `generators` can require transformation
1. Kustomize maps the Virtual Resources to plugins by their *Group*
   - Group matches the plugin name
   - Exits non-0 if no plugins are found any generator entries
1. For each Virtual Resource *Group*
  1. Kustomize execs the plugin `transform` command
  1. Kustomize writes the Virtual Resources in that Group to the process STDIN
  1. Kustomize writes **all** non-Virtual Resources to the process STDIN
  1. Kustomize reads the set of transformed Resources from the process STDOUT
  1. Kustomize reads error messages from the exec process STDERR
  1. Kustomize fails if the plugin exits non-0
  1. Kustomize replaces its current set of non-Virtual Resources with the set of emitted Resources.

The order of plugin execution is arbitrary.

##### Restrictions

For the initial iteration, transformers cannot add/remove Resources, or change their names/namespaces.
The ability doing so would be significantly make complex ordering interactions much more likely.  E.g. transformers
would need to keep and propagate transformations from other transformers.

We are prioritizing a restrictive but predictable API over a powerful but complex one.

### Phases

Follow is the Kustomize workflow:

1. Read all `generators` and `transformers`
1. Read all `patches`
1. Apply Virtual Resource `patches` to `generators` and `transformers`
1. Generate non-Virtual Resource set from Built-In Generators `inputs`, `bases`, `secretGenerator`, etc
1. Generate non-Virtual Resources from Plugin-Generators and add to non-Virtual Resource set
1. Transform non-Virtual Resources using Plugin-Generators
1. Transform non-Virtual Resources using Built-In Generators (including a second round of `patches`)
1. Built-In Sorting

### User Stories [optional]

#### DeclarativeEcosystem Solution Plugins

Alice uses GitOps for deployment of her Kubernetes Resource Config.  Alice has a Helm
chart that she would like to include into her GitOps workflow.  Alice would like to be
able to check the chart and values.yaml into her repository and have it deployed
just like her other config.  Alice would also like to be able to apply cross cutting
transformations - such as labels, annotations, name-prefixes, etc - to the Resources generated
by the Helm chart.

Alice uses the Kustomize helm chart plugin to declaratively generate
Resource Config using Helm charts, which will then have kustomizations applied.

1. Alice downloads and installs the Helm generator plugin from *github.com/kubernetes-sigs/kustomize/*
   and installs it at `$XDG_CONFIG_HOME/kustomize/plugins/generators/helm.kustomize.io`.
1. Alice creates the `chart.yaml` Resource Config and adds it to her `kustomization.yaml` as a `generators` entry.
  - Having the groupVersion `helm.kustomize.io` triggers the generator under `plugins/generators/helm.kustomize.io`

```yaml
groupVersion: helm.kustomize.io/v1beta1
kind: Chart
generate:
    chart: ./path/to/chart/from/kustomization.yaml
    values:
      k1: v1
      k2: v2
```

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

generators:
- chart.yaml
```

`$ helm.kustomize.io generate` is invoked with the `chart.yaml` contents passed on STDIN.  Kustomize
adds the emitted Resources to its set of Resources.

Note, the `transform` subcommand will also be invoked, which the plugin may choose to pass-through
by emitting its input.

#### Bespoke Abstractions for Organizations

Bob's organization deploys many variants of the same high-level conceptual set of Resources.  Creating the Resource Config
for each instance that needs to be deployed requires lots of boilerplate.  Instead Bob's organization develops
tools for generating standardized Resource Config based off of some inputs.

Bob creates a new generator for his organization that allow higher level abstractions to be defined as
new virtual Resource types that don't exist in the cluster, but are used to generate low-level types.

1. Bob builds and installs the new generator plugin `$XDG_CONFIG_HOME/kustomize/plugins/team.example.com`.
1. Bob creates the `app.yaml` Resource Config and adds it to his `kustomization.yaml`

```yaml
groupVersion: team.example.com/v1beta1
kind: CommonApp
generate:
  image: foo
  size: big
```

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

transformers:
- app.yaml
```

`$ team.example.com transform` is invoked with the `app.yaml` contents passed on STDIN.  Kustomize adds the
emitted Resource to its set of Resources.

#### Bespoke Configuration for Organizations

Alice's Organization requires that various fields are defaulted if unset.  SRE would like to be able to see
the full Resources that are being Applied and have this auditable in an scm, such as git, rather than
having Webhooks provide server-side mutions that are not capture in review or scm.

1. Alice builds and installs the new transformer plugin at `$XDG_CONFIG_HOME/kustomize/plugins/team.example.com`
1. Alice creates the `transform.yaml` Resource Config and adds it to her `kustomization.yaml` as a `transformers` entry.

```yaml
groupVersion: team.example.com
kind: CommonApp
transform:
  namePrefix: foo-
  commonAnnotations:
    foo: bar
```

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

transformers:
- transform.yaml
```

`$ team.example.com` is invoked with both the `transform.yaml` contents and the Resources passed on STDIN.
Kustomize replaces its Resources with the emitted Resources.

### Implementation Details/Notes/Constraints [optional]

### Risks and Mitigations

## Design Details

### Test Plan
1. Add unit tests
2. Add integration tests
3. Add example tests

### Graduation Criteria

Alpha -> Beta Graduation
1. Executing plugins for generated Resources
2. Plugin Generators interact with Kustomize build-in transformers

Beta -> GA Graduation
1. Plugin Transformers after Built-in Transformers
2. Support ordering


### Upgrade / Downgrade Strategy

NA - Client side only

### Version Skew Strategy

NA - Client side only

## Implementation History


## Drawbacks [optional]

- It allows users to do complex thing with Kustomize.
- Virtual Resources may be confusing

## Alternatives [optional]

- Imperatively piping generator executables to kubectl apply
- Writing scripts to invoke generator executables and piping them to kubectl apply
- Create a new platform that is purely plug-in based, and rebase kustomize on top of this as a plugin
- Support explicit ordering of plugin execution
- Use separate plugins for Generators and Transformers
- Allow arbitrary Transformation changes
  - Increases complexity + interactions
  - Reduces readability