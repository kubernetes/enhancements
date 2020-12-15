---
title: kubernetes-csi release process
authors:
  - "@pohly"
owning-sig: sig-storage
participating-sigs:
  - sig-testing
  - sig-release
reviewers:
  - "@msau42"
approvers:
  - "@msau42"
  - "sig-testing: TBD"
  - "sig-release: TBD"
editor: "@pohly"
creation-date: 2019-02-04
last-updated: 2019-02-04
status: provisional
---

# kubernetes-csi-release-process

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Versionioning](#versionioning)
    - [Release artifacts](#release-artifacts)
    - [Release process](#release-process)
  - [Implementation Details](#implementation-details)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Summary

The Kubernetes [Storage
SIG](https://github.com/kubernetes/community/tree/master/sig-storage)
maintains a set of components under the
[kubernetes-csi](https://github.com/kubernetes-csi) GitHub
organization. Those components are intentionally not part of core
Kubernetes even though they are maintained by the Kubernetes project.

This document explains how these components are released.

## Motivation

So far, the process for tagging and building components has been
fairly manual, with some assistance by Travis CI. There has been no
automatic testing of the release candidates on a Kubernetes cluster,
neither as stand-alone deployment of individual components nor as full
end-to-end (E2E) testing of several Kubernetes-CSI components. Testing
the stable releases of the components is included in the automatic
Kubernetes testing, but running those tests against pre-release
components had to be done manually.

With CSI support reaching GA in Kubernetes 1.13, it is time to define
and follow a less work-intensive and more predictable approach.

### Goals

- define the release process for each component
- unit testing as mandatory pre-submit check for all components
- testing on Kubernetes as mandatory pre-submit check for components that get
  deployed in a cluster

### Non-Goals

- a combined "Kubernetes-CSI release": each component gets released
  separately.  It is the responsibility of a CSI driver maintainer
  to pick and test sidecar releases for a combined deployment of that
  driver. The [hostpath deployment
  example](https://github.com/kubernetes-csi/csi-driver-host-path/tree/master/deploy)
  can be used as a starting point, but it's not going to be able to
  test all the possible combinations of features that a specific CSI
  driver may need.
- change responsibilities among the kubernetes-csi maintainers: as before, each
  component will have some main maintainer who is responsible for releasing updates
  of that component
- automatically generate release notes: this will be covered by a separate, future KEP


## Proposal

### Versionioning

Each of the components has its own documentation, versioning and
release notes. [Semantic versioning](https://semver.org/) with `v`
prefix (i.e. `v1.2.3`) is used for components that provide a stable
API, like for example:
- external-attacher
- external-provisioner
- external-snapshotter
- cluster-driver-registrar
- node-driver-registrar
- csi-driver-host-path
- csi-lib-utils

Other components are internal utility packages that are
not getting tagged:
- csi-release-tools

An additional suffix like `-rc1` can be used to denote a pre-release.

The [hostpath example
deployment](https://github.com/kubernetes-csi/csi-driver-host-path/tree/master/deploy)
defines the exact release of each sidecar that is part of that
deployment. When updating to newer sidecar releases, a new `csi-driver-host-path`
release is tagged with version numbers bumped according to the
same rules as the individual components (major version bump when any
of the sidecars had a major change, etc.). This will then also
rebuild the hostpath driver binary itself with that version number
embedded, regardless of whether its own source code has changed.

#### Release artifacts

Tagging a component with a `v*` tag triggers a
build for that component with that version. The output is primarily the container image
for components that need to be deployed. Those images get published as
`gcr.io/kubernetes-csi/<image>:<tag>`, which depends on [issue
158](https://github.com/kubernetes/k8s.io/issues/158) getting resolved.

Only binaries provided as part of a release with a semantic version tag without additional suffix
should be considered
production ready. Binaries are never going to be rebuilt, therefore an
image like `csi-node-driver-registrar:v1.0.2` will always pull the
same content and `imagePullPolicy: Always` can be omitted.

Eventually auxiliary files like the RBAC rules that are included in
each component might also get published in a single location, like
`https://dl.k8s.io/kubernetes-csi/`, but this currently outside of
this KEP.

#### Release process

1. A change is submitted against the master branch.

1. A [Prow](https://github.com/kubernetes/test-infra/blob/master/prow/README.md)
   job checks out the source of the modified component, rebuilds it and then
   runs unit tests and E2E tests with it as defined below.

1. Maintainers accept changes into the master branch.

1. The same Prow job runs for master and repeats the checks. If it succeeds,
   a new "canary" image is published for the component.

1. In preparation for the release of a major new update, a feature freeze is
   declared for the "master" branch and only changes relevant for that next
   release are accepted.

1. When all changes targeted for the release are in master, automatic
   test results are okay and and potentially some more manual tests,
   maintainers tag a new release directly on the master branch. This
   can be a `-rc` test release or a normal release.

1. Maintenance releases are prepared by creating a "release-X.Y" branch based on
   release "vX.Y" and backporting relevant fixes from master. The same
   prow job as for master also handles the maintenance branches.


### Implementation Details

For each component under kubernetes-csi (`external-attacher`,
`csi-driver-host-path`, `csi-lib-utils`, etc.), these Prow jobs need
to be defined:
- `kubernetes-csi-<component>-pr`: presubmit job
- `kubernetes-csi-<component>-build`: a postsubmits job that matches against
  `v*` branches *and* tags (see https://github.com/kubernetes/test-infra/pull/10802#discussion_r248900281)

In addition, for the `csi-driver-host-path` repo some more periodic
jobs need to be defined:
- `kubernetes-csi-stable`: deploys and tests the current hostpath
  example (`csi-driver-host-path/deploy/stable`) from the master
  branch on the latest Kubernetes development version
- `kubernetes-csi-canary-<k8s-release>`: deploys and tests the canary
  hostpath example (`csi-driver-host-path/deploy/canary`) from the
  master branch on a certain Kubernetes release (for example,
  `<k8s-release>` = `1.13`), using the same image revisions for the
  entire test run; initially we'll start with 1.13 and later will
  add more stable releases and remove unsupported ones
- `kubernetes-csi-canary-dev`: deploys and tests the canary hostpath
  example from the master branch on the latest Kubernetes development
  version

A periodic job that does regular maintenance tasks (like checking for
updated dependencies) might be added in the future.

These `kubernetes-csi` Prow job all provide the same environment where
the component is already checked out in the `GOPATH` at the desired
revision (PR merged tentatively into a branch or a regular
branch). This is provided by the
[podutils](https://github.com/kubernetes/test-infra/blob/master/prow/pod-utilities.md)
decorators. The base image is the latest
[kubekins](https://github.com/kubernetes/test-infra/tree/master/images/kubekins-e2e)
image.

The Prow job transfers control to a `.prow.sh` shell script which must
be present in a component that is configured to trigger the Prow job.

Each component has its own release configuration (what to build and
publish) and rules (scripts, makefile). The advantage is that those
can be branched and tagged together with the component. The version of
Go to use for building is also part of that configuration. The
requested version of Go will be installed from https://golang.org/dl/
if different from what is installed already.

To simplify maintenance and ensure consistency, the common parts can
be shared via
[csi-release-tools](https://github.com/kubernetes-csi/csi-release-tools/).

Unit testing is provided by `make test`. Images are pushed with `make
push`, which already determines image tags automatically. For Prow,
the image destination still needs to be determined
(https://github.com/kubernetes/k8s.io/issues/158).

For testing on Kubernetes, a real cluster can be brought up with
[kubetest](https://github.com/kubernetes/test-infra/tree/master/kubetest). This
might work with [kind](https://github.com/kubernetes-sigs/kind) and a
locally built image can be pushed directly into that cluster with
`docker save image | docker exec -it kind-control-plane docker load
-` (to be tested).

A shared E2E test could work like this:
- build one component from source
- deploy the hostpath example with that locally build component and
  everything else as defined in the current repository (i.e.
  each repository must vendor, copy or check out the example from
  `csi-drivers-host-path`)
- check out a certain revision of the kubernetes repo and
  run `go test ./test/e2e` with the parameter from https://github.com/kubernetes/kubernetes/pull/72836
  to test the deployed example; alternatively, the test suite
  can also be vendored, which would make the build self-contained
  and allow extending the test suite


### Risks and Mitigations

Pushing a new release image is triggered by setting a tag. Unless
there is a build failure, the result of the automatic build becomes
the latest release, without any further manual checking. There are
multiple risks:
- automatic testing of a single component cannot catch all bugs, so
  some bugs might make it into a tagged release
- the wrong revision gets tagged by the maintainer
- the build process itself is buggy and pushes a corrupt image

To mitigate this, maintainers can do a trial release first by tagging
a `-rc` version and doing additional manual tests with the result.

But ultimately the safeguard against such failures is that new CSI
sidecar containers only get used in a production cluster after
packagers of a CSI driver update the deployment files for their
driver.

## Graduation Criteria

- Test results are visible in GitHub PRs and test failures block merging.
- Test results are visible in the [SIG-Storage
  testgrid](https://k8s-testgrid.appspot.com/sig-storage-kubernetes) or
  a sub-dashboard.
- The Prow test output and/or metadata clearly shows what revisions of the
  different components were tested.
- All components have been converted to publishing images on `gcr.io` in addition
  to the traditional `quay.io`.

## Implementation History

- 2019-02-04: initial draft

## Drawbacks

Allowing individual maintainers to create releases without going
through a centralized release process implies that maintainers must be
more careful.

Because updated components get tested against the current set of other
components, breaking changes also break testing. If that ever becomes
necessary, manual intervention will be needed to release multiple
different component updates in sync.

## Alternatives

Automatically updating deployments by setting additional image tags
was originally suggested in the ["tag images using semantic
versioning" GitHub
issue](https://github.com/kubernetes-csi/driver-registrar/issues/77),
but further discussion in ["tag release images also with base
versions"](https://github.com/kubernetes-csi/csi-release-tools/issues/6)
rejected that idea because of the risks associated with automatically
changing versions in a production cluster.

A new hostpath driver binary gets created also when only the
deployment changes, for example because a sidecar gets updated. This
is a result of treating the driver binary and its deployment as a unit
with a single version number. The alternative would have been to
maintain the deployment in a separate repository with its own
independent versioning, but that has been rejected because it is more
work and because driver and deployment always need to be updated
together.

## Infrastructure Needed

* a Prow job that can start up test cluster(s), deploy images created as part of that job, and publish images
