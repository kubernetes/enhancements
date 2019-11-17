---
title: kubectl-v2
authors:
  - "@pwittrock"
owning-sig: sig-cli
reviewers:
  - "@soltysh"
  - "@seans3"
  - "sig-cli-all"
approvers:
  - "@soltysh"
  - "@seans3"
editor: TBD
creation-date: 2019-13-16
last-updated: yyyy-mm-dd
status: provisional
---

# kubectl-v2

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
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

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

kubectl-v2 is intended to be the second generation of kubectl commands, and
will be implemented as a kubectl plugin.
It will be focused on iterating upon the existing kubectl commands for
the management of Resources using declarative configuration.

kubectl-v2 will have notable differences from existing (v1) kubectl commands:
  
- Broader version skew support
  - Support backwards compatibility for all Kubernetes releases in preceding 18months
  - Forward compatibility to be provided by Kubernetes API backward compatibility
- Broader support for extension types
  - All commands developed used mechanisms that support extensions types
- Better interoperability with posix utilities
  - File processing commands work with `xargs`, `find -exec`, globbing
- Simpler / Minimal commands
  - Enable complex workflows through composition with other tools, rather than
    building an extensive set of options.

kubectl-v2 will initially implement the following workflows:

1. Working Resources using declarative configuration
  - Apply (e.g. `kubectl apply`)
  - Prune (e.g. `kubectl apply --prune`)
  - Status (e.g. `kubectl rollout status`)
2. Working with declarative configuration as files
  - Set workflows (e.g. `kubectl set image`)
    - High-level commands for manipulating configuration
  - List workflows (e.g. `kubectl get`)
    - High-level commands for printing configuration

## Motivation

Kubectl Kubernetes project owned cli tool for working with Kubernetes Resources and configuration.
Since kubectl 1.0 was release in mid 2015, our thinking regarding the needs of the Kubernetes cli
have matured, most notably in the following areas:

- Client/Server version skew
- API Extension support

Additionally, improvements can be made to the cli UX in:

- Simpler options designed to be composed common cli tools (e.g. `jq`)
- Interface to enabled composition using unix cli standards (`xargs`, `find -exec`, globbing)
- Retaining configuration comments, structure, etc

### Goals

Apply lessons learned from 4+ years of kubectl usage to develop a 2nd generation of commands.
Develop and publish commands outside of kubernetes/kubernetes.

### Non-Goals

Replace or deprecate existing kubectl commands.

## Proposal

Develop a `kubectl-v2` plugin outside of the kubernetes/kubernetes repo.

### Initial Commands

#### Declarative Client-Server Utilities

**Apply**: Write simplified `apply` command which uses server-side apply.  Drop poorly supported /
obsolete options (e.g. `--prune`, `--overwrite`, `--record`, `--template`).

**Prune:** Replace `apply --prune` with a `prune` command.  Redesign from the ground up to
be "safe".

**Status:** Enable extension and multi-resource support for `rollout status`.  Use Conditions
for determining Resource status when possible.

#### Declarative File / Stdin Utilities

**Modifying Files:**

- example: `set` which operates on files by default, and uses discoverable API information for
   configuring the command group and behavior.
- example: `fmt` to format yaml files -- putting `apiVersion`, `kind` and `name`
  before other fields.  Sort lists by `name` if the field is present.

**Printing and Display Input:**

- example: print the Kubernetes Resources in a directory -- ignoring non-Resource files
- example: display relationship between Resources as a `tree`

### Architecture

#### Release

The plugin will be published as its own go binary, and released independently of the kubernetes
releases.

#### Version Skew

Better support for backwards compatibility with older Kubernetes versions.
Better support for forwards compatibility by discovering data to configure type
specific commands.

- Plugin commands MUST support compatibility with the last 6 Kubernetes cluster releases.
  - Starting with Kubernetes 1.17
- New commands depending upon capabilities not available in all 6 preceding releases MUST
  check that the feature exists in the cluster before performing cluster operations.  If
  the feature does not exist, the command MUST provide an informative error message and
  exit non-0 before performing any mutations.
  - Discovery
  - OpenAPI schema
- Plugin will use server published information to configure commands to support forward
  compatibility and extension types.
  - Discovery
  - OpenAPI schema  

#### Input / Output

Better support for reading/writing yaml and json configuration.

- When reading/writing yaml files, commands SHOULD preserve yaml structure and comments by default
  - Commands SHOULD use a sensible and consistent format when writing yaml
    files -- e.g. for field ordering
  - Commands SHOULD avoid serializing yaml or json from type specific go structures which drops
    information.
- Commands which accept as input multiple files or directories SHOULD accept these as args
  so that they work with `xargs`, globbing, etc.
  - Commands SHOULD also accept `-f` for compatibility with `v1` kubectl command semantics
- Commands which accept as input files or directories and stdin SHOULD default to reading from
  stdin if no files or directories are provided as arguments or flags.

#### Extension

See [Version Skew](#version-skew)

- Commands SHOULD avoid hard-coding built-in type specific logic when that logic could be
  encoded in -- Discovery, OpenAPI, Json Schema, Duck-Typing, etc.
- Commands may include hard-code type specific logic intended to be published in the server
  as a bridge to may the logic available.

#### Location

`kubectl-v2` will be published out the the kustomize repo as its own module:
`sigs.k8s.io/kustomize/cmd/kubectl-v2`

Why in the Kustomize repo?

- Enables holistic ownership of components owned by sig-cli -- single dashboard for
  issues, PRs, etc.
- Has critical mass of contributors -- performing all k/k development in a single repo
  will help unify the area and reduce the on boarding inertia for new contributors.
- Enables applying consistent build, release, linting processes and machinery.

Why not elsewhere?

- `kubectl`, `cli-runtime`: published out of staging, and shouldn't contain
  their own code.
- `cli-utils`, `cli-experimental`: must smaller set of contributors, immature release process.

## Design Details

### Test Plan

Individual commands MAY be tested using one of:

- *Kind*
- A local apiserver + etcd

Commands MUST be tested without a full Kubernetes cluster.
Each command MUST be tested for backwards compatibility against the last 6 Kubernetes
releases, starting at release 1.17.

### Graduation Criteria

#### Pre-Alpha

Published as kubectl plugin, installable with `go get`

#### Alpha

After completion of initial round of `v2` commands in plugin.

`v2` command introduced `kubectl`, does **not** contain v2 plugin, but contains instructions
for downloading the plugin.

