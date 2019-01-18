---
kep-number: $i
title: Packaging of kubernetes artifacts
authors:
  - "@hoegaarden"
owning-sig: sig-release
participating-sigs:
  - sig-release
  - sig-cluster-lifecycle
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-01-18
last-updated: 2019-01-23
status: provisional
see-also:
  - ""
replaces:
  - ""
superseded-by:
  - ""
---

# Packaging kubernetes artifacts

## Table of Contents

* [Packaging kubernetes artifacts](#packaging-kubernetes-artifacts)
   * [Table of Contents](#table-of-contents)
   * [Release Signoff Checklist](#release-signoff-checklist)
   * [Summary](#summary)
   * [Motivation](#motivation)
      * [Goals](#goals)
      * [Non-Goals](#non-goals)
   * [Proposal](#proposal)
      * [Risks and Mitigations](#risks-and-mitigations)
   * [Design Details](#design-details)
      * [Test Plan](#test-plan)
      * [Graduation Criteria](#graduation-criteria)
         * [From nil to k/k, Introduce the new mechanism into k/k](#from-nil-to-kk-introduce-the-new-mechanism-into-kk)
         * [Deprecate all other options to build packages](#deprecate-all-other-options-to-build-packages)
         * [Make this the only available way to build packages](#make-this-the-only-available-way-to-build-packages)
      * [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
         * [Upgrade](#upgrade)
      * [Version Skew Strategy](#version-skew-strategy)
   * [Implementation History](#implementation-history)
   * [Drawbacks [optional]](#drawbacks-optional)
   * [Alternatives [optional]](#alternatives-optional)
   * [Infrastructure Needed [optional]](#infrastructure-needed-optional)

<!--
[Tools for generating]: https://github.com/ekalinin/github-markdown-toc
-->

<!--
**ACTION REQUIRED:** There must be an issue in
[kubernetes/enhancements](https://github.com/kubernetes/enhancements/issues)
referencing this KEP and targeting a release milestone **before [Enhancement
Freeze](https://github.com/kubernetes/sig-release/tree/master/releases) of the
targeted release**.
-->

## Release Signoff Checklist

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] k/enhancements issue in release milestone and linked to KEP (insert link here)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


## Summary

There are currently different ways to build packages (`deb`s and
`rpm`s) for the kubernetes components. All of which have some shortcomings and
it is hard to understand for users which one is the correct one.
The goal is to streamline all the package building mechanism, create one way on
how to generate packages, and retire all the other options.

Inspired by [RFC2119](https://tools.ietf.org/html/rfc2119) usage of the word
"must" in this document indicates *absolute requirements* while "should"
indicates *a nice to have* or *a feature to be considered*.

## Motivation

As mentioned in the summary, there are multiple different ways on how to create
packages for kubernetes:

- [`k/release/debian`][k-release-deb] & [`k/release/rpm`][k-release-rpm]
  - *Description*:  
    These tools are part of the official release process. After a new
    kubernetes version has been cut and released, but before the new release
    was announced, these tools will pull the build binaries from a bucket and
    wrap them in a `deb`/`rpm`.
  - *Usage*:
    ```
    k/release # ./debian/jenkins.sh
    k/release # cd ./rpm/ && ./docker-build.sh
    ```
  - *Cons*<a id="k-release-cons"></a>:
    - It uses a different (no) branching strategy then the artifacts build from `k/k` it is packaging
    - Pulls pre-existing artifacts from a GCS bucket
    - Cannot build just one package, builds all the packages for all architectures (on linux)
    - Needs manual change of the source code when bumping versions
  - *Pros*:
    - builds for multiple platforms (`linux/aarch64`, `linux/armhfp`, `linux/ppc64le`, `linux/s390x`, `linux/x86_64`)
    - also handles the upload of the packages (at least for `deb`s)
      - might also be considered a [con](#k-release-cons)?
  - *Parts to reuse*:
    - multi-arch support
- [`k/k/debs`][k-k-deb] & [`k/k/rpms`][k-k-rpm]
  - *Description*:  
    This is building the packages directly with bazel. Those packages are not
    used as official releases, they are however used in CI. The original use
    case was probably to replace the
    [packaging mechanism in `k/release`][k-release].
  - *Usage*:
    ```
    k/k # bazel build //build/debs
    k/k # bazel build //build/rpms
    ```
  - *Cons*:
    - No multi-arch support, builds only `linux/amd64`
    - currently rpm packaging needs `rpmbuild` to be installed on the host
      - Can be changed by dockerizing the packaging process, by making bazel
        download `rpmbuild` automatically or by some other means
    - `pkg_rpm` seems to be not actively maintained
  - *Pros*:
    - `bazel` integration
    - picks up proper version from git
    - package specs and additional files (systemd units, ...) are versioned in the same way
      as [k/k][k-k]
  - *Parts to reuse*:
    - some sort of `bazel` integration
    - versioning of the packaging specs and additional files alongside the code

[k-release]: https://github.com/kubernetes/release/tree/master/
[k-k]: https://github.com/kubernetes/kubernetes/tree/master/build/
[k-release-deb]: https://github.com/kubernetes/release/tree/master/debian
[k-release-rpm]: https://github.com/kubernetes/release/tree/master/rpm
[k-k-deb]: https://github.com/kubernetes/kubernetes/tree/master/build/debs
[k-k-rpm]: https://github.com/kubernetes/kubernetes/tree/master/build/rpms

To a good part this KEP tries to address most the [issue "k/k is the canonical source of all build artifacts"](https://github.com/kubernetes/kubernetes/issues/71677) and related
issues:
- [`kubeadm` package should depend on the same version of `kubelet` and
  `kubectl` packages](https://github.com/kubernetes/kubernetes/issues/72871)
- [Automate deb/rpm publishing](https://github.com/kubernetes/sig-release/issues/10)
- ...

There were also some discussion about or touching this KEP:
- [Kubernetes SIG Release deb rpm packaging discussion 20190123][vid-1]
  - including [meeting notes][notes-1]
- [20180124 sig cluster lifecycle packaging discussion][vid-2]

[vid-1]: https://www.youtube.com/watch?v=UxJR5zUdXWQ
[notes-1]: https://docs.google.com/document/d/1sSezV-vOSsmrL-Pm79ZJBPCL41KOTp4DJ8DwHgFaGMg/edit
[vid-2]: https://youtu.be/UyOLxD7ePCE

### Goals

- Fold the packaging specs, systemd unit files, dependency configuration, and
  "Parts to reuse" into one place in `k/k`.

  As a first step, the state from [`k/release/{rpm,debian}`][k-release] should
  be used, as those are the things we release right now. As a second step all
  the changes and additions from [`k/k/build/`][k-k] should be merged in.

  There must be **one** package configuration per package, all different
  packaging formats must use this as source of truth (a package configuration
  could be *as an example* a config file for a package-generator plus
  all the needed systemd unit files, ... )

- The tooling must be used for cutting packages in the official release
  process, it must also be usable in local development to generate one-off
  packages.

- <a id="pkg-list"></a>The tooling must support packaging
  - `kubelet`
  - `kubectl`
  - `kubeadm`
  - `cri-tools`
  - `kubernetes-cni`

  It should be easy to integrate new packages, especially when they are
  one-binary packages

- <a id="plat-list"></a>The tooling must support packaging for multiple platforms
  - `linux/amd64`
  - `linux/arm64`
  - `linux/armhf`
  - `linux/ppc64el`
  - `linux/s390x`

  It should be easy to integrate other platforms, given there is general
  support by the supported distributions and the cross-building tools.
  Examples: `kubectl` for `darwin/amd64`, `kubelet` for `windows/386`, ...

- The tooling must support building `deb`s and `rpm`s

  It should be possible to add new package types, e.g. NuGet for Windows

- The documentation for the process of building packages, local one-off builds
  and as part of the release process, must be excellent.

- Any other option to build packages must be retired eventually

  There should be documentation how and where the old packaging option can be
  found and how they can be used for cases where someone wants to build
  packages before the new tools were introduced.

- It must be equally easy to build packages from
  - the currently checked out revision (any revision after the tooling has been
    introduced into `k/k`)
  - a pre-build binary (pulling from some well-defined staging bucket or that like)

- The tools when run without special configuration must run as a container as
  opposed to on the host directly. The default behaviour is that it works
  everywhere where there is a docker engine which can run `linux/x86-64`
  container images.

### Non-Goals

- Uploading, publishing, ... of packages, there is a
  [different KEP][kep-publishing] discussing that
- Clean up [`k/k/build/`][k-k] (remove outdated scripts & documentation) other
  then "competitor" packaging mechanism

[kep-publishing]: https://github.com/kubernetes/enhancements/blob/bcaae7bb88eec844338d43fc05bd7384365413d6/keps/sig-release/20190121-artifact-management.md

## Proposal

All the following steps are only considered done when there is proper documentation on
- how to use those tools
- how they interact with each other
- how to (re-)build them
- how to update them
- how their configuration works

**Note**: This section refers to a couple of specifics that might be seen as
implementation details in here (e.g. references to a specific tools or specific
filesystem layouts). Those are just there to illustrate in more detail what we
aim towards, the details and actual tools can absolutely change in the actual
implementation.

- [ ] Define and document the package configuration

  The data referred to in this KEP as package configuration might be something
  like <a id="nfpm-config">a `nfpm` (compatible) configuration file</a> plus a set
  of additional files referred to in the [main configuration
  file](#nfpm-config), potentially organised in subdirectories

  ```sh
  ./main.conf
  ./linux/50-kubeadm.conf
  ./linux/amd64/50-kubeadm.conf
  [...]
  ```

- [ ] Create a tool that can create packages based on **one** package configuration
   - must be a container image
     - the Dockerfile must be committed to [`k/k`][k-k]
     - must be managed/promoted/... the same way as already existing container
       images in use in [`k/k`][k-k]
     - should be based on a existing build image which already holds a full
       cross-compile toolchain and other tools
   - must take the following inputs:
     - the package configuration
     - the pre-compiled binarie(s)
     - the version the binaries are compiled from
   - might take as input:
     - the platform(s) to package for (default: [all supported platforms](#plat-list))
   - must create the following outputs:
     - one package per platform (`linux/386`, `darwin/amd64`, ...) and package type (`rpm`,
       `deb`)
    - should be a thin wrapper around already existing tools like `nfpm`
    - must be discussed with [SIG Release][sig-release] and others if the tool
      we are going to wrap meets our needs and standard or if it is better to
      write it on our own and are happy to pay the maintenance costs

   Example run:
   ```bash
   docker build -t k8s-pkg-builder build/pkgs
   docker run \
     -v ${PWD}/_output/dockerized/bin:/in \
     -v ${PWD}/_output/pkgs/nightly:/out \
     -v ${PWD}/build/pkgs/kubeadm:/conf \
     -e KUBE_BUILD_VERSION='1.12.6~beta.0.1+12187918e930d4+dirty' \
     -e KUBE_BUILD_PLATFORMS='linux/amd64 linux/386' \
     k8s-pkg-builder
   ```

- [ ] Create all package configurations for all currently published
  [packages](#pkg-list) and [platforms](#plat-list)

  Should start with already existing specs from the [`k/release`][k-release] and
  merge in additional / new specs from [`k/k/build`][k-k-builds].

- [ ] The output of the above tool (a directory tree, holding a set of
  generated packages) ideally is in a format that can be uploaded to a GCS
  bucket which can be used as a debian package repo / yum repo.

  If that is not feasible there needs to be a tool that can re-arrange that
  directory and prepare it for upload. This might also give the opportunity to
  run certain tools to prepare the directory, e.g. [`dpkg-scanpacakges`,
  `debpool` or other).

- [ ] Create one tool that does "everything" that needs to be done in the
  release and can be called by [`anago`][anago].

  Without any special configuration is should do whatever is needed so that it
  can be called from within [`anago`][anago], however there should be options
  to specify which components to package, which components to package, get the
  artifacts by compiling them or pulling them from a staging bucket, ...

  This tool should be re-entrant and pick up where it left of in case it was
  aborted and started again.

  This might be seen as the
  "orchestrator tool" and has the following responsibilities:
  1. determinate the version to be packaged, based on the current git checkout
     ok `k/k` (a version like in `bazel-genfiles/build/version`)
  1. do a build (`build/run.sh make $COMPONENT` or such) or download all needed
     artifacts from from a staging bucket
  1. run the tool to prepare packages for upload
  1. upload the packages


[anago]: https://github.com/kubernetes/release/blob/master/anago

<!--
### User Stories [optional]

Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of the system.
The goal here is to make this feel real for users without getting bogged down.

#### Story 1

#### Story 2

### Implementation Details/Notes/Constraints [optional]

What are the caveats to the implementation?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they releate.
-->

### Risks and Mitigations

- *Risk*: creating another way on how to build packages  
  *Mitigations*:  
  - work with the current builder (@sumi) in google
  - writing super extra great docs so this is the goto way on how to build
- *Risk*: `nfpm` is not the right tool, it goes away, works differently, ...  
  *Mitigations*:  
  - we encapsulate the build of exactly one package, if things change, we can
    change the internal implementation but keep the external
    interface
- *Risk*: We cannot build proper windows packages but community wants/needs them  
  *Mitigations*:
  - Talk to people in sig windows, esp. about packaging their components
- *Risk*: What's the technical debt and maintenance cost of the tools and
  wrappers to be introduced?  
  *Mitigation*:
  - All tools need proper testing, so future refactors can be made in a safe
    manner
  - Proper languages, framework & abstractions must be used

## Design Details

### Test Plan

- different components will have unit tests as much as possible
- test jobs that build the artifacts and install them within containers.
  - ideally on various distributions
  - to start with, one linux distributions for `deb`s and one for `rpm`s
  - maybe to be expanded to multiple distributions or even different OSs

### Graduation Criteria

#### From `nil` to `k/k`, Introduce the new mechanism into `k/k`

- Iterate here on the KEP
- Do PoCs
- Compare packages build with the new tooling to packages generated with the
  [current release tooling][k-release]
- See no significant difference in the generated packages
- The new mechanism has means to prepare the packages in a way that they can be
  published via the current [`deb`][k-release-deb-publish] &
  [`rpm`][k-release-rpm-publish] publishing workflow

[k-release-deb-publish]: https://github.com/kubernetes/release/tree/63eaf48a52722144d232e1d017eb6b2b688fc6be/debian
[k-release-rpm-publish]: https://github.com/kubernetes/release/blob/master/rpm/docker-build.sh

#### Deprecate all other options to build packages

Deprecation of other packaging mechanisms is done either by a note in the
README or even in the source code to print a warning when building packages via
deprecated tools.

Before deprecation the following points should be implemented:

- Make it easy for the release team to switch between the old and the new process
- Make it easy for the release team to compare two (sets of) packages and
  assess if they are sufficiently similar
- Change the release (branch management & patch management) processes to use
  the new mechanism by default

#### Make this the only available way to build packages

- 2 releases have been cut with the new mechanism
  - must to be discussed with [SIG Release][sig-release]
- There was no feedback on bad packages

[sig-release]: https://github.com/kubernetes/sig-release

### Upgrade / Downgrade Strategy

#### Upgrade

N/A

<!--
**Note:** *Section not required until targeted at a release.*

If applicable, how will the component be upgraded and downgraded?  Make sure this is in the test plan.
-->

### Version Skew Strategy

One of the issues we saw is that `k/k` has release branches, while `k/release`
didn't have the same branch/release strategy.
This is fixed by adding the new tooling and package building specs to `k/k`
directly, it will be versioned alongside the things it will package.

## Implementation History

<!--
Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks [optional]

<!-- Why should this KEP _not_ be implemented. -->

## Alternatives [optional]

<!-- Similar to the `Drawbacks` section the `Alternatives` section is used to highlight and record other possible approaches to delivering the value proposed by a KEP. -->

## Infrastructure Needed [optional]

none
