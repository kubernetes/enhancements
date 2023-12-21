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
# KEP-4191: Split Image Filesystem
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
  - [Goals](#goals)
  - [Possible Extensions in Post Alpha](#possible-extensions-in-post-alpha)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
  - [User Deployment Options](#user-deployment-options)
    - [Current Deployment Options](#current-deployment-options)
      - [Image File system and Node (Kubelet) FS same](#image-file-system-and-node-kubelet-fs-same)
      - [Node FS and Image Filesystem separated](#node-fs-and-image-filesystem-separated)
    - [New Deployment Options](#new-deployment-options)
      - [Node And Writeable Layer on same disk while Images stored on separate disk](#node-and-writeable-layer-on-same-disk-while-images-stored-on-separate-disk)
  - [Comment on Future Extensions](#comment-on-future-extensions)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [CRI](#cri)
  - [Stats Summary](#stats-summary)
  - [Stats Provider](#stats-provider)
    - [CAdvisor Stats Provider](#cadvisor-stats-provider)
    - [CRI Stats Provider](#cri-stats-provider)
  - [Eviction Manager](#eviction-manager)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha [Release the CRI API and Kubelet Changes]](#alpha-release-the-cri-api-and-kubelet-changes)
    - [Alpha Part 2 [CRI-O, E2E Tests and CRITools]](#alpha-part-2-cri-o-e2e-tests-and-critools)
    - [Alpha To Beta Promotion](#alpha-to-beta-promotion)
    - [Stable](#stable)
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
  - [Kubelet Disk Stats in CRI](#kubelet-disk-stats-in-cri)
  - [Add container filesystem usage to image filesystem array](#add-container-filesystem-usage-to-image-filesystem-array)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [] (R) Production readiness review completed
- [] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP is about enhancing Kubelet to be aware if a container runtime splits the image filesystem.  
Aware in this case means that garbage collecting images, containers and reporting disk usage is all functional.

## Motivation

Kubelet has two distinct filesystems: Node and Image. In typical deployments, users deploy Kubernetes where both the Node and Image filesystems are on the same disk. There are some requests to separate the storage into separate disks.  The most common request is to separate the writable layer from the read-only layer. Kubelet and Container data would be stored on the same disk while images would have their own disk.  This could be beneficial because images occupy a lot of disk space while the writeable layer is typically smaller.

Container IO can impact Kubelet and adding the ability for more disks could increase performance of Kubelet.

However, it is not possible to separate the image layers and container writable layers on different disks.

In the current implementation of separate disks, containers and images must be stored on the same disk.  So garbage collection, in case of node pressure (really image disk pressure) would GC images/containers on the image filesystem.

If one separates writable layer (containers) from readable layer (images), then garbage collection and statistics must account for this separation.  Today this could potentially break Kubelet if the container runtime configures storage in this way.

One downside of the separate disk is that pod data can be written in multiple locations.  The writeable layer of a container would go on the image filesystem and volume storage would go to the root fs.  There is another request to separate the root and the image filesystem to be writeable and read-only respective.  This means that pod data can be written on one disk while the other disk can be read-only.  Separting the writeable layer and the read-only layer will achieve this.

### Goals

- Kubelet should still work if images/containers are separated into different disks.
  - Support writable layer being on same disk as Kubelet
  - Images can be on the separate filesystem.

### Possible Extensions in Post Alpha

Kubelet, Images and Containers on all separate disks.

This case is possible with this implementation as containerfs will be set up to read file statistics from a separate filesystem.  However, this is not in scope for Alpha.  
If there is interest in this, this KEP could be extended to support this use case. Main areas to add would be testing.

### Non-Goals

- Multiple nodes can not share the same filesystem.
- Separating kubelet data into different filesystems.
- Multiple image and/or container filesystems
  - This KEP will start support for this but more work needs to be done to investigate CAdvisor/CRIStats/Eviction to support this.

## Proposal

### User Stories

#### Story 1

As a user, I would like to have my node configured so that I have a writeable filesystem and a readable filesystem.  
Kubelet will write volume data and the container runtime will write writeable layers to the writeable filesystem while the container runtime will write the images to the read-only filesystem.

### User Deployment Options

It is not a common pattern to separate the filesystems in most Kubernetes deployments.
We will summarize the existing configurations that are possible today.

#### Current Deployment Options

##### Image File system and Node (Kubelet) FS same

sda0: [writeable layer, emptyDir, logs, read-only layer, ephemeral storage]

This is the default configuration for Kubernetes.  If container runtime is not configured in any special way, then NodeFS and ImageFS are assumed to be the same.

If the node only has a nodefs filesystem that meets eviction thresholds, the kubelet frees up disk space in the following order:

- Garbage collect dead pods and containers
- Delete unused images

The way that pods are ranked for eviction also changes based on the filesystem.

Kubelet sorts pods based on their total disk usage (local volumes + logs & writable layer of all containers)

[Node Pressure Eviction](https://kubernetes.io/docs/concepts/scheduling-eviction/node-pressure-eviction/#reclaim-node-resources) lists the possible
options for how to reclaim resources based on filesystem configuration.

[Ephemeral-Storage](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#configurations-for-local-ephemeral-storage) explains how ephemeral-storage tracking works with different filesystem configurations.

##### Node FS and Image Filesystem separated

sda0: [emptyDir, logs, ephemeral storage]
sda1: [writeable layer, read-only layer]

If the node has a dedicated imagefs filesystem for container runtimes to use, the kubelet does the following:

- If the node filesystem meets the eviction thresholds, the kubelet garbage collects dead pods and logs
- If the imagefs filesystem meets the eviction thresholds, the kubelet deletes all unused images and containers.
- If ImageFS has disk pressure we will mark node as unhealthy and not allow new pods to be admitted until image disk pressure is gone.

In case of disk pressure on each filesystem, what is garbage collected/stored on the disk?

Node Filesystem:

- Logs
- Pods
- Ephemeral Storage

Image Filesystem:

- Images
- Containers

CAdvisor detects the different disks based on mountpoints.  So if a user mounts a separate disk to /var/lib/containers,
Kubelet will think that the filesystem is split.

Users can write the writeable layer of a container and that would be stored on the image filesystem while data written in volumes can be written to the node filesystem.

Since this split case has two different filesystems that can have disk pressure, Pods are ranked differently based on what is experencing disk pressure.

Node Pressure:

- local volumes + logs of all containers

Image Pressure:

- sorts pods based on the writeable layer usage of all containers

#### New Deployment Options

##### Node And Writeable Layer on same disk while Images stored on separate disk

sda0: [writable layer, emptyDir, logs, ephemeral storage]
sda1: [read-only layer]

A goal is to allow kubelet to have separate disks for read-only layer and everything else could be stored on the same disk as Kubelet.

In case of disk pressure on each filesystem, what is garbage collected/stored on the disk?

Node Fileystem:

- Pods
- Logs
- Containers
- Ephemeral Storage

Image Filesystem:

- Images

Node Filesystem should monitor storage for containers in addition to ephemeral storage.

### Comment on Future Extensions

We foresee interest in the future for other use cases.
So we want to comment on what work would be required to support these usecases.

One extension can be multiple filesystems for images and containers.  
The API allows for a list of filesystem usage per images and containers but there has been no work done to support
this in the container runtimes or in Kubelet.

CAdvisor and Stats would need to be enhanced to allow for configurable amount of filesystems.  
Currently, eviction manager is harded code to support a 1-to-1 relationship with a filesystem and a eviction signal.

The following cases could be configured but we are not targeting these at the moment.

a. Node, Writeable layer and Image on separate filesystems.
b. Node and Images on same filesystem while Writeable layer on separate filesystem.

### Risks and Mitigations

By splitting the filesystem we allow more cases than what we currently support in Kubelet.
To avoid bugs, we will validate on cases we don't currently support in Kubelet and return an error.

The following cases will be validated and we will return an error if container runtime is set up for this:

- More than 1 filesystem for images and containers.

We will validate if the CRI implementation is returning more than 1 filesystem and log a warning.

A major risk of this feature will be increased evictions due to the addition of a new filesystem.  
The eviction manager monitors image filesystem, node filesystem and now container filesystem for disk pressure.  
Disk pressure can be inodes or storage limits.  
Once the disk is exceeds the limits set by `EvictionSoft` or `EvictionHard`, then that node will eventually be marked as having disk pressure.  
Garbage collection of containers, images or pods will be kicked off (depending on which filesystem experiences disk pressure).  
New workloads will not be accepted by that node until disk pressure resolves itself either by garbage collection removing enough or manually intervention.

A mitigation for this is to initially support the case of the writeable layer being on the node filesystem (containerfs = nodefs), so we really are only monitoring two filesystems for pressure.

## Design Details

### CRI

We will switch to using `ImageFsInfo` but this will be guarded by a feature gate.

CRI-O and Containerd return a single element in this case and Kubelet does not assume that there are multiple values in this array.  Regardless, we add an array to ImageFsInfoResponse.

```golang
// ImageService defines the public APIs for managing images.
service ImageService {
…
    rpc ImageFsInfo(ImageFsInfoRequest) returns (ImageFsInfoResponse) {}
}

message ImageFsInfoResponse {
    // Information of image filesystem(s).
    repeated FilesystemUsage image_filesystems = 1;
    + // Information of container filesystem(s).
    + // This is an optional field if container and image
    + // storage are separated.
    + // Default will be to return this as empty.
    + repeated FilesystemUsage container_filesystems = 2;
}
```

It is expected of the CRI implementation to return a unique identifier for images and containers so the Kubelet can ask CRI if the objects are stored on separate disks.
In the dedicated disk for container runtime, images_filesystem and container_fileystem will be set to the same value.

The CRI implementation can set this as needed.  The image and container filesystems are both arrays so this provides some extensibility in case these are stored on multiple disks.  

Container runtimes will need to implement ImageFsInfo

- [CRI-O Implementation](https://github.com/cri-o/cri-o/blob/main/server/image_fs_info.go)
- [Containerd implementation](https://github.com/containerd/containerd/blob/main/pkg/cri/server/imagefs_info.go)

An Alpha->Beta goal would be to have an implementation of `crictl imagefsinfo` that can allow for more detailed reports of the image fs info.

See [PR](https://github.com/kannon92/cri-tools/pull/1) for an example.

### Stats Summary

Stats Summary has a field called runtime and we will add a containerFs to the runtime field.

```golang

// RuntimeStats are stats pertaining to the underlying container runtime.
type RuntimeStats struct {
// Stats about the underlying filesystem where container images are stored.
// This filesystem could be the same as the primary (root) filesystem.
// Usage here refers to the total number of bytes occupied by images on the filesystem.
// +optional
ImageFs *FsStats `json:"imageFs,omitempty"`
+ // Stats about the underlying filesystem where container's writeable layer is stored.
+ // This filesystem could be the same as the primary (root) filesystem or the ImageFS.
+ // Usage here refers to the total number of bytes occupied by the writeable layer on the filesystem.
+ // +optional
+ ContainerFs *FsStats `json:"containerFs,omitempty"`
}
```

In this KEP, ContainerFs can either be the same as ImageFs or NodeFs.

We will add a more detailed function for ImageFsStats in the Provider Interface

```golang
type containerStatsProvider interface {
...
ImageFsStats(ctx context.Context) (*statsapi.FsStats, *statsapi.FsStats, error)
}
```

If we have a single image filesystem then ImageFs includes both writable and read-only layer.  In this case, `ImageFsStats` will return an identical object for ImageFs and ContainerFs.

In a case where the container runtime does not return a container filesystem, we will assume that the image_filesystem=container_filesystem.
This allows us kubelet to support container runtimes that have yet implemented the CRI implementation in `ImageFsInfo`.

### Stats Provider

The CRI Stats Provider uses the `ImageFsInfo` to get information about the filesystems,
but the CAdvisor Stats Provider uses `ImageStats` which will list the images and computes the overall size from this list.

This switch will be guarded by a feature gate.

#### CAdvisor Stats Provider

CRI-O uses the CAdvisor Stats provider.

CAdvisor has plugins for each container runtime under containers. [CRI-O](https://github.com/google/cadvisor/tree/master/container/crio)

The plugin in CRI-O relies on the endpoints `info` and `container/{id}`.  Info is used to get information about the storage filesystem and
container gets information about the mount points.  CRI-O will add a new field `storage_image` to tell when we are splitting the filesystem. 

This is used to gather file stats.  

CAdvisor labels CRI-O images as `crio-images` and that is assumed to be the mountpoint of the container.  When splitting the filesystem this
ends up pointing to the writeable layer of the container.

We will propose a new label in CAdvisor: `crio-containers` will point to the writeable layer and `crio-images` will point to the read-only layer.

In case of no split system, `crio-images` will be used for both layers.

We have created [CAdvisor PR](https://github.com/google/cadvisor/pull/3395) to suggest how CAdvisor's can be enhanced to support a container filesystem.

#### CRI Stats Provider

Containerd uses the CRI Stats Provider.

CRI Stats Provider calls `ImageFsInfo` and uses the `FsId` to get the filesystem information from `CAdvisor`.  One could label the `FsId` for the writeable layer and this will be used to get the file system information for the container filesystem.

No changes should be necessary in CAdvisor for this provider.

### Eviction Manager

A new signal will be added to the eviction manager to reflect the filesystem for the writeable layer.  
For the first release on this KEP, this will be either nodefs or imagefs.
In separate disks, this could be a separate filesystem.

```golang
 // SignalContainerFsAvailable is amount of storage available on filesystem that container runtime uses for container writable layers.
 SignalContainerFsAvailable Signal = "containerfs.available"
 // SignalContainerFsInodesFree is amount of inodes available on filesystem that container runtime uses for container writable layers.
 SignalContainerFsInodesFree Signal = "containerfs.inodesFree"

```

We do need to change the garbage collection based on the split filesystem case.

(Split Filesystem) Writable and root plus ImageFs for images

- NodeFs monitors ephemeral-storage, logs and writable layer.
- ImageFs monitors read-only

Eviction manager decides the priority of eviction based on which filesystem is experencing pressure.

If Node filesystem experiences pressure, ranking is done as `local volumes + logs of all containers + writeable layer of all containers`

If Image filesystem experiences pressure, ranking is done as `storage of images`.

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

- (pkg/kubelet/eviction): Sep 11th 2023 - 69.9
- (pkg/kubelet/stats): Sep 11th 2023 - 77.9
- (pkg/kubelet/server/stat): Sep 11th 2023 - 55

This KEP will enhance coverage in the eviction manager by covering the case where `dedicatedImageFs` is true.  
There is currently little test coverage when a separate imagefs is used.  [Issue-120061](https://github.com/kubernetes/kubernetes/issues/120061) has been created to help resolve this.

We will also provide test cases for rolling back the changes in the eviction manager.

We will add unit tests to cover using `ImageFsInfo` and we will have testing around rolling back this feature.

We will add test cases for `ImageStats` in case of positive and negative usage of the feature.  In negative cases, we will assume containerfs=imagefs.  In positive test cases, we will allow different configurations of the image filesystem.

##### Integration tests

Typically these type of tests are done with e2e tests.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

This code affects stats, eviction and the summary API.  

There should be e2e tests for each of these components with a split disk.  
However, there are a few complications with this goal.

1. E2E tests around eviction with a single disk are currently [CRI-O-eviction](https://testgrid.k8s.io/sig-node-cri-o#ci-CRI-O-cgroupv1-node-e2e-eviction) and [containerd-eviction](https://testgrid.k8s.io/sig-node-containerd#node-kubelet-containerd-eviction) failing.  

2. There is zero test coverage around a separate image filesystem.  There is an issue to improve this at the unit test level.

1 can be addressed by investigating the eviction tests and figure out the root cause of these failures.

As part of this KEP, we should add testing around separate disks in upstream Kubernetes.  Since this is already a supported use case in Kubelet, there should be testing around this.

Kubelet/CRI-O should be set up with configuration for a separate disk.  Eviction and Summary E2E tests should be added in the case of a separate disk.

And tests for split image filesystem should be added.

E2E Test Use Cases addition:

- E2E tests for summary api with separate disk.
  - Separate Disk - ImageFs reports separate disk from root when disk is mounted.
  - Split Disk - Writeable layer on Node, read-only layer on ImageFs
- E2E tests for eviction api with separate disk
  - Replicate existing disk pressure eviction e2e tests with disk.

E2E tests for separate disk:

- Presubmits - Added [separate-imagefs](https://testgrid.k8s.io/sig-node-cri-o#pr-crio-cgroupv2-imagefs-e2e-diskpressure)
- Presubmits - Added [conformance test for imagefs](https://testgrid.k8s.io/sig-node-cri-o#pr-crio-cgrpv2-imagefs-e2e)

### Graduation Criteria

#### Alpha [Release the CRI API and Kubelet Changes]

CRI API changes are composed in containerd and CRI-O so the CRI API must be released first.

- Using `ImageFsInfo` is guarded with a feature gate.
- Implementation for split image filesystem in Kubernetes.
  - Eviction manager modifications in case of split filesystem.
  - Summary and Stats Provider implementations
- CRI API merged.
- Unit tests
- E2E tests to cover separate image filesystem
  - It is not possible to have e2e tests for split filesystem at this stage.

#### Alpha Part 2 [CRI-O, E2E Tests and CRITools]

Shortly after this release and new CRI package, projects that consume the CRI API
can be updated to use the new API features.

- At least one CRI implementation supports split filesystem
- E2E tests supporting the CRI implementation with split image filesystem
- CRI-Tool changes for image fs

#### Alpha To Beta Promotion

- Gather feedback on other potential use cases.
- Always set `KubeletSeparateDiskGC` to true so `ImageFsInfo` is used instead of `ImageStats` in all cases.
- Always set `KubeletSeparateDiskGC` to true so that eviction manager will detect split file system and handle it correctly.

#### Stable

- More than one CRI implementation supports split filesystem

### Upgrade / Downgrade Strategy

There are two cases that this feature could impact users.

Case 1: Turning the feature on with no split filesystem.
In this case, the main difference will be that clusters that use the `CAdvisor Stats Provider`, we will switch to using `ImageFsInfo` to report
the image filesystem statistics.  Turning off this feature will use `ImageStats`.

Case 2: Feature is turned on and the container runtime is set up to split filesystem.
In this case, rolling back this feature is only supported if one also configures the container runtime to not split the filesystem.

Another case that is important to highlight is that some container runtimes may not support split filesystem,
We will guard against a container runtime not returing a container filesystem in `ImageFsInfo`.
In this case we would assume that the image filesystem and the container filesystem are identical.

Since older versions of the container runtimes do not have the ability to split the filesystem, we don't foresee much issue with this.
Kubelet will not behave differently if the container and image filesystems are identical.

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

### Version Skew Strategy

The initial release of this will be the CRI API and changes to Kubelet.  

We do not assume that container runtimes must implement this API so we will assume a single filesystem for images.

Once the container runtimes implement this API and the feature gate is enabled, then the feature would be active.

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

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

If a container runtime is configured to split the image filesystem, there is no really good way to roll these changes back.
We will include a feature gate for best practices to guard against our code.

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: KubeletSeparateDiskGC
  - Components depending on the feature gate: Kubelet
- [x] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
    It depends.  If the control plan is run on Kubelet, then yes.  If the control plane is not run on Kubelet, then no.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? Yes. One needs to restart the container runtime on the node to turn on support for split image filesystem.

Our recommendation to roll this change back:

1. Configure your container runtime to not split the image filesystem.
2. Restart the container runtime.
3. Restart Kubelet with feature flag off.

###### Does enabling the feature change any default behavior?

Yes, we will switch to using `ImageFsInfo` to compute disk stats rather than call `ImageStats`.

The eviction manager will monitor the container filesystem if the image filesystem is split.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

There are two possibilities for this feature.

1. Container runtime is configured for split disk
2. Container runtime is not configured for split disk.

If the feature toggle is disabled in 1, then turning off the feature will tell eviction manager that the containerfs=imagefs.  
The container garbage collection will try to delete the writeable layer on the image filesystem which may not be there.
Kubelet will still run but there could be a possibility that the container filesystem will grow unchecked and eventually cause disk pressure.

In case 2, rolling back this feature will be possible because we will use `ImageStats` to compute the filesystem usage.
Since the container runtime is configured to not split the disk, nothing would really be changed in this case.

###### What happens if we reenable the feature if it was previously rolled back?

Nothing as long as the container runtime is set up to split again.

###### Are there any tests for feature enablement/disablement?

Yes, even though roll back is not supported, we will be switching to using `ImageFsInfo` for stats on the file system.
This will be guarded by a feature gate and we will test negative and positive test cases.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

If the filesystem is not split, this rollout or rollback will be a no-op.

If the filesystem is split and you want to roll back the change that will require a change to the container runtime configuration.

If one does not want to change the container runtime configuration, there could be a possibility that node pressure could happen as garbage collection will not work.
The container filesystem would grow unbounded and would require users to clean up their disks to avoid disk pressure.

###### What specific metrics should inform a rollback?

If a cluster is evicting a lot more pods (`node_collector_evictions_total`) than normal, this could be caused by this feature.

The eviction manager monitors the image filesystem, node filesystem and the container filesystem for disk pressure.
If any of these filesystems are experencing I/O pressure, pods will start being evicted and the eviction manager will trigger garbage collection.
The metric `node_collector_evictions_total` will inform operators that something is wrong because pods will be evicted and until disk pressure resolves itself, new workloads are not able to run.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet.

We are testing with container runtimes not requiring this API implementation for initial alpha.

In future releases, we could test this.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

This feature will be hidden from the users mostly but if an operator wants to know, it is possible to use [crictl](https://github.com/kubernetes-sigs/cri-tools/blob/master/docs/crictl.md).

`crictl imagefsinfo` can be used to determine if the file systems are split.  

`crictl imagefsinfo` will return a json object of file system usage for the image filesystem and the container filesystem.  If the image filesystem is not split, then the image filesystem and container filesystem will have identical statistics.

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as last resort)

`crictl imagefsinfo` will give stats information about the different filesystems.

A user could check the filesystem for containers and images.

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

- [x] Metrics
  - Metric name: node_collector_evictions_total
  - Components exposing the metric: Kubelet

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

The container runtime needs to be able to split the writeable and the read-only layer.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

N/A.

###### Will enabling / using this feature result in any new calls to the cloud provider?

N/A.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

There is an additional field added to the CRI api for `ImageFsInfoResponse`.

- API type: protobuf array of FileSystem Usage
  - Estimated increase in size: 24 bytes and a variable length string for the mount point
  - Estimated amount of new objects: 1 element in the array
- API type: ContainerFilesystem in Summary Stats
  - Estimated increase in size: 24 bytes plus a variable length string for the mount point
  - Estimated amount of new objects: 1 ContainerFilesystem for Summary Stats.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Yes. We are adding a way to split the image filesystem so it will be possible for disk space to be used.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

Yes.  We are adding a way to split the image filesystem so it will be possible for inodes/disk space to be used.

We will add a new eviction api for containerfs to handle a case if the container filesystem has disk pressure.  

The split disk means that we will need to monitor image disk size on the imagefs and the writeable layer on the rootfs.

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

This feature does not interact with the API server and/or etcd as it is isolated to Kubelet.

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

- Initial Draft (September 12th 2023)
- KEP Merged (October 5th 2023)
- Alpha 1 PRs merged (October 31st 2023)

## Drawbacks

This could increase the amount of ways to configure Kubelet to work and provide more difficulty in trouble shooting.

## Alternatives

### Kubelet Disk Stats in CRI

In this case, we considered bypassing CAdvisor and have CRI return node usage information entirely.  This would require container runtimes to report disk usage/total stats in the ImageFsInfo endpoint.

We decided to not go this route as we intend to support only two filesystems so we don’t need a separate tracking filesystem.  We already have node and image statistics so we choose to use either use node or image in this KEP.

If one wants to support the writable layer as an entirely separate disk, then either extensions to CAdvisor or CRI may be needed as one will need to know information about the writable layer disk.

### Add container filesystem usage to image filesystem array

In the internal API, kubelet directly uses the image filesystem array rather than the `ImageFsInfoResponse`.  
To keep API changes minimal, we could have all containerd/cri-o add container filesystems to the image filesystem.
This would work but it would require some additions to the file system usage with a label for images/containers.

We decided to not go this route as there could be more use cases to add to ImageFsInfoResponse that would not fit in the array type.

## Infrastructure Needed (Optional)

E2E Test configuration with separate disks. It may be possible to use a tmpfs for this KEP.
