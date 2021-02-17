# package-generation

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Examples](#examples)
      - [Removing a deprecated flag](#removing-a-deprecated-flag)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
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


[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

Implementation of a standard, community supported and maintained process of generating and building rpm and deb packages.

## Motivation

Today, the tooling used to build kubernetes packages is separate from kubernetes itself. This can cause downstream divergence, and cannot be easily maintained by the kubernetes community as a whole.

The primary goal of this proposal is to simplify the lifecycle management of packaging by co-locating the sources and tooling for generating packages into the kubernetes/kubernetes repository.

### Goals

To create a maintainable and configurable method of building kubernetes artifacts supported and maintained by the community.

- Move packaging tools out of kubernetes/release and into kubernetes/kubernetes repository
- Enable customization for downstream packaging
    - Including the ability to bring their own source code/binaries and changelogs
    - Stop hard-coding the binary download URL to dl.k8s.io
- Enable community governance over packaging of the project.
    - Will be governed by sig-release
- Tightly couple the versioning of packing and the kubernetes/kubernetes release.
- Ensure that the same packages that are run in CI are the ones that are released.
- Dedupe the generation of packages (THERE CAN BE ONLY ONE!)
- Have one config file that is the source of truth for packaging details for both .rpm and .deb packages.
- Artifact generation is independent of source code compilation.

### Non-Goals

- A change in the release process. See this [KEP](https://github.com/kubernetes/enhancements/pull/843).
    - For example, package signing and distribution are not covered in this document and are covered in the above KEP.

- To fundamentally change the packages and their dependencies. All packages should remain consistent with what we build today.

## Proposal

As outlined in the following issue: https://github.com/kubernetes/kubernetes/issues/71677
Move all packaging into the kubernetes/kubernetes repo and create the following 3 building blocks to support it.
- Create a configuration yaml file to enable the configuring of the built packages.
- Create a Go lang app to parse the yaml config and submit simple commands to the container.
- Build inside a docker container based off [`fpm`](https://github.com/jordansissel/fpm).

### User Stories

#### Story 1

As a developer I would like to be able to install the entire set of alpha or beta (.deb/.rpm) packages to stand up a development cluster.  These packages should be equivalent to the future release artifacts.

#### Story 2

As a third party vendor, I would like to be able to package kubernetes deliverable artifacts using the same tooling that kubernetes’ uses.

#### Story 3

As a release manager I would like to enable simple and easy package generation that the community can manage.

### Implementation Details/Notes/Constraints [optional]

- Use an [`fpm`](https://github.com/jordansissel/fpm) based container to build packages
    - [`nfpm`](https://github.com/goreleaser/nfpm) was originally investigated but lacked features
- Use yaml to configure
- Create a go binary whose only job is to consume the yaml, build the right file structure and issue commands to the image to build the artifacts. Effectively the glue.

### Risks and Mitigations

Risk: Confusion as to what to use when. Mitigation: Coordination across groups to ensure that the changes made in k/k are consumed by the release team.

## Design Details

### Test Plan

Ideally these new packages will get tested inside the test pipeline. When [`KIND`](https://sigs.k8s.io/kind) goes to run kubeadm/kubelet/kubectl/etc tests, it should install the packages generated by these tools as they will be contained within the kubernetes/kubernetes repository.

This will give us a clear path to the package consumed when testing, enabling a more confident release.

### Graduation Criteria

N/A

#### Examples

1:
An independent developer wanting to generate packages in their own fork would be able to edit the config yaml in their fork, override a dependency name and then issue a command to the go packaging binary to generate the packages.
`[go packing binary] --config=myconfig.yaml`

2:
Bazel will be able to update the configuration yaml to dynamically update the configuration before simply executing commands directly against the go packing binary.

##### Removing a deprecated flag

N/A

[conformance tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

N/A

## Implementation History

N/A

## Drawbacks

N/A

## Alternatives

N/A

## Infrastructure Needed

N/A


