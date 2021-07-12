---
title: Moving kubeadm out of kubernetes/kubernetes
authors:
  - "@neolit123"
owning-sig: sig-cluster-lifecycle
participating-sigs:
  - sig-cluster-lifecycle
  - sig-release
  - sig-docs
reviewers:
  - "@timothysc"
  - "@fabriziopandini"
  - "@rosti"
  - "@ereslibre"
  - "@detiber"
  - "@yastij"
  - "@LucaLanziani"
  - "@chuckha"
  - "@justaugustus"
  - "@tpepper"
approvers:
  - "@timothysc"
  - "@fabriziopandini"
editor: "@neolit123"
creation-date: 2019-12-29
last-updated: 2020-01-26
status: implementable
---

# Moving kubeadm out of kubernetes/kubernetes

## Table of Contents

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Actions](#actions)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Location](#location)
    - [Handling older kubeadm versions](#handling-older-kubeadm-versions)
    - [Releasing and tagging](#releasing-and-tagging)
      - [Challenges](#challenges)
      - [Building kubeadm from tarballs](#building-kubeadm-from-tarballs)
      - [Bazel vs go build](#bazel-vs-go-build)
      - [Syncing tags and branches between k/k and k/kubeadm](#syncing-tags-and-branches-between-kk-and-kkubeadm)
        - [k/k creates a new release-x.yy branch](#kk-creates-a-new-release-xyy-branch)
        - [k/k ‘s release-x.yy branch is fast-forwarded to master](#kk-s-release-xyy-branch-is-fast-forwarded-to-master)
        - [k/release starts building kubeadm version x.yy.](#krelease-starts-building-kubeadm-version-xyy)
        - [k/k pushes a new tag](#kk-pushes-a-new-tag)
    - [Dependency updates](#dependency-updates)
      - [Scenarios:](#scenarios)
        - [A bug in k8s.io/[some-library] is found and fixed in /staging or /vendor of k/k](#a-bug-in-k8siosome-library-is-found-and-fixed-in-staging-or-vendor-of-kk)
        - [The libraries in k/k/staging have a lot of changes since the last k8s release](#the-libraries-in-kkstaging-have-a-lot-of-changes-since-the-last-k8s-release)
        - [Kubernetes released with bug fixes in dependency X, that kubeadm did not include](#kubernetes-released-with-bug-fixes-in-dependency-x-that-kubeadm-did-not-include)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [A portion of the kubeadm backend changes and breaks users](#a-portion-of-the-kubeadm-backend-changes-and-breaks-users)
    - [The post-submit job for tagging/branching k/kubeadm fails](#the-post-submit-job-for-taggingbranching-kkubeadm-fails)
    - [The kubeadm go.mod includes outdated components](#the-kubeadm-gomod-includes-outdated-components)
    - [A unwanted commit ends up in a k/kubeadm branch before release](#a-unwanted-commit-ends-up-in-a-kkubeadm-branch-before-release)
    - [The kubeadm module does not use SEMVER](#the-kubeadm-module-does-not-use-semver)
- [Design Details](#design-details)
  - [Enhancements proposals](#enhancements-proposals)
  - [Documentation](#documentation)
    - [Authored content](#authored-content)
    - [Reference documentation](#reference-documentation)
    - [Release notes](#release-notes)
  - [Test Plan](#test-plan)
    - [CI artifacts](#ci-artifacts)
    - [PR testing](#pr-testing)
    - [Periodics jobs](#periodics-jobs)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Open Questions](#open-questions)
    - [Can we stop building CI DEBs and RPMs from k/k?](#can-we-stop-building-ci-debs-and-rpms-from-kk)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Building the k/kubeadm source code from k/k](#building-the-kkubeadm-source-code-from-kk)
  - [Prow plugin vs post-submit job vs custom bot](#prow-plugin-vs-post-submit-job-vs-custom-bot)
  - [Manual tagging and branching of k/kubeadm](#manual-tagging-and-branching-of-kkubeadm)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone,
which links to KEP (this should be a link to the KEP location
in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and
SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website],
for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents,
links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This document outlines the proposal for moving kubeadm outside of the
kubernetes/kubernetes repository into the kubernetes/kubeadm
repository.

A proposal was already done in the following Google Doc and it
received a healthy discussion:
https://docs.google.com/document/d/1xFAGAjfPQZrcqmapTwpJR5aUA7-2ku1asVeq3dBrjnQ

## Motivation

The monolith of kubernetes/kubernetes (k/k) is splitting and components
such as kubeadm should reside in designated repositories. Multiple
discussions during 2019 happened where the SIG Architecture Code
Organization sub project suggested that it is time for kubeadm to
move and that the kubeadm maintainers should start investigating
what has to be done for this to happen.

Moving kubeadm will benefit both kubernetes/kubernetes and the kubeadm
maintainers:
- Less noise in k/k from kubeadm issues and PRs.
- Make it easier for the kubeadm maintainers to track kubeadm PRs.
- No need to run the full suite of k8s pre-submit e2e jobs on kubeadm PRs.

### Goals

- Move the kubeadm source code to the k/kubeadm repository.
- Preserve the current kubeadm versioning, matching the k8s versioning.
- Continue packing kubeadm as part of the "node" tarball for Windows
and Linux.
- Continue having kubeadm test jobs as release-informing (-blocking in the future).
- Continue testing kubeadm using the existing set of tools - e.g. kinder.
- Continue to release kubeadm documentation as part of the k8s.io website
every k8s release.

### Non-Goals

- Decouple kubeadm from the k8s versioning. This can only create confusion
for the users and those depending on k8s/kubeadm version skew.
- Not releasing kubeadm artifacts for end users to consume.
- Use Bazel for building kubeadm once in the k/kubeadm repository.
For a smaller repository go tooling is enough and the effort required
for supporting different build methods is not justified.
- Refactor the kubeadm source structure to easily distinguish between
frontend and backend. This work is out of scope for this proposal.
- Use the "staging" pattern from k/k to gradually move kubeadm out.
kubeadm already does not depend on k/k and this pattern was seen as redundant.

## Proposal

### Actions

Once all implementation details are clear between kubeadm maintainers
and stakeholder SIGs such as Release and Docs, the following actions should
be taken in this exact order:
1) Update the following tracking issue to collect all
actions: https://github.com/kubernetes/kubeadm/issues/1949
2) Copy kubeadm to the k/kubeadm repository.
3) Implement build / hack / verify tooling for k/kubeadm.
4) Add pre-submit e2e jobs for k/kubeadm. Ideally there should be
two per branch - one that verifies / builds the code base
and one that does a full e2e using "kinder".
5) Implement tooling in the k/release repository to build kubeadm artifacts
outside of k/k.
6) Implement a post-submit job that will sync the k/kubeadm repository
tags and branches with those of k/k. Keep the job dry-running as a start.
7) Adapt the reference documentation tooling maintained by SIG Docs to build
from the kubeadm repository.
8) Announce the move on the kubernetes-dev mailing list.
9) Remove kubeadm and references to cmd/kubeadm from k/k.
10) Enable the post-submit job (disable its dry-run), so that new k/k
tags and branches can be synced in k/kubeadm.

### User Stories

#### Story 1

As a user, I would like to build kubeadm without depending on the
kubernetes/kubernetes repository.

kubeadm no longer imports k/k packages, so this is already possible
if one just copies cmd/kubeadm outside of k/k and creates a new Go
module for it. It still requires some hacks around applying a version in the
kubeadm binary. Once kubeadm moves out both the build tooling and module
will be available to users for easy building.

#### Story 2

As a user, I would like to import the kubeadm public, versioned API without
importing kubernetes/kubernetes.

Currently importing the k/k root module will make your build fail.
The best solution for users is to fork the kubeadm API packages.
Once kubeadm moves out, importing the "kubeadm" module will make
it possible to easily import only the API portion.

#### Story 3

As a user, I would like to import utility packages from kubeadm.

Similar to the above user story, importing kubeadm from k/k will fail.
Once kubeadm moves out, utility functions will be exposed for public use,
but perhaps without any guarantees until the kubeadm backend is truly
announced for public consumption.

### Implementation Details/Notes/Constraints

#### Location

The proposed new location is the kubernetes/kubeadm repository.

This makes it possible to use `go get k8s.io/kubeadm` to obtain
the kubeadm source tree.

The current contents of the `k/k/cmd/kubeadm/app` directory
should be moved in the root of k/kubeadm repository.

The current `k/k/cmd/kubeadm/test` directory that contains
integration tests should be merged with the `k/kubeadm/tests`
directory under `k/kubeadm/tests/integration`.

Existing tools in the k/kubeadm repository such as "kinder" and
"operator" must continue existing as separate modules (sub-folders)
inside the kubeadm module and can be obtained using
`go get k8s.io/kubeadm/some-tool`.

Existing docs and tests must co-exist in sub-folders too.

Side effects:
- All contents of k/kubeadm will be branched and tagged in a SemVer
compliant fashion, but this is actually a positive change.
- `hack/verify*` scripts for the root kubeadm folder should skip "kinder"
and "operator". This area can see improvements in the future,
potentially sharing scripts for all modules.

#### Handling older kubeadm versions

Currently kubeadm in the `release-1.17` branch (and later) of k/k
does not depend on the rest of k/k. This was required for kubeadm
to move out, however the older branches of k/k include versions of
kubeadm that still depend on k/k.

The proposed way forward is to move only the latest kubeadm from k/k master
and build older version of kubeadm from k/k.

This creates the following complications:
- The post-submit job must handle tags and branches for kubeadm versions
larger than X.
- The kubernetes/release tooling must build kubeadm version X from the
kubeadm repository while build older kubeadm from k/k.

The cherry-pick script from k/k/hack seems to work out of the box
for the kubeadm repository so potentially no forking will be required.
The same cherry pick policy will apply as in k/k.

#### Releasing and tagging

##### Challenges

The current plan is for kubeadm to continue to follow the k8s version scheme
and release schedule. This will continue to make the skew of kubeadm
and k8s easy to understand for the users and continue to give motivation
to the kubeadm developers to get work done on time.

This creates some complications:
- Automation tools have to build kubeadm as part of the release
- The k/kubeadm tags and branches need to follow those of k/k
- k/k tags are not SEMVER, which can cause troubles when importing
the k/kubeadm repository

Currently kubeadm is released as part of the following k8s artifacts:
- Node tarballs (Linux / Windows)
- deb/rpm packages

The kubeadm binary is built using either bazel (amd64 only) or using Docker
(cross-build, all supported architectures).

The binary is built with the same version as the rest of the released binaries
like the kubelet. The version tooling resides in k/k under the /hack directory
and it must be copied to k/kubeadm for consistency. The tooling that perform
the packaging and building are location in k/release and require manual interaction.

##### Building kubeadm from tarballs

For the initial proposal this will not be supported, but if there
are plans to distribute the kubeadm source code as part of a k8s release,
the solution would be to write a static KUBE_VERSION file in the tarball.

If GitHub users download the source code directly from the kubeadm repository
the build process must be able to detect that this is not a Git repository
and ask the users to provide a KUBE_VERSION file.

##### Bazel vs go build

Bazel was investigated as an option for building kubeadm from the new
repository but the benefits are unclear. BUILD files present a maintenance
challenge and also the bazel WORKSPACE and gazelle files need to be kept
in sync with k8s. On the other hand building using “go build” can take
around ~20 sec on a fast machine with fast internet.

The majority voted to not use bazel in the k/kubeadm repository.

If caching is required a build container can be used.

##### Syncing tags and branches between k/k and k/kubeadm

A brief discussion of the #prow maintainers revealed the proposal that
a post-submit job can be used for that instead of prow plugins.

Supposedly such a job will trigger each time a new branch is created
or if a tag is pushed. This means that the post-submit can trigger any bash
script or application to get the latest list of branches and tags in k/k
and apply the same to the kubeadm repository.

Given kubeadm will potentially move within a single cycle this means that
not all tags and branches have to be synced - see the options in “Versions
in the support skew”. So potentially there must be a minimum version
check in the post-submit job.

Scenarios:

###### k/k creates a new release-x.yy branch

The post-submit job checks out master in k/kubeadm, creates a new branch
release-x.yy and pushes the branch.

###### k/k ‘s release-x.yy branch is fast-forwarded to master

Currently this is a manual step performed by someone on the release team.
It might be something that the k/kubeadm maintainers have to do manually
on a daily basis. Automation is possible but is tricky and needs further
discussion. Can be done as a nice-to-have in the future.

###### k/release starts building kubeadm version x.yy.

There is a chicken and egg problem where a tag in the k/kubeadm repository
cannot be automatically added before the k8s release is cut.
This means that the build tools in k/release must not depend on looking
for a tag in the k/kubeadm repository when building kubeadm and
instead always build from the latest commit of a branch.

The k/release tooling must pull the k/kubeadm repository and check out
branch "release-x.yy". k/kubeadm build tooling must write a KUBE_VERSION
file with static versions, but a dynamic build date/time.
Kubeadm’s version information in the binary should be based on the
KUBE_VERSION file.

The "gitVersion" of the KUBE_VERSION file can still include information
about the offset from the last tag.

We have to note that if a k/kubeadm branch becomes stale, i.e. due to
no new commits for a certain release, multiple tags can end up on the
same commit and the k/kubeadm build tooling must be able to handle
that.

###### k/k pushes a new tag

The post-submit picks the change and based on the k/k tag version,
checks out the according branch in k/kubeadm, finds the latest commit
in the branch and tags it with a tag matching k/k tag.

#### Dependency updates

With the kubeadm moves out of k/k, the project will be decoupled
from the /vendor and /staging dependencies that it currently has in k/k.
This poses the question of how often the kubeadm project will have to
update its k8s.io/ dependencies in the go.mod file.

Example of a go.mod file for kubeadm:
https://github.com/neolit123/kubeadm/blob/66b94f324b63005589bab7132193e8031001ba57/kubeadm/go.mod

The current proposal is to manually update these dependencies right
after each release or during a release cycle if something is critical.
The k/k issue tracker and Slack will be the source of signal if something
has to be updated.

##### Scenarios:

###### A bug in k8s.io/[some-library] is found and fixed in /staging or /vendor of k/k

The kubeadm maintainers need to watch for such changes or bug reports
and update the kubeadm go.mod file. Running the change in CI will confirm
that it is not regressing.

###### The libraries in k/k/staging have a lot of changes since the last k8s release

The kubeadm maintainers need to pick commits to update the kubeadm go.mod
file to. Running the changes in CI will confirm that they are not regressing.


###### Kubernetes released with bug fixes in dependency X, that kubeadm did not include

Kubeadm should consider backporting a dependency update depending on the
severity of the issue.
If the issue is low severity evaluate if the the backport can be skipped.
If the issue is high severity consider a backport.

### Risks and Mitigations

#### A portion of the kubeadm backend changes and breaks users

As mentioned in "User story 3", the kubeadm maintainers
will give no guarantees about the kubeadm backend until it is
announced as stable. README files in the kubeadm repository
should denote that.

It is out of scope for this proposal to try to solve the problem.

#### The post-submit job for tagging/branching k/kubeadm fails

In cases the post-submit job fails, maintainers should be able
to apply the required changes manually or ask Prow maintainers
to re-trigger the job. Sufficient number of kubeadm maintainers
should have access and know how to perform these actions.

Ideally the post-submit job should be setup to send an email
notification on every failure at the SIG Cluster Lifecycle
mailing list with a custom `testgrid` suffix:
`kubernetes-sig-cluster-lifecycle+testgrid@googlegroups.com`
This suffix is already used by kubeadm periodic jobs.

Close to a release, coordination with SIG Release will be required,
to make sure that the exact, desired SHA of kubeadm is released.

#### The kubeadm go.mod includes outdated components

Once kubeadm is decoupled from k/k it will no longer benefit
from all the updates and security fixes that its components under
"staging" see. The imports in the kubeadm go.mod will need
to be updated on a regular basis to mitigate CVEs and critical bugs.

Tools such as https://docs.renovatebot.com/golang/ were
proposed. Automatic bulk updates can increase the noise in the
kubeadm repository but always make sure kubeadm depends on the
latest dependencies. Manual updates can get sloppy and miss critical
commits.

The current proposal is to rely on manual updates and make sure
kubeadm is always up to date before a final release, but also invest
time into automating this process.

Coordination with the security team and relevant SIGs might be
required.

#### A unwanted commit ends up in a k/kubeadm branch before release

Right before the post-submit job tags the k/kubeadm repository, but
after kubeadm was already build, there is a change that a new commit
can be added to a branch.

This commit will be tagged by the post-submit job, but will not
be the commit from which kubeadm was released.

Earlier iterations of this KEP had a process where we can pre-tag
the commit from which we built kubeadm with a placeholder such as
`pre-<version>` and then the tag can be updated to just `<version>`.

This proposal saw some objection due to the complexity of tagging
in k/release tooling.

Given kubeadm is fairly low-volume and mostly stale near release dates,
the maintainers do not have major concerns that this will be an issue.

#### The kubeadm module does not use SEMVER

Users importing the kubeadm module will find that the k8s versioning
is actually not SEMVER, because breaking changes happen on MINOR
changes instead of MAJOR ones.

A proposed solution here is to dual-tag commits with a version that
is SEMVER friendly - e.g.:
- `v1.17.0` -> `v0.17.0`

Dual-tagging with `v17.0.0` is not an option, because Go modules
will require the `v17` package.

But the problem remains with `v0.17.0`, because once k8s becomes
`v2.0.0`, the dual-tags (SEMVER) will overlap with the non-SEMVER tags.

There are no good solutions for this problem and users might have to
use SHAs to import the kubeadm module.

See the related discussions at:
https://github.com/kubernetes/kubernetes/issues/72638
https://github.com/kubernetes/kubernetes/issues/84372

## Design Details

### Enhancements proposals

The current plan is for kubeadm to continue following the k/enhancements
and KEP process, given the project will be part of the k8s release.
There will be no extra burden on the release team as they are already
following PRs and issues in repositories other than k/k.

There is also the option to decouple the enhancement process and have future
kubeadm KEPs in the k/kubeadm repository, but needs further discussion
and there are no apparent benefits.

### Documentation

#### Authored content

There are no planned changes for how kubeadm will release its authored
documentation. Kubeadm already has multiple documents in the k8s.io website
and those are maintained with the help of sig-docs.

#### Reference documentation

The reference documentation update process needs changes.
Currently tooling maintained by sig-docs does the following:
1) Clones k/k
2) Runs tools to generate cmd/kubeadm/app Cobra reference documentation
3) Prettyfies the documentation with HTML/CSS

The tooling needs to change to start generating the documentation
from the k/kubeadm repository. SIG docs must be notified of this
and collaborate with the kubeadm maintainers when the move is about to happen.

There would not be a need for the tooling to continue building old
reference documentation from k/k instead, as the k8s.io website reference
documentation is only generated once before a release happens.

The tools for building k/kubeadm reference docs in k/k must be removed.
The initial proposal is to not support building reference documentation
from the tools in the k/kubeadm repository, but might need some discussion.
Currently the consumers of k/k reference documentation are unknown and the
documentation is no longer tracked in commits. See:
https://github.com/kubernetes/kubernetes/tree/master/docs

#### Release notes

The prow “release-note” plugin must be enabled for the kubeadm repository.
This will make PRs require release notes or the option NONE. Once a k8s
release has begun ideally the tool:
https://github.com/kubernetes/release/tree/master/cmd/release-notes
should be levereged to obtain all the release notes for kubeadm in this
release.

The alternative is for the kubeadm maintainers to collect the release
notes manually and send them to the release team, but this might be more
difficult to handle as the k8s release tooling sends emails with the release
notes right away. Also the kubeadm maintainers might make mistakes forgetting
about a certain feature/bug fix.

### Test Plan

#### CI artifacts

Currently the kubeadm binary is built and uploaded in the CI GCS bucket
next to the rest of the k8sbinaries like kubelet.

There is a complication as now another repository has to be monitored
for changes. Currently the CI build happens after a PR merged in the
k/k repository.

The proposal is to always rebuild kubeadm based on the latest tips of
the master and release-x.yy branches of the k/kubeadm repository after
a CI build for the k/k repository triggers. In this case the k/release
build tooling must not trigger the writing of a KUBE_VERSION file in the
k/kubeadm repository but rather let the version to generate based on the
latest Git tags and SHAs.

#### PR testing

There is already support for running pre-submit jobs on k/kubeadm PRs.
Once kubeadm is moved to the new location a collection of PR blocking
jobs should be added to make sure the project is always buildable with
the current version of Go that the k/k repository uses. Syncing Go versions
should be the responsibility of the k/kubeadm maintainers.

The new PR blocking jobs must support building kubeadm from source with
the latest commits that a PR for k/kubeadm introduces. The rest of the
k8s artifacts can be pulled from the CI GCS bucket. This can also allow
catching problems introduced in k/k when using kubeadm as the deployer.

The /hack/verify-* and update-* scripts that k/kubeadm/kubeadm uses
will not be a direct copy of k/k but will provide similar results:
Go versions are in sync Go-fmt, linter and import tools are in sync
as much as possible

The following lists of tests will be executed on PRs as blocking:
- Unit tests
- Integration tests (currently reside in k/k/cmd/kubeadm/test)
- E2e tests from the e2e-kubeadm suite (currently resides
in k/k/test/e2e-kubeadm)
- Full cluster deployment e2e, possibly upgrade and skew tests too

#### Periodics jobs

No changes to the current kubeadm periodic jobs will be required
as long as the CI build process continues to push the kubeadm
and the rest of the k8s binaries to the GCS buckets.

### Graduation Criteria

In the 1.18 cycle there could be partial work related to copying the
kubeadm source code to the k/kubeadm repository and experimenting
with build tooling and the post-submit job.

The finalization of the move is planned for the release of 1.19.

### Upgrade / Downgrade Strategy

No changes to the upgrade strategy will be required.
Downgrades are not supported by kubeadm.

### Version Skew Strategy

No changes will be applied to the current version skew support in kubeadm
as long as kubeadm preserves its existing versioning, which matches
the versioning of k8s.

## Implementation History

- 2019-12-29: This proposal document was created

## Open Questions

#### Can we stop building CI DEBs and RPMs from k/k?

Currently the Kubernetes CI build process uses Bazel specs from the
k/k `/build` directory to build DEBs and RPMs. Packages are generated
for kubelet, kubeadm as the main targets. The Bazel specs assume kubeadm
exists under `/cmd/kubeadm` so this process has to be either changed
or completely removed in the 1.18 or 1.19 cycles.

Arguments for the removal of this process from k/k:

- The existence of the CI artifacts is not documented in kubeadm
and kubelet documentation.
- Observations by triaging issues in k/k and k/kubeadm in the past
couple of years have revealed no users of the CI DEBs and RPMs.
- DEBs and RPMs are also managed in k/release thus the k/k package
specs can end up out of date.

This KEP proposes that the k/k DEB and RPM CI build process is completely
removed from k/k.

This KEP does not propose changes to the process of building release
DEBs and RPMs in k/release. There is ongoing work in k/release to refactor
this same package building. Possibly CI DEBs and RPMs can be built from
there in the future if there is user demand.

## Drawbacks

If the post-submit job proposal ends up not working very well,
the maintainers will have to tag the k/kubeadm branches manually until
a better proposal for automation comes along.

Miscommunication of dependency updates is an issue as the kubeadm
maintainers can end up releasing kubeadm with e.g. critical bugs
in dependency X. Adding automated dependency validation and warnings can help
in this area.

## Alternatives

### Building the k/kubeadm source code from k/k

Instead of building the kubeadm source code from the tooling in k/release,
a _very viable option_ is to build it from the k/k repository build tooling.

The same tooling (present under /hack) is already responsible for the
build of other components and packaging of the same components in
tarballs and Docker images.

A new function can be executed that does the following:
- Clone k/kubeadm for version X in a temporary folder.
- Build the kubeadm binary from the temporary folder.
- Place the output binary in the location where the rest of the component
binaries are built.

### Prow plugin vs post-submit job vs custom bot

For the automation of branching and tagging, a prow plugin was initially
considered. After discussions with the Prow maintainers it was recommended
to use a post-submit job instead. If both approaches fail there is also
the possibility to use an external service (e.g. a custom bot), but such a bot
will need to have write access to the k/kubeadm repository too.

### Manual tagging and branching of k/kubeadm

As noted already, if no good solutions are found in terms of automatic
tagging and branching of the k/kubeadm repository the maintainers may have
to perform this operation manually. This is going to be time consuming, requires
write access to the repository (only a few people can do it). It also
requires communication with the release team and watching mailing lists for
release events.

## Infrastructure Needed

NONE
