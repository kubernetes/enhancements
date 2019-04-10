---
title: Move Kubectl Code into Staging
authors:
  - "@seans3"
owning-sig: sig-cli
participating-sigs:
reviewers:
  - "@pwittrock"
  - "@liggitt"
approvers:
  - "@soltysh"
editor: "@seans3"
creation-date: 2019-04-09
last-updated: 2019-04-09
see-also:
status: implementable
---


# Move Kubectl Code into Staging

## Table of Contents
* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Proposal](#proposal)
* [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Testing](#testing)
* [Implementation History](#implementation-history)

## Summary

We propose to move the `pkg/kubectl` code base into staging. This effort would
entail moving `k8s.io/kubernetes/pkg/kubectl` to an appropriate location under
`k8s.io/kubernetes/staging/src/k8s.io/kubectl`. The `pkg/kubectl`
code would not have to move all at once; it could be an incremental process. When
the code is moved, imports of the `pkg/kubectl` code would be updated to the
analogous `k8s.io/kubectl` imports.

## Motivation

Moving `pkg/kubectl` to staging would provide at least three benefits:

1. SIG CLI is in the process of moving the kubectl binary into its own repository
under https://github.com/kubernetes/kubectl. Moving `pkg/kubectl` into staging
is a necessary first step in this kubectl independence effort.

2. Over time, kubectl has grown to inappropriately depend on various internal
parts of the Kubernetes code base, creating tightly coupled code which is
difficult to modify and maintain. For example, internal resource types (those
that are used as the hub in type conversion) have escaped the API Server and
have been incorporated into kubectl. Code in the staging directories can not
have dependencies on internal Kubernetes code. Moving `pkg/kubectl` to staging would
prove that `pkg/kubectl` does not contain internal Kubernetes dependencies, and it
would permanently decouple kubectl from these dependencies.

3. Currently, it is difficult for external or out-of-tree projects to reuse code
from kubectl packages. The ability to reuse kubectl code outside of
kubernetes/kubernetes is a long standing request. Moving code to staging
incrementally unblocks this use case. We will be publish the code from staging
under https://github.com/kubernetes/kubectl/ .

### Goals

- Structure kubectl code to eventually move the entire kubectl code base into
  its own repository.
- Decouple kubectl permanently from inappropriate internal Kubernetes dependencies.
- Allow kubectl code to easily be imported into other projects.

### Non-Goals

- Explaining the entire process for moving kubectl into its own repository will
not be addressed in this KEP; it will be the subject of its own KEP.

## Proposal

Move `k8s.io/kubernetes/pkg/kubectl` to a location under
`k8s.io/kubernetes/staging/src/k8s.io/kubectl`, and update all
`k8s.io/kubernetes/pkg/kubectl` imports.

## Risks and Mitigations

If a project vendors Kubernetes to import kubectl code, this will break them.
On the bright side, afterwards, these importers will have a much cleaner path to 
include kubectl code. Before moving forward with this plan, we will identify and
communicate to these projects.

## Graduation Criteria

Since this "enhancement" is not a traditional feature, and it provides no new
functionality, graduation criteria does not apply to this KEP.

## Testing

Except for kubectl developers, this change will be mostly transparent. There is
no new functionality to test; testing will be accomplished through the current
unit tests, integration tests, and end-to-end tests.

## Implementation History

See [kubernetes/kubectl#80](https://github.com/kubernetes/kubectl/issues/80) as
the umbrella issue to see the details of the kubectl decoupling effort. This
issue has links to numerous pull requests implementing the decoupling.
