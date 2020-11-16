---
title: go modules
authors:
  - "@liggitt"
owning-sig: sig-architecture
participating-groups:
  - sig-api-machinery
  - sig-release
  - sig-testing
reviewers:
  - "@sttts"
  - "@lavalamp"
approvers:
  - "@smarterclayton"
  - "@thockin"
creation-date: 2019-03-19
last-updated: 2019-11-01
status: implementable
---

# go modules

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [Manage vendor folders using go modules](#manage-vendor-folders-using-go-modules)
  - [Publish staging component modules to individual repositories](#publish-staging-component-modules-to-individual-repositories)
  - [Select a versioning strategy for published modules](#select-a-versioning-strategy-for-published-modules)
  - [Remove Godeps](#remove-godeps)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [Alternatives to vendoring using go modules](#alternatives-to-vendoring-using-go-modules)
  - [Alternatives to publishing staging component modules](#alternatives-to-publishing-staging-component-modules)
  - [Alternative versioning strategies](#alternative-versioning-strategies)
- [Reference](#reference)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

These checklist items _must_ be updated for the enhancement to be released.

- [x] kubernetes/enhancements issue in release milestone, which links to KEP: https://github.com/kubernetes/enhancements/issues/917
- [x] KEP approvers have set the KEP status to `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [x] "Implementation History" section is up-to-date for milestone
- [x] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes
- [x] User-facing documentation has been created

[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes

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
* Allow consumers of go modules to understand the versions of kubernetes libraries they are using

## Proposal

### Manage vendor folders using go modules
1. Generate `go.mod` files for `k8s.io/kubernetes` and each staging component (e.g. `k8s.io/client-go` and `k8s.io/api`) as a distinct go module
  
    * Add `require` and `replace` directives in all the `go.mod` files to register a preference for the same dependency versions currently listed in `Godeps.json`

        ```
        require (
          bitbucket.org/bertimus9/systemstat v0.0.0-20180207000608-0eeff89b0690
          ...
        )
        replace bitbucket.org/bertimus9/systemstat => bitbucket.org/bertimus9/systemstat v0.0.0-20180207000608-0eeff89b0690
        ...
        ```

    * Add `require` and `replace` directives in the `k8s.io/kubernetes` `go.mod` file to point to the staging component directories:

        ```
        require (
          k8s.io/api v0.0.0
          ...
        )
        replace k8s.io/api => ./staging/src/k8s.io/api
        ...
        ```

    * Add `require` and `replace` directives in the staging component `go.mod` files to point to the peer component directories:

        ```
        require (
          k8s.io/api v0.0.0
          ...
        )
        replace k8s.io/api => ../api
        ...
        ```

2. Change vendor creation and verification scripts in `kubernetes/kubernetes` to use go module commands to:

    * Sync dependency versions between `k8s.io/kubernetes` and in staging component `go.mod` files
    * Build the vendor directory (`go mod vendor`)
    * Generate the vendored `LICENSES` file

See the [alternatives](#alternatives-to-vendoring-using-go-modules) section for other vendoring tools considered.

### Publish staging component modules to individual repositories

Update the staging component publishing bot:
1. Rewrite the staging component `require` directives in `go.mod` files to require specific published versions (the same thing done today for `Godeps.json` files)
2. Remove the staging component `replace` directives in `go.mod` files (relative path references don't make sense for independent repositories)
3. Stop including `vendor` content in published modules. Vendor folders are ignored in non-top-level modules, and published `go.mod` files inform `go get` of desired versions of transitive dependencies.
4. Generate synthetic `Godeps.json` files containing the SHA or git tag of module dependencies,
for consumption by downstream consumers using dependency management tools like `glide`.
Continue publishing these at least until our minimum supported version of go defaults to enabling
module support (currently targeted for go 1.13, which is approximately Kubernetes 1.16-1.17 timeframe).

See the [alternatives](#alternatives-to-publishing-staging-component-modules) section for other publishing methods considered.

### Select a versioning strategy for published modules

As part of transitioning to go modules, we can select a versioning strategy.

State prior to adoption of go modules:
* `client-go` tagged a major version on every kubernetes release
* all components tagged a (non-semver) `kubernetes-1.x.y` version on each release
* no import rewriting occurred
* consumers could only make use of a single version of each component in a build

State after adoption of go modules:
* all components tagged a (non-semver) `kubernetes-1.x.y` version on each release
* no import rewriting occurred
* consumers could only make use of a single version of each component in a build

This proposes publishing components with the following tags:
* Non-semver tags of `kubernetes-1.x.y` (corresponding to kubernetes `v1.x.y`)
* Semver tags of `v0.x.y` (corresponding to kubernetes `v1.x.y`)

`v0.x.y` accurately convey the current guarantees around the go APIs release-to-release.
The semver tags are preserved in the go.mod files of consuming components, 
allowing them to see what versions of kubernetes libraries they are using.
Without semver tags, downstream components see "pseudo-versions" like
`v0.0.0-20181208010431-42b417875d0f` in their go.mod files, making it 
extremely difficult to see if there are version mismatches between the 
kubernetes libraries they are using.

This results in the following usage patterns:

Consumers:
* GOPATH consumers
  * import `k8s.io/apimachinery/...` (as they do today)
  * `go get` behavior (e.g. `go get client-go`):
    * uses latest commits of transitive `k8s.io/...` dependencies (likely to break, as today)
    * uses latest commits of transitive non-module-based dependencies (likely to break, as today)
* module-based consumers using a specific version
  * reference published module versions as `v0.x.y` or `kubernetes-1.x.y`
  * import `k8s.io/apimachinery/...` (as they do today)
  * `go get` behavior (e.g. `GO111MODULE=on go get k8s.io/client-go@v0.17.0` or `GO111MODULE=on go get k8s.io/client-go@kubernetes-1.17.0`):
    * uses `go.mod`-referenced versions of transitive `k8s.io/...` dependencies (unless overridden by top-level module, or conflicting peers referencing later versions)
    * uses `go.mod`-referenced versions of transitive non-module-based dependencies (unless overridden by top-level module, or conflicting peers referencing later versions)
* consumers are limited to a single copy of kubernetes libraries among all dependencies (as they are today)

Kubernetes tooling:
* minimal changes required

Compatibility implications:
* breaking go changes in each release impact consumers that have not pinned to particular tags/shas (as they do today)
* conflicting version requirements (direct or transitive) can result in impossible-to-build or impossible-to-update dependencies (as they do today)

This would not limit future changes to our versioning strategy:
* modules published and tagged this way could transition to semantic import versioning in the future, if desired
* modules published and tagged this way could transition to v1.x.y semver tagging in the future, if desired
  (this would require enforcement of go API compatibility in perpetuity, and prevent removal of *any* go API element,
  so we are unlikely to pursue this approach, but adding v0.x.y tags now does not remove the option)

See the [alternatives](#alternative-versioning-strategies) section for other possible versioning strategies considered for the initial move to modules.

### Remove Godeps

* Move aggregated `Godeps/LICENSES` file to `vendor/LICENSES` (and ensure it is packaged correctly in build artifacts)
* Remove `Godeps.json` files from `kubernetes/kubernetes` and staging component directories.
With the change to go modules, the only use for these is as a hint to non-module-based downstream consumers of the published staging components.
* Remove the custom `Godeps` fork from `kubernetes/kubernetes`
* Remove all other `Godeps` references in scripts, comments, and configurations files

## Design Details

### Test Plan

* CI scripts to verify vendor contents are recreatable and match referenced versions
* CI scripts to verify vendor licenses are up to date
* CI scripts to verify staging component dependencies are correct
* CI scripts to verify staging component publishing is successful
* CI scripts to verify examples using `k8s.io/client-go` can be consumed and build automatically and successfully by GOPATH and module-based consumers

### Graduation Criteria

* `k8s.io/kubernetes` vendor management uses go modules
  * CI verifies vendor management scripts succeed with `GOPATH` unset, in a directory structure not shaped like `$GOPATH`
* there are documented processes for:
  * adding/pinning a new dependency
  * updating the pinned version of an existing dependency
  * removing an unnecessary dependency
* published staging components can be successfully consumed by:
  * go module consumers using `go get`
  * GOPATH-based consumers using `go get`

### Upgrade / Downgrade Strategy

Not applicable

### Version Skew Strategy

Not applicable

## Implementation History

- 2019-03-19: Created
- 2019-03-26: Completed proposal
- 2019-03-26: Marked implementable
- 2019-04-03: Implemented go module support
- 2019-11-01: Added proposal for tagging published modules with v0.x.y

## Alternatives

### Alternatives to vendoring using go modules

* Continue using `godep` for vendor management. This is not viable for several reasons:
  * The tool is unmaintained (the project readme states "Please use dep or another tool instead."), and we have had to make our own fork to work around some edge cases.
  * There are significant performance problems (the pull-kubernetes-godeps CI job takes ~30 minutes to verify vendoring)
  * There are significant functional problems (the tool cannot be run in some environments, gets confused by diamond dependencies, etc)

* Use an alternate dependency mechanism (e.g. `dep`, `glide`). This is not preferred for several reasons:
  * Some of the other dependency tools (like `glide`) are also unmaintained
  * go modules are supported by the `go` tool, and have stronger support statements than independent vendoring tools
  * Defining `go.mod` files for published kubernetes components is desired for interoperability with the go ecosystem

* Move away from vendoring in `k8s.io/kubernetes` as part of the initial move to go modules
  * To ensure reproducible builds in hermetic build environments based solely on the published repositories, vendoring is still necessary
  * Moving away from vendoring is orthogonal to moving to go modules and could be investigated/pursued in the future if warranted.
  * In go1.12.x, vendor-based builds are still the default when building a component located in the GOPATH, so producing components that work when built with go modules or with GOPATH+vendor maximizes interoperability

### Alternatives to publishing staging component modules

Since `require` directives allow locating modules within other modules,
it is theoretically possible to stop publishing staging component repositories and 
require consumers to clone `k8s.io/kubernetes` and reference the staging component
modules within that clone.

Pros:
* Removes the need to publish separate staging component repositories

Cons:
* Git SHAs from before 1.15 would not be available as go modules, and would need to continue being accessed via the separately published repositories,
and [vanity import meta tags](https://golang.org/cmd/go/#hdr-Remote_import_paths) do not allow splitting the location `go get` looks at based on version

* Pointing to a relative location for a module like k8s.io/api works well for references from within kubernetes/kubernetes,
but it's unclear how a consumer outside `k8s.io/kubernetes` would use a `replace` directive to let `go build` automatically locate the nested module
([vanity import meta tags](https://golang.org/cmd/go/#hdr-Remote_import_paths) do not allow anything other than a repository root when pointing at a VCS,
so we could not indicate "k8s.io/api is located at k8s.io/kubernetes//staging/src/k8s.io/api")

* The `k8s.io/kubernetes` repository is extremely large, and forcing a clone of it to pick up `k8s.io/api`, for example, is unpleasant

### Alternative versioning strategies

* switch to tagging major versions on every `kubernetes/kubernetes` release (similar to what client-go does), and use semantic import versioning.
This remains a possibility in the future, but requires more tooling and consumer changes to accommodate rewritten imports,
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
          * uses `go.mod`-referenced versions of transitive non-module-based dependencies (unless overridden by top-level module, or conflicting peers referencing later versions)
    * kubernetes tooling
      * requires rewriting all `k8s.io/{component}/...` imports to `k8s.io/{component}/vX/...` at the start of each release
      * requires updating code generation scripts to generate versioned imports for `k8s.io/{api,apimachinery,client-go}`, etc
    * compatibility implications
      * allows breaking go changes in each kubernetes "minor" release
      * no breaking go changes are allowed in a kubernetes patch releases (need tooling to enforce this)
    * allowed versioning changes
      * modules published this way would have to continue using semantic import versioning
      * modules published this way could switch to incrementing major/minor versions at a difference cadence as needed

* tag major/minor versions as needed when incompatible changes are made, and use semantic import versioning.
This remains a possibility in the future, but requires more tooling and consumer changes to accommodate rewritten imports,
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
          * uses `go.mod`-referenced versions of transitive non-module-based dependencies (unless overridden by top-level module, or conflicting peers referencing later versions)
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
* https://github.com/golang/go/wiki/Modules, especially:
  * https://github.com/golang/go/wiki/Modules#semantic-import-versioning
  * https://github.com/golang/go/wiki/Modules#how-to-prepare-for-a-release
* https://golang.org/cmd/go/#hdr-The_go_mod_file
* https://golang.org/cmd/go/#hdr-Maintaining_module_requirements
* https://golang.org/cmd/go/#hdr-Module_compatibility_and_semantic_versioning
* [discussion of tagging with v0.x.y](https://github.com/kubernetes/kubernetes/issues/84608)