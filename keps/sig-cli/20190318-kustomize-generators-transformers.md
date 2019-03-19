---
title: Kustomize Generators and Transformers
authors:
  - "@pwittrock"
owning-sig: sig-cli
participating-sigs:
reviewers:
  - "@droot"
  - "@sethpollack"
approvers:
  - "@monopole"
editor: TBD
creation-date: yyyy-mm-dd
last-updated: yyyy-mm-dd
status: provisional
---

# Kustomize Generators and Transformers



## Table of Contents

A table of contents is helpful for quickly jumping to sections of a KEP and for highlighting any additional information provided beyond the standard KEP template.
[Tools for generating][] a table of contents from markdown are available.

- [Title](#title)
  - [Table of Contents](#table-of-contents)
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [User Stories [optional]](#user-stories-optional)
      - [Story 1](#story-1)
      - [Story 2](#story-2)
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

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

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

This KEP adds 2 fields to `kustomization.yaml`

- `generators`
  - plugins under `$XDG_CONFIG_HOME/kustomize/plugins/generators/<API group>`
- `transformers`
  - plugins under `$XDG_CONFIG_HOME/kustomize/plugins/transformers/<API group>`

Most, if not all, of the existing Kustomize functionality could be implemented through this plugin mechanism.

Built-In Generators:

- `resource`
- `bases`
- `secretGenerators`
- `configMapGenerators`

Built-In Transforms:

- `commonAnnotations`
- `commonLabels`
- `images`
- `namePrefix`
- `nameSuffix`
- `patches`
- `namespace`


## Motivation

1. Enable organizations to develop their own authoring solutions bespoke to their requirements, and to integrate
   those solutions with kustomize and kubectl.
1. Enable tools and authoring solutions developed by the ecosystem to natively integrate with kustomize and kubectl
   commands.
1. Enable Kustomize power users to augment Kustomize with their own *Transformers* and *Generators*.

### Background

Many users of Kubernetes require the ability to also write Config through high-level abstractions
rather than only as low-level APIs.

- Numerous tools and DSLs have been developed in the ecosystem in order to address this need
  - Helm
  - Ksonnet
- Organizations have built their own tooling for generating Kubernetes Resource Config
  - [kube-gen] (AirBnB) 

Support for connecting the tools developed by the ecosystem with kubectl relies on piping commands together,
however pipes are an imperative technique.  This KEP proposes a declarative approach for composing tools built
in the Kubernetes ecosystem or bespoke tools developed by organizations that are tightly coupled to their
environments.

[kube-gen]: https://events.linuxfoundation.org/wp-content/uploads/2017/12/Services-for-All-How-To-Empower-Engineers-with-Kubernetes-Melanie-Cebula-Airbnb.pdf

### Goals

- Allow config authoring solutions developed in the ecosystem to be integrated into kubectl as declarative
  Generators and Transformers
- Allow users and organizations to develop their own custom configu authoring solutions and integrate
  them into kubectl
- Allow Kustomize power users to augment Kustomize's own Tranformers with their own
- Allow users to perform client-side mutations so they they show up in diffs and are auditable

### Non-Goals

- Rewrite Kustomize on top of plugins
  - This should be possible, but isn't a problem we need to solve right now
- Develop a market place of any sort
  - The ecosystem solutions themselves will have market places.  This allows those ecosystem tools to
    plug their market places into kubectl via Kustomize.

## Proposal

Introduce 2 new types of `exec` plugins:

- Transformers
- Generators

#### Terms

- *Virtual Resource*
  - Resource that is not a server-side Kubernetes API object, but used to configure tools
  - Examples: kubeconfig, kustomization
- *Generator*
  - Kustomize directive that accepts a Virtual Resource as input, and emits non-Virtual Resources as output
  - Examples: `configMapGenerator`, `secretGenerator`
- *Transformer*
  - Kustomize directive that accepts 1 Virtual Resource *and* collection of non-Virtual Resources as input
    and emits non-Virtual Resources as output.
  - Examples: `commonAnnotations`, `namespace`
- *Built-In*
  - Either a Transformer or Generator that is embedded directly in the `kustomization.yaml` rather than
    as a separate API.

#### Generators

Generators generate 0 or more Resources for some virtual Resource input.

Generator plugins are run immediately after built-in generators:

- `resources`
- `bases`
- `secretGenerator`
- `configMapGenerator`

and before any built-in transformers such as:

- `namePrefix`
- `commonAnnotations`

Example: Kustomize secretGenerators and configMapGenerators generate Secrets and ConfigMaps
from a `kustomize.config.k8s.io/v1beta1/Kustomization` virtual Resource.

Generators have 2 components:

1. Generator virtual Resource API definition
  - By convention, API has a field `generate`
1. Generator executable plugin that reads the Resources on STDIN and emits generated Resources on STDOUT.

- A Generator is installed by adding the generator executable to
  `$XDG_CONFIG_HOME/kustomize/plugins/generators/<API group>`.
  - The name of the generator executable is an API *Group*, and the Generator is is responsible for
    all Virtual APIs in that group.
  - Example Generator executable name: `team.example.com`
- Generators are added as virtual Resources using the `kustomization.yaml` field `generators`

Steps for each `generator`:

1. Kustomize reads the `generator` entry
1. Kustomize uses the Resource's *Group* to find the plugin under `$XDG_CONFIG_HOME/kustomize/plugins/generators/`
1. Kustomize execs the plugin it finds, or exits non-0
1. Kustomize writes the `generator` Resource to the exec process STDIN
1. Kustomize reads the set of generated Resources from the exec process STDOUT
1. Kustomize reads error messages from the exec process STDERR
1. Kustomize fails if the plugin exits non-0
1. Kustomize adds the emitted Resources to its set of input Resources (e.g. from `resources`, `bases`, etc).

Notes:

- Kustomize will not execute any plugins for the output Resources at this time.
- If multiple generators are specified, they are run in the order they are listed.

#### Transforms

Transformers modify existing non-virtual Resources by modifying their fields or replacing them with new Resources.

Transformer plugins are run immediately *before* built-in transformers:

- `namePrefix`
- `commonAnnotations`

This ordering is so that built-in transformers will be applied to transformed Resources if any new Resources
are added by Transformers.

Transformers have 2 components:

1. Transformer virtual Resource API definition
  - By convention, API has a field `transform`
1. Transformer executable plugin that reads the Resources on STDIN and emits generated Resources on STDOUT.

- A Transformer is installed by adding the generator executable to
  `$XDG_CONFIG_HOME/kustomize/plugins/transformers/<API group>`.
  - The name of the transformer executable is an API *Group*, and the Transformer is is responsible for
    all Virtual APIs in that group.
  - Example Transformer executable name: `team.example.com`
- Transformers are added as virtual Resources using the `kustomization.yaml` field `transformers`

Steps for each `transformer`:

1. Kustomize reads the `transformer` entry
1. Kustomize uses the Resource's *Group* to find the plugin under `$XDG_CONFIG_HOME/kustomize/plugins/transformers/`
1. Kustomize execs the plugin it finds, or exits non-0
1. Kustomize writes the `transformer` Resource to the exec process STDIN (and `---` to mark its end)
1. Kustomize writes each input Resource (e.g. those from `resources`, `bases`, `generators`, `configMapGenerator`)
   to the exec process STDIN (separated by `---`)
1. Kustomize reads the set of transformed Resources from the exec process STDOUT
1. Kustomize reads error messages from the exec process STDERR
1. Kustomize fails if the plugin exits non-0
1. Kustomize replaces its set of input Resources (e.g. from `resources`, `bases`, etc) with the transformed Resources. 

Notes:

- Transformers may delete, modify or add Resources.
- If multiple transformers are specified, they are run in the order they are listed.


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

The `helm.kustomize.io` binary is invoked with the `chart.yaml` contents passed on STDIN.  Kustomize adds the emitted Resource
to its set of Resources.

#### Bespoke Abstractions for Organizations

Bob's organization deploys many variants of the same high-level conceptual set of Resources.  Creating the Resource Config
for each instance that needs to be deployed requires lots of boilerplate.  Instead Bob's organization develops
tools for generating standardized Resource Config based off of some inputs.

Bob creates a new generator for his organization that allow higher level abstractions to be defined as
new virtual Resource types that don't exist in the cluster, but are used to generate low-level types.

1. Bob builds and installs the new generator plugin `$XDG_CONFIG_HOME/kustomize/plugins/generators/team.example.com`.
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

generators:
- app.yaml
```

The `team.example.com` binary is invoked with the `app.yaml` contents passed on STDIN.  Kustomize adds the
emitted Resource to its set of Resources.

If multiple generators are specified, they are run in the order they are listed.

#### Bespoke Configuration for Organizations

Alice's Organization requires that various fields are defaulted if unset.  SRE would like to be able to see
the full Resources that are being Applied and have this auditable in an scm, such as git, rather than
having Webhooks provide server-side mutions that are not capture in review or scm.

1. Alice builds and installs the new transformer plugin at `$XDG_CONFIG_HOME/kustomize/plugins/transformers/team.example.com`
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

The `team.example.com` binary is invoked with both the `transform.yaml` contents and the Resources passed on STDIN.
Kustomize replaces its Resources with the emitted Resources.

### Implementation Details/Notes/Constraints [optional]

### Risks and Mitigations

## Design Details

### Test Plan


### Graduation Criteria

TBD

Consider:

- Executing plugins for generated Resources

### Upgrade / Downgrade Strategy

NA - Client side only

### Version Skew Strategy

NA - Client side only

## Implementation History


## Drawbacks [optional]

It allows users to do very complex thing with Kustomize.

## Alternatives [optional]

- Imperatively piping generator executables to kubectl apply
- Writing scripts to invoke generator executables and piping them to kubectl apply
