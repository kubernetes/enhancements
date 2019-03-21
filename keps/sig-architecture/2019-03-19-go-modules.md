---
title: go modules
authors:
  - "@liggitt"
owning-sig: sig-architecture
participating-sigs:
  - sig-api-machinery
  - sig-release
  - sig-testing
reviewers:
  - "@sttts"
  - "@lavalamp"
  - "@cblecker"
  - "@mattfarina"
approvers:
  - "@smarterclayton"
  - "@thockin"
creation-date: 2019-03-19
last-updated: 2019-03-19
status: provisional
---

# go modules

## Table of Contents <!-- omit in toc -->

- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [Manage vendor folders using go modules](#manage-vendor-folders-using-go-modules)
  - [Select a versioning strategy for published modules](#select-a-versioning-strategy-for-published-modules)
  - [Build synthetic godeps.json files](#build-synthetic-godepsjson-files)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
- [Reference](#reference)

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP
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

Manage the vendor folder in `kubernetes/kubernetes` using go modules,
and define `go.mod` module files for published components like `k8s.io/client-go` and `k8s.io/api`.

## Motivation

Since its inception, Kubernetes has used Godep to manage vendored 
dependencies, to ensure reproducible builds and auditable source.

As the go ecosystem matured, vendoring became a first-class concept,
Godep became unmaintained, Kubernetes started using a custom version of Godep,
other vendoring tools (like `glide` and `dep`) became available,
and dependency management was ultimately added directly to go in the form of go modules.

The [plan of record](https://blog.golang.org/modules2019) is for go1.13 to enable
go modules by default and deprecate GOPATH mode. To be ready for that, Kubernetes
components must make any required adjustments in the Kubernetes 1.15 release.

In addition to simply keeping up with the go ecosystem, go modules provide many benefits:
* rebuilding `vendor` with go modules provided a 10x speed increase over Godep in preliminary tests
* go modules can reproduce a consistent `vendor` directory on any OS
* if semantic import versioning is adopted, consumers of Kubernetes modules can use two distinct versions simultaneously (if required by diamond dependencies)

### Goals

* Remove the kubernetes/kubernetes dependency on godep (including the need for a customized version of godep)
* Increase speed and reliability of building and verifying the `vendor` directory in kubernetes/kubernetes
* Provide accurate go.mod module descriptors for kubernetes/kubernetes and published staging components
* Enable use of published kubernetes components by `go module` aware consumers
* Allow continued use of published kubernetes components by `go module` *unaware* consumers until go modules are enabled by default

## Proposal

### Manage vendor folders using go modules
1. Make `k8s.io/kubernetes` and each staging component (like `k8s.io/client-go` and `k8s.io/api`) a distinct go module
  * Pin dependencies to the same versions that are currently listed in Godeps.json
2. Change vendor management and verification scripts in `kubernetes/kubernetes` to use go module commands
3. Change the staging component publishing bot to modify the `go.mod` pinned versions of peer components, rather than rewriting Godeps.json files

### Select a versioning strategy for published modules

As part of transitioning to go modules, we can select a versioning strategy.

Current state:
* `client-go` tags a major version on every kubernetes release
* other components tag a (non-semver) `kubernetes-1.x.y` version on each release
* no import rewriting occurs
* consumers can only make use of a single version of each component

This proposes continuing to tag published components as-is (`kubernetes-1.x.y` for most components, `vX.0.0` for client-go).
This results in the following usage patterns:

Consumers:
* GOPATH consumers
  * import `k8s.io/apimachinery/...` (as they do today)
  * `go get` behavior (e.g. `go get client-go`):
    * uses latest commits of transitive `k8s.io/...` dependencies (likely to break, as today)
    * uses latest commits of transitive non-module-based dependencies (likely to break, as today)
* module-based consumers
  * reference published module versions as `v0.0.0-$date-$sha`
  * import `k8s.io/apimachinery/...` (as they do today)
  * `go get` behavior (e.g. `GO111MODULE=on go get client-go@v15.0.0`):
    * uses go.mod-referenced versions of transitive `k8s.io/...` dependencies (unless overridden by top-level module, or conflicting peers referencing later versions)
    * uses go.mod-referenced versions of transitive non-module-based dependencies (unless overridden by top-level module, or conflicting peers referencing later versions)
* consumers are limited to a single copy of kubernetes libraries among all dependencies (as they are today)

Kubernetes tooling:
* minimal changes required

Compatibility implications:
* breaking go changes in each release impact consumers that have not pinned to particular tags/shas (as they do today)
* conflicting version requirements (direct or transitive) can result in impossible-to-build or impossible-to-update dependencies (as they do today)

Allowed versioning changes:
* modules published this way can transition to semantic import versioning in the future

See the [Alternatives](#alternatives) section for other possible versioning strategies considered for the initial move to modules.

### Build synthetic godeps.json files

Godeps.json files are consumed by some dependency management tools (like `glide` and `dep`).
Existing consumers of published staging components that are not yet using go modules could be impacted by removal of Godeps.json files.
To mitigate this, we could consider generating synthetic Godeps.json files based on go.mod dependency versions.
This section is in progress, pending investigation into necessity and feasibility.

## Design Details

### Test Plan

* CI scripts to verify vendor contents are recreatable and match referenced versions
* CI scripts to verify vendor licenses are up to date
* CI scripts to verify staging component dependencies are correct
* CI scripts to verify staging component publishing is successful
* CI scripts to verify examples using `k8s.io/client-go` can be consumed and build automatically and successfully by GOPATH and module-based consumers

### Graduation Criteria

* kubernetes/kubernetes vendor management uses go modules
* published staging components can be successfully consumed by go module consumers
* published staging components can be successfully consumed by GOPATH-based consumers

### Upgrade / Downgrade Strategy

Not applicable

### Version Skew Strategy

Not applicable

## Implementation History

- 2019-03-19: Created

## Alternatives

* Continue using `godep` for vendor management. This is not viable for several reasons:
  * The tool is unmaintained (the project readme states "Please use dep or another tool instead.")
  * There are significant performance problems (the pull-kubernetes-godeps CI job takes ~30 minutes to verify vendoring)
  * There are significant functional problems (the tool cannot be run in some environments, gets confused by diamond dependencies, etc)

* Use an alternate dependency mechanism (e.g. `dep`, `glide`). This is not preferred for several reasons:
  * Some of the other dependency tools (like `glide`) are also unmaintained
  * go modules are supported by the `go` tool, and have stronger support statements than independent vendoring tools
  * Defining `go.mod` files for published kubernetes components is desired for interoperability with the go ecosystem

* Move away from vendoring as part of the initial move to go modules
  * To ensure reproducible builds based solely on the published repositories, vendoring is still necessary
  * In go1.12.x, vendor-based builds are still the default when building a component located in the GOPATH, so producing components that work when built with go modules or with GOPATH+vendor maximizes interoperability

* For versioning, switch to tagging major versions on every `kubernetes/kubernetes` release (similar to what client-go does), and use semantic import versioning.
This remains a possibility in the future, but requires more tooling and consumer changes to accomodate rewritten imports,
and doesn't fully allow multiple versions of kubernetes components to coexist as long as there are transitive non-module-based dependencies that change incompatibly over time.

    * consumers
      * GOPATH consumers
        * import `k8s.io/apimachinery/...` (as they do today)
        * are limited to a single copy of kubernetes libraries among all dependencies (as they are today)
        * `go get` behavior (e.g. `go get client-go`):
          * uses latest commits of transitive `k8s.io/...` dependencies (likely to break, as today)
          * uses latest commits of transitive non-module-based dependencies (likely to break, as today)
      * module-based consumers
        * reference published modules versions as a consistent `vX.y.z` version (e.g. `v15.0.0`)
        * import `k8s.io/apimachinery/v15/...` (have to rewrite kubernetes component imports on every major version bump)
        * can have multiple copies of kubernetes libraries (though non-semantic-import-version transitive dependencies could still conflict)
        * `go get` behavior (e.g. `GO111MODULE=on go get client-go@v15.0.0`):
          * uses transitive `k8s.io/{component}/v15/...` dependencies
          * uses go.mod-referenced versions of transitive non-module-based dependencies (unless overridden by top-level module, or conflicting peers referencing later versions)
    * kubernetes tooling
      * requires rewriting all `k8s.io/{component}/...` imports to `k8s.io/{component}/vX/...` at the start of each release
      * requires updating code generation scripts to generate versioned imports for `k8s.io/{api,apimachinery,client-go}`, etc
    * compatibility implications
      * allows breaking go changes in each kubernetes "minor" release
      * no breaking go changes are allowed in a kubernetes patch releases (need tooling to enforce this)
    * allowed versioning changes
      * modules published this way would have to continue using semantic import versioning
      * modules published this way could switch to incrementing major/minor versions at a difference cadence as needed

* For versioning, tag major/minor versions as needed when incompatible changes are made, and use semantic import versioning.
This remains a possibility in the future, but requires more tooling and consumer changes to accomodate rewritten imports,
and doesn't fully allow multiple versions of kubernetes components to coexist as long as there are transitive non-module-based dependencies that change incompatibly over time.

    * consumers
      * GOPATH consumers
        * import `k8s.io/apimachinery/...` (as they do today)
        * are limited to a single copy of kubernetes libraries among all dependencies (as they are today)
        * `go get` behavior (e.g. `go get client-go`):
          * uses latest commits of transitive `k8s.io/...` dependencies (likely to break, as today)
          * uses latest commits of transitive non-module-based dependencies (likely to break, as today)
      * module-based consumers
        * reference published modules versions as a variety of `vX.y.z` versions (e.g. `k8s.io/client-go@v15.0.0`, `k8s.io/apimachinery@v15.2.0`, `k8s.io/api@v17.0.0`)
        * import `k8s.io/apimachinery/v15/...` (have to rewrite kubernetes component imports on every major version bump)
        * can have multiple copies of kubernetes libraries (though non-semantic-import-version transitive dependencies could still conflict)
        * `go get` behavior (e.g. `GO111MODULE=on go get client-go@v15.0.0`):
          * uses transitive `k8s.io/{component}/vX/...` dependencies
          * uses go.mod-referenced versions of transitive non-module-based dependencies (unless overridden by top-level module, or conflicting peers referencing later versions)
    * kubernetes tooling
      * requires rewriting all `k8s.io/{component}/vX/...` imports when a major version bump occurs
      * requires updating code generation scripts to generate versioned imports for `k8s.io/{api,apimachinery,client-go}`, etc
      * requires tooling to detect when a breaking go change has occurred in a particular component relative to all tagged releases for the current major version
      * requires tooling to manage versions per component (instead of homogenous versions for staging components)
    * allowed versioning changes
      * modules published this way would have to continue using semantic import versioning
      * modules published this way could switch to incrementing major/minor versions at a difference cadence as needed


## Reference

* [@rsc description of options for kubernetes versioning](https://github.com/kubernetes/kubernetes/pull/65683#issuecomment-403705882)
* `go help modules`
* https://github.com/golang/go/wiki/Modules
  * Especially https://github.com/golang/go/wiki/Modules#semantic-import-versioning
* https://golang.org/cmd/go/#hdr-The_go_mod_file
* https://golang.org/cmd/go/#hdr-Maintaining_module_requirements
* https://golang.org/cmd/go/#hdr-Module_compatibility_and_semantic_versioning
