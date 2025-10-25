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
# KEP-4958: CSI Sidecars All In One

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Increased maintenance tasks on components](#increased-maintenance-tasks-on-components)
    - [CSI Sidecars releases](#csi-sidecars-releases)
  - [Maintenance tasks by CSI Driver authors and cluster administrators](#maintenance-tasks-by-csi-driver-authors-and-cluster-administrators)
  - [Resource utilization by the CSI Sidecar components](#resource-utilization-by-the-csi-sidecar-components)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Overview](#overview)
  - [User Stories (Optional)](#user-stories-optional)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Design Details](#design-details)
    - [Glossary](#glossary)
    - [AIO Monorepo](#aio-monorepo)
      - [Release Management](#release-management)
      - [RBAC policy](#rbac-policy)
      - [Command Line](#command-line)
      - [Code synchronization](#code-synchronization)
      - [Individual repo history](#individual-repo-history)
      - [Reproducible builds &amp; Dependencies Management](#reproducible-builds--dependencies-management)
    - [Risks And Mitigations](#risks-and-mitigations)
    - [Development workflow](#development-workflow)
  - [MileStone](#milestone)
    - [Milestone-modify-entrypoints-of-existing-sidecars-to-integrate-it-seamlessly-with-the-AIO-sidecar](#milestone-modify-entrypoints-of-existing-sidecars-to-integrate-it-seamlessly-with-the-aio-sidecar)
    - [Milestone-setting-up-a-Kubernetes-CSI-Storage-Repository-with-nested-directory-synchronization](#milestone-setting-up-a-kubernetes-csi-storage-repository-with-nested-directory-synchronization)
    - [Milestone-Build-the-project-using-a-modified-copy-of-release-tools](#milestone-build-the-project-using-a-modified-copy-of-release-tools)
    - [Milestone-set-up-new-test-infra-jobs-to-test-the-project-through-the-hostpath-CSI-Driver](#milestone-set-up-new-test-infra-jobs-to-test-the-project-through-the-hostpath-csi-driver)
    - [Milestone-ready-to-accept-PR-from-community](#milestone-ready-to-accept-pr-from-community)
    - [Milestone-define-the-path-for-2-CSI-Drivers-to-be-migrated.](#milestone-define-the-path-for-2-csi-drivers-to-be-migrated)
    - [Milestone: Have instructions for CSI Driver authors](#milestone-have-instructions-for-csi-driver-authors)
    - [Milestone-three-cloud-vendors-start-using-the-monorepo-component-for-multi-k8s-minor-releases](#milestone-three-cloud-vendors-start-using-the-monorepo-component-for-multi-k8s-minor-releases)
    - [Milestone-accept-PR-from-community](#milestone-accept-pr-from-community)
    - [milestone-all-individual-repo-has-been-into-featurefreeze-state](#milestone-all-individual-repo-has-been-into-featurefreeze-state)
    - [Milestone-all-individual-repo-has-been-into-deprecated-state](#milestone-all-individual-repo-has-been-into-deprecated-state)
    - [Milestone-merge-sidecar-informer-caches](#milestone-merge-sidecar-informer-caches)
  - [Questions and Answers](#questions-and-answers)
    - [Project Goal and Motivation](#project-goal-and-motivation)
    - [Development and Release Process](#development-and-release-process)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [AIO MonoRepo state definition](#aio-monorepo-state-definition)
    - [Individual repository state definition](#individual-repository-state-definition)
    - [Migration Process](#migration-process)
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
    - [ReleaseManagement](#releasemanagement)
    - [ReproducibleBuilds](#reproduciblebuilds)
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

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [X] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

We propose to combine the source code of the CSI Sidecars in a monorepo, Instead of just putting the code repositories together, 
it is expected that the program entries for all sidecars will be consolidated.
therefore we can:
- Improve the CSI Sidecar release process by reducing the number of components released
- Decrease the maintenance tasks the SIG Storage community maintainers do to maintain the Sidecars
- Propagate changes in common libraries used by CSI Sidecars immediately instead of through additional PRs
- Reduce the number of components CSI Driver authors and cluster administrators need to keep up to date in k8s clusters

As a side effects of combining the CSI Sidecars into a single component we also
- Reduce the memory usage/API Server calls done by the CSI Sidecars through the usage of a shared informer.
- Reduce the cluster resource requirements need to run the CSI Sidecars

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementers to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

## Motivation

### Increased maintenance tasks on components
[The SIG Storage community](https://github.com/kubernetes/community/tree/master/sig-storage) maintains CSI sidecars in separate repositories: external-attacher, external-provisioner, external-resizer and etc. The community must periodically **update** with go version bumps, library updates, CVE fixes, CHANGELOG updates and separate releases, which **takes** a lot of maintenance effort.


#### CSI Sidecars releases

The CSI Drivers/CSI Sidecars have an indirect dependency on the k8s version. This could happen because of:
- A new CSI feature that touches CSI Sidecars and k8s component - For example the [ReadWriteOncePod](https://kubernetes.io/blog/2021/09/13/read-write-once-pod-access-mode-alpha/) feature needs changes in k8s components (kube-apiserver, the kube-scheduler, the kubelet), CSI Sidecars

Because of this indirect dependency the SIG Storage community creates a minor release of each CSI Sidecar for every k8s minor release. We use csi-hospath (a CSI Driver used for testing purposes) to test the compatibility of the new releases with the latest k8s version.

We follow the instructions on [SIDECAR_RELEASE_PROCESS.md](https://github.com/kubernetes-csi/csi-release-tools/blob/master/SIDECAR_RELEASE_PROCESS.md) on every CSI Sidecar to create a minor release.


### Maintenance tasks by CSI Driver authors and cluster administrators

CSI driver authors face a continuous maintenance burden. They must constantly track and update their drivers to align with the ever-evolving CSI sidecars, ensuring compatibility, security, and access to new features.

![csi driver basic structure](./aio2.png "container components of csi driver")


### Resource utilization by the CSI Sidecar components

In Some CSI Driver control plane deployment setups each sidecar is configured with a minimum memory request, some examples of OSS CSI Driver deployments resource allocations:
- Memory request
  - EBS CSI Driver
    - In a CP node, sets a [40Mi memory request](https://github.com/kubernetes-sigs/aws-ebs-csi-driver/blob/a85fb6358eae7b83a083eb8003cf929b3f31d413/charts/aws-ebs-csi-driver/values.yaml#L234) for each CSI Sidecars(5 sidecars), a total of 200Mi per node.
    - In a worker node, sets a [40Mi memory request](https://github.com/kubernetes-sigs/aws-ebs-csi-driver/blob/a85fb6358eae7b83a083eb8003cf929b3f31d413/charts/aws-ebs-csi-driver/values.yaml#L323) for each CSI Sidecar(2 sidecars), a total of 80Mi per node
  - Azuredisk
    - In a CP node, sets a [20Mi memory request](https://github.com/kubernetes-sigs/azuredisk-csi-driver/blob/30d9bbfde6612c43aa5103bcf4fe4e1e70815892/charts/latest/azuredisk-csi-driver/values.yaml#L78) for each CSI Sidecars(5 sidecars), a total of 100Mi per node
    - In a worker node, sets a [20Mi memory request](https://github.com/kubernetes-sigs/azuredisk-csi-driver/blob/30d9bbfde6612c43aa5103bcf4fe4e1e70815892/charts/latest/azuredisk-csi-driver/values.yaml#L78) for each CSI Sidecars(2 sidecars), a total of 40Mi per node
  - AlibabaCloud Disk
    - In a CP node, sets a [16Mi memory request](https://github.com/kubernetes-sigs/alibaba-cloud-csi-driver/blob/9819c8b575acb5eadfb6fada4e42a4add2453c18/deploy/chart/templates/controller.yaml#L106) for each CSI Sidecars(average 4 sidecars) a total of 64Mi per node
    - In a worker node, sets a [16Mi memory request](https://github.com/kubernetes-sigs/alibaba-cloud-csi-driver/blob/9819c8b575acb5eadfb6fada4e42a4add2453c18/deploy/chart/templates/plugin.yaml#L82) for each CSI Sidecars(1 sidecars), a total of 40Mi per node
The 5x memory request is additional overhead in the control plane nodes, 2x in the worker nodes

### Goals

- To combine the source code of common Container Storage Interface (CSI) sidecars, controllers, and webhooks into a single monorepo.
- From the single repository, produce a single artifact(binary and container image) similar to how kube-controller-manager operates.
  - If we just merge the source code, we won't be able to reuse resources and realize the above advantages
  - To minimize the impact on CSI driver authors and cluster administrators, the migration process is unified into a single step, avoiding separate code and binary migrations. CSI driver authors and cluster administrators can keep building binaries and images with individual sidecars from the single repository. Each CSI driver author can start using the merged sidecar binary 

- The sidecars includes the following:
  - [external-attacher](https://github.com/kubernetes-csi/external-attacher)
  - [external-provisioner](https://github.com/kubernetes-csi/external-provisioner)
  - [external-resizer](https://github.com/kubernetes-csi/external-resizer)
  - [external-snapshotter](https://github.com/kubernetes-csi/external-snapshotter)
  - [livenessprobe](https://github.com/kubernetes-csi/livenessprobe)
  - [node-driver-registrar](https://github.com/kubernetes-csi/node-driver-registrar)
  - [volume-health-monitor](https://github.com/kubernetes-csi/external-health-monitor)
  - [volume-data-source-validator](https://github.com/kubernetes-csi/volume-data-source-validator)
- Retain git history logs of sidecars in new monorepo.

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

### Non-Goals

- Include [sig-storage-lib-external-provisioner](https://github.com/kubernetes-sigs/sig-storage-lib-external-provisioner) in the new repository. 
  - Because it doesn't depend on release-tools or csi-lib-utils. 

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

## Proposal

### Overview

The proposal consists of creating a monorepo which creates a single artifact with common sidecars combined in one binary:
- Combine the source code of all common CSI sidecars (external-attacher, external-provisioner, external-resizer, external-snapshotter, livenessprobe, node-driver-registrar), Controllers(snapshot controller, volume-health-monitor controller), Webhooks(csi-snapshot-validation-webhook) in a single repository. ***A total of 7 repositories including 6 sidecars, 2 controllers and 1 webhook.***
- Include the source code of helper utilities in the same repository([csi-release-tools](https://github.com/kubernetes-csi/csi-release-tools), [csi-lib-utils](https://github.com/kubernetes-csi/csi-lib-utils)), sidecars/apps use the local modules through go workspaces. A total of 1 release helper and 1 go module.
- Create a new cmd/ entrypoint that enables sidecars selectively, similar to kube-controller-manager and the --controllers flag.

![csi aio structure state](./aio4.png)

CSI Driver authors would include a single sidecar in their deployments(in both the control plane and node pools). while the artifact version is the same, the command/arguments will be different.

pictures:
![desired aio component structure](./aio5.png)

The CSI Driver deployment manifest would look like this in the control plane:

```yaml
kind: Deployment
apiVersion: app/v1
metadata:
  name: csi-driver-deployment
spec:
  replicas: 1
  templates:
    spec:
      containers:
        - name: csi-driver
          args:
            - "--v=5"
            - "--endpoint=unix:/csi/csi.sock"
        - name: csi-sidecars
          command:
            - csi-sidecars
            - "--csi-address=unix:/csi/csi.sock"
            # similar style as kube-controller-manager
            - "--controllers=attacher,provisioner,resizer,snapshotter"
            - "--feature-gates=Topology=true"
            # leader election flags for all the components as one
            - "--leader-election"
            - "--leader-election-namespace=kube-system"
            # global timeouts
            - "--timeout=30s"
            # per controller specific flags are prefixed with the component name
            - "--attacher-timeout=30s"
            - "--attacher-worker-thread=100"
            - "--provisioner-timeout=30s"
          volumeMounts:
            - mountPath: /csi
              name: socket-dir
```

The CSI Driver deployment manifest would look like this in the worker node

```yaml
kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: csi-driver-deployment
spec:
  template:
    spec:
      containers:
        - name: csi-driver
          args:
            - "--v=5"
            - "--endpoint=unix:/csi/csi.sock"
        - name: csi-sidecars
          command:
            - csi-sidecars
            - "--csi-address=unix:/csi/csi.sock"
            # similar style as kube-controller-manager
            - "--controllers=node-driver-registrar"
            - "--kubelet-registration-path=/var/lib/kubelet/plugins/<csi-driver>/csi.sock"
          volumeMounts:
            - name: registration-dir
              mountPath: /registration
            - name: plugin-dir
              mountPath: /csi
      volumes:
        - name: registration-dir
          hostPath:
            path: /var/lib/kubelet/plugins_registry/
            type: Directory
        - name: plugin-dir
          hostPath:
            path: /var/lib/kubelet/plugins/<csi-driver>/
            type: DirectoryOrCreate
```

Quantifiable characteristics of the current state and of the proposed state


| Characteristics/State | Current state of CSI Sidecars(let nr. of csi-sidecars=6)|  CSI Sidecars in signal component    | 
|-----------------|----------------------|-------------------|
|  Human effort of propagating csi-release-tools        |  (nr. of csi-release-tools changes * nr. of csi-sidecars)          |    0, (because csi-release-tools is part of the repo)     | 
|  Human effort of propagating csi-lib-utils        |  (nr. of csi-lib-utils changes * nr. of csi-sidecars)          |    0, (because csi-lib-utils is part of the repo)     | 
|  go mod dependency bumps        |  (nr. of dependency changes * nr. of csi-sidecars) * CSI releases supported(unknown)          |   nr. of dependency changes * releases supported(follow k8s release)    | 
|  runtime update  |  (nr. of csi-release-tools changes related with go runtime updates * nr. of csi-sidecars)        |   nr. of go runtime updates | 
|  members of CSI releases per k8s minor release  |   nr. of csi-sidecars     |   1 | 


Additional properties of a single CSI Sidecar component without a quantifiable benefit:


| Dimension | Pros |   Cons  | 
|-----------------|----------------------|-------------------|
|  Releases        |  <li> Easier releases <li> Better definition of which sidecar releases are supported for CVE fixs i.e. if our model of support is similar to k8s (last 3 releases) then the same applies to the CSI sidecar releases <li> Release nodes in csi-release-tools are part of the release. Currently, [commits in csi-release-tools with release notes get lost](https://github.com/kubernetes-csi/node-driver-registrar/pull/235) because the git subtree commands replays commits but loses the PR release note if csi-release-tools is part of the repo        |     <li> No longer able to do single releases per component.<li> More frequent major version bumps, Currently, we increase the major version of a sidecar when we remove a command line parameter or require new RBAC rules, We ended up with provisioner v5, attacher v4, and snapshotter v8. With a common repo, we would end up with 5+4+8=v17 in the worst case.  | 
|  Testability        |   <li> [Easier testing](https://slideshare.net/sttts/cutting-the-kubernetes-monorepo-in-pieces-never-learnt-more-about-git) <li> Test features that spawn multiple components e.g. the RWOP feature can be tested as a whole. @pohly  |       | 
|  Performance & Reliability       |  <li> Can use a shared informer decreasing the load on the API server. @msau42  |  <li> Container getting OOMKilled kills the entire CSI machinery, not just a single component.<ul><li>In HA, another replica would take over a few seconds.   | 
|  Simplicity    |  <li> Consolidation of common parameters like leader election, structured logging<ul><li>Avoids duplication of some features e.g. [structured logging](https://github.com/kubernetes-csi/livenessprobe/pull/202) would be implemented only once [instead of #csi-sidecar times](https://github.com/kubernetes-csi/livenessprobe/pull/202#issuecomment-1682406525)) @msau42</ul> <li> Combination of metrics/health ports @msau42 <li> Enables using additional sidecars that aren't used because of additional build pipelines that might be needed to support that additional component.  |  <li> Logs would be interleaved making it harder to trace what happened for a request <li> CSI utility liraries that are not only used by CSI Sidecars but by other project. <ul><li> make an external repo which is automatically synchronized from the internal csi-release-tools e.g. a similar analogy to k/k/staging/lib -> k/lib  | 
|  Integration with CSI Drivers    |   <li> Less config in the controller/node yaml manifest <li>Less confusion for CSI Driver authors on which CSI Sidecar versions to use @msau42  |   <li> Complex configuration for the single CSI Sidecar component <li> Difficulty expressing per CSI Sidecar configuration  e.g. kube-api-qps, kube-qpi-burst <ul><li> global flag, override through a CSI sidecar flag  e.g. kube-api-qps -> attacher-kube-api-qps | 




<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->


#### Glossary
- Individual repository:  An existing repository in the kubernetes-csi/ org in Github e.g. the external-attacher repository.
- AIO monorepo(monorepo): The monolithic repository where most of the code of the CSI Sidecars will be migrated.
- Monorepo component:  The source code of an individual repository that is currently being migrated or already migrated to the monorepo. 
- AIO Sidecar Image: The All-In-One sidecar image utilizes a monorepo
- repository root path: The portion of a module path that corresponds to a version control repository’s root directory.


#### AIO Monorepo 

##### Release Management

Use Semantic Versioning in monorepo.

[Alternatives](#releasemanagement)


##### RBAC policy

We designed the AIO monorepo's RBAC policy to mirror that of individual repos, where each controller maintains its own policy. Driver maintainers should apply proper RBAC when enabling specific controllers in AIO
more discuss info in [here](https://docs.google.com/document/d/1z7OU79YBnvlaDgcvmtYVnUAYFX1w9lyrgiPTV7RXjHM/edit?tab=t.0#bookmark=id.l9u181gxf6ie.)

We plan to combine informer caches of different controllers in the [future](#informer-merged)

##### Command Line

Divided the command lines into two types, a generic command line whose configuration is common to all controllers and is configured only once, and the other type of command lines whose configuration is different for each controller. these command lines each has a new unique name. prefix with the controller name.

```yaml
        - name: csi-sidecars
          command:
            - csi-sidecars
            - "--csi-address=unix:/csi/csi.sock"
            # similar style as kube-controller-manager
            - "--controllers=attacher,provisioner,resizer,snapshotter"
            - "--feature-gates=Topology=true"
            # leader election flags for all the components as one
            - "--leader-election"
            - "--leader-election-namespace=kube-system"
            # global timeouts
            - "--timeout=30s"
            # per controller specific flags are prefixed with the component name
            - "--attacher-timeout=30s"
            - "--attacher-worker-thread=100"
            - "--provisioner-timeout=30s"
```

example PR: https://github.com/kubernetes-csi/external-attacher/pull/620


##### Code synchronization

During the transition phase (before individual repositories are fully deprecated), code changes (especially bug fixes and CVE patches) need to be synchronized from individual repositories into the AIO MonoRepo.

This process will be automated using [shell scripts](https://github.com/mauriciopoppe/csi-sidecars-aio-poc/blob/main/hack/do_sync.sh).  This sync script will potentially performing necessary adjustments (like import path updates if needed by the dependency strategy).


##### Individual repo history

The Git history from each individual repository must be preserved during the consolidation into the AIO MonoRepo.

This is critical for traceability. It allows developers investigating bugs or changes in the MonoRepo to easily track the origin of the code back to its specific commit in the individual repository's history using standard Git tooling (git blame, git log).

This will likely be achieved using Git strategies designed for repository merging, such as careful merge commits, git graft, or potentially git replace during the initial import phase, ensuring commit hashes remain discoverable. Tooling will be developed to aid this process.


##### Reproducible builds & Dependencies Management

To keep reproducible builds of a Monorepo, when syncing codes from individual repositories, it is critical to enforce consistent dependency versions across all MonoRepo component.  Avoiding discrepancies that could break builds or introduce compatibility issues. 

To simplify dependency management, including ensuring reproducible builds, we will adopt a single go.mod and go.work file at the root directory. Nested, imported repositories will not have their own go.mod files.

[Alternatives](#reproduciblebuilds)


###### Future Sidecar Integration

Eligibility: Any sidecar that achieves General Availability (GA) is eligible for integration into the AIO monorepo.
Process: We will provide clear documentation outlining the steps for integration. While only GA projects are eligible for final integration, we encourage owners of pre-GA sidecars to follow these instructions as well. This will ensure your project is properly prepared for a smooth integration once it reaches GA status.
Ownership: The original developers will retain full ownership and control of their sidecar project.


##### Lease migration strategy

To safely migrate from multiple legacy leader-election leases (e.g., for csi-provisioner, csi-resizer) to a single, consolidated lease for the new all-in-one (AIO) csi-sidecars container. This process must prevent a "split-brain" scenario where both old and new sidecars are active simultaneously.

1. Acquire New Lease: The new process(in csi-lib-utils) starts and acquires the consolidated lease.
2. Check for Migration Marker: It inspects the lease object for a migration.csi.k8s.io/status: completed annotation.
   - If Marker Exists: The migration is already complete. The proceeds to run the controllers immediately.
   - If Marker is Missing: Migration is required. The process proceeds to the next step.
3. Attempt to Seize Old Leases: The process attempts to acquire the leases of all legacy sidecars (e.g., provisioner, resizer). This is an all-or-nothing check.
   - On Success: This confirms no old sidecars are active. The process then:
        1. Adds the migration.csi.k8s.io/status: completed annotation to the new lease.
        2. Releases the old leases it just acquired.
        3. Starts the main controllers.
   - On Failure: This means at least one old sidecar is still running and holds its lease. The AIO process will:
        1. Log a clear error message instructing the user to scale down the legacy sidecar deployment.
        2. Exit with an error, causing the pod to enter a CrashLoopBackOff state. This safely prevents a "split-brain" scenario and alerts the user to the required manual intervention.


#### Risks And Mitigations

- Breaking Changes Amplification: Breaking changes in one component forces the single release to be a breaking change

- Vulnerability Blast Radius: Vulnerability that might affects one component affects all other components

  This risk is considered acceptable. An analysis showed that most recent vulnerabilities were in shared dependencies (like parsers or metric libraries) that affected all sidecars anyway. The benefit of simplified patching outweighs this risk.
  
  Full discussion in: https://docs.google.com/document/d/1SD4YRas_qXMP363L4j3WBTV_F9anq-5FM5gdGmJq7h0/edit?usp=sharing

- Panic Propagation: Panic in one component restarts the sidecar
  
  For each sidecar define where in the stack a panic should be caught to possibly restart the controller. 
  
  List of fixed issues related with panics:
  - https://github.com/kubernetes-csi/external-provisioner/issues/839
  - https://github.com/kubernetes-csi/external-provisioner/issues/582 
  - https://github.com/kubernetes-csi/external-attacher/issues/502

  > panic like OOM doesn't count into this type(perhaps no good way to reduce the blast radius)

- Synchronization Complexity: Maintaining code consistency between the AIO MonoRepo and the individual repositories during the transition period (before full deprecation) requires careful management.



#### Development workflow

### MileStone

![overview](./aio6.png "overview of definition of different workflows and milestones")

> **workflow1:**

#### Milestone-modify-entrypoints-of-existing-sidecars-to-integrate-it-seamlessly-with-the-AIO-sidecar

Objective: Refactor the CSI Sidecar entrypoint (e.g. cmd/external-attacher/main.go) so that it also exposes a public function that can be reused from both the existing cmd/external-attacher/main.go and from the AIO Sidecar main.

Tasks:

1. For {external-attacher, external-provisioner, ...} split the main function
2. For {external-attacher, external-provisioner, ...} add per sidecar specific flags
3. Introduce the concept of global flags in the csi-lib-utils [already merged]
  - https://github.com/kubernetes-csi/csi-lib-utils/pull/202
4. Modify the individual sidecar entrypoint to reuse the global flags


> **workflow2:**
#### Milestone-setting-up-a-Kubernetes-CSI-Storage-Repository-with-nested-directory-synchronization

Objective: Create a new repository and mirror the nested directories of the existing sidecars to the new repository.

Tasks:

1. Create ```kubernetes-csi/csi-sidecars``` repository  
2. Mirror the nested directories of the all the seven sidecars repo to the new repository.
3. Add a README.md to the new repository.

#### Milestone-Build-the-project-using-a-modified-copy-of-release-tools

Objective: Use the release tools to build the project into AIO Sidecar images

Tasks:

1. Add new release logic of the release tools to support the AIO monorepo and individual repos at same time
2. Build the project into AIO Sidecar image with the release tools


<a id="e2e-test-passed"></a>
#### Milestone-set-up-new-test-infra-jobs-to-test-the-project-through-the-hostpath-CSI-Driver

Objective: Ensure the AIO MonoRepo is testable using existing e2e tests and new CI infrastructure.

Tasks:

1. Modified the test infra jobs to support the new repository
2. Validate prow jobs against new repo
3. Set up github actions to trigger tests for every new PR, including all the e2e test of individual repo

#### Milestone-ready-to-accept-PR-from-community

Objective: Once individual repositories enter the FeatureFreeze state, the monorepo will be open to accept PRs from contributors of those repositories.

Tasks:
1. Setup github actions(unit, golangci, etc) in new monorepo 
2. create CONTRIBUTING.md guidelines specific to the MonoRepo.

---

> **workflow3** 

<a id="migration-path-definition"></a>
#### Milestone-define-the-path-for-2-CSI-Drivers-to-be-migrated.

Objective: Develop detailed migration steps/examples for at least two representative CSI drivers.

#### Milestone: Have instructions for CSI Driver authors

Objective: Inform and guide CSI driver maintainers on how to adopt the new AIO sidecar model.

Tasks:
1. Socialize the KEP, document the migration process clearly.


<a id="accepted-by-three-cloud-vendor"></a>
#### Milestone-three-cloud-vendors-start-using-the-monorepo-component-for-multi-k8s-minor-releases 

Objective: 3 CSI Drivers using the AIO sidecar for 3 consecutive k8s minor releases.

Task:
1. utilizing the provided migration instructions.
2. Identify and support 3 cloud vendors using the AIO sidecar image in production across 3 consecutive Kubernetes minor releases


#### Milestone-accept-PR-from-community

Objective: Transition development fully to the MonoRepo as individual repositories freeze.

Tasks:
1. Mark external-provisioner as featurefreeze state
2. Accept external-provisioner Monorepo component's PRs
3. Mark external-attacher as featurefreeze state
4. Accept external-attacher Monorepo component's PRs
5. ....


> **workflow4** 
#### milestone-all-individual-repo-has-been-into-featurefreeze-state

objective: Systematically stop new feature development in individual repositories.

task:
1. Announce FeatureFreeze dates per individual repo
2. coordinate with maintainers to stop merging feature PRs of individual repo
3. merge pending PRs to the specific individual repo
4. Formally mark individual repo as feature-frozen.
5. Repeat sequentially for all individual repos.


<a id="all-individual-repo-deprecated"></a>
#### Milestone-all-individual-repo-has-been-into-deprecated-state

Objective: To gracefully deprecate individual repository while maintaining clear communication with its users and contributors, ensuring a smooth transition to monorepo.

Task:
1. Write a deprecation notice to the specific individual repo
2. Create a release in the individual repo and mark it as deprecated
3. Notify key contributors, and users of the planned deprecation through mailing lists
4. Assist Users in transitioning to monorepo through issues or slack.
5. Repeat sequentially for all repos.


<a id="informer-merged"></a>
#### Milestone-merge-sidecar-informer-caches

Objective: To merge the sidecar informer caches, which will allow us to use cache more efficiently.

This is a nice improvement that shouldn't be part of the MVP yet. 
It will happen after all of the CSI sidecars have been deprecated or migrated to the monorepo, and we will start it in another KEP 

### Questions and Answers

#### Project Goal and Motivation

Q: Will the snapshot-controller be part of the main AIO sidecar binary?

A: No. While the code for the snapshot-controller will be in the monorepo, it will be built as a separate binary and image. This is because it's not a true sidecar and is not deployed with the CSI driver. Including it in the same binary was considered an anti-pattern.

Q: How will shared resources like leader election and Kubernetes API informers be handled?

A: The single binary will use a shared leader election mechanism and a shared informer for Kubernetes resources. This is a key benefit of the proposal, as it improves performance and reduces resource consumption.

#### Development and Release Process

Q: What is the strategy for developing new features?

A: After a "hard cut" date, all new feature development will take place exclusively in the monorepo. This approach avoids the complexity of trying to maintain features in parallel across both the new monorepo and the old individual repositories.


### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->


##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->



##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->


##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->


##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

### Graduation Criteria

#### AIO MonoRepo state definition

- Design: The initial planning and definition phase (current state described in documentation).
- [Alpha](#e2e-test-passed): 
    Technical feasibility established. All seven sidecar repositories' code has been integrated into the MonoRepo structure, and all original end-to-end (e2e) tests from the individual repositories pass successfully within the MonoRepo's test infrastructure.
- [Beta](#migration-path-definition) (production-verified): 
    The MonoRepo is considered stable enough for early adoption by cloud vendors in production environments. Clear migration paths for CSI drivers are defined and documented. 
- [GA](#accepted-by-three-cloud-vendor) (released): 
    The MonoRepo actively maintained, and accepts contributions (PRs) from the SIG Storage developer community. Development focus shifts from individual repositories to the MonoRepo components. Requires adoption and use in production by at least three cloud vendors.
- [Standalone](#all-individual-repo-deprecated): 
    Final state. The AIO MonoRepo is the source of truth. Code synchronization from individual repositories is no longer necessary as they are all deprecated.

Beta graduation would be at least 2 CSI Drivers using the AIO sidecar for at least 2 consecutive k8s minor releases.
GA graduation would be at least 2 CSI Drivers using the AIO sidecar for 3 consecutive k8s minor releases.


#### Individual repository state definition

- Released: current state of individual repos
- FeatureFreeze: 
    - Any new feature PRs are not allowed to be filed to the master branch or release-X branches(Controlled by the individual repo maintainer, categorize it and reject it if it's a feature)
    - SIG Storage Developer file the feature PRs to AIO MonoRepo 
    - Except for the serious bugfixes or CVE fixes PRs (only from individual repo maintainer) which can be merged in master and backported to the other release-X branches
- Deprecated:
    - Active maintenance stops.
    - Eventually, building new images from this repository ceases (dependent on the full migration of all sidecars).
    - (future) archive it but not at the same time as the deprecation time, this is a terminal state so we can't undo it


![state change](./aio11.png "overview of workflow definition")

#### Migration Process

The migration follows a phased approach:

- Foundation & Setup: Create the new AIO MonoRepo, mirror the code (preserving history), adapt build/release tooling, and establish comprehensive test infrastructure (unit, integration, e2e, CI/GitHub Actions).
- Integration: Refactor the entry points (main.go) of individual repository to be callable functions, enabling them to run both standalone (for backward compatibility tests) and as part of the unified AIO binary, introducing global and component-specific flags.
- Adoption & Community Transition: Provide clear documentation and migration guidance for CSI driver authors. Engage with cloud vendors to test and adopt the AIO sidecar image in production (Beta -> GA trigger). Open the MonoRepo for community contributions as individual repositories enter FeatureFreeze.
- Individual Repository Phase-Out: Sequentially transition each individual repository into FeatureFreeze, followed by Deprecated status, communicating clearly with users and maintainers.
- Finalization: Once all individual repositories are deprecated, the AIO MonoRepo reaches the Standalone state.

![migration process](./aio10.png "")


### Upgrade / Downgrade Strategy


The entire switchover is relatively simple, as it does not involve a gradual upgrade of the kubernetes controller plane components and data plane component, only the yaml and image of the csi components need to be upgraded, and the rollback is achieved directly through ```kubectl rollout```.

### Version Skew Strategy

Nothing in particular.


## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->


It's actually not a feature, but we can enable it by deploy new version of csidriver and disable it by delete the new version and redeploy the old version

###### Does enabling the feature change any default behavior?

This won't make any changes to the default behavior of Kubernetes.


###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

It's actually not a feature, it's kind of architectural change. so user can deploy old version csi driver to disable it.


###### What happens if we reenable the feature if it was previously rolled back?

Nothing happened, it will act as usually

###### Are there any tests for feature enablement/disablement?

Yes. We will add unit tests with and without the feature gate enabled.


### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

No, it will not impact already running workloads.


###### What specific metrics should inform a rollback?

Should be aware of pvc/pv and pod related persistent external storage failures event


###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?


<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No, it does not.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?


Determine whether the `csi-provisioner` deployment includes a AIO Sidecar image by inspecting its container configuration.

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

Only if their csi plugin are working correctly.

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?


<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?


<!--
Pick one more of these and delete the rest.
-->
The SLI for the CSI AIO project is based on individual sidecars' SLIs, with metrics exposed by [csi-lib-utils](https://github.com/kubernetes-csi/csi-lib-utils/blob/master/metrics/metrics.go)

- [X] Metrics
  - Metric name: operationsLatencyMetricName
  - Components exposing the metric: Monorepo component

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Nothing in particular.

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

No.

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

No, It doesn't increase the number of API calls. In fact, it will decreasing it


###### Will enabling / using this feature result in introducing new API types?

Nope


###### Will enabling / using this feature result in any new calls to the cloud provider?

Nope

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Nope

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Nope

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

It will reduce disk and memory usage due to merging image and cache informer of csi driver

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

Nope

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?


###### What are other known failure modes?


<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

###### What steps should be taken if SLOs are not being met to determine the problem?


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

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->
#### ReleaseManagement
We are consider to switch semantic version to k8s version, there are some pros and cons 

pros:

- We don't need to reinvent the wheel about what our dev process is going to look like, we follow the same docs as k8s https://kubernetes.io/releases/release/. This is tried and tested for many releases
- Cluster administrators would know which version to use to match their CSI Driver deployment e.g. for a k8s 1.27 cluster they'd use the 1.27 release of the CSI Sidecar.

cons:

- Breaking changes might happen in a minor release, Cluster administrators MUST read sidecar release notes considering breaking changes before working on a big release.
- Version skew scenario becomes confusing for the cluster administrator e.g. they deploy the CSI Sidecars v1.x, cluster is upgraded to v1.{x+3} (CP upgrade first, NP later), nodepools would have CSI sidecar at v1.{x+3} with kubelet at v1.x
- k/k at 1.27.5 - CSI 1.27.0 or (different mapping still)

After investigation, we found that there isn't clear advantage to switch to k8s versioning, so we are going to stick with the semantic versioning.

#### ReproducibleBuilds


1. Using Go Workspaces (introduced in Go 1.18)

Using ```go work init``` and ```go work sync``` to manage multiple go.mod files within the MonoRepo.

2. Single Root

Removing monorepo component level go.mod/go.sum files and managing all dependencies via a single go.mod/go.sum at the repository root path.

Conclusion: To simplify dependency management, including ensuring reproducible builds, we will adopt a single go.mod and go.work file at the root directory. Nested, imported repositories will not have their own go.mod files.




## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
