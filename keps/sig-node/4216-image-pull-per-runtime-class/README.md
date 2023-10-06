# KEP

# KEP-4216: Image pull per runtime class

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [CRI/kubelet changes:](#crikubelet-changes)
    - [ImageSpec](#imagespec)
  - [Kubelet changes](#kubelet-changes)
  - [Tooling changes](#tooling-changes)
    - [CRICTL changes](#crictl-changes)
  - [Test Plan](#test-plan)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary


Today, container images in container runtimes are identified by the image name (aka. image reference) and/or image digest (usually a SHA256 cryptographic hash of the content). The content of the image referenced is either an image manifest or an image index manifest. The image index is typically arranged to contain a list of image manifests by platform (linux, windows,...).  When the reference is to an image index, platform matching logic is used to identify and pull the appropriately matching image manifest from the index for the platform that the container is being asked to run on.

<pre>
Example of an index image, Python:

{
   "schemaVersion": 2,
   "mediaType": "application/vnd.docker.distribution.manifest.list.v2+json",
   "manifests": [
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 2007,
         "digest": "sha256:8a164692c20c8f51986d25c16caa6bf03bde14e4b6e6a4c06b5437d5620cc96c",
         "platform": {
            "architecture": "amd64",
            "os": "linux"
         }
      },
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 2007,
         "digest": "sha256:ceac4b5b55ccba7b742e0a2d2765711c44cd228d1a990018a07b94b48c59577e",
         "platform": {
            "architecture": "arm",
            "os": "linux",
            "variant": "v5"
         }
      },
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 2007,
         "digest": "sha256:ea4f4ff16827bdc8e019284f964a397968c3769cc6534502009ff9516bd8c4f4",
         "platform": {
            "architecture": "arm",
            "os": "linux",
            "variant": "v7"
         }
      },
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 2007,
         "digest": "sha256:20d0d27bf4b7998f6deaa523de3f5dd5298d7b53e7e02adccb9b7df183b638c2",
         "platform": {
            "architecture": "arm64",
            "os": "linux",
            "variant": "v8"
         }
      },
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 2007,
         "digest": "sha256:717a9c1bdff7cd9e9ca31de78d7ffbdb3fb6f2b5d43f9cb3e75b21d48fd638c0",
         "platform": {
            "architecture": "386",
            "os": "linux"
         }
      },
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 2007,
         "digest": "sha256:732c0cf2c0a6cf206bb7ea71232bfec5075a8af25f1da05628959c8cdc6205f6",
         "platform": {
            "architecture": "ppc64le",
            "os": "linux"
         }
      },
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 2007,
         "digest": "sha256:f265d2f398ffce7252d6162ead0bc802afad2de309cf662ec16645d1e0e85564",
         "platform": {
            "architecture": "s390x",
            "os": "linux"
         }
      },
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 2831,
         "digest": "sha256:53c5f0dd905eef3899284d845431ccaa1045f97fc205edd87dfc2151c4331980",
         "platform": {
            "architecture": "amd64",
            "os": "windows",
           <mark> "os.version": "10.0.20348.1970"  </mark>
         }
      },
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "size": 2831,
         "digest": "sha256:5981df14a07aaa7fe0c7d80a4c61f33f4ad4d8d29a346fd1b2cacf090b3de8c2",
         "platform": {
            "architecture": "amd64",
            "os": "windows",
           <mark> "os.version": "10.0.17763.4851" </mark>
         }
      }
   ]
}
</pre>

For example, in containerd runtime, each platform has its own default platform matcher which is used to select the image manifest from an image index based on the OS/platform where the node is running (Linux, Windows, Darwin, etc).
The platform matcher on linux attempts to check for host and guest OS match and optionally for a variant match as well.
However, the platform matcher on windows checks for exact OS version match between the host and guest (OSVersion field in the oci ImageSpec is primarily used only for windows manifests) and if a manifest with such an exact match is found, it is pulled. Otherwise an error is thrown. <br> This is the desired behavior for process isolated windows containers running on the same version of the kernel that the host is running on. However, it is not ideal for hyperV isolated windows containers. This is because OS version of the host and the UVM image need not be the same.
For example, Windows Server 2019 container images can run inside a Utility VM (UVM) on a Windows Server 2022 host and because the container will be running in the UVM, the platform matching image pulls for the UVM instances should be allowed. For the full matrix of accepted behavior, check the link here: https://learn.microsoft.com/en-us/virtualization/windowscontainers/deploy-containers/version-compatibility?tabs=windows-server-2022%2Cwindows-10#windows-server-host-os-compatibility

