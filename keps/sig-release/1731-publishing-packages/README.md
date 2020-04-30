# Publishing kubernetes packages

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha -> Beta Graduation](#alpha---beta-graduation)
    - [Beta -> GA Graduation](#beta---ga-graduation)
    - [Removing deprecated publishing artifacts](#removing-deprecated-publishing-artifacts)
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

This document describes how deb & rpm packages get published as part of cutting a release, with the tooling the release and patch management teams have at hand ([anago], [gcbmgr] and others from [k/release], [k/k/build] and potentially other places).

[anago]: https://github.com/kubernetes/release/tree/master/anago
[gcbmgr]: https://github.com/kubernetes/release/tree/master/gdcbmgr
[k/release]: https://github.com/kubernetes/release
[k/k/build]: https://github.com/kubernetes/kubernetes/tree/master/build

## Motivation

Currently ...
- Only Googlers can publish packages.
- Package spec file updates are not committed to a public repository.
- The packages get published on Google infrastructure.
- After publishing a new release but before sending out the release
  notification emails the process needs to be paused. Googlers need to build
  and publish the deb and rpm packages before the branch management team can
  continue and send notification can be sent out.
- We can only publish packages for stable releases right now.
- We use different packages in CI then we officially release.

This all prolongs the release process, it is a hard dependency on a small group of people from one company (and its infrastructure), and we only ever publish and test packages very late in the release process.

### Goals

The whole process should be folded into the release tooling, it should be part of the release process, and should not involve anyone other than the release branch / patch release team.
For each release the release team cuts, packages should also be generated and published automatically.

There should be multiple channels users can subscribe to: **stable**, **dev**, and **nightly**.


### Non-Goals

The actual package generation is a different problem that is discussed in [this KEP][pkg-gen-kep].

[pkg-gen-kep]: https://github.com/kubernetes/enhancements/pull/858

## Proposal

- Make the infrastructure generic and simple enough to be easily handed off to the CNCF
    - Storage buckets (to store the staged/released packages)
    - DNS entries (e.g. apt.kubernetes.io, ...)
    - package mirror (e.g. a self hosted aptly/artifactory/... or as a service)
        - have multiple channels, e.g. `stable`, `dev`, `nightly`
- Run the [package building][pkg-gen-kep] as part of the staging process
    - on GCB / not on an individual's machine
- Have a safe way to store the signing key and make it available to the release team and release tooling
- Automatically sign the repository and packages
- Automatically build and publish packages on a nightly basis


### User Stories

*Note*: The following user stories are using the keywords of the [gherkin language][gherkin].

[gherkin]: https://docs.cucumber.io/gherkin/reference/


```
Scenario: Enduser installs a kubelet from the stable channel
  Given a user has configured the officially documented package mirror for stable releases for a specific kubernetes minor version ("the minor") on their machine
   When they use the system's package manager to query the list of kubelet packages available (e.g. apt-cache policy kubelet)
   Then they see a list of all stable patch versions of the kubelet that stem from the minor and a preference to install the latest patch version of the kubelet
    But don't see any alpha, beta, rc or nightly releases of the kubelet from this specific kubernetes minor version
    And they don't see any packages of any other kubernetes minor release
```

```
Scenario: Release tools automatically publish new packages
  Given a release team member ran `./gcbmgr stage master --build-at-head --nomock`
    And a release team member ran `./gcbmgr release master --buildversion=${{VERSIONID}} --nomock`
   When a user inspects the officially documented deb and rpm kubernetes repositories for this specific kubernetes minor version
   Then they see the newly cut alpha releases published in the alpha channel only
```

```
Scenario: End users can get the public key the packages or the repository metadata is signed with
  Given a user has a system configured with not allowing unsigned untrusted package repositories
    And they have setup the officially documented repository for a specific kubernetes minor release
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


This means, that end-users can configure their systems’ package managers to use those different `${channel}`s of a kubernetes `${k8s_release}` for their `${dist}`.

A configuration for the package managers might look something like:

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


Ideally, publishing/promoting a package means to commit a change to a configuration file which triggers a "package promotion tool".
That tool ...
- manages which packages need to go into which `${channel}` for which `${dist}` of which `${k8s_release}`
- guard that by the packages checksum
- is able to promote a package from a bucket and also from a `${channel}` to the other
- work off of a declarative configuration

This tool does for packages what the [Image Promoter][img-promoter] tool does
for container images. Therefore, ideally, we can implement this work-flow as part
of the [Image Promoter][img-promoter] or at least use its libraries.


As soon as the [redirector] is in place the repositories should be mirrored and the [redirector] should be used.

[redirector]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-release/20190121-artifact-management.md#http-redirector-design
[img-promoter]: https://github.com/kubernetes/enhancements/blob/7a2e7c25ee3f2a50f2218557801fbd8dd79fd0f2/keps/sig-release/k8s-image-promoter.md

All architectures that are supported by the [package building tool][pkg-gen-kep] should be published.
This KEP suggests to start with publishing a single supported architecture
(e.g. `linux/amd64`) and extend that iteratively, when we verify that creating
all packages for all architectures is fast enough to be done as part of the
release process. If it turns out this step takes too long, we need to think
about doing the package building & publishing asynchronous to the release
process (see also: [Risks](#risks-and-mitigations)).

### Risks and Mitigations

- *Risk*: We don't find a proper way to share secrets like the signing key  
  *Mitigation*: TBA
- *Risk*: Building all the packages for all the distributions and their version takes too long to be done nightly or via cutting the release  
  *Mitigation*: TBA

## Design Details

### Test Plan

There should be a post-publish tests, which can be run as part or after the release process
- pull packages from the  official mirrors (via the [redirector] if in place)
- assert that all the packages we expect to be published are actually published
- assert that the packages and the repo metadata is signed with the current signing key

### Graduation Criteria

In general we can keep the current way of publishing (via googlers onto google's infrastructure) and introduce new infrastructure in parallel.

Once the tests show that the mirrors are good, we can adapt the official documentation. This includes:
- for release team members:
  - How and where do the packages get published as part of the release process
  - How can the post-publish test be run
- for kubernetes contributors:
  - How and where do the nightlies get published
- for kubernetes users:
  - Which repository are available for users
  - How to configure their package managers




#### Alpha

- Needed infrastructure is in place (buckets, dns, repos, …)

#### Alpha -> Beta Graduation


- [anago] [creates packages][pkg-gen-kep] and published those packages as part of the release process
- post-publish tests are in places and run as part of the release process
- nightlies will be build and published on a daily basis
- documentation is in place

#### Beta -> GA Graduation


This new publishing infrastructure and mechanisms can be considered GA when no googler is needed anymore to publish the packages.

#### Removing deprecated publishing artifacts

When 2 releases have been cut and published with the new mechanism any of the older tools and processes (e.g. `k/release/{deb,rpm}`) can be removed.

N/A

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

TBA
