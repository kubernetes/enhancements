# Breaking apart the kubernetes test tarball

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Internal structure of the test tarball](#internal-structure-of-the-test-tarball)
    - [Binary artifacts](#binary-artifacts)
    - [Portable sources](#portable-sources)
  - [Updating dependencies on <code>kubernetes-test.tar.gz</code>](#updating-dependencies-on-)
    - [Dependencies outside the Kubernetes organization](#dependencies-outside-the-kubernetes-organization)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [References](#references)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

- [x] [k/enhancements issue in release milestone and linked to KEP](https://github.com/kubernetes/enhancements/issues/714)
- [x] KEP approvers have set the KEP status to `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [x] "Implementation History" section is up-to-date for milestone
- [x] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

The Kubernetes release artifacts include a "mondo" test tarball which includes both
"portable" test sources (such as shell scripts and image manifests) as well as
platform-specific test binaries for all supported client, node, and
server platforms.

This KEP proposes replacing the monolith test tarball with platform-specific
tarballs, matching the existing pattern used for the client, node, and server
tarballs.

## Motivation

As the number of supported client, server, and node platforms has increased, the
size of the test tarball has increased as well. As of Kubernetes v1.13.2, the
official `kubernetes-test.tar.gz` is approximately 1.2GB; previous releases
ranged from 1.3 - 1.5GB. Several years ago,
[there were complaints](https://github.com/kubernetes/kubernetes/issues/28435)
that the "full" `kubernetes.tar.gz` release tarball was too big at 1.4G.
Many of the motivations for breaking up that tarball echo into this proposal.

The Bazel effort is another driving motivation. It's possible to build all
release artifacts solely using Bazel, and there is progress being made on
supporting cross-compilation of binary artifacts, but combining multiple
target platforms in one Bazel call is currently not well-supported.
Separating this tarball would make it easier to ensure that we can use Bazel t
produce identical artifacts as the non-Bazel build.

### Goals

- The Kubernetes test artifacts are broken apart such that users only need to
  download binaries for the platforms they're using.
- Be largely invisible to most developers: everything should just keep working.

### Non-Goals

- Changing the underlying build system. Both the make/shell-based build
  system and the Bazel-based build system will be affected, but users can
  continue to use their existing workflow.
- Removing cruft from the test tarballs. Likely, there are binaries and portable
  sources no longer being used anywhere, but we won't prune them with this
  effort.
- Changing what is released independent of the test tarball; i.e. whether we
  should make test binaries able to be downloaded directly from GCS.

## Proposal

Instead of building and distributing a single `kubernetes-test.tar.gz` with all
portable sources and compiled binaries for all platforms, produce several
platform-specific tarballs, one for each platform defined in
[`KUBE_TEST_PLATFORMS`](https://github.com/kubernetes/kubernetes/blob/193f659a1cd454b93cbe1e7b1f13b77c21783461/hack/lib/golang.sh#L150-L160):

- `kubernetes-test-linux-amd64.tar.gz`
- `kubernetes-test-linux-arm.tar.gz`
- `kubernetes-test-linux-arm64.tar.gz`
- `kubernetes-test-linux-s390x.tar.gz`
- `kubernetes-test-linux-ppc64le.tar.gz`
- `kubernetes-test-darwin-amd64.tar.gz`
- `kubernetes-test-windows-amd64.tar.gz`

### Internal structure of the test tarball

At present, the Kubernetes test tarball has several components, all
rooted under a `kubernetes/` top-level directory.

#### Binary artifacts

The test binary artifacts are currently organized into directories divided by platform:

- `platforms/`
  - `darwin/`, `linux/`, `windows/`
    - `amd64/`, `arm/`, `arm64/`, `ppc64le/`, `s390x/`

For comparison, the existing platform-specific tarballs
(such as `kubernetes-client-linux-amd64.tar.gz`) place all binaries under
a constant path with no platform information: `kubernetes/client/bin/kubectl`.

Scripts (such as `cluster/get-kube-binaries.sh`) [extract these tarballs
back into platform-specific directories](https://github.com/kubernetes/kubernetes/blob/193f659a1cd454b93cbe1e7b1f13b77c21783461/cluster/get-kube-binaries.sh#L143-L156)
to support downloading multiple platforms into a single workspace.

The test tarball should follow the lead of the other platform-specific tarballs
and place the binaries under `test/bin`. We can then reuse the existing
functionality already implemented for the other tarballs.

#### Portable sources

Portable sources are basically copied directly from the source tree:

- `test/e2e/testing-manifests/`
- `test/images/`
- `test/kubemark/`
- `hack/` ([partially](https://github.com/kubernetes/kubernetes/blob/193f659a1cd454b93cbe1e7b1f13b77c21783461/hack/lib/golang.sh#L193-L197))

We have two options for these:

1. Continue to distribute as a separate tarball, either `kubernetes-test.tar.gz`,
   or possibly something like `kubernetes-test-portable.tar.gz`.
- This makes the distinction very clear vs. the binary artifacts
- There's already some precedent, such as the `kubernetes-manifest.tar.gz` tarball
- It slightly complicates downloading of test dependencies
2. Duplicate these sources into each binary-specific tarball.
- Simplifies test dependency distribution - may only need to download one
  tarball if client and server are same platform
- Portable sources are small (as of v1.13.2, approximately 2.7MB uncompressed
  or about 186KB compressed) so duplication isn't a huge worry
- Complicates extraction of tarballs with existing scripts, since they assume
  everything is platform-specific

We propose the first option as slightly preferable given the tradeoffs.
Since we intend to continue distributing the mondo test tarball over a
deprecation period, we'll use the name `kubernetes-test-portable.tar.gz` for the
portable sources.

### Updating dependencies on `kubernetes-test.tar.gz`

Currently the CI workflows and `kubetest` use the `cluster/get-kube.sh` and
`cluster/get-kube-binaries.sh` scripts to download all artifacts, and
conveniently `get-kube-binaries.sh` is versioned with the release artifacts in
`kubernetes.tar.gz`, so simply making `get-kube-binaries.sh` aware of the new
tarballs should be sufficient for most CI and developer needs.

Because the test tarball includes binaries used both on the host running tests
(such as the `e2e` binary), as well as binaries which may run the nodes
(`e2e.node`), we would need to make sure to download binary test artifacts
targeting the host platform, node platform, and possibly server platform.

A quick search reveals a few other uses of `kubernetes-test.tar.gz`, mostly in
`cluster/`. We can update these to use the platform-specific tarballs, possibly
with a fallback to the mondo-tarball if worried about versioning.

#### Dependencies outside the Kubernetes organization

Searching GitHub for references to `kubernetes-test.tar.gz` largely returns
forks of the main kubernetes repository (including some very old forks,
identifiable by the script `e2e-from-release.sh`). Since these forks are not
likely to depend on upstream release artifacts, we can ignore them.

The Samsung SDS CNCT kraken-lib repository has a reference to `kubernetes-test.tar.gz`
in its [conformance test script](https://github.com/samsung-cnct/kraken-lib/blob/aceab16c316bafcdb1f542dc67876dd2e5279f6b/build-scripts/conformance-tests#L16),
but this repo is also marked deprecated and read-only, and there have been no
changes since July 2018.

In vmware/simple-k8s-test-env, the `sk8.sh` file uses
[`kubernetes-test.tar.gz`](https://github.com/vmware/simple-k8s-test-env/blob/master/sk8.sh#L4481),
and this repo seems actively maintained, so we should make sure this continues
working.

The [reference](https://github.com/knative/test-infra/blob/8ef3dc1c2ed07e64024bc68c9dbd1a2e10e9e975/scripts/e2e-tests.sh#L118-L120)
to `kubernetes-test.tar.gz` in knative/test-infra is hilarious.

There may be other uses that are not easily identifiable, so we will follow a
deprecation process of the mondo test tarball as described in the next section.

### Risks and Mitigations

It's hard to tell who uses these test tarballs outside the core project or
without tools like `kubetest`. We'll need to broadcast this change widely
so that any downstream users are aware of the incompatible changes.

As this is an inherently breaking change, we must decide when to
cause the break. Assuming this effort is targeted for the 1.14 release:

1. We can continue to produce a mondo-tarball for 3 releases, along with new
   split tarballs; i.e., both 1.14 and 1.15 would contain both split and mondo
   test tarballs, while 1.16 would only use a split test tarball. This way one
   could continue to use the mondo tarball through the 1.15 release cycle,
   and then switch to using split test tarballs for 1.16, as all supported
   releases would then be producing split test tarballs.
2. We can make a clean break for 1.14, not producing any mondo test tarballs.
   Downstream users would need to account for the break immediately,
   and would also need to special-case for older releases that still use the
   mondo test tarball.
3. A somewhat hybrid approach, mixing 1 and 2 backwards in time:
   a. Produce both mondo test tarballs and split test tarballs on master for
      a few weeks.
   b. Backport split tarballs to older releases still supported (1.11 through
      1.13), but continue to produce mondo test tarballs. We would never remove
      the mondo test tarballs from these releases, instead continuing to
      produce both.
   c. Update all test infrastructure to use split test tarballs
   d. Remove the mondo test tarball from 1.14 before the first beta release.

Given the Kubernetes [deprecation policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/),
we should go for option 1 and continue to distribute mondo and split
test tarballs for the 1.14 release, and possibly for several releases
thereafter. (It's not entirely clear exactly which deprecation policy applies
to this change, however.)

We'll mark the mondo test tarball as deprecated in the 1.14 release, both
through announcements in the release notes, as well as a `DEPRECATION` notice
in the mondo test tarball.

### Test Plan

We'll start by building both the mondo test tarball and split test tarballs in
CI, followed by updating test infrastructure to use the new split tarballs.
We will monitor TestGrid jobs to ensure that nothing is noticeably broken by
the change, and our primary sources will be those on the
[sig-release-master-blocking](https://testgrid.k8s.io/sig-release-master-blocking),
[sig-release-master-informing](https://testgrid.k8s.io/sig-release-master-informing),
and [sig-release-master-upgrade](https://testgrid.k8s.io/sig-release-master-upgrade)
dashboards.

We'll also reach out to community members testing on non-amd64 architectures,
since they're most likely to be impacted by this change.

We'll work with any downstream consumers we can find to update them to use the
split tarballs ahead of the 1.14.0 release, but will continue to support
the mondo test tarball through at least 1.14's complete lifecycle.

### Graduation Criteria

To consider this effort complete, we should no longer be distributing a
mondo-tarball of test artifacts, and all TestGrid dashboards should show a
similar level of greenness.

While ideally we'd make a clean break, removing the mondo-tarball at the same
time as we create the platform-specific test tarballs, to ensure a smoother
rollout we will distribute both the split and mondo test tarballs for a while,
and this effort will not be deemed complete until the mondo test tarball is
gone.

## References

Similar discussion and work on the other release tarballs:

- [Release tarballs are too big](https://github.com/kubernetes/kubernetes/issues/28435)
- [Build release tars per-architecture](https://github.com/kubernetes/kubernetes/issues/28629)
- [Stop including arch-specific binaries in kubernetes.tar.gz](https://github.com/kubernetes/kubernetes/pull/35737)
- [Implicitly call cluster/get-kube-binaries.sh](https://github.com/kubernetes/kubernetes/issues/38725)
- [kubernetes-dev announcement about removing arch-specific binaries from kubernetes.tar.gz "full" tarball](https://groups.google.com/d/msg/kubernetes-dev/n9H9I8TrOT4/1cyV5r9fAAAJ)
- [Recording](https://www.youtube.com/watch?v=WbqRursx39k&t=13m28s) and
  [notes](https://docs.google.com/document/d/1z8MQpr_jTwhmjLMUaqQyBk1EYG_Y_3D4y4YdMJ7V1Kk/edit#heading=h.1fpwoneimh52)
  from sig-testing weekly meeting

## Implementation History

- 2019-01-18: proposal on Slack and creation of the KEP
- 2019-01-28: KEP announced on sig-testing and sig-release mailing lists
- 2019-01-29: discussion at sig-testing weekly meeting
- 2019-02-14: implementation https://github.com/kubernetes/kubernetes/pull/74065
  created, deprecation notice included in mondo test tarball
- 2019-02-22: implementation https://github.com/kubernetes/kubernetes/pull/74065
  merged
- 2019-09-24: Stop building kubernetes-test.tar.gz: https://github.com/kubernetes/kubernetes/pull/83093
- 2021-08-16: Retroactive stable declaration