` Note: ` UVM is short for utility VM. It is a light weight VM that boots a special version of Windows that is purpose built to do just one thing- run containers. UVMs provide strong isolation boundaries and do not share kernel with the host OS. They are transparent to the user and do not require any management from the user.

This problem can be fixed if container runtimes, CRI, and kubelet refer to images as a tuple of (imageName/imageDigest, runtimeClass) instead of the imageName/imageDigest. With this, we can specify the runtimeClass a particular image is being run for and the pull implementation on container runtimes can be extended to pull the appropriate manifest from the image index.
An example of how the pull implementation can be extended to support this is as follows:
On containerd, the windows platform matcher is responsible for pulling an appropriate image from an image index on windows platforms. Today, this platform matcher on windows always looks for an image manifest with the OSVersion that matches the host and pulls it. This can be changed to pick then image manifest with the platform OSVersion that matches the one set for a particular runtime class in containerd.toml instead of the host OSVersion.
An example of how the guest platform information can be set in containerd is as follows:

<pre>
        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runhcs-wcow-process]
          base_runtime_spec = ""
          cni_conf_dir = ""
          cni_max_conf_num = 0
          container_annotations = ["io.microsoft.container.*"]
          pod_annotations = []
          privileged_without_host_devices = false
          privileged_without_host_devices_all_devices_allowed = false
          runtime_path = ""
          runtime_type = "io.containerd.runhcs.v1"
          sandbox_mode = ""
          snapshotter = ""

<mark>
          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runhcs-wcow-process.guest_platform]
            Architecture = "amd64"
            OS = "windows"
            OSFeatures = []
            OSVersion = "10.0.20348"
            Variant = ""
</mark>
          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runhcs-wcow-process.options]
</pre>

On k8s, the pod and runtime class yamls would look as follows:

<pre>
apiVersion: node.k8s.io/v1
kind: RuntimeClass
metadata:
  name: runhcs-wcow-process
handler: runhcs-wcow-process 
</pre>

Corresponding pod definition would look as follows:

<pre>
apiVersion: v1 
kind: Pod 
metadata: 
  name: mypod 
spec:
  runtimeClassName: runhcs-wcow-runhcs
  containers: 
  - name: busybox 
    image: docker.io/library/busybox:1.35.0-musl 
</pre>

This document details the changes needed to support pulling images per runtime class.
Adding this support would enable us to have different versions of the same image on a given node and could be really beneficial in virtualization based runtimes. It will also be very helpful to support future projects to support snapshotter per runtimeclass. 
For windows in particular, different versions of a multiarch image could be pulled based on hyperV and process isolation and we could remove the necessity for exact host and guest OS version matches for hyperV isolation.

This KEP attempts to goes over the details of the most significant changes needed in CRI, kubelet and tools like ctr and crictl to support this feature. Container runtimes like containerd and CRI-O are responsible for making necessary changes in their runtime to support this feature. Backwards compatibility will always be maintained to ensure existing behavior in container runtimes do not break.

## Motivation

Currently, the windows platform matcher looks for an exact OS version match between the host and guest while pulling images. This behavior is not accurate for hyperV isolation as the UVM/sandbox OS version could be different from the host OS version.  

For example, a user might want to pull an image like python (a manifest list) on a WS2022 host with an intent of running it in on a WS2019 hyperV isolated UVM. But unfortunately, the current windows platform matcher behavior only forces the WS2022 version of the python image to be pulled (due to the exact version match between host and guest). If a manifest list does not have a version that matches the host OS version, then the image pull fails.
Extending image pulls to specify an optional runtime class would greatly help to extend windows platform matcher behavior to pull different manifests from the image index depending on what runtime class is used.
For example, we could have one runtime class pull one version of python image while another runtime class could be used to pull a completely different version of the same image.
Additionally, having support to pull images per runtime class might also help projects like supporting different snapshotters with different runtimes as well.

