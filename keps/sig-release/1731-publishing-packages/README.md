# Publishing kubernetes packages <!-- omit in toc -->

<!-- toc -->

- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [User Roles](#user-roles)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Using OBS instead of manually building and hosting packages](#using-obs-instead-of-manually-building-and-hosting-packages)
    - [How Open Build Service works?](#how-open-build-service-works)
    - [Packages, Operating Systems, and Architectures in Scope](#packages-operating-systems-and-architectures-in-scope)
    - [Repository Layout](#repository-layout)
    - [Projects and Packages in OBS](#projects-and-packages-in-obs)
    - [Packages in OBS](#packages-in-obs)
    - [Package Sources](#package-sources)
    - [Package Specs](#package-specs)
    - [Integrating OBS with our current release pipeline](#integrating-obs-with-our-current-release-pipeline)
    - [Authentication to OBS and User Management](#authentication-to-obs-and-user-management)
    - [How are packages used?](#how-are-packages-used)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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

## Summary

This document describes how deb and rpm packages get published in the same way as the currently [documented package mirror] as part of cutting a release, with the tooling the release management team has at hand ([krel] and others from [k/release], [k/k/build] and potentially other places).

[documented package mirror]: https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/#install-using-native-package-management
[gcbmgr]: https://github.com/kubernetes/release/tree/master/gdcbmgr
[k/release]: https://github.com/kubernetes/release
[k/k/build]: https://github.com/kubernetes/kubernetes/tree/master/build

## Motivation

Currently:

- Only Googlers can publish packages.
- The packages get published on Google infrastructure.
- After publishing a new release but before sending out the release notification
  emails the process needs to be paused. Googlers need to build and publish the
  deb and rpm packages before the branch manager can continue and send the
  release announcement.
- We can only publish packages for stable releases right now.
- We use different packages in CI then we officially release.

This all prolongs the release process, it is a hard dependency on a small group of people from one company (and its infrastructure), and we only ever publish and test packages very late in the release process.

### Goals

The whole process should be folded into the release tooling, it should be part of the release process, and should not involve anyone other than the release managers.
For each release the release team cuts, packages should also be generated and published automatically.

There should be multiple channels users can subscribe to: **stable**, **dev**, and **nightly**.

### Non-Goals

The original version of this KEP had the following statement:

> The actual package generation is a different problem that is discussed in [this KEP][pkg-gen-kep].

Based on a review of this KEP, Kubernetes contributor @saschagrunert wants SIG
Release to own the process of automatically building and publishing the packages
and this will implicitly achieve the main goal of
[#2503](https://github.com/kubernetes/enhancements/issues/2503).

[pkg-gen-kep]: https://github.com/kubernetes/enhancements/pull/858

## Proposal

- Make the infrastructure generic and simple enough to be easily handed off to the CNCF
  - Choose a Packages as a Service solution (e.g. Open Build Service...) or build
    the infrastructure manually. In case we build the infrastructure manually, we need
    at least:
    - Storage buckets (to store the staged/released packages) or anything similar
    - DNS entries (e.g. apt.kubernetes.io, ...)
    - Package mirror (e.g. a self hosted aptly/artifactory/... or as a service)
      - have multiple channels, e.g. `stable`, `dev`, `nightly`
- Run the package builds as part of [krel stage and release]
- Have a safe way to store the signing key and make it available to the release team and release tooling
  - Making key available to the Release Team/Managers is not a requirement if using 'as a service' solution
- Automatically sign the repository and packages
- Automatically build and publish packages on a nightly basis (not required)

[krel stage and release]: https://github.com/kubernetes/release/blob/master/docs/krel/README.md#usage

### User Stories

_Note_: The following user stories are using the keywords of the [gherkin language][gherkin].

[gherkin]: https://cucumber.io/docs/gherkin/reference/

#### User Roles

[Release Managers] are SIG Release Team Members that are

> ... Kubernetes contributors responsible for maintaining release branches and creating releases by using the tools SIG Release provides.

[release managers]: https://kubernetes.io/releases/release-managers

An [End User] is used in the CNCF sense to refer to consumers of the Kubernetes artifacts built and published by [Release Managers]

[end user]: https://www.cncf.io/enduser/

```
Scenario: End User lists the available packages from the stable channel
  Given an End User has configured the officially documented package mirror for stable releases on their machine
  When they use their system's package manager to query the list of kubelet packages available (e.g. apt-cache policy kubelet)
  Then they see a list of all stable patch versions of the kubelet and a preference to install the latest available patch version
  But they do not see any alpha, beta, release candidate or nightly releases of the kubelet package from this specific kubernetes minor version
  And they do not see any packages of any other kubernetes minor releases
```

```
Scenario: Release tools automatically publish new packages
  Given a Release Manager ran `./krel stage --nomock` and `./krel release --nomock` to create a new  release (alpha, beta, rc or final).
  When an End User inspects the officially documented deb and rpm kubernetes repositories for this specific kubernetes minor version
  Then they see the newly cut releases published in the appropriate channel only
```

```
Scenario: End users can retrieve the public key that signed the packages, or the repository metadata is signed with
  Given an End User configured their system to disallow the use of unsigned, untrusted package repositories
  And they have setup the officially documented repository for a specific kubernetes minor release
  When they download the public key from the location stated in the official documentation
  And they configure their system's package manager to use that key
  And they use their system's package manager to install a package from this specific kubernetes minor release
  Then their package manager does not complain about untrusted packages, sources or repositories
```

<!--
```
Scenario: [...]
  Given ...
    And ...
   When ...
   Then ...
```
-->

### Implementation Details/Notes/Constraints

Packages will be built and published using [Open Build Service (OBS)][obs]. openSUSE will sponsor the Kubernetes
project by giving us access to the [OBS instance hosted by openSUSE][obs-build].

[obs]: https://openbuildservice.org/
[obs-build]: https://build.opensuse.org/

#### Using OBS instead of manually building and hosting packages

The reasons for using Open Build Service (OBS) instead of building and hosting packages ourselves are:

- We want to handoff managing GPG keys to the third-party
  - Managing GPG keys ourselves represents a security risk. For example, if a Release Manager with access to the GPG key
    steps down, we might need to rotate the key. This is a process that affects End Users, therefore
    we want to avoid it
  - In this case, GPG keys are securely managed by the OBS platform hosted by openSUSE. No one from the Kubernetes
    project will have direct access to the key, mitigating one of the main risks of this proposal
- We want to avoid managing the infrastructure ourselves, including buckets, mirrors/CDNs...
- We want to provide 'as a service' access to the packages infrastructure to Release Managers and eventually other
  Kubernetes maintainers for their subprojects

#### How Open Build Service works?

From the [OBS website](https://openbuildservice.org/):

> The Open Build Service (OBS) is a generic system to build and distribute binary packages from sources in an automatic, consistent and reproducible way. You can release packages as well as updates, add-ons, appliances and entire distributions for a wide range of operating systems and hardware architectures.

OBS works in a way that we push sources and package spec files. Upon pushing packages/changes, OBS automatically
triggers builds for all chosen operating systems and architectures. Under the hood, OBS uses the same set of tools that
we use for building packages: `dpkg-buildpackage` and `rpmbuild`.

OBS implements a simple source-control management (SCM) system. It provides a complete history for all packages
allowing users to see what spec files and sources we used to build the concrete package. The history is accessible
via the OBS web interface.

Interaction with the OBS platform is done mainly via the [`osc` command-line tool][osc]. Alternatively, it's possible
to interact via the web interface. Currently there are no (Go) libraries that we can use instead of the `osc` tool.

[osc]: https://openbuildservice.org/help/manuals/obs-user-guide/cha.obs.osc.html

#### Packages, Operating Systems, and Architectures in Scope

We'll publish Debian-based (`deb`) and RPM-based (`rpm`) packages. Packages will be published for the following
architectures:

* `x86_64` (`amd64`)
* `aarch64`
* `armv7l` (`arm`)
* `ppc64le`
* `s390x`

We'll **build** packages for those architectures on the following Operating Systems:

- Ubuntu 20.04 (`aarch64`, `armv7l`, `ppc64le`, `s390x`, `x86_64`)
- CentOS Stream 8 (`aarch64`, `ppc64le`, `x86_64`)
  - CentOS Stream 8 doesn't support `armv7l` and `s390x` on OBS, hence, we use OpenSUSE Factory which does support
    those architectures
- OpenSUSE Factory (`armv7l`, `s390x`)

Those operating systems are used as **builders**. The resulting packages are supposed to work on any Debian-based
and RPM-based distribution, respectively. Those distributions were chosen based on recommendations from the OBS team
to provide the best compatibility with all all Debian-based and RPM-based distributions.

The following packages will be published for all operating system and architectures listed earlier. For simplicity,
we'll refer to those as packages as the **core packages**:

- cri-tools
- kubeadm
- kubectl
- kubelet
- kubernetes-cni

#### Repository Layout

We'll use this layout when creating repositories for the **core packages**:

- **`${channel}`**: can be `stable`, `dev`, `nightly`
  - `stable`: all official releases for `${k8s_release}`
    (e.g.: `1.26.0`, `1.26.1`, `1.26.2`, ...)
  - `dev`: all development releases for all minor releases in this `${k8s_release}`,
    including `alpha`s, `beta`s and `rc`s (e.g.: `1.26.0-rc.2`, `1.26.2-beta.0`, `1.26.1-alpha.3`, ...)
  - `nightly`: any package cut automatically from the `master` branch on a daily basis (optionally)
    - Nightly packages are currently out of scope and might be handled via a different KEP
- **`${k8s_release}`**: the version of Kubernetes `<major>.<minor>`
  (e.g. `1.12`, `1.13`, `1.14`, ...)

#### Projects and Packages in OBS

Speaking of OBS, **Packages** are located in **Projects**. We'll use two different types of projects:

- Building/Staging project - we'll build packages in a project of this type
- Publishing/Maintenance project - we'll publish packages from a project of this type

The reason for using two different types of projects is that OBS doesn't keep previously published packages.
For example, if we publish v1.26.0, and then v1.26.1, after v1.26.1 is published, the v1.26.0 package is removed.
To be able to keep all packages/versions (e.g. v1.26.0 and v1.26.1), we need to publish packages from the
maintenance project.

The repository layout mentioned earlier replicates in OBS as:

- The root OBS project is [**`isv:kubernetes`**](https://build.opensuse.org/project/show/isv:kubernetes)
- **`core`** subproject of the root project will be created to be used for **core packages**
  - In the future, we might decide to publish other packages (e.g. Minikube), so we want to have
    a proper and scalable layout from the beginning
- Each **`${channel}`** has a subproject of the **`core`** project (e.g. **`isv:kubernetes:core:stable`**)
- Each **`${k8s_release}`** has a **publishing** subproject of the **`${channel}`** subproject
  (e.g. **`isv:kubernetes:core:stable:v1.26`**)
  - We'll publish our packages from this project
- Each **`${k8s_release}`** has a **building** subproject as a subproject of the appropriate publishing subproject
  (e.g. **`isv:kubernetes:core:stable:v1.26:stage`**)
  - We'll build packages in this project
  
Having **`${k8s_release}`** subproject as a subproject of **`${channel}`** is required so we can build multiple
releases in parallel. This is because if changes are pushed to the package, the ongoing build process is aborted.
Running builds sequentially is not an option because that would slow down the release process too much.

Mentioned subprojects can be created manually via the web interface. Upon creating appropriate **`${k8s_release}`**
subprojects, the target operating systems and architectures (listed earlier) must be configured for those subprojects
(via the Repositories option). Additionally, some meta configuration is needed to declare the projects as publishing
and building. This is to be done by the Release Managers before cutting the first alpha release for that minor release.
The concrete configuration steps will be documented outside of this KEP, as part of the Release Managers Handbook.
We'll also consider automating this in some form.

#### Packages in OBS

**Package** object must be created in the **`${k8s_release}`** **building** subproject for each package that we want
to build and publish. This can be done via the `osc` command-line tool or the web interface. The created package 
inherits information about the target operating systems and architectures from the subproject.

Creating packages is to be done by the Release Managers before cutting the first alpha release for that minor release.
We'll consider automating this in some form.

Packages in the **publishing** subproject are created automatically for each published build. Those automatically
packages are named as `<package-name>.<timestamp>`, e.g. `kubectl.20230120135613`. This naming schema doesn't affect
package managers or users, i.e. those packages are still installable by their original name
(e.g. `apt install kubectl=1.26.0*`). More about building and publishing packages is explained in the next sections.

#### Package Sources

We'll build packages using pre-built binaries instead of pushing sources and then building binaries in the OBS pipeline.
The reasoning for this is:

- We already have our own release pipeline. Adding another release pipeline would increase the maintenance burden for
  Release Managers
- It would increase the effort for updating build dependencies such as Go
- It would increase the effort for validating correctness of created binaries
- Binary published by our release process would differ to binaries built by OBS
  - Additional efforts would be needed to get reproducible builds working
  - We would also lose cosign signatures for binaries built by OBS

`kubepkg` will be extended with a subcommand to create a tarball with all required binaries and files (e.g. systemd
units and config files). The tarball is supposed to be created with the maximum compression to save on bandwidth and
storage. The structure of the tarball is supposed to be:

- Root of tarball:
  - LICENSE file
  - README.md file
  - All accompanying files (e.g. systemd units)
  - Subdirectory for each target architecture:
    - Binary for that architecture (e.g. `kubectl`)

#### Package Specs

There are two key changes to the package specs compared to what specs we have at the time of writing this KEP:

- We'll maintain specs only for the RPM-based distros
- We'll have a dedicated spec for each package
  - Right now, for RPM-based distros, we have one spec file that builds all packages
  - This is to make it easier to maintain and update those spec files / packages, as well as, to make it easier for
    distributors to consume and use those spec files

The starting point for creating RPM specs is going to be the [RPM specs currently embedded in `kubepkg`][kubepkg-rpm].
The following changes are needed to those RPM specs:

- Parametrize specs so the build tooling is able to pick a binary for the correct target architecture
- Provide additional metadata needed for building Debian-based packages
- Ensure all spec files are passing rpm-lint

The reason for dropping deb specs is that maintaining and generating those specs is more complicated than
maintaining RPM specs. Considering that we use pre-built binaries, we can easily convert RPM specs to Debian specs
automatically using the [`debbuild` tool][debbuild]. The `debbuild` tool is already available in the OBS pipeline.
This tool can also be used by distributors if they want to build deb packages on their own.

The RPM specs will be generated by `kubepkg`, which already supports this. That said, we only need to update the spec
files.

[kubepkg-rpm]: https://github.com/kubernetes/release/tree/e10a44f8f9a9c08441260574e3d2a8711031fafe/cmd/kubepkg/templates/latest/rpm
[debbuild]: https://github.com/debbuild/debbuild

#### Integrating OBS with our current release pipeline

As described above, currently, it's up to the Release Manager to create the subproject and packages structure in OBS
before releasing the first alpha release. This should be the only manual steps required by Release Managers (besides
the user management, described below). We'll consider automating those steps in the future if possible.

The workflow for publishing packages is:

- Authenticate to OBS via `osc` and pull existing packages from the appropriate **building subproject**
- Update specs and generate the sources tarball using `kubepkg`
- Commit changes and push them to the **building subproject**
- Wait for packages to be built
  - There's [RabbitMQ][obs-rabbitmq] available that we can use to listen for events
  - This might not be feasible for all architectures, for example, building for `s390x` can take quite a while
- Once packages are built, release those packages by running `osc release`
  - Releasing (or publishing) packages means taking them from the **building subproject** and publishing them
    from the **publishing subproject**. This step makes packages available to end users
  - As described earlier, this ensures we keep previous builds and versions (e.g. both v1.26.0 and v1.26.1)

```mermaid
flowchart
    Authenticate --> Checkout[Checkout/Clone packages]
    Checkout --> Gen[Generate specs and sources]
    Gen --> Push[Push changes to OBS]
    Push --> Wait[Wait for builds]
    Wait --> Release[Release builds]
```

Ideally and optionally, publishing/promoting a package means to commit a change
to a configuration file which triggers a "package promotion tool", which:

- manages which packages need to go into which `${channel}` for which package manager of which `${k8s_release}`
- guard that by the packages checksum
- work off of a declarative configuration

This tool does for packages what the [Image Promoter][img-promoter] tool does
for container images. Therefore, ideally, we can implement this work-flow as part
of the [Image Promoter][img-promoter] or at least use its libraries.

[img-promoter]: https://github.com/kubernetes/enhancements/blob/7a2e7c25ee3f2a50f2218557801fbd8dd79fd0f2/keps/sig-release/k8s-image-promoter.md

We can achieve the same directly in `krel release`, means that the dedicated
promotion is an optional part of this KEP. As an intermediate solution we can
also leave the package publishing on the Google side and focus on building them
before graduating the KEP to GA.

At the time of writing this KEP, there are no Go libraries for working with OBS that we could use to integrate directly
with `krel`. Eventually, we could evaluate if it makes sense to build such a library for our purposes. Until then,
we'll use `osc` directly (by exec-ing), which also requires adding `osc` to our build images.

[obs-rabbitmq]: https://rabbit.opensuse.org/

#### Authentication to OBS and User Management

The concept of API tokens in OBS is very limited and provides access only to a very few endpoints. In other words,
it's not possible to use API tokens for publishing to OBS. Instead, we need to create some sort of a service account
to be used when publishing packages. This is one time operation that can be done by SIG Release Leads.

Users are managed manually via the OBS web interface. SIG Release Leads must have access to add/remove users from
our OBS project. Release Managers should be given read/write access, so they can maintain and create projects and
packages.

#### How are packages used?

The End Users can configure their systems’ package managers to use
those different `${channel}`s of a kubernetes `${k8s_release}` for their
corresponding package manager.

A configuration for the package managers might look something like:

- deb:
  ```
  # deb http://apt.kubernetes.io ${k8s_release} ${channel}
  deb [signed-by=/etc/keyrings/kubernetes-keyring.gpg] http://obs.kubernetes.io/core:/stable:/v1.26/deb/ /
  ```
- rpm/yum:
  ```
  [kubernetes]
  name=Kubernetes
  # baseurl=http://yum.kubernetes.io/${k8s_release}/${channel}
  baseurl=http://obs.kubernetes.io/core:/stable:/v1.26/rpm/
  enabled=1
  gpgcheck=1
  repo_gpgcheck=1
  gpgkey=file:///etc/pki/rpm-gpg/kubernetes.gpg.pub
  ```

Different architectures will be published into the same repos, it is up to the package managers to pull and install the correct package for the target platform.

### Risks and Mitigations

- _Risk_: The OBS installation provided by openSUSE is unable to serve the load generated by the Kubernetes project
  _Mitigation_: We can host our own mirrors and take some load from openSUSE (e.g. on Equinix Metal)
- _Risk_: Building all the packages for all the distributions and their version takes too long to be done nightly or via cutting the release  
  _Mitigation_: We do not deliver nightly packages or wait for packages to be published in the release pipeline.

## Design Details

### Test Plan

There should be post-publish tests, which can be run as part or after the release process

- pull packages from the official mirrors
- assert that all the packages we expect to be published are actually published
- assert that all published packages have the expected checksums
- assert that the packages and the repo metadata is signed with the current signing key
- assert that installed packages contain the correct binary versions

### Graduation Criteria

In general we can keep the current way of publishing (via googlers onto Google's
infrastructure) and introduce new infrastructure in parallel. For example, it is
still possible to optimize the current package building process by splitting up
the rapture script into a build script and sign/publish one. This would allow us
to intermediately automate the build step to publish artifacts on the release
GCS bucket. The sign/publish script would then be able to utilize those
artifacts and publish them into the current destination manually.

Once the tests show that the mirrors are good, we can adapt the official documentation. This includes:

- for release team members:
  - How and where do the packages get published as part of the release process
  - How can the post-publish tests be run
- for kubernetes contributors:
  - How and where do the nightly builds get published
- for kubernetes users:

  - Which repositories are available for users
  - How to configure their package managers

- There is a documented process to create and publish deb and rpm packages of Kubernetes components
- It is possible to consume the published deb and rpm packages using steps that document the process

#### Alpha

- Open Build Service is configured and ready to host packages
- Spec files are ready and can be used in OBS to bulid packages
- There is a documented process to create and publish deb and rpm packages of Kubernetes components
- It is possible to consume the published deb and rpm packages using steps similar to the documented process

#### Alpha -> Beta Graduation

- [ ] [krel] interacts with Open Build Service to automatically trigger package builds and publishing
- [ ] Packages are signed
- [ ] Post-publish tests are written and run as part of the release process
- [ ] Nightly builds will be built and published on a daily basis using [krel] which will be improved to take over this task from [kubepkg] making use of [pre-existing periodic jobs] (https://github.com/kubernetes/test-infra/blob/97cb34fa9e2bfc4af35de3e561cb9fc5a1094da1/config/jobs/kubernetes/sig-release/kubernetes-builds.yaml#L120-L166)
- [ ] Documentation written checked to be complete and correct.
- [ ] Outline deprecation policy and communication plan

[kubepkg]: https://github.com/kubernetes/release/tree/master/cmd/kubepkg
[krel]: https://github.com/kubernetes/release/blob/master/docs/krel/README.md

#### Beta -> GA Graduation

This new publishing infrastructure and mechanisms can be considered GA when no Googler is needed anymore to publish the packages.

- Removing remaining documentation for old Google hosted packages

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

N/A

## Implementation History

<!--
- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

TBA

## Drawbacks [optional]

N/A

## Alternatives [optional]

N/A
