---
title: Windows RuntimeClass Support
authors:
  - "@patricklang"
owning-sig: sig-windows
participating-sigs:
  - sig-node
reviewers:
  - "@tallclair"
  - "@derekwaynecarr"
  - "@benmoss"
  - "@ddebroy"
approvers:
  - "@dchen1107"
editor: "@patricklang"
creation-date: 2019-10-08
last-updated: 2019-10-15
status: implementable
see-also:
  - "/keps/sig-windows/20190424-windows-cri-containerd.md"
---

# RuntimeClass Support for Windows

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1 - Easy selection of Windows Server releases](#story-1---easy-selection-of-windows-server-releases)
    - [Story 2 - Forward compatibility with Hyper-V](#story-2---forward-compatibility-with-hyper-v)
    - [Story 3 - Choosing a specific multi-arch image](#story-3---choosing-a-specific-multi-arch-image)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Adding new label node.kubernetes.io/windows-build (done)](#adding-new-label-nodekubernetesiowindows-build-done)
    - [Adding annotations to ImageSpec](#adding-annotations-to-imagespec)
      - [ImageSpec changes](#imagespec-changes)
      - [ImageSpec as part of the Image struct](#imagespec-as-part-of-the-image-struct)
      - [Scenarios](#scenarios)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Adding new node label](#adding-new-node-label)
    - [Annotations in ImageSpec](#annotations-in-imagespec)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
    - [E2E Testing with CRI-ContainerD and Kubernetes](#e2e-testing-with-cri-containerd-and-kubernetes)
    - [Unit testing with CRITest](#unit-testing-with-critest)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
      - [Removing a deprecated flag](#removing-a-deprecated-flag)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [Support multiarch os/arch/version in CRI](#support-multiarch-osarchversion-in-cri)
  - [Make the scheduler aware of Multi-arch images](#make-the-scheduler-aware-of-multi-arch-images)
  - [Create a multi-arch Mutating admission controller](#create-a-multi-arch-mutating-admission-controller)
- [Future Considerations](#future-considerations)
  - [Pod Overhead](#pod-overhead)
  - [RuntimeClass Parameters](#runtimeclass-parameters)
- [Reference &amp; Examples](#reference--examples)
  - [Multi-arch container image overview](#multi-arch-container-image-overview)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) [kubernetes/enhancements issue in release milestone](https://github.com/kubernetes/enhancements/issues/1301)
- [x] (R) KEP approvers have set the KEP status to `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

RuntimeClass can be used to make it easier to schedule Pods onto appropriate nodes based on OS, OS Version, CPU Architecture and variant. These are all supported in container distribution & runtimes as part of [Multi-arch container images](#multi-arch-container-image-overview) today.

With Hyper-V available, Windows can run containers multiple Windows OS versions today, and Linux containers may be available in the future. This document proposes controlling those features through RuntimeClass as well, rather than adding new fields to the Pod API and changing Kubernetes scheduling behavior.

## Motivation

There are a few related customer experience problems when running mixed (multiple-OS and/or multiple-CPU-arch) clusters today. Kubernetes scheduling is based solely on what's declared in the deployment as part of the Pod spec. A Pod could be scheduled to an incompatible node and fail to pull. This error causes a retry and backoff loop, and doesn't automatically fail over to another node.

In addition to the stuck pull failure case, there's also a question of user intent. Some containers are published for multiple OS's and/or architectures - this often referred to as multi-arch but the OCI distribution spec actually includes the OS, OS version, CPU architecture and variant. If a node can run multiple permutations of these, then there's not a deterministic way to know what the user intended to run. Running a container under CPU emulation may have performance penalties, or cost more in terms of leased compute time.

Today with only a single supported Windows version (10.0.17763) & runtime (Docker), the problem is easily mitigated with NodeSelectors or Taints. This is documented today in [Guide for scheduling Windows containers in Kubernetes]. Moving these to RuntimeClasses simplifies the experience further, and can simplify deployment YAML for users in Kubernetes v1.16.

Next, SIG-Windows is adding support for [Windows CRI-ContainerD], Windows nodes will be able to handle running multiple Windows OS versions using Hyper-V isolation. This technology could be used to run Linux containers as well, leading to more ambiguity. This KEP aims to resolve the ambiguity of pulling and running a container image and pod when multiple os/versions/architecture/variants may be supported on a single node.

### Goals

- Schedule a Windows Pod to a compatible node
  - Allow matching Windows versions without Hyper-V isolation
  - Allow opt-in on a per-Pod basis to run containers using existing Windows versions with backwards-compatibility provided by Hyper-V on a new Windows OS version node
- Be able to schedule a specific image from a multi-arch manifest on a given node
- Provide a simpler experience (fewer lines of YAML) than adding os and version nodeSelector and tolerations to each Pod

### Non-Goals

- Linux container support on Windows is not a requirement or test target for Kubernetes 1.17, but it's not specifically excluded.
- Running newer Windows OS version containers (Windows Server version 1903) on an older OS version host (Windows Server version 1809). This is not supported today with or without Hyper-V isolation (see [Windows container version compatibility]).

## Proposal

For Kubernetes 1.17, we're proposing three key things to improve the experience deploying Windows containers across multiple versions, and enable experimentation with Hyper-V while ContainerD support is in alpha/beta stages.

1. Add `node.kubernetes.io/windows-build` label using Windows kubelet
2. Add RuntimeHandler to the CRI pull API
3. Use RuntimeHandler and separate ContainerD configurations to control runtime specific parameters such as overrides for OS/version/architecture and Hyper-V isolation.

As a fallback plan, steps 2/3 could use annotations instead during alpha, but this would make things more difficult for users trying out the new features as the Pod specs would change between versions. Comparable solutions based on ContainerD + Kata on Linux are moving away from annotations and to RuntimeClass, so we want to follow the same approach for ContainerD + Hyper-V.

### User Stories

#### Story 1 - Easy selection of Windows Server releases

As of Kubernetes 1.16, [RuntimeClass Scheduling] is in beta and can be used to simplify setting nodeSelector & tolerations. This makes it easier to steer workloads to a suitable Windows or Linux node using the existing labels. With the addition of a new `windows-build` label it can also distinguish between multiple Windows version in the same cluster. For more details on how and why build numbers are used, read [Adding new label node.kubernetes.io/windows-build](#adding-new-label-nodekubernetesiowindows-build) below.

> Note: There's an open PR [website#16697](https://github.com/kubernetes/website/pull/16697) to add `RuntimeClass` examples to existing documentation for Kubernetes 1.16.

- Example of a RuntimeClass that would restrict a pod to Windows Server 2019 / 1809 (10.0.17763)

    ```yaml
    apiVersion: node.k8s.io/v1beta1
    kind: RuntimeClass
    metadata:
      name: windows-1809
    handler: 'docker'
    scheduling:
      nodeSelector:
        kubernetes.io/os: 'windows'
        kubernetes.io/arch: 'amd64'
        node.kubernetes.io/windows-build: '10.0.17763'
      tolerations:
      - effect: NoSchedule
        key: windows
        operator: Equal
        value: "true"
    ```

- Example of a RuntimeClass that would restrict a pod to Windows Server version 1903 (10.0.18362)

    ```yaml
    apiVersion: node.k8s.io/v1beta1
    kind: RuntimeClass
    metadata:
      name: windows-1903
    handler: 'docker'
    scheduling:
      nodeSelector:
        kubernetes.io/os: 'windows'
        kubernetes.io/arch: 'amd64'
        node.kubernetes.io/windows-build: '10.0.18362'
      tolerations:
      - effect: NoSchedule
        key: windows
        operator: Equal
        value: "true"
    ```

#### Story 2 - Forward compatibility with Hyper-V

Once a new version of Windows Server is deployed using [Windows CRI-ContainerD], Hyper-V can be enabled to provide backwards compatibility and run Windows containers based on the previous OS version. Cluster admins can pick between two different approaches to move applications forward. Currently Windows Server has only supported backwards compatibility with Hyper-V, for example running a 1809 container on 1903. Therefore the node OS version should be the same or ahead of the container OS version used (see: [Windows container version compatibility]).

1. Create a new RuntimeClass that Pods can use to try out Hyper-V isolation on the new version of Windows.

    ```yaml
    apiVersion: node.k8s.io/v1beta1
    kind: RuntimeClass
    metadata:
      name: windows-1809-hyperv
    handler: 'containerd-hyperv-17763'
    scheduling:
      nodeSelector:
        kubernetes.io/os: 'windows'
        kubernetes.io/arch: 'amd64'
        node.kubernetes.io/windows-build: '10.0.18362'
      tolerations:
      - effect: NoSchedule
        key: windows
        operator: Equal
        value: "true"
    ```

1. Once sufficient testing is done, the RuntimeClass from Story 1 could be updated. This would cause new deployments to go to these nodes, without having to update them individually. The nodes running the previous Windows version could be drained and removed from the cluster. As the pods running on them are rescheduled, the new RuntimeClass will be applied.

    ```yaml
    apiVersion: node.k8s.io/v1beta1
    kind: RuntimeClass
    metadata:
      name: windows-1809
    handler: 'containerd-hyperv-17763'
    scheduling:
      nodeSelector:
        kubernetes.io/os: 'windows'
        kubernetes.io/arch: 'amd64'
        node.kubernetes.io/windows-build: '10.0.18362'
      tolerations:
      - effect: NoSchedule
        key: windows
        operator: Equal
        value: "true"
    ```

Another RuntimeClass could still be run on the same hosts to use updated containers without Hyper-V isolation.

```yaml
apiVersion: node.k8s.io/v1beta1
kind: RuntimeClass
metadata:
  name: windows-1903
handler: 'default'
scheduling:
  nodeSelector:
    kubernetes.io/os: 'windows'
    kubernetes.io/arch: 'amd64'
    node.kubernetes.io/windows-build: '10.0.18362'
  tolerations:
  - effect: NoSchedule
    key: windows
    operator: Equal
    value: "true"
```

#### Story 3 - Choosing a specific multi-arch image

Starting from story 2, the RuntimeClass `handler` field could also be used as a means os/version/architecture/variant used by the container runtime. The `handler` is matched up with a corresponding section in the ContainerD configuration file.

Here's an example of what corresponding ContainerD configurations could look like for the runtimes above. The first one would force Hyper-V for compatibility with containers needing 10.0.17763. The second one would be the default for the current version where no Hyper-V isolation is required for compatibility.

```toml
        [plugins.cri.containerd.runtimes.containerd-hyperv-17763]
          runtime_type = "io.containerd.runhcs.v1"
          [plugins.cri.containerd.runtimes.containerd-hyperv-17763.options]
            SandboxImage = "{{WINDOWSSANDBOXIMAGE}}"
            SandboxPlatform = "windows/amd64"
            SandboxOsVersion = "10.0.17763"
            SandboxIsolation = 1

        [plugins.cri.containerd.runtimes.default]
          runtime_type = "io.containerd.runhcs.v1"
          # No version is specified for process isolation, the node OS version is used
```

### Implementation Details/Notes/Constraints

There were multiple options discussed with SIG-Node & SIG-Windows on October 8 2019 prior to filing this KEP. That discussion & feedback were captured in [Difficulties in mixed OS and arch clusters]. If you're looking for more details on other approaches excluded, please review that document.

#### Adding new label node.kubernetes.io/windows-build (done)

> Done. This label was added in Kubernetes 1.17

In [Bounding Self-Labeling Kubelets], a specific range of prefixes were reserved for node self-labeling - `[*.]node.kubernetes.io/*`. Adding a new label within that namespace won't require any changes to NodeRestriction admission. As a new field it also won't require changes to any existing workloads.

Build numbers will be used instead of product names in the node labels for a few reasons:

- The same OS version may be marketed under two different names due to support / licensing differences. For example, Windows Server 2019 and Windows Server version 1809 are the same build number. The actual binary compatibility is based on build number and the same container can run on either.
- Windows product names may not have been announced when the Kubernetes project contributors start building and testing against it. Using build numbers instead avoids needing to change Kubernetes source once a name has been announced.

Here are the current product name to build number mappings to illustrate the point:

|Product Name                          |   Build Number(s)      |
|--------------------------------------|------------------------|
| Windows Server 2019                  | 10.0.17763             |
| Windows Server version 1809          | 10.0.17763             |
| Windows Server version 1903          | 10.0.18362             |
| Windows Server version 1909          | 10.0.18363             |

[Windows Update History] has a full list of version numbers by release & patch. Starting from an example of `10.0.17763.805` - the OS major, minor, and build number - `10.0.17763` - should match for containers to be compatible. The final `.805` refers to the monthly patches and isn't required for compatibility. Therefore, a value such as `node.kubernetes.io/windows-build = 10.0.17763` will be used. Each one of these Windows Server version will have corresponding containers released - see [Container Base Images] and [Windows Insider Container Images] for details.

This will pass the regex validation for labels (references: [src1](https://github.com/kubernetes/kubernetes/blob/release-1.16/staging/src/k8s.io/apimachinery/pkg/util/validation/validation.go#L30-L32) [src2](https://github.com/kubernetes/kubernetes/blob/release-1.16/staging/src/k8s.io/apimachinery/pkg/util/validation/validation.go#L88-L108)). Even the most specific identifier in the Windows registry `BuildLabEx`, for example `18362.1.amd64fre.19h1_release.190318-1202` is within the allowed length and characters to pass the existing validation, although we're not planning to use that whole string. Instead, the kubelet will shorten to just what's needed similar to what's returned today in the output from `kubectl describe node` ([src](https://github.com/kubernetes/kubernetes/blob/0599ca2bcfcae7d702f95284f3c2e2c2978c7772/vendor/github.com/docker/docker/pkg/parsers/operatingsystem/operatingsystem_windows.go#L10))

To make this easier to consume in the Kubelet and APIs, it will be updated in multiple places:

- Add a well-known label in
  - [k8s.io/api/core/v1/well_known_labels.go](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/api/core/v1/well_known_labels.go)
  - [pkg/kubelet/apis/well_known_labels.go](https://github.com/kubernetes/kubernetes/blob/5a790bce3b222b7fd1ef1225e3b20700c577088a/pkg/kubelet/apis/well_known_labels.go)
- Set & update it in Kubelet
  - [pkg/kubelet/kubelet_node_status.go](https://github.com/kubernetes/kubernetes/blob/4fda1207e347af92e649b59d60d48c7021ba0c54/pkg/kubelet/kubelet_node_status.go#L217)

#### Adding annotations to ImageSpec

Current proposal is to change the `ImageSpec` to incorporate annotations for runtime class. These can be passed down to the container runtimes where various actions can be taken as per the runtime class specified. Also proposed is the change to `Image` struct to include the `ImageSpec` and thereby include the annotations.

##### ImageSpec changes

Currently ImageSpec contains just a string.

```
type ImageSpec struct {
  Image string
}
```

The proposal is to add Annotations into the ImageSpec:

```
type ImageSpec struct {
  Image string
  Annotations []Annotation
}
```

The runtimeHandler annotation will be based on the Runtime Class specified by the user:

```
“kubernetes.io/runtimehandler”: “<corresponding values>”
```

We could potentially also add the kubernetes specification annotations for consideration of the runtime:

```
“kubernetes.io/arch”: “amd64”
“kubernetes.io/os”: ”linux”
```

Note that these are currently derived from from GOARCH and GOOS at runtime, which does not reflect the image which needs to be pulled but instead corresponds to the system on which kubelet is running. The runtime handler specified in the annotation will provide more accurate indication of the user intent to the runtime.

##### ImageSpec as part of the Image struct

`ListImage` returns `Image` struct from the runtime to Kubelet which currently does not include the `ImageSpec`:

```
type ImageService interface {
  // PullImage pulls an image from the network to local storage using the supplied
  // secrets if necessary. It returns a reference (digest or ID) to the pulled image.
   PullImage(image ImageSpec, pullSecrets []v1.Secret, podSandboxConfig *runtimeapi.PodSandboxConfig) (string, error)
  // GetImageRef gets the reference (digest or ID) of the image which has already been in
  // the local storage. It returns ("", nil) if the image isn't in the local storage.
  GetImageRef(image ImageSpec) (string, error)
  // Gets all images currently on the machine.
  ListImages() ([]Image, error)
  // Removes the specified image.
  RemoveImage(image ImageSpec) error
  // Returns Image statistics.
  ImageStats() (*ImageStats, error)
}
```

The proposal is to change the `Image` from the following :

```
type Image struct {
  // ID of the image.
  ID string
  // Other names by which this image is known.
  RepoTags []string
  // Digests by which this image is known.
  RepoDigests []string
  // The size of the image in bytes.
  Size int64
}
```

to include the ImageSpec as follows:

```
type Image struct {
  // ID of the image.
  ID string
  // Other names by which this image is known.
  RepoTags []string
  // Digests by which this image is known.
  RepoDigests []string
  // The size of the image in bytes.
  Size int64
  // ImageSpec for the image which includes the run time annotations
  Spec ImageSpec
}
```

Note that the ID field will be potentially duplicated in ImageSpec for backward compatibility.

##### Scenarios

Following is a scenario which is handled precisely by this approach:

1. Same image names with different handlers - `handler1` and `handler2` are getting pulled around the same time.
2. Image with `handler1` is pulled by `EnsureImageExists`
3. Image with `handler2` comes via `EnsureImageExists`. With the current code, since `ImageSpec` only has the name, `GetImageRef` would get a reference for the image downloaded for `handler1`.
4. With the runtime handler annotations added to `ImageSpec` and kept track by runtime, the right image reference will be sent back.

### Risks and Mitigations

#### Adding new node label

The names of aren't part of a versioned API today, so there's no risk to upgrade/downgrade from an API and functionality standpoint. However, if someone wants to keep the node selection experience consistent between Kubernetes 1.14 - 1.17, they may want to manually add the `node.kubernetes.io/windows-build` label to clusters running versions < 1.17. A cluster admin can choose to modify labels using `kubectl label node` after a node has joined the cluster.

#### Annotations in ImageSpec

These annotations will be optional parameters. The runtimes can optionally choose to implement specific behaviour based on these Annotations.

## Design Details

### Test Plan

#### E2E Testing with CRI-ContainerD and Kubernetes

We already have E2E tests targeting the CRI-ContainerD & Kubernetes integration running on Windows - see the KEP for [Windows CRI-ContainerD] for more details. We will add a few `RuntimeClass` definitions to those E2E tests to confirm that Pods can run with `runtime_handler` specified, and without it specified.

As ContainerD integration itself is an alpha feature, we'll be experimenting with these configurations and deciding on a set of recommended defaults before graduating to beta. These defaults will be tested at beta with new test cases.

The existing tests relying on `dockershim` will be run side by side until it is deprecated. This timeline hasn't been specified yet, but it will be announced before CRI-ContainerD for Windows is declared stable in 1.19 or later and follow a normal deprecation cycle.

#### Unit testing with CRITest

Unit tests will be added in CRITest which is in the [Cri-Tools] repo. Tests are already running on Windows - see [testgrid](https://k8s-testgrid.appspot.com/sig-node-containerd#cri-validation-windows).

### Graduation Criteria

#### Alpha

> Proposed for 1.19

For alpha graduation containerd annotations will be used instruct ContainerD which runtime class to target for a given pod (as mentioned in the [Proposal](#proposal)). This will reduce the number of changes needed in CRI/Kubernetes while runtime class support for Windows is being developed in containerd.

This timeline should follow that of [Windows CRI-ContainerD].

In addition to what's included there, for alpha:

- E2E tests for basic coverage of:
  - new label set by kubelet
  - scheduling pods with different runtime classes on the same Windows node

#### Alpha -> Beta Graduation

This timeline should follow that of [Windows CRI-ContainerD].

In addition to what's included there, for beta:

- Unit tests will be added to critest
- E2E tests for basic coverage of:
  - `runtime_handler` updates

#### Beta -> GA Graduation

- Define & add E2E testing for recommended default RuntimeClass configurations for Windows supported with CRI-ContainerD

##### Removing a deprecated flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality which deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include [conformance tests].**

[conformance tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md

### Upgrade / Downgrade Strategy

The new label `node.kubernetes.io/windows-build` can be set or removed if needed without impacting other components as described in [Risks and Mitigations](#risks-and-mitigations)

Users can only opt-in to use the new `runtime_handler` field after setting up and configuring ContainerD. On existing clusters without ContainerD set up, they must use `docker` as the `runtimeHandler` in the `RuntimeClass` today. Therefore they must update to a supported version of ContainerD as a prerequisite which is covered in the scope of another KEP - [Windows CRI-ContainerD].

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- 2019-10-08 - KEP started
- 2019-10-15 - KEP marked implementable
- 2020-01-14 - KEP updated to specify runtime handler as annotations in ImageSpec for alpha
- 2020-05-15 - Milestones updates

## Alternatives

These are some other options that were discussed in meetings with SIG-Node and SIG-Windows, but we're not proceeding with. If for some reason `RuntimeClass` support does not graduate to stable or the community prefers to investigate another option, these could serve as starting points for future KEPs.

### Support multiarch os/arch/version in CRI

The Open Container Initiative specifications for container runtime support specifying the architecture, os, and version when pulling and starting a container. This is important for Windows because there is no kernel compatibility between major versions. In order to successfully start a container with process isolation, the right `os.version` must be pulled and run. Hyper-V can provide backwards compatibility, but the image pull and sandbox creation need to specify `os.version` because the kernel is brought up when the sandbox is created. The same challenges exist for Linux as well because multiple CPU architectures can be supported - for example armv7 with qemu and binfmt_misc.

One way to make the experience uniform for dealing with multi-arch images is to add new optional fields to force a deployment to use a specific os/version/arch. This may be combined with RuntimeClass to simplify node placement if needed.

```
apiVersion: v1
kind: Pod
metadata:
  name: mypod
spec:
  os: windows
  osversion: 10.0.17763
  architecture: amd64
  runtimeClassName: windows-hyperv
  containers:
      - name: iis
        image: mcr.microsoft.com/windows/servercore/iis:windowsservercore-ltsc2019
```

Here's the steps needed to pull and start a pod+container matching a specific os/arch/version:

- ImageService.PullImage(PullImageRequest) - PullImageRequest includes a `PodSandboxConfig` reference
- RuntimeService.RunPodSandbox(RunPodSandboxRequest) - RunPodSandboxRequest includes a `PodSandboxConfig` reference
- RuntimeService.CreateContainer(CreateContainerRequest - the same `PodSandboxConfig` is passed as in the previous step

All of these use the same `PodSandboxConfig`, so they could be added there.

From https://github.com/kubernetes/cri-api/blob/24ae4d4e8b036b885ee1f4930ec2b173eabb28e7/pkg/apis/runtime/v1alpha2/api.proto#L310

```
message PodSandboxConfig {
    // Metadata of the sandbox. This information will uniquely identify the
    // sandbox, and the runtime should leverage this to ensure correct
    // operation. The runtime may also use this information to improve UX, such
    // as by constructing a readable name.
    PodSandboxMetadata metadata = 1;
    // Hostname of the sandbox. Hostname could only be empty when the pod
    // network namespace is NODE.
    string hostname = 2;
    // Path to the directory on the host in which container log files are
    // stored.
    // By default the log of a container going into the LogDirectory will be
    // hooked up to STDOUT and STDERR. However, the LogDirectory may contain
    // binary log files with structured logging data from the individual
    // containers. For example, the files might be newline separated JSON
    // structured logs, systemd-journald journal files, gRPC trace files, etc.
    // E.g.,
    //     PodSandboxConfig.LogDirectory = `/var/log/pods/<podUID>/`
    //     ContainerConfig.LogPath = `containerName/Instance#.log`
    //
    // WARNING: Log management and how kubelet should interface with the
    // container logs are under active discussion in
    // https://issues.k8s.io/24677. There *may* be future change of direction
    // for logging as the discussion carries on.
    string log_directory = 3;
    // DNS config for the sandbox.
    DNSConfig dns_config = 4;
    // Port mappings for the sandbox.
    repeated PortMapping port_mappings = 5;
    // Key-value pairs that may be used to scope and select individual resources.
    map<string, string> labels = 6;
    // Unstructured key-value map that may be set by the kubelet to store and
    // retrieve arbitrary metadata. This will include any annotations set on a
    // pod through the Kubernetes API.
    //
    // Annotations MUST NOT be altered by the runtime; the annotations stored
    // here MUST be returned in the PodSandboxStatus associated with the pod
    // this PodSandboxConfig creates.
    //
    // In general, in order to preserve a well-defined interface between the
    // kubelet and the container runtime, annotations SHOULD NOT influence
    // runtime behaviour.
    //
    // Annotations can also be useful for runtime authors to experiment with
    // new features that are opaque to the Kubernetes APIs (both user-facing
    // and the CRI). Whenever possible, however, runtime authors SHOULD
    // consider proposing new typed fields for any new features instead.
    map<string, string> annotations = 7;
    // Optional configurations specific to Linux hosts.
    LinuxPodSandboxConfig linux = 8;
}
```

Today, PodSandboxConfig has an annotation that can be used as a workaround, but it's stated that "Whenever possible, however, runtime authors SHOULD consider proposing new typed fields for any new features instead.".

The proposed additions are:

```
string os = 9;
string architecture = 10;
string version = 11;
```

These correspond to `platform.os`, `platform.architecture`, and `platform.os.version` as describe in the [OCI image spec](https://github.com/opencontainers/image-spec/blob/master/image-index.md)

### Make the scheduler aware of Multi-arch images

When scheduling a pod, it would need to retrieve the container manifest to infer what os, version and architecture was intended. This could be used to find a matching node and avoid the pull failure. However, this was previously rejected in discussions with SIG-Architecture and SIG-Scheduling because it would impact scheduler performance. Additionally, the scheduler makes no sync network calls to other services (such as a container registry) to make scheduling decisions. It can run in isolation and only connects to other trusted pods such as the APIServer.

### Create a multi-arch Mutating admission controller

Before a deployment is scheduled, it can be handed off to a mutating admission controller registered with a webhook. This could do extra work such as a manifest pull & analysis, then add additional info to the deployment such as a NodeSelector or RuntimeClass. The behavior would probably need to be configurable and tailored to a customer's deployment. To round off the experience, it could also reject requests that can't be fulfilled (no matching OS / architecture) immediately.

Based on a deployment, infer and determine the extra info needed that needs to be passed to ContainerD. This could work within the existing APIs (RuntimeClass / NodeSelector) or work with extended APIs (annotations or more specific pod API).

This approach would still impact scheduling latency if a pod has omitted any of the NodeSelector required by the heuristic. The synchronous calls to a container registry are still needed.

## Future Considerations

While this feature is in alpha, we can continue experimenting with the user experience. There are other enhancements that may work well with these changes to provide an easier experience.

### Pod Overhead

The [Pod overhead KEP] proposes adding an `Overhead` field using the same `ContainerResources` structure today. This would make the scheduler aware of how much resources are required by the container runtime, such as ContainerD with Hyper-V isolation, in addition to those requested for the Pod itself. If this KEP is implemented, it can be tested with multiple hypervisors and used to help build consistency across them.

### RuntimeClass Parameters

RuntimeClass aims to replace experimental pod annotations with a [Runtime Handler] instead. Today that doesn't include any CRI-specific configuration that's passed through CRI. However, if there's a clear use case and need to consolidate parameters across multiple sandbox implementations, it could be added. One easy example might be setting the cpu or memory topology presented from the hypervisor to the sandbox. If a consensus emerges between multiple sandboxed CRI providers such as Kata, Firecracker & Hyper-V, then RuntimeClass could be updated to include more standard parameters and replace some configurations kept in separate configuration files today.

## Reference & Examples

### Multi-arch container image overview

The Open Container Initiative specifications for container runtime support specifying the architecture, os, and version when pulling and starting a container.

Here's one example of a container image that has a multi-arch manifest with entries for Linux & Windows, multiple CPU architectures, and multiple Windows os versions.

```powershell
(docker manifest inspect golang:latest | ConvertFrom-Json)[0].manifests | Format-Table digest, platform

digest                                                                  platform
------                                                                  --------
sha256:a50a9364e9170ab5f5b03389ed33b9271b4a7b6bbb0ab41c4035adb3078927bc @{architecture=amd64; os=linux}
sha256:30526a829a37fe2ba8231c06142879f7f6873bc6ebe78bc99674f8ea0e111815 @{architecture=arm; os=linux; variant=v7}
sha256:a05d345bf4635df552ce9635708676c607d2b833278396470bf5788eea0a4b1c @{architecture=arm64; os=linux; variant=v8}
sha256:b11bad2ef5ef90ab7e5589d9a5af51bc3f65335278e73f95b18db2057c0505ae @{architecture=386; os=linux}
sha256:a7db5fe778800809dc1cacd6ae4a1c33ce3f4eb8f39d722b358d7fb27b3a1f1c @{architecture=ppc64le; os=linux}
sha256:a0a8410be5cb7970e00d98dff42e26afad237c08a746cdf375a1b1ad3e4df08c @{architecture=s390x; os=linux}
sha256:a6bf1ef2d20ecbf73d5d1729182a37377bd8820a0871a0422f27bcad6b928d76 @{architecture=amd64; os=windows; os.version=10.0.14393.3274}
sha256:5141a4422a77e493d48012f65f35c413f4d4ca7da5f450d96227b0c15b3de3e8 @{architecture=amd64; os=windows; os.version=10.0.17134.1069}
sha256:d22e5bf156af4df25a24cb268e955df3503cd91b50cd43b9bcf4bccf7a3c0804 @{architecture=amd64; os=windows; os.version=10.0.17763.805}
```

[Windows CRI-ContainerD]: /keps/sig-windows/20190424-windows-cri-containerd.md
[Guide for scheduling Windows containers in Kubernetes]: https://kubernetes.io/docs/setup/production-environment/windows/user-guide-windows-containers/#taints-and-tolerations
[RuntimeClass Scheduling]: https://kubernetes.io/docs/concepts/containers/runtime-class/#scheduling
[Gatekeeper]: https://github.com/open-policy-agent/gatekeeper
[Difficulties in mixed OS and arch clusters]: https://docs.google.com/document/d/12uZt-KSG8v4CSyUDr0EC6btmzpVOZAWzqYDif3EoeBU/edit#
[PodSandboxConfig]: https://github.com/kubernetes/cri-api/blob/24ae4d4e8b036b885ee1f4930ec2b173eabb28e7/pkg/apis/runtime/v1alpha2/api.proto#L310
[Bounding Self-Labeling Kubelets]: https://github.com/kubernetes/enhancements/blob/f1a799d5f4658ed29797c1fb9ceb7a4d0f538e93/keps/sig-auth/0000-20170814-bounding-self-labeling-kubelets.md
[Windows Update History]: https://support.microsoft.com/en-us/help/4498140
[Cri-Tools]: https://github.com/kubernetes-sigs/cri-tools
[Windows container version compatibility]: https://docs.microsoft.com/en-us/virtualization/windowscontainers/deploy-containers/version-compatibility
[Pod overhead KEP]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/20190226-pod-overhead.md#container-runtime-interface-cri
[Runtime Handler]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/runtime-class.md#runtime-handler
[Container Base Images]: https://docs.microsoft.com/en-us/virtualization/windowscontainers/manage-containers/container-base-images
[Windows Insider Container Images]: https://docs.microsoft.com/en-us/virtualization/windowscontainers/quick-start/using-insider-container-images#install-base-container-image