### Goals

This KEP describes the detailed design and API changes needed in kubelet and CRI to support image pull with optional runtime class option.
Goal is to keep the functionality same as it is today if user does not use the runtime class option with image pull.

### Non-Goals

This KEP only addresses the changes needed on kubelet and CRI. It does not go over the changes needed on container runtimes like containerd, CRI-O etc to support this feature. The implementation details should be handled by the respective container runtimes.

Also, this KEP does not intend to address snapshotter per runtime class. It is different from image pull per runtime class and would need a separate KEP to address it.

## Proposal

## Design Details

### CRI/kubelet changes:

With this feature, multiple versions of the same image can exist on a single node.
For example, python image can exist on the node for WS2022 (OSVersion = 20348) and WS2019(OSVersion = 17703) depending on the guestOSVersion mentioned in containerd's toml file for a runtime.
Therefore, it is important for kubelet, containerd, and other components to identify the image as its name and runtimeClass rather than just the image name.
The following sections details the major changes that would need to be made to the different components to support this.

#### ImageSpec
Include field on CRI `ImageSpec` message to specify optional runtime class to use to pull image


**Proposed:**

<pre>
// ImageSpec is an internal representation of an image.
message ImageSpec {
    // Container's Image field (e.g. imageID or imageDigest).
    string image = 1;
    // Unstructured key-value map holding arbitrary metadata.
    // ImageSpec Annotations can be used to help the runtime target specific
    // images in multi-arch images.
    map<string, string> annotations = 2;
    // The container image reference specified by the user (e.g. image[:tag] or digest).
    // Only set if available within the RPC context.
    string user_specified_image = 18;
    // Named runtime configuration to use for pulling the image.
    // If the runtime handler is unknown, this request should be rejected.  An
    // empty string should select the default handler, equivalent to the
    // behavior before this feature was added.
    <mark>string runtime_handler = 19;</mark>
}
</pre>

