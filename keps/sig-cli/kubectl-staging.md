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
status: implementable
see-also:
---


# Move Kubectl Code into Staging

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Adding the staging repository in kubernetes/kubernetes:](#adding-the-staging-repository-in-kuberneteskubernetes)
  - [Modify and Set-up the existing receiving kubernetes/kubectl repository](#modify-and-set-up-the-existing-receiving-kuberneteskubectl-repository)
  - [Move <code>pkg/kubectl</code> Code](#move--code)
  - [Timeframe](#timeframe)
- [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Testing](#testing)
- [Implementation History](#implementation-history)
<!-- /toc -->

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

The following steps to create the staging repository have been copied from
the base [staging directory README](https://github.com/kubernetes/kubernetes/tree/master/staging).

### Adding the staging repository in kubernetes/kubernetes:

1. Send an email to the SIG Architecture mailing list and the mailing list of
   the SIG CLI which would own the repo requesting approval for creating the
   staging repository.

2. Once approval has been granted, create the new staging repository.

3. Add a symlink to the staging repo in vendor/k8s.io.

4. Update import-restrictions.yaml to add the list of other staging repos that
   this new repo can import.

5. Add all mandatory template files to the staging repo as mentioned in
   https://github.com/kubernetes/kubernetes-template-project.

6. Make sure that the .github/PULL_REQUEST_TEMPLATE.md and CONTRIBUTING.md files
   mention that PRs are not directly accepted to the repo.

### Modify and Set-up the existing receiving kubernetes/kubectl repository

Currently, there are three types of content in the current kubernetes/kubectl
repository that need to be dealt with. These items are 1) the [Kubectl
Book](https://kubectl.docs.k8s.io/), 2) an integration test framework, and 3)
some kubectl openapi code. Since we intend to copy the staging code into this
kubernetes/kubectl repository, and since this repository must be empty, we need
to dispose of these items. A copy of the kubectl openapi code already exists
under `pkg/kubectl` in the kubernetes/kubernetes repository, so it can be
deleted. The following steps describe how we intend to modify the existing
kubernetes/kubectl repository:

1. We will create a backup of the entire [current
   kubectl](https://github.com/kubernetes/kubectl) repository in the [Google
   Cloud Source Repository](https://cloud.google.com/source-repositories/).

2. The [Kubectl Book](https://kubectl.docs.k8s.io/) should be in its own
   repository. So we will create a new repository, and copy this content into
   the new repository.

3. We will then clear the [current
   kubectl](https://github.com/kubernetes/kubectl) repository.

4. Setup branch protection and enable access to the stage-bots team by adding
   the repo in prow/config.yaml. See #kubernetes/test-infra#9292 for an
   example.

5. Once the repository has been cleared, update the publishing-bot to publish
   the staging repository by updating:

   rules.yaml: Make sure that the list of dependencies reflects the staging
   repos in the go modules.

   fetch-all-latest-and-push.sh: Add the staging repo in the list of repos to be
   published.

6. Add the staging and published repositories as a subproject for the SIG that
   owns the repos in sigs.yaml.

7. Add the repo to the list of staging repos in this README.md file.

8. We will re-introduce the integration test framework into the kubectl
   repository by submitting into the new staging directory.

### Move `pkg/kubectl` Code

Move `k8s.io/kubernetes/pkg/kubectl` to a location under the new
`k8s.io/kubernetes/staging/src/k8s.io/kubectl` directory, and update all
`k8s.io/kubernetes/pkg/kubectl` imports. This can be an incremental process.

### Timeframe

There are three remaining Kubernetes core dependencies that have to be resolved
before all of `pkg/kubectl` can be moved to staging. While we work on those
remaining dependencies, we can move some `pkg/kubectl` code to staging that is
currently being requested by other projects. Specifically, we will move
`pkg/kubectl/cmd/apply` into staging as soon as possible. The rest of the code
would be moved over the next two releases (1.16 and 1.17).

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
