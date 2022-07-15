# Publishing kubernetes packages

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
- [Infrastructure Needed](#infrastructure-needed)
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
  - Storage buckets (to store the staged/released packages) or anything similar
  - DNS entries (e.g. apt.kubernetes.io, ...)
  - package mirror (e.g. a self hosted aptly/artifactory/... or as a service)
    - have multiple channels, e.g. `stable`, `dev`, `nightly`
- Run the package builds as part of [krel stage and release]
- Have a safe way to store the signing key and make it available to the release team and release tooling
- Automatically sign the repository and packages
- Automatically build and publish packages on a nightly basis

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

Packages will be published for different:

- Package managers, like `apt` (consumes deb packages) and `yum` (consumes rpm packages).
- **`${k8s_release}`**: the version of kubernetes `<major>.<minor>`
  (e.g. `1.12`, `1.13`, `1.14`, ...)
- **`${channel}`**: can be `stable`, `dev`, `nightly`
  - `stable`: all official releases for `${k8s_release}`
    (e.g.: `1.13.0`, `1.13.1`, `1.13.2`, ...)
  - `dev`: all development releases for all minor releases in this `${k8s_release}`, including `alpha`s, `beta`s and `rc`s
    (e.g.: `1.13.0-rc.2`, `1.13.2-beta.0`, `1.13.1-alpha.3`, ...)
  - `nightly`: any package cut automatically on a daily basis (optionally)

This means, that End Users can configure their systems’ package managers to use
those different `${channel}`s of a kubernetes `${k8s_release}` for their
corresponding package manager.

A configuration for the package managers might look something like:

- deb:
  ```
  # deb http://apt.kubernetes.io ${k8s_release} ${channel}
  deb [signed-by=/etc/keyrings/kubernetes-keyring.gpg] http://apt.kubernetes.io/debian 1.26 nightly
  ```
- rpm/yum:
  ```
  [kubernetes]
  name=Kubernetes
  # baseurl=http://yum.kubernetes.io/${k8s_release}/${channel}
  baseurl=http://yum.kubernetes.io/fedora/1.26/nightly
  enabled=1
  gpgcheck=1
  repo_gpgcheck=1
  gpgkey=file:///etc/pki/rpm-gpg/kubernetes.gpg.pub
  ```

Different architectures will be published into the same repos, it is up to the package managers to pull and install the correct package for the target platform.

Ideally and optionally, publishing/promoting a package means to commit a change
to a configuration file which triggers a "package promotion tool", which:

- manages which packages need to go into which `${channel}` for which package manager of which `${k8s_release}`
- guard that by the packages checksum
- is able to promote a package from a bucket and also from a `${channel}` to the other
- work off of a declarative configuration

This tool does for packages what the [Image Promoter][img-promoter] tool does
for container images. Therefore, ideally, we can implement this work-flow as part
of the [Image Promoter][img-promoter] or at least use its libraries.

[img-promoter]: https://github.com/kubernetes/enhancements/blob/7a2e7c25ee3f2a50f2218557801fbd8dd79fd0f2/keps/sig-release/k8s-image-promoter.md

We can achieve the same directly in `krel release`, means that the dedicated
promotion is an optional part of this KEP. As an intermediate solution we can
also leave the package publishing on the Google side and focus on building them
before graduating the KEP to GA.

All architectures that are supported by the [package building tool][pkg-gen-kep] should be published.
This KEP suggests to start with publishing a single supported architecture
(e.g. `linux/amd64`) and extend that iteratively, when we verify that creating
all packages for all architectures is fast enough to be done as part of the
release process. If it turns out this step takes too long, we need to think
about doing the package building & publishing asynchronous to the release
process (see also: [Risks](#risks-and-mitigations)).

### Risks and Mitigations

- _Risk_: We don't find a proper way to share secrets like the signing key
  _Mitigation_: Using a third party tool like 1Password
- _Risk_: Building all the packages for all the distributions and their version takes too long to be done nightly or via cutting the release  
  _Mitigation_: We do not deliver nightly packages.

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

- Needed infrastructure is in place (buckets, DNS, repos, …)
- There is a documented process to create and publish deb and rpm packages of Kubernetes components
- It is possible to consume the published deb and rpm packages using steps similar to the documented process

#### Alpha -> Beta Graduation

- [ ] [krel] creates deb and rpm packages of Kubernetes components
- [ ] Packages are signed
- [ ] Repository indices are updated after packages are copied to the repositories
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

## Infrastructure Needed

New infrastructure is required to manage keys used to sign the deb and rpm
artifacts so as to remove the dependency on existing infrastructure and
personal.
