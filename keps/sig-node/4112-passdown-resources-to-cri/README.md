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
# [KEP-4112](https://github.com/kubernetes/enhancements/issues/4112): Pass down resources to CRI

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
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [CRI API](#cri-api)
    - [PodSandboxConfig](#podsandboxconfig)
    - [CreateContainer](#createcontainer)
    - [UpdateContainerResourcesRequest](#updatecontainerresourcesrequest)
    - [UpdatePodSandboxResources](#updatepodsandboxresources)
  - [kubelet](#kubelet)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
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
  - [Container annotations](#container-annotations)
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

The CRI runtime lacks visibility to the application resource requirements.

First, the resources required by the containers of a pod are not visible at the
pod sandbox creation time. This can be problematic for example in the case of
VM-based runtimes where all resources need to be reserved/prepared when the VM
(i.e. sandbox) is being created.

Second, the kubelet does not provide complete information about the container
resources specification of native and extended resources (requests and limits)
to CRI. However, various use cases have been identified where detailed
knowledge of all the resources can be utilized in container runtimes for more
optimal resource allocation to improve application performance and reduce
cross-application interference.

This KEP proposes CRI API extensions for providing complete view of pods
resources at sandbox creation, and, providing unobfuscated information about
the resource requests and limits to container runtimes.

## Motivation

When the pod sandbox is created, the kubelet does not provide the CRI runtime
any information about the resources (such as native resources, host devices,
mounts, CDI devices etc) that will be required by the application. The CRI
runtime only becomes aware of the resources piece by piece when containers of
the pod are created (one-by-one).

This can cause issues with VM-based runtimes
(e.g. [Kata containers](https://katacontainers.io/) and [Confidential Containers](https://www.cncf.io/projects/confidential-containers/)) that need to prepare the VM before containers are created.

For Kata to handle PCIe devices properly the CRI needs to tell the kata-runtime
how many PCIe root-ports or PCIe switch-ports the hypervisor needs to create at
sandbox creation depending on the number of devices allocated by the containers.
The PCIe root-port is a static configuration and the hypervisor cannot adjust it
once the sandbox is created. During container creation the PCIe devices are
hot-plugged to the PCIe root-port or switch-port. If the number of pre-allocated
pluggable ports is too low, the attachment will fail (container devices >
pre-allocated hot-pluggable ports).

In the case of Confidential Containers (uses Kata under the hood with additional
software components for attestation) the CRI needs to consider the cold-plug aka
direct attachment use-case. At sandbox creation time the hypervisor needs to
know the exact number of pass-through devices and its properties
(VFIO IOMMU group, the actual VFIO device - there can be several devices in a
IOMMU group, attach to PCIe root-port or PCIe switch-port (PCI-Bridge)).
In a confidential setting a user does not want to reconfigure the VM
(creates an attack-vector) on every create container request. The hypervisor
needs a fully static view of resources needed for VM sizing.

Independent of hot or cold-plug the hypervisor needs to know how the PCI(e)
topology needs to look like at sandbox creation time.

Updating resources of a container means also resizing the VM, hence the
hypervisors needs the complete list of resources available at a update container
request.

Another visibility issue is related to the native and extended resources.
Kubelet manages the native resources (CPU and memory) and communicates resource
parameters over the CRI API to the runtime. The following snippet shows the
currently supported CRI annotations that are provided by the Kubelet to e.g.
`containerd`:

```sh
pkg/cri/annotations/annotations.go

  // SandboxCPU annotations are based on the initial CPU configuration for the sandbox. This is calculated as the
  // sum of container CPU resources, optionally provided by Kubelet (introduced  in 1.23) as part of the PodSandboxConfig
  SandboxCPUPeriod = "io.kubernetes.cri.sandbox-cpu-period"
  SandboxCPUQuota  = "io.kubernetes.cri.sandbox-cpu-quota"
  SandboxCPUShares = "io.kubernetes.cri.sandbox-cpu-shares"

  // SandboxMemory is the initial amount of memory associated with this sandbox. This is calculated as the sum
  // of container memory, optionally provided by Kubelet (introduced in 1.23) as part of the PodSandboxConfig.
  SandboxMem = "io.kubernetes.cri.sandbox-memory"
```

However, the original details of
the resource spec are lost as they get translated (within kubelet) to
platform-specific (i.e. Linux or Windows) resource controller parameters like
cpu shares, memory limits etc. Non-native resources such as extended resources
and the device plugin resources completely invisible to the CRI runtime. However,
[OCI hooks](https://github.com/opencontainers/runtime-spec/blob/master/config.md),
[runC](https://github.com/opencontainers/runc) wrappers,
[NRI](https://github.com/containerd/nri) plugins or in some cases even
applications themselves would benefit on seeing the original resource requests
and limits e.g. for doing customized resource optimization.

Extending the CRI API to communicate all resources already at sandbox creation
and pass down resource requests and limits (of native and extended resources)
would provide a comprehensive and early-enough view of the resource usage of
all containers of the pod, allowing improved resource allocation without
breaking any existing use cases.

### Goals

- make the information about all required resources (e.g. native and extended
  resources, devices, mounts, CDI devices) of a Pod available to the CRI at
  sandbox creation time
- make container resource spec transparently visible to CRI (the container
  runtime)

### Non-Goals

- change kubelet resource management
- change existing behavior of CRI

## Proposal

### User Stories

#### Story 1

As a VM-based container runtime developer, I want to allocate/expose enough
RAM, hugepages, hot- or cold-pluggable PCI(e) ports, protected memory sections
and other resources for the VM to ensure that all containers in the pod are
guaranteed to get the resources they require.

#### Story 2

As a developer of non-runc / non-Linux CRI runtime, I want to know detailed
container resource requests to be able to make correct resource allocation for
the applications. I cannot rely on cgroup parameters on this but need to know
what the user requested to fairly allocate resources between applications.

#### Story 3

As a cluster administrator, I want to install an NRI plugin that does
customized resource handling. I run kubelet with CPU manager and memory manager
disabled (CPU manager policy set to `none`). Instead I use my NRI plugin to do
customized resource allocation (e.g. cpu and memory pinning). To do that
properly I need the actual resource requests and limits requested by the user.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

The proposal only adds new informational data to the CRI API between kubelet
and the container runtime with no user-visible changes which mitigates possible
risks considerably.

Data duplication/inconsistency with native resources could be considered a risk
as those are passed down to CRI both as "raw" requests and limits and as
"translated" resource control parameters (like cpu shares oom scoring etc). But
this should be largely mitigated by code reviews and unit tests.

## Design Details

The proposal is that kubelet discloses full resources information from the
PodSpec to the container runtime. This is accomplished by extending the
ContainerConfig, UpdateContainerResourcesRequest and PodSandboxConfig messages
of the CRI API.

With this information, the runtime can for example do detailed resource
allocation so that CPU, memory and other resources for each container are
optimally aligned. This applies to scenarios where the kubelet CPU manager is
disabled (by using the `none` CPU manager policy).

The resource information is included in PodSandboxConfig so that the runtime
can see the full picture of Pod's resource usage at Pod creation time, for
example enabling more holistic resource allocation and thus better
interoperability between containers inside the Pod.

Also the CreateContainer request is extended to include the unmodified resource
requirements. This make it possible for the CRI runtime to detect any changes
in the pod resources that happen between the Pod creation and container
creation in e.g.  scenarios where in-place pod updates are involved.

[KEP-1287][kep-1287] Beta ([PR][kep-1287-beta-pr]) proposes to add new
UpdatePodSandboxResources rpc to the CRI API. If/when KEP-1287 is implemented
as proposed, the UpdatePodSandboxResources CRI message is updated to include
the resource information of all containers (aligning with
UpdateContainerResourcesRequest).

[KEP-2837][kep-2837] Alpha ([PR][kep-2837-alpha-pr]) proposes to add new
Pod-level resource requirements field to the PodSpec. This information will be
be added to the PodResourceConfig message, similar to the container resource
information, if/when KEP-2837 is implemented as proposed.

### CRI API

#### PodSandboxConfig

The PodSandboxConfig message (part of the RunPodSandbox request) will be
extended to contain information about resources of all its containers known at
the pod creation time. The container runtime may use this information to make
preparations for all upcoming containers of the pod. E.g. setup all needed
resources for a VM-based pod or prepare for optimal allocation of resources of
all the containers of the Pod. However, the container runtime may continue to
operate as they did (before this enhancement). That is, it can ignore
the resource information presented here and allocate resources for each
container separately at container creation time with the `CreateContainer`
request.

```diff
 message PodSandboxConfig {
 
 ...
 
     // Optional configurations specific to Linux hosts.
     LinuxPodSandboxConfig linux = 8;
     // Optional configurations specific to Windows hosts.
     WindowsPodSandboxConfig windows = 9;
+
+    // Kubernetes resource spec of the containers in the pod.
+    PodResourceConfig pod_resources = 10;
 }
 
+// PodResourceConfig contains information of all resources requirements of
+// the containers of a pod.
+message PodResourceConfig {
+    repeated ContainerResourceConfig containers = 1;
+}
 
+// ContainerResourceConfig contains information of all resource requirements of
+// one container.
+message ContainerResourceConfig {
+    // Name of the container
+    string name= 1;
+
+    // Type of the container
+    ContainerType type= 2;
+
+    // Kubernetes resource spec of the container
+    KubernetesResources kubernetes_resources = 3;
+
+    // Mounts for the container.
+    repeated Mount mounts = 4;
+
+    // Devices for the container.
+    repeated Device devices = 5;
+
+    // CDI devices for the container.
+    repeated CDIDevice CDI_devices = 6;
+}

+enum ContainerType {
+    INIT_CONTAINER    = 0;
+    SIDECAR_CONTAINER = 1;
+    CONTAINER = 2;
+}
```

The Pod-level resources enhancement [KEP-2837][kep-2837]
([alpha PR][kep-2837-alpha-pr]) proposes to add new Pod-level resource
requirements fields to the PodSpec. This information will be be added to the
PodResourceConfig message, similar to the container resource information.

```diff
 message PodResourceConfig {
     repeated ContainerResourceConfig containers = 1;
+
+    // Kubernetes resource spec of the pod-level resource requirements.
+    KubernetesResources kubernetes_resources = 2;
 }
```

The implementation if adding the KubernetesResources field to the
PodResourceConfig is synced with [KEP-2837][kep-2837].

#### CreateContainer

The ContainerConfig message (used in CreateContainer request) is extended to
contain unmodified resource requests from the PodSpec.

```diff
+import "k8s.io/apimachinery/pkg/api/resource/generated.proto";

 message ContainerConfig {
 
 ...
 
     // Configuration specific to Windows containers.
     WindowsContainerConfig windows = 16;
 
     // CDI devices for the container.
     repeated CDIDevice CDI_devices = 17;
+
+    // Kubernetes resource spec of the container
+    KubernetesResources kubernetes_resources = 18;
 }
 
+// KubernetesResources contains the resource requests and limits as specified
+// in the Kubernetes core API ResourceRequirements.
+message KubernetesResources {
+    // Requests and limits from the Kubernetes container config.
+    map<string, k8s.io.apimachinery.pkg.api.resource.Quantity> requests = 1;
+    map<string, k8s.io.apimachinery.pkg.api.resource.Quantity> limits = 2;
+}
```

Note that mounts, devices, CDI devices are part of the ContainerConfig message
but are left out of the diff snippet above.

Including the KubernetesResources in the ContainerConfig message serves
multiple purposes:

1. Catch changes that happen between pod sandbox creation and container
   creation. For example, in-place pod updates might change the container
   before it was created.
2. Catch changes that happen over container restarts in in-place pod update
   scenarios
3. Consistency/completeness. Have enough information to make consistent action
   based only on information present in this rpc caal.

The resources (mounts, devices, CDI devices, Kubernetes resources) in the
CreateContainer request should be identical to what was (pre-)informed in the
RunPodSandbox request. If they are different, the CRI runtime may fail the
container creation, for example because changes cannot be applied after a
VM-based Pod has been created.

#### UpdateContainerResourcesRequest

The UpdateContainerResourcesRequest message is extended to pass down unmodified
resource requests from the PodSpec.

```diff
 message UpdateContainerResourcesRequest {
     // ID of the container to update.
     string container_id = 1;
     // Resource configuration specific to Linux containers.
     LinuxContainerResources linux = 2;
     // Resource configuration specific to Windows containers.
     WindowsContainerResources windows = 3;
     // Unstructured key-value map holding arbitrary additional information for
     // container resources updating. This can be used for specifying experimental
     // resources to update or other options to use when updating the container.
     map<string, string> annotations = 4;
+
+    // Kubernetes resource spec of the container
+    KubernetesResources kubernetes_resources = 5;
 }
```

Note that mounts, devices, CDI devices are not part of the
UpdateContainerResourcesRequest message and this proposal does not suggest
adding them.

#### UpdatePodSandboxResources

The In-Place Update of Pod Resources ([KEP-1287][kep-1287]) Beta
([PR][kep-1287-beta-pr]) proposes to add new UpdatePodSandboxResources rpc to
inform the CRI runtime about the changes in the pod resources.

The UpdatePodSandboxResourcesRequest message is extended similarly to the
[PodSandboxConfig](#podsandboxconfig) message to contain information about
resources of all its containers. In UpdatePodSandboxResourcesRequest this will
reflect the updated resource requirements of the containers.

```diff
 message UpdatePodSandboxResourcesRequest {
     // ID of the PodSandbox to update.
     string pod_sandbox_id = 1;
 
     // Optional overhead represents the overheads associated with this sandbox
     LinuxContainerResources overhead = 2;
     // Optional resources represents the sum of container resources for this sandbox
     LinuxContainerResources resources = 3;
 
     // Unstructured key-value map holding arbitrary additional information for
     // sandbox resources updating. This can be used for specifying experimental
     // resources to update or other options to use when updating the sandbox.
     map<string, string> annotations = 4;
+
+    // Kubernetes resource spec of the containers in the pod.
+    PodResourceConfig pod_resources = 5;
 }
```

The implementation will be synced with [KEP-1287][kep-1287].

### kubelet

Kubelet code is refactored/modified so that all container resources are known
before sandbox creation. This mainly consists of preparing all mounts (of all
containers) early.

Kubelet will be extended to pass down all resources of containers in all
related CRI requests (as described in the [CRI API](#cri-api) section). That
is:

- adding mounts, devices, CDI devices and the unmodified resource requests and
  limits of all containers into RunPodSandbox request
- adding unmodified resource requests and limits into CreateContainer and
  UpdateContainerResources requests

For example, take a PodSpec:

```yaml
apiVersion: v1
kind: Pod
...
spec:
  containers:
  - name: cnt-1
    image: k8s.gcr.io/pause
    resources:
      requests:
        cpu: 1
        memory: 1G
        example.com/resource: 1
      limits:
        cpu: 2
        memory: 2G
        example.com/resource: 1
    volumeMounts:
    - mountPath: /my-volume
      name: my-volume
    - mountPath: /image-volume
      name: image-volume
  volumes:
  - name: my-volume
    emptyDir:
  - name: image-volume
    image:
      reference: example.com/registry/artifact:tag
```

Then kubelet will send the following RunPodSandboxRequest when creating the Pod
(represented here in yaml format):

```yaml
RunPodSandboxRequest:
  config:
  ...
    podResources:
      containers:
      - name: cnt-1
        kubernetes_resources:
          requests:
            cpu: "1"
            memory: 1G
            example.com/resource: "1"
          limits:
            cpu: "2"
            memory: 2G
            example.com/resource: "1"
        CDI_devices:
        - name: example.com/resource=CDI-Dev-1
        mounts:
        - container_path: /my-volume
          host_path: /var/lib/kubelet/pods/<pod-uid>/volumes/kubernetes.io~empty-dir/my-volume
        - container_path: /image-volume
          image:
            image: example.com/registry/artifact:tag
          ...
        - container_path: /var/run/secrets/kubernetes.io/serviceaccount
          host_path: /var/lib/kubelet/pods/<pod-uid>/volumes/kubernetes.io~projected/kube-api-access-4srqm
          readonly: true
        - container_path: /dev/termination-log
          host_path: /var/lib/kubelet/pods/<pod-uid>/containers/cnt-1/<uuid>
```

Note that all device plugin resources are passed down in the
`kubernetes_resources` field but this does not contain any properties of the
device that was actually allocated for the container. However, these properties
are exposed through the `CDI_devices`, `mounts` and `devices` fields.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

No prerequisite testing updates have been identified.

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

- `k8s.io/kubernetes/pkg/kubelet/kuberuntime`: `2024-02-02` - `68.3%`

The
[fake_runtime](https://github.com/kubernetes/cri-api/blob/master/pkg/apis/testing/fake_runtime_service.go)
will be used in unit tests to verify that the Kubelet correctly passes down the
resource information to the CRI runtime.

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

For alpha, no new integration tests are planned.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

For alpha, no new e2e tests are planned.

For Beta: a suite of NRI tests will be added to verify that the runtime
receives the resource information correctly and passes it down to the NRI
plugins.

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

#### Alpha

- Feature implemented behind a feature flag
- Initial unit tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Feature gate enabled by default
- containerd and CRI-O runtimes have released versions that have adopted the
  new CRI API changes

#### GA

- No bugs reported in the previous cycle

### Upgrade / Downgrade Strategy

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

The feature gate (in kubelet) controls the feature enablement. Existing runtime
implementations will continue to work as previously, even if the feature is
enabled.

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

The feature is node-local (kubelet-only) so there is no dependencies or effects
to other Kubernetes components.

The behavior is unchanged if either kubelet or the CRI runtime running on a
node does not support the feature. If kubelet has the feature enabled but the
CRI runtime does not support it, the CRI runtime will ignore the new fields in
the CRI API and function as previously. Similarly, if the CRI runtime supports
the feature but the kubelet does not, the runtime will resort to the previous
behavior.

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

- [X] Feature gate
  - Feature gate name: KubeletContainerResourcesInPodSandbox
  - Components depending on the feature gate:
    - kubelet

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

Yes. The kubelet will start passing the extra information to the CRI runtime
for every container it creates. Whether this has any effect depends on if the
underlying CRI runtime supports this feature. For example, an NRI plugin
relying on the feature may cause the application to behave differently.

Long running pods that persist (without restart) over kubelet and CRI runtime
update which enables the feature may experience version skew of the metadata.
After enabling the feature, the CRI runtime does not have the aggregated
information of all resources of the pod, provided with this feature, as the
kubelet didn't restart these pods (didn't send the CreatePodSandbox CRI
request). This may affect some scenarios e.g. NRI plugins. This "metadata skew"
can be avoided by draining the node before updating the kubelet and the CRI
runtime.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes, disabling the `KubeletContainerResourcesInPodSandbox` feature gate will
disable the feature. Restarting pods may be needed to reset the information
that was passed down to the CRI.

###### What happens if we reenable the feature if it was previously rolled back?

New pods will have the feature enabled. Existing pods will continue to operate
as before until restarted.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

Unit tests for the feature gate will be added.

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

Rollback or rollout in the kubelet should not fail - it only enables/disabled
the information (fields in the CRI message) passed down to the CRI runtime.

However, if the CRI runtime depends on the feature, a rollout or rollback may
cause failures of applications on pod restarts. Running pods are not affected.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

Alpha: No new metrics are planned. Increase in the existing
`kubelet_started_pods_errors_total` metric can indicate a problem caused by
this feature.

Generally, non-ready pods with CreatePodSandboxError status (reflected by the
`kubelet_started_pods_errors_total` metric) is a possible indicator. The error
message will contain details if the CRI failure is related to the feature.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

Alpha: Manual testing of the feature gate is performed.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

By examing the kubelet feature gate and the version of the CRI runtime. The
enablement of the kubelet feature gate can be determined from the
`kubernetes_feature_enabled` metric.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

The end users do not see the status of the feature directly.

The cluster operator can verify that the feature is working by examining the
kubelet and CRI runtime logs.

The CRI runtime or NRI plugin developers depending on the feature can ensure
that it is working by verifying that all the required information is available
at pod sandbox creation time.

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

No increase in the `kubelet_started_pods_errors_total` rate.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `kubelet_started_pods_errors_total`
  - Components exposing the metric: kubelet

> NOTE: The `kubelet_started_pods_errors_total` metric is a general metric for
> any errors that occur when starting pods. The error message (Pod events,
> kubelet logs) will contain details if the CRI failure is related to the
> feature.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

N/A.

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

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

No.

However, the practical usability of this feature requires that also the CRI
runtime supports it. The feature is effectively a no-op if the CRI runtime does
not support it.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

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

No.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

Not noticeably.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No. The new data fields in the CRI API would not count as significant increase.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

No.

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

N/A. The feature is node-local.

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

The feature in Kubernetes is relatively straightforward - passing extra
information to the CRI runtime. The failure scenarios arise in the CRI runtime
level, e.g.:

- misbehaving CRI runtime or NRI plugin
- CRI runtime or NRI plugin is depending on the feature but it is not enabled
  in the kubelet
- configuration skew in the cluster where some nodes have the feature enabled
  and some do not

Pod events and CRI runtime logs are the primary sources of information for
these failure scenarios.

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A.

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

### Container annotations

Container annotations could be used as an alternative way to pass down the
resource requests and limits to the container runtime.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

<!-- References -->

[kep-1287]: https://github.com/kubernetes/enhancements/issues/1287
[kep-1287-beta-pr]: https://github.com/kubernetes/enhancements/pull/4704
[kep-2837]: https://github.com/kubernetes/enhancements/issues/2837
[kep-2837-alpha-pr]: https://github.com/kubernetes/enhancements/pull/4678
