<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-5040: Remove gitRepo volume driver.

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>

  <!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Validating Admission Policy](#validating-admission-policy)
- [Timeline](#timeline)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Disabled](#disabled)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Alternative 1: Add admission plugin that blocks gitRepo volumes](#alternative-1-add-admission-plugin-that-blocks-gitrepo-volumes)
  - [Alternative 2: Use ValidatingAdmissionPolicy](#alternative-2-use-validatingadmissionpolicy)
  - [Alternative 3: Make Git hooks disableable](#alternative-3-make-git-hooks-disableable)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

We propose removing support for the in-tree gitRepo volume driver. We aren't
proposing removing gitVOlumes from the Kubernetes API, meaning pods with gitRepo 
volumes will be admitted by kube-apiserver but kubelets with the feature-gate 
GitRepoVolumeDriver set to false will not run them and return an appropriate 
error to the user.

We acknowledge that this is highly unusual, but there are mitigating circumstances:
* gitRepo volume types can be exploited to gain remote code execution as root on the nodes as shown in [CVE-2024-10220](https://nvd.nist.gov/vuln/detail/cve-2024-10220)
* gitRepo has perfectly workable alternatives:
  * Using [git-sync](https://github.com/kubernetes/git-sync)
  * Using an initContainer. See https://kubernetes.io/docs/concepts/storage/volumes/#gitrepo for more details
* gitRepo has been deprecated for a long time and is unmaintained
* gitRepo has low usage (based on limited data)
* there exists precedent for removing volume drivers which were unmaintained like [glusterfs](https://kubernetes.io/docs/concepts/storage/volumes/#glusterfs)

Given this, we think kubernetes should EOL this feature with urgency.

## Motivation

Remove a vector for remote code execution through an ancient and unmaintained
volume plugin.

SIG Storage was actually very close to doing this and had even added a notice of
removal in the release notes for v1.32. But due to concerns raised by SIG Architecture
they rolled back the notice. SIG Storage provided [this rationale](https://github.com/kubernetes/kubernetes/issues/125983#issuecomment-2522201283) for considering removal.

### Goals

- Remove in-tree gitRepo volume support.

### Non-Goals

- Remove gitRepo volumes from the Kubernetes API.

## Proposal

Kubernetes drops support for gitRepo volume types by removing the in-tree driver
that actually mounts the volume in the pod. In this case we will still keep the
gitRepo volume in the API. Pods with gitRepo volumes will be admitted by 
kube-apiserver however, kubelet will fail to run them with an appropriate 
error message.


This removal is similar to removal of support for
[glusterfs](https://kubernetes.io/docs/concepts/storage/volumes/#glusterfs)
which was deprecated in v1.25 and removed in v1.26.

### Risks and Mitigations

The biggest risk is in breaking users. Despite being deprecated for years, there
may be some users who are still using it and their workloads would be broken 
when their clusters are upgraded. 

We plan to mitigate the risk by:
1. Allowing users to opt-in to re-enabling the driver for 3 versions to give
  them enough time to fix  workloads
2. Announcing the change via release notes, updates to the gitRepo documentation
  and publishing kubernetes release blog for this feature so that customers are
  aware and can take action before they upgrade
3. We will update the gitRepo volume documentation to say it say that the driver
  has been disabled by default and add information about alternative approaches.
  We will also document how users can turn on the gitRepo volume driver however,
  we will add a warning educating users on why they shouldn't do that.

## Design Details

We will add a new feature-gate to kubelet called `GitRepoVolumeDriver`. This
feature-gate will be defaulted to `false`, which will disable the 
[gitRepo driver](https://github.com/kubernetes/kubernetes/blob/master/pkg/volume/git_repo/git_repo.go).
If a workload with gitRepo volume is encountered, kubelet will fail to run such a
pod with an appropriate error message guiding users to either use an alternative
or setting the feature-gate `GitRepoVolumeDriver` to `true`.

### Validating Admission Policy

Users can prevent workloads with gitRepo volumes from being admitted 
using the following Validating Admission Policy (VAP). We plan to publish these
policies in a public repository.

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: "no-gitrepo-volumes"
spec:
  failurePolicy: Fail
  matchConstraints:
    resourceRules:
    - apiGroups:   [""]
      apiVersions: ["v1"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["pods", "podtemplates", "replicationcontrollers"]
    - apiGroups:   ["apps"]
      apiVersions: ["v1"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["deployments", "replicasets", "statefulsets", "daemonsets"]
    - apiGroups:   ["batch"]
      apiVersions: ["v1"]
      operations:  ["CREATE", "UPDATE"]
      resources:   ["jobs", "cronjobs"]
  validations:
    - expression: |-
        object.kind != 'Pod' ||
        !object.spec.?volumes.orValue([]).exists(v, has(v.gitRepo))
      message: "gitRepo volumes are not allowed."
    - expression: |-
        object.kind != 'PodTemplate' ||
        !object.template.spec.?volumes.orValue([]).exists(v, has(v.gitRepo))
      message: "gitRepo volumes are not allowed."
    - expression: |-
        !['Deployment','ReplicaSet','DaemonSet','StatefulSet','Job', 'ReplicationController'].exists(kind, object.kind == kind) ||
        !object.spec.template.spec.?volumes.orValue([]).exists(v, has(v.gitRepo))
      message: "gitRepo volumes are not allowed."
    - expression: |-
        object.kind != 'CronJob' ||
        !object.spec.jobTemplate.spec.template.spec.?volumes.orValue([]).exists(v, has(v.gitRepo))
      message: "gitRepo volumes are not allowed."
```

Users can using the following ValidatingAdmissionPolicyBinding to apply the 
ValidatingAdmissionPolicy above to all namespaces:

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicyBinding
metadata:
  name: "no-gitrepo-volumes"
spec:
  policyName: "no-gitrepo-volumes"
  validationActions:
  - Deny
```

## Timeline
v1.33 (04/2025): 
- We announce the removal in kubernetes release notes, documentation and via
kubernetes release blog post.
- We add the `GitRepoVolumeDriver` feature-gate to kubelet and
default it to `false`.

v1.34 (08/2025):
- No change.
- We will announce removal in kubernetes release notes, documentation and via
kubernetes release blog post.

v1.35 (11/2025):
- No change.
- We will announce removal in kubernetes release notes, documentation and via
kubernetes release blog post.

v1.36 (04/2026):
- Lock the feature-gate, i.e. users cannot change its value.
- We will announce removal in kubernetes release notes, documentation and via
kubernetes release blog post.

v1.37 (08/2026): 
- No change.
- We will announce removal in kubernetes release notes, documentation and via
kubernetes release blog post.

v1.38 (11/2026):
- No change.
- We will announce removal in kubernetes release notes, documentation and via
kubernetes release blog post.

v1.39 (04/2027):
- We will remove the feature-gate.
- We will remove the gitRepo driver code.
- We will announce removal in kubernetes release notes, documentation and via
kubernetes release blog post.


### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

We will add the following unit tests to the [git_repo](https://github.com/kubernetes/kubernetes/tree/master/pkg/volume/git_repo/git_repo_test.go) 
pacakage:

- When the feature-gate GitRepoVolumeDriver is false kubelet returns an error
  for pods that have gitRepo volumes in them, by marking them as failed and
  adding a condition
- When the feature-gate GitRepoVolumeDriver is false kubelet does not return an
  error for pods that have gitRepo volumes in them
- When the feature-gate GitRepoVolumeDriver is true kubelet does not returns an
  error for pods that have gitRepo volumes in them

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- <test>: <link to test coverage>

##### e2e tests

We will add the following e2e tests to the [e2e_node](https://github.com/kubernetes/kubernetes/tree/master/test/e2e_node) 
pacakage:

- When the feature-gate GitRepoVolumeDriver is false kubelet returns an error
  for pods that have gitRepo volumes in them
- When the feature-gate GitRepoVolumeDriver is false kubelet does not return an
  error for pods that have gitRepo volumes in them
- When the feature-gate GitRepoVolumeDriver is true kubelet does not return an
  error for pods that have gitRepo volumes in them

### Graduation Criteria

#### Disabled

Please see the proposed timelines for this removal in the 
[Design Details](#design-details) section. To move from the disabled to the 
removed stage the following criteria needs to be met:

- All code and tests to disable gitRepo volume driver by default is implemented
- Validating Admission Policy is published in a public repository
- Two versions have passed since introducing the default disablement of gitRepo volume driver (to address version skew)
- All feedback on usage/changed behavior, provided on GitHub issues has been addressed

### Upgrade / Downgrade Strategy

- If you are upgrading a cluster with no pods that have gitRepo volume then no
action is required
- If you are upgrading a cluster with pods that have gitRepo volume then there
are a few options:
  - Enable the feature-gate GitRepoVolumeDriver
  - Migrate the workloads to use [git-sync](https://github.com/kubernetes/git-sync)
  - Migrate the workloads to use emptyDir + initContainer to sync the git repo

Run the following command to check if there are pods with gitRepo volumes before
performing an upgrade.

```bash
kubectl get pods -A -o json \
| jq -r '.items[] | select(.spec.volumes[]? | has("gitRepo")) | .metadata.namespace + " " + .metadata.name'
```

### Version Skew Strategy

We aren't changing the Kubernetes API in regards to pods with gitRepo volumes
being admitted by kube-apiserver. These pods will still be admitted but kubelets
with GitRepoVolumeDriver=false will not run them and return an appropriate error
to the user.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
    - GitRepoVolumeDriver
  - Components depending on the feature gate:
    - kubelet

###### Does enabling the feature change any default behavior?

Yes, gitRepo volume driver will be disabled. So pods with gitRepo volumes will
fail to run.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

###### What happens if we reenable the feature if it was previously rolled back?

gitRepo volume driver will not work anymore.

###### Are there any tests for feature enablement/disablement?

Yes, please see testing sections above.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

This change does not affect the API servers. This change may affect the ability
of a kubelet to run a pod with gitRepo volumes.

If there are no workloads using gitRepo volumes we don't envision any
disruptions.

If there are workloads using gitRepo volumes then kubelets that have been
upgraded to have GitRepoVolumeDriver set to false will fail to run pods for
these workloads.

###### What specific metrics should inform a rollback?

Pods failing to start.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

Yes, we are disabling the in-tree gitRepo volume driver. We aren't removing it
from the API.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Operators can check if there are pods with gitRepo volumes in their cluster by
running the following command:

```bash
kubectl get pods -A -o json \
| jq -r '.items[] | select(.spec.volumes[]? | has("gitRepo")) | .metadata.namespace + " " + .metadata.name'
```

Operators can check if the feature-gate `GitRepoVolumeDriver` is enabled on a
particular node by running the following command:

```bash
kubectl get --raw "/api/v1/nodes/<node-name>/proxy/metrics" \
| grep kubernetes_feature_enabled \
| grep GitRepoVolumeDriver
```

###### How can someone using this feature know that it is working for their instance?

- [X] Other (treat as last resort)
  - Details:
    Assuming all nodes have been upgraded, create a pod with gitRepo volumes.
    This pod should fail to start. 

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

No change to kubelet SLOs.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [x] Other (treat as last resort)
  - Details: Existing SLIs for kubelet apply.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

N/A. Feature is only dependent on kubelet.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

This feature is not dependent on API server or etcd availability.

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

### Alternative 1: Add admission plugin that blocks gitRepo volumes
We add an in-tree admission plugin that, when enabled, will prevent Pods with
gitRepo volumes from being admitted.

Pros:
- Backwards compatible.
- Allows cloud providers to make a call if they want to support gitRepo volumes or not.

Cons:
- Not secure by default, requires cluster-admin action.
- The only escape valve is cluster-scoped.
- We'll be adding an in-tree admission plugin for a volume type that is not
  maintained.

### Alternative 2: Use ValidatingAdmissionPolicy
We publish a VAP which prevents Pods and other workloads from using gitRepo, and
strongly recommend cluster-admins (or cluster-providers) install it.

Pros:
- Backwards compatible.
- Allows cluster-admins to make a call if they want to support gitRepo volumes or not.
- Can be scoped to namespaces rather than whole clusters.
- Works on older clusters.

Cons:
- Not secure by default
- Requires cluster-admin action.

### Alternative 3: Make Git hooks disableable
We add a new field to gitRepo volume called `enableHooks` to
configure if Git hooks should be run as part of gitRepo volume mounting.
When this value is set to true kubelet will execute Git hooks when performing 
`git clone` and `git checkout`. If enableHooks is unset or set to false, kubelet
will not run any Git hooks when performing `git clone` and `git checkout`.

Pros:
- gitRepo volume type will still be supported and we will eliminate the risks.

Cons:
- We are adding new functionality to a deprecated and unmaintained volume type
  that has better alternatives which are not affected by the vulnerability.
- git hooks may not be the only source of vulnerabilies in the future and 
  we will be playing whack-a-mole with executing git as kubelet (root on host)


## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
