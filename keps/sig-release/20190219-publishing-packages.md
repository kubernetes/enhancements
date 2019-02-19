---
title: Publishing kubernetes packages
authors:
  - "@hoegaarden"
owning-sig: sig-release
participating-sigs:
  - sig-cluster-lifecycle
reviewers:
  - "@timothysc"
  - "@sumitranr"
  - "@Klaven"
  - "@ncdc"
  - "@ixdy"
approvers:
  - "@spiffxp"
  - "@tpepper"
editor: TBD
creation-date: 2019-02-19
last-updated: 2019-02-19
status: provisional
see-also:
#  - "/keps/sig-cluster-lifecycle/creating-packages.md"
  - "/keps/sig-release/20190121-artifact-management.md"
  - "/keps/sig-release/k8s-image-promoter.md"
  - "/keps/sig-testing/20190118-breaking-apart-the-kubernetes-test-tarball.md"
---

# Publishing kubernetes packages

## Table of Contents

   * [Publishing kubernetes packages](#publishing-kubernetes-packages)
      * [Table of Contents](#table-of-contents)
      * [Release Signoff Checklist](#release-signoff-checklist)
      * [Summary](#summary)
      * [Motivation](#motivation)
         * [Goals](#goals)
         * [Non-Goals](#non-goals)
      * [Proposal](#proposal)
         * [User Stories](#user-stories)
         * [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
         * [Risks and Mitigations](#risks-and-mitigations)
      * [Design Details](#design-details)
         * [Test Plan](#test-plan)
         * [Graduation Criteria](#graduation-criteria)
         * [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
         * [Version Skew Strategy](#version-skew-strategy)
      * [Implementation History](#implementation-history)
      * [Drawbacks [optional]](#drawbacks-optional)
      * [Alternatives [optional]](#alternatives-optional)
      * [Infrastructure Needed](#infrastructure-needed)


<!--
[Tools for generating]: https://github.com/ekalinin/github-markdown-toc
-->

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

This document describes how deb & rpm packages get published as part of cutting a release, with the tooling the release and patch management teams have at hand ([anago], [gcbmgr] and others from [k/release], [k/k/build] and potentially other places).

[anago]: https://github.com/kubernetes/release/tree/master/anago
[gcbmgr]: https://github.com/kubernetes/release/tree/master/gdcbmgr
[k/release]: https://github.com/kubernetes/release
[k/k/build]: https://github.com/kubernetes/kubernetes/tree/master/build

## Motivation

Currently ...
- only Googlers can publish packages
- package spec file updates are not committed to a public repository
- the packages get published on Google infrastructure
- the release process needs to be paused between other artifacts have been released and the release is officially announced
- we can only cut packages for stable releases right now
- we use different packages in CI then we officially release

This all prolongs the release process, it is a hard dependency on a small group of people from one company (and its infrastructure), and we only ever cut and test packages very late in the release process.
We should change that.


### Goals

The whole process should be folded into the release tooling, it should be part of the release process, and should not involve anyone other than the release branch / patch release team.
For each release the release team cuts, packages should also be generated and published automatically.
Packages published for an alpha release should end up in different channels then the stable releases, i.e. consumers need to be able to subscribe to stable packages only.

Ideally, publishing/promoting a package means to change and commit a configuration, which triggers a tool similar to the [Image Promoter][img-promoter] that manages a list of published packages based on declarative configuration.
It needs to be able to handle multiple channels, and move packages from a storage bucket to a repository.

As soon as the [redirector] is in place the repositories should be mirrored and the [redirector] should be used.

[redirector]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-release/20190121-artifact-management.md#http-redirector-design
[img-promoter]: https://github.com/kubernetes/enhancements/blob/7a2e7c25ee3f2a50f2218557801fbd8dd79fd0f2/keps/sig-release/k8s-image-promoter.md

### Non-Goals

The actual package generation is a different problem that is discussed in [this KEP][pkg-gen-kep].

[pkg-gen-kep]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-cluster-lifecycle/xxxxxxxx-pending-packaging-kep.md

## Proposal

- Get infrastructure ready where the release team can push the packages to and is owned by the CNCF
    - Storage buckets (to store the staged/released packages)
    - DNS entries (e.g. apt.kubernetes.io, ...)
    - package mirror (e.g. a self hosted aptly/artifactory/... or as a service)
        - have multiple channels, e.g. `alpha`, `rc`, `stable`
- Run the [package building][pkg-gen-kep] as part of the staging process
    - on GCB / not on an individual's machine
- Have a safe way to store the signing key and make it available to the release team and release tooling
- Automatically sign the packages (for rpms)
- Automatically sign the reposiroty metadata (for debs & rpms)
- Automatically build and publish packages on a nighly basis


### User Stories

*Note*: The following user stories are using the keywords of the [gherkin language][gherkin].

[gherkin]: https://docs.cucumber.io/gherkin/reference/


```
Scenario: Enduser installs a kubelet from the stable channel
  Given a user has configured the officially documented package mirror for stable releases for a specific kubernetes minior version on their machine
   When they use the systems package manager
   Then they get the latest stable of the kubelet from this specific kubernetes minor version installed on the machine
    But don't get the latest beta of the kubelet from this specific kubernetes minor version installed on the machine
```

```
Scenario: Release tools automatically publish new packages
  Given a release team member ran `./gcbmgr stage master --build-at-head --nomock`
    And a release team member ran `./gcbmgr release master --buildversion=${{VERSIONID}} --nomock`
   When a user inspects the officially documented deb and rpm kubernetes repositories for this specific kubernetes minor version
   Then they see the newly cut alpha releases published in the alpha channel only
```

```
Scenario: Endusers can get all stable releases from the stable channel
  Given a user subsrcibed to the latest package repository for a specific kubernete minor release
   When they inspect the list of kubelet packages available
   Then they see all the patch release of this specific kubernetes minor release
    And they don't see any alpha, beta, rc or nightly releases
    And they don't see any packages of any other kubernetes minor release
```

```
Scenario: Endusers can get the public key the packages or the repository metadata is signed with
  Given a user has a system configured with not allowing unsigned untrusted package repositories
    And they have a setup the officially documented repository for a specific kubernetes minor release 
   When they download the public key from the location stated in the official documentation
    And they configure their system's package manager to use that key
    And they use their system's package manager to install a package from this specific kubernetes minor release
   Then their package manager will not complain about untrusted packages, sources or repositories
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

There should be the following options endusers should be able to use as repositories for their systems' package managers:

Packages will be published for different ...
- **`${dist}`**: a combination of `<distribution>-<code-name-or-version>`
  (e.g. `debian-jessie`, `ubuntu-xenial`, `fedora-23`, `centos-7`, ...)
- **`${k8s_release}`**: the version of kubernetes `<major>.<minor>`
  (e.g. `1.12`, `1.13`, `1.14`, ...)
- **`${channel}`**: can be `stable`, `dev`, `nightly`
    - `stable`: all official releases for `${k8s_release}`
      (e.g.: `1.13.0`, `1.13.1`, `1.13.2`, ...)
    - `dev`: all development releases for all minor releases in this `${k8s_release}`, including `alpha`s, `beta`s and `rc`s
      (e.g.: `1.13.0-rc.2`, `1.13.2-beta.0`, `1.13.1-alpha.3`, ...)
    - `nightly`: any package cut automatically on a daily basis

Therefore a configuration for package managers might look something like:

- deb:
    ```
    # deb http://apt.kubernetes.io/${dist} ${k8s_release} ${channel}
    deb http://apt.kubernetes.io/debian-jessie 1.13 nightly
    ```
- rpm/yum:
    ```
    [kubernetes]
    name=Kubernetes
    # baseurl=http://yum.kubernetes.io/${dist}/${k8s_release}/${channel}
    baseurl=http://yum.kubernetes.io/fedora-27/1.13/nightly
    enabled=1
    gpgcheck=1
    repo_gpgcheck=1
    gpgkey=file:///etc/pki/rpm-gpg/kubernetes.gpg.pub
    ```

Different architectures will be published into the same repos, it is up to the package managers to pull and install the correct package for the target platform.


Implementation steps:
- [ ] get minimal infra in place
    - [ ] buckets, dns, repos, ...
- [ ] create additional step in [anago] to generate packages on staging step [via the new method][pkg-gen-kep]
- [ ] create additional step in [anago] to publish packages to new repos on publish time
- [ ] get tests in place (see: [Test Plan](#Test-Plan))
- [ ] adapt official documentation to point to new repositories

> [color=#ff0000] **TODO**
> - how to manage shared secrets (sigining key), what do we need for that?
> - which distributions, platforms, ... we support is probably depending on:
>   - which packages [we can build][pkg-gen-kep]
>   - how long all those builds will take
> - probably starting out with current debian, ubuntu, fedora and RHEL versions?

### Risks and Mitigations

> [color=#ff0000] **TODO**

- *Risk*: We don't find a proper way to share secrets like the singing key*
  *Mitigation*: ...
- *Risk*: Building all the packages for all the distributions and their version takes to long to be done nightly or via cutting the release
  *Mitigation*: ...

<!--
What are the risks of this proposal and how do we mitigate.
Think broadly.
For example, consider both security and how this will impact the larger kubernetes ecosystem.

How will security be reviewed and by whom?
How will UX be reviewed and by whom?

Consider including folks that also work outside the SIG or subproject.
-->

## Design Details

### Test Plan

> [color=#ff0000] **TODO**

potential tests:
- ensure that alpha cuts don't get installed when subscribed to the stable channel
- tests that install/upgrade from the new repos (`{apt,rpm}.kubernetes.io`)
- tests that configuring both mirrors (`packages.cloud.google.com` and the new `{apt,rpm}.kubernetes.io`) works and does not break anything.
- ...

### Graduation Criteria

In general we can keep the current way of publishing (via googlers onto google's infrastructure) and introduce new infrastructure in parallel.

Once the tests show that the mirrors are good, we can adapt the official documentation.

<!--
**Note:** *Section not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. Initial KEP should keep
this high-level with a focus on what signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning),
or by redefining what graduation means.

In general, we try to use the same stages (alpha, beta, GA), regardless how the functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

#### Examples

These are generalized examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

##### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

##### Beta -> GA Graduation

- N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least 2 releases between beta and GA/stable, since there's no opportunity for user feedback, or even bug reports, in back-to-back releases.

##### Removing a deprecated flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality which deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include [conformance tests].**

[conformance tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/conformance-tests.md
-->

### Upgrade / Downgrade Strategy

> [color=#ff0000] **TODO**

<!--
Talk abount
- when to switch the documentation to the new mirrors
- when and if delete old mirrors/buckets/...
-->

<!--
If applicable, how will the component be upgraded and downgraded? Make sure this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to make use of the enhancement?
-->

### Version Skew Strategy

It needs to be possible to use both repos in parallel (the current `packages.cloud.google.com` and the new `{apt,rpm}.kubernetes.io`).


## Implementation History

<!--
- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->


## Drawbacks [optional]

<!--
Why should this KEP _not_ be implemented.
-->

## Alternatives [optional]

<!--
Similar to the `Drawbacks` section the `Alternatives` section is used to highlight and record other possible approaches to delivering the value proposed by a KEP.
-->


## Infrastructure Needed

- some buckets, owned by CNCF
    - how many, which ones for what usecase
- package repositories
    - either: self-hosted (e.g. aptly, artifactory, dpkg-scanpackages & push to public bucket, ... also of course similar for rpms)
    - or: a service to host the repositories (e.g. packagecloud, gemfury, ...)
- some DNS records
    - apt.kubernetes.io
    - yum.kubernetes.io
- some shared secrets storage
    - to hold the package signing key<Paste>