The "RuntimeClassName" of [podspec](https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/core/types.go#L3084) can be used to specify the runtime class name and then can be populated in ImageSpec struct during **EnsureImageExists()** call in kubelet.

ImageSpec is referenced in the following APIs and can set the RuntimeHandler
field if needed:

1. PullImageRequest
2. RemoveImageRequest
3. ImageStatusRequest
4. ListImages

### Kubelet changes

Kubelet needs to change how it talks about images from just the image name to (imageName, runtimeclass). As discussed in earlier sections, extending the `Image` struct in CRI to add a field for runtimeclass will make it easier to extend Kubelet API to include this information.
` Note: ` The kubelet changes will be under a feature-gate called `RuntimeClassInImageCriApi`
The following are scenarios where kubelet behavior needs to be extended to refer to image as its name and runtimeClass:

1. Before running a container, kubelet checks to see whether the required image for the container already exists or not by calling the **EnsureImageExists()** function.

PullImageRequest has a field for ImageSpec (ImageSpec will have a newly added field for runtime class as mentioned in the section above) which can hold info for the runtime class being used to pull this image:

<pre>
message PullImageRequest {
    // Spec of the image.
   <mark> ImageSpec image = 1; </mark>
    // Authentication configuration for pulling the image.
    AuthConfig auth = 2;
    // Config of the PodSandbox, which is used to pull image in PodSandbox context.
    PodSandboxConfig sandbox_config = 3;
}
</pre>

2. Image status request/response can also make use of the newly added runtime class field in ImageSpec to specify the runtime class being used for the image. If none is specified, containerd will assume that the request is for the default runtime class defined for that platform.

<pre>
message ImageStatusRequest {
    // Spec of the image.
    ImageSpec image = 1;
    // Verbose indicates whether to return extra information about the image.
    bool verbose = 2;
}

message ImageStatusResponse {
    // Status of the image.
    Image image = 1;
    // Info is extra information of the Image. The key could be arbitrary string, and
    // value should be in json format. The information could include anything useful
    // for debug, e.g. image config for oci image based container runtime.
    // It should only be returned non-empty when Verbose is true.
    map<string, string> info = 2;
}
</pre>

3. Image cleanup/kubelet garbage collection: <br> Currently, kubelet GC will call into RemoveImage() on container runtime side to remove unused images. kubelet will only pass the image name and container runtime will currently delete all instances of the image. <br> This needs to be extended on kubelet to ask for deleting (imageName, runtimeclass) to be more efficient.

### Tooling changes

#### CRICTL changes

List of crictl image commands that need to be extended:

1. crictl pull image --runtime runtimeClassName imageName

2. crictl images will have new column to specify the runtime class name of the image as we could now have multiple versions of the same image based on what runtime class was used to pull the image. If no runtime class was used it will specify the default runtime class name for the platform

3. crictl rmi should have new option to specify runtime class name so that the appropriate image can be removed. (With this feature in place, multiple versions of the same image can exist on the node based on what runtime class was used to pull the image)
4. crictl image info should show runtime class

### Test Plan

We should have functional tests and regression tests for all platforms (windows, linux, darwin) to ensure functionality and stability.

- Regression testing to ensure existing functionality is not regressed when no runtime class is specified while pulling the image
- Unit tests and e2e tests for CRI, and kubelet.
- Integration tests would be challenging to add for these tests as it need container runtime
changes as well for full functionality. So we do not plan to add integration tests.

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.


##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- Tests to ensure existing behavior when runtimehandler field is not set in ImageSpec
- Tests to check for image pull behavior when non-default runtimehandler is used to pull images

### Graduation Criteria

#### Alpha

- Implement the above mentioned CRI changes
- Implement necessary kubelet changes to pass runtimeClass on CRI calls behind a feature flag
- Add related e2e and unit tests to ensure functionality and make sure there is no regression in
existing behavior when runtime class is not specified in the CRI calls to container runtime.

#### Beta

- Gather feedback from developers and surveys
- Work on feedback and add additional tests as needed
- Follow up on container runtime changes for containerd and CRI-O

#### GA

- Decision on GA will be made based on beta feedback

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

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: RuntimeClassInImageCriApi
  - Components depending on the feature gate: kubelet
- [x] Other
  - Describe the mechanism: `N/A`
  - Will enabling / disabling the feature require downtime of the control
    plane? `No`
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? `No`

###### Does enabling the feature change any default behavior?

No. Aim of this KEP to ensure default behavior is not changed. Any changes to CRI and kubelet will be under feature gate as mentioned in earlier sections.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.
All the code changes needed will be under feature gate `RuntimeClassInCriCalls` that will be off by default for alpha. When the feature-gate is disabled, some cleanup steps are needed. This is because there could be different versions of the same image and kubelet could pick a wrong image without cleanup.
Steps for cleanup: Delete deployment, prune images and restart pods.

###### What happens if we reenable the feature if it was previously rolled back?
When feature gate is reenabled, the same cleanup steps mentioned above need to be run. This is because the kubelet would be unaware of the runtime classes for a particular image.

###### Are there any tests for feature enablement/disablement?
Yes, this test will be added for alpha.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

`Note:` This section will be filled once we move to beta.

###### How can an operator determine if the feature is in use by workloads?

In Beta stage, we will add a metric that helps to understand how many images are using a non-default runtime class. This would help determine how many users are leveraging this feature.

###### How can someone using this feature know that it is working for their instance?

Image pull goes through successfully when non-default runtimehandler is used.
For example, if host is a windows node, user should be able to pull images for hyperV isolation even if the container image OSVersion does not match the host OS version.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Exisiting behavior of kubelet should not change when runtime class is not specified when image is pulled.

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

`N/A`

###### Are there any missing metrics that would be useful to have to improve observability of this feature?
`N/A`

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

`N/A`

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls? `No`

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

###### Will enabling / using this feature result in introducing new API types? `No`

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider? `No`

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects? 
Yes. RuntimeHandler field will be added to ImageSpec CRI as discussed in the sections above

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs? `No`

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components? `No`

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)? `No`

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable? `N/A`

###### What are other known failure modes?
`N/A`
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
`N/A`

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
