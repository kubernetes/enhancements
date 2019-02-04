---
kep-number: 0
title: kubernetes-csi release process
authors:
  - "@pohly"
owning-sig: sig-storage
participating-sigs:
  - sig-testing
  - sig-release
reviewers:
  - @msau42
approvers:
  - @msau42
  - sig-testing: TBD
  - sig-release: TBD
editor: @pohly
creation-date: 2019-02-04
last-updated: 2019-02-04
status: provisional
see-also:
replaces:
superseded-by:
---

# kubernetes-csi-release-process

## Table of Contents

- [kubernetes-csi-release-process](#kubernetes-csi-release-process)
    - [Table of Contents](#table-of-contents)
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
automatic end-to-end testing.

With CSI support reaching GA in Kubernetes 1.13, it is time to define
and follow a less work-intensive and more predictable approach.

### Goals

- define what a "kubernetes-csi release" is
- unit testing as mandatory pre-submit check for all components 
- E2E testing as mandatory pre-submit check for components that get
  deployed in a cluster
- define a release process for each component and a "kubernetes-csi release"

### Non-Goals

- change responsibilities among the kubernetes-csi maintainers - as before, each
  component will have some main maintainer who is responsible for releasing updates
  of that component
- automatically generate release notes
- automatically generate "kubernetes-csi release" artifacts - this will need further thoughts


## Proposal

### Versionioning

Each of the components has its own documentation, versioning and
release notes. [Semantic versioning](https://semver.org/) is used for
components that provide a stable API, like for example:
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

A "kubernetes-csi release" is a specific set of component
releases. It's also called the "combined release". Documentation for
combined releases is found on [kubernetes-csi
docs](https://kubernetes-csi.github.io/docs/). The combined release is
typically going to be prepared and announced less frequently than the
individual component releases. It is therefore only a recommendation
that downstream users use this combination of components. In
particular, individual components might also get minor updates after a
kubernetes-csi release without updating that combined release.

The
[hostpath example deployment](https://github.com/kubernetes-csi/csi-driver-host-path/tree/master/deploy)
defines the components that are part of a combined release. This
implies that the hostpath driver repo must be updated and tagged to
create a new combined release. Therefore the hostpath driver's version
becomes the kubernetes-csi release version, which is increased
according to the same rules as the individual components (major
version bump when any of its components had a major change, etc.).

#### Release artifacts

Tagging a component with a semantic version number triggers a release
build for that component. The output is primarily the container image
for components that need to be deployed. Those images get published as
`gcr.io/kubernetes-csi/<image>:<tag>` pending [issue
158](https://github.com/kubernetes/k8s.io/issues/158).

Only binaries provided as part of such a release should be considered
production ready. Binaries are never going to be rebuilt, therefore an
image like `csi-node-driver-registrar:v1.0.2` will always pull the
same content and `imagePullPolicy: Always` can be omitted.

Eventually auxiliary files like the RBAC rules that are included in
each component might also get published in a single location, like
`https://dl.k8s.io/kubernetes-csi/`, but this currently outside of
this KEP.

#### Release process

* A change is submitted against master.

* A [Prow](https://github.com/kubernetes/test-infra/blob/master/prow/README.md)
  job checks out the source of a modified component, rebuilds it and then
  runs unit tests and E2E tests with it.

* Maintainers accept changes into the master branch.

* The same Prow job runs for master and repeats the check. If it succeeds,
  a new "canary" image is published for the component.

* In preparation for the release of a major new update, a feature freeze is
  declared for the "master" branch and only changes relevant for that next
  release are accepted.

* When all changes targeted for the release are in master, automatic
  test results are okay and and potentially some more manual tests,
  maintainers tag each component directly on the master branch.

* Maintenance releases are prepared by branching a "release-X.Y" branch from
  release "vX.Y" and backporting relevant fixes from master. The same
  prow job as for master also handles the maintenance branches, but potentially
  with a different configuration.


### Implementation Details

Each component has its own release configuration (what to build and
publish) and rules (scripts, makefile). The advantage is that those
can be branched and tagged together with the component.

To simplify maintenance and ensure consistency, the common parts can
be shared via
[csi-release-tools](https://github.com/kubernetes-csi/csi-release-tools/).

The prow job then just provides a common execution environment, with
the ability to bring up test cluster(s), publish container images on
and other files.

### Risks and Mitigations

Pushing a new release image is triggered by setting a tag. Unless
there is a build failure, the result of the automatic build becomes
the latest release, without any further manual checking. There are
multiple risks:
- automatic testing of a single component cannot catch all bugs, so
  some bugs might make it into a tagged release
- the wrong revision gets tagged by the maintainer
- the build process itself is buggy and pushes a corrupt image

The safeguard against such failures is that new CSI sidecar containers
only get used in a production cluster after packagers of a CSI driver
update the deployment files for their driver.

## Graduation Criteria

- E2E test results are visible in GitHub PRs and test failures block merging.
- All components have been converted to publishing images on `gcr.io` in addition
  to the traditional `quay.io`.
- The example hostpath deployment works with those images.

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


## Infrastructure Needed

* a Prow job that can start up test cluster(s), deploy images created as part of that job, and publish images
