# KEP-1020: Move Kubectl Code into Staging

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Adding the staging repository in kubernetes/kubernetes:](#adding-the-staging-repository-in-kuberneteskubernetes)
  - [Modify and Set-up the existing receiving kubernetes/kubectl repository](#modify-and-set-up-the-existing-receiving-kuberneteskubectl-repository)
  - [Move <code>pkg/kubectl</code> Code](#move-pkgkubectl-code)
  - [Time frame](#time-frame)
- [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Testing](#testing)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [X] Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

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

### Time frame

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

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback


* **How can this feature be enabled / disabled in a live cluster?**
  Not aplicable.

* **Does enabling the feature change any default behavior?**
  Not applicable.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Not applicable.

* **What happens if we reenable the feature if it was previously rolled back?**
  Not applicable.

* **Are there any tests for feature enablement/disablement?**
  Not applicable.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**
  Not applicable.

* **What specific metrics should inform a rollback?**
  Not applicable.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Not applicable.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**
  Not applicable.

### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?**
  Not applicable.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
  Not applicable.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  Not applicable.

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**
  Not applicable.

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
  Not applicable.

### Scalability

* **Will enabling / using this feature result in any new API calls?**
  Not applicable.

* **Will enabling / using this feature result in introducing new API types?**
  Not applicable.

* **Will enabling / using this feature result in any new calls to the cloud
provider?**
  Not applicable.

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**
  Not applicable.

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**
  Not applicable.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Not applicable.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

* **How does this feature react if the API server and/or etcd is unavailable?**
  Not applicable.

* **What are other known failure modes?**
  Not applicable.

* **What steps should be taken if SLOs are not being met to determine the problem?**
  Not applicable.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

See [kubernetes/kubectl#80](https://github.com/kubernetes/kubectl/issues/80) as
the umbrella issue to see the details of the kubectl decoupling effort. This
issue has links to numerous pull requests implementing the decoupling.
